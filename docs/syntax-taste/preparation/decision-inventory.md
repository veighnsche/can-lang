# Can design decision inventory

24 September 2026 · P1 coverage ledger · preparation artifact, not an adopted design

## Status and reading rules

The user has fixed Can's audience: **AI coding agents**. Human readability, familiarity, authoring convenience and comfort are not design goals. Human difficulty is acceptable when it serves agents; difficulty is not itself a goal. On September 24 the user also selected **total tokens per successful AI coding task** as a secondary measured goal, including prompts, code, diagnostics and retries. This is a task-level outcome, not an instruction to minimize source length or a substitute for correctness. Both choices are now recorded in the [current design direction](../decisions.md#design-direction) and [preparation checklist](../can-design-preparation-2026-09-24.md).

The [current decisions](../decisions.md), [technical specification](../technical-spec.md), [platform/testing specification](../platform-testing-spec.md), and [September 22 disposition ledger](../../implementation/language-design-dispositions-2026-09-22.md) remain authoritative for admitted behavior and existing disposition. The [recommendation program](../can-recommendation-program-2026-09-24.md) and its [SaaS](../saas-language-review-2026-09-24.md) and [clean-room](../clean-room-language-review-2026-09-24.md) source reviews provide observations, recommendations, alternatives and evidence; their gate order is proposed, not a proven dependency order. The program reviewed source revision `4c2db1e1162a45e88a58ef154e848790a1a44c60`; this inventory was prepared at HEAD `02a549d28fd5fc5c3996160e65a97de332390d30`. The review's probes and file-specific claims are historical evidence and need reproduction or qualification against current source during P3 before they drive a design change. An earlier `implement` disposition does not select exact syntax or prove production acceptance. An inventory entry is neither a design approval nor an implementation task.

