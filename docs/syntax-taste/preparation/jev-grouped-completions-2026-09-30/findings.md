# Grouped completion payload binding

Implementation consultation, 30 September 2026. The user selected grouped named
error handlers, grouped bare forwarding and grouped assertion/fixture labels from
the [Manolea review](../../manolea-experience-review-2026-09-30.md). The initial
review already covered whether those features were worthwhile. This consultation
resolves the narrower payload-binding rule for the first implementation.

Three fresh requests present the same constraints and alternatives, with every
state paragraph, instruction and option description rewritten. The
[pre-send audit](wording-audit.json) records equivalence and request hashes. Full
requests, responses and validated call metadata are retained in this directory.
The API resolved `jev-latest` to `jev-1.13.0` for all calls.

| Request | Unbound groups | Common union alias | Check each binding context | Defer handlers | Reported confidence |
| --- | ---: | ---: | ---: | ---: | ---: |
| 1 | .75 | .04 | .21 | .00 | .66 |
| 2 | .73 | .05 | .21 | .01 | .64 |
| 3 | .72 | .14 | .04 | .10 | .62 |

All choose unbound groups, without overwhelming confidence. Alternatives retain
nonzero probability; there is no selection disagreement to resolve and no model
rationale to infer. Agreement is advice, not proof or removal of framing bias.

The implementation decision follows the source contracts and selected examples:
groups admit named, unaliased domain-error heads; they introduce no implicit
error payload aliases. Individual arms still bind and inspect exact payloads.
This avoids introducing a new error-union type system or checking one written
body in incompatible alternative-specific scopes. It also permits the common
body to be checked once, retaining one set of lexical call/fixture identities.
Existing exact-error IR branches share the checked body. Grouped bare arms
forward the original completion, preserving its occurrence and payload.

Duplicate coverage, exact generic selection, required coverage, success-last
ordering and escaping-error checks remain obligations. `ok`, `[_]`, explicit
aliases and wrapper policy keys remain individual. Grouped assertion labels
expand at checking boundaries into independent roots/fixture selectors; the
source AST retains one row for formatting and source traversal. Runtime and
compiler tests, rather than these probabilities, establish the implementation's
behavior.
