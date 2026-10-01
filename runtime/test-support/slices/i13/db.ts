// NT-I13 database-observer adapter. Owner-bound operations over one
// adapter-held DbObserverService (db-core-service.ts, a verbatim K22
// port) plus one adapter-held ReturningObserver
// (db-returning-service.ts, a verbatim K23 port): deterministic
// local mechanics, no transport, following the I03/I04 doubles
// precedent per the I13 Jev decision (evidence/I13-jev,
// full_adapter_held_service 2-1 with investigated dissent). A
// revoked or foreign owner fails test::stale_handle without
// touching either service. The admitted test-owner grant doubles
// as the service-level scenario owner, isolating callers within
// the shared double; grants are fixed 36-char ng1-hex and always
// satisfy the service name rule. Every service rejection maps
// VERBATIM to db::db_fault{layer, code}: K22/K23 rejection codes
// are verdicts the capability exists to detect (swap, forgery,
// arity, narrowing, caps), not handle staleness, so collapsing
// any of them into stale_handle would hide the finding from Can
// rows. (The consult briefs sketched an i11-style stale mapping
// for handle/owner/token codes; the I03 mapping correction plus
// the K22 check names prove these codes are verdicts, so verbatim
// mapping supersedes that gloss. The adapter-held shape is
// unaffected.) Service messages are dropped at the boundary:
// layer+code is the stable asserted contract (K22
// "every-failure-names-its-layer"); messages embed dynamic names.
// Facts map to nominal records with snake_case fields; ints cross
// as bigint; bytes cells cross as copies in both directions.
import { failure, success, type AssertionContext, type Completion } from "../../../completion.ts";
import { array, dataArray, dataProperty, record, recordIdentity } from "../../../data.ts";
import { createDomainRuntime } from "../../../domain.ts";
import { copyBytes, isBytes, ownBytes } from "../../../bytes.ts";
import { ownerTag, type TestOwner } from "../../owner.ts";
import {
  checkLimits,
  DbObserverError,
  DbObserverService,
  type ConnectionFacts,
  type DbCell,
  type DbObserverLimits,
  type NamespaceReceipt,
  type ReadFacts,
  type RowFacts,
} from "./db-core-service.ts";
import {
  ReturningObserver,
  type CompileRecord,
  type FinalRowsFacts,
  type ReturningComparison,
  type ReturningCreditVerdict,
  type ReturningPayloadFacts,
  type ReturningRowFacts,
} from "./db-returning-service.ts";

const origin = Object.freeze({
  source: "can:db",
  start: 0,
  end: 0,
  invocation: Object.freeze([]),
});

