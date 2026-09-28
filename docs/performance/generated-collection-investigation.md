# Generated/collection investigation (I1, read-only)

Source revision `318ed458`. No implementation, measurement, build, or
source edit was performed for this note; all paths below were inspected
read-only. Timing numbers are the saved historical run
`.performance/20260927T235706.371974Z/report.md` (older source revision;
hypothesis input only, not a current baseline).

## Saved measurements and fixture contracts

`tools/performance/drivers/runtime-bench.ts` runs the `generated` suite with
`size = 100`, wall time per repeated workload call including
await/completion and result publication; fixtures, initialization, and
correctness checks excluded; 6 independent processes per case.

| Case | can median | native median | ratio |
|---|---|---|---|
| doubled (bigint-array-map-frozen-success) | 78.2705 µs/op | 8.604 µs/op | 9.1x |
| generic-map (int-identity-callback-array-map) | 64.646 µs/op | 6.5005 µs/op | 9.9x |
| captured-map (offset-capture per map call) | 60.4165 µs/op | 7.6875 µs/op | 7.9x |
| fold-sum (bigint-reduction-zero-seed) | 60.7915 µs/op | 7.458 µs/op | 8.2x |
| frequency (immutable-map-fold, 10 unique words) | 260.834 µs/op | 13.3125 µs/op | 19.6x |
| branch-arithmetic | 44.208 µs/op | 30.6045 µs/op | 1.4x |
| failure-recovery | 126.188 µs/op | 23.208 µs/op | 5.4x |
| record-projection / record-update / tagged-variant | 35–59 µs/op | 24–45 µs/op | 1.2–1.5x |
| unicode-normalize / utf8-base64 | 11.27 / 4.85 µs/op | 9.29 / 1.5 µs/op | 1.2x / 3.2x |

Fixture contracts (`tools/performance/fixtures/runtime/src/*.can`):
`doubled.can` maps `callable double` over `int[]`;
`fold.can` folds `callable add` from `0`;
`frequency.can` folds `callable count_one` (get, then insert or replace with
match arms) starting from `collections::empty_map`, then a final `get`.
Checks assert values, frozen results/inputs, and input-unchanged.

Crucially, the native controls share the workload's copy strategy:
`doubled.native` is one `input.map` plus `Object.freeze` plus one `success`
box; `frequency.native` copies `new Map(counts)` per word just like the Can
side. The 8–20x gaps therefore cannot be the copy strategy; they sit in the
per-element/per-call layers both fixtures traverse. No profiling data
attributes the gap to individual layers (see caveat below).

## Concrete per-call costs (source-counted, not timed)

Emitter (`compiler/internal/emit/array.go`, `callables.go`): array
higher-order calls emit `$canArrayMap(source, action, trace)` with a
fresh `{site, origin, context}` trace object per call site evaluation;
every generated function is `export async function $canFunctionN(...,
$canContext?)` returning `Promise<Completion<...>>`; every generated call
is `$canInvoke(() => $canCallContext(ctx, site, fn, $canCallableInstance(fn)),
origin)`. A checked-in sample
(`docs/syntax-taste/evidence/2026-09-26/.../recurse-large/prod/...`) shows
one `$canOrigin = {...}` reallocation per subexpression span and labeled
match-region blocks per `match` arm.

Runtime array adapter (`runtime/collections/array.ts`): `map` runs
`observations`, which dispatches one `Array.fromAsync` task per index; each
element traverses `callback()` → `callContext()` → `invoke()` → the
generated `async` callback → `success()` box — at minimum five promise
allocations/ticks per element before user arithmetic. `fold`/`forEach`
chain one `.then` per element over the same per-element stack. The final
`map` collects `Completion<U>[]` and re-maps through `value()`.

Completion (`runtime/completion.ts`): every `success()` does
`Object.create(null)` + two `defineProperties` + `Object.freeze` + a
`WeakSet.add`; every `value()`/`invoke()` does `WeakSet.has` brand checks
plus kind dispatch; `invoke` is itself `async` with try/catch and
`locatedCompletion`.

Context (`runtime/assert/context.ts`): with `context === undefined`
(the benchmark shape) `callContext` is a direct passthrough, but it is
still declared `async`, so it costs a promise allocation and a microtask
hop per element callback. With a context it additionally allocates
invocation lineage and suspends/starts/finishes/resumes barrier frames.

