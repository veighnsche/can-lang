// F17: a concurrent withTransaction burst on a cold pool must not fail
// with resource_state before callbacks enter. Each round opens a fresh
// pool and bursts immediately with no warmup; table setup runs on a
// separate pool so the burst pool stays cold. PG/MySQL legs need
// CAN_TEST_POSTGRES_URL / CAN_TEST_MYSQL_URL (provision template with
// <db> selecting can_f8_run1); otherwise they skip. URLs and passwords
// are never logged. (SQLite file pools serialize writers by design, so
// the burst leg is PG/MySQL only.)
import { expect, test } from "bun:test";
import { createHash } from "node:crypto";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, type FailureShape } from "../domain.ts";
import { success, value, type Completion } from "../completion.ts";
import { dataProperty, record } from "../data.ts";
import { runOwnedRoot } from "../owner.ts";
import { createSQLDescriptors, type SQLDescriptorEntry } from "../platform/sql/descriptor.ts";
import { createSQLPools } from "../platform/sql/pool.ts";
import { createSQLTransactions } from "../platform/sql/transaction.ts";
import type { SQLPlan } from "../platform/sql/values.ts";

function dbUrl(env: string, scheme: string, fallbackDb: string): string | undefined {
  const template = process.env[env];
  if (template === undefined || template === "") return undefined;
  if (!template.startsWith(scheme)) throw new Error(`refusing non-${scheme} test URL`);
  return template.includes("<db>") ? template.replace("<db>", fallbackDb) : template;
}
const PG_URL = dbUrl("CAN_TEST_POSTGRES_URL", "postgres", "can_f8_run1");
const MYSQL_URL = dbUrl("CAN_TEST_MYSQL_URL", "mysql", "can_f8_run1");

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

function execEntry(
  dialect: SQLDescriptorEntry["dialect"],
  kind: string,
  text: string,
  version: number,
): SQLDescriptorEntry {
  return {
    dialect,
    cardinality: "execute",
    kind,
    segments: [{ text }],
    params: [],
    paramType: "p",
    rowType: "r",
    limit: 0,
    total: 0,
    version,
  };
}
function insertEntry(
  dialect: SQLDescriptorEntry["dialect"],
  kind: string,
  head: string,
  version: number,
): SQLDescriptorEntry {
  return {
    dialect,
    cardinality: "execute",
    kind,
    segments: [{ text: head }, { param: 1 }, { text: ")" }],
    params: ["payload"],
    paramType: "p",
    rowType: "r",
    limit: 0,
    total: 1,
    version,
  };
}
function allEntry(
  dialect: SQLDescriptorEntry["dialect"],
  kind: string,
  text: string,
  version: number,
): SQLDescriptorEntry {
  return {
    dialect,
    cardinality: "many",
    kind,
    segments: [{ text: `${text} LIMIT ` }, { param: 1 }],
    params: [],
    paramType: "p",
    rowType: "r",
    limit: 1,
    total: 1,
    version,
  };
}
const table: Record<string, Record<string, SQLDescriptorEntry>> = {
  "": {
    f17_setup_pg: execEntry(
      "postgresql",
      "CreateStmt",
      "CREATE TABLE IF NOT EXISTS f17_burst (id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY, payload TEXT NOT NULL)",
      170007,
    ),
    f17_clear_pg: execEntry("postgresql", "DeleteStmt", "DELETE FROM f17_burst", 170007),
    f17_insert_pg: insertEntry(
      "postgresql",
      "InsertStmt",
      "INSERT INTO f17_burst (payload) VALUES (",
      170007,
    ),
    f17_all_pg: allEntry("postgresql", "SelectStmt", "SELECT payload FROM f17_burst", 170007),
    f17_setup_mysql: execEntry(
      "mysql",
      "create_table_statement",
      "CREATE TABLE IF NOT EXISTS f17_burst (id BIGINT AUTO_INCREMENT PRIMARY KEY, payload TEXT NOT NULL)",
      80011,
    ),
    f17_clear_mysql: execEntry("mysql", "delete_statement", "DELETE FROM f17_burst", 80011),
    f17_insert_mysql: insertEntry(
      "mysql",
      "insert_statement",
      "INSERT INTO f17_burst (payload) VALUES (",
      80011,
    ),
    f17_all_mysql: allEntry("mysql", "SelectStmt", "SELECT payload FROM f17_burst", 80011),
  },
};
const descriptors = createSQLDescriptors(table);
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
  (name: string) => process.env[name],
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
const COMMIT = "app::commit";
const ROLLBACK = "app::rollback";
const leaves = { commit: COMMIT, rollback: ROLLBACK };
const emptyParams: SQLPlan = { params: { root: "app::empty", fields: [] } };
const insertParams: SQLPlan = {
  params: { root: "app::burst_parameters", fields: [{ name: "payload", kind: "str" }] },
};
const allParams: SQLPlan = {
  params: { root: "app::empty", fields: [] },
  rows: { root: "app::burst_row", fields: [{ name: "payload", kind: "str" }] },
};
const WIDTH = 8;
const ROUNDS = 3;

