// SQLite-specific native mapping: client establishment, native failure
// classification, and statement metadata. Only SQLiteError classifies;
// anything else is a driver defect and propagates to the generic fault
// path. Sanitized fields only: codes and validated configuration, never
// messages, filenames, or timeouts beyond their canonical digits.
import { success, type Completion } from "../../completion.ts";
import type { SQLCoreContracts, SQLFailures } from "./errors.ts";
import type { SQLiteFileConfig } from "./config.ts";

export function isSQLiteFailure(value: unknown): value is Error & { code: string } {
  return (
    value instanceof Error &&
    value.name === "SQLiteError" &&
    typeof (value as { code?: unknown }).code === "string"
  );
}

export function classifySQLite(
  operation: string,
  cause: unknown,
  failures: SQLFailures,
  contracts: SQLCoreContracts,
): Completion<never> {
  if (!isSQLiteFailure(cause)) throw cause;
  if (cause.code === "ERR_SQLITE_CONNECTION_CLOSED") {
    return failures.connectionFailed("query");
  }
  // Constraint failures carry the structured SQLite code, not a parsed
  // table or column name: messages are never read for classification.
  if (cause.code.startsWith("SQLITE_CONSTRAINT")) {
    return failures.fail(contracts.constraintFailed, [["constraint", cause.code]]);
  }
  // SQLITE_BUSY keeps its code so callers can distinguish lock contention
  // from other query failures after their busy timeout elapsed.
  return failures.queryFailed(operation, cause.code !== "" ? cause.code : "unknown");
}

export function sqliteAffectedRows(
  operation: string,
  result: unknown,
  failures: SQLFailures,
): Completion<bigint> {
  // Bun.SQL safeIntegers returns SELECT integer columns as bigint. Can's
  // SQLite execute path passes the connection-local changes() value here.
  if (typeof result === "bigint") {
    if (result < 0n) return failures.queryFailed(operation, "bad_count");
    return success(result);
  }
  const count = (result as { count?: unknown } | null)?.count;
  if (typeof count !== "number" || !Number.isSafeInteger(count) || count < 0) {
    return failures.queryFailed(operation, "bad_count");
  }
  return success(BigInt(count));
}

export function sqliteChangesCount(result: unknown): unknown {
  if (!Array.isArray(result) || result.length !== 1) return undefined;
  const row = result[0];
  if (row === null || typeof row !== "object") return undefined;
  return (row as { affected_rows?: unknown }).affected_rows;
}

function staticTemplate(text: string): TemplateStringsArray {
  return Object.freeze(
    Object.assign([text], { raw: Object.freeze([text]) }),
  ) as unknown as TemplateStringsArray;
}

const SQLITE_CHANGES = staticTemplate("SELECT changes() AS affected_rows");

export function sqliteChangesTemplate(): TemplateStringsArray {
  return SQLITE_CHANGES;
}

const FOREIGN_KEYS_ON = staticTemplate("PRAGMA foreign_keys = ON");
const FOREIGN_KEYS_READ = staticTemplate("PRAGMA foreign_keys");

async function failConnect(
  client: InstanceType<typeof Bun.SQL>,
  failures: SQLFailures,
): Promise<Completion<never>> {
  try {
    await client.close();
  } catch {
    /* already failed; report the connection */
  }
  return failures.connectionFailed("connect");
}

async function established(
  client: InstanceType<typeof Bun.SQL>,
  failures: SQLFailures,
): Promise<Completion<never> | undefined> {
  try {
    // Construction is lazy; awaiting connect proves establishment. A
    // refused file surfaces here as SQLITE_CANTOPEN.
    await client.connect();
  } catch {
    return failConnect(client, failures);
  }
  return undefined;
}

async function enforceForeignKeys(
  client: InstanceType<typeof Bun.SQL>,
  failures: SQLFailures,
): Promise<Completion<never> | undefined> {
  try {
    // Foreign keys are per-connection and default off, so every
    // constructor enables them before any application query or
    // transaction. safeIntegers delivers the flag as 1n; any other
    // read-back fails closed instead of returning a lax handle.
    await client(FOREIGN_KEYS_ON);
    const back = await client(FOREIGN_KEYS_READ);
    const row = (back as readonly unknown[])[0] as { foreign_keys?: unknown } | undefined;
    if (row?.foreign_keys !== 1n) return failConnect(client, failures);
  } catch {
    return failConnect(client, failures);
  }
  return undefined;
}

export async function openSqliteMemory(
  failures: SQLFailures,
): Promise<
  { ok: true; client: InstanceType<typeof Bun.SQL> } | { ok: false; failure: Completion<never> }
> {
  const client = new Bun.SQL({ adapter: "sqlite", filename: ":memory:", safeIntegers: true });
  const failed = await established(client, failures);
  if (failed !== undefined) return { ok: false, failure: failed };
  const keys = await enforceForeignKeys(client, failures);
  if (keys !== undefined) return { ok: false, failure: keys };
  return { ok: true, client };
}

export async function openSqliteFile(
  config: SQLiteFileConfig,
  failures: SQLFailures,
): Promise<
  { ok: true; client: InstanceType<typeof Bun.SQL> } | { ok: false; failure: Completion<never> }
> {
  const client = new Bun.SQL({
    adapter: "sqlite",
    filename: config.filename,
    safeIntegers: true,
    readonly: config.mode === "ro",
    create: config.mode === "rwc",
  });
  const failed = await established(client, failures);
  if (failed !== undefined) return { ok: false, failure: failed };
  const keys = await enforceForeignKeys(client, failures);
  if (keys !== undefined) return { ok: false, failure: keys };
  // PRAGMA values cannot bind (SQLite rejects placeholders there), so the
  // validated millisecond count travels as canonical digits inside one
  // static template through the tag call: never string-call, never unsafe.
  // A read-only handle accepts the pragma; failure still closes the pool.
  const pragma = `PRAGMA busy_timeout = ${config.busyTimeoutMs}`;
  try {
    await client(staticTemplate(pragma));
  } catch {
    return { ok: false, failure: await failConnect(client, failures) };
  }
  return { ok: true, client };
}
