# Language behavior contracts — 2026-09-22

Status: selected design for the user's requested behavior revisions, ready to drive implementation planning. **Not an implementation claim.** This document is the normative revision addendum for the sections listed in B9; it takes precedence there over the September 20 contracts and September 22 review sketches. Unmentioned semantics remain unchanged. New syntax below is specified source syntax for implementation, not syntax accepted by today's compiler. No compatibility spellings or runtime layouts are required.

This completes the requested normalization, wrappers, generic-error patterns, standard snapshots, build/assertion publication, fixture reuse and native-request testing designs. It preserves native AI forms, grouped state, explicit contracts, attached assertions, nominal immutable data and native JavaScript/Bun behavior. It does not adopt generalized effects, flattened state, anonymous functions, root-owned fixture overrides or platform extensions.

The user has explicitly gated larger abstractions on demonstrated need. Error-set parameters, changed capture syntax and state-callable redesign remain outside implementation scope under [the demonstrated-need gate](language-design-dispositions-2026-09-22.md#demonstrated-need-gate-for-larger-abstractions). Nothing in these behavior contracts makes them an implicit dependency.

## B1. Fetch/judge normalization

### Public value and identity

Allocate distribution error ID **1106** to `http::request_failed`. The catalogue declarations are:

```can
// Catalogue contracts; these are not project declarations.
variant failure_detail
    http::invalid_request
    http::credentials_missing
    http::transport_failed
    http::timeout
    http::body_limit
    http::status_error
    codec::invalid_data

error 1106 request_failed(http::failure_detail detail)
```

The qualified variant is `http::failure_detail`. Its leaves are the existing nominal error values, not new copies of their record types. `http::request_failed(detail)` is ordinary constructible error data; construction alone does not prove a request occurred. Existing error construction/forwarding rules distinguish data from failure completion. The finite variant is an infrastructure-detail contract, not a closed union of all application failures.

| Detail leaf | Public fields retained unchanged |
| --- | --- |
| `http::invalid_request` | `str reason` |
| `http::credentials_missing` | `str variable` |
| `http::transport_failed` | `str phase` |
| `http::timeout` | `int timeout_ms` |
| `http::body_limit` | `int limit` |
| `http::status_error` | `int status`, `http::header[] headers` |
| `codec::invalid_data` | `str path`, `str reason` |

Preserve A2/A4's finite reason tokens, phase values, normalized response headers and exposure rules. Do not append credentials, request/response bodies, raw provider messages or native exception objects. No new public `cause`, phase field or occurrence field is added. The original runtime occurrence remains a private diagnostic cause of the mapped occurrence. Mapping allocates a new occurrence at the boundary; forwarding preserves that occurrence. Reconstructing an equal error creates new failure identity as under C9.

### Which failures are normalized

Classification is determined by the executing boundary, not by an error's name or the dynamic call stack alone.

| Origin | Default behavior |
| --- | --- |
| Compiler-owned dynamic request/config/credential validation for fetch/judge | Map an applicable one of the seven leaves to `http::request_failed` |
| Compiler-owned request encoding, bounded HTTP transport/body consumption and response decoding for fetch/judge | Same mapping |
| Body-only fetch or judge non-2xx response | Map `http::status_error` |
| Envelope fetch with a complete non-2xx response | Return the existing response success; no intrinsic status error |
| Judge question preparation or answer-envelope validation producing `ai::invalid_question` / `ai::invalid_answer` | Preserve the AI error |
| Invalid UTF-8/JSON during judge native decoding | Map `codec::invalid_data` |
| Valid JSON that violates the judgment answer contract | Preserve `ai::invalid_answer` |
| Ordinary `given`/grouped `state` argument evaluation before entering the target | Outside the wrapper/normalization boundary; ordinary enclosing call-match rules apply |
| An authored helper called in a descriptor expression, question handler or judge continuation emits one of the seven names | Preserve its authored error, including `codec::invalid_data` |
| An ordinary explicit call to `codec::decode_json`, a standalone catalogue HTTP operation, LLM or SQL | Preserve its existing contract; this revision does not wrap it implicitly |
| Standard failure, including an unexpected adapter defect | Preserve the standard channel; never relabel it `http::request_failed` |
| Static invalid declaration, schema or configuration | Compile diagnostic; not a recoverable native outcome |

Compiler-owned encoding of values returned by descriptor expressions is inside normalization; evaluation of authored code that produces those values is outside the **native-origin** set even though it executes inside the declaration. The emitted-origin wrapper table in B2 can handle such authored domain failures after target entry.

The raw intrinsic set `N` is computed by declaration mode using A2.3: credential failure only with selected authentication; no intrinsic status failure for envelope fetch; codec obligations only for applicable encoding/decoding. Do not infer absence from constant-success speculation. Each member of `N` has default handler `e => http::request_failed(e)`.

Ordinary fetch/judge source explicitly declares the normalized upper bound, including all additional AI/authored errors:

```can
// Signature fragments: bodies and B7 assertions omitted.
fetch receipt load_json from service
    emits [http::request_failed]

judge float assess from classifier
    emits [http::request_failed, ai::invalid_question, ai::invalid_answer, output_failed]
```

An explicitly declared surplus error remains public. Raw infrastructure names are no longer required solely because native transport can produce them. Question `emits` bounds remain explicit and unchanged; they are not normalized independently. Judge still prepares the batch, sends once, validates all answers before any handler, then runs handlers in registration order. Earlier authored effects are not rolled back on later failure.

Selective recovery uses ordinary data matching:

```can
// Completion-arm fragment inside a call match.
http::request_failed => match http::request_failed.detail
    http::status_error => match http::status_error.status
        404 => ok receipt(0)
        _ => http::request_failed
    _ => http::request_failed
ok receipt found => ok found
```

No retry, additional request, cancellation, aggregate flattening or nesting change follows from normalization.

## B2. Operation wrappers

### Declaration and scope

Add contextual `wrap`, `handles`, `native`, `emitted`, `calculated` and `inherit` in their owning productions. A wrapper is a named top-level executable declaration:

```can
wrap cached_load from load_json
    emits calculated
    asserts
        absent: => ok receipt(0)
            using failure native http::status_error(404, [])
        busy: => http::request_failed(http::status_error(429, []))
            using failure native http::status_error(429, [])
    handles native
        http::status_error => match http::status_error.status
            404 => ok receipt(0)
            _ => inherit
```

This fragment assumes nullary `load_json` returns `receipt` and is a body-only fetch. `asserts` uses the exact inherited invocation grammar, including grouped state for a judge wrapper. Section order is `emits calculated`, nonempty `asserts`, optional `handles native`, optional `handles emitted`. At least one handling section and one arm are required. A second occurrence of any section is rejected.

`from` names exactly one fetch, judge or wrapper ultimately based on one. Inherit the complete ordinary input list, `near` flags where already permitted, grouped state, result type and connection identity. No repeated header return type, `given`, `state`, `from` connection replacement, generic wrapper parameter list or type-changing wrapper is admitted. Inherited input names are in scope in handlers; they are immutable. Judge state remains grouped and is not newly disclosed. A wrapper can be exported and called by its own name. Fetch wrappers have the same callable-reference eligibility as their target; judge wrappers remain subject to the existing grouped-state callable restriction.

Reject self/mutual base cycles, multiple bases, individual question/arm/LLM/ordinary-function targets, and repeated keys in one table. Declaration order within a package does not affect base resolution. Wrappers are operation-specific policy, not general inheritance.

### Two tables, one execution

Maintain two finite sets at the original operation boundary:

- `N`: raw native infrastructure obligations from B1.
- `E`: the original operation's non-native domain obligations, including complete declared bounds of called questions/helpers, escaping authored completions, native AI validation errors, and explicitly declared surplus errors. The intrinsic normalized contribution alone is not an emitted-origin entry. Preserve provenance where an authored `http::request_failed` or `codec::invalid_data` coexists with an intrinsic contribution.

A key is `(native | emitted, exact error specialization)`. `handles native` keys must belong to `N`. `handles emitted` keys must belong to `E`. An impossible key is a compile error. A derived wrapper can override a key that an ancestor has already consumed: lookup is against the original boundary sets, not just the ancestor's outward bound. A base-handler output is not a new input key. Default native rules normalize; default emitted rules forward unchanged.

Resolve each base chain before execution. The most-derived definition for a key replaces the predecessor; absent definitions inherit. Invoke the original target exactly once, observe its first terminal outcome, then apply at most one selected policy rule. A success bypasses policy. A domain failure carries private provenance established at its production boundary. Success from a native recovery completes the entire fetch/judge; it does not resume partially prepared questions or rerun handlers. A failure from authored code before target entry is not handled here.

An emitted-origin example is `handles emitted` with `ai::invalid_answer => ok fallback_value`. It recovers the judge's terminal outcome, not an individual question, and does not undo handlers already run. A handler that fails, including by calling another wrapped fetch, propagates out immediately. Never look up its new error in either table. Standard faults also escape directly.

### `inherit`, aliases and completions

Wrapper arms use B3 exact error patterns and optional aliases. They require `=>` and a terminal body, with ordinary `do`, value matching, call matching and named helper calls available. They have the inherited result type; returning another type is rejected. No standard catch or `ok` input arm belongs in a policy table.

`inherit` is a terminal completion statement allowed lexically in a wrapper handler, including its nested `do`/match branches. It invokes the immediately preceding rule for **that key and original failure**, with the same original inputs and occurrence. The predecessor can itself delegate. At the end of the finite chain, default native mapping or emitted forwarding runs. It cannot accept arguments, be stored, appear as an expression, escape through a callable or be called from a separate helper. It cannot retry the target.

A bare terminal error alias forwards that error under existing completion rules. In a native handler this deliberately exposes the raw error and adds it to the public contract. Authors wanting normal fallback use `inherit`. Returning a newly constructed `http::request_failed` explicitly maps a value but does not redispatch. `inherit` preserves the predecessor's behavior, not the currently selected child's rule.

### Calculated public errors

`emits calculated` is mandatory **only for `wrap`**. An explicit marker and target make the contract dependency visible; the compiler publishes the exact expanded finite upper bound and its provenance. Ordinary functions, questions, fetches, judges and LLMs still write `emits [...]`. An omitted wrapper marker, an explicit list on a wrapper, or `emits calculated` elsewhere is rejected. There is one source rule, not two equivalent wrapper spellings.

For each effective rule `h(k)`, calculate `escape(h)` from checked completion control flow:

1. `ok` contributes nothing; a forwarded or constructed error contributes its exact type.
2. A matched call contributes unhandled propagated outcomes plus errors escaping its selected handlers. Declared callee bounds remain upper bounds; do not inspect callee bodies to infer smaller sets.
3. `relay call` contributes the callee's full declared domain bound. Standard faults never enter this set.
4. Conditional/match branches contribute their union without proving branch feasibility. Incompatible or incomplete arms are still errors.
5. `inherit` contributes the immediately preceding rule's bound for the same key.

The wrapper bound is `union(escape(h(k)) for k in N ∪ E)`, with default rules included where not overridden. Distinguish native/emitted keys even if their error types match. Use exact nominal type identities, not registry numbers alone. Fully replaced rules contribute nothing unless delegated to. Do not subtract a normalized error just because one raw key is recovered: another native key can still normalize to it.

Handle dependency cycles explicitly: base cycles are rejected; calculated-bound dependencies through handler calls must be acyclic after ordinary explicit-bound calls are treated as leaves. Reject direct or indirect dependence on the same wrapper's calculated contract, including an `inherit` predecessor body calling the child wrapper. A named explicit-bound function can be a boundary, but its body must still typecheck against the calculated wrapper signature. No recursive effect-set solver is introduced.

A compile diagnostic for an escaping obligation includes its key, original target, selected handler and any `inherit` chain. Caller checking and callable compatibility consume the published calculated bound exactly as a written list. A changed base can therefore cause an ordinary caller's explicit contract check to fail; it cannot silently expand that caller's contract.

## B3. Exact generic-error patterns and match order

Extend an error-type arm head to `qualified_error [<type_arguments>] [as alias]`. Qualification uses existing package lookup. Type arguments use existing type grammar, with no new wildcards, inference search or constraints. For completion/policy arms, `as alias` binds the exact error **value**, not its message or fields.

```can
// Completion-arm fragment; a_failure and b_failure are named variants.
all_failed<a_failure> as first => relay call summarize_a(first.failures)
all_failed<b_failure> as second => relay call summarize_b(second.failures)
ok receipt value => ok value
```

An unaliased exact head may use the existing no-arrow forwarding shorthand, for example `all_failed<a_failure>` on its own line. An aliased head requires `=>`; it forwards explicitly through its alias when desired. Without an alias the existing short error-name alias is introduced in that arm's scope. With `as alias`, only that explicit alias is introduced by the head. The scrutinee retains ordinary narrowing in data matches. Explicit aliases avoid ambiguity and collisions; they do not change error identity. Forward with the bound value, such as terminal `first`, using the existing bare-error-value completion rule. No payload destructuring is added to completion heads.

A bare generic name is allowed only when its applicable input set has one concrete specialization. With two, diagnose ambiguity and show the exact alternatives. Explicit application matches that exact nominal specialization. It is not a wildcard over one type parameter, a subtype test or an implicit union. Bare/exact spellings covering the same instance are duplicates. Two exact specializations of the same declaration are distinct and can coexist. An explicit type-parameter reference in a generic body is checked at each concrete specialization.

| Context | Applicable set / ownership |
| --- | --- |
| Ordinary `match call` / `match chain` | Complete declared error union of the matched call/chain |
| Ordinary data match on an error-containing variant | Its exact nominal leaves; apply normal data exhaustiveness and narrowing |
| `concurrent` | Shared union of participants' domain bounds |
| `concurrent with error` | Each entry/spread's own bound, separately |
| `race with error` | Shared union of participants' domain bounds |
| Plain `race` | One newly created outer `all_failed<F>`; no direct participant-error arms |
| Wrapper policy | B2's origin-specific original key set |

For plain race, remove the Q6 rejection of distinct generic specializations in the collected leaf set. Keep complete types and deduplicate only identical types; retain all runtime occurrences. An explicit `all_failed<F>` arm selects a named finite failure variant `F`: every possible participant domain error plus `standard_failure` must inject into its leaves. Existing variants can list `all_failed<A>` and `all_failed<B>` as distinct leaves. The outer aggregate preserves those nested values without flattening. Reject an `F` omitting any required leaf. A bare `all_failed` can use the existing unique expected-variant inference, or remain private when the payload is ignored. Explicit `F` and expected uses must agree; no global variant search.

All success/error completion matches require every error arm, then the optional standard arm anywhere among those failures, then exactly one final `ok`. Individual error ordering is otherwise unconstrained. The rule applies to applicable shared/per-entry coordination regions and ordinary call/chain matches. It does not reorder ordinary data matches, question boolean/options/levels, or policy tables with no success input arm. Preserve existing mode-specific recovery result types. Standard failures in plain race remain aggregate members; `[_]` is not an outer participant arm.

## B4. Standard-failure snapshots

Use one value in ordinary and coordination standard catches:

```can
// Complete shape of a call-match fragment.
match call work()
    domain_failed
    [_] as standard_failure failure => ok failure.message
    ok str result => ok result
```

`[_] => body` remains an unbound catch. A bound catch must use `[_] as standard_failure name`; `str` and other binder types are rejected. There is no compatibility string-binder form. `.message` provides the former canonical text.

The opaque snapshot has read-only `int occurrence_id`, `str kind` and `str message`. Preserve C9's six propagatable kinds and message construction/sanitization rules. IDs are unique within one program run, stable for repeated observations of the same failure, and not promised stable between assertion workers or builds. Retain the native cause privately; no constructor, copy-update, wire encoding, raw exception access or public identity forgery is introduced. Snapshots remain admitted aggregate leaves; they are not domain errors and cannot occur in `emits`.

Taking a snapshot does not allocate another failure occurrence. An aggregate snapshot of the same occurrence has the same ID and observations. Storing/returning a snapshot as data does not rethrow it. This revision adds no new snapshot-rethrow expression; omitting a catch preserves automatic standard propagation. Do not coerce it into a domain failure or `ok` implicitly.

Ordinary catches still cover receiver/callee/argument evaluation and the matched invocation; coordination preparation stays outside participant catch regions. A fault raised by the selected catch body escapes that region. Fatal process failure is not promised catchable. Harness violations are separately recorded as sticky root failures: returning success after catching one cannot pass its assertion. Timeout termination is a supervisor result, not an injected catchable standard failure.

## B5. Verified build and publication

### Successful `build`

A successful `build` means all of the following were true for one captured build input identity:

1. The complete resolved source/dependency graph, manifests, registries, lock, referenced assets and fixture files passed their required validation.
2. Every required attached assertion in the graph, including dependency roots and native/wrapper roots, executed offline and passed; no selectors or cached passes omitted roots. Generic rows exercise their selected concrete instances, not all possible future instantiations.
3. No missing, ambiguous, malformed, mismatched or unused fixture, live-boundary attempt, sticky harness violation, root timeout, worker crash or required-coverage failure occurred. Every started participant was drained or the root failed its deadline.
4. Production generated TypeScript and its source maps passed existing output validation for the same checked snapshot and pinned compiler/catalogue/Bun/runtime identities.
5. The exact verified production generation was atomically selected as current.

Success does not certify live model quality, database/network availability, untested inputs, termination in general or runtime branches never exercised. It does not execute `main` against live services. A project with no reachable executable roots may have zero assertions; every declaration that requires attached assertions must still satisfy that static requirement.

### State machine and identity

The driver holds the existing exclusive writer protection and captures immutable input bytes before emission. Compile assertion and production plans from that captured graph, not independently reread working files. Fixture/asset bytes used by workers come from the captured snapshot. Hash all source/config/lock/catalogue/runtime inputs, raw fixtures, assets, assertion roots, timeout policy and relevant compiler options into verification identity; report the exact Bun version. Tests use fixture environment only, never ambient credentials.

For dependency locks, add required `fixtures_sha256` to each dependency entry. It hashes `can-fixture-tree-v1` followed by a zero byte, then all raw files statically referenced by that dependency's declarations/templates, sorted by UTF-8 bytes of their normalized manifest-relative paths. Encode each path byte length, path bytes, content byte length and content bytes using P2's unsigned 64-bit big-endian length framing. Deduplicate paths, retain distinct paths even for equal bytes, and hash just the prefix plus zero for an empty set. Resolve paths using the existing confinement rules before reading. Source fixture templates remain covered by `source_sha256`; external raw bytes are covered by this new digest. The build validates both against the captured dependency graph and never silently rewrites the lock. This prevents a dependency's test fixture from changing independently of its locked source identity.

`capture → check → private test staging → supervise every root → validate production staging → revalidate inputs → publish production`

Staged test artifacts are never production current. `assert` likewise stages privately and runs selected or all roots without calling production publication. A selected assertion success is labelled partial and cannot be reused as a full build attestation. No `--skip-asserts` or unlimited deadline exists on the verified build path. Existing checking/parsing commands may still be used as explicitly nonpublishing operations.

A failure at any prepublication step reports nonzero status, removes temporary executable staging when possible, and leaves the previous production current unchanged. Keep bounded diagnostic reports independently of current selection. Recheck the captured input identity immediately before publication; if inputs changed during verification, fail with `inputs changed` and require a new build. A later workspace edit cannot retroactively alter the artifact's captured identity; it simply makes the workspace differ from the published generation.

Publish an immutable generation through the existing atomic current-selection mechanism. An interrupted or failed publication must leave either the prior complete generation or the new fully verified generation selected, never a partial generation. The successful build report identifies generation ID, input identity, assertion totals/evidence/timeout policy and validation status. A publication interrupted after atomic selection but before response is an uncertain command result, not an unverified artifact. Existing reader leases remain valid for their complete generation.

### Deadlines and failures

Both `build` and `assert` accept `--assert-timeout-ms N`, a decimal integer in **1..600000**, default **5000**. No zero, negative, infinite or environment-derived value is accepted. This is a per-root wall-time budget recorded in reports and verification identity, not a Can source annotation. Run roots sequentially in stable `(package, declaration, assertion)` order initially; each uses its own Bun worker and fresh harness state.

The driver starts a monotonic deadline immediately before worker launch. It includes startup, assertion execution, all started-participant draining, fixture exhaustion checks and final root-result delivery. A worker cannot pass after its deadline: the supervisor's observed completion time decides, with timeout winning at equality. A CPU loop cannot prevent driver expiry. A pass received while participants/queues remain outstanding is an invalid worker result.

On expiry, terminate the worker through the platform process API; do not wait for Can cleanup or an in-worker timer. Allow at most 1000ms for process reaping after termination, then report a supervisor failure if exit cannot be confirmed. No production publication follows an unconfirmed termination. Worker launch/crash/protocol errors fail that root; safe subsequent roots may run for a complete report, but a supervisor isolation failure stops the suite. Any failure prevents build publication. Host suspension or scheduling delay can delay observation; this is not a hard real-time guarantee.

Workers report root start and invocation/fixture progress to the driver. A timeout report contains root identity, configured/observed durations, last-known pending paths and last-known executing source span when available. Mark stale or unavailable progress honestly. Reports do not include ambient secrets or raw private causes. Timeout is not recoverable in authored Can and does not alter production `Promise.race([])` pending semantics.

## B6. Typed fixture reuse with local ownership

Add an inert top-level `fixture` declaration for one exact executable target, with optional typed `given` parameters and a nonempty `cases` section. Its name follows ordinary package lookup/import/export rules. It is test data, not a callable or an assertion root, and is erased from production output.

```can
// Reusable content for the exact function find_receipt(str).
fixture absent_receipt for find_receipt
    given
        str key
    cases
        key => missing_receipt(key)

// Fragment at one lexical invocation; sample is still a local root selector.
match call find_receipt(requested)
    when
        sample: use absent_receipt("r-7")
    missing_receipt => ok receipt(0)
    ok receipt found => ok found
```

`for` and `use` are contextual within fixture productions. The template target can be an ordinary function, supported catalogue operation, fetch/judge/LLM or wrapper; native questions/arms are not independently invokable targets. A generic target must specify concrete type arguments in the template header. No generic template parameters or target-as-parameter mechanism is introduced.

Case input lists use the target's exact existing call grammar, including grouped state. Outcomes use the declared target result/error contract. Parameters have explicitly written types; no `near`, receiver capture, state section, effects or body execution is allowed in a template. Case expressions are restricted to the existing inert initialization expression subset with template parameters as additional immutable names. No calls, runtime locals, opaque-value constructors or new fixture-expression evaluator is introduced. Opaque values remain possible only through existing compiler-issued fixture representations, not template forgery.

At a lexical `when`, `selector: use template(arguments)` expands its cases **in place**, in source order, under that selector. Template arguments use the same inert subset in the site's static declaration environment, not runtime local bindings. Check target identity and exact generic specialization, arity, argument/result/error types at definition and expansion. Do not accept a template for a base operation at a derived wrapper site merely because signatures match. Existing ordinary literal rows can appear before or after an expansion; their relative FIFO order is preserved.

Expansion creates no extra call, root, callable instance or dispatch point. It owns no queue. Each use retains the existing queue identity `(root, lexical when-table)` and existing dynamic invocation/participant/callable fingerprint; repeated concurrent visits share that site's canonical FIFO assignment. Different sites/root runs get independent queues. Never search ahead for matching values. Preserve explicit-argument checks and frozen receiver/`near` comparisons. Diagnostics point to both template definition/case and local expansion plus the reserved invocation path.

Templates cannot include templates in this revision; recursive expansion and cycles are therefore impossible. A supplied-completion case remains consumer evidence and bypasses native work. A case may instead use B7's raw mode to execute a native target. The same target/ownership checks apply. Raw paths in templates resolve relative to the defining source file, and their content hashes enter build inputs. Template file bytes and referenced raw files in dependencies must also be covered by lock source identity; B9 specifies that revision. No function-ordinal root override, root selector in the template, wildcard argument match or detached expected-output registry is introduced.

## B7. Attached native and wrapper assertions

### Source attachment and execution modes

Fetch, judge, LLM and `wrap` declarations now require a nonempty attached `asserts` section. Place it after `given` and optional grouped `state`, before the method/asks/registration body; on wrappers it follows `emits calculated`. Existing ordinary function/arm rules remain otherwise unchanged. Individual questions continue to be tested through judge-owned cases, not separate transport roots.

`using`, `raw` and `failure` are contextual only in the assertion/fixture execution-mode production; they are not newly reserved data names. Native rows use the exact invocation inputs and final state group, followed by an expected success/domain completion. Under each row, indent exactly one execution-mode line:

```can
fetch receipt load_json from service
    emits [http::request_failed]
    asserts
        decoded: => ok receipt(7)
            using raw "fixtures/receipt-7.json"
    get "/receipt"
```

A named judge assertion supplying one state value writes `sample: (input_value) => ...`, not a flattened ordinary argument. The root invokes the declaration for real. `using raw "relative/path.json"` supplies a native exchange fixture at its owned target boundary while executing request construction, validation, decoder and handlers. Local calls made by those handlers retain their own lexical `when` tables and selected root name. A nested native call needs its own local fixture; the root's raw file cannot intercept another operation.

For wrappers, select either `using raw` for the underlying fetch/judge exchange, or `using failure native error_value` / `using failure emitted error_value` for a policy-only test. Injected values must typecheck against the exact original `N` or `E` key set. Injection occurs after root inputs are bound and before the policy lookup, skips the underlying operation and makes a fresh private occurrence with the selected origin. It cannot forge a standard failure or accept an arbitrary phase string. Parent delegation executes real handler code. Report this as `policy-fixture`, not request/decoder evidence.

At a native lexical `when` case, an indented `using raw` line changes that row from supplied-completion substitution to real-target execution with the local raw exchange. The row first checks actual arguments/fingerprints, then runs the operation and compares the resulting completion against the row expectation before delivering it to the caller. A mismatch is a sticky harness failure. Raw mode is allowed on fetch/judge/LLM/wrapper targets only; ordinary function calls cannot use it to seize nested boundaries. Raw/injected modes on inappropriate declarations, multiple mode lines and modes on a supplied opaque mock are compile errors. Injection is confined to attached wrapper assertions, not consumer `when` rows.

### Raw HTTP fixture contract

Use an external, strictly checked UTF-8 JSON object, identified by `schema: "can.native-fixture.v1"`. These are test artefacts, not arbitrary executable host adapters. Reject duplicate/unknown fields. Paths are source-relative, resolved within the owning project under the existing symlink/path confinement rules; no network loading. Required top-level fields are:

| Field | Contract |
| --- | --- |
| `schema` | Exactly `"can.native-fixture.v1"` |
| `target` | Canonical `package::declaration` of the lexical/root invoked target, including the wrapper name when applicable |
| `environment` | Object of fake credential-variable string values; only credential names read by the inherited connection are allowed, missing key simulates absence |
| `exchange` | `null` for expected failure before transport, or one request/response exchange object below |

An exchange has exactly `request` and `outcome`. Request has `method` (uppercase string), `url` (complete resolved request URL, including query, before redirects), `headers` (ordered `[name,value]` pairs) and `body` (body expectation). Compare headers at the compiler-owned pre-send boundary using A4's native Header normalization and order, before host-added transport headers. Exact equality is required; no ignored/unmatched authored headers. Simulated authorization is included in expectations. Host-generated connection/content-length headers are not authored request obligations and are not included at that boundary.

Body expectation is exactly one of `{"bytes_base64":"..."}` or `{"json_utf8":"..."}`. Base64 must be canonical and decodes exact bytes; empty bytes express no body. The JSON mode first validates UTF-8 and uses the existing exact-token JSON reader on both expected text and actual body, rejecting duplicate keys and ignoring object member order/insignificant whitespace. Compare decoded strings, booleans, null and arrays normally; compare number-token spellings exactly, without converting integer tokens through binary64. Thus `1` and `1.0` differ, and a numeric formatting change intentionally requires updated request expectations. Field additions/removals, changed state, prompt, thresholds, criteria or model fail equality. This comparison is test-only reuse of the selected codec, not another production JSON implementation. Response body always uses raw bytes so malformed payloads can reach the decoder.

`outcome` has exactly one of:

- `response`: object with `status` (integer 200..599), `headers` (raw ordered string pairs), `body_base64` (canonical base64). Run the real response/header/body/codec/protocol validation on it.
- `failure`: one of `{"kind":"transport","phase":"connect"}`, with phase any existing A2 transport phase; `{"kind":"timeout"}`; or `{"kind":"body_limit"}`. These represent the selected native boundary failure and derive limits from the connection, not untrusted arbitrary payload fields. They execute the real native-error normalization path without wall-clock waits. Body-limit injection denotes inbound consumption; outbound overflow must be caused by the actual prepared request.

Request comparison happens before releasing any outcome. A malformed *fixture schema* fails harness validation. A well-formed fixture containing malformed *provider bytes* is deliberate input and must produce the adapter's declared error, not a fixture-configuration error. A request mismatch fails the root before response delivery. `exchange: null` forbids a transport attempt; a pretransport failure can pass its expected completion, while attempted transport is sticky unexpected-boundary failure. Conversely a non-null exchange left unused by preparation failure fails the root as unused fixture. Each owned exchange is consumed exactly once; redirects are represented by the final response according to the existing adapter contract, not by extra Can invocations. No fixture enables live I/O.

A minimal response fixture for the preceding root:

```json
{
  "schema": "can.native-fixture.v1",
  "target": "app::load_json",
  "environment": {},
  "exchange": {
    "request": {
      "method": "GET",
      "url": "https://example.invalid/receipt",
      "headers": [],
      "body": {"bytes_base64": ""}
    },
    "outcome": {
      "response": {
        "status": 200,
        "headers": [],
        "body_base64": "eyJpZCI6N30="
      }
    }
  }
}
```

This fragment assumes `receipt` has `int id` and `service` has that endpoint, no auth/default headers and no additional generated request headers. Fixtures for actual modes must include their complete generated header contract; this example is not proof of today's adapter output. Raw HTTP protocol tests do not claim live transport or target conformance coverage.

### Required coverage and evidence

Native declarations require their own nonempty attached cases. At least one attached raw case must compare a prepared request and reach its post-response decoder; a supplied result, wrapper injection or pretransport-only failure cannot satisfy this obligation. Additional `exchange: null` cases cover preparation failures. A declaration without any passing request/decoder case cannot claim the required native coverage and fails the build; there is no inferred exception or unchecked coverage waiver.

Each locally authored wrapper key must be selected in at least one **attached** wrapper assertion; either raw or injected evidence can establish policy selection. Inherited keys need no duplicate child source assertions. Run the base's assertions independently; do not inherit their expected outputs, because an override can intentionally change them. Cover handler branch behavior with ordinary named cases; selection coverage alone is not full branch proof. A wrapper exposing only recovery still tests raw request behavior on its original native declaration.

Judge-owned raw cases must exercise its actual registered question preparation, whole-batch validation, and handlers. Require at least one valid-answer raw case reaching handler dispatch for every judge. Dynamic registrations may vary by input; the build report lists reached question/handler identities rather than claiming universal coverage. Catalogue conformance independently covers probability/threshold boundaries, generated-record assembly, static/dynamic choices, malformed later answers and source-ordered handlers. Authors add named cases for their own thresholds/fallbacks. No live model-quality requirement is introduced into build.

Keep evidence labels distinct: `real-can`, `supplied-completion`, `raw-provider-fixture`, new `policy-fixture`, `bun-conformance`, `live-quality`. A root can have several labels; report mode/path per exercised boundary. Mutating a prompt, request argument, grouped state field, expected header, handler result or frozen callable capture must fail the appropriate test with local ownership intact.

## B8. Acceptance matrix

The [complete acceptance evidence specification](language-change-acceptance-2026-09-22.md) expands these cases across all 16 accepted changes, including before/after programs, negative cases, fixture ownership and native lowering. BC identifiers remain stable references.

These are required implementation evidence, not claims that new forms compile today. Each row needs positive and negative fixtures plus the stated runtime observation where applicable.

| ID | Case | Required observation |
| --- | --- | --- |
| BC01 | Every reachable raw error; seven detail constructors | One `request_failed` with unchanged typed detail, sanitized fields, new mapped/private original occurrence |
| BC02 | Native decoder and authored handler both emit `codec::invalid_data` | Only native origin normalizes by default |
| BC03 | Body-only versus envelope 404 | Normalized status error versus response success |
| BC04 | Invalid JSON versus valid malformed AI answer; bad later question answer | Codec detail versus `ai::invalid_answer`; no handler runs before full validation |
| BC05 | Fetch/judge argument failure, descriptor helper failure, nested standard fault | Correct outer/inside origins and separate standard propagation |
| BC06 | Base → child → grandchild; omitted rule; selective `inherit` | Last override wins, omission inherits, one operation, finite predecessor delegation |
| BC07 | Handler returns another table's key or calls a failing wrapper | Propagate once, no redispatch/retry; standard handler fault also escapes |
| BC08 | Replace emitting ancestor, delegate on one branch, recover one of several native keys | Correct calculated finite bound and provenance; ordinary caller explicit bound remains enforced |
| BC09 | Illegal target, duplicate key, wrong origin, changed signature, base/effect cycle, misplaced `inherit` | Local diagnostics reject each; no silent inference or fallback |
| BC10 | Two generic specializations in ordinary data/call/chain and each applicable coordination mode | Exact arms discriminate; ambiguous bare, duplicate alias/exact coverage and missing specialization reject |
| BC11 | Race of calls emitting distinct nested `all_failed` values | One outer `all_failed<F>` preserves each nested value/occurrence; insufficient `F` rejects; no container covariance |
| BC12 | Success-first and standard-before/after domain arms | Success-first rejects; both failure orders accepted; data/question patterns unaffected |
| BC13 | Standard snapshot caught and observed in aggregate | Stable same-run occurrence/kind/message; constructor/update/emits/string binder reject; selected-handler fault escapes |
| BC14 | Success after caught harness violation | Root remains failed |
| BC15 | Failing root, CPU loop, pending race, startup crash, drain hang | Finite supervised failure and last-known diagnostics; no production publication |
| BC16 | Full build versus selected assert; all dependency roots | Full graph tested for build; selected assert reports partial and never switches production current |
| BC17 | Source/fixture edit during run, invalid TS, interrupted publish, active reader lease | Input change/validation prevents update; current always complete; old lease remains usable |
| BC18 | Same fixture template at two sites, recursive/concurrent visits, literal/expanded rows | Independent lexical queues and deterministic in-place FIFO, with exact arguments/captures |
| BC19 | Wrong template target/specialization, runtime capture, nested expansion, unused row | Reject or sticky failure with both definition and use locations |
| BC20 | Native raw request mutation, malformed provider response, malformed fixture schema, fake auth absent | Request mismatch versus declared decoder failure versus harness validation versus normalized missing credentials |
| BC21 | Raw fixture on wrapped judge plus handler's own locally mocked I/O | Judge/request/handler execute; nested mock belongs to its own call site/root |
| BC22 | Wrapper policy injection, missing local-key test, changed inherited result | Label policy evidence; missing selection coverage fails; parent tests independent |
| BC23 | Eight-fetch regression, no restructuring | Eight calls unchanged; seven raw errors disappear from ordinary public declarations/arms; typed selective recovery passes |
| BC24 | Native with only pretransport cases, judge with no valid-answer case | Coverage failure; additional negative cases cannot substitute for request/decoder/handler evidence |

### Rejected source examples

These are isolated negative fragments, not one program:

```can
// Wrong: calculated contracts are specific to operation wrappers.
fn int ordinary
    emits calculated

// Wrong: questions have no independent transport-wrapper boundary.
wrap changed_question from likelihood
    emits calculated

// Wrong: inherited state remains a final group for a one-state judge.
call wrapped_judge(email_text)
// Required invocation shape: call wrapped_judge((email_text))

// Wrong when both all_failed<a_failure> and all_failed<b_failure> are in scope.
all_failed => ok 0
// Use two exact specialization arms; one bare head cannot cover both.

// Wrong: an ordinary bound standard catch receives the snapshot.
[_] as str message => ok message
// Required binder shape: [_] as standard_failure failure => ok failure.message

// Wrong: inherit is not a reusable expression or helper call.
int value = inherit

// Wrong: a template argument cannot read a runtime local.
sample: use absent_receipt(requested)
// Use inert scenario data, such as the literal shown in B6.
```

For each rejection, the diagnostic points at the invalid head/input and identifies the required contract. It does not suggest an identity-erasing rewrite such as flattening state or merging error specializations.

## B9. Authority, traceability and implementation boundary

| Existing contract | Revision here |
| --- | --- |
| C2/C3 grammar, declarations and scopes | `wrap`, calculated bounds, exact error heads/aliases, fixture declarations, native assertion attachments: B2/B3/B6/B7 |
| C4/C5 nominal types, matching and completion ownership | Exact discrimination and errors-first order only: B3; no covariance or broad binder rewrite |
| C9 public domain identity and standard failures | Catalogue ID 1106 and detail variant B1; exact patterns B3; ordinary snapshots B4 |
| A2.2/A2.3 raw catalogue and native exported bounds | Preserve raw definitions; normalize fetch/judge and track origin using B1/B2 |
| A3/A7 executable grammar and judge phases | Wrapper forms/native asserts B2/B7; keep grouped state and batch phases B1 |
| A11/A12 native testing | Authored raw request cases B7 supplement conformance |
| Q5/Q6 arms, aggregate inference and snapshots | Exact specialization and preserved nested aggregates B3; consistent snapshot B4 |
| P2/P3/P4/P5 validation, assertion identity, raw fixtures and harness | Captured fixture inputs/build B5; local expansion B6; native roots/raw schema/evidence B7 |
| P15 release/verification boundary; driver build/assert output selection | Same-snapshot assertion-gated atomic production publication B5 |

The [disposition ledger](language-design-dispositions-2026-09-22.md) remains the complete scope inventory. This design closes the requested contract gates for LD07–LD12, LD15, LD17, LD22–LD26 and the fixture content design LD27, reopened by the user's latest request. LD27's new implement disposition is limited to the concrete inert lexical templates above. LD28 root-owned symbolic overrides remains deferred. Other dispositions are unchanged, including the separate runtime-check design gate in LD29.

Evidence comes from the current [driver build](../../compiler/internal/driver/commands.go), [assert driver](../../compiler/internal/driver/assert.go), [output publication](../../compiler/internal/driver/output.go), [native catalogue](../../compiler/internal/catalogue/catalogue.json), and C/A/Q/P specifications linked above through the [technical reading map](../syntax-taste/technical-spec.md). These were read, not modified into a new compiler. No proposed Can example was passed off as an admitted fixture. The [consultation record](../syntax-taste/evidence/2026-09-22/behavior-contracts/README.md) saves three freshly rewritten nine-choice requests, responses, audit and payload-disagreement investigation.

Implementation must use native operations for promises, process supervision, headers, byte handling, codec parsing and atomic filesystem publication, adding only adapters required by these contracts. The acceptance matrix supplies the next plan's evidence obligations. This document itself changes no compiler, runtime, golden or completed task status.
