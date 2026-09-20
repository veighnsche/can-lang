# Postflight coordination review

This review compares the three raw request/response pairs in this directory with the current contracts in [`technical-spec.md`](../../technical-spec.md), [`coordination-spec.md`](../../coordination-spec.md), and [`platform-testing-spec.md`](../../platform-testing-spec.md). It reassesses the engineering tradeoffs rather than treating repeated classifications as votes. No model API was called for this review.

All three responses identify the resolved model as `jev-1.13.0`, choose the same candidate in each topic, and received factually equivalent retrospective payloads. The responses provide no rationale, and the requests are independent paraphrases rather than repeated identical-wording controls. The probability changes therefore cannot distinguish wording effects from ordinary classifier variability. They also introduce no different runtime fact, added alternative, or knowledge of the current selected specification.

## Aggregate typing

| Round | Selected candidate | Selected probability | Confidence | Largest alternative |
|---|---:|---:|---:|---:|
| 1 | `lazy_expected_type` | 0.95 | 0.92 | `always_contextual_type` 0.05 |
| 2 | `lazy_expected_type` | 0.82 | 0.76 | `always_contextual_type` 0.13; `review_needed` 0.05 |
| 3 | `lazy_expected_type` | 0.98 | 0.98 | `always_contextual_type` 0.02 |

