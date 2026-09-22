# LD29 consultations and selected check contract — 2026-09-22

Three fresh Jev consultations each evaluated four choices. Full state, instructions and all option descriptions were rewritten in every request; exact identifiers/code remained stable. The [pre-dispatch audit](pre-dispatch-audit.md) records semantic equivalence checks. Requests, raw responses and per-response hash/timing metadata are saved alongside this file. No credentials are saved.

| Choice | Packet 1 | Packet 2 | Packet 3 |
| --- | --- | --- | --- |
| Failure channel | domain (0.68) | domain (0.66) | domain (0.73) |
| API | require (0.89) | require (1.00) | require (0.72) |
| If standard: negative assertions | check_only (0.76) | catcher (0.49) | check_only (0.57) |
| If domain: negative assertions | existing (0.99) | existing (1.00) | existing (1.00) |

Numbers are selected-option probabilities, not correctness estimates. All responses identify `jev-1.13.0`; reported usage totals 5265 tokens. The selected channel has moderate rather than overwhelming support. Jev supplies classifications, not explanatory reasoning; the rationale below is engineering judgment.

## Selection and disagreement analysis

Select an ordinary catalogue operation `checks::require(bool condition, str reason) -> void emits [checks::failed]`, with `checks::failed(str reason)` as a normal domain error. Existing expected-domain-error rows test false, so no standard-expectation grammar or seventh standard category is added. [C9.2](../../../technical-spec.md#c92-named-runtime-checks) owns the complete contract.

This choice fits the existing explicit-contract model: the operation deliberately reports an authored condition, its reason is typed ordinary data, callers can recover or forward, and negative assertions compare the complete payload. It does cost an `emits` entry and explicit matching/relay in forwarding code. That cost is intentional and must appear in before/after measurements; the shorthand unchecked `call check(...)` shown earlier is not valid for this errorful API. There is no hidden fixed domain set or implicit inference extension.

A standard check would keep current helper bounds empty and allow unchecked void statements. It would also require a new category and a decision on direct negative expectations or catching adapters. That is a credible alternative, not an unsafe one. For this bounded repair, the existing domain mechanism solves diagnosis and testing together without extending either standard failure semantics or assertion grammar. Neither lack of compatibility obligations nor a classifier majority is the reason for selecting it.

The hypothetical standard-expectation branch disagreed: packet 2 selected `catcher` with a rounded 0.49/0.49 distribution, while the others selected `check_only`. Both approaches preserve harness stickiness; they trade source brevity against an additional context-specific grammar rule. The evidence does not settle that tradeoff. Since the selected domain channel does not consume this conditional answer, do not majority-vote a standard matcher into scope. A future general standard-expectation proposal would require its own demonstrated need and design review.

All real harness violations remain sticky independently of application recovery. Catching `checks::failed` can legitimately produce success; catching a missing fixture cannot clear the root's failure state. Native fetch/judge normalization preserves this authored error as emitted origin. Check execution uses a native conditional and existing typed completion/diagnostic adapters, with no deliberate arithmetic fault and no second assertion engine.

## Evidence and limits

The existing helper was read in `compiler/testdata/current/fetch/main.can`. The current catalogue has no `checks` API; core error ID 1010 is available and selected in the specification, not allocated by changing the compiler here. Request construction follows the live [TypeSafe HTTP API](https://docs.typesafe.ai/api), read by direct HTTPS after web access failed; this source establishes API shape, not Can correctness.

[validate.py](validate.py) verifies all request/response hashes, complete wording variation, identical candidate sets, response presence, aggregate usage and the selected contract/acceptance links. It does not prove equivalence or language correctness. No proposed Can program was compiled and no runtime was changed during this consultation. LD29's design gate is closed; its implementation evidence remains required under AE29.
