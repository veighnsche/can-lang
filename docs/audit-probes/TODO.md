# Can language audit — master TODO

Source: [full audit](../can-language-audit.md), baseline `8312d85`.
Reproductions: [probe guide](README.md).

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
| Next | C. Current specification, type model, compiler core | Proposal | Reconcile actual behavior and intended guarantees |
| Next | D. Uniform successes, calls, bindings, patterns | Proposal | C; source/public ABI decisions |
| Next | E. Modules, ownership, effects, revision identity | Proposal | C; coordinate with B's identity fixes |
| Next | F. Tests, evidence, contracts, faults | Proposal | C; coordinate with D/E |
| Then | G. Generic outcomes, data, higher-order functions | Proposal | C/D; error/effect/termination decisions |
| Then | H. Public host boundary and wire schemas | Design pending | Design before D's ABI migration; implementation may be staged |
| Later | I. B05 async and resources | Competing designs; not implemented | C/E/H; reconciled B05 decision |
| Parallel | J. Formatter, syntax polish, diagnostics | Proposal | Settled grammar and semantic choices |
| Per slice | K. Stdlib migration and acceptance gates | Pending | Relevant approved workstreams |

Checkboxes are completion evidence, not approval. Mark a design task complete
when its decision and rationale are recorded; mark an implementation task
complete only when its acceptance gates pass. Record the implementation commit
and test evidence when closing an item. Keep baseline failures separate from
regressions introduced by a change.

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
before changing code; the probe fixtures describe the audit baseline.

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

- [ ] Reject conflicting duplicate record identities at their declarations.
- [ ] Audit corresponding brand/error ownership and collision paths.
- [ ] Decide how genuinely shared definitions are imported; remove accidental
  coalescing/redeclaration and migrate affected stdlib sources.
- [ ] Test both source orders, identical short names in different owners, and
  ambiguous imports. Resolution must not depend on file order.
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

### C1. Authoritative specification — C03

- [ ] Publish one current semantic specification and an implemented/proposed/
  historical/deferred feature inventory.
- [ ] Reconcile the root README, REQUIREMENTS, can-idioms, stdlib comments,
  contract-activation comments, and conflicting B01 examples/plans.
- [ ] Document which claims are checked, tested, universally verified, or trusted.
- [ ] Preserve historical documents as history with clear supersession links.
- [ ] Generate syntax/intrinsic/diagnostic inventories where practical.

### C2. Compositional types — T01

- [ ] Define a type AST for primitives, nominal applications, type variables,
  sequences, callable signatures, outcomes, error rows, and effect rows.
- [ ] Give generic parameters explicit kinds; separate data/error/effect parameters.
- [ ] Resolve nominal symbols to identities rather than display strings.
- [ ] Centralize substitution, equality, signature comparison, containment, and
  specialization instead of repeatedly parsing type strings.
- [ ] Define operation constraints such as equality and ordering; distinguish
  generic laws from successful tests of particular instantiations.
- [ ] Test nested applications, cross-module references, recursive references,
  and unsupported constraints through all consumers.

### C3. Shared typed core and stage contracts — C01

- [ ] Specify parsed, resolved/typed, elaborated, proof/evidence, and lowered forms.
- [ ] Replace late-parsed source text and scattered generic-head parsing with a
  lexer, recursive grammar, and structured nodes.
- [ ] Preserve source-origin and branch identities through elaboration and stamping.
- [ ] Share stage contracts across CLI, LSP, evaluator, prover, catalogue,
  normalizer, identity machinery, and emitter.
- [ ] Eliminate execution depending on partially mutated or unprepared provider ASTs.
- [ ] Add exhaustive visitor/canonicalization coverage for every semantic node kind.
- [ ] Retain the evaluator as an independent oracle during backend changes.

## D. Simplify successes, calls, bindings, and patterns

**Design approval required.** These are candidate replacements, not additions
that must coexist permanently with the old forms.

### D1. Uniform success values — S01, E05

- [ ] Decide the single-value source convention and public success envelope.
- [ ] Define the same success binder meaning in bodies, ensures, tests, scripts,
  named calls, and callbacks.
- [ ] Specify a genuine Unit type/value for no-payload success.
- [ ] Implement whole-value `Ok(value)` for every supported T if approved.
- [ ] Remove flattened/multi-field Ok construction, B11's typed-Ok workaround,
  and outcome-only `.value` unwrapping; retain actual data fields named `value`.
