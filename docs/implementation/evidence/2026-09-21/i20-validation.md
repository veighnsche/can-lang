# I20 validation

The private ownership core, compiler capture evidence, CLI supervision and
assertion supervision implement Q10/P6 lifecycle enforcement.

- [Full Go suite](i20-go-tests.txt) passed.
- [Runtime suite](i20-runtime-tests.txt): 83 tests and 1,589 expectations passed.
- [Strict TypeScript](i20-typescript-tests.txt) passed for the changed runtime,
  supervisor, callable and ownership test graph (empty output means success).
- [Offline staged integration](i20-offline-tests.txt) passed for integration,
  driver and emitter packages. The ownership test builds a fresh release and
  invokes its absolute private Bun with network denied, nonexistent PATH and an
  outside working directory. CLI, assertions and child process supervision use
  the actual bundled runtime.
- [Native capability qualification](i20-native-capabilities.json) passed with
  AsyncLocalStorage API and interleaved-context behavior required. This versioned
  report is separate from language conformance; runtime/archive identity is fixed.
- Checked/emitted callable tests prove resource capture indices and real lease
  retention through the emitted native callable. Metadata-only ordinary invocation
  does not introduce a self-close lease.
- Lifecycle tests cover preparation rollback before launch, late failure
  observation and occurrence deduplication, selected completion preservation,
  nested scopes, cross-root rejection, forged and wrong-kind handles, callback
  closure guards, callback dispatch without inherited context, reverse cleanup,
  reentrant idempotent close and lease-preserving deadlines.
- External child-process tests observe permanently pending owners, both with and
  without a timed-out leased resource, then terminate them externally. They do
  not claim native cleanup or a runtime exit status occurred.

Review corrections replace strong diagnostic deduplication with WeakSet, install
shared close promises before native reentrancy, preserve capture evidence through
host wrappers, and admit the qualified native import in staged output validation.
An additional native-dispatch regression requires explicit callback scope binding
and retention of active callbacks during drain.

[Context/lifecycle consultations](i20-jev/README.md) include a documented context
strategy disagreement and its investigation. [Capture policy consultations](i20-captures-jev/README.md)
record the ordinary-call deadlock investigation. Advice is not treated as proof.
Provider-specific server/SQL behavior and coordination algebra remain their own
unchecked tasks; these tests establish the shared ownership boundary.
