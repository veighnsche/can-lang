# a37 — Seq length, S2

Status: shipped. P0 approved under standing approval; P1–P4
implemented and green: `#` extended to `Seq<T>` operands
(checker/eval/emit), rows in `compiler/seq_s2_test.go`. One
deviation: the operator scanners (`findTop`/`findLastTop`) skip
`Seq<...>[...]` heads as atoms, so `#Seq<str>[...]` and leading-seq
binops never fracture at `<` — no design change, parser hygiene
the S1 embedding rule already implied.

## Goal

Element count for sequences, including zero. This supplies the
explicit traversal bound S3 needs (`fuel = #xs + 1`); it unlocks no
customer on its own.

## Success Criteria

- `#xs` where `xs: Seq<T>` evaluates to the element count as `int`.
- Empty is `0`; empties, repeats, and multi-scalar members each
  count once (elements, not scalars, not nonempty contributions).
- Non-sequence operands keep the existing rule byte-identical
  (`cannot count scalars of ...`, pinned in `lsp_test.go:1079`).

## Context And Current Facts

- `#` is the one length spelling (`compiler/parse.go` strlen,
  `compiler/types.go` value/stridx family, `compiler/eval.go`
  scalar count, `compiler/emit.go` `BigInt([...s].length)`).
- S1 gives `Seq<T>` static types (`typeOf` seqlit) and runtime
  values (`Value.Arr`), so length is a three-site extension:
  checker operand rule, evaluator count, emitter operand rule.
- Catalogue `std__seq__length` (`docs/archive/ASTRA_STDLIB.md:240`) is
  realized by `#`, not by a second `seq__length` function: a
  same-file wrapper would be a second spelling with zero new
  semantics, against the one-spelling rule (same argument as a16
  `+` doing add and concat).

## Constraints And Non-goals

- One operation. No `get`, traversal, append, or customers.
- No unit confusion: `#` on `str` counts scalars (unchanged); on
  `Seq` counts elements. Mixed or unknown operands keep failing
  exactly as today.
- No `REQUIREMENTS.md` edit: proposal only.

## Key Decisions

1. Extend `#`, not a `seq__length` kernel call. A kernel would
   need call-resolution plumbing (`isStoreOp`/`isDecParts`
   analogues across check/types/eval/emit) for identical
   semantics. Rejected.
2. Length is total: no empty-sequence error, no emits entry.
3. Brand erasure is irrelevant here (length never inspects
   members), but branded sequences are still pinned by a row.

## Work Plan

1. P0 — Proposal (this file). No code.
2. P1 — Checker: `strlen` accepts `Seq<T>` operands (`s.T=int`
   unchanged); everything else keeps the pinned diagnostic.
3. P2 — Eval: `seq` values count `len(Arr)` as `int`.
4. P3 — Emit: operand rule admits `Seq<T>`; lowers to
   `BigInt(<arr>.length)` like strings.
5. P4 — Rows below committed in `compiler/seq_s2_test.go`; README
   index row; three gates; commit.

## Validation Plan

| Input | Expected | Arm/row |
| --- | -------- | ------- |
| `Seq<str>[]` | `Ok(0)` | zero, not a failure |
| `Seq<str>[""]` | `Ok(1)` | singleton empty counts |
| `Seq<str>["", ""]` | `Ok(2)` | empties count |
| `Seq<str>["abc", "𝌆"]` | `Ok(2)` | elements, not scalars |
| `Seq<str>["b", "", "a", "b"]` | `Ok(4)` | repeats count |
| `Seq<M__B>[seal...]` (2 members) | `Ok(2)` | brands count |
| `#5` | `cannot count scalars of int` (unchanged) | existing pin |
| `#` over `bool` | same family | existing rule |

Each length row is a tested function returning a record with an
`int` field (`test ... => Ok(n = #...)`); a miscount fails the
test run (CAN4200), so clean means computed.

## Risks / Rollback

- Risk: `#` overloading confuses scalar vs element units.
  Mitigation: the catalogue already separates them by operand
  type; the diagnostic names scalars only for non-Seqs.
- Rollback: revert the three operand-rule arms and the test file;
  string length behavior byte-identical before and after.

## Open Questions

- None for S2. S3 owns whether `#xs + 1` fuel reads well enough
  to keep, or wants a named alias later (no second spelling now).
