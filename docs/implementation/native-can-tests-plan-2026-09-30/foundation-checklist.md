# Foundation, reference and external owner

Status: **planned, not started**. [Main plan](../native-can-tests-plan-2026-09-30.md) defines execution, commits, resource bounds and shared retirement gates.

Each checkbox is a task acceptance, not merely source completion. Draft after `start_after`; complete only after both dependency lists and the task checks pass. Record evidence using the [progress template](progress-template.json). Proposed paths do not imply existing implementation. Cards follow dependency order; independent lanes may run concurrently within the main plan’s limits.

## P00 — Record the implementation baseline and ownership assignments

- [x] **P00 accepted** — owner lane: `integration`. Evidence: [evidence/P00.json](evidence/P00.json).

**Start after:** none. **Additional acceptance prerequisites:** none.

**Files:** `docs/implementation/native-can-tests-plan-2026-09-30/`.

**Concrete change:** Reconcile HEAD, dirty files, source hashes, all ledger rows/delegates/callers and task owners; preserve unrelated edits and reuse an appropriate checkout.

**Acceptance checks:** No silent scope loss; current row inventory matches or has reviewed changes; write ownership and temporary cleanup registered before any executable work.

**Completion evidence:** evidence/P00.json with exact identities, checks, controls, cleanup and independent review.

**Early commit:** `docs(testing): record implementation baseline and lane ownership`.

## P01 — Version shared trust and evidence schemas

- [x] **P01 accepted** — owner lane: `shared-reference`. Evidence: [evidence/P01.json](evidence/P01.json).

**Start after:** P00. **Additional acceptance prerequisites:** none.

**Files:** `schemas/native-test/`.

**Concrete change:** Freeze versioned identity, scope, operation, limit, completeness, report, receipt and correction wire fields in **new** `schemas/native-test/`; document R/N/S/A/C roles and execution versus qualification modes.

**Acceptance checks:** Schema fixtures reject missing identity, unknown version, unbounded limit and false completeness.

**Completion evidence:** Save schema review and fixtures; commit contracts alone.

**Early commit:** `feat(testing): version shared trust and evidence schemas`.

## P02 — Implement typed bounded protocol adapters

- [ ] **P02 accepted** — owner lane: `shared-reference`.

**Start after:** P01. **Additional acceptance prerequisites:** none.

**Files:** `tools/native-test-owner/protocol/codec/`, `tests/native-can/src/protocol/`.

**Concrete change:** Add bounded, typed encode/decode and cross-language compatibility fixtures in **new** `tools/native-test-owner/protocol/` and `tests/native-can/src/protocol/`. This is transport, not test policy.

**Acceptance checks:** Bounded native wire codec and data fixtures reject duplicate identities, gaps, truncation and malformed envelopes without trusting subject stdout. Ordinary Can codec source and supplied fixtures are reviewed provisionally; full cross-language execution is required by P23 after R-core acceptance, not assumed to qualify N.

**Completion evidence:** Save fixture digest and results; commit protocol adapters.

**Early commit:** `feat(testing): implement typed bounded protocol adapters`.

## P03 — Select reference seed inputs

- [x] **P03 accepted** — owner lane: `shared-reference`. Evidence: [evidence/P03.json](evidence/P03.json).

**Start after:** P01. **Additional acceptance prerequisites:** none.

**Files:** `docs/implementation/native-can-tests-plan-2026-09-30/reference-seed.md`.

**Concrete change:** Inventory exact local seed inputs and select a proposed baseline R distribution/build recipe with source, compiler, catalogue, runtime, Bun and binding identities in `docs/implementation/native-can-tests-plan-2026-09-30/reference-seed.md` and compact manifest. Do **not** call candidate `canlc run` to build its judge.

**Acceptance checks:** Acceptance remains pending until independent controls; changed/missing artifact fails identity.

**Completion evidence:** Record provenance and limits; commit seed proposal.

**Early commit:** `feat(testing): select reference seed inputs`.

## P25 — Build transitional bootstrap witness T

