# Native Can migration ledger: lifecycle

Status: **proposed replacements; implementation and parity evidence pending**.

[Overview](../../native-can-tests-migration-ledger-2026-09-30.md) · [Machine-readable rows](lifecycle.json)

Every deletion condition is conjunctive with the overview's common gate. It covers **all** protected facets, variants and delegated oracles, even where a row's short replacement sentence mentions only its first facet. Row IDs name coverage obligations, not one-to-one implementation files.

## LIFE-001

**TestGate3ContractEdits** — [tests/integration/gate3_matrix_test.go](/Users/vince/Projects/can-lang/tests/integration/gate3_matrix_test.go:99)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-LIFE-001`. Evidence: pending.

**Protects**

- Shared-route edits invalidate dependency lock; mismatched mount handlers and renamed record fields identify stale uses
- Bogus active error registry and dangling predecessor chains reject under both assert and build

**Current observations:** mutate exactly one source site; compiler exit and named diagnostics

**Variants:** 5 edits x assert/build

**Environment/selection gates:** CAN_BUN_ARCHIVE required; absent archive currently skips staged bundled compiler/runtime, isolated homes; current offline helpers use macOS sandbox-exec

**Capabilities:** SUITE CAN PROC FILES WORK DIAG FAULT BUILD

**Proposed Can replacement:** Can stages each edit, runs candidate assert/build, and compares expected rejecting diagnostic evidence without retaining the Go edit table.

**Old harness may be deleted when:** Remove this host entry only after the Can replacement demonstrates every listed facet and required variant with independent observations; rewire its callers and retain shared helpers until their last consumer migrates.

**Related source:** [tests/integration/testdata/invoice/driver.ts](/Users/vince/Projects/can-lang/tests/integration/testdata/invoice/driver.ts)

**Notes:** 

## LIFE-002

**TestGate3RouteRebuildLive** — [tests/integration/gate3_matrix_test.go](/Users/vince/Projects/can-lang/tests/integration/gate3_matrix_test.go:269)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-LIFE-002`. Evidence: pending.

**Protects**

- Relocked route change produces stable rebuilt identity and updated emitted route
- Served form follows checked new URL; old route rejects; new route commits expected row

**Current observations:** two build IDs; emitted bytes; HTTP HTML/status; independent SQLite row inspection

**Variants:** old/new route; repeat build

**Environment/selection gates:** CAN_BUN_ARCHIVE required; absent archive currently skips staged bundled compiler/runtime, isolated homes; current offline helpers use macOS sandbox-exec free loopback ports and disposable SQLite; managed server and fd-3 credential snapshot

**Capabilities:** CAN BUILD WORK PROC CHILD HTTP DB FILES

**Proposed Can replacement:** Can edits/relocks source, compares two builds, serves output, posts old/new routes and checks raw stored revision/values.

**Old harness may be deleted when:** Remove this host entry only after the Can replacement demonstrates every listed facet and required variant with independent observations; rewire its callers and retain shared helpers until their last consumer migrates.

**Related source:** [tests/integration/testdata/invoice/driver.ts](/Users/vince/Projects/can-lang/tests/integration/testdata/invoice/driver.ts)

**Notes:** 

## LIFE-003

**TestGate3ServerMatrix** — [tests/integration/gate3_matrix_test.go](/Users/vince/Projects/can-lang/tests/integration/gate3_matrix_test.go:518)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-LIFE-003`. Evidence: pending.

**Protects**

- JSON save/load, exact totals, actor/tenant scope and stored line order
- Identical replay is one effect; changed/stale requests conflict; invalid fields do not write
- Foreign/missing/anonymous/revoked/ghost callers get nondisclosing 403; bad media/malformed/oversize requests reject
- Charset acceptance, raw GET-body non-effect, missing capture, store-outage 503 and recovery
- Keyed form reorder persists/renders positions; malformed order rejects without effect
- Served global noSwap and per-status error admission match current action contract
- SIGKILL during/after commit and identical retries converge to one revision/ledger effect

**Current observations:** status/headers/tagged JSON/HTML bytes; raw wire GET; native DB revision, lines and replay rows; restart outcomes

**Variants:** save/load authorization matrices JSON/form media and size failures mid-flight and lost-ack crashes

**Environment/selection gates:** CAN_BUN_ARCHIVE required; absent archive currently skips staged bundled compiler/runtime, isolated homes; current offline helpers use macOS sandbox-exec free loopback ports and disposable SQLite; managed server and fd-3 credential snapshot

**Capabilities:** CAN BUILD WORK CHILD HTTP WIRE DB FILES FAULT CLOCK

**Proposed Can replacement:** Can expresses named save/load/replay/form/fault/crash cases over compiled server, owns expected values and raw DB comparisons.

**Old harness may be deleted when:** Remove this host entry only after the Can replacement demonstrates every listed facet and required variant with independent observations; rewire its callers and retain shared helpers until their last consumer migrates.

**Related source:** [tests/integration/testdata/invoice/driver.ts](/Users/vince/Projects/can-lang/tests/integration/testdata/invoice/driver.ts)

**Notes:** File header mentions absence of per-element admission, but executable body explicitly requires hx-status 403/409/422/503; migrate actual current checks. Can split this composite test into stable case IDs.

## LIFE-004

**TestGate3AdapterMatrix** — [tests/integration/gate3_matrix_test.go](/Users/vince/Projects/can-lang/tests/integration/gate3_matrix_test.go:909)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-LIFE-004`. Evidence: pending.

