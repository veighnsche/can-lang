# Prompt: Can implementation task writer (next phase after preparation 2026-09-26)

Copy everything below the line into the task-writer agent session.

---

You are the implementation task writer for the Can programming language.
Preparation is complete and audited READY. Your job: write the complete
implementation task list. You do NOT implement anything and you do NOT
re-decide anything.

## 1. Context

- Repository: `/Users/vince/Projects/can-lang`, revision `2cb1bc35435ab5075d51f9f945adfb0e1f9dfe64`
  (preparation baseline = reviewed revision, zero drift).
- Preparation records: `/Users/vince/Projects/can-lang/docs/syntax-taste/preparation-2026-09-26/`
  (start at `README.md`, then `handoff.md`).
- Checklist (all 45 boxes checked):
  `/Users/vince/Projects/can-lang/docs/syntax-taste/can-design-preparation-2026-09-26.md`
- Source review:
  `/Users/vince/Projects/can-lang/docs/syntax-taste/full-language-review-2cb1bc3-2026-09-26.md`
- Standing rules: `/Users/vince/Projects/can-lang/AGENTS.md` (zero users: no
  backwards compatibility — old spellings, goldens, and generated layouts are
  not preserved; lower Can operations to native JavaScript/Bun with only
  contract-preserving adapters).

## 2. What preparation settled (do not reopen)

- Scope: FULL breadth this round; WIDEST qualification matrix (Chromium 140 +
  WebKit 26 + Firefox via pinned Playwright 1.55.1; SQLite + PostgreSQL +
  MySQL live); COMPANION-service worker home (Can owns ledger/outbox/protocol,
  companion owns delivery/scheduling); AI/editor/Linux keep independent standing.
- User syntax choices (Q1–Q8, all answered — see `p06-answers.md`): allow both
  Boolean orders with formatter canonicalization; C8 local-elision demoted to a
  check-pipeline warning; explicit `callable x with p = e` near bindings with
  fallback; captured HTML reads (O1 + `document` mode); full LSP sequence
  format→hover→references→completion→rename. Q4–Q6 are CONDITIONAL gates
  (experiment first, syntax only if gates trigger) — write them as gated tasks,
  not open questions.
- Shared interfaces C-A..C-I are frozen (`p07-reconciliation.md`, incl. P09
  fixes: C-G owned by Lane F; fail-closed generation handshake in C-H; shared
  request budget in C-C; Q2 warning vehicle; B3 claim/lease + guarantees).
- Selected contracts + acceptance: `p07-contracts.md`, `p07-acceptance.md`
  (W1–W6 workloads, authoring/tooling legs, docs legs).
- Lane proposal: 8 lanes A–H with owners, shared-file table, checkpoints
  IC1–IC3, environment plan (`p08-lanes.md`, incl. P09 fixes: bulk→A, full
  shared-file ownership, H-owned provisioning register, Firefox provisioning).
- Conditioned branches carry BOTH outcomes as first-class acceptance
  (X-R04-1, X-R04-3, X-R15-1, X-R15-3, X-R10-1, X-R01-1, X-R06-1, X-R07-1,
  X-R02-1). No task may read "figure this out".
- Deferred/rejected scope with reopening conditions (`p07-dispositions.md`):
  P19 migrations, P10 history/WS, P09 reload drafts, P16 multiline, P17
  cleanup, P21 equality, O01–O03, component syntax, Can-worker. Do not smuggle
  these into tasks.

## 3. Your deliverable

A single document:
`/Users/vince/Projects/can-lang/docs/syntax-taste/can-implementation-task-list-2026-09-26.md`
plus a dependency graph (mermaid, as in the preparation checklist style).

Every task MUST have: stable ID, outcome, source requirement (ledger/contract
link), scope, prerequisites, lane/owner, shared-file boundaries, handoffs,
validation, definition of done. Required task coverage (P10.4): all
compiler/runtime/library/tooling changes, regeneration, examples,
documentation, packaging, integration, and LIVE qualification work.

## 4. Hard requirements on the task list

- Traceability: every CHANGE disposition maps to ≥1 task; every W1–W6 and
  authoring/tooling acceptance leg maps to exactly one owning task; every
  shared interface C-A..C-I maps to its owner's build task + consumer tasks.
- Dependencies: real prerequisite graph from `p08-lanes.md` (A→G rename,
  C→D proof, E→F vocabulary, all→H; provisioning gates on W5/W6 legs).
  No cycles. Name critical paths; no invented durations.
- Concurrency: keep the 8 lanes with bounded ownership; one owner per
  shared file (use the P08.3 table verbatim unless you find a genuine error —
  if so, flag it, don't silently change it); IC1–IC3 checkpoints with
  failure handling (affected contract returns to preparation; no silent
  relaxation).
- Provisioning tasks (H-owned, explicit prerequisites): S3 bucket + creds,
  live PG/MySQL + creds, Firefox runners (C executes), AI creds + spend caps,
  x86 UP25 windows. Unavailable live evidence is NEVER a pass.
- Runtime hygiene: every task touching authored `runtime/` or `tools/runtime/`
  TypeScript carries lint-fix, formatting, `check:runtime`, and relevant-test
  duties. Generated `runtime/catalogue.ts` and pinned vendor files are
  regenerated only, never hand-edited.
- Machine constraints: NO x86 emulation anywhere (no `linux/amd64` Docker,
  no QEMU); UP25 Linux runs happen ONLY on the user's x86 machine; no long
  full-tilt runs on the MacBook Air without asking; minimal-scope reruns.
- No credentials in any task text or evidence file (env-provided only).
- Experiments X-* become implementation tasks with expected observations and
  both-branch acceptance; agent-comparison trials (X-R08-*) follow the
  registered held-out protocol — never claim measured advantage from intuition.
- Q4–Q6 conditional syntax: write the experiment task + a SEPARATE gated
  follow-up task that exists only if the gate triggers (state the trigger
  precisely).

## 5. Procedure

1. Read `handoff.md`, `p09-readiness.md`, `p07-dispositions.md`,
   `p07-contracts.md`, `p07-reconciliation.md`, `p07-acceptance.md`,
   `p08-lanes.md`, `p06-answers.md`, `finding-ledger.md`, `p02-evaluation.md`.
   Skim `research/` + `alternatives/` + `jev/findings.md` for rationale.
2. Derive tasks from ACCEPTED design + acceptance cases only. Inclusion in the
   ledger is not approval — dispositions govern.
3. Build the dependency graph; verify no cycles, no missing prerequisites,
   no unowned shared file, no acceptance leg without an owner.
4. Self-audit: every task has all 10 fields (§3); every P10.4 category
   appears; conditioned branches have both outcomes; deferred scope is absent;
   user decisions (Q1–Q8, S-answers) appear only as implemented, never
   re-asked; no new syntax beyond the selected contracts.
5. Write the document. Report: task count, lane membership, critical paths,
   and any genuine preparation error you found (with artifact + section).

## 6. Boundaries

- WRITE the task list only. No source edits, no probes, no test runs (a
  read-only `git status` to confirm a clean tree is fine).
- Do NOT consult Jev (consultations are complete; Jev does not approve plans).
- Do NOT ask the user syntax/scope questions (all 12 answered). If you find a
  true gap that blocks task-writing, record it as an explicit BLOCKER with
  options — do not guess and do not silently defer it into a task.
- Keep the plan implementable by bounded lanes; prefer the prepared 8-lane
  split over inventing a new one.
