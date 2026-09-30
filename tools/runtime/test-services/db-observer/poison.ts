// K25: poisoned transaction sentinel observations.
//
// An INDEPENDENT raw observer over in-memory doubles only: no real database,
// no driver, no sockets, no I/O. Beside one poisoned transaction attempt (an
// opaque test-supplied identity that is never executed here), this module
// records four raw facts on the observer's own path: the PG error-code
// observation (SQLSTATE code plus class), the terminal settlement outcome,
// the fresh raw sentinel read, and the replay sentinel read. Sentinel rows
// carry typed exact cells (number/text/bytes/null) in stored order, mirroring
// the K22 core, and every failure reuses the core's layered DbObserverError
// (engine/driver/sql).
//
// Non-proof boundary (normative):
//   A transaction callback reporting success can NEVER prove commit. Commit
//   or rollback is proven by terminal settlement facts plus fresh/replay
//   sentinel agreement only, and the QD3 verdict belongs to QD3. Callback
//   reports are recorded as inert labels with a fixed proves:"nothing" tag;
//   no helper here derives settlement from a callback. See
//   POISON_CALLBACK_NON_PROOF_BOUNDARY.
//
// Independence scope from the Can adapter under test (normative):
//   SHARES: nothing executable. The observer never imports, calls, or reads
//   adapter code, adapter-decoded values, or adapter connection state. The
//   only strings that may coincide are the attempt/sentinel labels under
//   test, supplied by the test, never by the adapter.
//   PROVES INDEPENDENTLY: PG error code/class, terminal settlement, exact
//   sentinel cells in fresh and replay reads, read order, and layered error
//   provenance from its own seeded doubles alone. Adapter claims are compared
//   AGAINST these facts; the observer never derives a fact FROM an adapter
//   claim. Consequently a coincidentally successful callback still proves
//   nothing when terminal settlement is missing.
//
// Pure in-memory mechanics for the bounded self-check: no I/O, timers,
// transports, services, or live runtimes. Local controls only.

import { createHash } from "node:crypto";
import {
  DbObserverError,
  DB_CELL_TAGS,
  makeBytesCell,
  makeNullCell,
  makeNumberCell,
  makeTextCell,
  type DbCell,
  type DbCellTag,
  type DbObserverLimits,
} from "./core.ts";

export const POISON_OBSERVER_SCHEMA_VERSION = "1" as const;

// Recorded non-proof boundary: callback success never proves commit.
export const POISON_CALLBACK_NON_PROOF_BOUNDARY: string =
  "non-proof: a transaction callback reporting success never proves commit; " +
  "commit or rollback is proven by terminal settlement facts plus fresh/replay " +
  "sentinel agreement, decided at QD3 from observer facts — callback reports " +
  "are inert labels and never imply settlement";

// What QD3 still needs beyond this module's raw facts. Listed so a reader can
// see the gap between "settlement recorded" and "rollback witnessed".
export const POISON_CREDIT_REQUIREMENTS: readonly string[] = Object.freeze([
  "terminal-settlement",
  "fresh-sentinel-read",
  "replay-sentinel-read",
  "qd3-verdict",
]);

export const POISON_SETTLEMENT_OUTCOMES = ["rolled-back", "committed"] as const;
export type PoisonSettlementOutcome = (typeof POISON_SETTLEMENT_OUTCOMES)[number];

export const POISON_CALLBACK_REPORTS = ["success", "threw"] as const;
export type PoisonCallbackReport = (typeof POISON_CALLBACK_REPORTS)[number];

export const POISON_MAX_ATTEMPTS = 8;
const MAX_STATEMENT_LEN = 4096;
const MAX_DETAIL_LEN = 512;

// Canonical SQLSTATE shape: five uppercase alphanumerics. Class "00" (success)
// is rejected for poison errors: a poison write that "succeeds" is not an
// error observation.
const PG_CODE_PATTERN = /^[0-9A-Z]{5}$/;

function checkAttemptId(value: unknown): string {
  if (typeof value !== "string" || !/^attempt-[0-9]+$/.test(value)) {
    throw new DbObserverError("malformed-statement", `not an attempt id: ${String(value)}`);
  }
  return value;
}

