// R14 token-ledger SQL backend: LedgerStore over SQLite, PostgreSQL,
// and MySQL (F04).
//
// Native service storage behind ai_budget admission, over the
// operator-owned schema in ledger-schema.ts. Short native operations
// only: load() reads one consistent snapshot, commit() rewrites the
// state in one transaction. Never a transaction or connection lease
// held across provider I/O — the service computes between the two
// calls, never inside them.
//
// Concurrency contract:
// - load() must see one atomic (version, tables) pair: a torn read
//   (new version with old tables) would let a later commit silently
//   drop another writer's rows. Every load therefore runs in a
//   repeatable-read transaction: PostgreSQL issues SET TRANSACTION
//   first, MySQL relies on its REPEATABLE-READ default (verified at
//   open; drift refuses the store), SQLite reads under its deferred
//   snapshot. Never lower MySQL below REPEATABLE-READ.
// - commit() runs one transaction that first bumps the version row
//   conditionally (UPDATE ... WHERE version = ?). Zero affected rows
//   means a lost race: roll back and report { committed: false } so
//   the service retries from a fresh load. The version bump is always
//   the first statement, so concurrent writers serialize on that one
//   row lock in a fixed order and cannot deadlock.
// - A commit that throws after the write is commit-unknown per the
//   LedgerStore contract: the caller reconciles by invocation ID and
//   never dispatches on the ambiguous reservation.
//
// Failure posture: missing tables, unreachable servers after open,
// and corrupt rows fail closed (load/commit throw; the service
// reports storage-unavailable). The store never creates schema —
// operators apply ledgerDDL() themselves (see the F04 operator-DDL
// recipe); the exported applyLedgerDDL() helper runs the same
// statements for disposable test databases only. Thrown errors carry
// static text plus the dialect, never URLs, credentials, or row data.
import { epochSchedule, type EpochSchedule } from "./epoch.ts";
import {
  correlationId,
  invocationId,
  poolId,
  tenantId,
  type CorrelationId,
  type InvocationId,
  type PoolId,
  type TenantId,
} from "./identity.ts";
import {
  epochRowKey,
  meteringProfile,
  profileId,
  profileKey,
  type EpochRow,
  type InvocationRecord,
  type InvocationState,
  type LedgerCommit,
  type LedgerState,
  type LedgerStore,
  type MeteringProfile,
  type Usage,
} from "./ledger.ts";
import { LEDGER_STATEMENTS, type LedgerDialect } from "./ledger-schema.ts";

type Client = InstanceType<typeof Bun.SQL>;
type Query = (strings: TemplateStringsArray, ...values: unknown[]) => Promise<unknown>;

export type SqlLedgerTarget =
  | Readonly<{ dialect: "sqlite"; filename: string }>
  | Readonly<{ dialect: "postgres"; url: string }>
  | Readonly<{ dialect: "mysql"; url: string }>;

export type SqlLedgerStore = LedgerStore & { close(): Promise<void> };

// RollbackSentinel aborts a commit transaction on a lost version race.
// It is private to this module and matched by identity outside begin;
// any other throw propagates as a commit-unknown failure.
class RollbackSentinel {
  private readonly marker = true;
  static is(value: unknown): value is RollbackSentinel {
    return value instanceof RollbackSentinel && value.marker;
  }
}

// createMutex serializes same-handle operations one at a time. Only the
// SQLite store uses it: one Bun.SQL sqlite client is one connection,
// so concurrent begin() calls fail instead of queueing. The mutex is
// held only across one short load()/commit() call, never across
// service computation or provider I/O. Cross-handle races (including
// cross-process) still meet at the version gate, not here.
function createMutex(): <T>(body: () => Promise<T>) => Promise<T> {
  let tail: Promise<void> = Promise.resolve();
  return async <T>(body: () => Promise<T>): Promise<T> => {
    const previous = tail;
    let release: () => void = () => undefined;
    tail = new Promise<void>((resolve) => {
      release = resolve;
    });
    await previous;
    try {
      return await body();
    } finally {
      release();
    }
  };
}

function templateOf(texts: readonly string[]): TemplateStringsArray {
  const parts = [...texts];
  return Object.freeze(
    Object.assign(parts, { raw: Object.freeze([...parts]) }),
  ) as unknown as TemplateStringsArray;
}

function runStatic(query: Query, text: string): Promise<unknown> {
  return query(templateOf([text]));
}

