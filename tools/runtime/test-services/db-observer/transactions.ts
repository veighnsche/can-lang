// K24: fresh-pool transaction identity observations.
//
// An INDEPENDENT raw observer over in-memory doubles only: no real database,
// no driver, no sockets, no I/O. Over one fresh pool (no warmup: the first
// actor entry goes straight to facts), this module records callback entry,
// pinned-connection actor identity, per-handle LAST_INSERT_ID, settled
// rollback facts, engine-specific settlement observations, and release
// acknowledgments. Every failure reuses the K22 core's layered
// DbObserverError (engine/driver/sql); typed exact id cells reuse the core's
// cell constructors.
//
// Facts-only boundary (normative):
//   This module records settlement facts only. It never decides rollback
//   expectations: there is no expectRollback, no commit verdict, and no path
//   from "settlement recorded" to "rollback proven". QD2 owns the D2 verdict
//   and reads these facts; see TRANSACTION_FACTS_ONLY_BOUNDARY.
//
// Independence scope from the Can adapter under test (normative):
//   SHARES: nothing executable. The observer never imports, calls, or reads
//   adapter code, adapter-decoded values, or adapter connection state. The
//   only strings that may coincide are the actor/connection labels under
//   test, supplied by the test, never by the adapter.
//   PROVES INDEPENDENTLY: callback entry order, pinned actor identity,
//   per-handle LAST_INSERT_ID binding, settled outcomes, engine tags, and
//   release acknowledgments from its own doubles alone. Adapter claims are
//   compared AGAINST these facts; the observer never derives a fact FROM an
//   adapter claim. Consequently a swapped actor and a wrong-handle
//   LAST_INSERT_ID are both detectable even when the carried values are
//   coincidentally correct.
//
// Pure in-memory mechanics for the bounded self-check: no I/O, timers,
// transports, services, or live runtimes. Local controls only.

import { createHash, randomBytes } from "node:crypto";
import {
  DbObserverError,
  makeBytesCell,
  makeNullCell,
  makeNumberCell,
  makeTextCell,
  type DbCell,
  type DbObserverLimits,
} from "./core.ts";

export const TRANSACTION_OBSERVER_SCHEMA_VERSION = "1" as const;

// Recorded facts-only boundary: settlement facts here, rollback verdict at QD2.
export const TRANSACTION_FACTS_ONLY_BOUNDARY: string =
  "facts-only: this module records callback entry, actor identity, LAST_INSERT_ID, " +
  "settlement, and release facts only; it never decides rollback expectations — " +
  "the D2 verdict belongs to QD2, which reads these facts, and nothing here " +
  "claims execution credit";

// What QD2 still needs beyond this module's raw facts.
export const TRANSACTION_QD2_REQUIREMENTS: readonly string[] = Object.freeze([
  "c-side-witness",
  "settlement-agreement",
  "qd2-verdict",
]);

// One bounded fresh-pool control covers exactly eight actors with no warmup.
export const TRANSACTION_MAX_ACTORS = 8;

export const TRANSACTION_ENGINES = ["postgres", "sqlite", "mysql"] as const;
export type TransactionEngine = (typeof TRANSACTION_ENGINES)[number];

// Settled outcomes are recorded facts, never expectations: "unknown" is an
// explicit unsettled-visible outcome, not a gap.
export const TRANSACTION_OUTCOMES = ["committed", "rolled-back", "unknown"] as const;
export type TransactionOutcome = (typeof TRANSACTION_OUTCOMES)[number];

const MAX_ACTOR_LEN = 128;

function checkActor(value: unknown): string {
  if (typeof value !== "string" || value === "" || value.length > MAX_ACTOR_LEN) {
    throw new DbObserverError(
      "malformed-statement",
      `actor must be a non-empty label of at most ${MAX_ACTOR_LEN} chars`,
    );
  }
  if (!/^[A-Za-z0-9][A-Za-z0-9_.:-]*$/.test(value)) {
    throw new DbObserverError("malformed-statement", `actor carries illegal characters: ${value}`);
  }
  return value;
}

function checkEngine(value: unknown): TransactionEngine {
  if (!(TRANSACTION_ENGINES as readonly unknown[]).includes(value)) {
    throw new DbObserverError("malformed-statement", `unknown engine: ${String(value)}`);
  }
  return value as TransactionEngine;
}

