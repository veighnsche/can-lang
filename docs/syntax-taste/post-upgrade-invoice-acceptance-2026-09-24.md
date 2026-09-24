# Invoice composition acceptance for the next Can upgrade

24 September 2026 · implementation gate for the [selected behavior](post-upgrade-selected-behavior-2026-09-24.md)

The existing [`examples/invoice`](../../examples/invoice) server and
[`examples/invoice-grid`](../../examples/invoice-grid) browser application are
the acceptance application. Passing a new toy example, a compiler-only fixture
or the current Gate 5 harness does not pass this gate. The three selected
actions—`load_invoice_grid`, `save_invoice_grid` and `save_invoice_html`—must
come from one handler-free, locked `invoice_contract` package imported by both
targets. The server binds real Can handlers to those declarations; the browser
consumes the checked JSON load/save projections. This is a future acceptance contract,
not a claim that the present checkout passes it.

## Build and execution boundary

Start from clean, staged copies of the two existing examples and their **one**
shared dependency snapshot. The qualification runner records the exact Can
compiler/distribution, Bun, Chromium and WebKit versions, the contract
package-instance identity, both build IDs, the browser manifest and the server
report. It runs assertions and builds through the installed supported toolchain,
using the selected command shape:

```text
canlc assert <invoice-grid-project>
canlc build --target browser <invoice-grid-project>
canlc assert <invoice-server-project>
canlc build --target bun --browser-manifest <verified-browser-manifest> <invoice-server-project>
```

