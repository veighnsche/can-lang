// E04: server HTTP client budget legs (cancel-present branch).
//
// performRequest caps the connection timeout by the remaining shared
// budget and links request-scope expiry to the timeout path; the
// native abort stops the wire request. Servers bind ephemeral loopback
// ports, so every run uses distinct ports.
import { describe, expect, test } from "bun:test";
import { createHash } from "node:crypto";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, domainFailureDiagnostics, type FailureShape } from "../domain.ts";
import { performRequest } from "../transport/fetch.ts";
import { createTransport, type HTTPTypes } from "../transport/http.ts";
import { Deadline, transportProblem } from "../transport/deadline.ts";
import type { Connection } from "../transport/request.ts";
import { runOwnedRoot } from "../owner.ts";
import { success, value, type Completion } from "../completion.ts";
import { createRequestBudget } from "../transport/request-budget.ts";
import { createRequestScope, runWithRequestScope } from "../transport/request-scope.ts";

const base = { timeoutMilliseconds: 10000, maxBodyBytes: 100, headers: [] };
const request = { path: "/", method: "GET" as const, query: [], headers: [] };
const decode = (bytes: Uint8Array) => success(new TextDecoder().decode(bytes));
async function root<T>(body: () => Promise<T>): Promise<T> {
  return value((await runOwnedRoot(async () => success(await body()))).completion);
}

function requestSignal(request: Request): AbortSignal | undefined {
  return (request as unknown as { signal?: AbortSignal }).signal;
}

async function waitFor(label: string, ready: () => boolean, budgetMs: number): Promise<void> {
  const started = Date.now();
  while (!ready()) {
    if (Date.now() - started > budgetMs) throw new Error(`timed out waiting for ${label}`);
    await Bun.sleep(20);
  }
}

type StallServer = {
  url: string;
  attempts: () => number;
  aborted: () => boolean;
  stop: () => void;
};

function stallServer(stillMs: number): StallServer {
  let attempts = 0;
  let aborted = false;
  const server = Bun.serve({
    hostname: "127.0.0.1",
    port: 0,
    async fetch(request) {
      attempts++;
      requestSignal(request)?.addEventListener(
        "abort",
        () => {
          aborted = true;
        },
        { once: true },
      );
      await Bun.sleep(stillMs);
      return new Response("late");
    },
  });
  return {
    url: server.url.href,
    attempts: () => attempts,
    aborted: () => aborted,
    stop: () => {
      void server.stop(true);
    },
  };
}

// A fake clock pins the remaining budget exactly, so the capped bound
// asserts to the millisecond instead of a timing window.
function exactBudget(totalMs: number, remainingMs: number) {
  let at = 1000;
  const budget = createRequestBudget(totalMs, () => at);
  at += totalMs - remainingMs;
  return budget;
}

