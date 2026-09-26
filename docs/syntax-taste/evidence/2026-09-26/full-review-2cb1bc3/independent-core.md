# Fresh core language review — Can 2cb1bc35435ab5075d51f9f945adfb0e1f9dfe64

Scope: types, abstractions, errors, functions/capture/iteration, assertions, packages, ordinary authoring. Independently inspected implementation, tests, gallery, invoice and std examples. Did not read earlier review/consultation/recommendation/implementation-plan verdicts. No repository edits made by this reviewer. Probes use a Go overlay whose entire source and outputs are retained at `/tmp/can-review-core-20260926/`.

## Judgment

The core is substantially more credible than a syntax demo. Nominal types, owner records, checked public generics, finite error contracts and typed fixture routing provide useful boundaries for SaaS code. I would preserve these. My remaining core conditions for an everyday SaaS recommendation concern predictable iteration, composable testing of encapsulated values, and the cost of extracting reusable code. These are more consequential than adding classes, nulls, conventional loops, implicit traits, or every familiar syntactic convenience.

The primary correctness observation below is a reproducible runtime limit. The other primary findings are intentional design tradeoffs that currently impose visible authoring cost. Passing the existing suite would not settle those tradeoffs.

## 1. High: valid tail recursion has no stack-safety guarantee

**Reproduction:** a checked `count(remaining,total)` uses `relay call count(remaining - 1, total + 1)` in the nonzero branch and returns total at zero. Calling with 100 passes. Calling with 20,000 fails with `standard native_exception`, underlying `RangeError: Maximum call stack size exceeded` on Bun 1.4.2 macOS arm64. Both compile successfully. Exact source, generated program, custom entry and output are in `recurse-small/`, `recurse-large/`, and `recurse-*-runtime-v3.log` (exit 0/1 respectively).

```can
fn int count
    emits []
    given
        int remaining
        int total
    asserts
        base: 0, 0 => ok 0
        small: 3, 0 => ok 3
    match remaining is 0
        false => relay call count(remaining - 1, total + 1)
        true => ok total
```

The emission awaits an invocation thunk; the runtime calls that thunk synchronously before its first await. Thus the async return shape does not break recursive stack growth. Evidence: `compiler/internal/emit/regions.go:493`, `runtime/completion.ts:90-100`. No tail-loop conversion was found in emission.

**Classification:** demonstrated operational limit, not claimed type unsoundness or violation of a documented termination guarantee. It matters when recursion is the available escape hatch for custom scans and recursive data processing. Tiny attached examples cannot reveal this capacity limit. `emits []` does not promise no standard failures.

**Counterevidence:** array `fold`, `map`, `filter`, `find`, `some`, `every`, `for_each` already eliminate many ordinary recursive scans; see `compiler/internal/check/array.go:119` and `compiler/internal/check/array_test.go:9`. Recursion after a genuinely suspending operation can have different behavior; this experiment does not claim every recursive workload fails at 20,000.

**Acceptance condition:** a supported constant-stack form for a 100,000-step state machine and a deep traversal, with documented limits and memory behavior. Direct tail calls can lower to native loops; an explicit iteration form would also work. Keep this contract independent of incidental promise scheduling. Add production-size test evidence, not just a new keyword.

## 2. High: mandatory attached assertions do not compose cleanly with fallible smart constructors

A useful SaaS boundary is `owner record user_id`, constructible only in package `ids`, with `parse(int)` returning it or `invalid()`. A separate package's `describe(ids::user_id)` cannot directly write its legitimate example as:

```can
    asserts
        sample: call ids::parse(1) => ok 1
```

The checker rejects `domain-fallible call requires explicit completion handling`. The row cannot provide the required handling/setup because assertion arguments are expressions and expected completions are single-line values/errors. Mandatory assertions apply to all functions. The exact two-package project and failure are `owner-test-input/` and `owner-test-input-compile-v2.log`.

Sources: mandatory first row in `compiler/internal/syntax/declarations.go:259-264`; expression-only assertion parser at `:285-349`; checking the synthetic invocation at `compiler/internal/check/assertions.go:87-108`. This is exactly the owner boundary that foreign construction and decoding correctly defend: `compiler/internal/check/owner_record_test.go:111-149`.

**Classification:** composition/ergonomics limitation, not an assertion that such APIs are impossible. A package can expose a suitable total sample factory, a consumer can author an extra helper that handles the impossible-for-this-fixture failure, or the domain may return validation as data. Those workarounds alter the public or helper surface to satisfy the test grammar. For a function extracted from a larger authenticated handler, the enclosing handler already has a legitimate owner value; the extracted helper nevertheless needs a new independent root example and a way to manufacture it. This is friction at the point encapsulation ought to help.

