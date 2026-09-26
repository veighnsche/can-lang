# Outbound policy (C-G, lane F)

Source: R11/R14/R01; F-R11-03; R14 accounting supplement
(`docs/syntax-taste/preparation-2026-09-26/blocker-resolution/README.md`,
BLK-01). Task F01 publishes the shared outbound identity vocabulary,
the destination/redirect policy with credential binding, and the R14
durable ledger service. H/E consume the native contracts below;
publication is independent of H implementation.

## 1. Identity vocabulary (implemented, F01)

`runtime/outbound/identity.ts` owns the single tenant/pool/correlation
vocabulary shared by HTTP companion delivery and AI budget
enforcement.

- `tenantId` / `poolId` / `correlationId` / `invocationId` validate
  one shared shape: 1..128 chars, leading ASCII alnum, then letters,
  digits, `.` `_` `:` `-`. Anything else is a `TypeError`.
- Tenant and pool come from trusted application authorization.
  Browser-supplied tenant claims alone never authorize an accounting
  identity, and the module offers no browser-claim parsing.
- `invocationContext(tenant, pool, correlation?)` builds the frozen
  native accounting identity H guards and E transports. An omitted
  correlation is minted (`crypto.randomUUID`); each provider attempt
  still needs its own invocation record and reservation.
- `nestContext(active, next)` enforces the scope rule: same
  tenant/pool nesting returns the active context unchanged, replacing
  the tenant/pool is a `TypeError` (H maps it to
  `ai_budget::unavailable`).
- `contextReport` projects `{ tenant, pool, correlation }` for
  diagnostics. Identity records hold no credentials, endpoints, or
  prompt text and are safe to log.

Contract tests: `runtime/test/outbound-identity.test.ts`.

## 2. Destination and redirect policy (implemented, F01)

`runtime/outbound/destination-policy.ts` owns tenant-webhook
admission at the companion boundary. Fixed-origin HTTP stays unless a
policy here admits the destination; unrestricted fetch stays rejected.

- `destinationPolicy(parsedJson)` strictly validates a companion
  policy file: pinned `version`, `rules`, `redirect` mode
  (`deny` | `same-origin` | `allowlisted`), `maxRedirectHops` (0..8),
  and `loopback` / `privateNetworks` scopes (default `deny`).
- Rules pin `scheme` (`http` | `https` | `any`), exact or
  `*.suffix` wildcard `host` (subdomains only — never the apex, never
  IP literals), optional exact `port` (absent means the scheme
  default port only), optional `pathPrefix`, and optional
  `credential` environment name. No regex: rules stay auditable.
- `evaluateDestination(policy, url)` admits or denies one URL with a
  finite safe reason. Userinfo and fragments deny; IP literals face
  the loopback/private scopes (`isPrivateAddressLiteral` is exported
  so the F05/D01 send path can re-check the resolved literal before
  connecting — DNS names can resolve private after the allowlist
  check, and that TOCTOU stays with the send path).
- `evaluateRedirect(policy, requestUrl, location, hopsSoFar)`
  rules one hop: `deny` stops every hop, `same-origin` keeps the
  posting origin (plus the scheme/userinfo/fragment/scope floor),
  `allowlisted` re-admits the target through the full policy.
  Relative `Location` values resolve against the posting URL; the
  caller counts hops and denies past `maxRedirectHops`.
- Credential binding is env-name-only
  (`/^[A-Z_][A-Z0-9_]*$/`, same spelling as `env::required`).
  `credentialValue` resolves the value at send time from the
  injected environment; the value goes straight into the send path
  and never enters a decision, report, log line, or artifact.
  `redactUrl` strips userinfo/fragments for logs; `policyReport`
  projects the safe `{ version, redirect, maxRedirectHops, rules }`
  summary (environment names are configuration, not secrets).

Fixtures: `runtime/outbound/fixtures/destination-fixtures.json`
(allowed/denied/redirect corpus with exact reasons, including
loopback/private, userinfo, fragment, wildcard-apex, and port cases)
and `runtime/outbound/fixtures/companion-policy.json` (the shipped
example companion file F05 loads). Contract tests:
`runtime/test/outbound-destination.test.ts`, including a canary test
proving credential values never reach decisions, redirects,
redactions, reports, or the serialized policy.

## 3. Fixed-epoch schedules (implemented, F01)

`runtime/outbound/epoch.ts` owns the configured fixed periods. Each
pool pins a token `limit` (nonnegative), `periodMs` (positive),
UTC `anchorMs`, and `version`. The ledger clock determines the
half-open epoch `[start, end)` holding admission; times before the
anchor are in no epoch. `transitionSchedule` moves a pool to a new
schedule effective only at a boundary of the old schedule — that
boundary is the new anchor — and rejects overlapping, retroactive,
or version-preserving transitions. Existing epoch rows always retain
their original start, end, and limit.

Fixed epochs are an allowance per configured period, not a rolling
rate limit: two allowances can fall on either side of a boundary.
That distinction is intentional; H/E documentation must preserve it.

Contract tests: `runtime/test/outbound-epoch.test.ts`.

## 4. Durable ledger service (implemented, F01)

`runtime/outbound/ledger.ts` is the native reserve/fence/settle/
reconcile contract H guards at dispatch. One tenant allowance
counts `input_tokens + output_tokens`, unweighted, per
(tenant, pool, epoch) row. All token arithmetic is checked safe
integers; comparisons that could overflow subtract from known-safe
headroom instead of summing unbounded counters.

