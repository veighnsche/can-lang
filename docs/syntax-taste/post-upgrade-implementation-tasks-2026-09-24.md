# Ordered implementation tasks and concurrent lanes

24–25 September 2026 · execution companion to the [implementation plan](post-upgrade-implementation-plan-2026-09-24.md)

All 27 tasks are **not started**. IDs are new to this upgrade; the historical
T01–T27 list remains an execution record. The numerical order below is a valid
serial order. Dependencies include semantic prerequisites and explicit shared-file
handoffs; finish and integrate them before starting the dependent task.
The [selected behavior](post-upgrade-selected-behavior-2026-09-24.md) and
[invoice gate](post-upgrade-invoice-acceptance-2026-09-24.md) define correctness.

## Lanes and write ownership

Use three coding workers, **C**, **R**, **B**, and coordinator **I**. Each coding
worker operates in its own checkout on a `codex/` branch; the coordinator owns
integration. A task's write set includes its focused tests. A source directory
listed below is a navigation aid, not permission to edit another lane's files.
Proposed new file names may change within the declared boundary.

| Lane | Responsibility | Exclusive boundaries and handoff |
| --- | --- | --- |
| C · compiler | Types, generic proof/specialization, action grammar/checking/lowering, browser generated calls and entry. Later contract mutation tests and docs. | Own common `compiler/internal/{syntax,resolve,types,check,ir,emit}` throughout UP02/05/08/11/14/17. Browser lane owns `compiler/internal/browser` and build driver. UP18 receives the narrow emitter/asset handoff after UP17. |
| R · server/application | Regex, request lifecycle, native route/form/JSON dispatch, HTML guard, invoice contract/schema/domain/server. Later live HTTP/DB tests. | Own `runtime/platform/{http,router,server,action-routes,action-json,form,html}.ts` through UP12, then hand server/asset integration to B for UP18. Own invoice shared dependency and server files; B consumes the shared contract after UP16 and owns only the grid. |
| B · browser/delivery | Explicit runtime context/profile, graph audit, browser operations, bundle/manifest/publication and grid. Later browser tests. | Own browser runtime/profile modules and `runtime/platform/browser.ts`, `compiler/internal/browser`, build-driver/browser tooling and grid source. No common compiler edits before the explicit UP18 handoff. |
| I · coordinator | Snapshot, interface records, merge order, generated files, full verification, installed qualification and independent audit. | Serialize catalogue generation, `runtime/modules.json`, lock publication, shared fixture integration and CI changes. Hold worker changes to shared outputs until their authored inputs are integrated. UP24/25/27 are exclusive integration gates. |

Default dispatch is Sol High for cross-file implementation and Sol Medium for
bounded tests/docs. Use Astra High for UP04/07 (ownership/profile), UP14
(generic soundness), UP16 (authorization/replay), UP18 (verified publication)
and the final independent review. A reviewer receives requirements and raw
evidence, not the implementer's verdict as an instruction.

Catalogue source changes belong to C in UP05/08/11. The coordinator generates
and checks all mirrors after each handoff; no worker hand-edits
`runtime/catalogue.ts`, generated Go/catalogue references or the pinned HTMX
asset. Module inventory updates follow authored runtime imports. Runtime
formatting runs only in the worker's isolated checkout, with unrelated changes
removed before handoff.

## Dependency ledger

The write-set labels make same-wave collisions checkable. They are expanded
into concrete paths in the task details. No two tasks in a wave may claim the
same label. Shared generated outputs remain coordinator-owned outside worker
waves and are not a concurrent write set.

