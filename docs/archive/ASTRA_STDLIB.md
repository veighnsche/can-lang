# can-lang standard library program

**Build one contract stack, not five competing frameworks: a monomorphic pure standard library, typed HTML construction, one HTTP handler model, one typed SQL query model, and one component model.**

The most useful next language work is **string operations and constructor-controlled brands**. Those unlock meaningful serialization and safe server rendering without waiting for async or a frontend runtime.

Three dependencies in the brief should be refined:

* **SQL parameter binding must not require string interpolation.** Query text and parameter values should remain separate, as they do in Go’s parameterized-query interface. String building is relevant to an optional query-construction facility, not to binding values. ([Go.dev][1])
* **HTML safety requires more than nominal distinction.** It needs protected constructors and separate contracts for text, attributes, URLs, and complete HTML fragments. Existing safe-HTML libraries make these distinctions because safety is contextual. ([Go Packages][2])
* **The layers are an adoption order, not a strict dependency chain.** Server rendering does not inherently need async. SQL does not need templates. Pure UI rendering does not need SQL.

All names below are **proposed contracts**, not claims that these functions already exist. A wishlist entry becomes blessed only when its motivating program and checks ship.

---

## Reading the catalogue

### Signature notation

`(x: int) → int` abbreviates a function returning a named success record containing an `int`. It does **not** assume direct scalar-return calls.

Unless an error list follows, the proposed function declares `emits []`. For example:

```text
(x: int, lower: int, upper: int) → int
  ! [math.invalid_bounds]
```

Every actual `.can` declaration must spell out its `emits`, revision, tests, and relevant call evidence.

Where two concrete names share a row, they are **separate monomorphic functions**, not an inferred overload. `N`, `T`, and `E` below are specification notation; polymorphic APIs remain blocked until explicit generic and error-set parameters exist.

### Dependency labels

| Label           | Meaning                                                                                                                                          |
| --------------- | ------------------------------------------------------------------------------------------------------------------------------------------------ |
| **NOW**         | Implementable in pure `.can` using the current brief’s features.                                                                                 |
| **NOW†**        | Also implementable today, but an obvious guarded implementation can be impractically expensive. Expressibility is not a performance endorsement. |
| **Text**        | String inspection and construction, with specified indexing and Unicode semantics.                                                               |
| **Numeric**     | Explicit decimal coefficient/scale access and practical exact numeric conversion primitives. No floats or implicit coercions.                    |
| **Collections** | Finite immutable collection values with explicit element types and checked traversal.                                                            |
| **Types**       | Explicit generic types/functions, tagged alternatives, and first-class outcomes where required.                                                  |
| **Functions**   | Function values with explicit argument, result, error, and capability contracts.                                                                 |
| **Brands**      | Constructor-controlled brands or opaque types: consumers cannot manufacture evidence with an unrestricted `seal`.                                |
| **Async**       | Structured tasks, explicit cancellation/deadlines, checked completion handling, and deterministic test scripts.                                  |
| **Resources**   | Opaque resource handles and checked acquisition/use/release protocols.                                                                           |
| **Schema**      | Explicit, revisioned schema descriptors with machine-checked codecs or query contracts.                                                          |
| **Host**        | A conforming TypeScript `extern` implementation. This is an implementation dependency, not a claim that `extern` syntax is missing.              |

**Important release gate:** the earlier reviewer guide treats cross-file `.can` calls as scripted rather than executed. Unless that has changed, the standard-library program needs an explicit decision about evaluating verified pure imports. A caller’s arbitrary script should not substitute for evaluating `std__int__clamp`. The function bodies below can still be `NOW`; trustworthy computational reuse is a separate linkage question. 

---

# 1. Generic standard library

## 1.1 Numeric fundamentals

**Bless explicit monomorphic numeric modules first.** Do not wait for generics to standardize ordinary arithmetic contracts.

| Module / functions                                       | One-line contract                                                                                          | Why this is the standard choice                                                                       | Dependency |
| -------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------- | ---------- |
| `std/int`: `std__int__abs`<br>`std/dec`: `std__dec__abs` | `(value: N) → N`; non-negative magnitude.                                                                  | One exact magnitude operation; no minimum-integer overflow exception under the current integer model. | NOW        |
| `std__int__negate`, `std__dec__negate`                   | `(value: N) → N`; returns `0 - value`.                                                                     | Explicit callable negation where expression syntax lacks unary negation.                              | NOW        |
| `std__int__sign`, `std__dec__sign`                       | `(value: N) → int`; exactly `-1`, `0`, or `1`.                                                             | One documented sign convention.                                                                       | NOW        |
| `std__int__min`, `std__dec__min`                         | `(left: N, right: N) → N`; smaller operand.                                                                | Canonical pairwise minimum.                                                                           | NOW        |
| `std__int__max`, `std__dec__max`                         | `(left: N, right: N) → N`; larger operand.                                                                 | Canonical pairwise maximum.                                                                           | NOW        |
| `std__int__clamp`, `std__dec__clamp`                     | `(value: N, lower: N, upper: N) → N ! [math.invalid_bounds]`.                                              | Bounds are inclusive; reversed bounds fail instead of being silently swapped.                         | NOW        |
| `std__int__distance`, `std__dec__distance`               | `(left: N, right: N) → N`; absolute difference.                                                            | No signed-distance ambiguity.                                                                         | NOW        |
| `std__int__square`, `std__dec__square`                   | `(value: N) → N`; exact square.                                                                            | Useful callable operation without introducing exponentiation syntax.                                  | NOW        |
| `std__int__pow`, `std__dec__pow`                         | `(base: N, exponent: int) → N ! [math.negative_exponent]`.                                                 | One non-negative-exponent contract; exponent zero returns one, including zero to zero.                | NOW        |
| `std__int__add_bounded`                                  | `(left, right, lower, upper: int) → int ! [math.invalid_bounds, math.out_of_range]`.                       | Checked application bounds without pretending unbounded integers overflow.                            | NOW        |
| `std__int__subtract_bounded`                             | Same inputs/errors; checks the exact difference.                                                           | Same range policy across arithmetic operations.                                                       | NOW        |
| `std__int__multiply_bounded`                             | Same inputs/errors; checks the exact product.                                                              | No saturation unless the caller explicitly chooses clamping.                                          | NOW        |
| `std__dec__lerp`                                         | `(start: dec, end: dec, fraction: dec) → dec ! [math.out_of_range]`; fraction must be within zero and one. | Exact interpolation with an explicit domain, not a floating-point utility.                            | NOW        |

Do **not** add `std__int__add`, `std__int__subtract`, or `std__int__multiply`: those merely provide competing names for existing operators.

## 1.2 Integer arithmetic without a division operator

**The absence of `/` does not make integer division inexpressible.** A guarded scan can count quotient and remainder using addition, subtraction, comparisons, and a decreasing integer budget.

That earns `NOW†`, not permission to ship an implementation taking one recursive step per unit of a huge bigint.

| Module / functions                             | One-line contract                                                                                   | Why this is the standard choice                                                               | Dependency |
| ---------------------------------------------- | --------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------- | ---------- |
| `std/int/division`: `std__int__divmod`         | `(dividend: int, divisor: int) → DivMod(quotient, remainder) ! [math.zero_divisor]`.                | Euclidean contract: `a = b*q + r`, with `0 <= r < abs(b)`. One answer for negative operands.  | NOW†       |
| `std__int__mod`                                | `(value: int, modulus: int) → int ! [math.zero_divisor]`; returns the Euclidean remainder.          | Same semantics as `divmod`, never a second remainder convention.                              | NOW†       |
| `std__int__is_multiple`                        | `(value: int, divisor: int) → bool ! [math.zero_divisor]`.                                          | Divisibility does not conflate a zero divisor with false.                                     | NOW†       |
| `std__int__is_even`, `std__int__is_odd`        | `(value: int) → bool`.                                                                              | Defined for negative values as well as positive ones.                                         | NOW†       |
| `std/int/number`: `std__int__gcd`              | `(left: int, right: int) → int`; non-negative result, with `gcd(0,0) = 0`.                          | One normalization convention for later rational arithmetic.                                   | NOW†       |
| `std__int__lcm`                                | `(left: int, right: int) → int`; non-negative result, zero when either input is zero.               | Shares the GCD normalization policy.                                                          | NOW†       |
| `std__int__sqrt_floor`                         | `(value: int) → IntRoot(root, remainder) ! [math.negative_input]`; `value = root*root + remainder`. | Integer result plus an exact witness, not an approximate decimal.                             | NOW†       |
| `std__int__is_prime`                           | `(value: int) → bool`; values below two return false.                                               | A precise small-domain utility; not a cryptographic primality API.                            | NOW†       |
| `std__int__next_power_of_two`                  | `(value: int) → int ! [math.nonpositive_input]`; least power of two not below the input.            | Explicit rounding direction for capacities and partitioning.                                  | NOW†       |
| `std/int/combinatorics`: `std__int__factorial` | `(value: int) → int ! [math.negative_input]`; zero factorial is one.                                | Straightforward guarded recurrence and boundary tests.                                        | NOW        |
| `std__int__binomial`                           | `(n: int, k: int) → int ! [math.invalid_count]`; requires `0 <= k <= n`.                            | Rejects invalid inputs instead of introducing a second “out-of-domain means zero” convention. | NOW†       |
| `std__int__sum_to`                             | `(upper: int) → int ! [math.negative_input]`; sums zero through `upper`.                            | Useful recurrence example without requiring division syntax.                                  | NOW        |

