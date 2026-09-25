// UP16 test-only setup/seeding/inspection harness for the invoice slice.
// Executed by tests/integration/invoice_test.go against a disposable SQLite
// file database. Modes: "setup" applies the example schema.sql through
// static templates; "seed" inserts the fixed session, membership, invoice
// and line rows over integer tenant/invoice keys (including hostile
// details/line-id text that must render escaped); "inspect" reports
// invoice, line and replay rows as JSON with integers stringified;
// "touch-replay" rewrites one ledger row's recorded stamp so retention
// tests can place rows before, at or after the expiry boundary. The
// compiled program serves every HTTP behavior itself; this driver only
// prepares the file the operator would prepare, adjusts its own seeded
// ledger stamps, and reads back rows the test asserts on.
import { SQL } from "bun";

function staticTemplate(text: string): TemplateStringsArray {
  const parts = [text];
  return Object.freeze(Object.assign(parts.slice(), { raw: Object.freeze(parts.slice()) })) as unknown as TemplateStringsArray;
}

function text(value: unknown): string {
  if (typeof value === "bigint") return value.toString(10);
  return String(value ?? "");
}

function literal(value: string): string {
  return `'${value.replaceAll("'", "''")}'`;
}

function integerText(value: string, name: string): string {
  if (!/^(0|[1-9][0-9]*)$/.test(value)) throw new Error(`invalid ${name}: ${value}`);
  return value;
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
        "INSERT INTO invoice_membership (actor, tenant) VALUES ('alice', 1), ('bob', 2)",
        "INSERT INTO invoice (tenant, id, revision, seats, details, updated_ms) VALUES (1, 7, 1, 2, 'Acme <em>&\" ''coop''\"', 900), (2, 8, 1, 1, 'Globex', 800)",
        "INSERT INTO invoice_line (invoice_id, line_key, id, quantity, price_minor, position) VALUES (7, 'k1', 'sku-1 <b>', 2, 1999, 0), (7, 'k2', 'plain', 1, 500, 1)",
      ];
      for (const seed of seeds) await sql(staticTemplate(seed));
      console.log(JSON.stringify({ sessions: 3, memberships: 2, invoices: 2, lines: 2 }));
    } else if (mode === "inspect") {
      const invoice = (await sql(
        staticTemplate("SELECT tenant, id, revision, seats, details, updated_ms FROM invoice ORDER BY id"),
      )) as Record<string, unknown>[];
      const lines = (await sql(
        staticTemplate("SELECT invoice_id, line_key, id, quantity, price_minor, position FROM invoice_line ORDER BY invoice_id, position"),
      )) as Record<string, unknown>[];
      const replays = (await sql(
        staticTemplate("SELECT actor, tenant, invoice_id, operation_id, digest, result, revision, recorded_ms FROM invoice_replay ORDER BY actor, tenant, invoice_id, operation_id"),
      )) as Record<string, unknown>[];
      const byInvoice = new Map<string, Record<string, string>[]>();
      for (const line of lines) {
        const id = text(line["invoice_id"]);
        const list = byInvoice.get(id) ?? [];
        list.push({
          key: text(line["line_key"]),
          id: text(line["id"]),
          quantity: text(line["quantity"]),
          price_minor: text(line["price_minor"]),
          position: text(line["position"]),
        });
        byInvoice.set(id, list);
      }
      console.log(
        JSON.stringify({
          invoices: invoice.map((row) => ({
            tenant: text(row["tenant"]),
            id: text(row["id"]),
            revision: text(row["revision"]),
            seats: text(row["seats"]),
            details: text(row["details"]),
            updated_ms: text(row["updated_ms"]),
            lines: byInvoice.get(text(row["id"])) ?? [],
          })),
          replays: replays.map((row) => ({
            actor: text(row["actor"]),
            tenant: text(row["tenant"]),
            invoice: text(row["invoice_id"]),
            operation: text(row["operation_id"]),
            digest: text(row["digest"]),
            result: text(row["result"]),
            revision: text(row["revision"]),
            recorded: text(row["recorded_ms"]),
          })),
        }),
      );
    } else if (mode === "touch-replay") {
      const actor = process.argv[4];
      const tenant = process.argv[5];
      const invoice = process.argv[6];
      const operation = process.argv[7];
      const recorded = process.argv[8];
      if (typeof actor !== "string" || typeof operation !== "string" || actor === "" || operation === "") {
        throw new Error("touch-replay needs actor and operation arguments");
      }
      const stamp = integerText(recorded ?? "", "recorded_ms");
      await sql(
        staticTemplate(
          `UPDATE invoice_replay SET recorded_ms = ${stamp} WHERE actor = ${literal(actor)} AND tenant = ${integerText(tenant ?? "", "tenant")} AND invoice_id = ${integerText(invoice ?? "", "invoice_id")} AND operation_id = ${literal(operation)}`,
        ),
      );
      const rows = (await sql(
        staticTemplate(
          `SELECT recorded_ms FROM invoice_replay WHERE actor = ${literal(actor)} AND tenant = ${integerText(tenant ?? "", "tenant")} AND invoice_id = ${integerText(invoice ?? "", "invoice_id")} AND operation_id = ${literal(operation)}`,
        ),
      )) as Record<string, unknown>[];
      if (rows.length !== 1) throw new Error("touch-replay matched no ledger row");
      console.log(JSON.stringify({ recorded: text(rows[0]["recorded_ms"]) }));
    } else {
      throw new Error(`unknown driver mode ${mode ?? ""}`);
    }
  } finally {
    await sql.close();
  }
} catch (error) {
  console.error(error instanceof Error ? error.message : String(error));
  process.exit(1);
}
