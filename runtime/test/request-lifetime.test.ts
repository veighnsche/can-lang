import { test, expect } from "bun:test";
import { createHash } from "node:crypto";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, type FailureShape } from "../domain.ts";
import { isStandardFailure, standardFailureDiagnostics, type StandardFailure } from "../failure.ts";
import { success, value, type Completion } from "../completion.ts";
import { array, dataArray, dataProperty, recordIdentity } from "../data.ts";
import { copyBytes } from "../bytes.ts";
import { launchOwned, resourceStatus, runOwnedRoot } from "../owner.ts";
import {
  abandonRequest,
  claimUpgrade,
  createRequests,
  createResponses,
  isHTTPValue,
  isRequest,
  offeredProtocols,
  requestSnapshot,
  revokeRequest,
  snapshotBodyBytes,
  snapshotRequest,
  snapshotRequestLazy,
} from "../platform/http.ts";
import { createRouter, dispatch } from "../platform/router.ts";
import { createServer } from "../platform/server.ts";
import { createWebSockets } from "../platform/websocket.ts";
import { createStreamReads } from "../transport/stream/readable.ts";

const hash = (kind: string, name: string) =>
  createHash("sha256")
    .update("can-concrete-type-v1\0" + JSON.stringify([kind, name]))
    .digest("hex");
