// F02 live experiment driver: PG locking-read descriptor probe (X-R10-1)
// plus the post-write-read/race-cost RETURNING demonstration. Executed by
// tests/integration/sql_f02_test.go against a disposable PostgreSQL named
// by CAN_TEST_POSTGRES_URL (a concrete per-run URL, or the provision
// template with a <db> placeholder selecting can_f02) and, optionally, a
// MySQL named by CAN_TEST_MYSQL_URL. All Can-path legs run through the
// real runtime (createSQLPools/createSQLTransactions) with descriptor
// segments mirroring compiler/internal/sql checker output for the pinned
// statements (admission itself is pinned by locking_probe_test.go);
// RETURNING appears only in raw oracle legs, never in a descriptor.
//
// The driver REPORTS: it always prints one JSON report and exits 0 unless
// the harness itself fails (then {error} + exit 1). Adverse outcomes are
// valid findings and surface as report fields, never as crashes.
import { createHash } from "node:crypto";
import { catalogue } from "../../../../runtime/catalogue.ts";
import {
  createDomainRuntime,
  domainFailureDiagnostics,
  type FailureShape,
} from "../../../../runtime/domain.ts";
import { success, value } from "../../../../runtime/completion.ts";
import { record, dataProperty } from "../../../../runtime/data.ts";
import { runOwnedRoot } from "../../../../runtime/owner.ts";
import {
  createSQLDescriptors,
  type SQLDescriptorEntry,
} from "../../../../runtime/platform/sql/descriptor.ts";
import { createSQLPools } from "../../../../runtime/platform/sql/pool.ts";
import { createSQLTransactions } from "../../../../runtime/platform/sql/transaction.ts";
import type { SQLPlan } from "../../../../runtime/platform/sql/values.ts";

const pgTemplate = process.env.CAN_TEST_POSTGRES_URL ?? "";
const mysqlTemplate = process.env.CAN_TEST_MYSQL_URL ?? "";
if (!pgTemplate) {
  console.log(JSON.stringify({ error: "set CAN_TEST_POSTGRES_URL to the disposable test database" }));
  process.exit(1);
}
if (!pgTemplate.startsWith("postgres")) {
  console.log(JSON.stringify({ error: "refusing non-postgres test URL" }));
  process.exit(1);
}
const pgUrl = pgTemplate.includes("<db>") ? pgTemplate.replace("<db>", "can_f02") : pgTemplate;
const mysqlUrl = !mysqlTemplate
  ? ""
  : mysqlTemplate.includes("<db>")
    ? mysqlTemplate.replace("<db>", "can_f02")
    : mysqlTemplate;

const identity = (kind: string, declaration: string) =>
  createHash("sha256")
    .update("can-concrete-type-v1\0" + JSON.stringify([kind, declaration]))
    .digest("hex");
const textShape: FailureShape = {
  identity: identity("primitive", "str"), kind: "primitive", declaration: "str",
  arguments: [], fields: [], leaves: [], inputs: [], errors: [],
};
const intShape: FailureShape = {
  identity: identity("primitive", "int"), kind: "primitive", declaration: "int",
  arguments: [], fields: [], leaves: [], inputs: [], errors: [],
};
const fieldTypes: Record<string, Record<string, string>> = {
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
  "http::credentials_missing": { variable: textShape.identity },
};
const declarations = catalogue.errors
  .filter((e) => fieldTypes[e.name] !== undefined)
  .map((e) => ({ identity: e.identity, name: e.name, parameters: 0 }));
const errorShapes: FailureShape[] = declarations.map((e) => ({
  identity: identity("error", e.identity), kind: "error", declaration: e.identity,
  arguments: [],
  fields: Object.entries(fieldTypes[e.name]!).map(([name, type]) => ({ name, type })),
  leaves: [], inputs: [], errors: [],
}));
const domain = createDomainRuntime({ declarations, shapes: [textShape, intShape, ...errorShapes] });
const id = (declaration: string) => identity("error", declaration);

