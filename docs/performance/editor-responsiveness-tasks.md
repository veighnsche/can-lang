# Editor responsiveness implementation checklist

Owner: Muse Spark 1.3 Contributor for implementation and checklist progress;
Codex for diagnosis, design, experiment supervision, independent review and
final integration. Status: complete, 2026-09-28; all implementation, independent
review, qualified comparison, storage and integration tasks are checked below.

Follow [the investigation plan](editor-responsiveness-plan.md) and AGENTS.md.
This file is the execution packet: work through each ready task continuously,
record concrete completion evidence, and stop only for a real design blocker.
Do not manufacture additional scope. No routine user approval is needed.

## Evidence and chosen design

The clean source started at `90a8c859`; the parent's independent AGENTS.md
preference commit advanced HEAD to `c693c7b7`. Existing full-system evidence
`.performance/20260927T235706.371974Z/evidence.zip` belongs to `40eaf6a8` and
must stay separate from this editor-only experiment.

Codex prepared one qualified, pinned local distribution and ran short functional
debugging exchanges. The laptop was busy (roughly 78–84% idle); these traces
are attribution evidence, **not accepted latency comparisons**. Accepted
measurements still need 90% idle for 30 continuous seconds. Trace evidence and
the exact full completion oracle currently reside in
`.performance/editor-20260928b/raw/`. That directory has a live standard scratch
owner held by Codex's waiting coordinator; leave its ownership files alone.

In `additional-A.trace.jsonl`, request sequence 5 records:

| Inclusive stage | Duration | Count |
| --- | ---: | ---: |
| decoded request through reply | 519.365 ms | 1 |
| snapshot | 519.137 ms | 1 |
| checking | 517.138 ms | 1 |
| callable metadata | 272.326 ms | 1 |
| body checking | 230.958 ms | 1 |
| catalogue inventory cloning | 509.435 ms | 451 |
| collection classification | 229.013 ms | 201 |
| resolution (both builds) | 2.579 ms | 2 |
| completion context / candidates | 0.023 / 0.027 ms | 1 each |

Cloning nests within callable/collection stages; never sum overlapping stages.
Subtracting clone time from the request leaves about 9.930 ms, including trace
overhead and all other work. The client clock is separate: reconcile round trips
using durations, never unrelated process timestamps. A valid edit immediately
followed by completion queues behind synchronous diagnosis and performs two
snapshots; the historical warm query waits for diagnostics first.

`Catalogue.Inventory` JSON-marshals/unmarshals the entire embedded inventory.
The inventory contains 299 operations. `checkProgramForTarget` repeats its copy
inside the intrinsic callable loop. `collectionOperation` copies it even for
ordinary source identities; 201 such lookups dominate this fixture's body cost.
Every current operation with lowering task `I25` or `A05` has the exact namespace
`can.std.collections@1::`.

**Implement one intervention: eliminate these redundant whole-inventory copies.**
Take one operations copy local to `checkProgramForTarget` and reuse it in its two
loops. Add an early `strings.HasPrefix(identity, "can.std.collections@1::")`
rejection to `collectionOperation`, before obtaining inventory. Retain the old
positive scan and defensive copying. Test the namespace assumption exhaustively
against the pinned inventory. No project state, parse tree, checked snapshot or
mutable catalogue value is retained between checks. All existing overlay reloads,
partial-world behavior, warning/error conversion and format/rename proposed-text
checks continue to execute.

Three fresh Jev requests and replies are saved as `decision-request-*.json` and
`decision-response-*.json`, with `decision-equivalence.json`. All explanatory
fields were independently rewritten; code/numbers/alternatives were kept
equivalent. Replies selected local/index/local, and all selected exhaustive
validation. The disagreement is meaningful: an identity index could reduce
positive catalogue lookup cost and avoid future namespace maintenance, but adds
an API/index and mutation obligations. The current closed catalogue proves the
guard; the invariant test fails if a future catalogue violates it. There is no
measured need to optimize positive collection lookups in this first patch.
Duplicate resolution saves only a few milliseconds. Snapshot caching brings
large invalidation obligations without addressing the demonstrated clone waste.
These judgments are advice; source invariants and tests establish correctness.

## Ownership and operating rules

Muse owns the authored files listed in the lanes below and this checklist.
Codex will not edit those files while Muse works. Do not edit AGENTS.md (the
parent chat owns it), generated catalogue mirrors, pinned vendor files, or
unrelated source. Do not commit, create worktrees, install dependencies, run broad
builds, or start accepted performance trials; Codex owns those integration steps.

Use sequential bounded checks with `GOMAXPROCS=2` and `go test -p=2 -parallel=2`.
Use the installed shared Go cache, never clear it or create a persistent private
copy. Each command needs a reasonable timeout. Any temporary directory must get
cleanup immediately, including interruption/failure; use `t.TempDir` or Python
`TemporaryDirectory` for tests. No process may retain full bundles or dependency
copies outside the already owned experiment. Run no performance benchmark loops.
Do not run tests during Codex's accepted measurement phase. Maximum experiment
scratch is initially 2 GiB, trace 32 MiB total, retained compressed evidence
10 MiB. Do not stop apps or change power settings. No runtime TypeScript changes
are planned; if one becomes necessary, first report the evidence and design
need, then follow the runtime lint/format/check instructions.

