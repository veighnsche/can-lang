# Implementation: remove per-element async arithmetic adapters

Design: [native map/fold workers](generated-integer-worker-plan.md). Follow this
actual dependency-ordered checklist in one sustained sole Muse run. Runtime
identity/goal/session/viewer/storage/attach/cleanup fields live only in
`.performance/performance-push-20260928/muse-run-10-owner.json`; no copies here.

Handoff covers I1–I5 in the design. Coordinator alone owns all files/progress,
integration and sequential commands; no extra agents needed. All production/test/
harness corrections are Muse-owned; Codex owns research/design/review. Auxiliary
lane and drivers are frozen. Native get_goal then matching reuse/create_goal BEFORE
implementation is mandatory; report_progress follows evidence, complete only ALL-
writer release and reviewable handoff. Unavailable goals are a blocker, not prose.
For a design question first release affected writers and give evidence/task IDs.
No arbitrary step/token budget. Long-command cadence1–5minutes/default2, native bash
initial120000ms, retain handle/deadline and await delivered completion; no empty polls.

Owned compiler lane: NEW integer_workers.go/integer_workers_test.go in
compiler/internal/emit; runtime_bindings.go,program_modules.go,regions.go,callables.go
for proof/companion/import/callable wiring only. No checker/type/syntax changes.
Owned runtime lane: runtime/callable.ts and runtime/collections/array.ts plus NEW
runtime/integer-workers.test.ts. Ordinary browser/owner/completion/failure/data/
reflect implementations untouched. Serial dependency order, shared interface frozen
in G61; no other source paths. Checklist/design clarification only after release.

- [x] **G61 — freeze source-backed proof and integer descriptor**
  - Prerequisites: accepted e57f3ad2, ALL-writer release and saved baseline.
  - Owner/files: coordinator, new proof file + checklist/packet11 evidence.
  - Changes: invoke native goal; inspect exact closed int IR/capture/companion
    imports and runtime guards, freeze optional ownCallable descriptor/query.
    Record eligibility against actual functions, not workload-name whitelist.
  - Acceptance: I1/I2/I4; exact identity/final target/specialization/body/type
    negatives; no emitted-annotation-only purity claim, native callback inspection
    or unplanned general effect analysis. Material conflict is explicit blocker.
  - Evidence: met: audited Region/ProgramFunction/Expression/Completion IR,
    ExpressionEmitter literal/binding/unary/binary lowering, ownCallable/
    registerCallableCaptures/callable emission (Positions/slots), cross-module
    function imports, array map/fold boundaries, caught/Completion branding,
    isHostProxy/array boxing. Frozen: proof identity->{Target,Companion,Source,
    Region} post-merge (int-only I/O, no Errors/Escapes/Requests/Steps, one
    success expr of literal-int/input-binding/unary-minus/+,-,*); companion
    `Target$Int(caps...,args...): bigint` with cold-origin catch throwing
    `caught(e,origin).value`; ownCallable optional 7th descriptor
    {companion,positions,arity,origin} validated via own props (malformed
    throws, non-bigint/ nonempty-resources declines); private WeakMap
    metadata + `integerWorker` query; array.ts fast path requires exact
    worker, undefined context+owner, trap-free frozen dense bigint guards,
    arity 1/2, bigint initial for fold; native map/reduce with typeof
    guards, fresh frozen output, boxed boundaries. No conflicts. Jev 1/1/1
    paired_worker advice only. Record in packet-11 evidence (g61).
- [x] **G62 — emit synchronous companions and proven residual registration**
  - Prerequisites: G61. Owner: coordinator, owned Go files above.
  - Changes: private proof, deterministic companions/imports with original
    expression/source mapping and cold error origin; unchanged async function.
    Exact proof adds optional descriptor to normal owned callable using saved
    captures/positions. Share into relevant emitters read-only; browser unchanged.
  - Acceptance: I1/I2/I3/I4, stored captured and generic-specialized int callbacks
    qualify; stale/rebound/mixed/unsupported/resource targets do not. No per-item
    success/async/origin allocation in valid raw companion. No source duplication.
  - Evidence: met: NEW integer_workers.go (proof type, checkedIntegerWorkers
    closed-int validation, provenIntegerWorker, IntegerCompanion with reused
    lowering/marks + cold `caught().value` origin catch); bindings field +
    post-merge build; regions.go emitter field; program_modules 3-site sharing,
    companion export + cross-module imports (browser omitted); callables.go
    optional 7th descriptor {companion,positions,arity,frozen origin} for exact
    proof + residual arity 1/2 only. Async functions untouched. One narrow
    explained old-test correction (forwarding descriptor expectation); emit
    package green. Record in packet-11 evidence (g62).
