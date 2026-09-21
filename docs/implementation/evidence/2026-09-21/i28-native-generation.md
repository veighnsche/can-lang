# I28 native text and structured generation

The compiler now lowers LLM declarations into an explicit stateless Responses
request plan. Only the final state group is encoded into `input`; ordinary given
inputs remain available to `asks` but are not implicitly disclosed. The asks
expression is lowered once. Both request state and returned records use the
shared exact codec.

`LLMSchema` admits closed, required-field, nonrecursive ordinary records containing
records, arrays, bool, str, int and float. It expands concrete shapes, reports
unsupported field paths, and enforces depth 8, expanded properties 1024 and schema
UTF-8 bytes 65536. Its deterministic name hashes the canonical type/schema pair.
The provider schema does not replace typed response validation.

The runtime sends the fixed nonstreaming, nonbackground, stateless request profile
with explicit model/output limit. It validates bounded protocol envelopes before
extracting text, preserves exact text concatenation, distinguishes refusal and
truncation, and rejects unsupported output kinds or malformed content. Refusal
selection occurs only after validating all content siblings. No retry, partial
record, fence stripping, substring extraction or live-fixture fallback exists.

## Evidence

- Schema unit tests cover closed nested objects, expansion, determinism,
  unsupported leaves, recursion and all three budget boundaries.
- Compiler tests cover concrete generic output, unsupported roots/fields and
  rejected stream/tool/previous-response configuration.
- Runtime response tests cover exact request fields/state integers, empty asks,
  invalid Unicode, reasoning/text parts, every finite status/error category,
  refusal precedence, duplicate/extra/null/nonintegral output fields and exact
  large integer spellings. Five tests, 127 expectations pass.
- `TestCurrentBundledGeneration` invokes the absolute staged runtime with an empty
  PATH. A synthetic-auth loopback server receives text, structured generation and
  a later Choice request. Text becomes explicit state for the structured call;
  a concrete generic nested record/array result becomes a transformed Choice
  description. The value 9007199254740993 survives provider decimal spelling,
  record decoding and the later request. Ordinary private given data is absent
  from provider input.
- Failure variants assert distinct 1130/1131/1132/1110 completions and exact
  request counts, proving no retries or later-stage execution. Empty asks fails
  with 1100 before credentials or launch. Ordinary consumer assertions make no
  provider requests.
- The compiled three-request flow also executes through exact raw HTTP fixtures
  with networking denied and no credentials. Its report requires both
  `raw-provider-fixture` and `real-can` evidence. Generated TypeScript passes
  strict checking.

The maintained profile follows A10 in `docs/syntax-taste/ai-io-spec.md`. Official
Responses create and structured-output documentation were checked during this
implementation; no paid or live generation call was used.

Final gate: `bun test runtime` passed 215 tests with 20827 expectations. Strict
TypeScript passed for the adapter and its tests. With the pinned Bun archive and
strict generated TypeScript enabled, `go test ./compiler/... ./tests/integration
-count=1` passed; integration completed in 87.492s.
