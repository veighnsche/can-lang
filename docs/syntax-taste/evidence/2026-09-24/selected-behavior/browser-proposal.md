# Browser repair and delivery proposal (B01, B02, U04, U05, S02)

24 September 2026. **Status: design proposal, not implemented or qualified.** This file distinguishes observed source behavior from choices for the next upgrade. The selected outcome is in [the dispositions](../../../post-upgrade-dispositions-2026-09-24.md); exact new catalogue spellings here are proposals for that outcome. There is no compatibility obligation to keep the current event record or the test bundle layout.

**Post-audit integration:** the [selected behavior packet](../../../post-upgrade-selected-behavior-2026-09-24.md#browser-build-and-event-execution) strengthens this proposal in three places. `URLSearchParams.getAll` alone is replacement-tolerant on `%ZZ` and `%FF`, so the selected query operation strictly validates raw pairs with native `decodeURIComponent` before using it. The invoice key retains the earlier two canonical `int` captures, rather than a single string ID. The supported `canlc build --target browser` result is served-ready and explicitly paired with a verified server build; the separate bundling step proposed below is historical.

## Evidence and mechanism choice

The compiler already emits a distinct `browser.ts` entry, checks a Can capability closure, and writes a content-addressed module manifest (`compiler/internal/emit/browser.go`, `compiler/internal/browser/browser.go`). The entry imports `runtime/diagnostics.ts`; the current reachable runtime also imports `node:crypto` (`domain.ts`), `node:util` (`data.ts`, `failure.ts`, `owner.ts`, diagnostics), and `node:async_hooks` (`owner.ts`). The Gate 5 bundler replaces `node:fs`, `node:util`, `node:crypto`, and `node:async_hooks`. Its async-context replacement holds one synchronous current value and loses that value after an `await`; the grid does not exercise a live owner context. Its file shim bakes the source index but discards module maps. The static browser audit skips runtime artifact bodies and rejects any authored emitted bytes containing `process.`, `Bun.`, `require(`, or `node:` even inside quoted text. The current event adapter turns a native event into four strings and discards every handler completion. The current regex scan increments `lastIndex` by one after an empty match.

**Choose a browser-specific runtime profile**, selected at build time. A fully maintained shim layer would need a faithful asynchronous `AsyncLocalStorage`, proxy detection, synchronous filesystem reads for diagnostics, and Node crypto semantics in every admitted browser. The existing four shims do not establish those contracts. The profile retains native JavaScript operations and replaces only unavailable host bindings and source metadata delivery. It is still one Can semantic contract: parity tests compare observable completions, equality, codecs, ownership, and diagnostics across Bun and browsers. No arbitrary authored JS/TS, worker, third-party asset, or general host adapter is admitted.

The profile must use a browser-specific generated entry and selected runtime module mapping, rather than bundler aliases for `node:`. Proposed bindings are:

| Current runtime dependency | Browser profile mapping | Required semantic evidence |
| --- | --- | --- |
| `domain.ts` → `node:crypto.createHash` for concrete type identity | Build computes and seals all concrete identities from checked declarations; browser module consumes immutable verified values. If an identity must be computed at runtime, use `crypto.subtle.digest("SHA-256", …)` asynchronously at initialization, before any action. | Same digest/identity bytes as Bun for all reachable shapes and forged-plan rejection; no handwritten hash. |
| `data.ts`, `failure.ts`, `owner.ts` → `node:util.types` | Keep Can-created records/arrays and failure occurrences branded by private `WeakMap`s; inspect only these for Can equality/containment. Untrusted host objects cannot acquire that brand. Sanitize host failures using bounded native `Error` checks in `try/catch`, otherwise report generic native exception; never read arbitrary getters or serialize a native cause. | Plain, nested, foreign, cyclic, proxy-wrapped, and throwing-proxy vectors; same Can result or safe finite failure, no attacker getter execution or leaked stack/message. |
| `owner.ts` → `node:async_hooks.AsyncLocalStorage` | Browser emission threads an opaque execution token through generated calls and runtime owner/coordination adapters, including participants and guarded callbacks. Callback registration captures the token; each async branch carries its own token across `await`. Native `Promise.all`, `allSettled`, `any`, and `race` still perform selection. | Concurrent roots and nested awaits cannot cross-acquire leases; admitted coordination drains losers and reports late failures under the correct owner. A synchronous global current-context shim is forbidden. |
| `diagnostics.ts` → `node:fs.readFileSync` | Bundle a build-generated immutable diagnostic table from checked Can spans and the final bundle source map. Browser entry initializes it synchronously before `$canMain`. Parse stack frames only as optional hints; always include the checked origin frame. | Offline startup and a deliberate fault show exact Can file/span, finite sanitized category and one occurrence; no filesystem fetch, native path/source content disclosure, or map dependency at fault time. |

The emitted TypeScript contract should be shaped as follows (names illustrative; the hidden token is compiler-owned and cannot be constructed in Can):

```ts
// compiler-emitted browser.ts after DOM readiness
import { $canInitialize } from "./state.ts";
import { configureBrowserDiagnostics } from "./runtime/browser/diagnostics.ts";
import { $canMain } from "./web/web.ts";
import { diagnosticTable } from "./diagnostics/browser-table.ts";
configureBrowserDiagnostics(diagnosticTable);
$canInitialize();
void $canMain(/* compiler-owned browser execution token */);
```

The supported command remains `canlc build --target browser <project>`, followed by a repository-owned production browser bundle step that reads the compiler manifest and produces the final JS, map, diagnostic table, and asset table. An empty Can app and the invoice grid take this same path. The server build publishes only the final content-addressed same-origin module route; the test harness cannot supply aliases, startup JS, or source metadata. Preserve `script-src 'self'`, `connect-src 'self'`, no `unsafe-eval`, no inline script, and exact asset paths. A final build failing this boundary fails delivery even if `canlc` emitted TypeScript successfully.

**Resolve the browser entry mismatch explicitly.** The accepted browser design requires one non-generic `void main` with no arguments and `emits []`; today's shared entry checker requires `void main(str[] args)`, and the grid's current `main(args)` reads its first item only because `build-grid.mjs` supplies handwritten boot code from `location.search`. Change the browser entry checker to require the accepted no-argument signature, while Bun entry checking keeps its own argument array. Change the generated browser entry to invoke `$canMain()` after DOM readiness and report a startup fault. Change the grid to obtain its invoice id through a bounded checked catalogue operation, proposed `browser::query_parameter("invoice") -> option<str>`. Its adapter uses `new URLSearchParams(location.search).getAll("invoice")`, returning none for zero values, the single immutable string for one value, and `browser::invalid_query` for duplicates or an oversized value. Admit only a literal ASCII key at compile time. The grid handles missing/invalid input visibly before `boot(invoice_id)`; it never treats query text as authorization, and the server validates the captured invoice again. This is a narrow bootstrap input, not a general `location`/navigation object. The generated entry contains the DOM-ready listener and no test-authored boot lines. A build with `main(str[] args)` must fail for `--target browser`; a no-arg main must compile and execute.

```can
// Proposed grid entry; query_parameter and show_boot_notice are proposed below.
fn void main
    emits []
    asserts
        missing: => ok
    match call browser::query_parameter("invoice")
        ok option::value<str> selected => match selected
            option::none => match call show_boot_notice("Choose an invoice to open")
                ok => ok
            option::some => match call boot(selected.value)
                ok => ok
        browser::invalid_query => match call show_boot_notice("Invalid invoice link")
            ok => ok
```

`show_boot_notice(str)` abbreviates an authored, checked helper. It mounts the server-rendered `invoice-grid` root, opens a view, creates a `role="status"`/`aria-live="polite"` node with `browser::create_element` and `browser::set_attribute`, writes the supplied text with `browser::create_text`/`append_child`, and attaches it under the root. Its own missing-root/disposed/rejected arms are explicit and produce a sanitized startup diagnostic if the required mount is absent; it never inserts HTML. This helper is included in both target checking and the production bundle. A real-browser test must observe the notice in the DOM and accessibility live region for missing and duplicate/oversized `invoice` parameters. The snippet follows the current `ok option::value<T> ... => match ...; option::some => ... .value` pattern in `examples/invoice-grid/src/web/web.can`; only the query operation and notice helper are new.

## B01: structural browser audit

Delete substring `serverTokens` classification. Retain the checker’s typed operation closure, then audit the **parsed final module graph** from the actual production bundle, including every reachable runtime module and generated metadata module. Allowed module edges must be local, accounted for, and resolved to manifest entries. Deny Node builtins, remote imports, `process`/`Bun` host operations, dynamic `import()`, `require`, and `eval` by parsed operation/import evidence; reject a runtime module as rigorously as an authored module. Report the first forbidden edge or operation with module path, import/use span, and originating Can call chain where available. A post-bundle token search may be defense in depth but cannot classify string literals as host access. The bundle and its maps must be checked for unapproved source content and secret bytes before publishing.

These Can functions **must compile** in a browser project and return their exact text:

```can
fn str label
    emits []
    asserts
        sample: => ok "Bun."
    ok "Bun."

fn str help
    emits []
    asserts
        sample: => ok "node: introduction"
    ok "node: introduction"
```

An actual call to `env::read`/`files::read`, an imported wrapper or generic specialization that reaches it, a server-only top-level initializer, or an emitted/runtime `node:fs` edge **must fail** with a useful location and chain. A quoted `"require("` and assembled `"Bu" + "n."` also compile. The exact input spelling of a forbidden call is subject to its catalogue signature; the required distinction is semantic host use versus data text.

## B02: native Unicode regex iteration

Keep regex compile admission (`i`, `m`, `s`, `u`, `v`; no authored `g`, `y`, `d`), the 0–10,000 result cap, fresh scan per call, UTF-16 offsets, and absent capture as `""`. Replace manual `RegExp.exec`/`lastIndex++` with the native iterator: `text.matchAll(new RegExp(source, flags + "g"))`, stopping as soon as `limit` records have been copied into immutable Can records. Native `matchAll` applies `AdvanceStringIndex` with Unicode mode for empty matches. This is an adapter for Can record shape and cap, not a second regex engine. `limit = 0` returns an empty array without iteration. Avoid spreading the whole iterator before capping.

```can
// Representative current catalogue calls; assert result starts, ends and groups.
match call text::compile_regex("(?:)", "u")
    ok pattern => match call text::matches(pattern, "😀x", 5)
        ok matches => ok matches
        error problem => error problem
    error problem => error problem
```

The `u` and supported `v` cases over `😀x` must report starts `[0, 2, 3]` and ends equal to starts. Non-Unicode empty scanning reports UTF-16 positions `[0, 1, 2, 3]`. A nonempty capture case, absent optional capture, `limit = 1`, and invalid/over-limit cases retain their current outcomes. Compare against native `String.prototype.matchAll` on Bun and each named browser.

## U05: cancellation during native dispatch

Current `browser::on_event(view, node, kind, handler)` takes a four-field immutable snapshot and invokes an asynchronous Can handler. Native default cancellation must happen before the event dispatch returns; awaiting a Can handler to decide is too late. Choose two bounded registration operations in addition to ordinary `on_event`:

```can
// Proposed catalogue spellings; callback still receives an immutable browser::event.
call browser::on_cancel_key(view, input, "keydown", "ArrowDown", callable on_field_key)
call browser::on_cancel_event(view, form_node, "submit", callable on_submit)
```

The first statically/runtime admits only `keydown`/`keyup` and one nonempty exact key. The second admits `submit` and any other explicitly listed cancelable kind, with a documented finite list; do not infer arbitrary cancelability from a string. At registration the checked arguments select the policy. In the native listener, test view liveness, event kind/key and `event.cancelable`, call `event.preventDefault()` synchronously for a matching policy, then snapshot and dispatch the Can handler. The ordinary `on_event` never cancels. If a view was disposed, its AbortController has removed the listener; a queued stale listener checks liveness again and cannot cancel. Do not put the raw Event or a mutable cancel token in Can data. Tests dispatch cancelable/noncancelable key and submit events, unmatched key, ordinary listener, and stale listener; inspect `defaultPrevented` immediately after `dispatchEvent`, before any callback promise settles. Failure in the later handler does not reverse cancellation or claim operation success.

## S02: observable handler faults

`runtime/platform/browser.ts` currently calls `invoke` and discards both completion and rejection in `settle`. Keep a view/app-scoped private browser reporter; every failed event or timer callback, including a thrown native exception converted by `invoke`, emits **one** sanitized diagnostic `{kind: "can.runtime-diagnostic", phase: "handler", occurrence, category, file, line, column}`. The diagnostic contains no native cause, stack text, input value, credential, or raw event. The occurrence is deduplicated by identity across any owner reporter and event reporter. The default production sink is a single `console.error` of this frozen data record; a supported host/test observer may subscribe through a compiler-owned hook with the same sanitized shape. A reporter failure cannot recursively report or suppress listener disposal. Successful callbacks report nothing. Handler failure must not be turned into a successful DOM/state transition by the adapter.

```can
// Current Can grammar: register this infallible handler, then click target "bad".
fn void failing_click
    emits []
    given
        browser::event e
    asserts
        sample: browser::event("click", "good", "", "") => ok
    match e.target is "bad"
        false => ok
        true => do
            int invalid = 1 / 0
            ok
```

The standard arithmetic failure from the click must produce one visible sanitized diagnostic with its Can source span, no unhandled rejection, then a later event still dispatches. Dispose/remount and timer failure receive the same checks. Startup faults should use the same browser diagnostic channel with a distinct `startup` phase.

## Delivery acceptance

The production build records its input compiler revision, Bun/bundler version, manifest hashes, bundle graph, generated JS/map/table hashes, asset routes and CSP. Inspect all reachable authored **and runtime** edges, the full source map (`sources`, `sourcesContent`, source paths), diagnostic table, JS strings and assets for forbidden modules/operations and known secret canaries. The server must verify and serve exactly the selected digest. A tampered file, missing map/table, unmanifested import, remote URL, or secret canary fails before publication. Tests fetch the actual route and assert content type, digest, CSP and no inline/eval execution.

Run an empty app and the invoice grid in the maintained build path in **Chromium 140.0.7339.186 and WebKit 26.0**, the two previously qualified versions; record current installed patch versions when rerun. Firefox remains unqualified until a complete named-version pass. For each named browser: check boot/offline boot, standard arithmetic/bounds/domain/native failure categories, structural equality, direct and transitive forbidden-capability rejection, source-span diagnostics, int64/finite float/option/variant/duplicate/unknown JSON parity with Bun, concurrent async callbacks, view disposal and stale events, cancellation timing, and admitted coordination (`all`, settled, first-success, first-completion) including a losing task holding a lease. Then execute the real invoice grid keyboard/edit/save/conflict/validation/denial/transport/reconnect/out-of-order/dispose matrix against the actual server and inspect DOM, focus, announcement, network bytes, server rows, listener/timer counts, and late diagnostics. Passing the former grid-only shim harness does not qualify this path.
