# Current scalar catalogue

The maintained scalar example is [current/src/main.can](current/src/main.can),
a standalone current-language project with 40 mandatory assertions covering all
16 I22 numeric/conversion operations, error bounds, callable references and
fixtures. Run it with the staged `canlc assert std/scalars/current` or
`canlc run std/scalars/current` command.

Arithmetic and comparisons use native bigint/binary64 operations. Explicit
conversions and formatting use the closed `number`/`text` catalogue and
`runtime/number.ts`; there is no decimal base type or recursive conversion kernel.
The current compiler rejects the obsolete syntax and does not execute its Go
arithmetic evaluator. Exact-amount adapters are the separate I23 task.

The adjacent old `scalars.can`, generated `scalars.ts` and `errors.json` are
superseded historical test inputs, scheduled with their old consumers for the
I43/I44 inventory retirement. They are not the current library or a fallback.

## Historical scalar implementation

# std-scalars — monomorphic scalar standard library

- `scalars.can` — `mod scalars`: boolean logic, three-way
  comparisons (`-1`/`0`/`1`, false before true), selection of
  already-computed values, int/dec predicates, int/dec
  fundamentals (abs, negate, sign, min, max, clamp, distance,
  square, power, bounded add/subtract/multiply, factorial, sum),
  decimal observation (`std__dec__parts` over the `dec__parts`
  kernel: coefficient plus scale, total) and its first consumers
  (`std__dec__truncate`, `std__dec__floor`, `std__dec__ceil` over a
  digit-stripping worker: total in, total out; `std__dec__divide_exact`
  over gcd plus 2/5-factor stripping: exact quotient or explicit
  refusal, with two dec-payload error kinds; witnessed half-even
  rounding, exact division with remainder, decimal rendering,
  exact int conversion, and exact square roots), the pure NOW
  conversions between bool, int, dec, and str, and a bounded
  backoff delay with
  zero, negative, saturated, and large-attempt cases. Every function
  returns a named success record; fallible recursion splits into a
  total `decreases` helper plus a checked entry, so no error arm is
  unreachable. String order is byte order, documented in the file.
- `scalars.ts` + `errors.json` — committed golden TS prod emit
  (tests stripped). Regenerate: `go run ./compiler --out
  std/scalars std/scalars/scalars.can`;
  verify: `go test ./...`.

Rules: `/REQUIREMENTS.md`. Program: `docs/a/a13-stdlib.md` (row 2).
Names follow the brief verbatim (issue 5).