| ID | Lane | Depends on | Write sets | Deliverable |
| --- | --- | --- | --- | --- |
| UP01 | I | — | integration | Record baseline, acceptance registry, internal interfaces and ownership. |
| UP02 | C | UP01 | compiler | Separate symbolic proof types from emitted concrete models. |
| UP03 | R | UP01 | regex | Correct Unicode empty-match advancement with capped native iteration. |
| UP04 | B | UP01 | browser-core | Establish explicit async owner-context runtime and lease semantics. |
| UP05 | C | UP02 | compiler, catalogue | Implement handler-free action declarations and canonical metadata. |
| UP06 | R | UP04 | server-runtime | Revoke request capabilities after body cleanup and owner drainage. |
| UP07 | B | UP02, UP04 | browser-core | Build the sealed portable browser runtime profile. |
| UP08 | C | UP05, UP07 | compiler, catalogue | Check and lower action mount/URL/request/form bindings. |
| UP09 | R | UP06, UP08 | server-runtime | Implement native captured-route JSON/form dispatch. |
| UP10 | B | UP05, UP07 | browser-audit | Replace substring scans with complete structural graph/asset audits. |
| UP11 | C | UP02, UP04, UP07, UP08, UP10 | compiler, catalogue | Emit reachable browser code with explicit context and compiler-owned entry. |
| UP12 | R | UP08, UP09 | server-html, other-apps | Implement checked HTMX status/task/target guards and migrate affected pages. |
| UP13 | B | UP08, UP11 | browser-core, browser-api | Implement query, cancel policies, callback reporting and action Fetch. |
| UP14 | C | UP02, UP11 | compiler | Validate public generic dependency components atomically. |
| UP15 | B | UP10, UP11, UP13 | build-driver, distribution | Produce served-ready browser JS/maps/diagnostics and manifest in verified builds. |
| UP16 | R | UP05, UP08, UP09, UP12 | invoice-contract, invoice-domain | Migrate shared invoice contract, schema, protected domain and replay ledger. |
| UP17 | C | UP14 | compiler, generic-tests | Complete concrete generic specialization, emission and execution proof. |
| UP18 | B | UP09, UP12, UP15, UP16, UP17 | build-driver, distribution, server-runtime, server-assets, compiler | Verify paired server/browser publication and seven-day asset retention. |
| UP19 | R | UP16, UP18 | invoice-server, invoice-render, invoice-domain | Bind live server handlers and serve both application pages. |
| UP20 | B | UP13, UP16, UP18 | invoice-grid | Migrate the Can grid and repair focus/notices/pending state. |
| UP21 | C | UP17, UP19, UP20 | contract-tests | Prove shared contract edits update both targets or diagnose disagreement. |
| UP22 | R | UP03, UP06, UP09, UP12, UP19 | server-tests | Qualify real HTTP, SQLite, auth, replay, statuses and lifecycle. |
| UP23 | B | UP03, UP10, UP13, UP17, UP18, UP19, UP20 | browser-tests | Qualify supported browser/runtime and complete invoice UI behavior. |
| UP24 | I | UP21, UP22, UP23 | integration | Integrate CI prerequisites and pass complete-tree/fresh-output verification. |
| UP25 | I | UP24 | integration | Run installed Linux and required-browser application qualification. |
| UP26 | C | UP25 | documentation | Update current language, CLI, examples and exact support claims. |
| UP27 | I | UP26 | integration | Independently audit all fixes, exclusions, tests and final evidence. |

## Conservative concurrent schedule

A wave is a dependency barrier, not a time estimate. Start earlier when all
dependencies are integrated and ownership allows it; do not wait for an
unrelated long-running lane. Empty cells reflect a real dependency bottleneck,
not a request to invent work. I remains available to integrate and review
during coding waves and takes an exclusive task only where shown.

| Wave | I | C | R | B |
| --- | --- | --- | --- | --- |
| 0 | UP01 | — | — | — |
| 1 | — | UP02 | UP03 | UP04 |
| 2 | — | UP05 | UP06 | UP07 |
| 3 | — | UP08 | — | UP10 |
| 4 | — | UP11 | UP09 | — |
| 5 | — | UP14 | UP12 | UP13 |
| 6 | — | UP17 | UP16 | UP15 |
| 7 | — | — | — | UP18 |
| 8 | — | — | UP19 | UP20 |
| 9 | — | UP21 | UP22 | UP23 |
| 10 | UP24 | — | — | — |
| 11 | UP25 | — | — | — |
| 12 | — | UP26 | — | — |
| 13 | UP27 | — | — | — |

