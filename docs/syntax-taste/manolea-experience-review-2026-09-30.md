# Can review from the Manolea implementation

Design review, 30 September 2026. This records findings and recommendations, not an implementation mandate. The original [complaints](preparation/jev-manolea-experience-2026-09-30/user-notes.txt), [three initial Jev consultations](preparation/jev-manolea-experience-2026-09-30/) and three corrected row-label follow-ups are preserved. Current source is authoritative: the pasted `emits []` examples predate the current `emits {}` spelling.

The largest product gap is authoring live application tests conveniently in Can. Several other complaints expose inconsistent syntax or cumbersome APIs. The two MIME examples, however, expose an implementation and discoverability failure: Can already has the language feature needed to flatten them. We should not infer missing language features from every awkward application implementation.

This was a source investigation across Can and Manolea, with independent testing, syntax, and domain-model reviews. No implementation files were changed, no application tests or compiler builds were run, and no performance measurements were taken. Recommendations below are not claims of measured improvement.

## Findings and dispositions

| Complaint | What the source establishes | Recommended disposition |
| --- | --- | --- |
| Nested extension/MIME tests | Ordinary value matches already support strings, `|` alternatives and `_`. | Rewrite the application with existing syntax and document the idiom. No MIME-specific language primitive. |
| Repeated error handler arms | Completion matching accepts only one named error per arm, unlike ordinary matching. | Strong candidate for grouped named error arms with exhaustive checking. |
| Repeated bare error forwarding | Each bare arm already forwards its original completion. | Allow the same explicit grouping for forwarding; preserve payload and occurrence identity. |
| Repeated `asserts` and `when` rows | Labels are execution identities and fixture selectors, not decorative names. Fixture templates only expand inside `when`. | Support a design for label-list expansion into independent rows; never merge roots or queues. |
| General error-handling ceremony | `match chain` and `relay call` already support sequencing/propagation, but repeated mappings remain. | Improve use of existing forms and investigate grouped arms before adding another control-flow family. |
| `emits` rarely used | Of 325 Manolea source functions, 152 have nonempty bounds and 173 have `emits {}`. | Keep error contracts; focus on why the project declares no domain errors. Empty bounds do not mean purity or freedom from standard failure. |
| Response helpers should be errors | The named helpers return HTTP responses; one also performs cleanup. | Keep boundary rendering/compensation. Evaluate typed domain errors inside service operations. |
| CRUD primitives | Typed SQL operations and transactions exist; atomic owner/state predicates carry application policy. | Explore readable typed packages/descriptors before dedicated grammar. |
| Authentication primitives | Basic security and transport capabilities exist; account/session policy is composed in the app. | Explore reusable typed workflow APIs with explicit policies before an authentication keyword. |
| Python application tests | Python manages live processes, fixtures, protocols and independent observations that attached assertions intentionally cannot perform. | Highest priority: a bounded Can-facing live test facility, building on existing capabilities. |
| HTML-specific chaining | General `match chain` is already used; constructor-level page authoring remains verbose. | Investigate HTML authoring independently of the withdrawn tree proposal; no replacement has been selected. |
| One-space indentation | The lexer, formatter and editor explicitly enforce/use four spaces. | Record one space as the user's preferred format. Treat it as a coordinated syntax/tooling migration; it does not remove nesting. |
| Unread generated values | Assertion modules receive broad imports and type graphs. The reported `dist/assertions` files and exact `$canFunctions` binding were unavailable in current sources. | Investigate fresh generated output and distinguish source redundancy from runtime work before optimizing. |
| Primitive `some`/`none` | Can already has typed optional values; absence is different from completing a function with `ok`. | Evaluate unqualified constructor/pattern names as a narrower convenience, without changing completion semantics. |

## The nested examples are already expressible

The [parser](/Users/vince/Projects/can-lang/compiler/internal/syntax/matches.go:130) and [checker](/Users/vince/Projects/can-lang/compiler/internal/check/patterns.go:225) implement alternatives. The [specification](/Users/vince/Projects/can-lang/docs/syntax-taste/technical-spec.md:156) explicitly distinguishes ordinary patterns from completion patterns. These are current syntax examples, source-reviewed but not compiled during this review:

```can
match call path::extension(name)
    ok str ext => match ext
        ".html" | ".css" | ".js" | ".mjs" | ".json" | ".txt" | ".png" | ".jpg" | ".jpeg" | ".webp" | ".woff" | ".woff2" => ok true
        _ => ok false
```

```can
match call path::extension(name)
    ok str ext => match ext
        ".html" => ok mime is "text/html"
        ".css" => ok mime is "text/css"
        ".js" | ".mjs" => ok mime is "text/javascript"
        ".json" => ok mime is "application/json"
        ".txt" => ok mime is "text/plain"
        ".png" => ok mime is "image/png"
        ".jpg" | ".jpeg" => ok mime is "image/jpeg"
        ".webp" => ok mime is "image/webp"
        ".woff" => ok mime is "font/woff"
        ".woff2" => ok mime is "font/woff2"
        _ => ok false
```

These replace the ladders in [drafts.can](/Users/vince/Projects/manolea-2/src/site_store/drafts.can:118) and [revisions.can](/Users/vince/Projects/manolea-2/src/site_store/revisions.can:19). The first line of alternatives is long, but its width is separate from the eliminated control-flow depth. A single shared extension-to-MIME table could also prevent drift: the same policy is already repeated in [public bundle rules](/Users/vince/Projects/manolea-2/src/site_agent/bundle_rules.can:27). Whether to share it as data or a function is an application design choice.

The authoring guidance should show this pattern. The [user guide](/Users/vince/Projects/can-lang/docs/user-guide/README.md:55) still lists several important recipes as pending. Available features are not helping if implementers repeatedly miss them.

## Testing: agree with the priority, narrow the requirement

There is no need for Python merely to select an assertion. `canlc assert` already selects by package/declaration/name, executes expected and actual regions, supports call fixtures, and supervises workers with timeouts and structured reports. See the [assertion guide](/Users/vince/Projects/can-lang/docs/implementation/assertions.md:13).

The actual Python probes do much more. [Recovery](/Users/vince/Projects/manolea-2/tests/auth_recovery/probe.py:72) checks HTTP responses, token state, expiry/reuse, rollback and session revocation. [Image delivery](/Users/vince/Projects/manolea-2/tests/examples/e03_probe.py:91) starts the application with disposable data, sends multipart requests and inspects real database/file results. [MCP](/Users/vince/Projects/manolea-2/tests/site_agent/mcp_probe.py:145) verifies protocol behavior and secrecy, with explicit teardown. These are valuable application tests, not disposable scaffolding.

Attached assertions deliberately [refuse live platform output](/Users/vince/Projects/can-lang/docs/implementation/assertions.md:55). Catching that refusal cannot make the assertion pass. Existing `scenario` declarations associate deterministic fixture rows; they do not supply a live application lifecycle. Removing these guards would change what a passing assertion means.

Can is not starting from zero: it has checks, files, SQL, declared fetches and bounded process execution. The [Can storage driver](/Users/vince/Projects/manolea-2/tests/site_store/driver.can:85) already runs real storage logic; its [Python wrapper](/Users/vince/Projects/manolea-2/tests/site_store/probe.py:20) arranges hostile fixtures and observes results. But [process::run](/Users/vince/Projects/can-lang/std/catalogue/README.md:513) waits for a child to finish, and [fetch configuration](/Users/vince/Projects/can-lang/docs/implementation/transport.md:7) is not an ergonomic general test client for ephemeral ports, session cookies and malformed traffic.

The useful requirement is: **a Can application developer should be able to author the ordinary application test suite in Can, including its HTTP, database, file and protocol cases, and run it through a supported tool without writing Python or TypeScript for those cases.** That tool may internally use Bun, Go or a browser driver. Requiring the test runner's implementation itself to be Can would be a separate self-hosting objective. Porting one probe is the first proof of the path, not the final definition of completion.

First acceptance experiment: port one existing probe without losing its observations. It must own an isolated database and files, start/stop the application, address an allocated port, carry session state, assert response headers/body and persisted results, enforce bounded timeouts, and clean up after success, failure or interruption. Register cleanup when allocating resources; reuse one verified build within a run, reclaim owned generated output and temporary workspaces, and recover abandoned owned work without removing active or foreign artifacts. Report cleanup failures and retain only compact evidence by default. Existing low-level capabilities should be reused, with only missing test-specific adapters added. Keep compiler/runtime conformance and browser execution as separately identified evidence. No mass deletion of external tests before coverage parity.

