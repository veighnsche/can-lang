# Native Can migration dispatch and integration plan

This is a planning view of [143 master tasks](tasks.json), not 143 additional
assignments. Companion ledgers are projections of the same identities. Sub-delivery
labels and row/file slices never increase task or coverage totals. The
[machine-readable plan](lane-plan.json) assigns every master ID exactly once and
records its exact prerequisites, owned paths, evidence location, slice boundaries
and all 867 dependency edges. Original acceptance text remains authoritative in
`tasks.json`; migration rows retain their exact `row_gates` and retained-only gates.

No implementation, acceptance run, performance measurement, host qualification or
coverage retirement is authorized or credited by this reorganization.

## Dispatch from the corrected frontier

Read current task status and evidence before dispatch. `start_after` means accepted
completion before source drafting/local checks; it is never a promise that another
worker will eventually finish. `accept_after` adds acceptance prerequisites without
blocking a task whose source prerequisites have already been accepted. Use both
lists for parent completion. A blocked predecessor does not release its consumers.

At this audit's corrected frontier, P12, K05, M32, M33 and Z01 have accepted source
prerequisites. P12 admission repair is the first core unlock. K05 needs an actual C-entry witness
hook and its bounded independent controls; K06 waits for accepted K05. P28 and K07 require accepted
P12 and must wait. P13/P14 have unresolved strict host acceptance; previously bounded
local controls remain useful evidence but cannot qualify the host envelope.

| Stage | Worker A | Worker B | Worker C | Integrator/reviewer |
| --- | --- | --- | --- | --- |
| Initial frontier | P12 admission floor/per-allocation repair | K05 actual C witness after native-domain handoff; fallback M32/M33 | Z01 Can coverage source; steal M33 if ready and unowned | Review P12 and K05/local controls and reserved shared-path patches |
| P12 accepted | P27 keys/producer/leases, then K19 | K07 observer; K08 only after K07 and P29 reaccept; ready browser mechanism repairs | M33; Z01 source; disjoint historical/coverage evidence | Immediately revalidate P29 local controls, then P28 bindings; shared generator/emitter/runtime changes |
| P27/P28 accepted | P18 staging then P19 controller/worker; submit driver hooks | Ready base I helper packages (I04/I13/I11/I17); K24/K26 only when their own start prerequisites accept | Ready M11/M27/M28 source and per-row gates; I01 helper may be stolen | Shared hooks; I binding queue; P21 dispatch after P19; P22 handoffs |
| P19 accepted | P22 authority and correction controls | Continue dependency-ready service/helper repairs | Eligible M source/row evidence | P21 and P26; independent R/N identities, no candidate promotion |
| Core and host accepted | QN1→QN2→QN3; QF1→QF2; P24 when ready | QHTTP and QD1; QD4/QD2/QD3/QWS/QStore as actual gates accept | QB0→QB1base, then QB1/QB2; QB3/QB4/QB5 by exact prerequisites | P23 minimal suite acceptance; separate I promotions/receipts and serialized heavy phases |
| Qualification frontiers open | Native/CLI/core migration rows | DB/transport/installed-artifact rows | Application/browser rows | Accumulate Z01 row reconciliation; Z02 only after aggregate Z01 accepts; ready Z03 caller work |
| Remaining scope accepted | Ready repairs and affected checks | Ready repairs and affected checks | Ready repairs and affected checks | Z04 remaining bootstrap retirement; Z05 full matrix; Z06 independent review/fixes |

Each table cell is a default queue, not a whole-stage barrier. Missing strict host
or service acceptance stops affected qualification. Continue source slices whose
actual `start_after` entries are accepted. Never start Q authoring ahead of its
parent's existing qualification prerequisites. For M source work, each exact row's
start gates permit progress without waiting for the group's union; execution credit
still needs every exact row acceptance gate, including retained-only conditions.

## Ready queues and ownership

