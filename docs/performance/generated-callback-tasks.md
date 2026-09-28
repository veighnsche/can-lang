# Generated callback follow-on checklist

Status: ready; first generated packet independently accepted and owner retired. Design: [callback plan](generated-callback-plan.md). Campaign:
[master checklist](performance-improvement-tasks.md). Muse alone implements source,
tests and corrections; Codex provides design/review/independent verification.
Execute all ready tasks below in dependency order in one sustained max-effort run.

## Ownership and prerequisites

Prerequisites: G1-G6 independently reviewed and accepted; all owned parent/child/
group and temporary prompt retired; actual G6 notes reconciled with this design;
three saved fresh equivalent callback consultations reviewed; no live measurement
or competing implementation. Catalogue timing and quiet waiting do not block this
packet. This is a generated async-path remedy, not a synchronous-only optimization.

Muse owns only `runtime/collections/array.ts`, `runtime/collections/map.ts`,
`runtime/collections/set.ts`, applicable map/set regressions in
`runtime/test/collections.test.ts` and relevant regressions in
`runtime/test/array.test.ts`, narrowly necessary existing context/owner/coordination
test files if a new case cannot fit that file, this task file's progress entries,
and `docs/performance/generated-next-investigation.md` read-only output. Do not edit
generic context/invoke/completion/owner implementations, emitter/compiler, catalogue,
harness, supervisor/publication, vendor or unrelated runtime sources. The first
packet's completion/map changes remain intact. Codex owns plan/master docs/evidence.

## Lane A — private generated-path simplification

- [x] G7 — prerequisites above. In private array callback, bypass callContext and
  callableInstance only when trace.context is undefined; return the existing invoke
  promise directly and forward identical owner/context positions. With a context,
  retain receipt identity, invocation lineage and frame handling unchanged. Retain
  invoke authentication/capture/origins and boxed data at each awaited boundary.
  Acceptance: immediate and truly delayed async callbacks both run serially in
  element order, no extra visit after failure, empty results and short-circuit
  rules unchanged; domain/standard occurrence and original/boundary origins preserved;
  hostile payload then/getters untouched, raw/forged results refused; explicit
  browser owner forwarding and real fixture/barrier behavior still pass. Use
  controllable gates/event order, not fixed microtask-delay assumptions. Record
  the reduced source boundary without claiming measured speed or exact hop counts.
- [x] G8 — prerequisite G7. Encode private successful-carrier provenance in narrow
  local types for observations, sort decorations and fold seed/accumulators. Replace
  repeated public unboxing with direct .value extraction through the same native
  algorithms. Keep every invoke/ok/failure guard and all public value/check branding.
  A local type assertion must cite the constructor/guard invariant; no arbitrary
  unchecked externally supplied carrier. Acceptance: identical map/filter/fold/
  sort/for_each/search behavior, immutable identity-preserving then/getter-bearing
  payloads, exact failure stopping/occurrence, empty fold seed, stable finite-key
  sorting and required native algorithm delegation. No custom traversal engine.

- [x] G9 — prerequisite G8. In map/set private backing helpers, reuse one metadata
  WeakMap lookup after the same non-null object and concrete identity validation.
  Keep public isMap/isSet, key checks, original resource-state origins, all native
  copies and containment/lease behavior. No yield/user code occurs between the
  previous lookups; do not broaden admitted values or expose metadata. Acceptance:
  matching instances work, wrong identities/instantiations and unknown/null/primitive/
  function/proxy/revoked inputs fail as before without traps; map/set old snapshots,
  ordering and keys remain. Use existing contract coverage plus meaningful missing
  negative cases, not a test that mirrors lookup implementation. This is ordinary
  source-proven local reuse rather than another scheduling/ownership design.

## Lane B — verification and reviewable handoff

- [x] G10 — prerequisites G7/G8/G9. Required sequential runtime lint-fix, format and
  check scripts; relevant array/owner/coordination/collections-owner/context/browser
  tests including G7/G8 regressions. Do not repeat unrelated broad tests. Validate
  real emitted doubled/generic/captured maps, fold and frequency/native controls
  through bounded owned preparation and existing validation mode; one perfemit
  build with shared cache/GOMAXPROCS=2/-p=2. Use zero-byte runtime path-inventory
  files (same checkout .ts relative paths) as emission dependencies; remove the
  emitted empty runtime directory and link the actual checkout runtime before any
  execution. This independent G4 technique passed 24 cases without a source/runtime/
  dependency copy or retained bundle. Do not execute placeholders. Existing Bun-backed callable qualification runs rather
  than skipping. Store compact correctness, commands/exits/counts, generated/module
  identity hashes and source hashes in `generated-packet-2-muse-evidence.json`.
  All execution scratch must be absent after success/failure/interruption.
- [x] G11 — prerequisite G10. Review actual diff for ownership, private provenance,
  unchanged assertion path/native algorithms, thenables, failure identity/origin,
  context/owner forwarding, leases/drain and cleanup. Record evidence and blockers
  here; do not commit, benchmark, install or expand implementation scope. Codex
  independently verifies before acceptance and sends any corrective tasks to Muse.

## Lane C — continue supported generated investigation

- [x] G12 — prerequisite G11. Read-only output `generated-next-investigation.md`:
  inspect remaining repeated allocations/boundaries in real generated callbacks and
  native controls, including origin objects and callable wrapper promise adoption.
  Separate static origin spans from dynamic tail-loop steps, substituted-definition
  source IDs and source-map marks. Explain failure/thenable/scheduling/fixture/owner
  obligations before any proposal. If generated startup remains supported, map
  actual unconditional imports/initializers and propose bounded loading-versus-
  initialization attribution. Give precise next ownership/tasks/checks or specific
  rejection/defer reasons. No speculative edits, harness redesign or measurements.
  Continue campaign through Codex's next reviewed design rather than declaring it
  complete after this packet.

