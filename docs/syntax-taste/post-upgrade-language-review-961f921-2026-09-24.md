# Can language design after the upgrade

24 September 2026 · reviewed revision `961f921a6a8cf40be54735683caf29613c19cbd8`

**Can is substantially closer to a credible full-stack SaaS language. I would recommend it more broadly if its new capabilities formed complete, reusable application contracts—from a shared declaration through a deployed server and browser—without application-specific repair work.**

The upgraded language has real browser execution, owner-controlled records, explicit pattern bindings, composable package identities, typed actions and forms, and substantial transactional examples. Those capabilities change the assessment. The remaining priorities are shared action/server binding, a supported browser runtime, ordinary generic composition, and predictable iteration and lifetime behavior. Adding many more isolated language forms would have less value than completing these paths.

For a bounded Bun service or server-rendered application, Can is a credible candidate for an evaluated pilot. I would still withhold a general recommendation as the sole language for arbitrary SaaS frontends and backends. This judgment concerns language and platform fit, not a claim that the existing examples cannot run.

## Review method and scope

Three fresh reviewers started without conversation history and independently inspected core language design, native/backend semantics, and frontend/platform composition. They were explicitly excluded from earlier reviews, recommendation programs, preparation alternatives, consultation results, reconciliation records, and acceptance verdicts during their first pass. Code, current tests, and examples were primary evidence. Current operational documentation was checked against implementation rather than assumed authoritative.

The coordinator retains the preceding conversation. Thus “clean room” describes the three independent initial reviews, not complete historical ignorance by every participant. Only after those reviews were complete were findings reconciled and remedies compared through three freshly worded Jev consultations. Classifier disagreements remain visible; they do not decide the design by vote.

All 19 current example projects were inventoried, covering 66 example `.can` files, plus four maintained standard-library sources. The gallery and standard examples received complete core checks. Application review concentrated on meaningful execution paths, especially invoice, invoice-grid, webhook, native AI, forms, and resource use; the evidence records read depth rather than claiming every line received equal scrutiny. No historical example or planned feature was counted as implemented.

The [evidence index](/Users/vince/Projects/can-lang/docs/syntax-taste/evidence/2026-09-24/post-upgrade-review-961f921/README.md) contains independent reports, source inventory, runnable probes, test logs, exact consultations, and reconciliation. No language implementation was changed.

## The foundation now earns substantially more confidence

| Area | What current source establishes | Remaining qualification |
| --- | --- | --- |
| Pattern matching | Explicit `bind` distinguishes bindings from nominal cases; coverage and usefulness checks remain substantive. | Coverage has finite analysis limits; assertions do not prove every input. |
| Domain values | `owner record` confines construction, projection, updates and destructuring to its owner, and blocks generic codec bypasses. | Automatic equality still depends on hidden representation. |
| Variants | Compatibility consistently follows admitted leaf sets; nominal records provide brands. | A variant name itself is not a distinct runtime brand. |
| Dependencies and errors | Instance-qualified package identities, aliases, lineage and qualified error kinds permit independently named libraries to compose. | Distribution, documentation and version evolution remain separate concerns. |
| Testing | Explicit scenario links replace accidental cross-package root-label activation; fixtures retain invocation/queue identity. | Same-package labels still select local rows; opaque `ok` expectations observe less than structural results. |
| Browser programming | Actual Can-authored invoice grid, versioned state, view disposal, events, timers, typed fetch and late-response protection exist. | General runtime delivery and host API breadth remain incomplete. |
| Backend programming | Parameterized SQL, transactions, explicit commit uncertainty, cookie/session/tenant checks, replay and outbox patterns are expressible. | A demonstration's deployment assumptions are not universal application guarantees. |
| Native execution | Bigint, native arrays/maps/sets, Promise modes, Bun transport/process/server operations, protected completions and owner leases are real implementation choices. | Adapters require conformance testing where host semantics differ. |

