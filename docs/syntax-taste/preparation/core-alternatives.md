# Core language alternatives for design preparation

24 September 2026 · P4 comparison packet. This records alternatives at the
time of comparison; later selections are in
[accepted technical decisions](accepted-technical-decisions.md) and
[confirmed syntax choices](confirmed-syntax-choices.md).

This packet covers DI-01–DI-08 in the [decision inventory](decision-inventory.md).
It uses the [core evidence](core-evidence.md),
[package/assertion evidence](packages-assertions-evidence.md),
[constraints](constraints.md), and [evaluation protocol](evaluation-protocol.md).
Those packets distinguish current source inspection, current focused tests,
historical executed probes, and unexecuted deductions. This document adds no new
execution evidence. Existing decisions remain authoritative. No alternatives
have been selected. This packet's author made no Jev consultation; the separate
[pattern-intent consultation](jev-pattern/findings.md) reports three fresh
requests with low-confidence disagreement and selects no rule.

The audience is AI coding agents. Correct behavior, reliable edits, explicit
contracts, and independent composition lead the comparison. Total tokens per
successful task is a secondary measured outcome; all cost claims below are
hypotheses until trials include context, diagnostics, retries, and validation.
Human familiarity and source length are not acceptance criteria. There is no
compatibility requirement for existing syntax, generated layouts, or goldens.

All illustrative contracts below are **semantic pseudocode**, not parseable Can
proposals, unless explicitly identified as an existing expression or identifier.
They intentionally avoid offering several spellings of one unresolved contract.
The candidate names are local comparison labels, not proposed language terms.

## DI-01 — Pattern intent and typo diagnostics

**Strongest current idiom.** Use qualified or explicitly specialized nominal
leaf patterns where admitted, or checked constructor patterns such as the
historically rejected `decliend()`, and reserve intentional whole-value binding
for cases that truly need it. The qualified/specialized lookup path diagnoses an
unknown leaf; a discipline or lint can require that path. Current plain bare
`decliend` still binds the whole value. Thus the strongest disciplined idiom
avoids the known failure, but current language admission does not enforce the
discipline. Named leaf records, a closed variant, and narrowing the scrutinee
inside each arm remain useful baseline behavior.

**Shared candidate requirement.** Every alternative below rejects a final bare
`decliend` arm intended as `declined`, even when no later arm exists. Adding
`refunded` then produces a missing-case diagnostic on a corrected three-case
match. None interprets an unknown bare identifier as a binder at an expected
variant case position. Explicitly
requested catch-alls can cover future leaves; they must be visibly different
to the checker and to an agent inspecting the contract. Spelling suggestions
are diagnostic assistance, never the source of pattern meaning.

| Alternative | Complete matching contract | Agent tradeoffs and native lowering |
| --- | --- | --- |
| P1: resolved cases with explicit bindings | Bare names in case positions must resolve to admitted nominal leaves; qualification and specialization use the same rule. A distinct binding construct introduces a whole-value or nested field binding. A wildcard deliberately discards any value. Leaf recognition precedes no fallback-to-binding rule. Nested nominal tests, field binders, literal tests, and wildcard coverage use the same separation. | A case typo and a missing binding marker produce local diagnostics instead of behavior changes. Agents must learn one explicit introduction rule. Existing leaf tag tests and local JS bindings suffice; the new distinction is chiefly checking and coverage. |
| P2: constructor patterns with explicit bindings | Every nominal case test is a constructor-pattern node, including a zero-field leaf. Bare identifiers are illegal pattern nodes; explicit binding nodes and wildcard nodes remain available at any valid nesting position. Constructors resolve to exact leaf identities and validate field arity and nested patterns. | Constructor arity makes representation changes visible, but agents may need edits for field changes that P1 could ignore through scrutinee narrowing. A bare final typo is an invalid pattern; a typo in a constructor node is an unknown constructor. Lower to native field/tag tests and bindings, with no runtime constructor call for testing. |
| P3: closed case dispatch separated from destructuring | A variant dispatch accepts only resolved case identities plus a separately designated default branch. Branches use the original narrowed scrutinee; field extraction and renaming are ordinary bindings inside the branch. General record destructuring is a separate construct, with explicit binder introduction and no implicit case lookup. Nested case discrimination uses another closed dispatch. | Case tables become mechanically enumerable and never conceal a binder. Deep patterns require more statements/dispatches and may increase total repair cost. Native switches/conditionals and local reads implement the behavior. No new matcher runtime is necessary. |
| P4: expected-type-sensitive binding | At every pattern position whose expected type is a closed variant, bare names must resolve to admitted leaves; whole-value capture is explicit and wildcard remains available. At positions proved non-variant, bare names remain binders under the retained capture rules. If a generic expected type is unresolved, a bare name cannot be accepted as a binder until its non-variant status is established; an explicit binder is always unambiguous. The expected-type rule applies recursively to arrays and record fields, not just the outer arm. | Changes fewer existing captures, but identifier intent can change when a nested field's type changes. Agents need expected-type-aware diagnostics and refactor checks. Native matching and bindings are the same as P1; the main difference is contextual checking. |

