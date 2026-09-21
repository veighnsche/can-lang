# Current expression checking

`Expressions.Check` consumes the current AST and sealed I06 type evidence. The
owning body pass provides eligible-kind value/function lookup and concrete nominal
constructor resolution; this module never reads predecessor compiler shapes or
executes a source expression to determine its type.

The I07 pass handles primitive literals/operators, comparison chains, array
construction/spread, ordinary record construction/copy-update/fields, length,
indexing, slicing and the catalogue array/string slice method. Integer and float
operands stay distinct, conditions require bool, and unreachable operands still
must typecheck. Equality requires eligible identical data types or a named variant
already admitting both operands; it never searches for an implicit union.
Existing arrays remain invariant, while expected array construction may admit
individual variant leaves. Empty arrays need an expected element type.

Resolved empty-bound callable contracts can occur as expression operands. A call
with domain errors is rejected here rather than being silently unwrapped. The
owning call/region pass supplies full methods, captures, generics, spread/state
argument handling and completion checks in subsequent tasks. Unsupported nodes
return a diagnostic; they never fall back to the predecessor pipeline.

Checking produces typed operations with original source spans and explicit
comparison equality modes. Runtime faults from bounds/integer arithmetic are
standard failures, not additions to authored domain emits. The private primitive
runtime keeps fixed kind/message observations by thrown-object identity for the
I49 failure boundary; native capacity exceptions remain unclassified native
exceptions for that boundary.
