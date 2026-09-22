# Verdict

**The language has gained real expressiveness, but its strongest correctness claims outrun its specified guarantees.** The main problems are not missing async or missing syntax. They are changes in what “proved” means when crossing from evaluator to TypeScript, from local calls to imported calls, and from complete values to error-kind-only expectations.

My re-rating is **7/10 → 8/10 for shipped, scenario-checked expressiveness**. That is **not** a soundness endorsement. The uploaded a05 plan provides the original 7/10 baseline, but no numerical subscore rubric; the more granular assessments below are mine, not reconstructed panel scores. *Source: `docs/archive/a/a05-expressiveness.md`, “Consequences.”* 

This is a language/specification audit. I did not execute the can-lang compiler. I distinguish **findings supported directly by the files** from **deduced counterexamples whose compiler acceptance remains unverified**. Example citations refer to the real repository paths embedded in `docs/ALL_EXAMPLES.can` (removed with the archived design records).

---

# Task 1 — Audit

## 1. Numeric exactness does not survive the documented TypeScript mapping

**Severity: soundness hole in the source-to-target contract.**

`REQUIREMENTS.md`, R10, promises arithmetic that is:

> “exact or loud”

But `docs/archive/a/a06-arithmetic.md`, “TS emit,” specifies native operators over `number` and claims:

> “`dec` within 15 significant digits”

is an exact mapping. These claims are incompatible.  

A counterexample requires no large numbers:

```can
fn audit__sum(a: dec, b: dec) -> Audit__Value rev 1
  emits []
  tests
    fractions(a = d"0.1", b = d"0.2") =>
      Ok(value = d"0.3")
=
  Ok(value = a + b)
```

This function fragment follows a06’s arithmetic rule. Its evaluator result is explicitly supported by a06’s “Eval semantics.” Under the specified target mapping, however, the computation becomes `0.1 + 0.2`. I evaluated that expression locally: it produces `0.30000000000000004`, and comparison with `0.3` is false. TypeScript’s `number` is the floating-point numeric type, not an exact decimal type. *Sources: a06, “Eval semantics” and “TS emit”; TypeScript Handbook, “Number.”*   ([TypeScript][1])

**The 15-digit caveat does not rescue this:** the input values and the mathematical result each have one significant digit.

The integer boundary also interacts with termination. Under the documented `number` mapping, I evaluated:

```text
1000000000000000000 - 1 == 1000000000000000000
```

as true. That value is within the evaluator’s stated `int64` domain, although outside its documented exact TypeScript domain. Consequently, an accepted source-level decrement need not decrease the emitted value. This is a **deduced production counterexample**, not a tested can-lang compilation. *Sources: a06, “Eval semantics” / “TS emit”; a08, “Loop shape.”*   

**Conclusion:** documentation of a numeric boundary is not enforcement of that boundary. Either the target preserves source arithmetic, or admissible source programs must be restricted accordingly.

---

## 2. The termination argument proves a narrower property than the language advertises

### 2.1 Cross-module recursion escapes the stated proof

**Severity: soundness hole in a whole-program termination claim; deduced acceptance risk.**

R7 says:

> “direct self-recursion only”

However, a07 restricts its cycle check to **same-file edges**, and a08’s halting argument explicitly relies on the fact that **foreign calls never execute during tests**. *Sources: `REQUIREMENTS.md`, R7; a07, “Termination”; a08, “The halting argument.”*   

Consider two modules with properly pinned imports, ordinary record declarations, and these function fragments:

```can
fn alpha__run(n: int) -> Alpha__Value rev 1
  emits []
  tests
    sample(n = 0) => Ok(value = 0)
=
  match call beta__run(n)
    given
      sample => [Ok(value = 0)]
    on Ok result => Ok(value = result.value)
```

```can
fn beta__run(n: int) -> Beta__Value rev 1
  emits []
  tests
    sample(n = 0) => Ok(value = 0)
=
  match call alpha__run(n)
    given
      sample => [Ok(value = 0)]
    on Ok result => Ok(value = result.value)
```

Under the written rules, each local call graph is acyclic, every test consumes its script, and every arm executes. Linking the actual provider bodies creates an unproved recursive cycle.

**Inference:** unless there is an additional whole-program cycle rule absent from these documents, moving a call across a file boundary can bypass the recursion policy.

The sandbox termination argument can remain valid. The claim that the deployed program has a termination proof cannot.