**Required checks:** all operand-sign combinations, zero cases, quotient/remainder identities, guard boundaries, and explicit resource expectations for the motivating workload.

## 1.3 Decimal representation, division, and rounding

**Never make rounding a mode that changes ordinary arithmetic.** Rounding belongs in explicitly named operations whose results expose discarded information.

| Module / functions                                        | One-line contract                                                                                                           | Why this is the standard choice                                                     | Dependency                           |
| --------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------- | ------------------------------------ |
| `std/dec/representation`: `std__dec__parts`               | `(value: dec) → DecimalParts(coefficient: int, scale: int)`.                                                                | One canonical representation: non-negative scale and normalized trailing zeros.     | Numeric                              |
| `std__dec__from_parts`                                    | `(coefficient: int, scale: int) → dec ! [math.negative_scale]`.                                                             | Exact construction; no parsing or host-number intermediary.                         | NOW†; Numeric for a practical kernel |
| `std__dec__scale_by_power_of_ten`                         | `(value: dec, exponent: int) → dec`.                                                                                        | Exact decimal-point movement in either direction.                                   | NOW†                                 |
| `std/dec/division`: `std__dec__divide_exact`              | `(dividend: dec, divisor: dec) → dec ! [math.zero_divisor, math.nonterminating_decimal]`.                                   | Exact success or an explicit refusal; no hidden rounding.                           | Numeric                              |
| `std__dec__divide_round_half_even`                        | `(dividend: dec, divisor: dec, scale: int) → RoundedQuotient(value, remainder) ! [math.zero_divisor, math.negative_scale]`. | A named rounding rule and the exact witness `dividend = divisor*value + remainder`. | Numeric                              |
| `std/dec/rounding`: `std__dec__round_half_even`           | `(value: dec, scale: int) → RoundedDecimal(value, discarded) ! [math.negative_scale]`.                                      | The original equals returned value plus discarded amount.                           | Numeric                              |
| `std__dec__floor`, `std__dec__ceil`, `std__dec__truncate` | `(value: dec) → dec`; integral-valued result with the named direction.                                                      | Three different operations, not an ambient rounding setting.                        | Numeric                              |
| `std__dec__sqrt_exact`                                    | `(value: dec) → dec ! [math.negative_input, math.nonrepresentable_result]`.                                                 | Refuses results not representable as finite decimals.                               | Numeric                              |

### Exact fractions as records, not a fifth base type

Reserve a small rational module for the first program that genuinely needs values such as one-third without rounding.

| Module / functions                        | One-line contract                                                   | Why this is the standard choice                                 | Dependency                      |
| ----------------------------------------- | ------------------------------------------------------------------- | --------------------------------------------------------------- | ------------------------------- |
| `std/ratio`: `std__ratio__make`           | `(numerator: int, denominator: int) → Ratio ! [math.zero_divisor]`. | Reduced fraction, positive denominator, protected construction. | Brands; NOW† integer algorithms |
| `std__ratio__add`, `std__ratio__multiply` | `(left: Ratio, right: Ratio) → Ratio`.                              | Exact arithmetic without changing `dec` semantics.              | Brands                          |
| `std__ratio__divide`                      | `(left: Ratio, right: Ratio) → Ratio ! [math.zero_divisor]`.        | No special infinities or invalid numeric values.                | Brands                          |
| `std__ratio__to_dec_exact`                | `(value: Ratio) → dec ! [math.nonterminating_decimal]`.             | Explicitly crosses the rational/finite-decimal boundary.        | Brands + Numeric                |

Do not bless scalar-returning `sin`, `cos`, `log`, `exp`, or general `sqrt` as though every answer were an exact `dec`. A later `std__math__sqrt_bounds` or `std__math__exp_bounds` must return **certified bounds**, with an explicit accuracy target and a checked stopping argument.

## 1.4 Boolean logic, comparisons, and predicates

| Module / functions                                                                                 | One-line contract                                                                           | Why this is the standard choice                                                  | Dependency |
| -------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------- | ---------- |
| `std/bool`: `std__bool__not`                                                                       | `(value: bool) → bool`.                                                                     | Complete two-row truth table.                                                    | NOW        |
| `std__bool__and`, `std__bool__or`, `std__bool__xor`                                                | `(left: bool, right: bool) → bool`.                                                         | Complete four-row truth tables; arguments are already evaluated.                 | NOW        |
| `std__bool__implies`, `std__bool__equivalent`                                                      | `(left: bool, right: bool) → bool`.                                                         | Named logical relations useful in validation contracts.                          | NOW        |
| `std/compare`: `std__compare__int`, `std__compare__dec`, `std__compare__str`, `std__compare__bool` | `(left: T, right: T) → int`; exactly `-1`, `0`, or `1`; boolean order is false before true. | One three-way comparison convention, with separate concrete types.               | NOW        |
| `std/predicate`: `std__int__is_zero`, `std__dec__is_zero`                                          | `(value: N) → bool`.                                                                        | Exact zero, not an epsilon comparison.                                           | NOW        |
| `std__int__is_positive`, `std__dec__is_positive`                                                   | `(value: N) → bool`.                                                                        | Strictly greater than zero.                                                      | NOW        |
| `std__int__is_negative`, `std__dec__is_negative`                                                   | `(value: N) → bool`.                                                                        | Strictly less than zero.                                                         | NOW        |
| `std__int__in_closed_range`, `std__dec__in_closed_range`                                           | `(value, lower, upper: N) → bool ! [math.invalid_bounds]`.                                  | Inclusive boundaries and invalid-range handling are fixed.                       | NOW        |
| `std__int__in_open_range`, `std__dec__in_open_range`                                               | Same shape/errors; excludes both endpoints.                                                 | No boolean flag that quietly changes boundary semantics.                         | NOW        |
| `std/str`: `std__str__is_empty`                                                                    | `(value: str) → bool`; equality with the empty string.                                      | Useful today despite opaque strings.                                             | NOW        |
| `std/select`: `std__select__int`, `std__select__dec`, `std__select__str`, `std__select__bool`      | `(condition: bool, when_true: T, when_false: T) → T`.                                       | Selection of already-computed values only. It is not short-circuit control flow. | NOW        |

## 1.5 Validation

**Separate predicates from validators.** A predicate returns a boolean; a validator either returns the accepted value or a producer-owned typed error.

| Module / functions                                                 | One-line contract                                                                                          | Why this is the standard choice                                                     | Dependency                      |
| ------------------------------------------------------------------ | ---------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------- | ------------------------------- |
| `std/validate`: `std__validate__require`                           | `(condition: bool, field: str, rule: str) → Pass ! [validation.failed]`.                                   | One structured assertion primitive; no panic.                                       | NOW                             |
| `std__validate__int_range`, `std__validate__dec_range`             | `(value, lower, upper: N) → N ! [validation.invalid_bounds, validation.out_of_range]` for int, `[validation.dec_invalid_bounds, validation.dec_out_of_range]` for dec. | Returns the original accepted value; never clamps invalid input. One kind has one field list, so dec payloads get dec kinds (same split as `math.dec_*`). | NOW |
| `std__validate__int_positive`, `std__validate__dec_positive`       | `(value: N) → N ! [validation.not_positive]` for int, `[validation.dec_not_positive]` for dec.              | Explicit strict positivity.                                                         | NOW                             |
| `std__validate__int_nonnegative`, `std__validate__dec_nonnegative` | `(value: N) → N ! [validation.negative_value]` for int, `[validation.dec_negative_value]` for dec.          | Zero is deliberately accepted.                                                      | NOW                             |
| `std__validate__str_nonempty`                                      | `(value: str) → str ! [validation.empty_value]`.                                                           | No implicit trimming or whitespace policy.                                          | NOW                             |
| `std__validate__exclusive_pair`                                    | `(left: bool, right: bool) → Pass ! [validation.exclusive_choice]`.                                        | Exactly one choice, not merely at least one.                                        | NOW                             |
| `std__validate__str_length`                                        | `(value: str, minimum: int, maximum: int) → str ! [validation.invalid_bounds, validation.invalid_length]`. | A named Unicode-scalar length policy.                                               | Text                            |
| `std__validate__one_of`                                            | `(value: T, allowed: Seq<T>, compare: Comparator<T>) → T ! [validation.not_allowed]`.                      | Explicit permitted values and equality semantics.                                   | Collections + Types + Functions |
| `std__validate__all`                                               | `(checks: Seq<ValidationOutcome>) → ValidationReport`.                                                     | Accumulates all failures in input order instead of hiding a fail-fast choice.       | Collections + Types             |
| `std__validate__schema`                                            | `(value: T, schema: Schema<T>) → T ! [validation.schema_violation]`.                                       | One reusable schema-validation surface with structured paths and complete payloads. | Schema + Types + Collections    |

Do not put business policies such as password strength, valid usernames, or accepted email domains into generic validation. Those belong to named domain contracts.

## 1.6 Conversions between the four base types

Every non-identity direction is listed here.

