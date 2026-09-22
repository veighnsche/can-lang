**Yes. The best things to borrow are Dafny/F*’s specification and proof features, plus F#’s explicit value binding and data modelling—not their convenience syntax.**

**The cheapest useful addition is typed immutable bindings. The biggest strategic addition is outcome-specific preconditions and postconditions.**

Your design should be cruel about **unmade decisions, unhandled cases, and unsupported claims**. Making an agent repeat the same arithmetic or flatten every record into arguments is not useful cruelty; it creates more opportunities for inconsistency.

Using the uploaded a13 rules as the baseline, these are my priorities. **All new `.can` syntax below is proposed; sketches are fragments, not programs the current compiler accepts.**

## Ranked shortlist

| Priority | Borrow                                            | From            | What it adds                                                                             | Relative proof cost                                   |
| -------- | ------------------------------------------------- | --------------- | ---------------------------------------------------------------------------------------- | ----------------------------------------------------- |
| **1**    | Explicit immutable local bindings                 | F# / F*         | Name a computation and evaluate it once without extracting another function.             | Low                                                   |
| **2**    | Outcome-specific preconditions and postconditions | Dafny / F*      | Prove relationships between inputs, results, and error payloads for all admitted inputs. | Significant, but foundational                         |
| **3**    | Closed tagged unions                              | F# / F* / Dafny | Represent alternatives without booleans, sentinels, or invalid field combinations.       | Moderate                                              |
| **4**    | Explicit refinement types                         | F* / Dafny      | Carry checked value constraints across function boundaries.                              | Moderate after #2                                     |
| **5**    | State-transition and unchanged-state contracts    | Dafny           | Specify what changes—and what remains unchanged—on success and each error.               | Moderate after #2                                     |
| **6**    | Ghost lemmas with checked erasure                 | Dafny / F*      | Reuse proofs without shipping proof computations.                                        | Moderate to substantial                               |
| **7**    | Units of measure                                  | F#              | Distinguish milliseconds, seconds, bytes, counts, and derived quantities.                | Moderate; a smaller numeric-brand version comes first |

The ordering is my design recommendation, not a measured implementation estimate. Several reinforce items already on your wishlist; they are not newly verified compiler bugs.

---

## 1. Typed immutable bindings: stop forcing computation into helper functions

F#’s `let` expressions associate a name with a value and give that name a lexical scope. Borrow that mechanism while rejecting optional annotations, destructuring shortcuts, and inferred generic parameters. ([Microsoft Learn][1])

For can-lang, make a binding **one expression with an explicitly nested continuation**, preserving the single-expression-body rule:

```can
let total: int = left + right
  match total <= limit
    true => Ok(value = total)
    false =>
      math.over_limit(value = total, limit = limit)
```

This would mean: evaluate `left + right` once, bind the resulting `int` to `total`, then evaluate the indented expression.

The current rules require expression bodies, but that does not logically require banning local binding expressions. The earlier reviewer guide explicitly says there is no `let`; this is a genuine addition rather than a formatting change.  

**The can-lang restrictions should be strict:** mandatory type, no reassignment, no shadowing, no recursive bindings, and initially only pure value expressions as initializers. Effectful calls would still use `match call`; `let` must not become a way to hide effects or unwrap `Ok`.

**Proof cost:** lexical binding and scope checks, exact initializer typing, evaluate-once semantics, and preservation of binder identity through termination checking and TypeScript lowering. A name matching the spelling of a ranking parameter must not be mistaken for that parameter.

**Agent benefit:** one expression to update, one typed intermediate value to inspect, and no artificial helper interface or extra decision table merely to reuse a calculation.

**Human cost:** every intermediate remains fully annotated and scope-explicit. But do not add inconvenience solely to make this feature unpleasant.

**Acceptance test:** change the calculation once and verify that every use observes the same bound value. Reject shadowing and reassignment. Check that emitted code computes the initializer once.

**Recommendation: implement this first.**

---

## 2. Outcome-specific contracts: prove more than selected examples

This is the most important borrowing.

Dafny’s `requires` and `ensures` describe obligations at function entry and exit. Its postconditions must hold for every invocation satisfying the preconditions, not merely the examples in a test suite. F* can express similarly precise input/output relationships through dependent function and refinement types. ([Dafny][2])

Your a13 system has executable decision tables, exhaustive outcomes, and branch coverage. Those are useful, but they do not establish arbitrary input/output properties for every possible input.  

### Adaptation: contracts indexed by the actual outcome

For a maximum function:

```can
ensures
  on Ok result
    result.value >= left
    result.value >= right
    match result.value == left
      true => true
      false => result.value == right
```

