# v0.6 — Arithmetic (spec)

Status: shipped. Expressiveness item 1+2: the smallest proof
cost on the a05 list. Three operators, exact everywhere, one deferral
with its reason written down.

## Rule

`+`, `-`, `*` over `int` and `dec`. Same-type operands only: `int`
with `int`, `dec` with `dec`. Mixed `int`/`dec` is `CAN6003`, like
every other conversion that does not exist. `str` and `bool` operands
are `CAN6003`. There is no unary minus on expressions: negation is
literal-only (`-3` parses as ever; `-(a+b)` does not). There is no
division (see below).

Precedence is conventional and total: `*` over `+` and `-`,
comparisons loosest (`a + b == c` parses as `(a+b) == c`).
`+`, `-`, `*` associate left (`10 - 3 - 2` is `(10-3)-2` = 5, never
`10-(3-2)` = 9). One spelling per meaning extends to grouping: there
are no other arithmetic forms.

## Eval semantics (exact or loud, never silent)

Proofs are exact. `dec` arithmetic runs on scaled integers
(mantissa/scale, `big.Int`), so `d"0.1" + d"0.2"` is `d"0.3"` —
the float-killer demo as a decision-table row. Addition and
subtraction align scales; multiplication adds them; results
re-normalize to canonical form (no trailing fractional zeros, `-0`
folds to `0.0`). Add/sub/mul of terminating decimals always
terminate, so rendering is total with no rounding rule.

`int` arithmetic is exact or loud: results that do not fit `int64`
are evaluation errors (`int overflow`), surfacing as test failures,
never silent wraps. Literals already cannot overflow (they parse
through `Atoi` at the boundary); only computation chains can, and
they fail closed.

> Amendment (a10): superseded — see `a10-numerics.md`. `int` is now
> mathematically unbounded (`*big.Int` literals, no overflow gate),
> so this paragraph's `int64` failure mode no longer exists.

The evaluator's dynamic rule stays dumb and unchanged in shape:
same-kind operands or `bad %s operands`. Static checking (`checkTypes`
binop rule, extended) fires first with `CAN6003` naming both types;
the dynamic error is unreachable past the gate, as before.

## Type rule (extends the a04 binop rule)

The existing rule — identical operand types, no conversions — already
covers arithmetic; only the result type changes. Comparisons yield
`bool`; `+`, `-`, `*` yield the operand type. The mismatch message
names the operator class (`cannot compare`, `cannot add`,
`cannot subtract`, `cannot multiply`) so agents do not file arithmetic
mistakes under comparison.

## TS emit (inside the documented boundary)

`+`, `-`, `*` emit as themselves; `dec` operands already emit as
`number`. The a04 boundary now covers both worlds explicitly: canlc
proofs are exact-or-loud (arbitrary-precision decimals, overflow
errors past `int64`); the TS mapping is exact for `|int| < 2^53` and
`dec` within 15 significant digits. A decimal/int64 runtime arrives
with its own proposal, not smuggled inside this one.

> Amendment (a10): superseded — see `a10-numerics.md`. `int` emits
> as `bigint` and `dec` as canonical-digit strings through exact
> `$canDec` helpers; the `number` mapping and both precision
> boundaries above are gone (the 15-digit `dec` claim was false even
> inside its stated domain: `0.1 + 0.2`). Remaining platform
> boundary: BigInt (ES2020+) and host `bigint` handling.

## Division deferred (the reason, written down)

`/` is not half an arithmetic feature. Terminating decimals are not
closed under division (`d"1" / d"3"` does not terminate), so any `/`
needs a rounding rule — truncation, banker's, floor — plus where the
rule is declared (per operation? per type? per module?) and how the
proof evaluates it exactly. Integer division has the same question
(truncation vs floor for negatives). That is a semantic proposal of
its own, not a footnote to this one. `a / b` today is a parse error
and stays one until the division spec lands.

## Implementation

- `parse.go`: `findLastTop` (last top-level occurrence, for
  left-associativity) beside `findTop`; `parseSmall` splits
  comparisons, then `+`/`-`, then `*`, before calls. String- and
  depth-awareness inherited from the existing scan.
- `eval.go`: `+`, `-`, `*` in the binop branch; `big.Int` int path
  with `IsInt64` gate; scaled-integer dec path with canonical
  rendering. `vEq` and ordering untouched (results are plain
  `int`/`dec` values).
- `types.go`: binop rule split by operator class; `typeOf` yields the
  operand type for arithmetic.
- `emit.go`: three operators added to the op map, nothing else.
- Grammar: `+`, `-`, `*` join the operator rule (before `->` stays
  first); `gramcheck` gains a sample.
- `modcheck`: untouched (expression-level; verified by running).

## Consequences (accepted before building)

- The flagship uses it honestly: `Auth__Session` gains
  `remaining_tries: int` computed as `3 - user.failed_attempts` at
  both `Ok` sites (non-negative by construction: the lockout arm
  catches `>= 3` first). `Auth__Session` 1 → 2, `auth__login` 3 → 4;
  `db` untouched, pins unchanged, no new names. `dec` is covered at
  unit level (no money in login;
  forcing currency into auth would be scope smuggled as demo).
- Goldens: `auth.ts` gains the computation, `errors.json`
  unchanged (no error change), normalize gains
  `remaining_tries = 3` on the two `Ok` rows. `db.ts` untouched.
- Tests: `lsp_test.go` gains mixed/str-operand mismatches with spans,
  a left-associativity decision table (`10 - 3 - 2 = 5`), and the
  `0.1 + 0.2 == 0.3` exactness row; a new `arith_test.go` pins
  overflow-loud and dec rendering at unit level.
- `broken-login/` keeps exactly its titled squiggles (verified, not
  assumed).

## Open decisions (do not block)

- Division spec (per above; its own number in this series).
- Whether `int64` overflow should be a static range error instead of
  a loud dynamic one (leans: dynamic; ranges need interval analysis,
  a proposal of its own).
  Amendment (a10): moot — overflow no longer exists (unbounded ints).
- Larger integer model (leans: no; `int64`-or-loud is honest, bignum
  emit waits for the runtime proposal).
  Amendment (a10): decided — unbounded ints with `bigint` emit
  (`a10-numerics.md`).
