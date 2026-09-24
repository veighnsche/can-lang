# T26 native-judgment candidate slot: deferred, current idiom retained

Case `native-judgment` (family "Native AI and later items"), owner
`T26-deferred-or-later` per `tests/baseline/registry.json`. This
directory is the registered candidate slot
`tests/baseline/candidates/T26-native-ai/`.

## Disposition: deferred

DI-17 (new native AI wrapper/batch/callable forms) stays deferred
pending the ledger's specific reopening evidence. No new language
mechanism is proposed here, so there is no new-mechanism fixture to
trial. The current idiom is retained:

- Current fixtures: `examples/native-ai/` (declared catalogue
  judgment calls with typed failures) and
  `tests/integration/judge_test.go` (staged native-judge execution).
- Held-out variant `native-judgment-heldout` (`case -> matter`,
  `batch -> bundle`, different prompts/inputs) is never shown in
  prompts; it applies to any future candidate, not to this
  deferral.

## Boundary restated (no change)

A judgment call observes its declared capability boundary: calls
inside the admitted catalogue pass; calls outside it are diagnosed
at check time (see the hidden checks `boundary-held` and
`admitted-passes` in the registry). Wrapper/batch/callable forms
stay within the admitted catalogue; there is no privileged ambient
access. This matches the T01 mechanism acceptance rule: five
attempts could not distinguish a new mechanism that was never
proposed, so the simpler current design is retained and the
hypothesis stays unresolved.

## Mechanical replay (T26)

- Both current-idiom fixtures exist on disk (checked by
  `tests/baseline/t26_product_guide_test.go`).
- `go test ./tests/integration/ -run 'TestJudge|TestNative'`
  without `CAN_BUN_ARCHIVE` skips staged legs by design; no live
  model trial ran in this environment.

See `candidate.json` for the machine-readable slot status and
`tests/baseline/reports/t26-agent-comparison-2026-09-24.md` for the
full comparison report. No significance is claimed.
