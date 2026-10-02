# Delete the legacy testing apparatus; replace Manolea tests

**Proposed plan only. Nothing has been deleted or changed in implementation.** This revision merges the new deletion brief into the existing investigation instead of creating a second diagnosis. It supersedes the first round's incremental keep/batch/defer recommendations. The factual diagnosis below retains its frozen evidence window.

The user’s endpoint is the **28 tracked records in `manolea-2/tests`: 25 code files and 3 documents**. Goal-oriented work either replaces one of those records faithfully or implements a native mechanism that a named record actually cannot run without. New framework features need that concrete edge. Existing compiler/product functionality outside the migration-specific cut remains the system that compiles and serves the subjects; this is not authorization to purge unrelated repository work.

## Deletion proposal: the burden is on keeping

The proposed closed cut is **1,787 tracked files / 36,155,161 logical bytes**. The exact path set is [deletion-inventory.tsv](deletion-inventory.tsv); [deletion-summary.json](deletion-summary.json) records groups and pinned baseline **`1c56acd5e29cbaa1bb0bf707b8cacd6e634639d6`**. The TSV is one mechanical removal list, not another task ledger: consume it with file tools, do not load its 1,787 rows into model prompts. No per-file prose, evidence gate or review is attached to it.

| Layer to remove entirely | Files | Bytes |
| --- | ---: | ---: |
| Staged migration corpus | 553 | 17,202,896 |
| Coverage campaign source | 49 | 1,478,481 |
| Remaining old suite source | 40 | 677,243 |
| Old suite configs/bootstrap/subjects/examples | 49 | 175,508 |
| Old plan evidence | 475 | 6,667,202 |
| Old plan tools | 5 | 63,660 |
| Old plan ledger/checklists/prompts/validator + sibling Markdown | 26 | 3,161,145 |
| Superseded native-test design ledgers/contracts | 13 | 347,613 |
| Superseded preparation evidence/tools | 223 | 2,760,122 |
| `tools/native-test-owner` | 108 | 1,025,411 |
| `tools/native-test-reference` | 22 | 78,028 |
| `tools/native-test-bootstrap` | 3 | 34,993 |
| `schemas/native-test` | 60 | 86,129 |
| `runtime/test-support` | 31 | 497,804 |
| `tools/runtime/test-services` | 51 | 1,371,358 |
| Legacy runtime adapter tests | 16 | 204,402 |
| Exclusive compiler driver/emitter/checker/catalogue tests | 63 | 323,166 |

These disjoint groups remove all of `tests/native-can`, the 143-task execution application, its Python proof tools, the simulated observer services and their dedicated adapters. There are **zero migration-specific survivors inside this removal set**. The evidence group contains 471 JSON files, three extensionless files and one text file; the brief's 373-JSON figure misses nested artifacts. The 49 coverage files are 48 slice sources plus `machinery.can`.

The 63 exclusive compiler paths are exact in the TSV: eight `driver/test_{stage,reuse,generation_lease,launch}` source/test files; twelve `emit/*_operations.go` source/test pairs for `test,native_values,http_peer,descriptor,descriptor_lease,c,late,db,db_transaction,db_poison,db_deadline,store`; thirteen exclusive checker files for those test APIs; six DB/store checker source/tests; and twelve corresponding catalogue-contract test files. Shared source files are edited as described below, never removed wholesale. General native AI declarations and normal SQL/HTTP/assertion behavior survive.

Ignored historical reference bundles are a separate proposed cleanup: two named roots under `run/acceptance` contain **742 files / 211,806,914 bytes**. Verify ownership and no running consumer before removing them. They are untracked binaries, so Git cannot restore their exact contents; retiring them accepts rebuilding for a historical rerun. Preserve the source/manifest in Git. The six small `.muse/worktrees` files, shared dependencies/caches, performance artifacts and unrelated ignored output have unproven ownership and stay outside this cut.

Keep exactly three historical specimens through **local Git object references**, without new archive copies:

- Evidence: `git show 1c56acd5:docs/implementation/native-can-tests-plan-2026-09-30/evidence/M01-CORE-002.json`.
- Slice: `git show 1c56acd5:tests/native-can/migration/m01-assertion-and-check-evidence/core-002-report-shape.can`.
- Ledger: `git show 1c56acd5:docs/implementation/native-can-tests-plan-2026-09-30/coverage-map.json`.

Those objects preserve the tax for inspection. No remote availability or prior push is assumed. No separate archive, retention service or specimen-copy campaign is needed.

## Resolve every live dependency in the same cut

This table covers crossing reads/imports, implicit discovery and removed-symbol callers. Internal dependencies disappear with their closed groups; none is a reason to preserve the full apparatus.

