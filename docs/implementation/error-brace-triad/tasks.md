# Error brace syntax: parallel-ready task list

Status: **planned; no implementation has started**. This is the execution order
for the [implementation plan](checklist.md). The [chosen source-level design](../../syntax-taste/preparation/jev-error-brace-triad-2026-09-29/source-comparison.md)
and [Jev review](../../syntax-taste/preparation/jev-error-brace-triad-2026-09-29/findings.md)
define the syntax and semantic contract. Use the IDs below for progress and
evidence; the older A–E IDs in `checklist.md` remain background detail.

## Dispatch rules

- **Critical path:** freeze a source-aware migration map while the old parser
  still works; then implement lexer/parser/constructor AST; then checker and
  formatter; then converge the migrated corpus; then run bounded checks.
- Four implementation lanes can advance on disjoint files. A single sustained
  Muse run should take the next ready task instead of waiting on another lane.
  If separate workers are later used for wall-time savings, keep at most two
  active code writers by default on this MacBook, give each a named file lease,
  and serialize Go/Bun test commands. Do not create disposable worktrees for
  each lane. The master checklist and evidence files remain the source of
  truth; do not let concurrent workers edit the same checklist lines.
- Every worker reads `AGENTS.md`, this task list, and the linked design. Give
  Muse the **actual task-list path** in its prompt, use the configured
  `muse-spark-1.3-contributor` CLI and authorized `--yolo` mode, and keep each
  run sustained through its ready tasks. Codex coordinates leases, reviews
  work and evidence, and runs the final checks. No implementation is authorized
  merely by writing this list.
- Do not keep old syntax for compatibility. Keep record constructors/calls in
  `()`, arrays in `[]`, constructor patterns and bare match heads unchanged,
  and `emits calculated` on wraps. An error constructor creates data; only a
  terminal completion emits a failure. Preserve native JavaScript/Bun
  lowering and immutable Can contracts.
- Each task needs its stated acceptance check and a short `Evidence:` entry
  before it is checked. If parallel workers are used, each writes
  `evidence/<lane>.md`; Codex copies the result into the relevant task entry
  at a handoff. Register and reclaim task-owned temporary storage immediately.
  No benchmarks, broad builds, full bundles, or `go test ./...` while deferred.
  In every migrated Go test package, a `strings.Replace`-style negative test
  must assert that its source mutation actually matched and changed the input.

## File leases and collision points

| Lane | Exclusive edit ownership | Handoff or existing conflict |
| --- | --- | --- |
| **F · syntax** | `compiler/internal/syntax/lexer.go`, `parser.go`, `native.go`, `format.go`, `format_trivia.go`; `lexer_test.go`, `parser_test.go`, `declarations_test.go`, `format_trivia_test.go` | Owns `declarations.go` through F02, then hands it to N02. Excludes N-owned `expressions_test.go` and `constructor_braces_test.go`. |
| **N · constructor and semantics** | `syntax/ast.go`, `expressions.go`, `expressions_test.go`, new `constructor_braces_test.go`; `compiler/internal/check/**`, `resolve/**`, and focused `emit/**` code/tests | Owns `declarations.go` only during N02. `emit/regions.go` and other emitter paths are already dirty; preserve their existing owner and edit production lowering only if a failing semantic test requires it. |
| **M · live corpus** | Active `.can` in `compiler/testdata/current`, `examples`, `tests`, `std`, `shared`, `tools`, and executable probes under `docs`; embedded Can in `tests/integration`, `tests/failure-conventions`, `host/conformance` Go tests | `compiler/testdata/current/http/main.can` is already dirty. Do not edit syntax/check/emit Go tests or `compiler/lsp*_test.go`; those belong to F, N, and P. |
| **P · product surfaces** | `compiler/internal/catalogue/**` and its four generated outputs; `compiler/lsp*_test.go`, `compiler/internal/driver/hover.go` if needed, `editors/vscode/**`, `tools/gramcheck/**`, current documentation | Catalogue JSON/mirrors, `README.md`, and `docs/user-guide/` already have unrelated changes. The generator alone writes `generated.go`, `runtime/catalogue.ts`, `std/catalogue/README.md`, and `std/catalogue/errors.json`. |

The task currently has unrelated dirty work. G00 must identify its live owner;
the table is an intended lease, not permission to overwrite that work. The
syntax and constructor lanes may edit different files in the same directory.
Any file not listed explicitly gets one owner before editing.

## Ready windows

