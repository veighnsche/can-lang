# E04 validation evidence (2026-09-26)

Task: `can-implementation-task-list-2026-09-26.md` E04 (absent-cancel
branch). Live PG/MySQL legs ran in isolated `can_e04_run2` (minted
via `distribution/provision-local.sh db mkdb`); SQLite legs in
memory; servers on loopback (ephemeral for fetch legs, fixed
E04-owned 18761–18767 for dispatch legs). Env credentials only, never
printed.

## Commits

- `3bd4f69f` boundary + request scope
- `06e3e00e` SQL bounds
- `fed699f6` fetch/action bounds
- `d462ee03` dispatch propagation + bounded ingress

## Checks (pass/fail)

| Check | Result |
|---|---|
| `bun test runtime/test/operation-budget.test.ts` (23) | PASS |
| `bun test runtime/test/sql-budget.test.ts` (15, live) | PASS |
| `bun test runtime/test/action-budget.test.ts` + `transport-budget.test.ts` (23) | PASS |
| `bun test runtime/test/server-budget.test.ts` (13, live + child SIGTERM) | PASS |
| E04 total: 74 new legs | PASS |
| `bun run check:runtime` (lint + format + typecheck) | PASS |
| Full suite `bun test runtime/test/` | 867 pass / 1 pre-existing env failure (`owner-explicit-browser`, missing Playwright module — fails identically without E04; untouched files) |
| Emits coverage (all budget outcomes inside existing emits) | OK, no catalogue change |

## Validation coverage vs task list

- Absent-cancel boundary-return: sql-budget (query/execute/tx ×
  PG/MySQL/SQLite) + server-budget disconnect legs.
- Held leases: `resourceStatus leases === 1` past the visible
  return, 0 after late settlement (query + tx legs).
- Commit unknown: `commit_unknown` mid-flight tx legs (PG + MySQL)
  with late-commit reread; never-started tx reports the phase.
- Overlapping pool use: concurrent short/long bounds on one pool;
  second-query-during-overrun in every core leg.
- Header/body stalls: header no-dispatch pin; drainBody 408/400/413
  legs; lazy live-read abandonment; shutdown-during-ingress 408
  over the wire; Bun bounds pinned by probe (below).
- Late settlement: linked `LateRecord` (resolved/rejected) in every
  overrun leg; no unhandled rejections.
- SIGTERM: child-process E2E (bounded `unknown:shutdown` to the
  live peer, escalation cause, clean exit) + signal-past-deadline
  leg (bounded `shutdown_failed(deadline)`, close owned, SIGKILL).

## E04 Bun ingress probes (raw sockets, Bun 1.4.2)

- Partial headers, never completed: fetch never fires (12s+);
  connection held open, no idle close observed. No Can hook exists.
- Full headers, zero body: fetch fires at once; body read pends;
  Bun aborts with `AbortError` at ~10s (`idleTimeout` default) and
  closes. Warn: "timed out a request after 10 seconds".
- 1 byte/3s drip: chunks delivered; Bun still aborts at ~10s from
  request start (total-bound, not idle-since-byte); serve-side
  signal aborts alongside.
- Disconnect mid-ingress: serve-side signal + read `AbortError`,
  both prompt (~ms).

E04 pins `idleTimeout: 10` (observed default, now explicit) and
maps `AbortError` drains to 408.

## Design consultation

Jev 3/3 (`e04-jev/`): ambient-scope propagation, reuse of
`sql::query_failed(code budget)`, 408 for stalled bodies. Agreement
treated as advice; the weak 0.30 stall-status leg investigated and
decided on honesty grounds (400 misclassifies stalls as malformed).
