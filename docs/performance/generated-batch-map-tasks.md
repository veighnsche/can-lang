# Ordered batch-map reduction assignments

Status: ready after R12 accepted `edeb59c8` and explicit ALL-writer release.
Follow [B1–B3 design](generated-batch-map-plan.md), the campaign plan and
AGENTS.md. Mutable native process, session, pane, goal, storage, viewer and
attach/cleanup values belong **only** to the authoritative monitor linked by
`.performance/performance-push-20260928/continuation.json`, never here. One sole
Contributor/MAX/scoped-YOLO Muse coordinator executes all ready G76–G83 tasks in
dependency order in one sustained session. Muse owns all implementation, tests,
checks and measurements; Codex owns R13/Q13 review/closure. No Muse commits.

## Lane 1 — evidence and finite proof

- [x] **G76 — freeze exact batch transition proof and private interfaces.**
  Prerequisite: R12 accepted and source/consultation evidence in B1. Muse owns
  NEW `compiler/internal/emit/map_batch_workers.go` and this checklist's
  progress/evidence entries. Read existing `map_leaf_workers.go`, actual gallery
  IR and final binding resolution; implement a distinct positive proof for the
  complete get→absent-insert / present-replace grammar in B1. Require exact
  concrete function/source/region/checked map specialization, all calls and
  pure positional prepared arguments, scoped int expressions, impossible
  recovery arms, zero captures/resources/body/errors/escapes. Reject every other
  node and standard/forward arm. Freeze exact descriptor and factory-owned
  private batch interface before downstream edits. No benchmark-name matching,
  general effect framework or reduction of the existing leaf proof. Acceptance:
  gallery qualifies only through actual checked IR; meaningful near misses
  decline with no emitted batch descriptor. If impossible, release ALL affected
  writers and report the exact IR obstruction; no runtime edits or weakened
  proof. Evidence: concrete proof counts/identities and short source audit.
  DONE: gallery `count_one` qualifies via checked IR only: target
  `$canFunction0`, companions `$BatchAbsent`/`$BatchPresent`, str,
  `$canCollection0`, 3 calls (get/insert/replace), pure literal/binary values,
  get span 317–347. Corpus str/int/bool qualify; every batch proof holds a
  leaf proof (subset). Declined with no descriptor: remove/wrong-value/
  statement/recovery-standard-arm/call-free/literal-key/division/swapped-ops
  (leaf-only where expected). Checker fixes domain-first arm order. Frozen
  9th-arg `{absent,present,keyKind,origin,factory}` descriptor, exact-method
  `mapBatchCapability` query, factory `run` with one clone/one `own()`,
  decline-before-traversal, first-call-site cold origin. Scratch IR/proof
  probes removed; `go build ./compiler/...` passes; gofmt clean.

## Lane 2 — owned builder, emission and guarded fold

- [x] **G77 — private native batch capability.** Prerequisite G76 interface.
  Muse owns `runtime/collections/map.ts` ONLY. Attach a private exact-method
  capability to each canonical factory, preserving its seven frozen async
  methods and existing get/insert/replace bodies. Before any visits validate
  genuine initial map identity/backing and all stored bigint values; clone its
  private Map once, update with native `has/get/set` under the proven transition,
  and publish with existing `own()` once. Empty input returns initial token.
  Never mutate an old backing Map, expose a mutable alias or omit opaque
  containment. Refuse unknown/wrong factory/malformed data without reading
  getters or entering the loop. Acceptance: old token snapshots/entries and
  resource containment stay correct, one final publication, ordinary methods
  byte-stable where feasible. Record actual contract tests and diff.
  DONE: `mapBatchCapability` (exact get/insert/replace method identity;
  remove/foreign refuse) + frozen per-factory `run` in `map.ts` (+71/-2,
  method bodies untouched). Scratch smoke: `[z:9]+[a b a]` →
  `z:9,a:2,b:1` order-kept, seed entries/snapshot intact, independent
  3-value opaque snapshot, empty returns identical token, foreign/proxy/
  non-bigint decline pre-visit, first throw stops (1 call) as standard
  failure. Permanent contracts land in G80. `lint:fix`/`format:runtime`
  clean.
