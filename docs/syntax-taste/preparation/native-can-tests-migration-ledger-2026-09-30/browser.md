# Native Can migration ledger: browser

Status: **proposed replacements; implementation and parity evidence pending**.

[Overview](../../native-can-tests-migration-ledger-2026-09-30.md) · [Machine-readable rows](browser.json)

Every deletion condition is conjunctive with the overview's common gate. It covers **all** protected facets, variants and delegated oracles, even where a row's short replacement sentence mentions only its first facet. Row IDs name coverage obligations, not one-to-one implementation files.

## BROWSER-001

**TestApplicationsStaged** — [tests/integration/applications_test.go](/Users/vince/Projects/can-lang/tests/integration/applications_test.go:273)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-BROWSER-001`. Evidence: pending.

**Protects**

- Discover and stage account-search, form-validation, dashboard and native-ai; each attached Can assertion passes with at least one real-can evidence row.
- Build each once and require the compiled entry to print usage with no developer tools on PATH and a disposable snapshot input.

**Current observations:** assert report roots/evidence, one build ID, CLI usage output and process exit

**Variants:** form-validation account-search dashboard native-ai

**Environment/selection gates:** CAN_BUN_ARCHIVE

**Capabilities:** CAN BUILD PROC WORK SUITE

**Proposed Can replacement:** Can functions discover the four application fixtures, run their attached assertions, require at least one real-can evidence row per staged example, build each once, and assert CLI usage output under tool-isolated execution. Generic staging/build/process mechanics expose observations; Can owns fixture selection and verdicts.

**Old harness may be deleted when:** Delete this Go test only after each listed obligation has a mapped passing Can case on its required environments, independent observations and negative controls remain effective, and all callers/report consumers have switched.

**Notes:** Proposed only; retain all scenario and oracle decisions in ordinary Can functions. Browser driver/native adapter may expose generic mechanics and observations, never this case sequence or verdict.

## BROWSER-002

**TestApplicationsStagedForms** — [tests/integration/applications_test.go](/Users/vince/Projects/can-lang/tests/integration/applications_test.go:335)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-BROWSER-002`. Evidence: pending.

**Protects**

- Serve staged form-validation; GET signup has form controls and same-origin CSP.
- POST validation accepts names/tags, rejects empty name with 422, escapes hostile tags; unknown route is 404 and wrong-method POST is 405.

**Current observations:** HTTP status/body/headers from compiled application

**Variants:** GET signup valid/empty/hostile form unknown route wrong method

**Environment/selection gates:** CAN_BUN_ARCHIVE

**Capabilities:** CAN CHILD HTTP WIRE BUILD WORK

**Proposed Can replacement:** Can functions serve the staged form-validation application and assert GET signup HTML/CSP, valid and invalid POST outcomes, hostile text escaping, missing route 404 and wrong-method 405. Generic HTTP/process mechanics expose responses; Can owns inputs and expectations.

**Old harness may be deleted when:** Delete this Go test only after each listed obligation has a mapped passing Can case on its required environments, independent observations and negative controls remain effective, and all callers/report consumers have switched.

**Notes:** Proposed only; retain all scenario and oracle decisions in ordinary Can functions. Browser driver/native adapter may expose generic mechanics and observations, never this case sequence or verdict.

## BROWSER-003

**TestApplicationsNativeAIStubbed** — [tests/integration/applications_test.go](/Users/vince/Projects/can-lang/tests/integration/applications_test.go:529)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-BROWSER-003`. Evidence: pending.

**Protects**

- Run compiled native-ai CLI against a local provider stub; verify refused, malformed plan and malformed answers stop at the correct stage.
- Check provider call order and raw request envelopes without leaking source secrets.

**Current observations:** exit and separated streams; stub request count/order/body

**Variants:** refusal bad plan bad answers

**Environment/selection gates:** CAN_BUN_ARCHIVE

**Capabilities:** CAN PROC HTTP WIRE FAULT WORK

**Proposed Can replacement:** Author an ordinary Can suite case that stages the named fixtures, chooses the listed variants and calls generic build/process/browser/database mechanics; Can compares Run compiled native-ai CLI against a local provider stub; verify refused, malformed plan and malformed answers stop at the correct stage.; Check provider call order and raw request envelopes without leaking source secrets.

**Old harness may be deleted when:** Delete this Go test only after each listed obligation has a mapped passing Can case on its required environments, independent observations and negative controls remain effective, and all callers/report consumers have switched.

**Notes:** Proposed only; retain all scenario and oracle decisions in ordinary Can functions. Browser driver/native adapter may expose generic mechanics and observations, never this case sequence or verdict.

## BROWSER-004

**TestApplicationsLive** — [tests/integration/applications_test.go](/Users/vince/Projects/can-lang/tests/integration/applications_test.go:601)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-BROWSER-004`. Evidence: pending.

**Protects**

- Seed disposable PostgreSQL; exercise account search/validation, dashboard page/data and unknown/wrong routes.
- Verify hostile query handling, dropped-table failures and reseed; run triage through stubbed provider and compare call sequence.

**Current observations:** seed/setup and reseed row counts, HTTP status/body, triage process output, stub request ledger

**Variants:** account-search dashboard triage dropped-table and hostile input

**Environment/selection gates:** CAN_BUN_ARCHIVE CAN_TEST_POSTGRES_URL

**Capabilities:** CAN DB CHILD HTTP WIRE FAULT WORK

**Proposed Can replacement:** Author an ordinary Can suite case that stages the named fixtures, chooses the listed variants and calls generic build/process/browser/database mechanics; Can compares Seed disposable PostgreSQL; exercise account search/validation, dashboard page/data and unknown/wrong routes.; Verify hostile query handling, dropped-table failures and reseed; run triage through stubbed provider and compare call sequence.

**Old harness may be deleted when:** Delete this Go test only after each listed obligation has a mapped passing Can case on its required environments, independent observations and negative controls remain effective, and all callers/report consumers have switched.

**Notes:** Source applications_test.go:603-780. The seed driver reports counts (7 accounts, 3 dashboard, 2 triage); the Go body does not independently inspect database rows. Move fixture and oracle choices into Can functions.

## BROWSER-005

**TestApplicationsBrowser** — [tests/integration/applications_test.go](/Users/vince/Projects/can-lang/tests/integration/applications_test.go:788)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-BROWSER-005`. Evidence: pending.

**Protects**

- Run the form-validation Chromium browser leg unconditionally after the archive/Playwright gate; run account-search and dashboard legs only when CAN_TEST_POSTGRES_URL is set. Preserve the delegated DOM/network/security assertions from each script.
- For each executed leg require passed JSON report, exact check count (forms 8, accounts 9, dashboard 5), requests prefixed by its loopback base URL, and a nonempty screenshot.

**Current observations:** browser report pass/check count and request URLs, screenshot existence/size; detailed DOM and faults are observed in called scripts

**Variants:** Chromium forms always after tool gate Chromium accounts and dashboard only with disposable PostgreSQL

**Environment/selection gates:** CAN_BUN_ARCHIVE node and pinned Playwright Chromium installation CAN_TEST_POSTGRES_URL only for accounts/dashboard CAN_BROWSER_EVIDENCE_DIR optional artifact copy

**Capabilities:** CAN BROWSER CHILD HTTP DB WORK SUITE

**Proposed Can replacement:** Can functions select forms, accounts and dashboard application legs and assert their DOM/network results, exact check obligations, loopback-only requests and screenshot evidence. Generic browser/process/database interfaces provide observations and seed/teardown; no authored host script or hidden host verdict remains.

**Old harness may be deleted when:** Delete this Go test only after each listed obligation has a mapped passing Can case on its required environments, independent observations and negative controls remain effective, and all callers/report consumers have switched.

**Related source:** [tests/integration/browser/forms.mjs](/Users/vince/Projects/can-lang/tests/integration/browser/forms.mjs); [tests/integration/browser/accounts.mjs](/Users/vince/Projects/can-lang/tests/integration/browser/accounts.mjs); [tests/integration/browser/dashboard.mjs](/Users/vince/Projects/can-lang/tests/integration/browser/dashboard.mjs)

**Notes:** Source applications_test.go:788-887. The Go body seeds/tears down PostgreSQL for the latter two legs but does not independently query it after browser execution; absent CAN_TEST_POSTGRES_URL still runs forms and returns after logging the skipped live legs.

## BROWSER-006

**TestCurrentBundledAssets** — [tests/integration/assets_test.go](/Users/vince/Projects/can-lang/tests/integration/assets_test.go:22)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-BROWSER-006`. Evidence: pending.

**Protects**

- Run asset Can roots and production program; compare repeated and relocated build IDs and staged bytes to declared/pinned assets.
- Inspect asset manifest, generated boot/import wiring, optional strict TS, loopback service, and negative path/digest/asset cases.

**Current observations:** assert report, output/status, byte/hash/manifest comparison, emitted artifacts, HTTP and compiler rejection

**Variants:** normal/rebuild/relocated invalid asset declarations and paths

**Environment/selection gates:** CAN_BUN_ARCHIVE CAN_TSC optional

**Capabilities:** CAN BUILD FILES DIAG HTTP FAULT WORK

**Proposed Can replacement:** Author an ordinary Can suite case that stages the named fixtures, chooses the listed variants and calls generic build/process/browser/database mechanics; Can compares Run asset Can roots and production program; compare repeated and relocated build IDs and staged bytes to declared/pinned assets.; Inspect asset manifest, generated boot/import wiring, optional strict TS, loopback service, and negative path/digest/asset cases.

**Old harness may be deleted when:** Delete this Go test only after each listed obligation has a mapped passing Can case on its required environments, independent observations and negative controls remain effective, and all callers/report consumers have switched.

**Notes:** Proposed only; retain all scenario and oracle decisions in ordinary Can functions. Browser driver/native adapter may expose generic mechanics and observations, never this case sequence or verdict.

## BROWSER-007

**TestCurrentBrowserAssets** — [tests/integration/assets_test.go](/Users/vince/Projects/can-lang/tests/integration/assets_test.go:350)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-BROWSER-007`. Evidence: pending.

**Protects**

- Build and serve browser-target asset app; inspect generated boot/runtime and run real browser checks for local styles/scripts, CSP, form swaps and no external requests.

**Current observations:** build manifest and artifact bytes; served HTTP; DOM/network/page faults/screenshot

**Variants:** browser engine in assets.mjs

**Environment/selection gates:** CAN_BUN_ARCHIVE node/Playwright installed

**Capabilities:** CAN BUILD BROWSER HTTP CHILD WORK

**Proposed Can replacement:** Author an ordinary Can suite case that stages the named fixtures, chooses the listed variants and calls generic build/process/browser/database mechanics; Can compares Build and serve browser-target asset app; inspect generated boot/runtime and run real browser checks for local styles/scripts, CSP, form swaps and no external requests.

**Old harness may be deleted when:** Delete this Go test only after each listed obligation has a mapped passing Can case on its required environments, independent observations and negative controls remain effective, and all callers/report consumers have switched.

**Related source:** [tests/integration/browser/assets.mjs](/Users/vince/Projects/can-lang/tests/integration/browser/assets.mjs)

**Notes:** Proposed only; retain all scenario and oracle decisions in ordinary Can functions. Browser driver/native adapter may expose generic mechanics and observations, never this case sequence or verdict.

## BROWSER-008

**TestCurrentPairedAssets** — [tests/integration/assets_test.go](/Users/vince/Projects/can-lang/tests/integration/assets_test.go:565)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-BROWSER-008`. Evidence: pending.

**Protects**

- Pair browser assets into server selection; verify report/pairing record, byte integrity, rebuild identity and changed-browser or repaired-server identity.
- Reject tampered/stale/mismatched shared snapshots without moving production current; verify bounded collection leaves no durable bytes.

**Current observations:** manifest/file hashes and reports; HTTP; retained selection before/after negative builds; collection ledger

**Variants:** browser variant server repair vendor mutation stale/matched snapshot collection

**Environment/selection gates:** CAN_BUN_ARCHIVE

**Capabilities:** CAN BUILD FILES HTTP FAULT WORK

**Proposed Can replacement:** Author an ordinary Can suite case that stages the named fixtures, chooses the listed variants and calls generic build/process/browser/database mechanics; Can compares Pair browser assets into server selection; verify report/pairing record, byte integrity, rebuild identity and changed-browser or repaired-server identity.; Reject tampered/stale/mismatched shared snapshots without moving production current; verify bounded collection leaves no durable bytes.

**Old harness may be deleted when:** Delete this Go test only after each listed obligation has a mapped passing Can case on its required environments, independent observations and negative controls remain effective, and all callers/report consumers have switched.

**Notes:** Proposed only; retain all scenario and oracle decisions in ordinary Can functions. Browser driver/native adapter may expose generic mechanics and observations, never this case sequence or verdict.

## BROWSER-009