- [ ] **P25 accepted** — owner lane: `native-owner`.

**Start after:** P01. **Additional acceptance prerequisites:** none.

**Files:** `tools/native-test-bootstrap/`.

**Concrete change:** Implement and identify the bounded transitional bootstrap parent T in **new** `tools/native-test-bootstrap/`: outside N's killable subtree, pre-register finite child/descriptors/scratch authority, retain stop/reap and independent release witness.

**Acceptance checks:** N/worker death leaves T alive; loss of T cannot issue a green receipt or grant new admission.

**Completion evidence:** Save T identity, bounded graph and cleanup controls; commit bootstrap witness before N qualification.

**Early commit:** `feat(testing): build transitional bootstrap witness t`.

## P04 — Materialize reference seed

- [ ] **P04 accepted** — owner lane: `shared-reference`.

**Start after:** P03, P25. **Additional acceptance prerequisites:** none.

**Files:** `tools/native-test-reference/`.

**Concrete change:** Build or locate the seed compiler distribution through the ordinary reviewed bootstrap tools; stage it under run/acceptance ownership, not a per-case copy. Owner: **new** `tools/native-test-reference/` and acceptance manifest, with existing `tools/distbuild/main.go` only through its normal interface.

**Acceptance checks:** Exact compiler/runtime/Bun contents and input graph are sealed; mutation after selection invalidates.

**Completion evidence:** Save build manifest and disposal receipt; commit seed materialization support.

**Early commit:** `feat(testing): materialize reference seed`.

## P05 — Independently accept reference seed

- [ ] **P05 accepted** — owner lane: `shared-reference`.

**Start after:** P04, P25. **Additional acceptance prerequisites:** none.

**Files:** `tests/native-can/bootstrap/reference/`.

**Concrete change:** Independently qualify seed R for the bounded judge features it must execute, with fixed native observations and existing host checks as transitional evidence. Owner: reference acceptance record and **new** `tests/native-can/bootstrap/reference/` controls.

**Acceptance checks:** Wrong observation, wrong compiler/runtime digest and failing assertion are detected; R acceptance is separate from candidate status and suite-source trust.

**Completion evidence:** Save accepted/blocked manifest and review; commit controls/record.

**Early commit:** `test(testing): independently accept reference seed`.

## P06 — Define separate reference capability refresh

- [ ] **P06 accepted** — owner lane: `shared-reference`.

**Start after:** P05. **Additional acceptance prerequisites:** none.

**Files:** `tools/native-test-reference/`.

**Concrete change:** Define capability refresh R1 as a separate promotion lane in reference manifest tooling: predecessor-compatible comparisons, independent new-feature observations, rollback identity and explicit bridge assumptions.

**Acceptance checks:** A new binding/syntax cannot self-promote via the candidate, and provisional suite edits do not rewrite R acceptance.

**Completion evidence:** Save simulated promotion/rollback control; commit refresh machinery before it is needed.

**Early commit:** `feat(testing): define separate reference capability refresh`.

## P07 — Journal ownership before effects

- [ ] **P07 accepted** — owner lane: `native-owner`.

**Start after:** P01. **Additional acceptance prerequisites:** none.

**Files:** `tools/native-test-owner/journal/`.

**Concrete change:** Implement a durable outside-workspace reservation/ownership journal in **new** `tools/native-test-owner/journal/`: register intent before effect, record owner/start/path identities, leases, operation ID and argument digest, partial acquisitions, monotonic state.

**Acceptance checks:** Crash between reservation and marker remains discoverable; conflicting replay is rejected; wrong PID/path is not authority.

**Completion evidence:** Save fault-point journal fixtures; commit journal.

**Early commit:** `feat(testing): journal ownership before effects`.

## P08 — Implement grants and idempotent dispatch

- [ ] **P08 accepted** — owner lane: `native-owner`.

**Start after:** P07, P02. **Additional acceptance prerequisites:** none.

**Files:** `tools/native-test-owner/protocol/dispatch/`, `tools/native-test-owner/service/`.