- [x] **G78 — proof-only generated transition and registration.** Prerequisites
  G76/G77. Muse owns NEW `map_batch_workers.go`, and only the necessary existing
  `compiler/internal/emit/runtime_bindings.go`, `program_modules.go`,
  `regions.go`, `callables.go`, plus `runtime/callable.ts`. Emit a deterministic
  synchronous pure transition with exact defining source/span mappings, no
  eager per-word origin objects, and keep normal authored async/leaf/integer
  code. Extend separate private callable metadata only for exact proof and
  exact factory binding; validate own-data arity/kind/positions with proxy-first
  checks, no getter/then reads. Keep generic invoke and public Can API unchanged.
  Acceptance: correct cross-module imports, proof-positive descriptor only,
  browser/defined/unknown/resource cases excluded, cold first-boundary mapping
  and fresh occurrences. Record affected emitted modules and source hashes.
  DONE: `BatchCompanions` (`$BatchAbsent($w0:key)`, `$BatchPresent($w0,$w1)`,
  mapping-only marks, cold get-site catch, each `export`ed — probe caught a
  missing second export, fixed); 9th-arg
  `{absent,present,keyKind,origin,factory:Receiver.get}` descriptor with
  `,undefined` padding; wired `mapBatches` through assembly/emitter/modules/
  imports; `runtime/callable.ts` registration (5-key shape, lengths 1/2,
  closed + exact-factory or silent decline, bigint-checked transitions).
  Gallery: both companions exported from defining module, imported by the
  consuming module, one descriptor at get span 317–347, leaf descriptor
  intact ahead of it. Rebound targets rejected; browser path untouched.
  Adjacent Go emit + 39 Bun tests pass; `check:runtime` clean. Permanent
  route/mapping/cold-fault contracts land in G80.
- [x] **G79 — native fold dispatch.** Prerequisites G77/G78. Muse owns
  `runtime/collections/array.ts` ONLY. At the *existing* fold boundary, enter
  private batch path only with undefined context and owner, exact authenticated
  descriptor/factory, real nonproxy ordinary frozen dense matching primitive
  array, genuine initial map and valid bigint state. All admission occurs before
  the first element visit; a failed guard selects the unchanged leaf/generic
  fallback. One local builder traverses in order and publishes one normal
  immutable result, boxed before any Promise exposure. Empty input returns the
  original token. Failure stops at first defining call site with original
  carrier/source/region/occurrence and no replay. Acceptance: integer and
  map-leaf paths unchanged for nonqualified callbacks; no per-word Completion,
  Promise, token, full snapshot or origin allocation on qualified success.
  DONE: batch guard ahead of leaf in `fold` (context/owner undefined,
  registered worker, shared `leafElements` kind guard, runner decline falls
  through with no effect). Scratch fold smoke: `[z:9]+[a b a]` via batch
  only (adapter 0, leaf 0, absent 2, present 1); foreign initial declines
  to leaf (1 call); mutable array declines past leaf to adapter (1 call,
  standard failure retained). Integer/leaf/generic paths textually
  unchanged; `check:runtime` + 32 adjacent tests pass.

## Lane 3 — executable contracts and one comparison

- [x] **G80 — focused real-route and refusal contracts.** Prerequisites G77–G79.
  Muse owns NEW `compiler/internal/emit/map_batch_workers_test.go` and
  `runtime/map-batch-workers.test.ts` ONLY; no old shared-test/helper edits.
  Execute actual checked/emitted str/int/bool map reductions with asymmetric
  values, repeat keys, negative/large bigint, empty arrays, original-map
  snapshots and true route discrimination. Cover proof negatives and real
  fallback for standard recovery, captured/effectful callbacks, context/owner/
  fixture, proxy/shaped/sparse/mutable/accessor/symbol arrays, malformed
  descriptors, wrong map identity and non-bigint host-injected values. Verify
  old map/ownership behavior, hostile boxed then/getter opacity and no replay.
  Use bounded cold wrong-host or injected worker fault to pin exact first
  authored call-site mapping and fresh occurrence; label host injection, no
  enormous BigInt allocation. Acceptance: no skips, actual new route entered,
  one native clone/publication versus no snapshots per word, first failure stop,
  unchanged fallback. Existing qualified owner/lease contracts may be reused.
  DONE: Go 13/13 (11 tests + 2 subtests, zero skips, `CAN_BUN` required):
  gallery/corpus proof, 8-shape near-miss declines + leaf subset, binding,
  companion text/cold literal, 1 batch descriptor + dual export + recovery
  exclusion, value mapping spans, raw emitted str/int/bool routes with
  snapshots/identical-empty-token, injected cold fault at get site 317–347
  with fresh occurrences, foreign-initial decline to emitted leaf with zero
  transition calls. Bun 23/23 (140 expects): registration/shape/arity/
  proxy/getter/unknown-factory contracts, str/int/bool routes (asymmetric,
  repeats, negatives, 2^62), one publication + intact seed + independent
  opaque snapshot, identical empty token, foreign/proxy/non-bigint decline
  to leaf, 8 hostile array shapes + context/owner/fixture/captured to
  adapter, thenable opacity (0 traps), first-stop + fresh occurrences +
  no-replay + clean retry. No shared-test edits; gofmt/`check:runtime`
  clean.