| Function                         | One-line contract                                                                            | Why this is the standard choice                                      | Dependency                                |
| -------------------------------- | -------------------------------------------------------------------------------------------- | -------------------------------------------------------------------- | ----------------------------------------- |
| `std__convert__int_to_dec`       | `(value: int) → dec`; exact numeric preservation.                                            | Never passes through a floating-point host value.                    | NOW†                                      |
| `std__convert__dec_to_int_exact` | `(value: dec) → int ! [convert.fractional_value]`.                                           | Fractional values are rejected, not truncated.                       | Numeric                                   |
| `std__convert__int_to_str`       | `(value: int) → str`; canonical base-ten representation.                                     | One spelling: no locale, grouping, or implicit alternate radix.      | Text; practical integer formatting kernel |
| `std__convert__str_to_int`       | `(value: str) → int ! [convert.invalid_integer]`.                                            | Strict documented grammar; no whitespace trimming or fallback value. | Text                                      |
| `std__convert__dec_to_str`       | `(value: dec) → str`; canonical exact decimal representation.                                | No exponential-format heuristic or loss of digits.                   | Text + Numeric                            |
| `std__convert__str_to_dec`       | `(value: str) → dec ! [convert.invalid_decimal]`.                                            | Exact digit accumulation; no host-number parser.                     | Text                                      |
| `std__convert__bool_to_str`      | `(value: bool) → str`; exactly `"true"` or `"false"`.                                        | Entire contract is a finite decision table.                          | NOW                                       |
| `std__convert__str_to_bool`      | `(value: str) → bool ! [convert.invalid_boolean]`; accepts only `"true"` and `"false"`.      | No case folding, truthiness, or accepted-synonym lottery.            | NOW                                       |
| `std__convert__bool_to_int`      | `(value: bool) → int`; false maps to zero, true to one.                                      | Explicit two-value encoding.                                         | NOW                                       |
| `std__convert__int_to_bool`      | `(value: int) → bool ! [convert.invalid_boolean_encoding]`; accepts only zero and one.       | Nonzero does not silently mean true.                                 | NOW                                       |
| `std__convert__bool_to_dec`      | `(value: bool) → dec`; false maps to zero, true to one.                                      | Same explicit encoding in the decimal domain.                        | NOW                                       |
| `std__convert__dec_to_bool`      | `(value: dec) → bool ! [convert.invalid_boolean_encoding]`; accepts only exact zero and one. | No numeric truthiness.                                               | NOW                                       |

Identity conversions need no functions.

**Explicitly forbidden conversion behavior:** fractional `dec → int` without a named rounding operation; arbitrary nonzero-number truthiness; silent parse fallback; brand-to-base conversion through generic helpers; and automatic bigint/decimal conversion through TypeScript `number`.

The `NOW†` classification of `int_to_dec` is intentional: an integer-controlled accumulation can implement it today. A useful implementation for large integers needs a better kernel, not a false claim that the conversion is mathematically impossible.

## 1.7 Outcome and optional-value combinators

**Do not pretend these are available as generic libraries today.** Generic outcome manipulation requires outcomes to be values, explicit type parameters, and—for mapping or recovery—function values.

| Module / function                    | One-line contract                                                                     | Why this is the standard choice                                                          | Dependency          |
| ------------------------------------ | ------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------- | ------------------- |
| `std/outcome`: `std__outcome__is_ok` | `(value: Outcome<T,E>) → bool`.                                                       | Inspects an outcome without discarding its payload.                                      | Types               |
| `std__outcome__map`                  | `(value: Outcome<T,E>, transform: PureFn<T,U>) → Outcome<U,E>`.                       | Changes success values while preserving every error and payload.                         | Types + Functions   |
| `std__outcome__and_then`             | `(value: Outcome<T,E1>, next: Fn<T,U,E2>) → Outcome<U,E1 union E2>`.                  | Explicit sequential composition with a declared combined error set.                      | Types + Functions   |
| `std__outcome__map_error`            | `(value: Outcome<T,E1>, mapping: TotalErrorMap<E1,E2>) → Outcome<T,E2>`.              | Every source error kind requires an explicit mapping.                                    | Types + Functions   |
| `std__outcome__recover`              | `(value: Outcome<T,E>, handlers: TotalRecovery<E,T>) → T`.                            | Recovery is exhaustive, not catch-all suppression.                                       | Types + Functions   |
| `std__outcome__zip`                  | `(left: Outcome<A,E1>, right: Outcome<B,E2>) → Outcome<Pair<A,B>, ZipErrors<E1,E2>>`. | Preserves left-only, right-only, and simultaneous failures of already-computed outcomes. | Types               |
| `std__outcome__collect`              | `(values: Seq<Outcome<T,E>>) → Outcome<Seq<T>, Seq<IndexedError<E>>>`.                | Accumulates failures with stable input positions.                                        | Collections + Types |
| `std/option`: `std__option__require` | `(value: Option<T>) → T ! [option.absent]`.                                           | Absence is explicit, never represented by an invented sentinel.                          | Types               |
| `std__option__map`                   | `(value: Option<T>, transform: PureFn<T,U>) → Option<U>`.                             | One optional-value mapping operation.                                                    | Types + Functions   |
| `std__option__value_or`              | `(value: Option<T>, alternative: T) → T`.                                             | The alternative is an explicit, already-evaluated argument—not a hidden default.         | Types               |

At specialization, every `E` must be an explicitly enumerated set. These APIs must not introduce an untyped `error`, automatic error-set inference, or generic `unwrap_or_panic`.

## 1.8 Future generic text, collections, and codecs

These remain part of the generic library programme, but **not its no-new-features first release**.

### Text

Choose Unicode-scalar indexing for text operations; keep byte operations separate. Do not normalize, trim, or case-fold implicitly.

| Functions                                                            | One-line contract                                                         | Why this is the standard choice                                                  | Dependency         |
| -------------------------------------------------------------------- | ------------------------------------------------------------------------- | -------------------------------------------------------------------------------- | ------------------ |
| `std__str__concat`                                                   | `(left: str, right: str) → str`.                                          | The basic explicit construction operation.                                       | Text               |
| `std__str__length_scalars`                                           | `(value: str) → int`.                                                     | The unit appears in the name.                                                    | Text               |
| `std__str__scalar_at`                                                | `(value: str, index: int) → int ! [text.index_out_of_range]`.             | Returns a Unicode scalar value, not an ambiguously sized “character.”            | Text               |
| `std__str__slice_scalars`                                            | `(value: str, start: int, end: int) → str ! [text.invalid_slice]`.        | Half-open range; invalid bounds never clamp.                                     | Text               |
| `std__str__contains`, `std__str__starts_with`, `std__str__ends_with` | `(value: str, pattern: str) → bool`.                                      | Exact comparisons without locale or normalization.                               | Text               |
| `std__str__find`                                                     | `(value: str, pattern: str) → int ! [text.not_found]`.                    | Scalar index or explicit absence; no `-1` sentinel.                              | Text               |
| `std__str__replace_all`                                              | `(value: str, old: str, replacement: str) → str ! [text.empty_pattern]`.  | One non-overlapping, left-to-right replacement rule.                             | Text               |
| `std__str__split`                                                    | `(value: str, separator: str) → Seq<str> ! [text.empty_separator]`.       | Retains empty fields; no implicit cleanup.                                       | Text + Collections |
| `std__str__join`                                                     | `(values: Seq<str>, separator: str) → str`.                               | Preserves sequence order.                                                        | Text + Collections |
| `std__str__trim_ascii`                                               | `(value: str) → str`.                                                     | A fixed named whitespace set, independent of locale.                             | Text               |
| `std__str__lower_ascii`, `std__str__upper_ascii`                     | `(value: str) → str`; transform ASCII letters and preserve other scalars. | Predictable protocol-oriented casing.                                            | Text               |
| `std__str__normalize_nfc`, `std__str__casefold_unicode`              | `(value: str) → str`.                                                     | Explicit transformations tied to revision-pinned Unicode data.                   | Text + pinned data |
| `std__str__length_graphemes`                                         | `(value: str) → int`.                                                     | A separate user-facing segmentation operation, never an alias for scalar length. | Text + pinned data |

### Collections

Bless **immutable ordered sequences**, **ordered-key maps**, and **sets with explicit ordering**. Avoid observable hash iteration order.