Each clause is a required predicate. Together, these say that the result is at least both inputs **and is one of them**. A function returning an arbitrarily large constant cannot satisfy the contract.

For a validator:

```can
ensures
  on Ok result
    result.value == value
    result.value >= lower
    result.value <= upper
  on validation.invalid_bounds err
    lower >= upper + 1
    err.lower == lower
    err.upper == upper
```

A complete declaration would include every declared error outcome. This excerpt illustrates the distinction between **declaring an error kind** and **specifying when that error is permissible and what its payload means**.

### Keep preconditions separate from input validation

A private arithmetic kernel might require:

```can
requires
  exponent >= 0
```

Its callers must prove that condition. A public function accepting untrusted integers should still validate negative input and return its declared error.

**Do not let the agent “fix” a failing proof by silently narrowing the public input domain.** Changes to `requires` and `ensures` are contract changes and should participate in revision review.

**Proof cost:** verification conditions at each call and each exit, with a deliberately limited initial logic. Start with booleans, record projections, equality, and linear integer arithmetic. Unsupported propositions and inconclusive verification fail closed; they do not become runtime assertions or assumed facts.

Checking that a declared condition follows from known facts is verification. It is not permission to invent missing types, preconditions, or effect declarations.

**Agent benefit:** the compiler can reject implementations that overfit examples. Callers can reason from a callee’s verified contract instead of reopening its implementation.

**Human cost:** even apparently obvious facts must be stated, and an algorithm may need intermediate proof steps.

**Acceptance test:** keep the existing examples green while introducing a bug outside their inputs; the universal contract must reject it. Also reject an impossible precondition masquerading as a proof: require concrete admitted test inputs for executable functions.

**Important boundary:** this does not solve the scripted-import issue by itself. A scripted response consistent with a contract still is not execution or verification of the provider body.

---

## 3. Closed tagged unions: make invalid combinations unrepresentable

F# discriminated unions describe a value as exactly one named case, each with its own payload. Its `option` type is one application; recursive trees are another. ([Microsoft Learn][3])

Borrow the first, small part: **finite, monomorphic, closed unions**.

```can
variant Login__State rev 1
  case Login__Anonymous()
  case Login__Authenticated(
    session: Auth__Session
  )
  case Login__Locked(
    user_id: str
    remaining_seconds: int
  )
```

Matching remains explicit:

```can
match state
  on Login__Anonymous _ =>
    Ok(message = "Sign in")
  on Login__Authenticated authenticated =>
    Ok(message = authenticated.session.user_id)
  on Login__Locked locked =>
    Ok(message = locked.user_id)
```

This is preferable to a record containing `authenticated: bool`, `locked: bool`, a possibly meaningless session, and a sentinel user ID.

**Do not import the whole F# union experience.** Require qualified case names, named payload fields, exact constructor types, and exhaustive matches without catch-all arms. Start without recursive variants or generics.

**Proof cost:** constructor membership, exact payload checking, exhaustive case coverage, and tag/payload-preserving TypeScript representation. Adding a case must invalidate every now-incomplete match.

**Agent benefit:** the compiler enumerates the meaningful states and the exact work required when a state is added. There is less dependence on comments explaining which fields “count” in which mode.

**Human cost:** adding one case can force edits across many handlers. That is useful cruelty: it exposes decisions that otherwise remain implicit.

**Acceptance test:** add `Login__Expired`; every incomplete consumer fails. Accessing `session` in the locked case also fails.

**Stdlib payoff:** optional SQL rows, HTTP methods, form states, UI messages, validation results, and later resource outcomes. General variants extend your existing exhaustive-outcome discipline rather than introducing a competing exception mechanism. 

---

## 4. Refinement types: keep the fact that validation established

F* refinement types describe values satisfying predicates, such as non-negative integers. F* also uses refinement subtyping to introduce and eliminate these refinements. **Borrow the predicates, not the implicit conversions.** ([FStar][4])

For can-lang:

```can
refine Retry__Attempts is int rev 1
  invariant
    value >= 0
    value <= 30
```

This type would mean more than “an integer with a different name.” Every inhabitant must satisfy the stated bounds.

A checked constructor validates an ordinary integer and returns either a `Retry__Attempts` value or a complete typed error. A proof-backed constructor may accept an expression only when the checker verifies those same bounds.

**Neither construction nor projection should be implicit.** Also keep refinements distinct from secret brands: authorizing explicit numeric projection must not create a generic declassification path for passwords or safe-HTML values.

Your existing brand mechanism is nominal and string-backed; constructor control is already explicitly deferred in a13. Refinements should build on that work, not bypass it with a new unchecked construction gate.  

**Proof cost:** predicates restricted to the supported verification logic; every construction establishes the invariant; mutation cannot invalidate it; conversions remain explicit. Arithmetic on a refined value must not silently preserve its refinement.

