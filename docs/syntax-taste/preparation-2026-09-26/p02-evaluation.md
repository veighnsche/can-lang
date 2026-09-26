# P02 success criteria and qualification matrix

## P02.1 Representative workloads and observable success criteria

Each workload names the observable outcome. Solution comparisons (P04) are
scored against these, not against intuition.

| Workload | Exercises | Observable success |
|---|---|---|
| W1 second application | R02/R03/R09 | A second Can browser app reuses the shared keyed table + field/error controls from the invoice grid without forking them; route/capture/wire edits propagate or diagnose on both targets. |
| W2 second integration | R01/R11 | A second vendor/external capability is added without a vendor-specific compiler patch or hand-edited generated code; capability rejection + lifecycle tests stay meaningful. |
| W3 reusable infrastructure | R06/R07 | One retry/trace helper serves two unrelated callback error sets; an error added to one caller does not force edits to unrelated consumers. Owner-accepting handler helper extracted with useful tests + minimal setup plumbing. |
| W4 large iteration | R05 | 100k-step state machine, growing aggregation, bounded worker batch complete with measured stack/memory; diagnostics identify the failing step on injected faults. |
| W5 controlled failures | R04/R12/R15 | Stalled SQL/headers/bodies, client disconnect, overlapping requests, SIGTERM, S3 cancel-while-replacing, never-settling awaits: bounded user-visible behavior, defined escalation, no post-disposal use, honest unknown-write outcomes, one redacted correlated report per failure. |
| W6 production operation | R10/R11/R13/R14 | Real PostgreSQL app (generated identities, precise values, nullable audit times, JSON, concurrent replay, migration recipe); durable worker (2-worker claims, bounded concurrency, backoff, poison, crash recovery); paired Linux deployment with old-browser-open rollout; AI tenant budgets/correlation + model-change evaluation. |

Design feasibility evidence belongs to preparation; production acceptance that
requires the implementation belongs in the eventual task list (specified in P07,
executed later).

## P02.2 Measurement protocols and thresholds

| Criterion | Applies to | Protocol | Threshold / comparison rule |
|---|---|---|---|
| Correctness | all | Existing suites + new acceptance cases (P07.4); targeted reruns only | No regressions; new cases pass in the owning lane's environment. |
| Agent authoring/repair effort | R06/R07/R08/R09 | Registered held-out tasks under [evaluation-protocol](../preparation/evaluation-protocol.md); attempts counted, prompts + source + diagnostics + retries logged | Token-efficiency is secondary and measured per successful task; zero live attempts to date — no claim until trials run. Do not equate fewer characters with fewer tokens. |
| Token costs | R08/R06/R07 | Same trials; model/tokenizer recorded | Compared, not thresholded; brevity never overrides contracts or repair reliability. |
| Stack/memory | R05 | Bun 1.4.2 countdown/state-machine probes; record heap + stack outcome at 100 / 20k / 100k steps | 100k steps completes; memory growth stated (linear-in-state allowed, quadratic-in-history rejected for aggregation). |
| Latency (bounded behavior) | R04/R15 | Stalled-dependency injection: SQL, headers, bodies, S3 awaits; disconnect; overlap; SIGTERM | User-visible bound stated per operation; shutdown escalation path demonstrated; unknown-write outcomes honest. No invented millisecond SLO without measurement. |
| Throughput | R05/R11 | Bounded worker batch; bulk aggregation | Reported with environment; used to compare alternatives, not to promise an SLO. |
| Operational behavior | R10/R11/R12/R13 | Live Postgres, 2-worker runs, crash recovery, paired deploy, old-browser rollout, failure-hook redaction | Each scenario has a named owner + destination in the dependency map (P08/P09.4). Unavailable live evidence is not a pass. |

No measured advantage is claimed from intuition at any point.

## P02.3 Existing evidence inventory and limits