- [x] **G63 — implement guarded native map/reduce runtime branch**
  - Prerequisites: G62. Owner: coordinator, callable.ts/collections/array.ts.
  - Changes: minimal private weak worker registration/query and descriptor checks;
    precise nonproxy frozen dense bigint admission; native map/reduce inside
    existing boxed boundaries; original fallback and scalar output refusal.
  - Acceptance: I1–I4, no getters/then assimilation, mutable/sparse/override/foreign
    resources preserve slow path, empty/captured/negative/large results, input
    unchanged/fresh frozen output. No metadata cache of foreign arrays or API modes.
  - Evidence: met: callable.ts optional 7th descriptor, own-data-prop validation
    (malformed throws, non-bigint/resources decline), frozen WeakMap metadata +
    `integerWorker` query, residual worker with callback-origin typeof guard;
    array.ts `integerElements` trap-free admission (proxy/proto/frozen/symbols/
    named-overrides/length/per-slot data-bigint) + native map/reduce fast paths
    inside existing boundaries, original adapters intact. lint/format/check 0;
    array/callable/collections suites green. Record in packet-11 (g63).
- [x] **G62a — restore the specified cold-only worker origin**
  - Prerequisites: released predecessor; G61-G63 are reported checkpoints, not
    independent acceptance. Owner: sole Muse coordinator, integer_workers.go
    and integer_workers_test.go only, with checklist/evidence progress.
  - Changes: replace raw companion `markNode`/`$canOrigin` assignments and eager
    region-origin initialization with mapping-only node/function comments.
    Preserve exact defining node sources/spans and original callback-region
    cold catch origin. Correct the partial test that currently requires eager
    `$canOrigin`; add a meaningful emitted hot-path absence check plus cold
    fault-origin execution. No change to ordinary authored function lowering.
  - Acceptance: no successful per-item origin object/array allocation or success
    carrier/async work, same source mappings and cold failure occurrence/origin.
    Source clarification: ProgramFunction.Requests is specialization provenance,
    not effects; generic instances qualify through the same complete IR proof.
  - Evidence: Codex bounded source review confirmed resetOriginCache disables
    caching, markNode assigns origin object/array per expression, and current
    companion eagerly initializes region origin. Existing I1/G62 requirement
    therefore unmet; review is proportional and requires no new design series.
  - Evidence: met: mapping-only integerWorkerMark, function-token prefix,
    direct static cold origin, emitter stateless; eager-$canOrigin test
    replaced with mappings/absence proof + CAN_BUN cold-fault execution
    (exact origin, fresh occurrences); 8/8 Go tests pass. Packet-11 (g62a).
- [x] **G64 — prove the actual new path and fallbacks**
  - Prerequisites: G63 and G62a. Owner: coordinator, NEW Go/runtime tests only.
  - Changes: real checked emission/assembly proof, exact companions/mappings and
    positive execution of direct/generic/locally stored captured map/fold with
    installed Bun. Meaningful malformed metadata/output/hostile-array/capture
    negatives; original defined frame/fixture scheduling, resource/async/owner/
    cancellation/lease fallbacks, raw caught origin/fresh first failure stop.
  - Acceptance: new route demonstrably used, zero skips; not manual invoke or
    function-entry surrogate. Preserve all standard carrier/origin contracts;
    reused successful unchanged generic tests need not be duplicated.
  - Evidence: met: Go emitted execution (4 endpoints incl. empties) +
    route discrimination (real descriptors/companions, stub slow path silent
    fast / 999n cookie on owner+mutable fallbacks), full emit package green
    zero skips; runtime/integer-workers.test.ts 11 tests/117 expects green
    (fast proof, captures, 16 malformed throws, 3 declines, 10 guard
    fallbacks, hostile refusal, fault first-stop, context/owner/arity
    fallbacks); forwarding correction reviewed test-only. Packet-11 (g64).
