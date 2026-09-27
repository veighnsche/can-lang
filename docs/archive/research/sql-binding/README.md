# SQL parser binding comparison (I36)

Historical I36 decision: select the PostgreSQL parser binding for Can SQL
descriptors. Both candidates wrap libpg_query, the actual PostgreSQL
parser/scanner; the question is which Go binding to pin, not whether to
trust the grammar. The retired comparison ran in its own module, never imported by
production code. Only the winner entered the root `go.mod`; the loser's
dependency never touched production.

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

go-pgquery returned pg_query_go/v6 protobuf types, so both harnesses checked
the same shapes. Same PostgreSQL major (17) and minor (7) on both sides.

## Historical method

The retired standalone Go module compared 24 identical SQL inputs with
both bindings. Cases covered parameter numbers, quotes, comments, UTF-8
byte offsets, malformed and empty inputs, statement counts, DML, and
LIMIT. The comparison checked raw parse/scan JSON without normalization,
plus hand-checked statement and parameter spans. Cold, warm, build,
binary-size, and memory measurements informed the choice.

The executable harnesses, self-tests, shell runner, dependency files,
private corpus, and detailed run outputs were removed after the decision.
Git history retains that evidence. This directory is a historical record,
not a runnable experiment or a current validation input.

## Results (Apple M4, darwin/arm64)

Correctness: **24/24 cases match byte for byte** — raw parse JSON, raw
scan JSON, token spans, statement spans, parameter numbers and spans,
error messages and cursors ([retained verdict](results/verdict.json)). The only divergence
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
[consultations](../../../implementation/evidence/2026-09-21/i36-jev/decision.md)), and
the evidence supports it: identical outputs, ~100x faster cold starts
per SQL-touching compiler invocation, ~25x lower transient memory, a
smaller binary, and a tagged official release. The cost is clang plus
`CGO_ENABLED=1` wherever canlc is built, which plan.md already allows
as a source-build prerequisite; release users still need no C toolchain,
and staged tests execute the CGo-built launcher with compilers absent
from `PATH`.

## Retirement audit

The comparison served its one-time dependency selection; keeping its
alternate parser, nested module, and experiment-only tests adds obsolete
maintenance work. A bounded repository reference audit found no imports,
execution paths, or fixture consumers in current compiler, runtime,
tools, or tests. The distribution SQL binding provenance record cites this
README for the rejected candidate's rationale; it remains valid.
Implementation planning and frozen I36 evidence describe the original
experiment historically. No current gate or test may depend on archive
files, as stated in the [archive policy](../../README.md).

The original [verdict](results/verdict.json) preserves the 24 case
comparisons and matching hashes. [Cold timing results](results/cold.json)
preserve the measured startup difference (values in milliseconds). Other
measurements above are the historical summary, not newly rerun claims.
Production parser contracts and their current tests remain outside this
archive. No builds, tests, benchmarks, or experiment reruns accompanied
this retirement.
