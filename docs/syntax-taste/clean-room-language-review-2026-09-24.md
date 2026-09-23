# Can language design: independent full review

**Consolidated recommendation:** [One recommendation program for Can](can-recommendation-program-2026-09-24.md) reconciles this review with the SaaS example review. This independent review remains evidence; shared findings are counted once in the consolidated program.

24 September 2026 · source revision `4c2db1e1162a45e88a58ef154e848790a1a44c60`

**I would recommend Can for a much wider range of SaaS development if its guarantees remained predictable when composing functions, libraries, tests, and UI boundaries.** Its strongest ideas already work together: nominal immutable data, explicit domain errors, attached executable assertions, and controlled native operations. The next design work should make those ideas available consistently to ordinary authors.

The current frontend is deliberately server rendered, with HTMX interaction. That is a credible product scope. Recommending Can for *all* SaaS frontends would additionally require a decision about browser computation, local state, and offline behavior. Improvements to HTML syntax alone would not close that gap.

This is a review and a proposed research order, not a change to the selected language specification. There is no compatibility obligation to preserve a design found wanting.

## Independence, evidence, and limits

Three reviewers started with fresh contexts and no conversation history. They independently reviewed core language design, native/completion behavior, and platform/frontend composition. They were instructed to avoid historical reviews, previous recommendations, consultation results, and one another's reports during their independent phase. They received the present source, normative contract sections, and selected constraints. The coordinating reviewer retained the surrounding conversation; therefore the independence claim applies to those three reviews, not to complete amnesia by the coordinator.

The review covers syntax, values and types, matching, generics, errors, callables, modules, assertions, native AI, coordination, resource ownership, HTML/forms/routing, SQL, collections, and build/tooling boundaries. Current examples were divided between reviewers; the source inventory contains 56 `.can` files under `examples/` and four maintained standard-library example files. This was source review, not execution of every example.

The coordinator constructed ten small programs against the current checker and assertion emitter. Three were rejected as expected; seven compiled and produced 19 assertion-root executions: 18 passed and one intentionally exposed a fixture-selection mismatch. Three fresh Jev consultations tested competing remedies; disagreement was investigated rather than resolved by vote. Raw reviews, exact requests/responses, probe sources, reproduction instructions, and verification logs are in the [evidence directory](/Users/vince/Projects/can-lang/docs/syntax-taste/evidence/2026-09-24/clean-room-full-review/README.md).

This review distinguishes observed behavior from preferred policy. Most findings are limitations or surprising consequences of the present contract, not compiler implementations violating their specification. It does not establish a general type-safety theorem, provider quality, production performance, or release qualification.

## What I would preserve

| Design choice | Why it earns its place |
| --- | --- |
| Nominal immutable records | Business concepts remain distinct even when their shapes match; native objects and freezing provide the representation. |
| Closed variants and substantive exhaustiveness checking | Useful support for state transitions and future changes. The pattern ambiguity below is repairable without discarding this model. |
| Explicit finite public domain-error bounds | Callers can see failure obligations. Standard failures remain a different category. |
| Named functions and immutable captures | Behavior has a stable declaration and assertion site. Anonymous functions are not required to fix the specific capture problem below. |
| Attached assertions and gated build publication | Executable examples belong to the implementation and are checked before a generation is published. This is evidence, not universal correctness. |
| Exact integer arithmetic and explicit conversions | Native bigint avoids silent integer precision loss; float and text contracts are deliberately specified. |
| Native AI declarations and explicit grouped state | Requests, decoding, state, and error contracts are visible. Their remaining composition costs should be measured within this model. |
| Native Promise operations with completion/ownership adapters | Selection follows native semantics; protected completions avoid thenable assimilation, and owners retain resource leases while work is live. |
| Opaque safe HTML, bounded SQL result queries, and parameterized mutations | These establish useful containment guarantees, provided their exact limits are stated. |

