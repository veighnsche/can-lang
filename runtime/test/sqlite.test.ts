import { describe, expect, test } from "bun:test";
import { createHash } from "node:crypto";
import { mkdtemp, rm } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, domainFailureDiagnostics, type FailureShape } from "../domain.ts";
import { success, value, type Completion } from "../completion.ts";
import { standardFailureDiagnostics, type StandardFailure } from "../failure.ts";
import { assertionContext } from "../assert/context.ts";
import { record, dataProperty, recordIdentity } from "../data.ts";
import { ownBytes, isBytes, copyBytes } from "../bytes.ts";

const bytesOrigin = { source: "test:sqlite", start: 0, end: 0, invocation: [] };
import { runOwnedRoot, resourceStatus } from "../owner.ts";
import { createSQLDescriptors, type SQLDescriptorEntry } from "../platform/sql/descriptor.ts";
import { createSQLPools, isSQLPoolValue } from "../platform/sql/pool.ts";
import { createSQLTransactions } from "../platform/sql/transaction.ts";
import type { SQLPlan } from "../platform/sql/values.ts";

async function owned(body: () => Promise<void>): Promise<void> {
  const result = await runOwnedRoot(async () => { await body(); return success(undefined); });
  expect(result.cleanupFailed).toBe(false);
  expect(result.completion.kind).toBe("ok");
}
const identity = (kind: string, declaration: string) =>
  createHash("sha256").update("can-concrete-type-v1\0" + JSON.stringify([kind, declaration])).digest("hex");
const textShape: FailureShape = { identity: identity("primitive", "str"), kind: "primitive", declaration: "str", arguments: [], fields: [], leaves: [], inputs: [], errors: [] };
const intShape: FailureShape = { identity: identity("primitive", "int"), kind: "primitive", declaration: "int", arguments: [], fields: [], leaves: [], inputs: [], errors: [] };
const fieldTypes: Record<string, Record<string, string>> = {
  "http::credentials_missing": { variable: textShape.identity },
  "sql::connection_failed": { phase: textShape.identity },
  "sql::query_failed": { operation: textShape.identity, code: textShape.identity },
  "sql::row_missing": { query: textShape.identity },
  "sql::row_count": { query: textShape.identity, actual: intShape.identity },
  "sql::schema_mismatch": { path: textShape.identity, reason: textShape.identity },
  "sql::constraint_failed": { constraint: textShape.identity },
  "sql::close_failed": { reason: textShape.identity },
  "sql::row_limit": { limit: intShape.identity },
  "sql::unsupported_value": { path: textShape.identity, reason: textShape.identity },
  "sql::transaction_failed": { phase: textShape.identity },
  "sql::commit_unknown": { transaction_id: textShape.identity },
};
const declarations = catalogue.errors
  .filter(e => fieldTypes[e.name] !== undefined)
  .map(e => ({ identity: e.identity, name: e.name, id: e.id, parameters: 0 }));
const errorShapes: FailureShape[] = declarations.map(e => ({
  identity: identity("error", e.identity), kind: "error", declaration: e.identity, arguments: [],
  fields: Object.entries(fieldTypes[e.name]!).map(([name, type]) => ({ name, type })),
  leaves: [], inputs: [], errors: [],
}));
const domain = createDomainRuntime({ declarations, shapes: [textShape, intShape, ...errorShapes] });
const id = (declaration: string) => identity("error", declaration);
const optionsName = "can.std.sql@1::sqlite_file_options";
const fileOptions = (mode: string, ms: bigint) => record(optionsName, [["mode", mode], ["busy_timeout_ms", ms]]);