## Error handling: distinguish domain outcomes from response construction

The empty [Manolea registry](/Users/vince/Projects/manolea-2/can.errors.json) and source census establish zero application error declarations. That deserves attention. It does not establish that Can lacks an error mechanism: [authored payload errors](/Users/vince/Projects/can-lang/tests/failure-conventions/owner-setup/src/ids/ids.can:8) and [bare forwarding](/Users/vince/Projects/can-lang/tests/failure-conventions/owner-setup/src/handler/handler.can:26) already work.

The named examples have different responsibilities:

- [store_unavailable](/Users/vince/Projects/manolea-2/src/jobs/jobs.can:29) builds a sanitized 503 response.
- [preview_error](/Users/vince/Projects/manolea-2/src/site_owner/preview.can:23) creates a response with privacy headers.
- [failed_image](/Users/vince/Projects/manolea-2/src/examples/upload.can:287) removes a previously written file before selecting a response.

An error constructor cannot replace that work. A clearer service design may emit domain errors which an HTTP boundary then renders through helpers like these. Construction creates nominal data; terminal emission and forwarding determine control flow. Functions and errors need not become the same abstraction.

Nor should every `false` or `none` become an error. [current_identity](/Users/vince/Projects/manolea-2/src/auth/context.can:22) intentionally treats absent, expired and unverified sessions as unauthenticated while keeping SQL failures distinct. Boolean validation predicates can also be entirely appropriate. Investigate cases where callers lose a distinction they need, rather than optimize for the number of error declarations.

The strongest syntax candidate is explicit grouping, for example the **proposed, currently unsupported** forms:

```can
sql::unsupported_value | sql::connection_failed | sql::query_failed => ok false
```

```can
html::invalid_structure | form::unknown_field | form::invalid_name
```

The second form must relay the original matching error, not reconstruct it. Define duplicate/overlap diagnostics, exact generic specializations, payload aliases and shared-binding rules. Keep complete error coverage. A conservative first version can restrict grouped heads to unbound named errors, covering the supplied examples without inventing heterogeneous payload typing.

Before redesigning error declarations, compare one internal service operation using current domain errors with its existing implementation. Inspect declaration/registry work, propagation, caller mappings, assertions and useful diagnostic detail. This separates application layering problems from specific language friction. `emits {}` remains a bound on escaping domain errors, not a purity or total-safety promise.

## Test-row compression must preserve separate cases

The [assertion parser](/Users/vince/Projects/can-lang/compiler/internal/syntax/declarations.go:412) accepts one label. The [checker](/Users/vince/Projects/can-lang/compiler/internal/check/assertions.go:28) uses it for root identity and fixtures. In the supplied example, `absent`, `configured`, `sample` and `serve` may select different nested behavior despite identical row text.

The proposed `absent | configured | sample | serve: ... => ok` should therefore expand to four independently addressable rows. Preserve selectors, scenario links, execution modes, order, reports and unused-fixture detection. Shared text must never turn four queues or roots into one. Conversely, if all inputs, fixtures and expected results are identical, four labels add no behavioral coverage; names such as `missing` and `too_much` do not make input `0` represent different values.

Fixture templates offer another reuse mechanism, but source inspection after the initial Jev disagreement exposed a material limitation: the [checker rejects template use on assertion roots](/Users/vince/Projects/can-lang/compiler/internal/check/assertions.go:40). Expansion is restricted to lexical `when` tables, and [each use retains one selector](/Users/vince/Projects/can-lang/compiler/internal/check/templates.go:246). Four selectors still require four uses. Templates help reuse boundary fixtures across sites; they do not replace the proposed local compression. The [Manolea example](/Users/vince/Projects/manolea-2/src/web/web.can:664) actually selects different downstream fixtures, so preserving all four names matters. I recommend the independent-row expansion design. The initial consultation state described templates too broadly; corrected follow-up requests explicitly include these limitations.

