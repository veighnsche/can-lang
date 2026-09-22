# a36 — Seq typed construction, S1

Status: shipped. P0 approved under standing approval; P1–P5
implemented and green (`go test -count=1 ./...`, `modcheck`,
`gramcheck`): typed literals, element checking, eval + structural
`vEq`, TS emit, committed decision tables in
`compiler/seq_s1_test.go`. Two deviations from the P0 as written:
bare-Seq returns are rejected now (decision 6 revised — an
untestable function is a compile error, not a deferred question),
and the element rule is uniform over known types (decision 5
revised — the restriction would have been the extra machinery).

This is the first Seq slice only: finite immutable sequence values
with explicit element types, including the empty case. Length,
checked access, append, and all customer operations (`str__join`,
`str__split`, `fragment__join`, `attributes__make`) are separate slices
with their own P0s. Nothing here unlocks a customer on its own.

## Goal

Let programs denote ordered immutable sequences with a checked element
type, so later slices can measure them (length), read them (checked
get + traversal), build them (append), and consume them (customers).
The empty sequence is a value, not a failure.

## Success Criteria

- Explicitly typed Seq literals parse, type-check, evaluate, and emit
  TS with value semantics: order preserved, empty fields preserved,
  repeated members preserved.
- Element-type checking at construction: a `Seq<str>` never admits an
  `int`, a brand, or a wrong-brand member; emptiness does not waive
  the check (`Seq<str>[]` vs `Seq<Html__Safe>[]` are distinct types).
- Brand ownership preserved inside literals: executable positions mint
  only their own module's brands (CAN6004); tests and given rows may
  name any declared brand as checked data (existing a15 rule).
- `go test -count=1 ./...`, `go run ./tools/modcheck`,
  `go run ./tools/gramcheck` all green, plus new committed goldens
  for the literal shapes and rejection rows.
- No new customer operation ships in this slice.

## Context And Current Facts

- `knownType` (`compiler/types.go:94-103`) admits base types, records,
  and brands only. There is no `Seq<T>`; this slice adds it.
- `Small{Kind: "list"}` (`compiler/parse.go:22-23,711-723`) parses
  `[...]` today, but that shape is a scripting form: `check.go:740`
  and `eval.go:820-821` treat outer lists as repeated exchanges, and
  `eval.go:515-516` rejects list literals outside `given` ("outside
  the v0 subset"). Bare `[...]` is not an available value syntax.
- `tycker` list checking (`compiler/types.go:538-541`) visits members
  with no element-type requirement. Reusing it as executable Seq
  without new rules would leave checks incomplete.
- Runtime values (`compiler/eval.go:15-26`) have no sequence kind;
  `vEq` (`compiler/eval.go:272-307`) structurally compares
  str/int/dec/bool/rec/ok/err and errors ("cannot compare") on
  anything else. Seq needs a value kind plus structural `vEq`
  support for test/linkage comparison, not a new language equality
  operator.
- Brand rules under test: executable seals mint only their own
  module's brands (`compiler/types.go:330-339`, CAN6004
  `compiler/code.go:80`); mismatch is CAN6003
  (`compiler/code.go:79`); inconsistent scripted results are CAN3110
  (`compiler/code.go:53`); untaken arms are CAN4107
  (`compiler/code.go:70`).
- Catalogue contract (`docs/archive/ASTRA_STDLIB.md:233-243`): immutable
  ordered sequences; `seq__empty`/`seq__singleton`,
  `seq__length`, `seq__get`, `seq__append`/`concat` carry
  `Collections + Types`. Customers needing Seq: `str__split`/`join`
  (226-227), `attributes__make` (326), `fragment__join` (329).
- Precedents: a28 (this file's shape: P0 first, rollback section,
  open questions); a15 (brand ownership); a18 (scripted-not-executed
  linkage: `vEq`-incomparable results are trusted, so Seq must be
  `vEq`-comparable or its tests are weak evidence).

## Constraints And Non-goals

