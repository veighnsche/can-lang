// E04: cancel-absent SQL boundary legs.
//
// Live PG/MySQL legs run in the isolated per-run database (CAN_E04_DB,
// default can_e04_run2, minted via distribution/provision-local.sh db
// mkdb); without the URLs they skip visibly. SQLite legs run in memory.
// The URL and database name never appear in assertions or messages.
import { describe, expect, test } from "bun:test";
import { createHash } from "node:crypto";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, domainFailureDiagnostics, type FailureShape } from "../domain.ts";
import { success, value, type Completion } from "../completion.ts";
import { dataArray, dataProperty, record } from "../data.ts";
import { runOwnedRoot, resourceStatus } from "../owner.ts";
import { createSQLDescriptors, type SQLDescriptorEntry } from "../platform/sql/descriptor.ts";
import { createSQLPools } from "../platform/sql/pool.ts";
import { createSQLTransactions } from "../platform/sql/transaction.ts";
import type { SQLPlan } from "../platform/sql/values.ts";
import { createCollectorSink } from "../transport/operation-budget.ts";
import { createRequestBudget } from "../transport/request-budget.ts";
import { createRequestScope, runWithRequestScope } from "../transport/request-scope.ts";

const PG_URL = process.env["CAN_TEST_POSTGRES_URL"];
const MYSQL_URL = process.env["CAN_TEST_MYSQL_URL"];
const RUN_DB = process.env["CAN_E04_DB"] ?? "can_e04_run2";
const pgLive = test.skipIf(PG_URL === undefined);
const myLive = test.skipIf(MYSQL_URL === undefined);

async function owned(body: () => Promise<void>): Promise<void> {
  const result = await runOwnedRoot(async () => {
    await body();
    return success(undefined);
  });
  expect(result.cleanupFailed).toBe(false);
  expect(result.completion.kind).toBe("ok");
}

