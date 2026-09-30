// K25 bounded self-check: poisoned transaction sentinel observations.
//
// Run with: bun tools/runtime/test-services/db-observer/poison-check.ts
// Local controls only. No databases, drivers, sockets, files, timers, live
// hosts, or live runtimes. Every loop below is bounded by a small constant.
// Live host-dependent controls wait for the qualified profile and Q task.
// QD3 holds D3 credit; nothing here claims execution credit.

import { strict as assert } from "node:assert";
import {
  checkLimits,
  DbObserverError,
  type DbObserverCode,
  type DbObserverLayer,
  type DbObserverLimits,
} from "./core.ts";
import {
  POISON_CALLBACK_NON_PROOF_BOUNDARY,
  POISON_CREDIT_REQUIREMENTS,
  POISON_MAX_ATTEMPTS,
  POISON_OBSERVER_SCHEMA_VERSION,
  PoisonObserver,
} from "./poison.ts";

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

function openObserver(): PoisonObserver {
  return new PoisonObserver(roomyLimits());
}

const SENTINEL_SCHEMA = ["number", "text"] as const;

function sentinelRow(): readonly unknown[] {
  return [
    { tag: "number", lexeme: "9007199254740993" },
    { tag: "text", text: "poison-sentinel" },
  ];
}

// --- Attempts: poison begin pins sentinel + statement digest ----------------

check("poison-attempt-pins-sentinel", () => {
  const observer = openObserver();
  const record = observer.beginPoisonAttempt(
    [...SENTINEL_SCHEMA],
    sentinelRow(),
    "INSERT INTO t (id) VALUES (1), (1)",
  );
  assert.equal(record.attempt, "attempt-0");
  assert.equal(record.kind, "poison");
  assert.deepEqual(Array.from(record.schema), ["number", "text"]);
  assert.ok(record.sentinelDigest.startsWith("sha256:"));
  assert.equal(record.sentinelDigest.length, "sha256:".length + 64);
  assert.ok(record.poisonDigest?.startsWith("sha256:"));
  assert.deepEqual(Object.keys(record).sort(), [
    "attempt",
    "kind",
    "poisonDigest",
    "schema",
    "sentinelDigest",
  ]);

  // The same sentinel under a second attempt pins independently.
  const again = observer.beginPoisonAttempt(
    [...SENTINEL_SCHEMA],
    sentinelRow(),
    "INSERT INTO t (id) VALUES (1), (1)",
  );
  assert.equal(again.attempt, "attempt-1");
  assert.equal(again.sentinelDigest, record.sentinelDigest);
  assert.equal(again.poisonDigest, record.poisonDigest);
  assert.equal(observer.attemptCount, 2);
  assert.equal(observer.attemptRecord("attempt-0").kind, "poison");

  // Malformed attempts reject at the sql layer with nothing stored.
  expectObserverError(
    () => observer.beginPoisonAttempt([], sentinelRow(), "SELECT 1"),
    "malformed-statement",
    "sql",
  );
  expectObserverError(
    () =>
      observer.beginPoisonAttempt(
        [...SENTINEL_SCHEMA],
        [{ tag: "number", lexeme: "1" }],
        "SELECT 1",
      ),
    "row-arity",
    "sql",
  );
  expectObserverError(
    () =>
      observer.beginPoisonAttempt(
        [...SENTINEL_SCHEMA],
        [
          { tag: "text", text: "1" },
          { tag: "text", text: "x" },
        ],
        "SELECT 1",
      ),
    "wrong-cell-type",
    "sql",
  );
  expectObserverError(
    () => observer.beginPoisonAttempt([...SENTINEL_SCHEMA], sentinelRow(), ""),
    "malformed-statement",
    "sql",
  );
  expectObserverError(() => observer.attemptRecord("not-an-attempt"), "malformed-statement", "sql");
  expectObserverError(() => observer.attemptRecord("attempt-99"), "unknown-connection", "driver");
  assert.equal(observer.attemptCount, 2);

  // The attempt table is bounded.
  const tight = openObserver();
  for (let index = 0; index < POISON_MAX_ATTEMPTS; index++) {
    tight.beginPoisonAttempt(["null"], [{ tag: "null" }], "SELECT 1");
  }
  expectObserverError(
    () => tight.beginPoisonAttempt(["null"], [{ tag: "null" }], "SELECT 1"),
    "capacity-exhausted",
    "engine",
  );
});

