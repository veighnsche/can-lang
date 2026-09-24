# Dispositions for the next Can upgrade

24 September 2026 · input: [post-upgrade reconciliation](post-upgrade-reconciliation-2026-09-24.md) at `fbd2a56`

**Selected scope:** repair the two conformance bugs, finish the previously
accepted shared action and browser delivery contracts, close the accepted UI
behavior gaps, and add safe public generic-to-generic composition. The existing
invoice server/grid is the cross-target acceptance workload. The remaining
findings below have an explicit retained boundary or reopening condition.
This is the scope decision, not a claim that these fixes have been coded.
The subsequent [selected behavior](post-upgrade-selected-behavior-2026-09-24.md)
specifies their source/runtime contracts; the [implementation plan](post-upgrade-implementation-plan-2026-09-24.md)
and [task lanes](post-upgrade-implementation-tasks-2026-09-24.md) now sequence
the work. The mechanism questions below record what remained open at scope selection.

**Fix** means the next upgrade must deliver an observable outcome. **Retain**
keeps a selected behavior or a documentation correction already completed.
**Defer** excludes the proposed change from this upgrade; the row says what
evidence would reopen it. A retained current rule is named even where an
extension is deferred. The source and earlier-decision links for every ID are
in the [reconciliation](post-upgrade-reconciliation-2026-09-24.md). The
39 primary dispositions are 11 Fix, 9 Retain and 19 Defer. The
[core](evidence/2026-09-24/post-upgrade-dispositions/core-proposal.md),
[platform](evidence/2026-09-24/post-upgrade-dispositions/platform-proposal.md)
and [native](evidence/2026-09-24/post-upgrade-dispositions/native-proposal.md)
reviews supply detailed acceptance and reopening cases.

## Fix in this upgrade

| ID | Decision and reason | Completion evidence required |
| --- | --- | --- |
| **B01** browser audit mistakes quoted text for host access | **Fix.** The current substring scan contradicts browser capability admission. | Harmless `"Bun."` and `"node: introduction"` compile, while actual forbidden imports/operations and reachable runtime dependencies reject with useful location evidence. Validate through the supported browser pipeline. |
| **B02** Unicode empty regex advance | **Fix.** The runtime's one-code-unit increment is wrong under `u`/`v`. | `text::matches` over `😀x` with empty Unicode pattern reports starts `[0,2,3]`, matching native behavior within the limit; non-Unicode, nonempty, capture and cap cases remain correct. |
| **U01** shared action declaration | **Fix.** A handler-free shared wire declaration was already selected; current `handles` requirement makes the browser duplicate actions with stubs. | Server and browser import the same action symbols from one locked package. Method/path/capture/body/case edits change or diagnose every linked use. No placeholder client handler or mirrored wire contract remains. |
| **U02** request and services at server mount | **Fix.** The accepted mount was request-aware; today's handler accepts only captures/body. | One startup pool is explicitly bound, the server derives actor identity from each request, and the checked GET/save handlers recheck authorization at the protected operation. No client data becomes an implicit permission. |
| **U03** invoice route and edit propagation | **Fix.** The declared captured GET still needs a test rewrite to reach the manually mounted query route. | The declared path reaches the live authenticated handler directly. JSON and form actions use their checked bindings. Changing a route, capture, wire field, body mode or result leaf rebuilds both targets from one contract or gives a diagnostic; real HTTP statuses, replay and denial remain correct. |
| **U04** supported browser delivery | **Fix.** A Can browser target exists, but the tested JavaScript requires grid-only Node semantic shims. | An empty app and the invoice grid boot through a maintained build/bundle path without importing the test bundler. The verified server build publishes the selected content-addressed same-origin asset under its CSP; inspect the emitted module graph, source maps and entire reachable runtime closure for server imports and secret bytes. Exercise standard failures, diagnostics, equality, async callbacks, disposal, admitted coordination and wire-codec parity in named browsers. A bundler is allowed; its semantics must be owned and tested. |
| **U05** cancelable admitted events | **Fix.** The earlier browser contract selected native cancellation, which the current event API cannot express. | An admitted keyboard/submit event can request native `preventDefault` while uncanceled events retain normal behavior; disposed views cannot cancel stale events. Specify the bounded cancellation channel without leaking raw mutable events. |
| **S01** visible HTML 503 | **Fix.** The selected invoice contract requires stale feedback to be replaced, while current HTMX policy does not swap 503. | A real 503 remains status 503 and visibly replaces the specified target feedback. The other finite cases keep their selected status and display policy. |
| **S02** observable browser handler faults | **Fix.** The accepted event contract requires reporting; current callback settlement discards failure. | A deliberately failing admitted handler yields one sanitized observable diagnostic and does not silently claim success; listener/view disposal still works. |
| **S03** grid focus, notice and pending-save state | **Fix.** The earlier grid packet selected focus restoration and visible save/error state; Gate 5 records three exceptions. | Add/move/reorder focuses an attached intended control, blocked-save notice survives rerender and is announced, pending save is visibly represented until settlement, and late replies/disposal cannot clobber newer edits. No exact notice wording or one UI framework is required. |
| **P03** public generic helper composition | **Fix as a bounded new capability.** DI-04 declaration checking works, but extracting a representation-independent public helper makes a valid generic fail. | A public identity/helper chain with opaque type arguments checks across packages and emits only valid concrete calls. A callee edit adding unsupported arithmetic fails at its declaration. Private templates cannot borrow a false parametric proof; expanding polymorphic recursion is rejected, and mutually recursive callees cannot use proof before their cycle is validated. Keep explicit callable/dictionary inputs for operations truly needed on the type parameter. |

