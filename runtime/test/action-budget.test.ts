// E04: browser-action fetch bound/cancel legs (cancel-present branch).
//
// Native fetch abort is qualified, so a bound expiry aborts the wire
// request and reports timeout — distinct from caller-cancelled abort.
// Servers bind ephemeral loopback ports, so every run uses distinct ports.
import { describe, expect, test } from "bun:test";
import { createHash } from "node:crypto";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, type FailureShape } from "../domain.ts";
import { errorPayload, errorType, type Completion } from "../completion.ts";
import { dataProperty } from "../data.ts";
import type { Schema } from "../codec/json.ts";
import {
  createJsonActionFetch,
  fetchJsonAction,
  type JsonFetchSite,
} from "../platform/action-json.ts";
import { createRequestBudget } from "../transport/request-budget.ts";

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

const strNode = { identity: "t24::str", kind: "primitive", name: "str" };
const pingResponse: Schema = {
  root: "t24::ping_outcome",
  nodes: [
    {
      identity: "t24::ping_outcome",
      kind: "variant",
      name: "ping_outcome",
      leaves: ["t24::pong", "t24::missing"],
    },
    {
      identity: "t24::pong",
      kind: "record",
      name: "pong",
      fields: [{ name: "label", type: "t24::str" }],
    },
    {
      identity: "t24::missing",
      kind: "record",
      name: "missing",
      fields: [{ name: "reason", type: "t24::str" }],
    },
    strNode,
  ],
};
const pingCases = [
  { leaf: "t24::pong", status: 200 },
  { leaf: "t24::missing", status: 404 },
];
const base = { method: "GET" as const, response: pingResponse, cases: pingCases };

type StallServer = {
  url: string;
  hits: () => number;
  aborted: () => boolean;
  stop: () => void;
};

// Every path stalls: legs assert the client bound, never routing.
function stallServer(stillMs: number): StallServer {
  let hits = 0;
  let aborted = false;
  const server = Bun.serve({
    hostname: "127.0.0.1",
    port: 0,
    async fetch(request) {
      hits++;
      requestSignal(request)?.addEventListener(
        "abort",
        () => {
          aborted = true;
        },
        { once: true },
      );
      await Bun.sleep(stillMs);
      return Response.json({ case: "pong", value: { label: "late" } });
    },
  });
  return {
    url: server.url.href,
    hits: () => hits,
    aborted: () => aborted,
    stop: () => {
      void server.stop(true);
    },
  };
}

