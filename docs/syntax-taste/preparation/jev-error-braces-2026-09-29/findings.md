# Error-only curly braces: design review

Date: 2026-09-29. This is a syntax assessment, not an implementation decision.

## Current evidence

- Bare braces are rejected by the lexer; indentation delimits blocks, parentheses handle construction, and square brackets handle finite emits bounds. See compiler/internal/syntax/lexer.go, docs/syntax-taste/technical-spec.md C2, and tests/failure-conventions/retry/src/profiles/profiles.can.
- Domain errors are nominal typed completion values. Declaration, bound, construction, match arm, forwarding, wrapper handler, and assertion each have different grammar roles. Standard failures are outside every domain emits set (technical spec C9).
- There are no external users or compatibility requirements. No concrete brace grammar or authoring study was supplied.

## Three fresh Jev consultations

The checked requests are request-1.json through request-3.json; full responses and call metadata are beside them. wording-audit.json records equivalent facts/options and distinct wording for every explanatory field before calls were made.

| Request | Selected option | Probability: retain / emits only / universal | Confidence |
| --- | --- | --- | --- |
| 1 | retain current syntax | 0.84 / 0.07 / 0.09 | 0.76 |
| 2 | retain current syntax | 0.48 / 0.39 / 0.13 | 0.22 |
| 3 | retain current syntax | 0.90 / 0.09 / 0.01 | 0.85 |

All three selected the brace-free grammar, but request 2 assigned much more weight to emits-only braces. I compared the full request wording: request 2 calls brackets "bounded error rows" and frames the narrow option as a dedicated enumeration; the other requests describe the same facts and alternative in different language. This is a possible wording effect, not an established cause. Jev returns a classification and probabilities without design reasoning, so the agreement is advice rather than proof.

## Assessment

Do not adopt a blanket "all error-related source uses braces" rule on this evidence. It would assign one delimiter to type membership, field declaration, value construction, branching, wrapper handling, standard catches, and test expectations, while those operations have different meanings. The current parentheses constructor convention keeps record and error values parallel; the emits keyword already distinguishes its bracketed finite bound.

The most coherent isolated brace experiment would be emits {unavailable, forbidden}. It should be judged against a concrete authoring complaint and full examples before changing the grammar. The review does not claim that a precise universal-brace design is impossible.
