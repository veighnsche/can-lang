# M04 corpus parse/format audit

Date: 2026-09-30. Corpus: 169 `.can` files from M01/M02 commits (no `.can`
added since the G02 freeze — verified by tree diff).

## Format fixpoint (`canlc format`, read-only audit, nothing written)

- 168/169 reach a format fixpoint (pass1 == pass2 byte-identical).
- 1 intentional parse failure: `compiler/testdata/current/lexer/core.can:19`
  (parenthesized-literal negative; line-15 bound migrated, failure intact).
- 63 files differ pass1-vs-original: ALL pre-existing canonicalization drift,
  none brace-related — assertion `=>` alignment (60), generic-arg spacing
  (`std/map`), trailing whitespace (`examples/webhook` model). Files left
  untouched; the 59-file `TestFormatTriviaTestdataRoundTrip` walker stays green.

## Residual old-spelling classification (all explained, none active)

- `error E(`: zero in live corpus and live Go (2 rejection lines in
  `syntax/declarations_test.go:387-388` assert parser rejection).
- `emits [` in live `.can`: only `examples/language-site/src/site.can:95,142`
  inside displayed-sample string literals (P05 owns the display decision).
- `emits [` in Go: only `project/overlay_test.go:43` (asserts SourceError
  structure, bound never reached), `syntax/parser_test.go:70` and
  `syntax/declarations_test.go` rejection cases.
- Frozen: 9 non-executed `implementation-gaps` projects + `diagnostic-parse`
  stay historical per G02; 3 executed ones migrated.
- Generated: `std/catalogue/README.md` old bounds regenerate in P03.
- Docs prose/values: inventoried for P05 (see tasks.md).
- Checks: compiler unit suites (check/emit/driver/browser/catalogue/LSP)
  pass over the migrated corpus; `tests/` fixtures parse (fixpoint above) —
  full semantic check of FC/integration projects awaits a Bun archive (V03).