The main chain is shared action/compiler plumbing → generated browser context
and APIs → bundling → server pairing → invoice migration → live qualification.
The generic proof and regex repair run beside it. UP18 deliberately waits for
the compiler lane's emitter handoff; UP21/22/23 use distinct test files.

## Task contracts

### UP01 — Record inputs and execution boundaries

Record source commit, preparation-document/evidence digests, existing dirty
changes, compiler/Bun/archive identities and available browser/DB/Linux test
prerequisites. Preserve pre-existing work; workers start from the same captured
integration baseline. Register named positive/negative cases for every Fix and
the invoice gate, with current failures/skips recorded separately from passes.
Inventory obsolete action/browser calls in source, tests, grammar, references
and maintained examples. Record internal handoffs: canonical action metadata,
mount callback arities, explicit execution-context parameter, browser runtime
imports/identity table, diagnostic record, manifest fields and server asset
selection. These constrain implementation details of the selected behavior;
they do not introduce new public semantics. Reserve files per this ledger.

**Done:** each worker can implement against named interfaces and evidence;
archives/engines that must be operated at UP24/25 are listed. No production
claim relies on a missing fixture, placeholder helper or skipped test.

### UP02 — Isolate symbolic proof graphs

Own `compiler/internal/types/specialize.go`, `check/program.go`, related model
construction, and `emit/{program_state,program_modules,regions}.go` as needed.
Keep declaration-only `types.Parameter` graphs out of runtime type declarations
without dropping required concrete nested types. Preserve hard rejection if
an opaque type actually reaches a runtime boundary. Add full-module regressions
to `check/exported_generics_test.go` and `emit/exported_generics_test.go` using
the saved six-fixture probe; public identity must check and emit before broader
symbolic calls are enabled. Run focused type/check/emit tests.

**Done:** assertion and production module emission both succeed for public
identity; concrete types/brands remain complete and no symbolic placeholder is
emitted. Hand the model boundary to UP07/11/14.

### UP03 — Repair native regex iteration

Own `runtime/text.ts` and `runtime/test/text.test.ts`. Replace manual
`lastIndex++` iteration with a fresh global native `RegExp` and capped
`String.prototype.matchAll`; do not materialize unbounded results. Exercise
`😀x` empty matches with `u`/`v` → `[0,2,3]`, without Unicode → `[0,1,2,3]`,
nonempty matches, unmatched captures as empty strings, repeated handle use,
zero/one/maximum caps and invalid pattern/flag/budget failures. Follow runtime
format/lint/check rules and run the focused text suite.

**Done:** observed native positions and Can projection agree; no new iteration
syntax, error or catalogue signature is introduced.

### UP04 — Establish explicit browser owner context

Own `runtime/owner.ts`, callable/coordination context adapters and proposed
portable profile modules, with focused owner tests. Define the explicit token
contract for generated calls without changing Bun's selected ownership or
drainage guarantees. Test interleaved awaits, nested callbacks, lease acquisition
after suspension, root settlement, late callback faults and disposal. Reuse
the browser-owner probe as a failing regression, then exercise the production
adapter in native browser tests; no synchronous ALS substitute.

**Done:** context/lease isolation passes runtime-level tests, Bun owner tests
remain correct, and UP11 has a stable generated-call interface. Generated Can
browser qualification remains explicitly pending UP11/23.

### UP05 — Shared action source and identity

Own `syntax/{native,format}.go`, action AST/IR, resolver/checker action files,
`emit/actions.go`, action metadata tests and authored catalogue inputs. Implement
the selected handler-free declarations, captures as a record, input-none/JSON/
form modes, limits, result leaves/status/swap metadata and locked declaration
identity. Remove required `handles`; reject it in the new grammar. Check shared
package action exports without importing executable handlers. Migrate grammar,
formatter and focused source fixtures; coordinator regenerates mirrors.

**Done:** the three selected declarations parse/check and expose canonical
metadata; wrong capture/field/case/body/limit and old syntax fail with spans.
Consumers receive the metadata interface used by UP08/09/10/16.

### UP06 — Request capability lifetime

