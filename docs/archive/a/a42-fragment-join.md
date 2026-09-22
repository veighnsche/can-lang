# a42 — fragment__join, customer 3

Status: shipped. P0 approved under standing approval; P1–P3
implemented and green: same-brand `+` (checker-only, rows in
`compiler/seq_s5_test.go`), `Html__Children` + worker + entry
in `std/html/html.can` (11 rows, entry seeded from shipped
`fragment__empty`), goldens regenerated. Added the missing
`TestGoldenStdHtml` gate (no test froze `html.ts` before).
No deviations; the attributes fork below awaits its verdict.

## Goal

Join explicit children into one safe fragment:
`html__fragment__join(children: Html__Children) ->
Html__SafeResult`, deterministic order, no raw-string
concatenation at call sites. Plus, in the same slice, the one
checker amendment that unlocks it: same-brand `+`.

## Success Criteria

- Decision rows below green in `std/html/html.can`; empty
  children coincide with shipped `fragment__empty` (the entry
  calls it — composition has the empty fragment as identity,
  in code, not just prose).
- Same-brand `+` rows green in `compiler/seq_s5_test.go`
  (checker-only change; eval/emit already erase).
- Goldens regenerated (`html.ts`, `errors.json` — no new
  error kinds).
- The `attributes__make` fork below gets a verdict (owner):
  this slice implements the fork-independent part only.

## Context And Current Facts

- Children carry safety evidence as `Html__Safe` members; the
  verdict's smallest wrapper is an ordinary record
  `Html__Children(items: Seq<Html__Safe>)` — no validating
  maker, no extra invariant.
- Verified by probe (not yet committed): same-brand `==`
  checks clean today (erased comparison); same-brand `+` is
  refused (`cannot add M__B with M__B`, CAN6003). So fragment
  assembly is blocked on exactly one rule, while emptiness
  tests need no machinery at all.
- `Html__Safe seals_from [Html__Text]` exists; this slice adds
  no seals, no promotions, no observers.

## Constraints And Non-goals

- One operation (`fragment__join`) plus one operator rule
  (same-brand `+`). No name observation, no `attributes__make`
  implementation, no `std/seq` module.
- No change to shipped producers, the absence law, or the
  `disabled`-only... (four boolean names today — the join law,
  not the membership list, is what travels).
- No `REQUIREMENTS.md` edit: proposal only.

## Key Decisions

1. `B + B -> B` when `B` is a str-backed brand (checker
   only). Soundness: both operands are already minted (no new
   minting capability — the file-ownership audit is untouched);
   the result stays branded (sink rule intact: nothing flows
   to `str`); every value still roots in constructor calls
   plus concats. This narrows a16's conservative refusal with
   reason; eval/emit treat brands as strings already, so no
   runtime change exists to get wrong. Different brands, and
   brand/str mixes, keep the existing refusal verbatim.
2. Worker `html__fragment__join_from(children, position, fuel,
   acc: Html__Safe)` with `decreases fuel`; entry relays
   through shipped `html__fragment__empty` for the identity
   seed, then `fuel = #children.items + 1`. Arms: `B`, step
   (`acc + child`), end. `B` witnessed by a zero-budget row
   (S3 invariant).
3. `Html__Children` is a plain record; callers build it with a
   literal (`Html__Children(items = Seq<Html__Safe>[...])`).
   Empty children are a value, and `fragment__empty` stays the
   zero-case owner (a33).

## Work Plan

1. P0 — Proposal (this file). No code.
2. P1 — Checker: same-brand str-backed `+` yields the brand
   (both `typeOf` and `value` arms); all other `+` behavior
   verbatim. Rows in `compiler/seq_s5_test.go` (operator
   rule, not Seq-specific): brand+brand clean with brand
   result type; brand+str, str+brand, cross-brand refusals;
   int/bool `+` unchanged.
3. P2 — `Html__Children`, worker, entry with rows in
   `std/html/html.can`; extend `provides`.
4. P3 — Regen `html.ts`/`errors.json`; update
   `std/html/README.md`; gates; commit with generated files.

## Validation Plan

| Children | Expected fragment |
| --- | --- |
| `[]` | `H("")` (via `fragment__empty` seed) |
| `[H("")]` | `H("")` |
| `[H("A"), H(""), H("B-esc")]` | `H("AB-esc")` (order, empties kept) |
| `[H("B"), H("A")]` | `H("BA")` (order, not sorted) |
| `[H(" "), H("A")]` | `H(" A")` (space kept) |

No error is emitted; empty children need no separate branch.

## Risks / Rollback

- Risk: same-brand `+` reads as minting without audit.
  Mitigation: no `seal` site is added or bypassed; grep-seal
  audit unchanged; rows pin brand-in/brand-out.
- Rollback: revert the `+` brand arm, the worker/entry/type,
  restore `provides`, regen goldens, rerun gates. String and
  int `+` byte-identical throughout.

## Open Questions — genuine fork: attributes__make observation

`fragment__join` needs only concatenation. `attributes__make`
additionally needs, per retained item, (a) emptiness (SOLVED:
`item == seal Html__Attribute("")`, verified working today),
(b) the retained NAME for duplicate detection, and (c) the
serialization for output. (b) is unobservable: brands admit no
projection, and full-serialization `==` misses `id='a'` vs
`id='b'`. Same-file ownership does not help — the operators
simply do not exist.

> Bounded question: with `Seq<Html__Attribute>` fixed as the
> input shape, compare (A) a narrow first-party name
> projector `html__attribute__name(attr) ->
> Html__TextAttributeName` (brand→brand, no `str` involved;
> secrecy untouched since attribute text is rendered output;
> explicit like `seals_from`) against (B) a catalogue
> amendment making items name-carrying (parallel
> record-returning makers; shipped bare makers untouched).
> Reject A if brand observation cannot stay behind the
> brand-world boundary (any `str`-typed leak fails it).
> Reject B if the catalogue signature is frozen. Which route
> keeps fewer independently maintained semantic rules?

Recommendation: A (no catalogue churn, one declared rule).
But A is a new compiler kernel slice (a43+) AFTER the
verdict — this slice ships the fork-independent fragment
only, and `attributes__make` waits for the decision. The
a30/a31 join law (omit empties, separators between retained
only, duplicates among retained names) is uncontested and
carries over unchanged.