### 2.2 Negative-entry failure is part of the mechanism, not merely a diagnostic convenience

a08 permits `p - k` without requiring that the call site establish `p >= k`. Its argument then relies on negative entry causing an evaluation error. *Source: a08, “Loop shape” / “The halting argument.”*  

Thus, the specified theorem is closer to:

> Evaluation cannot continue through infinitely many admitted non-negative recursive entries; it eventually returns or faults.

It is **not**:

> Every accepted invocation returns a declared `Ok` or `emits` outcome.

For example, a guarded function that stops only at zero and subtracts two can enter with one and fault on the next entry. Ordinary branch coverage does not exclude that input.

There is also a concrete verification obligation to check: a08’s implementation description places the negative-entry guard in `evLocalCall`. **That does not establish whether root tests and production entry points receive the same check.** I would test all entry routes; I am not asserting the implementation omits one. *Source: a08, “Implementation.”* 

### 2.3 The shipped “would hang without the proof” demonstration is false

The retry example says:

> “without the proof this shape would hang the build on never”

But its `never` trace is visibly finite:

```text
fuel 2 → down
fuel 1 → down
fuel 0 → exhausted
```

Deleting metadata does not change those expressions or that trace. It changes whether the compiler admits the program. The example demonstrates **proof-gated admission**, not a program that would otherwise diverge. *Sources: `docs/archive/sketches/retry-loop/retry.can`, header and `retry__fetch`; a05, “Consequences.”*   

Replace that demonstration with mutations that remove the base case or make the recursive argument unchanged.

---

## 3. Caller-authored scripts cannot establish provider honesty

### 3.1 `db.down` exposes an unresolved meaning of “exact emits”

**Severity: inconsistency, potentially a soundness hole in the advertised exactness guarantee.**

`FOR_REVIEWER.md`, §3, says:

> “a declared-but-never-raised emit is an error too”

Yet the shipped `db__get_user` declares `db.down` and has only two outcomes: `Ok` and `db.user_not_found`. The supposedly clean provider in `broken-login/db.can` does the same.   

R10 instead says:

> “unraised unstubbed `emits` are errors”

and clients do stub `db.down`.  

There are two interpretations, neither satisfactory:

| Interpretation                                       | Consequence                                                                           |
| ---------------------------------------------------- | ------------------------------------------------------------------------------------- |
| Evidence must belong to the declaring function.      | The shipped lookup violates the stale-emits rule.                                     |
| Consumer stubs count as evidence about the provider. | A consumer can manufacture the evidence that makes a provider’s declaration “honest.” |

**Recommendation:** make `emits` an explicit conservative upper bound. Keep “no undeclared escaping error”; delete mandatory realization of every declared error. That preserves the important safety guarantee while removing the circular evidence rule.

### 3.2 Scripts check outcomes, not whether the right request was made

R8 scripts outcome sequences. It does not specify expected call arguments. *Source: `REQUIREMENTS.md`, R8.* 

**Deduced mutation:** changing the first login lookup from:

```can
match call db__get_user(id)
```

to:

```can
match call db__get_user("the-wrong-user")
```

does not change the selected scripted outcomes under the documented semantics. Its argument remains correctly typed. *Source: `docs/archive/sketches/auth-login/auth.can`, `auth__login` first lookup.* 

This is not a faulty implementation of R8. It is an important property that R8 currently does not establish.

### 3.3 Error payload correctness is deliberately unobserved

`FOR_REVIEWER.md`, §2, explicitly says error expectations check the kind, not the fields. Consequently, changing:

```can
auth.login_failed(user_id = id)
```

to:

```can
auth.login_failed(user_id = "somebody-else")
```

can preserve the existing error expectations.

**Finding:** the language cannot currently express a decision-table assertion that the right user ID was returned in an error. This is an expressiveness gap, not an undisclosed checker bug. *Sources: reviewer guide §2; `auth__login` tests and missing-user arm.*   

---

## 4. Other rule/specification conflicts

