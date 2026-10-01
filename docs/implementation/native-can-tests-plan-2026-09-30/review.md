# Implementation-plan review

Status: planning audit on 2026-10-01 against implementation HEAD
`37e1e06d8b52567a8e6dd8ce9d68bbe93e441559`. This round changes records and
scheduling only; historical implementation is audited, not newly executed or
qualified. The findings below from the original planning review are historical;
the current audit section supersedes their progress statements. See the
[main plan](../native-can-tests-plan-2026-09-30.md) and
[static validation](validation.json).

Three independent planning work streams reviewed foundation/trust, capability
mechanisms and migration coverage. Their draft inputs remain under
`docs/syntax-taste/preparation/native-can-test-implementation-plan-2026-09-30/`;
the canonical task files in this directory supersede those drafts.

| Finding | Resolution in the canonical plan |
| --- | --- |
| Reference refresh could depend on the Can runner it must first compile | P25 bounded independent T, P05 seed acceptance, provisional code tasks, P26 core promotion, then P23 integrated core acceptance |
| Initial owner or strict host capability assumed to exist | P07–P14 implement and independently accept N; P13 implements/qualifies host enforcement or records the profile blocked |
| Native service ownership only named as a prerequisite | P29 implements generic external grants/recovery; K08/K16/K22/K24/K26/K27 own concrete N adapters |
| Base acceptance could accidentally require all browser/DB behavior | P23 live scope is process/files/report; offline assertion vectors do not credit unrun live capabilities |
| Structured diagnostics omitted producer files | P20 owns loader/actual-read instrumentation and diagnostics, not only JSON rendering |
| Shared protocol, receipt, emitter and staging paths overlapped | Narrowed subpaths and explicit sequential/integrator handoffs |
| Typed Can APIs were left until qualification | I01–I17 own complete Can helper packages and examples as well as catalogue/native bindings |
| New Can helpers could inherit an old suite receipt | I promotions and Q/migration case snapshots require separate current S/A acceptance and closure hashes |
| Browser service death could also kill cleanup and observation | K08/K16 own outside-service N cleanup; K07 owns an external observer; QB0 requires continuity and conservative invalidation on loss |
| Descriptor lease was only a fixture description | P10 explicitly owns real kernel fd 4 generation lifetime; QF2 supplies ordinary Can scenario acceptance |
| D4 server quiescence lacked an external implementation owner | K26 owns N `db_settlement.go`, retained namespace authority and server settlement/fence facts |
| Full DB-dependent browser gate could delay unrelated browser checks | QB1base accepts only shared route/input mechanics; full QB1 remains a distinct durable replay gate after QD1 |
| D4 unnecessarily waited for D2's eight-actor verdict | QD4 uses accepted transaction mechanisms and QD1 observation, without depending on the unrelated burst verdict |
| Group-level gates delayed simple rows and left a removed K12 ID | Per-row prerequisites now govern progress; K12 was replaced by explicit mechanism and qualification nodes |
| Shared historical files and delegates could be retired twice or too early | 292 rows have one owner; 30 delegated suites have one body owner and linked parents; only integrator retires shared files after all relevant rows/callers resolve |

The foundation reviewer’s final pass found no additional blockers in the combined
task graph. Capability findings above are incorporated. Source/row/delegate checks
are static evidence only: all 14 integrated feasibility gates remain unrun.

During implementation, any new architecture-changing finding must reopen its
specific design decision rather than weakening the acceptance contract. A missing
host/service or a failed negative control blocks only the affected qualification
and dependent coverage; independent ready source work continues.

## 2026-10-01 audit: validator scope conflict

`python3 tools/check-implementation.py` passes the existing 143-task structure.
`python3 validate_plan.py` fails at P01 because line 28 requires every task to be
`planned`. Its final output also hardcodes a never-started plan and regenerates
`tasks.json`; it is a frozen initial-planning validator, as the implementation
validator's own module documentation explicitly states. Resetting implementation
statuses to pass this check would falsify the ledger.

Concrete proposed correction, if the user permits this narrow documentation-tool
exception to the planning-only scope: replace the all-planned assertion with
validation of the four defined statuses, retain every structural/hash/coverage
check, validate source/master equality without overwriting the master, and write
current status counts plus audit-round execution facts to `validation.json`.
Require recorded complete-task evidence and reachable Git commits touching owned
paths; reject a complete task with unfinished prerequisites. Frozen
`planning-baseline.json` stays unchanged. No runtime/test/harness tooling changes
are involved. Until authorized, the Python tool remains untouched and this
failure remains explicit.