- [ ] Migrate source, contracts, host implementations, normalized evidence,
  catalogues, and generated output using resolved types—not text replacement.
- [ ] Test every producer/consumer combination, especially empty records,
  single-field records, brands, sequences, variants, and Fn factories.

### D2. Unified application — S02

- [ ] Decide one application spelling for named and indirect targets.
- [ ] Type callable parameters, fields, and returned/bound values through the
  same application rule; remove the parameter-location restriction when sound.
- [ ] Preserve single evaluation and specified argument order.
- [ ] Keep static/indirect target information for authority, preconditions,
  cycle analysis, script attribution, and code generation.
- [ ] Test identical signatures in every newly admitted target position.

### D3. Immutable bindings and sequencing — S03

- [ ] Specify expression blocks with immutable local bindings and a final result.
- [ ] Define irrefutable success binding for operations with an empty error set.
- [ ] Reject such binding for fallible operations; require dispatch or a separately
  specified completed-outcome transfer.
- [ ] Specify whole-outcome tail calls without hidden early exit or payload loss.
- [ ] Retire redundant chain/forward forms after their semantics are covered.
- [ ] Compare evaluation order, state traces, error payloads, and branch evidence.

### D4. Unified patterns — S04

- [ ] Define constructor-shaped outcome/data patterns and one canonical arm form.
- [ ] Specify record/tuple destructuring, whole nominal binding, constants,
  wildcards, ranges, or-patterns, and nested patterns.
- [ ] Include the parent sum type in case identity; allow unrelated parents to
  use the same short case name.
- [ ] Preserve exhaustiveness, overlap checks, priority, and explicit evaluation order.
- [ ] Define generic error-preserving relay separately from catch-all recovery.
- [ ] Test that adding a handled error kind cannot silently bypass coverage.

### D5. Contextual typing — S05

- [ ] Decide where bidirectional/contextual checking replaces redundant annotations.
- [ ] Keep explicit API and authority contracts and annotations for ambiguous locals.
- [ ] Require annotations when empty collections or polymorphic values are ambiguous.
- [ ] Prohibit inference of contracts or generic laws from example rows.
- [ ] Test equivalence of uniquely determined implicit and explicit annotations;
  preserve nominal typing and no numeric coercion.

## E. Modules, ownership, effects, and revisions

### E1. Namespaces and visibility — M01

- [ ] Specify module/package identity, local names, qualified imports, and public/private declarations.
- [ ] Replace global double-underscore resolution with scoped symbol resolution if approved.
- [ ] Give errors/types one owner and import them instead of redeclaring lookalikes.
- [ ] Stop exporting every implementation helper as public TS API.
- [ ] Test moves/renames/reordered files, duplicate short names, private access,
  brand ownership, revision resolution, and declaring-owner host bindings.

### E2. Authored versus generated metadata — M02

- [ ] Remove or deliberately enforce module-level `emits`; do not leave a list
  looking contractual while unchecked.
- [ ] Derive exports from public declarations rather than a duplicate `provides` list.
- [ ] Generate summaries from checked declarations while preserving explicit imports
  and real public error/effect boundaries.

### E3. Explicit effects and capabilities — M03

- [ ] Specify host observation/capability footprints independently of error sets.
- [ ] Carry authority transitively through named calls and callable types.
- [ ] Distinguish deterministic kernels from external observations.
- [ ] Preserve or explicitly replace brand-disclosure and asset-bridge certificates;
  never substitute an unrestricted cast/declassifier.
- [ ] Test that error-free impure operations cannot enter pure code or acquire
  concurrency/reordering permission from a missing annotation.

### E4. Revision and artifact identity — M04

- [ ] Decide public release/version policy versus private helper identity.
- [ ] Move exact artifact selection to an explicit immutable manifest where appropriate.
- [ ] Retain accepted interface-drift checks and separate executable/evidence invalidation.
- [ ] Remove misleading historical-version/coexistence promises not backed by a linker/store.
- [ ] Reject wrong/mixed artifacts and candidate-supplied acceptance authority.

## F. Tests, proofs, termination, and runtime faults

### F1. Location-independent pure tests — E01