**Protects**

- 404 and sorted Allow/405, malformed captures, encoded separators/dot normalization leave handler-entry evidence unchanged
- JSON/form media, syntax and byte/row limits reject before protected effects
- Success control advances revision/ledger once; fully written abandoned request still commits once; replay bytes stable and changes conflict
- Dropped replay table yields truthful 503 without revision change; restored identical request commits

**Current observations:** HTTP and raw socket traffic; revision plus ledger entry snapshots; independent DB fault injection

**Variants:** 4 unknown routes 7 wrong-method routes 4 malformed captures separator/self-dot/climb JSON/form media/budgets; 65 rows

**Environment/selection gates:** CAN_BUN_ARCHIVE required; absent archive currently skips staged bundled compiler/runtime, isolated homes; current offline helpers use macOS sandbox-exec free loopback ports and disposable SQLite; managed server and fd-3 credential snapshot

**Capabilities:** CAN BUILD WORK CHILD HTTP WIRE DB FAULT CLOCK

**Proposed Can replacement:** Can sends the route/transport matrix and compares pre/post entry evidence, then verifies success, abandoned-body and restore controls.

**Old harness may be deleted when:** Remove this host entry only after the Can replacement demonstrates every listed facet and required variant with independent observations; rewire its callers and retain shared helpers until their last consumer migrates.

**Related source:** [tests/integration/testdata/invoice/driver.ts](/Users/vince/Projects/can-lang/tests/integration/testdata/invoice/driver.ts)

**Notes:** 

## LIFE-005

**TestGate3Concurrency** — [tests/integration/gate3_matrix_test.go](/Users/vince/Projects/can-lang/tests/integration/gate3_matrix_test.go:1232)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-LIFE-005`. Evidence: pending.

**Protects**

- Eight distinct operations on one revision produce one winner/seven conflicts
- Six identical racing operations converge after allowed transient 503 to byte-identical replay and one shared effect
- Membership removal denies without disclosure; restore permits save
- Membership flaps racing saves permit at most one coherent authorized commit with matching ledger/revision; recovery control commits

**Current observations:** barrier-started HTTP requests; exact status/tag/body set; independent membership mutations and final rows

**Variants:** 8 distinct 6 identical membership absent/restored/flapping

**Environment/selection gates:** CAN_BUN_ARCHIVE required; absent archive currently skips staged bundled compiler/runtime, isolated homes; current offline helpers use macOS sandbox-exec free loopback ports and disposable SQLite; managed server and fd-3 credential snapshot

**Capabilities:** CAN BUILD WORK CHILD HTTP DB CLOCK

**Proposed Can replacement:** Can coordinates bounded simultaneous clients and membership changes, then checks outcome sets and DB invariants rather than relying on scheduling order.

**Old harness may be deleted when:** Remove this host entry only after the Can replacement demonstrates every listed facet and required variant with independent observations; rewire its callers and retain shared helpers until their last consumer migrates.

**Related source:** [tests/integration/testdata/invoice/driver.ts](/Users/vince/Projects/can-lang/tests/integration/testdata/invoice/driver.ts)

**Notes:** 

## LIFE-006

**TestGate3ReplayExpiry** — [tests/integration/gate3_matrix_test.go](/Users/vince/Projects/can-lang/tests/integration/gate3_matrix_test.go:1481)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-LIFE-006`. Evidence: pending.

**Protects**

- Revoked actor cannot recover saved outcomes via identical/changed/fresh requests or loads
- Unrevoked/live replay preserves exact bytes and retention stamp
- Expired replay conflicts and removes row; same operation with fresh base commits new revision
- Cleanup removes ancient replay but preserves near-boundary live rows

**Current observations:** signed-in requests, native session/replay timestamp changes, exact response bytes and row timestamps

**Variants:** revoked/unrevoked inside/outside retention cutoff fresh base after expiry

**Environment/selection gates:** CAN_BUN_ARCHIVE required; absent archive currently skips staged bundled compiler/runtime, isolated homes; current offline helpers use macOS sandbox-exec free loopback ports and disposable SQLite; managed server and fd-3 credential snapshot

**Capabilities:** CAN BUILD WORK CHILD HTTP DB CLOCK

**Proposed Can replacement:** Can sets explicit time-relative fixtures and checks authorization, replay, expiry and cleanup; retain existing exact-boundary model assertion.

**Old harness may be deleted when:** Remove this host entry only after the Can replacement demonstrates every listed facet and required variant with independent observations; rewire its callers and retain shared helpers until their last consumer migrates.

**Related source:** [tests/integration/testdata/invoice/driver.ts](/Users/vince/Projects/can-lang/tests/integration/testdata/invoice/driver.ts)

