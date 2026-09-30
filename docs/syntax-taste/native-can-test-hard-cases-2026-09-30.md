# Native Can tests: challenging the hardest existing cases

Status: **source challenge and post-probe reconciliation complete; integrated bindings and qualification remain pending**.

The chosen full Can → TypeScript/Bun architecture remains a reasonable starting point. Source analysis does **not** justify claiming that the proposed capability surface can already replace every host test. The hardest cases expose specific missing semantics: concurrent browser request handling, operations within one browser task, synchronous native values and callbacks, candidate/runtime separation, descriptor lifetimes, and independently observed database settlement.

This document follows the [capability contracts](native-can-test-capabilities-2026-09-30.md) and [lifecycle rules](native-can-test-lifecycle-2026-09-30.md). It corrects optimistic assumptions in those contracts and defines the questions that require executable evidence. It is not an implementation plan or an authorization to run experiments. No build, test, browser, database or measurement ran in this round.

The later [prerequisite inventory](native-can-test-prerequisites-2026-09-30.md)
records exact available candidate/Bun inputs and confirms that an accepted R
compiler artifact and qualified generic N owner are still missing. They block
integrated Can qualification, while narrowly scoped mechanics investigations may
use identified existing tooling and independent bounded ownership. The register
distinguishes these evidence scopes. Preparatory PM-N1/N2/F1 supported limited
mechanics and PM-B1 failed browser isolation despite route support. The
[design review](native-can-test-design-review-2026-09-30.md) incorporates those
results; all 14 integrated entries remain unrun.

## 1. Preserve the authoring and trust requirements

Ordinary unrestricted Can functions own scenarios, helpers, fixtures, expectations, selection, scheduling and report policy. They use normal effects and targets. AI agents remain the intended authors: use explicit typed arguments, handles, identities and structured errors rather than implicit timing, runtime source strings or convenience APIs that hide test policy.

- **R** compiles the trusted Can controller and judge. Its qualified artifact is still a prerequisite, not supplied by this source review.
- **C** is the candidate compiler/runtime/subject. Its code never imports into the trusted judge realm.
- **N** owns bounded resources outside killable workers and reports execution/cleanup facts. It does not decide whether a scenario passed.
- Static adversarial input, generated TypeScript and generic native bindings are legitimate artifacts. An authored JS/TS scenario or oracle hidden in a string, callback or native whole-test operation is not a completed migration.

No new language primitive per testing category is indicated. Missing operations belong in typed packages and reviewed native bindings, lowering to equivalent native operations with the adapters required for Can contracts. A synchronous JS getter does not become asynchronous merely because its caller is a Can program; admitting such a callback needs an explicit binding contract. Ordinary Can helper functions remain unrestricted regardless of which foreign callback sites can accept them.

Three evidence levels apply throughout:

| Level | Meaning |
| --- | --- |
| Source-established | The cited current implementation/test contains this behavior or constraint; no execution claim |
| Design composition | The case can be described with a finite sequence of Can decisions and typed native facts; bindings may still be absent |
| Feasibility gate | Source alone cannot establish the proposed bridge, ordering or cleanup behavior; an explicitly bounded future experiment is required |

## 2. Browser interception and callbacks

The detailed [browser source challenge](preparation/native-can-test-hard-cases-2026-09-30/browser-source-challenge.md) traces five representative families from `grid.mjs`, `invoice.mjs`, controls, browser codecs and remote Firefox. These include several different callback problems; they should not be collapsed into a generic `evaluate` operation.

