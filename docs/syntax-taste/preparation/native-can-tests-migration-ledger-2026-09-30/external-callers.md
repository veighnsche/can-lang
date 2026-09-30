# Native Can migration ledger: external-callers

Status: **proposed replacements; implementation and parity evidence pending**.

[Overview](../../native-can-tests-migration-ledger-2026-09-30.md) · [Machine-readable rows](external-callers.json)

Every deletion condition is conjunctive with the overview's common gate. It covers **all** protected facets, variants and delegated oracles, even where a row's short replacement sentence mentions only its first facet. Row IDs name coverage obligations, not one-to-one implementation files.

## EDGE-001

**main / native qualification** — [distribution/qualify.py](/Users/vince/Projects/can-lang/distribution/qualify.py:67)

Kind: `caller`. Proposed disposition: `rewire`. Replacement obligation: `CAN-EDGE-001`. Evidence: pending.

**Protects**

- Target OS/architecture/minimum version and glibc, archive size/hash and extracted runtime digest
- No ambient configuration/install/PATH runtime; offline isolation verified, never assumed
- Native API/behavior and deliberate rejection results combined into provenance-bound report

**Current observations:** platform/interface observations; archive bytes; sandbox/unshare; native.ts/native.test.ts exit/report results

**Variants:** None recorded.

**Environment/selection gates:** None recorded.

**Capabilities:** SUITE PROC WORK ENV ARCHIVE NATIVE FILES

**Proposed Can replacement:** Can owns qualification selection, expected pins, isolation requirements, probe expectations and aggregate verdict. Generic native archive/platform operations return raw facts.

**Old harness may be deleted when:** Rewire this caller after equivalent Can-owned scenarios and reports are qualified; remove only its test policy and obsolete invocations, retaining ordinary product/provisioning mechanics.

**Related source:** [tests/conformance/native.ts](/Users/vince/Projects/can-lang/tests/conformance/native.ts); [tests/conformance/native.test.ts](/Users/vince/Projects/can-lang/tests/conformance/native.test.ts); [.github/workflows/verifier.yml](/Users/vince/Projects/can-lang/.github/workflows/verifier.yml); [distribution/linux/smoke.sh](/Users/vince/Projects/can-lang/distribution/linux/smoke.sh)

**Notes:** Contains test policy outside /tests; replacing just two copied filenames would not eliminate it. HISTORY-143 records the same edge from the native qualification subject; EDGE-001 owns caller-specific isolation/provenance/aggregation obligations. These are linked views, not two independent coverage wins.

## EDGE-002

**installed artifact smoke** — [distribution/linux/smoke.sh](/Users/vince/Projects/can-lang/distribution/linux/smoke.sh:1)

Kind: `caller`. Proposed disposition: `rewire`. Replacement obligation: `CAN-EDGE-002`. Evidence: pending.

**Protects**

- Installed Linux runtime identity, offline native qualification, real process/files/crypto/SQLite example outputs and build determinism

**Current observations:** shell/embedded Python assertions; installed CLI output and JSON

**Variants:** None recorded.

**Environment/selection gates:** None recorded.

**Capabilities:** SUITE PROC ENV ARCHIVE CAN DB FILES WORK

**Proposed Can replacement:** Consolidate with the Can Linux installed-artifact profile; shell entry may only provision/invoke/propagate its result.

**Old harness may be deleted when:** Rewire this caller after equivalent Can-owned scenarios and reports are qualified; remove only its test policy and obsolete invocations, retaining ordinary product/provisioning mechanics.

**Related source:** [tests/integration/linux_distribution_test.go](/Users/vince/Projects/can-lang/tests/integration/linux_distribution_test.go); [distribution/qualify.py](/Users/vince/Projects/can-lang/distribution/qualify.py); [distribution/linux/smoke/sqlite-seed.ts](/Users/vince/Projects/can-lang/distribution/linux/smoke/sqlite-seed.ts); [tests/integration/testdata/sql/sqlite-driver.ts](/Users/vince/Projects/can-lang/tests/integration/testdata/sql/sqlite-driver.ts)

**Notes:** Fixed WORK deletion must become explicit task ownership. No independent Python expected-value checks remain in launcher.

## EDGE-003

