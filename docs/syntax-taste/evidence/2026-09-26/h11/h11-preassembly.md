# H11 pre-assembly — IC2 workload acceptance aggregation (2026-09-26)

Status: **PRE-ASSEMBLY ONLY — no IC2 verdict claimed.**
IC2 stays **OPEN behind H12** (W6-deploy native legs, BLOCKED-awaiting-UP25-grant),
which is the **single pending evidence input**.
All other H11 prerequisites have delivered owner seals; their legs are aggregated
below solely from those seals, with every mismatch, stale pin, and live-requirement
gap recorded as a finding (§6), never as a silent pass.

- Worktree base: `7480a791` (post-H12-staging main).
- Authority: H11 task (`docs/syntax-taste/can-implementation-task-list-2026-09-26.md:770-781`),
  P08.4 IC2 (`docs/syntax-taste/preparation-2026-09-26/p08-lanes.md:71-84`),
  P07 contracts/acceptance/dispositions, execution README + evidence-index.
- Method: aggregation of owner reports only. No live/heavy leg was rerun here,
  no evidence double-counted, no unavailable-live leg passed. H10 IC1 seals are
  consumed for joined interface slices; H10's own reruns are cited as supporting,
  never as a substitute for a workload-owner seal.

## 1. Prereq seal register

| Prereq | Seal artifact(s) | Revision pin | Env (as sealed) | Verdict in seal |
| --- | --- | --- | --- | --- |
| H10 (IC1 GREEN) | `execution-2026-09-26/h10-ic1-record.md`, `evidence/2026-09-26/h10/h10-manifest.md`, `h10-battery.md` | join main `932924f7` (first `7c2dbc2a`) | macOS, go1.27.1/bun 1.4.2, pinned darwin archive `90987a3a…be1d12f` | IC1 GREEN, 5/5 battery |
| A07 (W4 core) | `evidence/2026-09-26/a07/a07-report.md`, `a07-legs.json` | main `c63e6497` (workers `41996b95`,`0ac01bab`,`d8fa6025`) | macOS darwin-arm64, bun 1.4.2 (hash-verified), go1.27.1 | W4.1/2/4/5 PASS |
| A08 (comparisons) | `tests/authoring-policies/x-r08-comparison-2026-09-26.md`, `registry.json` + baselines/references | shipped surface `e3a4148c`; pre-change `ba619613` | darwin-arm64, go1.27.1, bun 1.4.2 isolated bundle | 3 records complete, deterministic legs only, 0 agent attempts |
| B01 (W3-helper) | `tests/failure-conventions/x-r06-1.md` | main `c691a127`+`dbafd204`+`50dc8d3c` | macOS, prebuilt bun 1.4.2 bundle, go1.27.1 | 11/11 legs; Q5 → inactive |
| B02 (W3-owner) | `tests/failure-conventions/x-r07-1.md` | main `c743cb77`+`7834c66f`+`50dc8d3c` | macOS, prebuilt bundle | 14 roots; Q4 → inactive |
| B03 (Q5 gate) | `x-r06-1.md` §Q5 assessment + trip conditions | — (no code; correct outcome) | n/a | INACTIVE |
| B04 (Q4 gate) | `x-r07-1.md` §Q4 assessment + trip conditions | — (no code; correct outcome) | n/a | INACTIVE |
| B05 (W3 final) | `evidence/2026-09-26/b05/b05-report.md`, `b05-legs.json` | base `b283d2ab` (STALE, pre-emitfix — see F4) | macOS, go1.27.1, `CAN_BUN_ARCHIVE` sha `90987a3a…be1d12f` | BLOCKED (emitter defect, since fixed by `b636674d`) |
| C06 (W1) | `evidence/2026-09-26/c06/c06-report.md`, `c06-legs.json`, `legs/` | main `932924f7` (10 worker → main; main re-qual green) | macOS + container FF141, Playwright 1.55.1, Ch140.0.7339.186/Wk26.0/FF141.0 | 99/99 served + check/build legs PASS |
| C07 (DOC-invoice) | commit `6fe2bc46` + evidence-index C07 row | main `6fe2bc46` (worker `e0474712`) | macOS darwin-arm64, bun 1.4.2 dist | `canlc assert examples/invoice` 344/344 |
| D03 (W2) | `host/d03-record.md`, `host/conformance/` | worktree `a6eb8f7b` + lane-D `3c1475b7`,`0c5f6093`,+slice3 → main `4d7efdbd` | Bun 1.4.2, go1.27.1, node 24.21.0 | runnable PASS + LIVE DEBT (D02/D03-LIVE-1..4) |
| E09 (W5) | `docs/implementation/evidence/2026-09-26/e09-w5.md` + `w5-lifetime.test.ts` + `w5-s3.test.ts` | main `784b2c2b`+`fa812d4a` | Bun 1.4.2, PG 17.11:55433, MySQL 8.4.11:3307, MinIO, isolated `can_e09_run1`/prefix/ports | 16/16 harness (L3 + S5b BLOCKED-evidence) |
| F04 (W6-data) | `evidence/2026-09-26/f04/f04-report.md`, `f04-live-report.json`, operator-DDL recipe, backend recommendation | main `217654f8` (workers `b8c52ca3`,`bea2a8a4`,`53ef28f9`) | PG 17.11:55433, MySQL 8.4.11:3307, SQLite 3.54, `can_f04_run1`/`run2` | 47/47 ×3 + 59/59 regress |
| F06 (W6-pair/W4.3) | `evidence/2026-09-26/f06/f06-report.md`, `f06-legs.json` | worker evidence run `e1ca3e55` → main `feb0a934` (+coord main rerun) | macOS darwin-arm64, bun 1.4.2 pinned archive, go1.27.1 | live PASS A–I + W4.3 |
| G05 (rename/workflow) | `evidence/2026-09-26/g05/g05-report.md` | worker `fc85d837`,`ace91b09` (main hash unrecorded — see F16) | macOS darwin/arm64, go1.27.1 | 19/19 + full `./compiler/` green |
| H08 (W6-AI) | `evidence/2026-09-26/h08/h08-report.md`, `h08-registration.md`, `h08-x-r14-1.md`, `h08-model-change-recipe.md`, `h08-offline-plumbing-run.json` | main `b4c4c33d` (5 worker) | macOS, bun 1.4.2 (offline; zero live calls) | harness 12/12; W6-AI BLOCKED honest |
| H12 (W6-deploy) | `evidence/2026-09-26/h12/h12-staging.md` (+`jev/`), `distribution/h12/window-check.sh` | staging main `275dd42d` (base `4d67bdb6`, post-IC1) | staging on macOS; native run pending | STAGED, 0 legs executed — §7 pending input |

