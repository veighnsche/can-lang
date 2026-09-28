# Generated async callback follow-on design

Status: ready; the first generated packet is independently accepted and retired. This is a continuing generated-execution remedy,
not a synchronous-only callback fast path or a completion of the campaign.
Codex designs, saves tasks and verifies; Muse implements all source/test/corrective
changes from [the follow-on checklist](generated-callback-tasks.md).

## Proven repeated boundaries

`runtime/collections/array.ts` sends each element through private `callback`,
`callContext` and `invoke`. `callContext` is async; with no assertion context it
returns `run(undefined)` without frame work. Here that run is already `invoke`,
which always returns a promise of a branded completion. The surrounding wrapper
adopts the existing promise, and `callableInstance(action)` performs an otherwise
unused receipt lookup on every visit. Actual emitted callbacks are async, so this
cost applies to doubled/generic/captured map, fold and frequency. It is source
repetition, with no claimed current allocation count, microtask count or latency.

Add only an absent-assertion-context branch inside the private array callback:
return the same `invoke` promise directly, retaining owner/context argument positions
and the same origin. Keep the existing assertion-context path, receipts and
`callContext` implementation. Do not change the emitter ABI, generic `callContext`,
callable construction, `invoke`, owners or native algorithms. Returning the invoke
promise still shields data-valued then/getters. Invoke still refuses immediate
unboxed/forged carriers before assimilation, authenticates promised completions,
locates synthetic standard failures and preserves occurrences. Promise adoption
hops can change; equivalence concerns specified observable callbacks, fixture
frames, participant selection and owner behavior, not fixed internal delays.

`observations` stores a result only after `invoke` authentication and an `ok` guard;
all constructors create privately branded frozen data carriers. Map/filter then
use public `value()` and repeat membership checks. Sort's stored decorations are
private `success()` results. Fold starts with `success(initial)` and only forwards
guarded successful callback results to the next accumulator. Encode these private
success invariants in narrow local TypeScript types and extract `.value` directly
using the current native operations. Keep every boundary admission/failure guard
and all public `value`/`checkedCompletion`/`invoke` validation. Search already uses
direct extraction after the same invoke/ok checks. No naked payload crosses an
await and no external caller receives a new unchecked API.

## Private map/set metadata reuse

Map and set `backing` helpers first call their public identity predicate (one
private WeakMap lookup), then read that same private map again to get values.
Reuse one locally obtained metadata record after preserving the exact non-null
object and concrete-identity guard. Keep public isMap/isSet predicates, key checks,
origin/failure construction and backing privacy unchanged. The two original
lookups have no yield or user code between them, so source establishes duplicate
metadata access without a scheduling or ownership design change. This helps the
map get/update path used by generated frequency and adjacent set operations.
Existing unknown/wrong-instantiation/proxy/revoked-container cases and immutable
histories still require tests. No removal of validation, key checks or native
collection copies. This routine local lookup reuse does not require another
architecture decision beyond the difficult callback consultation above.

## Design consultations and risk assessment

Three fresh fully reworded Jev requests/responses are saved as
`.performance/performance-push-20260928/generated-callback-{request,response}-[1-3].json`;
the equivalence record confirms identical facts/code and differing explanatory
prose before dispatch. All chose the private collection branch (probabilities
0.99/1.0/1.0) and trusted internal extraction (0.94/0.99/0.99). No categorical
disagreement occurred. Advice does not prove the private provenance or observable
scheduling contract; independent review must establish both.

The generic dispatcher alternative changes sync-throw/thenable behavior for many
consumers. Emitter-wide conditional lowering expands fixtures/origins/owner review.
Both are unnecessary for this narrower demonstrated path. Public unboxing of
arbitrary values retains its WeakSet guard; the internal extraction change is
limited to values created or authenticated and kind-checked by this same module.

Required checks include delayed asynchronous and immediate callbacks, one active
visit at a time, first-failure stopping and empty behavior; hostile then/getter
payloads and raw/forged refusal; exact domain/standard occurrences and origin
metadata; native fromAsync/filter/reduce/toSorted delegation; assertion lineage/
barriers; explicit browser owner forwarding; nested resource leases/close/drain
and coordination outcomes. Actual emitted doubled/generic/captured map, fold and
frequency/native endpoints must run. No fixed-microtask-delay scheduling tests.
Required runtime lint-fix/format/check and owned bounded sequential execution apply.

## Further queue

Keep generated origin allocation, callable promise wrappers and broad startup
imports under review. Origin literals allocate on marked expressions but hoisting
must preserve definition-file spans, tail-loop step metadata and source mapping.
Callable async wrappers adopt target promises, but dropping async can alter thrown
exceptions and scheduling across arbitrary targets. Neither is promoted here.
Optional future focused timing uses the campaign-only quiet waiver, honest
non-isolated/busy-host activity records, identical controls/sampling, no competing
task-owned checks and immediate scratch retirement. No broad audit or speculative
semantic rewrite. The campaign continues after this packet and final acceptance.

## Reconciliation with Muse G6

Muse independently found the same absent-context promise adoption and proposed
a general callContext dispatcher. Codex retains the narrower private array branch:
its run always returns invoke's authenticated completion promise, while generic
callContext permits synchronous T and emitter native-fixture IIFEs can return a
completion immediately. Avoid widening exception/assimilation review to unrelated
consumers. Existing generic context semantics remain untouched. Muse's source
boundary counts are useful; precise promise allocations/ticks are not established
by those counts alone. Its receipt/hash and native-algorithm rejection reasons are
consistent with this packet. G6 startup findings remain a separate supported lane.
