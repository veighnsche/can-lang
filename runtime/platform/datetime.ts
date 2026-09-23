// Instants, zoned formatting, and civil-to-instant resolution. An instant
// is an opaque handle over exact epoch milliseconds, range-checked to
// the native Date span (±8.64e15) before any Number conversion; local
// civil time and durations never masquerade as instants. Formatting goes
// through Intl with explicit locale and zone (locale default hour cycle,
// pinned ICU), and civil resolution takes an explicit earlier/later
// policy for DST overlaps while gaps reject: nothing here quietly adopts
// native Date normalization. Zone names are whatever the pinned ICU
// accepts; anything else is invalid_zone.
import { success, failure, caught, type Completion, type AssertionContext } from "../completion.ts";
import { record, dataProperty } from "../data.ts";
import { createDomainRuntime } from "../domain.ts";
const origin = Object.freeze({
  source: "can:time",
  start: 0,
  end: 0,
  invocation: Object.freeze([]),
});
const MAX_MILLIS = 8640000000000000n;
const instants = new WeakMap<object, bigint>();
const object = (value: unknown): value is object =>
  value !== null && (typeof value === "object" || typeof value === "function");
export function isTimeInstantValue(kind: string | undefined, value: unknown): boolean {
  return kind === "instant" && object(value) && instants.has(value);
}
const mint = (millis: bigint): object => {
  const handle = Object.freeze(Object.create(null));
  instants.set(handle, millis);
  return handle;
};
// Cross-module mint for adapters that project native timestamps (S3
// stat/list) into time::instant values. Native Dates always land
// inside the instant span, so out-of-range input is an adapter bug.
export function mintTimeInstant(millis: bigint): object {
  if (typeof millis !== "bigint" || millis < -MAX_MILLIS || millis > MAX_MILLIS)
    throw new TypeError("time instant out of range");
  return mint(millis);
}
const millisOf = (handle: unknown): bigint => {
  const millis = object(handle) ? instants.get(handle) : undefined;
  if (millis === undefined) throw new TypeError("invalid compiler time instant");
  return millis;
};
type Style = "full" | "long" | "medium" | "short" | undefined;
const style = (value: unknown, reason: string): Style => {
  if (value === "full" || value === "long" || value === "medium" || value === "short") return value;
  if (value === "none") return undefined;
  throw new Error(reason);
};
type Civil = {
  year: number;
  month: number;
  day: number;
  hour: number;
  minute: number;
  second: number;
  millisecond: number;
};
const civilOf = (value: unknown): Civil => {
  const field = (name: string): unknown => {
    try {
      return dataProperty(value, name);
    } catch {
      throw new TypeError("invalid compiler civil time");
    }
  };
  const out: Record<string, number> = {};
  for (const name of ["year", "month", "day", "hour", "minute", "second", "millisecond"]) {
    const entry = field(name);
    if (typeof entry !== "bigint") throw new TypeError("invalid compiler civil time");
    out[name] = Number(entry);
  }
  return out as Civil;
};
const civilValid = (civil: Civil): boolean =>
  Number.isInteger(civil.year) &&
  civil.year >= -271821 &&
  civil.year <= 275760 &&
  civil.month >= 1 &&
  civil.month <= 12 &&
  civil.day >= 1 &&
  civil.day <= 31 &&
  civil.hour >= 0 &&
  civil.hour <= 23 &&
  civil.minute >= 0 &&
  civil.minute <= 59 &&
  civil.second >= 0 &&
  civil.second <= 59 &&
  civil.millisecond >= 0 &&
  civil.millisecond <= 999;
const partsFormatter = (zone: string) =>
  new Intl.DateTimeFormat("en-US", {
    timeZone: zone,
    hourCycle: "h23",
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });
const zonedParts = (zone: string, instant: number): Civil | undefined => {
  try {
    const entries = partsFormatter(zone).formatToParts(new Date(instant));
    const get = (type: string): number => Number(entries.find((part) => part.type === type)?.value);
    return {
      year: get("year"),
      month: get("month"),
      day: get("day"),
      hour: get("hour"),
      minute: get("minute"),
      second: get("second"),
      millisecond: 0,
    };
  } catch {
    return undefined;
  }
};
const offsetAt = (zone: string, instant: number): number | undefined => {
  const parts = zonedParts(zone, instant);
  if (parts === undefined) return undefined;
  return utcMillis({ ...parts, millisecond: 0 }) - instant;
};
const sameSecond = (left: Civil, right: Civil): boolean =>
  left.year === right.year &&
  left.month === right.month &&
  left.day === right.day &&
  left.hour === right.hour &&
  left.minute === right.minute &&
  left.second === right.second;
