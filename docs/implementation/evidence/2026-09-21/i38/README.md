# I38 acceptance — transaction decisions and uncertain commit

Closed 2026-09-21. Checked `with_transaction<T>` calls bind to one
concrete result type plus nominal commit/rollback leaf identities, and
emit per-specialization binders into a scoped transaction runtime over
one native `begin`: exactly one live attempt per dynamic extent, a
private owner-scoped handle, drain-before-commit, rollback through a
private identity-checked sentinel, and commit-unknown classification
when the native layer rejects after a commit decision. In-transaction
queries reuse the I35 shared core (validation, classification,
decoding) with the `sql-tx` resource kind — never a parallel
subsystem.

Baseline `3c67d2a` (I35) plus the I38 worktree. Apple M4 (Mac16,12)
darwin/arm64, Go 1.27.1, Bun 1.4.2 (pinned archive sha256
`90987a3a…6be1`, hash-verified by the distribution builder),
TypeScript 7.0.2, disposable PostgreSQL 17.11 (Debian
17.11-1.pgdg13+2, aarch64) over loopback TCP.

## Design consultations

[i38-jev](../i38-jev/decision.md): three fresh Jev consultations (all
prose rewritten, same facts/options). Split on the begin-failure error
(connection 0.44, review 0.60, connection 0.68), decided for
connection_begin on module consistency: a refused begin is the same
native event as a refused query, already named by the begin phase.
Unanimous abstention on the commit-unknown ID (review_needed 0.81,
0.53, 0.85), decided on the merits for structured_sequence: a
pool-resource-derived identity plus a per-pool attempt counter
correlates attempts without carrying driver payload or credential.
Majority for unbounded_drain (0.87, 0.66, review_needed 0.62): the
operation offers no timeout input, so a bound would invent a deadline.
Judgments are advice; the checks below are the proof.

## Implementation

- `check/transaction.go`, `check/program.go`, `check/sql.go`,
  `check/http.go`: `with_transaction<T>` specialization (T must be
  data; callback contract exactly `(sql::transaction) ->
  sql::decision<T>`; outer contract returns T with the catalogue
  connection/transaction/commit errors), in-transaction query
  operations through the shared descriptor/cardinality checks with a
  handle-first call shape, and ingress-only scope-request admission
  for the handle type.
- `emit/sql.go`, `emit/program.go`, `emit/expressions.go`:
  per-specialization `$canSQLTransactionN.run` binders carrying the
  nominal commit/rollback leaves, transaction query receivers on
  `$canTransactions`, `$canSQLTransactionN` imports into package and
  assertion modules (a missing import was the staged-assert root
  cause below), and the `$canTransactions` state declaration,
  construction, and runtime import.
- `runtime/platform/transaction.ts`: `withTransaction` (fixture
  denial, leaf/callback validation, ALS nesting guard, pool lease,
  per-pool attempt counter, `poolId-attempt` IDs), commit/rollback
  decision classification by nominal leaf identity, verbatim primary
  completions through `TxPrimary`, rollback through the private
  `TxRollback` sentinel, commit-unknown after a commit decision with
  a sanitized payload, refused/closed begin mapped to
  connection_failed with the begin/callback phase, foreign throws
  propagated, and scope-managed handle registration so drain closes
  the handle without a cleanup mark.
- `runtime/callable.ts`, `runtime/assert/runner.ts`: fixture
  argument lists compare element-wise so callable arguments reach
  receipt (target plus captures) equality; whole-array deep equality
  would reject the same `callable name` named by a call and its
  fixture row.
- `compiler/testdata/current/sql/transactions.can`: commit, rollback,
  fetch, and nested fixtures with 12 passing assertions.

## Verification

- `go test -count=1 ./...`: all packages pass (includes the new
  `check/transaction_test.go` and the extended check/emit SQL tests).
- `bun test runtime/test/`: 850 pass, 0 fail, 63,285 expectations
  (142 files); focused `transaction.test.ts`: 15 pass, 98
  expectations over a patched `Bun.SQL` fake with `begin` (commit and
  rollback values, shared core validation/classification through the
  handle, handle death with its scope, refused begin phase,
  transactional begin failures, foreign-throw propagation, verbatim
  nested scope failure, verbatim primary domain failures, safe
  incrementing commit-unknown IDs, foreign post-commit rejection,
  drain-before-commit ordering, compiler-shape violations, and
  fixture-boundary denial). Every owned-root body asserts its
  completion so no expectation passes vacuously. The new
  `assertions.test.ts` case pins callable receipt equality inside
  argument lists (same target/captures equal; differing target,
  captures, foreign functions, and lengths unequal).
- Strict `tsc --noEmit --strict` (TS 7.0.2): the transaction module,
  both touched tests, the live driver, and the staged generated
  `entry.ts` clean.
- `cataloguegen --check`, `modcheck`, `gramcheck`: all pass.
- `gofmt`/`go vet` clean on every touched package (4 remaining
  `gofmt` findings are pre-existing in untouched files).
- `TestCurrentSQLTransactions`: staged bundle asserts the Can fixture
  (12/12), rebuilds with identical build identity, rebuilds
  identically after relocation, and rejects 4 staged negatives:
  cardinality mismatch, undeclared parameter type, row-type mismatch,
  and undeclared descriptor. Debugging record: the first staged run
  failed 5/12 with outcome mismatch and no fixture violation; tracing
  showed the inner call site never executed because `$canCallContext`
  argument evaluation threw `ReferenceError` on the unimported
  `$canSQLTransaction0` binder — fixed by emitting the binder imports.
- `TestCurrentSQLTransactionsLive`: 13/13 checks against the
  disposable PostgreSQL 17.11 through the compiled program CLI, with
  the credential in an fd-3 snapshot and never in the environment:
  fetch hit and rolled-back miss, commit landing at identity 7 with a
  duplicate constraint rolling back to 0, conflicting and rolled-back
  inserts consuming identities 8 and 9, Yara landing at 10, invalid
  JSON fault, nested attempt failing closed through the fallback arm
  (`-1`, never the inner `7`), and refused-port connection failure.
  The seed drops stale probe schema first, truncates with identity
  restart, and drops the table in teardown; the Go test redacts the
  URL/password from all output.
- I35/I37 regression: the full suite run includes
  `TestCurrentSQLQueries`, `TestCurrentSQLQueriesLive`,
  `TestCurrentSQLDescriptors`, and
  `TestCurrentSQLDescriptorsLive`, all passing.
