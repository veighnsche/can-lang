# SQL parser binding comparison (I36)

Bounded research: pick the production PostgreSQL parser binding for Can SQL
descriptors. Both candidates wrap libpg_query, the actual PostgreSQL
parser/scanner; the question is which Go binding to pin, not whether to
trust the grammar. Nothing here is imported by production code: the
comparison lives in its own module, and only the winner entered the root
`go.mod`. The loser's dependency never touched production.

## Candidates

| | CGo (official) | WASM/wazero |
|---|---|---|
| Module | `github.com/pganalyze/pg_query_go/v6` | `github.com/wasilibs/go-pgquery` |
| Revision | `v6.2.2` tag, 2026-01-28, `6a1adb4a` | pseudo `v0.0.0-20260915022521-81f99195012b`, 2026-09-15, `81f99195` |
| libpg_query | `17-6.2.2` | vendored 17.7 C, wasm build |
| PostgreSQL | 17.7 (`PG_VERSION_NUM 170007`) | 17.7 (`PG_VERSION_NUM 170007`) |
| Runtime deps | C toolchain at build, protobuf | wazero 1.12.0, 2488452-byte embedded wasm (no external files) |
| Needs | clang + `CGO_ENABLED=1` to build | WASM threads + exception handling (wazero provides) |
| Licenses | BSD-3-Clause + PostgreSQL | MIT + BSD NOTICE + PostgreSQL-derived + Apache-2.0 (wazero) |

go-pgquery returns pg_query_go/v6 protobuf types, so both harnesses assert
the same shapes. Same PostgreSQL major (17) and minor (7) on both sides.

## Corpus and harnesses

- `corpus/corpus.json`: 24 SQL cases with identical bytes for both sides:
  ordinary, repeated, and multi-digit parameters; single-quoted, escape,
  and dollar-quoted strings; line and nested block comments; multibyte
  UTF-8 before tokens; malformed grammar; unterminated strings; multi,
  empty, comment-only, and semicolon-only inputs; INSERT/UPDATE/DELETE
  with and without RETURNING; subquery and literal LIMIT.
- `corpus/pinned.json`: expected statement spans, parameter spans, token
  counts, and failures, derived from the reference run and hand-checked
  (byte offsets verified against multibyte inputs).
- `cgo/main.go`, `wasm/main.go`: structurally identical mains, each
  importing one candidate. Modes: run the corpus (results JSON plus
  per-case raw parse/scan JSON under `results/raw/`), `--once` for cold
  single-shot timing, `--batch` for memory.
- `compare/main.go`: `--pin` derives pins; `--check` verifies a run;
  `--verdict` diffs two runs case by case, including byte equality of
  the raw parse and scan JSON. Nothing is normalized away.
- `measure.sh`: the full protocol (builds, sizes, cold/warm/memory,
  `CGO_ENABLED=0` behavior). Outputs `results/measure.log`,
  `results/cold.json`; binaries under `results/bin/` are regenerable
  and git-ignored.

## Results (Apple M4, darwin/arm64)

Correctness: **24/24 cases match byte for byte** — raw parse JSON, raw
scan JSON, token spans, statement spans, parameter numbers and spans,
error messages and cursors (`results/verdict.json`). The only divergence
is the Go error wrapper name (`*parser.Error` vs `*pgerror.Error`) with
equal message and cursor.

| Measurement | CGo | WASM |
|---|---|---|
| Cold process parse+scan, P13 SELECT | ~5 ms wall / ~1 ms inner | ~454 ms wall / ~447 ms inner (one-time wazero compile) |
| Warm parse, P13 SELECT | ~14 us, 3.3 KB, 70 allocs | ~45 us, 85 KB, 78 allocs |
| Warm scan | ~2 us, 1.6 KB | ~5 us, 12 KB |
| Harness binary | 13461954 bytes | 20639362 bytes (15403074 with `CGO_ENABLED=0`) |
| Cold build, fresh cache | 6.7 s, needs clang | 7.4 s, pure Go |
| `CGO_ENABLED=0` build | refuses (`undefined: pg_query.Parse`) | builds |
| Peak RSS, 1200 parse+scan batch | ~12 MB | ~300 MB transient (~12 MB retained under `GOGC=20`) |

## Verdict

Pin the **CGo binding**. Jev agreed 3/3 at high confidence (see
`../docs/implementation/evidence/2026-09-21/i36-jev/decision.md`), and
the evidence supports it: identical outputs, ~100x faster cold starts
per SQL-touching compiler invocation, ~25x lower transient memory, a
smaller binary, and a tagged official release. The cost is clang plus
`CGO_ENABLED=1` wherever canlc is built, which plan.md already allows
as a source-build prerequisite; release users still need no C toolchain,
and staged tests execute the CGo-built launcher with compilers absent
from `PATH`.

## Handoff notes for I37

Upstream span quirks, all pinned by `compiler/internal/sql/parser_test.go`:

- `StmtLen` is 0 unless the statement ends with `;`. Derive unterminated
  ends from scanner tokens, never from `Length` alone.
- `StmtLocation` points just past the previous semicolon and can include
  leading whitespace.
- Token `Start`/`End` are byte offsets (verified across multibyte text).
  Failure `Cursor` is a character offset; map it to manifest byte spans.
- `;` scans as `ASCII_59`; comment-only input scans as one `SQL_COMMENT`.
  Empty input scans to zero tokens; all three parse to zero statements.
- Error Go types differ per binding; the adapter normalizes to message
  plus cursor. Classify from structure (counts, spans, token shapes),
  never by matching message substrings.