async function waitFor(label: string, ready: () => boolean, budgetMs: number): Promise<void> {
  const started = Date.now();
  while (!ready()) {
    if (Date.now() - started > budgetMs) throw new Error(`timed out waiting for ${label}`);
    await Bun.sleep(20);
  }
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

const lookup = (name: string): string | undefined => {
  if (name === "CAN_E04_PG" && PG_URL !== undefined) return PG_URL.replace("/<db>", `/${RUN_DB}`);
  if (name === "CAN_E04_MY" && MYSQL_URL !== undefined)
    return MYSQL_URL.replace("/<db>", `/${RUN_DB}`);
  return undefined;
};

type Dialect = "postgresql" | "mysql" | "sqlite";
function entries(
  dialect: Dialect,
  version: number,
  sleepWhere: string,
  slowInsert: (rowId: number, note: string) => string,
): Record<string, SQLDescriptorEntry> {
  const execute = (text: string): SQLDescriptorEntry => ({
    dialect,
    cardinality: "execute",
    kind: "execute_statement",
    segments: [{ text }],
    params: [],
    paramType: "p",
    rowType: "r",
    limit: 0,
    total: 0,
    version,
  });
  return {
    drop_probe: execute("DROP TABLE IF EXISTS e04_probe"),
    setup_probe: execute("CREATE TABLE e04_probe (id INTEGER PRIMARY KEY, note TEXT)"),
    seed_probe: execute("INSERT INTO e04_probe (id, note) VALUES (1, 'seed')"),
    sleep_mark: {
      dialect,
      cardinality: "many",
      kind: "select_statement",
      segments: [
        { text: `SELECT 'slept' AS mark FROM e04_probe WHERE ${sleepWhere} LIMIT ` },
        { param: 1 },
      ],
      params: [],
      paramType: "p",
      rowType: "r",
      limit: 1,
      total: 1,
      version,
    },
    slow_insert: execute(slowInsert(7, "slow")),
    tx_insert: execute(slowInsert(8, "txslow")),
    probe_note: {
      dialect,
      cardinality: "one",
      kind: "select_statement",
      segments: [
        { text: "SELECT note FROM e04_probe WHERE id = " },
        { param: 1 },
        { text: " LIMIT " },
        { param: 2 },
      ],
      params: ["id"],
      paramType: "p",
      rowType: "r",
      limit: 2,
      total: 2,
      version,
    },
    absent_table: {
      dialect,
      cardinality: "one",
      kind: "select_statement",
      segments: [
        { text: "SELECT note FROM e04_absent_xyz WHERE id = " },
        { param: 1 },
        { text: " LIMIT " },
        { param: 2 },
      ],
      params: ["id"],
      paramType: "p",
      rowType: "r",
      limit: 2,
      total: 2,
      version,
    },
  };
}

const table: Record<string, Record<string, SQLDescriptorEntry>> = {
  pg: entries(
    "postgresql",
    170007,
    "pg_sleep(2) IS NOT NULL",
    (rowId, note) =>
      `INSERT INTO e04_probe (id, note) SELECT ${rowId}, '${note}' WHERE pg_sleep(2) IS NOT NULL`,
  ),
  my: entries(
    "mysql",
    80011,
    "SLEEP(2) = 0",
    (rowId, note) =>
      `INSERT INTO e04_probe (id, note) SELECT ${rowId}, '${note}' FROM DUAL WHERE SLEEP(2) = 0`,
  ),
  lite: entries(
    "sqlite",
    15,
    "1 = 1",
    (rowId, note) => `INSERT INTO e04_probe (id, note) VALUES (${rowId}, '${note}')`,
  ),
};
const descriptors = createSQLDescriptors(table);
const coreContracts = {
  connectionFailed: id("can.std.sql@1::connection_failed"),
  queryFailed: id("can.std.sql@1::query_failed"),
  rowMissing: id("can.std.sql@1::row_missing"),
  rowCount: id("can.std.sql@1::row_count"),
  schemaMismatch: id("can.std.sql@1::schema_mismatch"),
  constraintFailed: id("can.std.sql@1::constraint_failed"),
  rowLimit: id("can.std.sql@1::row_limit"),
  unsupportedValue: id("can.std.sql@1::unsupported_value"),
};
const pools = createSQLPools(
  domain,
  {
    ...coreContracts,
    credentialsMissing: id("can.std.http@1::credentials_missing"),
    closeFailed: id("can.std.sql@1::close_failed"),
  },
  lookup,
  descriptors,
);
const transactions = createSQLTransactions(
  domain,
  {
    ...coreContracts,
    transactionFailed: id("can.std.sql@1::transaction_failed"),
    commitUnknown: id("can.std.sql@1::commit_unknown"),
  },
  descriptors,
);

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

const emptyPlan: SQLPlan = { params: { root: "app::empty", fields: [] } };
const markPlan: SQLPlan = {
  params: { root: "app::empty", fields: [] },
  rows: { root: "app::mark_row", fields: [{ name: "mark", kind: "str" }] },
};
const notePlan: SQLPlan = {
  params: { root: "app::id_parameters", fields: [{ name: "id", kind: "int" }] },
  rows: { root: "app::note_row", fields: [{ name: "note", kind: "str" }] },
};
const emptyParams = record("app::empty", []);
const idParam = (v: bigint) => record("app::id_parameters", [["id", v]]);
const COMMIT = "app::sql_commit_leaf";
const ROLLBACK = "app::sql_rollback_leaf";
const leaves = { commit: COMMIT, rollback: ROLLBACK };

async function setupSchema(
  open: () => Promise<Completion<unknown>>,
  owner: string,
): Promise<unknown> {
  const token = value(await open());
  for (const name of ["drop_probe", "setup_probe", "seed_probe"]) {
    const done = await pools.execute(
      descriptors.declareDescriptor(owner, name),
      emptyPlan,
      token,
      emptyParams,
    );
    if (done.kind !== "ok") throw new Error(`setup ${name} failed: ${done.kind}`);
  }
  return token;
}

async function pgSleepers(db: string): Promise<number> {
  const sql = new Bun.SQL(PG_URL!.replace("/<db>", `/${db}`), {
    adapter: "postgres",
    bigint: true,
    max: 1,
  });
  try {
    await sql.connect();
    const rows =
      (await sql`SELECT count(*)::int AS n FROM pg_stat_activity WHERE datname = ${db} AND state = 'active' AND query LIKE '%pg_sleep%' AND pid <> pg_backend_pid()`) as Array<{
        n: number;
      }>;
    return rows[0]?.n ?? -1;
  } finally {
    await sql.close();
  }
}

async function mysqlSleepers(db: string): Promise<number> {
  const sql = new Bun.SQL(MYSQL_URL!.replace("/<db>", `/${db}`), {
    adapter: "mysql",
    bigint: true,
    max: 1,
    tls: true,
  });
  try {
    await sql.connect();
    const rows = (await sql`SHOW PROCESSLIST`) as Array<{ Info?: unknown }>;
    return rows.filter((row) => typeof row.Info === "string" && row.Info.includes("SLEEP")).length;
  } finally {
    await sql.close();
  }
}

describe("sql budgets: sqlite (in-memory)", () => {
  test("an exhausted budget returns without starting the query", async () => {
    await owned(async () => {
      const token = value(await pools.sqliteOpenMemory());
      try {
        for (const name of ["drop_probe", "setup_probe", "seed_probe"]) {
          value(
            await pools.execute(
              descriptors.declareDescriptor("lite", name),
              emptyPlan,
              token,
              emptyParams,
            ),
          );
        }
        let at = 1000;
        const budget = createRequestBudget(10, () => at);
        at += 50;
        // The absent table would fail natively if the query started;
        // the budget outcome proves it never did.
        const outcome = await pools.queryOne(
          descriptors.declareDescriptor("lite", "absent_table"),
          notePlan,
          token,
          idParam(1n),
          undefined,
          { boundMs: 5000, budget },
        );
        expect(domainOutcome(outcome, "sql::query_failed")).toEqual({
          operation: "query_one",
          code: "budget",
        });
        // Control: the same query with a live budget fails natively,
        // proving the budget leg above returned before starting.
        const control = await pools.queryOne(
          descriptors.declareDescriptor("lite", "absent_table"),
          notePlan,
          token,
          idParam(1n),
        );
        const controlPayload = domainOutcome(control, "sql::query_failed");
        expect(controlPayload.operation).toBe("query_one");
        expect(controlPayload.code).not.toBe("budget");
        value(await pools.close(token, 5000n));
      } catch (cause) {
        value(await pools.close(token, 5000n));
        throw cause;
      }
    });
  });

  test("an exhausted budget leaves the write unstarted", async () => {
    await owned(async () => {
      const token = value(await pools.sqliteOpenMemory());
      try {
        for (const name of ["drop_probe", "setup_probe", "seed_probe"]) {
          value(
            await pools.execute(
              descriptors.declareDescriptor("lite", name),
              emptyPlan,
              token,
              emptyParams,
            ),
          );
        }
        let at = 1000;
        const budget = createRequestBudget(10, () => at);
        at += 50;
        const outcome = await pools.execute(
          descriptors.declareDescriptor("lite", "slow_insert"),
          emptyPlan,
          token,
          emptyParams,
          undefined,
          { boundMs: 5000, budget },
        );
        expect(domainOutcome(outcome, "sql::query_failed")).toEqual({
          operation: "execute",
          code: "budget",
        });
        const reread = await pools.queryOne(
          descriptors.declareDescriptor("lite", "probe_note"),
          notePlan,
          token,
          idParam(7n),
        );
        expect(reread.kind).toBe("domain");
        if (reread.kind !== "domain") throw new Error("wrong outcome");
        expect(domainFailureDiagnostics(reread.value).declaration.name).toBe("sql::row_missing");
        value(await pools.close(token, 5000n));
      } catch (cause) {
        value(await pools.close(token, 5000n));
        throw cause;
      }
    });
  });

  test("validation runs before the race, even on an exhausted budget", async () => {
    await owned(async () => {
      const token = value(await pools.sqliteOpenMemory());
      try {
        let at = 1000;
        const budget = createRequestBudget(10, () => at);
        at += 50;
        const mistyped = record("app::id_parameters", [["id", "seven"]]);
        const outcome = await pools.queryOne(
          descriptors.declareDescriptor("lite", "probe_note"),
          notePlan,
          token,
          mistyped,
          undefined,
          { boundMs: 5000, budget },
        );
        expect(domainOutcome(outcome, "sql::unsupported_value")).toEqual({
          path: "/id",
          reason: "type",
        });
        value(await pools.close(token, 5000n));
      } catch (cause) {
        value(await pools.close(token, 5000n));
        throw cause;
      }
    });
  });

  test("a settling query under a live bound is identical to the unbounded path", async () => {
    await owned(async () => {
      const token = value(await pools.sqliteOpenMemory());
      try {
        for (const name of ["drop_probe", "setup_probe", "seed_probe"]) {
          value(
            await pools.execute(
              descriptors.declareDescriptor("lite", name),
              emptyPlan,
              token,
              emptyParams,
            ),
          );
        }
        const bounded = value(
          await pools.queryRows(
            descriptors.declareDescriptor("lite", "sleep_mark"),
            markPlan,
            token,
            emptyParams,
            10n,
            undefined,
            { boundMs: 5000 },
          ),
        );
        const plain = value(
          await pools.queryRows(
            descriptors.declareDescriptor("lite", "sleep_mark"),
            markPlan,
            token,
            emptyParams,
            10n,
          ),
        );
        expect(bounded).toEqual(plain);
        value(await pools.close(token, 5000n));
      } catch (cause) {
        value(await pools.close(token, 5000n));
        throw cause;
      }
    });
  });

  test("an exhausted budget refuses the transaction before any callback runs", async () => {
    await owned(async () => {
      const token = value(await pools.sqliteOpenMemory());
      try {
        let at = 1000;
        const budget = createRequestBudget(10, () => at);
        at += 50;
        let callbackRan = false;
        const outcome = await transactions.withTransaction(
          token,
          async () => {
            callbackRan = true;
            return success(record(COMMIT, [["value", 1n]]));
          },
          leaves,
          undefined,
          { boundMs: 5000, budget },
        );
        expect(callbackRan).toBe(false);
        // Nothing ran, so the phase is reported — not commit uncertainty.
        expect(domainOutcome(outcome, "sql::transaction_failed")).toEqual({ phase: "budget" });
        value(await pools.close(token, 5000n));
      } catch (cause) {
        value(await pools.close(token, 5000n));
        throw cause;
      }
    });
  });
});

describe("sql budgets: postgresql (live)", () => {
  pgLive(
    "an overrun query reports the bound while the backend runs to completion",
    async () => {
      await owned(async () => {
        const token = await setupSchema(() => pools.open("CAN_E04_PG", 5n), "pg");
        try {
          const collected = createCollectorSink();
          const scope = createRequestScope({ sink: collected.sink });
          const started = Date.now();
          const outcome = await runWithRequestScope(scope, () =>
            pools.queryRows(
              descriptors.declareDescriptor("pg", "sleep_mark"),
              markPlan,
              token,
              emptyParams,
              10n,
              undefined,
              { boundMs: 150 },
            ),
          );
          const elapsed = Date.now() - started;
          expect(domainOutcome(outcome, "sql::query_failed")).toEqual({
            operation: "query_rows",
            code: "budget",
          });
          // The visible layer returned at the bound, long before the
          // two-second native sleep finished.
          expect(elapsed).toBeLessThan(1500);
          // The lease stays held past the visible return: no timer
          // revoked it, and the backend is still executing.
          expect(resourceStatus(token).leases).toBe(1);
          expect(await pgSleepers(RUN_DB)).toBe(1);
          // A second query on the same pool runs while the first is
          // still owned: overlapping pool use is unaffected.
          const overlap = await pools.queryOne(
            descriptors.declareDescriptor("pg", "probe_note"),
            notePlan,
            token,
            idParam(1n),
          );
          expect(overlap.kind).toBe("ok");
          // Escalation filed automatically at the boundary return.
          expect(collected.escalations).toHaveLength(1);
          expect(collected.escalations[0]).toMatchObject({ cause: "budget" });
          // Late settlement lands under ownership, then the lease drops.
          await waitFor("late pg settlement", () => collected.lates.length === 1, 8000);
          expect(collected.lates[0]).toMatchObject({ settled: "resolved" });
          expect(resourceStatus(token).leases).toBe(0);
          expect(await pgSleepers(RUN_DB)).toBe(0);
          value(await pools.close(token, 5000n));
        } catch (cause) {
          value(await pools.close(token, 15000n));
          throw cause;
        }
      });
    },
    30000,
  );

  pgLive(
    "an overrun write reconciles by reread, never by assuming rollback",
    async () => {
      await owned(async () => {
        const token = await setupSchema(() => pools.open("CAN_E04_PG", 5n), "pg");
        try {
          const collected = createCollectorSink();
          const scope = createRequestScope({ sink: collected.sink });
          const outcome = await runWithRequestScope(scope, () =>
            pools.execute(
              descriptors.declareDescriptor("pg", "slow_insert"),
              emptyPlan,
              token,
              emptyParams,
              undefined,
              { boundMs: 150 },
            ),
          );
          expect(domainOutcome(outcome, "sql::query_failed")).toEqual({
            operation: "execute",
            code: "budget",
          });
          await waitFor("late pg write settlement", () => collected.lates.length === 1, 8000);
          // The write landed late: the budget code never claimed rollback.
          const reread = value(
            await pools.queryOne(
              descriptors.declareDescriptor("pg", "probe_note"),
              notePlan,
              token,
              idParam(7n),
            ),
          );
          expect(reread).toMatchObject({});
          value(await pools.close(token, 5000n));
        } catch (cause) {
          value(await pools.close(token, 15000n));
          throw cause;
        }
      });
    },
    30000,
  );

  pgLive(
    "an overrun transaction reports commit-unknown and settles owned",
    async () => {
      await owned(async () => {
        const token = await setupSchema(() => pools.open("CAN_E04_PG", 5n), "pg");
        try {
          const collected = createCollectorSink();
          const scope = createRequestScope({ sink: collected.sink });
          const started = Date.now();
          const outcome = await runWithRequestScope(scope, () =>
            transactions.withTransaction(
              token,
              async (handle: unknown) => {
                const slept = await transactions.queryRows(
                  descriptors.declareDescriptor("pg", "sleep_mark"),
                  markPlan,
                  handle,
                  emptyParams,
                  10n,
                );
                if (slept.kind !== "ok") return slept as Completion<never>;
                const inserted = await transactions.execute(
                  descriptors.declareDescriptor("pg", "tx_insert"),
                  emptyPlan,
                  handle,
                  emptyParams,
                );
                if (inserted.kind !== "ok") return inserted as Completion<never>;
                return success(record(COMMIT, [["value", 1n]]));
              },
              leaves,
              undefined,
              { boundMs: 150 },
            ),
          );
          expect(Date.now() - started).toBeLessThan(1500);
          const payload = domainOutcome(outcome, "sql::commit_unknown");
          expect(typeof payload.transaction_id).toBe("string");
          expect((payload.transaction_id as string).length).toBeGreaterThan(0);
          expect(resourceStatus(token).leases).toBe(1);
          expect(collected.escalations).toHaveLength(1);
          await waitFor("late pg commit", () => collected.lates.length === 1, 10000);
          expect(collected.lates[0]).toMatchObject({ settled: "resolved" });
          expect(resourceStatus(token).leases).toBe(0);
          // The commit landed late; reread reconciles the unknown outcome.
          const reread = await pools.queryOne(
            descriptors.declareDescriptor("pg", "probe_note"),
            notePlan,
            token,
            idParam(8n),
          );
          expect(reread.kind).toBe("ok");
          value(await pools.close(token, 5000n));
        } catch (cause) {
          value(await pools.close(token, 15000n));
          throw cause;
        }
      });
    },
    30000,
  );

  pgLive(
    "the ambient request budget bounds calls without explicit bounds",
    async () => {
      await owned(async () => {
        const token = await setupSchema(() => pools.open("CAN_E04_PG", 5n), "pg");
        try {
          const collected = createCollectorSink();
          const scope = createRequestScope({ totalMs: 150, sink: collected.sink });
          const started = Date.now();
          // No explicit bounds: the remaining request budget still caps
          // the visible layer.
          const outcome = await runWithRequestScope(scope, () =>
            pools.queryRows(
              descriptors.declareDescriptor("pg", "sleep_mark"),
              markPlan,
              token,
              emptyParams,
              10n,
            ),
          );
          expect(Date.now() - started).toBeLessThan(1500);
          expect(domainOutcome(outcome, "sql::query_failed")).toEqual({
            operation: "query_rows",
            code: "budget",
          });
          await waitFor("ambient-budget late settlement", () => collected.lates.length === 1, 8000);
          value(await pools.close(token, 5000n));
        } catch (cause) {
          value(await pools.close(token, 15000n));
          throw cause;
        }
      });
    },
    30000,
  );

  pgLive(
    "scope expiry propagates disconnect into a running query",
    async () => {
      await owned(async () => {
        const token = await setupSchema(() => pools.open("CAN_E04_PG", 5n), "pg");
        try {
          const collected = createCollectorSink();
          const scope = createRequestScope({ sink: collected.sink });
          const raced = runWithRequestScope(scope, () =>
            pools.queryRows(
              descriptors.declareDescriptor("pg", "sleep_mark"),
              markPlan,
              token,
              emptyParams,
              10n,
              undefined,
              { boundMs: 30000 },
            ),
          );
          await Bun.sleep(300);
          scope.expire("disconnect");
          const outcome = await raced;
          expect(domainOutcome(outcome, "sql::query_failed")).toEqual({
            operation: "query_rows",
            code: "budget",
          });
          expect(collected.escalations).toHaveLength(1);
          expect(collected.escalations[0]).toMatchObject({ cause: "disconnect" });
          expect(resourceStatus(token).leases).toBe(1);
          await waitFor("disconnect-leg late settlement", () => collected.lates.length === 1, 8000);
          expect(resourceStatus(token).leases).toBe(0);
          value(await pools.close(token, 5000n));
        } catch (cause) {
          value(await pools.close(token, 15000n));
          throw cause;
        }
      });
    },
    30000,
  );

  pgLive(
    "overlapping bounded queries share one pool without interference",
    async () => {
      await owned(async () => {
        const token = await setupSchema(() => pools.open("CAN_E04_PG", 2n), "pg");
        try {
          const collected = createCollectorSink();
          const scope = createRequestScope({ sink: collected.sink });
          const [short, long] = await runWithRequestScope(scope, () =>
            Promise.all([
              pools.queryRows(
                descriptors.declareDescriptor("pg", "sleep_mark"),
                markPlan,
                token,
                emptyParams,
                10n,
                undefined,
                { boundMs: 150 },
              ),
              pools.queryRows(
                descriptors.declareDescriptor("pg", "sleep_mark"),
                markPlan,
                token,
                emptyParams,
                10n,
                undefined,
                { boundMs: 15000 },
              ),
            ]),
          );
          expect(domainOutcome(short, "sql::query_failed")).toEqual({
            operation: "query_rows",
            code: "budget",
          });
          expect(long.kind).toBe("ok");
          // The long call really ran the sleep query, not an empty stub.
          const marks = dataArray(value(long)).map((row) => dataProperty(row, "mark"));
          expect(marks).toEqual(["slept"]);
          // The short call's late settlement still arrives owned.
          await waitFor("overlap late settlement", () => collected.lates.length === 1, 8000);
          expect(collected.lates[0]).toMatchObject({ settled: "resolved" });
          value(await pools.close(token, 5000n));
        } catch (cause) {
          value(await pools.close(token, 15000n));
          throw cause;
        }
      });
    },
    30000,
  );

  pgLive(
    "close during an overrun drains by default",
    async () => {
      await owned(async () => {
        const token = await setupSchema(() => pools.open("CAN_E04_PG", 5n), "pg");
        const collected = createCollectorSink();
        const scope = createRequestScope({ sink: collected.sink });
        const outcome = await runWithRequestScope(scope, () =>
          pools.queryRows(
            descriptors.declareDescriptor("pg", "sleep_mark"),
            markPlan,
            token,
            emptyParams,
            10n,
            undefined,
            { boundMs: 150 },
          ),
        );
        expect(domainOutcome(outcome, "sql::query_failed")).toEqual({
          operation: "query_rows",
          code: "budget",
        });
        // The close waits for the still-owned query instead of revoking
        // it, then completes normally.
        value(await pools.close(token, 15000n));
        expect(collected.lates).toHaveLength(1);
      });
    },
    30000,
  );

  pgLive(
    "a short close deadline during an overrun leaves the close owned",
    async () => {
      // The intentional timeout marks root cleanup even though the close
      // later finishes: assert that contract instead of the helper's.
      const result = await runOwnedRoot(async () => {
        const token = await setupSchema(() => pools.open("CAN_E04_PG", 5n), "pg");
        const collected = createCollectorSink();
        const scope = createRequestScope({ sink: collected.sink });
        const outcome = await runWithRequestScope(scope, () =>
          pools.queryRows(
            descriptors.declareDescriptor("pg", "sleep_mark"),
            markPlan,
            token,
            emptyParams,
            10n,
            undefined,
            { boundMs: 150 },
          ),
        );
        expect(domainOutcome(outcome, "sql::query_failed")).toEqual({
          operation: "query_rows",
          code: "budget",
        });
        const closed = await pools.close(token, 100n);
        expect(domainOutcome(closed, "sql::close_failed")).toEqual({ reason: "timeout" });
        // The close continues owned: the query settles late and the pool
        // reaches closed without any post-disposal use.
        await waitFor(
          "owned close completion",
          () => resourceStatus(token).state === "closed",
          10000,
        );
        expect(collected.lates).toHaveLength(1);
        await expect(
          pools.queryOne(
            descriptors.declareDescriptor("pg", "probe_note"),
            notePlan,
            token,
            idParam(1n),
          ),
        ).rejects.toThrow();
        return success(undefined);
      });
      expect(result.cleanupFailed).toBe(true);
      expect(result.completion.kind).toBe("ok");
    },
    30000,
  );
});

describe("sql budgets: mysql (live)", () => {
  myLive(
    "an overrun query reports the bound while the server thread runs on",
    async () => {
      await owned(async () => {
        const token = await setupSchema(() => pools.mysqlOpen("CAN_E04_MY", 5n), "my");
        try {
          const collected = createCollectorSink();
          const scope = createRequestScope({ sink: collected.sink });
          const started = Date.now();
          const outcome = await runWithRequestScope(scope, () =>
            pools.queryRows(
              descriptors.declareDescriptor("my", "sleep_mark"),
              markPlan,
              token,
              emptyParams,
              10n,
              undefined,
              { boundMs: 150 },
            ),
          );
          expect(Date.now() - started).toBeLessThan(1500);
          expect(domainOutcome(outcome, "sql::query_failed")).toEqual({
            operation: "query_rows",
            code: "budget",
          });
          expect(resourceStatus(token).leases).toBe(1);
          expect(await mysqlSleepers(RUN_DB)).toBe(1);
          expect(collected.escalations).toHaveLength(1);
          await waitFor("late mysql settlement", () => collected.lates.length === 1, 8000);
          expect(collected.lates[0]).toMatchObject({ settled: "resolved" });
          expect(resourceStatus(token).leases).toBe(0);
          expect(await mysqlSleepers(RUN_DB)).toBe(0);
          value(await pools.close(token, 5000n));
        } catch (cause) {
          value(await pools.close(token, 15000n));
          throw cause;
        }
      });
    },
    30000,
  );

  myLive(
    "an overrun transaction reports commit-unknown and settles owned",
    async () => {
      await owned(async () => {
        const token = await setupSchema(() => pools.mysqlOpen("CAN_E04_MY", 5n), "my");
        try {
          const collected = createCollectorSink();
          const scope = createRequestScope({ sink: collected.sink });
          const started = Date.now();
          const outcome = await runWithRequestScope(scope, () =>
            transactions.withTransaction(
              token,
              async (handle: unknown) => {
                const slept = await transactions.queryRows(
                  descriptors.declareDescriptor("my", "sleep_mark"),
                  markPlan,
                  handle,
                  emptyParams,
                  10n,
                );
                if (slept.kind !== "ok") return slept as Completion<never>;
                const inserted = await transactions.execute(
                  descriptors.declareDescriptor("my", "tx_insert"),
                  emptyPlan,
                  handle,
                  emptyParams,
                );
                if (inserted.kind !== "ok") return inserted as Completion<never>;
                return success(record(COMMIT, [["value", 1n]]));
              },
              leaves,
              undefined,
              { boundMs: 150 },
            ),
          );
          expect(Date.now() - started).toBeLessThan(1500);
          const payload = domainOutcome(outcome, "sql::commit_unknown");
          expect(typeof payload.transaction_id).toBe("string");
          await waitFor("late mysql commit", () => collected.lates.length === 1, 10000);
          const reread = await pools.queryOne(
            descriptors.declareDescriptor("my", "probe_note"),
            notePlan,
            token,
            idParam(8n),
          );
          expect(reread.kind).toBe("ok");
          value(await pools.close(token, 5000n));
        } catch (cause) {
          value(await pools.close(token, 15000n));
          throw cause;
        }
      });
    },
    30000,
  );
});
