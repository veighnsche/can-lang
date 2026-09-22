# Language design fixes — ordered implementation tasks

Status: **21 pending tasks; implementation has not started for this revision list.** This is the new work queue for the September 22 fixes. The completed [I01–I50 list](tasks.md) remains historical. Read the [implementation plan](language-fixes-plan-2026-09-22.md) first.

Execute LF01 through LF21 in the written order. Every declared prerequisite occurs earlier; numeric order is a safe topological order. A task is complete only when its code, necessary current-source migration and listed evidence pass. Do not check a box because a specification already exists. Each task contributes to the full linked AE contract; LF21 audits integrated closure. Paths below are current owner directories/files; new focused files may be added inside them.

- [x] **LF01 — Freeze revision inputs and the acceptance baseline**

  **Depends on:** none. **Acceptance:** AE49.

  **Owners:** `docs/implementation`, `docs/syntax-taste/evidence/2026-09-22`, `distribution/target.json`.

  **Work:** Capture the implementation starting commit, working changes, canonical specification hashes, catalogue and bundled Bun identity. Reuse the saved eight-call source, four application measurements and G1–G5 projects. Create a result manifest for all 16 AE records; preserve original before evidence unchanged. Record actual commands/environment for reruns and keep raw data outside normal diagnostic output.

  **Positive evidence:** All accepted records have source/fixture/native evidence slots and the known baseline hashes resolve.

  **Negative evidence:** Refuse a comparison with changed operations, missing failure cases or substituted acceptance for implemented evidence.

  **Integration/exit:** Rerun only the bounded G1–G4 controls needed to establish the actual starting checkout; preserve G5 as the resource regression constraint. Save any drift rather than reclassifying it silently.

  **Completion evidence:** [baseline](evidence/2026-09-22/language-fixes/LF01-baseline.json), [controls](evidence/2026-09-22/language-fixes/LF01-controls.json), [AE manifest](evidence/2026-09-22/language-fixes/LF01-AE-results.json), [runner](evidence/2026-09-22/language-fixes/LF01-probes.py). Starting commit 7b1542c, clean tree, docs-only drift from 760405f; G1–G4 reproduced, G5 preserved.

- [x] **LF02 — Carry precise semantic source spans through diagnostics**

  **Depends on:** LF01. **Acceptance:** AE44.

  **Owners:** `compiler/internal/source`, `compiler/internal/resolve`, `compiler/internal/check/errors.go`, `compiler/internal/driver/diagnostics.go`, `tools`.

  **Work:** Introduce/extend structured primary and related spans from resolver/checker through diagnostic serialization and the existing editor bridge. Preserve source identity and byte-to-UTF-16 conversion. Establish span support before new wrapper/template diagnostics; do not infer offsets from message prose.

  **Positive evidence:** G4 line-8 name/type errors point to the expression, including astral text and unsaved overlays.

  **Negative evidence:** Lexical control diagnostics retain their correct locations; imported/absent or stale overlay locations cannot silently become line 1.

  **Integration/exit:** Read-only editor diagnostics do not build or run a program. Existing runtime source-map locations stay correct. Broader obligation explanations and validated fixes close in LF17.

  **Completion evidence:** [summary](evidence/2026-09-22/language-fixes/LF02-summary.json), [LSP results](evidence/2026-09-22/language-fixes/LF02-lsp.json), [runner](evidence/2026-09-22/language-fixes/LF02-lsp.py), [full suite log](evidence/2026-09-22/language-fixes/LF02-gotest.log). G4 semantic/resolve now highlight line 8; lexical control unchanged; astral/overlay/imported/unavailable covered; driver/source/resolve/check/compiler suites green (1 pre-existing syntax failure documented, untouched).

