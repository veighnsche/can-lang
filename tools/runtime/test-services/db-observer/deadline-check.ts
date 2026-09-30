// K26 bounded self-check: SQL deadline separate from server settlement.
//
// Run with: bun tools/runtime/test-services/db-observer/deadline-check.ts
// Local controls only. No databases, drivers, sockets, files, timers, live
// hosts, or live runtimes. Every loop below is bounded by a small constant.
// Live host-dependent controls wait for the qualified profile and Q task.
// QD4 holds D4 credit; nothing here decides settlement expectations.

import { strict as assert } from "node:assert";
import {
  checkLimits,
  DbObserverError,
  type DbObserverCode,
  type DbObserverLayer,
  type DbObserverLimits,
} from "./core.ts";
import {
  DEADLINE_CANCEL_BOUNDARY,
  DEADLINE_DRIVER_SERVER_BOUNDARY,
  DEADLINE_MAX_WORKS,
  DEADLINE_OBSERVER_SCHEMA_VERSION,
  DEADLINE_QD4_REQUIREMENTS,
  DeadlineObserver,
  isCancelCapable,
} from "./deadline.ts";

const passed: string[] = [];
function check(name: string, body: () => void): void {
  body();
  passed.push(name);
}

function expectObserverError(
  body: () => unknown,
  code: DbObserverCode,
  layer: DbObserverLayer,
): void {
  try {
    body();
  } catch (error) {
    if (!(error instanceof DbObserverError)) {
      assert.fail(`expected a DbObserverError, got ${String(error)}`);
    }
    assert.equal(error.code, code);
    assert.equal(error.layer, layer);
    assert.ok(error.message.includes(`[${layer}:${code}]`));
    return;
  }
  assert.fail(`expected failure ${layer}:${code}`);
}

function roomyLimits(): DbObserverLimits {
  return checkLimits({
    maxNamespaces: 4,
    maxConnections: 8,
    maxTables: 8,
    maxRows: 16,
    maxCellsPerRow: 8,
    maxCellBytes: 1024,
  });
}

function openObserver(): DeadlineObserver {
  return new DeadlineObserver("owner-n", "ns-alpha", roomyLimits());
}

// --- Dispatch runs at most once per work ------------------------------------

check("dispatch-at-most-once", () => {
  const observer = openObserver();
  const first = observer.dispatch("w-0", "UPDATE t SET a = 1", "postgres");
  assert.equal(first.facts.work, "w-0");
  assert.equal(first.facts.engine, "postgres");
  assert.equal(first.facts.namespace, "ns-alpha");
  assert.ok(first.facts.statementDigest.startsWith("sha256:"));
  assert.ok(!first.facts.statementDigest.includes("UPDATE"));
  assert.ok(first.facts.handleDigest.startsWith("sha256:"));
  assert.ok(first.token.length > 0);
  assert.equal(observer.workCount, 1);

  // A second dispatch of the same work refuses; the first fact stands.
  expectObserverError(
    () => observer.dispatch("w-0", "UPDATE t SET a = 2", "postgres"),
    "connection-busy",
    "driver",
  );
  assert.equal(observer.dispatchFacts("w-0").statementDigest, first.facts.statementDigest);

  // Unknown vocabulary rejects at the sql layer with nothing stored.
  expectObserverError(
    () => observer.dispatch("", "SELECT 1", "postgres"),
    "malformed-statement",
    "sql",
  );
  expectObserverError(
    () => observer.dispatch("bad work!", "SELECT 1", "postgres"),
    "malformed-statement",
    "sql",
  );
  expectObserverError(() => observer.dispatch("w-1", "", "postgres"), "malformed-statement", "sql");
  expectObserverError(
    () => observer.dispatch("w-1", "SELECT 1", "oracle"),
    "malformed-statement",
    "sql",
  );
  expectObserverError(() => observer.dispatchFacts("ghost"), "unknown-connection", "driver");
  assert.equal(observer.workCount, 1);

  // The work table is bounded at eight.
  assert.equal(DEADLINE_MAX_WORKS, 8);
  const tight = openObserver();
  for (let index = 0; index < DEADLINE_MAX_WORKS; index++) {
    tight.dispatch(`k-${index}`, "SELECT 1", "sqlite");
  }
  expectObserverError(
    () => tight.dispatch("k-overflow", "SELECT 1", "sqlite"),
    "capacity-exhausted",
    "engine",
  );
});

