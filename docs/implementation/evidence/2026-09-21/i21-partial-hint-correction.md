# I21 partial callback hints and concrete intrinsic references

Review of `480c744` found that partial callback hints were reaching the concrete
array-reference checker unchanged. An absent input equality overwrote the known
input of bound concat, and bound map could dereference a nil callback type.

Bound concat now retains its concrete captured array input when the hint has no
input equality. Append fills missing input evidence from its known array element
contract, including explicit type arguments. These operations do not guess an
error bound or fabricate a type. Other unresolved inputs produce a compiler
diagnostic before concrete contract checking. The contract checker also rejects
missing/unsealed argument types defensively.

Regressions cover `[].map(callable ([1]).concat)`, the invalid unresolved bound
map counterpart (which must reject without panicking), and
`[].fold([], callable append<int>)`. Existing conflict, ambiguity and spread
rejections remain covered. Checker and emitter suites passed.

The correction is validated in an isolated `243561b` snapshot containing only
these changes; preliminary I24 text runtime work is excluded. Full `go test ./compiler/... ./tests/integration -count=1` passed with
qualified Bun, the pinned local archive and strict generated-TypeScript checks.
The staged array suites now pass 74 assertion roots and both main programs.
