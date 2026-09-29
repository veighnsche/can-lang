# Error brace syntax: implementation checklist

Status: planned, not implemented. The user chose positional error payloads and
asked for the complete implementation plan. This document records scope and
acceptance; the [parallel-ready task list](tasks.md) controls execution order,
file ownership and the Muse handoff, followed by Codex review.

Supporting design: [three-form assessment](../../syntax-taste/preparation/jev-error-brace-triad-2026-09-29/findings.md), [actual-source comparison](../../syntax-taste/preparation/jev-error-brace-triad-2026-09-29/source-comparison.md), and the [three saved Jev requests, responses, and wording audit](../../syntax-taste/preparation/jev-error-brace-triad-2026-09-29/). Jev's answers disagreed; the source comparison, exact semantics, and verification below govern implementation. If implementation reveals a new difficult design choice, consult Jev three fresh times under `AGENTS.md` before deciding; save and audit every request and response.

## Target contract

| Role | Required Can spelling | Meaning |
| --- | --- | --- |
| Error declaration | `error missing{str key}`; `error empty{}` | Ordered, typed fields; generic parameters stay before `{`. |
| Finite domain bound | `emits {missing, other}`; `emits {}` | Set of declared domain error types in functions, native declarations, callable types, and choice-arm types. |
| Error value | `missing{"x"}`; `empty{}`; `all_failed<T>{[missing{"x"}]}` | Positional nominal data in every expression position. Braces alone never emit or throw. |
| Terminal completion | A terminal `missing{"x"}` | Emits a domain failure with the existing bound, origin, and source span rules. |
| Successful data | `ok wrapper(missing{"x"})` | Stores an error value inside a successful result. |

Keep record declarations and constructors, explicit calls, callable inputs,
constructor patterns, arrays, package headers, indentation blocks, bare match
heads, relay, and `[_]` standard-failure matching in their current forms.
`emits calculated` remains the special wrap-declaration form. `emits {}[]`
means an array of a callable whose error bound is empty. Delimited forms stay
on one physical source line; the current positional arity, type, spread, and
trailing-comma rules remain. No old spelling, compatibility parser, generated
layout alias, or old golden needs to survive. Generated TypeScript should keep
the existing native JavaScript/Bun lowering and only use adapters required by
Can's contracts and immutability.

Illustrative syntax fragments (not a standalone program):

```can
error missing{str key}
error empty{}
fn int load
    emits {missing}
    missing{"x"}
```

The parser must reject `error missing(str key)` and `emits [missing]`.
After name resolution, the checker must reject `missing("x")` and
`record_name{"x"}` while admitting error values nested in ordinary data.
`emits {}` must not admit a domain error, and must not change how standard
failures are handled.

## Execution rules

- Read `AGENTS.md` and capture current `HEAD`, `git status --short`, and active
  file ownership before edits. As of planning, unrelated work already touches
  catalogue mirrors, runtime files, `README.md`, and a current HTTP fixture.
  Preserve those changes and do not overwrite another task's files. Reuse a
  suitable existing worktree; if isolation is necessary, give the checkout an
  owner and a retirement step. A task waits on a true ownership conflict.
- Muse owns implementation. Codex coordinates, reviews the complete diff,
  checks evidence, and verifies the result. Use
  `muse exec --yolo --model muse-spark-1.3-contributor --workspace <repo> --worktree off --prompt-file <owned-temporary-prompt>`;
  the prompt must point to **[tasks.md](tasks.md)**, require progress and
  evidence against its task IDs, and direct Muse to continue every ready task
  in dependency order in one sustained run. Keep the configured reasoning
  effort. Verify current CLI help if needed. Do not start that run merely
  because this plan was written.
- Each checkbox is complete only when its acceptance check and concise
  evidence are recorded in its `Evidence:` line. Use bounded correctness
  commands with appropriate per-command timeouts. Do not run benchmarks,
  performance measurements, `go test ./...`, a distribution build, or a broad
  bundle while those are deferred.
- Register cleanup as soon as a task allocates a temporary directory or prompt;
  remove owned scratch on success, failure, and handled interruption. Recover
  abandoned **owned** scratch on the next run. Keep only compact counts, diffs,
  commands, and outcomes. Do not clear shared caches or delete foreign work.
  Do not hand-edit `runtime/catalogue.ts` or other generated catalogue mirrors.