The existing draft instrumentation and Python coordinator are uncommitted
Codex-authored starting points. Review and improve them within these tasks; do
not assume that a draft or a tool's success claim establishes correctness.

## Lane 1 — targeted production change

Prerequisite: chosen design above. Owned files:
`compiler/internal/check/program.go`, `collections.go`, and a new focused
checker regression test (or `collections_test.go`).

- [x] P1: Store the operations copy once at the existing contract-gathering loop,
  reuse it for intrinsic receiver metadata; preserve original iteration/order,
  signature construction, method flags and all checks. Avoid global state.
  Done: `catalogueOperations := catalogue.Builtin().Inventory().Operations`
  before the contract loop in `checkProgramForTarget`, reused for the
  receiver-metadata loop. Both loops read only; one defensive copy keeps the
  embedded inventory unreachable. (`program.go`)
- [x] P2: Reject identities outside `can.std.collections@1::` at the beginning
  of `collectionOperation`; retain positive-task filtering exactly.
  Done: `strings.HasPrefix` guard after the trace defer, before `Inventory()`.
  Positive `I25`/`A05` scan unchanged; classification stage row still emitted
  so trace counts show cause removal, not fewer checks. (`collections.go`)
- [x] P3: Add a deterministic test enumerating every embedded operation: derive
  the former `I25`/`A05` predicate, assert all admitted identities satisfy the
  namespace invariant, and compare complete returned metadata with that oracle.
  Also reject ordinary project identities, near-prefix impostors, unknown names,
  and noncollection built-ins. This protects future catalogue changes.
  Done: `TestCollectionOperationNamespaceInvariant` in `collections_test.go`
  (299 ops enumerated, 14 admitted, `reflect.DeepEqual` vs oracle, mutation
  isolation check, 20 rejection identities). Passes.
- [x] P4: Run focused checker/collection/generic/method regressions sequentially.
  Record command, exit status and test counts below. Existing tests plus the new
  invariant must cover method receiver contracts and generic collection calls.
  Done: `GOMAXPROCS=2 go test -p=2 -parallel=2 -count=1 -run
  'TestCollection|TestBulkBuilders|TestCheckedCallable|TestCallable|TestGeneric|TestProgramChecksMethods|TestExportedGenericMethod'
  ./compiler/internal/check/` exit 0, 54 `--- PASS` lines (incl. subtests).
  `gofmt -l` clean on the three touched files.

Completion gate: the production optimization is limited to the two proven copy
sites; returned catalogue values still cannot mutate the embedded inventory.
Record the exact patch rationale and test results.

## Lane 2 — reliable opt-in tracing

Prerequisite: P1–P3. Owned files: `compiler/internal/editortrace/trace.go` and its
tests; `compiler/lsp.go`; the trace-only edits already present in
`compiler/internal/{catalogue/catalogue.go,driver/diagnostics.go,resolve/symbols.go}`;
trace-only portions of the Lane 1 checker files.

- [x] T1: Review the draft tracer for disabled behavior, bounded sink creation,
  monotonic inclusive timings and metadata. Keep source text/payloads/paths out
  of trace rows; stdout must remain valid Content-Length JSON-RPC.
  Done: reviewed `trace.go`. Disabled `Request`/`Stage` are no-ops; sink uses
  `O_EXCL`+0600 with a 32 MiB cap; `time.Now`/`Since` monotonic inclusive;
  rows carry only sequence/method/id/version/stage/duration (key-set asserted
  in tests); sink is file-only so stdout stays valid JSON-RPC. No change needed
  beyond T2/T5 edits.
- [x] T2: Ensure malformed-request `continue` paths finish their request span,
  and the request timing starts immediately after frame decoding. Preserve
  method, request ID and document version attribution without changing dispatch.
  Done: `Request()` now called immediately after body decode; new
  `SetRequestVersion()` attributes the version before dispatch; all 8 post-span
  `continue` paths (didOpen/didChange/definition/hover/references/completion/
  rename/formatting) call `endRequest()` first. The pre-span undecodable-frame
  path stays spanless (no method/id exists to attribute). Dispatch unchanged.
- [x] T3: Make trace creation, write/cap and close failures observable in the
  server exit status; the draft only prints a deferred close error. Preserve
  normal nontraced server behavior and response semantics.
  Done: `runLSP` serves then checks `closeTrace()` explicitly, returning 2 on
  write/cap/close failure (creation failure already returned 2). Nontraced
  path byte-identical behavior; serving never aborts mid-run.
