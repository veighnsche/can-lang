# Native mechanisms and integrated qualification

Status: **audited 2026-10-01; see current task states below**. The [lane plan](lane-plan.md) defines dispatch and path reservations; the [master ledger](tasks.json) counts each of 143 task IDs exactly once. Historical source work exists; this planning audit runs no implementation or qualification.

Each checkbox is a task acceptance, not merely source completion. Draft after `start_after`; complete only after both dependency lists and the task checks pass. Record evidence using the [progress template](progress-template.json). Owned paths may contain historical source; completion still requires current acceptance evidence. Cards follow dependency order; independent lanes may run concurrently within the main plan’s limits.

## K01 — Freeze native-value typed operation schema

- [x] **K01: complete** — owner lane: `native-values`. Evidence: [evidence/K01.json](evidence/K01.json).

**Start after:** P01. **Additional acceptance prerequisites:** none.

**Files:** `tools/runtime/test-services/native-values/schema*`.

**Concrete change:** Finite operations, inert handles, sync/async assimilation and ownership contracts; hand catalogue descriptors to integration writer.

**Acceptance checks:** No arbitrary module/export/eval, raw hostile async reply, expected outcome or unlimited handle. Use local or bounded bootstrap controls only; live host-dependent controls wait for the corresponding qualified profile and Q task.

**Completion evidence:** Reviewed effect table and focused schema checks. Record under evidence/K01.json.

**Early commit:** `feat(test): define native-value service operations`.

## K02 — Build inert same-realm native sessions

- [x] **K02: complete** — owner lane: `native-values`. Evidence: [evidence/K02.json](evidence/K02.json).

**Start after:** K01, P02, P08. **Additional acceptance prerequisites:** none.

**Files:** `tools/runtime/test-services/native-values/session*`.

**Concrete change:** Raw handles, aliases, getter-backed then, throwing thenable, revoked Proxy and inert transport.

**Acceptance checks:** No implicit getter/then read; wrong-owner and stale handles reject; service close reclaims child. Use local or bounded bootstrap controls only; live host-dependent controls wait for the corresponding qualified profile and Q task.

**Completion evidence:** Bounded mechanics controls, counters and cleanup receipt. Record under evidence/K02.json.

**Early commit:** `feat(test): preserve hostile native values behind handles`.

## K03 — Add exact native observations and seals

- [x] **K03: complete** — owner lane: `native-values`. Evidence: [evidence/K03.json](evidence/K03.json).

**Start after:** K02. **Additional acceptance prerequisites:** none.

**Files:** `tools/runtime/test-services/native-values/observe*`.

**Concrete change:** IEEE bits, integer lexeme, aliases, explicit effectful reads, counter intervals and final seal.

**Acceptance checks:** Normalization, rounding, alias swap and assimilation are distinguishable. Use local or bounded bootstrap controls only; live host-dependent controls wait for the corresponding qualified profile and Q task.

**Completion evidence:** Focused value-matrix controls; QN1 holds integrated credit. Record under evidence/K03.json.

**Early commit:** `feat(test): expose exact native value facts`.

## K04 — Review generated C ingress seam

- [x] **K04: complete** — owner lane: `native-values`. Evidence: [evidence/K04.json](evidence/K04.json).

**Start after:** K01, P03, P04. **Additional acceptance prerequisites:** none.

**Files:** `tools/runtime/test-services/native-values/c-ingress*`.

**Concrete change:** Fix current-version C generated entry/export/signature/import manifest and same-realm completion handling; supply shared edits to integration writer.

**Acceptance checks:** No private runtime swap or arbitrary dynamic importer; seam reviewed before coding subject bridge. Use local or bounded bootstrap controls only; live host-dependent controls wait for the corresponding qualified profile and Q task.

**Completion evidence:** Signed boundary review with C/R artifacts and hook provenance. Record under evidence/K04.json.

**Early commit:** `docs(test): specify generated C native ingress`.

## K05 — Implement independent C adapter witness

- [ ] **K05: active** — owner lane: `native-values`. Evidence: [evidence/K05.json](evidence/K05.json).

**Audit correction:** The witness must observe actual adapter invocation through the generated C path. Current code only mints in-memory artifact tokens and accepts public begin/complete calls, with no generated artifact digest/hook or invoked adapter. Its positive controls directly call that model. QN2 defers live qualification, not this required witness implementation.

**Start after:** K02, K03, K04. **Additional acceptance prerequisites:** none.

**Files:** `tools/runtime/test-services/native-values/c-witness*`.

**Concrete change:** Witness actual adapter invocation through generated C path; return inert counter/identity facts.

**Acceptance checks:** Plausible bypass, wrong export/signature, eager read and forged local completion each detectable. Use local or bounded bootstrap controls only; live host-dependent controls wait for the corresponding qualified profile and Q task.

**Completion evidence:** Bounded witness controls with instrumented and uninstrumented artifact IDs; QN2 holds credit. Record under evidence/K05.json.

**Early commit:** `feat(test): witness generated C adapter invocation`.

## K06 — Expose late native occurrence and lease facts

- [ ] **K06: blocked** — owner lane: `native-values`. Evidence: [evidence/K06.json](evidence/K06.json).

**Audit correction:** K05 acceptance is invalidated, and the required bounded raw and C-subject controls are absent from the in-memory late-event model. Preserve local occurrence/lease controls and add actual subject evidence before completion.

**Start after:** K03, K05, P09, P11. **Additional acceptance prerequisites:** none.

**Files:** `tools/runtime/test-services/native-values/late*`.

**Concrete change:** Two gated participants, selected identity, late fault/use and terminal event/lease observations.

**Acceptance checks:** Occurrence swap, dropped late event and early release detectable; worker death is incomplete. Use local or bounded bootstrap controls only; live host-dependent controls wait for the corresponding qualified profile and Q task.

**Completion evidence:** Bounded raw and C-subject controls; QN3 holds credit. Record under evidence/K06.json.

**Early commit:** `feat(test): observe late native events and leases`.

## K07 — Implement independent browser host-effect observation

- [ ] **K07: active** — owner lane: `browser`. Evidence: [evidence/K07.json](evidence/K07.json). Independent-observer slice queued (sequential dispatch after K05/P27 health).