Sources include [native values](/Users/vince/Projects/can-lang/runtime/data.ts:23), [numeric conversion](/Users/vince/Projects/can-lang/runtime/number.ts:58), [completion protection](/Users/vince/Projects/can-lang/runtime/completion.ts:13), and [native coordination](/Users/vince/Projects/can-lang/runtime/coordination.ts:84). There is no evidence here that Can needs null, `any`, implicit coercions, unrestricted foreign JavaScript, inheritance, or a general trait system.

## 1. Make pattern intent unambiguous

**Highest-priority core change. Confirmed by checker and emitted execution.**

An unresolved bare name in a pattern becomes a binding for the entire value. Consequently, a misspelled final leaf is an accepted catch-all:

```can
    match value
        paid => ok true
        pending => ok false
        decliend => ok false
```

Here `decliend` was intended to mean `declined`. The current compiler accepts it. Adding a fourth payment state also leaves the function accepted, while the correctly spelled three-arm match becomes non-exhaustive. The mistake removes the future-change protection a user reasonably expects from a closed variant.

This follows the current [pattern resolution rule](/Users/vince/Projects/can-lang/compiler/internal/check/patterns.go:65): unresolved unqualified names become `any` patterns. The coverage algorithm then correctly treats the resulting binding as exhaustive. Explicit `decliend()` is rejected; the dangerous ambiguity is specific to bare names.

**Recommendation:** distinguish bindings syntactically from nominal tests, and retain an explicit intentional catch-all. Constructor syntax is another viable design. Pick the exact spelling through examples; a spelling heuristic alone cannot distinguish a deliberate binder from a typo reliably. Acceptance criterion: misspelling a leaf must never silently acquire catch-all meaning, including in the final arm.

## 2. Decide whether authored libraries may own value invariants

**High-priority library-scope decision. Confirmed limitation; remedy needs design.**

An exported `email` record plus a validating factory does not establish that every `email` is valid. Importers can construct the record directly and copy-update its public fields. All source records are [constructible](/Users/vince/Projects/can-lang/compiler/internal/resolve/symbols.go:291); public visibility does not separately hide construction or representation. Keeping the type private prevents its use in public signatures.

Nominality prevents accidentally treating an email as another record. Immutability prevents mutation of an existing instance. Neither enforces the predicate checked by the factory. The same issue arises for positive quantities and normalized identifiers. Authorization still belongs at the relevant operation boundary, even with abstract values.

**Recommendation:** evaluate a small package-owned representation boundary for ordinary immutable data. Export the type and selected observations; keep construction and representation updates under its owner's control. Specify equality, matching, decoding, schema derivation, and assertion construction together so that a generic decoder cannot bypass the factory.

The strongest alternative is deliberate transparency: validate inputs and sensitive operations, and keep all authored values as inspectable DTOs. That keeps matching and codecs simple. It becomes a significant constraint when independent libraries are expected to promise invariants. Compare three real packages—validated email, positive quantity, and tenant identifier—before accepting the additional feature. Resource-owner privileges need not accompany ordinary data abstraction.

## 3. Let ordinary libraries express useful generic contracts

**High-priority composition work; two separate problems.**

First, catalogue collection operations preserve a callback's concrete error set, but an authored generic wrapper cannot quantify over that set. A reusable `timed`, `retry`, `map_option`, or `with_audit` cannot state “return the callback's value and preserve exactly its domain errors” for arbitrary callers. The compiler has this ability through [catalogue specialization](/Users/vince/Projects/can-lang/compiler/internal/check/array.go:302), while authored [error bounds name concrete declarations](/Users/vince/Projects/can-lang/compiler/internal/resolve/symbols.go:593).

**Recommendation:** compare a narrowly explicit finite error-row parameter against ordinary nominal result-as-data wrappers. Keep bounds visible. Result data is workable, but creates a second composition style and conversion code; adding more privileged compiler combinators does not solve the general library problem. A prototype must show useful wrappers and comprehensible errors before broader effect machinery is considered.

