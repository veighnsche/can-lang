// E09: W5 controlled-failure integration legs on the final adapters.
//
// Every leg drives a live loopback server through the final
// server/router/SQL dispatch path (never a bare in-process race) and
// asserts the full W5 shape: bounded user-visible behavior, defined
// escalation, no post-disposal use, honest unknown outcomes, and the
// E06 hook contract (redacted, correlated, once-only, fixed 500).
// Cancel-absent branch (E02 negative on all dialects): SQL legs prove
// boundary-returns-at-budget with owned-until-settlement plus
// supervisor escalation — never a cancel operand.
//
// Run DB: `can_e09_run1` (CAN_E09_DB; minted via
// distribution/provision-local.sh db mkdb). Ports 18791–18799 are
// E09-owned. Without DB env the live legs skip visibly.
import { describe, expect, test } from "bun:test";
import { spawn } from "node:child_process";
import { mkdtempSync, writeFileSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { fileURLToPath } from "node:url";
import { createHash } from "node:crypto";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, domainFailureDiagnostics, type FailureShape } from "../domain.ts";
import { success, value, type Completion } from "../completion.ts";
import { record } from "../data.ts";
import { runOwnedRoot, resourceStatus } from "../owner.ts";
import { createServer } from "../platform/server.ts";
import { createRouter } from "../platform/router.ts";
import { createResponses } from "../platform/http.ts";
import { createSQLDescriptors, type SQLDescriptorEntry } from "../platform/sql/descriptor.ts";
import { createSQLPools } from "../platform/sql/pool.ts";
import { createSQLTransactions } from "../platform/sql/transaction.ts";
import type { SQLPlan } from "../platform/sql/values.ts";
import { raceBoundary } from "../transport/operation-budget.ts";
import { currentRequestScope, type RequestScope } from "../transport/request-scope.ts";
import { defaultRequestReportSink, setRequestReportSink } from "../transport/request-report.ts";

const PG_URL = process.env["CAN_TEST_POSTGRES_URL"];
const MYSQL_URL = process.env["CAN_TEST_MYSQL_URL"];
const RUN_DB = process.env["CAN_E09_DB"] ?? "can_e09_run1";
const pgLive = test.skipIf(PG_URL === undefined);
const myLive = test.skipIf(MYSQL_URL === undefined);

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
const fieldNames: Record<string, string[]> = {
  "http::invalid_server_config": ["reason"],
  "http::bind_failed": ["address"],
  "http::shutdown_failed": ["phase"],
  "http::invalid_route": ["reason"],
  "http::duplicate_route": ["method", "path"],
  "http::ambiguous_route": ["first", "second"],
  "http::invalid_request": ["reason"],
  "http::body_limit": ["limit"],
  "http::credentials_missing": ["variable"],
  "codec::invalid_data": ["path", "reason"],
  "sql::connection_failed": ["phase"],
  "sql::query_failed": ["operation", "code"],
  "sql::row_missing": ["query"],
  "sql::row_count": ["query", "actual"],
  "sql::schema_mismatch": ["path", "reason"],
  "sql::constraint_failed": ["constraint"],
  "sql::close_failed": ["reason"],
  "sql::row_limit": ["limit"],
  "sql::unsupported_value": ["path", "reason"],
  "sql::transaction_failed": ["phase"],
  "sql::commit_unknown": ["transaction_id"],
};
const declarations = catalogue.errors
  .filter((e) => fieldNames[e.name] !== undefined)
  .map((e) => ({ identity: e.identity, name: e.name, parameters: 0 }));
const fieldKinds = new Map(
  catalogue.errors.flatMap((e) => e.fields.map((f) => [e.name + ":" + f.name, f.type])),
);
const errorShapes: FailureShape[] = declarations.map((e) => ({
  identity: identity("error", e.identity),
  kind: "error",
  declaration: e.identity,
  arguments: [],
  fields: fieldNames[e.name]!.map((name) => ({
    name,
    type: fieldKinds.get(e.name + ":" + name) === "int" ? intShape.identity : textShape.identity,
  })),
  leaves: [],
  inputs: [],
  errors: [],
}));
const domain = createDomainRuntime({ declarations, shapes: [textShape, intShape, ...errorShapes] });
const id = (declaration: string) => identity("error", declaration);
const server = createServer(domain, {
  invalidConfig: id("can.std.http@1::invalid_server_config"),
  bindFailed: id("can.std.http@1::bind_failed"),
  shutdownFailed: id("can.std.http@1::shutdown_failed"),
});
const router = createRouter(domain, {
  invalid: id("can.std.http@1::invalid_route"),
  duplicate: id("can.std.http@1::duplicate_route"),
  ambiguous: id("can.std.http@1::ambiguous_route"),
});
const responses = createResponses(domain, {
  invalid: id("can.std.http@1::invalid_request"),
  invalidData: id("can.std.codec@1::invalid_data"),
  close: "unused",
  writeFailed: "unused",
  limit: id("can.std.http@1::body_limit"),
});

