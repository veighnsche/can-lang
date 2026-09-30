# Recovery runbook: N05 shell EMFILE outage (2026-09-30)

Session `01a0ef6b-d46a-7511-a29d-a94392b85799` (implementer). `muse.bash`
fails every call with `Too many open files (os error 24)`; file/search/edit
tools work. Prior recovery (N01–N03) fully committed (`f437c26e`,
`1f43d0e5`, `04776572`, `012cb40a`); N04 committed (`c1776b6e`).

## N05 state (UNCOMMITTED — verify with git status when shell returns)

- Production wording migrated: `check/program.go` (4), `check/form.go` (2
  comments), `check/completions.go` (`emits {}` + `region declares emits
  {%s}`), `check/action_bindings.go` (comment + `emits {%s}...requires
  emits {}`), `emit/browser_interface.go` (1 comment).
- Pass 1 (bounds + decls, `/tmp/can-n05/pass1.py`, may be gone after
  reboot): 909 `emits [`→`emits {}` + 11 `error N(`→`error N{}` across
  check/resolve/emit `*_test.go` (97 files). Includes boundary fix for
  `\nerror` inside interpreted Go strings; hand-fixed dynamic
  `errors_test.go:146` (`failed{item value}`).
- Pass 2 (`/tmp/can-n05/pass2.py`): diagnostic-driven `E(`→`E{}` loop.
  Rules: Go-string spans only; Can `//`/strings excluded; `=>`-pattern
  skip unless single `:` precedes `=>` on the line (assert-style);
  qualified names exact-global + any-qualifier in func/consts; bare names
  in failing func then consts; same-line matching; excludes
  `constructor_delimiter_test.go`, `error_value_test.go`.
- check: 54 → 40 failing; brace-diags exhausted. All 40 involve fixture
  reads (37 direct, 3 via `exactComposition` helper): converge at M01.
  `resolve` package GREEN.
- emit: loop converged (brace-diags exhausted, residual list printed but
  final count lost to outage — rerun to confirm). Residual triage NOT done.

## Shell-free progress (2026-09-30, session 01a0ef6b)

Without shell, via search/read/edit only (UNVERIFIED — rerun before commit):

- Static closure: converted 8 masked stragglers
  (`checks_test:35`, `wrap_test:292`, `coordination_test:59`,
  `static_origin_test:334+389`, `regions_test:231+417+418`,
  `closed_recovery_test:209`, `http_test:259`). Verified kept:
  records (`page_missing/found`, `note_failed`, `found`, `contract::failed`,
  `option::some`, `other`, `unit`, `seal_failed`), calls (`call …`,
  `set_timeout`, `use missing(2)` template name), paren patterns
  (`left(bind value)`), TS (`$failed()`, `$missing()`), N03/N04
  intentional negatives, `standard_failure{"boom"}` manufacturing
  negative (migrated to braces, still rejected).
- Guard sweep: added match+change guards to syntax-affected
  Replace-negatives — resolve (3: native_test ×2 hoists,
  symbols_test:228), check (13: checks map+numbered, native 107-map,
  coordination 150/154/177-map/220-map/284-map, transaction runner,
  callables:143, specialize 208-map, program 2 hoists). Emit: only one
  error-syntax Replace (action_bindings:223, positive builder) — none
  needed. Unscoped: byte-identical non-syntax mutations (dynamics
  provably unchanged) and the intentional `Replace(x,"box(3)","box(3)")`
  no-op (exported_generics:328, message assertions verify reason).
- F04 shell-free: migrated 6 positive sites
  (`syntax/wrap_test:11+19+61+86+88`,
  `syntax/declarations_test:216+408` incl. removing the F04 marker
  comment). Kept: old-decl/bound named rejects (387-389, 433-434,
  parser_test:70, 263-malformed list), paren constructor patterns
  (249 `pair(0,_)`, 293 reject entry), `box`/`receipt` records. Syntax
  Replace-negatives: single site already anchor-guarded
  (format_trivia_test:322). Did not touch N-owned
  `expressions_test.go`/`constructor_braces_test.go`. Terminal/generic
  coverage: N01/N02 tests + migrated positives.

## Resume sequence (shell required)

1. `echo ok`, then `git status --short` (expect N05 wording + ~100 test
   files; untracked `muse-monitor.md` Codex-owned, `muse-prompt.md`,
   runbooks stay untracked).
2. `go test -p 1 -count=1 ./compiler/internal/emit/ 2>&1 | grep '^--- FAIL'`
   — classify residuals fixture-vs-embedded (flag test bodies or
   their helpers containing `ReadFile|testdata|Fixture(t)|sourceProgram`;
   the rest are embedded and N05-owned).
3. `go test -p 1 -count=1 ./compiler/internal/check/ ./compiler/internal/resolve/`
   — confirm check residuals still fixture-only (expect ≤40), resolve
   still green (new guards must pass).
4. Focused verify: diagnostic wording tests
   (`TestOutwardErrorObligation` etc.), N03/N04 tests.
5. Load `bundled:git` (already loaded this session), commit N05
   (production + tests + tasks.md check), continue F04 → M01–M04 →
   P02–P05 → V01–V03. Leave V04 for Codex.
6. Cleanup: `rm -rf /tmp/can-n05` after N05 commit; report any failure.

## Temp files (owned, may not survive reboot)

- `/tmp/can-n05/pass1.py`, `/tmp/can-n05/pass2.py`, `/tmp/can-n05/canlayer.py`
  (spliced into pass2), `/tmp/can-n05/spot.go`, `/tmp/can-n05/check-fail.txt`.
- If lost: pass-1 rules above are sufficient to rebuild; pass-2's applied
  work persists in the tree, only triage/closure remain.
