# I36 acceptance — SQL parser binding comparison and pin

Closed 2026-09-21. Both the official CGo binding and the WASM/wazero
binding were measured on the same PostgreSQL 17.7 corpus; outputs match
byte for byte on all 24 cases, and the evidence plus three Jev
consultations select the CGo binding. Production adopts exactly
`pg_query_go/v6 v6.2.2` behind a small compiler-owned adapter; the WASM
dependency never entered production.

Baseline `b6f361d` (I34) plus the I36 worktree. Apple M4 darwin/arm64,
Go 1.27.1, Apple clang 21, TypeScript untouched by this task.

## Candidates and revisions

- CGo: `github.com/pganalyze/pg_query_go/v6 v6.2.2` (tag, 2026-01-28,
  commit `6a1adb4a`), libpg_query `17-6.2.2`, PostgreSQL 17.7
  (`PG_VERSION_NUM 170007`). Official binding, BSD-3-Clause plus
  PostgreSQL License.
- WASM: `github.com/wasilibs/go-pgquery
  v0.0.0-20260915022521-81f99195012b` (pseudo-version, no tags,
  2026-09-15) with wazero 1.12.0 and a 2488452-byte embedded WebAssembly
  module built from the same 17.7 sources; returns pg_query_go/v6
  protobuf types. MIT plus BSD NOTICE plus PostgreSQL-derived bytes
  plus Apache-2.0 wazero; needs WASM threads and exception handling.

Both expose `Parse`/`Scan` with statement spans, full scanner token
streams with byte offsets, and message-plus-cursor failures.

## Comparison method

`research/sql-binding/` is a separate throwaway module (own `go.mod`,
never imported by production) with a shared 24-case corpus
(`corpus/corpus.json`), structurally identical per-candidate harnesses
(`cgo/`, `wasm/`), a differ that normalizes nothing (`compare/`), pinned
expectations (`corpus/pinned.json`, hand-checked including multibyte
byte offsets), warmed benchmarks, and the scripted protocol
(`measure.sh`). Raw per-case parse/scan JSON for both sides is committed
under `results/raw/`; `results/verdict.json`, `results/cold.json`, and
`results/measure.log` record the runs.

## Results

Correctness: 24/24 cases byte-identical across raw parse JSON, raw scan
JSON, token spans, statement spans, parameter numbers and spans, and
error message plus cursor. Multi-statement and malformed inputs reject
identically; the lone divergence is the Go error wrapper name. Pins hold
for both runs.

Cost (second measurement run; first run agreed):

| Measurement | CGo | WASM |
|---|---|---|
| Cold parse+scan, P13 SELECT | ~5 ms wall / ~1 ms inner | ~454 ms wall / ~447 ms inner (one-time compile) |
| Warm parse | ~14 us, 3.3 KB, 70 allocs | ~45 us, 85 KB, 78 allocs |
| Warm scan | ~2 us, 1.6 KB | ~5 us, 12 KB |
| Harness binary | 13461954 bytes | 20639362 (15403074 at `CGO_ENABLED=0`) |
| Cold build, fresh cache | 6.7 s, needs clang | 7.4 s, pure Go |
| `CGO_ENABLED=0` build | refuses | builds |
| Peak RSS, 1200-pair batch | ~12 MB | ~300 MB transient (~12 MB retained under `GOGC=20`) |

## Choice

Jev advised `cgo_official` 3/3 at 0.97/0.98/0.91, separate Parse/Scan
entry points 3/3 (soft), and verbatim message-plus-cursor failures 2-1
with the dissent investigated (see `../i36-jev/decision.md`). I37
classifies from structure, never message substrings.

## Production adoption

- `compiler/internal/sql/parser.go`: `Version` (asserts major 17, reports
  170007), `Parse` (statement spans plus node tags), `Scan` (token spans
  plus token/keyword names), verbatim `Failure{Message, Cursor}`. All
  spans are validated against the input; no protobuf or backend AST
  crosses the boundary. `parser_test.go` pins version, P13 spans,
  hidden-parameter cases, multibyte byte offsets, verbatim failures,
  multi-statement spans (including the upstream length-0 quirk for
  unterminated statements), empty inputs, statement kinds, keyword
  kinds, and deterministic invalid-UTF-8 behavior.
- Root `go.mod`/`go.sum`: `pg_query_go/v6 v6.2.2` plus
  `protobuf v1.36.12` (explicitly aligned with the measured harness).
- `distribution/build.go`: launcher builds with `CGO_ENABLED=1`
  (developer prerequisite per plan.md); the binary links only
  `libSystem`/`libresolv`.
- `distribution/notices/`: BSD, PostgreSQL, and protobuf license texts
  plus `sql-binding.lock.json` provenance (revisions, hashes, PG
  version, rejected candidate and rationale). Binding provenance rides
  the existing launcher-hash compiler input in every build ID.

## Verification

- `go test -count=1 ./compiler/internal/sql/`: 10 tests pass.
- `go test -count=1 ./distribution ./compiler ./compiler/internal/...`:
  all pass (re-run after adoption; includes the CGo-built paths).
- Release candidate: `distbuild` produces a CGo-linked canlc that
  asserts (12 rows) and builds the asset fixture under `sandbox-exec`
  with `PATH=/nonexistent`; `otool -L` shows platform libraries only.
- Full `tests/integration/` suite with `CAN_BUN`, `CAN_BUN_ARCHIVE`,
  `CAN_TSC`: `ok ... 135.793s` after the `CGO_ENABLED` flip, every
  staged bundle building and running the CGo-linked launcher without a
  toolchain on `PATH`.
- `catalogue-check`, `modcheck`, `gramcheck`: unaffected (no catalogue
  or grammar change); re-run at commit time.
- `gofmt`/`go vet` clean on all touched packages including the research
  module.

## Limits and handoff

- `StmtLen` is 0 for statements without a trailing `;`, and
  `StmtLocation` can include leading whitespace; I37 derives ends from
  scanner tokens. Failure cursors are character offsets; map to manifest
  byte spans in I37. Comment-only input scans as one `SQL_COMMENT`.
- The research module stays as evidence; it is excluded from root gates
  by Go's nested-module rule and must never be imported by production.
