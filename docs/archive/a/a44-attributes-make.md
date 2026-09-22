# a44 — attributes__make, customer 4 (shipped)

Status: shipped. 235 html tests green; three gates green;
goldens regen (`html.ts`, `errors.json` gains
`html.duplicate_attribute` with all 5 dup rows in
`hit_by_tests`).

## Goal

Assemble retained attributes with single-space separators and
no first-wins/last-wins ambiguity:
`html__attributes__make(items: Seq<Html__NamedAttribute>) ->
Html__AttributesResult ! [html.duplicate_attribute]`. Last
customer of the workstream.

## Success Criteria

- Join law rows green: omit empty contributions, separators
  only between retained entries, duplicates judged among
  retained names (nonadjacent and different-value cases).
- Duplicate payload names the first-repeated retained name
  (`[id, title, title, id]` reports `title`).
- Goldens regen (`html.ts` extended; `errors.json` gains
  `html.duplicate_attribute`); three gates green.

## Context And Current Facts

- a43 gives name-carrying items; emptiness is brand `==`
  against the empty seal (verified working); same-brand `+`
  (a42) assembles; `seals_from` promotion (a26,
  `Html__Safe` from `Html__Text`) is the declared vehicle for
  brand-to-brand minting.
- Catalogue: no first-wins/last-wins ambiguity; Collections +
  Brands (`docs/archive/ASTRA_STDLIB.md:326`). Input shape amended by
  a43 verdict B (name-carrying items, not bare attributes).

## Constraints And Non-goals

- One operation: make (plus its two helpers, like every
  `_from` pair before it).
- Complete attributes with empty VALUES are retained
  (`title=''` is not absent — only the empty contribution is).
- Empty contributions leave no ghost name for duplicate
  detection (skipped before any name is kept).
- No `REQUIREMENTS.md` edit: proposal only.

## Key Decisions

1. Accumulator is `Html__Attribute` (same-brand `+` needs one
   brand): first-retained replaces the empty seed, later ones
   append `seal Html__Attribute(" ") + item`. Leaf arms seal
   up via declared `seals_from [Html__Attribute]` on the new
   `brand Html__Attributes is str rev 1` — explicit, one-way,
   same-module promotion, no parsing anywhere.
2. Duplicate scan is a `contains` helper over retained names
   (`==` on brands, `decreases` worker), not inline indices:
   one worker, one proof each.
3. First-retained test is `#kept == 0` (positional, like
   join's `position == 0`): the brand accumulator cannot be
   sniffed for emptiness, which is exactly why the kept-names
   sequence doubles as the first flag.
4. `error html.duplicate_attribute(name:
   Html__TextAttributeName)`: brand payload (no `str` leak),
   first-repeated-occurrence semantics.
5. `Html__AttributesResult(attributes: Html__Attributes)`
   return wrapper, mirroring `Html__AttributeResult`.

## Work Plan

1. P0 — Proposal (this file). No code.
2. P1 — Error decl + brands/records + `contains` + `make_from`
   + `make` with rows; extend `provides` and module `emits`.
3. P2 — Regen goldens; README lines; gates; commit.
4. P3 — Workstream closeout: round-trip audit across join +
   split + fragment + make; goal complete.

## Validation Plan

| Contributions | Expected |
| --- | --- |
| `[]` | `""` (end) |
| `[E, E]` | `""` (skip, end) |
| `[E, disabled]` | `"disabled"` (skip, first, end) |
| `[E, disabled, E, id='x', E]` | `"disabled id='x'"` |
| `[title='']` | `"title=''"` (first, end — retained) |
| `[title='', title='x']` | `duplicate(title)` |
| `[id='a', disabled, id='b']` | `duplicate(id)` |
| `[disabled, disabled]` | `duplicate(disabled)` |
| `[id, title, title, id]` | `duplicate(title)` (first repeated) |

(`E` = empty contribution with a name; names differ per row to
also prove ghosts are left by skipped items: a skipped `id`
followed by retained `id` must NOT report duplicate — add that
row: `[id-empty, id='a']` → `"id='a'"`.)

`contains` rows: found, missing, empty names, zero budget.

## Risks / Rollback

- Risk: `seals_from` on a fresh brand looks like a back door.
  Mitigation: same-module, one-way, exact, declared — the a26
  properties verbatim; rows pin the assembled values.
- Rollback: remove decls/worker/entry, restore `provides`
  and module `emits`, regen goldens, rerun gates.

## Open Questions

- None. After this slice the workstream closes with the
  acceptance audit.

## Shipped Notes

- Built as proposed: `html__attributes__contains` (found /
  missing / empty / zero-budget rows) + `html__named_item_at`
  (S3 checked-access reuse) + `make_from` (skip / first /
  later / dup arms, three error-relay arms) + `make` entry
  (`fuel = #items + 1`). `Html__Attributes` brand seals from
  `Html__Attribute`; leaf arms seal up via `seal
  Html__Attributes(acc)`.
- Decision rows shipped (11 `make` rows): `empty`,
  `two_empties`, `omit_first`, `full_mix`,
  `empty_value_retained` (`title=''` retained), `ghost_no_double`
  (skipped `id` leaves no ghost: `[id-empty, id='a']` →
  `"id='a'"`), `dup_title`, `dup_id_nonadjacent`,
  `dup_disabled`, `dup_after_skip`, `dup_first_repeated`
  (`[id, title, title, id]` reports `title`).
- Correction to the earlier finding: the untaken-relay
  exemption is deliberate design, not a hole — the
  identity-relay certificate (`relayStatus` in `compiler/lsp.go`)
  exempts a bound error arm of a local call whose body is
  exactly the same-kind reconstruction with unchanged fields.
  Three interlocking checks make it sound: exhaustiveness
  (`verifyExhaustiveAll`) forces the arm to exist exactly when
  the callee emits the kind (missing/stale arms are errors),
  the certificate proves transparency, and shadowed or foreign
  relays stay under the execution law (4107). Pinned by
  `TestDiagnoseValidRelayExempt`. `dup_after_skip` is kept as
  a semantic pin (skip+dup interaction, first-repeat through a
  skip), not a coverage requirement.
