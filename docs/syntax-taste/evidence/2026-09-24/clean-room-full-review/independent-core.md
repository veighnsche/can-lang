# Independent clean-room review of Can core

## Scope and overall assessment

Reviewed the current compiler's syntax, resolver, type graph, specialization, patterns, expression/completion checking and lowering; runtime data, primitive, numeric, text, callable and collection adapters; gallery and current standard examples. Read AGENTS.md and normative C2–C10 material. No historical review, recommendation/disposition ledger, sibling report or prior consultation was read. No source edits or Jev consultations were performed. The parent reviewer is independently probing the key acceptance/rejection cases below. Unless marked otherwise, examples below are deductions from the current checking rules, not claims of executed tests.

Can has a coherent value-oriented application core: nominal immutable records, closed variants, explicit domain failure bounds, named callables, and deliberately native arithmetic/data operations. Its strengths are real implementation properties, not merely proposed syntax. The main weakness is the seam between this strong local model and reusable APIs: library authors cannot establish abstract value invariants, cannot express the same fallible higher-order abstractions as the compiler catalogue, and cannot state the admissible domain of an operational generic. Several source-level rules also make meaning depend on spelling or presentation more than the stated simplicity goal warrants.

For full-stack SaaS, I would retain the selected language identity and repair these seams before adding many more special-purpose surface forms. Named functions, nominal immutable data, explicit public error bounds, mandatory attached assertions, native AI declarations and grouped AI state do not need to be abandoned to address the major findings.

## Prioritized findings

### 1. P1 — Bare variant patterns silently turn unresolved leaf names into catch-all bindings

Evidence: `compiler/internal/check/patterns.go:65` attempts leaf resolution, then `:101-109` makes an unresolved unqualified name an `any` pattern and binds the entire matched value. Qualified names and explicit type applications fail instead (`:102-103`). The matrix checker correctly treats that resulting `any` as coverage (`compiler/internal/check/pattern_coverage.go:29-39`); the defect is the ambiguous source intent before coverage analysis.

Minimal current-Can probe, with the misspelling intentional:

```can
package app
    provides []
    uses []
record paid
record pending
record declined
variant payment
    paid
    pending
    declined
fn bool accepted
    emits []
    given
        payment value
    asserts
        sample: paid() => ok true
    match value
        paid => ok true
        pending => ok false
        decliend => ok false
fn void main
    emits []
    given
        str[] args
    asserts
        sample: [] => ok
    ok
```

`decliend` becomes an unused binding for the remainder, so the function is considered exhaustive. Adding `refunded` to `payment` would leave this match accepted and silently reuse the wrong arm. A spelling error in a billing state machine defeats precisely the future-change assistance closed variants should supply. A typo in the first arm is more likely caught through unreachable later arms; the final-arm case is the important counterexample. This is a language-design ambiguity, not an unsound implementation of its present rule.

Make binding patterns syntactically distinguishable from nominal tests, retaining lowercase names: for example, an explicit binder production and `_` for an intentional catch-all. Alternatively require `paid()` for fieldless tests and explicit nominal constructor patterns generally; this is more repetitive for payload-bearing leaves. Merely suggesting qualified names is insufficient for local variants. Warn-only misspelling heuristics cannot provide the same guarantee because intentional arbitrary binding names are indistinguishable today.

### 2. P1 — Exporting a record exports the ability to forge its representation; ordinary libraries cannot enforce value invariants

Evidence: every source record is marked `Constructible = true` at `compiler/internal/resolve/symbols.go:291-295`; qualified lookup checks one `Public` bit for both type and constructor access at `:435-442`. Public field/signature traversal rejects inaccessible private types (`:535`, `:613-618`). There is no source abstract/opaque declaration in the type declaration inventory; runtime opacity exists for maintained catalogue values.

Minimal scenario: a package exports `record email` with field `str value` and a validation function producing `email`. Any importer can construct `addresses::email("not an email")`, inspect `.value`, and use `with value = "invalid"`. Keeping `email` private prevents public APIs from returning it. Hiding the implementation in a private nested field type is also rejected for public record fields. A closure-based public record is not an unforgeable value boundary: clients can construct their own record and behavior.

For SaaS, this affects normalized email addresses, tenant-scoped IDs, permission-bearing tokens, nonnegative amounts, sanitized fragments and state transitions. Immutability prevents modification of a given value; it does not establish that all inhabitants passed validation. Nominality prevents accidental cross-domain assignment; it does not establish a domain predicate.

Add a package-controlled representation boundary for authored nominal data: export the type while keeping construction, representation projection and copy-update inside its defining package. Keep the same immutable runtime representation and provide explicit validated factories and selected observations. Decide deliberately how decoding and schema derivation construct such types—generic codec reconstruction must not become another bypass. Public transparent data remains useful and should remain the simple case.

