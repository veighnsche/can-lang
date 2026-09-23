// MySQL-specific native mapping: client establishment, native failure
// classification, and statement metadata. Only MySQLError classifies;
// anything else is a driver defect and propagates to the generic fault
// path. Sanitized fields only: verified errno symbols and validated
// configuration, never messages, URLs, or credentials.
import { success, type Completion } from "../../completion.ts";
import type { SQLCoreContracts, SQLFailures } from "./errors.ts";

export function isMySQLFailure(value: unknown): value is Error & { code: string; errno?: unknown } {
  return value instanceof Error && value.name === "MySQLError" && typeof (value as { code?: unknown }).code === "string";
}

// Verified server errno symbols, each observed against the provisioned
// 8.4 service. Anything else keeps its errno-qualified fallback so
// unmapped server failures stay distinguishable, never "unknown".
const mysqlErrNames: Record<number, string> = {
  1044: "ER_DBACCESS_DENIED_ERROR",
  1045: "ER_ACCESS_DENIED_ERROR",
  1062: "ER_DUP_ENTRY",
  1064: "ER_PARSE_ERROR",
  1146: "ER_NO_SUCH_TABLE",
  1205: "ER_LOCK_WAIT_TIMEOUT",
  1213: "ER_LOCK_DEADLOCK",
};

export function classifyMySQL(operation: string, cause: unknown, failures: SQLFailures, contracts: SQLCoreContracts): Completion<never> {
  if (!isMySQLFailure(cause)) throw cause;
  if (cause.code === "ERR_MYSQL_CONNECTION_REFUSED" || cause.code === "ERR_MYSQL_CONNECTION_CLOSED") {
    return failures.connectionFailed("query");
  }
  const errno = typeof cause.errno === "number" ? cause.errno : 0;
  if (errno === 1045 || errno === 1044) {
    return failures.connectionFailed("auth");
  }
  // Duplicate keys carry the structured errno symbol, not a parsed
  // index name: messages are never read for classification.
  if (errno === 1062) {
    return failures.fail(contracts.constraintFailed, [["constraint", "ER_DUP_ENTRY"]]);
  }
  const known = mysqlErrNames[errno];
  if (known !== undefined) return failures.queryFailed(operation, known);
  if (errno !== 0) return failures.queryFailed(operation, `mysql_errno_${errno}`);
  return failures.queryFailed(operation, cause.code !== "" ? cause.code : "unknown");
}

export function mysqlAffectedRows(operation: string, result: unknown, failures: SQLFailures): Completion<bigint> {
  const affected = (result as { affectedRows?: unknown } | null)?.affectedRows;
  if (typeof affected !== "number" || !Number.isSafeInteger(affected) || affected < 0) {
    return failures.queryFailed(operation, "bad_count");
  }
  return success(BigInt(affected));
}

// Static pin template: one frozen literal, never interpolated.
const pinText = "SELECT @@session.time_zone AS tz";
const pinStrings = Object.freeze(Object.assign([pinText], { raw: Object.freeze([pinText]) })) as unknown as TemplateStringsArray;

export async function openMySQLClient(url: string, max: number, failures: SQLFailures): Promise<{ ok: true; client: InstanceType<typeof Bun.SQL> } | { ok: false; failure: Completion<never> }> {
  // TLS is always on: plaintext falls back to public-key retrieval,
  // which fails closed, so unencrypted auth never negotiates. Bun
  // does not verify the server chain (a self-signed server connects),
  // so this is encryption without server authentication.
  const client = new Bun.SQL(url, { adapter: "mysql", bigint: true, max, tls: true });
  try {
    // Construction is lazy; awaiting connect proves establishment. A
    // refused host or bad credentials surface here as connection
    // failures without ever exposing the URL.
    await client.connect();
  } catch {
    try { await client.close(); } catch { /* already failed; report the connection */ }
    return { ok: false, failure: failures.connectionFailed("connect") };
  }
  // The driver pins every session to UTC (+00:00 observed under all
  // client timezones and server defaults, on pooled and reconnected
  // handles alike); naive DATETIME rendering depends on that pin, so
  // the open verifies it and refuses the pool on any drift rather
  // than misrender values.
  let pinned = false;
  try {
    const rows = await client(pinStrings) as Array<{ tz?: unknown }>;
    pinned = Array.isArray(rows) && rows[0]?.tz === "+00:00";
  } catch {
    pinned = false;
  }
  if (!pinned) {
    try { await client.close(); } catch { /* already failed; report the connection */ }
    return { ok: false, failure: failures.connectionFailed("config") };
  }
  return { ok: true, client };
}
