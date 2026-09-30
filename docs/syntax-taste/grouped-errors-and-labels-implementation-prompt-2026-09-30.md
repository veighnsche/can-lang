# Implementation prompt

Implement grouped completion errors and grouped assertion/fixture labels in
`/Users/vince/Projects/can-lang`. Complete the existing partial implementation;
do not stop at another plan. Follow the repository's `AGENTS.md` and read:

- `/Users/vince/Projects/can-lang/docs/syntax-taste/grouped-errors-and-labels-implementation-plan-2026-09-30.md`
- `/Users/vince/Projects/can-lang/docs/syntax-taste/grouped-errors-and-labels-implementation-tasks-2026-09-30.md`

The design defines semantics; the 23-task checklist defines dependencies,
ownership and acceptance. The current draft is unverified. Inspect the working
tree before editing, preserve unrelated work, and finish the existing draft
rather than starting over. Keep implementation within the selected scope.

## Execution

1. Complete P01–P02 first. Record the starting HEAD, relevant pre-existing
   changes, shared interfaces and exclusive file owners. Split the mixed checker
   test file as specified before dispatching concurrent writers.
2. Use one coordinator and at most three workers in the existing checkout.
   Run syntax, error checking and assertion checking concurrently after the
   interface handoff. Reuse freed workers for execution tests and editor support.
   Start ready tasks immediately; do not add global wave barriers. The
   coordinator owns this checklist, integration, Git and the test queue.
3. Follow the file ownership rules. Workers must request changes to another
   lane's files through its owner. Keep compiler/Bun validation serialized and
   freeze the relevant input files during each command. Independent authoring
   can continue outside that set.
4. Continue through implementation, bounded validation, independent review,
   corrections and cleanup. Record task IDs, changed files, commands/results,
   commit hashes and limitations in the checklist or a linked compact evidence
   record. Distinguish authored work from passing acceptance gates. If genuinely
   blocked, finish independent ready work and report the precise blocker without
   claiming completion.

## Commit early and often

Make the first commit immediately after the initial inventory and review of the
task's existing draft, before substantial new implementation. Preserve the
identified task-owned draft and its design/evidence in an explicitly labelled
checkpoint; state which parts remain unvalidated. Do not wait until the whole
feature works before creating history.

Then commit each small, coherent milestone as soon as it is reviewable: interface
and ownership preparation, syntax, error semantics, assertion semantics,
behavioral tests, editor support, documentation and review fixes. Split or group
these according to actual dependencies and diff size; avoid both one enormous
final commit and artificial empty commits. Prefer passing relevant checks for
completed milestones. If a useful checkpoint precedes an integration gate,
clearly record that its checks are pending or blocked; a checkpoint is not
acceptance. Keep later fixes in additional commits.

Only the coordinator stages and commits. Before each commit, briefly pause
writers of the included files, inspect the exact diff, stage explicit task paths
or hunks, and verify the staged patch. Preserve unrelated working-tree and staged
changes; never use a blanket stage-all command. Workers report ready changes and
evidence instead of competing for the Git index. Use concise Conventional Commit
messages, for example `feat(check): validate grouped completion heads`. Do not
amend or rewrite existing history just to tidy the sequence, and do not push,
merge or open a PR as part of this prompt.

Record a verified **review base before the first checkpoint**, so the eventual
review includes the inherited draft as well as newly written code. If earlier
commits already contain part of the feature, identify and explain the appropriate
earlier base. Do not blindly assume the historical `5804b686` is still correct.
At completion record the actual full base and final HEAD hashes, the commit
sequence and any remaining uncommitted files. Commit all task-owned completed
changes while leaving unrelated work intact.

## Required behavior and verification