Supporting (prereqs-of-prereqs, consumed via H10 + index, not re-aggregated as
workload owners): A01/A02/A04/A05/A06✗, C02/C03/C04/C05✗, D01/D02, E01/E02/E04/E05/E06/E07/E08,
F01/F02/F03/F05, G01/G02/G03/G04, H01–H07/H09. F02/F03 reports are cited as W6-data
inputs (§3.6); G04/G01–G03 seals are cited for authoring legs (§3.9).

## 2. Gated syntax: inactive-or-qualified (per the H10 conditional table)

Source: `evidence/2026-09-26/h10/h10-manifest.md:42-61`. No gated syntax ships
active-but-unqualified:

| Gate | Outcome | Status for H11 |
| --- | --- | --- |
| Q6 → A06 (explicit iteration primitive) | INACTIVE — no needed W4 shape excluded (`a07-report.md:113-120`) | inactive ✓ |
| Q5 → B03 (finite error sets) | INACTIVE — parity 17/17, premium not syntax-removable (`x-r06-1.md:154-173`) | inactive ✓ |
| Q4 → B04 (checked setup) | INACTIVE — 9 lines/factory; LD29 closed (`x-r07-1.md:94-111`) | inactive ✓ |
| X-R02-1 → C05 (nested-view ownership) | INACTIVE — root-only rule stands | inactive ✓ |
| X-R04-1 / X-R04-3 | NEGATIVE cancel-absent / POSITIVE disconnect branches, published | qualified branch ✓ |
| X-R04-2 | O1 SUFFICIENT, O2 INACTIVE | qualified/inactive ✓ |
| X-R15-1 | NEGATIVE — `cancel_upload` REMOVED, `discard_upload` destructive contract | qualified branch ✓ |
| X-R15-3 | NEGATIVE — between-awaits bound only; W5 S3-deadline legs BLOCKED by design | qualified branch, W5 fails per contract (see F1) |
| X-R10-1 / RETURNING | ADMITTED per-dialect with published contract (`f03-returning-contract.md`); no new grammar. Runtime execution path unsealed — see F8 | checker-qualified; execution gap flagged |
| X-R14-1 | NO metering profile qualifies; H07 guard fails closed | qualified rejection; live blocked (see F3) |
| D01 tier | A/B→T2, C→T3 | qualified ✓ |

## 3. Per-workload leg aggregation (owner seals only)

Leg IDs per the task-list acceptance-ownership table
(`can-implementation-task-list-2026-09-26.md:855-908`).

### 3.1 W1 — second application (sole owner: C06)

| Leg | Owner verdict | Citation (pin: main `932924f7`; env: Playwright 1.55.1, Ch140.0.7339.186/Wk26.0/FF141.0) |
| --- | --- | --- |
| W1.1 shared controls unmodified in two apps × 3 browsers | PASS | `c06-report.md:110-127` (99/99 served: compare 13×3, w1-grid 12×3, drift 4×3×2); `c06-legs.json` matrix; check/build `c06-report.md:129-135` (grid 394/394, compare 221/221, server 347/347 real-can); main re-qual same counts `c06-report.md:151-176` |
| W1.2 breaking control-API edit diagnoses unedited app | PASS | `TestC06NegativeAPIBreak` — dropped `parse_qty` fails both consumers with located `fields::parse_qty is private` (`c06-report.md:132-133`) |
| W1.3 route/capture/wire/body/leaf edits propagate or diagnose | PASS | `TestC06CaptureEdits` 5/5 sublegs (`c06-report.md:133-135`, `164`) |
| W1.4 late reply/unmount, disposal mid-save, IME/caret | PASS | w1-grid + compare legs incl. caret/composing-input, drop-left disposal, silence-after-drop (`c06-legs.json` `w1-grid-*`, `compare-*`); mismatch save/load prompts + refresh-resolves (`c06-report.md:33-38`) |
| W1.5 paired captured HTML reads, truthful denied/not-found | PASS on substitute surface — see F7 | missing-id/page denials + 404 legs (`c06-legs.json` `w1-grid-*`); honest N/A: no document-mode GET consumer exists (`c06-report.md:70-86`) |

