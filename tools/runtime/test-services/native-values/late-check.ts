// K06 bounded self-check: late native occurrence and lease facts only.
//
// Run with: bun tools/runtime/test-services/native-values/late-check.ts
// Local controls only. No services, transports, timers, or live runtimes.
// Every loop below is bounded by a small constant.

import { strict as assert } from "node:assert";
import { checkFactsInert, NativeSchemaError } from "./schema.ts";
import {
  LATE_IDENTITIES,
  LateOccurrenceService,
  MAX_LATE_EVENTS,
  MAX_LATE_LEASES,
} from "./late.ts";

const passed: string[] = [];
function check(name: string, body: () => void): void {
  body();
  passed.push(name);
}

function expectReject(body: () => unknown, kind: string): void {
  try {
    body();
  } catch (error) {
    if (!(error instanceof NativeSchemaError)) {
      assert.fail("expected a NativeSchemaError");
    }
    assert.equal(error.outcome, "rejected");
    assert.equal(error.kind, kind);
    return;
  }
  assert.fail(`expected rejection with kind ${kind}`);
}

function cleanService(): LateOccurrenceService {
  const late = new LateOccurrenceService();
  late.select("late.alpha");
  late.enroll("worker");
  late.enroll("observer");
  return late;
}

// --- Positive control: clean two-participant run ------------------------------

check("positive-clean-two-participant-run", () => {
  const late = cleanService();

  const early = late.emit("worker", "use", 1);
  assert.equal(early.late, false);
  assert.deepEqual(early.dropped, []);
  checkFactsInert(early, "early event facts");

  const gateWorker = late.gate("worker");
  assert.equal(gateWorker.gate_seq, 2);
  assert.equal(gateWorker.joined, false);
  const gateObserver = late.gate("observer");
  assert.equal(gateObserver.gate_seq, 1);

  const lateFault = late.emit("worker", "fault", 2);
  assert.equal(lateFault.late, true);
  assert.equal(lateFault.identity, "late.alpha");
  const lateUse = late.emit("observer", "use", 1);
  assert.equal(lateUse.late, true);

  const workerLease = late.grantLease("worker");
  assert.equal(workerLease.lease, "llease1");
  const observerLease = late.grantLease("observer");
  assert.equal(observerLease.lease, "llease2");

  const workerTerminal = late.terminal("worker", "completed");
  assert.ok(workerTerminal.binding.startsWith("sha256:"));
  checkFactsInert(workerTerminal, "worker terminal facts");
  const observerTerminal = late.terminal("observer", "completed");
  assert.ok(observerTerminal.binding.startsWith("sha256:"));
  assert.notEqual(workerTerminal.binding, observerTerminal.binding);

  const seen = late.observe("worker", "worker", 2);
  assert.equal(seen.kind, "fault");
  assert.equal(seen.late, true);
  assert.equal(seen.terminal, "completed");
  assert.equal(seen.binding, workerTerminal.binding);
  checkFactsInert(seen, "observation facts");

  const boundLease = late.observeLease("worker", workerLease.lease);
  assert.equal(boundLease.released, false);
  const release = late.releaseLease("worker", workerLease.lease);
  assert.equal(release.terminal, "completed");
  assert.equal(release.binding, workerTerminal.binding);
  const reread = late.observeLease("worker", workerLease.lease);
  assert.equal(reread.released, true);
  const observerRelease = late.releaseLease("observer", observerLease.lease);
  assert.equal(observerRelease.binding, observerTerminal.binding);

  const workerBooks = late.reconcile("worker");
  assert.equal(workerBooks.admitted, 2);
  assert.equal(workerBooks.late, 1);
  assert.deepEqual(workerBooks.dropped, []);
  assert.equal(workerBooks.next_seq, 3);

  const done = late.outcome();
  assert.equal(done.outcome, "complete");
  assert.equal(done.reason, "both terminals witnessed");
  assert.equal(done.worker_dead, false);
  checkFactsInert(done, "outcome facts");

  const counters = late.counters();
  assert.deepEqual(counters, {
    participants: 2,
    admitted: 3,
    late: 2,
    dropped: 0,
    terminals: 2,
    leases: 2,
    releases: 2,
    observations: 3,
    rejected: 0,
  });
  checkFactsInert(counters, "late counters");
});

