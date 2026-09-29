# Synchronous checked map-recovery leaves

Status: authorized continuing production campaign; integer workers accepted
`10610e7e`. Supporting ordered work is in
[generated-map-leaf-tasks.md](generated-map-leaf-tasks.md). The master
[plan](performance-improvement-plan.md) and
[queue](performance-improvement-tasks.md) retain the final exhaustion obligation.

## Evidence and decision

The original integer-worker producer observed 1.65–1.99x faster execution of
four integer map/fold endpoints. Its corrected runtime passed independent
34 runtime tests/344 assertions, 11 emitted-worker tests with zero skips,
required runtime checks and source/custody acceptance. Corrected bytes were not
timed; original raw producers and records remain unchanged.

Frequency remains a dominant supported target: baseline 189.04us versus native
13.54us (13.96x), then 210.25us in the original integer after record, with
overlapping ranges. These busy-host, ordered observations do not isolate causes.
The checked callback in `tools/performance/fixtures/runtime/src/frequency.can`
has map/primitive inputs, no statements and nested terminal completion matches.
Each word enters the fold promise chain, invokes the callback, invokes map get,
then invokes insert or replace and handles domain recovery. Native get/insert/
replace perform synchronous work internally but publish native async methods.
The native control ALSO copies its Map per update. Can must retain immutable
point-update copies and opaque containment registration. Their residual cost and
domain failure creation are not attributed to async alone.

Select a finite checked map-leaf companion and synchronous Completion execution
path, preserving the normal async functions and factory surface. No benchmark
function-name recognition, bespoke counting kernel, general effect analysis,
whole-program async rewrite or mutable-map algorithm change. Map-only factoring
would leave the callback/reduction promise chain; deferral would leave this
source-supported substantial removal unattempted.

