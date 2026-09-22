# Current division catalogue

Use `number::divmod` for native truncation toward zero and
`number::euclidean_divmod` for a nonnegative remainder. Both return the ordinary
`number::division` record and emit `number::zero_divisor` for a zero denominator.
The maintained example and sign/tie assertions are in
[the current exact-amount project](../ratio/current/src/main.can).

Adjacent old sources/generated files are historical test inputs scheduled for
I43/I44 retirement, not a current compiler fallback.

## Historical division implementation

# division — exact Euclidean integer division

- `division.can` — `mod division`: `std__int__divmod`,
  `std__int__mod`, `std__int__is_multiple`, `std__int__is_even`,
  and `std__int__is_odd`, each with all-sign decision tables plus
  zero cases. Every input satisfies dividend = divisor × quotient
  + remainder with 0 ≤ remainder < abs(divisor); a zero divisor
  fails loud. The remainder is always non-negative, so oddness
  tests the false arm, never `value % 2 == 1`.
- `division.ts` + `errors.json` — committed golden TS prod emit
  (tests stripped; the `$canDivMod` helper emits inline).
  Regenerate: `go run ./compiler --out std/division
  std/division/division.can`; verify: `go test ./...`.

Rules: `/REQUIREMENTS.md`. Program: `docs/a/a17-division.md`.
`gcd`, `lcm`, roots, and primality wait on fuel-pattern
recursion (issue 4 stays open).