| Existing hard case | Can sequence and independent facts | Required correction / experiment |
| --- | --- | --- |
| Save while a request is held; edit or press save again | Arm capture → begin physical input → receive paused route → inspect/edit DOM → release route → settle input/response → compare request count/draft/tree → seal | Driver must process route/event requests while input is unsettled. Introduce logical `input_begin/input_settle`, or prove an equivalent nonblocking acknowledgment. **B1** |
| Two synchronous DOM clicks in one page task | Can supplies a finite immediate action list; native dispatch executes it without yielding; Can checks requests and DOM | Two separate RPCs weaken this test. Proposed `dom_batch` has no branches, loops, arbitrary callbacks or waits. **B2** |
| Commit upstream, then corrupt browser-visible response and replay | Fetch the held request once → retain complete upstream status/body → deliver Can-chosen corrupt bytes → observe uncertainty → send a new request with the same application `operation_id` → compare durable state | Separate upstream contact from browser delivery and decoded body from wire bytes. Never retry an unknown fetch. Independent committed-row observation strengthens the old route script's evidence. **B1**, after DB binding qualification |
| Missing swap target and cloned replacement | Clone target → begin request → remove original → release response → observe document-level occurrence → append clone → compare identities/fields/DOM | Clone/append is different from reattach. Add owned node generation/identity and custom-event data projection. **B2** |
| Target already absent before submit | Clone/remove before input → observe allowed no-request branch or request-phase `effect=none` → preserve draft/page → remount clone | Keep this separate from removal during a flight, whose occurrence is response-phase and uncertain. One subleg cannot credit both. **B2** |
| Real keys, synthetic composition, post-handler cancellation | Wait for app listener → install bubble watcher on the same node afterward → acknowledge → dispatch action → inspect native event/property and Can echo separately | Include `defaultPrevented`, listener-registration acknowledgment, `isTrusted` and physical/synthetic distinction. **B2** |
| Competing write in the current page generation | Read generation attribute → construct request in Can → native Fetch in page realm → observe response → stale save/conflict checks | External HTTP does not reproduce page cookies/origin/CSP. Add typed page-realm Fetch with explicit credentials/redirect policy and bounded response facts. **B3** |
| Browser codec parity | Ordinary browser-target Can functions produce facts; R compares Can-owned expected vectors with browser and Bun observations | Existing bundled TS vector module contains an oracle and cannot survive as a native pass/fail macro. Qualify artifact loading/DOM readiness and lossless fact export. **B4** |
| Remote Firefox cleanup | Own a context/connection; terminate one worker; close its resources; create another context | Shared launch server is not case-owned. Prove cleanup leaves it alive. **B5** |

The initial route model remains **tokens returned to ordinary Can**, with asynchronous route handling independent of in-flight actions. This covers deferred decisions without transferring a live controller closure. A richer locally compiled Can callback remains a separate binding if a concrete obligation requires it. It is not an assumed escape from the native callback problem below.

A terminal browser seal must account for route callbacks, body capture and event listeners. Quiet DOM or an elapsed delay provides a prefix, not absence of later faults. Correctly surfaced close/cancellation errors are facts for Can; dropped callbacks, unknown route effects and gaps prevent a complete pass. Existing engine limitations, such as a WebKit redirect leg, remain visible coverage dispositions rather than being silently converted into success.

PM-B1 adds two constraints: actual input settlement may fulfill after route abort,
so it cannot stand in for delivery/application success; and an owned headless
process/profile does not exclude native credential UI. The next browser admission
must satisfy the capability contract's test-credential/host-observer boundary.
No rerun is part of this design review, and adding missing launch flags alone
does not close B1. Retain the first control-oracle failure and later UI failure.

## 3. Unusual native values and the R/C bridge

The [native source challenge](preparation/native-can-test-hard-cases-2026-09-30/native-source-challenge.md) follows `tests/conformance/native.ts` and delegated array, collections, codec and coordination suites. The requirements include:

- Exact JSON source tokens beyond safe JS integers, signed zero, raw JSON serialization and ordered async traversal.
- Getter/then-call counters, forged or revoked Proxy rejection without touching traps, and unchanged payload identity.
- The same error occurrence and completion through failure forwarding, ordered outcomes, late loser diagnostics, and a resource held until late work settles.

Can chooses each input descriptor, operation, sequence and expectation. A native service constructs a raw value, performs one admitted operation and returns an opaque handle or tagged observation. It must never resolve an unboxed hostile value through an async RPC. Exact numbers use lexemes/bits; identity is a comparison performed within the owning realm, not equality of two copied JSON objects. Counter snapshots and final event seals are needed for zero-access claims.

There are **three distinct evidence scopes**:

| Scope | What it can establish | What it cannot establish alone |
| --- | --- | --- |
| Raw API observer under identified candidate Bun | Behavior of the native Bun/JS API and hostile values in that realm | Candidate Can code generation or its private runtime adapter |
| R-built probe in an isolated observation process | Behavior of its identified R code and the explicitly bound native environment | Behavior of C merely because the executable is C's Bun |
| C-built production subject with reviewed same-realm native ingress | Candidate generated code and adapter behavior against constructed hostile values | An independent verdict, or permission to load candidate code in R's judge |

`RunOutput` authenticates the runtime manifest and every private runtime import. Swapping C runtime files under an R-built probe is not a supported linking strategy. Completion brands and handles are realm-local; a copied object does not recreate an authenticated completion. Any C ingress must identify the exact current artifacts, exports, admitted types and local completion handling, returning inert facts to R. It introduces no backwards ABI promise.