Own `runtime/platform/{http,router,server}.ts` and focused request/server tests.
Separate body abandonment from capability revocation for buffered and lazy
requests. Materialize the handler response/upgrade, abandon unread body, drain
the per-request owner, then revoke in an outer `finally` before response return.
Cover handler fault, rejected route, upgrade, unread body and admitted child
work. Retained headers/body/snapshot/upgrade operations fail with the selected
resource-state failure only after the chosen lifetime ends.

**Done:** request-lifetime probe becomes a production regression; native server
and shutdown behavior preserves leases and closes once. Hand the dispatch
boundary to UP09; no simultaneous server/router edits with that task.

### UP07 — Seal the portable browser runtime profile

Own browser profile modules and portable portions of `runtime/{data,domain,
failure,completion,diagnostics,entry,callable,coordination}.ts`. Use UP04's
explicit context; seal concrete type/error identities and source locations at
build time. Remove reachable Node crypto/util/fs/async-hooks dependencies from
the browser production profile while preserving private Can value branding,
equality, int64/codecs, immutable completions and sanitized standard failures.
Do not admit arbitrary foreign objects/proxies by assuming they do not occur.
Assertion-only context remains outside the shipped profile.

**Done:** a complete module inventory and runtime exports are available to the
emitter/auditor; profile conformance tests and Bun regressions pass. Inventory
changes go through I; no grid-only shim supplies semantics.

### UP08 — Checked action consumers and lowering

Own `check/{actions,action_fetch,http,form}.go`, action IR, `emit/{actions,
form_action,fetch_action,runtime_bindings}.go` and catalogue source changes.
Implement symbol-based `mount/url/request/post`: exact request-first,
non-generic/non-variadic `emits []` handlers; typed `near` pool/origin captures;
distinct normal and structural-form renderers; checked form field/row names;
GET without body; JSON/client-only projection. Emit native callback and
metadata interfaces for R and B, including per-case HTML guard metadata.

**Done:** complete checked Can fixtures define their actual named helpers and
exercise binding shapes. Stale symbols, wrong captures/arity/errors/result,
missing services, JSON/HTML confusion and computed action names diagnose.
Focused checker/emitter tests pass; the app remains pending its real adapters.

### UP09 — Native captured routes and body adapters

Own `runtime/platform/{action-routes,action-json,router,server,http,form}.ts`.
Use native Bun routes and strict one-pass capture validation. Preserve method
selection, static precedence within a method, duplicate/ambiguity refusal,
reserved assets and lifecycle from UP06. Bind the exact checked callables;
bounded native form/JSON decoding uses shared codecs. Preserve 400/404/405
(with `Allow`)/413/415/structural-422/500 versus finite app statuses. Cover
canonical int64 endpoints and malformed/encoded separators/dot cases from the
guarded route probe, body budgets, callback-entry counts and renderer faults.

**Done:** full Can→emitter→adapter fixtures reach real HTTP handlers without
rewrites; invalid input never enters a protected handler. Run route, JSON,
form-action, request and server tests, including complete emitted fixtures.

### UP10 — Structural browser audit

Own `compiler/internal/browser/browser.go` and its tests, using the profile
inventory from UP07. Replace raw substring logic with structural resolution of
all generated/dependency/runtime imports and host operations. Provide pre-bundle
graph and post-bundle JS/map/table audit interfaces to UP15. Include static/
dynamic imports, callback/generic reachability, unknown edges, source maps,
diagnostic assets, secret canaries and actionable origin/span diagnostics.

**Done:** quoted `Bun.`, `node: introduction` and `require(` data passes, actual
forbidden operations fail, and runtime bodies cannot escape inspection. The
auditor cannot rely on a bundler silently removing forbidden reachable code.

### UP11 — Browser checking, reachability and generated execution

Own target-aware `check/program.go`/browser checks and common emitter files,
including `emit/{browser,runtime_browser,program_state,program_entry}.go` and
call/callable/coordination lowering. Add selected query/cancel/reporter catalogue
bindings through the coordinator. Thread UP04/07 context across every emitted
async edge; no runtime discovery of ambient browser ownership. Check browser
zero-argument `main` separately from Bun `main(str[] args)` and generate once-only
DOM-ready startup. Prune production emission by reached declarations/callbacks/
concrete specializations and required initialization; retain full source checks.

