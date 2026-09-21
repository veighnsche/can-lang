# I24 validation — native text and Unicode

Implemented all 14 text catalogue operations through native JavaScript/Bun APIs.
The compiler admits the closed string method and text function inventory, including
bound references, callback use, chains, literal spreads and fixtures. Runtime
adapters preserve completion/domain contracts and immutable array results.

Code-unit search and slicing deliberately retain native UTF-16 semantics. Named
Unicode operations reject unpaired surrogates; scalar construction rejects values
outside 0..0x10ffff and surrogate code points before conversion. Construction uses
4096-value native fromCodePoint chunks. Graphemes use Intl.Segmenter("und") and NFC
uses native normalize. The frozen target already requires these APIs, including
isWellFormed and replaceAll; no fallback or target relaxation was introduced.
Replacement uses a function so dollar sequences remain literal. Split preserves
empty pieces and rejects empty separators; replacement rejects empty patterns.

Validation completed:

- Native text tests: 5 tests, 158 expectations, including malformed Unicode,
  boundary scalar values, native casing/segmentation, literal dollar replacement,
  immutable outputs, UTF-16 versus scalar/grapheme behavior and 150,000 scalars.
- Strict TypeScript checking of the runtime adapter and tests passed.
- Compiler positive and negative tests passed: wrong argument types, unsupported
  methods and missing domain bounds reject.
- Staged release integration passed offline using its absolute private runtime:
  31 Can assertion roots, main execution, runtime tests and strict checking of
  generated TypeScript. This covers bound callbacks, bound slice, fixtures,
  literal spread replacement and chained string methods.
- Full compiler and integration gate passed with the pinned Bun archive and
  CAN_TSC configured: `go test ./compiler/... ./tests/integration -count=1`.
- Full runtime suite passed: 167 tests, 19,193 expectations, zero failures.

The executable current source project is `std/text/current`. Adjacent legacy
source/generated files are historical migration inputs, labelled in the README;
retirement remains assigned to I43/I44 and is not an active fallback.