The queues below cover all 143 identities, including already accepted tasks retained
for identity and any future invalidation. Skip accepted unchanged work; prioritise
ready unlocks and the longest remaining path, rather than replaying completed tasks.
Exact prerequisite/path/evidence records for each ID are in `lane-plan.json`.

| Queue | Ordered parent IDs | Ownership and handoff |
| --- | --- | --- |
| A: core, native, descriptors | P12, P27, P18, P19, P22, P24, K19, QN1, QN2, QN3, QF1, QF2; M01–M06, M09–M11, M27–M31, M35, M37 | Task-owned Can controller/worker/report/build/verification and qualification/migration prefixes; integrator applies all shared compiler/runtime hooks. P22 corrections path transfers from the owner writer before edits. |
| B: ownership and services | P07–P11, P13–P14, P25, P29; K05, K07–K11, K13–K16, K20–K27; QHTTP, QD1, QD4, QD2, QD3, QWS, QStore; M32, M22–M26, M38–M43 | Exact owner/service subpaths and distinct DB/HTTP/WS qualification/migration prefixes. Existing shared owner prefixes require scoped handoff, not concurrent broad-directory ownership. |
| C: coverage, browser and application | P15–P17; K01–K04, K06, K17–K18; QB0, QB1base, QB1–QB5; M33–M34, M36, M12–M21, M07–M08, M44–M45 | Disjoint Can qualification/migration prefixes. Delegate Z01's `tests/native-can/src/coverage/migration/` source to C while integrator retains aggregate evidence. |
| Integrator/reference | P00–P06, P20–P21, P28, P26, P23; I01–I17; Z01–Z06 | Shared schemas/reference manifests/compiler/catalogue/runtime edits, generated mirrors, global formatting, all commits, old-harness deletion and actual caller activation. |

Source workers prepare complete I helper/example slices in these disjoint paths:
`tests/native-can/src/capabilities/iNN/` and
`tests/native-can/examples/capabilities/iNN/`. A takes I01/I02/I03/I11/I12; B takes
I04/I05/I13/I14/I15/I16/I17; C takes I06/I07/I08/I09/I10. These are delegated source
parts of integrator-owned I parents, not duplicate tasks. Parent `start_after` still
controls readiness; integrator writes shared bindings and records separate promotion.

The integration queue immediately revalidates P29 after accepted P12 to reopen
K20/K22 service branches, then favours P28/P18/P19/P21/P22/P26/P23 core unlocks and
I04→I06 and I13→I14→I16, which feed browser/DB bottlenecks. Fit I01→I02→I03 and
I11→I12 alongside ready native/descriptor demand; I05/I07–I10/I15/I17 follow their
actual qualified row demand. Do not add any of those priority relationships to the
DAG. A ready unrelated slice need not wait for an unavailable preferred slice.

When a worker's preferred queue is blocked, take any ready unowned slice: first a
critical-path unlock, then helper/case source, row disposition/closure evidence or a
scoped review. Record exact ownership transfer in progress before writing. Pause
all affected writers before an integrator edit, formatter or commit. An immutable
review snapshot may be reviewed while a different reserved prefix is authored;
changes invalidate its closure and need a new review.

## Review-sized deliveries

Target one operation contract, one control family, or one row/facet slice per review;
roughly 200–400 changed lines is a useful attention target, never an acceptance cap.
If the coherent change is larger, preserve its complete control contract. Each
sub-delivery gets independently checkable evidence in the parent evidence record:
commit, exact paths, checks, positive/defect/missing-evidence controls, identities,
source closure, review disposition and cleanup. Parent acceptance stays indivisible
where its contract requires a join.

