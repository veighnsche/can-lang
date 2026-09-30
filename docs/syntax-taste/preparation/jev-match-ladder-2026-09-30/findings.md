# String-match ladder advisory consultation

Source inspection established that ordinary string matching already supports literal cases, `|` groups and `_`. The missing feature is actionable lint feedback. Manolea supplied both long equality ladders and a two-decision ladder containing OR groups. Agent generation and safe edits, rather than human typing comfort, determine this rule's scope.

Three fresh requests used independently rewritten context, instructions and option descriptions with the same facts and alternatives. All chose a two-decision minimum (probabilities 0.90, 0.92, 0.97), and a warning without automatic edits (1.00 each). There were no choice disagreements. Agreement is advice, not proof; the compiler analysis and regression tests establish applicability.

The chosen rule uses checked identities for one immutable `str` binding and disjoint literal strings. It accepts `is` comparisons, reversed operands, parentheses and OR groups, explicit true/false arms in either order, and direct false-branch continuation. It handles terminal and value matches. It excludes single predicates, mixed bindings, recomputation, projected values, intervening steps, relational comparisons, negative tests and overlapping literals. Valid subchains can still be diagnosed when surrounding logic does not qualify. An overlap invalidates the candidate across it; an independently disjoint suffix can qualify separately.

`CAN-CHECK-STRING-MATCH-LADDER` is nonblocking and identifies the first condition of each qualifying chain. Its message recommends `match <binding>`, literal arms, `|` for shared cases and `_` for fallback. No online service participates in linting. No automatic edit is supplied because comment preservation and transformation safety have not been established.

The requests and responses beside this record are credential-free. Current API shape was checked against [TypeSafe's API reference](https://docs.typesafe.ai/api) and [Choice guidance](https://docs.typesafe.ai/primitives/choice). The first sandboxed network attempt failed before contacting the service; the authorized network run completed all three consultations.
