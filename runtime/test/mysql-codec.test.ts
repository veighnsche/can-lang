import { describe, expect, test } from "bun:test";
import { createHash } from "node:crypto";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, domainFailureDiagnostics, type FailureShape } from "../domain.ts";
import type { Completion } from "../completion.ts";
import { dataProperty } from "../data.ts";
import { createValueCodec, mysqlValueProfile } from "../platform/sql/values.ts";
import { createSQLFailures } from "../platform/sql/errors.ts";
import { classifyMySQL, mysqlAffectedRows } from "../platform/sql/mysql.ts";

const origin = { source: "test:mysql-codec", start: 0, end: 0, invocation: [] };
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
  "sql::connection_failed": { phase: textShape.identity },
  "sql::query_failed": { operation: textShape.identity, code: textShape.identity },
  "sql::schema_mismatch": { path: textShape.identity, reason: textShape.identity },
  "sql::constraint_failed": { constraint: textShape.identity },
  "sql::unsupported_value": { path: textShape.identity, reason: textShape.identity },
};
const declarations = catalogue.errors
  .filter((e) => fieldTypes[e.name] !== undefined)
  .map((e) => ({ identity: e.identity, name: e.name, parameters: 0 }));
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
const contracts = {
  connectionFailed: id("can.std.sql@1::connection_failed"),
  queryFailed: id("can.std.sql@1::query_failed"),
  rowMissing: id("can.std.sql@1::row_missing"),
  rowCount: id("can.std.sql@1::row_count"),
  schemaMismatch: id("can.std.sql@1::schema_mismatch"),
  constraintFailed: id("can.std.sql@1::constraint_failed"),
  rowLimit: id("can.std.sql@1::row_limit"),
  unsupportedValue: id("can.std.sql@1::unsupported_value"),
};
const failures = createSQLFailures(domain, contracts, origin);
const codec = createValueCodec(origin, failures, mysqlValueProfile);

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
const rowsOf = (
  fields: Array<{ name: string; kind: string; inner?: string; some?: string; none?: string }>,
) => ({
  root: "app::row",
  fields: fields as Array<{
    name: string;
    kind: "bool" | "int" | "float" | "str" | "bytes" | "option";
    inner?: "bool" | "int" | "float" | "str" | "bytes";
    some?: string;
    none?: string;
  }>,
});
const decoded = (fields: Parameters<typeof rowsOf>[0], row: unknown): unknown => {
  const outcome = codec.decodeRow(rowsOf(fields), row);
  if (!outcome.ok)
    throw new Error(
      "expected row, got " + JSON.stringify(domainOutcome(outcome.failure, "sql::schema_mismatch")),
    );
  return outcome.value;
};
const mismatched = (
  fields: Parameters<typeof rowsOf>[0],
  row: unknown,
): Record<string, unknown> => {
  const outcome = codec.decodeRow(rowsOf(fields), row);
  expect(outcome.ok).toBe(false);
  if (outcome.ok) throw new Error("expected mismatch");
  return domainOutcome(outcome.failure, "sql::schema_mismatch");
};
const mysqlError = (code: string, errno?: unknown): Error & { code: string; errno?: unknown } => {
  const error = new Error("native") as Error & { code: string; errno?: unknown };
  error.name = "MySQLError";
  error.code = code;
  if (errno !== undefined) error.errno = errno;
  return error;
};