check("positive-every-identity-and-terminal", () => {
  for (const identity of LATE_IDENTITIES) {
    for (const tag of ["completed", "rejected", "failed"] as const) {
      const late = new LateOccurrenceService();
      late.select(identity);
      late.enroll("worker");
      late.enroll("observer");
      late.emit("worker", "use", 1);
      late.emit("observer", "use", 1);
      late.terminal("worker", tag);
      late.terminal("observer", tag);
      const done = late.outcome();
      assert.equal(done.outcome, "complete");
    }
  }
});

// --- Occurrence swap: cross-participant claims reject --------------------------

check("swap-observe-other-participant-event", () => {
  const late = cleanService();
  late.emit("worker", "fault", 1);
  late.emit("observer", "use", 1);
  // Occurrence swap detected: the observer cannot claim the worker's event.
  expectReject(() => late.observe("observer", "worker", 1), "wrong-owner");
  expectReject(() => late.observe("worker", "observer", 1), "wrong-owner");
  // Matching claims still bind.
  assert.equal(late.observe("worker", "worker", 1).kind, "fault");
  assert.equal(late.observe("observer", "observer", 1).kind, "use");
});

check("swap-lease-claim-other-holder", () => {
  const late = cleanService();
  const lease = late.grantLease("worker");
  late.terminal("worker", "completed");
  late.terminal("observer", "completed");
  // Lease swap detected on both the read and the release path.
  expectReject(() => late.observeLease("observer", lease.lease), "wrong-owner");
  expectReject(() => late.releaseLease("observer", lease.lease), "wrong-owner");
  assert.equal(late.observeLease("worker", lease.lease).holder, "worker");
});

// --- Dropped late events: sequence reconciliation ------------------------------

check("dropped-late-event-detected", () => {
  const late = cleanService();
  late.gate("worker");
  late.emit("worker", "fault", 1);
  const skipped = late.emit("worker", "use", 3);
  assert.deepEqual(skipped.dropped, [2]);
  const books = late.reconcile("worker");
  assert.equal(books.admitted, 2);
  assert.equal(books.late, 2);
  assert.deepEqual(books.dropped, [2]);
  assert.equal(books.next_seq, 4);
  checkFactsInert(books, "reconcile facts");
  // The dropped sequence stays reported on later reconciliations.
  late.emit("worker", "use", 4);
  assert.deepEqual(late.reconcile("worker").dropped, [2]);
  assert.equal(late.counters().dropped, 1);
});

check("dropped-replay-still-rejects", () => {
  const late = cleanService();
  late.emit("worker", "use", 1);
  expectReject(() => late.emit("worker", "use", 1), "stale-handle");
  expectReject(() => late.observe("worker", "worker", 9), "stale-handle");
});

// --- Early release: pre-terminal lease facts are unbound -----------------------

check("early-release-rejects-unbound", () => {
  const late = cleanService();
  const lease = late.grantLease("worker");
  // Lease facts observed before the witnessed terminal event reject,
  // mirroring the K05 eager-read rule.
  expectReject(() => late.observeLease("worker", lease.lease), "permission");
  // Releasing before the terminal event is witnessed is an early release.
  expectReject(() => late.releaseLease("worker", lease.lease), "permission");
  // The same lease binds once the terminal event is witnessed.
  late.terminal("worker", "completed");
  late.terminal("observer", "rejected");
  assert.equal(late.observeLease("worker", lease.lease).released, false);
  const release = late.releaseLease("worker", lease.lease);
  assert.equal(release.terminal, "completed");
  assert.ok(release.binding.startsWith("sha256:"));
});

