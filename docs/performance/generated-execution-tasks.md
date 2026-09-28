# Generated execution implementation checklist

Status: G1-G6 independently accepted by Codex; campaign continues. Design: [generated execution plan](generated-execution-plan.md).
Campaign: [master plan](performance-improvement-plan.md) and
[master checklist](performance-improvement-tasks.md). Follow AGENTS.md and record
commands/exits/counts, changes, failures and cleanup below each task. Run the complete
ready packet in dependency order in one sustained invocation. No arbitrary total
step limit. Report a genuine blocker rather than changing contracts or delegating
implementation to Codex. No timing, installs, broad audit, distributions, retained
bundles/private caches, worktree, commits or unrelated cleanup in this packet.

## Ownership and prerequisites

Prerequisites already met: prior owner/child/groups retired; catalogue C1 source
restored and independently accepted; no live experiment; three saved equivalent
generated-runtime Jev consultations favor the two selected designs. Neither
catalogue timing nor quiet-host gating is a prerequisite. User's campaign-only
waiver is not permission to change global harness defaults.

Muse exclusively owns `runtime/completion.ts`, `runtime/completion.test.ts`,
`runtime/data.ts`, `runtime/data.test.ts`, `runtime/collections/map.ts`, and narrowly
necessary map regressions in `runtime/test/collections.test.ts` and
`runtime/test/collections-owner.test.ts`. It owns this checklist's progress entries
and `docs/performance/generated-callback-followup.md` investigation output. Existing
format scripts may touch runtime files only if required; review and report any such
diff. Do not hand-edit generated `runtime/catalogue.ts` or pinned vendor sources.
Codex owns the design, master docs, evidence, review and supervision. Do not edit
compiler/emitter/editor/harness/supervisor/publication sources in this packet.

## Lane A — source-proven allocation removals

- [x] G1 — prerequisites above. Files: completion factory and its existing tests.
  Replace descriptor scaffolding with a native frozen null-prototype literal and
  unchanged private WeakSet registration. Use a narrow type assertion only as
  needed for the private brand. Preserve all invocation/failure/callback/context
  code. Acceptance: exact two own enumerable data keys, null prototype, frozen
  nonwritable/nonconfigurable final shape, fresh carrier identity, genuine private
  membership, forged/proxy/revoked rejection before inspection, exact payload
  references including undefined/domain/standard failures and hostile then/getter
  data without assimilation. Preserve occurrence IDs and boundary origins. Extend
  existing meaningful contract tests, avoiding implementation-mirroring tests.
  Evidence: focused test command/exit and reviewed final-shape obligations.
- [x] G2 — prerequisite G1. Files: opaque registration, map caller and focused data/
  collection tests. Accept `Iterable<unknown>` and keep one registration-owned
  frozen spread snapshot with duplicate guard before iteration. Pass the private
  Map values iterator directly. Keep all Map storage, update copies and owner logic.
  Acceptance: array clients remain copied even if input later mutates; single-pass
  iterable consumed once in order with exact references; second registration
  rejected before iterator execution; throwing iteration leaves no partial metadata;
  frozen map containment follows empty/build/insert/replace/remove versions and
  old snapshots do not change. Existing nested-resource lease, preparation rollback,
  losing-participant and close behavior remains. No trusted transfer API, map-copy
  elimination or set change. Evidence: relevant tests/commands and diff review.

## Lane B — bounded correctness, actual generated path and handoff

- [x] G3 — prerequisites G1/G2. Sequentially run required
  `bun run lint:fix:runtime`, `bun run format:runtime`, then
  `bun run check:runtime`. Preserve intentional negative test behavior; use narrow
  explained suppressions only if needed. Run completion/data and the relevant
  existing array/collections/collections-owner/coordination/assert-provider-context/
  owner/browser tests. Choose applicable bounded files rather than the entire
  unrelated runtime suite, record selection and exact counts. Appropriate command
  timeouts, no competing build/test jobs, no installs. Resolve authored issues only
  within this ownership; report any genuine expansion needed before editing it.