Callable (`runtime/callable.ts`): `ownCallable` (once per `callable`
expression evaluation, i.e. once per `doubled` call, not per element)
validates, allocates a `Set` for capture indices, registers captures,
computes an identity, and freezes a receipt plus a captures copy.
`callableInstance` (a receipt lookup) runs per generated call.

Map adapter (`runtime/collections/map.ts`): `get`/`insert`/`replace` are
`async`, each doing two `WeakMap` lookups (`isMap` + `backing`), a
`checkKey` dispatch, and a completion box. Every point update copies the
full backing store (`new Map(source)`) and `own()` additionally runs
`registerOpaqueContents(token, Array.from(values.values()))` — a full
values-array copy plus freeze per insert/replace/remove. `frequency`
therefore pays, per word: fold-chain hop + callback stack + async `get` +
match + async `insert`/`replace` with a full-map copy and a full-values
copy. Data constructors (`runtime/data.ts`): `array()` is one freeze
(cheap); `record()` is create-null + per-field `defineProperty` + `WeakMap`
set + freeze; `update()` is assign + per-replacement `defineProperty` +
freeze — consistent with the small 1.2–1.5x record gaps.

Owner layer (`runtime/owner-core.ts`, 834 lines): `runOwnedRoot` and
resource capture/registration paths exist but these workloads capture no
resources; no owner drain appears on the hot path. Owner cost is not an
attributed factor here.

## Semantic obligations any fix must preserve

Immutable values: frozen results, frozen inputs, input-unchanged checks.
Completion boxing: no naked payload may cross an `await` (thenable
assimilation would corrupt data-valued `then`; see the `observations`
comment). Failures: `ok`/`domain`/`standard` kinds, `errorType`/`payload`
extraction, per-callback error bounds, and match-arm failure routing
(`count_one` relies on the absent/exists arms). Callback order: `map`
observations settle per index; `fold` is strictly sequential with
accumulator threading. Context: fixture lineage, barrier frames, and
violation evidence when a context is present. Ownership: resource capture
indices and drain for capturing callables. Thenable safety for all
`Array.fromAsync`/await boundaries.

## Proven vs unprofiled (explicit separation)

Proven by source inspection: the per-element hop counts, per-box
`defineProperties`+freeze+`WeakSet` traffic, per-subexpression `$canOrigin`
reallocation, per-trace-object allocation, per-map-update full `Map` copy
plus full `Array.from(values)` copy, and the `async` declarations on
`callContext`/`invoke`/adapters even for synchronous callbacks.
Unprofiled: how the measured microseconds divide among these layers. In
particular, `frequency.native` shares the O(n) copy yet is 19.6x faster,
which bounds the copy's contribution but does not apportion the rest
without a bounded attribution exchange. No layer may be called the cause
until measured.

## Proposed small next assignments (no implementation yet)

1. Bounded attribution micro-exchange (Codex-supervised measurement):
   isolate `box`/`value`/`invoke`, undefined-context `callContext`,
   `ownCallable`/`callableInstance`, and one `insert`+`get` round trip at
   fixed map sizes. Files: `tools/performance/drivers/runtime-bench.ts`
   additions only; tests: existing bench contracts. Settles which layer
   dominates before any semantic change.
2. Sync-callback fast path in `runtime/collections/array.ts`: when the
   callback returns a synchronous `Completion`, trace has no context and
   no owner, and the action is not a browser callback, reduce hops while
   keeping boxing, order, failure routing, and thenable safety. Owner:
   runtime adapter + `runtime/test/array.test.ts` (+ generated-contract
   tests). Needs Codex design + three Jev consultations: failure and
   context edge semantics are subtle.
3. Cheaper completion box: investigate replacing per-box
   `defineProperties`+freeze+`WeakSet` traffic with an equivalent brand
   that keeps forged-value/proxy rejection. Owner: `runtime/completion.ts`
   + `runtime/test/completion.test.ts`. Security-sensitive; design +
   consultations required.
4. Map point-update copies: investigate whether `Array.from(values)` in
   `own()` and/or the full `new Map(source)` copy can be narrowed without
   breaking opaque-contents publication or immutable history. Owner:
   `runtime/collections/map.ts` + `runtime/test/collections*.test.ts`.
   Design + consultations required (ownership semantics).
5. Emitter trace/origin allocation: investigate reusing one frozen origin
   per call site instead of per-subexpression reassignment where
   diagnostics allow. Owner: `compiler/internal/emit/*` + emit goldens.
   Design + consultations required (diagnostic-span obligations).

Items 2–5 must not be promoted before item 1 (or equivalent attribution)
and the required three design consultations each.
