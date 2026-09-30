# Native Can tests: failure, lifecycle, coverage and reuse

Status: **lifecycle policy specified; implementation and qualification pending**.
The [post-probe reconciliation](native-can-test-design-review-2026-09-30.md)
adds host-effect admission and independent settlement facts. Preparatory checks
ran; integrated lifecycle qualification remains unrun.
This extends the [capability contracts](native-can-test-capabilities-2026-09-30.md),
[execution architecture](native-can-test-architecture-2026-09-30.md) and
[completion requirements](native-can-tests-completion-contract-2026-09-30.md).
These rules apply to every implementation lane. They are design decisions, not
results from running the proposed suite. No builds, live services, tests or
measurements were run in this round.

Ordinary Can owns plans, scenarios, comparisons, continuation/retry policy and
report reduction. Reference toolchain **R** builds the judges; candidate **C** is
the subject. External native owner **N** enforces resource/transport limits and
cleanup, including when a Can worker/controller cannot run. No failure behavior
here moves an old host test scenario into N.

The later [hard-case challenge](native-can-test-hard-cases-2026-09-30.md) applies
these rules to held browser routes, late native callbacks, descriptor writers
and SQL work that continues after a visible deadline. Its bounded experiment
register remains unrun at the integrated Can level; preparatory evidence does not establish all native settlement,
descendant cleanup or feasibility of the proposed bindings.

## 1. Keep scope, results and completion separate

Freeze the complete selected case/variant matrix, required check plans and finite
attempt policy before admission. An actual execution unit is
`case + variant + attempt`, mapped to a reviewed obligation/facet/environment in
the [migration ledger](native-can-tests-migration-ledger-2026-09-30.md). Cases,
checks, offline roots and the ledger's 292 rows are different units; do not add
their counts together or infer obligations from passing events. Append actual
attempts under existing selected cells with reason and immutable history; unused
optional retry capacity is not an unstarted selected case. Each created attempt
must be accounted for. Preparation failure before the plan can be expanded records
the requested selection and `plan=unavailable`, not a fabricated zero-case success.

| Dimension | Recorded facts |
| --- | --- |
| Intended scope | Profile/coverage-manifest identity; selected and unselected case/variant IDs with reasons; required environment matrix |
| Admission | Admitted, or blocked with the exact missing dependency, capability, host guarantee or budget |
| Execution | Not started, running, completed, broken or cancelled; cause, worker/transport identity and terminal records |
| Behavior | Matched, mismatched or undetermined; sticky named expectation results and their evidence |
| Check accounting | Matched, mismatched, not reached after a named failure, missing, or invalid evidence, for every planned check |
| Evidence | Required intervals sealed, partial or invalid; source/runtime/artifact identities, event boundaries, gaps and observation scope |
| Cleanup | Clean, failed or unresolved; released/remaining resources and all forced actions from N |
| Verification | Full required offline roots, a qualified focused closure, or incomplete verification; exact root identities and selection basis |

**Partial scope is different from incomplete execution.** A nonempty deliberately
selected subset may complete successfully for that subset. A selected case that
is blocked, abandoned, interrupted or loses evidence is missing selected work;
do not shrink the plan afterward and call the remainder a successful subset.
Every unstarted selected variant remains visible after fail-fast or interruption.

A normal named mismatch may short-circuit later checks. Record those checks as
`not_reached_after_failure`, preserving the failure. This is valid failed case
execution, not a malformed protocol, but the unperformed checks receive no
coverage credit. Bare `ok` with missing checks is a report/execution error. A
caught mismatch remains mismatched even if the function later returns `ok`.
Unknown/duplicate check IDs, wrong variant/attempt, wrong observation scope,
missing terminals and sequence gaps are errors, never zero-length success.

Distinguish three claims:

1. **Accounted execution:** all selected units have honest terminal dispositions,
   including declared short-circuit failures. Blocked or unrun units are accounted
   for but not executed.
2. **Complete exercised coverage:** every required check/facet/environment in the
   claimed scope has valid executed evidence. A checked mismatch can be exercised
   coverage; an unperformed tail cannot.
3. **Qualification passed:** the full reviewed obligation matrix and required
   offline roots have passing evidence, valid reference/suite authority, complete
   execution and clean resource receipts. Complete counts alone do not establish
   correct oracles or qualification.

