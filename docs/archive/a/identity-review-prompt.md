# Prompt: adversarial design review of §2.3 revision identity

Paste this into the reviewer with [docs/archive/a/a67-testing-contracts-design.md](a67-testing-contracts-design.md)
(§2.3 especially), [docs/archive/a/a72-variant-design.md](a72-variant-design.md)
(decision 4), [docs/archive/a/a69-contract-grammar.md](a69-contract-grammar.md),
[docs/archive/a/a73-variant-registry.md](a73-variant-registry.md), and
[docs/archive/a/a13-stdlib.md](a13-stdlib.md) (Order 0) attached.

Nothing here is implemented — this review attacks the design before
any slice lands. a68 (linked runner), a69 (contract grammar), and the
a73–a76 variant epic are shipped; do not re-litigate them except
where §2.3 depends on them.

---

Review the §2.3 design: one revision-identity mechanism covering
both function contracts and variant case sets, specified before
verifier implementation begins. For each numbered position return:
CONFIRM, or REVISE with a concrete counterexample (an exact
program shape or workflow where the position fails, silently
passes drift, or contradicts an attached doc), citing file and
section. No code exists; file:line evidence means the attached
docs.

Positions:

1. Fingerprint contents. Contracts: canonicalized interface +
   `emits` + `requires` + outcome-indexed `ensures` + referenced
   record/error shapes; positions, comments, and test rows
   excluded. Variants: qualified identity, resolved case names,
   named payload fields/types, depended-upon shapes. Is anything
   load-bearing missing (brand seals? cell effects? `given`
   tables?), and is anything included that can change without
   changing meaning?
2. Canonicalization. "Reject every normalized change at the same
   revision; no semantic compatibility inference." Enumerate what
   normalization must cover for the exclusion list (positions,
   comments, test rows) to be sound — and construct a change the
   normalizer would plausibly erase that alters checking, proof,
   or emit behavior.
3. Identity anchoring. Contracts belong to the function revision
   (no floating contract revision); cases inherit the parent
   revision (no floating case revisions). Attack the inheritance:
   provider bumps parent rev with an unchanged case set — silent
   for pinned consumers? Provider reorders cases or renames a
   payload field at a bumped rev — who re-reviews, and what
   forces it?
4. Baseline sourcing. Baselines are explicitly supplied and
   previously accepted (review base), never silently regenerated;
   a self-compared lockfile proves nothing. Who supplies the
   baseline in the actual workflow, and what stops a developer
   from blessing a drifted baseline to silence the diagnostic?
   Is there a workflow where no review base exists (new module,
   new function) and the mechanism must bootstrap without
   becoming self-certifying?
5. Enforcement strictness. Same revision + changed fingerprint =
   reject, even if apparently compatible or equivalent. Construct
   the strongest "obviously safe" change this rejects (comment
   reflow excluded already — find a semantic one, e.g. widening
   an error payload with an optional field, reordering union
   cases) and argue whether the rejection is load-bearing or
   gratuitous. If gratuitous, say what narrower rule still
   closes the drift the mechanism exists for.
6. Changed revision. Callers stay pinned, no automatic
   retargeting; unavailable old revisions keep failing. Trace
   the cross-module staleness walk: consumer pinned to rev 1,
   provider ships rev 2 with an identical fingerprint — does
   anything force the consumer to move, and should it? Provider
   ships rev 2 with a changed fingerprint while the consumer
   still pins rev 1 — which diagnostic fires, where, and does
   it name the changed case/field or just the rev?
7. Removal and freshness. Removal is a contract change, not an
   unchecked default; body or dependency-contract edits
   invalidate proof certificates with the contract unchanged.
   How far does invalidation propagate transitively, and what
   stops a removal from hiding as a rename (drop + add)?
8. Diagnostic. New symbolic code pointing at the changed
   case/field (or the declaration on removal). Given the a71
   explain-payload convention, what must the payload carry so
   the diagnostic is actionable without reading the design doc?
9. Probe separation. Added-case-with-incomplete-match (proof,
   a75) vs same-revision-addition-with-complete-matches
   (identity) stay separate probes; a version-bump test is not
   a substitute for the identity probe. Design the identity
   probe's exact setup: which module changes, which stays
   pinned, what the runner executes, and what must fail.
10. Sequencing. The claim: §2.3 specified (not implemented)
    unblocks verifier implementation, and stdlib rows may
    proceed on a specified-but-unenforced §2.3. Falsify the
    second half if you can: construct stdlib work that silently
    rots under specification-without-enforcement.
