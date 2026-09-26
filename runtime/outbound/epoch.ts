// R14 fixed-epoch schedule configuration and transitions (F01).
//
// Trusted server configuration supplies, per budget pool, a token
// limit, a fixed period length in milliseconds, and a UTC epoch anchor.
// The ledger's authoritative clock determines the half-open epoch
// [start, end) containing admission. Policy edits apply to future
// epochs only: every epoch row pins its start, end, limit, and schedule
// version, and an existing epoch never changes.
//
// Fixed epochs are an allowance per configured period, not a rolling
// rate limit: a caller can consume two allowances on either side of a
// boundary. That distinction is intentional and H/E documentation must
// preserve it.
import { poolId, type PoolId } from "./identity.ts";

export type EpochSchedule = Readonly<{
  pool: PoolId;
  limit: number;
  periodMs: number;
  anchorMs: number;
  version: string;
}>;

export type EpochBounds = Readonly<{ start: number; end: number }>;

const VERSION_PATTERN = /^[A-Za-z0-9][A-Za-z0-9._:-]{0,63}$/;

function checkTokens(name: string, value: unknown, minimum: number): number {
  if (!Number.isSafeInteger(value) || (value as number) < minimum)
    throw new TypeError(`invalid epoch ${name}`);
  return value as number;
}

function checkVersion(value: unknown): string {
  if (typeof value !== "string" || !VERSION_PATTERN.test(value))
    throw new TypeError("invalid epoch version");
  return value;
}

export function epochSchedule(init: {
  pool: unknown;
  limit: unknown;
  periodMs: unknown;
  anchorMs: unknown;
  version: unknown;
}): EpochSchedule {
  return Object.freeze({
    pool: poolId(init.pool),
    limit: checkTokens("limit", init.limit, 0),
    periodMs: checkTokens("period", init.periodMs, 1),
    anchorMs: checkTokens("anchor", init.anchorMs, 0),
    version: checkVersion(init.version),
  });
}

// epochContaining returns the half-open epoch holding nowMs, or
// undefined when nowMs predates the schedule anchor (no epoch exists
// yet). nowMs is UTC milliseconds from the ledger's authoritative
// clock.
export function epochContaining(schedule: EpochSchedule, nowMs: number): EpochBounds | undefined {
  checkTokens("clock", nowMs, 0);
  if (nowMs < schedule.anchorMs) return undefined;
  const elapsed = nowMs - schedule.anchorMs;
  const start = schedule.anchorMs + Math.floor(elapsed / schedule.periodMs) * schedule.periodMs;
  const end = start + schedule.periodMs;
  if (!Number.isSafeInteger(end)) throw new TypeError("epoch end unrepresentable");
  return Object.freeze({ start, end });
}

export type ScheduleChange = Readonly<{
  limit?: unknown;
  periodMs?: unknown;
  anchorMs?: unknown;
  version: unknown;
}>;

// transitionSchedule validates a period/anchor/limit change. The new
// schedule becomes effective only at a boundary of the old schedule,
// and that boundary is the new schedule's anchor: overlapping or
// retroactive transitions are TypeErrors, as is reusing the current
// version. Omitted fields inherit the current schedule; the pool never
// changes. Epoch rows created under the old schedule keep their pinned
// start, end, and limit.
export function transitionSchedule(
  current: EpochSchedule,
  change: ScheduleChange,
  nowMs: number,
): EpochSchedule {
  checkTokens("clock", nowMs, 0);
  const version = checkVersion(change.version);
  if (version === current.version) throw new TypeError("schedule version must advance");
  const limit = change.limit === undefined ? current.limit : checkTokens("limit", change.limit, 0);
  const periodMs =
    change.periodMs === undefined ? current.periodMs : checkTokens("period", change.periodMs, 1);
  let boundary: number;
  if (nowMs < current.anchorMs) {
    boundary = current.anchorMs;
  } else {
    const elapsed = nowMs - current.anchorMs;
    const offset = Math.floor(elapsed / current.periodMs);
    boundary =
      elapsed % current.periodMs === 0 ? nowMs : current.anchorMs + (offset + 1) * current.periodMs;
  }
  if (!Number.isSafeInteger(boundary)) throw new TypeError("schedule boundary unrepresentable");
  let anchorMs = boundary;
  if (change.anchorMs !== undefined) {
    anchorMs = checkTokens("anchor", change.anchorMs, 0);
    if (
      anchorMs < boundary ||
      anchorMs < current.anchorMs ||
      (anchorMs - current.anchorMs) % current.periodMs !== 0
    )
      throw new TypeError("schedule anchor must be a future boundary of the old schedule");
  }
  return Object.freeze({ pool: current.pool, limit, periodMs, anchorMs, version });
}