| Live/reference boundary | Paired removal or edit |
| --- | --- |
| [bootstrap/reference/accept.go:42](/Users/vince/Projects/can-lang/tests/native-can/bootstrap/reference/accept.go:42) pins `P04-manifest.json`; [line 180](/Users/vince/Projects/can-lang/tests/native-can/bootstrap/reference/accept.go:180) dynamically reads it; `accept_test.go:383,394` exercises it | Delete the complete 27-file bootstrap package together with the plan/evidence. No external code imports were found. Do not transplant the old seed trust contract into the new runner. If keeping bootstrap is later justified, re-home only the 96,034-byte manifest and change the constant in the same commit; that is an explicit amendment. |
| Bootstrap seed resolver and package-local joined fixture reads; reference-tool `select.go`, `refresh.go`, `build.go` and `testdata/rcore/observe.py` | Delete their entire consumer packages. The reference tool has no external imports/callers; its normal `distbuild` dependency stays as product build machinery. Ignored seed cleanup follows retirement of these consumers. |
| Old-plan `validate_plan.py` and five `tools/*.py` files | Delete all six tools with their JSON/source inputs and outputs. They dynamically read named evidence, source hashes and basename-selected migration files; their dependencies are not just harmless Markdown links. No external configured caller was found. |
| Generic project graph recursively reads `source_root=src`; 49 sources form package `migration` | Delete all `tests/native-can` together. No source outside the campaign subtree uses package `migration`. The 553 staged files lie outside the source root and are read by the removed bespoke gates. |
| [driver/test_launch.go:16](/Users/vince/Projects/can-lang/compiler/internal/driver/test_launch.go:16), its test and `test_reuse_test.go` import owner admission; reuse is also used by staging tests | Delete all eight dedicated stage/reuse/launch/lease files with the owner. Keep normal driver build/run/output/cleanup code. |
| Native-can f1/f2 descriptor packages import owner process/journal/codec and each other | Delete f1/f2 with the rest of native-can. Their compiler-source checks create no external caller. |
| Owner packages import each other: acceptance/admission/host/journal/process/cleanup/recovery/service/workspace/dispatch | Delete the whole owner closure. Admission's external driver imports are resolved above. Runtime mirror references to Go files are comments. |
| [emit/program_state.go:73](/Users/vince/Projects/can-lang/compiler/internal/emit/program_state.go:73), `:124`, `:493`, `:532` declare/initialize/import all legacy adapters | Remove all twelve dedicated declaration/init/import/value-import contributions with their operation files and runtime ports. Ordinary emitted programs must no longer require `runtime/test-support`. |
| [emit/runtime_bindings.go:145](/Users/vince/Projects/can-lang/compiler/internal/emit/runtime_bindings.go:145) collects their operation bindings | Remove those twelve binding contributions in the same increment. Keep general platform bindings. |
| [check/http.go:70](/Users/vince/Projects/can-lang/compiler/internal/check/http.go:70) invokes removed opaque-scope predicates; [check/completions.go:707](/Users/vince/Projects/can-lang/compiler/internal/check/completions.go:707) invokes removed native/C/late/DB/ws test validators | Remove only legacy predicates/dispatch branches. Preserve production HTTP, SQL, browser, native AI and ordinary scope checking. |
| Catalogue source entries and generated mirrors; `runtime/modules.json:858–1037` indexes 31 ports | Prune the dedicated `NT-*` API definitions/unused test types/errors and module entries. Run the official catalogue generator for `generated.go`, `runtime/catalogue.ts`, `std/catalogue/README.md`, `std/catalogue/errors.json`. Never hand-edit generated mirrors. Update obsolete task-qualification handling/comments in catalogue source/integration tests. |
| [compiler/main.go:196](/Users/vince/Projects/can-lang/compiler/main.go:196), `runTest` and CLI test functions preserve old `--reference`/`--owner-dir`/P23-refusal contract | Remove the obsolete command branch and its exclusive CLI tests/records; retain normal `run`, `assert`, `check`, build and other CLI tests. A thin new launcher is allowed only if a named Manolea case cannot use normal execution. No compatibility aliases. |
| Makefile, verifier workflow and baseline script discover Go packages through `go test ./...` | Whole-package removal needs no directory-specific repair. A broad run is still deferred; the executor uses bounded relevant package checks. |
| Two catalogue comments name the plan; retained historical docs may link to it | Comments have no execution dependency; remove/update stale references without recreating a live historical gate. Historical explanation can cite pinned Git objects. |

A whole-tree tracked code/config scan outside the deletion set found only the two catalogue comments and module-map paths as literal apparatus references. Removed-symbol analysis additionally found the two emitter and two checker shared files above. Shared catalogue data/generated files and the CLI are explicitly paired. This is static dependency evidence; successful post-cut compilation remains an executor acceptance check, not a result claimed by this planning round.

## Deletion mechanics and remaining tree

The earlier investigation is **this session**, committed in `d7498c26`, `1518b989`, `1c56acd5`. The new brief's uncommitted/HEAD warning is stale. At the new-round preflight the checkout was clean; the relevant sibling chats were idle, and a sanitized process snapshot found no matching Can/Muse executor. Snapshots are not locks. Before each destructive increment, recheck peers/processes and compare selected source blobs with the approved pinned inventory. Preserve unexpected work and stop that increment on drift. Never reset, clean or overwrite a peer's files. Reuse a suitable checkout/worktree rather than accumulating disposable copies.

Execute only after this plan and its contract choices are approved:

1. Remove top-level old plan/evidence/gates/docs, staged campaigns and **all native-can consumers including bootstrap**, together. This resolves the P04 read in the same coherent commit. Keep normal product/compiler tests.
2. Remove the dedicated compiler adapter/checker/catalogue closure and runtime test ports **with the shared edits and regenerated mirrors**, as one locally checkable increment. Remove old CLI source/tests. Run bounded affected compiler checks and required runtime scripts; commit immediately.
3. Remove owner/reference/bootstrap tool/schema closure and its exclusive driver machinery with any remaining imports in the same increment; check affected packages and commit. Do not postpone a compile fix to a later commit. Steps 2–3 may combine if their compile dependency makes splitting incoherent.
4. Retire the two owned ignored reference bundles only after their consumers are gone and ownership/running-use checks pass. Remove task-owned scratch on success, failure and interruption; leave foreign/shared storage alone.
5. Port and retire original **Manolea** host files by real behavior family. Keep each old host entry until its Can scenario, external observations, relevant controls and callers actually work. Commit each checked source increment immediately, independently of family qualification.

For baseline tracked source, the exact post-cut set is **`git ls-tree -r --name-only 1c56acd5` minus the TSV paths**, plus the explicitly paired edits/generated mirrors. No member of the TSV is kept. The current investigation/proposal is retained as one decision record and one mechanical removal list; it has no execution authority in product code. The exact new Manolea file dispositions are in [manolea-baseline.json](manolea-baseline.json): sixteen probes become corresponding `.can` entries, fake mail becomes `.can`, three Can drivers are adapted/registered, five browser assets become one Can fixture producer, and three documents are updated. This yields **21 Can files plus 3 documents**, with existing project configuration reused or minimally connected. Optional new native/launcher files require a named missing operation and are fixed at the first live-case checkpoint, rather than invented here.

What earns its place: normal Can parse/check/emit/assert and runtime owner/completion support compile actual subjects; native SQL observes auth/services/context/approval rows; files/bytes/image cover storage/uploads/recovery/assets; process and server facilities start/stop product subjects/mail/TLS fixtures; fetch/form/cookie/codec/crypto cover HTTP/auth/MCP observations; a narrow Playwright boundary supplies real browser facts. The old DB observer is simulated and earns no place as an independent SQLite oracle. Ordinary `check/native.go`, `native_bodies.go`, `runtime/owner.ts` and `owner-core.ts` remain actual product mechanisms, distinct from the deleted test owner. Existing production modules remain under their own product contracts.