// Hand-built sqlite descriptors (dialect sqlite, grammar version 15).
// DDL runs through execute descriptors: the factory shapes segments,
// never the SQL text, so tests bootstrap their own schema.
const D = "sqlite" as const;
const coverColumns = "id, flag, ratio, name, payload, note";
const table: Record<string, Record<string, SQLDescriptorEntry>> = {
  "": {
    setup_cover: {
      dialect: D, cardinality: "execute", kind: "create_table_statement",
      segments: [{ text: "CREATE TABLE cover (id INTEGER PRIMARY KEY, flag INTEGER, ratio REAL, name TEXT, payload BLOB, note TEXT)" }],
      params: [], paramType: "p", rowType: "r", limit: 0, total: 0, version: 15,
    },
    insert_cover: {
      dialect: D, cardinality: "execute", kind: "insert_statement",
      segments: [{ text: "INSERT INTO cover VALUES (" }, { param: 1 }, { text: ", " }, { param: 2 }, { text: ", " }, { param: 3 }, { text: ", " }, { param: 4 }, { text: ", " }, { param: 5 }, { text: ", " }, { param: 6 }, { text: ")" }],
      params: ["id", "flag", "ratio", "name", "payload", "note"], paramType: "p", rowType: "r", limit: 0, total: 6, version: 15,
    },
    cover_by_id: {
      dialect: D, cardinality: "one", kind: "select_statement",
      segments: [{ text: `SELECT ${coverColumns} FROM cover WHERE id = ` }, { param: 1 }, { text: " LIMIT " }, { param: 2 }],
      params: ["id"], paramType: "p", rowType: "r", limit: 2, total: 2, version: 15,
    },
    cover_all: {
      dialect: D, cardinality: "many", kind: "select_statement",
      segments: [{ text: `SELECT ${coverColumns} FROM cover LIMIT ` }, { param: 1 }],
      params: [], paramType: "p", rowType: "r", limit: 1, total: 1, version: 15,
    },
    update_flag: {
      dialect: D, cardinality: "execute", kind: "update_statement",
      segments: [{ text: "UPDATE cover SET flag = " }, { param: 1 }, { text: " WHERE id = " }, { param: 2 }],
      params: ["flag", "id"], paramType: "p", rowType: "r", limit: 0, total: 2, version: 15,
    },
    delete_cover: {
      dialect: D, cardinality: "execute", kind: "delete_statement",
      segments: [{ text: "DELETE FROM cover WHERE id = " }, { param: 1 }],
      params: ["id"], paramType: "p", rowType: "r", limit: 0, total: 1, version: 15,
    },
    setup_unique: {
      dialect: D, cardinality: "execute", kind: "create_table_statement",
      segments: [{ text: "CREATE TABLE uq (id INTEGER PRIMARY KEY, v TEXT UNIQUE)" }],
      params: [], paramType: "p", rowType: "r", limit: 0, total: 0, version: 15,
    },
    insert_unique: {
      dialect: D, cardinality: "execute", kind: "insert_statement",
      segments: [{ text: "INSERT INTO uq VALUES (" }, { param: 1 }, { text: ", " }, { param: 2 }, { text: ")" }],
      params: ["id", "v"], paramType: "p", rowType: "r", limit: 0, total: 2, version: 15,
    },
    broken_syntax: {
      dialect: D, cardinality: "execute", kind: "insert_statement",
      segments: [{ text: "INSERT INTO cover VALUES (" }],
      params: [], paramType: "p", rowType: "r", limit: 0, total: 0, version: 15,
    },
    drop_absent: {
      dialect: D, cardinality: "execute", kind: "delete_statement",
      segments: [{ text: "DELETE FROM absent" }], params: [], paramType: "p", rowType: "r", limit: 0, total: 0, version: 15,
    },
    journal_delete: {
      dialect: D, cardinality: "execute", kind: "pragma_statement",
      segments: [{ text: "PRAGMA journal_mode = DELETE" }], params: [], paramType: "p", rowType: "r", limit: 0, total: 0, version: 15,
    },
    begin_immediate: {
      dialect: D, cardinality: "execute", kind: "begin_statement",
      segments: [{ text: "BEGIN IMMEDIATE" }], params: [], paramType: "p", rowType: "r", limit: 0, total: 0, version: 15,
    },
    rollback_txn: {
      dialect: D, cardinality: "execute", kind: "rollback_statement",
      segments: [{ text: "ROLLBACK" }], params: [], paramType: "p", rowType: "r", limit: 0, total: 0, version: 15,
    },
    setup_parent: {
      dialect: D, cardinality: "execute", kind: "create_table_statement",
      segments: [{ text: "CREATE TABLE parent (id INTEGER PRIMARY KEY)" }], params: [], paramType: "p", rowType: "r", limit: 0, total: 0, version: 15,
    },
    setup_child: {
      dialect: D, cardinality: "execute", kind: "create_table_statement",
      segments: [{ text: "CREATE TABLE child (id INTEGER PRIMARY KEY, pid INTEGER REFERENCES parent(id) DEFERRABLE INITIALLY DEFERRED)" }], params: [], paramType: "p", rowType: "r", limit: 0, total: 0, version: 15,
    },
    pragma_fk: {
      dialect: D, cardinality: "execute", kind: "pragma_statement",
      segments: [{ text: "PRAGMA foreign_keys = ON" }], params: [], paramType: "p", rowType: "r", limit: 0, total: 0, version: 15,
    },
    insert_child: {
      dialect: D, cardinality: "execute", kind: "insert_statement",
      segments: [{ text: "INSERT INTO child (id, pid) VALUES (" }, { param: 1 }, { text: ", " }, { param: 2 }, { text: ")" }],
      params: ["id", "pid"], paramType: "p", rowType: "r", limit: 0, total: 2, version: 15,
    },
    child_by_id: {
      dialect: D, cardinality: "one", kind: "select_statement",
      segments: [{ text: "SELECT id, pid FROM child WHERE id = " }, { param: 1 }, { text: " LIMIT " }, { param: 2 }],
      params: ["id"], paramType: "p", rowType: "r", limit: 2, total: 2, version: 15,
    },
  },
};
const emptyParams: SQLPlan = { params: { root: "app::empty", fields: [] } };
const coverParams: SQLPlan = {
  params: { root: "app::cover_parameters", fields: [{ name: "id", kind: "int" }] },
  rows: {
    root: "app::cover_row",
    fields: [
      { name: "id", kind: "int" },
      { name: "flag", kind: "bool" },
      { name: "ratio", kind: "float" },
      { name: "name", kind: "str" },
      { name: "payload", kind: "bytes" },
      { name: "note", kind: "option", inner: "str", some: "app::option_some", none: "app::option_none" },
    ],
  },
};
const insertParams: SQLPlan = {
  params: {
    root: "app::cover_insert", fields: [
      { name: "id", kind: "int" },
      { name: "flag", kind: "bool" },
      { name: "ratio", kind: "float" },
      { name: "name", kind: "str" },
      { name: "payload", kind: "bytes" },
      { name: "note", kind: "option", inner: "str", some: "app::option_some", none: "app::option_none" },
    ],
  },
};
const childInsertParams: SQLPlan = {
  params: { root: "app::child_insert", fields: [{ name: "id", kind: "int" }, { name: "pid", kind: "int" }] },
};
const childParams: SQLPlan = {
  params: { root: "app::child_parameters", fields: [{ name: "id", kind: "int" }] },
  rows: { root: "app::child_row", fields: [{ name: "id", kind: "int" }, { name: "pid", kind: "int" }] },
};
const descriptors = createSQLDescriptors(table);
const poolContracts = {
  credentialsMissing: id("can.std.http@1::credentials_missing"),
  connectionFailed: id("can.std.sql@1::connection_failed"),
  queryFailed: id("can.std.sql@1::query_failed"),
  rowMissing: id("can.std.sql@1::row_missing"),
  rowCount: id("can.std.sql@1::row_count"),
  schemaMismatch: id("can.std.sql@1::schema_mismatch"),
  constraintFailed: id("can.std.sql@1::constraint_failed"),
  closeFailed: id("can.std.sql@1::close_failed"),
  rowLimit: id("can.std.sql@1::row_limit"),
  unsupportedValue: id("can.std.sql@1::unsupported_value"),
};
const pools = createSQLPools(domain, poolContracts, () => undefined, descriptors);
const transactions = createSQLTransactions(domain, {
  connectionFailed: id("can.std.sql@1::connection_failed"),
  queryFailed: id("can.std.sql@1::query_failed"),
  rowMissing: id("can.std.sql@1::row_missing"),
  rowCount: id("can.std.sql@1::row_count"),
  schemaMismatch: id("can.std.sql@1::schema_mismatch"),
  constraintFailed: id("can.std.sql@1::constraint_failed"),
  rowLimit: id("can.std.sql@1::row_limit"),
  unsupportedValue: id("can.std.sql@1::unsupported_value"),
  transactionFailed: id("can.std.sql@1::transaction_failed"),
  commitUnknown: id("can.std.sql@1::commit_unknown"),
}, descriptors);
const COMMIT = "app::commit";
const ROLLBACK = "app::rollback";
const leaves = { commit: COMMIT, rollback: ROLLBACK };

