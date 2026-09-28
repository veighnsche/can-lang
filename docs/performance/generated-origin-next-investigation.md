# Generated origin follow-up: async/callable, startup and codec lanes

Status: read-only investigation for Codex's next design. No implementation,
benchmark, harness change or measurement in this run. G13-G17 (lazy static
origin reuse) are implemented and verified separately; this note hands the
next supported step to Codex. The campaign continues after this packet;
nothing here declares it complete.

## Per-element costs after origin reuse

G13 removed per-call origin object plus invocation-array allocation for
mapped non-self-tail functions: use sites are now single-line
`(slot ??= Object.freeze(...))` with module-private hoisted vars, and the
24-case validation plus strict `tsc` pass with 14/14 adapter bindings. The
remaining production per-element chain for doubled, generic/captured map,
fold and frequency is unchanged from the prior note except for origins:

1. `observations`/`sortBy` drive `Array.fromAsync(source.keys(), ...)`
   (`runtime/collections/array.ts:86,305`); `fold`/`forEach` chain
   `Array.prototype.reduce` with `prior.then(...)` (`array.ts:160,191`).
2. Private `callback` (`array.ts:35-67`) calls
   `invoke(() => action(...), origin)` directly when `trace.context` is
   undefined (G7 bypass), else via `callContext` plus receipt lookup.
3. `invoke` (`runtime/completion.ts:72-84`) allocates one thunk closure per
   visit, awaits the action promise, authenticates the carrier
   (`checkedCompletion`), refuses unboxed/forged results before
   assimilation, and records the first standard `boundaryOrigin`.
4. The action is the emitted async callable wrapper
   (`compiler/internal/emit/callables.go:102,104`):
   `async (...): Promise<Completion> => savedTarget(...)`, adopting the
   emitted async leaf promise (one hop per element).
5. Success payloads flow through privately proven carriers with direct
   `.value` extraction (G8); every invoke/ok/failure guard and all public
   validation remain.

Semantic obligations that must survive any future change: completion
branding and `checkedCompletion` before inspection; thenable/getter refusal
without assimilation (naked payloads never cross an async return,
`array.ts:25-34`; `then` non-assimilation is pinned by
`regions_test.go` and `data_test.go`); `boundaryOrigin` first-boundary
recording (`completion.ts:67-71`); sync-throw capture into completions;
fixture barrier accounting with the parent frame active across visits;
callable receipts, capture timing and resource indices
(`runtime/callable.ts:18-46`); owner leases, rollback and browser `$canCtx`
threading; immutable histories and frozen snapshots; exact error identities
and occurrence freshness (pinned by the new
`TestStaticOriginBunSharingAndFreshness`).

Old ABI and spelling compatibility that need not constrain the next design
(zero external users): prior generated origin literal spellings (already
replaced by lazy slots in G13); `let` versus `var` slot spellings; prior
`$canOrigin` assignment shapes; prior adapter binding names or import
layouts between emitter and runtime (both are compiler-private and change
together); prior generated TypeScript line widths and module splits. Any
proposal that changes only these while preserving the semantic list above
carries no compatibility burden.

## Promise/callable: missing proof, not a promotion

No narrowly provable promise or callable removal is established here. Two
candidates were examined and are held for a specific missing proof each:

1. `invoke` thunk closure per visit. `callback` passes
   `() => action(...)` so `invoke` can catch synchronous throws and refuse
   unboxed sync results. Emitted callable wrappers are `async`
   (`callables.go:102,104`), and calling an async function never throws
   synchronously, which suggests a private promise-only fast path that
   awaits the action promise directly while keeping `locatedCompletion`,
   `boundaryOrigin` and unboxed refusal. Missing proof: that every
   production action reaching `$canArrayMap/Filter/Fold/ForEach/Find/Some/
   Every/SortBy` is such a never-throws-sync async function returning only
   carrier promises. The `Callback` type admits sync
   `Completion | Promise<Completion>` returns, and the proof must cover all
   emitted `arrayInvocation` call sites (`compiler/internal/emit/array.go`),
   every runtime-internal array caller, and browser-profile callbacks. One
   sync-throwing or sync-returning caller defeats the fast path.
2. `Array.fromAsync` mapper closure and its native scheduling.
   `observations` maps over `source.keys()` with an async per-index closure
   (`array.ts:79-91`); `search` (some/every/find) instead uses a sequential
   `for` loop (`array.ts:203-219`). Correction: the mapper is not
   concurrent. `Array.fromAsync` awaits each mapped value before advancing,
   so callback visits are already strictly serialized; the existing gated
   map test proves ordered starts with peak one
   (`runtime/test/array.test.ts:30-78`), and the sync/failure-stop order
   tests confirm in-order visits (`array.test.ts:455-562,596-660`).
   Replacing the mapper with a sequential loop would not change callback
   interleaving. A replacement still needs proof of native-algorithm
   equivalence (index iteration, awaiting, error propagation), unchanged
   scheduling through fixture barriers and owner leases, and thenable
   non-assimilation (the indices-are-numbers rationale at
   `array.ts:84-85`) — but an imaginary concurrent traversal is not a
   valid reason to reject it. Missing proof: that three-part equivalence
   for any specific proposed replacement. Without it, the call stays as is.