- [x] T4: Add focused deterministic tracing tests: disabled no-output behavior,
  exclusive sink creation, request/stage metadata, inclusive nesting, cap/write
  failure reporting, close retirement. Avoid latency thresholds or broad stress.
  Done: 8 tests in `trace_test.go`, all deterministic (cap forced via
  `r.bytes`, write failure via closed handle, nesting via monotonic ordering,
  no sleeps). `go test ./compiler/internal/editortrace/` exit 0, 8/8 pass.
- [x] T5: Retain counters for inventory clones, collection classifications,
  snapshots and resolver builds so Codex can show cause removal, not fewer
  semantic checks. Document inclusive/exclusive interpretation.
  Done: verified `catalogue-inventory-clone`, `collection-operation`
  (still emitted for guarded rejections), `snapshot`, `resolve-build` stages
  retained after Lane 1; inclusive semantics + never-sum-nested documented in
  the `editortrace` package comment.

Completion gate: tracing is explicitly opt-in, cannot corrupt protocol output,
and failures cannot masquerade as a complete trace. No concurrent scheduling is
introduced. Record diagnostics and tests.

## Lane 3 — harden the focused experiment contract

Prerequisite: Lane 2. Owned files: `tools/performance/editor_experiment.py`, a new
focused `test_editor_experiment.py`, and a narrowly justified change to
`tools/performance/drivers/compiler.py` only if transport verification requires
it. Do not change historical mixed diagnostic workload identity/semantics.

- [x] H1: Resolve public CLI paths and validate sampling/arguments before work.
  Keep the new split syntax/valid-semantic/edit-to-result workloads separate
  from the historical `editor.flat.diagnostics` case. Keep whole completion
  arrays exactly equal to the baseline oracle, with independent flat authored
  signatures/kinds/provenance validation.
  Done: new `validate_args()` resolves all public paths and rejects unknown
  fixtures/size mismatches/missing inputs before work (both modes). Split
  scenario names are disjoint from historical cases; report tags `split` vs
  `neighbor`. Oracle equality + `validate_completion` retained unchanged.
- [x] H2: Retain one qualified distribution per production variant. Capture
  compiler source, harness, executable, manifest, tool and fixture identities;
  include compact patches for new authored files as well as tracked edits.
  Account for the separate diagnostic `Atrace` build already created in owned
  scratch. Verify source stability during each preparation.
  Done: `source_patch()` appends `--no-index` diffs for untracked authored
  files; builds record untracked lists; manifest records harness sha, tool
  archive sha, per-fixture input identities; rows carry executable sha tied to
  the qualified bundle; trace trials reuse A/B (manifest `trace_policy`).
  Per-build source-stability check retained.
- [x] H3: Validate every raw trial and paired comparison before summarizing:
  matching contract, fixture/input identities, complete oracle, exact seven
  measured/two warmup samples, six independent processes per variant at size
  100, trace disabled for accepted trials, successful process exit, full ordered
  ABBAABBAABBA evidence. Enforce compatible harness contexts and expected
  two-per-variant verification at flat-10/invoice-compare; no missing samples may
  silently become a complete comparison. Validate neighboring driver cases
  through the existing schema helpers where appropriate. Keep raw requests.
  Done: `validate_trials()` enforces the exact 20-file ordered set, all
  identities, 7/2 counts, trace exclusion, exit steps, `schema.validate_result`
  on the six neighbor cases plus raw-latency retention. `summarize()` always
  validates first. No `drivers/compiler.py` change needed.
- [x] H4: Keep the unchanged 90%/30-second gate for accepted comparisons and
  process monitoring. Short correctness traces are explicitly functional-only.
  A busy-host timeout must leave a truthful incomplete checkpoint with no
  acceptance claim and verified cleanup. Preparation must be narrow and bounded.
  Done: reviewed; `quiet_window(30, 90, ...)` + monitor kept, trace trials skip
  the gate and are labeled `functional-trace`, timeout path yields incomplete
  manifest with no summary, preparation stays flat-10/100 + invoice-compare
  (manifest `preparation` note). No change needed.
- [x] H5: Review scratch registration, SIGTERM/interrupt/failure paths, standard
  reaping ownership, source-build reuse and retention limits. Measure bounded
  preparation headroom; report cleanup failure instead of hiding it. Never
  delete an active or foreign owner. Preserve useful compact evidence on error.
  Done: reviewed `storage.py`/`isolation.py`: `reap` only takes dead marked
  children (active/flock-held/foreign refused), `interruption_signals` funnels
  SIGTERM into the incomplete+cleanup path, 3 GiB headroom + 2 GiB/32 MiB/
  10 MiB caps enforced, cleanup failures recorded and raised. No change needed.
- [x] H6: Generate an editor-only report from verified saved trials, including
  independent-trial medians/ranges/MAD, ms saved and B/A ratios, warm completion,
  valid edits through completion, syntax/semantic diagnostics and neighbors.
  No p95 claim from seven samples. State the provisional 100 ms goals and do not
  mix new results with the existing full-system report.
  Done: `summary.json` is now `{contract, editor_only, goals_ms, goals_note,
  separation, comparisons}` with per-(fixture, scenario) medians/ranges/MAD/
  saved_ms/ratio and `split`/`neighbor` workload tags; comparisons carry no
  p95 key (asserted).
