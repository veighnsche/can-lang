import { describe, expect, test } from "bun:test";
import { createHash } from "node:crypto";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, domainFailureDiagnostics, type FailureShape } from "../domain.ts";
import { success, value, type Completion } from "../completion.ts";
import { standardFailureDiagnostics, type StandardFailure } from "../failure.ts";
import { assertionContext } from "../assert/context.ts";
import { record, dataProperty } from "../data.ts";
import { runOwnedRoot, registerResource, resourceStatus } from "../owner.ts";
import { createSQLDescriptors, type SQLDescriptorEntry } from "../platform/sql/descriptor.ts";
import { createSQLPools } from "../platform/sql/pool.ts";
import type { SQLPlan } from "../platform/sql/values.ts";
import { createSQLTransactions, isSQLTransactionValue } from "../platform/sql/transaction.ts";

const origin = { source: "can:test", start: 0, end: 0, invocation: [] };
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
const table: Record<string, Record<string, SQLDescriptorEntry>> = {
  "": {
    tx_add: {
      dialect: "postgresql", cardinality: "execute", kind: "InsertStmt",
      segments: [{ text: "INSERT INTO t (display_name) VALUES (" }, { param: 1 }, { text: ")" }],
      params: ["term"], paramType: "p", rowType: "r", limit: 0, total: 1, version: 170007,
    },
    tx_get: {
      dialect: "postgresql", cardinality: "one", kind: "SelectStmt",
      segments: [{ text: "SELECT id, display_name FROM t WHERE id = " }, { param: 1 }, { text: " LIMIT " }, { param: 2 }],
      params: ["id"], paramType: "p", rowType: "r", limit: 2, total: 2, version: 170007,
    },
  },
};
const descriptors = createSQLDescriptors(table);
const env = new Map<string, string>();
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
}, (name: string) => env.get(name), descriptors);
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
const txAdd = descriptors.declareDescriptor("", "tx_add");
const txGet = descriptors.declareDescriptor("", "tx_get");

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
function standardKind(cause: unknown): string {
  return standardFailureDiagnostics(cause as StandardFailure).kind;
}