| Window | Work that can overlap | Barrier to the next window |
| --- | --- | --- |
| **0** | G00 only. | Dirty-file owners and checkout settled. |
| **1** | F01 lexer, G01 inventory, and uncontested P01 editor/spec work. | F01 and G01 complete; G02 records the migration map while old syntax parses. |
| **2** | F02 declarations/bounds, N01 constructor AST/expressions, M01–M03 source migration, and remaining P01. | F02 releases `declarations.go`; N01 fixes the AST contract. Temporary parse failures in the shared tree are expected here; do not run broad tests. |
| **3** | N02 terminal detection; F03 formatter; N03 checker; P02 catalogue authoring. Then N04–N05, F04, M04, and P03–P05 as their dependencies clear. | All edited source, generated mirrors, docs, and test snippets converge. |
| **4** | V01 → V02 → V03 → V04. | Serialize heavier verification and review; clean scratch and retire an owned worktree only when free. |

These are readiness windows, not a demand to start four Muse processes. Work
on an independent ready task when a dirty-file lease or dependency blocks
another one. Keep the compiler/test runner single-file and `-p 1` where
appropriate to protect laptop load.

## Gate G — baseline and migration map

- [ ] **G00 · baseline and leases.** Depends: none. Record `HEAD`,
  `git status --short`, usable worktree, relevant tool versions, and owners of
  currently dirty files. Keep unrelated changes intact. Files: this list and
  `evidence/baseline.md` only. Done when the source diff can be separated from
  pre-existing work and every contested path has a release or isolation plan.
  Evidence: pending.
- [ ] **G01 · inventory the live syntax surface.** Depends: G00. Classify
  active `.can`, executable docs probes, Go-embedded Can, intentional negative
  fixtures, LSP examples, catalogue displays, generated outputs, and current
  docs. Exclude dated evidence from mechanical rewriting unless tests execute
  it. Files: `evidence/inventory.md` only. Done when each live surface has an
  owning lane and a migration disposition. Evidence: pending.
- [ ] **G02 · freeze a reviewed migration map.** Depends: G01,F01. While the
  old parser still accepts old source, use parser/lexer spans for declarations
  and bounds; classify each constructor by resolved nominal kind or a reviewed
  declaration/catalogue map. Record source hashes and ambiguous cases. A
  one-off helper stays temporary and is cleaned after use. Files: owned scratch
  plus `evidence/migration-map.md`. Done when nullary, generic, qualified,
  nested, catalogue, successful-data, and terminal candidates are covered;
  calls, records, patterns, strings, comments, and JSON are excluded.
  Evidence: pending.

## Lane F — lexical grammar and canonical formatting

- [ ] **F01 · balanced one-line braces.** Depends: G00. Files:
  `syntax/lexer.go`, `lexer_test.go`. Admit `{}` and enforce current delimiter
  matching, newline, and multiline-literal rules. Done when focused tests
  cover valid nesting, mismatches, missing closers, strings/comments, and the
  one-line diagnostic without introducing general brace blocks.
  Evidence: pending.
- [ ] **F02 · error declarations and finite bounds.** Depends: G02.
  Files: `syntax/parser.go`, `declarations.go`, `native.go`,
  `parser_test.go`, `declarations_test.go`. Parse `error E{fields}` and all
  finite `emits {types}` forms; preserve `emits calculated` and
  `emits {}[]` callable-array precedence. Done when new empty/generic/nested
  forms parse and old declaration/bound spellings fail. Release
  `declarations.go` to N02 after this task. Evidence: pending.
- [ ] **F03 · canonical rendering and trivia.** Depends: F02,N01.
  Files: `syntax/format.go`, `parser.go`, `format_trivia.go`,
  `format_trivia_test.go`. Render braces for error declarations/bounds and
  the AST-recorded constructor delimiter; preserve comments, CRLF, UTF-8,
  source spans, and fixpoint formatting. Done when focused parse → format →
  parse and trivia tests cover empty, generic and nested forms.
  Evidence: pending.
- [ ] **F04 · syntax-owned fixture and rejection sweep.** Depends:
  F03,N02. Files: F-owned syntax package tests only; do not edit N-owned
  `expressions_test.go` or `constructor_braces_test.go`. Migrate positive
  embedded Can there; keep old forms only as named rejection cases. Exercise
  terminal `E{}` and generic lookahead versus comparisons. Done when focused
  syntax tests pass and no positive syntax test still uses an old spelling.
  Evidence: pending.

## Lane N — constructor meaning and failure semantics

