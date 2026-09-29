# Error bounds and constructors with braces

Date: 2026-09-29. Design review only; no language implementation was requested.

## Exact proposal assessed

- Change a finite error bound from emits [E] to emits {E}, including the empty form emits {} and callable types.
- Change every nominal error value constructor from E(value) to E{value}. The user confirmed positional fields. Error declarations, records, calls, matching, forwarding, and wrapper emits calculated retain their roles.
- Preserve the distinction between constructing a typed error value and emitting it as a terminal completion. Standard failures remain nonconstructible and outside emits.

## Source evidence

- The parser centralizes finite emits parsing in compiler/internal/syntax/parser.go; formatting also spells bounds in compiler/internal/syntax/format.go.
- All nominal constructors currently share one syntax node and punctuation in compiler/internal/syntax/expressions.go and format.go. A brace-versus-parenthesis marker would have to survive parsing and be checked against the resolved error/record kind.
- Existing source uses error values as data, including nested generic aggregates in compiler/testdata/current/coordination/aggregate-composition.can. Therefore E{value} cannot itself mean throw or emit.
- docs/syntax-taste/technical-spec.md C2 notes that emits [][] can denote an array of pure callables. emits {}[] would visually separate the error bound from the array suffix.
- tests/failure-conventions/retry/src/profiles/profiles.can shows a bare matched error forwarded with its identity, versus a fresh constructor rebuilding a value.

## Jev consultations

Three separate requests with rewritten explanatory prose and equivalent facts/options are saved as request-1.json through request-3.json. wording-audit.json records the pre-send comparison. Full responses and call metadata are saved beside them.

| Request | Selected | Both | Emits only | Constructor only | Neither | Confidence |
| --- | --- | ---: | ---: | ---: | ---: | ---: |
| 1 | emits only | 0.15 | 0.56 | 0.10 | 0.19 | 0.41 |
| 2 | both | 0.50 | 0.40 | 0.08 | 0.02 | 0.34 |
| 3 | constructor only | 0.02 | 0.29 | 0.41 | 0.28 | 0.22 |

The selected option changes across all three calls, and confidence is low. I compared the complete requests and found no altered facts or alternatives. Their different emphasis may affect the probabilities: request 1 explicitly says to treat changes independently, request 2 emphasizes nested examples, and request 3 asks what readers infer from positional values inside braces. The classifier supplies no rationale, so these are hypotheses, not causal findings. The disagreement is strong evidence against treating a Jev selection as a decision.

## Assessment

The two substitutions are grammatically feasible and more coherent than putting braces around every error-related form. emits {E} has a clear set role and can separate nested callable bounds from array suffixes. E{value} can be reserved for error values after nominal resolution, but positional arguments in braces may look like object fields while still being positional. Nested forms such as all_failed<T>{[invalid_data{"a", "type"}]} also deserve a readability probe.

The semantic rule must be explicit: E{value} constructs a value in terminal completions, successful data, arrays, variants, top-level values, assertions, and native fixtures; it does not implicitly throw. Forwarding an existing bound error remains different from reconstructing it. A keyed-field version of E{...} would be a larger proposal than a delimiter swap.
