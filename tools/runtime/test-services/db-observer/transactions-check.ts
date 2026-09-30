// K24 bounded self-check: fresh-pool transaction identity.
//
// Run with: bun tools/runtime/test-services/db-observer/transactions-check.ts
// Local controls only. No databases, drivers, sockets, files, timers, live
// hosts, or live runtimes. Every loop below is bounded by a small constant.
// Live host-dependent controls wait for the qualified profile and Q task.
// QD2 holds D2 credit; nothing here decides rollback expectations.

import { strict as assert } from "node:assert";
import {
  checkLimits,
  DbObserverError,
  type DbObserverCode,
  type DbObserverLayer,
  type DbObserverLimits,
} from "./core.ts";
import {
  TRANSACTION_FACTS_ONLY_BOUNDARY,
  TRANSACTION_MAX_ACTORS,
  TRANSACTION_OBSERVER_SCHEMA_VERSION,
  TRANSACTION_QD2_REQUIREMENTS,
  TransactionObserver,
} from "./transactions.ts";

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

function openObserver(): TransactionObserver {
  return new TransactionObserver(roomyLimits());
}

// --- Callback entry pins one actor to one connection --------------------------

check("callback-entry-pins-actor", () => {
  const observer = openObserver();
  const first = observer.enterCallback("actor-0");
  assert.equal(first.facts.actor, "actor-0");
  assert.equal(first.facts.connection, "conn:actor-0");
  assert.equal(first.facts.entrySeq, 0);
  assert.ok(first.facts.handleDigest.startsWith("sha256:"));
  assert.equal(first.facts.handleDigest.length, "sha256:".length + 64);
  assert.ok(first.token.length > 0);

  // Entry order is observed: later actors take later sequence numbers.
  const second = observer.enterCallback("actor-1");
  assert.equal(second.facts.entrySeq, 1);
  assert.equal(second.facts.connection, "conn:actor-1");
  assert.notEqual(second.facts.handleDigest, first.facts.handleDigest);
  assert.equal(observer.actorCount, 2);

  // Identity facts name the pinned handle and the unsettled state.
  const identity = observer.actorFacts("actor-0");
  assert.equal(identity.actor, "actor-0");
  assert.equal(identity.connection, "conn:actor-0");
  assert.equal(identity.handleDigest, first.facts.handleDigest);
  assert.equal(identity.entrySeq, 0);
  assert.equal(identity.settled, false);
  assert.equal(identity.released, false);

  // Re-entry is refused: one actor carries one callback entry.
  expectObserverError(() => observer.enterCallback("actor-0"), "connection-busy", "driver");
  // Malformed actor labels reject at the sql layer with nothing stored.
  expectObserverError(() => observer.enterCallback(""), "malformed-statement", "sql");
  expectObserverError(() => observer.enterCallback("bad actor!"), "malformed-statement", "sql");
  expectObserverError(() => observer.actorFacts("ghost"), "unknown-connection", "driver");
  assert.equal(observer.actorCount, 2);

  // The actor table is bounded at eight.
  assert.equal(TRANSACTION_MAX_ACTORS, 8);
  const tight = openObserver();
  for (let index = 0; index < TRANSACTION_MAX_ACTORS; index++) {
    tight.enterCallback(`a-${index}`);
  }
  expectObserverError(() => tight.enterCallback("a-overflow"), "capacity-exhausted", "engine");
});

// --- One bounded eight-actor control, no warmup --------------------------------

check("eight-actor-fresh-pool-no-warmup", () => {
  // Fresh pool: the first call is actor entry, with no warmup round. Eight
  // actors enter, record an id, settle rolled-back, and release.
  const observer = openObserver();
  const tokens = new Map<string, string>();
  for (let index = 0; index < 8; index++) {
    const actor = `w-${index}`;
    const entered = observer.enterCallback(actor);
    assert.equal(entered.facts.entrySeq, index);
    tokens.set(actor, entered.token);
  }
  assert.equal(observer.actorCount, 8);
  for (let index = 0; index < 8; index++) {
    const actor = `w-${index}`;
    const token = tokens.get(actor) as string;
    const id = observer.recordLastInsertId(actor, token, {
      tag: "number",
      lexeme: `${100 + index}`,
    });
    assert.equal(id.actor, actor);
    const settlement = observer.recordSettlement(actor, token, "rolled-back", "postgres");
    assert.equal(settlement.outcome, "rolled-back");
    const ack = observer.release(actor, token);
    assert.equal(ack.released, true);
    assert.equal(ack.actor, actor);
  }
  // Every actor is settled and released; rereads are stable.
  for (let index = 0; index < 8; index++) {
    const actor = `w-${index}`;
    const identity = observer.actorFacts(actor);
    assert.equal(identity.settled, true);
    assert.equal(identity.released, true);
    assert.equal(observer.settlementFacts(actor).outcome, "rolled-back");
    assert.equal(observer.releaseAck(actor).released, true);
  }
});

