# a92: S2b cost comparison and migration call

Status: decided 2026-09-18. Consumer 3
(`sketches/notify/notify.can`) is green on the S2a check
ops; the comparison below is measured, not estimated.
The call migrates consumers 1–2; execution is the next
step, under the rule in §4.
Parent: `docs/a/a91-schema-s2-design.md` (S2b gate);
consumers 2–3: `sketches/reserve/`, `sketches/notify/`.

## Method

Both consumers carry 23 decision-table rows over the
same obligation shape (every constraint, every
malformed-policy outcome, boundaries, priority rows,
preservation, exact reporting, admit open/reject), so
row count is controlled and the deltas below are the
binding/interpretation cost with and without the check
layer. One-time check-layer cost (S2a, in `quota.can`):
~107 lines — 3 records, 3 ops, 21 rows, converter
scripts — paid once.

## Measured comparison

| Cost | Consumer 2 (producers) | Consumer 3 (check ops) |
| ---- | ---------------------- | ---------------------- |
| File lines | 171 | 138 (−19%) |
| Arms + rows + scripts (`=>`) | 99 | 81 (−18%) |
| Cross-file call sites | 9 (3 producers + 6 converter) | 3 (checks only) |
| `given` exchanges | 58 | 50 (−14%) |
| Error kinds declared | 5 (1 raised + 4 matched) | 1 (raised only) |
| Body raise sites | 6 reconstruction arms | 0 (forward only) |
| Converter calls in consumer | 6 | 0 |
| TS imports | `./quota` + `./scalars` | `./quota` alone |

Line breakeven on the 107-line layer lands at the
fourth consumer — but lines were never the gate's
point. The structural wins are already banked: token
formats, producer-outcome matching, and converter
scripting are each owned once, so a format change, a
new producer arm, or a converter evolution touches one
site instead of N, and no consumer can mis-reconstruct
a triple (consumer 3's body cannot build one at all).
Exchanges shrink least (−14%): every reached site
still scripts every reaching row. That residual is
inherent to scripted cross-file calls, not to the
check design, and linked vectors keep executing the
real bodies underneath.

## The call: MIGRATE both

- **Consumer 1 (quota pilot): migrate.** Its checks
  are same-file, so migration deletes its converter
  givens entirely — the largest single saving in the
  program. It is also the reference implementation:
  leaving it on the old idiom while blessing the
  checks would split the codebase into two dialects
  permanently.
- **Consumer 2 (reserve sketch): migrate.** Same
  frozen expectations; −33 lines, −8 exchanges, −4
  kind declarations, −1 TS import. Sketches demo the
  blessed idiom; a sketch demonstrating the superseded
  one is a trap for the next reader.

## Execution rule (next step)

Bodies and `given` tables only. Every test expectation
in both consumers stays byte-identical — migration is
behavior-preserving by construction, re-verified by
goldens, linked vectors, and parity. If any expectation
must change to migrate, that is a behavior change, not
a migration: stop and explain instead of editing the
row.

## S2 close-out

After migration executes green, S2 is done: the closed
vocabulary (3 constraints × `str`/`int`), the three
check ops, and three consumers on one idiom. S3
(generic `Schema<T>`) still needs order-8 generics plus
the checked field-binding mechanism; nothing here
advances those except the exhibits they will consume.

## Execution record (green, same day)

Both migrations held the rule: bodies and `given`
tables only, zero test-row lines changed (verified by
diff — the only non-script `=>` changes are the
removed boolean-dispatch match lines).

- Consumer 1: validate body 35 lines → 10, its own
  givens → zero. Scripts conserved, not deleted: the
  12 deleted exchanges reappear keyed by validate and
  envelope rows in the check ops' converter givens
  (same-file transitivity), leaving `quota.can` at 28.
  The win is reconstruction arms, not scripts.
- Consumer 2: 171 → 138 lines, 58 → 50 exchanges, 5 →
  1 kind declarations, TS imports 2 → 1 — converging
  exactly onto consumer 3's shape (50 exchanges),
  which is the migration's external validity check:
  the migrated consumer is indistinguishable in cost
  structure from the natively-written one.
- Full gates green (`go test -count=1 ./...`,
  `modcheck`, `gramcheck`); goldens, linked vectors,
  and parity re-verify every frozen expectation.
