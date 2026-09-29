# Closed recovery packet: ordered implementation checklist

Status: ready for the sole Muse coordinator after a matching native goal is
observed. Read the [design](generated-failure-recovery-plan.md),
[source investigation](generated-failure-recovery-investigation.md),
`AGENTS.md`, current
`.performance/performance-push-20260928/continuation.json` and the sole
mutable runtime monitor
`.performance/performance-push-20260928/muse-run-10-owner.json`.
That monitor alone owns live PID/start/argv/socket/session/pane/goal/storage/
viewer/cleanup/attach identities; do not copy them here. The prior packet-13
goal is complete; create a new matching native goal before implementation.
Muse owns all implementation, tests, qualified measurement and this checklist's
G-task progress. Codex owns R14/Q14 review and closure. No Muse commits.

The purpose is a finite native boolean branch **in generated TypeScript** for
an exact checked `checks::require` recovery, while normal Can lowering and
all declined routes remain intact. Never specialize a benchmark name, change
the fixed driver, add a public Can API or broaden to unproved effects. Hard
native session history bound: 64 MiB. Use installed Bun/shared Go cache,
registered temporary custody, one reused graph/build, no copies/private cache,
and cleanup on every exit. No competing source edits/tests/measurements while
sampling. Preserve foreign dirty checkout paths.

## Lane A — freeze positive checked proof

- [x] **G84** — Prerequisite: current sole session identity and actual native
  goal verified. Owner: Muse. Files: new
  `compiler/internal/emit/closed_recovery.go`, minimal wiring in
  `compiler/internal/emit/runtime_bindings.go`. Construct private proof from
  concrete `ProgramFunction`, exact final binding/source and full checked IR.
  Admit only one-int-input/bool-result Bun function with no body steps,
  errors/escapes and the terminal same-region single canonical
  `can.std.checks@1::require` completion match. Verify exact checked method
  contract/error, positional pure comparison and literal reason through the
  checker's ordered `Prepare` locals or direct arguments, no replay or
  effectful/foreign preparation, two exact literal arms, no payload/standard
  consumption. Browser, extra nodes and every unknown decline. If the actual
  fixture cannot qualify under those guards, release all affected writers
  with a compact concrete obstruction for Codex; do not weaken proof or
  proceed to generated fast lowering. Evidence: positive/negative proof
  identities, actual fixture admission and rejected shapes.
  DONE: `ClosedRecoveryProof` (identity/target/source/region/input/threshold/
  reason/predicate) + `checkedClosedRecoveries` wired post-bindings. Gallery
  `is_positive` qualifies via checked IR only: `$canFunction1`, threshold
  `0`, reason captured, predicate span 265–275, exact void(bool,str)/
  failed contract + `$canChecks.require` binding. Positive variant (threshold
  10) qualifies. Declined: unary threshold, binding threshold, `>=`,
  reversed sides, binding reason, swapped arms, wrong result/inputs/types,
  extra statement, `is`-comparison. Comparison admits exact operator `>` +
  checker `strict` mode only. Probes retired; `go build` clean; gofmt clean.

## Lane B — emit native branch and preserve failure semantics

- [x] **G85** — Prerequisite G84. Owner: Muse. File only `runtime/owner.ts`
  if an ambient owner admission adapter is required; reuse its existing
  `AsyncLocalStorage` store query. The adapter is compiler-private, Bun-only
  and side-effect-free. Do not alter lifecycle logic or browser semantics.
  If this file is edited, run required runtime lint:fix/format/check and
  focused owner tests. Evidence: owner-held path falls back; unowned path
  remains eligible; hashes and required exits. If no runtime edit is needed,
  record why with the actual owner guard used.
  DONE: edit required (no existing presence query). Added compiler-private
  `ambientOwnerPresent()` reusing `context.getStore()`; any present store
  (even closed scope) declines — side-effect-free, lifecycle/browser
  untouched. Forced consequential: canonical-owner export count pin 19→20
  in `browser-profile.test.ts` (flagged for R14; production ownership stays
  `owner.ts`-only). `lint:fix`/`format`/`check:runtime` clean. Focused owner
  tests 42/43: the 1 failure (`browserCallable` vs `canonicalCallable` parity
  on packet-11–13 worker exports) proven pre-existing via pristine-tree
  rerun; my count assertion passes. Owner-held/unowned route behavior is
  exercised in G87 execution tests.