**Representative counterexample.** Let `payment = paid | pending | declined`.
Match `paid`, `pending`, and final bare `decliend`; each candidate must diagnose
that final arm. For P2 also try its constructor-shaped typo. Correct the typo,
extend to `payment = paid | pending | declined | refunded`, and require a
non-exhaustive diagnostic. Repeat with a misspelled nested leaf. Separately
prove that an explicit whole-value binder/default deliberately accepts all
remaining leaves and cannot be confused with a case reference.

**Nested expected-type refactor counterexample.** Initially an array element or
record field has type `str`, and its pattern captures a bare name. Change that
field to `payment` without editing the pattern. P1/P2 require explicit binding
from the outset; P3 binds fields in statements separate from case dispatch; P4
must now reject the former binder as an unknown variant leaf unless explicitly
converted to a capture. Reverse the edit using a bare name that previously
resolved to `declined`: P4 can now turn that name into a binder at a non-variant
position. Test the resulting behavior change and diagnostic policy instead of
assuming contextual admission is invariant under type edits. Repeat under a
generic specialization and with a same-named field/leaf. The fresh Jev ranking
disagreement makes this a discriminating experiment, not a reason to discard
either uniform or contextual intent rules.

**Interactions.** The case universe and nominal leaf resolution depend on DI-07
and DI-05a. DI-02 may hide fields while allowing public nominal discrimination;
constructor-pattern arity must not accidentally expose an owner's private
representation. Generic error case discrimination must keep exact specialized
identity under DI-03 and DI-05b.

**Evidence gaps and syntax choice.** Current evidence establishes the unsafe
fallback, not comparative agent performance on nested patterns or opaque leaves.
P1–P4 need held-out rename/add-leaf tasks and diagnostics. Retaining the current
language plus mandatory compiler lint is a baseline trial, not an alternative
that permits the typo silently. Any admitted syntax change requires a later
user choice; the binding/case contract must be settled before choosing tokens.

## DI-02 — Package-owned validated values

**Strongest current idiom.** Keep a transparent nominal immutable record, an
explicit validating factory, and validation at every ingress and sensitive
operation. Decode a wire record first and call the factory. This can deliver
correct application behavior, but an importer can still construct or `with`
an invalid record, and generic decoding bypasses the factory. A private record
cannot currently appear in a public signature. Do not describe the baseline
as guaranteeing that every value inhabiting the nominal type is valid.

The test domain is an email, a positive quantity, and a tenant identifier. A
valid tenant identifier still confers no authorization: every tenant-sensitive
operation checks the current actor and context.

| Alternative | Construction, observation, and mutation | Decoding, equality, assertions, and lowering |
| --- | --- | --- |
| V1: owner-controlled record construction | A type can be public while direct construction and representation updates are restricted to its declaring package. Public projections may expose selected immutable fields; matching can discriminate its identity but cannot bind hidden fields. Only owner functions mint or update it, and the owner is responsible for validating all exported creation paths. | Generic decoding of the protected type is rejected; callers decode a public wire shape and invoke an owner factory. Encoding uses an explicit owner projection. Structural field equality is available only through an owner operation unless the owner deliberately exposes a fixed equality contract. Assertions obtain values through owner factories or owner-published validated constants; fixture admission cannot mint hidden records. Frozen nominal JS records can remain the representation; visibility is enforced at compile time. |
| V2: opaque authored value | A public nominal handle hides the entire private representation. Only the owner wraps/unwraps it. All observation, matching beyond identity, updates, and equality occur through explicitly exported owner operations; consumers cannot destructure it or use `with`. Public variants may still contain the opaque leaf if leaf testing does not reveal representation. | No derived codec/schema is admitted for the opaque value itself; owner encode/decode functions specify a separate wire contract and errors. Assertions compare a published observable projection or an owner-defined equality result, and cannot fabricate handles. JS may erase a wrapper if nominal identity cannot be lost through admitted operations; otherwise a frozen branded wrapper is the minimal adapter. This erasure question needs evidence before implementation. |
| V3: checked nominal construction | A public record declares one owner validation operation over its representation, with a finite error bound. Every admitted creation path, including initial construction and copy-update, becomes a checked fallible construction through that operation. Validation receives the representation without recursively constructing the target type; it must return a validated representation or reject. Fields remain publicly observable and nominal patterns remain available. | Generic decode first checks wire structure, then invokes the declared validator, propagating structural and domain failures through an explicit decoder contract. Schema advertises structural shape without claiming that shape expresses the entire predicate. Equality is the normal nominal field comparison after validation/canonicalization. Fixture-supplied completions must use the same checked construction path. Lower to native field operations and a call to the authored validator, then the existing immutable-record adapter. |

