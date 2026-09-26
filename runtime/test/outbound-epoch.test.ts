import { test, expect } from "bun:test";
import { epochContaining, epochSchedule, transitionSchedule } from "../outbound/epoch.ts";

const HOUR = 3_600_000;

function hourly(limit = 1000): ReturnType<typeof epochSchedule> {
  return epochSchedule({ pool: "default", limit, periodMs: HOUR, anchorMs: 0, version: "1" });
}

test("schedules pin pool, limit, period, anchor, and version", () => {
  const schedule = hourly();
  expect(schedule.pool as string).toBe("default");
  expect(schedule.limit).toBe(1000);
  expect(schedule.periodMs).toBe(HOUR);
  expect(schedule.anchorMs).toBe(0);
  expect(schedule.version).toBe("1");
  expect(Object.isFrozen(schedule)).toBe(true);
  // Zero limit is a valid closed pool; anything unrepresentable is not.
  expect(hourly(0).limit).toBe(0);
  for (const init of [
    { pool: "", limit: 1, periodMs: HOUR, anchorMs: 0, version: "1" },
    { pool: "p", limit: -1, periodMs: HOUR, anchorMs: 0, version: "1" },
    { pool: "p", limit: 1.5, periodMs: HOUR, anchorMs: 0, version: "1" },
    { pool: "p", limit: 1, periodMs: 0, anchorMs: 0, version: "1" },
    { pool: "p", limit: 1, periodMs: -5, anchorMs: 0, version: "1" },
    { pool: "p", limit: 1, periodMs: HOUR, anchorMs: -1, version: "1" },
    { pool: "p", limit: 1, periodMs: HOUR, anchorMs: 0, version: "" },
    { pool: "p", limit: 1, periodMs: HOUR, anchorMs: 0, version: "has space" },
  ]) {
    expect(() => epochSchedule(init)).toThrow(TypeError);
  }
});

test("admission lands in the half-open epoch containing the ledger clock", () => {
  const schedule = hourly();
  expect(epochContaining(schedule, 0)).toEqual({ start: 0, end: HOUR });
  // The end bound is exclusive: exactly HOUR opens the next epoch.
  expect(epochContaining(schedule, HOUR - 1)).toEqual({ start: 0, end: HOUR });
  expect(epochContaining(schedule, HOUR)).toEqual({ start: HOUR, end: 2 * HOUR });
  expect(epochContaining(schedule, 2 * HOUR + 7)).toEqual({ start: 2 * HOUR, end: 3 * HOUR });
  // A non-zero anchor shifts every boundary with it.
  const anchored = epochSchedule({
    pool: "p",
    limit: 1,
    periodMs: 100,
    anchorMs: 50,
    version: "1",
  });
  expect(epochContaining(anchored, 49)).toBeUndefined();
  expect(epochContaining(anchored, 50)).toEqual({ start: 50, end: 150 });
  expect(epochContaining(anchored, 150)).toEqual({ start: 150, end: 250 });
  expect(() => epochContaining(schedule, -1)).toThrow(TypeError);
  expect(() => epochContaining(schedule, 1.5)).toThrow(TypeError);
});

test("transitions take effect at the next old-schedule boundary", () => {
  const current = hourly();
  // Mid-epoch: the new anchor is the current epoch end, fields inherit.
  const next = transitionSchedule(current, { version: "2" }, HOUR / 2);
  expect({ ...next, pool: next.pool as string }).toEqual({
    pool: "default",
    limit: 1000,
    periodMs: HOUR,
    anchorMs: HOUR,
    version: "2",
  });
  // Exactly on a boundary, the change is effective immediately.
  const aligned = transitionSchedule(current, { limit: 500, version: "2" }, HOUR);
  expect(aligned.anchorMs).toBe(HOUR);
  expect(aligned.limit).toBe(500);
  // An explicit anchor must name a future boundary of the old schedule.
  const deferred = transitionSchedule(
    current,
    { periodMs: 2 * HOUR, anchorMs: 3 * HOUR, version: "2" },
    10,
  );
  expect(deferred.anchorMs).toBe(3 * HOUR);
  expect(deferred.periodMs).toBe(2 * HOUR);
});

test("overlapping, retroactive, and version-preserving transitions reject", () => {
  const current = hourly();
  // Same version is not a transition.
  expect(() => transitionSchedule(current, { version: "1" }, 10)).toThrow(TypeError);
  // Off-boundary, past, and pre-anchor anchors overlap the live schedule.
  for (const anchorMs of [10, HOUR / 2, HOUR + 1]) {
    expect(() => transitionSchedule(current, { anchorMs, version: "2" }, HOUR + 5)).toThrow(
      TypeError,
    );
  }
  // A boundary behind now is retroactive even though it is aligned.
  expect(() => transitionSchedule(current, { anchorMs: HOUR, version: "2" }, 2 * HOUR)).toThrow(
    TypeError,
  );
  // Bad fields reject alongside bad anchors.
  expect(() => transitionSchedule(current, { limit: -1, version: "2" }, 0)).toThrow(TypeError);
  expect(() => transitionSchedule(current, { periodMs: 0, version: "2" }, 0)).toThrow(TypeError);
  expect(() => transitionSchedule(current, { version: "" }, 0)).toThrow(TypeError);
  // The pool identity never changes across a transition.
  const next = transitionSchedule(current, { version: "2" }, 0);
  expect(next.pool as string).toBe(current.pool as string);
});