Second, generic bodies behave as templates: concrete instantiations are checked, but the public signature does not describe all operations required by the body. An apparently general `doubled<item>` can use `value + value` and work only for admitted concrete types. Changing the body can break a consumer specialization without changing the signature. This is a sound template model with a weak separate-library contract, not unchecked generated code. See [specialization](/Users/vince/Projects/can-lang/compiler/internal/check/program.go:487) and its [whole-body checks](/Users/vince/Projects/can-lang/compiler/internal/check/specialize_test.go:54).

**Recommendation:** first express required operations as existing named callable parameters or dictionary records. If those examples remain awkward, consider a small capability vocabulary. Do not leap from this problem to full traits, higher-kinded types, or operator overloading. Retaining documented template semantics is also defensible for local helpers.

## 4. Make independent packages independent

**High-priority ecosystem contract, even before package distribution.**

Two dependencies cannot both contain a package with the same short name, despite the graph already having project-qualified identities. Likewise, two independently allocated application error IDs collide across dependencies; retired IDs also participate. An application may be unable to combine unrelated libraries before their business types ever interact. See [package names](/Users/vince/Projects/can-lang/compiler/internal/project/graph.go:273) and [registry graph validation](/Users/vince/Projects/can-lang/compiler/internal/project/graph.go:375).

**Recommendation:** make dependency-qualified import identity and local aliases possible, and scope diagnostic error identity accordingly or assign collision-free build IDs. Preserve clear machine/human diagnostics; numeric IDs currently have a purpose and should not simply disappear. The design test is two independently authored packages with overlapping convenient names and locally allocated errors, consumed without editing either library.

Local dependency confinement, hash locks, the LSP, and editor integration already exist. This finding is about composition of names and contracts, not a missing editor or a need to build a registry first.

## 5. Give test scenarios explicit ownership across calls

**Confirmed by an emitted cross-package probe.**

Inline `when` fixtures select rows using the root assertion's short label. A caller's label can therefore activate a fixture inside a helper package; renaming only that label can stop the fixture from applying. In the probe, `customer` receives `"fixture"`; renaming the root to `renamed` runs the real `text::from_int(7)` and returns `"7"`. Static checking still succeeds, and the caller's emitted assertion fails as expected.

The relevant rule is [root-name fixture selection](/Users/vince/Projects/can-lang/runtime/assert/fixtures.ts:29). Full root identities and queues remain isolated; this is not a nondeterministic collision between concurrently running tests. Assertion templates provide lexical reuse but do not remove the root-label dependency.

**Recommendation:** explore explicit named scenario/seam references so the caller deliberately supplies the scenario consumed below it. Preserve argument matching, invocation identity, and FIFO occurrence behavior. Stubbing the entire helper is a valid unit-test alternative, but no longer tests the helper body in that path. Merely qualifying labels reduces collisions without necessarily eliminating caller/callee coupling.

Retain attached assertions. Improve their composition and explain observation strength: a bare successful `ok` for an opaque result establishes successful completion, not the exact HTML, resource contents, or all externally visible effects. Browser and native integration checks remain necessary.

## 6. Clarify variant identity instead of implying stronger branding

**Confirmed behavior; medium-priority contract decision.**

```can
record unit
variant tagged<item>
    unit
variant bridge
    unit
```

Directly returning `tagged<int>` where `tagged<str>` is expected fails. Assigning the same value to `bridge` first, then returning it, succeeds. Narrowing to the `unit` leaf also succeeds. The probes executed both admitted programs.

The [compatibility rule](/Users/vince/Projects/can-lang/compiler/internal/types/compatibility.go:18) rejects differing specializations of the same declaration, then permits leaf-set inclusion between distinct variants. These rules make assignment non-transitive. There is no runtime memory violation, and ordinary generic records retain their nominal identity.

