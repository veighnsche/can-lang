# Round 3 Jev request preflight

Reviewed `request-3.json` against the canonical `../jev-design-2026-09-20/request-1.json`, `problem-ledger.json`, and the independently rewritten round 1 and round 2 requests. This was a payload review only; no Jev or other model API was called.

## Mechanical and shared-state checks

- Model remains exactly `jev-latest`.
- The eight topic keys and eight question keys retain canonical order.
- Every question remains a `choice`, and every criteria key retains its canonical spelling and order, including `review_needed`.
- All six fixed constraints, `purpose`, `evidence_limits`, every fact block, all eight question instructions, and every criteria description are rewritten prose.
- `state.common.snapshot_scope` clearly frames the request as a retrospective replay, says baseline/proposal descriptions are historical rather than present normative claims, and withholds both prior Jev outputs and subsequent engineering selections.
- Technical-only section references are present in the common round format: `A6.1-A6.2, A10.1`; `A6.1, A10.1`; `Q6, Q9, C4/C9`; `Q9-Q10, P6/P10/P12`; `C7, Q11`; `P3-P4, Q2/Q10`; `C4`; and `C4-C7, Q2`.

## Topic-by-topic coverage

### `integer_codec`

Coverage is complete. The request retains arbitrary-precision native `bigint`, the shared fetch/standalone/generated-record codec, exact `JSON.parse` source tokens versus rounded `Number`, native `BigInt` lexical limits, JSON Schema's mathematically integral `1.0` and `1e0`, the historical integer-lexeme restriction and domain failure, every forbidden coercion, native general parsing, exact coefficient/exponent normalization, trailing-zero integrality, and pre-expansion digit/exponent/output limits for `1eHuge`. Both canonical alternatives and the inadequate-evidence outcome retain their tradeoffs.

One over-specific statement was corrected: “digit-only integer strings with an optional sign” became “integer digit strings,” avoiding an unsupported sign rule while leaving the `integer_lexemes` candidate's exact optional-minus policy intact.

### `generation_shapes`

Coverage is complete. The request preserves tagged named record/error variants, finite acyclic values of recursive records, closed `case`/`value` wire objects, finite inhabitation and specialization, the historical required-field/no-extra-field record profile, `bool`/`str`/`int`/`float`, compile-time rejection of variants and recursion, nested `anyOf`, recursive references, the ordinary-object root restriction, independent validation, the demonstrated summary-to-question need, graph-emitter and provider-conformance costs, all excluded capabilities, and the exact budgets 8, 1024, and 65536 plus runtime depth/node/byte bounds.

The bounded candidate was corrected from “acyclic record-and-array profile” to “nonrecursive record-and-array profile.” The former could admit a recursive type merely because a particular value happened to be acyclic; the canonical candidate rejects recursive shapes at the type-profile boundary.

### `aggregate_type`

Coverage is complete. The request retains finite type set `U`, participant domain errors plus opaque `standard_failure`, input ordering and duplicate occurrences, the bare unannotated arm, finite disjoint nominal variants, invariant generics, concrete generic-error identity, `all_failed<F>` with `F[]`, expected-context sources for `F`, the coverage requirement, absence of anonymous unions, compiler-private `U` for ignored payloads, named-helper context, multiple unrelated covers, and wrapper normalization for ambiguous bare generic specializations. All four criteria preserve their original semantics.

“Finite internal collection `U`” was corrected to “finite internal set `U`” so the prose does not turn a compile-time type set into a runtime collection. The `unique_cover_search` first sentence was independently rewritten to remove an exact sentence match with round 2.

### `resource_deadline`

Coverage is complete. The request retains permanently pending native `Promise.race([])` and nonsettling participant promises, lack of loser cancellation, owner/promise/lease retention through settlement, subleases, live-lease protection from close/commit/rollback, the historical graceful-close sequence, declared deadline failure with a still-owned and monitored closing resource, root drain, lack of guaranteed eventual nonzero exit, absence of a universal timeout or Can cancellation syntax, the abandonment semantics of host termination, the external-supervisor boundary, and pure pending work. Each of the three operational policies and `review_needed` preserves its distinct tradeoff.

“Can remain pending forever” was corrected to the definite “stays pending” for `Promise.race([])` and for an individual participant that never settles. This restores the canonical native-settlement fact rather than presenting it as a possibility.

### `sequential_map`

