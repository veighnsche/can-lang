# a41 — str__split, customer 2

Status: shipped. P0 approved under standing approval; P1–P3
implemented and green: `std__str__split_from` + `std__str__split`
returning new `Split__Result`, leftmost policy pinned,
`text.empty_separator` minted, goldens regenerated. No other
deviations.

## Goal

Split a string on a separator into ordered fields, retaining
empty fields: `std__str__split(value: str, separator: str)` with
`text.empty_separator` on empty separators. Second `.can`
customer; no new compiler surface.

## Success Criteria

- Decision rows below green in `std/text/text.can`, including
  the leftmost-overlap policy row and the empty-input rows.
- Round trip with join by paired fixtures (see decision 4).
- Goldens regenerated (`text.ts` extended; `errors.json` gains
  the `empty_separator` kind — correction from review: the
  catalogue named it but no module ever declared it, so this
  slice mints `error text.empty_separator()`).

## Context And Current Facts

- S1–S4 give everything split needs; join (a40) is its inverse
  partner. `replace_all_from` is the house left-to-right scan
  precedent; split consumes the head the same way.
- Catalogue: split retains empty fields, no implicit cleanup;
  Text + Collections (`docs/archive/ASTRA_STDLIB.md:226`).
- House wrapper convention: `.can` signatures return records
  (`length_scalars -> Int__Value` for catalogue `int`), so
  split returns a wrapper record, not a bare `Seq` (S1 T8
  stands). Join already takes bare `Seq<str>` params — params
  were never banned.

## Constraints And Non-goals

- One operation: split. No join changes, no HTML.
- No normalization: consecutive/leading/trailing separators
  yield empty fields, always.
- No `REQUIREMENTS.md` edit: proposal only.

## Key Decisions

1. Return `Split__Result(values: Seq<str>)` (new record,
   provided). Amends nothing: T8 banned bare-Seq RETURNS, and
   this complies by wrapping — the decision-6 amendment path
   a36 left open, taken in its mildest form.
2. Leftmost non-overlapping segmentation (consistent with
   `replace_all` left-to-right). `split("aaa", "aa")` is
   `["", "a"]`; the rightmost alternative (`["a", ""]`) is
   rejected by a committed row. This settles fork B.
3. Worker `std__str__split_from(value, separator, current, acc,
   fuel)` with `decreases fuel`; entry passes `current=""`,
   `acc=Seq<str>[]`, `fuel=#value+1` (every step consumes >=1
   scalar while `separator` is nonempty, so the bound holds).
   Arms: `B` (fuel out, defensive flush `acc+current`,
   unreachable from the entry — same precedent as join's `B`),
   finish (`value == ""` → `acc+current`), separator hit
   (new field), miss with room (`#value >= #separator` but no
   match), miss short (separator longer than remainder).
4. Round trip by paired fixtures, not execution: no
   in-language call composition exists in test expectations,
   so `split(",a,,", ",")` → `S["", "a", "", ""]` and
   `join(S["", "a", "", ""], ",")` → `",a,,"` pin both
   directions through the shared intermediate. The reverse law
   stays unpromised (join is not injective).
5. Empty separator checked at the ENTRY before anything else:
   `("", "")` and `("abc", "")` both raise
   `text.empty_separator` — no early empty-input success
   bypasses validation.

## Work Plan

1. P0 — Proposal (this file). No code.
2. P1 — Append worker + entry + `Split__Result` with rows to
   `std/text/text.can`; extend `provides`.
3. P2 — Regen goldens; update `std/text/README.md`.
4. P3 — Gates; commit with generated files.

## Validation Plan

| value, separator | Expected | Decisions |
| --- | --- | --- |
| `"", ","` | `S[""]` | finish |
| `",a,,", ","` | `S["", "a", "", ""]` | hit, miss, finish |
| `"abc", ","` | `S["abc"]` | miss, finish |
| `"", ""` | `text.empty_separator()` | reject (before finish) |
| `"abc", ""` | `text.empty_separator()` | reject |
| `" a&<b> ,X ", ","` | `S[" a&<b> ", "X "]` | miss, hit, finish (no cleanup) |
| `"aaa", "aa"` | `S["", "a"]` | leftmost policy, rejects rightmost |
| `"abc", "abcd"` | `S["abc"]` | miss-short, finish |
| direct `_from(..., fuel=0)` | flush `acc+current` | `B` |

## Risks / Rollback

- Risk: `B`-flush looks like silent truncation. Mitigation:
  unreachable from the entry by the fuel arithmetic
  (`#value+1` over >=1-scalar steps); documented as
  defensive, same as join's `B`.
- Rollback: remove worker/entry/record, restore `provides`,
  regen goldens, rerun gates.

## Open Questions

- None for split. HTML customers own the brand-composition
  hard stop (verdict §5), unchanged by this slice.
