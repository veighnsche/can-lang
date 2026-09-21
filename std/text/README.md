# Current native text and Unicode catalogue

The maintained project is [current/src/main.can](current/src/main.can). Its 31
mandatory assertions cover every I24 operation, bound string methods, spreads,
method chains, deterministic fixtures and the declared text errors. Run the
staged `canlc assert std/text/current` or `canlc run std/text/current` command.

String indexing and slicing remain UTF-16 code-unit operations. Named `scalars`,
`from_scalars`, `graphemes` and `normalize_nfc` validate Unicode scalar input and
use the pinned native runtime's code-point, segmentation and normalization APIs.
Literal replacement preserves dollar sequences; split preserves empty pieces.
The implementation is `runtime/text.ts`, with bounded fromCodePoint chunks.

Adjacent old Can/generated files are historical inputs for I43/I44 retirement,
not a current library or fallback. No custom Unicode, sorting or string kernel
is selected by the current CLI.

## Historical text implementation

# text — explicit text construction and scalar access

- `text.can` — `mod text`: `std__str__concat`, `is_empty`,
  `is_whitespace` (int-denoted scalars: literals are raw, so
  tab/LF/CR never appear in a table), `length_scalars`,
  `scalar_at`, `slice_scalars`, `find_from`/`find`, `contains`,
  `starts_with`, `ends_with`, `replace_all_from`/`replace_all`
  (non-overlapping, left to right), `trim_left`/`trim_right`/
  `trim_ascii` (space, tab, LF, CR), and `upper/lower_ascii`
  (`_from` workers over alphabet literals via same-file `find`).
  Concat preserves every scalar; safety lives in context
  encoders, never here. `find` locates the empty pattern at 0;
  `replace` rejects it. `join`/`join_from` walk `Seq<str>`
  positionally (first-element test is `position == 0`, never
  `acc == ""`). `split`/`split_from` return `Split__Result`,
  retain empty fields, go leftmost on overlaps, and mint
  `text.empty_separator`. Graphemes/casefold/normalize wait on
  pinned data, base64 on Bytes via
  `std__base64__encode`/`std__base64__decode`
  (`utf8` via `std__utf8__encode`/`std__utf8__decode`,
  `hex` via `std__hex__encode`/`std__hex__decode`), and the
  URL-safe unpadded `std__base64url__encode`/
  `std__base64url__decode` (pure-`.can` translation, padding
  strip, and strict re-pad over the base64 kernels, emitting
  `encoding.invalid_base64url`).
- `text.ts` + `errors.json` — committed golden TS prod emit
  (tests stripped). Regenerate: `go run ./compiler --out
  std/text std/text/text.can`; verify: `go test ./...`.

Rules: `/REQUIREMENTS.md`. Program: `docs/a16-text.md`,
surface: `docs/a20-text-operators.md` (`#`, `s[i]`, `s[a:b]`).
