# SQLite adapter review repairs

Continue the existing Muse session after its initial handoff. This checklist is
part of E02 in [the main checklist](checklist.md), with the selected backend and
scope fixed by [the design](design.md) and [decision](decision.md).

The concrete finding and native engine evidence are in
[Codex's review](evidence/codex-review.md). Keep all other established contracts.
Codex owns the review/decision files; Muse owns the implementation, tests, and
progress in this checklist until handoff. Do not commit, push, or archive.

## Lane R — nested compound validation

- [x] R01 (initial D05). Validate every reachable `*ast.SelectStmt` during the
  existing iterative, typed-nil-safe, budgeted node walk. Inspect each SELECT's
  own nonfinal direct cores for misplaced LIMIT/ORDER BY. Remove the root-only
  validation path and update helper comments/names to reflect actual behavior.
  Keep descriptor outer LIMIT extraction root-only; a valid subquery limit must
  never bound its parent. Ownership: `compiler/internal/sql/sqlite.go`.
  Acceptance: all invalid examples in the review fail through `Analysis.Failure`
  and `CheckDescriptorDialect`, with character-cursor conversion preserved; no
  recursive walk, duplicate full walk, new dependency, or engine planner added.
  Evidence: done 2026-09-28 — sqlite.go: renamed helper to
  misplacedCoreClause(sel *ast.SelectStmt) (one SELECT's own direct nonfinal
  cores), invoked for every *ast.SelectStmt inside the existing iterative
  budgeted typed-nil-safe collectBinds walk (fail-fast, deterministic; root
  visited first so root violations report exactly as before); removed the
  root-only call block; sqliteLimitShape untouched (outer extraction stays
  root-only); comments updated. All 3 review LIMIT examples + ORDER BY
  example fail via Failure at both levels (R02 tests). No new walk,
  dependency, or planner; cursor conversion reused.
- [x] R02 (R01). Add focused descriptor-level rejection cases for nested CTE,
  scalar-subquery, and INSERT SELECT compounds, covering both LIMIT and ORDER BY.
  Include valid nested controls and confirm an inner bound alone does not satisfy
  the outer SELECT's bound requirement. Ownership:
  `compiler/internal/sql/sqlite_contract_test.go`; relevant authored provenance
  wording if it says validation is root-only. Acceptance: tests distinguish the
  nested syntax defect from unrelated cardinality/parameter errors, and check a
  useful nested UTF-8 failure cursor. Evidence: done 2026-09-28 — 3 new
  tests in sqlite_contract_test.go, all hand-derived: Rejected (5 cases:
  CTE/scalar/INSERT LIMIT @20/17/23, CTE ORDER BY @29 term offset, UTF-8
  CTE LIMIT byte-24→rune-23, all with cleared statements/version);
  FailDescriptors (4 descriptor-level rejections asserting the engine
  wording, distinguishing from cardinality/param errors); ValidControls
  (legal last-core ORDER BY+LIMIT in CTE, scalar limit, INSERT SELECT
  compound execute, inner-bound-alone → "unbounded SELECT"). Provenance
  needs no change ("top-level LIMIT shape" = extraction, still root-only).
- [x] R03 (R01,R02). Run gofmt on touched Go files, the focused SQL package suite,
  and directly affected checker/emitter SQL checks with `-p 1` and bounded
  command timeouts. Update this checklist with exact commands/results and report
  handoff. Ownership: this checklist and compact evidence, plus owned temporary
  artifacts requiring cleanup. Acceptance: green checks, `git diff --check`, no
  new leftover probes/binaries/caches; no benchmarks, bundles, broad builds, or
  unrelated edits. Existing runtime checks need not be repeated for Go-only
  repairs. Evidence: done 2026-09-28 — `gofmt -l compiler/internal/sql/`
  empty; `go test -p 1 ./compiler/internal/sql/ -count=1 -timeout=180s`
  ok; emit `-run TestSQL` ok; check `-run 'TestSQL|TestPoolInputs'` ok;
  `git diff --check` exit 0. Changed files are exactly the initial set +
  R01/R02 edits to sqlite.go and sqlite_contract_test.go (no new files,
  no unrelated edits). No leftover probes/binaries/caches: /tmp probe
  module and compiler/tmpqualprobe verified absent; no new temp workspace
  allocated; Codex-owned node_modules symlink untouched. Runtime suites
  not re-run (Go-only repair per checklist). Handoff: repair complete,
  reviewable diff on codex/sqlite-external-parser, nothing committed.