Only the third claim authorizes full migration/release qualification. A green
quick profile, supplied assertion fixture, filtered engine or focused root closure
cannot stand in for it. Historical experiments/static fixtures keep their ledger
roles; unrun trials are not passes. Deleting a check and its local plan cannot
delete the independently reviewed coverage obligation.

## 2. State transitions and final reports

Run states are `recover → prepare → plan → active → quiesce → clean → terminal`.
Failures during preparation also enter quiesce/clean. Terminal records distinguish
clean completion from unresolved disposal. Quiescing never returns to active in
the same run. N can report an incomplete execution envelope without a Can report;
that envelope is not a fabricated Can verdict.

Attempt states are `planned → admitted → running → body_terminal → finalizing →
receipted`, with blocked/not-started/cancelled paths retaining the same identity.
`body_terminal` is provisional. Required browser/native observation intervals must
be sealed before a worker claims successful checks. N continues draining owned
resource/cleanup facts after worker completion; closing the worker's report stream
does not discard cleanup failures. This later tail is a separately identified
cleanup interval, not an extension silently excluded from an earlier absence check.
A late callback belonging to an already promised final seal invalidates that seal
and the affected evidence; Can's controller must record execution error even when
the worker's earlier comparisons matched. Tests needing post-cleanup behavior use
an intact outer judge over a subordinate owned scope. A worker crash has no valid
body terminal, even if earlier checks passed.

The completion boundary also includes declared host effects outside the page or
tracked child tree. Browser admission requires the host-UI contract in the
capability design. Process reaping and directory removal cannot substitute for
that observation. A known later isolation incident appends an invalidation linked
to the original run/report; consumers must not use superseded receipts to qualify
a replacement. Preserve both the earlier facts and the later failure. This is
an evidence correction, not an unbounded promise to monitor the machine forever.

Such a correction identifies the original report/receipt digests, affected
run/capability/obligations, observation source and interval, attribution limits,
and the acceptance it invalidates. Keep it with the compact qualification
registry, not only in a chat or mutable success flag. Qualification/deletion
consumers resolve the current correction chain; a missing/conflicting chain is
unavailable evidence. Later successful execution gets a fresh run identity and
cannot remove the historical invalidation. This does not require a new signing
system or make N responsible for assigning Can expectation verdicts.

The correction registry is qualification-authority data outside C's write
authority. Consumers use an identified, internally consistent registry snapshot
for the report/receipt and record its revision in their decision. An explicit
verified empty chain means no registered corrections through that revision;
missing access, missing registry data or conflicting revisions do not mean an
empty chain. When a registered later correction invalidates accepted evidence,
subsequent qualification/deletion decisions must use that correction rather than
an older cached snapshot. The selected scope and any offline freshness limit
must be explicit; an isolated stale export cannot claim current acceptance.

Normal finalization is:

1. The worker records checks, closes required observation intervals and sends one
   body terminal. N observes the worker exit and finalizes its owned case resources.
2. Can's controller combines those facts with the plan. No case passes before its
   required observations, successful worker termination and resource release are known.
3. The controller writes a bounded aggregate report, binds its digest to the plan
   and identities, and exits. Its reported success is still provisional.
4. N confirms controller completion, cleans run-owned builds/services/workspaces,
   then writes a receipt binding that report digest, selected-plan digest, run and
   toolchain identities, terminal boundaries and cleanup facts.
5. The consumer accepts only the matching pair. Missing, conflicting, stale or
   incomplete records are non-success. Atomic individual file publication plus
   digest binding does not pretend two files can be atomically committed together.

Reserve bounded control/error/receipt capacity independently of subject output.
Overflow must remain observable even when no worker can send its final message.
If even the final receipt cannot be persisted/delivered, return nonzero where
possible and leave the result incomplete. An old successful file at the report
path cannot satisfy a new run ID. Subject stdout, product CLI JSON and page events
never gain access to the private report/control channel.