// --- The visible deadline is its own fact ------------------------------------

check("visible-deadline-before-driver-settlement", () => {
  const observer = openObserver();
  const { token } = observer.dispatch("w-0", "UPDATE t SET a = 1", "postgres");

  // Driver settlement without an observed deadline refuses.
  expectObserverError(
    () => observer.recordDriverSettlement("w-0", token, "timed-out"),
    "no-conversation",
    "driver",
  );
  expectObserverError(() => observer.deadlineFacts("w-0"), "no-conversation", "driver");

  const deadline = observer.observeDeadline("w-0", token, 5000);
  assert.equal(deadline.work, "w-0");
  assert.equal(deadline.engine, "postgres");
  assert.equal(deadline.deadlineMs, 5000);
  assert.ok(deadline.digest.startsWith("sha256:"));
  assert.equal(observer.deadlineFacts("w-0").digest, deadline.digest);

  // Observing twice refuses; a malformed tick refuses without storing.
  expectObserverError(
    () => observer.observeDeadline("w-0", token, 6000),
    "connection-busy",
    "driver",
  );
  assert.equal(observer.deadlineFacts("w-0").deadlineMs, 5000);
  const second = observer.dispatch("w-1", "SELECT 1", "postgres");
  expectObserverError(
    () => observer.observeDeadline("w-1", second.token, -1),
    "malformed-statement",
    "sql",
  );
  expectObserverError(
    () => observer.observeDeadline("w-1", second.token, 1.5),
    "malformed-statement",
    "sql",
  );
  expectObserverError(() => observer.deadlineFacts("w-1"), "no-conversation", "driver");
});

// --- Driver settlement is local-only ------------------------------------------

check("driver-settlement-is-local-only", () => {
  const observer = openObserver();
  const timed = observer.dispatch("w-timeout", "UPDATE t SET a = 1", "postgres");
  observer.observeDeadline("w-timeout", timed.token, 50);
  const settlement = observer.recordDriverSettlement("w-timeout", timed.token, "timed-out");
  assert.equal(settlement.outcome, "timed-out");
  assert.equal(settlement.serverEffect, "unresolved");
  assert.ok(settlement.digest.startsWith("sha256:"));
  assert.equal(observer.driverFacts("w-timeout").digest, settlement.digest);

  const done = observer.dispatch("w-done", "SELECT 1", "mysql");
  observer.observeDeadline("w-done", done.token, 100);
  const completed = observer.recordDriverSettlement("w-done", done.token, "completed");
  assert.equal(completed.outcome, "completed");
  assert.equal(completed.serverEffect, "unresolved");

  // Unknown outcomes reject; settling twice refuses; the first fact stands.
  const third = observer.dispatch("w-2", "SELECT 1", "sqlite");
  observer.observeDeadline("w-2", third.token, 10);
  expectObserverError(
    () => observer.recordDriverSettlement("w-2", third.token, "maybe"),
    "malformed-statement",
    "sql",
  );
  expectObserverError(() => observer.driverFacts("w-2"), "no-conversation", "driver");
  expectObserverError(
    () => observer.recordDriverSettlement("w-timeout", timed.token, "completed"),
    "connection-busy",
    "driver",
  );
  assert.equal(observer.driverFacts("w-timeout").outcome, "timed-out");

  // The boundary names the split: driver-local facts never imply the server.
  assert.ok(DEADLINE_DRIVER_SERVER_BOUNDARY.startsWith("driver-local:"));
  assert.ok(DEADLINE_DRIVER_SERVER_BOUNDARY.includes("never implies server settlement"));
  assert.ok(DEADLINE_DRIVER_SERVER_BOUNDARY.includes("server acknowledgment alone"));
  assert.ok(DEADLINE_DRIVER_SERVER_BOUNDARY.includes("QD4"));
  assert.deepEqual(Array.from(DEADLINE_QD4_REQUIREMENTS), [
    "visible-deadline",
    "driver-settlement",
    "server-ack",
    "fence",
    "qd4-verdict",
  ]);
  assert.equal(DEADLINE_OBSERVER_SCHEMA_VERSION, "1");
});