| Oversized parent | Independent source/evidence deliveries | Join retained for complete |
| --- | --- | --- |
| P12 | Admission floor/per-allocation check; capacity/deadlines; race/rejection controls | Full admission contract; strict host qualification remains P13/P14 |
| P27 | Key/policy; producer/publication/leases; drift/corruption/cancellation controls | All reuse checks; authoritative within-run acceptance remains P23 |
| P28 | Typed operation descriptors; transport/owner adapters; emitter/generated mirrors; rejection controls | Complete cross-layer contract; separate R-core promotion remains P26 |
| P18/P19/P22 | Can policy/contract; thin shared hook patch; independent local control deltas | All original checks and dependencies; no provisional result becomes live acceptance |
| I01–I17 | Complete Can helper/example; shared binding; independent R/N and helper S/A promotion | Separate scope receipts, predecessor/native controls, exact current identities |
| Q parents | Reviewed case/vector snapshot; engine/profile positive and defect controls; observation/cleanup/correction evidence | Every required engine/profile/control leg; older S/A never authorises new case source |
| M parents | Exact row disposition; ordinary Can port; qualified row/retirement handoff | Every retained facet/variant/delegate and exact row gates; group complete only after all rows resolve |
| Z01/Z02/Z03/Z04 | One coverage row, file/symbol, caller, or bootstrap-policy scope | Aggregate completion remains gated by all original dependencies and shared deletion rules |

The following concrete ownership matrix strengthens the slice boundaries. Every
slice still waits for its parent's entire `start_after` list. Seal the named
cross-slice interface before parallel authoring; record the exact file reservation.
A proposed split creates files only inside the parent's canonical prefix, after
that seal; it does not authorize a new compiler/runtime path or a fourth worker.

| Parent / slice | Exclusive source paths and dependency | Independent slice acceptance / evidence |
| --- | --- | --- |
| K05 C-entry hook | B receives `tools/runtime/test-services/native-values/c-witness.ts` from C's native domain | Actual generated C export/signature path supplies independent facts; wrong export/signature and bypass detectable. `evidence/K05.json#subdeliveries.c-entry-hook` |
| K05 controls | Available existing worker receives `tools/runtime/test-services/native-values/c-witness-check.ts` after hook/observation contract seal | Eager read and forged local completion controls target actual subject/hook, with bounded independent facts. `evidence/K05.json#subdeliveries.c-witness-controls` |
| K07 observer | B owns `tools/native-test-owner/host/browser_observer/observer.go` and `observer_test.go` | Identified independent mechanism and coverage; observer loss/absence blocks. Headless/flags/HOME/samples do not prove isolation. `evidence/K07.json#subdeliveries.independent-observer` |
| K07 admission adapter | Available existing worker owns `tools/runtime/test-services/browser-driver/admission.ts` and `admission-check.ts` after observer receipt contract seal | Wrong/missing observer/profile identity, false completeness and lost continuity reject. `evidence/K07.json#subdeliveries.admission-adapter` |
| K26 engine settlement | B owns proposed canonical `tools/native-test-owner/external/db_settlement.go` | N retains namespace/recovery authority and reports actual server settlement/fence; unknown effect prevents clean release. `evidence/K26.json#subdeliveries.engine-quiescence` |
| K26 transport | Available existing worker owns `tools/runtime/test-services/db-observer/deadline.ts` and `deadline-check.ts` after settlement contract seal | Visible deadline differs from server settlement; no prestart dispatch/retry, unresolved delayed effect blocks cleanup. `evidence/K26.json#subdeliveries.deadline-transport` |
| P27 Can key/reuse policy | A owns proposed `tests/native-can/src/builds/policy/keys.can` and `reuse.can` after complete key contract seal | All inputs participate; input omissions/drift/corruption detected; within-run scope. `evidence/P27.json#subdeliveries.key-policy` |
| P27 native publication/lease | Integrator owns `compiler/internal/driver/test_reuse.go` after key/producer/lease contract seal | Atomic publication, one producer, waiter cancellation, real lease lifetime and no leased eviction. `evidence/P27.json#subdeliveries.publication-lease-hook` |
| P27 controls | Available existing worker owns proposed `tests/native-can/src/builds/controls/` after policy/hook seal | Exact drift/corrupt/partial-fill/cancel/evict positive and defect vectors; integrated credit remains P23. `evidence/P27.json#subdeliveries.reuse-controls` |
| P19 controller / worker / hook | A owns `tests/native-can/src/controller/`; available worker gets `tests/native-can/src/worker/` after protocol seal; integrator owns `compiler/internal/driver/test_launch.go` | Ordinary Can scheduling/completeness, fresh worker isolation and thin identified R/N handoff reviewed separately; full crash/timeout/control join remains parent acceptance. `evidence/P19.json#subdeliveries` |
| P22 authority / corrections | A owns `tests/native-can/src/report/authority/`; B receives `tools/native-test-owner/receipt/corrections/` after immutable chain schema seal and P11 writer release | Current matching receipts required; append-only corrections preserve failure and late effects. Separate `evidence/P22.json#subdeliveries` entries |
| Every I helper / binding / promotion | Helper worker owns exact `src/capabilities/iNN/` and `examples/capabilities/iNN/`; integrator owns canonical shared binding paths; integrator accepts manifests after both source slices and every parent acceptance prerequisite | Helper source/vector closure; generated binding/rejection controls; independent R/N/current S/A and cleanup each have separate `evidence/Ixx.json#subdeliveries` entries |