V1 keeps a record-like public observation surface with package-owned minting;
V2 treats representation and observation as an abstraction boundary; V3 makes
all public construction a validation operation. These are different contracts,
not keyword variants. V1/V2 trust the owner package's construction code; V3
centralizes the predicate but must define validator recursion, side effects,
normalization, and finite failures. None proves arbitrary predicates correct.

**Agent tradeoffs.** V1 minimizes special call-site rules but needs precise
visibility diagnostics and owner review. V2 gives agents the smallest legal
consumer surface at the cost of more API calls and fixture adapters. V3 makes
ordinary construction safe by admission, but adds fallibility to construction,
`with`, and codec signatures and may complicate sequencing. Compare successful
cross-package edits rather than assuming fewer accessors or shorter calls win.

**Acceptance counterexample.** From an independent importer, attempt direct
`email` construction with invalid text, `with` replacement to invalid text,
generic decode of invalid text, and a fixture-supplied invalid completion. V1/V2
must reject every unauthorized creation route before it manufactures the value;
V3 must produce the declared validation failure through each admitted route.
Then create a valid tenant identifier and use it for another tenant's invoice:
the operation must still reject access. Test canonicalized emails and equality
to reveal whether equality leaks hidden representation or bypasses validation.

**Interactions and gaps.** Package identity owns the boundary (DI-05a); codecs,
fixtures (DI-06), patterns (DI-01), variants (DI-07), and error bounds (DI-03)
must honor it. Current source demonstrates bypass mechanisms, but the exported
email program and agent comparisons have not been executed. V3 additionally
needs evidence that one validator can serve all paths without introducing a
general effect system. V1–V3 all require user review of authored declaration
and use syntax after the contract choice; no keyword is selected here.

## DI-03 — Finite higher-order error preservation

**Strongest current idiom.** First use `match chain` and named functions with
written concrete `emits` bounds. For reusable helpers, compare a fixed bound
with per-domain wrappers that explicitly relay cases. Include a nominal
result-as-data helper as a benchmark, while recording that the evidence packet
found no maintained example establishing its two-caller cost. Built-in array
callback propagation already supports specialized finite errors; that is not
evidence that arbitrary authored error polymorphism is cheap or necessary.

| Alternative | Complete public contract | Agent tradeoffs and lowering |
| --- | --- | --- |
| E1: retain concrete bounds and adapt by domain | Every callable parameter and outward function bound lists concrete named errors. A helper either chooses an explicit common domain bound or callers use separate wrappers. It handles or relays every admitted error; it cannot pretend that a broad bound specializes to a smaller caller set. `match chain` remains the local sequencing baseline. | Public failures stay inspectable with existing tools. Unrelated domains may require repetition or broad outward bounds, which must count in task cost. Existing callable invocation and error propagation lower to the current native JS call/async behavior plus required Can failure adapters. |
| E2: finite error-set parameter | A written parameter ranges over finite sets of nominal, fully specialized domain errors. A callback consumes that set and the helper explicitly declares the same set, or its union with named helper errors, outward. Specialization substitutes and normalizes the set; it may instantiate to empty. A helper can propagate an unknown member but cannot inspect fields without narrowing to a named error. Escaped errors and missing outward bounds diagnose. Public signatures never infer an unwritten error parameter. | One reusable helper can preserve each caller's exact bound. Agents gain reusable contracts but must maintain set parameters/unions and interpret substitution diagnostics. Parameters erase at runtime; native calls and existing error objects remain. No scheduler, general effect row, or extra exception transport follows. |
| E3: failures as explicit nominal data | A helper returns a generic closed success/failure data value, with a failure payload type supplied by the caller. Callbacks use that same data contract and are infallible in Can's domain-error channel, except for separately written fixed helper errors. Named boundary adapters turn admitted named errors into payload leaves and back where required; the conversion cannot silently widen or drop cases. | Generic value parameters express reuse with current concepts, but adapters and two failure channels can increase agent mistakes. Exhaustive result matching makes conversion losses visible. Lower to ordinary frozen tagged data and native calls; account for allocations if current runtime contracts require them. |