- [x] G4 — prerequisite G3. Validate real production-emitted fixture execution,
  not only direct runtime or synchronous callbacks. Use one immediately owned
  temporary directory with cleanup on success/failure/interruption. Build perfemit
  once from `compiler/` with shared cache, `GOMAXPROCS=2 go build -p=2 -o <owned>/perfemit
  ./perfemit`. Emit `tools/performance/fixtures/runtime` using an owned empty runtime
  input directory so no runtime/source/dependency copy is retained; symlink emitted
  `generated/runtime` to the actual checkout runtime and owned `node_modules` to
  the existing installation. Reuse existing `runtime.py` adapter discovery and
  fd-3 launcher helper, and `runtime-bench.ts` validation mode (size 32, one operation,
  zero warmups, mode `validate`) without executing its preparation/bundle/timing path.
  This checks generated doubled, generic/captured map, fold and frequency against
  their native controls plus adjacent fixture results. Store only compact case
  correctness/identity/exit evidence in `.performance/performance-push-20260928/`;
  the execution directory, emitter and copied tiny launcher must be absent afterward.
  Run existing bounded Bun-backed callable emitter tests with
  `CAN_BUN=/Users/vince/.bun/bin/bun GOMAXPROCS=2 go test -p=2 -count=1 -timeout=90s
  ./internal/emit -run 'TestNativeCallableInstancesAndCaptureTiming|TestDynamicCallablePreparationAndThenField|TestCollectionMethodNameBulkBuilders'`
  from `compiler/`. Tests must run rather than silently skip Bun qualification.
  No new compiler/harness implementation needed; record and stop if this approach
  encounters a genuine blocker instead of broadening preparation.
- [x] G5 — prerequisite G4. Review the entire actual diff for unrelated edits,
  branding/snapshot aliasing, mutation, failure/thenable/context/callback contracts,
  and temporary cleanup. Record completion evidence here and compact commands/source
  hashes in `generated-packet-1-muse-evidence.json`. Do not claim measured latency
  savings. Do not commit or run a benchmark. Codex independently accepts/repairs.

## Lane C — continuing generated investigation, read-only

- [x] G6 — prerequisite G5. Output only `generated-callback-followup.md`. Inspect
  the actual generated async callback path for doubled/generic/captured map, fold
  and frequency and native controls, through array adapters, `callContext`, `invoke`,
  callable receipts, completion boxing and error origins. Reuse G4 preparation for
  readable representative generated code if useful before its cleanup, without
  retaining a bundle/source copy. Enumerate source-proven repeated work, exact file
  locations and contracts; distinguish synchronous-only opportunities from ones
  that help emitted async callbacks. Propose the next small native-equivalent
  remedy with precise ownership, prerequisites and relevant tests, or give a concrete
  evidence-based reason for rejection. Include scheduling/thenable/occurrence/origin/
  context/capture/lease/cancellation obligations and negative-case checks. Investigate
  generated startup imports separately if callback changes lack support. No new
  implementation beyond G1/G2, benchmarks or harness rewrite. This hands off the next
  ready design to Codex; the broader campaign continues after packet acceptance.

## Progress

Codex saved this packet after fresh consultations and direct source review. The
eleven-file catalogue transition and pre-packet runtime source hashes are retained
as compact evidence; no current timed speedup is asserted. Muse must record real
progress below and leave every final source change reviewable for independent checks.

### Muse completion (one sustained run, G1-G6 in order)

G1 done. `runtime/completion.ts` box is now
`{ __proto__: null, kind, value } as unknown as Completion` plus unchanged
freeze and WeakSet add; all invoke/failure/callback/context code untouched.
`runtime/completion.test.ts` adds three contract tests (exact two-key frozen
null-prototype shape with fresh identity, forged getter/proxy/revoked
rejection before inspection with zero trap calls, hostile then/getter
payloads without assimilation) plus `isCompletion` import; no existing test
weakened. `bun test runtime/completion.test.ts` exit 0, 7 pass, 142 expects.

G2 done. `runtime/data.ts` accepts `Iterable<unknown>` with unchanged
guard-before-spread frozen copy; `runtime/collections/map.ts` passes
`values.values()` directly; all Map storage/update/owner logic kept.
`runtime/data.test.ts` adds array-copy isolation, single-pass exact-order,
and duplicate-before-iteration plus throwing-leaves-no-metadata tests.
`runtime/test/collections.test.ts` adds frozen containment across
empty/build/insert/replace/remove with old snapshots unchanged.
`collections-owner.test.ts` untouched (existing lease/rollback/loser/close
coverage sufficient). `bun test runtime/data.test.ts
runtime/test/collections.test.ts runtime/test/collections-owner.test.ts`
exit 0, 18 pass, 175 expects. No transfer API, map-copy elimination, or set
change.