- [x] **G86** — Prerequisites G84-G85. Owner: Muse. Files only
  `compiler/internal/emit/regions.go`,
  `compiler/internal/emit/program_modules.go` and, if needed for private
  import wiring, the same new `closed_recovery.go`. Prepend proof-selected
  native `bigint > <literal>` branch to the existing async generated
  TypeScript function. Admit only undefined assertion context, no ambient
  owner, and primitive bigint input. Return authenticated
  `$canSuccess(boolean)` behind the normal Promise. On false call the
  existing private occurrence allocator exactly once, on true zero times.
  Preserve the entire normal async branch for every decline. Emit mapping-only
  exact defining source/span comments on hot nodes; no hot `$canOrigin`
  initialization or origin object/array. Cold faults use original region
  origin and Completion catch. Evidence: emitted route/mappings, no hot
  origin allocations, exact source/module identities.
  DONE: `provenClosedRecovery` (exact region/target pair) + `Function()`
  prepend (Bun, non-lowered only) + `ClosedRecoveryBranch` (context/owner/
  bigint admission, single `>` eval, `$canSuccess`, one
  `$canAllocateOccurrenceID` on false / zero on true, cold region-origin
  catch) + per-module private failure/owner imports. Gallery emits the
  branch with cold origin recovery.can 238–355; unproven `main` byte-identical
  path. Adjacent suites green with `CAN_BUN` (one ad-hoc run without it hit
  the intended G80 fatal, not a regression). Probe retired.

## Lane C — executable contracts and qualification

- [x] **G87** — Prerequisites G84-G86. Owner: Muse. Files: new
  `compiler/internal/emit/closed_recovery_test.go` and, only if required,
  new `runtime/closed-recovery.test.ts`. Use real checked/emitted tests with
  actual checkout runtime linkage, not function-entry string assertions as a
  substitute for execution. Exercise the gallery function and at least two
  independently named eligible functions with distinct integer thresholds
  and reasons; test true/false boxed results, fresh process occurrence-ID
  continuity (false exactly one, true zero), source mapping/first boundary,
  defined assertion fixture/context fallback, active-owner/cancellation
  fallback, wrong-host and proxy input fallback, standard forwarding and
  browser exclusion. Add proof negatives for payload access, standard arm,
  extra/effectful/reordered arguments, wrong binding/error, extra statement,
  wrong type and other forbidden nodes. Preserve existing positive checks.
  Label deliberate host fault injection separately from valid Can values.
  No source copies, enormous allocations or unrelated test edits.
  DONE: 6 Go tests over real checked/emitted modules (eligible trio
  is_positive/is_big/is_huge, thresholds 0/10/100, distinct reasons):
  exact proof per fn; 13 negatives (wrong op, non-literal/folded
  threshold, non-ordered, effectful operand, reordered operands, swapped
  literals, extra statement, second input, int result, str input with
  `>`, payload-dependent arm, value match with standard arm); per-fn
  branch marks/cold origins/mappings with match-const == model failed
  identity; slow body byte-identical to legacy lowering per fn; browser
  emitter excludes the branch and keeps require. Executed routes in a
  fresh bun process per test: native true/false per fn with call/ID
  counters (0/0 and 0/1), legacy parity on honest inputs, real
  assertion-context declines (reason + call-site origin pinned, identity
  unchanged), owned-root decline, wrong-host number decline (Bun coerces
  mixed `>`, labeled host injection), proxy decline with standard
  forwarding through the else arm plus legacy parity. Runtime test
  `runtime/test/closed-recovery-owner.test.ts` (3/3) pins the admission
  query: false outside roots, true across awaits, true under closed
  scopes. Honest dispositions: require-call standard arms and extra call
  arguments are checker-unexpressible (proof gates defensive); wrong
  error collapses to the arm-set gate plus the identity pin;
  cancellation adds no third signal beyond store presence (closed-scope
  case covers it). Real fixture read-only; siblings appended in memory.