These conclusions are grounded in [owner-record checks](/Users/vince/Projects/can-lang/compiler/internal/check/owner_record_test.go:99), [pattern tests](/Users/vince/Projects/can-lang/compiler/internal/check/pattern_bind_test.go:43), [variant compatibility](/Users/vince/Projects/can-lang/compiler/internal/types/compatibility.go:9), [dependency tests](/Users/vince/Projects/can-lang/compiler/internal/project/instances_test.go:14), [error identity tests](/Users/vince/Projects/can-lang/compiler/internal/project/error_identity_test.go:10), and [scenario selection](/Users/vince/Projects/can-lang/runtime/assert/fixtures.ts:39).

## 1. Complete the shared action contract through authenticated server binding

**Highest-priority full-stack design work. Current-source evidence, including existing integration test expectations; the live matrix was not rerun here.**

Actions provide checked paths, captures, wire records, result variants, status mappings, URL construction and browser fetch contracts. Typed mounting and imported `package::action` references exist. Every action declaration still requires a handler; there is no handler-free transport declaration. A real server handler that reaches SQL or other server capabilities cannot enter the browser closure. Additionally, a realistic authenticated operation needs request-derived identity and startup-owned services, whereas the action handler signature admits only captures and body. Adding an actor, request scope or injected pool does not fit that signature. See [handler checking](/Users/vince/Projects/can-lang/compiler/internal/check/actions.go:308) and [imported action resolution](/Users/vince/Projects/can-lang/compiler/internal/check/action_fetch.go:198).

The invoice example exposes the consequence. Its server declares `/invoices/{invoice_id}`, but its live router mounts `/invoices/load` and manually reads the query and cookie. The grid fetches the declared captured path. The browser integration harness rewrites it to the mounted query endpoint; its direct-path test expects 404. The browser also duplicates action declarations with never-served stub handlers. See [the live router](/Users/vince/Projects/can-lang/examples/invoice/src/web/web.can:808), [client declarations](/Users/vince/Projects/can-lang/examples/invoice-grid/src/web/web.can:5), [the rewrite](/Users/vince/Projects/can-lang/tests/integration/gate5_frontend_test.go:437), and [the direct-path expectation](/Users/vince/Projects/can-lang/tests/integration/gate5_frontend_test.go:989).

This does not make manual HTTP callbacks unsafe or proxies inherently wrong. It means that a typed declaration does not yet eliminate the application's duplicated transport agreement and contextual binding work.

**I would recommend Can if one shared action contract could be imported by both targets, then bound on the server to explicit authentication and service dependencies.** Separate the wire declaration from its server implementation, or evaluate a bound-callable design with a genuinely server-free client projection. A shared contract should not require placeholder client handlers.

Acceptance example: one authenticated invoice GET and save, one startup pool, one shared request/result declaration, no path rewrite, no mirrored record/status definitions. Change the route and one result leaf; both sides should rebuild from the common contract or diagnose the disagreement. This does not require automatic authentication or implicit ambient dependencies.

## 2. Make browser delivery a maintained runtime contract

**A blocker to broad browser recommendation; primarily target/runtime/distribution work.**

The browser target emits TypeScript and a content-addressed asset manifest. Requiring a bundler is reasonable. The issue is that its shared runtime still imports Node filesystem, utility, crypto and async-context facilities. The grid test bundler implements four semantic shims, embeds diagnostics data, and intentionally omits native-frame module maps. Its synchronous `AsyncLocalStorage` substitute is justified for that grid's limited execution path, not as general async ownership support. See [browser entry emission](/Users/vince/Projects/can-lang/compiler/internal/emit/browser.go:70), [diagnostics loading](/Users/vince/Projects/can-lang/runtime/diagnostics.ts:30), and [test bundling](/Users/vince/Projects/can-lang/tests/integration/browser/build-grid.mjs:1).

The distinction matters: the grid is genuinely authored in Can. Its browser success demonstrates a real application subset plus explicit integration support, not a complete general browser runtime delivered by ordinary transpilation alone.

**I would recommend browser Can if a documented supported build/bundle path owned those semantics.** Prefer a browser runtime profile with a complete reachable-dependency audit. A maintained shim layer is also possible if it specifies and tests full behavior for every admitted operation; merely promoting the grid's assumptions to a universal adapter is insufficient.