After the cut, the system is an ordinary Can compiler/runtime plus Can scenario files that drive the real Manolea app, inspect independent HTTP/SQLite/filesystem/browser facts, assert expected behavior and clean their owned resources. Native JavaScript/Bun supplies mechanisms; Can owns the scenarios and verdicts. One file manifest tracks the 28 original obligations and shared family results. There is no reference promotion ladder, 143-task ledger, generated constant-campaign project or per-row receipt application. Buildability is proved by bounded affected checks in the executing increment; it has not been asserted from a deletion plan alone.

## Explicit risks of dropping old controls

| Removed contract/control | Accepted risk; retained minimum |
| --- | --- |
| Full sealed R/N/C/S/A framework, P23/Q promotion and correction receipts | Lose independent toolchain/owner promotion and that framework's hostile-scope identity guarantees. Preserve ordinary build/source identity and real independent product facts. This requires approval of a replacement execution contract. |
| Strict P13/P14 Mac resource enforcement as a prerequisite | Trusted cooperative execution has no hard adversarial RSS/tmp backstop. Keep capped commands/output, one heavy job, deadlines, owned scratch and shutdown/recovery. Never label it strict. Enforced hostile isolation remains a separate option with unknown cost. |
| Per-row vectors, seed-parity clones and type sabotage | Lose redundant generator self-consistency checks. Keep typechecking changed packages, independent actual expectations and a relevant deliberate defect per changed behavior boundary. |
| Z01 gate/manifest/partition/re-review hierarchy | Lose protection for an obsolete 292-row programme. One 28-file source/disposition reconciliation protects the actual target; missing required checks fail in the new real runner. |
| Detailed evidence/ledger/status projections/checkpoint prose | Lose convenient narrative reconstruction. Git, one family result, source hashes and a compact current checkpoint preserve usable provenance. |
| Per-slice independent review and automatic re-review | A shared pattern defect may affect several ports before correction. Inspect the actual native boundary/semantic expectation once; rerun only affected checks after a concrete defect. No new review apparatus. |
| Old static migration coverage and exclusive API tests | The added test-framework APIs disappear; their promised compatibility and qualification are abandoned. Original production tests outside this cut remain. Manolea behaviors cannot be retired without live equivalents. |
| Historical full binary seeds | Instant rerun of the old sealed framework is lost. Git preserves source/manifest; a historical run requires a deliberate rebuild. |

## New budget, stopping rules and decisions

The critical path is **dependency-safe cut → one actual Manolea case using normal execution → narrow missing native mechanisms → three parallel Can port lanes → shared family qualification/caller switch → per-file retirement**. The old P23 implementation and its entire qualification framework are **options**, not assumed prerequisites. A Can wrapper around relocated host verdict code is cheaper initially but fails the goal.

Proposed **24-hour wall-time envelope**, excluding explicitly unresolved external environments; this replaces the first round's 40-hour allocation. It is a target and a cost ceiling for escalation, not a proved ETA. Three source lanes cover auth/services/mail; storage/images/context/MCP; browser/delivery/preview. One writer owns shared compiler/runtime changes, and one heavy live/build command runs at a time.

| Phase | Wall h | Implementation h | All overhead h | Agent-attention allowance | Standard API-equivalent cost ceiling |
| --- | ---: | ---: | ---: | ---: | ---: |
| Closed dependency-safe removal and bounded checks | 3 | 2.4 | 0.6 | 5 h | $15 |
| First real case + minimum runner/native gap | 3 | 2.4 | 0.6 | 5 h | $15 |
| Parallel family ports and actual integration | 16 | 12.8 | 3.2 | 40 h | $75 |
| Remaining fixes, callers, documents and retirement | 2 | 1.6 | 0.4 | 4 h | $15 |
| Total | **24** | **19.2 / 80%** | **4.8 / 20%** | **54 h** | **$120** |

