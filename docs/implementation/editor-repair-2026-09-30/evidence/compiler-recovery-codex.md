# Compiler recovery implementation evidence

Owner: Codex compiler lane. Shared checkout: `/Users/vince/.codex/worktrees/editor-repair/can-lang`. No auxiliary checkout, private build cache, runtime execution, benchmark, or retained temporary directory was created. Root owns the checklist and current acceptance decisions. Root took over JSON/manifest/registry/lock validators; the editor lane owns fixes, hover and occurrence-index implementation.

## Implemented contracts

- `CheckSnapshotContext` loads, resolves and checks once; `CheckSnapshot` is its background wrapper. `CheckScratchSnapshot` builds an in-memory project without filesystem writes or invented imports. Cancellation propagates through source discovery, resolver package phases and checker file/function boundaries, and never becomes a diagnostic result.
- Loader retains partial syntax, canonical overlay-only sources, canonical config overlays (including an absent lock), input bytes and absent input reads. Source confinement precedes substitution. Independent registry, dependency, asset and raw-fixture failures do not erase valid sibling sources. Invalid manifest-component maps supplied by root distinguish blocked imports/SQL from undeclared names.
- Parser/lexer recovery uses original tokens and layout. It preserves healthy declarations, valid steps and signatures, explicit invalid fields/parameters/assertion rows/blocks, and canonical incomplete call/member contexts. Lexer recovery always advances. An unrecoverable nested native form falls back to its declaration boundary; this does not imply native-section or arbitrary nested-expression recovery.
- Resolver marks invalid/ambiguous/blocked symbols, retains independent scopes, reports duplicate origins, and blocks recovered invalid declaration names. Type-builder transactions restore both map membership and previously committed pointed-to node state. Recovering sealing partitions invalid type dependency closures from independently sealed nodes; strict sealing refuses errors.
- Checker collects independent declarations, bodies/statements, assertion rows, native contexts, actions/templates, generic proof components, wrapper components, initializers and SQL sites. Executable-unit checkpoints isolate specialization, binding, callable, request, annotation, site and IR mutations. Failed initializer dependencies and failed function-call/reference closures cannot retain executable regions. Strict public checks still return no emittable Program on error.
- Diagnostic bridge preserves severity, all independent issues, deterministic ordering/deduplication, exact UTF16 ranges and related origins. It has no old first-diagnostic/widen-to-line fallback. Truly unavailable attribution uses empty File/project status. `LocateCode` distributes known enclosing attribution to each joined child without replacing existing child locations. Mixed blocked/direct joins retain direct errors. Editor messages use stable semantic leaves; positional CLI wrappers do not change diagnostic identity after unrelated edits.
- Source coordinates now model bare CR as an editor line break while lexical validation continues rejecting bare CR. No source bytes are rewritten.

## Verified checks

All commands used `go test -p 1`, normal shared build cache, and a 10–90 second per-package timeout. Team members serialized Go processes.

1. Selected syntax/project/resolve/types/check/driver recovery, overlay, recursive type and strict-program gates passed. Resolver had no matching tests in the selected regex; its later full package passed.
2. Full `internal/source`, `project`, `resolve`, `types` passed (0.373/0.343/0.346/0.388 seconds in the full gate). Root independently reran the complete project package after additional JSON changes.
3. Full `internal/syntax` and `check` passed after repairing a real coordination precedence regression (0.293/12.911 seconds). Recovery must recognize `match call concurrent/race` as a coordination step before ordinary terminal classification.
4. Root subsequently ran the full `internal/driver` package successfully (9.522 seconds), before the final producer refinements listed below.
5. `TestRecovery` syntax/driver passed (0.384/0.437 seconds). It checks seven independent exact spans (wrong argument, operator, member, duplicate assertion plus origin, two unknown field types, duplicate field plus origin), registry failure with independent function diagnosis, valid initializer dependency closure, specialization rollback and dependent IR exclusion, overlay-only sources and parser field/input/assertion/body boundaries.
6. `TestCheckSnapshotParseError|TestCheckSnapshotCheckErrorSpan|TestRecoveryNativeAndSQLFamilies` passed (0.530 seconds). The last test checks two invalid native-body literals, two invalid semantic SQL parameter-type strings and one independent missing callee in one snapshot, each at its precise token.
7. Source regression tests passed for multiline/EOF/BOM/CRLF/scalars, bare-CR editor-coordinate round trips, and joined-error attribution. Full driver range tests now call the canonical `errorIssues`/`issueDiagnostic` path.

Tests that assumed exactly one lexical error, whole-expression missing-callee ranges, or arbitrary file/line-one attribution were updated to the intended contract. Valid coordination acceptance and strict semantic tests were preserved. Top-level `callable` remains intentionally invalid: technical-spec.md:259 explicitly excludes it from inert initialization, and the pre-change checker already rejected it.

## Final refinements awaiting the next bounded gate

After the last complete driver run: stable semantic leaf messages; precise forbidden initializer-child spans; `TestRecoveryMessagesStableAfterPrecedingEdit`; connection setting value/name provenance; SQL call-site related descriptor fields; and explicit failed deferred codec/HTTP/form/fetch specialization state with recorded use origins. These are formatted and ready for independent review, but must not be described as fully retested yet.

