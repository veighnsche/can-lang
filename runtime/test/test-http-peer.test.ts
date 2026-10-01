// NT-I04 http_peer contract: owner-bound listener/dial/accept/read/write/
// half-close and request build/upload/deliver/read/reissue over the ported
// K20 doubles, with the exact finite failure vocabulary. Markers http_peer::
// and can.std.http_peer@1 keep the NT-I04 catalogue package referenced from
// runtime test evidence.
import { test, expect } from "bun:test";
import { createHash } from "node:crypto";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, domainFailureDiagnostics, type FailureShape } from "../domain.ts";
import type { Completion } from "../completion.ts";
import { dataProperty, record, recordIdentity } from "../data.ts";
import { createTestOwner } from "../test-support/owner.ts";
import { createHttpPeerSupport } from "../test-support/slices/i04/support.ts";

const identity = (kind: string, declaration: string) =>
  createHash("sha256")
    .update("can-concrete-type-v1\0" + JSON.stringify([kind, declaration]))
    .digest("hex");
const scalar = (name: string): FailureShape => ({
  identity: identity("primitive", name),
  kind: "primitive",
  declaration: name,
  arguments: [],
  fields: [],
  leaves: [],
  inputs: [],
  errors: [],
});
const text = scalar("str");
const integer = scalar("int");
const declarations = catalogue.errors.filter(
  (error) => error.name.startsWith("test::") || error.name.startsWith("http_peer::"),
);
const shapeOf = (id: string, fields: readonly { name: string; type: string }[]): FailureShape => ({
  identity: identity("error", id),
  kind: "error",
  declaration: id,
  arguments: [],
  fields: fields.map((field) => ({
    name: field.name,
    type: field.type === "int" ? integer.identity : text.identity,
  })),
  leaves: [],
  inputs: [],
  errors: [],
});
const errorShapes: Map<string, FailureShape> = new Map(
  declarations.map((error) => [error.name, shapeOf(error.identity, error.fields)]),
);
const domain = createDomainRuntime({
  declarations: declarations.map((error) => ({
    identity: error.identity,
    name: error.name,
    parameters: 0,
  })),
  shapes: [text, integer, ...errorShapes.values()],
});
const err = (name: string): string => {
  const shape = errorShapes.get(name);
  if (shape === undefined) throw Error(`no shape for ${name}`);
  return shape.identity;
};

const SOME = "test:some";
const NONE = "test:none";
const typeId = (name: string): string => `test:type:${name}`;

function rig() {
  const owner = createTestOwner(domain, { invalidGrant: err("test::invalid_grant") });
  const support = createHttpPeerSupport(
    domain,
    {
      staleHandle: err("test::stale_handle"),
      closedHandle: err("test::closed_handle"),
      peerFault: err("http_peer::peer_fault"),
      httpFault: err("http_peer::http_fault"),
      some: SOME,
      none: NONE,
      header: typeId("http::header"),
      listenerFacts: typeId("http_peer::listener_facts"),
      connectionFacts: typeId("http_peer::connection_facts"),
      dialFacts: typeId("http_peer::dial_facts"),
      dialResult: typeId("http_peer::dial_result"),
      writeReceipt: typeId("http_peer::write_receipt"),
      readResult: typeId("http_peer::read_result"),
      connectionCloseReceipt: typeId("http_peer::connection_close_receipt"),
      listenerCloseReceipt: typeId("http_peer::listener_close_receipt"),
      requestFacts: typeId("http_peer::request_facts"),
      redirect: typeId("http_peer::redirect"),
      responseFacts: typeId("http_peer::response_facts"),
      headerReceipt: typeId("http_peer::header_receipt"),
      bodyChunkReceipt: typeId("http_peer::body_chunk_receipt"),
      requestCloseReceipt: typeId("http_peer::request_close_receipt"),
    },
    owner,
  );
  return { owner, ...support };
}

const GRANT = "ng1-0123456789abcdef0123456789abcdef";
const GRANT2 = "ng1-fedcba9876543210fedcba9876543210";

async function failed(completion: Promise<Completion<never>>) {
  const result = await completion;
  expect(result.kind).toBe("domain");
  if (result.kind !== "domain") throw Error("expected domain failure");
  return domainFailureDiagnostics(result.value);
}

async function ok<T>(completion: Promise<Completion<T>>): Promise<T> {
  const result = await completion;
  expect(result.kind).toBe("ok");
  if (result.kind !== "ok") throw Error("expected ok");
  return result.value;
}

