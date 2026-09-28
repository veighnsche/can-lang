# SQLite external parser execution checklist

Supporting design: [design.md](design.md), [brainstorm.md](brainstorm.md),
[prior Jev requests/responses](evidence/brainstorm-jev.json).

Executor: Muse `muse-spark-1.3-contributor`, configured reasoning unchanged,
authorized `--yolo`, worktree off. Codex coordinates, decides, reviews and verifies.
Every task records commands/results or evidence links below; do not check a box
because a file was merely written. Continue all ready tasks in dependency order
in one sustained session. Do not commit/push, create another checkout, publish,
run benchmarks or broad distribution builds. Report a genuine blocker precisely.

## Lane A — baseline and dependency qualification (Muse)

- [x] A01 (prerequisite: design). Inspect AGENTS.md and current status; record
  base commit, authored/vendor footprint and current bounded test surface.
  Ownership: this checklist and `evidence/qualification.md`. Acceptance: clean
  initial checkout accounted for and expected compiler/runtime contracts listed.
  Evidence: done 2026-09-28 — base 79830698227d2706271ff1427c107ef2325f32a,
  clean except untracked docs/implementation/sqlite-external-parser/;
  88 files / 6,729,649 B in sqlite/ (parser.c 5,809,039 B); authored:
  glue.c/go/_test.go + PROVENANCE.md; Analysis contract + Version=15;
  `go test ./compiler/internal/sql/ -run TestNothingMatchesThis` ok 0.310s;
  see evidence/qualification.md §A01.
- [x] A02 (A01). Resolve exact official grammar/runtime modules using the shared
  Go module cache; inspect archive source inclusion, module hashes, runtime ABI
  admission and license/provenance. Ownership: temporary probe + evidence only.
  Acceptance: actual SetLanguage and parse work, not pointer-only smoke test;
  no sources copied into Can. Evidence: done 2026-09-28 — grammar
  v0.1.1-0.20260503135658-7f69bb66845b (ABI15, C byte-identical to vendored),
  runtime v0.25.0 (admits 13..15, MIT); live v0.25.0 tag absent (ls-remote
  empty) vs proxy Ref refs/tags/v0.25.0 recorded; real SetLanguage+Parse
  A02-PROBE-PASS with glue_test-identical spans; tidy-rewrite anomaly noted
  (zero functional impact; v0.1.0 alt pin differs only in CI yml).
  See evidence/qualification.md §A02.
- [x] A03 (A02). Run a bounded official-binding probe over existing SQLite
  corpus and additional UTF-8/decoy/parameter/CTE/LIMIT/RETURNING/error cases.
  Check Close ownership and copied-tree/node count bounds if relevant.
  Ownership: temporary probe + evidence. Acceptance: compact raw outcomes and
  any grammar differences recorded; no performance measurements. Evidence: done
  2026-09-28 — 63/63 MATCH (21 corpus + 42 extra) via split binaries (258
  duplicate ts_* symbols forbid one binary); span-key fix found by diff;
  Close non-idempotent (double-Close segfault); cap guard + 40KB/5k-deep OK;
  raw: evidence/a03-official-outcomes.txt; §A03.
- [x] A04 (A03). Resolve Meyer v0.1.2 normally and run the same cases; inspect
  parser AST/visitor, byte positions, options, resource/depth limits and maintenance
  evidence. Ownership: temporary probe + evidence. Acceptance: concrete results
  support or disqualify using it for Can, with smallest failure examples.
  Evidence: done 2026-09-28 — dep-free, MIT, HEAD==v0.1.2, sqlc-main-only
  adoption; byte spans, Walk (comma-LIMIT order artifact), sigil-distinct
  numbering, `;`-widened spans, byte error offsets, maxDepth=10000 clean
  error; 53/74 identical, 21 diffs enumerated (6 accept/reject incl $1/:1/
  UPDATE..LIMIT/$name(arg); sigil + overflow numbering); raw:
  evidence/a04-meyer-outcomes.txt; §A04.
- [x] A05 (A03,A04). Use a bounded in-memory Bun SQLite correctness probe for
  disputed syntax/parameter behavior, especially sigils and `$1`; separate
  syntax and schema failures. Ownership: compact probe/evidence. Acceptance:
  intended contract assessment, not blind agreement with old bugs. Evidence: done
  2026-09-28 — engine 3.54.0 oracle: $1/:1/@1/$name(arg) ACCEPTED (Meyer right,
  tree-sitter over-strict); sigils DISTINCT 1,2,3 (old collapse overturned);
  UPDATE..LIMIT accepted by Bun build (Meyer needs option=true); ?0/overflow/
  mid-compound-LIMIT/deep5k engine-rejected but both parsers over-accept (open
  points); Meyer failures byte-identical to engine. Raw: a05-oracle.txt; §A05.