**Concrete change:** Implement run/case grants, typed handles, at-most-once operation dispatch and bounded framed request/reply/event channels in `tools/native-test-owner/protocol/` and `.../service/`.

**Acceptance checks:** Same ID/same arguments joins; changed arguments reject; lost acknowledgment reports indeterminate without redispatch. Subject stdout cannot forge a supervisor message.

**Completion evidence:** Save frame/dispatch controls; commit service core.

**Early commit:** `feat(testing): implement grants and idempotent dispatch`.

## P09 — Own bounded workspaces

- [ ] **P09 accepted** — owner lane: `native-owner`.

**Start after:** P07. **Additional acceptance prerequisites:** none.

**Files:** `tools/native-test-owner/workspace/`.

**Concrete change:** Implement owned scratch creation, identity-safe file operations, seal/lease and disposal in `tools/native-test-owner/workspace/`.

**Acceptance checks:** Partial allocation, symlink replacement, foreign/active lease, exceeded byte/entry/depth and failed deletion remain charged/unresolved.

**Completion evidence:** Save bounded filesystem controls; commit workspace owner.

**Early commit:** `feat(testing): own bounded workspaces`.

## P10 — Own processes and descriptors

- [ ] **P10 accepted** — owner lane: `native-owner`.

**Start after:** P08. **Additional acceptance prerequisites:** none.

**Files:** `tools/native-test-owner/process/`, `compiler/internal/driver/test_generation_lease.go`.

**Concrete change:** Implement process tree and descriptor authority in `tools/native-test-owner/process/`: exact executable/environment/fd map, spawn-start identity, separate offered/accepted/EOF/writer-exit/child-exit facts, signal/wait/reap and descendant edges. Implement the actual kernel fd 4 generation read-lease handoff and lifetime in the product driver/process boundary, coordinated with P27 reuse accounting. The child must retain the kernel lease across launcher exit; closing/missing fd 4 must end protection rather than leave a logical flag. The integration writer applies the driver file patch.

**Acceptance checks:** F1 fd 3 non-reader, delayed EOF, malformed frame, orphan/detached-child and PID reuse controls cannot produce clean release; fd 4 lease remains distinct. Bounded kernel-level control observes protection across launcher exit and loss after missing/early fd close, with independent release witness; Can scenario acceptance is QF2.

**Completion evidence:** Save OS-specific control record; commit process owner.

**Early commit:** `feat(testing): own processes and descriptors`.

## P11 — Drain, recover and receipt owned resources

- [ ] **P11 accepted** — owner lane: `native-owner`.

**Start after:** P07, P08, P09, P10. **Additional acceptance prerequisites:** none.

**Files:** `tools/native-test-owner/recovery/`, `tools/native-test-owner/receipt/cleanup/`.

**Concrete change:** Implement stop-admission, bounded drain, owner-only cleanup, recovery scan and compact completion receipt in `tools/native-test-owner/recovery/` and `.../receipt/`.

**Acceptance checks:** Worker/controller/N death with intact T, old unknown schema, replaced directory, active foreign process and incomplete liveness proof never authorize a signal/deletion or old-run pass. A later safe cleanup preserves the old incomplete outcome.

**Completion evidence:** Save crash/recovery receipts; commit lifecycle owner.

**Early commit:** `feat(testing): drain, recover and receipt owned resources`.

## P12 — Reserve host capacity and deadlines

- [ ] **P12 accepted** — owner lane: `native-owner`.

**Start after:** P07, P08, P09, P10, P11. **Additional acceptance prerequisites:** none.

**Files:** `tools/native-test-owner/admission/`.

**Concrete change:** Implement one host-wide live-run admission gate and atomic declared-peak reservation (one live case, one offline verification worker, one build producer) in `tools/native-test-owner/admission/`; charge cleanup reserve and enforce absolute monotonic deadlines.

**Acceptance checks:** Two repository roots cannot bypass the gate; nested producer demand does not deadlock; deadline-at-success is failure; no case starts if body plus cleanup cannot fit.

