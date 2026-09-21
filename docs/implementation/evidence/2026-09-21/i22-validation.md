# I22 — native numeric and conversion catalogue

The current compiler admits all sixteen I22 operations from the existing closed
catalogue. It resolves their exact input/result/error contracts through the same
intrinsic machinery as the byte catalogue. Ordinary calls, callable references,
fixtures and array callbacks share those contracts; no alternate evaluator or
legacy kernel dispatch is added.

`runtime/number.ts` supplies a domain-runtime-bound factory, following the existing
byte adapter pattern. Native String implements int/float/bool formatting; Number
and BigInt perform conversions; Math.floor/ceil/trunc/round and Number predicates
preserve native IEEE results. Int-to-float requires a finite result and equal
BigInt round-trip, admitting exact values beyond the safe-integer cutoff.
Float-to-int requires finiteness and integrality, then converts the actual native
binary64 value. Boolean conversion admits only integers 0 and 1.

Text parsing uses full-input grammar checks before native BigInt/Number. Integer
spelling is optional minus followed by zero or a nonzero decimal sequence. Float
parsing accepts decimal source numeric tokens from C2: integer spelling, or a
fraction/exponent form with the optional minus. Consequently `01` is rejected but
`01.0` and `00e1` are admitted, matching the source lexer. `.5`, `1.`, leading plus,
whitespace, base prefixes, separators, NaN/Infinity tokens, suffixes and trailing
newlines are rejected. A complete matched string is compared with the input so
JavaScript's permissive final-newline `$` anchor cannot admit a prefix. Parsed
floats must be finite; native finite underflow and signed zero are preserved.

The four existing domain errors retain their allocated identities and payloads:
1000 number::inexact(reason), 1001 text::invalid_number(input),
1002 text::invalid_bool(input), and 1003 number::invalid_bool(value).

## Acceptance evidence

- Runtime tests compare all Math operations and Number predicates directly with
  the qualified Bun results across NaN, infinities, signed zero, half ties,
  subnormal values and large finite values.
- Formatting tests cover exact large bigint digits, native shortest float
  spelling, nonfinite strings and the required `-0` to `"0"` result.
- Exact conversion tests distinguish 9007199254740992 from 9007199254740993 and
  admit exact powers beyond the safe-integer range. Fractional/nonfinite inputs
  and overflowing/inexact bigint conversions produce the declared domain error.
- Parsing tests cover the complete decimal grammar, nonfinite overflow, finite
  underflow, exact bigint parsing, malformed Unicode-looking digits/signs and
  embedded/trailing whitespace or control characters.
- Checker negatives reject implicit int/float/bool/string coercion, the obsolete
  decimal base type, unsupported numeric operations and discarded domain bounds.
- `std/scalars/current` is a standalone current-language project with 40 mandatory
  assertions. Its staged integration executes every operation, a first-class
  formatting reference used as an array callback, and an intrinsic fixture.

## Validation

- Qualified Bun numeric suite: 6 tests / 323 expectations passed.
- Strict TypeScript checking passed for the numeric runtime and tests.
- Full runtime suite: 157 tests / 1,440 expectations passed.
- Staged numeric integration passed all 40 assertion roots, executed main, and
  strictly type-checked all generated TypeScript. The absolute private runtime
  was invoked offline from an unrelated cwd with `PATH=/nonexistent`.
- Full `go test ./compiler/... ./tests/integration -count=1` passed with qualified
  `CAN_BUN`, the pinned local `CAN_BUN_ARCHIVE`, and strict `CAN_TSC` enabled.

The old adjacent scalar/ASCII programs and generated outputs remain historical
inputs for the old test consumers; their coordinated deletion belongs to I43/I44.
Their READMEs now identify the current replacement explicitly. They are not part
of the active CLI semantics and cannot serve as a fallback. I23 owns exact-amount
adapters, while I24 owns the separate named text/Unicode catalogue.
