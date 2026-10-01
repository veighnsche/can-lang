// NT-I15 poison-observer adapter. Owner-bound operations over one
// adapter-held PoisonObserver (db-poison-service.ts, a verbatim
// K25 port): deterministic local mechanics, no transport,
// following the I03/I04 doubles precedent per the I13 Jev
// decision (evidence/I13-jev), which settled the observer-family
// merge shape. A revoked or foreign owner fails
// test::stale_handle without touching the service. The K25
// observer takes no scenario owner; admission is adapter-side
// only. Every service rejection maps VERBATIM to
// db::db_fault{layer, code}: K25 rejection codes are verdicts
// the capability exists to detect (bad SQLSTATE, unsettled
// reads, missing evidence, cross-attempt probes), not handle
// staleness. Service messages are dropped at the boundary:
// layer+code is the stable asserted contract; messages embed
// dynamic names. Facts map to nominal records with snake_case
// fields; ints cross as bigint; cells cross as copies. The
// union-typed sentinelPresentIn read splits into per-read ops;
// poison_digest is option::value<str>, none for the control.
import { failure, success, type AssertionContext, type Completion } from "../../../completion.ts";
import { array, dataArray, dataProperty, record, recordIdentity } from "../../../data.ts";
import { createDomainRuntime } from "../../../domain.ts";
import { copyBytes, isBytes, ownBytes } from "../../../bytes.ts";
import { ownerTag, type TestOwner } from "../../owner.ts";
import {
  checkLimits,
  DbObserverError,
  type DbCell,
  type DbObserverLimits,
} from "../i13/db-core-service.ts";
import {
  PoisonObserver,
  type FreshSentinelFacts,
  type PoisonAttemptRecord,
  type PoisonCallbackFacts,
  type PoisonErrorFacts,
  type PoisonSettlementFacts,
  type ReplaySentinelFacts,
  type SentinelComparison,
  type SentinelRowFacts,
  type SettlementAgreement,
} from "./db-poison-service.ts";

const origin = Object.freeze({
  source: "can:db",
  start: 0,
  end: 0,
  invocation: Object.freeze([]),
});

export type DbPoisonErrors = Readonly<{
  staleHandle: string;
  dbFault: string;
  numberCell: string;
  textCell: string;
  bytesCell: string;
  nullCell: string;
  optionSome: string;
  optionNone: string;
  poisonAttemptRecord: string;
  poisonErrorFacts: string;
  poisonCallbackFacts: string;
  poisonSettlementRecord: string;
  sentinelRowFacts: string;
  freshSentinelFacts: string;
  replaySentinelFacts: string;
  sentinelComparison: string;
  settlementAgreement: string;
}>;

function readStr(value: unknown, what: string): string {
  if (typeof value !== "string") throw new TypeError(`${what} needs a str`);
  return value;
}

function readDataArray(value: unknown, what: string): unknown[] {
  try {
    return dataArray(value);
  } catch {
    throw new TypeError(`${what} needs an array`);
  }
}

function readStrArray(value: unknown, what: string): string[] {
  return readDataArray(value, what).map((entry) => readStr(entry, `${what} entry`));
}

function readCell(value: unknown, errors: DbPoisonErrors, what: string): DbCell {
  const identity = recordIdentity(value);
  if (identity === errors.numberCell) {
    return { tag: "number", lexeme: readStr(dataProperty(value, "lexeme"), `${what} lexeme`) };
  }
  if (identity === errors.textCell) {
    return { tag: "text", text: readStr(dataProperty(value, "text"), `${what} text`) };
  }
  if (identity === errors.bytesCell) {
    const data = dataProperty(value, "data");
    if (!isBytes(data)) throw new TypeError(`${what} data needs a bytes::buffer`);
    return { tag: "bytes", bytes: copyBytes(data, origin) };
  }
  if (identity === errors.nullCell) return { tag: "null" };
  throw new TypeError(`${what} needs a db::cell`);
}

function readRowCells(value: unknown, errors: DbPoisonErrors, what: string): DbCell[] {
  return readDataArray(dataProperty(value, "cells"), `${what} cells`).map((cell, index) =>
    readCell(cell, errors, `${what}[${index}]`),
  );
}

function readSeedRows(value: unknown, errors: DbPoisonErrors, what: string): DbCell[][] {
  return readDataArray(value, what).map((row, index) =>
    readRowCells(row, errors, `${what}[${index}]`),
  );
}

