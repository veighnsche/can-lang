# a28 — Multi-scrutinee match (proposal)

Status: shipped. Ratified as the R6 amendment quoted below;
single-scrutinee `match` is unchanged.

Research input (2026-09-17): deep-research survey of multi-scrutinee match
in Rust, OCaml, F#, Haskell/GHC, Scala 3, Elixir, and Swift. Its citations
are taken on trust (not independently verified); the refinements adopted
below are the ones that also stand on codebase logic. Net verdict: treat the
construct as a product-pattern table over a scrutinee vector (no tuple
values in the language), prove exhaustiveness over the product space
symbolically, keep OR-patterns/guards/multi-call out of v1.

## Goal

Let a `match` discriminate on two or more value scrutinees at once, so the
user's truth-table shape works directly:

```
match x, y
  true, true => ...
  true, false => ...
  false, true => ...
  false, false => ...
```

Today `match` takes exactly one scrutinee (`Node.Scrut *Small`,
`compiler/parse.go` `parseExprBlock`/`parseMatchArms`) with one pattern per
arm (`parsePattern`: `_`, `true`/`false`, `"str"`, `Ok x` / `err.Kind x`).
The only way to write the table above is nested single matches. This plan
adds tuple scrutinees + tuple patterns with product exhaustiveness, scoped
to value matches.

## Success Criteria

- `match x, y` with per-slot patterns (`true`/`false`, `"lit"`, `_`) parses,
  type-checks, evaluates, and emits TS with the same meaning as the
  equivalent nested single matches (first matching arm top-to-bottom wins).
- Exhaustiveness is proven statically over the product: a missing boolean
  combination fails the build naming the tuple (e.g. `missing (false, true)`);
  any table with a `str` slot still needs `_` coverage, generalizing the
  single-match rule.
- No change to single-match behavior: existing goldens byte-identical;
  `go test ./...`, `go run ./tools/modcheck`, `go run ./tools/gramcheck`
  all green, plus new committed goldens for the tuple shapes.
- No new diagnostic code: arity misuse reuses `CAN4105`, coverage gaps reuse
  `CAN4101`/`CAN4104` with tuple rendering.

## Context And Current Facts

- AST: `Node{IsMatch, Scrut *Small, Arms[]}` with `Arm{Pat Pattern, Rhs}`
  (`compiler/parse.go:52-73`). `Pattern.Kind` is one of
  `wild,bool,str,variant,variantWild` (`parsePattern`, `compiler/parse.go:1101`).
- Parse: `match <small>` via `parseSmall`, arms via `reArm`
  (`^(?:on\s+)?(.+?)\s*=>\s*(.*)$`), nested `match` in RHS handled in both
  `parseExprBlock` and `parseMatchArms` (`compiler/parse.go:809,1131,1193`).
  `splitTop` is string- and paren-aware (`compiler/parse.go:208`) and already
  splits top-level commas for args/fields — the reuse vehicle for both the
  scrutinee list and the per-arm pattern list.
- Eval: `evMatch` (`compiler/eval.go:688`) resolves call scrutinees
  (store/decparts/helper/foreign+`given`) or `evSmall` for values, then
  first-matching arm wins with `markTaken` coverage (`compiler/eval.go:783`).
- Exhaustiveness: `verifyExhaustiveAll` (`compiler/eval.go:1139`) has two
  branches — call matches (want `ok` + emits, `CAN4101`/`CAN4102`) and value
  matches (bool must be exactly `true`+`false` per `CAN4103`; `str` without
  `_` fails per `CAN4104`; variant on value fails per `CAN4106`).
- Types: `tycker.node` (`compiler/types.go:754`) values the scrutinee, then
  threads bindings only for call-match `variant` arms; value arms add no
  bindings.
- Emit: `stmtMatch` (`compiler/emit.go:815`) lowers call matches to
  `switch` on the result tag and value matches to an `if`/`else if` chain
  (`boolTotal` last arm becomes bare `else`; `wild` arm emits trailing
  lines and returns).
- Codes: registry + uniqueness test in `compiler/code.go`
  (`CAN4101` missing arm, `CAN4103` bool arms, `CAN4104` value-no-wild,
  `CAN4105` bad arm kind, `CAN4106` variant-on-value, `CAN4107` arm untaken).
- Precedent docs: `docs/archive/a/a03-branch-coverage.md` (test-per-arm law),
  `docs/archive/a/a05-expressiveness.md` (proof-cost sequencing rule).

## Constraints And Non-goals