- [x] **LF03 — Separate assertion staging from production publication**

  **Depends on:** LF01. **Acceptance:** AE22.

  **Owners:** `compiler/internal/driver/commands.go`, `compiler/internal/driver/assert.go`, `compiler/internal/driver/output.go`, `compiler/internal/driver/output_metadata.go`.

  **Work:** Split emission/output validation and private assertion staging from atomic production selection. Reuse owned dist confinement, writer protection and reader leases. Give assert a nonpublishing path on success, failure and interruption; do not replace the existing atomic selection mechanism.

  **Positive evidence:** An assertion run uses staged checked modules and leaves an existing production current selection unchanged.

  **Negative evidence:** Failed/hanging assertion execution never selects test output; malformed staging, ownership or confinement still rejects.

  **Integration/exit:** G2 reproducer no longer changes current, including a project with no prior current. Full build verification is deliberately not claimed until LF15.

  **Completion evidence:** [summary](evidence/2026-09-22/language-fixes/LF03-summary.json), [G2 results](evidence/2026-09-22/language-fixes/LF03-g2.json), [runner](evidence/2026-09-22/language-fixes/LF03-g2.py). Stage/SelectCurrent split with generation leases; assert never selects current on success/failure/kill; driver suite green.

- [x] **LF04 — Supervise assertion roots with bounded external deadlines**

  **Depends on:** LF02, LF03. **Acceptance:** AE23.

  **Owners:** `compiler/internal/driver/assert.go`, `compiler/internal/driver/runtime.go`, `runtime/assert`, `compiler/internal/ir/assertions.go`.

  **Work:** Implement P15.1 per-root workers, stable root order and fresh harness contexts. Add --assert-timeout-ms 1..600000, default 5000. Measure monotonic time before launch through result/drain delivery; timeout wins at equality. Kill outside the worker, cap reaping at 1000ms and stop on isolation failure. Extend worker/report protocol for root and last-known source/invocation progress without secrets.

  **Positive evidence:** Passing roots drain and report within budget; CLI selection remains explicitly partial and nonpublishing.

  **Negative evidence:** Test pending empty race, noncooperating CPU worker, late pass, crash/protocol failure, invalid budgets and unconfirmed reaping. The prior recursive stress case is not claimed as an established CPU hang.

  **Integration/exit:** G3 now returns a bounded failure report. Production empty-race semantics stay native and pending; shutdown deadlines cannot be caught in Can. Share this supervisor with build in LF15.

  **Completion evidence:** [summary](evidence/2026-09-22/language-fixes/LF04-summary.json), [CLI results](evidence/2026-09-22/language-fixes/LF04-assert.json), [runner](evidence/2026-09-22/language-fixes/LF04-assert.py), [full suite log](evidence/2026-09-22/language-fixes/LF04-gotest.log). Per-root workers with 5000ms default budget; G3 bounded; crash/protocol/late-pass judged; init-failure shape + broken-pipe contracts preserved; assert/coordination/fetch/generation/judge suites green.

- [x] **LF05 — Implement exact generic-error heads and aliases**

  **Depends on:** LF02. **Acceptance:** AE17.

  **Owners:** `compiler/internal/syntax/matches.go`, `compiler/internal/check/patterns.go`, `compiler/internal/check/pattern_coverage.go`, `compiler/internal/check/aggregate.go`, `compiler/internal/ir/regions.go`, `compiler/internal/emit/regions.go`.

  **Work:** Parse/resolve exact qualified error heads with type arguments and optional aliases under C5.1. Match complete nominal specializations, allow unambiguous bare heads and retain exact unaliased forwarding shorthand. Apply ordinary data/call/chain and Q ownership sets, including distinct nested all_failed leaves in one named outer race variant. No covariance, payload flattening or global variant search.

  **Positive evidence:** Two exact all_failed specializations dispatch correctly across every applicable mode; aliases bind only the chosen payload.

  **Negative evidence:** Ambiguous bare heads, duplicate coverage, missing specializations, alias without arrow and incomplete outer failure variants reject with spans.

  **Integration/exit:** Execute emitted matches and nested aggregates, preserving payloads and occurrence data. Add the pattern representation required by wrapper policy in LF12.

  **Completion evidence:** [summary](evidence/2026-09-22/language-fixes/LF05-summary.json), [emit](evidence/2026-09-22/language-fixes/LF05-emit.txt), [compiler log](evidence/2026-09-22/language-fixes/LF05-gotest.log), [integration log](evidence/2026-09-22/language-fixes/LF05-integration.log). Exact heads + aliases across parse/check/race; 25/25 fixture roots incl. nested outer aggregate; 9 negative subtests + span + generic-spec tests; all 23 bundled suites + strict tsc green.

