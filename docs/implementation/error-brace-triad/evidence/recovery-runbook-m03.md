# Recovery runbook — M03 + remainder (shell EMFILE, committed stack green)

Shell died with EMFILE (`Too many open files (os error 24)`) on every
`muse.bash` call mid-M03. File tools still work. This runbook records the
exact pending state. Committed stack below is verified green; do NOT
re-verify it blindly — resume at "Pending".

## Committed stack (verified, do not redo)

On top of `24797cc9` (foreign `docs/user-guide/README.md` writer, active —
see below):

- `effdc0f7` N05 check/resolve/emit (99 files)
- `bb8f2d2d` F04 syntax fixtures (2 files)
- `a2e2aeb6` P02 catalogue (3 files; mirrors intentionally stale → P03)
- `a7c72e5b` P04 LSP/current (11 files)
- `9e167f1b` N05-ext types/driver/project/browser (16 files)
- `49f6092b` M01 corpus testdata/std/shared (61 files, delimiter-only + 1 digest)
- `2a582cd5` M02 corpus examples/tests/tools/docs (111 files, delimiter-only + 2 digests)
- `87ddf1d0` fix(runtime) modules.json inventory sync (pre-existing drift, NOT triad)
- `c489707d` docs tasks.md evidence for the above (F04/N05/P02/P04/M01/M02 checked)

Suites green at `c489707d` (all `go test -p 1 -count=1`, `CAN_BUN=$(which bun)`
in env): syntax, resolve, check, emit (4 batch-route tests require CAN_BUN
by design), catalogue targeted (only `TestCompleteInventoryAndMirrors` red
by design until P03), compiler root, types, driver, project, browser;
`go run ./tools/gramcheck` OK; `bun test tools/runtime/module-inventory.test.ts`
passes. Emit showed 2 transient full-package failures in ~14 runs
(test unidentified — capture with `-v` to file on next failure; live-port
and batch tests all pass in isolation).

## Pending (uncommitted, 24 M03 Go files)

`git status` dirty (all ` M`, plus 5 known-untracked scratch files that must
NEVER be staged: `docs/implementation/error-brace-triad/evidence/muse-monitor.md`,
`.../recovery-runbook-n05.md`, `.../recovery-runbook.md`,
`.../muse-prompt.md`, `evidence/`):

- tests/failure-conventions/owner_test.go (5 bounds, 3 decls, 9 ctors)
- tests/failure-conventions/retry_test.go (12 bounds, 6 decls, 79 ctors)
- tests/integration/assertions_test.go, assets_test.go, browser_build_test.go,
  callables_test.go, checks_test.go, cli_test.go, coordination_test.go,
  crypto_test.go, files_test.go, format_test.go, gate5_frontend_test.go,
  generics_test.go, input_capture_test.go, markdown_test.go, process_test.go,
  questions_test.go, streams_test.go, verified_build_test.go, wrap_test.go
  (bounds+decls; `failed(`/`ai::invalid_answer{` ctors where classified)
- host/conformance/admission_test.go, w2_test.go, w2_e2e_test.go (1 bound each)

M03 verification DONE (static, shell was alive): zero `emits [` / `error N(`
remain in M03 Go; 40/40 replaceOnce anchors match exactly once per fixture
file; crypto/markdown fixture anchors 3/3 match migrated corpus; records
(`retry::rejected/completed`, `profile`, `receipt`, `user_id`, `tier_view`,
`records::flawed_save`, `helpers::receipt`) and calls (`call …`, `use …`,
`relay call …`) keep parens — see session log for the classification
evidence. `overlay_test.go:43` keeps `emits []` intentionally (N05, committed).

M03 verification NOT done (needs shell): `go vet` compile check;
`go test ./tests/failure-conventions/ ./tests/integration/ ./host/conformance/`
— expect SKIP without `CAN_BUN_ARCHIVE`/`CONV_BUNDLE` (no archive is
configured; bundles are out of bounds; record the environment skip per V03).

## Resume sequence (shell required)

1. `echo ok`; `git status --short` (expect the 24 M03 files + 5 untracked).
2. `go vet ./tests/integration/ ./tests/failure-conventions/ ./host/conformance/`
   then the M03 test runs above; fix any red (likely nothing — static checks
   passed), record skips honestly.
