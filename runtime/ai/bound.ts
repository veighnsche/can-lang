// Complete-call bound evidence substrate (H08 follow-up): exact
// per-measurement and sweep verification for bound-qualification
// campaigns, candidate-bound construction with U pinned as the exact
// input-plus-output sum, and split-tariff cost quoting. Everything
// here is hashes and counters only: measurements carry no request
// text, bodies, or credentials, and breakdown fields (cached,
// reasoning, or anything beyond top-level input/output) are refused
// by shape so they can never double-count into a total.
import { meteringProfile, type MeteringProfile } from "../outbound/ledger.ts";

function checkedAdd(a: number, b: number): number {
  const total = a + b;
  if (!Number.isSafeInteger(total)) throw new TypeError("token total exceeds safe integer");
  return total;
}

function checkTokens(kind: string, value: unknown): number {
  if (!Number.isSafeInteger(value) || (value as number) < 0)
    throw new TypeError(`invalid bound-evidence ${kind}`);
  return value as number;
}

function checkBound(kind: string, value: unknown): number {
  if (!Number.isSafeInteger(value) || (value as number) < 1)
    throw new TypeError(`invalid bound-evidence ${kind}`);
  return value as number;
}

// BoundMeasurement is one settled live call reduced to evidence:
// the request-body hash and length plus the authoritative top-level
// counters. The exact four keys are enforced: anything else (breakdown
// fields, text, bodies) fails construction instead of leaking into sums.
export type BoundMeasurement = Readonly<{
  requestSha256: string;
  requestBytes: number;
  inputTokens: number;
  outputTokens: number;
}>;

const MEASUREMENT_KEYS = ["inputTokens", "outputTokens", "requestBytes", "requestSha256"] as const;

export function boundMeasurement(input: unknown): BoundMeasurement {
  if (input === null || typeof input !== "object" || Array.isArray(input))
    throw new TypeError("invalid bound-evidence measurement");
  const shaped = input as Record<string, unknown>;
  const keys = Object.keys(shaped);
  if (keys.length !== MEASUREMENT_KEYS.length || !MEASUREMENT_KEYS.every((k) => keys.includes(k)))
    throw new TypeError("invalid bound-evidence measurement shape");
  const sha = shaped.requestSha256;
  if (typeof sha !== "string" || !/^[0-9a-f]{64}$/.test(sha))
    throw new TypeError("invalid bound-evidence request hash");
  const requestBytes = checkBound("request bytes", shaped.requestBytes);
  const inputTokens = checkTokens("input tokens", shaped.inputTokens);
  const outputTokens = checkTokens("output tokens", shaped.outputTokens);
  checkedAdd(inputTokens, outputTokens);
  return Object.freeze({ requestSha256: sha, requestBytes, inputTokens, outputTokens });
}

export type MeasurementVerdict =
  | Readonly<{ verdict: "within"; actual: number; released: number }>
  | Readonly<{ verdict: "breach"; actual: number; overBy: number }>;

// verifyMeasurementAgainstBound checks one authoritative actual
// against a candidate U with checked arithmetic: actual == U is
// within with zero release, actual == U+1 is a breach by exactly 1.
// Overflow never wraps: totals past MAX_SAFE_INTEGER throw.
export function verifyMeasurementAgainstBound(
  measurement: BoundMeasurement,
  upperBound: unknown,
): MeasurementVerdict {
  const bound = checkBound("upper bound", upperBound);
  const actual = checkedAdd(measurement.inputTokens, measurement.outputTokens);
  if (actual <= bound)
    return Object.freeze({ verdict: "within", actual, released: bound - actual });
  return Object.freeze({ verdict: "breach", actual, overBy: actual - bound });
}

// CandidateBound pins one candidate complete-call bound: the
// provider/model/version identity plus input and output allowances
// whose exact checked sum is U. The derivation names the proof
// reference (publication, bracket, campaign); constructing a
// candidate claims nothing about qualification by itself.
export type CandidateBound = Readonly<{
  provider: string;
  model: string;
  version: string;
  inputAllowance: number;
  outputAllowance: number;
  upperBound: number;
  derivation: string;
}>;

