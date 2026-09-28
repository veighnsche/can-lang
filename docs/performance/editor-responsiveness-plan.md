# Editor responsiveness: investigation and first improvement

Status: planned; source and saved-result review complete. No profiling, builds,
tests or performance runs were started to prepare this plan.

The first objective is to explain the roughly 498 ms completion response, remove
one demonstrated source of delay, and verify the improvement without weakening
editor correctness. This is a focused editor investigation, not a prerequisite
for another full-system audit. Generated TypeScript remains a separate efficiency
priority; its grade does not establish the largest user-facing cost.

## Evidence and the leading hypothesis

The completed measured run `20260927T235706.371974Z` contains six independent
process trials, seven measured batches per trial, two warmups and a 100-function
flat project. Its measured source revision is
`40eaf6a88d6e0376d3abd81a1504d81c451a64ce`; the source inspection for this plan is
against `206a02f`. These are different revisions: re-establish a current baseline
before claiming a fix caused a measured change.

| Observation | What it establishes |
|---|---|
| Completion 498 ms; hover and definition about 498 ms | These queries have similar end-to-end response costs in this fixture. |
| Formatting 999 ms; rename 996 ms | Changed-text formatting and rename take roughly twice as long. |
| Completion, hover and definition each call `CheckSnapshot` | They reload and check the project before answering. |
| Changed-text formatting and rename each check live and proposed snapshots | Two checks could explain their doubled cost; candidate validation protects correctness. |
| `CheckSnapshot` resolves the graph, then the checker resolves it again | Duplicate resolution exists; its time contribution is not yet known. |
| The server dispatches synchronously and diagnoses edits before reading the next request | An edit can delay a queued completion even if the completion handler itself becomes fast. |

**Leading hypothesis:** rebuilding the checked snapshot dominates query cost.
The one-check/two-check pattern supports this hypothesis; it does not prove the
bottleneck or the recoverable time. No fixed half-second timer was found in the
examined LSP, project, resolver and checker paths.

The existing query clock starts before writing the JSON-RPC request and stops
when its matching response arrives. Before each group of queries, the benchmark
restores the document and waits for diagnostics. Thus the 498 ms completion
measurement excludes that preceding edit's diagnosis. A query-only improvement
must not be presented as the complete typing-to-result improvement.

The diagnostics case mixes cheap syntax failures with expensive valid semantic
checks. In the first saved trial, those alternate around 0.6–0.9 ms and
495–501 ms. Its combined median is unsuitable for judging semantic feedback.

## Execution sequence and gates

| Stage | Work | Required output / exit condition |
|---|---|---|
| 1. Establish the experiment | Freeze source and fixture identities, confirm timing boundaries, prepare one qualified editor executable. | A small experiment manifest and a functioning exact completion oracle. |
| 2. Locate the delay | Trace one unchanged-document completion and one valid-edit-to-completion sequence. | Client/server timings account for the observed wait and identify the dominant stage. |
| 3. Choose the smallest fix | Compare only remedies supported by the trace; consult Jev for a substantive design tradeoff. | A short decision record naming the cause, chosen change, correctness obligations and rejected alternatives. |
| 4. Implement and protect behavior | Make one focused change with deterministic regression tests. | Exact semantic outputs remain correct; invalidation and lifecycle tests pass where relevant. |
| 5. Verify performance | Compare baseline and candidate with the same measurement harness, sequentially on a quiet machine. | A measured effect in milliseconds and ratios, with trial spread and edit-to-result cost reported separately. |
| 6. Review and integrate | Review correctness, reproduce the result, update the report and retire scratch. | A reviewable patch, compact evidence, explicit outcome against the proposed goals, and verified cleanup. |

### 1. Prepare only the editor experiment

1. Inspect current changes and active work before touching shared files. Record
   the baseline source revision, harness revision, executable identity, tool
   versions, fixture content hash and host conditions. Keep fixture semantics and
   timing boundaries identical between baseline and candidate.
2. Start with the existing 100-function fixture and exact completion result
   checks. Use one qualified build per revision, shared only within this
   experiment. Preserve distribution integrity checks.
3. Make setup respect selected workloads if necessary: the current compiler
   family preparation builds all three flat fixtures and both maintained anchors
   even when only the editor is selected. Do not launch that broad preparation
   unexamined. Prepare the single required fixture first; add the remaining
   fixtures only during the later scale/real-project check.
4. Run a bounded correctness exchange before taking measurements. Verify the
   complete authored candidate set, signatures, kinds and package provenance;
   an empty or stale response is not a fast successful completion.
5. Use the runner's normal isolation contract: at least 90% CPU idle for
   30 continuous seconds, no detected competing build/test work, one workload at
   a time. If the machine stays busy, defer measurement rather than weakening the
   gate or stopping the user's apps without authorization.

This plan authorizes no automatic execution by itself. Its preparation used
file inspection and existing records only.

### 2. Trace the request before optimizing

