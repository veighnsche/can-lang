# H10 IC1 interface manifest — 2026-09-26

Join revision: main `932924f7` (H10 validated first at `7c2dbc2a`, then
rebased-forward onto 9 main commits — C06 W1/evidence slices + tempcache
reclaim — and re-ran the full battery; all pins below re-verified).
Pinned Bun archive for H10 reruns: `/private/tmp/bun-darwin-aarch64.zip`
(sha256 `90987a3a…be1d12f`, matches `distribution/target.json`).
Local dev bundles were built from HEAD only (gitignored `dist/`, removed
after validation per AGENTS.md temp-cleanup rule, never committed).

## Published interface versions (C-A..C-I)

| IF | Owner slice(s) | Version / pin | Normative artifact(s) |
| --- | --- | --- | --- |
| C-A lowering | A04 `78668ec7` | proof predicate P09-M9 + hazard table | `compiler/internal/check/self_tail.go`, `compiler/internal/check/self_tail_coverage.md`; step detail `,"step:"+loopStep` in `compiler/internal/emit/regions.go:181,213` |
| C-A bulk | A05 `94eb71c5`+E-merge `e7982fa` | `collections::build_map/build_set`, catalogue 288→290 at seal | `runtime/collections/{map,set}.ts`, catalogue `collections` section |
| C-B result-data | B01 `c691a127`+`dbafd204`+`50dc8d3c` | convention R1–R9 + X-R06-1 verdict | `tests/failure-conventions/README.md`, `tests/failure-conventions/x-r06-1.md`, `retry*/` fixtures |
| C-B factory | B02 `c743cb77`+`7834c66f`+`50dc8d3c` | convention F1–F5 + X-R07-1 verdict | `tests/failure-conventions/x-r07-1.md`, `owner-*/` fixtures |
| C-C policy | E01 `2224e1b4`, E02 `2bf35a4c`, E04 `ea043250`, E05 `384a6080`, E06 `e3a4148c`, E07 `bef41a2d`, E08 `0688c1eb` | shared-budget policy + C-C vocabulary, O1 hedges, redaction hook, S3 destructive branch | `docs/implementation/request-policy.md`, `docs/implementation/e04-budget-contract.md`, `docs/implementation/request-failure-reporting.md`, `runtime/transport/{operation-budget,request-scope,deadline,request-report}.ts` |
| C-C S3 | E08 `0688c1eb` | `s3::discard_upload` replaces removed `s3::cancel_upload`; emits `[upload_closed, access_denied, service_error]`; between-awaits bound | catalogue `s3` section, `runtime/platform/s3.ts:205-241,420,736-739`, `docs/implementation/evidence/2026-09-26/e08-r15-remedy.md` |
| C-D native | C02 `ff18f134` (+`21adee9a`,`19a088b0`,`d6cdb438`,`3ceb315d`,`bfdc0f9c`) | 9 live ops + 6 snapshot fields + 3 record types, additive-only; catalogue types 102→105, ops 290→299 | catalogue `browser` section, `runtime/platform/browser.ts`, `docs/syntax-taste/evidence/2026-09-26/c02/c02-report.md` |
| C-D library | C04 `b7a3e214` (+`c5ab41dd`,`8c2b8c40`) | shared `grid-controls` + second app `invoice-compare` | `shared/grid-controls/`, `examples/invoice-compare/`, `examples/invoice-grid/` via `vendor/controls/` |
| C-E wire | A03 `dd3f3674` + C03 `9a2ea7b7` + E03 `db6d42b1` | GET-html→`document`, POST-html→`swap inner`, no per-case mixing; statuses 200–599 excl 204/205/304; `actionReject` 400\|405 | `compiler/internal/{syntax,ir,emit}` action slices, `compiler/internal/check/actions.go:370`, `runtime/platform/action-routes.ts:444,619` |
| C-F relational | F02 `d3e51ded` (probe) + F03 `bf83f0aa` | X-R10-1 admitted; RETURNING per-dialect (PG/SQLite `one`, MySQL §5 mapping); encodings; lookup-first; operator-DDL boundary | `docs/syntax-taste/evidence/2026-09-26/f03/{f03-returning-contract.md,f03-encoding-guide.md,f03-lookup-first-recipe.md,f03-report.md}` |
| C-G host | D01 `b837f710` + D02 `9bccd6ad`+`fb30a330` | Op A storage→T2, Op B clipboard→T2, Widget C chart→T3; protocol `d02.chart/1`, policy `2026-09-26.d02-chart`, manifest `2026-09-26.0-d02` | `tests/host-discrimination/x-r01-1.md:248-282`, `host/{adapters,companions,conformance}/`, `host/REVIEW-MANIFEST.json`, `host/d02-record.md` |
| C-G data | F01 `82750826`+`d3c414f3`+`cc9083e7`+`ee928ee1` | identity/epoch/destination policy + durable R14 ledger | `docs/implementation/outbound-policy.md`, `runtime/outbound/{identity,epoch,destination-policy,ledger,ledger-schema,file-ledger}.ts` |
| C-G companion | F05 `b9930b47` | carrier protocol v1 (`x-carrier-protocol: 1`, HMAC envelope), claim/lease 1000–600000ms, dead-letter rules | `examples/webhook/PROTOCOL.md`, `examples/webhook/companion/{protocol,worker,supervisor}.ts` |
| C-G AI | H07 `92f93263` | native budget guard + `ai_budget::within`; metered profiles disabled until bound qualifies; unqualified→reject | `runtime/ai/budget.ts` (`within` :561, registry :66–94), `examples/native-ai/README.md` |
| C-H deploy | H06 `1cf506e7` | generation `can-output-generation-v1` 64-hex buildID; `data-can-generation` + `can-generation` header; `generation_mismatch` schema v1 | `distribution/paired-deploy.md`, `compiler/internal/driver/output_handshake_test.go` |
| C-I authoring | A01 `19e45019`, A02 `1365d0a6` | false-first canonicalization, C8→`CAN-CHECK-UNNECESSARY-LOCAL` warning, `with` bindings + `CAN-CHECK-CAPTURE` | `compiler/internal/{check,syntax}/`, `compiler/internal/driver/` |
| C-I editor | G01 `94795946`, G02 `0d23fada`, G03 `20c45f48`, G04 `29715ce2`+`cb43cff6`+`5f3d8870`, G05 `fc85d837`+`ace91b09` | format→hover→refs→completion→rename incl R09 local-bindings (1A) | `compiler/lsp.go`, `compiler/lsp_g01_test.go`…`lsp_g05_test.go`, `docs/syntax-taste/evidence/2026-09-26/g04/g04-report.md`, `.../g05/g05-report.md` |

