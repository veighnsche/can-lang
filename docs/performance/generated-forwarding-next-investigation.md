# Generated forwarding follow-up: direct routes, remaining boundaries, startup

Status: read-only investigation for Codex's next design. No implementation,
benchmark, harness change or measurement in this note. G19-G23 (checked
authored callable forwarding) are implemented and verified separately; this
note hands the next supported step to Codex. The campaign continues after
this packet; nothing here declares it complete.

## Which generated callback routes now forward directly

After G19/G20, every non-array callable whose checked identity and resolved
emitted binding both match the assembly proof emits a non-async forwarding
arrow that returns the saved native async target's promise directly, keeping
the explicit `Promise<$canCompletion<T>>` annotation, saved target/capture
sequence, resource indices and owner/assertion context positions. The proof
covers production, browser-profile and assertion emission
(`program_modules.go` authored/native sites, `program_entry.go` assertion
site); direct region emitters without proof keep the legacy async adapter.

Fresh packet-4 evidence: the performance fixture emits 5 `$canOwnCallable`
constructions across 4 package modules and all 5 select direct forwarding
(`direct:5 async:0`); the 4 changed package modules are exactly the modules
containing those callables while `entry.ts`, `generated-adapter.ts`,
`program/state.ts` and the 2 callable-free package modules are byte-identical
to the pre-packet hashes. All 24 generated/native oracles pass with strict
`tsc` and 14/14 adapter bindings. G21 pins ordinary, captured, generic
(`callable identity<int>`), same-spelled-distinct-source and
browser/assertion selection plus async fallback for array, catalogue,
unclassified and stale-proof targets, all Bun-executed.

## Remaining repeated boundaries

1. `invoke` thunk per visit (`runtime/completion.ts`, `invoke`). Every array
   element visit allocates the `() => action(...)` closure; `invoke` admits
   a synchronous `Completion` return, refuses unboxed/forged results before
   assimilation, awaits carrier promises, records the first standard
   `boundaryOrigin` via `locatedCompletion`, and converts sync throws via
   `caught`. Semantic blockers for removing the thunk: the `Callback` type
   admits synchronous `Completion | Promise<Completion>` returns, and
   generic synchronous callers remain valid — the G21 fallback test executes
   a sync-returning then sync-throwing unclassified control through the
   async adapter. No universal promise-only runtime contract exists, so a
   runtime-wide promise-only fast path is defeated by design, not merely
   unproven. Proof opportunity, if any, is per-site at the emitter (this
   packet's shape), never a type-wide runtime assumption.
2. Array `callback` wrapper (`runtime/collections/array.ts:35-67`). The G7
   bypass already calls `invoke` directly when `trace.context` is undefined;
   the context path adds `callContext` plus a `callableInstance` receipt
   lookup. Blockers: fixture barrier accounting with the parent frame
   active across visits, receipt identity for fixture rows, and
   owner/context positions. No narrower proof is established here.
3. `observations` `Array.fromAsync` mapper and fold `reduce`/`then` chains
   (`array.ts:79-91,160,191`). Visits are proven strictly serialized
   (`runtime/test/array.test.ts` ordering/peak-one coverage); any sequential
   replacement still needs the origin follow-up's three-part proof (native
   algorithm equivalence, fixture/owner/lease scheduling, thenable
   non-assimilation). Carried unchanged.
4. Direct-call `invoke` thunk (`regions.go` `configure` Call:
   `await $canInvoke(() => target(...))`). Same sync-throw admission
   blocker as (1): call targets include catalogue/native operations outside
   authored proof. A per-site bypass for proven authored targets would still
   need to preserve `locatedCompletion` boundary recording and
   unboxed-refusal semantics; that is a future design with consultations,
   not a claim made here.

## Reconciliation with the origin follow-up's invoke-thunk proposal