function checkOutcome(value: unknown): TransactionOutcome {
  if (!(TRANSACTION_OUTCOMES as readonly unknown[]).includes(value)) {
    throw new DbObserverError(
      "malformed-statement",
      `unknown settlement outcome: ${String(value)}`,
    );
  }
  return value as TransactionOutcome;
}

function checkIdCell(value: unknown, maxCellBytes: number): DbCell {
  if (value === null || typeof value !== "object" || Array.isArray(value)) {
    throw new DbObserverError("wrong-cell-type", "LAST_INSERT_ID must be a tagged cell");
  }
  const record = value as Record<string, unknown>;
  switch (record["tag"]) {
    case "number":
      return makeNumberCell(record["lexeme"]);
    case "text": {
      const cell = makeTextCell(record["text"]);
      if ((cell as { text: string }).text.length > maxCellBytes) {
        throw new DbObserverError("capacity-exhausted", "id cell exceeds the byte cap");
      }
      return cell;
    }
    case "bytes": {
      const raw = record["bytes"];
      const bytes =
        raw instanceof Uint8Array
          ? raw
          : Array.isArray(raw) && raw.every((b) => Number.isInteger(b) && b >= 0 && b <= 255)
            ? Uint8Array.from(raw as number[])
            : (() => {
                throw new DbObserverError(
                  "wrong-cell-type",
                  "id cell needs a Uint8Array or byte array",
                );
              })();
      if (bytes.length > maxCellBytes) {
        throw new DbObserverError("capacity-exhausted", "id cell exceeds the byte cap");
      }
      return makeBytesCell(bytes);
    }
    case "null":
      return makeNullCell();
    default:
      throw new DbObserverError("wrong-cell-type", `unknown id cell tag: ${String(record["tag"])}`);
  }
}

function cellDigest(cell: DbCell): string {
  const hash = createHash("sha256");
  hash.update(cell.tag, "utf8");
  hash.update("|", "utf8");
  switch (cell.tag) {
    case "number":
      hash.update(cell.lexeme, "utf8");
      break;
    case "text":
      hash.update(cell.text, "utf8");
      break;
    case "bytes":
      hash.update(cell.bytes);
      break;
    case "null":
      hash.update("null", "utf8");
      break;
  }
  return `sha256:${hash.digest("hex")}`;
}

function copyCell(cell: DbCell): DbCell {
  if (cell.tag !== "bytes") return cell;
  return Object.freeze({ tag: "bytes", bytes: Uint8Array.from(cell.bytes) }) as DbCell;
}

function cellsEqual(a: DbCell, b: DbCell): boolean {
  if (a.tag !== b.tag) return false;
  switch (a.tag) {
    case "number":
      return a.lexeme === (b as { lexeme: string }).lexeme;
    case "text":
      return a.text === (b as { text: string }).text;
    case "bytes": {
      const x = a.bytes;
      const y = (b as { bytes: Uint8Array }).bytes;
      if (x.length !== y.length) return false;
      for (let i = 0; i < x.length; i++) {
        if (x[i] !== y[i]) return false;
      }
      return true;
    }
    case "null":
      return true;
  }
}

// ---------------------------------------------------------------------------
// Facts
// ---------------------------------------------------------------------------

export type CallbackEntryFacts = Readonly<{
  actor: string;
  connection: string;
  entrySeq: number;
  handleDigest: string;
}>;

export type ActorIdentityFacts = Readonly<{
  actor: string;
  connection: string;
  handleDigest: string;
  entrySeq: number;
  settled: boolean;
  released: boolean;
}>;

export type LastInsertIdFacts = Readonly<{
  actor: string;
  connection: string;
  handleDigest: string;
  id: DbCell;
  digest: string;
}>;

export type SettlementFacts = Readonly<{
  actor: string;
  connection: string;
  outcome: TransactionOutcome;
  engine: TransactionEngine;
  digest: string;
}>;

export type ReleaseAck = Readonly<{
  actor: string;
  connection: string;
  released: true;
  ackDigest: string;
}>;