**Recommendation:** choose and teach one precise contract. Extensional variants—named sets of admitted leaves—fit the existing flattened representation most naturally, with nominal records providing brands. Keeping the direct-specialization guard is defensible if explicitly described as a conservative direct-assignment restriction: moving through a general variant discards that distinction. Strong nominal variant wrappers are a larger alternative with consequences for matching and codecs. Compare all three against these probes, normal `option<T>`, and generic records; the observed behavior alone does not prove which policy is best.

## 7. Make captures and layout easier to refactor

**Medium-priority everyday authoring improvements.**

`near` captures resolve a callee parameter's exact spelling in the reference creator's scope. A parameter rename can break reference creation; another same-typed local with that name can change the captured value. The closure is safely immutable once created, but the binding decision is ambient. See [capture resolution](/Users/vince/Projects/can-lang/compiler/internal/check/callables.go:124).

Compare the existing explicit context-record/receiver idiom with a small explicit capture-binding form for named functions. Measure refactoring a handler with two values of the same domain type. This does not require anonymous functions or changing capture-once semantics.

Separately, the one-physical-line argument/constructor/assertion restrictions and compile-failing unnecessary-local rule turn presentation choices into language acceptance rules. A meaningful local such as `permitted` can be rejected when direct substitution is provably valid. See [argument parsing](/Users/vince/Projects/can-lang/compiler/internal/syntax/expressions.go:204) and [local elision enforcement](/Users/vince/Projects/can-lang/compiler/internal/check/locals.go:44).

Permit continuation inside explicit delimiters, retain indentation for blocks, and consider making unnecessary-local guidance advisory. The test is readable wide records and assertion rows under a normal review width, not syntactic minimalism measured only in line count.

## 8. Connect frontend contracts, then choose the frontend scope

**Important for SaaS adoption; a mixture of language/library design and product scope.**

Safe HTML is valuable, but current renderers spend much of their code manufacturing tags, attributes, URLs, nodes, and error branches for static structure. The [accounts renderer](/Users/vince/Projects/can-lang/examples/account-search/src/render/render.can:53) is a concrete example. A helper library should be the baseline comparison. If that remains too costly, prototype checked declarative HTML/components that lower to the same safe native representation; reject statically known structure errors before runtime.

Routes, form fields, and HTMX targets are also separate string contracts. Renaming a form field or target can compile successfully and fail only when a browser submits or swaps. Typed references could connect those boundaries, but should promise only what they prove: a safe URL does not establish that its endpoint exists, and a valid target identifier does not establish that it is present in a conditional DOM.

The router currently accepts exact paths and rejects parameter segments. Query-string IDs still support CRUD with normal HTTP methods; the gap concerns APIs and navigable resource URLs that require paths such as `/org/:org_id/projects/:project_id`. Typed captures with explicit parse/error behavior would extend the platform without requiring a general routing language. See [route admission](/Users/vince/Projects/can-lang/compiler/internal/check/http.go:304).

HTML form wire data being `str`, `str[]`, or optional `str` is reasonable. The missing convenience is a composable validation layer that builds nominal application values, accumulates field errors, and supports repeated rows. Avoid silently turning form decoding into arbitrary JSON-style coercion.

Finally, the current [browser boundary](/Users/vince/Projects/can-lang/docs/syntax-taste/technical-spec.md:421) explicitly excludes browser Can. The platform uses pinned HTMX, restricts executable project assets, and serves server-rendered fragments. It suits forms, search, content, and server-backed dashboards. A Can-only offline editor, canvas application, custom browser WebSocket client, or rich optimistic local interaction is outside that selected model.

**Recommendation:** keep “excellent server-driven SaaS” as a concrete first recommendation target. If “all SaaS frontends” remains the goal, separately compare browser Can against a controlled integration with an existing browser stack using a demanding client-state example. That is a new architecture decision, not an implementation bug or an automatic reversal of the current boundary.

## 9. State the exact SQL guarantee and remove demonstrated friction

