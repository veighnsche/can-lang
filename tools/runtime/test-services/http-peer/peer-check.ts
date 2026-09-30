// K20 bounded self-check: typed in-memory peer operations only.
//
// Run with: bun tools/runtime/test-services/http-peer/peer-check.ts
// Local controls only. No sockets, no network, no timers, no files. Every
// loop below is bounded by a small constant.

import { strict as assert } from "node:assert";
import { checkFactsInert } from "../native-values/schema.ts";
import {
  checkPeerLimits,
  HttpPeerError,
  HTTP_PEER_SCHEMA_VERSION,
  PeerListenerService,
  type PeerLimits,
} from "./peer.ts";

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

function serviceLimits(): PeerLimits {
  return checkPeerLimits({
    max_listeners: 4,
    max_connections: 16,
    max_pending_accepts: 8,
    max_dials: 16,
    max_buffer_bytes: 64,
    max_chunk_bytes: 16,
    max_explicit_retries: 3,
  });
}

function tinyLimits(): PeerLimits {
  return checkPeerLimits({
    max_listeners: 1,
    max_connections: 2,
    max_pending_accepts: 1,
    max_dials: 4,
    max_buffer_bytes: 4,
    max_chunk_bytes: 4,
    max_explicit_retries: 1,
  });
}

// --- Admission ---------------------------------------------------------------

check("declare-and-listen", () => {
  const service = new PeerListenerService(["peer-a", "peer-b"], serviceLimits());
  const listener = service.openListener("owner-a", "peer-a");
  assert.equal(listener.kind, "listener");
  assert.equal(listener.destination, "peer-a");
  assert(Object.isFrozen(listener));
  // Foreign destinations reject before any state changes.
  expectPeerError(() => service.openListener("owner-a", "peer-evil"), "foreign-destination");
  expectPeerError(() => service.dial("owner-a", "peer-evil"), "foreign-destination");
  assert.equal(service.live_listener_count, 1);
  assert.equal(service.dial_count, 0);
  assert.equal(service.pending_accept_count, 0);
});

check("dial-accept-pair", () => {
  const service = new PeerListenerService(["peer-a"], serviceLimits());
  const listener = service.openListener("owner-a", "peer-a");
  const dialed = service.dial("owner-a", "peer-a");
  assert.equal(dialed.state, "completed");
  assert.equal(dialed.fail_reason, null);
  assert.equal(dialed.attempt_count, 1);
  assert.equal(dialed.retry_count, 0);
  assert(dialed.connection !== null);
  assert.equal(dialed.connection.side, "dialer");
  assert.equal(service.pending_accept_count, 1);
  const server = service.accept("owner-a", listener.id);
  assert.equal(server.side, "acceptor");
  const clientFacts = service.connectionFacts("owner-a", dialed.connection.id);
  const serverFacts = service.connectionFacts("owner-a", server.id);
  assert.equal(clientFacts.peer_id, server.id);
  assert.equal(serverFacts.peer_id, dialed.connection.id);
  assert.equal(service.listenerFacts("owner-a", listener.id).accepted_total, 1);
});

// --- No implicit retry --------------------------------------------------------

check("no-implicit-retry", () => {
  const service = new PeerListenerService(["peer-a"], serviceLimits());
  const failed = service.dial("owner-a", "peer-a");
  assert.equal(failed.state, "failed");
  assert.equal(failed.fail_reason, "no-listener");
  assert.equal(failed.connection, null);
  assert.equal(failed.retry_count, 0);
  // A listener arriving later never completes the failed dial on its own.
  service.openListener("owner-a", "peer-a");
  const still = service.dialFacts("owner-a", failed.dial.id);
  assert.equal(still.state, "failed");
  assert.equal(still.attempt_count, 1);
  assert.equal(service.live_endpoint_count, 0);
  // Only an explicit, counted retryDial re-attempts.
  const retried = service.retryDial("owner-a", failed.dial.id);
  assert.equal(retried.state, "completed");
  assert.equal(retried.attempt_count, 2);
  assert.equal(retried.retry_count, 1);
  assert(retried.connection !== null);
  expectPeerError(() => service.retryDial("owner-a", failed.dial.id), "invalid-state");
});

