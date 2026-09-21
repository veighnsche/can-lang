# I34 acceptance — Packaged HTMX and assets

Closed 2026-09-21. The distribution embeds exactly the reviewed upstream
HTMX 4.0.0 bytes, project assets are build inputs with content-addressed
immutable URLs, `asset::url` resolves at compile time, and the server
serves only compiler-known paths with fixed security headers. No
compile/run-time CDN, no authored client script, no client Can runtime.

Baseline `7207dce` (I33) plus the I34 worktree; pinned target
`bun-1.4.2-darwin-arm64-v1` (Bun 1.4.2 revision `744846f84`),
TypeScript 7.0.2, Playwright 1.55.1 with Chromium 140.0.7339.186
(build v1193), macOS arm64.

## Design consultations

- `../i34-jev/decision.md`: three fresh Jev rounds. Unanimous: `.json`
  assets must parse at build time; names resolve in the caller's own
  manifest only, shared across dependencies through callee functions
  returning `html::url`. Binary signatures 2-1 for header-validated
  structure (dimensions, counts, versions, offsets; no checksums or
  decoding) over magic bytes alone. The weak 3/3 single-`missing`
  advice was overridden on the merits: sibling `invalid_url` reasons
  are per-cause specific and the fixes differ, so unknown names report
  `missing` and names owned by another project report `unowned`.

## What changed

Compiler: new `project/assets.go` (snapshot, confinement, closed
extension/MIME inventory, per-format signature validation, digests,
immutable `/__can/project/<digest>/<name>` URLs and
`assets/<digest>/<name>` artifacts) wired through `project/graph.go`;
new `check/assets.go` (static-literal enforcement, caller-manifest-only
resolution) with `Asset` context in `check/completions.go`,
`Program.Assets` in `check/program.go`, and `/__can` route reservation
in `check/http.go`; new `ir/assets.go`; new `emit/assets.go`
(manifest-owned bundle, collision guard, lowering to private
`declareAsset`/`rejectAsset`) with `$canAssetTable`/`$canAssets` wiring
in `emit/program.go`. One fix during verification: asset steps skipped
target resolution but still emitted `$canCallableInstance()`, which
strict `tsc` rejects; `emit/regions.go` now treats asset steps like
native/array steps with an `undefined` instance, covered by a
regression assertion.

Runtime: new `platform/assets.ts` (exact-table `Bun.file` serving,
per-request SHA-256 plus SHA-384 integrity for HTMX, fixed
MIME/nosniff/immutable-cache/ETag/304, 404/405, exported `browserPolicy`
CSP). `server.ts` applies the CSP to application, asset and error
responses; `http.ts` forbids application `content-security-policy` and
`hx-*` response headers; `html.ts` `runtimeHead` pins
`mode="same-origin"` with the compiler-owned `noSwap` list (204, 304,
400--599 except 422) and adds private `declareAsset`/`rejectAsset`;
`router.ts` reserves `/__can/`. Registered in `modules.json`.

Distribution: `assets/htmx-4.0.0.min.js` with `htmx.lock.json` (three
byte-identical cross-checked origins, size 36716, SHA-256
`e484d917…593f`, integrity
`sha384-BvJpBiO8Kh31EqtJe5DRIeWrHWnCGkwytKs9NKFi86Hhw96dEqdEMzZDeK9iEGTc`,
0BSD provenance) and `notices/htmx-LICENSE.txt`. `manifest.go`
`VerifyHTMX`/`HTMXAsset` enforce the P11 digests; `build.go` verifies
before publication. Closed inventory holds exactly the script and lock.

Tests: `compiler/testdata/current/assets/page.can` (document, 422/200
form, 204/500/polling endpoints, hostile text); Go tests beside each
pass; `runtime/test/assets.test.ts`; `tests/integration/assets_test.go`
(staged loopback plus negatives and determinism);
`tests/integration/browser/` (pinned Playwright harness, lockfile,
`assets.mjs`) with `TestCurrentBrowserAssets`.

## Verification

- `go test -count=1 ./distribution ./compiler ./compiler/internal/...`
  with `CAN_BUN_ARCHIVE`: all packages pass, including the HTMX vendor
  pin and the tampered-script build refusal (no staged residue).
- `bun test runtime/`: 277 pass, 0 fail, 21,987 expectations, 52 files.
- Strict `tsc` 7.0.2: asset/server/html runtime modules and tests
  clean; staged `entry.ts` clean.
- `catalogue-check`, `modcheck`, `gramcheck`: clean.
- `TestCurrentBundledAssets`: 12 staged Can rows (`real-can` plus
  `supplied-completion`), run, build; rebuild and relocated rebuild
  keep buildID `4a4d6c8…` with byte-identical asset trees; loopback
  through the real server proves page wiring, pinned-byte serving,
  conditional/HEAD/405 handling, traversal/unknown/map 404s, 422 named
  and blank forms, hostile escaping without a script element, 204/500,
  404/405+Allow, tamper-to-404 with restore recovery, and clean
  stop/wait. Staged negatives refuse a `.js` asset, a `../` escape,
  and a `/__can/evil` route with specific diagnostics.
- `TestCurrentBrowserAssets`: Chromium 140.0.7339.186, 12/12 harness
  checks, 11 loopback requests, 0 aborted externals. `report.json` and
  `screenshot.png` in this directory record the run: HTMX loaded from
  the pinned route with integrity, single script element, same-origin
  plus noSwap config, local stylesheet 200, CSP without eval, 422 and
  200 swaps, hostile text inert (`window.__pwned`/`__evil` undefined),
  204 and 500 no-swap, repeated dashboard polls rendering millisecond
  digits, and no request leaving loopback.
- Full `tests/integration/` suite with `CAN_BUN`, `CAN_BUN_ARCHIVE`,
  `CAN_TSC`: `ok ... 133.282s`, every staged test green including both
  I34 tests.

## Limits and follow-ups

- `tsc -p tscheck/tsconfig.json` over legacy `std/`/`sketches/` TS
  fails on files untouched by I34 (missing `./scalars`, `./quota`,
  `./host` modules, `Json__Frame` mismatches). Pre-existing; owned by
  I43/I44/I45, not this task.
- The Playwright browser download is local test infrastructure pinned
  by `tests/integration/browser/package-lock.json`. The Go test skips
  cleanly when node, the installed harness, or the chromium executable
  is absent.
