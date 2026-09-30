// K23: raw RETURNING and final-row observations beside an ordinary C fixture.
//
// An INDEPENDENT raw observer over in-memory doubles only: no real database,
// no driver, no sockets, no I/O. Beside one ordinary compiled C fixture (an
// opaque test-supplied identity that is never executed here), this module
// records three raw facts on the observer's own path: the compile record
// (statement digest plus pinned row schema), the RETURNING payload rows, and
// the final-row read-out. Payload and final rows carry typed exact cells
// (number/text/bytes/null) in stored order, mirroring the K22 core, and every
// failure reuses the core's layered DbObserverError (engine/driver/sql).
//
// Non-credit boundary (normative):
//   Raw RETURNING payload rows ALONE can never credit the compiled C fixture.
//   D1 credit needs the C-side witness plus payload/final-row agreement, and
//   that verdict belongs to QD1. This module records raw facts only: its
//   credit verdict is always { creditC: false }, and no helper here claims
//   execution credit. See RETURNING_NON_CREDIT_BOUNDARY.
//
// Independence scope from the Can adapter under test (normative):
//   SHARES: nothing executable. The observer never imports, calls, or reads
//   adapter code, adapter-decoded values, or adapter connection state. The
//   only strings that may coincide are the fixture/statement labels under
//   test, supplied by the test, never by the adapter.
//   PROVES INDEPENDENTLY: compile identity, pinned row schema, raw row count,
//   exact payload/final cells, row order, and layered error provenance from
//   its own seeded doubles alone. Adapter claims are compared AGAINST these
//   facts; the observer never derives a fact FROM an adapter claim.
//   Consequently a coincidentally correct adapter value still fails when the
//   observer's exact cell disagrees (narrowed bigint, re-encoded text,
//   dropped byte, reordered row, wrong schema).
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

export const RETURNING_OBSERVER_SCHEMA_VERSION = "1" as const;

// Recorded non-credit boundary: raw RETURNING alone must never credit C.
export const RETURNING_NON_CREDIT_BOUNDARY: string =
  "non-credit: raw RETURNING payload rows alone cannot credit the compiled C fixture; " +
  "D1 credit needs the C-side witness plus final-row agreement, decided at QD1 from " +
  "observer facts — this module records raw facts only and never claims execution credit";

// What QD1 still needs beyond this module's raw facts. Listed so a reader can
// see the gap between "payload recorded" and "C credited".
export const RETURNING_CREDIT_REQUIREMENTS: readonly string[] = Object.freeze([
  "c-side-witness",
  "final-row-agreement",
  "qd1-verdict",
]);

export const RETURNING_MAX_COMPILES = 8;
const MAX_FIXTURE_LEN = 128;
const MAX_STATEMENT_LEN = 4096;

function checkFixture(value: unknown): string {
  if (typeof value !== "string" || value === "" || value.length > MAX_FIXTURE_LEN) {
    throw new DbObserverError(
      "malformed-statement",
      `fixture must be a non-empty label of at most ${MAX_FIXTURE_LEN} chars`,
    );
  }
  if (!/^[A-Za-z0-9][A-Za-z0-9_.:-]*$/.test(value)) {
    throw new DbObserverError(
      "malformed-statement",
      `fixture carries illegal characters: ${value}`,
    );
  }
  return value;
}

function checkStatement(value: unknown): string {
  if (typeof value !== "string" || value === "") {
    throw new DbObserverError("malformed-statement", "statement must be a non-empty string");
  }
  if (value.length > MAX_STATEMENT_LEN) {
    throw new DbObserverError("capacity-exhausted", "statement exceeds the length cap");
  }
  return value;
}

function checkCompileId(value: unknown): string {
  if (typeof value !== "string" || !/^compile-[0-9]+$/.test(value)) {
    throw new DbObserverError("malformed-statement", `not a compile id: ${String(value)}`);
  }
  return value;
}