| Outer disposition | Meaning |
| --- | --- |
| `0` | Every selected unit passes with complete required evidence, verification for the declared scope and clean matching receipts; selection is nonempty. This can be subset success, never implicit full qualification. |
| `1` | At least one genuine Can mismatch, otherwise valid terminal execution/accounting and cleanup for all selected units; no blocked/unrun/broken units. Valid short-circuit failures can leave explicit unexercised checks. |
| `2` | Planning, blocked selected work, incomplete verification, malformed/missing report, judge/controller failure, run limit, integrity fault, selected unrun work, or failed/unresolved cleanup. Preserve simultaneous mismatches. |
| `130` / `143` | Handled SIGINT / SIGTERM of the top-level run. Preserve interruption and cleanup state even if cleanup also fails. These proposed values apply to the POSIX hosts in scope; an unhandled kill may supply no receipt/status controlled by N. |

N may enforce nonzero for transport/lifetime/cleanup failure, but does not invent
expectation verdicts. A successful list/plan command may exit zero with `mode=list`
or `mode=plan`; it supplies no execution/qualification receipt. Zero selected
units in **execute** mode is a planning error. A first interrupt stops admission
and allows bounded cleanup; a second shortens cooperative grace, never bypasses
ownership or declares cleanup complete. Late success cannot reverse a latched
deadline, protocol error or interruption.

## 3. Failure walkthroughs

| Failure point | Native action and retained evidence | Can/coverage consequence and continuation |
| --- | --- | --- |
| Expected subject nonzero exit, crash or timeout | Return actual code/signal/operation deadline, output completeness and cleanup receipt | Can may pass only the corresponding planned checks. A signal sent is not proof of application shutdown. |
| Subject unexpectedly hangs or crashes | Enforce subject lifetime; retain partial output, forced actions and release facts | Can records mismatch or broken observation as appropriate. Later independent cases may run only after confirmed containment/cleanup; this case never silently passes. |
| Case worker spins or stops responding | External judge deadline wins; stop/reap worker and owned descendants/services; retain prior checks and missing terminal | Broken execution. Expected subject timeout cannot excuse a judge timeout. Continue disjoint cases only after clean containment. |
| Worker exits zero with early `ok`, malformed report, wrong identity or missing checks | Preserve framed bytes and protocol failure within bounds; close case resources | Error, not pass. Exit code is insufficient. Duplicate terminal/check messages cannot increase coverage. |
| Case event channel overflows or loses a terminal | Mark explicit gap/overflow outside the full channel; terminate/drain within bounds | Required evidence is incomplete. A digest or later exit zero cannot reconstruct it. Local failure may be contained; shared-channel failure stops the run. |
| Candidate `check --json` crashes, reports malformed JSON or claims rejection with exit 2 | Keep raw bounded stdout/stderr, actual executable/input identity and process result | Cannot satisfy source rejection. Can's checking helper reports a broken product protocol; negative tests of that behavior must observe it from a separate intact case. |
| Controller crashes, hangs or exits after an invalid aggregate | Stop all new dispatch; contain current jobs; retain known plan/events and emit generic incomplete envelope | No authoritative Can aggregate can be synthesized by N. Selected unfinished IDs remain outstanding; stop run. |
| N or resource service crashes | Surviving qualified owner/watchdog may contain delegated resources; otherwise no immediate-cleanup claim. Recover registered ownership on next entry | No clean final receipt. Never infer success from Can's earlier aggregate or start a new run against uncertain capacity. |
| User interrupt / terminal cancellation | Latch cause, close admission, signal bounded cooperative cancellation, then external stop and cleanup | Completed evidence survives; active units cancelled/broken as observed and queued units not started. Never promote to success. |
| Required service/engine/toolchain absent before admission | Record missing endpoint/installation/identity/guarantee without creating dependent resources | Selected unit blocked, exit 2. Other already-planned independent cases may continue; no automatic skip or replacement runtime. |
| Service disappears after admission | Record connection/transport failure and ambiguous effects; preserve namespace ownership | Execution error unless that exact loss is deliberately induced and checked as subject behavior. Do not relabel as preflight skip. |
| Build/input/reference/shared artifact identity changes | Invalidate the affected artifact/authority; retain observed digest and users; stop live admission | Shared trust failure, not cache miss or automatic rebuild. Prior dependent evidence may be invalid; no qualification until a fresh valid run. |
| Operation acknowledgment is lost after possible write/commit | Return indeterminate; preserve operation ID, dispatch/contact record and pending ownership | No blind retry under a new ID. Can may use bounded independent read-only reconciliation; unresolved effect stops new live cases in the initial policy. |
| Directory/resource remains after cleanup | Keep owner/identity, error and charged quota/lease; stop live admission | Behavior may have matched, but cleanup failed/unresolved means non-success. Recovery cannot rewrite the old run as passed. |
| Browser causes unexpected native credential/permission/account UI | Stop admission; record observed UI and attribution limits; close only proven owned resources, preserve unresolved effects | Isolation failure, even when route checks and process cleanup succeed. No automatic dismissal, Keychain changes or passing retry. Missing host observation prevents qualification. |