export function candidateBound(input: {
  provider: unknown;
  model: unknown;
  version: unknown;
  inputAllowance: unknown;
  outputAllowance: unknown;
  derivation: unknown;
}): CandidateBound {
  const pinned: MeteringProfile = meteringProfile(input);
  const inputAllowance = checkBound("input allowance", input.inputAllowance);
  const outputAllowance = checkBound("output allowance", input.outputAllowance);
  if (typeof input.derivation !== "string" || input.derivation.trim() === "")
    throw new TypeError("invalid bound-evidence derivation");
  return Object.freeze({
    ...pinned,
    inputAllowance,
    outputAllowance,
    upperBound: checkedAdd(inputAllowance, outputAllowance),
    derivation: input.derivation,
  });
}

export type CandidateVerdict = Readonly<{
  within: boolean;
  count: number;
  totalInput: number;
  totalOutput: number;
  total: number;
  maxActual: number;
  peakOutput: number;
  firstBreach?: Readonly<{ index: number; actual: number; overBy: number }>;
}>;

// verifyCandidate checks a whole measurement sweep against a
// candidate: exact input/output/total sums with no double-count,
// the maximum actual and peak output, and the first breach when
// any measurement exceeds U. An empty sweep verifies nothing and
// is refused. Totals use checked addition throughout.
export function verifyCandidate(
  candidate: CandidateBound,
  measurements: readonly BoundMeasurement[],
): CandidateVerdict {
  if (measurements.length === 0) throw new TypeError("invalid bound-evidence empty sweep");
  let totalInput = 0;
  let totalOutput = 0;
  let maxActual = 0;
  let peakOutput = 0;
  let firstBreach: CandidateVerdict["firstBreach"];
  measurements.forEach((measurement, index) => {
    const checked = verifyMeasurementAgainstBound(measurement, candidate.upperBound);
    totalInput = checkedAdd(totalInput, measurement.inputTokens);
    totalOutput = checkedAdd(totalOutput, measurement.outputTokens);
    if (checked.actual > maxActual) maxActual = checked.actual;
    if (measurement.outputTokens > peakOutput) peakOutput = measurement.outputTokens;
    if (checked.verdict === "breach" && firstBreach === undefined)
      firstBreach = Object.freeze({ index, actual: checked.actual, overBy: checked.overBy });
  });
  return Object.freeze({
    within: firstBreach === undefined,
    count: measurements.length,
    totalInput,
    totalOutput,
    total: checkedAdd(totalInput, totalOutput),
    maxActual,
    peakOutput,
    ...(firstBreach === undefined ? {} : { firstBreach }),
  });
}

export type SplitTariff = Readonly<{ usdPerInputToken: number; usdPerOutputToken: number }>;
export type SplitQuote = Readonly<{ inputUsd: number; outputUsd: number; totalUsd: number }>;

// quoteSplitCost bills authoritative usage through a two-rate
// tariff (input and output priced separately, either leg possibly
// free). Unlike the single-price spend tracker — which must charge
// every settled token at one rate — the split quote bills each side
// at its own rate, so an output-free tariff never charges output.
export function quoteSplitCost(
  usage: Readonly<{ inputTokens: unknown; outputTokens: unknown }>,
  tariff: Readonly<{ usdPerInputToken: unknown; usdPerOutputToken: unknown }>,
): SplitQuote {
  const inputTokens = checkTokens("input tokens", usage.inputTokens);
  const outputTokens = checkTokens("output tokens", usage.outputTokens);
  const priceIn = tariff.usdPerInputToken;
  const priceOut = tariff.usdPerOutputToken;
  if (typeof priceIn !== "number" || !Number.isFinite(priceIn) || priceIn < 0)
    throw new TypeError("invalid bound-evidence input tariff");
  if (typeof priceOut !== "number" || !Number.isFinite(priceOut) || priceOut < 0)
    throw new TypeError("invalid bound-evidence output tariff");
  const inputUsd = inputTokens * priceIn;
  const outputUsd = outputTokens * priceOut;
  return Object.freeze({ inputUsd, outputUsd, totalUsd: inputUsd + outputUsd });
}
