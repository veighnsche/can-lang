import { describe, expect, test } from "bun:test";
import { createHash } from "node:crypto";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, domainFailureDiagnostics, type FailureShape } from "../domain.ts";
import { success, value, type Completion } from "../completion.ts";
import { standardFailureDiagnostics, type StandardFailure } from "../failure.ts";
import { assertionContext } from "../assert/context.ts";
import { record, dataProperty } from "../data.ts";
import { ownBytes, isBytes, copyBytes } from "../bytes.ts";
import { runOwnedRoot, resourceStatus } from "../owner.ts";
import { createSQLDescriptors, type SQLDescriptorEntry } from "../platform/sql/descriptor.ts";
import { createSQLPools, isSQLPoolValue } from "../platform/sql/pool.ts";
import type { SQLPlan } from "../platform/sql/values.ts";

const origin = { source: "can:test", start: 0, end: 0, invocation: [] };
async function owned(body: () => Promise<void>): Promise<void> {
  const result = await runOwnedRoot(async () => { await body(); return success(undefined); });
  // runOwnedRoot captures body throws into the completion; asserting here
  // keeps a failing expectation from passing vacuously.
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
const table: Record<string, Record<string, SQLDescriptorEntry>> = {
  "": {
    account_by_id: {
      cardinality: "one", kind: "SelectStmt",
      segments: [{ text: "SELECT id, display_name FROM t WHERE id = " }, { param: 1 }, { text: " LIMIT " }, { param: 2 }],
      params: ["id"], paramType: "p", rowType: "r", limit: 2, total: 2, version: 170007,
    },
    accounts_by_term: {
      cardinality: "many", kind: "SelectStmt",
      segments: [{ text: "SELECT id, display_name FROM t WHERE display_name ILIKE " }, { param: 1 }, { text: " LIMIT " }, { param: 2 }],
      params: ["term"], paramType: "p", rowType: "r", limit: 2, total: 2, version: 170007,
    },
    add_account: {
      cardinality: "execute", kind: "InsertStmt",
      segments: [{ text: "INSERT INTO t (display_name) VALUES (" }, { param: 1 }, { text: ")" }],
      params: ["term"], paramType: "p", rowType: "r", limit: 0, total: 1, version: 170007,
    },
  },
};
const descriptors = createSQLDescriptors(table);
const env = new Map<string, string>();
const reads: string[] = [];
const pools = createSQLPools(domain, {
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
}, (name: string) => { reads.push(name); return env.get(name); }, descriptors);
const byId = descriptors.declareDescriptor("", "account_by_id");
const byTerm = descriptors.declareDescriptor("", "accounts_by_term");
const addAccount = descriptors.declareDescriptor("", "add_account");

function domainOutcome(completion: Completion<unknown>, name: string): Record<string, unknown> {
  expect(completion.kind).toBe("domain");
  if (completion.kind !== "domain") throw new Error("wrong outcome");
  const details = domainFailureDiagnostics(completion.value);
  expect(details.declaration.name).toBe(name);
  // Return a plain copy: the record carries an enumerable nominal symbol
  // that deep equality would otherwise count as an extra key.
  const payload = details.payload as Record<string, unknown>;
  const plain: Record<string, unknown> = {};
  for (const key of Object.keys(payload)) plain[key] = payload[key];
  return plain;
}

type FakeCall = { strings: string[]; values: unknown[] };
type FakeHandle = {
  calls: FakeCall[];
  connectCalls: number;
  closeCalls: unknown[];
  query: (strings: readonly string[], values: readonly unknown[]) => Promise<unknown>;
  connect: () => Promise<void>;
  close: (options?: unknown) => Promise<void>;
};
const RealSQL = Bun.SQL;
function installFake(hooks?: { connect?: () => Promise<void> }): { constructed: { url: unknown; opts: unknown }[]; clients: FakeHandle[] } {
  const constructed: { url: unknown; opts: unknown }[] = [];
  const clients: FakeHandle[] = [];
  (Bun as unknown as { SQL: unknown }).SQL = function (url: unknown, opts: unknown) {
    constructed.push({ url, opts });
    const handle: FakeHandle = {
      calls: [], connectCalls: 0, closeCalls: [],
      query: async () => [], connect: hooks?.connect ?? (async () => {}), close: async () => {},
    };
    const callable = async function (strings: readonly string[], ...values: unknown[]) {
      handle.calls.push({ strings: [...strings], values });
      return handle.query(strings, values);
    };
    Object.assign(callable, {
      connect: async () => { handle.connectCalls++; return handle.connect(); },
      close: async (options?: unknown) => { handle.closeCalls.push(options); return handle.close(options); },
    });
    clients.push(handle);
    return callable;
  };
  return { constructed, clients };
}
function restoreSQL(): void { (Bun as unknown as { SQL: unknown }).SQL = RealSQL; }

function postgresError(code: string, extra?: Record<string, unknown>): Error {
  const cause = new Error("native message must never surface: " + code);
  cause.name = "PostgresError";
  (cause as unknown as Record<string, unknown>).code = code;
  (cause as unknown as Record<string, unknown>).detail = "secret detail";
  Object.assign(cause, extra);
  return cause;
}

const idParams: SQLPlan = {
  params: { root: "app::id_parameters", fields: [{ name: "id", kind: "int" }] },
  rows: { root: "app::account_row", fields: [{ name: "id", kind: "int" }, { name: "display_name", kind: "str" }] },
};
const termParams: SQLPlan = {
  params: { root: "app::search_parameters", fields: [{ name: "term", kind: "str" }] },
  rows: { root: "app::account_row", fields: [{ name: "id", kind: "int" }, { name: "display_name", kind: "str" }] },
  some: "app::option_some", none: "app::option_none",
};
const coverPlan: SQLPlan = {
  params: { root: "app::id_parameters", fields: [{ name: "id", kind: "int" }] },
  rows: {
    root: "app::cover_row",
    fields: [
      { name: "id", kind: "int" },
      { name: "payload", kind: "bytes" },
      { name: "note", kind: "option", inner: "str", some: "app::option_some", none: "app::option_none" },
    ],
  },
};

function fixtureContext() {
  return assertionContext({ package: "app", declaration: "probe", name: "probe" });
}

describe("sql pools", () => {
  test("open reads the named credential once and constructs a postgres client", async () => {
    const fake = installFake();
    try {
      env.set("CAN_TEST_POSTGRES", "postgres://fake/x");
      reads.length = 0;
      await owned(async () => {
        const token = value(await pools.open("CAN_TEST_POSTGRES", 5n));
        expect(isSQLPoolValue("pool", token)).toBe(true);
        expect(isSQLPoolValue("pool", {})).toBe(false);
        expect(isSQLPoolValue("other", token)).toBe(false);
        value(await pools.close(token, 1000n));
      });
      expect(reads).toEqual(["CAN_TEST_POSTGRES"]);
      expect(fake.constructed).toEqual([{ url: "postgres://fake/x", opts: { adapter: "postgres", bigint: true, max: 5 } }]);
      expect(fake.clients[0]!.connectCalls).toBe(1);
    } finally { env.clear(); restoreSQL(); }
  });
  test("open rejects bad names, missing credentials, and bad config", async () => {
    const fake = installFake();
    try {
      env.clear();
      await owned(async () => {
        expect(domainOutcome(await pools.open("lower", 5n), "http::credentials_missing")).toEqual({ variable: "lower" });
        expect(domainOutcome(await pools.open("CAN_MISSING", 5n), "http::credentials_missing")).toEqual({ variable: "CAN_MISSING" });
      });
      env.set("CAN_TEST_POSTGRES", "postgres://fake/x");
      await owned(async () => {
        expect(domainOutcome(await pools.open("CAN_TEST_POSTGRES", 0n), "sql::connection_failed")).toEqual({ phase: "config" });
        expect(domainOutcome(await pools.open("CAN_TEST_POSTGRES", 2147483648n), "sql::connection_failed")).toEqual({ phase: "config" });
        await expect(pools.open("CAN_TEST_POSTGRES", 5)).rejects.toThrow(TypeError);
      });
      expect(fake.constructed).toEqual([]);
    } finally { env.clear(); restoreSQL(); }
  });
  test("open surfaces refused connections and cleans up the client", async () => {
    const fake = installFake({ connect: async () => { throw postgresError("ERR_POSTGRES_CONNECTION_REFUSED"); } });
    try {
      env.set("CAN_TEST_POSTGRES", "postgres://fake/x");
      await owned(async () => {
        expect(domainOutcome(await pools.open("CAN_TEST_POSTGRES", 5n), "sql::connection_failed")).toEqual({ phase: "connect" });
      });
      // The failed client is closed so a refused pool never leaks a socket.
      expect(fake.clients[0]!.closeCalls.length).toBe(1);
    } finally { env.clear(); restoreSQL(); }
  });
  test("fixture boundaries never launch real SQL", async () => {
    const fake = installFake();
    try {
      env.set("CAN_TEST_POSTGRES", "postgres://fake/x");
      const context = fixtureContext();
      await owned(async () => {
        const token = value(await pools.open("CAN_TEST_POSTGRES", 5n));
        await expect(pools.open("CAN_TEST_POSTGRES", 5n, context)).rejects.toThrow();
        const params = record("app::id_parameters", [["id", 4n]]);
        await expect(pools.queryOne(byId, idParams, token, params, context)).rejects.toThrow();
        await expect(pools.execute(addAccount, termParams, token, record("app::search_parameters", [["term", "x"]]), context)).rejects.toThrow();
        await expect(pools.close(token, 1000n, context)).rejects.toThrow();
        expect(fake.clients[0]!.calls).toEqual([]);
        expect(value(await pools.close(token, 1000n))).toBe(undefined);
      });
      expect(fake.constructed.length).toBe(1);
    } finally { env.clear(); restoreSQL(); }
  });
});

describe("sql parameters", () => {
  async function withPool(run: (token: unknown, client: FakeHandle) => Promise<void>): Promise<FakeHandle[]> {
    const fake = installFake();
    try {
      env.set("CAN_TEST_POSTGRES", "postgres://fake/x");
      await owned(async () => {
        const token = value(await pools.open("CAN_TEST_POSTGRES", 5n));
        try { await run(token, fake.clients[0]!); }
        finally { value(await pools.close(token, 1000n)); }
      });
      return fake.clients;
    } finally { env.clear(); restoreSQL(); }
  }
  test("int range and unicode scalars validate before launch", async () => {
    await withPool(async (token, client) => {
      const over = record("app::id_parameters", [["id", 1n << 63n]]);
      expect(domainOutcome(await pools.queryOne(byId, idParams, token, over), "sql::unsupported_value"))
        .toEqual({ path: "/id", reason: "int_range" });
      const lone = record("app::search_parameters", [["term", "\ud800"]]);
      expect(domainOutcome(await pools.queryRows(byTerm, termParams, token, lone, 10n), "sql::unsupported_value"))
        .toEqual({ path: "/term", reason: "unicode_scalar" });
      const mistyped = record("app::id_parameters", [["id", 4]]);
      expect(domainOutcome(await pools.queryOne(byId, idParams, token, mistyped), "sql::unsupported_value"))
        .toEqual({ path: "/id", reason: "type" });
      expect(client.calls).toEqual([]);
    });
  });
  test("options encode to null or the inner scalar", async () => {
    const plan: SQLPlan = {
      params: { root: "app::opt", fields: [{ name: "note", kind: "option", inner: "str", some: "app::option_some", none: "app::option_none" }] },
      rows: idParams.rows,
    };
    await withPool(async (token, client) => {
      client.query = async () => [{ id: 1n, display_name: "n" }];
      const none = record("app::opt", [["note", record("app::option_none", [])]]);
      expect((await pools.queryOne(byId, plan, token, none)).kind).toBe("ok");
      const some = record("app::opt", [["note", record("app::option_some", [["value", "hi"]])]]);
      expect((await pools.queryOne(byId, plan, token, some)).kind).toBe("ok");
      expect(client.calls.map(call => call.values[0])).toEqual([null, "hi"]);
      const forged = record("app::opt", [["note", record("app::other", [["value", "hi"]])]]);
      expect(domainOutcome(await pools.queryOne(byId, plan, token, forged), "sql::unsupported_value"))
        .toEqual({ path: "/note", reason: "option_shape" });
      expect(client.calls.length).toBe(2);
    });
  });
  test("hostile text travels only as a bound parameter", async () => {
    await withPool(async (token, client) => {
      client.query = async () => [];
      const hostile = "x' OR '1'='1'; DROP TABLE t; --";
      const params = record("app::search_parameters", [["term", hostile]]);
      expect((await pools.queryRows(byTerm, termParams, token, params, 10n)).kind).toBe("ok");
      expect(client.calls.length).toBe(1);
      const call = client.calls[0]!;
      expect(call.values[0]).toBe(hostile);
      expect(call.strings.join("\0")).not.toContain(hostile);
      expect(call.strings).toEqual(["SELECT id, display_name FROM t WHERE display_name ILIKE ", " LIMIT ", ""]);
    });
  });
  test("bytes parameters are copied before launch", async () => {
    const plan: SQLPlan = {
      params: { root: "app::bin", fields: [{ name: "payload", kind: "bytes" }] },
      rows: idParams.rows,
    };
    await withPool(async (token, client) => {
      let sent: unknown;
      client.query = async (_strings, values) => { sent = values[0]; return [{ id: 1n, display_name: "n" }]; };
      const backing = new Uint8Array([1, 2, 3]);
      const params = record("app::bin", [["payload", ownBytes(backing)]]);
      expect((await pools.queryOne(byId, plan, token, params)).kind).toBe("ok");
      backing[0] = 9;
      expect(sent).toEqual(new Uint8Array([1, 2, 3]));
    });
  });
});

describe("sql rows", () => {
  test("query_one binds LIMIT 2 and decodes an immutable record", async () => {
    const fake = installFake();
    try {
      env.set("CAN_TEST_POSTGRES", "postgres://fake/x");
      await owned(async () => {
        const token = value(await pools.open("CAN_TEST_POSTGRES", 5n));
        const client = fake.clients[0]!;
        client.query = async () => [{ id: 4n, display_name: "Bob" }];
        const params = record("app::id_parameters", [["id", 4n]]);
        const row = value(await pools.queryOne(byId, idParams, token, params));
        expect(dataProperty(row, "id")).toBe(4n);
        expect(dataProperty(row, "display_name")).toBe("Bob");
        expect(Object.isFrozen(row)).toBe(true);
        expect(client.calls[0]!.values).toEqual([4n, 2]);
        value(await pools.close(token, 1000n));
      });
    } finally { env.clear(); restoreSQL(); }
  });
  test("missing and extra columns classify by path", async () => {
    const fake = installFake();
    try {
      env.set("CAN_TEST_POSTGRES", "postgres://fake/x");
      await owned(async () => {
        const token = value(await pools.open("CAN_TEST_POSTGRES", 5n));
        const client = fake.clients[0]!;
        const params = record("app::id_parameters", [["id", 4n]]);
        client.query = async () => [{ id: 4n }];
        expect(domainOutcome(await pools.queryOne(byId, idParams, token, params), "sql::schema_mismatch"))
          .toEqual({ path: "/display_name", reason: "missing_column" });
        client.query = async () => [{ id: 4n, display_name: "Bob", extra: 1 }];
        expect(domainOutcome(await pools.queryOne(byId, idParams, token, params), "sql::schema_mismatch"))
          .toEqual({ path: "/extra", reason: "extra_column" });
        client.query = async () => [[4n, "Bob"]];
        expect(domainOutcome(await pools.queryOne(byId, idParams, token, params), "sql::schema_mismatch"))
          .toEqual({ path: "", reason: "type" });
        value(await pools.close(token, 1000n));
      });
    } finally { env.clear(); restoreSQL(); }
  });
  test("null, unsafe numbers, and binary decode by rule", async () => {
    const fake = installFake();
    try {
      env.set("CAN_TEST_POSTGRES", "postgres://fake/x");
      await owned(async () => {
        const token = value(await pools.open("CAN_TEST_POSTGRES", 5n));
        const client = fake.clients[0]!;
        const params = record("app::id_parameters", [["id", 1n]]);
        client.query = async () => [{ id: 1n, payload: new Uint8Array([65]), note: null }];
        const row = value(await pools.queryOne(byId, coverPlan, token, params));
        expect(dataProperty(row, "note")).toEqual(record("app::option_none", []));
        expect(isBytes(dataProperty(row, "payload"))).toBe(true);
        client.query = async () => [{ id: null, payload: new Uint8Array(), note: null }];
        expect(domainOutcome(await pools.queryOne(byId, coverPlan, token, params), "sql::schema_mismatch"))
          .toEqual({ path: "/id", reason: "null" });
        client.query = async () => [{ id: 2 ** 53, display_name: "n" }];
        expect(domainOutcome(await pools.queryOne(byId, idParams, token, params), "sql::schema_mismatch"))
          .toEqual({ path: "/id", reason: "unsafe_integer" });
        client.query = async () => [{ id: 7, display_name: "n" }];
        expect(dataProperty(value(await pools.queryOne(byId, idParams, token, params)), "id")).toBe(7n);
        client.query = async () => [{ id: 1n << 63n, display_name: "n" }];
        expect(domainOutcome(await pools.queryOne(byId, idParams, token, params), "sql::schema_mismatch"))
          .toEqual({ path: "/id", reason: "int_range" });
        value(await pools.close(token, 1000n));
      });
    } finally { env.clear(); restoreSQL(); }
  });
  test("decoded binary is an owned copy", async () => {
    const fake = installFake();
    try {
      env.set("CAN_TEST_POSTGRES", "postgres://fake/x");
      await owned(async () => {
        const token = value(await pools.open("CAN_TEST_POSTGRES", 5n));
        const client = fake.clients[0]!;
        const backing = new Uint8Array([65, 66]);
        client.query = async () => [{ id: 1n, payload: backing, note: null }];
        const row = value(await pools.queryOne(byId, coverPlan, token, record("app::id_parameters", [["id", 1n]])));
        backing[0] = 90;
        expect(copyBytes(dataProperty(row, "payload"), origin)).toEqual(new Uint8Array([65, 66]));
        value(await pools.close(token, 1000n));
      });
    } finally { env.clear(); restoreSQL(); }
  });
  test("optional decodes none, some, and observed duplicates", async () => {
    const fake = installFake();
    try {
      env.set("CAN_TEST_POSTGRES", "postgres://fake/x");
      await owned(async () => {
        const token = value(await pools.open("CAN_TEST_POSTGRES", 5n));
        const client = fake.clients[0]!;
        const params = record("app::search_parameters", [["term", "Bob"]]);
        client.query = async () => [];
        expect(value(await pools.queryOptional(byTerm, termParams, token, params))).toEqual(record("app::option_none", []));
        client.query = async () => [{ id: 4n, display_name: "Bob" }, { id: 5n, display_name: "Bob" }];
        expect(domainOutcome(await pools.queryOptional(byTerm, termParams, token, params), "sql::row_count"))
          .toEqual({ query: "accounts_by_term", actual: 2n });
        client.query = async () => [{ id: 4n, display_name: "Bob" }];
        const some = value(await pools.queryOptional(byTerm, termParams, token, params));
        expect(dataProperty(dataProperty(some, "value"), "id")).toBe(4n);
        expect(client.calls.every(call => call.values[call.values.length - 1] === 2)).toBe(true);
        value(await pools.close(token, 1000n));
      });
    } finally { env.clear(); restoreSQL(); }
  });
  test("cardinality counts only observed bounded rows", async () => {
    const fake = installFake();
    try {
      env.set("CAN_TEST_POSTGRES", "postgres://fake/x");
      await owned(async () => {
        const token = value(await pools.open("CAN_TEST_POSTGRES", 5n));
        const client = fake.clients[0]!;
        const params = record("app::id_parameters", [["id", 4n]]);
        client.query = async () => [];
        expect(domainOutcome(await pools.queryOne(byId, idParams, token, params), "sql::row_missing"))
          .toEqual({ query: "account_by_id" });
        client.query = async () => [{ id: 4n, display_name: "a" }, { id: 4n, display_name: "b" }];
        expect(domainOutcome(await pools.queryOne(byId, idParams, token, params), "sql::row_count"))
          .toEqual({ query: "account_by_id", actual: 2n });
        value(await pools.close(token, 1000n));
      });
    } finally { env.clear(); restoreSQL(); }
  });
  test("query_rows binds max_rows+1 and enforces the exact bound", async () => {
    const fake = installFake();
    try {
      env.set("CAN_TEST_POSTGRES", "postgres://fake/x");
      await owned(async () => {
        const token = value(await pools.open("CAN_TEST_POSTGRES", 5n));
        const client = fake.clients[0]!;
        const params = record("app::search_parameters", [["term", "son"]]);
        client.query = async () => [{ id: 2n, display_name: "Jason" }, { id: 3n, display_name: "Mason" }];
        const rows = value(await pools.queryRows(byTerm, termParams, token, params, 2n));
        expect((rows as readonly unknown[]).length).toBe(2);
        expect(client.calls[0]!.values).toEqual(["son", 3n]);
        client.query = async () => [{ id: 2n, display_name: "a" }, { id: 3n, display_name: "b" }, { id: 4n, display_name: "c" }];
        expect(domainOutcome(await pools.queryRows(byTerm, termParams, token, params, 2n), "sql::row_limit"))
          .toEqual({ limit: 2n });
        expect(domainOutcome(await pools.queryRows(byTerm, termParams, token, params, -1n), "sql::unsupported_value"))
          .toEqual({ path: "max_rows", reason: "negative" });
        expect(domainOutcome(await pools.queryRows(byTerm, termParams, token, params, (1n << 63n) - 1n), "sql::unsupported_value"))
          .toEqual({ path: "max_rows", reason: "overflow" });
        value(await pools.close(token, 1000n));
      });
    } finally { env.clear(); restoreSQL(); }
  });
  test("execute validates affected counts and binds no limit", async () => {
    const fake = installFake();
    try {
      env.set("CAN_TEST_POSTGRES", "postgres://fake/x");
      await owned(async () => {
        const token = value(await pools.open("CAN_TEST_POSTGRES", 5n));
        const client = fake.clients[0]!;
        const params = record("app::search_parameters", [["term", "Zed"]]);
        client.query = async () => ({ count: 1 });
        expect(value(await pools.execute(addAccount, termParams, token, params))).toBe(1n);
        expect(client.calls[0]!.values).toEqual(["Zed"]);
        client.query = async () => ({ count: 1.5 });
        expect(domainOutcome(await pools.execute(addAccount, termParams, token, params), "sql::query_failed"))
          .toEqual({ operation: "execute", code: "bad_count" });
        client.query = async () => ({});
        expect(domainOutcome(await pools.execute(addAccount, termParams, token, params), "sql::query_failed"))
          .toEqual({ operation: "execute", code: "bad_count" });
        value(await pools.close(token, 1000n));
      });
    } finally { env.clear(); restoreSQL(); }
  });
});

describe("sql failures", () => {
  test("native failures classify finitely with sanitized payloads", async () => {
    const fake = installFake();
    try {
      env.set("CAN_TEST_POSTGRES", "postgres://fake/x");
      await owned(async () => {
        const token = value(await pools.open("CAN_TEST_POSTGRES", 5n));
        const client = fake.clients[0]!;
        const params = record("app::id_parameters", [["id", 4n]]);
        client.query = async () => { throw postgresError("ERR_POSTGRES_CONNECTION_REFUSED"); };
        expect(domainOutcome(await pools.queryOne(byId, idParams, token, params), "sql::connection_failed"))
          .toEqual({ phase: "query" });
        client.query = async () => { throw postgresError("query", { errno: "23505", constraint: "accounts_name_key" }); };
        expect(domainOutcome(await pools.execute(addAccount, termParams, token, record("app::search_parameters", [["term", "Zed"]])), "sql::constraint_failed"))
          .toEqual({ constraint: "accounts_name_key" });
        client.query = async () => { throw postgresError("query", { errno: "23503" }); };
        expect(domainOutcome(await pools.queryOne(byId, idParams, token, params), "sql::constraint_failed"))
          .toEqual({ constraint: "23503" });
        client.query = async () => { throw postgresError("query", { errno: "42P01" }); };
        const failed = domainOutcome(await pools.queryOne(byId, idParams, token, params), "sql::query_failed");
        expect(failed).toEqual({ operation: "query_one", code: "42P01" });
        expect(JSON.stringify(failed)).not.toContain("secret");
        client.query = async () => { throw new Error("driver defect"); };
        const defect = await pools.queryOne(byId, idParams, token, params);
        expect(defect.kind).toBe("standard");
        if (defect.kind !== "standard") throw new Error("wrong outcome");
        const defectDetails = standardFailureDiagnostics(defect.value);
        expect(defectDetails.kind).toBe("native_exception");
        expect(defectDetails.message).toContain("driver defect");
        value(await pools.close(token, 1000n));
      });
    } finally { env.clear(); restoreSQL(); }
  });
});

describe("sql close", () => {
  test("close validates the deadline and closes natively", async () => {
    const fake = installFake();
    try {
      env.set("CAN_TEST_POSTGRES", "postgres://fake/x");
      await owned(async () => {
        const token = value(await pools.open("CAN_TEST_POSTGRES", 5n));
        expect(domainOutcome(await pools.close(token, -1n), "sql::close_failed"))
          .toEqual({ reason: "invalid_timeout" });
        expect(fake.clients[0]!.closeCalls).toEqual([]);
        expect(value(await pools.close(token, 1000n))).toBe(undefined);
        expect(fake.clients[0]!.closeCalls.length).toBe(1);
        expect(resourceStatus(token).state).toBe("closed");
      });
    } finally { env.clear(); restoreSQL(); }
  });
  test("close denies new work, drains the lease, and keeps closing on timeout", async () => {
    const fake = installFake();
    try {
      env.set("CAN_TEST_POSTGRES", "postgres://fake/x");
      // The intentional timeout marks root cleanup even though the close
      // later finishes: assert that contract instead of the helper's.
      const result = await runOwnedRoot(async () => {
        const token = value(await pools.open("CAN_TEST_POSTGRES", 5n));
        const client = fake.clients[0]!;
        let release!: () => void;
        const gate = new Promise<void>(resolve => { release = resolve; });
        client.query = async () => { await gate; return [{ id: 4n, display_name: "Bob" }]; };
        const params = record("app::id_parameters", [["id", 4n]]);
        const flight = pools.queryOne(byId, idParams, token, params);
        while (client.calls.length === 0) await Bun.sleep(0);
        expect(domainOutcome(await pools.close(token, 20n), "sql::close_failed")).toEqual({ reason: "timeout" });
        expect(resourceStatus(token).state).toBe("closing");
        release();
        expect(dataProperty(value(await flight), "id")).toBe(4n);
        const begin = Date.now();
        while (resourceStatus(token).state !== "closed" && Date.now() - begin < 5000) await Bun.sleep(10);
        expect(resourceStatus(token).state).toBe("closed");
        expect(client.closeCalls.length).toBe(1);
        // New work is denied once the pool is closed.
        try {
          await pools.queryOne(byId, idParams, token, params);
          throw new Error("query admitted on a closed pool");
        } catch (cause) {
          expect(standardFailureDiagnostics(cause as StandardFailure).kind).toBe("resource_state");
        }
        return success(undefined);
      });
      expect(result.completion.kind).toBe("ok");
      expect(result.cleanupFailed).toBe(true);
    } finally { env.clear(); restoreSQL(); }
  });
  test("close on a non-pool token throws a resource failure", async () => {
    const fake = installFake();
    try {
      env.set("CAN_TEST_POSTGRES", "postgres://fake/x");
      await owned(async () => {
        try {
          await pools.close({}, 1000n);
          throw new Error("close admitted a non-pool");
        } catch (cause) {
          expect(standardFailureDiagnostics(cause as StandardFailure).kind).toBe("resource_state");
        }
      });
    } finally { env.clear(); restoreSQL(); }
  });
});

