// Bound-evidence substrate contract tests (H08 follow-up): exact
// measurement construction, per-measurement and sweep verification
// against candidate bounds, candidate U arithmetic, and split-tariff
// quoting. Breakdown smuggling, overflow wrap, and boundary
// off-by-ones are the pinned failure modes.
import { test, expect } from "bun:test";
import {
  boundMeasurement,
  candidateBound,
  quoteSplitCost,
  verifyCandidate,
  verifyMeasurementAgainstBound,
} from "../ai/bound.ts";

const sha = (seed: string): string =>
  Array.from(
    { length: 64 },
    (_, i) => "0123456789abcdef"[(seed.charCodeAt(i % seed.length) + i) % 16],
  ).join("");

const measurement = (inputTokens: number, outputTokens: number, seed = "m") =>
  boundMeasurement({ requestSha256: sha(seed), requestBytes: 100, inputTokens, outputTokens });

test("measurement accepts the exact four-key shape", () => {
  const made = boundMeasurement({
    requestSha256: sha("ok"),
    requestBytes: 143,
    inputTokens: 275,
    outputTokens: 21,
  });
  expect(made.inputTokens).toBe(275);
  expect(made.outputTokens).toBe(21);
  expect(Object.keys(made).sort()).toEqual([
    "inputTokens",
    "outputTokens",
    "requestBytes",
    "requestSha256",
  ]);
  expect(Object.isFrozen(made)).toBe(true);
});

test("measurement refuses breakdown smuggling and malformed fields", () => {
  const base = { requestSha256: sha("x"), requestBytes: 10, inputTokens: 1, outputTokens: 2 };
  expect(() => boundMeasurement({ ...base, cached_tokens: 9999 })).toThrow(TypeError);
  expect(() => boundMeasurement({ ...base, reasoning_tokens: 1 })).toThrow(TypeError);
  expect(() => boundMeasurement({ ...base, prompt_text: "leak" })).toThrow(TypeError);
  expect(() => boundMeasurement({ ...base, outputTokens: undefined })).toThrow(TypeError);
  expect(() => boundMeasurement({ ...base, inputTokens: -1 })).toThrow(TypeError);
  expect(() => boundMeasurement({ ...base, outputTokens: 1.5 })).toThrow(TypeError);
  expect(() => boundMeasurement({ ...base, requestBytes: 0 })).toThrow(TypeError);
  expect(() => boundMeasurement({ ...base, requestSha256: "not-a-hash" })).toThrow(TypeError);
  expect(() => boundMeasurement({ ...base, requestSha256: "f".repeat(63) })).toThrow(TypeError);
  expect(() => boundMeasurement(null)).toThrow(TypeError);
  expect(() => boundMeasurement([])).toThrow(TypeError);
});

test("measurement refuses counter pairs whose sum overflows", () => {
  expect(() =>
    boundMeasurement({
      requestSha256: sha("big"),
      requestBytes: 10,
      inputTokens: Number.MAX_SAFE_INTEGER,
      outputTokens: 1,
    }),
  ).toThrow(TypeError);
});

test("single-measurement verify is exact at the boundary", () => {
  expect(verifyMeasurementAgainstBound(measurement(100, 23), 200)).toEqual({
    verdict: "within",
    actual: 123,
    released: 77,
  });
  expect(verifyMeasurementAgainstBound(measurement(100, 23), 123)).toEqual({
    verdict: "within",
    actual: 123,
    released: 0,
  });
  expect(verifyMeasurementAgainstBound(measurement(100, 23), 122)).toEqual({
    verdict: "breach",
    actual: 123,
    overBy: 1,
  });
  expect(() => verifyMeasurementAgainstBound(measurement(1, 1), 0)).toThrow(TypeError);
  expect(() => verifyMeasurementAgainstBound(measurement(1, 1), 2.5)).toThrow(TypeError);
});

