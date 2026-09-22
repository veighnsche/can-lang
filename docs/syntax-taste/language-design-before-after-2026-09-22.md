# Can language design — before and after

> Selected behavior designs: [final contracts and acceptance cases](../implementation/language-behavior-contracts-2026-09-22.md). Use those rules instead of the unresolved behavior sketches below.

> Current scope: [finding dispositions](../implementation/language-design-dispositions-2026-09-22.md). That ledger supersedes the recommendation status and suggested sequence below; proposed examples remain sketches, including retained or deferred alternatives.

A concrete companion to the [full review](full-language-review-2026-09-22.md). **Before** shows current code or a shortened example using current syntax. **After** shows the reviewed proposal, not implemented syntax. Where the review left a decision open, both alternatives remain visible.

Examples are fragments: package headers and unrelated declarations are omitted. Behavior diagrams and test outlines are explicitly labelled where no source grammar has been selected.

## At a glance

| Area | Before | After proposed in the review |
| --- | --- | --- |
| Fetch/judge errors | Repeat up to seven infrastructure errors in declarations and matches | One public error with typed details |
| Wrapper inheritance | Ordinary wrappers cannot override a failure already handled by an inner function | Derive a wrapper; write only additions/overrides |
| Match order | Current examples commonly put `ok` first | Error arms first, `ok` last |
| Build | Checks and publishes; assertion execution is separate | Required assertions pass before production publication |
| Hanging assertions | One pending root can prevent the final report | Supervisor-enforced deadlines and explicit timeout reports |
| Standard catch | `[_] as str message` | `[_] as standard_failure failure` |
| Generic errors | Different specializations collide at a bare error arm | Match the exact specialization |
| Native callables | Judge/LLM references need ordinary function adapters | Represent their inputs in callable contracts; two designs to compare |
| Higher-order helpers | Authored helpers require fixed callback error bounds | Prototype explicit finite error-set parameters |
| Captures | `callable combine` captures matching local names implicitly | Explicit bindings at reference creation |
| Callable arrays | `callable int (int) emits [][]` | `(callable int (int) emits [])[]` |
| Fixtures | Repeated expected arguments and results in lexical tables | Share checked fixture content, keeping call-site binding |
| Native tests | Supplied completions can skip request construction | Also test prepared requests and real handlers against raw fixtures |
| Runtime checks | Some fixtures deliberately divide by zero | A named check operation with an explicit reason |
| Resource escape | Runtime ownership checks are the correctness boundary | Earlier conservative escape diagnostics, retaining runtime enforcement |
| Error IDs | Manual globally unique integers plus mirrored registries | Investigate canonical nominal identity and generated report codes |
| Tooling | Renderer loses comments; semantic errors can point to line 1 | Comment-preserving formatting and precise semantic spans |

## 1. Fetch errors: seven public outcomes become one

These fetch fragments are simplified from the [eight-call fixture](../../compiler/testdata/current/fetch/main.can). The request expressions are shortened so the error contract is easy to compare.

**Before**

```text
fetch receipt load_json from service
    emits [http::invalid_request, http::credentials_missing, http::transport_failed, http::timeout, http::body_limit, http::status_error, codec::invalid_data]
    get "/json"

fetch str send_text from service
    emits [http::invalid_request, http::credentials_missing, http::transport_failed, http::timeout, http::body_limit, http::status_error, codec::invalid_data]
    given
        str payload
    post "/text"
    body text payload
```

**After — proposed public contracts**

```text
fetch receipt load_json from service
    emits [http::request_failed]
    get "/json"

fetch str send_text from service
    emits [http::request_failed]
    given
        str payload
    post "/text"
    body text payload
```

The language supplies the default normalization. The author does not declare the seven-member base policy or write a normalizing function for each fetch. `connection service` still selects the transport configuration.

The reduction continues into callers. The following two-call fragment intentionally keeps the same nesting.

**Before**