SQL descriptors check statement shape, parameters, dialect/cardinality rules, and whether declared rows can use the supported codecs. They do not establish that a query against a versioned database schema actually returns those declared fields. A current positive compiler test accepts `SELECT id` with a wider declared row. Runtime row validation contains the mismatch. See [the accepted SQL example](/Users/vince/Projects/can-lang/compiler/internal/check/sql_test.go:84) and [descriptor checks](/Users/vince/Projects/can-lang/compiler/internal/check/sql_descriptors.go:47).

**Recommendation:** describe this accurately as checked parameterized-query contracts with runtime result validation. If compile-time database shape safety is a goal, introduce a versioned schema snapshot/build input and define migration drift behavior. Migration execution itself can remain tooling. Schema-aware checks and runtime codecs solve different problems; neither proves a deployed database matches the build input.

Also evaluate bounded mutation-returning descriptors. Current [cardinality rules](/Users/vince/Projects/can-lang/compiler/internal/sql/cardinality.go:24) exclude mutation `RETURNING`, so creating/updating then rendering canonical saved data requires another query. Existing transactions can protect the sequence, but do not remove the round trip. A narrowly specified dialect-aware descriptor is more direct than introducing an ORM.

## 10. Keep lifetime and coordination promises precise

These are deliberate policies or guarded limitations, not established safety defects:

| Area | Current behavior and implication | Appropriate next step |
| --- | --- | --- |
| Scoped results | `with_transaction<T>` can return data containing its now-closed transaction. Later use raises `resource_state`; runtime safety is preserved. | Explore diagnostics for obvious escapes of the scope's own handle. A blanket ban on resource-valued results is wrong: an enclosing-owned pool can remain valid. |
| Early settlement | Losing participants retain leases until they settle. A permanently pending loser can prevent clean root shutdown. | Document lifecycle responsibilities; consider explicit cancellation for operations that support it and bounded host supervision for others. Do not promise cancellation of arbitrary Can computation. |
| Empty dynamic race | `race with error` uses `Promise.race`; an empty expansion remains pending. First-success `race` uses `Promise.any` and has a different empty outcome. | Make guards/nonempty construction convenient and visible. Changing native empty behavior is a separate policy decision. |
| First-success diagnostics | Failures before a winner are consumed during the search; standard failures after a winner produce late diagnostics. | Treat this as an observability tradeoff. A replica failure may be hidden by a successful fallback, but reporting every consumed failure may create noise. Validate a service scenario before introducing an opt-in policy. |

Evidence: [transaction result checking](/Users/vince/Projects/can-lang/compiler/internal/check/transaction.go:74), [closed-handle guards](/Users/vince/Projects/can-lang/runtime/owner.ts:153), [scope drain](/Users/vince/Projects/can-lang/runtime/owner.ts:462), and [the diagnostic timing test](/Users/vince/Projects/can-lang/runtime/test/coordination.test.ts:120). A general borrow checker, implicit loser cancellation, or changed Promise selection does not follow from this review.

## 11. Finish native AI composition through concrete workflows

Native declarations are more structured than an untyped prompt-and-parse escape. Keep the explicit state and decoding contracts. Three boundaries deserve attention:

- **Recovery policy:** provenance-aware `wrap` supports fetch/judge/wrap, while LLM callers use ordinary wrappers. An ordinary wrapper cannot distinguish authored and native-decoder failures solely by a shared error kind. Test whether extending the existing policy to LLM removes meaningful duplication while retaining grouped state. [Current admission](/Users/vince/Projects/can-lang/compiler/internal/resolve/native.go:75).
- **Effects in batches:** response validation precedes all handlers, but handler effects are not rolled back if a later handler fails. Generated-record construction is all-or-nothing as a returned value, not a transaction over earlier writes. Teach the preparation, validation, handler, and result phases; test a failing second handler after an observable first effect. [Lowering](/Users/vince/Projects/can-lang/compiler/internal/emit/judge.go:244).
- **Callable composition:** judge/LLM grouped signatures do not become ordinary callables directly. Named wrappers work today. Only pursue bound grouped-state callables if a real dynamic fan-out example demonstrates substantial cost after using those wrappers. [Callable admission](/Users/vince/Projects/can-lang/compiler/internal/check/callables.go:13).