P12 floor and deadline slices share the existing `admission.go`/`admission_test.go`
files: the same writer handles them sequentially. JSON overrides preserve this
honest serialization. Existing worker availability determines helper/control
handoffs; a suggested second writer never adds another slot. The machine-readable
`slice_ownership_overrides` and `integration_slice_template` include each slice's
exact paths, dependencies, acceptance and expected evidence. All full parent checks
and required qualification joins remain unchanged.

The existing prose calls for incremental retirement, but canonical `Z02.start_after`
requires accepted aggregate Z01. A Z01 row receipt cannot satisfy that parent edge.
Workers accumulate independently reviewed Z01 row evidence while qualifying source
rows; Z02 waits for aggregate Z01 acceptance, then retires eligible files/symbols
individually without waiting for unrelated M groups. The aggregate barrier is an
unresolved scheduling limitation, retained conservatively rather than silently
reinterpreted. Z03/Z04 likewise begin only after their exact parent start gates;
their independently eligible caller/bootstrap slices do not complete the aggregate
parent until every acceptance prerequisite and check is satisfied.

## Dependency challenge and critical path

The completion union has 867 direct edges. The phase-aware audit classifies 480 as
load-bearing and 387 as advisory direct-edge duplication; **all are retained**.
Each edge has an exact phase and classification in `lane-plan.json`. An advisory
edge has an alternate accepted predecessor witness, with the dependency-direction
chain recorded. A `start_after` edge can only use another start predecessor as that
witness; an accept-only alternative cannot release drafting. Intermediate witness
steps use predecessor completion prerequisites because the predecessor must itself
be accepted. The reason template and canonical predecessor title identify the
required output/authority for every load-bearing edge. Direct redundancy never
means the predecessor artifact or its acceptance is dispensable. Deleting redundant
edges alone saves no time; finer source/row delivery and fewer integrator gaps do.

Compute paths from the accepted-completion DAG using topological dynamic programming:
`finish[v] = weight[v] + max(finish[p])`, with zero for an empty predecessor set.
Use the union of `start_after` and `accept_after`, and deterministic lexicographically greatest
path tie-breaking. Full unweighted weights are one. Residual weights are zero for
currently verified complete tasks and one (or estimated attention cost) for every
other task. Recompute after any status/edge correction. Blocked elapsed duration is
unknown even when its eventual review work has an estimated cost.

