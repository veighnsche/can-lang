// PostgreSQL-specific native mapping: client establishment, native
// failure classification, and statement metadata. Only PostgresError
// classifies; anything else is a driver defect and propagates to the
// generic fault path. Sanitized fields only: never message, detail,
// options, URL, or credentials.
import { success, type Completion } from "../../completion.ts";
import type { SQLCoreContracts, SQLFailures } from "./errors.ts";

export function isPostgresFailure(
  value: unknown,
): value is Error & { code: string; errno?: unknown; constraint?: unknown } {
  return (
    value instanceof Error &&
    value.name === "PostgresError" &&
    typeof (value as { code?: unknown }).code === "string"
  );
}

export function classifyPostgres(
  operation: string,
  cause: unknown,
  failures: SQLFailures,
  contracts: SQLCoreContracts,
): Completion<never> {
  if (!isPostgresFailure(cause)) throw cause;
  if (
    cause.code === "ERR_POSTGRES_CONNECTION_REFUSED" ||
    cause.code === "ERR_POSTGRES_CONNECTION_CLOSED"
  ) {
    return failures.connectionFailed("query");
  }
  const errno = typeof cause.errno === "string" ? cause.errno : "";
  const constraint = typeof cause.constraint === "string" ? cause.constraint : "";
  if (constraint !== "" || (errno.length === 5 && errno.startsWith("23"))) {
    return failures.fail(contracts.constraintFailed, [
      ["constraint", constraint !== "" ? constraint : errno],
    ]);
  }
  return failures.queryFailed(
    operation,
    errno !== "" ? errno : cause.code !== "" ? cause.code : "unknown",
  );
}

export function postgresAffectedRows(
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

export async function openPostgresClient(
  url: string,
  max: number,
  failures: SQLFailures,
): Promise<
  { ok: true; client: InstanceType<typeof Bun.SQL> } | { ok: false; failure: Completion<never> }
> {
  const client = new Bun.SQL(url, { adapter: "postgres", bigint: true, max });
  try {
    // Construction is lazy; awaiting connect proves establishment rather
    // than merely building a client. Any refusal surfaces here.
    await client.connect();
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
