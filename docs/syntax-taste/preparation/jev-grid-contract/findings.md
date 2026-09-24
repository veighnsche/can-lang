# Repeated invoice rows and first Can browser DOM architecture

24 September 2026. Three fresh `jev-latest` Choice requests compared two
independent questions after the [product consultation](../jev-product-contracts/README.md)
and concrete equal-length row-omission counterexample. The live TypeSafe
[API](https://docs.typesafe.ai/api.md) and [Choice](https://docs.typesafe.ai/primitives/choice.md)
guidance had been read. [Requests](request-1.json), [responses](response-1.json),
HTTP metadata and [wording audit](wording-audit.json) are saved for all three
rounds; [consult.py](consult.py) reproduces them. Every explanatory objective,
evidence field, instruction and option description was freshly rewritten
without changing facts, constraints or alternatives. Earlier answers were
not supplied. All replies identify `jev-1.13.0`, with 3,446 input and 273
output tokens total.

| Choice | Request 1 | Request 2 | Request 3 |
| --- | --- | --- | --- |
| Repeated rows | checked keyed rows 1.00, confidence .99 | same 1.00, 1.00 | same 1.00, .99 |
| Browser DOM model | explicit native-DOM catalogue 1.00, 1.00 | same 1.00, 1.00 | same 1.00, 1.00 |

There is no response disagreement. The row options were constrained by a
counterexample that parallel array length checks cannot pass safely; the
foreign bridge cannot satisfy a Can-authored frontend by definition; and a
reducer runner adds machinery not demonstrated as necessary. Those facts can
make one option dominant under this framing, so agreement is advice rather
than a measured agent outcome or proof of implementation safety. No
compiler, browser or database behavior was inferred from Jev. The separately
executed [native FormData probe](../keyed-row-probe.md) supplies the bounded
row-association observation.

Engineering selects a **bounded keyed-row HTML form decoder** and a
**main-thread browser target with explicit DOM/event/Fetch catalogue
operations**. The row decoder uses stable per-row keys and an exact order list,
retains known raw text on structural rejection and leaves domain conversion
explicit. The browser target uses ordinary named Can functions and immutable
draft records, with a small opaque state/view owner adapter over native DOM
and Fetch. It does not introduce a virtual DOM, implicit reactive system,
worker profile or general mutable alias. Exact semantics and acceptance are
in [keyed-row design](../keyed-row-design.md) and
[browser target design](../browser-target-design.md). These are planned
contracts, not current compiled Can features.