Acceptance: an empty app and the invoice grid both boot through the supported pipeline, without importing a test harness. Exercise standard failures, diagnostics, equality, async callbacks, disposal and any admitted coordination. Inspect the runtime dependency closure as well as authored modules. Native browser operations should remain the implementation basis.

## 3. Allow safe exported generics to compose

**Highest-priority core language issue. Fresh checker probe.**

Exported generic bodies now receive symbolic checking. This is a useful improvement: an implementation cannot quietly narrow a supposedly general public contract by using an unsupported operation on a type parameter. However, the implementation rejects even calling another generic identity function with that parameter:

```can
fn item wrapper<item>
    emits []
    given
        item value
    asserts
        sample: 1 => ok 1
    ok call identity(value)
```

With both `identity` and `wrapper` exported, the fresh full-program check rejects the opaque argument. Making them private permits the same composition under template checking. Extracting a representation-independent helper can therefore break a public generic. This is a deliberate current restriction, not an unsound concrete specialization. See [the symbolic checker](/Users/vince/Projects/can-lang/compiler/internal/check/exported_generics.go:13) and the full reproducer in the [core review](/Users/vince/Projects/can-lang/docs/syntax-taste/evidence/2026-09-24/post-upgrade-review-961f921/independent-core.md).

**I would recommend Can if declaration-checked generic contracts could call one another symbolically.** Substitute symbolic arguments into validated signatures without prematurely creating a concrete emitted specialization. Retain rejection of genuinely representation-dependent operations. Same-function recursion already provides a useful existing special case to generalize carefully.

Explicit callable/dictionary parameters remain appropriate when an algorithm actually needs an operation. Requiring callers to supply an identity function merely to extract a helper is a different cost. The design experiment should include helper extraction, nested calls, mutual recursion/cycle handling, composites, and a callee changed to use illegal arithmetic. Visibility-dependent private/public semantics should also be documented or made explicit.

## 4. Give generic failure composition a deliberate library contract

**Important remaining abstraction choice; exact remedy not settled.**

Catalogue map/fold operations preserve callback-specific finite error sets. Authored generic wrappers still cannot express that same dependency in their public `emits` contract. This limits reusable retry, tracing, traversal, and resource wrappers. It is separate from the symbolic-call restriction above: fixing generic helper calls alone does not parameterize errors.

The strongest simpler alternative is ordinary nominal result data, adapted to concrete completion bounds at application edges. Fixed application-specific error sets also work. Neither should be dismissed as inherently inferior, but both should be tested on a genuinely reusable library rather than a one-callback example. See [callback specialization](/Users/vince/Projects/can-lang/compiler/internal/check/array.go:276).

**I would recommend Can if reusable failure-preserving helpers had one teachable, economical authoring model.** Compare result-data APIs against a small explicit finite error-set parameter using two unrelated callback bounds and a wrapper around another wrapper. Preserve explicit public contracts. The consultation split on this choice; there is insufficient evidence to prescribe a full effects system or declare the current result-data alternative inadequate without that comparison.

## 5. Provide a predictable way to express growing iterative work

**Observed runtime limitation, not a promise currently violated by the compiler.**

A fresh emitted Can countdown using terminal `relay call countdown(remaining - 1)` succeeds at 100 and 1,000 steps but yields `native_exception` with a stack-overflow message at 10,000 on local Bun 1.4.2. The threshold is environment-dependent. The generated function was tested directly with the real completion runtime, outside the CLI supervisor; source, emitted TypeScript and output are [preserved](/Users/vince/Projects/can-lang/docs/syntax-taste/evidence/2026-09-24/post-upgrade-review-961f921/core-probes/can-core-countdown.can).

`relay` forwards completion; its name does not establish tail-call elimination. Array folds already provide stack-safe traversal for existing collections. Pagination, state machines, repeated polling and other dynamically continuing work need a clearly supported alternative to growing the native stack.

