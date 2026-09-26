// E05 X-R04-2: O1 hedge measurement legs (slice 1: harness contract,
// winner response, failure absorption, all-expired honesty).
//
// Every HTTP leg runs a real Can server frontend over loopback; replicas
// are cancel-absent-shaped bare work (the E04 SQL branch shape: no owner
// group, owned until settlement). Fetch-shaped (cancel-present) replicas
// follow in slice 2 over the stall-injection upstream.
import { describe, expect, test } from "bun:test";
import { createHash } from "node:crypto";
import { spawn } from "node:child_process";
import { mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { fileURLToPath } from "node:url";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, type FailureShape } from "../domain.ts";
import { success, value, type Completion } from "../completion.ts";
import { record } from "../data.ts";
import { resourceStatus, runOwnedRoot, useResource } from "../owner.ts";
import { createServer } from "../platform/server.ts";
import { createRouter } from "../platform/router.ts";
import { createResponses } from "../platform/http.ts";
import { createSQLDescriptors, type SQLDescriptorEntry } from "../platform/sql/descriptor.ts";
import { createSQLPools } from "../platform/sql/pool.ts";
import type { SQLPlan } from "../platform/sql/values.ts";
import { currentRequestScope, type RequestScope } from "../transport/request-scope.ts";
import { performRequest } from "../transport/fetch.ts";
import type { Connection } from "../transport/request.ts";
import { hedge, startStallUpstream } from "./http-hedge-harness.ts";

const PG_URL = process.env["CAN_TEST_POSTGRES_URL"];
const MYSQL_URL = process.env["CAN_TEST_MYSQL_URL"];
const RUN_DB = process.env["CAN_E05_DB"] ?? "can_e05_run1";
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
};
const declarations = catalogue.errors
  .filter((e) => fieldNames[e.name] !== undefined)
  .map((e) => ({ identity: e.identity, name: e.name, parameters: 0 }));
const errorShapes: FailureShape[] = declarations.map((e) => ({
  identity: identity("error", e.identity),
  kind: "error",
  declaration: e.identity,
  arguments: [],
  fields: fieldNames[e.name]!.map((name) => ({
    name,
    type:
      (e.name === "sql::row_count" && name === "actual") ||
      (e.name === "sql::row_limit" && name === "limit")
        ? intShape.identity
        : textShape.identity,
  })),
  leaves: [],
  inputs: [],
  errors: [],
}));
const domain = createDomainRuntime({
  declarations,
  shapes: [textShape, intShape, ...errorShapes],
});
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
  limit: "unused",
});

// SQLite slice for the in-process lease leg (H10); live PG/MySQL legs
// (H11/H12) extend this table in slice 5.
const liteEntries: Record<string, SQLDescriptorEntry> = {
  drop_probe: {
    dialect: "sqlite",
    cardinality: "execute",
    kind: "execute_statement",
    segments: [{ text: "DROP TABLE IF EXISTS e05_probe" }],
    params: [],
    paramType: "p",
    rowType: "r",
    limit: 0,
    total: 0,
    version: 15,
  },
  setup_probe: {
    dialect: "sqlite",
    cardinality: "execute",
    kind: "execute_statement",
    segments: [{ text: "CREATE TABLE e05_probe (id INTEGER PRIMARY KEY, note TEXT)" }],
    params: [],
    paramType: "p",
    rowType: "r",
    limit: 0,
    total: 0,
    version: 15,
  },
  seed_probe: {
    dialect: "sqlite",
    cardinality: "execute",
    kind: "execute_statement",
    segments: [{ text: "INSERT INTO e05_probe (id, note) VALUES (1, 'seed')" }],
    params: [],
    paramType: "p",
    rowType: "r",
    limit: 0,
    total: 0,
    version: 15,
  },
  probe_note: {
    dialect: "sqlite",
    cardinality: "one",
    kind: "select_statement",
    segments: [
      { text: "SELECT note FROM e05_probe WHERE id = " },
      { param: 1 },
      { text: " LIMIT " },
      { param: 2 },
    ],
    params: ["id"],
    paramType: "p",
    rowType: "r",
    limit: 2,
    total: 2,
    version: 15,
  },
};
type Dialect = "postgresql" | "mysql" | "sqlite";
function liveEntries(
  dialect: Dialect,
  version: number,
  sleepWhere: string,
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
    drop_probe: execute("DROP TABLE IF EXISTS e05_probe"),
    setup_probe: execute("CREATE TABLE e05_probe (id INTEGER PRIMARY KEY, note TEXT)"),
    seed_probe: execute("INSERT INTO e05_probe (id, note) VALUES (1, 'seed')"),
    sleep_mark: {
      dialect,
      cardinality: "many",
      kind: "select_statement",
      segments: [
        { text: `SELECT 'slept' AS mark FROM e05_probe WHERE ${sleepWhere} LIMIT ` },
        { param: 1 },
      ],
      params: [],
      paramType: "p",
      rowType: "r",
      limit: 1,
      total: 1,
      version,
    },
    probe_note: {
      dialect,
      cardinality: "one",
      kind: "select_statement",
      segments: [
        { text: "SELECT note FROM e05_probe WHERE id = " },
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
  lite: liteEntries,
  pg: liveEntries("postgresql", 170007, "pg_sleep(2) IS NOT NULL"),
  my: liveEntries("mysql", 80011, "SLEEP(2) = 0"),
});
const liveLookup = (name: string): string | undefined => {
  if (name === "CAN_E05_PG" && PG_URL !== undefined) return PG_URL.replace("/<db>", `/${RUN_DB}`);
  if (name === "CAN_E05_MY" && MYSQL_URL !== undefined)
    return MYSQL_URL.replace("/<db>", `/${RUN_DB}`);
  return undefined;
};
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
  liveLookup,
  descriptors,
);
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

