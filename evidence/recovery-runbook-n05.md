# Recovery runbook — N05/F04/P02/P04/N05-ext (shell-free, UNVERIFIED)

Shell has been down with EMFILE (`Too many open files (os error 24)`) since
the N05 pass-2 loop. All work below was implemented with file tools only:
no test has been run, nothing has been committed. Verify + commit per slice
as soon as the shell recovers.

Prior runbook (`evidence/recovery-runbook.md`, N01–N03 steps) is absent from
this tree along with the rest of `evidence/`; this file recreates what the
recovery sequence needs. Committed slices (from session log, re-verify with
`git log --oneline` when shell is back): F02 base `a45f3af4`, N01 `f437c26e`,
N02 `1f43d0e5`, F03 `04776572`, N03 `012cb40a`, N04 `c1776b6e`.

## Resume sequence (shell required)

1. `echo ok` then `git status --short`; confirm only owned paths are dirty.
2. `go test -p 1 -count=1 ./compiler/internal/catalogue/` (P02).
3. `go test -p 1 -count=1 ./compiler/` (P04: LSP + current_* + package_instance).
4. `go test -p 1 -count=1 ./compiler/internal/types/ ./compiler/internal/driver/ ./compiler/internal/project/ ./compiler/internal/browser/` (N05-ext).
5. Re-run N05 suites: `./compiler/internal/check/ ./compiler/internal/resolve/ ./compiler/internal/emit/` plus `./compiler/internal/syntax/` (F04).
6. Commit per slice with exact owned paths only (never stage
   `evidence/muse-monitor.md`, `muse-prompt.md`; never push):
   - N05 (check/resolve/emit wording + snippets)
   - F04 (`syntax/wrap_test.go`, `syntax/declarations_test.go`)
   - P02 (`catalogue/types.go`, `catalogue/generate.go`,
     `catalogue/callable_types_test.go`)
   - P04 (`driver/hover.go`, `compiler/lsp_*_test.go`, `compiler/current_*_test.go`,
     `compiler/package_instance_test.go`)
   - N05-ext (`types/*_test.go`, `driver/*_test.go` except hover.go,
     `project/*_test.go`, `browser/browser_test.go`)
7. Continue: M01–M04 map applier, P03 regen (`make catalogue-check`), P05
   docs, V01–V03. Leave V04 for Codex.

## P02 — authored catalogue presentation (done, unverified)

- `compiler/internal/catalogue/types.go:86`: `typeRef.String()` bound now
  renders `emits {...}` (was `emits [...]`). Round-trip holds because
  `parseConcreteType` goes through `syntax.ParseType`, whose `errorBound()`
  (`syntax/parser.go:107`) requires braces since F02.
- `compiler/internal/catalogue/generate.go:122,128`: generated README
  operation/callback domain bounds now `{a, b}` (was `[a, b]`).
- `compiler/internal/catalogue/callable_types_test.go`: all 20 descriptor
  sites migrated, incl. `emits [][]` → `emits {}[]` (L53, bound-only
  conversion; the `[]` is an array suffix). `typ+"[]"` assertions (L24, L108)
  intentionally unchanged. No `[[...]]` negative was added: the task text
  does not ask for one.
- `errors.json`/structured JSON untouched. `catalogue.json` contains no
  `emits [` (verified by search before the outage window).

## P04 — LSP rendering and completion (done, unverified)

- `compiler/internal/driver/hover.go:343` (`formatHoverBound`): renders
  `emits {...}`. The five `lsp_g02_test.go` hover `want` strings were
  migrated to match.
- All 7 `compiler/lsp_*_test.go` files migrated (33 bound sites):
  g04 (6, incl. cursor contexts `emits [|]` → `emits {|}` at L404/L409),
  g05 (3), server (7), g03 (7), g02 (8: 3 fixture + 5 hover wants),
  g01 (2), wire (0). No error decls/ctors in LSP fixtures (verified).
- `compiler/current_*_test.go` + `compiler/package_instance_test.go`
  (G01 assigns `current_*` to P04): `current_parse_test.go` L18/L56
  (L56 keeps its trailing-`given` breakage; only the bound migrated),
  `current_format_test.go` L17, `current_types_test.go` L57 (decl + bound),
  `package_instance_test.go` L31/L34 (decls). No ctors (one `call
  network::must_not_run()` kept).
- Still needs shell: `go run ./tools/gramcheck` after P01, focused LSP tests.

## N05-ext — types/driver/project/browser suites (done, unverified)

G01 assigns these packages' embedded Can to N05 but the N05 passes covered
only check/resolve/emit. Swept shell-free:

- types: `symbolic_test.go` L32/L42/L48 (decls + bounds; L42's dynamic
  `"emits [" + bound + "]"` → braces; `bound` values are bare type lists),
  `builder_test.go` L100/L101/L150/L157, `infer_test.go` L23/L40/L79.
  (One self-introduced brace typo on infer L79 was caught and repaired
  in the same pass; shell tests must confirm.)
- driver: `source_maps_test.go`, `fixes_test.go`, `output_test.go`,
  `diagnostics_test.go` (incl. L220 astral unicode-escape line, anchored
  without touching the escape), `invoice_grid_test.go` L868/L1279/L1280/
  L1295/L1296. `records::flawed_save(...)` at L962/963 is a RECORD ctor
  (`record flawed_save`, `examples/invoice/src/records/records.can:127`) —
  parens kept, and the `contractReplaceOnce` anchors stay valid.
- project: `fixtures_test.go` L61/L64/L116/L221, `overlay_test.go` L34
  (`overlayMain`). `overlay_test.go:43` (`broken` fixture) INTENTIONALLY
  KEEPS `emits []`: the test asserts only SourceError structure for the
  `fn int main(` breakage; the bound is never reached. Do not "fix".
- browser: `browser_test.go`, 42 bound sites + 2 error ctors
  (`files::not_found("gone")` → braces at L121/L122, assert + terminal
  tail). All other `pkg::name(` hits are `call ...` operations; record
  ctors (`email()`, `point()`, `saved()`, `invoice_wire()`,
  `form::rejected<...>()`, `found()`) keep parens. Bare match arms
  (`codec::invalid_data`, ...) unchanged per N04.

## Tooling notes for the next shell session

- `muse.search` `mode` defaults to `literal`: patterns containing `|` must
  pass `mode: regex` or they silently match nothing. Several sweeps were
  re-run for this reason; results above are from corrected searches.
- Broad-path searches (`paths: ["compiler"]`) can omit hits that per-file
  searches find (observed: `overlay_test.go:43` missing from one
  compiler-wide sweep with no truncation notice). Re-verify "zero
  remaining" claims per file or with `rg` in shell.
- Transient EMFILE also affects `read_file`/`edit_file`/`search`
  intermittently; retries succeed. `muse.bash` fails 100%.
- `/tmp/can-n05/` helper scripts may not survive a reboot; N05 pass-1/2
  rules are described in the session log, not here.
- Observed, out of scope (do not touch without lane-owner check):
  `tools/performance/test_editor_experiment.py:73,75` still embeds
  `emits []` fixtures. Benchmarks are excluded by task constraints; leave
  for whoever owns tools/ (P01-adjacent, Muse lane).
- `.can` corpus state could not be assessed shell-free (glob search over
  `compiler,examples,std,shared` skipped every file). M01–M04 must start
  from `evidence/migration-map.md` + `evidence/migration-edits.txt` in
  shell — note those files are also absent from this tree (see header);
  if they are unrecoverable from git, the G02 map must be rebuilt before
  the applier runs.
