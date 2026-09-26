# F03 — C-F RETURNING per-dialect contract (2026-09-26)

Source: R10; F-R10-01/02/04/05; C-F. This is the per-dialect
row/cardinality/error contract that gates the qualified RETURNING
admission demonstrated by F02 (need: generated-identity shapes without a
natural unique key; no need: app-keyed shapes). Live evidence:
[f03-live-report.json](f03-live-report.json),
[f03-report.md](f03-report.md). Checker:
`compiler/internal/sql` (`f03_returning_test.go`).

## 1. Admitted shape (the whole admission)

One shape, no variants:

- Statement: single `INSERT ... RETURNING ...` (any RETURNING list —
  columns, expressions, `*`).
- Cardinality: `one` only.
- Dialects: `postgresql`, `sqlite` only.
- Descriptor: `Total = len(parameters)`, `Limit = 0`, `Returning = true`.
  Downstream invariant: `(cardinality one, limit 0)` holds exactly for
  these descriptors. There is no row-limit site; the exactly-one shape
  comes from a single-row INSERT plus runtime enforcement (§3).

Everything else with a RETURNING clause is rejected by the checker with
a `RETURNING is not admitted ...` diagnostic:

| Shape | Verdict | Reason |
|---|---|---|
| `UPDATE`/`DELETE ... RETURNING`, any cardinality | rejected | no demonstrated need (F02 qualified INSERT only) |
| `INSERT ... RETURNING` under `many`/`optional`/`execute` | rejected | only the exactly-one generated-identity need is demonstrated |
| `INSERT ... RETURNING` on `mysql` | rejected, with mapping hint | MySQL 8.4.11 has no RETURNING clause (server error 1064, pinned live); see §5 |
| `INSERT ... RETURNING` declaring `row_limit_parameter` | rejected | no LIMIT site exists; the declaration would be a lie |
| multi-statement text containing RETURNING | rejected | one statement per descriptor, as before |

A zero `row_limit_parameter` with cardinality `one` and no RETURNING
clause still fails exactly as before, so the 0-limit shape cannot leak
onto SELECT.

## 2. Row contract

- The runtime binds exactly the application parameters — no limit value
  is appended — and decodes the returned rows with the descriptor's row
  record through the shared strict decoder: the RETURNING list must
  project exactly the row record's fields. Extra columns →
  `sql::schema_mismatch` (`extra_column`); missing columns →
  `missing_column`. (Decoder strictness pinned live through the runtime:
  `error_vocab.wide_decode`.)
- Single-row `INSERT ... RETURNING` yields exactly one row with the
  generated identity as a Can `int` (PG `BIGINT GENERATED ... AS
  IDENTITY` arrives as bigint; SQLite `INTEGER PRIMARY KEY
  AUTOINCREMENT` likewise). Pinned live: `returning_native.single_rows`
  = 1, `single_id_type` = `bigint`; `sqlite.single_rows` = 1.
- Multi-row `INSERT ... RETURNING` (multi-row VALUES, `INSERT ...
  SELECT`) natively yields N rows (pinned: 3 for 3 rows on PG and
  SQLite). The checker admits the shape structurally; the runtime
  enforces exactly-one (§3). Authors wanting N rows must not use this
  shape — it is the generated-identity shape, not a bulk-read shape.
- `INSERT ... ON CONFLICT DO NOTHING RETURNING` yields 1 row on insert,
  0 rows on conflict-skip (pinned live). Zero rows → `sql::row_missing`
  (§3): honest, and a deliberate discouragement of this combination —
  use the lookup-first recipe instead
  ([recipe](f03-lookup-first-recipe.md)).

## 3. Cardinality contract (runtime enforcement, E-owned)

The RETURNING execution path enforces `query_one` semantics over the
native rows, reusing the existing failure vocabulary (each pinned live
through the runtime on PG in `error_vocab`):

| Native rows | Outcome |
|---|---|
| 1 | decode the row (§2) |
| 0 | `sql::row_missing` (`query` = descriptor name) |
| >1 | `sql::row_count` (`query` = descriptor name, `actual` = 2 — the existing over-one encoding, cf. `two_row_one`) |

