# Row 2 — Fault-contract conformance (partial core, executable checks)

Status: shipped. The partial-core amendment (a20) is now backed by
executable checks on both runtimes plus an inventory of the
shipped wrappers' guard arguments.

## Contract

Three categories stay distinct: typed error outcomes (in `emits`,
returned as values), primitive domain faults (loud, in the
`/`-by-zero class — never silent, never typed), and resource
failures (production stack). `emits E` bounds returned
language-error outcomes only. Tables exercise their inputs; they
do not establish partial-operation domains for future calls.

## Evidence

- **Evaluator.** `TestTextOpsEval` pins seven loud shapes:
  index equal to length, negative index, inverted bounds,
  endpoints beyond length, negative bound, and huge
  beyond-`int64` index and slice bound (added this row — the
  verdict's list had no huge case). Valid values pinned
  alongside, astral plane included.
- **Emit (committed pins).** `TestStrFaultEmit` compiles a
  minimal operator-using module and asserts both throw contracts
  are emitted: `str index out of range`, `str slice out of
  range`. Pins, not goldens: the contract is named even where
  goldens rotate.
- **Emitted runtime (one-time probe, not gated).**
  `/tmp/fault-probe.mjs` executes the exact shipped helper text
  from `std/text/text.ts` under node: 7/7 invalid domains throw
  the exact messages, 4/4 valid cases agree with the
  evaluator-pinned values (`19990`, `119070`, count `7`,
  `"éll"`). Target execution stays outside the repo gates
  (issue #11 stands); the pins above plus this observed run are
  the evidence, and the disclosure, not a silent substitution.

## Wrapper inventory (bounded — shipped text functions)

- `scalar_at`: gates `index >= 0` and `index < #value`, raises
  `text.index_out_of_range` otherwise. Guard argument retained.
- `slice_scalars`: gates `start >= 0`, `end >= start`,
  `end <= #value`, raises `text.invalid_slice` otherwise. Guard
  argument retained.
- `find`/`contains`/`starts_with`/`ends_with`: delegate to slice
  compares internally; no new domains, no new faults.
- No stronger totality claim on these wrappers was found needing
  retraction: each total path already carries its guard.

## Still scheduled

Row 3 (D1 reporting/safeguards), row 4 conditional repairs, row 5
NUL policy, then a28. General site-level domain analysis stays
deferred as ordered.