- [x] H7: Add meaningful lightweight harness regressions for mismatch/missing
  samples, false oracle results, trace-vs-measurement distinction, excluded
  warmups, summary statistics and cleanup on handled failures. Avoid real
  builds/benchmarks in these tests; use mocks or bounded fake transports.
  Done: `test_editor_experiment.py`, 17 tests (fake LSP transport, synthetic
  evidence, 20 rejection subtests). Exit 0. Existing
  `test_compiler_driver`+`test_storage` (26 tests) still pass.

Completion gate: a new focused comparison cannot accept incompatible, incomplete
or trace-only evidence. Codex can resume measurements using qualified bundles
without copying the checkout or rebuilding per trial. Record exact tests.

## Lane 4 — focused correctness handoff

Prerequisite: all implementation lanes. Muse owns this checklist and its result
notes only; Codex owns final experiment execution, review and commits.

- [x] V1: Run `gofmt` on authored Go changes. Run focused Python harness tests
  and the existing compiler driver/storage/isolation contracts sequentially.
  Done: `gofmt -l` clean on all 9 touched Go files. Sequential:
  `test_editor_experiment` 17 OK, `test_compiler_driver` 7 OK, `test_storage`
  19 OK, `test_runner` (isolation orchestration) 17 OK. Zero failures.
- [x] V2: Run relevant LSP completion/scope, lifecycle/rapid edits/Unicode,
  hover/definition/reference, formatting and rename tests, and snapshot
  parse/resolve/check/overlay tests with bounded Go workers. Select tests by
  actual test names; do not invoke tests which build full distributions without
  evidence that this patch requires them.
  Done, all `GOMAXPROCS=2 -p=2` exit 0: `./compiler/ -run
  'TestG0[1-5]|TestServer|TestPublish'` 87 PASS; `./compiler/internal/driver/
  -run 'TestCheckSnapshot|TestSemanticDiagnostic|TestDefinition|TestWarning|
  TestValidateFix'` 9 PASS; `./compiler/internal/project/ -run
  'TestOverlay|TestSourceError'` 4 PASS. No distribution-building tests run.
- [x] V3: Because the optimized checker is shared, run the checker package's
  correctness suite once with bounded workers and a finite deadline. If that
  suite proves unexpectedly broad/heavy, stop expansion and record the exact
  blocker; do not hide a failed check or launch the twelve-slice performance run.
  Done: `GOMAXPROCS=2 go test -p=2 -count=1 -timeout 8m
  ./compiler/internal/check/` exit 0 in 16.6 s (288 test functions). No
  blocker, no expansion, no performance run.
- [x] V4: Inspect the final diff for skipped source checks, altered error/warning
  behavior, candidate overlay contamination, runtime changes and unrelated work.
  Record limitations. No new snapshot cache means no new project invalidation
  state; existing live and candidate snapshots must still be rebuilt.
  Done: diff is production optimization (2 sites) + trace-only stages +
  2 new test files + experiment hardening. All check calls, error returns,
  overlay/format/rename logic intact; no `runtime/`, `drivers/compiler.py`,
  AGENTS.md or unrelated changes. Limitations: (1) inner `check-*` stage rows
  are omitted when that stage fails (request/snapshot rows still complete);
  (2) undecodable-frame path stays spanless (no method/id exists); (3) trace
  tests share one global recorder and stay sequential. No new retained state.
- [x] V5: Update every checklist item with its outcome and concrete evidence.
  Report the patch, exact checks, remaining issues and readiness to Codex. Do not
  claim measured speedup or fulfilled 100 ms goals before Codex's accepted runs.
  Done: all lanes checked with evidence below; no speedup or goal claim made.
  Ready for Codex C1–C5.

## Codex integration lane (Muse must leave these to Codex)

- [x] C1: Independently review implementation and verification evidence; repair
  genuine issues through the same checklist/Muse executor where practical.
  Done: Muse completed production, trace and harness lanes plus R1-R12.
  Codex independently closed a final event-ordering gap (monitor events must
  follow quiet-ready, monitor-final must end monitoring, and quiet sampling
  cannot continue after readiness), with three admission regression subcases.
  Independent checks: 86 harness/driver/storage/runner tests; 87 LSP behavior,
  9 snapshot, 4 overlay, 8 trace and 1 exhaustive collection-invariant tests;
  Go vet, all 9 Go files formatted, and `bun run check:runtime`, all exit 0.
  The full checker suite had passed once under Muse (288 tests); no repeat.
  Pilot retired with verified cleanup and a 64,408-byte archive. Final A/B
  starts from the same final harness/tracer and the two recorded production
  source transitions, with a restoration finally path.
