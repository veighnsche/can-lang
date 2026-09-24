# DI-02 current-Can validated-value cases

24 September 2026. Registered before the first build. Source revision:
`02a549d28fd5fc5c3996160e65a97de332390d30`. This is a hand-authored
cross-package mechanism probe, not an agent comparison or syntax selection.
Use the pinned Bun archive to build the compiler from this checkout. Keep all
generated TypeScript and modified case copies under `/private/tmp`.

The `values` package publicly exports nominal `email` and `quantity` records.
Its factories reject an empty email and nonpositive quantity. The `main`
package imports them. Cases and expected observations:

1. **Valid control:** importers call both factories with accepted inputs, then
   observe the resulting fields. Expect checker admission and passing real-Can
   assertions.
2. **Rejected factory inputs:** call both factories with an empty email and
   zero quantity. Expect their named validation errors, proving the factory
   predicates ran.
3. **Invalid direct construction:** the importer directly constructs an empty
   `values::email` and zero `values::quantity`. Expect checker admission and
   runtime fields containing those inputs, bypassing factory predicates.
4. **Invalid copy-update:** the importer gets valid factory values, then uses
   `with` to replace their fields with an empty email and zero quantity.
   Expect checker admission and invalid runtime fields.
5. **Invalid generic JSON decode:** the importer passes structurally valid
   JSON with semantically invalid field values to
   `codec::decode_json<values::email>` and
   `codec::decode_json<values::quantity>`. Expect checker admission and decoded
   nominal values with the invalid fields; no factory call.
6. **Assertion fixture path:** if the grammar permits a foreign constructor in
   assertion arguments or supplied completions, exercise one such path. It
   may establish fixture admission, but only a real function path can establish
   runtime constructor/decode behavior.

Record diagnostics and assertion evidence. A passing assertion with an expected
invalid field is a successful demonstration of a *broken invariant*, not an
expected test failure. Do not infer actor authorization from structural tenant
identifier validation.
