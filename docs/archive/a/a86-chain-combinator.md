# a86 — Sequential fallible chains (implemented; approve migrated)

## Problem

`schema__asset__approve` is a 13-deep ladder: N sequential calls, each
`on Ok x` unwrapping (`.value` / `.first` / `.entry`) and remapping every
failure to one error kind. Audit (Sept 2026) proved the ladder fully live:
all 13 failure arms reachable, each selected by committed rows (including
count 0 *and* 2, all four time edges), and `verifyExhaustiveAll` bites the
moment one arm is removed — a trial deletion of the `entry_at` fallback
broke `recheck_wrong_site` with non-exhaustive match on bool, because that
arm was the `count == 1` match's `false`, not the call's. The size is
inherent to today's language, not unmigrated idiom:

- or-patterns merge arms on one scrutinee; the ladder has 13 scrutinees.
- `and`/`or` forbid calls in operands (a78 slice 5, eager by design).
- `forward` needs shape-aligned results; every rung unwraps and remaps.

The missing feature is sequential fallible composition: run steps in
order, bind each success value forward, remap every failure to one error.

## Semantics (recommended)

A chain is short-circuit sequence over fallible steps with a shared
failure arm. Unlike eager `and`, a failed step stops the chain: later
steps never evaluate. Unlike `forward`, each step binds its unwrapped
value for later steps (`sc.first` → `entry_at` → `found.entry`). One
possible surface (keyword and shape open):

```can
match chain
  call schema__role__check(request.role) as rc when rc.value,
  call schema__url__prefix(request.url) as pc when pc.value,
  call schema__snapshot__scan(snapshot.entries, request, site, 0, #snapshot.entries + 1) as sc when sc.count == 1,
  call schema__snapshot__entry_at(snapshot.entries, sc.first) as found,
  call schema__revoked__contains(snapshot.revoked, found.entry.id, found.entry.revision, 0, #snapshot.revoked + 1) as rv when not rv.value
else schema.asset_not_approved(asset = request)
```

Steps without `when` bind unconditionally (the call must still yield
`Ok`; a yielded error takes the shared arm). Pure-bool rungs
(`snapshot.policy == policy`) stay ordinary matches or fold into one
scrutinee-`and`, which already compiles today.

## Checker obligations

1. **Elaboration, not proof.** Desugar to nested `match call` at check
   time (slice-2 precedent). The prover, coverage, and emit see the
   ladder they already understand; the trusted base does not grow a
   new evaluation rule.
2. **Step-row law.** Every step's failure outcome needs a selecting
   committed row — lifts the slice-4 usefulness rule and the
   hostile-set practice (`TestAssetHostileSet`) into check-time law.
   A step whose failure no row selects is rejected, so the audit's
   misread (an apparently dead arm) becomes a machine-checked
   property instead of a human judgment call.
3. **No `given` on chain steps.** Lifts `TestAssetNoScriptedEvidence`:
   approval-path steps execute, never stub.
4. **Exhaustiveness.** A chain is total iff the shared arm is present
   and the tail yields — extends `verifyExhaustiveAll`; extra arms are
   stale by the existing rule.
5. **Non-oracle preserved.** One error kind, step order unobservable.
   The elaborated ladder is byte-equivalent in outcomes to today's.
6. **Fault contract.** Short-circuit is explicit: unevaluated steps
   cannot fault; evaluated steps fault loudly per
   `docs/fault-contracts.md` (never become typed outcomes).
7. **Termination.** Inside `decreases` functions each call site counts
   as today — free via elaboration, no new guard syntax.

## Approve under the chain

The 13 rungs become 11 steps plus the shared arm (policy/program/
sequence fold to one scrutinee-`and` first, or three `when` steps).
Same rows, same error kind, same order — the diff is nesting depth,
and every step carries the row that selects its failure.

## Non-goals

Parallel step evaluation; error-type polymorphism across steps;
short-circuit `and` (the slice-5 call ban stands); value-level chains
(that is `and` today, eager and unchanged).

## Open questions

Surface syntax (`match chain` vs extending `match call`); one shared
trailing arm vs per-step `else`; where the step-row law is enforced
(check vs a dedicated gate); migration order (approve first, then
sinks and `recheck`, which already delegates to approve).

## Toward approval

Accepting this note means: implement as one slice with probes in the
a78 style (parse, check-time elaboration, emit parity, row-law tests,
asset-guard extensions), migrate `approve` onto it with zero row
changes, and only then consider further migrations.

## Implementation record

Shipped as `MatchChain` (`compiler/chain.go`, elaborated in
`buildWorld` beside `elaborateForwards`) with probes in
`compiler/chain_test.go`. Surface deltas from the draft above:

- Newline-separated steps with a `then` tail and one shared `else`,
  not comma-separated steps with a trailing `else`. The `then` tail
  shares the arm-result grammar (an inline `match` tail carries the
  validity window), and `else` is a single expression re-parsed per
  level, never an aliased node.
- Step syntax `call F(args) as binder [when guard]`; `when` guards
  are call-free pure Smalls, so `when not rv.value` spells the
  inverted revoked rung with no new operator.
- Pure-bool rungs stay out of the chain per the draft: approve folds
  policy/program/sequence into one scrutinee-`and` *first*, as a
  plain value match wrapping the chain.
- `given` tables are grammatically allowed on chain steps (same table
  grammar, carried onto the elaborated call); the approval path stays
  script-free through `TestAssetNoScriptedEvidence`, not a ban.
- Checker obligations hold without new machinery: elaborated ladders
  reuse admission, prover, coverage, emit, and CAN4107, so the
  step-row law is the existing test-per-arm rule pointed at step
  lines, and short-circuit/termination/fault behavior is the ladder's.

Approve migration (`std/schema/schema.can`): 13 rungs become the
`and`-wrapper plus 8 chain steps plus the window tail, zero test-row
changes. `canlc normalize` old-vs-new is byte-identical over all 122
schema decision rows; the re-emitted `std/schema/schema.ts` differs
only in `approve` (the `and` wrapper with the `$canBoolAnd` helper,
`!(rv.value)` for the inverted rung). Full gates green:
`go test ./...`, `modcheck`, `gramcheck`, `tsc`.
