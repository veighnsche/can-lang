// R14 durable token ledger: reserve/fence/settle/reconcile (F01).
//
// Native service contract behind ai_budget admission. One tenant
// allowance counts input_tokens + output_tokens, unweighted, per
// (tenant, pool, epoch) row. Before provider I/O the guard atomically
// reserves a conservative upper bound U; settlement debits authoritative
// actuals in the original admission epoch; unknown usage keeps the full
// hold unresolved and durable. No timer, disposal, restart, or empty
// response refunds a hold — only authoritative usage or definitive
// no-dispatch proof reconciles it.
//
// The service is pure orchestration over a LedgerStore: optimistic
// whole-state transactions with bounded retries. Memory and file
// backends ship here; the SQL schema for F04 backend qualification
// lives in ledger-schema.ts. Pointers for H/E: malformed caller input
// is a TypeError (validate trusted context first); every
// state-dependent failure is a typed unavailable outcome H maps to
// ai_budget::exceeded / ai_budget::unavailable. Ledger records and
// reports never carry credentials, endpoints, or prompt text.
import { epochContaining, epochSchedule, transitionSchedule, type EpochSchedule } from "./epoch.ts";
import {
  correlationId,
  identityValue,
  invocationId,
  newCorrelationId,
  newInvocationId,
  poolId,
  tenantId,
  type CorrelationId,
  type InvocationId,
  type PoolId,
  type TenantId,
} from "./identity.ts";

export type MeteringProfile = Readonly<{
  provider: string;
  model: string;
  version: string;
}>;

export function meteringProfile(init: {
  provider: unknown;
  model: unknown;
  version: unknown;
}): MeteringProfile {
  return Object.freeze({
    provider: identityValue("metering provider", init.provider),
    model: identityValue("metering model", init.model),
    version: identityValue("metering version", init.version),
  });
}

// profileId is the pinned provider/model/version display identity: safe
// to log, and the quarantine key H reports on a profile breach.
export function profileId(profile: MeteringProfile): string {
  return `${profile.provider}/${profile.model}/${profile.version}`;
}

function profileKey(profile: MeteringProfile): string {
  return JSON.stringify([profile.provider, profile.model, profile.version]);
}

export type Usage = Readonly<{ inputTokens: number; outputTokens: number }>;

function checkUsageTokens(kind: string, value: unknown): number | undefined {
  if (!Number.isSafeInteger(value) || (value as number) < 0) return undefined;
  return value as number;
}

// meteringUsage validates trustworthy top-level provider counters as
// nonnegative integers. It returns undefined (never throws) because
// the input is provider data: invalid usage keeps the hold unresolved
// under invalid-usage, it is not a caller bug.
export function meteringUsage(inputTokens: unknown, outputTokens: unknown): Usage | undefined {
  const input = checkUsageTokens("input", inputTokens);
  const output = checkUsageTokens("output", outputTokens);
  if (input === undefined || output === undefined) return undefined;
  if (!Number.isSafeInteger(input + output)) return undefined;
  return Object.freeze({ inputTokens: input, outputTokens: output });
}

export function usageTotal(usage: Usage): number {
  return usage.inputTokens + usage.outputTokens;
}

export type InvocationState = "held" | "fenced" | "settled" | "released";

export type InvocationRecord = Readonly<{
  invocationId: InvocationId;
  tenant: TenantId;
  pool: PoolId;
  correlation: CorrelationId;
  epochStart: number;
  epochEnd: number;
  limit: number;
  scheduleVersion: string;
  profile: MeteringProfile;
  profileIdentity: string;
  upperBound: number;
  state: InvocationState;
  fenced: boolean;
  usage?: Usage;
  breach: boolean;
}>;

export type EpochRow = Readonly<{
  tenant: TenantId;
  pool: PoolId;
  start: number;
  end: number;
  limit: number;
  scheduleVersion: string;
  committed: number;
  held: number;
}>;

export type LedgerState = Readonly<{
  version: number;
  // Append-only schedule versions per pool in non-decreasing anchor
  // order: a transition takes effect at a future boundary, so times
  // before that boundary still admit under the superseded schedule.
  schedules: Readonly<Record<string, readonly EpochSchedule[]>>;
  epochs: Readonly<Record<string, EpochRow>>;
  invocations: Readonly<Record<string, InvocationRecord>>;
  quarantined: Readonly<Record<string, true>>;
}>;

