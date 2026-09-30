# Implementation-plan review

Status: planning review only. No implementation, live qualification or replacement
coverage is accepted. See the [main plan](../native-can-tests-plan-2026-09-30.md)
and [static validation](validation.json).

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