**Notes:** Live fixtures sit 60 seconds either side of a 300000ms window; exact edge belongs to save_replay_at_edge, not a timing-flaky live replacement.

## LIFE-007

**TestGate3RendererFault** — [tests/integration/gate3_matrix_test.go](/Users/vince/Projects/can-lang/tests/integration/gate3_matrix_test.go:1646)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-LIFE-007`. Evidence: pending.

**Protects**

- Post-commit unmintable row produces fixed non-disclosing 500 while committed rows and JSON read stay healthy
- Removing corrupt row restores rendered page
- Dropped lines table produces 503 alert/page without commit; identical save succeeds after restoration

**Current observations:** HTTP HTML/JSON plus independent corrupt-row insertion/deletion and revision reads

**Variants:** unmintable stored key drop-lines outage and recovery

**Environment/selection gates:** CAN_BUN_ARCHIVE required; absent archive currently skips staged bundled compiler/runtime, isolated homes; current offline helpers use macOS sandbox-exec free loopback ports and disposable SQLite; managed server and fd-3 credential snapshot

**Capabilities:** CAN BUILD WORK CHILD HTTP DB FAULT

**Proposed Can replacement:** Can introduces malformed stored data, checks separate HTML/JSON observations, and validates outage/recovery with row evidence.

**Old harness may be deleted when:** Remove this host entry only after the Can replacement demonstrates every listed facet and required variant with independent observations; rewire its callers and retain shared helpers until their last consumer migrates.

**Related source:** [tests/integration/testdata/invoice/driver.ts](/Users/vince/Projects/can-lang/tests/integration/testdata/invoice/driver.ts)

**Notes:** 

## LIFE-008

**TestGate3Lifecycle** — [tests/integration/gate3_matrix_test.go](/Users/vince/Projects/can-lang/tests/integration/gate3_matrix_test.go:1795)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-LIFE-008`. Evidence: pending.

**Protects**

- One pool-open/close call site; missing DB and occupied port fail before serving
- SIGTERM drains cleanly; restart preserves replay and accepts new writes
- SIGTERM during trickled body converges on retry to one effect
- Delegated buffered/lazy request token revocation and owned child lifetime obligations

**Current observations:** emitted call-site count; startup exit/status; SIGTERM wait; trickled HTTP body; SQLite rows; staged request-lifetime.test.ts

**Variants:** missing path/occupied port normal/inflight shutdown buffered/lazy/upgraded requests

**Environment/selection gates:** CAN_BUN_ARCHIVE required; absent archive currently skips staged bundled compiler/runtime, isolated homes; current offline helpers use macOS sandbox-exec free loopback ports and disposable SQLite; managed server and fd-3 credential snapshot

**Capabilities:** CAN BUILD WORK CHILD HTTP WIRE DB NATIVE FAULT CLOCK

**Proposed Can replacement:** Can splits pool/lifecycle/ingress/revocation cases, tests compiled server and ports, and replaces request-lifetime expectations with Can-owned checks over required raw token observations.

**Old harness may be deleted when:** Remove this host entry only after the Can replacement demonstrates every listed facet and required variant with independent observations; rewire its callers and retain shared helpers until their last consumer migrates.

**Related source:** [tests/integration/testdata/invoice/driver.ts](/Users/vince/Projects/can-lang/tests/integration/testdata/invoice/driver.ts); [runtime/test/request-lifetime.test.ts](/Users/vince/Projects/can-lang/runtime/test/request-lifetime.test.ts)

**Notes:** Exact emitted symbol spellings are incidental; retain single ownership/open-close behavior. Delegated host test success cannot remain the migrated oracle.

## LIFE-009