Add opt-in, bounded instrumentation that is disabled for final timing runs. Keep
JSON-RPC stdout untouched. The current benchmark discards stderr, so collect an
explicit trace sink within the experiment's owned temporary directory rather
than assuming stderr will be retained. Record request IDs, document/input
generation, method, duration and stage counters; omit source text and payloads.

Measure these boundaries:

| Boundary | Purpose |
|---|---|
| Client frame write → matching response received | Preserve the existing end-to-end request definition. |
| Server frame decoded → handler → response serialization/flush | Separate server execution from unexplained transport/scheduling delay. |
| `CheckSnapshot`: graph load → resolution → checking → diagnostic conversion | Locate the shared query cost. Report inclusive and exclusive timing explicitly. |
| Completion: snapshot retrieval → cursor/context walk → candidate rendering | Establish whether completion-specific work is expensive. |
| Valid `didChange` → diagnostics publication; immediate following completion | Measure edit processing and queueing that the old warmed-query case excludes. |

Use elapsed durations within each process. Do not subtract unrelated absolute
client/server clock readings. Reconcile server stage durations against the client
round trip and investigate the unexplained remainder. Count snapshot checks and
resolver builds as well as time spent.

Start with one short trace per scenario. Repeat a few bounded exchanges only to
confirm the attribution. If snapshot construction dominates, break down only the
expensive substage: source parsing/asset capture for loading, or resolution,
catalogue/declaration/body work for checking. Add a short CPU profile only when
stage timings leave a specific CPU hotspot unresolved. Profiles locate causes;
they are not the final before/after timing evidence.

Separate these scenarios rather than pooling their samples:

- Repeated completion on an unchanged, already checked document.
- Valid edit followed immediately by completion, including diagnostic queueing.
- Completion after that edit's diagnostics have finished.
- Invalid edit and recovery to a valid document.
- Unsaved sibling edit affecting the queried file.

Split diagnostic measurements into valid semantic edits and invalid syntax
edits. Preserve the original mixed case as historical evidence; changed cases
need new identities/contracts and new baselines, not a comparison that silently
changes the amount of work.

### 3. Choose the implementation from the evidence

| Trace finding | First implementation to evaluate | Required constraint |
|---|---|---|
| Harness/transport accounts for the delay | Correct framing, receipt accounting or timing placement. | Re-measure unchanged production code before claiming an editor optimization. |
| Duplicate resolution is material | Pass an already resolved world into the checker through a clear internal API. | Preserve partial-world availability on check errors and diagnostic attribution. |
| Rebuilding unchanged snapshots dominates | Reuse a bounded checked snapshot for an unchanged project input generation across diagnostics and semantic queries. | Prove invalidation and candidate isolation; do not key only by the requested document version. |
| Loading dominates after genuine edits | Reuse only the demonstrated immutable input/parse work, with complete change detection. | Preserve source, manifest, dependency, lock, asset and fixture validation. |
| Actual checking dominates | Optimize the measured checker hotspot or reuse proven unchanged work. | Preserve all language checks and assertion/warning behavior. |
| Waiting behind diagnostics dominates | Evaluate scheduling or cancellation separately. | Version-correct results and diagnostics must hold under rapid edits; asynchronous execution is not the default first change. |
| Completion-specific work dominates | Improve the measured walk/index/candidate construction. | Preserve identity, scoping, ordering and exact completion contents. |

Choose one intervention first. No blanket cache, asynchronous rewrite, Rust/WASM
migration or skipped semantic checks is selected by this plan.

For a difficult choice, give Jev the measured stage costs, source facts, options
and invalidation constraints in one shared state with multiple focused questions.
Make three fresh consultations with independently rewritten explanatory prose;
review equivalence before sending, save compact requests/responses, investigate
disagreements and retain responsibility for the decision. Do not ask Jev to
research missing facts or treat agreement as proof.

If snapshot reuse is selected, the design must account for unsaved siblings,
close/reopen and disk changes, added/deleted files, dependency/manifests/registry/
lock changes, assets and fixtures, canonical path/ownership changes, and the
open-file identity used for error attribution. Use bounded process-local state
with explicit retirement. A copied candidate overlay for formatting or rename
must never replace or mutate the live snapshot. If complete invalidation cannot
be established for the proposed scope, narrow the reuse or choose another fix.

### 4. Protect editor behavior

Run focused tests sequentially with bounded Go workers. Prefer deterministic
counts and output-equivalence assertions over machine-sensitive latency checks.

Existing coverage to preserve:

- Completion scope, shadowing, capture, declaration order, arity, member/type
  contexts, browser-qualified symbols, warning-bearing files, determinism and
  declined/unparseable positions in `compiler/lsp_g04_test.go`.
- Unsaved sibling changes, rapid edits, close/clear, Unicode positions, malformed
  protocol recovery and inert operation in `compiler/lsp_server_test.go`.
- Hover/definition/reference semantics; formatting's invalid-input refusal,
  warning handling and fixpoint; rename's cross-file identity, capture refusal
  and complete editing workflow in the related LSP test files.