| Functions                                                                                 | One-line contract                                                                   | Why this is the standard choice                            | Dependency                      |
| ----------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------- | ---------------------------------------------------------- | ------------------------------- |
| `std__seq__empty`, `std__seq__singleton`                                                  | `() → Seq<T>`; `(value: T) → Seq<T>`.                                               | Explicit construction, no null collection.                 | Collections + Types             |
| `std__seq__length`                                                                        | `(values: Seq<T>) → int`.                                                           | Supplies an explicit traversal bound.                      | Collections                     |
| `std__seq__get`                                                                           | `(values: Seq<T>, index: int) → T ! [sequence.index_out_of_range]`.                 | Checked indexing.                                          | Collections + Types             |
| `std__seq__append`, `std__seq__concat`                                                    | Sequence plus item, or two sequences, → `Seq<T>`.                                   | Immutable, order-preserving construction.                  | Collections + Types             |
| `std__seq__slice`                                                                         | `(values, start, end) → Seq<T> ! [sequence.invalid_slice]`.                         | Same half-open rule as text slicing.                       | Collections + Types             |
| `std__seq__map`, `std__seq__filter`                                                       | `(values, total_callback) → Seq<U>` or `Seq<T>`.                                    | Callbacks have explicit termination and error contracts.   | Collections + Types + Functions |
| `std__seq__fold`                                                                          | `(values: Seq<T>, initial: A, step: PureFn<A,T,A>) → A`.                            | Explicit initial value and left-to-right evaluation.       | Collections + Types + Functions |
| `std__seq__find`                                                                          | `(values, predicate) → Option<T>`.                                                  | First matching value in input order.                       | Collections + Types + Functions |
| `std__seq__all`, `std__seq__any`                                                          | `(values, predicate) → bool`.                                                       | Specify visit order and stopping behavior in the contract. | Collections + Types + Functions |
| `std__seq__sort`                                                                          | `(values, comparator: CheckedOrder<T>) → Seq<T>`.                                   | Stable sorting with an explicit ordering contract.         | Collections + Types + Functions |
| `std__seq__unique`                                                                        | `(values, equality: CheckedEquality<T>) → Seq<T>`.                                  | Retains the first occurrence in input order.               | Collections + Types + Functions |
| `std__map__get`                                                                           | `(map: Map<K,V>, key: K) → V ! [map.key_absent]`.                                   | Missing and present-with-a-value cannot be confused.       | Collections + Types             |
| `std__map__insert`                                                                        | `(map, key, value) → Map<K,V> ! [map.key_exists]`.                                  | Insertion never silently overwrites.                       | Collections + Types             |
| `std__map__replace`                                                                       | `(map, key, value) → Map<K,V> ! [map.key_absent]`.                                  | Replacement never silently inserts.                        | Collections + Types             |
| `std__map__remove`, `std__map__entries`                                                   | Remove an existing key or return ordered entries; removal emits `[map.key_absent]`. | Explicit mutation-like operations over immutable values.   | Collections + Types             |
| `std__set__contains`, `std__set__union`, `std__set__intersection`, `std__set__difference` | Typed set operations → `bool` or `Set<T>`.                                          | One equality/order contract shared with maps.              | Collections + Types             |

A few comparator examples do not prove a total order. Initially accept built-in checked orders and explicitly constructed lexicographic orders; do not bless arbitrary callbacks as ordering proofs.

### Encoding and schemas

| Functions                                          | One-line contract                                                                                                                         | Why this is the standard choice                                                    | Dependency                                    |
| -------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------- | --------------------------------------------- |
| `std__utf8__encode`                                | `(text: str) → Bytes`.                                                                                                                    | One text-to-wire encoding.                                                         | Text + Collections                            |
| `std__utf8__decode`                                | `(bytes: Bytes) → str ! [encoding.invalid_utf8]`.                                                                                         | Reject invalid input rather than inserting replacement characters silently.        | Text + Collections                            |
| `std__hex__encode`, `std__hex__decode`             | `Bytes → str`; `str → Bytes ! [encoding.invalid_hex]`.                                                                                    | One canonical case and strict decode grammar.                                      | Text + Collections                            |
| `std__base64__encode`, `std__base64__decode`       | `Bytes → str`; `str → Bytes ! [encoding.invalid_base64]`.                                                                                 | Fixed alphabet and padding contract.                                               | Text + Collections                            |
| `std__base64url__encode`, `std__base64url__decode` | Equivalent URL-safe operations, with `[encoding.invalid_base64url]` on decode.                                                            | Separate names instead of an ambient encoding flag.                                | Text + Collections                            |
| `std__json__encode`                                | `(value: T, schema: JsonSchema<T>) → Bytes ! [json.value_out_of_schema]`.                                                                 | Explicit field mapping and numeric wire representation.                            | Schema + Text + Numeric + Collections + Types |
| `std__json__decode`                                | `(bytes: Bytes, schema: JsonSchema<T>) → T ! [json.invalid_syntax, json.duplicate_key, json.schema_mismatch, json.numeric_out_of_range]`. | Reject duplicate keys and schema drift instead of selecting undocumented behavior. | Same                                          |
| `std__schema__migrate`                             | `(value: Old, migration: Migration<Old,New>) → New ! E`.                                                                                  | Every field transformation and possible failure is declared.                       | Schema + Types + Functions                    |

For interchange, **do not assume JSON numbers preserve can-lang’s arbitrary precision in other consumers**. RFC 8259 explicitly discusses numeric precision limits and interoperability problems. Each schema should declare exact integer/decimal strings or another explicit representation; no automatic conversion through a host number. ([RFC Editor][3])

### Host-support shelf—not pure `NOW` std

| Functions                   | One-line contract                                                                               | Why this is the standard choice                                                        | Dependency                  |
| --------------------------- | ----------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------- | --------------------------- |
| `std__clock__wall_now`      | `() → Instant ! [clock.unavailable]`.                                                           | Wall time is an explicit foreign observation.                                          | Host                        |
| `std__clock__monotonic_now` | `() → MonotonicTick ! [clock.unavailable]`.                                                     | Deadlines do not silently use wall time.                                               | Host                        |
| `std__random__bytes`        | `(count: int) → Bytes ! [random.invalid_count, random.unavailable]`.                            | One declared entropy capability; never a pure-looking random helper.                   | Collections + Host          |
| `std__hash__digest`         | `(bytes: Bytes, profile: HashProfile) → Digest ! [hash.unsupported_profile, hash.unavailable]`. | Revisioned algorithm contracts; implementations cannot change the meaning of a digest. | Collections + Brands + Host |
| `std__secret__equal`        | `(left: Secret, right: Secret) → bool ! [secret.provider_failure]`.                             | Secret comparison stays behind a reviewed host contract, not ordinary string equality. | Brands + Host               |
| `std__log__write`           | `(event: LogEvent) → Pass ! [log.unavailable]`.                                                 | Structured, schema-bound events with explicit disclosure rules.                        | Schema + Collections + Host |
| `std__env__read`            | `(name: EnvName) → str ! [environment.absent, environment.denied]`.                             | No ambient configuration lookup disguised as pure code.                                | Brands + Host               |

---

# 2. Backend HTML rendering

## Blessed abstraction: typed fragment construction

**Choose one safe-fragment API, not a string-template engine plus a second component DSL.**

The central types should distinguish:

| Type                  | Meaning                                                                 |
| --------------------- | ----------------------------------------------------------------------- |
| `Html__Text`          | Encoded text suitable for the supported HTML text contexts.             |
| `Html__AttributeText` | Encoded attribute data—not a complete attribute or HTML fragment.       |
| `Html__Url`           | A parsed URL admitted by a specified navigation/resource policy.        |
| `Html__Attribute`     | A supported attribute constructed through a typed operation.            |
| `Html__Safe`          | A complete safe fragment in the renderer’s supported fragment contexts. |
| `Html__Children`      | An explicitly ordered collection of safe child fragments.               |

The security argument must be **construction-based**:

1. Raw strings can enter only through context-appropriate encoders or validators.
2. Safe values cannot be forged by library consumers.
3. Element constructors accept only the appropriate safe types.
4. The renderer preserves the checked structure.

The earlier guide exposes literal `seal` construction. That general gate must not let consumers create `Html__Safe` directly. Nominality prevents accidental mixing; it does not establish that an arbitrary branded string was escaped correctly. 

Escaping still happens in an encoder. **The type checker proves safe composition; it does not remove the need to implement and verify encoding.** Go’s template documentation likewise distinguishes text, attribute, JavaScript, CSS, and URI contexts. ([Go Packages][2])

## 2.1 Module wishlist