**TestGate4FaultMatrix** — [tests/integration/gate4_fault_test.go](/Users/vince/Projects/can-lang/tests/integration/gate4_fault_test.go:261)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-LIFE-009`. Evidence: pending.

**Protects**

- Installed Linux artifact vs staged development identity and applicable build determinism/no stray output
- Complete-body client disconnect commits once, identical replay succeeds and changed replay conflicts
- Trickled signed webhook delivery accepted once; SIGTERM drain/restart preserve duplicate/new-delivery semantics
- Native stream read/close succeeds and missing file fails diagnostically
- Delegated shutdown, owner, server, transport-owned/late and coordination behavior with explicit limitations

**Current observations:** artifact runtime identity; raw/trickled HTTP; native DB rows; child termination; stream CLI; named runtime suites

**Variants:** linux/amd64 installed release; other hosts staged disconnect/trickle/terminate/stream six delegated runtime suites; server suite conditional on openssl

**Environment/selection gates:** CAN_BUN_ARCHIVE; no archive skips openssl absence currently skips server.test.ts with a limitation

**Capabilities:** CAN BUILD WORK PROC CHILD HTTP WIRE DB NATIVE ENV ARCHIVE FAULT CLOCK

**Proposed Can replacement:** Can owns separate fault/lifetime cases, installed-profile identity, stream outcomes and native lifecycle checks; retain distinct limitations and replace all delegated test verdicts.

**Old harness may be deleted when:** Remove this host entry only after the Can replacement demonstrates every listed facet and required variant with independent observations; rewire its callers and retain shared helpers until their last consumer migrates.

**Related source:** [tests/integration/testdata/invoice/driver.ts](/Users/vince/Projects/can-lang/tests/integration/testdata/invoice/driver.ts); [tests/integration/testdata/webhook/driver.ts](/Users/vince/Projects/can-lang/tests/integration/testdata/webhook/driver.ts); [runtime/test/shutdown.test.ts](/Users/vince/Projects/can-lang/runtime/test/shutdown.test.ts); [runtime/test/owner.test.ts](/Users/vince/Projects/can-lang/runtime/test/owner.test.ts); [runtime/test/server.test.ts](/Users/vince/Projects/can-lang/runtime/test/server.test.ts); [runtime/test/transport-owned.test.ts](/Users/vince/Projects/can-lang/runtime/test/transport-owned.test.ts); [runtime/test/transport-late.test.ts](/Users/vince/Projects/can-lang/runtime/test/transport-late.test.ts); [runtime/test/coordination.test.ts](/Users/vince/Projects/can-lang/runtime/test/coordination.test.ts)

**Notes:** Forced live deadline-expiry is not claimed by current live legs; separate runtime controls cover it. Preserve this distinction, not a fabricated stronger pass.

## LIFE-010

**TestWebhookSliceLive** — [tests/integration/webhook_test.go](/Users/vince/Projects/can-lang/tests/integration/webhook_test.go:470)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-LIFE-010`. Evidence: pending.

**Protects**

- Provider HMAC exact-byte auth, duplicate/conflict, malformed/empty/oversize rejection and no extra rows
- Carrier protocol/auth/stale/future/shape/budget/nonce rollback/replay matrix
- Claim exclusivity, leases, failure/reclaim, heartbeat version/holder and terminal replay
- Poison dead-letter after five failures, duplicate ack/dead-letter idempotence and unknown/already-done cases
- Server crash before acknowledgement gives at-least-once downstream delivery with one ledger effect; inflight crash converges
- Two pending items claim and settle without candidate-read failure

**Current observations:** real signed HTTP; controlled downstream receipt counts; independent ledger/outbox/attempt/dead/nonce rows; SIGKILL/restart

**Variants:** provider invalid signatures/bodies carrier protocol/auth/nonce cases claim/heartbeat/ack/dead-letter between-delivery/unacked/inflight crash; two-item batch

**Environment/selection gates:** CAN_BUN_ARCHIVE required; absent archive currently skips staged bundled compiler/runtime, isolated homes; current offline helpers use macOS sandbox-exec free loopback ports and disposable SQLite; managed server and fd-3 credential snapshot

**Capabilities:** CAN BUILD WORK CHILD HTTP PEER DB CLOCK FILES FAULT

**Proposed Can replacement:** Can implements provider/carrier/stub fixture helpers, drives compiled webhook, compares protocol and database facts and preserves at-least-once evidence.

**Old harness may be deleted when:** Remove this host entry only after the Can replacement demonstrates every listed facet and required variant with independent observations; rewire its callers and retain shared helpers until their last consumer migrates.

**Related source:** [tests/integration/testdata/webhook/driver.ts](/Users/vince/Projects/can-lang/tests/integration/testdata/webhook/driver.ts)

**Notes:** 

## LIFE-011

**TestCompanionPairLiveF06** — [tests/integration/webhook_f06_test.go](/Users/vince/Projects/can-lang/tests/integration/webhook_f06_test.go:550)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-LIFE-011`. Evidence: pending.

**Protects**

- Actual two companion processes, report step/failure/occurrence shape and secret redaction
- Wrong secret/invalid CLI flags leave rows untouched; two workers claim disjoint capped batches
- Serial/parallel concurrency bounds; five-attempt poison/dead-letter without sixth send
- Destination policy and credential binding failures do not touch denied peers; timeout/retry outcomes remain truthful
- SIGKILL with four gated sends yields 4 duplicated/4 single downstream receipts but one ledger effect each
- Bounded 200-row drain: all effects complete once, wall time <=120s, peak RSS >0 and <256MiB; throughput logged only
- 200-row leg requires both workers to take nonempty first-round batches, completes within at most 10 drain rounds, and sees each ID exactly once in report steps, stub receipts, ledger effects and delivered attempts
- Idle/backoff gaps 0.7..6s, continued polling, no restart and clean SIGTERM

**Current observations:** two actual product companion stdout/exit reports; gated/delayed HTTP stub receipts/peak concurrency; independent DB rows; process RSS and elapsed timestamps

**Variants:** A auth B 48-row split C concurrency 1/4 D poison E destination F credentials G timeout H crash W4.3 200-row batch I idle/term

**Environment/selection gates:** CAN_BUN_ARCHIVE required; absent archive currently skips staged bundled compiler/runtime, isolated homes; current offline helpers use macOS sandbox-exec free loopback ports and disposable SQLite; managed server and fd-3 credential snapshot resource/measurement execution currently deferred; ledger inclusion does not authorize running it

**Capabilities:** CAN BUILD WORK CHILD HTTP PEER DB CLOCK METRIC FAULT

**Proposed Can replacement:** Can owns A-I correctness scenarios and a separately identified resource-qualification case; launches the actual companion as subject, keeps native peer/process observations generic and checks results in Can.

**Old harness may be deleted when:** Remove this host entry only after the Can replacement demonstrates every listed facet and required variant with independent observations; rewire its callers and retain shared helpers until their last consumer migrates.

**Related source:** [tests/integration/testdata/webhook/driver.ts](/Users/vince/Projects/can-lang/tests/integration/testdata/webhook/driver.ts); [examples/webhook/companion/main.ts](/Users/vince/Projects/can-lang/examples/webhook/companion/main.ts)

**Notes:** Companion main.ts is a product under test, not a host test harness. Keep or explicitly retire/re-scope measurement obligation with evidence; no benchmark run in this task.

## LIFE-012

**TestLinuxInstalledArtifactSmoke** — [tests/integration/linux_distribution_test.go](/Users/vince/Projects/can-lang/tests/integration/linux_distribution_test.go:24)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-LIFE-012`. Evidence: pending.

