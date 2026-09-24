# DI-11: first Can browser target and grid state contract

**Status after the upgrade (24 September 2026):** Partially implemented accepted contract. The Can browser target and grid now exist. General runtime delivery and some selected event behavior remain unfinished; the tested grid uses a separate test bundler with semantic shims. The original design below is preserved. See the [current reconciliation](../post-upgrade-reconciliation-2026-09-24.md).

24 September 2026. Planned P8 design for a **Can-authored** main-thread
frontend, not current compiler behavior. The recommendation program requires
an editable invoice grid with keyboard interaction, local calculations,
optimistic save/rollback, slow/offline behavior and disposal. Current Can runs
on Bun and only distributes pinned HTMX as browser code. Three earlier
[Jev consultations](jev-product-contracts/README.md) favored a bounded
main-thread target, and three fresh [architecture consultations](jev-grid-contract/findings.md)
favored explicit DOM/event/Fetch operations over a new reducer/virtual-DOM
engine. Jev advice is not browser qualification or agent-task evidence.

## Build and capability boundary

`canlc build --target browser <project>` builds a distinct browser project
whose root exports exactly one non-generic `void main` with no arguments and
`emits []`. Its generated entry runs after DOM readiness as a same-origin
module asset; uncaught startup faults are sanitized and reported through the
browser adapter. The existing Bun/server build retains its own
`void main(str[] args)` contract. Server and browser projects can depend on the
same locked pure contract packages; each target uses the same canonical
package-instance identity. Target choice is a build parameter, not a runtime
branch inside one bundle. The generated browser asset is the only new admitted
application script class: it is compiler-produced, content-addressed,
same-origin and published with the verified server build. Arbitrary authored
JS/TS imports, inline scripts, raw HTML/SVG and third-party browser bundles
remain outside this target. The verified asset and CSP must allow only the
selected script path and necessary compiler-owned assets.

The compiler checks the entire reachable declaration/specialization/callback
closure of the browser entry. It admits pure Can values, selected shared
codecs, safe HTML data where explicitly supported, `browser` catalogue
operations and the client side of `action` contracts. It rejects server HTTP
mounts, SQL pools, process/file/environment/secret access, server crypto keys,
native server questions and any imported function that reaches them, even
through a generic specialization, callable reference or exported wrapper.
Unreachable server code is not bundled; a package shared across targets must
still pass ordinary source checks. The emitted module graph and sourcemaps are
audited for server-only imports and secret bytes. Browser `Worker` is a
different unselected profile; this one has `document` access only on the
main thread.

Shared wire codecs must produce the same bytes and rejection categories on
Bun and named supported browsers for primitives, int64 `bigint`, finite
floats, ordinary records, closed variants, `option<T>`, unknown/duplicate
fields and nested values. Native browser `Number` cannot silently stand in
for Can `int`; reuse the exact shared tokenizer/codec contract and adapt only
unavailable native JSON APIs. Owner records are never directly derived wire
types: both targets decode transparent wire records and call owner factories.
Server authorization and validation run again on every client submission.
Passing a browser type check cannot grant tenant access.

## Explicit native browser catalogue

The first browser catalogue is a small checked boundary, not a general DOM
escape. It provides opaque `app`, `view`, `node` and `state<T>` handles and
operations in these groups:

| Group | Planned Can-visible contract | Native lowering and limits |
| --- | --- | --- |
| Mount and view | Resolve a required root ID and create one view owner; missing/detached root is a named failure. Dispose releases every view-owned listener and timer exactly once. | `document.getElementById`, `AbortController`/listener removals and native timers. A node handle is valid only while its view is live. |
| DOM construction | Create from the safe admitted HTML tag/attribute inventory; set text, form value, disabled/ARIA state, children and focus through typed operations. Dynamic text always uses a text node or `textContent`, never string interpolation into markup. | `document.createElement`, `createTextNode`, `replaceChildren`, property/attribute setters and `focus`. Reject unsafe URL schemes, event-handler attributes and invalid structure at the same stated static/runtime boundaries as safe HTML. No unrestricted `innerHTML` or arbitrary JS handle. |
| Events | Register named Can callable handlers for admitted keyboard, input, change, click, focus and submit events, with typed snapshots of only approved event fields. The view owns registrations; event callbacks cannot retain raw native Event objects. | `addEventListener` with view-scoped removal. `preventDefault` is explicit and only for admitted cancelable events. Handler failures are observable and do not silently mutate state. |
| State | Create opaque `state<T>` from an immutable Can value. `snapshot` returns value plus monotonically increasing version; `compare_replace` publishes a new immutable value only if the version still matches. | A small native closure/cell stores the value. This is a resource adapter, not general mutable Can aliases or an implicit reactive loop. Async callbacks must re-read and conditionally update after suspension. |
| Network | Build requests only from checked action URLs/methods or explicitly admitted safe URLs; use native Fetch with declared codec/body limits and finite transport/status outcomes. | `fetch`, `AbortController`, exact shared codecs and response validation. Abort is local cancellation only, never evidence of server rollback. |

