# v1.0 — One numeric semantics (spec)

Status: shipped. Amendment to a04 (type discipline) and a06
(arithmetic): the evaluator/target split the audit found is closed.
There is now one numeric semantics, proved in the compiler and
preserved by the emit. No new source spelling; every existing
small-value program keeps its meaning.

## The hole (why this exists)

a06 proved `dec` on scaled integers and `int` as int64-or-loud, then
mapped both to TypeScript `number` with a documented boundary (exact
for `|int| < 2^53`, `dec` within 15 significant digits). The boundary
was documented, not enforced — and one of its claims was false:
`d"0.1" + d"0.2"` is `d"0.3"` in proofs but `0.30000000000000004` as
`number`, with one significant digit on each side. A documented
boundary is not enforcement of that boundary: either the target
preserves source arithmetic, or admissible programs must be
restricted. This spec takes the first option.

## Rule

`int` is mathematically unbounded: literals never overflow, `+`,
`-`, `*` never wrap and never fail loud. `dec` keeps its a06
scaled-integer semantics bit-for-bit (same canonicalization, same
add/sub-align and mul-sum-scales rules). Comparisons are unchanged
in meaning on both types.

## TS emit (exact, inside one stated platform boundary)

- `int` emits as `bigint`: literals gain the `n` suffix (`3` →
  `3n`), operators and ordering stay native — exact for every
  integer, including the old `int64` domain where `number` silently
  merged neighbors (`1000000000000000000 - 1` is distinct again).
- `dec` emits as a `string` carrying canonical digits. `==`/`!=`
  stay native (`===`/`!==` is exact on canonical strings);
  `+`, `-`, `*`, `>=`, `<=` route through emitted `$canDec`
  helpers that mirror `decArith`/`canonDec` exactly (align-to-wider
  scale, sum scales on multiply, renormalize, `-0` folds to `0.0`).
- Dispatch is static, never heuristic: the checker annotates every
  `ref`/`binop`/`seal` node with its resolved type, and emit reads
  the annotation. An unannotated arithmetic node fails loud instead
  of guessing.
- Helpers are emitted inline, only the used operations plus shared
  plumbing, in fixed order for byte-stable output. `$` prefixes are
  unspellable in can (`domain__verb` cannot start with `$`), so user
  code can never collide with them. Files without dec arithmetic
  gain no code.
- The one remaining platform boundary, stated: emitted code needs
  BigInt (ES2020+) and hosts must treat `bigint`/`dec`-string
  accordingly (notably, `JSON.stringify` rejects `bigint` without a
  replacer). That is a host-adapter contract, not a silent
  precision loss.

## Implementation

- `parse.go`: `Small.Num` is `*big.Int`; literals parse through
  `SetString`, so size is never a parse error.
- `eval.go`: `Value.N` is `*big.Int`; the `IsInt64` gate is deleted;
  comparisons, equality, and `normalize` all compare/render through
  `big.Int`. (a11 deletes the `decreases` negativity checks as
  unreachable past the guard rule.)
- `check.go`: the step rule compares unbounded literals (same
  syntactic shape; a11 tightens it to the unit step `p - 1`).
- `types.go`: the checker stamps `Small.T` on `ref`, `binop`, and
  `seal` nodes during the existing walk; nothing else changes.
- `emit.go`: `tsBase` and Ok-shape inference map `int`→`bigint`,
  `dec`→`string`; `emitValue` dispatches dec operations to helpers.
- Tests: `arith_test.go` inverts the old overflow test into exact
  big-value proofs; `numeric_emit_test.go` compiles a dec+int
  fixture and pins helper dispatch, literal shapes, and the absence
  of `number`-typed values. The `$canDec` helpers were additionally
  probed under node against a Python-`Fraction` oracle (17 cases:
  the float-killer, negatives, scale alignment, `-0` folding,
  30-digit values) — throwaway probe under `/tmp`, not a repo gate.

## Consequences

- Goldens: `auth.ts`, `db.ts`, `retry.ts`, `counter.ts` change
  `number`→`bigint` and gain `n` suffixes; `errors.json` and
  `normalize` output are byte-identical (no error or proof-value
  change). `broken-login/` titles unchanged.
- Breaks (stated plainly): every TS consumer of emitted
  types (fields are `bigint`/`string` now), host adapters and
  `.externs` stubs touching numeric fields, and any test asserting
  `int overflow`. Small-value can sources need no rewrite.
- a04's "15 significant digits" boundary and a06's "TS mapping is
  exact for `|int| < 2^53`" are superseded; both carry an
  `Amendment (a10)` pointer to this doc. The a05 plan baseline is
  left as history.

## Open decisions (do not block)

- Resource exhaustion from unbounded growth (a host-memory question)
  is modeled nowhere yet — same class as the a08 stack boundary:
  documented, not solved.
- Division stays deferred under its a06 reason; this spec changes
  nothing about `/`.