| Module / functions                                                                                                                     | One-line contract                                                                               | Why this is the standard choice                                                                   | Dependency                  |
| -------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------- | --------------------------- |
| `html/text`: `html__text__escape`                                                                                                      | `(raw: str) → Html__Text ! [html.invalid_text]`.                                                | One text encoder; rejects input outside its specified preservation domain.                        | Text + Brands               |
| `html__text__node`                                                                                                                     | `(text: Html__Text) → Html__Safe`.                                                              | Explicit promotion from encoded text to a child fragment.                                         | Brands                      |
| `html/attribute`: `html__attribute__text`                                                                                              | `(name: Html__TextAttributeName, raw: str) → Html__Attribute ! [html.invalid_attribute_value]`. | The name type excludes URL, event-handler, and style attributes.                                  | Text + Brands               |
| `html__attribute__boolean`                                                                                                             | `(name: Html__BooleanAttributeName, present: bool) → Html__Attribute`.                          | Presence/absence has an explicit HTML-specific meaning.                                           | Brands                      |
| `html__attribute__id`                                                                                                                  | `(value: str) → Html__Attribute ! [html.invalid_identifier]`.                                   | One identifier-validation policy.                                                                 | Text + Brands               |
| `html__attribute__classes`                                                                                                             | `(tokens: Seq<Css__Class>) → Html__Attribute`.                                                  | Classes are tokens, not an arbitrary CSS string.                                                  | Text + Collections + Brands |
| `html__attribute__href`, `html__attribute__src`                                                                                        | `(url: Html__Url) → Html__Attribute`.                                                           | URL-bearing attributes cannot accept escaped-but-dangerous raw strings.                           | Brands                      |
| `html__attributes__make`                                                                                                               | `(items: Seq<Html__Attribute>) → Html__Attributes ! [html.duplicate_attribute]`.                | No first-wins or last-wins ambiguity.                                                             | Collections + Brands        |
| `html/url`: `html__url__parse`                                                                                                         | `(raw: str, base: Web__Origin) → Html__Url ! [html.invalid_url, html.disallowed_scheme]`.       | One explicit URL policy; syntax validation and scheme admission are separate from HTML escaping.  | Text + Brands               |
| `html/fragment`: `html__fragment__empty`                                                                                               | `() → Html__Safe`.                                                                              | Explicit empty output.                                                                            | Brands                      |
| `html__fragment__join`                                                                                                                 | `(children: Html__Children) → Html__Safe`.                                                      | Deterministic ordering without raw-string concatenation at call sites.                            | Collections + Text + Brands |
| `html/el`: `html__el__div`, `html__el__span`, `html__el__p`                                                                            | `(attributes: TagAttributes, children: Html__Children) → Html__Safe`.                           | Typed constructors, not an arbitrary tag-name string.                                             | Text + Collections + Brands |
| `html__el__h1`, `html__el__h2`, `html__el__h3`, `html__el__h4`, `html__el__h5`, `html__el__h6`                                         | Same constructor shape with heading-specific contracts.                                         | Heading level is visible in the operation name.                                                   | Same                        |
| `html__el__main`, `html__el__section`, `html__el__article`, `html__el__nav`, `html__el__aside`, `html__el__header`, `html__el__footer` | Typed attributes and children → `Html__Safe`.                                                   | Standard semantic structure without a second layout language.                                     | Same                        |
| `html__el__ul`, `html__el__ol`, `html__el__li`                                                                                         | Typed attributes and appropriately constrained children → `Html__Safe`.                         | List structure is not assembled from unmatched strings.                                           | Same                        |
| `html__el__table`, `html__el__thead`, `html__el__tbody`, `html__el__tr`, `html__el__th`, `html__el__td`                                | Table-specific attributes and child categories → `Html__Safe`.                                  | Table contexts receive explicit contracts rather than inheriting a universal fragment assumption. | Same                        |
| `html__el__a`                                                                                                                          | `(attributes: AnchorAttributes, children: Html__Children) → Html__Safe`.                        | Its URL must already satisfy the link policy.                                                     | Same                        |
| `html__el__img`                                                                                                                        | `(attributes: ImageAttributes) → Html__Safe`.                                                   | No children; alternative-text choice is an explicit field.                                        | Text + Brands               |
| `html__el__br`, `html__el__hr`                                                                                                         | `(attributes: TagAttributes) → Html__Safe`.                                                     | Void elements cannot accidentally receive children.                                               | Text + Brands               |
| `html__el__form`, `html__el__label`, `html__el__button`                                                                                | Typed form attributes and children → `Html__Safe`.                                              | Form method, action, and button behavior are explicit.                                            | Text + Collections + Brands |
| `html__el__input`                                                                                                                      | `(attributes: InputAttributes) → Html__Safe`.                                                   | Input kind and relevant fields are checked together.                                              | Text + Types + Brands       |
| `html__el__textarea`, `html__el__title`                                                                                                | `(attributes: TagAttributes, text: Html__Text) → Html__Safe`.                                   | These take text, not arbitrary child markup.                                                      | Text + Brands               |
| `html/asset`: `html__asset__stylesheet`, `html__asset__script`                                                                         | `(asset: ApprovedAsset, policy: AssetPolicy) → Html__Safe`.                                     | Approved external assets, not raw CSS or script strings.                                          | Brands + Schema             |
| `html/render`: `html__render__document`                                                                                                | `(language, title, head, body) → Html__Safe ! [html.invalid_document_structure]`.               | One document assembly rule.                                                                       | Text + Collections + Brands |
| `html__render__utf8`                                                                                                                   | `(document: Html__Safe) → Bytes`.                                                               | The explicit wire-output boundary.                                                                | Text + Collections + Brands |

For the first release, **exclude arbitrary raw HTML, inline script, inline event handlers, arbitrary CSS strings, and foreign-content namespaces**. Add each only with its own construction and context argument—not by widening `Html__Safe` until its meaning becomes “someone said this was fine.”

## 2.2 Three syntax candidates

These are fragments, not complete functions. Child collection syntax is proposed and waits on collection values.

### Candidate A — ordinary calls and explicit child values: **recommended**

Assume `title` is already `Html__Text`, and both attribute values have been validated.

```can
match call html__text__node(text = title)
  on Ok text_node =>
    match call html__el__h1(
      attributes = heading_attributes
      children = Html__Children(
        items = [text_node.value]
      )
    )
      on Ok heading =>
        match call html__el__div(
          attributes = container_attributes
          children = Html__Children(
            items = [heading.value]
          )
        )
          on Ok container =>
            Ok(value = container.value)
```

**Why choose it:** it uses ordinary calls, ordinary outcome handling, and explicit child values. No template parser, implicit escaping context, or hidden outcome unwrapping is required.

It is tall and inconvenient. That is consistent with the language.

### Candidate B — nested element-call expressions

```can
html__el__div(
  attributes = container_attributes
  children = Html__Children(
    items = [
      html__el__h1(
        attributes = heading_attributes
        children = Html__Children(
          items = [
            html__text__node(text = title)
          ]
        )
      )
    ]
  )
)
```

This is visually attractive, but it is **not merely a library design**. Under the earlier guide’s call model, calls occur as match scrutinees; this version also needs a rule explaining how function outcomes become child values. 

**Reject for the initial library.** Do not smuggle expression calls and implicit success extraction into “template ergonomics.”

### Candidate C — a dedicated indentation template block

```can
template page__title(title: Html__Text)
  html__el__div(attributes = container_attributes)
    children
      html__el__h1(attributes = heading_attributes)
        children
          html__text__node(text = title)
```

**Reject.** It introduces a second execution surface and special child-binding rules. The improvement is mainly aesthetic, while the proof surface grows.

**Recommendation:** ship A. Reconsider B only if expression-call semantics become justified by ordinary `.can` programs independently of templates.

### Template acceptance programme

The first real sketch should be a server-rendered account page containing user-controlled display names, links, validation messages, and a form. Its checks must include adversarial text, quotes, ampersands, attempted closing tags, rejected URL schemes, duplicate attributes, forbidden brand construction, and wrong-context brand use.

Do not claim universal XSS protection from a handful of examples. The release needs the constructor/encoder argument, plus those regression tests.

---

# 3. Web backend standard

## Blessed abstraction: immutable request → explicit response

Use one application-handler shape:

```text
Http__Handler<E, C>
  (context: Http__Context, request: Http__Request)
  → asynchronous Http__Response
  emits E
  effects C
```

`E` and `C` are explicit parameters, not inferred sets.

A mounted route must supply an exhaustive error-to-response mapping. This lets the router store normalized handlers without erasing unhandled application errors.

**Borrow Go’s small handler boundary, not its mutable response-writer mechanics.** Go’s handler writes through `ResponseWriter`; for can-lang, returning a complete response is easier to give a deterministic outcome contract. ([Go Packages][4])

### Alternatives rejected

| Domain     | Blessed choice                                              | Rejected alternative                                                              |
| ---------- | ----------------------------------------------------------- | --------------------------------------------------------------------------------- |
| Routing    | Typed route descriptions, with ambiguous overlaps rejected. | Registration-order precedence, regex-first routing, convention-derived routes.    |
| Handlers   | Explicit context and immutable request → response.          | Mutable ambient request contexts and partially written responses.                 |
| Middleware | A typed wrapper with an explicit next-handler usage rule.   | Unrestricted callbacks that may invoke `next` repeatedly or retain it.            |
| Errors     | Exhaustive declared mapping at the mount boundary.          | A global catch-all that converts every failure into an undocumented 500 response. |

## 3.1 Wishlist

`Handler` below means the explicit shape above. These functions are pure unless marked `Host`.

