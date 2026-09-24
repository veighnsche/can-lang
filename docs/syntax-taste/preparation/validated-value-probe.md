# DI-02: current cross-package validated-value probe

24 September 2026. This is an executed baseline, not a new contract or syntax
choice. The [registered cases](probes/validated-value/case-registration.md) and
[complete source project](probes/validated-value/can.project.json) use two Can
packages: [`values`](probes/validated-value/src/values/values.can) exports nominal
`email` and `quantity` records plus factories; [`main`](probes/validated-value/src/main/main.can)
imports them. The factory predicates are intentionally bounded: email rejects
the empty string, and quantity rejects zero and negative numbers. This probe
tests whether those predicates govern every value of the exported nominal types;
it does not assert full email-format validation.

The cases were fixed before compilation. The checkout was
`02a549d28fd5fc5c3996160e65a97de332390d30`. I built `canlc` from that
source with `.local-deps/bun-darwin-aarch64.zip`, copied the project to
`/private/tmp/can-validated-value-probe`, and built and ran the copy. Generated
TypeScript remained in that temporary project. The initial authoring build
diagnosed `ordinary Boolean match requires false before true` in both factory
branches; ordering the Boolean arms `false`, then `true` resolved it. The final
build returned `validation: verified`, **11/11 assertion roots passed**, and
`evidence: ["real-can", "supplied-completion"]`. `canlc run` exited 0.

| Registered path | Current checker | Observed runtime or assertion outcome |
| --- | --- | --- |
| Valid importer calls to both factories | Accepted | Real Can calls return `email("valid@example.test")` and `quantity(3)`; importer observes the expected fields. |
| Rejected inputs to the factories | Accepted as calls with named errors | The owner assertions observe `invalid_email("blank")` for `""` and `invalid_quantity("nonpositive")` for `0`. |
| Importer direct construction | Accepted | The importer creates `values::email("")` and `values::quantity(0)`; a real Can assertion observes both invalid fields. |
| Importer `with` after valid factory creation | Accepted | A real Can assertion observes empty address and zero amount in fresh same-type records. The factory predicate is not called for the replacements. |
| Generic JSON decode of `{"address":""}` and `{"amount":0}` | Accepted | `codec::decode_json<values::email>` and `<values::quantity>` succeed. A real Can assertion observes the invalid nominal values, rather than `codec::invalid_data` or an owner error. |
| Foreign constructor in assertion input | Accepted | `fixture_foreign_constructor` supplies `values::email("")` as an assertion argument and passes. This exercises direct construction inside the assertion harness as well as ordinary importer code. |
| Supplied completion for a foreign factory call | Accepted | In `fixture_supplied_invalid`, a `when` row supplies `ok values::email("")`; its assertion passes and the build reports `supplied-completion`. That path simulates the factory call, so it cannot establish what the real factory would return. |

The direct constructor, update, and decoder results are newly executed evidence
for the previously source-derived concern in [core evidence](core-evidence.md#authored-validated-values-and-decoding).
They establish that a public nominal record plus an optional validating factory
does **not** make the predicate an invariant of every inhabitant. The fixture
result establishes a distinct assertion-admission route. The generated decoder
and current runtime projector explain the mechanism: a structural JSON match
can create a nominal record without invoking an authored factory. This probe
does not establish behavior for every codec format, JS interop route, or a
future protected-value design.

For an **owner-controlled record** candidate to make that invariant credible,
the creation boundary must close exactly these demonstrated non-owner routes:
direct constructor calls, representation-changing `with` updates, generic
document decoding into the protected type, and assertion inputs or supplied
completions that fabricate the type. Importers can instead decode a public
wire shape and call the owner's validating factory. Owner-authored factories,
constants, and update functions remain trusted minting routes and must enforce
their own predicates. Public type visibility must therefore be distinct from
non-owner construction permission; today's private-record rule cannot express
that boundary because a private type cannot appear in a public signature.
Any derived codec or schema, owner projection, equality, and pattern rule
would need a separate contract before implementation. No source spelling is
selected here.

A validated `tenant_id` would establish only that its identifier passed its
structural predicate. It is not evidence that an actor may edit that tenant's
invoice. The protected operation still needs an authorization check against
the current actor, tenant context, and target resource. This probe does not
run an authorization scenario.

Reproduce from the repository root:

```sh
go run ./tools/distbuild --archive .local-deps/bun-darwin-aarch64.zip --out /private/tmp/can-validated-value-bundle --version validated-value-probe
rsync -a docs/syntax-taste/preparation/probes/validated-value/ /private/tmp/can-validated-value-probe/
/private/tmp/can-validated-value-bundle/can-validated-value-probe-bun-1.4.2-darwin-arm64-v1/bin/canlc build /private/tmp/can-validated-value-probe
/private/tmp/can-validated-value-bundle/can-validated-value-probe-bun-1.4.2-darwin-arm64-v1/bin/canlc run /private/tmp/can-validated-value-probe
```
