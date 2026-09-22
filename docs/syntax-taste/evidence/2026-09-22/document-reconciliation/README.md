# Authoritative document reconciliation — 2026-09-22

The selected decisions and C/A/Q/P specifications now contain the September 22 behavior contracts in place. This is a documentation reconciliation, not compiler implementation or a new implementation task list.

| Contract | Detailed owner | Conflicting rules replaced |
| --- | --- | --- |
| Fetch/judge normalization | [A2.4](../../../ai-io-spec.md#a24-fetchjudge-normalization) | Raw seven-error public bounds, fetch consumer arms, envelope status/decode examples |
| Operation wrappers | [A3.2](../../../ai-io-spec.md#a32-operation-wrappers) | Missing wrapper grammar and stale unapproved-syntax notes; native/emitted origin, inheritance and calculated bounds now explicit |
| Exact generic patterns and final success | [C5.1](../../../technical-spec.md#c51-exact-generic-error-patterns-and-match-order), applied by [Q](../../../coordination-spec.md) | Bare-only heads, blanket rejection of two generic specializations, and success-first completion examples |
| Standard snapshots | [C9.1](../../../technical-spec.md#c91-standard-failure-snapshots) | String-only binder and examples; existing `occurrence_id`, kind, message and private cause retained |
| Verified build/publication | [P15.1](../../../platform-testing-spec.md#p151-verified-build-and-publication) | Unspecified assertion/publication gate, deadlines and dependency fixture identity; P2 lock schema includes `fixtures_sha256` |
| Fixture reuse | [P3.1](../../../platform-testing-spec.md#p31-typed-fixture-reuse-with-local-ownership) | Missing exact-target inert templates; lexical queues, capture/argument checking and source-relative ownership retained |
| Native testing | [P4.1](../../../platform-testing-spec.md#p41-attached-native-and-wrapper-assertions) | Function-only assertion assumptions, external-conformance-only raw fixtures, five-label reports and missing native assertion requirements |

[Selected decisions](../../../decisions.md) incorporates these owners and the [scope ledger](../../../../implementation/language-design-dispositions-2026-09-22.md). The [behavior document](../../../../implementation/language-behavior-contracts-2026-09-22.md) retains B1–B9 links and all 24 BC cases as navigation; it no longer repeats an overriding normative addendum. The original design-revisions discussion is explicitly historical. Current implementation notes remain descriptive, with links to selected requirements and verified gaps.

The [acceptance specification](../../../../implementation/language-change-acceptance-2026-09-22.md) still covers all 16 accepted changes in seven evidence dimensions. All 49 dispositions retain their statuses: 16 implement, 18 retain, 15 defer. LD29 was design-gated at this reconciliation; the subsequent [fresh consultation pass](../ld29-checks/README.md) closes it through C9.2 without new standard-expectation grammar. Error-set parameters, changed captures and state-callable redesign remain behind the demonstrated-need gate. Native AI forms, grouped state, explicit contracts and attached assertions remain the baseline.

## Existing evidence and decision boundary

This pass reuses the [three saved behavior consultation packets](../behavior-contracts/README.md), their [pre-dispatch equivalence audit](../behavior-contracts/pre-dispatch-audit.md) and [payload disagreement investigation](../behavior-contracts/disagreement-investigation.md). Those nine choices already select typed payload reuse, origin tables, calculated bounds, exact patterns, snapshots, all-root publication, supervised deadlines, lexical templates and attached native tests. Earlier [identity](../identity-review/README.md) and [full-review](../full-language-review/README.md) evidence continues to support the scope decisions. Agreement is advisory, not proof.

No genuinely new difficult decision was selected during reconciliation, so no additional Jev requests were made. Differences resolved here were stale text, ownership and examples, including retaining the already selected `occurrence_id` field. LD29 was not guessed into a contract during reconciliation; its later selection has its own fresh evidence linked above. No request or response was rewritten or relabelled as fresh evidence.

## Validation and limits

Run `python3 docs/syntax-taste/evidence/2026-09-22/document-reconciliation/validate.py`, plus the existing behavior-contract and acceptance-contract validators. The machine reports record local link/anchor integrity, seven migrated contracts, unchanged clause content apart from references/current-rule wording, prohibited stale rules, 49 disposition statuses, 24 BC cases, 16 acceptance records, and saved consultation hashes. The reconciliation validator inspects documentation, not admitted Can syntax.

Native declaration excerpts now identify their omitted mandatory assertions/raw files. Completion examples put failure arms before final success without changing coordination ownership. Compiler/runtime checks were not rerun for this documentation-only change. Executable acceptance remains future work under the acceptance specification; earlier targeted results remain in the verification report.

[behavior-contracts-before.md](behavior-contracts-before.md) is a frozen pre-migration evidence snapshot, not authority. Its relative links retain their original document context. [migration-map.json](migration-map.json) maps B identifiers to canonical locations; [validation.json](validation.json) records the current documentation check.
