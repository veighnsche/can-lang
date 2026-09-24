# Product and platform alternatives

24 September 2026 · P4 candidate contracts for DI-09–16. This is the comparison
baseline; later planned selections appear in
[accepted technical decisions](accepted-technical-decisions.md) and
[product resolution status](product-resolution-status.md).

This packet compares bounded options against the strongest current Can idiom. It
uses the [decision inventory](decision-inventory.md), [server evidence](server-evidence.md),
[browser evidence](browser-evidence.md), [lifetime/deployment evidence](lifetime-deployment-evidence.md),
[platform research](platform-research.md), [constraints](constraints.md), and
[evaluation protocol](evaluation-protocol.md). Their source baseline is
`02a549d28fd5fc5c3996160e65a97de332390d30`; upstream documentation describes
possibilities that still need verification on Can's pinned binaries. This packet
adds no executed probe, browser implementation, Linux qualification or Jev result.

Each topic keeps three evidence layers separate: **current implementation**,
**candidate semantics**, and **untested product requirements**. Current decisions
remain authoritative. Alternatives below are inputs to experiments and later
three-request Jev consultations for difficult choices, not accept/retain/reject/defer
dispositions. They select no spelling, framework, Linux target or offline promise.
The broader Can-authored frontend/backend recommendation remains the target; a
server-driven milestone or typed foreign-client bridge proves a narrower result.

Compare correctness, refactor safety, diagnostic repair and successful agent-task
tokens under the evaluation protocol. No source-length, human-familiarity or
compatibility argument settles a choice. Prefer native JavaScript/Bun/browser
operations with only the adapters required by Can's contracts and immutability.

## DI-09 — Linked server and client contracts

### DI-09a: endpoint identity, captures and URL construction

**Current implementation and strongest idiom.** Mount exact literal paths with
named callbacks. Put invoice and tenant identifiers in query parameters, extract
and validate them explicitly, and centralize URL construction in named functions.
Use safe URL values for rendered actions and protocol tests to catch drift. This
already supports CRUD; safe URL admission does not prove endpoint membership,
method, request shape or target presence. Dynamic route segments are rejected.

**Candidate semantics.** Compare three separable increments:

1. Keep exact routing and strengthen a library/tooling baseline: one application
   endpoint record or named builder centralizes method/path and tests all mounts
   against rendered actions. This reduces duplication but cannot claim compiler
   diagnosis of every string reference.
2. Introduce a checked endpoint identity for exact paths. Its references connect
   mount, method, URL/action construction and declared request/response contract.
   Changing the endpoint contract diagnoses incompatible references. This tests
   whether identity alone solves the observed drift before adding path grammar.
3. Extend that identity with bounded named path captures and a URL builder taking
   those captures. Specify capture wire text, typed conversion and parse failures;
   encoding round trips; static/captured route precedence; duplicate/ambiguous
   mounts; trailing slash, encoded slash and malformed encoding behavior. A
   native Bun router is a lowering candidate only after pinned-version probes.

A URL builder may prove it refers to a declared endpoint. Availability after
deployment, runtime authorization, and conditional DOM presence remain separate.
Endpoint export/import identity must compose with DI-05; a source rename and a
wire path change are distinct edits even without backwards-compatibility duties.

**Untested product requirements and discriminator.** The invoice resource URL
`/tenants/:tenant_id/invoices/:invoice_id` is a proposed navigation contract, not
an implemented mount. Compare creation, route rename, method change and bad
capture repair against the query baseline. Exercise links from both HTML and a
client action. Require diagnostics only where a candidate promises static links;
record protocol failures elsewhere. Native routing success does not establish
the linked Can contract.

### DI-09b: form wire fields and validated application values

**Current implementation and strongest idiom.** Decode a shallow wire record of
`str`, optional `str` and `str[]`; explicitly validate into domain records. Use
named validation/render helpers that return retained raw input plus accumulated
field errors. Keep parsing numeric seats and optional details explicit. Current
strict decoding rejects unknown fields and scalar duplicates and preserves
repeated array values; unrelated repeated columns are not checked line records.

