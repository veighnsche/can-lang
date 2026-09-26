# Request policy (C-C, lane E)

Source: R04; F-R04-01/04/06; C-C. Task E01 publishes the base policy and
vocabulary; E02/E04/E06 implement the conditioned native and escalation
mechanisms. Nothing in this spec claims a native interruption meaning
until its probe qualifies it (P07.5).

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

## 5. Owned remaining work (contract published; wiring in E04)

Cancel-absent branch: the request boundary still returns at budget
expiry while the native operation stays owned until settlement,
reporting the §4 outcome with supervisor escalation — bounded and
honest at the user-visible layer without pretending to abort the
unabortable. E01 proves the shape with a generic boundary race in
`request-budget.test.ts`; E04 wires it into SQL/fetch/action paths.

## 6. Supervisor escalation (contract published; mechanism in E04/E06)

Waiting deadlines leave work owned. The supervisor bounds nonsettling
work (SIGKILL as the final bound); escalation is automatic, not
caller-invoked. The `escalation: "supervisor"` marker field is the
handoff token E04/E06 mechanisms consume.

## 7. Per-adapter interruption contracts (conditioned, not implemented)

No caller deadline/cancel operand ships without a verified native
meaning. Each line below is conditioned on its probe; E01 adds no
operands and changes no adapter behavior.

- SQL cancel semantics: conditioned on X-R04-1 (E02 qualifies per
  dialect; SQLite vs live PG/MySQL recorded separately).
- Ingress disconnect projection: conditioned on X-R04-3 (E02; pinned
  Bun behavior tested separately from SQL cancel).
- S3/stream behavior: conditioned on X-R15-* (R15 O1/O2 branches).
- Fetch abort: wired in E04 against the qualified native contract only.
- Hedged-loss supervision (O2): designed only if X-R04-2 shows O1
  insufficient; no O2 policy work before the measurement.

## 8. Disconnect/SIGTERM propagation (E04)

E04 propagates ingress disconnect and SIGTERM through the budget and
ownership layers per §§1–6. E01 publishes the layers; no propagation
behavior is claimed now.

## 9. Shutdown escalation (E04/E06)

Service shutdown composes §2 drain with §6 escalation: bounded wait,
then supervisor bounds. Server `shutdownMs` close deadlines keep their
current meaning until E04 republishes them against this spec.

## 10. Identity plug (C-G, owned by F)

Budget/correlation identity vocabulary is owned by lane F (C-G). This
policy consumes it; neither owner invents the other's vocabulary. F01
picks up the §4 vocabulary from this publication.

## Conditioned-claim ledger (E01 done-state)

E01 claims exactly: §§1–4 implemented and tested; §§5–6 contracts
published with markers but no adapter wiring; §§7–9 explicitly not
implemented. Any native interruption, operand, propagation, or
escalation-mechanism claim before E02/E04/E06 lands is out of contract.
