# Grouped errors and test labels: implementation handoff

Status: implementation stopped at the user's request on 30 September 2026. This
plan is the handoff deliverable. Partial edits already exist in the working tree;
they are drafts, not a completed or validated implementation. The next agent
should review and finish them rather than assume either that nothing was started
or that the feature is ready to ship.

Workspace: `/Users/vince/Projects/can-lang`. The implementation started from
`5804b686`, with only the preceding Manolea review/evidence untracked. No branch,
managed worktree, commit or PR was created. No Manolea application files were
changed. The partial code is deliberately preserved in this checkout for the
next agent; do not reset unrelated changes that may arrive after this handoff.

## Selected scope

Implement these source forms:

```can
error_a | error_b => handler
error_a | error_b
absent | configured: arguments => expected
```

The first two apply to named domain errors in completion matches; the third
applies to `asserts` and lexical `when` rows. Use the existing `match chain` and
`relay call` for sequencing and unchanged propagation. No additional error
control-flow construct, error-union mechanism, live test runner, authentication
or CRUD primitive, HTML builder, option shorthand or indentation migration is
part of this handoff. No bulk Manolea rewrite or performance campaign is included.

Supporting investigation: [Manolea review](manolea-experience-review-2026-09-30.md),
[initial grouping consultations](preparation/jev-manolea-experience-2026-09-30/findings.md),
and [payload-binding consultations](preparation/jev-grouped-completions-2026-09-30/findings.md).
The last three requests favored unbound groups (.75/.73/.72), with moderate
confidence. Source contracts and acceptance tests remain authoritative.

## Proposed semantics

1. Group members are unaliased named domain-error heads, optionally exactly
   specialized. `ok`, `[_]`, typed binders and `as` aliases cannot join a group.
   Individual error arms keep their existing implicit/explicit payload bindings.
   Group bodies introduce no error-specific payload aliases.
2. Resolve every member against the exact input bound. Preserve missing-arm,
   duplicate/overlapping-head, impossible-head, ambiguous-generic and
   success-last diagnostics. Each forwarded error must fit the enclosing bound.
3. Bare grouping forwards the original protected completion. It must not
   reconstruct an error, replace its occurrence identity or alter its payload.
4. A grouped handler has one lexical body. Check it once, retaining a single set
   of nested call/fixture sites. Lower to existing exact-error IR arms referencing
   that checked body. No new runtime helper or completion representation is needed.
5. Assertion labels expand into independently named roots; fixture labels expand
   into independent selectors in source order. Arguments, expected completion,
   links and execution mode are retained and evaluated in each appropriate root.
   Never combine roots, report entries, contexts or fixture queues.
6. Duplicate names in assertion roots remain invalid. Repeated selectors on
   separate `when` rows remain valid FIFO fixtures. Duplicate names inside one
   group should be rejected rather than silently create repeated rows.
7. `scenario a | b:` applies the scenario prefix to every label. Each resolves
   independently. A grouped `when` template use expands independently per
   selector. Templates remain invalid on assertion roots. Existing restrictions
   on modes, links, receivers and native assertions are unchanged.
8. Coordination completion handlers should use the same grouping rules wherever
   their existing contracts permit the members. Preserve special first-success
   race/`all_failed` rules. Wrapper policy keys remain individual exact heads.

## Existing draft and known gaps

| Area | Draft files | State at handoff |
| --- | --- | --- |
| Syntax | `compiler/internal/syntax/{ast.go,matches.go,coordination.go,declarations.go,format.go,grouped_rows_test.go}` | AST, parsing, formatting, label expansion and tests drafted; package tests have not run. |
| Checking | `compiler/internal/check/{completion_matches.go,assertions.go,grouped_syntax_test.go}` | Shared body and row expansion drafted; incomplete edge-case audit and test coverage. |
| Emission tests | `compiler/internal/emit/grouped_syntax_test.go` | Static IR/emission checks and bounded Bun assertion-root tests drafted, never run. No production emitter change made. |
| Navigation | `compiler/internal/driver/{diagnostics.go,grouped_syntax_test.go}` | Lookup of every grouped error head, including coordination, and definition-jump tests drafted, never run. |
| Tokenizer | `editors/vscode/syntaxes/can.tmGrammar.json`, `editors/vscode/test/grammar.test.js` | All 11 grammar tests passed, including the new grouped-label case. |
| Documentation | `docs/implementation/{assertions.md,completions.md}`, `docs/syntax-taste/{decisions.md,technical-spec.md}` | Updated toward intended behavior, explicitly marked pending. Do not treat them as proof of shipped support. |