**Completion evidence:** Save bounded race controls; commit admission.

**Early commit:** `feat(testing): reserve host capacity and deadlines`.

## P13 — Implement and qualify strict host enforcement

- [ ] **P13 accepted** — owner lane: `native-owner`.

**Start after:** P12, P25. **Additional acceptance prerequisites:** none.

**Files:** `tools/native-test-owner/host/`.

**Concrete change:** Implement the selected host enforcement adapter, then qualify the host's strict process/memory/temporary-data mechanism, charged scope and finite overshoot bound, including detached children and writes outside workspace helpers. Owner: `tools/native-test-owner/host/` plus host-profile acceptance record.

**Acceptance checks:** Quick envelope (64 processes, 4 GiB, 512 MiB) and bounded-job envelope (64, 6 GiB, 2 GiB) are admitted only where enforcing mechanisms and disposal reserve are demonstrated. Poll-and-kill or sample peaks cannot qualify them; unavailable host remains blocked.

**Completion evidence:** Save host-specific negative controls; commit profile acceptance separately.

**Early commit:** `test(testing): qualify strict host enforcement`.

## P14 — Independently accept N

- [ ] **P14 accepted** — owner lane: `native-owner`.

**Start after:** P10, P11, P12, P13, P02, P25. **Additional acceptance prerequisites:** none.

**Files:** `tools/native-test-owner/acceptance/`, `docs/implementation/native-can-tests-plan-2026-09-30/n-acceptance.json`.

**Concrete change:** Qualify N as a separately built/versioned executable using T's independently witnessed finite process/resource graph; bind protocol, journal, enforcement, recovery and receipt behavior in the N acceptance manifest.

**Acceptance checks:** N death, forged/absent receipt, lingering descendant/descriptor and cleanup failure cannot appear as complete success. T's own loss leaves no green receipt. Scope acceptance to the exact qualified host profile. Another host requires explicit selection and its own receipt; do not silently move execution or treat unsupported local enforcement as a pass.

**Completion evidence:** Save N/T identities and independent release facts; commit N acceptance.

**Early commit:** `test(testing): independently accept n`.

## P15 — Create ordinary Can suite skeleton

- [ ] **P15 accepted** — owner lane: `can-suite-cli`.

**Start after:** P01. **Additional acceptance prerequisites:** none.

**Files:** `tests/native-can/can.project.json`, `tests/native-can/can.errors.json`, `tests/native-can/src/spec/`, `tests/native-can/src/main.can`.

**Concrete change:** Author the canonical ordinary-Can project skeleton, typed descriptor/check/variant/fixture records and attached boundary `asserts` in **new** `tests/native-can/`; keep case logic, selection, expected vectors and reduction in Can functions.

**Acceptance checks:** Complete canonical package with arbitrary ordinary helper functions, context construction, typed check references and attached assertions; static/type/fixture controls are provisional until P26/P23. No missing-assertion exemption or canned whole-case success. Seed duplicate IDs, wrong effect, missing/exhausted fixture and altered observation; diagnostics identify source and exact reproduction selector.

**Completion evidence:** Save source-check receipt; commit skeleton.

**Early commit:** `feat(testing): create ordinary can suite skeleton`.

## P16 — Plan cases and validate coverage

- [ ] **P16 accepted** — owner lane: `can-suite-cli`.

**Start after:** P15. **Additional acceptance prerequisites:** none.

**Files:** `tests/native-can/src/registry/`, `tests/native-can/src/coverage/`.

**Concrete change:** Add pure Can registration, profile/matrix selection, dependency planning, required-check inventories and independent ledger-derived coverage manifest in `tests/native-can/src/registry/` and `.../coverage/`.

**Acceptance checks:** Empty/duplicate/unknown selections fail; deletion of both local check and plan still conflicts with retained obligation; required-but-unavailable variant is blocked.

**Completion evidence:** Save all-root assertion controls; commit planner.

**Early commit:** `feat(testing): plan cases and validate coverage`.