## Progress

Codex prepared this follow-on design while the first executor owned its separate
files. All listed review/liveness prerequisites now pass; G6 scope reconciliation is
recorded in the design. No timings are needed before these source-proven removals.

### Muse completion (one sustained run, G7-G12 in order)

G7 done. `runtime/collections/array.ts` private `callback` bypasses `callContext`
and `callableInstance` only when `trace.context` is undefined, returning the
existing `invoke` promise directly with identical owner/context positions and
origin. Defined-context path keeps receipt identity, lineage and frame handling.
`runtime/test/array.test.ts` adds five regressions: sync serial/failure/empty/
short-circuit, delayed fold gates with peak 1, domain/standard occurrence and
`can:`-synthetic boundary origins, raw/forged refusal without then assimilation,
and browser owner ordering with undefined context. `bun test
runtime/test/array.test.ts` exit 0, 15 pass (pre-G8 count 168 expects).

G8 done. Narrow local `PrivateSuccess<T>` type encodes invoke-authenticated ok
guard, `success()`-constructed sort decorations and fold seed/accumulator
provenance with constructor/guard citations. `map`/`filter`/fold-accumulator/
sort-decoration public `value()` calls become direct `.value` extraction through
the same native `map`/`filter`/`reduce`/`toSorted`; every invoke/ok/failure guard
and public validation remains, and the unused `value` import is removed. One new
array test proves then/getter payload identity, frozen results, empty-seed
identity and zero assimilation. Native delegation still 3/1/2/1. `bun test
runtime/test/array.test.ts` exit 0, 16 pass, 177 expects.

G9 done. `runtime/collections/map.ts` and `runtime/collections/set.ts` private
`backing` helpers reuse one WeakMap lookup after the same non-null object and
concrete-identity guard; public `isMap`/`isSet`, key checks, origins, native
copies, ordering, snapshots and leases unchanged. One new collections test covers
null/undefined/primitive/function inputs across get/insert/replace/remove/entries
and contains/add, unknown objects, trap-counted proxies (0 traps), revoked
proxies, cross-specialization union/intersection/difference, and matching-instance
recovery. `bun test runtime/test/collections.test.ts
runtime/test/collections-owner.test.ts` exit 0, 14 pass, 279 expects.

G10 done. Sequential `bun run lint:fix:runtime` exit 0 (clean), `bun run
format:runtime` exit 0 (287 files, owned diff stays 5 runtime files), `bun run
check:runtime` exit 0. Bounded 9-file suite exit 0, 86 pass, 878 expects: array,
collections, collections-owner, coordination, assert-provider-context,
assert-execution, owner, browser-controls, callable. One `GOMAXPROCS=2 go build
-p=2` perfemit build exit 0; emission with 271 zero-byte `.ts` inventory files
(0 bytes) produced 281 artifacts exit 0; emitted empty runtime tree removed and
replaced with checkout runtime symlink plus node_modules link before execution;
existing `runtime.py` adapter and fd-3 helper ran `bun runtime-bench.ts generated
quick 1 0 32 validate` exit 0, 24/24 cases pass with 10/10 required
doubled/generic/captured/fold/frequency can+native present. No placeholders
executed. Bun-backed emitter tests exit 0, 3 PASS, 0 SKIP. Compact evidence in
`.performance/performance-push-20260928/generated-packet-2-muse-evidence.json`.
Owned `/tmp/can-genpkt2-odrDiJ` absent via trap and owned validation script
removed and verified absent; no bundle/source/dependency copy retained.

G11 done. Owned diff is exactly five Muse files (three source minimal, two tests
additive-only); `git diff --check` exit 0. Private provenance, unchanged
assertion path, native algorithms, thenable refusal, failure identity/origins,
context/owner forwarding, leases/drain and cleanup reviewed and preserved. No
commit, benchmark, install or scope expansion. Working tree also contains a
concurrent Codex-owned `performance-improvement-plan.md` checkpoint edit and
unrelated `docs/archive`/`docs/implementation` deletions outside Muse ownership;
both left untouched for Codex/user handling. No genuine blocker.

G12 done. Read-only `docs/performance/generated-next-investigation.md` inspects
remaining per-element boundaries, origin static spans versus tail-loop steps,
substituted-definition source IDs and source-map marks, callable wrapper promise
adoption, frequency map path and unconditional startup imports/initializers, with
failure/thenable/scheduling/fixture/owner obligations before each
rejection/deferral. It proposes only a bounded loading-versus-initialization
attribution design (no pruning, edits, redesign or measurements) and hands the
next step to Codex. Source-proven removal only; zero timed trials; no speed claim.

## Codex independent acceptance

G7-G12 accepted after actual exit 0 and independent parent/child/group and prompt
retirement. Reviewed the five-file runtime diff and all seven additive regressions.
53 applicable tests (604 expects), runtime lint/format/type check and whitespace
check passed. Fresh production emission passed all 24 generated/native oracles,
including doubled/generic/captured map, fold and frequency; emitted module and
driver hashes match Muse. Independent scratch is absent. The initial compact
recorder used the wrong contract field after validation passed; corrected recording
was independently rerun, with both scratch lifetimes cleaned. Compact evidence:
`.performance/performance-push-20260928/generated-packet-2-independent-*.json`.
No timing benefit is claimed. Origin/callable/startup investigation continues.