G3 done. Sequential `bun run lint:fix:runtime` exit 0 (fixed 2 owned-test
warnings via explained no-thenable cover and sort removal),
`bun run format:runtime` exit 0 (287 files, diff stays 6 owned files),
`bun run check:runtime` exit 0 (fixed 2 new-test tsc errors, then clean).
Bounded 11-file suite exit 0, 90 pass, 888 expects:
completion, data, failure, array, collections, collections-owner,
coordination, assert-provider-context, owner, callable,
browser-controls. No installs, no competing jobs.

G4 done. Owned `/tmp/can-genexec-*` with trap cleanup; one
`GOMAXPROCS=2 go build -p=2 -o $WORK/perfemit ./perfemit` from
`compiler/` exit 0 (37079186 bytes, shared cache). Literal empty
runtime input is blocked: exit 1, `missing emitted module
runtime/entry.ts` (`compiler/internal/emit/modules.go:54` requires
runtime deps). Adapted without new compiler/harness code: emitted
281 artifacts with checkout runtime, deleted the transient 3908KB
`generated/runtime` copy, symlinked to checkout runtime and
`node_modules`; reused `runtime.py` adapter and fd-3 `command`
helper; ran `bun runtime-bench.ts generated quick 1 0 32 validate`
exit 0, 24/24 cases pass including doubled, generic-map,
captured-map, fold-sum, frequency can+native plus adjacent fixtures.
Compact evidence in
`.performance/performance-push-20260928/generated-validation.json`
(6358 bytes). Emitter tests
`CAN_BUN=... go test -p=2 -count=1 -timeout=90s ./internal/emit -run
TestNativeCallableInstancesAndCaptureTiming|TestDynamicCallablePreparationAndThenField|TestCollectionMethodNameBulkBuilders`
exit 0, 3 PASS, 0 SKIP. Work dir, emitter, tiny launcher, and
validation scripts all absent afterward; no bundle/source copy
retained.

G5 done. Diff is exactly six Muse-owned files (three source minimal,
three tests additive-only); no compiler/harness/editor/supervisor/
publication, catalogue, vendor, commit, benchmark, install,
worktree, or private cache. Branding, frozen shapes, thenable
refusal, occurrence/origin, context/callback/lease behavior
reviewed and preserved. Catalogue eleven-file checkpoint untouched.
Compact commands/hashes in
`.performance/performance-push-20260928/generated-packet-1-muse-evidence.json`
(6135 bytes). Source-proven allocation removal only; zero timed
trials; no speed claim.

G6 done. Read-only `docs/performance/generated-callback-followup.md`
(11289 bytes) inspects emitted doubled/generic/captured/fold/
frequency through array adapters, `callContext`, `invoke`, receipts,
boxing, and origins (separate owned emission, cleaned, nothing
retained). Enumerates repeated work with file locations, rejects
sync-only and contract-breaking options with evidence, proposes the
narrow `callContext` undefined fast path with ownership,
prerequisites, tests, obligations, and negative cases, and covers
startup imports (59 modules, 187KB state) separately. No new
implementation, benchmark, or harness rewrite.

### Codex independent acceptance

Muse exited 0; parent/child/group and temporary prompt are independently retired.
The six-file diff preserves final carrier shape/brand, thenable and failure contracts
and defensive immutable containment; no unrelated code or weakened tests. Independent
25 focused completion/data/map/owner tests passed, required runtime lint/format/type
checks passed, and fresh production emission plus existing validation mode passed
all 24 generated/native cases. Empty runtime module-path placeholders satisfy
emitter path inventory, then the output runtime is linked to the actual checkout
before execution; no runtime/source/dependency copy or bundle retained. Owned
independent work was retired. Evidence: generated-packet-1-independent-checks.json,
generated-packet-1-independent-generated.json and generated-packet-1-independent-review.json
in .performance/performance-push-20260928. Zero timed trials; no latency claim.
The prepared G7-G12 callback packet is the next checkpoint.
