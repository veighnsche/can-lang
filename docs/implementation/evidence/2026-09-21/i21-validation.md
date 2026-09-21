# I21 — immutable arrays with asynchronous callbacks

I21 implements the twelve existing closed-catalogue entries: map, filter, fold,
for_each, find, some, every, sort_by, slice, concat, to_reversed and prelude append.
The catalogue inventory remains the declaration/identity source. Slice calls reuse
the checked native slicing implementation established by I07; bound slice
references now use that same implementation.

## Implementation

`runtime/collections/array.ts` maps native indices with `Array.fromAsync`, keeps
callback results in completion boxes, then unwraps synchronously. Filter collects
sequential boolean observations before native filter. Sort extracts each key once
before native toSorted, compares primitive keys with native pairwise comparisons,
and uses the original index for ties. Nonfinite float keys produce the existing
arithmetic failure. Fold and for_each use native reduce await chains. Only the
short-circuit searches use a bounded awaited traversal over the native values
iterator. Native copying preserves immutable source arrays and element aliases.

The checker admits exact callback inputs and results and retains the callback's
finite declared error bound. Existing finite inference receives concrete array
element/accumulator and available result constraints for generic named and bound
references. Near and receiver captures use existing once-only capture machinery.
Direct calls and first-class array references share the concrete contract checker.
Empty literals can use concrete callback input evidence; append supports inferred
and explicit element types. Array callback references require enough concrete
callable input evidence to determine their result and error contract.

Explicit array invocation IR carries operation and sealed option identities.
Emission retains ordinary argument preparation, fixture contracts, participant
hoisting and invocation frames. Runtime callback frames retain callable receipts;
the collection frame remains active across native traversal gaps. Captured array
references use existing ownership capture evidence. Find constructs the existing
nominal option specializations, with a generated TypeScript assertion justified
by the two checked identities rather than a second option representation.

Generic catalogue record/error constructor inference now reads closed metadata
through the existing specializer. This enables ordinary inferred `option::some(2)`
and keeps authored import/constructor eligibility checking in place. Private
catalogue parameter spelling is normalized for the authored type parser; no
placeholder types or checker-time user execution are introduced.

## Acceptance evidence

| Requirement | Evidence |
| --- | --- |
| Required order and empty behavior for all eight callback operations | Runtime controlled-gate/visit-trace tests; generated `arrays/main.can` assertions for all operations and empty inputs. |
| First domain/standard failure stops later callbacks | Runtime checks exact original completion identity and visits `[1,2]` for every adapter; generated domain and standard failure handlers, including a fallible first-class map reference. |
| Nonfinite sort rejection | Runtime NaN and both infinities; generated `nonfinite_sort` assertion. |
| Then-bearing inputs/results and accumulator remain data | Runtime zero-assimilation counters and alias checks; generated `protected_payloads` and `protected_fold`. |
| Key evaluated once and ties stable | Runtime int/float/string/bool and signed-zero traces; generated `stable_sort` consumes exactly three ordered fixture rows and preserves tied input order. |
| Source arrays unchanged | Runtime frozen-copy/alias assertions and generated `immutable_copies`. |
| Exact contracts and finite errors | Checker rejects wrong predicate/iteration/map/fold/key shapes, non-callables, element mismatches, wrong result arrays, and discarded callback error bounds. |
| Captures and coordination compose | Generated near/generic bound callbacks, map/find/reverse/slice/append references, and concurrent mapping sharing one lexical fixture FIFO. |
| Native delegation | Runtime instrumentation verifies fromAsync, filter, reduce and toSorted delegation; no fallback implementation is installed. |

## Verification

- Qualified Bun 1.4.2: all 151 runtime tests / 1,117 expectations passed.
- Array-specific runtime suite: 9 tests / 108 expectations passed, also executed
  from the staged release's absolute private runtime with network denied and
  `PATH=/nonexistent`.
- Strict TypeScript checking passed for the array runtime/tests.
- Full `go test ./compiler/... ./tests/integration -count=1` passed with `CAN_BUN`,
  the pinned local `CAN_BUN_ARCHIVE`, and `CAN_TSC` enabled.
- The final bound-slice reference and rejection-test additions were followed by
  passing checker/emitter suites and another staged array integration run.
- Staged source integration executes 59 mandatory assertion roots across `main`
  and `contracts`, runs both programs, and strictly type-checks every generated
  TypeScript artifact. The launcher is invoked from an unrelated directory with
  ambient Bun unavailable and network access denied.

Jev requests, responses and the pre-dispatch wording audit are in `i21-jev`.
Native algorithm references and the earlier runtime-only progress record are in
`i21-progress.md`. Advice and target qualification do not substitute for the
source/runtime acceptance evidence above.