const ADAPTER_LIMITS: DbObserverLimits = checkLimits({
  maxNamespaces: 4,
  maxConnections: 8,
  maxTables: 8,
  maxRows: 16,
  maxCellsPerRow: 8,
  maxCellBytes: 1024,
});

export function createDbPoison(
  domain: ReturnType<typeof createDomainRuntime>,
  errors: DbPoisonErrors,
  owner: TestOwner,
) {
  const service = new PoisonObserver(ADAPTER_LIMITS);

  function failStale(handleTag: string): Completion<never> {
    return failure(
      domain.create(
        errors.staleHandle,
        record(errors.staleHandle, [["handle", handleTag]]),
        origin,
      ),
    );
  }

  function failFault(layer: string, code: string): Completion<never> {
    return failure(
      domain.create(
        errors.dbFault,
        record(errors.dbFault, [
          ["layer", layer],
          ["code", code],
        ]),
        origin,
      ),
    );
  }

  function admitOwner(ownerHandle: unknown, what: string): string | Completion<never> {
    const tag = ownerTag(ownerHandle);
    if (tag === null) throw new TypeError(`${what} needs a test::owner handle`);
    const grant = owner.grantOf(ownerHandle);
    if (grant === null) return failStale(tag);
    return grant;
  }

  function attempt<T>(run: () => T): Completion<T> {
    try {
      return success(run());
    } catch (error) {
      if (error instanceof DbObserverError) return failFault(error.layer, error.code);
      throw error;
    }
  }

  function cellRecord(cell: DbCell): unknown {
    switch (cell.tag) {
      case "number":
        return record(errors.numberCell, [["lexeme", cell.lexeme]]);
      case "text":
        return record(errors.textCell, [["text", cell.text]]);
      case "bytes":
        return record(errors.bytesCell, [["data", ownBytes(cell.bytes)]]);
      case "null":
        return record(errors.nullCell, []);
    }
  }

  function digestOption(digest: string | undefined): unknown {
    if (digest === undefined) return record(errors.optionNone, []);
    return record(errors.optionSome, [["value", digest]]);
  }

  function attemptRecord(facts: PoisonAttemptRecord): unknown {
    return record(errors.poisonAttemptRecord, [
      ["attempt", facts.attempt],
      ["kind", facts.kind],
      ["schema", array([...facts.schema])],
      ["sentinel_digest", facts.sentinelDigest],
      ["poison_digest", digestOption(facts.poisonDigest)],
    ]);
  }

  function errorRecord(facts: PoisonErrorFacts): unknown {
    return record(errors.poisonErrorFacts, [
      ["attempt", facts.attempt],
      ["code", facts.code],
      ["class", facts.class],
      ["detail", facts.detail],
      ["detail_digest", facts.detailDigest],
    ]);
  }

  function callbackRecord(facts: PoisonCallbackFacts): unknown {
    return record(errors.poisonCallbackFacts, [
      ["attempt", facts.attempt],
      ["reported", facts.reported],
      ["proves", facts.proves],
    ]);
  }

  function settlementRecord(facts: PoisonSettlementFacts): unknown {
    return record(errors.poisonSettlementRecord, [
      ["attempt", facts.attempt],
      ["outcome", facts.outcome],
      ["terminal", facts.terminal],
    ]);
  }

  function sentinelRowRecord(facts: SentinelRowFacts): unknown {
    return record(errors.sentinelRowFacts, [
      ["seq", BigInt(facts.seq)],
      ["cells", array(facts.cells.map(cellRecord))],
      ["digest", facts.digest],
    ]);
  }

  function freshRecord(facts: FreshSentinelFacts): unknown {
    return record(errors.freshSentinelFacts, [
      ["attempt", facts.attempt],
      ["read", facts.read],
      ["row_count", BigInt(facts.rowCount)],
      ["rows", array(facts.rows.map(sentinelRowRecord))],
      ["digest", facts.digest],
    ]);
  }

  function replayRecord(facts: ReplaySentinelFacts): unknown {
    return record(errors.replaySentinelFacts, [
      ["attempt", facts.attempt],
      ["read", facts.read],
      ["row_count", BigInt(facts.rowCount)],
      ["rows", array(facts.rows.map(sentinelRowRecord))],
      ["digest", facts.digest],
    ]);
  }

  function comparisonRecord(facts: SentinelComparison): unknown {
    return record(errors.sentinelComparison, [
      ["match", facts.match],
      ["mismatches", array(facts.mismatches.map((n) => BigInt(n)))],
    ]);
  }

  function agreementRecord(facts: SettlementAgreement): unknown {
    return record(errors.settlementAgreement, [
      ["agree", facts.agree],
      ["reason", facts.reason],
    ]);
  }

  function readInt(value: unknown, what: string): number {
    if (typeof value !== "bigint") throw new TypeError(`${what} needs an int`);
    return Number(value);
  }

  function readSentinelRead(
    value: unknown,
    what: string,
  ): {
    attempt: string;
    rowCount: number;
    rows: { seq: number; cells: DbCell[]; digest: string }[];
  } {
    const rows = readDataArray(dataProperty(value, "rows"), `${what} rows`).map((row, index) => ({
      seq: index,
      cells: readRowCells(row, errors, `${what}[${index}]`),
      digest: "",
    }));
    return {
      attempt: readStr(dataProperty(value, "attempt"), `${what} attempt`),
      rowCount: readInt(dataProperty(value, "row_count"), `${what} row_count`),
      rows,
    };
  }

  return Object.freeze({
    async beginPoisonAttempt(
      ownerHandle: unknown,
      schema: unknown,
      sentinelRow: unknown,
      poisonStatement: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "db::begin_poison_attempt");
      if (typeof admitted !== "string") return admitted;
      const pin = readStrArray(schema, "db::begin_poison_attempt schema");
      const row = readRowCells(sentinelRow, errors, "db::begin_poison_attempt sentinel_row");
      const stmt = readStr(poisonStatement, "db::begin_poison_attempt poison_statement");
      return attempt(() => attemptRecord(service.beginPoisonAttempt(pin, row, stmt)));
    },

    async beginControlAttempt(
      ownerHandle: unknown,
      schema: unknown,
      sentinelRow: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "db::begin_control_attempt");
      if (typeof admitted !== "string") return admitted;
      const pin = readStrArray(schema, "db::begin_control_attempt schema");
      const row = readRowCells(sentinelRow, errors, "db::begin_control_attempt sentinel_row");
      return attempt(() => attemptRecord(service.beginControlAttempt(pin, row)));
    },

    async attemptRecord(
      ownerHandle: unknown,
      attemptId: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "db::attempt_record");
      if (typeof admitted !== "string") return admitted;
      const id = readStr(attemptId, "db::attempt_record attempt");
      return attempt(() => attemptRecord(service.attemptRecord(id)));
    },

    async recordPoisonError(
      ownerHandle: unknown,
      attemptId: unknown,
      code: unknown,
      detail: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "db::record_poison_error");
      if (typeof admitted !== "string") return admitted;
      const id = readStr(attemptId, "db::record_poison_error attempt");
      const sqlstate = readStr(code, "db::record_poison_error code");
      const text = readStr(detail, "db::record_poison_error detail");
      return attempt(() => errorRecord(service.recordPoisonError(id, sqlstate, text)));
    },

    async errorFacts(
      ownerHandle: unknown,
      attemptId: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "db::error_facts");
      if (typeof admitted !== "string") return admitted;
      const id = readStr(attemptId, "db::error_facts attempt");
      return attempt(() => errorRecord(service.errorFacts(id)));
    },

    async recordCallbackReport(
      ownerHandle: unknown,
      attemptId: unknown,
      reported: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "db::record_callback_report");
      if (typeof admitted !== "string") return admitted;
      const id = readStr(attemptId, "db::record_callback_report attempt");
      const label = readStr(reported, "db::record_callback_report reported");
      return attempt(() => callbackRecord(service.recordCallbackReport(id, label)));
    },

    async callbackFacts(
      ownerHandle: unknown,
      attemptId: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "db::callback_facts");
      if (typeof admitted !== "string") return admitted;
      const id = readStr(attemptId, "db::callback_facts attempt");
      return attempt(() => callbackRecord(service.callbackFacts(id)));
    },

    async recordPoisonSettlement(
      ownerHandle: unknown,
      attemptId: unknown,
      outcome: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "db::record_poison_settlement");
      if (typeof admitted !== "string") return admitted;
      const id = readStr(attemptId, "db::record_poison_settlement attempt");
      const out = readStr(outcome, "db::record_poison_settlement outcome");
      return attempt(() => settlementRecord(service.recordSettlement(id, out)));
    },

    async poisonSettlementFacts(
      ownerHandle: unknown,
      attemptId: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "db::poison_settlement_facts");
      if (typeof admitted !== "string") return admitted;
      const id = readStr(attemptId, "db::poison_settlement_facts attempt");
      return attempt(() => settlementRecord(service.settlementFacts(id)));
    },

    async recordFreshRead(
      ownerHandle: unknown,
      attemptId: unknown,
      rows: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "db::record_fresh_read");
      if (typeof admitted !== "string") return admitted;
      const id = readStr(attemptId, "db::record_fresh_read attempt");
      const fresh = readSeedRows(rows, errors, "db::record_fresh_read rows");
      return attempt(() => freshRecord(service.recordFreshRead(id, fresh)));
    },

    async freshFacts(
      ownerHandle: unknown,
      attemptId: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "db::fresh_facts");
      if (typeof admitted !== "string") return admitted;
      const id = readStr(attemptId, "db::fresh_facts attempt");
      return attempt(() => freshRecord(service.freshFacts(id)));
    },

    async recordReplayRead(
      ownerHandle: unknown,
      attemptId: unknown,
      rows: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "db::record_replay_read");
      if (typeof admitted !== "string") return admitted;
      const id = readStr(attemptId, "db::record_replay_read attempt");
      const replay = readSeedRows(rows, errors, "db::record_replay_read rows");
      return attempt(() => replayRecord(service.recordReplayRead(id, replay)));
    },

    async replayFacts(
      ownerHandle: unknown,
      attemptId: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "db::replay_facts");
      if (typeof admitted !== "string") return admitted;
      const id = readStr(attemptId, "db::replay_facts attempt");
      return attempt(() => replayRecord(service.replayFacts(id)));
    },

    async compareFreshToReplay(
      ownerHandle: unknown,
      fresh: unknown,
      replay: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "db::compare_fresh_to_replay");
      if (typeof admitted !== "string") return admitted;
      const have = readSentinelRead(fresh, "db::compare_fresh_to_replay fresh");
      const want = readSentinelRead(replay, "db::compare_fresh_to_replay replay");
      return attempt(() =>
        comparisonRecord(
          service.compareFreshToReplay(
            {
              attempt: have.attempt,
              read: "fresh",
              rowCount: have.rowCount,
              rows: have.rows,
              digest: "",
            },
            {
              attempt: want.attempt,
              read: "replay",
              rowCount: want.rowCount,
              rows: want.rows,
              digest: "",
            },
          ),
        ),
      );
    },

    async sentinelPresentInFresh(
      ownerHandle: unknown,
      attemptId: unknown,
      read: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "db::sentinel_present_in_fresh");
      if (typeof admitted !== "string") return admitted;
      const id = readStr(attemptId, "db::sentinel_present_in_fresh attempt");
      const facts = readSentinelRead(read, "db::sentinel_present_in_fresh read");
      return attempt(() =>
        service.sentinelPresentIn(id, {
          attempt: facts.attempt,
          read: "fresh",
          rowCount: facts.rowCount,
          rows: facts.rows,
          digest: "",
        }),
      );
    },

    async sentinelPresentInReplay(
      ownerHandle: unknown,
      attemptId: unknown,
      read: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "db::sentinel_present_in_replay");
      if (typeof admitted !== "string") return admitted;
      const id = readStr(attemptId, "db::sentinel_present_in_replay attempt");
      const facts = readSentinelRead(read, "db::sentinel_present_in_replay read");
      return attempt(() =>
        service.sentinelPresentIn(id, {
          attempt: facts.attempt,
          read: "replay",
          rowCount: facts.rowCount,
          rows: facts.rows,
          digest: "",
        }),
      );
    },

    async settlementAgreesWithReads(
      ownerHandle: unknown,
      attemptId: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "db::settlement_agrees_with_reads");
      if (typeof admitted !== "string") return admitted;
      const id = readStr(attemptId, "db::settlement_agrees_with_reads attempt");
      return attempt(() => agreementRecord(service.settlementAgreesWithReads(id)));
    },
  });
}
