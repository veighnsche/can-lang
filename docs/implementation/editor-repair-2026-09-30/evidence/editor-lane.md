# Editor feature lane evidence

The LSP builds one `driver.OccurrenceIndex` per immutable `driver.Snapshot` in `lspServer.featureData`. It reuses the existing scope-aware `refWalker` and `compWalker`, caches their output with the snapshot, and evicts it when the snapshot cache replaces that analysis. All feature queries use the same binding IDs. `Analysis.Index` is not the source of editor facts until the walker is moved into the driver; it must remain nil rather than implying that an empty index is complete. The source roles, spans and lexical contexts come from the canonical parser/resolver. Type and signature strings are published only for checked function regions or valid resolved record contracts.

The implemented LSP surface includes local definition/hover/references, recovered completion with exact text edits, checked function/method/constructor/callback signatures, prepareRename and versioned atomic rename, validated code actions, document/workspace symbols, semantic tokens, folding ranges, checked parameter hints, and scope-limited range/on-type formatting. Scratch buffers use in-memory analysis. Source-editing requests decline when two URIs alias one canonical path.

Bounded gates passed:

- `go test -p 1 ./compiler ./compiler/internal/driver -run 'TestG05|TestEditor|TestOccurrenceIndex|TestFixDiagnostic|TestValidateFix' -count=1 -timeout=90s` (compiler 1.434s; driver 0.423s).
- `go test -p 1 ./compiler -run 'TestEditorIncompleteMemberAndCalleeCompletionEdits' -count=1 -timeout=90s` (0.438s).
- `go test -p 1 ./compiler ./compiler/internal/driver -run 'TestEditor|TestOccurrenceIndex|TestFixDiagnostic|TestValidateFix' -count=1 -timeout=90s` after the alias and wire additions (compiler 0.617s; driver 0.679s).
- `go test -p 1 ./compiler ./compiler/internal/syntax ./compiler/internal/driver -run 'TestEditor|TestOccurrenceIndex|TestFixDiagnostic|TestValidateFix|TestCoreConsumerSpecification|TestRecoverySignature' -count=1 -timeout=90s` after reviewer repairs and the sealed signature join (compiler 0.818s; syntax 0.080s; driver 0.554s).
- `go test -p 1 ./compiler ./compiler/internal/syntax -run 'TestEditor|TestCoreConsumerSpecification' -count=1 -timeout=90s` after parser-owned incomplete with-pin value recovery (compiler 0.855s; syntax 0.197s).
- `go test -p 1 ./compiler -run 'TestEditor' -count=1 -timeout=90s` after shadowed-callee rejection and sealed callable input metadata (compiler 0.807s).
- `go test -p 1 ./compiler -run 'TestG02|TestEditor' -count=1 -timeout=90s` after hover precedence repair (compiler 0.949s).

These tests cover scoped local navigation, shadowing/rename, exported `provides`, constructor/method/callback and incomplete-call signatures, incomplete member/callee completion, UTF-16 semantic token ordering, symbols/folds, scratch assistance, positive and disabled hints, range/on-type scope, fix validation beside an independent error, duplicate-URI edit refusal, workspace symbol search without open files, versioned code-action edits, and immutability of returned index and mapped diagnostic slices.

The reviewer repair pass added exact selected-line formatting while preserving other unformatted lines; lexer-backed outer argument separator counting through nested arrays/constructors; exclusion of `near` inputs from positional signature and hints; raw multiline string folds; and semantic highlighting limited to resolved roles so TextMate declaration-specific keyword scopes remain visible. It also rejects unresolved `provides` names as facts, ties code actions to the exact current diagnostic and its range, validates scratch fixes in memory, and uses sealed function signatures for incomplete local input member completion. Complete checked bodies additionally prove explicit nominal local bindings. Constructor, method, and with-pin recovery contexts are parser-owned rather than guessed by a separate scanner.

Signature help now respects visible local shadowing: a noncallable local callee cannot borrow a package function's contract. Callable inputs whose header signature is sealed retain a proven signature when an unrelated body error or incomplete call invalidates the function region; body-local type facts remain region-gated.

The final source review additionally excluded generic instances from source binder promotion and keyed body-local type facts by exact checked declaration spans. These refinements passed the subsequent `TestG02|TestEditor` gate. Resolver stores methods by one package-scoped name, so same-spelled methods on distinct receivers cannot both be valid in a package.

Hover keeps canonical checked function and field contracts, including package provenance. Local binders retain priority over same-named package symbols, and a proven local-receiver field uses the occurrence fallback only when canonical hover cannot resolve it.