- [ ] **G88** — Prerequisite G87. Owner: Muse. Run gofmt/vet and bounded
  focused Go checks with `CAN_BUN`, `GOMAXPROCS=2`, `-p=2`, shared Go cache;
  relevant Bun recovery/completion/owner tests and mandatory runtime
  lint:fix/format/check if G86 or a runtime test changed. Register scratch
  before allocation. Build once for one reused qualified graph: strict TS,
  source mappings, all 24 actual generated/native oracle cases and 14 fresh
  saved bindings with 13 gates; record real input/driver/module/runtime/
  tool/dependency hashes before retiring it. Keep earlier qualified checks
  when exact input identities match; no broad auxiliary reruns. Evidence:
  actual commands/exits/test counts/oracle membership/input identities,
  scratch group and graph cleanup state.
  CUSTODY (registered before allocation): `/tmp/can-failure-recovery-held-graph/`
  (one frozen prepare + trial stage), `/tmp/can-failure-recovery-held-validate.json`
  (24-oracle validate output), `/tmp/can-failure-recovery-validate.py` (gate runner,
  retired after use), `/tmp/can-failure-recovery-graph-evidence.json` (gate evidence,
  packed into G91 then retired). Bounds 64MiB/500 files; held live only for G89.
  CHECKS DONE, GRAPH BLOCKED (unticked): gofmt clean (5 owned Go files);
  `go vet ./compiler/...` clean; full emit package green with
  `CAN_BUN`/`GOMAXPROCS=2`/`-p=2` (exit 0); `lint:fix`/`format`/`check:runtime`
  clean; 136/137 focused Bun tests across 15 files (sole failure is the
  G85-proven pre-existing browser-profile parity case). Frozen
  `runtime.py prepare --suites generated` FAILED at the binding adapter:
  `{"status": "failed", "reason": "emitter did not emit the requested
  workload bindings"}`. Mechanism: the frozen adapter regex
  (`tools/performance/drivers/runtime.py:62`) requires `let $canOrigin`
  on the line immediately after the function signature, but the
  plan-required prepended branch occupies lines 468-474 of the emitted
  `is_positive` (`$canFunction18`; branch + cold origin 238-355 verified
  in the graph module), so 13/14 bindings resolve and `is_positive`
  misses. Driver unmodified (frozen lane intact, verified via git). No
  compliant path exists unilaterally: editing the driver breaks the frozen
  lane and G89 driver identity; moving the branch breaks the reviewed
  plan and G86/G87; hand-completing prepare is unqualified. Failed graph
  scratch retired (40M/288 files, within bounds). Released to Codex for
  plan revision (R14). Evidence packed in
  `generated-packet-14-muse-evidence.json`.

## Lane D — one qualified comparison and handoff

- [ ] **G89** — Prerequisite G88. Owner: Muse. Check exact applicability of
  immutable `generated-batch-map-after.json` as before against the pre-edit
  producer and every relevant fixture/runtime/module/driver/tool/dependency/
  control identity; freeze and record hashes. If any identity cannot be
  qualified, record comparison unavailable and do not sample. If qualified,
  use the SAME G88 graph for ONE unchanged-driver after record
  `generated-failure-recovery-after.json`: six sequential fresh driver
  processes, size100/iterations1, two excluded warmups/seven accepted per
  case, all24 finite complete controls, at most300sec sampling,64MiB/500
  files, no competing owned work. No resampling, driver/tool repair or
  partial merge on failure. Retire child groups, graph and exact scratch
  even on handled error. Evidence: complete identities/membership/timing
  record, cleanup and unavailable reason if applicable.
  BLOCKED (unticked): no G88 graph exists (frozen prepare fails at the
  binding adapter; see G88 note), so no after record could be produced
  and no sampling was performed. Comparison unavailable for this packet.
  Old raw/evidence files byte-identical (`.performance/` and
  `tools/performance/` clean in git status).