- [ ] Decide normal execution of checked pure Can across module boundaries.
- [ ] Keep deliberate contract mocks explicit and real observations scripted.
- [ ] Specify stable effect-site identities for scripts and traces.
- [ ] Preserve request/response pairing, missing/leftover checks, and fresh test state.
- [ ] Test helper extraction/movement without changing behavior or evidence obligations.
- [ ] Prevent compile-time host execution and accidental integration-to-unit coverage credit.

### F2. Evidence and acceptance — E02

- [ ] Preserve executed/certified/uncovered distinctions and authorized certificates.
- [ ] Track obligations against source branches, not administrative lowering branches.
- [ ] Keep real branch obligations through optimization.
- [ ] Decide explicit build/review policy for pinned-row warnings versus errors.
- [ ] Test false contracts, changed acceptance rows, and evidence preservation through lowering.

### F3. Termination and runtime cost — E03

- [ ] Specify additional structural/well-founded measures before widening recursion admission.
- [ ] Add finite traversal constructs or primitives instead of ubiquitous manual fuel plumbing.
- [ ] Lower tail recursion to loops/frames and test beyond evaluator/JS stack limits.
- [ ] Benchmark repeated append and collection builders; permit private mutation
  optimizations only when source immutability and alias behavior are preserved.
- [ ] Test nondecreasing cycles and distinguish termination from resource/time bounds.

### F4. Fault contract — E04

- [ ] Separate declared domain errors from runtime/embedding-contract faults.
- [ ] Specify checked indexing and the proof obligation for unchecked indexing.
- [ ] Define bounds/platform conversion, malformed Unicode, canonical decimals,
  host throw/rejection, and resource-exhaustion behavior.
- [ ] Test that faults are neither fabricated successes nor undeclared ordinary errors.

### F5. One contract semantics — E05

- [ ] Align ensures binders with the new executable success/pattern semantics.
- [ ] Keep proof-fragment admission explicit and reject unsupported clauses whole.
- [ ] Distinguish ordinary checked/tested functions from universally verified functions.
- [ ] Expand proof support independently; keep TS shape checking separate from proof.
- [ ] Re-run false-contract and call-summary regressions after each relevant lowering change.

## G. Complete generic composition and data/function expressivity

### G1. Finite error rows and completed outcomes — T02

- [ ] Specify `Outcome<T,E>`, error values, finite row union/deduplication, and total mappings.
- [ ] Define handling, preservation, transfer-as-data, and explicit discard;
  do not rely on unused-variable warnings to prevent hidden error loss.
- [ ] Update catalogue accounting for those distinct operations.
- [ ] Use one canonical declaration for function failures; avoid requiring both
  an Outcome return and a duplicate `emits E` for the same boundary.
- [ ] Distinguish an Outcome returned as successful data from the function's own outcome.
- [ ] Decide error-row variance/subsumption and its effect on callback admission and witnesses.
- [ ] Implement generic map/and_then/map_error/recover/zip/collect with complete
  payload and simultaneous-failure preservation.

### G2. Higher-order source functions — T03

- [ ] Specify callable parameter lists, explicit captures, latent effects/errors,
  and preconditions consistently with ordinary function declarations.
- [ ] Replace capture-syntax whitelists with proven-pure data-expression rules where sound.
- [ ] Admit callbacks in additional source input/result/capture positions only
  after indirect-call and termination analysis covers them.
- [ ] Evaluate explicitly captured pure anonymous functions as a later convenience,
  not an excuse to admit effectful or arbitrary host callbacks.
- [ ] Test escaping values, shadowing, cycles, revisions, preconditions, and capability captures.

### G3. Data composition and nominal meaning — T01, T04

- [ ] Admit nested generic data, nested sequences, and sequences of variants through
  common type/representation rules rather than ad hoc exclusions.
- [ ] Preserve meaningful nominal wrappers; remove purely mechanical result carriers.
- [ ] Design opaque non-string representations and module-owned smart constructors.
- [ ] Keep validation predicates separate from nominal minting/authority claims.
- [ ] Specify finite recursive algebraic data and reject infinite products/host cycles.
- [ ] Test recursive equality/traversal, host validation, and a natural typed JSON tree.

## H. Specify four separate ABI contracts — F03, audit §7

Public ABI design is a prerequisite for the success-layout migration; full
boundary implementation can be split into later green slices.

### H1. Source and private execution

- [ ] Specify source values, exact numerics, immutable data, outcomes, and evaluation order
  without depending on JS layout or generated specialization names.
