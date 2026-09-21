# I26 connection default header representation correction

The checked connection policy stores canonical wire header names, while the
transport accepts source identifier names and normalizes them itself. Serializing
`content-type` directly therefore rejected a valid authored `content_type` default
with `http::invalid_request` (1100), before launching any request.

The emitter now converts canonical hyphens back to underscores at this boundary.
Authored identifiers cannot contain hyphens, so this preserves the checked header
identity. Policy validation still uses canonical names; transport validation,
forbidden names, duplicate detection and bearer ownership remain enforced.

The staged fetch regression reproduced 1100 before the correction. It now covers
inherited `x_default`, removal of `x_omitted`, a valid JSON content-type override,
and empty content-type overrides followed by native text/bytes defaults. A
computed non-ByteString header with no credential verifies error 1100 before
credential lookup and no request launch. Assertions run offline; real execution
uses only a loopback server and synthetic credentials.

Validation: the full compiler and integration gate passed (`go test
./compiler/... ./tests/integration -count=1`, integration 83.018s), with the pinned
Bun archive and strict generated TypeScript checks enabled. The runtime validator
was unchanged.
