# v0.8 — Termination + iteration (spec)

Status: shipped. Expressiveness item 5+6: the loop shape
and its halting proof in one doc, one gate, one shipment. They spec
together or not at all (a05); this is the together.

## The pairing

R7 executes every test at compile time, so iteration without a
halting proof is a build that can hang. The proof is therefore not a
second feature: it is the admission ticket, checked in the same
pre-execution gate as the a07 cycle ban, before anything runs or
emits. A loop whose termination the compiler cannot see is a cycle
error, and a cycle error blocks test execution entirely — the hang
is refused, not recovered from.

## Loop shape: proven self-recursion

The loop is a direct self-call as a match scrutinee (`match call
f(...)` inside `f`'s own body), admitted only with a `decreases p`
line naming one `int` param. No new statement, no iterator, no
accumulator syntax: base and step are ordinary match arms, the
"loop variable" is an ordinary parameter rebound per call, and
repetition is a call like any other. One spelling per meaning
extends to control flow.

A function may call itself iff it declares `decreases p` and every
self-call site passes `p - k` for `p`'s position, where `k` is a
positive `int` literal. Named or positional, each site checked on
its own terms; sites with the wrong arity belong to the arity rule,
not this one. `p - 0` is not a decrease. `p - k` where `k` is
computed is not a decrease the compiler can see: no inference, per
the project goal — if a behavior can be declared, it must be.

> Amendment (a11): tightened — see `a11-recursion.md`. Only the
> unit step `p - 1` is admitted (larger steps refused), and every
> site must sit under the false arm of the canonical `p <= 0`
> guard (unguarded sites refused as `CAN3009`). The recursion ban
> itself is program-wide, not same-file-only.

Why structural decrease and not the other two candidates: fuel
without a decrease rule is just an int parameter (it proves
nothing by itself — the site rule is what makes fuel mean
termination); total combinators need functions as values, which do
not exist. The decrease rule is the whole mechanism, and it is
small enough to check syntactically.

## The halting argument (written down, not waved at)

Each self-call strictly decreases `p` by at least 1. Entry to a
`decreases` function with `p < 0` is a loud evaluation error, so
every entered call chain walks a strictly decreasing sequence of
non-negative ints, which is finite. Non-recursive local calls form
a DAG (a07), foreign calls never execute. Therefore every test
terminates. The runtime entry check is exact-or-loud philosophy, not
the proof: the proof is static and runs before execution; the check
turns a missed base case into a test failure instead of a hang.

> Amendment (a11): superseded by the a11 theorem — see
> `a11-recursion.md`. Unit steps under the canonical guard make
> negative entries take the base arm, so every admitted invocation
> returns a declared outcome; the loud entry check is deleted at
> both routes. Cross-file cycles are refused program-wide, so the
> "foreign calls never execute" sandbox argument no longer carries
> the production claim alone.

What the proof does not cover, explicitly: mutual recursion stays
banned (`CAN3005` — lexicographic and global decrease arguments are
a proposal of their own, not smuggled inside this one). `dec`
decreases are out (ints only; the rule stays total). The 1024-deep
sandbox bound stays as a resource bound, loud on breach, same class
as int overflow: it is not the proof and never was.

> Amendment (a10): the int-overflow comparison is retired —
> unbounded ints have no overflow mode (`a10-numerics.md`). The
> sandbox bound itself is unchanged.

## Static rules (one rule, one code)

- `decreases p` must name a declared `int` param (`CAN3006`).
  Unknown name and non-int param are the same mistake: the
  annotation proves nothing about that name.
- `decreases` with no self-call site is stale (`CAN3007`), same
  family as stale emits: proof text that proves nothing is
  misleading, not harmless.
- A self-call site that does not pass `p - k` is not a loop step
  (`CAN3008`): unchanged `p`, `p + k`, `p - 0`, computed steps,
  and literals all fail here with the site line named.
  Amendment (a11): unit step only — the message now reads
  `pass p - 1`, and larger steps fail here too.
- A self-call site outside the positive branch is not admitted
  (`CAN3009`, a11): recursion must sit under the false arm of the
  canonical `p <= 0` guard.
- A self-call with no `decreases` line is the existing cycle error
  (`CAN3005`). That is the "would hang" shape, refused.
  Amendment (a11): the shape is finite either way — the refusal is
  proof-gated admission, not a diverted hang (`a11-recursion.md`).
- All three new errors block test execution exactly like the cycle
  ban: the gate is prove-first, run-after. `CAN4200` stays the
  coarse bucket for the loud runtime side (negative entry).

## Unchanged machinery (proof by non-interference)

- Parse gains one metadata line, `decreases <param>`, beside
  `emits`/`tests`; a duplicate is a parse error like a duplicate
  `given`. No expression syntax changes.
- Exhaustiveness: `match call self(...)` covers self `emits` plus
  `Ok` through the same table. Coverage: base and step arms obey
  the test-per-arm law; recursion needs tests that recurse and
  tests that stop.
- `given`: a self-call is a local call, so no table at the site;
  inner foreign tables script every reaching test, and a site hit
  N times per test takes an N-element list (R8 already says this:
  "same site hit twice (loops) = multi-element list").
- Types: the step expression typechecks as `int` through the
  existing binop and call rules; arity and brand discipline are
  untouched. `uses`/`provides`: self needs no pin.
- `modcheck`: untouched (expression-level; verified by running).
- Grammar: `decreases` joins the control-keyword rule;
  `gramcheck` gains a sample, per a06 precedent.

## TS emit (inside a documented boundary)

A self-call emits as a plain same-file recursive call; no new emit
machinery. The boundary note: prod recursion depth is host-limited
(the JS stack), same class as the a04 int53 boundary — documented,
not solved. Compile-time tests prove logic terminates; they do not
budget prod stack.

## Implementation

- `parse.go`: `Decreases string` on `FnDecl`; `reDecreases`
  (`^decreases\s+(\w+)$`) in the fn-metadata switch; duplicate
  line is a parse error.
- `check.go`: `checkDecreases` (annotation validity, staleness,
  per-site `p - k` shape with site lines via match nodes);
  `checkLocalCycles` exempts a self-edge iff the fn carries a
  `decreases` line (validity owned by `checkDecreases`, so one
  mistake yields one error family).
  Amendment (a11): unit step, guard-carrying walk (`isGuardScrut`),
  `CAN3009`, plus `checkGlobalCycles` over all can edges with
  caller-file attribution; the verdict feeds the prove-first gate
  in both CLI and editor (`a11-recursion.md`).
- `lsp.go`: the `blocked` gate extends to the three new codes —
  no test executes while a termination proof is open.
- `eval.go`: entry check (`p < 0` fails loud) in `evLocalCall`;
  sandbox bound 64 → 1024, documented as resource bound.
  Amendment (a11): the entry check is deleted at both routes (root
  tests and local calls) — unreachable past the guard rule, and
  keeping it would contradict the returned-outcome theorem. The
  1024 depth bound stays as the hang backstop.
- `code.go`: `CAN3006`/`CAN3007`/`CAN3008` in the registry.
  Amendment (a11): plus `CAN3009` (unguarded recursion).

## Consequences (accepted before building)

- A new sketch, `docs/archive/sketches/retry-loop/`: retry-with-fuel against a
  flaky server. Tests recurse and stop (`now`, `later`, `never`,
  `empty`); minus the `decreases` line the same file is a cycle
  error — asserted by a committed test, so the "would hang without
  the proof" claim is checked, not narrated. Goldens (`retry.ts`,
  `errors.json`) committed with a golden test mirroring
  `TestGoldenAuthLogin`; normalize gains the four rows.
  Amendment (a11): the "would hang" claim was false and is
  corrected — the committed test now asserts proof-gated admission
  (`a11-recursion.md`). No golden changes: the shipped shape was
  already guarded and unit-stepped.
- `auth-login/` untouched (helpers stay the demo there; loops get
  their own program). `broken-login/` keeps exactly its titled
  squiggles (verified, not assumed).
- Tests: `lsp_test.go`-style diagnose cases for each new code
  (bad name, non-int param, stale annotation, non-decreasing
  sites named and positional, missing-line cycle, negative-entry
  loud failure, multi-hit script lists), plus the
  minus-`decreases` rejection of the shipped sketch.
- `TestCodesUnique`/`TestAllDiagsCoded` cover the registry;
  `TestGoldenJSONDiags` is untouched (no existing diagnostic
  changes shape).

## Open decisions (do not block)

- Mutual-recursion decrease arguments (leans: separate proposal;
  the ban message says so).
- `dec` decreases or multi-param lexicographic order (leans: no;
  int-only keeps the rule syntactic).
- Whether prod emit should trampoline deep recursion (leans: no;
  host boundary, documented like int53).