- Values and literals only in v1 of this slice. No `length`, `get`,
  traversal, `append`/`concat`, `slice`, maps, sets, callbacks,
  generics beyond the admitted element types, comparators, or
  sequence equality surface.
- No normalization or silent repair: order, empties, and duplicates
  are preserved exactly (split/join contract needs this; it is not
  HTML's omission rule wearing another coat).
- No HTML composition and no retained-name observation. Preserving
  brands inside Seq does not authorize concatenating them or reading
  attribute names; those need their own first-party mechanism.
- No `REQUIREMENTS.md` edit here: this doc is the proposal; any
  amendment lands only if the design is accepted.

## Key Decisions

1. Typed literal `Seq<T>[...]`, element type always written
   (`Seq<str>[]`, `Seq<str>["a", ""]`). Rejected bare `[...]`
   values: that shape already means script rows, and
   `checkStubs`/`evMatch` depend on it — a bare list in a value
   position is CAN6007, pointing at the typed form. Rejected
   empty/singleton-constructor-only construction: without literals
   (or append, a later slice) arbitrary contents are inexpressible.
   Omitting `T` is a compile error (CAN6007 for bare lists;
   `CodeParse` for malformed `Seq<` shapes, which never cascade
   into binops). A valid head inside a larger expression
   (`Seq<str>["a"] == ...`) reparses parenthesized, so operator
   precedence is the normal cascade's; a trailing index parses and
   then fails the str-only index rule (S3 owns real indexing).
2. Values plus element checking ship as one admission boundary, not
   two shippable features. Rollback removes both together; values
   never linger while checking is reverted.
3. Element checking applies uniformly in body literals, call
   arguments, record fields, test arguments, expected results, and
   `given` exchange arguments/results. Entering a Seq literal does
   not reset `exec`: foreign seals inside executable literals stay
   CAN6004; the same seal in test/given data stays accepted checked
   data.
4. `vEq` gains structural Seq comparison (length + ordered element
   comparison) as test-evaluator support. Not a language operator;
   without it the new rows fall through the a18 "cannot compare,
   therefore trust" path.
5. Uniform element rule: any known non-sequence type may be the
   element type (base types, brands, declared records). A
   str-and-brands-only restriction would have been extra machinery
   with its own code for no customer need; the committed rows pin
   str and brands, and broader representations ride the same
   evaluator/emit paths with no HTML brand names hard-coded into
   the checker. Nesting (`Seq<Seq<str>>`) is rejected: not a v1
   shape in annotations, literals, or values.
6. Bare-Seq returns are rejected now (T8, mirroring the a26
   bare-brand rule). Expectations must be `Ok(...)` or an error
   kind, so a function returning a bare sequence could never be
   tested; entries return wrapper records. A future customer slice
   may amend this explicitly if it carries its own return
   convention.

## Work Plan

1. P0 — Proposal (this file). No code. Get `Approve` / changes /
   cancel.
2. P1 — Parse + AST. New typed-literal form; bare `[...]` keeps its
   scripting meaning. Comma splitting reuses the string/paren-aware
   splitter; members containing commas and Seq nested inside
   exchanges get parsing rows. Surfaces: `compiler/parse.go`,
   `Small`/`Value` shapes, `walkCalls`-class readers.
3. P2 — Types. `Seq<T>` admission in `knownType`; element checking
   at every value position above; `exec`-preserving recursion into
   literals; CAN6003 mismatches (including empty-typed mismatch);
   CAN6004 foreign seals in executable literals.
4. P3 — Eval + `vEq`. Sequence value kind; literal evaluation
   preserving order/empties/duplicates; structural `vEq` so
   false-success rows fail with CAN3110 instead of being trusted.
5. P4 — Emit. TS representation for Seq values in the positions P2
   admits; no new runtime helpers beyond what literals need.
6. P5 — Goldens + tooling + docs. Committed decision-table rows
   below; `gramcheck` prose only if the literal form needs a new
   scope; README + slice doc updated; generated files committed
   alongside sources.

## Validation Plan

Notation is semantic, not executable syntax: `S[...]` means
explicitly typed `Seq<str>`; `H(x)`/`A(x)` mean
constructor-produced `Html__Safe`/`Html__Attribute` with exact
serialization `x`; `Ok(payload)` is the catalogue's named success
record. "Terminal" means no match arm exists; do not manufacture
branches to claim coverage.

Construction fixtures (all succeed, no emitted error):

| Row | Construction | Expected | Arm |
| --- | ------------ | -------- | --- |
| V0 | Empty string-sequence literal | `Ok(S[])` | Terminal |
| V1 | Singleton empty string | `Ok(S[""])` | Terminal |
| V2 | Members `"b"`, `""`, `"a"`, `"b"` | `Ok(S["b", "", "a", "b"])` | Terminal |
| V3 | Identity on `Seq<Html__Safe>[H("A"), H("")]` | Same typed sequence | Terminal |
| V4 | Identity on `Seq<Html__Attribute>[A(""), A("disabled")]` | Same typed sequence | Terminal |
| V5 | Result record containing `S["a", ""]` | Same record and contents | Terminal |

V0 and V1 must be observably distinct. V2 rejects dropping
empties, reordering, or deduplicating.

Compile-rejection rows (no runtime arm; reject before execution):

| Row | Fixture | Required result |
| --- | ------- | --------------- |
| T1 | Explicit `Seq<str>` containing an `int` | CAN6003 |
| T2 | `h: Html__Safe` in `Seq<str>` | CAN6003 |
| T3 | `a: Html__Attribute` in `Seq<Html__Safe>` | CAN6003 |
| T4 | `Seq<Html__Safe>[]` where `Seq<str>` required | CAN6003 despite emptiness |
| T5 | Foreign-brand `seal` inside an executable Seq literal | CAN6004 |
| T6 | Same correctly typed foreign-brand seal in test data | Accepted as checked data |
| T7 | Element type omitted from the literal (bare `[...]`) | CAN6007, pointing at `Seq<T>[...]` |

Run T1–T3 in body literals, call arguments, record fields, test
arguments, expected results, and `given` arguments/results
(separate checking paths; literals-in-bodies alone is
insufficient).

Linkage rows (close the trust fallback):

| Fixture | Expected |
| ------- | -------- |
| Foreign provider computes `Ok(S["a", ""])`; script claims `Ok(S["a"])` | CAN3110 |
| Provider computes `Ok(S["a", "b"])`; script claims `Ok(S["b", "a"])` | CAN3110 |

Per slice: `go test -count=1 ./...`, `go run ./tools/modcheck`,
`go run ./tools/gramcheck` (forced, never cached).

## Risks / Rollback

- Literal syntax collides with script lists or comma splitting.
  Mitigation: new spelling keeps bare `[...]` untouched; P1
  carries comma-in-member and nested-in-exchange parsing rows.
- `vEq` structural comparison mis-specifies order/emptiness.
  Mitigation: V2 plus the CAN3110 rows pin order, empties, and
  duplicates; declarations alone are not accepted as evidence.
- Rollback: revert Seq consumers first (none in this slice by
  construction), then Seq admission across checking, evaluation,
  emission, tests/scripts, and declarations together with the P5
  goldens and doc claims. Never retain values while reverting
  element checking. Update the slice doc and README, regenerate
  outputs, rerun the three gates. Generated artifacts must not
  retain interfaces removed from source.

## Open Questions

- Resource envelope for large literals: termination is static;
  sandbox depth/host stack bounds still need conformance evidence,
  no silent truncation.
- Whether a future customer slice needs bare-Seq returns with its
  own return convention (see decision 6): rejected for now, amend
  explicitly if so.

Resolved in implementation: literal spelling is `Seq<T>[...]`
(decision 1, with the parenthesized-head embedding rule);
omitted-`T` diagnostics are CAN6007/`CodeParse`; no registry,
README, emitter, or conformance surface beyond `code.go` (CAN6007),
`docs/README.md`, and `compiler/seq_s1_test.go` changed — S1
needs no generated files.