| Module / functions                                                                                                                                                  | One-line contract                                                                                                                                                                                            | Why this is the standard choice                                                 | Dependency                                            |
| ------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------- | ----------------------------------------------------- |
| `http/route`: `http__route__get`, `http__route__post`, `http__route__put`, `http__route__patch`, `http__route__delete`, `http__route__head`, `http__route__options` | `(path: RoutePath, handler: MountedHandler) → Http__Route`.                                                                                                                                                  | Method selection is explicit; GET does not silently register HEAD.              | Text + Collections + Types + Functions + Async        |
| `http__route__path`                                                                                                                                                 | `(segments: Seq<RouteSegment>) → RoutePath ! [http.invalid_route]`.                                                                                                                                          | Literal and typed-capture segments, not a magic route string.                   | Text + Collections + Types                            |
| `http/router`: `http__router__make`                                                                                                                                 | `(routes: Seq<Http__Route>) → Http__Router ! [http.duplicate_route, http.ambiguous_route]`.                                                                                                                  | Order-independent registration.                                                 | Collections + Types + Functions + Async               |
| `http__router__resolve`                                                                                                                                             | `(router, method, path) → RouteMatch ! [http.not_found, http.method_not_allowed, http.invalid_path]`.                                                                                                        | One path-decoding and matching policy.                                          | Text + Collections + Types                            |
| `http/handler`: `http__handler__map_errors`                                                                                                                         | `(handler: Handler<E,C>, mapping: TotalErrorMap<E,Http__Response>) → MountedHandler<C>`.                                                                                                                     | Every application failure receives an explicit response policy.                 | Types + Functions + Async                             |
| `http/middleware`: `http__middleware__apply`                                                                                                                        | `(handler, wrappers: Seq<Middleware>) → Handler`.                                                                                                                                                            | Order is the sequence order; no discovered middleware.                          | Collections + Types + Functions + Async + Resources   |
| `http__middleware__authenticate`                                                                                                                                    | `(handler, verifier, rejection_response) → Handler`; verifier errors are explicitly mapped.                                                                                                                  | Authentication is one boundary, not a framework-global side effect.             | Same + Host where needed                              |
| `http__middleware__authorize`                                                                                                                                       | `(handler, policy, rejection_response) → Handler`.                                                                                                                                                           | Policy is supplied and typed; no hidden role hierarchy.                         | Types + Functions + Async                             |
| `http__middleware__limit_body`                                                                                                                                      | `(handler, maximum_bytes: int) → Handler ! [http.invalid_limit]`.                                                                                                                                            | No unbounded body read or hidden size default.                                  | Async + Resources                                     |
| `http__middleware__deadline`                                                                                                                                        | `(handler, duration: Duration) → Handler ! [http.invalid_deadline]`.                                                                                                                                         | Deadline behavior is explicit and tested.                                       | Async + Resources + Host                              |
| `http__middleware__cors`                                                                                                                                            | `(handler, policy: CorsPolicy) → Handler ! [http.invalid_cors_policy]`.                                                                                                                                      | Origins, credentials, and methods are explicit contract data.                   | Text + Collections + Async                            |
| `http/request`: `http__request__path_param`                                                                                                                         | `(match: RouteMatch, name: ParamName) → ParamValue ! [http.parameter_absent]`.                                                                                                                               | Typed route captures; no repeated ad hoc parsing.                               | Types + Brands                                        |
| `http__request__query_one`                                                                                                                                          | `(request, name: str) → str ! [http.parameter_absent, http.parameter_repeated]`.                                                                                                                             | Rejects accidental multiplicity instead of choosing a value.                    | Text + Collections                                    |
| `http__request__query_all`                                                                                                                                          | `(request, name: str) → Seq<str>`.                                                                                                                                                                           | Explicit multiplicity, preserving order.                                        | Text + Collections                                    |
| `http__request__header_one`                                                                                                                                         | `(request, name: HeaderName) → HeaderValue ! [http.header_absent, http.header_repeated]`.                                                                                                                    | Does not apply an unsafe universal comma-join rule.                             | Collections + Brands                                  |
| `http__request__body_bytes`                                                                                                                                         | `(request, maximum_bytes, context) → Bytes ! [http.body_too_large, http.cancelled, http.deadline_exceeded, http.transport_failure]`.                                                                         | Bounded, single-owner body consumption.                                         | Collections + Async + Resources + Host                |
| `http__request__json`                                                                                                                                               | `(request, schema: JsonSchema<T>, maximum_bytes, context) → T ! [http.body_too_large, http.unsupported_media_type, http.invalid_body, http.cancelled, http.deadline_exceeded, http.transport_failure]`.      | One schema-driven decoder; complete nested error details belong in the payload. | Schema + Text + Numeric + Collections + Types + Async |
| `http__request__form`                                                                                                                                               | Same bounded request shape → typed form record; emits `[http.body_too_large, http.unsupported_media_type, http.invalid_body, http.cancelled, http.deadline_exceeded, http.transport_failure]`.               | Duplicate fields and absent values follow the form schema.                      | Schema + Text + Collections + Async                   |
| `http/response`: `http__response__bytes`                                                                                                                            | `(status: HttpStatus, headers: Headers, body: Bytes) → Http__Response`.                                                                                                                                      | The lowest complete-response constructor.                                       | Collections + Brands                                  |
| `http__response__text`                                                                                                                                              | `(status, headers, body: str) → Http__Response`.                                                                                                                                                             | A fixed text encoding and media-type contract.                                  | Text + Collections + Brands                           |
| `http__response__html`                                                                                                                                              | `(status, headers, body: Html__Safe) → Http__Response`.                                                                                                                                                      | Raw strings cannot enter the HTML response sink.                                | HTML layer + Collections                              |
| `http__response__json`                                                                                                                                              | `(status, headers, value: T, schema: JsonSchema<T>) → Http__Response ! [json.value_out_of_schema]`.                                                                                                          | Reuses the single blessed codec.                                                | Schema + Text + Numeric + Collections + Types         |
| `http__response__redirect`                                                                                                                                          | `(status: RedirectStatus, destination: RedirectUrl, headers) → Http__Response`.                                                                                                                              | Redirect class and destination policy are explicit.                             | Text + Brands + Collections                           |
| `http/cookie`: `http__cookie__make`                                                                                                                                 | `(name, value, attributes: CookieAttributes) → Cookie ! [http.invalid_cookie]`.                                                                                                                              | Security attributes are required fields, not defaults.                          | Text + Brands                                         |
| `http/server`: `http__server__start`                                                                                                                                | `(config: ServerConfig, router, context) → Server ! [http.invalid_config, http.bind_failed, http.tls_failed, http.cancelled, http.deadline_exceeded]`.                                                       | Starts a host-driven service and returns a managed handle.                      | Async + Resources + Host                              |
| `http__server__stop`                                                                                                                                                | `(server: Server, context) → Pass ! [http.shutdown_failed, http.deadline_exceeded]`.                                                                                                                         | Explicit bounded shutdown and handle consumption.                               | Async + Resources + Host                              |
| `http/client`: `http__client__send`                                                                                                                                 | `(client, request, context) → Http__Response ! [http.name_resolution_failed, http.connect_failed, http.tls_failed, http.protocol_failure, http.cancelled, http.deadline_exceeded, http.response_too_large]`. | One transport contract; retries and redirects are not silently enabled.         | Async + Resources + Collections + Host                |

**Termination boundary:** do not introduce an infinitely running `.can` `serve()` function. The host owns the event loop; each `.can` request callback must terminate. Startup, shutdown, and each host operation have their own completion contracts.

The first HTTP sketch should expose a real status/account page with one successful route, a typed parameter, a form error, an authentication rejection, and a bounded external dependency.

---

# 4. Standard SQL connector

## Blessed abstraction: revisioned query + typed parameters + typed rows

Use one descriptor shape:

```text
Sql__Query<Parameters, Row, Cardinality, SchemaRevision, Dialect>
```

A query declaration fixes its statement, parameter fields, row fields, cardinality, schema revision, and dialect. Callers provide **parameter values**, not assembled SQL.

This follows the useful part of Go’s `database/sql` architecture: the standard interface is separate from database drivers. Go’s package does not include the drivers themselves. ([Go Packages][5])

### Correct dependency boundary

| Operation                             |   Collections needed? |                                                  String building needed? | Async needed for the proposed standard? |
| ------------------------------------- | --------------------: | -----------------------------------------------------------------------: | --------------------------------------: |
| Fixed query returning one typed row   |    No, not inherently |                                                                       No |                                     Yes |
| Fixed query returning an optional row | Tagged optional value |                                                                       No |                                     Yes |
| Fixed query returning many rows       |                   Yes |                                                                       No |                                     Yes |
| Batch parameter sets                  |                   Yes |                                                                       No |                                     Yes |
| Dynamic structural query construction |               Usually | A checked query representation or serializer—not parameter interpolation |                                     Yes |

A monomorphic one-row `extern` can be explored before the complete SQL standard. Do not falsely block that experiment on list values.

**Reject:** an ORM as the first standard, reflection-based row mapping, interpolated query strings, transparent cross-dialect promises, and arbitrary driver-specific error leakage.

One contract shape does not mean one SQL dialect. A PostgreSQL descriptor and a SQLite descriptor may share the standard shape without pretending their semantics are interchangeable.

## 4.1 Error contract

For compactness, the following table uses this explicitly enumerated query error set:

```text
QueryErrors =
  sql.cancelled
  sql.deadline_exceeded
  sql.unavailable
  sql.permission_denied
  sql.schema_mismatch
  sql.constraint_violation
  sql.serialization_failure
  sql.invalid_value
  sql.driver_fault
```

Actual `.can` declarations enumerate their applicable kinds. `sql.driver_fault` is a documented provider-contract failure with a sanitized diagnostic payload—not an excuse to omit known failure modes.

## 4.2 Wishlist