**TestBrowserBuildTarget** — [tests/integration/browser_build_test.go](/Users/vince/Projects/can-lang/tests/integration/browser_build_test.go:94)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-BROWSER-009`. Evidence: pending.

**Protects**

- Browser build emits isolated root/asset/manifest with no Bun supervisor, server-only import, test edge or node host token; Bun target still builds.
- Reject worker target, server-only/browser-unavailable capabilities and combined invalid flags; verify empty app, clean deterministic builds, sealed diagnostics and production output.
- Wrong assertion, tampered script or missing map must block publication and preserve current selection.

**Current observations:** compiler status/diagnostics; artifact graph, manifests, byte hashes and publication state

**Variants:** 12 named subtests: pure, bun, flags, worker, capability, empty, verified, deterministic, assertion failure, tamper, missing map, profile unavailable

**Environment/selection gates:** CAN_BUN_ARCHIVE

**Capabilities:** CAN BUILD DIAG FILES FAULT WORK

**Proposed Can replacement:** Author an ordinary Can suite case that stages the named fixtures, chooses the listed variants and calls generic build/process/browser/database mechanics; Can compares Browser build emits isolated root/asset/manifest with no Bun supervisor, server-only import, test edge or node host token; Bun target still builds.; Reject worker target, server-only/browser-unavailable capabilities and combined invalid flags; verify empty app, clean deterministic builds, sealed diagnostics and production output.; Wrong assertion, tampered script or missing map must block publication and preserve current selection.

**Old harness may be deleted when:** Delete this Go test only after each listed obligation has a mapped passing Can case on its required environments, independent observations and negative controls remain effective, and all callers/report consumers have switched.

**Notes:** Proposed only; retain all scenario and oracle decisions in ordinary Can functions. Browser driver/native adapter may expose generic mechanics and observations, never this case sequence or verdict.

## BROWSER-010

**TestBrowserWireCodecParity** — [tests/integration/browser_build_test.go](/Users/vince/Projects/can-lang/tests/integration/browser_build_test.go:786)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-BROWSER-010`. Evidence: pending.

**Protects**

- Compare runtime wire vectors under Bun, sealed browser bundle, Chromium and WebKit; Firefox remains optional.
- Audit all 14 overlay edges against compiler table, reject node imports/test shims, require JSON.rawJSON on required engines and exact result parity.

**Current observations:** independent Bun vector results, bundle bytes, browser executed results/version/UA

**Variants:** Chromium required WebKit required Firefox optional

**Environment/selection gates:** bun and node/Playwright installed CAN_BROWSER_EVIDENCE_DIR optional

**Capabilities:** BROWSER BUILD FILES NATIVE SUITE WORK

**Proposed Can replacement:** Author an ordinary Can suite case that stages the named fixtures, chooses the listed variants and calls generic build/process/browser/database mechanics; Can compares Compare runtime wire vectors under Bun, sealed browser bundle, Chromium and WebKit; Firefox remains optional.; Audit all 14 overlay edges against compiler table, reject node imports/test shims, require JSON.rawJSON on required engines and exact result parity.

**Old harness may be deleted when:** Delete this Go test only after each listed obligation has a mapped passing Can case on its required environments, independent observations and negative controls remain effective, and all callers/report consumers have switched.

**Related source:** [tests/integration/browser/build-vectors.mjs](/Users/vince/Projects/can-lang/tests/integration/browser/build-vectors.mjs); [tests/integration/browser/codec-vectors-entry.ts](/Users/vince/Projects/can-lang/tests/integration/browser/codec-vectors-entry.ts); [tests/integration/browser/codec-parity.mjs](/Users/vince/Projects/can-lang/tests/integration/browser/codec-parity.mjs); [runtime/test/browser-wire-vectors.ts](/Users/vince/Projects/can-lang/runtime/test/browser-wire-vectors.ts)

**Notes:** Proposed only; retain all scenario and oracle decisions in ordinary Can functions. Browser driver/native adapter may expose generic mechanics and observations, never this case sequence or verdict.

## BROWSER-011

**TestC02NativeMatrix** — [tests/integration/browser_controls_test.go](/Users/vince/Projects/can-lang/tests/integration/browser_controls_test.go:102)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-BROWSER-011`. Evidence: pending.

**Protects**

- Qualify static browser control facts independently of Can: checkbox IDL, dirty values, reset, selects, files, modifiers, composition and caret.
- Require 11 checks, no limitations, only index.html requests, screenshots on Chromium/WebKit/Firefox.

**Current observations:** native DOM properties/event traces, network ledger, page/console faults and screenshot

**Variants:** Chromium WebKit Firefox

**Environment/selection gates:** node/Playwright installed CAN_FIREFOX_WS optional

**Capabilities:** BROWSER NATIVE HTTP SUITE WORK

**Proposed Can replacement:** Author an ordinary Can suite case that stages the named fixtures, chooses the listed variants and calls generic build/process/browser/database mechanics; Can compares Qualify static browser control facts independently of Can: checkbox IDL, dirty values, reset, selects, files, modifiers, composition and caret.; Require 11 checks, no limitations, only index.html requests, screenshots on Chromium/WebKit/Firefox.

**Old harness may be deleted when:** Delete this Go test only after each listed obligation has a mapped passing Can case on its required environments, independent observations and negative controls remain effective, and all callers/report consumers have switched.

**Related source:** [tests/integration/browser/controls-native.mjs](/Users/vince/Projects/can-lang/tests/integration/browser/controls-native.mjs); [tests/integration/testdata/browser-controls-native/index.html](/Users/vince/Projects/can-lang/tests/integration/testdata/browser-controls-native/index.html)

**Notes:** Proposed only; retain all scenario and oracle decisions in ordinary Can functions. Browser driver/native adapter may expose generic mechanics and observations, never this case sequence or verdict.

## BROWSER-012

**TestC02ControlsMatrix** — [tests/integration/browser_controls_test.go](/Users/vince/Projects/can-lang/tests/integration/browser_controls_test.go:143)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-BROWSER-012`. Evidence: pending.

**Protects**

- Qualify emitted Can controls against native facts: browser/paired rebuild identity, no stray imports or API traffic, control snapshots and live calls.
- Require 17 checks and no limitations on three engines; independently verify seeded invoice rows unchanged and no replay rows.

**Current observations:** Can DOM verdict lines plus independent DOM state, network faults and database rows

**Variants:** Chromium WebKit Firefox control/action variants

**Environment/selection gates:** CAN_BUN_ARCHIVE node/Playwright installed CAN_FIREFOX_WS optional

**Capabilities:** CAN BROWSER DB BUILD NATIVE WORK

**Proposed Can replacement:** Author an ordinary Can suite case that stages the named fixtures, chooses the listed variants and calls generic build/process/browser/database mechanics; Can compares Qualify emitted Can controls against native facts: browser/paired rebuild identity, no stray imports or API traffic, control snapshots and live calls.; Require 17 checks and no limitations on three engines; independently verify seeded invoice rows unchanged and no replay rows.

**Old harness may be deleted when:** Delete this Go test only after each listed obligation has a mapped passing Can case on its required environments, independent observations and negative controls remain effective, and all callers/report consumers have switched.

**Related source:** [tests/integration/browser/controls.mjs](/Users/vince/Projects/can-lang/tests/integration/browser/controls.mjs); [tests/integration/testdata/invoice/driver.ts](/Users/vince/Projects/can-lang/tests/integration/testdata/invoice/driver.ts)

**Notes:** Proposed only; retain all scenario and oracle decisions in ordinary Can functions. Browser driver/native adapter may expose generic mechanics and observations, never this case sequence or verdict.

## BROWSER-013

**TestC06Positive** — [tests/integration/c06_test.go](/Users/vince/Projects/can-lang/tests/integration/c06_test.go:206)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-BROWSER-013`. Evidence: pending.

**Protects**

- Grid and compare projects assert with real-can evidence (currently 394 and 221 roots respectively).
- Compare browser build and server pairing each repeat with stable identity and no stray emit; inspect browser asset/import audit.

**Current observations:** assert reports, build IDs and pairing manifest

**Variants:** grid compare

**Environment/selection gates:** CAN_BUN_ARCHIVE

**Capabilities:** CAN BUILD WORK

**Proposed Can replacement:** Author an ordinary Can suite case that stages the named fixtures, chooses the listed variants and calls generic build/process/browser/database mechanics; Can compares Grid and compare projects assert with real-can evidence, rebuild/pair deterministically, and keep distinct paired identities.

**Old harness may be deleted when:** Delete this Go test only after each listed obligation has a mapped passing Can case on its required environments, independent observations and negative controls remain effective, and all callers/report consumers have switched.

**Notes:** Proposed only; retain all scenario and oracle decisions in ordinary Can functions. Browser driver/native adapter may expose generic mechanics and observations, never this case sequence or verdict.

## BROWSER-014

**TestC06NegativeAPIBreak** — [tests/integration/c06_test.go](/Users/vince/Projects/can-lang/tests/integration/c06_test.go:261)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-BROWSER-014`. Evidence: pending.

**Protects**

- Break vendored controls API in staged copy; both grid and compare check must fail with relevant located diagnostics.

**Current observations:** compiler rejection and diagnostic text per edited dependency

**Variants:** grid compare API break edits

**Environment/selection gates:** CAN_BUN_ARCHIVE

**Capabilities:** DIAG FAULT FILES WORK

**Proposed Can replacement:** Author an ordinary Can suite case that stages the named fixtures, chooses the listed variants and calls generic build/process/browser/database mechanics; Can compares Break vendored controls API in staged copy; both grid and compare check must fail with relevant located diagnostics.

**Old harness may be deleted when:** Delete this Go test only after each listed obligation has a mapped passing Can case on its required environments, independent observations and negative controls remain effective, and all callers/report consumers have switched.

**Notes:** Proposed only; retain all scenario and oracle decisions in ordinary Can functions. Browser driver/native adapter may expose generic mechanics and observations, never this case sequence or verdict.

## BROWSER-015

**TestC06CaptureEdits** — [tests/integration/c06_test.go](/Users/vince/Projects/can-lang/tests/integration/c06_test.go:297)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-BROWSER-015`. Evidence: pending.

**Protects**

- Apply five shared contract edits: route rename remains compatible and both server/grid production targets build; capture, wire shape, body mode and leaf edits must diagnose in both consumers.
- Require edited declaration names and located source clues in each rejection.

**Current observations:** compiler status/diagnostic and source spans for two targets; verified build results for compatible route

**Variants:** route compatible capture break wire arity break body mode break leaf break server/grid consumers

**Environment/selection gates:** CAN_BUN_ARCHIVE

**Capabilities:** CAN DIAG FAULT BUILD FILES WORK

**Proposed Can replacement:** Can functions apply the five shared contract edits and check the route-compatible production server/grid builds plus the four incompatible capture/wire/body/leaf diagnostic failures in both consumers, including declaration names and source clues. Generic source editing/build/diagnostic mechanics expose results; Can owns every edit and verdict.

**Old harness may be deleted when:** Delete this Go test only after each listed obligation has a mapped passing Can case on its required environments, independent observations and negative controls remain effective, and all callers/report consumers have switched.

**Notes:** Proposed only; retain all scenario and oracle decisions in ordinary Can functions. Browser driver/native adapter may expose generic mechanics and observations, never this case sequence or verdict.

## BROWSER-016

**TestC06ServedMatrix** — [tests/integration/c06_test.go](/Users/vince/Projects/can-lang/tests/integration/c06_test.go:529)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-BROWSER-016`. Evidence: pending.

**Protects**

- Server Can assertions with real evidence; deterministic compare/grid browser builds and paired IDs, distinct drift generations.
- Run compare, grid, rollout and rollback browser legs on all three required engines.

**Current observations:** assert reports/build manifests; browser reports, request/DOM observations, DB state

**Variants:** compare grid rollout rollback Chromium WebKit Firefox

**Environment/selection gates:** CAN_BUN_ARCHIVE node/Playwright installed CAN_FIREFOX_WS optional

**Capabilities:** CAN BUILD BROWSER CHILD DB SUITE WORK

**Proposed Can replacement:** Author an ordinary Can suite case that stages the named fixtures, chooses the listed variants and calls generic build/process/browser/database mechanics; Can compares Server Can assertions with real evidence; deterministic compare/grid browser builds and paired IDs, distinct drift generations.; Run compare, grid, rollout and rollback browser legs on all three required engines.

**Old harness may be deleted when:** Delete this Go test only after each listed obligation has a mapped passing Can case on its required environments, independent observations and negative controls remain effective, and all callers/report consumers have switched.

**Related source:** [tests/integration/browser/compare.mjs](/Users/vince/Projects/can-lang/tests/integration/browser/compare.mjs); [tests/integration/browser/w1-grid.mjs](/Users/vince/Projects/can-lang/tests/integration/browser/w1-grid.mjs); [tests/integration/browser/drift.mjs](/Users/vince/Projects/can-lang/tests/integration/browser/drift.mjs)

**Notes:** Proposed only; retain all scenario and oracle decisions in ordinary Can functions. Browser driver/native adapter may expose generic mechanics and observations, never this case sequence or verdict.

## BROWSER-017

