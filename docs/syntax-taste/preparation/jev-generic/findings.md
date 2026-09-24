# Generic-helper technical consultation

24 September 2026. Three fresh `jev-latest` Choice requests asked only which
bounded experiment should follow the [28-assertion current-Can probe](../generic-helper-probe.md).
The full `request-1..3.json`, `response-1..3.json` and metadata files are saved
here; the [wording audit](wording-audit.json) records the manual semantic check
and fully distinct explanatory prose. The [TypeSafe Choice documentation](https://docs.typesafe.ai/primitives/choice)
describes the returned distribution. No request asked Jev to enact syntax.

All three responses ranked `explicit_error_set_trial` first, with probabilities
1.00, 0.80 and 0.99. Request 2 gave `catalogue_intrinsics` 0.20; the others
were near zero. There is no top-choice disagreement to investigate. The
agreement is advice about **which trial is informative**, not evidence that
finite error-set parameters improve Can or that their spelling is accepted.
The comparison was framed around the observed fixed-bound and data-adapter
costs; it did not include independent agent trials or multi-error callers.

Engineering next step: design a disposable compiler/protocol experiment for a
named finite error-set parameter in an authored helper, with explicit callable
inputs for its required operations. Compare it against direct `match chain`,
fixed bounds and result-as-data on two larger disjoint error sets and controlled
edits. Check exact outward errors, missing-operation diagnostics, generated
native behavior and total tokens per successful task. If the prototype does not
produce a reproducible agent or contract benefit, retain the current idiom.
Any syntax must later be presented to the user with its complete error forms.