export function emptyLedgerState(): LedgerState {
  return Object.freeze({ version: 0, schedules: {}, epochs: {}, invocations: {}, quarantined: {} });
}

function epochRowKey(tenant: TenantId, pool: PoolId, start: number): string {
  return `${tenant as string}\n${pool as string}\n${start}`;
}

export type LedgerCommit = { committed: true } | { committed: false };

// LedgerStore is the durable backend contract. load returns the latest
// persisted state; commit persists next only when the persisted
// version still equals expectedVersion, assigning expectedVersion + 1.
// A commit that throws is commit-unknown: the caller reconciles by
// invocation ID and never dispatches on the ambiguous reservation.
// Backends fail closed (throw) on corruption or unavailability; they
// never auto-reset.
export type LedgerStore = {
  readonly name: string;
  load(): Promise<LedgerState>;
  commit(expectedVersion: number, next: LedgerState): Promise<LedgerCommit>;
};

export function createMemoryLedgerStore(name = "memory"): LedgerStore {
  let current = emptyLedgerState();
  return {
    name,
    load: async (): Promise<LedgerState> => current,
    commit: async (expectedVersion: number, next: LedgerState): Promise<LedgerCommit> => {
      if (current.version !== expectedVersion) return { committed: false };
      if (next.version !== expectedVersion + 1) throw new TypeError("ledger version must advance");
      current = next;
      return { committed: true };
    },
  };
}

export type UnavailableReason =
  | "missing-context"
  | "invalid-policy"
  | "storage-unavailable"
  | "ambiguous-commit"
  | "invalid-usage"
  | "conflicting-settlement"
  | "missing-qualification"
  | "breached-profile"
  | "unknown-invocation";

export type ReserveInput = Readonly<{
  tenant: unknown;
  pool: unknown;
  correlation?: unknown;
  profile: { provider: unknown; model: unknown; version: unknown };
  qualified: boolean;
  upperBound: unknown;
  invocationId?: unknown;
  nowMs?: number;
}>;

export type ReserveOutcome =
  | Readonly<{
      outcome: "admitted";
      record: InvocationRecord;
      remaining: number;
      epochResetMs: number;
      duplicate: boolean;
    }>
  | Readonly<{
      outcome: "exceeded";
      pool: PoolId;
      required: number;
      remaining: number;
      epochResetMs: number;
    }>
  | Readonly<{
      outcome: "unavailable";
      reason: UnavailableReason;
      correlation: CorrelationId;
      invocationId: InvocationId;
    }>;

export type FenceOutcome =
  | Readonly<{ outcome: "fenced"; record: InvocationRecord; duplicate: boolean }>
  | Readonly<{ outcome: "unavailable"; reason: UnavailableReason }>;

export type SettleInput = Readonly<{
  invocationId: unknown;
  inputTokens: unknown;
  outputTokens: unknown;
}>;

export type SettleOutcome =
  | Readonly<{
      outcome: "settled";
      record: InvocationRecord;
      actual: number;
      released: number;
      duplicate: boolean;
      breach?: Readonly<{ profile: string; upperBound: number; actual: number }>;
    }>
  | Readonly<{ outcome: "unavailable"; reason: UnavailableReason }>;

export type ReleaseOutcome =
  | Readonly<{ outcome: "released"; record: InvocationRecord; duplicate: boolean }>
  | Readonly<{ outcome: "held"; record: InvocationRecord }>
  | Readonly<{ outcome: "unavailable"; reason: UnavailableReason }>;

export type ReconcileOutcome =
  | Readonly<{ status: "found"; record: InvocationRecord }>
  | Readonly<{ status: "absent" }>
  | Readonly<{ status: "unavailable"; reason: "storage-unavailable" }>;

export type EpochStatusOutcome =
  | Readonly<{ status: "found"; row: EpochRow; unresolved: number }>
  | Readonly<{ status: "absent" }>
  | Readonly<{ status: "unavailable"; reason: "storage-unavailable" }>;

