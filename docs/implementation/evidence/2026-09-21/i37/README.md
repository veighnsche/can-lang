# I37 acceptance — static SQL descriptors and native bindings

Closed 2026-09-21. Manifest SQL descriptors are checked once per compile
against concrete Can records and the pinned upstream parser, carried in
typed IR, emitted as a provenance-checked table, and executed as native
Bun tagged-template bindings. No pool opens at validate/compile time, no
values are concatenated into SQL, and no second SQL scanner exists in
emitted code.

Baseline `0121303` (I36) plus the I37 worktree. Apple M4 (Mac16,12)
darwin/arm64, macOS 27.0, Go 1.27.1, Bun 1.4.2 (`744846f84`, pinned
archive sha256 `90987a3a…1d12f` re-acquired and hash-verified for this
run), TypeScript 7.0.2 with bun-types 1.4.2 and @types/node 24.10.1,
libpg_query 17.7 (`PG_VERSION_NUM 170007`) via pg_query_go/v6 v6.2.2,
disposable PostgreSQL 17.11 (Debian 17.11-1.pgdg13+2, aarch64) over
loopback TCP.

## Design consultations

[i37-jev](../i37-jev/decision.md): three fresh Jev consultations (all
prose rewritten, same facts/options). Unanimous and strong for preserving
trailing trivia (0.95, 0.95, 0.99): a descriptor is valid with exactly
one RawStmt while leading/trailing semicolons, comments, and whitespace
persist literally in the emitted segments. Limit exclusivity split
without conviction (shared 0.61 / once 0.61 / shared 0.50-vs-0.48) and
was overridden on the merits: P12 calls the top-level LIMIT the distinct
parameter after the application parameters, and a shared number would let
a call-site LIMIT value silently fill a filter position. The row-limit
number therefore appears in exactly one PARAM token, the top-level
LimitCount. Judgments are advice; the checks below are the proof.

## Changes

- `compiler/internal/sql/parser.go`: `Statement` gains top-level-only
  shape (`HasLimit`, `LimitParam`, `Returning`) read from the single
  statement node; subquery limits stay invisible by construction.
- `compiler/internal/sql/descriptors.go` **new**: `CheckDescriptor`
  feeds exact statement bytes to the I36 parser/scanner, requires one
  approved statement, contiguous `$1..$M` with the exclusive trailing
  LIMIT number for row-returning shapes, rejects unbounded SELECT and
  mutation RETURNING, and tiles the input into alternating literal/param
  segments from scanner byte offsets (quote/comment `$N` stays literal).
- `compiler/internal/check/sql.go` **new**: resolves parameter/row names
  to concrete ordinary records in the owning project, requires manifest
  parameter order to match record field order exactly, and admits only
  the SQL scalar cover (bool, int, float, str, bytes buffer, one option
  layer) for parameter and row fields.
- `compiler/internal/ir/sql.go` **new**, `check/program.go`,
  `emit/program.go`, `emit/sql.go` **new**: checked descriptors ride
  typed IR (owner, concrete identities, parameters) into the emitted
  `$canSQL` table. Static names never become runtime lookup strings;
  emit tests assert the table carries no query-execution surface and no
  `.unsafe(`/`.simple(`.
- `runtime/platform/sql-descriptor.ts` **new** (+ `modules.json`):
  compiler-owned `createSQLDescriptors` builds inert frozen values plus
  `template()`, which expands one prepared value per parameter number
  (repeated `$1` repeats structurally) and throws on unknown names,
  forged values, arity breaks, and parser-version mismatch.
- Fixtures `compiler/testdata/current/sql/descriptors.can`;
  `runtime/test/sql-descriptor.test.ts`; staged
  `tests/integration/sql_descriptors_test.go` with seed
  `tests/integration/testdata/sql/descriptors.sql` and the test-only
  live driver `descriptors-driver.ts` (below).

No Can-callable query intrinsic is admitted in I37: the catalogue
declares the I35 `sql::query_*` signatures but check/emit carry no
specialization, so descriptors stay inert under the C8 manifest case and
no stub silently accepts unsupported calls. Callable/generic call-site
binding resolves owner-local names against this table in I35.

## Verification

- `go test -count=1 ./compiler/...`: all packages pass (includes new
  `check/sql_test.go`, `emit/sql_test.go`, `descriptors_test.go` and the
  extended parser span/shape tests).
- `bun test runtime/test/`: 516 pass, 0 fail, 41,748 expectations
  (92 files); focused `sql-descriptor.test.ts`: 5 pass, 28 expectations
  (declare/template, repeat sharing, zero-param, forged/arity throws,
  corrupt-table refusal).
- Strict `tsc --noEmit --strict` (TS 7.0.2): the runtime module, its
  test, the live driver, and the staged generated `entry.ts` all clean.
- `cataloguegen --check`, `modcheck`, `gramcheck`: all pass.
- `gofmt`/`go vet` clean on every touched package.
- `TestCurrentSQLDescriptors`: staged bundle asserts the Can fixture,
  runs, rebuilds with identical build identity, rebuilds identically
  after relocation, and rejects 12 staged negatives: unbounded SELECT,
  RETURNING, 2 statements, gapped `$2`, undeclared `$3`, shared LIMIT
  (`appears 2 times`), literal LIMIT, field-order mismatch, non-scalar
  parameter, undeclared type, LIMIT on execute, and `SELECT FROM WHERE`
  syntax failure.
- `TestCurrentSQLDescriptorsLive`: 16/16 checks against the disposable
  PostgreSQL 17.11 via the compiled `$canSQL.template()` path and the
  pinned Bun.SQL tag — LIKE/quote/Unicode binding, `' OR '1'='1` and
  stacked `DROP TABLE` payloads matching 0 rows with the table intact,
  repeated-`$1` expansion, one/missing id lookup, INSERT count 1 with
  quote round-trip, comment `$9` literal, and LIMIT bound as a value.
  The driver drops its table in a `finally` (verified: no relations
  remain) and the Go test redacts the URL/password from all output.
- Full `tests/integration/` with `CAN_BUN`, `CAN_BUN_ARCHIVE`,
  `CAN_TSC`, `CAN_TEST_POSTGRES_URL`: `ok … 166.722s`.
- `make bundle` + `otool -L`: CGo-linked launcher, platform libraries
  only; staged `canlc` runs under `sandbox-exec` with `PATH=/nonexistent`
  (the live driver alone runs outside the network-deny sandbox because
  real parameterization needs loopback TCP; it is test-only and DSN-bound).

## Limits and handoff

- `check/sql.go:sqlScalar` duplicates the I35 schema cover by design;
  I35 must move it into the shared projection and keep the two identical
  until that extraction lands.
- Value validation before native launch, pool/row/close behavior, LIMIT
  2 / max_rows+1 binding, and error classification are I35's scope; I37
  proves the template/binding metadata I35 consumes.
- Live coverage uses one disposable database and one table; concurrent
  pools, transactions, and fault injection arrive with I35/I38.
