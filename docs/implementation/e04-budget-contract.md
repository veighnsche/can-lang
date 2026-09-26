# E04 budget parameter/failure contract (handoff to E09, C06, F data)

2026-09-26. E04 wires the runtime side of operation budgets; authored
operand surface (checker/emitter) is C-owned. This doc is the
parameter/failure contract E09 accepts against in W5, the native
adapter surface F data and C06 reuse, and the checker-need list for
the coordinator.

## Runtime inputs (wired, trailing-optional)

Trailing-optional so current emitted calls keep their shape; the
checker makes caller bounds mandatory (C-owned).

- SQL (`SQLBounds`, `runtime/platform/sql/pool.ts`): trailing after
  context on every pool/tx query/execute plus `withTransaction`.
  `{ boundMs?: number; budget?: RequestBudget }`. No cancel field
  exists — `raceBoundary` throws `TypeError` if one is ever passed
  with `source: "sql"` (X-R04-1, structural).
- Server HTTP client (`NativeRequest.budget`, `runtime/transport/fetch.ts`):
  explicit wins over ambient; effective `min(connection timeout,
  remaining)`, floored to whole ms. Ambient scope signal links to
  the timeout path (`Deadline.expireOn`); owner cancel still reports
  `cancelled`.
- Browser action fetch (`ActionFetchInput`, `runtime/platform/action-json.ts`):
  `timeoutMs?`, `budget?`, `signal?` (caller cancel — qualified
  native abort). Static site bound `JsonFetchSite.timeoutMs?`
  (compiler-spliced like `limit`, validated, fails closed).
- Ambient scope (`runtime/transport/request-scope.ts`): dispatch
  installs one `RequestScope` per handler run (unbounded until
  expired — no invented default total). Adapters resolve
  `{budget, scopeSignal, sink}` via `scopeRaceInput(explicit?)`.
  Server-only (`node:async_hooks`); browser clients take explicit
  inputs only.

## Failure mapping (all inside existing emits — no catalogue change)

| Expiry | Domain value | Notes |
|---|---|---|
| SQL query/execute at bound | `sql::query_failed{operation, code: "budget"}` | Never claims rollback; reread to reconcile a write. |
| `withTransaction` never started (effective 0) | `sql::transaction_failed{phase: "budget"}` | Nothing ran: phase, not commit uncertainty. |
| `withTransaction` started, unsettled | `sql::commit_unknown{transaction_id}` | Names the attempt; never retry (may have committed). |
| Server HTTP client at bound | `http::timeout{timeout_ms}` | Names the bound that fired (capped), not always the connection timeout. |
| Action fetch at bound | `http::transport_failed{phase: "timeout"}` | No commit knowledge; reread. Distinct from `cancelled`. |
| Action caller cancel | `http::transport_failed{phase: "cancelled"}` | Unchanged meaning. |
| Ingress body abandoned (stall/Bun bound/scope expiry) | Fixed `408 Request Timeout` | Bun-side aborts (`AbortError`) map here; corrupt reads stay 400, over-limit 413. |
| Signal-path drain past `shutdownMs` | `http::shutdown_failed{phase: "deadline"}` | Close stays owned; supervisor SIGKILL bounds the rest. |

Emits coverage verified: every `sql::query_*`/`execute` emits
`query_failed`; `with_transaction` emits `transaction_failed` and
`commit_unknown`; action/`fetch_json` ops emit `transport_failed`;
the HTTP client failure bound carries `http::timeout`. Nothing to
merge into `catalogue.json`.

## Honesty rules (E09 legs assert these)

- A budget outcome never claims rollback or non-start (except the
  never-started transaction phase, which ran nothing by construction).
- `commit_unknown` is never retried; reconciliation rereads.
- Leases are never timer-revoked: the native op holds its lease
  until settlement; late settlement is observed, never rewritten
  into the returned marker.
- Markers are frozen; escalation records carry no native messages,
  URLs, or credentials (causes: `budget`/`disconnect`/`shutdown`/`cancel`).
- No post-disposal use: closes drain by default; past-deadline
  closes stay owned (`closing`), and later use fails closed.

## Native budget adapters (for F data + C06)

- `raceBoundary({source, boundMs?, budget?, scopeSignal?, signal?, sink?}, start)` —
  cancel-absent boundary; `start` runs at most once, never when
  already expired; expiry files escalation + observes late
  settlement automatically. Pure, browser-safe
  (`runtime/transport/operation-budget.ts`).
- `effectiveBoundMs(boundMs?, budget?)` — min-bound math + validation.
- `createRequestScope({totalMs?})` / `runWithRequestScope` /
  `currentRequestScope` / `scopeRaceInput(explicit?)` — ambient
  scope (server-only).
- `EscalationRecord`/`LateRecord` (`seq`-linked) via
  `scope.collected`, or `drainBudgetEscalations()` ring fallback
  (bounded 256 + drop counter) for scope-less races.

## Checker needs (coordinator → C; emitter → A)

1. Thread authored caller bounds/budgets into the trailing runtime
   inputs (SQL, action fetch, server HTTP client); make bounds
   mandatory per R04.
2. Splice static `timeoutMs` into action fetch sites (validated
   shape already accepted); decide the browser caller-cancel signal
   source (no `AbortSignal` exists in Can today — open design
   question, not an E04 input to invent).
3. Server config: `request_budget_ms` (per-request scope total) and
   ingress bound threading; until then scopes are unbounded until
   expired and Bun's pinned `idleTimeout: 10` bounds request totals.
4. Optional promotion (not required): action `timeout` phase →
   `http::timeout` in the action failure bound (catalogue + checker).

## Open limitations (carried, not hidden)

- Header stalls never dispatch (pinned Bun behavior; no Can hook).
  Operator guidance: front with a timeout-enforcing proxy if
  header-stall bounding is required. E09 should assert the
  no-dispatch pin, not a timeout.
- S3/stream paths await X-R15-* (not E04).
- E06 reporter drains `scope.collected` + the module ring; E04 files
  but never prints.