export type LedgerHealth =
  | Readonly<{
      status: "ok";
      epochs: number;
      committed: number;
      held: number;
      unresolved: number;
      quarantined: readonly string[];
    }>
  | Readonly<{ status: "unavailable"; reason: "storage-unavailable" }>;

function checkUpperBound(value: unknown): number {
  if (!Number.isSafeInteger(value) || (value as number) < 1)
    throw new TypeError("invalid reservation upper bound");
  return value as number;
}

function checkClock(nowMs: number | undefined): number {
  const at = nowMs ?? Date.now();
  if (!Number.isSafeInteger(at) || at < 0) throw new TypeError("invalid ledger clock");
  return at;
}

// fits compares committed + held + upperBound against the limit without
// ever overflowing: each comparison subtracts from a known-safe
// headroom instead of summing unbounded counters.
function fits(limit: number, committed: number, held: number, upperBound: number): boolean {
  if (upperBound > limit) return false;
  const headroom = limit - upperBound;
  if (committed > headroom) return false;
  return held <= headroom - committed;
}

function remainingTokens(limit: number, committed: number, held: number): number {
  return Math.max(0, limit - committed - held);
}

type Applied<T> = { write: boolean; next?: LedgerState; value: T };

async function runTransaction<T>(
  store: LedgerStore,
  apply: (state: LedgerState) => Applied<T>,
): Promise<{ status: "ok"; value: T } | { status: "unavailable" } | { status: "ambiguous" }> {
  let loaded: LedgerState;
  try {
    loaded = await store.load();
  } catch {
    return { status: "unavailable" };
  }
  // Attempts must exceed plausible racers: every lost race reloads
  // and at least one writer retires per round, so bounded contention
  // always converges instead of surfacing storage-unavailable.
  for (let attempt = 0; attempt < 64; attempt += 1) {
    const applied = apply(loaded);
    if (!applied.write) return { status: "ok", value: applied.value };
    try {
      const commit = await store.commit(loaded.version, applied.next!);
      if (commit.committed) return { status: "ok", value: applied.value };
    } catch {
      return { status: "ambiguous" };
    }
    try {
      loaded = await store.load();
    } catch {
      return { status: "unavailable" };
    }
  }
  return { status: "unavailable" };
}

function nextVersion(state: LedgerState): number {
  return state.version + 1;
}

// applicableSchedule selects the newest schedule version whose anchor
// the ledger clock has reached. Times before every anchor are in no
// epoch yet.
function applicableSchedule(
  versions: readonly EpochSchedule[],
  nowMs: number,
): EpochSchedule | undefined {
  let picked: EpochSchedule | undefined;
  for (const version of versions) {
    if (version.anchorMs > nowMs) break;
    picked = version;
  }
  return picked;
}