Version/env cross-check: worker run (643s) and main re-qual (942s) use the same
pinned toolchain (`CAN_BUN_ARCHIVE` pinned zip, container FF141 via
`CAN_FIREFOX_WS`, Playwright 1.55.1); counts match exactly — no drift. Firefox
legs run on linux/arm64 while Ch/Wk run on macOS (pinned skew, C01 design);
FF UA arch string needs one-line clarification (F11, low).

### 3.2 W2 — second integration (sole owner: D03)

| Leg | Owner verdict | Citation (pin: main `4d7efdbd`; env: Bun 1.4.2, go1.27.1, node 24.21.0) |
| --- | --- | --- |
| W2.1 second vendor, no vendor-specific compiler patch / hand-edited generated code | PASS (runnable) | Vendor B reuses unchanged D02 client; D03 diff `host/`-only; catalogue/generated byte-identical; 11 vendor-B + all D02 legs green (`host/d03-record.md:18-46,57-75`) |
| W2.2 located unadmitted-capability rejection + lifecycle | PASS (runnable) + GAP | file-attributed `unknown package` rejections + read-only closure witness green; no end-to-end `canlc build --target browser` leg — recorded gap, not a pass (`host/d03-record.md:78-96,134,147-153`; see F6) |
| W2.3 assigned conformance in CI shape | PASS (runnable) + LIVE DEBT | `bun test host/conformance/` 51/51 + `go test` 9 pass/2 skip; live gates skip on D02/D03-LIVE-1..4 (`host/d03-record.md:135`; `host/conformance/live/README.md:65-75,138-151`; see F5) |

Version/env cross-check: bun + go suites run in one env; consistent. Debt was
recorded "until C01 unblocks" and C01 is now done — live legs are runnable but
unrun (F5). Base predates C01-done (stale pin for live purposes).

### 3.3 W3 — reusable infrastructure (sole owner: B05; B01/B02/B03✗/B04✗ inputs)

| Leg | Owner verdict | Citation |
| --- | --- | --- |
| W3.1 two error sets; oracle attempts/failure→success | BLOCKED in owner seal (base `b283d2ab`); green only in non-owner reruns — see F4 | owner: `b05-report.md:36-41`, `b05-legs.json:78-83`; inputs: `x-r06-1.md:105-152` (B01 11/11). Supporting (non-owner): H10 battery 11/11 rerun at `932924f7` (`h10-battery.md:15`), coordinator W3-rerun-green row (execution README B05) |
| W3.2 add-error isolation | same split as W3.1 | owner `b05-report.md:39`; input `x-r06-1.md:90-103` |
| W3.3 extraction + measured costs | same split as W3.1 | owner `b05-report.md:40`; input `x-r07-1.md:16-92` (9 lines/factory, helper byte-identical) |
| W3.4 fail-closed + no forge | same split as W3.1 | owner `b05-report.md:41`; inputs `x-r07-1.md:45-56` + check-time forge rejection |

Version/env cross-check: B01/B02 ran at `beb59317`+lane-B via prebuilt bun 1.4.2
bundle; B05 at `b283d2ab` via `CAN_BUN_ARCHIVE` pinned zip; both bun
1.4.2/go1.27.1 — provisioning differs by design, no version mismatch (F19,
info). The B05 seal predates emitfix `b636674d`; no B05 owner re-seal exists
at a post-emitfix revision (F4, medium).

### 3.4 W4 — large iteration (sole owners: A07 core, F06 batch)

| Leg | Owner verdict | Citation |
| --- | --- | --- |
| W4.1 100k-step state machine; stack/memory 100/20k/100k | PASS (scoped — see F10) | machine 100k=199999, scan 100k=5000050000 vs oracles; scalar-machine heap ~0 at 100k/1M; linear time (`a07-report.md:34-57`; `a07-legs.json` `machine`/`scan`). Env: macOS, bun 1.4.2 hash-verified, go1.27.1 |
| W4.2 growing bulk aggregation; order/failure identity; no quadratic history | PASS | 100k order identity; 20k→100k ~4x (linear band [2,10]); dup-key whole-build `key_exists` failure; spread contrast superlinear, not shipped (`a07-report.md:59-79`) |
| W4.3 bounded companion batch mirroring C-A step/failure/occurrence | PASS | 200 rows/6 batches/3.4s (59 rows/s), peak RSS 57,616 KiB < 256 MiB gate; every step carries mirror fields (`f06-report.md:68-83`; `f06-legs.json` `W4.3-batch-mirror`). Step-base note F9 |
| W4.4 non-lowerable recursion unchanged + not-lowered note | PASS | triangle/mutual/callable controls: nested calls, exact small results, 3× `CAN-CHECK-NOT-LOWERED` (`a07-report.md:81-91`) |
| W4.5 step-60k fault: declared failure + step + occurrence, no overflow | PASS | standard (`step:60000` + occurrence), loop-originated domain (`step:60000`), callee-created domain (creation-site origin kept, no step — pinned language rule) (`a07-report.md:93-111`) |

