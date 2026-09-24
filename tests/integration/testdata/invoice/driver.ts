// T15 test-only setup/seeding/inspection harness for the server-rendered
// invoice form slice. Executed by tests/integration/invoice_test.go against
// a disposable SQLite file database. Modes: "setup" applies the example
// schema.sql through static templates; "seed" inserts the fixed session,
// membership, invoice and line rows (including hostile customer/sku text
// that must render escaped); "inspect" reports invoice, line and replay
// rows as JSON with integers stringified. The compiled program serves
// every HTTP behavior itself; this driver only prepares the file the
// operator would prepare and reads back rows the test asserts on.
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
      if (statements.length !== 5) throw new Error(`schema file must hold 5 statements, saw ${statements.length}`);
      for (const setup of statements) {
        const prefix = "CREATE TABLE IF NOT EXISTS invoice";
        const rest = setup.startsWith(prefix) ? setup.slice(prefix.length) : "";
        if (!rest.startsWith("_") && !rest.startsWith(" (")) {
          throw new Error("schema file shape changed; refusing to run setup");
        }
        await sql(staticTemplate(setup));
      }
      console.log(JSON.stringify({ tables: statements.length }));
    } else if (mode === "seed") {
      const seeds = [
        "INSERT INTO invoice_session (token, actor, revoked_ms) VALUES ('tok-alice', 'alice', 0), ('tok-bob', 'bob', 0), ('tok-revoked', 'mallory', 1700000000000)",
        "INSERT INTO invoice_membership (actor, tenant) VALUES ('alice', 'tenant-a'), ('bob', 'tenant-b')",
        "INSERT INTO invoice (id, tenant, revision, customer, updated_ms) VALUES ('inv-1', 'tenant-a', 1, 'Acme <em>&\" ''coop''\"', 900), ('inv-2', 'tenant-b', 1, 'Globex', 800)",
        "INSERT INTO invoice_line (invoice_id, line_key, sku, qty, position) VALUES ('inv-1', 'k1', 'sku-1 <b>', 2, 0), ('inv-1', 'k2', 'plain', 1, 1)",
      ];
      for (const seed of seeds) await sql(staticTemplate(seed));
      console.log(JSON.stringify({ sessions: 3, memberships: 2, invoices: 2, lines: 2 }));
    } else if (mode === "inspect") {
      const invoice = (await sql(
        staticTemplate("SELECT id, tenant, revision, customer, updated_ms FROM invoice ORDER BY id"),
      )) as Array<Record<string, unknown>>;
      const lines = (await sql(
        staticTemplate("SELECT invoice_id, line_key, sku, qty, position FROM invoice_line ORDER BY invoice_id, position"),
      )) as Array<Record<string, unknown>>;
      const replay = (await sql(
        staticTemplate(
          "SELECT actor, invoice_id, operation_id, digest, revision, recorded_ms FROM invoice_replay ORDER BY actor, invoice_id, operation_id",
        ),
      )) as Array<Record<string, unknown>>;
      const freeze = (rows: Array<Record<string, unknown>>) =>
        rows.map((row) => Object.fromEntries(Object.entries(row).map(([key, value]) => [key, text(value)])));
      console.log(JSON.stringify({ invoice: freeze(invoice), lines: freeze(lines), replay: freeze(replay) }));
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