**Counterevidence:** genuine opaque ingress/browser handles have harness-supplied scope elision, so do not generalize this finding to all handles (`compiler/internal/check/browser_elision_test.go:8-66`). Public pure owner factories work and are extensively tested. Scenario links give fixtures nominal identities and cross-package refactor checks (`compiler/internal/check/core_integration_test.go:386-480`). Those are good tools, but they do not supply a general fallible assertion setup region.

**Acceptance condition:** test a private extracted consumer of an owner value constructed through a fallible public validation API, without adding a production sample/unsafe factory or a redundant production helper. Preserve inability to forge owner records. A scoped test setup region, reusable fixture producer with completion handling, or appropriately designed external tests could satisfy this. Also demonstrate a handler-to-helper refactor with existing scenario tests retained and minimal new test plumbing.

## 3. Medium-high: error contracts are strong locally but generic library abstraction cannot preserve an unknown caller error set

Public generics have real declaration-level parametric checking, explicit callable/dictionary inputs, stationary mutual recursion and acyclic nested instantiation. This is a strength; do not describe them as unrestricted duck templates. Private generics intentionally remain concrete template checking. Evidence: `compiler/internal/check/exported_generics.go:15-42`; `exported_generics_test.go:33`, `:133`, `:180`, `:292`; `symbolic_proof_test.go`.

The missing abstraction is narrower: an authored reusable `retry_once`, timed wrapper, or composition helper cannot accept arbitrary caller errors and return exactly those same errors. A direct attempt at the desired contract:

```can
fn item retry<item, failure>
    emits [failure]
    given
        callable item () emits [failure] operation
```

rejects `no eligible declaration for "failure"`. This is intentional: no error-set parameter kind (`exported_generics.go:33-34`). Exact rejected project: `error-generic/`, log `error-generic-compile.log`. This snippet is an illustration of the absent capability, not a proposed final syntax.

**Current viable alternative, checked and executed:** define `completed<item>`, `rejected<failure>` records and an `outcome<item,failure>` variant; accept a callable returning that variant with `emits []`; match the variant and retry `rejected`. `result-data-generic/` contains the full public generic, two callback fixtures, main and generated code. All five attached assertion roots passed (including success and repeated rejection), recorded in `result-data-generic-assert-v2.log`.

```can
fn outcome<item, failure> retry<item, failure>
    emits []
    given
        callable outcome<item, failure> () emits [] operation
    asserts
        pass: callable succeed => ok call succeed()
        fail: callable fail => ok call fail()
    match call operation()
        ok outcome<item, failure> first => match first
            completed<item> => ok first
            rejected<failure> => relay call operation()
```

**Tradeoff:** result-as-data is expressive and can be the correct business-domain model. But using it merely to write general infrastructure means wrapping existing native/SQL/HTTP emitted errors into data, then often unpacking them back to emissions. A finite declared union per library is another option, but couples a reusable helper to consumer domains and can overstate each caller's errors. Built-in collection operations already preserve callback error bounds (`compiler/internal/check/array_test.go:48-64`); ordinary user libraries cannot express the same generality.

**Acceptance condition:** show an authored, separately tested retry/composition helper reused with two unrelated error sets and a pure callable. Adding an error to one caller should update that caller's necessary handlers, without changing the helper or unrelated callers. If deliberately keeping result-as-data as the one generic solution, provide and validate a concise conversion story with accurate error identity/provenance instead. No need to add implicit traits or a broad effect system to solve this bounded need.

## 4. Medium: hard style gates and name-based capture increase refactoring cost without improving runtime contracts

Two fresh valid-behavior probes are compile errors:

* A Boolean match with `true` before `false`: `ordinary Boolean match requires false before true`. Exact initial source is `recurse-small/src/main.can.initial`, failure `recurse-small-compile.log`; implementation `compiler/internal/check/completion_matches.go:444-451`.
* `int subtotal = price * quantity` followed by `ok subtotal`: `unnecessary local subtotal ... replace with ok price * quantity`. `named-final-local/`, `named-final-local-compile.log`; rule `compiler/internal/check/locals.go:84-111`.

These are intentional style restrictions, not compiler correctness bugs. A named intermediate can carry domain meaning or be the natural intermediate state of extracting code. An exhaustive non-overlapping Boolean match is valid in either order. The restrictions make routine editing and teaching carry extra nonsemantic obligations. Formatting or optional lint is a better place for these choices.