- Value matches only in v1: every scrutinee in a multi-match must be a
  non-`call` Small. Call matches (`call f()` scrutinee, `Ok`/error patterns,
  `given` tables, store/decparts special cases in `evMatch`/`stmtMatch`) stay
  single-scrutinee. Rationale: `given` scripting, emits-based want-sets, and
  the store/decparts single-value paths all assume one call; product-over-
  outcomes needs its own design, not a ride-along.
- No trailing colon: arms already delimit with `=>` and bodies nest by
  indentation (curly ban, `parseModuleText`). `match x, y:` with `:` is a
  parse error; the proposal syntax is `match x, y`.
- No nested tuples, no per-slot variant patterns in v1 (variant arms are only
  legal on call matches anyway — multi has no calls, so they keep failing
  under the existing `CAN4106` family).
- No static overlap/duplicate-arm error in v1: later duplicates are dead by
  first-match-wins and get caught by the existing test-per-arm law
  (`CAN4107`), same as unreachable single arms today.
- No `REQUIREMENTS.md` edit in v1: this doc is the proposal; the amendment
  lands only if the design is accepted.

## Key Decisions

1. Syntax `match x, y` / arms `p1, p2 => rhs`, no colon. The user's sketch
   uses `match x, y:`; rejected because can match lines never take a
   terminator — `=>` + indentation already delimit. The optional `on` arm
   prefix keeps working uniformly (`on true, false => ...`).
2. Value-only multi. Rejected alternative: allow `match call f(), y` or
   multi-call. It would fork `given` scripting (which test scripts which
   call?), emits want-sets (product of outcome sets), and the
   store/decparts single-value fast paths — a second feature wearing the
   first one's coat.
3. Per-slot patterns restricted to `bool | str | wild`. Each slot accepts
   exactly what a single value match accepts minus variants (variants need a
   call scrutinee, excluded by decision 2). Arity must equal the scrutinee
   count on every arm.
4. AST: replace `Scrut *Small` with `Scruts []*Small`; single match is the
   `len == 1` case. Rejected additive second field: every consumer
   (`check.go` ~20 `Scrut` sites, `eval.go`, `emit.go`, `types.go:762`,
   `lsp.go:400`, `catalog.go:90`, `walkCalls`) branches on one shape either
   way; one list is the smaller, uniform diff. Single-match code paths read
   `Scruts[0]`.