## Lane A — baseline, inventory, and migration preparation

- [ ] **A01 — capture baseline and ownership.** Prerequisite: this plan.
  Record commit, dirty paths, available worktrees, tool versions, and the exact
  files another task currently owns. Resolve overlapping ownership before
  writing those paths. Ownership: this checklist plus compact
  `evidence/baseline.md`. Acceptance: a later diff can distinguish this syntax
  change from pre-existing work, and any checkout has a retirement owner.
  Evidence: pending.
- [ ] **A02 — classify every live source surface.** Prerequisite: A01.
  Inventory valid `.can` under `compiler/testdata/current`, `examples`,
  `tests`, `std`, `shared`, and `tools`; executable docs probes; embedded Can
  in Go tests; LSP samples; catalogue type strings and generated docs;
  current reference docs; and intentionally invalid fixtures. Treat dated
  preparation/evidence/archive files as history unless a test executes them.
  The initial search found 262 `.can` files, including 79 under `docs`, and
  many old `emits [` occurrences; recheck the live set rather than using those
  counts as a pass condition. Ownership: `evidence/inventory.md` only.
  Acceptance: every active positive, negative, generator, and documentation
  surface has a named owner and migration disposition. Evidence: pending.
- [ ] **A03 — prepare reviewed source edits.** Prerequisite: A02 and the old
  parser still available. Use lexer/parser spans to locate declarations and
  finite bounds. Classify constructor candidates using their resolved nominal
  type or a reviewed declaration/catalogue map; inspect ambiguous generic and
  qualified cases manually. Record file hashes and counts before edits. A
  temporary migration script may assist but must not become a production
  compatibility path. Do not apply a global parenthesis replacement or touch
  strings, comments, JSON arrays, calls, records, or constructor patterns.
  Ownership: owned temporary tool and `evidence/migration-map.md` only.
  Acceptance: a reviewable candidate map covers nullary, generic, nested,
  catalogue, successful-data, and terminal uses. Evidence: pending.

## Lane B — lexer, parser, AST, and formatter

- [ ] **B01 — admit balanced braces.** Prerequisite: A03. Update
  `compiler/internal/syntax/lexer.go` and `lexer_test.go` to tokenize `{}`,
  enforce matching and the existing single-line rule, and report unmatched,
  mismatched, unclosed, newline, and multiline-literal failures precisely.
  Braces in strings/comments remain inert. Acceptance: focused lexer tests
  cover valid nesting and each diagnostic; arbitrary brace blocks are not
  introduced. Evidence: pending.
- [ ] **B02 — parse declarations and finite bounds.** Prerequisite: B01.
  Update `syntax/declarations.go`, `parser.go`, and `native.go` paths using
  `errorBound()` so error declarations take typed fields inside `{}` and all
  finite bounds use `emits {}`. Preserve `emits calculated` on wraps, existing
  type validation, and `emits {}[]` ownership. Add parser tests for empty,
  multiple, nested callable, choice-arm, generic, native, and obsolete forms.
  Acceptance: old declaration/bound spellings fail parsing; all new forms
  round-trip through the syntax tree. Evidence: pending.
- [ ] **B03 — preserve constructor delimiter in the AST.** Prerequisite: B01.
  Update `syntax/ast.go`, `expressions.go`, and terminal detection in
  `declarations.go` to parse both constructor delimiters while recording which
  was authored. Extend `<...>` speculative constructor lookahead to `{`
  without breaking comparisons, nested `>>` fragments, or `()` record
  constructors. Admit `E{}` and positional/nested arguments; reject state
  groups and trailing commas as today. This is syntax only: the checker decides
  whether the resolved nominal is an error or record. Acceptance: tests cover
  nullary, generic, qualified, nested success-data and terminal constructs,
  and comparison rollback. Evidence: pending.
- [ ] **B04 — format the new grammar exactly.** Prerequisite: B02,B03.
  Update `syntax/format.go`, `parser.go` `FormatType`, and trivia formatting
  expectations. Emit braces from the recorded constructor delimiter, and use
  braces for declarations/bounds. Add parse → format → parse and
  `FormatTrivia` idempotence checks with same-line comments, UTF-8, CRLF,
  empty fields/bounds, generic nesting, and source spans. Acceptance: valid
  sources have one canonical spelling; formatting never silently changes the
  nominal meaning of a constructor. Evidence: pending.

