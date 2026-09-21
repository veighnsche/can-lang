// I38 test-only live setup harness. Executed by
// tests/integration/sql_test.go against a disposable PostgreSQL named by
// CAN_TEST_POSTGRES_URL. Modes: "setup" applies the fixed seed file through
// static templates and reports the server version; "teardown" drops the
// I38 table. Transaction behavior itself is exercised through the compiled
// program CLI, not here: this driver only prepares and removes schema.
import { SQL } from "bun";

function staticTemplate(text: string): TemplateStringsArray {
  const parts = [text];
  return Object.freeze(Object.assign(parts.slice(), { raw: Object.freeze(parts.slice()) })) as unknown as TemplateStringsArray;
}

try {
  const mode = process.argv[2];
  const seedPath = process.argv[3];
  const url = process.env.CAN_TEST_POSTGRES_URL;
  if (mode !== "setup" && mode !== "teardown") throw new Error(`unknown mode ${String(mode)}`);
  if (typeof seedPath !== "string" || seedPath === "") throw new Error("missing seed path argument");
  if (typeof url !== "string" || url === "") throw new Error("set CAN_TEST_POSTGRES_URL to the disposable test database");
  if (url.includes(" ") || !url.startsWith("postgres")) throw new Error("refusing non-postgres test URL");

  const sql = new SQL(url);
  try {
    const seed = await Bun.file(seedPath).text();
    const statements = seed
      .split("\n")
      .filter((line) => !line.trimStart().startsWith("--"))
      .join("\n")
      .split(";")
      .map((part) => part.trim())
      .filter((part) => part !== "");
    if (statements.length !== 5) throw new Error(`seed file must hold 5 statements, saw ${statements.length}`);
    const [resetAccounts, createAccounts, truncateAccounts, insertAccounts, dropAccounts] =
      statements as [string, string, string, string, string];
    if (
      !resetAccounts.startsWith("DROP TABLE") ||
      !createAccounts.startsWith("CREATE TABLE") || !truncateAccounts.startsWith("TRUNCATE") ||
      !insertAccounts.startsWith("INSERT INTO") || !dropAccounts.startsWith("DROP TABLE")
    ) {
      throw new Error("seed file shape changed; refusing to run setup/teardown");
    }
    if (mode === "setup") {
      for (const setup of [resetAccounts, createAccounts, truncateAccounts, insertAccounts]) {
        await sql(staticTemplate(setup));
      }
      const versionRows = (await sql(staticTemplate("SELECT version()"))) as Array<{ version: string }>;
      const serverVersion = String(versionRows[0]?.version ?? "");
      if (!serverVersion.startsWith("PostgreSQL 17.")) {
        throw new Error(`unexpected test database version: ${serverVersion.slice(0, 80)}`);
      }
      const accounts = (await sql(staticTemplate("SELECT COUNT(*)::int AS count FROM can_i38_accounts"))) as Array<{ count: number }>;
      console.log(JSON.stringify({ serverVersion, accounts: accounts[0]?.count ?? -1 }));
    } else {
      await sql(staticTemplate(dropAccounts));
      console.log(JSON.stringify({ dropped: true }));
    }
    await sql.close();
  } catch (error) {
    await sql.close().catch(() => undefined);
    throw error;
  }
} catch (error) {
  console.log(JSON.stringify({ error: error instanceof Error ? error.message : String(error) }));
  process.exitCode = 1;
}