- [x] **G81 — required checks and one reused graph.** Prerequisite G80. Muse
  owns integration/check execution and compact evidence, not new tool features.
  Run gofmt and focused affected Go tests with `CAN_BUN`, `GOMAXPROCS=2`, `-p=2`,
  installed Bun/shared Go cache. Run `bun run lint:fix:runtime`,
  `bun run format:runtime`, `bun run check:runtime` and focused relevant runtime
  tests. Use ONE bounded current graph for 13 validation gates, strict TS,
  mappings, 24 actual emitted/native oracles, 14 FRESH saved bindings and full
  current input/module/runtime/driver/tool/resolved-dependency identities.
  Actual checkout runtime symlink replaces inventory placeholder; benchmark
  hardlink/private fd3. Register graph/group/scratch custody **before** creation,
  hard64MiB/500files for compact measurement artifacts. Do not install, copy
  sources/dependencies, keep private caches or repeat broad auxiliary suites.
  Keep this graph in registered live custody only for G82.
  DONE: gofmt clean; `go vet` clean; full emit package green
  (`CAN_BUN` installed Bun, `GOMAXPROCS=2`, `-p=2`, shared cache; one w4
  busy-host ratio flake 1.93 vs [2,10] then green unmodified + in isolation);
  167/167 focused Bun tests (17 files: completion/data/domain/failure/
  primitive/workers/map-leaf/map-batch/array/callable/collections/owner/
  assertion); `lint:fix`/`format`/`check:runtime` clean. ONE graph: frozen
  `runtime.py prepare --suites generated` (one perfemit build 36504498B
  5123877c8a32), runtime copy replaced by checkout symlink, bench hardlinked
  (ino 347056138 = packet-12, sha 53995d55), node symlink, private fd3 via
  frozen runner. 13/13 gates: build/emission/links/hardlink/14 fresh
  bindings/batch engagement (same defining module path as packet-12)/
  descriptor origin 317–347/strictTS 0 errors/24 oracles/node/no-copies/
  bounds 42909011B + 309 files/identities. Graph HELD live for G82.
  CUSTODY (registered before allocation): `/tmp/can-batch-map-held-graph/`
  (one frozen prepare + trial stage), `/tmp/can-batch-map-held-validate.json`
  (24-oracle validate output), `/tmp/can-batch-map-validate.py` (gate runner,
  retired after use), `/tmp/can-batch-map-graph-evidence.json` (gate evidence,
  packed into G83 then retired). Owner: sole G81 coordinator; retire process
  groups then graph/scratch on success/failure/handled interruption, no
  foreign deletion; bounds 64MiB/500 files; held live only for G82.
- [x] **G82 — one qualified same-method after observation.** Prerequisite G81
  and byte-exact applicable producer/fixed-driver identity equality with the
  retained qualified `generated-map-leaf-after.json`. Muse owns NEW
  `generated-batch-map-after.json` and compact comparison only. If baseline
  fails equality, report unavailable; no invented old producer, retry or
  resampling. Otherwise six fresh sequential unchanged-driver processes,
  size100/iterations1, two excluded warmups/seven accepted samples per case,
  all24 finite controls/oracles, 300s sampling cap, bounded process-name/CPU
  activity. No competing owned tests/builds/edits. Compare median of six trial
  medians, range/MAD, frequency absolute cost/speedup/remaining Can/native gap
  and every unaffected control. Label busy ordered host/JIT/timer uncertainty;
  no heap, p95 or isolated-causal proof or arbitrary multiplier threshold.
  Failure means unavailable comparison and safe cleanup, not driver repair or
  partial timing merge. Retire process groups before the graph/scratch.
  CUSTODY (registered before allocation): `/tmp/can-batch-map-trial{1..6}.json`
  (six trial outputs), `/tmp/can-batch-map-activity-{before,after}.json`
  (bounded snapshots), `/tmp/can-batch-map-trials.py` (runner, retired after
  use), `/tmp/can-batch-map-trial-run.json` (run record, retired after
  assembly). Owner: sole G82 coordinator; no competing owned work during sampling;
  retire groups then graph/scratch per G82 order; trial/activity/runner files
  retire at G82 end after assembly; gate evidence + validate output held for
  G83 packing. QUALIFIED: 4 frozen drivers + bench + 9 fixtures + Bun/Go
  byte-exact vs retained after-record; method 6/2/7/1/100/standard/300s.
  DONE: `generated-batch-map-after.json` 236171B (6 embedded trials, 24 cases
  each, 7 accepted + 2 warmups, names == baseline, oracles passed; sampling
  0.804s < 300s cap; identities identical across sampling; case-set verified).
  Median-of-six-trial-medians: frequency.can 171.042us [162.375,205.333] MAD
  8.479 → 34.521us [31.500,36.875] MAD 1.750 (x4.955, non-overlapping
  ranges); frequency.native 12.479us → 16.563us on the same runs (host ran
  hotter: most controls regressed 5–48%, e.g. failure-recovery.can
  108.375→208.354 with after-MAD 19.104 vs 4.500, fold-sum.native
  3.812→7.354; two native controls improved). Remaining Can/native gap
  13.706x → 2.084x within-run. Read as busy ordered-host/JIT/timer
  observation only: no isolated causal attribution, heap, or p95 claim; no
  threshold applied. Preserved checkout carries additive-only catalogue
  entries (zero removals) + http/browser/docs deltas, none touching the
  measured path. No retry/resample/repair; groups verified reaped, then
  graph + 6 trials + activity + runner retired exact; prior bytes untouched.

