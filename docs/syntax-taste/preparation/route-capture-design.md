# DI-09a typed route-capture increment

24 September 2026. Design preparation for the selected source `action` contract. This document records the acceptance cases **before** disposable Bun probes and Jev consultations. No production capture parser, runtime adapter, or source grammar has been implemented.

## Registered questions and cases

The proposed declaration replaces the exact route/query pair with `post "/tenants/:tenant_id/invoices/:invoice_id"` and `captures invoice_key`. The only initial capture fields are required `str` and `int`, each occupying one path segment. The action identity still owns the method, builder, mount, form, and finite response policy. The invoice URL must be navigable as a resource path.

| Group | Positive cases | Negative or boundary cases | Required observation |
| --- | --- | --- | --- |
| `int` | `1`, `7`, zero and signed int64 endpoints if admitted | `+1`, `01`, `-0`, whitespace, fractional, exponent, beyond int64, empty | Exact wire grammar, typed value and canonical builder output. |
| `str` | ASCII, space, percent, non-ASCII Unicode | Empty, slash, backslash, dot segments, controls, malformed percent, invalid UTF-8 bytes, unpaired surrogate | Builder/matcher round trip without route-structure change. |
| Shape | Two required captures in the declared order | Missing/extra segments, trailing slash, wrong literal, duplicate capture names, record field mismatch | Compile or HTTP distinction, and no callback on rejection. |
| Routing | Exact/static match and one dynamic match; another method at same path | Duplicate method/pattern; static/dynamic overlap; two dynamic shapes matching one path; wrong method | Deterministic precedence or assembly rejection; 404/405 and `Allow`. |
| Native integration | Ordinary `URL`, `Request`, and pinned Bun `fetch` observations | Dot normalization, encoded separators, raw backslash, malformed escapes | Record what Bun has normalized before the current `normalizedPath` adapter receives a URL. |

The final design must specify 400 malformed capture versus 404 unmatched path versus 405 recognized path with unsupported method, exact `int` range and spelling, one-pass segment encoding/decoding, Unicode policy, mount ambiguity, and native lowering. A builder proves the path is syntactically admitted by the action; it does not prove deployment, resource existence, or actor permission.

## Disposition for the optional increment

Select a **guarded native Bun route** trial, separate from the already selected exact-path `action`. The bounded source change is:

