# Current immutable native map catalogue

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

# map — insertion-ordered maps with generic keys and values

- `map.can` — `mod map`: `std__map__get`, `std__map__insert`,
  `std__map__replace`, `std__map__remove`, and `std__map__entries`
  over `Map<K,V>` (one `Seq<Map__Pair<K,V>>` entries spine).
  A map is its ordered entries; replace keeps position, remove
  closes ranks, insert never overwrites. Errors are payloadless:
  error payloads cannot be generic.
- `map.ts` + `errors.json` — committed golden TS prod emit
  (tests stripped). Regenerate: `go run ./compiler --out std/map
  std/map/map.can`; verify: `go test ./...`.

Rules: `/REQUIREMENTS.md`. Program: `docs/archive/ASTRA_STDLIB.md` §1.8.