Version/env cross-check: A07 (installed `CAN_BUN` binary, hash-verified) and F06
(`CAN_BUN_ARCHIVE` pinned zip) both bun 1.4.2/go1.27.1 on macOS darwin-arm64 —
no mismatch (F20, info). F06 evidence run at worker `e1ca3e55`; main carries
harness (`7e8d2842`) + record (`feb0a934`); coordinator reran green on main
(146s). A06 stays INACTIVE (`a07-report.md:113-120`).

### 3.5 W5 — controlled failures (sole owner: E09)

Env (all legs): Bun 1.4.2, PG 17.11 on 127.0.0.1:55433, MySQL 8.4.11 on
127.0.0.1:3307, MinIO on :9000, isolated DB `can_e09_run1`, prefix
`s3e09/e09harness/<run>/`, ports 18790–18799 (`e09-w5.md:7-18`).

| Leg | Owner verdict | Citation |
| --- | --- | --- |
| W5.1 stalled SQL: verified cancel or owned-until-settlement branch | PASS (L1/L2); L3 BLOCKED (see F2) | L1 PG / L2 MySQL bounded `unknown:budget` + escalation + owned settlement (`e09-w5.md:43-44`); L3 handler-ok/peer-blocked (`e09-w5.md:45,54-86`) |
| W5.2 stalled headers, visible bound + honest ownership | PASS | L4 dispatch-fires-0; L5 413/408 over the wire (`e09-w5.md:46-47`) |
| W5.3 stalled body, visible bound + honest ownership | PASS | L5 (`e09-w5.md:47`) |
| W5.4 disconnect: qualified abort or cancel-absent branch | PASS | L6 (`e09-w5.md:48`) |
| W5.5 overlap/shared-pool, no post-disposal use | PASS | L7 (`e09-w5.md:49`) |
| W5.6 SIGTERM, drainage, supervisor escalation | PASS | L8 (`e09-w5.md:50`) |
| W5.7 once-only redacted fixed-500 hook incl. double-throw | PASS | L9/L10 + silence legs (`e09-w5.md:51-52`) |
| W5.8 S3 O1 preservation or O2 destructive/unknown evidence | PASS | S1–S4: name removal, discard destroys seed, observer window, orphan recipe (`e09-w5.md:97-100`) |
| W5.9 hung read/write/flush/end/stat bounded; deadline-negative fails W5 | BLOCKED by design (see F1) | S5b hung `source.read` pins unbounded; E07 H1–H5 pin remainder (`e09-w5.md:102,104-107`) |
| W5.10 S3 cleanup failures, pin release, abort, orphan risk | PASS | S4 + E07 F1–F7 (`e09-w5.md:100,120`) |

Prereq re-runs on final adapters all green: E04 61/61 + 38/38, E05 16/16,
E07 21/21, E08 5/5 + s3 19/19, W5 16/16; `check:runtime` green
(`e09-w5.md:122-141`). Full `bun test runtime/test/` shows 1038 pass / 17
pre-existing environmental fails (16 mysql URL-shape + 1 Playwright module;
no W5/prereq file fails) — info, F22.

### 3.6 W6-data — real PG app + MySQL parity (sole owner: F04; F02/F03 inputs)

| Leg | Owner verdict | Citation |
| --- | --- | --- |
| W6.1 real PG app: generated identities, precise values, nullable audit times, JSON, replay, executed operator-DDL recipe | PASS (ledger backend + shared-contract inputs; notes F8/F14) | F04: 47/47 ×3 across PG/MySQL/SQLite + 59/59 regress; per-run DBs; operator-DDL recipe + live CHECK-floor/verify probes (`f04-report.md:41-60`; `f04-live-report.json` `suite`/`services`). Inputs: F02 ambiguity/keyed/cost/MySQL-1064 legs (`f02-report.md:31-49`); F03 native RETURNING rows, cardinality vocabulary, PG aborted-txn+replay, 8-writer decide, encoding roundtrips incl int64, MySQL `LAST_INSERT_ID` mapping (`f03-report.md:28-34`); encodings incl nullable audit instants + JSON opaque-TEXT+codec (`f03-encoding-guide.md:28-40`) |
| W6.2 live MySQL parity + retained SQLite regression | PASS | MySQL qualified alongside PG/SQLite; strict-mode parity fix (digest quarantine key, TEXT metering cols) (`f04-report.md:26-31`; `f04-live-report.json` `verdicts`/`parity_fix`) |

Version/env cross-check: F02, F03, F04, E09 all pin PG 17.11 on :55433 +
MySQL 8.4.11 on :3307 + SQLite 3.54 — exact match. F04 backend recommendation
(PG default; memory/sqlite-memory disqualified) honored by H08 (`f04-report.md:64-68`;
`h08-report.md:83-89`).
Live-requirement notes: Can-path RETURNING execution unsealed (F8, medium);
literal operator-shell path runs on UP25 in H12 Leg 7 (F14, low); E04
unexercised by F04 (F18, low); F03 cold-pool E defect still open (F17, low).

### 3.7 W6-pair — companion operations (sole owner: F06)

Env: macOS darwin-arm64, bun 1.4.2 pinned archive, go1.27.1; fresh SQLite file
per run; live port 18495 (`f06-report.md:7-28`; `f06-legs.json`).

