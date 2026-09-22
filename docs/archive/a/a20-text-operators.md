# a20 — Scalar text operators (spec)

Status: shipped. Resolves the a16 open surface decision on the
operators side: three spellings, with proof costs, before a single
derived text function lands.

## Rule

Three operators over `str`, one spelling per meaning:

- `#s → int`: the number of Unicode scalar values in `s`.
- `s[i] → int`: the scalar VALUE (code point) at index `i`.
- `s[a:b] → str`: the half-open slice, scalars `a` through `b - 1`.

`#` is prefix and binds tightest (`#a + b` parses as `(#a) + b`
only where `+` typechecks — on strings it does not, so bare `#a +
b` is a type error, not a silent regrouping). `[i]` and `[a:b]`
are postfix at the same level and chain left (`s[i][j]` parses;
the checker refuses it). `:` appears only inside brackets; `a:b`
anywhere else is still a parse error, as are empty bounds
(`s[]`, `s[a:]`, `s[a:b:c]`).

## Why index yields int

The pair is closed, needing no bridge. `scalar_at`'s contract
returns a scalar value (`→ int`), so `s[i]` is its operator
directly; one-scalar strings come from slicing (`s[i:i+1]`), and
every string-building function (slice, replace, case maps,
int-to-string over an alphabet literal) composes slices with `+`.
The alternative — index-to-str plus an ord operator — costs a
fourth spelling for the same programs.

## Type rule

`#` needs `str` and yields `int`; `[]` needs `str` and `int` and
yields `int`; `[:]` needs `str`, `int`, `int` and yields `str`.
Anything else is `CAN6003` with an operator-naming message
(`cannot count scalars of`, `cannot index into/with`,
`cannot slice/slice with`). Result types are annotated for emit
dispatch like every other operator.

## Failure model: loud, never silent, never typed

Out-of-range index, inverted or out-of-range slice bounds, and
huge indices are loud evaluation faults (`str index out of
range`, `str slice out of range`), in the `/`-by-zero class —
not typed errors, so operators add no `emits` machinery. Standard
functions gate explicitly (`index < #value`) and raise their own
typed errors (`text.index_out_of_range`, `text.invalid_slice`);
the loud fault is unreachable past the gate, the a10 pattern.

Amendment (retrospective): the core is partial, and the contract
now says so explicitly. `emits E` bounds returned language-error
outcomes; it does not assert absence of specified primitive
faults. Compile-time tables exercise their inputs; they do not
establish every partial operation's domain for all future calls —
a guarded wrapper does not guard arbitrary user-written sites.
Loud faults are legitimate "loud" under exact-or-loud; what is
retracted is any unqualified promise that every admitted
invocation returns success or a declared error. Three categories
stay distinct: typed error outcomes, primitive domain faults,
and resource failures (e.g. production stack). Narrower library
totality claims survive individually only with their own
guard/algorithm argument.

## Parity boundary

Both runtimes count Unicode scalar values: Go over runes, TS over
spread code points (exact for every valid string, including the
astral plane). Inputs are scalar sequences — literals, concat of
literals, and slices thereof. Behavior on non-scalar input (lone
surrogates, invalid bytes) is outside the contract: Go substitutes
U+FFFD per bad byte while TS yields lone-surrogate code points,
so counts and scalar values diverge there, silently on both
sides. Stay inside valid Unicode; the contract covers exactly
what literals, concat, and slices can produce.

## Proof costs

- No new recursion: operators are strict, total-or-loud
  expressions, admissible anywhere a binop is (scrutinees, bodies,
  test args) but not in patterns — strings still admit no match
  decomposition.
- No linkage interaction: operators are language-level, so every
  file uses them same-file. The Text slice composes without a
  single cross-file call, which is why the operators choice
  dissolves the Order-0 problem for text.
- TS emit: `#` is inline `BigInt([...s].length)`; `[]` and `[:]`
  ride exact `$canStrAt`/`$canStrSlice` helpers with explicit
  bounds throws (bigint indices convert through a safe-integer
  gate, never silently).

## Implementation

- `parse.go`: `strlen`/`stridx`/`strslice` kinds (`Hi` carries the
  slice end); `#` prefix at atom level; `topBracket` postfix scan
  (index 0 keeps the list-literal branch).
- `check.go`/`eval.go` walkers descend the new operands;
  `catalog.go` descends them for stub kinds.
- `types.go`: `tokenOf`, result types, operand rules.
- `eval.go`: Go-rune semantics with loud faults.
- `emit.go`: inline length, two helpers, `childType` dispatch.
- Tests: `TestTextOpsEval` (values incl. astral, five loud
  shapes), `TestDiagnoseTextOpsMismatch` (four CAN6003 messages),
  `str__len`/`str__at`/`str__slice` fixture fns (proof tables plus
  helper-presence emit pins).
