# Current array catalogue

Ordered sequences are the native array catalogue: literals,
copy-update, field/index/slice/length, `.map`, `.filter`,
`.for_each`, `.fold`, `.find`, `.some`, `.every`, `.sort_by`,
`.slice`, `.concat`, `.to_reversed`, and prelude `append`, all with
native copies and sequential failure-stop traversal. `Seq__Values`
wrappers, `Seq__Item` elements, comparator callbacks beyond
`sort_by`, and group/zip/dedup helpers are excluded. Current
demonstrations live in the `arrays` fixtures and throughout the
admitted applications.

The adjacent legacy source and generated files below are historical
migration inputs scheduled for retirement by I43/I44, not the current
implementation.

## Historical implementation

# seq — immutable ordered sequences

- `seq.can` — `mod seq`: `std__seq__empty`, `std__seq__singleton`,
  `std__seq__length`, `std__seq__get`, `std__seq__append`,
  `std__seq__concat`, and `std__seq__slice`, each generic over the
  element type with int + str decision tables. Bare-Seq returns are
  rejected by the language, so sequences travel in `Seq__Values<T>`
  and elements in `Seq__Item<T>`. Slice is half-open, same rule as
  text slicing; invalid bounds never clamp. Higher-order traversal
  (`std__seq__map`, `std__seq__filter`, `std__seq__fold`,
  `std__seq__all`, `std__seq__any`, `std__seq__find`) takes total
  callbacks over data-only heads; workers visit left to right, and
  `all`/`any`/`find` stop at the first decisive element. `find`
  reports absence as `sequence.not_found()` (no `Option<T>`:
  generic variants do not exist, so the text.find error shape is
  used instead). `std__seq__sort` takes an explicit `Seq__Order`
  value (`Asc`/`Desc`; lexicographic deferred) over the
  per-instance built-in order — insertion sort, stable by
  construction since equals never reorder. `std__seq__unique`
  keeps first occurrences via per-instance `==`, the same
  contract maps and sets share.
- `seq.ts` + `errors.json` — committed golden TS prod emit
  (tests stripped). Regenerate: `go run ./compiler --out std/seq
  std/seq/seq.can`; verify: `go test ./...`.

Rules: `/REQUIREMENTS.md`. Program: `docs/ASTRA_STDLIB.md` §1.8.
