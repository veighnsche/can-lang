# Bounded feasibility register

Status of **every integrated entry: unrun**. This is future work defined by the [hard-case challenge](../../native-can-test-hard-cases-2026-09-30.md), not authorization to execute it or a claim of passing coverage.


Preparatory execution update: [three mechanics checks supported their narrow claims; browser acceptance failed on unexpected Keychain UI](../../native-can-test-preparatory-results-2026-09-30.md). These observations do not change any integrated status below.

## Shared execution contract

**Separate two evidence scopes before execution.** A mechanics experiment investigates a specific native/interop boundary using exact identified tools/subjects, independent fixed or raw observations and a bounded external cleanup parent. It may use narrowly reviewed temporary host scaffolding; that scaffolding cannot become a retained suite, scenario API or oracle. Such a result informs design only and cannot claim Can-owned replacement coverage or reference/owner qualification. Do not require the complete proposed runner as the instrument needed to discover whether its mechanics work.

An **integrated Can qualification** executes ordinary Can scenarios/expectations and therefore requires accepted R, identified C/native subjects, the nonpublishing entry, qualified external N and applicable host guarantees. C never compiles an authoritative judge. These acceptance requirements remain intact. The entries below often include both a mechanics question and end-to-end Can acceptance; a scoped job must explicitly name which subset it investigates and leave the other checks open.

Record each early mechanics investigation under its own preparatory evidence ID, linked to the relevant B/N/F/D entry. The registered entry keeps its Can-owned acceptance criteria and remains unrun/incomplete until those execute. A successful host-observed mechanics subquestion must never mark the corresponding registered Can experiment complete.

The [prerequisite inventory](../../native-can-test-prerequisites-2026-09-30.md) identifies local Bun inputs and candidate source but no accepted R compiler or qualified generic N. Its [record](../native-can-test-prerequisites-2026-09-30/prerequisites.json) leaves integrated admission false. Four mechanics jobs now have [concrete preparatory scopes](../../native-can-test-preparatory-checks-2026-09-30.md): PM-N1, PM-N2, PM-F1 and PM-B1. Each fixes its question, fixture, observation, control, budget and cleanup owner. All four preparatory jobs ran under their own scope: PM-N1/N2/F1 supported limited mechanics; PM-B1 failed host isolation despite route support. See the current execution summary linked above. The integrated entries below are unchanged by this partial evidence scope. Missing final R/N is not itself a universal prohibition on a bounded mechanics investigation.

Reserve owned storage, processes, descriptors and namespaces **before effects**, including cleanup for start failure, assertion failure, deadline and worker death. The independently identified external owner must remain alive in subordinate-death variants: N for integrated execution, or the reviewed host parent for a mechanics spike. Neither can claim unsupported descendant containment. No shared-service shutdown, foreign table deletion, repository/dependency copy, broad bundle build, browser download, private persistent cache, performance measurement or stress loop. Existing artifacts may be explicitly leased with their qualification status recorded. Build each tiny fixture once within the admitted job; failure is not permission for a broader bootstrap build.

Each row below has a total wall ceiling **including cleanup**; reserve at least 10 seconds for cleanup (15 for a 90-second row). One case is live at a time. Default caps: 4 owned processes, 1 browser context, 8 pending actions, 4 MiB total I/O, 2,000 events, 256 KiB compact retained facts and 32 MiB owned scratch. Browser engine child processes count under the outer host ceiling and need a separately admitted finite process/memory allowance; do not assume 4 processes is enough for a browser. DB connections and browser profiles count separately. Existing shared service storage is not task scratch; task-owned rows/tables/files are bounded and charged. These are proposed admission ceilings, not measured sufficiency. Refuse or explicitly revise an insufficient bound before execution; never drop a variant to fit.

Negative controls are finite declared sublegs with fresh scope/identities; a deliberate wrong expectation must yield the named Can failure, while broken mechanics must yield an honest error/incomplete result. Neither qualifies as a successful positive leg. Never seed a potentially uncontrolled leak: N retains authority to terminate the owned child/connection and prove release. Retain source identities, compact action/effect facts, Can results and matching N receipts; remove scratch on every path. Cleanup failure remains visible and stops further live admission.

