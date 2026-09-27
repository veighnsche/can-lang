# Launch prompt — Muse Code Spark 1.3 implementation coordinator

Paste the text below into a new task using the user's selected Muse Code Spark
1.3 model, or ask that task to read and execute this file. This file does not
change model settings or start execution by itself.

---

You are the execution coordinator for the Can implementation program. Create
the persistent goal below and carry it through implementation, integration and
qualification. Use multiple agents to reduce wall time. Commit early and often.

## Goal and scope

Repository: `/Users/vince/Projects/can-lang`

Authoritative task list:
`/Users/vince/Projects/can-lang/docs/syntax-taste/can-implementation-task-list-2026-09-26.md`

Use `get_goal` to inspect the current goal, then `create_goal` when appropriate
to create this objective (or the equivalent available goal tools):

> Finish all 57 tasks in the Can implementation task list dated 2026-09-26,
> including every activated conditional task, explicitly recorded inactive
> branches, IC1–IC3 integration checkpoints, required live qualification,
> supported documentation and early, frequent commits of validated changes.

Inspect the current goal first. Reuse a matching unfinished goal; do not
overwrite an unrelated unfinished goal. Do not set a token budget: none has
been specified. Follow the goal tools' actual lifecycle rules and mark complete
only when this objective has actually been achieved. If goal tools are absent,
state that limitation, keep the objective in the durable execution record and
continue the authorized work; never claim an unavailable native goal was created.

This is authorization to **execute the implementation plan**, including its
bounded experiments, relevant tests, necessary Jev consultations and commits.
Do not stop after another plan, the first task, a single lane or a summary of
what someone else should implement. The old task-writer prompt's “no source
changes / no tests / no Jev” boundary applied to writing the plan, not this run.

## Read the authority, then start

Read repository `AGENTS.md`, the complete task list, and these current records:

- `docs/syntax-taste/preparation-2026-09-26/handoff.md`
- `docs/syntax-taste/preparation-2026-09-26/p07-contracts.md`
- `docs/syntax-taste/preparation-2026-09-26/p07-reconciliation.md`
- `docs/syntax-taste/preparation-2026-09-26/p07-acceptance.md`
- `docs/syntax-taste/preparation-2026-09-26/p07-dispositions.md`
- `docs/syntax-taste/preparation-2026-09-26/p06-answers.md`
- `docs/syntax-taste/preparation-2026-09-26/blocker-resolution/README.md`

Resolve the paths above against the repository root. Read individual research
and experiment packets when their task needs them; do not reread all historical
preparation for each assignment.

Inspect actual HEAD, working-tree changes and any execution records. Preserve
other work. Resume already-started tasks and reuse valid evidence; never reset
the repository to the documented baseline or repeat completed implementation
because the plan originally began with unchecked boxes.

BLK-01 and BLK-02 are resolved: local variables are included in references and
safe rename; tenant AI allowances count input/output tokens and reject excess
immediately. The supplement fixes the accounting contract. Do not reopen these
choices. Its native profile qualification is still required implementation work.

## Coordinate concurrent implementation

Retain the eight lanes and their bounded ownership:

| Lane | Responsibility |
| --- | --- |
| A | Surface, lowering and bulk collections |
| B | Failure conventions and reusable infrastructure |
| C | Browser controls, shared UI and applications |
| D | Host integration |
| E | Lifetime, reporting, storage and catalogue integration |
| F | Data, companion service and durable budget ledger |
| G | Editor features |
| H | Provisioning, release, AI and documentation |

- Keep one coordinator and a continuously updated ready queue. Dispatch bounded
  tasks to available agents as prerequisites become ready; refill freed slots
  immediately. Do not wait for an entire wave or lane before starting unrelated
  work. Eight logical lanes do not require eight simultaneous workers.
- Follow the model-selection skill for workers and actual advertised model IDs.
  Keep the user's selected coordinator model. Use economical workers suited to
  each assignment; reserve costly escalation for consequential uncertainty.
- Initial ready candidates are A01–A05, B01/B02, C07, E01/E03 and H01/H09,
  subject to actual current progress. Start provisioning early. Favor work that
  unlocks A→C→D, E→F and A→G, while progressing B and independent H work.
- Give each agent its task IDs, relevant contracts, writable file boundaries,
  prerequisites, expected handoffs, validation and definition of done. Require
  prompt partial handoffs when they unblock another agent. Do not allow
  unrestricted recursive agent spawning.
- Use isolated worktrees/branches for independent mutations where useful.
  Isolation does not remove shared-file ownership: one owner integrates each
  shared file. Other agents submit patches or commits to that owner. In a shared
  checkout, disjoint writes and a single serialized Git index/commit owner are
  mandatory; agents must not stage or commit one another's unfinished changes.
- E alone integrates catalogue sections and regenerates matching outputs,
  incrementally. H serializes generation/packaging publication. Preserve the
  task list's complete ownership table and its cross-owner handoffs.
- Use distinct databases, storage prefixes, browser profiles and ports per run.
  Respect actual CPU, worker and environment capacity. H owns the x86 schedule
  and capped, serialized live AI evaluation.
- F05 portable lease claims do **not** depend on F02's locking-read probe.
  F04 budget-ledger backend qualification **does** feed H08. Follow the current
  task graph rather than stale historical edges.

## Work each task to completion

Perform a focused starting check: read its contract and relevant current code,
verify prerequisites, identify appropriate tests, then implement. Do not repeat
extensive research or preparation unless a named experiment or genuine new gap
requires it.

