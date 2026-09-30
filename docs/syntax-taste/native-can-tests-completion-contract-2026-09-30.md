# Native Can tests: completion contract

Status: **completion requirements defined; migration not implemented or verified.**
This is step 1 of the design work. It records the user's requirements and derives
observable acceptance conditions from them. It does not select an execution
backend, introduce syntax/APIs, classify every individual file, or authorize a
build, benchmark, migration or deletion in this documentation round.

The subsequent [execution architecture decision](native-can-test-architecture-2026-09-30.md)
selects existing Can TypeScript/Bun execution, a Can controller and fresh case
processes under generic native supervision, with a qualified reference toolchain
separate from the candidate. None of its implementation or trust gates is marked
complete by choosing that architecture.

Step 2 now has a [coverage migration ledger](native-can-tests-migration-ledger-2026-09-30.md)
with declaration-level obligations, a complete file map, shared capabilities and
conditional harness-deletion gates. All replacements remain pending.

**The migration is complete when every retained testing obligation within the
scope below has a Can-authored scenario and oracle, the required execution
evidence exists, the old host harnesses and their active callers are retired,
and resource/cleanup guarantees have been demonstrated.** Historical records and
necessary fixtures may remain with explicit roles. There must be no unresolved
coverage obligation hidden by relocation, skipping, or reclassification.

An *oracle* is the expected result and the logic deciding whether an observation
satisfies it. A *host harness* is authored Go, TypeScript, JavaScript, Python,
shell or other non-Can code that supplies test scenarios, suite policy or oracles.
Native language/runtime infrastructure is described separately below.

## 1. Requirements and their origin

| ID | Requirement | Basis |
| --- | --- | --- |
| R1 | Tests support arbitrary normal Can functions and reusable helpers, subject to ordinary target/effect rules | Explicit user clarification |
| R2 | Can owns test scenarios and expectations; host test harnesses are eliminated | Explicit user clarification and current request |
| R3 | Preserve current coverage obligations; distinguish executable checks, historical experiments and static fixtures | Current request |
| R4 | Bound laptop load and disk use; reliably clean owned temporary work | Current request and repository instructions |
| R5 | Test execution may use a backend other than TypeScript | Explicit permission, not a requirement to build another backend |
| R6 | No compatibility requirement for old syntax, ABI, generated layouts or goldens | Repository instructions; preserve current behavior contracts instead |
| R7 | AI coding agents are the intended Can authors; assess authoring by generation reliability, explicit contracts, local reasoning, precise diagnostics and safe edits, rather than human typing convenience | Explicit user clarification after the authoring design |

The conditions below make these requirements reviewable. They do not turn the
initial research recommendation into implementation evidence. The
[architecture decision](native-can-test-architecture-2026-09-30.md) now adopts
TypeScript/Bun plus native supervision as the initial backend; the
[comparison](native-can-test-backends-2026-09-30.md) records its alternatives.

## 2. Scope and inventory boundary

The starting inventory is [224 tracked files under `/tests`](preparation/native-can-tests-investigation-2026-09-30/source-inventory.json),
covering eight families. These are files, not 224 runnable tests. The inventory
was rechecked against HEAD `447f2273c468fd7a1149e6fd6fc93d3666ff09e2` for this
contract: no tracked `/tests` paths or contents had changed since capture.
Reconcile later additions and changes when creating the migration ledger and
again before final acceptance; this snapshot must not become a scope loophole.

Scope follows coverage dependencies as well as paths. It includes:

- Scenarios, fixture setup choices, suite selection/scheduling, comparisons,
  reporting policy and support machinery currently supplying `/tests` coverage.
