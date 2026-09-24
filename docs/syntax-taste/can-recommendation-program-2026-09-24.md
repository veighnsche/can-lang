# One recommendation program for Can

24 September 2026 · reviewed source revision `4c2db1e1162a45e88a58ef154e848790a1a44c60`

This document consolidates the [SaaS review](saas-language-review-2026-09-24.md) and the [clean-room review](clean-room-language-review-2026-09-24.md) into **one set of outcomes and evidence gates**. A finding repeated in both reviews appears once here. Distinct findings remain visible in the crosswalk below. The original reviews and their raw evidence remain the audit trail.

**Recommendation target:** Can should be a language I can recommend for a broad range of SaaS services and Can-authored user interfaces because its contracts survive ordinary changes, independent libraries compose, realistic applications work end to end, and supported deployments are qualified. The present implementation is not at that threshold. The current server-rendered/HTMX model can reach a narrower recommendation sooner; a broad claim covering rich Can-authored frontends additionally requires browser execution.

This is a design recommendation and acceptance program, not an amendment to the [current authoritative decisions](decisions.md) or the [September 22 disposition ledger](../implementation/language-design-dispositions-2026-09-22.md). Exact syntax and implementation work need a recorded design change. There are no external users or compatibility requirements to protect old spellings when that change is chosen.

The [design preparation](can-design-preparation-2026-09-24.md) has now evaluated these recommendations against the user-confirmed AI-agent identity and produced a [dependency-ordered implementation task list](can-implementation-task-list-2026-09-24.md). Total tokens per successful AI coding task are a secondary measured goal. The linked [disposition ledger](preparation/design-disposition-ledger.md) records accepted, retained and deferred recommendations; no production implementation is claimed by these documents.

## One language direction

Keep nominal immutable values, closed variants, named functions, explicit public domain-error bounds, attached assertions, native AI declarations with grouped state, controlled platform capabilities, and native JavaScript/Bun operations with only the adapters Can's contracts require. None of the reviews justifies `any`, implicit coercion, unrestricted JavaScript imports, general inheritance, a broad effect system, or replacement of the runtime owner model.

Make five promises more dependable:

1. **A typo cannot silently change the meaning of a contract.** Variant tests, package identities, and test scenarios should have explicit enough intent that a harmless rename does not select a different behavior accidentally. Capture refactoring is exercised in the product flow using explicit context records before selecting new capture syntax.
2. **An authored library can publish an honest contract.** A validated public value should be constructible only through its owner's admitted path when its API promises validity. A reusable helper should declare the operations and failures it requires rather than depending on hidden compiler privilege or an unadvertised template body.
3. **The server and rendered UI agree about an action.** Endpoint identity, path parameters, form fields, validation results, links, fragment targets, and displayed statuses should compose into a checkable flow. A safe URL alone does not prove that an endpoint exists or that a conditional DOM target is present.
4. **Operational behavior is demonstrated, not inferred from types.** Authorization performs a real check. Database contracts accurately state what is checked at compile time and runtime. Failure injection verifies cleanup, idempotency, uncertain commits, response display, and shutdown.
5. **A Can-authored frontend has a supported execution model.** Server HTML and HTMX remain useful. Local drafts, event handling, optimistic updates, rollback, and component disposal need a browser capability model if Can itself is to be recommended for those clients.

The first four promises plus a qualified service target establish the **server-driven SaaS gate**. All five establish the **Can-authored broad frontend gate**. A typed bridge to an existing browser stack is a practical supported integration, but it makes Can a server-language recommendation for that application, not the language of its browser frontend.

## Ordered gates

### Gate 1 — remove silent contract changes

