# LF20/AE49 after-implementation measurements — 2026-09-22

## Frozen comparison rerun

`measure_after.py` re-verifies the frozen before hash/counts and measures the
implemented after source with the same region rule (stop before
`fn str referenced`), counting normalized `http::request_failed` bounds/arms.
Results (`after-metrics.json`):

| Metric, fixed comparison region | Before | After | Required |
| --- | ---: | ---: | ---: |
| Fetch declarations / calls in `main` | 8 / 8 | 8 / 8 | 8 / 8 |
| Infrastructure-bound declarations | 9 | 9 | 9 |
| Infrastructure entries in those bounds | 63 | 9 | 9 |
| Infrastructure forwarding arms | 56 | 8 | 8 |
| Standard transport/codec distinctions recoverable from detail | 7 | 7 | 7 |

The after project is `compiler/testdata/current/fetch/main.can` (181 lines).
Behavioral gates pass before any comparison: `TestCurrentBundledFetch`
(staged, 20 assert rows + 8 live runtime requests) and an offline
`canlc assert` (20/20, evidence split 10 `when`-substituted, 8 raw-fixture,
2 pure). The 8 `checks::failed` arms in the region are separately required
B7 native-assertion costs, not infrastructure forwarding.

Full-project source cost with the documented line classifier:

| Version | Production lines | Assert/fixture lines | Raw fixtures |
| --- | ---: | ---: | --- |
| Before (`eight-fetch-before.can.txt`) | 165 | 26 | none |
| After (`fetch/main.can`) | 110 | 70 | 9 files, 3670 bytes |

## Application baselines rerun

Same method as `../full-language-review/measure.py`, plus the
production/assert-fixture split. `TestApplicationsStaged` passes.
account-search, dashboard, and form-validation reproduce their frozen counts
exactly (373/291/229 lines, same emits declarations and characters):
preserved baselines. native-ai evolved during implementation (352→345 lines,
67→39 raw-infrastructure mentions, 5 normalized `request_failed` mentions);
both values are reported, no ratio is claimed.

No TypeScript comparison is included: no equivalent typed-failure-boundary
baseline exists, and a non-equivalent sketch would be excluded by AE49
rather than scored.

## Edit scenarios (each ends green)

- E1 add a domain failure: new `empty_echo` error (registry id 1000000),
  `main` emits + arm, one assert row, three `when` rows at crossed
  boundaries; 81 diff lines (36 are mechanical re-indent), registry entry,
  21/21 rows pass. Failed runs before green: 3 (when-row indentation
  syntax error; stale copied `dist/` owner harness error; same-package
  qualifier check error). Traces: `E1-main.diff`, `E1-registry.diff`,
  `E1-report.json`.
- E2 change a request field: third `labels` value in `load_json` plus the
  recorded raw-request URL (`+11` bytes in `load_json.json`); 1 source line,
  20/20 rows pass, green on the first run. Traces: `E2-main.diff`,
  `E2-fixture.json`, `E2-report.json`.
- E3 change a callback capture value: `near int increment` row `3 → 4`
  (`10 → 11`) in the callables captures project; 1 line, 11/11 rows pass,
  green on the first run. Traces: `E3-main.diff`, `E3-report.json`.
- E4 reuse one scenario at two sites: second `send_text` call site with its
  own `when` row (`+7/−1` lines); the first site's row does not leak
  (independent per-site substitution); 20/20 rows pass, green on the first
  run. Traces: `E4-main.diff`, `E4-report.json`.

Affected declarations/callers/tests: E1 touches `main` (bound, one arm, one
assert row), three call-site `when` blocks, and the error registry; E2
touches `load_json` and its raw fixture; E3 touches one assert row of
`read`; E4 touches `main` only. No other declaration, caller, or test
needed changes in any scenario.

## Mutation negatives (`mutations.json`)

M1 omitted arm → check-time `missing completion arm`; M2 wrong path →
`harness violation` on the raw-request comparison; M3 corrupted fixture →
`outcome mismatch`; M4 mistyped capture → check-time row-region error. All
four detected on the first run and repaired to green by reverting the
mutation. No mutation produced a cheaper passing program.

## Authoring-workflow record

Agent: Muse Spark via Muse Code, default session settings. Scenario work
used 19 `canlc assert` invocations (1 baseline, 4 E1, 2 E2, 3 E3 incl.
baseline, 1 E4, 8 mutation detect/repair) plus 2 staged Go gate runs
(fetch, applications). Diagnostic cycles: 3 (all E1, listed above); E2–E4
and all four mutation detections completed on the first run. Token usage is
not measured in-session and is not reported. One run only; no statistical
generality is claimed, and no deferred abstraction is promoted from these
counts.
