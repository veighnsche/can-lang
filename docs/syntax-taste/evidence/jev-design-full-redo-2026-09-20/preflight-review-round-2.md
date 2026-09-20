# Independent preflight: full-redo round 2

Reviewed by the round-1 author after completing round 1; no TypeSafe API call was made. Sources: canonical original request-1.json and this redo's problem-ledger.json. The revised normative specification and prior model answers were not used as a preferred conclusion. Round 2 was reviewed for semantic equivalence, then compared mechanically with independent rounds 1 and 3 for full-payload wording variation.

## Verdict

PASS after the narrowly scoped corrections below. All shared facts and constraints, eight decisions, and 29 candidate meanings are retained. No alternative was added, removed, reordered, or strengthened to fit a prior outcome. Historical-baseline framing replaces misleading claims about current normative policy. The only added structural field is state.common.snapshot_scope.

## Shared material coverage

- Purpose preserves advisory-only classification, supplied-evidence limits, no research/reasoning explanation, no implementation permission, and an explicit uncertainty/inadequate-options result.
- Constraint 1 preserves immutable Can for AI coding agents, named top-level authored functions, asynchronous execution for all functions, sequential ordinary awaiting, and every exclusion: purity/effect system, anonymous authored functions, foreign-code escape, browser Can target.
- Constraint 2 retains native JavaScript/Bun lowering with contract/immutability adapters only, zero external users, no syntax/ABI/layout/golden compatibility, and simplicity conditional on preserved contracts.
- Constraint 3 retains the four Promise mappings in order, no automatic loser cancellation, one bare all_failed arm, original domain/standard occurrences, duplicates, and input ordering.
- Constraint 4 retains complete explicit finite emits, separate standard failures, named/generic data/error forms and finite variants, and no source union or error-set parameter syntax.
- Constraint 5 retains every approved native form, exact choice_arm<T> emits [errors], llm str, grouped state, named fetch encodings, immutable http::response<T>, explicit float formatting, Bun SSR/upstream HTMX, and tools exclusion.
- Constraint 6 retains authority to change technical defaults/adapters, protected syntax/product boundaries, and no new questionnaire.
- Evidence limits preserve documentation/local-probe scope and advisory non-proof/non-approval status. snapshot_scope explicitly excludes earlier Jev answers and later engineering selections from this historical replay.

## Topic-by-topic equivalence

| Topic | Fact coverage and all candidate meanings |
|---|---|
| integer_codec | Native bigint precision; one fetch/JSON/generated-record codec; JSON.parse original tokens versus rounded Number; BigInt lexical limitations; JSON Schema 1.0/1e0; historical integer-only rejection; four excluded coercions/features; native syntax authority; coefficient/exponent/trailing-zero normalization and pre-expansion resource guards. integer_lexemes retains narrow interoperability, exact_integral_tokens retains exact bounded broader acceptance, review_needed retains inadequate candidates/evidence. |
| generation_shapes | Tagged record/error leaves with case/value; finite acyclic recursive values and finite inhabitation/specialization; narrow closed required-field baseline; compile-time variant/recursion rejection; documented nested anyOf/recursive references with object root; independent validation; simple demonstrated workflow; graph-emitter/conformance cost; all excluded features; unchanged 8/1024/65536 profile budget and runtime bounds. Both scope alternatives and review_needed preserve their original costs and limits. |
| aggregate_type | Finite U with standard_failure; ordered repeated runtime occurrences; bare arm; nominal disjoint leaves/invariant generics/concrete error identity; all_failed<F> and F[] failures; expected argument/binding/forwarding emits inference; F covers U; private ignored payload; typed helpers; multiple unrelated covers; wrapper normalization. Lazy, always-contextual, global unique-cover search, and review_needed remain distinct and equivalent. |
| resource_deadline | Promise.race([])/nonsettling participant behavior; uncancelled losers; owners/promises/leases/subleases; no invalidation on close/commit/rollback; historical drain and declared deadline failure with ownership retained; pending root; no eventual-exit guarantee; no universal timeout/cancellation syntax; hard termination abandons effects without rollback; supervisor alternative; pure pending work invariant. All three policies and review_needed retain the same responsibility boundary. |
| sequential_map | Async effectful callbacks, immutable dense inputs, once/in-order/sequential visits, first failure/no partial result; eager predecessor graph baseline; Bun 1.4.2 and exact probe data; TC39 awaiting; pinned availability; thenable shielding; index/box/final map mechanics; no authored async comparator or concurrent ordinary traversal. All three lowerings retain their native-reuse/allocation/traversal tradeoffs, plus review_needed. |
| fixture_queue | Named fn assertions and existing when structure/repeated selectors; no participant syntax; complete mandatory root/call identities; concurrent/transitive argument-checked no-live fixtures; shared FIFO baseline; canonical reservation/barrier ordering distinct from host races; one schedule/separate timing suite; owner drain/no cross-test leakage. Shared FIFO, path-local replay, argument matching, and review_needed preserve their exact scripts/ambiguity/scheduler tradeoffs. |
| generic_checking | Explicit parameters/no constraint mechanisms; named operators/methods; invariance; declaration-independent plus all-branch concrete checks; expanding-recursion rejection/same-type recursion; public templates/concrete-only assertion evidence; type-independent alternative and concrete wrappers; no syntax change; finite catalogue callback error specialization. Concrete, parametric-only, and review_needed remain equivalent. |
| promise_payloads | Callable record fields including then; returned records/collections; Promise/await assimilation and effects/nontermination; native Promise requirement; historical Q tags/core shielding gap; no layout compatibility; source/wire name/value preservation; immutability/no added properties. Boxes, mangled fields, source-name restriction, and review_needed retain all original mechanisms and limitations without an added performance claim. |