Next investigation (read-only, no implementation): enumerate every
production caller of the eight array entry points from emitted
`arrayInvocation` shapes plus `rg` over `runtime/` for internal callers;
for each, record whether the action is provably an emitted async wrapper
(callables.go shape with `$canOwnCallable`) or a foreign/sync function.
Either the enumeration proves candidate 1 for a Codex fast-path design with
three fresh consultations, or it names the defeating caller and retires the
candidate with evidence. Do not implement `invokePromise` or de-async the
wrapper before that proof exists; do not replace the `observations` mapper
before its algorithm/scheduling/thenable equivalence is proven.

## Startup: imports versus initializers

`program/state.ts` is generated by `program_state.go:419-468`
(`stateImports`, fixed initializer order) over `programImports` /
`browserProgramImports` (`runtime_core.go:5-41`). Every Bun emission
parses, loads and evaluates the full platform graph (core collections,
context, coordination, bytes, callable, fixtures, policy, completion,
data, primitive, failure, plus unconditional domain factories: domain,
CLI, text, numbers, checks, codec, clock, random, log, HTML, form,
assets, SQL, crypto, utilities, HTTP, router, fetch, actions, server,
files, processes, streams, websockets, cookies, S3, markdown, browser
state, AI, environment, IO, arm descriptions) and runs its factory
initializers (`program_state.go:345` and per-domain builders). Browser
builds omit server/files/processes/streams/websockets/S3/AI/env/IO/arm/
SQL/crypto groups but still import the remaining surface. The gallery
fixtures exercise only arrays, maps, text, bytes and codec. Loading
(module parse/load/evaluate, import graph size) and initialization
(per-factory construction) remain unattributed; historical
generated/minimal launch medians do not separate them.

Proposed small sequential busy-host attribution design (design only; Codex
owns the reviewed plan, a future Muse packet implements only that plan):

- Identities. Loading identity: the exact emitted import graph (module
  paths plus bytes/hashes from the emission inventory) timed as real
  dynamic imports / fresh-process module import and evaluation of the
  prepared startup entries. Initialization identity: per-factory
  initializer invocations in `stateImports` order with factory names.
  Compilation identity (separate stage): `bun build` transpile/bundle
  work, which is not a runtime-loading measurement. Correctness
  identity: every factory constructs, error-plan identities resolve,
  and the 24-case validation passes unchanged.
- Method. Reuse the existing one-build perfemit emission and the
  zero-byte-inventory plus runtime-symlink preparation; no new harness,
  no installs, no retained bundles. Correction: `bun build` time must
  not stand in as a loading proxy; it measures transpile/bundle cost,
  not process module import/evaluation. Time the stages separately on
  the busy host with observed activity noted: (a) real dynamic imports
  / fresh-process loading of the prepared startup entries (loading),
  (b) per-factory timers around each `program/state.ts` initializer
  statement (initialization split), (c) `bun build` of the startup
  entries (compilation, reported apart from loading), (d) the existing
  `validate` bench mode (correctness gate). Report medians/ranges with
  busy-host labels only; no quiet claims.
- Ownership/tests. Codex designs; Muse implements attribution-only
  changes to the owned preparation script plus a focused emission test
  asserting the enumerated import/initializer inventory matches the
  attribution input. No pruning, lazy initialization, or factory
  reordering until the attribution is accepted; any later pruning
  proposal needs per-factory usage evidence and per-domain regression
  gates. Global harness defaults stay unchanged.

## Secondary lane: numeric JSON token retention

`runtime/codec/document.ts:28-57` retains `context.source` spelling for
every primitive via the reviver `tokens` map. The queue review records
that typed codec/JSONL `exactInt` and the raw assertion-provider number
branch need only numeric spellings. This stays a secondary lane behind
the main generated async/startup work: it touches the codec path rather
than the per-element callback path, and any change must preserve exact
integers, raw numeric comparison, root-holder tracking, duplicate
detection, native syntax checks, budgets and all rejection paths. No
promotion or implementation is proposed here.

## Handoff to Codex

- For the array-thunk question, supply the caller enumeration above plus
  `array.ts:35-67`, `completion.ts:72-84`, `callables.go:12-107` and
  `array.go:13-45` as consultation evidence; questions must cover
  sync-throw equivalence, thenable refusal, fixture/owner/lease
  scheduling equivalence for the specific replacement (visits are
  already serialized per `array.test.ts:30-78,455-562,596-660`, so
  interleaving change is not at issue), browser callbacks and receipt
  preservation.
- For startup, supply the `stateImports` inventory, the per-factory
  initializer list and the four-identity attribution sketch; questions
  must cover loading/init/compilation separation (real dynamic-import
  and fresh-process loading timed apart from factory construction and
  apart from `bun build`), factory-contract preservation and busy-host
  labeling.
- For the codec lane, supply `document.ts:28-57` and the queue-review
  disposition; no consultation is requested until the lane is promoted.
- Campaign state: G13-G17 implemented with 24/24 validation and strict
  TS; this note only continues the queue. Independent acceptance of the
  origin packet and the next ordered checklist belong to Codex.
