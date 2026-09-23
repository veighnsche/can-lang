# When I would recommend Can for full-stack SaaS

**Consolidated recommendation:** [One recommendation program for Can](can-recommendation-program-2026-09-24.md) merges this review with the independent clean-room review and supplies the current proposed priority and acceptance gates. This source review remains evidence; it is not a second backlog.

Review date: 24 September 2026. Source revision: `4c2db1e`.

This is a recommendation review under the user's broader SaaS ambition, not an approved syntax specification or implementation plan. Earlier retained decisions remain intact; deferred designs are not promoted by appearing here. There are no compatibility requirements.

I would recommend Can broadly when its contracts connect the whole application: a browser action, validated business input, authorized operation, database change, and observable response. Today there are strong contracts inside individual subsystems. Some relationships between those subsystems still depend on strings, conventions, repeated handlers, or tests outside the source-level assertion surface.

The most consequential change in ambition is the frontend. Can currently targets server-driven HTML/HTMX applications. Supporting rich browser applications requires a deliberate expansion of that product boundary.

## What was studied

All 56 `.can` files in all 16 projects under `examples/` were read, together with their relevant READMEs and configuration: account-search, cookies, crypto, dashboard, files, form-validation, gallery, language-site, markdown, mysql, native-ai, process, sqlite, stream, utilities, websocket. This is 3,146 lines of Can. Three independent reviews covered the frontend, backend and core language, reconciled against current checker/runtime code and relevant tests. See the [inventory and consultation evidence](evidence/2026-09-24/saas-review/README.md).

Historical documents sometimes retain old status text. Implementation evidence takes precedence: the current compiler does have formatting, normalized fetch/judge errors, wrapper contracts, fixture templates, supervised assertion execution and assertion-gated publication. These are not proposed here as missing features.

Can also already has nominal records, variants, exhaustive matches, generics, named higher-order functions, immutable collections, safe HTML, typed HTTP decoding, transactions, three SQL dialects, concurrency, streams, WebSockets, crypto, cookies, zoned time and exact integer-based financial rounding. The presence or absence of a small example is not a capability test.

## 1. I would recommend Can if rich browser behavior had a coherent Can model

**Impact: essential to the expanded frontend promise. Layer: language target and UI runtime.**