Capture has a more semantic authoring cost: `near` parameters resolve by the *callee's parameter spelling* in the caller scope (`compiler/internal/check/callables.go:127-153`). A named callback can be useful and capture is typechecked exactly, but renaming `tenant` to `organization` in a library's near parameter changes every capturing callsite even if the callable type has not changed. Callers can create aliases, so this is not an expressiveness blocker. Given that inline lambdas are absent, small callbacks require a named declaration, signature, assertions and often near-name alignment (`compiler/testdata/current/callables/captures.can:18-46`).

**Acceptance condition:** permit a named final intermediate and either Boolean branch order; formatting can canonicalize when desired. Demonstrate explicit binding of two differently named caller values to the same callback and a safe rename workflow. Measure extraction/refactor work in an actual SaaS handler, rather than optimizing only generated source size. A compact callback/binding form is a candidate, not a prerequisite to adopting an entire conventional lambda language.

## Additional reproducible observation: generic assertion inference rejects a natural variant-leaf expectation

The first version of `result-data-generic` wrote `pass: callable succeed => ok completed(3)` and `fail: callable fail => ok rejected("down")`. Although the callable signatures determine `outcome<int,str>`, this rejects `generic assertion pass ... inference shape mismatch: want variant, got record`. Using `ok call succeed()`/`ok call fail()` as expected values makes the assertions compile and all run green. Both sources are retained (`main.can.initial`, `main.can`) and both logs.

This is a concrete inference limitation worth minimizing separately. I did not trace the full inference algorithm or establish whether the intended contract explicitly disallows this form, so do not call it a proven regression. It is less important than the four issues above. An acceptance test is a generic return-variant assertion with a concrete expected leaf, while the callback determines all generic arguments.

## Strengths I would keep

* Immutable nominal records/variants, recursive data, exhaustive pattern coverage and checked nominal constructors reduce accidental interchange of IDs and payloads. `with` has a simple immutable meaning; the gallery explains it clearly (`examples/gallery/src/19-immutable-record-update.can`).
* Owner records defend both construction and representation; foreign codec decoding is denied. These are usable capability/domain-invariant boundaries, not only nominal labels. Existing positive and negative tests cover generic package passage and foreign forging (`owner_record_test.go`, `core_integration_test.go:186`).
* Public generics now fail at the declaration when they silently depend on representation; callable/dictionary parameters provide honest alternatives. Public signatures cannot leak private types (`compiler/internal/resolve/symbols.go:704-705`). Keep this clarity; public/private generic differences need explicit teaching.
* Explicit finite `emits`, checked propagation, `relay`, `chain`, variant data outcomes and first-class callable contracts make failure flow inspectable. Authored error completion is supported now (`compiler/internal/check/completions.go:349-361`). Do not mistake stale example text for a current prohibition.
* Attached executable examples, typed injected completions, and scenario identity provide valuable local feedback. Reports distinguish execution/fixture evidence. They are examples plus controlled seams, not proof that all input cases or real services work.
* Native array operations already avoid forcing all collection work through hand-authored recursion. Callback errors are not silently discarded.

## Documentation/example trust issue observed in scope

`examples/invoice/src/web/web.can:502-518` says Can failures originate only from catalogue calls, and implements `reject_startup` by requesting an invalid environment variable. Current parser/checker admit direct authored error completions. This example teaches obsolete workarounds despite the capability being present. Remove the poison operation and use a meaningful startup error when updating examples; no new language feature is required for this case.

## Validation and limits

* Ran 67 top-level focused tests plus their subtests across check/types/resolve, zero skips, all pass. Exact command/output: `focused-tests.log`. Covered exported generics and symbolic checking, callable contracts/captures, owner boundaries, patterns, scenarios, mandatory assertion typing, array error propagation, error allocation.
* Fresh probes checked/produced modules directly through current `project.Load`, `check.CheckProgram`, `ProgramModules`/`AssertionModules` using an additive Go `-overlay`; no repository source file was added or edited by this reviewer. Harness: `probe_test.go`, `overlay.json`.
* Stack probes execute current compiler-produced functions and a full copy of current runtime on Bun 1.4.2. A retained custom entry skips diagnostic-index configuration because the lower-level emitter does not write the publication driver's source index. Normal generated entry initially failed at initialization for this missing artifact; those initial logs are retained and are **not** language defects. The final reproducer calls the same initialized generated main and exposes underlying standard failure details. Not a qualified release/CLI build or browser test.
* Result-data retry roots use the same emitted assertions with a custom entry omitting the same diagnostic setup. Five roots run individually, all pass. A first launch without `root=N` was correctly refused; retained initial protocol-error log is not a finding.
* `owner-test-input` is a rejection probe, not a claim that every workaround was exhausted. No mock framework quality, property-testing facility, entire performance profile, or soundness proof is established here.
* Coordinator owns full compiler/runtime suites, broader platform/frontend review and required Jev consultations; none performed by this reviewer.
