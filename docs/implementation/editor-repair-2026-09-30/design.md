# Complete Can editor repair

## Authorization and baseline

The user first requested a full investigation, then explicitly authorized **the entire repair work**, including making the current Cursor instance use the reviewed current extension. This design closes the diagnostic, grammar, language-feature, lifecycle and delivery findings from that investigation. Baseline: `9c283544466876700ef7a079f0bc3136451c0008`. As of the user’s replacement AGENTS instructions on 2026-09-30, Codex owns implementation and coordination as well as independent review, installation/integration verification and closure. The prior Muse coordinator is stopped and its partial changes are retained for review. Historical Muse instructions in the launch artifacts are superseded.

Worktree: `/Users/vince/.codex/worktrees/editor-repair/can-lang`, branch `codex/editor-repair`. Main checkout: `/Users/vince/Projects/can-lang`. Another Muse session is attached to main: implementation must not edit main source or interfere with its process. The Codex monitoring record lives at `/Users/vince/Projects/can-lang/docs/implementation/editor-repair-2026-09-30/monitor.md` and is outside Muse ownership.

## Requirements

| ID | Required outcome and acceptance |
| --- | --- |
| R01 | Publish every independently diagnosable static error/warning in all applicable files, including independent findings after another error; suppress only genuine dependency cascades, with internal blocked reasons. |
| R02 | Attach the smallest explanatory token/expression range, correct original file and version, code, severity and related locations. No known-span whole-line expansion or arbitrary Can line-one fallback. |
| R03 | Share canonical language rules between CLI and editor; compilation rejects any error and never emits a partial invalid program. Old APIs, spellings and accidental goldens are not compatibility requirements. |
| R04 | Preserve warnings with unrelated errors and distinguish errors, warnings, informational notes and hints. Covers syntax, resolve, types, body/assertion contracts, native/HTTP/fetch/SQL contracts and project configuration. |
| R05 | Correct open/change/save/close, overlay-only files, disk create/change/delete, manifest/registry/lock/assets/dependency invalidation, multi-root isolation and cancellation/stale result behavior. Untitled Can buffers have honest scratch support without invented project semantics. |
| R06 | Grammar and language configuration agree with current compiler syntax: nested block comments, raw/ordinary/triple strings and unfinished text, all operators/numbers, contextual declarations/terminals, generics, assertions, braces, indentation and pairing. |
| R07 | Definition/hover work for scoped locals as well as existing symbols; references/rename share binding identity, include provides/import occurrences and preserve shadowing/capture correctness. |
| R08 | Useful completion on unfinished input, with eligible symbols only, precise replacement edits and no suggestions in comments/strings. Signature help reports checked call/constructor/method contracts and correct active argument. |
| R09 | Document/workspace symbols, semantic tokens, folding ranges and useful proven type/parameter inlay hints are implemented and capability-tested. |
| R10 | Expose compiler-proposed validated quick fixes and prepareRename; edits are versioned, atomic and revalidated. Reject stale or unsafe edits with useful reasons. |
| R11 | Preserve canonical trivia-aware formatting; add safe range/on-type formatting that changes only its promised scope and proves the spliced candidate valid. Never widen a selected-range action to a whole-file rewrite. |
| R12 | Client handles startup/failure/restart/configuration, named configuration files and file watches, accurate capabilities, file/untitled documents and visible server build identity. Honor user theme/settings; repair the obsolete .ail-only custom error scope during final Cursor integration if still present. |
| R13 | Reproducible host-targeted extension build/package checks, meaningful version/provenance and automated activation/protocol smoke prevent shipping stale bin/canlc. Install and verify actual active Cursor process and feature behavior. |
| R14 | Bounded correctness verification, independent review, compact evidence, ownership and full cleanup. No benchmarks, broad platform/build matrix, copied private caches or unsolicited runtime changes. |

## Established evidence

