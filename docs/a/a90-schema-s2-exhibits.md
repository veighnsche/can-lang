# a90: S2 admission — duplication exhibits and verdict

Status: decided 2026-09-18. Exhibits below are committed code;
the verdict admits closed-descriptor *design* under the bounds
in §5. No descriptor code in this step.
Parent: `docs/a/a89-schema-validation-design.md` (S2 gate);
consumers: `std/quota/quota.can` (S1a pilot),
`sketches/reserve/reserve.can` (S2 second consumer).

## The gate

a89 admits S2 only when "a second independent concrete
consumer shows repeated binding/interpretation work that
shared scalar calls do not remove", with proof of closed
field bindings, correct field types, complete rule dispatch,
deterministic ordering, termination/domain arguments, and
schema/record-mismatch rejection before implementation.
It also warns: "even then, specialized calls may stay
cheaper."

## What sharing removed (not exhibits)

Consumer 2 contains zero traversal or decision logic: the
length scan, range check, membership scan, and canonical
integer rendering all execute inside the shared bodies
(covered linked-pure across three files in both module
orders, plus node parity). No producer logic was copied.
The exhibits below are strictly the binding and
interpretation layer around those calls.

## Exhibits (all committed, all green)

1. **Per-field binding.** Each consumer rewrites the
   field→validator→bounds mapping from scratch:
   `str_length(guest, min, max)` /
   `int_range(party, lo, hi)` / `one_of(slot, allowed)`
   in `reserve__request__validate` mirrors
   `str_length(label, …)` / `int_range(amount, …)` /
   `one_of(mode, …)` in `quota__request__validate`.
   Shared calls supply the validators; nothing supplies
   the binding. A wrong-field binding (e.g. length-checking
   `slot` against guest bounds) is well-typed `.can` today
   and caught only by decision-table evidence.
2. **Fail-fast ordering.** Both bodies encode the same
   priority as a nested match chain
   (minimum-nonnegative → length → range → membership),
   each with its own priority rows
   (`label_beats_policy` / `guest_beats_policy`, …).
   The order lives in program text per consumer; no
   shared artifact owns it.
3. **Payload reconstruction.** Both translate producer
   outcomes into the frozen triple arm by arm: path
   strings, rule tokens, and value sourcing (original
   string passthrough, canonical int render,
   `minimum=<lo>;maximum=<hi>` /
   `lower=<lo>;upper=<hi>` formats). The formats and
   the `_`-binder discipline (S1a amendment 2) repeat
   verbatim; only field names and bounds arguments
   differ.
4. **Per-consumer scripting.** Consumer 2 scripts all
   four cross-file call sites: 58 `exchange` entries
   against quota's 12 (quota's validators are
   same-file, so only its converter scripts). Every
   future consumer pays this per-row scripting cost
   once per call site it reaches.
5. **Token vocabulary split.** Data-constraint tokens
   (`str.length_scalars`, `int.closed_range`,
   `str.one_of`) repeat identically across consumers;
   policy-constraint tokens (`schema.label_bounds` vs
   `schema.guest_bounds`, …) are per-schema-record by
   construction. The reusable half of the vocabulary
   is now exhibited twice.

## Verdict: ADMIT, bounded

The gate's showing is met: exhibits 1–3 are
binding/interpretation work, present in two independent
programs, untouched by sharing producers. The "may stay
cheaper" caution is respected by bounding S2 to what the
exhibits prove — no further.

## Bounds on the S2 descriptor slice (design first)

- Exactly the three exhibited constraint kinds (scalar
  length, closed int range, str membership) over exactly
  the two exhibited field types (`str`, `int`). No new
  kinds, no `dec`/`bool`, no nested descriptors beyond
  the S1b envelope pattern.
- Descriptors as ordinary records (no `Seq<Variant>`
  fields — still deferred per `checkDeclFields`
  Decision 6); interpretation as specialized closed
  dispatch, not callbacks (no Functions dependency).
- The a89 proof list stays the design's acceptance bar:
  closed bindings, correct field types, complete rule
  dispatch, deterministic ordering, termination/domain
  arguments, mismatch rejection — plus a third-consumer
  cost comparison before implementation, so the
  machinery never exceeds the specialized-call cost it
  replaces.