**Done:** generated Can fixtures preserve interleaved owners and failures,
target signatures diagnose correctly, shared unused form/server code adds no
browser runtime edge, and imported actions mount nothing. Export the concrete
call/entry interfaces to UP13/15. Bun emitter regressions remain covered.

### UP12 — Checked HTML status and target guards

Own `runtime/platform/html.ts`, proposed authored HTMX guard module, HTML tests
and affected non-invoice example render sources and
`compiler/testdata/current/assets/page.can`. Consume UP08 metadata. Set
global noSwap `[204,304,"4xx","5xx"]` with exact checked action exceptions;
before request validate the connected target, before response reject control
headers, before swap admit exactly one inner main task for the same node.
Cancel OOB/partial/extra/redirected tasks before any mutation. Report sanitized
missing-target/protocol occurrences with status and effect uncertainty intact.
Migrate intentional non-action 422 pages to checked local HTML action policy.

**Done:** pinned HTMX tests cover all five admitted statuses, both missing-target
timings, redirects/retargeting and OOB/partial content, including a committed
write followed by render/swap failure. Vendor bytes stay pinned; full invoice
DOM/SQL qualification follows UP19/22/23.

### UP13 — Browser query, events, diagnostics and Fetch

Own `runtime/platform/browser.ts`, browser-profile reporter/client modules and
focused browser DOM/wire tests. Implement strict literal-key/query budgets and
malformed/duplicate alias rejection; synchronous registration-time Enter/submit
cancellation with exactly one immutable snapshot; view disposal and ordinary
uncanceled events. Connect startup/event/timer failures to the sealed reporter
once, omit raw/native/secret data and permit subsequent events. Implement
native same-origin action Fetch with exact status/leaf/operation-ID checks,
GET no body and distinct transport/abort/codec/unexpected-status outcomes.

**Done:** runtime and generated-fixture positives/negatives pass, including
immediate `defaultPrevented`, unmatched/noncancelable events and late callbacks.
UP15 can bundle real operations without test-authored boot or semantics.

### UP14 — Public symbolic contracts and atomic recursive proof

Own `check/{exported_generics,specialize,program}.go` and related type/IR proof
code. Resolve public validated callees by canonical identity and substitute
opaque signature expressions without concrete specialization. Build the full
public dependency graph, check components with provisional internal contracts
and commit proofs only after every body succeeds. On every internal cyclic
edge allow only bare formals or fully closed types. Include self/mutual cycles,
permutations, duplication/drop, closed generic graphs and nested growth.

**Done:** public cross-package identity/helper/acyclic composite calls and
finite mutual cases pass; private symbolic calls, illegal arithmetic/codec/
owner operations, any growing edge and failed-component proof reuse reject
with call/declaration evidence. No `SpecializationKey` or emitted function is
created from symbolic calls. This is a soundness gate, not a recursion-depth promise.

### UP15 — Verified browser bundling

Own `compiler/internal/driver/browser.go`, build staging/source-map interfaces,
distribution build inputs and a maintained bundled build tool. Invoke native
`Bun.build({ target: "browser" })`; build from UP11/13 output and audit before
and after bundling. Emit final digest JS, map, immutable diagnostic table and
manifest covering **all** reachable runtime/published bytes. Include toolchain,
source and catalogue inputs; preserve supervised assertion verification and
atomic current selection. Ship the tool with the distribution.

**Done:** empty app and generated browser fixtures build without test imports
or node aliases; repeated clean builds are identical. Bundle/audit/assertion
failure, tampering, missing assets and unaccounted edges prevent publication.
Use driver/browser, verified-build and staged browser-build tests.

### UP16 — Shared invoice data, contract and protected operations

Own the new shared invoice dependency, both dependency edges/locks, server
`src/{model,records}`, `schema.sql`, SQL descriptors and seed/inspection data in
`tests/integration/testdata/invoice/driver.ts`. Use current lineage/lock format,
not obsolete UUID proposals from historical packets. Author the three exact
selected action declarations and transparent records once. Migrate existing
domain/schema to integer tenant/invoice keys and selected quantity/price/revision
wire shapes; preserve keyed row order and exact totals.

