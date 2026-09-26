# E05 design consultation (Jev 3/3, 2026-09-26)

Model `jev-1.13.0` via `https://api.typesafe.ai/v1/systemone`.
Three requests with fully rewritten prose, verified structurally
equivalent (same questions/option keys) with zero identical prose
fields pairwise (`/tmp/e05-jev-audit.py`, 10 fields, 0 identical
pairs). Requests/responses saved in this directory. Total usage:
4660 input + 387 output tokens. The O1-vs-O2 verdict itself was not
consulted: it is measurement-driven (H0-H12), not a taste judgment.

## Advice (agreement, not proof)

| Question | #1 | #2 | #3 |
|---|---|---|---|
| loser_observation | owner_observed 1.0 | owner_observed 1.0 | owner_observed 0.99 |
| terminal_precedence | unknown_dominates 0.86 | unknown_dominates 0.86 | unknown_dominates 0.82 |
| hedge_effects | idempotent_writes 0.83 | idempotent_writes 0.72 | idempotent_writes 0.68 |

No choice-level disagreements. The weakening 0.83 → 0.68
confidence on `hedge_effects` marks genuine ambiguity (usefulness
of hedged writes vs per-site idempotency proof burden);
investigated below.

## Decisions taken

1. **loser_observation: owner_observed.** The hedge owner observes
   loser fates via `fates()`/`settledAll`; the shared sink records
   only genuine boundary expiries (escalation + seq-linked late
   records). A post-win loser rejection that never expired a
   boundary (H6) leaves no sink record. O2 supervised-loser
   machinery is not built: every H-leg need is met without it, and
   the unanimous 1.0 advice agrees the owner account is complete.
2. **terminal_precedence: unknown_dominates.** When no replica
   settles and fates mix expiry with rejection, the hedge reports
   the unknown-write outcome (H0d pins this: the rejection lands
   first, the report is still unknown). A known failure can never
   resolve another replica's commit uncertainty; the unknown marker
   plus supervisor escalation therefore outranks the failure.
3. **hedge_effects: idempotent_writes, with mandatory per-shape
   qualification.** Replicas are reads, or writes proven
   idempotent for their exact shape with reread reconciliation.
   The advice is taken as the contract direction, but no E05 leg
   hedges a write — all measured replicas are reads — so zero
   hedged-write shapes are qualified. Like caller operands without
   a verified native meaning, no hedged write ships without its
   own qualification (idempotency proof plus H4/H5-style effects
   accounting). The harness comment records this; E09/F own any
   first write shape.

Not consulted: the fetch abort policy (H4 vs H5 measures it
directly — abort bounds, let-settle drain-blocks — leaving no
judgment call), and the O1-sufficient verdict itself (all W5
dimensions measured green; see `../e05-x-r04-2.md`).