Proposed gate:

```
go test -p 1 ./internal/source ./internal/check ./internal/driver -run 'Test(LocateJoined|MalformedBareCR|Recovery|CheckSnapshot|SemanticDiagnostic|IssueDiagnostics|Connection|Native|SQL|HTTP|Form|Fetch|Codec|Variant)' -timeout=90s
```

## Review focus and limits

- Review state rollback for any mutable specialization metadata not covered by checkpoints and verify blocked-unit reports agree with actual checked body evidence.
- A nonnil Snapshot.Program is partial evidence, not an emission permit. Editor typed locals require a nonnil verified function Region; strict entry points remain the only emission gate.
- Parser recovery explicitly covers safe field/parameter/assertion/statement boundaries and incomplete call/member context. Other damaged native/nested constructs may invalidate their complete declaration. No claim of arbitrary nested repair is made.
- No assertion that all 1,057 historical bare-error sites were independently exercised is made. Representative family regressions and canonical wrappers are verified; the diagnostic inventory must distinguish those facts from exhaustive producer coverage.
- A focused end-to-end SQL cross-site related-origin assertion and exact native connection-setting regression are advisable for the final refinements; existing complete SQL/native checks establish prior semantic acceptance, not those newly added related spans.

## Follow-up gate and callable-initializer correction

The proposed bounded source/check/driver gate above subsequently passed: source 0.388s, check 3.256s, driver 0.617s. This covers the final producer refinements and `TestRecoveryMessagesStableAfterPrecedingEdit`.

A concrete rename finding required one additional range correction: a forbidden top-level `callable helper` initializer previously highlighted the entire reference. `ReferenceExpr.KeywordSpan` now retains the parser's actual `callable` token, and inert initialization diagnoses that forbidden keyword. The callee name remains available for an otherwise safe semantic rename. `TestRecoveryCallableInitializerKeywordSpan` proves the exact keyword span before/after renaming the callee, stable diagnostic text, and continued strict rejection of the initializer. This regression passed in the same gate. No syntax or initialization rule was relaxed.

## Independent-review corrections and final compiler gate

The subsequent independent review identified concrete gaps; each was repaired in the canonical compiler path, without source masking or project replays:

- Loader lock verification no longer depends on the absence of unrelated errors. Source bytes and manifest digests remain usable after parse failures. Explicit incomplete-source and incomplete-fixture inventory flags gate only unavailable digest evidence. Lock edges and available dependency snapshots accumulate independent failures. Source walking records a path failure and continues trustworthy sibling entries, preserving parsed sources and absent input identities.
- Ambiguous package bindings remain unavailable to lookup. A separate declaration-symbol table gives duplicate function declarations private analysis identities, allowing intact signatures and ordinary concrete bodies to be checked independently. Ambiguous bodies never retain executable regions. Generic declarations still require their existing specialization/proof prerequisites; this does not invent concrete generic bodies.
- Function result/emits/input types and native signature inputs/state now accumulate independently resolvable errors. Connection settings, nested header/metadata entries, and fetch path/query/header/body entries collect independent semantic errors with canonical token spans. A repeated header key has a related first-key origin, while a malformed value on that same repeated entry is independently reported.
- Semantic SQL validation resolves parameter and row types independently and validates statement/dialect prerequisites separately. Each joined leaf is attributed to its own manifest value, rather than the first invalid field.
- Arguments and match arms use child transactions. Failed statements roll back before their invalid local placeholder is installed. Child rollback covers program specialization state, local/use maps, serial allocation, region escape state and aggregate inference state. Aggregate discovery defers a joined error only when every leaf is a discovery obligation; a real sibling error cannot disappear in a mixed join.
- Valid final-local warnings remain visible after an earlier failed statement. Their proof considers the newly introduced lexical identity, without requiring resolved-use evidence from earlier failed syntax.
- Executable validity closes over canonical function and native IR, including expression calls, callable references, invocation steps, judge registrations and wrapper plans. Assertion-invalid producers lose their body/plan, dependent functions and natives become blocked, and attached invalid/dependent assertion roots are removed. This occurs after independent checker findings have been collected. Strict compiler entry points still return no Program on errors.
- `Analysis.Index` is nil unless its builder was explicitly requested. The LSP feature cache remains a separate immutable snapshot projection; sealing does not publish a placeholder empty compiler index.

New `driver/recovery_review_test.go` regressions exercise simultaneous duplicate/body findings, function/native signature fields, connection settings and same-entry duplicate/value faults, paired SQL type values, seven fetch entry faults with duplicate origin, argument children, value/completion arms, same-body warning survival, assertion-invalid function/native callers, native descriptor dependencies and absent-index semantics. Every source fixture also verifies the strict no-emission gate. `check/recovery_transaction_test.go` checks rollback of previously committed local/inference/region state and mixed deferred/direct error preservation. Project regressions prove stale lock + source parse findings and a dangling source symlink + later malformed sibling.