export type DbObserverErrors = Readonly<{
  staleHandle: string;
  dbFault: string;
  numberCell: string;
  textCell: string;
  bytesCell: string;
  nullCell: string;
  namespaceReceipt: string;
  connectionFacts: string;
  rowFacts: string;
  readFacts: string;
  seedFacts: string;
  conversationFacts: string;
  rowComparison: string;
  compileFacts: string;
  returningRowFacts: string;
  returningPayloadFacts: string;
  finalRowsFacts: string;
  returningComparison: string;
  returningCreditVerdict: string;
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

function readCell(value: unknown, errors: DbObserverErrors, what: string): DbCell {
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

function readSeedRows(value: unknown, errors: DbObserverErrors, what: string): DbCell[][] {
  return readDataArray(value, what).map((row, index) =>
    readDataArray(dataProperty(row, "cells"), `${what}[${index}] cells`).map((cell, cellIndex) =>
      readCell(cell, errors, `${what}[${index}][${cellIndex}]`),
    ),
  );
}

function readCellArray(value: unknown, errors: DbObserverErrors, what: string): DbCell[] {
  return readDataArray(value, what).map((cell, index) =>
    readCell(cell, errors, `${what}[${index}]`),
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

const DECLARED_NAMESPACES: readonly string[] = ["tenant-alpha", "tenant-beta"];

export function createDbObserver(
  domain: ReturnType<typeof createDomainRuntime>,
  errors: DbObserverErrors,
  owner: TestOwner,
) {
  const core = new DbObserverService(DECLARED_NAMESPACES, ADAPTER_LIMITS);
  const returning = new ReturningObserver(ADAPTER_LIMITS);

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

  function cellArray(cells: readonly DbCell[]): unknown {
    return array(cells.map(cellRecord));
  }

  function intArray(values: readonly number[]): unknown {
    return array(values.map((n) => BigInt(n)));
  }

  function strArray(values: readonly string[]): unknown {
    return array([...values]);
  }

  function receiptRecord(facts: NamespaceReceipt): unknown {
    return record(errors.namespaceReceipt, [
      ["namespace", facts.namespace],
      ["owner", facts.owner],
      ["handle", facts.handle],
      ["handle_digest", facts.handleDigest],
      ["tables", BigInt(facts.tables)],
    ]);
  }

  function connectionRecord(facts: ConnectionFacts): unknown {
    return record(errors.connectionFacts, [
      ["connection", facts.connection],
      ["namespace", facts.namespace],
      ["owner", facts.owner],
      ["handle_digest", facts.handleDigest],
      ["pinned", facts.pinned],
      ["conversation_open", facts.conversationOpen],
    ]);
  }

  function rowRecord(facts: RowFacts): unknown {
    return record(errors.rowFacts, [
      ["seq", BigInt(facts.seq)],
      ["cells", cellArray(facts.cells)],
      ["digest", facts.digest],
    ]);
  }

  function readRecord(facts: ReadFacts): unknown {
    return record(errors.readFacts, [
      ["connection", facts.connection],
      ["namespace", facts.namespace],
      ["table", facts.table],
      ["rows", array(facts.rows.map(rowRecord))],
      ["row_count", BigInt(facts.rowCount)],
      ["digest", facts.digest],
    ]);
  }

  function seedRecord(facts: { table: string; rows: number }): unknown {
    return record(errors.seedFacts, [
      ["table", facts.table],
      ["rows", BigInt(facts.rows)],
    ]);
  }

  function conversationRecord(facts: { conversation: string }): unknown {
    return record(errors.conversationFacts, [["conversation", facts.conversation]]);
  }

  function rowComparisonRecord(facts: { match: boolean; mismatches: readonly number[] }): unknown {
    return record(errors.rowComparison, [
      ["match", facts.match],
      ["mismatches", intArray(facts.mismatches)],
    ]);
  }

  function compileFactsRecord(facts: CompileRecord): unknown {
    return record(errors.compileFacts, [
      ["compile", facts.compile],
      ["fixture", facts.fixture],
      ["statement_digest", facts.statementDigest],
      ["schema", strArray(facts.schema)],
    ]);
  }

  function returningRowRecord(facts: ReturningRowFacts): unknown {
    return record(errors.returningRowFacts, [
      ["seq", BigInt(facts.seq)],
      ["cells", cellArray(facts.cells)],
      ["digest", facts.digest],
    ]);
  }

  function payloadRecord(facts: ReturningPayloadFacts): unknown {
    return record(errors.returningPayloadFacts, [
      ["compile", facts.compile],
      ["row_count", BigInt(facts.rowCount)],
      ["rows", array(facts.rows.map(returningRowRecord))],
      ["digest", facts.digest],
    ]);
  }

  function finalRowsRecord(facts: FinalRowsFacts): unknown {
    return record(errors.finalRowsFacts, [
      ["compile", facts.compile],
      ["row_count", BigInt(facts.rowCount)],
      ["rows", array(facts.rows.map(returningRowRecord))],
      ["digest", facts.digest],
    ]);
  }

  function returningComparisonRecord(facts: ReturningComparison): unknown {
    return record(errors.returningComparison, [
      ["match", facts.match],
      ["mismatches", intArray(facts.mismatches)],
    ]);
  }

  function creditVerdictRecord(facts: ReturningCreditVerdict): unknown {
    return record(errors.returningCreditVerdict, [
      ["credit_c", facts.creditC],
      ["reason", facts.reason],
    ]);
  }

  function readInt(value: unknown, what: string): number {
    if (typeof value !== "bigint") throw new TypeError(`${what} needs an int`);
    return Number(value);
  }

  function readPayloadRows(
    value: unknown,
    what: string,
  ): {
    compile: string;
    rowCount: number;
    rows: { seq: number; cells: DbCell[]; digest: string }[];
  } {
    const rows = readDataArray(dataProperty(value, "rows"), `${what} rows`).map((row, index) => ({
      seq: index,
      cells: readDataArray(dataProperty(row, "cells"), `${what} cells`).map((cell, cellIndex) =>
        readCell(cell, errors, `${what}[${cellIndex}]`),
      ),
      digest: "",
    }));
    return {
      compile: readStr(dataProperty(value, "compile"), `${what} compile`),
      rowCount: readInt(dataProperty(value, "row_count"), `${what} row_count`),
      rows,
    };
  }

  return Object.freeze({
    async openNamespace(
      ownerHandle: unknown,
      namespace: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "db::open_namespace");
      if (typeof admitted !== "string") return admitted;
      const ns = readStr(namespace, "db::open_namespace namespace");
      return attempt(() => receiptRecord(core.openNamespace(admitted, ns)));
    },

    async receipt(
      ownerHandle: unknown,
      namespace: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "db::receipt");
      if (typeof admitted !== "string") return admitted;
      const ns = readStr(namespace, "db::receipt namespace");
      return attempt(() => receiptRecord(core.receipt(admitted, ns)));
    },

    async closeNamespace(
      ownerHandle: unknown,
      namespace: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<void>> {
      const admitted = admitOwner(ownerHandle, "db::close_namespace");
      if (typeof admitted !== "string") return admitted;
      const ns = readStr(namespace, "db::close_namespace namespace");
      return attempt(() => {
        core.closeNamespace(admitted, ns);
      });
    },

    async seed(
      ownerHandle: unknown,
      namespace: unknown,
      table: unknown,
      rows: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "db::seed");
      if (typeof admitted !== "string") return admitted;
      const ns = readStr(namespace, "db::seed namespace");
      const tab = readStr(table, "db::seed table");
      const seedRows = readSeedRows(rows, errors, "db::seed rows");
      return attempt(() => seedRecord(core.seed(admitted, ns, tab, seedRows)));
    },

    async pinConnection(
      ownerHandle: unknown,
      namespace: unknown,
      connection: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "db::pin_connection");
      if (typeof admitted !== "string") return admitted;
      const ns = readStr(namespace, "db::pin_connection namespace");
      const conn = readStr(connection, "db::pin_connection connection");
      return attempt(() => connectionRecord(core.pinConnection(admitted, ns, conn)));
    },

    async connectionTokenForTest(
      ownerHandle: unknown,
      namespace: unknown,
      connection: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "db::connection_token_for_test");
      if (typeof admitted !== "string") return admitted;
      const ns = readStr(namespace, "db::connection_token_for_test namespace");
      const conn = readStr(connection, "db::connection_token_for_test connection");
      return attempt(() => core.connectionTokenForTest(admitted, ns, conn));
    },

    async unpinConnection(
      ownerHandle: unknown,
      namespace: unknown,
      connection: unknown,
      token: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<void>> {
      const admitted = admitOwner(ownerHandle, "db::unpin_connection");
      if (typeof admitted !== "string") return admitted;
      const ns = readStr(namespace, "db::unpin_connection namespace");
      const conn = readStr(connection, "db::unpin_connection connection");
      const tok = readStr(token, "db::unpin_connection token");
      return attempt(() => {
        core.unpinConnection(admitted, ns, conn, tok);
      });
    },

    async beginRead(
      ownerHandle: unknown,
      namespace: unknown,
      connection: unknown,
      token: unknown,
      table: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "db::begin_read");
      if (typeof admitted !== "string") return admitted;
      const ns = readStr(namespace, "db::begin_read namespace");
      const conn = readStr(connection, "db::begin_read connection");
      const tok = readStr(token, "db::begin_read token");
      const tab = readStr(table, "db::begin_read table");
      return attempt(() => conversationRecord(core.beginRead(admitted, ns, conn, tok, tab)));
    },

    async fetch(
      ownerHandle: unknown,
      namespace: unknown,
      connection: unknown,
      token: unknown,
      table: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "db::fetch");
      if (typeof admitted !== "string") return admitted;
      const ns = readStr(namespace, "db::fetch namespace");
      const conn = readStr(connection, "db::fetch connection");
      const tok = readStr(token, "db::fetch token");
      const tab = readStr(table, "db::fetch table");
      return attempt(() => readRecord(core.fetch(admitted, ns, conn, tok, tab)));
    },

    async endRead(
      ownerHandle: unknown,
      namespace: unknown,
      connection: unknown,
      token: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<void>> {
      const admitted = admitOwner(ownerHandle, "db::end_read");
      if (typeof admitted !== "string") return admitted;
      const ns = readStr(namespace, "db::end_read namespace");
      const conn = readStr(connection, "db::end_read connection");
      const tok = readStr(token, "db::end_read token");
      return attempt(() => {
        core.endRead(admitted, ns, conn, tok);
      });
    },

    async compareRow(
      ownerHandle: unknown,
      stored: unknown,
      claimed: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "db::compare_row");
      if (typeof admitted !== "string") return admitted;
      const have = readCellArray(stored, errors, "db::compare_row stored");
      const want = readCellArray(claimed, errors, "db::compare_row claimed");
      return attempt(() => rowComparisonRecord(core.compareRow(have, want)));
    },

    async recordCompile(
      ownerHandle: unknown,
      fixture: unknown,
      statement: unknown,
      schema: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "db::record_compile");
      if (typeof admitted !== "string") return admitted;
      const fix = readStr(fixture, "db::record_compile fixture");
      const stmt = readStr(statement, "db::record_compile statement");
      const pin = readStrArray(schema, "db::record_compile schema");
      return attempt(() => compileFactsRecord(returning.recordCompile(fix, stmt, pin)));
    },

    async compileRecord(
      ownerHandle: unknown,
      compile: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "db::compile_record");
      if (typeof admitted !== "string") return admitted;
      const id = readStr(compile, "db::compile_record compile");
      return attempt(() => compileFactsRecord(returning.compileRecord(id)));
    },

    async recordReturning(
      ownerHandle: unknown,
      compile: unknown,
      rows: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "db::record_returning");
      if (typeof admitted !== "string") return admitted;
      const id = readStr(compile, "db::record_returning compile");
      const payload = readSeedRows(rows, errors, "db::record_returning rows");
      return attempt(() => payloadRecord(returning.recordReturning(id, payload)));
    },

    async payloadFacts(
      ownerHandle: unknown,
      compile: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "db::payload_facts");
      if (typeof admitted !== "string") return admitted;
      const id = readStr(compile, "db::payload_facts compile");
      return attempt(() => payloadRecord(returning.payloadFacts(id)));
    },

    async recordFinalRows(
      ownerHandle: unknown,
      compile: unknown,
      rows: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "db::record_final_rows");
      if (typeof admitted !== "string") return admitted;
      const id = readStr(compile, "db::record_final_rows compile");
      const final = readSeedRows(rows, errors, "db::record_final_rows rows");
      return attempt(() => finalRowsRecord(returning.recordFinalRows(id, final)));
    },

    async finalFacts(
      ownerHandle: unknown,
      compile: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "db::final_facts");
      if (typeof admitted !== "string") return admitted;
      const id = readStr(compile, "db::final_facts compile");
      return attempt(() => finalRowsRecord(returning.finalFacts(id)));
    },

    async comparePayloadToFinal(
      ownerHandle: unknown,
      payload: unknown,
      final: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "db::compare_payload_to_final");
      if (typeof admitted !== "string") return admitted;
      const have = readPayloadRows(payload, "db::compare_payload_to_final payload");
      const want = readPayloadRows(final, "db::compare_payload_to_final final");
      return attempt(() =>
        returningComparisonRecord(
          returning.comparePayloadToFinal(
            { compile: have.compile, rowCount: have.rowCount, rows: have.rows, digest: "" },
            { compile: want.compile, rowCount: want.rowCount, rows: want.rows, digest: "" },
          ),
        ),
      );
    },

    async compareClaimedRow(
      ownerHandle: unknown,
      stored: unknown,
      claimed: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "db::compare_claimed_row");
      if (typeof admitted !== "string") return admitted;
      const have = readCellArray(stored, errors, "db::compare_claimed_row stored");
      const want = readCellArray(claimed, errors, "db::compare_claimed_row claimed");
      return attempt(() => returningComparisonRecord(returning.compareClaimedRow(have, want)));
    },

    async creditVerdict(
      ownerHandle: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "db::credit_verdict");
      if (typeof admitted !== "string") return admitted;
      return attempt(() => creditVerdictRecord(returning.creditVerdict()));
    },
  });
}