**Candidate semantics.** Compare a reusable validation library with checked field
references derived from the wire record and an action-linked form contract.
Field references can connect control names, retained values and error paths;
they must specify what a rename diagnoses. The linked action declares its wire
decoder, validator and response cases, without implicitly converting form text
as if it were JSON. Validation may collect independent field failures while
keeping dependent checks ordered and preserving their finite error identities.

For line items compare (a) explicit parallel arrays with a validated length/order
invariant, (b) a declared keyed row representation and checked row-key/path
decoder, and (c) a separate JSON action using the existing exact codec contract.
Specify missing versus empty values, repeated row keys, removed/reordered rows,
partial rows, duplicate scalar controls, and stable mapping of errors back to
input. A JSON action is not automatically an HTML form decoder. Package-owned
validated values depend on DI-02; no decoder or fixture may mint them by bypassing
their owner if that candidate is later adopted.

**Untested product requirements and discriminator.** Retain rejected seats,
optional details and line rows; show all intended field errors; do not align a
price with the wrong row after reorder. Rename a wire field and a domain field
independently. Compare library and checked-link repair cost before proposing a
form-specific language mechanism. Server validation remains mandatory for a
typed browser client.

### DI-09c: authorization and request lifetime

**Current implementation and strongest idiom.** Authenticate using the available
cookie/CSRF/crypto primitives, derive an explicit server-controlled actor/context
record, and pass it to a named protected operation. Check membership and resource
access in the protected read/write; include tenant and revision predicates or an
explicit locking/isolation strategy. Current examples do not demonstrate a
tenant invoice authorization flow. A prior read in a default Read Committed
transaction does not by itself freeze authorization or the invoice revision.

**Candidate semantics.** Compare the explicit context approach with a scoped
operation capability only if the latter prevents a demonstrated misuse. Such a
capability must state actor, resource, permitted operation, scope and revocation
or revalidation behavior. Opaque identity alone is insufficient; a capability
that outlives the request cannot silently preserve an expired authorization
fact. Context capture can use existing named functions/context records; new
capture syntax belongs to DI-08, not this contract.

**Untested product requirements and discriminator.** Reject tenant B's invoice
for an actor allowed only tenant A; attempt identifier substitution, stale
revision, concurrent membership change and delayed callback use. Decide and
test the missing/forbidden disclosure policy. Inspect durable rows, not just
status codes. Capability complexity needs a concrete benefit over explicit
operation checks before becoming a candidate implementation task.

### DI-09d: action results and live UI targets

**Current implementation and strongest idiom.** Map finite application outcomes
to safe response builders in named handlers. Keep route/form/target names
centralized where possible and use companion protocol/browser tests. Existing
HTMX swaps 422, but excludes other 4xx/5xx; returning the account-search 503
fragment therefore does not establish visible feedback. That specific browser
reproduction is still missing.

**Candidate semantics.** Compare (a) explicit application mapping plus a narrowly
configured status/swap policy, (b) a checked action-result mapping that requires
success/invalid/conflict/forbidden/unavailable cases to choose status, rendering
and swap behavior, and (c) typed target references within a declared render
scope. A target reference proves at most its declared scope and allowed use;
conditional removal requires a runtime missing-target outcome or a restricted
renderer construction that actually guarantees presence. Do not claim a general
DOM-presence proof from an ID type. Status handling must preserve the real HTTP
failure status rather than changing failures to 200 solely to force a swap.

**Untested product requirements and discriminator.** Observe 200/422/409/403/503
status, body and headers, then the resulting DOM, retained input, focus and error
announcement. Remove the conditional target, rename it, and submit twice while
responses arrive out of order. This separates a declared action interface from
actual visible behavior and supplies DI-13's required observation strength.

## DI-10 — Safe HTML authoring