**Audit correction:** The required actual independent host-effect observer and durable publication are absent. observer.go uses scripted in-memory facts with no host observation channel; local mirror controls do not finish this mechanism.

**Start after:** P01, P12. **Additional acceptance prerequisites:** none.

**Files:** `tools/runtime/test-services/browser-driver/admission*`, `tools/native-test-owner/host/browser_observer/`.

**Concrete change:** Independent host-UI observer scope, launch attestation, interval through disposal, unknown/gap and correction keys. Implement the observer and admission adapter, including interruption, unknown scope, durable correction publication and the interval through disposal; specification alone does not complete this task. Implement the actual observer outside the launch/driver service and its killable subtree, under independent N/T ownership. Keep the TS admission file as a typed adapter. Observation survives driver death and covers late host effects through confirmed disposal; observer loss makes the interval unknown.

**Acceptance checks:** No headless/HOME/flags/process sample as no-UI proof; missing observer blocks launch. Use local or bounded bootstrap controls only; live host-dependent controls wait for the corresponding qualified profile and Q task.

**Completion evidence:** Reviewed observer coverage and host-profile requirements. Record under evidence/K07.json.

**Early commit:** `feat(test): define browser host-effect admission`.

## K08 — Own pinned-driver browser launches

- [ ] **K08: blocked** — owner lane: `browser`. Evidence: [evidence/K08.json](evidence/K08.json).

**Audit correction:** The outside-service discovery/containment/reclaim implementation remains a model: ObserveChildren accepts supplied identities, Discover reads a map, Contain flags state and Reclaim mints a witness without native kill/reap. Simulated controls are retained as a local slice.

**Start after:** K07, P29. **Additional acceptance prerequisites:** none.

**Files:** `tools/runtime/test-services/browser-driver/launch*`, `tools/native-test-owner/external/browser_local.go`.

**Concrete change:** N-owned service uses pinned driver's launcher and captures effective profile/child identities before context exposure. Implement the outside-service N child-discovery/containment/reclaim adapter so driver death mid-launch cannot orphan children; a receipt from the dead driver is not its own cleanup witness.

**Acceptance checks:** Source/local simulated controls establish launch identity and external ownership; no live browser at this code task. Actual launch, host effects and service-death controls are QB0. A missing qualified observer blocks that acceptance.

**Completion evidence:** Bounded launch and service-death controls plus host observer receipt. Record under evidence/K08.json.

**Early commit:** `feat(test): own browser driver launches`.

## K09 — Add browser event capture and terminal seal

- [ ] **K09: blocked** — owner lane: `browser`. Evidence: [evidence/K09.json](evidence/K09.json).

**Audit correction:** Historical local controls remain recorded, but accepted prerequisites were invalidated by this audit. Revalidate their current receipts before restoring aggregate completion.

**Start after:** K08. **Additional acceptance prerequisites:** none.

**Files:** `tools/runtime/test-services/browser-driver/events*`.

**Concrete change:** Context/page/navigation watches, complete body/error callbacks, cursors, checkpoint and close/seal.

**Acceptance checks:** Pending capture, gap or quiet prefix cannot prove absence; repeated header/decoding scope explicit. Use local or bounded bootstrap controls only; live host-dependent controls wait for the corresponding qualified profile and Q task.

**Completion evidence:** Focused event loss/close tests and terminal receipt. Record under evidence/K09.json.

**Early commit:** `feat(test): seal browser event observations`.

## K10 — Separate pending input from settlement

- [ ] **K10: blocked** — owner lane: `browser`. Evidence: [evidence/K10.json](evidence/K10.json).

**Audit correction:** Historical local controls remain recorded, but accepted prerequisites were invalidated by this audit. Revalidate their current receipts before restoring aggregate completion.

**Start after:** K09. **Additional acceptance prerequisites:** none.

**Files:** `tools/runtime/test-services/browser-driver/input*`.

**Concrete change:** input_begin/input_settle, bounded DOM capture, action identity and lock-free route service while click pending.

**Acceptance checks:** Click settlement remains distinct from route delivery and application witness; route abort may fulfill input. Use local or bounded bootstrap controls only; live host-dependent controls wait for the corresponding qualified profile and Q task.

**Completion evidence:** Bounded held-route mechanics and abort/close controls. Record under evidence/K10.json.

**Early commit:** `feat(test): expose pending browser input facts`.

## K20 — Implement controlled peer and HTTP operations

- [ ] **K20: blocked** — owner lane: `peer`. Evidence: [evidence/K20.json](evidence/K20.json).

**Audit correction:** Historical local controls remain recorded, but accepted prerequisites were invalidated by this audit. Revalidate their current receipts before restoring aggregate completion.

**Start after:** P29. **Additional acceptance prerequisites:** none.

**Files:** `tools/runtime/test-services/http-peer/peer*`, `tools/runtime/test-services/http-peer/http*`.

**Concrete change:** Typed listener/read/write/half-close and pending HTTP upload/header/body operations with finite destination/byte bounds.

**Acceptance checks:** No implicit retry/redirect; accepted bytes differ from consumption; repeated headers, EOF and partial state explicit. Use local or bounded bootstrap controls only; live host-dependent controls wait for the corresponding qualified profile and Q task.

**Completion evidence:** Bounded malformed, truncated, redirect, backpressure and cleanup controls. Record under evidence/K20.json.

**Early commit:** `feat(test): add controlled HTTP peer observations`.

## K11 — Implement one-contact route tokens

- [ ] **K11: blocked** — owner lane: `browser`. Evidence: [evidence/K11.json](evidence/K11.json).

**Audit correction:** Historical local controls remain recorded, but accepted prerequisites were invalidated by this audit. Revalidate their current receipts before restoring aggregate completion.

**Start after:** K09, K10, K20. **Additional acceptance prerequisites:** none.

**Files:** `tools/runtime/test-services/browser-driver/route*`.

**Concrete change:** Route token state, fetch once, immutable captured body, resolve modes and separate contact/delivery IDs.

