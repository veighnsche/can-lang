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
| A01 | done (AU-Q1, AU-Q2-core) | main `19e45019` (worker `adf94696`) | macOS, go1.27.1 | gofmt clean; `go test` check/syntax/driver/emit all ok (357s/0.3s/85s/32s); arm-order error removed, false-first canonicalization, C8→CAN-CHECK-UNNECESSARY-LOCAL warning, CLI exit 0 | pass |
| E03 | done (document runtime) | main `db6d42b1` (worker `5c51a96d`) | macOS, bun 1.4.2 | `bun test` action-document/routes/mount/json: 44/44 pass; `bun run check:runtime` green | pass |
| H02 | done (DB gate ready) | main `2a880eee` (worker `9482ff01`) | macOS docker arm64 | coordinator verified PG 17.11 exact pin + `can_e02/f02/f04/f06` DBs + MinIO live; worker: `mysql.test.ts` 12/12, `mysql-tx.test.ts` 5/5, isolation roundtrip | pass |
| H03 | done (S3 gate ready, protocol scope) | main `2a880eee` (worker `9482ff01`) | macOS docker arm64 | MinIO live verified; worker: SigV4 roundtrip + `s3.test.ts` 19/19; scope note: S3-protocol, not AWS-real | pass |
| H04 | done (blocked, ask recorded) | main `2d784b31` (worker `f9a0acdf`) | n/a (docs) | `distribution/ai-eval-access.md`: TYPESAFE_API_KEY + spend-cap ask; gate stays blocked until user grants | blocked |
| H05 | done (blocked, ask recorded) | main `21991bb3` (worker `494be3de`) | n/a (docs) | `distribution/x86-window.md`: machine designation + exclusive-window ask; UP25 stays blocked until user grants | blocked |
| A02 | done (AU-Q3-core) | main `1365d0a6` (worker `5bcae240`) | macOS, go1.27.1 | gofmt clean; `go test` check/syntax/project/driver all ok; explicit near bindings with located CAN-CHECK-CAPTURE negatives, order preserved, direct calls positional | pass |
| B01 | partial slices 1-2 (task open) | main `c691a127`+`dbafd204` (worker `5f6e6545`+`ec4bd92d`) | macOS, prebuilt bun 1.4.2 bundle | 6/6 pass: retry 35 green, fixed 15 green, 2 negatives fail-as-designed, add-error isolation both styles (unrelated identical); concision measure + Q5 decision still pending | partial |
| A03 | done (document grammar/emission) | main `dd3f3674` (worker `258a8299`) | macOS, go1.27.1 | gofmt clean; `go test` emit/syntax/check all ok (45s/0.3s/454s); per-case document mode, fragment-only swap table, JSON byte-identical; checker agreement handed to C03 | pass |
| C01 | open (blocked provision) | wiring `c37c40fc`+`a3c4a11f` pending in retained worktree, NOT on main | macOS 27, Playwright 1.55.1 | Firefox 141.0/v1490 installs, launch times out (worker + coordinator repro); Chromium 140.0.7339.186 + WebKit 26.0 launch OK; gate5 probe fail-fasts so wiring held back to keep suite green | blocked |
| G01 | done (AU-LSP-format, AU-Q2-LSP) | main `94795946` (worker `31f0e66a`) | macOS, go1.27.1 | gofmt clean; `go test` compiler + driver ok (16s/71s); formatSource+overlay validation, warning-severity publishDiagnostics, CLI parity | pass |

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
