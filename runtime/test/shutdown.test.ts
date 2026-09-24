// Bounded shutdown and cleanup-precedence contract (T18). Close and shutdown
// deadlines bound only the caller's wait: native work stays owned until it
// settles, disconnect never aborts an owned handler, and a cleanup failure
// never replaces an already selected outcome. Abort exists only where an
// operation offers it; there is no general finally, cancellation, or
// consumed-fault feature.
import { test, expect } from "bun:test";
import { createHash } from "node:crypto";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, domainFailureDiagnostics, type FailureShape } from "../domain.ts";
import { captureStandard, standardFailureKind } from "../failure.ts";
import { success, failure, value, invoke, type Completion } from "../completion.ts";
import {
  runOwnedRoot,
  registerResource,
  useResource,
  closeResource,
  resourceStatus,
  withScope,
  type OwnerDiagnostic,
} from "../owner.ts";
import { settle, handle } from "../coordination.ts";
import { dataArray, record } from "../data.ts";
import { copyBytes } from "../bytes.ts";
import { createStreamReads, openByteCell } from "../transport/stream/readable.ts";
import { registerReader } from "../transport/stream/lifecycle.ts";
import { createServer } from "../platform/server.ts";
import { createRouter } from "../platform/router.ts";
import { createResponses } from "../platform/http.ts";

const origin = { source: "can:test", start: 0, end: 0, invocation: [] };
const identity = (kind: string, declaration: string) =>
  createHash("sha256")
    .update("can-concrete-type-v1\0" + JSON.stringify([kind, declaration]))
    .digest("hex");
const textShape: FailureShape = {
  identity: identity("primitive", "str"),
  kind: "primitive",
  declaration: "str",
  arguments: [],
  fields: [],
  leaves: [],
  inputs: [],
  errors: [],
};
const intShape: FailureShape = {
  identity: identity("primitive", "int"),
  kind: "primitive",
  declaration: "int",
  arguments: [],
  fields: [],
  leaves: [],
  inputs: [],
  errors: [],
};
const fieldNames: Record<string, string[]> = {
  "http::invalid_server_config": ["reason"],
  "http::bind_failed": ["address"],
  "http::shutdown_failed": ["phase"],
  "http::invalid_route": ["reason"],
  "http::duplicate_route": ["method", "path"],
  "http::ambiguous_route": ["first", "second"],
  "http::invalid_request": ["reason"],
  "http::body_limit": ["limit"],
  "codec::invalid_data": ["path", "reason"],
  "stream::read_failed": ["reason"],
  "stream::write_failed": ["reason"],
  "stream::cancelled": ["reason"],
  "stream::close_failed": ["reason"],
  "files::limit_exceeded": ["limit"],
};
const declarations = catalogue.errors
  .filter((e) => fieldNames[e.name] !== undefined)
  .map((e) => ({ identity: e.identity, name: e.name, id: e.id, parameters: 0 }));
const fieldKinds = new Map(
  catalogue.errors.flatMap((e) => e.fields.map((f) => [e.name + ":" + f.name, f.type])),
);
const errorShapes: FailureShape[] = declarations.map((e) => ({
  identity: identity("error", e.identity),
  kind: "error",
  declaration: e.identity,
  arguments: [],
  fields: fieldNames[e.name]!.map((name) => ({
    name,
    type: fieldKinds.get(e.name + ":" + name) === "int" ? intShape.identity : textShape.identity,
  })),
  leaves: [],
  inputs: [],
  errors: [],
}));
const domain = createDomainRuntime({ declarations, shapes: [textShape, intShape, ...errorShapes] });
const id = (declaration: string) => identity("error", declaration);
const server = createServer(domain, {
  invalidConfig: id("can.std.http@1::invalid_server_config"),
  bindFailed: id("can.std.http@1::bind_failed"),
  shutdownFailed: id("can.std.http@1::shutdown_failed"),
});
const router = createRouter(domain, {
  invalid: id("can.std.http@1::invalid_route"),
  duplicate: id("can.std.http@1::duplicate_route"),
  ambiguous: id("can.std.http@1::ambiguous_route"),
});
const responses = createResponses(domain, {
  invalid: id("can.std.http@1::invalid_request"),
  invalidData: id("can.std.codec@1::invalid_data"),
  close: id("can.std.stream@1::close_failed"),
  writeFailed: id("can.std.stream@1::write_failed"),
  limit: id("can.std.http@1::body_limit"),
});
const reads = createStreamReads(domain, {
  readFailed: id("can.std.stream@1::read_failed"),
  cancelled: id("can.std.stream@1::cancelled"),
  closeFailed: id("can.std.stream@1::close_failed"),
  limitExceeded: id("can.std.files@1::limit_exceeded"),
});
const streamFail = (
  identity: string,
  fields: readonly (readonly [string, unknown])[],
  cause?: unknown,
) => failure(domain.create(identity, record(identity, fields), origin, cause));

