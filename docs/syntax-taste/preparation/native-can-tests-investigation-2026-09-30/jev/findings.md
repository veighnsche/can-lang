# Native Can test migration: Jev consultation evidence

Date: 30 September 2026. Investigation only; no implementation selected by vote.

The repository requires three fresh consultations for difficult design decisions.
Each request supplied the same current source facts, user constraints and
alternatives. All 19 explanatory fields per request were rewritten: context,
questions and option descriptions. `wording-audit.json` records full-field
differences, request hashes and manual semantic-equivalence review. Stable
technical identifiers, numeric inventory and schema fields intentionally repeat.
This process does not establish unbiased phrasing or make agreement proof.

The live [API reference](https://docs.typesafe.ai/api) and
[Choice reference](https://docs.typesafe.ai/primitives/choice) were read before
using `POST /v1/systemone`, model `jev-latest`. Jev is a classifier; it was given
the evidence and was not asked to inspect the repository or research capabilities.

## Results

All responses identify `jev-1.13.0`. Entries below are selected-option probability
and returned confidence, respectively.

| Question | Request 1 | Request 2 | Request 3 |
| --- | --- | --- | --- |
| Execution architecture | `suite_program`: .94 / .92 | `suite_program`: .93 / .90 | `suite_program`: .84 / .79 |
| Browser investigation boundary | `typed_driver`: .55 / .32 | `typed_driver`: .62 / .43 | `typed_driver`: .65 / .47 |
| Compiler-test runner provenance | `known_good`: .97 / .95 | `known_good`: .80 / .70 | `known_good`: .98 / .98 |

Total reported token usage: 4,545 input and 405 output. Full distributions are in
`response-1.json` through `response-3.json`, with matching saved requests.

There is no selected-label disagreement. Browser confidence is low, and
`capability_spike` receives .37/.22/.35. Follow-up source inspection explains a
real open issue: existing scripts use callback evaluation, synthetic/native user
input, routing/interception and asynchronous event observation, not only basic
clicks. See `tests/integration/browser/assets.mjs:37`, `:61`, `:98`, `:111`;
`compare.mjs:75`; `grid.mjs:382` and `:608`. A small typed automation surface may
not cover those operations. We therefore keep a capability experiment as a
design gate before committing to that browser API; agreement does not settle it.

The candidate-only bootstrap option varies from .00 to .12 across requests.
Source evidence remains decisive: the candidate compiler can affect the runner
it compiles, so a separately identified known-good runner gives useful failure
independence. Its refresh policy and behavior with new runner-language features
remain design work. Universal dual execution is not recommended by this packet;
reserve it for targeted qualification when evidence warrants the extra load.

## Transport and retention

The initial sandbox transport could not resolve the service hostname; no service
response arrived. Those failures are preserved in `transport-attempt-1-*.json`.
The same three audited requests then succeeded through the supported network
permission mechanism, once each. `response-*.metadata.json` records HTTP status,
elapsed time and request hashes. No response-based retry or wording adjustment
was made after reading model answers. Credentials and authorization headers were
never written to the evidence.