- [x] **G65 — required runtime checks and one reused generated graph**
  - Prerequisites: G64. Owner: coordinator, owned files + compact evidence.
  - Changes: gofmt; bounded new/affected Go tests CAN_BUN/GOMAXPROCS2/-p2/shared
    cache; runtime lint:fix:runtime,format:runtime,check:runtime and meaningful
    array/callable/owner/completion tests. ONE reused actual prepared graph with
    existing tools, real runtime link, hardlinked bench/private fd3, strictTS,
    mappings/24 controls/14 fresh bindings and complete real identities.
  - Acceptance: I1–I4;64MiB/500 files, saved bindings/compiled module inventory
    before cleanup, explained changed modules and eligible endpoints. No broad
    unrelated build/browser/auxiliary suite, installs/copies/private caches.
  - Evidence: met: gofmt/vet clean, check:runtime 0, 65+36 suites green;
    validate-only 13/13 (41.9MB/300 files), bench hardlink ino 347056138
    sha==baseline, strict tsc 0 over 11 files, 24/24 oracles stderr 0,
    14 fresh bindings + inventory saved; 4 companions (double/add/
    add_offset/identity) + 4 descriptors explained per module; 4 eligible
    endpoints. Packet-11 (g65).
- [x] **G66 — one unchanged-method after comparison**
  - Prerequisites: G65. Owner: coordinator, same retained qualified graph and
    generated-integer-worker-after.json + comparison/evidence only.
  - Changes: exactly design sampling parameters matching immutable qualified
    before; six fresh sequential drivers, bounded names/CPU observation, no
    competing owned checks/builds. Preserve all raw oracle/control/membership and
    before/after input/tool/driver/module/dependency identities. Retire groups/tree.
  - Acceptance: I5; all24 finite complete cases,42 accepted+12 excluded values
    each; median of six trial medians/range/MAD, four actual changes/native gaps,
    frequency unchanged scope. No resampling/driver/tool edits/p95/heap fabrication.
    Failure receives unavailable disposition and exact cleanup evidence.
  - Evidence: met: 6/6 fresh sequential trials complete, 24 cases each,
    42 accepted + 12 excluded per case, all oracles pass; after-JSON
    659225B with raw trials/validate/bindings/identities/activity;
    doubled 1.89x, fold 1.85x, generic 1.99x, captured 1.65x (medians
    of trial medians, non-overlapping ranges), ratios 4.78->2.57,
    12.37->4.91, 4.57->2.59, 4.99->3.63; frequency unchanged scope;
    graph retired via tool. Packet-11 (g66).
- [x] **G67 — truthful large-gain evidence and ALL-writer handoff**
  - Prerequisites: G66 complete or specific comparison failure disposition.
  - Owner/files: coordinator, checklist and generated-packet-11-muse-evidence.json.
  - Changes: source/commands/results/hashes/proof inventory/raw comparison/limits,
    work removed versus observed gains, native goal lifecycle, cleanup custody.
    Explicitly release ALL writers, complete matching native goal, idle same TUI.
  - Acceptance: I1–I5 evidenced; do not imply a multiplier from site counts or
    correctness, keep frequency/startup/all12 open; no commits/R11/Q11 ticks.
  - Evidence: met: packet-11 evidence (files/hashes/commands/results/
    comparison/limits/cleanup/goal lifecycle); probe/graph/stage all
    retired exact; ALL writers released; native goal completed at
    reviewable handoff; idle same TUI; R11/Q11 untouched. Packet-11 (g67).
- [x] **G63a — repair proven capture and real-array admission gaps**
  - Prerequisites: G67 ALL-writer release; first Codex review and saved probes.
    Sole Muse owns runtime/callable.ts and runtime/collections/array.ts only.
  - Changes: pair compact captures by ordinal with their original parameter
    positions; require matching capture count, bigint captures and position
    bounds against full companion arity. Accept valid positions [1] and [0,2].
    Require real arrays after proxy rejection for sources and metadata positions,
    read metadata length through own data descriptors, and reject malformed
    shape/length before slot traversal without accessor execution.
  - Acceptance: original I2-I4; true non-leading captures register and native
    traversal receives exact argument order; fake source arrays take conservative
    fallback; fake/accessor positions are refused with zero getter calls. No new
    cache, API, effect analysis or compiler production changes.
  - Evidence: independent inline Bun reproduced positions[1]/captures[5n]
    declining (slow1/fast0), frozen non-Array Array.prototype object admitted
    (slow0/fast1), and fake positions length getter called once. Arithmetic and
    473 source identities separately passed; production acceptance remains pending.
  - Correction: met: ordinal capture pairing (count/bigint/bounds, exact
    order), Array.isArray source gate after proxy rejection, real-array +
    own-data length positions validation before slot loop; callable.ts
    cd03a953678e, array.ts 3f4351463d7d; contracts preserved. Packet-11
    corrective (g63a).
