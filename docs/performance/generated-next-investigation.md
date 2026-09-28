# Next generated callback/origin/startup investigation

Status: read-only. No implementation, benchmark, harness redesign or measurement.
Packet G7-G11 is implemented and verified separately; this note hands the next
supported step to Codex for reviewed design. The campaign continues after this
packet; nothing here declares it complete.

## What remains on the per-element path

After G7 (absent-context `callContext`/receipt bypass) and G8 (private success
direct extraction), the production per-element chain for doubled/generic/captured
map, fold and frequency is, in `runtime/collections/array.ts`:

1. `observations`/`sortBy` drive `Array.fromAsync(source.keys(), ...)`; `fold`
   /`forEach` chain `Array.prototype.reduce` with `prior.then(...)`. Native
   delegation counts remain mapping 3, filtering 1, reducing 2, sorting 1
   (`runtime/test/array.test.ts` native-delegation test).
2. Private `callback` (array.ts) calls `invoke(() => action(...), origin)`
   directly when `trace.context` is undefined, forwarding identical
   owner/context positions. With a context it retains `callContext` plus
   `callableInstance(action)` receipt lookup, frame suspend/start/finish and
   invocation lineage unchanged.
3. `invoke` (`runtime/completion.ts:72-84`) allocates one thunk closure per
   visit, awaits the action promise, authenticates the carrier
   (`checkedCompletion`), refuses unboxed/forged results before assimilation,
   and records the first standard `boundaryOrigin` (`locatedCompletion`).
4. The action is the emitted async callable wrapper
   (`compiler/internal/emit/callables.go:102,104`):
   `async (...): Promise<Completion> => savedTarget(...)`. The target is an
   emitted async leaf (for example `double`) that computes synchronously and
   returns one `success` carrier. The wrapper adopts the target promise.
5. Success payloads flow through privately proven carriers
   (`PrivateSuccess` in array.ts): observations store only
   invoke-authenticated ok carriers past the ok guard; sort decorations are
   `success()` results; fold accumulators start from `success(initial)` and
   forward only guarded ok results. Direct `.value` extraction uses the same
   native `map`/`filter`/`reduce`/`toSorted`; every invoke/ok/failure guard
   and all public `value`/`checkedCompletion` validation remain.

Native controls (`tools/performance/drivers/runtime-bench.ts`) do one native
`map`/`reduce`/loop, one freeze and one `success` box, with no per-element
`invoke`, context threading, wrapper or carrier. The remaining gap is
structural (per-element failure boxing and context threading), not a single
allocation site. Counts below are source boundaries, not latency claims.

## Origin objects: static spans versus dynamic steps

Emitted origins are object literals assigned per function and per marked
expression:

- Function entry: `let $canOrigin = {source,start,end,invocation}` in
  `regions.go:241`, `wrap.go:47,101`, `coordination.go:32`,
  `participant.go:114`, `fetch.go:81`, `llm.go:49`, `judge.go:209,295`,
  `program_entry.go:121`.
- Per-expression reassignment: `$canOrigin = {...}` in `regions.go:195,216`,
  paired with source-map marks (two `mappingMark` tokens per assignment).
- Literal shape: `origin()` in `regions.go:178-184` emits
  `{source,start,end,invocation:[regionID]}` with `sourceID()` from
  `regions.go:185-190` (explicit `SourceID` override or region source).

Three dynamic dimensions prevent naive hoisting of every origin to one frozen
constant per function:

- Tail-loop steps: proven self-tail regions lower to `while (true)` with a
  `loopStep` counter (`regions.go:172-174,231-232,247,813`). Origins inside
  the loop append `"step:"+counter` to the invocation array
  (`regions.go:180-182,212-214`). Each iteration needs a distinct step value;
  a single hoisted origin would freeze the wrong step.
- Substituted-definition source IDs: template substitution moves definition
  nodes into use regions; `markNode` (`regions.go:203-216`) emits the
  definition file's source (`node.Source`) for substituted nodes and the
  region module otherwise, so offsets never pair with the wrong file. Hoisting
  must preserve per-node definition-file spans, not one module-level source.
