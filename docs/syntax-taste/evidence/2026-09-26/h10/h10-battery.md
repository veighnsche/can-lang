# H10 IC1 validation battery — 2026-09-26

Join revision: main `932924f7` (first validated at `7c2dbc2a`, rebased-forward
onto 9 C06/tempcache main commits, full battery re-run). H10-own reruns use a
dev bundle built from HEAD with the pinned Bun archive (`90987a3a…be1d12f`);
live/heavy legs are consumed through owner seals (revision-pinned below).
Nothing outside `docs/syntax-taste/execution-2026-09-26/` +
`docs/syntax-taste/evidence/2026-09-26/h10/` was modified; `dist/` build output
was gitignored and removed after validation.

## 1. Edit propagation — PASS

| Leg | Command / source | Result |
| --- | --- | --- |
| C-B harness (add-error isolation, edit-and-recheck, oracle both directions) | `CONV_BUNDLE=$PWD/dist/development/can-932924f7-… go test ./tests/failure-conventions/ -count=1 -v` | ok 95s; 11/11 top-level PASS, 0 fail/skip — H10 rerun at HEAD (also 11/11 at `7c2dbc2a`) |
| C-D shared-controls asserts | bundle `canlc assert shared/grid-controls` | 128/128 pass, complete, exit 0 — H10 rerun (C04 sealed 124/124; C06 added 4 notice legs, all green) |
| C-D breaking-edit negative (N1) | `tests/browser-controls/x-r02-1.md:40` (drop `parse_qty` → located `model::make_row: fields::parse_qty is private`) | sealed by C04 (`b7a3e214`): N1–N5 green |
| C-D app asserts | C04 P2/P3/P6 | sealed: grid 378/378, compare 217/217, 379/379 |
| C-I rename propagation | `go test ./compiler/ -run 'TestG05RenameWithBindingToCalleeParameter\|TestG05RenameSharedRecordField\|TestG05RenameCrossFile\|TestG05EditorWorkflow'` | 4/4 PASS — H10 rerun at HEAD (also 4/4 at `7c2dbc2a`) |
| C-I full rename suite | G05 seal | 19/19 + full `./compiler/` green (392s) at `ace91b09` |

## 2. Status honesty — PASS

| Leg | Command / source | Result |
| --- | --- | --- |
| C-G companion protocol v1 | `bun test examples/webhook/companion/protocol.test.ts` | 9/9 pass — H10 rerun |
| C-G companion pair (auth envelope, 401/404/409 truthfulness, claim/lease/dead-letter) | F05 seal (`b9930b47`) | 183 asserts, live authenticated pair, DOC-webhook |
| C-E statuses | `compiler/internal/check/actions.go:370` (204/205/304 exclusion) + `runtime/platform/action-routes.ts:619` + `actionReject(status: 400\|405)` at `:444` | present both layers — H10 audit |
| C-E wire suite | E03 seal (`db6d42b1`) + C03 seal (`9a2ea7b7`) | 44/44 runtime; checker GET/document + POST/swap-inner rules |
| C-C boundary honesty | E04 seal (`ea043250`) | 74/74 live (SQL pool/tx, fetch/action abort, dispatch scope, ingress) |
| C-G host denial floor | `bun test host/conformance/` | 51/51 pass (D02 40 + D03 11) — H10 rerun |
| C-G T2/T3 denial legs | D02 seal (`fb30a330`) + D02-LIVE-1..4 ledger (`host/conformance/live/README.md:65-75`) | runnable legs green; live-browser pinning is IC2 debt, interface complete |

## 3. Failure vocabulary / provenance — PASS

| Leg | Command / source | Result |
| --- | --- | --- |
| C-I check codes | `CAN-CHECK-UNNECESSARY-LOCAL` (`compiler/internal/check/warnings.go:14`), `CAN-CHECK-CAPTURE` (`compiler/internal/check/callables.go` + tests) | defined + tested — H10 audit |
| C-C unknown-write vocabulary | `request-policy.md:24,53,86,103,106` (`unknown-write`, `commit: unknown`, `commit_unknown`), `e04-budget-contract.md` | published, consumed by F (`PROTOCOL.md:99` C-C unknown-write; 404 `unknown` / 409 `lease_lost` / 401 `replay`/`stale`) — H10 audit |
| C-C redaction hook | `bun test runtime/test/request-report.test.ts` (within 25-file run below) | 17/17 pass — H10 rerun (E06 seal match) |
| C-B finite error sets | B01/B02 seals | 11/11 + 14 roots, fail-closed negatives, guided repair |
| C-A step index + occurrence | `,"step:"+loopStep` emitted (`compiler/internal/emit/regions.go:181,213,231-232`); occurrence linkage (A07 W4 seal `c63e6497`) | present — H10 audit + owner seal |
| S3 destructive vocabulary | catalogue `s3::discard_upload` emits `[s3::upload_closed, s3::access_denied, s3::service_error]` (exact) | H10 catalogue read |
| Bulk + redaction rerun | `bun test runtime/test/request-report.test.ts runtime/test/collections.test.ts` | 25/25 pass — H10 rerun (E06 17 + A05 8) |

## 4. Wire / generation identity — PASS

