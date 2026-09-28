# Checked authored callable forwarding

Status: design prepared while the origin corrective executor remains active.
Implementation waits for that owner and its group to retire and for independent
origin acceptance. The campaign continues through the [master queue](performance-improvement-tasks.md).
Assignments are in the [forwarding checklist](generated-forwarding-tasks.md).

## Source evidence and proposed change

`compiler/internal/emit/callables.go` saves a target once, evaluates captures in
order, and passes `async (...): Promise<Completion<T>> => savedTarget(...)` to
`ownCallable`. Authored targets come from checked `program.Functions`, receive
concrete bindings in `programAssembly.functionBindings`, and are emitted through
`RegionEmitter.Function` as native async functions. The assembly's complete
`functions` map also contains catalogue operations, so membership there alone is
insufficient evidence.

Bun and browser `ownCallable` register captures and freeze the supplied function;
`owner-core.registerCallableCaptures` returns that same function. Receipt equality
uses target and captures. The reviewed consumers do not inspect an AsyncFunction
constructor or expose returned promise identity as Can data.

Keep a finite compiler-private inventory pairing checked authored identities with
their actual emitted bindings. For a callable with no array declaration, remove
only the forwarding arrow's `async` when its identity and resolved target binding
both match this evidence. Retain the explicit `Promise<Completion<T>>` annotation
and return the saved target's native promise directly. Unavailable or inconsistent
evidence keeps the current async adapter. Never classify by identifier prefixes.

Calling a selected native async authored function cannot throw synchronously and
returns a native promise of a completion. Arguments inside this forwarding arrow
are saved captures or parameter/context values; no user expression is evaluated
there. This removes redundant async promise adoption at a proven generated site.
It is a source-supported allocation removal, with no measured workload gain yet.

All generic `invoke` thunks, carrier admission, catches and first-boundary origin
capture remain. Native/catalogue, array, dynamic/foreign and otherwise unclassified
targets keep their existing adapters. The proposal needs no universal proof that
every runtime callback is async: the runtime `Callback` type still admits immediate
carriers and synchronous throws. Native collection algorithms, including serial
`Array.fromAsync` visits, remain in place.

## Semantic obligations

Preserve target snapshot timing before ordered, single capture evaluation;
residual parameter/result/error validation; fresh callable construction and receipt
equality; resource indices; assertion context and isolated fixture accounting;
browser owner/context argument positions; cancellation and leases. Hostile
`then`/getter payloads stay boxed, failures retain fresh occurrences and exact
authored and first synthetic boundary locations, and mapping tokens stay paired
with the same evaluations. No fixed internal microtask-hop count is a contract.
There are zero external users and no old private generated ABI/layout obligation.

## Consultation and acceptance

Three fresh Jev requests rewrite every context, instruction and option description,
with identical structured facts and technical code. Saved evidence under
`.performance/performance-push-20260928/`:

- `callable-promise-independent-source-review.json`
- `generated-forwarding-request-[1-3].json` and corresponding responses
- `generated-forwarding-equivalence.json`
- `generated-forwarding-consultation-review.json`

All three select narrowly proven authored forwarding and exact identity/binding
proof. Recommendation probability varies from 0.69 to 0.98. Agreement is advice;
independent source review and executable contracts establish acceptance.

Require actual Bun/browser/assertion selection and conservative fallback tests,
meaningful capture/failure/fixture/owner behavior, exact mappings and origins,
strict TypeScript, and all 24 fresh actual emitted/generated/native validation
cases over doubled, generic/captured maps, fold and frequency. Use the established
one-build path inventory replaced by the checkout runtime link before execution.
Retire every owned prompt and execution directory. No runtime implementation,
new harness, installation or broad audit is needed by this design.

Sol owns design, tasks, review and independent verification. Muse owns every
implementation/test correction. Optional timing remains sequential, bounded and
explicitly busy-host/non-isolated; global defaults and historical evidence stay
separate. Commit each independently accepted checkpoint promptly. Startup import
and initialization attribution, codec retention and all twelve queue slices remain
active after this packet.