**Acceptance checks:** Duplicate upstream contact, partial body and unknown delivery cannot pass; repeated native operation joins. Use local or bounded bootstrap controls only; live host-dependent controls wait for the corresponding qualified profile and Q task.

**Completion evidence:** Route controls with operation journal and disposal receipt. Record under evidence/K11.json.

**Early commit:** `feat(test): preserve browser route contact and delivery`.

## K13 — Add immediate DOM and event provenance

- [ ] **K13: blocked** — owner lane: `browser`. Evidence: [evidence/K13.json](evidence/K13.json).

**Audit correction:** Historical local controls remain recorded, but accepted prerequisites were invalidated by this audit. Revalidate their current receipts before restoring aggregate completion.

**Start after:** K09, K10. **Additional acceptance prerequisites:** none.

**Files:** `tools/runtime/test-services/browser-driver/dom*`.

**Concrete change:** Finite same-task batch, clone/append, same-node listener acknowledgment and microtask-order observer.

**Acceptance checks:** Yield, wrong cancellation, returned original handle and missing occurrence detectable. Use local or bounded bootstrap controls only; live host-dependent controls wait for the corresponding qualified profile and Q task.

**Completion evidence:** Focused DOM event controls; QB2 holds credit. Record under evidence/K13.json.

**Early commit:** `feat(test): observe immediate browser DOM actions`.

## K14 — Invoke page-origin Fetch

- [ ] **K14: blocked** — owner lane: `browser`. Evidence: [evidence/K14.json](evidence/K14.json).

**Audit correction:** Historical local controls remain recorded, but accepted prerequisites were invalidated by this audit. Revalidate their current receipts before restoring aggregate completion.

**Start after:** K09, K20. **Additional acceptance prerequisites:** none.

**Files:** `tools/runtime/test-services/browser-driver/fetch*`.

**Concrete change:** Typed bounded page-realm Fetch with credentials and redirect policy.

**Acceptance checks:** Cookie/origin/CSP facts distinguish page Fetch from external HTTP; no source eval. Use local or bounded bootstrap controls only; live host-dependent controls wait for the corresponding qualified profile and Q task.

**Completion evidence:** Focused page-policy controls; QB3 holds credit. Record under evidence/K14.json.

**Early commit:** `feat(test): expose page-origin fetch facts`.

## K15 — Load Can browser artifacts and export facts

- [ ] **K15: blocked** — owner lane: `browser`. Evidence: [evidence/K15.json](evidence/K15.json).

**Audit correction:** Historical local controls remain recorded, but accepted prerequisites were invalidated by this audit. Revalidate their current receipts before restoring aggregate completion.

**Start after:** K03, K09. **Additional acceptance prerequisites:** none.

**Files:** `tools/runtime/test-services/browser-driver/artifact*`.

**Concrete change:** C artifact lease, module load, DOM-ready and bounded typed data export.

**Acceptance checks:** Missing ready/export and unsupported exact representation cannot become empty pass. Use local or bounded bootstrap controls only; live host-dependent controls wait for the corresponding qualified profile and Q task.

**Completion evidence:** Artifact identity and focused export controls; QB4 holds credit. Record under evidence/K15.json.

**Early commit:** `feat(test): load compiled Can browser fixtures`.

## K16 — Clean remote browser contexts after worker death

- [ ] **K16: blocked** — owner lane: `browser`. Evidence: [evidence/K16.json](evidence/K16.json).

**Audit correction:** RequestClose/ConfirmClose exchange in-memory tokens without an independent remote close channel or observed confirmation. Required outside-service authority remains to be implemented.

**Start after:** K08, K09, P29. **Additional acceptance prerequisites:** none.

**Files:** `tools/runtime/test-services/browser-driver/remote*`, `tools/native-test-owner/external/browser_remote.go`.

**Concrete change:** Own remote connection/context separately from shared Firefox launch server. Implement independent N remote-context cleanup authority and confirmation, preserving the shared launch server; missing independent close capability blocks qualification.

**Acceptance checks:** Case close-server attempt rejects before effect; lost close confirmation remains unresolved. Use local or bounded bootstrap controls only; live host-dependent controls wait for the corresponding qualified profile and Q task.

**Completion evidence:** Disposable controls and pinned instance receipt; QB5 holds credit. Record under evidence/K16.json.

**Early commit:** `feat(test): reclaim owned remote browser contexts`.

## K17 — Specify descriptor and environment contract

- [x] **K17: complete** — owner lane: `descriptor`. Evidence: [evidence/K17.json](evidence/K17.json).

**Start after:** P01. **Additional acceptance prerequisites:** none.

**Files:** `schemas/native-test/descriptors/`.

**Concrete change:** fd 0/1/2/3/4 directions, byte/EOF/lifetime states and distinct C CLI/direct-entry environments; N owner owns ExtraFiles.

**Acceptance checks:** No private owner fd or secret argv/log; each launch has fresh snapshot state. Use local or bounded bootstrap controls only; live host-dependent controls wait for the corresponding qualified profile and Q task.

**Completion evidence:** Contract review and descriptor-map checks. Record under evidence/K17.json.

**Early commit:** `feat(test): define managed descriptor fixtures`.

## K18 — Add F1 descriptor fixtures

- [x] **K18: complete** — owner lane: `descriptor`. Evidence: [evidence/K18.json](evidence/K18.json).

**Start after:** K17, P10. **Additional acceptance prerequisites:** none.

**Files:** `tests/native-can/subjects/descriptors/f1/`.

**Concrete change:** Binary stdin, Unicode/empty/missing names, malformed and unused snapshot fixtures. Keep stimuli as Can subjects or fixed inert bytes; scenario decisions and expectations are authored at QF1 in ordinary Can. Existing Go CLI remains the subject.

**Acceptance checks:** Offered/accepted bytes, EOF, writer exit and child result stay separate; non-reader does not strand writer. Use local or bounded bootstrap controls only; live host-dependent controls wait for the corresponding qualified profile and Q task.

**Completion evidence:** Bounded wrong-fd, withheld-EOF, truncation and sanitation controls; QF1 holds credit. Record under evidence/K18.json.

**Early commit:** `test: cover descriptor delivery and startup isolation`.