check("early-release-one-sided-terminal", () => {
  const late = cleanService();
  const workerLease = late.grantLease("worker");
  const observerLease = late.grantLease("observer");
  late.terminal("worker", "failed");
  // Only the witnessed side binds; the other side stays unbound.
  assert.equal(late.observeLease("worker", workerLease.lease).holder, "worker");
  expectReject(() => late.observeLease("observer", observerLease.lease), "permission");
  expectReject(() => late.releaseLease("observer", observerLease.lease), "permission");
  assert.equal(late.outcome().outcome, "incomplete");
});

// --- Worker death: terminal incomplete, never a pass, never silent ------------

check("worker-death-yields-incomplete", () => {
  const late = cleanService();
  late.emit("worker", "use", 1);
  late.emit("observer", "use", 1);
  const death = late.killWorker();
  assert.equal(death.outcome, "incomplete");
  assert.equal(death.reason, "worker-dead");
  assert.equal(death.worker_dead, true);
  checkFactsInert(death, "death outcome facts");
  // Worker admission closes: emits, terminals, and leases all reject.
  expectReject(() => late.emit("worker", "use", 2), "permission");
  expectReject(() => late.terminal("worker", "completed"), "permission");
  expectReject(() => late.grantLease("worker"), "permission");
  expectReject(() => late.enroll("worker"), "permission");
  // The observer can still finish, but the run never passes.
  late.terminal("observer", "completed");
  const done = late.outcome();
  assert.equal(done.outcome, "incomplete");
  assert.equal(done.reason, "worker-dead");
  assert.equal(done.observer_terminal, "completed");
  assert.equal(done.worker_terminal, null);
});

check("worker-death-before-any-terminal", () => {
  const late = cleanService();
  late.killWorker();
  const done = late.outcome();
  assert.equal(done.outcome, "incomplete");
  assert.equal(done.reason, "worker-dead");
  // Prior observations stay readable; the outcome names the death.
  expectReject(() => late.observe("worker", "worker", 1), "stale-handle");
});

// --- Identity, vocabulary, and bounds ------------------------------------------

check("identity-select-once", () => {
  const late = new LateOccurrenceService();
  assert.equal(late.select("late.beta").joined, false);
  assert.equal(late.select("late.beta").joined, true);
  expectReject(() => late.select("late.alpha"), "changed-input");
  expectReject(() => late.select("late.gamma"), "unsupported-capability");
  expectReject(() => late.select(42), "unsupported-capability");
});

check("identity-required-before-enroll", () => {
  const late = new LateOccurrenceService();
  expectReject(() => late.enroll("worker"), "invalid-request");
  expectReject(() => late.emit("worker", "use", 1), "invalid-request");
  expectReject(() => late.terminal("worker", "completed"), "invalid-request");
});

check("vocab-closed", () => {
  const late = cleanService();
  expectReject(() => late.enroll("controller"), "unsupported-capability");
  expectReject(() => late.gate("controller"), "unsupported-capability");
  expectReject(() => late.emit("worker", "spawn", 1), "unsupported-capability");
  expectReject(() => late.emit("worker", "use", 0), "invalid-request");
  expectReject(() => late.emit("worker", "use", 1.5), "invalid-request");
  expectReject(() => late.terminal("worker", "succeeded"), "unsupported-capability");
  expectReject(() => late.observeLease("worker", "llease0"), "not-found");
  expectReject(() => late.observeLease("worker", "llease99"), "not-found");
  expectReject(() => late.releaseLease("worker", "nope"), "not-found");
});

check("terminal-replay-rejects", () => {
  const late = cleanService();
  late.terminal("worker", "completed");
  expectReject(() => late.terminal("worker", "failed"), "stale-handle");
  expectReject(() => late.observe("controller", "worker", 1), "unsupported-capability");
});

