# I15 review corrections

Independent review reproduced an unexpected asynchronous decoder TypeError after
HTTP timeout. The original native-value helper wrapped every rejection in success
and immediately published it as selected, so the root drained the work but lost
the late standard diagnostic.

The helper now makes selection with its deadline. Timeout publishes no selected
participant; unexpected standard failures keep their occurrence and are reported
once. Recognized transport failures, including abort after expiry, remain expected
outcomes and do not create false standard diagnostics. Decoders use a separate
protected-completion helper so returned standard completions remain visible too.
Timeout still wins after synchronous late validation, and does not change later.

Review also identified a helper boundary hazard for future resource adapters.
Native and cleanup helpers now accept explicit capture lists. A continuation with
its own capture retains a resource after its parent times out; explicit close
waits until that continuation uses the resource and settles. HTTP's current native
Response/stream objects do not require Can-handle captures. The helper contract is
recorded in the transport documentation rather than relying on parent leases.

Validation:

- [Full runtime suite](i15-review-runtime-tests.txt): 109 tests and 1,693 expectations.
- [Fresh staged transport and ownership tests](i15-review-staged-tests.txt): passed;
  transport is restricted to loopback and ownership remains offline.
- [Strict transport TypeScript](i15-review-typescript-tests.txt): passed.
- New regressions cover the real loopback late decoder exception, synchronous late
  exception, returned boxed standard failure, no diagnostic for expected late
  cancellation, and a retained handle across parent timeout and explicit close.