- [x] C2: Trace the candidate briefly; show clone-count reduction with unchanged
  one/two-snapshot and two-resolver-per-snapshot semantics. Reconcile client and
  server durations and interpret nested timings correctly.
  Done: final qualified functional traces in
  `raw/final-attribution.json` in `.performance/editor-20260928-final/evidence.zip` shows warm
  inventory copies 451 -> 10 (snapshot 1, resolver builds 2 in both) and
  queued edit copies 902 -> 20 (snapshots 2, resolver builds 4 in both).
  Client minus sequential server durations is 0.125/0.111 ms warm and
  0.248/0.149 ms edit for A/B. All stages are inclusive; no nested-stage sums
  or cross-process absolute clocks were used. Remaining copies belong to
  resolver, error-registry and catalogue-contract setup. These traces are
  functional diagnosis only; accepted latency conclusions are recorded in C3.
- [x] C3: Complete gated counterbalanced size-100 comparison and size-10 plus
  maintained invoice-compare checks. Reuse existing A and newly qualified B.
  Expand to size 1000 only if small tests pass and resource headroom warrants it.
  Done: coordinator/supervisor session 91836 exited 0; all 27 steps and 20
  trials completed. Flat-100 ABBAABBAABBA (six A/six B), flat-10 ABBA and
  invoice-compare ABBA (two A/two B), each seven measured/two warmup batches.
  Codex independently revalidated all records, reproduced summary/original
  report bytes, recalculated all 30 comparisons, and checked all labeled
  90%/30-second quiet observation sequences plus 406 clean monitor events.
  Warm completion: 497.662 -> 15.607 ms (482.055 ms saved, B/A 0.03136).
  Edit-to-completion: 996.715 -> 31.230 ms (965.485 ms saved, B/A 0.03133).
  Both provisional goals pass; all six candidate trial medians are below
  100 ms. All three ABBA blocks show the large improvement; ranges do not
  overlap. Hover, definition, formatting and rename improve at both flat
  sizes. Invoice warm/edit improve to 196.638/393.101 ms, above 100 ms;
  syntax's 0.056 ms slower median has overlapping two-trial ranges.
  No size-1000 or full-system expansion was necessary. No p95 claim.
  Source restoration digests match; only the two selected production changes
  differ in captured A/B patches. Full independent evidence is in
  `.performance/editor-20260928-final/independent-review.json`.
- [x] C4: Save compact verified editor evidence and report, label any unmet goal
  or incomplete measurement honestly, verify all task scratch reclaimed.
  Done: `editor-responsiveness-result.md` was regenerated from verified raw
  evidence, with all 30 median/range/MAD/saved-ms/ratio comparisons, timing,
  goals, correctness, qualifications, source identities and remaining limits.
  Final immutable evidence.zip is 204,741 bytes with verified CRCs; traces
  total 978,954 bytes; peak observed scratch 214,789,615 bytes. All three
  task-owned runs have cleaned markers and no execution scratch. The
  evidence-only temporary review directory was immediately scoped/removed;
  shared caches and the pinned input archive were preserved. No worktree
  was created or attached. One final presentation correction fixes the
  renderer's Markdown delimiter width to 13 columns, with a regression
  assertion; 40 narrow tests passed. This post-measurement correction changes
  no clock, validation, sampling or compiler behavior; frozen originals stay
  in the archive. No successful broad checks were repeated.
- [x] C5: Commit reviewable changes (excluding unrelated AGENTS.md changes),
  update the investigation plan status and provide the final result with exact
  correctness, latency comparison, limits and evidence location.
  Done: production/tracing commit `aae2bf99` and qualified harness commit
  `bd96324` separate the implementation lanes. This documentation integration
  is committed as `docs(perf): record verified editor responsiveness improvement`.
  The plan and performance index point to the complete maintained result;
  all raw evidence stays compact and ignored. AGENTS.md was unchanged.
  The five-minute continuation is retired after final integration. No new
  broad tests, performance run, install or disposable checkout was needed.

## Completion evidence

Muse: append command/results and task notes here as work completes. Keep useful
evidence compact; do not paste noisy full logs or credentials.

### Muse handoff patch summary (recorded before Codex C1/C5)

Production (`compiler/internal/check/`): `program.go` takes one
`catalogueOperations` copy for both loops; `collections.go` adds the
`can.std.collections@1::` prefix guard before `Inventory()` (stage row still
emitted); `collections_test.go` gains `TestCollectionOperationNamespaceInvariant`
(299 ops enumerated, 14 admitted, DeepEqual oracle, mutation isolation,
20 rejections).

Tracing: `editortrace/trace.go` gains `SetRequestVersion` + inclusive-semantics
docs; `lsp.go` starts request spans immediately after decode, ends all 8
post-span `continue` paths, and returns exit 2 on trace open/write/cap/close
failure; new `editortrace/trace_test.go` (8 tests). Draft stages in
`catalogue.go`, `diagnostics.go`, `symbols.go`, `program.go` retained.

