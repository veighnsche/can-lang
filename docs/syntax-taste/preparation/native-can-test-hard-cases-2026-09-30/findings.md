# Jev consultation record: hardest existing test cases

Three fresh source-backed consultations completed using requested `jev-latest`, returned model `jev-1.13.0`. The [generator](consult.py), [wording/semantic audit](wording-audit.json), requests, complete responses and HTTP metadata are saved here. All explanatory context, instructions and option prose differ across the three requests; facts, constraints, choices and technical identifiers remain equivalent. No previous answer was supplied to a later consultation. Live TypeSafe [API](https://docs.typesafe.ai/api) and [Choice](https://docs.typesafe.ai/primitives/choice) documentation were inspected before sending.

| Question | Request 1 | Request 2 | Request 3 |
| --- | --- | --- | --- |
| Native bridge priority | `explicit_bridge`, confidence .99 | `explicit_bridge`, 1.00 | `explicit_bridge`, 1.00 |
| Browser decision location | `tokens_to_can`, 1.00 | `tokens_to_can`, .91 | `tokens_to_can`, .99 |
| SQL observer | `raw_driver_scoped`, .99 | `raw_driver_scoped`, .99 | `raw_driver_scoped`, .92 |

Files: [request 1](request-1.json) / [response 1](response-1.json), [request 2](request-2.json) / [response 2](response-2.json), [request 3](request-3.json) / [response 3](response-3.json).

No selected-answer disagreements occurred. Distribution differences are still informative: request 2 assigned .06 to local Can browser handlers; request 3 assigned .03 to two DB observers everywhere and .02 to CLI-default observation. The source review gives concrete conditions for those alternatives: if pending-input/route round trips cannot meet semantics, consider an identified driver-local Can callback; when the protected component is Bun SQL itself, require a different driver. Neither warrants upfront duplication across every test. A failed explicit C/native ingress needs a new bridge decision, not a claim that raw Bun facts prove candidate adapter behavior.

Decisions retain current full Can/Bun execution and external Go ownership, qualify the explicit native ingress rather than assume general synchronous callbacks, begin browser control with Can-side tokens, and scope independent raw SQL observations to the adapter they actually bypass. The [main challenge](../../native-can-test-hard-cases-2026-09-30.md) records the source-backed corrections and [14 experiments](experiments.md) record executable uncertainties.

Jev supplies classifications/probabilities, not reasoning or research. Its agreement does not prove the design correct, the bindings feasible, the rewritten requests free of bias, or the retained coverage complete. No consultation ran candidate code or changed the implementation. Total reported consultation usage: 5,102 input and 420 output tokens.