// reserve atomically admits U only if committed + held + U <= limit in
// the current epoch. Excess returns exceeded immediately — no provider
// I/O, no refill wait. A supplied invocation ID makes reserve
// idempotent: retrying after an ambiguous commit returns the existing
// admission instead of double-charging.
export async function reserve(store: LedgerStore, input: ReserveInput): Promise<ReserveOutcome> {
  const tenant = tenantId(input.tenant);
  const pool = poolId(input.pool);
  const correlation =
    input.correlation === undefined ? newCorrelationId() : correlationId(input.correlation);
  const profile = meteringProfile(input.profile);
  const upperBound = checkUpperBound(input.upperBound);
  const id =
    input.invocationId === undefined ? newInvocationId() : invocationId(input.invocationId);
  const at = checkClock(input.nowMs);
  if (typeof input.qualified !== "boolean" || !input.qualified)
    return Object.freeze({
      outcome: "unavailable",
      reason: "missing-qualification",
      correlation,
      invocationId: id,
    });
  const unavailable = (
    reason: UnavailableReason,
  ): Extract<ReserveOutcome, { outcome: "unavailable" }> =>
    Object.freeze({ outcome: "unavailable", reason, correlation, invocationId: id });

  const run = await runTransaction<ReserveOutcome>(store, (state) => {
    const versions = state.schedules[pool as string];
    const schedule = versions === undefined ? undefined : applicableSchedule(versions, at);
    if (schedule === undefined) return { write: false, value: unavailable("invalid-policy") };
    const epoch = epochContaining(schedule, at);
    if (epoch === undefined) return { write: false, value: unavailable("invalid-policy") };
    const existing = state.invocations[id as string];
    if (existing !== undefined) {
      if (
        existing.tenant !== tenant ||
        existing.pool !== pool ||
        existing.correlation !== correlation ||
        existing.upperBound !== upperBound ||
        existing.profileIdentity !== profileId(profile)
      )
        return { write: false, value: unavailable("conflicting-settlement") };
      const current = state.epochs[epochRowKey(tenant, pool, epoch.start)];
      const remaining =
        current === undefined
          ? schedule.limit
          : remainingTokens(current.limit, current.committed, current.held);
      return {
        write: false,
        value: Object.freeze({
          outcome: "admitted",
          record: existing,
          remaining,
          epochResetMs: epoch.end,
          duplicate: true,
        }),
      };
    }
    if (state.quarantined[profileKey(profile)] === true)
      return { write: false, value: unavailable("breached-profile") };
    const key = epochRowKey(tenant, pool, epoch.start);
    const current: EpochRow =
      state.epochs[key] ??
      Object.freeze({
        tenant,
        pool,
        start: epoch.start,
        end: epoch.end,
        limit: schedule.limit,
        scheduleVersion: schedule.version,
        committed: 0,
        held: 0,
      });
    if (!fits(current.limit, current.committed, current.held, upperBound))
      return {
        write: false,
        value: Object.freeze({
          outcome: "exceeded",
          pool,
          required: upperBound,
          remaining: remainingTokens(current.limit, current.committed, current.held),
          epochResetMs: current.end,
        }),
      };
    const record: InvocationRecord = Object.freeze({
      invocationId: id,
      tenant,
      pool,
      correlation,
      epochStart: current.start,
      epochEnd: current.end,
      limit: current.limit,
      scheduleVersion: current.scheduleVersion,
      profile,
      profileIdentity: profileId(profile),
      upperBound,
      state: "held",
      fenced: false,
      breach: false,
    });
    const row: EpochRow = Object.freeze({ ...current, held: current.held + upperBound });
    return {
      write: true,
      next: Object.freeze({
        version: nextVersion(state),
        schedules: state.schedules,
        epochs: Object.freeze({ ...state.epochs, [key]: row }),
        invocations: Object.freeze({ ...state.invocations, [id as string]: record }),
        quarantined: state.quarantined,
      }),
      value: Object.freeze({
        outcome: "admitted",
        record,
        remaining: remainingTokens(row.limit, row.committed, row.held),
        epochResetMs: row.end,
        duplicate: false,
      }),
    };
  });
  if (run.status === "ok") return run.value;
  return unavailable(run.status === "ambiguous" ? "ambiguous-commit" : "storage-unavailable");
}

// fence persists the once-only dispatch fence before provider I/O.
// Fencing is idempotent; fencing a settled or released attempt is a
// metering failure. A crashed or uncertain fenced attempt is never
// resent on the same reservation.
export async function fence(store: LedgerStore, id: unknown): Promise<FenceOutcome> {
  const key = invocationId(id) as string;
  const run = await runTransaction<FenceOutcome>(store, (state) => {
    const existing = state.invocations[key];
    if (existing === undefined)
      return {
        write: false,
        value: Object.freeze({ outcome: "unavailable", reason: "unknown-invocation" as const }),
      };
    if (existing.state === "settled" || existing.state === "released")
      return {
        write: false,
        value: Object.freeze({ outcome: "unavailable", reason: "conflicting-settlement" as const }),
      };
    if (existing.fenced)
      return {
        write: false,
        value: Object.freeze({ outcome: "fenced", record: existing, duplicate: true }),
      };
    const record: InvocationRecord = Object.freeze({ ...existing, state: "fenced", fenced: true });
    return {
      write: true,
      next: Object.freeze({
        version: nextVersion(state),
        schedules: state.schedules,
        epochs: state.epochs,
        invocations: Object.freeze({ ...state.invocations, [key]: record }),
        quarantined: state.quarantined,
      }),
      value: Object.freeze({ outcome: "fenced", record, duplicate: false }),
    };
  });
  if (run.status === "ok") return run.value;
  return Object.freeze({
    outcome: "unavailable",
    reason: run.status === "ambiguous" ? "ambiguous-commit" : "storage-unavailable",
  });
}

