package emit

// Browser call/entry interfaces for UP13 (platform adapter) and UP15
// (bundling/publishing). UP11 owns the emitter side; UP13 implements
// runtime/platform/browser.ts against the call shapes below, and UP15 seals
// the diagnostic table and serves the bundle. Do not change these strings
// without coordinating both consumers.
//
// Entry (emit/browser.go):
//   - Browser main is checked zero-argument void with emits [] (see
//     check.CheckBrowserProgram). Bun main(str[] args) diagnoses separately.
//   - browser.ts exports BROWSER_PROFILE="browser-main" and an inert
//     $canBrowserMain(): Promise<void> that runs once (guarded by
//     $canStarted), initializes Can state inside the explicit owner root,
//     and reports startup faults once through the sealed reporter. It
//     auto-runs via `void $canBrowserMain();` on module load; no
//     test-authored boot invokes main.
//   - The entry calls runBrowserEntry from runtime/browser/entry.ts with a
//     placeholder sealed table ($canTable with empty source-index). UP15
//     replaces it with the checked table and publishes it as a verified
//     asset. The table shape is {index:{schemaVersion:1,
//     kind:"can.source-index",sources:[],modules:[]},maps:{}}.
//   - main is invoked as $canMain($canCtx) with explicit OwnerContext; no
//     ambient AsyncLocalStorage discovery occurs in browser production.
//
// Calls (RegionEmitter with Browser=true):
//   - Authored functions take (args..., $canCtx: $canOwnerContext,
//     $canContext?: $canAssertionContext). Entry passes only $canCtx;
//     $canContext stays undefined in production (no fixtures ship).
//   - Browser catalogue ops (can.std.browser@1::*, including per-type
//     specializations) take (args..., $canCtx) only. UP13 implements
//     $canBrowser.mount/root/openView/disposeView/disposeApp/createElement/
//     createText/setText/setAttribute/removeAttribute/appendChild/removeNode/
//     focus/onEvent/setTimeout/queryParameter/onCancelKey/onCancelEvent with
//     this trailing explicit context, plus the factory contracts below.
//   - Shared catalogue ops (non-browser, including actions/fetch) take
//     (args..., $canContext?) only, as before; browser callers pass the
//     (undefined) assertion context. UP09 owns those adapters.
//   - Dynamic callable values take (residual..., $canCtx, $canContext?) and
//     forward the appropriate subset to their target. ownCallable is invoked
//     with 5 arguments (site, target, captures, value, resourceIndices) in
//     browser; the 6th assertion-context argument is Bun-only.
//   - Coordination uses $canCoordinateSettleWithContext($canCtx, mode,
//     participants) with task-scoped runs run:($canCtx)=>...; no positions,
//     site, fixtures, callContext or callableInstance appear in browser.
//   - Array options carry owner:$canCtx alongside context:$canContext; the
//     collections runtime ignores owner until it forwards task context to
//     element callbacks (UP13 or later).
//   - Browser authored/state modules import no /assert/* edges. ScopeRequest
//     fails closed in browser production.
//
// Factory contracts (emit/runtime_browser.go, consumed by UP13):
//   - $canCreateBrowser($canDomain, {missingRoot, disposed, rejected, event,
//     invalidQuery, some, none}) with build-sealed identities. invalidQuery
//     is the UP11 addition for query_parameter; UP13 threads it into the
//     adapter together with the UP13 some/none option leaves sealed from
//     the query_parameter intrinsic result (option::value<str>).
//   - Method names are queryParameter, onCancelKey, onCancelEvent (camelCase
//     of the catalogue operations). Cancel methods preventDefault
//     synchronously for matching cancelable events before dispatching one
//     immutable snapshot; query budgets and duplicate/malformed rejection
//     follow the selected behavior.
//
// Pruning (emit/browser_prune.go):
//   - Browser production emits only the entry closure (functions, callbacks,
//     concrete specializations, required initializers). Bun keeps all.
//   - $canActions metadata still lists all declared actions (no edges);
//     browser imports $canActionRoutes only for client url() building, never
//     for mount()/mountForm() (server-only, rejected by the capability gate).
const (
	// BrowserProfileMarker is the BROWSER_PROFILE export value.
	BrowserProfileMarker = "browser-main"
	// BrowserEntryModule is the compiler-owned startup path.
	BrowserEntryModule = "browser.ts"
	// BrowserQueryMethod is the $canBrowser binding for query_parameter.
	BrowserQueryMethod = "$canBrowser.queryParameter"
	// BrowserCancelKeyMethod is the $canBrowser binding for on_cancel_key.
	BrowserCancelKeyMethod = "$canBrowser.onCancelKey"
	// BrowserCancelEventMethod is the $canBrowser binding for on_cancel_event.
	BrowserCancelEventMethod = "$canBrowser.onCancelEvent"
)
