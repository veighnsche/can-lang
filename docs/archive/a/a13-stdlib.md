# a13 — Standard-library program (rows 0–2, first cut)

Status: landed (part). Rows 0 (decision, not mechanism), 1, and the
pure NOW slice of row 2 ship here. Everything else in the brief
stays explicitly deferred with its blocker named below.

## What landed

- `std/quota/quota.can` — row 1. Monomorphic scalar
  validators (`validate__require`, `validate__int_range`,
  `validate__int_nonnegative`, `validate__str_nonempty`,
  `validate__exclusive_pair`) plus a quota counter
  (`quota__consume`, `quota__usage`) that reuses them. Gate met:
  every boundary and complete error payload is a decision-table
  row (24 tests), and the counter consumes through the validators,
  so over-quota, negative, and reversed-bound inputs fail closed.
- `std/scalars/scalars.can` — row 2 NOW slice. Boolean
  logic, three-way comparisons, selection, int/dec predicates,
  int/dec fundamentals, bounded arithmetic, exact power and
  recurrence helpers, the six pure NOW conversions, and a bounded
  backoff delay. Gate met: zero, negative, saturated, and
  large-attempt cases are rows, and the termination proof and the
  resource policy are separate artifacts (see below).
- Both modules freeze as byte-identical goldens
  (`TestGoldenQuotaCounter`, `TestGoldenStdScalars`).

## Order 0 decision: linkage (decision, not mechanism)

Cross-file `.can` calls stay scripted. Until a linkage amendment
lands, standard helpers ship as same-file locals in the program
that uses them, and each file carries its own decision tables. A
caller's script for a cross-file helper is not evaluation of that
helper, and nothing in this cut pretends otherwise.

Acceptance test for the future amendment: a deliberately
incorrect scripted result for a verified pure helper must fail the
build. Today it passes. The amendment flips that test and must
say how scripts, rev pins, and per-test stores interact with
evaluated imports. The flagship (`auth.can` scripting `db.down()`,
an outcome its provider body can never produce) is the reason
this cannot be a silent default: evaluated imports would forbid
exactly the stub the flagship relies on.

## Name mapping (resolved by issue 5)

The R3 single-separator reading was wrong: the grammar's verb
class already admits hierarchical names, so spec names land
verbatim — `std__int__abs`, `std__bool__not`,
`std__compare__int`, `std__select__int`,
`std__validate__int_range`, `std__convert__bool_to_str`, and so
on per domain. Pinned by `TestMultiUnderscoreNameAccepted` and
`TestMalformedNamesRejected` in `compiler/naming_test.go`, which
also pin the remaining rejections (no separator,
leading/trailing/doubled runs). Two error kinds split by payload
type, since one kind has one field list:
`convert.invalid_boolean` for strings,
`convert.invalid_boolean_encoding` for ints,
`convert.invalid_dec_encoding` for decs. Every function returns
a named success record (`Int__Value`, `Dec__Value`,
`Str__Value`, `Bool__Value`, `Quota__Usage`, `Validate__Pass`);
the brief's `(x: int) -> int` notation already requires this.

## Emitter change: multi-shape ok union

No language rule limited a file to one Ok shape; the emitter
assumed it anyway and failed multi-shape files. `emitModule` now
emits one ok member per distinct Ok shape, sorted for stability.
Single-shape modules emit byte-identical output (gallery goldens
unchanged); per-fn coherence still lives in `collectOkShapes`.
Pinned by `TestMultiShapeEmit` and
`TestSingleShapeEmitUnchanged` in
`compiler/emit_multishape_test.go`.

## Conventions proven by the cut

- Fallible recursion splits in two: a total `decreases` helper
  with no emits under the canonical guard, and a checked entry
  that rejects negatives before delegating. No error arm is
  unreachable, so the test-per-arm law holds by construction.
  `int__pow`, `int__factorial`, `int__sum_to`, and `dec__pow`
  all follow it.
- Termination and cost are separate claims. `int__pow_from`
  terminates for every non-negative exponent by proof, while
  `backoff__delay` caps attempts at 30 as a cost policy at the
  call site. The 2^30 bound is exact; the ceiling keeps builds
  cheap.
- Bounds are inclusive and reversed bounds fail
  (`math.invalid_bounds`, `validation.invalid_bounds`); no
  silent swap anywhere.
- `compare__str` is byte order on the wire encoding:
  deterministic, not Unicode aware.

## Deferred with blockers (not silently dropped)

- Integer division family: the kernel landed as
  `docs/archive/a/a17-division.md` (`/`, `%`, `divmod`, `mod`,
  `is_multiple`, `is_even`, `is_odd`). `binomial` still needs
  its fuel-pattern home, and `gcd`, `lcm`, `sqrt_floor`,
  `is_prime`, `next_power_of_two` wait on fuel-pattern
  recursion. Issue 4 stays open.
- `int_to_dec` (NOW†): integer-controlled accumulation needs a
  digit-extraction kernel that does not exist without
  division. `dec_to_int_exact`, `int_to_str`, `str_to_int`,
  `dec_to_str`, `str_to_dec`: need the Numeric/Text layers.
- Text operations: concatenation landed as `docs/archive/a/a16-text.md`
  (`+` on strings, `std__str__concat` blessed); measurement and
  indexing wait on the scalar-access surface decision recorded
  there. Issue 3 stays open.
- Constructor-controlled brands: landed as `docs/archive/a/a15-brands.md`.
  Bodies seal only their own module's brands (CAN6004); tests and
  scripts may name any declared brand. `Html__Safe` and its sibling
  brands will arrive under this rule, unforgable by consumers.
- HTML rendering: needs Text, Collections, and the brand change
  above. No template flavor is blessed; candidate A (ordinary
  calls, explicit child values) is the only one expressible
  today, and child collections do not exist yet.
- HTTP, SQL, UI layers: need Functions, Async, Resources, and
  Schema. Recorded principles for their turn: parameter values
  stay separate from SQL text (binding is not interpolation);
  HTML safety is contextual (text, attribute, URL, fragment);
  the layers are an adoption order, and HTML ships before async.

## Consequences

- New goldens live beside their sources in `std/`; `go test ./...`,
  `go run ./tools/modcheck`, and `go run ./tools/gramcheck`
  cover them.
- The stdlib grows one operation at a time, each with its
  decision table, negative cases, and termination argument
  where it iterates. No speculative blessing.

## Open decisions (do not block)

- Linkage amendment design (see acceptance test above).
- Whether `dec` predicates and int predicates should share one
  parametrized contract once generics exist, or stay mirrored.
- The efficient integer-division kernel and its cost model.
