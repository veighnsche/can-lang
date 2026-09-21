# I35 acceptance — typed Bun.SQL pools and rows

Closed 2026-09-21. Checked I37 descriptors are bound to concrete Can
parameter/row records, emitted with shared scalar schemas, and executed
through a native `Bun.SQL` pool adapter: credentialed open with proven
establishment, pre-launch parameter validation, finite native failure
classification, immutable row decoding, exact LIMIT bounds, and
owner-registered close with lease drain. Values travel only as bound
template parameters; the string-call form, unsafe helpers, and second
lexers are never used.

Baseline `7794e5c` (I37) plus the I35 worktree. Apple M4 (Mac16,12)
darwin/arm64, macOS 27.0, Go 1.27.1, Bun 1.4.2 (`744846f84`, pinned
archive sha256 `90987a3a…16d7` re-acquired and hash-verified for this
run), TypeScript 7.0.2 with bun-types 1.4.2 and @types/node 24.10.1,
libpg_query 17.7 (`PG_VERSION_NUM 170007`) via pg_query_go/v6 v6.2.2,
disposable PostgreSQL 17.11 (Debian 17.11-1.pgdg13+2, aarch64) over
loopback TCP.

## Design consultations

[i35-jev](../i35-jev/decision.md): three fresh Jev consultations (all
prose rewritten, same facts/options). Unanimous for awaiting the native
connect handshake in pool_open (0.78, 0.84, 0.90): construction alone
cannot prove establishment, and a probe query wastes a round trip.
Unanimous and strong for reporting defective max_rows as
unsupported_value before launch (0.99, 0.99, 0.98) while row_limit
fires only on observed overflow: argument defects and observed counts
stay distinct. Unanimous but weak for classifying only PostgresError
(0.46, 0.74, 0.56); the deciding merit is that a foreign throw is a
driver defect, not a query outcome, and masking it would destroy test
visibility. Judgments are advice; the checks below are the proof.

## Implementation

- `compiler/internal/types/sql_schema.go`: finite SQL scalar/option
  shape derivation (bool, signed-64 int, finite float, scalar-valid
  str, owned bytes, one optional layer); arrays/JSON/timestamps and
  arbitrary variants reject at the source boundary.
- `check/sql.go`, `check/program.go`, `check/completions.go`,
  `check/specialize.go`, `check/infer.go`: query call-site checking
  against the finished I37 descriptor table (name resolution,
  parameter/row record identity, one/optional/many/execute
  cardinality match).
- `emit/sql.go`, `emit/program.go`, `emit/regions.go`, `ir/sql.go`,
  `ir/regions.go`: pool open/close function mappings plus per-site
  `run:` shapes carrying descriptor, shared plan, pool, and params
  into `$canSQLPools.queryOne/queryOptional/queryRows/execute`.
- `runtime/platform/sql.ts`: pool open (exact-name credential read
  once, config validation, `adapter: "postgres"`, `bigint: true`,
  awaited connect), pre-launch parameter encoding (int64 range,
  surrogate rejection, option shapes, bytes copies), native failure
  classification (PostgresError only; connection/constraint/query
  with sanitized fields), immutable row decoding (missing/extra
  columns, null rules, unsafe integers, bigint range, finite floats,
  copied binary, path-specific mismatch), LIMIT 2 for one/optional,
  max_rows+1 for many, affected-count validation for execute, and
  owner-registered close (deny new owners, drain leases, native
  close under the remaining deadline; timeout stays closing).
- `compiler/testdata/current/sql/queries.can`: open/query/close
  fixture with 9 passing assertions covering all five operations,
  the binary/option cover row, and every fault arm.

## Verification

- `go test -count=1 ./compiler/...`: all packages pass (includes new
  `types/sql_schema_test.go` and the extended check/emit SQL tests).
- `bun test runtime/test/`: 278 pass, 0 fail, 20,958 expectations
  (47 files); focused `sql.test.ts`: 20 pass, 186 expectations over
  a patched `Bun.SQL` fake (credential read once, refused connect
  cleanup, fixture-boundary denial, pre-launch validation with zero
  launches, option/bytes encoding, injection-as-value, decode rules,
  LIMIT bindings, exact bounds, affected counts, sanitized
  classification, foreign-throw propagation, close validation, and
  close-during-lease with timeout-then-drain). Every owned-root body
  asserts its completion so no expectation passes vacuously.
- Strict `tsc --noEmit --strict` (TS 7.0.2): the runtime module, its
  test, the live driver, and the staged generated `entry.ts` clean.
- `cataloguegen --check`, `modcheck`, `gramcheck`: all pass.
- `gofmt`/`go vet` clean on every touched package (4 remaining
  `gofmt` findings are pre-existing in untouched files).
- `TestCurrentSQLQueries`: staged bundle asserts the Can fixture
  (9/9), rebuilds with identical build identity, rebuilds
  identically after relocation, and rejects 4 staged negatives:
  cardinality mismatch, undeclared parameter type, row-type
  mismatch, and undeclared descriptor.
- `TestCurrentSQLQueriesLive`: 16/16 checks against the disposable
  PostgreSQL 17.11 through the compiled program CLI, with the
  credential in an fd-3 snapshot and never in the environment:
  close round trip, by-id row and missing fault, optional
  some/none/row_count, bounded rows and row_limit, execute count 1
  and duplicate constraint failure, binary/option cover rows with
  null and empty payloads, hostile input binding with the table
  intact, and refused-port connection failure. The seed drops stale
  probe schema first, truncates with identity restart, and drops
  both tables in teardown (verified: no relations remain); the Go
  test redacts the URL/password from all output.
- I37 regression: `TestCurrentSQLDescriptors` and
  `TestCurrentSQLDescriptorsLive` still pass (16/16 live checks).