const shape = (
  kind: string,
  declaration: string,
  fields: { name: string; type: string }[] = [],
): FailureShape => ({
  identity: hash(kind, declaration),
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
const declarations = catalogue.errors
  .filter((e) =>
    [
      "http::invalid_server_config",
      "http::bind_failed",
      "http::shutdown_failed",
      "http::invalid_route",
      "http::duplicate_route",
      "http::ambiguous_route",
      "http::invalid_request",
      "http::body_limit",
      "codec::invalid_data",
      "files::limit_exceeded",
      "stream::read_failed",
      "stream::write_failed",
      "stream::cancelled",
      "stream::close_failed",
      "ws::connect_failed",
      "ws::upgrade_failed",
      "ws::unsupported_protocol",
      "ws::send_failed",
      "ws::invalid_close",
      "ws::limit_exceeded",
      "ws::invalid_url",
      "ws::invalid_protocol",
    ].includes(e.name),
  )
  .map((e) => ({ ...e, parameters: 0 }));
const errors = catalogue.errors
  .filter((e) => declarations.some((d) => d.identity === e.identity))
  .map((e) =>
    shape(
      "error",
      e.identity,
      e.fields.map((f) => ({
        name: f.name,
        type: f.type === "int" ? int.identity : str.identity,
      })),
    ),
  );
const domain = createDomainRuntime({ declarations, shapes: [str, int, ...errors] });
const identity = (name: string) => errors.find((e) => e.declaration === name)!.identity;
const TEXT = "can.std.ws@1::text",
  BINARY = "can.std.ws@1::binary",
  DRAIN = "can.std.ws@1::drain",
  CLOSE = "can.std.ws@1::closed",
  CONNECTION = "can.std.ws@1::connection";
const servers = createServer(domain, {
  invalidConfig: identity("can.std.http@1::invalid_server_config"),
  bindFailed: identity("can.std.http@1::bind_failed"),
  shutdownFailed: identity("can.std.http@1::shutdown_failed"),
});
const routing = createRouter(domain, {
  invalid: identity("can.std.http@1::invalid_route"),
  duplicate: identity("can.std.http@1::duplicate_route"),
  ambiguous: identity("can.std.http@1::ambiguous_route"),
});
const requests = createRequests(domain, {
  invalid: identity("can.std.http@1::invalid_request"),
  limit: identity("can.std.http@1::body_limit"),
  invalidData: identity("can.std.codec@1::invalid_data"),
  header: "header",
  close: identity("can.std.stream@1::close_failed"),
  writeFailed: identity("can.std.stream@1::write_failed"),
  multipartForm: "unused",
  multipartField: "unused",
  multipartFile: "unused",
});
const responses = createResponses(domain, {
  invalid: identity("can.std.http@1::invalid_request"),
  invalidData: identity("can.std.codec@1::invalid_data"),
  close: identity("can.std.stream@1::close_failed"),
  writeFailed: identity("can.std.stream@1::write_failed"),
  limit: identity("can.std.http@1::body_limit"),
});
const reads = createStreamReads(domain, {
  readFailed: identity("can.std.stream@1::read_failed"),
  cancelled: identity("can.std.stream@1::cancelled"),
  closeFailed: identity("can.std.stream@1::close_failed"),
  limitExceeded: identity("can.std.files@1::limit_exceeded"),
});
const ws = createWebSockets(domain, {
  connectFailed: identity("can.std.ws@1::connect_failed"),
  upgradeFailed: identity("can.std.ws@1::upgrade_failed"),
  unsupportedProtocol: identity("can.std.ws@1::unsupported_protocol"),
  sendFailed: identity("can.std.ws@1::send_failed"),
  invalidClose: identity("can.std.ws@1::invalid_close"),
  limitExceeded: identity("can.std.ws@1::limit_exceeded"),
  invalidUrl: identity("can.std.ws@1::invalid_url"),
  invalidProtocol: identity("can.std.ws@1::invalid_protocol"),
  closeFailed: identity("can.std.stream@1::close_failed"),
  text: TEXT,
  binary: BINARY,
  drain: DRAIN,
  close: CLOSE,
  connection: CONNECTION,
});
const origin = { source: "test", start: 0, end: 0, invocation: [] };
async function textResult(body: string): Promise<Completion<unknown>> {
  return responses.text(value(await responses.ok()), value(await responses.emptyHeaders()), body);
}
// Every retained operation on a revoked token must raise the resource-state
// failure, whether it throws synchronously or rejects asynchronously.
async function expectResourceState(call: () => unknown): Promise<void> {
  try {
    await call();
  } catch (thrown) {
    expect(isStandardFailure(thrown)).toBe(true);
    expect(standardFailureDiagnostics(thrown as StandardFailure).kind).toBe("resource_state");
    return;
  }
  throw new Error("expected resource_state failure");
}
async function expectRevoked(token: unknown): Promise<void> {
  expect(isRequest(token)).toBe(false);
  expect(isHTTPValue("request", token)).toBe(false);
  await expectResourceState(() => requestSnapshot(token));
  await expectResourceState(() => requests.method(token));
  await expectResourceState(() => requests.path(token));
  await expectResourceState(() => requests.headers(token));
  await expectResourceState(() => requests.queryAll(token, "q"));
  await expectResourceState(() => requests.queryOne(token, "q"));
  await expectResourceState(() => requests.body(token, 100n));
  await expectResourceState(() => requests.bodyStream(token, 4n));
  await expectResourceState(() => snapshotBodyBytes(token));
  await expectResourceState(() => offeredProtocols(token));
  await expectResourceState(() => claimUpgrade(token));
  await expectResourceState(() => ws.accept(token, "", 65536n, 64n, 1048576n));
}
async function untilRevoked(token: unknown): Promise<void> {
  for (let n = 0; n < 500 && isRequest(token); n++) await Bun.sleep(10);
  expect(isRequest(token)).toBe(false);
}
async function untilDone(flag: { done?: boolean }): Promise<void> {
  for (let n = 0; n < 500 && !flag.done; n++) await Bun.sleep(10);
  expect(flag.done).toBe(true);
}
test("abandonment preserves the capability while revocation ends it", async () => {
  // Production form of the request-lifetime probe: abandonRequest is body
  // cleanup, not a lifetime guard, for buffered and lazy tokens alike.
  for (const kind of ["buffered", "lazy"] as const) {
    const native = new Request("https://example.test/invoices/7", {
      method: "POST",
      headers: { "x-probe": kind },
      body: "draft",
    });
    const captured =
      kind === "buffered"
        ? await snapshotRequest(native, 4096)
        : await snapshotRequestLazy(native, 4096);
    if (captured.kind !== "request") throw new Error(`unexpected ${kind} rejection`);
    const token = captured.value;
    expect(isRequest(token)).toBe(true);
    expect(requestSnapshot(token).method).toBe("POST");
    await abandonRequest(token);
    expect(isRequest(token)).toBe(true);
    expect(requestSnapshot(token).method).toBe("POST");
    const kept = value(await requests.headers(token));
    expect(kept.some((entry) => dataProperty(entry, "name") === "x-probe")).toBe(true);
    revokeRequest(token);
    await expectRevoked(token);
  }
});
test("revocation is silent and idempotent on foreign input", async () => {
  let traps = 0;
  const forged = new Proxy(
    {},
    {
      get() {
        traps++;
        throw Error("secret");
      },
      getPrototypeOf() {
        traps++;
        throw Error("secret");
      },
    },
  );
  const native = new Request("http://localhost/", { method: "POST", body: "x" });
  const captured = await snapshotRequest(native, 64);
  if (captured.kind !== "request") throw new Error("rejected request");
  revokeRequest(captured.value);
  await expectRevoked(captured.value);
  expect(() => {
    revokeRequest(captured.value);
    revokeRequest(null);
    revokeRequest(undefined);
    revokeRequest(42);
    revokeRequest("token");
    revokeRequest({});
    revokeRequest(Object.freeze(Object.create(null)));
    revokeRequest(forged);
  }).not.toThrow();
  expect(traps).toBe(0);
});
test("rejected routes abandon and revoke without running a handler", async () => {
  let calls = 0;
  const router = value(
    await routing.make(
      array([
        value(
          await routing.get("/x", async () => {
            calls++;
            return textResult("ok");
          }),
        ),
      ]),
    ),
  );
  const missing = await snapshotRequest(new Request("http://localhost/missing"), 64);
  if (missing.kind !== "request") throw new Error("rejected request");
  expect(value(await dispatch(router, missing.value)).status).toBe(404);
  await expectRevoked(missing.value);
  const wrong = await snapshotRequest(new Request("http://localhost/x", { method: "PUT" }), 64);
  if (wrong.kind !== "request") throw new Error("rejected request");
  const denied = value(await dispatch(router, wrong.value));
  expect(denied.status).toBe(405);
  expect(denied.headers.get("allow")).toBe("GET");
  await expectRevoked(wrong.value);
  let cancelled = 0;
  const stream = new ReadableStream<Uint8Array>({
    start(c) {
      c.enqueue(new TextEncoder().encode("unread"));
    },
    cancel() {
      cancelled++;
    },
  });
  const lazy = await snapshotRequestLazy(
    new Request("http://localhost/missing", { method: "POST", body: stream }),
    64,
  );
  if (lazy.kind !== "request") throw new Error("rejected request");
  expect(value(await dispatch(router, lazy.value)).status).toBe(404);
  expect(cancelled).toBe(1);
  await expectRevoked(lazy.value);
  expect(calls).toBe(0);
});
test("buffered server requests revoke after the response", async () => {
  const seen: { token?: unknown; during?: unknown } = {};
  const owned = await runOwnedRoot(async () => {
    const config = value(await servers.makeConfig("127.0.0.1", 18370n, 1048576n, 5000n));
    const route = value(
      await routing.post("/echo", async (request) => {
        seen.token = request;
        seen.during = {
          method: value(await requests.method(request)),
          path: value(await requests.path(request)),
          body: new TextDecoder().decode(
            copyBytes(value(await requests.body(request, 100n)), origin),
          ),
        };
        return textResult("ok");
      }),
    );
    const table = value(await routing.make(array([route])));
    const token = value(await servers.start(config, table));
    const response = await fetch("http://127.0.0.1:18370/echo", {
      method: "POST",
      headers: { "x-probe": "buffered" },
      body: "draft",
    });
    expect(response.status).toBe(200);
    expect(await response.text()).toBe("ok");
    // A completed response orders after the outer revoke, so the retained
    // token is already dead here without any further waiting.
    await expectRevoked(seen.token);
    expect(resourceStatus(token)).toMatchObject({ state: "open", leases: 0 });
    expect((await servers.stop(token)).kind).toBe("ok");
    return success(undefined);
  });
  expect(owned.cleanupFailed).toBe(false);
  expect(owned.completion.kind).toBe("ok");
  expect(seen.during).toEqual({ method: "POST", path: "/echo", body: "draft" });
});
test("lazy server requests abandon unread bodies and revoke", async () => {
  const seen: { skipped?: unknown; partial?: unknown } = {};
  const owned = await runOwnedRoot(async () => {
    const config = value(await servers.makeConfig("127.0.0.1", 18371n, 1048576n, 5000n));
    const skip = value(
      await routing.stream(
        value(
          await routing.post("/skip", async (request) => {
            seen.skipped = request;
            return textResult("skipped");
          }),
        ),
      ),
    );
    const part = value(
      await routing.stream(
        value(
          await routing.post("/part", async (request) => {
            seen.partial = request;
            const reader = value(await requests.bodyStream(request, 4n));
            const first = value(await reads.readMany(reader, 1n));
            const head = new TextDecoder().decode(copyBytes(first[0], origin));
            await reads.closeReader(reader);
            return textResult(head);
          }),
        ),
      ),
    );
    const table = value(await routing.make(array([skip, part])));
    const token = value(await servers.start(config, table));
    const ignored = await fetch("http://127.0.0.1:18371/skip", {
      method: "POST",
      body: "x".repeat(1000),
    });
    expect(ignored.status).toBe(200);
    expect(await ignored.text()).toBe("skipped");
    await expectRevoked(seen.skipped);
    const streamed = await fetch("http://127.0.0.1:18371/part", {
      method: "POST",
      body: "hello-world",
    });
    expect(streamed.status).toBe(200);
    expect(await streamed.text()).toBe("hell");
    await expectRevoked(seen.partial);
    const again = await fetch("http://127.0.0.1:18371/skip", {
      method: "POST",
      body: "reused",
    });
    expect(again.status).toBe(200);
    expect(await again.text()).toBe("skipped");
    expect(resourceStatus(token)).toMatchObject({ state: "open", leases: 0 });
    expect((await servers.stop(token)).kind).toBe("ok");
    return success(undefined);
  });
  expect(owned.cleanupFailed).toBe(false);
  expect(owned.completion.kind).toBe("ok");
});
test("handler faults revoke and release leases", async () => {
  const seen: { thrown?: unknown; failed?: unknown } = {};
  const owned = await runOwnedRoot(async () => {
    const config = value(await servers.makeConfig("127.0.0.1", 18372n, 1048576n, 5000n));
    const boom = value(
      await routing.get("/boom", async (request) => {
        seen.thrown = request;
        throw new Error("fault-injection");
      }),
    );
    const fail = value(
      await routing.get("/fail", async (request) => {
        seen.failed = request;
        return responses.makeBodyStatus(204n);
      }),
    );
    const table = value(await routing.make(array([boom, fail])));
    const token = value(await servers.start(config, table));
    const first = await fetch("http://127.0.0.1:18372/boom");
    expect(first.status).toBe(500);
    expect(await first.text()).toBe("Internal Server Error");
    await expectRevoked(seen.thrown);
    const second = await fetch("http://127.0.0.1:18372/fail");
    expect(second.status).toBe(500);
    expect(await second.text()).toBe("Internal Server Error");
    await expectRevoked(seen.failed);
    expect(resourceStatus(token)).toMatchObject({ state: "open", leases: 0 });
    expect((await servers.stop(token)).kind).toBe("ok");
    return success(undefined);
  });
  expect(owned.cleanupFailed).toBe(false);
  expect(owned.completion.kind).toBe("ok");
});
test("admitted child work keeps the token during drainage", async () => {
  const seen: { token?: unknown; header?: unknown; afterSettle?: unknown } = {};
  const settled = { handler: false };
  const owned = await runOwnedRoot(async () => {
    const config = value(await servers.makeConfig("127.0.0.1", 18373n, 1048576n, 5000n));
    const route = value(
      await routing.get("/x", async (request) => {
        seen.token = request;
        const group = launchOwned([
          {
            captures: [],
            run: async () => {
              await Bun.sleep(50);
              const headers = value(await requests.headers(request));
              seen.afterSettle = settled.handler;
              seen.header = headers
                .map((entry) => dataProperty(entry, "value"))
                .find((_, index) => dataProperty(headers[index], "name") === "x-child");
              return success(undefined);
            },
          },
        ]);
        group.publish([0]);
        settled.handler = true;
        return textResult("early");
      }),
    );
    const table = value(await routing.make(array([route])));
    const token = value(await servers.start(config, table));
    const response = await fetch("http://127.0.0.1:18373/x", {
      headers: { "x-child": "yes" },
    });
    expect(response.status).toBe(200);
    expect(await response.text()).toBe("early");
    // The child read after the handler settled but before revocation: the
    // token dies only when drainage completes, not when dispatch returns.
    expect(seen.afterSettle).toBe(true);
    expect(seen.header).toBe("yes");
    await expectRevoked(seen.token);
    expect(resourceStatus(token)).toMatchObject({ state: "open", leases: 0 });
    expect((await servers.stop(token)).kind).toBe("ok");
    return success(undefined);
  });
  expect(owned.cleanupFailed).toBe(false);
  expect(owned.completion.kind).toBe("ok");
});
test("upgraded requests revoke after the socket closes", async () => {
  const seen: { token?: unknown; done?: boolean } = {};
  const connectionOf = (completed: Completion<unknown>) => {
    const c = value(completed);
    expect(recordIdentity(c)).toBe(CONNECTION);
    return {
      session: dataProperty(c, "session"),
      events: dataProperty(c, "events"),
    };
  };
  const owned = await runOwnedRoot(async () => {
    const config = value(await servers.makeConfig("127.0.0.1", 18374n, 1048576n, 5000n));
    const route = value(
      await routing.get("/ws", async (request) => {
        seen.token = request;
        const conn = connectionOf(await ws.accept(request, "", 65536n, 64n, 1048576n));
        for (;;) {
          const batch = dataArray(value(await reads.readMany(conn.events, 1n)));
          if (batch.length === 0) break;
          for (const item of batch)
            if (recordIdentity(item) === TEXT)
              value(await ws.sendText(conn.session, "echo:" + dataProperty(item, "text")));
        }
        value(await reads.closeReader(conn.events));
        value(await ws.close(conn.session, 1000n, "server-bye"));
        seen.done = true;
        return success(undefined);
      }),
    );
    const table = value(await routing.make(array([route])));
    const token = value(await servers.start(config, table));
    const client = connectionOf(
      await ws.connect("ws://127.0.0.1:18374/ws", array([]), 65536n, 64n, 1048576n, 5000n, false),
    );
    expect(value(await ws.sendText(client.session, "a"))).toBe(1n);
    const back = dataArray(value(await reads.readMany(client.events, 1n)));
    expect(recordIdentity(back[0])).toBe(TEXT);
    expect(dataProperty(back[0], "text")).toBe("echo:a");
    expect(value(await ws.close(client.session, 1000n, "client-bye"))).toBe(undefined);
    const tail = dataArray(value(await reads.readMany(client.events, 2n)));
    expect(recordIdentity(tail[0])).toBe(CLOSE);
    expect((await reads.closeReader(client.events)).kind).toBe("ok");
    await untilDone(seen);
    // No HTTP response orders the upgrade path, so poll for the revoke that
    // the outer boundary runs after the handler and drainage settle.
    await untilRevoked(seen.token);
    await expectRevoked(seen.token);
    expect((await servers.stop(token)).kind).toBe("ok");
    return success(undefined);
  });
  expect(owned.cleanupFailed).toBe(false);
  expect(owned.completion.kind).toBe("ok");
});