**Current implementation and strongest idiom.** Use named field/table/message
render functions with the safe HTML builders. Text uses native escaping; URLs,
attributes and structure are validated. Literal structures incur runtime checks,
but the existing constructors and helpers are the baseline to beat, not an
unfactored call tree. Raw markup and authored scripts are outside the boundary.

**Candidate semantics.** Compare improved helper/library composition, compiler
checking of literal calls to existing builders, and a checked declarative
component representation. The latter two may reject illegal static tags,
attributes, child structure and linked field/action references while retaining
runtime checks for dynamic values. Define component input/output contracts and
what static checking actually knows about conditional children. All alternatives
must preserve escaping, URL scheme checks, immutable safe nodes and controlled
browser assets. No template spelling or unrestricted HTML escape hatch is
selected. Element additions needed for accessibility are distinct catalogue
work, not evidence that a new template form is necessary.

**Untested product requirements and discriminator.** Build the same invoice form
and row/message components with each candidate. Inject hostile text/attributes,
bad URLs, illegal literal structure, dynamically invalid structure and wrong
action/field references. Require precise failure at the promised boundary and
equivalent rendered behavior. Measure agent creation and repair outcomes; fewer
builder calls alone do not justify grammar.

## DI-11 — Browser scope, state and lifecycle

### DI-11a/b: execution target and shared contracts

**Current implementation and strongest idiom.** Can executes on Bun and serves
safe HTML/HTMX. A table containing ordinary form inputs, validation fragments,
indicators and polling is a useful server-driven baseline. The present admitted
surface does not supply Can-authored local state, custom keyboard handlers,
immediate draft calculations or rich optimistic rollback. Authored browser
JS/TS is not an existing supported workaround within current Can admission.

**Candidate semantics.** Compare an explicitly narrower server-driven milestone,
a supported typed bridge to another browser language, and a separately compiled
Can browser target. Only the last can establish a Can-authored rich frontend.
A bridge must itself be an admitted, checked platform proposal; generated client
contracts do not automatically solve endpoint drift. For a browser target start
with a bounded main-thread capability profile: shared data/validation code,
DOM/event operations, Fetch, explicit state transitions, rendering and disposal.
Worker execution is a separate profile because workers cannot directly operate
the document. No framework, reactive effect system or worker support is implied.

The compiler must reject server-only capabilities transitively through imports,
generic specializations and callbacks. Shared code needs exact cross-target
wire conformance for records, variants, unknown/duplicate fields, integers and
finite floats; native browser `Number` cannot be assumed to preserve Can's full
integer contract. Maintain immutable aliases and completion boxing. Server
secrets, environment access, SQL pools and process/file operations must neither
be callable nor bundled accidentally. Lower admitted operations to native
browser APIs with bounded ownership/codec adapters.

**Untested product requirements and discriminator.** Run one invoice grid with
local draft, immediate exact totals, keyboard use, optimistic save and rollback.
Change a route and a wire field across separately compiled targets. Verify
diagnostic boundaries, network bytes, durable rows, bundle contents and browser
behavior on named versions. A successful server slice or foreign-language grid
cannot be counted as a Can browser compiler result.

### DI-11a/c: offline behavior as technical alternatives

The evidence does not settle a durability promise. Compare the following
increasing contracts before a preference question; none is a present Can feature.
An online indicator is only a hint, so actual transport/status results drive
save state in every option.

| Candidate | Bounded contract | Required machinery and failure probes |
| --- | --- | --- |
| Memory draft with explicit retry | Keep unsaved edits while the view stays alive; show unsaved/failed state; retry deliberately after endpoint reachability returns. No reload or navigation survival promise. | Immutable draft and acknowledged revision, mutation identity, response ordering and conflict handling. Disconnect during editing/save, reconnect, repeated retry, late success and navigation. |
| Durable local draft, explicit submit | Persist a versioned draft locally and restore it after reload when storage remains available. Submission is a user action or separately declared policy; local persistence alone does not queue remote effects. | Browser storage adapter and ownership, schema migration, per-user/tenant isolation, sign-out handling, quota/permission/eviction outcomes and visible persistence status. Reload, failed writes, changed server revision, shared-device login change and storage loss. |
| Durable mutation queue with reconciliation | Persist identified pending mutations, retry under a declared ordering/conflict policy, and reconcile uncertain acknowledgements against server state. Promise deduplicated business effects only where the server protocol supports them. | Durable queue transitions, idempotent server action identities, replay/version policy, crash recovery, multi-tab coordination and authorization expiry handling. Crash before/after local enqueue, server commit and local acknowledgement; reconnect with conflicts; duplicate sends and multiple tabs. |