The detailed source notes establish why each question exists: [browser](browser-source-challenge.md), [native](native-source-challenge.md), [descriptors](descriptor-source-challenge.md), [database](database-source-challenge.md). Each experiment below must have its code/file list and exact admitted limits recorded before it runs. “Seed” means an explicit reviewed finite variant, not permanently corrupting shared runtime code.

## Browser

### B1 — Can resolve a route while input is pending

**Question:** Can the driver hold an intercepted POST across a Can decision, including one upstream fetch and altered delivery, while continuing to service the initiating input and all terminal callbacks?

**Depends on:** external browser/process ownership, admitted noninteractive credential/host-UI isolation with an independent observation scope, typed `input_begin/settle`, route/event protocol; the durable-effect subleg additionally needs a qualified independent DB observer. No dependency on general synchronous Can callbacks.

**Fixture/scope:** one tiny local page and owned endpoint with distinct native action IDs and an explicit application `operation_id` idempotency key; first hold/release without DB, then one durable mutation and a new intentional request using the same application `operation_id`. At most one paused route. Can sequences actions and chooses corrupt bytes. No full invoice build required for mechanism qualification; the actual invoice/grid facets still need later migration tests.

**Accept:** receive/resolve token before input settles; record exactly one upstream contact for that routed operation, distinct upstream/delivered body identities and complete body capture. Make the corruption route one-shot or unroute it before replay; then require successful uncorrupted replay using the same application `operation_id` and one durable effect after settlement. This new application request has fresh native action/request/route identities. It is not a redispatch of an uncertain native fetch or resolution. The endpoint's independent request trace is needed because idempotency could hide duplicated upstream dispatch in the final row count. Observe actual input settlement independently of delivery and the application witness; route abortion may fulfill a click. Close/seal includes all admitted callbacks plus the separately declared host-effect interval; unexpected UI invalidates isolation even after clean process release. A kill variant aborts delivery, records possible upstream effect and reclaims owned resources without replay.

**Controls:** use an inert simulated host-effect report/missing-observer record to verify isolation failure and unavailable admission without deliberately opening a real native dialog; do not grant this synthetic check credit for real host-observer detection. Reject an unqualified launch recipe before effects. Serialize route service behind input (bounded timeout must not pass); truncate capture but mark it complete (evidence validation fails); attempt a second fetch under the same token (reject); deliberately double dispatch in a controlled adapter variant (request trace fails even if DB deduplicates).

**Bounds:** 90 seconds total for all declared sublegs and cleanup, one browser context/server, two DB connections only for durable subleg, at most three HTTP requests per subleg, response ≤64 KiB. Engine/process allowance admitted explicitly. Close routes/context, terminate/reap endpoint, settle DB work and remove namespace. If sublegs cannot fit, register separate finite variants before execution instead of silently omitting them.

### B2 — Preserve immediate DOM action and event semantics

**Question:** Can a finite typed page-operation binding preserve same-task action ordering, DOM node identity and event phase/order without an authored JS callback?

**Depends on:** typed DOM handles, `dom_batch`, clone/append and event watcher with registration acknowledgment/`defaultPrevented`.

**Fixture/scope:** one local Can control fixture plus static DOM data. Can sends a two-click immediate batch; separately clone/remove/append a target; then physical Enter and explicitly synthetic composing input. The cancellation watcher is a bubble listener on the same node as the app's handler, installed after an explicit app-listener-ready barrier and acknowledged before input. Exercise held response/custom occurrences only after B1. Keep target absent **before** submit distinct from removal **after** request: the first preserves the old allowed no-request branch or request-phase `effect=none`; the latter requires a response-phase uncertain occurrence. Batch supports fixed actions only, no user code or control flow.

**Accept:** an event-realm microtask witness proves the ordering: a reviewed generic observer records each click synchronously and queues a marker with `queueMicrotask`; click 2 must precede click 1's marker. A driver-supplied batch label is insufficient. This fixed native observation contains no scenario oracle or foreign-authored callback. Native observer and Can echo agree on specified event data, `defaultPrevented` is captured by the declared same-node later listener, and clone identity differs from the original while remount has exactly one live target. Request- and response-phase missing-target sublegs have distinct checks/coverage. Preserve physical versus synthetic `isTrusted` facts and seal the declared no-request interval rather than treating a quiet prefix as permanent absence.

