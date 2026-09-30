// K21 bounded self-check: typed in-memory WebSocket observations only.
//
// Run with: bun tools/runtime/test-services/http-peer/websocket-check.ts
// Local controls only. No sockets, no network, no timers, no files. Every
// loop below is bounded by a small constant.

import { strict as assert } from "node:assert";
import { checkFactsInert } from "../native-values/schema.ts";
import { HttpPeerError, HTTP_PEER_SCHEMA_VERSION } from "./peer.ts";
import { checkWsLimits, WebSocketPeerService, type WsLimits } from "./websocket.ts";

const passed: string[] = [];
function check(name: string, body: () => void): void {
  body();
  passed.push(name);
}

function expectPeerError(body: () => unknown, kind: string): void {
  try {
    body();
  } catch (error) {
    if (!(error instanceof HttpPeerError)) {
      assert.fail("expected a HttpPeerError");
    }
    assert.equal(error.kind, kind);
    return;
  }
  assert.fail(`expected rejection with kind ${kind}`);
}

function serviceLimits(): WsLimits {
  return checkWsLimits({
    max_connections: 8,
    max_pending_events: 8,
    max_payload_bytes: 16,
    max_buffer_bytes: 64,
    max_close_reason_bytes: 32,
  });
}

function tinyLimits(): WsLimits {
  return checkWsLimits({
    max_connections: 2,
    max_pending_events: 2,
    max_payload_bytes: 4,
    max_buffer_bytes: 6,
    max_close_reason_bytes: 8,
  });
}

// --- Connect + admission --------------------------------------------------------

check("connect-and-admission", () => {
  const service = new WebSocketPeerService(["ws-a", "ws-b"], serviceLimits());
  const handle = service.connect("owner-a", "ws-a");
  assert.equal(handle.kind, "websocket");
  assert.equal(handle.destination, "ws-a");
  assert(Object.isFrozen(handle));
  const facts = service.connectionFacts("owner-a", handle.id);
  assert.equal(facts.state, "open");
  assert.equal(facts.local_close, null);
  assert.equal(facts.remote_close, null);
  // Foreign destinations reject before any state changes.
  expectPeerError(() => service.connect("owner-a", "ws-evil"), "foreign-destination");
  assert.equal(service.live_connection_count, 1);
  assert.equal(service.pending_event_count, 0);
});

// --- Send with opcode + payload --------------------------------------------------

check("send-opcode-payload", () => {
  const service = new WebSocketPeerService(["ws-a"], serviceLimits());
  const handle = service.connect("owner-a", "ws-a");
  for (const opcode of ["text", "binary", "ping", "pong"] as const) {
    const receipt = service.send("owner-a", handle.id, opcode, [1, 2, 3]);
    assert.equal(receipt.opcode, opcode);
    assert.equal(receipt.accepted, 3);
  }
  // Empty payloads are legal frames (control frames, empty messages).
  const empty = service.send("owner-a", handle.id, "ping", []);
  assert.equal(empty.accepted, 0);
  const facts = service.connectionFacts("owner-a", handle.id);
  assert.equal(facts.sent_frames, 5);
  assert.equal(facts.sent_bytes, 12);
  assert.equal(empty.sent_frames_total, 5);
  assert.equal(empty.sent_bytes_total, 12);
  // Closed opcode vocabulary; malformed payloads reject without effect.
  expectPeerError(() => service.send("owner-a", handle.id, "close", [1]), "unsupported-capability");
  expectPeerError(() => service.send("owner-a", handle.id, "text", "nope"), "invalid-request");
  expectPeerError(() => service.send("owner-a", handle.id, "text", [1, 300]), "invalid-request");
  expectPeerError(
    () =>
      service.send(
        "owner-a",
        handle.id,
        "binary",
        Array.from({ length: 17 }, () => 1),
      ),
    "resource-limit",
  );
  assert.equal(service.connectionFacts("owner-a", handle.id).sent_frames, 5);
});

// --- Explicit event delivery + poll ----------------------------------------------