The selected action shape is the earlier handler-free declaration plus
request-aware server mounting. The checker/emitter and service-binding details
still need a complete design packet before coding. For U04, the design packet
must compare a browser-specific runtime profile with a fully maintained shim
layer and choose one by reachable dependency and semantic evidence. P03 must
first establish how validated parametric callee contracts substitute symbolic
arguments without treating private templates as validated or emitting opaque
specializations. If a gate contradicts the chosen contract, return to design;
an implementation worker should not relax it silently.

## Retain or defer the other findings

Each row has one primary disposition. “Defer” applies to the proposed change;
the existing rule stated in that row remains in force.

| ID | Disposition and reason | Reopen condition or retained boundary |
| --- | --- | --- |
| **P01** bound-callable client projection | **Defer.** The previously selected handler-free declaration/mount split supplies U01/U02; this alternative would revise it. | Reopen only if a two-target prototype proves a strictly server-free client projection and a concrete advantage over the selected split. |
| **P02** runtime profile versus maintained shims | **Defer only a preference for either mechanism now; resolve the choice within the U04 Fix design packet.** Neither module layout is an earlier user choice. | Choose before implementation planning using full dependency inventory and the same empty-app/grid/failure/async acceptance. Grid-only synchronous async-context assumptions fail U04. |
| **P04** authored finite error-set parameters | **Defer.** DI-03 already deferred a new generic kind; catalogue callback precision alone does not establish an authored syntax benefit. | Compare fixed bounds and result data on two unrelated callback domains and a wrapper around a wrapper, including public signature, error identity and agent repair cost. Preserve explicit current `emits`. |
| **P05** stack-safe dynamic iteration | **Defer.** Deep `relay` overflow is real, but no tail-call promise exists and the native-loop contract is unresolved. | Compare self-tail lowering and immutable-state iteration on pagination/state-machine work, including exactly-once arguments, failures, fixtures and diagnostics. The U06 release-documentation gate owns the current recursion boundary: there is no portable numeric stack threshold. |
| **P06** early HTTP response with live race losers | **Defer changed ownership.** Retain owner drainage, leases and current shutdown semantics. | Measure a real HTTP hedge; compare operation-local deadlines, cooperative native abort and supervised loser ownership with response time, lease preservation, late-failure diagnostics, client disconnect, cleanup, effects and shutdown evidence. |
| **P07** nonempty race helper or new empty completion | **Defer an API/semantics change.** Retain `Promise.any`/`Promise.race` modes, including pending empty first-completion race. | Reopen with an actual dynamic-race use case; a nonempty guard is separate from changing empty completion to a typed failure. |
| **P08** richer input event fields | **Defer.** Checked state, modifiers, files and IME details were not selected with the first event snapshot; U05 is the accepted minimum. | Require an all-Can checkbox/IME/file workflow and a bounded immutable native projection with disposal and browser tests for each added field. |
| **P09** reload-persistent drafts | **Defer.** The selected grid promises an in-view draft, not reload survival or automatic queued writes. | Reopen with an explicit product need plus storage versioning, privacy and crash/uncertain-save reconciliation tests. |
| **P10** browser history and WebSocket | **Defer.** Server WebSocket does not confer a browser API, and neither capability is needed for U04. | Give each a separate navigation or live-update workflow with origin, authentication, reconnection and lifetime rules. |
| **P11** reviewed host adapters versus catalogue additions | **Defer a generic extension architecture.** Retain the controlled distribution catalogue and capability gate. | Compare two concrete missing native operations through both designs, including errors, ownership, assertions and transitive browser admission. No arbitrary JS/SDK escape is selected. |
| **P12** field/keyed-row reuse or components | **Defer supported component syntax and library API.** Current named functions/typed nodes compose; S03 must be repaired first. | Prototype a typed field and keyed editor on the existing grid, then measure one-field edits, focus, reorder, late replies and disposal. A useful library need not imply syntax. |
| **P13** explicit `near` binding | **Defer syntax.** Retain immutable name-based captures and explicit context records. | Run a registered same-type rename/shadow repair comparison against explicit site bindings; show context records or diagnostics are insufficient before changing grammar. |
| **P14** advisory local-elision rule | **Defer demotion.** Retain the narrow current validity predicate provisionally. | Compare agent edits using a meaningful domain local against an accidental single-use alias, preserving current type/error behavior. |
| **P15** Boolean arm ordering | **Retain.** `false` before `true` for a single Boolean scrutinee is an explicit earlier user decision; the review's taste judgment supplies no new safety evidence. | Keep its existing scoped checks. A reversal requires a distinct decision and must not silently alter other match modes. |
| **P16** multiline delimiters | **Defer.** Retain physical-line grammar and the shipped formatter. | Reopen if held-out agent creation/refactor/repair under formatter support shows material failure, with clear nested delimiter, comment and indentation behavior. |
| **P17** cleanup helper or `try/finally` lowering | **Defer.** Retain owner fallback and explicit cleanup; T18 already repaired the stream example. | Compare a named helper with a narrow native-lowered form on the same stream pump, including original-versus-close failure precedence and exactly-once close. |
| **P18** bulk Map/Set APIs | **Defer.** Retain native immutable copy-on-point-update semantics. | Measure representative unique and duplicate-heavy workloads, then trial one native build/immutable publication with defined collisions, order, callback errors and ownership. |
| **P19** schema-aware SQL build tooling | **Defer.** Retain statement/parameter/cardinality checking and runtime row validation; no versioned schema proof was selected. | Reopen with a versioned migration/deployment provenance and executed drift case. Do not describe current descriptors as schema-verified. |
| **P20** mutation `RETURNING` | **Defer.** Retain transaction plus SELECT; dialect support alone does not settle row/cardinality/error rules. | Demonstrate a real invoice/webhook post-write read or race cost, then specify dialect and affected-row semantics. |
| **P21** owner-selected equality | **Defer.** Retain eligible representation-based structural equality and the existing owner invariant boundary. | A two-version owner package must show stable public equivalence with changed hidden fields and define who selects the comparison. |
| **O01** AI batch atomicity/dependency | **Retain.** Validate whole batch before ordered handlers; authored effects are not rolled back, and dependent questions use sequential calls. | Transactional effects would need a separate workflow and decision. |
| **O02** dynamic Choice cardinality | **Retain.** Typed invalid-question outcome applies outside 2–255 options before transport. | Empty/singleton fallback remains application policy, not a compiler fix. |
| **O03** native recovery/grouped callables | **Retain** the shipped fetch/judge operation wrappers and named adapters; **defer** provenance-aware LLM recovery and DI-17 grouped-state syntax. | Reopen with same-workflow comparison preserving origin, failures and whole-task agent cost. |
| **O04** outbound origin policy | **Retain** named fixed-origin clients and confined paths; **defer** tenant-selected destinations. | A dynamic client needs destination, credential, redirect and assertion rules, not unrestricted fetch. |
| **O05** webhook carrier/security envelope | **Retain** the documented trusted-carrier sample scope. | Wider deployment or constant-time verification needs a concrete threat model and separate native/policy acceptance; its sample limitation is not a language-expressiveness bug. |
| **B03** stale shipping descriptions | **Retain the documentation fix** completed by the reconciliation. | Next release docs must continue to distinguish current source from historical plans and avoid the old “no HTTP client” claim. |
| **B04** overstated or outdated qualification | **Retain the documentation fix** completed by the reconciliation. | Report compiler TS versus test bundle, emulated Linux final acceptance, SQLite example use and exact browser versions without reusing earlier overbroad verdicts. |
| **U06** truthful current language/target reference | **Retain the completed reconciliation and make it a release gate.** | Each new implementation claim needs a selected contract, actual source/test evidence and explicit limits. This documentation obligation was addressed in the prior step. |

