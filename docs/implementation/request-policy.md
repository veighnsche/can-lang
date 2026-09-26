# Request policy (C-C, lane E)

Source: R04; F-R04-01/04/06; C-C. Task E01 publishes the base policy and
vocabulary; E02/E04/E06 implement the conditioned native and escalation
mechanisms. Nothing in this spec claims a native interruption meaning
until its probe qualifies it (P07.5). Request failure reporting
(R12/E06) is specified in
[request-failure-reporting.md](request-failure-reporting.md).

## 1. Shared request budget (implemented, E01)

Each request carries exactly one shared budget
(`runtime/transport/request-budget.ts`). Each operation's effective
deadline is min(its own bound, remaining budget). Serial per-operation
budgets (5s+5s+5s = 15s user-visible) are rejected as the default;
per-operation bounds remain mandatory inputs. The budget is pure time
accounting: it holds no lease, revokes nothing, and never settles the
operations it bounds.

- `createRequestBudget(total)` starts the budget; total and every
  operation bound are safe integers in [1, 2147483647]; anything else is
  a `TypeError`.
- `effectiveMilliseconds(bound)` reports `min(bound, remaining)`,
  floored at zero. Zero means "return unknown-write without starting",
  never a zero-length native wait.
- `remainingMilliseconds()` never reports below zero; `expired()` is
  `remaining() <= 0`.

Contract tests: `runtime/test/request-budget.test.ts` (composition,
exhaustion, validation, monotonic default clock).

## 2. Drain default (implemented, ownership core)

Closing waits for leases; it never force-closes or discards a live
lease (`beginClose` in `runtime/owner-core.ts`). Scopes drain admitted
work before they close. This is the default every request boundary,
adapter, and shutdown path inherits.

## 3. Lease preservation (implemented, E01 tests)

Timers and budgets hold no lease and revoke none. A budget expiring
while work holds a lease changes nothing about that lease: the resource
stays usable, fresh uses still work, and close still waits for release.
The E01 ownership test proves expiry-then-drain against the real owner
core.

## 4. Unknown-write vocabulary (implemented, E01)

One honest outcome across SQL, S3, actions, workers, and fetch when the
visible boundary returns before the native operation settles:

```ts
{ kind: "unknown-write", source, commit: "unknown", owned: true, escalation: "supervisor" }
```

- `source` is one of `sql | s3 | action | worker | fetch`; anything
  else is a `TypeError` at construction and `false` under
  `isUnknownWrite`.
- `commit: "unknown"` is never silently resolved to committed/aborted
  by the vocabulary itself. Lookalikes that resolve the commit, drop
  ownership, or skip escalation are not the vocabulary.
- Markers are frozen at creation. A late native settlement is observed
  separately and never rewrites a returned marker.
- Catalogue adapters construct domain values from these markers; native
  causes never become authored error fields (same rule as
  `TransportProblem`).

Consumers: E04 adapter wiring, F companion protocol (cancel/unknown
outcomes mirror Can-side honesty), S3 contracts (C-C).

## 5. Owned remaining work (E04 implemented)

Cancel-absent branch: the request boundary still returns at budget
expiry while the native operation stays owned until settlement,
reporting the §4 outcome with supervisor escalation — bounded and
honest at the user-visible layer without pretending to abort the
unabortable. `runtime/transport/operation-budget.ts` (`raceBoundary`)
is the boundary; SQL adapters race it (`SQLBounds`, no cancel input);
leases stay held past the visible return; late settlement is observed
through the sink without rewriting the marker.

## 6. Supervisor escalation (E04 mechanism; E06 reporting)

Waiting deadlines leave work owned. The supervisor bounds nonsettling
work (SIGKILL as the final bound); escalation is automatic, not
caller-invoked. Every unknown-write return files an
`EscalationRecord` through the request-scope collector
(`scope.collected`) or the module ring fallback
(`drainBudgetEscalations`); late settlement files a linked
`LateRecord`. E06's reporter drains these collections; E04 files but
never prints.

## 7. Per-adapter interruption contracts (E04 wired; operands in C)

No caller deadline/cancel operand ships without a verified native
meaning. Runtime inputs are wired; authored operands thread through
them in C-owned checker/emitter slices.

- SQL cancel semantics: X-R04-1 NEGATIVE on all dialects — no cancel
  input exists on any SQL path (`raceBoundary` rejects one
  structurally). Caller bounds race the shared budget; expiry reports
  `query_failed` code `budget` / `transaction_failed` phase `budget`
  / `commit_unknown`, all already in the emits unions.