Tradeoff: abstract values complicate codecs, pattern matching and derivation. That complexity is justified if reusable libraries promise domain invariants; it is not necessary if Can intentionally limits all user-defined records to transparent DTOs. The latter is a substantial SaaS library-scope constraint and should be stated explicitly, rather than treating nominality as sufficient abstraction.

### 3. P1/P2 — Fallible higher-order composition is privileged to compiler catalogue operations

Evidence: array specialization reads the actual callback errors into the operation contract (`compiler/internal/check/array.go:302-308`). Authored bounds resolve only actual nominal error declarations: `compiler/internal/resolve/symbols.go:593-610`, together with `Symbol.Eligible`'s `ErrorUse` case, and `compiler/internal/types/types.go`'s `function` error-kind check. Normative C4 explicitly forbids authored error-set parameters. Ordinary callable compatibility widens only to a written finite error set (`compiler/internal/types/compatibility.go:49-58`).

A reusable `timed`, `retry`, `paginate`, `with_audit`, `compose`, or `map_option` function cannot accept an arbitrary caller's `callable item (...) emits E` and preserve exactly E. Its author must list a closed application-specific bound; consumers must normalize every domain error to that bound; or the compiler must acquire yet another special intrinsic. A single generic data parameter standing for an error does not solve an arbitrary finite error set and is not admitted as an `emits` entry today.

This is especially visible when replacing a direct built-in `values.map(operation)` with a named library function wrapping the same operation: the public contract cannot preserve the existing degree of genericity. This is a composability limitation, not a request to infer errors silently or erase public error bounds.

Consider a deliberately limited, explicitly declared error-row parameter with finite substitution and set union, written in the public signature. A lower-complexity interim choice is an explicit nominal `result` value encoding at abstraction boundaries; generic containers and combinators then manipulate ordinary data, and adapters convert to/from named completion paths. The latter is workable but introduces parallel result/completion styles and extra records. Keep current concrete bounded higher-order functions as the simple application case. Do not solve this by making every callback emit an untyped catch-all error or by requiring an ever-growing compiler-owned combinator list.

### 4. P2 — Phantom generic variants expose a non-transitive assignment relation

Evidence: `compiler/internal/types/compatibility.go:18-21` refuses different specializations of the same nominal declaration even if their leaf sets coincide. `:23-36` then permits inclusion by leaf-set identity between different variant declarations. A named variant has no wrapper representation.

Minimal type declarations:

```can
record unit
variant tagged<item>
    unit
variant bridge
    unit
```

A function returning `tagged<str>` cannot directly return an input `tagged<int>` under the explicit same-template guard. But it can bind that input to `bridge` and return the bridge, because both adjacent assignments have identical admitted leaves. Moreover, `unit()` enters either specialization directly. The intermediate local should survive the unnecessary-local rule because direct substitution changes validity.

This does not forge a memory representation or violate leaf matching. It demonstrates that the apparent invariant nominal distinction is not an abstraction boundary and that insertion of a harmless-looking intermediate type changes program admissibility. Documentation saying both "nominal generics are invariant" and "variants are flattened leaf-set unions" needs a carefully qualified model here.

Choose one coherent interpretation. Treat variants extensionally as named admitted-leaf sets, removing the special phantom-invariance rejection; or make genuinely nominal tagged variants carry a wrapper/tag distinct from their leaves; or disallow variant parameters that do not survive into a concrete leaf identity. The first is simplest and preserves today's representation, while the second supports true branded phantom domains at the cost of constructors and more matching structure. Do not extend this concern to ordinary generic records: their concrete leaf identity already preserves arguments correctly.

### 5. P2 — Operational generics have no declaration-level admissibility contract

Evidence: generic bodies are checked for discovered concrete instances (`compiler/internal/check/program.go:487-518`), generated by `compiler/internal/check/specialize.go:72-130`; uninstantiated declarations are templates. Operator admission is on the concrete substituted operand types (`compiler/internal/check/expressions.go:169-184`). `compiler/internal/check/specialize_test.go:54-95` correctly insists that the whole instantiated body is checked, including unexecuted branches.

For example, `fn item doubled<item>` with input `item value` and body `ok value + value` describes a signature that can be instantiated for int, float and str, but not an arbitrary record. It has the same apparent generic domain as the gallery's `identity<item>`, despite radically different admissibility. Changing an implementation from pure forwarding to an equality, serialization or collection operation can break a consumer's specialization without changing the exported signature. The consumer must read source or discover constraints through errors.

This is an intentional template model, not type unsoundness: valid concrete bodies are checked. It is adequate for local generic helpers. It is a weak contract for separately developed reusable packages.

