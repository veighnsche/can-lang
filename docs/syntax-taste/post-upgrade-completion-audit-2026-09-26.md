# Post-upgrade completion audit (UP27)

26 September 2026 · auditor: session lane I · code candidate `08904291`
(+ UP23 verdict step and `TestHeavySlotCap` isolation, both
test/CI-only; qualified at `be95d009`).

## Review inputs

Selected contracts:

- [Dispositions](post-upgrade-dispositions-2026-09-24.md) — 11 Fix,
  9 Retain, 19 Defer.
- [Selected behavior](post-upgrade-selected-behavior-2026-09-24.md) —
  implementation contract for the 11 fixes.
- [Invoice acceptance](post-upgrade-invoice-acceptance-2026-09-24.md) —
  cross-target workload gates.
- [Reconciliation](post-upgrade-reconciliation-2026-09-24.md) —
  pre-fix baseline (status notes superseded per row).

Candidate source: this tree at the commit above; `git status` clean
except this audit's own documentation commit (docs-only, recorded
below). Pinned inputs: `distribution/target.json` (Bun 1.4.2
darwin-arm64), `distribution/target-linux-amd64.json`,
`tests/integration/browser/bun.lock` (Chromium 140.0.7339.186,
WebKit 26.0), `package.json` TS/tool pins, PostgreSQL 17.11.

Raw evidence (all archived under
`docs/syntax-taste/evidence/2026-09-26/`, digests in `SHA256SUMS`):

- `shards/shard0-units.log` — non-integration packages: exit 0,
  18 ok, 4 no-test-files, 0 failures.
- `shards/shard1-gate3.log` — `TestGate3`: exit 0, 8/8 PASS in
  929.7 s.
- `shards/shard2-contract.log` — `TestInvoiceContract*`: exit 0,
  9/9 PASS in 762.8 s.
- `shards/shard3-live.log` — invoice-live + gate4 + applications:
  exit 0, 13/13 PASS in 695.7 s.
- `shards/shard4-browser.log` — gate5 + browser: exit 0, 6/6 PASS
  in 478.9 s.
- `shards/shard5-rest.log` — remainder: exit 0, 65 PASS / 5 SKIP /
  0 FAIL in 599.7 s.
- `shards/shard1-gate3-killed-p4.log` — the superseded single-shape
  `-parallel 4` attempt, killed under swap exhaustion (18–20
  staged Bun workers, load 66–77, zero completions); kept as
  history for the shard1 diagnosis, not as candidate evidence.
- `up23-verdict.json` — UP23 guard verdict: 11/11 legs pass on
  Chromium 140.0.7339.186 and WebKit 26.0, bound to commit
  `be95d009180437d361682a6fd37a9b8c442cb86a`, consumed by the
  shards via `CAN_UP23_RESULTS`.
- `up24/` — 17 machine-readable gate/browser `report.json` files
  (every one `"passed": true`) plus screenshots, parity vectors,
  and the per-file inventory in its [README](evidence/2026-09-26/up24/README.md),
  captured via `CAN_BROWSER_EVIDENCE_DIR` during shards 2–4.

