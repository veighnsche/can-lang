# Native Can tests: shared capability contracts

Status: **contracts specified; implementation and qualification pending**.
The [post-probe design review](native-can-test-design-review-2026-09-30.md)
incorporates the executed mechanics checks. Its acceptance limits apply to every
operation below; no integrated capability is qualified by those checks.
This follows the [execution architecture](native-can-test-architecture-2026-09-30.md), [authoring design](native-can-test-authoring-2026-09-30.md) and [coverage ledger](native-can-tests-migration-ledger-2026-09-30.md). It defines observable behavior and authority, not implemented APIs, passing tests or permission to begin builds and measurements.

Operation names below are **logical interface notation**, not new Can syntax or final package names. They clarify the earlier examples' proposed helpers. Bind them through ordinary typed Can packages and the native catalogue, using equivalent native JavaScript/Bun/browser/driver operations with the adapters needed for Can contracts and immutability. No language primitive per test category, embedded scenario language or authored host harness is required.

The subsequent [lifecycle contract](native-can-test-lifecycle-2026-09-30.md) resolves failure transitions, coverage/exit semantics, cancellation, admission, initial resource ceilings and within-run build reuse across these operations.

The subsequent [hard-case source challenge](native-can-test-hard-cases-2026-09-30.md) corrects assumptions about browser action concurrency, synchronous callbacks and R/C linking, launcher environments, and database settlement. Its [bounded experiment register](preparation/native-can-test-hard-cases-2026-09-30/experiments.md) records unrun feasibility gates; the additions below remain proposed bindings.

## 1. Responsibilities

| Concern | Native mechanics | Ordinary Can decisions |
| --- | --- | --- |
| Workspace | Owned allocation, bounded I/O, identity, deletion and recovery facts | Fixture contents, mutations, expected outputs and retention requests |
| Process | Spawn, pipes, descriptors, signals, exit facts, deadlines and cleanup | Commands, readiness, request/signal ordering, retries and expected outcomes |
| Browser | Native actions, DOM/network/event facts, interception tokens and context cleanup | Engines, interactions, deliberate faults, response edits and comparisons |
| Independent observation | Raw values, identity, exact representations, counters and external driver facts | Probe composition, expected answers and the independence claim supported |
| Compiler diagnostics | General structured product CLI over identified inputs | Valid/invalid controls, expected phase/code/span and verdict |
| Run | External ownership, bounded transport and completion/cleanup receipts | Registry, selection, scheduling, required checks, aggregation and report policy |

The qualified reference toolchain **R** builds the suite/controller/judges. Candidate **C** produces subjects only. Go supervisor **N**, and resource services it owns, retain authority outside killable Can workers. Candidate code never enters the trusted controller/worker realm. Record observer, candidate compiler/runtime, subject artifact and input revision separately. Independence is from a named adapter/producer, not necessarily the entire stack.

For AI authors, prefer explicit argument/result records, typed handles, precise error kinds and one canonical operation per effect. Do not silently choose the first selector match, inherit environments, retry, assume readiness or fall back to another engine. Arbitrary ordinary Can functions compose operations under normal target/effect rules, with attached `asserts` and explicit boundary fixtures. No assertion exemption or opaque-input elision is introduced. The [agent-authoring audit](preparation/native-can-test-authoring-2026-09-30/agent-authorship/audit.md) still governs canonical package examples and their diagnostics.

## 2. Shared authority, outcomes and evidence

### Ownership

N creates run and case/variant/attempt scopes and grants them to specific worker connections. Can chooses the case plan; N checks it against the run envelope. Handles bind owner, kind and generation. Printed IDs, PIDs, paths and serialized records are not resource authority. Wrong-owner, stale-generation and closed-handle use are explicit errors. Read-only artifacts can be shared through explicit leases; mutable workspace, process, browser and value handles cannot be transferred as ordinary data between workers. Subject stdout, CLI JSON, browser messages and probe events are bounded untrusted data, never supervisor/control-channel commands.

Resources move through `reserved → opening → live → closing → closed`. The owner registers intent **before the first externally visible effect**, records acquired identity before exposing the handle, and retains partial acquisitions until reconciled. Close stops admission, joins admitted work under that operation's rules, and returns a receipt. Repeated close joins the same disposal. Recovery can advance an incomplete disposal but cannot erase its earlier failure.

### Request/reply protocol

Every request carries schema version, run/case/attempt, unique operation ID, owner grant, typed arguments and finite deadline within the outer envelope. Mutations durably record operation ID and argument digest before dispatch. Repeating identical ID/arguments joins or returns that operation; changed arguments under the same ID are a protocol error. After owner/service restart, a dispatch record without a terminal fact returns `indeterminate` until reconciled; never re-dispatch it, even if it might not have reached the target. This gives at-most-once dispatch within a recoverable scope, **not exactly-once external effects**. Durable pre-dispatch recording is an implementation gate. After a lost acknowledgment, query/reconcile; do not replay a write, spawn, signal, route resolution or commit on the assumption that it failed.

| Result | Meaning |
| --- | --- |
| `completed(facts)` | The operation's documented completion condition was observed |
| `rejected(kind)` | Validation/admission failed before the requested effect; any admission reservation remains accounted for |
| `failed(kind, partial)` | Mechanical failure with explicit partial bytes/effects and handle state |
| `deadline(partial)` | This action/wait did not complete in time; resource lifetime remains explicit |
| `indeterminate(partial)` | An effect may have occurred but its outcome cannot currently be established |

Mechanical kinds cover invalid request, unsupported capability, wrong owner, stale/closed handle, not found, permission, changed input, kind/collision, resource limit, native I/O, transport/protocol failure and unresolved cleanup. Preserve underlying causes and operation identity. Can may normalize these into the proposed `spec::failed`/`spec::broken` policy without losing evidence. Subject exit 1, HTTP 500, thrown values and compiler rejection are **data**, not automatically operation/test failures.

Every collection declares completeness, counts, truncation and gaps. Never turn unavailable evidence into false, zero or an empty successful result. Retained output is a bounded prefix/slice with offsets; a digest of discarded bytes does not recover content. Redact retained public evidence after authorized comparisons; do not journal credential bytes through generic argument/result dumps.

