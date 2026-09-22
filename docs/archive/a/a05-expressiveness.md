# v0.5 — Expressiveness gap (plan)

Status: landed. Direction endorsed by the six-judge clean-room panel
(unanimous 7/10: superb to verify, still a subset to write in) and
accepted by the project owner. Spec before build, one item at a time —
kept: items 1–6 and 8 shipped with proofs (a06–a09), item 7 closed
with no build (no real program was blocked). The plan below is the
baseline the future panel re-rates against.

## The gap

The panel's agreed weakness (6/6): v0 cannot write real programs. No
loops, no mutation, no arithmetic, no async. Bodies are single
expressions, calls work only as match scrutinees, bool-returning calls
are unscriptable, `seal` takes literals only, externs are module-local.
The flagship carries the evidence visibly: the retry arms duplicate
the password check twice (open question 3, NOTE in `auth.can`), and
there is no way to say "try three times" without copying the arm a
third time.

## Rule of the road

Every expressive power has a proof cost, and the costs sequence the
work. Nothing lands without the check that keeps it honest; a feature
whose proof is "future work" is a bug with a roadmap. Order below is
cheapest proof first, each item funding the next.

## 1 — Arithmetic (cheapest)

`+`, `-`, `*` (division waits: exactness of `/` on `dec` needs its own
rule) over `int`/`dec`, no mixed operands, no precedence table worth
naming. Proof cost: near zero. Exactness is already solved in proofs
(`big.Rat` under `dec`, machine `int` under `int`); the TS emit
boundary (exact to 15 digits) is already documented in a04. Small,
self-contained, breaks nothing. Unblocks nothing else either — it is
warm-up with a real payoff, not load-bearing.

## 2 — Helper extraction (medium)

Open question 3, and the flagship's own NOTE. Intra-module calls are
banned today ("v0 has no intra-module calls"), so shared logic is
copied per arm. Pure internal calls need no `given` stubs — they are
deterministic, so there is nothing to script — but the call machinery
(call resolution, `uses` pins, exhaustiveness, coverage, `UsesHere`
imports) assumes foreignness everywhere and must learn a second kind
of callee. Proof cost: a callee-kind distinction plus totality of
argument passing (arity and types already check). This one visibly
shrinks the flagship, which makes it the best demo of progress.

## 3 — Termination + iteration, as a pair (largest)

Correction to an earlier note, which called termination "scaffolding
with nothing to hold up." Wrong framing: termination discipline is the
admission ticket for loops. R7 executes every test at compile time, so
the day iteration lands, a non-terminating test hangs the build.
Termination is not a standalone feature; it is the first half of the
loops feature. They spec together or not at all: a fuel bound, a
structural-decrease rule, or total combinators only — the mechanism is
open, the pairing is not. Whatever iterates must provably halt before
anything runs or emits, same gate as exhaustiveness.

## Explicitly later

- Bool-returning calls, cross-module externs, non-literal `seal`,
  int-backed brands: polish on existing machinery, each a small
  proposal when its absence blocks a real program.
- Mutation and state: needs the effect story extended (R6 purity plus
  `call`/outcome handling is the seed, not the system).
- Async: distant. Nothing in v0 wants it yet.

## Consequences (accepted by writing this down)

- No v0.5 build starts without its own spec doc in this series. The
  panel rated the direction, not a design; ratings are not specs.
- The flagship stays the demo vehicle: arithmetic must earn its place
  in a decision table, helpers must visibly shrink `auth.can`, loops
  must arrive with a program that would hang without the proof.
  Amendment (a11): the "would hang" requirement produced a false
  demonstration — the retry trace is finite either way. The standing
  rule is proof-gated admission, and the committed test asserts
  exactly that (`a11-recursion.md`).
- The unanimous-7 panel report is the baseline: a future panel
  re-rates against this plan, and the score moves only on shipped
  expressiveness with proofs attached.

## Open decisions (do not block)

- Whether division ships with arithmetic or waits (leans: waits; `dec`
  division exactness deserves its own rule, not a footnote).
- Which termination mechanism (leans: undecided until the loops shape
  is sketched; the pair specs together).
- Whether helpers need `uses` pins for same-module callees (leans:
  no; pins are for cross-module edges, locality is visible).