test("candidate pins U as the exact input-plus-output sum", () => {
  const candidate = candidateBound({
    provider: "typesafe",
    model: "jev",
    version: "1.13.0",
    inputAllowance: 32768,
    outputAllowance: 512,
    derivation: "h08-followup:publication+bracket;unqualified",
  });
  expect(candidate.upperBound).toBe(33280);
  expect(candidate.inputAllowance + candidate.outputAllowance).toBe(candidate.upperBound);
  expect(Object.isFrozen(candidate)).toBe(true);
  expect(() =>
    candidateBound({
      provider: "typesafe",
      model: "jev",
      version: "1.13.0",
      inputAllowance: Number.MAX_SAFE_INTEGER,
      outputAllowance: 1,
      derivation: "overflow",
    }),
  ).toThrow(TypeError);
  expect(() =>
    candidateBound({
      provider: "",
      model: "jev",
      version: "1.13.0",
      inputAllowance: 1,
      outputAllowance: 1,
      derivation: "d",
    }),
  ).toThrow(TypeError);
  expect(() =>
    candidateBound({
      provider: "typesafe",
      model: "jev",
      version: "1.13.0",
      inputAllowance: 1,
      outputAllowance: 1,
      derivation: "  ",
    }),
  ).toThrow(TypeError);
});

test("sweep verify sums exactly with no double-count", () => {
  const candidate = candidateBound({
    provider: "typesafe",
    model: "jev",
    version: "1.13.0",
    inputAllowance: 32768,
    outputAllowance: 512,
    derivation: "h08-followup:test-sweep",
  });
  const inputs = [100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 100, 56];
  const sweep = inputs.map((inputTokens, i) => measurement(inputTokens, 23, `sweep-${i}`));
  const verdict = verifyCandidate(candidate, sweep);
  expect(verdict.within).toBe(true);
  expect(verdict.count).toBe(12);
  expect(verdict.totalInput).toBe(1156);
  expect(verdict.totalOutput).toBe(276);
  expect(verdict.total).toBe(1432);
  expect(verdict.maxActual).toBe(123);
  expect(verdict.peakOutput).toBe(23);
  expect(verdict.firstBreach).toBeUndefined();
});

test("sweep verify reports the first breach exactly", () => {
  const candidate = candidateBound({
    provider: "typesafe",
    model: "jev",
    version: "1.13.0",
    inputAllowance: 100,
    outputAllowance: 23,
    derivation: "h08-followup:test-breach",
  });
  const sweep = [measurement(100, 23, "a"), measurement(100, 24, "b"), measurement(500, 0, "c")];
  const verdict = verifyCandidate(candidate, sweep);
  expect(verdict.within).toBe(false);
  expect(verdict.firstBreach).toEqual({ index: 1, actual: 124, overBy: 1 });
  expect(verdict.maxActual).toBe(500);
  expect(verdict.total).toBe(747);
  expect(() => verifyCandidate(candidate, [])).toThrow(TypeError);
});

test("split quote bills each side at its own rate", () => {
  const free = quoteSplitCost(
    { inputTokens: 23546, outputTokens: 54 },
    { usdPerInputToken: 4.2e-8, usdPerOutputToken: 0 },
  );
  expect(free.outputUsd).toBe(0);
  expect(free.inputUsd).toBeCloseTo(23546 * 4.2e-8, 15);
  expect(free.totalUsd).toBe(free.inputUsd + free.outputUsd);
  // A single-price application at the input rate would charge
  // (23546+54)*4.2e-8; the split quote must differ from that.
  expect(free.totalUsd).not.toBeCloseTo((23546 + 54) * 4.2e-8, 12);
  const both = quoteSplitCost(
    { inputTokens: 100, outputTokens: 50 },
    { usdPerInputToken: 2e-6, usdPerOutputToken: 1e-6 },
  );
  expect(both.totalUsd).toBeCloseTo(2e-4 + 0.5e-4, 15);
  expect(
    quoteSplitCost(
      { inputTokens: 0, outputTokens: 0 },
      { usdPerInputToken: 1, usdPerOutputToken: 1 },
    ),
  ).toEqual({
    inputUsd: 0,
    outputUsd: 0,
    totalUsd: 0,
  });
  expect(() =>
    quoteSplitCost(
      { inputTokens: -1, outputTokens: 0 },
      { usdPerInputToken: 1, usdPerOutputToken: 0 },
    ),
  ).toThrow(TypeError);
  expect(() =>
    quoteSplitCost(
      { inputTokens: 1, outputTokens: 0 },
      { usdPerInputToken: NaN, usdPerOutputToken: 0 },
    ),
  ).toThrow(TypeError);
});
