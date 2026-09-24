# Independent platform/full-stack language review — 961f921

Scope: current source, examples, checker/emitter/runtime and current tests. No prior review, recommendation program, design-preparation alternatives, consultation responses, reconciliations, or disposition documents were opened. Some current integration tests contain comments categorizing their own limitations; conclusions below are grounded in executable code and direct source behavior, not those classifications. No production files changed, no Jev consultation, no subagents.

## Recommendation

Can now expresses a meaningful SaaS backend and a real stateful Can-authored browser editor. It is not just a server language whose browser demo was rewritten in JS. I would use it experimentally for a bounded backend or SSR product, but would not broadly recommend it as the sole language for a general SaaS frontend/backend yet. The strongest remaining reasons are incomplete binding between shared action declarations and live authenticated handlers, browser runtime packaging that needs bespoke semantic shims, and a closed host API surface that cannot express ordinary browser requirements without compiler/runtime work. These are different issues and should not be hidden under a generic “frontend immature” label.

## What exists and works

- Bun server target and distinct `browser-main` target exist. `compiler/internal/emit/browser.go:15,70` emits a capability-gated root exporting `$canBrowserMain`; assertion roots execute under Bun and are excluded from browser assets. `compiler/internal/browser/browser.go:106-145` checks reachable AND other emitted functions, initializers and server-backed declarations. Worker targets are rejected (`tests/integration/browser_build_test.go:259`).
- `examples/invoice-grid` contains 1,086 model, 150 record and 1,034 web lines of actual Can. It models editable keyed rows, parsed/raw quantities, stale responses, optimistic revision, explicit replay, conflict adoption, focus and status announcements. `runtime/platform/browser.ts:443-507` supplies versioned state; view disposal aborts listeners, clears timers, invalidates state and detaches DOM. This is enough for useful interactive applications, although a low-level vocabulary.
- HTML is an opaque safe value, with escaped text and admitted tags/attributes/URLs; forged `html::safe`, raw attributes and scripts are checker-rejected (`compiler/internal/check/html_test.go:8`). Ordinary functions compose HTML nodes and fragments, demonstrated by invoice/form-validation/language-site renderers. No template/view/component language declaration was found. Browser `view` means lifetime scope, not a declarative component.
- Typed form rows exist: `form::rows<T>`, named collections/fields, keyed input names, explicit order, structural issues, retained raw values. `examples/invoice/src/render/render.can:16-47` derives field names from types; `compiler/internal/check/form_test.go:143,201` covers field/collection refactor diagnostics. This is substantially better than untyped arbitrary form dictionaries.
- Actions declare method, path captures, body, result variant and exhaustive status cases (`compiler/internal/check/actions.go:38-50,148-238`). JSON response admissibility is checked; browser fetch validates response representation/status and same-origin transport. Forms can mount through `http::serve_form_action` and map both structural 422 and domain outcomes.
- SQL is parameterized with parsed statement/cardinality/parameter coverage and runtime typed rows. PostgreSQL, SQLite and MySQL backends exist; transactions expose commit/rollback and uncertainty rather than pretending failed transport proves failed commit. Invoice and webhook examples model protected mutations and replay/outbox behavior in ordinary Can.
- Multiple dependency libraries with a package named `model` compose through aliases and explicit lineage identities. Alias edits/relocation preserve identities; duplicate error names remain owner-qualified. `compiler/package_instance_test.go:95,152`, `compiler/internal/resolve/instances_test.go:122`, `compiler/internal/project/error_identity_test.go:12` directly exercise this. Numeric global error allocation is not a current composition blocker. Lock edges and content are checked (`compiler/internal/project/instances_test.go:195`).

## Findings ranked by consequence

### P1 — Action contracts do not yet bind a realistic authenticated resource flow end to end

