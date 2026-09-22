# a19 — Blessed recursion schemas (spec)

Status: shipped. Amendment to a11 (unit loop): two more canonical
shapes are admitted, each pairing one step spelling with one guard
spelling, each with its theorem written down. The a11 rule is
unchanged: the compiler sees the decrease, it never infers one. No
existing program changes meaning; `decreases p` still means the unit
loop and nothing else.

## Why this exists

Slice 1 of the stdlib program proved the unit rule admits every
NOW-dagger integer algorithm — as a linear downward scan with a
shifted guard. Correct, but O(n): each one trips the 1024 local-call
depth backstop on ordinary inputs (issue #4: the backstop is not a
cost model). The efficient forms — Euclid's mod step, binary-search
halving — are not unit steps and could not be spelled. This amendment
blesses exactly two more shapes, no more.

## Rule

The `decreases` line has three spellings, one per meaning:

1. `decreases p` — unit loop (a11, unchanged): every self-call site
   passes exactly `p - 1` under the false arm of `p <= 0`.
2. `decreases a, b by euclid` — Euclidean step: every self-call site
   passes exactly `(b, a % b)` (named or positional) under the false
   arm of `b <= 0`.
3. `decreases lo, hi by narrowing` — binary search: every self-call
   site passes `(lo, mid)` or `(mid, hi)` with `mid` exactly
   `(lo + hi) / 2`, under the false arm of `(hi - lo) <= 1`.

Anything else on the line is `bad decreases line` at parse,
including unknown schema names (`by half`), one name with a schema,
and three names. Step violations are `CAN3008`, guard violations
`CAN3009` — the same code families as the unit loop, with
schema-aware messages, so one mistake still yields one diagnostic.

## Theorems (written down, not waved at)

**Euclid.** A site reached under a false `b <= 0` guard entered with
`b >= 1`. The language defines Euclidean `%` (total; remainder in
`[0, b)` for positive `b`), so the site's new second component lands
in `[0, b)` — a natural chain that strictly descends while positive
and reaches the base arm. Entries with `b <= 0` (including negative
`b`) take the base arm immediately. Every admitted invocation
returns a declared outcome. The proof leans on the `%` contract, not
on inferring `a % b < b` — the shape is blessed, the inequality
comes with the operator.

**Narrowing.** A site reached under a false `(hi - lo) <= 1` guard
entered with `hi - lo >= 2`, so `(lo + hi) / 2` sits strictly inside
(`lo < mid < hi` by exact integer division) and both admitted shapes
strictly shrink the bound gap. Chains reach gap `<= 1` and take the
base arm; degenerate entries (`hi <= lo`) take it immediately. Every
admitted invocation returns a declared outcome. Correctness of the
returned bound (the caller's invariant, e.g. `lo*lo <= value`) is the
caller's business, exactly as the unit loop never promised which
base value a chain lands on — only that it lands.

## Cost note (answers #41's #4 half)

Admitted-efficient and admitted-expensive stay distinguished
in-tree: `gcd` and `sqrt_floor` graduated to `euclid`/`narrowing`
(log steps); primality, binomial, and `next_power_of_two` remain
documented linear scans until a schema covers them. The 1024 depth
backstop is still a resource bound, not the proof.

## Implementation

- `parse.go`: `DecNames`/`DecSchema` replace the single name; two
  regexes, one spelling each; anything else is a parse error.
- `check.go`: `checkDecreases` dispatches on schema with one step
  predicate and one guard predicate each (`isEuclidStep`,
  `isNarrowStep`/`isMid`, `isNarrowGuard`); the guarded arm-walk is
  shared. Codes reused: `CAN3008`/`CAN3009`.
- `loop_test.go`: clean, bad-step, bad-guard, bad-line, and non-int
  pins per schema.
- `std/scalars`: `gcd_euclid` and `sqrt_floor_search` replace the
  linear workers; entries and contracts unchanged.
