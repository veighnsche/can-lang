# I13 validation

All existing bytes catalogue operations are executable through the current
compiler and native runtime. The implementation extends the established private
WeakMap ownership adapter; it does not introduce old kernel/export-brand paths.

- [Full Go suite](i13-go-tests.txt) passed.
- [Native runtime suite](i13-runtime-tests.txt): 48 tests, 1,346 expectations.
- [Strict TypeScript](i13-typescript-tests.txt): runtime/tests and freshly staged
  generated entry/assertion graph passed.
- [Offline suites](i13-offline-tests.txt): integration, driver and emitter passed.
  TestCurrentBundledBytes uses the absolute staged release, denied networking,
  nonexistent PATH and an outside working directory.
- [Source assertions](i13-assertions.json): all nine passed. The staged entry
  completed with status zero and no output.
- Native tests cover every octet, negative/256/huge-bigint bounds before numeric
  narrowing, scalar extremes, lone surrogates, BOM preservation, overlong,
  truncated, surrogate and out-of-range UTF-8, and invalid continuation bytes.
- Returned arrays are distinct/frozen. Mutation of input arrays, Uint8Arrays,
  Buffer inputs and native output copies cannot change owned data. The Buffer
  regression specifically protects against its aliasing slice implementation.
- Forged tokens, proxies, plain objects, native arrays and null fail with
  resource_state; proxy traps are never evaluated. Static tests reject opaque
  construction/indexing, call syntax for length, incorrect input types and
  missing declared codec errors.
- TestBytesThroughCanCaptureAndTransportFixture compiles real Can capture and
  invocation regions. The same original byte handle reaches a raw native
  Request/Response fixture, its body round-trips, and attempted mutation of
  transported copies leaves the original unchanged. This fixture uses no network
  and does not substitute for later I26 named-fetch policy tests.

Strict generated TypeScript initially exposed an opaque input annotation
mismatch. Bytes consumers now admit unknown at the private runtime boundary and
validate membership before accessing storage; the final staged graph passes.