None of these proves a need to merge every native declaration into one form, remove grouped state, infer public errors, or add a language-wide purity system.

## 12. Give immutable collections efficient construction paths

Individual map insertion/replacement clones the native map; set addition does likewise. The gallery's word-count fold therefore performs quadratic copying when each input introduces a new key: `0 + 1 + ... + (n - 1)` previous entries. Ownership traversal adds work. This is a source-level complexity deduction, not a measured throughput claim. See [map operations](/Users/vince/Projects/can-lang/runtime/collections/map.ts:47) and [the gallery idiom](/Users/vince/Projects/can-lang/examples/gallery/src/24-map-word-frequencies.can:6).

**Recommendation:** native bulk construction/grouping/aggregation with explicit duplicate semantics, callback ordering/failures, and one final immutable publication. Compare that against an internal scoped builder only if necessary. Preserve immutable user values; neither a custom persistent-map engine nor general mutable aliases is the obvious first answer.

The same discipline applies to the closed native catalogue. It provides a controlled trust boundary, while an unsupported SDK currently requires authored HTTP/process integration or a compiler/runtime extension. Evaluate one genuinely awkward integration before designing a narrow typed adapter contract. No SDK was demonstrated impossible in this review; unrestricted JavaScript is not justified by general ecosystem anxiety.

## Recommended order and decision gates

| Order | Work | Evidence required before accepting the design |
| --- | --- | --- |
| 1 | Unambiguous patterns | The typo/new-leaf probes fail safely; intentional bindings and nested patterns remain readable. |
| 2 | Library boundaries: abstract values, generic/error contracts, package identity | Three independent domain packages plus reusable wrappers compose without special compiler privileges or source edits to dependencies. Compare existing idioms. |
| 3 | Assertion scenarios and variant identity | A caller-label rename has explicit, predictable fixture behavior; direct/bridge/leaf variant conversions follow one teachable contract. |
| 4 | Daily authoring and server-driven frontend | A realistic resource-edit flow uses readable captures/layout, validated forms, safe rendering, and connected route references. Compare helper-library and syntax approaches. |
| 5 | SQL, native AI, collections, and service lifetime | Mutation-returning, generic AI wrappers, bulk construction, and shutdown examples demonstrate the benefit and retain native-operation contracts. |
| Separate strategic decision | Browser computation and native extension scope | A rich client-state example and an unsupported integration show exactly what the present boundary cannot reasonably serve. |

My recommendation threshold is a language that can support several independently authored packages and a changing, realistic product without weakening its contracts or continually adding compiler-owned exceptions. Can has a substantial foundation for that. The most persuasive next progress would be stronger composition and clearer scope, backed by representative programs.

## Verification performed

- `bun run check:runtime` passed.
- Tests for six compiler packages passed: syntax, types, resolve, check, emit, and driver. The JSON stream records 1,089 passed test/subtest events and 14 skips: one intentional negative formatting fixture and 13 execution/package tests requiring `CAN_BUN` or `CAN_BUN_ARCHIVE` configuration.
- Nine selected runtime test files passed: **77 tests, zero failures**, covering coordination, owners, collections, assertions/identity, JSON, HTML, and SQL descriptors.
- Ten fresh probe programs exercised pattern intent, variant conversion, and cross-package fixture selection. The one expected assertion mismatch is evidence of the fixture behavior, not a regression introduced by this review.

The probe harness used the current internal checker/emitter and local Bun 1.4.2; it did not perform a packaged verified build or source-map qualification. Live databases, provider calls through generated Can, browser integration, all-example builds, and the full release suite were not rerun. No compiler, runtime, catalogue, or example implementation was changed.