Implement the accepted design and preserve its invariants. There are zero
external users and no compatibility obligation. Compile to native JavaScript/Bun
operations with only the adapters required for Can's contracts and immutability.
Do not add deferred features, parallel alternative implementations or legacy
spellings merely to avoid updating callers and goldens.

Run appropriate focused validation, fix failures and update affected examples,
documentation and generated artifacts. For authored TypeScript under `runtime/`
or `tools/runtime/`, including fixtures, run:

1. `bun run lint:fix:runtime`
2. `bun run format:runtime`
3. `bun run check:runtime` and relevant tests before completion.

Use `bun run lint:runtime --format=agent` for compact diagnostics. Preserve
intentional test behavior. Generated `runtime/catalogue.ts` and pinned vendor
files are regenerated through their proper paths, never hand-edited.

Preserve CAS staging: generation content is linked and read-only; tamper tests
must unlink and recreate files before changing bytes. Keep unique metadata
writes, unreferenced-entry collection, the link-based immutable test mirror and
full-copy mutable workspaces distinct. Do not reintroduce full-copy staging.

## Jev and genuinely new decisions

Consult Jev when a difficult technical decision is genuinely unresolved or new
evidence invalidates a selected assumption. Do not consult it for mechanical
implementation, routine fixes or approval of the plan. Read the TypeSafe skill
and current relevant primary documentation before using it.

Supply all relevant facts, constraints, counterevidence and viable alternatives.
Jev is a classifier: it cannot research or inspect the repository for you.
For each decision, prepare **three fresh requests before sending the first**.
Rewrite all explanatory prose—context, instructions, questions and option
descriptions—while preserving facts, constraints, alternatives and necessary
exact code/identifiers. Check semantic equivalence and whole-request wording
differences. Save exact requests, responses, metadata and the wording audit.
Investigate disagreements; record engineering judgment separately. Agreement
is advice, not proof, a majority-vote rule or guaranteed bias removal.

Conditional tasks need an explicit inactive outcome or a completed triggered
branch. If an experiment triggers a new contract, return only that contract to
preparation and publish its resolution before dependent implementation. Ask the
user about genuinely new or changed syntax, showing complete examples and
consequences. Do not re-ask settled choices or silently invent gated grammar.
Continue independent lanes while awaiting required answers.

## Commit early and often

- Make the first implementation commit as soon as a small coherent slice has
  passed its relevant checks. Continue committing each independently reviewable,
  validated slice or completed task; do not accumulate a giant final commit.
- Use concise Conventional Commit messages and record associated task IDs in
  the execution log or commit body. Do not create meaningless checkpoint commits
  solely to inflate frequency.
- Stage explicit files/hunks. Keep unrelated edits, credentials, temporary
  outputs and other agents' unfinished work out of commits. Preserve required
  generated artifacts with their source changes.
- Workers in isolated checkouts may commit their own slices. A single
  coordinator/integration owner integrates those commits in dependency order,
  resolves conflicts and revalidates affected behavior before consumers proceed.
  Agents sharing a checkout never concurrently manipulate the Git index.
- Record commit hashes and validation evidence as work lands. Normal scoped
  commits are already authorized; do not ask permission for each one. Respect
  actual tool/sandbox approval requirements when encountered.

## Qualification and checkpoints

Execute IC1, IC2 and IC3 in the task list's order. Candidate builds and W6
deployment qualification precede IC2; final release assembly follows it. Final
supported-story documentation follows IC3. Avoid introducing a checkpoint cycle.

Required live browser, database, storage, companion, Linux and AI legs are part
of the goal. Skips, missing credentials, unavailable machines and passing mocks
are not substitutes. For AI, rejection-only tests cannot satisfy successful W6
qualification: a real provider profile must qualify its complete-request bound.
Honor both outcomes of every registered experiment; do not quietly weaken a
failure condition into success.

No x86 emulation anywhere: no QEMU or emulated `linux/amd64` Docker. UP25 runs
only on the user's native x86 machine. Ask before sustained full-tilt work on the
MacBook Air. Keep credentials environment-provided and out of evidence. Live AI
evaluation needs the H-owned credentials/spend approval gate; this is separate
from the product's token allowance.

Request genuinely missing access, configuration or decisions early and precisely.
Complete independent work while blocked. A failed contract pauses only its
consumers and returns to preparation; an implementation defect under an unchanged
contract returns to its code owner for correction. Do not silently drop scope.

## Durable progress and finishing

Maintain `docs/syntax-taste/execution-2026-09-26/README.md` and an evidence index.
Record task state, assigned agent, worktree/branch, prerequisite/handoff status,
commit hashes, checks, experiment outcomes and remaining blockers. Link completed
task checkboxes to evidence; distinguish inactive conditional work from code
actually delivered. Record exact revisions, environments and pass/failure/skip
results without double-counting reused runs.

Keep updates concise and useful: completed outcomes, what is running, what has
become ready and what needs user input. At context/agent handoff, resume from
these durable records instead of repeating the investigation.

Continue until all required work is complete. If an external prerequisite truly
prevents further progress, finish unrelated work, record the exact blocker and
required action, and follow the goal tools' blocked-state rules. Never claim goal
completion while required work or qualification remains.

On success, mark the goal complete and report completed/inactive task counts,
lane/checkpoint results, commits, qualification evidence and remaining documented
limitations. The intended result is the implemented, validated program with
recorded completion evidence.
