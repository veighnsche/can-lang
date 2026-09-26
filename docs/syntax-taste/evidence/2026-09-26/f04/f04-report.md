# F04 — ledger backend qualification across SQLite/PG/MySQL (2026-09-26)

Source: R10/R14; F-R10-01..05, F-R14-01/02; C-F/C-G. Prerequisites done
on main: F01 (ledger service + memory/file backends + SQL schema), F03
(`bf83f0aa`: per-dialect RETURNING contract, lookup-first recipe,
encoding guide). Machine record:
[f04-live-report.json](f04-live-report.json). Live services: PG 17.11
(127.0.0.1:55433), MySQL 8.4.11 (127.0.0.1:3307), SQLite 3.54.0 (Bun
1.4.2); per-run databases `can_f04_run1` (two full passes) and
`can_f04_run2` (one full pass), created via `provision-local.sh db
mkdb`. Env credentials only, never printed.

## What F04 delivered

- `runtime/outbound/sql-ledger.ts`: `LedgerStore` over SQLite,
  PostgreSQL, and MySQL on native Bun.SQL clients. Loads run in a
  repeatable-read transaction (PG issues `SET TRANSACTION` first;
  MySQL default verified at open with drift refused; SQLite reads
  under its deferred snapshot) so the (version, tables) pair is
  atomic. Commits run one transaction led by the conditional version
  bump; losers roll back with `{ committed: false }` for service
  retry. SQLite same-handle calls serialize on a mutex (one client is
  one connection); cross-handle races use the version gate. Missing
  schema, unreachable servers, and corrupt rows fail closed; the
  store never creates schema.
- Parity fix (F-owned root cause): quarantine key is now a fixed
  64-hex-char digest and MySQL non-key metering columns are TEXT, so
  contract-valid 256-char metering names fit every dialect. Before
  the fix, MySQL strict mode rejected them with error 1406 while
  PG/SQLite accepted them (observed, not theorized). Key columns stay
  `VARCHAR(191)`; `epochRowKey`/`profileKey` exported for the backend.
- `runtime/test/outbound-ledger-sql.test.ts` (47 tests) +
  `runtime/test/ledger-race-worker.ts`: per-backend legs for
  exact-fit admission + immediate rejection, full lifecycle,
  unknown-hold settlement across restart, epoch transitions with
  late settlement, durable breach quarantine, in-process
  concurrency, stale-commit races, native CHECK floor, fail-closed
  missing schema, maximum-length identities, `MAX_SAFE_INTEGER`
  accounting, health counts, staged crash before/after commit — plus
  two-process exact-admission races on all four durable backends.
- [Operator-DDL recipe](f04-operator-ddl-recipe.md) for the token
  ledger (render/apply/verify/operate; P19 operator-owned boundary
  retained) and the [ranked H08 backend
  recommendation](f04-backend-recommendation.md): PostgreSQL
  (default), file (single-machine simplicity), MySQL 8.4,
  SQLite-file qualified; memory and sqlite-memory disqualified for
  metered legs.

## Validation summary

- `bun test runtime/test/outbound-ledger-sql.test.ts` — 47/47, three
  consecutive passes (run1 ×2, run2 ×1); pg/mysql legs skip cleanly
  without env.
- `bun test runtime/test/outbound-ledger.test.ts
  runtime/test/ai-budget.test.ts runtime/test/ai-budget-adapters.test.ts`
  — 59/59 (no F01/H07 regression from the digest change).
- `bun run check:runtime` (lint + format-check + typecheck) — clean.
- No E/runtime files touched (pool/transaction/descriptor/catalogue
  unchanged — the backend uses native Bun.SQL directly); no
  compiler, browser, LSP, generated, or vendor edits.

## Handoffs (to coordinator)

1. **To H08: backend qualification + operator recipe.** The
   recommendation above is the backend choice for H08's metered legs;
   the recipe is the provisioning procedure. Limits: ledgers only —
   H08/X-R14-1 must still qualify a real whole-call bound U; no
   profile, no successful budgeted operation.
2. **To E: none.** No runtime transport change needed; no defect
   found in E-owned code. (F03's cold-pool `withTransaction` burst
   note is E's existing item, untouched here.)
3. **To F06/H12/H13:** the `claim_outbox` lease path can reuse this
   backend's optimistic version-gate pattern; no code dependency.

## Scope retained

The W6 real-PG-app shape legs ride on F02/F03 evidence through the
shared SQL contract; operator-owned DDL stands (no versioned
migrations). No new syntax, column kinds, or claim-policy choice.