Evidence: invoice server declares GET `/invoices/{invoice_id}` at `examples/invoice/src/web/web.can:29` but mounts GET `/invoices/load` at line 817. The latter manually reads a query and cookie (`:482-519`). The browser consumes the declared captured path (`examples/invoice-grid/src/web/web.can:23,66`). Consequently, the current browser matrix's same-origin server rewrites captured GET to the query endpoint (`tests/integration/gate5_frontend_test.go:403,437-447`). The direct endpoint test at 989 expects the declared captured path to answer 404. This is not a theoretical route-refactoring concern.

Root contract pressure: `checkActionHandler` admits only path captures and body, requires an exact total result and does not bind request context/dependencies (`compiler/internal/check/actions.go:286-334`). The invoice save handler therefore opens/closes a pool per request and accepts a session-token field; the manual JSON route overwrites it with authoritative cookie data (`examples/invoice/src/web/web.can:40-48,475-479`). Browser declarations are duplicated with never-served stub handlers (`examples/invoice-grid/src/web/web.can:5-51`). Cross-project agreement currently comes from a repository-specific driver test (`compiler/internal/driver/invoice_grid_test.go:125`), not one imported contract in the application.

Minimal acceptance: define a shared save/load contract once; bind the server to one startup pool plus request-derived authenticated actor; compile the client against that shared contract; GET `/invoices/inv-1` must reach the checked live handler without origin rewriting. Renaming path, capture, request field, or result leaf must either rebuild both ends or produce a diagnostic. Client build must not import server credentials or require pretend handlers.

Preferred direction: separate transport contract from server binding, with typed context/dependency acquisition at mount. Alternative: make actions route-bound callables carrying closures; cost is harder client-only import/target partitioning. Counterargument: ordinary HTTP callbacks already implement secure flows and a proxy is valid deployment technology. True, so this is a blocker to the *typed full-stack promise*, not inability to build a backend at all.

### P1 — Browser-target output is a gated TS intermediate, not a self-contained usable browser runtime

TypeScript output requiring bundling is reasonable. Requiring app-specific implementations of runtime semantics is the concern. `compiler/internal/emit/browser.go:70-80` invokes diagnostics configuration; `runtime/diagnostics.ts:3-4,30-32` imports Node fs/util and synchronously reads diagnostics metadata. `runtime/owner.ts:3` uses Node AsyncLocalStorage and `runtime/data.ts:1` uses Node util. Current `tests/integration/browser/build-grid.mjs:1-29,62-146` supplies four Node shims (async_hooks/util/fs/crypto), bakes source spans, drops module maps and assumes no live owner execution context. Its generated app bootstrap is only three lines, so the app remains authored Can, but packaging is more than ordinary transpilation.

The current CLI guide only promises `browser.ts` plus an asset manifest (`docs/implementation/cli.md:15-22`); no maintained general production bundling procedure was found in the reviewed operational docs. Do not describe the test bundler as an emitted native browser runtime.

Acceptance: build a minimal Can browser app and the invoice grid, bundle using a documented supported command with no test-only plugins or semantic polyfills, boot and exercise errors/async operations in at least one browser. Audit runtime dependencies as well as authored emitted modules. Prefer browser-specific runtime implementations; a supported bundler that owns complete semantic adapters is an alternative. This is primarily runtime/distribution work, not a reason to redesign Can syntax.

### P1 for general frontend scope — Closed native surface blocks common browser requirements

The bounded DOM catalogue is useful but restrictive: tags/attributes/events are enumerated (`compiler/internal/check/browser.go:42-84`); events contain only kind, target id, value and key (`runtime/platform/browser.ts:244-250`). There is no checked state, modifier keys, selection/composition state, event cancellation, file list or pointer data in this event contract. There is no localStorage/IndexedDB/sessionStorage/history API in the reviewed browser catalogue/runtime. State is in-memory view-owned state. Server websocket support exists, but browser capability policy rejects websocket sessions (`compiler/internal/browser/browser.go:72`).