`generated-origin-next-investigation.md` held candidate 1 (private
promise-only `invoke` fast path) for missing proof that every production
array action is a never-throws-sync async function. This packet resolves the
shape of that question without implementing the candidate: the valid
optimization is emitter-side per-site de-async under exact
identity-plus-binding proof (shipped, 5/5 fixture routes, fallback
preserved), while the universal runtime variant is now affirmatively
defeated — sync-returning and sync-throwing callers are real, tested
controls (`TestForwardingFallbackKeepsAsyncExecutable`), and the `Callback`
type still admits them. The origin's required enumeration (every production
caller of the eight array entry points from `arrayInvocation` shapes plus
runtime-internal callers) remains the gate for any array-site-specific step;
generic synchronous callers stay valid regardless.

## Startup: actual stages, still unattributed

- Compilation (not loading): `bun build` transpile/bundle work in
  preparation (`runtime.py prepare`), and `runtime-transpile.ts`
  (`Bun.Transpiler`, module `.ts` to `.js` rewrite). Never a loading proxy.
- Loading: fresh-process module import/evaluation of the prepared graph —
  emitted `entry.ts` (`runEntry` in `runtime/entry.ts:61`), authored
  package modules, `program/state.ts`, and the full platform runtime import
  graph (`programImports`/`browserProgramImports`, `runtime_core.go:5,28`).
- Initialization: factory construction via `$canInitialize`
  (`emitStateModule`, `program_state.go:36`) in `stateImports` order, plus
  `configureDiagnostics` and error-plan resolution before `$canMain`.

Loading and initialization remain unattributed; no per-stage timing was
taken in this packet. Smallest bounded attribution packet (design only):
reuse one-build perfemit emission with the zero-byte inventory plus
runtime-symlink preparation; time separately on the busy host with observed
activity noted — (a) real dynamic imports / fresh-process loading of the
prepared startup entries, (b) per-factory timers around each
`program/state.ts` initializer statement in `stateImports` order,
(c) `bun build` of the startup entries reported apart as compilation,
(d) the existing `validate` bench mode as the correctness gate. Explicit
identities: loading = emitted import graph paths/bytes/hashes;
initialization = per-factory names in order; compilation = transpile/bundle
only; correctness = every factory constructs, error-plan identities
resolve, 24/24 validation unchanged. No new harness, installs, retained
bundles, pruning, lazy initialization or factory reordering. Ownership:
Codex designs with any required consultations; a future Muse packet
implements only that plan plus a focused emission test asserting the
enumerated import/initializer inventory matches the attribution input.

## Preserved lanes and dispositions

- Secondary numeric JSON retention candidate unchanged:
  `runtime/codec/document.ts:28-57` retains source spelling for every
  primitive while typed codec/JSONL `exactInt` and the raw
  assertion-provider number branch need only numeric spellings. Still
  secondary behind generated async/startup work; any change must preserve
  exact integers, raw numeric comparison, root-holder tracking, duplicate
  detection, native syntax checks, budgets and all rejection paths.
- All twelve queue-review slices
  (`performance-queue-review.md`, revision `40eaf6a88`) remain as disposed:
  this packet narrows only the generated per-element callback path (one
  async adoption layer at proven sites, unmeasured for latency) and closes
  no slice. Compiler, Editor, Generated, Runtime, Codecs, Startup,
  Assertions, Artifacts, Server, I/O, Journeys and Browser dispositions
  carry forward; the master final exhaustion review stays pending until no
  ready supported fix remains.

## Handoff to Codex

- For any array/invoke follow-up, supply `completion.ts` `invoke`/
  `locatedCompletion`, `array.ts:35-67`, `callables.go`, `array.go`, the
  packet-4 proof/predicate diff, and the G21 fallback execution test as the
  defeating evidence for universal promise-only designs; questions must
  cover per-site versus type-wide proof, sync-throw equivalence, thenable
  refusal, fixture/owner/lease scheduling, browser callbacks and receipt
  preservation.
- For startup, supply the `stateImports` inventory, per-factory initializer
  list and the four-identity attribution sketch above; questions must cover
  loading/init/compilation separation, factory-contract preservation and
  busy-host labeling.
- For the codec lane, supply `document.ts:28-57` and the queue-review
  disposition; no consultation until promoted.
- Campaign state: G19-G24 implemented with 24/24 validation and strict TS;
  independent acceptance of the forwarding packet and the next ordered
  checklist belong to Codex.
