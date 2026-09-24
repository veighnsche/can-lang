# Post-upgrade implementation status (coordinator I)

Living record. Integration branch: `integration/up01-inputs-boundaries`.
Source baseline `13b6cdb`; UP01 `163fd8a`; current baseline **`b1f74c7`**
(UP01 + coordinator digest fix). Workers base on the current baseline.

## Task status

| ID | State | Branch / evidence |
| --- | --- | --- |
| UP01 | done | `163fd8a` + `b1f74c7`; `implementation-up01/` (7 records, 18 digests verified) |
| UP02 | in progress (C) | `codex/up02-symbolic-proof`, wt `can-lang-wt/up02-c` |
| UP03 | done | `9b2005a` merged as `189f2e6`; text suite 9 pass, lint/check clean (coordinator-verified) |
| UP04 | in progress (B) | `codex/up04-owner-context`, wt `can-lang-wt/up04-b` |
| UP05–UP27 | not started | per dependency ledger in `post-upgrade-implementation-tasks-2026-09-24.md` |

## Coordinator decisions

- Browser pins for the new acceptance: Playwright **1.55.1** revisions
  **chromium@1193 + webkit@2203** (`tests/integration/browser`). Actual
  `Chromium x.y / WebKit a.b` version strings are recorded from the
  operated engines at UP23/UP25; prior T27 versions
  (140.0.7339.186 / 26.0) are history, not pins.
- Local cache holds only `chromium_headless_shell-1243` + ffmpeg (no
  WebKit); WebKit + full Chromium installs are arranged before UP23.
- CI browser setup lives in `verifier.yml` (`playwright install chromium`
  only); `release-qualify.yml` is a macOS offline smoke test with no
  browsers. UP24 extends coverage to Chromium **and** WebKit.
- Per-child model selection is unavailable to native subagents (all
  inherit the coordinator route); task-list Sol/Astra guidance is
  recorded but not enforceable per worker.

## Prerequisite gaps (arrange before UP22–UP25)

- [ ] WebKit + full Chromium engines (before UP23)
- [ ] Selected Linux target/root + installed-distribution procedure (UP25)
- [ ] Pinned Linux bun archive + `CAN_BUN_ARCHIVE`/`CAN_TEST_POSTGRES_URL`/`CAN_TSC` (UP24/25)
- [ ] PostgreSQL service `can@127.0.0.1:5433/can_test` (UP22/24)
