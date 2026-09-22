# a17 — Exact integer division (part)

Status: landed (part). `/` and `%` divide integers exactly under
the Euclidean contract; decimal division stays refused; the
remainder of the NOW-dagger family waits on fuel patterns.

## Rule

`/` and `%` share `*` precedence and associate left. Same-type
`int` operands only: the quotient and remainder satisfy
`dividend = divisor * quotient + remainder` with
`0 <= remainder < abs(divisor)` on every sign combination. A zero
divisor is loud (`int division by zero` in proofs; an explicit
`math.zero_divisor` outcome in blessed functions), never a value.

Decimal `/` and `%` are refused with CAN6005: 1/3 does not
terminate, so no rounding rule is smuggled in. Mixed operands
stay refused under the no-conversion rule. `-` and `*` are
untouched.

## Kernels, both verified

- Proofs run on `big.Int.DivMod` (Euclidean by construction),
  checked against the identity plus bounds on every sign
  combination — not on unit-step scans. The NOW-dagger label
  retires for the five blessed functions below.
- Emit rides `$canDivMod` (BigInt truncates toward zero, so a
  negative truncated remainder adjusts into range), emitted
  inline only when used. The helper text is pinned by
  `TestDivEmitHelper` and the Go-side contract by
  `TestDivModVectors` (identity plus bounds on every sign
  combination). Executing the emitted helper against the contract
  awaits a node gate (tsc verification stays suspended per #11).

## What ships

`std/division/division.can` blesses `std__int__divmod`,
`std__int__mod`, `std__int__is_multiple`, `std__int__is_even`,
and `std__int__is_odd`, each with all-sign decision tables plus
zero cases. The tables pin exact quotients and remainders, so
the Euclidean guarantee is test data, not prose. Even and odd
are one-liners over `%` — no recursion, no fuel.

## Consequences

- The a06 deferral is discharged for integers and recorded as
  still standing for decimals. `TestDiagnoseDivisionDeferred`
  now asserts the current boundary instead of the old one.
- A latent hole closed along the way: `decArith`'s `default`
  branch silently multiplied for unknown ops. It is now explicit
  `*` with a loud `default` — exact or loud, no third option.
- `std/division/` freezes as a golden like the other std modules.

## Still scheduled (not silently dropped)

`gcd`, `lcm`, `sqrt_floor`, `next_power_of_two`, and primality
need fuel-pattern recursion: the `decreases` rule admits only
exact `p - 1` steps, and a Euclidean mod step is not one. The
fuel needs a proven bound per call site (the backoff cap in row
2 is the precedent for separating termination from cost). Issue
4 stays open until that pattern lands with its own proof.

## Open decisions (do not block)

- Whether a unary minus after operators (`7 / -3` without
  parens) is worth a grammar change, or the paren form stands.
- Collation-style questions do not arise here; remainders are
  arithmetic, not order.