describe("performRequest budgets", () => {
  test("the remaining budget caps the connection timeout exactly", async () => {
    const stall = stallServer(3000);
    try {
      const connection: Connection = { ...base, endpoint: stall.url };
      await root(async () => {
        try {
          await performRequest(
            connection,
            { ...request, budget: exactBudget(5000, 120) },
            () => undefined,
            () => {
              throw new Error("decode must not run");
            },
          );
          throw new Error("accepted stall");
        } catch (cause) {
          // The timeout names the bound that fired: the capped
          // remaining budget, not the connection timeout.
          expect(transportProblem(cause)).toEqual({ kind: "timeout", milliseconds: 120 });
        }
        return undefined;
      });
      expect(stall.attempts()).toBe(1);
    } finally {
      stall.stop();
    }
  });

  test("an exhausted budget reports timeout without starting", async () => {
    const stall = stallServer(3000);
    try {
      const connection: Connection = { ...base, endpoint: stall.url };
      await root(async () => {
        try {
          await performRequest(
            connection,
            { ...request, budget: exactBudget(5000, 0) },
            () => undefined,
            decode,
          );
          throw new Error("started on an exhausted budget");
        } catch (cause) {
          expect(transportProblem(cause)).toEqual({ kind: "timeout", milliseconds: 0 });
        }
        return undefined;
      });
      expect(stall.attempts()).toBe(0);
    } finally {
      stall.stop();
    }
  });

  test("scope expiry aborts the wire and reports timeout", async () => {
    const stall = stallServer(3000);
    try {
      const connection: Connection = { ...base, endpoint: stall.url };
      const scope = createRequestScope();
      let decoded = false;
      await root(async () => {
        const raced = runWithRequestScope(scope, () =>
          performRequest(
            connection,
            request,
            () => undefined,
            () => {
              decoded = true;
              return success("late");
            },
          ),
        );
        await Bun.sleep(150);
        scope.expire("shutdown");
        try {
          await raced;
          throw new Error("survived scope expiry");
        } catch (cause) {
          expect(transportProblem(cause)).toEqual({ kind: "timeout", milliseconds: 10000 });
        }
        return undefined;
      });
      expect(decoded).toBe(false);
      await waitFor("server-side abort", stall.aborted, 2500);
      expect(stall.aborted()).toBe(true);
    } finally {
      stall.stop();
    }
  });

  test("a pre-expired scope reports timeout without starting", async () => {
    const stall = stallServer(3000);
    try {
      const connection: Connection = { ...base, endpoint: stall.url };
      const scope = createRequestScope();
      scope.expire("disconnect");
      await root(async () => {
        try {
          await runWithRequestScope(scope, () =>
            performRequest(connection, request, () => undefined, decode),
          );
          throw new Error("started on an expired scope");
        } catch (cause) {
          expect(transportProblem(cause)).toEqual({ kind: "timeout", milliseconds: 0 });
        }
        return undefined;
      });
      expect(stall.attempts()).toBe(0);
    } finally {
      stall.stop();
    }
  });

  test("a settling request under a live budget is identical to the unbounded path", async () => {
    const server = Bun.serve({
      hostname: "127.0.0.1",
      port: 0,
      fetch() {
        return new Response("ok");
      },
    });
    try {
      const connection: Connection = { ...base, endpoint: server.url.href };
      await root(async () => {
        const bounded = value(
          await performRequest(
            connection,
            { ...request, budget: createRequestBudget(5000) },
            () => undefined,
            decode,
          ),
        );
        const plain = value(await performRequest(connection, request, () => undefined, decode));
        expect(bounded).toBe("ok");
        expect(plain).toBe("ok");
        return undefined;
      });
    } finally {
      void server.stop(true);
    }
  });

  test("an ambient scope budget caps calls without explicit budgets", async () => {
    const stall = stallServer(3000);
    try {
      const connection: Connection = { ...base, endpoint: stall.url };
      const scope = createRequestScope({ totalMs: 120 });
      await root(async () => {
        try {
          await runWithRequestScope(scope, () =>
            performRequest(connection, request, () => undefined, decode),
          );
          throw new Error("accepted stall");
        } catch (cause) {
          const problem = transportProblem(cause);
          expect(problem?.kind).toBe("timeout");
          // Real clock: the bound is the ~120ms remaining budget,
          // never the 10s connection timeout.
          const ms = (problem as { milliseconds?: number }).milliseconds ?? -1;
          expect(ms).toBeGreaterThan(0);
          expect(ms).toBeLessThanOrEqual(120);
        }
        return undefined;
      });
    } finally {
      stall.stop();
    }
  });

  test("invalid connection timeouts keep their exact rejection", async () => {
    const connection: Connection = { ...base, endpoint: "http://127.0.0.1:1/" };
    await root(async () => {
      for (const bad of [0, -1, 1.5, 2147483648]) {
        await expect(
          performRequest(
            { ...connection, timeoutMilliseconds: bad },
            { ...request, budget: createRequestBudget(5000) },
            () => undefined,
            decode,
          ),
        ).rejects.toThrow("invalid transport deadline");
      }
      return undefined;
    });
  });
});