## P17 — Reduce Can evidence and outcomes

- [ ] **P17 accepted** — owner lane: `can-suite-cli`.

**Start after:** P15, P16. **Additional acceptance prerequisites:** none.

**Files:** `tests/native-can/src/report/reducer/`.

**Concrete change:** Add pure Can evidence validation/reduction in `tests/native-can/src/report/`: sequence and identity checks, sticky mismatches, `not_reached`, behavior/execution/cleanup dimensions, partial/full scope.

**Acceptance checks:** Early `ok`, missing/duplicate check, malformed report, forged terminal, caught mismatch and cleanup failure cannot pass; true behavior failure remains visible beside cleanup error.

**Completion evidence:** Save vector fixtures and assertion outcomes; commit reducer.

**Early commit:** `feat(testing): reduce can evidence and outcomes`.

## P27 — Implement run-scoped immutable build reuse and lease accounting

- [ ] **P27 accepted** — owner lane: `can-suite-cli`.

**Start after:** P09, P12, P15. **Additional acceptance prerequisites:** none.

**Files:** `tests/native-can/src/builds/`, `compiler/internal/driver/test_reuse.go`.

**Concrete change:** Can owns complete source/compiler/runtime/options/profile cache keys and reuse policy; native mechanics own atomic publication, leases, producer reservations and release facts. One producer per key; within-run scope only.

**Acceptance checks:** Detect input drift, partial fills, corrupt bundles, waiter cancellation, key omissions, leased eviction and reused determinism/corruption subjects. Failed fills do not publish; no persistent full bundle cache or hidden producer overlap; actual reuse qualification is included in P23.

**Completion evidence:** evidence/P27.json with exact identities, checks, controls, cleanup and independent review.

**Early commit:** `feat(testing): reuse immutable builds within one run`.

## P18 — Stage nonpublishing suite generation

- [ ] **P18 accepted** — owner lane: `can-suite-cli`.

**Start after:** P05, P15, P27. **Additional acceptance prerequisites:** none.

**Files:** `compiler/internal/driver/test_stage.go`, `compiler/internal/emit/program_entry.go`.

**Concrete change:** Add a nonpublishing live-suite staging path in **new** `compiler/internal/driver/test_stage.go` and a dedicated emitted entry in `compiler/internal/emit/program_entry.go`; check all source and execute all assertion roots for bootstrap, then lease one immutable generation for controller/workers.

**Acceptance checks:** `build` publication state stays untouched; `canlc run` does not compile the judge; changed source/runtime/entry invalidates lease; listing does no subject execution. Code and bounded bootstrap/local controls only at this task; authoritative live execution is gated by P14, P26 and P23.

**Completion evidence:** Save staged manifest/root set and controlled invalidation; commit stage/entry.

**Early commit:** `feat(testing): stage nonpublishing suite generation`.

## P19 — Launch Can controller and fresh workers

- [ ] **P19 accepted** — owner lane: `can-suite-cli`.

**Start after:** P18, P12, P02, P16, P17. **Additional acceptance prerequisites:** none.

**Files:** `tests/native-can/src/controller/`, `tests/native-can/src/worker/`, `compiler/internal/driver/test_launch.go`.

**Concrete change:** Implement one Can controller and fresh Bun worker per `(case, variant, attempt)` in `tests/native-can/src/controller/` and `.../worker/`, with **new** `compiler/internal/driver/test_launch.go` as thin R/N handoff.

**Acceptance checks:** Candidate code never imports into judge realm; worker crash, timeout or controller death lacks valid aggregate; source/output channels are separate and bounded. Code and bounded bootstrap/local controls only at this task; authoritative live execution is gated by P14, P26 and P23.

**Completion evidence:** Save local/provisional run-worker identity controls; commit the implementation. Authoritative live-path evidence is P23.

**Early commit:** `feat(testing): launch can controller and fresh workers`.

## P20 — Expose structured product diagnostics

- [ ] **P20 accepted** — owner lane: `can-suite-cli`.

