# Full Can language design review — 2026-09-22

> Selected behavior designs: [final contracts and acceptance cases](../implementation/language-behavior-contracts-2026-09-22.md). Use those rules instead of the unresolved behavior sketches below.

> Current scope: [finding dispositions](../implementation/language-design-dispositions-2026-09-22.md). That ledger supersedes the recommendation status and suggested sequence below; proposed examples remain sketches, including retained or deferred alternatives.

Status: recommendations for discussion, not adopted language rules or implementation authorization. This review preserves the three input documents and changes no compiler, runtime or admitted fixture. Every proposed source form below is a sketch unless explicitly described as current syntax.

Inputs: [redundancy review](redundancy-review-2026-09-22.md), [design review](design-review-2026-09-22.md), and [confirmed design revisions](../implementation/design-revisions.md). Evidence includes the four specifications, current implementation guides, checker/runtime code, the four applications, and focused regression fixtures. Three independent reviews are saved in the [evidence directory](evidence/2026-09-22/full-language-review/). Current implementation evidence takes precedence over old planning/status prose.

For concrete code comparisons, see [Before and after](language-design-before-after-2026-09-22.md).

## Assessment

Can has a coherent foundation: immutable nominal data, explicit domain-error contracts, executable examples attached to functions, native Bun behavior, and a controlled boundary to external systems. Keep that foundation. The largest design problem is that implementation details repeatedly become the caller's responsibility: transport taxonomies leak through every abstraction, grouped state prevents ordinary callable reuse, and user-defined higher-order functions cannot express the error relationships available to built-ins.

Reduce those costs by improving abstraction boundaries. Merely reducing the keyword count would not address them. Several proposed unifications would discard useful distinctions: a question is not a network request; a static choice arm is not just an ordinary closure; recovering one concurrent participant is not the same as recovering the whole operation; a checked native signature is not a complete host adapter.

The first changes should be a precise assertion-verified build guarantee, assertion liveness, fetch/judge normalization, structured standard failures, and precise handling of generic error specializations. Catalogue extensibility does not need to precede those changes. State/callable simplification and finite error-set polymorphism merit separate prototypes. No compatibility layer is warranted.

## Constraints carried forward

- Normalize the seven standard transport/codec errors in a language-owned fetch/judge wrapper. Application authors must not reconstruct that normalization themselves.
- Wrap the judge, never each `noul`, `choice` or `score`. Preserve one request, validate-all, then source-ordered handlers.
- Derived wrappers inherit omitted handling; the last override wins. Derived source should state additions and overrides only.
- Put errors before the final `ok` arm in success/error matches. Compiler enforcement is a recommendation here; the ordering itself is already required.
- Evaluate normalization against the eight-call fetch fixture. Reducing nesting is not an acceptance criterion.
- Preserve native JavaScript/Bun operations with only the adapters necessary for Can contracts and immutability.

The grouped-state spelling, primitive-first question headers and closed platform boundary were previously selected. Alternatives to those are explicitly new proposals, not interpretations that silently override earlier choices.

## 1. Make normalization a real public contract boundary

**Priority: first language redesign.** A named error set is insufficient: it abbreviates seven names but leaves callers handling seven outcomes. `emits [=callee]` has the same limitation and also couples a caller's public API to callee implementation changes.

Recommend a distribution-owned error, tentatively:

```text
// Proposed catalogue declarations; ID allocation is omitted from this sketch.
variant http::failure_detail
    http::invalid_request
    http::credentials_missing
    http::transport_failed
    http::timeout
    http::body_limit
    http::status_error
    codec::invalid_data

error http::request_failed(http::failure_detail detail)
```

Unlike a global variant of *all possible failures*, this union has a deliberately finite job: represent these seven infrastructure failures. Project errors remain independently typed.

Every fetch/judge receives a built-in default normalization policy. The checker computes its raw native obligations internally and exposes the normalized public signature. An ordinary declaration and caller can explicitly write `emits [http::request_failed]`; no author-written base wrapper or seven-member declaration is needed. Additional AI or application errors remain in the public bound unless explicitly handled. A fetch mode only maps its reachable intrinsic subset: envelope fetch, for example, does not intrinsically generate `http::status_error`. Existing examples sometimes overdeclare that error; an upper bound does not prove reachability.

Preserve all existing typed details:

| Original payload | Detail retained inside `http::request_failed` |
| --- | --- |
| `invalid_request` | `reason` |
| `credentials_missing` | environment variable name, never its value |
| `transport_failed` | `phase` |
| `timeout` | `timeout_ms` |
| `body_limit` | `limit` |
| `status_error` | `status` and the existing normalized header snapshot |
| `codec::invalid_data` | `path` and `reason` |

The consultations split between reusing the original typed payloads and introducing seven dedicated public detail records with the same fields. Both satisfy normalization. Prefer reuse initially because Can already admits error payloads as ordinary nominal data, and no different public detail contract has been demonstrated. Dedicated records become useful if a reviewed public contract intentionally differs from the raw adapter shape; the choice is not settled by majority agreement.

Keep the original failure occurrence as a private diagnostic cause. Existing storage of a cause does not mean the current terminal reporter displays its chain; any new presentation needs separate diagnostic work. A new normalized occurrence records the mapping boundary; forwarding it preserves its identity. Preserve the existing exposure rules for headers and diagnostics; do not add response bodies, credentials or arbitrary native exception objects to the public payload.

**The conversion must be phase-aware.** A native decoder failure and a question handler explicitly returning `codec::invalid_data` can have the same nominal type. Default normalization covers the native request preparation/transport/codec phases, not every occurrence with one of those names. At a judge, classify the native result before dispatching any question handler; preserve whole-batch validation and handler order. Handler-origin errors and standard failures propagate through their existing channels. A blanket outer catch keyed only by type would be wrong.

A caller handles the one public error and inspects its typed detail only when selective recovery is useful. Status 404 can be handled differently from 429; timeout and invalid JSON remain distinguishable. Normalization adds no network attempt, automatic retry, cancellation, or rollback. LLM and SQL normalization are separate design decisions; this review does not silently include them.

Evidence: [fetch declarations and caller](../../compiler/testdata/current/fetch/main.can), [native intrinsic-bound checking](../../compiler/internal/check/native_program.go), [AI/I/O error contracts](ai-io-spec.md), [domain occurrence representation](../../runtime/domain.ts), [AI review](evidence/2026-09-22/full-language-review/ai-review.md).

## 2. Define wrapper derivation as one resolved policy, not nested catches

Use three distinct concepts: `connection` selects transport configuration; an operation wrapper changes a fetch/judge's failure handling; an ordinary `fn` adapter currently changes calling convention. They should not share an ambiguous meaning of “wrapper.”

Recommend one base per wrapper initially. A derived wrapper inherits the target's input/result signature and configuration, then declares only changed handling. A possible source shape is:

```text
// Proposed syntax. load_json is an existing fetch returning receipt.
wrap cached_load from load_json
    handles native
        http::status_error => match http::status_error.status
            404 => ok receipt(0)
            _ => inherit
```

`inherit` is a proposed handler action: invoke the overridden handler for this same failure. It is not redispatch through the complete wrapper. Without such delegation, a selective override must duplicate its parent's fallback policy, defeating inheritance.

Specify these semantics before selecting final grammar:

1. Resolve the base chain and reject cycles. Targets are fetch, judge, or their derived wrappers, not individual question declarations or arbitrary functions.
2. A handler key contains the exact error specialization and its origin category. `handles native` addresses the native lifecycle; a separate explicit emitted-error category would be required to override application-handler outcomes. Never infer the origin solely from an error's spelling.
3. Flatten the chain before invocation. The most-derived definition for a key replaces the base definition; omitted keys remain inherited. Reject duplicate keys within one declaration. There is no multiple-base linearization to memorize.
4. Invoke the underlying operation once. Dispatch a failure once through the resolved policy. Returning a new failure from a selected handler propagates outward; it does not trigger another rule in that table.
5. Recovery must return the inherited success type. Type-changing wrappers, changed input signatures, retries and multi-base composition are outside this first mechanism.
6. Explicit `inherit` invokes the previous implementation for that key, including its declared output errors. It can walk a finite ancestor chain but cannot restart the operation.
7. Derive the outward error bound from reachable passthrough outcomes and **effective** handlers, including their inherited delegation. Fully overridden parent-handler errors do not remain merely because they existed in a base.

For a finite raw bound `R` and resolved handler table `H`, the escaping domain set is the union of unhandled outcomes and the output bounds of selected handlers. Native-origin and authored-origin occurrences must remain distinguishable during this calculation. A handler's internal selective branches must cover their input normally; if one branch delegates, include the inherited handler's possible failures.

