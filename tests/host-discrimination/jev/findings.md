# D01 Jev findings — per-class tier advice 2026-09-26

Model `jev-1.13.0` (requested `jev-latest`); 3/3 HTTP 200.
Usage: 4,633 input + 423 output tokens total. All requests prepared
before the first send; 18/18 explanatory fields differ pairwise (max
similarity 0.773; numbers and identifiers repeat by design) with
semantic equivalence manually reviewed — see
`jev-wording-audit.json`. Numbers are selected-option probabilities,
not design-correctness probabilities. Advice only: measurements
decide, and the chart disagreement below is preserved as the
counter-case rather than outvoted.

| Class | C1 | C2 | C3 |
|---|---|---|---|
| storage_tier | reviewed_adapter (0.84) | reviewed_adapter (0.71) | reviewed_adapter (0.93) |
| clipboard_tier | reviewed_adapter (0.97) | reviewed_adapter (0.92) | reviewed_adapter (0.97) |
| chart_tier | reviewed_adapter (0.56) | reviewed_adapter (0.58) | reviewed_adapter (0.88) |

Runner-ups: storage C2 catalogue 0.26; chart C1 companion 0.43
(confidence 0.33), C2 catalogue 0.36 (confidence 0.37); clipboard
companion 0.00 in all three.

## Agreement (advice, not proof)

- **clipboard → reviewed_adapter (0.97/0.92/0.97, confidence
  0.88–0.95).** Unanimous and strong; companion scores exactly zero
  in every wording, so the measured compare-target exclusion carried
  through the classifier. Supports the T2 assignment with T3
  excluded by measurement.
- **storage → reviewed_adapter (0.84/0.71/0.93).** Unanimous with a
  live counter-case: C2 holds catalogue at 0.26, preserving the
  kept-live T1 reading (generic surface, second-app condition).
  Supports the T2 assignment with T1 conditional.

## Disagreement (investigated)

- **chart: adapter wins 3/3 but the vote is unstable.** C1 nearly
  ties companion (0.56 vs 0.43, confidence 0.33); C2's runner-up is
  catalogue instead (0.36, confidence 0.37); only C3 is decisive
  (0.88). The runner-up swings across wordings while the winner's
  confidence stays low in two of three — the classifier finds this
  class genuinely contested, and no wording reaches clipboard-level
  confidence. Engineering lean before consulting favored companion
  for this workflow (zero callbacks, SDK bytes out of the audited
  browser asset, online-by-construction workflow absorbs the
  boundary's online cost). The consultations disagree with that
  lean, weakly.
- **State gap (acknowledged, not repaired by re-consulting).** The
  requests stated the audit-surface point but not the two deciding
  arguments: vendor-SDK churn forcing per-version re-review under
  T2, and the invoice workflow's online-by-construction property
  zeroing T3's marginal online cost. Re-running with repaired state
  after seeing the outcome would manufacture agreement, so the
  disagreeing read stands as the preserved counter-case: if D02/H
  weigh local selection latency and offline cached-draft rendering
  above churn absorption, Jev's T2 lean plus the measured T2 legs
  are the complete alternative case. Trip conditions for
  T2-after-all are in the record.
- **Why the classifier plausibly leans adapter here.** The state
  stressed measured shim burden (279 lines / 6051 bytes vs 360
  lines / 20236 bytes) and local selection; the unmeasured dominant
  terms (real SDK bytes, re-review cadence, roundtrip latency
  criticality) were absent by design. A classifier scoring stated
  burden should prefer the adapter — which is exactly why the
  decision rests on the structural analysis in the record, not on
  these votes.

## Engineering use

Adopt the Op A / Op B reads as corroborating advice for the
measurement-led T2 assignments. Treat the chart read as a
sensitivity flag confirming the class is the true contest: the
record assigns T3 scoped to the invoice-analytics workflow with
explicit trip conditions, and carries Jev's T2 lean as the
counter-case for D02/H to weigh. No majority-vote rule is applied.
