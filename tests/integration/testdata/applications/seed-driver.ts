// I42 test-only live setup harness. Executed by
// tests/integration/applications_test.go against a disposable PostgreSQL
// named by CAN_TEST_POSTGRES_URL. Modes: "setup" applies the fixed seed
// file through static templates and reports the server version plus row
// counts; "teardown" drops the I42 tables. Application behavior itself
// is exercised through staged builds and served programs, not here:
// this driver only prepares and removes schema.
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
    if (statements.length !== 12) throw new Error(`seed file must hold 12 statements, saw ${statements.length}`);
    const setup = statements.slice(0, 9);
    const teardown = statements.slice(9);
    for (const statement of setup.slice(0, 3)) {
      if (!statement.startsWith("DROP TABLE")) throw new Error("seed reset shape changed; refusing setup");
    }
    for (const statement of setup.slice(3)) {
      if (!statement.startsWith("CREATE TABLE") && !statement.startsWith("INSERT INTO")) {
        throw new Error("seed setup shape changed; refusing setup");
      }
    }
    for (const statement of teardown) {
      if (!statement.startsWith("DROP TABLE")) throw new Error("seed teardown shape changed; refusing teardown");
    }
    if (mode === "setup") {
      for (const statement of setup) {
        await sql(staticTemplate(statement));
      }
      const versionRows = (await sql(staticTemplate("SELECT version()"))) as Array<{ version: string }>;
      const serverVersion = String(versionRows[0]?.version ?? "");
      if (!serverVersion.startsWith("PostgreSQL 17.")) {
        throw new Error(`unexpected test database version: ${serverVersion.slice(0, 80)}`);
      }
      const count = async (table: string) => {
        const rows = (await sql(staticTemplate(`SELECT COUNT(*)::int AS count FROM ${table}`))) as Array<{ count: number }>;
        return rows[0]?.count ?? -1;
      };
      console.log(
        JSON.stringify({
          serverVersion,
          accounts: await count("can_i42_accounts"),
          dashboard: await count("can_i42_dashboard"),
          triage: await count("can_i42_triage"),
        }),
      );
    } else {
      for (const statement of teardown) {
        await sql(staticTemplate(statement));
      }
      console.log(JSON.stringify({ dropped: true }));
    }
  } finally {
    await sql.close();
  }
} catch (error) {
  console.log(JSON.stringify({ error: String((error as Error)?.message ?? error) }));
  process.exit(1);
}
