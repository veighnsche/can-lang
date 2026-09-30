# Muse handoff: implement the error-brace triad

Continue the authorized implementation in `/Users/vince/Projects/can-lang`.
This is an execution handoff, not another planning round. The complete ordered
task list is `docs/implementation/error-brace-triad/tasks.md`; the selected
design and acceptance criteria are in
`docs/implementation/error-brace-triad/checklist.md`. Read the repository
`AGENTS.md`, those documents, the linked source comparison, and Jev findings
before editing.

## Goal

Before delegating any work, inspect your native goal state with `get_goal`.
If the matching implementation goal already exists in this session, continue
it. If there is no matching goal, call `create_goal` with this full objective
and no token budget:

> Implement the complete error-brace triad task list in
> /Users/vince/Projects/can-lang and migrate all active Can files and embedded
> Can source to the new syntax, preserving unrelated work and verifying the
> completed implementation.

Never replace an unrelated active goal. Make the goal before delegating, keep
its progress aligned with evidenced checklist work, and mark it complete only
when every Muse-owned task is complete, writers are released, and the
implementation is ready for the independent Codex V04 review.

## Required syntax and meaning

* Error declaration: `error missing{str key}` and `error empty{}`.
* Finite error set: `emits {missing, other}` and `emits {}` wherever Can
  currently supports finite error bounds.
* Positional error value: `missing{"x"}` and `empty{}` in every expression
  position.
* Braces in a constructor expression create ordinary nominal data. Only a
  terminal error completion emits a domain failure.

Keep record declarations and constructors, explicit calls, callable inputs,
constructor patterns, arrays, package headers, indentation blocks, bare match
heads, relay, and `[_]` standard-failure matching in their existing forms.
Keep `emits calculated` as the special wrap form and preserve `emits {}[]`
precedence as an array of callables with an empty finite error set. Preserve
current positional arity, field types, spread and trailing-comma behavior,
source spans, native JavaScript/Bun lowering, and immutable Can contracts.
There is no backwards-compatible parser path.

## Preserve and inspect the interrupted work

First capture the current HEAD and `git status --short`. Do not assume the
baseline below is still current. An earlier run was stopped at the user's
request after completing G00, G01, and F01. The current tree contains uncommitted
work in at least:

* `compiler/internal/syntax/lexer.go`, `lexer_test.go`, and `README.md`
* `editors/vscode/language-configuration.json` and
  `editors/vscode/syntaxes/can.tmGrammar.json`
* `docs/syntax-taste/decisions.md` and `technical-spec.md`
* the task plan and evidence under `docs/implementation/error-brace-triad/`

Inspect the actual diff and evidence. Do not reset, revert, stash, or overwrite
these changes. The previous F01 entry reports a passing focused syntax test;
rerun or inspect the appropriate evidence before relying on it. G01's inventory
is at `docs/implementation/error-brace-triad/evidence/inventory.md` and the
baseline is at `docs/implementation/error-brace-triad/evidence/baseline.md`.
Keep the committed `html::email_href` implementation intact; migrate its Can
fixture syntax where the reviewed map requires it and let the catalogue
generator alone write generated mirrors.

## Implementation and commits

Complete all Muse-owned tasks G02 through V03 in `tasks.md`, in dependency
order, and migrate every active positive `.can` file, executable Can example,
and embedded positive Can source found by G01. Keep obsolete syntax only in
explicit rejection tests or classified historical material. Do not apply broad
punctuation replacements: use syntax spans and resolved nominal kinds so calls,
records, arrays, patterns, strings, comments, JSON, and successful error values
keep their meaning. Preserve negative-test intent, including assertions that
`strings.Replace` mutations actually matched. Leave V04 for Codex review.

Commit early and often. After each small task or independently passing vertical
slice, inspect the diff, run its bounded focused checks, stage only the exact
owned paths, inspect the staged diff, and make a small Conventional Commit.
Include the relevant tests and docs with the change they verify. Do not wait
until the entire task list is done to make the first commit. Do not use
`git add -A`, do not include unrelated or Codex-owned monitor state, and do not
commit a failing or unreviewed diff. If a committed slice needs repair, make a
follow-up commit; do not rewrite history. Never push unless the user separately
asks.

Use the four lanes and file leases in the task list. You are the only
coordinator; keep at most two code writers active, assign exclusive file
ownership, prevent concurrent task-list edits, and serialize Go/Bun test
commands. Let every ready task advance until all Muse-owned tasks are complete
or a genuine blocker remains. Do not impose a total-step cap or manufacture
work. For a new difficult design decision, follow `AGENTS.md`: make three fresh
Jev consultations with all explanatory prose reworded while preserving facts
and identifiers, save requests/responses, and investigate disagreements.

## Verification and handoff

Respect all `AGENTS.md` disk and laptop-load limits. Do not benchmark, run
`go test ./...`, build a distribution bundle, or perform broad measurements.
Run the bounded V01–V03 checks specified in the task list. If authored runtime
TypeScript changes, run `bun run lint:fix:runtime`, `bun run format:runtime`,
then `bun run check:runtime` and relevant runtime tests. Do not hand-edit
`runtime/catalogue.ts` or other generated catalogue mirrors.

Register cleanup immediately for every temporary artifact and clean it on
success, failure, or handled interruption. Record concise command outcomes in
the task evidence and each completed task's Evidence field. Before handoff,
verify active positive Can syntax has migrated, relevant checks pass, the diff
preserves unrelated work, all writers are released, and no owned temporary
artifacts remain. Leave V04 unchecked for independent Codex review.
