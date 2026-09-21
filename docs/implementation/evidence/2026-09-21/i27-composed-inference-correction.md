# I27 correction — composed inferred arm spreads

The first inference correction still asked declaration-shape discovery to validate
argument expressions whose types were already known. That made three valid
compositions diverge from ordinary Choice: a context-dependent nested empty-array
call, a computed nongeneric argument, and an inferred generic constructor.

Shape discovery now requests evidence only for unresolved type parameters. Once
an input pattern is fully resolved, the ordinary sealed checker supplies its
context and validates the expression. This avoids growing a second expression
checker for arithmetic, nested calls and other irrelevant shape evidence.
Constructor fields supply the same finite argument constraints used for functions;
the resolved concrete constructor shape can flow through inferred calls and
receiver methods. No candidate search or compatibility fallback is introduced.

This continues the private shape/public sealed-admission separation selected in
`i27-inference-jev`; that consultation is advisory, while the tests below verify
the corrected behavior.

Evidence:

- Before repair, all three new positive regressions failed with ambiguity or
  unsupported-shape diagnostics. Their ordinary Choice counterparts were admitted
  in the independent review.
- After repair, the matrix passes inferred/explicit calls, nested calls, array
  arguments, contextual empty arrays, variadics, receiver inference, nested empty
  calls, computed fixed inputs, inferred constructors and their combined form.
- Invalid nongeneric arguments, conflicting already-resolved arguments, invalid
  nested arguments, ambiguous parameters, arity mismatch and cycles still reject.
  Public inference still rejects unsealed evidence without mutating bindings.
- The maintained staged mixed fixture combines all three corrections in both
  ordinary and generated-record spreads. It preserves runtime arm selections,
  descriptor preparation, provider validation and handler ordering.

I28 drafts remain parked separately; this commit contains only the I27 correction.

The first staged run exposed a helper/question name collision in the new fixture;
renaming the helper corrected the fixture. The targeted staged mixed test then
passed (6.768 s), and the final full compiler/integration gate passed with pinned
archive and CAN_TSC: `go test ./compiler/... ./tests/integration -count=1`
(integration: 79.154 s). This includes strict generated TypeScript and offline
raw-provider execution with the combined inferred expression.
