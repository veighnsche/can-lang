# Design revisions

> Design completion: [September 22 behavior contracts](language-behavior-contracts-2026-09-22.md) resolves the normalization, wrapper, match and testing details discussed here. The original discussion record below is preserved.

> Scope follow-up: [finding dispositions](language-design-dispositions-2026-09-22.md) covers every requirement and additional candidate below. This document remains the original requirements record; final grammar and implementation tasks are separate work.

Status: historical requirements captured on 2026-09-21. The then-open questions below are preserved as discussion history; current selections live in [decisions.md](../syntax-taste/decisions.md) and its incorporated specifications. This document does not change the implementation plan or authorize implementation changes.

## Problem

The sequential fetch example in `compiler/testdata/current/fetch/main.can`, in `fn void main`, nests eight calls inside successive successful `match` branches. Each level repeats these seven standard errors:

- `http::invalid_request`
- `http::credentials_missing`
- `http::transport_failed`
- `http::timeout`
- `http::body_limit`
- `http::status_error`
- `codec::invalid_data`

The same list also appears in fetch declarations and the calling function's `emits` declaration. Repeating the same seven errors throughout a program fills the page with infrastructure declarations and error arms, obscuring the actual operations.

## Confirmed requirement

Normalize the standard fetch errors so that a Can author does not have to manually enumerate or handle all seven for ordinary fetch use.

In particular, authors must not have to repeat the same seven entries in `emits` declarations for every fetch and calling function. This repeated standard error list is the focus of this revision.

The language should own this standard normalization. Merely asking every application author to write a wrapper around the seven errors does not meet the stated requirement.

## Scope

Nesting is not currently considered a problem to solve in this revision. The user explicitly narrowed the discussion to normalization of the seven standard errors. Changes to sequential composition, success-branch indentation, or new control-flow syntax are out of scope for now.

## Questions open at the time of this record

- What normalized error contract does a fetch expose: one error with structured details, a named error family, or another representation?
- Which details remain available for diagnosis and selective recovery?
- How does normalization affect `emits` declarations and propagation through calling functions?
- How do assertion fixtures express normalized failures?

## Correction: wrap the judge, not individual Jev questions

Confirmed design correction from discussion on 2026-09-21: for the TypeSafe Jev question forms, `noul`, `choice`, and `score` should not be wrapped individually. Among those question forms and `judge`, only `judge` should be wrapped. The earlier idea that each individual question could be wrapped was a design mistake and is superseded by this correction. Fetch may also be wrapped.

The fetch and judge wrappers must also deal with the seven standard transport/codec errors listed above. They are the normalization boundary: ordinary authors must not have to enumerate or handle those seven errors again for every wrapped operation or calling function. This must be language-owned behavior, not boilerplate wrappers each application author has to write. Individual `noul`, `choice`, and `score` questions inside a judge must not each require their own wrapper for these errors.

Wrappers must support derivation: an author can make a new wrapper based on another wrapper, inheriting its error-handling behavior and overriding it. A derived wrapper only needs to declare its additions or overrides; it must not have to repeat everything in the base wrapper. Handling omitted from the derived wrapper remains inherited from the base. When inherited and overriding handling conflict, the last wrapper's handling wins. The proposal must make this precedence explicit; the syntax and any rules for combining multiple bases remain undecided.

The precise wrapper mechanism, syntax, and normalized error contract still need a proposal, including which failure details remain available for selective recovery. This note does not equate wrapping with the ordinary function wrappers used for named callable references, nor does it decide how application-specific or additional AI errors are handled. Review the existing design and implementation against this boundary when preparing the fix.

## Match arm order: errors first, success last

Confirmed requirement: in every match that handles success and errors, place all error arms before the successful `ok` arm. The `ok` arm must appear last, rather than first. This applies throughout the language's success/error matches, not only to fetch calls.

Examples and test fixtures should present failures first and the successful continuation last. The relative order of individual error arms has not been specified.

Whether the compiler enforces this source order or it is an authoring convention remains to be decided. This requirement concerns arm order; it does not request changes to nesting or failure semantics. Implementation and fixture edits are not part of this draft-only update.

## Review example

Use the existing eight-call fetch fixture to evaluate the eventual proposal. Compare repeated `emits` entries, standard error arms, and the ability to recover from a particular failure. A finished proposal must explain the normalized public error contract and failure propagation; reducing nesting is not an acceptance criterion.