Implement complete session/origin/load/save helpers. Use real transactions for
live authorization/revision checks and JSON replay digest plus committed result.
Recheck auth before ledger disclosure, serialize cleanup, validate positive W,
test seven-day before/at/after expiry without sliding, and enforce never-reused
revisions/IDs. Share protected writes between HTML and JSON while preserving
their distinct replay and raw-form behavior.

**Done:** named helpers contain real Can/SQL bodies; transaction, revocation,
same/different payload replay, expiry and uncertain-commit tests inspect actual
SQLite state. Freeze the shared package snapshot before UP19/20; subsequent
edits update both locks through I, never per-lane copies.

### UP17 — Concrete generic emission and execution

Own remaining concrete specialization/emission fixes and
`tests/integration/generics_test.go`/focused generic fixtures. Check/emit/execute
public chains across locked packages, including `nested<int>` directly calling
`identity<box<int>>`; canonical instances deduplicate. Exercise complete
assertion and production module emission, strict TypeScript, exact completion
propagation, source maps and missing concrete target failure. Cover false
proof after failed SCC and owner factory/construction negatives.

**Done:** public generic composition succeeds through the staged supported
toolchain with an operated archive, not only checker or function-emitter tests.
An unavailable staged leg remains incomplete. Hand common emitter ownership to
UP18 after tests pass.

### UP18 — Server/browser pairing and durable asset publication

B receives exclusive `compiler/main.go`, driver build/report/output inputs,
narrow emitter asset/entry changes and `runtime/platform/{assets,server}.ts`.
Own `tests/integration/assets_test.go` migration at this handoff. Its existing
handwritten **server** launcher exercises lower-level emitted asset behavior;
retain useful tests with updated ABI/policy assumptions, but do not count that
launcher as supported browser startup or invoice pipeline qualification.
Implement `--browser-manifest`, bind exact browser build ID and shared locked
snapshot into server inputs/report, verify all hashes/references and prevent
verification/publication races. Publish the whole matched asset set atomically;
failed rebuilds preserve the prior set. Keep replaced digest URLs at least
seven days across restart, cleanup and concurrent builds. Pages receive the
report-selected URL under CSP; no author JS or harness page enters the graph.

**Done:** mismatch/tamper/missing-map/table/secret/changed-input tests fail before
publication; matched builds serve exact JS/map/table bytes. Test retention
before/at/after the bound and report/asset consistency, including restart.
Release server file ownership before UP19/22.

### UP19 — Real mounted invoice handlers and pages

Own `examples/invoice/src/web`, render source and necessary server domain
call-site changes. Startup opens one pool, validates public origin/W, captures
services in all real mounts and closes once on failure/shutdown after drainage.
Use request cookies for GET and both saves; POST enforces exact Origin before
lookup/writes. Remove hidden/body session tokens, per-request pool opens,
manual result/codec tables and query-route workaround. Serve the two acceptance
pages and three direct routes using UP18's verified assets.

**Done:** complete Can assertions/builds and live direct GET/POST/form handler
smoke use real SQLite. Page/asset binding may use the verified compiler-built
minimal browser fixture from UP18 while UP20 migrates the grid; that proves
only server publication wiring. All success/error fragments are application
renderers and no helper is a placeholder. Combined invoice page/grid smoke
waits for both UP19 and UP20 and is required by UP21/23. UP22 owns the full
server matrix after this source handoff.

### UP20 — Existing grid migration and UI corrections

Own `examples/invoice-grid/src/{web,model,records}` and its source assertions;
consume UP16's shared dependency without altering it independently. Replace
local actions/stubs/wire copies and `main(args)` with checked action requests
and query bootstrap. Keep attempt ID/submitted draft/acknowledged revision and
current draft distinct; re-read state after await, preserve new edits and make
uncertain replay/reread deliberate. Construct/attach the complete tree before
focus by stable row key. Persist blocked notice and pending state in immutable
state. Dispose view resources without erasing app-owned identified saves.