Root independently owns the expression-child collection changes and their seven-case `driver/expression_recovery_test.go`; the full driver gate below includes that suite. Editor owns the later incomplete method/constructor parser work after release of `syntax/expressions.go`.

Validation, all serialized with `go test -p 1` and `-timeout=90s`:

- Full source, resolve and types packages passed: 0.291s / 0.312s / 0.379s.
- Final affected full packages passed: syntax 0.267s, project 0.359s, check 13.406s, driver 10.782s.
- Initial full-gate failures were corrected without relaxing language semantics: formatter AST comparison now excludes the new source-only `KeywordSpan`; the lock test expects the precise missing-edge message; constructor delimiter tests assert a real structured source span instead of requiring obsolete byte-offset prose.

No performance measurements, additional checkouts, private build caches, source copies or persistent temporary artifacts were created. Tests use automatic `t.TempDir` cleanup. This evidence records exercised families, not an unverified claim that all historical diagnostic producers were exhaustively tested. The safe native syntax recovery boundary limitation above remains explicit.

## Final residual producer corrections

A final source review found four specific remaining cases, now fixed and covered:

- A duplicate connection metadata key no longer hides its independently invalid value. Both carry exact token spans, and the repeated key retains the first key as related provenance. Ambiguous/invalid profile metadata does not feed a dependent profile diagnostic.
- Canonical recursive type validation accumulates unknown callable result/input/emits types and unknown generic arguments, including a separately unknown generic head. Each name retains its own origin; the enclosing declaration remains invalid.
- Ordinary value matches check all independent scrutinees transactionally. If any scrutinee lacks a proven type, pattern/arm inference is blocked and no partial match IR is returned.
- Unnecessary-local warnings select the alias name token. The complete binding span is retained separately for replacement provenance. A BOM/Unicode fixture verifies exact UTF16 columns and that warnings alone still pass strict compilation.

`TestRecoveryReviewConnectionMetadataDuplicateAndValue`, `TestRecoveryReviewNestedTypeChildren` (four annotation subcases), `TestRecoveryReviewMatchScrutineeChildren`, and `TestRecoveryReviewAliasWarningToken` cover these cases.

Final affected full suites passed with the same serialized bounded command: resolve 0.414s, check 12.911s, driver 10.986s (`go test -p 1 ./internal/resolve ./internal/check ./internal/driver -timeout=90s`). No source outside this concrete review scope was changed in this follow-up; editor/root ownership stayed unchanged.

## Sealed signature facts and staged ordinary-body annotations

`ProgramFunction.Signature` exposes an independently sealed callable contract for a nonambiguous ordinary declaration. It survives an invalid or incomplete body/assertion, but does not confer executable validity. Invalid headers and ambiguous bindings expose no signature. Generic instances are not conflated with source declaration facts. `recovery_signature_test.go` checks these boundaries and continued strict rejection; editor's shared feature/syntax/driver gate passed with those tests.

Following three fresh parent-run Jev consultations (saved in `evidence/jev-annotations/`), ordinary pre-seal function gathering now retains a successfully gathered header before visiting body/assertion annotations. It aggregates independent annotation failures and records only exact failed AST nodes. Later body checking treats those precise nodes as blocked prerequisites, preventing a second diagnostic or a synthetic "not gathered" cascade, while continuing unrelated statements. Direct gather errors are joined with later assertion/body errors, preserving UnitInvalid status even when later errors are exclusively blocked. Exact-node failures do not apply to concrete generic instances or to post-seal specialization gathering. Other native/generic gather boundaries are unchanged.

Constructor discovery uses the same narrow staging principle: a failed exact constructor AST node is blocked on its later checker visit. Unknown names and synthetic nominal-arity failures point to the constructor name; invalid type arguments retain their own token spans. Synthetic named types now carry actual source name spans. This change is entirely in `check/program.go`; the root-owned expression checker was not modified.

`TestRecoveryStagedBodyAnnotations` proves two independent bad local annotations, optional later missing callee, absence of duplicate/cascade diagnostics, retained sealed input-record signature, invalid producer and blocked dependent regions, strict no-emission, a healthy sibling, and a same-spelling generic annotation that remains valid. `TestRecoveryStagedConstructorGathering` covers unknown constructor, incorrect generic arity and unknown generic argument, each alongside a later missing call, with exact spans and no executable region.

Validation, serialized with the existing shared Go cache and `-p 1 -timeout=90s`:

- Targeted staged/signature driver tests passed (0.708s, followed by constructor gate 0.581s).
- Full `internal/check` and `internal/driver` passed (13.105s / 11.260s).
- Final shared gate `go test -p 1 . ./internal/driver -run 'TestEditorSignatureDeclinesNonCallableLocalShadow|TestRecovery(Staged|Signature)' -timeout=90s` passed (compiler 0.413s / driver 0.617s), including the additional synthetic constructor cases and editor's final shadowed-signature regression.
- `git diff --check` passed for the affected compiler files. No persistent temporary artifacts were created.

Implementation ownership and the Go slot were released to the parent after this gate. The partial-program and native nested-syntax limitations recorded above still apply.
