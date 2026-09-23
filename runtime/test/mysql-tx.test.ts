import { describe, expect, test } from "bun:test";
import { $ } from "bun";
import { createHash } from "node:crypto";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, domainFailureDiagnostics, type FailureShape } from "../domain.ts";
import { success, value, type Completion } from "../completion.ts";
import { record, dataProperty } from "../data.ts";
import { ownBytes } from "../bytes.ts";
import { runOwnedRoot } from "../owner.ts";
import { createSQLDescriptors, type SQLDescriptorEntry } from "../platform/sql/descriptor.ts";
import { createSQLPools } from "../platform/sql/pool.ts";
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

const MYSQL_URL = process.env["CAN_TEST_MYSQL_URL"];
const MYSQL_CONTAINER = process.env["CAN_TEST_MYSQL_CONTAINER"];
const live = test.skipIf(MYSQL_URL === undefined);
const dockerized = test.skipIf(MYSQL_URL === undefined || MYSQL_CONTAINER === undefined);
const lookup = (name: string): string | undefined => (MYSQL_URL !== undefined && name === "CAN_TEST_MYSQL" ? MYSQL_URL : undefined);

const D = "mysql" as const;
const V = 80011;
const table: Record<string, Record<string, SQLDescriptorEntry>> = {
  "": {
    drop_cover: {
      dialect: D, cardinality: "execute", kind: "drop_table_statement",
      segments: [{ text: "DROP TABLE IF EXISTS cover" }], params: [], paramType: "p", rowType: "r", limit: 0, total: 0, version: V,
    },
    setup_cover: {
      dialect: D, cardinality: "execute", kind: "create_table_statement",
      segments: [{ text: "CREATE TABLE cover (id BIGINT PRIMARY KEY, flag TINYINT(1), ratio DOUBLE, name VARCHAR(255), payload BLOB, note TEXT NULL)" }], params: [], paramType: "p", rowType: "r", limit: 0, total: 0, version: V,
    },
    insert_cover: {
      dialect: D, cardinality: "execute", kind: "insert_statement",
      segments: [{ text: "INSERT INTO cover VALUES (" }, { param: 1 }, { text: ", " }, { param: 2 }, { text: ", " }, { param: 3 }, { text: ", " }, { param: 4 }, { text: ", " }, { param: 5 }, { text: ", " }, { param: 6 }, { text: ")" }],
      params: ["id", "flag", "ratio", "name", "payload", "note"], paramType: "p", rowType: "r", limit: 0, total: 6, version: V,
    },
    cover_by_id: {
      dialect: D, cardinality: "one", kind: "select_statement",
      segments: [{ text: "SELECT id, flag, ratio, name, payload, note FROM cover WHERE id = " }, { param: 1 }, { text: " LIMIT " }, { param: 2 }],
      params: ["id"], paramType: "p", rowType: "r", limit: 2, total: 2, version: V,
    },
    update_flag: {
      dialect: D, cardinality: "execute", kind: "update_statement",
      segments: [{ text: "UPDATE cover SET flag = " }, { param: 1 }, { text: " WHERE id = " }, { param: 2 }],
      params: ["flag", "id"], paramType: "p", rowType: "r", limit: 0, total: 2, version: V,
    },
    session_tz: {
      dialect: D, cardinality: "one", kind: "select_statement",
      segments: [{ text: "SELECT @@session.time_zone AS tz LIMIT " }, { param: 1 }],
      params: [], paramType: "p", rowType: "r", limit: 1, total: 1, version: V,
    },
    sleep_query: {
      dialect: D, cardinality: "one", kind: "select_statement",
      segments: [{ text: "SELECT SLEEP(" }, { param: 1 }, { text: ") AS z LIMIT " }, { param: 2 }],
      params: ["secs"], paramType: "p", rowType: "r", limit: 2, total: 2, version: V,
    },
    set_lock_timeout: {
      dialect: D, cardinality: "execute", kind: "set_statement",
      segments: [{ text: "SET SESSION innodb_lock_wait_timeout = 1" }], params: [], paramType: "p", rowType: "r", limit: 0, total: 0, version: V,
    },
    set_wait_timeout: {
      dialect: D, cardinality: "execute", kind: "set_statement",
      segments: [{ text: "SET SESSION wait_timeout = 1" }], params: [], paramType: "p", rowType: "r", limit: 0, total: 0, version: V,
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
const flagParams: SQLPlan = { params: { root: "app::flag_update", fields: [{ name: "flag", kind: "bool" }, { name: "id", kind: "int" }] } };
const tzParams: SQLPlan = { params: { root: "app::empty", fields: [] }, rows: { root: "app::tz_row", fields: [{ name: "tz", kind: "str" }] } };
const sleepParams: SQLPlan = {
  params: { root: "app::sleep_parameters", fields: [{ name: "secs", kind: "float" }] },
  rows: { root: "app::sleep_row", fields: [{ name: "z", kind: "int" }] },
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
const pools = createSQLPools(domain, poolContracts, lookup, descriptors);
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
const mk = (idn: bigint, flag: boolean) => record("app::cover_insert", [["id", idn], ["flag", flag], ["ratio", 0], ["name", "n"], ["payload", ownBytes(new Uint8Array(0))], ["note", record("app::option_none", [])]]);

describe("mysql transactions", () => {
  live("transactions commit across awaits and roll back", async () => {
    await owned(async () => {
      const token = value(await pools.mysqlOpen("CAN_TEST_MYSQL", 2n));
      value(await pools.execute(descriptors.declareDescriptor("", "drop_cover"), emptyParams, token, record("app::empty", [])));
      value(await pools.execute(descriptors.declareDescriptor("", "setup_cover"), emptyParams, token, record("app::empty", [])));
      const insert = descriptors.declareDescriptor("", "insert_cover");
      const byId = descriptors.declareDescriptor("", "cover_by_id");
      expect(value(await transactions.withTransaction(token, async (handle: unknown) => {
        value(await transactions.execute(insert, insertParams, handle, mk(5n, true)));
        await Bun.sleep(20);
        const row = value(await transactions.queryOne(byId, coverParams, handle, record("app::cover_parameters", [["id", 5n]])));
        expect(dataProperty(row, "flag")).toBe(true);
        return success(record(COMMIT, [["value", 5n]]));
      }, leaves))).toBe(5n);
      expect(value(await transactions.withTransaction(token, async (handle: unknown) => {
        value(await transactions.execute(insert, insertParams, handle, mk(8n, false)));
        return success(record(ROLLBACK, [["value", 8n]]));
      }, leaves))).toBe(8n);
      expect(domainOutcome(await pools.queryOne(byId, coverParams, token, record("app::cover_parameters", [["id", 8n]])), "sql::row_missing")).toEqual({ query: "cover_by_id" });
      const dup = await transactions.withTransaction(token, async (handle: unknown) => {
        value(await transactions.execute(insert, insertParams, handle, mk(5n, true)));
        return success(record(COMMIT, [["value", 0n]]));
      }, leaves);
      expect(domainOutcome(dup, "sql::constraint_failed")).toEqual({ constraint: "ER_DUP_ENTRY" });
      value(await pools.close(token, 1000n));
    });
  });
  live("opposite-order updates deadlock exactly one attempt", async () => {
    await owned(async () => {
      const first = value(await pools.mysqlOpen("CAN_TEST_MYSQL", 2n));
      const second = value(await pools.mysqlOpen("CAN_TEST_MYSQL", 2n));
      value(await pools.execute(descriptors.declareDescriptor("", "drop_cover"), emptyParams, first, record("app::empty", [])));
      value(await pools.execute(descriptors.declareDescriptor("", "setup_cover"), emptyParams, first, record("app::empty", [])));
      const insert = descriptors.declareDescriptor("", "insert_cover");
      const update = descriptors.declareDescriptor("", "update_flag");
      const flag = (on: boolean, idn: bigint) => record("app::flag_update", [["flag", on], ["id", idn]]);
      value(await pools.execute(insert, insertParams, first, mk(101n, false)));
      value(await pools.execute(insert, insertParams, first, mk(102n, false)));
      let aLocked!: () => void;
      let bLocked!: () => void;
      const aGate = new Promise<void>(resolve => { aLocked = resolve; });
      const bGate = new Promise<void>(resolve => { bLocked = resolve; });
      const attempt = (token: unknown, mine: bigint, theirs: bigint, held: () => void, wait: Promise<void>) =>
        transactions.withTransaction(token, async (handle: unknown) => {
          value(await transactions.execute(update, flagParams, handle, flag(true, mine)));
          held();
          await wait;
          value(await transactions.execute(update, flagParams, handle, flag(true, theirs)));
          return success(record(COMMIT, [["value", mine]]));
        }, leaves);
      const [a, b] = await Promise.all([attempt(first, 101n, 102n, aLocked, bGate), attempt(second, 102n, 101n, bLocked, aGate)]);
      const codes = [a, b].map(outcome => outcome.kind === "domain" ? domainOutcome(outcome, "sql::query_failed") : outcome);
      expect(codes.filter(code => typeof code === "object" && (code as Record<string, unknown>)["code"] === "ER_LOCK_DEADLOCK")).toHaveLength(1);
      expect(codes.filter(code => typeof code === "object" && (code as Record<string, unknown>)["code"] !== "ER_LOCK_DEADLOCK")).toHaveLength(1);
      // The survivor committed and neither pool is poisoned.
      expect(value(await pools.execute(update, flagParams, first, flag(false, 101n)))).toBe(1n);
      value(await pools.close(first, 1000n));
      value(await pools.close(second, 1000n));
    });
  });
  live("lock waits time out with ER_LOCK_WAIT_TIMEOUT", async () => {
    await owned(async () => {
      const first = value(await pools.mysqlOpen("CAN_TEST_MYSQL", 1n));
      const second = value(await pools.mysqlOpen("CAN_TEST_MYSQL", 1n));
      value(await pools.execute(descriptors.declareDescriptor("", "drop_cover"), emptyParams, first, record("app::empty", [])));
      value(await pools.execute(descriptors.declareDescriptor("", "setup_cover"), emptyParams, first, record("app::empty", [])));
      value(await pools.execute(descriptors.declareDescriptor("", "insert_cover"), insertParams, first, mk(101n, false)));
      const update = descriptors.declareDescriptor("", "update_flag");
      const tx = transactions.withTransaction(first, async (handle: unknown) => {
        value(await transactions.execute(update, flagParams, handle, record("app::flag_update", [["flag", true], ["id", 101n]])));
        await Bun.sleep(2500);
        return success(record(COMMIT, [["value", 1n]]));
      }, leaves);
      await Bun.sleep(300);
      value(await pools.execute(descriptors.declareDescriptor("", "set_lock_timeout"), emptyParams, second, record("app::empty", [])));
      expect(domainOutcome(await pools.execute(update, flagParams, second, record("app::flag_update", [["flag", true], ["id", 101n]])), "sql::query_failed")).toEqual({ operation: "execute", code: "ER_LOCK_WAIT_TIMEOUT" });
      expect(value(await tx)).toBe(1n);
      value(await pools.close(first, 1000n));
      value(await pools.close(second, 1000n));
    });
  });
  live("idle expiry self-heals on next use", async () => {
    await owned(async () => {
      const token = value(await pools.mysqlOpen("CAN_TEST_MYSQL", 1n));
      value(await pools.execute(descriptors.declareDescriptor("", "set_wait_timeout"), emptyParams, token, record("app::empty", [])));
      await Bun.sleep(2500);
      const tz = value(await pools.queryOne(descriptors.declareDescriptor("", "session_tz"), tzParams, token, record("app::empty", [])));
      expect(dataProperty(tz, "tz")).toBe("+00:00");
      value(await pools.close(token, 1000n));
    });
  });
  dockerized("restart kills in-flight work and the pool recovers", async () => {
    await owned(async () => {
      const token = value(await pools.mysqlOpen("CAN_TEST_MYSQL", 1n));
      const sleep = descriptors.declareDescriptor("", "sleep_query");
      try {
        const flight = pools.queryOne(sleep, sleepParams, token, record("app::sleep_parameters", [["secs", 10]]));
        await Bun.sleep(500);
        await $`docker kill ${MYSQL_CONTAINER}`.quiet();
        // SIGKILL severs the socket mid-statement, so the in-flight SLEEP(10)
        // cannot complete: a graceful `docker stop` instead lets mysqld
        // interrupt SLEEP cleanly (it returns 1), which would not exercise
        // the disconnect path.
        expect(domainOutcome(await flight, "sql::connection_failed")).toEqual({ phase: "query" });
      } finally {
        await $`docker start ${MYSQL_CONTAINER}`.quiet();
      }
      for (let i = 0; i < 30; i++) {
        const probe = await pools.queryOne(descriptors.declareDescriptor("", "session_tz"), tzParams, token, record("app::empty", []));
        if (probe.kind === "ok") break;
        if (i === 29) throw new Error("server never came back");
        await Bun.sleep(1000);
      }
      const tz = value(await pools.queryOne(descriptors.declareDescriptor("", "session_tz"), tzParams, token, record("app::empty", [])));
      expect(dataProperty(tz, "tz")).toBe("+00:00");
      value(await pools.close(token, 1000n));
    });
  });
});