Concrete scenario: persist a half-completed invoice across reload, use native checked state for a checkbox, prevent a keyboard default, or integrate a file picker. The current Can event/capability surface cannot directly express these. Merely expanding the nine event names does not fix absent event payload/control semantics.

Ordinary library code cannot fill host access gaps because the catalogue is deliberately compiler-owned, with no project registration API (`compiler/internal/catalogue/catalogue.go:1-2,19` and its README). This combination is a platform design limit, not just missing convenience widgets. Preserve finite typed contracts, but provide an explicit reviewed host-adapter mechanism OR commit to a sufficiently broad maintained browser contract. An untyped eval escape hatch would undermine the contracts and is not needed.

Acceptance: an all-Can persistent draft example that retains raw invalid quantity text across refresh; checkboxes/radio/multiselect/file input and IME-aware edits; predictable default-prevention and modifier-key behavior; injected failures remain typed and lifecycle disposal works. Counterargument: a narrower forms-and-tables product may never need these; that supports a scoped recommendation, not a broad one.

### P2 — SQL contracts validate statements and representations, not schema correspondence; mutation return values are deliberately unavailable

`compiler/internal/check/sql_descriptors.go:50-110` resolves declared param/row records and checks field admissibility/parameter order, then statement structure. It does not consult database schema or prove SELECT projection correspondence. Existing test `compiler/internal/check/sql_test.go:84` accepts `SELECT id FROM accounts` with a wider declared row. Runtime row decoding and `sql::schema_mismatch` correctly guard that boundary; describe this as checked parameterization plus runtime row contract, not schema-derived static SQL safety.

`compiler/internal/sql/cardinality.go:29-55` requires SELECT plus dedicated LIMIT for any row-returning descriptor, and rejects RETURNING on mutations. Scenario: insert an identity-generated row and receive id/default timestamp atomically, or update with optimistic concurrency and return server-normalized values. Workarounds exist (app-generated identity, transaction plus SELECT), so this is a medium restriction rather than backend impossibility. Direct RETURNING support better matches native DB operations and avoids extra round trips. DDL/migrations are operator-owned (`examples/invoice/schema.sql:1-4`); that alone is tooling scope, not a language flaw.

Acceptance: mutation returning one/optional/many with finite cardinality and row decoding; duplicate/missing rows handled; schema/projection mismatch rejected by schema-aware tooling or surfaced as an explicit runtime contract with migration tests. Counterargument: explicit bounds and simple mutation forms reduce parser/runtime complexity. Valid, but not a reason to permanently reject native atomic operations.

### P2 — Composition exists, but UI work has high authoring cost and manual synchronization

The grid manually constructs DOM and binds handlers (`examples/invoice-grid/src/web/web.can:570-676`), disposes/rebuilds views for structural edits (`:684-718`), and separately updates status/totals on ordinary input (`:724-790`). HTML helpers similarly require many admitted constructors (`examples/form-validation/src/render/render.can:38-82`). This is not “components impossible”: named functions, typed records and opaque nodes do compose. The problem is that a normal resource editor needs much low-level view/model synchronization and repeated error plumbing.

Prefer a library-level typed component/render/update abstraction first, with keyed identity, focus/selection/IME preservation and disposal tests. Only add syntax if library expression proves insufficient. Acceptance: extract reusable text field and keyed-row editor; adding one field should not require duplicating multiple event/state/DOM paths; late response, row reorder and focus remain correct. Current complete lifecycle and explicit state are valuable foundations, not features to replace indiscriminately.

## Example inventory and coverage

All 19 current example directories were inventoried using `rg --files examples`, manifests/READMEs and source declaration scans. Exact source inventory is appended separately below. Coverage is intentionally not presented as every source line read:

- Deep selected-file/body coverage: invoice (records/model/render/web/manifest/schema), invoice-grid (all three files; model sampled transitions and raw/typed edits, web action/fetch/state/render/event/boot paths), form-validation (all render and model, web routes), account-search (model/render/web sections), dashboard (model and render/web sections), webhook (model and web transaction/request sections; README/schema), language-site (render/route structure).
- API/example inventory and selected bodies: cookies, crypto, files, markdown, mysql, process, sqlite, stream, utilities, websocket. Native-ai declarations and package split inspected, provider quality not evaluated. Gallery's 28 numbered lessons plus main enumerated by filename/declaration only; core language semantics are outside this lane.
- Browser JS files are harnesses. Grid's app behavior comes from Can-generated modules, whereas Playwright `grid.mjs` drives tests and `build-grid.mjs` supplies bootstrap/shims. Earlier account/search/dashboard form browser tests exercise SSR/HTMX, not compiler-emitted stateful Can client code.
- Operational-doc freshness warning: webhook README says “Can has no HTTP client”; current action-fetch/browser client implementation proves this statement too broad/stale. It was not used as evidence of absent HTTP support. Root README's platform/release assertions were not taken as proof of current coverage.

## Commands and outcomes

1. `git rev-parse --short HEAD` => 961f921. `git status --short` later showed only coordinator-owned untracked review evidence directory; this reviewer created no repo files.
2. `go test ./compiler/internal/check ./compiler/internal/emit ./compiler/internal/sql ./compiler -run 'Test(Browser|Action|Form|HTML|SQLDescriptor|InspectProject.*)' -count=1` => check PASS 46.132s; emit PASS 7.151s; compiler PASS 4.510s; SQL package had no matching tests (not counted as SQL evidence).
3. `bun test ./runtime/test/browser-dom.test.ts ./runtime/test/browser-assert.test.ts ./runtime/test/form-action.test.ts ./runtime/test/form-rows.test.ts ./runtime/test/action-routes.test.ts ./runtime/test/action-json.test.ts ./runtime/test/action-fetch.test.ts ./runtime/test/sql-descriptor.test.ts` => 73 pass, 0 fail, 1,119 expectations, 8 current files. Log `/tmp/can-platform-runtime-tests.log`. Initial unprefixed Bun filters traversed old `out/` copies; that run was discarded and exact `./` paths rerun.
4. `go test ./compiler/internal/sql ./compiler/internal/browser ./compiler/internal/driver -run 'Test(Descriptor|Browser|InvoiceGrid)' -count=1` => browser PASS 6.806s, driver PASS 8.136s; SQL again had no matching names. Log `/tmp/can-platform-go-tests.log`.
5. Additional correctly named descriptor/package-composition command and results appended after completion.
6. Read-only `ps` failed under sandbox; irrelevant to findings. No live production service, database deployment, or full Playwright matrix was run in this review lane. Existing integration test source was inspected; do not report its expected outcomes as newly executed results.

No authored runtime TypeScript was edited, so lint-fix/format were not invoked. Coordinator owns repository-wide runtime checks.

## Exact example file inventory