// --- Swapped actor identity is detectable --------------------------------------

check("swapped-actor-detectable", () => {
  const observer = openObserver();
  const a = observer.enterCallback("alpha");
  const b = observer.enterCallback("beta");

  // A swapped token is never authority: alpha's token on beta's handle fails.
  expectObserverError(
    () => observer.recordLastInsertId("beta", a.token, { tag: "number", lexeme: "1" }),
    "forged-token",
    "driver",
  );
  expectObserverError(
    () => observer.recordSettlement("beta", a.token, "rolled-back", "postgres"),
    "forged-token",
    "driver",
  );
  expectObserverError(() => observer.release("beta", a.token), "forged-token", "driver");
  expectObserverError(
    () => observer.recordLastInsertId("alpha", "invented-token", { tag: "number", lexeme: "1" }),
    "forged-token",
    "driver",
  );
  void b;

  // Both actors record the SAME id value on their own handles.
  const same = { tag: "number", lexeme: "42" };
  const storedA = observer.recordLastInsertId("alpha", a.token, same);
  assert.equal(storedA.actor, "alpha");

  // A claim carrying alpha's exact value but beta's identity still mismatches:
  // identity is compared independently of value.
  const swapped = observer.compareLastInsertId(storedA, {
    actor: "beta",
    connection: "conn:beta",
    id: { tag: "number", lexeme: "42" },
  });
  assert.equal(swapped.match, false);
  assert.deepEqual(Array.from(swapped.mismatches), [-1]);

  // The correctly bound claim matches.
  const bound = observer.compareLastInsertId(storedA, {
    actor: "alpha",
    connection: "conn:alpha",
    id: { tag: "number", lexeme: "42" },
  });
  assert.equal(bound.match, true);
  assert.deepEqual(Array.from(bound.mismatches), []);
});

// --- Wrong-handle LAST_INSERT_ID is detectable ---------------------------------

check("wrong-handle-last-insert-id-detectable", () => {
  const observer = openObserver();
  const a = observer.enterCallback("h-0");
  const c = observer.enterCallback("h-1");

  // Missing evidence stays missing: no id recorded yet.
  expectObserverError(() => observer.lastInsertIdFacts("h-0"), "no-conversation", "driver");

  const stored = observer.recordLastInsertId("h-0", a.token, { tag: "number", lexeme: "7" });
  assert.equal(stored.connection, "conn:h-0");
  assert.ok(stored.digest.startsWith("sha256:"));
  // Re-reading returns the identical digest: id facts are stable.
  assert.equal(observer.lastInsertIdFacts("h-0").digest, stored.digest);
  assert.equal((observer.lastInsertIdFacts("h-0").id as { lexeme: string }).lexeme, "7");

  // Wrong handle, coincidentally correct value: identity mismatch, not a pass.
  const wrongHandle = observer.compareLastInsertId(stored, {
    actor: "h-1",
    connection: "conn:h-1",
    id: { tag: "number", lexeme: "7" },
  });
  assert.equal(wrongHandle.match, false);
  assert.deepEqual(Array.from(wrongHandle.mismatches), [-1]);

  // Right handle, wrong value: value mismatch.
  const wrongValue = observer.compareLastInsertId(stored, {
    actor: "h-0",
    connection: "conn:h-0",
    id: { tag: "number", lexeme: "8" },
  });
  assert.equal(wrongValue.match, false);
  assert.deepEqual(Array.from(wrongValue.mismatches), [0]);

  // Wrong handle AND wrong value: both mismatches appear.
  const both = observer.compareLastInsertId(stored, {
    actor: "h-1",
    connection: "conn:h-1",
    id: { tag: "number", lexeme: "8" },
  });
  assert.equal(both.match, false);
  assert.deepEqual(Array.from(both.mismatches), [-1, 0]);

  // A mistyped claim cell is a value mismatch, never a throw.
  const mistyped = observer.compareLastInsertId(stored, {
    actor: "h-0",
    connection: "conn:h-0",
    id: { tag: "text", text: "7" },
  });
  assert.equal(mistyped.match, false);
  assert.deepEqual(Array.from(mistyped.mismatches), [0]);

  // Id cells are exact: narrowing and -0/0 confusion mismatch.
  const big = observer.recordLastInsertId("h-1", c.token, {
    tag: "number",
    lexeme: "9007199254740993",
  });
  assert.equal(
    observer.compareLastInsertId(big, {
      actor: "h-1",
      connection: "conn:h-1",
      id: { tag: "number", lexeme: "9007199254740992" },
    }).match,
    false,
  );
  assert.equal(
    observer.compareLastInsertId(big, {
      actor: "h-1",
      connection: "conn:h-1",
      id: { tag: "number", lexeme: "9007199254740993" },
    }).match,
    true,
  );

  // Bytes id cells cross as copies: caller mutation cannot corrupt the record.
  const buffer = new Uint8Array([1, 2, 3]);
  observer.recordLastInsertId("h-0", a.token, { tag: "bytes", bytes: buffer });
  buffer.fill(0);
  const reread = observer.lastInsertIdFacts("h-0");
  assert.deepEqual(Array.from((reread.id as { bytes: Uint8Array }).bytes), [1, 2, 3]);
});