// --- Server acknowledgment reconciles the effect -------------------------------

check("server-ack-reconciles-effect", () => {
  const observer = openObserver();
  const { token } = observer.dispatch("w-0", "UPDATE t SET a = 1", "postgres");

  // Reconciliation runs after the driver view settles.
  expectObserverError(
    () => observer.recordServerAck("w-0", token, "applied"),
    "no-conversation",
    "driver",
  );
  expectObserverError(() => observer.serverFacts("w-0"), "no-conversation", "driver");

  observer.observeDeadline("w-0", token, 50);
  observer.recordDriverSettlement("w-0", token, "timed-out");
  expectObserverError(
    () => observer.recordServerAck("w-0", token, "maybe"),
    "malformed-statement",
    "sql",
  );
  const ack = observer.recordServerAck("w-0", token, "applied");
  assert.equal(ack.effect, "applied");
  assert.equal(ack.engine, "postgres");
  assert.ok(ack.digest.startsWith("sha256:"));
  assert.equal(observer.serverFacts("w-0").digest, ack.digest);

  // A second acknowledgment refuses; the first fact stands.
  expectObserverError(
    () => observer.recordServerAck("w-0", token, "absent"),
    "connection-busy",
    "driver",
  );
  assert.equal(observer.serverFacts("w-0").effect, "applied");

  // "absent" and "unknown" are explicit recorded effects, not gaps.
  const gone = observer.dispatch("w-absent", "DELETE FROM t", "sqlite");
  observer.observeDeadline("w-absent", gone.token, 10);
  observer.recordDriverSettlement("w-absent", gone.token, "completed");
  assert.equal(observer.recordServerAck("w-absent", gone.token, "absent").effect, "absent");
  const lost = observer.dispatch("w-unknown", "UPDATE t SET b = 2", "mysql");
  observer.observeDeadline("w-unknown", lost.token, 10);
  observer.recordDriverSettlement("w-unknown", lost.token, "timed-out");
  assert.equal(observer.recordServerAck("w-unknown", lost.token, "unknown").effect, "unknown");
  assert.equal(observer.serverFacts("w-unknown").effect, "unknown");
});

// --- Unknown server effect blocks cleanup ---------------------------------------

