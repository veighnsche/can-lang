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
| E02 | done (X-R04-1 NEG, X-R04-3 POS) | main `2bf35a4c` (worker `a1b4e8e0`) | macOS, bun 1.4.2, live PG 17.11/MySQL 8.4.11 | 11/11 pass live (25s); `check:runtime` green; cancel()=client-flag-only all dialects, backends run to completion; Request.signal aborts on disconnect | pass |
| B02 | partial slice 1 (task open) | main `c743cb77` (worker `f41fba6b`) | macOS, prebuilt bun 1.4.2 bundle | `canlc assert owner-setup`: 8/8 pass, exit 0; pre-extraction leaky-handler baseline only — extraction, repair comparison, Q4 assessment pending | partial |
| F01 | done (C-G + R14 ledger) | main `82750826`..`ee928ee1` (worker 4 slices) | macOS, bun 1.4.2 | 38/38 pass across 4 outbound test files; `check:runtime` green; additive only (13 new files); fixtures env-names-only | pass |
| B01 | done (W3-helper) | main `c691a127`+`dbafd204`+`50dc8d3c` | macOS, prebuilt bun 1.4.2 bundle | 11/11 legs: two-domain, wrapper², oracle both directions, add-error isolation, negatives; simplification ledger + parity tables; Q5 → inactive | pass |
| B02 | done (W3-owner) | main `c743cb77`+`7834c66f`+`50dc8d3c` | macOS, prebuilt bundle | factory extraction, 14 roots (8 retained), trap probes, fail-closed negatives, guided repair byte-identical helper; Q4 → inactive | pass |
| B03 | inactive (Q5 gate) | — (no code; correct outcome) | n/a | per-layer parity 17/17, helper-once, wrapper only-in-style, +12/domain premium not removable by syntax; trip conditions recorded | inactive |
| B04 | inactive (Q4 gate) | — (no code; correct outcome) | n/a | 9 lines/factory, 1 arm/error, zero helper churn, fail-closed; LD29 stays closed; trip conditions recorded | inactive |
| G02 | done (AU-LSP-hover) | main `0d23fada` (worker `e8b10707`) | macOS, go1.27.1 | gofmt clean; `go test` compiler + driver ok; type-at-offset, declines where definition declines | pass |
| F02 | done (X-R10-1 admitted, RETURNING qualified need) | main `d3e51ded` (worker `ccfb1b22`) | macOS, live PG 17.11/MySQL 8.4.11 | probe + live legs pass (PG 17.11 + MySQL 8.4.11 observed); locking expressible w/o syntax change; RETURNING need only for keyless generated-identity shapes; MySQL rejects RETURNING (needs per-dialect mapping in F03) | pass |
| A04 | done (self-tail lowering + Q6 proof) | main `78668ec7` (worker `23ebcade`) | macOS, go1.27.1, bun 1.4.2 | gofmt clean; `go test` check+emit ok (278s/24s); hazard-free self relays → native while, exactly-once ordered temps, step diagnostics, NOT-LOWERED notes; 100k probe ok ~1ms vs overflow | pass |
| A06 | inactive (Q6 gate) | — (no code; correct outcome) | n/a | state machine + relay aggregation covered; callable-fold excluded with non-syntax refinement path (capture analysis) if A07 ever needs it; worker batch F06-owned; no needed W4 shape excluded | inactive |
| A05 | done (bulk map/set + catalogue) | main `94eb71c5` + E merge `e7982fa` | macOS, go1.27.1/bun 1.4.2 | bun 8/8; `go test` check+emit ok; taskID extended to lane IDs (+focused test); catalogue 288→290; cataloguegen --check ok; ops resolve | pass |
| C03 | done (C-E action wire) | main `9a2ea7b7` (worker `8f61b1b`) | macOS, go1.27.1 | gofmt clean; `go test` check ok; GET/document + POST/swap-inner rules, no per-case mixing, plain routes untouched | pass |
| H07 | done (R14 budget guard) | main `92f93263` (worker `73dd42a8`) | macOS, bun 1.4.2 | 36/36 pass; `check:runtime` green; secret scan clean; within-scope reserve/fence/settle, usage decoding, redaction, no bypass | pass |
| E06 | done (redacted reporting) | main `e3a4148c` (worker `c5499411`) | macOS, bun 1.4.2 | 17/17 pass; `check:runtime` green; boundary reporting, shared claim, fixed 500, no native leakage | pass |
| E04 | partial slices 1-2 (task open) | main `8318712b`+`4b10d46b` | macOS, bun 1.4.2, live PG/MySQL | 23/23 + 15/15 pass; `check:runtime` green; SQL pool/tx bounds; fetch/action/server/disconnect legs pending | partial |
| F03 | done (RETURNING slice + contracts) | main `bf83f0aa` (worker `4e500d8`) | macOS, live PG 17.11/MySQL 8.4.11 | sql pkg ok; live relational slice pass (incl. MySQL mapping leg); integration gap found+fixed: `can_f03` lane DB minted PG+MySQL and recorded in register | pass |

## Conditional branches

| Gate | Experiment | State | Evidence |
| --- | --- | --- | --- |
| A06 (Q6) | A04 | INACTIVE | self_tail_coverage.md: no needed W4 shape excluded; callable refinement path recorded |
| B03 (Q5) | B01 | INACTIVE | x-r06-1.md parity tables + trip conditions |
| B04 (Q4) | B02 | INACTIVE | x-r07-1.md burden measures + trip conditions |
| C05 (X-R02-1) | C04 | pending | — |
| O2 (X-R04-2) | E05 | pending | — |
| S3 cancel vs discard (X-R15-1) | E07 | pending | — |
| S3 deadline (X-R15-3) | E07/E09 | pending | — |
| D01 tier | D01 | pending | — |
| RETURNING | F02→F03 | pending | — |
| X-R14-1 profile | H07/H08 | pending | — |