**Conclusion: retain the current contract.** [Q6](../../coordination-spec.md#q6-what-exactly-is-all_failed) implements the candidate precisely: the compiler keeps a finite private `U` when the payload is ignored, and requires one expected named variant `F` when source observes, passes, or forwards `all_failed.failures`. It performs no global variant search. C4 supplies invariant named variants and the permitted `standard_failure` leaf; C9 requires wrappers for incompatible specializations of the same bare generic error kind. The runtime array remains occurrence-preserving and input-ordered even though the type-level leaf set is deduplicated.

Independently of the round 2 decrease, the engineering review must account for a real cost of the selected rule: the compiler carries two representations of the same aggregate boundary, a private erased leaf set for ignored fallbacks and a source-denotable `all_failed<F>` once expected typing is required. `always_contextual_type` would make the IR and diagnostics more uniform, but would force authors to invent a named variant or helper when they intentionally discard every failure. That burden conflicts with the approved bare arm's practical fallback use. `unique_cover_search` remains unstable because an unrelated declaration could change inference.

The current specification closes the points that could otherwise justify `review_needed`: it lists the contexts that supply `F`, defines elementwise injection only at aggregate creation, rejects inconsistent contexts, specifies empty aggregates, preserves full nominal error payloads and standard-failure identity, and rejects ambiguous generic specializations. The 0.82–0.98 range gives no evidence that a particular wording feature caused the change and does not itself identify a missing type rule.

**Concrete blocker:** none. The checker and conformance suite must implement the stated expected-type boundary exactly, but no contract change follows from the new results.

## Resource deadline and process boundary

| Round | Selected candidate | Selected probability | Confidence | Largest alternative |
|---|---:|---:|---:|---:|
| 1 | `external_supervisor_boundary` | 0.98 | 0.98 | `drain_without_host_policy` 0.01; `review_needed` 0.01 |
| 2 | `external_supervisor_boundary` | 0.88 | 0.83 | `drain_without_host_policy` 0.08; `review_needed` 0.04 |
| 3 | `external_supervisor_boundary` | 0.97 | 0.96 | `drain_without_host_policy` 0.03 |

**Conclusion: retain the current contract.** [Q10](../../coordination-spec.md#q10-who-owns-unfinished-calls-resources-and-late-rejections) keeps each early-settled operation's runtime owner, observers, captures, and leases until every participant settles. [P6](../../platform-testing-spec.md#p6-opaque-values-callbacks-and-resource-enforcement) blocks new owners during close, permits existing owners to acquire required subleases, and never closes beneath a live lease. Its close deadline bounds only the caller's wait; expiry returns the declared error while the resource remains closing, owned, and observed. P10 and P12 apply the same rule to servers, pools, and transactions. The root can consequently remain pending forever, and bounded process lifetime belongs to an external supervisor whose kill makes no Can claim of cancellation, rollback, clean drain, or runtime-controlled exit.

The lower round 2 probability warrants rechecking a narrow engineering distinction between the selected candidate and `drain_without_host_policy`, but the response does not show that this distinction caused the change. Their in-language settlement behavior is identical: both preserve leases and allow indefinite drain. They differ in whether the contract explicitly assigns bounded host lifetime to deployment supervision. Stating that boundary is operationally useful because it prevents operators from mistaking a resource-close deadline for a process deadline. It adds no hidden hard stop and no language cancellation behavior. `runtime_hard_stop` remains incompatible with preserving owned work, while `review_needed` is unnecessary because the specs define pending empties, late settlements, deadline failure, root drainage, and externally forced termination separately.

The current exit wording is coherent. A close-deadline failure is remembered for a nonzero runtime-controlled exit if the process eventually reaches one; a permanently live owner can prevent that exit entirely. A supervisor kill may produce a host status, but it is outside Can's cleanup and rollback guarantees. Those statements do not promise both an eventual exit and indefinite lease preservation.

**Concrete blocker:** none. Deployments that require bounded lifetime need a supervisor configuration, but that is the selected operational boundary rather than an unresolved language rule.

## Sequential map lowering

| Round | Selected candidate | Selected probability | Confidence | Largest alternative |
|---|---:|---:|---:|---:|
| 1 | `native_from_async` | 0.96 | 0.96 | handwritten traversal 0.02 |
| 2 | `native_from_async` | 0.94 | 0.91 | `review_needed` 0.05 |
| 3 | `native_from_async` | 0.93 | 0.91 | predecessor promises 0.06 |

**Conclusion: retain the current contract.** [C7](../../technical-spec.md#c7-collection-and-text-catalogue) and [Q11](../../coordination-spec.md#q11-how-do-async-callables-interact-with-native-callback-apis) use native `Array.fromAsync` over the source array's native index iterator. Indices are safe from thenable assimilation; the generated async mapper obtains the corresponding Can value, invokes the callback sequentially, and returns a private success box or rejects with the typed failure tag. `Array.fromAsync` waits before requesting the next index, so a rejection starts no later callback. A final native synchronous `.map` unwraps completed boxes. This uses native traversal and storage while adding only the completion adapter required by Can's async and payload contracts.

The 0.93–0.96 range leaves the categorical selection unchanged but supplies no causal account of the variation. On engineering grounds, the predecessor-chain candidate is behaviorally viable but allocates all downstream promises eagerly and preserves more generated scheduling machinery. A shared handwritten traversal would duplicate native iteration. The only material dependency is pinned-target availability and conformance; the current spec makes absence of `Array.fromAsync` a target-conformance error rather than silently changing scheduling.

**Concrete blocker:** none in the specification. Release conformance still has to cover ordered effects, first-failure stopping, empty input, capture behavior, typed failure tags, and callable-`then` payloads on the packaged Bun target.

## Promise payload shielding

| Round | Selected candidate | Selected probability | Confidence | Largest alternative |
|---|---:|---:|---:|---:|
| 1 | `private_completion_boxes` | 1.00 | 0.99 | none above 0 |
| 2 | `private_completion_boxes` | 0.95 | 0.93 | `review_needed` 0.04; mangling 0.01 |
| 3 | `private_completion_boxes` | 0.97 | 0.96 | `review_needed` 0.02; mangling 0.01 |

**Conclusion: retain the current contract.** [C4](../../technical-spec.md#c4-types-generics-immutable-data-and-callable-compatibility) requires every Can payload visible to native Promise resolution to stay inside a compiler-private non-thenable completion or element box. The box has a null prototype, fixed compiler-owned data fields, and no `then`; generated code unwraps only synchronously and reboxes before another Promise crossing. Q3 applies distinct private tags to domain and standard rejection categories, and C7/Q11 use index iteration and boxed results at collection boundaries. Authored field names, source access, and wire encodings remain unchanged, including legal callable field name `then`.

The small probability movement has no demonstrated cause. A worthwhile engineering recheck is the breadth of the universal Promise-crossing audit, but the responses do not establish that this obligation produced the variation. Field mangling would push Promise concerns into every record access and codec boundary. Reserving `then` would alter the general field grammar and still be a source restriction chosen for a backend hazard. Private boxes localize the adapter at the native assimilation boundary and satisfy the native-reuse requirement.

The contract also covers the common nested cases: rejection tags may retain exact domain payloads because rejection reasons are not assimilated; a domain error later exposed as ordinary data is protected by its enclosing success/element box; fold accumulators, fulfilled race winners, mapper results, and nested adapters are named explicitly. The rule does not depend on application records lacking `then`.

**Concrete blocker:** none. The implementation needs a complete Promise-crossing audit, but the representation rule itself is sufficient and internally consistent.

## Postflight decision

None of the four current contracts should change in response to these distributions. The two larger movements prompted rechecks of genuine, already documented tradeoffs: compiler representation complexity versus author boilerplate for aggregates, and an explicit deployment boundary versus silence about process supervision for resource deadlines. They do not prove that either tradeoff caused the probability variation. The raw payloads remain factually equivalent, all three categorical selections agree, and the current C/Q/P text supplies the detailed rules that the classifier requests intentionally omitted. No new specification blocker was found.