**Controls:** insert an explicit event-loop yield between clicks and detect it even if request count remains one; watch before the canceling handler and observe the wrong cancellation fact; return the original handle as the clone; drop an occurrence. Each fails its own check. A naive request-count-only control is insufficient.

**Bounds:** 60 seconds total per separately admitted engine variant, including all its declared sublegs and cleanup (Chromium first; WebKit/Firefox require their own registered variants before full coverage), one context, ≤16 native actions/64 event records per subleg. No engine download. Seal/dispose nodes/context and all owned engine children; shared servers remain alive. Split oversized subleg groups into explicit finite variants before admission.

### B3 — Page-origin Fetch

**Question:** Can a typed Fetch binding execute in the page's real origin/credential/CSP context rather than substituting external HTTP?

**Depends on:** page-realm Fetch, bounded response facts and owned endpoint; use B1 only if interception is part of the selected subleg.

**Fixture/scope:** one page with a generation marker, same-origin cookie and bounded endpoint; Can reads marker and supplies literal method/headers/body/credentials policy. An owned listener on a separately declared local origin/port is deliberately excluded by the page's CSP. Use allowed, generation-rejected and CSP-denied requests; no arbitrary evaluation or controller closure.

**Accept:** independently captured request receives the intended origin/credential/generation facts, Can receives exact status/body or the intended native Fetch failure, and an external request cannot be silently substituted. No retry or redirect beyond the declared policy.

**Controls:** wrong generation; omit cookie credentials; target the owned CSP-forbidden origin that an external HTTP client could contact. Require a complete no-contact interval from its independently armed listener and the browser CSP/network failure, distinct from a server generation rejection. This demonstrates page policy rather than relying only on an HTTP status.

**Bounds:** 45 seconds, one context/server process with at most two owned listening origins, ≤4 request attempts and 64 KiB response each. Close/seal listeners/context and reap; no database is necessary to establish this boundary.

### B4 — Ordinary Can browser codec facts

**Question:** Can a browser-target ordinary Can fixture express the required generic codec shapes and export lossless raw results for an R Can oracle without retaining the TS vector oracle?

**Depends on:** qualified R build path, C browser subject, admitted artifact-load/DOM-ready/data-export protocol. Exact fact transport from N1 is a prerequisite where shared.

**Fixture/scope:** one tiny Can module covers unsafe int64 and `-0`, one variant/nested shape and one deliberate decode rejection; one Bun leg and one selected browser leg use fixed Can-owned expected data. This subset tests expressibility, not all 12 old vectors.

**Accept:** source passes normal Can checking/assertion requirements; identified C artifact exports typed facts after a declared ready event; R compares exact values/error shape and distinguishes browser API absence from a successful result. No call to the old TS vector verdict function.

**Controls:** wrong expected large integer/sign; malformed vector; missing ready/export or unsupported `JSON.rawJSON` yields the appropriate mismatch or blocked capability, never an empty passing vector list.

**Bounds:** 90 seconds total including tiny fixture preparation and cleanup, one context, ≤8 vectors/128 KiB export. Existing qualified artifacts are leased rather than rebuilt; if tiny preparation cannot fit, revise admission explicitly before running. Close/seal, release artifact lease and remove tiny generation.

### B5 — Shared remote Firefox survives case death

**Question:** Can N reclaim a killed worker's remote context/connection without shutting down the shared Firefox service?

**Depends on:** explicit shared-service/owned-context authority and remote engine admission.

**Fixture/scope:** one connection/context, one intentional case-worker kill, external cleanup, then a second context loading a static local/approved endpoint. Do not actually test a forbidden shared-server shutdown.

**Accept:** first context is confirmed closed, second works under the same N-pinned endpoint/instance identity (browser version alone is insufficient), and no case-owned process/connection remains. The shared launch-server handle is inaccessible to case authority. Losing the remote connection without confirmation is unresolved cleanup.