// Date.UTC maps years 0..99 to 19xx; civil years are literal.
const utcMillis = (civil: Civil): number => {
  const guess = Date.UTC(
    civil.year,
    civil.month - 1,
    civil.day,
    civil.hour,
    civil.minute,
    civil.second,
    civil.millisecond,
  );
  if (civil.year >= 0 && civil.year <= 99) return new Date(guess).setUTCFullYear(civil.year);
  return guess;
};
export function createDateTimes(
  domain: ReturnType<typeof createDomainRuntime>,
  ids: { outOfRange: string; invalidZone: string; nonexistent: string; invalidOption: string },
) {
  const outOfRange = (millis: bigint) =>
    failure(domain.create(ids.outOfRange, record(ids.outOfRange, [["millis", millis]]), origin));
  const badZone = (zone: string) =>
    failure(domain.create(ids.invalidZone, record(ids.invalidZone, [["zone", zone]]), origin));
  const nonexistent = () =>
    failure(domain.create(ids.nonexistent, record(ids.nonexistent, []), origin));
  const badOption = (reason: string) =>
    failure(
      domain.create(ids.invalidOption, record(ids.invalidOption, [["reason", reason]]), origin),
    );
  return Object.freeze({
    async instantFromEpochMillis(
      millis: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<object>> {
      if (typeof millis !== "bigint") throw new TypeError("invalid compiler epoch millis");
      if (millis < -MAX_MILLIS || millis > MAX_MILLIS) return outOfRange(millis);
      return success(mint(millis));
    },
    async instantEpochMillis(
      handle: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<bigint>> {
      return success(millisOf(handle));
    },
    async formatInZone(
      handle: unknown,
      locale: unknown,
      zone: unknown,
      dateStyle: unknown,
      timeStyle: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<string>> {
      const millis = millisOf(handle);
      if (typeof locale !== "string") throw new TypeError("invalid compiler locale");
      if (typeof zone !== "string") throw new TypeError("invalid compiler zone");
      let date: Style, time: Style;
      try {
        date = style(dateStyle, "date_style");
      } catch {
        return badOption("date_style");
      }
      try {
        time = style(timeStyle, "time_style");
      } catch {
        return badOption("time_style");
      }
      try {
        const formatter = new Intl.DateTimeFormat(locale, {
          timeZone: zone,
          dateStyle: date,
          timeStyle: time,
        });
        return success(formatter.format(new Date(Number(millis))));
      } catch (cause) {
        if (cause instanceof RangeError) {
          try {
            new Intl.DateTimeFormat("en-US", { timeZone: zone });
          } catch {
            return badZone(zone);
          }
          return badOption("locale");
        }
        return caught(cause, origin);
      }
    },
    async resolveZonedTime(
      civil: unknown,
      zone: unknown,
      policy: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<object>> {
      const parts = civilOf(civil);
      if (typeof zone !== "string") throw new TypeError("invalid compiler zone");
      if (typeof policy !== "bigint" || (policy !== 0n && policy !== 1n))
        return badOption("policy");
      if (!civilValid(parts)) return badOption("parts");
      try {
        new Intl.DateTimeFormat("en-US", { timeZone: zone });
      } catch {
        return badZone(zone);
      }
      // The parts formatter reads whole seconds, so milliseconds travel
      // outside the offset arithmetic and reattach to the verified pick:
      // otherwise the truncated formatter reading would skew sub-second
      // inputs by their own millisecond field.
      const base = utcMillis({ ...parts, millisecond: 0 });
      if (!Number.isFinite(base)) return badOption("parts");
      // Collect the distinct zone offsets around the civil time: any
      // transition that could make this time ambiguous or missing falls
      // inside the window. Every candidate is VERIFIED by formatting back;
      // one that does not round-trip is a gap, never silently kept.
      const offsets = new Set<number>();
      for (const probe of [base - 86400000, base, base + 86400000]) {
        const off = offsetAt(zone, probe);
        if (off !== undefined) offsets.add(off);
      }
      const candidates: number[] = [];
      for (const off of offsets) {
        const candidate = base - off;
        const back = zonedParts(zone, candidate);
        if (back !== undefined && sameSecond(back, parts)) candidates.push(candidate);
      }
      if (candidates.length === 0) return nonexistent();
      candidates.sort((a, b) => a - b);
      const pick =
        candidates.length === 1
          ? candidates[0]!
          : policy === 0n
            ? candidates[0]!
            : candidates[candidates.length - 1]!;
      const millis = BigInt(pick) + BigInt(parts.millisecond);
      if (millis < -MAX_MILLIS || millis > MAX_MILLIS) return outOfRange(millis);
      return success(mint(millis));
    },
  });
}