| Leg | Owner verdict | Citation |
| --- | --- | --- |
| W6.3 two-worker claims/concurrency/backoff/poison/crash/restart, versioned auth, destination enforcement | PASS (legs A–I) | A-auth, B-pair 24/24 split + idempotent redrain, C-concurrency 1→1/4→4, D-poison 5 attempts then dead-letter, E-destination/F-credential denials without downstream touch, G-timeout bound, H-crash at-least-once redelivery (4 once/4 twice, ledger 1/row), I-backoff/idle + SIGTERM drain (`f06-report.md:30-66`; `f06-legs.json` legs A–I) |

Notes: no exactly-once claim (duplicates proven, collapsed on `delivery_id` +
idempotent ack); SQLite-only Can side is a documented deviation from the
PG/MySQL-mkdb task letter (F13, low); no `runtime/` edits, no RH duty.

### 3.8 W6-deploy — paired deployment on native x86 (sole owner: H12)

**No legs executed — the single pending evidence input (§7).**
Staged: 8 validation legs with exact commands/env/pass-criteria
(`h12-staging.md:94-238`), window-entry gate (`distribution/h12/window-check.sh`),
pins re-verified (IC1 `932924f7`, catalogue sha `87c05b44…` matching
`h10-manifest.md:38-39`, bun zip `36368fae…a913` 36646985 B, bun rev
`744846f84`, go1.27.1, PG17, Playwright 1.55.1/Ch140).
Per-leg staging completeness: `h12-staging.md:267-284`. Zero x86
execution/emulation; C06 off-target results explicitly not imported
(`h12-staging.md:240-265`; `h12-staging-record.md:40-45`).

### 3.9 W6-AI — tenant budgets + model-change eval (sole owner: H08)

| Leg | Owner verdict | Citation (pin: main `b4c4c33d`; offline, zero live calls) |
| --- | --- | --- |
| W6.5 enforced/measured tenant AI budgets/correlation + one feature model-change quality/cost/latency eval | BLOCKED honest (see F3) | harness `runtime/test/ai-eval-triage.test.ts` 12/12 incl. zero-dispatch rejections, exact token sums (1156+276), unknown-hold, quarantine, serial proof, PG metered leg (`can_h08_run1`); raw regression 106/106; offline PG plumbing run ledger 1432==1432, face-labeled `provenance: stub, evidence: false` (`h08-report.md:37-71`; `h08-offline-plumbing-run.json`); frozen protocol v2 + registration hashes (`h08-registration.md`); recipe (`h08-model-change-recipe.md`); X-R14-1: no profile qualifies (`h08-x-r14-1.md`) |

Missing inputs (all three required): spend-cap approval, pinned per-token USD
price table, qualified complete-call bound U (`h08-report.md:9-23`).
`TYPESAFE_API_KEY` present (presence only); no OpenAI credential adopted
(`h08-report.md:25-35`).

### 3.10 Authoring / tooling (owners: A01/A02/G01–G05 via H10 + seals; A08)

| Leg | Owner verdict | Citation |
| --- | --- | --- |
| AU-Q1 Boolean orders + false-first fixpoint/overlay | PASS | A01 seal: go test 4 pkgs ok, gofmt clean (evidence-index A01 row; H10 seal register `h10-battery.md:83`) |
| AU-Q2-core C8 warned, source kept, differential, exit 0 | PASS | A01 seal (same row) |
| AU-Q2-LSP advisory severity in publishDiagnostics | PASS | G01 seal: compiler+driver go test ok (evidence-index G01 row) |
| AU-Q3-core `with` check/emit/capture + CAN-CHECK-CAPTURE negatives | PASS | A02 seal: check/syntax/project/driver ok (evidence-index A02 row) |
| AU-Q3-rename safe rename incl. `with` pins + fallback coupling | PASS | G05 legs 6 + 14 (`g05-report.md:79,87,106-111`) |
| AU-LSP-format whole-document validated formatting | PASS | G01 seal (same row); H10 battery consumed |
| AU-LSP-hover checked contract inspection | PASS | G02 seal (evidence-index G02 row) |
| AU-LSP-references project + function-local, binding identity | PASS | G03: 13 TestG03 pass + suite green (evidence-index G03 row) |
| AU-LSP-completion scope/arity/near precision | PASS | G04: 24/24 + full-suite green incl. retirement gate (`g04-report.md:113-122`) |
| AU-LSP-rename full ordered workflow | PASS | G05 leg 13: format→hover→refs→complete→rename→apply→repair proof (`g05-report.md:85,120-127`); 19/19 + full `./compiler/` 392s green (`g05-report.md:131-135`) |
| X-R08-1/2/3 authoring-policy comparisons | COMPLETE (deterministic legs only; 0 agent attempts — see F15) | 3 records + comparison table + neutral/adverse + limitations + H14 handoff (`x-r08-comparison-2026-09-26.md`; `registry.json` `attemptsRun: 0`, frozen `frozen-x-r08`) |

Version/env cross-check: A01/A02/G01/G02/G03 seals span wave-1/2 mains; G04/G05
worker slices consumed by H10 at join `932924f7` with H10 rerun of rename 4/4
at HEAD (`h10-battery.md:19`) and coordinator re-verification
(`h10-ic1-record.md:80-102`). G04/G05 main-integration hashes unrecorded
(F16, info). A08 deterministic legs executed at `e3a4148c` (+`ba619613`
pre-change probes); bundle `a08-verify` — no mismatch, recorded.

### 3.11 Pre-publication examples (owners: C07, F05, H09; H14 owns final closeout)

