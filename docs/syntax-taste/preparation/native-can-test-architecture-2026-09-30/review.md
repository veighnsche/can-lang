# Architecture review and resolutions

Status: independent read-only review reconciled into the decision. No builds,
test execution, agent correctness trials or measurements were performed.

The source reviewer checked execution entries, assertion selection, runtime
identity, output leases, termination, output buffering and cleanup. The trust
reviewer independently examined the drafted reference/candidate relationship,
refresh across unsupported syntax/APIs and honest remaining implementation gates.

Two material corrections from the final review were applied:

1. **Planner ordering:** the controller previously appeared to determine required
   roots after offline verification. Initial bootstrap now explicitly verifies
   all roots. Focused operation requires an already-qualified planner running
   before verification, a complete checked root inventory, an input-bound
   selection receipt, conservative closure qualification and all-root fallback.
   A controller cannot self-exempt its newly edited assertions.
2. **Reference versus suite trust:** compiling with qualified R establishes
   provenance but does not qualify changed expected values. Full qualification
   now separately binds a reviewed suite, coverage manifest and expected-vector
   snapshot with negative controls. Development edits may execute provisionally;
   a passing local result does not silently update trusted coverage.

The reviewer found no need to reverse the selected backend/topology. The draft
correctly treats direct-worker-only termination, buffered output before
truncation, absent comprehensive containment, ignored cleanup errors and missing
abandoned-work recovery as requirements to implement rather than existing guarantees.

Reference refresh can use predecessor-compatible checks and explicit independent
bootstrap evidence for new semantics. It blocks affected qualification when that
bridge is unavailable, without requiring old syntax compatibility or accepting
candidate self-testing as proof. A reviewed digest/acceptance record is sufficient;
no new cryptographic signing mechanism is assumed.

The exact first qualified reference artifact, resource enforcement mechanisms,
live wire protocol and authoring APIs remain unimplemented. Those are named
gates within the chosen architecture, not evidence established by this review.