**TestGate5ServedMatrix** — [tests/integration/gate5_frontend_test.go](/Users/vince/Projects/can-lang/tests/integration/gate5_frontend_test.go:1111)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-BROWSER-017`. Evidence: pending.

**Protects**

- Probe required Chromium/WebKit/Firefox before builds; stage invoice server, grid, conformance fixture and empty app with real Can assertions.
- Repeat browser and paired builds for identity, import isolation and no stray output; run grid/conformance/empty/invoice live matrix with DB, network, DOM, fault and evidence checks.
- Keep the recorded WebKit same-origin redirect interception limitation explicit; Chromium qualifies that branch.

**Current observations:** browser report completeness, UA/version, HTTP/network/DOM, independent DB rows, build IDs and screenshot evidence

**Variants:** grid conformance empty invoice Chromium/WebKit/Firefox redirect qualified on Chromium

**Environment/selection gates:** CAN_BUN_ARCHIVE node/Playwright installed CAN_FIREFOX_WS optional

**Capabilities:** CAN BUILD BROWSER DB CHILD HTTP WORK SUITE

**Proposed Can replacement:** Author an ordinary Can suite case that stages the named fixtures, chooses the listed variants and calls generic build/process/browser/database mechanics; Can compares Probe required Chromium/WebKit/Firefox before builds; stage invoice server, grid, conformance fixture and empty app with real Can assertions.; Repeat browser and paired builds for identity, import isolation and no stray output; run grid/conformance/empty/invoice live matrix with DB, network, DOM, fault and evidence checks.; Keep the recorded WebKit same-origin redirect interception limitation explicit; Chromium qualifies that branch.

**Old harness may be deleted when:** Delete this Go test only after each listed obligation has a mapped passing Can case on its required environments, independent observations and negative controls remain effective, and all callers/report consumers have switched.

**Related source:** [tests/integration/browser/grid.mjs](/Users/vince/Projects/can-lang/tests/integration/browser/grid.mjs); [tests/integration/browser/conformance.mjs](/Users/vince/Projects/can-lang/tests/integration/browser/conformance.mjs); [tests/integration/browser/empty.mjs](/Users/vince/Projects/can-lang/tests/integration/browser/empty.mjs); [tests/integration/browser/invoice.mjs](/Users/vince/Projects/can-lang/tests/integration/browser/invoice.mjs); [tests/integration/testdata/invoice/driver.ts](/Users/vince/Projects/can-lang/tests/integration/testdata/invoice/driver.ts)

**Notes:** Proposed only; retain all scenario and oracle decisions in ordinary Can functions. Browser driver/native adapter may expose generic mechanics and observations, never this case sequence or verdict.

## BROWSER-018

**TestGate5GridStatic** — [tests/integration/gate5_frontend_test.go](/Users/vince/Projects/can-lang/tests/integration/gate5_frontend_test.go:1242)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-BROWSER-018`. Evidence: pending.

**Protects**

- Reject invoice server argv entry for browser target at entry-shape gate.
- Scan grid Can sources for no set_timeout/setTimeout token so timer-free leak surface remains.

**Current observations:** compiler rejection and source scan

**Variants:** server entry grid source scan

**Environment/selection gates:** CAN_BUN_ARCHIVE

**Capabilities:** DIAG FILES CAN FAULT

**Proposed Can replacement:** Author an ordinary Can suite case that stages the named fixtures, chooses the listed variants and calls generic build/process/browser/database mechanics; Can compares Reject invoice server argv entry for browser target at entry-shape gate.; Scan grid Can sources for no set_timeout/setTimeout token so timer-free leak surface remains.

**Old harness may be deleted when:** Delete this Go test only after each listed obligation has a mapped passing Can case on its required environments, independent observations and negative controls remain effective, and all callers/report consumers have switched.

**Notes:** Proposed only; retain all scenario and oracle decisions in ordinary Can functions. Browser driver/native adapter may expose generic mechanics and observations, never this case sequence or verdict.

## BROWSER-019

**TestInvoiceContractRoute** — [tests/integration/invoice_contract_test.go](/Users/vince/Projects/can-lang/tests/integration/invoice_contract_test.go:481)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-BROWSER-019`. Evidence: pending.

**Protects**

- Rename load route in shared contract; paired build must bind edit.
- Old route returns 405, unknown route 404, new route loads revision/lines, malformed capture 400; browser load/save uses edited route and DB replay records correct rows.

**Current observations:** paired entry/HTTP status and body; browser request ledger/DOM; independent DB reads

**Variants:** route rename old/new/unknown/malformed path

**Environment/selection gates:** CAN_BUN_ARCHIVE

**Capabilities:** CAN BUILD HTTP BROWSER DB FILES WORK

**Proposed Can replacement:** Author an ordinary Can suite case that stages the named fixtures, chooses the listed variants and calls generic build/process/browser/database mechanics; Can compares Rename load route in shared contract; paired build must bind edit.; Old route returns 405, unknown route 404, new route loads revision/lines, malformed capture 400; browser load/save uses edited route and DB replay records correct rows.

**Old harness may be deleted when:** Delete this Go test only after each listed obligation has a mapped passing Can case on its required environments, independent observations and negative controls remain effective, and all callers/report consumers have switched.

**Related source:** [tests/integration/browser/invoice-contract.mjs](/Users/vince/Projects/can-lang/tests/integration/browser/invoice-contract.mjs); [tests/integration/testdata/invoice/driver.ts](/Users/vince/Projects/can-lang/tests/integration/testdata/invoice/driver.ts)

**Notes:** Proposed only; retain all scenario and oracle decisions in ordinary Can functions. Browser driver/native adapter may expose generic mechanics and observations, never this case sequence or verdict.

## BROWSER-020

**TestInvoiceContractCapture** — [tests/integration/invoice_contract_test.go](/Users/vince/Projects/can-lang/tests/integration/invoice_contract_test.go:597)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-BROWSER-020`. Evidence: pending.

**Protects**

- Rename named capture across contract and caller; paired entry and HTTP load/save use new capture, malformed input rejected.
- Browser saves with renamed capture and database/DOM remain consistent.

**Current observations:** source edits/build entry; HTTP; browser DOM; independent DB

**Variants:** capture rename valid/malformed

**Environment/selection gates:** CAN_BUN_ARCHIVE

**Capabilities:** CAN BUILD HTTP BROWSER DB WORK

**Proposed Can replacement:** Author an ordinary Can suite case that stages the named fixtures, chooses the listed variants and calls generic build/process/browser/database mechanics; Can compares Rename named capture across contract and caller; paired entry and HTTP load/save use new capture, malformed input rejected.; Browser saves with renamed capture and database/DOM remain consistent.

**Old harness may be deleted when:** Delete this Go test only after each listed obligation has a mapped passing Can case on its required environments, independent observations and negative controls remain effective, and all callers/report consumers have switched.

**Related source:** [tests/integration/browser/invoice-contract.mjs](/Users/vince/Projects/can-lang/tests/integration/browser/invoice-contract.mjs); [tests/integration/testdata/invoice/driver.ts](/Users/vince/Projects/can-lang/tests/integration/testdata/invoice/driver.ts)

**Notes:** Proposed only; retain all scenario and oracle decisions in ordinary Can functions. Browser driver/native adapter may expose generic mechanics and observations, never this case sequence or verdict.

## BROWSER-021

**TestInvoiceContractField** — [tests/integration/invoice_contract_test.go](/Users/vince/Projects/can-lang/tests/integration/invoice_contract_test.go:689)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-BROWSER-021`. Evidence: pending.

**Protects**

- Rename invoice field; new-field JSON save commits while old-field payload is rejected.
- Browser request must carry new field exactly and never old field; DOM and database agree.

**Current observations:** HTTP request/response bytes, browser network/DOM, independent DB rows

**Variants:** new field old field

**Environment/selection gates:** CAN_BUN_ARCHIVE

**Capabilities:** CAN BUILD HTTP BROWSER DB FAULT WORK

**Proposed Can replacement:** Author an ordinary Can suite case that stages the named fixtures, chooses the listed variants and calls generic build/process/browser/database mechanics; Can compares Rename invoice field; new-field JSON save commits while old-field payload is rejected.; Browser request must carry new field exactly and never old field; DOM and database agree.

**Old harness may be deleted when:** Delete this Go test only after each listed obligation has a mapped passing Can case on its required environments, independent observations and negative controls remain effective, and all callers/report consumers have switched.

**Related source:** [tests/integration/browser/invoice-contract.mjs](/Users/vince/Projects/can-lang/tests/integration/browser/invoice-contract.mjs); [tests/integration/testdata/invoice/driver.ts](/Users/vince/Projects/can-lang/tests/integration/testdata/invoice/driver.ts)

**Notes:** Proposed only; retain all scenario and oracle decisions in ordinary Can functions. Browser driver/native adapter may expose generic mechanics and observations, never this case sequence or verdict.

## BROWSER-022

**TestInvoiceContractLeaf** — [tests/integration/invoice_contract_test.go](/Users/vince/Projects/can-lang/tests/integration/invoice_contract_test.go:778)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-BROWSER-022`. Evidence: pending.

**Protects**

- Rename domain failure leaf; invalid save returns 422 with new tag, valid save still works.
- Browser renders renamed error; invalid and valid database/replay legs stay distinct.

**Current observations:** HTTP error payload; browser DOM/request ledger; independent DB records

**Variants:** renamed leaf invalid/valid save

**Environment/selection gates:** CAN_BUN_ARCHIVE

**Capabilities:** CAN BUILD HTTP BROWSER DB FAULT WORK

**Proposed Can replacement:** Author an ordinary Can suite case that stages the named fixtures, chooses the listed variants and calls generic build/process/browser/database mechanics; Can compares Rename domain failure leaf; invalid save returns 422 with new tag, valid save still works.; Browser renders renamed error; invalid and valid database/replay legs stay distinct.

**Old harness may be deleted when:** Delete this Go test only after each listed obligation has a mapped passing Can case on its required environments, independent observations and negative controls remain effective, and all callers/report consumers have switched.

**Related source:** [tests/integration/browser/invoice-contract.mjs](/Users/vince/Projects/can-lang/tests/integration/browser/invoice-contract.mjs); [tests/integration/testdata/invoice/driver.ts](/Users/vince/Projects/can-lang/tests/integration/testdata/invoice/driver.ts)

**Notes:** Proposed only; retain all scenario and oracle decisions in ordinary Can functions. Browser driver/native adapter may expose generic mechanics and observations, never this case sequence or verdict.

## BROWSER-023

**TestInvoiceContractStatus** — [tests/integration/invoice_contract_test.go](/Users/vince/Projects/can-lang/tests/integration/invoice_contract_test.go:884)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-BROWSER-023`. Evidence: pending.

**Protects**

- Move save success status to 201 through shared contract and relock; server and browser both observe 201.
- Browser DOM and committed database rows reflect one save; pristine contract base remains unchanged.

**Current observations:** lock/build identity, HTTP status, browser DOM/network, independent DB

**Variants:** status edit pristine vs edited

**Environment/selection gates:** CAN_BUN_ARCHIVE

**Capabilities:** CAN BUILD HTTP BROWSER DB FILES WORK

**Proposed Can replacement:** Author an ordinary Can suite case that stages the named fixtures, chooses the listed variants and calls generic build/process/browser/database mechanics; Can compares Move save success status to 201 through shared contract and relock; server and browser both observe 201.; Browser DOM and committed database rows reflect one save; pristine contract base remains unchanged.

**Old harness may be deleted when:** Delete this Go test only after each listed obligation has a mapped passing Can case on its required environments, independent observations and negative controls remain effective, and all callers/report consumers have switched.

**Related source:** [tests/integration/browser/invoice-contract.mjs](/Users/vince/Projects/can-lang/tests/integration/browser/invoice-contract.mjs); [tests/integration/testdata/invoice/driver.ts](/Users/vince/Projects/can-lang/tests/integration/testdata/invoice/driver.ts)

**Notes:** Proposed only; retain all scenario and oracle decisions in ordinary Can functions. Browser driver/native adapter may expose generic mechanics and observations, never this case sequence or verdict.

## BROWSER-024

**TestInvoiceContractLimit** — [tests/integration/invoice_contract_test.go](/Users/vince/Projects/can-lang/tests/integration/invoice_contract_test.go:971)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-BROWSER-024`. Evidence: pending.

**Protects**

- Lower body budget: over-limit HTTP request gets 413 and does not commit; browser blocks oversized save before sending.
- Under-limit request succeeds and commits exactly the shrunk line; byte boundary is measured.

**Current observations:** body byte count, HTTP, browser request absence/DOM, independent DB

**Variants:** over-limit browser budget under-limit

**Environment/selection gates:** CAN_BUN_ARCHIVE

**Capabilities:** CAN BUILD HTTP BROWSER DB FAULT WORK

**Proposed Can replacement:** Author an ordinary Can suite case that stages the named fixtures, chooses the listed variants and calls generic build/process/browser/database mechanics; Can compares Lower body budget: over-limit HTTP request gets 413 and does not commit; browser blocks oversized save before sending.; Under-limit request succeeds and commits exactly the shrunk line; byte boundary is measured.

**Old harness may be deleted when:** Delete this Go test only after each listed obligation has a mapped passing Can case on its required environments, independent observations and negative controls remain effective, and all callers/report consumers have switched.