```text
match call load_json()
    ok receipt received => match call send_text("hello")
        ok str echoed => ok echoed
        http::invalid_request
        http::credentials_missing
        http::transport_failed
        http::timeout
        http::body_limit
        http::status_error
        codec::invalid_data
    http::invalid_request
    http::credentials_missing
    http::transport_failed
    http::timeout
    http::body_limit
    http::status_error
    codec::invalid_data
```

**After — normalization plus errors-first ordering**

```text
match call load_json()
    http::request_failed
    ok receipt received => match call send_text("hello")
        http::request_failed
        ok str echoed => ok echoed
```

The enclosing function declares `emits [http::request_failed]` instead of the seven raw errors. A caller that completely recovers from `http::request_failed` need not emit it. Additional application errors still have their own contracts.

For the full eight-fetch example:

| Repetition | Before | After |
| --- | ---: | ---: |
| Infrastructure entries in eight fetch bounds plus `main` | 63 | 9 |
| Infrastructure forwarding arms | 56 | 8 |
| Calls performed | 8 | 8 |
| Distinct infrastructure causes available for diagnosis | 7 | 7 |

See the [complete current fixture](../../compiler/testdata/current/fetch/main.can), [complete eight-call after sketch](evidence/2026-09-22/full-language-review/eight-fetch-proposal.can.txt), and [measurements](evidence/2026-09-22/full-language-review/measurements.json). The counts exclude the separate callable-reference test. This is an analytical comparison; the proposed contract has not been compiled.

## 2. Selective recovery: keep the detail inside the one error

A single public error does not mean a single undifferentiated string. Proposed `http::request_failed.detail` is a closed variant carrying the original seven typed payloads.

| Cause | Detail remains available |
| --- | --- |
| Invalid request | `reason` |
| Missing credentials | environment variable name, never its value |
| Transport failure | `phase` |
| Timeout | `timeout_ms` |
| Body limit | `limit` |
| HTTP status failure | `status`, normalized `headers` |
| Invalid data | `path`, `reason` |

**Before — recover from 404, forward everything else**

```text
match call load_json()
    http::invalid_request
    http::credentials_missing
    http::transport_failed
    http::timeout
    http::body_limit
    http::status_error => match http::status_error.status
        404 => ok receipt(0)
        _ => http::status_error
    codec::invalid_data
    ok receipt received => ok received
```

**After — inspect details only where needed**

```text
match call load_json()
    http::request_failed => match http::request_failed.detail
        http::status_error => match http::status_error.status
            404 => ok receipt(0)
            _ => http::request_failed
        _ => http::request_failed
    ok receipt received => ok received
```

The unhandled normalized occurrence is forwarded intact. Its original failure remains a private diagnostic cause. No retry is implied by this handling.

The review also considered seven new public detail-record types instead of reusing the original payloads. Both support this behavior. Reuse is the initial recommendation; the representation is not an adopted decision.

## 3. Wrapper inheritance: additions and overrides only

**Before — ordinary function composition is not policy inheritance**

An ordinary wrapper can recover from timeout and return a success. An outer function can no longer see that timeout to override its handling:

```text
Behavior trace, not Can syntax:

load_json encounters timeout
    → fallback function returns receipt(10)
    → outer function receives a success, not the original timeout
```

To change that earlier handling while keeping other recovery rules, the author must restructure or duplicate the policy. There is no current `wrap`/`inherit` declaration mechanism.

**After — proposed derivation syntax**

```text
wrap cached_load from load_json
    handles native
        http::status_error => match http::status_error.status
            404 => ok receipt(0)
            _ => inherit

wrap fallback_load from cached_load
    handles native
        http::timeout => ok receipt(10)

wrap reporting_load from fallback_load
    handles native
        http::timeout => ok receipt(20)
```

Each declaration inherits its target's inputs, result type, connection and omitted handling. The final override wins **before** the underlying operation runs; these are not three nested runtime catches.