**Representative contract and counterexample.** Use one `with_audit` helper with
an invoice callback that emits `invoice_missing | invoice_locked` and a payment
callback that emits `payment_declined`. Audit itself can fail with `audit_down`.
The first caller must not acquire `payment_declined`, and the second must not
acquire either invoice error merely because another caller uses the helper.
E1 must expose its broader bound honestly or use separate wrappers; E2 must
derive exactly each callback set union `audit_down`; E3 must preserve those
distinctions in payload types and conversions. Add a callback error, an empty
set callback, and a specialized generic error to the held-out tests. A `retry`
variant must additionally specify which failures are retryable and final-attempt
behavior; error polymorphism alone cannot make retrying every error correct.

**Interactions, gaps, and syntax choice.** DI-04 operation dictionaries can
contain fallible callables; their bounds must follow the chosen model. DI-05b
must distinguish error identity from reporting IDs, and DI-07 determines
result-as-data leaf compatibility. DI-02/V3 increases the importance of explicit
constructor failures. The two-domain helper benchmark and agent measurements
are missing. E1/E3 may be evaluated without language syntax changes. E2 reopens
LD14 and requires user choice on error-set declaration/application syntax only
after demonstrated benefit; it does not authorize a general effect system.

## DI-04 — Public generic operation requirements

**Strongest current idiom.** Pass the needed named callable explicitly; bundle
multiple operations into a dictionary record when reuse justifies it. A generic
body uses only those supplied operations and universally admitted operations.
Current concrete specialization still checks the whole body. Documentation can
describe local template semantics, but a public signature that hides a new `+`
requirement is weaker than this explicit baseline.

| Alternative | Public contract and checking | Agent tradeoffs and lowering |
| --- | --- | --- |
| G1: explicit callable/dictionary inputs | Generic signatures name operation values and their exact input/output/error types. Public-body checking verifies that all non-universal operations flow through those declared inputs; local template helpers may remain specialization checked under an explicitly documented separate policy. Dictionary fields are ordinary named members, not an implicit global instance search. | Agents can discover requirements from the signature and repair missing operations locally. Passing dictionaries may cost context and code. Native JS function calls/property reads suffice; compiler-known constant inputs may be optimized only without changing callable semantics. The stronger public-body restriction would itself be a semantic change from today's admission. |
| G2: small compiler-defined capabilities | Public generic parameters name a finite supported capability vocabulary, each defining exact permitted operations and laws only where the implementation actually enforces them. Generic bodies are checked using those capabilities. Instantiation proves the concrete type meets them; a new body requirement requires a signature change. No user-defined operator implementation, implicit coherence search, or higher-kinded type follows. | Public operator use is concise and requirements are visible, but agents must know the vocabulary and overload semantics. Native numbers/strings/collections use equivalent JS operations with existing Can contract adapters. Unsupported domain operations still need explicit callables. |
| G3: public templates with generated requirement artifacts | Retain specialization semantics and permit hidden body requirements, but publish a compiler-generated requirement summary tied to the dependency lock/body digest. Every known consumer specialization is checked; a dependency change invalidates the summary and requires consumer rechecking. Diagnostics include the dependency body requirement and instantiation path. The summary is advisory for unknown specializations, not a claim of independently checked universal correctness. | Requires less new source syntax, but agents need the artifact/body context and errors may appear late. It deliberately accepts that unchanged written signatures can break consumers. Lowering is current specialization/native operations; artifacts add tooling and lock work, not a runtime trait system. |

**Acceptance counterexample.** A library changes `doubled<T>` from returning
its argument to evaluating `value + value` without changing its written
signature. A consumer previously instantiated it with a type lacking addition.
G1 must reject the public body unless it adds an operation parameter; G2 must
require a capability declaration change. G3 must invalidate the published
requirement artifact and diagnose the consumer specialization on rebuild; it
cannot claim that the unchanged source signature fully described the contract.
Repeat with two unrelated operations and a fallible dictionary callback.

**Interactions and gaps.** DI-03 controls operation errors; DI-02 determines
whether capabilities may inspect protected fields; DI-05 controls stable
dependency and artifact identity. There is no maintained two-operation,
multi-package dictionary benchmark. The boundary between universally admitted
operations and concrete-only operations needs enumeration. G1 can first test
current syntax as a discipline before any enforcement change; G2 needs user
syntax choice. G3 may need only artifact/tooling decisions, but its deliberate
weaker public contract still needs an explicit technical disposition.

