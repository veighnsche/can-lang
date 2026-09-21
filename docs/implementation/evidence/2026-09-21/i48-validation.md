# I48 validation

Completed 2026-09-21. Initialization checking now admits exactly the C8 expression
families, checks them with sealed declaration types and builds a dependency graph
from resolved value identities. Ready values initialize in qualified-name order;
forward references work and dependency cycles fail. No source evaluator, purity
classifier or environment probe participates in that decision.

Native emission runs the ordered statements before entry invocation. Each value
is assigned once, ordinary records/arrays retain their immutable native data
representation, and a startup fault immediately throws an I49 standard occurrence
with the initializer's source span and invocation origin. Later initializers and
main cannot run past that throw.

The finite-local checker requires all four predicates:

1. The final typed binding immediately precedes `ok name` or a value-arm final
   name in the same block.
2. Complete resolved source-use accounting finds exactly one direct use and no
   implicit `near` capture. Equal spelling does not merge different bindings;
   missing name/capture evidence is rejected rather than assumed absent.
3. The initializer is inside the specified literal/name/group/field/primitive
   expression subset. Calls/references, construction, arrays, updates, matches,
   indexing/slicing and divide/remainder/power/shift remain excluded. Non-faulting
   field reads are checked through typed expression operations.
4. Rechecking substitution under the terminal expected type preserves the local's
   concrete type and the original expression's node-by-node typing/conversion
   evidence. Failure or changed inference retains the binding.

A successful diagnosis includes the original initializer text as the direct
replacement. No code elimination, effect inference, arbitrary equivalence proof
or cost model is performed.

Evidence:

- [Focused regressions](i48-regression-tests.txt): parsed source initializers,
  deterministic forward references and cycles, calls in skipped operands,
  environment/resource attempts, match/callable refusal, primitive startup faults,
  ordinary data/index/slice/update, named-arm admission evidence, direct-forward
  diagnostics, finite exclusions, shadowing/capture/use accounting, empty-array
  retention and preserved variant typing.
- [Full Go suite](i48-go-tests.txt): passed with pinned native emitted execution.
- [Staged offline integration](i48-offline-integration.txt): generated startup
  modules execute using the absolute staged Bun with network denied and PATH
  unavailable. Two imports observe one initialized record; faulting startup
  reports its expected source span and never reaches the entry marker. Sidecar
  identity, integrity and negative checks also pass.
- [Three Jev consultations](i48-jev/README.md): advisory design review with raw
  requests and responses retained.

These are compiler passes with explicit checked-context interfaces. I09 will own
whole-program ESM publication; I10's body/region checker supplies complete lexical
usage records and terminal contexts, I08 supplies named-reference capture records,
and the named-arm declaration checker supplies capture-free arm evidence. Tests
exercise the interfaces without claiming these later tasks or the I11 CLI are
already complete. The source location currently identifies the initializer;
expression-level source-map machinery remains I44.
