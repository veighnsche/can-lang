# DI-09 action-contract design packet

24 September 2026. Bounded preparation for fresh Jev advice and engineering
selection under the user's later no-questions instruction. **No
surface, capture policy, target guarantee or implementation is adopted here.**
This packet uses the [server evidence](server-evidence.md), [browser evidence](browser-evidence.md),
[product alternatives](product-alternatives.md#di-09--linked-server-and-client-contracts),
[current invoice probe](invoice-current-probe.md), [saved Jev advice](jev-product-contracts/README.md),
[confirmed syntax choices](confirmed-syntax-choices.md), and [evaluation protocol](evaluation-protocol.md).
Their implementation baseline is `02a549d28fd5fc5c3996160e65a97de332390d30`.
No new Jev consultation or executable probe was performed for this packet.

There are three comparisons: the strongest current idiom, **A: a source action
declaration**, and **B: a manifest action descriptor with generated ordinary
catalogue calls**. A and B expose the same bounded semantics. Captures are an
optional second increment within either surface, not a third language design.
All new `action`, field-reference and policy APIs below are illustrative
proposals, not existing catalogue entries or compiler-checked examples.

## Shared workload and independent boundaries

One actor edits an invoice using raw `seats`, optional `details`, and `revision`
form fields. The exact-path baseline posts to
`/invoice/edit?tenant_id=1&invoice_id=7`; the capture increment posts to
`/tenants/1/invoices/7`, from the pattern
`/tenants/:tenant_id/invoices/:invoice_id`. Both name the same kind of business
operation, but are separate trial configurations, not simultaneous compatibility
aliases. There is no backward-compatibility requirement.

| Layer | Proposed checked promise | What remains outside that promise |
| --- | --- | --- |
| Endpoint | Canonical package instance plus declaration identity links method, route, input and response cases; a URL builder uses that declaration. | A successful deployment, reachable server, authenticated actor, authorized invoice, or present DOM node. |
| Wire form | Controls and errors can refer to a shallow declared field; strict decoding produces raw strings. | Numeric bounds, meaningful revision, nested rows, business validation, or permission. |
| Domain operation | Existing explicit server context and a protected operation perform validation and authorization. | No public path/form/ID value is evidence of live authorization. DI-09 adds no permission capability. |
| Response mapping | Every finite application case has a real status and an explicit display policy. | Unexpected infrastructure responses, target presence, focus, announcements, persistence or commit certainty. |
| Observation | Companion HTTP, browser and database scenarios inspect their own layers. | Bare `=> ok` for an opaque response does not inspect any of these outcomes. |

Authentication/CSRF rejection must occur before a protected mutation. Resolve
the actor from server-controlled state and check membership/resource access at
the read/write operation, including tenant/revision predicates and a specified
concurrency strategy. A request-local earlier read does not freeze membership.
These obligations hold under all three comparisons and do not belong in the
choice between declaration spellings.

The five application cases are:

| Case | HTTP | Body and requested display in the proposed trial |
| --- | --- | --- |
| `saved` | 200 | Safe form fragment with acknowledged revision and success feedback; inner swap. |
| `invalid` | 422 | Retain exact rejected wire strings when decoding succeeded; field/form errors; inner swap. A malformed form has no fabricated typed raw record. |
| `conflict` | 409 | Preserve the submitted draft and show stale-revision feedback; inner swap. Do not silently replace the draft with server data. |
| `forbidden` | 403 | Generic denial with no invoice/tenant disclosure; inner swap. Do not echo protected server data. |
| `unavailable` | 503 | Preserve draft and show save uncertainty/retry guidance; inner swap. Never imply that retry is safe solely because this status was returned. |

This is a **candidate display fixture**, not a final product choice. Keep the
failure statuses; rewriting 409/403/503 as 200 to obtain a swap fails the trial.
A 503 after an uncertain commit needs reconciliation before another mutation.
Transport failure, malformed response, unexpected status and rendering failure
remain explicit adapter/observation outcomes outside these five domain cases.
The application result variant is not a serialized envelope in an HTML fragment.
A future JSON/browser action needs its own declared codec; it cannot decode
this HTML body as `edit_outcome` merely because source types are shared.

## Strongest current idiom: ordinary types and named helpers

Use the current nominal record/variant grammar for the finite result, explicit
raw validation, a protected operation, a total renderer, and one status mapper.
For example, these declarations need no new action grammar:

```can
package invoice_contracts
    provides [invoice_key, invoice_form, field_error, saved, invalid,
              conflict, forbidden, unavailable, edit_outcome]
    uses [option]

record invoice_key
    int tenant_id
    int invoice_id

record invoice_form
    str seats
    option::value<str> details
    str revision

record field_error
    str field
    str message

record saved
    invoice_form acknowledged
record invalid
    option::value<invoice_form> raw
    field_error[] errors
record conflict
    invoice_form raw
    str message
record forbidden
    str message
record unavailable
    invoice_form raw
    str message

variant edit_outcome
    saved
    invalid
    conflict
    forbidden
    unavailable
```

The baseline `field_error.field` is a conventionally centralized string, not a
checked field identity. A named renderer can explicitly branch over these five
cases, retain the raw form, construct controls with safe HTML helpers, and use
`html::text` for all submitted text. A domain record may independently name the
converted quantity `seat_count`; that does not alter the wire name `seats`.

This complete status helper uses existing syntax; an application response helper
can combine it with its finite-case renderer and `http::response_html`:

```can
fn int response_status
    emits []
    given
        invoice_contracts::edit_outcome result
    match result
        invoice_contracts::saved => ok 200
        invoice_contracts::invalid => ok 422
        invoice_contracts::conflict => ok 409
        invoice_contracts::forbidden => ok 403
        invoice_contracts::unavailable => ok 503
```

Use `http::make_body_status` to admit these numeric statuses. Define a named
request handler that extracts `http::query_one(req, "tenant_id")` and
`http::query_one(req, "invoice_id")`, rejects missing/duplicate/bad IDs, calls
`http::request_form<invoice_contracts::invoice_form>(req, 2048)`, validates its
raw record, and enters the protected operation. Map body-limit/media/codec
failures explicitly; retained fields are available only after successful decode.
The [probe's implementation](probes/invoice-current/src/main/main.can) demonstrates
the present validation/response shape, but its SQL descriptor is not connected
to a live protected write. Its 18 passing roots are not HTTP/DOM/database proof.

For positive integer resource keys, a complete centralized current URL helper is:

```can
fn html::url edit_url
    emits [html::invalid_url]
    given
        invoice_contracts::invoice_key key
    match call html::parse_url("/invoice/edit?tenant_id=" +
        call text::from_int(key.tenant_id) + "&invoice_id=" +
        call text::from_int(key.invoice_id))
        html::invalid_url
        ok html::url address => ok address
```

This example's integer serialization avoids raw user-text interpolation; it does
not validate positivity or authorization. Arbitrary string query values should
use the existing `url::with_query`/`url::query_pair` encoding facilities. Mount
with the current literal call `http::route_post("/invoice/edit", callable
handle_edit)` and build `htmx::post` from `edit_url`. Centralize field attributes
in helpers such as `seats_name()` and target spelling in `feedback_target()`.
The literal mount constraint means an ordinary shared path variable cannot
replace the mount literal today. Centralized helpers plus protocol tests are
therefore a stronger honest baseline than a claim of universal static linkage.

Current HTMX configuration swaps 200/422 and excludes 409/403/503. Consequently,
this baseline can express all five correct HTTP responses but **cannot promise
the fixture's three additional visible swaps through today's exposed policy**.
Adding a narrowly checked catalogue policy or repairing compiler-owned HTMX
configuration can address that gap without new Can grammar. It is still a
runtime/catalogue change requiring browser evidence. Authored script escape
hatches are outside the present browser boundary.

## Common proposed semantics for A and B

1. **Identity before captures.** The first prototype owns an exact literal path,
   method, typed query key, shallow form and finite response mapping. All mount,
   action and URL operations refer to that identity. A path change regenerates
   linked URLs; a declaration rename diagnoses stale references. A method change
   diagnoses a POST-specific consumer. Two equal-looking declarations are not
   interchangeable merely because their shapes match. Duplicate method/path
   mounts still fail in the assembled router.
2. **Ordinary raw data.** Keep current shallow `str`, `option::value<str>` and
   `str[]` decoding, unknown-field rejection and scalar-duplicate rejection.
   Missing `details` and present-empty `details` stay distinct. Missing required
   scalar fields fail. `form::field<invoice_form, str>` below is a proposed opaque
   reference to one direct field, carrying owning record identity and wire name;
   it is not a string path language or validator. `form::name`, `form::read`, and
   `form::error` use that reference for control name, retained value and error
   association. References to an optional or repeated field retain that exact
   field type. No implicit nested paths, row zipping or numerical coercion.
   In the two candidates, replace the baseline `field_error[] errors` member
   with proposed `form::problem<invoice_form>[] errors`. `form::error(field,
   message)` constructs a problem for that field's owning wire record;
   `form::form_error<invoice_form>(message)` constructs a form-level problem.
   This opaque problem preserves field ownership even when field value types
   differ; leaving the baseline arbitrary string member would lose that claim.
3. **Finite mapping.** The checked table covers each leaf exactly once, checks
   status/body/swap compatibility, and rejects duplicate/missing/unknown cases.
   A safe renderer accepts the whole `edit_outcome` and handles its cases using
   ordinary exhaustive matching. It is supplied at the server binding, so shared
   declarations do not import server SQL or renderer modules into a browser.
   Rendering errors produce an explicit infrastructure response, not an invented
   `saved` or converted 200. Adding a result case must invalidate the table and
   any incomplete renderer. Strict typo/exhaustiveness behavior coordinates with
   the confirmed DI-01 `bind` change; it is not silently assumed implemented.
4. **Policy belongs to the actual client adapter.** A generated HTMX binding uses
   the declared method, URL and response policy with a caller-supplied target.
   Compilation verifies compatibility; the pinned asset must actually implement
   it. If the current native `noSwap` setting is page-global, the prototype must
   either validate one compatible page policy or implement a qualified scoped
   adapter. It cannot promise endpoint-local behavior by emitting conflicting
   global configurations. Unexpected statuses remain no-swap with observable
   failure. Generic `html::url` conversion deliberately loses endpoint identity;
   code using that escape boundary receives only URL-safety guarantees.

For a concrete bounded comparison, both candidate adapters use the same proposed
boundary table below. These are fixture semantics to review, not an implicit
claim that today's decoder already returns these HTTP statuses:

| Boundary | Candidate result |
| --- | --- |
| Invalid query/capture conversion | 400; do not call the protected handler. |
| Unmatched path / matched path with wrong method | 404 / 405, with appropriate method metadata; no business callback. |
| Body over 2048 bytes / unsupported form media type | 413 / 415; no business callback; no swap under the five-case policy. |
| Malformed encoding, missing required scalar, duplicate scalar or unknown field | `invalid(none, [form-level problem])`, rendered as 422; do not invent retained typed data. |
| Valid wire form | Call `handle_edit(request, key, raw)` once; it returns `edit_outcome` with `emits []`, explicitly mapping its validation/auth/storage results. |
| Safe rendering | Call `render_edit(outcome)`; allowed declared rendering failures are caught by the adapter and produce a generic 500/no-swap response. |

These adapters need to preserve enough native decoder failure information to
distinguish 413/415/422; if current errors conflate a case, that is explicit adapter
work. Authentication/CSRF policy is still the handler's server-side obligation;
the adapter must not turn rejection into a protected operation. Rendering a
committed `saved` result can fail after the write: a 500 response then does not
prove rollback. The malformed-form branch can render only generic feedback and
public form structure, never load protected invoice data without authorization.

### Optional capture increment

Both alternatives may replace the exact route/query pair with:

```text
POST /tenants/:tenant_id/invoices/:invoice_id
captures invoice_key { tenant_id: int, invoice_id: int }
```

This notation explains the contract; it is not another proposed source form.
The candidate bounded rule is one named, required, nonempty segment per field;
no wildcard, optional segment, repeated name or nested capture record. Initial
support is `str` and `int`, with no automatic domain/permission construction.
The same `invoice_key(1, 7)` builds `/tenants/1/invoices/7`. The builder percent
encodes each segment once and the matcher decodes once. Missing/extra key fields
and wrong field types are compile errors. Invalid encoding/conversion rejects
before invoking the business handler; the HTTP mapping must distinguish malformed
path (candidate 400), unmatched route (404) and wrong method (405).

Freeze these details before a capture implementation: exact decimal `int`
grammar/range behavior and canonical output; Unicode; empty segments; trailing
slashes; raw and encoded slash/backslash; percent-encoded percent; dot segments;
duplicate captures; overlap with static routes; method precedence and ambiguous
patterns. A conservative trial can reject overlapping patterns and segment
values whose decoding changes route structure, including encoded separators,
instead of silently selecting a winner. That is a candidate trial restriction,
not a settled routing policy. Native Bun/URL normalization must be probed before
claiming round trips. A builder success implies admitted route syntax, never
resource existence or actor permission.

## A — Source declaration with checked catalogue operations

In the `invoice_contracts` package above, add `save_invoice` to `provides` and
declare the following **proposed new source syntax**:

```can
action save_invoice
    post "/invoice/edit"
    query invoice_key
    form invoice_form limit 2048
    returns edit_outcome
    body html
    cases
        saved status 200 swap inner
        invalid status 422 swap inner
        conflict status 409 swap inner
        forbidden status 403 swap inner
        unavailable status 503 swap inner
```

For the capture trial, replace only the `post` and `query` lines with:

```can
    post "/tenants/:tenant_id/invoices/:invoice_id"
    captures invoice_key
```

The declaration is a canonical action symbol, not a callable handler. The new
catalogue `action` operations consume that symbol as a checked static input,
analogous in purpose to existing static descriptor checks. New `form::field`
checks a direct field-name literal against its type argument; it needs catalogue
and checker work, but no special field-selection expression grammar:

```can
package invoice_page
    provides []
    uses [billing::invoice_contracts as invoice, action, form, html, htmx]

// Illustrative call expressions within normal functions/chains:
// Checked path/query builder; emits action::invalid_path for dynamic values.
call action::url(invoice::save_invoice, invoice::invoice_key(1, 7))
// Checked wire reference; result form::field<invoice::invoice_form, str>.
call form::field<invoice::invoice_form, str>("seats")
// Generated POST attributes include endpoint policy and supplied target.
call action::post(invoice::save_invoice, invoice::invoice_key(1, 7), target)
// Mount checks handler(request, key, raw) -> edit_outcome and renderer type.
call action::mount(invoice::save_invoice, callable handle_edit, callable render_edit)
```

The final call is server-only and belongs in a server package, not the shared
browser-safe module. `target` is supplied by the caller through a checked ID
constructor; target absence is treated below. `handle_edit` receives the opaque
request plus decoded key and raw form, derives server context and calls the
protected operation. The adapter applies the common boundary table above.
The renderer's allowed finite error set is supplied by its checked signature;
native/unexpected failures follow the server's infrastructure fault policy.
Exact catalogue signature spelling is still specification work, not magic
behavior inferred from this five-case declaration. The excerpt shows contract consumption, not an executable
server or a grammar proposal for comments/top-level call expressions.

**Cost and boundary.** A adds a top-level declaration and its contextual sections,
symbol resolution/export rules, checker metadata and catalogue operations. It
keeps contracts near ordinary types but must define how an action symbol is
passed without turning it into arbitrary mutable runtime data. No new renderer,
validator, permission, pattern, or component grammar is required by this option.

## B — Manifest descriptor with ordinary generated calls

Keep the ordinary Can types above. In the project that owns `invoice_contracts`,
the following **proposed manifest extension** defines the same exact-path action:

```json
{
  "actions": {
    "invoice_contracts::save_invoice": {
      "export": true,
      "method": "POST",
      "path": "/invoice/edit",
      "query": "invoice_contracts::invoice_key",
      "form": {"type": "invoice_contracts::invoice_form", "max_bytes": 2048},
      "result": "invoice_contracts::edit_outcome",
      "body": "html",
      "cases": {
        "invoice_contracts::saved": {"status": 200, "swap": "inner"},
        "invoice_contracts::invalid": {"status": 422, "swap": "inner"},
        "invoice_contracts::conflict": {"status": 409, "swap": "inner"},
        "invoice_contracts::forbidden": {"status": 403, "swap": "inner"},
        "invoice_contracts::unavailable": {"status": 503, "swap": "inner"}
      }
    }
  }
}
```

This is the complete proposed action section, merged with existing required
project fields; it is not a currently valid full `can.project.json`. For the
capture trial change `path` to `/tenants/:tenant_id/invoices/:invoice_id`, remove
`query`, and add `"captures": "invoice_contracts::invoice_key"`. Field names/types
are derived from that record, so the manifest does not separately duplicate them.

The compiler checks symbolic type/case strings, then synthesizes these public
ordinary call surfaces into the owner's package namespace:

```can
package invoice_page
    provides []
    uses [billing::invoice_contracts as invoice, form, html, htmx]

// Same ordinary call syntax; these functions do not exist today.
call invoice::save_invoice_url(invoice::invoice_key(1, 7))
call form::field<invoice::invoice_form, str>("seats")
call invoice::save_invoice_post(invoice::invoice_key(1, 7), target)
// Server-only generated mount operation:
call invoice::save_invoice_mount(callable handle_edit, callable render_edit)
```

Generated signatures carry the same route identity and callback/error semantics
as A; they are not arbitrary application wrappers that lose endpoint identity.
Public names, collisions with authored members, generated documentation, target
availability, private types and manifest ownership need explicit checking. The
server-only mount symbol must be rejected in a browser build while URL/form/post
metadata must not drag its handler or renderer implementation into that bundle.
Descriptors resolve within the declaring project's dependency environment, not
the consuming root's similarly spelled packages. There is no manifest import
handle table: that would contradict the already selected DI-05a direction.

**Cost and boundary.** B needs manifest schema/loading, typed symbol resolution,
synthetic package members, catalogue/checker/emitter support and the same runtime
adapters as A. It introduces no new Can grammar beyond the separately confirmed
dependency-qualified imports. A JSON key being a string does not prevent static
checking; the compiler must resolve it as a symbol and produce file/key-aware
diagnostics. Conversely, merely loading the strings dynamically would not meet
the proposal. B trades one new source declaration for a second checked authoring
surface and generated API names; this is an agent-task experiment, not a claim
that manifest placement is inherently simpler.

## Imports, field rename and absent targets

The examples honor the selected import form:

```can
uses [billing::invoice_contracts as invoice, crm::invoice_contracts as crm_invoice]
```

`billing` and `crm` are direct dependency keys; the canonical identity contains
the resolved project/package instance. Changing the local alias changes spelling,
not endpoint identity. Two dependencies defining `save_invoice` or `invoice_form`
remain distinct. An action from `invoice` cannot accept a same-shaped field
reference from `crm_invoice`. Inside either dependency, lookup uses its own direct
dependency table. These are planned DI-05 integration requirements, not existing
compiler behavior demonstrated by this packet.

| Controlled edit | Current idiom | A and B promised check |
| --- | --- | --- |
| Rename wire `seats` to `quantity` | Update raw record/validator, centralized control helper, error strings and protocol fixtures. Unchecked strings can survive. | Old `form::field<invoice::invoice_form, str>("seats")` fails; replacement `"quantity"` generates the new wire name. Old raw member reads also fail. Unknown submitted `seats` is rejected at runtime. |
| Rename only domain `seat_count` to `licensed_seats` | Update validator/domain consumers; keep wire `seats`. | Same: no wire rename is inferred from a domain rename. |
| Change exact wire path | Builder and mount literal can drift; protocol tests catch mismatch. | One declaration/descriptor change rebuilds linked URLs and mount. Arbitrary pasted strings outside the linked API still require tests. |
| Rename endpoint symbol | Ordinary helper consumers fail if renamed; unrelated strings may survive. | A's action references or B's generated calls fail; checked manifest references diagnose stale symbols. |
| Change method POST to PUT | Mount/action mismatch can survive string/type checks. | POST-only use fails; native client method support must exist before admitting the new binding. |
| Add a sixth result leaf | Exhaustive helper checks are the baseline, subject to current pattern limitations. | Mapping plus incomplete renderers fail; no silent default-success case. |

For both candidates, this target construction remains deliberately separate:

```can
call htmx::target_id("invoice_feedback")
call html::text_attribute("id", "invoice_feedback")
```

Centralize these in a helper for the baseline and the two trials. A future typed
render-scope target can diagnose ID-reference drift, but is not required by this
action proposal and is not a third candidate here. Neither a valid ID nor a typed
scope proves that a conditional element exists when a delayed response arrives.

Test a page where `invoice_feedback` is omitted initially, then a page where it
is removed after submission. The required bounded behavior is **no swap into an
unrelated node and an observable target-missing client failure**, with a recorded
HTTP outcome and unchanged durable-effect interpretation. The exact native HTMX
error/event and any user-visible fallback must be established on the pinned asset.
The current catalogue may need an adapter to expose that failure; this packet
does not claim one exists. A present target is a precondition for the five visible
swaps. A product promise to always show feedback requires a separately tested
stable fallback or render-presence guarantee. Slow/out-of-order replies additionally
need a submission/revision policy; endpoint typing does not solve stale DOM writes.

## What needs a syntax answer, and what does not

| Work | New Can grammar? | Decision boundary |
| --- | --- | --- |
| Explicit context, raw validation, finite variants, renderer/status helpers and companion tests | No | Already expressible; complete the real invoice operation and observations. |
| Checked literal field-reference API | No | Catalogue/checker extension; exact API spelling still belongs in reviewable design. |
| HTMX status policy plus missing-target observation | No | Catalogue/runtime/controlled-asset work, with pinned browser tests. |
| Exact route identity through B's checked manifest and generated calls | No | Requires manifest/API design, not just an ordinary helper library. |
| A's `action` declaration and static action-symbol references | Yes | User choice between A, B, or retaining the current idiom; settle exact declaration sections only if A is chosen. |
| Required typed captures and URL builder | Not necessarily | Route-checker/router/adapter work is required in both; A adds a section, B a manifest key. Capture semantics need evidence before adoption. |
| Dependency-qualified `uses` | Separately planned | Already selected by the user; do not reopen it as part of DI-09. |
| Authorization, nested form rows, Can browser execution, typed render scopes | Not determined here | Separate contracts/workstreams; no implied approval from selecting A or B. |

The later user-facing comparison should show the full declaration/descriptor and
consumer examples together. The material choice is whether the linked contract
is authored beside Can types or as a checked manifest descriptor with generated
calls. Keep exact-path identity, fields and finite cases separable for engineering
measurement; do not force the capture increment or a new component system into
that syntax choice. No selection is made in this packet.

## Bounded verification and advancement conditions

Register the same creation, rename and diagnostic-repair tasks under the existing
[evaluation protocol](evaluation-protocol.md), including a held-out action with
different field/case names. Record the strongest current helpers as the baseline;
do not compare either candidate only with an unfactored renderer.

- **Static:** wrong dependency action/field, nonexistent field, wrong field type,
  required/optional/repeated mismatch, stale wire name, method mismatch,
  missing/duplicate response mapping, incomplete renderer, private type leak,
  duplicate mounts, and browser use of a server-only generated function.
- **Protocol:** 200/422/409/403/503 retain real statuses and safe bodies; malformed,
  duplicate, unknown, absent and empty form values follow the declared decoder;
  captured URL construction/matching round-trip or reject precisely; a stale
  pasted action string demonstrates the limit of the checked API.
- **DOM:** observe all five present-target swaps, retained invalid/conflict text,
  escaped hostile data, focus and error announcements; initial and late target
  absence; target rename; responses arriving out of order; unexpected status and
  failed transport. Specifically reproduce the missing 503 browser case before
  claiming this gap repaired.
- **Protected effects:** actor A cannot modify tenant B; path/query substitution,
  stale revision and concurrent authorization change do not bypass the protected
  write; inspect persisted rows. A 200 fragment alone is insufficient. Reconcile
  uncertain commit before retry; preserve the original operation identity.

Advance a new mechanism only after its compiler/runtime contracts pass and a
reproduced baseline failure or material held-out agent benefit survives the
comparison. Correct behavior and safe edits come before complete successful-task
token cost. The existing Jev results prioritize exact endpoint identity,
wire-derived field references and finite action mapping, but do not choose A/B,
prove security or establish the missing browser effects. No runtime files were
edited; runtime lint/format/check and product tests are not claimed as run for
this document-only packet.
