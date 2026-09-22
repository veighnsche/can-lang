# a40 — str__join, customer 1

Status: shipped. P0 approved under standing approval; P1–P3
implemented and green: `std__str__join_from` + `std__str__join`
in `std/text/text.can` with the committed rows, goldens
regenerated (`text.ts` extended; `errors.json` byte-identical —
no new kinds, as predicted). No deviations.

## Goal

Join a string sequence with a separator, preserving order:
`std__str__join(values: Seq<str>, separator: str) -> str`. First
`.can` customer of the Seq core (S1–S4); no new compiler surface.

## Success Criteria

- Decision rows below all green in `std/text/text.can`
  (producer-owned outcomes, no normalization).
- First-element detection is positional (`position == 0`), never
  `acc == ""` (which misfires when the first field is empty).
- Empty separator is ordinary (concatenation); empty sequence is
  `""`. No emits entries.
- Goldens regenerated (`text.ts`, `errors.json` — no new error
  kinds, so `errors.json` should be byte-identical) and
  `TestGoldenStdText` green.

## Context And Current Facts

- S1–S4 give literals, `#`, `[]`, and `+` over `Seq<str>` —
  everything join needs. Same-file helper recursion with
  `decreases` is the house worker shape (`_from`/`find_from`
  precedent in `std/text/text.can`).
- Catalogue: `std__str__join` preserves sequence order, Text +
  Collections (`docs/archive/ASTRA_STDLIB.md:227`).
- Round-trip law (ratified): `join(split(s, sep), sep) = s` for
  nonempty `sep`. The reverse is NOT promised (join is not
  injective: `[]` and `[""]` both yield `""`).

## Constraints And Non-goals

- One operation: join. Split is the next slice with its own P0.
- No normalization, no trimming, no dedup, no empty-field
  skipping: `["b", "", "a"]` joined with `/` is `"b//a"`.
- No `REQUIREMENTS.md` edit: proposal only.

## Key Decisions

1. Worker `std__str__join_from(values, separator, position,
   fuel, acc)` with `decreases fuel`; entry passes
   `position=0, fuel=#values+1, acc=""`. Arms: `B` (fuel
   exhausted), first (`position == 0`), more (separator +
   member), end (position reached length). Full runs end in
   `end` with fuel left; `B` is witnessed by a direct
   zero-budget row (S3 invariant, not re-argued here).
2. Separator used verbatim, including `""`.

## Work Plan

1. P0 — Proposal (this file). No code.
2. P1 — Append `std__str__join_from` + `std__str__join` with
   rows to `std/text/text.can`; extend `provides`.
3. P2 — Regen: `go run ./compiler --out std/text
   std/text/text.can`; confirm `errors.json` unchanged in
   meaning (no new kinds); update `std/text/README.md`.
4. P3 — Gates (`go test -count=1 ./...`, `modcheck`,
   `gramcheck` — the golden test recompiles text.can);
   commit with generated files.

## Validation Plan

| values, separator | Expected | Arms |
| --- | --- | --- |
| `S[], "/"` | `""` | `B` via entry? No — entry fuel=1, position 0: `P, end`. `B` by zero-budget row |
| `S[""], "/"` | `""` | first, end |
| `S["", ""], "/"` | `"/"` | first, more, end |
| `S["b", "", "a"], "/"` | `"b//a"` | first, more, end |
| `S["a", "", "b"], ""` | `"ab"` | first, more, end |
| direct `_from(..., fuel=0)` | `acc` unchanged | `B` |

## Risks / Rollback

- Risk: `+` on strings inside the worker confuses with Seq
  append. Mitigation: operand types decide; rows pin both.
- Rollback: remove the two functions, restore `provides`,
  regen goldens, rerun gates.

## Open Questions

- None for join.
