# F03 — lookup-first PG conflict recipe + operator-DDL recipe (2026-09-26)

Source: R10; F-R10-04 (O1 lookup-first now; O2 savepoints only on
demonstrated need). Qualified live through the runtime on PG 17.11
([report](f03-live-report.json) `lookup_first`/`aborted_txn` legs,
`tests/integration/sql_f03_test.go`).

## 1. Why lookup-first

The webhook `decide_receive` shape — insert, catch `constraint_failed`,
reread the winner in the same transaction — works on SQLite (a failed
statement does not poison its transaction) but dies on PG: the first
failure aborts the transaction and every later statement fails `25P02`
(pinned: `aborted_txn.followup_outcome`). The transaction then still
resolves `ok` with nothing to commit, so an ignored query failure
silently loses the callback's work. Savepoints would recover the
transaction, but nothing in scope has demonstrated needing them; the
portable fix is to never let the transaction fail: look first.

## 2. The recipe (per decide transaction)

Within one transaction, in order:

1. `queryOptional` the row by its app-supplied key
   (`delivery_by_id`-shaped descriptor, cardinality `optional`).
2. If present: compare the stored digest/content with the incoming one
   without writing — equal means `duplicate`, different means
   `conflict` — and roll back with that receipt. No statement has
   failed; the transaction is healthy.
3. If absent: `execute` the insert, then commit with `accepted`.
4. Lost race (two writers both saw absent): exactly one insert wins;
   the loser's insert fails `constraint_failed`. The loser must roll
   back immediately and classify via a fresh refetch outside the
   poisoned transaction — never reread inside it. The refetch then
   reports `duplicate`/`conflict` by the step-2 rule.

Rules: a query failure is always the decision signal — handle it, roll
back, replay; never continue the transaction after one. The only
expected in-transaction failure in this shape is the step-4 lost race,
which is a rollback, not a reread.

Qualified live: 8 concurrent writers on one key through the runtime —
8 settled, exactly 1 accepted, the winner's digest visible to all, zero
`25P02` anywhere (`lookup_first`).

## 3. Operator-DDL recipe (P19 boundary stands)

Operator-owned DDL remains the boundary: no versioned migrations, no
drift detection. The F04 app follows this procedure, executed by hand:

1. Create the lane database and per-run databases with
   `distribution/provision-local.sh db mkdb <name>` (PG + MySQL).
2. Apply the app's `schema.sql` by hand (e.g. `psql` / `mysql` with
   operator credentials from the provisioned env files — values never
   printed, never committed).
3. Verify: connect, list tables, run one `INSERT`/`SELECT` roundtrip,
   then hand the per-run database URL to the harness via env only.
4. Schema change = new hand-applied DDL + a fresh per-run database;
   no in-place migration story exists. A versioned-migration + drift
   case reopens P19 via the coordinator.

The F03 driver's own setup/teardown (raw DDL around runtime legs over
`can_f03_run1`) follows exactly this shape.

## 4. Example-port spec (hand to coordinator / C)

The webhook `decide_receive` (`examples/webhook/src/model/model.can`)
still uses the catch-then-reread shape with a README-stated PG warning.
Porting it to §2 is example-source work (C-owned) plus assert updates:

- Restructure to lookup-first: `ledger_by_id` (`query_optional`)
  first; insert only on `none`; on `constraint_failed` roll back with
  `store_unavailable`-or-replay and classify via a fresh read.
- Keep all six `decide_receive` assert arms green (`canlc assert`, U06
  gate); the `lost_race` arm becomes rollback-then-refetch instead of
  reread-in-transaction.
- Update the README "Stated limits" PG bullet to name the recipe.
- Acceptance: webhook live suite green on SQLite (portable shape) +
  the F03 `lookup_first` legs as the PG reference.

F03 does not touch the example itself (C owns example patches).
