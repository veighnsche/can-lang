# Observable product acceptance matrix

24 September 2026. P8/P9 working contract for the recommendation program.
These observations test the **product result**, not just Can type checking or
attached assertions. They are prerequisites for claiming the server-driven
and then broad Can-authored frontend recommendations. Exact implementation
mechanisms remain in the topic design packets. A disposable current-Can
[invoice slice](invoice-current-probe.md) passed 18 assertions but exercised
no live database, HTTP response or DOM; a separate [HTMX probe](htmx-503-probe.md)
observed an actual unswapped 503. Neither qualifies the future invoice flow.

## Server-driven SaaS flow: tenant invoice edit

| Case | Required external observation | Contract checked separately |
| --- | --- | --- |
| Open and save valid invoice | Authenticated actor sees escaped invoice data; a valid POST returns actual 200, canonical committed values and one revision increment. Database row after response matches rendering. | Action identity, shallow wire decode, owner factories, safe HTML, transaction and authorization. |
| Invalid numeric seat count, optional detail and repeated rows | Actual 422; original submitted text and row order remain visible; field-specific messages identify each invalid control. No protected write occurs. Unknown, duplicate-scalar, missing and malformed form fields follow the specified pre-handler response rules. | Raw wire field identity is separate from domain value; no decoder mints an owner record. Row decoder rejects partial, duplicate and misaligned input rather than associating a price with another row. |
| Stale revision | Actual 409 with the unsaved draft and conflict feedback visible. The prior committed row is unchanged; no silent overwrite. | The protected mutation checks expected revision at write time. |
| Other tenant's invoice identifier | Denied without revealing invoice/tenant data in body, URL or logs available to the actor. Protected rows remain unchanged. Test a valid own ID, a foreign ID and a nonexistent ID under one declared disclosure policy. | Server-derived actor/context and operation-bound membership/resource/tenant checks; a valid typed ID is no permission. |
| Storage unavailable or uncertain commit | Actual 503 or declared infrastructure response; draft remains available and feedback is visible. After uncertain commit, the client reconciles by a stable operation key or fresh authorized read before retrying a mutation. | No status alone proves rollback. Inject failure before write, after write but before response, and during render. |
| Route, field or case change | A route/method/field/case refactor produces precise diagnostics at all statically linked consumers or an explicit protocol-test failure at the remaining dynamic boundary. The unedited other-tenant and stale-write negatives still pass. | Source `action` symbol, `form::field`, exhaustive response map, safe URL builder and matching coverage. |
| Added error and same-typed field reorder | Adding a named domain error forces an explicit handler/bound decision. Reordering adjacent same-typed fields preserves intended totals and recency through domain factories or a selected named construction form; a silent value swap fails acceptance even if source still compiles. | Compare the current positional/factory baseline before any named-field grammar; test changed files, diagnostics and actual output rather than source appearance. |
| Path-captured resource URL | The same invoice can be opened and posted through the accepted typed resource path; builder and matcher round-trip admitted int/str captures. Malformed or structurally ambiguous paths receive the specified 400/404/405 and never invoke the protected callback. | Exact action first, bounded captures second; native Bun normalization/precedence must be observed. |
| Response display | For saved/invalid/conflict/forbidden/unavailable, record status, headers, escaped body and resulting visible DOM; 422/409/403/503 must not rely on a successful 200 rewrite. Retained values, focus and error announcement follow each branch's stated policy. | The actual browser adapter must implement the declared policy. Conditional target absence and out-of-order responses are negative cases. |

The [selected keyed-row form contract](keyed-row-design.md) uses a stable key
in every row field name plus an exact order list. A [12-check native probe](keyed-row-probe.md)
showed that equal-length parallel arrays can misassociate values after
different rows omit controls. Structural failures retain known raw text in a
`form::rejected` view for a separate safe 422 renderer and never enter the
protected handler.

## Server-driven SaaS flow: signed provider webhook

