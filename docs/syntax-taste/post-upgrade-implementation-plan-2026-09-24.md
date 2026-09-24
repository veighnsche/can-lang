# Implementation plan for the post-upgrade fixes

24–25 September 2026 · source baseline `fbd2a5614dbba660b085b6fec8aef4f5d902c051`

Implement all 11 **Fix** findings from the [disposition ledger](post-upgrade-dispositions-2026-09-24.md),
using the [selected behavior](post-upgrade-selected-behavior-2026-09-24.md)
as the language/runtime contract and the [invoice acceptance gate](post-upgrade-invoice-acceptance-2026-09-24.md)
as the application completion test. The [ordered task list](post-upgrade-implementation-tasks-2026-09-24.md)
defines dependencies, file ownership and a three-worker schedule. This is a new
implementation round; it does not reopen or renumber the historical T01–T27
execution record. No task in this plan is marked implemented.

The baseline includes the preparation documents and saved experiments added
after the source commit above. The first execution task records the commit
containing these inputs before creating worker checkouts. A bare checkout of
the source commit above omits them.

## Scope and completion

| Finding | Implementation outcome | Tasks carrying the change and its proof |
| --- | --- | --- |
| B01 | Structural browser capability/asset auditing accepts harmless quoted host words and rejects actual forbidden access across the complete reachable runtime. | UP10, UP11, UP15, UP23 |
| B02 | Capped native `matchAll` iteration advances empty Unicode regex matches correctly. | UP03, UP23, UP24 |
| U01 | One handler-free action package; checked declaration symbols feed client and server without mirrored contracts or stubs. | UP05, UP08, UP16, UP19, UP20, UP21 |
| U02 | Request-aware mounted callables capture one startup pool and explicit origin; protected operations authorize live requests and tokens expire after drainage. | UP06, UP08, UP09, UP16, UP19, UP22 |
| U03 | Declared captured routes run directly; codecs, statuses and contract edits stay linked across targets. | UP08, UP09, UP13, UP19, UP20, UP21, UP22 |
| U04 | Compiler-owned browser entry/profile/bundle and verified server pairing run the empty app and invoice in required browsers. | UP04, UP07, UP10, UP11, UP13, UP15, UP18, UP20, UP23, UP25 |
| U05 | Checked event registration cancels admitted native defaults synchronously and respects disposal. | UP11, UP13, UP20, UP23 |
| S01 | Checked HTML cases, including real 503, update only the admitted live target; request/response guards reject unsafe task/header shapes. | UP08, UP12, UP19, UP22, UP23 |
| S02 | Startup and callback faults report one sanitized, source-located occurrence; later events and disposal still work. | UP07, UP11, UP13, UP23 |
| S03 | Grid focus occurs after attachment; notices/pending state persist; late outcomes preserve newer edits and view lifetime. | UP20, UP23 |
| P03 | Public generic composition checks parametrically, validates recursive components atomically and emits only reached concrete calls/types. | UP02, UP14, UP17, UP21, UP24 |

UP24–UP27 integrate, qualify and document the entire result. The 19 deferred
proposals remain excluded, especially general stack-safe iteration (P05),
changed HTTP race-loser ownership/cancellation (P06), components, reload-durable
drafts, error-set parameters and a general host extension API. Existing package
identity, owner records, variants, lexical fixtures, SQL adapters and Bun
coordination are retained foundations. This round consumes their current
interfaces; it does not reimplement their historical design packets.

There are no external compatibility consumers. Replace obsolete action
spellings, required `handles`, client stubs and the test boot contract, and
migrate maintained callers/tests in the same integration round. Do not retain
an alias, legacy syntax parser, duplicate route or old generated layout merely
to keep an old golden green. The seven-day browser asset retention and invoice
replay retention are selected runtime guarantees, not source compatibility.

## Architecture and implementation sequence

### 1. Establish the two difficult prerequisites early

UP02 makes declaration-only symbolic types ineligible for emitted models.
The [generic experiment](evidence/2026-09-24/selected-behavior/experiments/generic-recursion/findings.md)
shows that even public `identity<T>` currently checks but fails complete module
emission. Separate proof-time graphs from concrete runtime graphs; retain a
hard failure for a symbolic value or missing concrete callable that escapes
the boundary. Test complete assertion and production emission, not only an
individual emitted function.

