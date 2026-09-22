# a91: S2 closed checks — design (decided, bounded by a90)

Status: decided 2026-09-18 within the a90 bounds; S2a
implements here, S2b follows. No separate verdict round:
a90 already recorded the admission verdict, and this doc
fills exactly its bounds — anything outside them needs a
new decision, not a broad reading of this one.
Parent: `docs/a/a90-schema-s2-exhibits.md` (exhibits 1–5);
`docs/a/a89-schema-validation-design.md` (proof list).

## What S2 is (honest split)

Exhibits 3 (payload reconstruction), 4 (per-consumer
scripting), and the reusable half of 5 (token formats)
are *interpretation*: one constraint in, frozen triple
out. That centralizes today, with no new language
machinery. Exhibits 1 (per-field binding) and 2
(fail-fast ordering across fields) need record traversal
and stay per-consumer until S3 — this design does not
claim them. A third consumer built on S2 still writes
its own binding chain; what it stops rewriting is every
reconstruction arm, every converter script, and every
token format.

## The closed vocabulary

Exactly three constraint records (ordinary revisioned
records — no `Seq<Variant>` fields) and three check
operations, all in `std/quota/quota.can` (ruling 4):

```text
Validate__Length(path, tag, minimum, maximum)
  std__validate__length_check(value: str, constraint)
    -> Str__Value ! [validation.schema_violation]

Validate__Range(path, tag, lower, upper)
  std__validate__range_check(value: int, constraint)
    -> Int__Value ! [validation.schema_violation]

Validate__Membership(path, allowed: Seq<str>)
  std__validate__membership_check(value: str, constraint)
    -> Str__Value ! [validation.schema_violation]
```

Each op owns its producer call, its malformed-policy
detection, and its frozen-triple construction:

| Op | Malformed policy | Data violation | Success |
| -- | ---------------- | -------------- | ------- |
| length | minimum < 0 → `V("", "schema."+tag+"_minimum", render(minimum))`, before any producer call; reversed bounds → `V("", "schema."+tag+"_bounds", "minimum="+render(min)+";maximum="+render(max))` | `V(path, "str.length_scalars", value)` | Original string |
| range | reversed bounds → `V("", "schema."+tag+"_bounds", "lower="+render(lo)+";upper="+render(hi))` | `V(path, "int.closed_range", render(value))` | Original int |
| membership | none — empty `allowed` is a valid deny-all (P1) | `V(path, "str.one_of", value)` | Original string |

Complete rule dispatch is structural: every producer
outcome has its arm (`str_length`: invalid_bounds /
invalid_length / Ok; `int_range`: invalid_bounds /
out_of_range / Ok; `one_of`: not_allowed / Ok), each
with its row. Producer binders stay `_` (S1a amendment
2); only `Ok` binders are read.

## Rulings

1. **Tags and paths are parameters, pinned per
   consumer.** `tag` builds the policy-token infix
   (`"schema." + tag + "_bounds"`); `path` is the data
   path verbatim. Both are plain strings — a wrong tag
   is well-typed — so each consumer's decision tables
   pin its exact bytes (the check ops' own tables pin
   the formats with two distinct tags). No closed tag
   universe: that would be a string-enum mechanism in
   disguise, unasked for by any exhibit.
2. **Fail-fast within an op, ordering across ops stays
   out.** Length checks minimum-nonnegative before
   calling its producer (mirrors both pilots; row
   `len_min_negative_beats_data` pins it). Field order
   across checks remains each consumer's chain —
   exhibit 2 residual, owned by S3.
3. **No recursion, no new termination argument.** Check
   ops call producers and the public converter only;
   recursion stays inside those bodies. Domain: total
   converter, exhaustive producer matches.
4. **Placement: `quota.can`, `Validate__` types,
   `std__validate__` verbs.** The validation family
   already lives there; same-file producer calls keep
   the new `given` surface to the converter alone.
   `std/schema` stays the asset minimum (a90); a
   future `std/validate` split may move the whole
   family, not this slice.
5. **Mismatch honesty.** "Schema/record mismatch" has
   no runtime check in S2: binding correctness is
   evidenced per consumer by decision tables (a wrong
   check for a field fails its rows), exactly as in
   the pilots. Centralized mismatch rejection needs
   traversal and is S3 work.

## Proof mapping (a89 list)

Closed bindings: the three records fix
(path, tag, bounds) per call. Correct field types:
the checker (wrong-typed value is `CAN6003`).
Complete rule dispatch: structural, §tables above.
Deterministic ordering: within-op order frozen and
rowed; across-op order explicitly residual. No new
termination argument (ruling 3). Mismatch: per-
consumer tables (ruling 5).

## S2a / S2b split

- **S2a (this slice):** the three records + three ops
  with decision tables, goldens, linked vectors, node
  parity; §6 admission like any slice. No consumer
  changes — both pilots stay green untouched.
- **S2b (next):** a third consumer built on the check
  ops, then the a90-mandated cost comparison (S2b
  consumer cost vs consumer-2 cost: body lines,
  exchange count, new-proof burden) and the migration
  call for consumers 1–2. If the machinery exceeds
  the specialized-call cost it replaces, S2b says so
  and the check ops stand unmigrated.

## Non-goals

New constraint kinds; `dec`/`bool` fields; nested
descriptors; generic aliases; any compiler change;
migrating the pilots (S2b's call); the §1.5 row.