- [x] A06 (A05). Finish `evidence/qualification.md` with both candidates' raw
  checks and a recommendation; write `evidence/qualification-ready` and announce
  readiness. Codex supplies `decision.md` after review/required Jev consultations.
  While waiting, do independent provenance inventory, then poll only every 30–60s
  without ending the session; continue as soon as decision.md exists.
  Acceptance: decision input covers correctness, resource behavior, dependency
  identity and open limitations. Evidence: done 2026-09-28 —
  qualification.md §A06 (comparison table + Meyer+UDL recommendation +
  6 open questions); qualification-ready written; awaiting Codex
  decision.md. Independent provenance inventory done while waiting:
  evidence/provenance-inventory.md (C03 targets: PROVENANCE.md,
  sql/*.go comments, descriptor.ts:45-47,116; historical annotated).

## Lane B — chosen parser integration (Muse, after Codex decision)

- [x] B01 (A06 + decision.md). Pin only the chosen production module(s) in
  go.mod/go.sum; discard probe-only requirements. Ownership: go.mod/go.sum.
  Acceptance: normal external dependency resolution, verified sums, no replacements
  pointing at local copies, Go/toolchain constraints respected. Evidence: done
  2026-09-28 — `go get github.com/sqlc-dev/meyer@v0.1.2` + tidy with B02 import:
  go.mod direct v0.1.2, go.sum h1:40Ng9Glnx7CTf3yOYV7jfvDIN7voIFjrStkIULS+uws=
  (== A04 probe sums), `go list -m` v0.1.2, no replace, `go 1.25.0` satisfies
  Meyer's `go 1.24`; tidy dropped the pin until B02 imported it (Codex-noted
  mechanism) and corrected a stale tidb `// indirect` (directly imported by
  mysql.go) + tidy-completeness sums for the existing graph (zero new modules,
  zero tree-sitter). Build OK.
- [x] B02 (B01). Replace vendored SQLite parsing with a small authored adapter
  using upstream APIs. Ownership: compiler/internal/sql/sqlite.go and, only if
  necessary, authored files under compiler/internal/sql/sqlite/. Acceptance:
  exact byte spans, all occurrences, count/kind, top-level LIMIT and RETURNING,
  errors and deterministic cleanup; useful resource guard retained. Evidence: done
  2026-09-28 — new compiler/internal/sql/sqlite.go (Meyer, ~330 lines, no
  sqlite/ subdir needed): UDL:true ParseString, native `;`-spans, 26-kind map,
  upstream numbers, iterative budgeted collect + sort + span validation,
  last-core LIMIT, RETURNING, engine-verbatim misplaced-clause failures,
  1MiB + NUL guards, `#` policy rejection, no panics swallowed, no cgo
  (nothing to close). Corpus run: all sqlite verdicts hold except deliberate
  B05 updates (9× version 102, lite-17 matcher, lite-20 param_name); tiling
  passes (sites exact). Guard/edge regressions land in C01.
- [x] B03 (B02). Correct confirmed parameter-mapping bugs and no-longer-correct
  assumptions without compatibility shims. Ownership: same SQLite analysis files
  and directly related tests. Acceptance: SQLite numbering is explained and covered
  for bare, explicit, repeated named and sigil-distinct names. Evidence: done
  2026-09-28 — numbering now upstream-sourced (mapSQLiteNumbers deleted, no
  shims); CheckSiteNames comment corrected (sigils distinct, suffix check kept);
  `?0`/overflow rejected by existing CheckSites coverage (no duplication); `#`
  explicitly rejected; new sqlite_params_test.go (11 tests: bare/explicit/
  max+1/repeated/sigil-distinct with exact spans, digit+TCL parse+field-policy,
  `#`, `?0`/overflow, free limit name) all pass.
- [x] B04 (B02). Remove tracked third-party SQLite grammar/runtime C, headers,
  portable/unicode/wasm payload and obsolete custom C serialization; keep/rewrite
  useful authored tests. Ownership: compiler/internal/sql/sqlite/ only.
  Acceptance: no parser source payload retained as archive, copied Go, submodule,
  generated dump or replacement vendor tree. Evidence: done 2026-09-28 —
  compiler/internal/sql/sqlite/ deleted (88 files, 6,729,649 bytes incl.
  5.8MB parser.c, runtime C/headers, unicode/, portable/, wasm-stdlib/,
  glue.c/go, PROVENANCE.md); zero importers remain (only sqlite.go used
  it); sql package builds; deletions unstaged for review. Also removed a
  stray `go build` binary ./tmpqualprobe (owned artifact).
- [x] B05 (B02,B03). Update parser identity/content pin and metadata coherently
  through compiler/emitter/runtime test expectations as needed. Ownership:
  compiler/internal/sql, directly affected compiler tests/emission and minimal
  runtime metadata tests only when required. Acceptance: ABI is not falsely called
  unique content identity; no stale emitted metadata. Evidence: done 2026-09-28 —
  SQLiteVersion=102 (Meyer v0.1.2 release stamp); corpus/F03 use the const;
  tree-sitter comments → Meyer (parser/dialect.go); lite-20 corpus
  syntax→param_name + corrected note (engine accepts $1); syntax matcher +=
  "unrecognized token"; descriptor.ts sqlite:102 + comment; 24 runtime
  fixtures → 102 (hedge 4, returning 1, sqlite.test 18, budget 1; pg/mysql
  kept); emitter passes Version through (no pin); runtime lint:fix+format
  clean (5 files only); full sql package green. Notices/lock → C03.

## Lane C — regression coverage and provenance (Muse)

- [x] C01 (B02,B03). Add focused contract tests for UTF-8 spans, decoys, parameter
  spellings/reuse, comments/separators, CTE/nested limits, expression/OFFSET/comma
  limits, malformed input and DML RETURNING. Ownership: compiler/internal/sql tests.
  Acceptance: meaningful assertions of Analysis/descriptors and failures, no
  implementation-mirroring tests. Evidence: done 2026-09-28 — new
  sqlite_contract_test.go (20 tests, all hand-derived spans/cursors):
  UTF-8 spans, comment/string decoys, empty/separators, native `;` spans,
  CTE/nested limits, 8 LIMIT forms + numeric $1 limit, misplaced LIMIT/
  ORDER BY incl. masked combo, RETURNING shapes + decoys + params,
  malformed + UTF-8 error cursors, NUL/size/node budgets, flat-chain
  traversal, upsert/window binds, 25 kind mappings, UDL admission.
  Full sql package green. One test-side miscount caught and recounted.
- [x] C02 (B05). Check emitted descriptors preserve exact SQL segments and values;
  use focused compiler checks. Ownership: existing focused emitter/check tests.
  Acceptance: native Bun.SQL templates remain correct with updated parser identity.
  Evidence: done 2026-09-28 — new TestSQLDescriptorsEmitSQLiteTable (emit
  sql_test.go): full check→emit path for a named-param sqlite descriptor
  asserts exact text segments, param 1/2 order, kind select_statement and
  version 102 in program/state.ts. Bun.SQL template behavior verified via
  runtime sqlite/returning tests in D02 (fixtures already at 102).
- [x] C03 (B01,B04,B05). Replace incorrect provenance and update current parser
  documentation/notices/references. Historical evidence may be annotated rather
  than rewritten. Ownership: parser provenance plus relevant existing docs/notices.
  Acceptance: upstream module version, commit/hash, license, update procedure,
  authored ownership and shared-cache boundary stated accurately. Evidence: done
  2026-09-28 — new compiler/internal/sql/sqlite-provenance.md (pin, sums,
  MIT, UDL option, stamp-102 rule, ownership, cache boundary, update
  procedure, history correcting the "no Go module" claim); new
  notices/sqlite-binding.lock.json (sums == go.sum, zero transitives,
  rejected pair + reason) + meyer-LICENSE.txt (byte-identical to upstream
  raw fetch); notices/README + distribution/README updated. Historical
  decision/research mentions left untouched; only intentional code mention
  is the "not a Tree-sitter ABI" clarification.
- [x] C04 (B01,C03). Verify source module archive inclusion and normal prefetch then
  offline dependency resolution/build expectation without full distribution build.
  Ownership: evidence, only targeted provisioning edits if actually necessary.
  Acceptance: `GOPROXY=off`/`CGO_ENABLED=1` behavior preserved. Evidence: done
  2026-09-28 — `go mod verify`: all modules verified; archive contains
  parser/lexer/ast/LICENSE/go.mod; sums == go.sum/lock; `GOPROXY=off
  CGO_ENABLED=1 go build ./compiler/internal/sql/` OK; `GOPROXY=off
  CGO_ENABLED=1 go test ./compiler/internal/sql/ -run TestCorpus$` ok;
  offline `go list -m` resolves meyer v0.1.2 + pg_query. Strict
  `go list -m all` offline still wants uncached pre-existing graph
  leaves (0 meyer-related; unchanged by this task). No provisioning
  edits needed; shared cache only.

## Lane D — bounded verification and handoff (Muse)

- [x] D01 (B,C). Run gofmt and focused SQL package/corpus tests, then directly
  affected compiler checker/emitter tests with bounded concurrency/timeouts.
  Acceptance: exact commands/results retained, no hidden skips. Evidence: done
  2026-09-28 — gofmt clean (compiler/, distribution/); go vet sql clean;
  `go test -p 1 ./compiler/internal/sql/ -count=1` ok; emit `-run TestSQL`
  ok (incl. new sqlite table test); check `-run 'TestSQL|TestPoolInputs'`
  ok (incl. TestSQLTransaction*). Bounded -p 1 + timeouts. The later
  integration run with JSON test events confirmed TestMySQLDifferential is
  environment-skipped without CAN_TEST_MYSQL_URL; the original terse output
  did not expose that skip. No correctness failures.
- [x] D02 (D01). Run compact in-memory Bun.SQL descriptor/SQLite correctness
  checks when needed to verify changes; avoid staged full bundles. If authored
  runtime TypeScript was edited, run required lint:fix:runtime, format:runtime,
  check:runtime and relevant tests. Acceptance: runtime native behavior checked or
  unchanged boundaries explicitly supported by evidence. Evidence: done
  2026-09-28 — descriptor/sqlite/returning suites: 23 pass, 0 fail
  (2 live-DB skips); budget+hedge: 19 pass, 0 fail (12 live-DB skips);
  fixed 3 missed positional `15` literals in sql-returning.test.ts found
  by the version-mismatch failure (no test weakened); lint:fix + format
  + check:runtime (lint/format/typecheck) all green; only the 5 intended
  runtime files touched. No staged bundles.
- [x] D03 (D01,D02). Inspect complete diff, module graph, stale references and
  tracked payload; record removed bytes/files separately from authored edits.
  Acceptance: no unrelated changes or obsolete dependencies. Evidence: done
  2026-09-28 — REMOVED: 88 files / 6,729,649 bytes (vendored runtime C,
  5.8MB parser.c, unicode/portable/wasm-stdlib, glue bridge, old
  PROVENANCE). AUTHORED: sqlite.go Meyer adapter, 2 contract test files,
  provenance doc, sqlite lock + MIT notice + notice/README updates,
  lite-20 corpus correction, 27 version-102 fixture updates, go.mod/go.sum
  (meyer v0.1.2 + tidy tidb-direct correction). Diff: 110 files,
  +1058/-221753. Graph: meyer only, zero tree-sitter; tidy hash-stable;
  zero LanguageVersion refs; sqlite/ gone; no archives/copies. Only
  unrelated-adjacent change is the required tidy tidb-direct fix (noted).
- [x] D04 (D03). Clean every owned temporary probe/workspace/output; preserve
  compact source probes/evidence when useful, never binaries/dependency copies.
  Acceptance: cleanup verified and failures explicitly reported. Evidence: done
  2026-09-28 — /tmp/can-sqlite-qual-DmApBm + compiler/tmpqualprobe both
  `rm -rf` verified gone; post-cleanup sql build OK; compact sources kept
  (7 probe files, 24,253 bytes) + raw outcomes + oracle + inventory;
  binaries/outputs/go.sum scratch discarded. No cleanup failures. No new
  temp workspaces were allocated in lanes B–D.
- [x] D05 (D04). Mark completed tasks with evidence, report remaining limitations
  and hand back the reviewable diff without committing or archiving the worktree.
  Acceptance: Codex has exact verification commands and evidence for review.
  Evidence: done 2026-09-28 — all Muse lanes (A–D) complete with evidence
  above; HEAD still 79830698227dd2706271ff1427c107ef2325f32a on
  codex/sqlite-external-parser, nothing committed/pushed/archived, 111
  changed paths (88 D, 17 M, 5 A-intent, 1 untracked docs dir).
  VERIFY: `go test -p 1 ./compiler/internal/sql/ -count=1` (ok);
  emit `-run TestSQL` (ok); check `-run 'TestSQL|TestPoolInputs'` (ok);
  `bun run check:runtime` (ok); `bun test runtime/test/sqlite.test.ts
  runtime/test/sql-returning.test.ts runtime/test/sql-descriptor.test.ts`
  (23 pass); `bun test runtime/test/sql-budget.test.ts
  runtime/test/http-hedge.test.ts` (19 pass, live legs skip);
  `GOPROXY=off CGO_ENABLED=1 go build ./compiler/internal/sql/` (ok);
  `gofmt -l compiler/ distribution/` (empty); `go mod verify` (all).
  LIMITATIONS: Meyer young (v0.1.2, sqlc-main-only); ?0/overflow accepted
  at Analysis but rejected by CheckSites coverage (tested); mid-compound
  LIMIT/ORDER BY rejected top-level only (subquery-nested still
  over-accepted, engine rejects at prepare); >2k nesting over-accepted
  (Meyer caps 10k cleanly); EXPLAIN-wrapped compounds not unwrapped
  (unobservable — EXPLAIN never validates); spans now include `;` and
  failure texts changed (both intentional, no compat requirement);
  lite-20 corpus corrected syntax→param_name; tidb `// indirect` tidy
  fix bundled; strict `go list -m all` offline still needs network for
  pre-existing leaves (unchanged). No perf claims made. This is the initial
  handoff record: the nested/wrapped compound admission gap above was subsequently
  fixed by R01–R03 and independently reviewed in E02; it is no longer a limitation.

## Lane E — independent review and completion (Codex)

- [x] E01 (D05). Review full authored diff and candidate evidence independently;
  use separate read-only reviewer. Evidence: Codex reviewed the authored adapter,
  policy comments, tests, runtime metadata, module changes and provenance. A
  separate read-only reviewer found R01, independently corroborated against
  native SQLite. See evidence/codex-review.md. Initial handoff received; nested
  compound admission is a repair requirement, not an accepted tolerance.
- [x] E02 (E01). If defects found, add small repair tasks below and resume the same
  Muse session/checklist. Review repaired code and rerun only relevant checks.
  Evidence: [repair-checklist.md](repair-checklist.md), R01–R03, saved before
  resuming Muse. R01–R03 completed in the same native session; independent
  reviewer confirmed the repair and found no remaining concrete defect. Reviewed
  hashes and final checks are in evidence/codex-review.md and codex-verification.md.
- [x] E03 (E01,E02). Run independent focused verification and diff/cleanup checks.
  Leave reviewed work on codex/sqlite-external-parser; preserve checkout while
  unintegrated work needs review. Report exact choice, removed payload, test outcomes,
  limitations and cleanup. Evidence: evidence/codex-verification.md — full SQL
  plus focused checker/emitter tests passed offline with CGo enabled; runtime
  check passed; 42 runtime tests passed, 14 external-DB cases skipped, 0 failures.
  Pin/sums/license, removed bytes and stale-reference checks passed. All owned
  temps and the exact temporary dependency symlink reclaimed, target preserved.
  No cleanup failures; see evidence/codex-cleanup.json. Uncommitted reviewed work
  retains this existing worktree for integration; no commit/push/archival.

## Session and progress

- 2026-09-28: Codex read `muse --help` and `muse exec --help`; requested flags
  supported. Branch created from clean detached worktree. Complete checklist saved
  before dispatch. No model-step cap or reasoning override will be used.
- 2026-09-28: Muse executed lanes A–D in one sustained run. Qualification
  (74-case differentials + engine 3.54.0 oracle) recommended Meyer; Codex
  decided Meyer v0.1.2 via decision.md after 3 Jev rounds (Meyer/official/
  Meyer, non-unanimous, evidence-led). Implemented: Meyer adapter with
  guards, 88-file vendored removal (6,729,649 bytes), stamp 102, 31 new
  Go contract tests + 1 emitter test, 27 runtime fixture updates, lock +
  notices + provenance. All focused suites green; temps reclaimed.
  Diff left uncommitted on codex/sqlite-external-parser for Codex E-lane
  review. No commits, pushes, worktrees, benchmarks or broad builds.
- 2026-09-28: Codex resumed native Muse session
  01a0e816-7114-7251-9826-7717838b929b with the saved repair checklist. R01–R03
  completed; final independent review and bounded verification passed. The final
  suite adds 34 SQL contract tests and one emitter test. E01–E03 complete; owned
  temporary storage cleaned and reviewable uncommitted work preserved.