## K19 — Add inherited fd 4 lease fixture

- [ ] **K19: planned** — owner lane: `descriptor`.

**Start after:** K18, P10, P27. **Additional acceptance prerequisites:** none.

**Files:** `tests/native-can/subjects/descriptors/f2/`.

**Concrete change:** Gate child readiness, launcher exit, clean attempt, child release and prune observations. Subject helpers expose raw readiness/lease facts only; QF2 owns the Can sequence and verdict.

**Acceptance checks:** Actual inherited kernel lease protects generation; early/missing lease control loses protection. Use local or bounded bootstrap controls only; live host-dependent controls wait for the corresponding qualified profile and Q task.

**Completion evidence:** Bounded lease controls and cleanup receipt; QF2 holds credit. Record under evidence/K19.json.

**Early commit:** `test: cover child-held generation lease`.

## K21 — Implement external WebSocket client observations

- [ ] **K21: blocked** — owner lane: `peer`. Evidence: [evidence/K21.json](evidence/K21.json).

**Audit correction:** Historical local controls remain recorded, but accepted prerequisites were invalidated by this audit. Revalidate their current receipts before restoring aggregate completion.

**Start after:** K20. **Additional acceptance prerequisites:** none.

**Files:** `tools/runtime/test-services/http-peer/websocket*`.

**Concrete change:** Connect/send/event/close with opcode, payload and distinct local/remote close facts.

**Acceptance checks:** Dropped event and forged remote-close controls detectable; bounded pending work. Use local or bounded bootstrap controls only; live host-dependent controls wait for the corresponding qualified profile and Q task.

**Completion evidence:** Focused protocol checks; QWS holds Can credit. Record under evidence/K21.json.

**Early commit:** `feat(test): expose external websocket events`.

## K22 — Implement independent raw DB observer

- [ ] **K22: blocked** — owner lane: `database`. Evidence: [evidence/K22.json](evidence/K22.json).

**Audit correction:** Historical local controls remain recorded, but accepted prerequisites were invalidated by this audit. Revalidate their current receipts before restoring aggregate completion.

**Start after:** P29. **Additional acceptance prerequisites:** none.

**Files:** `tools/runtime/test-services/db-observer/core*`, `tools/native-test-owner/external/db_observer.go`.

**Concrete change:** Owned namespace, pinned raw connection, typed exact cells, engine/driver and SQL error provenance.

**Acceptance checks:** Rows retain exact number/text/bytes/null/order; independence scope from Can adapter stated. Use local or bounded bootstrap controls only; live host-dependent controls wait for the corresponding qualified profile and Q task.

**Completion evidence:** Focused transport controls and namespace receipt. Record under evidence/K22.json.

**Early commit:** `feat(test): add typed independent database observer`.

## K23 — Observe compiled RETURNING and final rows

- [ ] **K23: blocked** — owner lane: `database`. Evidence: [evidence/K23.json](evidence/K23.json).

**Audit correction:** Historical local controls remain recorded, but accepted prerequisites were invalidated by this audit. Revalidate their current receipts before restoring aggregate completion.

**Start after:** K22. **Additional acceptance prerequisites:** none.

**Files:** `tools/runtime/test-services/db-observer/returning*`.

**Concrete change:** D1 raw row-count and typed row observations beside ordinary C fixture.

**Acceptance checks:** Raw RETURNING alone cannot credit C; bigint narrowing and wrong row schema detectable. Use local or bounded bootstrap controls only; live host-dependent controls wait for the corresponding qualified profile and Q task.

**Completion evidence:** Selected PG/SQLite bounded controls; QD1 holds credit. Record under evidence/K23.json.

**Early commit:** `feat(test): observe returning rows independently`.

## K24 — Expose fresh-pool transaction identity

- [ ] **K24: blocked** — owner lane: `database`. Evidence: [evidence/K24.json](evidence/K24.json).

**Audit correction:** Engine settlement and release observations are supplied strings in an in-memory model; the explicitly required engine-specific observation adapter remains missing.

**Start after:** K22, K23, P12. **Additional acceptance prerequisites:** none.

**Files:** `tools/runtime/test-services/db-observer/transactions*`, `tools/native-test-owner/external/db_transaction.go`.

**Concrete change:** Callback entry, pinned connection, actor identity and settled rollback facts. Implement engine-specific settlement observations and release acknowledgment without deciding rollback expectations in native code.

**Acceptance checks:** Swapped actor and wrong-handle LAST_INSERT_ID detectable regardless of coincidentally correct values. Use local or bounded bootstrap controls only; live host-dependent controls wait for the corresponding qualified profile and Q task.

**Completion evidence:** One bounded eight-actor control, no warmup; QD2 holds credit. Record under evidence/K24.json.

**Early commit:** `feat(test): observe fresh-pool transaction identity`.

## K25 — Add poisoned transaction sentinel observations

- [ ] **K25: blocked** — owner lane: `database`. Evidence: [evidence/K25.json](evidence/K25.json).

**Audit correction:** Historical local controls remain recorded, but accepted prerequisites were invalidated by this audit. Revalidate their current receipts before restoring aggregate completion.

**Start after:** K22, K23, P29. **Additional acceptance prerequisites:** none.

**Files:** `tools/runtime/test-services/db-observer/poison*`.

**Concrete change:** PG error codes, terminal settlement and fresh raw sentinel/replay reads.

**Acceptance checks:** Omit-poison control reveals committed sentinel; callback success never proves commit. Use local or bounded bootstrap controls only; live host-dependent controls wait for the corresponding qualified profile and Q task.

**Completion evidence:** Bounded poison/control facts; QD3 holds credit. Record under evidence/K25.json.

**Early commit:** `feat(test): witness poisoned transaction rollback`.

## K26 — Separate SQL deadline from server settlement

- [ ] **K26: blocked** — owner lane: `database`. Evidence: [evidence/K26.json](evidence/K26.json).

**Audit correction:** The required engine-specific server quiescence/fence adapter is an in-memory boolean model. A postgres capability label supplies no native cancel channel or server acknowledgment.

**Start after:** K22, K24, P29. **Additional acceptance prerequisites:** none.

