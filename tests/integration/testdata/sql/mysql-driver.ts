// B1-03 test-only live setup harness. Executed by
// tests/integration/mysql_test.go against the provisioned disposable
// MySQL service. Modes: "setup" applies the fixed seed file through
// static templates and reports the server version; the compiled program
// CLI exercises query behavior itself, never this driver.
import { SQL } from "bun";

function staticTemplate(text: string): TemplateStringsArray {
  const parts = [text];
  return Object.freeze(Object.assign(parts.slice(), { raw: Object.freeze(parts.slice()) })) as unknown as TemplateStringsArray;
}

try {
  const mode = process.argv[2];
  const seedPath = process.argv[3];
  const url = process.env.CAN_TEST_MYSQL_URL;
  if (mode !== "setup") throw new Error(`unknown mode ${String(mode)}`);
  if (typeof seedPath !== "string" || seedPath === "") throw new Error("missing seed path argument");
  if (typeof url !== "string" || url === "") throw new Error("CAN_TEST_MYSQL_URL is not set");

  const sql = new SQL(url, { adapter: "mysql", bigint: true, max: 1, tls: true });
  try {
    const seed = await Bun.file(seedPath).text();
    const statements = seed
      .split("\n")
      .filter((line) => !line.trimStart().startsWith("--"))
      .join("\n")
      .split(";")
      .map((part) => part.trim())
      .filter((part) => part !== "");
    if (statements.length !== 2) throw new Error(`seed file must hold 2 statements, saw ${statements.length}`);
    const [reset, create] = statements as [string, string];
    if (!reset.startsWith("DROP TABLE") || !create.startsWith("CREATE TABLE")) {
      throw new Error("seed file shape changed; refusing to run setup");
    }
    for (const setup of [reset, create]) {
      await sql(staticTemplate(setup));
    }
    const versionRows = (await sql(staticTemplate("SELECT VERSION() AS version"))) as Array<{ version: string }>;
    const serverVersion = String(versionRows[0]?.version ?? "");
    if (serverVersion === "") throw new Error("missing mysql server version");
    const notes = (await sql(staticTemplate("SELECT COUNT(*) AS count FROM notes"))) as Array<{ count: bigint }>;
    console.log(JSON.stringify({ serverVersion, notes: String(notes[0]?.count ?? -1) }));
  } finally {
    await sql.close();
  }
} catch (cause) {
  console.error(String((cause as Error)?.message ?? cause));
  process.exit(1);
}