type FakeCall = { strings: string[]; values: unknown[] };
type FakeTx = {
  calls: FakeCall[];
  query: (strings: readonly string[], values: readonly unknown[]) => Promise<unknown>;
};
type FakeHandle = {
  txs: FakeTx[];
  beginCalls: number;
  begin: (run: (tx: unknown) => Promise<unknown>) => Promise<unknown>;
  connect: () => Promise<void>;
  close: (options?: unknown) => Promise<void>;
};
const RealSQL = Bun.SQL;
function installFake(hooks?: { begin?: (run: (tx: unknown) => Promise<unknown>, tx: FakeTx) => Promise<unknown> }): { clients: FakeHandle[] } {
  const clients: FakeHandle[] = [];
  (Bun as unknown as { SQL: unknown }).SQL = function () {
    const handle: FakeHandle = {
      txs: [], beginCalls: 0,
      begin: async () => { throw new Error("begin hook missing"); },
      connect: async () => {}, close: async () => {},
    };
    const callable = async function () { throw new Error("pool template calls are out of scope here"); };
    Object.assign(callable, {
      connect: async () => handle.connect(),
      close: async (options?: unknown) => handle.close(options),
      begin: async (run: (tx: unknown) => Promise<unknown>) => {
        handle.beginCalls++;
        const tx: FakeTx = { calls: [], query: async () => [] };
        const native = async function (strings: readonly string[], ...values: unknown[]) {
          tx.calls.push({ strings: [...strings], values });
          return tx.query(strings, values);
        };
        handle.txs.push(tx);
        if (hooks?.begin) return hooks.begin(run, tx);
        return run(native);
      },
    });
    clients.push(handle);
    return callable;
  };
  return { clients };
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

const COMMIT = "app::sql_commit_leaf";
const ROLLBACK = "app::sql_rollback_leaf";
const leaves = { commit: COMMIT, rollback: ROLLBACK };
const addPlan: SQLPlan = {
  params: { root: "app::search_parameters", fields: [{ name: "term", kind: "str" }] },
  rows: { root: "app::account_row", fields: [{ name: "id", kind: "int" }, { name: "display_name", kind: "str" }] },
};
const getPlan: SQLPlan = {
  params: { root: "app::id_parameters", fields: [{ name: "id", kind: "int" }] },
  rows: { root: "app::account_row", fields: [{ name: "id", kind: "int" }, { name: "display_name", kind: "str" }] },
};
function fixtureContext() {
  return assertionContext({ package: "app", declaration: "probe", name: "probe" });
}

describe("sql transactions", () => {
  test("commit runs scoped queries and returns the committed value", async () => {
    const fake = installFake();
    try {
      env.set("CAN_TEST_POSTGRES", "postgres://fake/x");
      await owned(async () => {
        const token = value(await pools.open("CAN_TEST_POSTGRES", 5n));
        const client = fake.clients[0]!;
        client.txs.length = 0;
        let seen: unknown;
        const outcome = await transactions.withTransaction(token, async (handle: unknown) => {
          seen = handle;
          expect(isSQLTransactionValue("transaction", handle)).toBe(true);
          expect(isSQLTransactionValue("pool", handle)).toBe(false);
          expect(isSQLTransactionValue("transaction", {})).toBe(false);
          const tx = client.txs[0]!;
          tx.query = async () => ({ count: 1 });
          const affected = value(await transactions.execute(txAdd, addPlan, handle, record("app::search_parameters", [["term", "Zed"]])));
          return success(record(COMMIT, [["value", affected]]));
        }, leaves);
        expect(value(outcome)).toBe(1n);
        expect(client.beginCalls).toBe(1);
        expect(client.txs.length).toBe(1);
        expect(client.txs[0]!.calls[0]!.strings).toEqual(["INSERT INTO t (display_name) VALUES (", ")"]);
        expect(client.txs[0]!.calls[0]!.values).toEqual(["Zed"]);
        expect(resourceStatus(seen).kind).toBe("sql-tx");
        expect(resourceStatus(seen).state).toBe("closed");
        value(await pools.close(token, 1000n));
      });
    } finally { env.clear(); restoreSQL(); }
  });
  test("rollback returns the rollback value without committing", async () => {
    const fake = installFake();
    try {
      env.set("CAN_TEST_POSTGRES", "postgres://fake/x");
      await owned(async () => {
        const token = value(await pools.open("CAN_TEST_POSTGRES", 5n));
        const outcome = await transactions.withTransaction(token, async () => {
          return success(record(ROLLBACK, [["value", 7n]]));
        }, leaves);
        expect(value(outcome)).toBe(7n);
        expect(fake.clients[0]!.beginCalls).toBe(1);
        value(await pools.close(token, 1000n));
      });
    } finally { env.clear(); restoreSQL(); }
  });
  test("in-transaction queries share pool validation and classification", async () => {
    const fake = installFake();
    try {
      env.set("CAN_TEST_POSTGRES", "postgres://fake/x");
      await owned(async () => {
        const token = value(await pools.open("CAN_TEST_POSTGRES", 5n));
        const client = fake.clients[0]!;
        const outcome = await transactions.withTransaction(token, async (handle: unknown) => {
          const tx = client.txs[0]!;
          tx.query = async () => [{ id: 4n, display_name: "Bob" }];
          const row = value(await transactions.queryOne(txGet, getPlan, handle, record("app::id_parameters", [["id", 4n]])));
          expect(dataProperty(row, "display_name")).toBe("Bob");
          expect(tx.calls[0]!.values).toEqual([4n, 2]);
          tx.query = async () => { throw postgresError("ERR_POSTGRES_CONNECTION_REFUSED"); };
          const refused = await transactions.queryOne(txGet, getPlan, handle, record("app::id_parameters", [["id", 4n]]));
          expect(domainOutcome(refused, "sql::connection_failed")).toEqual({ phase: "query" });
          const over = record("app::id_parameters", [["id", 1n << 63n]]);
          expect(domainOutcome(await transactions.queryOne(txGet, getPlan, handle, over), "sql::unsupported_value"))
            .toEqual({ path: "/id", reason: "int_range" });
          expect(tx.calls.length).toBe(2);
          return success(record(COMMIT, [["value", 4n]]));
        }, leaves);
        expect(value(outcome)).toBe(4n);
        // A pool token is not a transaction handle.
        try {
          await transactions.queryOne(txGet, getPlan, token, record("app::id_parameters", [["id", 4n]]));
          throw new Error("tx query admitted a pool token");
        } catch (cause) {
          expect(standardKind(cause)).toBe("resource_state");
        }
        value(await pools.close(token, 1000n));
      });
    } finally { env.clear(); restoreSQL(); }
  });
  test("the handle dies with its scope", async () => {
    const fake = installFake();
    try {
      env.set("CAN_TEST_POSTGRES", "postgres://fake/x");
      await owned(async () => {
        const token = value(await pools.open("CAN_TEST_POSTGRES", 5n));
        let held: unknown;
        const outcome = await transactions.withTransaction(token, async (handle: unknown) => {
          held = handle;
          return success(record(COMMIT, [["value", 1n]]));
        }, leaves);
        expect(value(outcome)).toBe(1n);
        try {
          await transactions.queryOne(txGet, getPlan, held, record("app::id_parameters", [["id", 4n]]));
          throw new Error("escaped handle admitted work");
        } catch (cause) {
          expect(standardKind(cause)).toBe("resource_state");
        }
        value(await pools.close(token, 1000n));
      });
    } finally { env.clear(); restoreSQL(); }
  });
  test("refused begin reports the connection failure with the begin phase", async () => {
    const fake = installFake({ begin: async () => { throw postgresError("ERR_POSTGRES_CONNECTION_REFUSED"); } });
    try {
      env.set("CAN_TEST_POSTGRES", "postgres://fake/x");
      await owned(async () => {
        const token = value(await pools.open("CAN_TEST_POSTGRES", 5n));
        let entered = false;
        const outcome = await transactions.withTransaction(token, async () => {
          entered = true;
          return success(record(COMMIT, [["value", 1n]]));
        }, leaves);
        expect(entered).toBe(false);
        expect(domainOutcome(outcome, "sql::connection_failed")).toEqual({ phase: "begin" });
        value(await pools.close(token, 1000n));
      });
    } finally { env.clear(); restoreSQL(); }
  });
  test("native callback failures classify by phase", async () => {
    const fake = installFake({
      begin: async (run) => { await run(async () => { throw new Error("unreachable"); }); throw postgresError("callback broke", { errno: "40001" }); },
    });
    try {
      env.set("CAN_TEST_POSTGRES", "postgres://fake/x");
      await owned(async () => {
        const token = value(await pools.open("CAN_TEST_POSTGRES", 5n));
        const outcome = await transactions.withTransaction(token, async () => {
          // A rollback decision keeps the native callback failure in the
          // callback phase: the commit was never entered.
          return success(record(ROLLBACK, [["value", 0n]]));
        }, leaves);
        // The rollback sentinel diverts before the native rejection matters.
        expect(value(outcome)).toBe(0n);
        value(await pools.close(token, 1000n));
      });
    } finally { env.clear(); restoreSQL(); }
  });
  test("non-postgres begin failures are transactional", async () => {
    const fake = installFake({ begin: async () => { throw postgresError("weird", { errno: "XX000" }); } });
    try {
      env.set("CAN_TEST_POSTGRES", "postgres://fake/x");
      await owned(async () => {
        const token = value(await pools.open("CAN_TEST_POSTGRES", 5n));
        const outcome = await transactions.withTransaction(token, async () => {
          return success(record(COMMIT, [["value", 1n]]));
        }, leaves);
        expect(domainOutcome(outcome, "sql::transaction_failed")).toEqual({ phase: "begin" });
        value(await pools.close(token, 1000n));
      });
    } finally { env.clear(); restoreSQL(); }
  });
  test("foreign throws propagate instead of classifying", async () => {
    const fake = installFake({ begin: async () => { throw new Error("driver defect"); } });
    try {
      env.set("CAN_TEST_POSTGRES", "postgres://fake/x");
      await owned(async () => {
        const token = value(await pools.open("CAN_TEST_POSTGRES", 5n));
        const defect = await transactions.withTransaction(token, async () => {
          return success(record(COMMIT, [["value", 1n]]));
        }, leaves);
        expect(defect.kind).toBe("standard");
        if (defect.kind !== "standard") throw new Error("wrong outcome");
        const details = standardFailureDiagnostics(defect.value);
        expect(details.kind).toBe("native_exception");
        expect(details.message).toContain("driver defect");
        value(await pools.close(token, 1000n));
      });
    } finally { env.clear(); restoreSQL(); }
  });
  test("nested entry surfaces the scope failure verbatim", async () => {
    const fake = installFake();
    try {
      env.set("CAN_TEST_POSTGRES", "postgres://fake/x");
      await owned(async () => {
        const token = value(await pools.open("CAN_TEST_POSTGRES", 5n));
        const outcome = await transactions.withTransaction(token, async () => {
          const inner = await transactions.withTransaction(token, async () => {
            return success(record(COMMIT, [["value", 1n]]));
          }, leaves);
          return inner as Completion<unknown>;
        }, leaves);
        expect(outcome.kind).toBe("standard");
        if (outcome.kind !== "standard") throw new Error("wrong outcome");
        expect(standardFailureDiagnostics(outcome.value).kind).toBe("resource_state");
        expect(fake.clients[0]!.beginCalls).toBe(1);
        value(await pools.close(token, 1000n));
      });
    } finally { env.clear(); restoreSQL(); }
  });
  test("primary domain failures return verbatim", async () => {
    const fake = installFake();
    try {
      env.set("CAN_TEST_POSTGRES", "postgres://fake/x");
      await owned(async () => {
        const token = value(await pools.open("CAN_TEST_POSTGRES", 5n));
        const client = fake.clients[0]!;
        const outcome = await transactions.withTransaction(token, async (handle: unknown) => {
          const tx = client.txs[0]!;
          tx.query = async () => [];
          const primary = await transactions.queryOne(txGet, getPlan, handle, record("app::id_parameters", [["id", 999n]]));
          expect(primary.kind).toBe("domain");
          return primary;
        }, leaves);
        expect(domainOutcome(outcome, "sql::row_missing")).toEqual({ query: "tx_get" });
        value(await pools.close(token, 1000n));
      });
    } finally { env.clear(); restoreSQL(); }
  });
  test("rejection after a commit decision is commit-unknown with a safe id", async () => {
    const fake = installFake({
      begin: async (run, tx) => {
        tx.query = async () => ({ count: 1 });
        const decided = await run(async function (strings: readonly string[], ...values: unknown[]) {
          tx.calls.push({ strings: [...strings], values });
          return tx.query(strings, values);
        });
        throw postgresError("commit broke: " + String(decided), { errno: "08006" });
      },
    });
    try {
      env.set("CAN_TEST_POSTGRES", "postgres://fake/x");
      await owned(async () => {
        const token = value(await pools.open("CAN_TEST_POSTGRES", 5n));
        const poolId = String(resourceStatus(token).id);
        const first = await transactions.withTransaction(token, async (handle: unknown) => {
          const affected = value(await transactions.execute(txAdd, addPlan, handle, record("app::search_parameters", [["term", "Zed"]])));
          return success(record(COMMIT, [["value", affected]]));
        }, leaves);
        const payload = domainOutcome(first, "sql::commit_unknown");
        expect(payload).toEqual({ transaction_id: `${poolId}-1` });
        expect(JSON.stringify(payload)).not.toContain("fake");
        expect(JSON.stringify(payload)).not.toContain("secret");
        const second = await transactions.withTransaction(token, async () => {
          return success(record(COMMIT, [["value", 2n]]));
        }, leaves);
        expect(domainOutcome(second, "sql::commit_unknown")).toEqual({ transaction_id: `${poolId}-2` });
        value(await pools.close(token, 1000n));
      });
    } finally { env.clear(); restoreSQL(); }
  });
  test("foreign throws after a commit decision are still commit-unknown", async () => {
    const fake = installFake({
      begin: async (run) => { await run(async () => ({ count: 1 })); throw new Error("socket died mid-commit"); },
    });
    try {
      env.set("CAN_TEST_POSTGRES", "postgres://fake/x");
      await owned(async () => {
        const token = value(await pools.open("CAN_TEST_POSTGRES", 5n));
        const poolId = String(resourceStatus(token).id);
        const outcome = await transactions.withTransaction(token, async () => {
          return success(record(COMMIT, [["value", 1n]]));
        }, leaves);
        expect(domainOutcome(outcome, "sql::commit_unknown")).toEqual({ transaction_id: `${poolId}-1` });
        value(await pools.close(token, 1000n));
      });
    } finally { env.clear(); restoreSQL(); }
  });
  test("commit waits for scope drain", async () => {
    let release!: () => void;
    const gate = new Promise<void>(resolve => { release = resolve; });
    let settled = false;
    let observed = false;
    const fake = installFake({
      begin: async (run, tx) => {
        const pending = run(async function (strings: readonly string[], ...values: unknown[]) {
          tx.calls.push({ strings: [...strings], values });
          return tx.query(strings, values);
        });
        void pending.then(() => { settled = true; }, () => { settled = true; });
        // The native callback stays pending through scope drain: the
        // commit decision alone must not resolve it.
        await Bun.sleep(25);
        expect(settled).toBe(false);
        observed = true;
        release();
        return pending;
      },
    });
    try {
      env.set("CAN_TEST_POSTGRES", "postgres://fake/x");
      await owned(async () => {
        const token = value(await pools.open("CAN_TEST_POSTGRES", 5n));
        let closed = false;
        const outcome = await transactions.withTransaction(token, async () => {
          registerResource("test-drain", {}, async () => {
            closed = true;
            await gate;
            return success(undefined);
          }, { scopeManaged: true });
          return success(record(COMMIT, [["value", 9n]]));
        }, leaves);
        expect(observed).toBe(true);
        expect(settled).toBe(true);
        expect(value(outcome)).toBe(9n);
        expect(closed).toBe(true);
        value(await pools.close(token, 1000n));
      });
    } finally { env.clear(); restoreSQL(); }
  });
  test("compiler-shape violations throw type errors", async () => {
    const fake = installFake();
    try {
      env.set("CAN_TEST_POSTGRES", "postgres://fake/x");
      await owned(async () => {
        const token = value(await pools.open("CAN_TEST_POSTGRES", 5n));
        const decide = async () => success(record(COMMIT, [["value", 1n]]));
        await expect(transactions.withTransaction(token, 5, leaves)).rejects.toThrow(TypeError);
        await expect(transactions.withTransaction(token, decide, {})).rejects.toThrow(TypeError);
        await expect(transactions.withTransaction(token, decide, { commit: COMMIT, rollback: COMMIT })).rejects.toThrow(TypeError);
        // A malformed decision surfaces as a standard completion through
        // the primary path, never as a second begin.
        const malformed = await transactions.withTransaction(token, async () => success(record("app::other", [["value", 1n]])), leaves);
        expect(malformed.kind).toBe("standard");
        if (malformed.kind !== "standard") throw new Error("wrong outcome");
        expect(standardFailureDiagnostics(malformed.value).kind).toBe("native_exception");
        expect(fake.clients[0]!.beginCalls).toBe(1);
        value(await pools.close(token, 1000n));
      });
    } finally { env.clear(); restoreSQL(); }
  });
  test("fixture boundaries never launch transactions", async () => {
    const fake = installFake();
    try {
      env.set("CAN_TEST_POSTGRES", "postgres://fake/x");
      const context = fixtureContext();
      await owned(async () => {
        const token = value(await pools.open("CAN_TEST_POSTGRES", 5n));
        const decide = async () => success(record(COMMIT, [["value", 1n]]));
        await expect(transactions.withTransaction(token, decide, leaves, context)).rejects.toThrow();
        await expect(transactions.queryOne(txGet, getPlan, token, record("app::id_parameters", [["id", 4n]]), context)).rejects.toThrow();
        expect(fake.clients[0]!.beginCalls).toBe(0);
        value(await pools.close(token, 1000n));
      });
    } finally { env.clear(); restoreSQL(); }
  });
});