Durable drafts do not imply offline reload of the application shell. If launching
or reloading the application while offline is required, packaging/cache/service
worker behavior becomes another explicit capability and qualification problem.
Background synchronization is likewise not implied by a foreground retry queue.
Storage quotas, eviction and device loss bound every durability claim; expose
storage failure instead of acknowledging persistence that did not happen.

For the representative experiment, use memory retention as the minimum case,
then fault-test durable draft and queued mutation prototypes independently. This
is an experiment ordering, not a selected product promise. The broader grid
requirement can be evaluated against all three, with measured state complexity,
agent repair cost and failure outcomes available for later Jev analysis and any
remaining product tradeoff.

### DI-11c: state transitions, accessibility and disposal

**Current implementation.** No Can browser state/lifecycle implementation is
established. Current HTML tables/forms and ARIA attributes provide a useful
server baseline; attributes alone do not implement an interactive grid's keyboard
behavior. Existing tests do not prove optimistic rollback or disposal.

**Candidate semantics.** Represent acknowledged revision, current draft, pending
mutation identity and save/error state explicitly. Compare save serialization
with revision/mutation-identified concurrent saves; a stale result must not
overwrite a newer draft. Compare pessimistic save with optimistic display as
separate interaction contracts. On rejection, retain newer edits and use a
declared rollback/rebase rule rather than blindly restoring an obsolete snapshot.
Transport failure is not proof that a write failed; reconcile uncertain results.

Compare native listener/timer/fetch ownership scoped to a view with application
scope for drafts or pending saves that intentionally survive view removal.
Disposal stops view updates and releases listeners/timers; aborting Fetch cannot
promise server rollback. Late completions require an explicit destination or
discard policy. Do not detach a durable mutation silently from its tracking owner.
Use ordinary table/input keyboard behavior as one baseline; an ARIA grid option
adds navigation/edit modes, focus movement, labels and announcements. Inventory
dialog/canvas/SVG only where the workload needs them.

**Untested product requirements and discriminator.** Slow and reorder saves;
edit again while a save is pending; return 409/422/503; remove the view mid-save;
remount; reload where persistence is promised. Observe focus, exact totals,
retained draft, network traffic, database state and leaked listeners/requests.
Test keyboard and assistive technology on a named matrix. A browser passing one
happy-path grid demo does not qualify every client capability.

## DI-12 — Sequencing and cleanup in product flows

**Current implementation and strongest idiom.** Use `match chain`, named helpers,
finite errors and scoped transaction operations; explicitly close/cancel resources
on intended terminal paths. Runtime owners protect scoped handles and record
omitted explicit close as cleanup failure. The stream example's error branches
do not fulfill its prose promise of closure on every path. Automatic cleanup
therefore cannot be presented as successful explicit cleanup.

**Candidate semantics.** First compare a complete chain/helper rewrite and a
scoped library API against a narrow recovery-and-continuation form. Preserve
original error distinctions and coordinate generic error forwarding with DI-03.
Separately compare cleanup reporting policies: primary body error plus observable
secondary cleanup failure; an explicit combined failure value; or cleanup taking
precedence with preserved primary provenance. The current scoped rule replaces
success with cleanup failure but preserves an existing body failure while
recording cleanup failure separately. A new helper/control form must state its
double-failure behavior and cannot silently mask either event. No policy is
selected by the example inconsistency.