This draft records the concern and requirement only; it does not claim that a particular replacement design has been approved.

## Additional design review candidates — 2026-09-21

These observations come from a read-only review of current Can fixtures for AI questions, coordination, callables, assertions, codecs and bytes. They are candidates for the user's later design review, not approved language changes or confirmed compiler defects. The active implementer and reviewer were not interrupted. Source line numbers below describe the inspected working tree and may move.

### Error enumeration grows across abstraction boundaries

Evidence: `compiler/testdata/current/native/questions.can:115`, `judge float assess`, exposes ten errors: the seven transport/codec errors plus two AI errors and an output error. Its caller repeats the list and the individual match arms. `native/noul.can` shows the same pattern.

Impact: a caller's declaration and handling burden grows with the implementation layers beneath an operation. The fetch normalization requirement has relevance beyond fetch declarations themselves.

Review question: how should standard error normalization compose through AI and user-defined operations while preserving meaningful selective recovery? This does not automatically authorize collapsing application errors.

### Combining failure aggregates requires conversion scaffolding

Evidence: `compiler/testdata/current/coordination/aggregate-composition.can:18` and `:46` define two recursive array conversions, `widen_a` and `widen_b`; `:32` and `:60` add wrappers rebuilding `all_failed<combined_failure>`. The three named variants in this fixture contain the same two leaves.

Impact: composition of two operations is surrounded by type-conversion code unrelated to the successful work. The current contract deliberately requires explicit normalization of distinct aggregate specializations (`docs/implementation/coordination.md`). This is an ergonomic cost of that design, not evidence of an implementation violation.

Review question: can common aggregate composition be expressed more directly without weakening nominal type guarantees? The fixture demonstrates one verbose route; it does not establish that recursion is the shortest legal route, since explicit mapping is available.

### Callable array types visually merge error bounds and array suffixes

Evidence: `compiler/testdata/current/callables/captures.can:80` spells an array of callables as `callable int (int) emits [][] actions`. The first `[]` is the empty error bound; the second makes the completed callable type an array. `assertions/queues.can` repeats this form.

Impact: readers must understand parser precedence to distinguish array-valued results, error bounds and collections of callables. `docs/syntax-taste/technical-spec.md` explicitly defines this spelling, so this is a readability concern about the selected syntax.

Review question: is that visual ambiguity acceptable in ordinary code, or should the eventual design offer a clearer way to name or write such types? No replacement syntax selected.

### Captured arguments are invisible at callable creation

Evidence: `compiler/testdata/current/callables/captures.can:41` and `:50` create `callable combine`. Its declaration captures `prefix` and `suffix` through `near`, by exact local binding name. Neither captured value appears at the creation expression.

Impact: understanding dependencies requires navigating to the target declaration; local names participate in binding semantics. This is deliberate behavior documented in `docs/implementation/callables.md`, with a tradeoff between concise references and visible dependencies.

Review question: does source tooling make these captures sufficiently visible, or is a language-level adjustment desirable? Do not treat implicit capture alone as proof the design is wrong.

### Runtime test checks signal failure through division by zero

Evidence: `compiler/testdata/current/native/questions.can:155`, `codec/roundtrip.can:56`, `callables/captures.can:91` and `coordination/aggregate-composition.can:99` use `int invalid = 1 / 0` when a checked condition fails.

Impact: an arithmetic failure communicates less intent than a failed check with an expected value or explanation. Repeated use deserves investigation into test-author ergonomics and diagnostics.

Review question: is this only a fixture convention, or does it reveal missing convenient runtime-check functionality? Existing `asserts` sections already cover input/output examples; these observations do not establish that the language lacks assertions in general.

### Interpretation limits

- The deeply composed generic calls and grouped values in current fixtures are deliberate regression probes. Their complexity alone is not evidence that ordinary programs require those expressions.
- `codec/roundtrip.can:13` and `native/questions.can:137` already use `match chain` to express sequential fallible calls with one set of outcome arms. The earlier fetch pyramid must not be taken as proof that the language always requires nesting. Checking its suitability for that exact fetch fixture was outside this review; nesting changes remain deferred.
- No compiler, runtime, test fixture, task status or approved specification was changed by this review.
