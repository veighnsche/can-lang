# Accepted technical design decisions for planning

24 September 2026. These are engineering selections for the **planned** Can
design after current-source research, alternatives, three fresh Jev requests
where the choice was difficult, and bounded probes. Current compiler behavior
and `decisions.md` are unchanged until P8 integrates the final specification.
The user's earlier syntax selections are recorded separately in
[confirmed syntax choices](confirmed-syntax-choices.md). Under the later
no-questions instruction, remaining syntax is resolved through evidence, three
fresh Jev consultations for difficult choices, and engineering judgment.

## DI-01 — Checked ordinary-data case names and explicit `bind`

In ordinary-data patterns, a bare identifier must resolve to an admitted
nominal leaf at the expected type or be rejected. It cannot fall back to a
whole-value binder. The user selected `bind name` as the intentional capture
form at top level and inside record/array patterns. `_` remains the explicit
discard/default, and `...rest` remains the visibly marked array-tail capture.
The rule is uniform across expected types and generic specializations: an
identifier never changes from nominal test to binder merely because a field's
type changed. Error/completion `as alias` patterns remain a separate grammar.

The [nested current-source probe](pattern-nested-probe.md) executed 13 cases:
final `decliend` and nested `paidd` silently bind today; the same `paid` text
tests a leaf under `payment` but binds under `int`. Three fresh
[Jev judgments](jev-pattern/findings.md) disagreed weakly between uniform and
contextual policies. The observed type-dependent meaning change is the
engineering reason to choose uniform intent. The
[user's exact syntax answer](confirmed-syntax-choices.md#ordinary-data-pattern-capture--di-01)
settles `bind`. P8 must specify parser/checker nodes, duplicate bindings,
alternative-arm binding equality, nested coverage, hidden-field behavior and
diagnostics. Production tests must include the final-arm/new-leaf and nested
array/record counterexamples plus a held-out type-refactor case.

## DI-02 — Owner-controlled public records for validated values

A declaring package may expose the nominal type of a record while reserving
construction and representation-changing updates to itself. An importer may
hold, pass and compare admitted public identity, and observe only projections
explicitly published by the owner. It cannot directly construct the protected
record, use `with` to replace its representation, bind hidden fields in a
pattern, or obtain one through a generic document decoder. Public wire records
remain decodable; an owner factory validates and mints the protected value.
Owner-authored factories, constants and update operations are trusted creation
paths and must themselves enforce the invariant. A validated tenant identifier
does not authorize an invoice operation; actor, membership, resource and
revision checks remain at the protected operation.

The [two-package probe](validated-value-probe.md) built and ran 11 assertion
roots. It observed importers creating empty email and zero quantity by direct
constructor, `with` and generic JSON decode despite rejecting owner factories.
It also observed fixture input and supplied-completion fabrication. These four
creation routes establish a current semantic gap for a claim that every value
of a nominal type is owner-validated. The [three fresh Jev
judgments](jev-core-contracts/findings.md#di-02-boundary-completeness-precedes-a-favored-prototype)
weakly favored the owner boundary; the executed bypasses and narrower runtime
cost than full opacity or universal fallible construction are the engineering
grounds for selecting it.

The [complete design](owner-record-design.md) selects `owner record email`,
with type export independent of construction rights, all stored fields hidden
from nonowners, ordinary exported projection functions, leaf-only public
matching and existing eligible structural equality. Generic document
encode/decode or wire-schema derivation is rejected whenever a type graph
reaches an owner record, even in the owner package; explicit transparent wire
data and owner factory/projection functions form the external boundary. Three
fresh [Jev requests](jev-owner-record/findings.md) informed the spelling and
boundary; the selected uniform codec rule deliberately differs from Jev's
owner-only derivation preference for the reasons in that packet. P8 must
integrate this with canonical package-instance ownership, assertion transport,
generic specialization and native lowering. Negative acceptance covers
importer constructor, `with`, field read, decode and fixture supply; positive
cases cover real owner factory/update, leaf match, equality and
wire-decode-then-factory flow. No production compiler or runtime change has
been made.

## DI-05a — Dependency-qualified package imports

An importing file identifies a package in a direct dependency with
`dependency_key::package_name`, optionally giving it a file-local alias. The
user selected the concrete surface
`uses [billing::model as bill_model, crm::model as crm_model]` for two direct
dependencies exposing `model` ([answer scope](confirmed-syntax-choices.md#dependency-qualified-source-imports--di-05a)).
Resolution uses the importing project's own dependency table, so the same
source spelling inside a dependency does not refer to a root alias. A package's
canonical identity is its locked resolved project/package instance; a local
alias or short package name does not determine nominal identity. Catalogue
names remain reserved, private `internal` visibility and direct-dependency
confinement remain checked, and output paths/source maps use canonical
identities. The current global short-name graph rejection is removed for
distinct resolved package instances.

The [package evidence](packages-assertions-evidence.md#package-and-error-identity-already-in-use)
shows that current aliases cannot compose the unchanged dependencies. The
[alternatives](core-alternatives.md#di-05a--independent-package-identity)
and [three fresh Jev judgments](jev-core-contracts/findings.md#di-05a-both-lookup-candidates-can-pass-author-renaming-cannot-count)
found source qualification and manifest handles both technically viable; the
user chose the former. The [complete package/error contract](package-error-contract.md)
selects owner-declared `project_id` and `instance_id` UUID lineage pairs as
canonical project instances. A canonical lock pins each instance's content
and owner-relative direct edges, interns identical repeated paths and rejects
divergent snapshots using the same pair in one build. It preserves nominal
identity across root alias edits, unrelated graph growth and checkout
relocation while distinguishing simultaneous divergent lineages. Three fresh
[Jev consultations](jev-package-instance/findings.md) informed this choice;
the exact graph and acceptance limits are specified in that packet. Production
acceptance must load both unchanged same-named packages, reject undeclared
transitive imports, preserve private visibility and reproduce identities
across two locked builds.

## DI-07 — Extensional closed-variant compatibility

Treat a named variant as its closed set of fully specialized nominal leaves for
assignment compatibility. A source variant is assignable to a target variant
when every source leaf is an admitted target leaf. A leaf is assignable when it
is admitted. The relation must not special-case a direct assignment between
different specializations of one variant declaration. Generic **record**
specializations retain their own nominal identities; generic leaf
specializations remain distinct. No runtime wrapper or variant provenance is
introduced by this decision.

The [16-case current-source matrix](variant-matrix-probe.md) reproduced the
present direct rejection but bridge/leaf admission, ordinary `option<T>`
distinctions and codec behavior: 12 builds/24 assertions passed, four expected
checker rejections. This demonstrates a path-sensitive static rule, not a
durable brand. Under the selected rule, equal-leaf `tagged<int>` and
`tagged<str>` become directly compatible as they already are through `bridge`;
whole `option<int>` remains incompatible with `option<str>` because their
`some<T>` leaves differ. The common `none` leaf remains admissible to both.

The [core alternatives](core-alternatives.md#di-07--variant-specialization-and-leaf-compatibility)
considered retaining the direct guard and nominal wrappers. The guard leaves
refactor-dependent admission without preventing bridge or shared-leaf
conversion; wrappers add injection and codec identity machinery for an
unestablished brand requirement. In three fresh
[Jev judgments](jev-core-contracts/findings.md#di-07-transitive-set-inclusion-versus-documented-path-dependence),
two favored extensional leaves and one favored completing the baseline matrix;
the matrix is now executed. This engineering decision follows the semantic
evidence, not a plurality. No user syntax choice is needed because the source
variant/leaf forms remain unchanged.

P8 must specify generic substitution, exhaustiveness, emitted/codec behavior,
diagnostics and negative cases. Production implementation must replace the
checker exception and run the direct/bridge/leaf, `option<T>`, generic record,
nested variant and codec regression matrix. Agent edit/repair costs are
unmeasured; they cannot be claimed as an observed benefit.

## DI-05b — Unnumbered application errors with qualified report identity

Authored application error declarations omit global integer IDs, for example
`error payment_declined(str reason)`. The nominal key is the declaration's
canonical owning package identity, name and fully specialized generic arguments.
Machine-readable diagnostics and reports carry an explicit qualified textual
key, concrete type identity and locked-build identity. Catalogue-owned numeric
metadata may remain internal to native adapters, but it does not identify an
application error. Owner retirement metadata records qualified declaration
keys to reject reuse within that owner's identity scope; foreign tombstones do
not reserve local names. Rename creates a new identity. No old numbered source
spelling or active-number registry is retained for compatibility.

The [consumer audit](error-identity-audit.md) found several current numeric
consumers but no requirement for a naked application number stable across
locked builds. Two independent libraries currently collide at `1000000`
despite distinct nominal declarations. [Three fresh Jev
consultations](jev-remaining-syntax/findings.md#di-05b--choose-unnumbered-application-declarations-and-qualified-reporting)
favored unnumbered qualified identity; the independent-composition contract
and elimination of unsupported global allocation are the engineering reasons.
The [complete package/error contract](package-error-contract.md) fixes the
`can.error.v2/<project_id>/<instance_id>/<package>/<declaration>` declaration
key, concrete generic identity, versioned report fields, lineage-scoped
retirement-only metadata and the two-build acceptance matrix. Retirement
checks require current active/retired disjointness and a supplied predecessor
lock chain; they do not claim to recover discarded history. P8 must reconcile
that contract with all loader/checker/emitter/runtime/CLI consumers and preserve
payload redaction. This is planned design, not a claim the current compiler
admits the declaration above.

## DI-09a/b/d — Source action contract for exact-path forms

The planned bounded action surface is a source `action save_invoice` declaration
beside its public key, shallow form and finite result types, as illustrated in
the [complete packet](action-contract-design.md#a--source-declaration-with-checked-catalogue-operations).
The declaration owns one exact path, method, wire form, size limit and complete
result-to-status/swap mapping. Its static symbol is consumed by checked URL,
POST and mount operations; it is not an arbitrary mutable runtime value.
`form::field` references carry the owning shallow wire record and exact field
type. Server-only mounting supplies a handler and safe renderer separately, so
shared contract modules do not import a database handler into the browser.
Route identity and URL safety do not establish authentication, authorization,
deployed route availability or DOM target presence.

For the first invoice contract, the five result cases map to actual 200, 422,
409, 403 and 503 statuses. The save UI must show each outcome while retaining
the submitted draft where required. A failure status is never rewritten to 200
to force a client swap. The [actual HTMX probe](htmx-503-probe.md) showed why
the adapter must be checked against browser behavior: an existing 503 fragment
arrived over HTTP but left stale 422 feedback visible. The adapter may use a
scoped policy or one verified compatible page policy; it cannot declare a
per-action swap that conflicts with global HTMX configuration. Unexpected
responses remain explicit observable failures.

The [three fresh Jev consultations](jev-remaining-syntax/findings.md#di-09--choose-a-source-action-declaration-with-bounded-semantics)
favored source placement with variable confidence. Ordinary named helpers
remain the evaluation baseline; no measured agent-token or product advantage
is claimed. P8 must settle exact grammar/catalogue signatures, pre-handler
400/404/405/413/415/422 distinctions, response mapping, duplicate mounts and
field diagnostics. The [bounded typed path-capture increment](route-capture-design.md)
is accepted separately for the required navigable invoice resource URL. It
uses required `str`/`int` record fields, canonical integer spelling, strict
single-segment decoding and guarded native Bun routes with method-first/static
within-method dispatch. Its pinned Bun probes, three successive fresh Jev
rounds and exact 400/404/405 boundaries are in that packet. The separately
[selected keyed-row form increment](keyed-row-design.md) uses stable row keys,
an exact order list and retained partial raw submissions for structural 422
responses. The [12-check FormData probe](keyed-row-probe.md) demonstrated that
bare equal-length parallel arrays can silently misassociate cross-row values.
Declarative markup and typed DOM target scopes remain unselected. Acceptance
includes route/method/field/case refactors,
malformed input, live HTTP/browser/database observations and a held-out
creation/repair comparison.

## DI-04 — Explicit operation inputs for exported generics

An exported generic body is checked once under symbolic type parameters. It
may perform an operation on a symbolic type only if the typing rule is valid
for every allowed substitution or the operation is supplied through a written
named callable/dictionary input. A body edit adding an unsupported `+`, field
read or equality requirement diagnoses at the public declaration until the
signature names that operation. Local non-exported templates retain the
documented specialization-checked policy. No implicit instance search, broad
trait system or new capability vocabulary is selected. Emitted TypeScript uses
ordinary native calls for supplied operations.

The [28-assertion helper probe](generic-helper-probe.md) established an
existing explicit-input baseline; [three fresh Jev judgments](jev-generic-offline/findings.md)
favored making that the public contract. P8 must specify the symbolic typing
rule and diagnostic path. Production acceptance changes a dependency body
from parametric forwarding to `value + value` and requires a declaration-site
error, then adds a callable to the signature and requires exact caller repair.
It also retains a local template specialization case. No agent-token benefit
is claimed without a controlled task comparison.

## DI-06 — Owner-scoped lexical fixtures and exported scenarios

An ordinary lexical `when` row or inert `fixture` template is selected only by
a root in its canonical declaring package instance. Foreign display-label
equality can no longer activate a helper row. For deliberate integrated
cross-package behavior, the helper may export a named scenario scoped to one
exact executable callee and a statically unique internal call path. The caller
attaches it to one exact lexical invocation with checked arguments; the real
helper body runs while typed internal completions are supplied. Missing,
stale, ambiguous, duplicate or unused links fail explicitly. Runtime
reservation remains root/table/invocation/participant isolated FIFO with
truthful `real-can` and `supplied-completion` labels.

The [hard-case probe](fixture-hard-case.md) showed that a separate exported
`stamp_with_clock` test entry is a viable current-language alternative but
publishes the private clock dependency. A disposable F2 spike preserved
`stamp(int)` and passed one/two sequential links, though it did not implement
source grammar or concurrency. Three fresh [post-spike Jev
consultations](jev-fixture-post-spike/findings.md) favored F2 with meaningful
residual support for the test entry. The [complete selected
contract](fixture-scenario-design.md) fixes source forms, bounded path rules,
ownership, evidence, lowering and acceptance. Production must prove real
static checks and concurrency before this is credited as supported; whole-task
agent gains remain unmeasured.

## DI-11 — Explicit main-thread browser target and in-view drafts

The [planned browser target](browser-target-design.md) is a separate
`canlc build --target browser` profile with a browser-only capability closure,
ordinary named Can state-transition functions and a small explicit catalogue
over native DOM, events and Fetch. Opaque view handles own listeners/timers;
an app owner tracks identified saves across view disposal; a versioned state
cell publishes immutable snapshots without general mutable aliases. No
virtual DOM, reducer runner, implicit reactivity or worker profile is adopted.
Pure shared contracts and exact codecs must agree between Bun and named
browsers, including int64, finite floats, variants, owner-record wire
boundaries and duplicate/unknown fields. Browser builds reject SQL, processes,
server secrets and other server-only capabilities transitively.

The first offline guarantee is live-view editability, visible unsaved/failed
state and explicit retry/reconciliation after reconnect. It does not promise
reload-durable drafts or automatic queued mutations. The [three fresh Jev
judgments](jev-generic-offline/findings.md) favored that bounded promise;
another [three](jev-grid-contract/findings.md) favored direct native-DOM
catalogue operations for the first client. These are design choices, not a
passing browser build. P8 must integrate the action/row contracts and target
capabilities; production acceptance is an actual keyboard/slow/offline/
rollback/disposal grid in the named browser/version matrix, with server rows,
wire bytes, focus, announcements and leaked resources observed.

## DI-09 final integration — Three actions, one protected invoice boundary

The [integrated action contract](integrated-action-contract.md) supersedes the
exact-path/query and illustrative malformed-form examples in the earlier
[comparison packet](action-contract-design.md). The HTML action uses typed
captures, keyed `form::rows` and a separate structural-rejection renderer. A
second source JSON POST action uses codec-admissible transparent row records and a
typed browser Fetch request/response; both saves call one protected write operation.
A third bodyless JSON GET action returns an authorized snapshot for initial load
and explicit reread. Three fresh [Jev consultations](jev-json-action/findings.md)
favored the HTML/JSON save split, and [three more](jev-grid-read/findings.md)
favored the typed GET read. The form-only/JSON codec boundary independently requires distinct wire
representations. The protected operation checks server-derived actor, tenant,
invoice, membership and revision at write time. Grid mutation IDs have a
durable same-payload replay contract for uncertain commits. The actions keep
actual 200/422/409/403/503 save statuses, with visible HTML swaps and typed
JSON results respectively; GET has 200/403/503. This is planned design, not an
implemented endpoint.
