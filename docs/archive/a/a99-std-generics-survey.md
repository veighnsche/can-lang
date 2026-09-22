# A99 — std-wide generics survey

Status: corrected 2026-09-19 (JEV verdict: forward-looking). The
inventories below describe PLANNED stdlib, not built stdlib: at the
time of writing, `std/` holds 8 module dirs (ascii, division, html,
quota, ratio, scalars, schema, text). The division rounding bundles,
the 13 codecs, and the outcomes Compare family do not exist yet in
the repo (verified: local tree, all branches, stash). Keep the tier
analysis for when they are built; do not cite the counts as audited.

Companion: G2 two-pass expansion landed green in
[a98](a98-generic-records-design.md) (Ratio__Value 3→1 pilot + golden/linked/parity).

Question: given all std planned so far (roadmap: ASTRA_STDLIB.md),
which modules will be generics candidates beyond the G2 pilot?

## Method (forward-looking)

Criteria applied to planned modules behind the open gates. Criterion:
will the module hand-specialize identical logic per payload type,
i.e. N copies of `X__Value` records / nominal wrappers whose bodies
differ only in the payload type? Re-audit against real sources once
built; counts below are roadmap claims, not evidence.

## Tier 1 — same shape as the G2 pilot (record-param candidates)

| module     | symptom (planned) |
|------------|-------------------|
| validation | `List__Invalids`, `Set__Invalids`, `Table__Invalids`, `TableRow__Invalids` — same invalids-carrier record × 4 collection types. G2-shaped when built. |
| division   | Planned per-rounding/per-type result bundles. Will need values-inside-records first (function-values / closures-over-values), else it multiplies per (type × rounding) instead of collapsing. |
| outcomes   | Planned `Compare_*` emission wrappers collapsing to one `Compare`. BLOCKED on bare `-> T` select: same root limit that forced select to stay reverted in G2. |

## Tier 2 — blocked on Values sharing architecture (planned)

`json`, `yaml`, `toml`, `msgpack`, `cbor`, `ron`, `csv`, `query`, `ini`,
`envfile`, `pathlib`, `multipart`, `etag` — planned codecs. If each
re-declares its own `Value` union, they will need a shared `Value`
cluster, which is an **architecture decision (open question)**:
exporting one canonical Value from a shared module vs. each codec
keeping its own. Needs the function-values design first (codecs need
function values), then the sharing decision.

## Ruled out — nominal-type unification (html)

`html`'s `provides [...]` wrapper types are the API surface, not
machinery: each wrapper is a distinct named type with no shared body
to parameterize. Unifying them under one `Node<T>` would erase the
nominal API. Ruled out.

## Pipeline

1. function-values design (b00 staged: syntax + expansion + static
   rejection now; invocation/runtime/emit deferred to a real customer)
2. outcomes design → unblocks the Compare collapse (needs bare `-> T` select)
3. Value-cluster sharing architecture decision → unblocks codec dedup
4. validation: G2-ready when built, can be piloted any time