The six shards execute all 106 `tests/integration` tests plus
every unit package exactly once (verified: every `func Test*` in
`tests/integration/*_test.go` has a `--- PASS/SKIP` line in
exactly one live shard log), at `-parallel 2`, one shard at a
time — the shape this 16 GB host sustains. Exact commands in the
[clean-build recipe](post-upgrade-clean-build-2026-09-26.md#full-staged-qualification).

No Jev consultation was used for this audit; every row below names
source and executed tests.

## Fix verification

Each row: contract clause → shipped source → proving tests → staged
verdict. Full source/test links live in the [fix evidence](post-upgrade-fix-evidence-2026-09-26.md);
this audit re-verifies the verdict cells against the raw log.

| ID | Contract | Staged verdict (`evidence/2026-09-26/shards/`) |
| --- | --- | --- |
| B01 | quoted host words compile; real host ops and runtime closure reject | PASS `TestBrowserBuildTarget` 62.3 s, `TestGate5ServedMatrix` 476.2 s (shard4) |
| B02 | `[0,2,3]` Unicode / `[0,1,2,3]` plain starts | PASS `runtime/test/text.test.ts` via `TestCurrentBundledText` 7.0 s (shard5) |
| U01 | one locked package, handler-free, no mirrors | PASS `TestGate3ContractEdits` 5.1 s (shard1), `TestInvoiceGridPagePaired` 173.2 s (shard3) |
| U02 | explicit pool/origin, per-request actor, protected-op recheck | PASS `TestGate3ServerMatrix` 148.9 s, `TestGate3Lifecycle` 280.4 s, `TestGate3Concurrency` 233.4 s (shard1), `TestGate4FaultMatrix` 178.1 s (shard3) |
| U03 | direct routes, edit propagation, replay semantics | PASS `TestGate3ServerMatrix` 148.9 s, `TestGate3RouteRebuildLive` 263.9 s, `TestGate3ContractEdits` 5.1 s, `TestGate3AdapterMatrix` 271.7 s, `TestGate3ReplayExpiry` 266.8 s (shard1) |
| U04 | maintained bundle path, verified pairing, CSP | PASS `TestGate5ServedMatrix` 476.2 s (shard4), `TestInvoiceGridPagePaired` 173.2 s (shard3), `TestBrowserBuildTarget` 62.3 s (shard4) |
| U05 | sync cancel policy, ordinary events untouched | PASS gate5 conformance "cancel" legs inside `TestGate5ServedMatrix` 476.2 s (shard4); archived `conformance-chromium`/`conformance-webkit` reports 17/17 checks passed |
| S01 | real 503 swaps; guard rejects unsafe shapes | PASS `TestInvoiceFormLive` 187.7 s, `TestInvoiceHTMLFragments` 221.3 s (shard3), `TestInvoiceBrowserGuardDOM` (shard3), `TestGate3RendererFault` 241.6 s (shard1); UP23 guard verdict 11/11 legs pass (`up23-verdict.json`) |
| S02 | one sanitized report per failed callback | PASS `runtime/test/browser-dom.test.ts`, direct pinned-Bun run 26 Sept: 14 pass / 0 fail / 412 expects (not in the staged Go suite, so verified standalone; note the run must be repo-scoped — an unscoped `bun test` fuzzy-matches two stale copies under `out/`) |
| S03 | attached focus, durable notice, pending state | PASS gate5 grid legs inside `TestGate5ServedMatrix` 476.2 s (shard4); archived `grid-chromium`/`grid-webkit` reports 32/32 checks passed |
| P03 | symbolic public callees, atomic SCC, concrete emission | PASS `TestCurrentBundledGenericChain` 8.5 s (shard5); `compiler/internal/check` ok 258.8 s incl. `TestExportedGeneric*` (shard0) |

Contract-edit coverage (every shared-contract edit kind rebuilds
both targets or diagnoses): route (U03-P2), capture (U03-P3),
wire field (U03-P3), body mode (U03-N1), result leaf (U03-P3),
limit (U03-P3) — all in `TestGate3ContractEdits` /
`TestGate3RouteRebuildLive` above. No claimed edit kind lacks an
executed leg.

## Boundary verification

- Ownership: eligible structural equality retained (P21);
  owner-only construction/update/reads with explicit projection;
  generic codec/schema derivation reaching an owner record rejected
  (`compiler/internal/check/owner_record_test.go`, staged suite).
- Security: per-request session from server-controlled headers;
  protected-operation recheck inside the mutation transaction;
  exact-Origin CSRF before mutation; foreign/nonexistent share one
  nondisclosing 403; session by HttpOnly cookie only (no wire/hidden
  token); served CSP `script-src 'self'`, `connect-src 'self'`, no
  inline script, no `unsafe-eval`; browser closure free of server
  imports and secret bytes (gate3/gate4/gate5 legs above).
- Profile: browser capability gate rejects server-only reachability
  including through generic/callable edges (B01/U04 legs);
  `main(str[] args)` rejected for `--target browser`
  (`TestGate5GridStatic`); unknown targets fail closed.

## Generated artifacts

- `compiler/internal/catalogue/catalogue.json` — `cataloguegen
  --check` clean at the candidate (rerun 26 Sept, exit 0, no output).
- `can.lock.json` files (invoice, grid, shared contract) — paired
  builds verify locked shared-package instances; mismatch fails
  (U04 pairing legs).
- Browser manifests/asset digests — content-addressed; tamper and
  mismatch legs fail before publication (U04-N2).
- Same-input rebuild identity — no staged-suite leg rebuilds the
  same input twice, so identity is evidenced by the installed
  run: copy-based grid rebuild returns the identical build ID
  (`b02636607e2c`), and the paired server report pins its
  browser build (`27d76733132b`) — see the clean-build
  [verification note](post-upgrade-clean-build-2026-09-26.md#verification-note).
  `TestGate5ServedMatrix` separately asserts the served page
  carries exactly the report-selected paired entry (once) with a
  sealed diagnostic table (`gate5ServedPairing`). No stale
  generated artifact.
- `modcheck` ("modules OK: 133 maintained sources"), `gramcheck`
  ("grammar OK"), `gofmt -l` (no files), `go vet ./...` (no
  findings) — all clean on rerun 26 Sept at the candidate.

## Exclusion verification (19 deferred)

Each deferred proposal was checked against current source and current
docs: not implemented, not described as shipped.

| ID | Proposal | Current state |
| --- | --- | --- |
| P01 | bound-callable client projection | Not implemented. The shipped "client projection" (`compiler/internal/emit/action_bindings.go`) is frozen JSON fetch metadata with explicitly no handler/renderer/form binding — the selected U01/U03 split, not the deferred alternative. |
| P02 | runtime profile vs maintained shims | Resolved inside U04 for the fix (browser-specific runtime profile via the compiler-owned bundle stage); no grid-only shim enters a supported path. No standalone mechanism preference is claimed beyond the shipped path. |
| P04 | authored finite error-set parameters | Not implemented: no `emits<…>` parameter syntax exists (source grep); exact finite `emits` retained. |
| P05 | stack-safe dynamic iteration | Not implemented: no tail-call lowering in `compiler/internal/emit` (source grep); no portable numeric stack threshold is claimed. |
| P06 | early HTTP response with live race losers | Ownership retained: losers drain under the owner scope (`runtime/owner-core.ts` `drain`; `docs/implementation/shutdown.md`); no supervised-loser or abort change. |
| P07 | nonempty race helper / new empty completion | Retained native modes: first-completion empty race stays pending; no new API. |
| P08 | richer input event fields | Not implemented: snapshot stays kind/target-id/value/key (`runtime/platform/browser.ts`); U05 is the shipped minimum. |
| P09 | reload-persistent drafts | Not implemented: no storage API in the browser runtime (source grep); in-view draft only, documented. |
| P10 | browser history and WebSocket | Not implemented: WebSocket sessions stay server-only (`compiler/internal/browser/browser.go`); no history API. |
| P11 | reviewed host adapters vs catalogue additions | Closed catalogue retained (`compiler/internal/catalogue/catalogue.go`: applications cannot load another inventory or register host implementations). |
| P12 | field/keyed-row reuse or components | No component syntax (`ComponentDecl` absent from `compiler/internal/syntax`); named functions/typed nodes compose. |
| P13 | explicit `near` binding | Name-based immutable captures retained (`compiler/internal/check/callables.go`). |
| P14 | advisory local-elision rule | Narrow validity predicate retained (`compiler/internal/check/locals.go` `CheckLocalForwarding`). |
| P16 | multiline delimiters | Physical-line grammar retained (`CAN-LEX-CONTINUATION`, `compiler/internal/syntax/lexer.go`). |
| P17 | cleanup helper / `try/finally` lowering | Not implemented: no `finally` in Can syntax/emission; explicit cleanup/ownership retained. |
| P18 | bulk Map/Set APIs | Not implemented: no bulk constructors (source grep); immutable copy-on-point-update retained. |
| P19 | schema-aware SQL build tooling | Not implemented: statement/parameter/cardinality checking plus runtime row validation only (`compiler/internal/sql/`); descriptors never described as schema-verified. |
| P20 | mutation `RETURNING` | Still rejected (`RETURNING is not admitted`, `compiler/internal/sql/cardinality.go`); transaction plus SELECT retained. |
| P21 | owner-selected equality | Representation-based structural equality retained (`compiler/internal/types/inhabitation.go`). |

Doc sweep: no current dated (26 September) document describes any
row above as shipped; the T26 guide's deferred list points here as
the current boundary (§8 "Post-upgrade boundary note" links the
disposition ledger and this audit). Grep transcript: deferred IDs
P01/P02/P04–P14/P16–P21 across the 26 September docs, filtered for
shipped-sounding verbs and excluding explicit negatives
("not implemented", "deferred", "retained", "rejected", …),
returns exactly one line — this audit's own P02 row, which
disclaims any standalone mechanism beyond the shipped U04 path.
No dated doc claims a deferred row as shipped.

## Retention verification (9 retained)

P15 (`false`-before-`true` user decision, still enforced in
`completion_matches.go`), O01 (batch validation without rollback),
O02 (2–255 choice cardinality, `runtime/ai/questions.ts`), O03
(shipped fetch/judge wrappers only), O04 (named fixed-origin clients,
origin confinement in `runtime/transport/request.ts`), O05
(trusted-carrier webhook scope), B03/B04 (reconciliation doc fixes
stand), U06 (this release gate: every new claim above carries
source/test evidence and limits). Spot-checked against the staged
suite; no retained rule was weakened to obtain green.

## Candidate match

- `git status` at audit close: clean; this documentation set is
  the audit's own docs-only commit. `git show --stat HEAD` lists
  only dated-doc and evidence paths: the three UP26/UP27 docs,
  the reconciliation/T26/cli/generics touch-ups, the two example
  READMEs, and `evidence/2026-09-26/` (shard logs, UP23 verdict,
  UP24 reports, SHA256SUMS).
- Commits after `08904291` touch only tests, the CI verifier
  workflow and docs: `a546bb20` (UP23 verdict step:
  `tests/integration/up23_verdict_test.go`,
  `tests/integration/browser/invoice.mjs` partial leg,
  `gate5_frontend_test.go` count/occurrence update,
  `.github/workflows/verifier.yml`) and `be95d009`
  (`TestHeavySlotCap` ambient-env isolation in
  `tests/integration/harness_cache_test.go`). They are covered by
  the standalone `TestUP23WriteVerdict` execution (11/11 legs,
  archived verdict) plus `TestInvoiceBrowser` and
  `TestGate5ServedMatrix` in shard3/shard4 and
  `TestHeavySlotCap`/`TestHeavySlotsBounded` in shard5. No
  product-code change after `08904291`: `git diff 08904291 HEAD
  -- compiler/ runtime/ shared/ examples/` is empty, and the full
  `git diff 08904291 HEAD --stat` lists only the five
  test/CI files above (177 insertions, 4 deletions).
- Pinned asset inputs unchanged by the upgrade work: `git diff
  08904291 HEAD -- distribution/target.json
  distribution/target-linux-amd64.json
  tests/integration/browser/bun.lock package.json` is empty.

## Task completion

UP01–UP24 staged work complete; UP25 deferred to the x86 Linux
machine (never emulate x86 on this MacBook Air); UP26 complete with
this documentation set; UP27 complete with this audit. No required
task remains open.

Exact versions: Bun 1.4.2+`744846f84` (darwin/arm64, pinned
archive), Chromium 140.0.7339.186, WebKit 26.0 (pinned Playwright
browsers), PostgreSQL 17.11 on 127.0.0.1:5433, `go version
go1.27.1 darwin/arm64`, TypeScript 7.0.2 (pinned
`tscheck/node_modules/typescript`).

Residual limits and skips in the green run:

- `TestLinuxInstalledArtifactSmoke`, `TestLinuxPostgresRoundtrip` —
  deferred UP25 (x86 machine owns them).
- `TestCurrentMySQLPersistence`, `TestMySQLDifferential`,
  `TestCurrentS3Objects` — unoperated external services; never
  claimed as qualified.
- `L-redirect-webkit` — no same-origin 302 producible under WebKit
  interception; the engine-independent redirect branch is qualified
  on Chromium (recorded limitation, not a silent skip).
- UP25 installed server/browser evidence — missing by deferral;
  staged evidence does not claim it.

Final report links: [fix evidence](post-upgrade-fix-evidence-2026-09-26.md),
[clean-build recipe](post-upgrade-clean-build-2026-09-26.md),
[UP24 evidence](evidence/2026-09-26/up24/) (machine-readable reports).

## Verdict

Green. All six shards exit 0 with zero failures: 18 unit-package
oks, 8/8 gate3, 9/9 contract, 13/13 invoice-live/gate4, 6/6
gate5/browser, and 65 pass / 5 skip in the remainder — 106 of 106
integration tests executed, every fix row above re-verified
against its shard log, all 17 archived browser reports passed,
and the UP23 guard verdict 11/11. The only skips are the listed
residuals (deferred UP25 Linux legs, unoperated MySQL/S3
services, the `CAN_UP23_OUT`-gated writer row whose verdict file
is archived). No gate was narrowed to obtain green: the earlier
shard1 `-parallel 4` failures were diagnosed as host swap
exhaustion (preserved log), and the rerun at the sustained
`-parallel 2` shape passed every leg.
