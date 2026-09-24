# Jev consultation: owner-record spelling and observation policy

24 September 2026. Three fresh requests to `jev-latest` returned
`jev-1.13.0`. They use the executed 11-root validated-value probe, current
grammar/runtime evidence, canonical package ownership, lexical generic
authority and the already accepted technical boundary. None includes another
request's answer. The [TypeSafe skill](../../../../.agents/skills/typesafe-ai/SKILL.md),
[live API reference](https://docs.typesafe.ai/api) and
[Choice guide](https://docs.typesafe.ai/primitives/choice) informed the calls.

| Judgment | Request 1 | Request 2 | Request 3 |
| --- | --- | --- | --- |
| Declaration | `owner record email`, probability .65, confidence .46 | `record email owner`, probability .64, confidence .45 | `record email owner`, probability .46, confidence .19 |
| Projections | Exported functions only, probability .69, confidence .37 | Marked public fields, probability .81, confidence .62 | Exported functions only, probability .59, confidence .19 |
| Generic codecs/schema | Owner-only derivation, probability .80, confidence .61 | Owner-only derivation, probability .84, confidence .68 | Owner-only derivation, probability .56, confidence .12 |

The retained artifacts are [request 1](request-1.json), [request 2](request-2.json),
[request 3](request-3.json), [response 1](response-1.json),
[response 2](response-2.json), [response 3](response-3.json), the
[wording/equivalence audit](wording-audit.json) and [reproduction script](consult.py).
Every explanatory state field, instruction and alternative description was
rewritten; code spellings and technical identifiers stayed exact. Requests
were reviewed for semantic equivalence before submission. Text uniqueness is
not proof that framing effects disappeared. The service used 3,289 input and
354 output tokens in total.

## Disagreement investigation and engineering resolution

**Header:** identical semantics did not yield a stable spelling preference.
The suffix won twice but its weakest winner had only .46 probability, and the
prefix won the other request. Current parser inspection shows one `record`
production with name, optional parameters and fields. Both modifiers can reuse
it; no measured edit or token result distinguishes them. Select `owner record`
as an explicit contextual prefix and retain the ordinary record category.
This is an engineering grammar choice, not evidence that Jev proved it best.
The separate `owned` head has no demonstrated semantic advantage. Do not reuse
`opaque`, whose existing resource semantics differ from this data boundary.

**Observations:** marked immutable fields cannot themselves fabricate a record,
so both candidates preserve the creation invariant. They trade direct reads
and representation coupling against named projections and one additional
function call. The current grammar has no per-field visibility, while exported
functions already express projections. Select hidden fields plus exported
functions for the initial complete design. No evidence yet warrants another
field grammar or the corresponding partial-destructuring rules. The split is
not resolved by claiming that public fields are unsound.

**Codecs:** all requests favored granting the owner automatic derivation, but
the third was close (.56/.44), and unanimity is not proof. Owner-only derivation
is consistent with trusted owner construction and would be viable. Select
uniform rejection of generic codecs/wire schemas reaching owner records,
including inside the owner, to preserve one explicit wire/factory conversion
path and context-independent schema admission. Automatic decoder minting adds
an implicit route around factory invocation; prohibiting it removes that route
without pretending to prove the owner constructor's predicate. This deliberately
overrides Jev's advice for the stated contract and implementation simplicity.

[The complete selected design](../owner-record-design.md) specifies ownership,
projection, matching, equality, codecs, fixtures, generics, lowering and planned
acceptance tests. These choices remain planned language semantics. No production
compiler/runtime file changed and no candidate implementation was benchmarked.