describe("http::timeout mapping", () => {
  const hash = (key: unknown) =>
    createHash("sha256")
      .update("can-concrete-type-v1\0" + JSON.stringify(key))
      .digest("hex");
  const shape = (
    kind: string,
    declaration: string,
    fields: { name: string; type: string }[] = [],
  ): FailureShape => ({
    identity: hash([kind, declaration]),
    kind,
    declaration,
    fields,
    arguments: [],
    leaves: [],
    inputs: [],
    errors: [],
  });
  const str = shape("primitive", "str"),
    int = shape("primitive", "int");
  const header = shape("record", "can.std.http@1::header", [
    { name: "name", type: str.identity },
    { name: "value", type: str.identity },
  ]);
  const headers: FailureShape = {
    ...shape("array", ""),
    identity: hash(["array", "", header.identity]),
    element: header.identity,
  };
  const declarations = catalogue.errors.filter((e) =>
    [
      "http::invalid_request",
      "http::credentials_missing",
      "http::transport_failed",
      "http::timeout",
      "http::body_limit",
      "http::status_error",
    ].includes(e.name),
  );
  const errors = declarations.map((d) =>
    shape(
      "error",
      d.identity,
      d.fields.map((f) => ({
        name: f.name,
        type: f.type === "str" ? str.identity : f.type === "int" ? int.identity : headers.identity,
      })),
    ),
  );
  const domain = createDomainRuntime({
    declarations: declarations.map((d) => ({ ...d, parameters: 0 })),
    shapes: [str, int, header, headers, ...errors],
  });
  const types = Object.fromEntries(
    ["invalid", "credential", "transport", "timeout", "limit", "status"].map((name, i) => [
      name,
      errors[i].identity,
    ]),
  ) as unknown as HTTPTypes;
  const api = createTransport(domain, { ...types, header: header.identity }, () => undefined);
  const origin = { source: "test:transport-budget", start: 0, end: 0, invocation: [] };

  test("the timeout failure names the capped bound that fired", async () => {
    const stall = stallServer(3000);
    try {
      const root = await runOwnedRoot(async () => {
        const result: Completion<unknown> = await api.request(
          { ...base, endpoint: stall.url },
          { ...request, budget: exactBudget(5000, 120) },
          () => success(0n),
          origin,
          "test:transport-budget/timeout",
        );
        expect(result.kind).toBe("domain");
        if (result.kind !== "domain") throw new Error("expected domain");
        const details = domainFailureDiagnostics(result.value);
        expect(details.declaration.name).toBe("http::timeout");
        expect(details.payload).toMatchObject({ timeout_ms: 120n });
        return success(undefined);
      });
      expect(root.cleanupFailed).toBe(false);
      expect(root.completion.kind).toBe("ok");
    } finally {
      stall.stop();
    }
  });
});

describe("Deadline.expireOn", () => {
  test("a pre-aborted scope signal expires immediately as timeout", async () => {
    const deadline = new Deadline(10000);
    try {
      const scope = createRequestScope();
      scope.expire("disconnect");
      deadline.expireOn(scope.signal);
      try {
        await deadline.wait(Bun.sleep(50).then(() => "late"));
        throw new Error("survived expiry");
      } catch (cause) {
        expect(transportProblem(cause)).toEqual({ kind: "timeout", milliseconds: 10000 });
      }
    } finally {
      deadline.dispose();
    }
  });

  test("mid-flight scope expiry reports timeout, never cancelled", async () => {
    const deadline = new Deadline(10000);
    try {
      const scope = createRequestScope();
      deadline.expireOn(scope.signal);
      const raced = deadline.wait(Bun.sleep(5000).then(() => "late"));
      await Bun.sleep(50);
      scope.expire("shutdown");
      try {
        await raced;
        throw new Error("survived expiry");
      } catch (cause) {
        expect(transportProblem(cause)).toEqual({ kind: "timeout", milliseconds: 10000 });
      }
      expect(deadline.signal.aborted).toBe(true);
    } finally {
      deadline.dispose();
    }
  });

  test("dispose unlinks the scope signal", async () => {
    const deadline = new Deadline(10000);
    const scope = createRequestScope();
    deadline.expireOn(scope.signal);
    deadline.dispose();
    scope.expire("shutdown");
    // The disposed deadline no longer observes the scope: the wait
    // settles with the operation instead of expiring.
    await expect(deadline.wait(Bun.sleep(10).then(() => "settled"))).resolves.toBe("settled");
    deadline.dispose();
  });
});
