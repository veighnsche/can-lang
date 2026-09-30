# Grouped errors and test labels: completion record (2026-09-30)

## Scope

Grouped completion errors (`error_a | error_b => handler`, bare `error_a | error_b`)
and grouped assertion/fixture labels (`absent | configured: arguments => expected`).
Design: `grouped-errors-and-labels-implementation-plan-2026-09-30.md`.
Checklist: `grouped-errors-and-labels-implementation-tasks-2026-09-30.md`.

## Review base and commits

- Review base (verified clean of feature code): `5804b6865897d34952c93429ac0feb10d63105b9`
- Final HEAD at validation: `8f6e884f3d57f88b2959a7e52d9d86c6e6175afd`
- Checkpoint (design evidence): `d6d5e6b1` (`docs(grouped-errors): checkpoint design evidence and pending language docs`)
- Sequence after base: `d6d5e6b1` docs checkpoint, `4e0dc54f` syntax, `fd7567d0`
  check, `6330400e` emit tests, `a8cc5624` editor, `92abe670` V01 fixes,
  `447f2273` V02 fixes, `6515b2de` V03 fixes, `8f6e884f` V04 fix.
- Working tree at handoff: clean except pre-existing unrelated untracked docs
  (`manolea-experience-review-2026-09-30.md`, `native-can-test*`, `preparation/`);
  plus this record and the ledger/status updates below, committed or pending as noted.

## Tasks completed

P01, P02, S01-S03, E01-E03, A01-A03, X01-X02, U01-U02, D01 (draft; pending-status
notes kept until acceptance), V01-V04. R01 attempted twice via workflow plus one
native-subagent fallback; all failed before start with environment
`Too many open files (os error 24)`. F01/F02 pending on review.

## Validation results (all observed, serial queue, shared cache)

- V01 `GOMAXPROCS=2 go test -p=2 ./compiler/internal/syntax -count=1` → ok.
  Two initial failures fixed in maintained tests: removed the now-valid grouped
  arm from the stale rejection list in `declarations_test.go`; corrected trivia
  comment spacing in `grouped_rows_test.go` to the established two-space gap.
- V02 `GOMAXPROCS=2 go test -p=2 ./compiler/internal/check -run 'TestGrouped' -count=1`
  → ok (17 tests). Full `./compiler/internal/check` package → ok. Fixture fixes:
  alias-rejection test scans parser diagnostics across recovery output; coordination
  test uses `codec` + registry-declared local error `stale` via
  `programFixtureRegistry` (avoids the `text` package/`fn str text` collision).
- V03 `GOMAXPROCS=2 go test -p=2 ./compiler/internal/emit -run 'TestGrouped' -count=1`
  → ok (7 tests, incl. execution proofs of occurrence/payload preservation, shared
  fixture FIFO, independent roots/reports/queues, scenario/template selectors).
  Full `./compiler/internal/emit` package → ok with `CAN_BUN` set to the bun binary.
  Fixture fixes: bare same-package scenario links; dispatch-scoped no-`.create()`
  check (fixture expectations legitimately construct errors above the dispatch).
- V04 `./compiler/internal/driver -run 'TestGroupedCompletionDefinitionJumps'` → ok
  (4 subtests); full driver package → ok; `bun run test:grammar` in
  `editors/vscode` → 11/11 including the grouped-label case.
- `gofmt -l` clean on touched packages; `git diff --check` clean at checkpoints.
- Full emit without `CAN_BUN` fails in `map_batch_workers_test.go` (G80); verified
  identical failure on a clean base worktree (since removed), so pre-existing and
  unrelated. No authored runtime TypeScript changed, so `bun run check:runtime`
  was not required.
- No benchmarks, broad builds, or private caches. One base-verification worktree
  was created and removed; no temp dirs, bundles, or sessions left behind.

## Changed files

- `compiler/internal/syntax/{ast.go,matches.go,coordination.go,declarations.go,format.go,grouped_rows_test.go}`
- `compiler/internal/check/{completion_matches.go,program.go,coordination.go,assertions.go,grouped_completions_test.go,grouped_assertions_test.go}`
  (`templates.go` intentionally unchanged; `wrap.go`/`aggregate.go`/`ir/regions.go`
  audited read-only; `project/fixtures.go` and `driver/fixes.go` audited read-only)
- `compiler/internal/emit/grouped_syntax_test.go` (no production emitter change)
- `compiler/internal/driver/{diagnostics.go,grouped_syntax_test.go}`
- `editors/vscode/{syntaxes/can.tmGrammar.json,test/grammar.test.js}`
- `docs/implementation/{assertions.md,completions.md}`,
  `docs/syntax-taste/{decisions.md,technical-spec.md}` (pending-status notes)
- This record; task ledger progress log; plan status line.

## Known limitations and residual risks

- R01 independent review in a fresh context has NOT run (environment fd
  exhaustion blocked workflow children, native subagents, and shell). The
  copy-pasteable review request below is the handoff for that gate.
- Parser recovery after a rejected grouped alias emits a leading generic
  `expected name` diagnostic before the correct grouped-heads diagnostic;
  pre-existing recovery behavior, out of scope, noted for the reviewer.
- `docs/implementation/completions.md` also carries `emits []` → `emits {}` and
  `ok error(...)` → `ok error{...}` modernization from the inherited draft.
- X-lane execution sequences for scenario/template selectors were recorded in
  the emit fixture header without a lane-A published sequence; reconcile at review.
- `mode-all` plain-concurrent grouped arms share lowering with the covered race
  path but were not executed; reviewer decides whether to add a case.