- Ingress disconnect projection: X-R04-3 POSITIVE — the serve-side
  signal expires the request scope; in-flight budgeted work returns
  its unknown-write outcome (cause `disconnect`).
- S3/stream behavior: conditioned on X-R15-* (R15 O1/O2 branches);
  still not wired.
- Fetch abort: cancel-present branch — expiry aborts the wire
  natively and reports `http::timeout` (server client, naming the
  bound that fired) or `transport_failed` phase `timeout` (action
  client); caller cancel reports `cancelled`. No commit knowledge
  either way.
- Hedged-loss supervision (O2): X-R04-2 measured O1 sufficient
  (E05, `evidence/2026-09-26/e05-x-r04-2.md`) — O2 is inactive.
  Selected O1 hedge shapes: S1 let-settle for cancel-absent
  replicas (SQL/bare work: pool leases, never request-scope
  groups, response never waits); S2 abort-loser for cancel-present
  replicas (fetch: qualified native abort at the win, drain
  releases). Loser observation stays owner-held (hedge fates plus
  genuine boundary expiries in the sink); terminal precedence is
  expired over rejected; replicas are reads or per-shape-proven
  idempotent writes (zero write shapes qualified). O2 trip
  conditions are recorded in the E05 evidence; absent those, no
  supervised-loser policy work exists.
- Ingress stalls (E04 probes): header stalls never dispatch
  (Bun-owned, no Can hook — documented, not worked around); body
  stalls and drips abort at Bun's ~10s request bound (pinned
  `idleTimeout: 10`), mapped to fixed 408; corrupt reads stay 400.

## 8. Disconnect/SIGTERM propagation (E04 implemented)

Dispatch installs one `RequestScope` per handler run (unbounded
until expired — no invented default total); budgeted SQL/fetch
adapters resolve it implicitly. Disconnect expires the scope with
cause `disconnect`; `stop()` and signal entries expire all live
scopes with cause `shutdown` before the close drains. Scope expiry
abandons ingress drains (408) and returns in-flight operations at
their boundaries; owned work runs to settlement with escalation
filed automatically.

## 9. Shutdown escalation (E04 republication; E06 reporting)

Service shutdown composes §2 drain with §6 escalation: bounded wait,
then supervisor bounds. E04 republication: `stop()` expires scopes,
then closes with the `shutdownMs` deadline as before; the signal
path now does the same (previously no deadline at all), and
`wait()` bounds the post-signal drain by `shutdownMs`, reporting
`shutdown_failed` phase `deadline` past it while the close stays
owned for the external supervisor (SIGKILL) to bound.

## 10. Identity plug (C-G, owned by F)

Budget/correlation identity vocabulary is owned by lane F (C-G). This
policy consumes it; neither owner invents the other's vocabulary. F01
picks up the §4 vocabulary from this publication.

## Conditioned-claim ledger (E04 done-state)

E04 claims: §§1–6 and §8 implemented and tested; §7 wired except
S3/stream (X-R15-*) and O2 (X-R04-2); §9 republished with the
signal-path bound; §10 unchanged (F-owned). Authored caller operands
(checker/emitter threading) and the E06 reporter are explicitly not
in E04. Any native-interruption, operand, or reporting claim beyond
this ledger is out of contract.

E05 adds: X-R04-2 resolves the §7 hedge line — O1 sufficient, O2
inactive, no policy code. The two selected hedge sub-shapes (S1
let-settle, S2 abort-loser) are measured in
`runtime/test/http-hedge.test.ts` (H0–H12) over the E04 vocabulary;
S3/stream hedge stays unmeasured and unclaimed.

## E02 qualification outcome (X-R04-1 / X-R04-3)

Evidence: `evidence/2026-09-26/e02-x-r04-1-3.md`; probes
`runtime/test/sql-cancel.test.ts`, `runtime/test/ingress-disconnect.test.ts`.

- SQL cancel (§7 line 1): NEGATIVE on all dialects. Pinned
  `Query.cancel()` flips a client flag only — live PG/MySQL backends
  keep executing to completion, SQLite is synchronous, and cancelling
  before execution leaves the await unsettled. No caller operand maps
  to it; SQL takes the cancel-absent branch (§5) in E04/E09.
- Ingress disconnect (§7 line 2): POSITIVE. Pinned `Bun.serve`
  projects peer disconnect as serve-side `Request.signal` abort for
  buffered and streaming handlers alike; the handler itself is not
  terminated. E04 may map observation-only disconnect response to it.