describe("fetchJsonAction bounds", () => {
  test("a bound expiry aborts the wire natively and reports timeout", async () => {
    const stall = stallServer(3000);
    try {
      const started = Date.now();
      const outcome = await fetchJsonAction({ ...base, url: `${stall.url}ok`, timeoutMs: 150 });
      expect(Date.now() - started).toBeLessThan(1500);
      expect(outcome).toEqual({ kind: "timeout" });
      expect(stall.hits()).toBe(1);
      // The abort propagated to the server: the wire request really
      // stopped (cancel-present), it did not run on detached.
      await waitFor("server-side abort", stall.aborted, 2500);
      expect(stall.aborted()).toBe(true);
    } finally {
      stall.stop();
    }
  });

  test("caller cancel still reports aborted, never timeout", async () => {
    const stall = stallServer(3000);
    try {
      const stop = new AbortController();
      const raced = fetchJsonAction({
        ...base,
        url: `${stall.url}ok`,
        timeoutMs: 10000,
        signal: stop.signal,
      });
      await Bun.sleep(100);
      stop.abort();
      expect(await raced).toEqual({ kind: "aborted" });
    } finally {
      stall.stop();
    }
  });

  test("a pre-aborted signal returns without fetching", async () => {
    const stall = stallServer(3000);
    try {
      const stop = new AbortController();
      stop.abort();
      const outcome = await fetchJsonAction({
        ...base,
        url: `${stall.url}ok`,
        timeoutMs: 10000,
        signal: stop.signal,
      });
      expect(outcome).toEqual({ kind: "aborted" });
      expect(stall.hits()).toBe(0);
    } finally {
      stall.stop();
    }
  });

  test("the remaining budget caps the caller bound", async () => {
    const stall = stallServer(3000);
    try {
      const budget = createRequestBudget(120);
      const started = Date.now();
      const outcome = await fetchJsonAction({
        ...base,
        url: `${stall.url}ok`,
        timeoutMs: 10000,
        budget,
      });
      expect(Date.now() - started).toBeLessThan(1500);
      expect(outcome).toEqual({ kind: "timeout" });
    } finally {
      stall.stop();
    }
  });

  test("an exhausted budget returns timeout without fetching", async () => {
    const stall = stallServer(3000);
    try {
      let at = 1000;
      const budget = createRequestBudget(10, () => at);
      at += 50;
      const outcome = await fetchJsonAction({
        ...base,
        url: `${stall.url}ok`,
        timeoutMs: 10000,
        budget,
      });
      expect(outcome).toEqual({ kind: "timeout" });
      expect(stall.hits()).toBe(0);
    } finally {
      stall.stop();
    }
  });

  test("a settling fetch under a live bound is identical to the unbounded path", async () => {
    const server = Bun.serve({
      hostname: "127.0.0.1",
      port: 0,
      fetch() {
        return Response.json({ case: "pong", value: { label: "here" } });
      },
    });
    try {
      const bounded = await fetchJsonAction({
        ...base,
        url: `${server.url.href}ok`,
        timeoutMs: 5000,
      });
      expect(bounded.kind).toBe("ok");
      if (bounded.kind !== "ok") throw new Error("wrong outcome");
      expect(bounded.leaf).toBe("t24::pong");
      expect(dataProperty(bounded.value, "label")).toBe("here");
      const plain = await fetchJsonAction({ ...base, url: `${server.url.href}ok` });
      expect(plain).toEqual(bounded);
    } finally {
      void server.stop(true);
    }
  });

  test("a body stall after headers reports timeout, not a partial read", async () => {
    const encoder = new TextEncoder();
    const server = Bun.serve({
      hostname: "127.0.0.1",
      port: 0,
      fetch() {
        // Headers plus a first chunk arrive; the rest never does.
        const stream = new ReadableStream({
          start(controller) {
            controller.enqueue(encoder.encode('{"case":"pong","value":{"label":"part'));
          },
        });
        return new Response(stream, { headers: { "content-type": "application/json" } });
      },
    });
    try {
      const outcome = await fetchJsonAction({
        ...base,
        url: `${server.url.href}drip`,
        timeoutMs: 200,
      });
      expect(outcome).toEqual({ kind: "timeout" });
    } finally {
      void server.stop(true);
    }
  });

  test("bad caller bounds throw before any byte is sent", async () => {
    const stall = stallServer(3000);
    try {
      for (const bad of [0, -1, 1.5, Number.NaN, 2147483648]) {
        await expect(
          fetchJsonAction({ ...base, url: `${stall.url}ok`, timeoutMs: bad }),
        ).rejects.toThrow(TypeError);
      }
      expect(stall.hits()).toBe(0);
    } finally {
      stall.stop();
    }
  });

  test("validation runs before the race, even on an exhausted budget", async () => {
    let at = 1000;
    const budget = createRequestBudget(10, () => at);
    at += 50;
    await expect(
      fetchJsonAction({
        method: "POST",
        url: "http://127.0.0.1:1/save",
        response: pingResponse,
        cases: pingCases,
        budget,
      }),
    ).rejects.toThrow("POST needs a body value");
  });
});