function checkPoisonStatement(value: unknown): string {
  if (typeof value !== "string" || value === "") {
    throw new DbObserverError("malformed-statement", "poison statement must be a non-empty string");
  }
  if (value.length > MAX_STATEMENT_LEN) {
    throw new DbObserverError("capacity-exhausted", "poison statement exceeds the length cap");
  }
  return value;
}

function checkPgCode(value: unknown): string {
  if (typeof value !== "string" || !PG_CODE_PATTERN.test(value)) {
    throw new DbObserverError(
      "malformed-statement",
      `not a PG error code (want five SQLSTATE chars): ${String(value)}`,
    );
  }
  if (value.startsWith("00")) {
    throw new DbObserverError("malformed-statement", `not a PG error code: ${value} is success`);
  }
  return value;
}

function checkDetail(value: unknown): string {
  if (typeof value !== "string" || value === "") {
    throw new DbObserverError("malformed-statement", "error detail must be a non-empty string");
  }
  if (value.length > MAX_DETAIL_LEN) {
    throw new DbObserverError("capacity-exhausted", "error detail exceeds the length cap");
  }
  return value;
}

function checkSchema(value: unknown, maxCellsPerRow: number): PoisonSentinelSchema {
  if (!Array.isArray(value) || value.length === 0) {
    throw new DbObserverError("malformed-statement", "schema pin must be a non-empty tag array");
  }
  if (value.length > maxCellsPerRow) {
    throw new DbObserverError("capacity-exhausted", "schema pin exceeds the cell cap");
  }
  const tags: DbCellTag[] = [];
  for (const entry of value) {
    if (!(DB_CELL_TAGS as readonly unknown[]).includes(entry)) {
      throw new DbObserverError("malformed-statement", `unknown schema tag: ${String(entry)}`);
    }
    tags.push(entry as DbCellTag);
  }
  return Object.freeze(tags);
}

function checkOutcome(value: unknown): PoisonSettlementOutcome {
  if (!(POISON_SETTLEMENT_OUTCOMES as readonly unknown[]).includes(value)) {
    throw new DbObserverError("malformed-statement", `not a settlement outcome: ${String(value)}`);
  }
  return value as PoisonSettlementOutcome;
}

function checkCallback(value: unknown): PoisonCallbackReport {
  if (!(POISON_CALLBACK_REPORTS as readonly unknown[]).includes(value)) {
    throw new DbObserverError("malformed-statement", `not a callback report: ${String(value)}`);
  }
  return value as PoisonCallbackReport;
}

// ---------------------------------------------------------------------------
// Facts
// ---------------------------------------------------------------------------

export type PoisonSentinelSchema = readonly DbCellTag[];

export type PoisonAttemptRecord = Readonly<{
  attempt: string;
  kind: "poison" | "control";
  schema: PoisonSentinelSchema;
  sentinelDigest: string;
  poisonDigest: string | undefined;
}>;

export type PoisonErrorFacts = Readonly<{
  attempt: string;
  code: string;
  class: string;
  detail: string;
  detailDigest: string;
}>;

// A callback report is an inert label: it records what the transaction
// callback claimed and carries a fixed proves:"nothing" tag so no reader can
// mistake it for settlement evidence.
export type PoisonCallbackFacts = Readonly<{
  attempt: string;
  reported: PoisonCallbackReport;
  proves: "nothing";
}>;

export type PoisonSettlementFacts = Readonly<{
  attempt: string;
  outcome: PoisonSettlementOutcome;
  terminal: true;
}>;

export type SentinelRowFacts = Readonly<{
  seq: number;
  cells: readonly DbCell[];
  digest: string;
}>;

export type FreshSentinelFacts = Readonly<{
  attempt: string;
  read: "fresh";
  rowCount: number;
  rows: readonly SentinelRowFacts[];
  digest: string;
}>;

export type ReplaySentinelFacts = Readonly<{
  attempt: string;
  read: "replay";
  rowCount: number;
  rows: readonly SentinelRowFacts[];
  digest: string;
}>;

export type SentinelComparison = Readonly<{
  match: boolean;
  mismatches: readonly number[];
}>;

export type SettlementAgreement = Readonly<{
  agree: boolean;
  reason: string;
}>;