async function setupLiveSchema(
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

async function textResult(body: string): Promise<Completion<unknown>> {
  return responses.text(value(await responses.ok()), value(await responses.emptyHeaders()), body);
}

async function startOn(
  port: number,
  handler: (request?: unknown) => Promise<Completion<unknown>>,
): Promise<{ token: unknown }> {
  const config = value(await server.makeConfig("127.0.0.1", BigInt(port), 1048576n, 5000n));
  const get = value(await router.get("/x", handler));
  const post = value(await router.post("/x", handler));
  const table = value(await router.make([get, post]));
  const started = await server.start(config, table);
  if (started.kind !== "ok") throw new Error("test server failed to start");
  return { token: value(started) };
}

async function waitFor(label: string, ready: () => boolean, budgetMs: number): Promise<void> {
  const started = Date.now();
  while (!ready()) {
    if (Date.now() - started > budgetMs) throw new Error(`timed out waiting for ${label}`);
    await Bun.sleep(20);
  }
}

describe("hedge contract (unit)", () => {
  test("H0a: all-rejected replicas report every error without throwing", async () => {
    const outcome = await hedge<string>({ boundMs: 5000, hedgeDelayMs: 0, abortLosers: false }, [
      {
        source: "worker",
        cancelable: false,
        start: async () => {
          await Bun.sleep(10);
          throw new Error("boom-a");
        },
      },
      {
        source: "worker",
        cancelable: false,
        start: async () => {
          await Bun.sleep(20);
          throw new Error("boom-b");
        },
      },
    ]);
    expect(outcome.kind).toBe("failed");
    if (outcome.kind !== "failed") throw new Error("wrong outcome");
    expect(outcome.errors.map((error) => (error as Error).message).sort()).toEqual([
      "boom-a",
      "boom-b",
    ]);
    const fates = await outcome.settledAll;
    expect(fates).toHaveLength(2);
    expect(fates.every((fate) => fate.fate === "rejected")).toBe(true);
  });

  test("H0b: arity, delay, and bound validation fail closed", async () => {
    const one = {
      source: "worker" as const,
      cancelable: false,
      start: async () => "x",
    };
    await expect(
      hedge({ boundMs: 100, hedgeDelayMs: 0, abortLosers: false }, [one]),
    ).rejects.toThrow(TypeError);
    await expect(
      hedge({ boundMs: 100, hedgeDelayMs: 0, abortLosers: false }, [one, one, one]),
    ).rejects.toThrow(TypeError);
    await expect(
      hedge({ boundMs: 100, hedgeDelayMs: -1, abortLosers: false }, [one, one]),
    ).rejects.toThrow(TypeError);
    await expect(
      hedge({ boundMs: 0, hedgeDelayMs: 0, abortLosers: false }, [one, one]),
    ).rejects.toThrow(TypeError);
  });

  test("H0c: zero delay starts both replicas before the first await", async () => {
    let started = 0;
    const pending = hedge<string>(
      { boundMs: 5000, hedgeDelayMs: 0, abortLosers: false },
      [0, 1].map((replica) => ({
        source: "worker" as const,
        cancelable: false,
        start: async () => {
          started++;
          await Bun.sleep(replica === 0 ? 30 : 5);
          return replica === 0 ? "primary" : "secondary";
        },
      })),
    );
    expect(started).toBe(2);
    const outcome = await pending;
    expect(outcome.kind).toBe("settled");
    if (outcome.kind !== "settled") throw new Error("wrong outcome");
    expect(outcome.winner).toBe(1);
    expect(outcome.value).toBe("secondary");
    expect(await outcome.settledAll).toHaveLength(2);
  });
});

describe("hedge over real HTTP (cancel-absent-shaped replicas)", () => {
  test("H1: the winner responds boundedly while the loser stays owned", async () => {
    const seen: Record<string, unknown> = {};
    const owned = await runOwnedRoot(async () => {
      const { token } = await startOn(18771, async () => {
        const scope = currentRequestScope();
        seen.scope = scope;
        let started = 0;
        const outcome = await hedge<string>(
          {
            boundMs: 30000,
            budget: scope?.budget,
            scopeSignal: scope?.signal,
            sink: scope?.sink,
            hedgeDelayMs: 50,
            abortLosers: false,
          },
          [
            {
              source: "worker",
              cancelable: false,
              start: async () => {
                started++;
                await Bun.sleep(2000);
                return "primary";
              },
            },
            {
              source: "worker",
              cancelable: false,
              start: async () => {
                started++;
                await Bun.sleep(10);
                return "secondary";
              },
            },
          ],
        );
        seen.started = started;
        if (outcome.kind !== "settled") return textResult("unexpected:" + outcome.kind);
        seen.winner = outcome.winner;
        void outcome.settledAll.then((fates) => {
          seen.fatesLate = fates.length;
        });
        return textResult(`winner:${outcome.winner}:${outcome.value}`);
      });
      const start = Date.now();
      const response = await fetch("http://127.0.0.1:18771/x");
      const body = await response.text();
      const elapsed = Date.now() - start;
      // Bounded by the winner (~60ms), not the two-second loser: the
      // request-scope drain does not wait for cancel-absent-shaped
      // (group-free) losers.
      expect(response.status).toBe(200);
      expect(body).toBe("winner:1:secondary");
      expect(elapsed).toBeLessThan(1500);
      expect(seen.started).toBe(2);
      const scope = seen.scope as RequestScope;
      expect(scope.collected.escalations).toHaveLength(0);
      expect(scope.collected.lates).toHaveLength(0);
      await waitFor("loser boundary settlement", () => seen.fatesLate === 2, 8000);
      // Still no escalation: nothing expired, nothing failed.
      expect(scope.collected.escalations).toHaveLength(0);
      const again = await fetch("http://127.0.0.1:18771/x");
      expect(again.status).toBe(200);
      expect((await server.stop(token)).kind).toBe("ok");
      return success(undefined);
    });
    expect(owned.cleanupFailed).toBe(false);
    expect(owned.completion.kind).toBe("ok");
  }, 30000);

  test("H2: the hedge absorbs a fast replica failure", async () => {
    const seen: Record<string, unknown> = {};
    const owned = await runOwnedRoot(async () => {
      const { token } = await startOn(18772, async () => {
        const scope = currentRequestScope();
        seen.scope = scope;
        const outcome = await hedge<string>(
          {
            boundMs: 30000,
            budget: scope?.budget,
            scopeSignal: scope?.signal,
            sink: scope?.sink,
            hedgeDelayMs: 20,
            abortLosers: false,
          },
          [
            {
              source: "worker",
              cancelable: false,
              start: async () => {
                await Bun.sleep(10);
                throw new Error("boom");
              },
            },
            {
              source: "worker",
              cancelable: false,
              start: async () => {
                await Bun.sleep(30);
                return "recovered";
              },
            },
          ],
        );
        if (outcome.kind !== "settled") return textResult("unexpected:" + outcome.kind);
        seen.winner = outcome.winner;
        const fates = await outcome.settledAll;
        seen.rejected = fates.filter((fate) => fate.fate === "rejected").length;
        return textResult(`winner:${outcome.winner}:${outcome.value}`);
      });
      const response = await fetch("http://127.0.0.1:18772/x");
      expect(response.status).toBe(200);
      expect(await response.text()).toBe("winner:1:recovered");
      expect(seen.rejected).toBe(1);
      const scope = seen.scope as RequestScope;
      // The absorbed failure is owner-observed (hedge fates) only: no
      // boundary expired, so the shared sink stays silent.
      expect(scope.collected.escalations).toHaveLength(0);
      expect(scope.collected.lates).toHaveLength(0);
      expect((await server.stop(token)).kind).toBe("ok");
      return success(undefined);
    });
    expect(owned.cleanupFailed).toBe(false);
    expect(owned.completion.kind).toBe("ok");
  }, 30000);

  test("H3: all-expired replicas return the honest unknown outcome", async () => {
    const seen: Record<string, unknown> = {};
    const owned = await runOwnedRoot(async () => {
      const { token } = await startOn(18773, async () => {
        const scope = currentRequestScope();
        seen.scope = scope;
        const outcome = await hedge<string>(
          {
            boundMs: 150,
            budget: scope?.budget,
            scopeSignal: scope?.signal,
            sink: scope?.sink,
            hedgeDelayMs: 20,
            abortLosers: false,
          },
          [0, 1].map((replica) => ({
            source: "worker" as const,
            cancelable: false,
            start: async () => {
              await Bun.sleep(2000);
              return replica === 0 ? "primary" : "secondary";
            },
          })),
        );
        if (outcome.kind !== "unknown") return textResult("unexpected:" + outcome.kind);
        seen.cause = outcome.escalation.cause;
        seen.marker = { ...outcome.marker };
        void outcome.settledAll.then((fates) => {
          seen.fatesLate = fates.length;
        });
        return textResult(`unknown:${outcome.escalation.cause}`);
      });
      const start = Date.now();
      const response = await fetch("http://127.0.0.1:18773/x");
      const body = await response.text();
      const elapsed = Date.now() - start;
      expect(response.status).toBe(200);
      expect(body).toBe("unknown:budget");
      expect(elapsed).toBeLessThan(1500);
      expect(seen.cause).toBe("budget");
      expect(seen.marker).toMatchObject({
        kind: "unknown-write",
        source: "worker",
        commit: "unknown",
        owned: true,
        escalation: "supervisor",
      });
      const scope = seen.scope as RequestScope;
      expect(scope.collected.escalations).toHaveLength(2);
      await waitFor("owned late settlement", () => scope.collected.lates.length === 2, 8000);
      expect(scope.collected.lates[0]).toMatchObject({ settled: "resolved" });
      expect(scope.collected.lates[1]).toMatchObject({ settled: "resolved" });
      // Seq linkage: each late settlement names its escalation.
      const seqs = new Set(scope.collected.escalations.map((record) => record.seq));
      for (const late of scope.collected.lates) expect(seqs.has(late.seq)).toBe(true);
      await waitFor("hedge fates", () => seen.fatesLate === 2, 8000);
      const again = await fetch("http://127.0.0.1:18773/x");
      expect(again.status).toBe(200);
      expect((await server.stop(token)).kind).toBe("ok");
      return success(undefined);
    });
    expect(owned.cleanupFailed).toBe(false);
    expect(owned.completion.kind).toBe("ok");
  }, 30000);
});

describe("hedge over real HTTP (cancel-present fetch replicas)", () => {
  const decodeText = (bytes: Uint8Array) => success(new TextDecoder().decode(bytes));

  test("H4: without loser abort the response waits for loser drain", async () => {
    const seen: Record<string, unknown> = {};
    const owned = await runOwnedRoot(async () => {
      const upstream = startStallUpstream();
      try {
        const connection: Connection = {
          endpoint: upstream.url,
          timeoutMilliseconds: 10000,
          maxBodyBytes: 1024,
          headers: [],
        };
        const { token } = await startOn(18774, async () => {
          const scope = currentRequestScope();
          seen.scope = scope;
          const outcome = await hedge<string>(
            {
              boundMs: 30000,
              budget: scope?.budget,
              scopeSignal: scope?.signal,
              sink: scope?.sink,
              hedgeDelayMs: 50,
              abortLosers: false,
            },
            [
              {
                source: "fetch",
                cancelable: true,
                start: async ({ signal }) =>
                  value(
                    await performRequest(
                      connection,
                      {
                        path: "/slow",
                        method: "GET",
                        query: [{ name: "ms", value: "2000" }],
                        headers: [{ name: "x_replica", value: "0" }],
                        ownerSignal: signal ?? undefined,
                        budget: scope?.budget,
                      },
                      () => undefined,
                      decodeText,
                    ),
                  ),
              },
              {
                source: "fetch",
                cancelable: true,
                start: async ({ signal }) =>
                  value(
                    await performRequest(
                      connection,
                      {
                        path: "/fast",
                        method: "GET",
                        query: [],
                        headers: [{ name: "x_replica", value: "1" }],
                        ownerSignal: signal ?? undefined,
                        budget: scope?.budget,
                      },
                      () => undefined,
                      decodeText,
                    ),
                  ),
              },
            ],
          );
          if (outcome.kind !== "settled") return textResult("unexpected:" + outcome.kind);
          seen.winner = outcome.winner;
          // The handler responds WITHOUT awaiting the loser: any delay
          // past the winner is the pre-response scope drain, which is
          // exactly what this leg measures.
          void outcome.settledAll.then((fates) => {
            seen.fates = fates.length;
          });
          return textResult(`winner:${outcome.winner}:${outcome.value}`);
        });
        const start = Date.now();
        const response = await fetch("http://127.0.0.1:18774/x");
        const body = await response.text();
        const elapsed = Date.now() - start;
        // Content follows the winner, but timing follows the loser: the
        // loser's native owner group sits in the request scope, so the
        // pre-response drain waits for its two-second settlement. This
        // is the mechanism O2 supervision would address — and H5 shows
        // the O1 abort already covers it.
        expect(response.status).toBe(200);
        expect(body).toBe("winner:1:fast");
        expect(elapsed).toBeGreaterThanOrEqual(1500);
        expect(elapsed).toBeLessThan(8000);
        await waitFor("hedge fates", () => seen.fates === 2, 8000);
        expect(
          upstream
            .hits()
            .map((hit) => hit.path)
            .sort(),
        ).toEqual(["/fast", "/slow"]);
        expect(upstream.hits().find((hit) => hit.path === "/slow")?.aborted).toBe(false);
        const scope = seen.scope as RequestScope;
        expect(scope.collected.escalations).toHaveLength(0);
        expect(scope.collected.lates).toHaveLength(0);
        expect((await server.stop(token)).kind).toBe("ok");
      } finally {
        upstream.stop();
      }
      return success(undefined);
    });
    expect(owned.cleanupFailed).toBe(false);
    expect(owned.completion.kind).toBe("ok");
  }, 30000);

  test("H5: aborting the loser bounds the response under O1", async () => {
    const seen: Record<string, unknown> = {};
    const owned = await runOwnedRoot(async () => {
      const upstream = startStallUpstream();
      try {
        const connection: Connection = {
          endpoint: upstream.url,
          timeoutMilliseconds: 10000,
          maxBodyBytes: 1024,
          headers: [],
        };
        const { token } = await startOn(18775, async () => {
          const scope = currentRequestScope();
          seen.scope = scope;
          const outcome = await hedge<string>(
            {
              boundMs: 30000,
              budget: scope?.budget,
              scopeSignal: scope?.signal,
              sink: scope?.sink,
              hedgeDelayMs: 50,
              abortLosers: true,
            },
            [
              {
                source: "fetch",
                cancelable: true,
                start: async ({ signal }) =>
                  value(
                    await performRequest(
                      connection,
                      {
                        path: "/slow",
                        method: "GET",
                        query: [{ name: "ms", value: "2000" }],
                        headers: [{ name: "x_replica", value: "0" }],
                        ownerSignal: signal ?? undefined,
                        budget: scope?.budget,
                      },
                      () => undefined,
                      decodeText,
                    ),
                  ),
              },
              {
                source: "fetch",
                cancelable: true,
                start: async ({ signal }) =>
                  value(
                    await performRequest(
                      connection,
                      {
                        path: "/fast",
                        method: "GET",
                        query: [],
                        headers: [{ name: "x_replica", value: "1" }],
                        ownerSignal: signal ?? undefined,
                        budget: scope?.budget,
                      },
                      () => undefined,
                      decodeText,
                    ),
                  ),
              },
            ],
          );
          if (outcome.kind !== "settled") return textResult("unexpected:" + outcome.kind);
          seen.winner = outcome.winner;
          // Identical handler shape to H4: respond without awaiting
          // the loser, so only the abort policy differs.
          void outcome.settledAll.then((fates) => {
            seen.loserFate = fates.find((fate) => fate.replica === 0)?.fate;
          });
          return textResult(`winner:${outcome.winner}:${outcome.value}`);
        });
        const start = Date.now();
        const response = await fetch("http://127.0.0.1:18775/x");
        const body = await response.text();
        const elapsed = Date.now() - start;
        expect(response.status).toBe(200);
        expect(body).toBe("winner:1:fast");
        expect(elapsed).toBeLessThan(1500);
        await waitFor("loser fate", () => seen.loserFate === "expired", 8000);
        const scope = seen.scope as RequestScope;
        // The aborted loser files the honest cancel-cause escalation,
        // and its native abort lands as the seq-linked late rejection.
        // The native cause itself never enters the records.
        expect(scope.collected.escalations).toHaveLength(1);
        expect(scope.collected.escalations[0]).toMatchObject({ cause: "cancel" });
        await waitFor("aborted late settlement", () => scope.collected.lates.length === 1, 8000);
        expect(scope.collected.lates[0]).toMatchObject({
          seq: scope.collected.escalations[0]!.seq,
          settled: "rejected",
        });
        // Native proof the wire request actually aborted upstream.
        await waitFor(
          "upstream abort observation",
          () => upstream.hits().find((hit) => hit.path === "/slow")?.aborted === true,
          8000,
        );
        expect(upstream.hits()).toHaveLength(2);
        expect((await server.stop(token)).kind).toBe("ok");
      } finally {
        upstream.stop();
      }
      return success(undefined);
    });
    expect(owned.cleanupFailed).toBe(false);
    expect(owned.completion.kind).toBe("ok");
  }, 30000);
});

describe("hedge lifetime: late failure, disconnect, shutdown, cleanup", () => {
  test("H6: a late loser failure is owner-observed after the response", async () => {
    const seen: Record<string, unknown> = {};
    const owned = await runOwnedRoot(async () => {
      const { token } = await startOn(18776, async () => {
        const scope = currentRequestScope();
        seen.scope = scope;
        const outcome = await hedge<string>(
          {
            boundMs: 30000,
            budget: scope?.budget,
            scopeSignal: scope?.signal,
            sink: scope?.sink,
            hedgeDelayMs: 0,
            abortLosers: false,
          },
          [
            {
              source: "worker",
              cancelable: false,
              start: async () => {
                await Bun.sleep(10);
                return "winner-value";
              },
            },
            {
              source: "worker",
              cancelable: false,
              start: async () => {
                await Bun.sleep(500);
                throw new Error("late-boom");
              },
            },
          ],
        );
        if (outcome.kind !== "settled") return textResult("unexpected:" + outcome.kind);
        void outcome.settledAll.then((fates) => {
          seen.loserFate = fates.find((fate) => fate.replica === 1)?.fate;
          seen.loserError = (
            fates.find((fate) => fate.replica === 1) as { error?: unknown }
          )?.error;
        });
        return textResult(`winner:${outcome.winner}:${outcome.value}`);
      });
      const start = Date.now();
      const response = await fetch("http://127.0.0.1:18776/x");
      const body = await response.text();
      const elapsed = Date.now() - start;
      expect(response.status).toBe(200);
      expect(body).toBe("winner:0:winner-value");
      // The response lands before the loser even fails.
      expect(elapsed).toBeLessThan(400);
      await waitFor("late loser rejection", () => seen.loserFate === "rejected", 8000);
      expect((seen.loserError as Error).message).toBe("late-boom");
      const scope = seen.scope as RequestScope;
      // No boundary expired, so the shared sink stays silent: the late
      // failure is observed by the hedge owner (fates) only. Nothing
      // rejects unobserved — the run completes clean.
      expect(scope.collected.escalations).toHaveLength(0);
      expect(scope.collected.lates).toHaveLength(0);
      const again = await fetch("http://127.0.0.1:18776/x");
      expect(again.status).toBe(200);
      expect((await server.stop(token)).kind).toBe("ok");
      return success(undefined);
    });
    expect(owned.cleanupFailed).toBe(false);
    expect(owned.completion.kind).toBe("ok");
  }, 30000);

  test("H7: disconnect expires every replica boundary while work stays owned", async () => {
    const seen: Record<string, unknown> = {};
    const owned = await runOwnedRoot(async () => {
      const { token } = await startOn(18777, async () => {
        const scope = currentRequestScope();
        seen.scope = scope;
        const start = Date.now();
        const outcome = await hedge<string>(
          {
            boundMs: 30000,
            budget: scope?.budget,
            scopeSignal: scope?.signal,
            sink: scope?.sink,
            hedgeDelayMs: 20,
            abortLosers: false,
          },
          [0, 1].map((replica) => ({
            source: "worker" as const,
            cancelable: false,
            start: async () => {
              await Bun.sleep(2000);
              return replica === 0 ? "primary" : "secondary";
            },
          })),
        );
        seen.boundaryAt = Date.now() - start;
        // First request (disconnect): unknown. Second request (health
        // check): settled with the primary winning the even race.
        if (outcome.kind !== "unknown") {
          if (outcome.kind !== "settled") return textResult("unexpected:" + outcome.kind);
          return textResult(`winner:${outcome.winner}:${outcome.value}`);
        }
        seen.cause = outcome.escalation.cause;
        return textResult("unknown");
      });
      const stop = new AbortController();
      const outcome = fetch("http://127.0.0.1:18777/x", { signal: stop.signal }).then(
        () => "responded",
        (cause: unknown) => (cause instanceof Error ? cause.name : typeof cause),
      );
      await Bun.sleep(300);
      stop.abort();
      expect(await outcome).toBe("AbortError");
      await waitFor("hedge boundary", () => seen.boundaryAt !== undefined, 5000);
      expect(seen.boundaryAt as number).toBeLessThan(1500);
      expect(seen.cause).toBe("disconnect");
      const scope = seen.scope as RequestScope;
      expect(scope.collected.escalations).toHaveLength(2);
      await waitFor("owned late settlement", () => scope.collected.lates.length === 2, 8000);
      const again = await fetch("http://127.0.0.1:18777/x");
      expect(again.status).toBe(200);
      expect(await again.text()).toBe("winner:0:primary");
      expect((await server.stop(token)).kind).toBe("ok");
      return success(undefined);
    });
    expect(owned.cleanupFailed).toBe(false);
    expect(owned.completion.kind).toBe("ok");
  }, 30000);

  test("H8: stop bounds the live hedge peer with the shutdown outcome", async () => {
    const seen: Record<string, unknown> = {};
    const owned = await runOwnedRoot(async () => {
      const { token } = await startOn(18778, async () => {
        const scope = currentRequestScope();
        seen.scope = scope;
        const outcome = await hedge<string>(
          {
            boundMs: 30000,
            budget: scope?.budget,
            scopeSignal: scope?.signal,
            sink: scope?.sink,
            hedgeDelayMs: 0,
            abortLosers: false,
          },
          [0, 1].map((replica) => ({
            source: "worker" as const,
            cancelable: false,
            start: async () => {
              await Bun.sleep(5000);
              return replica === 0 ? "primary" : "secondary";
            },
          })),
        );
        if (outcome.kind !== "unknown") return textResult("unexpected:" + outcome.kind);
        return textResult(`unknown:${outcome.escalation.cause}`);
      });
      const response = await Promise.all([
        fetch("http://127.0.0.1:18778/x").then((r) => r.text()),
        Bun.sleep(200).then(() => server.stop(token)),
      ]);
      // The peer stayed connected and received the bounded shutdown
      // outcome instead of waiting out the five-second replicas.
      expect(response[0]).toBe("unknown:shutdown");
      expect(response[1].kind).toBe("ok");
      const scope = seen.scope as RequestScope;
      expect(scope.collected.escalations).toHaveLength(2);
      expect(scope.collected.escalations[0]).toMatchObject({ cause: "shutdown" });
      await waitFor("owned late settlement", () => scope.collected.lates.length === 2, 10000);
      return success(undefined);
    });
    expect(owned.cleanupFailed).toBe(false);
    expect(owned.completion.kind).toBe("ok");
  }, 30000);

  test("H9: SIGTERM bounds the live hedge peer with the shutdown outcome", async () => {
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
 const config=value(await server.makeConfig("127.0.0.1",18779n,1048576n,5000n));
 const route=value(await router.get("/hedged",async ()=>{
   console.log("handler-enter");
   const scope=currentRequestScope();
   const outcome=await hedge({boundMs:30000,budget:scope?.budget,scopeSignal:scope?.signal,sink:scope?.sink,hedgeDelayMs:0,abortLosers:false},[0,1].map(()=>({source:"worker",cancelable:false,start:async()=>{await Bun.sleep(5000);return "slow";}})));
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
    const result = await runChild("sigterm-hedge", script, async (child, output) => {
      await waitForOutput(output, "listening");
      const pending = fetch("http://127.0.0.1:18779/hedged").then((r) => r.text());
      await waitForOutput(output, "handler-enter");
      await Bun.sleep(200);
      child.kill("SIGTERM");
      // The connected peer receives the bounded shutdown outcome for
      // the whole hedge, not one replica's five-second wait.
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

  test("H10: the pool lease survives the hedge response; close drains it", async () => {
    const seen: Record<string, unknown> = {};
    const owned = await runOwnedRoot(async () => {
      const poolToken = value(await pools.sqliteOpenMemory());
      for (const name of ["drop_probe", "setup_probe", "seed_probe"]) {
        value(
          await pools.execute(
            descriptors.declareDescriptor("lite", name),
            emptyPlan,
            poolToken,
            emptyParams,
          ),
        );
      }
      const { token } = await startOn(18780, async () => {
        const scope = currentRequestScope();
        seen.scope = scope;
        const outcome = await hedge<string>(
          {
            boundMs: 30000,
            budget: scope?.budget,
            scopeSignal: scope?.signal,
            sink: scope?.sink,
            hedgeDelayMs: 0,
            abortLosers: false,
          },
          [
            {
              source: "sql",
              cancelable: false,
              start: async () => {
                const rows = await pools.queryOne(
                  descriptors.declareDescriptor("lite", "probe_note"),
                  notePlan,
                  poolToken,
                  idParam(1n),
                );
                if (rows.kind !== "ok") throw new Error("probe query failed");
                return "winner-query";
              },
            },
            {
              source: "worker",
              cancelable: false,
              start: async () =>
                value(
                  await useResource(poolToken, "sql-pool", async () => {
                    await Bun.sleep(800);
                    return success("lease-held");
                  }),
                ),
            },
          ],
        );
        if (outcome.kind !== "settled") return textResult("unexpected:" + outcome.kind);
        void outcome.settledAll.then((fates) => {
          seen.fatesLate = fates.length;
        });
        return textResult(`winner:${outcome.winner}:${outcome.value}`);
      });
      const start = Date.now();
      const response = await fetch("http://127.0.0.1:18780/x");
      const body = await response.text();
      const elapsed = Date.now() - start;
      expect(response.status).toBe(200);
      expect(body).toBe("winner:0:winner-query");
      expect(elapsed).toBeLessThan(1500);
      // The lease is held past the visible return: no timer revoked it.
      // The pool lives in the test-root scope (the production long-lived
      // shape), so the request drain never waits for it.
      expect(resourceStatus(poolToken).leases).toBe(1);
      const closeStart = Date.now();
      value(await pools.close(poolToken, 5000n));
      // Close drains the lease instead of discarding it.
      expect(Date.now() - closeStart).toBeGreaterThanOrEqual(500);
      expect(resourceStatus(poolToken).leases).toBe(0);
      expect(resourceStatus(poolToken).state).toBe("closed");
      await waitFor("hedge fates", () => seen.fatesLate === 2, 8000);
      const scope = seen.scope as RequestScope;
      expect(scope.collected.escalations).toHaveLength(0);
      // No post-disposal use: work after close fails closed.
      await expect(
        pools.queryOne(
          descriptors.declareDescriptor("lite", "probe_note"),
          notePlan,
          poolToken,
          idParam(1n),
        ),
      ).rejects.toThrow();
      expect((await server.stop(token)).kind).toBe("ok");
      return success(undefined);
    });
    expect(owned.cleanupFailed).toBe(false);
    expect(owned.completion.kind).toBe("ok");
  }, 30000);
});

describe("hedge over live SQL (cancel-absent replicas)", () => {
  pgLive(
    "H11: the PG lease survives the hedge response; the backend runs on",
    async () => {
      const seen: Record<string, unknown> = {};
      const owned = await runOwnedRoot(async () => {
        const poolToken = await setupLiveSchema(() => pools.open("CAN_E05_PG", 5n), "pg");
        try {
          const { token } = await startOn(18781, async () => {
            const scope = currentRequestScope();
            seen.scope = scope;
            const outcome = await hedge<string>(
              {
                boundMs: 30000,
                budget: scope?.budget,
                scopeSignal: scope?.signal,
                sink: scope?.sink,
                hedgeDelayMs: 50,
                abortLosers: false,
              },
              [
                {
                  source: "sql",
                  cancelable: false,
                  start: async () => {
                    const rows = await pools.queryRows(
                      descriptors.declareDescriptor("pg", "sleep_mark"),
                      markPlan,
                      poolToken,
                      emptyParams,
                      10n,
                    );
                    if (rows.kind !== "ok") throw new Error("sleep query failed");
                    return "slow-sleep";
                  },
                },
                {
                  source: "sql",
                  cancelable: false,
                  start: async () => {
                    const one = await pools.queryOne(
                      descriptors.declareDescriptor("pg", "probe_note"),
                      notePlan,
                      poolToken,
                      idParam(1n),
                    );
                    if (one.kind !== "ok") throw new Error("probe query failed");
                    return "fast-probe";
                  },
                },
              ],
            );
            if (outcome.kind !== "settled") return textResult("unexpected:" + outcome.kind);
            seen.winner = outcome.winner;
            void outcome.settledAll.then((fates) => {
              seen.fatesLate = fates.length;
            });
            return textResult(`winner:${outcome.winner}:${outcome.value}`);
          });
          const start = Date.now();
          const response = await fetch("http://127.0.0.1:18781/x");
          const body = await response.text();
          const elapsed = Date.now() - start;
          expect(response.status).toBe(200);
          expect(body).toBe("winner:1:fast-probe");
          // Bounded by the fast probe, not the two-second sleep: live
          // SQL losers hold pool leases, not request-scope groups, so
          // the pre-response drain never waits for them.
          expect(elapsed).toBeLessThan(1500);
          expect(resourceStatus(poolToken).leases).toBe(1);
          expect(await pgSleepers(RUN_DB)).toBe(1);
          const overlap = await pools.queryOne(
            descriptors.declareDescriptor("pg", "probe_note"),
            notePlan,
            poolToken,
            idParam(1n),
          );
          expect(overlap.kind).toBe("ok");
          const scope = seen.scope as RequestScope;
          expect(scope.collected.escalations).toHaveLength(0);
          await waitFor("owned late settlement", () => seen.fatesLate === 2, 8000);
          expect(resourceStatus(poolToken).leases).toBe(0);
          expect(await pgSleepers(RUN_DB)).toBe(0);
          value(await pools.close(poolToken, 5000n));
          expect((await server.stop(token)).kind).toBe("ok");
        } catch (cause) {
          value(await pools.close(poolToken, 15000n));
          throw cause;
        }
        return success(undefined);
      });
      expect(owned.cleanupFailed).toBe(false);
      expect(owned.completion.kind).toBe("ok");
    },
    30000,
  );

  myLive(
    "H12: the MySQL thread survives the hedge response; the pool stays usable",
    async () => {
      const seen: Record<string, unknown> = {};
      const owned = await runOwnedRoot(async () => {
        const poolToken = await setupLiveSchema(() => pools.mysqlOpen("CAN_E05_MY", 5n), "my");
        try {
          const { token } = await startOn(18782, async () => {
            const scope = currentRequestScope();
            seen.scope = scope;
            const outcome = await hedge<string>(
              {
                boundMs: 30000,
                budget: scope?.budget,
                scopeSignal: scope?.signal,
                sink: scope?.sink,
                hedgeDelayMs: 50,
                abortLosers: false,
              },
              [
                {
                  source: "sql",
                  cancelable: false,
                  start: async () => {
                    const rows = await pools.queryRows(
                      descriptors.declareDescriptor("my", "sleep_mark"),
                      markPlan,
                      poolToken,
                      emptyParams,
                      10n,
                    );
                    if (rows.kind !== "ok") throw new Error("sleep query failed");
                    return "slow-sleep";
                  },
                },
                {
                  source: "sql",
                  cancelable: false,
                  start: async () => {
                    const one = await pools.queryOne(
                      descriptors.declareDescriptor("my", "probe_note"),
                      notePlan,
                      poolToken,
                      idParam(1n),
                    );
                    if (one.kind !== "ok") throw new Error("probe query failed");
                    return "fast-probe";
                  },
                },
              ],
            );
            if (outcome.kind !== "settled") return textResult("unexpected:" + outcome.kind);
            seen.winner = outcome.winner;
            void outcome.settledAll.then((fates) => {
              seen.fatesLate = fates.length;
            });
            return textResult(`winner:${outcome.winner}:${outcome.value}`);
          });
          const start = Date.now();
          const response = await fetch("http://127.0.0.1:18782/x");
          const body = await response.text();
          const elapsed = Date.now() - start;
          expect(response.status).toBe(200);
          expect(body).toBe("winner:1:fast-probe");
          expect(elapsed).toBeLessThan(1500);
          expect(resourceStatus(poolToken).leases).toBe(1);
          expect(await mysqlSleepers(RUN_DB)).toBe(1);
          const overlap = await pools.queryOne(
            descriptors.declareDescriptor("my", "probe_note"),
            notePlan,
            poolToken,
            idParam(1n),
          );
          expect(overlap.kind).toBe("ok");
          const scope = seen.scope as RequestScope;
          expect(scope.collected.escalations).toHaveLength(0);
          await waitFor("owned late settlement", () => seen.fatesLate === 2, 8000);
          expect(resourceStatus(poolToken).leases).toBe(0);
          expect(await mysqlSleepers(RUN_DB)).toBe(0);
          value(await pools.close(poolToken, 5000n));
          expect((await server.stop(token)).kind).toBe("ok");
        } catch (cause) {
          value(await pools.close(poolToken, 15000n));
          throw cause;
        }
        return success(undefined);
      });
      expect(owned.cleanupFailed).toBe(false);
      expect(owned.completion.kind).toBe("ok");
    },
    30000,
  );
});

// Child-process signal leg: signals cannot target the runner itself.
const childRuntime = fileURLToPath(new URL("../", import.meta.url));
const childHarness = fileURLToPath(new URL("./http-hedge-harness.ts", import.meta.url));
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
import {hedge} from ${JSON.stringify(childHarness)};
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
  killAfterMs = 10000,
): Promise<{ output: string; errors: string; code: number | null; signal: NodeJS.Signals | null }> {
  const directory = mkdtempSync(join(tmpdir(), "can-hedge-" + name + "-"));
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
