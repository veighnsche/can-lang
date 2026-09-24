# Planning checks and review

24–25 September 2026. Inputs are the [implementation plan](../../../post-upgrade-implementation-plan-2026-09-24.md),
[ordered tasks](../../../post-upgrade-implementation-tasks-2026-09-24.md),
selected behavior, invoice acceptance and saved mechanism experiments.
This directory records validation of the **plan**, not execution of its tasks.

Run [the validator](validate.py) from the repository root:

```sh
python3 docs/syntax-taste/evidence/2026-09-24/implementation-plan/validate.py
```

Its [result](validation.json) checks that all 27 tasks are topologically ordered,
scheduled exactly once, assigned to the correct lane, behind their prerequisites
and separated by declared write sets within a wave. It also verifies all 11
selected Fix findings have task coverage and local documentation links exist.
Concrete file ownership and the adequacy of tests still require source review;
the validator cannot prove those from abstract write-set names.

Three independent source audits supplied the action/server, browser/delivery
and core/generic task boundaries. Subsequent independent reviews check the
full plan for dependency, ownership and selected-contract coverage defects.
The review corrections and remaining limits are recorded in
[the review record](review.md). No runtime TypeScript or production compiler
source was edited for this planning task.