**Agent benefit:** validation produces reusable evidence. A downstream function receiving `Retry__Attempts` does not need to rediscover why the value lies between zero and thirty.

**Human cost:** an increment may require proving the result is still in range or handling a constructor failure.

**Acceptance test:** constructing attempts from `31` fails. Incrementing a valid value of `30` cannot return another `Retry__Attempts` without handling the boundary.

**Avoid a trap:** this proves a range, not a resource budget. The existing backoff cap is a particular cost policy; a refined integer alone does not prove an operation is cheap. 

---

## 5. State contracts: distinguish permission to mutate from the promised mutation

Dafny has frame specifications and two-state expressions such as `old(...)` for reasoning about entry and exit state. ([Dafny][5])

can-lang already declares state authority through `effects`. **Keep that mechanism. Do not add a second competing `modifies` manifest.** What is missing from that declaration is the relationship between the old state, new state, and returned outcome. 

For the quota counter:

```can
ensures
  on Ok result
    Quota__used == old(Quota__used) + amount
    result.used == Quota__used
    result.remaining == quota - result.used
  on validation.negative_value err
    Quota__used == old(Quota__used)
  on validation.invalid_bounds err
    Quota__used == old(Quota__used)
  on validation.out_of_range err
    Quota__used == old(Quota__used)
```

Now the contract says that failure leaves the counter unchanged—not merely that the function was allowed to read and write it.

Start with the existing private scalar cells. That avoids introducing arbitrary heap aliasing as part of this feature.

**Proof cost:** explicit entry-state snapshots in the verification model, symbolic state transitions, verification of every outcome’s postcondition, and composition through callee contracts. An invariant must be established by initialization and preserved by every admitted operation—not just tested from the initial store.

That matters because the state specification explicitly says production histories are outside the current fresh-store test argument. 

**Agent benefit:** safe refactoring of validation and mutation order. Moving a write before a failing check becomes a verifiable semantic change.

**Human cost:** error paths need precise state guarantees as well as error payloads.

**Acceptance test:** mutate the counter before returning `validation.out_of_range`; reject the implementation even when its returned error payload is correct.

**Scope boundary:** “unchanged at exit” does not mean “never touched,” and it does not roll back a database or network operation. External transaction semantics need their own contract.

---

## 6. Ghost lemmas: let an agent write proofs without shipping them

Dafny lemmas are proof-only declarations that cannot change state and are erased from executable code. F* similarly distinguishes ghost computations and prevents them from influencing retained computation. ([Dafny][6])

A small can-lang adaptation could look like:

```can
lemma range__width(lower: int, upper: int) rev 1
  requires
    upper >= lower
  proves
    upper - lower >= 0
  proof
    normalize_linear
```

Here, `normalize_linear` would be a **specific, bounded checker rule**: normalize the premise and conclusion into linear arithmetic form and verify the claimed step. It would not mean “ask a model whether this seems true.”

Later lemmas could contain explicit case splits and induction, but each extension needs its own termination and proof rules.

**Proof cost:** a small checked proof calculus, explicit lemma dependencies, termination for recursive proofs, and an erasure check. Ghost values must not decide runtime branches, construct runtime results, select host calls, or mutate state. Ban unchecked `assume`/`admit` escape hatches in blessed code.

**Agent benefit:** reusable proof fragments and localized failure messages. Instead of endlessly rewriting an implementation to please a verifier, the agent can supply the missing argument.

**Human cost:** maintaining a small theorem alongside a small function can feel disproportionate. For an agent-oriented language, that work is justified when it makes the guarantee explicit and reusable.

**Acceptance test:** replace the conclusion with `upper - lower >= 1`; reject it. Attempt to return a ghost-computed value at runtime; reject that too. Changing a valid proof without changing executable code must not change production behavior.

This also provides a disciplined answer to your fallible-recursion coverage problem: an impossible handler could carry a checked contradiction proof. It must never receive an unverified “ignore coverage” annotation. The a13 split-worker convention exists specifically to avoid such unreachable error arms today. 

---

## 7. Units of measure: prevent correct arithmetic on the wrong quantities

F# associates numeric values with units and checks that arithmetic has consistent dimensions. Its units are compile-time information rather than runtime wrappers. ([Microsoft Learn][7])

An explicit can-lang version might use:

```can
unit Time__Second rev 1
unit Time__Millisecond rev 1
unit Data__Byte rev 1

type Retry__Delay rev 1 (
  value: int unit Time__Millisecond
)

type Http__BodyLimit rev 1 (
  value: int unit Data__Byte
)
```

A function expecting milliseconds must reject seconds until an explicitly named conversion is called. A byte count must not be accepted merely because it shares the same integer representation.

