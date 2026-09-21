# HTML and HTMX

The current standard library exposes the catalogue-owned opaque constructors in
[P9](../../docs/syntax-taste/platform-testing-spec.md#p9-safe-html-and-htmx-constructors).
Their implementation is [runtime/platform/html.ts](../../runtime/platform/html.ts).
Use `html::text` for text, typed attributes for attributes, `html::element` for
validated structure, and `html::document` or `html::fragment` for immutable safe
output. Native `Bun.escapeHTML` performs all escaping.

The current compiler never accepts brands, seals, `asset_bridge` grants, raw
markup, inline scripts, style attributes or arbitrary HTMX selectors. The pinned
runtime-head constructor is the sole script-producing operation; its asset
serving and browser admission belong to I34.

[Current source example](../../compiler/testdata/current/html/main.can) and
[staged integration](../../tests/integration/html_test.go) exercise real rendering
of request-derived hostile text. [Historical notes](HISTORY.md), the old Can/TS
files here and `compiler/bridge.go` are superseded legacy material retained only
until the coordinated I43/I44 deletion; bundled build/run/assert do not use them.