```can
action save_invoice
    post "/tenants/:tenant_id/invoices/:invoice_id"
    captures invoice_key
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

The shown declaration is a **planned grammar line**, not accepted source today. `invoice_key` is an ordinary declared record with direct `int tenant_id` and `int invoice_id` members. For this first capture trial, `captures` replaces `query`; combining both is separate work. One checked action symbol still drives `action::url`, the POST consumer and `action::mount`. The generated builder for `(1, 7)` is `/tenants/1/invoices/7`, a navigable resource path. It does not witness existence, authorization or deployment. Resource-key positivity is domain validation after signed `int` parsing.

The route pattern starts with one slash and has required, nonempty segments. In this increment, each static segment is ASCII unreserved `[A-Za-z0-9._~-]+` except `.` and `..`; capture segments use `:name` with a unique `[a-z][a-z0-9_]*` name. Every capture name must correspond exactly to one direct record field of type `str` or `int`, with no extra fields, wildcards, optional captures, repeated names, nested records, query marker or fragment. A static path under reserved `/__can` stays forbidden. The grammar/parser diagnoses a missing field, extra field, wrong type, duplicate name, empty or trailing pattern segment, unsupported metacharacter, and unresolved package type before emission. This is intentionally narrower than a general route language.

`int` uses canonical raw ASCII `0` or `-?[1-9][0-9]*` in signed range `[-2^63, 2^63-1]`. Reject `+1`, `01`, `-0`, `%31`, decimal/exponent forms, whitespace and overflow with 400 after a matched path. The builder uses the native `bigint.toString()` decimal spelling after range checking. `str` is a well-formed Unicode scalar string, without empty value, `/`, `\\`, exact `.` or `..`, C0 controls or DEL. The builder checks this then applies native `encodeURIComponent` once to the single segment. The matched callback validates raw escapes with strict `decodeURIComponent` once, compares the result with the Bun parameter, and refuses malformed percent/UTF-8 and separators before the business handler. It must not trust Bun's lossy `req.params` decoding alone. Percent as data works: the builder emits `%25`; `%252F` decodes once to the literal text `%2F`, not a slash. No Unicode normalization or case folding is imposed. Builder failure is an explicit `action::invalid_path` result; invalid inbound captures get 400.

Use Bun's native `Bun.serve({routes})` for path/method selection, with **method first, then exact static path ahead of parameter path within that method**. The same pathname may therefore select different actions for different HTTP methods. That is consistent with action identity including its method. At assembly, reject duplicate method/pattern registrations and any two intersecting dynamic patterns for one method. A fully static route may overlap a dynamic route for the same method; native exact-route precedence selects it. Other overlapping mixed dynamic patterns remain rejected until their native ordering is qualified. The same structural pattern may be registered for distinct methods, with a shared capture spelling/type contract where its source path is the same. Check these collisions across source actions and existing literal mounts in one assembled table, not two unrelated registries.

The native callback must enter the existing guarded Can server lifecycle: request snapshot/body budget, scoped ownership, asset/reserved-path policy, response security headers and request abandonment still apply. It validates captures before calling the protected handler, then passes an immutable typed `invoice_key`; it must never hand raw `Request` or `req.params` to Can code. The current `normalizedPath`/exact router decodes the *whole* pathname and cannot be reused for this capture step. Generated TypeScript should use Bun's route table, native `URL`, `decodeURIComponent`, `encodeURIComponent` and `BigInt`; add only the strict validation, metadata-backed 400/405 fallback and lifecycle integration needed for Can's contracts. The existing server currently supplies only `fetch`, so integrating native route callbacks requires server adapter work and a test that these callbacks cannot bypass its security/body/scope behavior.

| Observation | Proposed result | Callback policy |
| --- | --- | --- |
| Native route matches method and path, captures valid | Normal action handler result | Invoke protected handler once with typed key. |
| Native route matches but captured text/encoding/type is invalid | 400 | Never invoke protected handler. |
| No native route matches; malformed percent/UTF-8 still visible in `Request.url` | 400 | Fallback never invokes business code. |
| No native route matches; normalized path shape is absent or has missing/extra/empty segment | 404 | Fallback never invokes business code. |
| No native route matches because the method is unsupported on an otherwise recognized, valid normalized path | 405 with `Allow` union of methods serving that pathname | Fallback never invokes business code. |
| Path shape fits but all candidate capture conversions fail, including a wrong-method request | 400 before 405 in the candidate trial | Fallback never invokes business code. |

The fallback classifies the normalized URL it can see; it is **not another dispatch path**. If several patterns fit its path shape, it returns 405 with the sorted union of methods for valid captures; it returns 400 only when none of those conversions is valid. Bun's native route callback can receive a captured `..` or raw backslash before `Request.url` normalizes them, so the callback rejects those as 400. An unmatched route with an original dot segment may already have a different normalized URL when it enters fallback. No contract can claim that fallback recovers the original request-target bytes or always returns 400 for that original spelling. It may report 404/405 for the normalized path. This does not authorize any protected operation. If future requirements demand a uniform original-target status, native `Request.url` is insufficient and the feature must revisit its ingress boundary.

## Evidence from pinned Bun and trial adapters

The [raw URL probe](probes/route-capture/probe.ts) and [observations](probes/route-capture/observations.json) ran under Bun **1.4.2**, revision `744846f844374847c902b5e7fd59b4342a51ef99`, using `URL`, `Request` and real `Bun.serve` requests sent by `curl --path-as-is`. URL preserves `%2F`, `%5C`, `%25` and malformed percent text until explicit decoding. Whole-path decoding turns `%2F` into a slash. Native URL construction removes raw and encoded dot segments and treats raw backslash as a separator before `Request.url` is observed.

The [native route probe](probes/route-capture/bun-native-routes.ts) and [response record](probes/route-capture/bun-native-routes.json) add a more important distinction: Bun's route matcher executes **before** the normalized `Request.url` becomes the callback's view. `POST /tenants/%2E%2E/invoices/7` called the invoice callback with `tenant_id=".."` although `req.url` showed `/invoices/7`; raw backslash similarly reached the captured parameter. Bun's decoded `req.params` replaced malformed `%ZZ` and bad UTF-8 with `�`, so it is not a strict decoder. `PATCH` on a POST route reached `fetch` as 404 without a Can adapter. `POST /invoices/new` selected the dynamic POST route, while GET and PUT selected their static same-method route. These are observed 1.4.2 behavior, not proof for another Bun target. [Bun's routing documentation](https://bun.sh/docs/runtime/http/routing) documents native parameter routes and decoding with replacement characters; the exact method precedence and dot behavior here come from the pinned live probe.

The first [custom fetch trial](probes/route-capture/trial.ts) passed **66 assertions** but demonstrated the cost of bypassing native matching: its normalized-path matcher dispatched a raw encoded-dot request to another action. The later [guarded native trial](probes/route-capture/guarded-native-trial.ts) passed **82 assertions** across 27 live HTTP cases, including canonical signed endpoints, malformed escapes, encoded separators, dot and backslash capture rejection, empty/extra segments, static/method selection and 405 `Allow`. Its [results](probes/route-capture/guarded-native-result.json) show only expected 200 cases entered callbacks. These scripts are disposable TypeScript probes, not Can compiler, HTTP lifecycle, Linux or application tests. The trial's literal `str` pattern and rejection policy need to be implemented through checked Can declarations and unified router assembly before any product claim.

## Jev advice and disagreement audit

Three fresh, fully rewritten requests in [initial route consultations](probes/route-capture/jev/) selected static-first path identity (.95/.88/.99), strict string segment admission (.99/.96/.99), and canonical signed integer wire text (.98/1.00/1.00). At that point the evidence covered `URL` and a `Bun.serve` `fetch` adapter, **not** Bun's native `routes` dispatch. A second three-request [dot consultation](probes/route-capture/jev-dot/) selected an explicit normalized-URL contract (1.00/.99/.99) after the custom fetch trial showed cross-route normalization. The later native route probe changed the relevant facts: native callbacks retained `..` in route params despite normalized `Request.url`. Three new, fully rewritten [native-route consultations](probes/route-capture/jev-native/) then selected guarded native method-first dispatch (1.00/.99/1.00) over custom static-first routing or postponement.

The apparent static-first/method-first disagreement is explained by the new observed native behavior and its adapter cost. The dot advice also applies only to the `Request.url`/fallback view; native callbacks can reject a matched dot capture before business code. Each consultation directory saves all three exact requests, raw responses, HTTP metadata and a full-wording audit. Every explanatory state, instruction and option description was rewritten while technical identifiers, constraints and alternatives remained semantically fixed within its trio. Jev returns choices and probabilities, not reasons or proof. The engineering selection rests on the pinned HTTP observations and the native-lowering rule, with the limits above.

## Acceptance for a later implementation task

1. Compile the illustrative `captures invoice_key` action with one canonical package/declaration identity; reject field/type/name/pattern errors, route collisions, a stale route/method reference and an imported action with a wrong capture record. A path change rebuilds both HTML and client action URLs; a declaration rename diagnoses stale references.
2. Verify builder-to-matcher round trips for `0`, `-1`, signed int64 endpoints, ASCII, space, `%`, Unicode and `%252F` as literal string data. Reject invalid builder values and inbound `+1`, `01`, `-0`, `%31`, overflow, malformed percent/UTF-8, encoded slash/backslash, dot segments and control bytes. No double-decoding or Number precision loss.
3. Exercise duplicate/ambiguous patterns, static-over-dynamic precedence *within one method*, distinct-method overlap, wrong methods with `Allow`, empty/trailing/extra path segments and fallback normalized-dot cases on the pinned Bun release and the selected Linux target. Record callback counts; invalid inputs must not enter the protected handler.
4. Prove native route callbacks share the current server's authentication/CSRF boundary, body limit, scope/disposal, asset precedence and response headers. Run real HTTP/browser/database observations, including cross-tenant and stale-revision attempts. A valid capture cannot grant authorization or weaken the five 200/422/409/403/503 response cases.
5. Compare exact-path/query and typed-capture resource-route tasks under the evaluation protocol, counting total agent tokens only for semantically successful creation/refactor/repair attempts. The capture increment is accepted only if it closes the registered navigable-resource/rename gap without a material reliability regression.