| Rule or promise                                                    | Counterexample or unresolved conflict                                                                                                                                                         | Assessment                                                                                                                                                     |
| ------------------------------------------------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **R6:** “A function touching the outside world says so in `uses`.” | `retry.can` has `uses []`, declares `extern net__fetch`, and calls it. R2 itself reserves `uses` for external **can** items.                                                                  | Rewrite R6: `uses` is not a complete external-effect declaration.                                                                                              |
| **R7:** “Each function carries `tests`.”                           | a07 says a function without tests is still a warning; `missing-tests.can` explicitly advertises a yellow squiggle.                                                                            | Mandatory evidence and warning-only absence are inconsistent. This does **not** prove that this particular gallery file builds; it contains other problems.    |
| **R9:** “types + signatures before bodies”                         | `auth__verify`’s signature follows the complete `auth__login` body.                                                                                                                           | An explicitly “to enforce” layout goal remains unmet—not a newly discovered implementation failure.                                                            |
| **R4:** “Provider bumps never break pinned callers.”               | Runtime coexistence of revisions remains open question 1. The example claims old hashes remain addressable, but does not establish retained executable versions or their dependency closures. | Unsupported guarantee, not a verified runtime failure.                                                                                                         |
| **R2:** `provides` names what the file defines.                    | Errors are omitted; private cells have an explicit exemption; module-local externs are included even though cross-module externs are unavailable.                                             | Define whether this is a declaration inventory or an export interface. Currently it is neither cleanly.                                                        |
| **R2:** module `emits` describes errors the module can produce.    | `retry` includes handled `net.down`; its only can function declares only `retry.exhausted`. The module/function relation is still explicitly open.                                            | Do not claim exact module-level escaping effects until the relation is defined.                                                                                |

I would **not** count a07’s original recursion ban versus a08’s amendment, or R8’s original all-calls wording versus its three-callee amendment, as contradictions. Those are identifiable, versioned changes. *Sources: Requirements status; R7–R8.*  

Two additional honesty corrections:

**The counter is not application-bounded.** Its `by: int` parameter and addition have no declared counter limit. “Bounded counter” should mean a specified limit with handling, not merely the existence of an integer representation. *Sources: a09, “Consequences”; `counter.can`, `count__bump`.*  

**Fresh-store semantics provides test isolation; repeated rows demonstrate a regression case.** Those rows do not prove arbitrary production histories. a09 correctly acknowledges that production histories are outside its tables; the reviewer guide should retain that qualification wherever it says the tables “prove” isolation. *Sources: a09, “The isolation argument”; reviewer guide §8.*  

---

## 5. Sibling bugs to hunt—without inventing findings

These are **required adversarial checks**, not claims that the current implementation fails them.

| Proof surface          | Adversarial cases and required result                                                                                                                                                                                              |
| ---------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Arithmetic structure   | `10 - 3 - 2` must be 5; `10 - (3 - 2)` must be 9. `fuel - -1` and `fuel - 1 + 2` must not satisfy the decrease rule. Recognition must use expression structure, not a matching substring. *a06 “Rule”; a08 “Loop shape.”*          |
| Named argument binding | Reordering named arguments must not change which argument supplies the ranking parameter. Duplicate names, missing names, and extra names must fail before any proof is credited. *a08 “Loop shape”; R10 decision-table checks.*   |
| Lexical boundaries     | Strings containing `)`, `=>`, commas, or operator-looking text must remain string contents. Nested `seal` constructions must not alter test-row or argument boundaries. *R1; shipped nested constructions.*                        |
| Entry invariants       | Exercise negative inputs through root tests, local calls, imported calls, and production exports. Also test step sizes larger than the remaining value. *a08 “Halting argument” / “Implementation.”*                               |
| Empty payloads         | A zero-field outcome must not acquire usable fields through nesting or forwarding. Prefer a universal empty-outcome binding rule over a `state__put` special case. *Reviewer guide §8; a09 refinement backlog.*                    |
| Script identity        | Two functions with a test called `happy` must have distinct identities when both reach one helper. The documents describe caller-name flow-through but do not fully specify its namespace. *a07 “Rule” / “Coverage.”*              |

The language-level requirement should be: **every checker reasons over the same bound, typed program structure**. This is a semantic consistency requirement, not a request to review parser implementation choices.

---

## 6. Are the ten rejection reasons convincing?

