# I46 shared contextual array-spread correction

Independent I24 review reproduced an older I46 admission gap:
`text::join(...[...[[]]], ",")` rejected an empty array despite the fixed call
input supplying the exact `str[]` construction context. Flat spreading worked.
The array-expression checker discarded context whenever an element was spread.

A spread operand that is a fresh array literal (possibly parenthesized) now
receives an array of the known element type. Recursion carries that evidence to
nested empty literals. Nonliteral operands keep their checked invariant type;
this does not add array covariance, loose placeholders or mixed-type widening.
The ordinary argument-preparation and once-only evaluation path is unchanged.

Regression coverage includes the exact text::join case, explicit and inferred
generic fixed-input spreads, and grouped nested array construction. The current
text project now has 36 assertion roots, including its generic helper row.
Negative mutations cover incompatible nested elements; a dedicated test rejects
an existing box<int>[] where selection[] is required through a nested spread.

The correction was isolated atop commit 014f193, excluding unfinished I26
transport changes, for full compiler and staged offline integration validation.

The isolated full gate passed with pinned archive and CAN_TSC configured:
`go test ./compiler/... ./tests/integration -count=1`. This includes all 36 text
assertion roots, main execution and strict generated TypeScript. After tightening
the invariance regression to require the precise expected-type mismatch diagnostic,
the focused text/invariance checks passed again. No runtime changes are part of
this correction.
