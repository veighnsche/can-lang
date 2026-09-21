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

## Error contracts

`ErrorDeclarations` connects the I05 manifest/source/lock-validated allocation
registry to canonical resolver declarations, including catalogue reserved IDs.
`Bound` accepts sealed concrete errors only; `CheckEscaping` compares exact
specializations, without widening or inference. The owning I10 region checker
will compute which errors escape handlers. Bare error patterns resolve only when
one concrete specialization exists for that declaration.

`Plan` serializes the sealed graph for the private domain runtime. Generated
catalogue descriptors validate catalogue payloads against the same inventory,
including generic arguments and nested data. Standard failures remain opaque and
outside domain bounds. Expression checking exposes only their three maintained
projections, lowered through private runtime adapters. Native cause and source
origin remain diagnostic-only metadata, not authored fields.

## Initialization and finite locals

`Initialization` consumes resolved top-level declarations and their checked type
contexts. It first rejects anything outside C8's closed initializer AST set,
checks expressions, then orders references by dependency and qualified name.
Names must identify another supplied top-level value or an explicitly admitted
capture-free named arm. Storing an arm does not execute it. The emitter runs the
ordered native statements before entry, preserving startup failure source spans.

`CheckLocalForwarding` implements C8's four-predicate diagnostic only. The owning
body pass supplies the final binding identity/type, the initializer's lexical
checker, terminal expected type and complete identity/capture records. Missing
records are errors. The pass counts source uses across the block, applies the
finite expression filter, rechecks substitution and compares concrete typing
including nested nodes. It never guesses purity, removes code, or assumes absent
capture evidence means no capture. It returns a source-spanned diagnostic with
the direct replacement only when every predicate holds.