The full longest unweighted chain is 20 vertices / 19 edges:
P00→P01→P07→P08→P10→P11→P12→P27→P18→P19→P22→P26→I04→I06→QB0→QB1base→QB2→M21→Z05→Z06.
The corrected residual unweighted chain is 14 vertices / 13 edges, beginning at P12
and following the same suffix. Equal-length variants exist; the JSON gives the
reproducible selected representative.

The estimated residual attention path is **63 hours**:
P12(4)→P27(4)→P18(3)→P19(5)→P22(3)→P26(4)→I04(2)→I06(4)→QB0(3)→QB1base(3)→QB1(4)→M18(8)→Z05(8)→Z06(8).
P21 runs beside P22 after P19; P28 runs beside P27 after accepted P12. QD1 after
I13 is a load-bearing join for full QB1. P23 after qualified N and R-core is required
before core execution/Q acceptance even when another structural path wins the max.

| Bottleneck attention assumption | Hours | Why it matters |
| --- | ---: | --- |
| P12 / P28 / P27 | 4 each | Repaired admission, base bindings and within-run builds unlock core source |
| P19 / P26 / P23 | 5 / 4 / 5 | Controller integration, independent reference promotion, first runner/S/A qualification |
| P13 / P14 | 4 / 3 after unblock | Qualifying strict host guarantees remain unresolved; unblock delay unknown |
| I06 / I13 / QD1 | 4 / 3 / 3 | Browser and DB accepted capability/observation joins |
| M18 / M29 / M30 / M32 / M33 | 8 / 13 / 14 / 12 / 12 | Complex retained controls or many independently reviewed rows |
| Z05 / Z06 | 8 each | Final full-scope reconciliation and independent review/fixes |

These are hypothetical planning assumptions about review/integration attention,
with no measured timing basis and no claim to estimate agent runtime. Use a
0.5×–2× sensitivity range; the JSON has an explicit cost for every ID. Migration
default is `max(2, 0.75 × original row count)` hours, with named complex-group
adjustments. Costs exclude implementation coding, host/service unblock time,
retained measurements, heavy run duration and findings-induced requalification.
The 63-hour DAG value assumes unlimited disjoint attention and is a lower bound,
not a completion ETA. At this corrected status frontier, summing remaining modeled
attention gives 422.25 hours if one integrator performs every review; resource
serialization, not just the longest DAG chain, controls wall time. Peer preparation
and small coherent evidence reviews reduce integrator congestion without changing
what must be accepted. Missing strict host/service guarantees or deferred retained
measurement legs make final completion time unknown.

## Build/run and shared-path handoffs

Keep at most three source writers plus one integrator. There is one admitted live
run, one live case, one verification worker and one build producer with the existing
phase ordering. Source work and immutable evidence review may run in parallel;
independent heavy jobs may not. Services/cleanup count in the same host envelope.

The integrator batches compatible **already-ready** I binding slices into one bounded
build where permitted. Each retains separate scope controls and R/N/S/A receipt;
one passing combined build is not slice acceptance. A slow unrelated slice never
becomes a batch prerequisite. Qualify only the ready capability scope and preserve
predecessor authority when a delta cannot yet qualify. Never merge R/N/S/A/C roles.

P06→P26 reference tooling, P18→P24 staging, P10/P27 descriptor leases, P11/P22
cleanup/corrections and P17/P22 reduction/authority retain named sequential path
handoffs. P01 hands the descriptor schema to K17 after its base schema commit;
Z04 receives bootstrap paths only after every relevant writer releases them.
Catalogue and mirrors remain integrator-generated; nobody hand-edits generated
`runtime/catalogue.ts`. The integrator coordinates mandatory runtime formatting
and exact-path commits, stages no unrelated changes, activates callers and retires
old host harnesses only through the complete existing gate.

Register task-owned temporary cleanup before allocation. Reuse immutable builds
within a run, retain compact evidence, and clean on success/failure/interruption.
This plan creates no execution workspace, private cache or disposable worktree.