check("retry-counted-and-capped", () => {
  const service = new PeerListenerService(["peer-a"], tinyLimits());
  const failed = service.dial("owner-a", "peer-a");
  assert.equal(failed.state, "failed");
  const once = service.retryDial("owner-a", failed.dial.id);
  assert.equal(once.state, "failed");
  assert.equal(once.attempt_count, 2);
  assert.equal(once.retry_count, 1);
  // Budget exhausted: the rejected attempt is not recorded.
  expectPeerError(() => service.retryDial("owner-a", failed.dial.id), "resource-limit");
  assert.equal(service.dialFacts("owner-a", failed.dial.id).attempt_count, 2);
});

check("retries-disabled-by-bounds", () => {
  const limits = checkPeerLimits({ ...tinyLimits(), max_explicit_retries: 0 });
  const service = new PeerListenerService(["peer-a"], limits);
  const failed = service.dial("owner-a", "peer-a");
  expectPeerError(() => service.retryDial("owner-a", failed.dial.id), "resource-limit");
});

// --- Accepted vs consumed, EOF, partial state ---------------------------------

check("write-read-roundtrip", () => {
  const service = new PeerListenerService(["peer-a"], serviceLimits());
  const listener = service.openListener("owner-a", "peer-a");
  const dialed = service.dial("owner-a", "peer-a");
  assert(dialed.connection !== null);
  const server = service.accept("owner-a", listener.id);
  const receipt = service.write("owner-a", dialed.connection.id, [1, 2, 3, 4, 5]);
  assert.equal(receipt.accepted, 5);
  assert.equal(receipt.peer_buffered, 5);
  assert.equal(receipt.peer_accepted_total, 5);
  // Accepted into the buffer, nothing consumed yet.
  const before = service.connectionFacts("owner-a", server.id);
  assert.equal(before.accepted_bytes, 5);
  assert.equal(before.consumed_bytes, 0);
  // Short read: partial state explicit, remainder stays buffered.
  const first = service.read("owner-a", server.id, 2);
  assert.deepEqual([...first.bytes], [1, 2]);
  assert.equal(first.truncated, true);
  assert.equal(first.eof, false);
  assert.equal(first.consumed_total, 2);
  assert.equal(first.accepted_total, 5);
  const rest = service.read("owner-a", server.id, 16);
  assert.deepEqual([...rest.bytes], [3, 4, 5]);
  assert.equal(rest.truncated, false);
  assert.equal(rest.eof, false);
  assert(Object.isFrozen(rest.bytes));
});

check("eof-explicit", () => {
  const service = new PeerListenerService(["peer-a"], serviceLimits());
  const listener = service.openListener("owner-a", "peer-a");
  const dialed = service.dial("owner-a", "peer-a");
  assert(dialed.connection !== null);
  const server = service.accept("owner-a", listener.id);
  // An empty read on a live connection is NOT eof.
  const empty = service.read("owner-a", server.id, 16);
  assert.deepEqual([...empty.bytes], []);
  assert.equal(empty.eof, false);
  assert.equal(empty.truncated, false);
  service.write("owner-a", dialed.connection.id, [9, 8, 7]);
  service.halfClose("owner-a", dialed.connection.id, "write");
  // Peer write side closed but buffer not drained: still not eof.
  const partial = service.read("owner-a", server.id, 2);
  assert.equal(partial.eof, false);
  assert.equal(partial.truncated, true);
  const last = service.read("owner-a", server.id, 16);
  assert.deepEqual([...last.bytes], [7]);
  assert.equal(last.eof, true);
  assert.equal(last.truncated, false);
  // EOF is stable once drained.
  const again = service.read("owner-a", server.id, 16);
  assert.equal(again.eof, true);
});

check("half-close-rules", () => {
  const service = new PeerListenerService(["peer-a"], serviceLimits());
  const listener = service.openListener("owner-a", "peer-a");
  const dialed = service.dial("owner-a", "peer-a");
  const conn = dialed.connection;
  assert(conn !== null);
  const server = service.accept("owner-a", listener.id);
  expectPeerError(() => service.halfClose("owner-a", server.id, "both"), "unsupported-capability");
  service.write("owner-a", conn.id, [1, 2, 3]);
  const facts = service.halfClose("owner-a", conn.id, "write");
  assert.equal(facts.write_closed, true);
  assert.equal(facts.read_closed, false);
  expectPeerError(() => service.write("owner-a", conn.id, [4]), "invalid-state");
  expectPeerError(() => service.halfClose("owner-a", conn.id, "write"), "invalid-state");
  // Read half-close discards buffered bytes explicitly and counts them.
  const serverFacts = service.halfClose("owner-a", server.id, "read");
  assert.equal(serverFacts.read_closed, true);
  assert.equal(serverFacts.discarded_bytes, 3);
  assert.equal(serverFacts.buffered_bytes, 0);
  expectPeerError(() => service.read("owner-a", server.id, 16), "invalid-state");
  expectPeerError(() => service.write("owner-a", conn.id, [5]), "invalid-state");
  // Writes to a read-closed peer reject (dialer write side is closed here,
  // so exercise the peer-read-closed path from a fresh pair instead).
  const dialed2 = service.dial("owner-a", "peer-a");
  const conn2 = dialed2.connection;
  assert(conn2 !== null);
  const server2 = service.accept("owner-a", listener.id);
  service.halfClose("owner-a", server2.id, "read");
  expectPeerError(() => service.write("owner-a", conn2.id, [6]), "invalid-state");
});

