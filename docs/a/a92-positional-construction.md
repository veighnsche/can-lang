# a92: positional construction — design (decided)

Status: decided 2026-09-18. Resolves `REQUIREMENTS.md`
open question #2 toward positional: value positions take
positional args with names optional, calls unchanged.
Parent: a74 ("positional construction stays outside the
subset" — this amendment admits it), a12 (complete
constructions: all fields still required, named or not),
can-idioms C6 (redundant names; guide since removed).

## Rulings

- **Positional binds by list index.** An unnamed arg at list
  index i claims field i (declaration order); named args
  bind by name in any order. Same rule as calls
  (`bindSlots`), not the stricter test-row rule: no
  "positional after named" fault exists.
- **One mechanism: resolve in checkCtor.** `checkCtor`
  assigns names by mutation (idempotent: resolved args are
  named, so a second run is a no-op). Types, runs, proofs,
  and emit all see one named form — emit output is
  byte-identical either way, so migration is output-neutral
  by construction.
- **Types run before a18.** `checkTypes` moves above
  `checkScriptConsistency`: mutation-before-eval is the
  invariant, keeping the honesty proof total for positional
  outcomes. Both checks are defensive about the other's
  faults, so the swap is safe.
- **C6 expands to constructions.** can-idioms C6 (guide since removed;
  "the only lists with a positional spelling") lapses its construction
  exemption: an in-order named field is reported. Exchange
  args stay exempt (grammar-mandated names).
- **Faults stay CAN6003.** Unknown, missing, and repeated
  fields already share the conversion-flavored code with
  free-form messages; over-arity and double-claim join them.
  No new code.

## Goal

Kill the token tax on constructions: `name = ` prefixes are
20.8% of `std/quota/quota.can` (540 sites, ~1.4k
chars÷4 tokens), all repeater names the declaration already
states. `Quota__Request("Ada", 5, "batch")` must compile,
run, prove, and emit exactly like the named form.

## Problem

Calls and test rows spell positionally, but every
`Type(...)`, `Ok(...)`, and error construction demands
`field = value` (a74). The asymmetry has no remaining
justification: order-independence was never load-bearing
(fields rarely reorder; same-type reorder is caught by
review, cross-type by the checker), while completeness
(all fields required) and payload proof (a12) survive
positionally — every field is still present, in a fixed,
declared order.

## Recommendation: positional construction (ADOPTED)

- Records, variant cases, dotted errors, and `Ok` against
  its wanted record accept positional args; mixed
  positional + named follows the calls rule.
- `Bytes` keeps its literal-only rule; exchange args stay
  all-named (grammar); patterns have no field syntax and
  are untouched; seal payloads stay exempt from C6.
- `TestLintStdClean` (std must lint clean) forces the
  migration in-slice: blessed code goes slim the same day,
  enforced by C6, not by convention.

## Rejected

- **Numbered slots** (`Point(0 = 1, 1 = 2)`): dominated.
  More tokens than bare positional, brand-new syntax for a
  spelling no other position uses, same reorder hazard,
  and worst readability (count-and-map per arg). Its only
  unique properties (rename-proof, reorderable) are thin:
  renames are compiler-checked breaks.
- **Names-required Zealotry** (status quo): keeps a 20%
  token tax to protect against field reorders the checker
  mostly catches anyway.
- **Reorder without pipeline move**: leaving `checkTypes`
  below a18 would silently trust every positional outcome
  (the sandbox rejects what types have not resolved yet).
  The lie would still fail at execution (CAN4200), but a
  proof hole by default is worse than a one-line move.

## Slices

- **S1: rule.** checkCtor binding + faults, pipeline move,
  eval reword, focused tests (accept, over-arity,
  double-claim, mixed, a18-still-proves-positional).
  Acceptance: full gates green, migration not started.
- **S2: C6 expansion.** `lintRedundantNames` walks
  constructions (records, cases, errors, Ok-via-return);
  gallery re-pinned (`stale.can` goes positional, stays
  one finding). Acceptance: `TestLintStdClean` fails only
  on std (proving the lint sees the fat), gallery green.
- **S3: migration.** All blessed `std/` slimmed
  mechanically; emit goldens prove output-neutrality.
  Acceptance: full gates green including `TestLintStdClean`.

## S3 rulings (migration discoveries)

- **Scalar Ok is the `value` singleton.** Against a scalar
  want, exactly one field named `value` exists by
  convention; a lone positional binds it, anything else
  positional faults. Payloads stay unchecked exactly as
  before (the old path was a bare return): projections
  like `v.value` on an int-typed binder are
  runtime-shaped, and checking them now would fault
  programs that run. C6 flags only the single name
  `value`; any other single name (`Ok(id = x)`) has no
  positional form and stays.
- **Raw contexts bind at evaluation.** Given outcomes,
  test expectations, and linked expect strings evaluate
  without a checking pass, so `evSmall` names positionals
  from declaration order (`bindCtorArgs`): Ok takes
  `[value]`, errors/variants/records their declared
  fields. Named verification (unknown/missing) and all
  error wording stay exactly as before; only positionals
  gain meaning. Checked bodies arrive already named.
- **The a18 sandbox binds unchecked files.** `diagnose`
  visits the open file only, so a callee body from another
  file reaches the sandbox unbound; the sandbox runs the
  checker's walk bind-only (faults discarded — they belong
  to the callee's own diagnosis) before evaluating.
  Idempotent: checked trees keep their names.
- **Linked expects bind against the root return.** A
  linked expect string parses raw; it binds against the
  root function's return — the same want the provider body
  saw — so record returns compare field-wise. A bad expect
  stays an evaluation mismatch, never a diagnostic.
- **Bound names point squiggles at values.** A type
  mismatch on a positional arg underlines the offending
  value (`tokenOf`), since the bound name never appears in
  source; named args keep underlining the name.
- **Shape checks accept positional.** The utf8-exporter
  proof (`export.go`) takes the lone Ok arg named or
  positional; only a contrary name breaks the unchanged
  shape. The relay detector already binds positionals by
  slot.
- **Migration is C6-driven.** Slim exactly the names C6
  flags — the flag is the proof of in-order safety. Never
  slim output text (printer goldens, normalize wants,
  span tokens, `Contains` pins): the printer still writes
  every name, and a92 changes construction input only.
  Never slim negative pins (fault shapes are
  load-bearing); step `Replace` mutations with their
  fixtures. Exchange arg names are grammar-mandated and
  never slimmed.

## S3 outcome

`std/` slimmed (3020 names across 7 modules), `sketches/counter`
and `sketches/lint-errors` migrated with it; remaining
sketches stay named (non-goal: legal, migrate
opportunistically). Output-neutrality verified by
re-emission: every module's `.ts` and `errors.json`
rebuild byte-identical (html with ascii+scalars, quota
with scalars, the rest standalone). `TestLintStdClean`
green; full gates green.

## Non-goals

No grammar change (parseArgs already parses positional);
no TextMate/modcheck/gramcheck change; no sketches
migration (named stays legal; demos migrate
opportunistically); no formatter decision (R10 still
pending — when it lands, positional is canonical except
where C6 keeps a name doing work).

## Rollback

Revert S1: positional construction is again "has no field"
(CAN6003) statically. S2/S3 revert mechanically (lint
exemption restored, names re-added). No data migration:
source spellings only.