// ---------------------------------------------------------------------------
// Exact cells, digests, and comparison (same contract as the K22 core)
// ---------------------------------------------------------------------------

function cellFromTagged(value: unknown, maxCellBytes: number): DbCell {
  if (value === null || typeof value !== "object" || Array.isArray(value)) {
    throw new DbObserverError("wrong-cell-type", "cell must be a tagged object");
  }
  const record = value as Record<string, unknown>;
  switch (record["tag"]) {
    case "number":
      return makeNumberCell(record["lexeme"]);
    case "text": {
      const cell = makeTextCell(record["text"]);
      if ((cell as { text: string }).text.length > maxCellBytes) {
        throw new DbObserverError("capacity-exhausted", "text cell exceeds the byte cap");
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
                  "bytes cell needs a Uint8Array or byte array",
                );
              })();
      if (bytes.length > maxCellBytes) {
        throw new DbObserverError("capacity-exhausted", "bytes cell exceeds the byte cap");
      }
      return makeBytesCell(bytes);
    }
    case "null":
      return makeNullCell();
    default:
      throw new DbObserverError("wrong-cell-type", `unknown cell tag: ${String(record["tag"])}`);
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
  // Bytes cells cross as copies so caller-held buffers can never alias or
  // corrupt observer state. Strings are immutable; only bytes need it.
  if (cell.tag !== "bytes") return cell;
  return Object.freeze({ tag: "bytes", bytes: Uint8Array.from(cell.bytes) }) as DbCell;
}

