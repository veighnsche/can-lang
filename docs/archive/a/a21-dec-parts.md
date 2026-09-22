# a21 — Decimal observation (dec__parts)

Status: shipped. Resolves issue #46 on the observation side: one
blessed `dec -> int` eliminator, with proof costs, before a single
Numeric consumer lands. The ten blocked functions stay scheduled,
not silently dropped.

## Rule

One reserved kernel call, one spelling:

- `dec__parts(value)` takes exactly one positional `dec` operand and
  yields `Ok(coefficient, scale)`: the signed integer denoted by the
  canonical digits and the fractional digit count.
- `d"12.34"` observes as `(1234, 2)`; `d"7.0"` as `(70, 1)`;
  `d"0.00"` and `d"-0.0"` as `(0, 1)`; the sign rides on the
  coefficient, never on the scale.
- `std__dec__parts` (in `std/scalars/`) wraps the kernel into the
  `Dec__Parts` record. It emits nothing: the observation is total.

## Why a reserved call, not an operator

a16 foresaw exactly this fork: a reserved pure-builtin callee kind
(precedent: `state__get` / `state__put`) or new operator spellings.
a20 chose operators for scalar access — but every a20 operator yields
a scalar (`int` / `str`), and parts yields a pair. A record-valued
operator would need its own construction rule on top of the spelling;
the reserved call reuses the match-scrutinee rule unchanged, needs no
`uses` entry, takes no `given` table, and needs no effects. One
spelling per meaning, same as ever.

## Type rule

The kernel bypasses callee resolution with a dedicated rule
(`checkDecParts`): positional args only, exactly one arg, `dec`
operand. Anything else is `CAN6003`, the operand-rule family. The
`Ok` arm binds a synthetic `parts` shape whose only fields are int
`coefficient` and `scale` — fixed by the kernel, never inferred, so
no declared record is consulted and no stdlib type name leaks into
the compiler.

## Failure model: total, never partial

Every `dec` value is canonical by construction (literals
canonicalize at parse, arithmetic re-normalizes, `-0` folds to
`0.0`), so coefficient and scale are always defined: there is no
missing case and no failure mode. Totality rests on canonical
input plus the terminating digit-reading algorithm; the
exhaustiveness gate proves consumers handle the declared outcome
mechanically — a kernel match wants exactly `{ok}`, so an error
arm is stale and an error-only match misses `ok` (both pinned). Static
misuse is `CAN6003`; past the gate the evaluator never fails. The
`from_parts` round trip holds by the digit reading: `from_parts`
of observed parts rebuilds the same canonical digits, which is why
the reading is the canonical digits rather than a minimal scale
(`d"10.0"` observes as `(100, 1)`, not `(1, 0)`). The identity
runs one way only: observed parts always rebuild through
`from_parts`, but `from_parts` accepts pairs no observation
produces: `parts(from_parts(0, 5))` is `(0, 1)`, not `(0, 5)` —
the constructor is permissive (any non-negative scale), the
observer normalizes (zero always reads scale 1, per the shipped
`parts_zero` row). (Correction: an earlier revision wrote
`(1000, 1)` here, contradicting the round-trip two sentences up.)
Totality itself rests on canonical input plus the terminating
digit-reading algorithm; the exhaustiveness gate proves consumers
handle the declared outcome — it composes with totality, it does
not establish it.

## Parity boundary

Both runtimes read stored digits: Go through `parseDecParts`, TS
through `$canDecParts` over `$canDecSplit` / `$canDecMant` (shared
plumbing, no new semantics). Canonical inputs — the only values the
language can produce — agree exactly, including negatives, zero,
and large scales. Executing the emitted helper against the contract
awaits the node gate (tsc verification stays suspended per #11);
until then parity rests on the mirrored code path plus the golden.

## Proof costs

- No new recursion: the kernel is a strict total observation,
  admissible as a match scrutinee anywhere a call is.
- No linkage interaction: the kernel is language-level, like the
  store ops, so every file uses it same-file with no `uses` pin.
- No effects: observation reads a value, not a cell, so no
  capability is declared or threaded.
- TS emit: one `$canDecParts` helper, emitted only when used,
  beside the existing dec runtime.

## Implementation

- `check.go`: `isDecParts` plus the three deterministic-call
  skips (unknown-callee, given-table, script-consistency).
- `types.go`: `checkDecParts`, `nodePartsArms`, the `parts`
  marker in `resolveRef`.
- `eval.go`: `evDecPartsOp` beside `evStoreOp`.
- `emit.go`: `stmtDecParts` plus the `parts` entry in
  `decRuntimeOps`.
- Tests: `TestDecPartsKernel` (vectors incl. zero, negatives,
  trailing digits, big values), `TestDecPartsKernelStoredDigits`
  (stored-digit reading), `TestDecPartsKernelLoud` (direct-caller
  faults), `TestDiagnoseDecPartsMisuse` (four `CAN6003`/given
  shapes), `TestDiagnoseDecPartsTotal` (stale arm, missing `ok`),
  `TestDecPartsEmitHelper` (helper present iff used), and the
  eight-row `std__dec__parts` decision table in `std/scalars/`.
- `std/scalars/`: `Dec__Parts` type plus `std__dec__parts`;
  goldens regenerated, `errors.json` unchanged (no new errors).

## Still scheduled (not silently dropped)

The #46 family spins off from here, one operation at a time, in
this order: the truncate/floor/ceil family first (parts plus
existing int arithmetic, smallest proof burden), then exact
division (2/5-factor removal with termination bound), then decimal
rendering (via `int_to_str` plus text slicing). `dec_to_int_exact`
rides the same observation. `ratio__to_dec_exact` additionally
waits on record-backed brands (v0 brands are string-backed only).

## Open decisions (do not block)

- Whether a second eliminator (e.g. direct scale query) ever earns
  its spelling, or parts stays the single observation.
- Whether the kernel's positional-only rule should generalize to a
  stated convention for future reserved calls.