For wrapper handlers, collect a conservative finite escape bound from declared callees and explicit completions. Count syntactically admitted branches rather than attempting semantic reachability proofs; `inherit` contributes its predecessor handler’s bound. This narrow handler rule does not introduce whole-function effect inference.

A derived declaration can omit the inherited `emits` spelling because its contract is computed from its explicit target and checked handler escapes, then published by inspection/tooling. Ordinary functions still declare their public bounds. An implementation should report why each escaping error exists: passthrough, new handler, or inherited delegation.

**Acceptance examples:** override status handling in base → child → grandchild and observe the grandchild's result; omit timeout handling and retain the base; recover only 404 and delegate 429; replace an error-emitting base handler with unconditional recovery and remove its old outward error; propagate a handler's arithmetic failure without redispatch. Test origin collisions explicitly.

## 3. Quantify the fetch improvement without changing control flow

The [measurement script](evidence/2026-09-22/full-language-review/measure.py) counts the eight fetch declarations plus `main`, excluding the separate callable-reference test beginning at line 169.

| Acceptance measure | Current | Proposed default normalization |
| --- | ---: | ---: |
| Infrastructure entries across nine `emits` lists | 63 | 9 |
| Infrastructure forwarding arms across eight calls | 56 | 8 |
| Fetch operations performed | 8 | 8 |
| Typed diagnostic distinctions retained | 7 | 7 |

That is an 85.71% reduction in these repeated entries/arms. It is a source-structure projection, not a compiled prototype or tokenizer measurement. The [eight-call sketch](evidence/2026-09-22/full-language-review/eight-fetch-proposal.can.txt) keeps the operations, success checks and nesting, substitutes the one public error, and places errors before `ok`. New catalogue types and boundary semantics would still need implementation.

Two test layers are necessary. Consumer `when` fixtures supply the normalized public error or recovered success. Raw transport/provider fixtures exercise the real decoder and wrapper, covering all seven applicable failures and proving detail/cause preservation. A supplied completion alone does not test normalization. Derived-wrapper tests must reach the underlying failure, rather than substituting the derived call's final result.

## 4. Define the build guarantee and bound every assertion

**Priority: immediate liveness fix.** `race with error` over an empty spread intentionally remains pending, and recursion is permitted without a termination proof. The current [assertion runner](../../runtime/assert/runner.ts) awaits roots without a deadline. The assertion command can therefore wait indefinitely without a final report.

There is also a concrete documentation/implementation mismatch beyond the earlier reviews: the root README says running assertions is part of every build, but [build dispatch](../../compiler/internal/driver/commands.go) calls `publishProgram(..., false)`. Its validation checks generated output without executing authored code, and [the separate assert command](../../compiler/internal/driver/assert.go) publishes assertion modules before executing them. These source paths do not provide assertion-gated production publication.

Recommend making the advertised build guarantee precise: stage output, execute the required assertion roots against that exact source/runtime identity, and publish the production artifact only after they pass. If compile-only publication is intentionally the product, amend the promise instead; do not call it assertion-verified. This review recommends the former, consistent with the stated contract-first build thesis. Record the verified artifact digest and evidence classes so supplied completions are not mistaken for native/provider coverage. This is a proposed integration change, not current behavior.

Define a finite default assertion budget with an explicit CLI/project override. Enforce a hard limit from the launcher or worker supervisor, outside the executing assertion, so both unresolved promises and a blocked event loop are bounded. A soft in-process diagnostic deadline can first collect pending invocation/frame/owner information; it must not be the only enforcement mechanism. The current suite runs roots sequentially in one process. A per-root budget therefore needs supervisor-visible root-start/checkpoint events and a rearmed hard timer, or isolated root workers. If killing the shared worker stops subsequent roots, mark those roots unrun; do not silently report an incomplete suite as complete. Bound initialization and the suite as well.

Report the assertion identity, elapsed limit, last known phase and pending paths where available. If a hard kill prevents a fresh snapshot, say that the last checkpoint is stale. Fail the assertion command; under the proposed assertion-gated build, fail publication and preserve previously published production output. A killed assertion process carries no promise of completed cleanup. Production `Promise.race([])` semantics need not change, and a test timeout must not quietly add production cancellation.