## Lane C — semantic contract, diagnostics, and lowering

- [ ] **C01 — require the correct delimiter after resolution.** Prerequisite:
  B03. In `compiler/internal/check/expressions.go` and the relevant generic
  inference/resolution paths, require `{}` for resolved `types.Error` and `()`
  for `types.Record`, including catalogue, imported, generic, and expected
  variant cases. Keep ordered arity, field types, inference, and existing owner
  checks. Diagnostics should name the expected form at the source span.
  Acceptance: positive and negative checker tests exercise local, qualified,
  generic and catalogue constructors; `E(...)` and `R{...}` fail. Evidence:
  pending.
- [ ] **C02 — prove value versus completion semantics.** Prerequisite: C01.
  Verify a brace-built error inside `ok wrapper(...)`, an array, a binding,
  or a nested generic value remains ordinary frozen nominal data. Verify only
  a terminal `E{...}` creates a domain completion; forwarding a matched
  failure retains its prior occurrence, reconstruction creates a new value,
  and standard failures remain unconstructible/outside `emits {}`. Preserve
  existing `ir/regions.go`, `emit/expressions.go`, `emit/regions.go`, and
  runtime behavior unless a focused failing test demonstrates a needed fix.
  Acceptance: checker/emitter tests assert the payload, bound, origin, and
  relevant source-map span rather than a changed runtime helper. Evidence:
  pending.
- [ ] **C03 — update authored syntax in diagnostics and display.**
  Prerequisite: B02,C01. Change user-facing `emits [` text and source examples
  in `check/completions.go`, `check/program.go`, `check/action_bindings.go`,
  related diagnostics, and `compiler/internal/catalogue/types.go`. Review
  `driver/hover.go` and other AST renderers for old spelling. Preserve machine
  diagnostic identity/positions unless the new punctuation changes the span.
  Acceptance: focused diagnostic, catalogue type, and hover tests show braces
  and exact current offsets. Evidence: pending.

## Lane D — migrate live corpus and product surfaces

- [ ] **D01 — migrate valid Can files.** Prerequisite: A03,B04,C01 and
  ownership from A01. Apply the reviewed edits to live `.can` files under the
  active roots from A02. Use `canlc format --write` only after the new parser
  accepts the migrated file; review all changes to record/call syntax.
  Includes `compiler/testdata/current/coordination/aggregate-composition.can`
  and the retry/failure-conventions programs. Ownership: those live `.can`
  files only. Acceptance: each parses/checks where it did before; no old
  positive declaration, finite bound, or error construction remains.
  Evidence: pending.
- [ ] **D02 — migrate embedded Can and intentional negatives.** Prerequisite:
  D01. Update Can source in `compiler/internal/{syntax,resolve,check,emit}` Go
  tests, `compiler/lsp*_test.go`, `tests/integration`, `host/conformance`, and
  `tests/failure-conventions`. Keep deliberately invalid old spellings only in
  explicit rejection tests. For tests using `strings.Replace` or similar
  mutations, assert the replacement actually changed the source before
  checking the expected failure. Ownership: affected test files. Acceptance:
  negatives still test their intended rule, and no fixture silently becomes
  a no-op. Evidence: pending.
- [ ] **D03 — update catalogue-generated presentation.** Prerequisite:
  B02,C03 and no concurrent owner of generator outputs. Update
  `compiler/internal/catalogue/generate.go` to render finite error bounds
  and callback signatures with braces; update catalogue type tests. The
  structured `catalogue.json` error arrays and JSON registry arrays stay JSON.
  Run the generator once, inspect its four output paths against A01's dirty
  baseline, then run `make catalogue-check`. Never hand-edit
  `runtime/catalogue.ts` or `compiler/internal/catalogue/generated.go`.
  Acceptance: generated mirrors are coherent and unrelated catalogue edits
  survive. Evidence: pending.
