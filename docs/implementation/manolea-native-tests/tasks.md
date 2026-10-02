# Replace Manolea's tests

This is the single working task list. Progress means baseline files replaced with running equivalents. Commit each checked source increment immediately, including partial progress within a task; never wait for task completion, another lane or an aggregate qualification gate to commit.

## Starting point

- Can baseline: `708c0b1fd203d612029d402988fe93380e0dd535`. The approved apparatus cut is already committed. All 1,787 inventoried paths are absent. Do not rebuild or rerun the deleted programme.
- Manolea baseline: `9405dd995f8b5d2502901bbfaa6a69f077c6c982`. All 28 recorded test files still match their baseline hashes. Replacements complete: **0/28**; code obligations **0/25**; documents **0/3**.
- The endpoint is 16 host probes, one mail helper, three existing Can drivers, five fixture assets and three documents. The proposed destinations total **21 Can files and 3 documents**; five assets share one fixture producer.
- [Design and retained behavior](../native-can-tests-investigation-2026-10-02/diagnosis-and-replan.md#file-by-file-completion-and-cheapest-proof-all-28-goal-records); [28-file baseline](../native-can-tests-investigation-2026-10-02/manolea-baseline.json). These are supporting references, not additional status ledgers. The baseline remains immutable.

This round creates the task list; it does not execute the ports. Codex is the default executor. The replacement contract uses trusted, bounded native execution; it makes no hostile-isolation or hard RSS/tmp enforcement claim.

## Execution rules

1. Run one real storage case before mass porting. Existing `canlc run`/`assert` and native runtime operations are the starting mechanisms. Can owns setup, actions and verdicts; native JavaScript/Bun supplies missing mechanisms. No relocated Python/TypeScript verdict engine behind a Can wrapper.
2. One writer owns shared compiler/runtime/project connections. The three port lanes own disjoint test directories below. A lane requests a shared change by naming its blocked test file and missing operation. Add only that operation. Do not open a speculative capability backlog.
3. One heavy build/live command at a time on the laptop; reuse its build within the run. Register owned scratch cleanup before writing, reclaim on success/failure/handled interruption, and report failures. Keep foreign work and shared caches intact.
4. A port finishes when every original obligation runs, an independent actual observation supports its expectation, relevant negative/control behavior fails as intended, required checks cannot silently disappear, and cleanup/callers work. One meaningful deliberate defect is shared per changed behavior boundary. Reuse one family command/result; no per-row receipts, promotion ladder or repeated unchanged review.
5. Retire each original host file as soon as its own replacement and callers pass. Preserve unconverted consumers until they switch. An unavailable browser/client leg stays an explicit blocker; it is never counted as a pass. Documents and fixtures need their actual invocation/byte checks, not separate execution campaigns.
6. Run bounded affected checks after code changes. For authored `runtime/` or `tools/runtime/` TypeScript, run `bun run lint:fix:runtime`, `bun run format:runtime`, then `bun run check:runtime` and relevant tests. Regenerate catalogue mirrors officially if their source changes. No broad builds or performance campaign is added by this list.
7. Mark a task complete with its completion commit(s) and one short result/blocker. Update this list on completed outcomes, not on every internal step. Git commits contain the detail. No mirrored JSON tasks, resume journal, flip script or new validator.

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

- [ ] **R01 — replace the storage probe and integrate its Can driver.** Owner: shared writer, then lane B. Files: `tests/site_store/probe.py → probe.can`, existing `driver.can`, `README.md`; minimum project/launcher/native connection only as actually required. Current `source_root=src` excludes tests; provide a concrete runnable connection without persisting copied product trees or inventing a new command hierarchy. Move Python-owned filesystem setup and independent judgments into Can. Run actual recover/load/save operations and preserve lock/stage name validation, directories/symlinks/FIFO/foreign bytes, stale/traversal refusal, atomic replacement and no escape/leak. First checkpoint: one actual storage operation, bad-result refusal and cleanup. Full completion retires the host probe, updates the invocation and covers all three owned baseline records. Commit checked increments immediately; checkpoint success alone does not complete R01.

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
- [ ] **B06 — replace MCP probe and integrate its Can driver.** `site_agent/mcp_probe.py → mcp_probe.can`, existing `mcp_driver.can`. Reuse the B05 project/SQL connection and actual production `site_agent::mcp_route` with storage/context sources. Preserve numeric JSON-RPC IDs, negotiation/auth/origin/body limits, privacy/pagination, conflict/quota/encoding/isolation/revocation. Qualify the real-client leg or leave it explicitly blocked. Update `docs/mcp-client.md`; retire the host probe only when its retained legs pass.

## Lane C — browser delivery, preview and fixtures

Own `tests/site_delivery`, `site_preview` and their live caller documentation. Use the installed Playwright boundary for actual facts, with Can-owned expectations. Share controlled TLS/browser setup; do not build a general browser testing platform.

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