function checkSchema(value: unknown, maxCellsPerRow: number): ReturningSchemaPin {
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

// ---------------------------------------------------------------------------
// Facts
// ---------------------------------------------------------------------------

export type ReturningSchemaPin = readonly DbCellTag[];

export type CompileRecord = Readonly<{
  compile: string;
  fixture: string;
  statementDigest: string;
  schema: ReturningSchemaPin;
}>;

export type ReturningRowFacts = Readonly<{
  seq: number;
  cells: readonly DbCell[];
  digest: string;
}>;

export type ReturningPayloadFacts = Readonly<{
  compile: string;
  rowCount: number;
  rows: readonly ReturningRowFacts[];
  digest: string;
}>;

export type FinalRowsFacts = Readonly<{
  compile: string;
  rowCount: number;
  rows: readonly ReturningRowFacts[];
  digest: string;
}>;

export type ReturningComparison = Readonly<{
  match: boolean;
  mismatches: readonly number[];
}>;

// The only credit verdict this module can express: never credit C.
export type ReturningCreditVerdict = Readonly<{
  creditC: false;
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
  rows: readonly ReturningRowFacts[];
  digest: string;
} {
  const facts: ReturningRowFacts[] = rows.map((cells, index) => {
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

type CompileEntry = {
  record: CompileRecord;
  payload: DbCell[][] | undefined;
  finalRows: DbCell[][] | undefined;
};

// ---------------------------------------------------------------------------
// Observer: compile records, RETURNING payloads, final-row read-outs
// ---------------------------------------------------------------------------

export class ReturningObserver {
  private readonly limits: DbObserverLimits;
  private readonly compiles = new Map<string, CompileEntry>();
  private nextCompile = 0;

  constructor(limits: DbObserverLimits) {
    this.limits = limits;
  }

  get compileCount(): number {
    return this.compiles.size;
  }

  // Record one compiled C fixture beside its raw observation handle. The
  // fixture label is opaque: it is named, never executed. The schema pin
  // freezes the expected per-column cell tags; every payload and final row
  // recorded under this compile must match it tag-for-tag, so a wrong row
  // schema is detectable at record time. Facts carry the statement digest
  // only, never authority beyond the pin.
  recordCompile(fixture: unknown, statement: unknown, schema: unknown): CompileRecord {
    const name = checkFixture(fixture);
    const text = checkStatement(statement);
    const pin = checkSchema(schema, this.limits.maxCellsPerRow);
    if (this.compiles.size >= RETURNING_MAX_COMPILES) {
      throw new DbObserverError("capacity-exhausted", "compile table full");
    }
    const compile = `compile-${this.nextCompile}`;
    this.nextCompile += 1;
    const record: CompileRecord = Object.freeze({
      compile,
      fixture: name,
      statementDigest: `sha256:${createHash("sha256").update(text, "utf8").digest("hex")}`,
      schema: pin,
    });
    this.compiles.set(compile, { record, payload: undefined, finalRows: undefined });
    return record;
  }

  compileRecord(compile: unknown): CompileRecord {
    return this.requireCompile(checkCompileId(compile)).record;
  }

  private requireCompile(compile: string): CompileEntry {
    const entry = this.compiles.get(compile);
    if (entry === undefined) {
      throw new DbObserverError("unknown-connection", `no such compile: ${compile}`);
    }
    return entry;
  }

  private checkRowsAgainstPin(
    compile: string,
    rows: readonly (readonly unknown[])[],
    what: string,
  ): DbCell[][] {
    const entry = this.requireCompile(compile);
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

  // Record the raw RETURNING payload rows: row count plus typed exact rows in
  // arrival order. Recording stores deep copies; facts are digest-stable.
  // Recording a payload is raw evidence only — it never credits C.
  recordReturning(compile: unknown, rows: readonly (readonly unknown[])[]): ReturningPayloadFacts {
    const id = checkCompileId(compile);
    const exact = this.checkRowsAgainstPin(id, rows, "payload");
    this.requireCompile(id).payload = exact.map((row) => row.map(copyCell));
    return this.payloadFacts(id);
  }

  payloadFacts(compile: unknown): ReturningPayloadFacts {
    const id = checkCompileId(compile);
    const stored = this.requireCompile(id).payload;
    if (stored === undefined) {
      throw new DbObserverError("no-conversation", `no RETURNING payload recorded for ${id}`);
    }
    const facts = rowsToFacts(stored);
    return Object.freeze({ compile: id, rowCount: stored.length, ...facts });
  }

  // Record the final-row read-out: the observer's own independent re-read of
  // the rows the payload claims. Same pin, same exactness, same order.
  recordFinalRows(compile: unknown, rows: readonly (readonly unknown[])[]): FinalRowsFacts {
    const id = checkCompileId(compile);
    const exact = this.checkRowsAgainstPin(id, rows, "final");
    this.requireCompile(id).finalRows = exact.map((row) => row.map(copyCell));
    return this.finalFacts(id);
  }

  finalFacts(compile: unknown): FinalRowsFacts {
    const id = checkCompileId(compile);
    const stored = this.requireCompile(id).finalRows;
    if (stored === undefined) {
      throw new DbObserverError("no-conversation", `no final rows recorded for ${id}`);
    }
    const facts = rowsToFacts(stored);
    return Object.freeze({ compile: id, rowCount: stored.length, ...facts });
  }

  // Compare the RETURNING payload against the final-row read-out. Both sides
  // are raw observer facts under one compile: row count, row order, and
  // every cell must agree tag-for-tag and bit-for-bit, so narrowing,
  // re-encoding, dropped bytes, and null/empty confusion are all detectable.
  // Agreement is still raw evidence only — it never credits C.
  comparePayloadToFinal(
    payload: ReturningPayloadFacts,
    final: FinalRowsFacts,
  ): ReturningComparison {
    if (payload.compile !== final.compile) {
      throw new DbObserverError("malformed-statement", "cannot compare facts across compiles");
    }
    if (payload.rowCount !== final.rowCount || payload.rows.length !== final.rows.length) {
      return Object.freeze({ match: false, mismatches: Object.freeze([-1]) });
    }
    const mismatches: number[] = [];
    for (let index = 0; index < payload.rows.length; index++) {
      const left = (payload.rows[index] as ReturningRowFacts).cells;
      const right = (final.rows[index] as ReturningRowFacts).cells;
      if (left.length !== right.length) {
        mismatches.push(index);
        continue;
      }
      let same = true;
      for (let column = 0; column < left.length; column++) {
        if (!cellsEqual(left[column] as DbCell, right[column] as DbCell)) {
          same = false;
          break;
        }
      }
      if (!same) mismatches.push(index);
    }
    return Object.freeze({ match: mismatches.length === 0, mismatches: Object.freeze(mismatches) });
  }

  // Compare one adapter-claimed row against the observer's exact row. Every
  // cell must match tag-for-tag and bit-for-bit, so a narrowed bigint
  // ("9007199254740992" for "9007199254740993"), a coerced decimal, or a
  // null/empty swap is detectable even when numerically or visually close.
  // The comparison reads observer facts only; it never trusts the claim.
  compareClaimedRow(stored: readonly DbCell[], claimed: readonly unknown[]): ReturningComparison {
    const mismatches: number[] = [];
    if (claimed.length !== stored.length) {
      return Object.freeze({ match: false, mismatches: Object.freeze([-1]) });
    }
    for (let index = 0; index < stored.length; index++) {
      let exact: DbCell;
      try {
        exact = cellFromTagged(claimed[index], this.limits.maxCellBytes);
      } catch {
        mismatches.push(index);
        continue;
      }
      if (!cellsEqual(stored[index] as DbCell, exact)) {
        mismatches.push(index);
      }
    }
    return Object.freeze({ match: mismatches.length === 0, mismatches: Object.freeze(mismatches) });
  }

  // The recorded non-credit boundary as a verdict: raw RETURNING evidence
  // alone always yields creditC false. There is no path in this module from
  // "payload recorded" to "C credited"; that verdict belongs to QD1.
  creditVerdict(): ReturningCreditVerdict {
    return Object.freeze({ creditC: false, reason: RETURNING_NON_CREDIT_BOUNDARY });
  }
}