// settle reconciles authoritative actuals in the original admission
// epoch: with A <= U the hold lifts, A debits, and U - A releases.
// Duplicate settlement has no effect; conflicting usage is a metering
// failure. Actuals above U record without clamping, quarantine the
// profile, and report the breach — H invalidates the budgeted result.
// Any invalid or missing usage leaves the hold unresolved via
// invalid-usage; settlement never resolves commit-unknown by guessing.
export async function settle(store: LedgerStore, input: SettleInput): Promise<SettleOutcome> {
  const key = invocationId(input.invocationId) as string;
  const usage = meteringUsage(input.inputTokens, input.outputTokens);
  if (usage === undefined)
    return Object.freeze({ outcome: "unavailable", reason: "invalid-usage" });
  const actual = usageTotal(usage);
  const run = await runTransaction<SettleOutcome>(store, (state) => {
    const existing = state.invocations[key];
    if (existing === undefined)
      return {
        write: false,
        value: Object.freeze({ outcome: "unavailable", reason: "unknown-invocation" as const }),
      };
    if (existing.state === "released")
      return {
        write: false,
        value: Object.freeze({ outcome: "unavailable", reason: "conflicting-settlement" as const }),
      };
    if (existing.state === "settled") {
      const settled = existing.usage!;
      if (settled.inputTokens !== usage.inputTokens || settled.outputTokens !== usage.outputTokens)
        return {
          write: false,
          value: Object.freeze({
            outcome: "unavailable",
            reason: "conflicting-settlement" as const,
          }),
        };
      const settledValue: Extract<SettleOutcome, { outcome: "settled" }> = existing.breach
        ? Object.freeze({
            outcome: "settled",
            record: existing,
            actual,
            released: 0,
            duplicate: true,
            breach: Object.freeze({
              profile: existing.profileIdentity,
              upperBound: existing.upperBound,
              actual,
            }),
          })
        : Object.freeze({
            outcome: "settled",
            record: existing,
            actual,
            released: 0,
            duplicate: true,
          });
      return { write: false, value: settledValue };
    }
    const rowKey = epochRowKey(existing.tenant, existing.pool, existing.epochStart);
    const row = state.epochs[rowKey];
    if (row === undefined)
      return {
        write: false,
        value: Object.freeze({ outcome: "unavailable", reason: "storage-unavailable" as const }),
      };
    const committed = row.committed + actual;
    if (!Number.isSafeInteger(committed))
      return {
        write: false,
        value: Object.freeze({ outcome: "unavailable", reason: "invalid-usage" as const }),
      };
    const breached = actual > existing.upperBound;
    const record: InvocationRecord = Object.freeze({
      ...existing,
      state: "settled",
      usage,
      breach: breached,
    });
    const nextRow: EpochRow = Object.freeze({
      ...row,
      committed,
      held: row.held - existing.upperBound,
    });
    const quarantined =
      breached && state.quarantined[profileKey(existing.profile)] !== true
        ? Object.freeze({ ...state.quarantined, [profileKey(existing.profile)]: true as const })
        : state.quarantined;
    const settledValue: Extract<SettleOutcome, { outcome: "settled" }> = breached
      ? Object.freeze({
          outcome: "settled",
          record,
          actual,
          released: 0,
          duplicate: false,
          breach: Object.freeze({
            profile: existing.profileIdentity,
            upperBound: existing.upperBound,
            actual,
          }),
        })
      : Object.freeze({
          outcome: "settled",
          record,
          actual,
          released: existing.upperBound - actual,
          duplicate: false,
        });
    return {
      write: true,
      next: Object.freeze({
        version: nextVersion(state),
        schedules: state.schedules,
        epochs: Object.freeze({ ...state.epochs, [rowKey]: nextRow }),
        invocations: Object.freeze({ ...state.invocations, [key]: record }),
        quarantined,
      }),
      value: settledValue,
    };
  });
  if (run.status === "ok") return run.value;
  return Object.freeze({
    outcome: "unavailable",
    reason: run.status === "ambiguous" ? "ambiguous-commit" : "storage-unavailable",
  });
}