**Related source:** [tests/integration/browser/invoice-contract.mjs](/Users/vince/Projects/can-lang/tests/integration/browser/invoice-contract.mjs); [tests/integration/testdata/invoice/driver.ts](/Users/vince/Projects/can-lang/tests/integration/testdata/invoice/driver.ts)

**Notes:** Proposed only; retain all scenario and oracle decisions in ordinary Can functions. Browser driver/native adapter may expose generic mechanics and observations, never this case sequence or verdict.

## BROWSER-025

**TestInvoiceContractBodyMode** — [tests/integration/invoice_contract_test.go](/Users/vince/Projects/can-lang/tests/integration/invoice_contract_test.go:1053)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-BROWSER-025`. Evidence: pending.

**Protects**

- Flip route body mode and relock: grid build must reject JSON-kind mismatch.
- Form route accepts form body; JSON-to-form and HTML-to-JSON get 415; untouched load and pristine JSON route continue working.

**Current observations:** compiler diagnostic; HTTP status/body and independent DB

**Variants:** form mode JSON mismatch HTML mismatch pristine

**Environment/selection gates:** CAN_BUN_ARCHIVE

**Capabilities:** CAN BUILD DIAG HTTP DB FAULT WORK

**Proposed Can replacement:** Author an ordinary Can suite case that stages the named fixtures, chooses the listed variants and calls generic build/process/browser/database mechanics; Can compares Flip route body mode and relock: grid build must reject JSON-kind mismatch.; Form route accepts form body; JSON-to-form and HTML-to-JSON get 415; untouched load and pristine JSON route continue working.

**Old harness may be deleted when:** Delete this Go test only after each listed obligation has a mapped passing Can case on its required environments, independent observations and negative controls remain effective, and all callers/report consumers have switched.

**Notes:** Proposed only; retain all scenario and oracle decisions in ordinary Can functions. Browser driver/native adapter may expose generic mechanics and observations, never this case sequence or verdict.

## BROWSER-026

**TestInvoiceContractManifestMismatch** — [tests/integration/invoice_contract_test.go](/Users/vince/Projects/can-lang/tests/integration/invoice_contract_test.go:1148)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-BROWSER-026`. Evidence: pending.

**Protects**

- Edited and pristine browser builds get distinct identities; pairing edited server with pristine or tampered browser manifest must be refused.
- A matched edited browser/server pair is the positive control and publishes a paired entry.

**Current observations:** build IDs, manifest integrity and compiler status/diagnostics for mismatched, tampered and matched pairs

**Variants:** pristine manifest tampered manifest

**Environment/selection gates:** CAN_BUN_ARCHIVE

**Capabilities:** BUILD FILES DIAG FAULT WORK

**Proposed Can replacement:** Can functions stage edited and pristine browser builds, require distinct identities, reject mismatched and tampered manifest pairing, and accept the matched edited browser/server pair as a positive control. Generic build/manifest mechanics expose observations; Can owns pairing choices and expected outcomes.

**Old harness may be deleted when:** Delete this Go test only after each listed obligation has a mapped passing Can case on its required environments, independent observations and negative controls remain effective, and all callers/report consumers have switched.

**Notes:** Proposed only; retain all scenario and oracle decisions in ordinary Can functions. Browser driver/native adapter may expose generic mechanics and observations, never this case sequence or verdict.

## BROWSER-027

**TestInvoiceContractNoRegistration** — [tests/integration/invoice_contract_test.go](/Users/vince/Projects/can-lang/tests/integration/invoice_contract_test.go:1211)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-BROWSER-027`. Evidence: pending.

**Protects**

- Unregister routes in staged server; load, save and form save return 404 without recording posts.
- Health and grid shell remain available so refusal is route-specific.

**Current observations:** HTTP status/body, independent DB row/replay count

**Variants:** load JSON save form save health/grid shell

**Environment/selection gates:** CAN_BUN_ARCHIVE

**Capabilities:** CAN BUILD HTTP DB FAULT WORK

**Proposed Can replacement:** Author an ordinary Can suite case that stages the named fixtures, chooses the listed variants and calls generic build/process/browser/database mechanics; Can compares Unregister routes in staged server; load, save and form save return 404 without recording posts.; Health and grid shell remain available so refusal is route-specific.

**Old harness may be deleted when:** Delete this Go test only after each listed obligation has a mapped passing Can case on its required environments, independent observations and negative controls remain effective, and all callers/report consumers have switched.

**Related source:** [tests/integration/browser/invoice-contract.mjs](/Users/vince/Projects/can-lang/tests/integration/browser/invoice-contract.mjs); [tests/integration/testdata/invoice/driver.ts](/Users/vince/Projects/can-lang/tests/integration/testdata/invoice/driver.ts)

**Notes:** Proposed only; retain all scenario and oracle decisions in ordinary Can functions. Browser driver/native adapter may expose generic mechanics and observations, never this case sequence or verdict.

## BROWSER-028

**TestInvoiceGridStagedBrowserBuild** — [tests/integration/invoice_grid_test.go](/Users/vince/Projects/can-lang/tests/integration/invoice_grid_test.go:39)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-BROWSER-028`. Evidence: pending.

**Protects**

- Stage invoice-grid browser build; all attached Can assertions pass with real evidence.
- Browser report and asset identity/content shape are valid.

**Current observations:** build report, assertion evidence, browser asset manifest/bytes

**Variants:** invoice-grid

**Environment/selection gates:** CAN_BUN_ARCHIVE

**Capabilities:** CAN BUILD FILES WORK

**Proposed Can replacement:** Author an ordinary Can suite case that stages the named fixtures, chooses the listed variants and calls generic build/process/browser/database mechanics; Can compares Stage invoice-grid browser build; all attached Can assertions pass with real evidence.; Browser report and asset identity/content shape are valid.

**Old harness may be deleted when:** Delete this Go test only after each listed obligation has a mapped passing Can case on its required environments, independent observations and negative controls remain effective, and all callers/report consumers have switched.

**Notes:** Proposed only; retain all scenario and oracle decisions in ordinary Can functions. Browser driver/native adapter may expose generic mechanics and observations, never this case sequence or verdict.

## BROWSER-029

**TestInvoiceFormLive** — [tests/integration/invoice_test.go](/Users/vince/Projects/can-lang/tests/integration/invoice_test.go:521)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-BROWSER-029`. Evidence: pending.

**Protects**

- Serve invoice form and grid with seeded SQLite; verify tenant/session/CSRF/origin/method denials and hostile escaping.
- Exercise form and JSON saves, validation/stale/structural/replay/conflict/status paths; inspect independent invoice/replay rows and retention effects.
- Check shell/CSP/headers/body, exact amounts and row order, failures and untouched state after rejection.

**Current observations:** HTTP status/headers/body, independent SQLite rows/replay, source assertions and process events

**Variants:** GET/form/JSON valid/stale/invalid/hostile/forbidden replay and retention fault injection

**Environment/selection gates:** CAN_BUN_ARCHIVE

**Capabilities:** CAN BUILD HTTP DB CHILD WIRE FAULT WORK

**Proposed Can replacement:** Author an ordinary Can suite case that stages the named fixtures, chooses the listed variants and calls generic build/process/browser/database mechanics; Can compares Serve invoice form and grid with seeded SQLite; verify tenant/session/CSRF/origin/method denials and hostile escaping.; Exercise form and JSON saves, validation/stale/structural/replay/conflict/status paths; inspect independent invoice/replay rows and retention effects.; Check shell/CSP/headers/body, exact amounts and row order, failures and untouched state after rejection.

**Old harness may be deleted when:** Delete this Go test only after each listed obligation has a mapped passing Can case on its required environments, independent observations and negative controls remain effective, and all callers/report consumers have switched.

**Related source:** [tests/integration/testdata/invoice/driver.ts](/Users/vince/Projects/can-lang/tests/integration/testdata/invoice/driver.ts)

**Notes:** Proposed only; retain all scenario and oracle decisions in ordinary Can functions. Browser driver/native adapter may expose generic mechanics and observations, never this case sequence or verdict.

## BROWSER-030

**TestInvoiceStartupRefusal** — [tests/integration/invoice_test.go](/Users/vince/Projects/can-lang/tests/integration/invoice_test.go:982)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-BROWSER-030`. Evidence: pending.

**Protects**

- Server must exit before listening for missing/invalid PUBLIC_ORIGIN or nonpositive/malformed replay window.

**Current observations:** process exit and readiness failure

**Variants:** bad origin missing origin zero/negative/malformed window

**Environment/selection gates:** CAN_BUN_ARCHIVE

**Capabilities:** CAN CHILD ENV FAULT WORK

**Proposed Can replacement:** Author an ordinary Can suite case that stages the named fixtures, chooses the listed variants and calls generic build/process/browser/database mechanics; Can compares Server must exit before listening for missing/invalid PUBLIC_ORIGIN or nonpositive/malformed replay window.

**Old harness may be deleted when:** Delete this Go test only after each listed obligation has a mapped passing Can case on its required environments, independent observations and negative controls remain effective, and all callers/report consumers have switched.

**Related source:** [tests/integration/testdata/invoice/driver.ts](/Users/vince/Projects/can-lang/tests/integration/testdata/invoice/driver.ts)

**Notes:** Proposed only; retain all scenario and oracle decisions in ordinary Can functions. Browser driver/native adapter may expose generic mechanics and observations, never this case sequence or verdict.

## BROWSER-031

**TestInvoiceGridPagePaired** — [tests/integration/invoice_test.go](/Users/vince/Projects/can-lang/tests/integration/invoice_test.go:1042)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-BROWSER-031`. Evidence: pending.

**Protects**

- Paired invoice server serves exactly report-selected digest script in grid and form shell; script bytes and non-HTML health response are correct.
- The empty browser fixture proves wiring only, not grid interactions.

**Current observations:** build/paired manifest, HTTP script/shell/health bytes

**Variants:** grid shell form shell health

**Environment/selection gates:** CAN_BUN_ARCHIVE

**Capabilities:** CAN BUILD HTTP FILES WORK

**Proposed Can replacement:** Author an ordinary Can suite case that stages the named fixtures, chooses the listed variants and calls generic build/process/browser/database mechanics; Can compares Paired invoice server serves exactly report-selected digest script in grid and form shell; script bytes and non-HTML health response are correct.; The empty browser fixture proves wiring only, not grid interactions.

**Old harness may be deleted when:** Delete this Go test only after each listed obligation has a mapped passing Can case on its required environments, independent observations and negative controls remain effective, and all callers/report consumers have switched.

**Related source:** [tests/integration/testdata/invoice/driver.ts](/Users/vince/Projects/can-lang/tests/integration/testdata/invoice/driver.ts)

**Notes:** Proposed only; retain all scenario and oracle decisions in ordinary Can functions. Browser driver/native adapter may expose generic mechanics and observations, never this case sequence or verdict.

## BROWSER-032

**TestInvoiceBrowser** — [tests/integration/invoice_test.go](/Users/vince/Projects/can-lang/tests/integration/invoice_test.go:1153)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-BROWSER-032`. Evidence: pending.

**Protects**

- Drive invoice.mjs on the compiled live invoice server and seeded SQLite; require a well-formed passing browser report and every reported check passed.
- Independently inspect the SQLite invoice row and require invoice 7 revision to differ from 1 after browser submission.

**Current observations:** browser JSON report verdict and each check, independent SQLite invoice revision

**Variants:** Chromium default of invoice.mjs tenant_id content gate on script source

**Environment/selection gates:** CAN_BUN_ARCHIVE pinned Playwright installation invoice.mjs contains tenant_id (currently true)

**Capabilities:** BROWSER DB SUITE WORK

**Proposed Can replacement:** Can functions drive the staged invoice browser interaction, compare each required DOM/security/report observation, and independently inspect the SQLite commit. Consolidation with Gate5 is acceptable only if every retained obligation and the independent commit check have explicit Can-owned coverage.

**Old harness may be deleted when:** Delete this Go test only after each listed obligation has a mapped passing Can case on its required environments, independent observations and negative controls remain effective, and all callers/report consumers have switched.

**Related source:** [tests/integration/testdata/invoice/driver.ts](/Users/vince/Projects/can-lang/tests/integration/testdata/invoice/driver.ts); [tests/integration/browser/invoice.mjs](/Users/vince/Projects/can-lang/tests/integration/browser/invoice.mjs)

**Notes:** Source invoice_test.go:1153-1223 and invoice.mjs:15-30. The source-content gate currently passes because invoice.mjs contains tenant_id. The script accepts omitted engine/script URL and defaults to Chromium; this row does not claim that a live run passed.

## BROWSER-033

