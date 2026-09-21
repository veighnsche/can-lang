# I31 URL scalar correction

Reading P10 for I32 exposed an I31 omission: the URL constructor must reject
unpaired UTF-16 surrogates, while text/HTML response sinks use native replacement.
Native URL parsing alone replaces an unpaired surrogate and therefore could not
establish the specified URL contract.

`html::parse_url` now checks `String.prototype.isWellFormed` before parsing.
Relative and absolute HTTPS regression inputs containing lone high/low surrogates
produce `html::invalid_url("syntax")`; there is no normalization fallback.
Existing native URL restrictions and text escaping behavior are unchanged.

The HTML runtime suite passes eight tests and 349 expectations. Strict TypeScript
passes for the adapter and its tests.

The full `go test ./compiler/... ./tests/integration -count=1` gate passed with
the pinned Bun archive and strict generated TypeScript checks enabled
(integration 112.506s).
