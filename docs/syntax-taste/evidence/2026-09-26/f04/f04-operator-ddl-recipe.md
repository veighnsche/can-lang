# F04 operator-DDL recipe: token/cost ledger schema (2026-09-26)

Source: R14/R10; F-R14-01/02; C-F/C-G. Migration stays operator-owned
(P19 deferral retained): this recipe is the procedure, not versioned
migrations. Canonical schema source is `ledgerDDL()` in
`runtime/outbound/ledger-schema.ts` — never hand-edit a deployed
schema; regenerate and compare.

## 1. Render the script for one dialect

```sh
# sqlite | postgres | mysql; prints the five tables plus the version seed
bun -e 'import("./runtime/outbound/ledger-schema.ts").then(m =>
  console.log(m.ledgerDDL(process.argv[2]))' -- postgres > /tmp/ledger-pg.sql
```

Tables: `ai_budget_state` (single version row, the optimistic commit
gate), `ai_budget_schedules` (append-only schedule versions per pool),
`ai_budget_epochs` (pinned epoch rows), `ai_budget_invocations`
(reservation records), `ai_budget_quarantine` (breached profiles).

## 2. Apply to a fresh database

```sh
# PostgreSQL (database must exist; app role must own or be granted it)
psql "$CAN_TEST_POSTGRES_URL" -v ON_ERROR_STOP=1 -f /tmp/ledger-pg.sql
# MySQL 8.0.16+ (older servers parse CHECK and ignore it — unqualified)
mysql --protocol=TCP -h 127.0.0.1 -P 3307 -u canapp -p can_f04 < /tmp/ledger-mysql.sql
# SQLite file
sqlite3 /var/lib/can/ledger.db < /tmp/ledger-sqlite.sql
```

The script is intentionally not idempotent: rerunning on a migrated
database fails on the existing tables instead of altering them. For a
disposable test database, `applyLedgerDDL()` in
`runtime/outbound/sql-ledger.ts` runs the same statements; production
DDL stays this explicit operator step. Never run DDL while the ledger
is serving traffic.

Least privilege for the app role: `SELECT/INSERT/UPDATE/DELETE` on the
five `ai_budget_*` tables only — no DDL, no other tables.

## 3. Verify before serving

```sql
-- Seed row present at version zero (all dialects).
SELECT version FROM ai_budget_state WHERE id = 1;  -- exactly one row: 0
-- CHECK floor enforced (each statement must FAIL; MySQL reports 3819).
INSERT INTO ai_budget_epochs (tenant, pool, epoch_start, epoch_end, token_limit,
  schedule_version, committed, held)
  VALUES ('probe','probe',0,1,0,'1',-1,0);
UPDATE ai_budget_invocations SET state = 'maybe' WHERE invocation_id = 'probe';
```

The F04 suite executes these probes live per dialect
(`the CHECK floor rejects invalid rows natively`); a server that
accepts any of them is unqualified — stop and fix the server, never
the application.

MySQL session requirement: transaction isolation at or above
`REPEATABLE-READ` (the shipped default). The backend verifies
`@@transaction_isolation` at open and refuses the store on drift,
because load snapshots need it. Do not lower it.

## 4. Operate

- Back up with native tooling (`pg_dump`, `mysqldump --single-transaction`,
  SQLite file copy while idle, ledger-JSON copy for the file backend).
- Health, without secrets: committed vs held vs unresolved counts come
  from `ledgerHealth()`/`epochStatus()`, or directly:
  `SELECT committed, held FROM ai_budget_epochs;` plus
  `SELECT state, COUNT(*) FROM ai_budget_invocations GROUP BY state;`.
- Unknown holds after a crash or restore: reconcile by invocation ID
  (`reconcile()`), never by guessing. Absent means no known
  reservation — never dispatch on it. Found-and-unresolved stays held
  until authoritative usage or definitive no-dispatch proof arrives;
  no timer or restart refunds it.
- The ledger never auto-resets: corruption or a missing schema fails
  closed (`storage-unavailable`, surfaced as `ai_budget::unavailable`).
  Recovery is restore-from-backup, then reconcile unknowns.
- Schedule changes go through the `transitionPool` rules (effective at
  an old-schedule boundary, never overlapping or retroactive), not
  through manual row edits.

## 5. Quarantine key note

`ai_budget_quarantine.profile_key` is a fixed 64-hex-char digest of the
metering provider/model/version tuple; the human-readable
`provider/model/version` spelling lives in `profile_identity` (and on
every invocation record). Metering names up to 256 chars each fit all
three dialects — verified by the maximum-length leg.