## Lane 4 — reviewable handoff

- [x] **G83 — compact evidence and ALL-writer release.** Prerequisite G82 or
  specific unavailable disposition. Muse owns this checklist and NEW
  `.performance/performance-push-20260928/generated-packet-13-muse-evidence.json`.
  Save actual commands/exits/times/top-level test counts/zero skips, proof and
  emitted-route results, current and prior raw producer identities, comparison
  arithmetic, full graph/input manifests, all temporary custody/retirement and
  native goal lifecycle. Distinguish source work removed from observed speed and
  uncertainty. Preserve prior raw/evidence files byte-identical. Release ALL
  coordinator/native writers explicitly; only then complete the matching native
  goal and remain idle in the SAME TUI for Codex. No commits; leave R13/Q13.
  DONE: `generated-packet-13-muse-evidence.json` 15259B (per-task evidence,
  bindings manifest, changed-path shas, comparison arithmetic, custody
  created/retired, identities, goal lifecycle, handoff). Source work removed:
  per-word Map copy/token/snapshot/Completion replaced by one clone + native
  has/get/set + one own(); observed speed (busy-host, non-overlapping
  frequency ranges, hotter-host controls) is evidence, not isolated proof.
  Prior bytes preserved; all temp custody retired exact and verified absent;
  no commit; R13/Q13 untouched. ALL writers released — terminal handoff.
- [x] **R13 — Codex independent acceptance.** Prerequisite G83 ALL-writer
  release and native goal complete. Review actual diff/proof/guards/first-failure
  behavior, source mappings and one-publish ownership; run proportionate changed
  checks only where Muse evidence does not suffice. Independently verify raw
  membership, controls/arithmetic/identities and exact scratch cleanup. Genuine
  changed-path defects return as saved tasks to the SAME Muse session after
  writer release; no second general review merely to approve the first.
  ACCEPTED: exact batch proof admits only the closed checked get/insert/replace
  tree with pure int values; runtime admission precedes visits, clones once,
  publishes through the existing ownership boundary once, and preserves cold
  failure/fallback behavior. Independent changed-path `TestMapBatch` Go and
  23 Bun tests passed. Both raw records independently rechecked all 24 cases,
  six trials, 42 accepted and 12 excluded samples per case, median/range/MAD,
  producer stability and cleanup. Fixed drivers, six fixture hashes, current
  five runtime hashes and after source hashes match; all 10 production/test
  paths match Muse evidence. The checklist hash differs from its pre-handoff
  evidence because G83 was ticked afterward; original raw bytes are intact.
  Frequency observed 171.042→34.521us (4.955x) and Can/native gap
  13.706→2.084x; native control 12.479→16.563us and other controls drifted on
  a busy ordered host. This is an observed result, not isolated causality,
  heap or p95 proof. Compact independent evidence is
  `generated-packet-13-independent-review.json`.
- [x] **Q13 — Codex commit and queue continuation.** Prerequisite R13 accepted.
  Commit only coherent owned accepted paths, preserve foreign README/docs work,
  and continue remaining supported generated/startup and all twelve slices.
  Stop campaign only at independently evidenced exhaustion, never at this patch.
  DONE: exact eleven-path batch implementation/test/checklist commit `b770b457`;
  foreign catalogue/browser/HTTP/docs and local poster output left untouched.
  The next bounded source investigation is generated failure recovery, whose
  current observed Can/native gap is large but not yet attributed. It is tracked
  in `generated-failure-recovery-investigation.md`; new implementation needs
  independent proof/design, three fresh equivalent Jev consultations and its
  own ordered checklist. Startup and the independent twelve-slice exhaustion
  review remain open.
