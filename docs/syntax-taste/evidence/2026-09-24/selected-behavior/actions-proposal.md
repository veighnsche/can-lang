# U01–U03: shared action declarations and request-aware server bindings

Design proposal, 24 September 2026. This file proposes implementation details for already accepted behavior; it does not claim that the examples compile today. The coordinator owns the three fresh Jev consultations. No runtime/compiler changes or consultation results are included here.

## Authority and current evidence

The selected architecture is [the integrated action contract](../../../preparation/integrated-action-contract.md#source-declarations-and-boundaries): one locked pure contract package, handler-free declarations, request-aware server mounting, protected operation checks, and native routing. [Dispositions U01–U03 and S01/S03](../../../post-upgrade-dispositions-2026-09-24.md#unfinished-accepted-requirements) reaffirm it. Bound-callable client projection is deferred P01; it is not a competing default. There is no compatibility obligation.

Current implementation differs in concrete ways:

- `compiler/internal/check/actions.go:286` requires a non-generic, total named handler in each declaration, with positional capture inputs followed by the body. There is no request input. `ActionDeclaration.Handler` at line 50 couples shared metadata to server code.
- `compiler/internal/emit/actions.go:41` emits that handler identity. `runtime/platform/action-json.ts:27` includes it in the action entry type. Both need a declaration/binding separation.
- `examples/invoice-grid/src/web/web.can:10` mirrors server declarations and supplies never-served stubs. `examples/invoice/src/web/web.can:808` mounts exact manual routes; load uses `/invoices/load?invoice_id=…` instead of its declared captured route.
- `runtime/platform/server.ts:194` already receives both native request and Can request snapshot, dispatches canonical captures, and abandons the request in `finally`; it has the needed request context at the runtime boundary.
- Existing named callable capture is checked in `compiler/internal/check/callables.go:123`; `compiler/testdata/current/callables/captures.can` demonstrates `near` values captured when a callable is constructed. It can bind a startup pool without a service container.
- `runtime/platform/html.ts:402` puts every 4xx/5xx except 422 in HTMX `noSwap`. Thus 403 and 409 also need qualification alongside S01's 503.
- `tests/integration/gate5_frontend_test.go:12` records route rewriting, focus before attachment, erased blocked-save notice and absent pending indicator as limitations; these become positive acceptance assertions.

The current action grammar is `body json|form <record>`, `handles <function>`, `result <variant>`, `{capture}` path placeholders plus individual `captures` rows, and `leaf => status` cases. The selected earlier grammar below differs deliberately; do not silently describe it as current syntax.

## Proposed declaration grammar

Use the earlier accepted spelling rather than retaining parallel dialects. All code in this section is **proposed grammar**. Ordinary record/variant definitions are omitted; their exact fields are the integrated packet's table, including operation IDs in every save result. These are transparent wire values, never SQL handles or authority tokens.

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

`invoice_key` is a transparent record with exactly `int tenant_id` and `int invoice_id`, as selected in the earlier typed-capture packet. Each path capture must occupy one whole segment, appear once, and match exactly one field; only `str` and canonical int64 `int` are admitted generally. Record field order does not alter URL binding: names determine the map. Int parsing, escape rejection and static precedence retain the capture packet's contracts. The emitted Bun route path uses native `:name` parameters, with the guarded raw-path validation required by Can's stricter capture contract.

Exactly one input clause, one output mode, one result type and one exhaustive case table are required. GET accepts only `input none` and JSON output. POST accepts form-to-HTML or JSON-to-JSON. Input/body limits are positive compile-time integer literals; `rows_limit` applies only to a form schema with rows. JSON application row/domain bounds remain authored validation. No `handles` clause is permitted. Distinct leaf/status consistency and the current ban on body-forbidden statuses are preserved; response schemas include every leaf exactly once. HTML swap policy is declaration metadata, not inferred from status classes.

An action reference is a compile-time symbol keyed by canonical locked package instance and declaration name. It is neither an ordinary string nor a first-class runtime value. APIs below consume it in a checked action-reference position. Runtime reflection, dynamic action lookup and user-authored descriptors are excluded.

## Proposed mount and callable contract

All new API spelling below is **proposed grammar/API**. `callable <name>` and `near` use the current named-callable mechanism. `action::mount` is one compiler-checked intrinsic with action-dependent signatures, not a general overload/variadic-function feature:

| Action | Explicit operands after action symbol | Effective handler inputs | Handler output/errors |
| --- | --- | --- | --- |
| JSON GET | handler | `http::request`, capture record | exact declared result; `emits []` |
| JSON POST | handler | `http::request`, capture record, declared JSON record | exact declared result; `emits []` |
| HTML POST | handler, normal renderer, rejection renderer | `http::request`, capture record, declared form record | exact declared result; `emits []` |

Renderers respectively accept the exact result or `form::rejected<wire>` and return `html::safe`, both `emits []`. Captured `near` parameters are absent from the effective callback input list. Generic, variadic, wrong-order and wrong-type handlers are rejected at mount. No-capture declarations omit the capture operand/input instead of inventing an empty-record convention. Callback values must retain the existing checked callable evidence; no new lambda syntax is needed.

Mount produces an `http::route`, consumed by existing `http::make_router`. All declaration validation happens at compile time. Route construction follows the existing `http::invalid_route` boundary; router composition detects `http::duplicate_route` and `http::ambiguous_route`. Statically known conflicts should also diagnose early, but dynamically assembled route lists retain checked runtime failure. Mere import never registers a route. Only mounted actions enter the server route table; unmounted declarations remain usable for clients. The same action can be mounted once per router and on independent server instances with different explicit pools.

```can
/// Proposed API call sites inside the existing invoice_routes chain:
call action::mount(contract::load_invoice_grid, callable load_grid) as http::route load
call action::mount(contract::save_invoice_grid, callable save_grid) as http::route save
call action::mount(contract::save_invoice_html, callable save_html, callable render_html, callable render_bad_form) as http::route form
call http::make_router([load, save, form]) as http::router built
```

The mounting function has `near sql::pool pool`, as today's invoice route builder does. Handlers have the following **proposed signatures**, with ordinary checked/asserted bodies supplied by the application:

```can
fn contract::grid_load_outcome load_grid
    emits []
    given
        near sql::pool pool
        http::request req
        contract::invoice_key key

fn contract::grid_edit_outcome save_grid
    emits []
    given
        near sql::pool pool
        http::request req
        contract::invoice_key key
        contract::grid_edit_input body
```

These are signature fragments, not complete compilable functions. `callable load_grid` at the mounting site captures the lexical `pool` under existing exact-name/exact-type rules. Different dependencies can be passed as additional explicit `near` inputs; no implicit current actor, service locator, per-request pool opening or environment read is introduced. The browser imports only `contract`; server handlers and closures live in the server project.

Client URL/request construction follows the already selected `action::url`/`action::request` contract with action-symbol and typed-record operands. POST adds the exact body record; GET rejects one. A method/body/result change invalidates mismatched constructions and mounts. Field edits invalidate constructors, form controls and projections. Route-only changes may compile cleanly and must update both actual server registration and client URL from shared metadata.

## Lifecycle, security and errors

1. Open the configured SQL pool once before constructing route closures. Bind it explicitly, start the server, and retain pool ownership until server shutdown has stopped admission and settled callbacks. Then close it exactly once. On routing/start failure close it as well. Reuse current resource registration, callback guarding and shutdown precedence; mount does not transfer pool ownership or silently close it.
2. Server ingress enforces its global request budget; the action adapter enforces the tighter declared budget and media/capture rules. It consumes the body once, creates immutable decoded wire values, and passes the request capability for header/session inspection. Re-reading its consumed body retains existing resource/body-consumption failure behavior. Native mutable `Request` never escapes through the Can handler.
3. Handler authenticates each request from server-controlled session state via request headers; no session token/actor field is accepted as client permission. All three actions use the same application-authored authentication path. POST additionally checks the selected CSRF mechanism before the protected mutation. Cross-origin/session-cookie defaults are not a substitute for that check.
4. Validate input and apply membership, tenant and invoice checks at the protected read/write. The write transaction predicates include authorization and expected revision, so a revoked member cannot commit on a stale earlier check. Authenticated foreign/nonexistent invoices share the same nondisclosing 403 result. Captures merely identify the resource. Auth/session failures map to the declared nondisclosing forbidden leaf; no protected draft/snapshot is echoed.
5. The request capability belongs to the callback lifecycle; retained use after completion must fail resource checks. The independent audit found that current `requestSnapshot` only checks WeakMap presence and `abandonRequest` does not revoke buffered or live snapshots, so token revocation after per-request owner drainage is required new adapter work. Server callbacks are guarded against closed captured resources. Compiler does not claim to prove application authorization, pool shutdown ordering or all lifetime escapes statically.
6. Normal handler outcomes use exact case statuses. Capture/JSON malformation is 400, route miss 404, wrong method 405 (with Allow), body excess 413, wrong media 415, structural form rejection 422 through its dedicated safe renderer. These are adapter responses, not forged application result leaves. Unexpected native/renderer faults produce sanitized 500 and server diagnostics. A fault after commit cannot be described as rollback.
7. Scope `operation_id` by server actor and captured invoice; store canonical-request digest and committed response in the same transaction as mutation. An independent audit clarified that every replay must first recheck current session, tenant membership, invoice ownership and disclosure in the serialized lookup transaction; a revoked actor gets nondisclosing 403, never a stored saved snapshot. An authorized identical replay returns that response without a second mutation; same ID/different payload conflicts. Include captured tenant/invoice in scope/digest so path edits cannot change an attempt's destination silently. Every save leaf echoes the ID where decodable and the client verifies ID/status/leaf agreement. Malformed inputs have no obligation to fabricate an ID.
8. 503/abort/lost response/disposal leaves commit uncertain. Preserve the exact attempt for explicit identical replay; never automatically create a new ID. HTML rereads before another uncertain write. A fresh read does not prove whether a particular older attempt committed. Ledger retention/cleanup must preserve the documented reconciliation window; reload-durable drafts remain excluded.

Native lowering is a JS closure over the existing pool handle, `Bun.serve` route registration, native `Request`/`Response`, native Fetch and `AbortController`, and existing SQL transaction adapters. Compiler-owned immutable schema/case/capture tables supply only the adapters Can needs. Do not implement a second HTTP router in Can, a custom transport stack, mutable service registry or a new promise engine. Preserve exact codec/int64 adapters where native JSON cannot meet Can's contract.

## Compile and runtime acceptance

These are implementation acceptance fixtures, not claims of passing tests today. Supply complete record/function/assertion definitions around the fragments.

| ID | Exact fixture/change | Required result |
| --- | --- | --- |
| A01 | All three declarations above in one locked contract package; server mounts them and browser references JSON actions; no handler in contract | Both targets compile; browser closure contains no SQL/server/handler entry |
| A02 | `action::mount(contract::load_invoice_grid, callable load_grid)` with signatures above and lexical `sql::pool pool` | Compiles, captured pool is explicit; two server instances use their own pools |
| A03 | Delete lexical `pool`, or provide same name with `str` type | Capture diagnostic at callable/mount site |
| A04 | Replace handler inputs with `(invoice_key)` or `(invoice_key, http::request)` | Mount signature diagnostic; missing/wrong-order request |
| A05 | Handler returns `grid_edit_outcome` for load, emits SQL failure, or is generic/variadic | Mount diagnostic before emission |
| A06 | `action::mount(contract::save_invoice_html, callable save_html)` | Missing two renderer operands diagnosed; swapping their order also fails |
| A07 | Add `handles load_grid` to shared action, use string/computed action name at mount, or mount inside reachable browser callback | Diagnose obsolete clause / non-symbol reference / target capability respectively |
| A08 | Rename `invoice_id` capture field to `id` without route edit; use nested/owner capture type | Capture-record/path mismatch or unsupported type diagnostic |
| A09 | Remove one result case; add unrelated leaf; add `swap inner` to JSON case; give GET `json grid_edit_input limit 8192` | Declaration diagnostic |
| A10 | Add request body to GET construction; pass old JSON body after switching action to form | Client construction diagnostic |
| A11 | Remove/rename wire field or result leaf in locked contract and rebuild both targets | Every stale constructor/control/match/mount fails or updates; no mirrored contract survives |
| A12 | Change only both JSON routes' literal prefix from `/api/` to `/v2/`, rebuild both targets | Both compile and browser reaches new live captured route; old route 404; no harness rewrite |
| A13 | Mount same action twice into one router; compose equal-shape captures or incomparable overlapping routes | Duplicate/ambiguous route diagnostic or named router failure; static route wins admitted precedence |

Real HTTP + database + named-browser acceptance must cover each declared status and exact body; direct captured authenticated GET and explicit reread; no cookie and hostile client identity; cross-tenant/nonexistent nondisclosure; concurrent revoke/revision races; malformed capture/JSON/form, duplicate/missing fields, exact numbers, 64-row boundary, budgets/media; same/different digest replay and lost acknowledgment after commit; renderer fault after commit; close/start failure and in-flight shutdown. Observe SQL effects and pool open/close counts, not just response text. Header/session access must work from every mounted handler without body reread or URL rewrite.

## S01 and S03 product acceptance tied to the contract

S01: all declared HTML leaves, including 403/409/503, return their real status and replace the admitted feedback target using `swap inner`. Structural 422 follows its rejected-form rendering path. Unrelated infrastructure errors must not become arbitrary body swaps merely because one action admits that status. Implement a compiler-owned checked action swap policy using the pinned HTMX integration; a blanket global “swap all errors” policy is insufficient. Prove policy with a real browser, existing stale feedback, a real 503, and an assertion that its replacement is visible. Keep text escaping, target admission and CSP intact. Exact HTMX hook/header mechanism requires a focused prototype against pinned vendor code; do not edit vendor code by hand.

S03: store notice/pending/focus intention in immutable grid state before rendering. Resolve stable row key and intended control after `replace_children` attaches the current tree, then focus only if view/state still matches. Add/move/reorder must land on the intended attached control; deletion chooses the documented surviving control. Preserve normal typing focus. Put the blocked-save notice in rendered state so immediate rerender cannot erase it, and expose it through the existing admitted live-region/ARIA surface. Represent save pending before awaiting Fetch and until that same attempt settles. Compare attempt ID and state version before applying late responses; never clear a newer pending operation or overwrite newer edits. Disposal releases listeners/requests and prevents late DOM/focus work. Test delayed requests, double-save, edit-during-save, stale completion and disposal with DOM/focus/announcement observations. No new component framework, exact text, reload persistence or automatic retry is required.

## Details to resolve during mechanism selection

- Confirm the concrete compiler representation for action-symbol operands and capture-record construction; it must remain static evidence rather than a string lookup API.
- The follow-up sections below specify the proposed exact-Origin CSRF policy and bounded replay cleanup semantics. A numerical reconciliation window remains an explicit product parameter needed before cleanup qualification.
- Prototype declaration-scoped HTMX status/target policy; the present global config alone cannot express the required selectivity.
- Implement and test request-token revocation after per-request owner drainage; the current WeakMap guard does not cover retained request misuse. Confirm pool drain-before-close separately.
- Reconcile existing wire field names and current example SQL schema to the accepted table, including operation ID response echo and removal of client session identity. Do not preserve old shapes merely to keep goldens passing.

## Concrete first-scope CSRF policy (follow-up proposal)

Select **mandatory exact Origin validation for both POST actions**, with one public origin explicitly configured at startup and captured by the server handlers. This is an application deployment policy, not an implicit language permission. It needs no browser-accessible cookie, secret, synchronizer token, custom header or mutable native event. Both the native HTML form/HTMX POST and native browser Fetch POST carry the browser-generated Origin header. Browser Can continues using checked same-origin requests and native cookie attachment.

This is a deliberately narrow same-origin browser deployment. The [Fetch Standard's Origin algorithm](https://fetch.spec.whatwg.org/#origin-header) attaches Origin for non-GET/HEAD requests and can serialize it as `null` under privacy/redirect conditions. Consequently the proposed policy rejects some otherwise legitimate privacy/sandbox/redirect configurations; qualification must use the actual supported browser/page configuration. Serve the application with `Referrer-Policy: strict-origin-when-cross-origin`, direct same-origin action URLs and no cross-origin POST redirects. [OWASP's origin verification guidance](https://cheatsheetseries.owasp.org/cheatsheets/Cross-Site_Request_Forgery_Prevention_Cheat_Sheet.html#using-standard-headers-to-verify-origin) supports checking the complete origin and failing closed when origin evidence is absent. The stricter no-Referer-fallback choice here is ours, not a claim that OWASP requires this exact policy.

Exact rules:

| Input/context | Outcome |
| --- | --- |
| Startup origin is one canonical `https://host[:port]` origin, with no path/query/fragment/userinfo; explicitly configured loopback HTTP origin in local tests | Accept configuration. Canonicalize/validate once with native URL semantics; capture the resulting exact serialized origin. |
| Missing/invalid startup origin, wildcard, list, or non-loopback HTTP deployment origin | Startup fails before serving; do not infer target origin from Host, Forwarded or X-Forwarded-* supplied on a request. |
| POST has exactly one Origin value byte-equal to configured canonical origin | Pass CSRF check only; continue session authentication and transaction authorization. |
| Origin missing, empty, `null`, malformed, multiple, comma-joined, wrong scheme/port/host, sibling subdomain, prefix/suffix lookalike, or trailing slash | Deny; a valid Referer, Sec-Fetch-Site or session cookie never overrides this failure. |
| GET load | No Origin requirement; no business mutation; request authentication and protected read checks still apply. |

For the first scope, Sec-Fetch-Site is not an additional admission dependency. [The Fetch Metadata specification](https://www.w3.org/TR/fetch-metadata/#sec-fetch-site-header) distinguishes `same-origin`, `same-site`, `cross-site`, and `none`; `same-site` must never be treated as proof of this application's exact origin. Keeping the mandatory Origin check authoritative avoids a second fallback matrix. This does not authenticate non-browser callers: they can supply an Origin header but still require valid server-authenticated credentials. It also does not solve same-origin script compromise, which remains governed by the browser asset/CSP boundary.

Check after bounded wire decoding and before any protected operation, including replay lookup. For a valid decoded JSON input denied by this policy return `grid_forbidden(body.operation_id, "request denied")` with actual 403; HTML returns its normal nondisclosing `forbidden` leaf and visible 403 renderer. No protected data or draft is echoed. Malformed capture/body/media/size failures keep the previously specified adapter precedence (400/413/415/structural 422); they cannot reach a mutation and need not fabricate an operation ID. A rejected-form renderer must not access protected data. This fixes ordering unambiguously without adding an unrelated ingress-policy API.

`http::request_headers` already returns immutable header records (`runtime/platform/http.ts:814`). `model::session_of` already parses cookies only in server Can (`examples/invoice/src/model/model.can:39`). Do not use its adjacent first-match `find_header` helper as Origin verification: author a pure `unique_header` helper that rejects multiple matching fields; a native-coalesced comma value fails exact comparison as well. Match header names ASCII case-insensitively, or rely only on an explicitly tested lowercase snapshot guarantee. No per-request URL parser is needed: compare the browser's serialized value to the validated configured string.

Proposed Can handler **sketch**, omitting required assertion rows and application helper bodies. Only the mount/capture-record contract is new language/API work; the helper functions below are ordinary application Can, not catalogue APIs:

```can
fn contract::grid_edit_outcome save_grid
    emits []
    given
        near sql::pool pool
        near str public_origin
        http::request req
        contract::invoice_key key
        contract::grid_edit_input body
    match call http::request_headers(req)
        ok http::header[] headers => match call origin_is_exact(headers, public_origin)
            ok bool allowed => match allowed
                false => ok contract::grid_forbidden(body.operation_id, "request denied")
                true => match call model::session_of(headers)
                    ok option::value<str> session => match session
                        option::none => ok contract::grid_forbidden(body.operation_id, "request denied")
                        option::some => ok call save_for_session(pool, session.value, key, body)
```

`origin_is_exact(headers, public_origin) -> bool emits []` follows the table, without first-match ambiguity. `save_for_session` is the proposed total transaction wrapper that maps SQL outcomes to the declared variant and rechecks session/membership/resource/revision at the write. HTML has the identical header gate followed by its own input/outcome adapter. Both mounting functions provide the lexical `public_origin` and `pool`; omission or wrong capture type diagnoses under ordinary `near` rules. Cookie issuance must use the deployment's server-only, HttpOnly session configuration; browser code receives no session token in bootstrap, wire body or rendered hidden control. Remove today's echoed session form field.

Qualification adds real form and Fetch POSTs in each supported browser, proving native Origin succeeds; valid cookie plus missing/null/foreign/sibling/wrong-port/multiple Origin all deny without SQL effect; valid Origin with absent/revoked session denies; valid Origin with cross-tenant captures denies; changed proxy Host cannot change configured origin; malformed bodies retain the listed adapter status; browser emitted closure has no cookie/secret access. Synthetic HTTP cases may deliberately set Origin as test transport input; browser production code must not set it itself.

## Replay retention: bounded rule and missing duration

A numeric duration cannot be derived from this repository's accepted contract. It promises a bounded reconciliation window but supplies no maximum disconnected/in-flight lifetime, product support interval, storage budget or permitted post-expiry experience. Choosing 24 hours or seven days would be a new product assumption, not evidence. Record this as a required invoice acceptance parameter `W` (a positive server-configured duration) before implementing cleanup or claiming the replay gate complete.

The recommended bounded rule is: transactionally record commit time, canonical digest and full response; retain the entry for at least `W` after commit, with no sliding extension on replay; cleanup only entries strictly older than that threshold using server time. Replay and cleanup serialize with the mutation transaction so a request cannot pass ledger lookup immediately before its entry is removed and execute again. Keep entries longer when the clock moves backward; cleanup delays may extend retention, never shorten the guarantee. Time source/cleanup cadence must be tested at the exact expiry boundary.

The bounded **replay-result guarantee** ends at `W`. After expiry the UI must describe unresolved old attempts as uncertain, offer an authorized reread and require deliberate user reconciliation before a new operation. A read alone still cannot prove the historical attempt committed. Silent new-ID retry remains forbidden. For this invoice sample, every mutation must increment a never-reused revision and every retry must carry the original expected revision: after a successful old commit, a ledger-missing identical replay must conflict on revision and cannot perform a second effect. No revision reset, wrap, invoice-ID reuse or restore-to-old-revision may be admitted under that argument. If such lifecycle operations are needed, introduce an incarnation/epoch or durable consumed-ID evidence before enabling cleanup. A failed old attempt that never committed can still commit later if authorization/revision remain valid; only explicit retry authorizes that possibility.

This rule bounds retained successful response history but does not bound traffic/storage per unit time; a hard storage bound additionally requires rate/admission capacity. It does not promise that an expired operation ID reused with different payload can always be detected after its ledger entry is deleted. If the selected product requires indefinite ID non-reuse detection or indefinite exact-result replay, finite deletion is incompatible without a stronger token/epoch/tombstone design. Until `W` and these expiry semantics are selected, do not delete the current ledger or assert cleanup qualification.