The emitter normally lowers named callable values to asynchronous `Promise<Completion<T>>` closures. Specialized `value`/`invokeSync` paths exist, so “Can has no synchronous paths” would be false. They do not prove that arbitrary Can functions can implement JS getters, Proxy traps or comparators synchronously. Initial hostile descriptors may use reviewed immediate native return/throw/count/delegate behavior. A finite constructor vocabulary must not grow into a scenario interpreter. General synchronous Can callback companions are **not** a prerequisite invented for all tests; if a retained obligation cannot be expressed without one, that precise callback binding becomes a blocking design question.

Qualify lossless inert transport in **N1**, same-realm C adapter ingress and its synchronous boundary in **N2**, and late identity/lifetime observation in **N3**. If N2 fails, raw Bun facts do not earn candidate-adapter coverage. Revisit the bridge/compiler contract before considering an interpreter; none of this source evidence yet establishes that a second full Can executor is necessary or cheaper.

The narrower PM-N1 getter/alias and PM-N2 source-export checks support continuing
this design. They omit the integrated value matrix, C-generated fixture/ingress,
completion authentication and R oracle. Place minimal N1/N2 acceptance ahead of
dependent migration work in the implementation plan; do not promote the archived
host subjects into permanent bindings or silently reuse their expected answers.

## 4. Descriptor credentials and launcher isolation

The [descriptor source challenge](preparation/native-can-test-hard-cases-2026-09-30/descriptor-source-challenge.md) follows the input/environment integration test, distribution launch tests and current Go launcher. Two launch policies must remain explicit:

1. **Test C's product launcher:** pass the declared hostile environment to C's Go CLI unchanged. The test protects C's sanitization of its Bun child and preservation of original application values. If N removes `BUN_OPTIONS`/`NODE_OPTIONS` first, it masks the defect.
2. **Launch a generated entry directly:** supply sanitized startup configuration plus fd 3 application snapshot, fd 0 binary input, and a declared fd 4 generation lease. R/N startup and their private channel stay outside the subject's environment/descriptors.

`runtime/environment.ts` reads fd 3 synchronously to EOF during import. The writer therefore starts concurrently with the child and must close without waiting for application readiness. Malformed/missing snapshot data is a startup/import failure, distinct from a successful lookup returning missing credentials. Present-empty and missing remain different, and invalid-name validation must precede lookup.

The existing Go implementation also handles children that never read a one-MiB snapshot: close/unblock the writer after child exit, tolerate the expected unused-pipe condition without masking a real child failure, and reap all resources. The current ordinary process binding lacks the extra-descriptor interface and external ownership. Installed Bun 1.4.2 declarations show possible extra-FD mechanics; they are not a successful runtime qualification and do not displace the existing Go owner.

An output lease survives launcher death through the child descriptor. Do not prune merely because the launcher exited. **F1** qualifies delivery/startup isolation; **F2** qualifies inherited lease lifetime. Compiler-internal lease tests are supporting evidence here, not new `/tests` migration obligations silently added to the ledger.

## 5. Independent database checks