**PostgreSQL smoke** — [distribution/linux/postgres.sh](/Users/vince/Projects/can-lang/distribution/linux/postgres.sh:1)

Kind: `caller`. Proposed disposition: `rewire`. Replacement obligation: `CAN-EDGE-003`. Evidence: pending.

**Protects**

- Installed sidecar and PostgreSQL-17 row/version roundtrip expectations

**Current observations:** shell executable checks plus embedded Python assertions on pg-driver JSON

**Variants:** None recorded.

**Environment/selection gates:** None recorded.

**Capabilities:** SUITE PROC ENV DB WORK

**Proposed Can replacement:** Use same Can installed-PostgreSQL scenario as Linux Go test, preserving native DB observations.

**Old harness may be deleted when:** Rewire this caller after equivalent Can-owned scenarios and reports are qualified; remove only its test policy and obsolete invocations, retaining ordinary product/provisioning mechanics.

**Related source:** [distribution/linux/smoke/pg-driver.ts](/Users/vince/Projects/can-lang/distribution/linux/smoke/pg-driver.ts); [tests/integration/linux_distribution_test.go](/Users/vince/Projects/can-lang/tests/integration/linux_distribution_test.go)

**Notes:** 

## EDGE-004

**native PostgreSQL roundtrip** — [distribution/linux/smoke/pg-driver.ts](/Users/vince/Projects/can-lang/distribution/linux/smoke/pg-driver.ts:1)

Kind: `caller`. Proposed disposition: `migrate`. Replacement obligation: `CAN-EDGE-004`. Evidence: pending.

**Protects**

- Direct native create/insert/select/drop, PostgreSQL version, row count/content and pool closure

**Current observations:** Bun.SQL facts and built-in host-side expectations

**Variants:** None recorded.

**Environment/selection gates:** None recorded.

**Capabilities:** DB NATIVE WORK

**Proposed Can replacement:** Move schema sequence, version/row expectations and cleanup policy into Can; independent native DB binding returns facts without table-specific verdicts.

**Old harness may be deleted when:** Rewire this caller after equivalent Can-owned scenarios and reports are qualified; remove only its test policy and obsolete invocations, retaining ordinary product/provisioning mechanics.

**Related source:** [distribution/linux/postgres.sh](/Users/vince/Projects/can-lang/distribution/linux/postgres.sh); [tests/integration/linux_distribution_test.go](/Users/vince/Projects/can-lang/tests/integration/linux_distribution_test.go)

**Notes:** Contains expectations internally as well as in its Go caller; needs full policy migration.

## EDGE-005

**native SQLite seed** — [distribution/linux/smoke/sqlite-seed.ts](/Users/vince/Projects/can-lang/distribution/linux/smoke/sqlite-seed.ts:1)

Kind: `caller`. Proposed disposition: `migrate`. Replacement obligation: `CAN-EDGE-005`. Evidence: pending.

**Protects**

- Native insertion/readback of the smoke row and pool closure

**Current observations:** Bun.SQL row count/body check then JSON

**Variants:** None recorded.

**Environment/selection gates:** None recorded.

**Capabilities:** DB NATIVE WORK

**Proposed Can replacement:** Can chooses insert/query and checks row facts through independent DB observations.

**Old harness may be deleted when:** Rewire this caller after equivalent Can-owned scenarios and reports are qualified; remove only its test policy and obsolete invocations, retaining ordinary product/provisioning mechanics.

**Related source:** [distribution/linux/smoke.sh](/Users/vince/Projects/can-lang/distribution/linux/smoke.sh); [tests/integration/linux_distribution_test.go](/Users/vince/Projects/can-lang/tests/integration/linux_distribution_test.go)

**Notes:** 

## EDGE-006

**native qualification / browser provision / UP23 / whole-repository gate** — [.github/workflows/verifier.yml](/Users/vince/Projects/can-lang/.github/workflows/verifier.yml:31)

Kind: `caller`. Proposed disposition: `rewire`. Replacement obligation: `CAN-EDGE-006`. Evidence: pending.

**Protects**

- Runs native qualification and browser provision before integration gates
- Creates UP23 candidate-bound verdict for consumers; retains unit/runtime checks independent from migrated integration cases

**Current observations:** workflow commands/env and exit statuses; CAN_UP23 output consumed later