Preserve exhaustive exact-error checking. Bare groups must forward the original
protected completion, payload and occurrence identity. A shared handler is
checked once and retains one lexical fixture/call site; grouped handlers have
no error-specific payload aliases. Grouped test labels produce independent
roots/selectors, reports, contexts and FIFO queues. Audit generic alternatives,
coordination/`all_failed`, recovery state, scenarios/templates, raw/native modes
and source-only consumers as specified in the checklist.

Run its focused syntax, checker, emitted Bun, driver and TextMate checks, plus
appropriate nearby regressions. Prove occurrence/payload preservation and fixture
behavior through execution, not only generated text or IR pointer equality.
Use `GOMAXPROCS=2`, `go test -p=2`, the existing shared cache and bounded commands.
Do not launch broad builds, benchmarks, performance measurements or private cold
caches. Follow required runtime lint/format rules if runtime TypeScript changes;
run `bun run check:runtime` and relevant tests before completion. Immediately
register cleanup for temporary allocations and retain only compact evidence.

Keep native JavaScript/Bun lowering and avoid compatibility scaffolding. The
settled design already has Jev evidence. For a genuinely new difficult design
decision, follow the repository's three fresh, fully reworded Jev consultation
procedure and save the evidence. Do not reopen settled decisions unnecessarily.

Complete the checklist's independent review in a fresh reviewer context, giving
requirements, diff and raw evidence without implementer verdicts. Resolve its
findings, commit corrections and repeat only checks invalidated by the changes.
Do not replace that review with merely writing a request for a future review.

## Final response

Report the implemented behavior, completed task IDs, commit sequence, actual
validation results, remaining limitations and cleanup status concisely. Update
the planning-status text in the handoff/checklist to reflect the real result.
Do not claim a blocked gate passed.

**Finish your final response with a copy-pasteable review request for another
agent.** Fill in every placeholder below using actual full commit hashes,
absolute evidence paths and remaining risks. It must cover the entire feature,
including the early draft checkpoint, and must not imply that the reviewer is
expected to agree with earlier conclusions. Produce the request; do not send it
to another user-owned chat automatically.

## Review request template — populate at implementation completion

```text
Independently review the grouped completion errors and grouped assertion/fixture
labels implementation in /Users/vince/Projects/can-lang.

Review the complete change from <FULL_REVIEW_BASE_SHA> to <FULL_REVIEW_HEAD_SHA>,
including the inherited draft committed at <CHECKPOINT_SHA>. The working-tree
status at handoff is <ACTUAL_STATUS_AND_ANY_EXCLUDED_UNRELATED_CHANGES>.

Read AGENTS.md and these requirements before assessing the implementation:
- /Users/vince/Projects/can-lang/docs/syntax-taste/grouped-errors-and-labels-implementation-plan-2026-09-30.md
- /Users/vince/Projects/can-lang/docs/syntax-taste/grouped-errors-and-labels-implementation-tasks-2026-09-30.md

Raw validation evidence: <ABSOLUTE_EVIDENCE_PATH_AND_RELEVANT_SECTIONS>.
Checks not run or still blocked: <ACTUAL_LIMITATIONS_OR_NONE>.

Inspect correctness and missing regression coverage, especially exact generic
resolution/exhaustiveness; duplicate, impossible and escaping heads; grouped
binding rules; recovery checkpoints; coordination and all_failed; unchanged
payload/occurrence identity; a single shared lexical handler/fixture site;
independent assertion roots/reports/queues; scenario/template expansion and FIFO;
raw/native modes; source preservation; navigation and formatting. Check that
tests demonstrate runtime behavior rather than just implementation structure.

Review independently from requirements and code; do not treat completed boxes
or passing tests as proof. Use bounded relevant checks, the shared cache and the
repository's cleanup rules. Do not run benchmarks or broad builds. This is a
read-only review: do not edit files or commit fixes.

Return actionable findings first, ordered by severity, with absolute file/line,
a concrete failing scenario, impact and a suggested correction or regression
test. If none are found, state that explicitly. Finish with residual risks,
coverage gaps and any checks you could not perform. Do not implement fixes.
```
