# Replace Manolea's tests

This is the single working task list. Progress means baseline files replaced with running equivalents. Commit each checked source increment immediately, including partial progress within a task; never wait for task completion, another lane or an aggregate qualification gate to commit.

## Starting point

- Can baseline: `708c0b1fd203d612029d402988fe93380e0dd535`. The approved apparatus cut is already committed. All 1,787 inventoried paths are absent. Do not rebuild or rerun the deleted programme.
- Manolea baseline: `9405dd995f8b5d2502901bbfaa6a69f077c6c982`. All 28 recorded test files still match their baseline hashes. Replacements complete: **0/28**; code obligations **0/25**; documents **0/3**.
- The endpoint is 16 host probes, one mail helper, three existing Can drivers, five fixture assets and three documents. The proposed destinations total **21 Can files and 3 documents**; five assets share one fixture producer.
- [Design and retained behavior](../native-can-tests-investigation-2026-10-02/diagnosis-and-replan.md#file-by-file-completion-and-cheapest-proof-all-28-goal-records); [28-file baseline](../native-can-tests-investigation-2026-10-02/manolea-baseline.json). These are supporting references, not additional status ledgers. The baseline remains immutable.

This round creates the task list; it does not execute the ports. Codex is the default executor. The replacement contract uses trusted, bounded native execution; it makes no hostile-isolation or hard RSS/tmp enforcement claim.

## Continuity with the earlier testing design

The task grouping is new because the old migration targeted Can's own 292-row suite, not these 28 Manolea records. The scenario authoring, observation and outcome semantics below carry forward the earlier design. The approved deletion replan explicitly supersedes qualified reference/owner promotion, hard hostile resource enforcement and their proof hierarchy; that is a contract change, not a claim that the new list implements the old architecture unchanged.

Historical sources were read directly from Git at `1c56acd5`, without restoring deleted files. For example, `git show 1c56acd5:docs/syntax-taste/native-can-test-authoring-2026-09-30.md`. The anchors below use filenames relative to `docs/syntax-taste/`. They record provenance, not extra execution gates: use the retained contracts here and each original Manolea source; do not restore the old programme or repeat this reconciliation before each task.

| Earlier requirement | Historical source | Current responsibility |
| --- | --- | --- |
| Ordinary Can functions, typed context/check references, required attached assertions; supplied boundaries exercise real control flow | `native-can-test-authoring-2026-09-30.md:13,194,233`; `native-can-test-backends-2026-09-30.md:121` | R01 and every port; execution rule 1 |
| Actual production subject, Can-owned scenario/verdict, independent native facts | `native-can-tests-completion-contract-2026-09-30.md:88,180`; `native-can-tests-investigation-2026-09-30.md:55,130` | A/B/C port tasks and rules 1/4; existing product APIs are reused |
| Honest selection, partial/incomplete results, sticky failures and bounded retries | `native-can-tests-completion-contract-2026-09-30.md:199`; `native-can-test-lifecycle-2026-09-30.md:50,465`; old P16/P17 | R01 defines ordinary execution/results once; F01 checks claimed scope |
| Exact HTTP/SQLite/native representations and truthful observation scope | `native-can-test-hard-cases-2026-09-30.md:112`; `native-can-test-capabilities-2026-09-30.md:50`; old K20/K22 | A01–A04, B01–B06, C03; rule 4 and B06 |
| Browser actions, input settlement, delivery/application witnesses and completed observation interval | `native-can-test-capabilities-2026-09-30.md:128,197,230`; `native-can-test-preparatory-results-2026-09-30.md:18`; old K09/K10/K14 | C02/C03, using the narrow installed-browser boundary |
| Concrete command/output bounds, shutdown/reap, owned storage and recovery, within-run build reuse | `native-can-test-lifecycle-2026-09-30.md:205,232`; `native-can-test-preparatory-checks-2026-09-30.md:51`; old P10/P11/P27 | R01 establishes finite values and normal cleanup; all lanes reuse them |
| Executed compiler/product/environment identity and source-aware diagnostics | `native-can-test-prerequisites-2026-09-30.md:54`; `native-can-tests-completion-contract-2026-09-30.md:199`; old P20/P27 | One compact family result; use retained `check --json` diagnostics when relevant, without creating a new diagnostic framework |
| Retire only resolved original behavior and switch its callers | `native-can-tests-completion-contract-2026-09-30.md:276`; old Z02/Z03 | All 28 ownership rows; per-file retirement and F01, without aggregate Z01 gating |
| Qualified R/N/S/A promotion, strict host isolation, sealed receipts/corrections, general native/descriptor/engine qualification | `native-can-test-architecture-2026-09-30.md:44,134`; `native-can-test-design-review-2026-09-30.md:142`; old P05/P13/P14/P22/P23 and Q/I gates | Superseded by the approved deletion replan and D00. Do not recreate these as prerequisites under new names |
| The 292-row/224-file Can-suite migration and its delegated/CI matrix | `native-can-tests-migration-ledger-2026-09-30.md:21`; old M01–M45/Z01/Z05 | Outside this 28-record endpoint. Existing unrelated compiler/product tests remain under their own contracts |

This one-time reconciliation covered all 13 primary historical testing documents (architecture, authoring, backends, capabilities, design review, hard cases, lifecycle, preparatory checks/results, prerequisites, completion contract, investigation and migration ledger), the main implementation plan, its four canonical task sets (foundation, capabilities, integration, migration), and dispatch/review/reference context. Task/checklist projections and preparatory drafts do not add new authority. Individual evidence receipts were not rerun or promoted. No new task IDs, status mirror, validator or review ladder results from this comparison.

## Execution rules

1. Run one real storage case before mass porting. Existing `canlc run`/`assert` and native runtime operations are the starting mechanisms. Scenarios/helpers remain ordinary Can functions with typed contexts/check references and their required attached `asserts`; no new testing DSL. Supplied offline boundary observations exercise real Can control flow/comparisons but never count as live coverage. Can owns setup, actions, selection, expectation/retry policy and verdicts; native JavaScript/Bun supplies missing mechanisms. Independently specify expected values; do not make them call the helper computing the actual result. No relocated Python/TypeScript verdict engine, scenario-bearing native callback, executable source string or whole-case success operation behind a Can wrapper. Keep fixtures once and use immutable updates for drift inputs rather than cloning whole records/manifest tables into every case.
2. One writer owns shared compiler/runtime/project connections. The three port lanes own disjoint test directories below. A lane requests a shared change by naming its blocked test file and missing operation. Add only that operation. Do not open a speculative capability backlog.
3. One heavy build/live command at a time on the laptop; reuse its build within the run, invalidating it when relevant source/compiler/runtime/options change. Give commands finite deadlines and output/observation caps, reserve cleanup time, stop writers before deleting their resources, and close/reap owned children, descriptors, servers and browser contexts. R01 assigns actual cooperative values in executable configuration, starting from the storage probe's 180 s execution/10 s reap limits; other cases retain their relevant original per-call limits. Register owned scratch cleanup before writing, reclaim on success/failure/handled interruption, and recover demonstrably abandoned owned work on later runs. Report unresolved resources; keep active/foreign work and shared caches intact. These are cooperative controls, not restored P13/P14 enforcement.
4. A port finishes when every original obligation runs, an independent actual observation supports its expectation, relevant negative/control behavior fails as intended, required checks cannot silently disappear, and cleanup/callers work. Preserve exact observation types/bytes/order where the original checks require them; independent SQLite reads bypass the production Can query/decoding path being checked. Distinguish expected subject rejection from observation/mechanism failure. Caught expectation mismatches stay failures; missing/truncated observations, unexpected timeout/death or unresolved cleanup prevent completion. Cleanup failure must not erase the original mismatch. Bound readiness polling/retries; a later green retry does not erase an earlier failure or unresolved effect, and partial attempts cannot be stitched into a completed case. One meaningful deliberate defect is shared per changed behavior boundary. Reuse one compact family command/result carrying selected cases/checks, actual compiler/runtime and product build/source identity, relevant environment, outcomes and cleanup. No per-row receipts, promotion ladder or repeated unchanged review.
5. Retire each original host file as soon as its own replacement and callers pass. Preserve unconverted consumers until they switch. An unavailable browser/client leg stays an explicit blocker; it is never counted as a pass. Documents and fixtures need their actual invocation/byte checks, not separate execution campaigns.
6. Run bounded affected checks after code changes. For authored `runtime/` or `tools/runtime/` TypeScript, run `bun run lint:fix:runtime`, `bun run format:runtime`, then `bun run check:runtime` and relevant tests. Regenerate catalogue mirrors officially if their source changes. No broad builds or performance campaign is added by this list.
7. Mark a task complete with its completion commit(s) and one short result/blocker. Update this list on completed outcomes, not on every internal step. Git commits contain the detail. No mirrored JSON tasks, resume journal, flip script or new validator.

R01 defines usable discovery/selection through that ordinary Can connection: listing performs no subject execution; invalid/empty/duplicate selections fail. A filtered passing run reports success for its selected subset, never full-suite success. All-skipped, early-ended and missing-selected-check runs are not complete. Declare only the goal cases/required checks in ordinary source, once; the 28-row ownership index below supplies the file obligation. These result semantics are implemented and checked with the first real case, not sent through a separate qualification pipeline.

## Order, ownership and remaining budget

`R01 first real case → three port lanes → per-file retirement → F01 final reconciliation`.

R01's shared-mechanism writer remains available to all lanes. After R01's runnable connection works, ready lane tasks may proceed; they need not wait for every storage edge case. Do not credit the storage records until R01 is fully complete. If two authored ports wait for one missing mechanism, stop producing more waiting ports and fix that mechanism.

| Work | Owner | Dependencies | Remaining wall allowance | API-equivalent ceiling |
| --- | --- | --- | ---: | ---: |
| R01 first real case and minimum connection | Shared writer | Current baseline | 3 h | $15 |
| A, B and C ports, including later R01 storage completion if needed | Three lanes + shared writer | Working R01 connection; local dependencies below | 16 h | $75 |
| F01 remaining callers/documents/retirement reconciliation | Coordinator + affected lane | Port results | 2 h | $15 |
| Remaining allocation | | | **21 h** | **$105** |

These are the remaining proposed phase allowances, not measured historical costs or a guaranteed ETA. Do not repeat the completed deletion phase or claim it consumed exactly its 3 h allowance. Each phase budgets at least 80% implementation and at most 20% total overhead. Reuse execution telemetry; at most 15 minutes may add the already-proposed fixed activity/usage record to the launcher. Compute overlapping wall intervals correctly, classify checks/review/evidence/waiting as overhead and leave unknown time uncredited. Dollar spend is unavailable without actual provider usage; do not infer it from subscription limits or hours. Use the design's cost method rather than researching prices again.

The launcher must stop dispatching extra proof/bookkeeping when a phase exceeds 20% overhead and flag the coordinator; required correctness checks remain completion gates. Ready implementation continues. At 3 h, the first-case checkpoint must show an actual product operation, a relevant bad-result refusal and cleanup, or identify the exact gap and revise the estimate. A time/cost breach calls for a concrete revised estimate, never silent coverage weakening or another accounting application. The whole remaining run's minimal controls retain the design's 3.45 h allowance, within 4.2 h remaining overhead; revise honestly if findings exceed it.

## Completed prerequisite

- [x] **D00 — remove the legacy apparatus.** Completion: `708c0b1f`. Its commit records paired emitter/checker/catalogue/CLI/runtime edits, bounded verification and two additional same-class strays. Historical reference bundles were retired separately. No new acceptance campaign is needed. Reported pre-existing emit environment/syntax-pin failures are baseline limitations, not tasks to rebuild the old test framework.

## First running replacement

- [ ] **R01 — replace the storage probe and integrate its Can driver.** Owner: shared writer, then lane B. Files: `tests/site_store/probe.py → probe.can`, existing `driver.can`, `README.md`; minimum project/launcher/native connection only as actually required. Current `source_root=src` excludes tests; provide a concrete runnable connection without persisting copied product trees or inventing a new command hierarchy. Carry the authoring/selection/result/bound/identity contracts above in this connection, without a separate framework task. Move Python-owned filesystem setup and independent judgments into Can. Run actual recover/load/save operations and preserve lock/stage name validation, directories/symlinks/FIFO/foreign bytes, stale/traversal refusal, atomic replacement and no escape/leak. First checkpoint: one actual storage operation, bad-result refusal and cleanup. Full completion retires the host probe, updates the invocation and covers all three owned baseline records. Commit checked increments immediately; checkpoint success alone does not complete R01.

## Lane A — auth, mail and services

Own `tests/auth_login`, `auth_signup`, `auth_recovery`, `services` and their live caller documentation. Shared HTTP/SQLite/process fixes go through the shared writer. A02/A03/A04 can proceed in parallel once their required mechanisms exist; A01 is a useful HTTP/SQL pattern, not an aggregate acceptance barrier.

- [ ] **A01 — replace login.** `auth_login/probe.py → probe.can`. After R01's runnable connection, add only needed native HTTP/process/independent SQLite mechanisms. Preserve generic privacy responses, verified-account/session/expiry rules, account/network budgets, concurrency, oversized body/capacity/DB failure and unchanged-state checks. Switch the invocation in `docs/auth-login-evidence.md`; retire this host file after its actual checks pass.
- [ ] **A02 — replace signup and its mail sender together.** `auth_signup/probe.py → probe.can`, `fake_mail.py → fake_mail.can`. Preserve real sender stdin/record shape, intentional sender failure, verifier-only token storage, GET nonmutation, verification/privacy and login transition, including unconfigured transport. The current probe rewrites the helper's shebang; replace that caller in the same increment. Retire both original files only after the Can sender is actually exercised.
- [ ] **A03 — replace recovery.** `auth_recovery/probe.py → probe.can`. Use the A02 sender when useful; retain independent mail/SQL facts for privacy, token expiry/reuse, GET rules, rollback, sender failure, budgets and session revocation. Do not wait for unrelated auth tasks to qualify this file.
- [ ] **A04 — replace both service probes.** `services/probe.py → probe.can`, `publication_probe.py → publication_probe.can`. Share one real service build/session fixture. Preserve two-owner CRUD, CSRF/escaping/uniqueness/IDs/drafts, publish/unpublish completeness and parent visibility, republish persistence and refusal leaving rows unchanged. Commit and retire each probe independently when ready.

## Lane B — assets, credentials, context and MCP

Own `tests/examples`, `site_store`, `site_agent` and their live caller documentation. Coordinate storage continuation with R01's shared writer. Use real schema/data and ordinary SQL/HTTP/files/image operations; the removed simulated DB observer does not return.

- [ ] **B01 — replace image upload.** `examples/e02_probe.py → e02_probe.can`. Preserve real multipart JPEG/PNG/WebP acceptance, MIME/path/SVG/HTML/animation/pixel/byte/count/aggregate rejection, independent DB/filesystem residue checks and retained existing links. Share auth/session setup without importing another probe's verdict.
- [ ] **B02 — replace image delivery.** `examples/e03_probe.py → e03_probe.can`. Reuse the asset fixture/build; preserve exact delivered bytes/headers, secrecy/escaping/IDs/publication/restart and corrupt/missing/MIME mismatch behavior with references intact. Dependency: actual delivery/session fixture; B01 completion is not a promotion gate.
- [ ] **B03 — replace image recovery.** `examples/e04_probe.py → e04_probe.can`. Execute the actual production recovery source. Preserve ownership and shared/draft/hidden/foreign/symlink/directory/FIFO behavior, interrupted upload cleanup, idempotency and failure deleting nothing. Keep the focused source/project connection bounded and disposable.
- [ ] **B04 — replace agent credential checks.** `site_agent/probe.py → probe.can`. Preserve actual credential lifecycle, exact verifier/expiry, owner/ID/secrecy, concurrent cap and expired-credential behavior through real HTTP/SQL facts.
- [ ] **B05 — replace context probe and integrate its Can driver.** `site_agent/context_probe.py → context_probe.can`, existing `context_driver.can`, `CONTEXT.md`. Preserve the production credential/context/bundle sources and required SQL descriptors, independent setup and two-maker projections/pagination/privacy/hidden/revoked/expired/bundle rules. Move external setup/expectations into Can and update the actual invocation/limits. Dependency: needed credential/SQL mechanism, not B04's completion status. Do not claim this context run proves MCP transport.
- [ ] **B06 — replace MCP probe and integrate its Can driver.** `site_agent/mcp_probe.py → mcp_probe.can`, existing `mcp_driver.can`. Reuse the B05 project/SQL connection and actual production `site_agent::mcp_route` with storage/context sources. Preserve exact integer/fractional JSON-RPC IDs and signed-zero behavior without lossy conversion, negotiation/auth/origin/body limits, privacy/pagination, conflict/quota/encoding/isolation/revocation. Qualify the real-client leg or leave it explicitly blocked. Update `docs/mcp-client.md`; retire the host probe only when its retained legs pass.

## Lane C — browser delivery, preview and fixtures

Own `tests/site_delivery`, `site_preview` and their live caller documentation. Use the installed Playwright boundary for only the typed actions/observations these cases need, with sequencing and expectations in Can; do not relocate former `page.evaluate` scenarios into native callbacks/source strings. Keep uploaded fixture JavaScript as subject content. Compare input settlement, response/redirect delivery and resulting application state separately; click fulfillment alone does not prove success. Complete relevant body/error callbacks before exact-count/absence claims; a quiet delay alone proves nothing. Share controlled TLS/browser setup with fresh test contexts/test-only credential behavior, without personal profiles or the known failed direct-launch configuration. Unexpected native credential UI stops browser work and remains a blocker; reclaim owned resources without changing the user's Keychain. No general browser testing or host-attestation platform is added.

- [ ] **C01 — produce the five fixture assets from Can.** Create `tests/site_delivery/fixtures.can`, materializing the exact five baseline payloads once into owned test storage. Check their baseline bytes once. Keep JavaScript/CSS/HTML behavior as uploaded-site subject content. Give C02/C03 a callable fixture producer and owned paths. Do not delete the tracked originals while unconverted delivery/preview consumers still read them. This is a supporting task; it replaces no baseline record until C04 switches those consumers and retires the assets.
- [ ] **C02 — replace delivery and Strict-redirect probes.** `site_delivery/proof.mjs → proof.can`, `strict-redirect.mjs → strict-redirect.can`. Dependency: runnable C01 fixture producer and required TLS/browser facts. Preserve real module/style/image/font/event/deep-link checks, CSP/cookie/host separation, expiry/revocation, leak positive/negative controls and clean URLs. Strict-cookie assertions must use actual clicks/POST/redirect and same-origin recovery, not only browser request APIs. Switch both fixture consumers to owned generated paths; commit/retire each host entry independently.
- [ ] **C03 — replace preview and approval probes.** `site_preview/probe.ts → probe.can`, `approval_probe.ts → approval_probe.can`, `README.md`. Dependency: C01 fixture producer and actual production preview service plus HTTP/SQLite/browser mechanisms. Preserve issuance/owner/CSRF/revision, exchange/cookie, restart/integrity/expiry/revocation, real POST handoff and browser facts; approval must preserve prior state on every wrong/corrupt/expired/revoked/GET/CSRF refusal and replace it on valid approval. Keep production handler semantics via the existing service entry; no helper owns the verdict. Switch generated fixture paths, update invocation/limits and `docs/s04-owner-approval-checklist.md`. External handler/environment exclusions remain visible.
- [ ] **C04 — retire the five original assets after their consumers switch.** Dependency: C01 producer verified and C02/C03 fixture reads converted; their unrelated remaining scenario checks do not gate this retirement. Remove `site_delivery/fixture/{index.html,pages/work.html,assets/site.css,assets/site.js,assets/value.js}` from tracked test source. Verify no current caller still requires those repository paths and generated bytes remain exact. Update `docs/site-delivery-proof.md`. Credit these five records here, once.

## Finish

- [ ] **F01 — reconcile and finish.** Owner: coordinator with affected lane. Require exactly one completed disposition for all 28 records below, current runnable family commands, no active retired-host caller, no hidden blocked leg, and cleaned owned resources. Three documents must describe the actual commands/limits. Stop when actual behavior, its relevant defect/refusal control and cleanup work; do not add final re-review, correction receipts or another gate. Commit remaining checked changes immediately and report any concrete unresolved blocker. Counts alone cannot establish behavior.

## Baseline ownership: exactly 28 records

All paths below are relative to `/Users/vince/Projects/manolea-2`. Completion is carried by the owning checkbox/commit above; this table is a fixed coverage index, not a second status projection.

| Original record | Replacement/disposition | Owner task |
| --- | --- | --- |
| `tests/auth_login/probe.py` | `tests/auth_login/probe.can` | A01 |
| `tests/auth_recovery/probe.py` | `tests/auth_recovery/probe.can` | A03 |
| `tests/auth_signup/fake_mail.py` | `tests/auth_signup/fake_mail.can` | A02 |
| `tests/auth_signup/probe.py` | `tests/auth_signup/probe.can` | A02 |
| `tests/examples/e02_probe.py` | `tests/examples/e02_probe.can` | B01 |
| `tests/examples/e03_probe.py` | `tests/examples/e03_probe.can` | B02 |
| `tests/examples/e04_probe.py` | `tests/examples/e04_probe.can` | B03 |
| `tests/services/probe.py` | `tests/services/probe.can` | A04 |
| `tests/services/publication_probe.py` | `tests/services/publication_probe.can` | A04 |
| `tests/site_agent/CONTEXT.md` | Update actual context command/limits | B05 |
| `tests/site_agent/context_driver.can` | Adapt existing Can driver | B05 |
| `tests/site_agent/context_probe.py` | `tests/site_agent/context_probe.can` | B05 |
| `tests/site_agent/mcp_driver.can` | Adapt existing Can driver | B06 |
| `tests/site_agent/mcp_probe.py` | `tests/site_agent/mcp_probe.can` | B06 |
| `tests/site_agent/probe.py` | `tests/site_agent/probe.can` | B04 |
| `tests/site_delivery/fixture/assets/site.css` | Exact bytes from `tests/site_delivery/fixtures.can` | C04 |
| `tests/site_delivery/fixture/assets/site.js` | Exact bytes from `tests/site_delivery/fixtures.can` | C04 |
| `tests/site_delivery/fixture/assets/value.js` | Exact bytes from `tests/site_delivery/fixtures.can` | C04 |
| `tests/site_delivery/fixture/index.html` | Exact bytes from `tests/site_delivery/fixtures.can` | C04 |
| `tests/site_delivery/fixture/pages/work.html` | Exact bytes from `tests/site_delivery/fixtures.can` | C04 |
| `tests/site_delivery/proof.mjs` | `tests/site_delivery/proof.can` | C02 |
| `tests/site_delivery/strict-redirect.mjs` | `tests/site_delivery/strict-redirect.can` | C02 |
| `tests/site_preview/README.md` | Update actual preview/approval command/limits | C03 |
| `tests/site_preview/approval_probe.ts` | `tests/site_preview/approval_probe.can` | C03 |
| `tests/site_preview/probe.ts` | `tests/site_preview/probe.can` | C03 |
| `tests/site_store/README.md` | Update actual storage command/limits | R01 |
| `tests/site_store/driver.can` | Adapt existing Can driver | R01 |
| `tests/site_store/probe.py` | `tests/site_store/probe.can` | R01 |
