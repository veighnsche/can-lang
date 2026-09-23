import { describe, expect, test } from "bun:test";
import { createHash } from "node:crypto";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, domainFailureDiagnostics, type FailureShape } from "../domain.ts";
import { value, type Completion } from "../completion.ts";
import { record } from "../data.ts";
import { createDateTimes, isTimeInstantValue } from "../platform/datetime.ts";

const identity = (kind: string, declaration: string) =>
  createHash("sha256").update("can-concrete-type-v1\0" + JSON.stringify([kind, declaration])).digest("hex");
const textShape: FailureShape = { identity: identity("primitive", "str"), kind: "primitive", declaration: "str", arguments: [], fields: [], leaves: [], inputs: [], errors: [] };
const intShape: FailureShape = { identity: identity("primitive", "int"), kind: "primitive", declaration: "int", arguments: [], fields: [], leaves: [], inputs: [], errors: [] };
const fieldTypes: Record<string, Record<string, string>> = {
  "time::out_of_range": { millis: intShape.identity },
  "time::invalid_zone": { zone: textShape.identity },
  "time::nonexistent_time": {},
  "time::invalid_option": { reason: textShape.identity },
};
const declarations = catalogue.errors
  .filter(e => fieldTypes[e.name] !== undefined)
  .map(e => ({ identity: e.identity, name: e.name, id: e.id, parameters: 0 }));
const errorShapes: FailureShape[] = declarations.map(e => ({
  identity: identity("error", e.identity), kind: "error", declaration: e.identity, arguments: [],
  fields: Object.entries(fieldTypes[e.name]!).map(([name, type]) => ({ name, type })),
  leaves: [], inputs: [], errors: [],
}));
const domain = createDomainRuntime({ declarations, shapes: [textShape, intShape, ...errorShapes] });
const id = (declaration: string) => identity("error", declaration);
const api = createDateTimes(domain, {
  outOfRange: id("can.std.time@1::out_of_range"), invalidZone: id("can.std.time@1::invalid_zone"),
  nonexistent: id("can.std.time@1::nonexistent_time"), invalidOption: id("can.std.time@1::invalid_option"),
});

function outcome(completion: Completion<unknown>, name: string): Record<string, unknown> {
  expect(completion.kind).toBe("domain");
  if (completion.kind !== "domain") throw new Error("wrong outcome");
  const details = domainFailureDiagnostics(completion.value);
  expect(details.declaration.name).toBe(name);
  const payload = details.payload as Record<string, unknown>;
  const plain: Record<string, unknown> = {};
  for (const key of Object.keys(payload)) plain[key] = payload[key];
  return plain;
}
const civil = (year: number, month: number, day: number, hour: number, minute: number, second: number, millisecond = 0) =>
  record("time::civil", [["year", BigInt(year)], ["month", BigInt(month)], ["day", BigInt(day)], ["hour", BigInt(hour)], ["minute", BigInt(minute)], ["second", BigInt(second)], ["millisecond", BigInt(millisecond)]]);

describe("instants", () => {
  test("epoch millis round-trip exactly inside the native span", async () => {
    for (const millis of [0n, -1n, 1758625140000n, 8640000000000000n, -8640000000000000n]) {
      const handle = value(await api.instantFromEpochMillis(millis));
      expect(isTimeInstantValue("instant", handle)).toBe(true);
      expect(value(await api.instantEpochMillis(handle))).toBe(millis);
    }
    expect(outcome(await api.instantFromEpochMillis(8640000000000001n), "time::out_of_range")).toEqual({ millis: 8640000000000001n });
    expect(outcome(await api.instantFromEpochMillis(-8640000000000001n), "time::out_of_range")).toEqual({ millis: -8640000000000001n });
    expect(JSON.stringify(value(await api.instantFromEpochMillis(0n)))).toBe("{}");
  });
});