3. Commit M03 (exact 24 paths) + tasks.md M03 evidence/checkbox.
4. M04 audit: parse/format fixpoint over the migrated corpus incl. `tests/`
   fixtures (no bun needed — use the 59-file walker + `canlc parse/format`);
   classify remaining old spellings (strings in `examples/language-site`,
   9 non-executed frozen projects, `diagnostic-parse`, intentional negatives).
5. P03: `go run ./compiler/internal/catalogue/cmd/cataloguegen`, inspect the
   4 mirror diffs (expect README-only brace bounds + email_href preserved),
   `make catalogue-check` must pass; commit.
6. P05 docs: `README.md`, `REQUIREMENTS.md`, user guides; CAREFUL with
   `docs/user-guide/README.md` — foreign writer committed `24797cc9` mid-run
   ("nested file failure fixtures", may embed old syntax); migrate syntax
   spans only, preserve their prose, or escalate.
7. V01→V02→V03 serialized (`-p 1 -count=1`, `CAN_BUN` set); record commands
   and skips in `evidence/verification.md`. Leave V04 for Codex.
8. Cleanup: delete `/tmp/can-migration-apply.py`, `/tmp/http-*.can`,
   `/tmp/emit-*.log`, `/tmp/*-files.txt`, `/tmp/*minus.txt`, `/tmp/*plus.txt`;
   report any failure. The map applier already served its purpose (M01/M02
   committed); M04 must NOT re-apply it.

## Notes

- `restart-prompt.md` (untracked scratch holding the restart text) vanished
  mid-run; no session command deleted it (likely a test workspace sweep).
  Content survives in the session log; no recovery needed.
- `.muse/worktrees/*` contains foreign/abandoned checkouts — never touch.

## Shell-free appendix (EMFILE turns 2–3; needs shell verification)

M03 file count is now 24 Go files (added `tests/integration/wrap_test.go:176`,
anchor `ai::invalid_answer(` → braces, verified against migrated
`compiler/testdata/current/wrap/main.can:89`).

M02 generators DONE shell-free (string-literal swaps only, no Python syntax
impact — still run `python3 -m py_compile` on recovery):
`tools/performance/drivers/compiler.py:53,56` (`{{}}` f-string escape), `:216`,
`tools/performance/test_editor_experiment.py:73,75` (`emits []` → `emits {}`).
Benchmark-only (no gate runs them); commit as M02 follow-up with the M03 stack
or separately. Pending dirty total: 26 files (24 M03 Go + 2 .py).

P05 span inventory (G01 scope + spec values; migrate on recovery):
- `REQUIREMENTS.md:54` ("No curly braces outside string literals" — the claim
  to correct), `:69`, `:114` (schematic `emits [...]` prose).
- `docs/user-guide/README.md:114,133,174` (real `emits [...]` snippets in the
  ACTIVE foreign-writer file — syntax spans only, or escalate to the user).
- `tests/failure-conventions/README.md:53,108`,
  `tests/failure-conventions/x-r07-1.md:21`,
  `examples/account-search/README.md:50` (schematic `emits []` prose).
- `docs/syntax-taste/decisions.md:1521` (prose `` `missing()` ``), `:1557`,
  `:1575`, `:1576`, `:1777`, `:1780` (`missing()`, `access_denied()`,
  `below_minimum(...)` values; doc is live, bounds done in P01).
- `docs/syntax-taste/technical-spec.md:442,446` (`llm::refused(...)` values).
- Historical: all `docs/syntax-taste/*review*.md`, `ai-io-spec.md`,
  `implementation-gaps` non-executed projects, Jev evidence — DO NOT TOUCH.
- Root `README.md`: no old-spelling hits found.

M04 heads-up: `examples/language-site/src/site.can:95,142` keep `emits []`
inside string literals (displayed sample code — map excluded strings by
construction; confirm intent, likely keep). 9 non-executed frozen projects +
`diagnostic-parse` stay historical. FC `*-negative`/`gallery-failing` .can
files WERE migrated (their negative intent lives in Go assertions, not syntax).