- [ ] **G90** — Prerequisite G89. Owner: Muse. Compare all complete named
  cases with median of six trial medians, range and MAD; show recovery
  absolute before/after cost and Can/native ratio plus unaffected controls.
  Separate source work removed from observed difference and busy-host
  uncertainty. Never claim p95, heap, isolated causality or a guaranteed
  multiplier. Preserve all old raw/evidence files byte-identical. Evidence:
  compact arithmetic and original/current producer hashes.
  BLOCKED (unticked): no after record exists (see G89 note); no
  comparison arithmetic performed and none claimed.
- [x] **G91** — Prerequisite G90 or explicit unavailable disposition.
  Owner: Muse. Save compact
  `.performance/performance-push-20260928/generated-packet-14-muse-evidence.json`
  with actual commands/exits/times, checked proof counts, tests, 13 gates,
  24 controls/14 bindings if executed, all source/tool/driver identities,
  measurements or unavailable reason, cleanup and semantic limits. Tick
  only truly evidenced G-tasks. Explicitly release ALL coordinator/native
  writers and child commands, complete the matching native goal only at
  reviewable handoff, and leave the SAME TUI idle for Codex. No commit.
  DONE (via explicit unavailable disposition): evidence file saved (26
  keys: G84-G91 records, checks, identities, custody, goal lifecycle,
  handoff, semantic limits). Ticked only G84-G87 (evidenced) + G91;
  G88-G90 unticked with obstruction notes. ALL writers released (sole
  contributor, no children/worktrees); native goal completed at this
  handoff; same TUI idle for Codex R14/Q14; no commit. Foreign work
  preserved; old raw/evidence byte-identical; all owned scratch retired.

## Codex integration

- [x] **R14** — After G91 and explicit ALL-writer release, independently
  inspect actual source diff, exact proof and emitted route, changed-path
  contracts, producer identities, original/after arithmetic and cleanup.
  Reuse successful Muse checks; run only proportionate independent tests
  justified by a concrete coverage risk. A genuine defect returns through
  a saved corrective task and SAME Muse session, never a parallel editor.
  DONE: [independent review](../../.performance/performance-push-20260928/generated-packet-14-independent-review.json)
  accepted the actual proof/guard/emitted TypeScript diff after the G91a
  ALL-writer and matching native-goal-complete handoff. Focused Go and Bun
  contracts passed; both 24-case/six-trial records, identities and summary
  arithmetic were independently checked, with only the intended recovery
  module differing. The observed recovery cost was 208.354 to 42.1665 us
  (4.94x), native control 20.7915 to 20.896 us, on a busy ordered host.
  The review records two compact handoff wording/count corrections; neither
  changes raw samples, production behavior or acceptance. All explicitly
  registered graph/trial scratch paths checked absent. Q14 remains.
- [x] **Q14** — After R14 acceptance, commit only coherent owned
  source/tests/checklist/plan/closure paths, preserving foreign checkout
  work and byte-identical prior records. Report measured versus unmeasured
  effects honestly. This is the user's final performance packet: close the
  released sole Muse session and exact task-owned resources, then retire the
  matching scheduled heartbeat. Do not start another performance packet.
  DONE: exact recovery source, tests, design and checklist committed without
  staging foreign checkout work; the sole released Muse coordinator exited
  normally, exact registered session/view and owned tmux/prompt directories
  were retired after identity/no-open checks, and the matching 15-minute
  heartbeat was deleted in the app. The rotated bootstrap pathname had a
  mismatched registered inode and was deliberately excluded. Compact raw
  records and independent review remain in `.performance/`; no further
  performance packet is scheduled.

## Codex-approved narrow correction after first G91 handoff

The first G88 graph failed before validation or sampling because the unchanged
`tools/performance/drivers/runtime.py` binding adapter requires the mapped
`let $canOrigin = ...gallery::is_positive` line immediately after the emitted
function signature. The native branch was placed before that line. Codex
verified that the origin expression is backed by a module-private lazy frozen
slot (`regions.go:260-276`); moving the branch after the existing first line
costs one slot lookup/assignment per call and at most one origin object/array
per emitted site, rather than per call. It retains the original fallback
origin and the frozen driver/baseline identities. The linked design above is
revised accordingly. This is a production tradeoff to make the generated
function usable by the existing workload adapter, not authorization to edit
the driver or fabricate a binding. Muse owns all code/test corrections; Codex
owns the saved design, final review and commit.

