# Native map and fold for checked integer callbacks

The user prioritizes large generated/native wins. Accepted core dispatch
`e57f3ad2` removes work but has no measured gain. This packet targets per-element
promises and completion allocations in arithmetic map/fold, rather than another
context-wrapper change. Auxiliary measurement tools remain frozen.

## Evidence and outcome

Current unchanged-driver baseline is
`.performance/performance-push-20260928/generated-large-gap-baseline-qualified.json`:
six fresh sequential Bun processes, standard size100, iterations1, two excluded
warmups/seven accepted batches per case. All24 controls passed, complete real
identities agree before/after, raw samples retained, owned graph/groups retired.
An earlier preparation ended before any timing on sandbox process observation;
permission was corrected with separate bounded process-name/CPU snapshots. Its
failed record stays separate. Host was busy; fixed case order, timer/JIT effects
and broad trial ranges limit attribution. No isolated causality, heap or p95 claim.

| Workload | Can median µs | Native median µs | Can/native |
| --- | ---: | ---: | ---: |
| doubled | 58.00 | 12.13 | 4.78 |
| fold-sum | 45.85 | 3.71 | 12.37 |
| generic-map | 48.69 | 10.65 | 4.57 |
| captured-map | 47.48 | 9.52 | 4.99 |
| frequency | 189.04 | 13.54 | 13.96 |

These are medians of six trial medians for a100-element operation. The map native
controls box/freeze the whole result once; fold native performs one native reduce.
`runtime/collections/array.ts` instead awaits invoke/carriers for each element,
uses Array.fromAsync for map and a Promise chain for fold. Checked double/add/
identity<int>/add_offset still return async boxed values. Captured_map stores its
callable in a local. Frequency additionally invokes error recovery/map operations;
its native reference ALSO copies Map per update. This packet cannot assign its
14x gap to copying or optimize that callback by pretending it is pure arithmetic.

## Requirements

| ID | Behavior | Acceptance |
| --- | --- | --- |
| I1 | Remove per-element promises and carriers for positively checked integer expressions | Actual emitted map/fold with real registered raw workers and native map/reduce; four baseline endpoints exercise them |
| I2 | Preserve normal callable/API and conservative routes | Defined assertion context, browser, unknown/rebound/async/effectful callbacks, resource captures and failed data guards use old adapters |
| I3 | Preserve values, evaluation and immutable data | Captures evaluated once/in order, positions/arity exact, negative/large bigint, empty arrays/fold, fresh frozen output, unchanged inputs |
| I4 | Preserve failure and ownership contracts | Same callback source/span/region on raw failure, first boundary/occurrence preserved, early stop; no then/getter assimilation; no owner/lease/cancellation/fixture bypass |
| I5 | Demonstrate actual workload gain honestly | One existing-method after run; raw membership/identities/oracles; absolute costs, speedup and remaining native ratios/range/MAD, with busy-host limitations |

## Selected architecture and interfaces

Three fresh entirely reworded equivalent Jev requests/responses and preflight/
reconciliation are saved as generated-integer-worker-{request,response}-[1-3].json
and -equivalence/-reconciliation.json. All advise paired_worker with probability1;
this is advice. Source hashes accompany the requests. The decisive evidence is
closed checked IR and real contracts, followed by observed gains.

1. Build a SEPARATE private integer-worker proof after final authored binding
   resolution. Each entry ties concrete ProgramFunction identity/specialization,
   exact final target, defining source and full checked region to a companion
   binding. Require int-only inputs/result; no Errors/Escapes/Requests; no body
   statements; one SuccessCompletion for the same region with a nonnil expression.
   Recursively admit only int literals, bound region input references, unary
   minus, and binary +,-,* with exact arity/type/operator checks. Reject everything
   else, including division/remainder/power, fields/index/match/calls, coordination,
   captures of resource-bearing values and arbitrary effect annotations. Empty
   emits alone, prefixes, Promise types or mixed Functions are insufficient.
2. Emit a synchronous companion returning bigint alongside the unchanged async
   authored entry. Reuse ExpressionEmitter lowering and exact authored marks.
   Deterministic companion names/imports across modules must match actual proof;
   browser omits this feature. Put try/catch around the raw expression. On error,
   throw the private failure value produced by caught at the SAME original
   callback-region source/span/invocation; never throw the completion carrier or
   recreate an occurrence. Origin construction belongs in the cold catch, not
   every element. The existing array boundary/invoke carries that failure onward.
3. Extend Bun ownCallable with an OPTIONAL final integer descriptor after its
   existing context argument. Preserve construction of the same normal frozen
   callable, receipt, captures and registerCallableCaptures. For exact compiler
   proof, pass a residual sync worker using already saved captures/positions and
   the resolved companion, arity1/2 and callback origin. Private WeakMap metadata
   stores the worker on that exact returned callable; no property on user data,
   public Can API, source-string evaluation, name guessing or memoized plan.
   Resource indices must be empty and captures bigint. Bad scalar capture values
   decline fast registration and retain normal behavior. Validate descriptor
   structure through own data properties, without getters/proxies; reject malformed
   registration honestly. Freeze maintained metadata. Expose only the minimal
   compiler-private query required by array.ts. Other ownCallable calls stay intact.
