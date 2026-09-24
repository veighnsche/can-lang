// T19 smoke-only seed helper. Inserts the one row the sqlite example
// renames and reads back, then reports the stored row. Executed by the
// installed sidecar; the compiled Can program exercises SQL behavior
// itself, never this driver.
import { SQL } from "bun";

function staticTemplate(text: string): TemplateStringsArray {
  const parts = [text];
  return Object.freeze(Object.assign(parts.slice(), { raw: Object.freeze(parts.slice()) })) as unknown as TemplateStringsArray;
}

try {
  const dbPath = process.argv[2];
  if (typeof dbPath !== "string" || dbPath === "") throw new Error("missing database path argument");
  const sql = new SQL({ adapter: "sqlite", filename: dbPath, safeIntegers: true });
  try {
    await sql(staticTemplate("INSERT INTO notes (id, body) VALUES (7, 'seed')"));
    const rows = (await sql(staticTemplate("SELECT id, body FROM notes WHERE id = 7"))) as Array<{ id: bigint; body: string }>;
    if (rows.length !== 1 || rows[0]?.body !== "seed") throw new Error("seed row missing");
    console.log(JSON.stringify({ id: String(rows[0].id), body: rows[0].body }));
  } finally {
    await sql.close();
  }
} catch (cause) {
  console.error(String((cause as Error)?.message ?? cause));
  process.exit(1);
}