- [ ] **N01 · constructor AST and expression parser.** Depends: G02,F01.
  Files: `syntax/ast.go`, `expressions.go`, `expressions_test.go`. Record
  whether a constructor used `()` or `{}`; admit `E{}` and `E<T>{...}`;
  extend generic lookahead to braces without breaking `<` comparisons or
  record constructors. Do not touch `declarations.go` yet. Done when focused
  expression tests cover qualified, nested, empty, and comparison cases.
  Evidence: pending.
- [ ] **N02 · terminal constructor detection.** Depends: F02,N01.
  Files: `syntax/declarations.go` only after F02 releases it, plus new N-owned
  `syntax/constructor_braces_test.go`. Recognize
  brace constructors at terminal statement/completion positions while keeping
  success-data construction and bare match forwarding distinct. Done when
  dedicated tests reach the existing `FailureBody` and preserve source spans.
  Return the file lease to F for final syntax integration. Evidence: pending.
- [ ] **N03 · nominal delimiter checks.** Depends: N01. Files:
  `check/expressions.go`, `check/infer.go`, relevant resolve code and new
  focused checker tests. After resolution, require `{}` for `types.Error` and
  `()` for `types.Record`, including imported, catalogue, generic, and
  expected-variant cases. Keep positional arity, field types, inference and
  owner rules. Code can proceed alongside F02; run its source-level
  acceptance tests after F02 supplies the new declaration/bound grammar.
  Done when `E(...)` and `R{...}` fail with useful spans and valid
  constructions type-check. Evidence: pending.
- [ ] **N04 · ordinary value versus emitted failure proof.** Depends:
  N02,N03. Files: focused `check`/`emit` tests; production IR/emitter/runtime
  only for a demonstrated failure and after resolving existing dirty owners.
  Prove nested `E{...}` stays frozen data, terminal `E{...}` emits with the
  right bound/origin, forwarding retains an occurrence, reconstruction builds
  a value, and standard failures remain outside finite bounds. Done when
  checker-to-emitter assertions confirm these contracts. Evidence: pending.
- [ ] **N05 · diagnostics and owned Go snippets.** Depends: F02,N03.
  Files: `check/completions.go`, `program.go`, `action_bindings.go` and
  affected `check`, `resolve`, and `emit` Go tests. Replace user-facing
  `emits [` wording and migrate positive embedded source; retain old forms
  only in explicit negative tests. Do not touch catalogue or LSP files.
  Done when diagnostics show braces with correct offsets and no replacement
  based negative test silently becomes a no-op. Evidence: pending.

## Lane M — live Can corpus and external harnesses

- [ ] **M01 · current fixtures and shared libraries.** Depends: G02 and
  release of contested files from G00. Files: active `.can` under
  `compiler/testdata/current`, `std`, `shared`. Apply the reviewed map,
  including generic aggregate and nested catalogue errors; leave calls and
  records in `()`. Done when candidate edits are checked against original
  hashes and every change is classified. Evidence: pending.
- [ ] **M02 · examples and executable programs.** Depends: G02. Files:
  active `.can` under `examples`, `tests`, `tools`, plus any `.can` under
  `docs` that G01 marked as executable. Migrate declarations, finite bounds,
  and error values; keep historical docs probes untouched. Done when all
  active positives use the
  new spellings and no record/call/pattern delimiter changed accidentally.
  Evidence: pending.
- [ ] **M03 · external Go-embedded source.** Depends: G02. Files:
  `tests/integration`, `tests/failure-conventions`, `host/conformance` Go
  tests only. Rewrite positive snippets and ensure any `strings.Replace`
  mutation actually finds its target. Done when test intent is preserved and
  obsolete syntax appears only in explicit rejection cases.
  Evidence: pending.
- [ ] **M04 · corpus parse/format audit.** Depends: M01–M03,F03,N03.
  Files: M-owned corpus and compact `evidence/corpus.md`. Run the new
  formatter only after it accepts a migrated file; compare its output to a
  second pass and inspect any unrelated source diff. Search live Can and
  external snippets, including executable docs probes, for old forms,
  classifying intentional negatives.
  Done when every active positive parses/checks where it did before and
  no unexplained old spelling remains. Evidence: pending.

## Lane P — catalogue, editor, LSP, and current docs

- [ ] **P01 · early editor and spec update.** Depends: G00. Files:
  `editors/vscode/{language-configuration.json,syntaxes/can.tmGrammar.json}`,
  `tools/gramcheck/main.go`, `docs/syntax-taste/technical-spec.md`,
  `compiler/internal/syntax/README.md`, and uncontested live spec text.
  Add brace pairing/highlighting and update the three source forms; retain
  `emits calculated`, arrays, headers, calls and matches. Done when these
  files describe the chosen contract without treating a plan as shipped code.
  Evidence: pending.
