# a43 — named attribute wrappers

Status: shipped. P0 approved under standing approval; P1–P2
implemented and green: `Html__NamedAttribute` record + result,
five relay wrappers (text/boolean/id/href/src, 14 rows
including NUL/error relays), goldens regenerated. No
deviations. a44 owns `contains` + `make`.

## Goal

Give every shipped attribute maker a named twin that pairs the
minted `Html__Attribute` with its `Html__TextAttributeName` by
construction, so `attributes__make` (a44) can judge duplicates
among retained names without ever parsing a serialization.

## Fork resolution (from a42)

Verdict for **B (name-carrying items)** over A (name-projector
kernel), for three reasons:

1. The module's own precedent is reconstruction, not
   extraction: `html__attribute__spelling` rebuilds the
   spelling from the closed admitted domain instead of parsing
   the brand. Carrying the name alongside follows the house
   pattern; parsing serializations in a kernel invents a second
   source of name truth that drifts from makers.
2. B needs zero compiler surface (records, brand `==`, and
   same-brand `+` all shipped); A needs a first-ever brand
   observer plus a a15 amendment.
3. The verdict's own warning stands: caller-built pairs admit
   mismatched metadata. So the pairs are minted by wrapper
   makers, never by hand — this slice.

Catalogue consequence: `html__attributes__make` takes
`Seq<Html__NamedAttribute>` (amendment to the `Seq<Html__Attribute>`
line, recorded here and in `ASTRA_STDLIB.md` if it tracks the
row). Shipped bare makers are untouched.

## Success Criteria

- `Html__NamedAttribute(name, attribute)` record; five thin
  wrappers relaying the five shipped makers
  (text/boolean/id/href/src) and pairing correctly.
- Each wrapper pinned by rows; a mismatched pairing is
  unwritable by construction (no row can even express it —
  the pairing happens inside the wrapper body).
- No new error kinds; goldens regen; three gates green.

## Context And Current Facts

- Shipped makers return `Html__AttributeResult(Ok(attribute =
  seal Html__Attribute(...)))`; each takes a brand-typed name
  or validates one. Wrappers relay via `match call` and re-pair
  the name the caller already holds.
- Brand `==` (erased, verified) and same-brand `+` (a42) are
  the only brand operators a44 needs beyond this slice.

## Constraints And Non-goals

- One operation family: named construction. No duplicate
  logic, no `attributes__make`, no new brands.
- No change to shipped makers, their rows, or the absence law.
- No `REQUIREMENTS.md` edit: proposal only.

## Key Decisions

1. Record (transparent, no validation needed — validity comes
   from the wrapper relay, and bare construction pairs caller
   values the wrapper pattern discourages but cannot forbid;
   the duplicate rows in a44 use wrapper output only):

   ```text
   type Html__NamedAttribute rev 1 (
     name: Html__TextAttributeName,
     attribute: Html__Attribute
   )
   type Html__NamedAttributeResult rev 1 (
     item: Html__NamedAttribute
   )
   ```

   Hmm — wrapper return shape: makers return `XResult(Ok(...))`.
   A named wrapper returns
   `Ok(item = Html__NamedAttribute(name = name, attribute =
   a.attribute))` under a new result record, mirroring
   `Html__AttributeResult`. Decided: yes, `Html__NamedAttributeResult`.
2. Five wrappers, one per maker kind. Each is a relay (no logic
   copy): on maker error, forward the error unchanged
   (nul_byte, invalid_identifier, invalid_url); on Ok, pair.
   Error-forwarding arms need rows (nul case for text; bad id;
   bad url) or coverage fires.
3. Boolean wrapper takes `(name, present)` like its maker;
   absent booleans pair the EMPTY attribute with the name —
   a44 omits them by the emptiness rule (name retained for
   nothing). This is correct: absence is per-item, omission is
   per-assembly.

## Work Plan

1. P0 — Proposal (this file). No code.
2. P1 — Record + result + five wrappers with rows in
   `std/html/html.can`; extend `provides`.
3. P2 — Regen goldens; README line; gates; commit.

## Validation Plan

Per wrapper: one Ok-pairing row (name and serialization both
pinned) plus each maker-error relay row the maker owns:

| Wrapper | Rows |
| --- | --- |
| text | plain pair; nul_byte relay |
| boolean | present pair; absent pair (empty attr + name) |
| id | valid pair; invalid_identifier relay |
| href | valid pair; invalid_url relay |
| src | valid pair; invalid_url relay |

## Risks / Rollback

- Risk: record pairs invite hand-built mismatches. Accepted:
  constructors-of-record are transparent everywhere in the
  language; the a44 rows use wrapper output, and no row blesses
  a hand pair.
- Rollback: remove record/result/wrappers, restore
  `provides`, regen goldens, rerun gates.

## Open Questions

- None. a44 owns `contains` + `make_from` + `make` +
  `html.duplicate_attribute`.