Final DOC-story/DOC-assert closeout remains H14 after IC3 and is NOT an IC2
prerequisite (H11 task record). Aggregated here as pre-publication inputs only:

| Leg | Owner verdict | Citation |
| --- | --- | --- |
| DOC-invoice selected authored startup failure + truthful comment | PASS (seal predates app edits — see F12) | `6fe2bc46` bare `env::invalid_name("")` + comment rewrite; `canlc assert examples/invoice` 344/344 (evidence-index C07 row). Join-revision app coverage via C06 `TestC06Positive` (grid 394/compare 221/server 347 at `932924f7`) |
| DOC-webhook authenticated carrier envelope + honest limits | PASS | F05 seal: 183 asserts, live authenticated pair, carrier protocol v1 (`evidence-index.md:59`; H10 battery `h10-battery.md:26-27`) |
| DOC-Linux two sentences accurate, no premature claim | PASS | H09 seal: both README sentences corrected, link targets verified (evidence-index H09 row) |

## 4. Workload version/env cross-check summary

| Workload | Legs share env/pins? | Result |
| --- | --- | --- |
| W1 (C06) | worker 643s ≡ main re-qual 942s (same pinned toolchain, counts match); FF linux/arm64 vs Ch/Wk macOS is pinned C01 skew | MATCH (+F11 info) |
| W2 (D03) | bun 51/51 + go 9+2skip in one env | MATCH internally; live pin stale vs C01-done (F5) |
| W3 (B01/B02/B05) | bun 1.4.2/go1.27.1 throughout; bundle vs archive provisioning by design | MATCH on versions; B05 seal revision stale (F4) |
| W4 (A07/F06) | bun 1.4.2/go1.27.1/macOS both; `CAN_BUN` binary vs `CAN_BUN_ARCHIVE` by design | MATCH (+F20 info); step-base note F9 |
| W5 (E09) | single env block for all 16 legs + prereq re-runs | MATCH |
| W6-data (F02/F03/F04) | PG 17.11:55433 + MySQL 8.4.11:3307 + SQLite 3.54 exact across all three | MATCH |
| W6-pair (F06) | single harness run A–I + stability rerun | MATCH |
| W6-deploy (H12) | staging only; pins match H10 manifest (catalogue sha, IC1 rev) | MATCH (staged) |
| W6-AI (H08) | offline harness + PG plumbing on F04-default backend | MATCH (blocked) |
| Authoring (A/G) | seals span wave-1/2 → join `932924f7`; H10 reran subsets at HEAD | MATCH via H10 join (+F16 info) |
| Examples (C07/F05/H09) | C07 seal predates C06 app edits; join coverage via C06 positive legs | NOTE F12 |

## 5. Double-count / unavailable-live audit

- Experiment records (B01/B02 x-r06/x-r07, F02 probe, F03 slice, E02/E05/E07/E08
  verdicts, A08 comparisons) are cited as *inputs* to final-owner legs, never as
  duplicate leg owners — per the task-list acceptance-ownership rule (one final
  owning task per leg).
- H10 battery reruns (C-B 11/11, rename 4/4, companion 9/9, host 51/51, C-H 2/2,
  redaction+bulk 25/25) are cited as supporting context, never as workload-owner
  evidence — notably NOT as a substitute for the stale B05 owner seal (F4).
- C06's off-target (macOS) results are not imported into any UP25 verdict (H12
  live-probe policy). D03's unpinned-browser supplement (HeadlessChrome 153) is
  cited as non-qualifying, closing no debt item. H08's stub plumbing run is
  face-labeled `evidence: false`. E09's BLOCKED-evidence legs pass as *shape
  pins*, never as accepted live passes.

## 6. Findings (severity + routing)

Severity: BLOCKING (gates any future IC2-green) / HIGH (verdict-blocking
preparation-or-grant item) / MEDIUM (owner re-seal or disposition required) /
LOW (accepted-with-note or owner clarification) / INFO (recorded, no action).

- **F0 — H12 W6-deploy: zero legs executed, BLOCKED-awaiting-UP25-grant.**
  Severity: BLOCKING (the single pending evidence input, §7). Not a defect —
  staging is complete. Routed: user UP25 grant → H12 native run.
- **F1 — E09 W5-S5b (+E07 H1–H5) BLOCKED by design (X-R15-3 negative).**
  Severity: HIGH. Hung S3 awaits unbounded on final adapters; per the task-list
  experiment table the X-R15-3 negative *fails W5* and needs explicit user scope
  return — "no silent reduced scope or green IC2"
  (`can-implementation-task-list-2026-09-26.md:933`). A future IC2-green is
  impossible without that scope return. Routed: preparation + user scoping.
  (`docs/implementation/evidence/2026-09-26/e09-w5.md:102-107`)
- **F2 — E09 W5-L3 at-budget peer response BLOCKED (new E09 finding).**
  Severity: HIGH. Overrun-tx peer gated on tx-callback settlement via the
  request-scope drain (handler 155ms vs peer 2009ms); detaching tx ownership is
  an E04 semantic change, not a leg fix. Leg pins the split and fails loudly on
  change. Routed: preparation return (E owner). (`e09-w5.md:54-86`)
