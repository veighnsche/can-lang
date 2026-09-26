# F04 backend qualification for H08 metered legs (2026-09-26)

Which ledger backends H08 may use for live AI evaluation spend, and
which it may not. Evidence: [f04-live-report.json](f04-live-report.json),
`runtime/test/outbound-ledger-sql.test.ts` (47 tests, three consecutive
47/47 passes on `can_f04_run1`/`can_f04_run2`), plus the F01 memory/file
suites. HONOR BOTH OUTCOMES: two backends below are disqualified, and
no unqualified backend is presented as ready.

## Ranked recommendation

1. **PostgreSQL — recommended default.** Full 13-leg matrix green;
   two-process race exactly 25 admitted / 25 exceeded with final held
   1000 (677–994 ms per 50-op race here); restart/reconnect durable;
   crash-before/after-commit recovery verified; isolation pinned
   explicitly per load transaction. Multi-process and multi-machine
   safe by construction; state is SQL-inspectable for audit.
2. **File backend — recommended when single-machine simplicity wins.**
   Restart survival and exact cross-process admission green here (H07
   race plus the F04 2×25 race, 97–271 ms) with zero database
   infrastructure. Single machine only; corruption fails closed and
   needs restore-from-backup (never auto-reset).
3. **MySQL 8.4 — qualified alternate.** Same full matrix green
   (two-process race exact, 1000–1411 ms). Two standing constraints:
   server ≥ 8.0.16 for CHECK enforcement, and isolation at or above
   REPEATABLE-READ — the backend refuses the store on drift. Pick it
   where MySQL is the house database.
4. **SQLite file — qualified local alternate.** Same full matrix
   green; two-process race exact (93–128 ms). Single machine;
   same-handle operations serialize on the one connection by design,
   cross-handle races use the version gate. Pick it for
   zero-infrastructure local runs that still want SQL-inspectable
   state.

DISQUALIFIED for metered legs: **memory** and **sqlite-memory** —
process-local, non-durable by construction (a second handle starts
empty; proven, not assumed). Valid for unit tests only. If H08 runs
any metered leg on either, that leg's accounting evidence is void.

## What was proven per qualified backend

- Exact-fit admission and immediate typed rejection (`exceeded`
  carries required/remaining/reset; the rejection reserves nothing).
- Full reservation lifecycle: fence idempotence, partial-release
  settlement, duplicate/conflicting settlement, fenced-hold vs
  unfenced-release, unknown-invocation outcomes.
- Epoch transitions: live rows keep pinned limit/version, expired
  holds never spend the new epoch, late usage debits the original
  admission epoch.
- Unknown holds: invalid usage keeps the full hold; holds survive
  restart; only authoritative usage reconciles.
- Breach quarantine persists across restart and blocks the profile.
- CHECK floor enforced natively on all three SQL dialects.
- Missing schema fails closed (`storage-unavailable`, never
  auto-create); unreachable servers fail with static text (no URL
  leak).
- 256-char metering names and `MAX_SAFE_INTEGER` limits round-trip
  exactly on all dialects (this required the F04 parity fix below).

## Parity fix applied during qualification (F-owned)

Contract-valid 256-char metering provider/model/version names
overflowed MySQL `VARCHAR(191)` columns (observed: strict-mode error
1406 on a 256-char value) while fitting PG/SQLite TEXT — a real
cross-dialect divergence, not a theoretical one. Fixed at the root:
the quarantine key is now a fixed 64-hex-char digest and the non-key
metering columns are TEXT on MySQL. Key columns stay `VARCHAR(191)`.
No compat shim: zero external users.

## Limits of this qualification

- Ledgers only: this qualifies the accounting substrate, not a
  metering profile. H08/X-R14-1 must still qualify a real
  whole-call bound U before claiming successful budgeted operation;
  without one, W6-AI stays blocked per BLK-01 even though every
  ledger leg here passes.
- Loopback scale: races are 2 processes × 25 reservations; enough to
  prove exactness, not a throughput benchmark. Absolute timings are
  reported for stability comparison only.
- The real-PG-app shape legs (generated identities, precise values,
  nullable audit times, JSON, concurrent replay) ride on F02/F03
  evidence through the shared SQL contract; F04 adds the ledger
  backend matrix, not a second app.
