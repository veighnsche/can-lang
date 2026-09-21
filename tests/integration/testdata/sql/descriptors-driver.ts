// I37 test-only live descriptor harness. This driver is executed by
// tests/integration/sql_descriptors_test.go against a disposable PostgreSQL
// named by CAN_TEST_POSTGRES_URL. It imports the *compiled* program state,
// builds every query through the compiler-owned $canSQL.template() path, and
// executes the resulting {strings, values} pairs with the pinned Bun.SQL tag.
// It never concatenates values into SQL and never adds a product query API:
// setup/teardown statements come from the fixed seed file as static templates
// with zero interpolations, and every exercised query flows through a checked
// manifest descriptor. Not a raw-driver escape: it cannot run without a
// compiler-built descriptor table.
import { SQL } from "bun";

interface Check {
  name: string;
  ok: boolean;
  detail: string;
}

interface Template {
  strings: TemplateStringsArray;
  values: readonly unknown[];
}

interface DescriptorTable {
  declareDescriptor(owner: string, name: string): unknown;
  template(descriptor: unknown, values: readonly unknown[]): Template;
}

function staticTemplate(text: string): TemplateStringsArray {
  const parts = [text];
  return Object.freeze(Object.assign(parts.slice(), { raw: Object.freeze(parts.slice()) })) as unknown as TemplateStringsArray;
}

function fail(checks: Check[], name: string, detail: string): never {
  checks.push({ name, ok: false, detail });
  throw new Error(`${name}: ${detail}`);
}

function pass(checks: Check[], name: string, detail: string): void {
  checks.push({ name, ok: true, detail });
}

function names(rows: Array<{ display_name?: unknown }>): string[] {
  return rows.map((row) => String(row.display_name));
}

function assertNames(checks: Check[], name: string, rows: Array<{ display_name?: unknown }>, want: string[]): void {
  const got = names(rows);
  if (got.length !== want.length || got.some((value, index) => value !== want[index])) {
    fail(checks, name, `want [${want.join(", ")}] got [${got.join(", ")}]`);
  }
  pass(checks, name, `[${got.join(", ")}]`);
}

const checks: Check[] = [];
const report = (extra: Record<string, unknown>) => {
  console.log(JSON.stringify({ checks, ...extra }));
};