- **F3 — H08 W6-AI BLOCKED honest: zero live provider calls.**
  Severity: HIGH. Missing spend-cap approval, pinned per-token USD price table,
  and qualified complete-call bound U; X-R14-1: no profile qualifies, guard
  fails closed before send. Per H08's definition of done this blocks W6-AI;
  rejection-only fixtures are not a live pass. Routed: user (H04 ask at
  `distribution/ai-eval-access.md:35-44` + price table) + provider-published
  enforced bound + live verification. (`h08-report.md:3-35`; `h08-x-r14-1.md`)
- **F4 — B05 W3 owner seal stale (BLOCKED at `b283d2ab`, pre-emitfix).**
  Severity: MEDIUM. Post-emitfix (`b636674d`) W3-green exists only from
  non-owners (coordinator README row; H10 battery 11/11 at `932924f7`). Per the
  sole-owner rule, W3 aggregation rests on a BLOCKED owner seal plus supporting
  reruns. Routed: B owner — re-seal W3 at a post-emitfix revision via the single
  rerun command in `b05-report.md:15-22`. (`b05-report.md:6-13`; `b05-legs.json:4-6`)
- **F5 — D02/D03 live-browser debt unrun (D02-LIVE-1..4, D03-LIVE-1..4).**
  Severity: MEDIUM. Recorded "until C01 unblocks"; C01 is now done (container
  FF141 + native Ch/Wk, Playwright 1.55.1), so the unblock commands are runnable
  but unrun and no `report.json` is attached. D03 base predates C01-done (stale
  pin for live purposes). Routed: D owner — run both unblock commands on all
  three pinned browsers at a current revision. (`host/conformance/live/README.md:65-75,138-151`;
  `host/d03-record.md:3-10,172-173`)
- **F6 — D03 W2-negative end-to-end gap (no `canlc build --target browser` leg).**
  Severity: MEDIUM. Runnable evidence is check-phase rejection + read-only
  closure witness; the full browser-build negative needs H-packaged distribution
  (`CAN-DIST-UNBUNDLED` in worktree). Recorded as a gap for H11, not a pass.
  Routed: H (packaging) + D (leg), or IC2 disposition accepting check-phase
  evidence. (`host/d03-record.md:134,147-153`)
- **F7 — W1.5 evidenced on a substitute surface (no captured document-mode GET consumer).**
  Severity: MEDIUM. No program declares a document-mode GET action, so C06
  evidences denied/not-found on captured JSON loads + query-addressed pages
  through paired builds; compare makes no server calls by design; form-page
  prompt is follow-up. Whether the substitute satisfies W1.5 ("paired captured
  HTML reads") needs IC2/preparation disposition. Routed: preparation/IC2
  disposition. (`c06-report.md:70-86`)
- **F8 — Can-path RETURNING execution unsealed (checker admits, runtime lacks).**
  Severity: MEDIUM. F03 admits `INSERT...RETURNING` (`one`, PG/SQLite) and
  specifies the runtime path (manifest `row_limit 0`, descriptor `(one,0)`,
  `pool.ts` execution) as an E/coordinator handoff (`f03-report.md:51-57`);
  F03's own legs use raw-SQL oracles (`f03-live-driver.ts:11-13,295-302`); no
  RETURNING execution path exists under `runtime/` (grep: only a `cli.ts:1`
  comment hit). W6.1 "generated identities" is evidenced via raw oracles + MySQL
  mapping + ledger identities, not end-to-end Can SQL. Routed: E owner (runtime)
  + coordinator (`project/` manifest routing); IC2 disposition on whether W6.1
  needs the Can-path leg.
- **F9 — W4 step-index base differs between A07 (0-based) and F06 mirror (1-based).**
  Severity: LOW. A07 handoff: `step:<0-based iteration>` (`a07-report.md:124-128`);
  F06 W4.3: 1-based contiguous claim-order index (`f06-report.md:74-77`).
  Occurrence linkage present both sides. Routed: A/F owners — confirm the mirror
  contract documents the base difference (loop iteration vs batch claim order).
- **F10 — A07 W4.1 memory scoping (scan legs recorded, not gated).**
  Severity: LOW. JSC conservative-stack nondeterminism; loop flatness proven via
  scalar-machine legs (~0 at 100k/1M) sharing the exact loop shape. Routed: IC2
  disposition (accepted scoping note). (`a07-report.md:47-53`)
- **F11 — C06 Firefox UA reports `x86_64` vs pinned arm64 container.**
  Severity: LOW. All FF legs report UA `X11; Linux x86_64` (`c06-legs.json`
  `compare-firefox`/`drift-*-firefox`/`w1-grid-firefox`) while C01 pins FF141
  arm64 noble (aarch64 verified, no emulation). Likely a UA default string, but
  it conflicts on its face with the arch attestation. Routed: C owner one-line
  clarification.
- **F12 — DOC-invoice seal predates C06 app edits (344 vs 347).**
  Severity: LOW. C07 sealed 344/344 at `6fe2bc46`; join-revision server asserts
  are 347/347 via C06 positive legs — delta unexplained in C07 terms. Join
  coverage exists; final DOC-assert closeout is H14-owned (not IC2). Routed:
  H14 note. (evidence-index C07 row; `c06-report.md:129-135,159-168`)
- **F13 — F06 isolation deviation (SQLite file per run, no PG/MySQL mkdb).**
  Severity: LOW. Webhook Can side is SQLite-only by F05 design; documented
  deviation from the mkdb task letter, never a shared table. Routed: IC2
  disposition (accept or require PG/MySQL pair legs). (`f06-report.md:92-98`)
- **F14 — F04 literal operator-shell path deferred to H12 Leg 7.**
  Severity: LOW. W6.1 "executed operator-DDL recipe" evidenced via
  `applyLedgerDDL()` (same statements) + live CHECK-floor/verify probes per
  dialect; the literal §2 shell path (psql/mysql CLI) runs on UP25.
  Routed: H12 Leg 7. (`f04-operator-ddl-recipe.md:22-31,43-58`; `h12-staging.md:216-228`)
- **F15 — A08 agent legs unrun (0 attempts, no model access).**
  Severity: LOW. Records complete per A08's definition of done (deterministic
  legs + neutral/adverse + limitations); tokens/model/tokenizer unmeasured; no
  superiority claimed. IC2 aggregates deterministic legs only. Routed: H14 —
  story must state no measured agent advantage.
  (`x-r08-comparison-2026-09-26.md:32-37,147-181`; `registry.json`)