**Do not start by building a full dimensional-algebra language.** First ship the useful subset through int/dec-backed nominal types and explicit conversion functions. Add compound dimensions when a real program needs rates, sizes, or products whose units must compose.

**Proof cost:** unit compatibility, explicit conversion ratios, and exactness of the numeric conversion. Converting integer milliseconds to integer seconds must reject a non-integral result or use an explicitly named rounding operation.

**Agent benefit:** units become part of the checked interface instead of a naming convention such as `_ms` or `_bytes`.

**Human cost:** `1000` is no longer enough information at a quantity boundary; the unit and conversion decision must be explicit.

**Acceptance test:** passing `Time__Second` into a millisecond parameter fails. Converting 1,500 integer milliseconds to exact integer seconds fails rather than returning one.

---

## Two existing roadmap items these languages can sharpen

### Opaque interfaces, not merely “restricted seal”

F# signatures can hide record fields and union constructors from consumers. Borrow that abstraction boundary, but keep can-lang’s authoritative contract in the `.can` source rather than creating a second manually synchronized signature file. ([Microsoft Learn][8])

For HTML, the desired guarantee is not just “the consumer cannot write `seal Html__Safe(...)`.” It is:

> The consumer cannot construct, inspect, or bypass the representation except through the explicitly exported operations.

That is a concrete direction for the constructor-controlled-brand work already listed in a13. 

### Richer explicit termination measures

Dafny supports lexicographic measures and measures based on finite structures. Borrow those **when collection/tree programs need them**, while retaining explicit annotations and rejecting its termination opt-out. ([Dafny][9])

For example:

```can
decreases
  lexicographic
    remaining_pages
    remaining_items
```

The checker must prove that every recursive edge decreases that tuple in a well-founded domain. It must not guess the tuple.

This could replace your current hard-coded unit-decrement shape with a more general proof obligation, but it does **not** replace the need for an efficient division kernel or a cost model. The current requirement is specifically a guarded integer parameter with `p - 1`. 

---

## What not to steal

| Avoid                                                  | Better can-lang choice                                                            |
| ------------------------------------------------------ | -------------------------------------------------------------------------------- |
| Optional annotations and inferred public contracts     | Require every public type, error set, capability set, and refinement explicitly. |
| Implicit refinement introduction/elimination           | Explicit checked construction and explicit projection.                           |
| Multiple equivalent convenience syntaxes               | One spelling per binding, construction, call, and proof step.                    |
| A general-purpose tactic or macro language immediately | A small fixed proof calculus with bounded checking.                              |
| Termination or proof escape hatches                    | Reject the unproved program; no “verified except this part” blessed build.       |
| Weak contracts that merely describe the happy path     | Outcome-specific relationships, complete error payloads, and state guarantees.   |

The risk is not only an incorrect implementation. An agent can also produce a **weak specification that an incorrect implementation satisfies**. Dafny’s own tutorial demonstrates this with an absolute-value specification that allows a constant result until its postconditions are strengthened. ([dafny.org][2])

Keep the decision tables, therefore. Use **examples to anchor intended behavior, universal contracts to constrain all admitted behavior, and an independent check that TypeScript preserves those semantics**. None substitutes for the other two.

**My next three feature specs would be: typed immutable bindings; outcome-specific contracts together with their verifier; then closed tagged unions.** Those provide the strongest immediate foundation for the standard library without turning can-lang into a full dependent-type language.

[1]: https://learn.microsoft.com/en-us/dotnet/fsharp/language-reference/functions/let-bindings "let Bindings - F# | Microsoft Learn"
[2]: https://dafny.org/latest/OnlineTutorial/guide "Getting Started with Dafny: A Guide | Dafny Documentation"
[3]: https://learn.microsoft.com/en-us/dotnet/fsharp/language-reference/discriminated-unions "Discriminated Unions - F# | Microsoft Learn"
[4]: https://fstar-lang.org/tutorial/book/part1/part1_getting_off_the_ground.html "Getting off the ground — Proof-Oriented Programming in F* documentation"
[5]: https://dafny.org/latest/DafnyRef/DafnyRef "Dafny Documentation"
[6]: https://dafny.org/latest/OnlineTutorial/Lemmas "Lemmas and Induction | Dafny Documentation"
[7]: https://learn.microsoft.com/en-us/dotnet/fsharp/language-reference/units-of-measure "Units of Measure - F# | Microsoft Learn"
[8]: https://learn.microsoft.com/en-us/dotnet/fsharp/language-reference/signature-files "Signature files - F# | Microsoft Learn"
[9]: https://dafny.org/latest/OnlineTutorial/Termination "Termination | Dafny Documentation"