The last flag is the [selected proposed CLI spelling](post-upgrade-selected-behavior-2026-09-24.md#browser-build-and-event-execution),
not a current command. A clean rebuild with the same inputs must yield the
same checked identities and asset bytes. The server verifies the browser
manifest's locked shared instance and asset hashes before publication, then
serves its paired digest-named JavaScript, source map and diagnostic table from
the application's own origin under the selected CSP. The invoice page uses
that report's one script URL. An independently rebuilt or tampered browser
asset, missing map/table, mismatched shared package or failed rebuild cannot
replace the served pair; the previous verified pair remains available.

Run the resulting server entry with a seeded SQLite database and configured
public origin. For this acceptance fixture, the application itself serves a
grid page at `GET /invoice-grid?tenant=1&invoice=7` and a form page at
`GET /tenants/1/invoices/7/edit`. Open **that same server origin** in the
named browsers. The grid page's only Can module script URL must equal the
digest URL in the active server build report; the form page uses the selected
pinned HTMX asset and posts to the shared HTML action. Neither page is
test-authored or reverse-proxied. These page URLs are fixture routes, distinct
from the three shared action routes; they do not grant authorization.
The test may seed or inspect SQLite, drive browser controls, observe network
traffic and inject transport failures. It cannot provide an action handler,
HTML/JSON result, route translation, browser boot module, module alias or
runtime shim. Ordinary successful GET and save traffic must go straight to
the application server; a fault injector may abort or corrupt a selected
response only in a named failure leg. `canlc assert` evidence is useful, but
fixtures and supplied completions cannot substitute for the live handler,
database mutation or native browser behavior.

The current [test bundler](../../tests/integration/browser/build-grid.mjs)
provides a three-line boot module and four `node:` shims; the current
[origin](../../tests/integration/gate5_frontend_test.go#L403) rewrites the
captured GET to a query route. Both must disappear from this qualification
path. Its grid matrix and database inspections can be retained as **test
observations** after those dependencies are removed.

## Baseline application observations

The base contract and example must pass the following on real HTTP, Bun and
browser execution. The test records request/response bytes and statuses,
handler-entry counts, relevant SQLite rows/revisions/replay rows, DOM text,
focus, live-region announcements, browser diagnostics and asset URLs. A green
compiler build or matching emitted strings alone is insufficient.

| Gate | Required observation |
| --- | --- |
| Shared source | Both build reports resolve the shared package instance and every action/wire type they reference to the same canonical locked declaration identity. The contract has no `handles` clause, server dependency, renderer, SQL, session secret or executable handler. Neither example mirrors an action, wire record, status table or never-served client stub. A browser import of the shared package does not pull server implementation or unused HTML form/renderer code into its reachable closure. |
| Direct routes | Authenticated `GET /api/tenants/1/invoices/7` enters the bound load handler and returns the declared 200 snapshot. `POST` to that same API path enters the bound JSON save; `POST /tenants/1/invoices/7` enters the bound form save. No query-to-capture rewrite is involved. Invalid canonical captures fail before handler entry; unmatched paths give 404 and recognized paths with wrong methods give 405. The old `/invoices/load?invoice_id=…` workaround is not the application's load route. |
| Explicit service and actor | Startup opens one pool and validates one public origin. `action::mount` binds that pool and origin through typed lexical captures; each handler receives the actual `http::request` and typed `invoice_key`. The server derives the session from its HttpOnly cookie on every request, never a wire field or query. A protected read or write checks live session, membership, tenant/invoice ownership and disclosure; a write checks expected revision within the mutation transaction. Foreign and missing invoices are equally nondisclosing. POST also enforces the exact-Origin policy before replay lookup or mutation. Pool close and request-token revocation occur at their selected lifetime boundaries. |
| Finite transport outcomes | The JSON GET returns only declared 200/403/503 application cases; JSON and HTML saves return the declared 200/422/409/403/503 cases with actual HTTP statuses. Malformed capture/JSON, wrong media, oversized body, structural form rejection, missing route, wrong method and unexpected fault remain distinct pre-handler or infrastructure outcomes (400/415/413/422/404/405/500), not fabricated application leaves. HTTP bytes and decoded cases agree, including operation ID on a JSON save. HTML 503 visibly replaces stale feedback in exactly its admitted connected target. Missing-target submission and in-flight target loss report the selected finite failure; OOB/partial markup and response-control headers cannot mutate an unrelated node. |
| Protected effects and uncertainty | A successful edit changes the expected SQLite invoice and lines once; a concurrent revision or membership change blocks stale/unauthorized commit. An authorized identical operation-ID/payload replay returns the recorded result with no second effect; changed payload conflicts, and a revoked actor cannot recover a prior saved result. Losing an acknowledgement after commit does not silently roll back or mint a new operation ID. The grid preserves its draft and reconciles the identified attempt through replay or an authorized reread. Test the selected seven-day ledger boundary and revision invariant separately from ordinary happy-path saves. |
| Browser application | The compiler-owned zero-argument `main` runs once after DOM readiness on the application-served grid page and obtains bounded, canonical `tenant`/`invoice` query keys without the test passing `str[]`. The actual generated asset loads, edits, reorders, calculates, saves and reconciles the invoice. The application-served form page exercises the HTML action. A pending attempt is visible; blocked feedback survives rerender; add/move/reorder focus lands on an attached control. Late responses and view disposal cannot overwrite newer edits or mutate a removed view. Admitted event cancellation happens during native dispatch, and a failing callback produces one sanitized located report while later events continue. |
| Browser/runtime closure | An empty Can browser app and this grid use the same supported build path. The pre-bundle reachable graph and final JS/map/table are structurally audited for server-only imports, actual Node/Bun operations and secret canaries, while harmless quoted host words remain legal. Exercise exact JSON/int64 rejection parity, standard failures, equality, interleaved async callbacks after `await`, owner leases/drainage, disposal and admitted coordination in required named Chromium and WebKit versions. CSP permits the paired same-origin asset without inline script or `unsafe-eval`; source locations resolve from the published map/table. |

The existing [server matrix](../../tests/integration/gate3_matrix_test.go#L392),
[form browser test](../../tests/integration/browser/invoice.mjs) and
[27-check grid matrix](../../tests/integration/gate5_frontend_test.go#L608)
provide reusable scenarios and observers. Their present conclusions cannot
be copied unchanged: current direct captured GET is 404, the grid's actions
are duplicated with placeholders, the form 503 does not swap, and Gate 5
depends on a test origin and semantic shims.

## Contract-edit tests

Each edit is made **once in the shared contract** in a disposable copy. Update
that copy's lock, rebuild **both** projects and execute the affected flow
through the served application. A pure metadata edit must move both generated
sides automatically. Where the edit invalidates consumer source, first leave
that source stale and require a located diagnostic before publication; then
update only the necessary Can consumer logic and require both builds and live
behavior to pass. A source edit that changes only a browser bundle while the
server keeps serving its old contract fails. The edit fixtures do not change
the base application's selected routes or statuses.

| Single contract edit | Stale-consumer result | Repaired, live result |
| --- | --- | --- |
| Change `load_invoice_grid` literal to `/api/v2/tenants/:tenant_id/invoices/:invoice_id`. | Checked action consumers update automatically. A manually copied route in client construction or server routing fails the no-mirror source/asset audit and the live old-path/new-path checks; an arbitrary string literal is not claimed to be statically linkable. | Browser Fetch uses the new path and the server's actual route receives it; authenticated GET is 200 there and old path is 404, with no server router edit or proxy rewrite. |
| Rename `invoice_key.invoice_id` and its capture to `document_id`. | Old named builder/handler field or inconsistent capture declaration fails at a source location. | Typed builder, mount, handler and browser request agree; valid new capture reaches the handler, malformed canonical integer remains 400 before it. |
| Rename `grid_edit_input.revision` to `expected_revision`. | Any stale named field use fails checking. A positional constructor may remain valid only if both generated codecs move with the shared field; a grid-only assertion is not enough. | Browser request bytes contain the new field exactly once, server decodes that same field and the protected write uses it. An old-field payload is rejected without mutation. |
| Rename `grid_invalid` to `grid_rejected` in `grid_edit_outcome` and its case table. | A stale server return, action case, client match or renderer fails checking; an old tag cannot be silently interpreted as the new leaf. | A real invalid save returns 422 with the new case tag, the browser renders its draft/errors, and both targets' codec tables share the new identity. |
| Change a case status in an isolated action fixture, and separately change `grid_edit_input` body limit. | Checked server/client metadata updates automatically; a manually mirrored status table fails the no-mirror audit. A deliberately nonconforming live response reports the finite unexpected-status failure. | Actual response status and checked client case table change together; over-limit raw HTTP is 413 before handler entry, and the browser applies the same declared request budget. The base invoice retains its selected 200/422/409/403/503 statuses and 8192-byte JSON limit. |
| Change JSON POST body mode to HTML in an isolated action fixture. | Existing `action::request`/`browser::fetch`, JSON handler or renderer binding fails the action-kind/signature check. | A fully updated HTML-only fixture may pass separately; the base grid remains JSON and never treats HTML as JSON. |

Additional negative fixtures reject a shared `handles` clause, a mirrored
local action, missing or extra action case, computed/string action symbol,
wrong capture field type, wrong callback request position, generic or mistyped mount handler, missing
pool/origin capture, and any direct or transitive server capability reachable
from browser `main`. A declaration import alone must register no server
route. The server build must reject a mismatched browser manifest; it cannot
ship an old client alongside a new server after a contract edit.

## Pass record

The qualification report names every executed case and records its artifact
IDs, versions, direct URLs, statuses, browser network trace, DOM/focus/
diagnostic observations, database effects and mutation-test diagnostics.
Contract-edit legs record handler-entry counts at old/new paths, rejected
captures/bodies and the successful edited flow, so a 400/404/413 cannot hide
entry into a protected handler.
Chromium and WebKit are required for the selected browser claim; an unavailable
required engine or skipped live/database leg is **not a pass**. A local test
may skip for missing prerequisites, but the implementation plan cannot mark
this gate complete from that skip. The report distinguishes assertion fixtures,
native mechanism probes, staged toolchain tests and the final installed
distribution run on the selected Linux target. A failure returns the relevant
mechanism to implementation or design rather than weakening this gate.

This invoice gate proves the application composition milestone. The separate
[selected compile/rejection matrix](post-upgrade-selected-behavior-2026-09-24.md#required-compile-rejection-and-execution-matrix)
also gates the two conformance fixes and public generic composition; passing
the invoice alone cannot mark those independent findings complete.
