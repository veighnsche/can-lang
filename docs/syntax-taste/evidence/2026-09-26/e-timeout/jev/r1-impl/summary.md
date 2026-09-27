# E-TIMEOUT R1 implementation consultation (Jev 3/3, 2026-09-27)

Model `jev-1.13.0` via `https://api.typesafe.ai/v1/systemone`.
Settled R1 mechanism (race_and_own, layered, reuse_service_error) is
fixed context, not relitigated: only the upload-box-after-expiry and
discard-race implementation points are decided here. Three requests
with fully rewritten prose, verified structurally equivalent (same
state fields, questions, option keys) with zero shared 8-grams
pairwise. Requests/responses/audit saved in this directory
(`request-N.json`, `response-N.json`,
`response-N.metadata.json`, `wording-audit.json`, `consult.py`).

## Advice (agreement, not proof)

| Question | #1 | #2 | #3 |
|---|---|---|---|
| upload_expiry | poison_discard 0.83 | poison_discard 0.87 | poison_discard 0.91 |
| discard_race | unraced_cleanup 0.66 | race_like_others 0.51 | unraced_cleanup 0.59 |

One disagreement: `discard_race` splits 2-1 for
unraced_cleanup, with round #2 a coin flip (0.51, confidence 0.02)
and rounds #1/#3 weak (confidences 0.31/0.17). Investigated below.

## Disagreement investigation: discard_race

Verified by reading, not assumed:

- One `retire()` serves both paths: the scope-drain close callback
  (`s3.ts` upload close) and explicit `discardUpload` call the same
  function. Racing explicit discard forks retire into two behaviors
  with different observability; leaving it unraced keeps one retire.
- Drain-time retire cannot race under any option: the close callback
  returns a `Completion` that drain surfaces only as
  `cleanupFailed` — there is no timeout channel back to a caller.
- E08 D4 qualifies the synchronous timeout-path scrub wire shape
  (complete-then-delete with asserted op triples). Racing discard
  preserves none of that meaning: the timeout answer displaces the
  scrub outcome, a second discard during a pending scrub needs a
  pending-state answer outside the current vocabulary, and the scrub
  verdict lands nowhere readable.
- Coherence with the settled mechanism: deferred convergence ("when
  the hung native later settles, the adapter runs the standard
  destructive scrub") already runs scrubs to completion unraced in
  the background. Racing explicit discard but not deferred scrubs
  would race the same scrub in one call path and not another.

Deciding evidence for the majority: race_like_others buys a bound on
explicit cleanup at the cost of unobserved convergence plus a new
pending-state vocabulary plus single-flight join semantics — new
machinery of exactly the kind the main decision rejected — while
unraced_cleanup narrows the surviving indefinite wait to deliberate
cleanup against a wedged service (data paths stay fully bounded) and
keeps every scrub outcome observable. The 0.51 round-#2 vote is
treated as the coin flip it is; mechanism plus the majority agree.

## Decisions taken (my rationale; Jev agreement is advice only)

1. **upload_expiry: poison_discard.** On upload_write/upload_finish
   expiry the box transitions to discarded immediately (sink
   retained locally for cleanup), the standard destructive scrub
   (end then delete) attaches deferred to the hung native
   settlement, and all further caller ops on the handle answer
   `upload_closed` with state discarded — the exact E08 D3 shape.
   Rationale beyond the advice: retry-while-hung is unqualified
   (concurrent sink.write never probed), finish/discard during the
   hang would wait on the same stuck state E07 H2 measured, and a
   finish after a timed-out write would ratify bytes the timeout
   honestly called unknown. Cost accepted: a recovered handle is
   lost (caller reopens) and the deferred scrub may delete an
   intact write — destructive convergence consistent with a remedy
   that already destroys partial work. Deferred scrub is
   single-flight per box (concurrent same-box expiries attach
   once); scrub failures stay best-effort like the existing
   pump-finally scrub.
2. **discard_race: unraced_cleanup.** Explicit `discardUpload` and
   scope-drain retire share one unraced scrub to completion and
   always report the honest converged outcome. The surviving
   indefinite wait narrows to explicit cleanup against a wedged
   service, documented as a residual alongside the
   pin-until-settlement residual. Note the interaction: after an
   upload expiry poisons the box, a caller discard short-circuits
   (the non-open guard answers upload_closed before retire runs)
   promptly with no wire calls — only a discard whose own sink
   awaits hang first can still block.