function rowsToFacts(rows: readonly (readonly DbCell[])[]): {
  rows: readonly SentinelRowFacts[];
  digest: string;
} {
  const facts: SentinelRowFacts[] = rows.map((cells, index) => {
    const out = cells.map(copyCell);
    const digest = `sha256:${createHash("sha256")
      .update(out.map((cell) => cellDigest(cell)).join(","), "utf8")
      .digest("hex")}`;
    return Object.freeze({ seq: index, cells: Object.freeze(out), digest });
  });
  const digest = `sha256:${createHash("sha256")
    .update(facts.map((row) => row.digest).join(","), "utf8")
    .digest("hex")}`;
  return { rows: Object.freeze(facts), digest };
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

function rowEquals(a: readonly DbCell[], b: readonly DbCell[]): boolean {
  if (a.length !== b.length) return false;
  for (let index = 0; index < a.length; index++) {
    if (!cellsEqual(a[index] as DbCell, b[index] as DbCell)) return false;
  }
  return true;
}

type AttemptEntry = {
  record: PoisonAttemptRecord;
  sentinel: DbCell[];
  error: PoisonErrorFacts | undefined;
  callback: PoisonCallbackFacts | undefined;
  settlement: PoisonSettlementFacts | undefined;
  fresh: DbCell[][] | undefined;
  replay: DbCell[][] | undefined;
};

// ---------------------------------------------------------------------------
// Observer: poison errors, terminal settlement, fresh/replay sentinel reads
// ---------------------------------------------------------------------------

export class PoisonObserver {
  private readonly limits: DbObserverLimits;
  private readonly attempts = new Map<string, AttemptEntry>();
  private nextAttempt = 0;

  constructor(limits: DbObserverLimits) {
    this.limits = limits;
  }

  get attemptCount(): number {
    return this.attempts.size;
  }

  private checkSentinelRow(schema: PoisonSentinelSchema, row: unknown): DbCell[] {
    if (!Array.isArray(row)) {
      throw new DbObserverError("row-arity", "sentinel row must be an array of cells");
    }
    if (row.length !== schema.length) {
      throw new DbObserverError(
        "row-arity",
        `wrong sentinel row schema: want ${schema.length} cells, got ${row.length}`,
      );
    }
    return row.map((cell, index) => {
      const built = cellFromTagged(cell, this.limits.maxCellBytes);
      if (built.tag !== schema[index]) {
        throw new DbObserverError(
          "wrong-cell-type",
          `wrong sentinel row schema at column ${index}: want ${schema[index] as string}, ` +
            `got ${built.tag}`,
        );
      }
      return built;
    });
  }

  private beginAttempt(
    kind: "poison" | "control",
    schema: unknown,
    sentinelRow: unknown,
    poisonStatement: unknown,
  ): PoisonAttemptRecord {
    const pin = checkSchema(schema, this.limits.maxCellsPerRow);
    const sentinel = this.checkSentinelRow(pin, sentinelRow);
    let poisonDigest: string | undefined;
    if (kind === "poison") {
      const text = checkPoisonStatement(poisonStatement);
      poisonDigest = `sha256:${createHash("sha256").update(text, "utf8").digest("hex")}`;
    } else if (poisonStatement !== undefined) {
      throw new DbObserverError(
        "malformed-statement",
        "control attempt omits the poison write: no poison statement allowed",
      );
    }
    if (this.attempts.size >= POISON_MAX_ATTEMPTS) {
      throw new DbObserverError("capacity-exhausted", "attempt table full");
    }
    const attempt = `attempt-${this.nextAttempt}`;
    this.nextAttempt += 1;
    const sentinelDigest = `sha256:${createHash("sha256")
      .update(sentinel.map((cell) => cellDigest(cell)).join(","), "utf8")
      .digest("hex")}`;
    const record: PoisonAttemptRecord = Object.freeze({
      attempt,
      kind,
      schema: pin,
      sentinelDigest,
      poisonDigest,
    });
    this.attempts.set(attempt, {
      record,
      sentinel: sentinel.map(copyCell),
      error: undefined,
      callback: undefined,
      settlement: undefined,
      fresh: undefined,
      replay: undefined,
    });
    return record;
  }

  // Begin one poisoned transaction attempt: the sentinel row the transaction
  // writes plus the poison statement that must fail. The statement is named
  // by digest only, never executed.
  beginPoisonAttempt(
    schema: unknown,
    sentinelRow: unknown,
    poisonStatement: unknown,
  ): PoisonAttemptRecord {
    return this.beginAttempt("poison", schema, sentinelRow, poisonStatement);
  }

  // Begin the omit-poison control: the same sentinel write with the poison
  // statement skipped. The control must settle committed with the sentinel
  // visibly present, proving the read path can see a committed sentinel.
  beginControlAttempt(schema: unknown, sentinelRow: unknown): PoisonAttemptRecord {
    return this.beginAttempt("control", schema, sentinelRow, undefined);
  }

  attemptRecord(attempt: unknown): PoisonAttemptRecord {
    return this.requireAttempt(checkAttemptId(attempt)).record;
  }

  private requireAttempt(attempt: string): AttemptEntry {
    const entry = this.attempts.get(attempt);
    if (entry === undefined) {
      throw new DbObserverError("unknown-connection", `no such attempt: ${attempt}`);
    }
    return entry;
  }

  // Record the PG error-code observation for the poison write: the SQLSTATE
  // code, its two-character class, and a bounded detail string. Single
  // record: a second error observation on one attempt throws.
  recordPoisonError(attempt: unknown, code: unknown, detail: unknown): PoisonErrorFacts {
    const id = checkAttemptId(attempt);
    const entry = this.requireAttempt(id);
    if (entry.record.kind !== "poison") {
      throw new DbObserverError(
        "malformed-statement",
        `control attempt ${id} omits the poison write: no poison error possible`,
      );
    }
    if (entry.error !== undefined) {
      throw new DbObserverError("connection-busy", `poison error already recorded for ${id}`);
    }
    const observed = checkPgCode(code);
    const text = checkDetail(detail);
    const facts: PoisonErrorFacts = Object.freeze({
      attempt: id,
      code: observed,
      class: observed.slice(0, 2),
      detail: text,
      detailDigest: `sha256:${createHash("sha256").update(text, "utf8").digest("hex")}`,
    });
    entry.error = facts;
    return facts;
  }

  errorFacts(attempt: unknown): PoisonErrorFacts {
    const id = checkAttemptId(attempt);
    const stored = this.requireAttempt(id).error;
    if (stored === undefined) {
      throw new DbObserverError("no-conversation", `no poison error recorded for ${id}`);
    }
    return stored;
  }

  // Record what the transaction callback claimed: "success" or "threw". The
  // report is inert (proves:"nothing") and is never consulted by settlement
  // facts or agreement verdicts.
  recordCallbackReport(attempt: unknown, reported: unknown): PoisonCallbackFacts {
    const id = checkAttemptId(attempt);
    const entry = this.requireAttempt(id);
    if (entry.callback !== undefined) {
      throw new DbObserverError("connection-busy", `callback report already recorded for ${id}`);
    }
    const facts: PoisonCallbackFacts = Object.freeze({
      attempt: id,
      reported: checkCallback(reported),
      proves: "nothing",
    });
    entry.callback = facts;
    return facts;
  }

  callbackFacts(attempt: unknown): PoisonCallbackFacts {
    const id = checkAttemptId(attempt);
    const stored = this.requireAttempt(id).callback;
    if (stored === undefined) {
      throw new DbObserverError("no-conversation", `no callback report recorded for ${id}`);
    }
    return stored;
  }

  // Record the terminal settlement outcome: rolled-back or committed. The
  // record is terminal: a second settlement on one attempt throws, and no
  // path here derives settlement from a callback report.
  recordSettlement(attempt: unknown, outcome: unknown): PoisonSettlementFacts {
    const id = checkAttemptId(attempt);
    const entry = this.requireAttempt(id);
    if (entry.settlement !== undefined) {
      throw new DbObserverError(
        "connection-busy",
        `terminal settlement already recorded for ${id}`,
      );
    }
    const facts: PoisonSettlementFacts = Object.freeze({
      attempt: id,
      outcome: checkOutcome(outcome),
      terminal: true,
    });
    entry.settlement = facts;
    return facts;
  }

  settlementFacts(attempt: unknown): PoisonSettlementFacts {
    const id = checkAttemptId(attempt);
    const stored = this.requireAttempt(id).settlement;
    if (stored === undefined) {
      throw new DbObserverError("no-conversation", `no terminal settlement recorded for ${id}`);
    }
    return stored;
  }

  private checkReadRows(
    entry: AttemptEntry,
    rows: readonly (readonly unknown[])[],
    what: string,
  ): DbCell[][] {
    if (!Array.isArray(rows)) {
      throw new DbObserverError("row-arity", `${what} rows must be an array of rows`);
    }
    if (rows.length > this.limits.maxRows) {
      throw new DbObserverError("capacity-exhausted", `${what} rows exceed the row cap`);
    }
    const pin = entry.record.schema;
    const exact: DbCell[][] = [];
    for (const row of rows) {
      if (!Array.isArray(row)) {
        throw new DbObserverError("row-arity", `each ${what} row must be an array of cells`);
      }
      if (row.length !== pin.length) {
        throw new DbObserverError(
          "row-arity",
          `wrong ${what} row schema: want ${pin.length} cells, got ${row.length}`,
        );
      }
      exact.push(
        row.map((cell, index) => {
          const built = cellFromTagged(cell, this.limits.maxCellBytes);
          if (built.tag !== pin[index]) {
            throw new DbObserverError(
              "wrong-cell-type",
              `wrong ${what} row schema at column ${index}: want ${pin[index] as string}, ` +
                `got ${built.tag}`,
            );
          }
          return built;
        }),
      );
    }
    return exact;
  }

  // Record the fresh raw sentinel read: the observer's own fresh-connection
  // read-out of the sentinel rows after settlement, in stored order.
  recordFreshRead(attempt: unknown, rows: readonly (readonly unknown[])[]): FreshSentinelFacts {
    const id = checkAttemptId(attempt);
    const entry = this.requireAttempt(id);
    if (entry.fresh !== undefined) {
      throw new DbObserverError(
        "connection-busy",
        `fresh sentinel read already recorded for ${id}`,
      );
    }
    const exact = this.checkReadRows(entry, rows, "fresh");
    entry.fresh = exact.map((row) => row.map(copyCell));
    return this.freshFacts(id);
  }

  freshFacts(attempt: unknown): FreshSentinelFacts {
    const id = checkAttemptId(attempt);
    const stored = this.requireAttempt(id).fresh;
    if (stored === undefined) {
      throw new DbObserverError("no-conversation", `no fresh sentinel read recorded for ${id}`);
    }
    const facts = rowsToFacts(stored);
    return Object.freeze({ attempt: id, read: "fresh", rowCount: stored.length, ...facts });
  }

  // Record the replay sentinel read: the observer's second independent
  // re-read confirming the fresh read-out. Fresh and replay must agree.
  recordReplayRead(attempt: unknown, rows: readonly (readonly unknown[])[]): ReplaySentinelFacts {
    const id = checkAttemptId(attempt);
    const entry = this.requireAttempt(id);
    if (entry.replay !== undefined) {
      throw new DbObserverError(
        "connection-busy",
        `replay sentinel read already recorded for ${id}`,
      );
    }
    const exact = this.checkReadRows(entry, rows, "replay");
    entry.replay = exact.map((row) => row.map(copyCell));
    return this.replayFacts(id);
  }

  replayFacts(attempt: unknown): ReplaySentinelFacts {
    const id = checkAttemptId(attempt);
    const stored = this.requireAttempt(id).replay;
    if (stored === undefined) {
      throw new DbObserverError("no-conversation", `no replay sentinel read recorded for ${id}`);
    }
    const facts = rowsToFacts(stored);
    return Object.freeze({ attempt: id, read: "replay", rowCount: stored.length, ...facts });
  }

  // Compare the fresh read against the replay read. Both sides are raw
  // observer facts under one attempt: row count, row order, and every cell
  // must agree tag-for-tag and bit-for-bit.
  compareFreshToReplay(fresh: FreshSentinelFacts, replay: ReplaySentinelFacts): SentinelComparison {
    if (fresh.attempt !== replay.attempt) {
      throw new DbObserverError("malformed-statement", "cannot compare reads across attempts");
    }
    if (fresh.rowCount !== replay.rowCount || fresh.rows.length !== replay.rows.length) {
      return Object.freeze({ match: false, mismatches: Object.freeze([-1]) });
    }
    const mismatches: number[] = [];
    for (let index = 0; index < fresh.rows.length; index++) {
      const left = (fresh.rows[index] as SentinelRowFacts).cells;
      const right = (replay.rows[index] as SentinelRowFacts).cells;
      if (!rowEquals(left, right)) mismatches.push(index);
    }
    return Object.freeze({ match: mismatches.length === 0, mismatches: Object.freeze(mismatches) });
  }

  // Whether the attempt's sentinel row is present (exact tag-for-tag and
  // bit-for-bit row equality) in one recorded read.
  sentinelPresentIn(attempt: unknown, read: FreshSentinelFacts | ReplaySentinelFacts): boolean {
    const id = checkAttemptId(attempt);
    if (read.attempt !== id) {
      throw new DbObserverError("malformed-statement", "cannot probe a read from another attempt");
    }
    const sentinel = this.requireAttempt(id).sentinel;
    return read.rows.some((row) => rowEquals(row.cells, sentinel));
  }

  // Agreement verdict: terminal settlement plus fresh/replay presence must
  // line up — rolled-back needs the sentinel absent from BOTH reads,
  // committed needs it present in BOTH. Any missing evidence throws instead
  // of defaulting; callback reports are never consulted.
  settlementAgreesWithReads(attempt: unknown): SettlementAgreement {
    const id = checkAttemptId(attempt);
    const entry = this.requireAttempt(id);
    const settlement = entry.settlement;
    if (settlement === undefined) {
      throw new DbObserverError("no-conversation", `no terminal settlement recorded for ${id}`);
    }
    const fresh = entry.fresh;
    if (fresh === undefined) {
      throw new DbObserverError("no-conversation", `no fresh sentinel read recorded for ${id}`);
    }
    const replay = entry.replay;
    if (replay === undefined) {
      throw new DbObserverError("no-conversation", `no replay sentinel read recorded for ${id}`);
    }
    const inFresh = fresh.some((row) => rowEquals(row, entry.sentinel));
    const inReplay = replay.some((row) => rowEquals(row, entry.sentinel));
    if (inFresh !== inReplay) {
      return Object.freeze({ agree: false, reason: "fresh-replay-diverged" });
    }
    if (settlement.outcome === "rolled-back") {
      return inFresh
        ? Object.freeze({ agree: false, reason: "sentinel-survived-rollback" })
        : Object.freeze({ agree: true, reason: "sentinel-absent-after-rollback" });
    }
    return inFresh
      ? Object.freeze({ agree: true, reason: "sentinel-present-after-commit" })
      : Object.freeze({ agree: false, reason: "sentinel-missing-after-commit" });
  }
}
