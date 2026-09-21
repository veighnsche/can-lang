# I32 handler-assertion consultations: pre-dispatch audit

All three requests ask the same single question over the same facts,
constraints, and alternatives. Exact identifiers are preserved verbatim:
`callable http::server_response (http::request) emits []`, `http::request`,
`req`, P3/P10/P12 section roles, `when`, `snapshotRequest`.

Facts held constant: fixed handler contract; mandatory assertions with the
probe failure (no request-typed expression in scope); ingress-only requests
with no P10 constructor; P12's supplied-completion callback assertions;
P3 harness-token identity for opaque fixture operands; when-rows naming
in-scope opaque locals; staged test-bridge execution availability; closed
catalogue without test-only execution shortcuts.

Wording differences (intentionally rewritten, semantically equivalent):
- Request 1 frames the probe as "no eligible declaration" and the question
  as "reconciles mandatory handler assertions with ingress-only requests".
- Request 2 frames values as "born solely in runtime ingress snapshots"
  and asks "how should handler assertions obtain their request argument".
- Request 3 frames the contract as "one rigid shape" and asks for "the
  approach that keeps handler assertions writable".

No request adds, drops, or narrows an option: all four offer elided_scope /
token_constructor / real_harness_request / staged_only with matching
definitions in identical order.
