# I31 opaque HTML and HTMX constructors

Current HTML authority resides in private WeakMaps keyed by frozen empty tokens.
Nodes retain rendered text and immutable tag/head/descendant metadata. Attributes
retain their validated construction context; URLs retain native normalization and
whether their source was a single-slash local path. Safe results are immutable
rendered documents/fragments. No authored code can forge these handles or import
the private renderer used by HTTP sinks and the conformance harness.

All text and attribute data passes to Bun.escapeHTML. URL uses the native parser
and rejects controls, backslashes, credentials, scheme-relative forms and
non-HTTPS schemes; normalization cannot turn a local path into a scheme-relative
spelling. Structural checks cover the complete P9 tag inventory, void children,
list/select/table/definition-list rules, descendant anchors/forms, head placement,
attribute applicability/enumeration and case-normalized duplicate names.

HTMX attributes have only the fixed typed constructors and finite intervals.
Target/indicator inputs are identifiers, never selectors. The runtime head emits
the pinned local script with integrity and the fixed no-eval/no-script,
self-request/response handling policy. Actual asset serving and browser admission
remain I34; rendering its constructor does not claim those gates are complete.

The current compiler admits catalogue constructors and recognizes opaque HTML
provenance in domain payloads. Brands, seals, raw constructors and asset_bridge
syntax provide no authority in current build/run/assert. `std/html/README.md`
describes this replacement. The old std source/TS and bridge implementation are
labelled historical pending the explicit I43/I44 inventory/deletion dependencies;
there is no current HTML fallback through them.

## Evidence

- Eight runtime tests with 345 expectations cover every author tag and every
  tag-checked text attribute, successful rendering, malicious markup/attributes,
  enumerated values, duplicate names, child categories, nested anchors/forms,
  local/external URL rules, head-only nodes, fixed HTMX policy, interval bounds,
  forged handles and proxy inputs with zero traps.
- Compiler tests reject direct safe/node construction, raw/script operations,
  brands and bridge grants; maintained catalogue operations compile normally.
- `TestCurrentBundledHTML` builds with the pinned distribution and strict generated
  TypeScript, then executes real Can assertions without HTML fixtures. A separate
  native loopback harness obtains hostile request-derived text and passes it to
  the compiled renderer. It checks complete document framing, escaped title and
  attribute/text contexts, exact fragment rendering, and only the compiler-owned
  script node. Both rendering functions are actual generated exports.
- Three fresh Jev consultations, their responses and full-wording/semantic audit
  are retained in `i31-jev/`; advice favored the private token representation.
  Tests establish the claimed behavior, not model agreement.

The HTML attribute index, boolean-attribute grammar and link-type tables were
read from the WHATWG HTML standard while refining P9's finite attribute matrix.
No browser/HTMX execution is claimed by this constructor-only task.

Final gate: `bun test runtime` passed 234 tests and 21350 expectations. Strict
TypeScript passed for the HTML adapter/tests and staged generated source. The
full compiler/integration gate passed with the pinned Bun archive and strict
TypeScript enabled: `go test ./compiler/... ./tests/integration -count=1`
(integration 111.590s).