**TestInvoiceHTMLFragments** — [tests/integration/invoice_test.go](/Users/vince/Projects/can-lang/tests/integration/invoice_test.go:1231)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-BROWSER-033`. Evidence: pending.

**Protects**

- Server page declares same-origin htmx noSwap and allowed per-status swaps; emitted fragments have exact HTML content type and safe escaped bytes.
- Save statuses 200/403/409/422/503 and denied/malformed/redirect cases preserve draft/error semantics.

**Current observations:** HTTP status/content-type/body and served page policy

**Variants:** success forbidden conflict validation busy malformed/redirect

**Environment/selection gates:** CAN_BUN_ARCHIVE

**Capabilities:** CAN HTTP FAULT FILES WORK

**Proposed Can replacement:** Author an ordinary Can suite case that stages the named fixtures, chooses the listed variants and calls generic build/process/browser/database mechanics; Can compares Server page declares same-origin htmx noSwap and allowed per-status swaps; emitted fragments have exact HTML content type and safe escaped bytes.; Save statuses 200/403/409/422/503 and denied/malformed/redirect cases preserve draft/error semantics.

**Old harness may be deleted when:** Delete this Go test only after each listed obligation has a mapped passing Can case on its required environments, independent observations and negative controls remain effective, and all callers/report consumers have switched.

**Related source:** [tests/integration/testdata/invoice/driver.ts](/Users/vince/Projects/can-lang/tests/integration/testdata/invoice/driver.ts)

**Notes:** Proposed only; retain all scenario and oracle decisions in ordinary Can functions. Browser driver/native adapter may expose generic mechanics and observations, never this case sequence or verdict.

## BROWSER-034

**TestInvoiceBrowserGuardDOM** — [tests/integration/invoice_test.go](/Users/vince/Projects/can-lang/tests/integration/invoice_test.go:1418)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-BROWSER-034`. Evidence: pending.

**Protects**

- Validate supplied UP23 verdict record includes engine/commit and pass plus evidence for every required DOM leg: 200/403/409/422/503 swaps, missing targets, rejected out-of-band/partial/control headers and remount.
- A supplied report is historical/indirect evidence, not live execution by this Go test.

**Current observations:** external verdict JSON presence, fields and named-leg pass/evidence

**Variants:** 11 named verdict legs

**Environment/selection gates:** CAN_UP23_RESULTS supplied or skipped

**Capabilities:** DOC FILES SUITE BROWSER

**Proposed Can replacement:** Author an ordinary Can suite case that stages the named fixtures, chooses the listed variants and calls generic build/process/browser/database mechanics; Can compares Validate supplied UP23 verdict record includes engine/commit and pass plus evidence for every required DOM leg: 200/403/409/422/503 swaps, missing targets, rejected out-of-band/partial/control headers and remount.; A supplied report is historical/indirect evidence, not live execution by this Go test.

**Old harness may be deleted when:** Delete this Go test only after each listed obligation has a mapped passing Can case on its required environments, independent observations and negative controls remain effective, and all callers/report consumers have switched.

**Notes:** Currently consumes CAN_UP23_RESULTS only; replacement must run/record named live Can browser legs or explicitly preserve this as provenance, without counting unrun legs.

## BROWSER-035

**accounts.mjs** — [tests/integration/browser/accounts.mjs](/Users/vince/Projects/can-lang/tests/integration/browser/accounts.mjs:6)

Kind: `scenario`. Proposed disposition: `migrate`. Replacement obligation: `CAN-BROWSER-035`. Evidence: pending.

**Protects**

- Chromium account-search HTMX page: pinned local scripts, 200 search swap, quiet 422 search, escaped hostile search, validation 422/200 swaps and dashboard polling.
- Require no outbound request and screenshot/report evidence.
- Named old checks/legs to account for: htmx-loaded, pinned-scripts, search-200-swap, search-422-quiet, search-hostile-escaped, validate-422-swap, validate-200-swap, dashboard-poll, no-external-requests.

**Current observations:** browser DOM and response/route ledger, intercepted requests, screenshot

**Variants:** Chromium valid/empty/hostile search valid/invalid validation

**Environment/selection gates:** Playwright/Chromium live seeded PostgreSQL

**Capabilities:** BROWSER HTTP DB WORK

**Proposed Can replacement:** Author ordinary Can case/helper logic for Chromium account-search HTMX page: pinned local scripts, 200 search swap, quiet 422 search, escaped hostile search, validation 422/200 swaps and dashboard polling.; Require no outbound request and screenshot/report evidence. Use generic native browser/build/database mechanics to return observations; Can applies all case expectations and reporting.

**Old harness may be deleted when:** Retire authored host source only after every protected facet has passing Can-owned evidence in required environments, its active callers and CI paths are rewired, and historical references are preserved or updated; confirm no test-specific scenario or verdict remains in native infrastructure.

**Related source:** [tests/integration/applications_test.go](/Users/vince/Projects/can-lang/tests/integration/applications_test.go); [tests/integration/browser/bun.lock](/Users/vince/Projects/can-lang/tests/integration/browser/bun.lock); [tests/integration/browser/package.json](/Users/vince/Projects/can-lang/tests/integration/browser/package.json)

**Notes:** No unchanged Playwright script may be invoked by a Can wrapper as migration evidence. Generic browser infrastructure may provide actions and observations; Can chooses the sequence, expectations, engine profile and verdict.

## BROWSER-036

**assets.mjs** — [tests/integration/browser/assets.mjs](/Users/vince/Projects/can-lang/tests/integration/browser/assets.mjs:8)

Kind: `scenario`. Proposed disposition: `migrate`. Replacement obligation: `CAN-BROWSER-036`. Evidence: pending.

**Protects**

- Browser asset page: pinned HTMX/guard and local stylesheet, CSP/config, form 422 quiet/200 swap, hostile escaping, 204/500 no-swap and dashboard polling.
- Require no outbound requests and screenshot/report evidence.
- Named old checks/legs to account for: htmx-loaded, pinned-scripts, htmx-config, stylesheet-local, csp-header, form-422-quiet, form-200-swap, hostile-escaped, 204-no-swap, 500-no-swap, dashboard-polling, no-external-requests.

**Current observations:** DOM, response headers, request ledger, page faults and screenshot

**Variants:** Chromium form/status variants

**Environment/selection gates:** Playwright/Chromium

**Capabilities:** BROWSER HTTP FILES WORK

**Proposed Can replacement:** Author ordinary Can case/helper logic for Browser asset page: pinned HTMX/guard and local stylesheet, CSP/config, form 422 quiet/200 swap, hostile escaping, 204/500 no-swap and dashboard polling.; Require no outbound requests and screenshot/report evidence. Use generic native browser/build/database mechanics to return observations; Can applies all case expectations and reporting.

**Old harness may be deleted when:** Retire authored host source only after every protected facet has passing Can-owned evidence in required environments, its active callers and CI paths are rewired, and historical references are preserved or updated; confirm no test-specific scenario or verdict remains in native infrastructure.

**Related source:** [tests/integration/assets_test.go](/Users/vince/Projects/can-lang/tests/integration/assets_test.go); [tests/integration/browser/bun.lock](/Users/vince/Projects/can-lang/tests/integration/browser/bun.lock); [tests/integration/browser/package.json](/Users/vince/Projects/can-lang/tests/integration/browser/package.json)

**Notes:** No unchanged Playwright script may be invoked by a Can wrapper as migration evidence. Generic browser infrastructure may provide actions and observations; Can chooses the sequence, expectations, engine profile and verdict.

## BROWSER-037

**build-grid.mjs** — [tests/integration/browser/build-grid.mjs](/Users/vince/Projects/can-lang/tests/integration/browser/build-grid.mjs:29)

Kind: `support`. Proposed disposition: `historical_candidate`. Replacement obligation: `CAN-BROWSER-037`. Evidence: pending.

**Protects**

- Historical test bundler authored a three-line grid bootstrap plus four Node shims and diagnostic source index; verify whether any current claimed path still depends on these semantics before retiring.

**Current observations:** bundle output and shim behavior once used by old grid gate

**Variants:** browser.ts grid entry four shim modules

**Environment/selection gates:** Bun bundler

**Capabilities:** BUILD FILES NATIVE DOC

**Proposed Can replacement:** Retain only as historical evidence if the active-caller audit confirms no current coverage need; move any still-required digest/bundle qualification into Can-owned scenarios with generic native build observations.

**Old harness may be deleted when:** Retire authored host source only after every protected facet has passing Can-owned evidence in required environments, its active callers and CI paths are rewired, and historical references are preserved or updated; confirm no test-specific scenario or verdict remains in native infrastructure.

**Related source:** [tests/integration/browser/sha256-shim.mjs](/Users/vince/Projects/can-lang/tests/integration/browser/sha256-shim.mjs); [tests/baseline/t27_release_audit_test.go](/Users/vince/Projects/can-lang/tests/baseline/t27_release_audit_test.go)

**Notes:** No active tracked integration Go caller found by repository-wide reference search; historical docs/evidence and the baseline T27 source-presence check still reference this file. Retirement is a candidate only after those callers/evidence roles are reconciled.

## BROWSER-038

**build-vectors.mjs** — [tests/integration/browser/build-vectors.mjs](/Users/vince/Projects/can-lang/tests/integration/browser/build-vectors.mjs:11)

Kind: `support`. Proposed disposition: `replace_support`. Replacement obligation: `CAN-BROWSER-038`. Evidence: pending.

**Protects**

- Bundle shared wire vectors through the exact 14-edge browser overlay; fail if another node builtin reaches the graph.
- Keep overlay-table drift detectable against compiler canonical table.

**Current observations:** bundle success/logs, emitted bundle imports and overlay edge equality

**Variants:** 14 overlay edges unexpected node import

**Environment/selection gates:** Bun bundler

**Capabilities:** BUILD FILES FAULT

**Proposed Can replacement:** Author ordinary Can case/helper logic for Bundle shared wire vectors through the exact 14-edge browser overlay; fail if another node builtin reaches the graph.; Keep overlay-table drift detectable against compiler canonical table. Use generic native browser/build/database mechanics to return observations; Can applies all case expectations and reporting.

**Old harness may be deleted when:** Retire authored host source only after every protected facet has passing Can-owned evidence in required environments, its active callers and CI paths are rewired, and historical references are preserved or updated; confirm no test-specific scenario or verdict remains in native infrastructure.

**Related source:** [compiler/internal/browser/audit.go](/Users/vince/Projects/can-lang/compiler/internal/browser/audit.go); [tests/integration/browser/codec-vectors-entry.ts](/Users/vince/Projects/can-lang/tests/integration/browser/codec-vectors-entry.ts); [tests/integration/browser_build_test.go](/Users/vince/Projects/can-lang/tests/integration/browser_build_test.go)

**Notes:** No unchanged Playwright script may be invoked by a Can wrapper as migration evidence. Generic browser infrastructure may provide actions and observations; Can chooses the sequence, expectations, engine profile and verdict.

## BROWSER-039

**codec-parity.mjs** — [tests/integration/browser/codec-parity.mjs](/Users/vince/Projects/can-lang/tests/integration/browser/codec-parity.mjs:1)

Kind: `scenario`. Proposed disposition: `migrate`. Replacement obligation: `CAN-BROWSER-039`. Evidence: pending.

**Protects**

- Launch named browser, report engine/version/UA and JSON.rawJSON availability, execute vector bundle and return raw results.
- Can case, not this script, must decide required-engine policy and exact parity.

**Current observations:** browser-executed vector values and capability metadata

**Variants:** Chromium WebKit Firefox

**Environment/selection gates:** Playwright/browser engines

**Capabilities:** BROWSER NATIVE SUITE

**Proposed Can replacement:** Author ordinary Can case/helper logic for Launch named browser, report engine/version/UA and JSON.rawJSON availability, execute vector bundle and return raw results.; Can case, not this script, must decide required-engine policy and exact parity. Use generic native browser/build/database mechanics to return observations; Can applies all case expectations and reporting.

**Old harness may be deleted when:** Retire authored host source only after every protected facet has passing Can-owned evidence in required environments, its active callers and CI paths are rewired, and historical references are preserved or updated; confirm no test-specific scenario or verdict remains in native infrastructure.

**Related source:** [runtime/test/browser-wire-vectors.ts](/Users/vince/Projects/can-lang/runtime/test/browser-wire-vectors.ts); [tests/integration/browser/bun.lock](/Users/vince/Projects/can-lang/tests/integration/browser/bun.lock); [tests/integration/browser/codec-vectors-entry.ts](/Users/vince/Projects/can-lang/tests/integration/browser/codec-vectors-entry.ts); [tests/integration/browser/package.json](/Users/vince/Projects/can-lang/tests/integration/browser/package.json); [tests/integration/browser_build_test.go](/Users/vince/Projects/can-lang/tests/integration/browser_build_test.go)

**Notes:** No unchanged Playwright script may be invoked by a Can wrapper as migration evidence. Generic browser infrastructure may provide actions and observations; Can chooses the sequence, expectations, engine profile and verdict.

## BROWSER-040

**codec-vectors-entry.ts** — [tests/integration/browser/codec-vectors-entry.ts](/Users/vince/Projects/can-lang/tests/integration/browser/codec-vectors-entry.ts:4)

Kind: `support`. Proposed disposition: `replace_support`. Replacement obligation: `CAN-BROWSER-040`. Evidence: pending.

**Protects**

