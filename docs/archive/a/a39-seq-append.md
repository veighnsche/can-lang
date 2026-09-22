# a39 — Seq append, S4

Status: shipped. P0 approved under standing approval; P1–P4
implemented and green: `Seq<T> + T` across checker (member rule,
concat refusal), evaluator (copy-on-append), and emitter
(spread), with rows in `compiler/seq_s4_test.go`. No deviations.

## Goal

Immutable, order-preserving append for computed construction:
`xs + x` is a new sequence with `x` after every member of `xs`;
`xs` itself is unchanged. This is what makes an ordinary `.can`
`split` possible (arbitrary field counts from input, which
literals alone cannot denote). It unlocks no customer on its own.

## Success Criteria

- `Seq<T> + T` evaluates to a new `Seq<T>`; the original value
  is observably unchanged (aliasing row: two appends to one base
  share nothing).
- Order, empties, and repeats preserved; no duplicate, empty, or
  content validation — append is total over well-typed members.
- Wrong-type members and sequence-to-sequence `+` stay compile
  errors; existing arithmetic keeps its diagnostics verbatim.
- `go test -count=1 ./...`, `modcheck`, `gramcheck` green.

## Context And Current Facts

- Calls ride match scrutinees only (CAN3003, `check.go:468`);
  a split worker carries its accumulator through call ARGUMENTS,
  which are value positions. So append must be a value-form
  operator, not a kernel call: a `seq__append` kernel would force
  a match tower per appended member.
- `+` is already the one construction operator (a16: add for
  ints/decs, concat for strings). `Seq<T> + T` extends that
  family; the emitter already lowers per-operand-type.
- Values are immutable elsewhere (append copies); `vEq` compares
  structurally, so original-vs-appended assertions verify in
  both evaluator and linkage paths.

## Constraints And Non-goals

- One operation: append of one member. No concat (`Seq+Seq`),
  no slice, no prepend/insert/remove, no customers.
- Concat stays rejected with its own diagnostic (cut, not
  oversight): split needs append only, and `+` doing three jobs
  needs three contracts. A later slice may bless concat without
  touching this one.
- No `std/seq` module yet (same reasoning as S3): append needs
  no `.can` wrapper. First customer slice creates the module.
- No `REQUIREMENTS.md` edit: proposal only.

## Key Decisions

1. `xs + x` with `xs: Seq<T>`, `x: T`. Rejected kernel call
   (above). Rejected new keyword syntax (a third value form
   for one operation).
2. `Seq + Seq` is CAN6003 (`cannot add ... : sequence
   concatenation is not in v1`), not silent append-of-either.
   `T + Seq` (member on the left) is the existing
   no-implicit-conversions CAN6003.
3. TS lowering is spread: `[...xs, x]`, element type precise.
   Evaluator copies `Arr` (never shares the tail array).
4. Empty appends are ordinary: `S[] + "a"` is `S["a"]`,
   `S["a"] + ""` keeps the empty member.

## Work Plan

1. P0 — Proposal (this file). No code.
2. P1 — Checker: `+` admits `Seq<T> + T` yielding `Seq<T>`
   (member mismatch is the existing CAN6003 path); `Seq+Seq`
   gets its own diagnostic; all other `+` behavior verbatim.
3. P2 — Eval: copy-and-append; wrong-type members cannot
   arrive (checker owns them), out-of-range is impossible.
4. P3 — Emit: `[...a, b]` spread.
5. P4 — Rows below in `compiler/seq_s4_test.go`; README index
   row; three gates; commit.

## Validation Plan

Fixture shape: functions returning BOTH sequences so mutation
cannot hide behind a correct final result
(`Ok(before = ..., after = ...)`).

| Original, item | Expected original | Expected appended |
| --- | --- | --- |
| `S[], "a"` | `S[]` | `S["a"]` |
| `S["a"], ""` | `S["a"]` | `S["a", ""]` |
| `S["b", "", "a"], "b"` | unchanged | `S["b", "", "a", "b"]` |

Aliasing: append `"b"` and `"c"` separately to one `S["a"]`;
expect `S["a"]`, `S["a","b"]`, `S["a","c"]`.

| Rejection | Expected |
| --- | --- |
| `Seq<str> + 1` | CAN6003 member mismatch |
| `Seq<str> + Seq<str>` | CAN6003 concatenation-not-in-v1 |
| `1 + Seq<str>` | existing CAN6003, verbatim |
| branded `Seq<M__B> + seal` | clean, stays `Seq<M__B>` |
| branded `Seq<M__B> + "x"` | CAN6003 |

Emit: `[...xs, x]` shape asserted on a branded append (erasure
visible).

## Risks / Rollback

- Risk: `+` doing three jobs confuses readers. Mitigation: the
  catalogue already scopes `+` per operand type; the
  concatenation refusal names itself.
- Rollback: revert the `+` seq arm in checker/eval/emit plus
  the test file; string/int/dec `+` byte-identical.

## Open Questions

- None for S4. The first customer slice owns `std/seq`
  creation and the canonical `sequence.*` namespace.