The selected catalogue spelling is `browser::application() -> app`,
`browser::open(app, html::id) -> view`, `browser::close(view)`,
`browser::element(view, html::tag) -> node`, `browser::text(view, str) -> node`,
`browser::replace_children(view, node, node[])`,
`browser::set_text(view, node, str)`, `browser::set_value(view, node, str)`,
`browser::set_attribute(view, node, html::attribute)`,
`browser::focus(view, node)`, `browser::on(view, node, browser::event_kind,
callable handler)`, `browser::state<T>(app, T) -> state<T>`,
`browser::snapshot<T>(state<T>) -> versioned<T>`,
`browser::compare_replace<T>(state<T>, versioned<T>, T) -> bool`, and
`browser::fetch(app, action::request) -> action::response`. `browser::on`
requires an infallible named handler taking a bounded immutable
`browser::event` snapshot; the handler itself handles its own domain errors.
`browser::versioned<T>` carries an opaque compare token and immutable value,
not a forgeable public integer counter. `action::request` and
`action::response` retain the checked method, wire codec, finite statuses and
transport/unknown-status errors of their declaration. Construction and event
operations reject `browser::disposed` after their view closes; `open` reports
`browser::missing_root`; dynamic DOM admission reports
`browser::invalid_dom`; `fetch` reports named transport/abort/codec failures
without converting them to a domain success. Exact error payload fields and
static catalogue signature encoding follow these contracts; they are not a
license to add another public operation or alter ownership during coding.
The compiler statically validates literal tags/attributes and action/field
references; dynamic values receive runtime checks at the native boundary.

## Grid state and save ownership

Use ordinary named Can functions over immutable records to compute totals and
apply keyboard/input events. The view's `state<grid_state>` stores the current
draft, acknowledged server revision, display order, focused cell and
visible save/error state. A synchronous event takes a snapshot, computes a
new value and conditionally publishes it; a stale version retries from the
latest state rather than overwriting another event. Rendering is explicit
DOM work by the application using the catalogue above. There is no compiler
inferred dependency graph, virtual DOM, reducer syntax or hidden effect queue.

The first offline promise is [in-view editing with explicit retry](jev-generic-offline/findings.md):
while the grid view lives, disconnection does not erase its draft or stop
local totals; unsaved/failed state is visible. Reconnect permits a deliberate
save or reconciliation. No reload/navigation persistence or automatic offline
replay is promised. Native `navigator.onLine` may be a hint but cannot decide
save success; actual Fetch result and server revision do. Drafts are not
silently discarded after a failed request.

The grid's save endpoint is the separate typed JSON POST action in the
[integrated action contract](integrated-action-contract.md). Its row records
and finite JSON result are codec-admissible; the HTML form's `form::rows` and
rendered fragments are not browser state payloads. The HTML and JSON save
actions share one protected write operation. A third checked JSON GET action
returns the authorized initial/current grid snapshot. An identified uncertain JSON save can replay only
the identical payload and mutation ID against the durable server replay ledger,
or reconcile by an authorized read.

An attempted save receives a stable application-owned mutation ID, submitted
draft snapshot and expected revision. The app owner, not the view, tracks the
in-flight request and its settlement/reconciliation while the page's Can
runtime remains alive. View disposal removes event listeners/timers and
prevents writes to detached DOM; it does not infer that the server rolled back
or forget an identified save. A late response updates the acknowledged
revision only after checking mutation ID and server outcome, and never
overwrites edits made after the request snapshot. A 409 retains the newer
draft and shows conflict; 422 retains raw rejected text and field errors; 403
reveals no protected data; 503/transport loss marks the operation uncertain
until an authorized reconciliation read or idempotent server lookup resolves
it. No blind retry follows an uncertain commit. Rollback means correcting the
optimistic **display** while preserving the user's current unsaved draft, not
restoring an obsolete object over newer edits.

The grid uses semantic table/form controls initially. Keyboard navigation
and edit/commit modes are explicit handlers with focus restoration and live
error announcement; ARIA attributes alone are insufficient. Add `dialog`,
canvas or SVG only if a concrete grid behavior requires them and the safe
catalogue contract is specified. A server-driven HTMX rendering path remains
a separate narrow milestone; it is not evidence that browser Can passed.

## Acceptance and native evidence

Build the same shared contract package for server and browser, change an
action route, wire field and variant leaf, and verify both targets diagnose
or regenerate linked uses without silent drift. Reject direct and transitive
server-only capabilities in browser code, including through generic and
callable paths; scan emitted assets for server modules and secrets. Compare
JSON byte/rejection behavior for int64 endpoints, finite floats, duplicate/
unknown fields, option and variant values in Bun and each supported browser.

In a named browser/version matrix, actually edit the invoice grid with
keyboard only, reorder keyed rows, inspect exact totals, save successfully,
edit again while a save is pending, inject 409/422/403/503 and transport loss,
reconnect, remove and remount the view, and deliver responses out of order.
Observe DOM text, focus, announcements, network bytes, server rows, leaked
listeners/timers and ownership of late responses. Crash/reload tests must
confirm the **absence** of a durability claim rather than count a lost
in-memory draft as a defect under this first contract. Fault-test storage/
queue designs separately before expanding that promise. The first Can browser
target is qualified only when the actual generated asset performs these
operations; no foreign-language bridge or Bun-only assertion substitutes for
the browser observation.