Testing a crashed runner, malformed suite report or cleanup failure requires an
**outer intact Can test and owner** observing a subordinate runner as its subject.
The subordinate's expected failure is data. Its resources must remain within an
outer owned/delegated scope. Killing the actual top-level judge/N cannot constitute
a passing self-test, and an expected subordinate leak is not permission to leave
real resources behind. Final outer cleanup must still be confirmed.

## 4. Stop, drain, cleanup and recovery

Use this order on abnormal termination:

1. Latch the initiating cause. Stop new scenario/resource admission in the affected
   scope, but keep an owner-only lane for cancellation, drain and disposal. Reject
   late mutations; retain already admitted operations and bounded late facts.
2. Signal cooperative cancellation to live Can code for a finite grace. N retains
   authority and must act even if it does not respond. Revoke shared artifact access
   only through safe lease disposal; never delete files out from under live users.
3. Stop producers and quiesce admitted operations: resolve/abort paused browser
   routes, end clients/peers, stop subject processes, drain owned callbacks/streams,
   and confirm descendant/descriptor release. Release gates only by declared safe
   cancellation semantics; never issue new writes merely to make cleanup progress.
4. Resolve/terminate owned transactions and connections, then dispose namespaces,
   object prefixes and workspaces after all possible writers are stopped. An
   indeterminate external commit or future server-side write is not cleared merely
   by closing the local socket. Independent reconciliation or an enforced remote
   fence is necessary before claiming quiescence.
5. Release only confirmed resources and their quota charges. Append cleanup errors
   to the original failure; preserve unresolved ownership and emit bounded receipts.

The dependency graph, not a universal resource-type order, controls step 3/4:
closing a browser before its server can preserve final events, while a deliberate
server-shutdown case may already have closed the server. Every wait has a deadline.
No `finally`, bottom-of-function call, successful `RemoveAll` attempt, process-group
signal or exited leader by itself proves all dependent resources are gone.

Continue by default after ordinary mismatches and isolated judge/driver faults
**only when** N proves case containment, completed cleanup and intact shared
identity/transport/capacity. Can may choose a stricter fail-fast policy in the
plan. If fail-fast leaves selected cases unstarted, retain them and return 2;
the earlier mismatch remains visible. Initial policy stops all live admission
for unresolved cleanup/effects, unknown descendant liveness, owner/journal damage,
run-wide capacity/enforcement loss, shared artifact corruption, controller failure
or interruption. Read-only diagnosis and bounded cleanup remain allowed.

Recovery precedes new live admission. Inspect only registered run ownership and
bounded candidate records. Require owner/start identity, path identity, leases
and a qualified inactivity proof. Active/foreign work remains untouched. A path
replaced by a different object, unknown schema, missing marker, unavailable liveness
tool, incomplete scan or uncertain remote ownership is not deletion authority.
Likewise, an argv/path substring, matching process name or reused PID is not
signal authority. Track verified creation identities and owned descendant edges;
a foreign process mentioning the workspace may block safe deletion but does not
become a process the runner may kill. Qualify detached-descendant behavior before
claiming containment beyond the explicitly owned graph.
Report unresolved prior **owned** resources and block initial live admission until
reconciled; do not let a random foreign directory block or be claimed by the suite.
Discovery must not depend solely on markers inside possibly abandoned directories:
the outside-owner reservation ledger covers acquisition before marker creation.