In parallel, UP04 establishes an explicit browser execution-context channel.
It preserves context across native `await`, interleaved owners, callbacks,
coordination, resource leases and drainage. The [browser experiment](evidence/2026-09-24/selected-behavior/experiments/browser-owner/findings.md)
rules out the current synchronous `AsyncLocalStorage` shim. A small runtime
test can qualify an internal adapter, but generated Can integration in UP11
and real supported builds in UP23 must qualify the actual profile. Do not
describe the experiment's toy token control as an implementation.

UP03 independently fixes Unicode regex advancement using a fresh native
global `RegExp` and capped `String.prototype.matchAll`. Retain existing flags,
UTF-16 offsets, capture defaults, result shape, caps and named failures.

### 2. Replace action coupling with checked declarations and mounts

UP05 implements the selected grammar, canonical declaration identity, typed
record captures, body budgets and exhaustive case metadata. UP08 implements
`action::mount`, `action::url`, `action::request` and `action::post`, including
request-first handlers, exact result/error bounds and distinct HTML renderers.
All action-specific schema information is derived from the shared declaration.
Importing one does not register a route or pull server handlers into a browser.

UP06 first makes request-token revocation explicit: materialize the response
or upgrade decision, abandon unread body, drain admitted work, revoke in the
outer `finally`, then return the native response. UP09 builds JSON/form
dispatch on that boundary and native `Bun.serve({ routes })` matching. Reuse
the [guarded route evidence](preparation/route-capture-design.md) for strict
capture decoding, canonical int64, method/static precedence and 400/404/405
classification. Do not replace native routing with a second language-level
router or claim to recover original request bytes already normalized by Bun.

UP12 adds the compiler-owned HTMX guard and exact checked status policy. It
uses the pinned asset's request/response/swap events, the submitted target's
node identity and the entire task list before mutation. Real 200/422/409/403/503
action responses get their declared inner swap; unrelated failures remain
blocked. Existing non-action examples that relied on the global 422 exception
must move to checked local HTML action policy. That migration uses the selected
action mechanism; it does not add an unrestricted status override API or alter
the pinned vendor asset.

### 3. Make browser delivery a supported toolchain operation

UP07 splits portable runtime semantics from Bun-only machinery. Seal concrete
type/error identities and immutable source diagnostics at build time; retain
private value branding, exact codecs, standard failures and completion rules.
Share portable code where possible and add only profile adapters required by
the host. A second ad hoc runtime or aliases for unavailable Node semantics
are not completion paths.

UP10 structurally validates full generated/dependency/runtime import graphs
and final published JS/map/diagnostic bytes. UP11 makes browser entry checking
target-aware, threads explicit context through all generated async paths, and
emits only the reachable production closure, including callbacks and concrete
specializations. Ordinary source checking still applies to shared packages;
unreachable HTML/SQL implementation is not emitted merely because its package
was imported. Browser `main` has zero arguments and starts once after DOM
readiness. Bun/server `main(str[] args)` keeps its separate target contract.

UP13 supplies strict query decoding, synchronous cancel-policy listeners,
source-located fault reporting and native same-origin Fetch using action
metadata. UP15 invokes the bundled native `Bun.build({ target: "browser" })`
inside verified build staging and produces served-ready digest JS, map,
diagnostic table and manifest. Assertions still run supervised; an assertion
failure, bundle failure or graph violation prevents selecting the build.

UP18 introduces the explicit server `--browser-manifest` handoff. Verify shared
locked instances, the browser build ID, every asset hash and all published
references before selecting the server report/asset table together. Include
the browser input in the server build identity and guard against input changes
between verification and publication. Retain replaced digest assets for at
least seven days, including across restart/cleanup; partial or failed rebuilds
leave the previous verified set intact. The app page obtains its script URL
from that report under the selected same-origin CSP.

### 4. Migrate the existing invoice in place

UP16 establishes the shared `invoice_contract` dependency using the repository's
current project lineage and lock format, and migrates invoice data/SQL/domain
logic together. The selected wire shapes and integer tenant/invoice keys
replace the current duplicated string-ID wire shapes. Complete all helpers
shown only as named call sites in the design packet. SQLite schema,
descriptors, seed/inspection data and assertions must agree with the new model.
This is an example migration, not new schema-aware SQL language tooling.

Protected reads/writes recheck live session, membership, tenant/invoice and
revision inside their transaction. JSON replay stores the canonical digest
and committed result with the mutation, checks authorization again on every
lookup, and has a commit-time-anchored seven-day fixture window with no sliding
extension. Cleanup serializes with lookup/mutation. Revisions are never reused.
Expiry, lost acknowledgement and view disposal do not permit a blind new-ID
retry. HTML and JSON saves share the protected domain operation while retaining
their different transport/result types and replay promises.