- [x] **LF06 — Enforce failure arms before final success**

  **Depends on:** LF05. **Acceptance:** AE11.

  **Owners:** `compiler/internal/check/completion_matches.go`, `compiler/internal/check/coordination.go`, `compiler/internal/syntax/matches.go`, `compiler/testdata/current`.

  **Work:** Enforce C5.1 order in each applicable completion region and migrate affected admitted fixtures/goldens. Keep shared and per-entry coordination owners distinct; do not reorder ordinary data patterns, native question options or handler execution.

  **Positive evidence:** Failure-first call/chain and applicable coordination arms accept and retain original recovery result types.

  **Negative evidence:** Success before a later error/standard arm, missing success and duplicate success reject at the actual offending head.

  **Integration/exit:** Compare runtime dispatch/settlement before and after source migration; changing source order must not change native Promise combinators or scheduling.

  **Completion evidence:** docs/implementation/evidence/2026-09-22/language-fixes/LF06-summary.json, LF06-gotest.log (15 ok; only pre-existing TestCoreConsumerSpecification docs-drift panic), LF06-integration.log (integration+driver+emit all ok), LF06-runtime.log (321 pass), LF06-emit-diff.txt (branch-order-only diff; settle multiset identical). Reproduction commands in the summary.

- [x] **LF07 — Expose standard snapshots in ordinary catches**

  **Depends on:** LF05, LF06. **Acceptance:** AE15.

  **Owners:** `compiler/internal/check/completions.go`, `compiler/internal/emit/regions.go`, `runtime/failure.ts`, `runtime/coordination.ts`, `runtime/assert`.

  **Work:** Replace string binders with standard_failure snapshot binders in ordinary and applicable coordination catches. Reuse occurrence_id, kind, message, six standard categories, canonical sanitization and private cause storage. Update current sources directly; no legacy binder alias.

  **Positive evidence:** The same failure observed through ordinary catch and aggregate snapshot retains identity and projections.

  **Negative evidence:** Reject str binders, snapshot construction/update/wire encoding or snapshot domain bounds. Proxy/getter/throwing-toString descriptions cannot execute arbitrary code.

  **Integration/exit:** Run argument-failure catch boundaries, handler-failure escape and sticky harness recovery cases. Timeout remains outside Can catches.

  **Completion evidence:** docs/implementation/evidence/2026-09-22/language-fixes/LF07-summary.json, LF07-gotest.log (15 ok; only pre-existing TestCoreConsumerSpecification docs-drift panic), LF07-integration.log (integration+driver+emit all ok), LF07-runtime.log (321 pass). Reproduction commands in the summary.

- [ ] **LF08 — Add named runtime checks and migrate arithmetic traps**

  **Depends on:** LF02, LF06. **Acceptance:** AE29.

  **Owners:** `compiler/internal/catalogue/catalogue.json`, `compiler/internal/catalogue/cmd/cataloguegen`, `runtime`, `compiler/testdata/current/fetch`.

  **Work:** Implement C9.2 checks::require(bool, str) -> void emits [checks::failed], core ID 1010 and exact reason payload. Generate catalogue mirrors through the existing generator. Lower to a native boolean branch with existing domain/diagnostic adapters. Replace deliberate divide-by-zero helpers and update real caller bounds/arms and attached negative assertions.

  **Positive evidence:** True, false, empty reason and ordinary recovery behave as specified; both arguments evaluate once in order.

  **Negative evidence:** Wrong types/arity, unchecked errorful calls, missing bounds/arms and manufactured standard expectations reject. A caught check cannot clear a separate harness violation.

  **Integration/exit:** Execute the complete C9.2 library and migrated helpers. Check correct call-site spans, exact expected domain errors, native lowering and no production disabling. No new standard category or assertion DSL.

  **Completion evidence:** pending; attach exact inputs, commands, result manifests and relevant logs before checking this item.

- [ ] **LF09 — Represent failure origin at native operation boundaries**

  **Depends on:** LF02, LF05. **Acceptance:** AE07, AE09.

  **Owners:** `compiler/internal/check/error_plan.go`, `compiler/internal/ir/fetch.go`, `compiler/internal/ir/judge.go`, `runtime/domain.ts`, `runtime/transport`, `runtime/ai`, `compiler/internal/emit`.

  **Work:** Add compiler-private native/emitted origin and original-operation metadata where failures arise. Keep raw native obligations N distinct from declared authored obligations E, including same-named errors. Preserve original cause and forwarding identity; argument failures before target entry stay outside the wrapper boundary. This task supplies infrastructure, not a new public error API.

  **Positive evidence:** Native codec failure and authored codec::invalid_data remain distinguishable even with identical payloads.

  **Negative evidence:** Broad catch-around-handlers, nominal-name-only classification, relabelling unknown native exceptions or caller argument failures fails targeted regression checks.

  **Integration/exit:** Exercise one-request judge preparation, whole-answer validation and ordered handlers; authored/check failures retain emitted provenance. Save private occurrence traces for LF10/LF12.

  **Completion evidence:** pending; attach exact inputs, commands, result manifests and relevant logs before checking this item.