**Untested product requirements and discriminator.** Inject validation/read/write,
decode/limit, close and cancellation failures, including primary plus close
failure. Observe terminal completion, diagnostic identity, native use after scope,
close invocation and actual resource release separately. Exactly-once intended
close must not be inferred from owner fallback. Compare agent repair before a
new sequencing or cleanup syntax is justified; production example repair follows
the chosen contract, not this preparation note.

## DI-13 — Observable assertions

**Current implementation and strongest idiom.** Keep attached assertions for
Can computation and fixture argument/queue behavior, then companion HTTP,
browser and live-database tests for external observations. Bare `=> ok` on an
opaque server response does not inspect status, headers, body, DOM or effects.
This existing layered test idiom is the comparison baseline.

**Candidate semantics.** Compare better companion-test tooling/scenario reuse
with admitted read-only protocol projections or assertion operations for opaque
responses, and with an authored scenario runner that connects source tests to
browser/database observations. Specify which layer each assertion can observe;
an HTTP projection still cannot prove a target swap. Preserve fixture invocation
identity, checked arguments, isolation and FIFO behavior. Cross-package scenario
selection depends on DI-06 and must survive caller-label renames before reusable
product fixtures are trusted. No universal property proof follows from any runner.

**Untested product requirements and discriminator.** Deliberately construct a
200 response with the wrong body, an escaped-looking but executable payload,
a 503 fragment excluded from swapping, a correct UI with wrong persisted rows,
and a duplicate webhook effect with a success response. Each relevant observation
must fail at its promised layer. Report conditional integration-test prerequisites
and skipped tests; a passed source assertion cannot substitute for an absent
browser or real-database run.

## DI-14 — SQL mutation results and schema assurance

### DI-14a: bounded mutation results

**Current implementation and strongest idiom.** Use static descriptors and bound
parameters, `execute` affected-row counts, then a read in the same scoped
transaction when canonical server data is required. Apply authorization/revision
predicates at the write. Zero affected rows need application classification.
Current row-returning descriptors require bounded `SELECT`; mutation `RETURNING`
is rejected. Native transaction failure after a commit decision can produce
`commit_unknown`.

**Candidate semantics.** Compare known-value responses or the two-statement
baseline with dialect-aware mutation-returning descriptors. Specify permitted
INSERT/UPDATE/DELETE forms, finite row/cardinality bounds, zero-row result,
returned projection codec and constraint/error handling. Preserve parameter
binding and runtime row checks. PostgreSQL native `RETURNING` is an implementation
candidate, not an ORM or a cross-dialect guarantee. It does not resolve
authorization, stale revision or commit uncertainty on its own.

**Untested product requirements and discriminator.** Measure one invoice update
that returns canonical values and one ledger/outbox mutation under realistic
concurrency. Test zero/multiple rows, constraints, wrong projection and uncertain
commit, then compare round trips and agent repair against the baseline. A shorter
query sequence is insufficient without equivalent business invariants.

### DI-14b: schema-checking claims

**Current implementation and strongest idiom.** State the three existing checks
accurately: descriptor/parameter/cardinality admission, live execution, and runtime
row validation. Pair migrations with real-database integration tests. Present
compile-time descriptors do not verify projections against a database schema.

**Candidate semantics.** Compare the accurate runtime-tested contract with a
versioned schema snapshot as a deterministic build input. A snapshot alternative
must define dialect/version, provenance, supported query analysis, migrations,
staleness detection and deployed-schema verification. A database-connected build
check is a separate option with availability/reproducibility costs; it must name
which database/schema was checked. Neither guarantees that production matches
the build unless deployment verifies it. Runtime codecs remain required under
all options.

**Untested product requirements and discriminator.** Change a projected column,
scalar type, nullability and migration order; test stale snapshot and drifted
deployed schema. Distinguish compile-time rejection, startup/deploy rejection and
runtime `schema_mismatch`. Decide only after evidence whether a stronger build
promise materially improves agent task success over accurate documentation and
live integration tests.

## DI-15 — Cancellation, races and shutdown

### DI-15a: owned work and bounded service termination