- Test policy in external callers such as
  [distribution qualification](../../distribution/qualify.py#L128), and any
  delegated runtime test oracle used to satisfy an in-scope obligation.
- CI, release and host entrypoints that invoke the affected harnesses, consume
  their results or depend on their support files. Those integrations need a valid
  replacement before shared files are removed.

This does not automatically include every compiler or runtime unit test in the
repository. Independent suites may remain in their present languages. Where an
external suite supplies an in-scope oracle, a Can-owned replacement must cover
that obligation; invoking the external suite cannot be credited as its migration.

Ignored dependency installations are not authored tests. Their provisioning and
use still have to meet the resource policy. Ordinary dependencies, browser
engines, databases and the compiler/runtime need not be rewritten in Can.

## 3. What may execute outside Can

**C1 — Ordinary Can remains the test language.** Test bodies and helper libraries
may use the same function, type, closure, error, coordination and module features
as other Can programs. No restricted test-only subset or silent host fallback may
be presented as full support. Normal target/effect restrictions remain valid:
browser-only operations need the appropriate target, and offline assertions keep
their supplied-fixture contract. Live scenarios must have an explicit live
execution path.

**C2 — All test policy is authored in Can.** This includes case registration and
selection, scenario order, fixture choices, case-specific readiness conditions, retries,
scheduling policy, expected results, comparisons, coverage aggregation and
reporting decisions. Static manifests may index Can case entrypoints and hold
fixture inputs, expected vectors or capability metadata. Can defines the cases,
selects them, implements their action sequences and applies comparison policy.
A manifest containing executable scenario scripts would conceal test code as
data and does not meet this requirement. Ordinary Can table-driven cases over
static input/expected vectors remain valid. Helpers such as expecting a compiler
rejection are ordinary Can code.

**C3 — Native infrastructure exposes mechanics and observations.** It may implement
compilation, runtime operations, browser-driver calls, process isolation, hard
resource limits, transport, ownership tracking and cleanup/recovery. It may
return values, bytes, events, exit status, timeout or execution failure. It must
not hard-code migrated cases, their action sequences or their expected answers.
Generic enforcement may fail a broken execution protocol or report a controller
crash without waiting for unavailable Can code; it may never synthesize a passing
test verdict from absent or invalid evidence.

CI and launchers may provision dependencies, select a documented suite profile,
invoke the Can entrypoint and propagate its exit status. They must not retain
case-specific qualification, expectation or result-aggregation logic. Generated
TypeScript/JavaScript is an output of compiling Can, not an authored host harness.

Examples of the boundary:

| Acceptable role | Role that would leave a host harness |
| --- | --- |
| Launch a process and return separated streams, status and signal | Run the compiler, match an expected diagnostic and return `test passed` from native code |
| Browser operations return DOM, network and console observations | An old Playwright scenario, unchanged, launched by a Can wrapper |
| A generic native probe constructs or observes a host value Can cannot represent | A probe with a particular test's expected trace or pass/fail assertion embedded in it |
| Supervisor enforces a deadline and reaps a worker | Supervisor selects retries or declares expected outcomes for named test cases |
| SQL schema/seed data applied by a Can-selected scenario | A foreign driver owns the whole database scenario and checks its answer |

Renaming a harness, putting it in an adapter, moving it outside `/tests`, embedding
it as a string, or generating it from test data does not change its role.

## 4. Coverage preservation and material classification

**C4 — Account for obligations individually.** Each executable check and fixture
dependency needs a documented disposition. The later migration ledger must link
the old source/behavior to its replacement Can case(s), required environment and
observation, or to evidence explaining retirement. Consolidation is allowed when
the combined checks preserve every retained obligation. File/test/root counts
and old generated layouts are not acceptance criteria.

| Disposition | Completion condition |
| --- | --- |
| Retain current coverage | Can scenario/oracle, required observations and successful qualification evidence exist |
| Consolidate duplicate coverage | Every old obligation maps to an identified replacement; unique qualifiers remain covered |
| Retain fixture/input | Role and consuming Can cases are recorded; content is an input or subject, not concealed orchestration/oracle code |
| Preserve historical evidence | Mark it as historical with original scope and provenance; do not execute it as a current gate or count it as a current pass |
| Retire obsolete experiment/check | Record what is removed, why it is no longer a current requirement, and evidence that no retained obligation or active caller depends on it |
| Unresolved | May appear during design/migration; prevents declaring the migration complete |

Difficulty of porting, lack of a native API, a missing service or a failing test
does not establish obsolescence. A historical filename or an unadmitted prototype
label alone does not establish that all behavior in the file is irrelevant.
Retiring an implementation requires disposing of its executable callers too.

Static JSON, SQL, bytes, HTML, malformed source and expected vectors need no
mechanical translation into Can statements. Foreign-language source can remain
as a necessary fixture when that source or its behavior is itself the subject
under test; it cannot drive the surrounding scenario or issue the test verdict.
Stored expected data is consumed by Can comparisons. Preserve exact bytes only
where they protect an identified current contract.

The initial classification is deliberately per role, not a blanket folder action:

| Family | Starting classification and guardrail |
| --- | --- |
| `integration/` | Predominantly current executable integration/end-to-end checks plus their fixtures; preserve compiler/CLI, service, database, browser and release obligations |
| `failure-conventions/` | Mixed live regression checks, Can positive/negative projects, comparison projects and historical measurement reports; preserve retry/owner/failure contracts and checks that deliberate defects are detected |
| `conformance/` | Executable native-runtime qualification and rejection controls; expectations must move to Can while observations remain independent |
| `support/` | Resource/cache mechanics plus their tests; preserve ownership, contamination, concurrency and cleanup obligations even if the implementation is replaced |
| `baseline/` | Mixed frozen registries/reports and executable toolchain, reproducibility and documentation checks; inspect current obligations before archiving historical material |
| `host-discrimination/` | Unadmitted prototypes with runnable comparisons, admission checks and recorded choices; distinguish guards against prototype leakage from historical size/shape comparisons before retaining or retiring obligations |
| `authoring-policies/` | Registered comparison fixtures and recorded deterministic baseline/reference legs, with no current runner in this family; agent trials are recorded as unrun; identify any still-current policy obligations without inventing trial passes |
| `browser-controls/` | Historical experiment evidence; its recorded checker/build results are not live-browser evidence or a replacement for current browser tests |

Sources: [baseline](../../tests/baseline/README.md),
[failure conventions](../../tests/failure-conventions/README.md),
[prototype status](../../tests/host-discrimination/README.md),
[authoring trials](../../tests/authoring-policies/README.md), and
[browser evidence limits](../../tests/browser-controls/x-r02-1.md).
No individual test or file is deleted or finally retired by this classification.
For example, the historical browser-controls document does not retire the
current [browser-controls integration scenarios](../../tests/integration/browser_controls_test.go)
or their required browser matrix. Similarly, archived experiment results do not
replace live checks of a still-current compiler authoring policy.

**C5 — Preserve what a test can detect.** Positive cases alone are insufficient
when existing checks require compiler rejection, failed assertions, wrong retry
counts, hostile inputs, faulty native APIs, corruption or cleanup failures.
Retain those negative controls and the necessary independence of observations.
Checking a production adapter against itself must not replace an existing
independent database/native observation without equivalent detection evidence.

**C6 — Exercise the claimed production path.** Cases claiming production codegen
or runtime coverage must compile and execute production artifacts through the
appropriate backend and, where required, actual browser/database/service.
Compiler-rejection cases stop at the intended rejection phase. Offline supplied
fixtures, raw-exchange checks and live observations retain distinct evidence
roles. An alternate test executor passing does not itself qualify production
emission. Identify the runner/executor and candidate independently in evidence;
follow the reference selection/refresh protocol in the architecture decision.
The first actual qualified reference artifact still requires evidence.

## 5. Honest outcomes and qualification

**C7 — A complete pass must be reconstructible from evidence.** Reports identify
the suite/profile, selected cases, required coverage, source/toolchain/candidate
provenance, environment/capabilities, case outcomes and cleanup outcome. Exact
field names and serialization are deferred. Listing/filtering must be usable
without implying that unselected cases ran.

A case can pass because an expected rejection, crash or other negative behavior
was observed and checked by Can. An unexpected timeout/crash, malformed or missing
report, assertion failure, missing required service, or unresolved cleanup cannot
be counted as a passing case. Preserve useful distinctions among failure, skip,
blocked and incomplete execution in the suite's reporting semantics.

A successful selected subset may be reported as successful for that subset.
It must not be labelled full-suite success. Zero selected cases, all-skipped
runs, early termination, disabled browser legs and unrun experiments cannot
supply complete coverage evidence. Unexpected cleanup failure prevents clean
successful completion even if behavioral expectations passed.

Complete migration qualification must cover all retained obligations and their
required environments. Evidence may come from bounded jobs on appropriate hosts,
with compatible recorded source/input identities; it need not be one huge run on
the laptop. A quick local subset is insufficient to declare overall completion.
No broad runs or measurements are authorized by writing this contract.

## 6. Resource limits and cleanup

**C8 — Execution is explicitly bounded.** The final design must publish defaults,
configuration and enforcement for concurrency, worker/suite deadlines, output and
task-owned temporary storage. Bounds must address non-yielding code and child
process trees. Do not assume a timer within a stuck worker can stop that worker.
The subsequent [lifecycle contract](native-can-test-lifecycle-2026-09-30.md#5-concurrency-and-resource-admission)
sets initial finite policy ceilings and phase admission. Host enforcement and
executable profile qualification remain required; an unspecified promise to be
lightweight does not satisfy completion.

Reuse candidate builds within a run when the case permits it; tests of building,
determinism or corruption may need separate builds. No persistent private build
caches, duplicate dependency/source trees or retained full bundles by default.
Ordinary installed/shared toolchains and explicitly managed bootstrap artifacts
are not disposable per-run caches; their ownership and retention must be clear.
Performance work remains deferred. Qualification must use bounded correctness
checks appropriate to the change until broader runs are explicitly resumed.

**C9 — Ownership and cleanup are part of correctness.** Register cleanup when
allocating temporary storage or other owned resources. Release owned processes,
handles and workspaces after success, failure and handled interruption. A killed
owner cannot promise to execute cleanup; parent supervision or later recovery
must cover that case. Recover abandoned work only with ownership, path identity
and inactivity established. Preserve active, foreign and caller-owned resources;
uncertain liveness must not authorize deletion.

Browser isolation also covers host effects outside its DOM/profile/process tree.
Test-only credential behavior and the absence of unexpected native UI require a
qualified observation boundary; headless launch, process exit and removed scratch
alone do not establish it. Missing observations are unknown, and a known later
host-effect failure invalidates the affected acceptance evidence. See the
[post-probe review](native-can-test-design-review-2026-09-30.md).

Keep compact evidence rather than execution workspaces. Heavy diagnostic
retention requires an explicit request, recorded expiry and automatic reclamation.
Report cleanup failures and unresolved owned resources. Do not clear shared
system caches, Docker volumes or swap to compensate for task-owned leaks.

Demonstrate the lifecycle through bounded controls for normal exit, test failure,
timeout, interruption, forcibly terminated owners, abandoned recovery, concurrent
or foreign work, and cleanup failure. Observations must show that cleanup and
recovery obey ownership rules; an appended remove-directory call is insufficient.

## 7. Final acceptance evidence

The eventual completion review must have all of the following:

- [ ] A reconciled file/obligation ledger with no unresolved dispositions, including
  delegated oracles and external callers; all retained cases map to Can source.
- [ ] A source and invocation audit showing no remaining active in-scope host
  harness, including embedded/generated foreign scenario code. Allowed native
  infrastructure and fixture exceptions have explicit roles.
- [ ] Reproducible commands and current evidence for every required coverage
  family/environment, with source/toolchain identities and honest partial scopes.
- [ ] Evidence that expected failures and deliberately broken controls are
  detected; invalid or missing results cannot produce a complete pass.
- [ ] Evidence of ordinary Can helper support, preserved offline/live rules and
  appropriate production artifact execution for the coverage claimed.
- [ ] Documented enforced resource limits and lifecycle evidence, with no
  unreported task-owned leftovers or unexplained retention.
- [ ] Updated CI/release/host callers, retired executable harnesses and usable
  instructions for discovery, selection, execution and diagnosing failures.

These boxes are migration acceptance gates and are intentionally unchecked.
The [coverage/disposition ledger](native-can-tests-migration-ledger-2026-09-30.md)
now records replacement and deletion obligations. The subsequent
[test authoring design](native-can-test-authoring-2026-09-30.md) works through five
ordinary Can cases, mandatory offline assertions, live execution and reporting.
The [execution architecture](native-can-test-architecture-2026-09-30.md) now fixes
the starting backend and authority boundary. The [capability contracts](native-can-test-capabilities-2026-09-30.md)
and [failure/lifecycle rules](native-can-test-lifecycle-2026-09-30.md) specify shared
operations, failure behavior, coverage, concurrency and reuse. Parallel
implementation tasks still require concrete binding/enforcement design; these
documents do not satisfy the unchecked migration gates.

Supporting research: [inventory and capability investigation](native-can-tests-investigation-2026-09-30.md)
and [execution backend comparison](native-can-test-backends-2026-09-30.md).
