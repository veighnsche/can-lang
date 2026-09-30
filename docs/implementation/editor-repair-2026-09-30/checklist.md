# Implementation checklist: complete Can editor repair

## Active executor override — 2026-09-30 Codex takeover

The user's replacement AGENTS instructions make Codex the default executor. The legacy Muse run is stopped (PID99423, SIGINT, exit130, no descendants); native goal tools were unavailable. All implementation, coordination and checklist ownership now belongs to Codex and explicitly assigned Codex subagents. No new Muse execution without explicit current-task opt-in. The historical Muse-specific native-goal and schedule/yield clauses below document the original launch only and do not constrain Codex. Keep all substantive requirements, design decisions, resource limits and acceptance checks. F00-F03 are claimed implemented by the stopped run and remain independently unverified. The coordinator may repair those interfaces; communicate changes to active lane owners first.

Current lane ownership is assigned by Codex via collaboration messages. All compiler/editor/server source corrections, independent review and integrated/race verification are complete. All writers are released. Source commit ab764c08 is integrated into main; host package, live Cursor verification and cleanup remain open. Subagents do not edit this checklist or monitor; they report compact evidence to root. One heavy compiler/test command at a time, no benchmarks or broad builds. No edits to the main checkout's code until reviewed integration. The dead tmux pane has an attached viewer; preserve its socket until the viewer detaches.

## Handoff contract

Read [design.md](design.md) in full; it contains requirements R01-R14, evidence, selected contracts D1-D6, consultation disposition, acceptance examples and constraints. This file is the complete ordered assignment, not a prompt to replace the plan. User authorized the entire repair, including later reviewed installation and actual Cursor verification.

- **Implementation workspace:** `/Users/vince/.codex/worktrees/editor-repair/can-lang`, branch `codex/editor-repair`, baseline `9c283544466876700ef7a079f0bc3136451c0008`.
- **Codex monitor:** `/Users/vince/Projects/can-lang/docs/implementation/editor-repair-2026-09-30/monitor.md`. Codex alone owns it. It is authoritative for mutable runtime/cleanup/heartbeat state.
- **Implementation executor:** one Muse Spark 1.3 Contributor coordinator at MAX. Coordinator alone writes this checklist's progress/evidence; its agents edit assigned source files only. Do not launch another Muse CLI coordinator or use unrequested worktrees.
- **Native goal:** before delegating, create a native goal to execute all Muse-owned tasks through evidenced handoff with writers released. Inspect/reuse matching goal on continuation; don't overwrite unrelated goals. No arbitrary token/total-step budget. Keep report_progress grounded in evidence and reserve 100% for handoff; update_goal complete only at H75. If native goal tools are missing/rejected, report the blocker.
- **Command cadence:** under that goal, routine reinspection of long commands waits 1–5 minutes (default 2) from the last check. Use native bash initial `yield_time_ms: 120000`, adjustable 60000–300000. Retain handle/deadline in command state; goal wakeups do not reset it or cause empty bash_input polling. Await automatic completion; use runtime-overdue inspection only after deadline. Act immediately on completion/failure/input/intervention. Propagate this contract into subagent prompts. If runtime caps waits shorter, preserve deadline across supported increments or report the limit.
- **Ownership:** initially compiler lane owns F01-R27, grammar/client lane G50-G53/C60-C62; editor/server lane E30-S43 starts only after shared contracts and explicit file transfer. Because `lsp.go`, driver files and manifests overlap, serialize their writers or extract interfaces first. Coordinator records actual ownership below. At most two coding agents at once and only one heavy compile/test/build at a time.
- **Resources:** ~9.6GiB free on a 256GB MacBook. No benchmarks or broad distribution/build matrix. Bounded serialized correctness checks and one reused host release build are permitted. Use normal shared caches, no private full caches/source copies. Allocate temporary directories only with registered success/failure/interruption cleanup; preserve compact evidence, not transcripts/bundles. If runtime TypeScript is edited, run AGENTS runtime lint-fix/format/check requirements.
- **Boundary:** no code edits to main checkout or manolea-2, no installing into user Cursor before Codex review, no app/runtime/assertion execution during editor analysis, no credential reading/transmission. Jev consultation uses existing TYPESAFE_API_KEY without printing it if another difficult design decision arises; three fresh full rewrites required.
- **Plan changes:** report material design blockers with evidence and affected IDs; stop affected writers and explicitly release ownership before Codex edits contracts. Routine implementation choices within this design are yours. Never silently narrow multi-diagnostic or feature acceptance to finish.
- **Handoff:** all Muse-owned tasks must be implemented and evidenced, then all writers released. Leave V80, V81, D90-D92 and C99 unticked for Codex. An idle process or goal dispatch is not completion.

## Current ownership

- 2026-09-30 F00: coordinator = this Muse session (sole checklist/evidence writer). Compiler lane
  (F01-R27) = coordinator inline; no separate writer yet. Grammar/client lane (G50-G53/C60-C62) =
  unassigned, ready after F00. Editor/server lane (E30-S43) = unassigned, blocked until F03 contracts +
  explicit transfer. `compiler/lsp.go` + driver files serialized to one writer. Max two coding agents,
  one heavy command. Native goal tools absent — see evidence/baseline.md blocker; no goal ID assigned.

## Dependency map

Foundation F00-F03 -> parser/project P10-P13 -> resolver/types/checker R20-R27. Grammar G50-G53 can run independently after F00. Shared editor facts E30 wait for recovered/compiler contracts; E31-E39 share one feature owner. S40-S43 integrate snapshot/worker/lifecycle after compiler/feature joins. C60-C62 join grammar/client to protocol. I70-I71 and H75 finish implementation. Codex then reviews, integrates, installs, verifies Cursor and cleans up.

## Tasks