**Current implementation and strongest idiom.** Keep owners/leases, operation
deadlines and explicit fetch/stream cancellation. Guard service work, drain it
and let a named supervisor enforce the outer process grace/kill policy. Race
losers retain leases until settlement. A close deadline limits a caller's wait;
it does not cancel arbitrary work or release the underlying lease. Permanently
pending work can prevent clean root drain.

**Candidate semantics.** Compare better operation-specific APIs/documentation
and host qualification with an explicit cancellation context propagated only
through adapters that support it. Inventory waiting for acquisition, acquired
resource, active native operation, response/body consumption and completed side
effect separately. A supported SQL reservation-wait signal does not establish
running-query cancellation. Distinguish cooperative cancellation requests,
observed settlement, clean resource release and supervisor kill. Worker/process
isolation is a separate option for bounded computation termination and carries
serialization/effect-recovery costs; no general arbitrary-Can abort is promised.

**Untested product requirements and discriminator.** Client disconnect during
invoice commit, pending outbox delivery, hung race loser, close timeout, shutdown
during active request and forced termination must report their actual outcomes.
Observe connection acceptance, outstanding owners, leases, durable rows and
external effects. Host kill bounds process lifetime but proves neither cleanup
nor rollback. Retry/reconciliation after restart belongs to the service contract.

### DI-15b: empty dynamic participants

**Current implementation and strongest idiom.** Explicitly guard a dynamic list
where an application needs a participant. Retain native `Promise.race` pending
behavior for an empty expansion and `Promise.any`'s all-failed outcome. Use an
external test deadline to observe pending behavior without calling it completion.

**Candidate semantics.** Compare application guards with a reusable validated
nonempty collection/helper. A static nonempty requirement or changed empty-race
result is a distinct larger proposal crossing LD24, not a silent bug fix. Measure
whether guard omissions recur before considering stronger admission.

**Untested product requirements and discriminator.** Run zero/one/many candidate
services, collection filtering to zero, failure-before-success and a permanent
loser. Verify native winner behavior, diagnostics and service shutdown separately.

### DI-15c: consumed-fault visibility

**Current implementation and strongest idiom.** Choose current first-success
race for fallback and use explicit participant-level reporting if needed. Earlier
failures can be consumed by a later winner; late standard failures are diagnostic.

**Candidate semantics.** Compare explicit reporting with opt-in observation of
consumed faults while keeping winner selection unchanged. Specify standard versus
domain failures, attempt/participant identity, diagnostic timing, deduplication,
owner lifetime and observer failure handling. Aggregate/bounded observation may
be useful for noisy retries but needs a stated loss policy. DI-18b is the same
observability seam and must not generate a duplicate feature task.

**Untested product requirements and discriminator.** Use a replica/fallback flow
with deliberate domain fallback, accidental native failure, late failure and
repeated attempts. Measure whether an agent identifies the failing dependency
without treating routine fallback as an incident. More logs alone are not success.

## DI-16 — A qualified Linux target

**Current implementation and strongest idiom.** Use the qualified pinned Darwin
ARM64 distribution and describe its scope accurately. The builder, qualifier,
verifier and isolation checks are Darwin-specific. A source test run on Linux or
an unqualified Bun installation is not a Linux Can release.

**Candidate semantics.** Compare qualifying one named Linux OS/architecture/libc/
Bun combination first with a larger matrix only if deployment requirements
justify it. Candidate packaging can be a native archive or a pinned container
artifact; a container still needs an architecture, libc, provenance, runtime and
host signal/isolation contract. Specify archive/executable verification, launcher
build, native behavior inventory, Linux isolation, install/upgrade integrity,
service signals and supervisor grace/kill behavior. Do not select versions or
claim support before checking the target environment. This is distribution work,
not grammar.

**Untested product requirements and discriminator.** Build, verify and install
the selected artifact on the actual target; run native API conformance and the
same invoice/webhook, database, browser-facing HTTP and shutdown scenarios against
that installed artifact. Record clean exit, reported timeout and supervisor kill
as distinct outcomes. A wider support matrix must repeat target-specific
qualification rather than inherit evidence by name.