**Start after:** P01. **Additional acceptance prerequisites:** none.

**Files:** `compiler/internal/driver/check_json.go`, `compiler/internal/driver/diagnostics.go`, `compiler/internal/project/`.

**Concrete change:** Add product `canlc check --json` in **new** `compiler/internal/driver/check_json.go`, using `compiler/internal/driver/diagnostics.go` and project loader facts; assign phase, completed/rejected/failed status, actual-read and declared closure identities, completeness and bounded output. Instrument the producer paths for actual input reads/lookups and typed loader causes; do not infer them by parsing rendered text.

**Acceptance checks:** Paired valid/invalid controls distinguish source rejection from missing root/dependency/I/O; Unicode, spanless, related locations, warnings and report overflow behave as specified.

**Completion evidence:** Save CLI vectors; commit diagnostics driver before shared CLI dispatch.

**Early commit:** `feat(testing): expose structured product diagnostics`.

## P21 — Integrate test and check CLI dispatch

- [ ] **P21 accepted** — owner lane: `can-suite-cli`.

**Start after:** P18, P19, P20. **Additional acceptance prerequisites:** none.

**Files:** `compiler/main.go`.

**Concrete change:** Add the narrow `test` and `check --json` dispatch/argument integration in the single-writer `compiler/main.go`; keep test plans/retries/verdicts in Can and N as owner only.

**Acceptance checks:** Candidate and reference selection are explicit; malformed args/unknown schema/absent R or N produce nonzero; `--list` is nonexecuting and `check` never emits/publishes/runs.

**Completion evidence:** Save CLI control outputs; commit dispatch integration.

**Early commit:** `feat(testing): integrate test and check cli dispatch`.

## P22 — Resolve reports, receipts and corrections

- [ ] **P22 accepted** — owner lane: `can-suite-cli`.

**Start after:** P11, P17, P19. **Additional acceptance prerequisites:** none.

**Files:** `tests/native-can/src/report/authority/`, `tools/native-test-owner/receipt/corrections/`.

**Concrete change:** Join the Can report to N's matching final receipt and append-only correction registry in `tests/native-can/src/report/` plus `tools/native-test-owner/receipt/`; expose conservative current-chain resolution to consumers.

**Acceptance checks:** Report-only pass, stale/missing/conflicting receipt, late credible host effect, unresolved cleanup or correction-registry failure prevents qualification; a fresh pass never erases old failure. Code and bounded bootstrap/local controls only at this task; authoritative live execution is gated by P14, P26 and P23.

**Completion evidence:** Save before/after digest chain; commit report authority.

**Early commit:** `feat(testing): resolve reports, receipts and corrections`.

## P28 — Integrate base typed owner bindings and generated catalogue

- [ ] **P28 accepted** — owner lane: `integration`.

**Start after:** P02, P10, P12, P15. **Additional acceptance prerequisites:** none.

**Files:** `compiler/internal/catalogue/catalogue.json`, `compiler/internal/catalogue/cmd/cataloguegen/`, `runtime/test-support/transport.ts`, `runtime/test-support/owner.ts`, `compiler/internal/emit/test_operations.go`.

**Concrete change:** Add base typed owner/workspace/process/evidence bindings and lower them to native operations with the minimum contract adapters. Merge queued descriptors through one writer and regenerate all mirrors via catalogue tooling.

**Acceptance checks:** Catalogue/checker/emitter agree on normal effects and opaque handle admission. Reject forged/stale owner, arbitrary dispatch/eval, raw thenable transport and wrong target; generated files are reproducible. Treat this as provisional R-core input until P26.

**Completion evidence:** evidence/P28.json with exact identities, checks, controls, cleanup and independent review.

**Early commit:** `feat(testing): integrate typed owner bindings`.

## P26 — Accept the core reference generation and its capability delta

- [ ] **P26 accepted** — owner lane: `shared-reference`.

**Start after:** P05, P06, P18, P19, P20, P21, P22, P27, P28, P25. **Additional acceptance prerequisites:** none.