- **F16 — G04/G05 main-integration hashes unrecorded.**
  Severity: INFO. README/index cite worker hashes only; H10 consumed the same
  pins and reran rename 4/4 at HEAD. Cosmetic provenance gap. Routed: coordinator.
- **F17 — F03 cold-pool `withTransaction` burst defect still open (E-owned).**
  Severity: LOW. Concurrent burst on a cold pool fails `resource_state` before
  callbacks enter (PG+MySQL, timing-dependent); F03 legs warm pools first;
  generated apps will hit it without an E fix. F04 confirms untouched. Routed: E
  owner. (`f03-report.md:61-69`; `f04-report.md:69-71`)
- **F18 — F04 exercised no E04 budget path (E04 prereq unexercised).**
  Severity: LOW. Ledger backend uses native Bun.SQL directly; no E file touched.
  No W6 leg names ledger-under-budget evidence, but the E04 prereq contributed
  nothing. Routed: IC2 disposition (confirm nothing required). (execution README
  F04 row; `f04-report.md:58-60`)
- **F19 — B01/B02 vs B05 provisioning differs by design (bundle vs archive).**
  Severity: INFO. Both bun 1.4.2/go1.27.1; no version mismatch. Recorded for the
  cross-check. (`x-r06-1.md:3-7`; `b05-legs.json:6`)
- **F20 — A07 vs F06 Bun provisioning differs by design (binary vs archive).**
  Severity: INFO. Both pinned bun 1.4.2. Recorded. (`a07-report.md:7-10`;
  `f06-legs.json` `env`)
- **F21 — H10-NOTE-01 carried (E spec-prose sync, non-blocking).**
  Severity: INFO. `request-policy.md` S3/stream lines still read "conditioned …
  not wired"; normative artifacts correct; joined view republished in
  `h10-manifest.md:67-77`. Still E-routed. (H10-NOTE-02 resolved by coordinator.)
- **F22 — E09 full-suite 17 pre-existing environmental fails.**
  Severity: INFO. 1038 pass / 17 fail in `bun test runtime/test/` (16 mysql
  URL-shape + 1 Playwright module); no W5/prereq file fails; same env failures
  E04/E05 recorded. (`e09-w5.md:132-137`)

## 7. Pending input (single): H12 W6-deploy native legs

Pointer: `docs/syntax-taste/evidence/2026-09-26/h12/h12-staging.md`
(+ `jev/` findings; `distribution/h12/window-check.sh`;
record `docs/syntax-taste/execution-2026-09-26/h12-staging-record.md`).
Status: **BLOCKED-awaiting-UP25-grant**. Staging complete; zero native legs
executed; no machine probed; no emulation anywhere. The H12 W6-deploy leg was
deliberately NOT claimed, emulated, or worked around in this pre-assembly.

Exact UP25 ask (quoted):

From `distribution/x86-window.md:19-24` (H05):

1. Designate the machine: hostname/address + login user + which key or access
   path to use, and confirm it is Debian 13+ amd64/glibc.
2. Grant an exclusive window: single queue — no other jobs on the box during
   qualification; state start time and duration (or "on demand").

From `execution-2026-09-26/h12-staging-record.md:14-23` (H12 staging record,
item 3):

3. Reachability for the PG legs: either a `DATABASE_URL` to a live PG 17
   reachable from the UP25 host, or approval to natively install PG 17 on the
   box during the window (isolated database).

The minute access lands, H12 runs `window-check.sh` (READY gates everything),
then Legs 1–8 per the runbook, and hands W6-deploy evidence + lifecycle recipe
to H11/H13/H14. Live-probe policy: BLOCKED legs stay BLOCKED with cause, never
green; only native-x86 observations count.

## 8. Jev consultations

None. This turn is routine aggregation of delivered owner seals against explicit
contracts (acceptance-ownership table, experiment-branch table, H10 conditional
table); no difficult design decision arose, so per AGENTS.md no SystemOne
consultation was run and there are no requests/responses to file.

## 9. IC2 verdict

**None claimed.** IC2 stays OPEN behind H12 (single pending evidence input).
A future IC2 close additionally requires disposition of the BLOCKED/debt
findings above (F1–F8 at minimum: X-R15-3 scope return, L3 preparation return,
W6-AI grants+bound, B05 re-seal, D live legs, W2-negative and W1.5 and
RETURNING dispositions) — none of which this pre-assembly resolves or waives.