check("event-deliver-poll", () => {
  const service = new WebSocketPeerService(["ws-a"], serviceLimits());
  const handle = service.connect("owner-a", "ws-a");
  // No spontaneous events: an empty poll on a live connection is null.
  const none = service.pollEvent("owner-a", handle.id);
  assert.equal(none.event, null);
  assert.equal(none.pending_events, 0);
  service.deliverEvent("owner-a", handle.id, "text", [9, 8]);
  service.deliverEvent("owner-a", handle.id, "binary", []);
  service.deliverEvent("owner-a", handle.id, "pong", [7]);
  // Accepted by delivery, nothing consumed yet.
  const before = service.connectionFacts("owner-a", handle.id);
  assert.equal(before.delivered_events, 3);
  assert.equal(before.consumed_events, 0);
  assert.equal(before.pending_events, 3);
  assert.equal(before.pending_bytes, 3);
  // FIFO order with opcode + payload preserved.
  const first = service.pollEvent("owner-a", handle.id);
  assert(first.event !== null);
  assert.equal(first.event.opcode, "text");
  assert.deepEqual([...first.event.payload], [9, 8]);
  assert(Object.isFrozen(first.event.payload));
  assert.equal(first.consumed_total, 1);
  assert.equal(first.delivered_total, 3);
  const second = service.pollEvent("owner-a", handle.id);
  assert(second.event !== null);
  assert.equal(second.event.opcode, "binary");
  assert.deepEqual([...second.event.payload], []);
  const third = service.pollEvent("owner-a", handle.id);
  assert(third.event !== null);
  assert.equal(third.event.opcode, "pong");
  assert.equal(third.pending_events, 0);
  const after = service.connectionFacts("owner-a", handle.id);
  assert.equal(after.consumed_events, 3);
  assert.equal(after.delivered_events, 3);
  // Bad opcodes and payloads reject without queueing anything.
  expectPeerError(
    () => service.deliverEvent("owner-a", handle.id, "close", [1]),
    "unsupported-capability",
  );
  expectPeerError(
    () => service.deliverEvent("owner-a", handle.id, "text", [1, 999]),
    "invalid-request",
  );
  assert.equal(service.connectionFacts("owner-a", handle.id).delivered_events, 3);
});

// --- Distinct local/remote close facts, both orders -------------------------------

check("close-local-first", () => {
  const service = new WebSocketPeerService(["ws-a"], serviceLimits());
  const handle = service.connect("owner-a", "ws-a");
  const local = service.close("owner-a", handle.id, 1000, "done");
  assert.equal(local.origin, "local");
  assert.equal(local.close_code, 1000);
  assert.equal(local.terminal, false);
  // Local close records only the local fact; the remote fact stays absent.
  const half = service.connectionFacts("owner-a", handle.id);
  assert.equal(half.state, "local-closed");
  assert(half.local_close !== null);
  assert.equal(half.local_close.origin, "local");
  assert.equal(half.local_close.close_code, 1000);
  assert.equal(half.local_close.reason, "done");
  assert.equal(half.remote_close, null);
  // Half-closed: sends and duplicate local closes reject, but the remote
  // side may still deliver events and its own close.
  expectPeerError(() => service.send("owner-a", handle.id, "text", [1]), "invalid-state");
  expectPeerError(() => service.close("owner-a", handle.id, 1000, "again"), "invalid-state");
  service.deliverEvent("owner-a", handle.id, "text", [5]);
  const polled = service.pollEvent("owner-a", handle.id);
  assert(polled.event !== null);
  assert.deepEqual([...polled.event.payload], [5]);
  const remote = service.deliverRemoteClose("owner-a", handle.id, 1000, "bye");
  assert.equal(remote.origin, "remote");
  assert.equal(remote.terminal, true);
  assert.equal(remote.dropped_events, 0);
  const full = service.connectionFacts("owner-a", handle.id);
  assert.equal(full.state, "closed");
  assert(full.local_close !== null && full.remote_close !== null);
  assert.equal(full.remote_close.origin, "remote");
  assert.equal(full.remote_close.reason, "bye");
  assert.equal(service.live_connection_count, 0);
});

check("close-remote-first", () => {
  const service = new WebSocketPeerService(["ws-a"], serviceLimits());
  const handle = service.connect("owner-a", "ws-a");
  service.deliverEvent("owner-a", handle.id, "binary", [1, 2]);
  const remote = service.deliverRemoteClose("owner-a", handle.id, 1001, "away");
  assert.equal(remote.terminal, false);
  const half = service.connectionFacts("owner-a", handle.id);
  assert.equal(half.state, "remote-closed");
  assert.equal(half.local_close, null);
  assert(half.remote_close !== null);
  assert.equal(half.remote_close.close_code, 1001);
  // After the remote close, further remote events contradict the observed
  // close and reject; queued events still drain via poll.
  expectPeerError(() => service.deliverEvent("owner-a", handle.id, "text", [3]), "invalid-state");
  expectPeerError(() => service.send("owner-a", handle.id, "text", [3]), "invalid-state");
  const drained = service.pollEvent("owner-a", handle.id);
  assert(drained.event !== null);
  assert.equal(drained.event.opcode, "binary");
  const local = service.close("owner-a", handle.id, 1000, "ok");
  assert.equal(local.terminal, true);
  assert.equal(service.connectionFacts("owner-a", handle.id).state, "closed");
});

// --- Dropped events detectable -----------------------------------------------------

