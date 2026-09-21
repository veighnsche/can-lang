# Concrete generics

The current compiler checks one concrete function body per resolved declaration and ordered list of concrete type identities. Calls, callable references and assertion rows share that instance across source modules. Instances remain in their declaration's emitted module, with ordinary ESM imports connecting callers. Recursive nominal data retains its canonical identity.

Inference solves exact equalities from arguments, receiver types, expected results and expected callable contracts. References also constrain parameters from exact-name `near` captures. Generic constructors infer from fields and available expected nominal types. A unique matching leaf of an expected variant can supply constructor constraints; multiple possible leaves do not trigger an instantiation search. Deferred expressions such as empty arrays are retried after another constraint determines their expected type. An unresolved parameter produces an ambiguity diagnostic requiring explicit types.

Every assertion row infers its own concrete application, including types supplied by its expected success expression. Those instances are checked even if application code never calls them. Every discovered instance checks its entire body, including branches not exercised by assertions. This does not establish correctness for all possible type arguments or prove termination.

Same-instance recursion reuses the cache. Structural growth of a repeated declaration's arguments is rejected as expanding polymorphic recursion. A separate finite discovery limit handles excessive discovery without claiming a termination proof. Recursive type graph admission retains the existing inhabitation and expansion checks. Body diagnostics identify the generic source and requesting application sites.

Body-discovered types are admitted in isolated graphs: concrete argument graphs are copied with their recursive edges, the new graph is validated and sealed, and only then is its immutable type inventory merged. A failed admission does not reopen or invalidate existing checked types. Type compatibility remains nominal and invariant; inference does not synthesize unions, numeric widening or error sets.

The legacy stamping implementation is outside the active CLI path and remains scheduled for deletion by I44. The current implementation lives in `compiler/internal/types/{infer,specialize}.go` and `compiler/internal/check/{infer,specialize,methods}.go`.