Prefer a small explicit capability vocabulary for operations that truly need it—equality, wire admissibility, keys—or require operations as named callable parameters/dictionary records, which fits the current language and avoids a broad trait system. Another legitimate choice is strict parametric generics: unconstrained parameters may only be stored, forwarded and passed through explicitly supplied operations. That loses convenient ad hoc templates but makes public signatures honest. Do not immediately introduce full higher-kinded types, overlapping instances or arbitrary operator overloading; the concrete problem does not require them.

### 6. P2 — `near` captures turn a callee's parameter spelling into an ambient call-site dependency

Evidence: `compiler/internal/check/callables.go:124-138` creates a new unqualified name expression from `declaration.Names[i]`, resolves it in the reference creator's scope, then checks exact type. Immutable capture-once behavior itself is sound (`runtime/callable.ts:25-51`).

For a named predicate with `near int tenant_id`, `callable allowed` requires an in-scope value spelled exactly `tenant_id`. A caller possessing `current_tenant` cannot explicitly bind it at the reference expression; it must introduce a spelling-matched local or another named wrapper. A nested same-typed `tenant_id` silently changes what a reference captures. Renaming the callee's near parameter changes every reference creator's required lexical environment, while direct positional calls remain unaffected.

This matters for reusable request handlers and authorization predicates where several tenant/user/session values of one type are in scope. Exact typing cannot distinguish mistaken selection among same-typed values; stronger nominal IDs only partly help when both values are legitimate IDs of the same domain.

Retain named function bodies and immutable native closures, but allow explicit capture binding or ordinary partial application at reference creation. A receiver method on an explicit context record is an existing workaround, with the cost of another nominal record and the receiver-package ownership restriction. Making `near` capture behavior visible at every call site would make refactoring safer. This is a justified reconsideration of capture ergonomics within the selected named-function model, not a demand for anonymous lambdas.

### 7. P2 — Physical-line restrictions and compile-failing style rules impose cost without semantic protection

Evidence: argument parsing accepts expressions and separators but no continuation (`compiler/internal/syntax/expressions.go:204-224`, grouped expression at `:149-153`), consistent with normative C2's one-physical-line rule. The local-forwarding checker deliberately returns a compile error after proving an otherwise semantically useful program's final name is substitutable (`compiler/internal/check/locals.go:44-111`).

For example, `bool permitted = is_owner or is_admin` followed by `ok permitted` is rejected when the rule's precise predicates hold. That local can still be a useful name for a business decision or a debugging location. Replacing `or` with a call, constructor, division or other excluded form changes whether the exact same naming style compiles. A record constructor with a dozen fields, a deeply nested callable type, or a large assertion row must occupy one physical line regardless of review width. Intermediate bindings mitigate some cases but do not preserve adjacent field/value readability or assertion layout.

Permit newline continuation within explicit delimiters and make unnecessary-local guidance a formatter/linter suggestion. This preserves indentation-based blocks and all selected semantic forms, including mandatory attached assertions. If strong uniform style is important, a deterministic formatter can enforce it without changing which well-typed computations exist. This is a design ergonomics issue, not a request for free-form whitespace everywhere or removal of the assertion model.

### 8. P2 — The advertised immutable collection idiom has quadratic construction cost and no bulk escape in the source library model

Evidence: map `insert` and `replace` clone the whole native map (`runtime/collections/map.ts:47-67`), then ownership registration also visits its values (`:25-28`). Set `add` similarly clones the native set (`runtime/collections/set.ts:42-47`). The gallery teaches counting words with a fold of individual map operations (`examples/gallery/src/24-map-word-frequencies.can:6-27`). With n distinct words, map copying alone visits 0 + 1 + ... + n-1 prior entries: quadratic work. This is a complexity deduction, not a benchmark claim.

For small request-local data this is fine. For SaaS CSV imports, indexing jobs and analytics batches, the natural source program cannot express the native single-build operation, while mutable aliases are correctly forbidden. Advising authors to prefer collection combinators over recursion does not remove this particular cost because the fold still copies the accumulator.

Add closed bulk constructors/aggregation operations using native Map/Set construction or grouping, with explicit duplicate-key semantics, one final immutable ownership boundary, and normal callback failure/ordering contracts. That follows AGENTS.md's native-operation direction. A scoped internal builder could also work but needs a stronger nonescape contract; do not make ordinary Can values mutable or write a custom persistent map just to avoid native copying. Evaluate native bulk adapters before designing a general mutable-builder language feature.

## What already works, and why I would preserve it