// releaseUndispatched releases U only when the dispatch fence proves no
// send could have occurred — the attempt was never fenced. A fenced
// attempt keeps its full hold (outcome held); a settled attempt
// conflicts. Release is idempotent.
export async function releaseUndispatched(
  store: LedgerStore,
  id: unknown,
): Promise<ReleaseOutcome> {
  const key = invocationId(id) as string;
  const run = await runTransaction<ReleaseOutcome>(store, (state) => {
    const existing = state.invocations[key];
    if (existing === undefined)
      return {
        write: false,
        value: Object.freeze({ outcome: "unavailable", reason: "unknown-invocation" as const }),
      };
    if (existing.state === "settled")
      return {
        write: false,
        value: Object.freeze({ outcome: "unavailable", reason: "conflicting-settlement" as const }),
      };
    if (existing.state === "released")
      return {
        write: false,
        value: Object.freeze({ outcome: "released", record: existing, duplicate: true }),
      };
    if (existing.fenced)
      return { write: false, value: Object.freeze({ outcome: "held", record: existing }) };
    const rowKey = epochRowKey(existing.tenant, existing.pool, existing.epochStart);
    const row = state.epochs[rowKey];
    if (row === undefined)
      return {
        write: false,
        value: Object.freeze({ outcome: "unavailable", reason: "storage-unavailable" as const }),
      };
    const record: InvocationRecord = Object.freeze({ ...existing, state: "released" });
    const nextRow: EpochRow = Object.freeze({ ...row, held: row.held - existing.upperBound });
    return {
      write: true,
      next: Object.freeze({
        version: nextVersion(state),
        schedules: state.schedules,
        epochs: Object.freeze({ ...state.epochs, [rowKey]: nextRow }),
        invocations: Object.freeze({ ...state.invocations, [key]: record }),
        quarantined: state.quarantined,
      }),
      value: Object.freeze({ outcome: "released", record, duplicate: false }),
    };
  });
  if (run.status === "ok") return run.value;
  return Object.freeze({
    outcome: "unavailable",
    reason: run.status === "ambiguous" ? "ambiguous-commit" : "storage-unavailable",
  });
}

// reconcile reads the durable record for an invocation ID without
// sending or mutating: the recovery path after a crash or an ambiguous
// commit. Absent means no known reservation — never dispatch on it.
export async function reconcile(store: LedgerStore, id: unknown): Promise<ReconcileOutcome> {
  const key = invocationId(id) as string;
  let state: LedgerState;
  try {
    state = await store.load();
  } catch {
    return Object.freeze({ status: "unavailable", reason: "storage-unavailable" });
  }
  const record = state.invocations[key];
  if (record === undefined) return Object.freeze({ status: "absent" });
  return Object.freeze({ status: "found", record });
}

// epochStatus reports one pinned epoch row with its unresolved attempt
// count for health and operator reports. Counts only — no secrets.
export async function epochStatus(
  store: LedgerStore,
  tenant: unknown,
  pool: unknown,
  epochStart: unknown,
): Promise<EpochStatusOutcome> {
  const tenantChecked = tenantId(tenant);
  const poolChecked = poolId(pool);
  if (!Number.isSafeInteger(epochStart) || (epochStart as number) < 0)
    throw new TypeError("invalid epoch start");
  let state: LedgerState;
  try {
    state = await store.load();
  } catch {
    return Object.freeze({ status: "unavailable", reason: "storage-unavailable" });
  }
  const row = state.epochs[epochRowKey(tenantChecked, poolChecked, epochStart as number)];
  if (row === undefined) return Object.freeze({ status: "absent" });
  let unresolved = 0;
  for (const record of Object.values(state.invocations)) {
    if (
      record.tenant === tenantChecked &&
      record.pool === poolChecked &&
      record.epochStart === (epochStart as number) &&
      (record.state === "held" || record.state === "fenced")
    )
      unresolved += 1;
  }
  return Object.freeze({ status: "found", row, unresolved });
}