**Protects**

- Linux build/release/install selection and pinned runtime identity from actual installed executable
- Offline qualification APIs/behaviors, negative controls and glibc evidence
- Installed process/files/crypto/SQLite examples assert and run; SQLite build identity stable

**Current observations:** install selection, runtime-check JSON, native qualification report, exact stdout/stderr, native SQLite seed and repeated build IDs

**Variants:** process/files/crypto match+mismatch/SQLite native qualification when python present in old harness

**Environment/selection gates:** linux/amd64; CAN_BUN_ARCHIVE network denied container; python3 currently makes native leg optional

**Capabilities:** CAN PROC FILES WORK BUILD ENV ARCHIVE DB NATIVE

**Proposed Can replacement:** Can uses product distribution entrypoints and independent installation/native observations, runs installed examples and verifies the complete Linux profile.

**Old harness may be deleted when:** Remove this host entry only after the Can replacement demonstrates every listed facet and required variant with independent observations; rewire its callers and retain shared helpers until their last consumer migrates.

**Related source:** [distribution/qualify.py](/Users/vince/Projects/can-lang/distribution/qualify.py); [distribution/linux/smoke/sqlite-seed.ts](/Users/vince/Projects/can-lang/distribution/linux/smoke/sqlite-seed.ts); [tests/integration/testdata/sql/sqlite-driver.ts](/Users/vince/Projects/can-lang/tests/integration/testdata/sql/sqlite-driver.ts); [tests/integration/testdata/sql/sqlite-seed.sql](/Users/vince/Projects/can-lang/tests/integration/testdata/sql/sqlite-seed.sql)

**Notes:** A missing Python/native qualification leg is partial evidence, not full target qualification. Current pinned literals must track selected target contract, no compatibility promise.

## LIFE-013

**TestLinuxPostgresRoundtrip** — [tests/integration/linux_distribution_test.go](/Users/vince/Projects/can-lang/tests/integration/linux_distribution_test.go:265)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-LIFE-013`. Evidence: pending.

**Protects**

- Installed Linux sidecar reaches PostgreSQL 17 and roundtrips one row
- Native create/insert/select/drop and server identity observed independently

**Current observations:** raw Bun.SQL external driver JSON serverVersion/id/body

**Variants:** PostgreSQL 17 installed-sidecar profile

**Environment/selection gates:** linux/amd64; CAN_LINUX_INSTALL_ROOT; DATABASE_URL; absent configuration skips

**Capabilities:** PROC WORK ENV DB NATIVE

**Proposed Can replacement:** Can owns isolated create/insert/select/drop, compares native version/types/row facts and guarantees owned-table cleanup.

**Old harness may be deleted when:** Remove this host entry only after the Can replacement demonstrates every listed facet and required variant with independent observations; rewire its callers and retain shared helpers until their last consumer migrates.

**Related source:** [distribution/linux/smoke/pg-driver.ts](/Users/vince/Projects/can-lang/distribution/linux/smoke/pg-driver.ts)

**Notes:** 

## LIFE-014

**TestCurrentS3Objects** — [tests/integration/s3_test.go](/Users/vince/Projects/can-lang/tests/integration/s3_test.go:44)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-LIFE-014`. Evidence: pending.

**Protects**

- Offline assertion/build plus live cross-process S3 persistence and fd-3 credential input
- Put/get/stat/exists/range/stream, bounded large-object rejection and invalid UTF-8 payload
- Paginated listing, signed GET/PUT/method rejection, multipart/copy/abort/delete
- Missing/invalid-config/invalid-number/access-denied outcomes and no signed URL leakage
- Happy-path final object listing is empty; deferred cleanup is attempted but failure-path cleanup is not reliably asserted

**Current observations:** compiled CLI stdout/stderr/status; independent S3 native seed/stat and ordinary HTTP on signed URLs; final object list

