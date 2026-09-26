// F03 live qualification driver: per-dialect RETURNING contract legs,
// lookup-first PG recipe, and blessed-encoding roundtrips. Executed by
// tests/integration/sql_f03_test.go against a disposable PostgreSQL named
// by CAN_TEST_POSTGRES_URL (a concrete per-run URL, or the provision
// template with a <db> placeholder selecting can_f03) and, optionally, a
// MySQL named by CAN_TEST_MYSQL_URL. SQLite legs run in-memory.
//
// Shape rule, mirroring F02: everything expressible through checked
// descriptors today runs through the real runtime
// (createSQLPools/createSQLTransactions) with descriptor segments
// mirroring compiler/internal/sql checker output; RETURNING appears only
// in raw oracle legs, never in a descriptor, because the admitted
// INSERT...RETURNING execution path is the E-owned handoff this task
// specifies but does not implement.
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
import { record, recordIdentity, dataProperty } from "../../../../runtime/data.ts";
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
const pgUrl = pgTemplate.includes("<db>") ? pgTemplate.replace("<db>", "can_f03") : pgTemplate;
const mysqlUrl = !mysqlTemplate
  ? ""
  : mysqlTemplate.includes("<db>")
    ? mysqlTemplate.replace("<db>", "can_f03")
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