// --- PG error-code observations ---------------------------------------------

check("pg-error-code-observations", () => {
  const observer = openObserver();
  const record = observer.beginPoisonAttempt(
    [...SENTINEL_SCHEMA],
    sentinelRow(),
    "INSERT INTO t (id) VALUES (1), (1)",
  );
  // Missing error evidence stays missing: never an empty-pass default.
  expectObserverError(() => observer.errorFacts(record.attempt), "no-conversation", "driver");

  const facts = observer.recordPoisonError(record.attempt, "23505", "duplicate key value");
  assert.equal(facts.attempt, record.attempt);
  assert.equal(facts.code, "23505");
  assert.equal(facts.class, "23");
  assert.equal(facts.detail, "duplicate key value");
  assert.ok(facts.detailDigest.startsWith("sha256:"));
  assert.deepEqual(Object.keys(facts).sort(), [
    "attempt",
    "class",
    "code",
    "detail",
    "detailDigest",
  ]);
  assert.equal(observer.errorFacts(record.attempt).code, "23505");

  // A second error observation on one attempt throws; the first stands.
  expectObserverError(
    () => observer.recordPoisonError(record.attempt, "40001", "serialization failure"),
    "connection-busy",
    "driver",
  );
  assert.equal(observer.errorFacts(record.attempt).code, "23505");

  // Malformed codes reject at the sql layer. Each probe uses a fresh
  // attempt on a dedicated observer so failures leave no trace.
  const probes = openObserver();
  for (const bad of ["", "2350", "235055", "2350x", "23 05", "abcde", 23505, null]) {
    const probe = probes.beginPoisonAttempt([...SENTINEL_SCHEMA], sentinelRow(), "SELECT 1");
    expectObserverError(
      () => probes.recordPoisonError(probe.attempt, bad, "detail"),
      "malformed-statement",
      "sql",
    );
    expectObserverError(() => probes.errorFacts(probe.attempt), "no-conversation", "driver");
  }
  assert.equal(probes.attemptCount, POISON_MAX_ATTEMPTS);

  // Success-class codes are refused as non-errors on their own observer.
  const coded = openObserver();
  const successProbe = coded.beginPoisonAttempt([...SENTINEL_SCHEMA], sentinelRow(), "SELECT 1");
  for (const nonError of ["00000", "00100"]) {
    expectObserverError(
      () => coded.recordPoisonError(successProbe.attempt, nonError, "not an error"),
      "malformed-statement",
      "sql",
    );
  }
  expectObserverError(() => coded.errorFacts(successProbe.attempt), "no-conversation", "driver");

  // Empty and oversized details reject; a distinct class records distinctly.
  expectObserverError(
    () => coded.recordPoisonError(successProbe.attempt, "22P02", ""),
    "malformed-statement",
    "sql",
  );
  expectObserverError(
    () => coded.recordPoisonError(successProbe.attempt, "22P02", "x".repeat(513)),
    "capacity-exhausted",
    "engine",
  );
  const typed = coded.recordPoisonError(
    successProbe.attempt,
    "22P02",
    "invalid text representation",
  );
  assert.equal(typed.class, "22");
  assert.notEqual(typed.detailDigest, facts.detailDigest);
});

// --- Terminal settlement facts ----------------------------------------------

check("terminal-settlement-facts", () => {
  const observer = openObserver();
  const record = observer.beginPoisonAttempt([...SENTINEL_SCHEMA], sentinelRow(), "SELECT 1");
  // No settlement yet: explicitly missing, never a default outcome.
  expectObserverError(() => observer.settlementFacts(record.attempt), "no-conversation", "driver");

  const facts = observer.recordSettlement(record.attempt, "rolled-back");
  assert.equal(facts.attempt, record.attempt);
  assert.equal(facts.outcome, "rolled-back");
  assert.equal(facts.terminal, true);
  assert.deepEqual(Object.keys(facts).sort(), ["attempt", "outcome", "terminal"]);
  assert.equal(observer.settlementFacts(record.attempt).outcome, "rolled-back");

  // Settlement is terminal: a second record throws and the first stands.
  expectObserverError(
    () => observer.recordSettlement(record.attempt, "committed"),
    "connection-busy",
    "driver",
  );
  assert.equal(observer.settlementFacts(record.attempt).outcome, "rolled-back");

  // Unknown outcomes and unknown attempts reject without recording anything.
  const probe = observer.beginPoisonAttempt([...SENTINEL_SCHEMA], sentinelRow(), "SELECT 1");
  for (const bad of ["", "commit", "aborted", "ROLLBACK", null, 0]) {
    expectObserverError(
      () => observer.recordSettlement(probe.attempt, bad),
      "malformed-statement",
      "sql",
    );
  }
  expectObserverError(
    () => observer.recordSettlement("attempt-99", "rolled-back"),
    "unknown-connection",
    "driver",
  );
  expectObserverError(() => observer.settlementFacts(probe.attempt), "no-conversation", "driver");
  expectObserverError(() => observer.settlementFacts("attempt-99"), "unknown-connection", "driver");
});