const field = (value: unknown, name: string): unknown => dataProperty(value, name);
const someValue = (value: unknown): unknown => {
  expect(recordIdentity(value)).toBe(SOME);
  return dataProperty(value, "value");
};
const expectNone = (value: unknown): void => {
  expect(recordIdentity(value)).toBe(NONE);
};

test("can.std.http_peer@1 open_listener mints opaque handles; foreign destinations fault", async () => {
  const { owner, peer } = rig();
  const handle = await ok(owner.admitGrant(GRANT));
  const listener = await ok(peer.openListener(handle, "peer-a"));
  expect(typeof listener).toBe("object");
  expect(Object.keys(listener as object)).toEqual([]);
  const details = await failed(
    peer.openListener(handle, "peer-evil") as Promise<Completion<never>>,
  );
  expect(details.declaration.name).toBe("http_peer::peer_fault");
  expect(details.payload).toMatchObject({ kind: "foreign-destination" });
});

test("http_peer:: dial/accept/write/read/half_close/close flow binds facts", async () => {
  const { owner, peer } = rig();
  const handle = await ok(owner.admitGrant(GRANT));
  const listener = await ok(peer.openListener(handle, "peer-a"));
  const dialed = await ok(peer.dialPeer(handle, "peer-a"));
  expect(field(dialed, "state")).toBe("completed");
  expectNone(field(dialed, "fail_reason"));
  const dialHandle = field(dialed, "dial");
  const conn = someValue(field(dialed, "connection"));
  const dialFacts = await ok(peer.readDialFacts(handle, dialHandle));
  expect(field(dialFacts, "destination")).toBe("peer-a");
  const connHandle = conn;
  const server = await ok(peer.accept(handle, listener));
  const receipt = await ok(peer.write(handle, connHandle, [1n, 2n, 3n]));
  expect(field(receipt, "accepted")).toBe(3n);
  const read = await ok(peer.read(handle, server, 16n));
  expect(field(read, "bytes")).toEqual([1n, 2n, 3n]);
  expect(field(read, "eof")).toBe(false);
  const facts = await ok(peer.halfClose(handle, connHandle, "write"));
  expect(field(facts, "write_closed")).toBe(true);
  const drained = await ok(peer.read(handle, server, 16n));
  expect(field(drained, "eof")).toBe(true);
  const closed = await ok(peer.closeConnection(handle, connHandle));
  expect(field(closed, "accepted_bytes")).toBe(0n);
  expect(field(closed, "consumed_bytes")).toBe(0n);
});

test("http_peer:: failed dials stay failed until an explicit retry_dial", async () => {
  const { owner, peer } = rig();
  const handle = await ok(owner.admitGrant(GRANT));
  const failed = await ok(peer.dialPeer(handle, "peer-a"));
  expect(field(failed, "state")).toBe("failed");
  expect(someValue(field(failed, "fail_reason"))).toBe("no-listener");
  expectNone(field(failed, "connection"));
  const dialHandle = field(failed, "dial");
  // A listener arriving later never completes the failed dial on its own.
  await ok(peer.openListener(handle, "peer-a"));
  const still = await ok(peer.readDialFacts(handle, dialHandle));
  expect(field(still, "state")).toBe("failed");
  expect(field(still, "attempt_count")).toBe(1n);
  const retried = await ok(peer.retryDial(handle, dialHandle));
  expect(field(retried, "state")).toBe("completed");
  expect(field(retried, "attempt_count")).toBe(2n);
  expect(field(retried, "retry_count")).toBe(1n);
  // The retry reuses the presented dial handle: same table entry.
  expect(retried !== null && field(retried, "dial")).toBe(dialHandle);
});

test("http_peer:: backpressure and closed handles fault closed", async () => {
  const { owner, peer } = rig();
  const handle = await ok(owner.admitGrant(GRANT));
  const listener = await ok(peer.openListener(handle, "peer-a"));
  const dialed = await ok(peer.dialPeer(handle, "peer-a"));
  const connHandle = someValue(field(dialed, "connection"));
  await ok(peer.accept(handle, listener));
  // Fill the 64-byte peer buffer, then overfill it: the whole chunk rejects.
  const chunk = Array.from({ length: 16 }, () => 7n);
  for (let i = 0; i < 4; i++) {
    await ok(peer.write(handle, connHandle, chunk));
  }
  const pressured = await failed(
    peer.write(handle, connHandle, [9n]) as Promise<Completion<never>>,
  );
  expect(pressured.declaration.name).toBe("http_peer::peer_fault");
  expect(pressured.payload).toMatchObject({ kind: "backpressure" });
  await ok(peer.closeConnection(handle, connHandle));
  const closed = await failed(peer.write(handle, connHandle, [1n]) as Promise<Completion<never>>);
  expect(closed.declaration.name).toBe("test::closed_handle");
});