- [ ] **LF10 — Normalize public fetch/judge infrastructure errors**

  **Depends on:** LF06, LF09. **Acceptance:** AE07.

  **Owners:** `compiler/internal/catalogue/catalogue.json`, `compiler/internal/check/fetch.go`, `compiler/internal/check/judge.go`, `compiler/internal/emit/fetch.go`, `runtime/transport`, `runtime/ai`, `compiler/testdata/current`.

  **Work:** Implement A2.4 request_failed with ID 1106 and failure_detail containing the seven existing exact typed errors. Change required fetch/judge public bounds and adapter completion mapping at native origins only. Migrate current source and catalogue references as one coherent change, retaining explicit authored surplus errors.

  **Positive evidence:** All seven detail payloads survive selective recovery. Body-only status fails; envelope non-2xx succeeds; valid JSON with invalid AI answer remains ai::invalid_answer.

  **Negative evidence:** Unknown standard exceptions, authored raw-named failures, LLM/SQL/standalone codec errors and pre-entry arguments must not be swept into normalization.

  **Integration/exit:** Run eight-call equivalent source and actual decoder/transport fixtures; raw cause is private, mapped occurrence is fresh and forwarding stable. Source-count targets alone do not close AE07.

  **Completion evidence:** pending; attach exact inputs, commands, result manifests and relevant logs before checking this item.

- [ ] **LF11 — Attach native assertion roots and raw exchange fixtures**

  **Depends on:** LF04, LF08, LF10. **Acceptance:** AE25.

  **Owners:** `compiler/internal/syntax/native.go`, `compiler/internal/check/assertions.go`, `compiler/internal/ir/assertions.go`, `compiler/internal/emit`, `runtime/assert`, `runtime/transport`, `runtime/ai`.

  **Work:** Implement P4.1 mandatory fetch/judge/LLM assertions, inherited input/state binding and using raw mode for attached roots and native lexical when rows. Validate can.native-fixture.v1 request/environment/exchange data, execute real preparation/decoder/handlers and compare actual completion before delivery. Questions/arms retain judge-owned transport tests. Migrate existing native declarations and add their raw files in this task; dependency fixture hashing completes in LF14.

  **Positive evidence:** Test request method/URL/header/body comparison, exact JSON integer tokens, fake credentials, envelope responses, full judge validation/dispatch and preparation failures with null exchange.

  **Negative evidence:** Reject malformed/extra modes, wrong targets/inputs, missing request/decoder coverage, live calls, unused exchanges, wrong outcomes and raw mode on ordinary function targets.

  **Integration/exit:** Run real native roots under LF04 supervision; keep supplied-completion, raw-provider-fixture and real-can evidence distinct. Exercise dependency and grouped-state roots; no real credentials or live quality calls.

  **Completion evidence:** pending; attach exact inputs, commands, result manifests and relevant logs before checking this item.

- [ ] **LF12 — Implement derived wrappers, calculated bounds and handler tests**

  **Depends on:** LF05, LF07, LF09, LF10, LF11. **Acceptance:** AE09, AE25.

  **Owners:** `compiler/internal/syntax`, `compiler/internal/resolve`, `compiler/internal/check/error_plan.go`, `compiler/internal/ir`, `compiler/internal/emit`, `runtime/assert`.

  **Work:** Implement A3.2 wrap declarations with one base, explicit emits calculated, attached tests and origin-specific handlers. Resolve original N/E tables, overrides and terminal inherit; compute finite public bounds with explicit-bound callees as leaves. Extend native roots/raw execution to wrappers and add attached-only typed policy injection and policy-fixture reports. Keep complete target signature, grouped state, connection and callable eligibility.

  **Positive evidence:** Three-generation wrappers override consumed keys, delegate selectively, retain omitted handlers and expose exact calculated bounds. Test each locally declared key without borrowing parent coverage.

  **Negative evidence:** Reject base/bound cycles, impossible/duplicate keys, multiple bases, signature changes, ordinary/LLM/question targets, misplaced inherit, injected standard failures and consumer policy injection.

  **Integration/exit:** Run eight-call selective recovery, 404 override/429 inherit and same-type native/emitted failures. Original operation runs once; handler failures never redispatch; fully replaced bounds disappear only when no other path emits them.

  **Completion evidence:** pending; attach exact inputs, commands, result manifests and relevant logs before checking this item.

