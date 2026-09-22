# Pre-dispatch Jev packet audit — 2026-09-22

Scope: read-only comparison of `docs/syntax-taste/evidence/2026-09-22/full-language-review/request-{1,2,3}.json`. No request was dispatched or edited. **Do not dispatch the current packets unchanged:** request 3 has three option descriptions that are materially looser or narrower than their counterparts. The full-request semantic equivalence should be checked again after editing.

## Structural and wording checks

All three JSON files parse, use `model: "jev-latest"`, and contain the same nine choice IDs in the same order: `normalization`, `inheritance`, `state_calling`, `error_polymorphism`, `aggregates`, `standard_failures`, `fixtures`, `extensibility`, `error_identity`. Each choice has `type`, `instructions`, and `criteria`; every corresponding option key matches. I compared the `state.constraints` text, all nine `state.evidence` entries, all nine question instructions, and all 36 option descriptions across the three packets.

The explanatory prose has been rewritten throughout rather than copied: no whole context, evidence entry, question, or option description is verbatim across packets. Exact technical identifiers, schema keys, and error signatures repeat as expected. The different question phrasings generally preserve one decision per key. Shared constraints remain: zero external users/no compatibility, native Bun operations, advisory classification, seven-error fetch/judge wrapper normalization only, no per-question wrappers, inherited omissions/last override, error-first/`ok`-last, no nesting acceptance condition, one judge request with whole-answer validation and ordered handlers. The source texts are at `request-1.json:4-14`, `request-2.json:4-14`, and `request-3.json:4-14`.

## Blockers to fix before dispatch

1. **Normalization option differs in type contract.** `request-1.json:22` specifies a **closed variant** of seven original typed payloads; `request-2.json:22` says seven existing payload types in a detail variant. `request-3.json:22` says only a “typed union,” which could admit structural/open union semantics or omit the requirement that each leaf retain its original nominal type. Reword request 3 to say a closed nominal detail variant with all seven original typed payloads. Its “keep diagnostic origins” should also explicitly retain the original occurrence/cause privately, as packets 1/2 do; an origin alone is less information.

2. **Error-identity option narrows the wire contract.** `request-1.json:102` permits explicitly versioned wire **identifiers** where required; `request-2.json:102` says stable protocol **IDs** at declared serialization boundaries. `request-3.json:102` says versioned wire **names**, which appears to force textual identifiers and also says “solely for protocols,” potentially excluding another explicit serialization boundary. Use “explicit versioned wire identifiers at serialization boundaries that require them” in request 3, with fresh prose around it.

3. **Aggregate option names the wrong match category ambiguously.** `request-1.json:62` says exact generic error specializations in **completion arms**; `request-2.json:62` says ordinary **error patterns** with concrete type arguments. `request-3.json:62` says “ordinary matches,” which in Can could be read as value `match` rather than completion-error arms. Clarify “completion-error arms” or equivalent in request 3. This matters because the candidate is a narrow syntax change, not a general nominal-value matching redesign.

## Smaller alignment checks

- `request-2.json:4` states a judge performs one **POST**, while `request-1.json:4` says one request and `request-3.json:4` says one HTTP exchange. These do not contradict, but if the HTTP method is a supplied material fact, say POST in all three freshly worded contexts.
- `request-2.json:12` describes fixture allocation using “closure identity”; packets 1/3 say callable receipts/creation receipt. The latter is the precise supplied identity mechanism. If the distinction matters to the fixture choice, replace “closure identity” with a rephrased callable-instance/creation-receipt description.
- `request-3.json:22` says application/AI/standard outcomes are excluded “outside the targeted adapter phases.” It is probably intended to preserve those outcomes, but “exclude” can be read as excluding them from the *public bound*. State explicitly that they pass through unchanged and that only native preparation/transport/codec-origin errors are converted.
- The option sets are consistent but heavily dominated for two decisions. In `normalization`, `alias_only` leaves seven caller arms and `opaque_single` removes selective typed recovery, both conflicting with stated requirements; `structured_single` bundles a default wrapper, one error, a closed detail type and private provenance. In `inheritance`, `stacked_runtime` intentionally permits recatching transformed errors and `multiple_bases` adds complexity without evidence. Jev agreement here would mostly confirm the constraints encoded in the alternatives, not independently validate the bundled implementation choice. Treat confidence/consensus accordingly and, if this consultation is meant to choose among viable designs, add a second credible structured normalization design and a second viable derivation model consistently to all three packets before dispatch. `extensibility` similarly pits a documented closed catalogue against a signature-only manifest that omits required runtime semantics; interpret its vote as policy triage, not proof that all checked extension designs are inferior.

## Equivalence by choice

| Choice | Comparison |
|---|---|
| `normalization` | State facts and four option identities align. Request-3 `structured_single` needs the closed nominal variant/original occurrence clarification above. |
| `inheritance` | One-parent flattened overlay, recatching nested wrappers, multiple ordered bases, and defer align. Fixed input/result, handler-origin escape, parent delegation, no implicit retry, and recomputed finite effects appear in all three full packets. |
| `state_calling` | Explicit per-parameter markers, group-aware callable contracts, implicit last-record state, and defer align. Prior user selection, empty grouped call, field-name preservation and non-accidental serialization are present. |
| `error_polymorphism` | Finite identity-based effect parameters, fixed aliases, body inference and defer align. All include callback/expected-contract constraints, union/subset/subtraction, handler errors and unresolved-set rejection, with standard failures separate. |
| `aggregates` | Generic exact-arm patterns, distribution-closed union, covariance and defer align apart from request-3 arm wording. All preserve nominal specialization and invariant arrays as facts. |
| `standard_failures` | Structured snapshot, string-only catch, mandatory checked channel and defer align. Private causes, automatic propagation and handler boundaries are stated. |
| `fixtures` | Lexical templates, ordinal-root tables without argument checks, symbolic-root seams with checks, and defer align. Root/ordinal/occurrence/participant/callable identity facts are present, subject to the request-2 receipt wording above. |
| `extensibility` | Closed distribution with evidence gate, arbitrary type-only manifest, full-contract third-party packages, and defer align. No demonstrated blocked application is stated. |
| `error_identity` | Nominal primary, globally generated numbers, manual current registry and defer align except request-3 “wire names” narrowing. Active/retired JSON and lock obligations appear in each packet. |

After those edits, rerun a machine check for identical question/option keys and manually re-read the full packets. A textual similarity score is insufficient to establish semantic equivalence; these three issues arose despite broad prose rewrites.