describe("action fetch lowering", () => {
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
  const errorNames = [
    "http::transport_failed",
    "http::invalid_request",
    "http::body_limit",
    "http::status_error",
    "codec::invalid_data",
  ];
  const declarations = catalogue.errors.filter((e) => errorNames.includes(e.name));
  const fieldKinds = new Map(
    catalogue.errors.flatMap((e) => e.fields.map((f) => [e.name + ":" + f.name, f.type])),
  );
  const errors = declarations.map((d) =>
    shape(
      "error",
      d.identity,
      d.fields.map((f) => ({
        name: f.name,
        type:
          fieldKinds.get(d.name + ":" + f.name) === "int"
            ? int.identity
            : fieldKinds.get(d.name + ":" + f.name) === "http::header[]"
              ? headers.identity
              : str.identity,
      })),
    ),
  );
  const domain = createDomainRuntime({
    declarations: declarations.map((d) => ({ ...d, parameters: 0 })),
    shapes: [str, int, header, headers, ...errors],
  });
  const id = (declaration: string) => hash(["error", declaration]);
  const api = createJsonActionFetch(domain, {
    transport: id("can.std.http@1::transport_failed"),
    invalidRequest: id("can.std.http@1::invalid_request"),
    bodyLimit: id("can.std.http@1::body_limit"),
    statusError: id("can.std.http@1::status_error"),
    invalidData: id("can.std.codec@1::invalid_data"),
    header: header.identity,
  });

  function checkFailure(result: Completion<unknown>, identity: string, payload: object) {
    expect(result.kind).toBe("domain");
    if (result.kind !== "domain") throw new Error("expected domain failure");
    expect(errorType(result)).toBe(identity);
    expect(errorPayload(result)).toMatchObject(payload);
  }

  // withForwardingFetch runs the adapter through the same-origin path a
  // browser takes: the adapter only ever builds relative URLs, and the
  // stub asserts that relativity before forwarding to the loopback
  // server on its ephemeral port.
  async function withForwardingFetch<T>(port: number, run: () => Promise<T>): Promise<T> {
    const realFetch = globalThis.fetch;
    globalThis.fetch = (async (url: unknown, init?: unknown) => {
      expect(typeof url).toBe("string");
      const target = url as string;
      expect(target.startsWith("/") && !target.startsWith("//")).toBe(true);
      return realFetch(`http://127.0.0.1:${port}${target}`, init as never);
    }) as unknown as typeof fetch;
    try {
      return await run();
    } finally {
      globalThis.fetch = realFetch;
    }
  }

  test("a site bound expiry lowers to transport_failed phase timeout", async () => {
    const stall = stallServer(3000);
    try {
      const port = new URL(stall.url).port;
      const site: JsonFetchSite = {
        action: "t24::ping",
        method: "GET",
        path: "/slow",
        captures: [],
        response: pingResponse,
        cases: pingCases,
        timeoutMs: 150,
      };
      const outcome = await withForwardingFetch(Number(port), () =>
        api.get("ping", site, undefined),
      );
      checkFailure(outcome, id("can.std.http@1::transport_failed"), { phase: "timeout" });
    } finally {
      stall.stop();
    }
  });

  test("a settling site call under its bound returns the domain case", async () => {
    const server = Bun.serve({
      hostname: "127.0.0.1",
      port: 0,
      fetch() {
        return Response.json({ case: "pong", value: { label: "here" } });
      },
    });
    try {
      const port = new URL(server.url.href).port;
      const site: JsonFetchSite = {
        action: "t24::ping",
        method: "GET",
        path: "/fast",
        captures: [],
        response: pingResponse,
        cases: pingCases,
        timeoutMs: 5000,
      };
      const outcome = await withForwardingFetch(Number(port), () =>
        api.get("ping", site, undefined),
      );
      expect(outcome.kind).toBe("ok");
      if (outcome.kind !== "ok") throw new Error("wrong outcome");
      expect(dataProperty(outcome.value, "label")).toBe("here");
    } finally {
      void server.stop(true);
    }
  });

  test("a malformed site bound fails closed before any byte is sent", async () => {
    const stall = stallServer(3000);
    try {
      const port = new URL(stall.url).port;
      for (const bad of [0, -10, 1.5, Number.NaN, 2147483648]) {
        const site: JsonFetchSite = {
          action: "t24::ping",
          method: "GET",
          path: "/slow",
          captures: [],
          response: pingResponse,
          cases: pingCases,
          timeoutMs: bad,
        };
        await expect(
          withForwardingFetch(Number(port), () => api.get("ping", site, undefined)),
        ).rejects.toThrow("malformed caller bound");
      }
      expect(stall.hits()).toBe(0);
    } finally {
      stall.stop();
    }
  });
});