## DI-05a — Independent package identity

**Strongest current idiom.** Give every resolved package a globally unique
short name and use file-local `uses ... as ...` aliases for convenient local
references. That works for coordinated authors but fails the required independent
composition case before aliases can resolve. Internal project-qualified IDs
and hashed output paths already exist; changing graph lookup alone would leave
source selection ambiguous.

| Alternative | Name resolution and identity contract | Agent tradeoffs and native lowering |
| --- | --- | --- |
| N1: explicit dependency-qualified imports | Every nonlocal import identifies a direct dependency key and its package name, with an optional file-local alias. Local packages resolve in the current project; catalogue imports use a reserved namespace. Inside a dependency, its imports resolve relative to its own dependency table, never the root's alias table. Two loaded instances have distinct graph identities even if short names coincide. | Agents receive exact unknown-dependency/package diagnostics and do not rename vendor source. More qualified imports cost tokens but can reduce ambiguity/repair. Emit native module imports using qualified graph identity/output mapping. |
| N2: manifest-bound import handles | A project's manifest maps unique local handles to a direct dependency plus package; source imports use those handles and may alias them per file. Dependency manifests own their own mappings. Missing/ambiguous mappings fail at graph loading; locks record the resolved targets. Catalogue handles cannot be shadowed. Canonical identity remains the resolved package instance, not the local handle. | Source stays compact and bulk remapping is centralized, but agents must consult another file and distinguish handle rename from identity change. Native module import lowering is the same as N1; no runtime resolver is needed. |
| N3: compiler-enforced globally unique package declarations | Require declarations to contain an author/project-qualified package identity and retain local import aliases. Different dependency versions/instances still receive distinct resolved identities; declaration identity alone cannot merge conflicting instances. Reject independently published packages that claim the exact same qualified declaration identity unless an explicit dependency-resolution rule selects one. | Can compose duplicate terminal short names only when their authors selected distinct full names. This imposes author naming coordination and does not satisfy a test requiring unchanged existing bare package declarations. It may reduce import-map indirection, but is a weaker fit for the stated unchanged-library benchmark. Lower to native module imports keyed by the resolved identity. |

**Acceptance counterexample.** Root consumes two independent dependencies each
declaring `model`, uses both in one file, and imports neither dependency's private
package. N1/N2 must compose without editing either library; N3 must state its
additional authored-identity precondition and cannot be credited as passing that
unchanged-library case. Rename only root-local aliases, introduce a transitive
dependency with another `model`, and relocate the checkout: emitted references,
nominal identities, and lock results must remain deterministic under the chosen
identity contract. An undeclared transitive import must remain rejected.

**Interactions, gaps, and syntax choice.** DI-05b IDs, DI-02 owners, DI-06 root
and seam identities, source maps, and DI-07 nominal leaves use this identity.
Package naming and dependency-version identity are separate questions. Current
tests cover pieces, not the full independently authored graph or alias/lock
refactor. N1/N3 need source syntax choice; N2 primarily needs a manifest contract
and may preserve the existing import grammar. Reserved catalogue names and
canonical `internal` visibility must survive every candidate.

## DI-05b — Application error identity and reporting codes

**Strongest current idiom.** Allocate globally unique application numbers,
maintain active/retired entries in `can.errors.json`, and lock their snapshots.
Nominal identity already uses the declaring package/declaration plus generic
arguments; the integer is a diagnostic code, not a sufficient nominal type.
Manual coordination is workable in one organization but does not compose
independently allocated duplicates. All loader, checker, registry, lock, emitted
plan, runtime admission, and reporting consumers need one consistent revision.

| Alternative | Canonical/report identity and retirement | Agent tradeoffs and lowering |
| --- | --- | --- |
| I1: scoped authored diagnostic numbers | Preserve authored numeric codes and registry retirement within the owning project identity. The report key is the pair of canonical owner identity and local number; nominal matching still uses declaration identity and concrete type arguments. Retired codes prevent owner-local reuse only. Locks record the scoped registry. | Minimal conceptual change for allocating codes, but every consumer must accept/report the scope with the number. Agents cannot identify an error from a naked integer. JS error/failure metadata carries the pair; use native maps keyed by an unambiguous canonical serialization. |
| I2: generated build-local numeric codes plus canonical keys | Source errors have canonical declaration identities; build generation assigns compact unique numeric report codes deterministically from the complete locked graph and emits a reverse map. Retired declaration identities can be tombstoned for reporting history, but no author reserves graph-wide numbers. A report must include the build/manifest identity because adding dependencies may renumber codes. Generic specializations share a declaration code while carrying concrete arguments separately. | Removes allocation collisions and duplicate source/registry numbers, but agents and support tooling must retrieve the exact reverse map for a report. Native integers/maps implement lookup; archive the map with release artifacts. This does not promise cross-build numeric stability. |
| I3: canonical textual report keys | Reports carry the qualified declaration key directly, plus concrete generic arguments when relevant. A compact numeric code, if a consumer requires it, is optional build-local metadata with an explicit map, never canonical identity. Retirement applies to stable declaration keys, and rename/reuse policy is recorded in owner metadata. | Self-identifying reports reduce map lookup but may increase output tokens. Agents must distinguish renaming a declaration from changing its identity. JS strings and ordinary error metadata suffice; no custom global allocator is needed. |