// --- Fresh raw sentinel reads ------------------------------------------------

check("fresh-raw-sentinel-reads", () => {
  const observer = openObserver();
  const record = observer.beginPoisonAttempt([...SENTINEL_SCHEMA], sentinelRow(), "SELECT 1");
  // No fresh read yet: explicitly missing, never an empty-pass default.
  expectObserverError(() => observer.freshFacts(record.attempt), "no-conversation", "driver");

  const otherRow: readonly unknown[] = [
    { tag: "number", lexeme: "-0" },
    { tag: "text", text: "other" },
  ];
  const fresh = observer.recordFreshRead(record.attempt, [otherRow, sentinelRow()]);
  assert.equal(fresh.attempt, record.attempt);
  assert.equal(fresh.read, "fresh");
  assert.equal(fresh.rowCount, 2);
  assert.deepEqual(
    fresh.rows.map((row) => row.seq),
    [0, 1],
  );
  // Exact cells cross verbatim in stored order: lexeme, code units, seq.
  const second = fresh.rows[1]?.cells as readonly { lexeme?: string; text?: string }[];
  assert.equal(second[0]?.lexeme, "9007199254740993");
  assert.equal(second[1]?.text, "poison-sentinel");

  // Re-reading returns the identical digest: fresh facts are stable.
  const reread = observer.freshFacts(record.attempt);
  assert.equal(reread.digest, fresh.digest);
  assert.deepEqual(
    reread.rows.map((row) => row.digest),
    fresh.rows.map((row) => row.digest),
  );

  // A second fresh read on one attempt throws; the first stands.
  expectObserverError(
    () => observer.recordFreshRead(record.attempt, [sentinelRow()]),
    "connection-busy",
    "driver",
  );
  assert.equal(observer.freshFacts(record.attempt).digest, fresh.digest);

  // Wrong-schema rows reject with the read left unrecorded.
  const bare = openObserver();
  const pin = bare.beginPoisonAttempt([...SENTINEL_SCHEMA], sentinelRow(), "SELECT 1");
  expectObserverError(
    () => bare.recordFreshRead(pin.attempt, [[{ tag: "number", lexeme: "1" }]]),
    "row-arity",
    "sql",
  );
  expectObserverError(
    () =>
      bare.recordFreshRead(pin.attempt, [
        [
          { tag: "text", text: "1" },
          { tag: "text", text: "x" },
        ],
      ]),
    "wrong-cell-type",
    "sql",
  );
  expectObserverError(
    () => bare.recordFreshRead(pin.attempt, [["not-a-cell", { tag: "text", text: "x" }]]),
    "wrong-cell-type",
    "sql",
  );
  // Missing evidence stays missing: nothing was stored by the failures.
  expectObserverError(() => bare.freshFacts(pin.attempt), "no-conversation", "driver");
});

// --- Replay reads confirm the fresh read-out ---------------------------------

