# I21 literal-spread and empty-array inference corrections

Independent review confirmed two compiler admission gaps after `abb30f1`:
direct array operations rejected known literal spreads accepted by C5, and empty
array receivers failed before using available map-result/fold-accumulator
constraints to specialize generic callbacks.

Fixed-arity inference now reads syntactically known literal elements, including
nested and empty spreads. Actual checking and emission use the existing ordinary
argument-preparation path on the original argument list. Thus every spread is
still a homogeneous array, prepared once in source order; flattening for inference
does not become a runtime flattening shortcut. This covers array callback and
copy operations, append's receiver/value positions, and native slice calls.
Unknown-length spreads into fixed positions, extra arguments, state groups and
incompatible element types remain rejected.

For empty map receivers, the expected array element supplies the callback-result
equality. For empty fold receivers, the checked initial value supplies the
accumulator input and result equalities. Unavailable input equalities are omitted
from the existing finite solver; no unknown type is inserted into a sealed graph.
The resulting concrete callback input then determines the receiver element type.
Conflicting near captures, result/accumulator conflicts and unconstrained generic
inputs still fail. Nominal methods sharing an array operation's spelling continue
through ordinary method resolution, including dynamic variadic spreads.

Regression evidence includes:
- Direct map/concat/append/slice literal spreads; nested/empty spread composition.
- Generic `[].map(callable identity)`, `[].fold(0, callable generic_add)`, and
  a generic map callback passed through a literal spread.
- A fixture-backed spread operand that can consume its single row only once.
- Negative arity, heterogeneous spread, runtime-length spread, inconsistent fold,
  callback arity, near/result conflict, and unresolved generic input checks.
- A nominal variadic `map` method to guard against overbroad intrinsic admission.

The correction was tested in an isolated `b624246` snapshot containing only these
changes, excluding the uncommitted I23 exact-amount implementation. Full
`go test ./compiler/... ./tests/integration -count=1` passed with qualified Bun,
the pinned offline archive and strict generated-TypeScript checks. The final
nominal-method guard was followed by the targeted checker and staged array gates, which passed. The staged array
suites now execute 72 assertion roots, run both programs and strictly type-check
all generated artifacts.