// toSafeInt normalizes the three drivers' integer rendering to a Can
// integer: PostgreSQL/SQLite BIGINT arrive as bigint; MySQL BIGINT
// arrives as bigint above 2^53 and number below. Anything outside the
// safe-integer range, or any other type, is corruption: fail closed.
function toSafeInt(value: unknown, what: string): number {
  if (typeof value === "number") {
    if (Number.isSafeInteger(value)) return value;
  } else if (typeof value === "bigint") {
    if (value >= BigInt(Number.MIN_SAFE_INTEGER) && value <= BigInt(Number.MAX_SAFE_INTEGER))
      return Number(value);
  }
  throw new Error(`ledger SQL row corrupt: ${what}`);
}

function toNonNegativeInt(value: unknown, what: string): number {
  const parsed = toSafeInt(value, what);
  if (parsed < 0) throw new Error(`ledger SQL row corrupt: ${what}`);
  return parsed;
}

function toText(value: unknown, what: string): string {
  if (typeof value !== "string") throw new Error(`ledger SQL row corrupt: ${what}`);
  return value;
}

function toFlag(value: unknown, what: string): boolean {
  const parsed = toSafeInt(value, what);
  if (parsed !== 0 && parsed !== 1) throw new Error(`ledger SQL row corrupt: ${what}`);
  return parsed === 1;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return value !== null && typeof value === "object" && !Array.isArray(value);
}

function rowsOf(result: unknown, what: string): Record<string, unknown>[] {
  if (!Array.isArray(result) || !result.every(isRecord))
    throw new Error(`ledger SQL result corrupt: ${what}`);
  return result;
}

// affectedRows mirrors the runtime's own per-dialect mapping:
// PostgreSQL/SQLite report count, MySQL reports affectedRows.
// Anything else is a driver-shape defect and fails closed.
function affectedRows(dialect: LedgerDialect, result: unknown, what: string): number {
  const field = dialect === "mysql" ? "affectedRows" : "count";
  // Update results are row arrays carrying the count as an extra key,
  // so plain arrays qualify here (unlike row decoding in rowsOf).
  const count =
    typeof result === "object" && result !== null
      ? (result as Record<string, unknown>)[field]
      : undefined;
  if (typeof count !== "number" || !Number.isSafeInteger(count) || count < 0)
    throw new Error(`ledger SQL result corrupt: ${what}`);
  return count;
}

function checkTarget(target: SqlLedgerTarget): void {
  if (!isRecord(target)) throw new TypeError("invalid ledger target");
  if (target["dialect"] === "sqlite") {
    const filename = target["filename"];
    if (typeof filename !== "string" || filename === "" || filename.includes("\0"))
      throw new TypeError("invalid ledger target");
    return;
  }
  if (target["dialect"] === "postgres" || target["dialect"] === "mysql") {
    if (typeof target["url"] !== "string" || target["url"] === "")
      throw new TypeError("invalid ledger target");
    return;
  }
  throw new TypeError("invalid ledger target");
}

async function closeQuietly(client: Client): Promise<void> {
  try {
    await client.close();
  } catch {
    /* already failed; report the open */
  }
}

async function openClient(target: SqlLedgerTarget): Promise<Client> {
  checkTarget(target);
  if (target.dialect === "sqlite") {
    const client = new Bun.SQL({
      adapter: "sqlite",
      filename: target.filename,
      safeIntegers: true,
      create: true,
      readonly: false,
    });
    try {
      await client.connect();
      // Contended writers wait inside the backend instead of
      // surfacing SQLITE_BUSY as a commit-unknown outcome: the
      // bounded wait preserves exactly-once commit semantics. The
      // count travels as canonical digits; PRAGMA rejects
      // placeholders.
      await runStatic(client as Query, "PRAGMA busy_timeout = 5000");
    } catch {
      await closeQuietly(client);
      throw new Error("ledger sqlite unreachable");
    }
    return client;
  }
  if (target.dialect === "postgres") {
    const client = new Bun.SQL(target.url, { adapter: "postgres", bigint: true, max: 8 });
    try {
      await client.connect();
    } catch {
      await closeQuietly(client);
      throw new Error("ledger postgres unreachable");
    }
    return client;
  }
  const client = new Bun.SQL(target.url, { adapter: "mysql", bigint: true, max: 8, tls: true });
  try {
    await client.connect();
  } catch {
    await closeQuietly(client);
    throw new Error("ledger mysql unreachable");
  }
  // The torn-read invariant needs REPEATABLE-READ load snapshots;
  // refuse the store if server configuration drifted below it.
  let isolation: unknown;
  try {
    const found = rowsOf(
      await runStatic(client as Query, "SELECT @@transaction_isolation AS iso"),
      "iso",
    );
    isolation = found.length === 1 ? found[0]["iso"] : undefined;
  } catch {
    isolation = undefined;
  }
  if (isolation !== "REPEATABLE-READ") {
    await closeQuietly(client);
    throw new Error("ledger mysql isolation drifted below REPEATABLE-READ");
  }
  return client;
}