- [ ] **LF13 — Expand typed fixture templates at lexical owners**

  **Depends on:** LF11, LF12. **Acceptance:** AE27.

  **Owners:** `compiler/internal/syntax`, `compiler/internal/resolve`, `compiler/internal/check/assertions.go`, `compiler/internal/check/sites.go`, `compiler/internal/project`, `runtime/assert`.

  **Work:** Implement P3.1 fixture declarations and local selector: use expansion with typed inert arguments and an exact target identity, including supported concrete generic targets and wrappers. Preserve queue ownership, invocation/participant/callable fingerprints and FIFO assignment. Resolve raw paths at the defining source file; retain definition/case/use provenance for LF14/LF17.

  **Positive evidence:** One template reused at two sites creates independent local queues; imported/raw/grouped-state cases behave like equivalent inline rows.

  **Negative evidence:** Reject executable/captured template arguments, wrong exact targets, nested templates, wildcard matching, duplicate claims and missing/unused rows.

  **Integration/exit:** Compare inline versus expanded argument/capture checks, repeated concurrent visits and source-relative raw files in dependencies. No root/ordinal override mechanism or extra runtime dispatch is introduced.

  **Completion evidence:** pending; attach exact inputs, commands, result manifests and relevant logs before checking this item.

- [ ] **LF14 — Capture complete verification inputs and lock raw fixtures**

  **Depends on:** LF03, LF11, LF12, LF13. **Acceptance:** AE22, AE27.

  **Owners:** `compiler/internal/project/graph.go`, `compiler/internal/project/lock.go`, `compiler/internal/project/assets.go`, `compiler/internal/driver/output_metadata.go`.

  **Work:** Extend immutable input capture to every referenced fixture/asset and all assertion/native/wrapper roots. Implement required dependency fixtures_sha256 using exact P15.1 prefix, path ordering, deduplication and 64-bit length framing. Resolve confinement before reading and feed workers captured bytes. Bind source/config/lock/catalogue/runtime/compiler/options/root/timeout identities without silently rewriting locks.

  **Positive evidence:** Empty/shared/imported template fixture trees produce stable digests; lock and staged workers observe the same bytes.

  **Negative evidence:** Path escape, symlink violations, stale locks, missing files and source/fixture/asset edits between capture and revalidation reject.

  **Integration/exit:** Change only a dependency raw fixture and prove verification identity/lock mismatch changes. Keep active generation leases and owned dist invariants intact.

  **Completion evidence:** pending; attach exact inputs, commands, result manifests and relevant logs before checking this item.

- [ ] **LF15 — Gate atomic production publication on all-root verification**

  **Depends on:** LF04, LF08, LF11, LF12, LF13, LF14. **Acceptance:** AE22.

  **Owners:** `compiler/internal/driver/commands.go`, `compiler/internal/driver/assert.go`, `compiler/internal/driver/output.go`, `runtime/assert`, `tests/integration`.

  **Work:** Wire P15.1 capture → check → private test staging → supervise every root → validate production staging → revalidate inputs → publish. Include required dependency/native/wrapper roots, no selected-only or cached-pass bypass. Publish exactly the verified production generation and report identities, evidence totals and deadline policy. Assert remains nonpublishing; build does not execute live main.

  **Positive evidence:** All required passing roots publish one complete verified generation; selected assert runs report their partial scope.

  **Negative evidence:** Any failed/missing/timed-out root, sticky violation, coverage gap, invalid generated TS, worker isolation failure or input drift prevents publication. No skip/disabled-deadline flag.

  **Integration/exit:** G1/G2 regressions pass; fault-inject interrupted publication and preserve either the old complete generation or fully verified new one. Existing readers retain leases. No unexpected live work occurs.

  **Completion evidence:** pending; attach exact inputs, commands, result manifests and relevant logs before checking this item.