// Identity comparison: -1 names an identity (actor/connection/handle)
// mismatch, 0 names a value mismatch. Identity is checked independently of
// value, so a swapped actor or a wrong-handle LAST_INSERT_ID mismatches even
// when the carried value is coincidentally correct.
export type IdentityComparison = Readonly<{
  match: boolean;
  mismatches: readonly number[];
}>;

export type LastInsertIdClaim = Readonly<{
  actor: string;
  connection: string;
  id: unknown;
}>;

type ActorEntry = {
  actor: string;
  connection: string;
  token: string;
  entrySeq: number;
  lastInsertId: DbCell | undefined;
  settlement: { outcome: TransactionOutcome; engine: TransactionEngine } | undefined;
  released: boolean;
};

function digestToken(token: string): string {
  return `sha256:${createHash("sha256").update(token, "utf8").digest("hex")}`;
}

// ---------------------------------------------------------------------------
// Observer: callback entry, actor identity, LAST_INSERT_ID, settlement, ack
// ---------------------------------------------------------------------------

export class TransactionObserver {
  private readonly limits: DbObserverLimits;
  private readonly actors = new Map<string, ActorEntry>();
  private nextEntrySeq = 0;

  constructor(limits: DbObserverLimits) {
    this.limits = limits;
  }

  get actorCount(): number {
    return this.actors.size;
  }

  // Observe one actor entering its transaction callback on the fresh pool.
  // Entry pins the actor to exactly one connection: the connection label is
  // derived from the actor (`conn:<actor>`) so two actors can never share a
  // handle. The token is minted here and returned alongside the facts exactly
  // once; every later call must present it, so a swapped actor presenting a
  // foreign token fails with forged-token. No warmup: the first entry lands
  // directly in the facts table.
  enterCallback(actor: unknown): Readonly<{ facts: CallbackEntryFacts; token: string }> {
    const name = checkActor(actor);
    if (this.actors.has(name)) {
      throw new DbObserverError("connection-busy", `actor already entered: ${name}`);
    }
    if (this.actors.size >= TRANSACTION_MAX_ACTORS) {
      throw new DbObserverError("capacity-exhausted", "actor table full");
    }
    const token = `txn-${randomBytes(16).toString("hex")}`;
    const entry: ActorEntry = {
      actor: name,
      connection: `conn:${name}`,
      token,
      entrySeq: this.nextEntrySeq,
      lastInsertId: undefined,
      settlement: undefined,
      released: false,
    };
    this.nextEntrySeq += 1;
    this.actors.set(name, entry);
    return Object.freeze({
      facts: Object.freeze({
        actor: entry.actor,
        connection: entry.connection,
        entrySeq: entry.entrySeq,
        handleDigest: digestToken(token),
      }),
      token,
    });
  }

  private requireEntry(actor: string): ActorEntry {
    const entry = this.actors.get(actor);
    if (entry === undefined) {
      throw new DbObserverError("unknown-connection", `actor never entered: ${actor}`);
    }
    return entry;
  }

  private requireToken(entry: ActorEntry, token: string): void {
    if (token === "" || token !== entry.token) {
      throw new DbObserverError("forged-token", "actor token is not the pinned token");
    }
  }

  actorFacts(actor: unknown): ActorIdentityFacts {
    const entry = this.requireEntry(checkActor(actor));
    return Object.freeze({
      actor: entry.actor,
      connection: entry.connection,
      handleDigest: digestToken(entry.token),
      entrySeq: entry.entrySeq,
      settled: entry.settlement !== undefined,
      released: entry.released,
    });
  }

  // Record the LAST_INSERT_ID visible on the actor's own pinned handle. The
  // id cell is stored bound to the handle: read-out always names the actor,
  // connection, and handle digest alongside the value, so a wrong-handle
  // read is detectable by identity even when the value matches.
  recordLastInsertId(actor: unknown, token: string, id: unknown): LastInsertIdFacts {
    const entry = this.requireEntry(checkActor(actor));
    this.requireToken(entry, token);
    if (entry.released) {
      throw new DbObserverError("closed-handle", "handle already released");
    }
    if (entry.settlement !== undefined) {
      throw new DbObserverError("no-conversation", "transaction already settled");
    }
    entry.lastInsertId = copyCell(checkIdCell(id, this.limits.maxCellBytes));
    return this.lastInsertIdFacts(entry.actor);
  }