Three fresh fully rewritten equivalent Choice consultations are saved under
`.performance/performance-push-20260928/generated-map-leaf-worker-request/response-[1-3].json`.
They select `leaf_worker` with probabilities .96/.92/.80. Reconciliation retains
the .17 map-only and .03 defer alternatives in the weakest response: added
proof/boundary complexity and remaining metadata costs are real uncertainty,
resolved by executable contracts and a current before/after comparison, not
agreement. This is advice only. Request equivalence and source hashes are saved.
The live [API](https://docs.typesafe.ai/api) and
[Choice contract](https://docs.typesafe.ai/primitives/choice) were checked.

## M1: a closed structural proof

Create private `MapLeafProof` separate from authored, collection/core dispatch
and integer worker proof. Tie each entry to exact concrete ProgramFunction
identity/specialization, final resolved binding, complete source and IR region.
Admit exactly two original inputs `(map<K,int>, K)` and the identical map result,
where K is int/bool/str; no Escapes. At callable construction require no captures,
no resource indices and residual arity two. Generic request records remain
specialization provenance, not runtime effects.

The recursive finite grammar admits only statement-free blocks and same-region
success, relay or completion-match terminals. Completion arms may be ok/domain/
standard or forwarding, retaining exact checked bindings/error bounds. Expressions
admit literals, in-scope bindings and int unary minus/+,-,* with exact type/arity.
Calls must have one plain InvocationStep and actual checked canonical get/insert/
replace specialization for that SAME map type: Entry-selected map factory,
operation/contract/error pairs and exact final receiver.method must all agree.
Extend scope only from actual checked inputs, success bindings and arm bindings.

Reject every unlisted node/field, ordinary data matches, explicit domain creation,
body statements, authored/recursive/indirect calls, native expressions, fixtures,
array/asset/SQL/FormAction/JSONFetch/Action calls, coordination, resources, browser
and mismatched/rebound proof. The actual checker prepares even the gallery
get call into two locals. Independent source review accepts only this finite
normalization: every prepared value must satisfy its positional map/key/int
grammar, local and value types must agree, and each step argument must be the
exact corresponding local binding. No arbitrary preparation or reordering
qualifies; shared invocation lowering preserves once-only evaluation order.
This clarifies the original no-effect preparation requirement after ALL-writer
release. Standard arms preserve standard snapshots. The current proof also
conservatively excludes declared outgoing domain errors; recovered gallery
leaves qualify, unhandled domain-forwarding functions use the normal route.

## M2: shared native factory bodies and synchronous boundaries

Inside `runtime/collections/map.ts`, factor ONLY get/insert/replace into private
synchronous Completion bodies reused by their existing frozen own native async
methods. Keep all key/backing checks, native Map copies, insertion order, old
snapshots, opaque ownership children, domain occurrence construction and origins.
The returned seven-method API and its async promise contracts remain intact.
Associate the exact factory-owned method functions with their bodies in a private
WeakMap. A compiler-private `mapMethodWorker` query returns only authenticated
entries, without foreign property/getter inspection; unknown functions refuse.
Do not expose backing storage, callable methods or a Can API/cache.

Add `invokeSync` beside existing invoke in `runtime/completion.ts`. It accepts
only an immediately authenticated Completion, uses the SAME locatedCompletion
standard first-boundary logic and caught constructors, and catches synchronous
throws. Unboxed/forged/Promise/thenable results fail without await, then/getter
inspection or assimilation. Existing invoke and all its admission behavior remain
byte-identical. Do not perform effects again through a slow route after a worker
has run. Standard/domain occurrence identity and first-boundary-once are required.

## M3: companion emission and private callable metadata

Emit deterministic synchronous Completion companions beside unchanged normal
async authored functions, with exact companion imports/exports for cross-module
callbacks. Reuse checked region/block/completion-match/expression lowering under
a separate isolated proof-selected mode; no default asynchronous lowering change.
Every proven map call uses its exact target's private native worker through
invokeSync at the ORIGINAL authored call span. Missing worker identity fails
closed, with that boundary; it does not replay a partially completed callback.

Preserve exact definition-file mapping tokens/spans, scope/evaluation order,
standard snapshot binding, domain arm selection, forwarding and cold catches.
Reuse finite lazy origin slots at first actual call-site use; no fresh origin
object/array on every successful element. Do not mutate an ordinary emitter's
state or reset its accepted origin cache. Runtime/native calls keep their own
origins while the first checked boundary records the exact authored site.

In `runtime/callable.ts`, add a SEPARATE optional eighth private map-leaf
descriptor after the existing seventh integer descriptor. Its fields are
`companion`, `keyKind`, `origin`; companion's own data length must be two and
keyKind exactly int/bool/str. Proxy checks precede own-data descriptors; reject
missing/extra/malformed/accessor metadata without running it. Register only zero
captures/resources and two-input checked proof, retain normal guarded/frozen
callable, receipt and resource handling. Separate private WeakMap query
`mapLeafWorker` must not expand integer or forwarding admission. Integer-only
call sites stay identical. Raw payloads never cross an async boundary unboxed.

## M4: guarded native fold

Use this metadata ONLY in array fold when assertion context AND explicit owner
are strictly undefined. Require a real nonproxy ordinary frozen dense array of
the matching primitive key kind, own data slots, Array.prototype, no symbols or
own constructor/keys/map/reduce. Reuse/generalize the existing private admission
helper narrowly; retain all current integer routes and their contracts.

Use native reduce with invokeSync around each registered companion; unwrap only
authenticated ok carriers synchronously. First non-ok stops traversal immediately
and carries that same failure through the existing boundary. Final result is a
fresh Completion before any Promise-visible return. Empty traversal performs no
callback and preserves the original initial payload/boxing behavior. Initial
opaque map authenticity is checked by the unchanged shared factory body when
called; do not add an empty-fold access. Do not bypass generic outer invoke.

Guard rejection chooses the original adapter BEFORE any callback. Assertion/
fixture scheduling, parent frames, explicit owner/lease/cancellation, unknown/
async/resource callables and browser retain existing conservative routes. The
leaf grammar invokes only immutable native map operations and integer arithmetic,
with no user callbacks, I/O or coordination to observe internal removed yields.

## M5: qualification, measurement and custody

Meaningful new runtime/checked-emitted tests must distinguish the actual fast
route, preserve three primitive key kinds, native immutable/domain operations,
old snapshots and opaque children, nested recovery/forwarding, scope and mappings,
exact first standard authored boundary/fresh occurrences, standard snapshot arms,
hostile boxed payload/carrier no-access, and fail-first stop. Use bounded explicit
fault injection for cold failures, clearly distinguished from valid production
positives. Reuse qualified integer/generic owner/lease/assertion tests rather than
building a duplicate broad suite. A new test stages generated artifacts only and
links the actual runtime; zero-byte inventory must not execute or copy sources.

G68 records ONE baseline of current accepted10610e7e BEFORE production changes,
using existing fixed tools and one bounded reused build, then retires its graph.
After required runtime scripts, relevant Go/Bun tests and ONE post-change actual
graph (13 gates, strictTS/mappings,24 native/emitted oracles,14 fresh saved bindings,
all actual input/tool/runtime/module/dependency identities), G74 permits ONE after
record from that SAME qualified graph. Save full bindings/module inventories
BEFORE cleanup and explain actual added leaf sites/changed modules.

Both records use six fresh sequential unchanged-driver processes, standard
size100/iterations1, two excluded warmups/seven accepted batches, all24 finite
complete controls/oracles/identities,300-second sampling cap,64MiB/500-file scratch
caps and no competing task-owned build/test work. Disclose busy-host bounded
process-name/CPU observations without secrets, fixed ordering/JIT/timer spread;
no idle polls/app/power changes. Compare median of six trial medians/range/MAD,
absolute frequency cost, speedup and Can/native ratio, plus unaffected integer/
other controls. Any phase failure is unavailable disposition/safe cleanup,
NOT resampling, tool repair or partial merge. No p95/heap/isolated-causality claim
or guaranteed improvement threshold. Preserve all prior raw/evidence producers.

Muse owns all production/tests/corrections. Codex owns design, consultations,
actual saved assignments, supervision and proportional independent acceptance.
Use one sole same TUI, native matching get_goal/create_goal before implementation,
evidence progress and ALL-writer release before completed goal/handoff. No extra
agents, installs/checkouts/copies/private caches. Register temporary custody before
allocation; retire groups then exact scratch success/failure/interruption, refusing
active/foreign/replaced ownership. User permits exact owned-viewer maintenance;
shared caches/servers are preserved. Native history64MiB and63MiB join margin
remain; ordinary source work does not require renewed user permission. All twelve
lanes/startup/residual native gaps and final exhaustion remain open after this packet.
