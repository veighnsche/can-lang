// B1-02 test-only live setup harness. Executed by
// tests/integration/sql_test.go against a disposable SQLite file database.
// Modes: "setup" applies the fixed seed file through static templates and
// reports the library version; the compiled program CLI exercises query
// behavior itself, never this driver.
import { SQL } from "bun";

function staticTemplate(text: string): TemplateStringsArray {
  const parts = [text];
  return Object.freeze(Object.assign(parts.slice(), { raw: Object.freeze(parts.slice()) })) as unknown as TemplateStringsArray;
}

try {
  const mode = process.argv[2];
  const dbPath = process.argv[3];
  const seedPath = process.argv[4];
  if (mode !== "setup") throw new Error(`unknown mode ${String(mode)}`);
  if (typeof dbPath !== "string" || dbPath === "") throw new Error("missing database path argument");
  if (typeof seedPath !== "string" || seedPath === "") throw new Error("missing seed path argument");

  const sql = new SQL({ adapter: "sqlite", filename: dbPath, safeIntegers: true });
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
    const versionRows = (await sql(staticTemplate("SELECT sqlite_version() AS version"))) as Array<{ version: string }>;
    const libraryVersion = String(versionRows[0]?.version ?? "");
    if (libraryVersion === "") throw new Error("missing sqlite library version");
    const notes = (await sql(staticTemplate("SELECT COUNT(*) AS count FROM notes"))) as Array<{ count: bigint }>;
    console.log(JSON.stringify({ libraryVersion, notes: String(notes[0]?.count ?? -1) }));
  } finally {
    await sql.close();
  }
} catch (cause) {
  console.error(String((cause as Error)?.message ?? cause));
  process.exit(1);
}
