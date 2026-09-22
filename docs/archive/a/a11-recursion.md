# v1.1 — Global recursion policy + guarded steps (spec)

Status: shipped. Amendment to a07 (helper termination) and a08
(termination + iteration): the recursion ban now spans the whole
program, the admitted loop step is one canonical shape, and the
theorem promises a returned outcome instead of a loud fault. No new
source spelling; the shipped retry already has the admitted shape.

## The hole (why this exists)

The a07 cycle check covered same-file edges only, and foreign calls
never execute in the sandbox. Two modules with properly pinned
imports could therefore call each other, pass every test on stubs,
and link into an unproved recursive cycle in production — verified
against the compiler during this amendment (alpha/beta probe: both
tests PASS, emit is infinite mutual recursion). The sandbox
termination argument was never wrong; the whole-program claim was
unproved.

Separately, a08's theorem relied on negative entry failing loud:
the proof showed chains cannot run forever, not that every accepted
invocation returns a declared outcome. And the retry file claimed
its finite fuel trace would "hang the build" without the decreases
line — false. Deleting metadata changes admission, not the trace.

## Rule

The can call graph — every can-to-can edge, same-file and
cross-file — must be acyclic except proven self-edges. A self-call
is admitted iff its function declares `decreases p` over an `int`
param and every self-call site has both properties:

1. Unit step: the site passes exactly `p - 1` (named or positional).
   Larger steps terminate but break one spelling per meaning and are
   refused (`CAN3008`).
2. Positive branch: the site sits under the false arm of the
   canonical `p <= 0` guard (`CAN3009`). `p >= 1`, `p > 0`, `p == 0`
   spell the same bound; only this shape admits recursion.

The param never rebinds, so a site reached under a false guard
entered with `p >= 1` and stepped to `p - 1 >= 0`. Negative entries
take the base arm and return a declared outcome — no loud fault, no
fault-bounded theorem. Sites in the true arm would diverge (each
entry steps further negative) and are refused, guard or no guard.

Externs are host code outside the proof and never form cycle edges;
unknown callees belong to other codes. A cycle touching two or more
files reports once, at the call site that closes it, in the caller's
file (`CAN3005`, the same rule as the local ban). Same-file cycles
stay with the local check: one mistake, one diagnostic.

## Theorem (written down, not waved at)

Every admitted invocation of a `decreases` function returns a
declared `Ok` or `emits` outcome. Chains walk `p, p - 1, …, 0` into
the base arm; negative entries take the base arm immediately. The
sandbox depth bound stays as a resource bound, loud on breach —
same class as before, still not the proof.

## Implementation

- `check.go`: `isDecrease` requires exactly `1`; `checkDecreases`
  walks match arms carrying the guard flag (`isGuardScrut`
  recognizes only `p <= 0`); unguarded unit sites are `CAN3009`.
- `check.go`: `checkGlobalCycles` builds the whole-program can-edge
  graph (sorted for determinism), skips externs and unknown
  callees, skips same-file-only cycles (local check owns them), and
  attributes each reported cycle to its closing call site.
- `lsp.go` / `main.go`: the global verdict feeds the existing
  prove-first gate (`extBlocked` into `checkSem`, `CAN3009` joins
  the blocking codes), per the R10 world-error rule — reported
  per-line, suppressing only execution-dependent checks, in both
  the CLI and the editor.
- `eval.go`: both negative-entry failures (root tests and local
  calls) are deleted — unreachable past the guard rule, and keeping
  them would contradict the returned-outcome theorem.
- `code.go`: `CAN3009` in the registry.

## Consequences

- The alpha/beta probe is refused with
  `call cycle alpha__run -> beta__run -> alpha__run` and no test
  executes. Same-file behavior is unchanged (same messages, same
  codes); `retry.can`, `auth.can`, and `counter.can` compile
  byte-identically — no golden changes.
- Breaks (stated plainly): larger-step recursion (`p - 2` and
  friends), unguarded self-calls, and cross-file cycles are now
  errors. `TestLoopNegativeEntryLoud` is inverted: negative entries
  take the base arm. The `CAN3008` message now names the unit step.
- The retry header, the reviewer guide's loop section, and the a08
  "would hang" passages are corrected to proof-gated admission;
  each carries an `Amendment (a11)` pointer here. The a05 plan
  baseline is left as history.

## Open decisions (do not block)

- Multi-param lexicographic and annotated mutual-recursion
  arguments stay future proposals (a08's stance, unchanged).
- `dec` decreases stay out (ints only; the rule stays total).
- Whether prod emit should trampoline deep recursion stays a
  documented host boundary (a08's stance, unchanged).