- `editors/vscode/client/extension.js:12-35` selects configured/bundled/PATH once and only `file` documents. Installed and local ignored bundled servers both report `1015eb3` (Sept 22), while source is Sept 30. Installed initialize returns only definitionProvider and textDocumentSync. Active Cursor 3.22.12 processes use the installed path. Package is still 0.1.0; installed grammar lacks braces; user error-color customization uses `.ail`.
- `compiler/internal/driver/diagnostics.go:75-111` returns one load/resolve/check error; notes at :121 become warnings. :236 broadens multiline syntax ranges; :282 chooses first loaded file for some unlocated failures. `lsp.go:378-433` close/diagnose creates cross-root clearing and stale sibling issues.
- `syntax.Parse` discards partial AST; Lex/project loading stop at first failure. `resolve.Build` shares mutable global passes; `types.Builder` failure sticks; checker shares bindings/registry/specializations/SQL-site state. Naive continue is unsafe.
- A missing callee currently underlines the whole call (`driver/diagnostics_test.go:148`). Duplicate assertion names can produce unlocated errors. **Correction:** removing the entire required asserts block fails parsing, so it is not the representative semantic first-line reproduction.
- Original-byte/UTF-16 conversion already tests BOM, CRLF, astral and combining characters. Reuse it. Callee/name spans exist; operator AST locations need retention.
- Current formatter uses `FormatTrivia` with comment/current-brace tests; it is not a comment-loss rewrite target. Existing references/completion walkers have reusable scope/identity logic. `driver/fixes.go` already proposes and validates insert-only compiler repairs but the LSP does not expose them.
- A probe using Cursor's actual TextMate/Oniguruma packages showed block comments colored as code, missing << and & operator scopes, string coloring leaking past an unfinished ordinary string, owner treated as a variable and assertion values before => colored as tags. `tools/gramcheck` is only a regex smoke test. Current targeted source/driver/LSP G01-G05/grammar tests passed, proving the old suite is insufficient for the new contract.

## Selected architecture

See [Jev evidence](evidence/jev/decision.md). Three fresh, fully reworded final consultations agreed on canonical recovery and one serialized worker. Independent evidence and tests remain required.

### D1. One authoritative analysis and diagnostic model

Introduce/refactor a compiler-owned analysis result (name is an implementation choice) containing immutable source/config snapshots, complete input fingerprint, partial graph/world, positioned issues, per-unit validity/blocking and a shared semantic index. A unit can be valid, invalid or blocked by identified prerequisites. Preserve exact source bytes and original offsets throughout. Issue fields include source URI/path, byte span or explicit unavailable/project attribution, code, severity, message, related spans and optional compiler fix proposals. Convert offsets to zero-based UTF-16 exactly once at the editor boundary; preserve end line and zero-width insertion diagnostics when correct.

Use one canonical lexer/parser and the existing semantic validators. Recovery belongs at safe lexical, declaration, field/argument/statement, resolver and checked-context boundaries; never synthesize a second editor language. Lexical recovery must make forward progress; an unterminated structure may block that construct but must not silently discard unrelated files. Keep valid declarations and recoverable invalid nodes with their real text/spans. Do not let missing recovered declarations generate unknown-name cascades.

Types and checker state need explicit invalid/error propagation and transactional/per-unit commit. A failed builder or specialization must not contaminate later independent work. Validate dependency components/SCCs in suitable order and record why dependent work is blocked. Preserve trustworthy errors/warnings from clean portions of a partially erroneous graph. A phase-level collector that hides all body diagnostics after an unrelated bad declaration is a milestone, not completion. Do not re-run the whole project once per failure or mask/remove source to discover later errors. Existing API shapes can change; shared wrappers are fine only when useful, not compatibility layers.

Compilation must reject analysis with errors, even if partial typed state exists. Never emit or execute recovered invalid IR. Runtime/assertion execution, network, generated build-only failures and debugger/test-run integration are outside automatic static analysis and outside this repair's feature additions.

### D2. Accurate producer attribution

Maintain a diagnostic-family inventory linking each producer/family to its code, owning source construct, location policy, severity and representative regression. Use the callee/name/argument/member/field/operator/type token that explains the failure, and attach another site as relatedInformation for duplicates or obligation origins. Add operator/qualifier member spans where missing. Do not parse prose messages to guess token occurrences or broaden known ranges. Preserve locations through type/error/assertion/native/HTTP/fetch/SQL metadata and cross-site joins.

