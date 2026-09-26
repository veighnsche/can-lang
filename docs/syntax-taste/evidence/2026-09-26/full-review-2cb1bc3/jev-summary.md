# Fresh design consultations

All three requests were prepared before any response was read. Each of the 32 corresponding explanatory fields was independently worded, with the same facts, constraints, questions and alternatives retained. Technical identifiers and option keys remain stable. The saved wording audit checks full corresponding field differences; semantic equivalence was also manually reviewed. Requests do not contain past review verdicts or previous classifier answers.

Read the live [documentation index](https://docs.typesafe.ai/llms.txt), [API contract](https://docs.typesafe.ai/api.md) and [Choice guide](https://docs.typesafe.ai/primitives/choice.md). Markdown subpages were fetched with curl after the web reader could not open them. Request model `jev-latest`; actual response model `jev-1.13.0`. All requests returned HTTP 200. Combined usage: 5,135 input tokens, 791 output tokens. Credentials were read from the environment and are absent from saved artifacts.

Numbers below are the selected option's probability, not probability of design correctness. Full distributions/confidence are in the raw responses.

| Decision | Consultation 1 | Consultation 2 | Consultation 3 |
| --- | --- | --- | --- |
| host_extension | reviewed_adapters (0.9) | catalogue (0.67) | companion (0.35) |
| failure_abstraction | result_data (0.49) | result_data (0.52) | result_data (0.63) |
| assertion_setup | factory_pattern (0.58) | factory_pattern (0.83) | factory_pattern (0.84) |
| iteration | iteration_primitive (0.53) | iteration_primitive (0.36) | iteration_primitive (0.69) |
| request_lifetime | operation_deadlines (0.72) | operation_deadlines (0.91) | scope_policy (0.64) |
| ui_composition | library (1.0) | library (0.82) | library (0.92) |

## Evidence-led reconciliation

- **Host boundary:** complete disagreement, with a near tie in the third response. Current source distinguishes three jobs: a few universal DOM capabilities fit the catalogue; reusable behavior fits ordinary Can libraries; a genuinely required third-party widget tests whether an adapter or companion is warranted. The recommendation is a small comparative experiment, not automatically building an unrestricted FFI. See `platform-reconciliation.md`.
- **Request lifetime:** operation deadlines win twice, scope policy once. Source inspection distinguishes native HTTP abort from owned settlement, SQL waits from server-close wait deadlines, and process escalation from mere timeout reporting. The report recommends an explicit request/shutdown policy plus enforceable operation contracts. It does not infer that adding a timeout parameter everywhere guarantees bounded cleanup. Follow-up also found S3's cancellation/deletion contradiction and polling-only deadline; see `backend-reconciliation.md`.
- **Error abstraction:** result data selected in all forms, with moderate distribution concentration. A fresh compiler/runtime prototype establishes that the representation works. Its existing examples do not prove retry count or success-after-failure, so the next experiment must use a distinguishable outcome sequence and independent oracle. No effect/error-row system is prescribed from this consultation alone. See `core-reconciliation.md`.
- **Assertion setup:** all forms prefer existing factories first. A consumer-private helper can lawfully obtain an owner value and abort on impossible fixture rejection. That avoids a public sample API but retains setup/mandatory assertion overhead; the review asks for an economical demonstrated convention before adding setup syntax or relaxing testing rules.
- **Iteration:** all forms select an iteration primitive, but one response is nearly uniform. The strong evidence is stack growth in emitted synchronous recursion. Native self-tail lowering remains a viable alternative and the report does not claim the consultations establish a unique design.
- **UI composition:** library-first preference is consistent and compatible with available named functions, records and callables. Missing host state cannot be manufactured by those libraries; native input additions precede the reusable widget exercise. This does not prove that no new UI syntax will ever be useful.

These are six bounded design comparisons, not classifier research or certification. Agreement is advice. Initial independent reports were complete before recommendations and classifier responses were shared for reconciliation.