- [ ] Define the private call/closure/tag/frame/specialization convention.
- [ ] Permit private layout optimizations only under semantic differential tests.
- [ ] Do not make every internal operation pay external ingress-validation costs.

### H2. Public host facade

- [ ] Specify public entry points and supported generic instantiations.
- [ ] Generate a versioned manifest of schemas, identities, effects/errors, ABI
  format, and exact host binding ownership.
- [ ] Decide the success/error envelope and nominal variant/error tag layout.
- [ ] Generate readonly/opaque TS surfaces without treating them as runtime guarantees.
- [ ] Define copy/transfer/owned-view rules for arrays, buffers, records, and returned data.
- [ ] Validate types, canonical decimals, Unicode, tags, shapes, and recursive structure.
- [ ] Reject or safely handle getters, exotic prototypes, cycles, and post-completion mutation.
- [ ] Require checked callable provenance at public Fn boundaries; do not certify
  arbitrary JS callbacks through `typeof function` or a TS cast.
- [ ] Specify any explicitly trusted zero-copy/host-callback path separately.
- [ ] Define stateful world instances, root reentry policy, and the embedding-fault channel.
- [ ] Preserve brand minting/disclosure authority; shape validation is not authorization.
- [ ] Add adversarial boundary and ownership conformance tests.

### H3. Wire and persistence

- [ ] Specify wire codecs independently of private layout and host calling convention.
- [ ] Define exact bigint/decimal encoding and stable schema/case identities.
- [ ] Keep generated `$T$` names from becoming accidental persistent identities.
- [ ] Exclude callables, state/runtime handles, and unauthorized opaque/secret values from codecs.
- [ ] Define schema evolution and mixed-build rejection without promising legacy adapters.

## I. Resolve B05 async and resources — audit §8

- [ ] Reconcile the Promise/completion proposal and private machine/frame proposal
  using complete Can context; keep B05 as the design identifier.
- [ ] Resolve labeled versus positional join-product syntax and remove obsolete
  record-success assumptions from the chosen proposal.
- [ ] Choose private execution machinery separately from the public root-completion facade.
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

- [ ] Review the brace ban, especially punctuation inside comments, on semantic grounds.
- [ ] Decide declaration and named-argument spelling; preserve useful labels if approved.
- [ ] Decide exact decimal and raw/escaped string literal families without adding coercion,
  binary-float semantics, rounding, or Unicode normalization.
- [ ] Decide regular collection type spelling on compositionality, not fashion.
- [ ] Decide eager versus short-circuit boolean semantics and any branch-evidence obligations.
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

- [ ] Inventory which single-field/empty records are semantic types versus old return carriers.
- [ ] Migrate approved return/call/type/module changes across stdlib, sketches,
  contracts, scripts, hosts, and compiler fixtures.
- [ ] Ship canonical optional APIs and decide `seq.find` migration.
- [ ] Ship outcome combinators and generic `schema__migrate` only after their actual
  error-algebra and type requirements are implemented.
- [ ] Re-probe remaining catalogue blockers rather than claiming async/resources,
  Unicode, or host APIs are solved by unrelated syntax work.
- [ ] Update implementation records and distinguish prototype, shipped, and pending APIs.

### K2. Every implementation slice

- [ ] Record the approved semantic/ABI delta and explicit non-goals.
- [ ] Write positive, negative, authority, and false-proof regressions before closing the slice.
- [ ] Migrate with parsed/resolved structure; remove obsolete forms instead of
  introducing a permanent old/new compatibility mode.
- [ ] Compare values, effect/error traces, and source evidence across evaluator and target.
- [ ] Regenerate and review affected TS, catalogues, schemas, manifests, and goldens.
- [ ] Run Go tests, modcheck, gramcheck, whole-stdlib compilation, and strict TS
  on complete fresh output bundles; retain known-failure status until each new gate is fixed.
- [ ] Run Node, linked/LSP, boundary, witness, identity, and proof regressions as applicable.
- [ ] Update docs and probe status; turn repaired findings into ordinary regression tests.
- [ ] Record implementation commit and gate evidence without blessing a candidate baseline.

### K3. Measure improvement

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
specification and gates. Workstream A is complete; its design dispositions are
explicitly recorded, with ten choices unresolved. Neither archived nor replacement
JEV probabilities approve a redesign or remove the need for semantic decisions.