**Variants:** None recorded.

**Environment/selection gates:** None recorded.

**Capabilities:** SUITE ENV WORK BROWSER BUILD

**Proposed Can replacement:** Invoke Can qualification and integration profiles, provision selected engines/services generically, hand off compact source-bound evidence and preserve independent unit gates.

**Old harness may be deleted when:** Rewire this caller after equivalent Can-owned scenarios and reports are qualified; remove only its test policy and obsolete invocations, retaining ordinary product/provisioning mechanics.

**Related source:** [distribution/qualify.py](/Users/vince/Projects/can-lang/distribution/qualify.py); [tests/integration/up23_verdict_test.go](/Users/vince/Projects/can-lang/tests/integration/up23_verdict_test.go); [tests/integration/browser/package.json](/Users/vince/Projects/can-lang/tests/integration/browser/package.json)

**Notes:** Whole-repository Go/Bun gate invocations are not themselves permission to rewrite every independent unit test. Named runtime oracles delegated by /tests are explicitly tracked separately.

## EDGE-007

**release qualification** — [.github/workflows/release-qualify.yml](/Users/vince/Projects/can-lang/.github/workflows/release-qualify.yml:32)

Kind: `caller`. Proposed disposition: `rewire`. Replacement obligation: `CAN-EDGE-007`. Evidence: pending.

**Protects**

- Build/release/install/runtime-check and in-scope release/install/update cases with offline execution

**Current observations:** host shell/Python pinned-runtime comparison plus selected Go test invocations

**Variants:** None recorded.

**Environment/selection gates:** None recorded.

**Capabilities:** SUITE ENV ARCHIVE WORK BUILD

**Proposed Can replacement:** Can owns release-smoke expectations and migrated cases; CI provisions and invokes profile, while independent distribution unit tests retain their own gate.

**Old harness may be deleted when:** Rewire this caller after equivalent Can-owned scenarios and reports are qualified; remove only its test policy and obsolete invocations, retaining ordinary product/provisioning mechanics.

**Related source:** [tests/integration/release_test.go](/Users/vince/Projects/can-lang/tests/integration/release_test.go); [tests/integration/distribution_test.go](/Users/vince/Projects/can-lang/tests/integration/distribution_test.go)

**Notes:** 

## EDGE-008

**fresh emit and strict TypeScript gate** — [.github/workflows/tsc.yml](/Users/vince/Projects/can-lang/.github/workflows/tsc.yml:42)

Kind: `caller`. Proposed disposition: `rewire`. Replacement obligation: `CAN-EDGE-008`. Evidence: pending.

**Protects**

- Every maintained project emits fresh checked output consumed by strict tsc gate

**Current observations:** CAN_FRESH_EMIT_DIR + TestStdlibMaintained; tsc exit; uploaded outputs

**Variants:** None recorded.

**Environment/selection gates:** None recorded.

**Capabilities:** SUITE CAN BUILD PROC FILES WORK

**Proposed Can replacement:** Can maintained-project scenario owns current compilation/coverage expectations; product TypeScript checker may still be invoked on resulting artifacts with bounded staging and cleanup.

**Old harness may be deleted when:** Rewire this caller after equivalent Can-owned scenarios and reports are qualified; remove only its test policy and obsolete invocations, retaining ordinary product/provisioning mechanics.

**Related source:** [tests/integration/stdlib_test.go](/Users/vince/Projects/can-lang/tests/integration/stdlib_test.go)

**Notes:** Current upload retains full emitted tree; redesign default retention to compact diagnostics and explicitly requested expiring heavy evidence. Do not keep old generated layouts as compatibility requirements.

## EDGE-009

**canlcCache / TestMain / canlcBinary** — [host/conformance/admission_test.go](/Users/vince/Projects/can-lang/host/conformance/admission_test.go:237)

Kind: `caller`. Proposed disposition: `rewire`. Replacement obligation: `CAN-EDGE-009`. Evidence: pending.

**Protects**

- External consumer needs run-owned temporary compiler and cleanup failure propagation

**Current observations:** tempcache import, once-per-run Go build and TestMain cleanup

**Variants:** None recorded.

**Environment/selection gates:** None recorded.

**Capabilities:** WORK PROC BUILD