test("http_peer:: cross-owner and unknown handles are stale; dead owners refuse", async () => {
  const { owner, peer } = rig();
  const first = await ok(owner.admitGrant(GRANT));
  const second = await ok(owner.admitGrant(GRANT2));
  const listener = await ok(peer.openListener(first, "peer-a"));
  const foreign = await failed(
    peer.readListenerFacts(second, listener) as Promise<Completion<never>>,
  );
  expect(foreign.declaration.name).toBe("test::stale_handle");
  await ok(owner.releaseGrant(first));
  const stale = await failed(peer.readListenerFacts(first, listener) as Promise<Completion<never>>);
  expect(stale.declaration.name).toBe("test::stale_handle");
  await expect(peer.openListener("not-a-handle", "peer-a")).rejects.toThrow(TypeError);
});

test("can.std.http_peer@1 request/upload/deliver/read/close flow binds facts", async () => {
  const { owner, http } = rig();
  const handle = await ok(owner.admitGrant(GRANT));
  const request = await ok(http.openRequest(handle, "origin-a", "POST", "/upload", "1.1"));
  const added = await ok(http.addHeader(handle, request, "content-type", "text/plain"));
  expect(field(added, "header_count")).toBe(1n);
  const sent = await ok(http.sendBodyChunk(handle, request, [10n, 11n]));
  expect(field(sent, "accepted")).toBe(2n);
  const ended = await ok(http.endUpload(handle, request));
  expect(field(ended, "upload_complete")).toBe(true);
  expect(field(ended, "response_delivered")).toBe(false);
  const delivered = await ok(
    http.deliverResponse(
      handle,
      request,
      200n,
      [
        record(typeId("http::header"), [
          ["name", "x-trace"],
          ["value", "t1"],
        ]),
      ],
      [20n],
    ),
  );
  expect(field(delivered, "status")).toBe(200n);
  const chunk = await ok(http.readBodyChunk(handle, request, 16n));
  expect(field(chunk, "bytes")).toEqual([20n]);
  expect(field(chunk, "eof")).toBe(true);
  const response = someValue(await ok(http.readResponseFacts(handle, request)));
  expect(field(response, "body_consumed")).toBe(1n);
  const closed = await ok(http.closeRequest(handle, request));
  expect(field(closed, "upload_bytes")).toBe(2n);
  expect(field(closed, "response_body_unread")).toBe(0n);
});

test("http_peer:: closed HTTP vocabularies fault unsupported-capability", async () => {
  const { owner, http } = rig();
  const handle = await ok(owner.admitGrant(GRANT));
  for (const [method, version] of [
    ["PUT", "1.1"],
    ["GET", "2"],
  ] as const) {
    const details = await failed(
      http.openRequest(handle, "origin-a", method, "/x", version) as Promise<Completion<never>>,
    );
    expect(details.declaration.name).toBe("http_peer::http_fault");
    expect(details.payload).toMatchObject({ kind: "unsupported-capability" });
  }
  const request = await ok(http.openRequest(handle, "origin-a", "GET", "/x", "1.1"));
  const header = await failed(
    http.addHeader(handle, request, "x-evil", "v") as Promise<Completion<never>>,
  );
  expect(header.payload).toMatchObject({ kind: "unsupported-capability" });
  await ok(http.endUpload(handle, request));
  const status = await failed(
    http.deliverResponse(handle, request, 418n, [], []) as Promise<Completion<never>>,
  );
  expect(status.payload).toMatchObject({ kind: "unsupported-capability" });
});

test("http_peer:: reissue chains explicitly; response_facts is none until delivery", async () => {
  const { owner, http } = rig();
  const handle = await ok(owner.admitGrant(GRANT));
  const first = await ok(http.openRequest(handle, "origin-a", "GET", "/old", "1.1"));
  expectNone(await ok(http.readResponseFacts(handle, first)));
  await ok(http.endUpload(handle, first));
  const second = await ok(http.reissue(handle, first));
  const facts = await ok(http.readRequestFacts(handle, second));
  expect(field(facts, "reissue_count")).toBe(1n);
  expect(typeof someValue(field(facts, "supersedes"))).toBe("string");
  expect(field(facts, "header_count")).toBe(0n);
});
