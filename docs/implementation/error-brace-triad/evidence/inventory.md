# G01 inventory of the live syntax surface

Date: 2026-09-29. Baseline HEAD `4393a89e`. Counts rechecked live; A02's 262-file
total confirmed (183 active + 79 docs).

## Active `.can` (lane M)

- 183 files under `compiler/testdata`, `examples`, `tests`, `std`, `shared`, `tools`.
- `emits [` hits in ~120 files (list from `search emits [`); `error Name(` declarations
  in 11 `.can`: `compiler/testdata/current/generics/chain-core.can`,
  `compiler/testdata/current/project/vendor/sample/src/shared.can`,
  `tests/failure-conventions/{owner-negative,owner-forge,retry,retry-fixed,retry-negative,owner-setup}`
  profiles/billing/ids files.
- M01: `compiler/testdata/current`, `std`, `shared` (incl. `html/main.can`; preserve
  email-link lines, migrate only syntax spans).
- M02: `examples`, `tests`, `tools`, executable docs probes below.
- `emits calculated` (keep): `compiler/testdata/current/wrap/main.can`.
- Intentional negatives stay negative: `tests/failure-conventions/owner-negative`,
  `tests/failure-conventions/retry-negative`, `tests/integration/testdata/gallery-failing`.
  Old spellings there survive only as named rejection cases.

## Executable docs probes (parsed or built by tests — migrate despite age)

- `docs/syntax-taste/technical-spec.md` §C10 (2x `emits []`) and §Consumer:
  parsed by `syntax/declarations_test.go` (`specificationProgram`). Owner P01 text, F test.
- `docs/syntax-taste/decisions.md` `## Packages` (13 complete ```text examples incl.
  callable `emits [...]`, `emits []`): parsed by `TestCompleteCoreDecisionExamples`.
  Owner P01/P05 text, F test. Rest of decisions.md is prose/history.
- `docs/syntax-taste/evidence/2026-09-22/implementation-gaps/projects/*/src/main.can`
  (13 frozen projects, `emits []` + one finite bound): copied and built by
  `tests/integration/verified_build_test.go` when `CAN_BUN_ARCHIVE` is set. Owner M02.
- All other 79-docs `.can` and dated evidence/probe/Jev-prep files: historical, excluded
  from mechanical rewriting.
- `t26/t27` baseline tests assert on prose/audit files, not Can syntax, except the
  t27 numbered-error scan pinned to `syntax/declarations_test.go` + `check/checks_test.go`:
  new tests must not introduce `error\s+[0-9]` spellings elsewhere.

## Go-embedded Can

- F: `compiler/internal/syntax/*_test.go` except N-owned `expressions_test.go` and new
  `constructor_braces_test.go`; authored strings in `parser.go`, `format.go`, `native.go`.
- N: `syntax/expressions_test.go`, `constructor_braces_test.go` (new), `check/**`,
  `resolve/**`, focused `emit/**` tests; authored `emits [` wording in
  `check/completions.go`, `check/program.go`, `check/action_bindings.go`, `check/form.go`.
  Unlisted compiler-internal suites with embedded bounds (`types`, `driver` except
  `hover.go`, `project`, `browser`) assigned to N05 as affected Go tests (one owner each,
  recorded here per lease rule).
- M03: `tests/integration/*_test.go`, `tests/failure-conventions/*_test.go`,
  `host/conformance/{w2,w2_e2e,admission}_test.go`. Every `strings.Replace` mutation must
  assert it changed its input.
- P04: `compiler/lsp_*_test.go`, `compiler/current_*_test.go` (current-fixture product
  surface, assigned here), `driver/hover.go` iff it renders old syntax.
- `tools/performance/drivers/compiler.py`, `tools/performance/test_editor_experiment.py`
  generate `emits []` programs: live generators, owner M02.

## Catalogue, editor, LSP, docs (lane P)

- P02: `catalogue/types.go`, `catalogue/generate.go`, catalogue tests (concrete
  callable/choice display bounds with braces; structured JSON arrays stay JSON).
- P03 (generator writes only): `catalogue/generated.go`, `runtime/catalogue.ts`,
  `std/catalogue/README.md`, `std/catalogue/errors.json`. Inspect diffs vs G00 baseline;
  never hand-edit; `html::email_href` must survive.
- P01: `editors/vscode/{language-configuration.json,syntaxes/can.tmGrammar.json}`
  (no brace rules today), `tools/gramcheck/main.go` (pinned example + comment),
  `docs/syntax-taste/technical-spec.md`, `compiler/internal/syntax/README.md`.
- P05: `README.md`, `REQUIREMENTS.md`, current user guides, `tests/failure-conventions/README.md`,
  `tests/failure-conventions/x-r07-1.md`, `examples/account-search/README.md`;
  correct any "no braces" claim; leave generated catalogue README to P03 and dated
  research/Jev evidence historical.

## Disposition summary

Every live positive (`.can`, executed probe, embedded snippet, generator, display, doc)
migrates to `error E{...}` / `emits {...}` / `E{...}`; old forms remain only in explicit
rejection tests or clearly classified historical material. Calls, records, arrays,
patterns, strings, comments, JSON, and successful error data keep their meaning.
