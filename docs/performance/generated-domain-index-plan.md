# Reuse immutable catalogue indexes in the domain runtime

This is a production runtime optimization. `createDomainRuntimeWithDigest` in
`runtime/domain-core.ts` currently searches 110 frozen catalogue errors for every
declaration, then recreates two 215-entry type-shape Maps for every runtime.
The canonical core serves Bun and browser; browser `verifyErrorPlan` constructs
it before final domain creation. Repeated factories repeat identical metadata
work. The catalogue arrays and every entry are recursively frozen plain data,
verified against the actual installed runtime. No caller receives these Maps.

The previous auxiliary startup-attribution lane is accepted within builtin-only
scope, committed a544630c and frozen. Its work is not a performance gain. Use the
existing tool unchanged for a single baseline and a single post-change comparison;
do not improve, instrument or extend the measuring device in this packet.

Three fresh fully reworded equivalent Jev requests/responses are saved as
`generated-domain-index-request/response-[1-3].json`, with equivalence and
reconciliation records in `.performance/performance-push-20260928`. All select
lazy private indexing; probabilities .94/.97/.65 vary materially. First-use
startup benefit is uncertain. Source proves repeated lookup/allocation work,
not its wall-time contribution. This choice is independently grounded in the
frozen data and per-instance ownership boundaries; consultations are advice.

Implement two first-needed private caches in the canonical core: an error index
and a paired shape-by-name/shape-by-identity index. Construct error metadata only
when the current valid declaration first needs its catalogue lookup; construct
shape metadata at the existing lookup-table point after per-plan snapshots.
Build locals completely before publishing a cache, leave it unset on construction
failure, and keep the paired shape publication atomic. Error indexing preserves
`find` first-match selection; shape indexing preserves existing Map last-entry
selection. Use native Map lookup with private read-only types; no public cache,
API, instrumentation counter, altered generated layout or generated catalogue edit.
The bounded retained metadata is precisely 110 errors and two 215-entry indexes.

Each runtime still creates its own declaration and concrete-shape maps, clones
and freezes its input snapshots, validates in the same order, invokes every
digest, and independently performs payload/opaque/callable admission. Do not
memoize plans or validation, remove hashing, share plan-dependent closures/state,
alter failure identity/origin/provenance/occurrence freshness or change native
lowering. Reentrant digest callbacks see fully built catalogue caches while their
plan/runtime state stays separate. Both browser verification/admission sequencing
and initialization-phase rejection remain unchanged. Zero external users: old
ABI/source/golden compatibility adds no requirement.

Meaningful tests exercise repeated valid and invalid factories after cache warmup,
declaration/catalogue/digest rejection precedence, separate snapshots and fresh
failure occurrences, plus existing Bun/browser domain and entry contracts. Avoid
tests that merely count cache internals. All authored runtime TS requires the
repository lint-fix/format/check scripts and relevant tests. One bounded generated
validation graph reuses installed Bun/shared Go cache, actual checkout runtime
link and current driver; strict TS/mapping and 24 actual emitted/native oracles
cover the shared runtime change. Retire the graph and every group immediately.

The sole Muse coordinator first captures one baseline using the already qualified
startup-attribution tool before editing runtime code, then completes implementation
and correctness gates, then runs one after record. Both are explicit busy-host,
non-isolated observations: six independent sequential trials, two excluded warmups,
seven accepted batches per four profiles, ordinary counterbalance, diagnostic
last, one fresh startup per batch. Preserve complete controls/oracles/identities,
300-second sampling and 64 MiB/500-file bounds, existing producer/version hashes,
and source identities explaining the intended changes. No competing owned checks
while sampling; no idle polls, app/power control or p95 claims. Do not repeat
failed/partial runs or repair the harness: mark measurement unavailable with a
specific reason and continue production correctness/handoff. Timing records are
compact evidence; no retained bundles, sources, dependencies or private caches.

Compare ordinary module import/initialization and parent intervals, plus inclusive
domain statement observations, with median/range/MAD and sampling membership.
Versions are sequential before/after, not counterbalanced, so host drift prevents
an isolated causal claim. Single first-startup observations do not measure the
repeated-factory benefit. Report production work removed, actual observations
and uncertainties separately; a noisy or absent speedup must be stated plainly.
After this coherent checkpoint continue authored invoke/numeric JSON and all
twelve slice dispositions; final exhaustion review remains outstanding.