**Files:** `tools/runtime/test-services/db-observer/deadline*`, `tools/native-test-owner/external/db_settlement.go`.

**Concrete change:** Dispatch, visible deadline, driver settlement, server acknowledgment/quiescence, retained lease and fence facts. Implement the engine-specific N quiescence/fence adapter and retained namespace authority; do not assume a generic P29 grant supplies server cancellation.

**Acceptance checks:** At most one dispatch; unknown server effect blocks cleanup; no unsupported cancellation claim. Use local or bounded bootstrap controls only; live host-dependent controls wait for the corresponding qualified profile and Q task.

**Completion evidence:** Bounded delayed-write/prestart controls; QD4 holds credit. Record under evidence/K26.json.

**Early commit:** `feat(test): reconcile SQL work after visible deadline`.

## K27 — Implement owned object-store observations

- [ ] **K27: blocked** — owner lane: `object-store`. Evidence: [evidence/K27.json](evidence/K27.json).

**Audit correction:** Historical local controls remain recorded, but accepted prerequisites were invalidated by this audit. Revalidate their current receipts before restoring aggregate completion.

**Start after:** P29. **Additional acceptance prerequisites:** none.

**Files:** `tools/runtime/test-services/object-store/service*`, `tools/native-test-owner/external/object_store.go`.

**Concrete change:** Scoped prefix, put/get/list/delete/close, exact bytes, pagination and accepted-write settlement.

**Acceptance checks:** Caller prefix is not authority; incomplete pages/eventual empty list cannot prove cleanup; shared bucket untouched. Use local or bounded bootstrap controls only; live host-dependent controls wait for the corresponding qualified profile and Q task.

**Completion evidence:** Focused pending-write and paginated-cleanup controls; QStore holds Can credit. Record under evidence/K27.json.

**Early commit:** `feat(test): own object-store prefix observations`.

## QN1 — Qualify integrated N1 Can case

- [ ] **QN1: planned** — owner lane: `acceptance-native`.

**Start after:** P23, I01. **Additional acceptance prerequisites:** none.

**Files:** `tests/native-can/qualification/qn1/**`.

**Concrete change:** Can owns N1 descriptors, sequencing and exact expectations.

**Acceptance checks:** Full N1 matrix plus assimilation/normalization/rounding/alias controls, final seal and N receipt. Before crediting results, independently review this case/check/expected-vector snapshot and bind the R-built suite S/A identities; an older accepted suite snapshot cannot authorize newly added test code. Require the complete N1 gate (fixture scope, Accept, Controls and Bounds) and shared execution contract in normative_gate_source; the condensed card grants no omission.

**Completion evidence:** Can case/check IDs, R/C/observer IDs, seeded controls and cleanup receipt. Record evidence/QN1.json with qualified scope and all unrun engine legs.

**Full normative gate:** [N1 — Inert, exact native-value transport](../../../docs/syntax-taste/preparation/native-can-test-hard-cases-2026-09-30/experiments.md:100). Require the complete shared execution contract and gate, including all fixture, Accept, Controls and Bounds details.

**Early commit:** `test: qualify exact native value transport`.

## QN2 — Qualify integrated N2 C adapter ingress

- [ ] **QN2: planned** — owner lane: `acceptance-native`.

**Start after:** QN1, I02. **Additional acceptance prerequisites:** none.

**Files:** `tests/native-can/qualification/qn2/**`.

**Concrete change:** Can invokes ordinary C-compiled subject and checks independent boundary witness.

**Acceptance checks:** Bypass, wrong export, eager getter and forged completion controls fail. Require independent version-bound invocation and local-completion authentication plus instrumented/uninstrumented parity; a C-supplied invoked flag is insufficient. Before crediting results, independently review this case/check/expected-vector snapshot and bind the R-built suite S/A identities; an older accepted suite snapshot cannot authorize newly added test code. Require the complete N2 gate (fixture scope, Accept, Controls and Bounds) and shared execution contract in normative_gate_source; the condensed card grants no omission.

**Completion evidence:** Generated artifact/import IDs, witness facts, Can report and N receipt. Record evidence/QN2.json with qualified scope and all unrun engine legs.

**Full normative gate:** [N2 — Hostile value reaches the actual C adapter](../../../docs/syntax-taste/preparation/native-can-test-hard-cases-2026-09-30/experiments.md:114). Require the complete shared execution contract and gate, including all fixture, Accept, Controls and Bounds details.

**Early commit:** `test: qualify generated C native ingress`.

## QN3 — Qualify integrated N3 late identity

- [ ] **QN3: planned** — owner lane: `acceptance-native`.

**Start after:** QN2, I03. **Additional acceptance prerequisites:** none.

**Files:** `tests/native-can/qualification/qn3/**`.

**Concrete change:** Can controls gates, selected occurrence and lease release.

**Acceptance checks:** Late event/identity/early release controls fail; sealed complete interval and cleanup. Before crediting results, independently review this case/check/expected-vector snapshot and bind the R-built suite S/A identities; an older accepted suite snapshot cannot authorize newly added test code. Require the complete N3 gate (fixture scope, Accept, Controls and Bounds) and shared execution contract in normative_gate_source; the condensed card grants no omission.

**Completion evidence:** Separate raw/C leg reports and receipts. Record evidence/QN3.json with qualified scope and all unrun engine legs.

**Full normative gate:** [N3 — Late result, occurrence identity and lease release](../../../docs/syntax-taste/preparation/native-can-test-hard-cases-2026-09-30/experiments.md:128). Require the complete shared execution contract and gate, including all fixture, Accept, Controls and Bounds details.

**Early commit:** `test: qualify late native occurrence identity`.

## QB0 — Qualify browser host admission and launch ownership

- [ ] **QB0: planned** — owner lane: `acceptance-browser`.

**Start after:** P14, P23, I06. **Additional acceptance prerequisites:** none.

**Files:** `tests/native-can/qualification/qb0/**`.

**Concrete change:** Independently qualify observer scope and enforcing launch profile before admitting the first bounded browser. Then check effective pinned-driver profile, launch-to-disposal host interval, service death and inherited children. Unobservable host effects fail closed; do not launch to discover whether an unqualified observer works.

