# a34 — Boolean names top-up (readonly, required, checked)

Status: shipped. The gate admits four names; the spelling
reconstruction is a live N-1 comparison chain; the composer is
unchanged and name-agnostic. No new functions, types, or
errors.

## Names and their honest contracts

- `readonly`: prevents editing in text-like controls.
  Meaningless elsewhere; browsers ignore it there — the safe
  direction.
- `required`: marks a control mandatory for constraint
  validation and blocks submission while empty. Real
  behavior, disclosed not blessed; its interaction with
  `disabled` (skipped validation) stays with the consuming
  form contract per a30.
- `checked`: initial checked state for checkbox and radio.
  Pure initial state, the quietest of the three.
- Still rejected, pinned as near-misses: `autofocus` (moves
  focus — behavior needing its own slice), `selected` and
  `multiple` (option/select contexts), `open` (details and
  dialog contexts).

The constructor guarantees approved serialization only —
unchanged from a30. None of these names is advertised as
inert or as appropriate on any element.

## Why now

The a30 review said not to add names merely to appear
broader, and no element consumer exists yet. This ships on
explicit program direction overriding that timing advice,
with the override stated here rather than buried: the
mechanism (N-1 chain) was verified, the justifications are
per-name, and waiting would not have improved either. The
caller rule itself stands for future additions.

## Proof costs

- The N-1 chain predicted in a30 is now live code: three
  comparisons plus a final literal, every false side taken
  by the remaining names, no fallback arm. The fifth name
  extends the chain and its rows the same way.
- 9 new decision-table rows (4 gate accepts with 4 new
  near-miss rejections replacing the 3 promoted names, 3
  spelling round-trips, 2 composer end-to-end pins),
  190/190 green with the pre-existing 181.
- Independent oracle: the generated `html.ts` runs under
  node over all four names present and absent plus five
  rejections — 17/17 green. Scratch probe at
  `/tmp/bool-topup-probe.mjs`, not committed.
- `errors.json` regenerated; `go test -count=1 ./...`,
  modcheck, and gramcheck all green fresh.
