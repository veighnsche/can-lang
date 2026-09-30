# Recovery runbook: shell EMFILE outage (2026-09-30)

`muse.bash` fails every call with `Too many open files (os error 24)`;
a probe subagent also failed (model transport error). File tools work.
All work below F02's commit `a45f3af4` is staged in the tree, self-reviewed
by reading, but NOT test-verified. Do not check N01/N02/F03/N03 until the
recovery checks pass.

## Staged, unverified tree changes (all by Muse, this session)

- N01: `syntax/ast.go` (+`Braces` on `ConstructorExpr`),
  `syntax/expressions.go` (`primary` admits `{`, `constructorTypes`
  accepts `{`, new `constructorArguments`), `syntax/expressions_test.go`
  (`TestBraceConstructorsRecordDelimiter`, `Braces` assert, 4 new rejects).
- N02: `syntax/declarations.go` (`startsConstructor` admits `{`; F02
  released the file), new `syntax/constructor_braces_test.go` (5 tests).
- F03: `syntax/format.go` (render recorded delimiter),
  `syntax/format_trivia_test.go` (`TestFormatTriviaBraceConstructors`).
- N03: `check/expressions.go` (delimiter-vs-kind enforcement),
  new `check/constructor_delimiter_test.go` (valid program + 9 negatives).

## Recovery sequence (shell required)

1. `echo ok` — shell back.
2. `git status --short` — expect only the 9 paths above modified/new,
   plus untracked `evidence/muse-monitor.md` (Codex-owned, do not touch)
   and `muse-prompt.md` (leave untracked).
3. `go test -p 1 -count=1 ./compiler/internal/syntax/` — must pass.
4. Focused: `go test -p 1 -count=1 -run
   'TestBraceConstructorsRecordDelimiter|TestTerminalBrace|TestBraceValueInside|TestBareErrorForwarding|TestFormatTriviaBraceConstructors'
   ./compiler/internal/syntax/ -v`.
5. `go test -p 1 -count=1 -run TestConstructorDelimiterRequiresResolvedKind
   ./compiler/internal/check/` — must pass. If the valid program fails,
   read the diagnostic first; likely candidates: `all_failed<outcome>`
   constraint, `leaf::` cross-file visibility, `held<missing>` inference.
6. Load `bundled:git`, then commit per slice (inspect each staged diff):
   - N01: ast.go, expressions.go, expressions_test.go
   - N02: declarations.go, constructor_braces_test.go
   - F03: format.go, format_trivia_test.go
   - N03: check/expressions.go, constructor_delimiter_test.go
   Mark each task `[x]` in `tasks.md` with its Evidence line in its commit.
7. Resume `tasks.md`: N04, N05, F04, M01–M04 (map applier: verify
   sha256:12 from `migration-map.md`, apply `migration-edits.txt` offsets
   per file, assert old bytes), P02–P05, V01–V03. Leave V04 for Codex.

## Map applier sketch (python3, no new deps)

Parse `migration-edits.txt` lines `path:line:col off=N "o"->"n" role`;
group by path; for each target file assert current sha256[:12] equals the
`migration-map.md` row, assert each offset holds `o`, write bytes with `n`.
Skip `diagnostic-parse` (historical) and the 10 unexecuted frozen projects;
hand-edit `lexer/core.can:15` (`emits []` → `emits {}`) keeping line 19 red.
