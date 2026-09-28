# Domain catalogue index production tasks

Design: [generated-domain-index-plan.md](generated-domain-index-plan.md).
Master: [performance-improvement-tasks.md](performance-improvement-tasks.md).
Runtime/goal/ownership/viewing/cleanup: [authoritative Codex monitor](../../.performance/performance-push-20260928/muse-run-10-owner.json).
All prior Muse writers released. One SAME Contributor/MAX/YOLO coordinator,
sequential small packet, no new agents/executor/worktree. Native get_goal then
matching create_goal if previous complete; report_progress follows task evidence,
complete only at ALL-writer handoff. No arbitrary token/step/duration cap.
Long commands: initial native bash yield120000ms; routine reinspection1-5minutes,
default2, preserve handle/deadline and await delivered completion, no busy polls.

- [x] **G31 — capture unchanged production baseline**
  - Prerequisite: read design/consultations, current native corrective goal active.
  - Owner: coordinator, compact evidence only; runtime files unchanged yet.
  - Action: one existing unchanged startup-attribution.py explicit --measure to
    `generated-domain-index-baseline.json`; retain source/driver/runtime identities,
    original six-trial/two-warmup/seven-batch/four-profile controls and busy-host
    activity; sequential bounded load, immediate group/scratch retirement.
  - Acceptance: complete finite rows/oracles/identities and honest cleanup, or
    specific measurement-unavailable disposition. No rerun/harness repair.
  - Evidence: met 2026-09-28T19:27:34Z-19:27:45Z exit 0, status complete,
    216/216 accepted (48 warmup + 168 accepted, 0 rejections), exact 7+2
    cells, frozen == after, counterbalanced diagnostic last, 639/639
    processes, scratch retired (42.5 MiB/298 files), no strays. Raw
    `7cf4f8eb` (1059611 B). Tool `f8f8bc51` used unchanged; runtime
    untouched (clean tree before run). Import 16.846 modules / 12.144
    bundle, init 2.926 / 2.781; $canDomain 0.898. Compact record in
    `generated-packet-6-muse-evidence.json` (g31).

- [x] **G32 — share only immutable catalogue lookup indexes**
  - Prerequisite: G31 disposition, design contracts frozen.
  - Owner/files: coordinator, runtime/domain-core.ts only for production.
  - Action: first-needed private error index and atomic paired type-shape indexes;
    preserve first-match errors/last-entry shapes, cache retry on failed build,
    current validation placement and all per-instance snapshots/digests/state.
  - Acceptance: no plan memoization, public cache, generated table edits or native
    algorithm reimplementation; all existing domain failure contracts preserved.
  - Evidence: met 2026-09-28T19:30Z: `runtime/domain-core.ts` (`ad4a4704`,
    +39/-6 with checklist) adds module-private lazy error index
    (first-match) and atomically published paired shape indexes
    (last-entry) at the existing points; invalid declarations never build
    the error index; per-instance maps/snapshots/validation/digests
    untouched; existing 5 domain tests pass (59 expects). Avoids per
    runtime one 110-scan per declaration plus two 215-entry Map rebuilds
    (browser verify+create paid 2x); retains exactly 110 + 2x215 entries.
    Compact record in `generated-packet-6-muse-evidence.json` (g32).

- [x] **G33 — protect repeated-factory semantics**
  - Prerequisite: G32 implementation and frozen private interfaces.
  - Owner/files: coordinator, runtime/domain.test.ts; browser test additions in
    runtime/test/browser-profile.test.ts only if necessary for a missing contract.
  - Action: narrow meaningful warm-cache valid/invalid and digest precedence,
    input snapshot/instance separation and occurrence freshness regressions;
    reuse existing browser verification and initialization-phase tests.
  - Acceptance: behavior oracles rather than tests mirroring cache internals.
  - Evidence: met 2026-09-28T19:32Z: 4 new behavior tests in
    `runtime/domain.test.ts` (snapshot/instance separation + fresh
    occurrences, warm rejection precedence, per-factory digest-set
    equality, post-rejection factory health); 9/9 pass, 76 expects; no
    cache-internals counting. No new browser test — existing contracts
    reused, none missing. Compact record in
    `generated-packet-6-muse-evidence.json` (g33).

- [x] **G34 — qualify runtime and actual generated execution**
  - Prerequisites: G32/G33 complete, no other writers/jobs.
  - Owner: coordinator, sequential integration/checks and compact evidence.
  - Action: required bun run lint:fix:runtime, format:runtime, check:runtime and
    applicable domain/browser/entry/assert-identity tests; preserve foreign work.
    One reused bounded validation graph, actual runtime link, strict TS/mappings
    and 24 actual emitted/native oracles with recorded source/module/driver
    identities. No full unrelated broad suite or auxiliary224 suite repetition.
  - Acceptance: required gates pass; cleanup verified on success/failure/interruption.
  - Evidence: met 2026-09-28T19:35:15Z: lint/format/check all exit 0 (only
    3 owned files changed); domain 9/9, failure pass, assert-identity pass,
    entry 6/6, browser 6/7 with 1 failure proven pre-existing on pristine
    tree (stale owner-export inventory omitting `bindNativeCallback`, outside file
    scope, preserved); staged browser domain execution with G32 code
    passes. One validate-only graph complete 13/13, 14/14 bindings
    resolve, strict tsc 0, 24/24 oracles stderr 0 against symlinked
    actual runtime (`ad4a4704`) + hardlinked bench (zero copy); retired
    True, parent absent, no strays. Compact record in
    `generated-packet-6-muse-evidence.json` (g34).