- [x] **G86a** — Prerequisites: first G91 ALL-writer release and a new matching
  native goal in the SAME sole Muse session. Owner: Muse. File only
  `compiler/internal/emit/regions.go`. Put the existing mapped function
  prefix first and the already-proved native branch immediately after it;
  browser and self-tail output stay unchanged. Keep the normal async body,
  exact source maps, cold catch, occurrence-ID logic and all admission guards.
  The first generated line must still identify the real function for the
  unchanged adapter. Evidence: emitted order, no per-call origin object/array
  allocation after first slot fill, complete changed-file diff.
  DONE: `Function()` now emits `prefix` then `branch` (was `branch` then
  `prefix`); comment updated to the accepted tradeoff. Branch text,
  admission guards, occurrence logic, cold catch, slow body, browser and
  self-tail paths untouched. The prefix is the existing lazy
  `(slot ??= Object.freeze(...))` expression: one cached slot
  lookup/assignment per call, at most one origin object/array per emitted
  site on first fill, none per hot call after. `go build` clean; gofmt
  clean; diff confined to `regions.go`.
- [x] **G87a** — Prerequisite G86a. Owner: Muse. File only
  `compiler/internal/emit/closed_recovery_test.go` if a test must change.
  Adjust only assertions invalidated by the revised emitted order; preserve
  the six executed real checked/emitted contracts and all proof negatives.
  Assert the mapped prefix precedes the branch and the legacy slow body is
  unchanged. Re-run the affected Go tests with installed Bun/shared Go cache.
  DONE: no existing assertion was invalidated (all six passed unchanged:
  proof exact/negatives are emission-order independent; the legacy
  slow-body strip is index-based and still byte-matches). Added per-function
  assertions only: adapter binding line (`let $canOrigin` immediately after
  the signature, per eligible identity), cached-slot `??=` form, and branch
  after prefix. 6/6 green with installed Bun/shared Go cache; gofmt clean;
  diff confined to `closed_recovery_test.go`.
- [x] **G88a** — Prerequisite G87a. Owner: Muse. Reuse the successful G88
  correctness gates where owned input hashes remain exact; run required
  changed-path Go checks and ONE bounded prepare/13-gate strict-TS/24-oracle/
  14-fresh-binding graph with unchanged fixed drivers. Register exact scratch
  before allocation, use checkout runtime linkage, inventory all relevant
  producer/tool/dependency/source identities, and hold the qualified graph
  only for G89a. On failure, retire exact owned scratch and report an
  unavailable comparison; do not edit or retry the driver.
  DONE: reused G88 Bun (136/137, sole pre-existing failure) and runtime
  gates (lint/format/check clean) after verifying all 8 runtime input
  hashes byte-exact; re-ran gofmt (clean), `go vet ./compiler/...` (clean)
  and full emit package green (`CAN_BUN`, `GOMAXPROCS=2`, `-p=2`, 4.831s).
  ONE graph: frozen `runtime.py prepare --suites generated` (one perfemit
  build 36521938B 4899ace9806d), runtime copy verified byte-identical then
  replaced by checkout symlink, bench hardlinked (ino 347056138 =
  packet-13, sha 53995d55), node symlink via frozen prepare. 13/13 gates:
  build/emission/links/hardlink/14 fresh bindings (module hashes
  re-verified)/branch engagement ($canFunction18)/adapter-line + cold
  origin 238-355/strictTS 0 errors/24 oracles/node v24.21.0/no-copies/
  bounds 42938351B + 310 files/identities. Graph HELD live for G89a.
  CUSTODY (registered before allocation): `/tmp/can-failure-recovery-a-held-graph/`
  (one frozen prepare + trial stage), `/tmp/can-failure-recovery-a-held-validate.json`
  (24-oracle validate output), `/tmp/can-failure-recovery-a-validate.py` (gate runner,
  retired after use), `/tmp/can-failure-recovery-a-graph-evidence.json` (gate evidence,
  packed into G91a then retired). Bounds 64MiB/500 files; held live only for G89a.
