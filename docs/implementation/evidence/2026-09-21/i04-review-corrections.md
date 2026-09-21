# I04 independent review corrections

Two independently reproduced parser issues were corrected on 2026-09-21.

Q6 states that generic arguments do not appear in error patterns. Completion
patterns now parse a qualified name directly and diagnose a following type
application. Regression tests cover ordinary and qualified errors, the bare
`all_failed` race arm, and participant-local concurrent handlers. Removing the
type arguments from each negative fixture gives a valid round-tripping program.
Generic arguments remain supported in type annotations and declared bounds.

Formatting a numeric receiver followed by field access previously removed the
lexical separator, so `1 /* separator */ .value` became invalid `1.value`.
Formatting preserves a space before the dot for numeric literal receivers.
This deliberately preserves syntactically valid diagnostic fixtures, even though
the later type checker must reject the nonexistent field. Integer, fractional,
exponent, negative and chained-field cases pass structural AST round-trip checks.
No extra grouping node is introduced.

Evidence:

- `compiler/internal/syntax/review_regression_test.go` reproduces both failures.
- [Full Go suite](i04-review-go-tests.txt): passed.
- [Pinned staged offline integration](i04-review-offline-integration.txt): passed.

These are direct corrections to explicit grammar/rendering contracts and required
no new language or architectural decision.
