# a67: testing + contracts whole-story design

Status: proposal (pre-decision, revised per chatbot
verdicts 2026-09-17). No rule, no code, no golden.
This doc settles the testing story (including the a68
`runLinkedPure` shape) and proposes the contracts-epic
design (F# borrow #2, whose first step is a design
proposal). Refinements stay parked. Implementation slices
follow only on greenlight, probe-first, one per version.
Note: the verdicts were given against the Part 2 claims
as quoted (the doc was uncommitted at the review
snapshot); the replacements below are applied verbatim
in substance.

## Part 1 — Testing story (settled)

### 1.1 Hermetic units: kept

Decision tables + `given` keep their semantics (verdict B).
A caller tests its response to any permitted outcome,
including errors the current provider never produces,
without depending on incidental provider behavior.
Failure injection is the point, not a limitation.

### 1.2 Arm coverage: kept

CAN4107 stays exactly as is (verdict C, pinned by
`compiler/diag_4107_test.go`): taken-ness is execution of
the particular arm under a passing test, or the existing
local identity-relay certificate. Script presence is not
coverage. The relay exception stays local-only; foreign
calls never qualify.

### 1.3 Linked-pure integration: the one addition (a68)

Adopt verdict B's `runLinkedPure` contract unchanged:

- Explicitly selected by committed Go tests only. No new
  `.can` call syntax; `checkGiven` default untouched.
- Inputs: explicit module set, root function + revision,
  signature-checked args. Module presence never implies
  execution (verdict's core objection to presence-based
  admission).
- Precondition: ordinary whole-program checks, tables,
  revisions, export certification green first.
- Reachable graph: checked pure CAN or deterministic
  kernels only, all branches inspected. Reachable
  externs, state ops, effects, unresolved calls, or
  cycles refuse. Cross-module cycles stay rejected;
  checked direct self-recursion allowed.
- Execution: real bodies across modules, no `given`
  consumed, no script fallback, no mixed mode.
- Identity preserved (`Module.ID`, ownership,
  certificates); fresh context per vector.
- Failure fails; nothing is "trusted". Traces stay
  separate and never satisfy CAN4107.
- Falsifier (verdict): `runLinkedPure(middle__copy@1,
  "A")` returns `""` against `"A"`, order-reversed,
  plus a correct-leaf positive control.

### 1.4 Diagnostics: done

a62 (unknown-callee single diagnostic) and a63
(stale-arm nesting hint) shipped. No further
diagnostic work is queued by this story.

### 1.5 What testing does not give

Execution is not verification. A scripted response
consistent with a contract is still not execution or
verification of the provider body (borrow-doc §2
boundary). §1.3 closes the execution gap; Part 2
closes the universality gap. Neither subsumes the
other, and real execution alone cannot exercise a
hypothetical provider failure admitted by an
upper-bound `emits`.

## Part 2 — Contracts epic (proposal)

### 2.1 Outcome-indexed contracts

Borrow-doc §2 shape, unchanged: `requires` preconditions
plus `ensures` postconditions indexed by the actual
outcome (`on Ok result ...`, `on <err> err ...`). Every
declared error outcome gets its clause set: when the
error is permissible and what its payload means. The
`max` and validator sketches in the borrow doc are the
reference examples.

### 2.2 First-cut verification logic (revised per verdict)

The first cut admits Boolean constants, sort-correct
equality, record projections, linear integer
expressions and comparisons, and exhaustive Boolean
case expressions. Multiple predicate clauses are
conjoined. The maximum and integer-range-validator
pilots require no user-written quantifiers or stronger
arithmetic. Contract verification is universally
quantified over inputs satisfying `requires`; this
does not add quantified expression syntax.
Unsupported predicates or proof obligations are
rejected, and inconclusive verification is not
acceptance. The validator pilot retains `requires
true` and specifies both `invalid_bounds` and
`out_of_range`. Checking a declared condition from
known facts is verification; it is not permission to
invent types, preconditions, or effects.

The borrow sketch's Boolean `match` is the
disjunction spelling (finite Boolean expression, not
a quantifier): `match result.value == left / true =>
true / false => result.value == right` is logically
`result.value == left OR result.value == right`. The
membership clause is not redundant: a fixed constant
already fails unbounded inputs, but the clause
additionally excludes input-dependent overlarge
results (maximum plus one).

Validator pilot, completed against the existing
`std__validate__int_range` (which declares both
`validation.invalid_bounds` and
`validation.out_of_range`, see
`std/quota/quota.can`): the parent excerpt's `Ok` +
`invalid_bounds` alone is an incomplete outcome
contract (e.g. value `-1` in `[0,1]` satisfies
neither), so the pilot specifies all three outcomes
and their input partition (`lower > upper` →
`invalid_bounds`; in-range → `Ok` preserving input;
otherwise → `out_of_range`). Narrowing `requires`
to evade a case (e.g. `requires lower <= upper`) is
the forbidden silent-domain-narrowing of §2.3, not a
fix. For the initial proof implementation the
supported body language stays explicit too:
unsupported body operations or unverified callees
are never silently assumed facts.

### 2.3 Contract revision identity (revised per verdict)

Contracts are part of a function revision's immutable
reviewed interface. Before verification is introduced,
a dedicated slice must define canonical contract
identity, its trusted comparison baseline, and the
diagnostic for same-revision drift. A revision-number
comparison is insufficient. The first mechanism
rejects every normalized contract change at the same
revision; it does not attempt semantic compatibility
inference. Verification certificates are additionally
invalidated by body or dependency-contract changes.
Ordinary checking never silently refreshes either
baseline or certificate.

Minimal mechanism (dedicated slice, before verifier):

- Identity: contracts belong to the existing function
  revision, not a floating contract revision.