## Exact mechanical checks

- Valid JSON; model remains jev-latest; all eight question types remain choice.
- Topic keys/order, question keys/order, all 29 criterion keys/order, match the canonical payload and ledger. Section references retain their identities; root deliberately shortened aggregate_type to Q6, Q9, C4/C9 and promise_payloads to C4-C7, Q2, removing explanatory prose rather than semantic references.
- Every original explanatory string changed: 53/53. With snapshot_scope, all 54/54 corresponding prose values differ from round 1, and all 54/54 differ from round 3.
- No repeated complete sentence of at least 50 characters was found against the canonical request, round 1, or round 3. Short shared technical phrases and literal identifiers are intentionally retained. This check supplements, rather than proves, semantic review or independence.
- Literal inventory checked: shared JavaScript/Bun/ABI, four Promise APIs, all_failed/emits, choice_arm<T> emits [errors], llm str, http::response<T>, HTMX; topic JSON.parse/Number/BigInt/JSON Schema, 1.0, 1e0, 1eHuge; case/value and bool/str/int/float; anyOf, summary->dynamic question, 8, 1024, 65536; U/F, standard_failure, all_failed<F>, F[] failures; Promise.race([]); Array.fromAsync, Bun 1.4.2, [1,2,3], start1/end1/start2/end2/start3/end3, [2,4,6], [1,2], []; TC39, fn, when, FIFO, then, await, Q, TypeScript.
- Probe facts still include rejection at 2, a maximum of one active callback, and empty-input behavior. Zero external users and all schema counts/limits remain semantically unchanged.

## Exact corrections made during this review

1. `state.common.fixed_constraints.0`

   Before: Their generated implementation is asynchronous, while ordinary source calls wait in sequence.

   After: Every function executes asynchronously underneath, while ordinary source calls wait in sequence.

2. `state.common.fixed_constraints.0`

   Before: purity or effect annotations

   After: a purity/effect system

3. `state.common.evidence_limits`

   Before: come from primary documentation or narrow local experiments

   After: come from documentation or narrow local experiments

4. `state.topics.generation_shapes.facts`

   Before: summary-to-dynamic-question program

   After: summary->dynamic question program

5. `state.topics.generation_shapes.facts`

   Before: The existing provider schema limits are depth 8, 1024 properties, and 65536 bytes

   After: The historical Can generation-profile schema budget is depth 8, 1024 properties, and 65536 bytes

6. `state.topics.sequential_map.facts`

   Before: Authored comparators remain unavailable

   After: Authored async comparators remain unavailable

7. `state.topics.fixture_queue.facts`

   Before: Runtime call identity can also include lexical site

   After: Each runtime call identity includes lexical site

8. `state.topics.promise_payloads.facts`

   Before: Such records, including arrays containing them, can be returned

   After: Such records and collections containing them can be returned

9. `questions.generation_shapes.criteria.reuse_broader_type_graph`

   Before: finite provider-schema and runtime budgets

   After: finite generated-schema and runtime budgets

10. `questions.promise_payloads.criteria.reserve_then_name`

   Before: This removes the assimilation trigger with little runtime machinery, but narrows

   After: This removes the assimilation trigger but narrows

11. `state.topics.promise_payloads.facts`

   Before: Q already wrapped coordination success, domain failure, and standard failure in private tags

   After: Q already distinguished coordination success, domain failure, and standard failure with private tags

12. `state.common.fixed_constraints.2`

   Before: The four established coordination forms lower, in order, to Promise.all, Promise.allSettled, Promise.any, and Promise.race.

   After: Match the fixed coordination headers, respectively, to native Promise.all, Promise.allSettled, Promise.any, and Promise.race.

13. `state.topics.sequential_map.facts`

   Before: Every Can callback is asynchronous and may perform effects.

   After: Effects remain possible in each Can callback, and every callback is asynchronous.

14. `questions.aggregate_type.criteria.unique_cover_search`

   Before: Search declared named variants for covers of U and infer F only when exactly one qualifies.

   After: Choose F by enumerating declared named variants and finding exactly one that covers U.

These corrections restore the original scope or literal identity; none reflects a prior classifier result. No outstanding material omission or semantic change was identified after correction.

Final request-2.json SHA-256: `79f1e147575c20fa726d6fda5397dec65e4897fe0536e2c5dcacdb009b7be3c0`