| Module / functions                           | One-line contract                                                                                                                                                | Why this is the standard choice                                                                     | Dependency                                              |
| -------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------- | ------------------------------------------------------- |
| `sql/query`: `sql__query__declare`           | `(statement: StaticSql, parameter_schema, row_schema, cardinality, schema_revision, dialect) → Sql__Query`; invalid declarations fail compilation.               | One offline-checkable query contract.                                                               | Schema + Types                                          |
| `sql/parameters`: `sql__parameters__bind`    | `(query: Query<P,R>, parameters: P) → BoundQuery<R> ! [sql.invalid_value]`.                                                                                      | Names, types, and exact numeric representability are checked; values remain separate from SQL text. | Schema + Types                                          |
| `sql/pool`: `sql__pool__open`                | `(driver: Driver, config: PoolConfig, context) → Pool ! [sql.invalid_config, sql.authentication_failed, sql.unavailable, sql.cancelled, sql.deadline_exceeded]`. | Explicit pool limits and acquisition policy; liberal conforming drivers.                            | Brands + Async + Resources + Host                       |
| `sql__pool__close`                           | `(pool: Pool, context) → Pass ! [sql.close_failed, sql.deadline_exceeded]`.                                                                                      | Explicit lifecycle, no abandoned pool handles.                                                      | Async + Resources + Host                                |
| `sql__pool__ping`                            | `(pool, context) → Pass ! [sql.unavailable, sql.cancelled, sql.deadline_exceeded]`.                                                                              | A real connectivity check rather than assuming construction connected.                              | Async + Host                                            |
| `sql/query`: `sql__query__one`               | `(pool, query: Query<P,R>, parameters: P, context) → R ! QueryErrors + [sql.no_rows, sql.too_many_rows]`.                                                        | Exactly one means exactly one.                                                                      | Schema + Types + Async + Host                           |
| `sql__query__optional`                       | Same inputs → `Option<R> ! QueryErrors + [sql.too_many_rows]`.                                                                                                   | Absence is data; multiplicity remains an error.                                                     | Schema + Types + Async + Host                           |
| `sql__query__rows`                           | `(pool, query, parameters, maximum_rows: int, context) → Seq<R> ! QueryErrors + [sql.invalid_limit, sql.row_limit_exceeded]`.                                    | Bounded materialization; exceeding the bound never silently truncates.                              | Schema + Types + Collections + Async + Host             |
| `sql__query__execute`                        | `(pool, command: Command<P>, parameters: P, context) → Execution(affected_rows: int) ! QueryErrors`.                                                             | Commands do not pretend to return rows or a portable “last inserted ID.”                            | Schema + Types + Async + Host                           |
| `sql__query__batch`                          | `(pool, command, parameters: Seq<P>, context) → Seq<Execution> ! QueryErrors + [sql.batch_limit_exceeded]`.                                                      | Ordered results and an explicitly specified atomicity policy.                                       | Schema + Types + Collections + Async + Resources + Host |
| `sql/row`: `sql__row__decode`                | `(wire_row, schema: RowSchema<R>) → R ! [sql.column_missing, sql.unexpected_column, sql.null_violation, sql.invalid_value]`.                                     | No lossy decimal conversion or null sentinel.                                                       | Schema + Types + Numeric                                |
| `sql/transaction`: `sql__transaction__begin` | `(pool, isolation: Isolation, access: AccessMode, context) → Transaction ! QueryErrors + [sql.unsupported_isolation]`.                                           | No driver-default isolation choice.                                                                 | Types + Async + Resources + Host                        |
| `sql__transaction__commit`                   | `(transaction, context) → CommitReceipt ! [sql.serialization_failure, sql.constraint_violation, sql.commit_unknown, sql.cancelled, sql.deadline_exceeded]`.      | An uncertain commit is not falsely reported as a rollback.                                          | Async + Resources + Host                                |
| `sql__transaction__rollback`                 | `(transaction, context) → RollbackReceipt ! [sql.rollback_failed, sql.cancelled, sql.deadline_exceeded]`.                                                        | Explicit rollback and handle consumption.                                                           | Async + Resources + Host                                |
| `sql/schema`: `sql__schema__verify`          | `(pool, expected: SchemaRevision, context) → SchemaAttestation ! [sql.schema_mismatch, sql.unavailable, sql.cancelled, sql.deadline_exceeded]`.                  | Runtime deployment drift has a declared outcome.                                                    | Schema + Async + Host                                   |
| `sql/migration`: `sql__migration__plan`      | `(current_schema, target_schema, migrations) → MigrationPlan ! [sql.migration_gap, sql.migration_conflict, sql.unapproved_data_loss]`.                           | Migration order and declared destructive changes are inspectable.                                   | Schema + Collections                                    |
| `sql__migration__apply`                      | `(pool, plan, context) → MigrationReceipt ! QueryErrors + [sql.migration_failed, sql.commit_unknown]`.                                                           | Applying a migration is explicit—not an import-time side effect.                                    | Schema + Collections + Async + Resources + Host         |

**No live database during ordinary compilation.** Query checks use a pinned schema artifact. Runtime schema verification and driver conformance tests cover different obligations; scripted `.can` tests do not establish that an arbitrary TypeScript driver binds parameters correctly.

The motivating sketch should be a small ledger or reservation service—not a read-only “hello database.” It must exercise exact decimals, absence, duplicate matches, nullable columns, constraint failure, transactional rollback, and uncertain commit handling.

---

# 5. Preact-ish frontend standard

## Blessed abstraction: pure view + explicit state transition + typed commands

Borrow the component-and-props idea, not every execution model available in Preact. Preact documents both function and class components; can-lang should bless only one shape. ([Preact][6])

```text
Ui__Component<Props, State, Message>

initialize
  Props → State

view
  Props × State × Children → Ui__Node

update
  State × Message → Transition<State, Message>

Transition
  next_state
  explicit commands
```

The view is pure. Event callbacks produce typed messages. Updates produce a new state and an explicit command collection. Host operations execute commands; their complete outcomes return as messages.

**Reject:** class components, hook-order protocols, implicit dependency tracking, render-time I/O, arbitrary DOM mutation from view functions, and a second state-management framework.

Use one controlled-state form model. That is a deliberate contract choice, not a claim that Preact universally recommends controlled inputs. Preact’s documentation describes synchronization pitfalls when a controlled input is not re-rendered; can-lang’s driver conformance suite must test actual DOM resynchronization, including updates that return unchanged state. ([Preact][7])

## 5.1 Wishlist