**Done:** Can checks/builds and focused transition/render tests pass, including
attached-tree focus intent, persistent notice and attempt-owned pending state.
No test supplies a bootstrap argument. Real keyboard/DOM/focus observations in
the combined application wait for UP19 and are required by UP23; a component
assertion is not credited as that browser qualification.

### UP21 — Shared-contract mutation and negative compiler suite

Own `compiler/internal/driver/invoice_grid_test.go` and a new integration
contract-edit suite, with a task-local browser observer such as
`tests/integration/browser/invoice-contract.mjs`; do not edit Gate 5 or shared
browser observers owned by B. For each isolated
route/capture/field/leaf/status/limit/body-mode edit, update the shared source
once and both locks, rebuild both targets and observe bytes/paths/cases/DB/DOM.
Check old path 404/new path 200 with real handler entries; field/tag changes
propagate or stale named uses diagnose. Positional construction is allowed
only when actual wire agreement updates. Wrong captures/signatures/services,
mirrored actions, server closure imports and mismatched manifests reject.

**Done:** every row of the invoice contract-edit table has a nonvacuous result
and saved source spans/artifact identities. Text comparisons of two copied
contracts and a browser-only build cannot satisfy the suite.

### UP22 — Live server/database/lifecycle matrix

Own `tests/integration/{invoice_test,gate3_matrix_test}.go`, server-only fixtures
and invoice DB inspector extensions; browser scripts remain B-owned. Reuse
existing scenarios with updated direct paths/statuses/schema. Exercise real
200/422/409/403/503, all adapter failures, exact-Origin admission, foreign/missing
nondisclosure, concurrent revision/membership, identical/different/revoked replay,
expiry/revision invariants, lost acknowledgement and post-commit renderer fault.
Inspect rows/results and count protected-handler entries. Verify one pool,
startup failure, shutdown/drainage and buffered/lazy token revocation.

**Done:** server cases execute real Can/HTTP/SQLite rather than supplied
completions; fault injection records whether a write committed. HTML fragment
bytes/status checks pass; browser DOM guard results are supplied by UP23.

### UP23 — Supported browser/runtime and invoice matrix

Own `tests/integration/{gate5_frontend_test,browser_build_test,invoice_grid_test}.go`,
browser harness/scripts (including `browser/assets.mjs`) and proposed conformance
fixtures. UP21's new contract-mutation observer remains C-owned. Remove obsolete
Gate 5 rewrite/static-edit expectations (UP21 supplies the replacement edits).
Run generated empty app and invoice against application-served pages/assets;
remove `build-grid.mjs`/boot/shims from every claimed supported path. Migrate
codec-vector harnesses too if their current bundler supplies semantic aliases.

Require named Chromium and WebKit for query edge cases, diagnostics/equality/
exact codecs, interleaved owners after await, listeners/timers/leases/disposal,
sync cancellation, all save outcomes and uncertainty. Observe actual focus,
live notices, pending state, late/disposed responses, network bytes and SQLite.
Test HTML guard on every admitted status with OOB/partial/control headers,
missing target before/during request and stable resource counts after remount.
Inspect CSP, final JS/map/table and the exact paired script URL.

**Done:** every required browser leg runs without test-supplied application or
runtime semantics; no invocation-only focus proof or unavailable-engine skip
counts. Preserve machine-readable reports and raw evidence for UP24/25.

### UP24 — Integrated repository and CI verification

I integrates all lanes, reconciles shared catalogue/module/lock/fixture inputs
and updates CI to operate both required browsers and required services. The
lower-level asset harness from UP18 remains separately labeled; supported
startup evidence comes from UP23/25. Remove
obsolete syntax, references and old success assertions; run the complete
verification recipe below and fresh emitted TypeScript. Check current maintained
examples and retained webhook/SQL/owner/fixture regressions. Catch accidental
browser admission of server dependencies through generic/callable edges.

**Done:** the complete tree is green with prerequisites operated, no claimed
gate hidden by skip, no stale generated artifact and no test-only delivery
dependency. Record this exact candidate commit/input snapshot for UP25.

