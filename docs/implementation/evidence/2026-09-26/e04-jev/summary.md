# E04 design consultation (Jev 3/3, 2026-09-26)

Model `jev-1.13.0` via `https://api.typesafe.ai/v1/systemone`.
Three requests with fully rewritten prose, verified structurally
equivalent (same questions/option keys) with zero identical prose
fields pairwise. Requests/responses saved in this directory.

## Advice (agreement, not proof)

| Question | #1 | #2 | #3 |
|---|---|---|---|
| propagation | ambient_scope 1.0 | ambient_scope 0.95 | ambient_scope 0.99 |
| sql_failure | reuse_query_failed 0.75 | reuse_query_failed 0.90 | reuse_query_failed 0.79 |
| stall_status | status_408 0.87 | status_408 0.69 | status_408 0.30 |

No choice-level disagreements. The weak 0.30 confidence on
`stall_status` #3 marks genuine ambiguity (minimal-change pressure
vs honest classification); investigated below.

## Decisions taken

1. **propagation: ambient_scope.** Dispatch installs one request
   scope per handler run (shared budget, abort signal, escalation
   collector); SQL/fetch adapters resolve it implicitly, so
   disconnect/shutdown expire each in-flight operation with its own
   unknown-write outcome. Outer-race-only would leave operations
   settling silently inside an abandoned handler, losing exactly
   the per-op unknown-write outcomes the R04 contract requires.
2. **sql_failure: reuse_query_failed.** Boundary expiry maps to the
   already emitted `sql::query_failed{operation, code}` with
   `code: "budget"`. Open code/phase strings are the established
   extension point (transport phases, query codes); a new catalogue
   failure would ripple through checker unions and every example
   for zero semantic gain. The adapter lowers the shared
   unknown-write marker into this domain value; a later slice can
   still promote the code to a dedicated failure in one function.
3. **stall_status: status_408.** A body read outlasting its stall
   bound answers fixed 408 Request Timeout; read errors stay 400,
   over-limit stays 413. 400-for-stall misclassifies (clients do
   not retry 400) while 408 names the event honestly. The union
   change is contained: `SnapshotResult` has exactly one consumer
   (`server.ts`), and existing status assertions are unaffected.

Not consulted: the escalation observation channel is already
determined by lane boundaries — `owner-core.ts` is out of scope
(so no new owner-diagnostic phase) and E06 owns reporting (so E04
collects records for E06 to drain rather than printing).