check("dropped-events", () => {
  const service = new WebSocketPeerService(["ws-a"], serviceLimits());
  const handle = service.connect("owner-a", "ws-a");
  service.deliverEvent("owner-a", handle.id, "text", [1, 2, 3]);
  service.deliverEvent("owner-a", handle.id, "binary", [4, 5]);
  service.close("owner-a", handle.id, 1000, "me-first");
  // Two events still queued when the handshake completes: counted drops,
  // never silent loss and never pollable afterwards.
  const completing = service.deliverRemoteClose("owner-a", handle.id, 1000, "you-too");
  assert.equal(completing.terminal, true);
  assert.equal(completing.dropped_events, 2);
  assert.equal(completing.dropped_bytes, 5);
  const facts = service.connectionFacts("owner-a", handle.id);
  assert.equal(facts.dropped_events, 2);
  assert.equal(facts.dropped_bytes, 5);
  assert.equal(facts.pending_events, 0);
  assert.equal(facts.delivered_events, 2);
  assert.equal(facts.consumed_events, 0);
  expectPeerError(() => service.pollEvent("owner-a", handle.id), "closed-handle");
  // Seeded negative: dropping nothing reports exact zeros.
  const handle2 = service.connect("owner-a", "ws-a");
  service.close("owner-a", handle2.id, 1000, "");
  const clean = service.deliverRemoteClose("owner-a", handle2.id, 1000, "");
  assert.equal(clean.dropped_events, 0);
  assert.equal(clean.dropped_bytes, 0);
});

// --- Forged remote-close detection --------------------------------------------------

check("forged-remote-close", () => {
  const service = new WebSocketPeerService(["ws-a"], serviceLimits());
  const handle = service.connect("owner-a", "ws-a");
  // Missing evidence: with no delivery, no remote close can be claimed.
  assert.equal(service.connectionFacts("owner-a", handle.id).remote_close, null);
  service.deliverRemoteClose("owner-a", handle.id, 1000, "real");
  // A second remote close contradicts the single observed one.
  expectPeerError(
    () => service.deliverRemoteClose("owner-a", handle.id, 1000, "forged"),
    "invalid-state",
  );
  const facts = service.connectionFacts("owner-a", handle.id);
  assert(facts.remote_close !== null);
  assert.equal(facts.remote_close.reason, "real");
  service.close("owner-a", handle.id, 1000, "local");
  // After the terminal state, any further close injection rejects.
  expectPeerError(
    () => service.deliverRemoteClose("owner-a", handle.id, 1000, "late"),
    "closed-handle",
  );
  expectPeerError(() => service.close("owner-a", handle.id, 1000, "late"), "closed-handle");
  // Terminal operations reject as closed; facts stay readable.
  expectPeerError(() => service.send("owner-a", handle.id, "text", [1]), "closed-handle");
  expectPeerError(() => service.deliverEvent("owner-a", handle.id, "text", [1]), "closed-handle");
  assert.equal(service.connectionFacts("owner-a", handle.id).state, "closed");
});

// --- Bounded pending work ------------------------------------------------------------

check("bounded-pending-work", () => {
  const service = new WebSocketPeerService(["ws-a"], tinyLimits());
  const handle = service.connect("owner-a", "ws-a");
  service.deliverEvent("owner-a", handle.id, "text", [1, 2]);
  service.deliverEvent("owner-a", handle.id, "text", [3, 4]);
  // Queue full (2 events): the whole event rejects, nothing partially queued.
  expectPeerError(() => service.deliverEvent("owner-a", handle.id, "text", [5]), "backpressure");
  assert.equal(service.connectionFacts("owner-a", handle.id).pending_events, 2);
  assert.equal(service.pending_event_count, 2);
  // Byte cap (6 bytes) binds on a second connection: 4 queued + 3 more
  // exceeds it while the 4-byte payload cap still admits the frame.
  const handle2 = service.connect("owner-a", "ws-a");
  service.deliverEvent("owner-a", handle2.id, "binary", [1, 2, 3, 4]);
  expectPeerError(
    () => service.deliverEvent("owner-a", handle2.id, "binary", [5, 6, 7]),
    "backpressure",
  );
  const facts = service.connectionFacts("owner-a", handle2.id);
  assert.equal(facts.pending_events, 1);
  assert.equal(facts.pending_bytes, 4);
  assert.equal(facts.delivered_events, 1);
  // Payload cap rejects oversize frames on both paths.
  expectPeerError(
    () => service.deliverEvent("owner-a", handle2.id, "text", [1, 2, 3, 4, 5]),
    "resource-limit",
  );
  expectPeerError(
    () => service.send("owner-a", handle2.id, "text", [1, 2, 3, 4, 5]),
    "resource-limit",
  );
  // Connection cap: two live max.
  expectPeerError(() => service.connect("owner-a", "ws-a"), "resource-limit");
});

// --- Negative controls ----------------------------------------------------------------

