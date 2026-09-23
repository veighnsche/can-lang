// B1-03 live EXPLAIN differential: every mysql row of sql-corpus.json goes
// through a real server-side EXPLAIN on the provisioned MySQL service and
// the accept/reject verdict is recorded next to the tidb verdict the Go
// corpus runner pins. SQL-level PREPARE is unusable here (Bun sends every
// statement through the binary protocol, where the server answers PREPARE
// with errno 1295), and EXPLAIN parses and plans without executing, which
// is exactly the comparison the differential needs. Bun 1.4.2 is the only
// wire driver; usage:
//   CAN_TEST_MYSQL_URL=mysql://... bun mysql-differential-probe.ts <rows.json> <out.json>
// rows.json holds [{id, statement, sites}]; the Go differential test feeds
// each row's tidb site count back as the binding count, so a miscounted
// site (a `?` inside a comment or literal, or a missed placeholder) fails
// the differential instead of silently shifting the comparison. The probe
// creates and drops a scratch `users` table covering every column the
// corpus references, so name resolution can never skew the comparison.
import { SQL } from "bun";

const rowsPath = process.argv[2];
const outPath = process.argv[3];
const url = process.env.CAN_TEST_MYSQL_URL;
if (typeof rowsPath !== "string" || rowsPath === "") throw new Error("missing rows path argument");
if (typeof outPath !== "string" || outPath === "") throw new Error("missing output path argument");
if (typeof url !== "string" || url === "") throw new Error("CAN_TEST_MYSQL_URL is not set");

function staticTemplate(text: string): TemplateStringsArray {
  const parts = [text];
  return Object.freeze(Object.assign(parts.slice(), { raw: Object.freeze(parts.slice()) })) as unknown as TemplateStringsArray;
}

type Row = { id: string; statement: string; sites: number };
const rows = (await Bun.file(rowsPath).json()) as Row[];
if (rows.length !== 13) throw new Error(`mysql differential moved: saw ${rows.length} rows, want 13`);

const sql = new SQL(url, { adapter: "mysql", bigint: true, max: 1, tls: true });
const verdicts: Record<string, string> = {};
try {
  const versionRows = (await sql(staticTemplate("SELECT VERSION() AS v"))) as Array<{ v: string }>;
  const version = String(versionRows[0]?.v ?? "");
  if (version === "") throw new Error("missing server version");
  const statusRows = (await sql(staticTemplate("SHOW SESSION STATUS WHERE Variable_name IN ('Ssl_version', 'Ssl_cipher')"))) as Array<{ Variable_name: string; Value: string }>;
  const status: Record<string, string> = {};
  for (const row of statusRows) status[row.Variable_name] = String(row.Value);
  const modeRows = (await sql(staticTemplate("SELECT @@SESSION.sql_mode AS m"))) as Array<{ m: string }>;
  const sqlMode = String(modeRows[0]?.m ?? "");
  await sql(staticTemplate("DROP TABLE IF EXISTS users"));
  await sql(staticTemplate("CREATE TABLE users (id BIGINT PRIMARY KEY, name VARCHAR(255), nick VARCHAR(255), a BIGINT, b BIGINT, `select` VARCHAR(255))"));
  try {
    for (const row of rows) {
      const bindings = Array.from({ length: row.sites }, () => 1);
      try {
        await (sql as unknown as { unsafe: (text: string, params: unknown[]) => Promise<unknown> }).unsafe("EXPLAIN " + row.statement, bindings);
        verdicts[row.id] = "ok";
      } catch (cause) {
        const code = (cause as { code?: unknown })?.code;
        const errno = (cause as { errno?: unknown })?.errno;
        verdicts[row.id] = "err:" + (typeof code === "string" && code !== "" ? code : "unknown")
          + (typeof errno === "number" ? "/" + errno : "");
      }
    }
  } finally {
    await sql(staticTemplate("DROP TABLE IF EXISTS users"));
  }
  const report = {
    bun: Bun.version,
    mysql: { version, sql_mode: sqlMode, Ssl_version: status.Ssl_version ?? "", Ssl_cipher: status.Ssl_cipher ?? "" },
    verdicts,
  };
  await Bun.write(outPath, JSON.stringify(report, null, 1) + "\n");
} finally {
  await sql.close();
}