**Acceptance checks:** Observer-loss and planted host-effect controls invalidate admission; prompt/unknown interval or unconfirmed disposal cannot produce success. Noninteractive credential behavior and strict host resource profile are proven independently of browser flags. PM-B1 remains recorded failed; do not re-credit its route-only result. Before crediting results, independently review this case/check/expected-vector snapshot and bind the R-built suite S/A identities; an older accepted suite snapshot cannot authorize newly added test code. Demonstrate observer continuity across driver/service death and conservative invalidation on observer loss.

**Completion evidence:** evidence/QB0.json: accepted host/observer/browser profile, bounded controls, disposal receipts and correction registry

**Early commit:** `test(testing): qualify browser host isolation`.

## QHTTP — Qualify Can-controlled peer and HTTP observations

- [ ] **QHTTP: planned** — owner lane: `acceptance-peer`.

**Start after:** P23, I04. **Additional acceptance prerequisites:** none.

**Files:** `tests/native-can/qualification/qhttp/**`.

**Concrete change:** Can chooses raw bytes, read/write/half-close scheduling, request policy and expected status/body/EOF; native peer returns finite observations.

**Acceptance checks:** Wrong bytes, missing EOF, truncated body, hidden retry/redirect and incomplete cleanup controls fail. Exact header provenance and stream completeness are explicit. Before crediting results, independently review this case/check/expected-vector snapshot and bind the R-built suite S/A identities; an older accepted suite snapshot cannot authorize newly added test code.

**Completion evidence:** evidence/QHTTP.json: Can cases, controls, peer seals and N cleanup receipt

**Early commit:** `test(testing): qualify controlled HTTP observations`.

## QB1base — Qualify pending browser input and routes without DB replay

- [ ] **QB1base: planned** — owner lane: `acceptance-browser`.

**Start after:** QB0, QHTTP, I06, P23. **Additional acceptance prerequisites:** none.

**Files:** `tests/native-can/qualification/qb1base/**`.

**Concrete change:** Ordinary Can checks held-route pending click, abort/close, captured body, duplicate contact and missing receipt controls. Save input/contact/delivery/application facts separately. This qualifies shared browser mechanics only; durable DB replay is QB1.

**Acceptance checks:** Actual input settlement observed independently; route abort need not reject click. A plausible response, missing seal, dropped event or host-effect gap fails; Can supplies all sequencing and expected outcomes. Before crediting results, independently review this case/check/expected-vector snapshot and bind the R-built suite S/A identities; an older accepted suite snapshot cannot authorize newly added test code.

**Completion evidence:** evidence/QB1base.json: per-engine cases/controls, route journal, host interval and N receipt; explicitly not full B1

**Early commit:** `test(testing): qualify pending input and route mechanics`.

## QD1 — Qualify integrated D1 RETURNING

- [ ] **QD1: planned** — owner lane: `acceptance-database`.

**Start after:** P23, I13. **Additional acceptance prerequisites:** none.

**Files:** `tests/native-can/qualification/qd1/**`.

**Concrete change:** Can exercises C RETURNING and checks independent final rows.

**Acceptance checks:** PG/SQLite selected legs and narrowing/wrong-row controls; absent service blocked. Before crediting results, independently review this case/check/expected-vector snapshot and bind the R-built suite S/A identities; an older accepted suite snapshot cannot authorize newly added test code. Require the complete D1 gate (fixture scope, Accept, Controls and Bounds) and shared execution contract in normative_gate_source; the condensed card grants no omission.

**Completion evidence:** C path, raw rows, Can checks and namespace receipt. Record evidence/QD1.json with qualified scope and all unrun engine legs.

**Full normative gate:** [D1 — Compiled Can RETURNING and raw row fidelity](../../../docs/syntax-taste/preparation/native-can-test-hard-cases-2026-09-30/experiments.md:174). Require the complete shared execution contract and gate, including all fixture, Accept, Controls and Bounds details.

**Early commit:** `test: qualify returning rows in Can`.

## QB1 — Qualify integrated B1 pending route

- [ ] **QB1: planned** — owner lane: `acceptance-browser`.

**Start after:** QB1base, QD1. **Additional acceptance prerequisites:** none.

**Files:** `tests/native-can/qualification/qb1/**`.

**Concrete change:** Can owns pending-click, route abort and durable corruption/replay choreography. Reuse only the application payload operation_id on replay; new native action/request/route IDs are mandatory. Never redispatch an uncertain native operation.

**Acceptance checks:** Input, upstream contact, browser delivery, application witness, native/app IDs, seal and host effects separately pass controls. Full B1 is still unrun until durable corruption/replay controls pass; QB1base alone does not close B1. Before crediting results, independently review this case/check/expected-vector snapshot and bind the R-built suite S/A identities; an older accepted suite snapshot cannot authorize newly added test code. Require the complete B1 gate (fixture scope, Accept, Controls and Bounds) and shared execution contract in normative_gate_source; the condensed card grants no omission.

**Completion evidence:** Per-engine Can checks, route/DB facts, host observer and cleanup receipts. Record evidence/QB1.json with qualified scope and all unrun engine legs.

**Full normative gate:** [B1 — Can resolve a route while input is pending](../../../docs/syntax-taste/preparation/native-can-test-hard-cases-2026-09-30/experiments.md:28). Require the complete shared execution contract and gate, including all fixture, Accept, Controls and Bounds details.

**Early commit:** `test: qualify browser route delivery and settlement`.

## QB2 — Qualify integrated B2 DOM semantics

- [ ] **QB2: planned** — owner lane: `acceptance-browser`.

**Start after:** QB1base, I07. **Additional acceptance prerequisites:** none.

**Files:** `tests/native-can/qualification/qb2/**`.

**Concrete change:** Can owns same-task clicks, clone/missing target, physical/synthetic events and cancellation checks.

**Acceptance checks:** Yield, cancellation, clone and missing-occurrence controls fail; pre-submit and in-flight absence distinct. Before crediting results, independently review this case/check/expected-vector snapshot and bind the R-built suite S/A identities; an older accepted suite snapshot cannot authorize newly added test code. Require the complete B2 gate (fixture scope, Accept, Controls and Bounds) and shared execution contract in normative_gate_source; the condensed card grants no omission.