The [database source challenge](preparation/native-can-test-hard-cases-2026-09-30/database-source-challenge.md) traces [HISTORY-007 and HISTORY-008](preparation/native-can-tests-migration-ledger-2026-09-30/sql-history.md#history-007), their delegated F02/F03 drivers, current SQL runtime and checker.

Use **separately owned direct-driver observations**, bypassing Can's descriptor construction, decoding, error mapping and resource wrapper. Record that raw Bun SQL still shares Bun's driver and the database server. A claim about Bun SQL itself requires another implementation; every adapter test does not automatically need a second driver.

The native DB contract needs more than `query(sql)`:

- Distinct connection, transaction, attempt and operation identities. A transaction uses its pinned native connection through drain/commit/rollback. `LAST_INSERT_ID()` on a different pooled connection proves nothing about that writer.
- Explicit asynchronous start/settle facts. A deadline is not native query cancellation; cancel support, server acknowledgment and later settlement must be reported separately.
- Ordered typed cells with exact int64/decimal text, float bits, bytes, null and date/time encoding; structured SQLSTATE/errno/constraint and raw row/affected counts. Never convert an unsafe integer to a JSON number before Can compares it.
- Server-supported, externally owned namespace isolation and cleanup. Old fixed table names, `DROP IF EXISTS` and swallowed cleanup errors are not deletion authority.

Distinguish driver-promise settlement from **server-side effect quiescence**. A connection error can settle the client while the server effect is unknown. A final read needs a server acknowledgment or independently qualified fence/reconciliation, as well as stopping newly admitted writes. If that evidence is unavailable, retain `indeterminate`; connection close alone is not proof that dropping the namespace is safe.

The source review changes how several claims should be made:

| Finding | Design consequence |
| --- | --- |
| F02 holds A's transaction, probes through runtime B, then releases A; its raw path is used elsewhere | Retain actor evidence and prove causal ordering A-acquired → B-probe/result → A-release → native settlement. An additional raw lock check is stronger evidence, not something the old harness already did. |
| F03 calls RETURNING a future feature, but current checker/runtime admit the supported `one` INSERT form for PG/SQLite | Follow current source. Raw RETURNING success is not coverage of a compiled Can RETURNING path. **D1** checks that path with independent final rows. |
| The poison case has no successful write before the duplicate | Its `ok`/25P02/replay facts do not independently prove a preceding write was rolled back. **D3** adds a sentinel and a non-poison control explicitly as stronger qualification evidence. |
| Old warmup avoids a callback-scope defect; current code contains a binding fix | Preserve historical evidence without assuming the defect persists. **D2** checks one bounded fresh-pool burst; do not preserve warmup as a permanent requirement. |
| SQL deadlines race native work; a write can finish later and transactions can report `commit_unknown` | An immediate empty read cannot prove no write. Stop further admitted work, establish native settlement, then make a fresh independent read. If settlement cannot be established, keep the effect indeterminate and cleanup unresolved. **D4** |

Eight/16-writer scenarios remain separate ledger coverage, not replaced by a two-connection mechanism probe. F02's 60-iteration timing comparison remains deferred descriptive measurement; it has no correctness threshold. Qualification cannot silently include it in a quick run or claim it was covered here.

## 6. Scoped feasibility work before implementation planning

The [experiment register](preparation/native-can-test-hard-cases-2026-09-30/experiments.md) gives **14 unrun experiments**, each with one concrete boundary question, minimum fixture, acceptance/negative controls, dependencies, finite resource envelope and cleanup requirement. Support notes contain broader source examples; this register governs future execution scope.

Priority is determined by the consequence of failure:

1. **N2 and B1** test the most consequential bridge/concurrency assumptions. N1's inert-value mechanics precede N2; F1 supplies any required descriptor launch. Define artifact/authority seams before writing prototype code.
2. **B2/B3/B4 and D1/D2/D3/D4** establish specialized bindings and stronger oracles. Each database/browser environment is admitted explicitly; absent required services mean blocked, not a passing skip.
3. **N3, F2 and B5** challenge lifetime/late-evidence/shared-service behavior using externally owned disposable subjects. They are still required for the affected migration facets, even if ordinary happy paths work first.

This is dependency order for feasibility, not permission to run all jobs in parallel on the laptop. The existing lifecycle's single-live-case admission and aggregate budgets apply. Source/doc work may proceed in parallel. Individual probe success credits only its named mechanism; final migration still needs every retained ledger facet, negative controls, callers and cleanup receipts.

If a probe fails, retain the failing fact and affected obligation. First distinguish a binding/ownership defect from a fundamental backend limitation. Broaden a probe or change architecture only through an explicit updated question and acceptance condition; do not turn a small experiment into a suite migration or benchmark.

## 7. Consultation and review evidence

Three fresh Jev requests used semantically equivalent facts and alternatives with every explanatory context, question and option rewritten. All chose explicit native bridge qualification, Can-side route tokens first, and a scoped direct-driver SQL observer. One browser response gave 0.06 probability to local Can callbacks; one DB response gave a combined 0.05 to additional-driver alternatives. No selected-answer disagreement occurred. Those alternatives remain relevant if round-trip feasibility fails or an obligation targets the underlying driver.

Jev is classifier advice, not source research or proof. Agreement does not establish feasible callbacks, complete coverage or bias removal. See the [consultation record](preparation/native-can-test-hard-cases-2026-09-30/findings.md), [pre-send audit](preparation/native-can-test-hard-cases-2026-09-30/wording-audit.json), and saved requests/responses beside them.

The [review record](preparation/native-can-test-hard-cases-2026-09-30/review.md) lists corrections from the browser/native/SQL reviewers. [Documentation validation](preparation/native-can-test-hard-cases-2026-09-30/validation.json) checks references, saved consultations and source identities; it is not executable feasibility evidence.

The implementation plan must carry qualified R/nonpublishing suite bootstrap,
external N, typed C/native ingress, browser isolation/concurrency/task semantics,
descriptor ownership and independently settled DB observations as dependencies.
Preparatory results inform those tasks; integrated acceptance remains future work.
The current reconciliation and review disposition are recorded in the
[design review](native-can-test-design-review-2026-09-30.md).