| Gallery file        | Judgment                                                                                                                                                                                                    |
| ------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `bad-name.can`      | Consistent with the naming policy. It rejects a natural name, `go`, but this is deliberate convention—not a soundness benefit.                                                                              |
| `dangling-test.can` | Convincing: the new test reaches a foreign call without an outcome script.                                                                                                                                  |
| `dead-script.can`   | Convincing as stale evidence, but its yellow-warning description contradicts presenting the gallery as uniformly rejected.                                                                                  |
| `failing-test.can`  | Convincing: the expected success disagrees with the error-producing branch.                                                                                                                                 |
| `foreign-raise.can` | Convincing: forwarding is natural, but the escaping error must be declared. The mistake is the manifest, not forwarding itself.                                                                             |
| `missing-arm.can`   | Convincing: the declared `db.down` outcome lacks a handler.                                                                                                                                                 |
| `missing-rev.can`   | Convincing under the explicit-version policy.                                                                                                                                                               |
| `missing-tests.can` | Convincing in principle, inconsistent in severity. It also contains a `happy` script with no corresponding test, so it does not isolate one mistake.                                                        |
| `stale-emits.can`   | **The weakest rejection.** Conservative effect declarations are reasonable. Furthermore, `auth.stale` is not declared as an error, so the specimen confounds “unknown error” with “unused declared error.”  |
| `unknown-call.can`  | Convincing: no resolved callee means no contract to check.                                                                                                                                                  |

The “one squiggle” presentation should not become a language rule that conceals independent mistakes. For example, `failing-test.can` also lacks a green test reaching its `db.down` arm. Suppressing dependent diagnostics can be sensible; calling the specimen single-fault is not. *Sources: that specimen; R10’s green-table coverage rule.*  

---

## 7. Re-rating against a05

**Overall: 7 → 8 for demonstrated, scenario-checked expressiveness.**

| Area                          | Movement                  | What actually earns credit                                                                                                                                                                                            |
| ----------------------------- | ------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Arithmetic                    | **Up, narrowly**          | Arithmetic appears in shipped login and counter logic. No credit for exact-decimal TypeScript semantics: that claim fails above.                                                                                      |
| Helpers                       | **Up substantially**      | Both login success paths execute one shared verifier. This directly satisfies the a05 duplication target.                                                                                                             |
| Iteration                     | **Up, sandbox-qualified** | The retry example exercises immediate success, delayed success, exhaustion, and zero fuel through one recursive body. Credit the admitted loop and its restricted argument—not the false “would hang” demonstration.  |
| State                         | **Up, scenario-local**    | `count__twice` observes two writes within one evaluation; the specification supplies fresh-store isolation between tests. Production-history correctness earns no additional credit.                                  |
| Deferred features             | **Unchanged**             | No shipped credit for async, int-backed brands, or the other explicitly deferred capabilities.                                                                                                                        |
| End-to-end proof preservation | **No upward movement**    | The numeric and cross-module issues prevent upgrading this claim merely because more programs can be written.                                                                                                         |

---

# Task 2 — Prioritized improvements

The following sketches are proposals unless described as existing syntax.

## 1. Establish one numeric semantics

**Severity: soundness hole.**
*Problem source: a06, “Eval semantics” versus “TS emit.”*  

**Proposal:** make `int` mathematically unbounded and map it to TypeScript `bigint`; represent `dec` exactly using integer coefficient and scale. Native BigInt supplies the integer representation without a third-party dependency. Exact decimal support may require emitted support code; “dependency-free” must not mean “semantically incorrect.” ([TypeScript][1])

No new source spelling is needed:

```can
Ok(
  count = count + 1
  amount = amount + d"0.1"
)
```

**Breaks:** numeric TypeScript interfaces, host adapters, numeric goldens, and tests expecting `int64` overflow. The supplied small-value can examples need no conceptual rewrite.

## 2. Make recursion policy global, and simplify its local proof

**Severity: soundness hole.**
*Problem source: a07 “Termination”; a08 “Halting argument.”*  

**Proposal:** inspect all can call edges for cycles, including pinned imports. Initially allow only direct self-recursion. Simplify admission to a guarded unit decrement:

```can
decreases fuel
...
match fuel <= 0
  true => retry.exhausted()
  false =>
    ...
    match call retry__fetch(fuel - 1)
```

Require each recursive call to occur under the positive branch. Negative inputs then reach an ordinary declared base outcome; remove negative-entry failure as the foundation of the theorem.

**Breaks:** presently admitted unguarded recursion and larger-step forms. The shipped retry already has this structure. Negative-entry diagnostic expectations would change.

## 3. Make contracts producer-owned and observations complete

**Severity: inconsistency plus expressiveness gap.**
*Problem source: R10’s “raised or stubbed” rule; reviewer guide §2’s kind-only expectations.*  