## Current ledger audit and corrections

The audit input is HEAD `37e1e06d8b52567a8e6dd8ce9d68bbe93e441559`.
There are **143 unique tasks**, partitioned into P=30, K=26, I=17, Q=19,
M=45 and Z=6. `tasks.json` equals the concatenation of `foundation.json`,
`capabilities.json`, `integration.json` and `migration-tasks.json`. The category
views are not extra tasks: adding the capability/migration counts to the master
would double-count 90 identities. Slice receipts also add zero master tasks.

Before audit: 50 complete, 1 active, 92 planned. After audit: **25 complete,
4 active, 93 planned, 21 blocked**. No implementation progress was made in this
round. Twenty-one of the revoked completions are now blocked; P12, K05, M32 and M33 retain
active partial work. P28 returns to planned. Detailed per-task evidence, exact
expanded/recovered commits and owned-path matches are in the
[143-task status audit](evidence/status-audit-input.json).

| Error found | Correction and remaining acceptance |
| --- | --- |
| P12 omitted the lifecycle disk admission floor | Restore the 10 GiB available-disk floor and per-allocation recheck, finite capability caps and refusal controls in the card/checklist. Mark active: current admission code/receipts do not establish that guard. Existing gate/deadline work is retained. |
| P13 counted declared peaks/free disk as strict enforcement | Mark blocked and invalidate use of the historical host receipt for current promotion. Its own limitations admit memory has no kernel backstop and tmp bypass is bounded by available disk rather than the envelope. Preserve both required envelopes; no sampled or best-effort substitute. |
| P14 accepted the wrong scope | Mark blocked: P13 is unqualified and its N artifact is an ACK test helper, while ownership work runs in the parent. A separately built real N service with exact host acceptance and independent T controls remains necessary. |
| K05/K06 credited modeled calls without C-subject evidence | K05 is active: implement the actual generated-C invocation witness, artifact binding and bounded bypass controls. K06 is blocked by K05 and still lacks its required bounded raw/C-subject controls. Preserve the token/occurrence model checks as partial evidence. |
| K07 called an in-memory model the actual independent observer | Preserve local controls; restore the actual host observation channel, external ownership, interval-through-disposal and durable correction publication as unfinished K07 work. |
| K08, K16, K24 and K26 credited models as promised native adapters | Preserve simulated controls, but keep native child discovery/containment/reclaim, independent remote close/confirmation, engine settlement, and server quiescence/fence adapters unfinished under their original owners. I/Q integration and qualification cannot silently inherit missing K implementation. |
| Completed dependents relied on invalidated predecessors | Block P29, K09, K10, K11, K13, K14, K15, K20, K21, K22, K23, K25 and K27. Their local evidence is retained; restoring a predecessor permits review/revalidation, not automatic promotion. K07/K08/K16/K24/K26 are also blocked by prerequisites in addition to their own missing work. |
| M32/M33 complete despite retained ports lacking qualification | Mark active. Fifteen M32 and eight M33 ports still require P23 plus actual-facet capability gates, reviewed S/A closure, R/N/C identities, positive/defect/missing-evidence controls and environment/cleanup receipts. Their historical relevance work is preserved. M34/M36 remain complete only for reviewed historical/static dispositions. |
| P28 stale active assignment | Reset planned: no task receipt or authored owner/transport/emitter binding artifacts; old catalogue files are not implementation evidence. The historical Muse checkpoint no longer dispatches workers. |
| P20 commit placeholder; 18 minimal receipts lack SHAs | Recover exact reachable Git commits and verify owned paths. Normalize abbreviated references without inventing commands, deadlines or new outcomes. P14's supporting P10 process fix is recorded as an integration handoff. |
| Q cards lose details when summarizing full gates | Bind all 14 integrated Q cards to the exact normative gate section and shared execution contract. For example QB4 still needs Bun and browser legs, QF1 fresh fd3 snapshots and separate Go CLI/direct-entry environment behavior, and QD4 the pool budget leg and exact prestart error. |
| Z01/Z05 omit campaign completeness controls | Restore the append-only, preregistered campaign/job/attempt index, failed/blocked/interrupted entries, compatible identities and rejection of cherry-picked reports or stitched incomplete attempts. |
| Checklists/top-level state still say not started | Synchronize task states, complete acceptance text and receipt links in all four checklists and source views; update coverage progress annotations, progress template and resume pointer. Preserve original coverage obligations and retirement inventory. |
| Reference proposal still describes P04/P05 as pending | Mark the original input inventory as a historical selection snapshot and link the recorded P04/P05 bounded seed receipts. Keep R-core, actual N and suite acceptance unfinished; this does not revalidate a retained executable today. |
| Initial validator rejects all real implementation statuses | Record the exact failure and narrow proposed correction above; never falsify task states to satisfy its obsolete assertion. |

