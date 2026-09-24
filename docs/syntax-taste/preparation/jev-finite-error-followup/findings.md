# Finite-error helper follow-up consultation

24 September 2026. Three fresh `jev-latest` Choice requests assessed two details of
the proposed finite-set **trial** after the [28-assertion current-Can
probe](../generic-helper-probe.md). [Requests](request-1.json),
[responses](response-1.json), the other numbered request/response pairs, API
metadata, and the [wording audit](wording-audit.json) are saved in this directory.
All explanatory state, question, instruction, and option prose was rewritten for
each request; stable option keys and exact technical identifiers were retained.
The live [TypeSafe Choice documentation](https://docs.typesafe.ai/primitives/choice.md)
was consulted before sending. Jev supplied judgments, not research or an
implementation proof.

| Question | Request 1 | Request 2 | Request 3 |
| --- | --- | --- | --- |
| Determine E at an invocation | Mandatory explicit row 0.73; callback static bound 0.24 | Callback static bound 0.57; mandatory explicit row 0.43 | Callback static bound 0.65; mandatory explicit row 0.34 |
| Relay unknown E member | Explicit grouped forward arm 0.89 | Explicit grouped forward arm 0.90 | Explicit grouped forward arm 0.89 |

The specialization disagreement is material and low confidence. The first
wording emphasized the *call-site* contract; the later two emphasized a
callback declaration edit and its diagnostic. That difference plausibly moved
weight toward an explicit row at each call in request 1. Both options are
semantically checkable: mandatory rows can intentionally widen E if they only
require callback subset, whereas deriving E from the callback's declared bound
keeps the outward set exact without assuming that runtime behavior narrows the
declaration. The trial should use callback-derived E and permit an **optional,
exactly equal** explicit E assertion at a call where an agent wants a pinned
boundary. This combines precise default specialization with a local diagnostic
when a callback's declared errors change. It is an engineering choice; the
three Jev answers do not prove that it will improve agent completion or token
cost.

All three requests strongly preferred one written `E as failure => failure`
completion arm over implicit call propagation or per-specialization enumerated
arms. The arm makes the forwarding site visible while keeping the source helper
independent of the caller's concrete error names. A compiler experiment must
still prove that named arms, a remainder E arm, and an empty E are unambiguous;
agreement is not an acceptance test.
