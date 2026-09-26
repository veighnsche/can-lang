# Core evidence reconciliation — 2026-09-26

This supplements `/tmp/can-review-20260926-core.md`. It narrows the decision claims in response to the coordinator's three fresh consultations. Consultation agreement is advice; the following conclusions come from retained probes and current implementation. No further implementation or broad test work was performed.

## Result-data first: supported as an engineering direction, not yet demonstrated as a robust retry library

The retained `result-data-generic` experiment establishes these exact facts:

1. A public generic can accept `callable outcome<item,failure> () emits []`, match generic variant leaves, return the first successful value, and call the operation again in the rejection arm. The compiler accepts this without knowing a consumer-specific failure type. Generated TypeScript executes against the current runtime.
2. Calling that helper with a deterministic callback returning `completed(3)` produces the expected success variant.
3. Calling it with a deterministic callback returning `rejected("down")` produces the expected rejection variant.
4. The callback functions have their own literal expected-value assertions, and these pass. The retry roots, callback roots and main root all passed: five roots total.

The retry expectations are `ok call succeed()` and `ok call fail()`, needed in this probe because direct concrete leaf expectations triggered the separate generic inference limitation. These are value factories, not independent retry-behavior oracles. Their literal callback tests constrain their values, so the whole test set is stronger than an entirely unchecked expected-value computation. Nevertheless, **deleting the second operation call and returning the first outcome would still satisfy both retry assertions**. The results therefore do not independently verify the number of calls, failure-then-success, distinct first/second failure payloads, effect ordering, or bounded retry count. The source branch is visible in generated code, but that is implementation inspection, not behavioral verification.

The report's phrase “success and repeated rejection” should be read as the configured deterministic input/output cases. It must not be summarized as “retry semantics proven” or “production-ready reusable retry demonstrated.”

A small next acceptance exercise, if this direction is adopted, is a first-call rejection followed by a distinguishable success, with an independent expected success value and an assertion that no third attempt occurs. Also test distinct repeated-failure payloads to fix which failure is returned. Existing lexical fixtures/scenarios or a controlled fake can supply the sequence; this reconciliation does not prescribe a new test system.

**Decision implication:** result-data first is justified as the smaller viable abstraction path. I would not require error-set polymorphism merely because it is absent. Validate one real reusable retry/composition adapter and the conversion boundary for native emitted failures before deciding that result-data creates unacceptable boilerplate. The earlier review correctly identified a capability difference between built-in collection contracts and authored libraries, but that difference alone is not a reason to add an effect system.

## Owner fixture factories: possible today without public sample APIs

The reproduced rejection is precisely the inline row `sample: call ids::parse(1) => ok 1`, where `parse` emits `invalid`. It does not establish that the consumer cannot test `describe(ids::user_id)` or that the owner package must export an unsafe/sample constructor.

The consumer can define a **private named fixture helper** (`provides []` can remain unchanged). That helper calls the existing public `ids::parse(1)`, explicitly handles its named failure, and returns the legitimate owner value on success. Assertions for `describe` then use `call fixture_id()` as their input. The owner boundary remains intact: the helper never constructs or decodes the record itself. The owner package already has literal tests establishing that `parse(1)` succeeds with the expected representation and `parse(0)` fails.

There are two distinctions to keep explicit:

* `emits []` means that no domain error escapes; it does not establish mathematical totality or exclude standard failure. A fixture helper can refuse an unexpected factory rejection through an ordinary standard-failure path, causing assertion execution to fail. Existing arithmetic/indexing faults can implement that today, albeit with poor intent-revealing syntax. This is analogous to a test setup assertion failing. No public production sample API is needed.
* If “total helper” means guaranteed to return a valid owner even when an arbitrary legitimate constructor rejects every input, no such helper can exist without another successful constructor or a supplied owner value. That is the correct consequence of encapsulation. General test setup syntax would also have to fail the fixture in that case; it would not legitimately manufacture a value.

For the exact pure `parse(1)` fixture in the probe, successful construction is established by the implementation and its tests. A no-domain-error wrapper is a normal practical fixture factory. It is therefore better to frame the outstanding issue as **test authoring cost and clarity**, not inability to test an encapsulated API.

The cost today:

1. Introduce a named helper solely to put completion handling around a test input.
2. Because every function requires attached assertions, give that fixture helper its own row as well.
3. Arrange a well-typed expected owner value. When the consumer has no public total owner factory, calling the same private fixture producer in its expected expression is possible in principle, but provides an identity/value self-comparison rather than an independent assertion of factory semantics. The owner package's literal tests and consumer's public projection tests must carry the independent behavioral evidence.
4. Choose and explain how an unexpectedly rejected fixture aborts the test. Deliberate division/indexing faults are available, but are less readable than an explicit test-setup failure.
5. Keep this private helper source in the consumer's ordinary source package; I did not establish a separate test-only declaration facility in this review.

No additional helper variant was compiled in this reconciliation, so these details are a source-based description of the existing mechanism, not a new claimed passing reproducer. The original direct-expression rejection remains the only executed owner-fixture probe.

**Decision implication:** document and exercise a private named fixture-factory pattern first. Evaluate an extraction/refactoring example using it. Add test setup syntax only if that measured pattern remains cumbersome. It is not justified to treat a general external test system or mandatory-assertion removal as a prerequisite solely from this probe. A clearer test-setup failure mechanism may be a smaller useful change.

## Iteration: the evidence specifies an operational contract, not a unique syntax

The strongest unchanged finding is the checked tail-recursive counter failing at 20,000 steps on Bun 1.4.2, while its 100-step control passes. The runtime invokes recursive thunks synchronously before awaiting them. This is a genuine capacity limit of the current lowering, not a speculative feature comparison.

A purpose-built iteration primitive could solve general state-machine work while preserving immutable user values and lowering to native loops. Tail-call lowering could solve the measured example more narrowly. Existing array operations already solve many common traversals. The experiment by itself does not select among these designs, nor prove that a broad loop construct is necessary.

**Decision implication:** require a documented, tested constant-stack path for ordinary long-running state transitions and traversal; choose the smallest surface that meets concrete workload examples. A new primitive should demonstrate at least one task not comfortably served by current fold/find/for_each, preserve error and early-exit semantics, and execute a large workload. Do not claim general recursion became safe merely because the new primitive or one tail-call case works. The close consultation round should be treated as uncertainty about the implementation choice, while the measured stack limit itself is unambiguous.