function decodeSchedules(
  rows: Record<string, unknown>[],
): Record<string, readonly EpochSchedule[]> {
  const grouped: Record<string, EpochSchedule[]> = {};
  for (const row of rows) {
    const schedule = epochSchedule({
      pool: toText(row["pool"], "schedule.pool"),
      limit: toNonNegativeInt(row["token_limit"], "schedule.token_limit"),
      periodMs: toSafeInt(row["period_ms"], "schedule.period_ms"),
      anchorMs: toNonNegativeInt(row["anchor_ms"], "schedule.anchor_ms"),
      version: toText(row["version"], "schedule.version"),
    });
    const key = schedule.pool as string;
    const group = grouped[key] ?? [];
    group.push(schedule);
    grouped[key] = group;
  }
  const frozen: Record<string, readonly EpochSchedule[]> = {};
  for (const [pool, group] of Object.entries(grouped)) {
    group.sort((a, b) => a.anchorMs - b.anchorMs);
    frozen[pool] = Object.freeze(group);
  }
  return Object.freeze(frozen);
}

function decodeEpochs(rows: Record<string, unknown>[]): Record<string, EpochRow> {
  const epochs: Record<string, EpochRow> = {};
  for (const row of rows) {
    const tenant: TenantId = tenantId(toText(row["tenant"], "epoch.tenant"));
    const pool: PoolId = poolId(toText(row["pool"], "epoch.pool"));
    const start = toNonNegativeInt(row["epoch_start"], "epoch.epoch_start");
    const epoch: EpochRow = Object.freeze({
      tenant,
      pool,
      start,
      end: toSafeInt(row["epoch_end"], "epoch.epoch_end"),
      limit: toNonNegativeInt(row["token_limit"], "epoch.token_limit"),
      scheduleVersion: toText(row["schedule_version"], "epoch.schedule_version"),
      committed: toNonNegativeInt(row["committed"], "epoch.committed"),
      held: toNonNegativeInt(row["held"], "epoch.held"),
    });
    epochs[epochRowKey(tenant, pool, start)] = epoch;
  }
  return Object.freeze(epochs);
}

const INVOCATION_STATES: readonly string[] = Object.freeze([
  "held",
  "fenced",
  "settled",
  "released",
]);

function decodeInvocations(rows: Record<string, unknown>[]): Record<string, InvocationRecord> {
  const invocations: Record<string, InvocationRecord> = {};
  for (const row of rows) {
    const id: InvocationId = invocationId(toText(row["invocation_id"], "invocation.id"));
    const correlation: CorrelationId = correlationId(
      toText(row["correlation"], "invocation.correlation"),
    );
    const profile: MeteringProfile = meteringProfile({
      provider: toText(row["provider"], "invocation.provider"),
      model: toText(row["model"], "invocation.model"),
      version: toText(row["profile_version"], "invocation.profile_version"),
    });
    const state = toText(row["state"], "invocation.state");
    if (!INVOCATION_STATES.includes(state))
      throw new Error("ledger SQL row corrupt: invocation.state");
    const input = row["input_tokens"];
    const output = row["output_tokens"];
    let usage: Usage | undefined;
    if (input !== null || output !== null) {
      if (input === null || output === null)
        throw new Error("ledger SQL row corrupt: invocation.usage");
      usage = Object.freeze({
        inputTokens: toNonNegativeInt(input, "invocation.input_tokens"),
        outputTokens: toNonNegativeInt(output, "invocation.output_tokens"),
      });
    }
    invocations[id as string] = Object.freeze({
      invocationId: id,
      tenant: tenantId(toText(row["tenant"], "invocation.tenant")),
      pool: poolId(toText(row["pool"], "invocation.pool")),
      correlation,
      epochStart: toNonNegativeInt(row["epoch_start"], "invocation.epoch_start"),
      epochEnd: toSafeInt(row["epoch_end"], "invocation.epoch_end"),
      limit: toNonNegativeInt(row["token_limit"], "invocation.token_limit"),
      scheduleVersion: toText(row["schedule_version"], "invocation.schedule_version"),
      profile,
      profileIdentity: profileId(profile),
      upperBound: toSafeInt(row["upper_bound"], "invocation.upper_bound"),
      state: state as InvocationState,
      fenced: toFlag(row["fenced"], "invocation.fenced"),
      ...(usage === undefined ? {} : { usage }),
      breach: toFlag(row["breach"], "invocation.breach"),
    });
  }
  return Object.freeze(invocations);
}