**Acceptance counterexample.** Two unchanged dependencies allocate the same
active number, or one allocates a number the other retired. Both must load under
I1; I2/I3 must specify the source/registry migration needed by their new contract
and then compose independent declarations without coordination. Inject each
error, confirm exact matching and distinct diagnostics, and interpret reports
from two builds after a dependency is added. A generic error instantiated twice
must not accidentally become one nominal type because its declaration report
code is shared. A hash by itself is not a collision-free numeric allocation
contract; any derived compact scheme needs collision detection/resolution and
an explicit stability scope.

**Interactions, gaps, and syntax choice.** DI-05a supplies canonical owners;
DI-03 substitutes generic errors; assertions and runtime failure plans consume
the same identity. The evidence audits key current consumers, but external
reporting requirements and cross-release identity/rename requirements remain
unestablished. I1 may keep source syntax while changing report shape. I2/I3 may
remove or change authored IDs and therefore need user syntax choice. Do not
delete numbers merely because no compatibility obligation exists; first decide
which stable machine-readable contract real consumers need.

## DI-06 — Cross-package fixture scenario ownership

**Strongest current idiom.** Use inert exact-target templates expanded into a
lexical `when`, with checked arguments and root-local FIFO queues. For a caller
unit test, stub the entire helper at the caller's lexical call site and separately
exercise the helper under its own attached assertions. This avoids hidden
cross-package label dependence at the cost of losing helper-body integration
coverage on that caller path. Merely qualifying the current root label does not
make caller-to-helper activation an explicit owned relationship.

All candidates retain attached assertion roots, exact target/argument checking,
full root invocation identity, lexical table identity, FIFO occurrence, and
reservation paths for recursion/callables/concurrent participants. Fixture
content is inert until deliberately attached. They must not search arguments
to allocate a queue occurrence or weaken evidence labels for supplied values.

| Alternative | Ownership, selection, and failure contract | Agent tradeoffs and native lowering |
| --- | --- | --- |
| F1: lexical-only fixtures plus whole-helper stubs | Select a lexical fixture only from assertions owned by the same declared lexical test scope; remove implicit activation by a foreign root's short label. Callers can stub the helper's completion at their call site; helper roots test its internals. Templates reuse content but grant no activation privilege. Cross-package integration tests execute real helper internals unless the API exposes ordinary injected dependencies. | Lowest new linking machinery, with an explicit integration-coverage limitation. Agents must not mistake separate unit tests for one end-to-end fixture path. Existing assertion context/queues can enforce owner-scoped selection; real execution stays native. |
| F2: explicit exported helper scenarios | A helper exports a named scenario describing its own lexical fixture configuration. A caller assertion explicitly links a particular helper invocation site to that scenario. The link identifies the callee declaration and scenario, not the caller's display label. Nested helper calls require declared scenario links; recursion/participants allocate isolated invocation paths under the linked scenario. Missing, inaccessible, ambiguous, unused, and signature-incompatible links diagnose. | Agents can reuse a coherent helper-owned simulation without knowing every internal call site. Scenario refactors affect published scenario contracts; internal changes can remain local if the scenario still validates. Lower links to compile-time plan entries and existing per-invocation context routing, without ambient name matching. |
| F3: caller-selected exported seams | A helper explicitly exports typed symbolic seams at approved lexical calls. A caller assertion binds fixture content to a named seam along a checked call path. The helper controls seam existence/target signature, while the root controls supplied behavior. Nesting and recursion use the call path plus runtime invocation identity; each participant owns its FIFO queue. Unused required bindings and changed seam signatures fail rather than falling back to real calls. | Gives agents precise integration control but exposes more helper structure and more refactor obligations. Avoid an unbounded global override registry: only declared seams along checked paths are eligible. Compile plans to existing context/queue routing with stable seam metadata; native real calls execute when no fixture is deliberately requested. |