check("replay-reads-confirm", () => {
  const observer = openObserver();
  const record = observer.beginPoisonAttempt([...SENTINEL_SCHEMA], sentinelRow(), "SELECT 1");
  // No replay yet: explicitly missing, never an empty-pass default.
  expectObserverError(() => observer.replayFacts(record.attempt), "no-conversation", "driver");

  const fresh = observer.recordFreshRead(record.attempt, [sentinelRow()]);
  const replay = observer.recordReplayRead(record.attempt, [sentinelRow()]);
  assert.equal(replay.attempt, record.attempt);
  assert.equal(replay.read, "replay");
  assert.equal(replay.rowCount, 1);
  assert.equal(replay.digest, fresh.digest);
  const comparison = observer.compareFreshToReplay(fresh, replay);
  assert.equal(comparison.match, true);
  assert.deepEqual(Array.from(comparison.mismatches), []);

  // A second replay on one attempt throws; the first stands.
  expectObserverError(
    () => observer.recordReplayRead(record.attempt, [sentinelRow()]),
    "connection-busy",
    "driver",
  );

  // Seeded divergence: one narrowed lexeme in the replay names its row.
  const drifted = openObserver();
  const drift = drifted.beginPoisonAttempt([...SENTINEL_SCHEMA], sentinelRow(), "SELECT 1");
  const driftFresh = drifted.recordFreshRead(drift.attempt, [
    sentinelRow(),
    [
      { tag: "number", lexeme: "1.10" },
      { tag: "text", text: "pad" },
    ],
  ]);
  const driftReplay = drifted.recordReplayRead(drift.attempt, [
    sentinelRow(),
    [
      { tag: "number", lexeme: "1.1" },
      { tag: "text", text: "pad" },
    ],
  ]);
  assert.notEqual(driftReplay.digest, driftFresh.digest);
  const diverged = drifted.compareFreshToReplay(driftFresh, driftReplay);
  assert.equal(diverged.match, false);
  assert.deepEqual(Array.from(diverged.mismatches), [1]);

  // A row-count gap between fresh and replay mismatches too.
  const short = openObserver();
  const gap = short.beginPoisonAttempt([...SENTINEL_SCHEMA], sentinelRow(), "SELECT 1");
  const gapFresh = short.recordFreshRead(gap.attempt, [sentinelRow()]);
  const gapReplay = short.recordReplayRead(gap.attempt, []);
  assert.deepEqual(Array.from(short.compareFreshToReplay(gapFresh, gapReplay).mismatches), [-1]);

  // Reads from different attempts never compare: cross-attempt verdicts throw.
  const sibling = observer.beginPoisonAttempt([...SENTINEL_SCHEMA], sentinelRow(), "SELECT 1");
  const siblingReplay = observer.recordReplayRead(sibling.attempt, [sentinelRow()]);
  expectObserverError(
    () => observer.compareFreshToReplay(fresh, siblingReplay),
    "malformed-statement",
    "sql",
  );
  expectObserverError(
    () => observer.sentinelPresentIn(sibling.attempt, fresh),
    "malformed-statement",
    "sql",
  );
  expectObserverError(
    () => observer.sentinelPresentIn(record.attempt, siblingReplay),
    "malformed-statement",
    "sql",
  );
});

// --- Omit-poison control reveals the committed sentinel ----------------------