- Snapshot parse/resolve/check failure behavior and overlay substitution in the
  driver/project tests.

Add only tests required by the selected fix. For snapshot reuse these include
same-generation reuse, every admitted invalidation source, failed-snapshot
recovery, close reverting to disk, and isolation of proposed edits. For resolver
reuse, prove output/diagnostic equivalence and the existing distinction between
resolver failure and checker failure. Add race testing only if introducing
concurrent state or scheduling.

Do not remove format/rename candidate validation merely to halve their timers.
Do not treat a reduced candidate set, stale result or fewer checks as a speedup.

### 5. Measure whether the fix helps

First compare the unchanged-document and edit-to-result scenarios at size 100.
Use the same instrumented-capable harness with tracing disabled for both source
revisions. Preserve the quiet gate and workload identity. Run baseline and
candidate trials sequentially in a counterbalanced order, such as A/B/B/A, with
at least six independent process trials per revision. Each ordinary editor trial
retains the standard seven measured batches and two excluded warmups. Record the
order; never run competing variants at the same time.

If the shared runner cannot reuse one preparation within this comparison order,
add a narrow paired-comparison path with the same evidence validation and
cleanup ownership. Do not repeatedly rebuild full distributions simply to
alternate trial order. Incompatible harness identities must be reported rather
than bypassing the existing strict comparison checks.

Report each scenario's median, independent-trial range/MAD, absolute difference,
ratio, correctness result and snapshot/resolution counts. Retain individual
request samples. Do not label batch medians or a handful of observations as a
reliable request p95; collect a separately budgeted request sample only if a tail
claim becomes necessary.

Then verify that the gain survives at size 10 and on a maintained multi-file
project such as invoice-compare. Add size 1000 only after the small cases pass and
resource headroom is confirmed. Exercise hover/definition and format/rename for
shared-path regressions. A full twelve-slice run is not a prerequisite for this
first fix; broaden validation only when the changed checker/project code affects
other slices and the focused evidence warrants it.

Proposed acceptance goals, established before viewing candidate results:

- **Correctness is mandatory:** exact results, current-document versions,
  invalidation, safe candidate checks and cleanup all pass.
- **Warm completion goal:** at most 100 ms median on the existing 100-function
  checked-document fixture. This is a provisional engineering goal, not a
  validated universal LSP standard.
- **Edit-to-result goal:** report it independently and aim for 100 ms. If only
  warm queries meet their goal, label the change as a partial improvement and
  identify the remaining edit/check cost. Never claim the typing delay is solved.
- **Evidence of benefit:** the before/after difference must exceed ordinary trial
  variation. If results overlap materially or vary by run order, mark them
  inconclusive and investigate before claiming success.
- **Neighbor behavior:** no unexplained repeatable regression in valid-edit
  diagnostics, hover, definition, formatting or rename. Report conflicting
  outcomes explicitly rather than hiding them in an average.

The user-visible benefit is milliseconds saved for the specified interaction;
Jev's letter grade is secondary reporting, not the acceptance test.

### 6. Finish with a compact, reviewable result

Deliver the patch with the demonstrated cause, what changed, exact correctness
checks, old/new measurements, remaining limits and the next bottleneck if a goal
was missed. Regenerate the editor report from saved results and keep the existing
full-system run distinguishable from the new editor-only experiment. Do not
replace other slices with measurements from a different source revision while
presenting them as one coherent combined run.

Retain a compact result archive and a short explanation. Register scratch cleanup
as soon as allocating it; clean on success, failure and handled interruption,
and use the existing ownership checks for abandoned work. Reuse builds within
this experiment and remove execution bundles/copies/private caches when it ends.
Start with a 32 MiB total trace cap and retain at most 10 MiB of compressed
comparison evidence; if a larger diagnostic is necessary, specify its expiry and
obtain the required explicit retention authorization. Estimate execution-space
needs before preparation with an initial 2 GiB scratch cap. If the qualified
build cannot fit, reduce preparation scope or report the constraint before
starting; do not silently exceed the cap. Leave
shared Go caches, Docker data and operating-system swap alone.

## Code anchors

- [LSP state and dispatch](../../compiler/lsp.go) — `lspServer`, `serveLSP`,
  `diagnose`, `completion`, `hover`, `definition`, `formatting`, `rename`.
- [Snapshot pipeline](../../compiler/internal/driver/diagnostics.go) —
  `CheckSnapshot`, partial-world and diagnostic behavior.
- [Checker entry](../../compiler/internal/check/program.go) —
  `CheckAssertionProgram` and resolver/catalogue construction.
- [Project loading](../../compiler/internal/project/graph.go),
  [assets](../../compiler/internal/project/assets.go), and
  [overlays](../../compiler/internal/project/overlay.go).
- [Editor benchmark](../../tools/performance/drivers/compiler.py) —
  `LSP`, `editor`, exact completion/rename oracles and preparation scope.
- [Fixture and timing contract](../../tools/performance/fixtures/compiler/README.md).
- [Runner and evidence guide](../../tools/performance/README.md).