All 50 previously complete tasks had evidence files. Every parsable recorded SHA
resolved; the sole placeholder was P20. Eighteen minimal receipts needed recovered
commit links (K07–K11, K13–K16, K20–K27 and M36). The 25 retained complete tasks
now have fully resolved reachable primary SHAs in the audit, touching their owned paths, historical
acceptance/control records, and complete prerequisites. These are bounded/static
acceptances, not a newly verified full runtime. P02/P15/P16/P17 explicitly permit
provisional source acceptance pending P26/P23; P05 is only the bounded R-seed
bootstrap scope, never R-core or full suite authority. No other planned task has
finished work sufficient to promote it. No original recorded check was rerun.

The 867 unique predecessor/task edges resolve, are acyclic and have no self-edge
or reverse reachability conflict. Lists and task identities are unchanged. The
[lane plan](lane-plan.md) classifies every phase-aware edge with its reason or
alternate dependency witness; advisory direct edges remain recorded and impose
the same order. Removing a transitively implied edge would not save wall time.
All 292 coverage rows agree with their embedded migration gates, and all 224
tracked source files, 30 delegates and 16 external callers remain accounted for.
No facet, variant, environment, delegate or retirement gate was removed.

Gate fidelity was checked against the main plan, completion/capability/lifecycle
contracts and the complete integrated register. Lifecycle requirements restored
above are at `native-can-test-lifecycle-2026-09-30.md:322` (admission) and
`:480` (campaign integrity). Row gates remain minima: the implementation's actual
facets may add gates; a convenient group summary never waives those obligations.

## Dispatch, review size and limits

The [lane plan](lane-plan.md) and [machine-readable queues](lane-plan.json) give
ordered work, owned paths, prerequisites, evidence, sub-deliveries, handoffs,
cost assumptions and critical-path calculations. Each ready parent can produce
small independently reviewed source/control/evidence increments; acceptance still
joins every required increment and gate. Ready workers steal only unreserved,
dependency-eligible work. The integrator alone edits shared compiler/catalogue
files, formats globally, commits, activates callers and retires harnesses.

Three fresh [Jev consultations](evidence/audit-2026-10-01/jev/decision.json)
favored this parent-preserving decomposition (probabilities 1.00, 0.97 and 0.98).
All context, instruction and option prose was rewritten and checked for semantic
equivalence before sending. Requests and responses are saved; no choices
disagreed. Agreement is advice, not proof or guaranteed removal of bias.

Estimates are planning assumptions, not performance measurements or an ETA.
The dependency path is only a lower bound: integrator capacity, source-writing
time, one phased heavy-execution slot, unavailable strict host enforcement and
deferred retained metrics can dominate. The plan cannot guarantee a busy worker
when no eligible disjoint source work remains, and it does not invent work or
relax gates to create apparent utilization.

Only this plan directory is changed. The pre-existing untracked
`muse-replan-prompt.txt` is preserved and excluded from the commit. No temporary
directory, checkout, build, browser or service was allocated by this audit;
there are no owned temporary artifacts to reclaim. Source/harness files and the
frozen planning baseline are untouched. No push is authorized.

## Final verification record

`python3 tools/check-implementation.py` passes with 143 unique IDs, corrected
25/4/93/21 complete/active/planned/blocked totals, all 292 rows, unchanged hashes
for 224 tracked files and 30 delegated suites, and valid local links.
`python3 validate_plan.py` still exits 1 at its frozen P01 all-planned assertion;
the Python tool is unchanged. The permission question for the concrete narrow
validator repair remains pending, so **both validators passing is not claimed**.
[validation.json](validation.json) records both outcomes and the scope limitation.

An independent read-only final reviewer found the additional K05/K06 overcredit;
those corrections are included. That reviewer independently confirmed the
phase-aware alternate-edge witnesses, residual longest-path length and preserved
gate/worker/resource constraints. Historical executable checks were not rerun.
