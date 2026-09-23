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
  const count = (result as { count?: unknown } | null)?.count;
  if (typeof count !== "number" || !Number.isSafeInteger(count) || count < 0) {
    return failures.queryFailed(operation, "bad_count");
  }
  return success(BigInt(count));
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
    try {
      await client.close();
    } catch {
      /* already failed; report the connection */
    }
    return failures.connectionFailed("connect");
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
  // PRAGMA values cannot bind (SQLite rejects placeholders there), so the
  // validated millisecond count travels as canonical digits inside one
  // static template through the tag call: never string-call, never unsafe.
  // A read-only handle accepts the pragma; failure still closes the pool.
  const pragma = `PRAGMA busy_timeout = ${config.busyTimeoutMs}`;
  const strings = Object.freeze(
    Object.assign([pragma], { raw: Object.freeze([pragma]) }),
  ) as unknown as TemplateStringsArray;
  try {
    await client(strings);
  } catch {
    try {
      await client.close();
    } catch {
      /* already failed; report the connection */
    }
    return { ok: false, failure: failures.connectionFailed("connect") };
  }
  return { ok: true, client };
}