- Test-only TypeScript entry imports runtime/test/browser-wire-vectors.ts and writes results to globalThis; its delegated vector oracle must be represented by Can-authored vectors and comparison policy.
- Preserve all 12 delegated vector obligations: int64-beyond-safe, int64-huge-exponent, float-finite, float-nonfinite-rejected, variant-canonical-tag, variant-wrong-tag, option-some-none, duplicate-member, duplicate-native-syntax, extra-member, missing-member, nested-roundtrip-stable (runtime/test/browser-wire-vectors.ts:45-157).

**Current observations:** browser global vector results

**Variants:** wire vector inputs

**Environment/selection gates:** None recorded.

**Capabilities:** BROWSER NATIVE CAN

**Proposed Can replacement:** Author ordinary Can case/helper logic for Test-only TypeScript entry imports runtime/test/browser-wire-vectors.ts and writes results to globalThis; its delegated vector oracle must be represented by Can-authored vectors and comparison policy. Use generic native browser/build/database mechanics to return observations; Can applies all case expectations and reporting.

**Old harness may be deleted when:** Retire authored host source only after every protected facet has passing Can-owned evidence in required environments, its active callers and CI paths are rewired, and historical references are preserved or updated; confirm no test-specific scenario or verdict remains in native infrastructure.

**Related source:** [runtime/test/browser-wire-vectors.ts](/Users/vince/Projects/can-lang/runtime/test/browser-wire-vectors.ts); [tests/integration/browser_build_test.go](/Users/vince/Projects/can-lang/tests/integration/browser_build_test.go)

**Notes:** The current vector oracle is delegated to runtime/test/browser-wire-vectors.ts; running that TypeScript entry unchanged from Can does not fulfill Can ownership. Preserve vector coverage with a Can-authored oracle and independent browser observations.

## BROWSER-041

**compare.mjs** — [tests/integration/browser/compare.mjs](/Users/vince/Projects/can-lang/tests/integration/browser/compare.mjs:12)

Kind: `scenario`. Proposed disposition: `migrate`. Replacement obligation: `CAN-BROWSER-041`. Evidence: pending.

**Protects**

- Can compare client: boot panels, paired identity and generation slot, left/right edit and mirror/drop lifecycles, invalid review block.
- Real keystrokes, caret and composing input keep intended values; network, console and page faults stay clean.
- Named old checks/legs to account for: boot-panels, client-identity, generation-slot, edit-left-keeps-right, review-blocks-on-invalid, real-keystrokes, caret-preserved-across-folds, composing-input-folds, mirror-folds-left-into-right, drop-left-right-works, mirror-after-drop-silent, network-ledger-clean, no-page-faults.

**Current observations:** DOM values/focus/selection, network ledger and page faults

**Variants:** Chromium/WebKit/Firefox edit/mirror/drop/composition

**Environment/selection gates:** Playwright/browser engines

**Capabilities:** BROWSER HTTP CHILD WORK

**Proposed Can replacement:** Author ordinary Can case/helper logic for Can compare client: boot panels, paired identity and generation slot, left/right edit and mirror/drop lifecycles, invalid review block.; Real keystrokes, caret and composing input keep intended values; network, console and page faults stay clean. Use generic native browser/build/database mechanics to return observations; Can applies all case expectations and reporting.

**Old harness may be deleted when:** Retire authored host source only after every protected facet has passing Can-owned evidence in required environments, its active callers and CI paths are rewired, and historical references are preserved or updated; confirm no test-specific scenario or verdict remains in native infrastructure.

**Related source:** [tests/integration/browser/bun.lock](/Users/vince/Projects/can-lang/tests/integration/browser/bun.lock); [tests/integration/browser/package.json](/Users/vince/Projects/can-lang/tests/integration/browser/package.json); [tests/integration/c06_test.go](/Users/vince/Projects/can-lang/tests/integration/c06_test.go)

**Notes:** No unchanged Playwright script may be invoked by a Can wrapper as migration evidence. Generic browser infrastructure may provide actions and observations; Can chooses the sequence, expectations, engine profile and verdict. Current page.evaluate and callback/interception behavior requires generic browser mechanisms for DOM property/event probes, request interception, response injection or controlled rendezvous. Test-specific JavaScript callbacks embedded in an adapter would leave an authored host harness.

## BROWSER-042

**conformance.mjs** — [tests/integration/browser/conformance.mjs](/Users/vince/Projects/can-lang/tests/integration/browser/conformance.mjs:14)

Kind: `scenario`. Proposed disposition: `migrate`. Replacement obligation: `CAN-BROWSER-042`. Evidence: pending.

**Protects**

- Compiler-built browser fixture: client identity; equality/codec vectors and query plain/missing/duplicate/alias/oversized/malformed/no/unknown cases.
- Timer order/disposal silence, view release, cancel/Enter and post-attach focus; no foreign request or page fault.
- Named old checks/legs to account for: client-identity, equality, codecs, query-plain, query-missing-probe, query-duplicate, query-encoded-alias, query-huge-value, query-huge-search, query-malformed, no-scenario, unknown-scenario, timers, dispose, cancel, focus, network-ledger-clean.

**Current observations:** DOM verdict rows, request ledger, console/page faults

**Variants:** Chromium/WebKit/Firefox query variants timer/dispose/cancel/focus

**Environment/selection gates:** Playwright/browser engines

**Capabilities:** BROWSER CAN HTTP CLOCK WORK

**Proposed Can replacement:** Author ordinary Can case/helper logic for Compiler-built browser fixture: client identity; equality/codec vectors and query plain/missing/duplicate/alias/oversized/malformed/no/unknown cases.; Timer order/disposal silence, view release, cancel/Enter and post-attach focus; no foreign request or page fault. Use generic native browser/build/database mechanics to return observations; Can applies all case expectations and reporting.

**Old harness may be deleted when:** Retire authored host source only after every protected facet has passing Can-owned evidence in required environments, its active callers and CI paths are rewired, and historical references are preserved or updated; confirm no test-specific scenario or verdict remains in native infrastructure.

**Related source:** [tests/integration/browser/bun.lock](/Users/vince/Projects/can-lang/tests/integration/browser/bun.lock); [tests/integration/browser/package.json](/Users/vince/Projects/can-lang/tests/integration/browser/package.json); [tests/integration/gate5_frontend_test.go](/Users/vince/Projects/can-lang/tests/integration/gate5_frontend_test.go)

**Notes:** No unchanged Playwright script may be invoked by a Can wrapper as migration evidence. Generic browser infrastructure may provide actions and observations; Can chooses the sequence, expectations, engine profile and verdict.

## BROWSER-043

**controls-native.mjs** — [tests/integration/browser/controls-native.mjs](/Users/vince/Projects/can-lang/tests/integration/browser/controls-native.mjs:10)

Kind: `scenario`. Proposed disposition: `migrate`. Replacement obligation: `CAN-BROWSER-043`. Evidence: pending.

**Protects**

- Independent native DOM facts: checkbox IDL vs value, dirty live value vs attribute, form reset, multi/single selection, file metadata.
- Real and synthetic modifiers, autofill-simulated input, composing flag, selection/caret; network and page errors clean.
- Named old checks/legs to account for: checkbox-idl, dirty-divergence, form-reset, multiselect, single-last-wins, files, modifiers, autofill-simulated, composing, selection, network-ledger-clean.

**Current observations:** native DOM properties/events captured independently of Can adapter

**Variants:** Chromium/WebKit/Firefox real/synthetic input file/select/reset

**Environment/selection gates:** Playwright/browser engines

**Capabilities:** BROWSER NATIVE WORK

**Proposed Can replacement:** Author ordinary Can case/helper logic for Independent native DOM facts: checkbox IDL vs value, dirty live value vs attribute, form reset, multi/single selection, file metadata.; Real and synthetic modifiers, autofill-simulated input, composing flag, selection/caret; network and page errors clean. Use generic native browser/build/database mechanics to return observations; Can applies all case expectations and reporting.

**Old harness may be deleted when:** Retire authored host source only after every protected facet has passing Can-owned evidence in required environments, its active callers and CI paths are rewired, and historical references are preserved or updated; confirm no test-specific scenario or verdict remains in native infrastructure.

**Related source:** [tests/integration/browser/bun.lock](/Users/vince/Projects/can-lang/tests/integration/browser/bun.lock); [tests/integration/browser/package.json](/Users/vince/Projects/can-lang/tests/integration/browser/package.json); [tests/integration/browser_controls_test.go](/Users/vince/Projects/can-lang/tests/integration/browser_controls_test.go); [tests/integration/testdata/browser-controls-native/index.html](/Users/vince/Projects/can-lang/tests/integration/testdata/browser-controls-native/index.html)

**Notes:** No unchanged Playwright script may be invoked by a Can wrapper as migration evidence. Generic browser infrastructure may provide actions and observations; Can chooses the sequence, expectations, engine profile and verdict. Current page.evaluate and callback/interception behavior requires generic browser mechanisms for DOM property/event probes, request interception, response injection or controlled rendezvous. Test-specific JavaScript callbacks embedded in an adapter would leave an authored host harness.

## BROWSER-044

**controls.mjs** — [tests/integration/browser/controls.mjs](/Users/vince/Projects/can-lang/tests/integration/browser/controls.mjs:13)

Kind: `scenario`. Proposed disposition: `migrate`. Replacement obligation: `CAN-BROWSER-044`. Evidence: pending.

**Protects**

- Emitted Can control fixture identity/boot; text/key/check/multi/files echoes; setvalue/setchecked/setselected/setcaret/readall/reset/reject operations.
- Synthetic composition and autofill, dirty reset, engine-specific caret direction; no API calls or foreign traffic.
- Named old checks/legs to account for: client-identity, boot, text-echo, key-echo, check-echo, multi-echo, files-echo, setvalue, setchecked, setselected, setcaret, composing-synthetic, autofill-simulated, dirty-reset, readall, reject, network-ledger-clean.

**Current observations:** Can verdict lines plus independent DOM live properties and network ledger

**Variants:** Chromium/WebKit/Firefox 14 named control legs engine caret direction

**Environment/selection gates:** Playwright/browser engines paired server

**Capabilities:** CAN BROWSER NATIVE HTTP WORK

**Proposed Can replacement:** Author ordinary Can case/helper logic for Emitted Can control fixture identity/boot; text/key/check/multi/files echoes; setvalue/setchecked/setselected/setcaret/readall/reset/reject operations.; Synthetic composition and autofill, dirty reset, engine-specific caret direction; no API calls or foreign traffic. Use generic native browser/build/database mechanics to return observations; Can applies all case expectations and reporting.

**Old harness may be deleted when:** Retire authored host source only after every protected facet has passing Can-owned evidence in required environments, its active callers and CI paths are rewired, and historical references are preserved or updated; confirm no test-specific scenario or verdict remains in native infrastructure.

**Related source:** [tests/integration/browser/bun.lock](/Users/vince/Projects/can-lang/tests/integration/browser/bun.lock); [tests/integration/browser/package.json](/Users/vince/Projects/can-lang/tests/integration/browser/package.json); [tests/integration/browser_controls_test.go](/Users/vince/Projects/can-lang/tests/integration/browser_controls_test.go); [tests/integration/testdata/browser-controls/src/main.can](/Users/vince/Projects/can-lang/tests/integration/testdata/browser-controls/src/main.can)

**Notes:** No unchanged Playwright script may be invoked by a Can wrapper as migration evidence. Generic browser infrastructure may provide actions and observations; Can chooses the sequence, expectations, engine profile and verdict. Current page.evaluate and callback/interception behavior requires generic browser mechanisms for DOM property/event probes, request interception, response injection or controlled rendezvous. Test-specific JavaScript callbacks embedded in an adapter would leave an authored host harness.

## BROWSER-045

**dashboard.mjs** — [tests/integration/browser/dashboard.mjs](/Users/vince/Projects/can-lang/tests/integration/browser/dashboard.mjs:6)

Kind: `scenario`. Proposed disposition: `migrate`. Replacement obligation: `CAN-BROWSER-045`. Evidence: pending.

**Protects**

- Chromium HTMX dashboard: pinned scripts, initial poll swap and repeated polls; no external requests.
- Named old checks/legs to account for: htmx-loaded, pinned-scripts, dashboard-poll-swap, dashboard-repeated-polls, no-external-requests.

**Current observations:** DOM text changes and counted HTTP poll requests

**Variants:** Chromium first/repeated poll

**Environment/selection gates:** Playwright/Chromium live seeded PostgreSQL

**Capabilities:** BROWSER HTTP DB CLOCK

**Proposed Can replacement:** Author ordinary Can case/helper logic for Chromium HTMX dashboard: pinned scripts, initial poll swap and repeated polls; no external requests.; Named old checks/legs to account for: htmx-loaded, pinned-scripts, dashboard-poll-swap, dashboard-repeated-polls, no-external-requests. Use generic native browser/build/database mechanics to return observations; Can applies all case expectations and reporting.

**Old harness may be deleted when:** Retire authored host source only after every protected facet has passing Can-owned evidence in required environments, its active callers and CI paths are rewired, and historical references are preserved or updated; confirm no test-specific scenario or verdict remains in native infrastructure.

