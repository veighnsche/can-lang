# Clean-room authoring review

Status: **revised design reviewed; no implementation or execution evidence**.
I rechecked the current [authoring draft](../../native-can-test-authoring-2026-09-30.md)
against the parser, checker, assertion runtime and cited host scenarios. No
builds, tests, browser/server runs or native probes were launched.

The revised package-level `test_case[] cases` avoids an unasserted registry
function; package-level callable records are source-backed
([example](../../../../compiler/lsp_g03_test.go#L23)). The five case examples
now separate one complete subprocess function from four expressly incomplete
body excerpts. They keep `match chain` free of `when` rows and put the subprocess
fixture on `match call`, as the parser requires
([matches.go](../../../../compiler/internal/syntax/matches.go#L47)). The revised
browser marker, exact Can-owned console warning policy, workspace path/lease
distinction and required-check inventory address the earlier review findings.

## Assertion and scenario syntax checks

- `sample: call spec::offline_context("case-id") => ok` is syntactically
  possible. Assertion inputs are ordinary invocation arguments
  ([parser](../../../../compiler/internal/syntax/declarations.go#L449)); a
  current fixture uses `call bytes::empty()` in that position
  ([source](../../../../compiler/testdata/current/fetch/main.can#L56)). The
  proposed `offline_context` must return `spec::context` with `emits {}` and
  satisfy its **own** mandatory attached assertions. Every concrete function
  needs an `asserts` block and at least one row
  ([parser](../../../../compiler/internal/syntax/declarations.go#L382),
  [checker](../../../../compiler/internal/check/assertions.go#L499)).
- `spec::failed{"exit-code"}` is valid error-payload syntax if `spec::failed`
  is declared with one `str` field and admitted in the project's error
  registry. Existing attached rows construct an analogous
  `checks::failed{"positive required"}`
  ([source](../../../../examples/gallery/src/18-relay.can#L12)). The
  `wrong_code` row in the complete subprocess example therefore has a
  legitimate expected completion. It must be reached through the real
  `expect::int` call after the supplied result, as the draft states.
- A root row can link an exported scenario in an imported helper package, for
  example `link compiler_io::scripted`, while that helper's `when` row spells
  `scenario scripted:` locally. Qualified selectors are **not** written in
  `when`; the parser expects a single name after `scenario`
  ([parser](../../../../compiler/internal/syntax/declarations.go#L417),
  [cross-package test](../../../../compiler/internal/check/scenario_test.go#L38)).
  Multiple exported scenario links are supported in one root
  ([test](../../../../compiler/internal/check/scenario_test.go#L185)).

## Remaining construction details

1. **Give `offline_context` a meaningful row without relying on unproven
   nested callable equality.** `spec::context` is now a constructible record
   containing a reporter callable. An attached assertion expecting a *separately
   constructed* context record may compare two distinct callable wrappers.
   `assertionEqual` handles direct callables and array elements by receipt, but
   falls through to `Bun.deepEquals` for records
   ([runner](../../../../runtime/assert/runner.ts#L88)); a `callable` expression
   emits a fresh `ownCallable` wrapper
   ([emitter](../../../../compiler/internal/emit/callables.go#L99),
   [runtime](../../../../runtime/callable.ts#L312)). The repository tests
   direct callable arguments, not this nested record assertion
   ([test](../../../../runtime/test/assertions.test.ts#L122)). A safe design is
   to reuse one package-level `offline_reporter` callable value in both actual
   and expected contexts, or put a directly constructed `spec::context` in the
   case row and omit the `offline_context` helper. If the helper remains,
   demonstrate its own nontrivial assertion and record equality in the first
   bounded vertical slice.
2. **Pin the live reporter's capture type.** A reporter may close over a
   constructible immutable channel identifier, with the actual write authority
   checked in the worker's owned native scope. It should not capture an
   unconstructible opaque channel handle and assume its `near` input vanishes
   from mandatory assertions: ordinary opaque inputs are not elided
   ([checker](../../../../compiler/internal/check/assertions.go#L89),
   [refusal test](../../../../compiler/internal/check/browser_elision_test.go#L64)).
   Its callable effect set must match the `spec::context` field exactly, and
   its own assertion should supply the generic transport boundary without
   writing a live event. The offline reporter must not silently turn a failed
   comparison into `ok`; it may avoid persistent I/O while the helper still
   emits `spec::failed` after the reporting call.
3. **Keep case-specific supplied traces in Can wrapper packages.** The draft
   now says generic native APIs have no case fixtures, which is the right
   boundary. The four body-only examples still depend on authoring their
   case-owned wrapper functions and individual `match call` fixture tables.
   Those wrappers should supply only raw boundary facts, leaving source
   mutation, parsing, comparison, order and final verdict in ordinary Can.
   This is a pending library design/evidence task, not an existing API or a
   source-syntax contradiction. The complete subprocess example gives the
   concrete pattern; its fixture row reuses the same `c` and `options`
   invocation values, so argument matching does not need to compare a fresh
   context record with its nested reporter.

The current draft's Can-owned `required_checks` plan closes the early-`ok`
loophole: the reducer requires one final event for every planned check and
rejects missing, duplicate, unexpected and wrong-scope evidence. Its browser
case now preserves the host scenario's exact warning policy
([source](../../../../tests/integration/browser/controls.mjs#L292)). The
compiler rejection uses a real `CAN-CHECK-CAPTURE` mutation
([checker](../../../../compiler/internal/check/callables.go#L147)); the proposed
structured CLI is correctly marked as absent. The native JSON example keeps
source tokens, signed zero and raw serialization distinct from a production
Can codec round trip. I found no further source-backed semantic contradiction
in those example oracles.
