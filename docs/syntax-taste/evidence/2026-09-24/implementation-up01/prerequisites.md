# UP01 prerequisites record

Integration baseline `13b6cdb` on branch `integration/up01-inputs-boundaries`.
Observed 24 September 2026 on the coordinator host; nothing below was operated
as a qualification leg by UP01.

## Pinned toolchain identities (must be operated at UP24/UP25)

| Item | Pinned identity | Source |
| --- | --- | --- |
| Bun archive | `bun-darwin-aarch64.zip`, sha256 `90987a3a…be1d12f`, 25377591 bytes | `distribution/target.json`, `.github/workflows/verifier.yml` |
| Bun binary | `1.4.2+744846f84`, runtime sha256 `35d20dd0…08a7a5b5`, target `bun-1.4.2-darwin-arm64-v1` | `distribution/target.json` |
| Go directive | `go 1.25.0` (`go.mod`); UP01 host ran `go1.27.1 darwin/arm64` | `go.mod` |
| TypeScript | `7.0.2` via `CAN_TSC` (`tscheck/node_modules/typescript/bin/tsc`) | `tscheck/package.json`, `.github/workflows/tsc.yml` |
| Playwright harness | `1.55.1`, `playwright install chromium` today | `tests/integration/browser/package.json`, `.github/workflows/verifier.yml` |
| PostgreSQL service | `postgres://can@127.0.0.1:5433/can_test`, PostgreSQL 17 | `.github/workflows/verifier.yml` |
| SQLite | example/acceptance database engine (invoice fixture) | invoice acceptance gate |

## Local availability observed at UP01 (not qualification)

- Bun `1.4.2` installed; Go `go1.27.1 darwin/arm64`; `sqlite3 3.54.0`;
  `psql (PostgreSQL) 17.11 (Homebrew)`.
- Google Chrome `153.0.8010.53` installed; Playwright cache holds only
  `chromium_headless_shell-1243` and `ffmpeg-1011` — **no WebKit engine
  installed locally**.
- No `CAN_BUN_ARCHIVE`, `CAN_TEST_POSTGRES_URL`, or `CAN_TSC` environment
  operated in this shell; no `/tmp/bun.zip` present.
- Host is macOS arm64; **no selected Linux target and no installed
  distribution root were operated here**.

## Gates that must be operated before completion credit

1. UP24: both required browsers (**Chromium and WebKit** — CI currently
   installs Chromium only and must be extended per the task contract),
   pinned Bun archive, PostgreSQL service, pinned `tsc`, full
   `go test ./...`, `bun test runtime/test/`, fresh-emit `tsc` workflow,
   and `git diff --check`, with zero required-leg skips.
2. UP25: installed distribution in a disposable root on the selected Linux
   target (record archive/image hashes, Bun revision, OS/arch, installed
   paths, actual vs emulated host mode), full invoice gate in named
   Chromium and WebKit against the application origin, same-input rebuild
   identity, and asset retention/restart legs.
3. UP23: named Chromium **and** WebKit legs for the browser/runtime and
   invoice matrix; an unavailable required engine is incomplete, not a pass.

No production claim in this record relies on a missing fixture,
placeholder helper, or skipped test.