check("omit-poison-control-reveals-committed-sentinel", () => {
  const observer = openObserver();
  const control = observer.beginControlAttempt([...SENTINEL_SCHEMA], sentinelRow());
  assert.equal(control.kind, "control");
  assert.equal(control.poisonDigest, undefined);

  // The control omits the poison write, so no poison error is possible and
  // none is ever recorded: error facts stay explicitly missing.
  expectObserverError(
    () => observer.recordPoisonError(control.attempt, "23505", "duplicate key value"),
    "malformed-statement",
    "sql",
  );
  expectObserverError(() => observer.errorFacts(control.attempt), "no-conversation", "driver");

  // Skipping the poison write still leaves the sentinel visibly committed:
  // fresh and replay reads both reveal the exact sentinel row.
  observer.recordSettlement(control.attempt, "committed");
  const fresh = observer.recordFreshRead(control.attempt, [sentinelRow()]);
  const replay = observer.recordReplayRead(control.attempt, [sentinelRow()]);
  assert.equal(observer.sentinelPresentIn(control.attempt, fresh), true);
  assert.equal(observer.sentinelPresentIn(control.attempt, replay), true);
  assert.equal(observer.compareFreshToReplay(fresh, replay).match, true);
  const agreement = observer.settlementAgreesWithReads(control.attempt);
  assert.equal(agreement.agree, true);
  assert.equal(agreement.reason, "sentinel-present-after-commit");

  // Seeded negative: a committed control whose sentinel is missing from the
  // reads disagrees — the absence is detectable, never a silent pass.
  const lost = openObserver();
  const lostControl = lost.beginControlAttempt([...SENTINEL_SCHEMA], sentinelRow());
  lost.recordSettlement(lostControl.attempt, "committed");
  const lostFresh = lost.recordFreshRead(lostControl.attempt, [
    [
      { tag: "number", lexeme: "1" },
      { tag: "text", text: "stranger" },
    ],
  ]);
  const lostReplay = lost.recordReplayRead(lostControl.attempt, [
    [
      { tag: "number", lexeme: "1" },
      { tag: "text", text: "stranger" },
    ],
  ]);
  assert.equal(lost.sentinelPresentIn(lostControl.attempt, lostFresh), false);
  assert.equal(lost.sentinelPresentIn(lostControl.attempt, lostReplay), false);
  const lostAgreement = lost.settlementAgreesWithReads(lostControl.attempt);
  assert.equal(lostAgreement.agree, false);
  assert.equal(lostAgreement.reason, "sentinel-missing-after-commit");

  // A narrowed lookalike is not the sentinel: presence is bit-exact.
  const near = openObserver();
  const nearControl = near.beginControlAttempt([...SENTINEL_SCHEMA], sentinelRow());
  near.recordSettlement(nearControl.attempt, "committed");
  const nearFresh = near.recordFreshRead(nearControl.attempt, [
    [
      { tag: "number", lexeme: "9007199254740992" },
      { tag: "text", text: "poison-sentinel" },
    ],
  ]);
  assert.equal(near.sentinelPresentIn(nearControl.attempt, nearFresh), false);
});

// --- Callback success never proves commit ------------------------------------

check("callback-success-never-proves-commit", () => {
  const observer = openObserver();
  const record = observer.beginPoisonAttempt([...SENTINEL_SCHEMA], sentinelRow(), "SELECT 1");
  // No callback yet: explicitly missing, never a default claim.
  expectObserverError(() => observer.callbackFacts(record.attempt), "no-conversation", "driver");

  // The callback claims success while every settlement-side fact is missing.
  const report = observer.recordCallbackReport(record.attempt, "success");
  assert.equal(report.attempt, record.attempt);
  assert.equal(report.reported, "success");
  assert.equal(report.proves, "nothing");
  assert.deepEqual(Object.keys(report).sort(), ["attempt", "proves", "reported"]);

  // Success proves nothing: settlement facts stay missing and the agreement
  // verdict throws instead of inferring commit from the callback.
  expectObserverError(() => observer.settlementFacts(record.attempt), "no-conversation", "driver");
  expectObserverError(
    () => observer.settlementAgreesWithReads(record.attempt),
    "no-conversation",
    "driver",
  );

  // Even with both reads recorded and the sentinel present, commit stays
  // unproven until terminal settlement is recorded.
  observer.recordFreshRead(record.attempt, [sentinelRow()]);
  observer.recordReplayRead(record.attempt, [sentinelRow()]);
  expectObserverError(
    () => observer.settlementAgreesWithReads(record.attempt),
    "no-conversation",
    "driver",
  );

  // Terminal settlement decides against the callback: success reported, yet
  // rolled-back recorded — the callback does not override settlement facts.
  observer.recordSettlement(record.attempt, "rolled-back");
  assert.equal(observer.settlementFacts(record.attempt).outcome, "rolled-back");
  const agreement = observer.settlementAgreesWithReads(record.attempt);
  assert.equal(agreement.agree, false);
  assert.equal(agreement.reason, "sentinel-survived-rollback");

  // A second callback report throws; the first stands. Unknown reports reject.
  expectObserverError(
    () => observer.recordCallbackReport(record.attempt, "threw"),
    "connection-busy",
    "driver",
  );
  const probe = observer.beginPoisonAttempt([...SENTINEL_SCHEMA], sentinelRow(), "SELECT 1");
  for (const bad of ["", "ok", "failed", null, 1]) {
    expectObserverError(
      () => observer.recordCallbackReport(probe.attempt, bad),
      "malformed-statement",
      "sql",
    );
  }
  expectObserverError(() => observer.callbackFacts(probe.attempt), "no-conversation", "driver");
  const threw = observer.recordCallbackReport(probe.attempt, "threw");
  assert.equal(threw.proves, "nothing");

  // The boundary is a fixed export naming QD3 as the verdict holder.
  assert.ok(POISON_CALLBACK_NON_PROOF_BOUNDARY.startsWith("non-proof:"));
  assert.ok(POISON_CALLBACK_NON_PROOF_BOUNDARY.includes("never proves commit"));
  assert.ok(POISON_CALLBACK_NON_PROOF_BOUNDARY.includes("QD3"));
  assert.ok(POISON_CALLBACK_NON_PROOF_BOUNDARY.includes("never imply settlement"));
  assert.deepEqual(Array.from(POISON_CREDIT_REQUIREMENTS), [
    "terminal-settlement",
    "fresh-sentinel-read",
    "replay-sentinel-read",
    "qd3-verdict",
  ]);
  assert.equal(POISON_OBSERVER_SCHEMA_VERSION, "1");
});

