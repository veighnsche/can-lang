# Generated callback follow-up

Status: investigation only. No implementation beyond G1/G2, no benchmark,
no harness rewrite. Packet G1-G5 accepted separately; this hands the next
ready design to Codex. All generated code was inspected in an owned
temporary emission and removed afterward; no bundle or source copy retained.

## Actual emitted async path

Every emitted function is `async` and returns `Promise<Completion>`.
Representative bindings from `tools/performance/fixtures/runtime` (via
`drivers/runtime.py` origin-identity discovery):

- `doubled` maps via `values.map(callable double)`. Leaf `double`
  (`$canFunction0`) does `value * 2n` and one `success` box. Wrapper
  `doubled` (`$canFunction1`) mints one `ownCallable` per call, then
  `await invoke(() => callContext(ctx, site, () => arrayMap(...)))`.
- `generic_pass` maps via `callable identity<int>`; same shape, zero captures.
- `captured_map` mints one `ownCallable` with captures `[offset]`
  (`resourceIndices: []` for scalar offset) and an async forwarder
  `(arg, ctx) => add_offset(offset, arg, ctx)` per call to `captured_map`.
- `sum` folds via `values.fold(0, callable add)` through `arrayFold`,
  which chains `prior.then(...)` per element.
- `frequency` folds words through `count_one`, which per word does
  `map.get` then `map.insert` or `map.replace`, each via its own
  `invoke(callContext(...))` pair inside the fold callback.

Per-element map chain in production (`context === undefined`,
`owner === undefined`), `runtime/collections/array.ts:36-78`:

1. `observations` (`array.ts:66`) drives `Array.fromAsync(source.keys(),
   async (index) => ...)` sequentially (peak 1, proven by
   `array.test.ts:25`).
2. `callback` (`array.ts:36`) builds a per-element closure and calls
   `callContext(undefined, site, run, callableInstance(action))`.
   `callableInstance` (`callable.ts:51`) is a `WeakMap` lookup of the
   same `action` every element.
3. `callContext` (`assert/context.ts:285`) is `async`; its undefined
   path is `return run(undefined)`, which adds one async-frame
   microtask even though `run` already returns a native `Promise`.
4. `run` calls `invoke(() => action(...args, undefined), origin)`
   (`completion.ts:76`). `invoke` is `async`, allocates a thunk
   closure per element, awaits the action promise, and runs
   `locatedCompletion` (`completion.ts:71`) for the standard
   boundary check.
5. `action` is the emitted async wrapper
   (`async (arg, ctx) => leaf(arg, ctx)`), which awaits the leaf.
6. The leaf (for example `double`) is async, computes synchronously,
   and returns one `success` carrier (G1 shape).

After all elements, `map` (`array.ts:92`) does
`(await observations(...)).map(value)`, one `array()` freeze, one
`success` box, inside `boundary` (`array.ts:55`). Fold
(`array.ts:134`) serializes the same per-element `callback` through
`Array.prototype.reduce`; `search`/`sortBy` share `callback`.

Native controls (`tools/performance/drivers/runtime-bench.ts:38-113`)
do one native `map`/`reduce`/loop, one freeze, one `success` box, with
no per-element `invoke`, `callContext`, wrapper, or carrier. The gap
is structural (per-element failure boxing and context threading), not
a single allocation site.

Error origins: each generated step assigns `$canOrigin` (lexical
source/start/end/invocation) before evaluation; failures carry
`occurrenceID`, `origin`, and first-`invoke` `boundaryOrigin`
(`completion.ts:71-87`, `failure.ts`). `value()` rethrows non-ok
payloads; `boundary` converts thrown completions and foreign causes.

## Source-proven repeated work

Exact locations; counts are structural boundaries, not latency claims.

1. `callContext` undefined microtask. `assert/context.ts:285-291`:
   `async function` with `if (context === undefined) return
   run(undefined)`. Every array element and every generated call
   site (22 `canCallContext` sites in the six fixture modules) pays
   one extra async hop. `run` in `array.ts:42` and all generated
   call sites returns a native `Promise` (from `invoke`, `arrayMap`,
   collection methods). Helps emitted async callbacks.
2. `callableInstance` per element. `array.ts:52` looks up the same
   `action` in a `WeakMap` once per element. Loop-invariant; could
   hoist to once per `map`/`fold`/`filter`/`sortBy` operation.
   Helps async callbacks but each lookup is one cheap native
   operation; negligible alone.
3. Emitted async wrapper per element. Generated
   `async (arg, ctx) => leaf(arg, ctx)` (for example doubled
   `$canRegion6`, captured_map `$canRegion2`) awaits a leaf that
   already returns a native `Promise`. A direct `(arg, ctx) =>
   leaf(arg, ctx)` return would save one hop per element. Helps
   async callbacks but requires emitter change
   (`compiler/internal/emit`, callable emission).
4. `ownCallable` receipt trio per operation (not per element).
   `callable.ts:33-44` plus `owner-core.ts:533-541` plus
   `assert/lineage.ts:195-209` allocate three frozen capture arrays,
   two frozen records, one SHA-256 digest (`assert/identity.ts:36`),
   and two tokens per `ownCallable`. O(1) per map/fold call
   (one callable per workload in these fixtures), not O(n).
   Synchronous setup cost; does not help the per-element async path.
5. Collection method async hops per frequency word. `count_one` pays
   three `invoke`+`callContext` pairs per word (fold callback plus
   `get` plus `insert`/`replace`). Each map method
   (`collections/map.ts:39-100`) is async and boxes one carrier.
   Required for failure/lease semantics; making map methods sync
   would still leave `callContext`+`invoke` async above them.