check("backpressure", () => {
  const service = new PeerListenerService(["peer-a"], tinyLimits());
  const listener = service.openListener("owner-a", "peer-a");
  const dialed = service.dial("owner-a", "peer-a");
  const conn = dialed.connection;
  assert(conn !== null);
  const server = service.accept("owner-a", listener.id);
  service.write("owner-a", conn.id, [1, 2, 3, 4]);
  // Buffer full: the whole chunk rejects, nothing partially accepted.
  expectPeerError(() => service.write("owner-a", conn.id, [5]), "backpressure");
  const facts = service.connectionFacts("owner-a", server.id);
  assert.equal(facts.accepted_bytes, 4);
  assert.equal(facts.buffered_bytes, 4);
  assert.equal(facts.consumed_bytes, 0);
});

// --- Negative controls ---------------------------------------------------------

check("wrong-owner-and-unknown", () => {
  const service = new PeerListenerService(["peer-a"], serviceLimits());
  const listener = service.openListener("owner-a", "peer-a");
  const dialed = service.dial("owner-a", "peer-a");
  const conn = dialed.connection;
  assert(conn !== null);
  expectPeerError(() => service.accept("owner-b", listener.id), "wrong-owner");
  expectPeerError(() => service.write("owner-b", conn.id, [1]), "wrong-owner");
  expectPeerError(() => service.read("owner-b", conn.id, 4), "wrong-owner");
  expectPeerError(() => service.retryDial("owner-b", dialed.dial.id), "wrong-owner");
  expectPeerError(() => service.accept("owner-a", "l999"), "invalid-request");
  expectPeerError(() => service.read("owner-a", "c999", 4), "invalid-request");
  expectPeerError(() => service.retryDial("owner-a", "d999"), "invalid-request");
  // Malformed chunks and bounds reject before any effect.
  expectPeerError(() => service.write("owner-a", conn.id, []), "invalid-request");
  expectPeerError(() => service.write("owner-a", conn.id, [1, 256]), "invalid-request");
  expectPeerError(() => service.write("owner-a", conn.id, "bytes"), "invalid-request");
  expectPeerError(
    () =>
      service.write(
        "owner-a",
        conn.id,
        Array.from({ length: 17 }, () => 1),
      ),
    "resource-limit",
  );
  expectPeerError(() => service.read("owner-a", conn.id, 0), "invalid-request");
  expectPeerError(() => service.read("owner-a", conn.id, 17), "invalid-request");
  expectPeerError(() => service.openListener("", "peer-a"), "invalid-request");
});

check("accept-empty-and-stale", () => {
  const service = new PeerListenerService(["peer-a"], serviceLimits());
  const listener = service.openListener("owner-a", "peer-a");
  expectPeerError(() => service.accept("owner-a", listener.id), "invalid-state");
  const dialed = service.dial("owner-a", "peer-a");
  assert(dialed.connection !== null);
  service.close("owner-a", dialed.connection.id);
  // The only queued dialer closed first: accept skips the stale entry and
  // reports an empty queue; the skip stays observable in facts.
  expectPeerError(() => service.accept("owner-a", listener.id), "invalid-state");
  assert.equal(service.listenerFacts("owner-a", listener.id).stale_skipped, 1);
});

// --- Cleanup -------------------------------------------------------------------