**Proposal:** delete stale-emits rejection; retain explicit upper bounds and exhaustive handling. Require complete error expectations and argument-bound script steps:

```can
tests
  missing(id = "u_99") =>
    auth.login_failed(user_id = "u_99")

given
  missing => [
    exchange
      args (id = "u_99")
      outcome db.user_not_found(id = "u_99")
  ]
```

A script should prove “this request received this permitted response,” not merely supply the next response.

**Breaks:** bare error expectations and outcome-only script rows. `stale-emits.can` must become a positive upper-bound example, after declaring its error properly.

## 4. Give effects stable identities, including extern authority

**Severity: inconsistency and expressiveness gap.**
*Problem source: a09 makes cells private but requires foreign-effect supersets; extern bodies remain outside the proof.*  

**Proposal:** export named, revisioned capabilities rather than private storage names. Declare extern authority explicitly and propagate it through callers.

```can
capability Net__Fetch rev 1

extern net__fetch() -> Retry__Doc rev 2
  effects [Net__Fetch]
  emits [net.down]
```

For state access, a public capability should map to private cells inside its defining module. Importers name that capability, not the hidden cell.

**Breaks:** effectful extern signatures and their callers’ effect declarations. It also requires an explicit statement of which host capability contracts are trusted rather than proved.

## 5. Remove helper restrictions that contribute no proof

**Severity: expressiveness gap.**
*Problem source: a07 “TS emit,” especially module-wide Ok-shape agreement and exclusion of record parameters.* 

**Proposal:** allow explicitly typed record parameters and give every function its own result shape.

```can
fn auth__verify(
  user: Db__User
  pw: Auth__Password
) -> Auth__Session rev 2
```

The signature already supplies the information needed to type the call and its `Ok` binding. Flattening `user` into several parameters adds opportunities for incorrect argument wiring without adding a guarantee.

**Breaks:** no existing well-typed source needs to become invalid. Refactored functions need revision changes; TypeScript result declarations and goldens may change.

## 6. Consolidate evidence and binding rules

**Severity: sharp edge and inconsistency.**
*Problem source: a07’s caller-test flow-through, warning-only missing tests, and the put-specific empty-payload rule.*   

**Proposal:** missing tests and dead script keys are errors; script keys identify their root function and test; zero-field outcomes bind only `_`.

```can
given
  auth__login.happy => [Ok()]
  auth__verify.vh_ok => [Ok()]

on Ok _ => Ok(user_id = user_id, remaining_tries = 3)
```

The last rule removes the need for a store-specific “bound but unusable value” rule.

**Breaks:** existing unqualified script keys and empty-payload names such as the verifier’s `ok`. Most changes are mechanical.

## 7. Define revision identity over executable dependencies

**Severity: inconsistency.**
*Problem source: R4; a07’s unpinned local calls; a09’s unversioned state initialization.*   

**Proposal:** specify that a pinned function identifies its executable dependency closure, including local helpers, captured state initialization, and imported revisions—not merely its own textual body.

```can
fn count__bump(by: int) -> Count__Tally rev 2
```

A relevant initialization or helper change can therefore require this bump even when the function’s own expression is unchanged. Amend “bump without changing code” to account for changed executable dependencies.

**Breaks:** lock identity and revision-bump rules. Whether the present implementation already hashes such closures is **not established by the uploaded files**.

---

# Task 3 — New features: useful to agents, unpleasant to humans

## 1. Inductive state contracts plus explicit history scenarios

**Motivation:** a09 proves fresh-store scenarios but excludes production histories. *Source: a09, “The isolation argument” / “Open decisions.”*  

```can
state Count__total: int = 0
  invariant nonnegative
    Count__total >= 0

scenario count__history
  execution serial
  initial [Count__total = 0]
  steps
    count__bump(by = 3) => Ok(total = 3)
    count__bump(by = 3) => Ok(total = 6)
  final [Count__total = 6]
```

**Proof obligation:** initialization establishes the invariant; every write preserves it under explicit function preconditions. An agent supplies checkable certificates for those obligations. Separately, scenarios check complete ordered histories and final stores.

The two guarantees must stay distinct: **finite history tests are not the induction proof**.

**Why humans hate it:** preconditions, preservation certificates, explicit initial state, and final-state expectations accompany trivial mutations.

**Why agents benefit:** an exact account of legal state transitions and reproducible multi-call failures.