| Case | Required external observation | Contract checked separately |
| --- | --- | --- |
| Valid first delivery | Authenticate the provider signature over the exact received bytes; persist one ledger business effect and an outbox item in one transaction; return the declared protocol acknowledgement. | Use existing native crypto, HTTP and SQL primitives; no new queue/billing grammar. |
| Replay of delivery key | A second identical delivery yields no second business effect; the durable ledger records or returns the established outcome. | Idempotency key scope, uniqueness and transaction boundary are explicit. |
| Changed bytes or bad signature | Reject before protected mutation; no ledger or outbox effect. | Signature timing/algorithm and raw-body limits are protocol-specific, not inferred from a typed value. |
| Commit outcome uncertain | On `commit_unknown`, look up the stable delivery key in a new transaction before deciding whether to retry/reconcile; never assume rollback. Inject a lost acknowledgement after durable commit. | Current SQL transaction error contract; no mutation `RETURNING` dependency. |
| Outbox dispatch failure and crash | Durable item remains eligible for bounded retry. Crash after external success and before local acknowledgement does not make a duplicate business effect at a deduplicating receiver; where the receiver lacks deduplication, publish the reconciliation/duplicate policy instead of claiming exactly-once delivery. | The outbox promises durable attempts, not one physical external delivery. |
| Schema and SQL change | Statement/parameter/cardinality checks run at build; live row shape is validated at runtime. A declared projection wider than SQL must not be described as statically proven against a versioned database. | Snapshot/schema-drift tooling is separate optional work. |

Use a real disposable database and a controlled protocol/provider fixture.
Inspect committed rows and outbound attempts, not merely the Can handler's
returned value. Companion tests supplement attached assertions and distinguish
`real-can`, supplied completion and provider fixture evidence.

## Supported service artifact and shutdown

The first Linux target is [Debian 13 amd64/glibc with official Bun 1.4.2
Linux x64](linux-target.md). Pin the archive and image bytes, build the
distribution, install it on that target, and run the same invoice and webhook
suite against the installed artifact. Record native capability availability,
file/process/crypto/SQL behavior and generated TS/Bun version. Inject a slow
dependency, client disconnect, pending race loser, read-plus-close failure and
host termination deadline. Record which operations abort and which stay owned
until settlement; the host must obey a stated shutdown bound without asserting
that arbitrary Can computation is cancellable. An empty dynamic
`race with error` remains pending under the selected native contract.

## Broad Can-authored frontend: editable invoice grid

Run a separately compiled Can browser target against the qualified server
JSON POST save and JSON GET load actions, sharing the HTML action's protected
invoice boundary. The [integrated contract](integrated-action-contract.md)
fixes typed row payloads, initial/reread snapshots, finite outcomes and mutation
replay. Shared records and codecs must agree across Bun and named browser
versions, including integer range, finite floats, unknown/duplicate fields,
variants and protected owner-record wire boundaries. Browser compilation must
reject transitive SQL, process, secret and server-only operations. Use native
DOM, event and Fetch operations with only Can-contract adapters.

The grid first loads an authorized typed snapshot through GET, and uses the same
read path after uncertainty to learn current state. A foreign/nonexistent
invoice returns the same nondisclosing denial. A reread cannot by itself prove
which writer committed when revisions have advanced; identical operation-ID
replay settles the identified save. The grid acceptance case includes keyboard
navigation, immediate exact totals,
local unsaved drafts, optimistic save, rollback of failed saves, slow and
disconnected network behavior, reconnect/retry, target removal, out-of-order
responses and view disposal. The [selected initial offline contract](jev-generic-offline/findings.md)
keeps a draft editable in the living view, shows unsaved/failed state and
requires explicit retry/reconciliation after reconnect. It does not promise
reload durability or an automatic offline queue. Application-owned identified saves
must remain distinct from view-owned event listeners, timers and fetches so
navigation does not discard a committed operation or leak a dead view.

The broad recommendation is earned only after actual interaction, accessible
focus/announcement checks, lifecycle fault tests and the named support matrix
pass. A typed bridge to another browser language may remain a narrower
integration path, but it does not establish a Can-authored rich frontend.
