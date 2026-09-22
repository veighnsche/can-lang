# a23 — Exact decimal division (divide_exact)

Status: shipped. The hardest #46 promise first: exact success or
explicit refusal, with non-termination detected rather than rounded
away. No new kernel, no new runtime helpers.

## Rule

- `std__dec__divide_exact(dividend, divisor)` returns the exact
  quotient or one of two errors: `math.dec_zero_divisor` on a zero
  divisor, `math.nonterminating_decimal` when the quotient does not
  terminate. `1/2` is `d"0.5"`; `1/3` is refused, never rounded.
- `std__dec__strip257_from(fuel, current, twos, fives)` is the one
  worker: a unit `decreases fuel` loop removing factors of two and
  five while counting them. Returns `Dec__Factors(rest, twos,
  fives)`; `rest == 1` means the denominator was of the form
  2^a·5^b.
- Method: observe both parts, scale coefficients to integers
  (`N = ca·10^sb`, `D = cb·10^sa`), reduce by `gcd`, strip the
  reduced denominator, and — only when it strips to one — rebuild
  with `m = max(twos, fives)`, `Q = N'·(10^m/D')`, result
  `Q·10^-m`. Sign applies last, at the dec level.

## Why this shape

Two refusals precede the method. A `pow`-shaped quotient
(`N/D` rounded to scale) would smuggle a rounding rule past the
a17 refusal of decimal `/`: exact or refusal, no third option.
And a `divmod`-first shape cannot see termination at all — the
integer remainder of scaled operands says nothing about whether
the decimal expansion ends. The 2/5-strip is the complete
termination test for base ten, and the counts it already returns
are exactly the exponent the rebuild needs. Nothing is computed
twice.

## Totality and the fuel bound

Every composition in the entry is total (`parts`, `pow_from`,
`abs`, `gcd`, `int_to_dec`, `scale_by_power_of_ten`), so the only
arms are the two refusals plus `Ok` — all witnessed by table
rows. Denominators are proven nonzero by construction (`cb ≠ 0`
checked; `g ≥ 1` and `D' ≥ 1` follow), and the entry divides by
them without checks, the `lcm` precedent. The worker's fuel is
`|D'|`: the strips needed are v₂+v₅, strictly less than `|D'|`
for every integer `D' ≥ 1`, so exhaustion is unreachable past
real inputs and surfaces as refusal, never a wrong quotient.

## Error contract

The brief's `[math.zero_divisor, ...]` predates the a13 payload
split: one kind, one field list, so the dec operands get their
own kinds — `math.dec_zero_divisor(divisor: dec)` and
`math.nonterminating_decimal(dividend: dec, divisor: dec)` —
beside the int `math.zero_divisor`, never reusing it. Both are
raised with complete payloads and hit by named rows.

## Proof costs

- No new recursion shape: one unit loop with evolving
  accumulators, the `binomial_from` precedent, nesting three
  matches deep under the fuel guard (accepted: guardedness carries
  through nesting).
- No linkage interaction, no effects, no kernel changes, no TS
  helpers (the worker rides `$canDivMod`).

## Implementation

- `std/scalars/`: two error kinds, the `Dec__Factors` type, the
  worker, and the entry (18 new table rows: exact, scaled,
  all-sign, zero, and three refusals); goldens regenerated.
- No compiler changes; no new Go tests (no new Go code path
  exists — the a21 kernel pins plus the tables plus the golden
  cover the slice).

## Still scheduled (not silently dropped)

`divide_round_half_even` and `round_half_even` (same observation,
witnessed rounding), then `dec_to_str` and `dec_to_int_exact`,
then `sqrt_exact` (scale parity plus `int__sqrt_floor`). Row 4
closes with decimal rendering, not here.

## Open decisions (do not block)

- Whether `ratio__to_dec_exact` waits on record-backed brands or
  earns its own construction argument first; v0 brands stay
  string-backed either way.