check("unknown-server-effect-blocks-cleanup", () => {
  const observer = openObserver();
  const { token } = observer.dispatch("w-0", "UPDATE t SET a = 1", "postgres");

  // Every missing step on the path blocks release.
  expectObserverError(() => observer.release("w-0", token), "connection-busy", "driver");
  observer.observeDeadline("w-0", token, 50);
  observer.recordDriverSettlement("w-0", token, "timed-out");
  expectObserverError(() => observer.release("w-0", token), "no-conversation", "driver");
  expectObserverError(() => observer.releaseAck("w-0"), "no-conversation", "driver");

  // An explicitly unknown server effect blocks cleanup even after fencing.
  observer.recordServerAck("w-0", token, "unknown");
  observer.quiesceEngine("postgres");
  observer.fence("w-0", token);
  expectObserverError(() => observer.release("w-0", token), "connection-busy", "driver");
  assert.equal(observer.leaseFacts("w-0").retained, true);

  // A known effect still needs the fence before release.
  const known = observer.dispatch("w-1", "UPDATE t SET a = 1", "mysql");
  observer.observeDeadline("w-1", known.token, 50);
  observer.recordDriverSettlement("w-1", known.token, "timed-out");
  observer.recordServerAck("w-1", known.token, "applied");
  expectObserverError(() => observer.release("w-1", known.token), "no-conversation", "driver");
  observer.quiesceEngine("mysql");
  observer.fence("w-1", known.token);
  const ack = observer.release("w-1", known.token);
  assert.equal(ack.work, "w-1");
  assert.equal(ack.engine, "mysql");
  assert.equal(ack.released, true);
  assert.ok(ack.ackDigest.startsWith("sha256:"));
  assert.equal(observer.releaseAck("w-1").ackDigest, ack.ackDigest);
  assert.equal(observer.leaseFacts("w-1").retained, false);

  // Releasing twice fails; nothing runs on a released handle.
  expectObserverError(() => observer.release("w-1", known.token), "closed-handle", "engine");
  expectObserverError(
    () => observer.observeDeadline("w-1", known.token, 60),
    "closed-handle",
    "engine",
  );
  expectObserverError(
    () => observer.recordServerAck("w-1", known.token, "applied"),
    "closed-handle",
    "engine",
  );
});

// --- No unsupported cancellation claim -------------------------------------------

check("no-unsupported-cancellation-claim", () => {
  assert.equal(isCancelCapable("postgres"), true);
  assert.equal(isCancelCapable("sqlite"), false);
  assert.equal(isCancelCapable("mysql"), false);

  const observer = openObserver();
  const pg = observer.dispatch("w-pg", "UPDATE t SET a = 1", "postgres");
  const lite = observer.dispatch("w-lite", "UPDATE t SET a = 1", "sqlite");

  // Only a cancel-capable engine mints cancel grants.
  const held = observer.acquireCancelGrant("postgres");
  assert.equal(held.facts.engine, "postgres");
  assert.ok(held.facts.grantDigest.startsWith("sha256:"));
  expectObserverError(() => observer.acquireCancelGrant("sqlite"), "malformed-statement", "sql");
  expectObserverError(() => observer.acquireCancelGrant("mysql"), "malformed-statement", "sql");
  expectObserverError(() => observer.acquireCancelGrant("oracle"), "malformed-statement", "sql");

  // Cancellation without a capable grant refuses: unknown, invented, and
  // wrong-engine grants are never authority, and neither is the work token.
  expectObserverError(
    () => observer.requestCancel("w-pg", pg.token, "invented-grant"),
    "forged-token",
    "driver",
  );
  expectObserverError(
    () => observer.requestCancel("w-pg", pg.token, pg.token),
    "forged-token",
    "driver",
  );
  expectObserverError(
    () => observer.requestCancel("w-lite", lite.token, held.grant),
    "forged-token",
    "driver",
  );
  expectObserverError(() => observer.cancelFacts("w-pg"), "no-conversation", "driver");

  // A backed cancel records facts that prove nothing about the server.
  const cancel = observer.requestCancel("w-pg", pg.token, held.grant);
  assert.equal(cancel.work, "w-pg");
  assert.equal(cancel.proves, "nothing-about-server");
  assert.equal(observer.cancelFacts("w-pg").digest, cancel.digest);
  expectObserverError(
    () => observer.requestCancel("w-pg", pg.token, held.grant),
    "connection-busy",
    "driver",
  );

  // The cancel changes no server fact and unblocks no cleanup: the work
  // still needs its deadline, driver settlement, known server effect, and
  // fence before release.
  expectObserverError(() => observer.serverFacts("w-pg"), "no-conversation", "driver");
  // Dispatch the stale-cancel work before quiescence closes the engine.
  const stale = observer.dispatch("w-stale", "SELECT 1", "postgres");
  observer.observeDeadline("w-pg", pg.token, 50);
  observer.recordDriverSettlement("w-pg", pg.token, "timed-out");
  observer.recordServerAck("w-pg", pg.token, "unknown");
  observer.quiesceEngine("postgres");
  observer.fence("w-pg", pg.token);
  expectObserverError(() => observer.release("w-pg", pg.token), "connection-busy", "driver");

  // Once the server has acknowledged, a cancel is stale and refuses.
  observer.observeDeadline("w-stale", stale.token, 10);
  observer.recordDriverSettlement("w-stale", stale.token, "completed");
  observer.recordServerAck("w-stale", stale.token, "applied");
  expectObserverError(
    () => observer.requestCancel("w-stale", stale.token, held.grant),
    "connection-busy",
    "driver",
  );

  // Cancel on a released handle refuses at the engine layer.
  observer.fence("w-stale", stale.token);
  observer.release("w-stale", stale.token);
  expectObserverError(
    () => observer.requestCancel("w-stale", stale.token, held.grant),
    "closed-handle",
    "engine",
  );

  assert.ok(DEADLINE_CANCEL_BOUNDARY.startsWith("capable-grant-only:"));
  assert.ok(DEADLINE_CANCEL_BOUNDARY.includes("without a capable grant refuses"));
  assert.ok(DEADLINE_CANCEL_BOUNDARY.includes("proves nothing about the server"));
});