The review's positive foundation findings are **retained**: explicit pattern
bindings, owner record confinement, extensional variants, canonical package
and error identities, scenario links and fixture queues, Can-authored browser
execution, parameterized SQL/transactional examples, native Bun operations,
and supervised assertions/build publication. Their recorded tests prove bounded
behaviors, not every deployment or application result. No replacement of
native operations, owner safety, explicit error contracts or the closed
capability boundary is in scope. `emits []` still does not promise purity or
termination; an opaque assertion `ok` still needs companion observations for
visible effects where the accepted product contract calls for them.

## Evidence, sequencing and decision limits

This disposition considers three independent source proposals and [three fresh
Jev consultations](evidence/2026-09-24/post-upgrade-dispositions/jev/findings.md).
Jev favored the selected action split but also favored leaving browser delivery
at the test grid. That latter classification conflicts with an earlier accepted
contract and this upgrade's stated priority, so U04 remains Fix. Jev preferred
an experiment before generic composition; the soundness experiment is mandatory
inside the P03 Fix before its compiler design is accepted. The classifier
split on optional error, host and authoring choices; none is promoted by vote.
The saved requests, response distributions, wording audit and disagreement
analysis remain in the evidence directory.

B01/B02 can proceed independently. U01/U02 need one declaration/mount design;
U03 then proves linked real routes. U04 needs a complete runtime closure audit
and maintained delivery design before final browser qualification. U05/S02
share the event boundary, S03 exercises it in the grid, and S01 is an
independent HTML-response slice. P03 is a separate core lane with a checker
soundness gate. These are dependency facts for the later implementation plan,
not task assignments or estimates. No full release, live database/provider
flow, HTTP hedge, Playwright matrix, Linux host run or agent comparison was
newly executed for this disposition.

The subsequent [selected-behavior packet](post-upgrade-selected-behavior-2026-09-24.md)
specifies source forms and server binding/lifetime semantics for U01–U03,
chooses the U04 runtime delivery path, defines U05/S02 event behavior and
S01/S03 UI acceptance, and sets P03's validated symbolic-call boundary.
Ordered implementation tasks follow that packet. Deferred mechanisms remain
outside those tasks unless a recorded reopening decision changes this ledger.
