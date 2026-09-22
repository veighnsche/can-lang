# Full-language-review evidence and Jev reconciliation

This directory supports the [review](../../../full-language-review-2026-09-22.md). All conclusions are design recommendations. No compiler/runtime changes or implementation approval are represented.

## Recorded work

- [Core review](core-review.md), [AI/native review](ai-review.md), [platform/testing review](platform-review.md): independent source-based reviews, subsequently reconciled by the coordinator.
- [Document audit](consolidated-document-audit.md): caught the actual build/assert execution mismatch and clarified the scope of exact generic-error patterns. Both were corrected in the final report; handler-bound and timeout-granularity clarifications were also incorporated.
- [Measurement script](measure.py) and [JSON result](measurements.json): source counts, not model tokens or a TypeScript comparison.
- [Eight-fetch sketch](eight-fetch-proposal.can.txt) and [generator](make-fetch-sketch.py): analytical before/after with identical operations/success checks/nesting. It requires proposed normalized public contracts and has not been compiled.

## Consultation method

Read the live TypeSafe documentation index, HTTP API, Choice guidance and parallel-questions cookbook. The web tool could not open the index; HTTPS retrieval of the official Markdown pages succeeded. Sent three independent POST requests to the documented System One endpoint using an existing environment credential. No credential is stored in this evidence. Every response was HTTP 200 and identified `jev-1.13.0`.

Each packet asked nine independent Choice questions about the same evidence. All explanatory state, questions and alternative descriptions were rewritten across the three packets. The [initial audit](pre-dispatch-initial-audit.md) found semantic wording drift before any request was sent. The [second audit](pre-dispatch-final-audit.md) cleared the corrected full packets and a second credible normalization alternative. A [coordinator supplement](pre-dispatch-coordinator-audit.md) records the final equivalent coordination-pattern scope correction, final hashes and pre-dispatch check.

Final sent bytes are [request 1](request-1.json), [request 2](request-2.json), [request 3](request-3.json). Complete replies are [response 1](response-1.json), [response 2](response-2.json), [response 3](response-3.json); corresponding metadata files retain status, duration, endpoint and request hashes. All nine response keys and option distributions were checked against their request; selected options are distribution maxima and the rounded distributions sum to approximately one.

## Results

Values below are selected option / its probability / API-reported confidence. These are classifier outputs, not calibrated probabilities that a language design is correct.

| Choice | Packet 1 | Packet 2 | Packet 3 |
| --- | --- | --- | --- |
| normalization | `structured_single` / 0.71 / 0.65 | `dedicated_details` / 0.61 / 0.51 | `structured_single` / 0.69 / 0.61 |
| inheritance | `linear_overlay` / 1.00 / 1.00 | `linear_overlay` / 1.00 / 1.00 | `linear_overlay` / 1.00 / 0.99 |
| state_calling | `grouped_callable` / 0.91 / 0.88 | `grouped_callable` / 0.81 / 0.74 | `explicit_markers` / 0.54 / 0.39 |
| error_polymorphism | `finite_parameters` / 0.60 / 0.47 | `finite_parameters` / 0.70 / 0.59 | `finite_parameters` / 0.89 / 0.86 |
| aggregates | `exact_patterns` / 0.98 / 0.98 | `exact_patterns` / 0.97 / 0.96 | `exact_patterns` / 1.00 / 1.00 |
| standard_failures | `snapshot` / 0.71 / 0.61 | `string` / 0.78 / 0.71 | `snapshot` / 0.73 / 0.65 |
| fixtures | `lexical_templates` / 0.91 / 0.88 | `lexical_templates` / 0.74 / 0.65 | `lexical_templates` / 0.85 / 0.80 |
| extensibility | `closed_evidence` / 1.00 / 1.00 | `closed_evidence` / 0.97 / 0.96 | `closed_evidence` / 1.00 / 1.00 |
| error_identity | `nominal_primary` / 0.96 / 0.94 | `nominal_primary` / 0.91 / 0.88 | `nominal_primary` / 0.86 / 0.81 |

Reported usage: 8,663 input tokens and 1,443 output tokens, 10,106 total across all three calls. No repeated sampling was used to chase a preferred answer.

## Investigating disagreement

**Normalization:** packets 1/3 chose reusing the seven original nominal payloads; packet 2 chose seven dedicated public detail records retaining the same fields. Both preserve one public error, typed recovery, phase-aware conversion and private provenance. Inspected A2's payload contracts, `runtime/domain.ts` data/variant admission and `runtime/transport/http.ts` typed completions. Can already supports errors as ordinary immutable data. No separate public detail semantics have been demonstrated, so the report recommends reuse to avoid a duplicate nominal representation and conversion table. Dedicated detail records remain valid if a public contract intentionally differs from the raw shape. This is an engineering judgment, not a 2–1 proof.

**State callables:** packets 1/2 chose preserving grouped inputs in callable contracts, while packet 3 chose explicit markers with confidence 0.39 and probability 0.37 on defer. Inspected `check/callables.go`, `native_program.go`, `emit/judge.go` and `emit/llm.go`: grouping/state metadata already exists and serialization is declaration-driven; first-class admission is restricted by frontend contracts. Thus grouped callable types are a credible alternative, not something the runtime makes impossible. The report compares markers for uniform ordinary invocation with group-aware callable contracts and explicitly recognizes the earlier user-selected spelling; it no longer favors either before prototype evidence. Emitted TypeScript inputs are already flat, so this is a source/type-contract restriction rather than a backend ABI requirement. Neither alternative is adopted here.

**Standard failures:** packets 1/3 preferred snapshots; packet 2 retained strings. Inspected `runtime/failure.ts`: the same opaque value already stores kinds, occurrence IDs and safe messages, with private causes. Ordinary catching already exists, so exposing existing snapshot projections does not expand the caught failure set. The report recommends that consistency while retaining automatic propagation, fatal-termination limits and non-recatching handlers. The API supplied no explanations; these observations investigate the design disagreement without attributing hidden reasoning to Jev.

**Agreement with uncertainty:** all three chose finite error-set parameters, but the first two distributions gave fixed bounds/aliases 0.40 and 0.28. Retain the one-real-combinator prototype gate. All three preferred nominal identity, but the numeric reporting consumer audit remains necessary. Consistent answers do not remove either uncertainty.

The independent [disagreement investigation](disagreement-investigation.md) supplements these source checks. No fourth consultation was used to erase dissent.

## Limits

Option order was held fixed. Some alternatives intentionally represent earlier proposals that violate supplied requirements, so strong agreement can reflect those constraints. Rewording is not guaranteed bias removal, consensus is not proof, and none of these classifications overrides confirmed user requirements. Testing, current-code evidence and executable prototypes remain necessary.

No full Go/Bun/TypeScript suite was run for this documentation-only task. The scripts validate counts and sketch structure; link/file checks validate the authored documentation. The build/assertion distinction and missing assertion timeout are source-traced findings, not newly executed hang reproductions. Scoped-resource escape, new state syntax, wrapper inheritance and effect parameters remain design/prototype work.
