// T16 test-only setup/inspection harness for the signed provider
// webhook slice. Executed by tests/integration/webhook_test.go against
// a disposable SQLite file database. Modes: "setup" applies the
// example schema.sql through static templates; "inspect" reports the
// ledger, outbox and attempt rows as JSON with integers stringified.
// The compiled program serves every HTTP behavior itself; this driver
// only prepares the file the operator would prepare and reads back
// rows the test asserts on.
import { SQL } from "bun";

function staticTemplate(text: string): TemplateStringsArray {
  const parts = [text];
  return Object.freeze(Object.assign(parts.slice(), { raw: Object.freeze(parts.slice()) })) as unknown as TemplateStringsArray;
}

function text(value: unknown): string {
  if (typeof value === "bigint") return value.toString(10);
  return String(value ?? "");
}

try {
  const mode = process.argv[2];
  const dbPath = process.argv[3];
  if (typeof dbPath !== "string" || dbPath === "") throw new Error("missing database path argument");
  const sql = new SQL({ adapter: "sqlite", filename: dbPath, safeIntegers: true });
  try {
    if (mode === "setup") {
      const schemaPath = process.argv[4];
      if (typeof schemaPath !== "string" || schemaPath === "") throw new Error("missing schema path argument");
      const schema = await Bun.file(schemaPath).text();
      const statements = schema
        .split("\n")
        .filter((line) => !line.trimStart().startsWith("--"))
        .join("\n")
        .split(";")
        .map((part) => part.trim())
        .filter((part) => part !== "");
      if (statements.length !== 3) throw new Error(`schema file must hold 3 statements, saw ${statements.length}`);
      for (const setup of statements) {
        if (!setup.startsWith("CREATE TABLE IF NOT EXISTS webhook_")) {
          throw new Error("schema file shape changed; refusing to run setup");
        }
        await sql(staticTemplate(setup));
      }
      console.log(JSON.stringify({ tables: statements.length }));
    } else if (mode === "inspect") {
      const ledger = (await sql(staticTemplate("SELECT delivery_id, digest, event, subscription, outcome, received_ms FROM webhook_ledger ORDER BY delivery_id"))) as Array<Record<string, unknown>>;
      const outbox = (await sql(staticTemplate("SELECT delivery_id, event, subscription, state, attempts, updated_ms FROM webhook_outbox ORDER BY delivery_id"))) as Array<Record<string, unknown>>;
      const attempts = (await sql(staticTemplate("SELECT id, delivery_id, outcome, at_ms FROM webhook_attempt ORDER BY id"))) as Array<Record<string, unknown>>;
      const freeze = (rows: Array<Record<string, unknown>>) =>
        rows.map((row) => Object.fromEntries(Object.entries(row).map(([key, value]) => [key, text(value)])));
      console.log(JSON.stringify({ ledger: freeze(ledger), outbox: freeze(outbox), attempts: freeze(attempts) }));
    } else {
      throw new Error(`unknown mode ${String(mode)}`);
    }
  } finally {
    await sql.close();
  }
} catch (cause) {
  console.error(String((cause as Error)?.message ?? cause));
  process.exit(1);
}