**Related source:** [tests/integration/applications_test.go](/Users/vince/Projects/can-lang/tests/integration/applications_test.go); [tests/integration/browser/bun.lock](/Users/vince/Projects/can-lang/tests/integration/browser/bun.lock); [tests/integration/browser/package.json](/Users/vince/Projects/can-lang/tests/integration/browser/package.json)

**Notes:** No unchanged Playwright script may be invoked by a Can wrapper as migration evidence. Generic browser infrastructure may provide actions and observations; Can chooses the sequence, expectations, engine profile and verdict.

## BROWSER-046

**drift.mjs** — [tests/integration/browser/drift.mjs](/Users/vince/Projects/can-lang/tests/integration/browser/drift.mjs:11)

Kind: `scenario`. Proposed disposition: `migrate`. Replacement obligation: `CAN-BROWSER-046`. Evidence: pending.

**Protects**

- Boot V1 page, signal readiness, wait for server restart to V2; old page POST must get exact 409 generation mismatch and prompt, without committing.
- Refresh must show V2 and unchanged invoice revision; run both rollout and rollback with no unexpected page faults.
- Named old checks/legs to account for: boot-v1, old-page-refused, refused-save-never-ran, no-page-faults.

**Current observations:** browser DOM/request+response bytes, generation slot, independent server transition

**Variants:** Chromium/WebKit/Firefox rollout rollback

**Environment/selection gates:** Playwright/browser engines two paired server generations

**Capabilities:** BROWSER CHILD HTTP WIRE WORK

**Proposed Can replacement:** Author ordinary Can case/helper logic for Boot V1 page, signal readiness, wait for server restart to V2; old page POST must get exact 409 generation mismatch and prompt, without committing.; Refresh must show V2 and unchanged invoice revision; run both rollout and rollback with no unexpected page faults. Use generic native browser/build/database mechanics to return observations; Can applies all case expectations and reporting.

**Old harness may be deleted when:** Retire authored host source only after every protected facet has passing Can-owned evidence in required environments, its active callers and CI paths are rewired, and historical references are preserved or updated; confirm no test-specific scenario or verdict remains in native infrastructure.

**Related source:** [tests/integration/browser/bun.lock](/Users/vince/Projects/can-lang/tests/integration/browser/bun.lock); [tests/integration/browser/package.json](/Users/vince/Projects/can-lang/tests/integration/browser/package.json); [tests/integration/c06_test.go](/Users/vince/Projects/can-lang/tests/integration/c06_test.go)

**Notes:** No unchanged Playwright script may be invoked by a Can wrapper as migration evidence. Generic browser infrastructure may provide actions and observations; Can chooses the sequence, expectations, engine profile and verdict. Current page.evaluate and callback/interception behavior requires generic browser mechanisms for DOM property/event probes, request interception, response injection or controlled rendezvous. Test-specific JavaScript callbacks embedded in an adapter would leave an authored host harness.

## BROWSER-047

**empty.mjs** — [tests/integration/browser/empty.mjs](/Users/vince/Projects/can-lang/tests/integration/browser/empty.mjs:10)

Kind: `scenario`. Proposed disposition: `migrate`. Replacement obligation: `CAN-BROWSER-047`. Evidence: pending.

**Protects**

- Minimal generated browser program served through paired shell: exact selected script, clean startup, untouched DOM.
- Serve source map and sealed diagnostic table with correct shape; loopback-only requests and clean page console.
- Named old checks/legs to account for: served-shell, paired-startup-clean, served-map-table, network-ledger-clean.

**Current observations:** HTTP response/headers/map/table, DOM, network and page errors

**Variants:** Chromium/WebKit/Firefox

**Environment/selection gates:** Playwright/browser engines paired empty app

**Capabilities:** BROWSER BUILD HTTP FILES

**Proposed Can replacement:** Author ordinary Can case/helper logic for Minimal generated browser program served through paired shell: exact selected script, clean startup, untouched DOM.; Serve source map and sealed diagnostic table with correct shape; loopback-only requests and clean page console. Use generic native browser/build/database mechanics to return observations; Can applies all case expectations and reporting.

**Old harness may be deleted when:** Retire authored host source only after every protected facet has passing Can-owned evidence in required environments, its active callers and CI paths are rewired, and historical references are preserved or updated; confirm no test-specific scenario or verdict remains in native infrastructure.

**Related source:** [tests/integration/browser/bun.lock](/Users/vince/Projects/can-lang/tests/integration/browser/bun.lock); [tests/integration/browser/package.json](/Users/vince/Projects/can-lang/tests/integration/browser/package.json); [tests/integration/gate5_frontend_test.go](/Users/vince/Projects/can-lang/tests/integration/gate5_frontend_test.go)

**Notes:** No unchanged Playwright script may be invoked by a Can wrapper as migration evidence. Generic browser infrastructure may provide actions and observations; Can chooses the sequence, expectations, engine profile and verdict.

## BROWSER-048

**firefox-remote.mjs** — [tests/integration/browser/firefox-remote.mjs](/Users/vince/Projects/can-lang/tests/integration/browser/firefox-remote.mjs:10)

Kind: `support`. Proposed disposition: `replace_support`. Replacement obligation: `CAN-BROWSER-048`. Evidence: pending.

**Protects**

- Select local Firefox launch or CAN_FIREFOX_WS remote connection; isolate each leg in a fresh context and never close shared launchServer.

**Current observations:** connection mode, browser lifecycle and context cleanup

**Variants:** local Firefox remote Firefox

**Environment/selection gates:** CAN_FIREFOX_WS optional

**Capabilities:** BROWSER ENV WORK

**Proposed Can replacement:** Author ordinary Can case/helper logic for Select local Firefox launch or CAN_FIREFOX_WS remote connection; isolate each leg in a fresh context and never close shared launchServer. Use generic native browser/build/database mechanics to return observations; Can applies all case expectations and reporting.

**Old harness may be deleted when:** Retire authored host source only after every protected facet has passing Can-owned evidence in required environments, its active callers and CI paths are rewired, and historical references are preserved or updated; confirm no test-specific scenario or verdict remains in native infrastructure.

**Related source:** [tests/integration/browser/bun.lock](/Users/vince/Projects/can-lang/tests/integration/browser/bun.lock); [tests/integration/browser/compare.mjs](/Users/vince/Projects/can-lang/tests/integration/browser/compare.mjs); [tests/integration/browser/conformance.mjs](/Users/vince/Projects/can-lang/tests/integration/browser/conformance.mjs); [tests/integration/browser/controls-native.mjs](/Users/vince/Projects/can-lang/tests/integration/browser/controls-native.mjs); [tests/integration/browser/controls.mjs](/Users/vince/Projects/can-lang/tests/integration/browser/controls.mjs); [tests/integration/browser/drift.mjs](/Users/vince/Projects/can-lang/tests/integration/browser/drift.mjs); [tests/integration/browser/empty.mjs](/Users/vince/Projects/can-lang/tests/integration/browser/empty.mjs); [tests/integration/browser/grid.mjs](/Users/vince/Projects/can-lang/tests/integration/browser/grid.mjs); [tests/integration/browser/invoice.mjs](/Users/vince/Projects/can-lang/tests/integration/browser/invoice.mjs); [tests/integration/browser/package.json](/Users/vince/Projects/can-lang/tests/integration/browser/package.json); [tests/integration/browser/w1-grid.mjs](/Users/vince/Projects/can-lang/tests/integration/browser/w1-grid.mjs)

**Notes:** No unchanged Playwright script may be invoked by a Can wrapper as migration evidence. Generic browser infrastructure may provide actions and observations; Can chooses the sequence, expectations, engine profile and verdict. Browser package.json pins Playwright 1.55.1 and bun.lock locks the installation; these are dependency evidence, not executable test cases. A native replacement may use an open backend while preserving engine checks/limitations.

## BROWSER-049

**forms.mjs** — [tests/integration/browser/forms.mjs](/Users/vince/Projects/can-lang/tests/integration/browser/forms.mjs:6)

Kind: `scenario`. Proposed disposition: `migrate`. Replacement obligation: `CAN-BROWSER-049`. Evidence: pending.

**Protects**

- Chromium signup form: HTMX and pinned scripts, no inline handlers, CSP, 422 and 200 swaps, hostile escaping with no script execution.
- All browser requests stay loopback and screenshot/report are captured.
- Named old checks/legs to account for: htmx-loaded, pinned-scripts, no-inline-handlers, csp-header, form-422-swap, form-200-swap, hostile-escaped, no-external-requests.

**Current observations:** DOM, response status/headers, request ledger and screenshot

**Variants:** Chromium invalid/valid/hostile form

**Environment/selection gates:** Playwright/Chromium

**Capabilities:** BROWSER HTTP FAULT

**Proposed Can replacement:** Author ordinary Can case/helper logic for Chromium signup form: HTMX and pinned scripts, no inline handlers, CSP, 422 and 200 swaps, hostile escaping with no script execution.; All browser requests stay loopback and screenshot/report are captured. Use generic native browser/build/database mechanics to return observations; Can applies all case expectations and reporting.

**Old harness may be deleted when:** Retire authored host source only after every protected facet has passing Can-owned evidence in required environments, its active callers and CI paths are rewired, and historical references are preserved or updated; confirm no test-specific scenario or verdict remains in native infrastructure.

**Related source:** [tests/integration/applications_test.go](/Users/vince/Projects/can-lang/tests/integration/applications_test.go); [tests/integration/browser/bun.lock](/Users/vince/Projects/can-lang/tests/integration/browser/bun.lock); [tests/integration/browser/package.json](/Users/vince/Projects/can-lang/tests/integration/browser/package.json)

**Notes:** No unchanged Playwright script may be invoked by a Can wrapper as migration evidence. Generic browser infrastructure may provide actions and observations; Can chooses the sequence, expectations, engine profile and verdict.

## BROWSER-050

**grid.mjs** — [tests/integration/browser/grid.mjs](/Users/vince/Projects/can-lang/tests/integration/browser/grid.mjs:14)

Kind: `scenario`. Proposed disposition: `migrate`. Replacement obligation: `CAN-BROWSER-050`. Evidence: pending.

**Protects**

- Live grid: boot/client identity/query edges, money exactness, row add/move/remove and focus, blocked/rejected saves and Enter/real-keystroke behavior.
- Slow/midflight/single-flight saves, offline and busy replay, truncated/garbage after-commit replay, stale conflict keep/adopt.
- Detached/disposed silence, denied/unavailable loads, reload/no durability, navigation, duplicate IDs/node bound and network cleanliness.
- Named old checks/legs to account for: boot-loads, client-identity, query-edges, edit-totals, money-exact, add-line-focus, move-row-focus, remove-row-focus, blocked-save, rejected-422, enter-saves, real-keystrokes, slow-save-pending, midflight-press-single-grid, single-flight, offline-replay, busy-replay, busy-replay-commit, truncate-replay, garbage-replay, stale-conflict, conflict-keep-adopt, detached-click-silent, disposed-response-silent, ghost-denied, unavailable-load, denied-load, reload-no-durability, navigation-stable, no-duplicate-ids, node-count-bounded, network-ledger-clean.

**Current observations:** DOM/focus/selection, intercepted HTTP request/response bytes and timing, independent DB rows via Go consumer, console/page faults

**Variants:** Chromium/WebKit/Firefox normal/slow/offline/busy/corrupt responses DOM disposal/reload

**Environment/selection gates:** Playwright/browser engines paired invoice server and SQLite

**Capabilities:** BROWSER CAN DB HTTP WIRE CLOCK WORK

**Proposed Can replacement:** Author ordinary Can case/helper logic for Live grid: boot/client identity/query edges, money exactness, row add/move/remove and focus, blocked/rejected saves and Enter/real-keystroke behavior.; Slow/midflight/single-flight saves, offline and busy replay, truncated/garbage after-commit replay, stale conflict keep/adopt. Use generic native browser/build/database mechanics to return observations; Can applies all case expectations and reporting.

**Old harness may be deleted when:** Retire authored host source only after every protected facet has passing Can-owned evidence in required environments, its active callers and CI paths are rewired, and historical references are preserved or updated; confirm no test-specific scenario or verdict remains in native infrastructure.

**Related source:** [tests/integration/browser/bun.lock](/Users/vince/Projects/can-lang/tests/integration/browser/bun.lock); [tests/integration/browser/package.json](/Users/vince/Projects/can-lang/tests/integration/browser/package.json); [tests/integration/gate5_frontend_test.go](/Users/vince/Projects/can-lang/tests/integration/gate5_frontend_test.go); [tests/integration/testdata/invoice/driver.ts](/Users/vince/Projects/can-lang/tests/integration/testdata/invoice/driver.ts)

**Notes:** No unchanged Playwright script may be invoked by a Can wrapper as migration evidence. Generic browser infrastructure may provide actions and observations; Can chooses the sequence, expectations, engine profile and verdict. Current page.evaluate and callback/interception behavior requires generic browser mechanisms for DOM property/event probes, request interception, response injection or controlled rendezvous. Test-specific JavaScript callbacks embedded in an adapter would leave an authored host harness.

