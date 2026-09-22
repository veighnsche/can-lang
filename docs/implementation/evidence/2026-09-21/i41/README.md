# I41 acceptance — current diagnostics, LSP and editor grammar

Closed 2026-09-21. The editor path now diagnoses through the current
pipeline only: `canlc lsp` keeps its stdio JSON-RPC transport and
replaces every semantic hook with an inert bridge over in-memory
overlay snapshots — parse, resolve, and check on every keystroke, with
definition jumps through resolved declaration identities. The
TextMate grammar highlights current declaration/section syntax, and
`gramcheck`/`modcheck` gate on the maintained fixtures instead of
predecessor goldens.

Baseline `d0ae72d` (I39) plus the I41 worktree. Apple M4 (Mac16,12)
darwin/arm64, Go 1.27.1, Bun 1.4.2 (pinned archive sha256
`90987a3a…6be1`, hash-verified), TypeScript 7.0.2 with `@types/bun`
1.4.2 and `@types/node` 24.13.6 for the integration strict-TS leg.

## Design consultations

[i41-jev](../i41-jev/decision.md): three fresh Jev consultations (all
prose rewritten, same facts/options). Unanimous for file_line_anchor
(1.0, 1.0, 1.0): spanless resolve/check failures report the
CLI-identical message on line 1 of the attributed file. Near-unanimous
for file_scope_symbols (0.81, 0.97, 0.99): definition resolves through
file, package, prelude, and import scopes to nominal, function,
constructor, and generated declarations, declining body-local names.
Unanimous for ignore_until_saved (1.0, 0.96, 0.99): buffers absent
from the loaded project publish empty diagnostics until saved. All
three followed. Judgments are advice; the checks below are the proof.

## Implementation

- `compiler/internal/driver/diagnostics.go` (+ test): the inert
  bridge. `CheckSnapshot` loads disk plus overlays and runs resolve
  and check without requiring an entry point; load failures surface
  the first spanned lexer/parser diagnostic with its code through
  `internal/source` UTF-16 positions, while resolve/check failures
  anchor CLI-identically to the attributed file's first line.
  `Definition` resolves the identifier at an editor offset through
  file/package/prelude/import scopes, including record and
  generated-record fields behind annotation-known receivers.
- `compiler/internal/project/overlay.go` (+ test), `graph.go`:
  manifest-backed unsaved-source snapshots keyed by canonical path
  plus document version, substituted before parse; entries for paths
  the walk never visits are ignored until saved.
- `compiler/lsp.go`: the server region is new transport over the
  bridge — didOpen/didChange diagnose the overlay snapshot and fan
  out per-URI publishes with version echoes, didClose clears,
  definition answers from resolved identities, and `lsp --baseline`
  is refused with exit 2 (the baseline-veto handshake is retired).
  The legacy `diagnose`/`checkSem`/`proofDiag` region stays because
  package-wide predecessor tests and production (`expand.go`,
  `main.go`) still share it; its removal belongs to I44, and no
  editor request routes through it.
- `compiler/lsp_server_test.go`: ten framed protocol exchanges —
  clean publish, unsaved sibling coherence, duplicate basenames,
  rapid edits, close clears, malformed-input survival, UTF-16 wire
  fidelity against the bridge, same-file definition, generated-field
  definition, and a divide-by-zero poison fixture proving the server
  parses without executing.
- `editors/vscode/syntaxes/can.tmGrammar.json`: current keywords
  (hard lexer words minus header/literal/type words), primitive
  types, unified numbers (hex/bin/oct/floats), `::` scoped names,
  raw/triple strings with pinned rule order, given/asserts blocks
  scoping case keys. Retired: `extern`/`rev`/`mod`/`dec` keywords,
  `@N` pins, `d"`/`e"` strings, `__` identities, `E`-codes, the `-`
  row marker, and capitalized type names.
- `tools/gramcheck/`: every sample must match its scope and occur
  verbatim in `compiler/testdata/current`; required/retired keyword
  sets, retired-scope absence, string-rule order, and the
  given/asserts block rule are pinned structurally.
- `tools/modcheck/`: rewritten for `package` headers over the 42
  maintained fixtures — header presence, duplicate provides/uses
  entries, alias collisions, uses resolution against catalogue
  packages plus tree packages (mirroring `resolve.Build`), and bans
  on predecessor headers, `extern`, `tests` sections, `d"`/`e"`
  strings, `@N` pins, and braces outside strings.
- `editors/vscode/{client/extension.js,language-configuration.json,
  README.md}`, `package.json` (unchanged, verified): the client
  falls back to `canlc` on PATH when no bundle exists, braces left
  the bracket pairs, and the README documents the bridge, the
  retired `--baseline` flag, and the lint limitation below.

## Verification

- `go test -count=1 ./...`: all 16 packages pass, including
  `tests/integration` (184s) with `CAN_BUN` 1.4.2,
  `CAN_BUN_ARCHIVE` (hash-verified), and `CAN_TSC` (TS 7.0.2;
  must point at `typescript/bin/tsc`, since the tests derive
  typeRoots from its path).
- `bun test runtime/test/`: 850 pass, 0 fail, 63,285 expectations
  (142 files) — unchanged, proving no runtime regression.
- `cataloguegen --check`, `modcheck` (42 fixtures), `gramcheck`:
  all pass.
- `gofmt`/`go vet` clean on every touched package (4 remaining
  `gofmt` findings are pre-existing in untouched files).
- Negative cases: `--baseline` exits 2 with a retired-flag error;
  unknown flags/positionals are usage errors; unknown packages,
  duplicate entries, alias collisions, pins, and every retired
  shape fail `modcheck` unit checks; retired keywords/scopes fail
  `gramcheck` unit checks; garbage bytes and corrupt frames never
  wedge the server; unresolvable siblings and poison asserts
  diagnose without execution or disk writes.
- CLI/LSP parity: the Unicode exchange asserts the published
  range/code/message equal the bridge diagnostic exactly, and the
  driver tests assert overlay diagnosis equals saved-file diagnosis
  and the editor message equals the CLI check error.

## Limitations

- Strictness (lint) squiggles are not published; the editor shows
  parse, resolve, and check diagnostics only.
- Contextual (HTTP/AI section) words are unscored: highlighting
  them everywhere would mis-scope plain identifiers.
- The `tscheck` project gate over `std/`/`sketches/` goldens fails
  on pre-existing predecessor rot (missing modules, tag-type
  drift); those trees are untouched here and belong to I43/I44.
