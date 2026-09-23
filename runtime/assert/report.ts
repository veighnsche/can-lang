// Evidence records the boundary exercised, not the strength of an assertion's
// result. In particular, supplying a completion never implies provider or Bun
// conformance coverage.
export const evidenceLabels = Object.freeze([
  "real-can",
  "supplied-completion",
  "raw-provider-fixture",
  "policy-fixture",
  "bun-conformance",
  "live-quality",
] as const);
export type EvidenceLabel = (typeof evidenceLabels)[number];
export type EvidenceScope = "assertion" | "bun-conformance" | "live-quality";
declare const evidenceBrand: unique symbol;
export type Evidence = Readonly<{ [evidenceBrand]: true }>;
type State = { scope: EvidenceScope; labels: Set<EvidenceLabel> };
const records = new WeakMap<object, State>();
function state(value: Evidence): State {
  const found = value !== null && typeof value === "object" ? records.get(value) : undefined;
  if (!found) throw new TypeError("invalid evidence record");
  return found;
}
export function createEvidence(scope: EvidenceScope): Evidence {
  if (!["assertion", "bun-conformance", "live-quality"].includes(scope))
    throw new TypeError("invalid evidence scope");
  const value = Object.freeze(Object.create(null)) as Evidence;
  records.set(value, { scope, labels: new Set() });
  return value;
}
export function recordEvidence(value: Evidence, label: EvidenceLabel): void {
  const current = state(value);
  if (!evidenceLabels.includes(label)) throw new TypeError("invalid evidence label");
  if ((label === "bun-conformance" || label === "live-quality") && current.scope !== label) {
    throw new TypeError("evidence requires an explicit matching job");
  }
  current.labels.add(label);
}
export function evidenceReport(value: Evidence): readonly EvidenceLabel[] {
  return Object.freeze([...state(value).labels].sort());
}
// Release tooling may combine jobs, but must retain all six distinct buckets.
// Empty categories remain visible and cannot inherit another category's result.
export function evidenceSummary(values: readonly Evidence[]) {
  const counts: Record<EvidenceLabel, number> = {
    "real-can": 0,
    "supplied-completion": 0,
    "raw-provider-fixture": 0,
    "policy-fixture": 0,
    "bun-conformance": 0,
    "live-quality": 0,
  };
  for (const value of values) for (const label of evidenceReport(value)) counts[label]++;
  return Object.freeze(counts);
}
