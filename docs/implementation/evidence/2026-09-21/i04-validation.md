# I04 — current core parser

Report version: 1. Date: 2026-09-21. Target: the I01 macOS arm64 development
layout, built from the pinned local archive `/tmp/can-i01-bun.zip`.

## Implemented boundary

`compiler/internal/syntax` now owns an independent recursive-descent/precedence
parser and explicit AST. `canlc parse [--render] FILE.can` reads this syntax
without entering the predecessor checker, evaluator, emitter, or provider paths.
The old grammar file is renamed `compiler/legacy_parse.go` and explicitly marked
as a predecessor implementation pending the scheduled checker/driver/editor and
I44 retirement tasks. There is no current-to-legacy AST adapter or grammar fallback.

Core coverage includes:

- Ordered package headers, aliases, generic parameters, records, numeric-ID
  errors, variants, top-level values, return-first functions, receivers, near and
  variadic inputs, mandatory emits/asserts, and named positional assertion rows.
- Nominal/array/callable/choice-arm types, exact error-bound ownership, nested
  generic right-angle splitting with original spans, and inert fragment entry points.
- Native-operator precedence, right-associative power, unary operators, comparison
  chains, calls/references/constructors, field/index/slice access, array spread,
  and single/multiple-field copy updates with the selected parentheses rule.
- Ordinary value/completion matches, multiple scrutinees, literal/range/array/
  constructor/alternative patterns, call matching, when rows, forwarding arms,
  chain bindings, terminal relay, and do with two or more steps.
- All four coordination headers, direct and spread participants, typed result
  bindings, and the distinct shared/per-participant arm placement. Unbound
  coordination remains a step and requires an enclosing terminal completion.

The parser retains original source and comment trivia. Canonical rendering omits
comments but preserves authored expression grouping. Tests compare AST node kinds
and payloads, excluding source spans/trivia, after render/reparse; stable text alone
is not the round-trip oracle. Empty/multiple native state groups remain argument-only
syntax, while single-value groups preserve their GroupExpr spelling. No tuple type
or value is created. Declaration resolution and I16 determine their admissibility.

## Acceptance evidence

- Both complete Can programs in C10/C11 parse and re-render directly from the
  authoritative technical specification. All 13 complete core function examples
  from the selected-decision document do the same. Additional fixtures exercise
  declarations, patterns, coordination, multiline literals, and nested callable types.
- Negative cases cover obsolete rev/extern/decimal-literal spellings, missing or
  out-of-order sections, trailing commas, multiline calls/assertions, one-step do,
  statements after completion, ordinary call/chain result binding, completion
  wildcards/payload destructuring, and incorrect coordination arm placement.
- Unicode, BOM and CRLF inputs preserve original-byte diagnostics and distinct
  scalar/UTF-16 coordinates. A 100-level parenthesized-call regression prevents
  the speculative argument reparsing identified during review. Recursive syntax
  has a 256-level diagnostic limit; generic speculation holds token fragments
  locally instead of mutating lexer tokens or cloning the full stream per name.
- [Targeted parser/command tests](i04-parser-tests.txt) and the
  [full Go suite](i04-go-tests.txt) pass.
- [File-parser fuzzing](i04-file-fuzz.txt) and
  [expression-parser fuzzing](i04-expression-fuzz.txt) exercise termination,
  valid diagnostic spans, and file rendering/reparse stability. Earlier type
  fuzzing completed 138,366 executions without a failure.
- [Offline staged integration](i04-offline-integration.txt) invokes the absolute
  built compiler through a symlink, from unrelated cwd, with PATH unavailable,
  hostile Bun configuration, and all network access denied. The fixture retains
  an assertion containing division by zero and an unresolved HTTP invocation;
  both remain inert. Rendering is stable, old syntax fails, and source/bundle
  hashes remain unchanged. The existing seven sidecar refusal cases also pass.

One intermediate integration run detected a source-tree change because the agent
edited the syntax README while its source-hash guard was active. The exact output
is retained in [edit-race evidence](i04-offline-integration-edit-race.txt). The final
run is performed after source edits stop; this was a test orchestration error,
not ignored evidence of a compiler write.

The three rewritten [Jev consultations](i04-jev/README.md) agreed on an independent
current parser and inert command. Their responses are advice; the checks above
establish the implemented boundary.

## Limits belonging to subsequent tasks

Parsing does not claim executable current-language support. Name/import resolution,
type checking, completion exhaustiveness, error identity, range ordering, and
effect/coordination contracts belong to I05 onward. Native AI/fetch declaration
sections and handler-specific `%` remain I16. The historical default compiler and
editor paths are not reinterpreted as current-language support and remain scheduled
for replacement/retirement. This task neither evaluates source in Go nor invents
fallback runtime behavior.
