# I25 correction — resources retained through opaque maps

Independent review found that map values could hide resources from participant
lease preparation. An early winner could close a map-held resource before a
losing participant used it. A map containing a closed resource also incorrectly
allowed participants to start.

Each maintained immutable map now registers a frozen snapshot of its values in
private WeakMap containment metadata. The owner recursively follows that evidence
using its existing cycle/resource deduplication, scope validation and atomic
pre-launch lease acquisition/rollback. Construction records evidence only and
acquires no leases; a closed resource can still be stored, but preparation rejects
its use. Native map copies, replacement and removal register their own contents,
while older aliases preserve their original evidence. No backing storage or
user-visible properties are exposed. Sets retain scalar keys and need no resource
containment registration.

Regression checks cover a direct map and nested map/array/callable/record captures,
zero construction leases, exactly one deduplicated capture lease, close waiting
for a losing participant, later successful map retrieval/resource use, closed
resource rejection before every participant, partial acquisition rollback,
replacement/removal versus old aliases, and zero proxy traps. The compiler's
existing conservative opaque capture classification is explicitly checked for
collections::map<int,sql::pool>.

The four new tests run against the unmodified a4b1337 runtime fail in three cases
(lease/close timing, closed-resource preparation, old-alias retention); the proxy
control passes. With the correction, all four pass with 29 expectations. Strict
TypeScript checking and the compiler's executable resource-capture check pass.

Full validation uses an isolated a4b1337 snapshot plus this correction, excluding
unfinished I26 transport changes. Three fresh Jev consultations are retained in
[i25-owner-jev](i25-owner-jev/README.md); agreement is advice, not proof.

The isolated full runtime suite passed: 175 tests, 19,283 expectations, zero
failures. The full compiler and offline integration gate passed with pinned Bun
archive and CAN_TSC configured: `go test ./compiler/... ./tests/integration -count=1`.
This includes the staged map/set project and generated strict TypeScript checks.