4. Inside existing map/fold boundaries, look up exact private metadata and require
   trace.context === undefined AND trace.owner === undefined. Admit source only
   after trap-free isHostProxy rejection: ordinary Array.prototype, real frozen
   dense array with own bigint data slots, no own constructor/keys/map/reduce or
   symbol overrides; inspect descriptors without executing getters. Fold requires
   bigint initial and worker arity2; map arity1. Guard failure chooses existing
   adapter without replaying callback effects or touching then. Do not use an
   O(n) copy/descriptor-map allocation or cache foreign arrays; a bounded direct
   descriptor scan is enough. Guard overhead is a measured uncertainty.
5. Use native Array.prototype.map/reduce, preserving left-to-right once-only
   operands/accumulation. Native map transfers its fresh result to array() for
   freezing; fold returns its scalar. Worker outputs must be bigint by typeof
   before they reach any Promise; invalid outputs fail without reading then.
   Keep all async-visible results behind success/Completion, unchanged generic
   invoke admission/caught/locatedCompletion and original public promise shape.
   No raw payload is returned from an async function or Promise callback.

Primitive captures, closed non-coordinating integer expressions and absent owner/
assertion context justify removing internal per-element yields. Ordinary callable
registration adds no ownership rule (owner-core.ts); no resource is involved in
this proof. Retain actual cancellation/lease/fixture behavior for every fallback,
verify that pure traversal inside an owned root keeps root settling correct, and
stop/report a material contract conflict rather than expanding proof silently.
No fair-scheduling/async-callback purity assumption is inferred for other programs.

Literal-only array-site inlining was rejected because stored captured callables
would stay slow and capture/mapping machinery would duplicate. General whole-
program async rewriting is deferred. This paired feature is a production
architecture change with measurable removed per-element work, not a promise of
any multiplier. Do not expand into frequency/transient maps, other scalar types,
other array operations, callback recursion/effects, browser or measurement tools.

## Ownership, qualification and measurement

Sole current Muse coordinator sequentially owns compiler proof/companion/callable
integration and runtime callable/array implementations plus focused new tests.
No extra executor/agents/copies/worktrees/caches. Codex owns design, consultations,
supervision and independent acceptance. Runtime edits require lint:fix:runtime,
format:runtime,check:runtime and relevant tests. New executable tests must prove
positive raw execution and conservative negatives; reusing a slow path is not a
fast-path test. Exact assembly binding/specialization/import/mapping contracts,
raw failure source/first boundary/fresh occurrence, malformed metadata, mutable/
sparse/accessor/proxy/non-bigint arrays, hostile values, captures, defined contexts,
owner/resource/async fallback and empty/negative/large arithmetic are relevant.
Reuse successful unchanged generic contracts; no broad unrelated repeats.

One bounded actual graph suffices for generated qualification and after sampling:
existing frozen preparation, checkout runtime link, hardlinked existing benchmark,
private fd3, strictTS,24 actual native controls,14 freshly saved bindings and all
fresh source/module/runtime/driver/dependency identities. One reused build, shared
Go cache/GOMAXPROCS2/-p2,64MiB/500 compiled files. Then ONE after run: unchanged
standard size100/iterations1, six independent sequential driver processes, two
excluded warmups/seven batches, complete finite samples/control checks, no competing
owned builds/tests,300-second sampling cap. Save compact raw target2MiB with exact
before/after tool/input identities. No resampling/tool repair on failure: unavailable
comparison disposition with clean custody. Host observation is bounded names/CPU,
never arguments/environment or CPU-idle waits; no app/power controls. Preserve all
before records. Use median of trial medians, range/MAD and exact parameter equality;
report four affected endpoints, unchanged native references and remaining frequency
separately. No allocation-byte/p95/isolated-causality claim. Production acceptance
and performance evidence remain distinct; regressions/no gain require explanation,
not fabricated gains or a tiny follow-on hardening loop.

Register temporary custody before allocation, retire groups before exact-tree
cleanup on success/failure/interruption and report failures. Full bundles/source/
dependency copies or private caches are not retained. Native session storage and
viewer/tmux cleanup stay with the authoritative same-chat monitor,64MiB bound;
report/release before a storage-bound violation. No full reasoning/log export.

After ALL-writer/native-goal handoff Codex independently checks actual diff/new
contracts/comparison arithmetic/identities, commits accepted exact paths and returns
to dominant frequency/startup/remaining twelve lanes. Campaign exhaustion is not
claimed by this packet. No backwards ABI/layout/golden obligation.

Cold fault qualification must stay bounded: a direct emitted raw companion with
deliberately wrong host arguments can exercise its TypeError catch/origin, while
a controlled registered worker fault exercises the actual fast native traversal/
first-stop propagation. Record these as fault injection, separately from positive
fully checked bigint routes. Do not allocate enormous BigInts merely to force an
engine limit or claim malformed inputs pass the fast admission guard.