// --- Settled rollback facts are recorded, never decided ------------------------

check("settled-rollback-facts-only", () => {
  const observer = openObserver();
  const entered = observer.enterCallback("s-0");

  // Missing settlement stays missing: unsettled actors throw, never default.
  expectObserverError(() => observer.settlementFacts("s-0"), "no-conversation", "driver");

  // Unknown engines and outcomes reject; nothing is stored by the failures.
  expectObserverError(
    () => observer.recordSettlement("s-0", entered.token, "rolled-back", "oracle"),
    "malformed-statement",
    "sql",
  );
  expectObserverError(
    () => observer.recordSettlement("s-0", entered.token, "maybe", "postgres"),
    "malformed-statement",
    "sql",
  );
  expectObserverError(() => observer.settlementFacts("s-0"), "no-conversation", "driver");

  // The settled rollback fact records outcome plus engine under one digest.
  const settled = observer.recordSettlement("s-0", entered.token, "rolled-back", "postgres");
  assert.equal(settled.actor, "s-0");
  assert.equal(settled.connection, "conn:s-0");
  assert.equal(settled.outcome, "rolled-back");
  assert.equal(settled.engine, "postgres");
  assert.ok(settled.digest.startsWith("sha256:"));
  assert.equal(observer.settlementFacts("s-0").digest, settled.digest);
  assert.equal(observer.actorFacts("s-0").settled, true);

  // Settling twice is refused; the first fact stands.
  expectObserverError(
    () => observer.recordSettlement("s-0", entered.token, "committed", "postgres"),
    "connection-busy",
    "driver",
  );
  assert.equal(observer.settlementFacts("s-0").outcome, "rolled-back");

  // No lifecycle step runs after settlement except release.
  expectObserverError(
    () => observer.recordLastInsertId("s-0", entered.token, { tag: "number", lexeme: "1" }),
    "no-conversation",
    "driver",
  );

  // Facts only, decided at QD2: the boundary names no expectation API.
  assert.ok(TRANSACTION_FACTS_ONLY_BOUNDARY.startsWith("facts-only:"));
  assert.ok(TRANSACTION_FACTS_ONLY_BOUNDARY.includes("never decides rollback expectations"));
  assert.ok(TRANSACTION_FACTS_ONLY_BOUNDARY.includes("QD2"));
  assert.ok(
    TRANSACTION_FACTS_ONLY_BOUNDARY.includes("never") ||
      TRANSACTION_FACTS_ONLY_BOUNDARY.includes("nothing here"),
  );
  assert.deepEqual(Array.from(TRANSACTION_QD2_REQUIREMENTS), [
    "c-side-witness",
    "settlement-agreement",
    "qd2-verdict",
  ]);
  assert.equal(TRANSACTION_OBSERVER_SCHEMA_VERSION, "1");
  assert.equal((observer as unknown as Record<string, unknown>)["expectRollback"], undefined);
});

