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
remain distinct roots. Generic instance assertions require I46's specialization
pass, which remains pending.

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

The first fixture slice supports `when` on one resolved named invocation,
including a method with an implicit receiver. A matching root-name row replaces
only that call; preparation and the surrounding body still execute. Selected
rows advance in written FIFO order, never by argument search. Wrong arguments,
exhaustion and leftovers fail. Production `run` ignores these substitutions.
The [basic fixture](../../compiler/testdata/current/assertions/basic.can) demonstrates
large integer computation and a substituted call followed by real computation.

I18 still owns complete typed-AST invocation paths, captured-callable instances,
coordination reservation/barriers, and the full fixture-token/reporting admission.
Chain/native-expression fixture scheduling is not admitted by this first slice;
unsupported forms fail explicitly. I20 owns late work and resource drain.

The launcher awaits a versioned `can.assertion-report` JSON object on stdout.
Each root records its immutable identity, pass/fail result, safe reason, harness
violations and evidence labels. `real-can` and `supplied-completion` remain
separate. The report does not claim provider, raw transport, Bun capability or
live-quality evidence. Values, native messages, stacks and credentials are not
included. Status is 0 for a passing suite, 1 for failure and 2 for malformed CLI
usage. Static failures produce stderr diagnostics and do not replace current
output. Initialization failure prevents subject execution and fails the suite.
