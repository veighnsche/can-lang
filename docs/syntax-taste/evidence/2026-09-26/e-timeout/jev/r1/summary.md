# E-TIMEOUT R1 design consultation (Jev 3/3, 2026-09-27)

Model `jev-1.13.0` via `https://api.typesafe.ai/v1/systemone`.
Three requests with fully rewritten prose, verified structurally
equivalent (same state fields, questions, option keys) with zero
shared 8-grams pairwise. Requests/responses/audit saved in this
directory (`request-N.json`, `response-N.json`,
`response-N.metadata.json`, `wording-audit.json`, `consult.py`).

## Advice (agreement, not proof)

| Question | #1 | #2 | #3 |
|---|---|---|---|
| bound_mechanism | race_and_own 1.0 | race_and_own 1.0 | race_and_own 1.0 |
| bound_source | layered 0.99 | layered 0.87 | layered 0.85 |
| timeout_vocabulary | reuse_service_error 1.0 | reuse_service_error 0.99 | reuse_service_error 1.0 |

No choice-level disagreements. The softest cell is `bound_source`
#3 at 0.85 (simplicity pressure for a single ceiling vs honoring
caller intent); all three rounds still choose layered.

## Decisions taken (my rationale; Jev agreement is advice only)

1. **bound_mechanism: race_and_own.** Every native S3 await races its
   effective bound through the E04 cancel-absent `raceBoundary` with
   source `s3`: expiry returns the honest timeout outcome while the
   native stays owned until settlement, one escalation is filed, and
   late settlement is observed without rewriting the marker. Cleanup
   converges deferred through the already qualified destructive scrub
   (end then delete, with the delete-coupled abort where F3 qualifies
   it) once the hung native settles. Rationale beyond the advice: the
   natives admit no abort, so race_and_scrub's eager concurrent delete
   would pile unqualified native interplay (delete-vs-hung-end was
   never probed; the coupled abort holds only on the part-failure
   path) onto the worst path while doubling unsettled work. Deferred
   convergence reuses qualified machinery; the pin-until-settlement
   residual is documented honestly with live evidence.
2. **bound_source: layered.** Each per-await race bound is
   min(explicit `deadline_ms` remainder on write_stream, trailing
   runtime `boundMs`, ambient request-budget remainder, fixed adapter
   ceiling). Rationale: caller intent wins when stated (a 100ms caller
   must not wait out a full ceiling inside one hung await), the
   ceiling guarantees a bound in the common budgetless case since
   dispatch invents no request total, and the shape follows E04
   precedent (min-bound math, trailing-optional runtime inputs with
   checker-mandatory later). The ceiling constant is the only new knob.
3. **timeout_vocabulary: reuse_service_error.** Expiry lowers to
   `s3::service_error{code: "timeout", operation}` at every await
   site. write_stream already returns exactly this value for
   between-awaits expiry, every affected emits union already carries
   `service_error`, and the unknown-write marker plus escalation path
   preserves the honest uncertainty record — so no catalogue,
   checker, emitter, or example change is needed. The outcome never
   claims cancellation or rollback; nothing is cancelled because the
   natives cannot be.
