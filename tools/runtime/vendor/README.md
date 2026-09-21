# Output validation parser

`acorn-8.18.0.mjs` is the unchanged `package/dist/acorn.mjs` from the exact npm
archive in `acorn.lock.json`. Acquisition verified npm's SHA-512 integrity before
extracting either file; the lock also records archive/module/license SHA-256 and
upstream revision. `distribution/notices/acorn-LICENSE.txt` retains its MIT notice.

Upstream: https://github.com/acornjs/acorn/tree/d788421b242ddccb28040f1431438ee5cf474208

`output-graph.ts` uses Acorn's `parse` API with module source type on JavaScript
produced by the qualified Bun transpiler. The parser performs no resolution or
execution. An iterative AST walk rejects every remaining `ImportExpression` and
checks static import/re-export edges against the supplied inventory. Bun's
separate import scan continues to reject non-ESM edges. This is a consistency
gate for generated code, not a general-purpose sandbox for hostile JavaScript.

The sidecar builder bundles these files and pins their bytes in its complete
manifest; distribution resolution verifies them before invoking output validation.
No runtime package download or npm lookup is needed. `TestPinnedOutputParser`
checks source bytes against this lock, and packaged tests exercise the parser
with network access denied. Do not modify the vendored parser in place.
