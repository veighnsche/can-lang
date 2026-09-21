# Emitted assertions

```sh
/absolute/version/bin/canlc assert /absolute/canonical/project
/absolute/version/bin/canlc assert /absolute/canonical/project PACKAGE ASSERTION
/absolute/version/bin/canlc assert /absolute/canonical/project PACKAGE DECLARATION ASSERTION
```

Use the full package/declaration identities printed in the report, such as
`can.project.root/arithmetic` and `can.project.root/arithmetic::square`.
A short assertion name is accepted only when unique in the specified package.
The default runs all concrete assertion roots in the loaded graph. Libraries
need no `main`; an existing root `main` must still have the entry signature.

Every concrete function's mandatory assertion rows and call-site `when` rows
are checked before publication, including roots not selected for execution.
Inputs use ordinary invocation/receiver/variadic checking. Expected completions
must match the declared success type or a declared domain error specialization.
Duplicate names within a declaration fail; equal names in different declarations
remain distinct roots. Generic assertions are checked for each reachable concrete
specialization by the specialization pass.

An assertion has separate checked regions for its actual invocation and expected
completion. Both execute in emitted Bun code. The runner evaluates the expected
completion, invokes the subject once, and compares protected results. Native
strict `Bun.deepEquals` handles ordinary data; the harness first enforces opaque
and callable identity. Domain comparisons use the exact nominal specialization
and payload rather than freshly allocated occurrence IDs. Actual unhandled
standard failures fail the assertion.

Generated functions pass a compiler-private assertion context explicitly through
calls. Platform output adapters refuse live work when it is present. The refusal
is both a standard failure and a sticky harness violation: catching it in Can
cannot make the assertion pass. Independent roots retain separate contexts across
awaits. Pure native operations, including UTF-8 conversion and arithmetic, run
normally.

A `when` table on a checked single invocation replaces only that exact call;
preparation and the surrounding body still execute. This includes deterministic
native operations such as `"abc".slice(1, 3)`. Production `run` executes the real
operation. Chain fixtures must be placed inside a named wrapper.

Each substitutable call carries a declaration/preorder lexical site, a
parent-local occurrence, a reserved direct/spread participant path and, when
applicable, a callable creation receipt. Receivers and `near` captures freeze at
closure construction. Initialization-created receipts can appear under separate
roots without sharing queues; assertion-created receipts remain root-bound.

Selected rows form one FIFO per root and lexical table, shared by recursive,
transitive and concurrent visits. The harness reserves the complete coordination
batch before launch. Explicit invocation frames keep fixtures pending until every
active continuation reaches a harness await or finishes; the least full path is
then released. Native Promise operations still select coordination outcomes. A
selection continuation remains active across native reactions, and roots drain
owned losers before checking unused rows. Fixed microtask delays and argument
searching do not allocate rows.

The [basic fixture](../../compiler/testdata/current/assertions/basic.can),
[native slice](../../compiler/testdata/current/assertions/native-slice.can) and
[shared queues](../../compiler/testdata/current/assertions/queues.can) demonstrate
real computation, exact substitution, repeated arguments, recursion, mutual
recursion, callable captures, spreads and all four coordination forms.

The launcher awaits a versioned `can.assertion-report` JSON object on stdout.
Each root records its immutable identity, pass/fail result, safe reason, harness
violations, evidence labels and full fixture invocation paths. Queue errors also
include the expected table/row/path and actual path. Capture values, native
messages, stacks and credentials are not included.

Evidence distinguishes `real-can`, `supplied-completion`, `raw-provider-fixture`,
`bun-conformance` and `live-quality`. Supplying a completion does not claim raw
provider coverage. The latter two categories require an explicit matching
conformance or external-quality job and are never promoted from ordinary
assertion results. Release evidence summaries preserve all five categories,
including empty ones. Status is 0 for a passing suite, 1 for failure and 2 for malformed CLI
usage. Static failures produce stderr diagnostics and do not replace current
output. Initialization failure prevents subject execution and fails the suite.
