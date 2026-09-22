# I45 acceptance — release-candidate gates

Closed 2026-09-21. One extended macos-15 verifier job operates
every required service and runs the unnarrowed suite with zero
skips; a recreated tsc workflow typechecks freshly emitted output
from all eight maintained examples and uploads the tree; pins for
every toolchain and test dependency live in committed lockfiles
and target records. No release upload, credentialed signing, or
live provider call was added or performed.

Baseline `ebddcae` (I44) plus the I45 worktree. Apple M4 (Mac16,12)
darwin/arm64, macOS 27.0, Go 1.27.1 (CI uses go.mod's Go 1.25),
Bun 1.4.2 (`744846f84`, archive sha256 `90987a3a…6be1`,
hash-verified), TypeScript 7.0.2 with `@types/bun` 1.4.2 and
`@types/node` 24.13.6, Node v24.21.0, Playwright 1.55.1 with
Chromium 140.0.7339.186, PostgreSQL 17.11 (brew and container
both exercised).

## Design consultations

[i45-jev](../i45-jev/decision.md): three fresh Jev consultations (all
prose rewritten, same facts/options). Unanimous at full confidence
for env_copy (1.0 × 3): the staged stdlib test copies dist trees
under `CAN_FRESH_EMIT_DIR`. Unanimous for upload_emit (0.84, 0.79,
0.77): the emit tree plus tsc log upload as artifacts. Unanimous
in direction for extend_verifier (0.70, 0.82, 0.58) with a round-3
hedge toward separate gates (0.31), investigated: separate
operation stays only where warranted (release-qualify for offline
install/update), so the suite runs in one job with one verdict.
Judgments are advice; the checks below are the proof.

## Implementation

- `.github/workflows/verifier.yml`: one macos-15 job that acquires
  the pinned Bun archive, qualifies it offline, extracts and
  revision-checks the Bun binary (`1.4.2+744846f84`, archive
  sha256 asserted), operates disposable PostgreSQL 17 via brew
  `initdb`/`pg_ctl` on 127.0.0.1:5433, installs the pinned
  browser harness (`npm ci` plus `playwright install chromium`)
  and TypeScript leg, then runs `gofmt`, `go vet ./...`,
  `cataloguegen --check`, `modcheck`, `gramcheck`,
  `go test -count=1 ./...`, and `bun test runtime/test/`,
  uploading all three logs plus the qualification report.
- `.github/workflows/tsc.yml` (recreated): acquires the archive,
  installs `tscheck/`, emits all eight examples fresh via
  `TestStdlibMaintained` with `CAN_FRESH_EMIT_DIR`, runs pinned
  `tsc -p tsconfig.json`, and uploads the emit tree plus log as
  `fresh-emit-tsc-v1`.
- `tscheck/` (recreated): `package.json` pins `typescript`
  7.0.2, `@types/bun` 1.4.2, `@types/node` 24.13.6 with a
  committed lockfile; `tsconfig.json` mirrors the staged-leg
  strictness (strict, bundler resolution, `bun`+`node` types)
  over `.fresh-emit/**/*.ts`; README documents the local run.
- `tests/integration/stdlib_test.go`: `CAN_FRESH_EMIT_DIR` copies
  each project's built `dist/` to `<dir>/<project>/` (~10 lines,
  same staged path as the suite).
- `.gitignore`: covers `tscheck/.fresh-emit/`.
- Pins: Bun archive URL/sha/size/revision in
  `distribution/target.json` (verified by `qualify.py` plus the
  CI revision assertion); TypeScript and type definitions in
  `tscheck/package-lock.json`; Playwright in
  `tests/integration/browser/package-lock.json`; Go in `go.mod`;
  seed tooling imports `bun` only (no npm dependency).

## Verification

Every CI step below was executed locally with identical commands
(the authorized alternative to watching CI); the workflows
themselves parse as valid YAML.

- `qualify.py` over the pinned archive: 14/14 native checks pass.
- Bun extract: archive sha256 and `bun --revision`
  `1.4.2+744846f84` asserted exactly as CI does.
- PostgreSQL: the exact brew `initdb`/`pg_ctl`/`createdb` block
  starts PostgreSQL 17.11 (port 5434 locally — 5433 holds the
local docker occupant; CI uses 5433); all live SQL, transaction,
  and application legs pass against it.
- Browser: `npm ci` plus `playwright install chromium` from a
  clean slate yields Chromium 140.0.7339.186 and the browser
  legs pass.
- `gofmt` clean, `go vet ./...` clean, `cataloguegen --check`,
  `modcheck` (62 sources), `gramcheck`: all pass.
- `go test -count=1 ./...`: all 16 packages pass (integration
  256s) with archive, database, and tsc operated; a verbose
  double run reports zero `--- SKIP` lines, so there are no
  no-run rows to justify.
- `bun test runtime/test/`: 850 pass, 0 fail, 63,285
  expectations (142 files).
- Fresh emit plus `tsc -p tsconfig.json`: exit 0 over 504
  emitted files from all eight projects. Scratch negative: a
  mistyped line appended to one emitted entry fails with
  `TS2322` at exit 1; the scratch tree was removed after.
- Negative gate coverage: broken emit fails tsc (above);
  unowned catalogue operations fail the inclusion inventory
  (I43 scratch); legacy launcher symbols fail the retirement
  gate (I44 scratch); missing manifests fail stdlib discovery
  (I43 scratch). `tests/conformance/native.test.ts` runs
  inside the `qualify.py` step (14/14 native checks).

## Limitations

- CI ran by local simulation, not by watching GitHub-hosted
  runners; first push will confirm runner-side behavior.
- The sim PostgreSQL used port 5434 for a local occupant; the
  workflow uses 5433 with the identical command block.
- `release-qualify.yml` is unchanged and still performs no
  signing, notarization, or upload.