5. No new `CAN` codes. Arity mismatch (wrong comma count, empty slot, bad
   per-slot spelling) reuses `CAN4105` (arm shape doesn't fit the match);
   uncovered boolean tuples reuse `CAN4101` rendered as tuples
   (`missing (false, true)`); `str`-slot tables without `_` coverage reuse
   `CAN4104`. Rationale: one rule one code, and the golden JSON-diag suite
   freezes every code (`compiler/code.go`), so reuse avoids registry churn.
6. Symbolic product-space subtraction, not Cartesian expansion. Each arm
   denotes a product space (`true` = `{true}`, `_` = whole slot domain);
   `str` slots partition into mentioned literals plus an `OTHER_STRING`
   bucket (every string not literally named). The checker threads
   `covered`: per arm in source order, `useful = arm − covered`
   (empty = unreachable, contributes nothing); `covered += arm`. The match
   is exhaustive iff `total − covered` is empty. Same lineage as
   Maranget usefulness (OCaml/Rust) and Liu-style space algebra
   (Scala/Swift SpaceEngine), miniaturized to bool/str slots. v1 uses the
   op for exhaustiveness only; unreachable arms keep surfacing via the
   existing dynamic test-per-arm law (`CAN4107`), matching single-match
   behavior where duplicate value arms also collapse silently at check
   time. No GHC-style model cap and no Swift-style "unable to check"
   escape hatch in v1 — the slot fragment is too small to need one.
7. Concrete-counterexample diagnostics. Boolean gaps render as exact tuples
   (`missing (false, true)`, Rust/Swift precedent); genuinely unconstrained
   residual dimensions may generalize (`missing (false, _)`). `OTHER_STRING`
   is never rendered as `_` (that would falsely imply all strings missing):
   the witness generator picks a concrete string literal not consumed by the
   table. Cap witnesses at three; reuse `CAN4101`/`CAN4104`, no new codes.
8. Emit as simplified conjunction chains over temporaries. Scrutinees lower
   to `const $m0 = x; const $m1 = y; …`, then each arm is
   `if (c1 && c2 && ...)` with per-slot conditions (`$m` / `!($m)` for bool,
   `$m === "lit"` via existing `normStr` for str, no conjunct for `_`),
   simplifying away conditions implied by failed earlier arms (after
   `$m0 === true` fails, `$m0` is `false` — the residual arm tests only what
   is still unknown). Bool-total last arm becomes bare `else`; an all-`_`
   arm emits as trailing lines like today's single `wild`. The final `else`
   is the optimized shape of a statically proved residual arm, not trusted
   catch-all. Rejected `switch` on serialized tuples: stringly keys invent
   a runtime encoding the evaluator doesn't share.
9. Exactly-once, left-to-right scrutinee evaluation, specified now even
   though v1 admits values only (tuple-precedent languages disagree:
   Rust/Scala are LTR, OCaml/Erlang leave order unspecified, Haskell is
   lazy — inherit none of that accidentally). `_` skips *testing* its
   scrutinee's value; it never suppresses *evaluating* it.

## Recommended Approach

Ship in vertical slices behind the value-only restriction, in this order:
parse+AST → exhaustiveness → eval → types → emit → editor/tooling+goldens.
Each slice keeps `go test ./...` green by handling `len(Scruts) == 1`
exactly as today and adding the multi path only where the value-only gate
holds. Parser rejects multi-call/multi-mixed at parse time (`given` on
multi is also rejected: only single `call` matches take `given`); the
checker re-asserts it so hand-built ASTs can't slip through.

## Work Plan

1. P0 — Proposal (this file). No code. Get `Approve` / changes / cancel.
2. P1 — Parse + AST. `Node.Scrut *Small` → `Scruts []*Small`; `match` line
   splits scrutinees with `splitTop(s, ',')` (each via `parseSmall`); arm
   LHS splits the same way (each via `parsePattern`); arity + value-only
   (`call` in multi, `given` on multi) rejected at parse with `CAN4105`-
   class errors. Migrate all `Scrut` readers to `Scruts[0]` for `len == 1`.
   Nested `match` in RHS (`parseExprBlock`, `parseMatchArms` sub-branch)
   gains multi automatically. Surfaces: `compiler/parse.go`,
   `compiler/check.go` scrut-walks, `compiler/eval.go:walkCalls`,
   `compiler/catalog.go:90`, `compiler/lsp.go:400`.
3. P2 — Exhaustiveness. Extend `verifyExhaustiveAll` value branch with the
   subtraction checker from decision 6: slot spaces are bool
   true/false/any plus str literal/any over the literals-mentioned-plus-
   `OTHER` partition; per arm compute `useful = arm − covered`, accumulate
   `covered`, then report `total − covered` as up-to-three concrete tuple
   witnesses (`CAN4101`; `CAN4104` when a `str` slot's remainder is uncovered).
   Per-slot variant kinds → existing variant-on-value error. Keep the call
   branch single-only (multi never reaches it post-P1). Add LSP anchoring
   for the tuple rendering next to the existing
   `non-exhaustive match, missing ` anchor (`compiler/lsp.go:510`).
4. P3 — Eval. `evMatch`: evaluate every scrutinee exactly once,
   left-to-right via `evSmall` (decision 9), then per arm require all slots
   to match (`wild` always, `bool`/`str` equality); first full hit runs with
   `markTaken`. Error when no arm hits (unreachable post-P2, same as today).
   Surfaces: `compiler/eval.go:776-819`.
5. P4 — Types. `tycker.node` values each scrutinee; multi arms bind nothing
   (no variant slots), RHS checks with the unchanged env. No other change.
6. P5 — Emit. `stmtMatch` value path: temporaries + simplified conjunction
   chain per decision 8; single path emits byte-identical output to today.
7. P6 — Goldens + tooling + amendment draft. New cases: 2x2 bool table,
   bool+str+wild table, arity-mismatch error, missing-tuple error,
   multi-vs-nested equivalence (same decision table, same outcomes).
   Test law stays one obligation per reachable source arm, not per product
   cell: `true, _ => A` needs one hit, not separate `(true, true)` and
   `(true, false)` hits.
   `tools/gramcheck` prose productions for `match` (open question 8 in
   `REQUIREMENTS.md` names them as prose-only today) gain the tuple form.
   Draft the `REQUIREMENTS.md` amendment text (R6 `match` line) for review;
   do not apply it here.

## Validation Plan

- Highest-risk step first: P2 product exhaustiveness — enumerate by hand a
  2-bool table missing one cell and a str-table missing `_` coverage, and
  assert the exact `missing (…)` / no-wild diagnostics before touching emit.
- Per slice: `go test ./...` (golden gates + diagnosis suites must stay
  green; single-match emit byte-identical), `go run ./tools/modcheck`,
  `go run ./tools/gramcheck`.
- P6 adds committed golden tests (sketch + emit + decision-table run);
  multi-vs-nested equivalence test evaluates both spellings over the full
  input product and diffs outcomes.
- Manual: compile the Goal snippet through `canlc` to TS and eyeball the
  `if (… && …) / else` chain once.

## Risks / Rollback

- `Scrut` → `Scruts` touches many call sites; miss one and single matches
  regress. Mitigation: P1 migrates with the compiler (rename forces every
  use to be visited), and goldens pin single-match emit.
- Coverage cost on wide tables: the subtraction checker never materializes
  the product, and witnesses are capped at three, so no GHC-style model cap
  or Swift-style "unable to check" hatch in v1. If tables wider than ~3
  bool scrutinees appear, revisit with a compact uncovered-set rendering or
  an arity guideline. Never weaken to "trust the `_`" — partial `_`
  coverage is exactly what P2 computes.
- Overlap silently favors the first arm. Accepted (matches nested-`if`
  semantics); the backstop is `CAN4107` test-per-arm, not a new static rule.
- Rollback: each slice is independently revertible; P1 without P2+ is
  unreachable code behind the value-only gate (parser rejects multi until
  P2 lands — sequence P1+P2 together if a lone-P1 tree is unwanted).

## Status (2026-09-17)

Slices 1–4 implemented and green (`go test ./...`, `modcheck`,
`gramcheck`): `Scruts`/`Pats` AST, tuple syntax with parse gates,
symbolic product exhaustiveness with tuple witnesses, exactly-once LTR
evaluation, TS emit with temporaries + residual simplification + bare
`else`. The R6 amendment below is ratified and applied in
`REQUIREMENTS.md`.

Two deviations from the plan as written: `gramcheck` needed no change
(it checks keyword-level TextMate highlighting, and `match`/`,`/`=>`
need no new scope); the residual bare-`else` uses the shared cover
machinery (`valueCoverOf` + bounded `residualInLast`) rather than only
the known-facts simplification, so fully-enumerated tables like the 2×2
also lower their last cell to `else`.

## Native-match refactor (post-a28, same R6)

Follow-up unification, no language change: the match family is decided
once at parse (`MatchKind`: call outcome vs value table) instead of
re-deriving it per phase from arity and scrutinee shape. Each phase has
one value path for arity 1..N — one evaluator (`evValueMatch` over
per-slot `matchSlot`), one checker (`verifyValueMatch` over the shared
cover core, with the historical arity-1 bool/string policy preserved
verbatim), one emitter (`emitValueMatch`, inline reference for arity 1
to pin golden bytes, temporaries past that). The proof travels from
checking to emit as one bit (`analysis.emitFinalElse`); the emitter
recomputes nothing. `singleScrut`/`singlePat`, `evMultiMatch`,
`stmtMultiMatch`, and the emitter-side inventory are deleted. All
goldens and diagnostics byte-identical. Parked, not smuggled in: the
arity-1 bool+wild unification (needs its own amendment), structured
diagnostics, decision trees, OR-patterns, guards, multi-call.

## Amendment (ratified, applied to R6)

R6, after "`match` is always exhaustive; a missing arm is a compile
error, not a coverage warning.":

> A value match may take several scrutinees at once (`match x, y`,
> arms `p1, p2 => …`) with one `bool`/`"str"`/`_` pattern per slot and
> the same arity on every arm. Exhaustiveness is proven over the product
> space: a missing boolean combination is a compile error naming the
> tuple, and any table with a string slot needs `_` coverage of the open
> remainder. Arms win top-to-bottom; scrutinees evaluate exactly once,
> left to right. Call matches keep a single `call` scrutinee with
> `given` tables; multi matches take no `given`. Coverage stays one
> obligation per reachable source arm, not per product cell.

## Open Questions

- Soft arity/complexity guideline (e.g. prefer nesting past 3 scrutinees)?
  No cap proposed in v1; real tables will tell.
- Should a future v2 allow multi-call (`match call f(), call g()`) with
  product outcome coverage and per-call `given` scoping? Explicitly out of
  v1; needs its own proof design (scripting × emits × coverage). The
  research-backed stepping stone, if ever: let-bound call results matched
  as values (`let a = call f()` … `match a, b`), with `Ok`/error-variant
  arms staying a single-call feature until then.
- `REQUIREMENTS.md` R6 wording for the amendment + whether open question 8
  (formal `match` grammar) closes with this doc — deferred to P6 review.