| Native outcome | `cached_load` | `fallback_load` | `reporting_load` |
| --- | --- | --- | --- |
| Status 404 | `ok receipt(0)` | Inherited `ok receipt(0)` | Inherited `ok receipt(0)` |
| Timeout | `http::request_failed` | `ok receipt(10)` | **`ok receipt(20)`** |
| Status 429 | Inherited normalization | Inherited normalization | Inherited normalization |
| Body limit | Inherited normalization | Inherited normalization | Inherited normalization |

`inherit` invokes the previous handler for the same failure. It does not retry the operation. A new failure produced by a selected handler escapes; it is not caught again by that wrapper's table.

The derived public error bound comes from its effective handlers and passthrough outcomes. A fully overridden parent handler does not keep contributing errors. An `inherit` branch does contribute its predecessor's bound. The author writes only changes, rather than repeating the inherited signature or handling list.

This first proposal uses one base and a fixed input/result signature. Application-handler errors need an explicit origin category before they can be overridden; `handles native` must not accidentally capture an identically named error from user code.

## 4. Judge normalization: one wrapper around the batch

From the [native AI application](../../examples/native-ai/src/oracles/oracles.can).

**Before**

```text
judge records::triage_scored review from classifier
    emits [http::invalid_request, http::credentials_missing, http::transport_failed, http::timeout, http::body_limit, http::status_error, codec::invalid_data, ai::invalid_question, ai::invalid_answer]
    state
        records::triage_bundle bundle
    call pick(bundle.options) as str keyword
    call likely() as float n
    call rate() as float s
    ok => ok records::triage_scored(keyword, (n + s) / 2.0)
```

**After — normalization alone**

```text
judge records::triage_scored review from classifier
    emits [http::request_failed, ai::invalid_question, ai::invalid_answer]
    state
        records::triage_bundle bundle
    call pick(bundle.options) as str keyword
    call likely() as float n
    call rate() as float s
    ok => ok records::triage_scored(keyword, (n + s) / 2.0)
```

The question declarations do not acquire wrappers. The judge still prepares one batch, sends one request, validates all answers, and runs handlers in source order. The example's grouped state is unchanged here; its separate redesign appears below.

Only native preparation/transport/codec failures are normalized. If a question's authored handler emits `codec::invalid_data`, that handler error remains separately visible even though a native decoding error of the same name is normalized. Additional AI errors are not silently collapsed. This proposal does not extend normalization to LLM or SQL operations.

## 5. Match ordering: put the continuation last

The required ordering applies even when normalization is irrelevant. For example, using the [codec chain](../../compiler/testdata/current/native/questions.can):

**Before**

```text
match chain
    call codec::encode_json<choice_weights>(original) as bytes::buffer encoded
    call codec::decode_json<choice_weights>(encoded) as choice_weights decoded
    ok => ok decoded
    codec::invalid_data
```

**After**

```text
match chain
    call codec::encode_json<choice_weights>(original) as bytes::buffer encoded
    call codec::decode_json<choice_weights>(encoded) as choice_weights decoded
    codec::invalid_data
    ok => ok decoded
```

The operations and failure semantics stay the same. The review recommends compiler enforcement of the already-required ordering; it does not select an order among individual error arms.

## 6. Standard failures: a snapshot instead of message parsing

Current catch from [array contracts](../../compiler/testdata/current/arrays/contracts.can); `quotient` divides 10 by its input.

**Before**

```text
match call [1, 0, 3].map(callable quotient)
    ok int[] output => ok "unexpected"
    [_] as str message => ok message
```

**After — same returned message, richer binding**

```text
match call [1, 0, 3].map(callable quotient)
    [_] as standard_failure failure => ok failure.message
    ok int[] output => ok "unexpected"
```

The handler could also inspect `failure.kind` or `failure.occurrence_id`, using the snapshot already exposed through failure aggregates. `.kind` remains a string with a defined vocabulary; this does not introduce an enum. Standard failures remain outside `emits`, and their native causes remain private.

## 7. Generic failures: match the specialization directly

The [composition fixture](../../compiler/testdata/current/coordination/aggregate-composition.can) has `original_a` emitting `all_failed<a_failure>` and `original_b` emitting `all_failed<b_failure>`. Today it converts both to `all_failed<combined_failure>` before composing them.