const claimHead = "SELECT id, payload FROM f02_claims WHERE worker = ";
const claimTail = " ORDER BY id LIMIT ";
const table: Record<string, Record<string, SQLDescriptorEntry>> = {
  "": {
    claim_skip: {
      dialect: "postgresql", cardinality: "many", kind: "SelectStmt",
      segments: [{ text: claimHead }, { param: 1 }, { text: claimTail }, { param: 2 }, { text: " FOR UPDATE SKIP LOCKED" }],
      params: ["worker"], paramType: "p", rowType: "r", limit: 2, total: 2, version: 170007,
    },
    claim_block: {
      dialect: "postgresql", cardinality: "many", kind: "SelectStmt",
      segments: [{ text: claimHead }, { param: 1 }, { text: claimTail }, { param: 2 }, { text: " FOR UPDATE" }],
      params: ["worker"], paramType: "p", rowType: "r", limit: 2, total: 2, version: 170007,
    },
    claim_nowait: {
      dialect: "postgresql", cardinality: "many", kind: "SelectStmt",
      segments: [{ text: claimHead }, { param: 1 }, { text: claimTail }, { param: 2 }, { text: " FOR UPDATE NOWAIT" }],
      params: ["worker"], paramType: "p", rowType: "r", limit: 2, total: 2, version: 170007,
    },
    claim_nokey: {
      dialect: "postgresql", cardinality: "many", kind: "SelectStmt",
      segments: [{ text: claimHead }, { param: 1 }, { text: claimTail }, { param: 2 }, { text: " FOR NO KEY UPDATE SKIP LOCKED" }],
      params: ["worker"], paramType: "p", rowType: "r", limit: 2, total: 2, version: 170007,
    },
    ins_claim: {
      dialect: "postgresql", cardinality: "execute", kind: "InsertStmt",
      segments: [{ text: "INSERT INTO f02_claims (worker, payload) VALUES (" }, { param: 1 }, { text: ", " }, { param: 2 }, { text: ")" }],
      params: ["worker", "payload"], paramType: "p", rowType: "r", limit: 0, total: 2, version: 170007,
    },
    get_claim_by_payload: {
      dialect: "postgresql", cardinality: "one", kind: "SelectStmt",
      segments: [{ text: claimHead }, { param: 1 }, { text: " AND payload = " }, { param: 2 }, { text: " LIMIT " }, { param: 3 }],
      params: ["worker", "payload"], paramType: "p", rowType: "r", limit: 3, total: 3, version: 170007,
    },
    get_by_payload: {
      dialect: "postgresql", cardinality: "many", kind: "SelectStmt",
      segments: [{ text: "SELECT id, payload FROM f02_gen WHERE payload = " }, { param: 1 }, { text: " ORDER BY id LIMIT " }, { param: 2 }],
      params: ["payload"], paramType: "p", rowType: "r", limit: 2, total: 2, version: 170007,
    },
    ins_gen: {
      dialect: "postgresql", cardinality: "execute", kind: "InsertStmt",
      segments: [{ text: "INSERT INTO f02_gen (payload) VALUES (" }, { param: 1 }, { text: ")" }],
      params: ["payload"], paramType: "p", rowType: "r", limit: 0, total: 1, version: 170007,
    },
  },
};
const descriptors = createSQLDescriptors(table);
const env = new Map<string, string>([["F02_PG", pgUrl]]);
const pools = createSQLPools(
  domain,
  {
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
  },
  (n) => env.get(n),
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

const COMMIT = "app::sql_commit_leaf";
const ROLLBACK = "app::sql_rollback_leaf";
const leaves = { commit: COMMIT, rollback: ROLLBACK };
const claimRow = {
  root: "r",
  fields: [{ name: "id", kind: "int" as const }, { name: "payload", kind: "str" as const }],
};
const workerPlan: SQLPlan = {
  params: { root: "p", fields: [{ name: "worker", kind: "str" }] }, rows: claimRow,
};
const payloadPlan: SQLPlan = {
  params: { root: "p", fields: [{ name: "payload", kind: "str" }] }, rows: claimRow,
};
const keyPlan: SQLPlan = {
  params: { root: "p", fields: [{ name: "worker", kind: "str" }, { name: "payload", kind: "str" }] },
  rows: claimRow,
};
const insClaimPlan: SQLPlan = {
  params: { root: "p", fields: [{ name: "worker", kind: "str" }, { name: "payload", kind: "str" }] },
};
const insGenPlan: SQLPlan = {
  params: { root: "p", fields: [{ name: "payload", kind: "str" }] },
};

const failName = (c: { kind: string; failure?: { name?: string } }) =>
  c.kind === "ok" ? "ok" : String((c as { failure?: { name?: string } }).failure?.name ?? c.kind);
const domainOutcomeOf = (c: { kind: string; value?: unknown }) => {
  const details = domainFailureDiagnostics((c as { value: unknown }).value);
  const payload = details.payload as Record<string, unknown>;
  const plain: Record<string, unknown> = {};
  for (const key of Object.keys(payload)) plain[key] = payload[key];
  return { name: details.declaration.name, payload: plain };
};

try {
  const report: Record<string, unknown> = {};
  // DDL is operator-owned; Can cannot run it, so setup/teardown stays raw.
  const raw = new Bun.SQL(pgUrl, { adapter: "postgres", bigint: true, max: 8 });
  try {
    const serverVersion = (await raw`SELECT version() AS v` as { v: string }[])[0]!.v;
    report.serverVersion = serverVersion.split(" ").slice(0, 2).join(" ");
    await raw`DROP TABLE IF EXISTS f02_claims`;
    await raw`DROP TABLE IF EXISTS f02_gen`;
    await raw`CREATE TABLE f02_claims (id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY, worker TEXT NOT NULL, payload TEXT NOT NULL)`;
    await raw`CREATE TABLE f02_gen (id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY, payload TEXT NOT NULL)`;

    const rootResult = await runOwnedRoot(async () => {
      const poolA = value(await pools.open("F02_PG", 5n));
      const poolB = value(await pools.open("F02_PG", 5n));
      const claimSkip = descriptors.declareDescriptor("", "claim_skip");
      const claimBlock = descriptors.declareDescriptor("", "claim_block");
      const claimNowait = descriptors.declareDescriptor("", "claim_nowait");
      const claimNokey = descriptors.declareDescriptor("", "claim_nokey");
      const insClaim = descriptors.declareDescriptor("", "ins_claim");
      const getClaimByPayload = descriptors.declareDescriptor("", "get_claim_by_payload");
      const getByPayload = descriptors.declareDescriptor("", "get_by_payload");
      const insGen = descriptors.declareDescriptor("", "ins_gen");
      const wparams = record("p", [["worker", "w"]]);

      for (const p of ["a", "b", "c", "d"]) {
        const r = await pools.execute(
          insClaim, insClaimPlan, poolA, record("p", [["worker", "w"], ["payload", p]]),
        );
        if (r.kind !== "ok") throw new Error("seed failed: " + failName(r));
      }

      // Part 1: locking-read probe through the runtime path.
      const locking: Record<string, unknown> = {};
      let release1!: () => void;
      const gate1 = new Promise<void>((r) => (release1 = r));
      const tx1 = transactions.withTransaction(poolA, async (h: unknown) => {
        const rows = value(await transactions.queryRows(claimSkip, workerPlan, h, wparams, 10n));
        locking.tx1_claimed = (rows as unknown[]).length;
        await gate1;
        return success(record(COMMIT, [["value", 1n]]));
      }, leaves);
      for (let i = 0; i < 200 && locking.tx1_claimed === undefined; i++) await Bun.sleep(20);
      if (locking.tx1_claimed === undefined) throw new Error("tx1 never claimed");

      const tx2skip = await transactions.withTransaction(poolB, async (h: unknown) => {
        const rows = value(await transactions.queryRows(claimSkip, workerPlan, h, wparams, 10n));
        return success(record(COMMIT, [["value", BigInt((rows as unknown[]).length)]]));
      }, leaves);
      locking.tx2_skipped_count = tx2skip.kind === "ok"
        ? Number(value(tx2skip) as bigint)
        : "tx-failed:" + failName(tx2skip);

      const nowait = await pools.queryRows(claimNowait, workerPlan, poolB, wparams, 10n);
      locking.tx2_nowait = nowait.kind === "ok"
        ? { outcome: "unexpected-ok" }
        : { outcome: "domain", ...domainOutcomeOf(nowait) };

      let inTxnSeen = "none";
      const txNowait = await transactions.withTransaction(poolB, async (h: unknown) => {
        const r = await transactions.queryRows(claimNowait, workerPlan, h, wparams, 10n);
        inTxnSeen = r.kind === "ok" ? "unexpected-ok" : JSON.stringify(domainOutcomeOf(r));
        return success(record(COMMIT, [["value", 1n]]));
      }, leaves);
      locking.tx2_nowait_in_txn = { query: inTxnSeen, txn: failName(txNowait) };

      release1!();
      locking.tx1_commit = failName(await tx1);
      const tx2after = await transactions.withTransaction(poolB, async (h: unknown) => {
        const rows = value(await transactions.queryRows(claimSkip, workerPlan, h, wparams, 10n));
        return success(record(COMMIT, [["value", BigInt((rows as unknown[]).length)]]));
      }, leaves);
      locking.tx2_after_release = tx2after.kind === "ok"
        ? Number(value(tx2after) as bigint)
        : "tx-failed:" + failName(tx2after);
      const blk = await transactions.withTransaction(poolB, async (h: unknown) => {
        const rows = value(await transactions.queryRows(claimBlock, workerPlan, h, wparams, 10n));
        return success(record(COMMIT, [["value", BigInt((rows as unknown[]).length)]]));
      }, leaves);
      locking.blocking_for_update_count = blk.kind === "ok"
        ? Number(value(blk) as bigint)
        : "tx-failed:" + failName(blk);
      const nokey = await pools.queryRows(claimNokey, workerPlan, poolB, wparams, 10n);
      locking.nokey_count = nokey.kind === "ok" ? (value(nokey) as unknown[]).length : "failed:" + failName(nokey);
      report.locking = locking;

      // Part 2A: round-trip cost of insert-plus-refetch vs single RETURNING.
      const N = 60;
      const costKeys = Array.from({ length: N }, (_, i) => `cost-${i}`);
      const tCan0 = performance.now();
      for (const key of costKeys) {
        const r = await transactions.withTransaction(poolA, async (h: unknown) => {
          value(await transactions.execute(
            insClaim, insClaimPlan, h, record("p", [["worker", "cost"], ["payload", key]]),
          ));
          const row = value(await transactions.queryOne(
            getClaimByPayload, keyPlan, h, record("p", [["worker", "cost"], ["payload", key]]),
          ));
          if (dataProperty(row as never, "payload") !== key) throw new Error("refetch mismatch");
          return success(record(COMMIT, [["value", 1n]]));
        }, leaves);
        if (r.kind !== "ok") throw new Error("cost leg failed: " + failName(r));
      }
      const tCan1 = performance.now();
      const tRaw0 = performance.now();
      for (const key of costKeys) {
        await raw`INSERT INTO f02_gen (payload) VALUES (${"ret-" + key}) RETURNING id`;
      }
      const tRaw1 = performance.now();
      report.cost = {
        iterations: N,
        can_txn_insert_plus_refetch_ms: +((tCan1 - tCan0).toFixed(1)),
        raw_single_returning_ms: +((tRaw1 - tRaw0).toFixed(1)),
      };

      // Part 2B: refetch ambiguity without a natural unique key.
      await raw`TRUNCATE f02_gen`;
      const K = 8;
      const inserts = await Promise.all(Array.from({ length: K }, () =>
        transactions.withTransaction(poolA, async (h: unknown) => {
          value(await transactions.execute(insGen, insGenPlan, h, record("p", [["payload", "same"]])));
          return success(record(COMMIT, [["value", 1n]]));
        }, leaves)));
      if (!inserts.every((r) => r.kind === "ok")) throw new Error("ambiguity inserts failed");
      const cands = value(
        await pools.queryRows(getByPayload, payloadPlan, poolA, record("p", [["payload", "same"]]), 100n),
      ) as unknown[];
      report.ambiguity = {
        concurrent_identical_inserts: K,
        refetch_candidate_rows: cands.length,
        writer_identifiable: cands.length === 1,
      };
      const ret = (await raw`INSERT INTO f02_gen (payload) VALUES ('same') RETURNING id` as { id: bigint }[]);
      report.returning_oracle = { rows: ret.length };

      // Part 2C: keyed control — same-txn refetch by app-supplied unique key.
      const M = 16;
      const raced = await Promise.all(Array.from({ length: M }, (_, i) =>
        transactions.withTransaction(poolA, async (h: unknown) => {
          const key = `race-${i}`;
          value(await transactions.execute(
            insClaim, insClaimPlan, h, record("p", [["worker", "race"], ["payload", key]]),
          ));
          const row = value(await transactions.queryOne(
            getClaimByPayload, keyPlan, h, record("p", [["worker", "race"], ["payload", key]]),
          ));
          const got = dataProperty(row as never, "payload");
          return success(record(COMMIT, [["value", got === key ? 1n : 0n]]));
        }, leaves)));
      report.keyed_control = {
        writers: M,
        all_refetched_own_row: raced.every((r) => r.kind === "ok" && (value(r) as bigint) === 1n),
      };

      // MySQL leg: INSERT...RETURNING admissibility on the second dialect.
      if (!mysqlUrl) {
        report.mysql = "skipped:no-mysql-url";
      } else {
        const my = new Bun.SQL(mysqlUrl, { adapter: "mysql", max: 1, tls: true });
        try {
          const mv = (await my`SELECT VERSION() AS v` as { v: string }[])[0]!.v;
          await my`CREATE TABLE IF NOT EXISTS f02_ret (id BIGINT AUTO_INCREMENT PRIMARY KEY, payload TEXT)`;
          let insertReturning: unknown;
          try {
            await my`INSERT INTO f02_ret (payload) VALUES ('x') RETURNING id`;
            insertReturning = "admitted";
          } catch (e) {
            insertReturning = {
              rejected: true,
              code: (e as { code?: unknown }).code ?? null,
              errno: (e as { errno?: unknown }).errno ?? null,
            };
          }
          report.mysql = { serverVersion: mv, insert_returning: insertReturning };
          await my`DROP TABLE IF EXISTS f02_ret`;
        } finally {
          await my.close();
        }
      }

      value(await pools.close(poolA, 5000n));
      value(await pools.close(poolB, 5000n));
      return success(undefined);
    });
    if (rootResult.completion.kind !== "ok" || rootResult.cleanupFailed) {
      throw new Error("owned run failed");
    }
    await raw`DROP TABLE IF EXISTS f02_claims`;
    await raw`DROP TABLE IF EXISTS f02_gen`;
    await raw.close();
    console.log(JSON.stringify(report, (_, v) => (typeof v === "bigint" ? Number(v) : v)));
  } catch (error) {
    try {
      await raw`DROP TABLE IF EXISTS f02_claims`;
      await raw`DROP TABLE IF EXISTS f02_gen`;
    } catch { /* best effort */ }
    await raw.close().catch(() => undefined);
    throw error;
  }
} catch (error) {
  console.log(JSON.stringify({ error: error instanceof Error ? error.message : String(error) }));
  process.exitCode = 1;
}