- Source-map marks: `mappingMark` (`mappings.go:16-19`) wraps each assignment
  in NUL-delimited tokens removed before TypeScript parsing. Two marks stack
  per assignment; two marks at one generated coordinate fail the build
  (`self_tail_test.go:167-169`). Any origin motion must keep mark pairing and
  coordinates intact.

`failure.ts:54-61` (`freezeOrigin`) already copies the invocation array on
capture, so a hoisted frozen origin would still need per-capture invocation
handling for step-bearing origins. Origin hoisting is therefore an
emitter-owned change touching regions/wrap/coordination/participant/fetch/llm/
judge emission, mapping extraction and failure capture. It is not promoted
here: no supported runtime-only step remains, and any proposal needs a Codex
design with difficult-change consultations, plus regression gates for
definition-file spans, tail-loop step metadata, stacked-mark rejection and
`boundaryOrigin`/occurrence preservation.

## Callable wrapper promise adoption

`callables.go:99-105` mints one `ownCallable` per callable expression with an
`async (...): Promise<Completion> => invoke` forwarder, where `invoke` is
`savedTarget(...)` (or an array invocation). Every emitted target is async, so
the wrapper always adopts a native promise: one hop per element for map/fold
callbacks. The wrapper also threads `$canContext` (and `$canCtx` owner context
in browser profile, `callables.go:79-83,92-97`), preserves
`resourceIndices` capture evidence (`callables.go:17-23,102,104`), and carries
the callable mark (`callables.go:100`).

Dropping `async` from the wrapper (returning the target promise directly)
would change:

- Failure obligations: sync throws from arbitrary targets currently become
  rejections through the async wrapper; a sync forwarder would throw
  synchronously into its caller. `invoke` catches sync throws (`completion.ts:
  76-83`), but timing, stack capture and the every-function-is-async contract
  change for all callers, not just array adapters.
- Thenable obligations: the async wrapper assimilates returned thenables
  before `invoke` sees them; a sync forwarder would hand thenables to
  `invoke`, which refuses unboxed results before assimilation. Behavior for
  forged thenables would change (arguably safer, but still a contract change
  across arbitrary targets).
- Scheduling/fixture/owner obligations: wrapper adoption hops interact with
  fixture barrier accounting (parent frame active across visits), participant
  selection, capture timing (`ownCallable` receipts in `callable.ts:18-46`,
  `callableInstance` identity) and browser owner threading. An emitter-wide
  conditional lowering would expand fixtures/origins/owner review far beyond
  the private array branch.

Recommendation: reject a speculative wrapper de-async in this lane. If Codex
wants it, it needs a dedicated emitter design, three fresh equivalent
consultations, and gates covering sync-throw/rejection equivalence, thenable
refusal without assimilation, fixture gaps, coordination selection, owner
leases/rollback, browser controls and callable receipts. No runtime-only
wrapper change is supported.

## Frequency map path

`frequency` folds words through `count_one`: per word, one fold-callback
`invoke` plus `map.get` plus `map.insert`/`map.replace`, each an async method
with its own carrier. G9 reduced each `backing` call to one metadata lookup
after the same non-null object and concrete-identity guard; public
`isMap`/`isSet`, key checks (`checkKey`), resource-state origins, native Map
copies (`new Map(source)`) and opaque containment/leases are unchanged.

Remaining per-word work (immutable `new Map` copies, three invoke/context
pairs, per-word carriers) preserves immutability, failure/lease semantics and
old-snapshot histories (`collections.test.ts`, `collections-owner.test.ts`).
Making map methods sync would still leave async `callContext`+`invoke` above
them and would need a scheduling/ownership design. No further map/set removal
is proposed here.

## Startup: unconditional imports and initializers

`program/state.ts` is generated by `program_state.go:419-468`
(`stateImports`, fixed initializer order) on top of `programImports` /
`browserProgramImports` (`runtime_core.go:5-41`):