- `examples/account-search/README.md`
- `examples/account-search/assets/site.css`
- `examples/account-search/can.errors.json`
- `examples/account-search/can.project.json`
- `examples/account-search/src/model/model.can`
- `examples/account-search/src/records/records.can`
- `examples/account-search/src/render/render.can`
- `examples/account-search/src/web/web.can`
- `examples/cookies/can.errors.json`
- `examples/cookies/can.project.json`
- `examples/cookies/src/main.can`
- `examples/crypto/can.errors.json`
- `examples/crypto/can.project.json`
- `examples/crypto/src/main.can`
- `examples/dashboard/README.md`
- `examples/dashboard/assets/site.css`
- `examples/dashboard/can.errors.json`
- `examples/dashboard/can.project.json`
- `examples/dashboard/src/model/model.can`
- `examples/dashboard/src/records/records.can`
- `examples/dashboard/src/render/render.can`
- `examples/dashboard/src/web/web.can`
- `examples/files/can.errors.json`
- `examples/files/can.project.json`
- `examples/files/src/main.can`
- `examples/form-validation/README.md`
- `examples/form-validation/assets/site.css`
- `examples/form-validation/can.errors.json`
- `examples/form-validation/can.project.json`
- `examples/form-validation/src/model/model.can`
- `examples/form-validation/src/records/records.can`
- `examples/form-validation/src/render/render.can`
- `examples/form-validation/src/web/web.can`
- `examples/gallery/README.md`
- `examples/gallery/can.errors.json`
- `examples/gallery/can.project.json`
- `examples/gallery/src/01-hello-world.can`
- `examples/gallery/src/02-cli-argument.can`
- `examples/gallery/src/03-arithmetic.can`
- `examples/gallery/src/04-temperature.can`
- `examples/gallery/src/05-leap-year.can`
- `examples/gallery/src/06-fizzbuzz.can`
- `examples/gallery/src/07-fibonacci.can`
- `examples/gallery/src/08-factorial.can`
- `examples/gallery/src/09-gcd.can`
- `examples/gallery/src/10-prime-check.can`
- `examples/gallery/src/11-palindrome.can`
- `examples/gallery/src/12-word-character-count.can`
- `examples/gallery/src/13-boolean-match.can`
- `examples/gallery/src/14-record-variant-match.can`
- `examples/gallery/src/15-exhaustive-match.can`
- `examples/gallery/src/16-optional-values.can`
- `examples/gallery/src/17-named-error-recovery.can`
- `examples/gallery/src/18-relay.can`
- `examples/gallery/src/19-immutable-record-update.can`
- `examples/gallery/src/20-array-map.can`
- `examples/gallery/src/21-array-filter.can`
- `examples/gallery/src/22-array-fold.can`
- `examples/gallery/src/23-array-sort.can`
- `examples/gallery/src/24-map-word-frequencies.can`
- `examples/gallery/src/25-generic-function.can`
- `examples/gallery/src/26-callable-argument.can`
- `examples/gallery/src/27-recursion-vs-collection.can`
- `examples/gallery/src/28-attached-assertions.can`
- `examples/gallery/src/main.can`
- `examples/invoice/can.errors.json`
- `examples/invoice/can.project.json`
- `examples/invoice/schema.sql`
- `examples/invoice/src/model/model.can`
- `examples/invoice/src/records/records.can`
- `examples/invoice/src/render/render.can`
- `examples/invoice/src/web/web.can`
- `examples/invoice-grid/can.errors.json`
- `examples/invoice-grid/can.project.json`
- `examples/invoice-grid/src/model/model.can`
- `examples/invoice-grid/src/records/records.can`
- `examples/invoice-grid/src/web/web.can`
- `examples/language-site/README.md`
- `examples/language-site/assets/site.css`
- `examples/language-site/can.errors.json`
- `examples/language-site/can.project.json`
- `examples/language-site/src/site.can`
- `examples/markdown/can.errors.json`
- `examples/markdown/can.project.json`
- `examples/markdown/src/main.can`
- `examples/mysql/can.errors.json`
- `examples/mysql/can.project.json`
- `examples/mysql/src/main.can`
- `examples/native-ai/README.md`
- `examples/native-ai/can.errors.json`
- `examples/native-ai/can.project.json`
- `examples/native-ai/src/app/main.can`
- `examples/native-ai/src/model/model.can`
- `examples/native-ai/src/oracles/fixtures/draft.json`
- `examples/native-ai/src/oracles/fixtures/review.json`
- `examples/native-ai/src/oracles/oracles.can`
- `examples/native-ai/src/records/records.can`
- `examples/process/can.errors.json`
- `examples/process/can.project.json`
- `examples/process/src/main.can`
- `examples/sqlite/can.errors.json`
- `examples/sqlite/can.project.json`
- `examples/sqlite/src/main.can`
- `examples/stream/can.errors.json`
- `examples/stream/can.project.json`
- `examples/stream/src/main.can`
- `examples/utilities/can.errors.json`
- `examples/utilities/can.project.json`
- `examples/utilities/src/main.can`
- `examples/webhook/README.md`
- `examples/webhook/can.errors.json`
- `examples/webhook/can.project.json`
- `examples/webhook/schema.sql`
- `examples/webhook/src/model/model.can`
- `examples/webhook/src/records/records.can`
- `examples/webhook/src/web/web.can`
- `examples/websocket/can.errors.json`
- `examples/websocket/can.project.json`
- `examples/websocket/src/main.can`

