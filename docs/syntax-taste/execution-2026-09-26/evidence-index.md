# Evidence index — Can implementation 2026-09-26

One row per task outcome. Link task checkboxes to these entries on completion.
Format: revision, environment, command, result (pass/fail/skip), artifact path.

| Task | Outcome | Revision | Env | Commands / evidence | Result |
| --- | --- | --- | --- | --- | --- |
| — | discovery wave | beb59317 | macOS | workflow `Can discovery and ready queue` (4/5 compact reports) | partial (task-list parse gap closed inline by coordinator) |
| C07 | done (DOC-invoice) | main `6fe2bc46` (worker `e0474712`) | macOS darwin-arm64, bun 1.4.2 dist | `canlc assert examples/invoice`: 344/344 passed, exit 0; intended startup rejection via `env::invalid_name("")` | pass |
| H01 | done (register open) | main `3c672e77` (worker `f197c925`) | n/a (docs) | `distribution/provisioning-register.md` reviewed: 6 provisions tracked separately, credential env-names only, no values | pass |
| H09 | done (DOC-Linux) | main `58060191` (worker `d69be330`) | n/a (docs) | both README sentences corrected; link targets `distribution/target-linux-amd64.json`, `distribution/linux/README.md` verified present; no UP25 pass claimed | pass |
| E01 | done (C-C foundations) | main `2224e1b4` (worker `6e9914f6`) | macOS, bun 1.4.2 | `bun test runtime/test/request-budget.test.ts`: 6/6 pass; `bun run check:runtime` green; new files only (request-budget.ts, test, request-policy.md) | pass |

## Conditional branches

| Gate | Experiment | State | Evidence |
| --- | --- | --- | --- |
| A06 (Q6) | A04 | pending | — |
| B03 (Q5) | B01 | pending | — |
| B04 (Q4) | B02 | pending | — |
| C05 (X-R02-1) | C04 | pending | — |
| O2 (X-R04-2) | E05 | pending | — |
| S3 cancel vs discard (X-R15-1) | E07 | pending | — |
| S3 deadline (X-R15-3) | E07/E09 | pending | — |
| D01 tier | D01 | pending | — |
| RETURNING | F02→F03 | pending | — |
| X-R14-1 profile | H07/H08 | pending | — |
