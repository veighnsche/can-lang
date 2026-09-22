# AI, connections, fetch, generation, and wire contracts

## A1. Status, authority, and scope

This is the selected AI/I/O specification, reconciled 22 September 2026, completing technical contracts recorded in [the current decisions](decisions.md) and findings F01, F04–F08, F12–F13 of [the deep review](deep-design-review-2026-09-20.md). It is not an implementation claim. The complete current decision record, review, project AGENTS.md, and ASTRA_STDLIB capability catalogue were read. Historical syntax, adapters, decimal arithmetic, uncertainty bands, and proof machinery are not inherited.

**User-selected surface:** native `emits` sections; typed `choice_arm<T> emits [...]` values; `llm str` and record results; ordinary `given` inputs followed by grouped `state` arguments; named fetch method/query/headers, explicit `body json`, immutable `http::response<T>`; explicit numeric-to-string library calls. The error names, numeric IDs, data fields, metadata keys, protocol profiles, validation limits, and operational policies below are **technical specification choices**, not additional user taste selections. Body encoding names `text` and `bytes` complete the previously approved encoding-slot design. Examples labeled fragments omit surrounding declarations deliberately.

Can executes on Bun. Server-rendered HTML and upstream HTMX supply the initial web interaction model. This specification does not introduce a browser compiler, project-authored backend adapters, embedded executable code, or executable generated descriptions. LLM tool calling is entirely out of scope. It is not deferred work or an adapter capability to implement.

The initial maintained protocol profiles are `typesafe_systemone_v1` for judgments and `openai_responses_v1` for generation. This is a concrete production design choice, with documented wire mappings in A8/A10; it is not a claim that those adapters already exist. Local endpoints are allowed only when compatible with their selected protocol. The selected OpenAI profile uses nonstreaming stateless Responses requests and a conservative generated-schema subset. No generated Can provider request or production-adapter validation was performed. Separate direct Jev design consultations are recorded in the [advisory evidence](jev-design-consultations-2026-09-20.md).

## A2. Shared library data and finite error identities

### A2.1 Data shapes

These are catalogue contracts expressed with ordinary record notation, not project registration points:

```text
/// One immutable HTTP header entry.
record header
    str name
    str value

/// A fully consumed response snapshot.
record response<body_type>
    int status
    header[] headers
    body_type body
```

Their canonical qualified identities are `http::header` and `http::response<T>`, with positional constructor order exactly as shown. They contain no native Response, stream, mutation method, or resource handle. Ordinary construction and `with` are permitted: these data records do not attest receipt from a real server. `status` is an ordinary int; actual fetch output validates 200–599. Server response construction applies the server catalogue's status rules separately.

The predeclared `choice_option` type is an ordinary immutable record with positional fields `str key`, `str description`. Thus `choice_option("billing", "Payments and refunds.")` is its constructor. It is one explicitly predeclared catalogue name, not an implicit import of every AI declaration. Construction itself is total; question preparation validates the complete candidate collection. It contains no executable handler.

`bytes::buffer` is a catalogue-owned opaque immutable byte sequence, backed by native byte storage. Authors can name and pass it but cannot construct its representation, mutate elements, use record `with`, or decode an arbitrary JSON object into it. Catalogue operations are:

| Operation | Signature and declared domain errors |
|---|---|
| `bytes::empty` | `() -> bytes::buffer`, `emits []` |
| `bytes::from_ints` | `(int[]) -> bytes::buffer`, `emits [codec::invalid_data]`; each element must be 0–255 |
| `bytes::to_ints` | `(bytes::buffer) -> int[]`, `emits []`; returns a new immutable collection |
| `bytes::from_utf8` | `(str) -> bytes::buffer`, `emits [codec::invalid_data]`; rejects unpaired surrogates |
| `bytes::to_utf8` | `(bytes::buffer) -> str`, `emits [codec::invalid_data]`; fatal UTF-8 decoding, no replacement characters |
| `buffer.length` | exact Can int byte count; no call marker |
| `codec::encode_json<T>` | `(T) -> bytes::buffer`, `emits [codec::invalid_data]`; T must satisfy A6 |
| `codec::decode_json<T>` | `(bytes::buffer) -> T`, `emits [codec::invalid_data]`; T must satisfy A6 |

Type parameters here are ordinary type parameters, not schema objects or a new JSON type. Unsupported static T is a compile diagnostic. Runtime-invalid values/bytes use the declared error. Byte adapters use native Uint8Array/Buffer/TextEncoder/TextDecoder operations with range, immutability, and Unicode checks; they do not expose backing aliases.

### A2.2 Domain-error registry

IDs 1100–1199 are reserved for this specification in the distribution catalogue. These are domain-error IDs, separate from standard-runtime-failure tags and compiler diagnostic codes. Qualified names are nominal identities. Fields and their positional order are fixed:

| ID | Error and payload | Precisely covers |
|---|---|---|
| 1100 | `http::invalid_request(str reason)` | Dynamic URL/path/origin/header/body-mode/config-value validation fails before sending. |
| 1101 | `http::credentials_missing(str variable)` | A selected credential environment variable is absent or empty at request preparation. |
| 1102 | `http::transport_failed(str phase)` | A recognized native transport/body failure; phase is `connect`, `body`, `protocol`, or `cancelled`. No complete usable result is available; headers may already have arrived. |
| 1103 | `http::timeout(int timeout_ms)` | This request's A4 deadline expires. |
| 1104 | `http::body_limit(int limit)` | Encoded outbound bytes or consumed inbound bytes exceed the selected byte bound. |
| 1105 | `http::status_error(int status, http::header[] headers)` | Raw final non-2xx response in body-only fetch or an AI protocol request; fetch/judge normalize it under A2.4. No arbitrary response-body text is included. |
| 1106 | `http::request_failed(http::failure_detail detail)` | Public fetch/judge infrastructure failure; A2.4 fixes its seven typed detail alternatives and origin boundary. |
| 1110 | `codec::invalid_data(str path, str reason)` | Invalid UTF-8/JSON, duplicate member, schema mismatch, unsupported runtime value, nonfinite numeric input, depth/node/byte budget, or exact numeric representation failure. |
| 1120 | `ai::invalid_question(str reason)` | Invalid evaluated instructions, criteria, option/level support, threshold, or batch count. |
| 1121 | `ai::invalid_answer(str question, str reason)` | Valid JSON fails the selected judgment envelope/answer/distribution contract. `question` is the stable registration identifier, or empty for a whole-envelope defect. |
| 1130 | `llm::refused(str reason)` | Recognized refusal or provider content-filter termination, rather than a value of the declared output type. |
| 1131 | `llm::truncated()` | Recognized output-token termination; no partial success payload is exposed. |
| 1132 | `llm::invalid_response(str reason)` | Generation envelope/output structure or completion state violates A10, including provider-declared failed/cancelled responses. |

Reasons are finite catalogue tokens defined at their production sites below; they are not raw provider messages, arbitrary thrown-value strings, credentials, state, request bodies, or model text. Paths use RFC 6901 escaping, with empty string for the root; a duplicate member points to the duplicate key. AI registration identifiers are `q0`, `q1`, etc. Error matching exposes ordinary payload fields; callers handle/forward each declared kind explicitly. Catching `[_]` does not catch these errors.

### A2.3 Full authored bounds and standard failures

Every question, arm, fetch, judge and LLM declaration writes `emits [...]`; derived `wrap` declarations use the explicit `emits calculated` rule in A3.2. It is the **entire exported finite domain-error upper bound**, including applicable intrinsic errors and escaping handler errors. The compiler checks the required set is a subset; it never silently adds an error. Extra declared kinds remain part of the public bound and must be handled by callers even when a particular implementation cannot currently produce them. Examples with `emits []` on network declarations in earlier sketches are not complete valid signatures under this rule.

Required exported obligations, before adding escaping authored errors, are:

| Declaration/operation | Required kinds |
|---|---|
| Native `noul`, ordinary/record `choice`, ordinary/record `score` | `ai::invalid_question`, `ai::invalid_answer` |
| `choice_arm` | No intrinsic domain kinds; its authored computation determines its bound |
| `judge` | `http::request_failed` for native infrastructure; union of the **declared** question bounds; `ai::invalid_question`/`ai::invalid_answer` for request-level defects. Authored errors sharing raw infrastructure names are preserved. |
| Body-only named fetch | `http::request_failed`; its raw native set contains 1100,1102–1105, plus 1101 if authenticated and 1110 for applicable codecs. |
| Envelope named fetch | `http::request_failed`; raw set excludes 1105. Byte-only, bodyless requests need no raw codec obligation. |
| `llm` | 1100,1102–1105; 1101 if authenticated; 1110; 1130–1132 |

Intrinsic sets are fixed by declaration mode, not narrowed by constant-success speculation. Fetch/LLM have no executable success body; their authored `emits` may conservatively expose extra errors but cannot manufacture user error conversions. Fetch/judge operation wrappers can convert their original boundary failures under A3.2; ordinary callers remain available. LLM conversions still use ordinary callers. Static invalid schemas/configuration/options are compile diagnostics, not errors to hide behind a caller arm.