**I would recommend Can if ordinary iteration had an explicit stack-safety story.** Compare native-loop lowering for proven self-tail relay with a small immutable-state iteration primitive. Preserve exactly-once argument evaluation, domain/standard failure behavior, fixture identity and meaningful diagnostics. Full general recursion elimination is not necessary. Documenting stack limits is an acceptable narrow-scope policy, but should be an intentional recommendation boundary.

## 6. Make race selection and response lifetime independently understandable

**A deliberate safety/latency tradeoff requiring product-level design.**

Native Promise selection works promptly, but owners retain losing work and drain it before scope completion. A reproducible root probe selected a result around 2 ms and returned around 83 ms because another participant ran for 80 ms. These are indicative local timings, not a benchmark. The HTTP adapter awaits a per-request scope before publishing its response, so the same policy can make response latency depend on losers; that HTTP implication is source-derived, not a freshly measured live HTTP trace. See [owner drain](/Users/vince/Projects/can-lang/runtime/owner.ts:526) and [response publication](/Users/vince/Projects/can-lang/runtime/platform/server.ts:243).

This policy protects leases, cleanup and diagnostics. Releasing live resources when a winner appears would be wrong. Nevertheless, users racing replicas may expect reduced request latency and receive only earlier internal result selection.

**I would recommend Can if request-response timing and losing-work ownership were explicit parts of the contract.** First measure a real HTTP hedge. Compare operation-local deadlines under current drainage against explicit cooperative cancellation or server-supervised ownership of response-independent losers. Specify cleanup, shutdown and side-effect responsibility. Do not imply that cancelling a promise undoes effects.

Also, an empty dynamic `race with error` remains pending because it maps to `Promise.race([])`; first-success `race` has the different native `Promise.any` empty result. A guard/nonempty constructor is the smallest improvement. A new typed empty completion is a separate semantics choice, not a bug fix required by native fidelity.

## 7. Expand frontend capability and reuse through real product examples

The browser API has useful lifecycle and state foundations, but its event record exposes only kind, target identifier, value and key. Native checked state, modifier keys, event cancellation, file lists and composition details are absent. Local persistence and browser history are not currently exposed; server WebSocket support does not imply a browser WebSocket API. Ordinary Can packages cannot fill a missing host capability because the catalogue is closed. See [event projection](/Users/vince/Projects/can-lang/runtime/platform/browser.ts:244) and [target restrictions](/Users/vince/Projects/can-lang/compiler/internal/browser/browser.go:58).

For a general SaaS frontend, keeping an invalid draft across reload, handling a file picker or respecting IME composition should have an explicit supported path. **I would recommend Can if ordinary browser needs could be added without repeatedly redesigning the compiler.** Compare a bounded catalogue expansion against a reviewed typed host-adapter mechanism. Start with persistent drafts and richer native input behavior; arbitrary JavaScript escape is not required. Agreement on the need does not establish which extension architecture is best.

UI composition itself is possible through named functions and typed nodes. The grid's rendering, focus management, view reconstruction and state synchronization are simply expensive to author. Prototype reusable typed fields and a keyed-row editor before committing to a component syntax. Measure what adding one field requires, and test selection, IME, focus, row reorder, late replies and disposal. A `browser::view` currently owns lifetime; it is not already a declarative component abstraction. See [grid rendering](/Users/vince/Projects/can-lang/examples/invoice-grid/src/web/web.can:556).

## 8. Remove avoidable refactoring and presentation restrictions

Several remaining rules impose avoidable authoring and refactoring costs:

- **Ambient capture spelling:** `near` looks up the callee parameter's name where a callable reference is formed. Renaming it can select another valid same-typed local. Immutable capture timing is sound; choosing the intended value should be explicit at the reference site. Compare capture bindings with an explicit context-record idiom. [Capture resolution](/Users/vince/Projects/can-lang/compiler/internal/check/callables.go:124).
- **Compile-failing style:** naming `amount * 2` as `invoice_total` before returning it is rejected when the local-elision predicates hold. A Boolean match must put `false` before `true`. These observed rejections enforce taste rather than exhaustiveness or type safety. A formatter or advisory lint could preserve consistency without making explanatory code illegal. [Local rule](/Users/vince/Projects/can-lang/compiler/internal/check/locals.go:84).
- **Physical-line limits:** multiline delimited expressions remain forbidden, while wide records and mandatory assertion rows can become long. A delimiter-continuation rule would improve layout without changing block indentation semantics. [Lexer rule](/Users/vince/Projects/can-lang/compiler/internal/syntax/lexer.go:90).
- **Cleanup repetition:** explicit closing branches can repeat across every domain-error outcome. The owner already provides safe fallback cleanup; an authored cleanup abstraction would additionally need a clear primary-versus-cleanup error policy. Compare a named helper and a narrowly scoped native `try/finally` lowering on a stream pump before adding syntax.