- **Nominal immutable records plus closed variants are a good business-domain base.** Same-shape records cannot mix accidentally; constructor fields and updates keep exact type identity. `runtime/data.ts:23-66` uses native objects/arrays and freezing with a compiler-private nominal identity, rather than implementing a second data engine. Copy-update evaluates a fresh record while sharing immutable children. Abstract construction boundaries are the missing layer, not a reason to adopt structural records everywhere.
- **Closed unions are explicit, finite and checkable.** Variant flattening rejects duplicate leaves and variant-only recursion; inhabitation uses a least fixed point rather than optimistic recursion (`compiler/internal/types/inhabitation.go:10-119`). Optional values are ordinary `none`/`some<T>` records. Recursive records via option or arrays are enough for finite business trees. Disallowing arbitrary primitive leaves avoids ambiguous role matching such as two semantically different strings; wrapper records are useful domain names.
- **Pattern analysis has genuine substance.** The usefulness matrix handles nested constructors, arrays, integer intervals and alternatives with an explicit work budget. Fixing bare-name intent would strengthen an otherwise valuable mechanism. Guards, open variants and arbitrary pattern predicates are not prerequisites for useful application state machines.
- **Error data and emitted domain failures have a coherent distinction.** An error record may be returned as successful data, whereas a terminal error construction emits it. Explicit matches and relays expose domain failure control flow; `chain` is available to avoid arbitrary levels of nesting. Do not replace that with implicit throwing because a few convenience cases are verbose. Error-row abstraction can preserve explicit public bounds.
- **Native numeric contracts are unusually explicit.** Int is bigint, not silently rounded Number; mixed int/float arithmetic is rejected; conversion checks exactness (`runtime/number.ts:58-74`). Exact ratios and half-even rounding have a small native-bigint adapter. Float `Object.is` equality distinguishes signed zero and equates NaN, whereas ordering has ordinary IEEE behavior. This is teachable if examples clearly distinguish equality from total order, and it does not require changing IEEE arithmetic or introducing an imprecise money default.
- **Text exposes native units honestly and already has explicit human-text alternatives.** UTF-16 indexing can produce an unpaired surrogate, but scalar/grapheme/NFC functions validate before operating (`runtime/text.ts:109-118`). The gallery palindrome explicitly limits itself to ASCII. Keep the distinction between code units, scalars and graphemes; do not call each of them a generic character or rewrite native string storage. Teach user-facing truncation through graphemes and validation at wire boundaries.
- **Equality respects data eligibility.** Callables/resources cannot accidentally acquire misleading structural equality through a containing record (`compiler/internal/types/inhabitation.go:131-174`). Native `Object.is`/`Bun.deepEquals` lowerings are deliberate (`compiler/internal/emit/expressions.go:188-225`). Equality-eligible ordinary values and opaque resources should remain separate concepts.
- **Specialization checking is conservative in the right ways.** The complete concrete body is checked; repeated constraints must agree; arrays and generic records do not silently widen. There is no hidden numeric coercion and no guesswork that manufactures an anonymous union. A source-level constraint contract would improve this architecture rather than replace it.
- **Native closures preserve capture and receiver evaluation once.** The current `near` lookup policy is the concern; immutable capture ownership itself is not. Function input/result invariance is conservative but coherent. It may demand wrappers when adapting variant-returning callbacks, which is a usability cost rather than a correctness flaw.
- **Modules protect names and accessible signatures.** File imports, package declarations and public signatures are checked in distinct stages, and forward declaration/import cycles are supported for signatures. Keeping methods with the receiver's defining package avoids incoherent extension resolution. Global uniqueness of method names within a package is restrictive but tolerable compared with the missing abstract-data boundary.

## Tempting changes I reject or defer

I would not restore implicit null/undefined, structural record duck typing, automatic int/float conversion, implicit domain-error propagation, freely mutable captured state, or unchecked native escape hatches. They would weaken the strongest current contracts. I would not add anonymous functions solely to fix capture binding: explicit binding of existing named functions is enough. I would not demand a general trait/effect/HKT system all at once; begin with the narrow abstraction failures above. I would not declare UTF-16 or bigint intrinsically unsuitable for SaaS: their present native contracts and explicit adapters are largely sound.

Likewise, I would not remove mandatory assertions or native AI forms as a shortcut for source brevity. Their verification and runtime implications are owned by separate reviewers. This review finds no need to revisit grouped AI state merely to improve ordinary callable binding. Whether the browser/runtime/platform surface fulfills full-stack ambitions is also a separate evaluation; this report concerns the quality of the core language when composing substantial application code.

## Suggested decision order

First settle the semantic promises: unambiguous pattern binding, transparent versus abstract nominal values, and the meaning of variant identity. Then settle reusable API contracts: finite error-row abstraction and constrained versus template generics. Finally remove unnecessary friction from captures/layout and provide native bulk collection paths. These changes address both trustworthiness and extensibility without sacrificing Can's distinctive core.