// --- Engine quiescence and fence survey, never settle ------------------------------

check("quiescence-fence-adapter", () => {
  const observer = openObserver();
  const { token } = observer.dispatch("w-0", "UPDATE t SET a = 1", "postgres");
  observer.observeDeadline("w-0", token, 50);
  observer.recordDriverSettlement("w-0", token, "timed-out");

  // Fencing requires quiescence first.
  expectObserverError(() => observer.fence("w-0", token), "no-conversation", "driver");
  expectObserverError(() => observer.fenceFacts("w-0"), "no-conversation", "driver");
  expectObserverError(() => observer.quiesceEngine("oracle"), "malformed-statement", "sql");

  // Quiescence reports the live set and stops new dispatches on that
  // engine only; other engines still admit.
  assert.deepEqual(observer.quiesceEngine("postgres"), ["w-0"]);
  assert.deepEqual(observer.quiesceEngine("postgres"), ["w-0"]);
  expectObserverError(
    () => observer.dispatch("w-late", "SELECT 1", "postgres"),
    "namespace-closed",
    "engine",
  );
  const other = observer.dispatch("w-other", "SELECT 1", "sqlite");
  assert.equal(other.facts.engine, "sqlite");

  // The fence surveys even with an unknown server effect, joins on
  // repeats, and settles nothing.
  observer.recordServerAck("w-0", token, "unknown");
  const fence = observer.fence("w-0", token);
  assert.equal(fence.work, "w-0");
  assert.equal(fence.settles, "nothing");
  assert.ok(fence.ackDigest.startsWith("sha256:"));
  assert.equal(observer.fence("w-0", token).ackDigest, fence.ackDigest);
  assert.equal(observer.fenceFacts("w-0").ackDigest, fence.ackDigest);
  expectObserverError(() => observer.release("w-0", token), "connection-busy", "driver");
});

// --- The namespace lease is retained until release ----------------------------------