**Complications:** the current counter accepts arbitrary signed increments, so it would need a precondition or declared rejection. State contracts become revision-relevant. Foreign accessors need contractual state summaries; async must preserve the same invariants.

## 2. Int-backed brands with checked inhabitants

**Motivation:** int-backed brands are explicitly deferred, while current brands are string-backed and literal-sealed. *Sources: reviewer guide §§3 and 11.*  

```can
brand Retry__Fuel is int rev 1
  invariant nonnegative
    value >= 0

match call brand__check(Retry__Fuel, raw)
  on brand.invalid _ =>
    retry.invalid_fuel(value = raw)
  on Ok validated =>
    Ok(fuel = validated.value)
```

**Proof obligation:** literal construction must satisfy the invariant statically; checked construction may return the brand only after validating it. Branded arithmetic never silently preserves the brand: its result needs a new proof or checked construction.

**Why humans hate it:** incrementing a counter can require reconstructing and revalidating a branded value.

**Why agents benefit:** units, fuel, attempt counts, and identifiers become mechanically non-interchangeable; numeric constraints travel with the value.

**Complications:** this depends on exact integer semantics. `decreases` needs an explicit permitted numeric measure. Also, distinguish **numeric refinement** from **secret opacity**: adding numeric projection must not silently introduce an escape hatch for existing secret brands.

## 3. Compositional resource budgets

**Motivation:** a08 distinguishes termination from its depth cap and explicitly does not budget production stack usage. *Source: a08, “Halting argument” / “TS emit.”*  

```can
fn retry__fetch(fuel: int) -> Retry__Doc rev 2
  requires [fuel >= 0]
  decreases fuel
  budget
    net__fetch.calls <= fuel
    recursive.frames <= fuel + 1
  proof budget
    base
      0 <= fuel
    step
      1 + (fuel - 1) <= fuel
```

**Proof obligation:** every path’s resource use fits its declared budget; sequential calls add costs; alternatives take the maximum; recursive summaries satisfy supplied induction certificates.

**Why humans hate it:** refactoring an extra lookup into a helper changes accounting obligations throughout the call chain.

**Why agents benefit:** mechanically checked limits for retries, writes, allocation units, or API invocations.

**Complications:** count **language-visible invocations**, not unverifiable claims about work inside an extern. Golden evidence gains cost traces. A proven finite frame bound still needs comparison with an explicitly selected deployment resource limit.

## 4. Structured async with exhaustive joins and authority partitioning

**Motivation:** async is deferred and the current store has single-threaded semantics. *Sources: a05 “Explicitly later”; a09 “The isolation argument.”*  

Illustrative join fragment, assuming each branch has one declared failure kind:

```can
match join Pair__Read
  order [left, right]
  left: Left__Value
    effects [Left__Store.read]
    call left__read()
  right: Right__Value
    effects [Right__Store.read]
    call right__read()
  on left.Ok l right.Ok r =>
    Ok(left = l.value, right = r.value)
  on left.down _ right.Ok _ =>
    pair.left_failed()
  on left.Ok _ right.down _ =>
    pair.right_failed()
  on left.down _ right.down _ =>
    pair.both_failed()
```

**Proof obligations:** every spawned task is joined exactly once; the outcome product is exhaustive; branch authority is explicitly partitioned. For concurrent state access, require each branch’s writes to be disjoint from every other branch’s reads and writes, unless a separately specified protocol proves safety.

**Why humans hate it:** two three-outcome operations require nine explicit combination cases unless the language gains a separately justified proof abstraction.

**Why agents benefit:** no implicit “first error wins,” forgotten task, detached work, or scheduler-dependent failure selection.

**Complications:** new trace semantics, join goldens, task result types, and state interference rules. Termination of an external service is **not** proved by a join type. Initially admit terminating can tasks; host async requires an explicit trusted completion or timeout contract. Never smuggle that assumption into “proof-carrying.”

---

# Task 4 — Open questions

There is a source mismatch worth correcting: **the uploaded `REQUIREMENTS.md` ends with eight questions different from the six named in the request.** I address both sets. *Source: Requirements, “Open questions.”* 

## The six requested questions

