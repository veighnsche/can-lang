# Supporting-contract Jev consultation record

24 September 2026 · source revision `02a549d28fd5fc5c3996160e65a97de332390d30` · preparation only

This record covers difficult technical choices in [DI-17–23](../support-alternatives.md), except DI-21. The billing/calendar boundary already has a routine conditional path: build the policy-explicit domain library with existing exact minor-unit and zoned-time operations, then add a controlled primitive only if an executable case proves a gap. There is no distinct hard choice to ask Jev before that evidence exists. No request here selects syntax, changes a disposition, or adopts a production contract.

## Method and audit

The evidence supplied to the requests came from [support alternatives](../support-alternatives.md), [AI support](../ai-support-evidence.md), [lifetime and deployment](../lifetime-deployment-evidence.md), [core evidence](../core-evidence.md), the [evaluation protocol](../evaluation-protocol.md), [constraints](../constraints.md), and repository `AGENTS.md`. I read the live [TypeSafe index](https://docs.typesafe.ai/llms.txt), [Choice guidance](https://docs.typesafe.ai/primitives/choice.md), and [HTTP API contract](https://docs.typesafe.ai/api.md) on 24 September. Choice returns one option with a distribution and confidence; it does not explain causality or prove correctness. The API response identifies all twelve successful calls as `jev-1.13.0`.

Four sets each contain related, independently useful Choice questions over a shared state. Each set has three fresh requests. Before sending, I manually checked that every variant retained the same present behavior, constraints, alternatives and test discriminators; the request generator also asserts distinct wording in the state, question instructions, and every option description. Technical identifiers and option keys remain exact. The full request JSON, successful response JSON, time/status metadata, and the generator are in the sibling set directories. `native-ai/request-1.json` first received HTTP 529; that failed response is preserved as `native-ai/response-1.attempt-1.json`. The same request succeeded on retry, recorded in `response-1.attempt-2.json` and metadata. No prior Jev answer was included in any request.

The wording changes preserve meaning but cannot mathematically rule out framing effects. Several choices are also complementary at different layers, although Choice requires one answer. Disagreement is analyzed below as a warning against interpreting a plurality as a decision.

| Question | Request 1 | Request 2 | Request 3 | Reading of the advice |
| --- | --- | --- | --- | --- |
| DI-17a LLM recovery | ordinary helpers .59; wrap trial .41 | wrap trial .95 | wrap trial .97 | Split; prototype only if the same-kind provenance failure is reproduced in an agent flow. |
| DI-17b batch effects | phase documentation .92 | transaction .68 | phase documentation .90 | Split across complementary steps; precise phase contract is immediate, with transaction/outbox trials tied to the ledger requirement. |
| DI-17c grouped callables | named adapters .84 | named adapters .95 | named adapters .76 | Stable advice to use adapters as the control; direct-reference cost remains unmeasured. |
| DI-18a scoped escape | narrow checker .61 | runtime context .82 | runtime context .79 | Split; retain runtime safety and compare contextual diagnostics with a strictly limited checker probe. |
| DI-18b race faults | application observers .77 | opt-in policy .83 | opt-in policy .62 | Split; service signal/noise evidence is needed before a language policy. |
| DI-19 provider boundary | distribution adapter .78 | protocol/catalogue .89 | distribution adapter .77 | Split; start with the typed HTTP baseline because no required SDK gap has been demonstrated. |
| DI-20 bulk collections | bulk operations .94 | bulk operations .56 | bulk operations .74 | Stable interest in a targeted bulk trial, conditional on realistic throughput and contract measurements. |
| DI-22 named fields | named-form trial .99 | positional controls .64 | positional controls .69 | Split; same-type refactor trial must compare both and measure silent misbindings. |
| DI-23a layout | formatter first .67 | formatter first .95 | formatter first .89 | Stable advice to finish already accepted formatter/diagnostic work before a continuation prototype. |
| DI-23b local validity | advisory lint .46 | hard rule/fix .42 | semantic-only rule .62 | No stable preference; controlled agent edits and negative semantic examples must drive validity policy. |

Numbers are each selected option's probability, not a measured chance that the engineering choice is correct. Full alternative distributions, confidence and token usage are in the raw responses.

## Disagreement investigation and engineering reading

For DI-17a, the first request puts “without assuming the new policy is needed” in the instruction and emphasizes avoiding extra replay in the ordinary-helper criterion. The other two ask more directly about origin-sensitive recovery and describe a native provenance adapter. Both describe the same known limitation. The .59/.41 first result and later strong wrap preference show sensitivity to framing, not evidence that the extension is necessary. The authored/decoder same-kind failure and no-replay probe should decide whether the current helper can meet the workflow.

For DI-17b, the choices are not mutually exclusive: documentation states batch semantics, a database transaction can protect eligible writes, and an outbox handles durable delivery. The second request foregrounds “handler one writes and handler two fails” and chose a transaction; the first and third emphasize phase clarification and chose documentation. An engineering trial should document current ordering and test transaction rollback/uncertain commit and outbox duplicate delivery against the actual ledger policy. None implies rollback of provider effects.

For DI-18a, a richer `resource_state` report and a narrow checker diagnostic could coexist. The first wording asks for a “diagnostic experiment” and chose the checker narrowly; later phrasings highlight the existing runtime safety boundary and choose context. Keep the guard regardless. Test false positives for an enclosing pool and missed closure/alias escapes before claiming value from static analysis.

For DI-18b, all variants preserve `Promise.any` selection, pending losers, and owner leases. The first specifically asks about consumed failures “without changing” selection and leans toward service observers; the later requests describe an adapter policy more prominently and lean toward opt-in reporting. Both can report the same underlying fault, and the service fallback has not been observed. Compare diagnostic identity, useful signal, ordinary noise, timing, redaction and sink failure on the stated zero/one/many participant cases before expanding the race contract.

For DI-19, the first and third variants ask how to respond to a provider gap, which may make a distribution adapter sound timely; the second stresses that no SDK blockage has been shown and selects the protocol/catalogue path. All three retain HTTP as the concrete baseline and restrict a manifest to a proved SDK-only gap. The evidence justifies building the named provider flow first, with a narrow distribution adapter only if the baseline fails or costs materially more under the registered comparison.

For DI-22, the first instruction emphasizes whether named fields reduce same-type reorder errors and strongly selects that trial. The other two frame the question as an experiment for existing positional safety and choose positional controls. Neither has an observed production misbinding. Register the same rename/add/reorder task and compare valid output, silent swaps, compiler diagnostics, agent completion and tokens across positional/domain-type/factory, tooling, and a bounded named prototype. A pass-through two-`int` factory cannot count as protection by itself.

For DI-23b, the three responses select three different policies, with low to moderate confidence (.19, .13, .44). The options partly overlap because a semantic-only hard rule can also provide advisory lint and a safe fix. The state contains a plausible meaningful `permitted` local but no observed agent cost or ambiguity. First measure the current rejection and fix quality on both meaningful and accidental aliases; only a demonstrated semantic hazard can support hard invalidity for a narrower case.

The DI-17c, DI-20 and DI-23a agreement is advice about trial order, not a finding that adapters, bulk operations or one-line grammar are permanently best. DI-20's request 2 gives the bulk option only .56; actual volume, duplicate behavior and failure timing remain decisive. DI-23a's accepted formatter and span work is already committed by the current disposition; the consultation supplies no independent reason to advance multiline grammar.

## Next evidence boundary

Use the [evaluation protocol](../evaluation-protocol.md) before turning these comparisons into accept/retain/reject/defer dispositions: pre-register the same task and hidden checks for baseline and candidates, preserve current contracts, run held-out creation/refactor/diagnostic cases, and count full tokens only for successful attempts. DI-17 requires exact request count, provenance, disclosure and ledger state; DI-18 requires owner and race negative cases on a named service target; DI-19 needs a real provider operation; DI-20 needs realistic collection sizes; DI-22/23 need the same controlled agent edits. Jev cannot supply those observations or research missing platform facts.
