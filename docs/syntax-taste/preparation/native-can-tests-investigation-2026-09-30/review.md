# Focused research audit

A separate read-only Sol reviewer checked the investigation against cited source
for material factual errors and loopholes in the user's host-harness-elimination
requirement. No builds, tests or implementation changes were performed.

Two findings were raised and incorporated into the report's scope and section 4.7:

1. `distribution/qualify.py:128-171` contains qualification and verdict policy,
   not merely a launcher. That policy must move into Can along with the native
   conformance scenarios; a changed executable path alone is insufficient.
2. `/tests` delegates some existing oracles to `runtime/test/*.test.ts`, e.g.
   `tests/integration/process_test.go:60-67`. Those obligations need Can-owned
   replacements. Independent low-level unit suites can remain outside this
   scope, but cannot be used as the replacement integration oracle.

The reviewer found no other material factual error in the cited code inspected.
This is a focused research review, not validation of a future implementation or
an exhaustive case-by-case migration plan.

## Completion contract review, 2026-09-30

A separate bounded read-only review checked the
[completion contract](../../native-can-tests-completion-contract-2026-09-30.md)
against the user requirements and the baseline, failure-conventions,
host-discrimination, authoring-policy and browser-controls source evidence.
The review required no additional backend or API choice; the existing Jev
consultations remain architecture advice rather than an acceptance decision.

Two wording corrections were incorporated:

1. Static manifests may index Can entrypoints and supply input/expected vectors,
   but may not hide executable scenario scripts. Can owns case definitions,
   sequences, selection and comparison policy.
2. The authoring-policy directory contains recorded deterministic comparison
   legs, not a current runner. Its unrun agent trials are not passing coverage.

The reviewer found no other material issue with backend neutrality, required
ordinary Can functions, foreign source-under-test fixtures, scope, coverage or
cleanup. Classification remains per obligation; no files were deleted or marked
finally retired. All 224 inventoried `/tests` files remained unchanged. Local
documentation links and line anchors were checked. No implementation tests,
builds, benchmarks, temporary execution workspaces or dependency copies were run
or created for this contract.