Cost numbers are proposed ceilings, not historical invoices or expected consumption. Price reported uncached/cached/output tokens separately; output includes billed reasoning. As checked against [official OpenAI pricing](https://developers.openai.com/api/docs/pricing), Sol Standard short-context rates are $2/$0.10/$10 per million; Fast rates are $4/$0.20/$20, so a fixed dollar ceiling buys fewer tokens. Include billed cache-write charges when present. Codex subscription limits are shared account usage and cannot establish this task's dollar spend. If per-run token/charge telemetry is unavailable, report monetary cost as unavailable, enforce elapsed/attention ceilings, and do not invent a conversion from hours.

**Mechanical tripwires, priced within those 4.8 overhead hours:**

- Add at most **0.25 h** of work to the executor launcher for one append-only activity/usage CSV and a compact budget line after each event. Record phase, fixed activity class, monotonic start/end, active implementation interval, command waiting, and provider token/charge totals where available. Supervisor assigns job classes; plan/evidence/review writes and verification calls are overhead. Unknown time is not credited as implementation. Compute wall time from interval unions, not summed parallel agent-hours. Category accuracy is still a limitation; automatic arithmetic does not make hidden model thinking observable.
- On cumulative overhead **above 20% of a phase**, the launcher stops dispatch of more proof/bookkeeping jobs and flags the coordinator to cut/batch the cost. Ready authorized implementation continues. A phase is never labelled 80% compliant with missing activity telemetry. No per-row time receipts or new scheduler are allowed.
- On **two authored goal-file replacements waiting for a missing shared execution mechanism**, stop producing more waiting ports and reassign capacity to that mechanism. Do not recreate the 46-task acceptance queue.
- At the **three-hour first-live-case checkpoint**, require a real product result, one relevant bad-result refusal and cleanup, or name the exact missing mechanism and revise the critical path. This is an escalation threshold, not a manufactured short cap on completing authorized work.
- Reject new apparatus work without a named one-of-28 file and a missing required behavior. One shared family receipt replaces per-file transcripts; no new proof layer is dispatched because a previous proof can hypothetically be forged.
- A wall/attention/cost ceiling breach triggers a concrete revised estimate and explanation before scope expansion. Continue bounded authorized ready work; external coverage or isolation changes require the user decision below. Do not spend the budget on a new accounting application.

Minimal assurance prices are **allowances for the whole remaining run**, not observed old hours: file/source/required-check reconciliation 0.3 h; meaningful wrong-result and missing-check controls 0.6 h; real independent semantic/native-boundary inspection 0.5 h; timeout/death/cleanup controls 0.5 h; bounded affected compile/runtime checks 0.5 h; relevant browser fact/hand-off controls 0.8 h. With accounting 0.25 h, these allocate **3.45 h** of the 4.8 h overhead ceiling; **1.35 h** remains for findings, commit/coordination and necessary evidence. Controls overlap existing scenario runs. Stop after the actual boundary works and its relevant control fails as intended; repeated unchanged review, mutation or full-check sweeps have no allocation. Complex findings can exceed these allowances: revise the estimate honestly instead of weakening silently or pretending the ceiling predicts completion.

The control catches in the retained diagnosis justify the minimal forms: M10 IDs/source drift → simple source/case correspondence; F1 omission → actual file/required-check completeness; I11 wrong expectation → executed independent semantic vector; owner pipe/refusal defects → real shutdown/refusal checks. No evidence supports old **hours per defect**; those figures remain unknown.

| Open decision | Recommendation | Option cost and risk |
| --- | --- | --- |
| Aggressive closed cut | Approve the enumerated removal plus paired repairs, with no renamed legacy subsystem | 3 h phase allowance; keeping the whole framework has unknown additional serial effort and still leaves every Manolea probe |
| Execution contract | Trusted bounded native runner, with Can owning verdicts | 3 h first-case allowance plus needed mechanisms in ports; old strict P23/reference/owner completion has unknown host unblock and exceeds this narrow budget |
| Resource guarantee | Explicit cooperative profile for trusted scenarios, never a strict claim | Keep load/storage/cleanup limits inside normal runs; enforcing hostile hard memory/tmp containment or moving to a qualified host has unknown additional time |
| Five browser asset files | Materialize exact bytes from one Can fixture source | 0.5–1 h within port allowance; leaving five static assets costs near zero but does not satisfy literal all-code-file replacement |
| Unavailable external browser/client/environment leg | Preserve as an explicit blocker and continue independent ports | Provisioning wait unknown; omitting that leg changes coverage and needs a named decision |
| Cost/time telemetry | Tiny launcher record with fixed tripwires | 0.25 h allowance; if unavailable, exact 80% and dollar compliance cannot be claimed |

**Proposed short standing rules for the user to confirm:** “Replace Manolea's tests.” “Keep only what those tests need.” “Commit each checked change immediately.” “Spend at least 80% of implementation-run time implementing.” “Stop adding proof once the behavior, defect control and cleanup work.” “Never count an unrun test as passed.” These are proposals, not rules inferred from checkpoints or old agent prose. The current brief is a planning request; approval of deletion/execution choices is a later step. No message was sent to another executor or chat.

## Historical wall-time diagnosis, reused without new measurement campaigns

The original figures below remain evidence for the frozen September 30–October 2 window. Narrower row-like logs now identify **164 worker log lifetimes: median 8.42 minutes, sum 23.73 agent-hours**. That measures the whole worker interval, including coding, proof and waiting; **proof-tax minutes per row and historical control-hours remain unidentifiable**. There is no honest conversion of a deadline into elapsed time. The first-round top-five interval proxy and its large unassigned portion are retained. The demonstrated failure is 292-row-scaled source/proof production with zero requested file replacements, regardless of an unproved exact “half” percentage.

## 1. The finish line moved to the wrong repository

The existing ledger describes 146 Go test functions, 58 Bun declarations and related Can harness/history/caller obligations: [ledger:21](/Users/vince/Projects/can-lang/docs/syntax-taste/native-can-tests-migration-ledger-2026-09-30.md:21). Its snapshot is explicitly a Can commit at [ledger:43](/Users/vince/Projects/can-lang/docs/syntax-taste/native-can-tests-migration-ledger-2026-09-30.md:43).

The first coverage source is `tests/integration/applications_test.go`: [coverage-map:10](/Users/vince/Projects/can-lang/docs/implementation/native-can-tests-plan-2026-09-30/coverage-map.json:10). All **224/224 retirement paths exist in Can; 0/224 exist in Manolea**. No Manolea reference was found in these maps or the native migration tree. The 292-row decomposition therefore cannot establish completion of the user's endpoint.

Manolea has **28 tracked test files: 25 code files totaling 192,222 bytes and 3 Markdown documents**. These comprise 16 host probe entrypoints, one Python mail helper, three existing Can drivers, and five uploaded-site browser fixtures. Its tests are unchanged since September 30. [manolea-baseline.json](manolea-baseline.json) records the exact paths, baseline repository HEAD and SHA-256 of every file.

The three Can drivers already exercise production functionality. Their surrounding host probes still own setup and judgments. This is the useful starting point for the requested replacement.

## 2. What the large assertion files actually contain

| Source area | Can files | Bytes | Lines |
| --- | ---: | ---: | ---: |
| Active `tests/native-can/src` | 89 | 2,155,724 | 14,971 |
| Staged `tests/native-can/migration` | 553 | 17,202,896 | 142,955 |

The full native test tree contains 691 files / 19,534,128 bytes. Its [project configuration](/Users/vince/Projects/can-lang/tests/native-can/can.project.json:1) sets `source_root` to `src`. The staged 17.2 MB migration tree is excluded from that project graph; attributing its entire size to each active compiler check would be wrong.

Concrete amplification:

- **Manifest clones:** 49 active migration manifest arrays occur in full as both expected assertion output and returned body. One duplicate copy totals **479,560 characters**, approximately 22.25% of active source. In [m30a.can:21](/Users/vince/Projects/can-lang/tests/native-can/src/coverage/migration/m30a.can:21) and [line 25](/Users/vince/Projects/can-lang/tests/native-can/src/coverage/migration/m30a.can:25), the same constructor expression is **25,295 characters** long.
- **Quadratic mutation tables:** [core-015-run-shape.can:237](/Users/vince/Projects/can-lang/tests/native-can/migration/m10-distribution-verified-builds-a/core-015-run-shape.can:237) is **213,158 bytes / 587 lines**. Its observation has 61 fields, a default and 61 single-field drift cases. Every variant spells the entire record; the complete table appears again in the implementation at [line 304](/Users/vince/Projects/can-lang/tests/native-can/migration/m10-distribution-verified-builds-a/core-015-run-shape.can:304). The `observe` function occupies about **152,612 characters / 71.6%** of the file.
- **Repeated local harnesses:** 502 staged `observe` functions occupy about **7.87 MB**. `selector`, `require_bool`, `require_int` and `require_text` appear hundreds of times and together occupy about **1.21 MB**.
- **Repeated expected expressions:** exact repeated `=> ok …` right-hand sides within individual files total **532,845 extra characters** in active source and **4,774,440** in staged source. These counts overlap the preceding categories; do not add them together or treat every duplicate as safely removable.

The giant payloads are predominantly constructor expressions containing repeated strings and records. The audit did not find giant assertion failure messages: [runner.ts:149](/Users/vince/Projects/can-lang/runtime/assert/runner.ts:149) reports a short reason, and [comparison:88](/Users/vince/Projects/can-lang/runtime/assert/runner.ts:88) already uses strict `Bun.deepEquals` behind necessary identity guards. String equality lowers to native `===`.

There is a second, independent compiler amplifier. Each assertion module imports every authored function/native binding and declares the full model type graph: [program_entry.go:71](/Users/vince/Projects/can-lang/compiler/internal/emit/program_entry.go:71). Authored modules likewise emit all model aliases: [program_modules.go:85](/Users/vince/Projects/can-lang/compiler/internal/emit/program_modules.go:85). Generated output bytes and timing were not measured. Narrowing these dependencies is a plausible bounded compiler improvement; a rewrite or promised speedup is unsupported.

For equivalent duplication introduced into genuine goal-file replacements, reuse the existing immutable record `with` operation for drift inputs, keep independently specified affected-field/verdict checks, and keep provenance once. Do not make expected answers call the same helper that computes the actual result. Exact protocol, serialization and malformed-input bytes remain fixtures. A canonical digest can replace a manifest clone if its independently pinned representation is specified; it loses immediate field-level mismatch detail. New assertion-template syntax is unnecessary for this bounded remedy and could merely hide emitted expansion. The listed historical corpus is proposed for deletion; this observation does not allocate work to compact it first.

## 3. The “ports” still require real scenario implementation

[compiler/main.go:270](/Users/vince/Projects/can-lang/compiler/main.go:270) describes dispatch that never stages or spawns. [Line 363](/Users/vince/Projects/can-lang/compiler/main.go:363) refuses execution: `test execution is gated by P23: selection validated, live R/N runner unimplemented`.

[Suite main:5](/Users/vince/Projects/can-lang/tests/native-can/src/main.can:5) is a descriptor renderer. [Line 48](/Users/vince/Projects/can-lang/tests/native-can/src/main.can:48) explicitly says no subject runs. All 553 staged packages import only `text` (502 files) or nothing (51), and expose only pure/local failure effects. They do not perform live browser, server or database observations.

For example, [LIFE-001:205](/Users/vince/Projects/can-lang/tests/native-can/migration/m38-gate3-route-and-server-matrix/life-001-base-semantics.can:205) manufactures strings from the original harness; [its body:299](/Users/vince/Projects/can-lang/tests/native-can/migration/m38-gate3-route-and-server-matrix/life-001-base-semantics.can:299) compares them. Such tables can document expectations or test a reducer. They do not preserve the original subject behavior by themselves. **Completing P23 will still leave live scenario bodies to implement.**

[Z01 evidence](/Users/vince/Projects/can-lang/docs/implementation/native-can-tests-plan-2026-09-30/evidence/Z01.json:1) records 171 retained, 51 decision, 12 blocked and 58 missing rows, 49 static manifests, and no qualification jobs. These classifications sum to 292; a source campaign for every row does not mean 292 runnable replacements.

## 4. What can be established about the last 48 hours

[activity-summary.json](activity-summary.json) freezes the window **September 30 03:54:14 to October 2 03:54:14 Europe/Brussels**. It contains **497 commits**, rather than reproducing the brief's earlier 505-count window.

- `feat` 268 (53.9%), `test` 57 (11.5%), `fix` 21 (4.2%), `refactor` 2 (0.4%): **70.0% implementation-labelled commits**.
- `docs` 101 (20.3%), `chore` 47 (9.5%), other 1. Total added lines: **524,314**.
- Tests account for **31.0%** of additions; compiler/runtime **15.7%**, generated catalogue mirrors **10.6%**. Treating catalogue churn as product implementation inflates that combined number to 26.3%.
- Tools account for **13.8%**, other paths 12.6%, plan ledger/docs **11.1%**, plan evidence **4.9%**, plan tools 0.3%. The path categorization is retained in the summary.
- Median adjacent commit gap is **108 seconds**, with **208 gaps under one minute**. Frequent commits already happened. The execution prompt requires immediate small commits at [line 54](/Users/vince/Projects/can-lang/docs/implementation/native-can-tests-plan-2026-09-30/muse-implementation-prompt.txt:54), but workers were told to hand uncommitted work to an integrator.

The planning baseline was captured September 30 at 17:19 Brussels; the final pre-investigation commit was October 2 at 03:50. The rolling window also includes earlier language/editor work. Treating all 48 hours as one uninterrupted migration implementation run is unsupported.

### Estimated activity shares: deliberately limited evidence

The following **five largest buckets** use an explicit proxy: assign an adjacent commit gap of at most 30 minutes to the ending commit's activity; leave longer gaps unassigned. This assigns **26.75 hours** of the 47.59-hour commit span. **20.85 hours remain unassigned.** Mixed work and parallel workers make this an uncertain estimate, not a time audit.

| Activity bucket | Assigned interval proxy | Share of assigned intervals |
| --- | ---: | ---: |
| Static migration/coverage campaigns, including fixture preparation and their checks | 10.14 h | 37.9% |
| Compiler, runtime and native capability changes | 4.96 h | 18.5% |
| Other language/editor/documentation work in the window | 4.37 h | 16.4% |
| Host owner and harness engineering | 3.23 h | 12.1% |
| Can controller/protocol/capability source | 2.07 h | 7.8% |

Ledger/evidence/checkpoint commits form the remaining **1.98 h / 7.4%** of assigned intervals. Their actual preparation cost can be buried in other buckets or long gaps. No credible method turns this into “only 40–50% coding,” “50% gates” or exact review hours.

A separate check found 170 surviving project worker logs. Their filesystem birth-to-last-write lifetimes sum to **24.45 agent-hours**, with **11.35 wall-hours** in the union and a median **8.42 minutes** per log. These intervals include reasoning, source generation, checks, prose and waiting; incomplete logs and filesystem copies can distort them. They corroborate sustained parallel work, without proving its internal time split. No private session material was copied.

**The 80% rule is presently unauditable.** Neither commit labels, added lines, command deadlines nor file timestamps establish it. The executor's 40–50% estimate remains an unverified estimate. The causal failures above are demonstrable without inventing that percentage.

## 5. The structural cost scales with rows

The plan contains **505 files / 9.87 MB**. Evidence contributes **472 files / 6.67 MB**; **234 row receipts alone contribute 5.33 MB**. These file counts include nested consultation evidence.

There are **3,915 command records**, including **3,168 in row receipts**: about **13.5 recorded commands per receipt**. They are not unique execution counts: one M32 live check is copied into 27 receipts; a Z01 batch check is repeated 46 times. Exact repeated JSON strings over 50 bytes account for approximately **971 KB excess**, including **572 KB repeated result prose**.

The related worker prompt requires roughly six assurance families for each row: byte pins/vocabulary, parse, mutation-table audit, isolated staged typecheck, deliberately mistyped-copy typecheck, and a live-tree check; it additionally requires evidence, environment probes and cleanup narration. The CORE-046 receipt spells this out at [evidence:8](/Users/vince/Projects/can-lang/docs/implementation/native-can-tests-plan-2026-09-30/evidence/M10-CORE-046.json:8). The row count therefore determines setup, dispatch, proof prose and reconciliation cost even when one reusable generator creates the source.

Repeated metadata is another multiplier: master task cards mirror roughly 476 KB of source cards, the migration checklist is 299 KB, and resume history is 118 KB / 1,142 lines. Surviving `/tmp/muse*` prompts/logs comprise **339 entries / 1.30 MB of regular files**. They are earlier-task artifacts with unverified ownership; this audit did not delete them. No new temporary directory or execution workspace was allocated.

Measured package durations in 49 result records are usually seconds: K24 Go 0.399s/race 1.240s; P21 compiler 4.1s; K07 repeat 0.6s/race 4.5s; P18/P19 driver runs 18.622s/22.6s. See [K24:76](/Users/vince/Projects/can-lang/docs/implementation/native-can-tests-plan-2026-09-30/evidence/K24.json:76), [P21:29](/Users/vince/Projects/can-lang/docs/implementation/native-can-tests-plan-2026-09-30/evidence/P21.json:29) and [K07:42](/Users/vince/Projects/can-lang/docs/implementation/native-can-tests-plan-2026-09-30/evidence/K07.json:42). Preparation, expanded code generation and assurance orchestration are supported suspects. The available data does not establish hours of CPU test waits.

## 6. Acceptance controls: what actually caught defects

There are no per-control elapsed totals. The “cost” column states observable repetition or individual timings; aggregate hours are unknown. Lack of a documented catch does not prove a control useless.

| Control | Observed defect | Observed cost; proposed minimum |
| --- | --- | --- |
| Case/check ID reconciliation | CORE-015 evidence used wrong `dist/sidecar-*` IDs; `a406fc79` corrected them | 49 Z01 campaign receipts; retain one batch manifest/source reconciliation |
| Independent completeness anchor + mutation review | Dropped row passed old gate, M5 rc=0; manifest partition now rejects missing CORE-002 | 48 manifest stems; retain deterministic omission/duplicate check and one representative mutation |
| Source hash/correction invalidation | HISTORY-091..098 pilot hashes stale after `0e669cc7`; gate fails on 30+ lines | Retain hashes per changed source and targeted requalification |
| Standalone staged typecheck | All 27 M33 slices had exported-private type failures before repair | Batch changed packages in one staging/build session; preserve isolated diagnostics |
| Independent semantic vectors | I11 expected false for `direction_known("hold")`; static checks missed it, I12 inspection caught it, `726bc5a7` fixed it | Execute focused vectors once per changed behavior; do not substitute syntax mutation for semantics |
| Real owner lifecycle/race controls | P14 exposed P10 pipe early-close and refusal-denial bugs; P13 found overflow and unreconciled-temp defects | Individual bounded runs; preserve timeout/death/release/leak controls for actual mechanisms |
| Per-row malformed-type “bite” | Demonstrates checker inclusion; no additional behavioral defect catch documented | Hundreds of scratch/check/count rituals; retain per generator/type-boundary change, batch elsewhere |
| Repeated byte/vocabulary/dead-seed audits | Some generation, citation and vocabulary errors; clones remain self-consistency checks | Keep deterministic generator validation and drift checks; eliminate manual repetition per unchanged template |
| Repeated environment/global-check prose and checkpoint arithmetic | No unique subject-semantic catch established; copied arithmetic itself required corrections | One shared command receipt and compact current checkpoint |

The F1/F2 and I11 findings are documented at [resume:1084](/Users/vince/Projects/can-lang/docs/implementation/native-can-tests-plan-2026-09-30/muse-resume.md:1084) and [I11 values:21](/Users/vince/Projects/can-lang/tests/native-can/src/capabilities/i11/values.can:21). The pause checkpoint's list of “uncommitted” paths is stale: initial HEAD already committed them. Its next-action list remains useful only after reconciling current files.

## 7. Real engineering and the completion bottleneck

Live recomputation: **143 unique tasks = 47 complete / 46 active / 40 planned / 10 blocked**. The task sources agree. Summary fields disagree: [tasks.json:9964](/Users/vince/Projects/can-lang/docs/implementation/native-can-tests-plan-2026-09-30/tasks.json:9964) retains older counts, while [validation.json:13](/Users/vince/Projects/can-lang/docs/implementation/native-can-tests-plan-2026-09-30/validation.json:13) is one promotion behind.

Under the present contract the narrow serial chain is:

`P13 strict host enforcement → P14 actual independently built N owner → P23 live runner → affected qualification → semantic M acceptance → Z01 aggregate acceptance → Z02 retirement → Z03/Z04 callers/bootstrap → Z05 matrix → Z06 review`.

- P13 genuinely lacks a kernel memory backstop and strict envelope-bound temporary storage: [profile.go:51](/Users/vince/Projects/can-lang/tools/native-test-owner/host/profile.go:51). Ledger admission cannot claim those enforcement properties.
- P14's ACK helper is not the actual owner: [P14 evidence:118](/Users/vince/Projects/can-lang/docs/implementation/native-can-tests-plan-2026-09-30/evidence/P14.json:118).
- P23 is actual missing execution, not an administrative signature. It gates 196 rows directly plus 71 conditionally if retained; **267/292 rows potentially depend on it**. Twelve I parents are active and unpromoted, with P14 required.
- Browser mechanisms gate 38 rows. The live longest residual unit chain is `K07→K08→K09→K10→K11→I06→QB0→QB1base→QB2→M21→Z05→Z06`, 12 unfinished vertices. Host/P23 dependencies still join it.
- [lane-plan:132](/Users/vince/Projects/can-lang/docs/implementation/native-can-tests-plan-2026-09-30/lane-plan.md:132) deliberately preserves the aggregate Z01→Z02 barrier. This delays per-file retirement even when a file's own proof could be complete. The 14-node/63-hour/422.25-hour planning estimates are stale and hypothetical.

Process costs can be reduced. Real memory/storage enforcement, independently observed browser behavior, reliable native adapters and a live runner require engineering. Completing the unrelated Can-suite programme is an additional product scope decision.


## File-by-file completion and cheapest proof: all 28 goal records

Each executable obligation requires Can-owned setup/action/expectations, real observations, required-check completeness, bounded execution/cleanup and current callers. One relevant deliberate defect is shared per changed behavior boundary; documents and fixture assets do not each get a new campaign. Retire a host entry/helper only when its actual behavior is replaced. The three already-Can drivers are integrated/adapted. The five browser fixture assets are materialized as exact fixture bytes if “all code files” literally permits only Can source. Their JavaScript semantics must remain unchanged as uploaded-site subjects.

Paths below are relative to `/Users/vince/Projects/manolea-2/tests`; the linked original is the source of the obligation. This table is a compact guide, not permission to drop unnamed checks in those files.

| Baseline file | Retained behavior and disposition | Cheapest proof |
| --- | --- | --- |
| [auth_login/probe.py](/Users/vince/Projects/manolea-2/tests/auth_login/probe.py:65) | Replace: private generic login responses, sessions/expiry, account/network budgets, concurrent attempts, body/capacity/DB failure; independent SQL facts | AUTH: HTTP/session/SQL facts |
| [auth_recovery/probe.py](/Users/vince/Projects/manolea-2/tests/auth_recovery/probe.py:72) | Replace: privacy, verifier-only storage, GET nonmutation, expiry/reuse, rollback, session revocation, sender failure and budgets | AUTH: HTTP/SQL/mail/rollback facts |
| [auth_signup/probe.py](/Users/vince/Projects/manolea-2/tests/auth_signup/probe.py:69) | Replace: private account/token/mail behavior, verification/GET rules, login transition, sender/unconfigured transport failure | AUTH: HTTP/SQL/mail facts |
| [auth_signup/fake_mail.py](/Users/vince/Projects/manolea-2/tests/auth_signup/fake_mail.py:8) | Replace: bounded stdin sender subject, exact mail shape/recording, intentional failure | AUTH: same sender fixture and one failure |
| [examples/e02_probe.py](/Users/vince/Projects/manolea-2/tests/examples/e02_probe.py:208) | Replace: real multipart JPEG/PNG/WebP; MIME/path/SVG/HTML/animation/pixel/byte/count/aggregate rejection, no residual DB/files, existing links | ASSET: multipart/DB/files facts |
| [examples/e03_probe.py](/Users/vince/Projects/manolea-2/tests/examples/e03_probe.py:189) | Replace: persistent exact image delivery/headers, secrecy/escaping/IDs/publication/restart, corrupt/missing/MIME mismatch with references preserved | ASSET: delivery/restart/DB/files facts |
| [examples/e04_probe.py](/Users/vince/Projects/manolea-2/tests/examples/e04_probe.py:193) | Replace: recovery ownership, shared/draft/hidden/foreign/symlink/directory/FIFO retention, interrupted-upload cleanup, idempotency, failure deletes nothing | ASSET: recovery/kinds/idempotency facts |
| [services/probe.py](/Users/vince/Projects/manolea-2/tests/services/probe.py:84) | Replace: two-owner create/edit, CSRF, escaping, uniqueness, drafts, ID handling, cross-owner refusal and unchanged rows | SERVICE: HTTP/two-owner SQL facts |
| [services/publication_probe.py](/Users/vince/Projects/manolea-2/tests/services/publication_probe.py:116) | Replace: publish/unpublish ownership/completeness/CSRF, parent visibility and republish persistence | SERVICE: same service build, HTTP/SQL facts |
| [site_agent/probe.py](/Users/vince/Projects/manolea-2/tests/site_agent/probe.py:154) | Replace: credential lifecycle, exact verifier/expiry, secrecy/ownership/IDs, concurrent cap and expired credentials | AGENT: HTTP/verifier/SQL facts |
| [site_agent/context_probe.py](/Users/vince/Projects/manolea-2/tests/site_agent/context_probe.py:104) | Replace: SQLite setup, production projections, two-maker pagination, privacy, hidden/revoked/expired contexts and bundle rules | AGENT: actual projections + independent SQL |
| [site_agent/context_driver.can](/Users/vince/Projects/manolea-2/tests/site_agent/context_driver.can:92) | Adapt/register actual production sequence and incorporate external setup/expectations | AGENT: shared context execution |
| [site_agent/mcp_probe.py](/Users/vince/Projects/manolea-2/tests/site_agent/mcp_probe.py:224) | Replace: exact JSON-RPC numeric IDs, negotiation/auth/origin/body caps, privacy/pagination, draft conflict/quota/encoding/isolation/revocation; real-client leg explicitly qualified or blocked | AGENT: raw JSON-RPC/HTTP/SQL facts |
| [site_agent/mcp_driver.can](/Users/vince/Projects/manolea-2/tests/site_agent/mcp_driver.can:29) | Adapt/register production MCP fixture; preserve handler behavior without test-specific reimplementation | AGENT: shared production-handler execution |
| [site_store/probe.py](/Users/vince/Projects/manolea-2/tests/site_store/probe.py:81) | Replace: real recovery/replacement, stale/traversal refusal, foreign kinds, no escaping writes/leaks | STORE: actual operation + file-kind facts |
| [site_store/driver.can](/Users/vince/Projects/manolea-2/tests/site_store/driver.can:117) | Adapt/register storage exercise with Can-owned preparation and independent facts | STORE: same operation execution |
| [site_delivery/proof.mjs](/Users/vince/Projects/manolea-2/tests/site_delivery/proof.mjs:98) | Replace: resources/deep links, cookies/CSP/independent hosts, expiry/revocation, bearer-referrer leak and clean-URL defense, positive and negative controls | BROWSER: one controlled real-browser family |
| [site_delivery/strict-redirect.mjs](/Users/vince/Projects/manolea-2/tests/site_delivery/strict-redirect.mjs:61) | Replace: browser request versus actual click/POST, redirect CSP, Strict-cookie withholding and same-origin recovery | BROWSER: same TLS/browser fixture, real clicks |
| [site_preview/probe.ts](/Users/vince/Projects/manolea-2/tests/site_preview/probe.ts:170) | Replace: production issuance/delivery, owner/CSRF/revision refusal, exchange/cookie, revision/expiry/revocation/restart/integrity, actual POST handoff and browser facts | BROWSER: production preview + HTTP/SQL/browser |
| [site_preview/approval_probe.ts](/Users/vince/Projects/manolea-2/tests/site_preview/approval_probe.ts:125) | Replace: exact owner/session/revision approval; GET/CSRF and wrong/corrupt/expired/revoked refusal preserves prior state; valid replacement | BROWSER: same service build, HTTP/SQL approval |
| [site_delivery/fixture/index.html](/Users/vince/Projects/manolea-2/tests/site_delivery/fixture/index.html) | Preserve exact module/style/resource/contact subject bytes; materialize from Can fixture source | FIXTURE: shared five-file hash/materialization check |
| [site_delivery/fixture/pages/work.html](/Users/vince/Projects/manolea-2/tests/site_delivery/fixture/pages/work.html) | Preserve deep link and relative resources as emitted fixture bytes | FIXTURE: same hash check + browser deep link |
| [site_delivery/fixture/assets/site.css](/Users/vince/Projects/manolea-2/tests/site_delivery/fixture/assets/site.css) | Preserve style/font subject bytes | FIXTURE: same hash check + computed styles |
| [site_delivery/fixture/assets/site.js](/Users/vince/Projects/manolea-2/tests/site_delivery/fixture/assets/site.js:1) | Preserve actual uploaded JavaScript behavior as fixture bytes | FIXTURE: same hash check + real event/module facts |
| [site_delivery/fixture/assets/value.js](/Users/vince/Projects/manolea-2/tests/site_delivery/fixture/assets/value.js:1) | Preserve separate module import/increment subject bytes | FIXTURE: same hash check + real module facts |
| [site_agent/CONTEXT.md](/Users/vince/Projects/manolea-2/tests/site_agent/CONTEXT.md) | Update current context invocation and limits; keep historical facts | Link/command check against the AGENT family run; no separate execution |
| [site_store/README.md](/Users/vince/Projects/manolea-2/tests/site_store/README.md) | Update storage invocation, ownership assumptions and limits | Link/command check against the STORE family run; no separate execution |
| [site_preview/README.md](/Users/vince/Projects/manolea-2/tests/site_preview/README.md) | Update preview/approval invocation and qualified environment | Link/command check against the BROWSER family run; no separate execution |

One machine-readable execution manifest should add to this baseline: disposition, replacement case/required-check IDs, retained environment/facet obligations, source and subject hashes, a shared family run ID, per-case result/exception, deliberate-defect result, cleanup result, current blocker and retirement commit. Save each command result once. Final reconciliation requires exactly one disposition per baseline file, every retained behavior mapped to an actual execution, no hidden blocked leg counted as pass, and no active caller still invoking a retired host probe.

Active caller updates include [auth-login evidence:29](/Users/vince/Projects/manolea-2/docs/auth-login-evidence.md:29), [MCP client:84](/Users/vince/Projects/manolea-2/docs/mcp-client.md:84), [approval checklist:68](/Users/vince/Projects/manolea-2/docs/s04-owner-approval-checklist.md:68), [site-delivery proof:20](/Users/vince/Projects/manolea-2/docs/site-delivery-proof.md:20) and the three test Markdown records. Historical evidence stays provenance. Production preview should use its [existing service entry](/Users/vince/Projects/manolea-2/services/site-delivery/main.ts:9), with generic native fixture routing rather than a helper that owns the verdict.


## Planning verification and consultation

This round used three bounded read-only investigations for inventory, dependencies and minimum goal-linked capabilities. It reused the earlier diagnosis rather than repeating broad activity analysis. Exact TSV paths are unique/tracked at the pinned source HEAD; group bytes are logical file sizes. Shared-file symbol and path references were reconciled. No post-deletion build, test or performance result is claimed, and no temporary directory was allocated.

The repository's required three fresh Jev consultations were made for the changed architecture. Facts and alternatives stayed equivalent; every context, instruction and option description was rewritten and checked before sending. All three `jev-1.13.0` responses chose the minimal native trusted runner, dependency-paired closed cut and a stopping rule based on real results/defect/completeness/cleanup. No disagreement occurred. They used 4,045 input / 360 output tokens; agreement remains advice. Requests/responses and wording checks are saved as `deletion-jev-*` beside this merged report. The earlier consultation records remain provenance for the first diagnosis.

The proposed deletion is not implemented. Current source and all 28 Manolea baseline contents remain intact. A separate uncommitted `AGENTS.md` change appeared during final verification; it is left untouched and excluded from these planning commits. The executor must apply paired repairs, run bounded affected checks and the required runtime lint/format/check scripts after authored runtime edits, and commit coherent increments immediately. Git source objects are the backup; this plan adds no archive or re-review ladder.