**Proposed Can replacement:** Keep this independent unit suite outside oracle migration unless separately scoped; replace its dependency on deleted /tests support with allowed generic resource infrastructure.

**Old harness may be deleted when:** Rewire this caller after equivalent Can-owned scenarios and reports are qualified; remove only its test policy and obsolete invocations, retaining ordinary product/provisioning mechanics.

**Related source:** [tests/support/tempcache/cache.go](/Users/vince/Projects/can-lang/tests/support/tempcache/cache.go)

**Notes:** Only consumer integration is required here. Relocating generic resource mechanics may be valid; relocating any suite policy/expectations would not be.

## EDGE-010

**browser dependency consumer** — [host/conformance/live_test.go](/Users/vince/Projects/can-lang/host/conformance/live_test.go:28)

Kind: `caller`. Proposed disposition: `rewire`. Replacement obligation: `CAN-EDGE-010`. Evidence: pending.

**Protects**

- Independent live host suite resolves shared Playwright install and helper

**Current observations:** NODE_PATH and tests/integration/browser dependency lookup

**Variants:** None recorded.

**Environment/selection gates:** None recorded.

**Capabilities:** BROWSER ENV WORK

**Proposed Can replacement:** Update generic driver provisioning/import paths before retiring browser support; preserve this independent suite coverage without using it as replacement for /tests scenarios.

**Old harness may be deleted when:** Rewire this caller after equivalent Can-owned scenarios and reports are qualified; remove only its test policy and obsolete invocations, retaining ordinary product/provisioning mechanics.

**Related source:** [host/conformance/live/storage-clipboard.mjs](/Users/vince/Projects/can-lang/host/conformance/live/storage-clipboard.mjs); [tests/integration/browser/firefox-remote.mjs](/Users/vince/Projects/can-lang/tests/integration/browser/firefox-remote.mjs); [tests/integration/browser/package.json](/Users/vince/Projects/can-lang/tests/integration/browser/package.json)

**Notes:** 

## EDGE-011

**vendor B browser dependency consumer** — [host/conformance/live_vendor_b_test.go](/Users/vince/Projects/can-lang/host/conformance/live_vendor_b_test.go:28)

Kind: `caller`. Proposed disposition: `rewire`. Replacement obligation: `CAN-EDGE-011`. Evidence: pending.

**Protects**

- Independent vendor-B live suite resolves shared browser dependency/helper

**Current observations:** shared Playwright path and remote Firefox import

**Variants:** None recorded.

**Environment/selection gates:** None recorded.

**Capabilities:** BROWSER ENV WORK

**Proposed Can replacement:** Rewire shared native driver mechanics while leaving independent host cases separately identified.

**Old harness may be deleted when:** Rewire this caller after equivalent Can-owned scenarios and reports are qualified; remove only its test policy and obsolete invocations, retaining ordinary product/provisioning mechanics.

**Related source:** [host/conformance/live/vendor-b.mjs](/Users/vince/Projects/can-lang/host/conformance/live/vendor-b.mjs); [tests/integration/browser/firefox-remote.mjs](/Users/vince/Projects/can-lang/tests/integration/browser/firefox-remote.mjs)

**Notes:** 

## EDGE-012

**Linux provisioning/launcher** — [distribution/linux/run.sh](/Users/vince/Projects/can-lang/distribution/linux/run.sh:1)

Kind: `caller`. Proposed disposition: `rewire`. Replacement obligation: `CAN-EDGE-012`. Evidence: pending.

**Protects**

- Build/provision and launch Linux qualification entrypoint

**Current observations:** container invocation of smoke.sh

**Variants:** None recorded.

**Environment/selection gates:** None recorded.

**Capabilities:** ENV WORK PROC

**Proposed Can replacement:** Retain generic target provisioning but point to Can-owned qualification; account for owned work/cleanup.

**Old harness may be deleted when:** Rewire this caller after equivalent Can-owned scenarios and reports are qualified; remove only its test policy and obsolete invocations, retaining ordinary product/provisioning mechanics.

**Related source:** [distribution/linux/smoke.sh](/Users/vince/Projects/can-lang/distribution/linux/smoke.sh)

**Notes:** 

## EDGE-013

**qualification prerequisites and browser-path preflight** — [distribution/h12/window-check.sh](/Users/vince/Projects/can-lang/distribution/h12/window-check.sh:84)

