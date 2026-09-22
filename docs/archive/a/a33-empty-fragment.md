# a33 — Empty fragment (zero-case pin)

Status: shipped. One function: `html__fragment__empty()` seals
`""` as `Html__Safe`. No errors, one decision-table row.

## Rule

Empty content is admissible as `Html__Safe`. It serializes to
nothing, carries no authority, and composes neutrally wherever
fragments join. The join owns separator and duplicate semantics
when it lands; this pin constrains it to treat emptiness as
absence, matching the attribute-side rule from a30.

## Why now

The outside review deferred this as consumerless, which was
correct at the time. It ships now as a contract pin, not a
feature: later builders (elements, join) inherit the zero case
instead of assuming it, and the assumption is on record before
anyone depends on it. No new machinery, no new errors, no
judgment call — nullary shape and literal sealing both have
shipped precedent.

## Proof costs

- 1 new row, 181/181 green with the pre-existing 180.
- `errors.json` regenerated; `go test -count=1 ./...`,
  modcheck, and gramcheck all green fresh.