Recover cleanup, not execution. A resumed invocation creates a new run ID and
does not resume a partially executed Can function, replay uncertain mutations or
adopt a prior build as a cache hit. Keep compact earlier failure/recovery evidence.
An old run without a final receipt remains incomplete even if later cleanup is
successful. Recovery backlogs have explicit limits; exhausting them reports pending
owned work and refuses live admission rather than growing storage indefinitely.

## 5. Concurrency and resource admission

The initial execution policy is deliberately serial across cases:

| Work | Initial limit and ordering |
| --- | --- |
| Can controller | One for the run |
| Live case worker | One admitted case through its final cleanup; acquire the next slot only after release is confirmed |
| Offline verification | One worker, completed before live case execution; all roots at bootstrap, focused closure only after planner qualification |
| Build producer | One active compiler/bundle production job per run; no speculative/background prebuild overlap with unrelated live work |
| Build needed by current case | May run inside that case's scope while it waits; its compiler and any assertion children consume the shared process/memory budget |
| Browser matrix | One active owned context by default; engine legs sequential. A case needing several contexts/pages declares their finite peak before admission within the profile's explicit cap. Browser internal child processes count in the full tree budget. |
| Concurrent subjects within a case | Permitted only by explicit finite demand for that case: e.g. two companion workers, held requests and a controlled peer. This does not create additional uncounted case slots. |
| Multiple local suite invocations | One live managed run on the host initially, enforced by shared host admission across repositories/ownership roots; a different root cannot bypass the gate. Read-only listing is allowed. |

Can chooses dependency order and ready cases. N atomically reserves each case's
declared peak scarce-resource demand across all phases before starting it; include
resources that remain held while a later phase runs. Phase leases consume that
reservation, not acquire a second global allowance. Unavailable capacity yields
bounded wait or blocked admission before partial acquisition can hold resources
needed by a producer. State required build/browser/database conflicts in the plan.
A reservation is capacity, not possession of the active producer token: a case
requesting a build must let that producer acquire and use the one execution slot.
No semaphore recursion or hidden compiler/browser/driver fan-out is allowed.
Shared externally managed DB/browser services are identified dependencies; charge
their owned connections/contexts and do not claim authority over the whole service.
The host admission mechanism must cover every managed ownership domain before
claiming a host-wide cap; independent per-directory locks are insufficient. This
does not claim to schedule or cap unrelated host applications. Reserve N's
enforcement/disposal capacity within the envelope so subjects cannot consume the
last capacity needed for cleanup or reporting.

Process counts include R preparation, Can workers, C, compiler children, probes,
browser processes and native helper services as appropriate to each phase. A
different-language subprocess, internal thread or old `t.Parallel` convention does
not exempt work from the run envelope. Raising concurrency is a reviewed profile
change with host enforcement evidence, not `CPU count` autodetection or a silent
copy of the old three-heavy-slot setting.

### Initial finite envelopes

These are conservative **policy ceilings, not measured cost predictions**. They
do not authorize running anything now. Exceeding them is explicit incomplete
execution/admission failure; it must not cause silent skips, unlimited fallback or
an automatic increase. Named case profiles can request different finite values
through reviewed configuration while preserving the shared host envelope.

| Limit | `quick` job | `bounded-job` for larger planned legs |
| --- | ---: | ---: |
| Total run wall time, including preparation/final cleanup | 10 minutes | 45 minutes |
| Reserved final cleanup portion of that total | 60 seconds | 180 seconds |
| Aggregate recovery/preparation/planning, including all offline verification | 5 minutes | 10 minutes |
| Default live-case body deadline | 60 seconds | 120 seconds |
| Maximum explicitly declared case body | 180 seconds | 30 minutes |
| Maximum one build/preparation producer | 180 seconds | 10 minutes |
| Per-case cleanup within remaining run budget | 15 seconds | 30 seconds |
| Cooperative cancellation grace | 2 seconds | 2 seconds |
| Offline assertion root default | 5 seconds | 5 seconds |
| Owned resident process-tree ceiling | 64 processes | 64 processes |
| Owned active memory ceiling | 4 GiB | 6 GiB |
| All task-owned temporary data, including staging and ready builds | 512 MiB | 2 GiB |
| Workspace entry/depth ceiling | 50,000 entries / depth 64 | 100,000 entries / depth 64 |
| Case evidence stream | 1 MiB | 1 MiB |
| Default stdout and stderr capture, each process/stream | 1 MiB each | 1 MiB each |
| Run-wide captured streams/events/journal/report data | 64 MiB | 64 MiB |
| Final Can report / N receipt maximum | 16 MiB / 1 MiB | 16 MiB / 1 MiB |
| Owner emergency/control reserve within run data budget | 1 MiB | 1 MiB |