// --- Rollback success and failure are both witnessed -------------------------

check("rollback-witnessed-by-absent-sentinel", () => {
  const observer = openObserver();
  const record = observer.beginPoisonAttempt(
    [...SENTINEL_SCHEMA],
    sentinelRow(),
    "INSERT INTO t (id) VALUES (1), (1)",
  );
  observer.recordPoisonError(record.attempt, "23505", "duplicate key value");
  observer.recordCallbackReport(record.attempt, "threw");
  observer.recordSettlement(record.attempt, "rolled-back");
  // Rolled back: both reads show rows without the sentinel.
  const stranger: readonly unknown[] = [
    { tag: "number", lexeme: "7" },
    { tag: "text", text: "unrelated" },
  ];
  const fresh = observer.recordFreshRead(record.attempt, [stranger]);
  const replay = observer.recordReplayRead(record.attempt, [stranger]);
  assert.equal(observer.compareFreshToReplay(fresh, replay).match, true);
  assert.equal(observer.sentinelPresentIn(record.attempt, fresh), false);
  const agreement = observer.settlementAgreesWithReads(record.attempt);
  assert.equal(agreement.agree, true);
  assert.equal(agreement.reason, "sentinel-absent-after-rollback");

  // Seeded negative: rolled-back settlement, yet the sentinel survived in
  // both reads — the failed rollback is detectable with its own reason.
  const leaked = openObserver();
  const leak = leaked.beginPoisonAttempt([...SENTINEL_SCHEMA], sentinelRow(), "SELECT 1");
  leaked.recordPoisonError(leak.attempt, "40001", "serialization failure");
  leaked.recordSettlement(leak.attempt, "rolled-back");
  leaked.recordFreshRead(leak.attempt, [sentinelRow()]);
  leaked.recordReplayRead(leak.attempt, [sentinelRow()]);
  const leakAgreement = leaked.settlementAgreesWithReads(leak.attempt);
  assert.equal(leakAgreement.agree, false);
  assert.equal(leakAgreement.reason, "sentinel-survived-rollback");

  // Seeded negative: fresh and replay disagree about the sentinel — the
  // divergence blocks any agreement verdict instead of picking a side.
  const split = openObserver();
  const torn = split.beginPoisonAttempt([...SENTINEL_SCHEMA], sentinelRow(), "SELECT 1");
  split.recordSettlement(torn.attempt, "rolled-back");
  split.recordFreshRead(torn.attempt, []);
  split.recordReplayRead(torn.attempt, [sentinelRow()]);
  const tornAgreement = split.settlementAgreesWithReads(torn.attempt);
  assert.equal(tornAgreement.agree, false);
  assert.equal(tornAgreement.reason, "fresh-replay-diverged");

  // Missing-everything controls: each missing leg throws before any verdict.
  const empty = openObserver();
  const bare = empty.beginPoisonAttempt([...SENTINEL_SCHEMA], sentinelRow(), "SELECT 1");
  expectObserverError(
    () => empty.settlementAgreesWithReads(bare.attempt),
    "no-conversation",
    "driver",
  );
  empty.recordSettlement(bare.attempt, "committed");
  expectObserverError(
    () => empty.settlementAgreesWithReads(bare.attempt),
    "no-conversation",
    "driver",
  );
  empty.recordFreshRead(bare.attempt, [sentinelRow()]);
  expectObserverError(
    () => empty.settlementAgreesWithReads(bare.attempt),
    "no-conversation",
    "driver",
  );
});

