# Can language audit — master TODO

Current basis: [replacement audit and dispositions](workstream-a/README.md),
language/compiler baseline `8bbcf13`, reviewed 2026-09-19.
The [original audit](../can-language-audit.md), baseline `8312d85`, remains the
source of candidate redesigns, not a settled implementation plan.
Reproductions: [current findings and gates](workstream-a/README.md#current-findings-and-validation)
and [probe guide](README.md).

**Status:** Workstream A audit/review completed at `8bbcf13` on 2026-09-19;
[replacement dossier, judgments and dispositions](workstream-a/README.md).
Remaining workstreams are planning checklists. No language redesign or defect
repair has been implemented by this document. Audit recommendations are proposals,
not approved specifications. The original JEV reviews are withdrawn as decision support:
the supplied Can context was insufficient. Reproducible compiler findings remain
evidence independently of those reviews.

**Constraint:** zero external users. Preserve guarantees, not old syntax, ABI,
source spellings, generated TS layouts, or goldens. Do not create permanent
compatibility modes. Migrations still need explicit semantic decisions and tests.

## Overview

| Order | Workstream | Current status | Depends on |
|---|---|---|---|
| Done | A. Correct the JEV review process | Replacement review complete; design dispositions recorded | [Dossier and evidence](workstream-a/README.md) |
| Now | B. Confirmed correctness/integration defects | Reproduced at `8bbcf13`, including new F06; fixes pending | Can proceed without syntax redesign |
| Now / decision | C. Current specification and compiler foundations | C1 supported next work; C2/C3 architecture unresolved | Current inventory and concrete compiler pressure cases |
| Decision | D. Successes, calls, bindings, patterns | Success/call/context choices unresolved; other changes remain candidates | Explicit semantics; H before any public ABI migration |
| Repair / decision | E. Modules, ownership, effects, revisions | Strict duplicate rejection supported; namespace/visibility redesign unresolved | B2/B3; ownership and authority decisions |
| Decision / gates | F. Tests, evidence, contracts, faults | Pure-test default unresolved; existing guarantees and repair gates retained | B1/B5; chosen source and evidence semantics |
| Decision | G. Errors, data, higher-order functions | Error abstraction unresolved; expressivity extensions are candidates | Chosen error/effect/termination and type rules |
| Decision | H. Host and wire boundaries | Trust policy/public facade unresolved | Real embedding requirements; separate source/private/public/wire contracts |
| Design process | I. B05 async and resources | Contract-first comparison supported; source/host/backend choices unresolved | Real customer, settlement/resource requirements and acceptance vectors |
| Evidence / decision | J. Formatter, surface syntax, diagnostics | Collection/literal/boolean/label choices unresolved | Agent trials and explicit grammar/evaluation rules |
| Per decided slice | K. Stdlib migration and acceptance gates | Repairs and gates pending; redesign migration conditional | Relevant recorded decisions and working implementation |

Checkboxes are completion evidence, not approval. Mark a design task complete
when its decision and rationale are recorded; mark an implementation task
complete only when its acceptance gates pass. Record the implementation commit
and test evidence when closing an item. Keep baseline failures separate from
regressions introduced by a change.

## Decisions carried forward from the replacement audit

**Accepted** below means supported as an audit recommendation within its stated
scope, not approval of a new language or ABI. The review accepted three bounded
recommendations and left ten choices unresolved. JEV's highest-scoring option
alone does not choose a design; the [full distributions and reviewer rationale](workstream-a/README.md#replacement-jev-results-and-audit-disposition)
remain the evidence record.

| Reviewed question | Disposition | Consequence for this TODO |
|---|---|---|
| Sequencing | **Accepted:** repair and specify first | Prioritize B, C1 and working gates. Do not assume a new core must precede every semantic slice; decide C2/C3 from concrete needs. |
| Successes | **Unresolved** | D1 must compare declared-shape/B11, uniform whole-value and raw-total conventions; no removal or ABI migration is selected. |
| Calls | **Unresolved long-term** | D2 must compare split call/invoke, unified explicit call and bare application. Repair F06 independently; wider callable positions need authority/cycle analysis. |
| Modules | **Accepted only for immediate repair:** strict uniqueness | B2/B3 reject duplicate definitions and use one owned schema. Scoped/package namespaces and visibility remain open in E1. |
| Pure-source tests | **Unresolved** | F1 must compare scripted boundaries plus linked gates, executing pure source by default, and explicit test modes. |
| Host boundary | **Unresolved** | H2 must choose trusted embedding, a validated data-only facade, or a validated facade with callable provenance before imposing an ingress policy. |
| Error abstraction | **Unresolved** | G1 must compare concrete combinators, rows-first relay, and rows plus completed outcomes; neither `Outcome<T,E>` nor generic error rows are selected. |
| B05 process | **Accepted:** contracts before backend selection | I must resolve a real customer's source/host/resource obligations, then prototype only uncertainties needing measurement. No mandatory pair of full implementations. |
| Contextual typing | **Unresolved** | D5 must compare fully explicit, constructor/pattern-only context, and broader deterministic local context. |
| Collection spelling | **Unresolved** | J1 must compare `Seq<T>`, `[T]` and `T[]` independently of fixing compositional type rules. |
| Literals | **Unresolved** | J1 must compare current explicit families, ordinary exact decimals plus escaped strings, and a numeric-only change using edit/error evidence. |
| Boolean evaluation | **Unresolved as a permanent choice** | Keep current eager semantics during repairs; conditional evaluation needs explicit branch/fault rules and trials. |
| Argument labels | **Unresolved** | Repair F06 first; compare optional labels, canonical positional forms and required multi-argument labels separately from parameter binding. |

**Checklist interpretation:** B's confirmed repairs and C1's current inventory
are concrete next work. C2/C3 and the redesign parts of D–J describe decisions
and conditional implementation work: first choose the design, then implement
only the applicable branch. They do not require implementing every alternative
or carrying both old and new forms. Preserving current behavior while a decision
is open creates no permanent compatibility obligation. Additional proposals such
as pattern unification, local blocks, resources or tail-call lowering were not
independently approved by the thirteen judgments.

Do not carry forward these rejected inferences: scores approve a redesign;
current winners must remain forever; B05 differs only in its backend; TS types
establish host ownership/purity; or green Can rows/unchanged goldens establish
strict target validity or universal proof.

## A. Redo the JEV review correctly

Completed as an audit, not a redesign approval. All thirteen replacement
requests/responses, context, input checks, current probes and reviewer decisions
are preserved in [Workstream A](workstream-a/README.md). Three bounded
process/correctness recommendations are accepted; ten design choices remain
unresolved. No implementation commit applies: compiler behavior is unchanged.

- [x] Flag the original JEV reviews as withdrawn in the audit and probe index;
  retain the original request/response files as historical evidence.
- [x] Research Can ourselves and assemble a complete, self-contained context
  dossier. Do not ask JEV to research, inspect files, follow links, or fill gaps.
- [x] Include the language's goals, zero-user constraint, guarantees, non-goals,
  and the distinction between current rules and rules being reconsidered.
- [x] Include actual source examples and current behavior for values, success/
  error returns, calls, patterns, generics, function values, modules, effects,
  tests, contracts, revisions, recursion, and host boundaries.
- [x] Include relevant compiler rules and implementation evidence—not merely
  summaries of the preferred alternatives or repository paths.
- [x] Include current TS layouts, exact numeric representations, ownership/trust
  assumptions, runtime faults, and the separate source/private/public/wire layers.
- [x] Include stdlib customers, counterexamples, confirmed F01–F05 findings, and
  both competing B05 proposals. Label implemented, proposed, and historical facts.
- [x] Resolve documentation contradictions before presenting them as facts.
- [x] For each decision, supply balanced alternatives, semantic consequences,
  proof/authority obligations, migration costs, and concrete before/after examples.
- [x] Check input limits before submission. Every request must contain the
  context it needs, including the shared Can model; no assumed repository access,
  cross-request memory, or silent truncation. Narrow the question if necessary.
- [x] Re-run the eight architecture questions: sequencing, successes, calls,
  modules, pure-source tests, host boundary, error abstraction, and B05 ABI process.
- [x] Re-run the five surface questions: contextual typing, collection spelling,
  literals, boolean evaluation, and argument labels.
- [x] Preserve full supplied context, questions, alternatives, model/version,
  responses, probabilities, confidence, and any subsequent reviewer decision.
- [x] Review the answers against actual evidence; do not treat probabilities as
  soundness proofs, usability measurements, or user approval.
- [x] Record which recommendations are accepted, rejected, or unresolved before
  using them as implementation requirements.

## B. Repair confirmed defects and incomplete gates

These repairs do not need to wait for a new parser. Reproduce at current HEAD
before changing code; the current audit reproduced F01/F02/F04/F05 and added
F06 at `8bbcf13`. F03 is a confirmed trusted-boundary behavior/design gap, tracked
under H; the review did not classify it as a hostile-host sandbox breach.

### B1. Acceptance and canonicalization — F01

- [ ] Add a normal regression test: changing a pinned callback target together
  with the factory body must produce the acceptance-change warning.
- [ ] Encode function-reference target, revision/identity, captures, and relevant
  specialization information in canonical evidence.
- [ ] Include semantically relevant typed-Ok and pattern annotations.
- [ ] Include invocation arguments in canonical executable structure.
- [ ] Reject unsupported semantic node kinds instead of hashing an
  `unknown-kind(...)` placeholder.
- [ ] Separate interface, executable, dependency, and acceptance identities;
  audit which consumers use each canonical form.
- [ ] Version the canonical format and reject incomplete old evidence. Do not
  silently promote a regenerated candidate baseline to accepted authority.
- [ ] Test changed targets, captures, invocation arguments, type arguments,
  dependencies, and formatting-only edits.

**Done when:** semantic changes are distinguished, formatting is inert, and the
pinned-factory reproduction warns. Do not claim a proof-cache exploit from the
narrower structural probes alone.

### B2. Declaration identity — F02

Immediate scope: deterministic strict uniqueness in the current global model.
This repair does not require selecting scoped namespaces or visibility in E1.

- [ ] Reject duplicate record definitions at their declarations, including
  identical redeclarations that currently coalesce as well as conflicting shapes.
- [ ] Audit corresponding brand/error ownership and collision paths.
- [ ] Decide how genuinely shared definitions are imported; remove accidental
  coalescing/redeclaration and migrate affected stdlib sources.
- [ ] Test both source orders, identical and conflicting duplicate definitions,
  distinct global names and explicit shared imports. Resolution must not depend
  on file order; same short names in separate scopes depend on a later E1 decision.
- [ ] Make CLI, LSP, checker, revision machinery, and emitter agree on identity.

**Done when:** the duplicate-record probe cannot silently select int versus str
by reordering inputs, and migrations preserve explicit ownership.

### B3. Whole-stdlib composition — F04

- [ ] Resolve the incompatible `math.nonterminating_decimal` declarations in
  ratio and scalars: one owned schema or distinct error identities.
- [ ] Check all other duplicated error/type declarations for conflicting schemas.
- [ ] Add a required all-stdlib compilation gate and representative consumer
  subsets, not only per-module checks.
- [ ] Report conflicting schemas at their definitions rather than as a later
  unrelated payload mismatch.

**Done when:** all stdlib sources compose into one checked world without erasing
payload types or silently merging incompatible identities.

### B4. Generated TypeScript and packaging — F05

- [ ] Add strict TS checking of freshly emitted complete dependency bundles.
- [ ] Fix JSON's imported/local `Bool__Value`, `Dec__Value`, `Int__Value`, and
  `Str__Value` conflicts in concert with declaration ownership.
- [ ] Preserve literal discriminant types in nested sequence/record construction;
  fix the fresh JSON TS2322 failures without `any` or widened variant types.
- [ ] Fix repository TS-project artifact layout: relative host, quota, and
  scalars dependencies must be present or resolved by an explicit build layout.
- [ ] Execute the newly strict-checked bundles under Node and compare semantics.
- [ ] Distinguish source checked, examples passed, target emitted, target
  strict-checked, and target executed in build reports.

**Done when:** fresh JSON output passes both its 785 baseline Can rows and strict
TS; whole-library/consumer bundles compile and execute with their real dependencies.
A byte-identical but invalid golden is not a passing target gate.

### B5. Named-argument evaluation order — F06 (new in Workstream A)

Current-HEAD reproduction: [evaluator/target fault-priority probe](workstream-a/evidence-argument-order.txt).
Well-typed `xs=[]`, `index=0`, `divisor=0` produce different first primitive
faults for source `right = xs[index], left = 1 / divisor`: evaluator reports
sequence bounds, emitted TS reports division by zero. Non-faulting values agree.

- [ ] Record the intended argument evaluation order; reconcile the binder's
  source-order contract with the emitter's parameter-order expression lowering.
- [ ] Add an evaluator/target regression with two independently faulting
  expressions, reordered labels, and non-faulting controls.
- [ ] Evaluate each argument exactly once in the specified order, then place
  evaluated values into parameter slots; audit the other binding consumers.
- [ ] Preserve correct parameter binding and test both first-fault behavior and
  ordinary outcomes without relying on invalid host types or side effects.

**Done when:** evaluator and freshly emitted target agree on values and first
faults for reordered named arguments, with the chosen order documented.

## C. Establish the current specification and compiler foundations

C1 is supported next work. C2/C3 are candidate architecture changes, not a
selected prerequisite for every repair or semantic improvement.

### C1. Authoritative specification — C03

- [ ] Publish one current semantic specification and an implemented/proposed/
  historical/deferred feature inventory.
- [ ] Reconcile the root README, REQUIREMENTS, can-idioms, stdlib comments,
  contract-activation comments, and conflicting B01 examples/plans.
- [ ] Carry the dossier's factual corrections into the current specification:
  all three recursion schemas; bare-parameter-only invocation; active but limited
  contract verification; module `emits` unenforced; no historical revision store;
  and B10/B11 success support superseding old record-only/bare-Seq restrictions.
- [ ] Document which claims are checked, tested, universally verified, or trusted.
- [ ] State current evidence maturity: stdlib rows have no `requires`, `ensures`
  or `pinned` markers at the audit baseline; A-light needs an accepted baseline,
  records no pinner identity, and warns rather than blocking emission.
- [ ] Preserve historical documents as history with clear supersession links.
- [ ] Generate syntax/intrinsic/diagnostic inventories where practical.

### C2. Compositional types — T01

- [ ] Compare introducing a shared type representation first with extracting it
  through a bounded end-to-end semantic slice; record the chosen scope and costs.
- [ ] If selected, define a type AST for currently admitted primitives, nominal
  applications, type variables, sequences and callable signatures. Add outcomes,
  error/effect rows and associated kinds only when their semantics are decided.
- [ ] Resolve nominal symbols to identities rather than display strings.
- [ ] Centralize substitution, equality, signature comparison, containment, and
  specialization instead of repeatedly parsing type strings.
- [ ] Define operation constraints such as equality and ordering; distinguish
  generic laws from successful tests of particular instantiations.
- [ ] Test nested applications, cross-module references, recursive references,
  and unsupported constraints through all consumers.

### C3. Shared typed core and stage contracts — C01

- [ ] Decide whether a new shared core/lexer is justified before semantic work
  or should be extracted from a concrete vertical slice; do not treat the old
  audit's preferred architecture as an accepted requirement.
- [ ] Specify parsed, resolved/typed, elaborated, proof/evidence, and lowered forms.
- [ ] Under the chosen architecture, replace unsupported late-parsed source and
  scattered generic parsing with shared structured rules where needed.
- [ ] Preserve source-origin and branch identities through elaboration and stamping.
- [ ] Share stage contracts across CLI, LSP, evaluator, prover, catalogue,
  normalizer, identity machinery, and emitter.
- [ ] Eliminate execution depending on partially mutated or unprepared provider ASTs.
- [ ] Add exhaustive visitor/canonicalization coverage for every semantic node kind.
- [ ] Retain the evaluator as an independent oracle during backend changes.

## D. Decide success, call, binding, and pattern semantics

**Unresolved design choices.** Uniform successes and unified calls remain
candidates, alongside retaining deliberately specified current conventions.
The implementation/removal steps below apply only to a selected replacement.

### D1. Success convention — S01, E05

- [ ] Compare declared-shape/B11 successes, uniform whole-value `Ok(T)`, and raw
  values for total functions; record the semantic choice independently of any
  public envelope/tag or private layout.
- [ ] Specify the chosen construction/binder matrix for bodies, ensures, tests,
  scripts, named calls and callbacks, including real fields named `value`.
- [ ] Decide empty nominal records versus a genuine Unit type/value rather than
  assuming that uniform successes or a new Unit have already been selected.
- [ ] Implement whole-value `Ok(value)` for every supported T if approved.
- [ ] If whole-value semantics replace the current convention, remove flattened
  construction, redundant typed-Ok forms and outcome-only `.value` unwrapping;
  retain actual data fields named `value`. Otherwise document the chosen matrix.
- [ ] Migrate source, contracts, host implementations, normalized evidence,
  catalogues, and generated output using resolved types—not text replacement.
- [ ] Test every producer/consumer combination, especially empty records,
  single-field records, brands, sequences, variants, and Fn factories.

### D2. Application convention — S02

- [ ] Compare current separate `call`/`invoke`, one explicit `call`, and bare
  application; decide spelling separately from newly admitted callable positions.
- [ ] For any widened target positions, specify signature, precondition,
  authority and indirect-cycle checking before allowing fields or bound/returned
  Fn values to be invoked. Current invocation admits only bare Fn parameters.
- [ ] Repair B5 independently; preserve single evaluation and specified argument
  order under every application alternative.
- [ ] Keep static/indirect target information for authority, preconditions,
  cycle analysis, script attribution, and code generation.
- [ ] Test identical signatures in every newly admitted target position.

### D3. Immutable bindings and sequencing — S03

- [ ] Decide whether expression blocks/immutable bindings improve the current
  match/chain/forward forms; this is still a proposal, not a separate approved
  result of the application-spelling review.
- [ ] If selected, specify expression blocks with immutable bindings and a final result.
- [ ] Define irrefutable success binding for operations with an empty error set.
- [ ] Reject such binding for fallible operations; require dispatch or a separately
  specified completed-outcome transfer.
- [ ] Specify whole-outcome tail calls without hidden early exit or payload loss.
- [ ] Retire chain/forward forms only if a selected replacement covers their
  evaluation, outcome and evidence semantics.
- [ ] Compare evaluation order, state traces, error payloads, and branch evidence.

### D4. Unified patterns — S04

- [ ] Compare unified constructor-shaped patterns with the current pattern
  families; decide destructuring and arm syntax before implementing extensions.
- [ ] Specify record/tuple destructuring, whole nominal binding, constants,
  wildcards, ranges, or-patterns, and nested patterns.
- [ ] Include the parent sum type in case identity; allow unrelated parents to
  use the same short case name.
- [ ] Preserve exhaustiveness, overlap checks, priority, and explicit evaluation order.
- [ ] Define generic error-preserving relay separately from catch-all recovery.
- [ ] Test that adding a handled error kind cannot silently bypass coverage.

### D5. Contextual typing — S05

- [ ] Compare fully explicit applications, constructor/pattern-only context, and
  broader deterministic local context; decide exactly where omission is admitted.
- [ ] Keep explicit API and authority contracts and annotations for ambiguous locals.
- [ ] Require annotations when empty collections or polymorphic values are ambiguous.
- [ ] Prohibit inference of contracts or generic laws from example rows.
- [ ] Test equivalence of uniquely determined implicit and explicit annotations;
  preserve nominal typing and no numeric coercion.

## E. Modules, ownership, effects, and revisions

### E1. Namespaces and visibility — M01

**Immediate decision:** B2/B3 strict uniqueness and explicit reuse are supported.
Long-term namespace and visibility choices remain unresolved.

- [ ] Compare strict globals, owned global spellings, module-scoped symbols and
  package scope; decide visibility separately from collision repair.
- [ ] If selected, specify module/package identity, local names, qualified imports
  and public/private declarations, then replace the existing resolution model.
- [ ] Give errors/types one owner and import them instead of redeclaring lookalikes.
- [ ] Specify the intended public/private surface before changing generated helper
  exports; a strict-global repair alone does not introduce source visibility.
- [ ] Test moves/renames/reordered files, brand ownership, revision resolution
  and declaring-owner host bindings. Add same-short-name/private-access tests
  if the selected namespace/visibility design admits those distinctions.

### E2. Authored versus generated metadata — M02

- [ ] Remove or deliberately enforce module-level `emits`; do not leave a list
  looking contractual while unchecked.
- [ ] Decide whether exports remain in `provides` or derive from public
  declarations under E1; generated exports are conditional on that choice.
- [ ] Make summaries agree with checked declarations while preserving explicit
  imports and real public error/effect boundaries.

### E3. Explicit effects and capabilities — M03

Current externs are rejected by linked-pure/Fn-target admission even with empty
`emits`; the gap is absence of host capability rows, not automatic extern purity.

- [ ] Specify host observation/capability footprints independently of error sets.
- [ ] Carry authority transitively through named calls and callable types.
- [ ] Distinguish deterministic kernels from external observations.
- [ ] Preserve or explicitly replace brand-disclosure and asset-bridge certificates;
  never substitute an unrestricted cast/declassifier.
- [ ] Test that error-free impure operations cannot enter pure code or acquire
  concurrency/reordering permission from a missing annotation.

### E4. Revision and artifact identity — M04

- [ ] Decide public release/version policy versus private helper identity.
- [ ] Decide whether an immutable artifact manifest is warranted; do not assume
  a historical store or multi-version linker exists or has been selected.
- [ ] Retain accepted interface-drift checks and separate executable/evidence invalidation.
- [ ] Remove misleading historical-version/coexistence promises not backed by a linker/store.
- [ ] Reject wrong/mixed artifacts and candidate-supplied acceptance authority.

## F. Tests, proofs, termination, and runtime faults

### F1. Pure-source test semantics — E01

**Unresolved.** The replacement review did not choose a new default. Current
ordinary rows execute local source and script foreign source; linked-pure
integration is a separate Go helper with no unit-witness credit.

- [ ] Compare scripted contract boundaries plus required linked integration,
  real checked-pure execution by default, and explicit unit/linked test modes.
- [ ] Define deliberate mocks and real observations for the chosen mode; retain
  full reachable-graph admission and prohibit actual host execution at compile time.
- [ ] Specify stable effect-site identities for scripts and traces.
- [ ] Preserve request/response pairing, missing/leftover checks, and fresh test state.
- [ ] Specify and test helper extraction/movement under the chosen mode. If
  file-based contract boundaries remain, document the changed mock obligations
  rather than claiming location independence.
- [ ] Prevent compile-time host execution and accidental integration-to-unit coverage credit.

### F2. Evidence and acceptance — E02

- [ ] Preserve executed/certified/uncovered distinctions and authorized certificates.
- [ ] Track obligations against source match arms, not administrative lowering
  branches; current CAN4107 has no general proved-unreachable exemption.
- [ ] Keep real branch obligations through optimization.
- [ ] Decide explicit build/review policy for pinned-row warnings versus errors.
- [ ] Preserve the distinction between a source `pinned` marker and authority in
  an accepted baseline; do not claim A-light records an authenticated pinner.
- [ ] Test false contracts, changed acceptance rows, and evidence preservation through lowering.

### F3. Termination and runtime cost — E03

- [ ] Preserve and document the current unit-descent, Euclid and binary-narrowing
  schemas; none establishes resource bounds or target stack safety.
- [ ] Specify additional structural/well-founded measures before widening recursion admission.
- [ ] Evaluate finite traversal constructs/primitives against manual fuel customers;
  add them only after choosing their semantics and termination obligations.
- [ ] Evaluate tail-recursion lowering to loops/frames against measured stack
  limits, then implement and test the selected backend change.
- [ ] Benchmark repeated append and collection builders; permit private mutation
  optimizations only when source immutability and alias behavior are preserved.
- [ ] Test nondecreasing cycles and distinguish termination from resource/time bounds.

### F4. Fault contract — E04

- [ ] Separate declared domain errors from runtime/embedding-contract faults.
- [ ] Specify checked indexing and the proof obligation for unchecked indexing.
- [ ] Define bounds/platform conversion, malformed Unicode, canonical decimals,
  host throw/rejection, and resource-exhaustion behavior.
- [ ] Include B5's first-fault ordering in evaluator/target conformance; keep a
  future embedding-contract fault protocol distinct from today's primitive throws.
- [ ] Test that faults are neither fabricated successes nor undeclared ordinary errors.

### F5. Contract semantics and success binders — E05

- [ ] Reconcile ensures binders with the explicitly chosen success/pattern
  semantics; do not assume D1/D4 already selected a uniform replacement.
- [ ] Keep proof-fragment admission explicit and reject unsupported clauses whole.
- [ ] Distinguish ordinary checked/tested functions from universally verified functions.
- [ ] Expand proof support independently; keep TS shape checking separate from proof.
- [ ] Re-run false-contract and call-summary regressions after each relevant lowering change.

## G. Complete generic composition and data/function expressivity

### G1. Error-composition abstraction — T02

**Unresolved.** Concrete combinators, finite rows with relay first, and rows plus
first-class completed outcomes are distinct options. The following expansion
work applies only to the selected abstraction.

- [ ] Compare those three directions using actual map/recover/compose/zip/collect
  customers, implementation costs and explicit failure-accounting obligations.
- [ ] If rows are selected, specify their kinds, union/deduplication, total maps,
  variance/subsumption and relay versus recovery rules.
- [ ] If completed outcomes are selected, specify `Outcome<T,E>` and error
  values, including storing, nesting and transferring them as successful data.
- [ ] Define handling, preservation, transfer-as-data, and explicit discard;
  do not rely on unused-variable warnings to prevent hidden error loss.
- [ ] Update catalogue accounting for those distinct operations.
- [ ] Use one canonical declaration for function failures; avoid requiring both
  an Outcome return and a duplicate `emits E` for the same boundary.
- [ ] Distinguish any completed Outcome returned as successful data from the
  function's own outcome; do not implicitly flatten the two.
- [ ] Test the chosen abstraction's effects on callback admission and witnesses.
- [ ] Specify an explicit error product/collection when simultaneous failures
  must be retained: a union of error kinds alone cannot represent both failures.
- [ ] Implement only combinators supported by the chosen abstraction, with
  complete payload preservation and explicit simultaneous-failure semantics.

### G2. Higher-order source functions — T03

These are candidate extensions. The call-spelling review did not approve wider
callable positions, arbitrary host callbacks or a general effectful Fn model.

- [ ] Specify callable parameter lists, explicit captures, latent effects/errors,
  and preconditions consistently with ordinary function declarations.
- [ ] Replace capture-syntax whitelists with proven-pure data-expression rules where sound.
- [ ] Admit callbacks in additional source input/result/capture positions only
  after indirect-call and termination analysis covers them.
- [ ] Evaluate explicitly captured pure anonymous functions as a later convenience,
  not an excuse to admit effectful or arbitrary host callbacks.
- [ ] Test escaping values, shadowing, cycles, revisions, preconditions, and capability captures.

### G3. Data composition and nominal meaning — T01, T04

- [ ] Inventory position-specific nesting restrictions and existing composition
  (including Map's sequence of generic pairs); do not describe current nesting
  as uniformly absent. Decide and implement supported extensions through common rules.
- [ ] Preserve meaningful nominal wrappers; remove mechanical result carriers
  only where the selected result/type convention makes them unnecessary.
- [ ] Design opaque non-string representations and module-owned smart constructors.
- [ ] Keep validation predicates separate from nominal minting/authority claims.
- [ ] Specify finite recursive algebraic data and reject infinite products/host cycles.
- [ ] Test recursive equality/traversal, host validation, and a natural typed JSON tree.

## H. Specify four separate ABI contracts — F03, audit §7

**Host trust policy remains unresolved.** F03 establishes the current direct,
trusted JS boundary; it does not establish that a hostile-input facade is the
chosen product requirement. Keep source, private execution, public embedding
and wire contracts separate. Decide the public contract before any selected
success-layout migration; implementing every possible boundary is not required.

### H1. Source and private execution

- [ ] Specify source values, exact numerics, immutable data, outcomes, and evaluation order
  without depending on JS layout or generated specialization names.
- [ ] Document the current private call/closure/tag/specialization convention;
  define replacements only for selected changes. Frames are a B05 candidate.
- [ ] Permit private layout optimizations only under semantic differential tests.
- [ ] Do not make every internal operation pay external ingress-validation costs.

### H2. Public host facade

- [ ] Choose among an explicitly trusted direct embedding, a validated data-only
  facade, and a validated facade admitting checked callables. Use a real customer,
  intended trust boundary and measured copying/validation costs.
- [ ] Specify public entry points and supported generic instantiations.
- [ ] Decide the manifest/versioning policy for schemas, identities, effects/errors,
  ABI format and exact host binding ownership, then generate the selected artifacts.
- [ ] Decide the success/error envelope and nominal variant/error tag layout.
- [ ] Under a trusted policy, explicitly assign ownership, canonical-value,
  callable-provenance and fault responsibilities to the embedder; do not claim
  those guarantees are enforced against arbitrary JS callers.
- [ ] Under a validated policy, define copy/transfer/owned-view rules and check
  types, exact decimals, Unicode, tags, shapes and recursive structure.
- [ ] For validated ingress, reject or safely handle getters, exotic prototypes,
  cycles and post-completion mutation; test the actual selected policy.
- [ ] If public Fn values are admitted by a validated facade, require checked
  provenance; if data-only is selected, keep them private or specify capability
  handles. Neither `typeof function` nor a TS cast proves source purity.
- [ ] Generate readonly/opaque TS surfaces where useful without treating them as
  runtime guarantees; specify any trusted zero-copy/callback path explicitly.
- [ ] Decide stateful world instances, root reentry and the embedding-fault channel
  where required by the selected host/B05 contract.
- [ ] Preserve brand minting/disclosure authority; shape validation is not authorization.
- [ ] Add boundary and ownership conformance tests for the promised enforcement
  or trusted responsibilities; do not label documented trust as sandboxing.

### H3. Wire and persistence

- [ ] Specify wire codecs independently of private layout and host calling convention.
- [ ] Define exact bigint/decimal encoding and stable schema/case identities.
- [ ] Keep generated `$T$` names from becoming accidental persistent identities.
- [ ] Exclude callables, state/runtime handles, and unauthorized opaque/secret values from codecs.
- [ ] Define schema evolution and mixed-build rejection without promising legacy adapters.

## I. Resolve B05 async and resources — audit §8

**Accepted process, unresolved design.** Establish contracts against a real
customer before choosing a backend. Neither proposal is implemented, and the
review did not require building two complete runtimes or choose a public Promise ABI.

- [ ] Select a real async/resource customer and define its required source,
  authority, completion, cleanup and resource acceptance vectors.
- [ ] Reconcile proposal A's uniform completion-capable callable ABI (Promise is
  one possible lowering) with proposal B's private machine/frame model; keep B05
  as the design identifier and use the current success-type inventory.
- [ ] Resolve the source differences: labeled versus positional outcome slots,
  join-site identifiers, singleton admission, and obsolete record-only successes.
- [ ] Resolve the host-contract differences: A's dynamic deadline, declared
  rejection mapping and `concurrency isolated` versus B's timeout/settle bounds,
  owned receipts, `Host__World.use`, reentry rules and failed-world faults.
- [ ] Do not credit A's timeout result with B's cleanup/settlement guarantee;
  specify when child authority retires and which later effects are forbidden.
- [ ] Choose private execution machinery separately from the public root-completion facade.
- [ ] Identify uncertainties needing measurements and run only the bounded
  prototypes needed to resolve them against the same chosen acceptance vectors.
- [ ] Preserve joined ownership, complete outcome products, authority partition,
  and simultaneous-failure information.
- [ ] Define timeout versus actual settlement/cleanup; late effects must not be
  mislabeled as successful completion.
- [ ] State platform progress and scheduling assumptions honestly, including bigint work.
- [ ] Specify resources separately: acquisition/use/release, lifetimes, ownership,
  cleanup, and any cancellation/deadline authority.
- [ ] Add conformance tests for duplicate completions, reentry, mutation, late effects,
  settlement failure, and complete joins before claiming HTTP/SQL/UI unblocked.

## J. Formatter, syntax polish, and intrinsic tooling

### J1. Grammar and surface choices — S06

The collection, literal, boolean and label questions remain unresolved. Current
spellings/eager semantics remain the implementation baseline during repairs;
there is no requirement to preserve them after a deliberate replacement decision.

- [ ] Review the brace ban, especially punctuation inside comments, on semantic grounds.
- [ ] Compare optional labels, canonical positional forms and required
  multi-argument labels using actual edit/error trials; repair F06 independently
  and keep binding/evaluation order separate from formatting policy.
- [ ] Compare current `d"..."`/raw/`e"..."` families, ordinary exact decimals
  plus escaped strings, and a numeric-only change. Preserve exactness, canonical
  decimal values and Unicode semantics without coercion or binary floats.
- [ ] Compare `Seq<T>`, `[T]` and `T[]` independently of compositional nesting;
  measure usability rather than treating a punctuation preference as a type fix.
- [ ] Compare eager `and`/`or` with conditional evaluation and any explicit
  conditional spelling. Specify skipped/evaluated branch evidence and fault
  behavior before changing semantics; ordinary function arguments stay strict.
- [ ] Treat keyword renaming as low priority; do not confuse it with semantic simplification.

### J2. Formatter and diagnostics — C02

- [ ] Ship a canonical source formatter; outcome normalization is not source formatting.
- [ ] Separate compiler errors, advisory structural refactors, and explicit review policy.
- [ ] Do not let formatting silently rewrite control flow.
- [ ] Test parse/format/parse equivalence, idempotence, comments, literals, parentheses,
  source positions, grammar highlighting, and LSP/CLI agreement.

### J3. Intrinsics/runtime helpers — C04

- [ ] Centralize signatures, effects, evaluation, proof admission, lowering, and
  authority requirements in an intrinsic registry.
- [ ] Use ordinary source application/type rules for kernels without pretending
  deterministic kernels are host observations.
- [ ] Decide shared/versioned runtime helpers versus standalone emit using size,
  performance, and hermetic-build measurements.

## K. Migration and completion gates

### K1. Real stdlib customers

- [ ] Select migrations only after the applicable decision in D–J is recorded;
  the thirteen reviews do not authorize a blanket syntax or ABI rewrite.
- [ ] Inventory which single-field/empty records are semantic types versus old return carriers.
- [ ] Migrate approved return/call/type/module changes across stdlib, sketches,
  contracts, scripts, hosts, and compiler fixtures.
- [ ] Ship canonical optional APIs and decide `seq.find` migration.
- [ ] Ship outcome combinators and generic `schema__migrate` only after their actual
  chosen error-algebra and type requirements are implemented; do not assume
  first-class completed outcomes were selected over rows-first or concrete APIs.
- [ ] Re-probe remaining catalogue blockers rather than claiming async/resources,
  Unicode, or host APIs are solved by unrelated syntax work.
- [ ] Update implementation records and distinguish prototype, shipped, and pending APIs.

### K2. Every implementation slice

- [ ] Record the approved semantic/ABI delta and explicit non-goals.
- [ ] Write positive, negative, authority, and false-proof regressions before closing the slice.
- [ ] Migrate with parsed/resolved structure; remove obsolete forms instead of
  introducing a permanent old/new compatibility mode.
- [ ] Compare values, first-fault order, effect/error traces and source evidence
  across evaluator and target, including reordered named arguments from F06.
- [ ] Regenerate and review affected TS, catalogues, schemas, manifests, and goldens.
- [ ] Run Go tests, modcheck, gramcheck, whole-stdlib compilation, and strict TS
  on complete fresh output bundles; retain known-failure status until each new gate is fixed.
- [ ] Run Node, linked/LSP, boundary, witness, identity, and proof regressions as applicable.
- [ ] Update docs and probe status; turn repaired findings into ordinary regression tests.
- [ ] Record implementation commit and gate evidence without blessing a candidate baseline.

### K3. Measure improvement

- [ ] Identify which unresolved decision each measurement can resolve; record
  the absence of usability/performance evidence instead of substituting JEV scores.
- [ ] Benchmark large sequence/JSON inputs, allocation, stack depth, and runtime cost.
- [ ] Measure generic specialization growth, build time, and LSP latency.
- [ ] Run agent tasks comparing successful edits, error rates, and diagnostic recovery.
- [ ] Evaluate syntax choices using those results—not merely shorter source or JEV scores.

## Coverage map

| Audit section | Checklist |
|---|---|
| F01 / F02 / F03 / F04 / F05 | B1 / B2 / H / B3 / B4 |
| F06 (Workstream A follow-up) | B5 |
| S01 / S02 / S03 / S04 / S05 / S06 | D1 / D2 / D3 / D4 / D5 / J1 |
| T01 / T02 / T03 / T04 | C2 + G3 / G1 / G2 / G3 |
| M01 / M02 / M03 / M04 | E1 / E2 / E3 / E4 |
| E01 / E02 / E03 / E04 / E05 | F1 / F2 / F3 / F4 / F5 |
| Four ABI contracts / B05 | H / I |
| C01 / C02 / C03 / C04 | C3 / J2 / C1 / J3 |
| Illustrative destination / migration / measurements | D + approval decisions / K1–K2 / K3 |
| JEV review correction | A |

**Next actionable work:** repair the independently confirmed defects in B,
including the new F06 argument-order finding, and establish the accurate current
specification in C1 and complete dependency/target gates in B3/B4/K2. Use I's
contract-first process when beginning a real async customer. Choose C2/C3 and
the applicable D–J alternatives before their conditional implementation tasks.
Workstream A is complete; all thirteen dispositions are carried into this TODO,
with ten choices unresolved. Neither archived nor replacement JEV probabilities
approve a redesign or remove the need for semantic decisions.