Config/lock/registry errors belong to their own actual JSON source/key/value or the configuration path token naming a missing file; share authoritative parsing/validation. When there truly is no source range (e.g. unreadable project root), expose an explicit project diagnostic/status rather than an invented Can token. Valid source errors must not silently use span-unavailable fallback.

### D3. Snapshot lifecycle and serialized work

Protocol input remains responsive; one compiler worker processes immutable snapshots. Coalesce pending edits per affected root and propagate cancellation/stale checks at safe compiler stage boundaries. Bound queue/state retention, never launch concurrent mutations of one compiler state, and test cancellation with deterministic synchronization rather than timing benchmarks.

Snapshot identity is a vector/fingerprint of **every relevant open document version and exact text**, plus manifest, lock, registry, assets, imported source/dependency disk identities. A scalar root version is insufficient. Cache one current analysis/index per applicable project with bounded replacement; discard old snapshots. Invalidate dependent roots on any input change. Publish only if the entire fingerprint still matches. Scope clearing to owned project publications; preserve unrelated roots. Closing an overlay rechecks disk-backed dependents. Existing/new on-disk paths, untitled scratch URI identity, symlink canonicalization and duplicate basenames must be handled intentionally.

Source discovery includes overlay-only .can files under declared source roots; do not persist scratch files. Untitled buffers receive lexical/syntax/locally proven assistance. Use project context only when rooted unambiguously, otherwise decline project semantics honestly. Support relevant configuration overlays/watches without hijacking arbitrary JSON documents.

### D4. Shared semantic editor facts

Extract/reconcile existing refWalker/compWalker logic into one compiler-owned, version-bound occurrence/scope index. Record binding identity, declaration/name/selection ranges, scope, visibility, checked type/signature, call/argument ranges and validity. Queries use this result, not separate ad-hoc name scanners or repeated check passes. Publish no guessed type, import or member where required facts are unavailable.

Definitions/hover cover local declarations and uses, captures, pins, fields, module functions and generated/catalogue contracts. Completion uses recovered current syntax and visible valid symbols; lexical keyword completion remains possible when semantics are blocked. `call combi(` and `shared.` should offer eligible candidates, but unknown imports and strings/comments should not. Use exact replacement text edits. Signature help derives active argument from nested argument spans, not raw comma counting.

References and rename reuse identity, include `provides` and imports and support prepareRename. Return versioned atomic edits; validation prevents capture, collision, invalid spellings, wrong visibility and stale source. Workspace symbols stay within applicable roots. Semantic tokens use recognized lexical/resolved roles, valid UTF-16 delta ordering and no comment/string misclassification. Fold known declaration/body/comment regions. Inlays show only proven useful type/parameter information and honor client requests/configuration.

### D5. Edits and formatting

Expose `SuggestedFixes`/`ValidateFix` through code actions. Revalidation checks version and exact bytes, preserves contracts and resolves the target issue without creating new errors. Existing unrelated warnings must not suppress an otherwise safe fix; with multi-errors, unchanged unrelated findings need not all disappear. No speculative generic repair is presented as compiler-validated.

Retain FormatTrivia and its current-syntax/comment guarantees. Full formatting should depend on trustworthy syntax and safe semantic validation, not discard comments. Range/on-type edits are derived from canonical formatting, restricted to a safely isolatable syntactic/line region, then reparse/recheck the exact spliced candidate. If isolation cannot be proved, return no edit. Respect formatting options where meaningful; Can's required indentation remains canonical.

### D6. Grammar, client and release

TextMate remains the immediate lexical coloring layer; semantic tokens refine it. Add nested block comments; scope ordinary/raw/triple strings with the compiler's actual escape and newline rules; recover ordinary unterminated strings at their legal boundary. Enumerate current operators/numeric forms and current contextual syntax. Prevent generic/comparison confusion and contextual names from becoming global keywords. Function/record/field/call/type/error/namespace roles get standard compatible scopes. Given/asserts tags follow actual indentation and syntax. Add appropriate block-comment, brackets/autoclose/surrounding and 4-space indentation rules without pairing inside comments/strings incorrectly.

Test tokenization through pinned ordinary `vscode-textmate`/`vscode-oniguruma` dependencies (not a hardcoded Cursor application path in repo tests). Keep fixtures for current syntax and broken typing states; integrate grammar and compiler examples in CI. Use editor JSON schema contributions for named Can config files if needed to expose canonical configuration validation; do not duplicate divergent project rules.