// --- Engine-specific settlement observations -----------------------------------

check("engine-specific-settlement", () => {
  const observer = openObserver();
  const engines = ["postgres", "sqlite", "mysql"] as const;
  const digests = new Set<string>();
  for (let index = 0; index < engines.length; index++) {
    const actor = `e-${index}`;
    const entered = observer.enterCallback(actor);
    const settlement = observer.recordSettlement(actor, entered.token, "committed", engines[index]);
    assert.equal(settlement.engine, engines[index]);
    assert.equal(settlement.outcome, "committed");
    // The same outcome on different engines yields distinct observations.
    digests.add(settlement.digest);
  }
  assert.equal(digests.size, engines.length);

  // "unknown" is an explicit recorded outcome, not a gap.
  const entered = observer.enterCallback("e-unknown");
  const unknown = observer.recordSettlement("e-unknown", entered.token, "unknown", "sqlite");
  assert.equal(unknown.outcome, "unknown");
  assert.equal(unknown.engine, "sqlite");
  assert.equal(observer.settlementFacts("e-unknown").digest, unknown.digest);
});

// --- Release acknowledgment ----------------------------------------------------

check("release-acknowledgment", () => {
  const observer = openObserver();
  const entered = observer.enterCallback("r-0");

  // Release before settlement is refused: unsettled work is never dropped.
  expectObserverError(() => observer.release("r-0", entered.token), "connection-busy", "driver");
  expectObserverError(() => observer.releaseAck("r-0"), "no-conversation", "driver");

  observer.recordSettlement("r-0", entered.token, "committed", "mysql");
  const ack = observer.release("r-0", entered.token);
  assert.equal(ack.actor, "r-0");
  assert.equal(ack.connection, "conn:r-0");
  assert.equal(ack.released, true);
  assert.ok(ack.ackDigest.startsWith("sha256:"));
  assert.deepEqual(Object.keys(ack).sort(), ["ackDigest", "actor", "connection", "released"]);

  // The acknowledgment is stable and re-readable; releasing twice fails.
  assert.equal(observer.releaseAck("r-0").ackDigest, ack.ackDigest);
  assert.equal(observer.actorFacts("r-0").released, true);
  expectObserverError(() => observer.release("r-0", entered.token), "closed-handle", "engine");

  // Nothing runs on a released handle.
  expectObserverError(
    () => observer.recordLastInsertId("r-0", entered.token, { tag: "null" }),
    "closed-handle",
    "engine",
  );
  expectObserverError(
    () => observer.recordSettlement("r-0", entered.token, "committed", "mysql"),
    "closed-handle",
    "engine",
  );
});

// --- Every failure names its layer ---------------------------------------------

check("every-failure-names-its-layer", () => {
  const observer = openObserver();
  // Driver: conversation discipline and token authority.
  expectObserverError(() => observer.enterCallback(""), "malformed-statement", "sql");
  const entered = observer.enterCallback("l-0");
  expectObserverError(() => observer.enterCallback("l-0"), "connection-busy", "driver");
  expectObserverError(
    () => observer.recordLastInsertId("l-0", "wrong", { tag: "null" }),
    "forged-token",
    "driver",
  );
  expectObserverError(() => observer.lastInsertIdFacts("l-0"), "no-conversation", "driver");
  // SQL: cell shape and statement vocabulary.
  expectObserverError(
    () => observer.recordLastInsertId("l-0", entered.token, "not-a-cell"),
    "wrong-cell-type",
    "sql",
  );
  expectObserverError(
    () => observer.recordLastInsertId("l-0", entered.token, { tag: "number", lexeme: "1x" }),
    "malformed-lexeme",
    "sql",
  );
  // Engine: capacity and lifecycle.
  expectObserverError(
    () =>
      observer.recordLastInsertId("l-0", entered.token, { tag: "text", text: "x".repeat(2000) }),
    "capacity-exhausted",
    "engine",
  );
  observer.recordSettlement("l-0", entered.token, "rolled-back", "sqlite");
  observer.release("l-0", entered.token);
  expectObserverError(() => observer.release("l-0", entered.token), "closed-handle", "engine");
});

console.log(
  JSON.stringify({
    kind: "can.db-observer-transactions-check",
    schema_version: TRANSACTION_OBSERVER_SCHEMA_VERSION,
    checks: passed,
    count: passed.length,
  }),
);