Events include source/resource identity, operation ID when known, local sequence, clock domain and monotonic timestamp. Stream chunks include stream and byte offset. Per-channel order is meaningful; broker arrival order or different clocks do not prove cross-process causality. Can uses arrival/release/settlement events. Old cursors report gaps, never silently resume with a shortened apparently complete sequence.

### Bounds and cleanup

Profiles supply positive finite run/case/operation/cleanup deadlines and limits for processes, handles, pending actions, workspace bytes/entries/depth, input/output/events and retained artifacts. N enforces the outer envelope independently of Can progress. The [lifecycle contract](native-can-test-lifecycle-2026-09-30.md#5-concurrency-and-resource-admission) now specifies one case, one verification worker and one build producer with phase ordering and initial numeric ceilings. Admission reports supported guarantees and the qualified host profile; a required unsupported guarantee makes the capability unavailable. Complete type-specific limits and qualify enforcement before live execution; do not scatter defaults through adapters.

`clock.after(scope, duration)` creates a deadline in the service's clock domain; `clock.remaining(deadline)` and `clock.wait_until(deadline)` return timing facts. Can owns cadence/readiness. An operation deadline does not implicitly dispose a resource unless specified; a hard resource-lifetime deadline starts N's bounded cleanup.

Cleanup runs on close, worker/controller death and hard limits. Stop or drain admitted effects before deleting their workspace/namespace. Receipts list released resources, forced actions, remaining resources and failures. Release a lease/quota charge only after its corresponding resource is confirmed released; unresolved resources stay owned and charged in recovery records, preventing admission that assumes freed capacity. Preserve subject outcomes and Can verdicts separately. Consumers require both the Can report and N's matching completion/cleanup receipt; missing required evidence or unknown cleanup prevents clean success. After N dies, recovery needs durable ownership, path/process identity and inactivity proof. Age alone is insufficient.

**Host enforcement is still an implementation gate.** Path normalization does not confine a child; process groups do not contain detached descendants; disk polling is not a strict quota; input hashes do not prevent temporary mutation followed by restoration. Claim only demonstrated guarantees. Never clean foreign work, shared caches, browser servers or Docker volumes to satisfy task budgets.

## 3. Owned temporary workspaces

A workspace is a registered private allocation containing selected inputs, scratch/output and compact evidence. No default full repository/dependency copy or persistent private build cache. Its path projection for child argv/cwd is descriptive; deletion still requires authority.

| Operation | Inputs | Native effect and result |
| --- | --- | --- |
| `work.reserve` | Scope, purpose, budgets, retention/expiry policy | Register intent and reserve unpredictable root identity; return reservation/lease. No fixture code runs. Created allocations remain owned on failure. |
| `work.materialize` | Reservation, selected byte entries or authorized source manifest/digests, destination, create/replace rule | Import only selected entries; enforce limits before retention; verify bytes; return revision and per-entry outcomes. Multi-file partial publication is explicit. |
| `work.read` | Handle, relative path, offset, byte limit, optional expected revision | Observed bytes, kind, length/digest, revision and stability strength; distinguish absent, escape, wrong kind, detected change, truncation and I/O failure. |
| `work.stat` / `work.list` | Handle, relative path, bounded page/cursor | No-follow node identity/kind/metadata or revision-bound page with stability strength. Capability-mediated mutation invalidates pagination; unknown external mutation cannot produce a claimed stable complete inventory. |
| `work.write` | Handle, relative path, bytes, create/replace/compare-revision mode | Stage and publish one file atomically where qualified, charging staging bytes too; return prior/new revision and digest. Failure reports visible/uncertain state. No implied multi-file transaction. |
| `work.make_node` | Handle, relative path, directory/symlink/permission description | Apply exactly that fixture operation and return native metadata/error. Symlink targets are literal data; traversal remains forbidden. No writable links to shared sources. |
| `work.seal` | Input root(s) or selected paths/revision, declared full-tree or partial scope | Stop API mutation within the scope; return sorted inventory of files, directories, links and relevant absence claims, with content/metadata identity and protection strength. Full-tree seals cover additions/removals as well as byte changes. Scratch/output outside those roots remain writable. New input mutation requires a fresh revision. |
| `work.dispose` | Handle, deadline | Stop admission, dispose dependent leases/resources, remove identity-verified owned entries, report remaining paths/errors. Repeats join disposal. |
| `work.recover` | N's run record and inactivity proof | Reconcile registered allocations/leases; dispose abandoned owned entries or report active, foreign/mismatched or unresolved identities. Cannot target arbitrary caller paths. |

Reject NUL, absolute paths and `..`. Use directory-anchored, race-resistant no-escape operations, not string-prefix checks followed by unrestricted I/O. Inspect symlinks without following them. Imports reject links by default; any safe link-copy mode must have its own explicit rule. Fixture permissions cannot affect another workspace or installation.

Read/list/seal receipts distinguish `scoped_revision` (capability-mediated mutations accounted for), `qualified_snapshot` (the named host snapshot guarantee), `changed_observed` and `unknown`. Only a qualified snapshot may claim stability against out-of-band writes; even an open file handle alone does not freeze its bytes. Expected stable revisions reject when the required strength cannot be provided. Hashing received bytes describes those bytes, not an undetectable history of the original file. Unknown stability cannot satisfy a complete directory inventory or immutable-input requirement.

A logical seal prevents API writes only. A partial seal identifies selected paths and cannot claim the complete compiler project namespace: adding a file can change resolution without changing existing file hashes. Compiler fixtures therefore declare complete input/dependency roots, inventories and meaningful absent lookups. A compiler can hash the exact bytes it reads without proving that an external process lacked other filesystem access. Strong immutable shared-input or candidate filesystem-confinement claims require qualified host enforcement. Before/after hashes alone cannot prove them. Record strength and refuse admission if a case requires more. Can chooses content, file selection, mutations, expected bytes and reuse. Build common artifacts once per run, grant read leases and retain compact manifests/digests. Heavy retention needs explicit authorization, expiry and automatic reclaim. Failed compiler output remains owned even when no artifact handle was returned.

To create the invalid control after sealing a valid one, Can makes a new reservation, materializes the selected input manifest through a read lease, applies its exact mutation while the new inputs are writable, then seals the new revision. `work.write` never reopens a sealed input. These are bounded fixture copies, not full repository/build clones or a new implicit transaction primitive.

## 4. Managed processes

Reuse native mechanics behind existing `process::run`; extend them through reviewed bindings for outside-worker ownership and streaming. The one-shot API is a facade over shared mechanics, not proof that managed children already exist.

| Operation | Inputs | Native effect and result |
| --- | --- | --- |
| `proc.spawn` | Executable identity, literal argv, cwd lease, explicit startup environment, descriptor map, I/O/lifetime limits, containment requirements | Reserve quota and external launch ownership before releasing exec; return child handle/start identity/launch facts or classified failure. No health/readiness wait. |
| `proc.write_input` | Child, writable descriptor, bytes, deadline | Bounded backpressure; bytes accepted into pipe plus partial/uncertain delivery. Acceptance does not mean child consumption. Preserve binary NUL. |
| `proc.close_input` | Child, descriptor | Deliver EOF once; repeat returns existing outcome. Descriptors are independent. |
| `proc.read_events` | Child, cursor, item/byte cap, wait deadline | stdout/stderr bytes, EOF/read errors, exit and limit/forced-action facts with offsets/cursor. Read deadline does not terminate child. |
| `proc.status` | Child | Point-in-time leader/stream/owned-descendant state, not application progress or future liveness. |
| `proc.signal` | Child, target `leader` or `owned_tree`, signal | OS delivery attempt against verified identities; sent/already-exited/error. No claim that the application handled it. |
| `proc.wait_exit` | Child, deadline | Exit code or terminating signal, or pending/deadline with retained handle. Code and signal are distinct; no success assertion. |
| `proc.collect` | Child, cursor, deadline | Join exit/stream completion; return final bounded stream facts only when EOF/gaps and descendant state are known. Leader exit alone is insufficient. |
| `proc.terminate` | Child, TERM grace and final deadline | Apply declared TERM then KILL escalation; record actions, exit/reap, outstanding descendants/descriptors and errors. Repeats join. |
| `proc.close` | Child, deadline | Stop new input; terminate if necessary under disposal policy; finish capture/reap. Release corresponding leases/quota only on confirmed resource release; unresolved descendants/descriptors remain owned/charged. Return receipt. N uses this after worker death. |

Order state updates/input writes, but a pending wait/read must not hold a lock that blocks input, signal, drain or close. Exit can race every operation. Descendants can retain pipes after leader exit. Record output/event overflow immediately; subsequent drain/discard or termination stays bounded. A blocked pipe must not cause an unlimited wait. Lost required output is incomplete even after exit zero. Forced cleanup cannot satisfy graceful-shutdown expectations.

No implicit shell or environment inheritance. Can chooses an explicit snapshot, validated against the executable's launcher contract. Do not accidentally inherit injection-sensitive startup flags. Hostile environment values may intentionally be subject data without affecting R/N startup. Secrets use scoped channels and restricted evidence metadata.

Distinguish **candidate product-launcher input** from **direct generated-entry startup**. When testing C's Go CLI, preserve its explicitly declared hostile `BUN_OPTIONS`/`NODE_OPTIONS` inputs so C's own child sanitization is exercised; N must not mask the defect by stripping them first. A direct Bun entry instead receives the sanitized startup profile plus application snapshot. Synthetic preload payload files are adversarial input fixtures, not host-authored scenarios or oracles.

The descriptor map declares direction, byte/EOF policy and lifetime: fd 0 stdin, 1/2 captured output, optional fd 3 application credential/environment bytes in the named wire format, and declared output leases such as fd 4. Reject collisions and undeclared inheritance. Never pass N's private channel to C. Distinguish raw fixture bytes from launcher-encoded snapshots; do not double encode.

The current environment import reads fd 3 synchronously to EOF: deliver/close it concurrently with startup, before waiting for readiness. Malformed snapshot/import failure differs from a missing-name lookup. A non-reading child must not leave a blocked writer; record partial delivery and close/reap without turning an expected unused pipe into a false subject failure. Each launch gets fresh descriptor state. A child-held generation lease remains authoritative after launcher exit; wait for actual release before pruning. These mechanics require F1/F2 qualification in the hard-case register.

PM-F1 supports the POSIX-to-Bun mechanics only. The implemented launch binding
must record bytes offered, bytes accepted, EOF/close, writer termination and child
exit independently. An expected unused pipe never implies that the full snapshot
was delivered. A withheld-EOF deadline stays failed even if later cleanup closes
the pipe successfully. Go `ExtraFiles`, candidate CLI sanitization, fresh launch
offsets and inherited generation leases still require their own integrated checks.

Can implements health polling with generic HTTP operations and explicit status/body/retry rules. A lifecycle case holds real work, observes peer arrival, sends a signal, obtains an external shutdown witness, then releases work and examines effects. Spawn, sleep and exit alone do not establish readiness, draining, rollback or exactly one durable effect.

## 5. External browser operations

A native driver, potentially Playwright, supplies mechanics. It contains no test names, expected application text/status, fixed grid/invoice scenario or case-supplied JavaScript evaluation. Reviewed bindings can call browser APIs internally; typed parameters never become executable source.

| Operation | Inputs | Native effect and result |
| --- | --- | --- |
| `browser.open` | Engine/profile and qualified local executable or remote endpoint grant | Acquire owned local process/connection; identify engine/version/UA, driver and capabilities. Unsupported engine is explicit, not skipped automatically. |
| `browser.context` / `browser.page` | Browser/context, network policy and limits | Own a fresh storage context/page before exposure. Arm required event/network boundaries before navigation or executable content. |
| `browser.navigate` / `browser.reload` | Page, URL where needed, named milestone, deadline | Navigation ID, redirects, final observed URL, response or explicit no-response, attained milestone. No implicit application-ready verdict. |
| `browser.cookies` / `browser.offline` | Context, cookie set/clear or offline state | Apply scoped browser state; return native outcome. No login/session/recovery assertion. |
| `browser.nodes` | Page/frame, typed selector, required cardinality, limit | Return 0/1/many or ordered bounded node handles; never silently pick the first ambiguous match. Bind handles to document/frame generation. |
| `browser.capture` | Node(s)/page, explicit projection, bounds | Immutable text/markup, attribute presence/value, native properties, option/file metadata, focus/caret/selection, connected state, child order/counts. Preserve missing/null/empty and native read errors. |
| `browser.input` | Node/page, physical click/key/type/fill/check/select/focus/file action | Perform that mechanism with bounded actionability waits; return dispatch facts/error. Fill, typing and synthetic mutation remain distinct. No fallback. |
| `browser.input_begin` / `browser.input_settle` | Same input action and later pending-action handle/deadline | Start with explicit action identity and await terminal facts separately. Route/event operations remain serviceable while input is pending; no driver lock may require the held route to finish before Can can resolve it. A proven equivalent acknowledgment contract may replace this split. |
| `browser.dom_action` | Node, typed property/attribute write, selection, reset, detach/reattach, DOM click or event | Apply that native operation; record synthetic provenance/outcome. Event fields explicitly specify modifier, keyboard, composition, input and bubbling/cancelability. |
| `browser.dom_batch` | Finite list of admitted immediate DOM actions, document generation and action cap | Dispatch synchronously in one page task with no intervening microtask; return ordered action/error facts and stop-on-error position. No loops, branches, waits, callbacks or executable strings. Can selects the actions and checks effects. |
| `browser.clone_subtree` / `browser.append` | Owned node; later parent/child handles from the same document generation | Clone to a distinct owned node handle or append that node; report identity/connected state and native errors. Cloning does not copy event listeners or mean reattaching the original. |
| `browser.fetch` | Page, method/URL/headers/body, explicit credentials/redirect policy, bounds | Invoke native Fetch in the page's real origin/cookie/CSP context; return pending identity and bounded status/body/native-failure facts. Never substitute an external HTTP request or arbitrary source evaluation. |
| `browser.observe` | Typed feature/global-presence/property/event-detail query | Bounded primitive facts or tagged JSON-safe snapshot. Effectful getters/serialization are explicit, not implicit. Identity uses native handles where necessary. |
| `browser.load_artifact` | Blank page, leased browser artifact, declared entry and DOM-ready contract | Load ordinary Can-generated artifact; return distinct module-load, DOM-ready and declared data-export/error facts. No authored JS string or old TS vector oracle. Record compiler/bindings provenance and subject-reported values; loading alone is not completion. |
| `browser.watch` / `browser.read_events` | Context/page, kinds, cursor, limits | Arm/read request/response/failure and console/page errors; preserve navigation/request identity, bounded bytes, repeated headers and error details. Arm acknowledgment confirms the listener boundary, not future event completion. |
| `browser.watch_dom_event` | Page/node target, event type, listener phase/registration boundary, projection and limits | Acknowledge listener installation before the selected action, at the declared point relative to application boot/listeners. Copy delivered event type/key/modifiers/composition/input data, bubbles/cancelable, `defaultPrevented`, `isTrusted`, target/document identity and sequence as seen by that listener. Custom-event detail uses the declared bounded representation. No application adapter echo or case callback. |
| `browser.route` | Context, finite method/origin/path/resource matcher, priority, pending cap/deadline | Register interception; token/snapshot precede that request's driver-managed upstream contact, not all browser egress. First matching priority wins; duplicate priority rejects. Unmatched traffic follows network policy. |
| `browser.route_fetch` | Paused token, explicit redirect policy, body bound, deadline | Contact upstream once while retaining paused delivery. Completed means full bounded body captured with terminal EOF plus status/repeated headers/hop facts and response identity; timeout/overflow returns partial/incomplete facts. No retry. |
| `browser.route_resolve` | Token, continue-original/abort/fulfill-fetched/fulfill-bytes | Resolve once. After fetch dispatch, continue-original rejects; fulfill-fetched requires complete retained response, while fulfill-bytes explicitly supplies exact replacement. Return delivery identity/outcome separately from upstream contact facts. |
| `browser.unroute` | Rule | Stop new matches; report outstanding tokens, which remain owned until resolved/aborted. |
| `browser.checkpoint` | Watch cursor, bounded observation boundary | Return prefix watermark and pending/gap facts; never claim terminal event completion. |
| `browser.seal` | Watch cursor, owned terminal close, deadline | Stop context activity and drain tracked callbacks; return terminal watermark, pending routes/body captures, gaps and exact final completeness scope. |
| `browser.screenshot` | Page, leased output, byte/dimension bounds | Owned bytes/artifact digest and dimensions or error. Can chooses retention and comparison. |
| `browser.close` | Page/context/connection, deadline | Stop work, resolve/abort owned actions, close resources, drain events, return receipt. Shared remote browser server remains running. |

### Browser launch admission and host effects

An owned profile and temporary HOME are insufficient isolation. `browser.open`
must bind engine/driver identities, the complete effective launch configuration,
test-only credential policy, destination policy, and the externally owned
process/connection scope. It must neither consult personal profiles/credentials
nor request native credential, account, permission or default-browser UI.
Missing required guarantees make the selected capability unavailable before
navigation. Never dismiss prompts automatically or modify the user's Keychain
to make a test proceed.

The preferred implementation candidate is the pinned driver's launch path inside
an N-owned service, with acquisition intent recorded before calling the driver.
This retains reviewed platform defaults; it does not establish cleanup authority
by itself. N must retain verified process identities and reclamation authority
if the service dies during or after launch. A direct native launch would instead
need an explicitly versioned complete recipe audited against driver defaults.
The [design consultations](preparation/native-can-test-design-review-2026-09-30/consultation.md)
support the driver launch as the initial implementation candidate. Both approaches
require identical host-isolation and service-death qualification, not a
case-authored launch script.

The failed PM-B1 launch omitted `--password-store=basic` and
`--use-mock-keychain` from the installed Playwright defaults. Adding these is a
candidate correction, not proof of isolation or a universal recipe for other
engines. Future qualification must name an independent host-effect observer,
its covered UI classes, visibility/permission boundaries and interval through
browser/service disposal. A missing observer, a sampling gap or an unobservable
native surface is `unknown`, not `no_ui`. Do not infer absence from headless mode,
DOM events, process exit or an empty scratch directory. If the selected host
cannot demonstrate the requirement, block it or explicitly select a qualified
isolated execution host; do not silently weaken the claim.

Unexpected native UI invalidates launch isolation and stops further live
admission. Record the observed effect even when exact process attribution is
unknown, dispose only proven owned resources, and preserve uncertainty. A known
incident reported after a receipt is an append-only invalidation of the affected
qualification evidence; it cannot be erased by a later clean run. PM-B1 remains
failed; its prompt was already canceled by the user and needs no further action.

### Input, delivery and disposal are separate observations

`input_begin` acknowledges an admitted action, not a completed click. Report the
actual driver's input promise settlement, route contact/body/resolution facts,
and application witness under separate identities. PM-B1 showed that aborting a
route during close may still fulfill that click promise. Neither fulfillment nor
rejection substitutes for response delivery, application state or clean disposal.
Can decides which combination satisfies the case.

On closing, stop new actions and preserve settlement events for admitted input.
Do not perform a success-only DOM lookup once its observation interval has closed.
The generic driver emits facts even if a case no longer wants that lookup; it
must not suppress real diagnostics or classify route abort as test success.
Can separately validates the declared close/abort/settlement/seal ordering.

DOM capture is independent of the candidate Can control adapter. Live input `value` differs from its attribute; checkbox `checked` differs from `value`; selection differs from default selection. Retained nodes can become disconnected: explicit native DOM inspection/click may still work, while a physical action can reject non-actionability. Navigation invalidates old document actions except explicitly supported retained observations. Can compares dirty reset, composition, caret direction, node counts and order.

Bodies become immutable when captured. Request/response events may precede body completion: emit separate body-completion/error records and track pending capture. Preserve duplicate headers/binary data available at the declared capture layer, and report unavailable or normalized header fields. HTTP error, aborted request, missing body and capture deadline are distinct. State uncaptured/redacted fields so no comparison is claimed without its input. Body records identify transfer/content decoding and any text conversion: browser/HTTP-client bytes are not automatically exact wire bytes. `fulfill-fetched` uses the driver's qualified pass-through representation; `fulfill-bytes` declares representation and matching headers, rejecting unsupported combinations rather than silently changing the intended response.

Arming returns a confirmed listener boundary and starting cursor before Can submits the action. Action replies include action ID, document/navigation generation and a same-driver checkpoint watermark; this is not proof that all resulting events have arrived. Timers/polls are not attributed to the latest click without evidence. Can waits for its explicit witnesses, reads cursor deltas and finally seals before absence/exact-count claims covering the full leg. Local watermarks never manufacture cross-process causality.

For the existing cancellation check, wait for application-listener readiness, then install and acknowledge a bubble listener on the same node after its canceling handler. Same-task batch qualification needs an independent event-realm ordering witness, such as a reviewed generic observer recording delivery and its queued microtask marker; a driver batch ID or unchanged request count alone is insufficient. These native observers expose facts only; Can owns comparison and phase expectations.

Can owns interception choreography: hold a save, change controls, fetch the real upstream response, then corrupt delivered bytes after commit. The driver chooses neither the race nor its verdict. Token states are paused, fetching, fetched, resolving and resolved/aborted, with explicit failed/indeterminate outcomes. Fetch dispatch permanently consumes the original-contact opportunity even if its response acknowledgment is lost. Rejoin/query the same operation/token after lost acknowledgments; never issue a second upstream contact or resolution under a new ID. Preserve upstream attempt/hop identities and delivered-response identity separately. Overflow/truncation cannot support exact pass-through/corruption evidence. All upstream hops obey context destination policy; indeterminate contact/delivery remains reported even if the token is later aborted. Each token has a hard deadline; expiry aborts and records incomplete execution, never successful empty response. Actions/backpressure stay bounded if Can stops.

Application replay uses the same payload `operation_id`/idempotency key in a new
intentional request with fresh native action/request/route identities. It does
not reuse a native operation ID, which would only join the original operation,
or redispatch an uncertain native fetch/resolution. Can may request application
replay only after the prior native contact/delivery facts and relevant server
effects have been reconciled under the case contract. Record the two identity
spaces separately.

Arm observation before actions. Final sealing stops/closes context activity, drains all tracked body/error callbacks and records watermark, cancellations and unresolved work. Closing can produce errors: retain their causal facts for Can's explicit expectations. A quiet interval or DOM predicate is only a **prefix checkpoint**, not proof of no later error. Driver death, gaps, unknown callbacks or seal deadline means incomplete evidence. Absence claims apply only to the declared complete interval.

Pre-contact policy, when claimed, must cover redirects, preconnect, service workers, WebSockets and remote-browser egress. Origin routing/request logs alone do not prove this. Qualify the enforcing host/forwarder boundary; otherwise the required leg is unavailable. A narrower observed-request report is not equivalent coverage. Can chooses origins and interprets denied attempts within N's envelope. Close only owned remote Firefox contexts/connections, not shared `launchServer`.

## 6. Independent native observations and faults

[`tests/conformance/native.ts`](../../tests/conformance/native.ts) embeds operations and answers together. Move selection, expected values, sequencing and verdicts to Can. Native services execute one reviewed operation per call; no `qualify_json`, `test_array` or source-eval escape hatch.

| Operation | Inputs | Native effect and result |
| --- | --- | --- |
| `native.open` | Candidate runtime/bindings, observer identity, scope, limits | Isolated subject/observation process, session and supported catalogue. Never silently use R's runtime. |
| `native.describe` | Session, registered API | Effect-free known presence/descriptor metadata or explicit unknown. Never execute getters/Proxy traps to infer value type or callable availability; effectful discovery requires an explicit observed action. |
| `native.make` | Primitive bits/text, ordered entries or reviewed hostile descriptor | Raw value handle and construction result/error; no expected answer. |
| `native.invoke` | Registered typed operation, receiver/argument handles | Exactly that operation; returned/thrown-value handle or pending action. Missing API explicit; subject exceptions are data. |
| `native.settle` | Pending action, deadline | Fulfillment/rejection or pending, with raw handles/events; no extra implicit assimilation. |
| `native.observe` | Handle(s), observation kind, bounds | Scalar tag/bits, source lexeme, descriptor, ordered entries, identity, serialized bytes, counters/events. State gaps and deliberate effectful reads. |
| `native.gate` / `native.release` | Session, gate, bounded action | Allocate/release once; return arrival/release facts. Only asynchronous continuation points may suspend on a gate. |
| `native.fault` / `native.restore` | Fresh session, reviewed target, absent/throw/revoked/counted-delegate mode | Install/remove fault with timing/outcome. Never patch R, another case or shared runtime. |
| `native.close` | Session, deadline | Stop admission, settle/cancel work, destroy handles/process, restore or destroy faults, return receipt. |

Every operation-catalogue entry declares whether it returns a raw value synchronously or starts an asynchronous action, and exactly which assimilation/await behavior belongs to the native API itself. RPC wrapping/decoding must not add a thenable read or call; use inert handle/completion envelopes, not the raw value as a host Promise return. A settle deadline leaves the owned action pending until explicit cancellation/close or its hard lifetime limit.

PM-N1 supports one getter/alias construction and its contamination control, not
the full N1 value matrix. PM-N2 supports direct source `encodeJSON` access only.
Before migrating candidate-adapter facets, an ordinary C-compiled fixture must
exercise generated ingress/adapter wiring, with artifact/export and invocation
witnesses, authenticated local completion handling, an R-owned oracle and a
seeded adapter defect. A native service may expose a known typed operation; it
may not accept an arbitrary module path/export/eval string or run the old probe's
expectation table. Retain the tiny source instruments as research evidence only.

The N2 invocation witness has an explicit contract. Its reviewed observer/binding
identity, C artifact/export/signature, generated entry and runtime/import manifest
are fixed before launch. The observer records at the actual production-adapter
boundary reached by that generated entry, using captured native counter/identity
operations; candidate-returned `invoked: true` or a plausible error shape cannot
authenticate the call. R checks those inert facts against its Can-owned plan.
The precise current-version hook/linkage must be reviewed and qualified as part
of N2; no stable ABI or arbitrary dynamic importer is assumed. If instrumentation
changes the subject, record both artifact identities and its scope, and retain an
uninstrumented production leg before claiming production parity. A bypass that
returns plausible output without reaching the adapter must fail the witness;
wrong export/signature, eager getter access and forged local completion are
separate controls. This is a trusted observation boundary against candidate
defects, not a claim that same-realm instrumentation is a sandbox for malicious
JavaScript. If it cannot observe the actual generated path independently, N2
remains blocked and its binding design must be revised.

Production parity requires the same declared fixture inputs and expected Can
checks in both legs, with matching output/error semantics, raw identity/access
facts and complete observation intervals under the same limits. Identify the
observer-only instrumentation delta and otherwise bind compiler/options/runtime
and fixture identities. Ignore only explicitly declared observer events or
nonsemantic identifiers/timestamps. Each leg must independently satisfy R's
expectations; equality of two wrong outputs is insufficient. Any unexplained
behavioral divergence blocks parity, and merely running an uninstrumented
artifact supplies no acceptance credit.

Raw handles are not Can objects and cannot cross sessions. Compare identity in the original realm, not after JSON copies. Use tagged IEEE-754 bits where needed so signed zero, NaN and infinity survive. JSON reviver tokens/large integer lexemes cross as exact strings with unambiguous property/index paths; absent source context is explicit. Parse, raw-JSON construction and stringify are separate actions. Preserve collection order; Can decides whether it matters.

Hostile descriptors support immediate return/throw, counted property reads, reviewed Proxy traps/revocation, and counted thenable read/call/resolve/reject. Counters record actual invocations, never expected counts. Describing a value must not accidentally trigger a getter or assimilate a thenable; use an explicit get/await operation. Synchronous getters/Proxy traps **cannot wait for controller RPC or return a Promise in place of their synchronous result**. They use fixed immediate behavior; asynchronous settlement may use gates.

Richer observer callbacks are a **binding proposal, not established expressibility**. Ordinary named Can callables normally lower to async `Promise<Completion<T>>`; specialized `value`/`invokeSync` paths do not prove arbitrary synchronous getter/Proxy/comparator companions. Any synchronous callback must execute locally under a checked admitted contract. Immediate reviewed native descriptors suffice for some hostile values; unavailable richer bindings remain explicit gates without restricting ordinary Can helpers.

Keep raw candidate-Bun facts, R-built probe behavior and C production-adapter behavior distinct. `RunOutput` validates the generation runtime identity/private module hashes: an R probe cannot silently import substituted C runtime bytes. R code running under C's Bun still uses its identified R helpers. A hostile value passed to C needs reviewed same-realm typed ingress and local handling of C completion brands; return only inert facts to the trusted judge. No candidate code enters R's judge, no live controller closure crosses realms, and no foreign source string becomes a callback. A C-generated callback is subject code. Record all artifact/binding provenance, with no backwards ABI promise. Qualify this boundary through N1/N2 in the hard-case register before claiming candidate-adapter coverage.

Can compares native reference/error-occurrence identity, signed zero, exact lexemes, getter/then-call counts, Set order and immutable-array inputs against its own expectations and negative controls. Matching two outputs through the same adapter does not prove that adapter. Zero-call/absence-of-late-fault claims require complete observation.

## 7. Additional shared observations required by the ledger

These inherit the same authority/outcome/budget contracts; no family-specific host scenarios.

| Surface | Native operations and facts | Can policy |
| --- | --- | --- |
| HTTP | `http.begin`, `write_body`, `end_body`, `read_headers`, `read_body`, `close`: owned pending request; raw status/repeated headers, bounded bytes, EOF/error and upload progress. No default retry/redirect; explicit redirect policy records hops. | URLs, bodies, decoding, readiness/retry and accepted responses |
| Controlled peer | `peer.listen`, `accept`, `read`, `write`, `half_close`, `close`: hold real listening socket, report arrival/bytes/connection state; literal bytes permit malformed/truncated transport. | Protocol bytes, gates, trickle/disconnect schedule and expectations |
| WebSocket | `ws.connect`, `send`, `read_events`, `close`: opcode/payload, open/error/close, locally requested versus remotely observed code/reason, bounded pending work; external to subject server. | Message order, shutdown witness, held work/release and comparisons |
| Database | `db.namespace`, `connect`, `begin`, `execute`, `query`, `settle`, qualified `cancel`, `commit`, `rollback`, `close`: owned namespace, pinned connection/transaction, parameterized SQL, operation start/settlement identity, engine/driver, native SQLSTATE/errno/constraint, affected count and typed ordered rows. Ambiguous commit is indeterminate. | Schema/SQL, concurrency, isolation/locks, rows/counts, replay/rollback and independent reads |
| Object store | `store.prefix`, `put`, `get`, `list`, `delete`, `close`: register bucket/prefix authority before writes; exact bytes/digests, pagination and errors; quiesce accepted writes before deletion. | Names/content, fault sequence, expected durability and allowed retention |
| Artifacts/environment | Owned bytes/files, managed product CLI calls, explicit scoped environment snapshots/read leases | Archive/install, formatting/docs/catalogue freshness, artifact and qualification expectations |

Streaming reads distinguish data, pending/deadline and terminal EOF/error. Writes report accepted bytes, not peer consumption. HTTP headers/body expose the native capture layer and any normalization/decoding, matching browser evidence rules; exact malformed-wire obligations use controlled peers. Every resource has destination policy, byte/pending limits and external cleanup. Interfaces accept data operations, not scripts with native branching/expectations/retries.

The database observer uses a separately identified connection/driver path, not the Can adapter under test. If both use Bun SQL, independence is from the **Can adapter**, not Bun SQL itself; testing the lower driver needs a different observer. Preserve int64/decimal as exact text, bytes, null, booleans, floating bits and explicit date/time encodings. Do not sort rows or retry ambiguous writes. Cleanup authority must be restricted to owned fixtures, not inferred solely from caller-chosen prefix strings.

Transaction operations retain the exact native transaction/connection through drain and settlement; a pooled query cannot substitute for a session-bound `LAST_INSERT_ID`. Return accepted/started, visible deadline, driver settlement, server-side quiescence and resource release separately. A cancel request states support and server acknowledgment; a JS/Can timeout alone does not cancel SQL or release its lease. Current budget machinery does not support SQL cancel signals. For `commit_unknown` or a budget-expired write, fence further mutations, establish server settlement through acknowledgment or independently qualified reconciliation, then make a fresh independent read. Client promise rejection or connection close may still leave a server effect unknown. An earlier empty result cannot establish no effect; unresolved server state prevents clean disposal/success. Namespace deletion requires proven quiescence and owned authority. The hard-case review records which stronger sentinel/independent-read checks are new qualification evidence rather than observations present in the old F02/F03 drivers.

Object-store ownership precedes seeding. Cleanup waits for accepted writes to finish or become unresolved, enumerates all pages and deletes only owned objects. Empty eventually consistent listing is not proof of cleanup: report the qualified consistency guarantee or unresolved state. Never delete shared buckets or retain credential contents in reports.

`METRIC` ledger rows remain obligations under the user's measurement deferral. Later authorized native counters may report identified wall/RSS facts; thresholds stay in Can. This design starts no measurements and substitutes no estimates.

## 8. Structured diagnostics as a product CLI

Specify **`canlc check --json PROJECT_DIRECTORY`**, not `compiler::expect_rejection`. It loads/resolves/checks without assertions, emission, publication, application execution or dependency network fetching; a runnable entry is unnecessary. Dependencies/options are identified inputs. Inability to provision them is not source rejection.

[`driver.CheckSnapshot`](../../compiler/internal/driver/diagnostics.go) already produces inert diagnostics with codes, severity, related locations and zero-based UTF-16 positions. It lacks this public CLI and the phase/completion/input-identity contract. Its `CAN-PROJECT` diagnostics include load failures, so neither that code nor message text establishes source-versus-infrastructure classification.

Stdout contains one bounded versioned JSON document with these normative fields. Final wire spelling is chosen once without retaining a legacy format.

| Field | Contract |
| --- | --- |
| `schemaVersion`, `kind` | Explicit diagnostic schema/product command kind |
| `status`, `result` | Completed plus accepted/rejected; failed envelope has a failure kind and **no result** |
| `compiler` | Artifact/checker identity; Can separately binds the supervisor's actual executable lease, not self-identification alone |
| `inputs` | Declared input-root snapshot identities plus sorted logical paths, lengths/SHA-256 of bytes actually read, directory-enumeration/absent-lookup facts affecting resolution, manifest/options identities and overall digest; no private absolute paths |
| `diagnostics` | Complete ordered records: producer phase, severity, stable code, message, primary and related locations |
| `completeness` | Per-stage finished/stopped/not-entered state, stopping diagnostic where applicable, and unvisited units; no hidden I/O/cancellation/panic/report-cap omission |

Phases are `project`, `lex`, `parse`, `resolve`, `check`, attached at the producing stage. Locations use logical input path and zero-based UTF-16 start/end line/column, end exclusive. Spanless means null with a reason, never a fabricated line-zero squiggle. Optional byte offsets must be named separately, bound to digested input and converted correctly across astral Unicode. Keep related locations ordered/distinct. Minimal checking does not need fixes; future fixes require source identity and are never applied by check.

| Exit | Meaning | Report |
| --- | --- | --- |
| `0` | Completed accepted; warnings/notes allowed, errors forbidden | Completed/accepted, complete identity and diagnostics |
| `1` | Completed source/project rejection with an attributable source error | Completed/rejected, identified input and producer phase |
| `2` | Invocation, I/O/identity, dependency provisioning, internal, resource, cancellation or incomplete-report failure | Bounded failed envelope where possible; no rejection result |

Malformed source/manifest content or deliberately missing declaration in a known input set may reject. Missing root, denied read, unavailable external dependency, unknown/mutating input or panic fails execution. Missing referenced project module versus unavailable external dependency is classified by the loader/resolver's typed cause and declared manifest, not generic filesystem error text. Analysis may stop after a real parse error; it need not type-check an unparseable graph. Interrupted analysis cannot be accepted as a negative result.

The declared/leased input closure and the actual-read set are separate fields. Early rejection can leave declared units unvisited; identify them and the stopping phase rather than representing the read subset as a fully checked project. Hash the compiler's actual read snapshot, including relevant manifest/dependency bytes, and record namespace lookups affecting resolution. An immutable input lease is the strongest launch guarantee; post-hoc hashes alone do not establish confinement or absence of temporary mutations. Can compares reported identities with its fixture manifest and executable lease. Enforce report size during production/encoding: exceeding budget is failed/incomplete, never truncated diagnostics labeled complete. Stderr is bounded auxiliary detail, not the semantic interface.

Can invokes through `proc`, parses data and validates status/report agreement. Signals, timeout, absent/malformed/truncated JSON, unknown schema, identity mismatch or exit 2 cannot satisfy expected rejection. Can owns the paired valid control, exact mutation, expected phase/code/severity/count/span, unrelated-error policy and warnings. Wrong-phase rejection fails. The CLI receives no expected diagnostic or test verdict rule.

## 9. Compositions and coverage gates

These outlines are composed by ordinary Can functions with normal fixture-backed assertions, not a new test DSL.

| Scenario | Can-authored sequence |
| --- | --- |
| Compiler rejection | Materialize/seal valid input → candidate check → require accepted → make exactly one invalid revision → check → compare code/phase/span/count → dispose and inspect receipt |
| Subprocess | Spawn with explicit fd map → write/close input → collect bytes/exit → compare → close; partial delivery/output gaps remain visible |
| Server lifecycle | Spawn → poll chosen health predicate → hold request and observe arrival → signal → observe external shutdown witness → release work → inspect response/raw committed DB state → collect exit/cleanup |
| Browser race | Open/watch → navigate → hold request → alter controls → fetch upstream once → inspect independent committed row → fulfill selected altered response → compare DOM/network/occurrences → final close/seal and late-error checks |
| Native runtime | Open candidate session → construct raw/hostile values → invoke native/identified adapter → observe identity/bits/source/counters → compare expected facts → settle/close |

| Ledger family | Required contracts before retirement |
| --- | --- |
| Compiler/CLI/distribution | Checking schema, real candidate build/install artifacts, paired/wrong-phase/identity controls |
| Native/runtime and delegated oracles | Identity/exact encodings, hostile read/assimilation counters, scoped faults, order/late-settlement facts; no copied old qualification oracle |
| Browser BROWSER-001–056 | Native controls and emitted clients, required engines, typed actions/DOM, interception, wire artifacts and final event seal; limitations explicit |
| SQL/storage/lifecycle | Raw rows and transport witnesses, transaction/replay/crash schedules, pre-write namespace ownership, quiescent/paginated cleanup |
| External callers/release | Can report plus matching N receipt; reject missing checks, partial scope, gaps and unknown cleanup; reference acceptance remains separate |

The [ledger](native-can-tests-migration-ledger-2026-09-30.md) remains authoritative for individual facets/gates. This table neither migrates 292 obligations nor proves every required binding exists. Retirement requires equivalent protected behavior, required independent observations, deliberately failing controls and resource completion. Historical experiments/static fixtures keep their ledger dispositions.

Shared implementation admission controls must cover interrupted acquisition, worker/controller death, duplicate/conflicting operation IDs, lost acknowledgments, stale/foreign handles, symlink/path races, overflow, lingering descriptors/descendants, interrupted cleanup and safe recovery. Browser controls also cover late callbacks, unresolved routes, upstream duplication, driver death and shared-service ownership. Diagnostics need warnings-only acceptance, each rejection stage, Unicode/spanless/related locations and infrastructure controls. These are future bounded correctness checks, not runs authorized now.

## 10. Evidence and remaining engineering design

| Existing source | Missing guarantee |
| --- | --- |
| [File writes](../../runtime/platform/files/write.ts), [path helpers](../../runtime/platform/files/path.ts) | Owned confinement, race-resistant operations, storage bounds, seals and recovery |
| [Child spawn](../../runtime/platform/process/spawn.ts), [group termination](../../runtime/platform/process/wait.ts) | External pre-exec ownership, live handles, private descriptors and proved descendant containment |
| [Supervisor](../../compiler/internal/driver/supervise.go), [launcher](../../compiler/internal/driver/runtime.go) | Bound buffers while receiving; report cleanup failures; recover ownership; prove host containment |
| [Browser scripts](../../tests/integration/browser/grid.mjs) | Can-owned sequences plus typed actions/observations; event sealing and qualified egress |
| [Native qualification](../../tests/conformance/native.ts) | Separate actions from answers; isolated raw observations/faults with provenance |
| [Diagnostic snapshots](../../compiler/internal/driver/diagnostics.go) | Product CLI, phase/completion/read-identity and typed failure classification |

Three fresh Jev consultations preferred outside-worker ownership, typed native facts composed by Can and a product diagnostics CLI. Selected probabilities were `1/1/1`, `1/1/1` and `0.98/0.99/1.00`. The [consultation/reconciliation record](preparation/native-can-test-capabilities-2026-09-30/findings.md) preserves all requests/responses and wording audit. Advice is not proof of sufficiency, enforceability or agent-authoring optimality.

Before freezing implementation lanes, complete host-specific enforcement/recovery design and supported guarantees, executable profile admission for the lifecycle contract's finite ceilings, canonical Can types/errors/catalogue declarations with full boundary fixtures, exact protocol wire schemas, and the mapping of every ledger facet to concrete bindings. These are engineering proof obligations, not missing user preferences. Preserve the reference refresh and nonpublishing suite launch contracts; no new interpreter decision is required.

Supporting source reviews: [workspace/process](preparation/native-can-test-capabilities-2026-09-30/workspace-process-review.md), [browser](preparation/native-can-test-capabilities-2026-09-30/browser-review.md), [observation/diagnostics](preparation/native-can-test-capabilities-2026-09-30/observation-diagnostics-review.md). This main contract governs where exploratory notes are narrower/ambiguous; findings record corrections. The [final review](preparation/native-can-test-capabilities-2026-09-30/review.md) and [documentation validation](preparation/native-can-test-capabilities-2026-09-30/validation.json) do not constitute runtime evidence.
