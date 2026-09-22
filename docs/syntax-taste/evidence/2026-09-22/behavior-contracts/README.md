# Behavior-contract consultations — 2026-09-22

Three fresh consultations cover nine difficult choices. All explanatory prose was independently rewritten and checked for equivalence before dispatch: [audit](pre-dispatch-audit.md). Requests and raw responses are saved as `request-1.json` through `request-3.json` and `response-1.json` through `response-3.json`; per-response metadata preserves status, timing and content hashes. No credentials are saved.

| Choice | Packet 1 | Packet 2 | Packet 3 |
| --- | --- | --- | --- |
| payload | reuse (0.71) | reuse (0.89) | new_records (0.64) |
| wrapper_policy | origin_tables (1.00) | origin_tables (1.00) | origin_tables (1.00) |
| bounds | calculated_marker (0.79) | calculated_marker (0.86) | calculated_marker (0.95) |
| patterns | unique_bare (0.85) | unique_bare (1.00) | unique_bare (1.00) |
| snapshot | snapshot (0.90) | snapshot (0.71) | snapshot (0.70) |
| publication | all_roots (1.00) | all_roots (1.00) | all_roots (1.00) |
| deadline | root_worker (1.00) | root_worker (1.00) | root_worker (1.00) |
| reuse | lexical_templates (1.00) | lexical_templates (0.97) | lexical_templates (1.00) |
| native_tests | attached (1.00) | attached (1.00) | attached (1.00) |

Parentheses show selected-option probability, not correctness. All responses identify `jev-1.13.0`; combined reported usage is 7261 tokens.

The payload representation split is investigated in [disagreement investigation](disagreement-investigation.md). The selected design keeps existing typed payload leaves based on the data model and unchanged public fields, not a majority vote. All other choices agree across packets; that remains advisory.

The [behavior-contract index](../../../../implementation/language-behavior-contracts-2026-09-22.md) now links to the actual rules in their canonical specifications, including details completed by engineering review rather than by the classifier: exact source grammar, finite bound dependency rejection, deadline/report behavior, fixture hashing, source ownership and evidence obligations.

The live official [HTTP API](https://docs.typesafe.ai/api) and [Choice guidance](https://docs.typesafe.ai/primitives/choice) were used to form the consultations, not as evidence for Can language correctness. Live Markdown was fetched directly after web access failed; the API exposes typed advisory choices over supplied state.

Validation for this documentation-only change checks local links, disposition/acceptance coverage, request wording and shape consistency, request/response hashes, JSON examples and existing catalogue payloads/ID availability. Proposed Can syntax has not been compiled. No compiler/runtime source was edited and no full compiler suite was claimed. Implementation must establish BC01–BC24.