**Acceptance counterexample.** Rename caller assertion `customer` to `renamed`
without altering the helper. Under F2/F3 the helper still produces the explicitly
selected fixture completion; under F1 no foreign label ever selected that row,
so behavior remains consistently real or whole-helper-stubbed before/after the
rename. This distinction is essential: F1 removes the fragile link but does not
deliver the same integration coverage as F2/F3. Then duplicate assertion labels
in two roots, invoke the same helper twice, recurse, and invoke it concurrently.
Verify independent queues, exact arguments, deterministic FIFO allocation,
unused fixture errors, and no silent real-call fallback after a declared
scenario/seam disappears.

**Interactions, gaps, and syntax choice.** DI-05a supplies root/callee identity;
DI-02 prevents fixture fabrication of protected values; DI-03/05b determine
supplied failure typing. Existing historical rename evidence demonstrates the
problem, but there is no candidate implementation or measured comparison
against whole-helper stubs and ordinary injected callables. F1 needs a semantic
restriction but may reuse source forms. F2/F3 reopen LD28 and need later user
choice for exported references, attachment, occurrence policy, and diagnostics.

## DI-07 — Variant specialization and leaf compatibility

**Strongest current idiom.** Use nominal record leaves and flattened closed
variants; construct a leaf, assign it into a compatible variant, and narrow by
matching. For a real abstraction boundary today, put the value inside a nominal
record and control operations as far as current visibility permits. Do not
rely on a phantom generic variant parameter to prove durable branding: current
direct rejection can be bypassed through a different variant with the same
leaf set. Generic records have their own specialized nominal identity and are
not shown broken by that variant probe.

| Alternative | Assignment, matching, and representation contract | Agent tradeoffs and codecs/lowering |
| --- | --- | --- |
| T1: extensional closed leaf sets | Variant compatibility is set inclusion over fully specialized nominal leaves. Remove the exceptional direct-specialization guard when both sets contain exactly the same leaves. Phantom parameters confer no identity beyond their resulting leaves. Nominal record leaves retain identity; `some<int>` and `some<str>` remain distinct, while the common `none` leaf belongs to both ordinary option variants. Matching observes leaves only. | A uniform relation supports predictable direct/bridge reasoning, but agents cannot use a variant name as an opaque brand. Existing flattened JS records/tag tests and codec leaf unions fit; variant names need no runtime wrapper. |
| T2: explicit retention of current path-sensitive admission | Keep the direct check rejecting different specializations of the same generic variant before applying ordinary leaf-set inclusion for other sources. Document assignment as this two-stage rule, not a transitive subtype relation or durable brand. Shared leaves and bridge variants remain legal. Matching observes the underlying leaf; no provenance is tracked. | Small semantic change, but agents must reason about the expression's immediate static type and may experience refactor-dependent assignment results. Diagnostics must mention the direct-specialization restriction and cannot claim all alternate paths are illegal. Existing flattened lowering and codecs remain. |
| T3: nominal variant wrappers | Each variant declaration and specialization has a distinct identity and a representation containing one admitted leaf. Construction explicitly injects a leaf into that variant; matching unwraps it under controlled narrowing. Variant-to-variant conversion requires explicit exhaustive mapping/injection. Generic specialization identity cannot be erased by assigning its wrapper to a shared leaf type. An explicitly exported leaf can be reinjected into another variant, but this is a visible conversion, not implicit bridge compatibility. | Agents get nominal guarantees with visible conversion sites but must write more wrappers/adapters and maintain codecs. JS uses a frozen nominal tagged wrapper unless analysis proves erasure preserves identity. Codec schemas must encode or contextually restore the exact wrapper identity; wire leaf shape alone cannot select between wrappers. |

**Acceptance counterexample.** Test `tagged<int> -> tagged<str>`,
`tagged<int> -> bridge -> tagged<str>`, and passage through a shared `unit` leaf.
For T1 all equal-leaf paths are admitted. For T2 only the direct specialized
assignment is rejected, and the documented matrix must say so. For T3 implicit
cross-wrapper paths fail, while deliberate unwrap/reinject operations are
tested according to visibility. Also test ordinary `option<int>`/`option<str>`,
`none`, `some<int>`, nested variants, generic nominal record leaves, codec
roundtrips, and a leaf-addition edit. No candidate may silently change the
identity of a generic record to make its variant tests pass.