**Variants:** usage/CRUD/range/stream/size/listing/signature/multipart/copy/cancel/error cases

**Environment/selection gates:** CAN_BUN_ARCHIVE required; absent archive currently skips staged bundled compiler/runtime, isolated homes; current offline helpers use macOS sandbox-exec CAN_TEST_S3_ENDPOINT/REGION/BUCKET/ACCESS_KEY/SECRET_KEY all required

**Capabilities:** CAN PROC BUILD WORK CHILD FILES STORE HTTP FAULT

**Proposed Can replacement:** Can owns all CLI/storage matrices, creates generic independent seed/observations, registers run-prefix cleanup before first upload and reports cleanup failures.

**Old harness may be deleted when:** Remove this host entry only after the Can replacement demonstrates every listed facet and required variant with independent observations; rewire its callers and retain shared helpers until their last consumer migrates.

**Related source:** [tests/integration/testdata/s3/s3-driver.ts](/Users/vince/Projects/can-lang/tests/integration/testdata/s3/s3-driver.ts); [tests/integration/testdata/s3/payload.bin](/Users/vince/Projects/can-lang/tests/integration/testdata/s3/payload.bin)

**Notes:** Old deferred cleanup suppresses all failures and registers after seed. Replacement must satisfy stronger immediate ownership and reported cleanup contract, not copy best-effort silence. Setup writes before registration at s3_test.go:91–108/168–187; setup/report failure may leak the prefix. Final empty listing runs only on the happy path. Failure/interruption recovery and an unresolved-cleanup failure verdict are additional acceptance requirements derived from C9.

## LIFE-015

**TestUP23WriteVerdict** — [tests/integration/up23_verdict_test.go](/Users/vince/Projects/can-lang/tests/integration/up23_verdict_test.go:51)

Kind: `go_test`. Proposed disposition: `migrate`. Replacement obligation: `CAN-LIFE-015`. Evidence: pending.

**Protects**

- 11 guard verdict legs proven on both named browsers with 20-check full report and expected guard occurrences
- Paired browser/server build and final DB revision 5/four replay rows
- Verdict records exact commit and engine versions; missing engine/leg is not success

**Current observations:** live invoice.mjs reports; native SQLite facts; git revision and structured verdict file

**Variants:** Chromium and WebKit swap 200/403/409/422/503; target absence before/during; OOB/partial/control header; remount

**Environment/selection gates:** CAN_UP23_OUT and CAN_BUN_ARCHIVE; node/Playwright and both engines; current missing config skips

**Capabilities:** CAN BUILD WORK PROC CHILD HTTP BROWSER DB FILES SUITE

**Proposed Can replacement:** Can runs guard cases directly on Chromium/WebKit, verifies complete leg sets and DB state, then writes provenance-bound compact evidence for consumers.

**Old harness may be deleted when:** Remove this host entry only after the Can replacement demonstrates every listed facet and required variant with independent observations; rewire its callers and retain shared helpers until their last consumer migrates.

**Related source:** [tests/integration/browser/invoice.mjs](/Users/vince/Projects/can-lang/tests/integration/browser/invoice.mjs); [tests/integration/invoice_test.go](/Users/vince/Projects/can-lang/tests/integration/invoice_test.go); [tests/integration/testdata/invoice/driver.ts](/Users/vince/Projects/can-lang/tests/integration/testdata/invoice/driver.ts); [.github/workflows/verifier.yml](/Users/vince/Projects/can-lang/.github/workflows/verifier.yml)

**Notes:** Can consolidate repeated invoice browser execution by sharing same-run qualified facts; do not accept stale commit or omit engine evidence.

## LIFE-016

**TestHarnessKeyStable** — [tests/integration/harness_cache_test.go](/Users/vince/Projects/can-lang/tests/integration/harness_cache_test.go:16)

Kind: `go_test`. Proposed disposition: `consolidate`. Replacement obligation: `CAN-LIFE-016`. Evidence: pending.

**Protects**

- Identical bundle inputs produce a stable nonempty cache key

**Current observations:** key repeated on same source/archive

**Variants:** None recorded.

**Environment/selection gates:** None recorded.

**Capabilities:** CAN FILES WORK BUILD CLOCK

**Proposed Can replacement:** Can asserts deterministic build-input key calculation

**Old harness may be deleted when:** Delete this Go check when the replacement shared-resource Can suite demonstrates this listed property; retire the old helper only after integration/failure-convention/host consumers have migrated.

**Notes:** Current repeated call can hit memo; replacement should exercise deterministic calculation directly too.

## LIFE-017

**TestHarnessKeySensitive** — [tests/integration/harness_cache_test.go](/Users/vince/Projects/can-lang/tests/integration/harness_cache_test.go:35)

Kind: `go_test`. Proposed disposition: `consolidate`. Replacement obligation: `CAN-LIFE-017`. Evidence: pending.

**Protects**

- Changed archive bytes invalidate build reuse

**Current observations:** two different archive inputs produce different keys

**Variants:** None recorded.

**Environment/selection gates:** None recorded.

**Capabilities:** CAN FILES WORK BUILD CLOCK