describe("zoned formatting", () => {
  test("explicit locale and zone format deterministically on the pinned ICU", async () => {
    const noon = value(await api.instantFromEpochMillis(BigInt(Date.UTC(2026, 8, 23, 12, 0, 0))));
    expect(value(await api.formatInZone(noon, "en-US", "Asia/Tokyo", "medium", "short"))).toBe("Sep 23, 2026 at 9:00 PM");
    expect(value(await api.formatInZone(noon, "en-US", "UTC", "full", "none"))).toBe("Wednesday, September 23, 2026");
    expect(value(await api.formatInZone(noon, "en-US", "UTC", "none", "medium"))).toBe("12:00:00 PM");
    expect(value(await api.formatInZone(noon, "de-DE", "Europe/Berlin", "short", "short"))).toBe("23.09.26, 14:00");
  });
  test("unknown zones, styles, and locales reject distinctly", async () => {
    const instant = value(await api.instantFromEpochMillis(0n));
    expect(outcome(await api.formatInZone(instant, "en-US", "Mars/Olympus", "medium", "short"), "time::invalid_zone")).toEqual({ zone: "Mars/Olympus" });
    expect(outcome(await api.formatInZone(instant, "en-US", "UTC", "sometimes", "short"), "time::invalid_option")).toEqual({ reason: "date_style" });
    expect(outcome(await api.formatInZone(instant, "en-US", "UTC", "medium", "whenever"), "time::invalid_option")).toEqual({ reason: "time_style" });
    expect(outcome(await api.formatInZone(instant, "!", "UTC", "medium", "short"), "time::invalid_option")).toEqual({ reason: "locale" });
  });
});

describe("civil resolution", () => {
  const zone = "America/New_York";
  test("ordinary civil times resolve exactly", async () => {
    const handle = value(await api.resolveZonedTime(civil(2026, 1, 15, 12, 30, 45), zone, 0n));
    expect(value(await api.instantEpochMillis(handle))).toBe(BigInt(Date.UTC(2026, 0, 15, 17, 30, 45)));
  });
  test("spring gaps reject; fall overlaps resolve by explicit policy", async () => {
    expect(outcome(await api.resolveZonedTime(civil(2026, 3, 8, 2, 30, 0), zone, 0n), "time::nonexistent_time")).toEqual({});
    const earlier = value(await api.resolveZonedTime(civil(2026, 11, 1, 1, 30, 0), zone, 0n));
    const later = value(await api.resolveZonedTime(civil(2026, 11, 1, 1, 30, 0), zone, 1n));
    expect(value(await api.instantEpochMillis(earlier))).toBe(BigInt(Date.UTC(2026, 10, 1, 5, 30, 0)));
    expect(value(await api.instantEpochMillis(later))).toBe(BigInt(Date.UTC(2026, 10, 1, 6, 30, 0)));
  });
  test("impossible dates reject as nonexistent; malformed parts and policies reject as options", async () => {
    expect(outcome(await api.resolveZonedTime(civil(2026, 2, 30, 0, 0, 0), zone, 0n), "time::nonexistent_time")).toEqual({});
    expect(outcome(await api.resolveZonedTime(civil(2026, 13, 1, 0, 0, 0), zone, 0n), "time::invalid_option")).toEqual({ reason: "parts" });
    expect(outcome(await api.resolveZonedTime(civil(2026, 1, 1, 25, 0, 0), zone, 0n), "time::invalid_option")).toEqual({ reason: "parts" });
    expect(outcome(await api.resolveZonedTime(civil(2026, 1, 1, 0, 0, 0), zone, 2n), "time::invalid_option")).toEqual({ reason: "policy" });
    expect(outcome(await api.resolveZonedTime(civil(2026, 1, 1, 0, 0, 0), "Mars/Olympus", 0n), "time::invalid_zone")).toEqual({ zone: "Mars/Olympus" });
  });
  test("low civil years stay literal instead of mapping to 19xx", async () => {
    const handle = value(await api.resolveZonedTime(civil(99, 6, 15, 12, 0, 0), "UTC", 0n));
    const expected = new Date(Date.UTC(99, 5, 15, 12, 0, 0));
    expected.setUTCFullYear(99);
    expect(value(await api.instantEpochMillis(handle))).toBe(BigInt(expected.getTime()));
  });
});
