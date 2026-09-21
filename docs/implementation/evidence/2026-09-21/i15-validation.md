# I15 validation

Connection policy and native transport implement the shared A4 request boundary.

- [Full Go suite](i15-go-tests.txt) passed, including literal connection schema,
  metadata/profile, range, header-conflict and duplicate checks.
- [Runtime suite](i15-runtime-tests.txt): 104 tests, 1,677 expectations passed.
- [Strict TypeScript](i15-typescript-tests.txt) passed for transport, its tests and
  the changed owner module. Empty output indicates success.
- [Staged integration, driver and emitter suites](i15-offline-tests.txt) passed.
  TestBundledTransportLoopback builds a fresh release, runs its absolute private
  Bun outside the project with nonexistent PATH, denies external network and
  permits only localhost sockets. Existing staged offline tests remain denied
  all network access. The sandbox profile uses macOS's required localhost spelling.
- [Native qualification](i15-native-capabilities.json) passed, separately from
  language conformance, with String.prototype.isWellFormed required and tested
  against paired and unpaired surrogates. Runtime/archive identity is unchanged.
- Real loopback servers verify one attempt, manual redirects with no target
  request, no cookie persistence, repeated query order, bearer capture, stalled
  body timeout and no decode entry, synchronous late decode without handler entry,
  delivered decompressed byte limits, and outgoing overflow before auth or I/O.
- Domain-boundary tests construct exact catalogue 1100–1105 occurrences, while an
  unexpected decoder TypeError remains standard. Owner cancellation is transport
  phase cancelled unless monotonic deadline expiry takes precedence.
- Controlled native promises prove that caller timeout precedes native settlement
  while the root still waits; cleanup rejection produces a late diagnostic without
  replacing the selected outcome. Native continuation tests finish a scoped handle
  operation using the existing owner's lease during scope drain.
- A then-named decoded value remains boxed without Promise assimilation. Response
  header tests preserve separate cookies and the native combined ordinary values.

[Three fresh consultations](i15-jev/README.md) document the retention decision;
all request/response evidence is retained. Deterministic tests establish behavior.

The parser and connection declaration wiring are I16. Named body/media-type modes
are I26; provider envelopes remain I17/I28. I15 supplies the common checked policy,
protected-result adapter and real native HTTP boundary without claiming those
later source forms are complete.

Follow-up: [independent review corrections](i15-review-corrections.md) preserve late standard outcomes and document explicit native capture retention.