These are secondary to action and generic composition, but they affect almost every day of authoring. More compile errors are not automatically stronger contracts.

## 9. Improve data work without weakening immutability

Current map/set point updates copy native collections. With only empty construction and point updates, inserting distinct keys repeatedly performs quadratic copying. The current word-frequency gallery teaches exactly that fold. Both source review and a small native Set probe support the concern; one recorded run took roughly 184/649/2,527 ms for 10k/20k/40k unique additions. That is illustrative local evidence, not a prediction for whole applications. See [native map copying](/Users/vince/Projects/can-lang/runtime/collections/map.ts:47).

Provide bulk constructors or grouping/aggregation operations that build one native Map/Set and publish one immutable handle, with specified duplicate handling, ordering, callback failures and ownership. Preserve point-update semantics. There is no need to infer general mutable aliases or a custom persistent collection engine from this workload.

SQL also benefits from narrower improvements. Its parameterization, statement/cardinality rules and runtime row validation are real guarantees; it does not verify the declared projection against a database schema. A positive current test accepts a declared row wider than `SELECT id`. Mutation `RETURNING` remains excluded even for dialects supporting it. Native mutation-returning descriptors and schema-aware build tooling are independent choices; a transaction plus SELECT is an existing safe workaround when used correctly. See [descriptor checking](/Users/vince/Projects/can-lang/compiler/internal/check/sql_descriptors.go:50) and [cardinality rules](/Users/vince/Projects/can-lang/compiler/internal/sql/cardinality.go:29). Read limits do not imply every mutation has a bounded affected-row count.

## 10. Keep abstraction, AI, assertions and native policy claims exact

The remaining review areas do not all justify new language syntax:

| Area | Assessment and next decision |
| --- | --- |
| Owner-record equality | Invariant protection works. However, automatic equality observes hidden representation; an internal field change can change equality or remove eligibility. Decide whether stable abstract equality must be owner-selected. This does not invalidate construction confinement. |
| AI batches | Typed shared state, response validation and ordered handlers are useful. Whole-response validation is atomic; effects of earlier handlers are not rolled back if a later handler fails. Dependent questions need sequential calls, not a misleading within-batch dependency. |
| AI edge cases | Empty/one-option dynamic choice needs an application policy before a 2–255-option remote question. The runtime reports a typed invalid-question outcome; that is not compiler unsoundness. |
| Native recovery/callables | Explicit wrappers and named ordinary callables are workable. Further abstraction should follow a real workflow comparison, preserving native-origin and authored-failure distinctions. |
| Outbound integrations | Literal connection origins and origin-confined paths are auditable. Arbitrary tenant-selected destinations are outside that model. A typed policy-controlled client would need destination, credential and assertion rules; unrestricted dynamic fetch is not an automatic improvement. |
| Assertions | Explicit scenario links materially improve cross-package tests. Attached examples and verified publication remain valuable. `emits []` does not mean pure, terminating or immune to standard failures; bare opaque `ok` does not prove rendered content or effects. Browser integration and meaningful negative cases remain essential. |
| Tooling/docs | LSP, reproducible identities and build gates exist. Some operational docs describe older boundaries, including broad “no HTTP client” wording despite implemented clients. Keep one current language/target reference, with historical decisions visibly separated. |

The webhook example illustrates good HMAC-before-decode and transactional ledger/outbox structure, but its loopback operator routes rely on a trusted carrier boundary and its string signature comparison is not a constant-time primitive. These are sample/deployment and native-library concerns, not evidence that Can cannot express authorization. Avoid treating an example's support envelope as a universal production guarantee.