- [ ] **LF16 — Replace recursive aggregate widening with explicit native mapping**

  **Depends on:** LF05, LF07. **Acceptance:** AE18.

  **Owners:** `compiler/testdata/current`, `tests/integration`, `runtime/collections/array.ts`, `docs/implementation/coordination.md`.

  **Work:** Build the AE18 named element converter and explicit all_failed reconstruction against the implemented exact heads/snapshots. Replace demonstrated recursive slicing scaffolding where a shared aggregate is actually needed. Reuse the existing protected native Array.fromAsync mapping; do not add a covariance rule or collection implementation.

  **Positive evidence:** Empty, singleton, repeated, large and mixed/nested failure arrays preserve values, order and snapshot identity.

  **Negative evidence:** Implicit narrow-array-to-wide-array assignment, missing target variant leaves and incompatible callback bounds still reject.

  **Integration/exit:** Run recursive and map baselines on identical data, including data-valued then fields; measure actual reduced scaffolding. If the idiom fails, save the reproducer and resolve the defect without inventing an abstraction.

  **Completion evidence:** pending; attach exact inputs, commands, result manifests and relevant logs before checking this item.

- [ ] **LF17 — Complete obligation diagnostics and checker-validated fixes**

  **Depends on:** LF02, LF05, LF06, LF07, LF12, LF13, LF15. **Acceptance:** AE44.

  **Owners:** `compiler/internal/check`, `compiler/internal/driver/diagnostics.go`, `compiler/internal/project/overlay.go`, `tools`, `runtime/assert`.

  **Work:** Add explanations and related spans for missing arms, calculated bounds/override chains, generic alternatives, fixture definition/use, capture requirements and pending invocation progress. Generate only fixes that can be checked against an isolated current overlay snapshot. Preserve declared contracts and recovery ownership; omit unavailable safe fixes.

  **Positive evidence:** Valid repairs typecheck with exact primary/secondary locations, including cross-package astral/overlay cases.

  **Negative evidence:** Reject stale edits and fixes that suppress errors, flatten state, silently widen APIs or change completion ownership. Reports never dump private captures/causes.

  **Integration/exit:** Run AE44 end to end across the final new source forms; read-only tooling never builds, publishes or calls providers. Keep existing runtime source mappings unchanged except corrected metadata.

  **Completion evidence:** pending; attach exact inputs, commands, result manifests and relevant logs before checking this item.

- [ ] **LF18 — Preserve source trivia and provide safe formatting**

  **Depends on:** LF05, LF06, LF11, LF12, LF13, LF17. **Acceptance:** AE45.

  **Owners:** `compiler/internal/syntax/lexer.go`, `compiler/internal/syntax/format.go`, `compiler/internal/syntax/native_format.go`, `compiler`, `tests/integration`.

  **Work:** Extend the existing parser/renderer with owned trivia, not a second parser. Document and implement a single-file canlc format <file> interface: stdout by default, --write for validated replacement. Resolve the containing project for semantic validation; check proposed content through an overlay before write, recheck original file identity and use atomic replacement. Add command help and tests together. This is a planned tooling interface, not already available syntax.

  **Positive evidence:** Comments/docstrings/blank groups/raw and triple strings survive; output is idempotent across all new forms and CRLF/UTF-8 inputs.

  **Negative evidence:** Invalid syntax/semantics, stale source, failed validation/write or unsupported trivia leaves source untouched. Never repair arm order or change string contents silently.

  **Integration/exit:** Before/after checked IR and runtime completions agree, ignoring source-position shifts; lexical preorder and fixture FIFO allocation stay equivalent. No multiline language extension or runtime formatter.

  **Completion evidence:** pending; attach exact inputs, commands, result manifests and relevant logs before checking this item.