| Leg | Command / source | Result |
| --- | --- | --- |
| C-H handshake contract | `go test ./compiler/internal/driver/ -run 'TestPairedHandshakeIdentity\|TestMetadataWritesAreUnique'` | 2/2 PASS — H10 rerun |
| C-H generation strings | `can-output-generation-v1`, `data-can-generation`, `can-generation` in `distribution/paired-deploy.md` (+ server splice + mismatch schema v1) | published — H10 audit |
| Carrier v1 pin | `x-carrier-protocol: 1` in `PROTOCOL.md:16`, `protocol.ts:86` (`PROTOCOL_VERSION`), `protocol.test.ts:42,122`, `worker.test.ts:58` | consistent both sides — H10 audit |
| C-G chart protocol pin | `d02.chart/1`, policy `2026-09-26.d02-chart`, token env `D02_CHART_COMPANION_TOKEN` | published in `host/companions/chart.ts`, `chart-recipe.md`, `REVIEW-MANIFEST.json` |
| Catalogue identity | sha256 `87c05b44…b9006894b99f` in `catalogue.json` == `runtime/catalogue.ts:10` | match — H10 hash check |
| Browser closure (cross-lane witness) | `TestBrowserClosureSuiteGreen` inside `go test ./host/conformance/` | PASS (55.8s) — H10 rerun |

## 5. Catalogue / runtime generated consistency — PASS

| Leg | Command / source | Result |
| --- | --- | --- |
| Generated-code check | `make catalogue-check` (cataloguegen `--check`) | exit 0, `git status` clean — H10 rerun |
| Catalogue unit suite | `go test ./compiler/internal/catalogue/ -count=1` | ok (0.58s) — H10 rerun |
| Runtime hygiene gate | `bun run check:runtime` | oxlint GREEN, oxfmt GREEN (281 files), tsc GREEN — fully green at join revision (a C06-WIP tsc break seen at `7c2dbc2a` was fixed upstream before the rebase) |
| E-side catalogue merge discipline | A05 E-merge `e7982fa`, C02 append-only C-D slice, E08 S3 slice | single-owner (E) merges only — history audit |

## 6. Conditional republication gate — PASS (1 non-blocking note)

All 13 conditioned resolutions verified published — see
`h10-manifest.md` (conditional table + H10-NOTE-01). Negative-assertion legs
pinning the S3 rename run in-tree:

- `s3::cancel_upload` absent: asserted by `runtime/test/s3-e08-remedy.test.ts:163`
  (E1) and `runtime/test/w5-s3.test.ts:207` (W5-S1); H10 grep confirms zero
  `cancel_upload` hits in `catalogue.json`, `runtime/catalogue.ts`,
  `runtime/`, `compiler/`, `std/` outside those two tests.
- Seal citations for live probe evidence: E02 11/11 (X-R04-1 NEG / X-R04-3
  POS), E05 16/16 (X-R04-2 O1), E07 21/21 (X-R15-1/3 NEG), E08 remedy
  (5/5 + rebases), F02 (X-R10-1 admitted), F03 (3× live slice), H08
  `h08-x-r14-1.md` (no qualified profile), D01 54/54 (tier assignment).

## 7. Owner-seal register (consumed, not rerun — live/heavy)

A01 (`19e45019`, 4 go pkgs) · A02 (`1365d0a6`, 4 go pkgs) ·
A03 (`dd3f3674`, emit/syntax/check) · A04 (`78668ec7`, check+emit, 100k probe) ·
A05 (`94eb71c5`, bun 8/8) · C02 (`ff18f134`, 11/11×3 + 17/17×3 + gate5 12/12) ·
C03 (`9a2ea7b7`, check) · D01 (`b837f710`, 54/54) · E02/E04/E05/E07/E08 (live
suites above) · F01 (`ee928ee1`, 38/38; H10 rerun now 39/39 — +1 ledger leg
since seal, all green) · F02 (`d3e51ded`, live PG+MySQL) · F03 (`bf83f0aa`,
sql pkg + 3× live slice) · F05 (`b9930b47`, 183 asserts) · G01–G04 seals
(format/hover/13 refs/24 completion + full-suite greens) · H06 (`1cf506e7`,
5/5) · H07 (`92f93263`, 36/36; H10 rerun 36/36 within the 75/75 file set).

H10 rerun of the H07+F01 file set:
`bun test runtime/test/ai-budget{,-adapters}.test.ts runtime/test/outbound-{identity,epoch,destination,ledger}.test.ts`
→ 75/75 across 6 files, 0 fail. Split: 24+12 (H07 seal match) /
5+4+7+23 (F01: 39 vs 38 sealed — +1 ledger leg since the F01 seal, green).

## Non-blocking follow-ups (owner-routed, none fail IC1)

- H10-NOTE-01 (E): `request-policy.md` §7 S3 line + conditioned-claim ledger
  still say "conditioned … not wired" — sync to the E07/E08 verdicts.
  Normative artifacts already republished; joined view in `h10-manifest.md`.
- H10-NOTE-02 (D): `host/conformance/admission_test.go` walks top-level
  `internal/`, deleted by `af0f42a2` (concurrent refactor) after the D02/D03
  seals → `TestDeliverablesUnreferenced` fatals on the missing path at HEAD
  (still open after the rebase; the C06 delta touched the file but not the
  walk list). Go conformance otherwise 9 pass + 2 live-debt skips. H10 ran the
  equivalent marker sweep over all existing owned trees at the join revision:
  0 hits — the contract assertion holds; the test needs a one-line owner
  update + focused rerun.
- Coordinator-owned staleness (cosmetic, contradict README + evidence):
  `evidence-index.md` E04 row ("partial slices 1-2") and gate rows (O2,
  X-R15-1/3, RETURNING, X-R14-1, C05, D01 tier still "pending"); G03 worker
  hash `8268bb5b` not in worktree history (main-side `20c45f48` verified).