**Control:** submit a request to close the shared server with case authority; it must reject before effect. Use a wholly owned disposable stand-in only if a shutdown fault variant is separately scoped; never kill the user's shared server as a negative control.

**Bounds:** 45 seconds, at most one live context at a time, two sequential contexts and one worker kill. No browser installation or remote service restart.

## Native values

### N1 — Inert, exact native-value transport

**Question:** Can raw hostile native values be constructed and observed through handles without transport itself reading `then`, invoking getters or losing representation/identity?

**Depends on:** identified candidate Bun, closed typed operation catalogue, inert handle protocol and external session owner.

**Fixture/scope:** `-0`, one unsafe JSON integer lexeme, two aliases of one object, getter-backed `then`, throwing thenable and revoked Proxy; fixed immediate native descriptors only. Can supplies expectations and explicit get/await operations.

**Accept:** exact lexeme/bits and alias identity; zero trap/then accesses before the explicit action; precisely recorded access afterward; a complete counter interval; no raw hostile object crosses an async return or report channel.

**Controls:** explicitly assimilate/unbox in a disposable native variant; normalize signed zero; round the large integer; swap one alias. Each named check fails. Merely reporting an opaque ID without same-realm identity observation does not pass.

**Bounds:** 30 seconds, one native child, ≤16 handles/64 actions and 64 KiB facts. Close destroys handles and reaps child even if the Can worker dies.

### N2 — Hostile value reaches the actual C adapter

**Question:** Can a reviewed typed same-realm ingress present an N1 hostile value to C's production adapter and return facts without violating runtime integrity or trusting C as the judge?

**Depends on:** N1, exact R/C artifact identities, explicit current-version linkage/export design and admitted bindings. Design the seam before coding; no private runtime file swap.

**Fixture/scope:** one C-compiled ordinary Can fixture reaches a production codec/collection path with a hostile value constructed in that same isolated realm. Choose one reject-with-zero-traps and one identity-preserving case. Fixed immediate descriptor behavior is enough initially. Record whether any retained source obligation actually requires a richer synchronous Can companion; do not assume one exists.

**Accept:** the independent version-bound invocation witness specified in the capability contract proves the named C adapter/export was reached through the ordinary C-generated subject path, its own local completion is inspected only by the reviewed subject-side binding, zero accesses/identity hold as expected and R receives only inert facts. Exact type/effect signatures are admitted. R-only behavior or raw Bun behavior is insufficient.

**Controls:** bypass the adapter while returning plausible output and require the independent invocation witness to fail; wrong C export/signature rejects; a C adapter fault that eagerly reads the getter causes Can failure; a locally forged completion-shaped value in the subject realm does not authenticate. Altered R runtime import hash is a separate R integrity control, not evidence that C ingress ran. Require the independent C-artifact identity/actual-adapter invocation witness from acceptance even if that integrity control passes. No cross-runtime completion ABI is assumed. If a synchronous authored callback is needed, its immediate-return and checked companion semantics must be separately demonstrated or marked blocked within this entry.

**Bounds:** 90 seconds total including tiny fixture preparation and cleanup, one subject and one observer service, ≤16 values/32 operations; no general dynamic module importer or evaluator. Revise admission before running if preparation cannot fit. Destroy the subject realm and release generation leases.

### N3 — Late result, occurrence identity and lease release

**Question:** Can a selected completion be observed while a losing participant later faults/uses a resource, with accurate identity and a complete final event interval?

**Depends on:** N1/N2 where C adapter is targeted, generic asynchronous gates, external lease ownership and event seal.

**Fixture/scope:** two gated participants, one selected result, one late failure and one captured resource. Can chooses gate release order; native operations expose arrival/settlement/identity facts.

**Accept:** same-realm strict reference comparison and runtime occurrence provenance establish selected occurrence identity; matching serialized IDs alone is insufficient. Preserve late diagnostic order, lease through late use/settlement and subsequent release, and all events at terminal seal. Raw-native and C-adapter legs have separate evidence scopes; an unqualified N2 prevents C-adapter credit. Judge-worker death always leaves incomplete execution even with successful cleanup. An intentionally killed C subject is an observed outcome only if the surviving Can judge checks the declared death facts; it never substitutes for the normal late-event leg.