**Completion evidence:** Per-engine Can checks, event seal and host receipt. Record evidence/QB2.json with qualified scope and all unrun engine legs.

**Full normative gate:** [B2 — Preserve immediate DOM action and event semantics](../../../docs/syntax-taste/preparation/native-can-test-hard-cases-2026-09-30/experiments.md:42). Require the complete shared execution contract and gate, including all fixture, Accept, Controls and Bounds details.

**Early commit:** `test: qualify browser DOM event semantics`.

## QB3 — Qualify integrated B3 page Fetch

- [ ] **QB3: planned** — owner lane: `acceptance-browser`.

**Start after:** QB0, QHTTP, I08, P23. **Additional acceptance prerequisites:** none.

**Files:** `tests/native-can/qualification/qb3/**`.

**Concrete change:** Can chooses same-origin cookie/generation/CSP requests.

**Acceptance checks:** CSP forbidden origin has sealed independent no-contact; external HTTP substitution fails. Before crediting results, independently review this case/check/expected-vector snapshot and bind the R-built suite S/A identities; an older accepted suite snapshot cannot authorize newly added test code. Require the complete B3 gate (fixture scope, Accept, Controls and Bounds) and shared execution contract in normative_gate_source; the condensed card grants no omission.

**Completion evidence:** Page/server Can checks, listener seal and host receipt. Record evidence/QB3.json with qualified scope and all unrun engine legs.

**Full normative gate:** [B3 — Page-origin Fetch](../../../docs/syntax-taste/preparation/native-can-test-hard-cases-2026-09-30/experiments.md:56). Require the complete shared execution contract and gate, including all fixture, Accept, Controls and Bounds details.

**Early commit:** `test: qualify page-origin fetch policy`.

## QB4 — Qualify integrated B4 Can browser codec

- [ ] **QB4: planned** — owner lane: `acceptance-browser`.

**Start after:** QB0, QN1, I09, P23. **Additional acceptance prerequisites:** none.

**Files:** `tests/native-can/qualification/qb4/**`.

**Concrete change:** Ordinary Can browser fixture exports exact codec facts for R Can oracle.

**Acceptance checks:** Unsafe int64/-0, variant/nested and decode rejection; malformed/missing ready/export controls fail. Before crediting results, independently review this case/check/expected-vector snapshot and bind the R-built suite S/A identities; an older accepted suite snapshot cannot authorize newly added test code. Require the complete B4 gate (fixture scope, Accept, Controls and Bounds) and shared execution contract in normative_gate_source; the condensed card grants no omission.

**Completion evidence:** C artifact identities, browser/Bun facts, Can checks and receipts. Record evidence/QB4.json with qualified scope and all unrun engine legs.

**Full normative gate:** [B4 — Ordinary Can browser codec facts](../../../docs/syntax-taste/preparation/native-can-test-hard-cases-2026-09-30/experiments.md:70). Require the complete shared execution contract and gate, including all fixture, Accept, Controls and Bounds details.

**Early commit:** `test: qualify browser codec facts in Can`.

## QB5 — Qualify integrated B5 remote Firefox cleanup

- [ ] **QB5: planned** — owner lane: `acceptance-browser`.

**Start after:** QB0, I10, P23. **Additional acceptance prerequisites:** none.

**Files:** `tests/native-can/qualification/qb5/**`.

**Concrete change:** Can checks worker death, first context close and second context on same server.

**Acceptance checks:** Shared server survives and case close-server rejects before effect; unknown close blocks. Before crediting results, independently review this case/check/expected-vector snapshot and bind the R-built suite S/A identities; an older accepted suite snapshot cannot authorize newly added test code. Require the complete B5 gate (fixture scope, Accept, Controls and Bounds) and shared execution contract in normative_gate_source; the condensed card grants no omission.

**Completion evidence:** Pinned server instance, Can check and N cleanup receipt. Record evidence/QB5.json with qualified scope and all unrun engine legs.

**Full normative gate:** [B5 — Shared remote Firefox survives case death](../../../docs/syntax-taste/preparation/native-can-test-hard-cases-2026-09-30/experiments.md:84). Require the complete shared execution contract and gate, including all fixture, Accept, Controls and Bounds details.

**Early commit:** `test: qualify remote browser context recovery`.

## QF1 — Qualify integrated F1 descriptors

- [ ] **QF1: planned** — owner lane: `acceptance-descriptor`.

**Start after:** P23, I11. **Additional acceptance prerequisites:** none.

**Files:** `tests/native-can/qualification/qf1/**`.

**Concrete change:** Can owns binary/Unicode/malformed/unused snapshot and startup policy cases.

**Acceptance checks:** Independent fd 3 EOF, blocked writer cleanup and wrong-fd/withheld-EOF/sanitization controls. Before crediting results, independently review this case/check/expected-vector snapshot and bind the R-built suite S/A identities; an older accepted suite snapshot cannot authorize newly added test code. Require the complete F1 gate (fixture scope, Accept, Controls and Bounds) and shared execution contract in normative_gate_source; the condensed card grants no omission.

**Completion evidence:** Can checks and writer/child/descriptor receipts. Record evidence/QF1.json with qualified scope and all unrun engine legs.

**Full normative gate:** [F1 — Credentials, EOF and launch isolation](../../../docs/syntax-taste/preparation/native-can-test-hard-cases-2026-09-30/experiments.md:144). Require the complete shared execution contract and gate, including all fixture, Accept, Controls and Bounds details.

**Early commit:** `test: qualify descriptor delivery in Can`.

## QF2 — Qualify integrated F2 inherited lease

- [ ] **QF2: planned** — owner lane: `acceptance-descriptor`.

**Start after:** QF1, I12. **Additional acceptance prerequisites:** none.

**Files:** `tests/native-can/qualification/qf2/**`.

**Concrete change:** Can checks child-held fd 4 across launcher exit and prune.