| Evidence | Covers | Limit |
|---|---|---|
| `bun run check:runtime` (review) | lint/format/typecheck at 2cb1bc3 | Pass; rerun only if runtime sources change. |
| `bun test ./runtime/test/` (review) | 705 pass, 39 skip, 1 sandbox Chromium-launch failure; browser-owner file reran 2/0 outside sandbox | Skips: 16 MySQL, 12 S3, 11 browser-guard live. Not a zero-skip live qualification. |
| `go test ./compiler/internal/...` (review) | 12 packages pass, 2,298 pass events, 5 skips | Skips: 3 qualified-archive, 1 live MySQL, 1 negative fixture. |
| Whole-program checks (review) | 23 projects, 444 functions, 1,131 assertion roots; grid browser closure | Checking, not fresh execution of every root or qualified publication. |
| Focused core tests (review) | 67 top-level + subtests | Overlaps compiler suite; not additive. |
| Core probes: recursion | 100-step success vs 20k native overflow (sync tail relay) | Emitted-program observation on Bun 1.4.2; does not establish a universal recursion limit (async-after-suspension differs). |
| Core probes: result-data generic | 5 assertion roots pass | Representation + returned values only; a no-retry implementation passes the same examples. Must use distinguishable outcome sequences + independent oracle next. |
| Core probes: owner/error-generic/named-local | Checker rejection cases | Establishes current rejections; private fixture helper is a valid workaround (no impossibility claimed). |
| Platform probes: DOM semantics | Chromium dirty-input attribute/property divergence; identical checkbox snapshots | Native-semantic probe matching the adapter; not end-to-end emitted-Can. |
| Backend probes: race-drain | Winner selected; root waits for loser | Deterministic runtime experiment; not an HTTP latency benchmark. |
| S3 findings | Source-confirmed contract/deadline discrepancies | No destructive live storage experiment performed. |
| UP26/UP27 shards (audit) | 106 integration + all unit packages at be95d009 (docs-only delta to 2cb1bc3) | Staged qualification; full release matrix, live S3, sustained load, model-quality eval not rerun. |
| Prior preparation packets | Current idioms/gaps per topic (Sept 24) | Candidate comparisons + product qualification belong to this round where reopened. |

Reuse rule: keep source inspection, native probes, emitted-Can execution,
assertion execution, and live qualification distinct. Rerun only on changed
code, missing coverage, or uncertainty.

## P02.4 Required environments and fixtures

| Environment / fixture | Needed for | Availability | Cost / credential notes |
|---|---|---|---|
| Bun 1.4.2 darwin-arm64 (pinned) | All probes | Local | None. |
| Go 1.27.1 + GOCACHE | Compiler checks | Local | None; keep runs scoped (this Air is fanless). |
| Chromium 140 / WebKit 26 (pinned bun.lock) | R02/R09/R13 | Local via repo pins | Must run outside filesystem sandbox for browser launch. |
| PostgreSQL 17.11 | R10/R11/R04 | Local per audit pins | Scoped runs; no credentials in evidence files. |
| MySQL live | R10 (skipped-suite parity) | Prior skips show it unavailable in review sandbox | Record as unavailable-live unless provisioned; never a silent pass. |
| Disposable object storage (S3-compatible) | R15 | 12 S3 tests skipped in review | Isolated storage qualification planned; needs bucket + credentials at implementation time — substitutes (local fakes) cannot establish full acceptance. |
| Linux packaging (Debian 13+ amd64) | R13 | **Deferred to user's x86 machine (UP25). Never x86-emulate on this Air.** | No Docker `linux/amd64`, no QEMU/Rosetta-Linux here. |
| Multiple workers + fault/stall injection | R04/R11/R15 | Partially local (2-worker claims, stall injection harness) | Crash-recovery + sustained load need implementation-time runs. |
| AI provider evaluations | R14 | Raw-provider fixtures exist; live quality/cost/latency not evaluated | Model-change eval + tenant budgets need live runs with credentials at implementation time. |
| Fault/stall injection fixtures | R04/R05/R12/R15 | Race-drain, recursion, DOM probes retained; S3 stall + disconnect + SIGTERM fixtures to be specified in P07 | Bounded preparation probes only; full matrices belong to implementation. |

Credentials policy: environment-provided, never written to evidence files.
Long full-tilt runs on this MacBook Air require asking first; prefer
minimal-scope reruns (per machine constraints).