Harness: `editor_experiment.py` gains `validate_args`, `harness_identity`,
`source_patch` (tracked + untracked diffs), `validate_trials` (exact 20-file
ordered set, identities, 7/2 counts, trace exclusion, neighbor schema +
latencies, exit steps), and an editor-only report object with goals and
separation note; new `test_editor_experiment.py` (17 tests, fake transport).

### Checks (all exit 0, sequential, bounded workers)

- `gofmt -l` on all 9 touched Go files: clean.
- `go vet` on `compiler/` + 5 touched internal packages: clean.
- Focused check regressions: 54 PASS (collection/callable/generic/method).
- `editortrace`: 8/8 pass.
- Harness: 17 (new) + 7 driver + 19 storage + 17 runner = 60 pass.
- V2: LSP 87 PASS; driver snapshot 9 PASS; project overlay 4 PASS.
- V3: full `check` suite, 288 functions, 16.6 s, `GOMAXPROCS=2 -timeout 8m`.

### Handoff to Codex

No speedup or 100 ms goal claimed; no performance trials started; no commits,
worktrees, installs, or broad builds. Codex-owned `.performance/` scratch and
coordinator stdin untouched. Ready for C1 review, C2 trace, C3 gated
comparison. Known minor limits are listed under V4.

## Independent review correction lane — required before C2/C3

Codex reviewed the completed patch against the plan. The production change is
supported; these concrete issues supersede the earlier harness readiness claim.
Muse owns the same files as before while carrying out this correction lane.
Read `.performance/editor-20260928b/raw/review-notes.json` for the observations.
Keep the same selected optimization; no new architecture or user approval gate.

- [x] R1: Move every split-workload oracle outside the clock. Split receipt from
  validation: start before writing the frame, receive the matching result,
  capture elapsed, then validate. For edit-to-result receive both versioned
  diagnostics and completion before stopping, then check both outputs. Add a
  deterministic fake-clock test where oracle validation advances the clock;
  verify it cannot affect any recorded response duration. Keep 7/2 sampling and
  complete oracle checks. Update the declared timing boundary precisely.
  Done: `focused()` splits `recv_diagnostics`/`check_diagnostics` and
  `recv_result`/`check_result`; `measure(target,name,timed,validate)` stops
  the clock before validation; edit clocks receive both diagnostics and
  completion before stopping. Timing string now declares receipt-vs-validation
  boundary. New `test_oracle_validation_outside_clock` (fake 100ns ticks,
  50ms validation advancement via `validate_completion` + diagnostics bool)
  asserts all 6 scenarios record exactly 100ns. `test_editor_experiment`
  18/18 pass.
- [x] R2: Correct distribution qualification. `compiler/main.go` returns for
  `version` and `lsp` before `driver.Resolve`, so version is not an integrity
  check. Use a bounded existing production path that resolves and validates
  the pinned manifest/assets (for example `canlc catalogue-check`, outside all
  clocks), and ensure closed asset inventory is verified. Reuse existing
  verification APIs where practical; no broad fixture preparation, builds,
  downloads or new mutable cache. Add a focused mocked admission test proving
  failed qualification cannot proceed to a trial; record the actual checks.
  Done: `build()` now calls `qualify()` which runs `canlc runtime-check`
  (Resolve + bounded tool, timeout 30, outside clocks) and validates JSON
  `kind`/`schemaVersion`. `catalogue-check` also resolves but currently fails
  on stale 290-vs-299 assertion, so `runtime-check` is the correct gate; both
  verify stamped manifest hash, required assets, pinned runtime and every
  manifest file hash. Manifest records `qualification: runtime-check`.
  New `QualifyTests` (6 tests): valid report, non-JSON/wrong-kind rejection,
  resolving-tool proof, failed-exit and mismatched-report admission blocks.
- [x] R3: Validate the actual execution order in manifest steps, not merely an
  unordered filename set. Bind each accepted trial to its observed 90%/30-second
  quiet-ready gate and successful monitoring/exit evidence; reject missing gates,
  reordered/duplicated steps, lower thresholds, or observed competing jobs.
  Preserve the raw isolation observations. Check neighboring case contracts
  (unit, parameters, scope, iteration count) consistently across each fixture's
  A/B trials; reject mismatched size or sampling contracts. Add focused tests.
  Done: `validate_step_order` requires exact ordered trial steps with no dups;
  `validate_isolation_gates` binds Nth trial to Nth quiet-ready (preceding
  sample 90%/30s clean) plus clean monitor-final before next gate, reading
  `isolation.jsonl` without modification; manifest isolation thresholds
  checked; `validate_neighbor_consistency` requires identical unit/params/
  scope/iterations per fixture and size matching. Tests: 12 new rejection
  subtests + preservation check; `test_editor_experiment` 25/25 pass.