**Proposed Can replacement:** Can varies each relevant declared input and checks reuse invalidation

**Old harness may be deleted when:** Delete this Go check when the replacement shared-resource Can suite demonstrates this listed property; retire the old helper only after integration/failure-convention/host consumers have migrated.

**Notes:** Current executable test varies only archive bytes; broader source/toolchain dimensions are derived acceptance work, not existing measured coverage.

## LIFE-018

**TestHarnessVerifyClean** — [tests/integration/harness_cache_test.go](/Users/vince/Projects/can-lang/tests/integration/harness_cache_test.go:86)

Kind: `go_test`. Proposed disposition: `consolidate`. Replacement obligation: `CAN-LIFE-018`. Evidence: pending.

**Protects**

- Pristine manifest-listed files pass content verification

**Current observations:** fake bundle with independently calculated hashes

**Variants:** None recorded.

**Environment/selection gates:** None recorded.

**Capabilities:** CAN FILES WORK BUILD CLOCK

**Proposed Can replacement:** Can creates compact valid bundle fixture and requires successful verification

**Old harness may be deleted when:** Delete this Go check when the replacement shared-resource Can suite demonstrates this listed property; retire the old helper only after integration/failure-convention/host consumers have migrated.

**Notes:** 

## LIFE-019

**TestHarnessVerifyContaminated** — [tests/integration/harness_cache_test.go](/Users/vince/Projects/can-lang/tests/integration/harness_cache_test.go:96)

Kind: `go_test`. Proposed disposition: `consolidate`. Replacement obligation: `CAN-LIFE-019`. Evidence: pending.

**Protects**

- Mutated cached file is detected before reuse

**Current observations:** overwrite launcher after manifest capture; verification error

**Variants:** None recorded.

**Environment/selection gates:** None recorded.

**Capabilities:** CAN FILES WORK BUILD CLOCK

**Proposed Can replacement:** Can tampers with one owned bundle file and checks rejection/rebuild facts

**Old harness may be deleted when:** Delete this Go check when the replacement shared-resource Can suite demonstrates this listed property; retire the old helper only after integration/failure-convention/host consumers have migrated.

**Notes:** 

## LIFE-020

**TestHarnessEntryComplete** — [tests/integration/harness_cache_test.go](/Users/vince/Projects/can-lang/tests/integration/harness_cache_test.go:107)

Kind: `go_test`. Proposed disposition: `consolidate`. Replacement obligation: `CAN-LIFE-020`. Evidence: pending.

**Protects**

- Missing/wrong completion key cannot look complete; correct key can

**Current observations:** missing/foreign/matching marker observations

**Variants:** None recorded.

**Environment/selection gates:** None recorded.

**Capabilities:** CAN FILES WORK BUILD CLOCK

**Proposed Can replacement:** Can supplies incomplete/foreign/valid cache states and checks admission

**Old harness may be deleted when:** Delete this Go check when the replacement shared-resource Can suite demonstrates this listed property; retire the old helper only after integration/failure-convention/host consumers have migrated.

**Notes:** Marker spelling/layout may change; preserve incomplete-publication rejection.

## LIFE-021

**TestHeavySlotsBounded** — [tests/integration/harness_cache_test.go](/Users/vince/Projects/can-lang/tests/integration/harness_cache_test.go:130)

Kind: `go_test`. Proposed disposition: `consolidate`. Replacement obligation: `CAN-LIFE-021`. Evidence: pending.

**Protects**

- Capacity blocks third acquisition and release admits waiter

**Current observations:** cap-2 barrier/timing observations

**Variants:** None recorded.

**Environment/selection gates:** None recorded.

**Capabilities:** CAN WORK CHILD CLOCK

**Proposed Can replacement:** Can launches bounded jobs and checks peak admission plus waiter progress on release

**Old harness may be deleted when:** Delete this Go check when the replacement shared-resource Can suite demonstrates this listed property; retire the old helper only after integration/failure-convention/host consumers have migrated.

**Notes:** No permanent Go semaphore implementation requirement.

## LIFE-022

**TestHeavySlotCap** — [tests/integration/harness_cache_test.go](/Users/vince/Projects/can-lang/tests/integration/harness_cache_test.go:158)

Kind: `go_test`. Proposed disposition: `consolidate`. Replacement obligation: `CAN-LIFE-022`. Evidence: pending.

**Protects**

- Default/valid override/invalid override slot policy

**Current observations:** isolated CAN_TEST_HEAVY_SLOTS parsing

**Variants:** None recorded.

**Environment/selection gates:** None recorded.

**Capabilities:** CAN WORK CHILD CLOCK

**Proposed Can replacement:** Can asserts selected documented resource configuration rules

**Old harness may be deleted when:** Delete this Go check when the replacement shared-resource Can suite demonstrates this listed property; retire the old helper only after integration/failure-convention/host consumers have migrated.

**Notes:** Current default=3, override=7, invalid=0 falls back; specific defaults may be reselected with rationale.

## LIFE-023

**TestHarnessSharedEndToEnd** — [tests/integration/harness_cache_test.go](/Users/vince/Projects/can-lang/tests/integration/harness_cache_test.go:178)