Kind: `caller`. Proposed disposition: `rewire`. Replacement obligation: `CAN-EDGE-013`. Evidence: pending.

**Protects**

- Preflight currently assumes Python qualification and old browser directory; checks selected archive identity and isolation

**Current observations:** tool/path probes; embedded archive check; pass/block summaries

**Variants:** None recorded.

**Environment/selection gates:** None recorded.

**Capabilities:** ENV PROC FILES WORK

**Proposed Can replacement:** Reconcile prerequisites with selected Can profiles; move in-scope archive/isolation expectations to Can or share their product observation path; retain generic provision checks with explicit role.

**Old harness may be deleted when:** Rewire this caller after equivalent Can-owned scenarios and reports are qualified; remove only its test policy and obsolete invocations, retaining ordinary product/provisioning mechanics.

**Related source:** [distribution/qualify.py](/Users/vince/Projects/can-lang/distribution/qualify.py); [tests/integration/browser/package.json](/Users/vince/Projects/can-lang/tests/integration/browser/package.json)

**Notes:** Do not require obsolete Python simply because the retired harness needed it; current product target/pin obligations remain.

## EDGE-014

**operator commands** — [distribution/README.md](/Users/vince/Projects/can-lang/distribution/README.md:15)

Kind: `caller`. Proposed disposition: `rewire`. Replacement obligation: `CAN-EDGE-014`. Evidence: pending.

**Protects**

- Documented entrypoints for native and integration qualification

**Current observations:** README command references

**Variants:** None recorded.

**Environment/selection gates:** None recorded.

**Capabilities:** DOC

**Proposed Can replacement:** Update instructions to the Can suite commands and accurate required environment/partial-run semantics.

**Old harness may be deleted when:** Rewire this caller after equivalent Can-owned scenarios and reports are qualified; remove only its test policy and obsolete invocations, retaining ordinary product/provisioning mechanics.

**Related source:** [distribution/qualify.py](/Users/vince/Projects/can-lang/distribution/qualify.py); [tests/integration](/Users/vince/Projects/can-lang/tests/integration)

**Notes:** Documentation itself is not an executable harness.

## EDGE-015

**Linux operator commands** — [distribution/linux/README.md](/Users/vince/Projects/can-lang/distribution/linux/README.md:38)

Kind: `caller`. Proposed disposition: `rewire`. Replacement obligation: `CAN-EDGE-015`. Evidence: pending.

**Protects**

- Linux smoke/PostgreSQL qualification reproductions

**Current observations:** documented smoke.sh and Go-test commands

**Variants:** None recorded.

**Environment/selection gates:** None recorded.

**Capabilities:** DOC

**Proposed Can replacement:** Point to consolidated Can Linux profiles and separate provisioning from qualification evidence.

**Old harness may be deleted when:** Rewire this caller after equivalent Can-owned scenarios and reports are qualified; remove only its test policy and obsolete invocations, retaining ordinary product/provisioning mechanics.

**Related source:** [distribution/linux/smoke.sh](/Users/vince/Projects/can-lang/distribution/linux/smoke.sh); [distribution/linux/postgres.sh](/Users/vince/Projects/can-lang/distribution/linux/postgres.sh)

**Notes:** 

## EDGE-016

**shared browser installation instructions** — [host/conformance/live/README.md](/Users/vince/Projects/can-lang/host/conformance/live/README.md:48)

Kind: `caller`. Proposed disposition: `rewire`. Replacement obligation: `CAN-EDGE-016`. Evidence: pending.

**Protects**

- External callers can install/find shared browser machinery

**Current observations:** old tests/integration/browser path in operator commands

**Variants:** None recorded.

**Environment/selection gates:** None recorded.

**Capabilities:** DOC BROWSER ENV

**Proposed Can replacement:** Update location/ownership and cleanup documentation when shared browser machinery is replaced.

**Old harness may be deleted when:** Rewire this caller after equivalent Can-owned scenarios and reports are qualified; remove only its test policy and obsolete invocations, retaining ordinary product/provisioning mechanics.

**Related source:** [tests/integration/browser/package.json](/Users/vince/Projects/can-lang/tests/integration/browser/package.json)

**Notes:** 