**Before — excerpt of one conversion wrapper**

```text
match call original_a()
    ok
    all_failed => do
        combined_failure[] values = call widen_a(all_failed.failures)
        all_failed<combined_failure>(values)
```

There is a second wrapper for `original_b`. The current fixture also includes recursive array conversions. Its consumer calls the normalized functions:

```text
str selected = match call race with error
    normalized_a()
    normalized_b()
    ok int value => ok "unexpected"
    all_failed => ok call describe(all_failed.failures[0])
```

**After — consume the original types**

```text
str selected = match call race with error
    original_a()
    original_b()
    all_failed<a_failure> => ok call describe_a(all_failed.failures[0])
    all_failed<b_failure> => ok call describe_b(all_failed.failures[0])
    ok int value => ok "unexpected"
```

Here `describe_a` and `describe_b` are ordinary functions accepting `a_failure` and `b_failure`, respectively. They perform the same leaf inspection as the current `describe`; no array reconstruction is needed just to inspect one failure. The new syntax is the exact specialization on each error arm.

Apply this to explicit declared-error dispatch, including applicable coordination handlers. Plain first-success `race` keeps its single implicit aggregate arm. If a downstream API genuinely requires one common aggregate type, explicit conversion is still needed; the proposal does not introduce array covariance.

## 8. Native callables: remove calling-convention adapters

Current [LLM declaration](../../examples/native-ai/src/oracles/oracles.can), with the error list elided equally in the declaration sketches below:

**Before**

```text
llm records::triage_plan draft from generator
    emits [/* existing LLM error bound */]
    state
        str account
    asks "Draft"
```

```text
call draft((account))

// A plain fn adapter currently supplies the ordinary callable:
callable draft_account

// Currently rejected:
callable draft
```

The real `draft_account` adapter repeats the error bound, adds ordinary assertions and call-site fixtures, invokes `draft((account))`, and forwards outcomes. Emitted TypeScript inputs are already flat; the obstacle is the source/type contract.

**After A — prototype explicit state markers**

```text
llm records::triage_plan draft from generator
    emits [/* the same existing LLM error bound */]
    given
        state str account
    asks "Draft"
```

```text
call draft(account)
callable draft
```

A marked parameter is serialized as state. Ordinary arguments are not implicitly sent merely because they happen to be records. This example changes calling convention only, not the LLM failure contract.

**After B — prototype grouped callable contracts**

```text
Behavior contract; callable type grammar remains undecided:

Direct invocation keeps: call draft((account))
Reference becomes valid: callable draft
Callable types and higher-order calls represent the state group explicitly.
No authored draft_account adapter is needed just to reshape arguments.
```

Neither alternative was selected by the review. Compare mixed ordinary/state parameters, empty state, callback use and fixture matching before deciding. Removing an adapter must relocate its useful tests, not silently discard them.

## 9. Captures and callable-array types: make the structure visible

The current [capture declaration](../../compiler/testdata/current/callables/captures.can) includes:

```text
fn int combine
    emits []
    given
        near int prefix
        int value
        near int suffix
    asserts
        sample: 3, 4, 5 => ok 12
    ok prefix + value + suffix
```

**Before — the local names must match the capture names**

```text
int prefix = 3
int suffix = 5
callable int (int) emits [] action = callable combine
```

**After — proposed explicit capture bindings**

```text
int left = 3
int right = 5
callable int (int) emits [] action = callable combine with (prefix = left, suffix = right)
```

The remaining ordinary argument is still passed when calling `action`. Captures evaluate once and retain immutable values.

A separate type-readability change:

| Before | After proposed |
| --- | --- |
| `callable int (int) emits [][] actions` | `(callable int (int) emits [])[] actions` |

The type still means an array of callables, not a callable returning an array. Receiver ownership and `choice_arm` metadata are separate contracts and remain intact.

## 10. Higher-order functions: preserve each callback's error contract