Acceptance: a failed assertion must prevent publication of its proposed production artifact. Actual emitted Can assertions with an empty race, a never-settling owned participant and a long-running computation must terminate within the configured harness bound and yield intelligible reports. This is stronger than testing a JavaScript timer in isolation.

## 5. Expose `standard_failure` in ordinary catches

The current asymmetry is real: ordinary `[_]` binds `str`, while aggregates expose an opaque snapshot with `kind`, `message` and `occurrence_id`. Reuse that snapshot:

```text
// Proposed bound catch form; unbound [_] remains useful.
match call work()
    domain::unavailable
    [_] as standard_failure failure => relay call recover(failure)
    ok result value => ok value
```

Here `work` and `recover` are illustrative named functions returning the same nominal `result` type. The proposed change is the binding type; invocation still uses ordinary `relay call`.

Jev also split between retaining strings and exposing snapshots. Source inspection confirms that snapshot storage and safe projections already exist; exposing those same observations in ordinary catches adds no new catchability. That is the concrete reason to recommend consistency despite the split.

Keep standard failures outside domain `emits`, preserve automatic propagation, and retain private causes and current handler ownership. Exposing a kind does not promise recovery from fatal termination. Do not turn all possible arithmetic, bounds and native exceptions into a mandatory effect list; that would make `emits` much noisier without establishing total safety.

The current `kind` projection is `str` with a specified finite vocabulary; this proposal does not invent a closed enum or expose the native exception. Use the same snapshot identity in ordinary catches and aggregate observations. Verify that it cannot be constructed or updated, and that a handler's new failure escapes instead of being caught by its own arm. This is a small consistency improvement with direct diagnostic value.

Evidence: [ordinary catch checking](../../compiler/internal/check/completion_matches.go), [standard failure observations](coordination-spec.md), [core review](evidence/2026-09-22/full-language-review/core-review.md).

## 6. Fix generic error discrimination before changing aggregates

Two costs are currently conflated: invariant containers require explicit widening, and bare error patterns cannot distinguish concrete specializations. The [aggregate composition fixture](../../compiler/testdata/current/coordination/aggregate-composition.can) rebuilds aggregates, but [completion checking](../../compiler/internal/check/completion_matches.go) rejects type arguments on error arms and [error-bound checking](../../compiler/internal/check/errors.go) then rejects ambiguous bare names.

Allow exact specialized patterns wherever an explicit declared error bound is dispatched—including ordinary completion matches, `race with error` shared arms and `concurrent with error` per-entry arms—for example proposed `all_failed<a_failure> => ...` and `all_failed<b_failure> => ...`. Coverage and dispatch use the full nominal specialization. A bare name is acceptable only where one specialization is determined. This lets a consumer inspect two aggregate types without rebuilding either.

When a function actually promises one common aggregate type, retain explicit conversion. Prefer an ordinary element-level widening function plus native-backed `.map` over recursive array rebuilding. This is a source rewrite to verify, not evidence that implicit array covariance is required. Preserve every failure payload, input ordering, duplicate occurrence and standard snapshot.

Reject R7's distribution-closed variant of all failure kinds: application-defined errors cannot fit it. Its alternative `race_outcome<success>` sketch also admits primitives/arrays directly as variant leaves, which current Can forbids; wrapping those in records fixes syntax but not the missing open error payload contract. Furthermore, a plain race may ignore its failure payload without declaring a fresh variant. “Every race needs a bespoke variant” is too broad.

Keep plain `race`’s one implicit aggregate arm, the existing first-success contract and `all_failed<F>` while testing this narrower improvement. Do not extend implicit race aggregation and ordinary calls returning existing aggregates as though they were the same typing operation.

## 7. Keep the four coordination meanings; explain ownership better

| Current mode | Native selection | Meaning of recovery |
| --- | --- | --- |
| `concurrent` | `Promise.all` | One participant failure can determine failure of the whole result |
| `concurrent with error` | `Promise.allSettled` | Each participant outcome is available for its mapping |
| `race` | `Promise.any` | First success, otherwise one ordered aggregate |
| `race with error` | `Promise.race` | First settlement, whether success or failure |

R4's common arm layout does not eliminate these semantic differences. A per-entry recovery in `Promise.any` can create a success that wins; a whole-result fallback for `Promise.all` has type `T[]`, whereas participant recovery returns `T`. Treating them as layout variations changes behavior.