- Fingerprint: canonicalized interface + `emits` +
  `requires` + outcome-indexed `ensures` + referenced
  record/error shapes. Exclude positions, comments,
  test rows.
- Baseline: explicitly supplied, previously accepted;
  never silently regenerated or overwritten. A
  mutable lockfile compared only with itself proves
  nothing; the baseline comes from the review base.
- Same revision, changed fingerprint: reject, even if
  apparently compatible or equivalent.
- Changed revision: callers stay pinned, no automatic
  retargeting; unavailable old revisions keep failing.
- Removal is a contract change, not an unchecked
  default.
- Proof freshness: body or dependency-contract edits
  invalidate the proof with the contract unchanged.

Diagnostic: new symbolic code (e.g.
`CodeContractRevision`, numeric code allocated at
implementation) pointing at the changed
`requires`/`ensures` clause in the provider (or the
fn declaration when a block was removed), naming the
old revision and baseline. Probes: same-revision
`requires` strengthening, `ensures` weakening or
removal, changed error-payload predicates, contract
removal; whitespace/comments/additive rows stay
silent; a proper bump leaves stale callers visibly
stale.

An agent must not "fix" a failing proof by silently
narrowing the public input domain (borrow-doc §2;
e.g. `requires x >= 0` to `x >= 1` at the same rev
while all rows use `x = 2`). Private kernels may
`require` (callers prove); public functions validate
untrusted input into declared errors.

### 2.4 Tables stay mandatory (confirmed per verdict)

Concrete admitted test inputs are still required for
executable functions (borrow-doc acceptance test):
contracts quantify universally, tables witness
concretely. CAN3110 (scripted-Ok contradiction) stays
as the linkage check between the two. Precise rule:
every executable function retains its mandatory
decision table; every ordinary input row must satisfy
that function's `requires` (a violating row is an
error, not a skipped test); at least one admitted row
must execute successfully against its expected
outcome, which may itself be a declared language
error. Coverage stays the existing passing-execution
law plus the authorized identity-relay exception.
Tables are the established anti-vacuity mechanism
(they reject impossible preconditions such as `x >=
0` with `x <= -1`, which no row can inhabit); no
contract-only executable category is added in this
epic (proof-only declarations would need the parked
ghost design). Tables do not prevent every narrowing
(`requires x >= 100` keeps its `100` row), which is
why §2.3's revision mechanism is separate.

### 2.5 Three complementary obligations (revised per verdict)

Three complementary obligations, with overlapping
defect detection. CAN3110 rejects a scripted success
contradicted by a successfully modeled provider
execution; absence of a contradiction does not
establish agreement when the provider could not be
modeled (scripted errors are deliberately trusted,
and "not contradicted" is not agreement). CAN4107
requires execution evidence for each source arm under
passing tests, subject only to the existing
authorized identity-relay exception (no "proved
unreachable, therefore exempt" rule is introduced
while ghosts stay parked). Contracts establish
outcome-indexed properties of verified bodies for
all inputs satisfying their declared preconditions,
within the supported verification model. These
obligations do not replace one another, may reject
the same implementation defect (beneficial
redundancy: e.g. a wrong `max` branch fails both a
scripted check and the membership clause), and do
not prove that the written specification captures
the intended operation (specification adequacy, e.g.
a missing is-one-of-them clause, is a specification
defect no proof invents).

Two boundaries. First, `given` rows are not
universal assumptions: the verifier establishes
call preconditions and reasons from verified callee
contracts, never from examples or injected outcomes.
Second, preserving `given` means preserving the
injected-error/actual-outcome distinction: existing
failure-injection scenarios are not automatically
rejected because a new contract proves the provider
would not produce that error there, nor do they
become production-reachability evidence. Changing
their admissibility needs a separate explicit
decision.

### 2.6 Parked, explicitly

Standalone refinements, ghosts, lexicographic
decreases: parked (unchanged). First follow-up on top
of this epic is cell state contracts
(`quota__consume` unchanged-on-error), already queued
behind #2. Refinements stay parked per 2026-09-17
decision even though §2.1 founds them.

## Part 3 — Slice order (needs greenlight, rationale confirmed per verdict)

Implement a68 `runLinkedPure` before the contracts
slices as a concrete linked-execution oracle and an
independently tested preservation of module identity.
This is not a prerequisite of the contract logic and
does not turn tests into universal proofs; neither
feature's finite examples select the other's
semantics. Scheduling criterion: a68's
normal/scripted versus linked/real distinction is
visibly tested before contract checking starts using
callee summaries. Contract identity/revision design
(§2.3) may proceed in parallel on paper, but its
enforcement mechanism must be specified before
verifier implementation.

1. a68: `runLinkedPure` (§1.3) — shipped.
2. a69: `requires`/`ensures` grammar + AST — shipped
   (parse and store; proves nothing).
3. a73–a76: closed tagged unions (borrow #3) — shipped.
4. a77: revision identity enforcement (§2.3) — shipped
   (CAN6013; proof caching explicitly out of scope).
5. Verifier core (§2.2 logic), now unblocked, sliced
   probe-first per the §2.2 verdict: a80 admission /
   well-formedness (no solver), a81 obligation
   generation + solver boundary with modular-call
   handling and pilots (max/validator/composition),
   a82 activation + evidence reporting. No verifier
   slice reopens identity enforcement.
6. Stdlib contract pilots land after the verifier
   proves them; unsupported contract-bearing
   interfaces do not ship as accepted source.

Each slice: failing probes first, existing `.can`
untouched except additive rows, committed
docs/generated, `go test -count=1 ./...` +
`modcheck` + `gramcheck` green.

## Rollback

Delete this file. Nothing else references it yet.
