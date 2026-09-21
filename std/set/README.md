# Current immutable native set catalogue

The maintained `collections` catalogue uses private native Map/Set storage with
immutable opaque values, scalar keys (`int`, `bool`, `str`) and insertion order.
The shared current source project for both maps and sets is
[`std/map/current`](../map/current). Run it with the staged release's
`canlc assert std/map/current` and `canlc run std/map/current`.

All twelve operations are compiler-checked native intrinsics. Map insertion
rejects duplicate keys; get, replace and remove reject absent keys. Replacement
keeps position and copies preserve aliases. Set union appends unseen right keys;
intersection and difference preserve left order. Intersection uses native left
array filtering and right membership to repair the native smaller-set ordering.

The adjacent legacy source and generated files below are historical migration
inputs scheduled for retirement by I43/I44, not the current implementation.

## Historical implementation

# set — sets with explicit ordering

- `set.can` — `mod set`: `std__set__contains`,
  `std__set__union`, `std__set__intersection`, and
  `std__set__difference` over `Set__Members<T>`. Members keep
  first-occurrence order; union appends only absent members;
  intersection and difference preserve the left order. Equality
  is per-instance `==`, shared with maps.
- `set.ts` + `errors.json` — committed golden TS prod emit
  (tests stripped). Regenerate: `go run ./compiler --out std/set
  std/set/set.can`; verify: `go test ./...`.

Rules: `/REQUIREMENTS.md`. Program: `docs/ASTRA_STDLIB.md` §1.8.