| Question                     | Recommendation                                                                                                                                                                                                                                                                                 | Required proof obligation                                                                                                                                                                                                                 |
| ---------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Division semantics**       | Start with explicit Euclidean integer `divmod`. For decimals, offer exact division returning either an exact result, zero-divisor error, or non-terminating-decimal error. Rounding need not be mandatory: rejecting an inexact quotient is another honest choice. *a06 “Division deferred.”*  | Integer results satisfy `a = b*q + r` and `0 <= r < abs(b)`. Decimal success requires equality with the exact quotient; failure cases are declared and exhaustive. Rounded division, later, must explicitly name scale and rounding mode. |
| **Integer model**            | Choose unbounded mathematical integers. Remove the evaluator/target split instead of adding more boundary caveats. *a06 “Eval semantics” / “Open decisions.”*                                                                                                                                  | Literal parsing, arithmetic, comparison, state storage, and target execution preserve the same integer. Host conversions are explicit checked operations. Resource exhaustion is separately modelled.                                     |
| **Mutual recursion**         | Keep it banned for now, but enforce the ban across the whole program. Later admit explicitly annotated recursive components, not arbitrary cycles. *a08 “Open decisions.”*                                                                                                                     | Every intra-component edge strictly decreases an explicit well-founded ranking tuple. No inferred ranking function and no exemption for imported calls.                                                                                   |
| **Production state sharing** | Keep cells private; share through accessors and exported capabilities. Add history scenarios and inductive state contracts before exposing raw cell references. *a09 “Open decisions.”*                                                                                                        | Stable capability identity, invariant establishment/preservation, and exact sequential state-transition semantics. Scenario tests cover selected histories; invariants cover all admitted transitions.                                    |
| **Async**                    | Defer unrestricted async. Start with structured joins and explicit authority partitioning as above. *a05 “Explicitly later.”*                                                                                                                                                                  | Join completeness, exhaustive combined outcomes, absence of conflicting effects, and an explicit liveness assumption or bound for every task.                                                                                             |
| **Int-backed brands**        | Add after exact integers. Separate nominal distinctions from optional refinement invariants. *Reviewer guide §11.*                                                                                                                                                                             | Every constructor establishes the invariant; arithmetic and conversions preserve it only with evidence. Ranking projections are explicitly declared.                                                                                      |

## The actual questions at the end of Requirements

The following recommendations correspond to the eight entries in that section. 

| Actual question                           | Recommendation and obligation                                                                                                                                                                                                                                    |
| ----------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **1. Multiple revisions at runtime?**     | Initially permit one revision of each symbol per build. Reject incompatible worlds explicitly. Delete the unconditional “provider bumps never break callers” promise until immutable historical dependency closures are genuinely part of the language contract. |
| **2. Named versus positional arguments?** | Choose named arguments only, in declaration order. Check every name exactly once. This is deliberately unpleasant but removes a real ambiguity and makes parameter reordering harmless to call meaning. Existing positional calls require mechanical migration.  |
| **3. Helper extraction?**                 | Keep the decided local execution rule. No additional pin is needed for same-file calls, provided the containing executable revision captures their dependencies.                                                                                                 |
| **4. Multi-call scripts?**                | Add an explicit ordered interaction trace with site identity, actual argument expectations, and outcomes. Every event is consumed once. Do not infer ordering from independent per-site lists.                                                                   |
| **5. Type migration?**                    | Require explicit total migration functions and per-field treatment. State preservation laws; require explicit declarations of intentional information loss. Never silently initialize new fields.                                                                |
| **6. Production stripping?**              | Specify stripping as removal of evidence only. The obligation is preservation of results, error payloads, call arguments/order, and state transitions for identical external outcomes—not merely byte-stable goldens.                                            |
| **7. Module `emits`?**                    | **Delete it.** Function and extern manifests already contain the declarations. This removes an unresolved duplicate authority without weakening per-function checking. All module headers change mechanically.                                                   |
| **8. Formal grammar?**                    | Freeze productions, binding rules, and one canonical form now. Require unambiguous parsing and structure-preserving normalization. A language that rejects ambiguity should not leave its proof-bearing syntax to competing prose interpretations.               |

## The three findings I would fix before adding any feature

1. **The false numeric preservation claim:** make evaluator and TypeScript arithmetic agree, especially for decimals and ranking integers.
2. **The incomplete termination boundary:** reject cross-module recursive cycles and stop presenting fault-bounded sandbox evaluation as a whole-program return guarantee.
3. **The circular emits-honesty rule:** stop treating consumer-written stubs as evidence of provider behavior; make `emits` a clear, producer-owned contract.

[1]: https://www.typescriptlang.org/docs/handbook/basic-types.html "TypeScript: Handbook - Basic Types"