check("cleanup-receipts", () => {
  const service = new PeerListenerService(["peer-a"], serviceLimits());
  const listener = service.openListener("owner-a", "peer-a");
  const dialed = service.dial("owner-a", "peer-a");
  const conn = dialed.connection;
  assert(conn !== null);
  const dropped = service.closeListener("owner-a", listener.id);
  assert.equal(dropped.pending_accepts_dropped, 1);
  assert.equal(service.pending_accept_count, 0);
  // The orphaned dialer observes peer-gone explicitly.
  const orphan = service.connectionFacts("owner-a", conn.id);
  assert.equal(orphan.peer_gone, true);
  expectPeerError(() => service.write("owner-a", conn.id, [1]), "invalid-state");
  const drained = service.read("owner-a", conn.id, 16);
  assert.equal(drained.eof, true);
  expectPeerError(() => service.accept("owner-a", listener.id), "closed-handle");
  expectPeerError(() => service.closeListener("owner-a", listener.id), "closed-handle");
  // Closing with unread bytes records them in the receipt.
  const listener2 = service.openListener("owner-a", "peer-a");
  const dialed2 = service.dial("owner-a", "peer-a");
  assert(dialed2.connection !== null);
  const server2 = service.accept("owner-a", listener2.id);
  service.write("owner-a", dialed2.connection.id, [7, 7]);
  const receipt = service.close("owner-a", server2.id);
  assert.equal(receipt.unread_bytes, 2);
  assert.equal(receipt.accepted_bytes, 2);
  assert.equal(receipt.consumed_bytes, 0);
  // Facts stay readable after close; operations reject as closed.
  assert.equal(service.connectionFacts("owner-a", server2.id).closed, true);
  expectPeerError(() => service.read("owner-a", server2.id, 4), "closed-handle");
  expectPeerError(() => service.close("owner-a", server2.id), "closed-handle");
  // The surviving peer observes peer-gone and drains to EOF.
  assert.equal(service.connectionFacts("owner-a", dialed2.connection.id).peer_gone, true);
});

check("caps-and-limits", () => {
  const service = new PeerListenerService(["peer-a", "peer-b"], tinyLimits());
  service.openListener("owner-a", "peer-a");
  expectPeerError(() => service.openListener("owner-a", "peer-b"), "resource-limit");
  const first = service.dial("owner-a", "peer-a");
  assert.equal(first.state, "completed");
  // Pending queue full: the dial records failure, nothing queues.
  const queued = service.dial("owner-a", "peer-a");
  assert.equal(queued.state, "failed");
  assert.equal(queued.fail_reason, "accept-queue-full");
  // Connection cap: two live endpoints max; accept would mint a third.
  const listener = service.listenerFacts("owner-a", "l1");
  assert.equal(listener.pending_accepts, 1);
  const server = service.accept("owner-a", "l1");
  assert.equal(server.side, "acceptor");
  assert.equal(service.live_endpoint_count, 2);
  expectPeerError(() => service.openListener("owner-a", "peer-a"), "resource-limit");
  // Limits validation rejects malformed bounds.
  expectPeerError(() => checkPeerLimits(null), "invalid-request");
  expectPeerError(() => checkPeerLimits({ ...tinyLimits(), max_listeners: 0 }), "invalid-request");
  expectPeerError(
    () => checkPeerLimits({ ...tinyLimits(), max_chunk_bytes: 8, max_buffer_bytes: 4 }),
    "invalid-request",
  );
  expectPeerError(
    () => checkPeerLimits({ ...tinyLimits(), max_explicit_retries: 17 }),
    "invalid-request",
  );
  // Declared-set validation rejects malformed sets.
  expectPeerError(() => new PeerListenerService([], tinyLimits()), "invalid-request");
  expectPeerError(() => new PeerListenerService(["a", "a"], tinyLimits()), "invalid-request");
});

check("facts-inert", () => {
  const service = new PeerListenerService(["peer-a"], serviceLimits());
  const listener = service.openListener("owner-a", "peer-a");
  const dialed = service.dial("owner-a", "peer-a");
  assert(dialed.connection !== null);
  const server = service.accept("owner-a", listener.id);
  service.write("owner-a", dialed.connection.id, [1]);
  service.read("owner-a", server.id, 16);
  checkFactsInert(service.listenerFacts("owner-a", listener.id), "listener facts");
  checkFactsInert(service.connectionFacts("owner-a", server.id), "connection facts");
  checkFactsInert(service.dialFacts("owner-a", dialed.dial.id), "dial facts");
  checkFactsInert(dialed, "dial result");
  checkFactsInert(service.close("owner-a", server.id), "close receipt");
});

console.log(
  JSON.stringify({
    kind: "can.http-peer-peer-check",
    schema_version: HTTP_PEER_SCHEMA_VERSION,
    checks: passed,
    count: passed.length,
  }),
);