- [ ] **D04 — update editor and LSP surfaces.** Prerequisite: B04,D02.
  Add `{}` pairing and appropriate tokenization in
  `editors/vscode/language-configuration.json` and
  `editors/vscode/syntaxes/can.tmGrammar.json`; update the pinned example in
  `tools/gramcheck/main.go`. Update LSP completion/hover/format fixtures,
  especially cursor contexts that currently use `emits [|]`. Acceptance:
  `go run ./tools/gramcheck` and focused `./compiler` LSP tests pass with
  brace positions and no lost completions. Evidence: pending.
- [ ] **D05 — update current documentation.** Prerequisite: B04,D01.
  Update the live grammar, examples and delimiter rule in
  `docs/syntax-taste/technical-spec.md`, `compiler/internal/syntax/README.md`,
  `README.md`, `REQUIREMENTS.md`, relevant current user guides, and active
  catalogue docs. Correct any claim that Can has no braces. Keep historical
  decision, probe, and Jev files intact; annotate a historical reference only
  if it is otherwise mistaken for current syntax. Acceptance: a new reader
  can distinguish error values, terminal failures, bounds, and unchanged
  `()`, `[]`, `[_]`, and `emits calculated`. Evidence: pending.

## Lane E — bounded verification, review, and cleanup

- [ ] **E01 — syntax and corpus gates.** Prerequisite: B01–B04,D01,D02.
  Run focused lexer/parser/formatter tests, then package-level
  `go test -p 1 ./compiler/internal/syntax ./compiler/internal/resolve` with
  a bounded timeout. Parse and format every active positive `.can` fixture;
  verify the new rendering reaches a fixpoint and the 59 current fixtures in
  the existing round-trip walker remain covered. Search active source, Go
  snippets, and docs for old forms, manually classifying any remaining hits
  as explicit negative or history. Acceptance: zero unexplained old positives
  and no parser/formatter regression. Evidence: pending.
- [ ] **E02 — checker, emitter, catalogue, and LSP gates.** Prerequisite:
  C01–C03,D03,D04,E01. Run targeted tests for constructor inference, exact bounds,
  completion origins, catalogue callable types, and LSP formatting/hover/
  completion; then bounded package checks for
  `./compiler/internal/check`, `./compiler/internal/emit`,
  `./compiler/internal/catalogue`, `./compiler`, and `./tools/gramcheck`.
  Run `make catalogue-check` and `go run ./tools/gramcheck`. Record skips and
  their environment conditions. Acceptance: the same failure payload and
  terminal behavior reaches generated TypeScript; display uses braces.
  Evidence: pending.
- [ ] **E03 — integration and runtime gate.** Prerequisite: E02. Run relevant
  `./tests/failure-conventions` and selective `./tests/integration` cases for
  retry, generic aggregation, callables, standard failure, and assertion
  location. Use an already configured Bun archive if a test requires one;
  do not build a new distribution for this task. If authored TypeScript under
  `runtime/` or `tools/runtime/` was edited, first run
  `bun run lint:fix:runtime` and `bun run format:runtime`, preserving test
  behavior and using narrow explained suppressions only if necessary. In all
  cases run `bun run check:runtime` before completion, with
  `bun run lint:runtime --format=agent` for compact lint diagnostics. Run
  relevant Bun tests only if runtime code or generated runtime behavior
  changed. Acceptance: bounded integration and required runtime checks pass,
  or an actual environment skip is recorded precisely. Evidence: pending.
- [ ] **E04 — independent review and retirement.** Prerequisite: E01–E03.
  Codex reviews the complete diff against A01, the migration map, and the
  target examples. Verify no compatibility branch, unrelated edit, generated
  file hand edit, or heavy retained artifact remains. Remove every owned
  temporary prompt, script output and workspace; report any cleanup failure.
  Record exact commands/results and remaining limitations here. Archive a
  completed managed worktree only after confirming no running task needs it
  and preserving unique work through the supported archive mechanism.
  Acceptance: a reviewable diff and compact evidence establish all three
  syntax changes and unchanged error semantics. Evidence: pending.

## Final completion criteria

The implementation is complete when the old declaration and finite-bound forms
are rejected; old error `()` construction and record `{}` construction fail
after resolution; all active valid source uses the new syntax; format, LSP,
catalogue, and current docs agree; empty, generic, nested, stored, terminal,
forwarded, and standard-failure cases pass; required runtime checks pass; and
temporary storage is reclaimed. No commit, push, benchmark, or deployment is
part of this plan.