**Acceptance checks:** Generation lives only until confirmed lease release; missing/early lease controls fail. Before crediting results, independently review this case/check/expected-vector snapshot and bind the R-built suite S/A identities; an older accepted suite snapshot cannot authorize newly added test code. Require the complete F2 gate (fixture scope, Accept, Controls and Bounds) and shared execution contract in normative_gate_source; the condensed card grants no omission.

**Completion evidence:** Can check and N generation/child receipts. Record evidence/QF2.json with qualified scope and all unrun engine legs.

**Full normative gate:** [F2 — Generation lease outlives its launcher](../../../docs/syntax-taste/preparation/native-can-test-hard-cases-2026-09-30/experiments.md:158). Require the complete shared execution contract and gate, including all fixture, Accept, Controls and Bounds details.

**Early commit:** `test: qualify inherited generation leases`.

## QWS — Qualify Can HTTP/peer/WebSocket cases

- [ ] **QWS: planned** — owner lane: `acceptance-peer`.

**Start after:** QHTTP, I05. **Additional acceptance prerequisites:** none.

**Files:** `tests/native-can/qualification/qws/**`.

**Concrete change:** Can supplies bytes, hold/release schedule, status/body/message/shutdown expectations.

**Acceptance checks:** Missing EOF/event, wrong close provenance and incomplete peer cleanup fail. Before crediting results, independently review this case/check/expected-vector snapshot and bind the R-built suite S/A identities; an older accepted suite snapshot cannot authorize newly added test code.

**Completion evidence:** Can check IDs, peer event seals and N receipts. Record evidence/QWS.json with qualified scope and all unrun engine legs.

**Early commit:** `test: qualify controlled peer protocols in Can`.

## QD2 — Qualify integrated D2 fresh pool

- [ ] **QD2: planned** — owner lane: `acceptance-database`.

**Start after:** QD1, I14. **Additional acceptance prerequisites:** none.

**Files:** `tests/native-can/qualification/qd2/**`.

**Concrete change:** Can runs one eight-actor burst and rollback with independent final reads.

**Acceptance checks:** No warmup, exact actor/connection identity and wrong-handle controls; MySQL separate. Before crediting results, independently review this case/check/expected-vector snapshot and bind the R-built suite S/A identities; an older accepted suite snapshot cannot authorize newly added test code. Require the complete D2 gate (fixture scope, Accept, Controls and Bounds) and shared execution contract in normative_gate_source; the condensed card grants no omission.

**Completion evidence:** Can checks, native settlement IDs and namespace receipt. Record evidence/QD2.json with qualified scope and all unrun engine legs.

**Full normative gate:** [D2 — Fresh-pool callback and connection ownership](../../../docs/syntax-taste/preparation/native-can-test-hard-cases-2026-09-30/experiments.md:188). Require the complete shared execution contract and gate, including all fixture, Accept, Controls and Bounds details.

**Early commit:** `test: qualify fresh-pool transaction scope`.

## QD3 — Qualify integrated D3 rollback sentinel

- [ ] **QD3: planned** — owner lane: `acceptance-database`.

**Start after:** QD1, I15. **Additional acceptance prerequisites:** none.

**Files:** `tests/native-can/qualification/qd3/**`.

**Concrete change:** Can inserts sentinel, poisons PG transaction and checks replay via raw observer.

**Acceptance checks:** 23505/25P02, settled rollback, absent sentinel and omit-poison control. Before crediting results, independently review this case/check/expected-vector snapshot and bind the R-built suite S/A identities; an older accepted suite snapshot cannot authorize newly added test code. Require the complete D3 gate (fixture scope, Accept, Controls and Bounds) and shared execution contract in normative_gate_source; the condensed card grants no omission.

**Completion evidence:** Can checks, server/driver settlement and namespace receipt. Record evidence/QD3.json with qualified scope and all unrun engine legs.

**Full normative gate:** [D3 — Poisoned transaction has an independent rollback witness](../../../docs/syntax-taste/preparation/native-can-test-hard-cases-2026-09-30/experiments.md:202). Require the complete shared execution contract and gate, including all fixture, Accept, Controls and Bounds details.

**Early commit:** `test: qualify poisoned transaction rollback`.

## QD4 — Qualify integrated D4 SQL deadline

- [ ] **QD4: planned** — owner lane: `acceptance-database`.

**Start after:** QD1, I16. **Additional acceptance prerequisites:** none.

**Files:** `tests/native-can/qualification/qd4/**`.

**Concrete change:** Can separates visible deadline from driver/server settlement and later raw reads.

**Acceptance checks:** Stable commit_unknown, at-most-once dispatch, retained lease, prestart no-dispatch; unresolved blocks cleanup. Before crediting results, independently review this case/check/expected-vector snapshot and bind the R-built suite S/A identities; an older accepted suite snapshot cannot authorize newly added test code. Require the complete D4 gate (fixture scope, Accept, Controls and Bounds) and shared execution contract in normative_gate_source; the condensed card grants no omission.

**Completion evidence:** Can checks, fence/quiescence facts and namespace receipt. Record evidence/QD4.json with qualified scope and all unrun engine legs.

**Full normative gate:** [D4 — Visible SQL deadline versus native settlement](../../../docs/syntax-taste/preparation/native-can-test-hard-cases-2026-09-30/experiments.md:216). Require the complete shared execution contract and gate, including all fixture, Accept, Controls and Bounds details.

**Early commit:** `test: qualify SQL deadline settlement`.

## QStore — Qualify Can object-store cases

- [ ] **QStore: planned** — owner lane: `acceptance-object-store`.

**Start after:** P23, I17. **Additional acceptance prerequisites:** none.

**Files:** `tests/native-can/qualification/qstore/**`.

**Concrete change:** Can supplies object names/content, faults and durability expectations.

**Acceptance checks:** Owned prefix, full pagination, accepted-write quiescence and qualified consistency before deletion. Before crediting results, independently review this case/check/expected-vector snapshot and bind the R-built suite S/A identities; an older accepted suite snapshot cannot authorize newly added test code.

**Completion evidence:** Can checks, service identity and cleanup receipt. Record evidence/QStore.json with qualified scope and all unrun engine legs.

**Early commit:** `test: qualify owned object-store lifecycle`.