- [x] **G64a — cover the repaired paths and remove new runtime source copies**
  - Prerequisites: G63a. Sole Muse owns integer_workers_test.go and
    runtime/integer-workers.test.ts only, plus checklist/evidence.
  - Changes: real checked/emitted non-leading and separated near captures with
    exact positions/order and positive metadata/native-route discrimination;
    meaningful fake-array/fake-positions/accessor-length and mismatch negatives.
    New TestIntegerWorkerEmittedExecution must use a checkout runtime link in
    owned t.TempDir scratch, replacing zero-byte inventory before execution, not
    liveRuntimeArtifacts' full runtime bytes. Shared historical helpers untouched.
  - Acceptance: zero-skip applicable new/repaired tests; bounded runtime
    lint:fix:runtime/format:runtime/check:runtime and affected worker/callable/array
    tests. Reuse accepted strictTS/24 prefix-workload evidence with explicit
    runtime identity delta and unchanged compiler/fixture/module production
    inputs; no broad unrelated graph/tool suite or further sampling.
  - Evidence: met: runtime 16 tests/149 expects ([1]/[0,2] order proof,
    4 mismatch declines, fake-array fallback, zero-getter refusal);
    Go capture fixture (descriptors [1]/[0,2], asymmetric execution);
    execution via path-inventory + checkout runtime link, zero
    liveRuntimeArtifacts refs; lint/format/check 0, 34/34 bun,
    emit package ok. Packet-11 corrective (g64a).
- [x] **G67a — truthful corrected-runtime handoff without resampling**
  - Prerequisites: G64a. Sole Muse owns checklist and new compact corrective
    evidence, preserving original raw, original packet-11 evidence and all hashes.
  - Changes: get_goal then matching create/reuse before repair; progress follows
    evidence. Keep measured before/after records byte-identical, actual producers
    unchanged. Original four prefix-workload measurements remain observations
    of the original after producer; corrected runtime is qualified separately
    with no new timing claim. Include actual commands/times/hashes/custody/limits,
    release ALL writers, complete goal only at reviewable handoff; idle same TUI.
  - Acceptance: no rewriting raw identities or claiming current bytes were timed;
    no extra executor/agents/source copies/private caches; R11/Q11 untouched.
  - Evidence: met: corrective evidence written; 3 originals SHA-verified
    byte-identical before/after; corrected runtime UNMEASURED; no
    sampling/repair/claims; ALL writers released; goal complete at
    reviewable handoff; idle same TUI. Packet-11 corrective (g67a).
- [x] **R11 — independent acceptance and coherent exact-path commit**
  - Prerequisites: G67a/ALL writers released. Codex owns actual diff/new contracts,
    one proportional acceptance, comparison arithmetic/identities and cleanup.
    Reuse successful evidence. Real defects go through saved tasks/SAME Muse.
  - Acceptance: concrete supported production behavior and honest measured gain/
    limitation, exact owned paths committed, foreign docs preserved.
  - Evidence: first arithmetic/input review plus narrow corrective acceptance in
    generated-packet-11-independent-review.json. Independent 34 runtime tests /
    344 assertions, 11 actual top-level emitted-worker tests with zero skips,
    runtime lint/format/type checks and diff check passed. Muse's 12-test count
    is superseded by the actual 11; no checks were repeated to reconcile it.
    The original three evidence files remain byte-identical. All 473 checked
    inputs retain identity except the two corrected runtime production files
    and their new runtime test; compiler production, fixtures and drivers match.
    Original producer observed 1.65–1.99x faster for four endpoints on a busy,
    ordered host; corrected runtime is independently qualified and unmeasured.
    New executable tests link checkout runtime without source copies; graph and
    probe are absent. Exact owned paths are accepted for the coherent commit.
- [ ] **Q11 — next dominant gap and final twelve-slice exhaustion**
  - Prerequisites: R11. Codex owns current frequency14x/source-supported next
    remedies and remaining twelve dispositions; no more speculative micro-series.
  - Acceptance: continue substantial ready fixes; final exhaustion only after all
    candidates have verified fixes/specific dispositions and independent review
    finds no ready supported fix; cleanup after writers AND viewers release.
  - Evidence: pending.

## Resource-blocker release note (2026-09-29, goal paused, no work resumed)
- Codex observed owned native storage 67875475B > 64MiB bound and interrupted
  via Esc with the goal visibly paused. Muse released ALL writers: no live
  command children, no agents, no builds/tests/timing after the interrupt.
- G61-G63 implementation stands as written (proof/descriptor freeze recorded,
  companions + registration, guarded native map/reduce); G64-G67 unfinished,
  unchecked above; R11/Q11 remain Codex-owned. Native goal left paused, not
  complete. Codex owns storage-custody resolution and campaign continuation.
