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

- [ ] **G61 — freeze source-backed proof and integer descriptor**
  - Prerequisites: accepted e57f3ad2, ALL-writer release and saved baseline.
  - Owner/files: coordinator, new proof file + checklist/packet11 evidence.
  - Changes: invoke native goal; inspect exact closed int IR/capture/companion
    imports and runtime guards, freeze optional ownCallable descriptor/query.
    Record eligibility against actual functions, not workload-name whitelist.
  - Acceptance: I1/I2/I4; exact identity/final target/specialization/body/type
    negatives; no emitted-annotation-only purity claim, native callback inspection
    or unplanned general effect analysis. Material conflict is explicit blocker.
  - Evidence: pending.
- [ ] **G62 — emit synchronous companions and proven residual registration**
  - Prerequisites: G61. Owner: coordinator, owned Go files above.
  - Changes: private proof, deterministic companions/imports with original
    expression/source mapping and cold error origin; unchanged async function.
    Exact proof adds optional descriptor to normal owned callable using saved
    captures/positions. Share into relevant emitters read-only; browser unchanged.
  - Acceptance: I1/I2/I3/I4, stored captured and generic-specialized int callbacks
    qualify; stale/rebound/mixed/unsupported/resource targets do not. No per-item
    success/async/origin allocation in valid raw companion. No source duplication.
  - Evidence: pending.
- [ ] **G63 — implement guarded native map/reduce runtime branch**
  - Prerequisites: G62. Owner: coordinator, callable.ts/collections/array.ts.
  - Changes: minimal private weak worker registration/query and descriptor checks;
    precise nonproxy frozen dense bigint admission; native map/reduce inside
    existing boxed boundaries; original fallback and scalar output refusal.
  - Acceptance: I1–I4, no getters/then assimilation, mutable/sparse/override/foreign
    resources preserve slow path, empty/captured/negative/large results, input
    unchanged/fresh frozen output. No metadata cache of foreign arrays or API modes.
  - Evidence: pending.
- [ ] **G64 — prove the actual new path and fallbacks**
  - Prerequisites: G63. Owner: coordinator, NEW Go/runtime tests only.
  - Changes: real checked emission/assembly proof, exact companions/mappings and
    positive execution of direct/generic/locally stored captured map/fold with
    installed Bun. Meaningful malformed metadata/output/hostile-array/capture
    negatives; original defined frame/fixture scheduling, resource/async/owner/
    cancellation/lease fallbacks, raw caught origin/fresh first failure stop.
  - Acceptance: new route demonstrably used, zero skips; not manual invoke or
    function-entry surrogate. Preserve all standard carrier/origin contracts;
    reused successful unchanged generic tests need not be duplicated.
  - Evidence: pending.
- [ ] **G65 — required runtime checks and one reused generated graph**
  - Prerequisites: G64. Owner: coordinator, owned files + compact evidence.
  - Changes: gofmt; bounded new/affected Go tests CAN_BUN/GOMAXPROCS2/-p2/shared
    cache; runtime lint:fix:runtime,format:runtime,check:runtime and meaningful
    array/callable/owner/completion tests. ONE reused actual prepared graph with
    existing tools, real runtime link, hardlinked bench/private fd3, strictTS,
    mappings/24 controls/14 fresh bindings and complete real identities.
  - Acceptance: I1–I4;64MiB/500 files, saved bindings/compiled module inventory
    before cleanup, explained changed modules and eligible endpoints. No broad
    unrelated build/browser/auxiliary suite, installs/copies/private caches.
  - Evidence: pending.
- [ ] **G66 — one unchanged-method after comparison**
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
  - Evidence: pending.
- [ ] **G67 — truthful large-gain evidence and ALL-writer handoff**
  - Prerequisites: G66 complete or specific comparison failure disposition.
  - Owner/files: coordinator, checklist and generated-packet-11-muse-evidence.json.
  - Changes: source/commands/results/hashes/proof inventory/raw comparison/limits,
    work removed versus observed gains, native goal lifecycle, cleanup custody.
    Explicitly release ALL writers, complete matching native goal, idle same TUI.
  - Acceptance: I1–I5 evidenced; do not imply a multiplier from site counts or
    correctness, keep frequency/startup/all12 open; no commits/R11/Q11 ticks.
  - Evidence: pending.
- [ ] **R11 — independent acceptance and coherent exact-path commit**
  - Prerequisites: G67/ALL writers released. Codex owns actual diff/new contracts,
    one proportional acceptance, comparison arithmetic/identities and cleanup.
    Reuse successful evidence. Real defects go through saved tasks/SAME Muse.
  - Acceptance: concrete supported production behavior and honest measured gain/
    limitation, exact owned paths committed, foreign docs preserved.
  - Evidence: pending.
- [ ] **Q11 — next dominant gap and final twelve-slice exhaustion**
  - Prerequisites: R11. Codex owns current frequency14x/source-supported next
    remedies and remaining twelve dispositions; no more speculative micro-series.
  - Acceptance: continue substantial ready fixes; final exhaustion only after all
    candidates have verified fixes/specific dispositions and independent review
    finds no ready supported fix; cleanup after writers AND viewers release.
  - Evidence: pending.