## Integrated server slice and dependency handoffs

The invoice and webhook are acceptance workloads, not existing product evidence.
The invoice slice connects authentication, server-controlled context, protected
tenant/revision write, raw-form validation, safe rendering and observed browser
feedback. The webhook slice must name one provider signature scheme, verify
exact raw bytes and framing, timestamp/key rules and appropriate comparison
semantics, then use a durable unique event identity. In one transaction record
receipt, business mutation and outbox work. A repeated event returns the declared
duplicate outcome. After `commit_unknown`, look up the durable identity before
deciding whether to retry or reconcile.

Outbox delivery is at-least-once unless a stronger receiver contract proves more.
External success followed by failure before local acknowledgement must be
injected. Stable provider idempotency keys or a documented reconciliation policy
are needed to claim deduplicated effects; neither transactions nor a queue alone
make remote delivery exactly once. The current HMAC primitives do not themselves
establish a provider adapter or constant-time header comparison operation.

| Dependency handoff | Contract supplied or needed before implementation planning |
| --- | --- |
| DI-05/06 → DI-09/13 | Composable endpoint/package identities and rename-stable shared fixture ownership; preserve queue/argument guarantees. |
| DI-02/03/08/22 ↔ DI-09/12/14 | Raw versus validated values, finite validation/transaction errors, explicit context capture, and safe same-typed record construction. Authorization remains operation bound. |
| DI-09/10/13 ↔ DI-11 | Endpoint method/captures, request codec, response cases, field/error paths and actual target behavior feed both server and browser trials. |
| DI-14 ↔ invoice/webhook/browser queue | Revision predicates, durable event/mutation identity, uncertain commit lookup and dialect/cardinality behavior underpin retries and reconciliation. |
| DI-12/15/18 ↔ DI-11/16 | Ownership, cleanup precedence, abortability and late-fault reporting must agree across disposal, request shutdown and installed deployment. |
| DI-19 ↔ DI-11/15/16 | Any new native browser/storage/provider adapter needs admitted signatures, error/fixture/owner contracts, native lowering and target conformance. |

## Evidence gates before a final task list

These are required experiments and decision inputs, not approved implementation
tasks or a final ordering of language changes:

1. Register comparable current-idiom and candidate workloads, exact models/effort,
   held-out edits, semantic acceptance checks and task-token accounting before
   trials. Keep compiler correctness/security/lifetime tests separate from agent
   success measurements.
2. Execute the current invoice and signed-webhook baselines with a real database,
   including tenant isolation, revision races, invalid retained input, duplicate
   events, outbox recovery and commit uncertainty. Supply provider-specific facts
   before any signed-webhook safety claim.
3. Reproduce 503 visibility on the pinned browser asset and test 200/422/409/403/503
   across protocol, DOM and persistent effects. Then compare endpoint/form/target
   links, safe component options and observable assertion alternatives.
4. Prototype a bounded Can browser slice and compare the narrower baselines.
   Test capability closure, wire parity, keyboard/state/rollback/disposal, then
   each offline contract and its storage/reconciliation faults separately.
5. Inject cleanup/cancellation/shutdown faults with native observations, including
   zero-participant races and pending losers. Qualify any claimed Linux target on
   its installed artifact with the same server slices.
6. For difficult technical choices, prepare three semantically equivalent, fully
   reworded Jev requests using these facts and concrete experiment results; save
   requests/responses, compare disagreement and investigate it. Advice does not
   replace tests. Resolve any remaining product tradeoffs after research and
   consultation; exact syntax still requires the later syntax decision stage.
7. Record accept/retain/reject/defer for every DI-09–16 subtopic with evidence and
   reopening conditions. Only then derive dependency-ordered implementation
   tasks, changed authoritative contracts, required runtime lint/format/check
   work and acceptance tests. No task may silently promote an untested product
   requirement into an implemented guarantee.
