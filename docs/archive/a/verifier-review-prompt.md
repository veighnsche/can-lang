# Prompt: adversarial design review of the verifier core (§2.2)

Paste this into the reviewer with
[docs/archive/a/a67-testing-contracts-design.md](a67-testing-contracts-design.md)
(§2.2, §2.5, §2.6, Part 3),
[docs/archive/a/a69-contract-grammar.md](a69-contract-grammar.md),
[docs/archive/a/a77-revision-identity.md](a77-revision-identity.md),
[docs/archive/a/a68-linked-pure.md](a68-linked-pure.md), and
[docs/archive/a/a13-stdlib.md](a13-stdlib.md) (Order 0) attached.

Nothing here is implemented — this review attacks the design
before any verifier slice lands. a68 (linked runner), a69
(contract grammar), the a73–a76 variant epic, and a77 (identity
enforcement) are shipped; do not re-litigate them except where
the verifier depends on them. a67 §2.6 parks refinements,
ghosts, and lexicographic decreases — parked stays parked; do
not propose them as fixes.

---

Review the first-cut verification logic: universal proofs of
`requires`/`ensures` over inputs satisfying `requires`, with no
quantifier syntax, over Boolean constants, sort-correct
equality, record projections, linear integer
expressions/comparisons, conjoined clauses, and exhaustive
Boolean case expressions. Unsupported predicates or obligations
are rejected; inconclusive verification is not acceptance. For
each numbered position return: CONFIRM, or REVISE with a
concrete counterexample (an exact contract/body pair that
proves wrongly, rejects wrongly, or slips through), citing file
and section. No code exists; file:line evidence means the
attached docs.

Positions:

1. Proof domain boundary. The first cut admits bool consts,
   sort-correct equality, record projections, linear integer
   exprs/comparisons, exhaustive bool matches, conjunction.
   Define the boundary precisely enough to implement: what is
   "sort-correct" for equality over brands, Seqs, Bytes, dec,
   and variant carriers? What is "linear" (division?
   mixed int/dec? string ops? comparisons between
   non-integers)? What projection depth and what record
   nesting? Anything outside is rejected — state the
   rejection rule and whether one unsupported clause poisons
   the whole contract or just itself.
2. Disjunction spelling. The borrow sketch's Boolean `match`
   is logical OR. What exhaustiveness does the prover demand
   of that match (both arms present? arms in the contract
   limited to bool literals, or nested matches allowed)? May
   `requires` contain a match, or only `ensures` arms? Give
   the membership clause (`result is one of left/right`) as
   the worked example and break it: construct a body the
   clause shape accepts but should not, or rejects but should
   not.
3. Outcome coverage. The validator pilot specifies all three
   outcomes with an input partition; the parent excerpt's
   two-outcome contract admits `-1` in `[0,1]` satisfying
   neither. Does the verifier require the outcomes to cover
   the input space, or does it prove each arm independently
   (letting incomplete contracts verify vacuously)? If
   coverage is required, state the decision procedure for
   "every input satisfying requires lands in exactly one
   outcome's domain" within the first-cut domain — or admit
   it is undecided and say what weaker check holds instead.
4. Callee reasoning. The verifier reasons from verified
   callee contracts, never bodies; unsupported body
   operations or unverified callees are never silently
   assumed facts. Define "verified callee" operationally
   given a77 stores no proof certificates (reverify the
   selected program each run? cached derivations with
   validity chains?). Then attack: recursive calls (induction
   or rejection?), mutually recursive contracts, and
   externs — a foreign body the compiler never sees. Is an
   extern's contract an axiom? If yes, construct the trust
   hole; if no, state what an extern call needs before its
   contract may be used.
5. Call-site obligations. "The verifier establishes call
   preconditions": each call site discharges the callee's
   `requires` from caller context. State exactly which facts
   the caller context contains: caller `requires`, match-arm
   narrowing (is `x <= 0` available inside the true arm of
   `x <= 0`?), binder projections, prior call outcomes?
   Construct a call that is safe only with path-sensitive
   facts and say whether the first cut proves or rejects it.
6. Failure-injection boundary. `given` rows are not universal
   assumptions; a proven contract does not invalidate an
   existing injection scenario, nor does injection become
   reachability evidence. Make this operational: a contract
   proves the provider never emits `E` on some input, while a
   committed `given` row injects `E` there and the decision
   table passes. What does each artifact mean after the
   proof lands, what (if anything) is reported, and what
   breaks if a later reader treats the passing row as
   "E is reachable"?
7. Certificates and reruns. a77 models invalidation without
   a store. Does the first verifier slice persist
   certificates or reverify the contracted program on every
   run? State the scaling argument for the choice, what
   "proof" output (if any) a passing compile produces, and
   how a stale proof is distinguished from a fresh one
   without re-running the prover.
8. Contract diagnostics. "Unsupported" vs "unproven" are
   different failures (cannot express vs cannot establish)
   and inconclusive is not acceptance. Propose the
   diagnostic split: which findings are errors, what each
   names (predicate? obligation? body operation?), and what
   the legal fix is for each. No silent acceptance paths.
9. Pilot adequacy. The max and int-range-validator pilots
   (the latter against the existing
   `std__validate__int_range`) are the acceptance set.
   Argue they exercise every first-cut feature at least
   once, and name what real contract shape would still be
   untested after both pass — or concede a third pilot and
   specify it.
10. Sequencing with stdlib. a77 enforcement is active;
    verifier implementation may now start. Can stdlib ship
    contract-bearing interfaces the first-cut verifier
    cannot yet prove (unsupported domain), with tests and
    identity as the only evidence? If yes, state the
    labeling that keeps such interfaces from silently
    becoming "verified" later; if no, state what blocks
    them and what unblocks.
