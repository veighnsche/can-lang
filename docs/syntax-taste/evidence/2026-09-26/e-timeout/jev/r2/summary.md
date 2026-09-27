# E-TIMEOUT R2 design consultation (Jev 3/3, 2026-09-27)

Model `jev-1.13.0` via `https://api.typesafe.ai/v1/systemone`.
Three requests with fully rewritten prose, verified structurally
equivalent (same state fields, questions, option keys) with zero
shared 8-grams pairwise. Requests/responses/audit saved in this
directory (`request-N.json`, `response-N.json`,
`response-N.metadata.json`, `wording-audit.json`, `consult.py`).

## Advice (agreement, not proof)

| Question | #1 | #2 | #3 |
|---|---|---|---|
| peer_release | respond_then_drain 1.0 | respond_then_drain 1.0 | respond_then_drain 1.0 |
| background_drain | observed_reported 0.99 | observed_reported 0.97 | observed_reported 0.98 |
| revocation_timing | revoke_after_drain 0.69 (conf 0.38) | revoke_after_drain 0.97 (conf 0.94) | revoke_at_response 0.57 (conf 0.13) |

One disagreement: `revocation_timing` splits 2-1 for
revoke_after_drain, with rounds #1 and #3 near-uncertain (0.38/0.13
confidence) and only #2 confident (0.94). Investigated below.

## Disagreement investigation: revocation_timing

Both options work mechanically — verified by reading, not assumed:

- Correlation survives either way: `serveOuter` reports with
  `requestContext(native)` off the native `Request`, never off the
  snapshot token, so `requestNativeRequest` falling back on a revoked
  token changes nothing.
- No scoped-resource close callback consults the snapshot maps:
  stream-reader closes run `releaseReader` (direct `reader.cancel` +
  `releaseLock`), and `dispatch` already runs `abandonRequest` in its
  own finally before handler completion propagates.
- `rejected()` in `router.ts` revokes immediately, but only for routes
  that never entered handler work ("no per-request child can hold the
  token") — different from a drain that still closes scoped readers.

Deciding evidence for the majority: revoke_at_response inverts the
close-before-revoke sequence on every request kind (including
upgrade/stream paths whose future close logic may consult the
snapshot) for a tightness gain nothing can observe — the handler
already returned, so no user code can reach the token during the
background drain. revoke_after_drain preserves the invariant bit for
bit with a precisely characterized unreachable window, and moves the
existing finally block verbatim into the drain continuation. The 0.13
round-#3 vote is treated as the coin flip it is; mechanism plus the
confident round agree.

## Decisions taken (my rationale; Jev agreement is advice only)

1. **peer_release: respond_then_drain.** The handler `Response` is
   published via a side channel at handler completion; the peer is
   released immediately while the request owner scope drains owned in
   the background to the same settled end state. The peer gets the
   honest `unknown:commit` body at the bound; `stop`/`wait` keep
   bounding shutdown via `shutdownMs` with overdue closes owned for
   the supervisor. Requests without pending owned work are unchanged
   (instant drains stay instant). Ownership, `commit_unknown`,
   escalation, and late-settlement semantics are untouched — only
   response timing decouples from drain timing. The gated alternative
   would discard the reconcilable transaction id exactly on the path
   the requirement exists to fix.
2. **background_drain: observed_reported.** Background drain failures
   flow to the redacted request reporter exactly as `serveOuter` does
   for inline drains today (one correlated record; clean drains and
   ordinary late settlements stay silent). Supervisors see failure for
   failure what they see now; only the delivery moment shifts behind
   the response.
3. **revocation_timing: revoke_after_drain.** `revokeRequest` stays
   post-drainage; the documented post-response token window is
   unreachable by user code. See the investigation above.