- Core (every workload): `collections/array.ts`, `assert/context.ts`,
  `coordination.ts`, owner types, `bytes.ts`, `callable.ts`,
  `assert/fixtures.ts`, `assert/policy.ts`, `completion.ts` (+ types),
  `data.ts`, `primitive.ts`, `failure.ts`. Browser profile swaps coordination
  to `settleWithContext`, adds owner types, and drops context/fixtures/policy.
- Domain factories (unconditional, non-browser): `domain.ts`, `platform/cli.
  ts`, `bytes.ts`, collection state, `text.ts`, `number.ts`, `checks.ts`,
  `codec/json.ts`, `platform/clock.ts`, `platform/random.ts`,
  `platform/log.ts`, `platform/html.ts`, `platform/form.ts`,
  `platform/assets.ts`, SQL state, crypto state, utilities state,
  `platform/http.ts`, `platform/router.ts`, fetch state, action state,
  `platform/server.ts`, file/process/stream/websocket state, cookies state,
  S3 state, markdown state, browser state, AI state, `environment.ts`,
  `platform/io.ts`, `platform/env.ts`, arm descriptions. Browser builds omit
  server/files/processes/streams/websockets/S3/AI/env/IO/arm/SQL/crypto
  groups but still import the remaining platform surface.

The gallery fixtures exercise only arrays, maps, text, bytes and codec, yet
every emission parses, loads and evaluates the full platform graph and runs
its factory initializers (`program_state.go:345` and per-domain state
builders). This is a distinct loading (module parse/load/evaluate) versus
initialization (factory construction, for example `$canCreateIO`,
`$canCreateEnv`, SQL pools, crypto, router/server) candidate.

Proposed bounded attribution (design first, no implementation here):

- Ownership: Codex designs the attribution; a future Muse packet implements
  only what that design specifies. No pruning or lazy initialization in this
  note.
- Tasks: enumerate the actual unconditional `stateImports` targets and the
  initializer statements each enables for the runtime fixture emission;
  separate module-load cost (parse/evaluate, import graph size) from factory-
  construction cost (per-factory initializer invocations); keep browser and
  non-browser import lists separate; preserve all factory contracts, error
  identities and state value imports (`stateValueImportNames`,
  `program_state.go:472+`).
- Checks: attribution must not change emitted behavior; any later pruning
  proposal needs per-factory usage evidence, loading-vs-initialization
  numbers from the accepted attribution only, and regression gates for every
  platform domain whose import/initializer moves. Global measurement defaults
  stay unchanged; the campaign-only quiet waiver applies only as the master
  plan allows.
- Defer reason if not pursued: without the accepted attribution, startup
  pruning risks breaking untested platform initializers for no proven gain.
  Defer pruning until the attribution exists; do not combine it with callback
  changes.

## Precise next steps

1. Codex reviews G7-G11 (source diff, 86-test bounded suite, 24-case emitted
   validation, evidence JSON) and either accepts or sends narrow corrective
   tasks to Muse. No further Muse implementation until that review lands.
2. If Codex wants origin work, it supplies a reviewed emitter design plus
   three fresh equivalent consultations covering static-span hoisting with
   preserved definition-file sources, tail-loop steps and source-map marks;
   Muse implements only that design with the mapping/occurrence gates above.
   Without that design, origin work stays deferred for the stated reasons.
3. If Codex wants callable-wrapper work, it supplies a reviewed emitter design
   plus consultations covering sync-throw/thenable/scheduling/fixture/owner
   equivalence; otherwise the wrapper stays as is.
4. If generated startup remains supported, Codex designs the bounded
   loading-versus-initialization attribution above; Muse implements only the
   attribution harness changes in that design, with no pruning until the
   attribution is accepted.
5. Otherwise, Codex selects the next supported generated candidate from the
   reviewed queue and saves a new ordered Muse checklist; the campaign
   continues through that handoff, not through speculative edits here.