check("terminal-closes-admission", () => {
  const late = cleanService();
  late.emit("worker", "use", 1);
  late.terminal("worker", "completed");
  // The terminal binding covers every admitted occurrence: nothing more
  // can arrive after it.
  expectReject(() => late.emit("worker", "use", 2), "permission");
  expectReject(() => late.emit("worker", "fault", 9), "permission");
  // The other participant is unaffected until its own terminal.
  late.emit("observer", "use", 1);
  assert.equal(late.reconcile("worker").admitted, 1);
});

check("terminal-kind-event-is-not-the-witness", () => {
  const late = cleanService();
  const lease = late.grantLease("worker");
  // An occurrence merely labeled "terminal" is not the witnessed terminal
  // event: lease facts stay unbound until terminal() runs.
  const labeled = late.emit("worker", "terminal", 1);
  assert.equal(labeled.kind, "terminal");
  expectReject(() => late.observeLease("worker", lease.lease), "permission");
  expectReject(() => late.releaseLease("worker", lease.lease), "permission");
  late.terminal("worker", "completed");
  assert.equal(late.observeLease("worker", lease.lease).holder, "worker");
});

check("gate-joins-and-release-joins", () => {
  const late = cleanService();
  const first = late.gate("worker");
  assert.equal(first.joined, false);
  const joined = late.gate("worker");
  assert.equal(joined.joined, true);
  assert.equal(joined.gate_seq, first.gate_seq);
  const lease = late.grantLease("worker");
  late.terminal("worker", "completed");
  late.terminal("observer", "completed");
  const release = late.releaseLease("worker", lease.lease);
  const repeat = late.releaseLease("worker", lease.lease);
  assert.equal(repeat.binding, release.binding);
  assert.equal(repeat.lease, lease.lease);
  // The repeat joins: same facts, no additional count.
  assert.equal(late.counters().releases, 1);
});

check("bounds-event-cap", () => {
  const late = cleanService();
  for (let seq = 1; seq <= MAX_LATE_EVENTS; seq += 1) {
    late.emit("worker", "use", seq);
  }
  expectReject(() => late.emit("observer", "use", 1), "resource-limit");
  assert.equal(late.counters().admitted, MAX_LATE_EVENTS);
});

check("bounds-gap-cap", () => {
  const late = cleanService();
  late.emit("worker", "use", 1);
  // A gap wider than the service can ever admit rejects instead of
  // allocating an unbounded dropped list.
  expectReject(() => late.emit("worker", "use", 2 + MAX_LATE_EVENTS + 1), "resource-limit");
  expectReject(() => late.emit("worker", "use", Number.MAX_SAFE_INTEGER), "resource-limit");
  // The boundary gap admits with every skipped number recorded.
  const edge = late.emit("worker", "use", 2 + MAX_LATE_EVENTS);
  assert.equal(edge.dropped.length, MAX_LATE_EVENTS);
  assert.deepEqual(late.reconcile("worker").dropped.slice(0, 3), [2, 3, 4]);
});

check("bounds-lease-cap", () => {
  const late = cleanService();
  for (let index = 0; index < MAX_LATE_LEASES; index += 1) {
    late.grantLease(index % 2 === 0 ? "worker" : "observer");
  }
  expectReject(() => late.grantLease("worker"), "resource-limit");
  assert.equal(late.counters().leases, MAX_LATE_LEASES);
});

check("bounds-rejections-counted", () => {
  const late = cleanService();
  expectReject(() => late.observe("observer", "worker", 1), "wrong-owner");
  expectReject(() => late.select("late.beta"), "changed-input");
  assert.equal(late.counters().rejected, 2);
  checkFactsInert(late.counters(), "late counters");
});

console.log(
  JSON.stringify({
    schema: "can.native-test.late-check",
    passed: passed.length,
    checks: passed,
  }),
);