## BROWSER-051

**invoice-contract.mjs** — [tests/integration/browser/invoice-contract.mjs](/Users/vince/Projects/can-lang/tests/integration/browser/invoice-contract.mjs:8)

Kind: `scenario`. Proposed disposition: `migrate`. Replacement obligation: `CAN-BROWSER-051`. Evidence: pending.

**Protects**

- Contract-edit browser observer: boot revision/rows and paired script identity.
- Scenario save dispatches exactly one POST; invalid edit renders new error; budget edit blocks POST after network quiescence.
- Named old checks/legs to account for: boot-loads, client-identity, contract-save, contract-invalid, contract-budget.

**Current observations:** DOM, request/response ledger, CSP, page faults

**Variants:** Chromium save invalid budget

**Environment/selection gates:** Playwright/Chromium paired contract server

**Capabilities:** BROWSER HTTP DB WORK

**Proposed Can replacement:** Author ordinary Can case/helper logic for Contract-edit browser observer: boot revision/rows and paired script identity.; Scenario save dispatches exactly one POST; invalid edit renders new error; budget edit blocks POST after network quiescence. Use generic native browser/build/database mechanics to return observations; Can applies all case expectations and reporting.

**Old harness may be deleted when:** Retire authored host source only after every protected facet has passing Can-owned evidence in required environments, its active callers and CI paths are rewired, and historical references are preserved or updated; confirm no test-specific scenario or verdict remains in native infrastructure.

**Related source:** [tests/integration/browser/bun.lock](/Users/vince/Projects/can-lang/tests/integration/browser/bun.lock); [tests/integration/browser/package.json](/Users/vince/Projects/can-lang/tests/integration/browser/package.json); [tests/integration/invoice_contract_test.go](/Users/vince/Projects/can-lang/tests/integration/invoice_contract_test.go)

**Notes:** No unchanged Playwright script may be invoked by a Can wrapper as migration evidence. Generic browser infrastructure may provide actions and observations; Can chooses the sequence, expectations, engine profile and verdict. Current page.evaluate and callback/interception behavior requires generic browser mechanisms for DOM property/event probes, request interception, response injection or controlled rendezvous. Test-specific JavaScript callbacks embedded in an adapter would leave an authored host harness. The script accepts three engine names, but invoice_contract_test.go:382 currently invokes only Chromium.

## BROWSER-052

**invoice.mjs** — [tests/integration/browser/invoice.mjs](/Users/vince/Projects/can-lang/tests/integration/browser/invoice.mjs:16)

Kind: `scenario`. Proposed disposition: `migrate`. Replacement obligation: `CAN-BROWSER-052`. Evidence: pending.

**Protects**

- Live invoice form/HTMX: pinned scripts and policy; rendered inputs/escaped hostile content; 200/422/409/503/403 swaps preserve draft and focus.
- Missing target before/during, remount and restored/final save; reject OOB/partial/control headers and redirects; no outbound requests.
- Named old checks/legs to account for: htmx-loaded, pinned-scripts, htmx-config-policy, form-renders, save-200-swap, validation-422-swap, draft-and-focus-preserved, stale-409-swap, busy-503-swap, denied-403-swap, target-absence-stable, remount-stable, target-restored-save, missing-target-during, final-save, redirect-rejected, no-external-requests.

**Current observations:** DOM and focus, network/response interception, browser console/page faults and DB commit via Go consumer

**Variants:** Chromium/WebKit/Firefox five statuses missing target/remount hostile response injection

**Environment/selection gates:** Playwright/browser engines paired invoice server and SQLite

**Capabilities:** BROWSER HTTP WIRE FAULT DB WORK

**Proposed Can replacement:** Author ordinary Can case/helper logic for Live invoice form/HTMX: pinned scripts and policy; rendered inputs/escaped hostile content; 200/422/409/503/403 swaps preserve draft and focus.; Missing target before/during, remount and restored/final save; reject OOB/partial/control headers and redirects; no outbound requests. Use generic native browser/build/database mechanics to return observations; Can applies all case expectations and reporting.

**Old harness may be deleted when:** Retire authored host source only after every protected facet has passing Can-owned evidence in required environments, its active callers and CI paths are rewired, and historical references are preserved or updated; confirm no test-specific scenario or verdict remains in native infrastructure.

**Related source:** [tests/integration/browser/bun.lock](/Users/vince/Projects/can-lang/tests/integration/browser/bun.lock); [tests/integration/browser/package.json](/Users/vince/Projects/can-lang/tests/integration/browser/package.json); [tests/integration/gate5_frontend_test.go](/Users/vince/Projects/can-lang/tests/integration/gate5_frontend_test.go); [tests/integration/invoice_test.go](/Users/vince/Projects/can-lang/tests/integration/invoice_test.go); [tests/integration/up23_verdict_test.go](/Users/vince/Projects/can-lang/tests/integration/up23_verdict_test.go)

**Notes:** No unchanged Playwright script may be invoked by a Can wrapper as migration evidence. Generic browser infrastructure may provide actions and observations; Can chooses the sequence, expectations, engine profile and verdict. Current page.evaluate and callback/interception behavior requires generic browser mechanisms for DOM property/event probes, request interception, response injection or controlled rendezvous. Test-specific JavaScript callbacks embedded in an adapter would leave an authored host harness.

## BROWSER-053

**sha256-shim.mjs** — [tests/integration/browser/sha256-shim.mjs](/Users/vince/Projects/can-lang/tests/integration/browser/sha256-shim.mjs:75)

Kind: `support`. Proposed disposition: `historical_candidate`. Replacement obligation: `CAN-BROWSER-053`. Evidence: pending.

**Protects**

- Historical test bundler synchronous SHA-256 createHash subset; independent digest-vector fidelity was required for the old grid bundle.
- Audit any remaining active use before retiring this test-only semantic shim.

**Current observations:** digest output against independent vectors if retained

**Variants:** sha256 string input hex output

**Environment/selection gates:** Bun bundler

**Capabilities:** NATIVE FILES DOC

**Proposed Can replacement:** Retain only as historical evidence if the active-caller audit confirms no current coverage need; move any still-required digest/bundle qualification into Can-owned scenarios with generic native build observations.

**Old harness may be deleted when:** Retire authored host source only after every protected facet has passing Can-owned evidence in required environments, its active callers and CI paths are rewired, and historical references are preserved or updated; confirm no test-specific scenario or verdict remains in native infrastructure.

**Related source:** [tests/integration/browser/build-grid.mjs](/Users/vince/Projects/can-lang/tests/integration/browser/build-grid.mjs); [tests/baseline/t27_release_audit_test.go](/Users/vince/Projects/can-lang/tests/baseline/t27_release_audit_test.go)

**Notes:** No active tracked integration Go caller found by repository-wide reference search; historical docs/evidence and the baseline T27 source-presence check still reference this file. Retirement is a candidate only after those callers/evidence roles are reconciled.

## BROWSER-054

**w1-grid.mjs** — [tests/integration/browser/w1-grid.mjs](/Users/vince/Projects/can-lang/tests/integration/browser/w1-grid.mjs:9)

Kind: `scenario`. Proposed disposition: `migrate`. Replacement obligation: `CAN-BROWSER-054`. Evidence: pending.

**Protects**

- Grid contract: boot/client/generation identity, action headers, missing-id/foreign page denials and unknown 404.
- Caret preservation through edits; stale generation save/load prompt and refresh resolution; clean page fault record.
- Named old checks/legs to account for: boot-loads, client-identity, generation-slot, header-on-actions, missing-id-denied, page-denied, unknown-path-404, grid-caret, mismatch-save-prompt, mismatch-load-prompt, refresh-resolves, no-page-faults.

**Current observations:** DOM/selection, intercepted request/response ledger, page faults

**Variants:** Chromium/WebKit/Firefox denied IDs generation mismatch

**Environment/selection gates:** Playwright/browser engines paired invoice server

**Capabilities:** BROWSER HTTP WIRE CHILD WORK

**Proposed Can replacement:** Author ordinary Can case/helper logic for Grid contract: boot/client/generation identity, action headers, missing-id/foreign page denials and unknown 404.; Caret preservation through edits; stale generation save/load prompt and refresh resolution; clean page fault record. Use generic native browser/build/database mechanics to return observations; Can applies all case expectations and reporting.

**Old harness may be deleted when:** Retire authored host source only after every protected facet has passing Can-owned evidence in required environments, its active callers and CI paths are rewired, and historical references are preserved or updated; confirm no test-specific scenario or verdict remains in native infrastructure.

**Related source:** [tests/integration/browser/bun.lock](/Users/vince/Projects/can-lang/tests/integration/browser/bun.lock); [tests/integration/browser/package.json](/Users/vince/Projects/can-lang/tests/integration/browser/package.json); [tests/integration/c06_test.go](/Users/vince/Projects/can-lang/tests/integration/c06_test.go)

**Notes:** No unchanged Playwright script may be invoked by a Can wrapper as migration evidence. Generic browser infrastructure may provide actions and observations; Can chooses the sequence, expectations, engine profile and verdict.

## BROWSER-055

**invoice/driver.ts** — [tests/integration/testdata/invoice/driver.ts](/Users/vince/Projects/can-lang/tests/integration/testdata/invoice/driver.ts:20)

Kind: `support`. Proposed disposition: `replace_support`. Replacement obligation: `CAN-BROWSER-055`. Evidence: pending.

**Protects**

- Setup schema and seed deterministic sessions, memberships, invoices and lines including hostile text; independent readback stringifies exact integer values and replay rows.
- Can-selected mutations: touch replay timestamp, drop allowlisted table, read revision, revoke session, add/remove membership and inject/delete lines.

**Current observations:** independent SQLite queries/rows and mutation results

**Variants:** setup seed inspect touch-replay fault revision revoke member/unmember inject-line/delete-line

**Environment/selection gates:** disposable SQLite database

**Capabilities:** DB NATIVE FILES WORK FAULT

**Proposed Can replacement:** Can suite helpers select seed, readback, mutation and teardown actions and compare results; native database mechanics expose SQL execution and values without selecting these case modes or deciding pass/fail.

**Old harness may be deleted when:** Remove this TypeScript driver only after every named mode has a Can-owned caller/oracle or documented fixture role, all cross-family consumers are migrated, and independent database observation and cleanup evidence pass.

**Related source:** [tests/integration/browser_controls_test.go](/Users/vince/Projects/can-lang/tests/integration/browser_controls_test.go); [tests/integration/c06_test.go](/Users/vince/Projects/can-lang/tests/integration/c06_test.go); [tests/integration/gate3_matrix_test.go](/Users/vince/Projects/can-lang/tests/integration/gate3_matrix_test.go); [tests/integration/gate4_fault_test.go](/Users/vince/Projects/can-lang/tests/integration/gate4_fault_test.go); [tests/integration/gate5_frontend_test.go](/Users/vince/Projects/can-lang/tests/integration/gate5_frontend_test.go); [tests/integration/invoice_contract_test.go](/Users/vince/Projects/can-lang/tests/integration/invoice_contract_test.go); [tests/integration/invoice_test.go](/Users/vince/Projects/can-lang/tests/integration/invoice_test.go); [tests/integration/up23_verdict_test.go](/Users/vince/Projects/can-lang/tests/integration/up23_verdict_test.go)

**Notes:** Direct source consumers are browser_controls_test.go:157, c06_test.go:556, gate3_matrix_test.go:296, gate4_fault_test.go:312, gate5_frontend_test.go:1140, invoice_contract_test.go:504-1237, invoice_test.go:148/1129, and up23_verdict_test.go:85. These share an external SQLite oracle; preserve independent readback through generic mechanics while Can chooses fixture mutations and assertions.

## BROWSER-056

**applications/seed-driver.ts** — [tests/integration/testdata/applications/seed-driver.ts](/Users/vince/Projects/can-lang/tests/integration/testdata/applications/seed-driver.ts:8)

Kind: `support`. Proposed disposition: `replace_support`. Replacement obligation: `CAN-BROWSER-056`. Evidence: pending.

**Protects**

- Validate 12-statement fixed seed shape, setup PostgreSQL 17 tables/rows and teardown owned tables; report server version and three row counts.

**Current observations:** independent PostgreSQL version and row counts

**Variants:** setup teardown

**Environment/selection gates:** CAN_TEST_POSTGRES_URL disposable PostgreSQL

**Capabilities:** DB ENV FILES WORK

**Proposed Can replacement:** Can suite helpers select seed, readback, mutation and teardown actions and compare results; native database mechanics expose SQL execution and values without selecting these case modes or deciding pass/fail.

**Old harness may be deleted when:** Remove this TypeScript driver only after every named mode has a Can-owned caller/oracle or documented fixture role, all cross-family consumers are migrated, and independent database observation and cleanup evidence pass.

**Related source:** [tests/integration/applications_test.go](/Users/vince/Projects/can-lang/tests/integration/applications_test.go)

**Notes:** This is authored test setup/observation policy, not merely passive SQL input. Keep static schema/seed data as fixtures if useful. No TypeScript driver wrapped by Can may count as migration.