UP19 binds the real GET/save/form handlers to one startup pool and canonical
public origin. It removes body/hidden session tokens, per-request pool opening,
manual action status/codec tables and query-route workarounds. The application
serves `/invoice-grid?tenant=1&invoice=7` and `/tenants/1/invoices/7/edit` itself.
UP20 simultaneously migrates the existing grid to that shared package,
zero-argument bootstrap and checked requests. It repairs focus, notice and
pending state using immutable application records, with app-owned attempt
settlement and view-owned DOM/listener lifetime.

### 5. Finish generic composition beside platform work

UP14 introduces canonical public proof contracts, dependency ordering and
atomic strongly connected component (SCC) validation. Symbolic calls substitute
validated signatures without creating concrete specialization keys. Every
internal cyclic edge admits only bare caller formals or fully closed types;
stationary, permutation, duplicate/drop cases stay finite, while growing
`T[]`/`box<T>` cycles reject. Acyclic `identity<box<T>>` remains legal. No member
of a failed component publishes a valid proof. Private templates cannot borrow
a public proof, and representation-dependent operations still require explicit
operations on the opaque type.

UP17 closes the loop through reached concrete specialization, source maps,
strict emitted TypeScript and actual execution. A public generic helper that
checks but fails full emission is not implemented. Runtime recursion depth and
termination remain outside this feature.

### 6. Prove the combined capability and ship truthful documentation

UP21 edits each shared contract once and rebuilds both targets. Metadata-only
edits update both sides automatically; stale source references diagnose before
publication. Observe new and old route responses, request bytes, result tags,
handler entries, DOM and database effects. A positional constructor can remain
valid after a field rename if both generated codecs update. Do not replace this
test with text comparison of duplicated declarations.

UP22 proves direct native HTTP/SQLite behavior, authorization and lifecycle.
UP23 proves empty-app/runtime and invoice behavior in named Chromium and WebKit
through final served assets. Remove `build-grid.mjs`, test-written host pages,
proxy rewrites and runtime aliases from the passing path. Keep useful browser
observers and named transport-fault injection, without letting tests supply
application handlers, boot or runtime semantics. Preserve unrelated existing
webhook, form, SQL and owner regressions.

UP24 operates all required prerequisites and verifies the integrated tree and
fresh emitted output. UP25 repeats the full invoice gate on the installed
selected Linux distribution, recording actual versus emulated execution and
each browser's version. Missing archives, databases or required engines are
incomplete qualification, even when a development test normally skips them.
UP26 updates source/CLI/reference/example documentation from this evidence.
UP27 independently checks all 11 fixes, every deferred boundary and artifact
provenance before recording completion. No public deployment or release upload
is part of this plan.

## Parallel work and integration rules

Use three coding workers plus one coordinator. The compiler lane owns common
checker, type, IR and emitter changes; the server lane owns server adapters and
invoice domain/server code; the browser lane owns runtime profile, graph audit,
delivery and grid code. Roles move at explicit handoffs in the task list.
Concurrent work uses isolated `codex/` branches/worktrees based on the recorded
integration snapshot. A worker receives its task, dependencies, write set,
contract links and required validation; it reports its commit, tests, remaining
limitations and any interface change.

The coordinator serializes catalogue generation, module-inventory updates,
shared lock changes and formatting/merge reconciliation. Workers propose
changes to shared generated inputs through the owning task; they never edit
`runtime/catalogue.ts` or pinned vendor files by hand. Run repository-wide
runtime format commands in a worker's own checkout, then inspect the diff so
unrelated formatting does not enter a task.

Work may start when its dependencies are integrated and its write sets are
free. Numerical order is a valid serial execution order; the wave table is a
conservative concurrent order, not a duration estimate. An incompatible API
cutover may temporarily leave dependent examples unbuildable on the integration
branch until their listed migration tasks land. Record those known failures;
do not mark the application gate green or add compatibility shims. Every task
must pass its own meaningful checks, and UP24 is the required complete-tree
green barrier before final installed qualification.

If a mechanism fails a proof/runtime gate, stop its dependent tasks, preserve
the failing evidence and revisit that mechanism without reducing the acceptance
contract. Independent lanes can continue. The saved [three fresh Jev consultations](evidence/2026-09-24/selected-behavior/jev/findings.md)
inform the selected design; scheduling needs no new semantic decision. A new
difficult design decision requires three fresh, fully reworded consultations
with saved requests/responses and a disagreement audit under `AGENTS.md` before
the contract and affected tasks change. Jev agreement is advice, never a test.