check("retained-lease-until-release", () => {
  const observer = openObserver();
  const { facts, token } = observer.dispatch("w-0", "UPDATE t SET a = 1", "postgres");
  const lease = observer.leaseFacts("w-0");
  assert.equal(lease.work, "w-0");
  assert.equal(lease.engine, "postgres");
  assert.equal(lease.namespace, "ns-alpha");
  assert.equal(lease.handleDigest, facts.handleDigest);
  assert.equal(lease.retained, true);

  // The lease survives every intermediate step and drops only on release.
  observer.observeDeadline("w-0", token, 50);
  assert.equal(observer.leaseFacts("w-0").retained, true);
  observer.recordDriverSettlement("w-0", token, "timed-out");
  observer.recordServerAck("w-0", token, "absent");
  observer.quiesceEngine("postgres");
  observer.fence("w-0", token);
  assert.equal(observer.leaseFacts("w-0").retained, true);
  assert.equal(observer.liveWorkCount, 1);
  observer.release("w-0", token);
  assert.equal(observer.leaseFacts("w-0").retained, false);
  assert.equal(observer.liveWorkCount, 0);
  expectObserverError(() => observer.leaseFacts("ghost"), "unknown-connection", "driver");
});

// --- Bounded delayed-write and prestart controls --------------------------------------

check("delayed-write-and-prestart", () => {
  // Delayed write: the driver times out at the visible deadline, the
  // server applies late, and the reconciled cleanup still completes.
  const observer = openObserver();
  const { token } = observer.dispatch("w-delayed", "UPDATE t SET a = 1", "postgres");
  observer.observeDeadline("w-delayed", token, 25);
  const driver = observer.recordDriverSettlement("w-delayed", token, "timed-out");
  assert.equal(driver.serverEffect, "unresolved");
  const server = observer.recordServerAck("w-delayed", token, "applied");
  assert.equal(server.effect, "applied");
  observer.quiesceEngine("postgres");
  observer.fence("w-delayed", token);
  assert.equal(observer.release("w-delayed", token).released, true);

  // Prestart: quiescence and fence over an engine with no live work is an
  // empty survey, not an error, and later dispatches still refuse there.
  const fresh = openObserver();
  assert.deepEqual(fresh.quiesceEngine("sqlite"), []);
  expectObserverError(
    () => fresh.dispatch("w-prestart", "SELECT 1", "sqlite"),
    "namespace-closed",
    "engine",
  );
});

// --- Every failure names its layer -----------------------------------------------------

check("every-failure-names-its-layer", () => {
  const observer = openObserver();
  // SQL: statement, engine, outcome, and tick vocabulary.
  expectObserverError(
    () => observer.dispatch("w-0", "SELECT 1", "oracle"),
    "malformed-statement",
    "sql",
  );
  const { token } = observer.dispatch("w-0", "SELECT 1", "postgres");
  expectObserverError(
    () => observer.observeDeadline("w-0", token, Number.NaN),
    "malformed-statement",
    "sql",
  );
  // Driver: tokens, dispatch discipline, and lifecycle order.
  expectObserverError(() => observer.observeDeadline("w-0", "wrong", 10), "forged-token", "driver");
  expectObserverError(
    () => observer.dispatch("w-0", "SELECT 1", "postgres"),
    "connection-busy",
    "driver",
  );
  expectObserverError(() => observer.driverFacts("w-0"), "no-conversation", "driver");
  // Engine: capacity and released-handle lifecycle.
  const tight = openObserver();
  for (let index = 0; index < DEADLINE_MAX_WORKS; index++) {
    tight.dispatch(`c-${index}`, "SELECT 1", "mysql");
  }
  expectObserverError(
    () => tight.dispatch("c-overflow", "SELECT 1", "mysql"),
    "capacity-exhausted",
    "engine",
  );
  observer.observeDeadline("w-0", token, 10);
  observer.recordDriverSettlement("w-0", token, "completed");
  observer.recordServerAck("w-0", token, "applied");
  observer.quiesceEngine("postgres");
  observer.fence("w-0", token);
  observer.release("w-0", token);
  expectObserverError(() => observer.release("w-0", token), "closed-handle", "engine");
});

console.log(
  JSON.stringify({
    kind: "can.db-observer-deadline-check",
    schema_version: DEADLINE_OBSERVER_SCHEMA_VERSION,
    checks: passed,
    count: passed.length,
  }),
);