Rejected synchronous-only or contract-breaking alternatives:

- `Array.fromAsync` to `for` loop: breaks the intentional
  native-delegation contract (`array.test.ts:386`, mapping 3,
  reducing 2, sorting 1). Rejected.
- Leaf sync functions: breaks every-emitted-function-is-async.
  Rejected.
- Map copy elimination or trusted transfer: breaks immutability and
  G2 containment. Rejected.
- `ownCallable` hash/receipt removal: breaks fixture identity
  (`callable.test.ts`, `assertions.test.ts`) for O(1) gain.
  Rejected.
- Bypassing `invoke` in array callbacks: loses standard
  `boundaryOrigin` capture (`completion.ts:73`); `boundary`
  alone returns thrown standard carriers without recording the
  boundary. Rejected.

## Proposed next remedy

Make the `callContext` undefined path return the inner promise
without an extra async frame, preserving the never-throw-sync
contract and the defined-context fixture path exactly.

- Files: `runtime/assert/context.ts` (`callContext`, currently
  lines 285-302); focused tests in `runtime/test/array.test.ts`,
  `runtime/test/assert-execution.test.ts`,
  `runtime/test/assert-provider-context.test.ts`, and existing
  `runtime/test/coordination.test.ts`,
  `runtime/test/owner.test.ts`,
  `runtime/test/browser-controls.test.ts`,
  `runtime/test/callable.test.ts` as regression gates; no
  compiler, emitter, harness, catalogue, or vendor edits.
- Shape (illustrative, Codex to specify): keep the exported
  signature; branch `if (context === undefined)` in a
  non-`async` outer function as
  `try { const result = run(undefined); return result instanceof
  Promise ? result : Promise.resolve(result); } catch (cause) {
  return Promise.reject(cause); }`, delegating the defined path to
  the current async barrier logic unchanged. The `callable`
  argument stays ignored on the undefined path, as today.
- Why native-equivalent: returns the already-native promise from
  `invoke`/collection/array methods instead of re-wrapping it;
  sync values still go through `Promise.resolve`, sync throws
  still become rejections.
- Prerequisites: G1-G5 independently accepted; three fresh
  equivalent Jev consultations if Codex judges assertion
  infrastructure difficult (AGENTS.md); no quiet-host or catalogue
  timing prerequisite.
- Ownership: Codex designs and supervises; Muse implements the
  narrow `context.ts` change plus focused regression assertions
  and reruns G3 gates plus G4 validation (size 32 validate, same
  no-retained-copy procedure).
- Tests: existing sequential/peak-1 (`array.test.ts:25`),
  failure-prevents-later-visits (`array.test.ts:198`), fixture
  gaps (`array.test.ts:314`), native delegation counts
  (`array.test.ts:386`), assertion execution/provider-context,
  coordination selection, owner leases/rollback, browser
  controls, callable receipts; add narrow assertions that
  undefined-context `callContext` returns the inner native
  promise identity for promise runs, wraps sync values, converts
  sync throws to rejections, and never invokes `then` on data.
- Negative cases: sync `run` value; sync `run` throw carrying a
  `Completion` and carrying foreign data; rejected inner promise
  with domain/standard/foreign cause; hostile `then`/getter
  payloads in results (no assimilation); defined-context barrier
  suspend/resume/finish unchanged; empty-source callbacks never
  run; `sortBy` nonfinite rejection still standard-arithmetic.

Obligations preserved:

- Scheduling: per-element sequential order, peak 1, failure
  short-circuit, fixture-gap serialization, parent frame active
  across visits; no new concurrency.
- Thenable: no `await` on data-valued `then`; indices only through
  `Array.fromAsync`; `Completion` carriers expose no `then`;
  `instanceof Promise` discriminates native promises from sync
  values without invoking user `then`.
- Occurrence/origin: `invoke` still records the first standard
  boundary; origins still flow from generated `$canOrigin`
  through `callContext` site threading; occurrence IDs unchanged.
- Context: `undefined` still threads as `undefined`; defined
  contexts still reserve/suspend/start/finish/abandon frames and
  allocate fixtures identically.
- Capture: `ownCallable` receipts, `callableInstance` identity,
  and `resourceIndices` filtering unchanged; this remedy does not
  touch capture storage.
- Lease: construction still acquires none; participant
  preparation/rollback, losing-participant retention, and close
  drain unchanged; callback path adds no lease.
- Cancellation: no deadline or abort semantics change; array
  adapters still run to completion or first failure.

## Startup imports (separate)

`generated/program/state.ts` (187 KB, 583 lines in the inspected
emission) statically imports 59 runtime modules covering the full
platform (CLI, SQL descriptors/pools/transactions, crypto, URLs,
datetime, HTTP, router, server, files, processes, streams,
websockets, cookies, CSRF, S3, markdown, browser, IO, env), plus
array/coordination/callable/completion/data/primitive/failure/domain
cores. Entry (`generated/entry.ts`) imports runtime entry, state
initialization, diagnostics, and main. Package modules add 207
import lines. The gallery fixtures exercise only arrays, maps,
text, bytes, and codec; most platform factories are never
constructed per validation, but they are parsed, loaded, and
evaluated at startup. This is a distinct loading/initialization
candidate requiring isolated attribution before any pruning or
lazy-initialization remedy; it is not proposed here because the
callback fast path above has direct per-element support. Do not
combine startup pruning with the `callContext` change.
