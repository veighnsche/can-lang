# T01 implementation baseline and agent-comparison registry

Dated 2026-09-24. This directory freezes the starting point for the
T01–T27 implementation graph and registers the agent-comparison cases
before any trial runs. It implements task T01 only.

## Files

- `baseline.json` — frozen baseline: date, compiler revision, supported
  Bun/Go/TypeScript toolchain, disposable-directory policy, the core
  regression commands (mirroring `.github/workflows/verifier.yml`) and
  the acceptance rules from the evaluation protocol.
- `registry.json` — one registered case per evaluation-protocol workload
  family. Each case carries creation, controlled-refactor and
  diagnostic-repair prompts, current-idiom fixtures that exist on disk,
  a pending candidate slot owned by a later task, a held-out variant
  with changed names/data, hidden checks and frozen model/effort
  settings (primary `gpt-6-sol` medium, efficient `gpt-6-luna` medium,
  escalation `gpt-6-astra` high, five attempts, fresh workspace each).
- `run.sh` — repeatable harness. `--smoke` (default) verifies the
  baseline fast; `--full` runs the whole core regression. Every run
  uses a fresh disposable directory and writes a JSON report there.
- `baseline_test.go` — committed tests: toolchain matches the freeze,
  the registry is complete, and the smoke harness is repeatable.

## Use

```sh
# Fast baseline verification (gofmt, disposable canlc build, version smoke).
tests/baseline/run.sh --smoke

# Full core regression (vet, cataloguegen, modcheck, gramcheck,
# runtime checks, go test ./..., bun test runtime/test/).
tests/baseline/run.sh --full

# Committed T01 tests.
go test ./tests/baseline/
```

Set `KEEP_DIR=1` to keep the disposable run directory for inspection.

## Rules for later tasks

- Never edit a registered prompt, held-out variant or hidden check in
  place. If a case must change, open a new dated registry revision.
- Add candidate (new-mechanism) fixtures under
  `tests/baseline/candidates/<owner-task>-<case>/`.
- Record every compiler/tooling baseline change explicitly; a pilot
  run may debug the harness but is never counted among the five
  attempts and credits no language mechanism.