The current [HTMX surface](../../runtime/platform/html.ts#L353) supplies get/post, replacement targets, a small trigger vocabulary and polling. The [account search renderer](../../examples/account-search/src/render/render.can#L52) and [dashboard renderer](../../examples/dashboard/src/render/render.can#L39) use those mechanisms. Named render functions already provide component composition. What is missing is a Can browser execution target for local state, event handling and lifecycle. The admitted HTML surface also excludes elements such as dialog, canvas and SVG ([tag inventory](../../runtime/platform/html.ts#L64)).

I would want a typed local model, named event handlers, state transitions, asynchronous commands, and clear ownership when a component disappears. Server secrets and server capabilities must remain unavailable to browser code. Shared types and validation should have explicit wire contracts. Existing server rendering and HTMX remain useful; they do not need to be discarded.

**Acceptance scenario:** a tenant's editable invoice grid supports keyboard navigation, local unsaved drafts, immediate calculated totals, optimistic saves, a failed-save rollback and useful state during a slow connection, with behavior authored in Can. Native browser operations should implement the result. This does not imply shipping Bun-dependent server runtime modules to the browser.

This reopens the deliberate server-driven product boundary. It is not a defect in the implementation of that boundary.

## 2. I would recommend Can if routes, actions and forms were connected by types

**Impact: high. Layer: compiler and web APIs.**

The URL `/signup/validate` is constructed in the [renderer](../../examples/form-validation/src/render/render.can#L46) and separately mounted in the [server](../../examples/form-validation/src/web/web.can#L53). HTMX receives a safe URL, but that type establishes URL safety, not membership in a router or agreement with an endpoint's request contract. Form control names are likewise separate strings.

Typed JSON and form decoding already exist. The next step is a checked relationship between an endpoint, its input, its result, its URL builder and a form/action that invokes it. Renaming an endpoint or field should expose every affected use at compile time. A browser action should have explicit success, validation, conflict and unavailable outcomes.

Path parameters also matter: the [route checker](../../compiler/internal/check/http.go#L336) rejects `:id` and `*`, while runtime dispatch is an exact path map. The [form schema checker](../../compiler/internal/types/form_schema.go#L18) accepts only shallow `str`, `str[]` and `option::value<str>` members. Numeric or boolean values can be parsed manually; the limitation is the absence of a reusable richer form projection and linked validation contract.

**Acceptance scenario:** `/tenants/:tenant_id/invoices/:invoice_id` has typed parameters and a generated safe URL builder. An invoice form has numeric seats, optional details and repeated line items, retains rejected values and reports field-specific errors. A route or field rename produces precise diagnostics in its consumers. A shared authentication guard supplies an explicit validated principal/tenant context before database work.

This should not infer authorization from a type name. The guard must perform the actual authorization, and business capabilities must remain scoped to the request and tenant.

## 3. I would recommend Can if safe UI construction were substantially more direct

**Impact: high. Layer: static UI checking and catalogue design.**

The [signup page](../../examples/form-validation/src/render/render.can#L38) and [account page](../../examples/account-search/src/render/render.can#L52) express simple markup through many calls to create tags, attributes, URLs and nodes. Known literal structure goes through [fallible runtime builders](../../runtime/platform/html.ts#L194), and renderers and HTTP callbacks handle the resulting structure errors.

Can's HTML safety is valuable. A more direct typed component or template form could check static structure during compilation, preserve automatic escaping, and leave runtime validation for genuinely dynamic data. The exact spelling is secondary. Existing named render functions should compose naturally with any new surface.

**Acceptance scenario:** reusable form field, button, table and validation-message components accept typed properties. Static illegal nesting and attributes fail at compile time. Hostile text remains inert. The simple signup page no longer requires a long sequence of fallible calls for literal markup. Compare an improved library of current builders with a checked template prototype before choosing new syntax.

Broader accessible HTML coverage is a related catalogue task, not a reason to admit arbitrary script or raw markup.

## 4. I would recommend Can if applications could enforce their own valid value construction

**Impact: high. Layer: core abstraction and serialization.**

Nominal records prevent mixing unrelated declared types. Immutability prevents mutation after construction. Neither guarantees that a publicly constructible record contains a valid business value. The [ordinary record/update rules](technical-spec.md#c4) and wire decoding permit construction without calling an application validator. Protected provenance currently belongs to catalogue types such as `html::safe` and resource handles ([platform contract](platform-testing-spec.md#p6)).

A validated email, positive seat count, money amount with currency and scale, or authorized tenant handle should be usable as a type whose construction has a controlled owner. A named factory alone does not establish that guarantee if callers can construct or copy-update the same exported record directly.

**Acceptance scenario:** a package exports an email type and validation operation. Another package cannot forge an invalid value by construction, `with`, or JSON decoding. Decoding returns an explicit validation outcome. Fixtures can test the validator without acquiring a general escape from its rules. An authorized handle's validity must also account for request/tenant lifetime; opacity alone is not an authorization system.

Evaluate package-controlled constructors or application-owned opaque values using this concrete benchmark. This need was previously deferred, so the recommendation is to establish and prototype the contract, not to silently select a new keyword or refinement/proof system.

## 5. I would recommend Can if fallible workflows and cleanup stayed readable as they grew

**Impact: high. Layer: core composition plus scoped platform APIs.**

The [native AI entry point](../../examples/native-ai/src/app/main.can#L5) combines AI, SQL, encoding, output and closure through deep success continuations and repeated cleanup. `match chain` already supports ordered fallible calls and shared handlers. A fair comparison must first rewrite a real workflow using that feature and named helpers; nesting in an example does not prove that new sequencing syntax is necessary.

The remaining design question is local recovery followed by continued work, together with one clear cleanup policy. Ordinary `match call` and `match chain` are terminal; coordination has a local-result form. Built-in collections also preserve a callback's finite error set, while authored higher-order functions use concrete bounds. These asymmetries deserve evaluation in a real transaction, retry, authorization or audit helper, not an abstract desire for more generics.

There is concrete cleanup evidence in the examples: [stream `pump`](../../examples/stream/src/main.can#L5) says every terminal path closes the reader, but its read-error arms forward without calling `close_reader`; the explicit close is only on EOF. The [runtime read path](../../runtime/transport/stream/readable.ts#L103) marks read failures without replacing that explicit-close obligation. [Ownership enforcement](../implementation/ownership.md#L38) performs fallback cleanup and treats omitted explicit closure as failure. This is an example inconsistency and a useful ergonomics case, not a claim that Can leaks resources without enforcement.

**Acceptance scenario:** validate an invoice command, authorize its tenant, perform a transactional update, publish a result, and close resources. Inject failure at every stage. Preserve the original failure and report cleanup failures according to an explicit policy. Shared helpers retain their callers' error distinctions without duplicating an entire infrastructure error list.

Start with the best current chain/helper solution and scoped library operations. Only reopen sequencing or error-set parameters if measurable costs remain. Preserve the [existing demonstrated-need gate](../implementation/language-design-dispositions-2026-09-22.md#demonstrated-need-gate-for-larger-abstractions).

## 6. I would recommend Can if assertions described observable product behavior more naturally

**Impact: high. Layer: assertion observations, reusable scenarios and examples.**

Mandatory assertions and verified publication are useful. Their guarantee is that the authored assertions pass, not that the application meets every business requirement.

Opaque HTTP response assertions commonly use `=> ok`. This limitation is explicit in the [assertion checker](../../compiler/internal/check/assertions.go#L38): those rows do not compare status, body or headers. External integration and browser tests already supply stronger observations, and native raw fixtures already exercise real request preparation and decoding. The opportunity is to make meaningful protocol and lifecycle assertions easy to author and reuse alongside Can code.

A concrete cross-boundary mismatch illustrates the need. The [account search handler](../../examples/account-search/src/web/web.can#L46) returns a 503 HTML feedback fragment for SQL failures. The [default HTMX configuration](../../runtime/platform/html.ts#L399) excludes 503 from swaps. Returning an error fragment therefore does not itself display that feedback in the target. The source-level success expectation does not establish the intended browser result. This is a source-verified policy interaction; no new browser reproduction was run during this review.

**Acceptance scenario:** attached or reusable route scenarios check 200/422/409/503 status, headers, escaped content, target behavior and retained form values. Stream failure scenarios verify exactly-once closure and error preservation. Shared tests exercise tenant isolation and duplicate webhook handling. Property-based or generated cases can be evaluated later where they add coverage, without replacing attached examples or claiming universal proof.

## 7. I would recommend Can if common SaaS operations fit the supported platform boundary

**Impact: adoption-critical. Layer: catalogue, runtime and deployment, mostly not core syntax.**

Several requirements should be tracked separately from language grammar:

| Requirement | Current evidence | Recommendation gate |
| --- | --- | --- |
| Atomic writes with useful results | Transactions and `commit_unknown` exist. [SQL admission](../../compiler/internal/sql/cardinality.go#L29) requires SELECT for row-returning descriptors and rejects mutation RETURNING. | A PostgreSQL or SQLite webhook transaction can insert a ledger/outbox entry and obtain typed generated results with a finite row bound. Repeated delivery and uncertain commit are reconciled explicitly. |
| Request and shutdown lifetimes | Owned work and close deadlines exist. An early race winner does not cancel losers; a pending owner can keep drain pending. | Demonstrate slow dependency handling, client disconnect and bounded shutdown. Specify which native operations can actually abort and what supervision guarantees when they cannot. Do not change race semantics implicitly. |
| Billing/calendar helpers | Exact minor-unit arithmetic, half-even rounding and zoned civil-time resolution already exist. | A maintained example covers monthly renewal, short months, leap years, DST policies, currencies, taxes and refunds. Most of this belongs in domain libraries. |
| External integrations | Typed fetch, crypto and processes already cover many protocols. Arbitrary project-authored SDK bindings are excluded by the controlled catalogue. | Implement a required payment/email/queue integration using the best current route. Only a reproduced substantial shortfall should trigger an adapter proposal, with codecs, immutable results, lifetimes and deterministic fixtures. |
| Linux deployment | [The qualified target](../../distribution/target.json#L3) is Bun on Darwin ARM64. The README explicitly declines Linux support. | Qualify Linux build/run and the supported runtime capabilities, including server behavior and graceful shutdown. This is an immediate gate for recommending Can for Linux-hosted SaaS. |

The broad SaaS objective makes extensibility worth investigating. It does not make unrestricted npm imports the correct design, and there is no reproduced blocked SDK in this review. Likewise, a queue, billing system or auth provider does not each need a dedicated language keyword.

## 8. I would recommend Can more readily if everyday edits were less fragile

**Impact: useful after the structural priorities. Layer: grammar, construction and tooling.**

Two small design experiments have concrete motivation:

- Named-field construction: [`triage_report`](../../examples/native-ai/src/records/records.can#L11) contains adjacent `int total` and `int recent` fields, passed positionally [here](../../examples/native-ai/src/app/main.can#L61). Swapping them preserves type correctness. Named construction and stronger domain types address different parts of this risk. Measure a record-field reorder across a realistic project before selecting syntax.
- Multiline argument/constructor/assertion layout: the current [grammar](technical-spec.md#c2) keeps those forms on one physical line. The examples contain 154 lines longer than 120 characters and a maximum of 336 characters, while the small gallery has none. This is authoring/readability evidence, not proof of a semantic flaw. Allowing structured continuation could make real contract-heavy programs easier to review without removing mandatory assertions or explicit errors.

The compiler also rejects a narrowly defined immediately forwarded local even when it is semantically valid ([implementation](../../compiler/internal/check/locals.go#L41)). I would reassess whether such style preferences belong in formatting/lint feedback rather than the language's validity rules. This is a lower-priority design opinion; it has no bearing on nominality, immutability or contract correctness.

## The program that should drive the next design round

Use one cohesive, multitenant subscription/invoicing application as the benchmark. It should have login and roles, typed tenant routes, an editable invoice grid, validated values and field errors, exact monetary calculations, a transaction with an outbox, duplicate webhook handling, a real provider integration, failure-safe resource cleanup, and observable HTTP/browser tests. Build the simplest working version with current Can first.

Then introduce controlled changes: rename a route and form field, reorder same-typed record fields, add a business rejection, make a dependency fail after a write, replay a webhook, disconnect a browser and shut down with work pending. Measure the files touched, compiler diagnostics, repeated contracts, tests needed and behavior preserved.

That experiment would distinguish a library gap from a true language abstraction gap. It would also give the deferred composition, opaque-value and extension discussions concrete evidence. There is no need to redesign all of Can or erase its native AI identity to perform it.

## Consultation and validation

Three fresh Jev System One requests rewrote every explanatory state field, instruction and option description while preserving facts and alternatives. All six selected alternatives agreed: investigate a Can browser model, linked web contracts, protected application values, current-idiom composition measurements, evidence-gated SDK adapters and stronger behavioral observations. Probability varied, especially for the browser and domain-value questions. Agreement is advisory; it neither proves correctness nor removes framing bias. All [requests, responses and wording audit](evidence/2026-09-24/saas-review/README.md) are saved.

Validation run during the review:

- `bun run check:runtime` passed.
- `GOCACHE=/tmp/can-saas-review-go-cache go test ./compiler/internal/syntax ./compiler/internal/check ./compiler/internal/types` passed. The initial attempt with the default cache was blocked by filesystem permissions; relocating the cache resolved it.
- Forty focused current-runtime tests passed for HTML, form decoding, numbers, datetime and transactions; [output](evidence/2026-09-24/saas-review/runtime-tests.txt) is saved. An earlier non-exact filter also selected staged copies; the final saved run targets only current source paths.

This was source and design review, not full release qualification. Live databases/providers, Linux deployment, all example builds and the browser suite were not rerun. No compiler, runtime or example behavior was changed.