Each **DI** ID is stable for the decision topic. The observations below distinguish source-verified behavior, executed probes, and source-level deductions from unverified expectations. A candidate is a hypothesis to compare against the best current Can idiom. **Agent reassessment** flags arguments whose rationale invokes human readability, familiarity, visual comfort, line length or syntax taste. A flag does not reject an outcome: refactoring errors, diagnosability and measured agent editing cost can still justify it. All 23 rows of the [consolidated crosswalk](../can-recommendation-program-2026-09-24.md#consolidated-crosswalk) map to the IDs here; independent subquestions have suffix IDs. No compatibility migration is needed for old spellings, ABI, goldens or generated layouts ([repository AGENTS.md](../../../AGENTS.md)).

## Coverage map

| Program crosswalk item | Destination |
| --- | --- |
| Pattern typo becoming catch-all | DI-01 |
| Authored validated values | DI-02 |
| Higher-order error preservation | DI-03 |
| Public generic operation requirements | DI-04 |
| Package names and error IDs | DI-05a, DI-05b |
| Fixture selection across packages | DI-06 |
| Variant specialization identity | DI-07 |
| `near` and callable capture | DI-08 |
| Routes, forms, auth context, and UI targets | DI-09a–DI-09d |
| Safe HTML authoring | DI-10 |
| Browser Can scope | DI-11a–DI-11c |
| Fallible sequencing and explicit cleanup | DI-12a, DI-12b |
| Observable product assertions | DI-13 |
| SQL mutation results and schema promise | DI-14a, DI-14b |
| Service lifetime and race policies | DI-15a–DI-15c |
| Linux release qualification | DI-16 |
| Native AI wrapper/batch/callable limits | DI-17a–DI-17c |
| Scoped results and race fault visibility | DI-18a, DI-18b |
| Native catalogue / external SDKs | DI-19 |
| Bulk immutable collection building | DI-20 |
| Billing/calendar rules | DI-21 |
| Named-field construction | DI-22 |
| Multiline layout and local style | DI-23a, DI-23b |

## Findings and unresolved decisions

### DI-01 — Pattern intent and typo diagnostics

**Observed:** The checker treats an unrecognized bare name in a match as a binding. A final `decliend` arm, intended as `declined`, becomes a catch-all; adding a new `refunded` leaf leaves that typo accepted while the correctly spelled old match becomes non-exhaustive. This is an executed checker/emitter probe, not just a taste objection ([clean-room §1](../clean-room-language-review-2026-09-24.md#1-make-pattern-intent-unambiguous), [probe evidence](../evidence/2026-09-24/clean-room-full-review/README.md#fresh-behavior-probes)).

**Outcome and candidates:** Intentional whole-value binding and nominal leaf testing must be distinguishable, including in the final arm; unknown intended cases should be diagnosed. Compare explicit binding syntax against constructor-only nominal patterns, including nested patterns, catch-all forms and diagnostics. A spelling heuristic is inadequate. Exact syntax is unselected ([program Gate 1](../can-recommendation-program-2026-09-24.md#gate-1--remove-silent-contract-changes)).

**Authority/gap:** [LD37](../../implementation/language-design-dispositions-2026-09-22.md#syntax-documentation-and-authoring-feedback) retains the broader bare-name vocabulary; [LD17](../../implementation/language-design-dispositions-2026-09-22.md#failure-contracts-and-coordination) separately accepts exact generic-error discrimination. Reopen only binding-versus-leaf ambiguity, without treating `ok`/`as`/spread as selected for redesign. Need syntax examples and negative acceptance cases. **Agent reassessment:** evaluate typo resistance and edit repair for agents; human familiarity of a pattern spelling is irrelevant.

### DI-02 — Package-owned validated values

**Observed:** Nominal immutable records protect type identity and later mutation, but ordinary exported records and `with` remain constructible without an application validator; generic decoding may likewise mint a representation. Catalogue-owned protected values exist, authored package-owned representations do not ([SaaS §4](../saas-language-review-2026-09-24.md#4-i-would-recommend-can-if-applications-could-enforce-their-own-valid-value-construction), [clean-room §2](../clean-room-language-review-2026-09-24.md#2-decide-whether-authored-libraries-may-own-value-invariants)).

**Outcome and candidates:** Test an exported email, positive quantity and tenant identifier whose owner controls construction and representation updates. Compare a package-owned immutable representation boundary or authored opaque value with current public-record-plus-factory idiom. Define observation, equality, matching, `with`, JSON/schema decoding and assertion fixtures together; a generic decoder must not bypass the owner. Authorization remains an operation-bound, lifetime-sensitive check, not a property conferred by a type name. No keyword, refinement calculus or general escape hatch is selected ([program Gate 2](../can-recommendation-program-2026-09-24.md#gate-2--let-libraries-keep-their-promises)).

**Authority/gap:** Current records and decoding remain authoritative ([decisions: records](../decisions.md#records), [technical C4](../technical-spec.md#c4)); the September 22 ledger has no adopted package-owned record abstraction. Need baseline programs across independent packages and decoder/fixture counterexamples. **Agent reassessment:** test invariant survival under agent generation and edits, not whether an opaque spelling feels natural to humans.

### DI-03 — Finite higher-order error preservation

**Observed:** Built-in collection callbacks can preserve specialized finite error sets, while authored `retry`, `map_option`, `with_audit` or scoped helpers must name concrete bounds or return nominal result data. `match chain` and named helpers already handle some sequential composition ([clean-room §3](../clean-room-language-review-2026-09-24.md#3-let-ordinary-libraries-express-useful-generic-contracts), [SaaS §5](../saas-language-review-2026-09-24.md#5-i-would-recommend-can-if-fallible-workflows-and-cleanup-stayed-readable-as-they-grew)).

**Outcome and candidates:** Benchmark best current chain/helper, fixed bound and result-as-data idioms on one helper used by callers with distinct domain-error sets. If a material agent/workflow cost remains, trial a *written finite error-set parameter* with explicit substitution, outward bounds, diagnostics and native lowering. Result data remains a viable alternative with conversion costs. More privileged built-ins and a broad effect system are separate, unsupported solutions ([program Gate 2](../can-recommendation-program-2026-09-24.md#gate-2--let-libraries-keep-their-promises)).

**Authority/gap:** [LD14](../../implementation/language-design-dispositions-2026-09-22.md#demonstrated-need-gate-for-larger-abstractions) is deferred under a user-confirmed demonstrated-need gate; explicit public `emits` remains ([LD12](../../implementation/language-design-dispositions-2026-09-22.md#failure-contracts-and-coordination)). Need realistic callback examples and evidence beyond shorter spelling. **Agent reassessment:** judge agent composition and repair cost, not perceived verbosity.

### DI-04 — Public generic operation requirements

**Observed:** A generic body is checked at concrete specialization but its public signature need not declare operations used in the body. A `doubled<item>` body using `value + value` can therefore fail for a new consumer after a body change with no signature change. This is a template-contract limitation, not unchecked emitted code ([clean-room §3](../clean-room-language-review-2026-09-24.md#3-let-ordinary-libraries-express-useful-generic-contracts)).

**Outcome and candidates:** Compare explicit named callable parameters and dictionary records first; if independent library use remains materially awkward, trial a small capability vocabulary. Documented template semantics may remain appropriate for local helpers. Full traits, higher-kinded types and broad operator overloading do not follow. Need a consumer-breakage and separate-library example plus changed-signature diagnostics. **Authority:** current generic specialization and callable rules remain ([decisions: generic types](../decisions.md#generic-type-spelling), [callables](../decisions.md#callable-types)); no matching adopted ledger replacement. **Agent reassessment:** assess whether an agent can infer and maintain the contract from signatures and diagnostics.

### DI-05a — Dependency-qualified package identity

**Observed:** Project-qualified graph identity exists, but duplicate short package names in unrelated dependencies are rejected, even when their authors are independent. Current file-local import aliases do not resolve that graph collision ([clean-room §4](../clean-room-language-review-2026-09-24.md#4-make-independent-packages-independent), [program Gate 1](../can-recommendation-program-2026-09-24.md#gate-1--remove-silent-contract-changes)).

**Outcome and candidates:** Test two independent packages with the same convenient short name and consume both without editing either. Compare dependency-qualified import identity with local aliases and precise collision diagnostics. Package distribution/registry work is not a prerequisite. **Authority/gap:** current flat unique names and `uses ... as` are specified ([decisions: packages](../decisions.md#packages)); this collision is a new finding rather than merely LD36. Need rules for source names, file-local resolution and emitted/import identities.

### DI-05b — Diagnostic application-error identity

**Observed:** Independently allocated numeric application error IDs collide across dependency graphs; retired IDs participate. Numeric identity has existing diagnostic/reporting use ([clean-room §4](../clean-room-language-review-2026-09-24.md#4-make-independent-packages-independent)).

**Outcome and candidates:** Audit all ID consumers, then compare qualified canonical identity with scoped or collision-free generated reporting IDs. Preserve stable machine-readable diagnostics where needed; merely deleting numeric IDs is not justified. Test two independent libraries with overlapping IDs. **Authority/gap:** [LD36](../../implementation/language-design-dispositions-2026-09-22.md#assertions-resources-and-platform-contracts) defers manual-ID redesign until consumer and reporting requirements are audited. No compatibility requirement protects old IDs. This is coupled to DI-05a in the composition test but has its own reporting contract.

### DI-06 — Cross-package fixture scenario ownership

**Observed:** A helper package's `when` row is selected by the caller's root assertion short label. In an emitted probe, renaming the root from `customer` to `renamed` changes a helper result from fixture text to real `text::from_int(7)`, while static checking still passes. Full root identities and queues remain isolated; this is deliberate selection by a fragile label, not a concurrency collision ([clean-room §5](../clean-room-language-review-2026-09-24.md#5-give-test-scenarios-explicit-ownership-across-calls), [probe evidence](../evidence/2026-09-24/clean-room-full-review/README.md#fresh-behavior-probes)).

**Outcome and candidates:** Compare explicit caller-to-helper named scenario/seam references with lexical fixture templates and whole-helper stubs. Preserve checked arguments, invocation identity, queue isolation and FIFO occurrence. Qualifying labels alone may leave the caller/callee coupling. Need rename, collision, refactor and concurrency acceptance examples. **Authority/gap:** [LD27](../../implementation/language-design-dispositions-2026-09-22.md#assertions-resources-and-platform-contracts) accepts inert exact-target lexical templates; [LD28](../../implementation/language-design-dispositions-2026-09-22.md#assertions-resources-and-platform-contracts) defers root-owned symbolic seams. The new emitted rename probe supports reopening a bounded trial, not adopting a seam syntax. Source-attached assertions and checked arguments remain ([LD26](../../implementation/language-design-dispositions-2026-09-22.md#assertions-resources-and-platform-contracts)).

### DI-07 — Variant specialization and leaf compatibility

**Observed:** Direct `tagged<int>` to `tagged<str>` conversion fails, while passage through a distinct `bridge` variant with the same leaf set succeeds; narrowing to a shared `unit` leaf succeeds. The admitted paths were executed. The checker combines a direct-specialization guard with leaf-set inclusion, so assignment is non-transitive; there is no demonstrated runtime memory safety defect ([clean-room §6](../clean-room-language-review-2026-09-24.md#6-clarify-variant-identity-instead-of-implying-stronger-branding)).

**Outcome and candidates:** Choose a precise policy after comparing (1) extensional named leaf sets and nominal record leaves, (2) current direct-specialization restriction clearly stated, and (3) genuinely nominal variant wrappers with representation, matching and codec consequences. Test `option<T>`, generic records and all direct/bridge/leaf paths. The flattened representation favors the first candidate but does not decide it. **Authority/gap:** current [variant decisions](../decisions.md#variants) and compatibility rules apply; the program proposes Gate 1 clarification, not an adopted change. **Agent reassessment:** replace a purely “teachable” or familiar-rule argument with predictive diagnostics and agent edit behavior.

### DI-08 — Explicit `near` capture binding

**Observed:** `near` resolves a callee parameter's exact spelling in the reference-creation scope. Renaming that parameter may break creation; a same-typed local with the new name can silently change the captured value. The created closure remains immutable ([clean-room §7](../clean-room-language-review-2026-09-24.md#7-make-captures-and-layout-easier-to-refactor)).

**Outcome and candidates:** Use an explicit context-record/receiver baseline in the invoice handler before testing a small explicit binding form. Exercise two values of the same domain type and a rename/shadow edit; preserve capture-once semantics and ownership checks. Anonymous functions are not required. **Authority/gap:** exact-name capture is selected ([decisions: near inputs](../decisions.md#near-inputs)); [LD39](../../implementation/language-design-dispositions-2026-09-22.md#demonstrated-need-gate-for-larger-abstractions) defers changed syntax pending a concrete dependency-edit failure and comparison to tooling. [LD38](../../implementation/language-design-dispositions-2026-09-22.md#syntax-documentation-and-authoring-feedback) retains distinct receiver/capture/argument roles. **Agent reassessment:** measure agent refactor correctness, not human comfort with implicit scope.

### DI-09a — Route identity, typed path captures and URL builders

**Observed:** Routes are exact paths; `:id`/wildcard segments are rejected. Safe URL values do not prove endpoint membership. Renderers can construct URL strings separately from route mounts ([SaaS §2](../saas-language-review-2026-09-24.md#2-i-would-recommend-can-if-routes-actions-and-forms-were-connected-by-types), [clean-room §8](../clean-room-language-review-2026-09-24.md#8-connect-frontend-contracts-then-choose-the-frontend-scope)).

**Outcome and candidates:** Test `/tenants/:tenant_id/invoices/:invoice_id` with typed captures, parse/error behavior and a linked safe URL builder. Query IDs already support CRUD; resource paths serve navigation and endpoint composition. A route rename should diagnose consumers. Need distinguish compile-time endpoint existence from runtime routing, and type the response/action relationship. **Authority/gap:** current exact route admission and safe URL contracts remain ([platform specification](../platform-testing-spec.md)); no new path grammar is selected.

### DI-09b — Form wire fields and application validation

**Observed:** HTML form schema admits shallow `str`, `str[]` and optional `str`; numeric and boolean values require parsing. Field names are separate strings and rejected input/field errors are not automatically connected to a nominal domain value ([SaaS §2](../saas-language-review-2026-09-24.md#2-i-would-recommend-can-if-routes-actions-and-forms-were-connected-by-types), [clean-room §8](../clean-room-language-review-2026-09-24.md#8-connect-frontend-contracts-then-choose-the-frontend-scope)).

**Outcome and candidates:** Keep string-oriented wire data, compare reusable validation library and linked form/action contracts for numeric seats, optional details and repeated rows. Accumulate field-path errors and retain rejected values; a field rename should diagnose uses. Do not infer JSON-style coercion. Coordinate with DI-02, DI-09a, DI-10 and DI-13. **Authority:** current form decoder and nominal records remain; richer projection is proposed, not selected.

### DI-09c — Operation-bound authorization and request context

**Observed:** A validated principal/tenant value does not alone prove access to a particular invoice. A capability or opaque handle may also outlive the request/tenant context unless lifetime is specified ([SaaS §2 and §4](../saas-language-review-2026-09-24.md#2-i-would-recommend-can-if-routes-actions-and-forms-were-connected-by-types), [program Gate 3](../can-recommendation-program-2026-09-24.md#gate-3--finish-one-server-driven-product-flow)).

**Outcome and candidates:** In the invoice flow, authenticate, derive server-validated context and check access at the protected operation before database work. Test a cross-tenant identifier and context lifetime. Compare explicit context records before any capture syntax change. This is primarily an application/platform contract; a new type spelling is not implied.

### DI-09d — Action response and conditional UI targets

**Observed:** Endpoint identity, forms, response statuses and HTMX target IDs are separate strings. A target ID's validity does not prove that element is present in conditional DOM. Current source can return a 503 feedback fragment while the default HTMX swap policy excludes 503, so the fragment may never appear in the intended target ([program Gate 3](../can-recommendation-program-2026-09-24.md#gate-3--finish-one-server-driven-product-flow), [SaaS §6](../saas-language-review-2026-09-24.md#6-i-would-recommend-can-if-assertions-described-observable-product-behavior-more-naturally)).

**Outcome and candidates:** Connect action results for success/validation/conflict/forbidden/unavailable to response rendering and target behavior; test visible outcomes in a browser/protocol scenario. Explore typed target references only to their actual guarantee. Distinguish static references from conditional presence. **Authority/gap:** current HTMX server model and policy remain ([decisions: initial web frontend](../decisions.md#initial-web-frontend-server-rendered-html-with-htmx)); no endpoint/DOM proof system is selected.

### DI-10 — Safe HTML authoring and static structure checks

**Observed:** Existing renderers use many tag/attribute/URL/node builder calls and error arms even for literal structure. Builders preserve safety, but statically known structure is validated at runtime. Named render functions already give a component idiom ([SaaS §3](../saas-language-review-2026-09-24.md#3-i-would-recommend-can-if-safe-ui-construction-were-substantially-more-direct), [clean-room §8](../clean-room-language-review-2026-09-24.md#8-connect-frontend-contracts-then-choose-the-frontend-scope)).

**Outcome and candidates:** Compare an improved named-component helper library with checked declarative HTML/components that lower to the same safe native representation. Require static illegal structure/attributes to fail, hostile text to remain inert, dynamic data to keep runtime validation, and reusable field/table/message components. Broader accessible element support is a separate catalogue issue in DI-11c. Arbitrary raw markup and script do not follow. **Authority/gap:** current safe HTML boundary remains; no template grammar selected. **Agent reassessment:** “more direct,” line count and human readability need agent-generation/repair evidence and semantic error-rate measurements.

### DI-11a — Product scope: server-driven milestone and Can browser target

**Observed:** The selected Bun/server-rendered HTMX model serves forms, search and dashboards but cannot express a Can-authored offline editor, local draft, custom browser event handling or rich optimistic interaction. This is a deliberate scope boundary, not an implementation defect ([SaaS §1](../saas-language-review-2026-09-24.md#1-i-would-recommend-can-if-rich-browser-behavior-had-a-coherent-can-model), [clean-room §8](../clean-room-language-review-2026-09-24.md#8-connect-frontend-contracts-then-choose-the-frontend-scope)).

**Outcome and candidates:** Decide the final application claim and interim milestones. Compare Can browser execution with a supported typed bridge to an established browser language on the editable invoice grid: keyboard input, local drafts, immediate totals, optimistic save, rollback and offline/slow connection. The bridge supports a narrower Can-server recommendation; a Can-authored rich-frontend claim requires Can code running in the browser. A successful grid qualifies a class of applications, not every SaaS product. **Authority/gap:** [U7](../decisions.md#initial-web-frontend-server-rendered-html-with-htmx) excludes browser Can from initial scope; Gate 5 is a proposed new product decision. No browser target or syntax is selected. **Agent reassessment:** evaluate agent ability to build and maintain the full flow, not familiarity with existing browser frameworks.

### DI-11b — Browser capabilities, shared contracts and separation

**Observed:** Current Can execution and runtime are Bun/server focused. Shared nominal data need explicit wire codecs; server secrets, SQL pools, process access and service catalogue operations are unsuitable in browser code ([program Gate 5](../can-recommendation-program-2026-09-24.md#gate-5--support-a-can-authored-rich-client), [SaaS §1](../saas-language-review-2026-09-24.md#1-i-would-recommend-can-if-rich-browser-behavior-had-a-coherent-can-model)).

**Outcome and candidates:** Explore a browser-safe capability profile, shared nominal/wire contracts, named state transitions, event handlers, asynchronous commands, renderers and disposal/cancellation using native browser operations plus small contract adapters. Specify browser/version support and how a shared route/data-shape change propagates. Do not assume a full frontend framework, implicit effects or Bun server modules in browser bundles. **Authority/gap:** controlled catalogue and native lowering remain ([decisions: Can-to-Bun boundary](../decisions.md#can-to-bun-boundary), [LD32](../../implementation/language-design-dispositions-2026-09-22.md#assertions-resources-and-platform-contracts)); browser architecture is open.

### DI-11c — Accessible client surface and lifecycle qualification

**Observed:** Current admitted HTML excludes dialog, canvas and SVG; HTMX offers a small trigger/replacement surface. A component disappearing while async work is live needs an ownership and cleanup policy ([SaaS §1](../saas-language-review-2026-09-24.md#1-i-would-recommend-can-if-rich-browser-behavior-had-a-coherent-can-model), [program Gate 5](../can-recommendation-program-2026-09-24.md#gate-5--support-a-can-authored-rich-client)).

**Outcome and candidates:** Inventory actual accessible elements, controls and browser APIs required by the grid; test keyboard operation, failure/rollback, reconnection, navigation and disposal. Choose minimal capabilities from real scenarios and native browser equivalents. Dynamic DOM presence and cancellation must be specified honestly. **Authority/gap:** current HTML catalogue remains; no added elements, lifecycle or accessibility contract is adopted.

### DI-12a — Fallible sequencing and local recovery

**Observed:** The native AI example nests success continuations and cleanup. `match chain` already sequences fallible calls and shared handlers, so nesting alone does not prove a missing syntax. Ordinary `match call`/`match chain` are terminal while coordination has local result handling; local recovery then continuation remains a design question ([SaaS §5](../saas-language-review-2026-09-24.md#5-i-would-recommend-can-if-fallible-workflows-and-cleanup-stayed-readable-as-they-grew)).

**Outcome and candidates:** Rewrite a real invoice or AI workflow using current `match chain`, named helpers and scoped operations. If measured cost remains, compare a narrow sequencing/recovery form. Preserve original domain-error distinctions; coordinate with DI-03. **Authority/gap:** [LD19](../../implementation/language-design-dispositions-2026-09-22.md#failure-contracts-and-coordination) retains chain and distinct coordination meanings; [LD20](../../implementation/language-design-dispositions-2026-09-22.md#failure-contracts-and-coordination) defers new syntax under a demonstrated-need gate. **Agent reassessment:** deep nesting or shortness matters only if it harms agent work or outcomes.

### DI-12b — Explicit cleanup and failure precedence

**Observed:** Stream `pump` claims explicit reader closure on every terminal path, but read-error arms relay without `close_reader`; owner enforcement supplies fallback cleanup and treats omitted explicit closure as failure. This is an example/contract inconsistency, not proof of an unguarded leak ([SaaS §5](../saas-language-review-2026-09-24.md#5-i-would-recommend-can-if-fallible-workflows-and-cleanup-stayed-readable-as-they-grew), [program Gate 4](../can-recommendation-program-2026-09-24.md#gate-4--qualify-the-service-that-runs-the-flows)).

**Outcome and candidates:** Correct the example or wording after selecting its intended policy; inject failure at every stage and define original-error versus cleanup-error precedence and exactly-once closure observation. Compare current scoped APIs before proposing a new control form. **Authority/gap:** runtime owner model is retained by [LD30](../../implementation/language-design-dispositions-2026-09-22.md#assertions-resources-and-platform-contracts); production correction belongs later, not in this preparation inventory.

### DI-13 — Observable product assertions and scenario reuse

**Observed:** Bare `=> ok` on opaque responses proves completion, not status/body/headers, escaped content, target swap or every external side effect. Native raw fixtures exercise preparation/decoding, while browser integration remains a separate observation. Source-level 503/HTMX mismatch illustrates the gap ([SaaS §6](../saas-language-review-2026-09-24.md#6-i-would-recommend-can-if-assertions-described-observable-product-behavior-more-naturally), [clean-room §5](../clean-room-language-review-2026-09-24.md#5-give-test-scenarios-explicit-ownership-across-calls)).

**Outcome and candidates:** Retain attached assertions and lexical fixture guarantees while adding or arranging protocol/browser observations for 200/422/409/503, headers, escaping, retained fields, tenant isolation, duplicate webhook, target behavior and resource closure. Compare companion tests and authored assertion forms; do not claim universal proof. Property/generated cases are later coverage options where useful. **Authority/gap:** [LD22–LD29](../../implementation/language-design-dispositions-2026-09-22.md#assertions-resources-and-platform-contracts) separately address publication, deadlines, native behavior, templates and failed checks; DI-06 tracks fixture ownership. Need observation-strength contract and end-to-end fixtures.

### DI-14a — SQL mutation results

**Observed:** Transactions and `commit_unknown` exist; row-returning descriptors require SELECT, so mutation `RETURNING` is rejected. A save then render canonical data may need another read despite being in a transaction ([SaaS §7](../saas-language-review-2026-09-24.md#7-i-would-recommend-can-if-common-saas-operations-fit-the-supported-platform-boundary), [clean-room §9](../clean-room-language-review-2026-09-24.md#9-state-the-exact-sql-guarantee-and-remove-demonstrated-friction)).

**Outcome and candidates:** Compare existing two-step transaction with bounded, dialect-aware mutation-returning descriptors for ledger/outbox and invoice saves. Specify finite cardinality, error/row codecs and native SQL behavior. This is a targeted platform extension, not an ORM. **Authority/gap:** current descriptor rules remain; no `RETURNING` contract selected. The real-database webhook flow should demonstrate the need.

### DI-14b — SQL schema-checking promise

**Observed:** Descriptors check statement shape, parameters, dialect/cardinality and supported codecs, but do not prove that declared projection matches a versioned database schema; runtime row validation catches mismatch. A current positive checker test admits `SELECT id` with a wider declared row ([clean-room §9](../clean-room-language-review-2026-09-24.md#9-state-the-exact-sql-guarantee-and-remove-demonstrated-friction)).

**Outcome and candidates:** State the current guarantee accurately. If compile-time schema agreement is desired, evaluate a versioned schema snapshot/build input, migration drift and deployed-database verification separately. Runtime codecs and schema snapshots solve different problems. **Authority/gap:** current compiler-validated SQL descriptors are retained by [LD35](../../implementation/language-design-dispositions-2026-09-22.md#assertions-resources-and-platform-contracts); no compile-time database-shape promise is adopted.

### DI-15a — Owned work, abortability and shutdown

**Observed:** Race losers retain leases until settlement. Permanently pending work can prevent root drain. An early winner does not cancel losers; runtime ownership still enforces resource lifetime ([SaaS §7](../saas-language-review-2026-09-24.md#7-i-would-recommend-can-if-common-saas-operations-fit-the-supported-platform-boundary), [clean-room §10](../clean-room-language-review-2026-09-24.md#10-keep-lifetime-and-coordination-promises-precise)).

**Outcome and candidates:** Qualify slow dependencies, client disconnect, pending owned work, close deadlines and bounded host shutdown. Inventory which native operations can abort, which cannot, and host supervision guarantees. Explicit cancellation for supported operations is a candidate; arbitrary Can computation must not be promised cancellable. **Authority/gap:** [LD30](../../implementation/language-design-dispositions-2026-09-22.md#assertions-resources-and-platform-contracts) retains owners/leases; no implicit loser cancellation is selected.

### DI-15b — Empty dynamic races

**Observed:** `race with error` delegates to `Promise.race`; an empty dynamic expansion remains pending. First-success `race` delegates to `Promise.any` and has a different empty outcome. This is selected native behavior, not a defect inferred from hanging tests ([clean-room §10](../clean-room-language-review-2026-09-24.md#10-keep-lifetime-and-coordination-promises-precise)).

**Outcome and candidates:** Make nonempty construction or explicit guards visible in real programs and assert the native outcome. A production semantic change is a separate policy decision. **Authority/gap:** [LD24](../../implementation/language-design-dispositions-2026-09-22.md#assertions-resources-and-platform-contracts) retains pending empty-race semantics; [LD23](../../implementation/language-design-dispositions-2026-09-22.md#assertions-resources-and-platform-contracts) accepts external assertion deadlines.

### DI-15c — Service race fault visibility

**Observed:** First-success race consumes failures before the winner and reports late standard failures after it. A successful fallback may hide a replica failure; reporting all consumed failures may add noise ([clean-room §10](../clean-room-language-review-2026-09-24.md#10-keep-lifetime-and-coordination-promises-precise)).

**Outcome and candidates:** Test a replica/fallback service and determine whether opt-in consumed-fault observation is useful while preserving native winner selection and ordinary fallback behavior. Specify diagnostic timing and ownership. **Authority/gap:** [LD19](../../implementation/language-design-dispositions-2026-09-22.md#failure-contracts-and-coordination) retains coordination meanings; no observability mode selected.

### DI-16 — Linux packaging and runtime qualification

**Observed:** The pinned qualified distribution target is Bun on Darwin ARM64; current documentation declines Linux support. Passing source tests does not qualify a Linux release ([SaaS §7](../saas-language-review-2026-09-24.md#7-i-would-recommend-can-if-common-saas-operations-fit-the-supported-platform-boundary), [program Gate 4](../can-recommendation-program-2026-09-24.md#gate-4--qualify-the-service-that-runs-the-flows)).

**Outcome and candidates:** Decide supported Linux target/version, package, install, native capability and graceful-shutdown acceptance, then run invoice and webhook slices there. This is distribution/qualification, not grammar. **Authority/gap:** [decisions: initial distribution](../decisions.md#initial-distribution-scope) commits macOS initially and leaves Linux possible; the recommendation program treats Linux as a proposed SaaS deployment gate. No Linux target is selected.

### DI-17a — Native AI wrapper recovery provenance

**Observed:** Provenance-aware `wrap` supports fetch/judge, while LLM callers use ordinary wrappers. Shared error kind alone cannot let an ordinary wrapper distinguish authored versus decoder-origin failure ([clean-room §11](../clean-room-language-review-2026-09-24.md#11-finish-native-ai-composition-through-concrete-workflows)).

**Outcome and candidates:** Test a real LLM recovery workflow before extending native wrapper policy to LLM, preserving grouped state, explicit public bounds and origin-specific recovery. Ordinary composition remains the baseline. **Authority/gap:** [LD07–LD10](../../implementation/language-design-dispositions-2026-09-22.md#failure-contracts-and-coordination) select fetch/judge normalization/wrap boundaries and defer automatic LLM extension; native AI declarations remain foundational ([decisions: design direction](../decisions.md#design-direction)).

### DI-17b — Native AI batch handler effects

**Observed:** A judge batch validates responses before handlers, then executes handlers in source order; earlier handler effects are not rolled back if a later one fails. All-or-nothing returned-record construction is not a transaction over prior writes ([clean-room §11](../clean-room-language-review-2026-09-24.md#11-finish-native-ai-composition-through-concrete-workflows)).

**Outcome and candidates:** Document preparation, validation, handler and returned-result phases; test failing second handler after an observable first effect. If the workflow requires atomic effects, compare explicit transaction/outbox idioms before changing batching. **Authority/gap:** [LD08](../../implementation/language-design-dispositions-2026-09-22.md#failure-contracts-and-coordination) retains judge batching; no rollback promise exists.

### DI-17c — Grouped-state callable composition

**Observed:** Judge/LLM grouped signatures are not ordinary callables directly; named wrappers work. The adapter cost is real, but a dynamic fan-out example demonstrating material cost is absent ([clean-room §11](../clean-room-language-review-2026-09-24.md#11-finish-native-ai-composition-through-concrete-workflows)).

**Outcome and candidates:** Benchmark named wrappers on real dynamic fan-out. Only if needed, prototype a grouped-state-preserving callable covering references, capture, fixtures and error bounds. Do not flatten grouped state, merge native kinds or infer public errors. **Authority/gap:** [LD05](../../implementation/language-design-dispositions-2026-09-22.md#native-ai-state-and-connections) retains grouped state; [LD06](../../implementation/language-design-dispositions-2026-09-22.md#demonstrated-need-gate-for-larger-abstractions) defers callable redesign pending demonstrated need. **Agent reassessment:** measure adapter mistakes/edit propagation, not human distaste for wrappers.

### DI-18a — Scoped results carrying expired resources

**Observed:** `with_transaction<T>` can return an aggregate containing its own closed transaction; later use raises `resource_state` before native work. An enclosing-owned pool may legitimately be returned. This is a runtime-enforced lifetime limitation, not a safety bypass ([clean-room §10](../clean-room-language-review-2026-09-24.md#10-keep-lifetime-and-coordination-promises-precise)).

**Outcome and candidates:** Explore only bounded diagnostics for obvious escapes of the scope's own handle, with returned enclosing-resource counterexamples. A blanket ban on resource-valued results and a general borrow checker are unsupported. **Authority/gap:** [LD30](../../implementation/language-design-dispositions-2026-09-22.md#assertions-resources-and-platform-contracts) retains runtime ownership; [LD31](../../implementation/language-design-dispositions-2026-09-22.md#assertions-resources-and-platform-contracts) defers static escape diagnostics until useful case and sound bounded analysis are shown.

### DI-18b — Race fault observation

This is the same underlying first-success failure-timing issue as DI-15c, retained here as an explicit crosswalk alias. There is **one** design question, one disposition and one evidence packet; service qualification and scoped/lifetime review both consume it ([program crosswalk](../can-recommendation-program-2026-09-24.md#consolidated-crosswalk)).

### DI-19 — Controlled catalogue and external SDKs

**Observed:** Typed fetch, crypto and processes cover many protocols. Project-authored arbitrary SDK bindings are excluded by the distribution-controlled catalogue. No required SDK was shown impossible or materially impractical in the source reviews ([SaaS §7](../saas-language-review-2026-09-24.md#7-i-would-recommend-can-if-common-saas-operations-fit-the-supported-platform-boundary), [clean-room §12](../clean-room-language-review-2026-09-24.md#12-give-immutable-collections-efficient-construction-paths)).

**Outcome and candidates:** Attempt a real payment/email/queue integration with current HTTP/process facilities; only a reproduced material shortfall should trigger a narrow typed adapter contract with codecs, immutable results, lifetime, error and deterministic fixture behavior. Distribution catalogue extension is another option. Arbitrary JS/Bun imports are not justified. **Authority/gap:** [LD32](../../implementation/language-design-dispositions-2026-09-22.md#assertions-resources-and-platform-contracts) retains the closed boundary; [LD33](../../implementation/language-design-dispositions-2026-09-22.md#assertions-resources-and-platform-contracts) accepts an admission process; [LD34](../../implementation/language-design-dispositions-2026-09-22.md#assertions-resources-and-platform-contracts) defers third-party manifests. Native lowering is mandatory ([decisions: native operations](../decisions.md#reuse-native-operations-do-not-reimplement-them)).

### DI-20 — Bulk immutable collection construction

**Observed:** Individual map insertion/replacement and set addition clone native collections. A word-count fold adding unique keys performs a sum of prior sizes, hence quadratic copying by source-level complexity; no throughput benchmark was reported ([clean-room §12](../clean-room-language-review-2026-09-24.md#12-give-immutable-collections-efficient-construction-paths)).

**Outcome and candidates:** Benchmark realistic batch sizes, then compare native bulk construction/grouping/aggregation with explicit duplicates, callback ordering/failures and one immutable publication. An internal scoped builder is a fallback; general mutable aliases and a custom persistent-map engine are not first answers. **Authority/gap:** immutable values and native JS/Bun delegation remain ([decisions: native operations](../decisions.md#reuse-native-operations-do-not-reimplement-them)); this is a later item and no adapter API is selected.

### DI-21 — Billing/calendar and domain-library boundaries

**Observed:** Exact minor-unit arithmetic, half-even rounding and zoned civil-time resolution already exist; realistic subscription renewal/refund rules have not been qualified in a maintained end-to-end example ([SaaS §7](../saas-language-review-2026-09-24.md#7-i-would-recommend-can-if-common-saas-operations-fit-the-supported-platform-boundary)).

**Outcome and candidates:** After the core flow, maintain a domain-library/example scenario for short months, leap years, DST policy, currencies, taxes and refunds. Distinguish library gaps and policy choices from language semantics. Dedicated queue/billing/auth keywords are unsupported. **Authority/gap:** current exact numeric/time operations remain; proposed later work has no selected syntax or platform extension.

### DI-22 — Named-field construction and reorder safety

**Observed:** Adjacent same-typed positional record fields such as `total` and `recent` can be swapped while remaining type-correct. This is a plausible edit hazard; the review did not demonstrate a production misbinding or agent failure ([SaaS §8](../saas-language-review-2026-09-24.md#8-i-would-recommend-can-more-readily-if-everyday-edits-were-less-fragile)).

**Outcome and candidates:** Reorder fields across a realistic project and compare current positional constructors, stronger domain types and named-field construction. Measure silent behavior changes and diagnostics before selecting syntax. **Authority/gap:** positional record construction remains ([decisions: records](../decisions.md#records)); Gate 3 proposes a trial, not a grammar change. **Agent reassessment:** this topic has a concrete agent-edit correctness question; do not select named fields because they look more readable to a person.

### DI-23a — Multiline layout, formatter and diagnostics

**Observed:** Calls, constructors and assertion rows have one-physical-line restrictions; the SaaS review counted 154 example lines over 120 characters, max 336, while the gallery had none. Those measurements describe human presentation pressure, not a semantic failure or token cost. Existing `parse --render` is a renderer, and [LD45](../../implementation/language-design-dispositions-2026-09-22.md#syntax-documentation-and-authoring-feedback) accepts a comment-preserving canonical formatter; [LD44](../../implementation/language-design-dispositions-2026-09-22.md#syntax-documentation-and-authoring-feedback) accepts precise diagnostic spans ([SaaS §8](../saas-language-review-2026-09-24.md#8-i-would-recommend-can-more-readily-if-everyday-edits-were-less-fragile), [clean-room §7](../clean-room-language-review-2026-09-24.md#7-make-captures-and-layout-easier-to-refactor)).

**Outcome and candidates:** After formatter/builder improvements, measure agent generation, parsing and repair on representative long forms. Compare continuation inside explicit delimiters while retaining block indentation and attached assertion semantics. No multiline grammar selected. **Authority/gap:** [LD46](../../implementation/language-design-dispositions-2026-09-22.md#syntax-documentation-and-authoring-feedback) defers multiline forms pending remaining pressure and a consistent layout proposal. **Agent reassessment required:** normal review width, visual readability and human comfort cannot be the deciding metric; source tokens are not assumed to improve with fewer lines.

### DI-23b — Unnecessary-local validity rule

**Observed:** The compiler rejects a narrowly defined immediately forwarded local even though the program can be semantically valid; `permitted` is one example of a potentially meaningful local. The source review proposes moving this style pressure toward formatter/lint advice ([SaaS §8](../saas-language-review-2026-09-24.md#8-i-would-recommend-can-more-readily-if-everyday-edits-were-less-fragile), [clean-room §7](../clean-room-language-review-2026-09-24.md#7-make-captures-and-layout-easier-to-refactor)).

**Outcome and candidates:** Compare current rejection with advisory guidance on an agent-edit benchmark, including whether the local carries useful domain meaning or supports refactoring. Do not change validity solely for human style preference. **Authority/gap:** current local rule remains; no distinct September 22 disposition specifically selects its replacement. This is a later, lower-priority item. **Agent reassessment required.**

## Cross-topic evidence and disposition backlog

The [program gates](../can-recommendation-program-2026-09-24.md#ordered-gates) give useful outcome checkpoints, but their ordering must be tested against dependencies. DI-01, DI-05 and DI-06 affect whether later library and assertion evidence is reliable. DI-02/03/04/05 meet at independent package composition. DI-09/10/13/14/22 meet in the invoice and webhook flows. DI-11 depends on shared wire/route contracts and browser capability separation. DI-12/15/18 meet in cleanup/shutdown qualification. DI-17 and DI-20 are later workflow and scale trials. This is a dependency hypothesis, not an implementation schedule ([preparation P4 and P10](../can-design-preparation-2026-09-24.md)).

The source reviews also preserve valuable baselines: nominal immutable records, closed variants and substantive exhaustiveness, explicit finite public domain errors, named functions and immutable captures, attached assertions and gated publication, exact arithmetic/conversion, native AI forms with grouped state, native Promise selection with owners, safe HTML and bounded SQL, controlled capabilities, and native JavaScript/Bun lowering ([clean-room preservation table](../clean-room-language-review-2026-09-24.md#what-i-would-preserve), [program language direction](../can-recommendation-program-2026-09-24.md#one-language-direction), [September 22 ledger](../../implementation/language-design-dispositions-2026-09-22.md)). None of the findings supplies a demonstrated need for `any`, implicit coercion, unrestricted backend imports, general inheritance, a broad effect system, general mutable aliases or replacing the owner model. Preserve or revise each through explicit evidence, not compatibility inertia.

Before a final implementation list, every ID needs a recorded **accept, retain, reject or defer** decision, with a reason and reopening condition for deferrals. The particularly clear deferred/later candidates are DI-03 finite rows, DI-08 capture syntax, DI-17c grouped-state callables, DI-18a static resource diagnostics, DI-19 third-party binding manifests, DI-20 bulk adapters pending scale evidence, DI-21 domain libraries, and DI-23 layout/style. Earlier deferred LD items remain deferred unless explicitly changed. Research, three freshly worded Jev consultations where the technical choice is difficult, syntax resolution, experiments and acceptance contracts occur in later preparation stages; the user's later instruction bars further questions, so remaining syntax uses Jev advice and recorded engineering judgment ([preparation P3–P10](../can-design-preparation-2026-09-24.md)).
