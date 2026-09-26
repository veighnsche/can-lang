# Preparation handoff — 2026-09-26 round (P10)

## Readiness status: READY

Independent audit ([report](p09-audit-report.md)) delivered NOT READY with
9 blockers + 9 majors + 6 minors; all resolved with recheck in the
[readiness record](p09-readiness.md). Implementation tasks can now be
written without hiding design decisions inside them.

## What preparation produced

- Baseline: [p01-baseline.md](p01-baseline.md) (HEAD = reviewed revision,
  zero drift) · Ledger: [finding-ledger.md](finding-ledger.md) (R01–R16,
  every review concern + acceptance condition + prior requirement placed)
- Evaluation: [p02-evaluation.md](p02-evaluation.md) (W1–W6 workloads,
  protocols, evidence inventory, environments)
- Research: [research/](research/README.md) (4 packets, all findings
  source-anchored; no reruns needed at identical revision)
- Alternatives: [alternatives/](alternatives/README.md) (per-topic options +
  interaction map C-A..C-I + experiment register X-*)
- Jev: [jev/](jev/README.md) (7 questions × 3 fresh wordings, wording audit,
  responses, [findings](jev/findings.md) with disagreement analysis)
- User decisions: [packet](p06-syntax-questions.md) →
  [answers](p06-answers.md) (12/12 answered: full breadth, widest matrix,
  companion worker, Q1–Q8)
- Selected design: [dispositions](p07-dispositions.md) →
  [contracts](p07-contracts.md) → [interfaces](p07-reconciliation.md) →
  [acceptance](p07-acceptance.md)
- Lanes: [p08-lanes.md](p08-lanes.md) (8 lanes A–H, ownership, checkpoints
  IC1–IC3, environment plan, cycle check)
- Audit: [report](p09-audit-report.md) + [scenario walk](p09-scenario-walk.md)
  + [readiness](p09-readiness.md)

## Proposed lane boundaries (from P08)

A surface/lowering · B failure conventions · C browser/apps · D host
decision · E lifetime/observation/storage · F data/companion pair ·
G editor · H release/AI/docs. Critical paths: C→D host chain, E→F data
chain. Shortest dependent chains: A→G rename, E→F vocabulary, all→H.
Every shared file has exactly one owner with handoffs; generated catalogue
+ vendor files regenerate only.

## Settled decisions

Q1/Q2/Q3/Q7/Q8 surfaces (incl. Q2 warning vehicle, Q3 `with` grammar);
R03-O1 + `document` mode; companion worker home + B3 claim/lease/guarantees;
builtin_auto redacted hook; lookup-first PG recipe; shared-budget request
composition; fail-closed generation handshake; bulk construction contract;
S3 conditioned branches (incl. O2 `discard_upload` replacement); full-
breadth + widest-matrix scope; all shared interfaces C-A..C-I (C-G → F);
provisioning register (H); Firefox provisioning (C).

## Deferred scope (with reopening conditions)

P19 migrations, P10 history/WS, P09 reload drafts, P16 multiline, P17
cleanup, P21 equality, O01–O03, component syntax, setup-region/error-param/
iteration-primitive syntax (Q4–Q6 experiment gates), R02-ownership primitive
(X-R02-1 gate), O2 scope policy (X-R04-2 gate), savepoints (PG-app gate),
new column kinds (encoding-insufficiency gate), streaming AI
(accepted-requirement gate). Can-worker rejected for this round (companion
chosen) with a stated reopening prototype.

## Conditioned branches (probe-gated, both branches specified)

SQL cancel semantics (X-R04-1), disconnect signal (X-R04-3), S3 `end(Error)`
release (X-R15-1), S3 effective deadline (X-R15-3), locking-read
expressibility (X-R10-1), host discrimination (X-R01-1), result-data
concision (X-R06-1), factory cost (X-R07-1), library-first proof (X-R02-1).

## Production qualification during implementation

W1–W6 live legs, widest browser/DB matrix (Chromium/WebKit/Firefox ×
SQLite/PG/MySQL), UP25 Linux on x86 (never here), isolated S3 qual,
AI budget/eval with spend caps, companion pair qual, old-browser rollout.
Provisioning prerequisites (H register): S3 bucket + creds, live PG/MySQL +
creds, x86 access + windows, AI creds + spend approval, Firefox runners.
Unavailable live evidence is not a pass.

## Remaining blockers

None in preparation. Implementation-time provisions above are the complete
blocker list and each has an owner + gate.

## Next deliverable (P10.3/P10.4 structure)

Each implementation task: stable ID, outcome, source requirement, scope,
prerequisites, lane/owner, shared-file boundaries, handoffs, validation,
definition of done. Must include compiler/runtime/library/tooling,
regeneration, examples, docs, packaging, integration, live qual. Runtime
tasks carry lint-fix/format/check/test duties; generated catalogue + vendor
files never hand-edited. Writing that list is the next phase; implementing
it is a subsequent phase.
