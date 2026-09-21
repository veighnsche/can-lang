# I11 — current CLI through the bundled runtime

Implemented manifest loading → eligible names → sealed concrete types → checked
initializers/completion regions → private ESM → verified generation publication →
absolute pinned sidecar execution. No legacy checker, evaluator or emitter is
called by `build`/`run`. Shared emitted artifacts now live in IR, avoiding an
emitter/driver import cycle.

- **P+:** the staged `echo.can` fixture receives application-only argv and writes
  exact Unicode, spaces, empty text and a leading BOM through awaited native I/O.
  Flag-shaped arguments include `--inspect`, `--preload=/missing.ts`, `-e`, and
  literal `--`. The final regression found that Bun consumes its first `--`
  after the script; the launcher now supplies that separator itself. The generated
  entry awaits `runEntry`, and unit tests hold main and diagnostic completion open
  to verify that root status cannot settle early. A second source module and
  forward-reference initializers execute through the same native program.
- **N−:** wrong entry signatures, mismatched static arguments, missing calls,
  invalid unused concrete bodies and initialization cycles fail. A static failure
  leaves the previous current generation selected. Missing argv produces a bounds
  standard failure. Startup division by zero prevents main/output. An authored
  domain failure reports its stable ID without its private payload. A real stdout
  pipe whose reader is closed before launch maps to `io::write_failed` (1211).
  Usage errors return 2, checked/runtime failures return 1, and successful void
  completion returns 0. Forged carriers/byte tokens reject without executing traps.
- **INT:** `TestCurrentBundledCLI` assembles a distribution from the pinned local
  archive and invokes its launcher through a symlink, outside the project cwd,
  with networking denied and `PATH=/nonexistent`. A hostile `BUN_OPTIONS` value
  cannot preload code. The existing offline distribution/output/region suites
  also pass after the argv separator correction.

Validation:

1. `CAN_BUN=/Users/vince/.bun/bin/bun GOCACHE=/tmp/can-i02-go-cache go test ./...`
   — [full Go output](i11-go-tests.txt).
2. `/Users/vince/.bun/bin/bun test runtime` — 30 tests, 1,154 expectations;
   [runtime output](i11-runtime-tests.txt).
3. TypeScript 7.0.2 strict checking with Bun 1.4.2 types over a freshly emitted
   multi-file CLI generation and the new maintained runtime/test modules —
   [strict output](i11-typescript-tests.txt). Options: `--noEmit --strict
   --skipLibCheck --target esnext --module esnext --moduleResolution bundler
   --allowImportingTsExtensions --types bun,node`.
4. `CAN_BUN_ARCHIVE=/private/tmp/can-i01-bun.zip
   CAN_BUN=/Users/vince/.bun/bin/bun GOCACHE=/tmp/can-i02-go-cache go test
   ./tests/integration ./compiler/internal/driver ./compiler/internal/emit
   -count=1` — [offline results](i11-offline-tests.txt). The outer Go command ran
   outside the agent sandbox so macOS could apply the tests' network-denied child
   sandboxes.
5. `TestProgramChecksMethodsAndNestedRegions` additionally checks a concrete
   receiver method and nested typed success/standard arms; all program tests pass.

[Three Jev consultations](i11-jev/README.md) informed the initialization boundary;
model agreement was not used as implementation proof. [CLI instructions](../../cli.md)
document status, output and the current scope. Only UTF-8 encoding and awaited
stdout/stderr adapters were brought forward for this first slice. Full byte
inventory, assertions, generic specialization, bounded I/O and resource draining
remain the separate unchecked I13/I12/I46/I29/I20 tasks.