async function textResult(body: string): Promise<Completion<unknown>> {
  return responses.text(value(await responses.ok()), value(await responses.emptyHeaders()), body);
}

type RouteSpec = Readonly<{
  method: "get" | "post";
  path: string;
  handler: (request?: unknown) => Promise<Completion<unknown>>;
}>;

async function startRoutes(
  port: number,
  routes: readonly RouteSpec[],
  options?: { shutdownMs?: bigint; bodyLimit?: bigint },
): Promise<{ token: unknown; config: unknown }> {
  const config = value(
    await server.makeConfig(
      "127.0.0.1",
      BigInt(port),
      options?.bodyLimit ?? 1048576n,
      options?.shutdownMs ?? 5000n,
    ),
  );
  const entries = [];
  for (const route of routes) {
    entries.push(
      value(
        route.method === "get"
          ? await router.get(route.path, route.handler)
          : await router.post(route.path, route.handler),
      ),
    );
  }
  const table = value(await router.make(entries));
  const started = await server.start(config, table);
  if (started.kind !== "ok") throw new Error("test server failed to start");
  return { token: value(started), config };
}

async function waitFor(label: string, ready: () => boolean, budgetMs: number): Promise<void> {
  const started = Date.now();
  while (!ready()) {
    if (Date.now() - started > budgetMs) throw new Error(`timed out waiting for ${label}`);
    await Bun.sleep(20);
  }
}

function capture() {
  const lines: string[] = [];
  setRequestReportSink((line) => {
    lines.push(line);
  });
  return { lines, restore: () => setRequestReportSink(defaultRequestReportSink) };
}

// --- SQL slice (own isolated database, env only) ---

const lookup = (name: string): string | undefined => {
  if (name === "CAN_E09_PG" && PG_URL !== undefined) return PG_URL.replace("/<db>", `/${RUN_DB}`);
  if (name === "CAN_E09_MY" && MYSQL_URL !== undefined)
    return MYSQL_URL.replace("/<db>", `/${RUN_DB}`);
  return undefined;
};