## Two concrete implementation defects to repair separately

These findings deserve focused fixes; neither requires a broad language redesign.

1. **The browser artifact audit mistakes text for host access.** Fresh checker/emitter/audit probes accept a function returning `"hello"`, reject harmless literal `"Bun."` and `"node: introduction"`, and accept `"Bu" + "n."`. The audit searches substrings in generated source while skipping runtime bodies. Use structured import/operation evidence and validate the actual runtime closure; do not classify quoted user content as capability use. [Audit code](/Users/vince/Projects/can-lang/compiler/internal/browser/browser.go:591), [probe results](/Users/vince/Projects/can-lang/docs/syntax-taste/evidence/2026-09-24/post-upgrade-review-961f921/browser-audit-results.json).
2. **Unicode empty regex matches advance incorrectly.** Current `text::matches` with `(?:)` and `u` or `v` over `😀x`, limit 5, returns starts `[0,0,0,0,0]`; native `matchAll` yields `[0,2,3]`. Incrementing by one enters a surrogate pair. Prefer capped native iteration or a correct Unicode advance adapter. The cap prevents an endless scan, but the results are wrong. [Runtime loop](/Users/vince/Projects/can-lang/runtime/text.ts:172), [probe output](/Users/vince/Projects/can-lang/docs/syntax-taste/evidence/2026-09-24/post-upgrade-review-961f921/core-probes/regex-output.txt).

## Recommended sequence

| Order | Deliverable | Completion criterion |
| --- | --- | --- |
| 1 | Repair the two conformance defects | Harmless browser text compiles; runtime dependency checks remain strict; Unicode regex offsets match native behavior. |
| 2 | Supported browser runtime and contextual action binding | Invoice server/client build from shared contracts and run through the supported delivery path without test-only semantic shims or route rewrites. |
| 3 | Composable public generics | Extracting an identity/helper preserves a public generic's validity; unsupported representation operations still fail at their declaration. |
| 4 | Practical library experiments | Compare result data/error parameters, UI libraries, cleanup helpers and bulk constructors on real reusable APIs. |
| 5 | Explicit iteration and request lifetime | Large iteration is predictably supported; a measured HTTP race has documented response/cleanup behavior and tested cancellation or deadline policy. |
| 6 | Broader frontend host support and authoring ergonomics | Persistent drafts and rich native inputs work; capture refactors and readable layout are unsurprising. |

The most persuasive next milestone is one independently reusable full-stack application path: shared domain package, shared action contract, contextual server binding, supported browser artifact, persistent editable UI, and a small generic helper library. That would turn the upgraded capabilities into a recommendation a new team could reproduce.

## Verification and limits

- `bun run check:runtime` passed.
- `CAN_BUN=/Users/vince/.bun/bin/bun GOCACHE=/tmp/can-upgrade-review-go-cache go test -json ./compiler/internal/...` passed: **12 test-bearing packages, 1,943 passed test/subtest events**, five actual skipped tests and two packages with no tests. Skips cover packaged archive requirements, live MySQL differential testing, and an intentional negative formatting fixture. Local Bun is 1.4.2; supplying it enables emitted-code tests but is not release qualification.
- Exact maintained runtime invocation `bun test ./runtime/test/` passed: **592 tests, zero failures, 28 live MySQL/S3 skips across 81 files**. Core scalar/data/completion test files outside that directory also passed in the core reviewer's focused invocation.
- Fresh probes covered exported-generic extraction, style acceptance, emitted recursion, Unicode regex, browser content auditing, owner selection/drain timing, empty native race and native Set construction. Gallery and all four maintained standard example projects passed fresh full `CheckProgram` checks.
- Initial unprefixed Bun path filters also discovered generated copies; those logs are retained but excluded from the maintained-runtime count.

No full release build, live database/provider application flow, complete Playwright matrix, deployment, or security audit was rerun. Existing integration-test expectations are identified as inspected source, not newly observed live results. Test success does not prove the preferred design alternatives. The reviewed revision remained unchanged during the work.