### Catalogue / generated-identity pins at join revision

- `compiler/internal/catalogue/catalogue.json`: schema 1, revision 1,
  target `bun-1.4.2-darwin-arm64-v1`, 37 packages / 105 types /
  299 operations / 110 errors.
- sha256 `87c05b44978ff20cea35c8e67a427c535a1aab20ab15e041a0cfb9006894b99f`,
  identical in `runtime/catalogue.ts:10` (`catalogueSHA256`).
- `make catalogue-check` (cataloguegen `--check`): exit 0, tree clean.

## Conditional interfaces: resolution + republication

Every conditioned branch is resolved and its winning branch published.
H10 republishes the joined view here; owners hold the normative artifacts.

| Branch | Outcome | Published in |
| --- | --- | --- |
| Q6 → A06 | INACTIVE (no needed W4 shape excluded; callable-fold = capture-analysis path, not trigger) | `docs/syntax-taste/evidence/2026-09-26/a07/a07-report.md` §Q6 gate; execution README A06 row |
| Q5 → B03 | INACTIVE (parity 17/17, +12/domain premium not syntax-removable) | `tests/failure-conventions/x-r06-1.md` §Q5 assessment + trip conditions |
| Q4 → B04 | INACTIVE (9 lines/factory; LD29 stays closed) | `tests/failure-conventions/x-r07-1.md` §Q4 assessment + trip conditions |
| X-R02-1 → C05 | INACTIVE (root-only rule stands) | `tests/browser-controls/x-r02-1.md:98-116` + trip conditions |
| X-R04-1 | NEGATIVE: cancel-absent — boundary returns at budget, op owned to settlement, honest unknown-write + supervisor escalation | `docs/implementation/request-policy.md:99-103` (§5), E04 runtime (`raceBoundary` rejects SQL cancel structurally) |
| X-R04-3 | POSITIVE: serve-side signal expires request scope; in-flight work returns unknown-write (cause `disconnect`) | `request-policy.md:104-106`, `runtime/transport/request-scope.ts` |
| X-R04-2 | O1 SUFFICIENT (S1 let-settle cancel-absent, S2 abort-loser cancel-present), O2 INACTIVE, zero write shapes qualified | `request-policy.md:114-123,167-171`, `docs/implementation/evidence/2026-09-26/e05-x-r04-2.md` |
| X-R15-1 | NEGATIVE: `s3::cancel_upload` REMOVED from catalogue; `s3::discard_upload` carries the destructive contract + unknown-outcome reporting | catalogue (0 `cancel_upload` hits outside negative-assertion tests; `discard_upload` present `catalogue.json`+`runtime/catalogue.ts`), `runtime/platform/s3.ts:205-241`, `e08-r15-remedy.md:30-71`, E1/W5-S1 absence legs |
| X-R15-3 | NEGATIVE: between-awaits bound only (never a hung await); W5 S3-deadline legs stay BLOCKED by design | `runtime/platform/s3.ts:420`, `e08-r15-remedy.md:76-93,142-143` |
| X-R10-1 | ADMITTED: locking reads expressible, no syntax change; RETURNING need qualified (keyless generated-identity only); MySQL rejects RETURNING (1064) → per-dialect mapping | `docs/syntax-taste/evidence/2026-09-26/f02/f02-report.md`, `.../f03/f03-returning-contract.md` (§3 cardinality, §5 MySQL mapping) |
| X-R14-1 | NO metering profile qualifies; H07 guard fails closed (unqualified→reject before send); no live-qualification claim | `docs/syntax-taste/evidence/2026-09-26/h08/h08-x-r14-1.md:3,11`, `runtime/ai/budget.ts:16-17`, `examples/native-ai/README.md:36-40` |
| D01 tier | A/B→T2, C→T3 (T3 excluded for B by measurement; T1 sketches stay live with selection conditions) | `tests/host-discrimination/x-r01-1.md:248-282`, `host/` deliverables + `REVIEW-MANIFEST.json` |

Supplemental contracts: R09 local-bindings (user 1A) integrated in G03/G05
(binding-identity refs + rename); R14 blocker-resolution supplement
integrated (F01 durable ledger → H07 native guard → F04 backend; E shared
context/transport).

### Known republication gap (non-blocking, E-owned)

H10-NOTE-01: `docs/implementation/request-policy.md` §7 line 107 still reads
"S3/stream behavior: conditioned on X-R15-* … still not wired", and the
§10-adjacent conditioned-claim ledger still lists S3/stream as unwired.
E07/E08 verdicts were never republished into the C-C spec prose. The
normative artifacts (catalogue, `s3.ts`, `e08-r15-remedy.md`, E09 W5 legs)
are correct and consistent; only the spec cross-reference is stale. This H10
record republishes the joined resolution (rows X-R15-1/X-R15-3 above).
Follow-up: E owner syncs the two spec lines + focused revalidation; no
consumer is blocked and no contract test depends on the stale lines.