function decodeQuarantine(rows: Record<string, unknown>[]): Record<string, true> {
  const quarantined: Record<string, true> = {};
  for (const row of rows) {
    quarantined[toText(row["profile_key"], "quarantine.profile_key")] = true;
  }
  return Object.freeze(quarantined);
}

async function loadSnapshot(dialect: LedgerDialect, client: Client): Promise<LedgerState> {
  return client.begin(async (native: unknown): Promise<LedgerState> => {
    const tx = native as Query;
    if (dialect === "postgres") {
      // First statement in the transaction: every SELECT below then
      // shares one snapshot, so the (version, tables) pair is atomic.
      await runStatic(tx, "SET TRANSACTION ISOLATION LEVEL REPEATABLE READ");
    }
    const stateRows = rowsOf(
      await runStatic(tx, "SELECT version FROM ai_budget_state WHERE id = 1"),
      "state",
    );
    if (stateRows.length !== 1) throw new Error("ledger SQL row corrupt: state.version");
    const version = toNonNegativeInt(stateRows[0]["version"], "state.version");
    const schedules = decodeSchedules(
      rowsOf(
        await runStatic(
          tx,
          "SELECT pool, token_limit, period_ms, anchor_ms, version FROM ai_budget_schedules",
        ),
        "schedules",
      ),
    );
    const epochs = decodeEpochs(
      rowsOf(
        await runStatic(
          tx,
          "SELECT tenant, pool, epoch_start, epoch_end, token_limit, schedule_version, committed, held FROM ai_budget_epochs",
        ),
        "epochs",
      ),
    );
    const invocations = decodeInvocations(
      rowsOf(
        await runStatic(
          tx,
          "SELECT invocation_id, tenant, pool, correlation, epoch_start, epoch_end, token_limit, schedule_version, provider, model, profile_version, upper_bound, state, fenced, input_tokens, output_tokens, breach FROM ai_budget_invocations",
        ),
        "invocations",
      ),
    );
    const quarantined = decodeQuarantine(
      rowsOf(await runStatic(tx, "SELECT profile_key FROM ai_budget_quarantine"), "quarantine"),
    );
    return Object.freeze({ version, schedules, epochs, invocations, quarantined });
  });
}

async function insertRow(
  tx: Query,
  table: string,
  columns: readonly string[],
  values: readonly unknown[],
): Promise<void> {
  const texts: string[] = [`INSERT INTO ${table} (${columns.join(", ")}) VALUES (`];
  for (let index = 0; index < values.length; index += 1) {
    texts.push(index + 1 < values.length ? ", " : ")");
  }
  await tx(templateOf(texts), ...values);
}

// quarantineDisplay pairs a stored digest key with its human-readable
// profile identity for the operator-visible column. Every quarantine
// entry is written alongside its breaching invocation, so the lookup
// always hits; a missing invocation can only come from out-of-band
// writes, and then the digest itself is the honest placeholder.
function quarantineDisplay(invocations: Record<string, InvocationRecord>, key: string): string {
  for (const record of Object.values(invocations)) {
    if (profileKey(record.profile) === key) return record.profileIdentity;
  }
  return key;
}