- [x] **G89a** — Prerequisite G88a. Owner: Muse. Recheck packet-13 after as
  current pre-edit baseline across all required identities/controls. Only if
  qualified, use that SAME graph for ONE six-process, two-warmup/seven-accepted,
  24-case standard size100/iterations1 after record under the existing
  300-second/64MiB/500-file bounds. Freeze identities before and after,
  preserve raw membership, retire child groups and exact graph/scratch on
  every exit. Identity failure means unavailable; no resampling or tool repair.
  CUSTODY (registered before allocation): `/tmp/can-failure-recovery-a-trial1..6.json`
  (six fresh-process trial outputs), `/tmp/can-failure-recovery-a-activity-before.json`
  and `/tmp/can-failure-recovery-a-activity-after.json` (bounded busy-host snapshots),
  `/tmp/can-failure-recovery-a-trials.py` (trial runner, retired after use). Bounds
  64MiB/500 files; all retired after the after record is packed.
  DONE: baseline QUALIFIED — 5/5 drivers, 6/6 fixtures,
  map_batch_workers.go, 5/5 baseline runtime_ts files and tools
  (bun1.4.2/go1.27.1/tsc7.0.2/python3.14.7) byte-exact; 13/14 emitted
  modules byte-identical (same binding/path/sha) with ONLY is_positive's
  module differing (owned treatment); 24 oracle names identical in order.
  Owned treatment deltas recorded: perfemit 4899ace9806d, owner.ts
  (+presence query, behavior-neutral off-branch per G87), emitter files.
  ONE after record `generated-failure-recovery-after.json` (220204B):
  six sequential fresh standard size100/iterations1/warmups2 processes
  via the SAME held graph, 0.632s sampling (cap 300s), all 24 cases ×
  (7 accepted + 2 warmup) finite with correctness passed, identities
  frozen before/after with zero drift, no competing owned work, no retry.
- [x] **G90a** — Prerequisite G89a. Owner: Muse. Recompute the complete
  median-of-six-trial-medians, range and MAD; report absolute recovery Can
  and native costs, ratio, and all unaffected controls. Separate removed
  source work from measured busy-host variation; no isolated causal, heap,
  p95 or guaranteed multiplier claim. If G89a is unavailable, state that
  explicitly without a performance claim.
  DONE: recovery.can 208.354µs [115.500,231.708] MAD19.104 →
  42.166µs [28.833,57.667] MAD12.417 (median of six trial medians; before/
  after ranges do not overlap). recovery.native 20.792µs → 20.896µs
  (stable host control). Can/native ratio 10.021x → 2.018x. All 22 other
  cases within busy-host noise of baseline (largest control median shift
  2.9µs on captured-map.can, within its MAD 3.0). Removed per-call source
  work: require invocation, reason record, domain carrier, match dispatch;
  added: cached prefix slot lookup + admission checks + occurrence
  allocator on false. Busy-host/fixed-order/JIT caveat applies (trial
  medians spread in both records); no isolated-causal, heap, p95 or
  guaranteed-multiplier claim.
- [x] **G91a** — Prerequisite G90a or explicit unavailable disposition.
  Owner: Muse. Write a NEW compact corrective evidence record preserving
  `generated-packet-14-muse-evidence.json` and all prior raw records byte
  identical. Record actual commands/exits, tests, identities, comparison or
  unavailability, cleanup and native goal lifecycle. Complete the matching
  goal only with explicit ALL-writer/child release and leave SAME TUI idle for
  Codex R14/Q14. No commits.
  DONE: `generated-packet-14-corrective-muse-evidence.json` saved (24 keys:
  G86a-G91a records, checks, gates, identities, comparison, custody, goal
  lifecycle, handoff, semantic limits). First-handoff evidence and all prior
  raw records byte-identical (read-only; sole new writes are the after
  record + this evidence). G86a-G91a ticked; G88-G90 keep their obstruction
  notes; R14/Q14 untouched. ALL writers released (sole contributor, no
  children/worktrees); native goal completed at this handoff; same TUI idle
  for Codex; no commit. Foreign work and Codex's plan revision preserved;
  all owned scratch retired (verified absent).
