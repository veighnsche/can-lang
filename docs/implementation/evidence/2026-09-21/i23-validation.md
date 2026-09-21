# I23 — finite exact-amount adapters

The three C8 catalogue operations are admitted through the existing intrinsic
checker and emitted through `createExactAmounts` in `runtime/number.ts`:

- `number::divmod` uses native bigint division/remainder with truncation toward
  zero and the dividend's remainder sign.
- `number::euclidean_divmod` adjusts only a negative native remainder and its
  quotient, preserving `numerator = quotient * denominator + remainder` and
  `0 <= remainder < abs(denominator)` for either denominator sign.
- `number::round_ratio_half_even` normalizes the denominator positive, performs
  native division/remainder, applies the specified finite nearest/even adjustment,
  and recomputes the exact remainder numerator. It does not reduce the fraction.

Zero denominators produce the existing allocated domain error
`number::zero_divisor()` (1009) before native division. Outputs are frozen ordinary
catalogue `number::division` and `number::rounded` records with compiler-supplied
sealed identities. Existing direct-call, callable-reference, completion and
error-bound machinery is reused. No Number conversion, decimal base type,
recursive fuel computation, GCD algorithm or alternate arithmetic evaluator is
introduced.

## Acceptance evidence

Runtime tests independently characterize results across signed input grids:
truncating results agree with native `/` and `%`; Euclidean results satisfy the
exact reconstruction and remainder range; rounded results preserve the rational
by cross multiplication, minimize distance against both neighbors and choose an
even value at ties. Explicit vectors cover positive/negative tie-even/tie-odd,
non-ties and both denominator signs. Additional vectors cover a balance above
binary64's consecutive-integer range, 500-digit integers, and unreduced `2/4`.
Every zero-denominator operation checks the exact allocated error and payload
identity. Frozen output and nominal record identity are checked separately.

`std/ratio/current` supplies 23 mandatory current-language assertions covering all
three operations, their zero errors and a callable reference. Its account example
adds 2 to 9007199254740993 minor units, rounds half the resulting amount to
4503599627370498 with remainder -1/2, and encodes that record using the existing
integer-token JSON codec. Main executes this same path and verifies the encoded
integer digits. Compiler negatives reject float operands, an omitted zero-divisor
bound and the obsolete decimal type.

## Validation

- Exact-amount runtime suite: 5 tests / 17,595 expectations passed on qualified Bun.
- Strict TypeScript passed for the numeric runtime and exact-amount tests.
- Full runtime suite: 162 tests / 19,035 expectations passed.
- Staged exact-amount integration: all 23 assertions, main execution and strict
  generated-TypeScript checking passed. The absolute private Bun and launcher
  run with network denied, ambient Bun absent and an unrelated working directory.
- Full `go test ./compiler/... ./tests/integration -count=1` passed with qualified
  `CAN_BUN`, the pinned local `CAN_BUN_ARCHIVE`, and strict `CAN_TSC` enabled.

The current ratio project replaces the old examples as maintained evidence. The
ratio/division READMEs identify their adjacent obsolete sources/generated outputs
as historical inputs awaiting coordinated I43/I44 retirement; they are not active
CLI semantics or fallback implementations.