**Controls:** change occurrence identity, drop late event, close resource before late use; each prevents complete success. Never treat the first successful prefix as terminal evidence.

**Bounds:** 45 seconds, ≤2 participants/4 gates/64 events; explicit cancellation and outer termination after deadline, followed by release receipts.

## Descriptors

### F1 — Credentials, EOF and launch isolation

**Question:** Can the external launch binding deliver independent fd 3 snapshot and fd 0 input, with correct startup-environment policy and no blocked-writer leak?

**Depends on:** external pre-exec descriptor ownership and explicit CLI-subject versus direct-entry launch policies.

**Fixture/scope:** one reusable tiny entry; Unicode/present-empty/missing-name facts, binary stdin, malformed snapshot, one-MiB unused snapshot. A C CLI leg receives synthetic hostile startup variables unchanged; its child must preserve them as data while not executing preload. R/N startup must stay isolated.

**Accept:** exact data/error classifications, EOF delivery before application readiness for the snapshot-reading child, and clean writer/child termination. The non-reader must exit without draining its snapshot; its blocked writer is then closed/unblocked and reaped. Secret snapshot data never enters argv/logs/default files/Bun startup environment, and no private owner descriptor enters C. Observe admitted descriptor metadata without dumping payload secrets. The C Go CLI's intentional synthetic hostile environment remains the explicit exception under test, not R/N or Bun startup state.

**Controls:** wrong fd, withheld EOF with bounded timeout, truncated JSON, and deliberate sanitization of C's hostile input (missing required data must fail). Correct direct-entry sanitization must still pass. Repeated launches cannot share an exhausted snapshot offset.

**Bounds:** 45 seconds, ≤2 simultaneous owned children, ≤2 MiB input/4 MiB output; close both pipe ends and reap writer/child on all outcomes.

### F2 — Generation lease outlives its launcher

**Question:** Does a correctly inherited child descriptor keep an owned generation alive after its subordinate launcher exits, and permit reclamation only after the child releases it?

**Depends on:** F1 descriptor map, a tiny owned generation/read lease and external N authority.

**Fixture/scope:** N owns subordinate launcher and child. Observe child-ready, close/exit launcher, request clean, then stop/reap child and request prune. Use a gate, not a long sleep, to establish readiness.

**Accept:** exact generation remains while child holds fd 4, disappears after confirmed release and permitted prune, and no descendant survives. Test the actual inherited kernel lease, not a marker file.

**Controls:** omit the child lease or close it early in a fresh owned subleg and detect loss of protection. Never kill N or touch a shared generation.

**Bounds:** 30 seconds, ≤2 children, one tiny generation, ≤1 MiB scratch; retain only release/prune facts and cleanup receipt.

## Database

### D1 — Compiled Can RETURNING and raw row fidelity

**Question:** Can normal compiled Can exercise the currently admitted RETURNING path while a separate raw connection preserves exact final row facts?

**Depends on:** owned namespace, pinned raw connection/typed cells, current C fixture build; required PG/SQLite legs selected explicitly.

**Fixture/scope:** `one` INSERT RETURNING yields one row; conflict-do-nothing yields zero; a two-row insert under `one` yields row-count failure; one exact int64/null/TEXT-JSON row. Compiler-only invalid dialect/cardinality controls are separately named, not DB transport results.

**Accept:** exact generated bigint and final rows, intended `row_missing`/`row_count`, structured constraint failure when seeded, and evidence of C source → generated path. Tag text and numbers before serialization; no old TS driver boolean is the oracle.

**Controls:** wrong expected ID/row schema fails; deliberate number narrowing loses exactness and is detected. Raw RETURNING alone cannot satisfy the C leg.

**Bounds:** 90 seconds total per engine including tiny fixture preparation and cleanup, ≤2 connections/4 shapes/8 rows, one owned namespace or SQLite file. Settle/close both paths before namespace deletion; absent selected PG is blocked. Revise admission before running if preparation cannot fit.

### D2 — Fresh-pool callback and connection ownership

**Question:** Does a fresh C pool preserve callback scope and transaction connection identity on the first bounded concurrent burst without warmup?

