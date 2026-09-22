# I44 acceptance — retired predecessor production paths

Closed 2026-09-21. The predecessor toolchain is deleted: 304 files
and 87,641 lines removed, leaving a launcher of five production
files, eleven commands, four explicit retired stubs, and no
Z3/extern/old-syntax dispatch anywhere in Go sources. Retired std
and sketch sources are gone per the I43 deletion list; the
committed-golden tscheck gate is deleted with its subjects and I45
recreates TypeScript gating around fresh emit.

Baseline `154eca1` (I43) plus the I44 worktree. Apple M4 (Mac16,12)
darwin/arm64, macOS 27.0, Go 1.27.1, Bun 1.4.2 (`744846f84`, pinned
archive sha256 `90987a3a…6be1`, hash-verified), TypeScript 7.0.2 with
`@types/bun` 1.4.2 and `@types/node` 24.13.6, Playwright 1.55.1 with
Chromium 140.0.7339.186, disposable PostgreSQL 17.11 over loopback.

## Design consultations

[i44-jev](../i44-jev/decision.md): three fresh Jev consultations (all
prose rewritten, same facts/options). Unanimous and strong for
both_gates (0.96, 0.99, 0.97): a static machine check plus behavior
probes. Unanimous in direction with strengthening confidence for
explicit_retired (0.53, 0.65, 0.98): named exit-2 stubs per removed
command, matching the I41 `lsp --baseline` retired-flag precedent;
the round-1 near-tie was investigated on the merits. Majority for
delete_now (0.90, 0.70) over a low-confidence review_needed round 2
(0.50); investigated — the gate cannot pass with zero inputs and
pausing would keep a dead config, so tscheck/ and tsc.yml are
deleted and I45 item 2 recreates the gate. Judgments are advice;
the checks below are the proof.

## Implementation

- Ownership table by symbol reachability from the surviving
  dispatch roots (`run`, `runLSP`, `runCurrentParse`,
  `runInspectProject`, `runInspectTypes`): 26 root files fully
  unreachable (parser, checker, evaluator, prover, Z3, baselines,
  explain, lint), `lsp.go` 14/45 live (transport plus the current
  bridge), `main.go` 5/18 live (dispatch plus version).
- Deleted: the 26 unreachable production files, 27 predecessor
  symbols from `lsp.go` (877 lines of legacy diagnosis), 13 from
  `main.go` (four commands plus the legacy parse/check/report
  chain), and 121 test files whose contracts were superseded.
- Surviving launcher: `current_parse.go`, `current_project.go`,
  `current_types.go`, `lsp.go`, `main.go`; commands assert, build,
  run, parse, inspect-project, inspect-types, runtime-check,
  catalogue-check, version, lsp, clean. `explain`, `lint`,
  `baseline`, `normalize` exit 2 with `canlc <name> was retired
  with the predecessor toolchain in I44`.
- Deleted every retired `std/*/*.can`, `std/*/*.ts`, the twelve
  retired `errors.json` files, `host.externs.ts`,
  `platform.d.ts`, all twenty `sketches/*/` programs, plus
  `tscheck/` and `.github/workflows/tsc.yml`. Kept:
  `std/catalogue/` (generated, verified fresh),
  `std/html/HISTORY.md`, `sketches/README.md`,
  `sketches/CLEAN_ROOM_REVIEW.md`, and all maintained projects.
- `go.mod` simplified: `go mod tidy` dropped the now-unused
  `go-cmp` dependency; `pg_query_go` stays for the SQL parser.
- `compiler/retirement_test.go`: the static gate pins the five
  production files, restricts launcher imports to the standard
  library plus `compiler/internal/`, and forbids defining or
  referencing 33 retired entry/engine/grant symbols in any
  launcher source including tests.
- `compiler/current_parse_test.go`: legacy-syntax rejection now
  covers five shapes (`rev`/`extern`, `mod` headers, `effects`
  clauses, `given` tables, `decreases` fuel); `cli_test.go`
  covers all four retired stubs at exit 2.

## Verification

- `go test -count=1 ./...`: all 16 packages pass, including
  `tests/integration` (255s) with `CAN_BUN` 1.4.2,
  `CAN_BUN_ARCHIVE` (hash-verified), `CAN_TSC` (TS 7.0.2), and
  `CAN_TEST_POSTGRES_URL` on the disposable container.
- `TestNoPredecessorPaths` passes; scratch negative (a probe file
  defining `legacyParsePaths`) fails both the file-set pin and
  the symbol blocklist; restored after.
- Searches over all Go sources: zero `z3`/`smtBinary` references,
  zero `legacyParse`/`runTestValue`/`VerifyProve`/
  `FingerprintProgram` references outside the gate's own
  blocklist, and the only `extern` mention is modcheck's
  retired-shape rejection message. (`internal/check` owns an
  unrelated current `checkProgram`; the gate correctly scopes to
  the launcher package.)
- A freshly built `canlc` exits 2 with the retired message for
  all four removed commands and still reports `canlc dev` for
  `version`.
- `bun test runtime/test/`: 850 pass, 0 fail, 63,285 expectations
  (142 files) — unchanged, proving no runtime regression.
- `cataloguegen --check`, `modcheck` (62 sources), `gramcheck`:
  pass.
- `gofmt` reports zero files and `go vet` is clean across the
  repo — the four pre-existing `gofmt` findings lived in deleted
  files.
- No audit-probes archive exists anywhere in the tree; none was
  restored.

## Limitations

- Prose references to removed commands and goldens remain in
  design documents; I50 owns the documentation reconciliation.
- The untracked `tscheck/node_modules/` directory stays on disk
  for I45's gate recreation to reuse or replace.
- No strict TypeScript gate runs until I45 recreates it; staged
  emitted-TS legs (`CAN_TSC`) still typecheck every build.