Keep explicit regions and improve diagnostics: identify whether an arm completes a participant, a whole coordination result, or the enclosing function. Apply errors-first/`ok`-last wherever those arms coexist. Preserve preparation-before-launch, native selection, once-only evaluation, protected promise payloads, retained loser ownership, and the rule that selected-handler failures are not redispatched.

Do not remove `match chain` based on nesting in one application. Current codec and native-question fixtures use it. It is terminal and has concrete binding/fixture limits that should be measured with an ordinary workflow. Any local binding or early-return extension is a separate control-flow proposal, outside fetch normalization.

## 8. Make native callables ordinary without hiding state serialization

The grouped-state source calling convention is not represented in ordinary callable contracts, and the checker therefore rejects judge/LLM references. The emitted TypeScript inputs are already flat; this is a source/type-contract restriction, not a required backend ABI limitation. That forces [forwarding functions in the AI example](../../examples/native-ai/src/oracles/oracles.can), increasing effects and fixtures simply to adapt argument shape.

Compare two prototypes: preserve grouped-state source calls and represent the group in callable contracts, or use one positional parameter list with explicit per-parameter serialization markers. The second could look like:

```text
// Proposed replacement, not current grammar.
llm records::triage_plan draft from generator
    emits [/* its public errors */]
    given
        state str account
    asks "Draft"

// Calls and references use the ordinary input signature.
call draft(account)
callable draft
```

The marker owns serialization only. The field name still comes from the declaration, an unmarked argument is not sent as state, and the callable signature uses ordinary positional input types. Judge shared-state preparation remains one operation before batching. Empty state produces an empty object without a special nested argument group.

Jev’s three consultations split on this choice: two preferred retaining grouped inputs in callable contracts, while one preferred explicit markers with low confidence. Keep both as prototype candidates without declaring a winner. Require each to pass an ordinary `callable draft` through a higher-order operation without an author-written forwarder, with mixed ordinary/state inputs and empty state. Compare the real AI wrappers, state fixtures and receiver/capture behavior before selecting either.

This changes an earlier selected spelling and needs an explicit design decision. The alternative is extending callable types to represent grouped inputs everywhere; that preserves the call syntax but spreads a special case into higher-order operations. Do not implicitly serialize “the last record argument”: positional coincidence is a poor contract for what leaves the process.

This redesign is independent of error normalization. Implementing it must not create individually wrapped or independently network-executing questions.

## 9. Close the built-in versus authored higher-order gap selectively

Can's built-in `.map` can preserve an arbitrary finite callback error set. An authored combinator cannot express that same relationship; it must choose a fixed error bound. This is a deliberate expressiveness boundary, not a compiler defect. Named error sets abbreviate fixed bounds and do not remove it.

If authored traversal/composition helpers are intended, prototype explicit finite error-set parameters, separate from data type parameters. A conceptual signature is `callback: callable U(T) emits E` and outward `emits E`; this is mathematical notation, not selected source syntax.

Keep the solver finite: exact nominal error identities; set union; subset compatibility; subtraction only after complete exact handling; add any errors emitted by handlers. Solve variables from explicitly typed callbacks and expected signatures, not whole-body inference. Reject unresolved or expanding sets. Standard failures remain outside these sets. Generic data specialization continues independently.

The useful test is one authored combinator that accepts callbacks with two different error sets while preserving each caller's precise public contract. If it cannot substantially improve such a program, retain the fixed-bound limitation and document it. Do not add general inferred effects or a universal untyped failure to make the example pass.

Named domain-error sets may still be useful after normalization, but add them only for measured residual repetition. They should remain transparent finite aliases, with inspection showing their expanded contract. Never sell aliases as the solution to the confirmed seven-error problem.

## 10. Improve captures and type readability without removing receivers

`callable combine` currently captures locals by the exact parameter names declared with `near`. Renaming a captured parameter can break otherwise unrelated caller names. Prefer evaluating an explicit binding form such as proposed `callable combine with (prefix = left, suffix = right)`, restricted to declared capture inputs. This makes dependencies visible and permits ordinary local naming.

Preserve left-to-right once-only capture, immutable aliases, receiver ownership, callable identity and fixture receipts. Do not make methods aliases for first-`near` functions unless package ownership and method resolution are separately preserved. Removing `near` in favor of artificial receiver records also imposes unnecessary data declarations.