**Before — an authored helper needs a fixed bound**

```text
// Signature excerpt using current syntax.
fn int apply
    emits [codec::invalid_data]
    given
        callable int (int) emits [codec::invalid_data] action
        int value
```

That contract cannot also accept a callback emitting an unrelated `domain::unavailable`. Broadening it requires naming the additional error; callers then see the broader bound. Built-in collection operations already support more general callback-bound specialization.

**After — conceptual signature, not proposed source grammar**

```text
For each finite error set E:

apply(
    action: callable int (int) emits E,
    value: int
) -> int emits E
```

| Actual callback bound | Specialized helper bound |
| --- | --- |
| `[]` | `[]` |
| `[codec::invalid_data]` | `[codec::invalid_data]` |
| `[domain::unavailable]` | `[domain::unavailable]` |

`E` is an explicit finite error-set parameter, distinct from a data type parameter. It does not mean “infer any failures from the body.” This remains a prototype candidate, justified only if a concrete authored combinator benefits.

## 11. Fixtures: share content without weakening matching

Current [SQL fixture rows](../../examples/native-ai/src/model/model.can):

**Before — repeated argument/result content**

```text
when
    sample: pool, "recent_accounts", records::search_parameters("%"), 25 => ok [records::account_row(1, "Ann"), records::account_row(2, "Bo")]
    down: pool, "recent_accounts", records::search_parameters("%"), 25 => sql::connection_failed("recent_accounts")
    triage: pool, "recent_accounts", records::search_parameters("%"), 25 => ok [records::account_row(1, "Ann"), records::account_row(2, "Bo")]
    unclosed: pool, "recent_accounts", records::search_parameters("%"), 25 => ok [records::account_row(1, "Ann"), records::account_row(2, "Bo")]
```

**After — proposed sharing behavior; template grammar is not selected**

| Define once | Expected arguments | Supplied completion |
| --- | --- | --- |
| `recent_accounts_ok(pool)` | `pool`, `"recent_accounts"`, `search_parameters("%")`, `25` | The two account rows |
| `recent_accounts_down(pool)` | The same argument tuple | `sql::connection_failed("recent_accounts")` |

| Binding at the existing lexical `when` site | Reused content |
| --- | --- |
| `sample` | `recent_accounts_ok(pool)` |
| `down` | `recent_accounts_down(pool)` |
| `triage` | `recent_accounts_ok(pool)` |
| `unclosed` | `recent_accounts_ok(pool)` |

These are parameterized fixture templates, not production calls. Expansion remains typed at the call site, with explicit assertion-root binding, argument comparison and existing FIFO/occurrence identity. Changing the query limit without changing the expected limit must still fail the test. Fixtures are not retargeted using a fragile numeric call ordinal.

## 12. Native tests: check the request that actually gets prepared

**Before — a consumer supplies the answer**

```text
match call draft((account))
    when
        sample: ("Ann") => ok records::triage_plan(["vip", "new"], true)
    // Outcome arms omitted.
```

This tests the consumer with a supplied completion. It does not prove that changing `asks "Draft"` preserves the actual request or provider handling. Raw provider fixtures already exist separately.

**After — add deterministic authored-native cases**

```text
Test-case outline, not Can grammar:

inputs:
    account = "Ann"

expect prepared request:
    configured operation/profile
    instructions corresponding to "Draft"
    state field account = "Ann"

supply:
    a raw response in the selected provider protocol

expect:
    decoded triage_plan, or the specified typed failure
```

The real preparation and decoder run. Judge cases also exercise the authored question handlers and threshold boundaries after whole-batch validation. Keep consumer fixtures for their separate purpose. Live model quality is not turned into a mandatory build dependency.

## 13. Build and assertion behavior: verification before publication

**Before — source-traced current paths**

```text
canlc build PROJECT
    check Can → emit production modules → validate output → publish

canlc assert PROJECT
    check assertions → emit/publish assertion modules → run assertions
```

