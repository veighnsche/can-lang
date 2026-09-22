# Language behavior contracts — 2026-09-22

Status: reconciliation index for selected behavior contracts. **The authoritative rules now live in decisions.md and the C/A/Q/P specifications linked below.** This document preserves the B1–B9 navigation and BC acceptance identifiers; it is not a competing addendum or an implementation claim.

The [scope ledger](language-design-dispositions-2026-09-22.md), [acceptance evidence](language-change-acceptance-2026-09-22.md) and [verified implementation gaps](implementation-gap-verification-2026-09-22.md) have separate roles. Native forms, grouped state, explicit contracts and attached assertions remain the baseline. Error-set parameters, changed captures and state-callable redesign remain deferred; LD29 is now specified in [C9.2](../syntax-taste/technical-spec.md#c92-named-runtime-checks) after its separate three-consultation pass.

## B1. Fetch/judge normalization

Selected rules: [A2.4](../syntax-taste/ai-io-spec.md#a24-fetchjudge-normalization). This is the sole detailed contract for this topic.

## B2. Operation wrappers

Selected rules: [A3.2](../syntax-taste/ai-io-spec.md#a32-operation-wrappers). This is the sole detailed contract for this topic.

## B3. Exact generic-error patterns and match order

Selected rules: [C5.1](../syntax-taste/technical-spec.md#c51-exact-generic-error-patterns-and-match-order). This is the sole detailed contract for this topic.

## B4. Standard-failure snapshots

Selected rules: [C9.1](../syntax-taste/technical-spec.md#c91-standard-failure-snapshots). This is the sole detailed contract for this topic.

## B5. Verified build and publication

Selected rules: [P15.1](../syntax-taste/platform-testing-spec.md#p151-verified-build-and-publication). This is the sole detailed contract for this topic.

## B6. Typed fixture reuse with local ownership

Selected rules: [P3.1](../syntax-taste/platform-testing-spec.md#p31-typed-fixture-reuse-with-local-ownership). This is the sole detailed contract for this topic.

## B7. Attached native and wrapper assertions

Selected rules: [P4.1](../syntax-taste/platform-testing-spec.md#p41-attached-native-and-wrapper-assertions). This is the sole detailed contract for this topic.

## B8. Acceptance matrix

The [complete acceptance evidence specification](language-change-acceptance-2026-09-22.md) expands these cases across all 16 accepted changes, including before/after programs, negative cases, fixture ownership and native lowering. BC identifiers remain stable references.

These are required implementation evidence, not claims that new forms compile today. Each row needs positive and negative fixtures plus the stated runtime observation where applicable.

| ID | Case | Required observation |
| --- | --- | --- |
| BC01 | Every reachable raw error; seven detail constructors | One `request_failed` with unchanged typed detail, sanitized fields, new mapped/private original occurrence |
| BC02 | Native decoder and authored handler both emit `codec::invalid_data` | Only native origin normalizes by default |
| BC03 | Body-only versus envelope 404 | Normalized status error versus response success |
| BC04 | Invalid JSON versus valid malformed AI answer; bad later question answer | Codec detail versus `ai::invalid_answer`; no handler runs before full validation |
| BC05 | Fetch/judge argument failure, descriptor helper failure, nested standard fault | Correct outer/inside origins and separate standard propagation |
| BC06 | Base → child → grandchild; omitted rule; selective `inherit` | Last override wins, omission inherits, one operation, finite predecessor delegation |
| BC07 | Handler returns another table's key or calls a failing wrapper | Propagate once, no redispatch/retry; standard handler fault also escapes |
| BC08 | Replace emitting ancestor, delegate on one branch, recover one of several native keys | Correct calculated finite bound and provenance; ordinary caller explicit bound remains enforced |
| BC09 | Illegal target, duplicate key, wrong origin, changed signature, base/effect cycle, misplaced `inherit` | Local diagnostics reject each; no silent inference or fallback |
| BC10 | Two generic specializations in ordinary data/call/chain and each applicable coordination mode | Exact arms discriminate; ambiguous bare, duplicate alias/exact coverage and missing specialization reject |
| BC11 | Race of calls emitting distinct nested `all_failed` values | One outer `all_failed<F>` preserves each nested value/occurrence; insufficient `F` rejects; no container covariance |
| BC12 | Success-first and standard-before/after domain arms | Success-first rejects; both failure orders accepted; data/question patterns unaffected |
| BC13 | Standard snapshot caught and observed in aggregate | Stable same-run occurrence/kind/message; constructor/update/emits/string binder reject; selected-handler fault escapes |
| BC14 | Success after caught harness violation | Root remains failed |
| BC15 | Failing root, CPU loop, pending race, startup crash, drain hang | Finite supervised failure and last-known diagnostics; no production publication |
| BC16 | Full build versus selected assert; all dependency roots | Full graph tested for build; selected assert reports partial and never switches production current |
| BC17 | Source/fixture edit during run, invalid TS, interrupted publish, active reader lease | Input change/validation prevents update; current always complete; old lease remains usable |
| BC18 | Same fixture template at two sites, recursive/concurrent visits, literal/expanded rows | Independent lexical queues and deterministic in-place FIFO, with exact arguments/captures |
| BC19 | Wrong template target/specialization, runtime capture, nested expansion, unused row | Reject or sticky failure with both definition and use locations |
| BC20 | Native raw request mutation, malformed provider response, malformed fixture schema, fake auth absent | Request mismatch versus declared decoder failure versus harness validation versus normalized missing credentials |
| BC21 | Raw fixture on wrapped judge plus handler's own locally mocked I/O | Judge/request/handler execute; nested mock belongs to its own call site/root |
| BC22 | Wrapper policy injection, missing local-key test, changed inherited result | Label policy evidence; missing selection coverage fails; parent tests independent |
| BC23 | Eight-fetch regression, no restructuring | Eight calls unchanged; seven raw errors disappear from ordinary public declarations/arms; typed selective recovery passes |
| BC24 | Native with only pretransport cases, judge with no valid-answer case | Coverage failure; additional negative cases cannot substitute for request/decoder/handler evidence |

### Rejected source examples

These are isolated negative fragments, not one program:

```can
// Wrong: calculated contracts are specific to operation wrappers.
fn int ordinary
    emits calculated

// Wrong: questions have no independent transport-wrapper boundary.
wrap changed_question from likelihood
    emits calculated

// Wrong: inherited state remains a final group for a one-state judge.
call wrapped_judge(email_text)
// Required invocation shape: call wrapped_judge((email_text))

// Wrong when both all_failed<a_failure> and all_failed<b_failure> are in scope.
all_failed => ok 0
// Use two exact specialization arms; one bare head cannot cover both.

// Wrong: an ordinary bound standard catch receives the snapshot.
[_] as str message => ok message
// Required binder shape: [_] as standard_failure failure => ok failure.message

// Wrong: inherit is not a reusable expression or helper call.
int value = inherit

// Wrong: a template argument cannot read a runtime local.
sample: use absent_receipt(requested)
// Use inert scenario data, such as the literal shown in B6.
```

For each rejection, the diagnostic points at the invalid head/input and identifies the required contract. It does not suggest an identity-erasing rewrite such as flattening state or merging error specializations.

## B9. Authority, traceability and implementation boundary

[Selected decisions](../syntax-taste/decisions.md) owns surface/product choices. C owns core types, patterns and standard snapshots; A owns normalized operation boundaries and wrapper policy; Q applies those rules to distinct coordination regions; P owns test/fixture and publication behavior. The B1–B7 links above locate the detailed clauses. Cross-document references incorporate those clauses rather than redefining them.

The [document reconciliation record](../syntax-taste/evidence/2026-09-22/document-reconciliation/README.md) maps removed conflicts to their current owners and records validation. Existing [three-packet behavior consultations](../syntax-taste/evidence/2026-09-22/behavior-contracts/README.md) and the payload disagreement investigation remain the supporting evidence. No existing consultation is relabelled as a fresh call.

These documents select requirements; implementation acceptance remains governed by the [16-change evidence specification](language-change-acceptance-2026-09-22.md). The completed historical implementation task list is not rewritten as if the new designs were implemented.