- [x] R4: Have the worker itself stamp the actual executable digest before/after
  the exchange; check fixture and harness identity remain stable across it.
  Caller metadata alone cannot prove which executable was used. Nonzero LSP
  process exit must fail the worker, including a trace cap/write/close failure
  after valid responses. A narrow `drivers/compiler.py` transport change is
  authorized here. Add deterministic transport/failure regressions.
  Done: `LSP.close()` now raises on nonzero exit (covers trace 2 after valid
  responses and prior crashes). `worker()` stamps `executable_sha256` + `_before`
  and verifies exe/fixture/harness stability across the exchange; `trial()`
  verifies worker stamps instead of overwriting; `validate_trials` requires
  both stamps. Tests: 4 `WorkerTests` (stamp, exe/fixture change, close
  failure) + 3 LSP transport tests; editor 29/29, driver 10/10 pass.
- [x] R5: Register recorder cleanup immediately after each test's successful
  `Open`, so `t.Fatal` paths cannot leave the global recorder active or handles
  open. Explicit failure assertions must still see the real close/write errors.
  Strengthen collection mutation isolation by mutating a nested input field or
  parameter in the returned operation, and verify a later lookup still equals
  the original metadata. Run only the changed tracing and invariant tests.
  Done: all 8 trace tests (9 Opens) register `t.Cleanup` immediately after
  `Open` (no-op after explicit close, so cap/write assertions still see real
  errors). Invariant test now mutates `Inputs[0].Type/Name` + `Parameters[0]`
  and requires `DeepEqual` vs original. Tracing 8/8 pass, invariant passes.
- [x] R6: Put all setup after Scratch allocation inside its cleanup/exception
  boundary immediately. Failures in manifest hashing/writing or raw/work/tmp
  creation currently occur before the session's try/finally. Add a test injecting
  an early setup failure and proving owned work retires and incomplete evidence
  remains honest. Reuse standard storage ownership; do not touch active/foreign
  work or alter storage.py without a concrete necessity.
  Done: `session()` now enters try immediately after `Scratch()`; manifest/
  raw/work/tmp/environment/observer setup inside boundary with `manifest=None`
  guard; early failure yields incomplete manifest + verified retirement +
  archived evidence. No storage.py change. New `SessionSetupTests` injects tmp
  failure, proves work retired, evidence.zip honest (incomplete, no summary,
  marker cleaned). Editor 30/30 pass.
- [x] R7: Provide a human-readable editor-only report generated from verified
  records (Markdown is sufficient), with the existing medians/ranges/MAD/ratios,
  goals, raw timing boundary, scope and historical-separation notes. JSON alone
  does not finish H6's report deliverable. Add a narrow report-content regression.
  Done: `summarize()` now writes `report.md` via `render_editor_report()` from
  validated rows (contract, goals, separation, timing boundary, correctness,
  scope, full comparison table). Archived via existing `report.md` slot. New
  `test_markdown_report_content` checks headers, goals, timing, scope and
  crafted 100/50/0.5 values. Editor 31/31 pass.
- [x] R8: Run changed harness/driver/storage/runner tests sequentially, plus
  tracing/invariant tests. Record real exit status (avoid pipelines which mask
  failures). Update this lane with exact results and hand back to Codex. Do not
  rerun the full checker suite without a new production change or failure.
  Done, sequential, no pipes, real exits: `test_editor_experiment` 31 OK
  exit 0; `test_compiler_driver` 10 OK exit 0; `test_storage` 19 OK exit 0;
  `test_runner` 17 OK exit 0; `go test ./compiler/internal/editortrace/` OK
  exit 0 (8/8); `go test -run TestCollectionOperationNamespaceInvariant
  ./compiler/internal/check/` OK exit 0. `gofmt -l` clean on 2 touched Go
  files. Full checker suite not rerun (no production change). Ready for
  Codex C1–C5.

Codex integration adjustment: retire the current diagnostic pilot through its
existing owner after your prompt/output files close. Its in-memory coordinator
predates the final harness and cannot be used to claim hardened admission. Then
Codex will prepare fresh final A/B with the same final tracer, tests and harness,
removing only the two selected production optimizations temporarily for A and
restoring candidate bytes in a guaranteed finally path. Each final variant will
be qualified once and reused for all final trials. This avoids source/dependency
copies or extra worktrees and isolates the optimization from trace/harness edits.
The pilot's exact baseline reconstruction, trace and Jev advice remain compact
separate evidence; no pilot timing is an accepted before/after comparison.

### Review addendum while the correction lane is running

Codex has inspected the R2/R3 draft. Finish these obligations before handing
back; these are completion of existing requirements, not a new design lane.