- [x] **F00 — Record baseline and exclusive lane ownership**
  - Prerequisites: none.
  - Owner: Muse coordinator.
  - Files/interfaces: checklist.md; evidence/baseline.md; read-only repository state.
  - Changes/traceability: Record HEAD, clean/foreign changes, available disk, required tools, initial targeted tests and an ownership table. Re-read AGENTS.md. Assign at most two coding lanes concurrently and one heavy check process. Do not edit main checkout code or launch another coordinator. (R14; design baseline/resource policy; [design](design.md)).
  - Acceptance: Baseline state and ownership recorded; no foreign changes touched; real assigned file conflicts serialized.
  - Evidence: evidence/baseline.md — HEAD 9c283544, clean except task docs, 7.1GiB avail, go1.27.1/bun1.4.2, native-goal-tools-missing blocker, lane table, AGENTS re-read.

- [x] **F01 — Inventory diagnostic producers and exact location policy**
  - Prerequisites: F00.
  - Owner: Muse compiler lane.
  - Files/interfaces: compiler/internal/{syntax,project,resolve,types,check,driver}; evidence/diagnostic-coverage.md.
  - Changes/traceability: Enumerate every authored-source diagnostic family and warning/note producer, including bare errors and shared validation sites. Map code/severity/minimal span/related locations/test to producer; mark internal/project-only failures honestly. Record old tests needing replacement rather than preserving accidental ranges. (R01-R04; D1-D2; [design](design.md)).
  - Acceptance: Inventory accounts for syntax, imports/exports, type/error declarations, expressions, assertions, native/HTTP/fetch/SQL and config; no unexplained first-line source fallbacks.
  - Evidence: evidence/diagnostic-coverage.md — 12 families, 1057 bare vs 51 located sites, sticky Builder, single-syntax-code panic, first-diagnostic-only bridges, note→warning bug, old-test replacement list.