function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((r) => {
    resolve = r;
  });
  return { promise, resolve };
}
const tick = async () => {
  for (let i = 0; i < 12; i++) await Promise.resolve();
};
async function textResult(body: string): Promise<Completion<unknown>> {
  return responses.text(value(await responses.ok()), value(await responses.emptyHeaders()), body);
}
async function startOn(
  port: number,
  handler: (request?: unknown) => Promise<Completion<unknown>>,
): Promise<{ token: unknown }> {
  const config = value(await server.makeConfig("127.0.0.1", BigInt(port), 1048576n, 5000n));
  const get = value(await router.get("/x", handler));
  const post = value(await router.post("/x", handler));
  const table = value(await router.make([get, post]));
  const started = await server.start(config, table);
  if (started.kind !== "ok") throw new Error("test server failed to start");
  return { token: value(started) };
}
async function withoutUnhandled<T>(run: () => Promise<T>): Promise<T> {
  const seen: unknown[] = [];
  const listener = (reason: unknown) => {
    seen.push(reason);
  };
  process.on("unhandledRejection", listener);
  try {
    const out = await run();
    await Bun.sleep(20);
    expect(seen).toEqual([]);
    return out;
  } finally {
    process.off("unhandledRejection", listener);
  }
}

test("the original failure wins over close failure during scope cleanup", async () => {
  const primary = failure(captureStandard(new Error("primary"), origin));
  const diagnostics: OwnerDiagnostic[] = [];
  const result = await runOwnedRoot(
    async () =>
      withScope(async () => {
        registerResource(
          "test.cleanup",
          {},
          () => {
            throw new Error("close fault");
          },
          { scopeManaged: true },
        );
        return primary;
      }),
    (diagnostic) => {
      diagnostics.push(diagnostic);
    },
  );
  expect(result.completion).toBe(primary);
  expect(result.cleanupFailed).toBe(true);
  expect(diagnostics.map((diagnostic) => diagnostic.phase)).toEqual(["cleanup"]);
});

test("concurrent close runs the native adapter exactly once", async () => {
  const gate = deferred<void>();
  let calls = 0;
  const result = await runOwnedRoot(async () => {
    const token = registerResource("test.once", {}, async () => {
      calls++;
      await gate.promise;
      return success(undefined);
    });
    const first = closeResource(token, "test.once");
    const second = invoke(() => closeResource(token, "test.once"), origin);
    expect(resourceStatus(token).state).toBe("closing");
    gate.resolve();
    const [closed, refused] = await Promise.all([first, second]);
    expect(closed.kind).toBe("ok");
    expect(refused.kind).toBe("standard");
    if (refused.kind === "standard")
      expect(standardFailureKind(refused.value)).toBe("resource_state");
    expect(resourceStatus(token).state).toBe("closed");
    return success(undefined);
  });
  expect(calls).toBe(1);
  expect(result.cleanupFailed).toBe(false);
});

test("a scope-owned handle returned as the scope result rejects later use", async () => {
  const outer = await runOwnedRoot(async () => {
    const scoped = await withScope(async () => {
      const token = registerResource("test.scoped", { n: 1 }, () => success(undefined), {
        scopeManaged: true,
      });
      return success(token);
    });
    expect(scoped.kind).toBe("ok");
    if (scoped.kind !== "ok") throw new Error("wrong outcome");
    expect(resourceStatus(scoped.value).state).toBe("closed");
    const refused = await invoke(
      () => useResource(scoped.value, "test.scoped", () => success(1)),
      origin,
    );
    expect(refused.kind).toBe("standard");
    if (refused.kind === "standard")
      expect(standardFailureKind(refused.value)).toBe("resource_state");
    return success(undefined);
  });
  expect(outer.cleanupFailed).toBe(false);
});