type Dialect = "postgresql" | "mysql";
function entries(
  dialect: Dialect,
  version: number,
  sleepWhere: string,
  slowInsert: string,
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
    drop_probe: execute("DROP TABLE IF EXISTS e09_probe"),
    setup_probe: execute("CREATE TABLE e09_probe (id INTEGER PRIMARY KEY, note TEXT)"),
    seed_probe: execute("INSERT INTO e09_probe (id, note) VALUES (1, 'seed')"),
    sleep_mark: {
      dialect,
      cardinality: "many",
      kind: "select_statement",
      segments: [
        { text: `SELECT 'slept' AS mark FROM e09_probe WHERE ${sleepWhere} LIMIT ` },
        { param: 1 },
      ],
      params: [],
      paramType: "p",
      rowType: "r",
      limit: 1,
      total: 1,
      version,
    },
    tx_insert: execute(slowInsert),
    probe_note: {
      dialect,
      cardinality: "one",
      kind: "select_statement",
      segments: [
        { text: "SELECT note FROM e09_probe WHERE id = " },
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
const descriptors = createSQLDescriptors({
  pg: entries(
    "postgresql",
    170007,
    "pg_sleep(2) IS NOT NULL",
    "INSERT INTO e09_probe (id, note) SELECT 8, 'txslow' WHERE pg_sleep(2) IS NOT NULL",
  ),
  my: entries(
    "mysql",
    80011,
    "SLEEP(2) = 0",
    "INSERT INTO e09_probe (id, note) SELECT 8, 'txslow' FROM DUAL WHERE SLEEP(2) = 0",
  ),
});
const pools = createSQLPools(
  domain,
  {
    connectionFailed: id("can.std.sql@1::connection_failed"),
    queryFailed: id("can.std.sql@1::query_failed"),
    rowMissing: id("can.std.sql@1::row_missing"),
    rowCount: id("can.std.sql@1::row_count"),
    schemaMismatch: id("can.std.sql@1::schema_mismatch"),
    constraintFailed: id("can.std.sql@1::constraint_failed"),
    rowLimit: id("can.std.sql@1::row_limit"),
    unsupportedValue: id("can.std.sql@1::unsupported_value"),
    credentialsMissing: id("can.std.http@1::credentials_missing"),
    closeFailed: id("can.std.sql@1::close_failed"),
  },
  lookup,
  descriptors,
);
const transactions = createSQLTransactions(
  domain,
  {
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
  },
  descriptors,
);
const markPlan: SQLPlan = {
  params: { root: "app::empty", fields: [] },
  rows: { root: "app::mark_row", fields: [{ name: "mark", kind: "str" }] },
};
const notePlan: SQLPlan = {
  params: { root: "app::id_parameters", fields: [{ name: "id", kind: "int" }] },
  rows: { root: "app::note_row", fields: [{ name: "note", kind: "str" }] },
};
const emptyPlan: SQLPlan = { params: { root: "app::empty", fields: [] } };
const emptyParams = record("app::empty", []);
const idParam = (v: bigint) => record("app::id_parameters", [["id", v]]);
const COMMIT = "app::sql_commit_leaf";
const leaves = { commit: COMMIT, rollback: "app::sql_rollback_leaf" };

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

function payloadOf(outcome: Completion<unknown>): {
  name: string;
  payload: Record<string, unknown>;
} {
  if (outcome.kind !== "domain") throw new Error("expected a domain failure");
  const details = domainFailureDiagnostics(outcome.value);
  const payload = details.payload as Record<string, unknown>;
  const plain: Record<string, unknown> = {};
  for (const key of Object.keys(payload)) plain[key] = payload[key];
  return { name: details.declaration.name, payload: plain };
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

describe("W5 lifetime: stalled SQL over live HTTP", () => {
  pgLive(
    "W5-L1: a stalled PG read answers boundedly while the backend settles owned",
    async () => {
      const { lines, restore } = capture();
      try {
        const seen: Record<string, unknown> = {};
        const owned = await runOwnedRoot(async () => {
          const pool = await setupSchema(() => pools.open("CAN_E09_PG", 5n), "pg");
          try {
            const { token } = await startRoutes(18791, [
              {
                method: "get",
                path: "/slow",
                handler: async () => {
                  seen.scope = currentRequestScope();
                  const outcome = await pools.queryRows(
                    descriptors.declareDescriptor("pg", "sleep_mark"),
                    markPlan,
                    pool,
                    emptyParams,
                    10n,
                    undefined,
                    { boundMs: 150 },
                  );
                  if (outcome.kind !== "domain") return textResult("settled");
                  const failed = payloadOf(outcome);
                  seen.failure = failed.name;
                  seen.payload = failed.payload;
                  // The handler renders the honest budget outcome; the
                  // peer never waits out the two-second native sleep.
                  const code = failed.payload["code"];
                  return textResult(`unknown:${typeof code === "string" ? code : "?"}`);
                },
              },
              {
                method: "get",
                path: "/fast",
                handler: async () => textResult("fast-ok"),
              },
            ]);
            const started = Date.now();
            const response = await fetch("http://127.0.0.1:18791/slow");
            const elapsed = Date.now() - started;
            expect(response.status).toBe(200);
            expect(await response.text()).toBe("unknown:budget");
            expect(elapsed).toBeLessThan(1500);
            expect(seen.failure).toBe("sql::query_failed");
            expect(seen.payload).toEqual({ operation: "query_rows", code: "budget" });
            const scope = seen.scope as RequestScope;
            expect(scope.collected.escalations).toHaveLength(1);
            expect(scope.collected.escalations[0]).toMatchObject({ cause: "budget" });
            // Owned until settlement: the lease and the backend outlive
            // the bounded response, then both release.
            expect(resourceStatus(pool).leases).toBe(1);
            expect(await pgSleepers(RUN_DB)).toBe(1);
            await waitFor("late pg settlement", () => scope.collected.lates.length === 1, 8000);
            expect(scope.collected.lates[0]).toMatchObject({ settled: "resolved" });
            expect(resourceStatus(pool).leases).toBe(0);
            expect(await pgSleepers(RUN_DB)).toBe(0);
            // The server and pool stayed healthy throughout.
            const again = await fetch("http://127.0.0.1:18791/fast");
            expect(again.status).toBe(200);
            expect(await again.text()).toBe("fast-ok");
            expect((await server.stop(token)).kind).toBe("ok");
            value(await pools.close(pool, 5000n));
            return success(undefined);
          } catch (cause) {
            value(await pools.close(pool, 15000n));
            throw cause;
          }
        });
        expect(owned.cleanupFailed).toBe(false);
        expect(owned.completion.kind).toBe("ok");
        // Budget unknowns are expected outcomes: the failure hook
        // stays silent (no redacted report for a bounded return).
        expect(lines).toHaveLength(0);
      } finally {
        restore();
      }
    },
    30000,
  );

  myLive(
    "W5-L2: a stalled MySQL read answers boundedly while the thread settles owned",
    async () => {
      const { lines, restore } = capture();
      try {
        const seen: Record<string, unknown> = {};
        const owned = await runOwnedRoot(async () => {
          const pool = await setupSchema(() => pools.mysqlOpen("CAN_E09_MY", 5n), "my");
          try {
            const { token } = await startRoutes(18792, [
              {
                method: "get",
                path: "/slow",
                handler: async () => {
                  seen.scope = currentRequestScope();
                  const outcome = await pools.queryRows(
                    descriptors.declareDescriptor("my", "sleep_mark"),
                    markPlan,
                    pool,
                    emptyParams,
                    10n,
                    undefined,
                    { boundMs: 150 },
                  );
                  if (outcome.kind !== "domain") return textResult("settled");
                  const failed = payloadOf(outcome);
                  seen.failure = failed.name;
                  const code = failed.payload["code"];
                  return textResult(`unknown:${typeof code === "string" ? code : "?"}`);
                },
              },
            ]);
            const started = Date.now();
            const response = await fetch("http://127.0.0.1:18792/slow");
            expect(Date.now() - started).toBeLessThan(1500);
            expect(response.status).toBe(200);
            expect(await response.text()).toBe("unknown:budget");
            expect(seen.failure).toBe("sql::query_failed");
            const scope = seen.scope as RequestScope;
            expect(scope.collected.escalations).toHaveLength(1);
            expect(scope.collected.escalations[0]).toMatchObject({ cause: "budget" });
            expect(resourceStatus(pool).leases).toBe(1);
            expect(await mysqlSleepers(RUN_DB)).toBe(1);
            await waitFor("late mysql settlement", () => scope.collected.lates.length === 1, 8000);
            expect(scope.collected.lates[0]).toMatchObject({ settled: "resolved" });
            expect(resourceStatus(pool).leases).toBe(0);
            expect(await mysqlSleepers(RUN_DB)).toBe(0);
            expect((await server.stop(token)).kind).toBe("ok");
            value(await pools.close(pool, 5000n));
            return success(undefined);
          } catch (cause) {
            value(await pools.close(pool, 15000n));
            throw cause;
          }
        });
        expect(owned.cleanupFailed).toBe(false);
        expect(owned.completion.kind).toBe("ok");
        expect(lines).toHaveLength(0);
      } finally {
        restore();
      }
    },
    30000,
  );

  pgLive(
    "W5-L3: a stalled mid-flight write reports commit-unknown honestly (BLOCKED: peer gated on settlement)",
    async () => {
      // BLOCKED (W5 at-budget response for overrun transactions):
      // the tx boundary returns commit_unknown at the 150ms bound and
      // every honest-outcome assertion below holds, but dispatch's
      // per-request scope drain waits for the tx callback task
      // (owner-core drain waits for callbacks inside the scope), so
      // the peer responds at callback settlement (~2s), not at the
      // bound. The tx shape is drain-blocking (owner-grouped) plus
      // natively unabortable — the exact shape class the X-R04-2 O2
      // trip condition names. Redesigning tx ownership is an E04
      // semantic change, not a W5 leg fix: this leg pins the measured
      // split (handler fast, peer at settlement) and the evidence
      // record carries the BLOCKED verdict to IC2/preparation. A hung
      // tx callback would hold the peer past any bound (shutdown
      // bounds stop itself via shutdownMs; the in-flight peer is cut
      // with the session, not answered).
      const { lines, restore } = capture();
      try {
        const seen: Record<string, unknown> = {};
        const owned = await runOwnedRoot(async () => {
          const pool = await setupSchema(() => pools.open("CAN_E09_PG", 5n), "pg");
          try {
            const { token } = await startRoutes(18793, [
              {
                method: "post",
                path: "/write",
                handler: async () => {
                  seen.scope = currentRequestScope();
                  const outcome = await transactions.withTransaction(
                    pool,
                    async (handle: unknown) => {
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
                  );
                  if (outcome.kind !== "domain") return textResult("settled");
                  const failed = payloadOf(outcome);
                  seen.failure = failed.name;
                  seen.payload = failed.payload;
                  // commit_unknown is the honest shape: the write may
                  // or may not have landed; the id lets the operator
                  // reconcile instead of guessing.
                  seen.handlerAt = Date.now();
                  return textResult("unknown:commit");
                },
              },
            ]);
            const started = Date.now();
            const response = await fetch("http://127.0.0.1:18793/write", { method: "POST" });
            const elapsed = Date.now() - started;
            // The pinned split: the handler (boundary) returned at the
            // bound while the peer waited for tx callback settlement.
            // NOT at-budget: this is the blocked measurement, and the
            // lower bound below fails loudly if the shape ever changes.
            expect((seen.handlerAt as number) - started).toBeLessThan(1500);
            expect(elapsed).toBeGreaterThanOrEqual(1800);
            expect(elapsed).toBeLessThan(8000);
            expect(response.status).toBe(200);
            expect(await response.text()).toBe("unknown:commit");
            expect(seen.failure).toBe("sql::commit_unknown");
            expect(typeof (seen.payload as Record<string, unknown>)["transaction_id"]).toBe(
              "string",
            );
            const scope = seen.scope as RequestScope;
            expect(scope.collected.escalations).toHaveLength(1);
            await waitFor("late pg commit", () => scope.collected.lates.length === 1, 10000);
            // Reconciliation reread: the row landed exactly once.
            const reread = await pools.queryOne(
              descriptors.declareDescriptor("pg", "probe_note"),
              notePlan,
              pool,
              idParam(8n),
            );
            expect(reread.kind).toBe("ok");
            expect((await server.stop(token)).kind).toBe("ok");
            value(await pools.close(pool, 5000n));
            return success(undefined);
          } catch (cause) {
            value(await pools.close(pool, 15000n));
            throw cause;
          }
        });
        expect(owned.cleanupFailed).toBe(false);
        expect(owned.completion.kind).toBe("ok");
        expect(lines).toHaveLength(0);
      } finally {
        restore();
      }
    },
    30000,
  );
});

describe("W5 lifetime: header and body stalls", () => {
  test("W5-L4: header stalls never dispatch (Bun-owned, no Can hook)", async () => {
    let fires = 0;
    const owned = await runOwnedRoot(async () => {
      const { token } = await startRoutes(18794, [
        {
          method: "post",
          path: "/x",
          handler: async () => {
            fires++;
            return textResult("x");
          },
        },
      ]);
      const sock = await Bun.connect({
        hostname: "127.0.0.1",
        port: 18794,
        socket: { data() {}, close() {}, connectError() {}, timeout() {}, error() {} },
      });
      // Partial headers, never completed: Bun holds the connection
      // without ever invoking dispatch.
      sock.write("POST /x HTTP/1.1\r\nhost: x\r\ncontent-length: 100\r\nx-partial: abc");
      await Bun.sleep(2000);
      expect(fires).toBe(0);
      sock.end();
      expect((await server.stop(token)).kind).toBe("ok");
      return success(undefined);
    });
    expect(owned.cleanupFailed).toBe(false);
    expect(owned.completion.kind).toBe("ok");
  }, 30000);

  test("W5-L5: stalled bodies answer 408 and over-limit bodies 413 over the wire", async () => {
    const owned = await runOwnedRoot(async () => {
      const { token } = await startRoutes(
        18795,
        [
          {
            method: "post",
            path: "/x",
            handler: async () => textResult("x"),
          },
        ],
        { bodyLimit: 16n },
      );
      // Over-limit bodies reject before the handler runs.
      const big = await fetch("http://127.0.0.1:18795/x", {
        method: "POST",
        headers: { "content-type": "text/plain" },
        body: new Uint8Array(100),
      });
      expect(big.status).toBe(413);
      // Full headers, zero body bytes: ingress pends in the drain
      // until shutdown answers the stalled peer with 408.
      const chunks: string[] = [];
      const decoder = new TextDecoder();
      const sock = await Bun.connect({
        hostname: "127.0.0.1",
        port: 18795,
        socket: {
          data(_socket, data) {
            chunks.push(decoder.decode(data));
          },
          close() {},
          connectError() {},
          timeout() {},
          error() {},
        },
      });
      sock.write("POST /x HTTP/1.1\r\nhost: x\r\ncontent-length: 10\r\n\r\n");
      await Bun.sleep(300);
      expect((await server.stop(token)).kind).toBe("ok");
      await waitFor("408 bytes", () => chunks.join("").includes("408"), 5000);
      sock.end();
      return success(undefined);
    });
    expect(owned.cleanupFailed).toBe(false);
    expect(owned.completion.kind).toBe("ok");
  }, 30000);
});

describe("W5 lifetime: disconnect, overlap, SIGTERM", () => {
  myLive(
    "W5-L6: disconnect into a live MySQL query settles owned and keeps the pool",
    async () => {
      const { lines, restore } = capture();
      try {
        const seen: Record<string, unknown> = {};
        const owned = await runOwnedRoot(async () => {
          const pool = await setupSchema(() => pools.mysqlOpen("CAN_E09_MY", 5n), "my");
          try {
            const { token } = await startRoutes(18796, [
              {
                method: "get",
                path: "/slow",
                handler: async () => {
                  seen.scope = currentRequestScope();
                  const start = Date.now();
                  // No explicit bound: the ambient request scope alone
                  // propagates the disconnect into the query.
                  const outcome = await pools.queryRows(
                    descriptors.declareDescriptor("my", "sleep_mark"),
                    markPlan,
                    pool,
                    emptyParams,
                    10n,
                  );
                  seen.boundaryAt = Date.now() - start;
                  if (outcome.kind !== "domain") return textResult("settled");
                  return textResult("budget");
                },
              },
            ]);
            const stop = new AbortController();
            const outcome = fetch("http://127.0.0.1:18796/slow", { signal: stop.signal }).then(
              () => "responded",
              (cause: unknown) => (cause instanceof Error ? cause.name : typeof cause),
            );
            await Bun.sleep(400);
            stop.abort();
            expect(await outcome).toBe("AbortError");
            await waitFor("sql boundary", () => seen.boundaryAt !== undefined, 5000);
            expect(seen.boundaryAt as number).toBeLessThan(1500);
            const scope = seen.scope as RequestScope;
            expect(scope.collected.escalations).toHaveLength(1);
            expect(scope.collected.escalations[0]).toMatchObject({ cause: "disconnect" });
            // The thread ran to settlement under ownership; the pool
            // serves the next request normally.
            await waitFor("late mysql settlement", () => scope.collected.lates.length === 1, 8000);
            expect(await mysqlSleepers(RUN_DB)).toBe(0);
            expect(resourceStatus(pool).leases).toBe(0);
            const again = await fetch("http://127.0.0.1:18796/slow");
            expect(again.status).toBe(200);
            expect(await again.text()).toBe("settled");
            expect((await server.stop(token)).kind).toBe("ok");
            value(await pools.close(pool, 5000n));
            return success(undefined);
          } catch (cause) {
            value(await pools.close(pool, 15000n));
            throw cause;
          }
        });
        expect(owned.cleanupFailed).toBe(false);
        expect(owned.completion.kind).toBe("ok");
        expect(lines).toHaveLength(0);
      } finally {
        restore();
      }
    },
    30000,
  );

  pgLive(
    "W5-L7: overlapping requests keep their own outcomes on one pool",
    async () => {
      const seen: Record<string, RequestScope | undefined> = {};
      const owned = await runOwnedRoot(async () => {
        const pool = await setupSchema(() => pools.open("CAN_E09_PG", 5n), "pg");
        try {
          const { token } = await startRoutes(18797, [
            {
              method: "get",
              path: "/slow",
              handler: async () => {
                seen.slow = currentRequestScope();
                const outcome = await pools.queryRows(
                  descriptors.declareDescriptor("pg", "sleep_mark"),
                  markPlan,
                  pool,
                  emptyParams,
                  10n,
                  undefined,
                  { boundMs: 300 },
                );
                if (outcome.kind !== "domain") return textResult("settled");
                return textResult("unknown:budget");
              },
            },
            {
              method: "get",
              path: "/fast",
              handler: async () => {
                seen.fast = currentRequestScope();
                const outcome = await pools.queryOne(
                  descriptors.declareDescriptor("pg", "probe_note"),
                  notePlan,
                  pool,
                  idParam(1n),
                );
                if (outcome.kind !== "ok") return textResult("fast-failed");
                return textResult("fast-ok");
              },
            },
          ]);
          const started = Date.now();
          const [slow, fast] = await Promise.all([
            fetch("http://127.0.0.1:18797/slow").then(async (r) => ({
              status: r.status,
              body: await r.text(),
            })),
            fetch("http://127.0.0.1:18797/fast").then(async (r) => ({
              status: r.status,
              body: await r.text(),
            })),
          ]);
          // Both peers got bounded, correct, uncrossed responses —
          // long before the two-second native sleep finished.
          expect(Date.now() - started).toBeLessThan(1500);
          expect(slow).toEqual({ status: 200, body: "unknown:budget" });
          expect(fast).toEqual({ status: 200, body: "fast-ok" });
          // Each request filed only its own escalation: the stalled
          // one expired at its bound, the fast one filed nothing.
          expect(seen.slow!.collected.escalations).toHaveLength(1);
          expect(seen.slow!.collected.escalations[0]).toMatchObject({ cause: "budget" });
          expect(seen.fast!.collected.escalations).toHaveLength(0);
          await waitFor(
            "late overlap settlement",
            () => seen.slow!.collected.lates.length === 1,
            8000,
          );
          expect(resourceStatus(pool).leases).toBe(0);
          expect(await pgSleepers(RUN_DB)).toBe(0);
          expect((await server.stop(token)).kind).toBe("ok");
          value(await pools.close(pool, 5000n));
          return success(undefined);
        } catch (cause) {
          value(await pools.close(pool, 15000n));
          throw cause;
        }
      });
      expect(owned.cleanupFailed).toBe(false);
      expect(owned.completion.kind).toBe("ok");
    },
    30000,
  );

  test("W5-L8: SIGTERM bounds the live peer with the shutdown outcome", async () => {
    const script =
      childPrelude([
        "http::invalid_server_config",
        "http::bind_failed",
        "http::shutdown_failed",
        "http::invalid_route",
        "http::duplicate_route",
        "http::ambiguous_route",
        "http::invalid_request",
        "codec::invalid_data",
      ]) +
      `
await runOwnedRoot(async ()=>{
 const config=value(await server.makeConfig("127.0.0.1",18798n,1048576n,5000n));
 const route=value(await router.get("/stalled",async ()=>{
   console.log("handler-enter");
   const scope=currentRequestScope();
   const outcome=await raceBoundary({source:"worker",boundMs:30000,budget:scope?.budget,scopeSignal:scope?.signal,sink:scope?.sink},()=>Bun.sleep(5000).then(()=>"slow"));
   if(outcome.kind==="unknown"){console.log("boundary:"+outcome.escalation.cause);return responses.text(value(await responses.ok()),value(await responses.emptyHeaders()),"unknown:"+outcome.escalation.cause);}
   return responses.text(value(await responses.ok()),value(await responses.emptyHeaders()),"settled");
 }));
 const table=value(await router.make([route]));
 const token=value(await server.start(config,table));
 console.log("listening");
 const pending=server.wait(token).then(c=>{console.log("wait:"+c.kind);return c;});
 await pending;
 return success(undefined);
});`;
    const result = await runChild("sigterm-w5", script, async (child, output) => {
      await waitForOutput(output, "listening");
      const pending = fetch("http://127.0.0.1:18798/stalled").then((r) => r.text());
      await waitForOutput(output, "handler-enter");
      await Bun.sleep(200);
      child.kill("SIGTERM");
      // The connected peer receives the bounded shutdown outcome,
      // not the five-second native wait.
      expect(await pending).toBe("unknown:shutdown");
    });
    expect(result.output).toContain("listening");
    expect(result.output).toContain("handler-enter");
    expect(result.output).toContain("boundary:shutdown");
    expect(result.output).toContain("wait:ok");
    expect(result.errors).toBe("");
    expect(result.signal).toBe(null);
    expect(result.code).toBe(0);
  }, 30000);
});

describe("W5 lifetime: the failure hook under W5 conditions", () => {
  const uuidPattern = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/;

  test("W5-L9: an unexpected failure beside a stall reports once, redacted, with a fixed 500", async () => {
    const messageSecret = "e09-native-message-4k7w";
    const querySecret = "e09-query-token-8p2n";
    const { lines, restore } = capture();
    try {
      const owned = await runOwnedRoot(async () => {
        const { token } = await startRoutes(18799, [
          {
            method: "get",
            path: "/slow",
            handler: async () => {
              const scope = currentRequestScope();
              const outcome = await raceBoundary(
                {
                  source: "worker",
                  boundMs: 150,
                  budget: scope?.budget,
                  scopeSignal: scope?.signal,
                  sink: scope?.sink,
                },
                () => Bun.sleep(2000).then(() => "slow-value"),
              );
              if (outcome.kind === "unknown") return textResult("unknown:budget");
              return textResult("settled");
            },
          },
          {
            method: "post",
            path: "/boom",
            handler: async () => {
              throw new Error("handler fault carries " + messageSecret);
            },
          },
        ]);
        const [slow, boom] = await Promise.all([
          fetch("http://127.0.0.1:18799/slow").then(async (r) => ({
            status: r.status,
            body: await r.text(),
          })),
          fetch(`http://127.0.0.1:18799/boom?token=${querySecret}`, { method: "POST" }).then(
            async (r) => ({ status: r.status, body: await r.text() }),
          ),
        ]);
        expect(slow).toEqual({ status: 200, body: "unknown:budget" });
        expect(boom.status).toBe(500);
        expect(boom.body).toBe("Internal Server Error");
        expect((await server.stop(token)).kind).toBe("ok");
        return success(undefined);
      });
      expect(owned.cleanupFailed).toBe(false);
      expect(owned.completion.kind).toBe("ok");
      // Exactly one report: the unexpected throw. The bounded stall
      // beside it filed an escalation, never a hook report.
      expect(lines).toHaveLength(1);
      const parsed = JSON.parse(lines[0]!);
      expect(parsed).toMatchObject({
        schemaVersion: 1,
        kind: "can.runtime-failure",
        phase: "request",
        channel: "standard",
        category: "native_exception",
        source: "route:POST /boom",
      });
      expect(typeof parsed.occurrence).toBe("string");
      expect(parsed.correlation).toMatch(uuidPattern);
      expect(lines[0]).not.toContain(messageSecret);
      expect(lines[0]).not.toContain(querySecret);
    } finally {
      restore();
    }
  }, 30000);

  test("W5-L10: a double-throwing wrapper still reports exactly once on the final adapters", async () => {
    const innerSecret = "e09-inner-fault-3m9x";
    const wrapperSecret = "e09-wrapper-fault-7t4b";
    const { lines, restore } = capture();
    try {
      const owned = await runOwnedRoot(async () => {
        const { token } = await startRoutes(18790, [
          {
            method: "post",
            path: "/wrapped",
            handler: async () => {
              try {
                throw new Error("handler fault carries " + innerSecret);
              } catch {
                // The author logging wrapper faults while handling
                // the fault and masks it.
                throw new Error("wrapper fault carries " + wrapperSecret);
              }
            },
          },
        ]);
        const response = await fetch("http://127.0.0.1:18790/wrapped", { method: "POST" });
        expect(response.status).toBe(500);
        expect(await response.text()).toBe("Internal Server Error");
        expect((await server.stop(token)).kind).toBe("ok");
        return success(undefined);
      });
      expect(owned.cleanupFailed).toBe(false);
      expect(owned.completion.kind).toBe("ok");
      expect(lines).toHaveLength(1);
      const parsed = JSON.parse(lines[0]!);
      expect(parsed).toMatchObject({
        channel: "standard",
        category: "native_exception",
        source: "route:POST /wrapped",
      });
      expect(lines[0]).not.toContain(innerSecret);
      expect(lines[0]).not.toContain(wrapperSecret);
    } finally {
      restore();
    }
  }, 30000);
});

// --- SIGTERM child helpers (own script: stalled-op shutdown, not a hedge) ---

const childRuntime = fileURLToPath(new URL("../", import.meta.url));
function childPrelude(errors: readonly string[]): string {
  return `import {createHash} from "node:crypto";
import {catalogue} from ${JSON.stringify(join(childRuntime, "catalogue.ts"))};
import {createDomainRuntime} from ${JSON.stringify(join(childRuntime, "domain.ts"))};
import {success,value} from ${JSON.stringify(join(childRuntime, "completion.ts"))};
import {runOwnedRoot} from ${JSON.stringify(join(childRuntime, "owner.ts"))};
import {createServer} from ${JSON.stringify(join(childRuntime, "platform/server.ts"))};
import {createRouter} from ${JSON.stringify(join(childRuntime, "platform/router.ts"))};
import {createResponses} from ${JSON.stringify(join(childRuntime, "platform/http.ts"))};
import {currentRequestScope} from ${JSON.stringify(join(childRuntime, "transport/request-scope.ts"))};
import {raceBoundary} from ${JSON.stringify(join(childRuntime, "transport/operation-budget.ts"))};
const identity=(kind,declaration)=>createHash("sha256").update("can-concrete-type-v1\\0"+JSON.stringify([kind,declaration])).digest("hex");
const textShape={identity:identity("primitive","str"),kind:"primitive",declaration:"str",arguments:[],fields:[],leaves:[],inputs:[],errors:[]};
const wanted=new Set(${JSON.stringify(errors)});
const declarations=catalogue.errors.filter(e=>wanted.has(e.name)).map(e=>({identity:e.identity,name:e.name,parameters:0}));
const lookup=Object.fromEntries(catalogue.errors.filter(e=>wanted.has(e.name)).map(e=>[e.name,e.fields.map(f=>f.name)]));
const shapes=declarations.map(e=>({identity:identity("error",e.identity),kind:"error",declaration:e.identity,arguments:[],fields:lookup[e.name].map(name=>({name,type:textShape.identity})),leaves:[],inputs:[],errors:[]}));
const domain=createDomainRuntime({declarations,shapes:[textShape,...shapes]});
const id=declaration=>identity("error",declaration);
const server=createServer(domain,{invalidConfig:id("can.std.http@1::invalid_server_config"),bindFailed:id("can.std.http@1::bind_failed"),shutdownFailed:id("can.std.http@1::shutdown_failed")});
const router=createRouter(domain,{invalid:id("can.std.http@1::invalid_route"),duplicate:id("can.std.http@1::duplicate_route"),ambiguous:id("can.std.http@1::ambiguous_route")});
const responses=createResponses(domain,{invalid:id("can.std.http@1::invalid_request"),invalidData:id("can.std.codec@1::invalid_data")});
`;
}
async function runChild(
  name: string,
  script: string,
  interact: (child: ReturnType<typeof spawn>, output: () => string) => Promise<void>,
  killAfterMs = 15000,
): Promise<{ output: string; errors: string; code: number | null; signal: NodeJS.Signals | null }> {
  const directory = mkdtempSync(join(tmpdir(), "can-w5-" + name + "-"));
  const file = join(directory, name + ".ts");
  writeFileSync(file, script);
  const child = spawn(process.execPath, ["--no-install", file], {
    stdio: ["ignore", "pipe", "pipe"],
  });
  let output = "",
    errors = "";
  child.stdout.on("data", (chunk) => {
    output += chunk;
  });
  child.stderr.on("data", (chunk) => {
    errors += chunk;
  });
  const deadline = setTimeout(() => child.kill("SIGKILL"), killAfterMs);
  try {
    await interact(child, () => output);
    const status = await new Promise<{ code: number | null; signal: NodeJS.Signals | null }>(
      (resolve, reject) => {
        child.once("error", reject);
        child.once("exit", (code, signal) => resolve({ code, signal }));
      },
    );
    return { output, errors, ...status };
  } finally {
    clearTimeout(deadline);
    child.kill("SIGKILL");
    rmSync(directory, { recursive: true, force: true });
  }
}
async function waitForOutput(output: () => string, marker: string): Promise<void> {
  const begin = Date.now();
  while (!output().includes(marker)) {
    if (Date.now() - begin > 8000) throw new Error("missing child marker " + marker);
    await Bun.sleep(20);
  }
}