| Module / functions                       | One-line contract                                                                                                               | Why this is the standard choice                                                       | Dependency                                       |
| ---------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------- | ------------------------------------------------ |
| `ui/component`: `ui__component__define`  | `(initialize, view, update, command_contract) → Component<P,S,M>`; invalid contracts fail compilation.                          | One explicit component lifecycle.                                                     | Types + Functions + Collections                  |
| `ui__component__initialize`              | `(component, props: P) → S ! E_initialize`.                                                                                     | Initialization errors are declared, not thrown from constructors.                     | Types + Functions                                |
| `ui__component__view`                    | `(component, props: P, state: S, children) → Ui__Node ! E_view`.                                                                | Pure deterministic rendering with complete outcomes.                                  | Types + Functions + Collections                  |
| `ui__component__update`                  | `(component, state: S, message: M) → Transition<S,M> ! E_update`.                                                               | State changes are ordinary testable functions.                                        | Types + Functions + Collections                  |
| `ui/children`: `ui__children__make`      | `(children: Seq<KeyedNode>) → Ui__Children ! [ui.duplicate_key]`.                                                               | Stable explicit identity; no array-index key default.                                 | Collections + Types + Brands                     |
| `ui/node`: `ui__node__text`              | `(value: str) → Ui__Node ! [ui.invalid_text]`.                                                                                  | A text node is not an HTML injection sink.                                            | Text + Types                                     |
| `ui__node__fragment`                     | `(children: Ui__Children) → Ui__Node`.                                                                                          | Explicit grouping without introducing an element.                                     | Collections + Types                              |
| `ui__node__element`                      | `(tag: SupportedTag, attributes: TypedAttributes, children) → Ui__Node ! [ui.invalid_structure]`.                               | One checked element representation shared with server rendering where possible.       | HTML contracts + Types + Collections             |
| `ui/slot`: `ui__slot__fill`              | `(slot: SlotContract, children: Ui__Children) → FilledSlot ! [ui.slot_mismatch]`.                                               | Named child contracts instead of untyped prop bags.                                   | Types + Collections                              |
| `ui/event`: `ui__event__click`           | `(callback: PureFn<ClickEvent,M>) → EventBinding<M>`.                                                                           | Explicit event-to-message conversion; no inline script strings.                       | Types + Functions + Brands                       |
| `ui__event__input`                       | `(callback: PureFn<InputEvent,M>) → EventBinding<M>`.                                                                           | Input payloads have one typed shape.                                                  | Types + Functions + Brands                       |
| `ui__event__submit`                      | `(callback: PureFn<SubmitEvent,M>, behavior: SubmitBehavior) → EventBinding<M>`.                                                | Browser-default prevention is an explicit decision.                                   | Types + Functions + Brands                       |
| `ui/form`: `ui__form__define`            | `(schema: FormSchema<T>, state: FormState<T>, bindings) → Ui__Node ! [ui.invalid_form_contract]`.                               | One typed form and validation model.                                                  | Schema + Types + Functions + Collections         |
| `ui__input__text`, `ui__input__textarea` | `(value: str, attributes, input_binding) → Ui__Node`.                                                                           | Controlled text state and explicit callbacks.                                         | Text + Types + Functions                         |
| `ui__input__checkbox`                    | `(checked: bool, attributes, change_binding) → Ui__Node`.                                                                       | Boolean state, not string truthiness.                                                 | Types + Functions                                |
| `ui__input__select`                      | `(options: Seq<OptionItem<T>>, selected: Option<T>, binding) → Ui__Node ! [ui.invalid_selection, ui.duplicate_option]`.         | Typed values and explicit absent selection.                                           | Collections + Types + Functions                  |
| `ui__button__make`                       | `(behavior: ButtonBehavior, attributes, children, binding) → Ui__Node`.                                                         | Submit versus ordinary button behavior is mandatory.                                  | Types + Functions + Collections                  |
| `ui__form__validate`                     | `(state: FormState<T>, schema: FormSchema<T>) → ValidationReport<T>`.                                                           | Reuses the generic validation contracts.                                              | Schema + Types + Collections                     |
| `ui/list`: `ui__list__render`            | `(items: Seq<T>, key: KeyFn<T>, view: PureFn<T,Ui__Node>) → Ui__Node ! [ui.duplicate_key]`.                                     | Key and rendering functions are explicit.                                             | Collections + Types + Functions                  |
| `ui/table`: `ui__table__render`          | `(rows: Seq<R>, columns: ColumnSchema<R>, key) → Ui__Node ! [ui.duplicate_key, ui.invalid_columns]`.                            | One data-table composition primitive, not a separate grid framework.                  | Schema + Collections + Types + Functions         |
| `ui/command`: `ui__command__http`        | `(request, response_schema, complete: TotalOutcomeMap<M>) → Ui__Command<M>`.                                                    | Every transport and decode outcome maps to a message.                                 | HTTP client + Schema + Types + Functions + Async |
| `ui__command__delay`                     | `(duration, completed_message: M) → Ui__Command<M> ! [ui.invalid_duration]`.                                                    | Time is an explicit command with cancellation ownership.                              | Types + Async + Resources + Host                 |
| `ui/runtime`: `ui__runtime__mount`       | `(component, props, target: DomTarget, limits) → MountedComponent ! [ui.target_missing, ui.mount_failed, ui.limit_exceeded]`.   | Explicit target, lifecycle, and resource bounds.                                      | Types + Functions + Async + Resources + Host     |
| `ui__runtime__dispatch`                  | `(mounted, message: M) → Pass ! [ui.dispatch_failed]`.                                                                          | One ordered update entry point.                                                       | Types + Async + Resources + Host                 |
| `ui__runtime__unmount`                   | `(mounted, context) → Pass ! [ui.cleanup_failed, ui.deadline_exceeded]`.                                                        | Subscriptions and commands cannot outlive their owner silently.                       | Async + Resources + Host                         |
| `ui/render`: `ui__render__html`          | `(node: Ui__Node) → Html__Safe ! [ui.invalid_structure]`.                                                                       | Reuses the HTML security contract rather than inventing another escaping system.      | HTML layer + Types + Collections                 |
| `ui__render__hydrate`                    | `(target, node, manifest: HydrationManifest) → MountedComponent ! [ui.target_missing, ui.hydration_mismatch, ui.mount_failed]`. | Mismatches fail explicitly rather than silently rebuilding under a “hydration” label. | Schema + Async + Resources + Host                |
| `ui/router`: `ui__router__resolve`       | `(routes, location: Web__Location) → Ui__Route ! [ui.route_not_found, ui.invalid_location]`.                                    | Reuses the web URL/path contracts.                                                    | Text + Collections + Types                       |
| `ui__router__navigate`                   | `(destination: Web__Location, history: HistoryBehavior) → Pass ! [ui.navigation_denied, ui.navigation_failed]`.                 | Push versus replace is explicit.                                                      | Brands + Host                                    |

Each concrete component declares its complete initialization, view, and update error sets. Framework code must not infer them or hide them behind a universal `ui.error`.

The first full UI sketch should be an editable account or reservation form with typed validation, a pending command, success and failure messages, cancellation on unmount, and hydration. A static counter is insufficient to settle this contract.

**The browser application may remain active indefinitely; each initializer, view, update, and completion callback must terminate.** That distinction must be part of the language’s host-callback model, not an undocumented exception.

---

# 6. Sequenced roadmap

## Admission rule for every entry

A module is not blessed until it has a real motivating `.can` sketch, complete producer-owned outcomes, green compile-time decision tables, negative examples for its invariants, a checked termination argument where it iterates, and target/driver conformance evidence where execution crosses into TypeScript.

These are separate obligations. A scripted success from an escaping or SQL extern is not evidence that its implementation escapes or binds correctly.

## Recommended sequence

Each row is a milestone, **not one giant implementation change**. Within it, spec and land one operation at a time.

| Order  | Deliverable                                                                              | What it unlocks                                                                                                                  | Real-program gate and required checks                                                                                                  |
| ------ | ---------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------- |
| **0**  | Standard-library contract and linkage rules                                              | A trustworthy meaning of importing pure std code; fixed conventions for result records, errors, revisions, and host conformance. | A deliberately incorrect scripted result for a pure imported helper must not masquerade as evaluation of that helper.                  |
| **1**  | Monomorphic scalar validation and range helpers                                          | Useful std code with no language additions.                                                                                      | Extend the counter into a quota counter. Start with `std__validate__int_range`; test every boundary and complete error payload.        |
| **2**  | Boolean utilities, comparisons, finite conversions, guarded arithmetic                   | Shared decision logic and retry calculations.                                                                                    | Retry/backoff sketch with zero, negative, exhausted, and large-value cases. Separate termination from acceptable resource cost.        |
| **3**  | Text primitives and explicit text semantics                                              | Parsing, formatting, protocol tokens, and encoding work.                                                                         | A configuration-record parser with malformed input and Unicode boundary tests. No implicit trimming or normalization.                  |
| **4**  | Decimal representation and practical exact conversion                                    | Decimal formatting, exact division, explicit rounding witnesses.                                                                 | Invoice/ledger calculation. Check large integers, fractional rejection, non-terminating division, and evaluator/TypeScript agreement.  |
| **5**  | Constructor-controlled brands                                                            | Trustworthy encoded text, safe URLs, capability handles, and schema-validated values.                                            | Attempts to forge `Html__Safe`, pass a safe URL as attribute text, or bypass a constructor must be rejected.                           |
| **6**  | Finite collections and tagged alternatives                                               | Child lists, repeated form fields, multiple rows, optional values.                                                               | A small catalogue page. Check empty collections, bounds, deterministic ordering, and explicit traversal decreases.                     |
| **7**  | Safe HTML constructors and document rendering                                            | Useful server-rendered pages before async.                                                                                       | Account page with untrusted names, links, forms, and adversarial inputs. Ship the encoder/construction argument with it.               |
| **8**  | Explicit generics, first-class outcomes, and typed function values                       | Outcome combinators, collection transforms, reusable handler/component composition.                                              | Extract duplication from existing sketches. No inferred type or error parameters; higher-order calls must preserve termination checks. |
| **9**  | Schemas and canonical codecs                                                             | Typed JSON, forms, row decoding, migration descriptors, hydration data.                                                          | Round-trip a real request/response record containing a large integer, exact decimal, optional field, and collection.                   |
| **10** | Resource protocols, then structured async                                                | Managed tasks, body streams, connections, cancellation, and bounded completion.                                                  | A two-operation workflow with success, both failure orders, cancellation, and cleanup. Every handle’s disposition must be checked.     |
| **11** | HTTP router, handler adapters, and server driver                                         | A deployable backend using one request/response contract.                                                                        | Serve the account page; exercise route conflicts, malformed input, authentication failure, body limits, and shutdown.                  |
| **12** | SQL query descriptors, one-row operations, transactions, then rows/batches               | Typed persistence without an ORM.                                                                                                | Ledger or reservation service; parameter separation, schema drift, exact numeric decoding, rollback, and unknown commit outcomes.      |
| **13** | Pure component model, then browser command/runtime integration                           | The complete preact-ish layer.                                                                                                   | Editable form with typed events, controlled inputs, explicit commands, unmount cancellation, and hydration checks.                     |
| **14** | Streaming, richer query construction, advanced migrations, additional rendering contexts | Extensions justified by actual blocked programs.                                                                                 | No speculative blessing. Each capability supplies its own motivating sketch and proof boundary before implementation.                  |

The dependency branches should remain visible: **HTML can ship before async; a fixed one-row SQL experiment can precede collection-heavy SQL; pure component rendering can develop independently of the database layer.**

**Start with the quota-counter validation module. Then prioritize strings and protected constructors—the shortest path from today’s pure helpers to a genuinely useful, contract-checked web page.**

[1]: https://go.dev/doc/database/sql-injection "Avoiding SQL injection risk - The Go Programming Language"
[2]: https://pkg.go.dev/html/template "template package - html/template - Go Packages"
[3]: https://www.rfc-editor.org/rfc/rfc8259 "RFC 8259: The JavaScript Object Notation (JSON) Data Interchange Format | RFC Editor"
[4]: https://pkg.go.dev/net/http "http package - net/http - Go Packages"
[5]: https://pkg.go.dev/database/sql "sql package - database/sql - Go Packages"
[6]: https://preactjs.com/guide/a10/components/ "Components – Preact Guide"
[7]: https://preactjs.com/guide/a10/forms/ "Forms – Preact Guide"
