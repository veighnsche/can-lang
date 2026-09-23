# Finite language rules — 2026-09-22

Status: current. This document replaces absolute "no inference/coercion"
wording with the actual finite rules, and "JSON Schema validated" with the
actual compiler validation mechanism (LD47/AE47). Every rule links to an
executable example in `examples/2026-09-22/lf19/cases/`; run them with
`run.py --bundle <dev-bundle> --out results.json`. Observed outputs below were
recorded under the pinned runtime, Bun 1.4.2 (`bun-1.4.2-darwin-arm64-v1`).

The finite generic-inference engine is described in full in
[generics.md](generics.md): exact equalities solved from arguments, receivers,
expected results, and expected callable contracts; deferred empty arrays
retried once another constraint fixes their type; ambiguity diagnostics
otherwise. What follows states each rule with its positive and negative
example.

## 1. Expected typing of empty literals

`[]` carries no element type of its own; the checker takes it from the
expected type, including through generic calls and nesting.

- Positive: `cases/empty-literal-expected` — `ok []` against declared `int[]`,
  `int[] out = []`, `[]` rows against `int[]` parameters, and
  `option::none()` against `option::value<int>`. Nested `[[]]` against
  `int[][]` also checks.
- Negative: `cases/unconstrained-empty` — `match []` has no expected type and
  fails with `empty array requires expected element type`.

## 2. Exact generic equality inference

One type variable, one equal type: every argument, receiver, and expected
result must agree exactly. Inference never widens numbers, synthesizes
unions, or searches for an instantiation.

- Positive: `cases/generic-inference` — `call identity(2)` infers `item =
  int`, `call identity("x")` infers `item = str`, and `call identity<str>("x")`
  pins the argument explicitly. Expected results constrain too: `call
  identity([])` against `int[]` checks with `item = int[]`.
- Negative: `cases/unresolved-generic` — `call pick(1, "x")` against `fn item
  pick<item>(item first, item second)` fails inside the generic call because
  `str` does not fit the `int` already fixed for `item`.

## 3. Leaf and narrower-variant inclusion

A value of a listed leaf is accepted in its named variant, and a narrower
variant enters a wider named variant when its complete leaf set is included
(`docs/syntax-taste/technical-spec.md` §C). Existing variant values do enter
wider variants; the replacement claim that they never do is false (LD47).

- Positive: `cases/leaf-inclusion` — `option::some(3)` where
  `option::value<int>` is expected; a `round` (one leaf) where `shape` (two
  leaves) is expected; matching with record-constructor and bare-leaf
  patterns.
- Negative: `cases/nominal-mismatch` — `point_a` where `point_b` is expected
  fails with `expression type does not fit expected type`. No general
  structural record subtyping exists.

## 4. Callable error-subset compatibility

A callable whose bound declares fewer errors is accepted where a wider bound
is expected; the call site still handles the full declared bound.

- Positive: `cases/callable-subset` — `callable total` (`emits []`) binds to
  `callable int (int) emits [codec::invalid_data]`, and the call handles the
  (unreachable) `codec::invalid_data` arm plus `ok`.

## 5. Container invariance

Arrays and generic constructors are invariant: element inclusion never lifts
to the container.

- Negative: `cases/container-covariance` — `option::some<int>[]` passed where
  `option::value<int>[]` is expected fails with `expression type does not fit
  expected type`, even though each element would include on its own.

## 6. Explicit numeric conversion

Operators require identical operand types (`operator + requires identical
operand types`); mixed `int`/`float` arithmetic without conversion is
rejected. Conversion is explicit through the `number` catalogue operations,
which declare their failure modes.

- Positive: `cases/numeric-conversion` — `int_to_float`/`float_to_int` with
  exact rows plus `number::inexact` rows (`9007199254740993` is not exactly
  representable; `1.5` is not a finite integer; `-0.0` converts to `0`), and
  total `bool_to_int`.
- Negative: `cases/mixed-arithmetic` — `whole + part` with `int` and `float`
  operands fails.

## 7. Surprising native cases (observed outputs, Bun 1.4.2)