**Pattern intent is the first core fix.** The current checker accepts a final `decliend` arm intended as `declined` by treating it as a binding/catch-all. After adding `refunded`, that match remains accepted while the correctly spelled old match becomes non-exhaustive. The [fresh probes](evidence/2026-09-24/clean-room-full-review/README.md#fresh-behavior-probes) establish this directly. I recommend distinct syntax for whole-value binding while retaining concise checked nominal tests; constructor-only nominal patterns are a reasonable fallback if the binder grammar proves awkward. An unknown intended case must be a diagnostic, including in the final arm. This is a targeted reconsideration of the earlier broad bare-name retention, not a request to rewrite all pattern vocabulary.

**Cross-package fixtures need deliberate scenario ownership.** Renaming a caller's assertion label currently changes which `when` row a helper package uses. Prototype explicit caller-to-helper scenario links, preserving exact argument checking, lexical invocation identity, queue isolation, and FIFO behavior. Compare the current lexical templates and whole-helper stubs. A descriptive root-label rename should not silently change dependency behavior.

**Independent packages need independent names.** Two unrelated dependencies currently collide on short package names or numeric application error IDs. Adopt dependency-qualified package identity with local import aliases and a scoped or collision-free diagnostic error identity. Preserve useful stable diagnostics; test two independently authored libraries without editing either library's source or registry.

**Variant identity needs one teachable rule.** Current generic variants reject a direct conversion between different specializations but accept the same value through another variant with identical leaves. This is not memory unsafety. Compare three policies against the recorded probes and ordinary `option<T>`: extensional named leaf sets with nominal record leaves; the existing direct-specialization guard stated precisely as a conservative restriction; or genuinely nominal variant wrappers with explicit representation, matching and codec consequences. The first fits today's flattened representation best, but the behavior alone does not select the policy.

Gate 1 passes when the misspelled leaf is rejected, a descriptive caller assertion-label rename preserves deliberately linked helper behavior, and two unrelated dependencies using the same short package name and locally allocated numeric error ID compose without edits. The direct/bridge/leaf variant probes must follow one explicitly chosen and teachable rule. This work comes before relying on further assertion or library evidence.

### Gate 2 — let libraries keep their promises

Prototype **package-controlled construction for ordinary immutable values**. Transparent records remain the default for DTOs. A library should also be able to export a nominal `email`, positive quantity, or other validated value while retaining construction and representation updates inside its package. Specify equality, observation, pattern matching, `with`, decoding, schema derivation, and assertion fixtures together. Generic decoding must not mint a value that bypasses its factory. A tenant-scoped identifier still does not prove current authorization; authorization requires an operation-bound check and a suitable lifetime.

For reusable fallible functions, compare the best current `match chain`, named-helper, fixed-bound, and result-as-data idioms against real `retry`, `audit`, and scoped-operation helpers. Require one helper to serve two callers with distinct finite domain-error sets without converting both into an undifferentiated error. **If current idioms materially fail**, trial a small *written finite error-set parameter*, with explicit substitution and outward bounds. That does not authorize implicit public error inference or a general effect system. For generic operations hidden inside bodies, first pass named operations or dictionary records explicitly; add only evidenced small capabilities if that baseline is poor.

Gate 2 passes when independently authored domain libraries can export validated values and useful generic helpers, and consumers understand required operations and possible failures from public contracts. Benchmark email, positive quantity and tenant identifiers, plus one helper serving two distinct caller error sets; the number of packages is not itself a product requirement. The prototype comparison may show that an individual extra mechanism is unnecessary; the outcome, rather than a preferred keyword, is the gate.

### Gate 3 — finish one server-driven product flow

Build a narrow, complete **tenant invoice edit** using current Can first: authenticate an actor, derive server-validated tenant context, check that actor's access to the specific invoice at the protected operation, open a page, parse a form with numeric seats, optional fields and repeated rows, validate into domain values, save inside a transaction, and render either the committed result or retained invalid input with field-specific messages. Include success, validation, conflict, forbidden, and unavailable responses. Test an actor submitting another tenant's invoice identifier. Exercise one resource path with typed captures and a safe URL builder. Current query-string IDs remain valid CRUD; path captures serve the navigable resource URL case.

Link an endpoint to its URL builder, request decoder, form/action use, and response contract. A route or form-field rename should diagnose affected consumers. Retain the shallow string-oriented HTML form wire model; build typed application values through an explicit validation layer with field paths and accumulated errors. Compare a good named-component helper library with a checked declarative HTML prototype before adopting more syntax. Existing safe HTML and native escaping remain the safety boundary.

Make the product outcome observable. Current source can return a 503 feedback fragment while the default HTMX policy prevents that status from swapping into the target. The end-to-end check must confirm what the user actually sees, not merely that an opaque HTTP response returns `ok`. Assert status, headers, escaped body, retained fields, target behavior, and tenant isolation in attached or companion protocol/browser scenarios. Attached assertions remain necessary but do not prove every external effect.

Add a separate **signed provider webhook** slice: replay a delivery, perform an idempotent ledger/outbox change, handle an uncertain commit, and prove one business effect under a stated receiver/provider idempotency contract. The outbox itself supplies durable retry attempts, not exactly-once delivery. Test a crash after external success but before outbox acknowledgement; where the receiver cannot deduplicate, define a reconciliation or duplicate policy. Use existing HTTP, crypto, SQL and transaction capabilities first. A bounded mutation `RETURNING` descriptor may remove a demonstrated read-after-write; it is a targeted platform extension, not a reason to introduce an ORM. State SQL's present guarantee accurately: statement/parameter/cardinality checking and runtime row validation do not prove the declared projection against an actual versioned schema. A versioned schema snapshot is a separate optional tooling design if compile-time database agreement is promised.

Gate 3 passes when the two flows work against a real database and protocol fixture/provider test, and controlled changes—route rename, field rename, added error, reordered same-typed fields, duplicate webhook, failure after write—produce useful diagnostics or deliberate behavior. Compare named-field construction with existing positional constructors during the reorder. Use explicit context records for captured request values, then verify that a parameter rename or same-typed shadow does not silently change the selected context; consider `near` syntax only if this baseline is materially costly. Measure changed files, repeated contracts, diagnostic quality, test observations, and behavior preserved; do not impose an arbitrary line-count target.

### Gate 4 — qualify the service that runs the flows

The currently pinned distribution target is Darwin ARM64 Bun. A recommendation for Linux-hosted SaaS requires a named Linux target, packaged build, installation, native API/behavior qualification, and execution of the invoice and webhook slices there. Exercise slow dependencies, client disconnect, pending owned work, close deadlines and shutdown under a documented host termination policy. State which operations can abort natively and which work remains live. An early race winner does not cancel losers, and an empty dynamic `race with error` remains pending under the selected native semantics.

Keep two related limits precise during qualification. `with_transaction<T>` can return a now-closed handle inside otherwise successful data; later use is rejected by runtime ownership. Evaluate a narrow diagnostic for an obvious escape of that transaction while allowing a still-live resource owned by an enclosing scope. A first-success race consumes failures that occur before its winner and reports late standard failures; test whether a service needs an optional fault-observation policy without changing native winner selection or generating routine fallback noise.

Fix the demonstrated stream example contract or its wording: `pump` claims every terminal path explicitly closes its reader, while read-error arms relay without calling `close_reader`. Runtime ownership provides fallback enforcement, but the example should teach the intended cleanup policy. Preserve original versus cleanup failures according to an explicit contract and test failure at each stage.

Gate 4 is the point at which I would recommend Can for the **demonstrated class of production server-driven SaaS**, subject to its stated platform and UI scope. Passing compiler tests alone is insufficient. Billing calendars, tax/refund policies, and specific provider SDKs should be maintained libraries/examples when they are needed, not language keywords.

### Gate 5 — support a Can-authored rich client

Use one demanding **editable invoice grid**: keyboard navigation, local unsaved drafts, immediate totals, optimistic save, failed-save rollback, offline/slow-connection behavior, and cleanup when a view disappears. Compare two explicit product paths: a Can browser target and a supported typed wire bridge to an established browser language. For the user's goal of Can on both frontend and backend, choose and qualify the Can browser path if the grid proves it practical. The bridge remains a useful integration and a narrower recommendation, not a substitute for a Can-authored client claim.

A browser prototype should reuse shared nominal/wire contracts while admitting only browser-safe capabilities. Server secrets, SQL pools, process access and service-only catalogue operations must be unavailable to browser code. Explore named state transitions, event handlers, asynchronous commands, renderers and disposal/cancellation on the real grid; do not assume a complete frontend framework, implicit effects, or shipping Bun server modules to browsers. Inventory accessible HTML and client capabilities needed by the example, including the current dialog/canvas/SVG gaps, without admitting arbitrary raw markup or scripts. The target can still use native browser operations with small Can contract adapters.

Gate 5 passes on a named browser/version support matrix after real interaction, accessibility and lifecycle tests cover edits, failures, reconnection and navigation, and the shared server/client contract survives route and data-shape changes. Only then is a broad **Can-authored backend and frontend** recommendation supported. No language can be certified for every possible SaaS product by one example; this gate establishes the claimed class of rich applications.

## Consolidated crosswalk

The two reviews contribute to these single work items. “Trial” means compare with the current idiom before adding a language mechanism; “later” means the finding is retained without blocking the gates above.

| Work item | SaaS review | Clean-room review | Consolidated treatment |
| --- | --- | --- | --- |
| Pattern typo becoming catch-all | — | §1 | Gate 1 targeted semantic repair |
| Authored validated values | §4 | §2 | Gate 2 package-owned immutable value trial |
| Higher-order error preservation | §5 | §3 | Gate 2 current-idiom benchmark, then finite-row trial if needed |
| Public generic operation requirements | — | §3 | Gate 2 explicit operation baseline; small constraints only if needed |
| Package names and error IDs | — | §4 | Gate 1 independent dependency composition |
| Fixture selection across packages | §6 reusable scenarios | §5 root-rename probe | Gate 1 explicit scenario ownership; retain lexical queue checks |
| Variant specialization identity | — | §6 | Gate 1 coherent contract, no unsoundness claim |
| `near` and callable capture | — | §7 | Later explicit binding comparison against context records |
| Routes, forms, auth context, and UI targets | §2 | §8 | Gate 3 typed product flow and checked rename behavior |
| Safe HTML authoring | §3 | §8 | Gate 3 helper-library versus checked-template trial |
| Browser Can scope | §1 | §8 | Gate 5; Gate 4 has an honestly narrower recommendation |
| Fallible sequencing and explicit cleanup | §5 | — | Gates 3–4; measure `match chain`, named helpers and owner behavior before new sequencing |
| Observable product assertions | §6 | §5 | Gate 3 protocol/browser outcomes; fixtures handled at Gate 1 |
| SQL mutation results and schema promise | §7 | §9 | Gate 3 targeted `RETURNING` trial; describe actual runtime row validation |
| Service lifetime and race policies | §7 | §10 | Gate 4 shutdown/abort qualification; keep native selection semantics |
| Linux release qualification | §7 | — | Gate 4 deployment condition, not grammar |
| Native AI wrapper/batch/callable limits | — | §11 | Later workflow-specific trials; preserve AI forms/grouped state |
| Scoped results and race fault visibility | — | §10 | Gate 4 bounded diagnostic/observability trials; retain runtime safety and native selection |
| Native catalogue / external SDKs | §7 | §12 | Later blocked-integration benchmark; retain controlled boundary |
| Bulk immutable collection building | — | §12 | Later native bulk adapter if batch-size benchmark warrants it |
| Billing/calendar rules | §7 | — | Domain library/example after core flow; existing exact numbers/time are baseline |
| Named-field construction | §8 | — | Gate 3 reorder comparison; syntax remains unselected |
| Multiline layout and local style | §8 | §7 | Later authoring improvements; formatter/diagnostics first |

This ordering resolves the reports' different emphases. The SaaS review correctly makes browser Can necessary for the *expanded* frontend promise; the clean-room review correctly puts silent meaning changes and library composition before trusting a large client target. The narrower server-driven recommendation is an intermediate milestone, not a claim that HTMX covers offline or highly interactive applications.

## Relation to earlier decisions

The September 22 ledger remains the current disposition record. The consolidated review supplies new evidence to consider changing particular entries, while preserving the rest:

| Earlier disposition | New evidence or clarified scope | Recommendation status here |
| --- | --- | --- |
| LD37 retained bare binding patterns | Executed final-arm typo/new-leaf counterexample | Reopen only the binding-versus-leaf ambiguity; keep unrelated `ok`/`as`/spread choices |
| LD28 deferred root-owned symbolic seams | Emitted cross-package assertion-label rename changes helper behavior | Reopen a bounded explicit scenario-link trial; preserve lexical identity and checked arguments |
| LD36 deferred error-ID redesign; package-name collision is a separate new finding | Current graph rejects duplicate application error IDs across dependencies and duplicate package short names in distinct directories | Audit diagnostic identity under LD36; separately prototype qualified imports/aliases for package composition |
| LD31 deferred static resource-escape diagnostics | Scoped result may carry its expired handle; later use is runtime-guarded | Evaluate only proven local-scope diagnostics with valid enclosing-resource counterexamples |
| LD14 deferred error-set parameters | Authored wrappers cannot match catalogue callback-error genericity | Keep demonstrated-need gate; compare real current wrappers before promotion |
| LD39 deferred explicit capture binding | Exact-name `near` creates an ambient refactoring dependency | Compare context-record baseline and same-type rename case before promotion |
| LD46 deferred multiline forms | Examples contain long lines and the grammar keeps calls/constructors/assertions on one physical line | Measure again after formatter/builder improvements; no new grammar selected here |
| LD24/LD30/LD32 retained native race, runtime ownership, controlled catalogue | Reviews describe operational limits but no safety bypass or blocked SDK | Retain; qualify lifetimes and integration experience separately |
| U7 excluded browser Can from initial scope | The user now asks for a broadly recommendable backend and frontend language | Treat browser execution as a proposed new product gate, not an existing missing implementation |

This is a recommendation to revise those decisions where the evidence warrants it. It does not silently replace the authoritative grammar, authorize a compiler edit, or claim that all candidate syntax has been selected.

## Consultation, validation, and stopping rule

Three fresh Jev SystemOne consultations used fully rewritten explanatory state, instructions, questions, and option descriptions with fixed facts and alternatives. They favored explicit pattern bindings, owner-controlled values, deliberate fixture seams, and measuring current error-wrapper idioms before a finite-row trial in all three wordings. The frontend path split: one response favored Can servers with a typed external browser client, while two favored qualifying server-driven Can first and then Can browser execution. I choose the staged Can browser path for the stated goal of **Can-authored frontend and backend**; the external-client choice remains a valid narrower product route. These are advisory classifications, not proof. [Exact requests, responses and disagreement analysis](evidence/2026-09-24/consolidated-recommendation/README.md) are saved.

The [SaaS evidence](evidence/2026-09-24/saas-review/README.md) covers all 56 current example `.can` files and focused runtime checks. The [clean-room evidence](evidence/2026-09-24/clean-room-full-review/README.md) contains three independent source audits, ten checker/emitter probes, tests across six compiler packages, and selected runtime tests. No new runtime behavior was inferred from Jev, and this consolidation did not rerun every example, provider, database or browser test.

**Stop adding language mechanisms once the gates pass.** A larger syntax feature earns its place only if a representative program with the best current Can idiom fails a stated contract or carries a measured material cost. That rule protects the language's small, explicit core while giving the user a concrete path to a recommendation.