- [ ] **P02 · authored catalogue presentation.** Depends: F02. Files:
  `compiler/internal/catalogue/types.go`, `generate.go`, and catalogue tests
  after resolving any pre-existing owner. Render concrete callable/choice
  types and generated documentation bounds with braces. Keep structured
  JSON arrays as JSON. Done when targeted type/display tests expect the new
  spelling and the generator is ready. Run final focused type tests after F03
  supplies canonical `FormatType` output. Evidence: pending.
- [ ] **P03 · regenerate catalogue mirrors.** Depends: P02,F03 and G00 owner
  release for every output. The generator alone writes
  `compiler/internal/catalogue/generated.go`, `runtime/catalogue.ts`,
  `std/catalogue/README.md`, `std/catalogue/errors.json`. Inspect all four
  diffs against the dirty baseline; never hand-edit mirrors or discard
  unrelated catalogue work. Done when `make catalogue-check` passes and
  generated outputs are coherent. Evidence: pending.
- [ ] **P04 · LSP rendering and completion.** Depends: F03,N02,N03,P01.
  Files: all `compiler/lsp*_test.go` and `compiler/internal/driver/hover.go`
  only if needed. Migrate hover/format/completion snippets and cursor
  contexts, especially `emits {|}`; check source spans. Run
  `go run ./tools/gramcheck` after P01. Done when focused LSP tests show
  brace syntax and no lost completion. Evidence: pending.
- [ ] **P05 · remaining live documentation.** Depends: M01,M02 and G00
  owner release. Files: `README.md`, `REQUIREMENTS.md`, current user guides
  and other live prose/examples; leave generated catalogue README to P03.
  Correct the claim that Can has no braces. Keep dated research and Jev
  evidence historical. Done when current docs distinguish error data from
  emitted completion and agree with migrated examples. Evidence: pending.

## Gate V — serialized verification and handback

- [ ] **V01 · front-end and corpus gate.** Depends: F04,N05,M04.
  Run focused lexer/parser/formatter/resolve tests, then bounded
  `go test -p 1 ./compiler/internal/syntax ./compiler/internal/resolve`.
  Confirm all current positive fixtures reach a format fixpoint, including
  the existing 59-file round-trip walker. Record commands, timeouts, skips
  and any remaining old-form hits in `evidence/verification.md`.
  Evidence: pending.
- [ ] **V02 · semantics and product gate.** Depends: V01,N04,P03,P04.
  Run focused constructor, bound, origin, catalogue and LSP tests; then
  bounded checks for `./compiler/internal/check`, `./compiler/internal/emit`,
  `./compiler/internal/catalogue`, `./compiler`, and `./tools/gramcheck`.
  Run `make catalogue-check` and `go run ./tools/gramcheck`. Serialize
  heavier commands. Done when checker/emitter contracts and displayed syntax
  agree. Evidence: pending.
- [ ] **V03 · integration and runtime gate.** Depends: V02,M03,P05.
  Run relevant failure-conventions and selective integration cases using an
  already configured Bun archive. If authored `runtime/` or `tools/runtime/`
  TypeScript changed, run `bun run lint:fix:runtime` and
  `bun run format:runtime` first. In all cases run
  `bun run check:runtime`; use `bun run lint:runtime --format=agent` for
  compact diagnostics and only relevant Bun tests if runtime behavior changed.
  No performance measurements or full distribution bundle. Done when the
  bounded end-to-end cases pass or an actual environment skip is recorded.
  Evidence: pending.
- [ ] **V04 · independent diff review and cleanup.** Depends: V01–V03.
  Codex reviews the complete diff and evidence against the design: no old
  compatibility path, unintended call/record change, generated-file hand
  edit, or unrelated overwrite. Clean owned prompts, migration scratch,
  outputs and workspaces on success/failure; report cleanup failures. Archive
  a free managed worktree only through its supported archive mechanism after
  preserving unique work. Done when the result is reviewable, the exact
  checks are recorded, and no task-owned heavy artifact remains.
  Evidence: pending.

Completion means `error E{fields}`, `emits {E}`, and positional `E{value}`
are implemented throughout active source; old positive forms fail at the
right parser/checker boundary; stored error values, terminal failures, and
standard failures retain their established semantics; product surfaces agree;
bounded checks pass; and task-owned scratch is gone.