- [ ] **LF19 — Publish accurate capability and language guides**

  **Depends on:** LF08, LF10, LF12, LF13, LF15, LF16, LF17, LF18. **Acceptance:** AE33, AE47.

  **Owners:** `docs/implementation`, `docs/syntax-taste`, `std/catalogue`, `README.md`, `compiler/testdata/current`.

  **Work:** Write one capability admission/evolution guide with a real pure operation and resource operation: signature, native mapping, error translation, immutability/lifetime, fixture/conformance and registry artifacts. Correct finite inference/variant inclusion/container invariance, explicit numeric conversion, configuration validation, formatter and build/assert claims using actual results. Keep historical review quotes labelled; update current docs when behavior really exists.

  **Positive evidence:** Executable positive examples cover finite expected typing/inference, leaf inclusion, callable subsets and native numeric/string edge cases.

  **Negative evidence:** Pair with mixed numeric, unresolved inference, nominal mismatch/covariance and invalid manifest negatives; classify an incomplete capability proposal as incomplete.

  **Integration/exit:** Cross-check current guides/specs/catalogue and reported outputs. Pure documentation changes need link/worked-example checks rather than unrelated runtime rewrites; no third-party native binding mechanism.

  **Completion evidence:** pending; attach exact inputs, commands, result manifests and relevant logs before checking this item.

- [ ] **LF20 — Measure equivalent programs and authoring workflows**

  **Depends on:** LF01, LF10, LF12, LF13, LF15, LF16, LF17, LF18, LF19. **Acceptance:** AE49.

  **Owners:** `docs/syntax-taste/evidence/2026-09-22`, `compiler/testdata/current`, `tests/integration`.

  **Work:** Rerun the frozen eight-call comparison and four application baselines after behavior acceptance. Keep eight operations/nesting and full error/fixture coverage. Measure adding a domain failure, changing a request field, changing an existing capture value and reusing a scenario at two sites. Separate production and total test/fixture source costs. Record model/settings/attempts/edits/diagnostic cycles/tokens when actually measured.

  **Positive evidence:** Equivalent complete projects pass behavioral gates before source-cost comparisons; target 63→9 bound entries and 56→8 infrastructure arms in the frozen region.

  **Negative evidence:** Omitted errors/tests, altered requests/order, wrong-boundary mocks or non-equivalent TypeScript baselines invalidate comparisons, even if shorter.

  **Integration/exit:** Publish raw metrics and edit traces including failures; no arbitrary ratio or statistical generality from one run. Deferred abstractions require their separate demonstrated-need gate, not automatic promotion from counts.

  **Completion evidence:** pending; attach exact inputs, commands, result manifests and relevant logs before checking this item.

- [ ] **LF21 — Qualify the integrated revision and close evidence records**

  **Depends on:** LF15, LF16, LF17, LF18, LF19, LF20. **Acceptance:** AE07, AE09, AE11, AE15, AE17, AE18, AE22, AE23, AE25, AE27, AE29, AE33, AE44, AE45, AE47, AE49.

  **Owners:** `tests/integration`, `tests/conformance`, `distribution`, `docs/implementation/evidence/2026-09-22/language-fixes-plan`.

  **Work:** Run the relevant complete compiler/runtime/bundled integration suites on the pinned target, regenerate/check catalogue mirrors and audit every AE dimension and BC case. Regress retained native AI/state/capture/contracts/coordination/resource behavior. Qualify a local bundle with no external runtime or live credentials. Update per-task evidence and the current implementation docs only from passing results; release upload/signing is separate authorization.

  **Positive evidence:** Every accepted AE record has a complete result manifest, correct negative diagnostics, runtime/native/fixture evidence and no unresolved completion gate.

  **Negative evidence:** Skipped, timed-out, stale, wrong-boundary, contradictory or documentation-only behavior evidence cannot close an item. G5 runtime resource enforcement must remain intact without adding static escape scope.

  **Integration/exit:** Exercise the end-to-end normalized request → inherited wrapper → local/raw/template assertion → verified build → reader/run flow. Preserve old complete generation on failure. Task list closure means these revisions passed, not that arbitrary live services or untested inputs are certified.

  **Completion evidence:** pending; attach exact inputs, commands, result manifests and relevant logs before checking this item.

## Contract and scope checks

All AE IDs refer to [the acceptance specification](language-change-acceptance-2026-09-22.md). All 49 finding dispositions remain in [the scope ledger](language-design-dispositions-2026-09-22.md); only its 16 implement rows enter this queue. The [machine-readable dependency graph](evidence/2026-09-22/language-fixes-plan/tasks.json) and [validator](evidence/2026-09-22/language-fixes-plan/validate.py) check order and coverage. This list creates no desktop task, branch, release or automatic execution.
