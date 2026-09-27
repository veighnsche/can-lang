// F8 RETURNING live legs: admitted INSERT ... RETURNING descriptors
// execute end-to-end through the runtime on SQLite (always) and
// PostgreSQL (env-gated), with the MySQL LAST_INSERT_ID mapping legs
// (env-gated). Live legs need CAN_TEST_POSTGRES_URL /
// CAN_TEST_MYSQL_URL (provision template with <db> selecting
// can_f8_run1); otherwise they skip. URLs and passwords are never
// logged. Contract: f03-returning-contract.md.
import { expect, test } from "bun:test";
import { createHash } from "node:crypto";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, domainFailureDiagnostics, type FailureShape } from "../domain.ts";
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

function returningEntry(
  dialect: SQLDescriptorEntry["dialect"],
  kind: string,
  statement: string,
  version: number,
): SQLDescriptorEntry {
  const marker = statement.includes("$1") ? "$1" : "?1";
  const parts = statement.split(marker);
  const segments: SQLDescriptorEntry["segments"] = [{ text: parts[0]! }];
  for (const tail of parts.slice(1)) {
    segments.push({ param: 1 }, { text: tail });
  }
  return {
    dialect,
    cardinality: "one",
    kind,
    segments,
    params: ["payload"],
    paramType: "p",
    rowType: "r",
    limit: 0,
    total: 1,
    version,
  };
}
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
const table: Record<string, Record<string, SQLDescriptorEntry>> = {
  "": {
    f8_setup_pg: execEntry(
      "postgresql",
      "CreateStmt",
      "CREATE TABLE IF NOT EXISTS f8_gen (id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY, payload TEXT NOT NULL, tag TEXT NOT NULL UNIQUE)",
      170007,
    ),
    f8_insert_pg: returningEntry(
      "postgresql",
      "InsertStmt",
      "INSERT INTO f8_gen (payload, tag) VALUES ($1, $1) RETURNING id",
      170007,
    ),
    f8_skip_pg: returningEntry(
      "postgresql",
      "InsertStmt",
      "INSERT INTO f8_gen (payload, tag) VALUES ($1, $1) ON CONFLICT (tag) DO NOTHING RETURNING id",
      170007,
    ),
    f8_clear_pg: execEntry("postgresql", "DeleteStmt", "DELETE FROM f8_gen", 170007),
    f8_skip_lite: returningEntry(
      "sqlite",
      "insert_statement",
      "INSERT INTO f8_gen (payload, tag) VALUES (?1, ?1) ON CONFLICT (tag) DO NOTHING RETURNING id",
      15,
    ),
    f8_setup_lite: execEntry(
      "sqlite",
      "create_table_statement",
      "CREATE TABLE IF NOT EXISTS f8_gen (id INTEGER PRIMARY KEY AUTOINCREMENT, payload TEXT NOT NULL, tag TEXT NOT NULL UNIQUE)",
      15,
    ),
    f8_insert_lite: returningEntry(
      "sqlite",
      "insert_statement",
      "INSERT INTO f8_gen (payload, tag) VALUES (?1, ?1) RETURNING id",
      15,
    ),
    f8_insert3_lite: {
      dialect: "sqlite",
      cardinality: "one",
      kind: "insert_statement",
      segments: [
        { text: "INSERT INTO f8_gen (payload, tag) VALUES (" },
        { param: 1 },
        { text: ", " },
        { param: 1 },
        { text: "), (" },
        { param: 2 },
        { text: ", " },
        { param: 2 },
        { text: "), (" },
        { param: 3 },
        { text: ", " },
        { param: 3 },
        { text: ") RETURNING id" },
      ],
      params: ["a", "b", "c"],
      paramType: "p",
      rowType: "r",
      limit: 0,
      total: 3,
      version: 15,
    },
    f8_setup_mysql: execEntry(
      "mysql",
      "create_table_statement",
      "CREATE TABLE IF NOT EXISTS f8_gen (id BIGINT AUTO_INCREMENT PRIMARY KEY, payload TEXT NOT NULL)",
      80011,
    ),
    f8_insert_mysql: {
      dialect: "mysql",
      cardinality: "execute",
      kind: "insert_statement",
      segments: [{ text: "INSERT INTO f8_gen (payload) VALUES (" }, { param: 1 }, { text: ")" }],
      params: ["payload"],
      paramType: "p",
      rowType: "r",
      limit: 0,
      total: 1,
      version: 80011,
    },
    f8_refetch_mysql: {
      dialect: "mysql",
      cardinality: "one",
      kind: "SelectStmt",
      segments: [
        { text: "SELECT id, payload FROM f8_gen WHERE id = LAST_INSERT_ID() LIMIT " },
        { param: 1 },
      ],
      params: [],
      paramType: "p",
      rowType: "r",
      limit: 1,
      total: 1,
      version: 80011,
    },
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
const emptyParams: SQLPlan = { params: { root: "app::empty", fields: [] } };
const genParams: SQLPlan = {
  params: { root: "app::gen_parameters", fields: [{ name: "payload", kind: "str" }] },
  rows: { root: "app::gen_row", fields: [{ name: "id", kind: "int" }] },
};

test("sqlite: single-row RETURNING yields the generated id", async () => {
  await owned(async () => {
    const token = value(await pools.sqliteOpenMemory());
    const setup = descriptors.declareDescriptor("", "f8_setup_lite");
    const insert = descriptors.declareDescriptor("", "f8_insert_lite");
    value(await pools.execute(setup, emptyParams, token, record("app::empty", [])));
    const row = value(
      await pools.queryOne(
        insert,
        genParams,
        token,
        record("app::gen_parameters", [["payload", "lite-1"]]),
      ),
    );
    expect(dataProperty(row, "id")).toBe(1n);
    value(await pools.close(token, 1000n));
  });
});

test("sqlite: multi-row RETURNING enforces exactly-one", async () => {
  await owned(async () => {
    const token = value(await pools.sqliteOpenMemory());
    const setup = descriptors.declareDescriptor("", "f8_setup_lite");
    value(await pools.execute(setup, emptyParams, token, record("app::empty", [])));
    const multi = descriptors.declareDescriptor("", "f8_insert3_lite");
    const plan: SQLPlan = {
      params: {
        root: "app::gen3_parameters",
        fields: [
          { name: "a", kind: "str" },
          { name: "b", kind: "str" },
          { name: "c", kind: "str" },
        ],
      },
      rows: { root: "app::gen_row", fields: [{ name: "id", kind: "int" }] },
    };
    expect(
      domainOutcome(
        await pools.queryOne(
          multi,
          plan,
          token,
          record("app::gen3_parameters", [
            ["a", "m-1"],
            ["b", "m-2"],
            ["c", "m-3"],
          ]),
        ),
        "sql::row_count",
      ),
    ).toEqual({ query: "f8_insert3_lite", actual: 2n });
    value(await pools.close(token, 1000n));
  });
});

test("sqlite: conflict-skip yields row_missing, duplicate yields constraint_failed", async () => {
  await owned(async () => {
    const token = value(await pools.sqliteOpenMemory());
    const setup = descriptors.declareDescriptor("", "f8_setup_lite");
    const insert = descriptors.declareDescriptor("", "f8_insert_lite");
    const skip = descriptors.declareDescriptor("", "f8_skip_lite");
    value(await pools.execute(setup, emptyParams, token, record("app::empty", [])));
    const first = value(
      await pools.queryOne(
        insert,
        genParams,
        token,
        record("app::gen_parameters", [["payload", "dup-1"]]),
      ),
    );
    expect(dataProperty(first, "id")).toBe(1n);
    expect(
      domainOutcome(
        await pools.queryOne(
          skip,
          genParams,
          token,
          record("app::gen_parameters", [["payload", "dup-1"]]),
        ),
        "sql::row_missing",
      ),
    ).toEqual({ query: "f8_skip_lite" });
    const outcome = domainOutcome(
      await pools.queryOne(
        insert,
        genParams,
        token,
        record("app::gen_parameters", [["payload", "dup-1"]]),
      ),
      "sql::constraint_failed",
    );
    expect(typeof outcome["constraint"]).toBe("string");
    expect((outcome["constraint"] as string).length).toBeGreaterThan(0);
    value(await pools.close(token, 1000n));
  });
});

test("sqlite: RETURNING inside a transaction commits", async () => {
  await owned(async () => {
    const token = value(await pools.sqliteOpenMemory());
    const setup = descriptors.declareDescriptor("", "f8_setup_lite");
    const insert = descriptors.declareDescriptor("", "f8_insert_lite");
    value(await pools.execute(setup, emptyParams, token, record("app::empty", [])));
    const committed = value(
      await transactions.withTransaction(
        token,
        async (handle: unknown) => {
          const row = value(
            await transactions.queryOne(
              insert,
              genParams,
              handle,
              record("app::gen_parameters", [["payload", "tx-1"]]),
            ),
          );
          return success(record(COMMIT, [["value", dataProperty(row, "id")]]));
        },
        leaves,
      ),
    );
    expect(committed).toBe(1n);
    value(await pools.close(token, 1000n));
  });
});

const pgTest = PG_URL === undefined ? test.skipIf(true) : test;
pgTest("pg: RETURNING single-row, skip, conflict, and transaction twin", async () => {
  process.env["F8_PG_URL"] = PG_URL!;
  try {
    await owned(async () => {
      const token = value(await pools.open("F8_PG_URL", 5n));
      const setup = descriptors.declareDescriptor("", "f8_setup_pg");
      const clear = descriptors.declareDescriptor("", "f8_clear_pg");
      const insert = descriptors.declareDescriptor("", "f8_insert_pg");
      const skip = descriptors.declareDescriptor("", "f8_skip_pg");
      value(await pools.execute(setup, emptyParams, token, record("app::empty", [])));
      value(await pools.execute(clear, emptyParams, token, record("app::empty", [])));
      const row = value(
        await pools.queryOne(
          insert,
          genParams,
          token,
          record("app::gen_parameters", [["payload", "pg-1"]]),
        ),
      );
      expect(typeof dataProperty(row, "id")).toBe("bigint");
      expect(
        domainOutcome(
          await pools.queryOne(
            skip,
            genParams,
            token,
            record("app::gen_parameters", [["payload", "pg-1"]]),
          ),
          "sql::row_missing",
        ),
      ).toEqual({ query: "f8_skip_pg" });
      expect(
        domainOutcome(
          await pools.queryOne(
            insert,
            genParams,
            token,
            record("app::gen_parameters", [["payload", "pg-1"]]),
          ),
          "sql::constraint_failed",
        ),
      ).toEqual({ constraint: "f8_gen_tag_key" });
      const committed = value(
        await transactions.withTransaction(
          token,
          async (handle: unknown) => {
            const txRow = value(
              await transactions.queryOne(
                insert,
                genParams,
                handle,
                record("app::gen_parameters", [["payload", "pg-tx"]]),
              ),
            );
            return success(record(COMMIT, [["value", dataProperty(txRow, "id")]]));
          },
          leaves,
        ),
      );
      expect(typeof committed).toBe("bigint");
      value(await pools.close(token, 1000n));
    });
  } finally {
    delete process.env["F8_PG_URL"];
  }
});

const mysqlTest = MYSQL_URL === undefined ? test.skipIf(true) : test;
mysqlTest("mysql: mapping refetches exactly the own row, concurrently", async () => {
  process.env["F8_MYSQL_URL"] = MYSQL_URL!;
  try {
    await owned(async () => {
      const token = value(await pools.mysqlOpen("F8_MYSQL_URL", 8n));
      const setup = descriptors.declareDescriptor("", "f8_setup_mysql");
      const insert = descriptors.declareDescriptor("", "f8_insert_mysql");
      const refetch = descriptors.declareDescriptor("", "f8_refetch_mysql");
      value(await pools.execute(setup, emptyParams, token, record("app::empty", [])));
      const refetchPlan: SQLPlan = {
        params: { root: "app::empty", fields: [] },
        rows: {
          root: "app::gen_pair",
          fields: [
            { name: "id", kind: "int" },
            { name: "payload", kind: "str" },
          ],
        },
      };
      const insertPlan: SQLPlan = {
        params: { root: "app::gen_parameters", fields: [{ name: "payload", kind: "str" }] },
      };
      const writers = ["my-1", "my-2", "my-3", "my-4"].map((tag) =>
        transactions.withTransaction(
          token,
          async (handle: unknown) => {
            value(
              await transactions.execute(
                insert,
                insertPlan,
                handle,
                record("app::gen_parameters", [["payload", tag]]),
              ),
            );
            const row = value(
              await transactions.queryOne(refetch, refetchPlan, handle, record("app::empty", [])),
            );
            return success(
              record(COMMIT, [
                ["value", dataProperty(row, "id")],
                ["tag", dataProperty(row, "payload")],
              ]),
            );
          },
          leaves,
        ),
      );
      const done = await Promise.all(writers);
      const ids = done.map((d) => value(d) as bigint);
      expect(new Set(ids).size).toBe(4);
      value(await pools.close(token, 1000n));
    });
  } finally {
    delete process.env["F8_MYSQL_URL"];
  }
});