Kind: `go_test`. Proposed disposition: `consolidate`. Replacement obligation: `CAN-LIFE-023`. Evidence: pending.

**Protects**

- Two bundle requests share one suite-owned artifact with <=1 new fill

**Current observations:** path identity and real fill counter

**Variants:** None recorded.

**Environment/selection gates:** CAN_BUN_ARCHIVE required

**Capabilities:** CAN FILES WORK BUILD CLOCK

**Proposed Can replacement:** Can requests same build twice and checks one verified reusable result plus bounded fill evidence

**Old harness may be deleted when:** Delete this Go check when the replacement shared-resource Can suite demonstrates this listed property; retire the old helper only after integration/failure-convention/host consumers have migrated.

**Notes:** Requires pinned archive; retain source-key invalidation and per-hit verification as separate cases.

## LIFE-024

**TestMain** — [tests/integration/harness_test.go](/Users/vince/Projects/can-lang/tests/integration/harness_test.go:149)

Kind: `support`. Proposed disposition: `replace_support`. Replacement obligation: `CAN-LIFE-024`. Evidence: pending.

**Protects**

- One per-run owned build cache; input-key reuse; manifest hash verification on hits; atomic fill and bounded contention
- Suite cleanup runs after failed tests and cleanup error overrides successful exit
- Heavy job admission policy and recovery of abandoned owned resources

**Current observations:** owned tempcache lease; verified manifest/files and fill marker; exit/stderr; active job facts

**Variants:** None recorded.

**Environment/selection gates:** CAN_TEST_CACHE parent; CAN_TEST_HEAVY_SLOTS override

**Capabilities:** SUITE WORK FILES BUILD CHILD PROC CLOCK

**Proposed Can replacement:** Can owns reuse/scheduling policy and its expectations; generic lifecycle substrate supplies leases, isolation, cleanup and raw facts.

**Old harness may be deleted when:** Remove this host entry only after the Can replacement demonstrates every listed facet and required variant with independent observations; rewire its callers and retain shared helpers until their last consumer migrates.

**Related source:** [tests/support/tempcache/cache.go](/Users/vince/Projects/can-lang/tests/support/tempcache/cache.go); [tests/failure-conventions/harness_test.go](/Users/vince/Projects/can-lang/tests/failure-conventions/harness_test.go); [host/conformance/admission_test.go](/Users/vince/Projects/can-lang/host/conformance/admission_test.go)

**Notes:** Age-only stale fill-lock reaping is current mechanism, not the final safety contract; future cleanup requires inactivity/ownership evidence.

## LIFE-025

**setup / inspect** — [tests/integration/testdata/webhook/driver.ts](/Users/vince/Projects/can-lang/tests/integration/testdata/webhook/driver.ts:1)

Kind: `support`. Proposed disposition: `replace_support`. Replacement obligation: `CAN-LIFE-025`. Evidence: pending.

**Protects**

- Five-table webhook schema setup validation
- Ordered ledger/outbox/attempt/dead-letter/nonce observations with large integers preserved

**Current observations:** direct Bun.SQL static templates and native SQLite results normalized to strings

**Variants:** None recorded.

**Environment/selection gates:** None recorded.

**Capabilities:** DB NATIVE FILES WORK

**Proposed Can replacement:** Can chooses schema/actions/queries and checks shape; raw independent DB bindings return values and close resources.

**Old harness may be deleted when:** Delete after every webhook/lifecycle scenario uses Can setup/inspection and independent row sensitivity remains demonstrated; generic native SQL calls may remain without table/test policy.

**Related source:** [tests/integration/webhook_test.go](/Users/vince/Projects/can-lang/tests/integration/webhook_test.go); [tests/integration/webhook_f06_test.go](/Users/vince/Projects/can-lang/tests/integration/webhook_f06_test.go); [tests/integration/gate4_fault_test.go](/Users/vince/Projects/can-lang/tests/integration/gate4_fault_test.go)

**Notes:** 

## LIFE-026

**setup** — [tests/integration/testdata/s3/s3-driver.ts](/Users/vince/Projects/can-lang/tests/integration/testdata/s3/s3-driver.ts:1)

Kind: `support`. Proposed disposition: `replace_support`. Replacement obligation: `CAN-LIFE-026`. Evidence: pending.

**Protects**

- Independent reachable-bucket seed of 8192-byte binary payload
- Report host/bucket/key/size without credentials

**Current observations:** native S3Client list/write/stat and binary file bytes

**Variants:** None recorded.

**Environment/selection gates:** None recorded.

**Capabilities:** STORE NATIVE FILES WORK

**Proposed Can replacement:** Can owns seed/readiness/report policy with independent storage observations and registers prefix ownership before writes.

**Old harness may be deleted when:** Delete when the live S3 Can case seeds/observes native storage independently and reports cleanup failure; keep payload.bin as fixture.

**Related source:** [tests/integration/s3_test.go](/Users/vince/Projects/can-lang/tests/integration/s3_test.go)

**Notes:** 