const table: Record<string, Record<string, SQLDescriptorEntry>> = {
  "": {
    delivery_by_id: {
      dialect: "postgresql", cardinality: "optional", kind: "SelectStmt",
      segments: [{ text: "SELECT delivery_id, digest, state, updated_ms FROM f03_deliveries WHERE delivery_id = " }, { param: 1 }, { text: " LIMIT " }, { param: 2 }],
      params: ["delivery_id"], paramType: "p", rowType: "r", limit: 2, total: 2, version: 170007,
    },
    delivery_insert: {
      dialect: "postgresql", cardinality: "execute", kind: "InsertStmt",
      segments: [{ text: "INSERT INTO f03_deliveries (delivery_id, digest, state, updated_ms) VALUES (" }, { param: 1 }, { text: ", " }, { param: 2 }, { text: ", " }, { param: 3 }, { text: ", " }, { param: 4 }, { text: ")" }],
      params: ["delivery_id", "digest", "state", "updated_ms"], paramType: "p", rowType: "r", limit: 0, total: 4, version: 170007,
    },
    gen_insert: {
      dialect: "postgresql", cardinality: "execute", kind: "InsertStmt",
      segments: [{ text: "INSERT INTO f03_gen (payload) VALUES (" }, { param: 1 }, { text: ")" }],
      params: ["payload"], paramType: "p", rowType: "r", limit: 0, total: 1, version: 170007,
    },
    gen_by_payload_one: {
      dialect: "postgresql", cardinality: "one", kind: "SelectStmt",
      segments: [{ text: "SELECT id, payload FROM f03_gen WHERE payload = " }, { param: 1 }, { text: " ORDER BY id LIMIT " }, { param: 2 }],
      params: ["payload"], paramType: "p", rowType: "r", limit: 2, total: 2, version: 170007,
    },
    money_insert: {
      dialect: "postgresql", cardinality: "execute", kind: "InsertStmt",
      segments: [{ text: "INSERT INTO f03_money (id, price_minor, recorded_ms, audit_ms, result) VALUES (" }, { param: 1 }, { text: ", " }, { param: 2 }, { text: ", " }, { param: 3 }, { text: ", " }, { param: 4 }, { text: ", " }, { param: 5 }, { text: ")" }],
      params: ["id", "price_minor", "recorded_ms", "audit_ms", "result"], paramType: "p", rowType: "r", limit: 0, total: 5, version: 170007,
    },
    money_by_id: {
      dialect: "postgresql", cardinality: "one", kind: "SelectStmt",
      segments: [{ text: "SELECT id, price_minor, recorded_ms, audit_ms, result FROM f03_money WHERE id = " }, { param: 1 }, { text: " LIMIT " }, { param: 2 }],
      params: ["id"], paramType: "p", rowType: "r", limit: 2, total: 2, version: 170007,
    },
    money_wide: {
      dialect: "postgresql", cardinality: "one", kind: "SelectStmt",
      segments: [{ text: "SELECT * FROM f03_money WHERE id = " }, { param: 1 }, { text: " LIMIT " }, { param: 2 }],
      params: ["id"], paramType: "p", rowType: "r", limit: 2, total: 2, version: 170007,
    },
    my_gen_insert: {
      dialect: "mysql", cardinality: "execute", kind: "InsertStmt",
      segments: [{ text: "INSERT INTO f03_mygen (payload) VALUES (" }, { param: 1 }, { text: ")" }],
      params: ["payload"], paramType: "p", rowType: "r", limit: 0, total: 1, version: 80011,
    },
    my_gen_by_last_id: {
      dialect: "mysql", cardinality: "one", kind: "SelectStmt",
      segments: [{ text: "SELECT id, payload FROM f03_mygen WHERE id = LAST_INSERT_ID() LIMIT " }, { param: 1 }],
      params: [], paramType: "p", rowType: "r", limit: 1, total: 1, version: 80011,
    },
  },
};
const descriptors = createSQLDescriptors(table);
const env = new Map<string, string>([["F03_PG", pgUrl]]);
if (mysqlUrl) env.set("F03_MY", mysqlUrl);
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
const SOME = "app::option_some";
const NONE = "app::option_none";
const deliveryRow = {
  root: "r",
  fields: [
    { name: "delivery_id", kind: "str" as const },
    { name: "digest", kind: "str" as const },
    { name: "state", kind: "str" as const },
    { name: "updated_ms", kind: "int" as const },
  ],
};
const genRow = {
  root: "r",
  fields: [{ name: "id", kind: "int" as const }, { name: "payload", kind: "str" as const }],
};
const moneyRow = {
  root: "r",
  fields: [
    { name: "id", kind: "int" as const },
    { name: "price_minor", kind: "int" as const },
    { name: "recorded_ms", kind: "int" as const },
    { name: "audit_ms", kind: "option" as const, inner: "int" as const, some: SOME, none: NONE },
    { name: "result", kind: "str" as const },
  ],
};
const narrowRow = {
  root: "r",
  fields: [
    { name: "id", kind: "int" as const },
    { name: "price_minor", kind: "int" as const },
  ],
};
const deliveryKeyPlan: SQLPlan = {
  params: { root: "p", fields: [{ name: "delivery_id", kind: "str" }] },
  rows: deliveryRow, some: SOME, none: NONE,
};
const deliveryInsPlan: SQLPlan = {
  params: {
    root: "p",
    fields: [
      { name: "delivery_id", kind: "str" },
      { name: "digest", kind: "str" },
      { name: "state", kind: "str" },
      { name: "updated_ms", kind: "int" },
    ],
  },
};
const genInsPlan: SQLPlan = {
  params: { root: "p", fields: [{ name: "payload", kind: "str" }] },
};
const genOnePlan: SQLPlan = {
  params: { root: "p", fields: [{ name: "payload", kind: "str" }] }, rows: genRow,
};
const moneyInsPlan: SQLPlan = {
  params: {
    root: "p",
    fields: [
      { name: "id", kind: "int" },
      { name: "price_minor", kind: "int" },
      { name: "recorded_ms", kind: "int" },
      { name: "audit_ms", kind: "option", inner: "int", some: SOME, none: NONE },
      { name: "result", kind: "str" },
    ],
  },
};
const moneyOnePlan: SQLPlan = {
  params: { root: "p", fields: [{ name: "id", kind: "int" }] }, rows: moneyRow,
};
const moneyNarrowPlan: SQLPlan = {
  params: { root: "p", fields: [{ name: "id", kind: "int" }] }, rows: narrowRow,
};
const myInsPlan: SQLPlan = {
  params: { root: "p", fields: [{ name: "payload", kind: "str" }] },
};
const myLastIdPlan: SQLPlan = {
  params: { root: "p", fields: [] }, rows: genRow,
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
const outcomeOf = (c: { kind: string; value?: unknown }) =>
  c.kind === "ok" ? { outcome: "ok" } : { outcome: "domain", ...domainOutcomeOf(c) };

try {
  const report: Record<string, unknown> = {};
  // DDL is operator-owned; Can cannot run it, so setup/teardown stays raw.
  const raw = new Bun.SQL(pgUrl, { adapter: "postgres", bigint: true, max: 8 });
  try {
    const serverVersion = (await raw`SELECT version() AS v` as { v: string }[])[0]!.v;
    report.serverVersion = serverVersion.split(" ").slice(0, 2).join(" ");
    await raw`DROP TABLE IF EXISTS f03_deliveries`;
    await raw`DROP TABLE IF EXISTS f03_gen`;
    await raw`DROP TABLE IF EXISTS f03_money`;
    await raw`CREATE TABLE f03_deliveries (delivery_id TEXT PRIMARY KEY, digest TEXT NOT NULL, state TEXT NOT NULL, updated_ms BIGINT NOT NULL)`;
    await raw`CREATE TABLE f03_gen (id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY, payload TEXT NOT NULL)`;
    await raw`CREATE TABLE f03_money (id BIGINT PRIMARY KEY, price_minor BIGINT NOT NULL, recorded_ms BIGINT NOT NULL, audit_ms BIGINT, result TEXT NOT NULL)`;

    const rootResult = await runOwnedRoot(async () => {
      const pool = value(await pools.open("F03_PG", 8n));
      const deliveryById = descriptors.declareDescriptor("", "delivery_by_id");
      const deliveryInsert = descriptors.declareDescriptor("", "delivery_insert");
      const genInsert = descriptors.declareDescriptor("", "gen_insert");
      const genByPayloadOne = descriptors.declareDescriptor("", "gen_by_payload_one");
      const moneyInsert = descriptors.declareDescriptor("", "money_insert");
      const moneyById = descriptors.declareDescriptor("", "money_by_id");
      const moneyWide = descriptors.declareDescriptor("", "money_wide");

      // Part 1: native RETURNING row behavior (raw oracle for the
      // per-dialect row contract the runtime path will enforce).
      const returning: Record<string, unknown> = {};
      const single = (await raw`INSERT INTO f03_gen (payload) VALUES ('one') RETURNING id` as { id: unknown }[]);
      returning.single_rows = single.length;
      returning.single_id_type = typeof single[0]!.id;
      const multi = (await raw`INSERT INTO f03_gen (payload) VALUES ('m1'), ('m2'), ('m3') RETURNING id` as { id: unknown }[]);
      returning.multi_rows = multi.length;
      const first = (await raw`INSERT INTO f03_deliveries (delivery_id, digest, state, updated_ms) VALUES ('d0', 'h', 'accepted', 1000) ON CONFLICT DO NOTHING RETURNING delivery_id` as unknown[]);
      const second = (await raw`INSERT INTO f03_deliveries (delivery_id, digest, state, updated_ms) VALUES ('d0', 'h', 'accepted', 1000) ON CONFLICT DO NOTHING RETURNING delivery_id` as unknown[]);
      returning.conflict_first_rows = first.length;
      returning.conflict_repeat_rows = second.length;
      try {
        await raw`INSERT INTO f03_deliveries (delivery_id, digest, state, updated_ms) VALUES ('d0', 'h', 'accepted', 1000) RETURNING delivery_id`;
        returning.conflict = "unexpected-ok";
      } catch (e) {
        returning.conflict = {
          rejected: true,
          errno: (e as { errno?: unknown }).errno ?? null,
          constraint: (e as { constraint?: unknown }).constraint ?? null,
        };
      }
      report.returning_native = returning;

      // Part 2: the cardinality/error vocabulary the RETURNING runtime
      // path reuses, pinned through the runtime on live PG.
      const vocab: Record<string, unknown> = {};
      const dup = await pools.execute(
        deliveryInsert, deliveryInsPlan, pool,
        record("p", [["delivery_id", "d0"], ["digest", "h"], ["state", "accepted"], ["updated_ms", 1000n]]),
      );
      vocab.duplicate_insert = outcomeOf(dup);
      const missing = await pools.queryOne(
        moneyById, moneyOnePlan, pool, record("p", [["id", 424242n]]),
      );
      vocab.absent_one = outcomeOf(missing);
      value(await pools.execute(genInsert, genInsPlan, pool, record("p", [["payload", "pair"]])));
      value(await pools.execute(genInsert, genInsPlan, pool, record("p", [["payload", "pair"]])));
      const pair = await pools.queryOne(
        genByPayloadOne, genOnePlan, pool, record("p", [["payload", "pair"]]),
      );
      vocab.two_row_one = outcomeOf(pair);
      value(await pools.execute(
        moneyInsert, moneyInsPlan, pool,
        record("p", [["id", 7n], ["price_minor", 1099n], ["recorded_ms", 1727222400000n], ["audit_ms", record(NONE, [])], ["result", "{}"]]),
      ));
      const wide = await pools.queryOne(
        moneyWide, moneyNarrowPlan, pool, record("p", [["id", 7n]]),
      );
      vocab.wide_decode = outcomeOf(wide);
      report.error_vocab = vocab;

      // Part 3: aborted-transaction behavior and replay on live PG.
      const aborted: Record<string, unknown> = {};
      const poisoned = await transactions.withTransaction(pool, async (h: unknown) => {
        const bad = await transactions.execute(
          deliveryInsert, deliveryInsPlan, h,
          record("p", [["delivery_id", "d0"], ["digest", "h"], ["state", "accepted"], ["updated_ms", 1000n]]),
        );
        aborted.first_outcome = outcomeOf(bad);
        const followup = await transactions.queryOne(
          moneyById, moneyOnePlan, h, record("p", [["id", 7n]]),
        );
        aborted.followup_outcome = outcomeOf(followup);
        return success(record(COMMIT, [["value", 1n]]));
      }, leaves);
      aborted.txn_outcome = outcomeOf(poisoned);
      const replayed = await transactions.withTransaction(pool, async (h: unknown) => {
        value(await transactions.execute(
          deliveryInsert, deliveryInsPlan, h,
          record("p", [["delivery_id", "d1"], ["digest", "h"], ["state", "accepted"], ["updated_ms", 1000n]]),
        ));
        const row = value(await transactions.queryOptional(
          deliveryById, deliveryKeyPlan, h, record("p", [["delivery_id", "d1"]]),
        ));
        const some = recordIdentity(row) === SOME ? dataProperty(row as never, "value") : undefined;
        return success(record(COMMIT, [["value", some === undefined ? 0n : 1n]]));
      }, leaves);
      aborted.replay_ok = replayed.kind === "ok" && (value(replayed) as bigint) === 1n;
      report.aborted_txn = aborted;

      // Part 4: lookup-first decide under concurrency — the PG conflict
      // recipe. No statement ever fails, so no 25P02 can appear. The pool
      // is warm by now (parts 1-3 ran serial runtime operations first);
      // a first-burst race on cold pools is a known E-owned issue (see
      // the MySQL leg note), so this leg must keep running after them.
      const N = 8;
      let saw25P02 = false;
      const note25P02 = (o: { outcome: string; payload?: unknown }) => {
        if ((o.payload as { code?: unknown } | undefined)?.code === "25P02") saw25P02 = true;
      };
      const attempts = await Promise.all(Array.from({ length: N }, (_, i) =>
        transactions.withTransaction(pool, async (h: unknown) => {
          const seen = value(await transactions.queryOptional(
            deliveryById, deliveryKeyPlan, h, record("p", [["delivery_id", "race"]])),
          ) as unknown;
          if (recordIdentity(seen) === SOME) {
            const row = dataProperty(seen as never, "value") as never;
            const state = dataProperty(row, "digest") === "race-digest" ? "duplicate" : "conflict";
            return success(record(ROLLBACK, [["value", record("v", [["state", state]])]]));
          }
          const ins = await transactions.execute(
            deliveryInsert, deliveryInsPlan, h,
            record("p", [["delivery_id", "race"], ["digest", "race-digest"], ["state", "accepted"], ["updated_ms", BigInt(1000 + i)]]),
          );
          if (ins.kind !== "ok") {
            const o = outcomeOf(ins);
            note25P02(o as { outcome: string; payload?: unknown });
            return success(record(ROLLBACK, [["value", record("v", [["state", "lost-race"]])]]));
          }
          return success(record(COMMIT, [["value", record("v", [["state", "accepted"]])]]));
        }, leaves)));
      let accepted = 0;
      let settled = 0;
      for (const a of attempts) {
        if (a.kind !== "ok") continue;
        settled++;
        if ((dataProperty(value(a) as never, "state") as string) === "accepted") accepted++;
      }
      const winner = value(await pools.queryOptional(
        deliveryById, deliveryKeyPlan, pool, record("p", [["delivery_id", "race"]]),
      )) as unknown;
      const winnerDigest = recordIdentity(winner) === SOME
        ? dataProperty(dataProperty(winner as never, "value") as never, "digest")
        : undefined;
      report.lookup_first = {
        writers: N, settled, accepted,
        winner_digest: winnerDigest ?? null,
        saw_25P02: saw25P02,
      };

      // Part 5: blessed encodings roundtrip through the runtime.
      const doc = { total_minor: 1099, lines: [{ id: "l1", qty: 2 }], note: "café ☃ \"quoted\"" };
      const wire = JSON.stringify(doc);
      value(await pools.execute(
        moneyInsert, moneyInsPlan, pool,
        record("p", [["id", 8n], ["price_minor", 1099n], ["recorded_ms", 1727222400000n], ["audit_ms", record(SOME, [["value", 1727222500000n]])], ["result", wire]]),
      ));
      const enc: Record<string, unknown> = {};
      const nullRow = value(await pools.queryOne(
        moneyById, moneyOnePlan, pool, record("p", [["id", 7n]]),
      )) as never;
      enc.null_audit_is_none = recordIdentity(dataProperty(nullRow, "audit_ms")) === NONE;
      enc.minor_unit = Number(dataProperty(nullRow, "price_minor") as bigint) === 1099;
      enc.ms_epoch = Number(dataProperty(nullRow, "recorded_ms") as bigint) === 1727222400000;
      const fullRow = value(await pools.queryOne(
        moneyById, moneyOnePlan, pool, record("p", [["id", 8n]]),
      )) as never;
      const audit = dataProperty(fullRow, "audit_ms") as unknown;
      enc.set_audit = recordIdentity(audit) === SOME &&
        Number(dataProperty(audit as never, "value") as bigint) === 1727222500000;
      enc.json_codec = JSON.stringify(JSON.parse(dataProperty(fullRow, "result") as string)) === JSON.stringify(doc);
      value(await pools.execute(
        moneyInsert, moneyInsPlan, pool,
        record("p", [["id", 9n], ["price_minor", 9223372036854775807n], ["recorded_ms", -9223372036854775808n], ["audit_ms", record(NONE, [])], ["result", "{}"]]),
      ));
      const edgeRow = value(await pools.queryOne(
        moneyById, moneyOnePlan, pool, record("p", [["id", 9n]]),
      )) as never;
      enc.int64_max = (dataProperty(edgeRow, "price_minor") as bigint) === 9223372036854775807n;
      enc.int64_min = (dataProperty(edgeRow, "recorded_ms") as bigint) === -9223372036854775808n;
      report.encodings = enc;

      // Part 6: MySQL — RETURNING rejection plus the LAST_INSERT_ID
      // mapping through the runtime transaction path.
      if (!mysqlUrl) {
        report.mysql = "skipped:no-mysql-url";
      } else {
        const my = new Bun.SQL(mysqlUrl, { adapter: "mysql", max: 4, tls: true });
        try {
          const mv = (await my`SELECT VERSION() AS v` as { v: string }[])[0]!.v;
          await my`CREATE TABLE IF NOT EXISTS f03_mygen (id BIGINT AUTO_INCREMENT PRIMARY KEY, payload TEXT)`;
          let insertReturning: unknown;
          try {
            await my`INSERT INTO f03_mygen (payload) VALUES ('x') RETURNING id`;
            insertReturning = "admitted";
          } catch (e) {
            insertReturning = {
              rejected: true,
              code: (e as { code?: unknown }).code ?? null,
              errno: (e as { errno?: unknown }).errno ?? null,
            };
          }
          const mypool = value(await pools.mysqlOpen("F03_MY", 8n));
          const myIns = descriptors.declareDescriptor("", "my_gen_insert");
          const myLastId = descriptors.declareDescriptor("", "my_gen_by_last_id");
          // Pool warmup: a concurrent burst on a cold pool flakes with
          // `resource_state` before callbacks even enter (observed on PG
          // and MySQL alike; raw Bun bursts are sound, so the race is in
          // the runtime wrapper — E-owned, handed off). Serial warmup
          // establishes the pool first; the mapping proof below (every
          // concurrent writer refetches exactly its own row) is
          // unaffected by it.
          for (let w = 0; w < 8; w++) {
            value(await transactions.withTransaction(mypool, async (h: unknown) => {
              value(await transactions.execute(myIns, myInsPlan, h, record("p", [["payload", `warm-${w}`]])));
              return success(record(COMMIT, [["value", 1n]]));
            }, leaves));
          }
          const K = 8;
          const mine = await Promise.all(Array.from({ length: K }, (_, i) =>
            transactions.withTransaction(mypool, async (h: unknown) => {
              const key = `mine-${i}`;
              value(await transactions.execute(myIns, myInsPlan, h, record("p", [["payload", key]])));
              const row = value(await transactions.queryOne(myLastId, myLastIdPlan, h, record("p", []))) as never;
              const own = dataProperty(row, "payload") === key ? 1n : 0n;
              return success(record(COMMIT, [["value", record("v", [["own", own], ["id", dataProperty(row, "id")]])]]));
            }, leaves)));
          const ids = new Set<string>();
          let allOwn = true;
          for (const m of mine) {
            if (m.kind !== "ok") { allOwn = false; continue; }
            const v = value(m) as never;
            if ((dataProperty(v, "own") as bigint) !== 1n) allOwn = false;
            ids.add(String(dataProperty(v, "id")));
          }
          report.mysql = {
            serverVersion: mv, insert_returning: insertReturning,
            mapping: { writers: K, all_own_row: allOwn, distinct_ids: ids.size },
          };
          value(await pools.close(mypool, 5000n));
          await my`DROP TABLE IF EXISTS f03_mygen`;
        } finally {
          await my.close();
        }
      }

      // Part 7: SQLite native RETURNING (runtime execution awaits E).
      const lite = new Bun.SQL({ adapter: "sqlite", filename: ":memory:", safeIntegers: true });
      try {
        const lv = (await lite`select sqlite_version() as v` as { v: string }[])[0]!.v;
        await lite`CREATE TABLE f03_lite (id INTEGER PRIMARY KEY AUTOINCREMENT, payload TEXT)`;
        const one = (await lite`INSERT INTO f03_lite (payload) VALUES ('a') RETURNING id` as unknown[]);
        const three = (await lite`INSERT INTO f03_lite (payload) VALUES ('b'), ('c'), ('d') RETURNING id` as unknown[]);
        report.sqlite = { version: lv, single_rows: one.length, multi_rows: three.length };
      } finally {
        await lite.close();
      }

      value(await pools.close(pool, 5000n));
      return success(undefined);
    });
    if (rootResult.completion.kind !== "ok" || rootResult.cleanupFailed) {
      throw new Error("owned run failed");
    }
    await raw`DROP TABLE IF EXISTS f03_deliveries`;
    await raw`DROP TABLE IF EXISTS f03_gen`;
    await raw`DROP TABLE IF EXISTS f03_money`;
    await raw.close();
    console.log(JSON.stringify(report, (_, v) => (typeof v === "bigint" ? Number(v) : v)));
  } catch (error) {
    try {
      await raw`DROP TABLE IF EXISTS f03_deliveries`;
      await raw`DROP TABLE IF EXISTS f03_gen`;
      await raw`DROP TABLE IF EXISTS f03_money`;
    } catch { /* best effort */ }
    await raw.close().catch(() => undefined);
    throw error;
  }
} catch (error) {
  console.log(JSON.stringify({ error: error instanceof Error ? error.message : String(error) }));
  process.exitCode = 1;
}