For callable arrays, consider grouping the completed callable type: proposed `(callable int (int) emits [])[]`. The current `callable int (int) emits [][]` is defined and unambiguous to the parser, but harder for a reader to segment. This does not require general type aliases or any callable variance change.

Retain `choice_arm` as a distinct checked capability until a replacement accounts for its static description, option identity, capture restriction and registration-time use. Merely spelling its stored type as an ordinary callable hides those requirements.

## 11. Improve testing at the actual blind spots

**Authored native behavior.** The existing evidence categories already distinguish consumer substitution, raw provider fixtures, Bun conformance and live quality. The gap is mandatory coverage of an author's request expressions, criteria, thresholds and handlers. A consumer fixture returning `ok` can continue passing after a prompt or request field accidentally changes.

Add deterministic native cases at fetch/judge/LLM request boundaries: given inputs, compare the prepared descriptor/request and supply raw responses that reach the actual decoder and authored handlers. For judges, cover all question preparation together and exercise threshold boundaries and malformed answers. Individual question descriptors/handlers may be tested as part of that boundary; they should not acquire independent transport wrappers. A descriptor snapshot by itself does not cover threshold behavior. Live model quality remains a separately labelled evaluation, outside mandatory offline builds.

**Fixture reuse.** Start with typed reusable fixture content expanded at lexical `when` sites. Keep expected arguments, explicit assertion-root association, occurrence paths and FIFO ownership. Parameterize the repeated scenario data and let each local table bind it to the relevant root; a helper should not need a copied block of every caller's data.

Do not hoist fixtures using author-visible `function#0` targets and drop argument matching. Ordinals survive formatting but shift when an earlier call is added. A larger root-owned-fixture design needs explicit stable symbolic seams, visibility rules and the same dynamic queue identity. That is a valid future alternative, not a free use of existing ordinals.

**Runtime checks.** Replace intentional division-by-zero failure idioms with a small native-backed `check`/`require` operation carrying an explicit reason. Distinguish an application runtime check from a sticky harness violation: catching an assertion infrastructure failure must still fail its test. Decide whether authors can expect a runtime standard failure directly in assertion rows before adding such syntax; today unhandled standard failures fail the root. No broad testing DSL is needed just to name a failed predicate.

Acceptance should mutate a prompt, one SQL argument, and a callback capture and demonstrate that the relevant fixture fails with a useful location. Source brevity alone is not a valid measure of a testing improvement.

## 12. Add sound scoped-resource checks while retaining runtime ownership

The existing runtime registry is a safety mechanism, not automatically a design contradiction. Native resources can outlive the winning promise; runtime leases and owner draining remain necessary even with stronger static checks.

A simple direct-return prohibition misses a transaction inside a record, an array or a closure. Add a conservative scoped-resource analysis at the known scope boundary. Track direct and transitive contained handles, aliases, and callable captures; use creation/capture information where the callable type alone does not reveal lifetime. Reject definite or conservatively unsafe escape through results or storage outside the scope. In-scope callbacks and captures must remain usable.

This is deliberately narrower than a general ownership type system. Its precision/cost must be evaluated on real transaction helpers. The [transaction checker](../../compiler/internal/check/transaction.go) does not itself show a recursive result/capture escape check; the current runtime safety contract is intentional. This review has not compiled an escape reproducer and does not claim a newly demonstrated runtime safety defect.

Acceptance needs direct, record-contained, array-contained and callable-captured escapes, plus accepted in-scope equivalents and concurrent lease behavior. Keep runtime checks as the final authority for asynchronous lifetime violations.

## 13. Keep the platform boundary; simplify duplicated identity bookkeeping

**Extensibility.** Retain the distribution catalogue until a concrete blocked application demonstrates a missing operation. Publish admission criteria: typed contract, native mapping, failure translation, immutability/lifetime rules, assertion substitution and target conformance. A signature manifest alone cannot establish those behaviors. Supporting a missing safe HTML tag is a narrower decision than arbitrary native access.

Third-party capability packages could eventually satisfy the full same contract, but that is a substantial trust, versioning and conformance design. The earlier review does not supply a blocked program that justifies doing it now. The catalogue decision does not block assertion deadlines or normalization.