// ledgerHealth aggregates committed tokens, held tokens, and unresolved
// attempts across epochs for operator reports. Totals saturate at the
// largest safe integer rather than losing precision.
export async function ledgerHealth(store: LedgerStore): Promise<LedgerHealth> {
  let state: LedgerState;
  try {
    state = await store.load();
  } catch {
    return Object.freeze({ status: "unavailable", reason: "storage-unavailable" });
  }
  const add = (total: number, value: number): number => {
    const sum = total + value;
    return Number.isSafeInteger(sum) ? sum : Number.MAX_SAFE_INTEGER;
  };
  let committed = 0;
  let held = 0;
  for (const row of Object.values(state.epochs)) {
    committed = add(committed, row.committed);
    held = add(held, row.held);
  }
  let unresolved = 0;
  for (const record of Object.values(state.invocations)) {
    if (record.state === "held" || record.state === "fenced") unresolved += 1;
  }
  const quarantined = Object.freeze(
    Object.values(state.invocations)
      .filter(
        (record, index, all) =>
          record.breach &&
          all.findIndex((other) => other.profileIdentity === record.profileIdentity) === index,
      )
      .map((record) => record.profileIdentity)
      .sort(),
  );
  return Object.freeze({
    status: "ok",
    epochs: Object.keys(state.epochs).length,
    committed,
    held,
    unresolved,
    quarantined,
  });
}

// configurePool pins the first schedule for a pool. Reconfiguration
// goes through transitionPool; re-pinning here is invalid-policy.
export async function configurePool(
  store: LedgerStore,
  schedule: EpochSchedule,
): Promise<Readonly<{ configured: true } | { configured: false; reason: UnavailableReason }>> {
  const pinned = epochSchedule({
    pool: schedule.pool,
    limit: schedule.limit,
    periodMs: schedule.periodMs,
    anchorMs: schedule.anchorMs,
    version: schedule.version,
  });
  const run = await runTransaction<
    Readonly<{ configured: true } | { configured: false; reason: UnavailableReason }>
  >(store, (state) => {
    if (state.schedules[pinned.pool as string] !== undefined)
      return {
        write: false,
        value: Object.freeze({ configured: false, reason: "invalid-policy" as const }),
      };
    return {
      write: true,
      next: Object.freeze({
        version: nextVersion(state),
        schedules: Object.freeze({
          ...state.schedules,
          [pinned.pool as string]: Object.freeze([pinned]),
        }),
        epochs: state.epochs,
        invocations: state.invocations,
        quarantined: state.quarantined,
      }),
      value: Object.freeze({ configured: true }),
    };
  });
  if (run.status === "ok") return run.value;
  return Object.freeze({
    configured: false,
    reason: run.status === "ambiguous" ? "ambiguous-commit" : "storage-unavailable",
  });
}

// transitionPool moves a pool to a new schedule under the epoch
// transition rules: effective at a boundary of the old schedule, never
// overlapping or retroactive, with an advanced version. Existing epoch
// rows keep their pinned start, end, and limit; only future epochs use
// the new schedule.
export async function transitionPool(
  store: LedgerStore,
  pool: unknown,
  change: { limit?: unknown; periodMs?: unknown; anchorMs?: unknown; version: unknown },
  nowMs?: number,
): Promise<
  | Readonly<{ transitioned: true; schedule: EpochSchedule }>
  | Readonly<{ transitioned: false; reason: UnavailableReason }>
> {
  const key = poolId(pool) as string;
  const at = checkClock(nowMs);
  const run = await runTransaction<
    | Readonly<{ transitioned: true; schedule: EpochSchedule }>
    | Readonly<{ transitioned: false; reason: UnavailableReason }>
  >(store, (state) => {
    const versions = state.schedules[key];
    if (versions === undefined || versions.length === 0)
      return {
        write: false,
        value: Object.freeze({ transitioned: false, reason: "invalid-policy" as const }),
      };
    const schedule = transitionSchedule(versions[versions.length - 1], change, at);
    return {
      write: true,
      next: Object.freeze({
        version: nextVersion(state),
        schedules: Object.freeze({
          ...state.schedules,
          [key]: Object.freeze([...versions, schedule]),
        }),
        epochs: state.epochs,
        invocations: state.invocations,
        quarantined: state.quarantined,
      }),
      value: Object.freeze({ transitioned: true, schedule }),
    };
  });
  if (run.status === "ok") return run.value;
  return Object.freeze({
    transitioned: false,
    reason: run.status === "ambiguous" ? "ambiguous-commit" : "storage-unavailable",
  });
}
