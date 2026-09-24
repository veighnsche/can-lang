# T26 held-out agent comparison report

24 September 2026. Task T26 replay of the seven T01 registry cases
(`tests/baseline/registry.json`) for the new language mechanisms
against their best current idioms. Machine-readable form:
`t26-agent-comparison-2026-09-24.json` in this directory.

## What ran

No live model trials ran in this environment (no model access), so
attempts = 0 on every case and whole-task tokens are unmeasured
(`null` in the JSON). The frozen trial settings (primary
`gpt-6-sol` medium, efficient `gpt-6-luna` medium, escalation
`gpt-6-astra` high, five attempts, fresh workspace each) are
recorded unchanged for any future trial run. Registry prompts,
held-out variants and hidden checks are byte-unchanged by T26.

What T26 did replay mechanically, observed in this session:

- All 12 current-idiom fixtures exist on disk (closed-variant-arm,
  two-libraries, fixture-link, invoice-edit, stream-cleanup,
  invoice-grid, native-judgment).
- `go test ./compiler/internal/check/ -run
  'TestPatternBindCapturesAtEveryDepth|TestOwnerRecordBoundaryPositives|TestScenarioLinkResolves'`
  → ok.
- `bun test runtime/test/shutdown.test.ts` → 6 pass / 0 fail.
- `go run ./compiler/internal/catalogue/cmd/cataloguegen --check`
  → clean; `gofmt` clean; `go test ./tests/baseline/` → ok.

## Per-case outcome

| Case | Owner | Disposition |
| --- | --- | --- |
| closed-variant-arm | T02 | Shipped by gate evidence; agent comparison not run here |
| two-libraries | T03 | Shipped by gate evidence; agent comparison not run here |
| fixture-link | T08 | Shipped by gate evidence; agent comparison not run here |
| invoice-edit | T14 | Shipped by gate evidence; agent comparison not run here |
| stream-cleanup | T18 | Shipped by gate evidence; agent comparison not run here |
| invoice-grid | T24 | Shipped by gate evidence; agent comparison not run here |
| native-judgment | T26-deferred-or-later | Deferred; current idiom retained (DI-17 not proposed) |

Candidate slots owned by other tasks (`T03-two-libraries`,
`T08-fixture-link`, `T14-invoice-edit`, `T18-stream-cleanup`,
`T24-invoice-grid`) remain pending with their owners; T26 does not
fill them. The T26 slot `tests/baseline/candidates/T26-native-ai/`
records the native-AI deferral with the boundary restated and no
new mechanism proposed.

## Significance and acceptance

No significance is claimed from these replays: no trials ran, no
token effect was measured, and no language mechanism is credited by
this report. The six shipped mechanisms stand on their gate suites
(T09/T17/T20/T25 evidence), not on agent comparisons. The one T26
candidate fails the T01 mechanism acceptance rule by construction
(no mechanism proposed, nothing to distinguish in five attempts),
so it is deferred and the simpler current design is retained. The
held-out variants stay sealed for any future trial.