**Interactions, gaps, and syntax choice.** DI-01 exhaustiveness depends on the
selected leaf/wrapper universe, DI-02 may prohibit unwrapping, DI-03 result data
uses closed variants, and DI-05 supplies nominal identity. Current tests and
historical probes establish the matrix for T2 but do not establish comparative
agent edit cost. T1 needs compatibility and codec regression probes, T3 needs
representation and allocation measurements. T1/T2 can be specified without a
new authored surface, though examples/docs change. T3 requires user choice for
injection/matching/conversion syntax. Keeping current behavior is a candidate,
not evidence that the surprising assignment matrix is desirable.

## DI-08 — `near` capture and refactor identity

**Strongest current idiom.** A named function takes an explicit immutable
context record (or ordinary arguments), and its callable reference captures
only a receiver where the current method rule requires it. This makes the
selected request/actor values visible at the call site. Current `near` is a
shorter alternative: it resolves each captured parameter by the callee's exact
parameter name and type at the reference site. A callee rename may fail, while
a same-typed new local can silently become the captured value
([source evidence](core-evidence.md#captures-and-construction)).

| Alternative | Capture contract | Agent and lowering consequences |
| --- | --- | --- |
| C1: explicit context data | Keep capture composition in ordinary parameters and immutable context records. Avoid `near` in a service flow where identity matters, or retain it only as an opt-in convenience with its documented name lookup. | Existing native closure values and record reads suffice. Agents write more explicit arguments but can inspect exactly which actor/tenant context is passed. Test whether this idiom stays manageable after handler extraction and rename. |
| C2: explicit capture map | A callable reference states each callee captured parameter and the exact source expression supplying it. The checker validates names and types; a callee parameter rename diagnoses the map, and adding a same-typed local cannot redirect it. Captures are evaluated once when the callable is made and retain immutable values. | More syntax and checker work, but no new runtime ownership mechanism: emit a native closure with the checked values and current Can owner/callable adapters. An agent can repair a rename at the reference site. |
| C3: declaration-bound capture identity | Captured parameters acquire stable compiler identities distinct from their display names. A reference binds to those identities, with explicit source expressions or a generated binding record. A display-name rename need not alter selection, while a new parameter is a new identity and must be supplied. | Potentially strongest rename stability, but stable IDs or generated records add source/manifest/tooling complexity. Agents must understand identity changes and exported callable evolution. No evidence yet establishes a benefit over C2's simpler explicit map. |

**Discriminator.** In the tenant invoice example, create `request_context`
and `audit_context` with the same type, reference a handler, then rename the
callee's captured parameter and introduce a same-typed local with the new
name. Observe which value the closure retains, compiler diagnostics and
agent repair. Repeat inside a scoped transaction and after its owner closes;
explicit capture selection must not bypass resource lifetime or make a tenant
identifier an authorization proof. Compare C1 before adding syntax. If C2 or
C3 earns adoption, the user chooses the complete callable-reference and
capture-list syntax; no spelling is selected here.

## Shared experiment boundary and decision dependencies

These comparisons should use two independently authored libraries and one
consumer, including a held-out naming/data variant. Record current and candidate
source revision, exact model/effort, supplied context, trial count, compiler
attempts, diagnostic repairs, behavior tests, and total task tokens before
claiming an agent benefit. Each benchmark includes creation, rename/refactor,
and repair. A passing hand-written example or Jev agreement cannot substitute
for negative-case execution and comparable agent trials.

| Dependency | Why it matters before specification or implementation |
| --- | --- |
| DI-05a identity → DI-05b, DI-02, DI-06, DI-07 | Error owners, protected types, fixture links, and nominal leaves need one coherent qualified package identity. Trial syntax can proceed provisionally, but cannot finalize divergent identity schemes. |
| DI-07 variants ↔ DI-01 patterns | Leaf-set/wrapper semantics determine case coverage and legal narrowing; typo rejection is required under every variant candidate and need not await wrapper adoption. |
| DI-02 values ↔ codecs and DI-06 fixtures | Constructor restrictions are ineffective if decode or supplied completions can mint unchecked values. The owner/validator policy must cover them together. |
| DI-03 errors ↔ DI-04 operations and DI-05b reports | Callable dictionaries and validation operations need precise finite failures; reports must preserve declaration/specialization distinctions without conflating them with numbers. |
| DI-06 fixtures → experiment credibility | Caller-label-independent selection and evidence labels are necessary to know which implementation paths comparative tests actually executed. |

The table is a design dependency map, not an instruction to implement all topics
in that order. Independent negative-case prototypes can run before a final
choice. P4 leaves technical selections, three fresh Jev consultations where
required, later user syntax decisions, and comparative experiments outstanding.
