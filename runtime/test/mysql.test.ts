import { describe, expect, test } from "bun:test";
import { createHash } from "node:crypto";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, domainFailureDiagnostics, type FailureShape } from "../domain.ts";
import { success, value, type Completion } from "../completion.ts";
import { standardFailureDiagnostics, type StandardFailure } from "../failure.ts";
import { assertionContext } from "../assert/context.ts";
import { record, dataProperty, recordIdentity } from "../data.ts";
import { ownBytes, isBytes, copyBytes } from "../bytes.ts";

const bytesOrigin = { source: "test:mysql", start: 0, end: 0, invocation: [] };
import { runOwnedRoot } from "../owner.ts";
import { createSQLDescriptors, type SQLDescriptorEntry } from "../platform/sql/descriptor.ts";
import { createSQLPools, isSQLPoolValue, poolDialect } from "../platform/sql/pool.ts";
import type { SQLPlan } from "../platform/sql/values.ts";

async function owned(body: () => Promise<void>): Promise<void> {
  const result = await runOwnedRoot(async () => {
    await body();
    return success(undefined);
  });
  expect(result.cleanupFailed).toBe(false);
  expect(result.completion.kind).toBe("ok");
}
const identity = (kind: string, declaration: string) =>
  createHash("sha256")
    .update("can-concrete-type-v1\0" + JSON.stringify([kind, declaration]))
    .digest("hex");
const textShape: FailureShape = {
  identity: identity("primitive", "str"),
  kind: "primitive",
  declaration: "str",
  arguments: [],
  fields: [],
  leaves: [],
  inputs: [],
  errors: [],
};
const intShape: FailureShape = {
  identity: identity("primitive", "int"),
  kind: "primitive",
  declaration: "int",
  arguments: [],
  fields: [],
  leaves: [],
  inputs: [],
  errors: [],
};
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
  .filter((e) => fieldTypes[e.name] !== undefined)
  .map((e) => ({ identity: e.identity, name: e.name, id: e.id, parameters: 0 }));
const errorShapes: FailureShape[] = declarations.map((e) => ({
  identity: identity("error", e.identity),
  kind: "error",
  declaration: e.identity,
  arguments: [],
  fields: Object.entries(fieldTypes[e.name]!).map(([name, type]) => ({ name, type })),
  leaves: [],
  inputs: [],
  errors: [],
}));
const domain = createDomainRuntime({ declarations, shapes: [textShape, intShape, ...errorShapes] });
const id = (declaration: string) => identity("error", declaration);

// Live tests run only with a provisioned service URL; without it they
// skip visibly instead of passing vacuously. The URL never appears in
// assertions, messages, or snapshots: only the variable name travels.
const MYSQL_URL = process.env["CAN_TEST_MYSQL_URL"];
const live = test.skipIf(MYSQL_URL === undefined);
const lookup = (name: string): string | undefined => {
  if (MYSQL_URL === undefined) return undefined;
  if (name === "CAN_TEST_MYSQL") return MYSQL_URL;
  if (name === "CAN_TEST_MYSQL_REFUSED") return MYSQL_URL.replace(":3307/", ":3399/");
  if (name === "CAN_TEST_MYSQL_BADPASSWORD") return MYSQL_URL.replace(":test-pw@", ":wrong-pw@");
  if (name === "CAN_TEST_MYSQL_NODB") return MYSQL_URL.replace("/can_b1_03", "/absent_db");
  return undefined;
};