check("wrong-owner-and-unknown", () => {
  const service = new WebSocketPeerService(["ws-a"], serviceLimits());
  const handle = service.connect("owner-a", "ws-a");
  expectPeerError(() => service.send("owner-b", handle.id, "text", [1]), "wrong-owner");
  expectPeerError(() => service.deliverEvent("owner-b", handle.id, "text", [1]), "wrong-owner");
  expectPeerError(() => service.pollEvent("owner-b", handle.id), "wrong-owner");
  expectPeerError(() => service.close("owner-b", handle.id, 1000, "x"), "wrong-owner");
  expectPeerError(() => service.deliverRemoteClose("owner-b", handle.id, 1000, "x"), "wrong-owner");
  expectPeerError(() => service.connectionFacts("owner-b", handle.id), "wrong-owner");
  expectPeerError(() => service.pollEvent("owner-a", "w999"), "invalid-request");
  expectPeerError(() => service.send("owner-a", "w999", "text", [1]), "invalid-request");
  expectPeerError(() => service.close("owner-a", "w999", 1000, "x"), "invalid-request");
  // Malformed close codes and reasons reject before any effect.
  expectPeerError(() => service.close("owner-a", handle.id, 999, "x"), "invalid-request");
  expectPeerError(() => service.close("owner-a", handle.id, 5000, "x"), "invalid-request");
  expectPeerError(() => service.close("owner-a", handle.id, 1000.5, "x"), "invalid-request");
  expectPeerError(() => service.close("owner-a", handle.id, 1000, 7), "invalid-request");
  expectPeerError(
    () => service.close("owner-a", handle.id, 1000, "bad\nnewline"),
    "invalid-request",
  );
  expectPeerError(
    () => service.deliverRemoteClose("owner-a", handle.id, 42, "x"),
    "invalid-request",
  );
  expectPeerError(
    () =>
      service.deliverRemoteClose(
        "owner-a",
        handle.id,
        1000,
        "this reason is far too long for the cap",
      ),
    "invalid-request",
  );
  assert.equal(service.connectionFacts("owner-a", handle.id).state, "open");
  expectPeerError(() => service.connect("", "ws-a"), "invalid-request");
});

check("caps-and-limits", () => {
  // Limits validation rejects malformed bounds.
  expectPeerError(() => checkWsLimits(null), "invalid-request");
  expectPeerError(() => checkWsLimits({ ...tinyLimits(), max_connections: 0 }), "invalid-request");
  expectPeerError(
    () => checkWsLimits({ ...tinyLimits(), max_payload_bytes: 8, max_buffer_bytes: 6 }),
    "invalid-request",
  );
  expectPeerError(
    () => checkWsLimits({ ...tinyLimits(), max_pending_events: 257 }),
    "invalid-request",
  );
  expectPeerError(
    () => checkWsLimits({ ...tinyLimits(), max_close_reason_bytes: -1 }),
    "invalid-request",
  );
  // Zero reason cap admits empty reasons only.
  const strict = new WebSocketPeerService(
    ["ws-a"],
    checkWsLimits({ ...tinyLimits(), max_close_reason_bytes: 0 }),
  );
  const handle = strict.connect("owner-a", "ws-a");
  strict.close("owner-a", handle.id, 1000, "");
  expectPeerError(
    () => strict.deliverRemoteClose("owner-a", handle.id, 1000, "x"),
    "invalid-request",
  );
  // Declared-set validation rejects malformed sets.
  expectPeerError(() => new WebSocketPeerService([], tinyLimits()), "invalid-request");
  expectPeerError(() => new WebSocketPeerService(["a", "a"], tinyLimits()), "invalid-request");
});

check("facts-inert", () => {
  const service = new WebSocketPeerService(["ws-a"], serviceLimits());
  const handle = service.connect("owner-a", "ws-a");
  service.send("owner-a", handle.id, "text", [1]);
  service.deliverEvent("owner-a", handle.id, "binary", [2]);
  service.pollEvent("owner-a", handle.id);
  checkFactsInert(service.connectionFacts("owner-a", handle.id), "connection facts");
  checkFactsInert(service.send("owner-a", handle.id, "ping", []), "send receipt");
  checkFactsInert(service.deliverEvent("owner-a", handle.id, "pong", [3]), "deliver receipt");
  service.close("owner-a", handle.id, 1000, "done");
  checkFactsInert(service.deliverRemoteClose("owner-a", handle.id, 1000, "bye"), "close receipt");
  checkFactsInert(service.connectionFacts("owner-a", handle.id), "terminal facts");
});

console.log(
  JSON.stringify({
    kind: "can.websocket-check",
    schema_version: HTTP_PEER_SCHEMA_VERSION,
    checks: passed,
    count: passed.length,
  }),
);
