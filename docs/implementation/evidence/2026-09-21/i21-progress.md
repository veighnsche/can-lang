# I21 historical runtime-stage progress

This record predates compiler integration. The completed implementation and final
verification are documented in [I21 validation](i21-validation.md). At the time of
this record only the runtime component had been implemented.

`runtime/collections/array.ts` uses native `Array.fromAsync` over indices for mapping, predicate observations and key extraction; native filter/toSorted own selection and sorting. Fold and for_each use native reduce promise chains. Short-circuit find/some/every use the bounded await adapter over native values iteration. Concat, toReversed and append retain immutable output and original element aliases; slice already has its checked primitive adapter.

Every callback result remains a protected completion. First domain/standard failure stops later visits and retains the original failure box. Callback invocations enter child assertion frames with the callable creation receipt. The parent collection frame remains active between native reactions, preventing a later participant fixture from overtaking the traversal. Find constructs compiler-supplied option identities; it does not invent a new option representation. Nonfinite sort keys use the existing arithmetic standard-failure constructor.

Validation: `runtime/test/array.test.ts` passes 9 tests with 108 expectations on qualified Bun. Tests cover controlled sequential gates, thenable input/output and accumulators, empty operations, predicate stopping, exact first-failure propagation for all eight callback adapters, stable int/float/text/bool sorting, signed-zero ties, nonfinite-key stopping, native operation delegation, immutable copying and fixture scheduling under coordination. Strict TypeScript checking passes for the module and tests.

Native references read on 2026-09-21: [Array.fromAsync algorithm](https://tc39.es/proposal-array-from-async/), [native reduce](https://tc39.es/ecma262/multipage/indexed-collections.html#sec-array.prototype.reduce), [native toSorted](https://tc39.es/ecma262/multipage/indexed-collections.html#sec-array.prototype.tosorted). The implementation follows repository C7/Q11; these upstream sources describe the native mechanisms rather than superseding the pinned Bun qualification.

## Integration work remaining at the time of this record

- Admit the closed array catalogue in the checker, deriving exact callback inputs/results and finite error bounds from sealed types. Generic references need input constraints from the source element/accumulator, plus contextual output constraints when present; do not manufacture unknown/widened callable types.
- Use the existing catalogue inventory as declaration evidence, including array.slice/concat/to_reversed and prelude append. Preserve ordinary name eligibility and receiver rules.
- Construct option::value<T> through the private canonical catalogue scope and the existing specializer, without requiring a handwritten shape or leaking a new authored type facility.
- Add checked array-operation IR and native invocation emission. Existing Native expression steps assume synchronous pure values; async completion-returning array adapters require an explicit checked invocation path. Maintain once-only receiver/initial/callback preparation and coordinated participant hoisting.
- Support fixtures through the same checked contract and reserve callback occurrences within the enclosing collection invocation. Include callback captures in existing owner capture evidence.
- Prove positive source composition, contextual generic callbacks, exact escaping bounds, illegal callback/result/key rejection, first-failure stopping and immutable aliases through emitted strict TypeScript and offline staged assertions/run.
- Complete final runtime/compiler/integration gates, evidence report, task ledger update and commit only after all requirements pass.
