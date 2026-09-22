# a22 — Decimal truncation family (truncate, floor, ceil)

Status: shipped. First consumers of the a21 observation primitive:
three total functions over parts plus integer arithmetic, with no
new kernel, no new errors, and no new runtime helpers.

## Rule

- `std__dec__strip_from(remaining, current, exact)` is the one
  worker: a unit `decreases remaining` loop dividing out one factor
  of ten per step (Euclidean `/` and `%`, so the stripped value is
  the floor at every step) while threading whether every dropped
  digit was zero. Returns `Dec__Strip(stripped, exact)`.
- `std__dec__floor` returns the stripped value: Euclidean stripping
  is flooring by construction.
- `std__dec__truncate` returns the stripped value when exact or
  non-negative, else stripped plus one (toward zero).
- `std__dec__ceil` returns the stripped value when exact, else
  stripped plus one.
- Results rebuild through `std__convert__int_to_dec`, so every
  result is integral-valued (`d"2.0"`, never `d"2"`).

## Why a strip worker, not pow plus divmod

The obvious shape — `int__pow(10, scale)` then one `divmod` —
dies at the coverage gate. `int__pow` emits
`math.negative_exponent`, and the kernel guarantees a non-negative
scale, so the error arm would be statically unreachable; the
test-per-arm law (`checkCoverage`) makes untaken arms compile
errors, not warnings. The relay certificate would compile it, but
only by laundering an impossible error into `truncate`'s emits for
every caller to handle forever. The strip worker keeps the slice
total in, total out: every arm is witnessed by a table row, and no
emits entry is declared anywhere in the slice. Totality here is not
a style preference; it is the only shape the gate admits.

## Contracts

`floor(d"2.7")` is `d"2.0"`; `floor(d"-2.7")` is `d"-3.0"`;
`truncate(d"-2.7")` is `d"-2.0"`; `truncate(d"-0.5")` is `d"0.0"`;
`ceil(d"2.7")` is `d"3.0"`; `ceil(d"-0.5")` is `d"0.0"`. Exact
inputs pass through (`ceil(d"2.0")` is `d"2.0"`); zero stays
`d"0.0"` on all three. The worker's degenerate entries land on the
base arm under standard unit-loop semantics; kernel scales are
never negative, so the base arm doubles as the bound.

## Proof costs

- No new recursion shape: one unit loop with two evolving
  accumulators, the `binomial_from` precedent.
- No linkage interaction: same-file locals, no `uses` entries.
- No effects, no emits, no kernel changes, no TS helpers (the
  worker rides the existing `$canDivMod`).

## Implementation

- `std/scalars/`: `Dec__Strip` type plus the worker and three
  entries (29 new table rows); goldens regenerated,
  `errors.json` unchanged.
- No compiler changes; no new Go tests (no new Go code path
  exists — the kernel pins from a21 plus the tables plus the
  golden cover the slice).

## Still scheduled (not silently dropped)

Exact division next (`divide_exact` via 2/5-factor removal from
the divisor coefficient, with its termination bound), then
`divide_round_half_even` and `round_half_even`, then decimal
rendering. Each spins off separately, one operation at a time.

## Open decisions (do not block)

- Whether the strip worker stays public blessed surface or a
  future visibility rule lets helpers stay module-private; today
  every `_from` worker ships in `provides`, so this one does too.
