# Integrated invoice action and browser wire contract

**Status after the upgrade (24 September 2026):** Partially implemented accepted contract. Actions, typed routes, forms and browser Fetch exist, but the shared request-aware binding and linked invoice client/server path specified below remain unfinished. The illustrative grammar below is not the current compiler grammar. See the [current reconciliation](../post-upgrade-reconciliation-2026-09-24.md).

24 September 2026. This is the selected **planned** P8 specification, not current Can behavior. It reconciles the [HTML action packet](action-contract-design.md), [typed capture increment](route-capture-design.md), [keyed form rows](keyed-row-design.md), [browser target](browser-target-design.md), [JSON-action judgments](jev-json-action/findings.md) and [grid-read judgments](jev-grid-read/findings.md). Earlier exact-route/query and malformed-form examples are historical alternatives; this document governs the selected invoice shape. No compatibility alias is required.

## Source declarations and boundaries

The shared contract package exports transparent `invoice_key`, HTML `invoice_form`, JSON `grid_edit_input`, JSON `grid_edit_outcome`, JSON `grid_load_outcome` and three action symbols. The selected declaration grammar is `action <name>`, one `post` or `get` literal route, `captures <record>`, exactly one input line, `returns <variant>`, `body html|json`, then one exhaustive `cases` table. HTML POST has `form <wire> limit <bytes> [rows_limit <count>]`; JSON POST has `json <wire> limit <bytes>`; JSON GET has `input none` and no request body. GET with request input or HTML output is outside the first grammar and diagnosed. JSON cases use `status` without `swap`; `swap inner` is HTML-only. The declaration's canonical package-instance identity plus name defines its static symbol. Method, route, capture record, wire and result types and case table are compile-time metadata, never an unchecked runtime descriptor. JSON POST admits 8192 bytes; HTML admits 2048 bytes and 64 rows.

```can
action save_invoice_html
    post "/tenants/:tenant_id/invoices/:invoice_id"
    captures invoice_key
    form invoice_form limit 2048 rows_limit 64
    returns edit_outcome
    body html
    cases
        saved status 200 swap inner
        invalid status 422 swap inner
        conflict status 409 swap inner
        forbidden status 403 swap inner
        unavailable status 503 swap inner

action save_invoice_grid
    post "/api/tenants/:tenant_id/invoices/:invoice_id"
    captures invoice_key
    json grid_edit_input limit 8192
    returns grid_edit_outcome
    body json
    cases
        grid_saved status 200
        grid_invalid status 422
        grid_conflict status 409
        grid_forbidden status 403
        grid_unavailable status 503

action load_invoice_grid
    get "/api/tenants/:tenant_id/invoices/:invoice_id"
    captures invoice_key
    input none
    returns grid_load_outcome
    body json
    cases
        grid_loaded status 200
        grid_load_forbidden status 403
        grid_load_unavailable status 503
```

`grid_edit_input` is an ordinary transparent JSON-codec record with `str operation_id`, `str revision`, and `grid_line_wire[] lines`; each line keeps `str key`, `str id`, `str quantity`, `str price` together. The server validates a 64-row maximum, unique stable row keys, complete members, domain numbers and protected ownership. The first result wire shapes are fixed:

| Record leaf | Direct fields |
| --- | --- |
| `grid_problem` | `option::value<str> row_key`, `str field`, `str message`; field names come from checked wire members or a form-level marker. |
| `grid_snapshot` | `str revision`, `grid_line_wire[] lines`, `int total_minor_units`; only authorized public projections enter it. |
| `grid_saved` | `str operation_id`, `grid_snapshot acknowledged`. |
| `grid_invalid` | `str operation_id`, `grid_edit_input draft`, `grid_problem[] errors`. |
| `grid_conflict` | `str operation_id`, `grid_edit_input draft`, `str message`. |
| `grid_forbidden` | `str operation_id`, `str message`; no protected data or echoed draft. |
| `grid_unavailable` | `str operation_id`, `str message`; the browser retains its own draft. |
| `grid_loaded` | `grid_snapshot current`. |
| `grid_load_forbidden` / `grid_load_unavailable` | `str message` each; no protected snapshot. |