**Depends on:** D1 observer transport/namespace mechanics and C transaction fixture, native callback/settlement facts.

**Fixture/scope:** one eight-actor burst, unique writer keys, each reads within its own transaction; one declared rollback actor. MySQL same-handle LAST_INSERT_ID is a separate environment variant, not implied by PG success. N reserves the entire peak before start.

**Accept:** every admitted callback enters once, no unexplained `resource_state`, each committed actor reads its own payload/ID, final independent rows/unique IDs match, rollback row absent after settlement. Log actual connection/transaction identity so a coincidentally correct value is insufficient.

**Controls:** swap actor identity, substitute a different connection for LAST_INSERT_ID and detect the identity violation regardless of coincidental ID equality. No retries or stress repetitions.

**Bounds:** 90 seconds per selected dialect, ≤8 actor connections plus 1 observer, ≤8 actors/16 writes/32 queries. Drain native work, close pool/observer, remove owned namespace.

### D3 — Poisoned transaction has an independent rollback witness

**Question:** Does a successful write before a PG poison disappear after native settlement, with a fresh transaction still usable?

**Depends on:** D1 raw observer and pinned transaction settlement facts; this strengthens the old F03 oracle.

**Fixture/scope:** insert unique sentinel, trigger duplicate `23505`, observe followup `25P02`, request current commit decision and await native settlement. A fresh raw connection checks sentinel/original key; one fresh Can transaction inserts replay key.

**Accept:** exact error classifications and terminal facts, sentinel absent only after settled poisoned transaction, original unchanged and replay present once. Do not equate a successful callback/commit leaf with a committed database transaction.

**Control:** omit poison and commit sentinel; raw observer must see it. Unknown native settlement cannot pass either leg.

**Bounds:** 60 seconds, ≤2 connections/3 keys per subleg, one namespace. Fenced writes and complete settlement precede cleanup.

### D4 — Visible SQL deadline versus native settlement

**Question:** Can the system preserve and reconcile a mutation that continues after a Can-visible deadline without claiming cancellation or retrying it?

**Depends on:** externally owned DB operations, qualified dispatch/driver-settlement and server-quiescence observations, owned namespace and bounded server-side statement lifetime. Current SQL budget machinery does not support cancel signals; record native cancellation as unsupported and test continued work deliberately. Missing external containment/reconciliation or bounded server lifetime blocks this probe before effects.

**Fixture/scope:** one uniquely keyed transaction/write with a bounded server-side delay inside the actor statement, using only actor and raw-observer connections; no third lock-holder connection. Observe dispatch, expire visible budget, then await driver settlement and server acknowledgment/quiescence under a larger bounded lifetime. Can compares fresh raw reads before and after confirmed server settlement while further mutation is fenced.

**Accept:** started transaction reports stable `commit_unknown`, at most one dispatch, no lease release on visible deadline, final effect interpreted only after known server settlement/quiescence. Driver promise resolution/rejection or connection closure alone is insufficient after ambiguous transport failure; require server acknowledgment or an independently qualified fence/reconciliation. A pool execute-budget subleg retains the same distinction. If server state remains unknown, report indeterminate/unresolved cleanup; never delete the live fixture and call it a pass.

**Control:** prestart-expired budget dispatches nothing and yields `transaction_failed("budget")`; incorrectly declaring early row absence final is rejected by missing settlement evidence.

**Bounds:** 60 seconds, ≤2 connections/one write per subleg, statement lifetime ≤10 seconds and at least 15 seconds cleanup reserve. Can timeout is shorter than the controlled native delay; avoid a benchmark. Establish server-side quiescence/reconciliation before dropping the owned namespace; a terminated client session is not automatically sufficient.

## Planning exit gate

For every attempted entry, save its exact question, selected variants, code/artifact identities, positive and seeded-control results, remaining unknowns and resource receipt. A successful subset can guide implementation planning but cannot be relabeled complete feasibility of unrun variants. If an experiment requires changing full Can semantics, crossing R/C authority, retaining foreign scenario code, weakening an oracle or exceeding admitted resources, stop that experiment and return a concrete design decision before expanding it.