  lastInsertIdFacts(actor: unknown): LastInsertIdFacts {
    const entry = this.requireEntry(checkActor(actor));
    if (entry.lastInsertId === undefined) {
      throw new DbObserverError("no-conversation", `no LAST_INSERT_ID recorded for ${entry.actor}`);
    }
    const id = copyCell(entry.lastInsertId);
    return Object.freeze({
      actor: entry.actor,
      connection: entry.connection,
      handleDigest: digestToken(entry.token),
      id,
      digest: cellDigest(id),
    });
  }

  // Compare one claimed LAST_INSERT_ID against the observer's bound facts.
  // Identity (actor/connection) is compared independently of the id value:
  // a claim from the wrong actor or the wrong handle mismatches with [-1]
  // even when the id value is coincidentally correct, and a wrong value
  // mismatches with [0]. Both can appear together.
  compareLastInsertId(stored: LastInsertIdFacts, claimed: LastInsertIdClaim): IdentityComparison {
    const mismatches: number[] = [];
    if (claimed.actor !== stored.actor || claimed.connection !== stored.connection) {
      mismatches.push(-1);
    }
    let exact: DbCell | undefined;
    try {
      exact = checkIdCell(claimed.id, this.limits.maxCellBytes);
    } catch {
      exact = undefined;
    }
    if (exact === undefined || !cellsEqual(stored.id, exact)) {
      mismatches.push(0);
    }
    return Object.freeze({ match: mismatches.length === 0, mismatches: Object.freeze(mismatches) });
  }

  // Record the settled outcome of one actor's transaction as observed on the
  // given engine. Facts only: recording a settlement states what the engine
  // showed, never what the transaction should have done. There is no
  // expectation argument and no verdict return.
  recordSettlement(
    actor: unknown,
    token: string,
    outcome: unknown,
    engine: unknown,
  ): SettlementFacts {
    const entry = this.requireEntry(checkActor(actor));
    this.requireToken(entry, token);
    if (entry.released) {
      throw new DbObserverError("closed-handle", "handle already released");
    }
    if (entry.settlement !== undefined) {
      throw new DbObserverError("connection-busy", "transaction already settled");
    }
    entry.settlement = { outcome: checkOutcome(outcome), engine: checkEngine(engine) };
    return this.settlementFacts(entry.actor);
  }

  // The engine-specific settlement observation: outcome plus the engine tag
  // it was observed on, under one digest. Missing settlement stays missing:
  // unsettled actors throw rather than yielding a default.
  settlementFacts(actor: unknown): SettlementFacts {
    const entry = this.requireEntry(checkActor(actor));
    if (entry.settlement === undefined) {
      throw new DbObserverError("no-conversation", `no settlement recorded for ${entry.actor}`);
    }
    const { outcome, engine } = entry.settlement;
    return Object.freeze({
      actor: entry.actor,
      connection: entry.connection,
      outcome,
      engine,
      digest: `sha256:${createHash("sha256")
        .update(`${entry.actor}|${entry.connection}|${outcome}|${engine}`, "utf8")
        .digest("hex")}`,
    });
  }

  // Release the actor's pinned handle after settlement. Release before
  // settlement is refused so an unsettled transaction can never be silently
  // dropped; the acknowledgment names the released handle.
  release(actor: unknown, token: string): ReleaseAck {
    const entry = this.requireEntry(checkActor(actor));
    this.requireToken(entry, token);
    if (entry.released) {
      throw new DbObserverError("closed-handle", "handle already released");
    }
    if (entry.settlement === undefined) {
      throw new DbObserverError("connection-busy", "settle before release");
    }
    entry.released = true;
    return this.releaseAck(entry.actor);
  }

  releaseAck(actor: unknown): ReleaseAck {
    const entry = this.requireEntry(checkActor(actor));
    if (!entry.released) {
      throw new DbObserverError("no-conversation", `no release recorded for ${entry.actor}`);
    }
    return Object.freeze({
      actor: entry.actor,
      connection: entry.connection,
      released: true as const,
      ackDigest: `sha256:${createHash("sha256")
        .update(`release|${entry.actor}|${entry.connection}`, "utf8")
        .digest("hex")}`,
    });
  }
}