describe("mysql codec", () => {
  test("datetimes render naive UTC wall text", () => {
    expect(
      dataProperty(
        decoded([{ name: "t", kind: "str" }], { t: new Date("2026-01-15T12:00:00.000Z") }),
        "t",
      ),
    ).toBe("2026-01-15 12:00:00");
    expect(
      dataProperty(
        decoded([{ name: "t", kind: "str" }], { t: new Date("2026-03-04T05:06:07.890Z") }),
        "t",
      ),
    ).toBe("2026-03-04 05:06:07.890");
    expect(
      dataProperty(
        decoded([{ name: "t", kind: "str" }], { t: new Date("1999-12-31T23:59:59.001Z") }),
        "t",
      ),
    ).toBe("1999-12-31 23:59:59.001");
    expect(mismatched([{ name: "t", kind: "str" }], { t: new Date(NaN) })).toEqual({
      path: "/t",
      reason: "type",
    });
  });
  test("integers decode from bigints, safe numbers, and canonical strings", () => {
    const f = [{ name: "v", kind: "int" }] as Parameters<typeof rowsOf>[0];
    expect(dataProperty(decoded(f, { v: 7 }), "v")).toBe(7n);
    expect(dataProperty(decoded(f, { v: 9007199254740993n }), "v")).toBe(9007199254740993n);
    expect(dataProperty(decoded(f, { v: "9007199254740993" }), "v")).toBe(9007199254740993n);
    expect(dataProperty(decoded(f, { v: "-9223372036854775808" }), "v")).toBe(-(1n << 63n));
    expect(dataProperty(decoded(f, { v: "9223372036854775807" }), "v")).toBe((1n << 63n) - 1n);
  });
  test("out-of-range and malformed integers reject without coercion", () => {
    const f = [{ name: "v", kind: "int" }] as Parameters<typeof rowsOf>[0];
    expect(mismatched(f, { v: "18446744073709551615" })).toEqual({
      path: "/v",
      reason: "int_range",
    });
    expect(mismatched(f, { v: 18446744073709551615n })).toEqual({
      path: "/v",
      reason: "int_range",
    });
    expect(mismatched(f, { v: "12.5" })).toEqual({ path: "/v", reason: "type" });
    expect(mismatched(f, { v: "+7" })).toEqual({ path: "/v", reason: "type" });
    expect(mismatched(f, { v: "" })).toEqual({ path: "/v", reason: "type" });
    expect(mismatched(f, { v: 1.5 })).toEqual({ path: "/v", reason: "unsafe_integer" });
  });
  test("booleans accept native and exact 0/1 only", () => {
    const f = [{ name: "v", kind: "bool" }] as Parameters<typeof rowsOf>[0];
    expect(dataProperty(decoded(f, { v: true }), "v")).toBe(true);
    expect(dataProperty(decoded(f, { v: false }), "v")).toBe(false);
    expect(dataProperty(decoded(f, { v: 1 }), "v")).toBe(true);
    expect(dataProperty(decoded(f, { v: 0 }), "v")).toBe(false);
    expect(dataProperty(decoded(f, { v: 1n }), "v")).toBe(true);
    expect(mismatched(f, { v: 2 })).toEqual({ path: "/v", reason: "type" });
    expect(mismatched(f, { v: "true" })).toEqual({ path: "/v", reason: "type" });
  });
  test("objects and buffers mismatch outside bytes", () => {
    const s = [{ name: "v", kind: "str" }] as Parameters<typeof rowsOf>[0];
    expect(mismatched(s, { v: { k: [1] } })).toEqual({ path: "/v", reason: "type" });
    expect(mismatched(s, { v: Buffer.from([1]) })).toEqual({ path: "/v", reason: "type" });
    const b = [{ name: "v", kind: "bytes" }] as Parameters<typeof rowsOf>[0];
    expect(mismatched(b, { v: "AQ==" })).toEqual({ path: "/v", reason: "type" });
  });
  test("classification maps verified codes and falls back with errno", () => {
    expect(
      domainOutcome(
        classifyMySQL("query_one", mysqlError("ERR_MYSQL_CONNECTION_REFUSED"), failures, contracts),
        "sql::connection_failed",
      ),
    ).toEqual({ phase: "query" });
    expect(
      domainOutcome(
        classifyMySQL("query_one", mysqlError("ERR_MYSQL_CONNECTION_CLOSED"), failures, contracts),
        "sql::connection_failed",
      ),
    ).toEqual({ phase: "query" });
    expect(
      domainOutcome(
        classifyMySQL("query_one", mysqlError("ERR_MYSQL_SERVER_ERROR", 1045), failures, contracts),
        "sql::connection_failed",
      ),
    ).toEqual({ phase: "auth" });
    expect(
      domainOutcome(
        classifyMySQL("execute", mysqlError("ERR_MYSQL_SERVER_ERROR", 1062), failures, contracts),
        "sql::constraint_failed",
      ),
    ).toEqual({ constraint: "ER_DUP_ENTRY" });
    expect(
      domainOutcome(
        classifyMySQL("execute", mysqlError("ERR_MYSQL_SYNTAX_ERROR", 1064), failures, contracts),
        "sql::query_failed",
      ),
    ).toEqual({ operation: "execute", code: "ER_PARSE_ERROR" });
    expect(
      domainOutcome(
        classifyMySQL("query_one", mysqlError("ERR_MYSQL_SERVER_ERROR", 1146), failures, contracts),
        "sql::query_failed",
      ),
    ).toEqual({ operation: "query_one", code: "ER_NO_SUCH_TABLE" });
    expect(
      domainOutcome(
        classifyMySQL("query_one", mysqlError("ERR_MYSQL_SERVER_ERROR", 1213), failures, contracts),
        "sql::query_failed",
      ),
    ).toEqual({ operation: "query_one", code: "ER_LOCK_DEADLOCK" });
    expect(
      domainOutcome(
        classifyMySQL("query_one", mysqlError("ERR_MYSQL_SERVER_ERROR", 1205), failures, contracts),
        "sql::query_failed",
      ),
    ).toEqual({ operation: "query_one", code: "ER_LOCK_WAIT_TIMEOUT" });
    expect(
      domainOutcome(
        classifyMySQL("query_one", mysqlError("ERR_MYSQL_SERVER_ERROR", 1054), failures, contracts),
        "sql::query_failed",
      ),
    ).toEqual({ operation: "query_one", code: "mysql_errno_1054" });
    expect(
      domainOutcome(
        classifyMySQL("query_one", mysqlError("ERR_MYSQL_SOMETHING"), failures, contracts),
        "sql::query_failed",
      ),
    ).toEqual({ operation: "query_one", code: "ERR_MYSQL_SOMETHING" });
    expect(() => classifyMySQL("query_one", new Error("boom"), failures, contracts)).toThrow(
      "boom",
    );
  });
  test("affected rows read the mysql count field", () => {
    const ok = mysqlAffectedRows("execute", { affectedRows: 2, count: 0 }, failures);
    expect(ok.kind).toBe("ok");
    if (ok.kind !== "ok") throw new Error("wrong outcome");
    expect(ok.value).toBe(2n);
    expect(
      domainOutcome(mysqlAffectedRows("execute", { count: 0 }, failures), "sql::query_failed"),
    ).toEqual({ operation: "execute", code: "bad_count" });
    expect(
      domainOutcome(
        mysqlAffectedRows("execute", { affectedRows: -1 }, failures),
        "sql::query_failed",
      ),
    ).toEqual({ operation: "execute", code: "bad_count" });
    expect(
      domainOutcome(mysqlAffectedRows("execute", null, failures), "sql::query_failed"),
    ).toEqual({ operation: "execute", code: "bad_count" });
  });
});
