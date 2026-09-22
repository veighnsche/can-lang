# Deferred: eager outcome-product matching

Status: **DEFERRED — no consumer.** This document is a deferral record,
not a ready-to-implement epic. It authorizes no compiler changes — not
even preparatory AST or scripting refactors. History: drafted as a
call-arm multi-match proposal, subjected to adversarial review, parked
under the defer-without-a-consumer precedent.

## 1. Background (preserved)

- a28 added multi-scrutinee product matches for pure values
  (`docs/a/a28-multi-scrutinee-match.md`): eager exactly-once LTR
  scrutinee evaluation, first fully-matching arm wins, proved `else`,
  proof before emit, one test obligation per arm.
- Migration converted ~45 pure tables (`std/scalars`, `std/quota`,
  `std/text`). Everything else nested is `match call` chains with
  effects — v1 serves the minority pattern. That observation motivated
  the original proposal; what follows records why it does not
  currently justify implementation.

## 2. Migration audit: the targets are not candidates

Each proposed migration fails for a structural reason. Eager products
evaluate every scrutinee unconditionally; these programs are
conditional or dependent. Preserving evaluation order (LTR) does not
preserve *which calls occur*.

- **auth-login — payload dependence.** `on Ok user => match call
  auth__verify(user.id, …)`. Slot 2 cannot name `user.id` before any
  arm binds it. Products cannot express dependency.
- **retry-loop — conditional invocation.** The fetch runs only while
  fuel remains. An eager product calls it unconditionally: behavior
  change, not refactoring.
- **html escape ladders — conditional recursion.** The recursive call
  sits under a character test. Eager evaluation recurses on every
  input: non-termination, not refactoring.
- **counter — payload dependence** (`c.value + by`).

The legitimate shape is narrow: **unconditional invocation followed
by joint outcome dispatch** (e.g. two independent credential checks
crossed in one table, or one call outcome crossed with a pure
condition). The password×OTP example establishes *expressibility* of
that shape, not demonstrated need — insufficient to overturn the
deferral precedent. A synthetic table pins intended behavior; it
cannot establish the surface is worth its obligations (call-site
scripting, product coverage, recursive-call checking, effect
accounting, regression protection).

> The original migration targets require payload-dependent or
> conditional sequencing and are not candidates for eager product
> matching. The revised feature has a plausible semantic design, but
> no demonstrated consumer currently justifies its implementation.
> Existing nested `match call` remains the intended representation
> for those programs.

## 3. Revised semantic direction (preserved, not frozen)

If a consumer reopens this, the starting design is outcome capture,
not abort-on-error:

- **Two error channels, stated explicitly.** Declared callee errors
  are *values*: each call slot captures the same `Ok` or
  declared-error outcome the single-call form captures, stores it,
  evaluation continues LTR, dispatch happens after every slot has
  produced its outcome (**D1′**). *Traps* (partial operations:
  division by zero, out-of-bounds index) are not outcomes: they abort
  before dispatch and reach no arm. The original draft blurred this
  boundary; the distinction is load-bearing for coverage, effect
  accounting, and the meaning of `_`.
- **Demand rule.** Each reached slot evaluates exactly once per
  dynamic entry, LTR, surviving lowering (no CSE across identical
  slots, no reordering). Haskell-style laziness and OCaml-style
  unspecified order are both rejected; the LTR commitment is explicit.
- **Call-site scripting identity.** `given` targets a unique syntactic
  call site; slots inside one match are distinguished by slot index
  within that owning match. Slot number alone is not identity across
  matches, and repeated dynamic entries still need a defined queue
  consumption rule (missing outcomes, unused expectations, invocation
  ordering). D1′ simplifies this — every slot evaluates once per
  entry in every passing test — but does not eliminate the
  specification. None of this machinery ships speculatively.
- **Coverage over declared regions, reachability unknown.** Product
  coverage stays conservative over the declared outcome space
  (`{true,false}` × `{Ok} ∪ emits`, emits an upper bound). Witnesses
  summarize uncovered *declared-outcome regions*; at most three
  representative examples are shown. An outcome permitted by `emits`
  but absent from tests is not proven unreachable, and a product
  combination is not proven realizable merely because its individual
  outcomes are. No reachability-analysis subsystem is to be built to
  improve these diagnostics.
- **Recursive scrutinees: requirement open.** Every recursive call in
  a scrutinee position must either be prohibited or checked in the
  pre-selection environment — arm-pattern facts cannot justify its
  decrease, and every termination/cycle traversal must visit the new
  representation. The choice between prohibition and pre-selection
  checking is deferred to implementation. (Separating case: `call
  f(n)` as a scrutinee at `n = 0` recurses before the base-case arm.)
- **Implementation shape: candidate, not commitment.** A separate
  match kind plus shared typed pattern analysis plus an explicit
  evaluation plan (temporaries from the plan, dispatch over the
  temporaries, arity-one emission adapter for legacy output) is the
  preferred candidate. Golden stability is a regression requirement
  to *demonstrate*, not a consequence of the enum design; reassess
  against the compiler's actual structure when implementation has a
  consumer. Call slots must never be lowered into effectful pattern
  predicates.
- **`_` and `else`.** `_` on a call slot covers all *value* outcomes
  of that slot; neither `_` nor `else` observes pre-dispatch traps.
  `else` ranges over the same declared domain as coverage, and the
  proof is separate from test execution.
- **Diagnostic identity follows the rule**, not the AST shape. No new
  codes are anticipated, but any genuinely new rule takes a new code
  per the one-rule-one-code doctrine; confirm at implementation time.

## 4. Preserved ambiguity resolutions

- Slot-1-major witness enumeration with a specified constructor
  order (do not depend on map iteration or declaration edits
  shifting diagnostics silently).
- Repeated binders across slots (`Ok(x), Ok(x)`) and duplicate tags
  across slots need one row-wide binding rule, stated at
  implementation time.
- If every required arm must carry an executed test witness,
  unrealized-error arms are exercised through scripted outcomes;
  the scripting rules above must make that possible wherever
  coverage demands it.
- The three-example cap is a *diagnostic-output* budget. Any
  analysis resource limit fails closed, never treating unexplored
  combinations as covered.

## 5. Reopening condition

Reopen when there is **one concrete consumer whose required behavior
is unconditional invocation followed by joint outcome dispatch**,
including continued evaluation after an earlier declared error.

That consumer must show a meaningful advantage over existing nested
single-call matches — source complexity, duplicated dispatch logic,
or demonstrated agent mistakes — while preserving the required call
trace. Mere writability with the proposed feature does not qualify.

## 6. Explicit non-commitments

- No implementation schedule. The earlier four-slice plan
  (eval → check/coverage → emit → dogfood) is dropped, not postponed.
- D7's kind design, the D4 consumption protocol, and the D6
  prohibition-vs-checking choice are all reassessed at
  implementation time against the compiler as it then stands.
- No `CAN4109`-style lint, diagnostic, or grammar affordance is
  implied for this surface; the eager-trap lint covers the shipped
  product form only.
