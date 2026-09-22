# a88 — match-arm minimization check (DRAFT proposal, not accepted)

A linter/formatter pass that minimizes each function's match
arms: fewer arms, identical outcomes. Manual sweeps (role/ws
or-folds, tokens/digest-tail chains) proved the savings are real
but hand-found; the check makes them mechanical.

## Problem

Idiomatic arms are found by inspection today. The `role` ladder
(two identical true-arms) and the `ws` ladder (five identical
true-arms) survived every gate for months because nothing measures
arm redundancy: the checker rejects *stale* arms but never suggests
*mergeable* ones. Each missed fold is nesting depth every reader
pays, and hand-folding is what introduced the only regressions in
recent sweeps (mis-indented `then`, shifted splice anchors).

## Semantics (recommended)

The check works one match at a time, syntax only:

1. **Group** a value match's arms by RHS syntax (normalized
   whitespace, alpha-renamed binders excluded — see below).
2. **Merge** groups of size ≥ 2 whose patterns are compatible on
   the same scrutinee(s) into one or-pattern arm (`|`), keeping
   the first arm's line for diagnostics and coverage. Arms whose
   binders feed the RHS never group (see below).
3. **Report**; rewrite only under an explicit `--fix`.

Equivalence oracle is the existing stack, not a new proof:
`canlc normalize` byte-identical over the module, plus the full
gates (`go test ./...`, modcheck, gramcheck, tsc). Merged arms
stay covered: rows select by outcome, so rows that selected the
old arms still select the merged one, and CAN4107 needs no
adjustment. Idempotence is the acceptance test: the check on
its own output reports nothing.

## Checker obligations

1. **Same scrutinee only.** Cross-scrutinee ladders (the
   pre-chain approve shape) never merge — that is the chain
   combinator's job — out of scope here.
2. **Binder discipline.** An arm whose binder (or wildcard
   position) is referenced in its RHS never merges; alpha-renaming
   does not make two binders one. Value matches only — call-match
   `on` arms, guards, and `given` tables are out of scope.
3. **Wildcard stays separate.** `_` inside an or-pattern is
   CAN4112, so a wildcard arm never folds into an or-arm; it
   remains the fallback arm.
4. **Byte safety.** Sources carry literal NUL/control bytes with
   documented positions; the check reads and writes bytes, never
   normalizes them, and any `--fix` run must preserve the NUL
   count exactly (verify with `tr -d -c '\000' | wc -c`).
5. **Check before fix.** It ships as a check-mode reporter first
   (mergeable groups with projected line savings); rewriting
   follows only after the reporter runs clean across the repo.

## Non-goals

Pattern synthesis (discovering `0..31` from scattered codes),
boolean threading (`and`-folding `@`-rejection and authority
state), and any cross-match restructuring — that is Level-2
work needing the verifier's value tables as oracle, a design
note of its own. No readability judgments: the check counts arms,
never style. No new diagnostic codes for the reporter; `--fix`
output must introduce none either.

## Open questions

Where the check lives (`canlc` subcommand vs `tools/` vs a check gate);
warn vs fail in CI; the tool's name (no working title proposed here); whether
merged-arm line attribution should point at the surviving arm
or the whole group for coverage; interaction with the editor
(LSP code action vs CLI-only).

## Golden test capability

The reporter output is golden-testable, following the
`jsonGolden` / `normalize` precedent (byte-compared committed
goldens, not self-asserting probes):

1. **Canonical report rendering.** One line per mergeable group,
   field-ordered and sorted by file, line, then column
   (`file:line:col: surviving-arm <= folded-arms (saved N lines)`),
   so formatting churn never shifts goldens. The sort is by
   source position, never map iteration.
2. **Committed report goldens.** Each fixture is a small `.can`
   input plus its expected report golden, covering: a 2-arm
   merge (the `role` shape), a 5-arm merge (the `ws` shape),
   binder refusal, guard/call/`given` refusal, wildcard
   separation, and the empty report on already-minimal input
   (idempotence golden).
3. **`--fix` before/after goldens.** Fixed sources are committed
   as goldens alongside their inputs; equivalence is shown by
   `canlc normalize` byte-identical over each pair, plus the
   full gates (`go test ./...`, modcheck, gramcheck, tsc).
   NUL preservation is pinned by a byte-count assertion on the
   fixed golden, not by inspection.
4. **No registry change.** Report lines are not diagnostics, so
   the "no new diagnostic codes" rule stands; goldens pin the
   report text instead of codes.

## Toward approval

Accepting this note means: implement the check-mode reporter
with committed golden tests per above (merge cases,
binder/guard/call refusals, wildcard separation, NUL
preservation, idempotence), run it across the repo, and only
then propose `--fix`.
