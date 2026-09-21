# I32 handler-assertion consultation decision

## Results

All three responses reject `real_harness_request` (0.05, 0.01, 0.03) and
`staged_only` (0.02, 0.07, 0.01). The contest is `elided_scope` versus
`token_constructor`: request 1 selects `token_constructor` at confidence
0.31 (0.48 vs 0.45), requests 2 and 3 select `elided_scope` at confidence
0.65 and 0.54 (0.74 and 0.65).

## Disagreement investigation

Request 1 is a near-tie expressing genuine uncertainty between the two
token-scope designs, not a counter-argument: all three agree the runner
supplies harness-token requests and body readers are `when`-supplied. The
explicit-constructor side values source explicitness; the elision side
values catalogue minimalism. Decisive evidence beyond the consultations:

- Elision is total and deterministic, not heuristic: no Can expression can
  be `http::request`-typed outside an ingress scope, so every
  `http::request`-typed root input is always harness-supplied; a row either
  supplies exactly the non-scope arguments or it is an arity error. There is
  no optional-omission ambiguity for explicitness to resolve.
- Elision needs no catalogue addition, no new source syntax, and no
  assertion-context gating; the constructor needs all three.
- Elision generalizes to every ingress-scoped function (helpers taking or
  capturing requests); the constructor gates one more op by context.
- C excludes opaque values from whole-completion assertion comparison, so
  `http::request` results cannot be expected-compared; the checker must
  reject those rows explicitly either way.
- Readers deny live execution under assertion context (clock precedent via
  `denyLiveBoundary`), so token-scope reads without `when` rows fail as
  missing fixture, and staged loopback keeps real-reader evidence separate.

## Decision (advice, not proof)

Elided scope arguments: assertion rows bind provided arguments to
non-`http::request` root inputs in order; `http::request`-typed root inputs
(ordinary and `near`) are harness-supplied tokens compared by P3 token
identity in `when` rows. Body readers must carry `when`-supplied outcomes.
`http::request` results are rejected in expected `ok` completions. The
private staged bridge keeps entering compiled handlers with real snapshots
as distinct real-can evidence. This gives `scoped` its meaning: a supplied
boundary over a harness-provided scope.

Compiler, runner, and staged tests must still prove elision arity, token
identity comparison, missing-fixture diagnostics, and the real/supplied
evidence split before I32 can close.