An executable profile must additionally give finite type-specific handle/pending
request/byte ceilings for its selected capabilities; no omitted field means
unlimited. Establish a 10 GiB available-disk admission floor and recheck before
allocation; this is an admission guard, not a claim to reserve disk against other
programs. Owned strict disk/memory/process limits still require a qualified host
mechanism; if unavailable, the profile is unavailable rather than best-effort.
Installed shared toolchains/browser engines are outside temporary-byte accounting
but must be preprovisioned/identified, never copied into every workspace.

Preparatory browser peaks (1.72 GiB, 15 observed processes and about 12.8 MiB
scratch) are bounded-run observations, not capacity predictions or enforcement
proof. Keep the existing ceilings and serial admission; do not reduce budgets,
raise concurrency or certify strict limits from these samples. An executable
host profile must identify the enforcing mechanism, charged scope and finite
overshoot bound for each promised limit, including detached children and writes
outside workspace helpers. Poll-and-kill has no demonstrated strict bound here.
Without qualified enforcement, integrated execution on that profile remains
unavailable. Small reviewed preparatory jobs and their weaker claims cannot
silently become a best-effort full-suite profile.

Use N's monotonic clock and these absolute deadlines:

- `D_end = entry_time + total_run_budget`.
- `D_execute = D_end - final_cleanup_reserve`.
- `D_prepare = min(entry_time + preparation_budget, D_execute)`; all recovery,
  suite production, verification roots and planning share this aggregate cap.
- At case admission time `t`, require `t + declared_body + case_cleanup <= D_execute`.
  If it does not fit, stop/mark the selected unit unstarted for insufficient run
  budget; do not launch it with a secretly shortened promised observation window.
- `D_body = t + declared_body`; `D_case_cleanup <= min(body_end + case_cleanup,
  D_execute)`. `body_end` is N's latched fault/deadline or confirmed normal
  terminal-and-exit time, never later than `D_body`; cooperative stop grace is
  charged to cleanup. Reserve equivalent disposal time for non-case build producers.

Operation deadlines are bounded by the owning body/phase and resource lifetime.
Unused body time does not extend cleanup allowance; unused cleanup time remains
available to later admitted work. At a run interrupt/hard stop, final cleanup
can use the remaining reserve up to `D_end`. All disposal steps share the absolute
deadline, not a fresh full allowance each. Reporting uses reserved space/time;
publication failure stays explicit. A limit reached at the same instant as late
success wins. A `wait` deadline may leave a subject alive, but its separate hard
lifetime applies. Judge deadlines cannot count as expected subject timeouts.

Browser/lifecycle descriptors must override the short default when their declared
startup, held-work and observation path needs longer. A retained 120-second drain
cannot be assigned a 120-second body that must also launch, observe and finish it.
Validate the mandatory critical-path budgets before admission. Maxima in the table
are not a promise that all maximum-sized phases fit together or that every selected
case can run in one job; remaining-time admission still applies.

Existing lifecycle/browser contexts reach 12–30 minutes; `quick` is therefore a
chosen subset, not a claim that the full suite fits in ten minutes. Full
qualification can combine bounded jobs on appropriate hosts. No profile may
silently omit a slow case, shrink its required observations or count a run-budget
timeout as that case's expected subject timeout. The proposed ceilings need later
bounded enforcement/admission controls; changing them is explicit, not evidence
that the current unmeasured values are sufficient for every retained case.

## 6. Build reuse and invalidation

Compile the suite once with R per run and lease that immutable generation to the
controller/workers. All-source checking and required root verification precede
live use. Candidate builders never replace R or compile an authoritative judge.
If R cannot understand new suite APIs/syntax, use the architecture's reviewed
reference-refresh gate, not a fallback to C.

Can maintains the run-local build plan/memo; N owns production processes, staging,
artifacts and leases. The reuse key contains:

- Source-tree namespace/content digest, including relevant absence/link metadata;
  dependency/lock/manifest/asset identities and generated build inputs, including
  paired browser/server manifests where relevant.
- Compiler, runtime/native catalogue, observer/builder recipe and schema identities;
  pinned upstream archives and compiler toolchain identity when distributing a bundle.
- Target OS/architecture/backend, build mode, flags/features/options and any
  environment values that affect output, bound through a non-disclosing scoped identity.
- Required validation/verification recipe and evidence kind, so a build omitting
  assertions or production processing cannot satisfy a stronger requirement.

Reconcile declared inputs with actual-read/directory-resolution facts from the
build. Unexpected inputs invalidate reuse and any stronger closed-input claim;
they cannot be added silently to an already-published key. Unknown dependencies
or build-affecting inputs mean **not reusable**. A mutable path, mtime, case name
or compiler version string alone is not a key. Runtime
credentials and mutable DB/page/server state are not cached artifacts; when
configuration is genuinely embedded in output it becomes an identified build input
without writing secrets to the public key/report.

Entry states are `absent → reserved → building → validating → ready → retiring →
disposed`, plus failed/invalid/unresolved outcomes. Publish `ready` only after the
producer's valid terminal result, artifact identity/content verification and required
validation receipt; use atomic publication of the sealed generation. Failed or
partial fills are never hits. One producer owns a key; other consumers join its
result within their deadlines, not launch duplicate compilers or steal an old lock.
This single-producer key is `(run_id, content_key, reuse_mode, production_id)`:
normal reuse shares one production ID; `fresh_subject_build` assigns a distinct
production/attempt ID and bypasses joining another build. Preserve the content
key separately so determinism comparisons can verify identical inputs without
accidentally comparing one cached artifact to itself. The one-producer concurrency
limit still applies to these distinct fresh executions.

Every hit verifies the named generation/lease and required validation identity.
Only immutable artifacts are shared. If unexpected mutation/corruption is detected,
mark invalid, stop live admission and identify all consumers since the last proven
integrity boundary. If the affected interval is unknown, invalidate all dependent
evidence. Do **not** transparently repair/rebuild and preserve prior passing claims.
A fresh run after cleanup may rebuild; intentional corruption cases use private
copies whose expected failures are checked by an intact judge.

Build/determinism/reproducibility/relocation/corruption/build-failure cases declare
`fresh_subject_build`: distinct production executions and output roots, even for
identical keys, while read-only installed toolchains may still be shared. Do not
reuse a compiler rejection or prior failed production as a new observation. A cache
hit must never satisfy the behavior being tested. Other cases may reuse the
successful artifact but get fresh mutable homes, DB namespaces, pages, source
mutations and run output. Case functions and their expected observations remain
ordinary Can, not native cache callbacks.

If a waiter cancels, remove only its lease/request. A run-owned producer can
continue only while a remaining declared consumer needs it and the run is active;
when none remain, cancel it and dispose staging within bounds. Run interruption
cancels producers regardless of waiters. A failed producer blocks dependent cases
with its actual cause; never relaunch it automatically for every waiter. An explicit
diagnostic retry needs a fresh attempt/output and keeps the first failure visible.

Reuse consumes the same temporary-byte budget as everything else. Can may retire
unleased entries before admitting more work. An entry still leased cannot be
evicted; an insufficient budget blocks the next phase. Default retention ends at
run cleanup: delete staging, full bundles, copied inputs and ready artifacts; keep
compact keys, manifests, validation/report and recovery facts. No cross-run private
cache, unowned checkout or retained bundle is introduced. Crash recovery disposes
old ownership rather than converting abandoned artifacts into fresh cache hits.

## 7. Missing services, retries and cross-job qualification

Profiles declare engine/DB/storage/target variants before capability probing.
Missing selected service or enforcement guarantee is blocked. A quick profile may
exclude a family explicitly, but lack of credentials/engine installation cannot
silently rewrite a full-profile plan. A nominally optional diagnostic leg receives
no required coverage credit; optionality must be declared before execution.
Preflight does not automatically start/install shared services. If the plan owns
a service launch, startup and failed acquisition follow normal scope cleanup.

