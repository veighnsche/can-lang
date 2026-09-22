# a24 — Row 4 remainder (rounding, rendering, exact int, exact root)

Status: shipped. Row 4 of the standard-library program closes with
five operations over the a21 observation primitive. No new kernel,
no new runtime helpers. `ratio__to_dec_exact` stays out: v0 brands
are string-backed only.

## What lands

- `std__dec__round_half_even(value, scale)` → `Dec__Rounded(value,
  discarded)`; ties go to the even neighbor, symmetric around
  zero; the witness identity `value = rounded + discarded` holds
  on every row. Emits `math.negative_scale`.
- `std__dec__divide_round_half_even(dividend, divisor, scale)` →
  `Dec__RoundedQuotient(value, remainder)` with the exact witness
  `dividend = divisor·value + remainder`. Emits
  `math.dec_zero_divisor` and `math.negative_scale`; the
  divisor check precedes the scale check (param order, pinned by
  the both-bad row).
- `std__convert__dec_to_str(value)` → canonical digits via
  `int_to_str` plus text slicing; no host-number parser.
- `std__convert__dec_to_int_exact(value)` → the integer or
  `convert.fractional_value`, reusing the a22 strip worker.
- `std__dec__sqrt_exact(value)` → the exact root or
  `math.dec_negative_input` / `math.nonrepresentable_result`,
  via scale parity plus the `sqrt_floor_search` worker called
  directly (total — the checked entry's error is never reachable
  and therefore never relayed).

## Name mapping

The brief's `RoundedDecimal` / `RoundedQuotient` violate R3
(`Domain__Name` is grammar-enforced, and the compiler rejects
`RoundedDecimal` outright). They land as `Dec__Rounded` and
`Dec__RoundedQuotient`. Rules beat wishlists.

## Totality budget

Every composition is total, so the only arms are the declared
refusals — all witnessed. Two consequences worth recording:

- The `int__pow` / `sqrt_floor` *entries* are never called from
  this slice, only their total workers; calling an entry would
  force an unreachable error arm (a22 rule).
- Int-valued conversions loop once per unit of the value, so
  tables stay under the 1024 local-call backstop (`s2s_large`
  stops at `d"999.99"`; the `i2s_thousand` row already skirts
  the same ceiling). Large-value rendering awaits an efficient
  kernel — the standing NOW† note, unchanged.

## Proof costs

- No new recursion: one new unit worker nowhere (all loops
  reuse `pow_from`, `strip_from`, `strip257_from`,
  `sqrt_floor_search`); the only new looping shape is none.
- Duplication follows the bounded-arithmetic precedent: the
  four rounding tails spell out their quotient expression
  rather than threading it through a shared helper, except
  `divide_round_half_even_result`, whose sign-plus-remainder
  tail is big enough to earn its own tested function.
- No linkage interaction, no effects, no kernel changes, no TS
  helpers.

## Implementation

- `std/scalars/`: three error kinds, two record types, six
  functions (52 new table rows); goldens regenerated.
- No compiler changes; no new Go tests (no new Go code path
  exists — the a21 kernel pins plus the tables plus the golden
  cover the slice).

## Row 4 is done

Representation (`parts`, `from_parts`, `scale_by_power_of_ten`),
directional rounding (`truncate`, `floor`, `ceil`,
`round_half_even`, `divide_round_half_even`), exact division,
exact int conversion, decimal rendering, and exact roots are all
blessed with decision tables and frozen goldens. The invoice /
ledger gate from the program doc is now writable in pure `.can`.
`ratio__to_dec_exact` and certified-bound transcendental
operations remain future work with their blockers named.