Standard failures remain separate: primitive faults, unexpected compiler-generated defects and recoverable native exceptions outside the classified adapter boundary propagate by the ordinary standard-failure rules. Fatal process termination is not promised recoverable. Adapter catches surround only their own native URL/header/transport/codec boundary calls, never user handlers or the judge continuation. Known validation failures become the specific declared errors above; unexpected defects do not become successful defaults. Native failure strings are sanitized by the common standard-failure contract, not copied into these domain payloads.



<a id="a24-fetchjudge-normalization"></a>
### A2.4. Fetch/judge normalization

#### Public value and identity

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

#### Which failures are normalized

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

Compiler-owned encoding of values returned by descriptor expressions is inside normalization; evaluation of authored code that produces those values is outside the **native-origin** set even though it executes inside the declaration. The emitted-origin wrapper table in [A3.2](ai-io-spec.md#a32-operation-wrappers) can handle such authored domain failures after target entry.

The raw intrinsic set `N` is computed by declaration mode using A2.3: credential failure only with selected authentication; no intrinsic status failure for envelope fetch; codec obligations only for applicable encoding/decoding. Do not infer absence from constant-success speculation. Each member of `N` has default handler `e => http::request_failed(e)`.

Ordinary fetch/judge source explicitly declares the normalized upper bound, including all additional AI/authored errors:

```can
// Signature fragments: bodies and P4.1 assertions omitted.
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

## A3. Executable regions, signatures, and visibility

An inline question handler completes that question's handler region. In an ordinary question its successful payload has the header return type; in a generated-record declaration it supplies one field of the header field type. The judge continuation completes the judge. A nested `match`, `match call`, or `do` retains its enclosing region; it does not create an authored anonymous function. `relay call` forwards into that region, with ordinary success/error compatibility checks. Handler domain errors must fit the native declaration's `emits`; standard failures propagate without being recategorized as bad model answers.

Native question declarations are only registered inside a judge. A direct call outside a judge is a compile error because no shared-state invocation contract exists there. They are not ordinary callable references. `judge` and `llm` use final grouped state arguments, including `call empty_state_judge(())` or `call configured_judge(setting, ())` when no state fields are declared. Empty `given`/`state` sections are omitted. One state input remains parenthesized as `call summarize((email_text))`; the group is never an ordinary tuple value. A `given` variadic parameter, if present, is last among ordinary inputs; the parser separates the final state group before applying ordinary variadic arity rules. State fields are never variadic or `near`.

`callable fetch_name` is permitted with the ordinary callable signature and any already defined ordinary capture rules for its `given` inputs. Native question/arm names are not substituted for ordinary functions. Judge/LLM callable references are rejected initially: ordinary callable types do not describe grouped state arguments. A named top-level wrapper with ordinary inputs calls the judge/LLM and is the callable used by collection/coordination operations. No new delayed-call syntax is needed. Question, judge and LLM `given` sections do not allow `near`, since they have no selected reference-creation operation.

Question declaration scope contains its ordinary inputs, inherited package declarations, and its own explicit metadata binders; it does not capture the caller's locals or judge state variables. Only selected question expression results are serialized. Passing a value in `given` does not by itself disclose it to the model. A question handler can use a state value deterministically only if the author also passes that value explicitly as an ordinary input. The provider sees the judge's shared state through A7, independently of lexical handler visibility.

Within one native declaration, inputs and named confidence/score/selected-key binders cannot duplicate each other. Generated field names occupy the generated record's field table; their explicit metadata names are checked there too. Handler locals have ordinary child scopes. `%` is a contextual float expression in its owning Noul/Choice/level handler, not an identifier, a capture, an argument to an implicit global, or a general prefix operator. It remains available in nested expressions lexically inside that handler; it is not available in a separately declared function called from it. Pass it as an ordinary argument when needed.

Native executable forms are named and top-level. Fetch, judge, LLM and operation wrappers require nonempty attached assertions under P4.1; their raw cases execute request construction, decoding and actual handlers. Questions and arms do not acquire independent transport roots. Ordinary named function adapters retain mandatory assertions. A11 supplies complementary catalogue/adapter conformance.

### A3.1 Section grammar and indentation closure

Each executable native header is followed by exactly one `emits` section, then optional nonempty `given`. Repeated singleton sections are errors. No omitted `emits`, inferred state, inline body on the declaration header, or native generic-parameter declaration is accepted. `from` is mandatory for questions, judges, fetches and LLMs, absent for arms.

- Noul: `asks expression`, then optional same-indent `minimum expression`. True/false described handler lines are indented one level beneath the minimum when present, otherwise beneath asks. Exactly one of each, either written order; dispatch still uses its bool meaning. No confidence setting.
- Ordinary/record Choice has two admitted productions. Without minimum, optional `confidence as name` precedes `asks`, whose option block follows immediately. With ordinary Choice minimum, `asks expression` is followed at the same indentation by optional confidence then `minimum expression => fallback`, and options nest beneath minimum. Confidence may instead precede asks in that second form, but cannot appear twice. A confidence setting after asks without a following minimum is rejected; it would leave the option block without its selected parent. Record Choice has no minimum and uses the first production. These preserve both shown layouts without general section reordering. Dynamic description mode ends its option block with the shared selected-key handler. Static mode has an inline described handler or record spread on each option line.
- Ordinary/record Score: optional confidence and score binders, then optional ordinary-only minimum/fallback, all before the final asks section. Binder order may be either; it determines generated metadata field order. Levels nest beneath asks. Ordinary levels have descriptions only and end with exactly one `ok =>` handler; record levels each have their own described handler and no shared ok.
- Judge: given, optional nonempty state, mandatory asserts, one or more registration lines, final `ok =>` continuation. Question `call` registration lines cannot have their own when/match handlers. Registrations with nonvoid results require `as type name`; void registrations omit it.
- LLM: given, optional nonempty state, mandatory asserts, final asks expression, and no executable success handler.
- Fetch: given, mandatory asserts, method/path line, optional nonempty query, optional nonempty headers, optional body encoding/expression, in that order. Fetch has no state/asks or authored success body. Query/header entries are identifier `=` expression.
- Choice arm: emits, describes expression, ordinary completion body. No given/state/from section.

Connection settings may appear in any order, each once. Endpoint/auth/timeout/max-body use named single-value lines; auth alone has its selected `bearer env` prefix. Metadata entries are identifier followed by a literal (no `=`), while header defaults use identifier `=` literal as other header sections do. Compiler-known metadata keys are data names, not new global reserved words. Option/level descriptions and their completion arrows stay on one physical line unless the description is a selected multiline-string token or the handler is the ordinary `do`/match block form. Standard indentation and one-line rules otherwise apply. This grammar accepts both approved Choice confidence placements without imposing Score's different selected ordering on them.



<a id="a32-operation-wrappers"></a>
### A3.2. Operation wrappers

#### Declaration and scope

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

#### Two tables, one execution

Maintain two finite sets at the original operation boundary:

- `N`: raw native infrastructure obligations from [A2.4](ai-io-spec.md#a24-fetchjudge-normalization).
- `E`: the original operation's non-native domain obligations, including complete declared bounds of called questions/helpers, escaping authored completions, native AI validation errors, and explicitly declared surplus errors. The intrinsic normalized contribution alone is not an emitted-origin entry. Preserve provenance where an authored `http::request_failed` or `codec::invalid_data` coexists with an intrinsic contribution.

A key is `(native | emitted, exact error specialization)`. `handles native` keys must belong to `N`. `handles emitted` keys must belong to `E`. An impossible key is a compile error. A derived wrapper can override a key that an ancestor has already consumed: lookup is against the original boundary sets, not just the ancestor's outward bound. A base-handler output is not a new input key. Default native rules normalize; default emitted rules forward unchanged.

Resolve each base chain before execution. The most-derived definition for a key replaces the predecessor; absent definitions inherit. Invoke the original target exactly once, observe its first terminal outcome, then apply at most one selected policy rule. A success bypasses policy. A domain failure carries private provenance established at its production boundary. Success from a native recovery completes the entire fetch/judge; it does not resume partially prepared questions or rerun handlers. A failure from authored code before target entry is not handled here.

An emitted-origin example is `handles emitted` with `ai::invalid_answer => ok fallback_value`. It recovers the judge's terminal outcome, not an individual question, and does not undo handlers already run. A handler that fails, including by calling another wrapped fetch, propagates out immediately. Never look up its new error in either table. Standard faults also escape directly.

#### `inherit`, aliases and completions

Wrapper arms use [C5.1](technical-spec.md#c51-exact-generic-error-patterns-and-match-order) exact error patterns and optional aliases. They require `=>` and a terminal body, with ordinary `do`, value matching, call matching and named helper calls available. They have the inherited result type; returning another type is rejected. No standard catch or `ok` input arm belongs in a policy table.

`inherit` is a terminal completion statement allowed lexically in a wrapper handler, including its nested `do`/match branches. It invokes the immediately preceding rule for **that key and original failure**, with the same original inputs and occurrence. The predecessor can itself delegate. At the end of the finite chain, default native mapping or emitted forwarding runs. It cannot accept arguments, be stored, appear as an expression, escape through a callable or be called from a separate helper. It cannot retry the target.

A bare terminal error alias forwards that error under existing completion rules. In a native handler this deliberately exposes the raw error and adds it to the public contract. Authors wanting normal fallback use `inherit`. Returning a newly constructed `http::request_failed` explicitly maps a value but does not redispatch. `inherit` preserves the predecessor's behavior, not the currently selected child's rule.

#### Calculated public errors

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

## A4. Connections and native transport policy

### A4.1 Configuration

A connection is an immutable compile-time declaration, not a runtime mutable client. `from` selects a connection on native questions, judges, fetches and LLM declarations. On `wrap`, `from` selects one operation/base wrapper and inherits its connection; it never implicitly selects a connection by value or lexical proximity. There is no implicit default connection, parent-directory configuration search, project-provided implementation, or live provider discovery during compilation.

Connection setting names are entries in the selected named-setting grammar, not globally reserved language keywords. The initial complete setting schema is:

| Setting | Type and policy |
|---|---|
| `endpoint` | Required nonempty string literal: absolute HTTP(S) URL, no embedded credentials, query, or fragment. Path allowed. |
| `auth bearer env` | Optional; nonempty environment-variable-name string literal. Only bearer or no authentication is supported initially. |
| `timeout_ms` | Required integer constant, 1–2147483647 inclusive; checked conversion to native timer units. |
| `max_body_bytes` | Optional integer constant, 1–67108864 inclusive; default 8388608. Applies separately to request and response bytes. |
| `headers` | Optional indented static `name = string` entries, using A4.3 name mapping; selected transport defaults, never AI prompt content. |
| `metadata` | Optional typed entries; mandatory for AI use. Closed key inventory below. |

Settings cannot call Can functions, read files, inject backend code, or inspect the environment except through the explicit authentication source. Duplicated/unknown settings, invalid literal values and conflicts are compile errors. The max-body default is a deliberate Can policy, not a fact inferred from an example or native fetch. Per-request byte accounting is bounded; global memory exhaustion remains a standard/resource failure.

Allowed metadata: `protocol` (string enum `typesafe_systemone_v1` or `openai_responses_v1`), `model` (nonempty string), and `max_output_tokens` (int, 1–32768, OpenAI profile only, default 2048). Protocol and model are mandatory when used by the respective AI form. Generic fetch can share a configured AI connection but ignores these recognized AI entries: none becomes a header or implicit body field. Unknown entries are compile errors, not arbitrary forwarding. TypeSafe rejects `max_output_tokens` at connection checking. No project can register another protocol string; adding an adapter belongs to the maintained distribution.

TypeSafe questions/judges require the first profile; LLM requires the second. Generic fetch is independent of profile. For AI, endpoint is the complete POST destination, with no appended path. Example endpoints are `https://api.typesafe.ai/v1/systemone` and `https://api.openai.com/v1/responses`. A local compatible endpoint may use HTTP; choosing it does not prove compatibility or change validation.

`auth bearer env` is observed once per reached request, after input preparation and before network launch. Missing/empty gives 1101. CR/LF, NUL or a value invalid for a native Authorization header gives 1100 with `credential_value`; it is never returned/logged. The captured value is used for that one request even if the process environment changes later. No credential is embedded at compilation or exposed to HTMX/browser assets. No auth means no compiler-added Authorization. Header-based custom auth schemes beyond bearer require a separately maintained transport contract; arbitrary metadata is not that mechanism.

### A4.2 URLs, redirects, attempts, deadline

Reject unpaired surrogates in evaluated request paths as invalid_request `url` before native URL conversion. Named fetch resolves its evaluated path with native `new URL(path, endpoint)`, then requires exactly the endpoint origin. Relative paths follow native base-path rules: against `https://example.test/api/`, `items` produces `/api/items`, while `/items` produces `/items`; without a trailing slash `/api` acts as a final segment. Fragments, embedded credentials, non-HTTP(S) schemes, and changed origins are rejected before authentication is attached. Absolute same-origin request URLs are permitted. Query text in the path is rejected; use the query section. Endpoint normalization happens once; no string-concatenation URL resolver is emitted.

Every reached fetch/judge/LLM makes at most one application request attempt. Use native fetch with `redirect: "manual"`; never follow a redirect or forward credentials to its destination automatically. Generic response-envelope fetch may inspect its 3xx status/Location; body-only fetch and AI requests return 1105. No retries, SDK retries, provider failover, cookie jar, or persisted conversation are implicit. DNS/connection behavior internal to one native fetch is not an extra application attempt. An explicit caller retry is a new invocation with a fresh deadline and fixture occurrence. A lost response/timeout does not prove remote side effects did not happen.

The request deadline starts immediately before native fetch launch, after ordinary Can argument/descriptor computation, credential lookup and request encoding. It covers waiting for response headers, body consumption, UTF-8/JSON decoding and response validation; it ends before Can handlers run. Use an abort signal/timer and monotonic elapsed-time checks before exposing the validated result. Synchronous validation cannot be preempted by a timer while the event loop is blocked; a late completed validation still returns timeout. This is a bounded-wait policy, not a hard realtime guarantee. The native operation is aborted on timeout; no rollback is promised. Runtime-owner cancellation maps a recognized abort to 1102 phase `cancelled`, unless the request's deadline already expired, in which case 1103 wins.

Inspect body-only/AI status as soon as headers arrive; on rejected status cancel/discard the body and return 1105 without decoding it. For accepted status, count bytes delivered by the native body stream, after native content decoding. Cancel and return 1104 on the first chunk that exceeds the limit. Do not buffer unbounded bytes and check afterward. The runtime uses native stream reading/materialization, with a byte-count adapter. Compare encoded outgoing UTF-8 bytes to the same limit before launch. Dispose/cancel remaining stream resources in all paths. Cleanup failure does not replace an already selected domain error; unexpected cleanup faults go to the runtime owner diagnostic path.

### A4.3 Headers and query values

Source `headers` keys are identifiers mapped by replacing each underscore with `-` and ASCII-lowercasing: `content_type` means `content-type`. No other key transformation occurs. Source duplicates after mapping are compile errors. Values are str or str[]; array entries append in order, and empty arrays contribute nothing. Require scalar Unicode before native header validation; invalid surrogate code units produce `header_value`. Native header byte-string restrictions also apply; no implicit Unicode-to-ASCII escape encoding occurs. The connection header defaults are static str entries; request-level entries replace the corresponding complete default value list, then native Headers performs value validation/normalization. Known dynamic invalid values produce `invalid_request("header_value")`. Source literal failures are compile diagnostics.

The compiler owns `host`, `content-length`, `transfer-encoding`, `connection`, `upgrade`, `proxy-authorization`, `proxy-connection`, `keep-alive`, `te`, and `trailer`; application entries for these are rejected. Bearer configuration owns Authorization: a conflicting connection/request entry is rejected rather than choosing precedence. AI adapters own Content-Type and Accept (`application/json`), so conflicting values are compile errors. Named JSON bodies require JSON Content-Type; text/byte defaults are specified in A5. Transport sends no automatic ambient cookies.

Fetch returns a **normalized header snapshot**, not original header-line bytes. Lowercase names sort lexicographically. For every name except `set-cookie`, retain the single normalized native Headers value; native combination may have erased physical repeated-line boundaries. For `set-cookie`, use native `getSetCookie()` and retain separate entries in its returned order. Do not comma-split arbitrary response values. Repeated `http::header` records remain representable as ordinary data; this normalization is the fetch-production contract, not a universal record invariant. Status-error headers use the same snapshot.

Query entries are `name = expression`, where name is a literal source identifier retained exactly, including underscores. Values are str or str[]; repeated values preserve array order; empty arrays omit that key. Query values must contain scalar Unicode; invalid surrogate code units produce invalid_request `query_value` before native URLSearchParams could substitute a replacement character. No automatic int/bool/float conversion, `null`, missing sentinel, or map-to-query coercion exists. Entries preserve source order. Use native URLSearchParams append/encoding: spaces become `+`, literal plus signs become `%2B`, percent signs are encoded, and an empty str produces `name=`. Do not pre-escape values. The native URL receives the resulting search string. Dynamic query/header expressions are evaluated once, in written section/entry order, before I/O; ordinary expression calls obey ordinary awaiting/error typing.

The following request/transport clauses name raw failures at their production sites. Fetch/judge apply A2.4 before exposing them to callers; authored expression failures remain emitted. LLM uses its separately specified public bounds.

## A5. Named fetch: request and result modes

The finite method inventory is `get`, `head`, `post`, `put`, `patch`, `delete`, `options`, mapped to uppercase HTTP tokens. There is exactly one method/path line. GET/HEAD cannot have a body. Other methods may omit it or have exactly one `body json expression`, `body text expression`, or `body bytes expression`. JSON accepts an A6-admissible concrete type, text requires str, bytes requires bytes::buffer. There is no implicit form encoding, multipart encoding, query-from-record conversion or body serialization inferred from an arbitrary value. A body expression is evaluated once before launch; request expressions follow written order after inputs.

A present JSON body has Content-Type `application/json`; a supplied Content-Type must equal that media type, optionally with `charset=utf-8`. A text body defaults to `text/plain; charset=utf-8`, bytes to `application/octet-stream`. Explicit text/bytes Content-Type overrides are permitted as ordinary validated header values, since media type does not change their encoding. Text always encodes UTF-8 with no BOM. Absent body adds no Content-Type. Header conflicts statically knowable from literals are compile errors; dynamic conflicts produce `http::invalid_request("content_type")`. A5 does not silently serialize text as a JSON string.

The declared result selects decoding without another source clause:

| Header result | Body decoding / status policy |
|---|---|
| Ordinary record T | A6 JSON into T; require 2xx |
| str | Strict UTF-8 text, including empty text; require 2xx |
| bytes::buffer | Raw delivered bytes, including empty bytes; require 2xx |
| http::response<T> | Same body mode for record/str/bytes T; retain any received final status 200–599 |

Other result roots, including primitive numeric/bool, arrays, variants and nested response envelopes, are rejected initially; wrap JSON data in an ordinary record. This is a named-fetch interface restriction, not a restriction on the standalone codec. HEAD supports only str/bytes or envelopes of them. An empty successful body is not `{}`, `null` or a missing record: JSON decoding fails. Consequently a JSON-record envelope can fail decoding a non-2xx error body of another shape. To inspect arbitrary statuses/bodies, declare `http::response<bytes::buffer>` and explicitly decode in ordinary code after inspecting status. The envelope never fabricates a T on a rejected body.

JSON result decoding requires Content-Type `application/json` or a media type whose subtype ends in `+json`; an absent or different type produces codec reason `media_type`. A declared charset must be UTF-8 (case-insensitive); another charset produces `charset`. Text accepts any media type or absence but also rejects an explicitly non-UTF-8 charset. Bytes bypass media-type/charset checks. Use native header parsing for normalization with a small strict media-type adapter; ambiguous or malformed Content-Type produces `media_type`. A JSON BOM is rejected; text preserves an initial U+FEFF. No sniffing or platform-dependent decoding occurs.

The response owns a detached immutable snapshot after complete bounded consumption and validation. No stream, mutable Headers, live response or body-consumption method escapes. Transport faults after headers still produce `transport_failed("body")`, not a partial response. Native opaque/network statuses outside 200–599 produce `transport_failed("protocol")`; this Bun target has no browser opaque-response success case.

For HTTP-client producers, the complete 1100 reason vocabulary is `url`, `origin`, `path_query`, `fragment`, `userinfo`, `header_value`, `header_name`, `query_value`, `credential_value`, `content_type`; the LLM producer additionally uses `instructions`. Platform/server catalogue producers may add their own explicitly enumerated tokens in P without changing error identity. Static occurrences are diagnostics. Native connection errors before usable headers map to phase `connect`, body-stream faults to `body`, impossible native HTTP status/protocol outcomes to `protocol`, recognized owner cancellation to `cancelled`. A JavaScript TypeError is not globally equivalent to a transport error: classify only within the specific fetch/stream call that raised it.

## A6. Shared exact typed JSON codec

### A6.1 Admitted values and representations

There is no authored untyped JSON value and no automatic shape coercion. The compiler derives a codec from the concrete type at each boundary:

| Can type | Wire representation and validation |
|---|---|
| bool | JSON true/false only |
| str | JSON string containing Unicode scalar values; reject unpaired UTF-16 surrogates after escape decoding |
| int | Any syntactically valid JSON number token whose exact mathematical value is integral, including `1.0` and `1e0`; derive it from the original token without binary64 rounding, under A6.2 budgets. Encode as signed decimal digits without fraction/exponent. Every spelling of negative zero decodes to integer zero. |
| float | JSON number converted by native binary64 semantics; must be finite; preserve negative zero; no int-to-float source coercion implied |
| T[] | Dense JSON array; each item decoded as T; preserve order |
| Ordinary nominal record/error | JSON object with exactly its declared field names, every field required, no additional fields; decode into that nominal type |
| Variant | Tagged object exactly `{"case":"canonical-leaf-type","value":{...}}`; validate tag against the variant's admitted record/error leaves, then decode that leaf |

Fieldless records/errors encode as `{}`. No null is admitted anywhere by these rules. `option::value<T>` uses the ordinary variant encoding of `option::none` or `option::some<T>`; it is never implicitly a JSON null/missing field. A canonical leaf tag is its fully qualified package/declaration name, followed when generic by `<` and comma-separated recursively canonical argument type names and `>`; array types append `[]`, with no whitespace. Tags identify actual concrete leaves, so `some<int>` and `some<str>` differ. Variant wrappers are codec representation only; they do not change the core unwrapped nominal-value representation.

Records emit fields in declaration order; variants emit `case` then `value`. Decoder object-member order is irrelevant. Exact required-member checks use own properties, never prototype inheritance. Duplicate members, even equal ones or keys with equivalent escaped spellings, are rejected. Error data encodes declared payload fields only, never implicit error IDs, native stacks, completion tags or messages. The special prelude standard-failure value and catalogue opaque types are not wire-admissible through this general rule.

Reject callables, choice arms, connections, opaque resources, bytes::buffer, void and containers reaching them at compile time. There is no automatic base64 representation for bytes; an author may choose an explicit declared string representation through a separate catalogue API. Recursive records/variants are codec-admissible if the core accepts their finite inhabitation and finite specialization graph. Values must be finite acyclic data; defensive cycle detection rejects a malformed native cyclic representation. Shared immutable subtrees may be serialized repeatedly without being mistaken for cycles.

### A6.2 Preserve numeric tokens while reusing native JSON

Native JSON.parse remains the syntax/value parser and JSON.stringify the encoder. An int must never be derived from the already-rounded Number value returned by JSON.parse. On supported Bun, a JSON.parse source-context reviver records each primitive's original token against its holder/property (including the root). Typed traversal subsequently uses that source for int and native parsed values for other fields. A bottom-up reviver cannot infer a field type merely from its key: two nested `count` fields can have different declared types. Keep the exact per-holder token association until schema validation completes. A token parsed natively as Infinity or zero can still denote a bounded exact int: decide from the source token when the expected type is int, without first applying float's finite check. For float, native parse rounding and finite underflow apply; overflow is rejected.

Integer decoding accepts mathematically integral JSON number tokens by the following bounded normalization. This is a narrow representation adapter over a token already accepted by native JSON.parse, not a replacement number grammar, arbitrary-precision decimal type, or general decimal arithmetic parser:

1. Split the token into sign, integer digits, optional fraction digits, and optional signed exponent digits. Concatenate the integer/fraction digits and remove leading zeros to obtain coefficient digits C. Let f be the number of original fraction digits. An omitted exponent is zero; exponent digits consisting only of zeros also mean zero regardless of their optional sign. Do not evaluate the token through Number, expand its exponent, or construct a bigint from an unbounded exponent.
2. If C is empty, the exact value is zero regardless of the exponent's sign or magnitude. Require one remaining canonical-integer byte, then return native bigint zero. `-0`, `-0.000`, `0e999999999999999999999` and `-0e-999999999999999999999` all take this path after native syntax validation. No negative-zero int exists.
3. Otherwise let L be C's length, t its trailing-zero count, s be 1 for a negative token and 0 otherwise, and R be the remaining canonical-integer byte budget below. Exact integrality requires e >= f − t. Compare the signed exponent token to this bounded threshold using sign, leading-zero removal, digit length and lexicographic digit comparison; do not parse an arbitrarily long exponent as a native Number or bigint. If it is smaller, fail `integer_token`, even if native Number underflow/rounding produced an apparent integer.
4. The canonical signed decimal length is D+s, where D = L+e−f. Compare e to the upper bound R−s−L+f in the same bounded way. A larger exponent fails `byte_limit` before output allocation. Perform the integrality check before this budget check, so a nonintegral token consistently fails `integer_token`. Threshold magnitudes are at most 2B+1 for effective byte budget B, and therefore exactly representable as native integer counts on the bounded target.
5. Only an exponent between the two bounds is converted to an exactly representable native index count. For shift e−f >= 0 append that many zeros to C; otherwise remove exactly f−e trailing zeros. The preceding checks prove the removal is exact and leaves at least one digit. Prefix the original negative sign if needed, then use native BigInt on those canonical decimal digits. Charge D+s against the remaining budget. There is no rounding, truncation of a nonzero digit, or source-level int/float coercion.

The effective byte budget B is the existing standalone/connection codec budget. In addition to bounding input bytes, each decode operation shares an initially B-byte budget across the sum of all decoded int values' canonical decimal lengths, including minus signs and one digit for zero. This prevents many short exponent tokens from each expanding independently to B bytes. Traversal consumes this budget in the existing deterministic schema/array order. Coefficient/token inspection stays within bounded input storage; zero padding, canonical output allocation, and BigInt conversion occur only after the checks above. Budget exhaustion is `codec::invalid_data(path, "byte_limit")`; a nonintegral number is `codec::invalid_data(path, "integer_token")`; a quoted number is the ordinary `type` mismatch. No new numeric setting, error kind or authored syntax is added.

Examples: `9007199254740993.0` and `90071992547409930e-1` both decode exactly to 9007199254740993; `1.25e2` becomes 125; `100e-2` becomes 1; `1.25`, `100e-3`, and nonzero `1e-999999999999999999999` fail `integer_token`. Nonzero `1e999999999999999999999` fails `byte_limit` before expansion. For a single root int with fresh budget B, `1e(B−1)` denotes the explanatory token whose exponent is B−1 and occupies exactly B canonical digits; increasing that exponent by one fails `byte_limit`. A negative sign consumes another byte. These are exact decimal-value decisions, independent of native binary64 overflow or rounding.

Encoding first performs a schema-directed budget walk, alongside scalar/depth/node/cycle validation, before formatting the whole document. Start with remaining serialized-output budget R=B; reserve/count every array/object delimiter, comma, colon, variant-wrapper field and natively JSON-escaped field key in the exact selected field order. Charge each primitive's exact UTF-8 JSON length. The walk counts the selected representation; it does not concatenate JSON or implement a second serializer.

Before calling native String on an int, let s be 1 for a negative value and 0 otherwise. If R−s < 1, fail the applicable byte-limit completion. Otherwise compute native bigint threshold `10n ** BigInt(R−s)`, whose exponent is already bounded by B. Reject a nonnegative value >= threshold or a negative value <= −threshold **before decimal formatting**. Compare signed values directly: do not allocate an absolute-value copy of an arbitrarily large incoming bigint. The guarded String(value) is then at most R ASCII bytes; charge its actual length. Threshold intermediates are bounded by O(B) and are not retained in an unbounded cache. Zero follows the same one-digit budget rule. This uses native bigint comparison/exponentiation and native decimal formatting, not a decimal digit formatter. An equivalent cheaper native bound may prove a value fits before threshold construction: for example, when R >= 20, the strict range −10^19 < value < 10^19 guarantees the signed decimal representation fits. This avoids computing a budget-sized threshold for ordinary small integers; every path still proves the bound before formatting.

For a str or field key, first require its UTF-16 code-unit length <= remaining R, then validate scalar well-formedness and obtain native JSON.stringify(string) plus its UTF-8 byte count. Reject an actual escaped length beyond R before retaining that formatted fragment. This necessary precheck bounds temporary escaping/encoding storage by a constant multiple of the remaining budget even when every character needs escaping. Finite floats, bools and the explicit float-negative-zero token have bounded native formatted lengths, which are charged too. Formatted primitive fragments may be cached for the final pass only within the same total B-byte accounting; do not retain rejected fragments or per-node threshold caches. A failed budget check is `codec::invalid_data(path, "byte_limit")` for standalone encoding and `http::body_limit(B)` when preparing an HTTP/AI request body, as below.

After successful preflight, use JSON.stringify with generated schema-directed replacer/access adapters as the final formatter; encode int through native JSON.rawJSON(the guarded decimal bigint String(value)), and float negative zero through JSON.rawJSON("-0"). Never convert bigint to Number. Native JSON.stringify's default bigint exception and negative-zero erasure are not Can policy. Strings use native escaping after scalar validation. Unsupported nonfinite floats fail before native stringify can replace them with null. The resulting complete UTF-8 output is checked against the counted budget before exposure. A logical output limit is not a zero-extra-memory guarantee: bounded native intermediates and the final output may coexist, and an unrelated native allocation failure remains a standard failure. Generated compiler callbacks are backend machinery, not authored anonymous Can functions.

Native JSON.parse accepts duplicate object members and its reviver sees only the retained member. Therefore a narrow preflight scanner tracks object-key tokens and nesting. Decode each quoted key using native JSON.parse and reject repeated decoded keys within one object. The scanner does not construct values or supply a replacement JSON grammar/parser; the complete input must still pass native JSON.parse. Check byte/depth budgets first; for otherwise bounded inputs native syntax failure takes precedence over a pending duplicate-key diagnosis. No handwritten general JSON value parser, decimal arithmetic engine or Can-owned serialization tree is introduced.

Every codec operation has a maximum nesting depth of 64 (root container depth 1; each nested container adds one) and 1,000,000 value nodes (each object/array/scalar value counts once; keys do not count). Standalone encode/decode also has an 8,388,608-byte UTF-8 budget. Inside a request adapter, the connection's max_body_bytes replaces only that byte budget, preserving the same depth/node rules. Preflight guards bound native parser exposure; typed encoder traversal guards depth/nodes/cycles before stringify. Exceeding codec budgets gives 1110; crossing a transport request/stream byte limit gives 1104. Encode at the request boundary checks its byte count as 1104, while explicit standalone codec calls use 1110. The native allocator can still suffer an unrelated standard resource/native failure below a logical limit.

Decode bytes using TextDecoder UTF-8 fatal mode with BOM preservation, then reject a leading BOM for JSON. Strings supplied internally by provider wrappers still undergo scalar checking. There is no BOM stripping, replacement-character recovery, coercion from quoted numbers, omission of unknown fields, nullable fallback, or implicit date/timestamp conversion.

The finite 1110 reasons are `utf8`, `unicode_scalar`, `byte_range`, `invalid_json`, `duplicate_member`, `missing_member`, `extra_member`, `type`, `integer_token`, `nonfinite`, `variant_tag`, `cycle`, `depth_limit`, `node_limit`, `byte_limit`, `media_type`, `charset`. Structural validation reports the first defect in declared-field/array order; unknown members sort by decoded key for deterministic diagnosis. Syntax/resource errors use the root path unless a safely known structural path exists. This diagnostic ordering is not a promise to recover all errors from one payload.

## A7. Judge execution phases and the batch boundary

A judge has at least one statically listed question registration. Its return type is its final continuation result, not an implicit collection of answers. The following phases are ordered and normative:

1. Evaluate all invocation ordinary arguments then final grouped-state arguments left to right exactly once. Construct the shared state object with the declared state names in declaration order, even with one field; zero state is `{}`. All state fields must be A6-admissible. Only these explicitly selected state fields are disclosed as shared state.
2. Prepare registrations in source order. Evaluate each argument, `asks`, criterion description and threshold expression once under that question's declared inputs. These descriptor expressions may use ordinary expressions/calls only with empty domain-error bounds; standard failures propagate. No request has launched and no answer/handler result is in scope. Assign each occurrence the transport identifier `q0`, `q1`, etc. Repeating the same named question produces distinct IDs and does not deduplicate or cache it.
3. Validate the entire prepared question set and state before credential lookup/network launch. Each question must resolve to the exact same connection declaration as the judge. Imported aliases of that declaration are equal; independently declared connections with identical fields are not. Static mismatch is a compile error. The Can adapter permits 1–256 registrations per request, a Can limit rather than a claimed provider maximum. There is no dynamic question-registration collection in this syntax.
4. Encode one request containing the model, shared state and all prepared questions. Read the selected environment credential once and send one POST attempt. No question sends its own HTTP request. Connection metadata supplies only the profile/model/settings enumerated in A4.
5. Consume and validate the full response and **every registered answer**, including answers whose result will not be used. Require exactly the registered ID set. Do not execute any Can handler until all payloads satisfy A8. A malformed later answer cannot cause an earlier handler's external effect.
6. Execute question handlers in registration order. Ordinary Noul/Choice/Score executes one selected handler/fallback. Generated-record forms execute every field handler in source-expanded field order. Bind a question result only after that question completely succeeds. Stop at the first domain or standard failure; propagate it through the judge bound, without running later handlers or the final continuation. Earlier authored handler effects are not rolled back.
7. Enter the judge's `ok =>` continuation with all successful answer bindings and its own given/state inputs. It computes the declared result by ordinary completion rules. Its escaping domain errors also belong in the judge's authored bound.

Registrations cannot refer to any answer binding, including an earlier textual registration. Question handlers cannot access another registration's answer or the judge's lexical locals. Answer bindings exist only in the final judge continuation; repeat names in one judge are compile errors. The `as` type must equal the question's declared result. A void-result question uses a registration without `as` and still executes its selected handler. Removing an unused binding must never remove its request or handler. A dependency on an earlier answer requires a later, separate judge invocation under an ordinary chain or caller. That is also why LLM generation followed by a judgment is two requests.

A question exports its own validation errors plus handler errors; the enclosing judge exports their complete declared union plus its intrinsic request/codec errors and continuation errors. There is no per-question failure-recovery arm on registration syntax. The caller can handle the judge's error normally. This design preserves a single registered batch while retaining the ordinary finite public error contract.

## A8. TypeSafe System One judgment profile

### A8.1 Descriptors, probabilities and handler context

`asks` and each criterion/level description are required str expressions, nonempty after native Unicode whitespace trim, containing scalar Unicode. Noul requires exactly one true and one false description/handler. Choice requires 2–255 distinct options; Score requires 2–10 levels. These are deliberately bounded Can-profile constraints; a one-option Choice is rejected instead of relying on undocumented provider behavior. Description-only dynamic options have the same limits. Criteria keys are stable distinct nonempty scalar strings at most 256 UTF-8 bytes. Source option/level names satisfy ordinary identifier rules; runtime choice_option keys need not be source identifiers. No Unicode normalization or case folding occurs. All description/request lengths additionally obey A4/A6 budgets.

All probabilities, confidence values, thresholds, scores and `%` have type float. Probability/confidence/threshold values must be finite and in [0.0,1.0]. No automatic percentage scaling, rounding, default string formatting or int widening occurs. Explicit `minimum` expressions require float; literal `1` is not a threshold float. Values computed as NaN/infinity fail descriptor validation before launch. Bindings must use distinct ordinary names.

| Form | Selection, defaults and context |
|---|---|
| Noul | Probability of true `p >= minimum` selects true; default minimum 0.5; equality selects true. `%` is the same p in either handler, never its complement. There is no separate confidence binding or fallback. |
| Ordinary Choice | Use the provider's selected key after validating it is a maximum-probability key. Ties accept whichever maximizing key the provider reports; Can does not choose source-first or rewrite the distribution. `%` in its option handler is that key's probability. Optional confidence binder exposes provider confidence. |
| Ordinary Score | Ordered levels have indices 0..n−1. Expose provider weighted score through optional `score as name`, confidence through optional confidence binder. No argmax handler, automatic integer conversion or rounding. `%` is unavailable in its single success handler because there is no selected level. |
| Ordinary Choice/Score minimum | If present, confidence below the explicit threshold runs its fallback; equality runs normal success. If omitted, no minimum filter/fallback applies. The fallback can read named confidence/score metadata and ordinary inputs, but has no `%` context. |
| Record Choice/Score | Run every option/level handler with its own `%`; no minimum/fallback is accepted. Named confidence/score metadata becomes extra float fields without invoking a handler. No implicit metadata field or selected-key field is generated. |

A question may omit metadata binders when unused; metadata remains validated. `%` remains a float when the handler result is str: use `call text::from_float(%)`, whose core native String semantics do not multiply by 100. Record handler field type may be any ordinary data type including a named variant, but not void; the metadata fields remain float regardless of that field type. Ordinary questions/arms may return void, using ordinary void completion.

### A8.2 Concrete request/response mapping

The profile sends `{"model":model,"state":state_object,"questions":{"q0":descriptor,...}}`. Noul descriptor is `{"type":"noul","instructions":asks,"criteria":{"true":true_description,"false":false_description}}`. Choice descriptor uses `"type":"choice"` and `"criteria"` mapping each key to its description. Score uses `"type":"score"` and `"criteria"` as the ordered description array. Can thresholds/handlers/metadata binder names are local policy and are not forwarded as extra provider properties. Source Score names remain local field names; the provider indexes scale positions.

The response is a JSON object containing `model` string and `answers` object. The `model` may be a resolved model name rather than the requested alias. Validate its nonempty type but do not equate the strings. Accept documented auxiliary top-level metadata such as usage without exposing it to authored code. Answers must contain exactly the q-ID set. Noul answer has `type:"noul"` and finite `noul`. Choice answer has `type:"choice"`, string `choice`, finite `confidence`, and `probabilities` object. Score answer has `type:"score"`, finite `score`, finite `confidence`, and `probabilities` object keyed by decimal zero-based index strings, plus required `legend` with the same keys and exactly the submitted description strings. Unknown question/answer type is invalid. Required data must exist; provider metadata outside these required fields may be ignored only at the protocol-envelope layer, never in user-record decoding.

Choice distribution keys equal the complete submitted key set. Score keys equal `"0"` through `String(n−1)`. Every probability is finite in [0,1]; the sum must differ from 1.0 by at most 0.000001 in native binary64 arithmetic. This absolute tolerance is an explicit Can adapter acceptance rule for decimal serialization, not normalization. Preserve the received probabilities unchanged. A selected Choice probability must equal the maximum of the received probabilities exactly; ties remain the provider's choice. Score is in [0,n−1] and must differ from the native weighted sum by no more than 0.000001 × max(1,n−1). Preserve the returned score, not the recomputed one. Confidence is validated independently and is neither maximum probability nor automatically recomputed entropy. The provider documents it as a distribution-derived statistic but does not publish a contract sufficient to reimplement its formula.

Malformed JSON/duplicates/UTF-8 or codec budgets fail as 1110. A valid JSON envelope with missing answers, wrong types, null required data, extra/missing IDs, invalid range/distribution/selected key or inconsistent weighted score fails as 1121. Noul primitive numbers and protocol metadata are decoded by this maintained adapter, not an exposed untyped Can JSON type. Errors 401/422/429/529 remain ordinary non-2xx `http::status_error`; no hidden retry or guessed semantic conversion occurs.

The finite 1120 reasons are `instructions`, `criterion`, `option_count`, `option_key`, `duplicate_option`, `level_count`, `threshold`, `batch_count`. The finite 1121 reasons are `envelope`, `question_ids`, `answer_type`, `missing_value`, `probability`, `confidence`, `distribution_keys`, `distribution_sum`, `selected_key`, `selected_probability`, `score`, `legend`. Whole-envelope defects use question `""`; local defects name their q-ID. Validate IDs first, then answers in registration order, then required properties/distribution entries in their descriptor order.

### A8.3 Evidence and limits of provider authority

TypeSafe's current [API reference](https://docs.typesafe.ai/api), [Choice](https://docs.typesafe.ai/primitives/choice), [Score](https://docs.typesafe.ai/primitives/score), [Noul](https://docs.typesafe.ai/primitives/noul), and [confidence guide](https://docs.typesafe.ai/confidence) were inspected on 20 September 2026. They establish the shared-state mixed batch, independent question IDs, full Choice distribution/selection, Score's ordered weighted result, Choice's 255-option maximum and Score's 10-level maximum. These pages do not establish Can handler ordering, captured state, tie policy, exact numeric codec, retries, or confidence default. Those are selected above. Provider compatibility tests must verify request/response field spellings against pinned raw fixtures before adapter implementation is accepted; documentation is not evidence of an existing Can adapter.

## A9. Static arms, runtime descriptions and generated nominal records

A named choice_arm is a capture-free top-level executable value. It has no given/near/state section, implicit selected-key/confidence value, or free caller-local capture. `describes` is a nonempty scalar str constant expression under the core top-level initialization rules; its handler may refer to package declarations and `%`. There is no anonymous arm constructor or ordinary `call arm()` operation. Bare named arm values can be stored, passed and selected through ordinary typed data. Storing them does not execute a handler. The compiler represents them by a constant description and a generated native closure receiving its contextual probability; no generated user code is evaluated.

An arm's declared result type must exactly equal the expected stored field/handler type, and its declared error set must be a subset of the expected stored arm bound. A `choice_arm<T> emits [E]` field is invariant in T. To return different admitted leaves, declare each arm's result as the same named variant and use ordinary completion inclusion inside its handler; storing a narrower-result arm is not an implicit wrapper. An expanded question exports the full declared error bounds of every stored arm field, including arms whose implementation happens not to fail. A wider field contract remains wider after storing a narrower named arm.

A record spread of executable arms expands **static record field names** in their declaration order. Every field must have a compatible choice_arm type. Literal inline arms and multiple static record spreads can mix; reject duplicate expanded names and metadata collisions. Record expressions are evaluated once during descriptor preparation, and their selected arm values are retained for handler execution. They can select among already named compatible arms at runtime, but cannot create new field names or captures. Arrays of executable arms are not admitted: no name-generation policy was selected.

A spread of choice_option[] is the separate description-only mode. One or more such spreads concatenate in written order, followed by exactly one `ok str selected_key => ...` handler. No inline executable options or arm-record spreads may mix into that declaration. Validate the concatenated keys/descriptions/count before sending; duplicate keys across arrays are errors. Ordinary Choice confidence/minimum is still available. The shared handler receives the validated selected key and `%` as its selected probability; the fallback receives no key/% because no option handler is selected. Runtime keys remain str data, not source identifiers or type fields. Record Choice cannot use this dynamic mode. Generation produces these descriptions as data, never executable arms.

`record R choice T q` and `record R score T q` declare two package-level names: the nominal generated record R and question q. Both participate in normal declaration collision/export rules; neither is a compiler-private alias of the other. The generated record receives the declaration's doc comment. It has option/level fields in expanded source order, followed by explicitly named metadata fields in metadata-setting order. Each transformed field has type T; metadata fields have float type. Reject duplicate metadata names, metadata names matching an option/level, and collisions of either package-level name with an existing declaration/import. Reject R=q. Static generated field names remain stable regardless of description values or arm identity. Do not reserve an implicit `confidence`, `score`, `selected`, or question-name field.

Generated records support ordinary constructors, equality when eligible, serialization when eligible, copy-update, field access and independently explicit `provides` exports. Exported question signatures cannot hide their generated return type. No globally generated names such as `q_result` are introduced: backend helpers use hygienic non-source identities. Generic instantiation does not change user-visible record naming; generated native declarations themselves are nongeneric initially, though their concrete input/field types may instantiate ordinary generic records. This avoids an unselected native generic-declaration grammar.

## A10. Concrete production generation profile

### A10.1 OpenAI Responses mapping and schema subset

`openai_responses_v1` is a maintained concrete protocol profile, not a claim that all “OpenAI-compatible” servers implement it. The selected production endpoint is `https://api.openai.com/v1/responses`, with explicit bearer environment auth. A connection must set protocol/model. `gpt-4.1-mini-2025-04-14` is a documented example snapshot supporting Responses and structured outputs; selecting another model is an explicit configuration change whose compatibility is checked by fixtures/integration tests, not guessed from its name. No provider discovery or automatic fallback occurs.

LLM input preparation evaluates ordinary given inputs then the final state group, then `asks` once. `asks` requires nonempty scalar str. Its lexical scope contains both its given/state inputs, but only the explicit state object is automatically disclosed. Encode that object by A6 and pass its UTF-8 JSON text as the user `input` string; `instructions` is the evaluated asks string. This is a fixed role mapping: supplied state cannot become a new system/developer message or an arbitrary provider request property. The provider receives ordinary text whose provenance is application input; this contract does not guarantee instruction adherence.

The complete emitted request fields are `model`, `instructions`, `input`, `max_output_tokens`, `store:false`, `stream:false`, `background:false`, `truncation:"disabled"`, and `text:{format:...}`. No conversation ID, previous-response ID, continuation, implicit transcript, SDK default state or extra authored metadata is forwarded. For `llm str`, format is `{type:"text"}`. For an ordinary record result, format is `{type:"json_schema",name:...,strict:true,schema:...}`. Its compiler-generated name is `can_` plus 32 lowercase hexadecimal digits from SHA-256 of the canonical type/schema identity. This synthetic transport name is not a source declaration and cannot collide with source lookup. Schema descriptions are not inferred from arbitrary comments.

The initial generative output subset is an ordinary nonrecursive root record, recursively containing nonrecursive ordinary records, arrays, bool, str, int and float. Generate JSON Schema object/properties/required/additionalProperties:false; every declared field is required, every object is closed. Arrays use items, numeric types use integer/number, strings/booleans their corresponding types. Resolve concrete generics before schema generation. There are no optional/null fields, schema enums, refinements, tagged variants, error leaves, byte/resource/callable fields or recursive records in **this generation profile**, even though A6 can encode more. Reject unsupported types at compile time with a path to the unsupported field. This restriction is deliberately narrower than provider support; no second schema language or implicit lossy projection is exposed.

Can caps generated schema nesting at 8 container levels, total object properties after expansion at 1024, and UTF-8 schema bytes at 65536. Root depth is 1; array items/nested objects add a level. Expand repeated nonrecursive shapes deterministically rather than needing provider-specific reference semantics. These conservative Can limits sit below documented provider structural limits. Provider schema adherence is not a replacement for A6 validation: a mathematically nonintegral token for an int field, an integer expansion beyond the codec budget, a nonfinite parsed float, duplicate member, null or unexpected field still fails the exact Can contract. Integral spellings such as `1.0` and `1e0` decode exactly under A6.2; source typing remains unchanged. No casts or rounding repair an output.

### A10.2 Completion validation and error precedence

HTTP/UTF-8/JSON/size/deadline failures use A2/A4 before generation extraction. The protocol envelope permits internal JSON null values where the documented API uses them; this does not introduce a nullable Can value type. Require an object, nonempty `id`, `object:"response"`, a recognized status and appropriately shaped output. Extra provider envelope metadata may be ignored; user output-record fields may not. Native parsing retains original generated output text until typed decoding is complete.

For a bounded valid JSON response, apply these checks in order:

1. Missing/ill-typed envelope fields give `llm::invalid_response("envelope")`.
2. `status:"incomplete"` with incomplete_details.reason `max_output_tokens` gives `llm::truncated()`. Reason `content_filter` gives `llm::refused("content_filter")`. An unknown/missing reason gives invalid_response `incomplete_reason`. Never expose a partial output as success.
3. `failed` or `cancelled` gives invalid_response `provider_failed` or `provider_cancelled`; `queued`/`in_progress` gives `nonterminal_response` because this profile requests synchronous nonbackground completion. A completed response with a nonnull `error` gives `provider_failed`.
4. A completed output must contain exactly one completed assistant message, whose content is a nonempty array. Known reasoning items may accompany it and are ignored as provider envelope data; any other output item kind gives `unsupported_output`. Refusal content with a string refusal gives `llm::refused("provider_refusal")`, discarding its text from the error payload. Unknown content kinds or wrong types give `output_shape`.
5. Concatenate that message's output_text string parts in content order; require at least one text part when no refusal exists. No separators, whitespace trimming, fence stripping, JSON substring search or retry is inserted. Text result returns this scalar-valid string; record result decodes this exact string through A6. Invalid structured text is codec::invalid_data with its exact path/reason. Plain text may be empty if an output_text part exists.

Reasoning items are accepted only with `type:"reasoning"`, nonempty string id, and array summary; they never execute or become the user result. Unknown envelope fields in these ignored items need no recursive Can shape projection, but still count toward raw JSON budgets. Output text annotations/logprob metadata likewise do not alter decoded text. Refusal detection validates the complete content array's kinds/types before choosing refusal, preventing a malformed sibling from bypassing envelope checks. A content array mixing a valid refusal and valid text yields refusal. The finite 1130 reasons are exactly `provider_refusal`, `content_filter`; 1132 reasons are `envelope`, `incomplete_reason`, `provider_failed`, `provider_cancelled`, `nonterminal_response`, `unsupported_output`, `output_shape`. Invalid evaluated asks text is `codec::invalid_data("/instructions", "unicode_scalar")` for invalid Unicode and `http::invalid_request("instructions")` for empty text; add `instructions` to A5's 1100 vocabulary for this LLM-only producer.

A raw fixture that violates provider schema still reaches the runtime output validator; this is required defense, not evidence that normal provider responses will violate their contract. Provider refusal and token truncation are expected distinct error completions. Every LLM declaration must author the full relevant error list including these cases. There is no return-record fabricated from a refusal or failed schema parse.

### A10.3 Primary evidence and unsupported differences

The official [structured outputs guide](https://developers.openai.com/api/docs/guides/structured-outputs), [Responses create reference](https://developers.openai.com/api/reference/cli/resources/responses/methods/create), [text generation guide](https://developers.openai.com/api/docs/guides/text), and [GPT-4.1 mini model page](https://developers.openai.com/api/docs/models/gpt-4.1-mini) were read on 20 September 2026. They document Responses format selection, closed required-field JSON schemas, refusal handling, incomplete output and the selected model snapshot. OpenAI supports some recursive schemas that the initial Can profile deliberately rejects; provider capability alone does not select a Can codec or source feature. Other providers' generation endpoints, schema dialects, refusal envelopes, batching and streaming behavior are not interchangeable with this profile. An endpoint claiming compatibility must pass the same raw request/response conformance suite. No live generation request or production-adapter implementation was performed for this specification.

## A11. Conformance, native reuse and evidence boundary

Use the platform specification's deterministic raw-provider fixture harness for compiler/adapter conformance. These fixtures supply raw HTTP status/headers/bytes after matching the emitted method, URL, sanitized header shape and exact body. They exercise the real adapter; they are distinct from ordinary assertion supplied completions, which exercise consumer logic without codec coverage. Fixtures never require real credentials or paid traffic. P4.1 attaches authored raw fixture files through `using raw` and supplies fake credential inputs only in that test context, never as hidden production defaults.

The minimum fixture matrix includes:

- Exact integer round-trip `9007199254740993`; exact normalization of `9007199254740993.0`, `90071992547409930e-1`, `1.25e2` and `100e-2`; rejection of quoted/nonintegral values; all zero spellings and huge positive/negative exponents; signed and cumulative canonical-integer byte-budget boundaries with checks before expansion; encoder rejection of preexisting oversized positive/negative bigint before String, exact punctuation/escaped-key/string accounting and escaped-string temporary bounds; float -0 preservation, nonfinite float rejection, escaped duplicate keys, null/missing/extra fields, malformed UTF-8, and recursive finite values/depth exhaustion.
- Fetch repeated queries, URL-origin rejection before auth, normalized repeated headers/Set-Cookie, zero-length text/bytes, JSON empty-body failure, every error in its applicable intrinsic bound, rejected statuses before body decoding, accepted envelope statuses with bad body, deadline across body/validation, and cancellation cleanup.
- Noul default/explicit threshold equality; full false-branch original `%`; Choice ties honoring either returned maximum, out-of-range/conflicting distributions, Score weighted fractional values and legend validation; mixed batch/repeated question IDs, missing/extra answers, all-answer validation before any handler, first handler failure and no later-handler execution.
- Static arm spreads with widened finite error bounds, key collisions, dynamic duplicate/empty/one-option rejection, no runtime-generated record fields, metadata collisions and generated record constructors/serialization.
- OpenAI text and record success, refusal, content filter, truncation, failed/nonterminal states, invalid output envelope, duplicate generated members, unsupported schema compilation, and exact integer token handling after structured generation.

Native reuse: fetch/AbortController/Headers/URL/URLSearchParams/stream reading handle transport; Uint8Array/TextEncoder/TextDecoder handle bytes; JSON.parse/source context, BigInt, JSON.rawJSON and JSON.stringify handle values. The Can additions are typed schema checks, finite limits, duplicate-key guard, bounded exact-integer token normalization, immutable snapshots, grouped request mapping, handler dispatch and completion normalization. Do not replace these native operations with a general Can HTTP/JSON stack.

Read-only local probes used Bun 1.4.2. The native parse reviver received original token `9007199254740993`; constructing BigInt from it retained the value, while default parse rounded it. JSON.rawJSON enabled exact bigint emission and preserved a requested -0 token; default stringify rejected bigint. Duplicate JSON input retained only its last member and invoked its reviver once, confirming why a guard is necessary. Headers combined ordinary repeated values while getSetCookie retained separate cookie entries. URL and URLSearchParams exhibited the relative-path and space/plus encoding rules above. These are narrow observed facts, not completed adapter tests. The normative ECMA-262 [JSON.parse reference](https://tc39.es/ecma262/multipage/structured-data.html#sec-json.parse), [JSON.stringify reference](https://tc39.es/ecma262/multipage/structured-data.html#sec-json.stringify), and [Bun fetch documentation](https://bun.sh/docs/runtime/networking/fetch) support these native mappings. Compiler startup/target conformance must require the source-context/rawJSON/Set-Cookie capabilities; do not silently fall back to rounded JSON on an older runtime.

## A12. Closed execution traces

These are specification examples for the selected compiler, not claims that the current compiler accepts the syntax. Catalogue imports and error-qualified names follow the core/package specification. No native declaration hides transport failures behind `emits []`. Native declaration fragments omit the mandatory P4.1 attached assertions and raw files; complete declarations must include them. Source blocks are declaration fragments inside a package whose `uses` imports the shown catalogue packages; they are not standalone source files.

### A12.1 Generated data followed by runtime Choice

```text
/// One candidate generated as ordinary data.
record candidate
    str key
    str description

/// Generated question and candidate descriptions.
record suggestion
    str question
    candidate[] candidates

connection generator
    endpoint "https://api.openai.com/v1/responses"
    auth bearer env "OPENAI_API_KEY"
    timeout_ms 30000
    metadata
        protocol "openai_responses_v1"
        model "gpt-4.1-mini-2025-04-14"

connection classifier
    endpoint "https://api.typesafe.ai/v1/systemone"
    auth bearer env "TYPESAFE_API_KEY"
    timeout_ms 30000
    metadata
        protocol "typesafe_systemone_v1"
        model "jev-latest"

llm suggestion propose from generator
    emits [http::invalid_request, http::credentials_missing, http::transport_failed, http::timeout, http::body_limit, http::status_error, codec::invalid_data, llm::refused, llm::truncated, llm::invalid_response]
    state
        str email
    asks "Propose a routing question and two to five distinct department candidates with useful descriptions."

choice str select_route from classifier
    emits [ai::invalid_question, ai::invalid_answer]
    given
        str question
        choice_option[] candidates
    asks question
        ...candidates
        ok str selected_key => ok selected_key

judge str route from classifier
    emits [http::request_failed, ai::invalid_question, ai::invalid_answer]
    given
        str question
        choice_option[] candidates
    state
        str email
    call select_route(question, candidates) as str selected
    ok => ok selected

/// Copy one generated data candidate into the catalogue shape.
fn choice_option to_option
    emits []
    given
        candidate value
    asserts
        example: candidate("billing", "Payments.") => ok choice_option("billing", "Payments.")
    ok choice_option(value.key, value.description)
```

Invocation trace, in an ordinary caller's explicitly handled chain:

1. `call propose((email)) as suggestion proposed` invokes generation with state `{"email":"Refund my duplicate payment."}`. Its schema has required `question` string and `candidates` array of closed key/description objects. A raw completed assistant output_text containing `{"question":"Which department handles this?","candidates":[{"key":"billing","description":"Payments."},{"key":"technical","description":"Product faults."}]}` decodes to suggestion. It has no handlers or executable values.
2. `call proposed.candidates.map(callable to_option) as choice_option[] options` performs an ordinary empty-error-bound data transformation. The output keys/descriptions retain their values/order. The generic mapper's error bound is empty because the named callback's bound is empty.
3. `call route(proposed.question, options, (email)) as str selected` prepares q0 and validates the complete candidate set before a separate TypeSafe request. Its descriptor has `type:"choice"`, `instructions` equal to proposed.question, and `criteria` mapping billing/technical to the descriptions. State is the same explicitly passed email object. The generation result was not a question within this batch.
4. Synthetic TypeSafe response `{"model":"jev-fixture","answers":{"q0":{"type":"choice","choice":"billing","probabilities":{"billing":0.9,"technical":0.1},"confidence":0.8}},"usage":{"input_tokens":1,"output_tokens":1}}` validates. The shared handler receives key billing and `%` 0.9, returns str billing, and the judge returns that str. The caller's chain success continuation receives selected. Its error arms must cover the union of both network declarations, including generation-only refusal/truncation and judgment validation errors; C11 gives an equivalent consumer fragment using nested `match call` and named assertion fixtures.

The response is a fabricated raw fixture, not an observed model judgment. TypeSafe model aliases may resolve to another name; this fixture's name is intentionally irrelevant to routing. Thresholds were omitted, so confidence 0.8 does not introduce an implicit minimum.

Failure traces close at explicit completions:

| Changed fixture/input | Exact terminal result and stopped work |
|---|---|
| OpenAI completed message contains a well-formed refusal part | `llm::refused("provider_refusal")`; no suggestion, mapping, or TypeSafe request |
| OpenAI status incomplete, reason max_output_tokens | `llm::truncated()`; no partial suggestion |
| Generated JSON omits candidates | `codec::invalid_data("/candidates", "missing_member")`; no map/judge |
| Generated two candidates have the same key | suggestion decoding succeeds because keys are ordinary str data; later judge preparation returns `ai::invalid_question("duplicate_option")`; no TypeSafe request |
| TypeSafe q0 selects billing but gives technical probability 0.9 and billing 0.1 | `ai::invalid_answer("q0", "selected_probability")`; no handler |
| HTTP 429 from the generation provider | `http::status_error(429, normalized_headers)`; no hidden retry |
| HTTP 429 from the judge provider | `http::request_failed(http::status_error(429, normalized_headers))`; no hidden retry |

### A12.2 Envelope fetch and exact integer trace

```text
/// Payment submission.
record receipt_request
    int amount_minor

/// Server-assigned receipt.
record receipt
    int id

connection service
    endpoint "https://example.test/api/"
    timeout_ms 5000

fetch http::response<receipt> save_receipt from service
    emits [http::request_failed]
    given
        receipt_request payload
    post "receipts"
    body json payload
```

For payload receipt_request(9007199254740993), the one request is POST `https://example.test/api/receipts`, Content-Type application/json, raw body `{"amount_minor":9007199254740993}`. A fixture status 201, Content-Type application/json and body `{"id":9007199254740993}` returns `http::response<receipt>(201, normalized_headers, receipt(9007199254740993))` exactly. A status 409 with that same valid body also returns an envelope with status 409. Status 409 with `{"message":"duplicate"}` instead returns `http::request_failed(codec::invalid_data("/id", "missing_member"))`; use a bytes envelope for arbitrary error shapes. No raw status_error belongs to this envelope mode; an otherwise identical body-only fetch stops at 409 before decoding and exposes `http::request_failed` with status-error detail.

### A12.3 Full distribution and phase failure trace

For a record Choice with float option fields billing/technical and explicit confidence name conf, validated probabilities 0.25/0.75 and confidence 0.4 execute both handlers in billing-then-technical order. A billing handler `ok call text::from_float(%)` is a type error for that float field; choose field type str to obtain `"0.25"`/`"0.75"`, with conf still float 0.4. A scaling expression must use `100.0 * %`, not mixed int `100 * %`. If billing's handler produces its declared domain error, no technical handler or record assembly runs; the judge propagates that error. If a second registered answer is malformed, neither first-question handler runs at all, because whole-batch validation precedes handler execution. These are distinct phases and must have separate fixtures.