Attempts default to one; a diagnostic retry mode declares a finite maximum and
trigger policy in the original plan. Actual retry IDs are appended when chosen,
without enlarging the selected coverage matrix or resetting the run budget.
Readiness polling is bounded Can logic within an attempt,
not repeated case passes. Transport retry is allowed only when the operation
contract proves no effect or Can has reconciled the effect. Same-ID rejoin queries
the existing operation; a new ID is a new effect, not deduplication.

Diagnostic reruns retain every attempt, failure, candidate/input identity and
cleanup receipt. A later pass cannot make the current run clean after a real
earlier mismatch/execution fault. There is no automatic flaky-test exception for
qualification. A corrected fresh run can qualify after the cause is resolved;
this does not require all historical runs ever to have passed. Unknown prior
effects/cleanup must first be reconciled, even when new case inputs are different.

Can's qualification reducer may combine bounded jobs only against an explicit
reviewed evidence manifest and campaign ID with an append-only index of registered
jobs/attempts, including failed, blocked and interrupted ones. Register a campaign
job before execution and reconcile every entry; a collection assembled only from
passing report files is insufficient. The index covers that declared campaign,
not an unverifiable claim to know every historical exploratory run. Bind
suite/coverage/check-plan identity, accepted R
and N profiles, candidate source/build recipe, exact per-platform artifacts,
fixture variants, required environments and observation/verification scope.
Platform artifacts can differ only where the declared target matrix requires it;
do not require macOS and Linux binaries to have one hash or merge arbitrary source
revisions. Each job supplies its own matching report/cleanup receipt. Missing,
blocked, stale, invalid or unexecuted matrix cells remain visible. Do not union
checks from different partial attempts to manufacture a case that never completed.
Keep superseded failures and the explicit corrective-run selection; never select
only the best attempt from unexplained flaky outcomes.

## 8. Contracts every implementation lane must inherit

| Lane | Required behavior / future bounded negative controls |
| --- | --- |
| Can registry/reducer | Stable selected/check IDs; subset versus incomplete distinction; sticky mismatches; not-reached versus missing checks; zero/all-blocked cases; failed retry never erases history |
| Native owner/transport | External deadlines, pre-effect ownership, reserved failure channel, identity/sequence/frame validation, cancellation race, worker/controller/N death, failed close and quota retention |
| Workspace/recovery | Partial allocation before markers, directory replacement/symlink, active/foreign lease, incomplete liveness scan, failed deletion and later safe recovery without old-run pass |
| Process/services | Child hangs/exits with descendant-held pipes, lost acknowledgments, service absent versus lost, uncertain commit, bounded reaping and no cleanup-before-writer-stop |
| Browser/native facts | Pending route/body/thenable at cancellation, late error during close, gap/overflow, observer faults, expected subordinate crash with intact outer cleanup |
| Builds/bootstrap | Same-key single producer, mismatched input/toolchain/options, failed fills, wait cancellation, exact validation receipt, leased eviction refusal, corruption visible, mandatory fresh builds |
| CLI/CI/report consumer | No stdout-to-control confusion; malformed/stale/missing report or receipt, deadline-at-success boundary, successful subset labeled partial, full matrix with one missing cell nonpassing |

None of these controls has been run. Existing [harness reuse](../../tests/integration/harness_test.go),
[temporary cache ownership](../../tests/support/tempcache/cache.go) and
[assertion supervision](../../compiler/internal/driver/supervise.go) are source
bases, not proof that the new contracts are satisfied. Current age-based fill-lock
reaping, automatic rebuild of contaminated hits, worker-local cleanup and ignored
removal errors must not be imported as the new policy without the stated changes.

Three fresh Jev consultations supported separate outcome dimensions, continuation
gated by containment and same-run verified reuse with visible corruption. The
[decision evidence](preparation/native-can-test-lifecycle-2026-09-30/findings.md)
preserves requests, responses, wording checks and alternative reconciliation.
[Source reviews and final review](preparation/native-can-test-lifecycle-2026-09-30/review.md)
record corrections; [documentation validation](preparation/native-can-test-lifecycle-2026-09-30/validation.json)
does not constitute runtime qualification. Host enforcement mechanisms, exact Can
declarations/protocol schemas and full ledger-to-binding implementation remain
engineering work; no additional user product preference is needed for these rules.