- [x] R9: Complete R2's closed bundle inventory check. `distribution.Build`
  only writes the manifest and launcher and renames the stage; it does not call
  `VerifyBundle`. Production `driver.Resolve` checks every manifest asset but
  does not reject unknown files. The current `qualify` draft therefore still
  needs closed-inventory verification (including symlinks/nonregular entries),
  using a narrow adapter or existing verification API. Record the successful
  check and add unexpected-file and symlink rejection tests. Keep `runtime-check`
  as the bounded production pin verification; no runtime source edits needed.
  Done: new `verify_closed_bundle_inventory()` narrow adapter mirrors
  `VerifyBundle`'s closed check (manifest shape, full `os.scandir` walk with
  no-follow symlink rejection, unknown-file and non-regular rejection,
  launcher stamp contains manifest digest); `qualify()` runs it after the
  `runtime-check` report validates, so any violation aborts preparation
  before any trial. Manifest records
  `qualification: runtime-check+closed-inventory`. No runtime source edits.
  Tests: `make_bundle` helper + 7 `QualifyTests` (closed accept, unexpected
  file, file symlink, dir symlink, fifo non-regular, stamp mismatch,
  non-launcher path); resolving-tool test now uses a real stamped bundle.
- [x] R10: Complete R3's actual trial binding. Numbering unlabeled quiet-ready
  events by position can associate an unrelated monitor-final with a result.
  Emit labeled trial-start/trial-end observations around each trial's own
  quiet gate and monitored execution, then verify their exact expected order,
  one matching quiet-ready and successful monitored exit within each labeled
  interval. Reject mismatched labels, missing/duplicate finals and events from
  another interval. Require the neighbor contract's iteration count to be 1
  and unit to be `ns/op`, not merely equal between A/B (the report converts
  these values from ns to ms). Add focused admission regressions.
  Done: `trial()` logs `trial-start`/`trial-end` with the trial label around
  each quiet gate + monitored execution (trace trials included).
  `validate_isolation_gates` pairs labeled intervals, requires the 20
  measurement intervals in exact expected order with exactly one quiet-ready
  (preceding in-interval sample 90%/30s clean) and exactly one clean
  monitor-final each; rejects mismatched/unclosed/nested markers, stray
  gate/monitor events outside intervals, and quiet-ready inside
  non-measurement intervals. Exit success stays bound via manifest steps.
  `validate_neighbors` now requires unit `ns/op` and iterations 1 absolutely
  (schema alone would admit `ms/op`/2). Tests: labeled evidence helper with
  lock + 2 trace + 20 trial intervals; 10 new isolation rejection subtests
  (start/end missing, end mismatch, order swap, nested start, duplicate
  ready/final, stray ready/final, ready-in-trace) + 2 absolute neighbor
  subtests applied to every flat-100 row so drift checks would pass.
- [x] R11: Finish immediate LSP lifetime protection in `focused`: currently
  `driver.LSP` is allocated before the existing oracle JSON is read, and that
  read precedes the `try/finally`. Read/validate the oracle before allocating
  the child, or place every operation after allocation under its cleanup
  boundary. Add a malformed-oracle regression proving no child is started or
  any started child is closed. Preserve the corrected response clocks.
  Done: oracle read/parse moved before `driver.LSP` allocation; malformed
  JSON raises `RuntimeError('... not valid JSON ...')` and non-list/empty
  oracles raise `RuntimeError('... non-empty completion array')`, all before
  any child exists. Response clocks untouched. Test:
  `test_malformed_oracle_starts_no_child` (3 subtests: bad JSON, `{}`, `[]`)
  asserts `FakeLSP.last is None`.
- [x] R12: Extend early setup cleanup coverage to the distinct `manifest=None`
  failure path: inject `harness_identity` failure while the initial manifest
  is being constructed, before raw/work creation. Require cleaned ownership,
  honest incomplete archived evidence and no summary. The existing test covers
  tmp creation after the manifest exists; keep both bounded regressions.
  Done: new `test_manifest_construction_failure_archives_honest_evidence`
  patches `harness_identity` to raise during manifest-dict construction
  (manifest stays None, raw/work never created); asserts the error
  propagates, `PROCESS_OBSERVER` reset, no work/raw dirs, evidence.zip holds
  an incomplete manifest with the error and verified retirement, no summary,
  marker state cleaned. No session-code change needed; the R6 boundary
  already covers this path. Existing tmp-failure test kept.

Run the newly affected narrow tests with real exit statuses and update these
checkboxes along with R1-R8. Do not start performance trials or alter the
selected production optimization. Codex has made no concurrent source edits.

### Addendum completion (R9-R12, 2026-09-28)

All four required addendum tasks are complete; R1-R8 entries above are
unchanged. Files touched: `tools/performance/editor_experiment.py` (qualify
adapter + trial markers + isolation binding + neighbor absolutes + oracle
ordering) and `tools/performance/test_editor_experiment.py` (helpers + 9 new
tests + 12 new rejection subtests) plus this checklist. No production,
runtime, transport, or Go changes; no trials, builds, installs, commits,
worktrees, Jev consultations, or coordinator contact.

Narrow tests, sequential, real exits, no masking pipes:
`test_editor_experiment` 40 OK exit 0; `test_compiler_driver` 10 OK exit 0;
`test_storage` 19 OK exit 0; `test_runner` 17 OK exit 0. Full checker suite
not rerun (no production change). Ready for Codex C1–C5.