- [x] **F02 — Establish structured issues and complete source ranges**
  - Prerequisites: F01.
  - Owner: Muse compiler lane.
  - Files/interfaces: compiler/internal/source/*; compiler/internal/syntax/{ast,expressions,parser}.go; direct constructors/tests.
  - Changes/traceability: Define shared issue/related/fix/severity representation; retain operator and missing qualifier/member token spans while reusing existing callee/name spans. Specify invalid/blocked unit state. Preserve original byte offsets and valid EOF/insertion ranges; remove known-span widening. (R02-R04; D1-D2; [design](design.md)).
  - Acceptance: Small range tests prove callee/operator/argument spans and UTF-16 conversion for BOM/CRLF/astral/combining/EOF/multiline cases; no invalid AST constructor sites.
  - Evidence: evidence/f02-ranges.md — Issue/severity/UnitStatus, operator+qualifier/member spans, widening removed; syntax/source/driver suites green, build+vet clean.

- [x] **F03 — Freeze recoverable analysis and semantic-index interfaces**
  - Prerequisites: F02.
  - Owner: Muse coordinator with compiler lane.
  - Files/interfaces: compiler/internal/driver/editor*.go or appropriate compiler-owned analysis package; checklist interface evidence.
  - Changes/traceability: Define immutable analysis sources, full dependency fingerprint, partial graph/world, issues, per-unit validity, proven semantic facts and occurrence-index contracts. Record exact Go types/ownership before dependent lanes edit them. Keep compiler mutable state private until valid commit. (R01-R05,R07-R10; D1,D3-D4; [design](design.md)).
  - Acceptance: Contract review and compile-level tests establish deterministic issue ordering/deduplication, explicit blocking and no publication of partially mutable state.
  - Evidence: evidence/f03-contracts.md — editor_analysis.go + editor_index.go frozen types, 4 contract tests pass.

- [x] **P10 — Recover lexical findings without losing later independent source**
  - Prerequisites: F03.
  - Owner: Codex compiler/root lanes.
  - Files/interfaces: compiler/internal/syntax/lexer.go; lexer tests.
  - Changes/traceability: Collect recoverable independent lexical problems with bounded forward progress and real spans. Preserve tokens/trivia for safe continuation. Treat unterminated structures honestly and keep other files analyzable; no shadow lexer. (R01-R03,R06; D1; [design](design.md)).
  - Acceptance: Two independent lexical faults report at correct tokens; malformed escape/comment/string/nesting inputs terminate; valid lexer behavior remains correct.
  - Evidence: [compiler-recovery-codex.md](evidence/compiler-recovery-codex.md), [expression-review.md](evidence/expression-review.md), and [server-review.md](evidence/server-review.md). Full source/syntax/project/resolve/types/check/driver suites pass; concrete independent review findings closed with token-range and strict no-IR regressions. Final integration/provider gate remains I70/V80.

- [x] **P11 — Retain parser structure across safe recovery boundaries**
  - Prerequisites: P10.
  - Owner: Codex compiler/root lanes.
  - Files/interfaces: compiler/internal/syntax/{declarations,parser,expressions,native}.go; recovery tests.
  - Changes/traceability: Recover per declaration and safe field/argument/statement boundaries using canonical parser rules. Retain valid nodes plus explicit invalid regions. Synchronize by actual token/depth/indent structure; guard progress and do not invent executable syntax. (R01-R03,R08; D1; [design](design.md)).
  - Acceptance: Independent malformed declarations/files yield multiple findings; later valid declarations survive; incomplete call/member/type forms expose useful cursor context; invalid programs cannot count as parse success.
  - Evidence: [compiler-recovery-codex.md](evidence/compiler-recovery-codex.md), [expression-review.md](evidence/expression-review.md), and [server-review.md](evidence/server-review.md). Full source/syntax/project/resolve/types/check/driver suites pass; concrete independent review findings closed with token-range and strict no-IR regressions. Final integration/provider gate remains I70/V80.

- [x] **P12 — Locate project and configuration validation issues**
  - Prerequisites: F03.
  - Owner: Muse compiler lane, serialized with P10/P11 if files overlap.
  - Files/interfaces: compiler/internal/project/{manifest*,graph*,lock*,registry*,json*}.go and corresponding tests.
  - Changes/traceability: Carry JSON source/key/value locations and naming path-token origins through manifest/lock/error-registry/dependency/asset errors. Support source bytes for unsaved named config documents as needed. Use explicit project issues for truly locationless I/O instead of an arbitrary Can line. (R02,R04-R05; D2-D3; [design](design.md)).
  - Acceptance: Malformed JSON, missing dependency/path, registry mismatch and invalid SQL descriptor attribute correct JSON/token or explicit project status; no false Can squiggle.
  - Evidence: [compiler-recovery-codex.md](evidence/compiler-recovery-codex.md), [expression-review.md](evidence/expression-review.md), and [server-review.md](evidence/server-review.md). Full source/syntax/project/resolve/types/check/driver suites pass; concrete independent review findings closed with token-range and strict no-IR regressions. Final integration/provider gate remains I70/V80.

- [x] **P13 — Load partial graphs and overlay-only source files**
  - Prerequisites: P11,P12.
  - Owner: Codex compiler/root lanes.
  - Files/interfaces: compiler/internal/project/{graph,overlay}.go; project tests.
  - Changes/traceability: Discover every applicable disk and overlay-only Can source, retain partial syntax and configuration provenance, and aggregate independent load issues. Preserve canonical paths/source confinement and explicit missing-package/dependency blocking. Never write scratch buffers. (R01,R03,R05; D1,D3; [design](design.md)).
  - Acceptance: Two bad sources coexist with a valid source and its metadata; new unsaved files are checked; duplicate basenames/symlinks/close-to-disk behavior remain correct; bad package header does not fabricate imports.
  - Evidence: [compiler-recovery-codex.md](evidence/compiler-recovery-codex.md), [expression-review.md](evidence/expression-review.md), and [server-review.md](evidence/server-review.md). Full source/syntax/project/resolve/types/check/driver suites pass; concrete independent review findings closed with token-range and strict no-IR regressions. Final integration/provider gate remains I70/V80.

- [x] **R20 — Recover resolver symbols transactionally**
  - Prerequisites: P13.
  - Owner: Codex compiler/root lanes.
  - Files/interfaces: compiler/internal/resolve/{symbols,native}*.go and tests.
  - Changes/traceability: Collect duplicate/export/import/signature issues at exact tokens. Preserve independent valid symbols and mark ambiguous/invalid/blocked identities. Roll back failed symbol commits and carry dependency reasons into later phases. (R01-R03; D1-D2; [design](design.md)).
  - Acceptance: Multiple independent resolution errors coexist; unrelated valid symbols remain queryable; invalid imports/declarations do not produce invented unknown-name cascades.
  - Evidence: [compiler-recovery-codex.md](evidence/compiler-recovery-codex.md), [expression-review.md](evidence/expression-review.md), and [server-review.md](evidence/server-review.md). Full source/syntax/project/resolve/types/check/driver suites pass; concrete independent review findings closed with token-range and strict no-IR regressions. Final integration/provider gate remains I70/V80.

- [x] **R21 — Isolate type-builder failures and retain independent type facts**
  - Prerequisites: R20.
  - Owner: Codex compiler/root lanes.
  - Files/interfaces: compiler/internal/types/{builder,annotations}*.go; compiler/internal/check/errors.go; tests.
  - Changes/traceability: Replace sticky global failure propagation with explicit valid/invalid dependency state or transactional construction. Collect declaration/field/error-bound issues, retaining precise source provenance and independently valid types. Handle recursive dependency components without accepting invalid cycles. (R01-R04; D1-D2; [design](design.md)).
  - Acceptance: A bad type/field does not poison unrelated type/function checks; cyclic/recursive errors terminate; invalid types never become successful typed facts or emitted output.
  - Evidence: [compiler-recovery-codex.md](evidence/compiler-recovery-codex.md), [expression-review.md](evidence/expression-review.md), and [server-review.md](evidence/server-review.md). Full source/syntax/project/resolve/types/check/driver suites pass; concrete independent review findings closed with token-range and strict no-IR regressions. Final integration/provider gate remains I70/V80.

- [x] **R22 — Make checker contexts and specialization state recoverable**
  - Prerequisites: R21.
  - Owner: Codex compiler/root lanes.
  - Files/interfaces: compiler/internal/check/{program,specialize,template,bindings}*.go; state tests.
  - Changes/traceability: Refactor shared checker passes into independently committed contexts with explicit blocked prerequisites. Safely manage annotations, bindings, callables, specialization queues, registry and SQL-site/warning data. Reuse canonical validators, avoiding repeated whole-project replay or source masking. (R01,R03-R04; D1; [design](design.md)).
  - Acceptance: Failure in one body/specialization leaves unrelated bodies checkable; dependency chains and generic failures do not contaminate valid state; no nil panic on recovery inputs.
  - Evidence: [compiler-recovery-codex.md](evidence/compiler-recovery-codex.md), [expression-review.md](evidence/expression-review.md), and [server-review.md](evidence/server-review.md). Full source/syntax/project/resolve/types/check/driver suites pass; concrete independent review findings closed with token-range and strict no-IR regressions. Final integration/provider gate remains I70/V80.

- [x] **R23 — Collect expression and body diagnostics at minimal spans**
  - Prerequisites: R22.
  - Owner: Codex compiler/root lanes.
  - Files/interfaces: compiler/internal/check/{expressions,calls,region,completion,match}*.go and other located producers.
  - Changes/traceability: Migrate authored body errors to structured codes and exact name/callee/operator/member/argument/type spans. Continue independent child/statement checks where prerequisites are trustworthy. Suppress propagated invalid-type cascades and preserve useful related locations. (R01-R04; D2; [design](design.md)).
  - Acceptance: Unknown callee highlights only callee; bad arguments/operators/members select correct subranges; multiple independent body errors appear; unchanged valid language acceptance remains correct.
  - Evidence: [compiler-recovery-codex.md](evidence/compiler-recovery-codex.md), [expression-review.md](evidence/expression-review.md), and [server-review.md](evidence/server-review.md). Full source/syntax/project/resolve/types/check/driver suites pass; concrete independent review findings closed with token-range and strict no-IR regressions. Final integration/provider gate remains I70/V80.

- [x] **R24 — Locate and aggregate assertion and native contract findings**
  - Prerequisites: R22.
  - Owner: Codex compiler/root lanes.
  - Files/interfaces: compiler/internal/check/{assertions,native*,http,fetch,form,llm,questions,wrap}*.go; tests.
  - Changes/traceability: Carry exact declaration/row/contract field locations through duplicate assertion labels, expected expressions, native result/emits, connection/fetch/HTTP/form and other native validation. Continue independent contexts and retain duplicate origins. (R01-R04; D2; [design](design.md)).
  - Acceptance: Duplicate assertion rows underline the duplicate label with related original; independent native/assertion issues coexist and do not anchor line one.
  - Evidence: [compiler-recovery-codex.md](evidence/compiler-recovery-codex.md), [expression-review.md](evidence/expression-review.md), and [server-review.md](evidence/server-review.md). Full source/syntax/project/resolve/types/check/driver suites pass; concrete independent review findings closed with token-range and strict no-IR regressions. Final integration/provider gate remains I70/V80.

- [x] **R25 — Preserve SQL cross-site and remaining global diagnostic attribution**
  - Prerequisites: R22.
  - Owner: Codex compiler/root lanes.
  - Files/interfaces: compiler/internal/check/{sql,sql_sites,program}*.go; compiler/internal/sql adapters only if needed; tests.
  - Changes/traceability: Add source file/span to cross-site records and global contract validation, including SQL cardinality/type/site relationships and every remaining family in F01. Validate only complete prerequisite sets and distinguish blocked work. (R01-R04; D2; [design](design.md)).
  - Acceptance: SQL call-site issue points to the responsible Can construct with related descriptor/origin; F01 has no unaccounted authored-source bare errors.
  - Evidence: [compiler-recovery-codex.md](evidence/compiler-recovery-codex.md), [expression-review.md](evidence/expression-review.md), and [server-review.md](evidence/server-review.md). Full source/syntax/project/resolve/types/check/driver suites pass; concrete independent review findings closed with token-range and strict no-IR regressions. Final integration/provider gate remains I70/V80.

- [x] **R26 — Merge warnings, notes and independent errors into editor analysis**
  - Prerequisites: R23,R24,R25.
  - Owner: Codex compiler/root lanes.
  - Files/interfaces: compiler/internal/driver/diagnostics.go; compiler/internal/check/warnings.go; analysis tests.
  - Changes/traceability: Make CheckSnapshot consume canonical recoverable analysis once, retaining partial valid graph/world/index and all trustworthy issues. Preserve note versus warning severity, deterministic order/deduplication and related locations. Any cap must be explicit and must not masquerade as complete coverage. (R01-R04; D1-D2; [design](design.md)).
  - Acceptance: Two independent errors across files and a valid warning appear together; informational note is not a warning; repair removes affected issue and unblocks dependents; exact ranges and source versions are preserved.
  - Evidence: [compiler-recovery-codex.md](evidence/compiler-recovery-codex.md), [expression-review.md](evidence/expression-review.md), and [server-review.md](evidence/server-review.md). Full source/syntax/project/resolve/types/check/driver suites pass; concrete independent review findings closed with token-range and strict no-IR regressions. Final integration/provider gate remains I70/V80.

- [x] **R27 — Gate strict compiler output on complete valid analysis**
  - Prerequisites: R26.
  - Owner: Codex compiler/root lanes.
  - Files/interfaces: compiler/current*.go; compiler/internal/driver relevant check/build entry points; compiler/internal/check exported entry points.
  - Changes/traceability: Adapt shared strict entry points to updated canonical interfaces. Retain intended Can semantics, require zero errors before returning an emittable Program, and prevent recovery placeholders from emit/build/assert execution. Do not preserve obsolete API or first-error goldens solely for compatibility. (R03,R14; D1; [design](design.md)).
  - Acceptance: Selected known-invalid programs are rejected before emission, valid current fixtures still check, warning-only programs remain valid, and inert analysis never executes assertion/network poison.
  - Evidence: [compiler-recovery-codex.md](evidence/compiler-recovery-codex.md), [expression-review.md](evidence/expression-review.md), and [server-review.md](evidence/server-review.md). Full source/syntax/project/resolve/types/check/driver suites pass; concrete independent review findings closed with token-range and strict no-IR regressions. Final integration/provider gate remains I70/V80.

- [x] **E30 — Build the shared scope and occurrence index**
  - Prerequisites: F03,R20,R21,R22.
  - Owner: Muse editor lane after explicit ownership grant.
  - Files/interfaces: compiler/lsp.go refWalker/compWalker extraction; compiler/internal/driver/editor_index*.go or chosen package.
  - Changes/traceability: Consolidate existing reference/completion identity and cursor logic into the frozen analysis index. Record declaration/selection/name ranges, scopes, visibility, checked type/signature, call arguments and validity. Include provides/import entries and new syntax. Coordinate all lsp.go edits through one owner. (R07-R10; D4; [design](design.md)).
  - Acceptance: Shadowing/capture/pins/fields/generated/catalogue identities are stable; one snapshot/index serves multiple queries without repeated whole-project checks.
  - Evidence: [codex-review.md](evidence/codex-review.md), [acceptance.md](evidence/acceptance.md), and [integration.md](evidence/integration.md).

- [x] **E31 — Complete local definition, hover and reference behavior**
  - Prerequisites: E30,R26.
  - Owner: Muse editor lane.
  - Files/interfaces: compiler/internal/driver/{hover,diagnostics,editor*}.go; LSP feature files/tests.
  - Changes/traceability: Route definitions, hover and references through the shared index, supporting local declarations/uses/captures/pins and checked inferred types where proven. Preserve valid existing symbol navigation and related contract documentation. (R07; D4; [design](design.md)).
  - Acceptance: Local use jumps to exact declaration and hovers proven type; same-spelled shadowed bindings remain distinct; unresolved/blocked tokens never get guessed results.
  - Evidence: [codex-review.md](evidence/codex-review.md), [acceptance.md](evidence/acceptance.md), and [integration.md](evidence/integration.md).

- [x] **E32 — Provide useful completion on unfinished source**
  - Prerequisites: E30,P11.
  - Owner: Muse editor lane.
  - Files/interfaces: compiler completion/index/cursor feature files; completion tests.
  - Changes/traceability: Use recovered current syntax plus eligible scoped facts for completion, exact replacement edits and appropriate insertion text. Support incomplete callee/member/type/header/with contexts while keeping strings/comments and unavailable imports honest. (R08; D4; [design](design.md)).
  - Acceptance: Incomplete call and member probes return valid candidates; lexical-only contexts offer appropriate keywords; no private/owner/import leaks; repair to valid source agrees with canonical strict candidates.
  - Evidence: [codex-review.md](evidence/codex-review.md), [acceptance.md](evidence/acceptance.md), and [integration.md](evidence/integration.md).

- [x] **E33 — Add checked signature help**
  - Prerequisites: E30,E32.
  - Owner: Muse editor lane.
  - Files/interfaces: compiler signature-help/index feature files; wire tests.
  - Changes/traceability: Expose callable, method, constructor, callback and bound-input signatures using checked contracts. Derive active argument from nested spans; supply labels and documentation where known. (R08; D4; [design](design.md)).
  - Acceptance: Nested calls and commas in strings do not change outer active argument; method/constructor and with/pin contexts report correct remaining inputs; unresolved callees decline.
  - Evidence: [codex-review.md](evidence/codex-review.md), [acceptance.md](evidence/acceptance.md), and [integration.md](evidence/integration.md).

- [x] **E34 — Complete prepareRename and safe atomic rename**
  - Prerequisites: E30,E31.
  - Owner: Muse editor lane.
  - Files/interfaces: compiler rename/reference/index feature files; rename tests.
  - Changes/traceability: Include exported provides/import occurrences, prepareRename and clear rejection errors. Validate name, visibility, collision/capture and complete fingerprint before versioned atomic edits. Preserve validated shared field semantics. (R07,R10; D4-D5; [design](design.md)).
  - Acceptance: Exported rename updates declaration/header/cross-file uses together; shadowing remains distinct; stale/colliding/capturing proposals produce no partial edits.
  - Evidence: [codex-review.md](evidence/codex-review.md), [acceptance.md](evidence/acceptance.md), and [integration.md](evidence/integration.md).

- [x] **E35 — Expose compiler-validated code actions**
  - Prerequisites: R26,E30.
  - Owner: Muse editor lane with exclusive driver/fixes.go grant.
  - Files/interfaces: compiler/internal/driver/fixes.go; compiler code-action feature files; tests.
  - Changes/traceability: Connect compiler proposals to LSP actions with diagnostic identity/version and UTF-16 edits. Revalidate exact source, target resolution and no new errors; tolerate unchanged unrelated warnings/errors safely. Avoid broad speculative fixes. (R10; D5; [design](design.md)).
  - Acceptance: Missing-arm action appears and resolves its target beside unrelated findings; stale edits and edits changing contracts are refused; unsupported diagnostic has no fabricated action.
  - Evidence: [codex-review.md](evidence/codex-review.md), [acceptance.md](evidence/acceptance.md), and [integration.md](evidence/integration.md).

- [x] **E36 — Add document and workspace symbols**
  - Prerequisites: E30.
  - Owner: Muse editor lane.
  - Files/interfaces: compiler symbol feature files; wire tests.
  - Changes/traceability: Emit hierarchical document symbols with declaration/selection ranges and workspace search across applicable loaded roots. Include current declaration families and record fields; retain trustworthy partial symbols. (R09; D4; [design](design.md)).
  - Acceptance: Function/record/field hierarchy and name-selection ranges are correct, current native forms appear, and workspace queries do not leak unrelated roots.
  - Evidence: [codex-review.md](evidence/codex-review.md), [acceptance.md](evidence/acceptance.md), and [integration.md](evidence/integration.md).

- [x] **E37 — Add semantic tokens from compiler facts**
  - Prerequisites: E30.
  - Owner: Muse editor lane.
  - Files/interfaces: compiler semantic-token feature files; extension semantic contribution metadata via client lane handoff.
  - Changes/traceability: Define supported legend and produce deterministic nonoverlapping UTF-16 token data from lexical/resolved roles. Distinguish declarations/references/types/parameters/properties/functions/namespaces/errors where proven. Handle invalid syntax without misclassifying comments or strings. (R06,R09; D4-D6; [design](design.md)).
  - Acceptance: Protocol legend/data and Unicode delta ordering tests pass; lexical TextMate fallback remains usable while semantic analysis is pending or blocked.
  - Evidence: [codex-review.md](evidence/codex-review.md), [acceptance.md](evidence/acceptance.md), and [integration.md](evidence/integration.md).

- [x] **E38 — Add folding and proven inlay hints**
  - Prerequisites: E30.
  - Owner: Muse editor lane.
  - Files/interfaces: compiler folding/inlay feature files; tests.
  - Changes/traceability: Fold AST and comment/string regions correctly and provide useful checked local-type/parameter hints without duplicates or guessed facts. Respect requested ranges and user settings/capabilities. (R09; D4; [design](design.md)).
  - Acceptance: Nested declaration/comment folds and Unicode inlay positions are correct; hints respect shadowing, no-op settings and incomplete source; no unproven type display.
  - Evidence: [codex-review.md](evidence/codex-review.md), [acceptance.md](evidence/acceptance.md), and [integration.md](evidence/integration.md).

- [x] **E39 — Add safe range and on-type formatting**
  - Prerequisites: P11,R27.
  - Owner: Muse editor lane with formatting-file ownership.
  - Files/interfaces: compiler/current_format.go; compiler/internal/syntax/format_trivia.go as needed; LSP formatting files/tests.
  - Changes/traceability: Reuse canonical trivia formatter, derive isolated edits and validate the exact spliced result. Handle requested range/on-type trigger boundaries, preserve comments and current braces, and decline unsafe isolation rather than widening. (R11; D5; [design](design.md)).
  - Acceptance: Selected range changes only its promised region; on-type indentation/format action is safe and idempotent; comments/CRLF/current constructs preserved; invalid candidate emits no edits.
  - Evidence: [codex-review.md](evidence/codex-review.md), [acceptance.md](evidence/acceptance.md), and [integration.md](evidence/integration.md).

- [x] **S40 — Implement complete analysis fingerprint and bounded cache**
  - Prerequisites: F03,P13,R26.
  - Owner: Muse server lane, serialized with editor lane for shared files.
  - Files/interfaces: compiler server/analysis snapshot/cache files; project input provenance tests.
  - Changes/traceability: Fingerprint all relevant document versions/text and disk manifest/lock/registry/assets/import dependencies. Reuse completed analysis/index across queries, bound cache replacement and invalidate dependent projects correctly. (R05,R14; D3; [design](design.md)).
  - Acceptance: Sibling-only edit and config/dependency/asset disk mutation invalidate results; unchanged features share one analysis; old snapshots are released.
  - Evidence: [codex-review.md](evidence/codex-review.md), [acceptance.md](evidence/acceptance.md), and [integration.md](evidence/integration.md).

- [x] **S41 — Keep protocol intake responsive with one analysis worker**
  - Prerequisites: S40.
  - Owner: Muse server lane.
  - Files/interfaces: compiler/lsp.go dispatcher or extracted server/scheduler files; deterministic tests.
  - Changes/traceability: Separate reading/dispatch from serialized mutable compiler analysis, coalesce edits, check cancellation at stage boundaries and reject stale publication/edits by entire fingerprint. Bound queues; ensure clean shutdown and protocol errors. (R05,R14; D3; [design](design.md)).
  - Acceptance: Deterministic paused-worker test proves intake/cancellation works; rapid edits publish only current results; one heavy analyzer at a time; no races/deadlocks on shutdown.
  - Evidence: [codex-review.md](evidence/codex-review.md), [acceptance.md](evidence/acceptance.md), and [integration.md](evidence/integration.md).

- [x] **S42 — Repair document, configuration and workspace lifecycle**
  - Prerequisites: S41,P12,P13.
  - Owner: Muse server lane.
  - Files/interfaces: compiler LSP sync/workspace/config handlers; project overlay tests.
  - Changes/traceability: Handle didOpen/change/save/close, watched create/change/delete, workspace add/remove and relevant config overlays. Partition publications by owning root and dependencies. Provide no-disk-write untitled scratch support with unambiguous context rules. (R05,R12; D3; [design](design.md)).
  - Acceptance: Two roots retain findings independently; close reloads disk and rechecks siblings; new unsaved source is checked; file/config/dependency deletion clears or relocates findings correctly; scratch queries remain honest.
  - Evidence: [codex-review.md](evidence/codex-review.md), [acceptance.md](evidence/acceptance.md), and [integration.md](evidence/integration.md).

- [x] **S43 — Join capabilities, severity, build identity and protocol validation**
  - Prerequisites: E31,E32,E33,E34,E35,E36,E37,E38,E39,S42.
  - Owner: Muse server lane.
  - Files/interfaces: compiler LSP initialize/publish/request boundaries; wire tests.
  - Changes/traceability: Advertise only implemented capabilities, all severities and exact ranges; provide serverInfo identity and clear unsupported/invalid/stale errors. Validate client coordinate/capability assumptions and protect framing/resource boundaries without arbitrary silent truncation. (R02,R04,R07-R13; D3-D6; [design](design.md)).
  - Acceptance: Initialize/provider requests agree; publication carries URI/version/code/severity/related data; malformed request does not kill service; valid UTF-16 ranges and build identity are observable.
  - Evidence: [codex-review.md](evidence/codex-review.md), [acceptance.md](evidence/acceptance.md), and [integration.md](evidence/integration.md).

- [x] **G50 — Repair comment and string grammar behavior**
  - Prerequisites: F00.
  - Owner: Codex grammar/client lane; independently reviewed.
  - Files/interfaces: editors/vscode/syntaxes/can.tmGrammar.json; tokenizer fixtures.
  - Changes/traceability: Add nested block comments and correct ordinary/raw/triple string boundaries and escape scopes. Stop ordinary unfinished string leakage according to compiler syntax; preserve raw escapes and comment-like string contents. (R06; D6; [design](design.md)).
  - Acceptance: Real tokenizer tests show nested comments entirely comment-scoped, all string styles correct and unfinished ordinary string does not color following declarations as strings.
  - Evidence: 24 client, TextMate/Oniguruma and provenance tests pass (`bun run test`, 2026-09-30); independent grammar/client source review corrections accepted. Host package and actual Cursor activation remain C62/D91/D92.

- [x] **G51 — Align all current syntax scopes with canonical grammar**
  - Prerequisites: G50.
  - Owner: Codex grammar/client lane; independently reviewed.
  - Files/interfaces: editors/vscode/syntaxes/can.tmGrammar.json; syntax coverage fixtures.
  - Changes/traceability: Cover actual operators/numbers, generic/type contexts, contextual declaration/terminal words, names/calls/fields/errors/namespace headers, braces and assertion tags at correct indentation. Remove old misleading spellings/scopes without compatibility scaffolding. (R06; D6; [design](design.md)).
  - Acceptance: Corpus maps current lexer/parser constructs to expected scopes; valid < > & ^ ~ << >> ** .. ... handled; contextual identifiers and comparison/generic ambiguity do not get blanket wrong scopes.
  - Evidence: 24 client, TextMate/Oniguruma and provenance tests pass (`bun run test`, 2026-09-30); independent grammar/client source review corrections accepted. Host package and actual Cursor activation remain C62/D91/D92.

- [x] **G52 — Repair editor indentation, comments and pairing**
  - Prerequisites: G51.
  - Owner: Codex grammar/client lane; independently reviewed.
  - Files/interfaces: editors/vscode/language-configuration.json; package language defaults; config tests.
  - Changes/traceability: Define block comments, brackets/autoclose/surrounding pairs and significant four-space indentation rules. Avoid autoclose inside inappropriate comment/string contexts; add meaningful language defaults. (R06; D6; [design](design.md)).
  - Acceptance: Bracket/comment commands and pairing metadata match current Can; indentation fixtures cover functions/assertions/match/native forms and dedentation.
  - Evidence: 24 client, TextMate/Oniguruma and provenance tests pass (`bun run test`, 2026-09-30); independent grammar/client source review corrections accepted. Host package and actual Cursor activation remain C62/D91/D92.

- [x] **G53 — Replace regex-only confidence with actual tokenizer gates**
  - Prerequisites: G51,G52.
  - Owner: Codex grammar/client lane; independently reviewed.
  - Files/interfaces: editors/vscode/package.json/bun.lock; tokenizer test harness/fixtures; tools/gramcheck; .github/workflows/verifier.yml.
  - Changes/traceability: Use pinned ordinary TextMate/Oniguruma dev dependencies and real ordered scope tests; keep useful compiler corpus parity checks. Wire a working extension test command and CI. No hardcoded installed Cursor paths in project tests. (R06,R14; D6; [design](design.md)).
  - Acceptance: Tokenizer test command passes positive/negative/incomplete corpus and demonstrably fails restored old block-comment/string/operator regressions; CI invokes actual checks.
  - Evidence: 24 client, TextMate/Oniguruma and provenance tests pass (`bun run test`, 2026-09-30); independent grammar/client source review corrections accepted. Host package and actual Cursor activation remain C62/D91/D92.

- [x] **C60 — Make client lifecycle and server identity reliable**
  - Prerequisites: F03,G52,S43.
  - Owner: Codex grammar/client lane; independently reviewed.
  - Files/interfaces: editors/vscode/client/extension.js; client tests; package configuration/commands.
  - Changes/traceability: Await/handle startup and shutdown; expose restart/status/build identity; validate configured or bundled executable existence/platform, restart on serverPath changes, and prevent duplicate clients. Retain library error output and make failures actionable. (R05,R12; D6; [design](design.md)).
  - Acceptance: Mock client tests prove startup/failure/config-change/deactivation ordering, commands work and build identity reflects server handshake; no abandoned language server.
  - Evidence: 24 client, TextMate/Oniguruma and provenance tests pass (`bun run test`, 2026-09-30); independent grammar/client source review corrections accepted. Host package and actual Cursor activation remain C62/D91/D92.

- [x] **C61 — Connect file watches, scratch documents and config assistance**
  - Prerequisites: C60,S42.
  - Owner: Codex grammar/client lane; independently reviewed.
  - Files/interfaces: editors/vscode/client/extension.js; manifest contributions/schema resources if needed; client tests.
  - Changes/traceability: Select file/untitled Can documents and narrowly named Can config JSON files where supported; register source/config/dependency watches and workspace events consistent with server. Reuse canonical config rules for any schema contribution. (R05,R12; D3,D6; [design](design.md)).
  - Acceptance: Lifecycle notifications reach server; scratch does not create files; named config validation works without hijacking arbitrary JSON; multi-root watches clean up on removal.
  - Evidence: 24 client, TextMate/Oniguruma and provenance tests pass (`bun run test`, 2026-09-30); independent grammar/client source review corrections accepted. Host package and actual Cursor activation remain C62/D91/D92.

- [x] **C62 — Add reproducible packaging and activation smoke coverage**
  - Prerequisites: C60,C61,G53,S43.
  - Owner: Muse grammar/client lane.
  - Files/interfaces: editors/vscode build/package scripts and manifest; distribution version entry points only as needed; docs/tests.
  - Changes/traceability: Advance extension version, build host binary with verified source provenance, exclude dev-only payloads, verify VSIX identity/binary/platform and macOS signature. Add activation/protocol smoke harness and current install instructions with automatic task-owned temporary cleanup. Do not install into user Cursor yet. (R12-R14; D6; [design](design.md)).
  - Acceptance: A stale bundled binary cannot silently pass packaging; archive inspection/initialize capabilities and source identity tests pass; host-only artifact generation is bounded and output ownership/expiry recorded.
  - Evidence: [delivery.md](evidence/delivery.md), [release-protocol.json](evidence/release-protocol.json), and lane verification records.

- [x] **I70 — Run integrated multi-diagnostic and editor acceptance suite**
  - Prerequisites: R27,E31,E32,E33,E34,E35,E36,E37,E38,E39,S43,G53,C62.
  - Owner: Muse coordinator after lane release.
  - Files/interfaces: all changed code/tests; evidence/acceptance.md; diagnostic-coverage.md.
  - Changes/traceability: Run bounded relevant source/parser/project/resolve/types/check/driver/LSP/formatter/tokenizer/client tests serialized. Add missing end-to-end oracles for error+warning coexistence, recovery/no-cascade, exact ranges, all lifecycle transitions and all providers. Inspect coverage inventory for hidden first-error exits and fake locations. (R01-R14; acceptance examples 1-11; [design](design.md)).
  - Acceptance: Every design example has actual evidence; known old regression behaviors are tested away; no success claim based only on old passing tests or checklist ticks.
  - Evidence: [delivery.md](evidence/delivery.md), [release-protocol.json](evidence/release-protocol.json), and lane verification records.

- [x] **I71 — Check real project analysis and release readiness without executing code**
  - Prerequisites: I70.
  - Owner: Muse coordinator.
  - Files/interfaces: read-only sample/project inputs; owned temp probes; evidence/acceptance.md.
  - Changes/traceability: Exercise current real Can fixtures and bounded representative project snapshots through the new server, including large-enough dependency shape for correctness but no timing measurements. Check no execution/network effects, output artifact limits, no leaked temporary directories and host packaging readiness. (R01-R05,R13-R14; D1,D6; [design](design.md)).
  - Acceptance: Compact request/response evidence confirms current capabilities, precise findings and clean files; no application source edits or runtime execution; cleanup report lists no failures.
  - Evidence: [delivery.md](evidence/delivery.md), [release-protocol.json](evidence/release-protocol.json), and lane verification records.

- [x] **H75 — Hand off complete implementation with released writers**
  - Prerequisites: I71.
  - Owner: Muse coordinator.
  - Files/interfaces: checklist.md; evidence/handoff.md; native goal state.
  - Changes/traceability: Confirm every Muse task evidence, summarize changed files/decisions/tests/remaining real limits, release all implementation agents/writers and record goal/session IDs. Mark native goal complete only now. Leave Codex tasks below unticked. Do not treat an unfinished requirement as an unrelated follow-up. (R14; review/closure policy; [design](design.md)).
  - Acceptance: Explicit complete handoff and all writers released; evidence paths useful; no unknown active process still editing code.
  - Evidence: [delivery.md](evidence/delivery.md), [release-protocol.json](evidence/release-protocol.json), and lane verification records.

- [x] **V80 — Independently review actual implementation and run required verification**
  - Prerequisites: H75.
  - Owner: Codex.
  - Files/interfaces: actual diff; evidence/codex-review.md; relevant tests.
  - Changes/traceability: Inspect source and tests against every requirement/diagnostic family; independently test material invariants and changes, including strict compile gating and incomplete-source assistance. Use bounded subagent review for disjoint complex areas if useful. Checklist ticks are not proof. (R01-R14; independent review; [design](design.md)).
  - Acceptance: No unresolved material finding; independent checks pass with recorded limits. Any defect maps to concrete unmet task/requirement.
  - Evidence: [codex-review.md](evidence/codex-review.md), [acceptance.md](evidence/acceptance.md), and [integration.md](evidence/integration.md).

- [x] **V81 — Resolve review defects through the same checklist/session**
  - Prerequisites: V80 if findings, otherwise record not needed.
  - Owner: Muse repairs then Codex verifies.
  - Files/interfaces: affected code/tests only; checklist/evidence.
  - Changes/traceability: Resume existing coordinator after verified ownership release with precise repair tasks. Preserve settled design, re-review changed areas only unless a concrete broader risk justifies it. Do not launch competing writers or silently defer scope. (R01-R14; review loop policy; [design](design.md)).
  - Acceptance: Material findings closed by diff/test evidence; required checks pass; review rationale stays compact.
  - Evidence: [codex-review.md](evidence/codex-review.md), [acceptance.md](evidence/acceptance.md), and [integration.md](evidence/integration.md).

- [x] **D90 — Integrate reviewed work safely into the main checkout**
  - Prerequisites: V80,V81.
  - Owner: Codex.
  - Files/interfaces: codex/editor-repair branch; main checkout git state; evidence/integration.md.
  - Changes/traceability: Recheck main checkout current changes/process ownership and safely preserve/integrate unique reviewed branch changes without overwriting user/other work. Account for any new commits or conflicts; use supported coordination/approval only if necessary. Keep deployment source identity bound to final integrated code. (R13-R14; integration policy; [design](design.md)).
  - Acceptance: Main checkout contains intended reviewed repair; other work preserved; conflicts reviewed; relevant post-integration checks justified and passed.
  - Evidence: [codex-review.md](evidence/codex-review.md), [acceptance.md](evidence/acceptance.md), and [integration.md](evidence/integration.md).

- [x] **D91 — Build and install the reviewed host extension in Cursor**
  - Prerequisites: D90.
  - Owner: Codex.
  - Files/interfaces: host compiler/VSIX artifact; ~/.cursor/extensions through authorized install; existing user settings narrow update.
  - Changes/traceability: Use reviewed package workflow, host architecture and signature; verify bundle/source identity and extension version before installation. Install into existing Cursor and reload/restart language server without losing unsaved user work. Correct only obsolete Can error-color scope if still present, preserving all other settings. (R12-R14; D6; [design](design.md)).
  - Acceptance: Installed extension contains final reviewed build and grammar; startup succeeds; temporary packaging artifact reclaimed; exact version/hash and install result recorded.
  - Evidence: [delivery.md](evidence/delivery.md), [release-protocol.json](evidence/release-protocol.json), and lane verification records.

- [ ] **D92 — Verify actual active Cursor behavior and acceptance**
  - Prerequisites: D91.
  - Owner: Codex.
  - Files/interfaces: Cursor process/activation output; controlled temporary or scratch probe; evidence/cursor-verification.md.
  - Changes/traceability: Confirm the actual current Cursor window uses the new server process/build and source grammar. Exercise precise error+warning squiggles and key completion/hover/definition/rename/format actions in a controlled buffer; inspect scope/protocol where useful. Never edit application sources just to provoke errors. (R01-R13; acceptance 11; [design](design.md)).
  - Acceptance: Live server path/build/capabilities match installed source; visible or exact editor diagnostic range evidence confirms correct position/severity; no startup failures or stale prior server remains responsible.
  - Evidence: pending.

- [ ] **C99 — Preserve evidence, clean owned resources, archive worktree and retire monitor**
  - Prerequisites: D92; all writers/processes/viewers released.
  - Owner: Codex.
  - Files/interfaces: monitor.md; owned temp prompt/socket/session resources; managed worktree artifact; heartbeat.
  - Changes/traceability: Preserve compact design/checklist/review/install evidence in main. Remove only recorded task-owned temporary files, terminate only released owned terminal/session resources, archive managed worktree through supported tool after unique work is preserved, and delete/retire matching heartbeat. Report any cleanup/verification failure explicitly. (R14; resource/closure policy; [design](design.md)).
  - Acceptance: No required work remains; unique code/evidence preserved; no leaked owned workspaces/bundles/caches; archive/monitor retirement verified; final user result is accurate.
  - Evidence: pending.


## Final cost/scope instruction

The user requested no new tasks and immediate merge/closure. All source is committed on main through 7fa98c82; documentation is preserved locally in main. Worktree is archived and heartbeat deleted. D92 remains pending one user-triggered Cursor reload; C99 has only the small registered probe cleanup tied to that check. No poster/story work was started.