**Error identity.** A new issue beyond the earlier recommendations is mandatory global integer allocation. Can already identifies errors by package, declaration and specialization, yet requires globally unique numbers, active/retired registries and lock snapshots. Two unrelated libraries can collide on a number even though their nominal errors differ.

Investigate making canonical nominal identity authoritative and generating report codes only when needed. Require explicit stable protocol identifiers only for actual wire boundaries that need them. Preserve generic specialization and distinguish persisted protocol names from runtime occurrence IDs. Do not derive supposedly stable IDs from source order or promise collision-free truncated hashes.

Stable numeric report codes are an intentional existing use case. Audit their consumers before removing them; zero external users removes migration obligations, not the need to define useful future reporting. Automating the present registry is a smaller alternative if global numbers prove necessary. Evidence: [C9](technical-spec.md#c9), [registry consistency and collision checking](../../compiler/internal/project/graph.go), [registry tests](../../compiler/internal/project/graph_test.go).

**Configuration.** Manifest/lock/SQL validation is appropriately a build-time boundary. The files are compiler-validated JSON, not evidence of a JSON Schema implementation. Moving every descriptor into Can adds grammar without automatically adding database or provider assurance. Generate duplicated data where possible, retain static validation, and link diagnostics back to the relevant field and source declaration.

## 14. Keep primitive semantics and initialization disciplined; fix tooling

The numeric/data model is largely sound for the selected host: exact native `bigint` integers, explicit binary64 floats/conversion, `Object.is` scalar float equality, UTF-16 string indexing plus named scalar/grapheme operations, immutable native-backed collections, and nominal records/variants. Document the surprising cases with small examples: signed zero, NaN, truncating negative integer division, half-open slices, and unpaired surrogates. Do not redesign these simply to differ from JavaScript.

Replace the inaccurate “no inference/coercion” slogan with the actual finite rules: exact generic equality inference; expected typing of literals; leaf and narrower-variant inclusion; callable error-subset compatibility; invariant generic containers; explicit numeric conversions. Existing variant values can enter wider named variants, so the earlier review's proposed “existing values never change type” line is also too absolute.

Keep inert top-level initialization. Full effectful initialization would need an order across files and cyclic imports, resource ownership, fixture context and failure policy. “Run in source order” does not answer those questions for a multi-file package. Explicit startup in `main` is a clearer boundary.

For early-stopping accumulation, a `fold_until` returning `continue`/`done` records could be a useful catalogue addition, implemented with the smallest native iteration adapter. It is an ergonomic/performance improvement, not an expressiveness necessity: same-specialization recursion already exists. Avoid adding general loop control until a real program justifies it.

The formatter situation needs correction. `canlc parse --render` already calls `syntax.Format`; the current renderer discards comments, so it is not a safe write-back formatter. Preserve trivia/comments and add a formatting entry point before promising canonical source editing. Then consider multiline delimiter lists for long calls, bounds and assertions; normalization may remove much of the pressure first.

Semantic diagnostics are more urgent than cosmetic keyword consolidation. [The current bridge](../../compiler/internal/driver/diagnostics.go) can anchor resolve/check failures at the first line of a file. Carry precise source spans and obligation provenance for missing arms, undeclared outward errors, fixture mismatches and inference failures. Offer fixes that the compiler can validate rather than text guesses.

Smaller cleanup candidates: one section order for all question kinds; one Choice production; a deliberate home for receiver-oriented collection operations; a consistent field layout where it improves readability. Keep `noul`/`choice`/`score`, explicit connection selection and the core success vocabulary. Raw-string quote discomfort is too small a reason to make “raw” process an exceptional escape.

## Disposition of all earlier findings

| Input | Assessment and replacement |
| --- | --- |
| R1 native declarations | Keep semantic kinds; normalize section order. “One checker” still needs kind-specific validation. |
| R2 choice arms/callables | Keep distinct until static descriptions, identity and capture rules have a complete replacement. |
| R3 connections | Preserve explicit nominal connection selection; avoid implicit lexical choice or value-equality batching. |
| R4 completion forms | Preserve distinct selection/recovery regions; improve explanation and diagnostics. |
| R5 token overloading | No broad keyword rewrite justified; prefer consistent completions and errors-first ordering. |
| R6 given/state | Compare grouped callable contracts with explicit serialization markers; prior selected spelling requires a new decision. |
| R7 failure channels | Structured ordinary snapshots and exact generic patterns; reject a closed global failure union. |
| R8 emits | Normalize the public boundary first; aliases are optional later and do not satisfy normalization. |
| R9 fixtures | Checked reuse at lexical sites; stable symbolic seams required before root-owned overrides. |
| R10 captures/receivers | Explicit capture bindings are narrower than deleting receiver semantics. |
| R11 error/record layout | Small consistency candidate, below behavioral issues. |
| R11 bare patterns | Explicit aliases may help; avoid changing narrowing, forwarding and nominal specialization together. |
| R11 collections | Choose a consistent API home during catalogue revision; no runtime rewrite needed. |
| R11 initialization | Reject unrestricted effectful top-level initialization. |
| R11 strings | Keep raw semantics; do not add a special escaped-quote exception without evidence. |
| F1 extensibility | Real product boundary, not the first blocker. Admit capabilities with complete contracts. |
| F2 natives/standard catch | Real narrower gaps: authored request/handler coverage and inconsistent snapshots; native conformance already exists. |
| F3 chain/traversal | Evidence does not justify removing chain. Halting fold is useful, not necessary for expressibility. |
| F4 resources | Add transitive/capture-aware scope checks; direct syntax rules alone are insufficient. |
| F5 hanging assertions | Confirmed liveness gap; add an externally enforced deadline. |
| F6 verbosity | Measure equivalent behavior, tests and edits; do not compare to bare TypeScript alone. |
| F7 inference claims | Correct both the slogan and the proposed replacement wording. |
| F8 JSON contracts | Keep build-time validation; eliminate duplicated identity data where justified. |
| F9 formatter/diagnostics | Renderer exists but loses comments; improve it and semantic source spans. |

## Suggested implementation order and decision gates

1. **Build guarantee, liveness and feedback:** reconcile assertion execution with publication, add the supervisor deadline and precise reports, structured standard snapshots, and errors-first enforcement. These can proceed independently of larger syntax choices once adopted.
2. **Confirmed normalization:** settle the normalized payload and origin categories, then wrapper inheritance/parent delegation and public-bound computation. Prove the eight-call reduction and selective recovery without restructuring nesting.
3. **Composition:** exact generic error patterns, practical map-based aggregate conversion, deterministic native request cases, and lexical fixture-content reuse.
4. **Prototype before adoption:** ordinary state-callable signatures, finite error-set parameters, explicit capture bindings, and scoped-resource analysis. Use one concrete application per proposal and include rejected-program diagnostics.
5. **Polish and scope:** comment-preserving formatting, delimiter layout, catalogue API consistency, error-ID consumer audit, and only then any justified platform extension.

Do not turn this list into one large compiler rewrite. Each adopted item needs a clear semantic contract, a before/after program, negative cases, and native lowering evidence. Remove superseded forms directly instead of carrying compatibility branches.

The four applications currently contain 229–373 source lines each; the longest lines range from 187 to 314 characters. The AI example has 67 mentions of the seven infrastructure identifiers and 1,316 characters in its `emits` declarations excluding indentation. [Exact measurements](evidence/2026-09-22/full-language-review/measurements.json) are reproducible. These are not model token counts. A fair Can/TypeScript comparison must include equivalent error contracts, tests, fixtures and boundary adapters, and should also measure how many places change when adding one failure or modifying one request.

## Consultation and verification record

Nine difficult choices were presented to Jev in three freshly reworded complete packets. Context, questions and all option descriptions were varied while preserving material facts and alternatives; requests, responses, wording audit and reconciliation are linked from the [consultation record](evidence/2026-09-22/full-language-review/README.md). Agreement is advisory, not proof or a vote that overrides requirements. Prototype-gated choices remain prototype-gated regardless of classification confidence.

The current [TypeSafe HTTP API](https://docs.typesafe.ai/api), [Choice guidance](https://docs.typesafe.ai/primitives/choice), and [parallel-questions cookbook](https://docs.typesafe.ai/cookbooks/parallel_questions) supplied the consultation interface: typed choices over supplied evidence, batched independent questions, and recorded probability distributions. They do not establish the correctness of any Can design.

This was a design/evidence review, not a full executable conformance or security audit. No compiler/runtime suite was rerun because their sources were not changed. Static source checks and measurement validation were performed; proposed syntax was not compiled. The distinction matters especially for resource escape analysis, new effect parameters and wrapper inheritance, which need dedicated prototypes before implementation claims are made.