try {
  const buildDir = process.argv[2];
  const seedPath = process.argv[3];
  const url = process.env.CAN_TEST_POSTGRES_URL;
  if (typeof buildDir !== "string" || buildDir === "") throw new Error("missing build directory argument");
  if (typeof seedPath !== "string" || seedPath === "") throw new Error("missing seed path argument");
  if (typeof url !== "string" || url === "") throw new Error("set CAN_TEST_POSTGRES_URL to the disposable test database");
  if (url.includes(" ") || !url.startsWith("postgres")) throw new Error("refusing non-postgres test URL");

  const state = (await import(`${new URL(`file://${buildDir}/program/state.ts`).href}`)) as {
    $canInitialize: () => void;
    $canSQL: DescriptorTable;
  };
  state.$canInitialize();
  const sqlHandles = state.$canSQL;
  const sql = new SQL(url);
  try {
    const versionRows = (await sql(staticTemplate("SELECT version()"))) as Array<{ version: string }>;
    const serverVersion = String(versionRows[0]?.version ?? "");
    if (!serverVersion.startsWith("PostgreSQL 17.")) {
      throw new Error(`unexpected test database version: ${serverVersion.slice(0, 80)}`);
    }

    const seed = await Bun.file(seedPath).text();
    const statements = seed
      .split("\n")
      .filter((line) => !line.trimStart().startsWith("--"))
      .join("\n")
      .split(";")
      .map((part) => part.trim())
      .filter((part) => part !== "");
    if (statements.length !== 4) throw new Error(`seed file must hold 4 statements, saw ${statements.length}`);
    const [create, truncate, insert, drop] = statements as [string, string, string, string];
    if (!create.startsWith("CREATE TABLE") || !truncate.startsWith("TRUNCATE") || !insert.startsWith("INSERT INTO") || !drop.startsWith("DROP TABLE")) {
      throw new Error("seed file shape changed; refusing to run setup/teardown");
    }
    try {
      for (const setup of [create, truncate, insert]) {
        await sql(staticTemplate(setup));
      }
      pass(checks, "setup seed", "6 rows");

      const search = sqlHandles.declareDescriptor("", "search_accounts");
      const repeat = sqlHandles.declareDescriptor("", "repeat_search");
      const byId = sqlHandles.declareDescriptor("", "account_by_id");
      const add = sqlHandles.declareDescriptor("", "add_account");
      const commented = sqlHandles.declareDescriptor("", "commented_search");

      const run = async (descriptor: unknown, values: readonly unknown[]) => {
        const tpl = sqlHandles.template(descriptor, values);
        return (await sql(tpl.strings, ...tpl.values)) as Array<Record<string, unknown>>;
      };

      // Structural expansion first: repeated $1 must repeat the same prepared
      // value, and the comment's $9 must not become a binding site.
      const repeatTpl = sqlHandles.template(repeat, ["%o%", 10]);
      if (JSON.stringify(repeatTpl.values) !== JSON.stringify(["%o%", "%o%", 10])) {
        fail(checks, "repeat expansion", JSON.stringify(repeatTpl.values));
      }
      pass(checks, "repeat expansion", JSON.stringify(repeatTpl.values));
      const commentedTpl = sqlHandles.template(commented, ["%", 100]);
      if (JSON.stringify(commentedTpl.values) !== JSON.stringify(["%", 100])) {
        fail(checks, "comment expansion", JSON.stringify(commentedTpl.values));
      }
      pass(checks, "comment expansion", JSON.stringify(commentedTpl.values));

      assertNames(checks, "search like", await run(search, ["%son%", 10]), ["Jason", "Mason"]);
      assertNames(checks, "search quote", await run(search, ["O'Brien", 10]), ["O'Brien"]);
      assertNames(checks, "search unicode", await run(search, ["Zoë", 10]), ["Zoë"]);

      const injectionOr = await run(search, ["%' OR '1'='1", 10]);
      if (injectionOr.length !== 0) fail(checks, "injection or", `matched ${injectionOr.length} rows`);
      pass(checks, "injection or", "0 rows");
      const stacked = await run(search, ["'; DROP TABLE can_i37_accounts;--", 10]);
      if (stacked.length !== 0) fail(checks, "injection stacked", `matched ${stacked.length} rows`);
      pass(checks, "injection stacked", "0 rows");
      const survivors = (await run(search, ["%", 100])) as Array<{ display_name?: unknown }>;
      if (survivors.length !== 6) fail(checks, "table survives injection", `saw ${survivors.length} rows`);
      pass(checks, "table survives injection", "6 rows");

      assertNames(checks, "repeat binds once", await run(repeat, ["%o%", 10]), ["Jason", "Mason", "Bob", "O'Brien", "Zoë"]);

      const one = (await run(byId, [4, 1])) as Array<{ id?: unknown; display_name?: unknown }>;
      if (one.length !== 1 || one[0]?.id !== 4 || one[0]?.display_name !== "Bob") {
        fail(checks, "by id one", JSON.stringify(one));
      }
      pass(checks, "by id one", "Bob");
      const missing = await run(byId, [999, 1]);
      if (missing.length !== 0) fail(checks, "by id missing", `matched ${missing.length} rows`);
      pass(checks, "by id missing", "0 rows");

      const addTpl = sqlHandles.template(add, ["Q'ueen"]);
      const added = (await sql(addTpl.strings, ...addTpl.values)) as unknown as {
        count?: unknown;
      };
      if (added?.count !== 1) fail(checks, "insert execute", JSON.stringify(added));
      pass(checks, "insert execute", "count 1");
      assertNames(checks, "insert round trip", await run(search, ["Q'ueen", 10]), ["Q'ueen"]);

      const commentedRows = await run(commented, ["%", 100]);
      if (commentedRows.length !== 7) fail(checks, "comment literal", `saw ${commentedRows.length} rows`);
      pass(checks, "comment literal", "7 rows");

      const limited = await run(search, ["%", 2]);
      if (limited.length !== 2) fail(checks, "limit bound", `saw ${limited.length} rows`);
      pass(checks, "limit bound", "2 rows");

      report({ serverVersion });
    } finally {
      await sql(staticTemplate(drop));
    }
    await sql.close();
  } catch (error) {
    await sql.close().catch(() => undefined);
    throw error;
  }
} catch (error) {
  if (checks.length === 0 || checks[checks.length - 1]?.ok !== false) {
    checks.push({ name: "harness", ok: false, detail: error instanceof Error ? error.message : String(error) });
  }
  report({});
  process.exitCode = 1;
}
if (checks.some((check) => !check.ok)) process.exitCode = 1;
