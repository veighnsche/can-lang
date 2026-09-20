# I03 validation

The current C2 lexical boundary is implemented in compiler/internal/source and
compiler/internal/syntax. Original UTF-8 byte spans are retained; logical LF and
CRLF handling, literal decoding, diagnostics, LSP positions and source-map
coordinates share one source object. Current fixtures live under
compiler/testdata/current/lexer.

Passed checks:

- `go test -count=1 ./...` (GOCACHE=/tmp/can-i02-go-cache): all packages passed.
- Explicit source/lexer tests cover the full hard/contextual keyword inventories,
  four-space layout, blank/comment-only lines, nested/doc comments, raw and
  triple strings, exact escape handling, native binary64 overflow rejection,
  finite underflow, bases/ranges/signs, forbidden punctuation and precise
  negative spans. [Saved output](i03-lexer-tests.txt).
- Unicode tests cover astral and combining scalars, UTF-8 byte boundaries,
  UTF-16 surrogate boundaries, BOM and CRLF. Every fixture token endpoint
  round-trips through the LSP conversion and agrees with source-map coordinates.
  A diagnostic after an astral character is checked independently in all three
  coordinate conventions.
- A 10-second initial lexer fuzz run passed 242,062 executions. After the nested
  generic-closer correction, a final 5-second run replayed 117 cached seeds and
  passed 20,490 executions. [Final fuzz output](i03-lexer-fuzz.txt).
- A 5-second source-coordinate fuzz run passed 44,730 executions and its 5 seed
  cases, checking inverse UTF-16 conversion for every accepted byte boundary.
  The fixed seed cases also run in the ordinary Go suite.

Grammar-owned placement checks remain I04 work: mandatory nonempty declaration
blocks, trailing commas, one-line assertion placement and full expressions.
The lexer supplies NEWLINE boundaries and multiline-literal spans instead of
heuristically parsing those constructs. Nested generic closing angles remain
lexable; there is no unsigned-right-shift token.

Old line/brace helpers were renamed LegacyBraceOutsideString and
LegacyHasBraceOutsideString, with their remaining old-parser/tool callers
explicitly updated and no compatibility aliases. These helpers are not used by
the current lexer. Replacing their remaining consumers belongs to I04/I41/I44.
Actual source-map encoding and LSP protocol integration remain I40/I41 gates;
these tests establish their shared coordinate foundation, not those later gates.

During this cycle a reviewed I02 staging collision was also fixed and committed
separately as a35ce37. Its archive-backed rejection and offline baseline evidence
is [i02-collision-regression.txt](i02-collision-regression.txt).