`grid_edit_outcome` is the closed variant of its five distinct save leaves; `grid_load_outcome` contains exactly its three load leaves. The load action rechecks server-derived actor, membership, tenant and invoice on every request. It serves initial load and explicit reconciliation read with one typed snapshot shape. A fresh snapshot cannot alone prove whether a specific lost save committed if another writer intervened; identical-ID replay against the durable ledger settles that attempt. All result types must be codec-admissible. Owner records cross only through transparent wire records and owner factories/projections. Bun and browser use the same strict JSON codec contract, including exact Can `int` behavior.

`action::url` is checked against the action symbol and typed captures; `action::post` binds the HTML action to checked form field/row names and an admitted target. `action::request` constructs a typed browser Fetch request only for a JSON action, with method, canonical URL and response codec plus request codec/body limit for POST; GET has no body. `browser::fetch` returns `action::response` with a finite typed application outcome plus named transport, abort, codec and unexpected-status failures; it never interprets HTML as JSON. HTML `action::mount` binds three distinct callables: valid-wire handler `(request, invoice_key, invoice_form) -> edit_outcome`, normal renderer `edit_outcome -> html::safe`, and pre-handler rejection renderer `form::rejected<invoice_form> -> html::safe`. JSON POST mount binds `(request, invoice_key, grid_edit_input) -> grid_edit_outcome`; JSON GET mount binds `(request, invoice_key) -> grid_load_outcome`. Both use their declared response codec. Malformed JSON POST is pre-handler 400, not fabricated `grid_invalid`. Catalogue signatures express these callback arities and finite error bounds; compiler diagnostics reject mismatched symbols, method, body mode, field, renderer or result leaf.

## Request, operation and response policy

The guarded native Bun route table owns method/path matching. Typed captures follow canonical int64 and strict single-segment rules in the [capture packet](route-capture-design.md); valid keys are **not** permissions. All three routes enter the same lifecycle and server-controlled actor lookup; state-changing POSTs additionally enforce CSRF and body budgets. The protected read checks tenant membership, invoice ownership and disclosure policy. The protected write checks membership, invoice/tenant and expected revision at the write, with a database transaction predicate preventing a stale or revoked actor from committing. Foreign and nonexistent invoices yield the same nondisclosing 403 after authentication; malformed captures get 400 before protected entry. Each transport adapter maps the shared domain decision to its own typed cases. Neither decoder mints an owner record or permission.

For grid writes, `operation_id` is scoped to authenticated actor and invoice, validated as a bounded opaque client-generated identifier, and stored durably with a digest of the canonical request and committed response in the same transaction as the invoice mutation. A same-ID, same-digest replay returns the prior committed result without a second effect; same ID with different digest is a conflict. A 503, dropped connection, abort or view disposal does not imply rollback. The browser retains the identified attempt and may deliberately retry the **identical** payload and ID or make an authorized read to learn current state. It never creates a new ID and blindly retries after uncertainty. The HTML form has no automatic replay guarantee; after uncertainty it rereads before another write. A bounded retention window and cleanup rule must preserve the documented reconciliation path. No reload-durable browser draft or automatic offline queue is promised.

The HTML action preserves known raw controls and row order in `form::rejected` for structural 422 with a separate safe renderer; fully decoded domain-invalid forms use the typed `invalid` case. Keyed rows prevent cross-row misassociation. HTML 200/422/409/403/503 have actual statuses and a verified visible inner-swap policy; 503 must replace stale feedback. JSON save has those actual statuses with typed bodies; JSON GET has 200/403/503. Malformed capture/JSON gets 400, unmatched path 404, wrong method 405, body limit 413, wrong media 415, and unexpected infrastructure failure 500; these remain separate adapter outcomes and cannot become a fake success. Renderer failure after a committed write may produce 500 with uncertain state. On save, status, `operation_id` and typed result must agree; mismatch is a client protocol failure.

## Qualification boundary

Compile all three declarations from the same locked contract package, change route, field, leaf and body mode to prove checked linkage, and reject server-only capabilities from the browser closure. Real HTTP/browser/database tests cover initial GET and reread, every declared status, malformed form/JSON, keyed reorder/omission, cross-tenant/nonexistent denial, concurrent revision/membership changes, same/different digest replay, lost acknowledgement after commit, late response after view disposal, visible 503, exact-number codec parity and static route precedence. Observe rows, bytes, DOM, focus, announcements and resources; disposable probes are not production qualification.