async function coldBurst(
  openPool: () => Promise<Completion<unknown>>,
  setup: string,
  clear: string,
  insert: string,
  all: string,
): Promise<void> {
  const setupEntry = descriptors.declareDescriptor("", setup);
  const clearEntry = descriptors.declareDescriptor("", clear);
  const insertEntryDesc = descriptors.declareDescriptor("", insert);
  const allEntryDesc = descriptors.declareDescriptor("", all);
  for (let round = 0; round < ROUNDS; round++) {
    const setupPool = value(await openPool());
    value(await pools.execute(setupEntry, emptyParams, setupPool, record("app::empty", [])));
    value(await pools.execute(clearEntry, emptyParams, setupPool, record("app::empty", [])));
    value(await pools.close(setupPool, 1000n));
    // The burst pool is fresh: no warmup of any kind before the burst.
    const pool = value(await openPool());
    const tags = Array.from({ length: WIDTH }, (_, i) => `r${round}-w${i}`);
    const outcomes = await Promise.all(
      tags.map((tag) =>
        transactions.withTransaction(
          pool,
          async (handle: unknown) => {
            value(
              await transactions.execute(
                insertEntryDesc,
                insertParams,
                handle,
                record("app::burst_parameters", [["payload", tag]]),
              ),
            );
            return success(record(COMMIT, [["value", tag]]));
          },
          leaves,
        ),
      ),
    );
    for (let i = 0; i < WIDTH; i++) {
      expect(outcomes[i]!.kind).toBe("ok");
      if (outcomes[i]!.kind === "ok") expect(value(outcomes[i]!)).toBe(tags[i]);
    }
    const rows = value(
      await pools.queryRows(allEntryDesc, allParams, pool, record("app::empty", []), 100n),
    ) as unknown[];
    const byPayload = (a: unknown, b: unknown) =>
      (a as string) < (b as string) ? -1 : (a as string) > (b as string) ? 1 : 0;
    expect(rows.map((row) => dataProperty(row as never, "payload")).sort(byPayload)).toEqual(
      [...tags].sort(byPayload),
    );
    value(await pools.close(pool, 1000n));
  }
}

const pgTest = PG_URL === undefined ? test.skipIf(true) : test;
pgTest("cold pool transaction burst commits on postgres", async () => {
  process.env["F17_PG_URL"] = PG_URL!;
  try {
    await owned(async () => {
      await coldBurst(
        () => pools.open("F17_PG_URL", 8n),
        "f17_setup_pg",
        "f17_clear_pg",
        "f17_insert_pg",
        "f17_all_pg",
      );
    });
  } finally {
    delete process.env["F17_PG_URL"];
  }
});

const mysqlTest = MYSQL_URL === undefined ? test.skipIf(true) : test;
mysqlTest("cold pool transaction burst commits on mysql", async () => {
  process.env["F17_MYSQL_URL"] = MYSQL_URL!;
  try {
    await owned(async () => {
      await coldBurst(
        () => pools.mysqlOpen("F17_MYSQL_URL", 8n),
        "f17_setup_mysql",
        "f17_clear_mysql",
        "f17_insert_mysql",
        "f17_all_mysql",
      );
    });
  } finally {
    delete process.env["F17_MYSQL_URL"];
  }
});