The source AST retains the first `MatchArm.Outcome` and adds
`AlternateOutcomes []OutcomePattern`; one body/forward flag belongs to the group.
`Assertion.Name` remains the first label and `AlternateNames []Token` contains
only subsequent names. `syntax.ExpandAssertions` copies a source row per label,
clears its alternatives, updates `Name` and any scenario token, and preserves the
shared source row span and payload. The formatter sees the grouped source tree;
semantic entry points consume expanded rows.

Two high-priority audits are still open:

- `compiler/internal/check/program.go` has a pointer-specific
  `*syntax.OutcomePattern` exception for gathering bare generic error heads.
  The new alternatives are value elements in a slice. Verify every head goes
  through equivalent generic handling without traversing the common body twice.
  This file was not changed before the stop.
- `compiler/internal/check/coordination.go` has special paths that examine only
  `arm.Outcome`, especially first-success race and `all_failed`. They must not
  silently ignore `AlternateOutcomes`. Validate every member or reject invalid
  groups explicitly. This file was not changed before the stop.

Also inspect recovery checkpoints and shared-IR-body consumers. Error collection
must not leave partial coverage state that creates misleading later diagnostics;
IR traversals or optimization passes must not corrupt shared bodies or create
distinct fixture sites merely because several exact dispatch arms reference one.

Project raw-fixture discovery and driver source-preservation fixes intentionally
traverse the grouped source row/body once: they inspect static paths or source
text, not individual execution identities. No expansion was added there.

## Implementation task list

The [parallel implementation task list](grouped-errors-and-labels-implementation-tasks-2026-09-30.md)
is the authoritative execution order and progress ledger. It replaces the former
G01–G08 outline with small assignments, exact dependencies, exclusive file owners,
acceptance checks and evidence requirements. No implementation task is accepted
complete; existing drafts remain starting material.

After a short interface/ownership handoff, syntax, error checking and assertion
checking can be authored concurrently. Reuse freed workers for execution tests
and editor support, with one coordinator and at most three workers. Validation
runs remain serialized and independent review is a separate gate. See the task
list for the detailed schedule, preparation test-file split and progress log.

For execution acceptance, a static assertion that the emitter does not call `.create()` is useful
but insufficient by itself to prove occurrence identity. Compare the returned
completion or its occurrence/payload at execution time. Similarly, pointer sharing
in IR is not enough without a test reaching the same nested fixture site through
different alternatives and observing the expected queue behavior.

## Validation and resource constraints

Only `bun run test:grammar` in `editors/vscode` completed: **11 passed**.
`git diff --check` was clean at the last review. The Go syntax and checker test
attempts both stopped before compilation because the sandbox denied access to
`/Users/vince/Library/Caches/go-build`. That is an environment failure, not a test
result. No Go package test, emitted Bun test, runtime check or distribution build
has validated these changes.

The next executor should obtain access to the existing shared Go cache through
the environment's supported permission mechanism. Do not work around it by
building a full private cold cache or retaining copied dependencies. Suggested
first commands, after fixing the known open checker paths:

```sh
GOMAXPROCS=2 go test -p=2 ./compiler/internal/syntax -count=1
GOMAXPROCS=2 go test -p=2 ./compiler/internal/check -run 'TestGrouped' -count=1
GOMAXPROCS=2 go test -p=2 ./compiler/internal/emit -run 'TestGrouped' -count=1
GOMAXPROCS=2 go test -p=2 ./compiler/internal/driver -run 'TestGroupedCompletionDefinitionJumps' -count=1
```

Then run appropriate nearby completion, generic, assertion, scenario, template,
format-equivalence and coordination regressions. Choose bounded package checks;
do not launch the entire distribution/browser/performance campaign by default.
Retain the existing native JavaScript/Bun lowering and add no runtime abstraction
unless a demonstrated contract requires it. If authored runtime TypeScript is
changed, apply the required runtime lint/format scripts; before completion follow
the repository's `bun run check:runtime` and relevant-test requirements.

Use immediately registered cleanup (`t.TempDir` in Go tests), one build per run,
runtime symlinks rather than copies, bounded process timeouts and no retained
bundles/private caches. Report cleanup failure. No temporary directory, private
GOCACHE or running build/session was left by this stopped round. Durable review
documents and small consultation JSON files are intentional evidence.

## Handoff instruction

Continue the selected grouped-error and grouped-test-label implementation in
`/Users/vince/Projects/can-lang` using the linked parallel task list and this
design. Review the existing partial
diff first. Do not treat draft documentation or unexecuted tests as evidence of
completion. Preserve unrelated work, finish the known generic/coordination and
execution-identity checks, run bounded verification, and record results here or
in a linked compact completion record. Keep scope to the selected features.
