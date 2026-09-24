// T19 PostgreSQL qualification driver. Runs a create/insert/select/drop
// roundtrip against DATABASE_URL through the installed sidecar and
// reports the server version with the fetched row. The database is a
// disposable postgres:17 container (see provenance.json); nothing here
// is production migration or pooling policy.
import { SQL } from "bun";

function staticTemplate(text: string): TemplateStringsArray {
  const parts = [text];
  return Object.freeze(Object.assign(parts.slice(), { raw: Object.freeze(parts.slice()) })) as unknown as TemplateStringsArray;
}

try {
  const url = Bun.env.DATABASE_URL;
  if (typeof url !== "string" || url === "") throw new Error("DATABASE_URL is required");
  const sql = new SQL(url);
  try {
    const versionRows = (await sql(staticTemplate("SHOW server_version"))) as Array<{ server_version: string }>;
    const serverVersion = String(versionRows[0]?.server_version ?? "");
    if (!serverVersion.startsWith("17.")) throw new Error(`requires PostgreSQL 17, saw ${serverVersion}`);
    await sql(staticTemplate("DROP TABLE IF EXISTS t19_smoke"));
    await sql(staticTemplate("CREATE TABLE t19_smoke (id BIGINT PRIMARY KEY, body TEXT NOT NULL)"));
    await sql(staticTemplate("INSERT INTO t19_smoke (id, body) VALUES (7, 'remember')"));
    const rows = (await sql(staticTemplate("SELECT id, body FROM t19_smoke WHERE id = 7"))) as Array<{ id: string; body: string }>;
    if (rows.length !== 1 || rows[0]?.body !== "remember") throw new Error("roundtrip row missing");
    await sql(staticTemplate("DROP TABLE t19_smoke"));
    console.log(JSON.stringify({ serverVersion, id: String(rows[0].id), body: rows[0].body }));
  } finally {
    await sql.close();
  }
} catch (cause) {
  console.error(String((cause as Error)?.message ?? cause));
  process.exit(1);
}
