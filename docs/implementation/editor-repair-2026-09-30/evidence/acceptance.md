# Integrated editor acceptance record

This record describes actual bounded checks, with delivery recorded separately. Codex replaced the stopped legacy executor and reviewed the actual source; checklist ticks are not treated as proof. No benchmark or platform build matrix was run, no runtime TypeScript was changed, and no application sources were edited to create errors.

| Design example | Source-level evidence |
| --- | --- |
| 1: independent diagnostics and warnings | driver recovery, expression sibling, same-body warning, transactional rollback, dependency closure and strict-gate tests |
| 2: smallest explanatory ranges | source/syntax/driver location tests; review cases for recursive types, connection metadata, fetch, SQL, assertions, aliases and config JSON paths |
| 3: original byte/UTF-16 positions | BOM, CRLF, emoji, combining marks, EOF and multiline regressions; exact diagnostic and versioned edit assertions |
| 4: lifecycle and roots | LSP snapshot/config/overlay/close/watch, multi-root publication ownership, alias agreement/conflict and dependency asset invalidation tests |
| 5: intake and stale cancellation | deterministic paused-worker cancellation, latest-version publication, bounded retry, exit and shutdown tests |
| 6: local semantic assistance | scoped occurrence index, input/checked local record facts, recovered member/call/constructor/method/with signatures, argument separator regressions |
| 7: safe edits | exported provides/cross-file rename, collision/capture/stale/alias rejection, diagnostic-associated compiler-validated code actions and scratch edits |
| 8: highlighting and grammar | actual TextMate/Oniguruma 24-test client/grammar/provenance run; gramcheck asset/corpus gate |
| 9: semantic products | symbols, semantic tokens, multiline folds and proven inlays; unresolved provides names excluded; lexical keyword colors left to TextMate |
| 10: formatting | canonical formatting, selected-line splice validation with other unformatted lines preserved, partial unsafe declines, on-type scope tests |
| 11: reviewed release | host package/build/provenance and actual Cursor verification pending final source gate; see delivery evidence when complete |

Completed gates before final joins: full source/resolve/types/syntax/project/check/driver suites; repeated checks were limited to concrete review corrections. Latest compiler residual gate passed resolve (0.414s), check (12.911s), driver (10.986s). Editor/signature joined gate passed compiler (0.818s), syntax (0.080s), driver (0.554s); final incomplete with-pin gate passed compiler (0.855s), syntax (0.197s). Bun client/tokenizer/provenance suite passed 24/24; subsequent source-input identity correction passed 3/3 provenance tests. Gramcheck test and real grammar/corpus validation passed.

Final annotation-gather, constructor provenance, local shadowed signature and incomplete callback-input review corrections passed their focused gates and independent review. The final integrated editor/server/config gate passed compiler 2.851s / project 0.074s. Canonical hover regressions found by the first integrated run were repaired, re-reviewed and passed in the rerun. Race and delivery results will be appended after observed completion.
