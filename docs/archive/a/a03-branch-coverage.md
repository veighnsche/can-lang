# v0.3 — Test-per-arm law (spec)

Status: shipped. Panel runner-up, first true language feature after the
machine-artifacts bundle.

## Rule

Every match arm must execute at least once across its function's
decision-table run. An arm no test takes is dead code or a missing test:
both are compile errors, `CAN4107`, at the arm's row, naming the pattern:

- `no test takes "on db.down _" in auth__login`
- `no test takes "on true" in db__get_user`
- `no test takes _ in db__get_user`

Pattern rendering: `variantWild` → `on <Name> _`, `variant` →
`on <Name> <var>` (`on Ok u`), `bool` → `on true/false`, `str` →
`on "<s>"`, `wild` → `_`. The squiggle covers the kind/name token on the
arm row (whole line fallback as usual).

Every `emits` kind must be produced during the run or stubbed by a caller
test in the world: the existing stale-emits diagnostic (`CAN4003`) is
promoted from warning to error with no message change. Together the two
rules are airtight: every static raise site sits in a taken arm or an
always-evaluated plain body/scrutinee, so a green, covered program
provably exercises its whole error contract. (Known wrinkle, out of
scope: call-scrutinee argument expressions are never evaluated by
`evMatch`, so a constructor there counts as raised without executing.
Fixing that means evaluating scrutinee args, a separate proposal.)

## Hit semantics (dynamic, not static)

Taken-ness is observed, never inferred: the evaluator records
`(match node, arm index)` when a pattern matches, before evaluating the
arm body (an arm that errors mid-body still counts as taken). Static
reachability analysis would duplicate the evaluator and drift from it;
the decision table already runs at compile time, so coverage is free.

- Shadowing falls out for free: a shadowed arm can never match, so it
  can never be taken. Dead-arm detection is not a separate rule.
- Wildcard arms are taken by fall-through tests, like any other arm.
- Functions with no tests are exempt (the missing-tests diagnostic
  already fires); coverage is assessed over green tables only — a
  function with any failing test reports the failures, not coverage.
  Fix red first, then prove covered. This also keeps demos crisp:
  a file demonstrating a failure shows its failure, not cascade.

## Implementation

- `Ctx` gains `Cov map[*Node]map[int]bool` (nil means untracked; the
  evaluator stays safe for all existing callers).
- `evMatch` records the taken arm index at each of its six match sites.
- `runTestTracked(fn, test, prog, cov)` wraps the existing split;
  `runTest` keeps its signature with nil coverage.
- `checkSem` builds one coverage map per function, runs its tests
  through it, then emits `CAN4107` per untaken arm. Shared by editor
  and CLI; no separate CLI path.
- `allCodes` gains `CodeArmUntaken = "CAN4107"`.
- Stale-emits: severity warning → error in `checkEmits` only.

## Consequences (accepted before building)

- `docs/archive/sketches/auth-login/auth.can` does not comply today: the inner retry
  arms and both password/lockout bool arms are unreachable under the
  current four tests. Shipping this law requires extending the flagship
  decision table (`locked`, `badpw`, `retry_badpw` with scripts at both
  call nodes), which regenerates the normalize and errors.json goldens.
  The law found real dead arms in our own example; that is the point.
- `broken-login/stale-emits.can` turns red (was yellow); its header
  comment is updated. `foreign-raise.can` and `dead-script.can` gain
  passing tests so each demo keeps exactly its titled squiggle(s).
- JSON golden gains no new lines from existing fixtures (all their arms
  are taken); the `CAN4003` line changes severity warning → error.

## Open decisions (do not block)

- Whether `normalize` should refuse uncovered programs (leans: yes, it
  already gates on the full suite, so it inherits this law for free).
- Whether shadowing deserves its own earlier, clearer error instead of
  surfacing as untaken-arm (leans: no; one rule, one message).

## Amendment (retrospective): the law states evidence, not execution

The accurate obligation is execution evidence **or the specifically
authorized structural certificate** (identity relay, `CAN4108`) —
"every arm executed" is no longer literally true, and certified arms
must never acquire fabricated hits. Likewise `emits` states Safety
(escaping errors belong to the declared set), not Realization: the
living requirements allow unrealized entries, and no enforcement of
"every declared error observed escaping" exists. Report executed,
certified, and uncovered as separate categories.