test("race returns the native winner while pending losers keep their leases", async () => {
  const gate = deferred<void>(),
    published = deferred<void>();
  const events: number[] = [];
  const diagnostics: OwnerDiagnostic[] = [];
  let finished = false;
  const late = streamFail(id("can.std.stream@1::read_failed"), [["reason", "late"]]);
  const root = runOwnedRoot(
    async () => {
      const choice = await settle("race", [
        { captures: [], run: () => success("winner") },
        {
          captures: [],
          run: async () => {
            await gate.promise;
            return late;
          },
        },
        {
          captures: [],
          run: async () => {
            await gate.promise;
            return success("slow");
          },
        },
      ]);
      published.resolve();
      return handle(
        choice,
        {
          each: () => {
            throw new Error("losing handler ran");
          },
          shared: (index, completion) => {
            events.push(index);
            return completion;
          },
          allFailed: () => {
            throw new Error("unexpected aggregate");
          },
        },
        false,
        origin,
      );
    },
    (diagnostic) => {
      diagnostics.push(diagnostic);
    },
  ).then((result) => {
    finished = true;
    return result;
  });
  await published.promise;
  await tick();
  expect(finished).toBe(false);
  gate.resolve();
  const result = await root;
  expect(result.completion).toMatchObject({ kind: "ok", value: "winner" });
  expect(events).toEqual([0]);
  expect(diagnostics).toEqual([]);
});

test("cancel aborts only the targeted reader", async () => {
  let release!: () => void;
  const gate = new Promise<void>((resolve) => {
    release = resolve;
  });
  let started = 0;
  const source = () => {
    let pulls = 0;
    return new ReadableStream<Uint8Array>({
      async pull(controller) {
        started++;
        await gate;
        pulls++;
        try {
          if (pulls === 1) controller.enqueue(new Uint8Array([7]));
          else controller.close();
        } catch {}
      },
    });
  };
  const result = await runOwnedRoot(
    async () => {
      const first = registerReader(
        openByteCell(source(), 8n),
        streamFail,
        id("can.std.stream@1::close_failed"),
      );
      const second = registerReader(
        openByteCell(source(), 8n),
        streamFail,
        id("can.std.stream@1::close_failed"),
      );
      const interrupted = reads.readMany(first, 4n);
      const spared = reads.readMany(second, 4n);
      while (started < 2) await Promise.resolve();
      expect(await reads.cancelReader(first, "deadline")).toEqual(success(undefined));
      release();
      const [cancelled, delivered] = await Promise.all([interrupted, spared]);
      expect(cancelled.kind).toBe("domain");
      if (cancelled.kind !== "domain") throw new Error("wrong outcome");
      const details = domainFailureDiagnostics(cancelled.value);
      expect(details.declaration.name).toBe("stream::cancelled");
      expect(details.payload).toMatchObject({ reason: "deadline" });
      if (delivered.kind !== "ok") throw new Error("sibling read failed");
      expect(
        dataArray(delivered.value).map((chunk) => Array.from(copyBytes(chunk, origin))),
      ).toEqual([[7]]);
      expect(await reads.closeReader(second)).toEqual(success(undefined));
      process.stderr.write("FIRST-STATUS " + JSON.stringify(resourceStatus(first)) + "\n");
      process.stderr.write("SECOND-STATUS " + JSON.stringify(resourceStatus(second)) + "\n");
      const refused = await invoke(() => reads.closeReader(first), origin);
      process.stderr.write("REFUSED " + refused.kind + "\n");
      expect(refused.kind).toBe("standard");
      if (refused.kind === "standard")
        expect(standardFailureKind(refused.value)).toBe("resource_state");
      return success(undefined);
    },
    (d) => console.log("DIAG", JSON.stringify(d)),
  );
  expect(result.cleanupFailed).toBe(false);
});

test("client disconnect does not abort owned handler work", async () => {
  await withoutUnhandled(async () => {
    const owned = await runOwnedRoot(async () => {
      let entered!: () => void;
      const gate = new Promise<void>((resolve) => {
        entered = resolve;
      });
      let release!: () => void;
      const held = new Promise<void>((resolve) => {
        release = resolve;
      });
      let finished = false;
      const { token } = await startOn(18421, async () => {
        entered();
        await held;
        finished = true;
        return textResult("done");
      });
      const controller = new AbortController();
      const pending = fetch("http://127.0.0.1:18421/x", { signal: controller.signal });
      const outcome = pending.then(
        () => "delivered",
        () => "aborted",
      );
      await gate;
      controller.abort();
      expect(await outcome).toBe("aborted");
      expect(finished).toBe(false);
      release();
      expect((await server.stop(token)).kind).toBe("ok");
      expect(finished).toBe(true);
      expect(resourceStatus(token)).toMatchObject({ state: "closed", leases: 0 });
      return success(undefined);
    });
    expect(owned.cleanupFailed).toBe(false);
    expect(owned.completion.kind).toBe("ok");
  });
});
