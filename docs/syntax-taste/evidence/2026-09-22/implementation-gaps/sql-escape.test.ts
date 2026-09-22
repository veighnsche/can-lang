// Setup copied from runtime/test/transaction.test.ts for this bounded offline probe.
import { describe, expect, test } from "bun:test";
import { createHash } from "node:crypto";
import { catalogue } from "../../../../../runtime/catalogue.ts";
import { createDomainRuntime, domainFailureDiagnostics, type FailureShape } from "../../../../../runtime/domain.ts";
import { success, value, type Completion } from "../../../../../runtime/completion.ts";
import { standardFailureDiagnostics, type StandardFailure } from "../../../../../runtime/failure.ts";
import { assertionContext } from "../../../../../runtime/assert/context.ts";
import { record, dataProperty } from "../../../../../runtime/data.ts";
import { runOwnedRoot, registerResource, resourceStatus, registerCallableCaptures } from "../../../../../runtime/owner.ts";
import { createSQLDescriptors, type SQLDescriptorEntry } from "../../../../../runtime/platform/sql-descriptor.ts";
import { createSQLPools, type SQLPlan } from "../../../../../runtime/platform/sql.ts";
import { createSQLTransactions, isSQLTransactionValue } from "../../../../../runtime/platform/transaction.ts";

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
      cardinality: "execute", kind: "InsertStmt",
      segments: [{ text: "INSERT INTO t (display_name) VALUES (" }, { param: 1 }, { text: ")" }],
      params: ["term"], paramType: "p", rowType: "r", limit: 0, total: 1, version: 170007,
    },
    tx_get: {
      cardinality: "one", kind: "SelectStmt",
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


for (const shape of ["direct", "record", "array", "callable"] as const) {
  test(`transaction commit returns ${shape}, but escaped use cannot reach native SQL`, async () => {
    const fake = installFake();
    try {
      env.set("CAN_TEST_POSTGRES", "postgres://fake/x");
      await owned(async () => {
        const pool = value(await pools.open("CAN_TEST_POSTGRES", 5n));
        const outcome = await transactions.withTransaction(pool, async (handle: unknown) => {
          const query = () => transactions.queryOne(txGet, getPlan, handle, record("app::id_parameters", [["id", 4n]]));
          const held = shape === "record" ? record("app::held", [["handle", handle]]) :
            shape === "array" ? Object.freeze([handle]) :
            shape === "callable" ? registerCallableCaptures(query, [handle]) : handle;
          return success(record(COMMIT, [["value", held]]));
        }, leaves);
        expect(outcome.kind).toBe("ok");
        const held = value(outcome);
        let rejected = false;
        try {
          if (shape === "callable") await (held as Function)();
          else {
            const handle = shape === "record" ? dataProperty(held, "handle") : shape === "array" ? (held as unknown[])[0] : held;
            await transactions.queryOne(txGet, getPlan, handle, record("app::id_parameters", [["id", 4n]]));
          }
        } catch (cause) { rejected = true; expect(standardKind(cause)).toBe("resource_state"); }
        expect(rejected).toBe(true);
        expect(fake.clients[0]!.beginCalls).toBe(1);
        expect(fake.clients[0]!.txs[0]!.calls.length).toBe(0);
        value(await pools.close(pool, 1000n));
      });
    } finally { env.clear(); restoreSQL(); }
  });
}
