# Match a string once instead of nesting boolean comparisons

Use an ordinary value match when several literal values choose a result. Match
the value itself, join equivalent cases with `|`, and use `_` for the fallback.
This keeps the case table flat and makes the accepted values explicit.

For example, put this complete library in `src/main.can`:

```can
package asset_types
    provides [allowed, matches_mime]
    uses []

fn bool allowed
    emits {}
    given
        str extension
    asserts
        html: ".html" => ok true
        shell: ".sh" => ok false
    match extension
        ".html" | ".css" | ".js" | ".mjs" | ".json" | ".txt" | ".png" | ".jpg" | ".jpeg" | ".webp" | ".woff" | ".woff2" => ok true
        _ => ok false

fn bool matches_mime
    emits {}
    given
        str extension
        str mime
    asserts
        html: ".html", "text/html" => ok true
        mismatch: ".html", "text/javascript" => ok false
        script: ".js", "text/javascript" => ok true
    match extension
        ".html" => ok mime is "text/html"
        ".css" => ok mime is "text/css"
        ".js" | ".mjs" => ok mime is "text/javascript"
        ".json" => ok mime is "application/json"
        ".txt" => ok mime is "text/plain"
        ".png" => ok mime is "image/png"
        ".jpg" | ".jpeg" => ok mime is "image/jpeg"
        ".webp" => ok mime is "image/webp"
        ".woff" => ok mime is "font/woff"
        ".woff2" => ok mime is "font/woff2"
        _ => ok false
```

Use `{"source_root":"src","error_registry":"can.errors.json"}` as the
project manifest and `{"active":[],"retired":[]}` as the error registry.
For filenames, obtain the extension with `path::extension(name)` and match the
`ok str extension` value using the same table. Add `path` to the package's `uses`.
These comparisons are exact: unknown extensions, uppercase spellings and a MIME
mismatch take their written false branch. Group only cases with the same result;
`.js`/`.mjs` and `.jpg`/`.jpeg` share MIME types in this example.

This expresses an application's allowlist. It does not inspect file bytes or
establish their media type. Keep content validation separate where required.

## Verification scope

On 2026-09-30, the example and the updated application's three extension helpers
passed parsing and `inspect-types` with
`canlc 0.2.0+7fa98c82a7ab.feea7763a822`. The example's checks used a temporary
project, reclaimed immediately afterward. The updated helper bodies also matched
the formatter output; unrelated existing formatting was preserved.

Commands are `canlc parse FILE`, `canlc inspect-types PROJECT`, and
`canlc format FILE`. With a bundled toolchain, use `canlc assert PROJECT` to run
the attached assertions. Runtime assertions were not executed in this round:
the available editor compiler returned `CAN-DIST-UNBUNDLED`. Static type checking
is not an assertion result or an application integration test.