**Files:** `tools/native-test-reference/`, `docs/implementation/native-can-tests-plan-2026-09-30/reference-core.json`.

**Concrete change:** Build the provisional changed compiler/runtime once in a bounded owned scope; independently accept live-entry/base-binding and protocol/fixture changes as R-core before using it to judge authoritative Can tests. Bind every runtime/catalogue/observer input and accepted host scope.

**Acceptance checks:** Predecessor-compatible checks plus independently reviewed fixed/native controls catch wrong transport, wrong target/emission, changed runtime and bypassed assertion behavior. R-seed stays authoritative where expressible; record explicit bootstrap bridge assumptions for new capabilities. No candidate self-promotion.

**Completion evidence:** evidence/P26.json with exact identities, checks, controls, cleanup and independent review.

**Early commit:** `test(testing): accept the core reference generation`.

## P23 — Qualify first complete runner and suite

- [ ] **P23 accepted** — owner lane: `can-suite-cli`.

**Start after:** P14, P26, P16, P17, P18, P19, P20, P21, P22, P27. **Additional acceptance prerequisites:** none.

**Files:** `tests/native-can/bootstrap/runner/`.

**Concrete change:** Independently qualify the first complete runner and reviewed suite/expected-vector snapshot. Owner: `tests/native-can/bootstrap/runner/` and separate suite acceptance manifest.

**Acceptance checks:** Independently reviewed fixed wrong-observation, early-ok, missing check, malformed report, controller/worker death, timeout, cleanup failure and identity drift all go non-green. Live positive scope is minimal owned process/files/report execution, Go/Can wire compatibility and within-run reuse. Compiler-rejection, server/browser-boundary/native examples may use supplied offline assertion vectors here; they do not accept live browser/DB/native-adapter behavior. Separately review the suite and expected-vector snapshot. Bind the receipt to the qualified host profile; unavailable strict enforcement leaves live qualification blocked while independent code work continues.

**Completion evidence:** Record R trust, N trust and S/A trust separately; commit first-runner acceptance.

**Early commit:** `test(testing): qualify first complete runner and suite`.

## P24 — Qualify focused root planner

- [ ] **P24 accepted** — owner lane: `can-suite-cli`.

**Start after:** P18, P19, P20, P21, P22, P23. **Additional acceptance prerequisites:** none.

**Files:** `tests/native-can/src/verification/`, `compiler/internal/driver/test_stage.go`.

**Concrete change:** Add the conservative focused assertion-root closure planner and its own controls in `tests/native-can/src/verification/` plus narrow `compiler/internal/driver/test_stage.go` integration. All roots remain the default until this qualifies.

**Acceptance checks:** Every source remains checked; uncertain callable/generic/scenario dependency falls back to all roots; edited controller cannot exempt its own roots; focused receipt is partial and cannot publish production.

**Completion evidence:** Save closure-vs-all-root controls; commit optional focused mode.

**Early commit:** `test(testing): qualify focused root planner`.

## P29 — Own external service resources and recovery grants

- [ ] **P29 accepted** — owner lane: `native-owner`.

**Start after:** P08, P11, P12. **Additional acceptance prerequisites:** none.

**Files:** `tools/native-test-owner/external/`.

**Concrete change:** Implement journaled service admission, finite listeners/destinations, remote connection/context ownership, database namespace and credential grants, object-store prefixes, cleanup dependencies and quiescence/fence acknowledgments. Register before effect, retain unsettled resource charges and reject caller-invented authority. Engine-specific release implementations belong to K08/K16/K22/K24/K26/K27; generic grants alone do not prove cleanup.

**Acceptance checks:** Local controls reject foreign namespace/context/server/prefix, forged release, duplicate uncertain dispatch and crash-before-journal. Recovery touches only evidenced owned resources; missing release stays unresolved. Accepted N-core remains unchanged until each I slice independently accepts this delta.

**Completion evidence:** evidence/P29.json with local positive/negative ownership controls and explicit unqualified service legs.

**Early commit:** `feat(testing): journal external service ownership`.