// --- Bytes are copies; every failure names its layer -------------------------

check("sentinel-bytes-are-copies", () => {
  const observer = new PoisonObserver(roomyLimits());
  const record = observer.beginPoisonAttempt(
    ["bytes"],
    [{ tag: "bytes", bytes: new Uint8Array([1, 2, 3]) }],
    "SELECT 1",
  );
  const buffer = new Uint8Array([4, 5, 6]);
  const fresh = observer.recordFreshRead(record.attempt, [[{ tag: "bytes", bytes: buffer }]]);
  // Mutating the caller's buffer cannot corrupt the recorded read.
  buffer.fill(0);
  const reread = observer.freshFacts(record.attempt);
  const rereadBytes = reread.rows[0]?.cells[0] as { bytes: Uint8Array };
  assert.deepEqual(Array.from(rereadBytes.bytes), [4, 5, 6]);
  assert.equal(reread.digest, fresh.digest);

  // Mutating a returned facts buffer cannot corrupt later reads either.
  const cell = fresh.rows[0]?.cells[0];
  assert.ok(cell !== undefined && cell.tag === "bytes");
  if (cell.tag === "bytes") {
    cell.bytes[0] = 99;
  }
  const again = observer.freshFacts(record.attempt);
  const againBytes = again.rows[0]?.cells[0] as { bytes: Uint8Array };
  assert.equal(againBytes.bytes[0], 4);
  assert.equal(again.digest, fresh.digest);

  // A bit-exact bytes sentinel is still found by presence probing.
  const sentinel = observer.beginPoisonAttempt(
    ["bytes"],
    [{ tag: "bytes", bytes: new Uint8Array([9, 9]) }],
    "SELECT 1",
  );
  const present = observer.recordFreshRead(sentinel.attempt, [
    [{ tag: "bytes", bytes: [9, 9] }],
    [{ tag: "bytes", bytes: [9, 8] }],
  ]);
  assert.equal(observer.sentinelPresentIn(sentinel.attempt, present), true);
});

check("every-failure-names-its-layer", () => {
  const observer = openObserver();
  // Row and length caps name the engine layer.
  const record = observer.beginPoisonAttempt([...SENTINEL_SCHEMA], sentinelRow(), "SELECT 1");
  const tooMany: readonly (readonly unknown[])[] = Array.from({ length: 17 }, () => sentinelRow());
  expectObserverError(
    () => observer.recordFreshRead(record.attempt, tooMany),
    "capacity-exhausted",
    "engine",
  );
  expectObserverError(
    () => observer.recordReplayRead(record.attempt, tooMany),
    "capacity-exhausted",
    "engine",
  );
  expectObserverError(
    () =>
      observer.beginPoisonAttempt(
        [...SENTINEL_SCHEMA],
        sentinelRow(),
        `SELECT '${"x".repeat(5000)}'`,
      ),
    "capacity-exhausted",
    "engine",
  );
  expectObserverError(
    () =>
      observer.beginPoisonAttempt(
        ["null", "null", "null", "null", "null", "null", "null", "null", "null"],
        sentinelRow(),
        "SELECT 1",
      ),
    "capacity-exhausted",
    "engine",
  );
  // Unknown attempts reject before any row validation.
  expectObserverError(
    () => observer.recordFreshRead("attempt-99", [[{ tag: "text", text: "x".repeat(2000) }]]),
    "unknown-connection",
    "driver",
  );
  const textPin = observer.beginPoisonAttempt(["text"], [{ tag: "text", text: "s" }], "SELECT 1");
  expectObserverError(
    () => observer.recordFreshRead(textPin.attempt, [[{ tag: "text", text: "x".repeat(2000) }]]),
    "capacity-exhausted",
    "engine",
  );
  // Malformed row containers reject without recording anything.
  expectObserverError(
    () =>
      observer.recordReplayRead(textPin.attempt, [
        "not-a-row",
      ] as unknown as readonly (readonly unknown[])[]),
    "row-arity",
    "sql",
  );
  expectObserverError(() => observer.replayFacts(textPin.attempt), "no-conversation", "driver");
});

console.log(
  JSON.stringify({
    kind: "can.db-observer-poison-check",
    schema_version: POISON_OBSERVER_SCHEMA_VERSION,
    checks: passed,
    count: passed.length,
  }),
);