- `configurePool` pins a pool's first schedule; re-pinning is
  `invalid-policy` and reconfiguration goes through
  `transitionPool` (§3 rules).
- `reserve` atomically admits the conservative upper bound U only
  if `committed + held + U <= limit` in the current epoch.
  Otherwise it returns `exceeded` immediately — `{ pool,
  required, remaining, epochResetMs }`, no provider I/O, no refill
  wait. Unqualified profiles return `missing-qualification`;
  quarantined profiles return `breached-profile`; unknown pools
  and pre-anchor clocks return `invalid-policy`. A supplied
  invocation ID makes reserve idempotent: retrying after an
  ambiguous commit returns the existing admission instead of
  double-charging, while changed parameters conflict.
- `fence` persists the once-only dispatch fence before provider
  I/O. Fencing is idempotent; fencing a settled or released
  attempt conflicts. A crashed or uncertain fenced attempt is
  never resent on the same reservation.
- `settle` reconciles authoritative actuals in the original
  admission epoch: with A <= U the hold lifts, A debits, and U-A
  releases. Duplicate settlement is a no-op; conflicting usage is
  `conflicting-settlement`; invalid or missing usage is
  `invalid-usage` and the full hold stays unresolved. Actuals
  above U record without clamping, quarantine the profile, and
  report the breach (H invalidates the budgeted result with
  `ai_budget::unavailable`).
- `releaseUndispatched` releases U only when the fence proves no
  send could have occurred (never fenced). Fenced attempts keep
  their hold (`held` outcome); usage after a proven no-dispatch
  release conflicts.
- `reconcile` reads one durable record by invocation ID without
  mutating: the crash/ambiguous-commit recovery path. Absent
  means no known reservation — never dispatch on it.
- `epochStatus` and `ledgerHealth` report pinned rows, committed
  vs held tokens, unresolved attempts, and quarantined profiles
  for operators. Totals saturate rather than lose precision;
  reports carry no secrets.
- Malformed caller input is a `TypeError` (validate trusted
  context first with §1); every state-dependent failure is a
  typed `unavailable` outcome with a finite reason H maps to
  `ai_budget::unavailable`. `ambiguous-commit` always carries the
  attempted invocation ID and correlation for reconciliation.
  Storage failures fail closed as `storage-unavailable`; nothing
  auto-resets and nothing resolves commit-unknown by guessing.

Backends: `createMemoryLedgerStore` (unit tests),
`openFileLedgerStore` in `runtime/outbound/file-ledger.ts`
(JSON + atomic rename + fsync under a lockfile, shared across
processes, surviving restart; stale locks are stolen past a bound
and corruption fails closed without auto-repair). F04 qualifies
live SQL backends against `runtime/outbound/ledger-schema.ts`,
which publishes operator-owned DDL for SQLite, PostgreSQL, and
MySQL 8.0.16+ plus the version-CAS atomic-op mapping (short
native transactions, never held across provider I/O). F01
executes the SQLite leg (DDL, round-trip, CHECK floor); the
Postgres/MySQL legs await live backends in F04.

Contract tests: `runtime/test/outbound-ledger.test.ts`
(exact-fit/rejected admission, idempotent fencing/settlement,
concurrent writers on both backends, restart, commit-unknown
recovery, unknown/invalid/conflicting usage, breach quarantine,
late settlement, epoch transitions, fail-closed storage).

## 5. Failure mapping for H07

- Ledger `exceeded` → `ai_budget::exceeded` with the opaque pool
  ID, required U, remaining tokens, and epoch reset time.
- Ledger `unavailable` reasons (`missing-context`,
  `invalid-policy`, `storage-unavailable`, `ambiguous-commit`,
  `invalid-usage`, `conflicting-settlement`,
  `missing-qualification`, `breached-profile`,
  `unknown-invocation`) → `ai_budget::unavailable` reason
  variants with safe correlation/occurrence identity. Never
  attach native messages, endpoints, credential values, or raw
  provider bodies. Catalogue/checker integration for the scope
  operation and typed failures goes through existing A/E
  ownership; no generated hand edits.
- `missing-context` is reserved for H's guard (absent scope
  context at dispatch); the ledger itself reports the other
  reasons. H also validates that the context pool is the guarded
  connection's configured permitted pool — an arbitrary pool name
  never creates fresh allowance.

## 6. Handoffs

- H07/E: C-G identity (`identity.ts`), epoch schedules
  (`epoch.ts`), and the durable ledger contract (`ledger.ts` +
  `file-ledger.ts`) are published and tested; E integrates the
  shared invocation/transport context, H guards dispatch and
  settlement. No F/H design cycle remains on these contracts.
- D01/F05: network rules (`destination-policy.ts`, fixtures,
  `companion-policy.json`) are published; the companion send
  path owns resolve-then-recheck of DNS literals via
  `isPrivateAddressLiteral` plus redirect hop counting.
- F04: `ledger-schema.ts` plus the executed SQLite leg are the
  backend-qualification input; live PG/MySQL evidence stays
  open until provisioned backends exist.

## 7. Done-state ledger (F01)

F01 claims exactly: §§1–4 implemented and tested; §5 mapping
published for H07; §6 handoffs unblocked. Not claimed: guarded
dispatch, usage extraction, profile qualification, the scope
operation, typed catalogue failures (H07); transport/context
integration (E); companion delivery (F05); live PG/MySQL ledger
qualification (F04).