- [x] **G35 — compare one qualified after observation**
  - Prerequisite: G34 gates pass and no competing task-owned jobs.
  - Owner: coordinator, existing tool unchanged and compact records/report.
  - Action: one identical-method after run to generated-domain-index-after.json,
    then generated-domain-index-comparison.md describing actual medians/range/MAD,
    controls/identities and expected source deltas. Use existing report arithmetic.
    No measurement-device implementation or repeat sampling if unavailable.
  - Acceptance: distinguish source-proven work removal, first-startup observations,
    unmeasured repeated-factory benefit and busy-host/version-order limitations;
    no p95/isolated causation/fabricated gain. Retire all execution scratch.
  - Evidence: met 2026-09-28T19:35:47Z-19:35:57Z exit 0, status complete,
    216/216 accepted, exact membership, frozen == after, retired scratch,
    no strays. After raw `993e8424` (1059505 B); bundle `27ebf9cf` ->
    `5e8ca435` (edited runtime embedded), state identical. Init medians
    lower with disjoint ranges (2.742/2.620/2.833 vs 2.926/2.781/3.017),
    $canDomain 0.749 vs 0.898; imports/control overlap. Sequential
    busy-host versions: observation only, no isolated causal claim;
    repeated-factory gain unmeasured. Comparison
    `generated-domain-index-comparison.md` (`b0f96384`). Compact record
    in `generated-packet-6-muse-evidence.json` (g35).

- [x] **G36 — production checkpoint handoff**
  - Prerequisite: G31-G35 complete or concrete safe dispositions.
  - Owner: coordinator, this checklist and compact generated-packet-6-muse-evidence.json.
  - Action: exact changed paths/hashes/commands/real timestamps, production benefit
    and measurement uncertainty, native goal lifecycle and explicit ALL-writer
    release; remain idle for independent review. No code commit or exhaustion claim.
  - Evidence: met 2026-09-28T19:37Z: changed `runtime/domain-core.ts`
    (`ad4a4704`), `runtime/domain.test.ts` (`b3f7704b`), this checklist,
    comparison (`b0f96384`); records `7cf4f8eb`/`993e8424`; tool
    `f8f8bc51` unchanged, no reruns; disjoint init observations with
    stated sequential-host limits; 1 pre-existing stale browser
    failure preserved. Terminal handoff below. ALL writers released;
    TUI idle. Native goal completion is recorded in the runtime monitor.

## Muse terminal handoff (G31-G36 production checkpoint, 2026-09-28T19:37Z)

Sole Contributor/MAX/YOLO coordinator, same TUI, no extra executor.
Shipped the production domain catalogue index: baseline `7cf4f8eb`
(tool unchanged, runtime untouched), G32 private lazy error + paired
shape indexes in `runtime/domain-core.ts` only (`ad4a4704`), 4 behavior
regressions (`b3f7704b`, 9/9 domain tests), G34 gates (lint/format/check
0; domain/entry/assert-identity green; browser 6/7 with 1 pre-existing
stale expectation proven on pristine tree and preserved; one validation
graph 13/13 with strict tsc 0 and 24/24 oracles against the linked
checkout runtime, retired verified absent), after record `993e8424`,
and comparison `b0f96384` (init medians lower with disjoint ranges,
imports/control overlap; sequential busy-host observation, no isolated
causal claim; repeated-factory gain unmeasured). Source-proven removal:
per-factory 110-scan per declaration plus two 215-entry Map rebuilds
(browser paid 2x); retained 110 + 2x215 entries. No tool edits, reruns,
commits, or gain/exhaustion claims. ALL file writers released; no
groups/keeper/scratch retained. R6/Q6 and the remaining queue belong to
Codex. Native goal completion is recorded in the authoritative runtime monitor.

- [x] **R6 — independent acceptance and coherent commit**
  - Owner: Codex after ALL-writer release; inspect diff, independently verify
    applicable new contracts/actual generated identities and comparison arithmetic,
    reuse successful checks appropriately, commit only exact independently accepted
    paths promptly. No auxiliary-lane expansion.
- [ ] **Q6 — continue supported production queue and exhaustion review**
  - Owner: Codex; authored invoke/numeric JSON/startup/all twelve dispositions,
    independent final review required before campaign completion. No fabricated work.

## Independent acceptance (Codex, 2026-09-28)

Actual runtime diff preserves first-match error and last-entry shape semantics,
lookup/validation order, per-plan digest checks, snapshots and fresh failures.
Independently passed 31 applicable contracts/254 expects, with one separately
dispositioned pre-existing browser owner-export inventory failure: its source and
both owner implementations match HEAD and omit an existing export in the expectation.
Required runtime checks and strict TS passed. One actual generated verification
passed all24 emitted/native oracles with14 resolved bindings and exact emitted
identities matching the after observation. Compiler/origin sources and emitted
modules remain unchanged, reusing accepted mapping contracts. A verifier symlink
setup error was corrected with the established hardlinked driver; both owned
graphs/groups retired and cleanup verified. No source/dependency copies retained.
Both216-row comparisons, exact warmup/batch membership, all69 statistical summaries
each and actual runtime/fixture/driver hashes independently match. Only domain-core
and its test changed in the runtime input inventory; emitted modules and producer
drivers are identical between versions. Observed initialization decrease remains
busy-host/sequential evidence, with repeated-factory benefit unmeasured.
Compact evidence: generated-packet-6-independent-review.json and
generated-packet-6-independent-generated.json. Auxiliary lane remains frozen.