function domainOutcome(completion: Completion<unknown>, name: string): Record<string, unknown> {
  expect(completion.kind).toBe("domain");
  if (completion.kind !== "domain") throw new Error("wrong outcome");
  const details = domainFailureDiagnostics(completion.value);
  expect(details.declaration.name).toBe(name);
  const payload = details.payload as Record<string, unknown>;
  const plain: Record<string, unknown> = {};
  for (const key of Object.keys(payload)) plain[key] = payload[key];
  return plain;
}
function fixtureContext() {
  return assertionContext({ package: "app", declaration: "probe", name: "probe" });
}
const MIN_INT64 = -(1n << 63n);
const MAX_INT64 = (1n << 63n) - 1n;

describe("sqlite pools", () => {
  test("memory pools round-trip the whole scalar cover exactly", async () => {
    await owned(async () => {
      const token = value(await pools.sqliteOpenMemory());
      expect(isSQLPoolValue("pool", token)).toBe(true);
      const setup = descriptors.declareDescriptor("", "setup_cover");
      const insert = descriptors.declareDescriptor("", "insert_cover");
      const byId = descriptors.declareDescriptor("", "cover_by_id");
      value(await pools.execute(setup, emptyParams, token, record("app::empty", [])));
      const payload = ownBytes(new Uint8Array([1, 2, 250]));
      const rows: [bigint, boolean, number, string, unknown, unknown][] = [
        [9007199254740993n, true, 1.5, "héllo", payload, record("app::option_some", [["value", "first"]])],
        [MIN_INT64, false, -0.0, "", ownBytes(new Uint8Array(0)), record("app::option_none", [])],
        [MAX_INT64, true, 3, "x", payload, record("app::option_none", [])],
      ];
      for (const [idn, flag, ratio, name, bytes, note] of rows) {
        const affected = value(await pools.execute(insert, insertParams, token,
          record("app::cover_insert", [["id", idn], ["flag", flag], ["ratio", ratio], ["name", name], ["payload", bytes], ["note", note]])));
        expect(affected).toBe(1n);
      }
      for (const [idn, flag, ratio, name, bytes, note] of rows) {
        const row = value(await pools.queryOne(byId, coverParams, token, record("app::cover_parameters", [["id", idn]])));
        expect(dataProperty(row, "id")).toBe(idn);
        expect(dataProperty(row, "flag")).toBe(flag);
        // SQLite REAL normalizes -0.0 to 0 (observed through Bun.SQL);
        // value equality still holds, only the signed zero is lost.
        expect(dataProperty(row, "ratio")).toBe(Object.is(ratio, -0) ? 0 : ratio);
        expect(dataProperty(row, "name")).toBe(name);
        const back = dataProperty(row, "payload");
        expect(isBytes(back)).toBe(true);
        expect(recordIdentity(dataProperty(row, "note"))).toBe(recordIdentity(note));
        if (recordIdentity(note) === "app::option_some") {
          expect(dataProperty(dataProperty(row, "note"), "value")).toBe(dataProperty(note, "value"));
        }
        void bytes;
      }
      // Encoded bytes are copied: mutating the source afterwards cannot move the row.
      const probe = new Uint8Array([9, 9, 9]);
      value(await pools.execute(insert, insertParams, token,
        record("app::cover_insert", [["id", 7n], ["flag", true], ["ratio", 1], ["name", "m"], ["payload", ownBytes(probe)], ["note", record("app::option_none", [])]])));
      probe.fill(0);
      const row = value(await pools.queryOne(byId, coverParams, token, record("app::cover_parameters", [["id", 7n]])));
      expect([...copyBytes(dataProperty(row, "payload"), bytesOrigin)]).toEqual([9, 9, 9]);
      value(await pools.close(token, 1000n));
    });
  });
  test("execute reports exact affected counts including zero", async () => {
    await owned(async () => {
      const token = value(await pools.sqliteOpenMemory());
      const setup = descriptors.declareDescriptor("", "setup_cover");
      const insert = descriptors.declareDescriptor("", "insert_cover");
      const update = descriptors.declareDescriptor("", "update_flag");
      const remove = descriptors.declareDescriptor("", "delete_cover");
      value(await pools.execute(setup, emptyParams, token, record("app::empty", [])));
      const params = (idn: bigint) => record("app::cover_insert", [["id", idn], ["flag", false], ["ratio", 0], ["name", "n"], ["payload", ownBytes(new Uint8Array(0))], ["note", record("app::option_none", [])]]);
      expect(value(await pools.execute(insert, insertParams, token, params(1n)))).toBe(1n);
      expect(value(await pools.execute(insert, insertParams, token, params(2n)))).toBe(1n);
      const flagPlan: SQLPlan = { params: { root: "app::flag_parameters", fields: [{ name: "flag", kind: "bool" }, { name: "id", kind: "int" }] } };
      expect(value(await pools.execute(update, flagPlan, token, record("app::flag_parameters", [["flag", true], ["id", 1n]])))).toBe(1n);
      const idPlan: SQLPlan = { params: { root: "app::id_only", fields: [{ name: "id", kind: "int" }] } };
      expect(value(await pools.execute(remove, idPlan, token, record("app::id_only", [["id", 999n]])))).toBe(0n);
      expect(value(await pools.execute(remove, idPlan, token, record("app::id_only", [["id", 1n]])))).toBe(1n);
      // query_rows binds max_rows+1 against the real engine and enforces the bound.
      const all = descriptors.declareDescriptor("", "cover_all");
      const rowsPlan: SQLPlan = { ...coverParams, params: { root: "app::empty", fields: [] } };
      expect((value(await pools.queryRows(all, rowsPlan, token, record("app::empty", []), 5n)) as unknown[]).length).toBe(1);
      expect(domainOutcome(await pools.queryRows(all, rowsPlan, token, record("app::empty", []), 0n), "sql::row_limit")).toEqual({ limit: 0n });
      value(await pools.close(token, 1000n));
    });
  });
  test("file pools persist across close and reopen with enforced modes", async () => {
    const directory = await mkdtemp(join(tmpdir(), "can-sqlite-"));
    try {
      const filename = join(directory, "store.sqlite");
      await owned(async () => {
        const token = value(await pools.sqliteOpenFile(filename, fileOptions("rwc", 100n)));
        const setup = descriptors.declareDescriptor("", "setup_cover");
        const insert = descriptors.declareDescriptor("", "insert_cover");
        value(await pools.execute(setup, emptyParams, token, record("app::empty", [])));
        value(await pools.execute(insert, insertParams, token,
          record("app::cover_insert", [["id", 11n], ["flag", true], ["ratio", 2], ["name", "kept"], ["payload", ownBytes(new Uint8Array([5]))], ["note", record("app::option_none", [])]])));
        value(await pools.close(token, 1000n));
        const reopened = value(await pools.sqliteOpenFile(filename, fileOptions("ro", 0n)));
        const byId = descriptors.declareDescriptor("", "cover_by_id");
        const row = value(await pools.queryOne(byId, coverParams, reopened, record("app::cover_parameters", [["id", 11n]])));
        expect(dataProperty(row, "name")).toBe("kept");
        // A read-only handle refuses writes with the structured readonly code.
        expect(domainOutcome(await pools.execute(insert, insertParams, reopened,
          record("app::cover_insert", [["id", 12n], ["flag", false], ["ratio", 0], ["name", "n"], ["payload", ownBytes(new Uint8Array(0))], ["note", record("app::option_none", [])]])),
          "sql::query_failed")).toEqual({ operation: "execute", code: "SQLITE_READONLY" });
        value(await pools.close(reopened, 1000n));
        // Missing files refuse at connect for ro/rw, and create for rwc.
        expect(domainOutcome(await pools.sqliteOpenFile(join(directory, "absent.sqlite"), fileOptions("ro", 0n)), "sql::connection_failed")).toEqual({ phase: "connect" });
        expect(domainOutcome(await pools.sqliteOpenFile(join(directory, "absent.sqlite"), fileOptions("rw", 0n)), "sql::connection_failed")).toEqual({ phase: "connect" });
        const created = value(await pools.sqliteOpenFile(join(directory, "fresh.sqlite"), fileOptions("rwc", 0n)));
        value(await pools.close(created, 1000n));
        // Bad configuration never reaches the filesystem.
        expect(domainOutcome(await pools.sqliteOpenFile("", fileOptions("rwc", 0n)), "sql::connection_failed")).toEqual({ phase: "config" });
        expect(domainOutcome(await pools.sqliteOpenFile(filename, fileOptions("wide", 0n)), "sql::connection_failed")).toEqual({ phase: "config" });
        expect(domainOutcome(await pools.sqliteOpenFile(filename, fileOptions("rwc", -1n)), "sql::connection_failed")).toEqual({ phase: "config" });
        expect(domainOutcome(await pools.sqliteOpenFile(filename, fileOptions("rwc", 2147483648n)), "sql::connection_failed")).toEqual({ phase: "config" });
        await expect(pools.sqliteOpenFile(7, fileOptions("rwc", 0n))).rejects.toThrow(TypeError);
      });
    } finally {
      await rm(directory, { recursive: true, force: true });
    }
  });
  test("dialect mismatch fails before any native call", async () => {
    await owned(async () => {
      const token = value(await pools.sqliteOpenMemory());
      const pg: SQLDescriptorEntry = {
        dialect: "postgresql", cardinality: "execute", kind: "InsertStmt",
        segments: [{ text: "INSERT INTO t VALUES (1)" }], params: [], paramType: "p", rowType: "r", limit: 0, total: 0, version: 170007,
      };
      const mixed = createSQLDescriptors({ "": { pg_insert: pg } });
      // A postgres descriptor on a sqlite pool: the agreement check fires,
      // so no native call observes postgres text.
      expect(domainOutcome(await pools.execute(mixed.declareDescriptor("", "pg_insert"), emptyParams, token, record("app::empty", [])),
        "sql::query_failed")).toEqual({ operation: "execute", code: "dialect_mismatch" });
      value(await pools.close(token, 1000n));
    });
    // The mirror direction with a recording fake: zero native calls.
    const RealSQL = Bun.SQL;
    const calls: unknown[][] = [];
    try {
      (Bun as unknown as { SQL: unknown }).SQL = function () {
        const callable = async function (...args: unknown[]) { calls.push(args); return []; };
        Object.assign(callable, { connect: async () => {}, close: async () => {} });
        return callable;
      };
      const fakePools = createSQLPools(domain, poolContracts, () => "postgres://fake/x", descriptors);
      await owned(async () => {
        const pgToken = value(await fakePools.open("CAN_X", 1n));
        const lite = descriptors.declareDescriptor("", "delete_cover");
        const idPlan: SQLPlan = { params: { root: "app::id_only", fields: [{ name: "id", kind: "int" }] } };
        expect(domainOutcome(await fakePools.execute(lite, idPlan, pgToken, record("app::id_only", [["id", 1n]])),
          "sql::query_failed")).toEqual({ operation: "execute", code: "dialect_mismatch" });
        value(await fakePools.close(pgToken, 1000n));
      });
    } finally {
      (Bun as unknown as { SQL: unknown }).SQL = RealSQL;
    }
    expect(calls).toEqual([]);
  });
  test("constraint, syntax, and missing-table failures classify with codes only", async () => {
    await owned(async () => {
      const token = value(await pools.sqliteOpenMemory());
      const setup = descriptors.declareDescriptor("", "setup_unique");
      const insert = descriptors.declareDescriptor("", "insert_unique");
      const broken = descriptors.declareDescriptor("", "broken_syntax");
      const uqPlan: SQLPlan = { params: { root: "app::uq_parameters", fields: [{ name: "id", kind: "int" }, { name: "v", kind: "str" }] } };
      value(await pools.execute(setup, emptyParams, token, record("app::empty", [])));
      value(await pools.execute(insert, uqPlan, token, record("app::uq_parameters", [["id", 1n], ["v", "a"]])));
      // UNIQUE violation: the structured constraint code, never the message.
      expect(domainOutcome(await pools.execute(insert, uqPlan, token, record("app::uq_parameters", [["id", 2n], ["v", "a"]])),
        "sql::constraint_failed")).toEqual({ constraint: "SQLITE_CONSTRAINT_UNIQUE" });
      // Malformed text and missing tables are query failures with codes.
      expect(domainOutcome(await pools.execute(broken, emptyParams, token, record("app::empty", [])),
        "sql::query_failed")).toEqual({ operation: "execute", code: "SQLITE_ERROR" });
      const dropAbsent = descriptors.declareDescriptor("", "drop_absent");
      expect(domainOutcome(await pools.execute(dropAbsent, emptyParams, token, record("app::empty", [])),
        "sql::query_failed")).toEqual({ operation: "execute", code: "SQLITE_ERROR" });
      value(await pools.close(token, 1000n));
    });
  });
  test("lock contention surfaces SQLITE_BUSY without blocking the loop", async () => {
    const directory = await mkdtemp(join(tmpdir(), "can-sqlite-busy-"));
    try {
      const filename = join(directory, "busy.sqlite");
      await owned(async () => {
        const first = value(await pools.sqliteOpenFile(filename, fileOptions("rwc", 100n)));
        const setup = descriptors.declareDescriptor("", "setup_cover");
        value(await pools.execute(setup, emptyParams, first, record("app::empty", [])));
        // Rollback-journal mode plus an immediate write transaction holds a
        // RESERVED lock; the second pool uses timeout zero so the busy
        // failure is immediate and deterministic.
        const journal = descriptors.declareDescriptor("", "journal_delete");
        const begin = descriptors.declareDescriptor("", "begin_immediate");
        const rollback = descriptors.declareDescriptor("", "rollback_txn");
        value(await pools.execute(journal, emptyParams, first, record("app::empty", [])));
        value(await pools.execute(begin, emptyParams, first, record("app::empty", [])));
        const insert = descriptors.declareDescriptor("", "insert_cover");
        value(await pools.execute(insert, insertParams, first,
          record("app::cover_insert", [["id", 1n], ["flag", false], ["ratio", 0], ["name", "n"], ["payload", ownBytes(new Uint8Array(0))], ["note", record("app::option_none", [])]])));
        const second = value(await pools.sqliteOpenFile(filename, fileOptions("rw", 0n)));
        expect(domainOutcome(await pools.execute(insert, insertParams, second,
          record("app::cover_insert", [["id", 2n], ["flag", false], ["ratio", 0], ["name", "n"], ["payload", ownBytes(new Uint8Array(0))], ["note", record("app::option_none", [])]])),
          "sql::query_failed")).toEqual({ operation: "execute", code: "SQLITE_BUSY" });
        // The event loop stayed live: rollback releases and the write lands.
        value(await pools.execute(rollback, emptyParams, first, record("app::empty", [])));
        expect(value(await pools.execute(insert, insertParams, second,
          record("app::cover_insert", [["id", 2n], ["flag", false], ["ratio", 0], ["name", "n"], ["payload", ownBytes(new Uint8Array(0))], ["note", record("app::option_none", [])]])))).toBe(1n);
        value(await pools.close(first, 1000n));
        value(await pools.close(second, 1000n));
      });
    } finally {
      await rm(directory, { recursive: true, force: true });
    }
  });
  test("sqlite transactions commit across awaits and roll back", async () => {
    await owned(async () => {
      const token = value(await pools.sqliteOpenMemory());
      const setup = descriptors.declareDescriptor("", "setup_cover");
      const insert = descriptors.declareDescriptor("", "insert_cover");
      const byId = descriptors.declareDescriptor("", "cover_by_id");
      value(await pools.execute(setup, emptyParams, token, record("app::empty", [])));
      const committed = value(await transactions.withTransaction(token, async (handle: unknown) => {
        value(await transactions.execute(insert, insertParams, handle,
          record("app::cover_insert", [["id", 5n], ["flag", true], ["ratio", 1], ["name", "tx"], ["payload", ownBytes(new Uint8Array([8]))], ["note", record("app::option_none", [])]])));
        await Bun.sleep(5);
        const row = value(await transactions.queryOne(byId, coverParams, handle, record("app::cover_parameters", [["id", 5n]])));
        expect(dataProperty(row, "name")).toBe("tx");
        return success(record(COMMIT, [["value", dataProperty(row, "id")]]));
      }, leaves));
      expect(committed).toBe(5n);
      const seen = value(await pools.queryOne(byId, coverParams, token, record("app::cover_parameters", [["id", 5n]])));
      expect(dataProperty(seen, "name")).toBe("tx");
      const rolled = value(await transactions.withTransaction(token, async (handle: unknown) => {
        value(await transactions.execute(insert, insertParams, handle,
          record("app::cover_insert", [["id", 6n], ["flag", false], ["ratio", 0], ["name", "gone"], ["payload", ownBytes(new Uint8Array(0))], ["note", record("app::option_none", [])]])));
        return success(record(ROLLBACK, [["value", 6n]]));
      }, leaves));
      expect(rolled).toBe(6n);
      expect(domainOutcome(await pools.queryOne(byId, coverParams, token, record("app::cover_parameters", [["id", 6n]])), "sql::row_missing")).toEqual({ query: "cover_by_id" });
      // A domain failure inside the callback propagates and rolls back.
      const failed = await transactions.withTransaction(token, async (handle: unknown) => {
        value(await transactions.execute(insert, insertParams, handle,
          record("app::cover_insert", [["id", 8n], ["flag", false], ["ratio", 0], ["name", "gone"], ["payload", ownBytes(new Uint8Array(0))], ["note", record("app::option_none", [])]])));
        return transactions.queryOne(byId, coverParams, handle, record("app::cover_parameters", [["id", 999n]]));
      }, leaves);
      expect(domainOutcome(failed, "sql::row_missing")).toEqual({ query: "cover_by_id" });
      expect(domainOutcome(await pools.queryOne(byId, coverParams, token, record("app::cover_parameters", [["id", 8n]])), "sql::row_missing")).toEqual({ query: "cover_by_id" });
      value(await pools.close(token, 1000n));
    });
  });
  test("sqlite transaction body standard failure rolls back and propagates", async () => {
    await owned(async () => {
      const token = value(await pools.sqliteOpenMemory());
      const setup = descriptors.declareDescriptor("", "setup_cover");
      const insert = descriptors.declareDescriptor("", "insert_cover");
      const byId = descriptors.declareDescriptor("", "cover_by_id");
      value(await pools.execute(setup, emptyParams, token, record("app::empty", [])));
      const outcome = await transactions.withTransaction(token, async (handle: unknown) => {
        value(await transactions.execute(insert, insertParams, handle,
          record("app::cover_insert", [["id", 20n], ["flag", false], ["ratio", 0], ["name", "boom"], ["payload", ownBytes(new Uint8Array(0))], ["note", record("app::option_none", [])]])));
        throw new Error("boom");
      }, leaves);
      expect(outcome.kind).toBe("standard");
      if (outcome.kind !== "standard") throw new Error("wrong outcome");
      expect(standardFailureDiagnostics(outcome.value).kind).toBe("native_exception");
      expect(domainOutcome(await pools.queryOne(byId, coverParams, token, record("app::cover_parameters", [["id", 20n]])), "sql::row_missing")).toEqual({ query: "cover_by_id" });
      value(await pools.close(token, 1000n));
    });
  });
  test("sqlite commit failure reports commit-unknown and lands nothing", async () => {
    await owned(async () => {
      const token = value(await pools.sqliteOpenMemory());
      const setupParent = descriptors.declareDescriptor("", "setup_parent");
      const setupChild = descriptors.declareDescriptor("", "setup_child");
      const pragma = descriptors.declareDescriptor("", "pragma_fk");
      const insert = descriptors.declareDescriptor("", "insert_child");
      const byId = descriptors.declareDescriptor("", "child_by_id");
      value(await pools.execute(setupParent, emptyParams, token, record("app::empty", [])));
      value(await pools.execute(setupChild, emptyParams, token, record("app::empty", [])));
      value(await pools.execute(pragma, emptyParams, token, record("app::empty", [])));
      // The deferred foreign key holds through the write and fails at
      // COMMIT: the native layer rejects after the commit decision, so
      // the outcome is commit-unknown with the safe attempt identity.
      const outcome = await transactions.withTransaction(token, async (handle: unknown) => {
        value(await transactions.execute(insert, childInsertParams, handle,
          record("app::child_insert", [["id", 1n], ["pid", 999n]])));
        return success(record(COMMIT, [["value", 1n]]));
      }, leaves);
      const payload = domainOutcome(outcome, "sql::commit_unknown");
      expect(typeof payload["transaction_id"]).toBe("string");
      expect(payload["transaction_id"] as string).toMatch(/^\d+-\d+$/);
      expect(domainOutcome(await pools.queryOne(byId, childParams, token, record("app::child_parameters", [["id", 1n]])), "sql::row_missing")).toEqual({ query: "child_by_id" });
      value(await pools.close(token, 1000n));
    });
  });
  test("sqlite close during a leased transaction preserves the primary", async () => {
    await owned(async () => {
      const token = value(await pools.sqliteOpenMemory());
      const setup = descriptors.declareDescriptor("", "setup_cover");
      const insert = descriptors.declareDescriptor("", "insert_cover");
      const byId = descriptors.declareDescriptor("", "cover_by_id");
      value(await pools.execute(setup, emptyParams, token, record("app::empty", [])));
      const tx = transactions.withTransaction(token, async (handle: unknown) => {
        value(await transactions.execute(insert, insertParams, handle,
          record("app::cover_insert", [["id", 30n], ["flag", true], ["ratio", 1], ["name", "slow"], ["payload", ownBytes(new Uint8Array(0))], ["note", record("app::option_none", [])]])));
        await Bun.sleep(50);
        const row = value(await transactions.queryOne(byId, coverParams, handle, record("app::cover_parameters", [["id", 30n]])));
        expect(dataProperty(row, "name")).toBe("slow");
        return success(record(COMMIT, [["value", 30n]]));
      }, leaves);
      await Bun.sleep(10);
      expect(value(await pools.close(token, 5000n))).toBe(undefined);
      expect(value(await tx)).toBe(30n);
      expect(resourceStatus(token).state).toBe("closed");
    });
  });
  test("sqlite close timeout keeps the commit and records cleanup", async () => {
    const directory = await mkdtemp(join(tmpdir(), "can-sqlite-txclose-"));
    try {
      const filename = join(directory, "tx.sqlite");
      const diags: unknown[] = [];
      const result = await runOwnedRoot(async () => {
        const token = value(await pools.sqliteOpenFile(filename, fileOptions("rwc", 5000n)));
        const setup = descriptors.declareDescriptor("", "setup_cover");
        const insert = descriptors.declareDescriptor("", "insert_cover");
        const byId = descriptors.declareDescriptor("", "cover_by_id");
        value(await pools.execute(setup, emptyParams, token, record("app::empty", [])));
        const tx = transactions.withTransaction(token, async (handle: unknown) => {
          value(await transactions.execute(insert, insertParams, handle,
            record("app::cover_insert", [["id", 31n], ["flag", true], ["ratio", 1], ["name", "slow"], ["payload", ownBytes(new Uint8Array(0))], ["note", record("app::option_none", [])]])));
          await Bun.sleep(100);
          return success(record(COMMIT, [["value", 31n]]));
        }, leaves);
        await Bun.sleep(10);
        expect(domainOutcome(await pools.close(token, 5n), "sql::close_failed")).toEqual({ reason: "timeout" });
        // The timed-out close keeps closing in the background: the
        // primary commits, and a fresh connection reads it back.
        expect(value(await tx)).toBe(31n);
        const reopened = value(await pools.sqliteOpenFile(filename, fileOptions("ro", 0n)));
        const row = value(await pools.queryOne(byId, coverParams, reopened, record("app::cover_parameters", [["id", 31n]])));
        expect(dataProperty(row, "name")).toBe("slow");
        value(await pools.close(reopened, 1000n));
        expect(resourceStatus(token).state).toBe("closed");
        return success(undefined);
      }, (d: unknown) => { diags.push(d); });
      expect(result.completion.kind).toBe("ok");
      expect(result.cleanupFailed).toBe(true);
      expect(diags.some((d) => (d as { phase?: unknown }).phase === "cleanup")).toBe(true);
    } finally {
      await rm(directory, { recursive: true, force: true });
    }
  });
  test("sqlite close validates, closes, and denies later work", async () => {
    await owned(async () => {
      const token = value(await pools.sqliteOpenMemory());
      expect(domainOutcome(await pools.close(token, -1n), "sql::close_failed")).toEqual({ reason: "invalid_timeout" });
      expect(value(await pools.close(token, 1000n))).toBe(undefined);
      try {
        await pools.queryOne(descriptors.declareDescriptor("", "cover_by_id"), coverParams, token, record("app::cover_parameters", [["id", 1n]]));
        throw new Error("query admitted on a closed pool");
      } catch (cause) {
        expect(standardFailureDiagnostics(cause as StandardFailure).kind).toBe("resource_state");
      }
    });
  });
  test("fixture boundaries never open sqlite pools", async () => {
    const context = fixtureContext();
    await owned(async () => {
      await expect(pools.sqliteOpenMemory(context)).rejects.toThrow();
      await expect(pools.sqliteOpenFile("x.sqlite", fileOptions("rwc", 0n), context)).rejects.toThrow();
    });
  });
});