## 4. Error contract

- Conflicting `INSERT ... RETURNING` surfaces the native unique
  violation: PG reports SQLSTATE `23505` with the constraint name
  (pinned raw: `returning_native.conflict`), which the existing
  classifier maps to `sql::constraint_failed` with the constraint name
  (pinned through the runtime: `error_vocab.duplicate_insert`). No new
  failure, no new mapping.
- A statement failure inside a PG transaction aborts the transaction:
  every later statement in that transaction fails with `25P02`,
  surfacing as `sql::query_failed` (`operation`, `code = "25P02"`) at
  the query level (pinned: `aborted_txn.followup_outcome`), while the
  transaction itself still resolves `ok` with nothing to commit
  (`aborted_txn.txn_outcome`; extends the F02 NOWAIT observation from
  lock contention to statement failure). Recipes must treat the query
  failure as the decision signal and never ignore it: anything the
  callback wrote after the failure is silently lost. A fresh replay on
  the same pool then succeeds (`aborted_txn.replay_ok`).
- SQLite and MySQL legs carry no new error semantics: SQLite reuses the
  shared classifier; MySQL has no RETURNING path at all (§5).

## 5. MySQL mapping (no RETURNING on MySQL)

MySQL 8.4.11 rejects `INSERT ... RETURNING` with syntax error 1064
(re-pinned live: `mysql.insert_returning`). Generated-identity writes
on MySQL use two ordinary descriptors in one transaction:

1. `INSERT INTO t (payload) VALUES (?)`, cardinality `execute`;
2. `SELECT id, payload FROM t WHERE id = LAST_INSERT_ID() LIMIT ?`,
   cardinality `one`.

`LAST_INSERT_ID()` is connection-scoped, so the refetch is exact under
concurrency as long as both statements share one transaction (one
connection) — never split them across pool checkouts. The checker needs
no change for this shape: the refetch is an ordinary checked SELECT
(pinned by `TestF03MySQLMappingSelect`). Qualified live through the
runtime: 8 concurrent writers, each refetched exactly its own row, 8
distinct ids (`mysql.mapping`).

## 6. Implementation handoff (E + coordinator routing)

F03 implements the checker side only. End-to-end execution of admitted
descriptors needs, in order:

1. Manifest (`compiler/internal/project/manifest_sql.go`, not F-owned):
   allow cardinality `one` with `row_limit_parameter` 0/absent; the
   checker then requires `INSERT ... RETURNING` (any other shape fails
   as in §1). No other manifest rule changes.
2. Emit: no change required — `Limit`/`Total`/`Kind` already flow into
   the descriptor table. Optional hardening: emit the checker's
   `Returning` flag explicitly rather than deriving it downstream.
3. Runtime (`runtime/platform/sql/`, E-owned):
   - `descriptor.ts`: allow `(cardinality "one", limit 0)`; require an
     INSERT kind (`InsertStmt`/`insert_statement`) for that pair (or the
     explicit flag if (2) adds it). All other (cardinality, limit)
     pairs keep today's validation.
   - `pool.ts` shared core: a RETURNING path that binds exactly the
     application parameters (no appended limit) and enforces §3 over
     the native rows through the shared decoder (§2) and the existing
     `row_missing`/`row_count`/`schema_mismatch` failures. Pool and
     transaction twins, as today.
   - No catalogue failure additions, no classifier changes (§4 reuses
     every mapping).
4. Call-site checking (`compiler/internal/check/sql.go`, not F-owned):
   no change — `query_one` already wants cardinality `one`, which is
   what admitted descriptors carry.

## 7. Explicitly not admitted

No new column kinds, no savepoints, no migrations, no `RETURNING` on
`UPDATE`/`DELETE`, no `many`/`optional` RETURNING, no MySQL RETURNING
syntax. Encoding insufficiency or a savepoint need returns to
preparation via the coordinator — never automatic syntax work.