## CRUD, authentication and HTML

The goal of readable CRUD/authentication is sound. Dedicated keywords are not yet justified. Manolea's [job_open SQL](/Users/vince/Projects/manolea-2/can.project.json:240) atomically combines owner, draft-state and contact requirements. A shorter `update` abstraction must preserve those predicates and conflict outcomes. Evaluate query/descriptor ergonomics, affected-row contracts and reusable domain functions first; do not hide authorization or transactional boundaries behind generic CRUD names.

Authentication has reusable mechanisms and variable policy. Sessions, credential handling and token operations are candidates for maintained typed packages; signup eligibility, verification requirements and product-specific permissions need explicit policy. Test a package design against login, recovery, revocation and denial before deciding which part, if any, requires grammar. This is not a proposal to implement new cryptographic algorithms in Can.

HTML authoring remains an open design question. The owner explicitly discarded the tree syntax on 1 October 2026; the [withdrawal record](html-authoring-investigation-2026-09-30.md) supersedes the former components-versus-tree recommendation. Preserve existing HTML trust boundaries and useful error locations while investigating another approach. This withdrawal does not select a replacement.

## Indentation, option names and generated output

One space is the user's recorded preference. It is feasible with no compatibility obligation, but changes [lexing](/Users/vince/Projects/can-lang/compiler/internal/syntax/lexer.go:88), [formatting](/Users/vince/Projects/can-lang/compiler/internal/syntax/format.go:161), trivia handling, editor settings and the source corpus. Use one canonical format rather than mixed conventions. Flattening the MIME ladder removes depth; changing indentation only reduces its displayed width.

`some`/`none` can become easier to write without becoming new completion kinds. Can already represents [option values](/Users/vince/Projects/can-lang/runtime/catalogue.ts:14775) as a typed variant. `ok` says the computation succeeded; an optional value says whether data is present. The current `ok option::none()` is a meaningful combination; `ok none` would be hypothetical shorthand. Rust likewise models absence with [Option variants](https://doc.rust-lang.org/std/option/). Evaluate short constructor/pattern names or a prelude, including naming collisions and inference, before introducing more keywords.

The generated-code concern is plausible but the exact reported output could not be reproduced by inspection: `dist/assertions` is absent from this Manolea checkout, and `$canFunctions` was not found in the current Can compiler/runtime sources. Current [assertion emission](/Users/vince/Projects/can-lang/compiler/internal/emit/program_entry.go:38) still imports all authored functions and broad state bindings for each assertion and includes a broad type graph. Bun authored-module emission [retains every declaration](/Users/vince/Projects/can-lang/compiler/internal/emit/program_modules.go:10), whereas browser production has reachability pruning.

Type declarations disappear during TypeScript transpilation. Unread imports, initialized bindings and side-effecting module initialization are different optimization questions. Confirm the retained executable graph before inferring a runtime or disk win. Existing [startup attribution work](/Users/vince/Projects/can-lang/docs/performance/generated-startup-plan.md:10) already warns against equating broad imports with safe removal. Fresh bounded emission/typechecking can establish redundancy; performance measurements remain separate and were not launched here.

## Recommended order and acceptance

1. Prove a Can-facing live test path by matching one existing probe's behavior and cleanup guarantees.
2. Correct the MIME implementation using current patterns and add the missing authoring recipe.
3. Design grouped error handling/forwarding with exact coverage and occurrence-preservation checks; compare label-list expansion against existing templates.
4. Trial one service boundary with meaningful domain errors and independently investigate HTML authoring on a real page.
5. Apply the chosen canonical indentation in a coordinated migration; evaluate short option names and typed CRUD/auth packages with representative source examples.
6. Audit fresh emitted imports, declarations and initialization separately; optimize only behavior proven unnecessary.

The [Jev consultations](preparation/jev-manolea-experience-2026-09-30/findings.md) are advisory. The initial three agree most strongly on a live test facility, grouped error arms, package-level CRUD/auth exploration and comparative HTML prototypes. Their label-grouping disagreement exposed missing template-scope evidence; three corrected follow-ups favor independent row expansion, with varying confidence. The domain-error trial also showed material wording sensitivity. These results do not prove the designs or erase shared framing bias.