Coverage is complete. The request retains asynchronous effectful callbacks, dense immutable arrays, exactly-once ascending visits, await-before-next sequencing, first-failure stop with no partial result, predecessor-linked native `map` plus `Promise.all`, eager downstream allocation, native `Array.fromAsync` on Bun 1.4.2, the exact `[1,2,3]`, `start1/end1/start2/end2/start3/end3`, `[2,4,6]`, `[1,2]`, and `[]` observations, the TC39 await step, pinned-runtime qualification, thenable shielding, index/boxed-completion adapter possibility, native final unboxing, and the comparator/concurrency exclusions. All lowering candidates and their costs remain intact.

“Must box user payloads” was corrected to “must shield user payloads.” Boxing is one candidate mechanism; the canonical fact requires protection from assimilation without prematurely excluding mangled storage. Two sentences were also rewritten to eliminate round 1 and round 2 sentence reuse while retaining `Bun 1.4.2` and `Array.fromAsync` exactly.

### `fixture_queue`

Coverage is complete. The request retains named assertions on every `fn`, existing `when` selectors, explicit expected arguments, typed completions, repeated selectors, the participant-selector exclusion, root and invocation identities, immutable callable-capture fingerprints, deterministic concurrent and recursive fixtures, argument validation, prohibition on live calls, the historical shared FIFO, reservation and path-ordered release barriers, the one-schedule qualification, separate controlled native-timing coverage, and assertion-scoped owner drainage. The shared canonical FIFO, participant-local FIFO, argument matching, and missing-contract outcomes retain their tradeoffs.

The historical FIFO was clarified as one queue keyed by the root and lexical `when` table. This avoids the possible reading that “root plus lexical table” creates two queues.

### `generic_checking`

Coverage is complete. The request retains explicit generic parameters and every absent constraint form, ordinary parameter operations, nominal record/callable invariance, historical declaration checks, every branch of each reachable concrete specialization, test independence, expanding-specialization rejection, same-type recursion, downstream public instantiation, the limited evidence of assertions, the parametric-only alternative, unchanged syntax, and finite compiler-known callback-error specialization for the closed catalogue. Both policy candidates and the constraint-language fallback preserve their canonical meaning.

No semantic correction was required.

### `promise_payloads`

Coverage is complete. The request retains callable record fields, legal field name `then`, direct and collection returns, JavaScript Promise/`await` thenable assimilation, invocation and nontermination risk, mandatory native Promise reuse, historical private success/domain/standard category tags, the missing universal shield, absence of layout compatibility obligations, preservation of authored access and wire names/values, and record/callable immutability. The private-box, mangled-field, reserved-name, and inadequate-representation outcomes retain their distinct tradeoffs.

The fact block was corrected from saying the historical Q design “wrapped coordination outcomes” to saying private tags distinguished those categories; the canonical material does not establish that all payloads were already protected. The `reserve_then_name` candidate no longer claims the unprovided advantage that it avoids every runtime wrapper.

## Literal identifier and numeric gate

The final payload contains the required spellings and observations, including:

- `Promise.all`, `Promise.allSettled`, `Promise.any`, `Promise.race`, and `Promise.race([])` in the fixed order and applicable topic;
- `all_failed`, `all_failed<F>`, `F[]`, `U`, `F`, and `standard_failure`;
- `choice_arm<T>`, `emits [errors]`, `llm str`, `http::response<T>`, Bun, HTMX, and the no-tools boundary;
- `JSON.parse`, `Number`, `BigInt`, `1.0`, `1e0`, and `1eHuge`;
- `anyOf`, schema limits 8, 1024, and 65536;
- Bun 1.4.2, `Array.fromAsync`, `[1,2,3]`, `start1/end1/start2/end2/start3/end3`, `[2,4,6]`, `[1,2]`, `[]`, and TC39.

No required numeric observation, API name, source identifier, exclusion, option, or tradeoff is omitted after correction. One shared constraint was sharpened from generic float-to-string “conversion” to explicit formatting during that conversion, preserving the approved formatting boundary.

## Wording verdict

Pass. Excluding intentionally stable structural values (`jev-latest`, `choice`, keys, and section references), round 3 shares no complete natural-language sentence with the canonical request, round 1, or round 2. The exact duplicated coordination, questionnaire, Bun capability, callback, and variant-cover sentences found during review were rewritten without changing their technical content.

The corrected request remains a neutral historical presentation. It supplies no earlier model selection, no later adopted design, and no wording that recommends one candidate before Jev classifies it.
