# Review of the three-form error brace proposal

Date: 2026-09-29. Design assessment only; no compiler change was requested.

## Proposal

Use error E{type field} for a declaration, emits {E} for a finite domain-error bound, and E{value} for positional construction of an error value. Keep record constructors and calls in parentheses, arrays in brackets, statement blocks indented, and match/relay/standard-failure syntax unchanged. The user explicitly chose positional values.

## Source findings

- Current C2 grammar spells these as error E(type field), emits [E], and E(value). The lexer rejects bare braces. The three proposed contexts have no inherent competing brace syntax.
- Error construction must continue to make a nominal value in every position. A terminal E{value} emits, but ok wrapper(E{value}) stores successful data. A bare matched error forwards its prior occurrence; E{value} reconstructs. Standard failures are outside emits, including emits {}, and remain nonconstructible.
- emits {}[] must retain current callable-array precedence, parallel to emits [][]. The special wrapper bound emits calculated is unaffected.
- The compiler currently shares one constructor AST for records and errors. Parsing must preserve whether source used braces or parentheses, and checking must reject a brace-built record and a parenthesized error. Generic constructor lookahead, terminal completion detection, formatting, LSP/editor syntax, fixtures, and the source corpus need coordinated migration. No compatibility layer is required.
- Readability is mixed. The three forms make errors visibly distinct, but E{value} uses positional data inside a delimiter many readers associate with named object fields. Nested aggregate values such as all_failed<T>{[missing{key}]} require a real reading test.

Primary evidence: docs/syntax-taste/technical-spec.md C2/C9; compiler/internal/syntax/{lexer.go,declarations.go,expressions.go,parser.go,format.go}; compiler/internal/check/expressions.go; tests/failure-conventions/retry/src/profiles/profiles.can; compiler/testdata/current/coordination/aggregate-composition.can.

## Three fresh Jev consultations

The full requests, responses, and call metadata are saved as request-1.json through request-3.json and matching response files. wording-audit.json records the same facts/options and distinct explanatory prose checked before sending.

| Request | Selected | Full triad | Bound + constructor | Declaration + constructor | Bound only | Current | Confidence |
| --- | --- | ---: | ---: | ---: | ---: | ---: | ---: |
| 1 | current | 0.34 | 0.12 | 0.02 | 0.07 | 0.45 | 0.33 |
| 2 | full triad | 0.79 | 0.09 | 0.03 | 0.01 | 0.08 | 0.75 |
| 3 | current | 0.07 | 0.06 | 0.26 | 0.03 | 0.58 | 0.47 |

The choice and distribution differ sharply. I compared every state field, question, and option: the factual checklist and alternatives match, while all explanatory fields were rewritten. Request 2 frames the proposal as a "coordinated punctuation revision" and explicitly asks about a mnemonic; that may contribute to its favorable result, but request 3 also asks about a helpful visual family and still favors current syntax. There is no dependable causal explanation from a classifier that supplies no rationale. Agreement or majority vote would not prove the design sound; here there is no stable consensus.

## Recommendation and completed source comparison

The initial assessment found the full triad feasible but left readability open. The subsequent side-by-side review in source-comparison.md covers a simple declared/raised error, empty error, nested generic error-as-data, emits {}[], forwarding versus reconstruction, a standard-failure catch, and a deeply nested retry assertion. Independent reviewers disagreed about the dense cases. After inspecting their exact rewrites, I judge braces helpful as error-value boundaries without changing the underlying nesting, and recommend adopting all three spellings as the proposed grammar.

No compiler implementation has been requested or performed. Parser/formatter round trips and semantic checks for the listed cases are implementation gates, not unresolved design research. Preserve the value-versus-emission contract, one-line delimiter rules, standard-failure separation, and emits calculated exception.