Client should advertise only implemented features, await/catch lifecycle operations (respect library errors), restart safely when serverPath changes, expose restart/status/build identity and watch all analysis inputs. Add serverInfo version/build provenance and extension identity; make health errors understandable. Version the repaired extension meaningfully (0.2.0 unless current branch changes require higher); build stamp includes source revision plus dirty/content identity, and package verification proves bundled server matches the packaged source. Test installed-host architecture and macOS signing. Avoid shipping stale ignored bin/canlc or development-only dependencies. No new compatibility policy is required.

## Acceptance examples

1. Two independent bad expressions/declarations/files plus a valid unnecessary-alias warning all publish together. Adding a bad sibling does not erase existing independent findings. Broken prerequisites block only dependents. Repair removes only affected issues.
2. `missing_fn` highlights the callee; wrong argument, member, type, operator and duplicate assertion each select their own token. Duplicate origin is related info. Native/SQL/type/config failures never land in arbitrary Can line one.
3. Exact UTF-16 ranges survive ASCII, BMP, emoji, combining marks, BOM, CRLF, EOF/insertion and multiline diagnostics. All returned edits/ranges index the exact current overlay bytes.
4. Two workspace roots retain separate diagnostics; opening/editing/closing a broken sibling, adding unsaved new source, external create/change/delete and config edits refresh all affected roots and no unrelated root.
5. Rapid changes and cancellation cannot publish an older complete fingerprint. One deterministic concurrency test proves message intake continues while one check is paused, without a benchmark.
6. Local definition/hover respects shadowing and captures. Completion during an unfinished call/member access offers only eligible symbols; invalid imports do not leak suggestions. Nested call signature help counts the correct outer argument.
7. Rename of an exported declaration updates its provides entry and cross-file references atomically; capture/collision/stale proposals fail safely. Compiler missing-arm quick fix works alongside an unrelated warning/error only if it introduces no new error.
8. Actual TextMate scopes cover nested comments, escaped/raw/triple and broken strings, all admitted operators, current contextual/native/error forms, generic/comparison ambiguity, assertions and indentation. Comment text never receives code roles.
9. Document/workspace symbols, semantic tokens, folds and inlays have protocol and source-position tests, including incomplete source with no fabricated semantics.
10. Whole/range/on-type formatting preserves comments and brace syntax; range edits remain inside the promised range and unsafe partial rewrites are declined.
11. Package/activation tests prove correct binary/source identity and capabilities. Final user Cursor process runs the reviewed build; a controlled source probe demonstrates precise error/warning ranges and working semantic operations. Do not edit the user's application source to test it.

## Resource and verification policy

MacBook has 256GB total and ~9.6GiB free at launch. Reuse shared Go/Bun caches; no private full build caches, full source/dependency copies, benchmark runs or broad distribution matrix. At most one heavy compile/test command at a time. Prefer `go test -p 1` selected packages/groups with explicit timeouts, then affected compiler suites as justified by the cross-cutting change. Host compiler and VSIX build are necessary delivery steps and should be built once/reused after code acceptance. Register temporary cleanup on allocation and recover only abandoned task-owned data. Store compact test counts/results/reproduction facts; no unbounded transcripts.

If runtime/ or tools/runtime/ authored TS changes become necessary, run repository-required lint fix, format and check scripts plus relevant tests; don't edit generated catalogue/vendor files manually. Ordinary editor work should not touch runtime implementations. No work in /Users/vince/Projects/manolea-2 except final read-only Cursor verification unless separately authorized.

## Review, integration and closure

Muse owns implementation and checklist progress, not this Codex monitor. Handoff requires all Muse tasks evidenced, all agents/writers released and no silently deferred acceptance. Codex independently reviews the actual diff and tests, repairs material gaps through the same checklist/session, then integrates safely with the main checkout after rechecking its current work/ownership. Never overwrite other changes. Install only reviewed code. Verify actual current Cursor process/build and capabilities, not just presence on disk. Preserve compact plans/evidence, clean owned temporary/session artifacts and archive this managed worktree only after unique work is preserved and no process needs it. Retire the matching heartbeat when everything is complete.
