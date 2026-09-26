# F03 — qualified relational slice and encoding guidance (2026-09-26)

Source: R10; F-R10-01/02/04/05; C-F. Prerequisite F02 done on main
(d3e51ded; `../f02/f02-report.md`). Machine record:
[f03-live-report.json](f03-live-report.json). Live services: PG 17.11
(127.0.0.1:55433), MySQL 8.4.11 (127.0.0.1:3307), SQLite 3.54.0
(in-memory, Bun 1.4.2); lane run database `can_f03_run1` (PG + MySQL,
created via `db mkdb`). Env credentials only, never printed.

## What F03 delivered

- Qualified RETURNING admission in the checker
  (`compiler/internal/sql`, `f03_returning_test.go`): exactly
  `INSERT ... RETURNING` with cardinality `one` and no row-limit site
  on `postgresql`/`sqlite`. `UPDATE`/`DELETE ... RETURNING`,
  `many`/`optional`/`execute` RETURNING, any RETURNING on `mysql`, and
  row-limit/coverage incoherence stay rejected with the pinned
  `RETURNING is not admitted ...` wording (corpus category intact).
- The gating per-dialect row/cardinality/error contract with the MySQL
  mapping: [f03-returning-contract.md](f03-returning-contract.md).
- The blessed encoding guide (minor-unit money, ms-epoch ints incl.
  nullable audit instants, opaque TEXT + codecs):
  [f03-encoding-guide.md](f03-encoding-guide.md). No insufficiency
  found; no new column kinds.
- The lookup-first PG conflict recipe + operator-DDL recipe:
  [f03-lookup-first-recipe.md](f03-lookup-first-recipe.md). No
  savepoint need found; savepoints stay excluded.
- Live qualification (`tests/integration/sql_f03_test.go` +
  `testdata/sql/f03-live-driver.ts`): native RETURNING rows per
  dialect (1 / N / 0 shapes), the runtime cardinality/error vocabulary
  the RETURNING path reuses, PG aborted-transaction + replay,
  8-writer lookup-first decide, encoding roundtrips incl. int64 edges,
  MySQL 1064 rejection + 8-writer `LAST_INSERT_ID` mapping, SQLite
  native RETURNING.

## Validation summary

- `go test ./compiler/internal/sql/` — pass (new F03 tests + corpus +
  F02 pins + existing suites).
- `go test ./compiler/internal/check/ ./compiler/internal/project/
  ./compiler/internal/emit/` — pass (no regressions from the checker
  change; F02's `execute`-RETURNING rejections still hold).
- `go test ./tests/integration/ -run TestF03LiveRelationalSlice
  -count=3` — pass (PG + MySQL + SQLite legs, distinct run DB).
- `gofmt`/`go vet` on touched Go files — clean.
- No `runtime/` or E files touched; no manifest/emit/check changes (all
  handed off, see below). No example sources touched (C-owned).

## Handoffs (to coordinator)

1. **To E (runtime + catalogue): RETURNING execution.** F03 specifies
   but does not implement the runtime path (contract §6): manifest
   `row_limit 0` allowance for `one` (needs coordinator routing —
   `project/` is not F-owned), `descriptor.ts` `(one, 0)` validation,
   and the `pool.ts` RETURNING execution path (bind app params only,
   enforce §3 through the shared decoder). No catalogue additions, no
   classifier changes.
2. **To F04: C-F descriptor/row contracts + operator-DDL recipe.** The
   three F03 docs above are the F04 input, alongside the F02 handoff.
3. **To C (via coordinator): webhook `decide_receive` lookup-first
   port.** Exact spec in the recipe §4 (restructure + asserts + README
   bullet, U06 gate).
4. **To E (defect, found during F03): concurrent `withTransaction`
   burst on a cold pool fails with `resource_state` before callbacks
   enter.** Dialect-independent (PG + MySQL), timing-dependent; raw
   Bun bursts are sound (8/8), so the race is in the runtime wrapper
   (suspected late-granted `begin` losing ambient owner context — not
   proven). F03 legs warm pools with serial transactions first (noted
   in-driver); generated apps will hit this without an E fix.

## Scope retained

New column kinds, savepoints, and migrations remain excluded; no
preparation return was needed (encodings sufficient, no savepoint
need). Unnecessary RETURNING stays rejected per §1 of the contract.
