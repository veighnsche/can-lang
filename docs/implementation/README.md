# Can implementation work

The current work queue is the September 22 language-design revision. The previous I01–I50 compiler implementation is recorded as completed; these new fixes have **not** been implemented merely because their designs are settled.

Read in order:

1. [Finding dispositions](language-design-dispositions-2026-09-22.md) — all 49 findings: 16 implement, 18 retain, 15 defer.
2. [Selected decisions](../syntax-taste/decisions.md) and incorporated specifications — current language contract, including the resolved LD29 check API.
3. [Acceptance evidence](language-change-acceptance-2026-09-22.md) and [verified gaps](implementation-gap-verification-2026-09-22.md) — what must pass and what has actually been reproduced.
4. [Language-fixes implementation plan](language-fixes-plan-2026-09-22.md) — milestones, code ownership, evidence and scope boundaries.
5. [Ordered language-fixes task list](language-fixes-tasks-2026-09-22.md) — **21 pending tasks**, in dependency order, starting with LF01.

The [behavior-contract index](language-behavior-contracts-2026-09-22.md) locates detailed contracts in the canonical specifications. Native AI forms, grouped state, explicit contracts and attached assertions remain the baseline. Deferred abstractions are not prerequisites for this work.

## Completed implementation history

The September 21 preparation and execution records remain available: [research](research.md), [brainstorm](brainstorm.md), [reconciliation](reconciliation.md), [original plan](plan.md), [completed I01–I50 tasks](tasks.md), [closed file-level handoff](remaining-tasks.md), [coverage](coverage.md), [native reuse](native-reuse.md) and [saved evidence](evidence/2026-09-21/). Those records describe their own baseline and do not establish acceptance of the September 22 revisions.