The [build implementation](../../compiler/internal/driver/commands.go) does not execute assertions. The [assert command](../../compiler/internal/driver/assert.go) executes them separately. This differs from the README's claim that every build runs assertions.

**After — recommended build guarantee**

```text
canlc build PROJECT
    check → stage candidate output
          → execute required assertions for that source/runtime identity
              pass          → publish production output
              fail/timeout  → report failure; preserve previous production output
```

Assertion liveness changes too:

| Before | After proposed |
| --- | --- |
| An empty `race with error` can leave an assertion pending indefinitely | A finite supervisor-enforced deadline fails the assertion |
| A hung root can prevent the final suite report | Report the timed-out root and last known pending paths |
| Killing a shared runner can leave later roots without results | Explicitly mark later roots unrun |

A per-root budget requires isolated workers or supervisor-visible root-start events; a suite-wide timer alone is not a fresh budget for every assertion. Production `Promise.race([])` keeps its native pending behavior. Killing a test does not promise that cleanup finished.

## 14. Runtime checks: name the failed condition

Current [fixture helper](../../compiler/testdata/current/fetch/main.can), body excerpt:

**Before**

```text
match condition
    true => ok
    false => do
        int invalid = 1 / 0
        ok
```

The failure says integer division by zero, even when the intended check concerns a response value.

**After — illustrative operation spelling, not a selected API**

```text
call check(condition, "unexpected response value")
ok
```

The proposed operation would fail with an explicit reason. The final name, failure classification and assertion expectation syntax remain open. An application runtime check is not the same as a sticky harness violation; catching a fixture-infrastructure violation must not make a test pass.

## 15. Resources, identity and tooling: before/after behavior

These proposals need behavior comparisons more than new source syntax.

| Area | Before | After target | Decision boundary |
| --- | --- | --- | --- |
| Scoped handles | Runtime owner/lease checks enforce lifetime | Diagnose unsafe escape through direct results, records, arrays and callable captures before execution | Prototype conservative analysis; retain runtime checks. No escape reproducer was compiled in the review. |
| Error identity | Author allocates a globally unique integer and mirrors it in active/retired registry data | Use canonical package/declaration/specialization identity; generate report codes only when needed | Audit numeric report/wire consumers before changing the contract |
| Formatting | `parse --render` prints canonical structure but loses comments | Formatting preserves comments and can safely update source | Extend the existing renderer rather than claiming none exists |
| Diagnostics | Resolve/check errors can be anchored at the file's first line | Point to the actual missing arm, incompatible type or fixture argument | Carry source spans and explain the unmet contract |
| Long lines | Calls, arrays, constructors and assertions use one physical line | Evaluate multiline delimiter lists after measuring normalization's benefit | Layout grammar remains undecided |
| Native section order | Question kinds have differing orders; Choice admits two layouts | One documented order where semantics permit it | Retain the meaningful `noul`/`choice`/`score` kinds |
| Stopping a fold | Ordinary `.fold` cannot stop early; recursion can express stopping | A possible `fold_until` returns `continue`/`done` records | Catalogue/API candidate, not evidence that Can cannot express the algorithm |

## Designs that do not need a replacement

The review recommends retaining these foundations:

- The four coordination meanings and their native `Promise.all`, `allSettled`, `any` and `race` mappings.
- `match chain`; current code already uses it.
- Explicit nominal connection selection and the distinction between questions and network requests.
- Nominal data, invariant containers, explicit numeric conversions and immutable native-backed operations.
- Separate typed application errors. A distribution-closed union cannot contain every application-defined failure.
- Inert top-level initialization and the controlled platform catalogue.

Their “after” is clearer documentation, diagnostics or narrower improvements—not replacement syntax.

## Status

This document illustrates recommendations already developed in the [full review](full-language-review-2026-09-22.md); it does not adopt them or introduce another language redesign. The earlier [Jev evidence and disagreements](evidence/2026-09-22/full-language-review/README.md) remain the consultation record. Before snippets were checked against current source; after snippets and behavior outlines are not compiled implementations.