async function rewriteState(
  dialect: LedgerDialect,
  client: Client,
  next: LedgerState,
): Promise<void> {
  await client.begin(async (native: unknown): Promise<void> => {
    const tx = native as Query;
    // The conditional bump is always first: losers roll back before
    // touching any table, and concurrent writers serialize on this
    // one row lock in the same order.
    const bumped = await tx(
      templateOf([
        "UPDATE ai_budget_state SET version = version + 1 WHERE id = 1 AND version = ",
        "",
      ]),
      next.version - 1,
    );
    if (affectedRows(dialect, bumped, "state.bump") !== 1) throw new RollbackSentinel();
    await runStatic(tx, "DELETE FROM ai_budget_invocations");
    await runStatic(tx, "DELETE FROM ai_budget_epochs");
    await runStatic(tx, "DELETE FROM ai_budget_schedules");
    await runStatic(tx, "DELETE FROM ai_budget_quarantine");
    for (const versions of Object.values(next.schedules)) {
      for (const schedule of versions) {
        await insertRow(
          tx,
          "ai_budget_schedules",
          ["pool", "token_limit", "period_ms", "anchor_ms", "version"],
          [
            schedule.pool as string,
            schedule.limit,
            schedule.periodMs,
            schedule.anchorMs,
            schedule.version,
          ],
        );
      }
    }
    for (const row of Object.values(next.epochs)) {
      await insertRow(
        tx,
        "ai_budget_epochs",
        [
          "tenant",
          "pool",
          "epoch_start",
          "epoch_end",
          "token_limit",
          "schedule_version",
          "committed",
          "held",
        ],
        [
          row.tenant as string,
          row.pool as string,
          row.start,
          row.end,
          row.limit,
          row.scheduleVersion,
          row.committed,
          row.held,
        ],
      );
    }
    for (const record of Object.values(next.invocations)) {
      await insertRow(
        tx,
        "ai_budget_invocations",
        [
          "invocation_id",
          "tenant",
          "pool",
          "correlation",
          "epoch_start",
          "epoch_end",
          "token_limit",
          "schedule_version",
          "provider",
          "model",
          "profile_version",
          "upper_bound",
          "state",
          "fenced",
          "input_tokens",
          "output_tokens",
          "breach",
        ],
        [
          record.invocationId as string,
          record.tenant as string,
          record.pool as string,
          record.correlation as string,
          record.epochStart,
          record.epochEnd,
          record.limit,
          record.scheduleVersion,
          record.profile.provider,
          record.profile.model,
          record.profile.version,
          record.upperBound,
          record.state,
          record.fenced ? 1 : 0,
          record.usage?.inputTokens ?? null,
          record.usage?.outputTokens ?? null,
          record.breach ? 1 : 0,
        ],
      );
    }
    for (const key of Object.keys(next.quarantined)) {
      await insertRow(
        tx,
        "ai_budget_quarantine",
        ["profile_key", "profile_identity"],
        [key, quarantineDisplay(next.invocations, key)],
      );
    }
  });
}

// openSqlLedgerStore binds a LedgerStore to one SQL database. The
// schema must already exist (operator-applied DDL): missing tables
// fail closed at first load, never auto-create. The store holds one
// native client; close it when done.
export async function openSqlLedgerStore(target: SqlLedgerTarget): Promise<SqlLedgerStore> {
  const dialect: LedgerDialect = target.dialect;
  const client = await openClient(target);
  const label = target.dialect === "sqlite" ? `sqlite:${target.filename}` : `sql:${dialect}`;
  const exclusive = dialect === "sqlite" ? createMutex() : undefined;
  const guarded = <T>(body: () => Promise<T>): Promise<T> =>
    exclusive === undefined ? body() : exclusive(body);
  return {
    name: label,
    load: async (): Promise<LedgerState> => guarded(() => loadSnapshot(dialect, client)),
    commit: async (expectedVersion: number, next: LedgerState): Promise<LedgerCommit> => {
      if (next.version !== expectedVersion + 1) throw new TypeError("ledger version must advance");
      try {
        await guarded(() => rewriteState(dialect, client, next));
      } catch (cause) {
        if (RollbackSentinel.is(cause)) return { committed: false };
        throw cause;
      }
      return { committed: true };
    },
    close: async (): Promise<void> => {
      await client.close();
    },
  };
}

// applyLedgerDDL runs the operator-owned schema script on a disposable
// database: test-harness setup and operator tooling only. It is not
// idempotent — rerunning on a migrated database fails on the existing
// tables rather than altering them. Production DDL stays an explicit
// operator step (psql/mysql CLIs in the F04 recipe).
export async function applyLedgerDDL(target: SqlLedgerTarget): Promise<void> {
  const client = await openClient(target);
  try {
    for (const statement of LEDGER_STATEMENTS[target.dialect]) {
      await runStatic(client as Query, statement);
    }
  } finally {
    await client.close();
  }
}