### UP25 — Installed target qualification

Build/release/install the candidate into disposable roots using the repository's
pinned distribution and selected Linux target procedures. Run the full invoice
acceptance gate with required named browsers, direct application origin,
SQLite, build/manifest tampering and contract edits against that installed
toolchain. Record archive/image hashes, Bun revision, OS/architecture, installed
paths and actual/emulated host mode. Repeat same-input build and asset retention/
restart tests. No stale result from a prior candidate satisfies this task.

**Done:** installed server/browser evidence meets every invoice gate and support
claim; missing host/engine remains incomplete. This creates local qualification
artifacts, not a public release upload or deployment.

### UP26 — Current documentation and example instructions

Own current language/CLI/catalogue/runtime/browser and example docs plus the
review reconciliation status links. Describe supported commands, shared package
layout, real auth/pool binding, direct pages/routes, exact status/replay and
asset retention, generic bounds and named tested platforms. Include a reproducible
clean-build/run recipe from the installed distribution. Mark new behavior
shipped only when its required gate passed; preserve older probes/reports as
history. No new measured performance, token or portability claims are implied.

**Done:** all 11 Fix rows link current source and passing evidence; retained and
deferred boundaries remain explicit. Relative links and generated references
are checked and the example can be followed without test harness imports.

### UP27 — Independent completion audit

Give a fresh reviewer the selected contracts, candidate source and raw UP21–25
evidence. Audit all 11 fixes, every contract edit, ownership/security/profile
boundaries, generated artifacts and 19 exclusions. Verify the final documented
candidate matches the qualified code/asset inputs; any code change caused by
review reruns affected checks and requalifies changed artifacts. I records
completed task IDs, exact tests/versions, residual limits and final report links.

**Done:** no required task remains open and no unsupported behavior is described
as shipped. A failing gate reopens its task and dependents; it is not waived by
Jev advice or by narrowing the original acceptance.

## Verification recipe and handoff format

Each task uses focused existing tests plus the stated new regression cases.
Commands below are repository commands, not tests already run for this plan.
For authored runtime TypeScript, run after edits:

```sh
bun run lint:fix:runtime
bun run format:runtime
bun run check:runtime
```

Use `bun run lint:runtime --format=agent` for compact diagnostics and run the
relevant `bun test runtime/test/<suite>.test.ts`. Preserve intentional test
behavior; explain narrow suppressions. Go changes get `gofmt` and focused
package tests. At UP24, with pinned Bun/archive, PostgreSQL and browser engines
operated, run the existing full gates:

```sh
go vet ./...
go run ./compiler/internal/catalogue/cmd/cataloguegen --check
go run ./tools/modcheck
go run ./tools/gramcheck
go test -timeout=60m -count=1 -v ./...
bun test runtime/test/
bun run check:runtime
git diff --check
```

Also verify no unformatted Go files and execute the fresh emit/TypeScript path
in [the tsc workflow](../../.github/workflows/tsc.yml): `TestStdlibMaintained`
with `CAN_FRESH_EMIT_DIR`, then the pinned `tscheck` TypeScript compiler. Match
[the verifier](../../.github/workflows/verifier.yml) service setup and
[release qualification](../../.github/workflows/release-qualify.yml), extending
its current Chromium-only installation to required Chromium **and WebKit**.
`CAN_BUN_ARCHIVE`, `CAN_TEST_POSTGRES_URL`, `CAN_TSC` and evidence destinations
must point to operated prerequisites; a zero exit code with skipped required
legs is insufficient. Re-run broad tests after code changes, failures or a
newly discovered concern, not repeatedly without a reason.

Every handoff records task ID, integrated dependency IDs, branch/commit and
changed paths; the supplied interface/manifest identity; commands with pass,
fail or skip and evidence paths; unresolved risks; and shared generated-input
changes for I. No task receives completion credit for a placeholder handler,
test-provided runtime semantics or a probe that did not execute its production
path. The schedule/coverage checker under
[planning evidence](evidence/2026-09-24/implementation-plan/README.md) verifies
the dependency order, wave assignments, write-set separation and Fix coverage.
