# Jev advice on UP06 revocation placement

25 September 2026. Three new `jev-latest` requests were prepared before any
response was sent. Each state paragraph, instruction, and option description
was rewritten in all three versions while preserving facts, constraints, and
alternatives. The [wording audit](wording-audit.json) records the 11 compared
explanatory fields and full-request hashes. Requests [1](request-1.json),
[2](request-2.json), and [3](request-3.json), their exact
[responses](response-1.json) ([2](response-2.json), [3](response-3.json)),
transport metadata, and [probability summary](summary.json) are saved. Each
call returned HTTP 200 and model `jev-1.13.0`; total reported usage was 3,618
input and 267 output tokens. No request contained another response. Jev did
not inspect source or run code; the supplied state summarized the token
tables, the abandon probe, the mandated order, and the production paths.

The table shows selected options and probabilities, not measured design
correctness. Exact option meanings are in each request.

| Decision | Request 1 | Request 2 | Request 3 | Engineering selection |
| --- | --- | --- | --- | --- |
| Revocation boundary | split .57 (server-only .43) | split .84 | split .89 | **Split boundary.** Dispatch abandons everywhere and revokes on rejected-route paths that admitted no handler; the server outer boundary revokes handled tokens after owner drainage in a finally before response return. |
| Revoke failure mode | silent .91 | silent .97 | silent .95 | **Silent idempotent revoke.** Unknown, forged, or already-revoked input is a no-op, keeping the primitive safe in finally paths and under the overlapping dispatch-plus-server calls on rejected server requests. |

The boundary result is wording-sensitive: request 1 nearly splits between the
split and server-only options (confidence 0.35). The investigated difference
is exactly the rejected-route path outside a server scope. Server-only
revocation leaves tokens from direct dispatch calls (tests, harnesses, and
the later captured-routes dispatch work) live after rejection, pushing a
revoke duty onto every future direct caller. The split alternative closes the
token at the layer that makes the rejection decision, where no handler ran
and therefore no per-request child work can hold the token; in server flow
the outer revoke then repeats idempotently after drainage. Dispatch-only
revocation scored 0.0 in all three requests, confirming that revoking handled
tokens before drainage would break the mandated "only after the lifetime
ends" rule for admitted child work. Agreement is treated as advice: the
selections above rest on these repository facts and the regression tests,
not on the classifier vote.