// Hand-built mysql descriptors (dialect mysql, grammar version 80011).
// DDL and session statements run through execute descriptors: the
// factory shapes segments, never the SQL text, so tests bootstrap
// their own schema. Tables drop before create so reruns are clean.
const D = "mysql" as const;
const V = 80011;
const drop = (table: string): SQLDescriptorEntry => ({
  dialect: D,
  cardinality: "execute",
  kind: "drop_table_statement",
  segments: [{ text: `DROP TABLE IF EXISTS ${table}` }],
  params: [],
  paramType: "p",
  rowType: "r",
  limit: 0,
  total: 0,
  version: V,
});
const table: Record<string, Record<string, SQLDescriptorEntry>> = {
  "": {
    drop_cover: drop("cover"),
    drop_uq: drop("uq"),
    drop_my: drop("mycover"),
    drop_misc: drop("misc"),
    drop_ts: drop("tscover"),
    drop_seq: drop("seq"),
    setup_cover: {
      dialect: D,
      cardinality: "execute",
      kind: "create_table_statement",
      segments: [
        {
          text: "CREATE TABLE cover (id BIGINT PRIMARY KEY, flag TINYINT(1), ratio DOUBLE, name VARCHAR(255), payload BLOB, note TEXT NULL)",
        },
      ],
      params: [],
      paramType: "p",
      rowType: "r",
      limit: 0,
      total: 0,
      version: V,
    },
    insert_cover: {
      dialect: D,
      cardinality: "execute",
      kind: "insert_statement",
      segments: [
        { text: "INSERT INTO cover VALUES (" },
        { param: 1 },
        { text: ", " },
        { param: 2 },
        { text: ", " },
        { param: 3 },
        { text: ", " },
        { param: 4 },
        { text: ", " },
        { param: 5 },
        { text: ", " },
        { param: 6 },
        { text: ")" },
      ],
      params: ["id", "flag", "ratio", "name", "payload", "note"],
      paramType: "p",
      rowType: "r",
      limit: 0,
      total: 6,
      version: V,
    },
    cover_by_id: {
      dialect: D,
      cardinality: "one",
      kind: "select_statement",
      segments: [
        { text: "SELECT id, flag, ratio, name, payload, note FROM cover WHERE id = " },
        { param: 1 },
        { text: " LIMIT " },
        { param: 2 },
      ],
      params: ["id"],
      paramType: "p",
      rowType: "r",
      limit: 2,
      total: 2,
      version: V,
    },
    cover_all: {
      dialect: D,
      cardinality: "many",
      kind: "select_statement",
      segments: [
        { text: "SELECT id, flag, ratio, name, payload, note FROM cover LIMIT " },
        { param: 1 },
      ],
      params: [],
      paramType: "p",
      rowType: "r",
      limit: 1,
      total: 1,
      version: V,
    },
    update_flag: {
      dialect: D,
      cardinality: "execute",
      kind: "update_statement",
      segments: [
        { text: "UPDATE cover SET flag = " },
        { param: 1 },
        { text: " WHERE id = " },
        { param: 2 },
      ],
      params: ["flag", "id"],
      paramType: "p",
      rowType: "r",
      limit: 0,
      total: 2,
      version: V,
    },
    delete_cover: {
      dialect: D,
      cardinality: "execute",
      kind: "delete_statement",
      segments: [{ text: "DELETE FROM cover WHERE id = " }, { param: 1 }],
      params: ["id"],
      paramType: "p",
      rowType: "r",
      limit: 0,
      total: 1,
      version: V,
    },
    setup_uq: {
      dialect: D,
      cardinality: "execute",
      kind: "create_table_statement",
      segments: [{ text: "CREATE TABLE uq (id INT PRIMARY KEY, v VARCHAR(32) UNIQUE)" }],
      params: [],
      paramType: "p",
      rowType: "r",
      limit: 0,
      total: 0,
      version: V,
    },
    insert_uq: {
      dialect: D,
      cardinality: "execute",
      kind: "insert_statement",
      segments: [
        { text: "INSERT INTO uq VALUES (" },
        { param: 1 },
        { text: ", " },
        { param: 2 },
        { text: ")" },
      ],
      params: ["id", "v"],
      paramType: "p",
      rowType: "r",
      limit: 0,
      total: 2,
      version: V,
    },
    setup_my: {
      dialect: D,
      cardinality: "execute",
      kind: "create_table_statement",
      segments: [
        {
          text: "CREATE TABLE mycover (id BIGINT PRIMARY KEY, amount DECIMAL(30,10), stamp DATETIME(3), ubig BIGINT UNSIGNED, bit1 BIT(1), bit4 BIT(4))",
        },
      ],
      params: [],
      paramType: "p",
      rowType: "r",
      limit: 0,
      total: 0,
      version: V,
    },
    insert_my: {
      dialect: D,
      cardinality: "execute",
      kind: "insert_statement",
      segments: [
        { text: "INSERT INTO mycover VALUES (" },
        { param: 1 },
        { text: ", " },
        { param: 2 },
        { text: ", " },
        { param: 3 },
        { text: ", " },
        { param: 4 },
        { text: ", " },
        { param: 5 },
        { text: ", " },
        { param: 6 },
        { text: ")" },
      ],
      params: ["id", "amount", "stamp", "ubig", "bit1", "bit4"],
      paramType: "p",
      rowType: "r",
      limit: 0,
      total: 6,
      version: V,
    },
    my_by_id: {
      dialect: D,
      cardinality: "one",
      kind: "select_statement",
      segments: [
        { text: "SELECT id, amount, stamp, ubig, bit1, bit4 FROM mycover WHERE id = " },
        { param: 1 },
        { text: " LIMIT " },
        { param: 2 },
      ],
      params: ["id"],
      paramType: "p",
      rowType: "r",
      limit: 2,
      total: 2,
      version: V,
    },
    setup_misc: {
      dialect: D,
      cardinality: "execute",
      kind: "create_table_statement",
      segments: [
        {
          text: "CREATE TABLE misc (id INT PRIMARY KEY, j JSON, ti TIME, y YEAR, e ENUM('a','b'), s SET('x','y'))",
        },
      ],
      params: [],
      paramType: "p",
      rowType: "r",
      limit: 0,
      total: 0,
      version: V,
    },
    insert_misc: {
      dialect: D,
      cardinality: "execute",
      kind: "insert_statement",
      segments: [
        { text: "INSERT INTO misc VALUES (" },
        { param: 1 },
        { text: ", " },
        { param: 2 },
        { text: ", " },
        { param: 3 },
        { text: ", " },
        { param: 4 },
        { text: ", " },
        { param: 5 },
        { text: ", " },
        { param: 6 },
        { text: ")" },
      ],
      params: ["id", "j", "ti", "y", "e", "s"],
      paramType: "p",
      rowType: "r",
      limit: 0,
      total: 6,
      version: V,
    },
    misc_by_id: {
      dialect: D,
      cardinality: "one",
      kind: "select_statement",
      segments: [
        { text: "SELECT id, j, ti, y, e, s FROM misc WHERE id = " },
        { param: 1 },
        { text: " LIMIT " },
        { param: 2 },
      ],
      params: ["id"],
      paramType: "p",
      rowType: "r",
      limit: 2,
      total: 2,
      version: V,
    },
    setup_ts: {
      dialect: D,
      cardinality: "execute",
      kind: "create_table_statement",
      segments: [{ text: "CREATE TABLE tscover (id INT PRIMARY KEY, ts TIMESTAMP(3))" }],
      params: [],
      paramType: "p",
      rowType: "r",
      limit: 0,
      total: 0,
      version: V,
    },
    insert_ts: {
      dialect: D,
      cardinality: "execute",
      kind: "insert_statement",
      segments: [
        { text: "INSERT INTO tscover VALUES (" },
        { param: 1 },
        { text: ", " },
        { param: 2 },
        { text: ")" },
      ],
      params: ["id", "ts"],
      paramType: "p",
      rowType: "r",
      limit: 0,
      total: 2,
      version: V,
    },
    misc_text_by_id: {
      dialect: D,
      cardinality: "one",
      kind: "select_statement",
      segments: [
        { text: "SELECT ti, y, e, s FROM misc WHERE id = " },
        { param: 1 },
        { text: " LIMIT " },
        { param: 2 },
      ],
      params: ["id"],
      paramType: "p",
      rowType: "r",
      limit: 2,
      total: 2,
      version: V,
    },
    ts_by_id: {
      dialect: D,
      cardinality: "one",
      kind: "select_statement",
      segments: [
        { text: "SELECT id, ts FROM tscover WHERE id = " },
        { param: 1 },
        { text: " LIMIT " },
        { param: 2 },
      ],
      params: ["id"],
      paramType: "p",
      rowType: "r",
      limit: 2,
      total: 2,
      version: V,
    },
    setup_seq: {
      dialect: D,
      cardinality: "execute",
      kind: "create_table_statement",
      segments: [{ text: "CREATE TABLE seq (id BIGINT AUTO_INCREMENT PRIMARY KEY)" }],
      params: [],
      paramType: "p",
      rowType: "r",
      limit: 0,
      total: 0,
      version: V,
    },
    insert_seq: {
      dialect: D,
      cardinality: "execute",
      kind: "insert_statement",
      segments: [{ text: "INSERT INTO seq VALUES ()" }],
      params: [],
      paramType: "p",
      rowType: "r",
      limit: 0,
      total: 0,
      version: V,
    },
    last_id: {
      dialect: D,
      cardinality: "one",
      kind: "select_statement",
      segments: [{ text: "SELECT LAST_INSERT_ID() AS id LIMIT " }, { param: 1 }],
      params: [],
      paramType: "p",
      rowType: "r",
      limit: 1,
      total: 1,
      version: V,
    },
    session_tz: {
      dialect: D,
      cardinality: "one",
      kind: "select_statement",
      segments: [{ text: "SELECT @@session.time_zone AS tz LIMIT " }, { param: 1 }],
      params: [],
      paramType: "p",
      rowType: "r",
      limit: 1,
      total: 1,
      version: V,
    },
    broken_syntax: {
      dialect: D,
      cardinality: "execute",
      kind: "insert_statement",
      segments: [{ text: "INSERT INTO cover VALUES (" }],
      params: [],
      paramType: "p",
      rowType: "r",
      limit: 0,
      total: 0,
      version: V,
    },
    drop_absent: {
      dialect: D,
      cardinality: "execute",
      kind: "delete_statement",
      segments: [{ text: "DELETE FROM absent" }],
      params: [],
      paramType: "p",
      rowType: "r",
      limit: 0,
      total: 0,
      version: V,
    },
    bad_column: {
      dialect: D,
      cardinality: "one",
      kind: "select_statement",
      segments: [
        { text: "SELECT nope FROM cover WHERE id = " },
        { param: 1 },
        { text: " LIMIT " },
        { param: 2 },
      ],
      params: ["id"],
      paramType: "p",
      rowType: "r",
      limit: 2,
      total: 2,
      version: V,
    },
    pg_by_id: {
      dialect: "postgresql",
      cardinality: "one",
      kind: "SelectStmt",
      segments: [
        { text: "SELECT id FROM cover WHERE id = " },
        { param: 1 },
        { text: " LIMIT " },
        { param: 2 },
      ],
      params: ["id"],
      paramType: "p",
      rowType: "r",
      limit: 2,
      total: 2,
      version: 170007,
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
      {
        name: "note",
        kind: "option",
        inner: "str",
        some: "app::option_some",
        none: "app::option_none",
      },
    ],
  },
};
const insertParams: SQLPlan = {
  params: {
    root: "app::cover_insert",
    fields: [
      { name: "id", kind: "int" },
      { name: "flag", kind: "bool" },
      { name: "ratio", kind: "float" },
      { name: "name", kind: "str" },
      { name: "payload", kind: "bytes" },
      {
        name: "note",
        kind: "option",
        inner: "str",
        some: "app::option_some",
        none: "app::option_none",
      },
    ],
  },
};
const coverRows: SQLPlan = { params: { root: "app::empty", fields: [] }, rows: coverParams.rows! };
const flagParams: SQLPlan = {
  params: {
    root: "app::flag_update",
    fields: [
      { name: "flag", kind: "bool" },
      { name: "id", kind: "int" },
    ],
  },
};
const deleteParams: SQLPlan = {
  params: { root: "app::cover_delete", fields: [{ name: "id", kind: "int" }] },
};
// ubig inserts as digit text (the server validates the unsigned range);
// reads decode as int, so values outside int64 reject as int_range.
const myInsertParams: SQLPlan = {
  params: {
    root: "app::my_insert",
    fields: [
      { name: "id", kind: "int" },
      { name: "amount", kind: "str" },
      { name: "stamp", kind: "str" },
      { name: "ubig", kind: "str" },
      { name: "bit1", kind: "bool" },
      { name: "bit4", kind: "bytes" },
    ],
  },
};
const myParams: SQLPlan = {
  params: { root: "app::my_parameters", fields: [{ name: "id", kind: "int" }] },
  rows: {
    root: "app::my_row",
    fields: [
      { name: "id", kind: "int" },
      { name: "amount", kind: "str" },
      { name: "stamp", kind: "str" },
      { name: "ubig", kind: "int" },
      { name: "bit1", kind: "bool" },
      { name: "bit4", kind: "bytes" },
    ],
  },
};
const miscInsertParams: SQLPlan = {
  params: {
    root: "app::misc_insert",
    fields: [
      { name: "id", kind: "int" },
      { name: "j", kind: "str" },
      { name: "ti", kind: "str" },
      { name: "y", kind: "int" },
      { name: "e", kind: "str" },
      { name: "s", kind: "str" },
    ],
  },
};
const miscParams: SQLPlan = {
  params: { root: "app::misc_parameters", fields: [{ name: "id", kind: "int" }] },
  rows: {
    root: "app::misc_row",
    fields: [
      { name: "id", kind: "int" },
      { name: "j", kind: "str" },
      { name: "ti", kind: "str" },
      { name: "y", kind: "int" },
      { name: "e", kind: "str" },
      { name: "s", kind: "str" },
    ],
  },
};
const miscTextParams: SQLPlan = {
  params: { root: "app::misc_parameters", fields: [{ name: "id", kind: "int" }] },
  rows: {
    root: "app::misc_text_row",
    fields: [
      { name: "ti", kind: "str" },
      { name: "y", kind: "int" },
      { name: "e", kind: "str" },
      { name: "s", kind: "str" },
    ],
  },
};
const tsInsertParams: SQLPlan = {
  params: {
    root: "app::ts_insert",
    fields: [
      { name: "id", kind: "int" },
      { name: "ts", kind: "str" },
    ],
  },
};
const tsParams: SQLPlan = {
  params: { root: "app::ts_parameters", fields: [{ name: "id", kind: "int" }] },
  rows: {
    root: "app::ts_row",
    fields: [
      { name: "id", kind: "int" },
      { name: "ts", kind: "str" },
    ],
  },
};
const uqInsertParams: SQLPlan = {
  params: {
    root: "app::uq_insert",
    fields: [
      { name: "id", kind: "int" },
      { name: "v", kind: "str" },
    ],
  },
};
const idParams: SQLPlan = {
  params: { root: "app::id_parameters", fields: [{ name: "id", kind: "int" }] },
  rows: { root: "app::id_row", fields: [{ name: "id", kind: "int" }] },
};
const lastIdParams: SQLPlan = {
  params: { root: "app::empty", fields: [] },
  rows: { root: "app::id_row", fields: [{ name: "id", kind: "int" }] },
};
const tzParams: SQLPlan = {
  params: { root: "app::empty", fields: [] },
  rows: { root: "app::tz_row", fields: [{ name: "tz", kind: "str" }] },
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

describe("mysql pools", () => {
  live("open validates credentials, config, and the UTC pin", async () => {
    await owned(async () => {
      const token = value(await pools.mysqlOpen("CAN_TEST_MYSQL", 2n));
      expect(isSQLPoolValue("pool", token)).toBe(true);
      expect(poolDialect(token)).toBe("mysql");
      const tz = value(
        await pools.queryOne(
          descriptors.declareDescriptor("", "session_tz"),
          tzParams,
          token,
          record("app::empty", []),
        ),
      );
      expect(dataProperty(tz, "tz")).toBe("+00:00");
      value(await pools.close(token, 1000n));
      expect(
        domainOutcome(await pools.mysqlOpen("bad-name", 2n), "http::credentials_missing"),
      ).toEqual({ variable: "bad-name" });
      expect(
        domainOutcome(await pools.mysqlOpen("CAN_TEST_ABSENT", 2n), "http::credentials_missing"),
      ).toEqual({ variable: "CAN_TEST_ABSENT" });
      expect(
        domainOutcome(await pools.mysqlOpen("CAN_TEST_MYSQL", 0n), "sql::connection_failed"),
      ).toEqual({ phase: "config" });
      expect(
        domainOutcome(
          await pools.mysqlOpen("CAN_TEST_MYSQL_REFUSED", 2n),
          "sql::connection_failed",
        ),
      ).toEqual({ phase: "connect" });
      expect(
        domainOutcome(
          await pools.mysqlOpen("CAN_TEST_MYSQL_BADPASSWORD", 2n),
          "sql::connection_failed",
        ),
      ).toEqual({ phase: "connect" });
      expect(
        domainOutcome(await pools.mysqlOpen("CAN_TEST_MYSQL_NODB", 2n), "sql::connection_failed"),
      ).toEqual({ phase: "connect" });
    });
  });
  live("pools round-trip the shared cover exactly", async () => {
    await owned(async () => {
      const token = value(await pools.mysqlOpen("CAN_TEST_MYSQL", 2n));
      const drop = descriptors.declareDescriptor("", "drop_cover");
      const setup = descriptors.declareDescriptor("", "setup_cover");
      const insert = descriptors.declareDescriptor("", "insert_cover");
      const byId = descriptors.declareDescriptor("", "cover_by_id");
      value(await pools.execute(drop, emptyParams, token, record("app::empty", [])));
      value(await pools.execute(setup, emptyParams, token, record("app::empty", [])));
      const rows: Array<[bigint, boolean, number, string, Uint8Array, unknown]> = [
        [
          9007199254740993n,
          true,
          1.5,
          "n1",
          new Uint8Array([1]),
          record("app::option_some", [["value", "s1"]]),
        ],
        [MIN_INT64, false, 0, "n2-ünïcødé-🎉", new Uint8Array(0), record("app::option_none", [])],
        [
          MAX_INT64,
          true,
          -2.5,
          "n3",
          new Uint8Array([9, 9, 9]),
          record("app::option_some", [["value", "s3"]]),
        ],
      ];
      for (const [idn, flag, ratio, name, bytes, note] of rows) {
        expect(
          value(
            await pools.execute(
              insert,
              insertParams,
              token,
              record("app::cover_insert", [
                ["id", idn],
                ["flag", flag],
                ["ratio", ratio],
                ["name", name],
                ["payload", ownBytes(bytes)],
                ["note", note],
              ]),
            ),
          ),
        ).toBe(1n);
      }
      for (const [idn, flag, ratio, name, bytes, note] of rows) {
        const row = value(
          await pools.queryOne(
            byId,
            coverParams,
            token,
            record("app::cover_parameters", [["id", idn]]),
          ),
        );
        expect(recordIdentity(row)).toBe("app::cover_row");
        expect(dataProperty(row, "id")).toBe(idn);
        expect(dataProperty(row, "flag")).toBe(flag);
        expect(dataProperty(row, "ratio")).toBe(ratio);
        expect(dataProperty(row, "name")).toBe(name);
        expect(isBytes(dataProperty(row, "payload"))).toBe(true);
        expect([...copyBytes(dataProperty(row, "payload"), bytesOrigin)]).toEqual([...bytes]);
        expect(recordIdentity(dataProperty(row, "note"))).toBe(recordIdentity(note));
        if (recordIdentity(note) === "app::option_some")
          expect(dataProperty(dataProperty(row, "note"), "value")).toBe(
            dataProperty(note, "value"),
          );
      }
      value(await pools.close(token, 1000n));
    });
  });
  live("execute reports exact affected counts including zero", async () => {
    await owned(async () => {
      const token = value(await pools.mysqlOpen("CAN_TEST_MYSQL", 2n));
      value(
        await pools.execute(
          descriptors.declareDescriptor("", "drop_cover"),
          emptyParams,
          token,
          record("app::empty", []),
        ),
      );
      value(
        await pools.execute(
          descriptors.declareDescriptor("", "setup_cover"),
          emptyParams,
          token,
          record("app::empty", []),
        ),
      );
      const insert = descriptors.declareDescriptor("", "insert_cover");
      const update = descriptors.declareDescriptor("", "update_flag");
      const remove = descriptors.declareDescriptor("", "delete_cover");
      const mk = (idn: bigint) =>
        record("app::cover_insert", [
          ["id", idn],
          ["flag", false],
          ["ratio", 0],
          ["name", "n"],
          ["payload", ownBytes(new Uint8Array(0))],
          ["note", record("app::option_none", [])],
        ]);
      expect(value(await pools.execute(insert, insertParams, token, mk(1n)))).toBe(1n);
      expect(value(await pools.execute(insert, insertParams, token, mk(2n)))).toBe(1n);
      expect(
        value(
          await pools.execute(
            update,
            flagParams,
            token,
            record("app::flag_update", [
              ["flag", true],
              ["id", 1n],
            ]),
          ),
        ),
      ).toBe(1n);
      expect(
        value(
          await pools.execute(
            remove,
            deleteParams,
            token,
            record("app::cover_delete", [["id", 2n]]),
          ),
        ),
      ).toBe(1n);
      expect(
        value(
          await pools.execute(
            remove,
            deleteParams,
            token,
            record("app::cover_delete", [["id", 2n]]),
          ),
        ),
      ).toBe(0n);
      value(await pools.close(token, 1000n));
    });
  });
  live("mysql-native types map exactly", async () => {
    await owned(async () => {
      const token = value(await pools.mysqlOpen("CAN_TEST_MYSQL", 2n));
      for (const name of ["drop_my", "drop_misc", "drop_ts"])
        value(
          await pools.execute(
            descriptors.declareDescriptor("", name),
            emptyParams,
            token,
            record("app::empty", []),
          ),
        );
      for (const name of ["setup_my", "setup_misc", "setup_ts"])
        value(
          await pools.execute(
            descriptors.declareDescriptor("", name),
            emptyParams,
            token,
            record("app::empty", []),
          ),
        );
      expect(
        value(
          await pools.execute(
            descriptors.declareDescriptor("", "insert_my"),
            myInsertParams,
            token,
            record("app::my_insert", [
              ["id", 1n],
              ["amount", "12345678901234567890.1234567890"],
              ["stamp", "2026-03-04 05:06:07.890"],
              ["ubig", "5"],
              ["bit1", true],
              ["bit4", ownBytes(new Uint8Array([10]))],
            ]),
          ),
        ),
      ).toBe(1n);
      const row = value(
        await pools.queryOne(
          descriptors.declareDescriptor("", "my_by_id"),
          myParams,
          token,
          record("app::my_parameters", [["id", 1n]]),
        ),
      );
      expect(dataProperty(row, "amount")).toBe("12345678901234567890.1234567890");
      expect(dataProperty(row, "stamp")).toBe("2026-03-04 05:06:07.890");
      expect(dataProperty(row, "ubig")).toBe(5n);
      expect(dataProperty(row, "bit1")).toBe(true);
      expect([...copyBytes(dataProperty(row, "bit4"), bytesOrigin)]).toEqual([10]);
      expect(
        value(
          await pools.execute(
            descriptors.declareDescriptor("", "insert_misc"),
            miscInsertParams,
            token,
            record("app::misc_insert", [
              ["id", 1n],
              ["j", '{"k":[1,2]}'],
              ["ti", "11:22:33"],
              ["y", 2026n],
              ["e", "b"],
              ["s", "x,y"],
            ]),
          ),
        ),
      ).toBe(1n);
      // The JSON column parses server-side, so the driver hands back an
      // object the str field rejects; the textual columns map exactly.
      // (misc_by_id is exercised for j in the rejection test below.)
      expect(
        value(
          await pools.execute(
            descriptors.declareDescriptor("", "insert_ts"),
            tsInsertParams,
            token,
            record("app::ts_insert", [
              ["id", 1n],
              ["ts", "2026-01-15 12:00:00.123"],
            ]),
          ),
        ),
      ).toBe(1n);
      const ts = value(
        await pools.queryOne(
          descriptors.declareDescriptor("", "ts_by_id"),
          tsParams,
          token,
          record("app::ts_parameters", [["id", 1n]]),
        ),
      );
      expect(dataProperty(ts, "ts")).toBe("2026-01-15 12:00:00.123");
      const ti = value(
        await pools.queryOne(
          descriptors.declareDescriptor("", "misc_text_by_id"),
          miscTextParams,
          token,
          record("app::misc_parameters", [["id", 1n]]),
        ),
      );
      expect(dataProperty(ti, "ti")).toBe("11:22:33");
      expect(dataProperty(ti, "y")).toBe(2026n);
      expect(dataProperty(ti, "e")).toBe("b");
      expect(dataProperty(ti, "s")).toBe("x,y");
      value(await pools.close(token, 1000n));
    });
  });
  live("unsigned overflow and JSON objects reject without coercion", async () => {
    await owned(async () => {
      const token = value(await pools.mysqlOpen("CAN_TEST_MYSQL", 2n));
      for (const name of ["drop_my", "drop_misc"])
        value(
          await pools.execute(
            descriptors.declareDescriptor("", name),
            emptyParams,
            token,
            record("app::empty", []),
          ),
        );
      for (const name of ["setup_my", "setup_misc"])
        value(
          await pools.execute(
            descriptors.declareDescriptor("", name),
            emptyParams,
            token,
            record("app::empty", []),
          ),
        );
      expect(
        value(
          await pools.execute(
            descriptors.declareDescriptor("", "insert_my"),
            myInsertParams,
            token,
            record("app::my_insert", [
              ["id", 2n],
              ["amount", "1.5"],
              ["stamp", "2026-01-01 00:00:00"],
              ["ubig", "18446744073709551615"],
              ["bit1", false],
              ["bit4", ownBytes(new Uint8Array([1]))],
            ]),
          ),
        ),
      ).toBe(1n);
      expect(
        domainOutcome(
          await pools.queryOne(
            descriptors.declareDescriptor("", "my_by_id"),
            myParams,
            token,
            record("app::my_parameters", [["id", 2n]]),
          ),
          "sql::schema_mismatch",
        ),
      ).toEqual({ path: "/ubig", reason: "int_range" });
      expect(
        value(
          await pools.execute(
            descriptors.declareDescriptor("", "insert_misc"),
            miscInsertParams,
            token,
            record("app::misc_insert", [
              ["id", 2n],
              ["j", '{"k":1}'],
              ["ti", "00:00:01"],
              ["y", 2000n],
              ["e", "a"],
              ["s", "x"],
            ]),
          ),
        ),
      ).toBe(1n);
      expect(
        domainOutcome(
          await pools.queryOne(
            descriptors.declareDescriptor("", "misc_by_id"),
            miscParams,
            token,
            record("app::misc_parameters", [["id", 2n]]),
          ),
          "sql::schema_mismatch",
        ),
      ).toEqual({ path: "/j", reason: "type" });
      // Malformed decimal text is server-rejected under strict mode;
      // the adapter never parses it, so nothing coerces.
      expect(
        domainOutcome(
          await pools.execute(
            descriptors.declareDescriptor("", "insert_my"),
            myInsertParams,
            token,
            record("app::my_insert", [
              ["id", 3n],
              ["amount", "banana"],
              ["stamp", "2026-01-01 00:00:00"],
              ["ubig", "1"],
              ["bit1", false],
              ["bit4", ownBytes(new Uint8Array([1]))],
            ]),
          ),
          "sql::query_failed",
        ),
      ).toEqual({ operation: "execute", code: "mysql_errno_1366" });
      value(await pools.close(token, 1000n));
    });
  });
  live("dialect mismatch fails before any native call", async () => {
    await owned(async () => {
      const token = value(await pools.mysqlOpen("CAN_TEST_MYSQL", 2n));
      const pg = descriptors.declareDescriptor("", "pg_by_id");
      expect(
        domainOutcome(
          await pools.queryOne(pg, idParams, token, record("app::id_parameters", [["id", 1n]])),
          "sql::query_failed",
        ),
      ).toEqual({ operation: "query_one", code: "dialect_mismatch" });
      value(await pools.close(token, 1000n));
    });
  });
  live("failures classify with verified codes", async () => {
    await owned(async () => {
      const token = value(await pools.mysqlOpen("CAN_TEST_MYSQL", 2n));
      value(
        await pools.execute(
          descriptors.declareDescriptor("", "drop_uq"),
          emptyParams,
          token,
          record("app::empty", []),
        ),
      );
      value(
        await pools.execute(
          descriptors.declareDescriptor("", "setup_uq"),
          emptyParams,
          token,
          record("app::empty", []),
        ),
      );
      const insert = descriptors.declareDescriptor("", "insert_uq");
      expect(
        value(
          await pools.execute(
            insert,
            uqInsertParams,
            token,
            record("app::uq_insert", [
              ["id", 1n],
              ["v", "a"],
            ]),
          ),
        ),
      ).toBe(1n);
      expect(
        domainOutcome(
          await pools.execute(
            insert,
            uqInsertParams,
            token,
            record("app::uq_insert", [
              ["id", 2n],
              ["v", "a"],
            ]),
          ),
          "sql::constraint_failed",
        ),
      ).toEqual({ constraint: "ER_DUP_ENTRY" });
      expect(
        domainOutcome(
          await pools.execute(
            descriptors.declareDescriptor("", "broken_syntax"),
            emptyParams,
            token,
            record("app::empty", []),
          ),
          "sql::query_failed",
        ),
      ).toEqual({ operation: "execute", code: "ER_PARSE_ERROR" });
      expect(
        domainOutcome(
          await pools.execute(
            descriptors.declareDescriptor("", "drop_absent"),
            emptyParams,
            token,
            record("app::empty", []),
          ),
          "sql::query_failed",
        ),
      ).toEqual({ operation: "execute", code: "ER_NO_SUCH_TABLE" });
      value(
        await pools.execute(
          descriptors.declareDescriptor("", "drop_cover"),
          emptyParams,
          token,
          record("app::empty", []),
        ),
      );
      value(
        await pools.execute(
          descriptors.declareDescriptor("", "setup_cover"),
          emptyParams,
          token,
          record("app::empty", []),
        ),
      );
      expect(
        domainOutcome(
          await pools.queryOne(
            descriptors.declareDescriptor("", "bad_column"),
            coverParams,
            token,
            record("app::cover_parameters", [["id", 1n]]),
          ),
          "sql::query_failed",
        ),
      ).toEqual({ operation: "query_one", code: "mysql_errno_1054" });
      value(await pools.close(token, 1000n));
    });
  });
  live("hostile text travels only as a bound parameter", async () => {
    await owned(async () => {
      const token = value(await pools.mysqlOpen("CAN_TEST_MYSQL", 2n));
      value(
        await pools.execute(
          descriptors.declareDescriptor("", "drop_cover"),
          emptyParams,
          token,
          record("app::empty", []),
        ),
      );
      value(
        await pools.execute(
          descriptors.declareDescriptor("", "setup_cover"),
          emptyParams,
          token,
          record("app::empty", []),
        ),
      );
      const evil = "x' OR '1'='1' -- /*";
      expect(
        value(
          await pools.execute(
            descriptors.declareDescriptor("", "insert_cover"),
            insertParams,
            token,
            record("app::cover_insert", [
              ["id", 77n],
              ["flag", true],
              ["ratio", 0],
              ["name", evil],
              ["payload", ownBytes(new Uint8Array(0))],
              ["note", record("app::option_none", [])],
            ]),
          ),
        ),
      ).toBe(1n);
      const row = value(
        await pools.queryOne(
          descriptors.declareDescriptor("", "cover_by_id"),
          coverParams,
          token,
          record("app::cover_parameters", [["id", 77n]]),
        ),
      );
      expect(dataProperty(row, "name")).toBe(evil);
      const rows = value(
        await pools.queryRows(
          descriptors.declareDescriptor("", "cover_all"),
          coverRows,
          token,
          record("app::empty", []),
          10n,
        ),
      );
      expect((rows as readonly unknown[]).length).toBe(1);
      value(await pools.close(token, 1000n));
    });
  });
  live("descriptors reuse native prepares across executions", async () => {
    await owned(async () => {
      const token = value(await pools.mysqlOpen("CAN_TEST_MYSQL", 2n));
      value(
        await pools.execute(
          descriptors.declareDescriptor("", "drop_cover"),
          emptyParams,
          token,
          record("app::empty", []),
        ),
      );
      value(
        await pools.execute(
          descriptors.declareDescriptor("", "setup_cover"),
          emptyParams,
          token,
          record("app::empty", []),
        ),
      );
      const insert = descriptors.declareDescriptor("", "insert_cover");
      const byId = descriptors.declareDescriptor("", "cover_by_id");
      for (let i = 0; i < 5; i++) {
        const idn = BigInt(100 + i);
        expect(
          value(
            await pools.execute(
              insert,
              insertParams,
              token,
              record("app::cover_insert", [
                ["id", idn],
                ["flag", i % 2 === 0],
                ["ratio", i],
                ["name", `r${i}`],
                ["payload", ownBytes(new Uint8Array([i]))],
                ["note", record("app::option_none", [])],
              ]),
            ),
          ),
        ).toBe(1n);
        const row = value(
          await pools.queryOne(
            byId,
            coverParams,
            token,
            record("app::cover_parameters", [["id", idn]]),
          ),
        );
        expect(dataProperty(row, "name")).toBe(`r${i}`);
      }
      value(await pools.close(token, 1000n));
    });
  });
  live("generated ids read back through LAST_INSERT_ID", async () => {
    await owned(async () => {
      const token = value(await pools.mysqlOpen("CAN_TEST_MYSQL", 2n));
      value(
        await pools.execute(
          descriptors.declareDescriptor("", "drop_seq"),
          emptyParams,
          token,
          record("app::empty", []),
        ),
      );
      value(
        await pools.execute(
          descriptors.declareDescriptor("", "setup_seq"),
          emptyParams,
          token,
          record("app::empty", []),
        ),
      );
      expect(
        value(
          await pools.execute(
            descriptors.declareDescriptor("", "insert_seq"),
            emptyParams,
            token,
            record("app::empty", []),
          ),
        ),
      ).toBe(1n);
      expect(
        value(
          await pools.execute(
            descriptors.declareDescriptor("", "insert_seq"),
            emptyParams,
            token,
            record("app::empty", []),
          ),
        ),
      ).toBe(1n);
      const row = value(
        await pools.queryOne(
          descriptors.declareDescriptor("", "last_id"),
          lastIdParams,
          token,
          record("app::empty", []),
        ),
      );
      expect(dataProperty(row, "id")).toBe(2n);
      value(await pools.close(token, 1000n));
    });
  });
  live("close validates, closes, and denies later work", async () => {
    await owned(async () => {
      const token = value(await pools.mysqlOpen("CAN_TEST_MYSQL", 2n));
      expect(domainOutcome(await pools.close(token, -1n), "sql::close_failed")).toEqual({
        reason: "invalid_timeout",
      });
      expect(value(await pools.close(token, 1000n))).toBe(undefined);
      try {
        await pools.queryOne(
          descriptors.declareDescriptor("", "cover_by_id"),
          coverParams,
          token,
          record("app::cover_parameters", [["id", 1n]]),
        );
        throw new Error("query admitted on a closed pool");
      } catch (cause) {
        expect(standardFailureDiagnostics(cause as StandardFailure).kind).toBe("resource_state");
      }
    });
  });
  test("fixture boundaries never open mysql pools", async () => {
    const context = fixtureContext();
    await owned(async () => {
      await expect(pools.mysqlOpen("CAN_TEST_MYSQL", 2n, context)).rejects.toThrow();
    });
  });
});
