# P03: bounded composition of public generic functions

24 September 2026. This is a design proposal for the selected P03 outcome, not a description of implemented behavior. The Can spelling in the examples is **existing source grammar** (`provides`, `<item>`, `call`, `asserts`); the proposed change is the static meaning of a call whose type argument contains a public caller's opaque parameter. No new syntax, generic kind, trait search, or error-set parameter is proposed.

## Current boundary and required result

[DI-04](../../../preparation/accepted-technical-decisions.md#di-04--explicit-operation-inputs-for-exported-generics) requires an exported body to check once under symbolic parameters and requires written callable or dictionary inputs for operations that are not valid for every substitution. The current checker does this in `check/exported_generics.go`: symbolic parameter identities are declaration-qualified and sealed, and the checked symbolic region is discarded. `check/specialize.go` currently permits an opaque generic call only for identical self-recursion. Every other generic call with an opaque argument is rejected, including a call to a separately valid public identity function. Private generic bodies remain templates checked at their concrete applications. `types.SpecializationKey` rejects opaque arguments; emitted functions come only from the concrete `Program.Functions` list. These are source facts, not decisions made here.

The [reconciliation](../../../post-upgrade-reconciliation-2026-09-24.md) and [disposition](../../../post-upgrade-dispositions-2026-09-24.md) select a bounded fix: a public generic may compose with a validated public generic while opaque values remain opaque. The review's [minimal reproducer](../post-upgrade-review-961f921/independent-core.md#c1--broad-library-blocker-exporting-a-generic-helper-destroys-safe-composition) fails today only after export. The declaration proof must not turn a private template into a universal contract or create a fake concrete instance.

## Proposed rule

Treat a public generic declaration as a universally quantified callable contract only after its symbolic body has been validated. For a call from public `F<A...>` to public `G<B...>` where a `B` contains any opaque `A`, resolve `G` by canonical declaration identity, substitute each `B` into **G's already checked signature**, and check the ordinary argument, result, receiver, named input, and finite `emits` obligations against that substituted signature. The body of `G` is not rechecked using `F`'s opaque types. Only the signature substitution is proof reuse. An opaque call to a nonpublic generic remains invalid: concrete-only template checks establish no claim about all types.

Proof reuse is permitted only for a callee whose validation is committed, or for a declaration in the same explicitly validated recursion component described below. The checker should record a declaration-only symbolic call edge carrying the resolved callee, source span, ordered type expressions and substituted callable contract. This contract needs a call-site-local proof identity: `G`'s own symbolic parameter identity is distinct from `F`'s, while two calls to `G` may substitute different types. It must **not** call the concrete instance creator, use `SpecializationKey`, append `ProgramFunction`, or publish an emission target during symbolic checking. A failed symbolic check must invalidate the proposed component and report the callee call span with a related callee declaration. A callee body failure remains located at the callee, even when a consumer's concrete argument would happen to make that body legal.

This is a narrow extension of DI-04: public bodies may forward opaque values through validated signatures and known composites; they still cannot use arithmetic, ordering, equality, direct field/index/method access or codec/native operations requiring a concrete layout on the bare opaque value. A field of a known `box<item>` can be projected because the record structure is known while `item` remains opaque. Any actual operation on `item` still needs an explicit named callable or dictionary input. Current finite nominal error bounds remain exact; no `emits [item]` or inferred error parameter follows from this rule.

## Concrete source cases

**Positive, across packages.** These are complete Can declaration files in existing grammar, but the symbolic `pass`/`wrap` calls are rejected by today's checker. A runnable project also needs its ordinary manifest and entry function. Each `asserts` row requests a normal concrete application. The final consumer can live in a third package.

```can
package core
    provides [identity]
    uses []
fn item identity<item>
    emits []
    given
        item value
    asserts
        number: 3 => ok 3
    ok value
```

```can
package helpers
    provides [pass, wrap, box]
    uses [core]
record box<item>
    item value
fn item pass<item>
    emits []
    given
        item value
    asserts
        number: 3 => ok 3
    ok call core::identity<item>(value)
fn box<item> wrap<item>
    emits []
    given
        item value
    asserts
        number: 3 => ok box(3)
    ok box(call pass<item>(value))
```

`pass` establishes the simple helper chain. `wrap` establishes a nested call under a nominal composite: the known `box` constructor carries, but does not inspect, `item`. A consuming `call helpers::wrap<owner::token>(token)` must resolve the owner type through its canonical package identity; it must not gain authority to construct or inspect that token outside its owner. A same-spelled type in another package remains distinct. The public signature visibility and owner-construction checks already apply independently of this call rule. The example uses `int` assertions for concrete evidence; a complete acceptance fixture must add a cross-package owner-valued call and inspect the generated TS.

**Negative, callee body edit.** If `core::identity` is changed to `ok value + value`, the public `identity` declaration fails under DI-04 even though an `int` assertion could pass. The caller cannot certify the callee by supplying `int`. If the algorithm truly needs addition, its public signature must add a named `callable item (item, item) emits [] plus` (or a named dictionary field), and `pass` must pass that operation explicitly; its own callers must then repair their signatures/calls.

**Negative, private template.** A private `double<item>` with `ok value + value` can be valid at a concrete `int` application under today's template policy. A public `pass<item>` calling `double<item>(value)` must fail at that call, even if `pass` has an `int` assertion. Exporting `double` then makes its own declaration fail, as DI-04 requires. This deliberately preserves the current visibility distinction; changing that distinction needs a separate decision.

```can
package local
    provides [pass]
    uses []
fn item double<item>
    emits []
    given
        item value
    asserts
        number: 3 => ok 6
    ok value + value
fn item pass<item>
    emits []
    given
        item value
    asserts
        number: 3 => ok 6
    ok call double<item>(value)
```

**Positive, stationary mutual recursion.** For public `first<item>` and `second<item>` with identical parameter positions, `first` may call `second<item>(value, true)` and `second` may call `first<item>(value, true)` behind a Boolean base case. The component is admitted only after *both* bodies validate symbolically. A self call with identical parameters keeps the existing behavior. No termination guarantee is claimed.

```can
package pair
    provides [first, second]
    uses []
fn item first<item>
    emits []
    given
        item value
        bool stop
    asserts
        base: 3, true => ok 3
    match stop
        false => ok call second<item>(value, true)
        true => ok value
fn item second<item>
    emits []
    given
        item value
        bool stop
    asserts
        base: 3, true => ok 3
    match stop
        false => ok call first<item>(value, true)
        true => ok value
```

**Negative, expanding cycle.** Public `a<item>(item value)` calling `b<item[]>([value])`, with public `b<item>(item value)` calling `a<item>(value)`, creates `a<T> → b<T[]> → a<T[]>`. It must be rejected at declaration checking, even if a concrete assertion has a base case and even before concrete instance discovery reaches its finite limit. Nested acyclic `F<T> → G<box<T>>` is admissible when both declarations validate; the constructor alone is not an error. The cycle's changing type vector is the reason for rejection.

```can
package grow
    provides [a, b]
    uses []
fn void a<item>
    emits []
    given
        item value
        bool stop
    asserts
        base: 3, true => ok
    match stop
        false => relay call b<item[]>([value], true)
        true => ok
fn void b<item>
    emits []
    given
        item value
        bool stop
    asserts
        base: 3, true => ok
    match stop
        false => relay call a<item>(value, true)
        true => ok
```

## Recursion and proof order

Build symbolic-call dependencies among public generic declarations and validate them in dependency order. A declaration in progress is not a validated callee. For a strongly connected component, provisional signatures may be used to type its calls, but no member's proof is committed until every body and every internal call has checked. This is ordinary mutually recursive checking under declared signatures, with one deliberately conservative extra rule: **on every internal edge, each type argument must be either a bare formal parameter of the caller or a closed concrete type**. “Closed” means its resolved sealed graph contains no opaque parameter, including within nominal or callable arguments. Parameter permutation, duplication and dropping are safe under this rule: a concrete run can only select from its initial finite set of argument types plus the finite concrete types written on the component's edges. No edge manufactures a new type constructor around a variable. This bounds concrete instance discovery within the component; the symbolic proof itself checks each declaration once and commits the component together. It does not claim runtime termination. The rule conservatively rejects some cycles that might stabilize after constructing a type, because proving that requires a more involved composed-transition analysis. A nested `F<T> → G<box<T>>` call is allowed when it is acyclic: `G` is checked first, and one concrete `F<X>` application reaches `G<box<X>>` without feeding that larger type back into `F`.

The graph must use resolved declaration IDs, not short names or source spelling. Include all public-generic call edges in the component graph, including calls whose arguments are closed, so a concrete edge cannot hide a return path. Diagnose a prohibited cycle with its sequence of call sites and the first constructor applied to an opaque argument. A cycle with an invalid body must fail as a unit; no other declaration can consume any member as validated proof. Private template callees cannot be inserted into a public proof component. Concrete applications still undergo per-instance body checks and the existing recursive specialization protections. The existing finite discovery/type-graph limits remain resource guards for shapes this bounded rule does not prove terminating; they are not a substitute for rejecting the expanding symbolic cycle above.

## Concrete lowering and acceptance gate

Once symbolic validation succeeds, check every reached **concrete** application as today. At a concrete `F<int>` call to `G<box<int>>`, create or reuse `G<box<int>>` with the ordinary canonical specialization key, record the concrete target in the checked IR, and emit that actual function in its declaring ESM module. The call becomes an ordinary direct TS/JS call with existing completion and assertion-context handling. The symbolic call edge itself has no emitted function, runtime dictionary, type test, or opaque-key stand-in. Explicit callable inputs lower to native JS function calls; Can arithmetic inside a concrete supplied callable remains native JS arithmetic. The emitter should fail on a missing concrete target instead of silently rendering a symbolic identity.

Acceptance requires fresh checker and emitter tests for: two- and three-level public chains across packages; direct and nested opaque arguments; exact error bounds and an explicit callable through a chain; owner/sealed type identity across a vendor package; private-template rejection; an illegal callee arithmetic edit diagnosed at the callee; stationary self/mutual recursion; and direct plus mutual expanding cycles. Inspect generated TS to verify that only concrete reached instances and ordinary calls exist. Run the relevant compiler tests and one full project build after implementation. These are proposed gates, not tests run for this document.

For this proposal, all five fenced Can declaration files passed the current `canlc parse` command. Parsing checks their grammar only; the public composition and cycle examples intentionally await the proposed checker rule, and no project build or emitted TS is claimed here.

## Design limit

This proposal reuses public parametric proofs only for authored generic functions. Catalogue operations such as codecs that require concrete layout keep their existing opaque-argument rejection. It does not add error-set generics, trait constraints, implicit operations, a public-template spelling, generalized polymorphic recursion, or a promise that recursion is stack safe. Those boundaries keep P03 aligned with the selected disposition and DI-04 while making ordinary helper extraction stable.
