# Selected behavior for the next Can upgrade

24 September 2026 · design baseline `fbd2a56`

This is the implementation contract for the 11 **Fix** findings in the
[disposition ledger](post-upgrade-dispositions-2026-09-24.md). It specifies the
observable source, build and execution behavior to be delivered through the
[ordered implementation plan](post-upgrade-implementation-plan-2026-09-24.md)
and [task lanes](post-upgrade-implementation-tasks-2026-09-24.md).
**None of the new action grammar, mount
bindings, browser bundle path or public generic call rule is implemented by
this document.** Existing syntax in examples is identified where it already
parses; other snippets are proposed Can and must become positive compiler
fixtures. No old action spelling, generated TypeScript layout or grid boot
harness is a compatibility target.

The design keeps the earlier [shared action](preparation/integrated-action-contract.md),
[browser target](preparation/browser-target-design.md) and
[DI-04 generic](preparation/accepted-technical-decisions.md#di-04--explicit-operation-inputs-for-exported-generics)
contracts. The invoice server and Can-authored grid are the acceptance
application under the [invoice composition gate](post-upgrade-invoice-acceptance-2026-09-24.md).
The [evidence index](evidence/2026-09-24/selected-behavior/README.md),
[source proposals](evidence/2026-09-24/selected-behavior/actions-proposal.md),
[browser investigation](evidence/2026-09-24/selected-behavior/browser-proposal.md),
[generic proof](evidence/2026-09-24/selected-behavior/generics-proposal.md),
[pinned HTMX probe](evidence/2026-09-24/selected-behavior/htmx-proposal.md) and
[three fresh Jev consultations](evidence/2026-09-24/selected-behavior/jev/findings.md)
record the evidence and alternatives. Jev advice does not override accepted
requirements or executable checks.

## One shared action contract, three server bindings

The shared locked package contains transparent wire records, finite result
variants and handler-free `action` declarations. Its imports cannot reach SQL,
server routes, session secrets or renderer code. This **proposed Can** package
uses the selected JSON result fields and HTML keyed-row form shape from the
[integrated contract](preparation/integrated-action-contract.md#source-declarations-and-boundaries)
and [keyed-row design](preparation/keyed-row-design.md#wire-shape-and-checked-names).

```can
package invoice_contract
    provides [invoice_key, grid_line_wire, grid_snapshot, grid_loaded, grid_load_forbidden, grid_load_unavailable, grid_load_outcome, grid_problem, grid_edit_input, grid_saved, grid_invalid, grid_conflict, grid_forbidden, grid_unavailable, grid_edit_outcome, line_wire, invoice_form, field_error, saved, invalid, conflict, forbidden, unavailable, edit_outcome, load_invoice_grid, save_invoice_grid, save_invoice_html]
    uses [form, option]

record invoice_key
    int tenant_id
    int invoice_id
record grid_line_wire
    str key
    str id
    str quantity
    str price
record grid_snapshot
    str revision
    grid_line_wire[] lines
    int total_minor_units
record grid_loaded
    grid_snapshot current
record grid_load_forbidden
    str message
record grid_load_unavailable
    str message
variant grid_load_outcome
    grid_loaded
    grid_load_forbidden
    grid_load_unavailable

record grid_problem
    option::value<str> row_key
    str field
    str message
record grid_edit_input
    str operation_id
    str revision
    grid_line_wire[] lines
record grid_saved
    str operation_id
    grid_snapshot acknowledged
record grid_invalid
    str operation_id
    grid_edit_input draft
    grid_problem[] errors
record grid_conflict
    str operation_id
    grid_edit_input draft
    str message
record grid_forbidden
    str operation_id
    str message
record grid_unavailable
    str operation_id
    str message
variant grid_edit_outcome
    grid_saved
    grid_invalid
    grid_conflict
    grid_forbidden
    grid_unavailable

record line_wire
    str id
    str quantity
    str price
record invoice_form
    str seats
    option::value<str> details
    str revision
    form::rows<line_wire> lines
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
```

The same package also declares `save_invoice_grid` as JSON POST with
`grid_edit_input limit 8192` and five `grid_saved/grid_invalid/grid_conflict/
grid_forbidden/grid_unavailable` cases at 200/422/409/403/503. Its
`grid_edit_input` contains `operation_id`, `revision` and keyed
`grid_line_wire[] lines`, never a client-supplied actor or session permission.
`save_invoice_html` is HTML POST with `invoice_form limit 2048 rows_limit 64`
and five `saved/invalid/conflict/forbidden/unavailable` cases at the same
statuses, each declaring `swap inner`. These two declarations use exactly the
accepted [source form](preparation/integrated-action-contract.md#source-declarations-and-boundaries),
including typed record captures and distinct body modes. The `:name` segment
is one strict decoded segment, matched by a required `str` or canonical int64
`int` field of the capture record. This invoice uses the earlier selected
two-`int` key: `(1, 7)` builds `/tenants/1/invoices/7`. The builder checks
signed int64 range, and inbound captures reject `+1`, `01`, `-0`, `%31` and
out-of-range text with 400 before entering a handler. Domain positivity is
checked separately. Method, literal path, capture names/types,
input/body limit, result leaves, status and HTML swap policy are compile-time
action metadata keyed by canonical locked package identity plus declaration
name. An action is not an ordinary runtime string or a route merely by being
imported.

The server and browser import that **same** package instance. These are
proposed call sites; ordinary `near` callable capture and `match chain` are
already Can syntax. The complete handler bodies remain application code,
not generated authorization. The shown `model::load_authorized` and
`model::save_authorized` are named invoice helpers whose required semantics
are defined below, not catalogue magic.
The browser may type-check the shared `form::rows` declaration metadata but
only its reachable JSON client projection is emitted; a form parser, HTML
renderer or server handler cannot enter the browser closure merely because
the package exports their types and action symbol.

```can
package invoice_server
    provides [routes]
    uses [billing::invoice_contract as contract, action, http, sql, option, model]

fn contract::grid_load_outcome load_grid
    emits []
    given
        near sql::pool pool
        http::request req
        contract::invoice_key key
    asserts
        no_session: contract::invoice_key(1, 7) => ok
    match call http::request_headers(req)
        when
            no_session: req => ok []
        ok http::header[] headers => match call model::session_of(headers)
            ok option::value<str> session => match session
                option::none => ok contract::grid_load_forbidden("denied")
                option::some => match call model::load_authorized(pool, session.value, key)
                    ok contract::grid_load_outcome found => ok found

fn contract::grid_edit_outcome save_grid
    emits []
    given
        near sql::pool pool
        near str public_origin
        http::request req
        contract::invoice_key key
        contract::grid_edit_input body
    asserts
        no_session: contract::invoice_key(1, 7), contract::grid_edit_input("op-1", "r1", []) => ok contract::grid_forbidden("op-1", "denied")
    match call http::request_headers(req)
        when
            no_session: req => ok []
        ok http::header[] headers => match call model::session_of(headers)
            ok option::value<str> session => match session
                option::none => ok contract::grid_forbidden(body.operation_id, "denied")
                option::some => match call model::exact_origin(headers, public_origin)
                    ok bool admitted => match admitted
                        false => ok contract::grid_forbidden(body.operation_id, "denied")
                        true => match call model::save_authorized(pool, session.value, key, body)
                            ok contract::grid_edit_outcome result => ok result

fn http::router routes
    emits [http::invalid_route, http::duplicate_route, http::ambiguous_route]
    given
        near sql::pool pool
        near str public_origin
    asserts
        sample: => ok
    match chain
        call action::mount(contract::load_invoice_grid, callable load_grid) as http::route load
        call action::mount(contract::save_invoice_grid, callable save_grid) as http::route save
        call action::mount(contract::save_invoice_html, callable save_html, callable render_html, callable render_bad_form) as http::route form
        call http::make_router([load, save, form]) as http::router built
        http::invalid_route
        http::duplicate_route
        http::ambiguous_route
        ok => ok built
```

`save_grid` takes `(http::request, invoice_key, grid_edit_input)` and returns
the exact `grid_edit_outcome`; `save_html` takes the form wire instead. The
normal HTML renderer takes `edit_outcome`; the structural-rejection renderer
takes `form::rejected<invoice_form>`. All these bound callables are
non-generic, non-variadic and `emits []`; the application maps fallible
authentication, validation and storage into declared leaves. The complete
positive fixture must define `save_html`, both renderers and the named
`model::session_of`, `model::exact_origin`, `model::load_authorized` and
`model::save_authorized` helpers with the exact signatures, finite error
mapping and protected-operation behavior stated here; this excerpt alone is
only parseable call-site syntax. `near sql::pool
pool` captures the one pool opened at startup when
`routes(pool, public_origin)` is called. Startup validates one canonical
public origin and passes it as the second explicit input; both save callables
capture that `near str public_origin` from `routes`.
The captured service is explicit and statically typed; there is no ambient
actor, service registry or per-request database opening. The checked mount
intrinsic consumes an action symbol plus the exact callback signature and
returns an `http::route`; it does not add general function overloading.

Every mounted handler receives a request capability. It derives the session
from server-controlled request headers on **each** GET or POST; a capture,
wire field or browser token cannot authorize. The protected read transaction
checks live session, tenant membership, invoice ownership and disclosure
before returning a snapshot. The protected write checks the same facts and
expected revision inside the mutation transaction, so a concurrent revoke
cannot pass on an earlier stale check. Foreign and nonexistent invoices give
the same nondisclosing 403 after authentication. POSTs additionally enforce
this invoice application's **exact-Origin CSRF policy** before mutation: one
canonical HTTPS public origin is configured at startup (explicit loopback
HTTP is allowed for local tests), captured as `near str public_origin` by
both save handlers, and compared with exactly one browser-generated `Origin`
header after bounded decoding but before replay lookup or SQL writes.
Missing, `null`, duplicate, comma-joined, malformed, sibling-domain,
wrong-scheme or wrong-port Origin denies with the typed 403 leaf; `Referer`,
`Sec-Fetch-Site` and a valid session cannot override it. GET needs no Origin
but still authenticates. The browser never receives a session token in wire
data or a hidden form control; the server reads its HttpOnly session cookie.
This narrow policy relies on the supported same-origin browser POST behavior
and rejects deployment configurations that suppress Origin; the
[Fetch Standard](https://fetch.spec.whatwg.org/#origin-header) and
[OWASP guidance](https://cheatsheetseries.owasp.org/cheatsheets/Cross-Site_Request_Forgery_Prevention_Cheat_Sheet.html#using-standard-headers-to-verify-origin)
are the external inputs, and the [action proposal](evidence/2026-09-24/selected-behavior/actions-proposal.md#concrete-first-scope-csrf-policy-follow-up-proposal)
records the exact admission table. A pool stays live through
admitted callbacks and owner drainage, then closes once after shutdown;
startup/mount failure closes it too. **Request revocation is new work:** the
current request WeakMaps retain snapshots after `abandonRequest`, as the
[buffered/lazy token probe](evidence/2026-09-24/selected-behavior/experiments/request-lifetime/findings.md)
confirms. After the
handler settles and materializes its response or upgrade decision, dispatch
abandons any unread live body; the per-request owner scope then drains
admitted child work. The outer request boundary revokes the token in a
`finally` path after drainage and before returning the native response.
Header, body, upgrade and snapshot operations on a retained
token then fail with the named resource-state failure. A buffered request is
revoked too; a failed handler and a rejected route cannot leak a usable token.

The JSON save ledger keys `operation_id` by authenticated actor and captured
invoice, and commits canonical-payload digest and result in the mutation
transaction. **Every replay rechecks the current session, membership,
invoice ownership and disclosure rule** before reading or returning a ledger
result, in the same serialized database transaction as lookup. A revoked
actor gets the nondisclosing 403 leaf even for an identical ID/payload;
the prior saved snapshot is never disclosed. An authorized identical
ID/payload replay returns the prior result with no second effect; changed
payload under that ID conflicts. A 503, abort, lost
response or disposed view does not prove rollback. The browser retains the
exact attempt for a deliberate identical replay or an authorized reread; a
new ID is never an automatic retry. The ledger's bounded retention window
is a required positive startup parameter `W`; the invoice acceptance fixture
uses **7 days** and tests just before, at and after expiry. A committed result
is retained at least `W` after server commit time, with no sliding extension;
lookup and cleanup serialize with mutations. After expiry, exact-result
replay is no longer promised and the UI requires deliberate reconciliation.
Every successful invoice mutation advances a never-reused revision, so an
identical old request whose ledger row has expired cannot cause a second
effect under its original expected revision. Revision reset, wrap and invoice
ID reuse are forbidden under that argument; a future lifecycle needing them
requires an incarnation or durable consumed-ID record first. HTML has no
automatic replay. The [retention analysis](evidence/2026-09-24/selected-behavior/actions-proposal.md#replay-retention-bounded-rule-and-missing-duration)
states the limits of this bounded guarantee.

The adapter distinguishes typed application leaves from pre-handler errors:
malformed capture/JSON is 400, unmatched route 404, wrong method 405, body
limit 413, wrong media 415, structural form rejection 422, and unexpected
fault 500. Real application 200/422/409/403/503 statuses retain their
declared result bodies. It does not invent `grid_invalid` after malformed
JSON or rewrite 503 to 200. A post-commit rendering fault yields an uncertain
500, not rollback. Exact `action::url`, `action::request` and `action::post`
references share the declaration's route, codecs, body mode and case table;
the GET request has no body. `browser::fetch` uses native same-origin Fetch,
checks finite status/leaf/operation-ID agreement, and reports transport,
abort, codec and unexpected-status failures separately.

## Browser build and event execution

The accepted browser entry is one exported non-generic `void main` with no
arguments and `emits []`; Bun/server `main(str[] args)` remains separate.
Current grid `main(args)` works only because a test-authored boot script reads
`location.search`. The supported browser target removes that script and
admits one narrow **proposed** `browser::query_parameter` operation. It takes
a compile-time literal ASCII key matching `[a-z][a-z0-9_]*` (at most 64
bytes). The adapter bounds the entire raw `location.search` to 8192 UTF-8
bytes and each decoded value to 256 UTF-8 bytes. Before constructing native
`URLSearchParams`, it scans **every** raw pair: malformed `%` escapes,
invalid percent-encoded UTF-8, or an ill-formed Unicode string yields
`browser::invalid_query`. The scan uses native `decodeURIComponent` after
replacing `+` with space; `URLSearchParams` then supplies decoded pairs.
`%69nvoice` is the `invoice` key, `in+voice` is not, and either spelling of
the selected key counts toward duplicate detection. Zero occurrences return
immutable `option::none`, one returns `option::some`, and two or more fail.
This strict precheck is necessary because `URLSearchParams` alone accepts
`%ZZ` and replaces invalid UTF-8, as the [native query check](evidence/2026-09-24/selected-behavior/query-native-probe.md)
shows. The app handles absent/invalid input
visibly. It reads `tenant` and `invoice` as text, uses `text::to_int` and a
`text::from_int` round trip to require canonical decimal spelling, constructs
`invoice_key`, then uses the checked `action::url` path builder to enforce
int64 range; the server reauthorizes on every action. A query value is
never a permission. This is a read-only bootstrap operation, not a browser
history or navigation API. This source form must compile after the upgrade:

```can
fn void main
    emits []
    asserts
        no_invoice: => ok
    match call browser::query_parameter("invoice")
        browser::invalid_query => match call show_boot_notice("Invalid invoice link")
            ok => ok
        ok option::value<str> selected => match selected
            option::none => match call show_boot_notice("Choose an invoice")
                ok => ok
            option::some => match call boot(selected.value)
                ok => ok
```

The full invoice grid reads and validates both numeric keys; this single-key
entry isolates the bootstrap rule. `show_boot_notice` uses admitted
text-node, ARIA live-region and view operations. It must be a complete checked
Can helper in its fixture; no raw DOM or inline script is implied. The grid
build no longer supplies `str[]` or a test-written entry.

`canlc build --target browser <project>` must produce a **served-ready**
content-addressed JavaScript module, source map, immutable diagnostic table
and manifest from the compiler's browser TypeScript. Its maintained build
stage uses native `Bun.build({ target: "browser" })` against a browser-specific
runtime profile. The profile seals concrete type/error identities at build
time, accepts only privately branded Can values, threads owner execution
context explicitly across generated async calls, and embeds checked source
locations; it does not alias Node modules with grid-only shims. The
[interleaved-owner experiment](evidence/2026-09-24/selected-behavior/experiments/browser-owner/findings.md)
shows why the current synchronous shim is insufficient after `await`; it
does not prove the proposed profile. Native
`Promise` coordination, DOM methods, Fetch and `AbortController` remain the
execution primitives. The empty app and invoice grid use this same command.
The compiler-owned entry configures the immutable diagnostic table and Can
state, then invokes checked zero-argument `main` exactly once after DOM
readiness (immediately if already ready). A startup fault goes through the
same sanitized reporter as later callback faults; no test-authored boot
module invokes `main`.

The server build takes an **explicit path to the verified browser manifest**
(proposed CLI spelling
`canlc build --target bun --browser-manifest <file> <server-project>`).
It verifies the browser build ID, locked shared-package
instances, declared asset graph and hashes, then records the exact browser
build ID and asset digests in its own verified build report. The server asset
table publishes only that matched same-origin digest URL (for example
`/__can/assets/<sha256>.js`) and the matching map/table routes. Publication
atomically switches the report and entire asset table after all bytes pass
verification; a failed rebuild leaves the previous verified set intact, and
old digest URLs remain available for at least seven days after replacement
to cover cached pages during that declared asset lifetime. A page receives
exactly one paired script URL from the selected server report. An
independently rebuilt or mismatched browser manifest cannot be served under
the current server report. CSP admits the selected module
and needed compiler-owned assets with `script-src 'self'`, `connect-src 'self'`,
no inline script and no `unsafe-eval`. No author JS import or test harness
alias enters the production graph.

The compiler checks the reachable Can call/callback/specialization closure.
Before bundling, a structural audit resolves **every** generated, dependency
and runtime module and every static/dynamic import edge in the reachable
graph; this is where a server-only runtime dependency is rejected. After
bundling, a second audit parses final JavaScript and verifies every published
JS/map/diagnostic-table byte and reference against the manifest. It rejects
server SQL,
environment, process, filesystem, key/secret and native server capabilities;
remote/dynamic/unaccounted module edges; actual `Bun`/`process`/`require`
operations; and known secret canaries. It reports the forbidden module/span
and originating Can call chain when available. It must not reject ordinary
string data containing `"Bun."`, `"node: introduction"` or `"require("`.
That replaces the current raw-substring B01 audit and extends checking to
the runtime closure. Tampered assets, missing maps or mismatched hashes fail
before publication. Named Chromium and WebKit versions qualify startup,
standard failures, diagnostics, equality, int64/JSON parity, concurrent
callbacks, view disposal and admitted coordination; Firefox requires its own
future named-version pass.

For U05, cancellation is a **registration policy**, because ordinary Can
callbacks settle asynchronously and native default behavior must be stopped
inside the original event dispatch. These proposed calls are allowed only
for their finite admitted kinds:

```can
call browser::on_cancel_key(view, input, "keydown", "Enter", callable on_field_key)
call browser::on_cancel_event(view, form_node, "submit", callable on_submit)
```

`on_cancel_key` admits only `keydown` or `keyup` and a nonempty exact key;
`on_cancel_event` initially admits only `submit`. The native
`addEventListener` closure checks view liveness, then dispatches **one**
immutable snapshot to its Can handler for every event of its registered
kind. For a matching key (or admitted `submit`) that is cancelable, it calls
`event.preventDefault()` synchronously **before** that dispatch. A different
key or a noncancelable event still reaches the handler once, without
cancellation. Ordinary `browser::on_event` never cancels. Registering both
ordinary and cancel-policy callbacks on one node/kind is two explicit
subscriptions, never a hidden second invocation of one callback. Neither a raw
Event
nor a mutable cancellation token enters Can data. A disposed listener cannot
cancel; `defaultPrevented` must be observable immediately after native
dispatch, without awaiting Can. A later handler failure does not undo the
cancellation or report success. The [native event timing experiment](evidence/2026-09-24/selected-behavior/experiments/event-cancel/findings.md)
distinguishes synchronous cancellation from a later Promise continuation in
installed Chromium.

For S02, failed admitted event/timer callbacks and startup faults produce
one sanitized, source-located diagnostic occurrence through a compiler-owned
browser reporter. Its public data is category, phase and Can file/line/column;
native cause, raw input, stack and secret bytes are omitted. The default
sink is `console.error` of the frozen record; a later valid event still runs,
and disposal still removes listeners/timers. Successful callbacks produce
no fault report. S03 keeps notice, pending-save state and desired focus in
immutable grid state: render constructs all rows **without** focusing,
attaches the complete new tree, then resolves and focuses its stable row key
and verifies `document.activeElement` in browser acceptance tests. The
current `build_rows` calls `maybe_focus` before the grid is attached, so
moving only the root-level focus call is insufficient. Blocked-save feedback
survives rerender in a live region, and a pending indicator exists until that
attempt settles. Version/attempt checks
prevent late replies or disposal from overwriting newer edits or DOM state.

## Public generic helpers

P03 changes checking, not source syntax. These are existing Can declaration
forms; the `pass`/`wrap`/`nested` calls currently fail because they forward an
opaque public type argument. All four functions must compile after the
upgrade, including when `helpers` imports `core` through a locked dependency.

```can
package core
    provides [identity]
    uses []
fn item identity<item>
    emits []
    given
        item value
    asserts
        number: 3 => ok 3
    ok value
```

```can
package helpers
    provides [box, pass, wrap, nested]
    uses [core]
record box<item>
    item value
fn item pass<item>
    emits []
    given
        item value
    asserts
        number: 3 => ok 3
    ok call core::identity<item>(value)
fn box<item> wrap<item>
    emits []
    given
        item value
    asserts
        number: 3 => ok box(3)
    ok box(call pass<item>(value))
fn box<item> nested<item>
    emits []
    given
        item value
    asserts
        number: 3 => ok box(3)
    ok call core::identity<box<item>>(box(value))
```

For a public generic call with an opaque argument, resolve a **public,
symbolically validated** callee by canonical declaration identity. Substitute
the caller's symbolic type expressions into the callee's checked signature,
then check normal inputs, result and exact finite `emits`. The callee's body
is checked once at its own declaration; an illegal `value + value` edit fails
there even if an `int` assertion passes. A private generic has only
concrete-template evidence, so it cannot be a symbolic callee. A known
`box<item>` may carry `item` without examining it; arithmetic, equality,
field access on bare `item`, concrete codecs and other
representation-dependent operations still need an explicit named callable
or dictionary input where applicable. This does not add traits or error-set
parameters.

The `nested` body specifically proves an acyclic `G<box<T>>` call: a reached
`nested<int>` must instantiate and directly call
`core::identity<helpers::box<int>>`. Symbolic call edges never allocate a concrete specialization, invoke
`SpecializationKey`, or enter emitted functions. Validate public-generic
dependencies in order; for a mutual strongly connected component, check
all bodies and internal signatures under provisional contracts and commit
**none** until the entire component passes. On every internal cyclic edge,
each type argument must be a bare caller formal or a closed type containing
no opaque formal anywhere in its sealed graph. Permutation, duplication or
dropping of bare formals stays within a finite instance-state set. A
constructor such as `T[]` around an opaque formal on a cyclic edge fails
with the call chain; an acyclic call to `G<box<T>>` is allowed after G's
proof is complete. This bounds declaration proof and concrete instance
discovery, not runtime recursion depth or termination.

At a reached concrete call, normal specialization creates or reuses a
canonical `G<int>` or `G<box<int>>` instance, checks its body, and emits an
ordinary direct TypeScript function call with current completion handling.
Only concrete reached functions are emitted. There is no opaque runtime type
object, implicit dictionary, dynamic dispatch or fake symbolic function.
The [focused recursion probe](evidence/2026-09-24/selected-behavior/experiments/generic-recursion/findings.md)
also found an independent current emission gap: even an exported
`identity<T>` that passes `CheckProgram` leaves a symbolic `types.Parameter`
in `Program.Model`, and `NativeTypeDeclarations` rejects it as `unknown
emitted type`. P03 must remove declaration-only symbolic types from the
emitted graph before a successful checked public helper counts as built.
An acyclic-only rule would also reject finite stationary mutual recursion;
the selected SCC rule needs both that positive case and an expanding-cycle
rejection. The probe does not implement or prove the SCC rule.

## Native execution boundary

| Can operation | Generated TypeScript / native operation | Can-specific adapter retained |
| --- | --- | --- |
| `action::mount`, `action::url`, `action::request`, `action::post` | Checked declaration metadata builds `Bun.serve({routes})` entries, native `URL`/`encodeURIComponent` paths, native `Request`/`Response` and browser `fetch`. | Strict one-pass capture validation, bounded codecs, immutable request snapshot, finite case/status mapping and request-token revocation. Importing a declaration alone registers nothing. |
| Protected `model::load_authorized` / `model::save_authorized` | Existing SQL pool and transaction adapter execute database operations; one pool is captured by ordinary generated JS closures. | Auth, exact Origin, tenant/invoice and revision predicates stay explicit application logic. Ledger digest/result commits with the mutation; no generated permission service exists. |
| `browser::query_parameter`, `browser::on_cancel_*` | `location.search`, native `decodeURIComponent` and `URLSearchParams`; `addEventListener`, `preventDefault` and `AbortController`. | Literal key and size admission, immutable event snapshot, view liveness and sanitized callback reporting. |
| `text::matches` | Capped iteration over native `String.prototype.matchAll` using a fresh global `RegExp`. | Existing Can record shape, capture defaults, result cap and named failures. |
| Public generic calls | Checker-only symbolic declaration proof; reached concrete instances become direct JS/TS calls. | Canonical type-identity specialization, exact finite errors and completion propagation; no runtime generic interpreter. |

## Required compile, rejection and execution matrix

These are acceptance fixtures for the future implementation, not tests
claimed to pass today. Rebuild **both** server and browser from the same
locked contract for every edited-declaration case.

| Finding | Must compile or execute | Must reject or preserve a distinct failure |
| --- | --- | --- |
| U01/U03 | One handler-free package imported by both targets; URL, POST and GET derive from the declared route; changing route alone moves actual server registration and browser request together. | `handles` in a shared declaration, mirrored local action, computed/string action symbol, missing/extra case, changed capture name/type, wrong body mode or stale field/leaf use gets a declaration/call-site diagnostic. Old path is 404 after a route edit; no test URL rewrite. |
| U02 | One startup pool and validated public origin captured by mount callables; GET and both saves receive `http::request`; each checks current session and protected tenant/invoice/revision at the operation. Retained request use after owner drainage fails. | Missing/mistyped lexical pool or origin, wrong request position, wrong result/error bound, generic handler, unauthorized/foreign/nonexistent read or write and revoked membership cannot acquire data or commit. Browser closure rejects mounts and SQL. |
| U03 | GET has no body; JSON POST and form POST use distinct checked codecs; exact statuses, authorized same-ID replay and lost-acknowledgment reconciliation work over real HTTP/SQL. | JSON malformed/over limit/media errors remain 400/413/415, not application `invalid`; GET body, old result leaf or HTML/JSON body mismatch fails checking; same ID/new digest conflicts without second effect; revoked replay gets nondisclosing 403 and no old result. |
| U04/B01 | Empty app and grid boot exactly once after DOM readiness from a server-paired final asset under CSP; valid and absent canonical numeric query keys reach the expected view; harmless quoted host words work; pre-bundle closure and final JS/maps/diagnostics contain no server import or secret. | Browser `main(str[] args)`, malformed `%ZZ`/`%FF`, duplicate encoded-key alias, oversized query or invalid numeric spelling reports the finite boot failure; direct/transitive server capability, actual Node/Bun host operation, tampered/unmanifested/remote/dynamic asset edge, missing map or wrong server pairing prevents publication with location evidence. |
| B02 | Empty Unicode regex over `😀x` returns UTF-16 starts `[0,2,3]` under `u`/`v`, `[0,1,2,3]` without Unicode; caps/captures retain behavior. | Invalid flags/pattern and limits retain named failures; no one-code-unit empty-match loop or unbounded materialization. Use capped native `String.prototype.matchAll` iteration. |
| U05/S02 | Matching key/submit default is prevented during native dispatch; mismatched key and noncancelable events still invoke one handler without cancellation; ordinary events are not canceled; failed callback yields one sanitized diagnostic and later callbacks run. | Unadmitted event/key policy rejects, disposed view cannot cancel, async handler cannot decide cancellation too late, failure cannot disappear or expose raw Event/native cause; one registration cannot double-dispatch its handler. |
| S01 | Declared HTML 200/422/409/403/503 return real statuses and `swap inner` visibly replaces exactly the admitted connected feedback node; initial/mid-flight missing target yields a sanitized `action::missing_target` client occurrence, with HTTP status and effect uncertainty preserved where a response exists. | Unrelated/undeclared failure, any OOB/partial task in an admitted response, response-control header, retarget/reswap redirection or missing target cannot change an unrelated node; a post-commit renderer fault remains uncertain 500. |
| S03 | Add/move/reorder restores focus after the complete tree including rows is attached, blocked-save notice persists and is announced, saving stays visible until its own settlement. | Late response, double save, removed view and stale render cannot erase newer edits/notice or focus a detached node. |
| P03 | A standalone public `identity<T>` checks **and emits** with no symbolic type in the emitted graph; public identity/helper/known-composite chains compile across packages, including `nested<T> → identity<box<T>>`; stationary and finite-permutation mutual calls check atomically; reached `nested<int>` directly calls concrete `identity<box<int>>`. | Private template under opaque argument, illegal callee arithmetic, false proof after failed component, expanding cyclic `T[]`, missing concrete target and bogus owner construction all fail with declaration/call-site evidence. |

The HTML swap policy uses one page-level `noSwap: [204,304,"4xx","5xx"]`
plus exact `hx-status:422/409/403/503` attributes generated only from the
checked HTML action's cases. The pinned asset checks exact status before the
broad family, as confirmed in the [Chromium probe](evidence/2026-09-24/selected-behavior/htmx-proposal.md).
A separate [guard experiment](evidence/2026-09-24/selected-behavior/experiments/htmx-guard/findings.md)
exercises the event positions, admitted OOB/partial tasks and both
missing-target timings against the pinned asset.
A compiler-owned adapter associates an action source and its **exact target
node identity** with each request. At `htmx:before:request`, it cancels a
submission whose target is absent or disconnected and reports one sanitized
`action::missing_target` occurrence; no HTTP outcome exists for that case.
At `htmx:before:response`, it rejects action response-control headers such
as `HX-Retarget`, `HX-Reswap`, `HX-Reselect`, redirect, refresh and history
controls before HTMX applies them. The action renderer itself has no API to
set these headers. At `htmx:before:swap`, it inspects the pinned asset's
entire `detail.tasks` array before any task executes. A blocked failure
(`swap === "none"`) cancels every task, including OOB content. An admitted
action response requires **exactly one `main` task**, `innerHTML` style, and
the same still-connected target node captured at submission; any `oob`,
`partial`, second, missing or redirected task cancels the whole swap. This
also rejects OOB markup in otherwise admitted 200/422/409/403/503 bodies,
which the earlier unrelated-500 probe did not test. A missing target during
flight reports `action::missing_target` with its received HTTP status; a
committed write remains committed or uncertain according to its server
outcome, never rolled back by the DOM failure. Other rejected task/header
shapes report a finite sanitized action-protocol occurrence. The same node
and task policy applies to success 200. Existing non-action 422 pages that
intentionally swapped need a checked local policy when the global exception
is removed. Browser tests must exercise both missing-target timings and OOB
markup in every admitted status, and observe DOM, focus and live-region state
as well as HTTP bytes.

The [native regex probe](evidence/2026-09-24/selected-behavior/regex-native-probe.md)
confirms the intended `matchAll` advancement on Bun 1.4.2. The browser
profile, action lowering, generic checker and HTMX adapter still require
implementation and full acceptance tests. A soundness or runtime-closure
gate failure returns that mechanism to design; it never silently weakens
the selected Fix. These concrete rules can now be decomposed into ordered
implementation tasks without reopening the 19 deferred proposals.
The [focused experiment index](evidence/2026-09-24/selected-behavior/README.md#focused-mechanism-experiments)
separates observed current failures and native event behavior from the
unimplemented profile, action adapter and SCC proof. General stack-safe
iteration (P05) and HTTP hedge/request cancellation (P06) remain deferred;
their review probes do not promote them into this upgrade.

For this design review, `canlc parse` accepted the shared record/variant
portion (24 declarations), the server call-site excerpt (three
declarations), the proposed browser `main` with a minimal package header,
and both generic packages. Parsing does not resolve missing application
helpers or type-check the proposed action grammar. No browser bundle,
authenticated invoice request or public generic call was built by this
documentation change.