## Remaining test result and exact implementation coverage

`go test ./compiler/internal/sql ./compiler/internal/project ./compiler/internal/resolve -run 'Test(CheckDescriptor|.*Lineage|.*Lock|QualifiedImports|FormerNumericCollision)' -count=1` => all PASS: SQL 0.217s, project 0.180s, resolve 0.140s. `/tmp/can-platform-composition-tests.log`.

Implementation/test files read fully or in explicitly targeted ranges/searches: `AGENTS.md`, `README.md`, `compiler/README.md`, `package.json`, `compiler/current_types.go`, `compiler/package_instance_test.go`, `compiler/internal/check/browser.go`, `compiler/internal/check/templates.go` (fixture templates, not HTML), `compiler/internal/check/actions.go`, `compiler/internal/check/actions_json_test.go`, `compiler/internal/check/actions_test.go` (test inventory), `compiler/internal/check/form.go` (mount checks), `compiler/internal/check/form_test.go`, `compiler/internal/check/html_test.go`, `compiler/internal/check/sql.go`, `compiler/internal/check/sql_descriptors.go`, `compiler/internal/check/sql_test.go`, `compiler/internal/emit/browser.go`, `compiler/internal/emit/actions.go`, `compiler/internal/browser/browser.go`, `compiler/internal/browser/browser_test.go` (test inventory), `compiler/internal/browser/browser_catalogue_test.go` (test inventory), `compiler/internal/driver/invoice_grid_test.go`, `compiler/internal/sql/descriptors.go`, `compiler/internal/sql/descriptors_test.go`, `compiler/internal/sql/cardinality.go`, `compiler/internal/sql/parameters.go`, `compiler/internal/sql/parser.go`/`postgres.go`/`mysql.go`/`sqlite.go` (targeted RETURNING/schema search), `compiler/internal/project/instances_test.go`, `compiler/internal/project/error_identity_test.go`, `compiler/internal/resolve/instances_test.go`, `compiler/internal/resolve/symbols.go`/`native.go`/`native_test.go` (native/import inventory search), `compiler/internal/catalogue/catalogue.go`, `compiler/internal/catalogue/catalogue.json` (browser/native operations inventory), `compiler/internal/catalogue/README.md`, `compiler/internal/syntax/declarations.go`, `runtime/platform/browser.ts`, `runtime/platform/action-routes.ts`, `runtime/platform/sql/values.ts`, `runtime/platform/sql/descriptor.ts` (targeted codec search), `runtime/diagnostics.ts`, `runtime/owner.ts`, `runtime/data.ts`, `runtime/test/browser-dom.test.ts`, `tests/integration/browser_build_test.go`, `tests/integration/invoice_grid_test.go`, `tests/integration/browser/build-grid.mjs`, `tests/integration/browser/grid.mjs` (targeted wire/route search), `tests/integration/gate5_frontend_test.go` (build, origin rewrite, direct-route and mutation test source; no saved acceptance report), `docs/implementation/cli.md` (browser workflow only).

The implementation tests listed in command 3 were executed, but not all were read line by line. Broad file inventories (runtime modules, compiler check/emit, integration/browser test paths) were searched to establish presence/absence and locate primary implementation. No saved prior verdict or task report was used.
