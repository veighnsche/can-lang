// NT-I05 websocket adapter contract: the 7 operations drive
// one adapter-held K21 service with owner admission first;
// the admitted grant doubles as the service owner string;
// facts map to nominal records with snake_case fields (ints
// as bigint, payloads as int arrays, nulls as options);
// revoked owners fail test::stale_handle without touching
// the service; closed-handle becomes test::closed_handle;
// wrong-owner becomes test::stale_handle; and every other
// K21 rejection maps to http_peer::peer_fault{kind, reason}
// with the service message preserved verbatim (I04
// failService precedent).
import { test, expect } from "bun:test";
import { createHash } from "node:crypto";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, domainFailureDiagnostics, type FailureShape } from "../domain.ts";
import type { Completion } from "../completion.ts";
import { dataArray, dataProperty, recordIdentity } from "../data.ts";
import { createTestOwner } from "../test-support/owner.ts";
import { createHttpPeerWs } from "../test-support/slices/i05/websocket.ts";

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
const declarations = catalogue.errors.filter(
  (error) =>
    error.name === "test::stale_handle" ||
    error.name === "test::closed_handle" ||
    error.name === "http_peer::peer_fault",
);
const shapeOf = (id: string, fields: readonly { name: string; type: string }[]): FailureShape => ({
  identity: identity("error", id),
  kind: "error",
  declaration: id,
  arguments: [],
  fields: fields.map((field) => ({ name: field.name, type: text.identity })),
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
  shapes: [text, ...errorShapes.values()],
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
  const owner = createTestOwner(domain, { invalidGrant: err("test::stale_handle") });
  const ws = createHttpPeerWs(
    domain,
    {
      staleHandle: err("test::stale_handle"),
      closedHandle: err("test::closed_handle"),
      peerFault: err("http_peer::peer_fault"),
      some: SOME,
      none: NONE,
      connectionHandle: typeId("http_peer::ws_connection_handle"),
      closeFacts: typeId("http_peer::ws_close_facts"),
      event: typeId("http_peer::ws_event"),
      connectionFacts: typeId("http_peer::ws_connection_facts"),
      sendReceipt: typeId("http_peer::ws_send_receipt"),
      deliverReceipt: typeId("http_peer::ws_deliver_receipt"),
      pollResult: typeId("http_peer::ws_poll_result"),
      closeReceipt: typeId("http_peer::ws_close_receipt"),
    },
    owner,
  );
  return { owner, ws };
}

const GRANT = "ng1-0123456789abcdef0123456789abcdef";
const GRANT2 = "ng1-fedcba9876543210fedcba9876543210";

async function ok<T>(completion: Promise<Completion<T>>): Promise<T> {
  const result = await completion;
  expect(result.kind).toBe("ok");
  if (result.kind !== "ok") throw Error("expected ok");
  return result.value;
}

async function failed(completion: Promise<Completion<never>>) {
  const result = await completion;
  expect(result.kind).toBe("domain");
  if (result.kind !== "domain") throw Error("expected domain failure");
  return domainFailureDiagnostics(result.value);
}

const field = (value: unknown, name: string): unknown => dataProperty(value, name);
const someValue = (value: unknown): unknown => {
  expect(recordIdentity(value)).toBe(SOME);
  return dataProperty(value, "value");
};
const expectNone = (value: unknown): void => {
  expect(recordIdentity(value)).toBe(NONE);
};

async function bootstrap() {
  const { owner, ws } = rig();
  const handle = await ok(owner.admitGrant(GRANT));
  const connected = await ok(ws.connect(handle, "ws-a"));
  return { owner, ws, handle, id: field(connected, "id") as string };
}

test("ws connect admits declared destinations and binds the grant", async () => {
  const { owner, ws, handle, id } = await bootstrap();
  expect(id).toBe("w1");
  const facts = await ok(ws.readConnectionFacts(handle, id));
  expect(field(facts, "owner")).not.toBe("");
  expect(field(facts, "destination")).toBe("ws-a");
  expect(field(facts, "state")).toBe("open");
  const foreign = await failed(ws.connect(handle, "ws-evil") as Promise<Completion<never>>);
  expect(foreign.declaration.name).toBe("http_peer::peer_fault");
  expect(field(foreign.payload, "kind")).toBe("foreign-destination");
  expect(field(foreign.payload, "reason")).toBe("destination is not declared");
  const other = await ok(owner.admitGrant(GRANT2));
  const stolen = await failed(ws.readConnectionFacts(other, id) as Promise<Completion<never>>);
  expect(stolen.declaration.name).toBe("test::stale_handle");
  const unknown = await failed(ws.readConnectionFacts(handle, "w9") as Promise<Completion<never>>);
  expect(unknown.declaration.name).toBe("http_peer::peer_fault");
  expect(field(unknown.payload, "kind")).toBe("invalid-request");
});

test("ws send frames on open connections with exact counts", async () => {
  const { ws, handle, id } = await bootstrap();
  for (const opcode of ["text", "binary", "ping", "pong"]) {
    const receipt = await ok(ws.send(handle, id, opcode, [1n, 2n, 3n]));
    expect(field(receipt, "opcode")).toBe(opcode);
    expect(field(receipt, "accepted")).toBe(3n);
  }
  const empty = await ok(ws.send(handle, id, "ping", []));
  expect(field(empty, "accepted")).toBe(0n);
  expect(field(empty, "sent_frames_total")).toBe(5n);
  expect(field(empty, "sent_bytes_total")).toBe(12n);
  const badOpcode = await failed(ws.send(handle, id, "close", [1n]) as Promise<Completion<never>>);
  expect(badOpcode.declaration.name).toBe("http_peer::peer_fault");
  expect(field(badOpcode.payload, "kind")).toBe("unsupported-capability");
  const tooBig = await failed(
    ws.send(
      handle,
      id,
      "binary",
      Array.from({ length: 17 }, () => 7n),
    ) as Promise<Completion<never>>,
  );
  expect(field(tooBig.payload, "kind")).toBe("resource-limit");
});

test("ws delivery and polls separate admitted from consumed", async () => {
  const { ws, handle, id } = await bootstrap();
  const quiet = await ok(ws.pollEvent(handle, id));
  expectNone(field(quiet, "event"));
  expect(field(quiet, "consumed_total")).toBe(0n);
  const receipt = await ok(ws.deliverEvent(handle, id, "text", [9n, 8n]));
  expect(field(receipt, "pending_events")).toBe(1n);
  expect(field(receipt, "pending_bytes")).toBe(2n);
  expect(field(receipt, "delivered_total")).toBe(1n);
  const polled = await ok(ws.pollEvent(handle, id));
  const event = someValue(field(polled, "event"));
  expect(field(event, "opcode")).toBe("text");
  expect([...dataArray(field(event, "payload"))]).toEqual([9n, 8n]);
  expect(field(polled, "consumed_total")).toBe(1n);
  expect(field(polled, "delivered_total")).toBe(1n);
  const facts = await ok(ws.readConnectionFacts(handle, id));
  expect(field(facts, "delivered_events")).toBe(1n);
  expect(field(facts, "consumed_events")).toBe(1n);
  expect(field(facts, "pending_events")).toBe(0n);
});

test("ws delivery backpressure rejects whole events", async () => {
  const { ws, handle, id } = await bootstrap();
  for (let n = 0; n < 8; n++) {
    await ok(ws.deliverEvent(handle, id, "binary", [1n]));
  }
  const full = await failed(
    ws.deliverEvent(handle, id, "binary", [1n]) as Promise<Completion<never>>,
  );
  expect(full.declaration.name).toBe("http_peer::peer_fault");
  expect(field(full.payload, "kind")).toBe("backpressure");
  const facts = await ok(ws.readConnectionFacts(handle, id));
  expect(field(facts, "pending_events")).toBe(8n);
  expect(field(facts, "delivered_events")).toBe(8n);
});

test("ws local and remote closes stay independent until terminal", async () => {
  const { ws, handle, id } = await bootstrap();
  const local = await ok(ws.close(handle, id, 1000n, "done"));
  expect(field(local, "origin")).toBe("local");
  expect(field(local, "close_code")).toBe(1000n);
  expect(field(local, "terminal")).toBe(false);
  const half = await ok(ws.readConnectionFacts(handle, id));
  expect(field(half, "state")).toBe("local-closed");
  expect(field(someValue(field(half, "local_close")), "reason")).toBe("done");
  expectNone(field(half, "remote_close"));
  const sendHalf = await failed(ws.send(handle, id, "text", [1n]) as Promise<Completion<never>>);
  expect(sendHalf.declaration.name).toBe("http_peer::peer_fault");
  expect(field(sendHalf.payload, "kind")).toBe("invalid-state");
  const remote = await ok(ws.deliverRemoteClose(handle, id, 1001n, "away"));
  expect(field(remote, "origin")).toBe("remote");
  expect(field(remote, "terminal")).toBe(true);
  const shut = await ok(ws.readConnectionFacts(handle, id));
  expect(field(shut, "state")).toBe("closed");
  const again = await failed(ws.close(handle, id, 1000n, "done") as Promise<Completion<never>>);
  expect(again.declaration.name).toBe("test::closed_handle");
  const polled = await failed(ws.pollEvent(handle, id) as Promise<Completion<never>>);
  expect(polled.declaration.name).toBe("test::closed_handle");
});

test("ws second remote closes and contradicting deliveries reject", async () => {
  const { ws, handle, id } = await bootstrap();
  await ok(ws.deliverRemoteClose(handle, id, 1000n, "bye"));
  const twice = await failed(
    ws.deliverRemoteClose(handle, id, 1000n, "bye") as Promise<Completion<never>>,
  );
  expect(twice.declaration.name).toBe("http_peer::peer_fault");
  expect(field(twice.payload, "kind")).toBe("invalid-state");
  const late = await failed(
    ws.deliverEvent(handle, id, "text", [1n]) as Promise<Completion<never>>,
  );
  expect(field(late.payload, "kind")).toBe("invalid-state");
  await ok(ws.close(handle, id, 1000n, "done"));
  const afterTerminal = await failed(
    ws.deliverRemoteClose(handle, id, 1000n, "bye") as Promise<Completion<never>>,
  );
  expect(afterTerminal.declaration.name).toBe("test::closed_handle");
});

test("ws completing close counts queued events as explicit drops", async () => {
  const { ws, handle, id } = await bootstrap();
  await ok(ws.deliverEvent(handle, id, "text", [1n, 2n]));
  await ok(ws.deliverEvent(handle, id, "ping", []));
  const local = await ok(ws.close(handle, id, 1000n, "done"));
  expect(field(local, "terminal")).toBe(false);
  expect(field(local, "dropped_events")).toBe(0n);
  const remote = await ok(ws.deliverRemoteClose(handle, id, 1000n, "bye"));
  expect(field(remote, "terminal")).toBe(true);
  expect(field(remote, "dropped_events")).toBe(2n);
  expect(field(remote, "dropped_bytes")).toBe(2n);
  const facts = await ok(ws.readConnectionFacts(handle, id));
  expect(field(facts, "dropped_events")).toBe(2n);
  expect(field(facts, "dropped_bytes")).toBe(2n);
  expect(field(facts, "pending_events")).toBe(0n);
});

test("ws close codes and reasons validate narrowly", async () => {
  const { ws, handle, id } = await bootstrap();
  for (const code of [999n, 5000n]) {
    const refused = await failed(ws.close(handle, id, code, "done") as Promise<Completion<never>>);
    expect(refused.declaration.name).toBe("http_peer::peer_fault");
    expect(field(refused.payload, "kind")).toBe("invalid-request");
  }
  const nonAscii = await failed(ws.close(handle, id, 1000n, "doné") as Promise<Completion<never>>);
  expect(field(nonAscii.payload, "kind")).toBe("invalid-request");
  const overlong = await failed(
    ws.close(handle, id, 1000n, "x".repeat(33)) as Promise<Completion<never>>,
  );
  expect(field(overlong.payload, "kind")).toBe("invalid-request");
  const emptyReason = await ok(ws.close(handle, id, 1000n, ""));
  expect(field(emptyReason, "origin")).toBe("local");
});

test("ws revoked owners fail stale without touching the service", async () => {
  const { owner, ws } = rig();
  const handle = await ok(owner.admitGrant(GRANT));
  const connected = await ok(ws.connect(handle, "ws-a"));
  await ok(owner.releaseGrant(handle));
  const stale = await failed(
    ws.readConnectionFacts(handle, field(connected, "id")) as Promise<Completion<never>>,
  );
  expect(stale.declaration.name).toBe("test::stale_handle");
  const staleSend = await failed(ws.send(handle, "w1", "text", []) as Promise<Completion<never>>);
  expect(staleSend.declaration.name).toBe("test::stale_handle");
});

test("ws malformed inputs throw before the service runs", async () => {
  const { ws, handle, id } = await bootstrap();
  await expect(ws.readConnectionFacts(handle, 7)).rejects.toThrow(
    "http_peer::ws_read_connection_facts connection",
  );
  await expect(ws.send(handle, id, "text", "nope")).rejects.toThrow("http_peer::ws_send payload");
  await expect(ws.send(handle, id, "text", [1n, "x"])).rejects.toThrow(
    "http_peer::ws_send payload",
  );
  await expect(ws.close(handle, id, "1000", "done")).rejects.toThrow(
    "http_peer::ws_close close_code",
  );
});
