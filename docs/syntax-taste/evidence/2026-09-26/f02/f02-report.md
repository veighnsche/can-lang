# F02 — SQL expressibility and generated-identity experiments (2026-09-26)

Source: R10; F-R10-01/03/05; X-R10-1; C-F. Machine record:
[f02-live-report.json](f02-live-report.json). Live services: PG 17.11
(127.0.0.1:55433) and MySQL 8.4.11 (127.0.0.1:3307); lane database `can_f02`
plus per-run `can_f02_run1`. All Can-path legs ran through the real runtime
(`createSQLPools`/`createSQLTransactions`) with descriptor segments mirroring
`compiler/internal/sql` checker output; RETURNING appears only in raw oracle
legs, never in a descriptor. No grammar, checker, runtime, or catalogue file
was changed by this experiment.

## X-R10-1 locking-read probe — verdict: EXPRESSIBLE (admitted)

- Checker: `FOR UPDATE`, `FOR UPDATE SKIP LOCKED`, `FOR UPDATE NOWAIT`,
  `FOR NO KEY UPDATE SKIP LOCKED`, and `FOR SHARE` all check as ordinary
  row-returning `SelectStmt` descriptors with the trailing `LIMIT $N`
  intact (pinned by `compiler/internal/sql/locking_probe_test.go`).
- Live: tx1 claimed 4 rows via `SKIP LOCKED`; a concurrent tx2 `SKIP
  LOCKED` saw 0 rows; after release tx2 saw 4 again; blocking
  `FOR UPDATE` and `FOR NO KEY UPDATE SKIP LOCKED` each returned 4 rows.
- `NOWAIT` under contention classifies to `sql::query_failed` with code
  `55P03`, both outside and inside a Can transaction (in-txn: query fails,
  `withTransaction` still resolves ok with nothing to commit).
- The rejected outcome was honorably possible (a parse, execution, or
  classification failure would have been recorded as such); the observed
  outcome is admission. Either way, no syntax change: none was needed.
- Portable lease claims (`claim_outbox` lookup-first, P09-B3) stand
  regardless of this outcome and remain the selected claim policy; this
  probe decides expressibility only, never claim policy.

## RETURNING need demonstration — verdict: QUALIFIED NEED

- Ambiguity leg (need): 8 concurrent identical-payload inserts into a
  generated-identity table with no natural unique key refetch as 8
  indistinguishable candidate rows; the writer cannot identify its own
  generated id (`writer_identifiable: false`). Post-write read is
  unexpressible-correctly for this shape without RETURNING (or equivalent).
  Raw `INSERT ... RETURNING id` returns exactly 1 row with the id.
- Keyed control (no-need): 16 concurrent writers using app-supplied unique
  keys, each insert + same-transaction refetch by key, all refetched
  exactly their own row. The endorsed O1 shape is exactly correct here.
- Cost leg (secondary): 60 iterations on loopback — Can txn
  insert-plus-refetch 57.5ms total (~0.96ms/op) vs raw single RETURNING
  13.9ms total (~0.23ms/op), ~4.1x per op (two statements + transaction
  vs one statement). Real factor, modest absolute cost; not the deciding
  argument.
- MySQL leg: `INSERT ... RETURNING` is rejected by MySQL 8.4.11 (syntax
  error 1064). Any admission therefore requires a per-dialect
  row/cardinality/error contract with a MySQL mapping, not a PG-only rule.

## Handoff to F03

- Locking evidence: admitted — claim-via-locking-read is expressible
  through checked descriptors and the live runtime path today.
- RETURNING record: qualified need demonstrated for generated-identity
  shapes without a natural unique key (W6 "generated identities"); no need
  for app-keyed shapes, where same-transaction refetch is exact.
- RETURNING stays rejected until F03 publishes the per-dialect
  row/cardinality/error contract this demo gates; the checker rejection
  (`RETURNING is not admitted`) is pinned unchanged.
- Observation for E (not a defect, no action asked): a `55P03` NOWAIT
  failure inside `withTransaction` surfaces at the query level while the
  transaction still resolves ok when nothing was written; F03 lease-claim
  recipes should treat the query failure as the decision signal.
- No new SELECT expansion and no claim-policy choice are hidden in this
  experiment.