| Case | Program | Observed |
| --- | --- | --- |
| Signed zero | `-0.0 is not 0.0` | `true` (`Object.is`) |
| NaN | `(0.0 / 0.0) is (0.0 / 0.0)` | `true` (`Object.is`) |
| Truncating division | `-7 / 2`, `7 / -2` | `-3`, `-3` (native bigint `/`) |
| Remainder signs | `-7 % 2` | `-1` (native bigint `%`) |
| Half-open slice | `"A😀B".slice(1, 3)` | `"😀"` (clamped native slice) |
| UTF-16 indexing | `"A😀B"[1] + "A😀B"[2]` | `"😀"` (two surrogates recombined) |
| Scalar projection | `text::scalars("A😀")` | `[65, 128512]` |
| Grapheme projection | `text::graphemes("A😀")` | `["A", "😀"]` |

Positives: `cases/numeric-edges`, `cases/string-edges` (all `real-can`
evidence). Integer division by zero is a standard `arithmetic` failure, not a
value (`runtime/primitive.ts`).

## 8. Native lowering

- Integers are native `bigint` with truncating `/` and `fail`-on-zero
  adapters (`runtime/primitive.ts`: `intDivide`, `intRemainder`, `intPower`).
- Scalar float equality is `Object.is`; other primitive equality is `===`;
  records/arrays/errors use `Bun.deepEquals(..., true)` over
  compiler-owned data (`compiler/internal/emit/expressions.go`).
- Slices/indexing are native with bounds adapters (`$canSlice`, `$canIndex`);
  string positions are UTF-16 units, with explicit `scalars`/`graphemes`/
  `normalize_nfc` operations for scalar and grapheme views.
- Explicit conversions live in `runtime/number.ts` (`intToFloat`,
  `floatToInt`, …) with exactness/finiteness/integral guards.

## 9. Configuration validation (compiler-side, not JSON Schema)

Project manifests, error registries, and SQL descriptors are validated by the
compiler's own checks (`compiler/internal/project/manifest.go`,
`json.go`): exact known keys, duplicate-JSON-key rejection, path
normalization with parent-traversal refusal, and per-section rules
(postgresql-only SQL, named parameters, fully qualified nominal types). No Can
runtime JSON-Schema subsystem exists or is claimed.

- Negatives: `cases/manifest-badkey` (`unknown field "bogus"`),
  `cases/manifest-duplicate` (`duplicate JSON key "source_root"`),
  `cases/manifest-escape` (`parent traversal is forbidden`).

## 10. Build versus assert (current behavior)

`assert` executes assertion rows in emitted code and never publishes:
passing, selected and failing runs leave production `current` untouched,
and selected runs report partial scope. `build` executes the same rows
as gate verification (bounded by `--assert-timeout-ms`, default 5000ms)
and publishes one complete verified generation only when every root in
the project and its locked dependencies passes; any failure, pending
timeout or missing native coverage blocks publication with no `current`
update (P15.1 verified build; `verified_build_test.go` G1/G2 gates).
Verification issues no live traffic and never executes live `main`, and
production generations ship no assertions tree or runner import.
Documentation must not present `build` as publishing unverified output.

## 11. Renderer and formatter (current behavior)

The single parser/renderer preserves owned trivia, and `canlc format`
provides validated source formatting (stdout by default, `--write` for
checked atomic replacement). The historical "no renderer"/"loses comments"
wording is superseded; see the LF18 evidence.

## 12. Superseded review quotes

The following historical review passages stay verbatim in
`docs/syntax-taste/design-review-2026-09-22.md` and are marked superseded
there; they must not be quoted as current behavior:

- F8: "(P2/P12) are JSON Schema validated by the compiler" — superseded by §9.
- F9: "no formatter in the current specs" — superseded by §11.
- F7 context: the "no inference" shorthand — superseded by §1–§2.

Cross-check (2026-09-22): `technical-spec.md`, `ai-io-spec.md`,
`coordination-spec.md`, `platform-testing-spec.md`, `decisions.md`,
`generics.md`, and `coverage.md` carry no contradicting absolute claims; every
remaining "no coercion" occurrence names a specific boundary (operators,
query strings, JSON codec) that the checker enforces.
