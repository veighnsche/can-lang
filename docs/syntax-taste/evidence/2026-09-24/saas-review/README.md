# SaaS language review evidence

Parent review: [When I would recommend Can for full-stack SaaS](../../../saas-language-review-2026-09-24.md).

Source reviewed: `4c2db1e`, 24 September 2026. The source checkout was clean before adding this report and evidence. No implementation files were changed.

## Coverage

[`inventory.json`](inventory.json) lists every reviewed example source: 56 `.can` files across 16 projects, 3,146 source lines. It records 154 lines longer than 120 characters; the longest is 336 characters. Counts include comments and blank lines. These are layout observations, not an authoring-error study.

Independent review coverage was divided into frontend (account-search, form-validation, dashboard, language-site), backend (cookies, crypto, files, markdown, mysql, native-ai, process, sqlite, stream, utilities, websocket), and core language (gallery plus checker/type/specification evidence). The coordinator reconciled findings, inspected cited implementation paths, and checked current tests. Historical missing-feature claims were not accepted without checking the implementation.

## Jev method

The TypeSafe skill and current official documentation were read:

- [Documentation index](https://docs.typesafe.ai/llms.txt)
- [HTTP API](https://docs.typesafe.ai/api.md)
- [Choice questions](https://docs.typesafe.ai/primitives/choice.md)
- [State](https://docs.typesafe.ai/concepts/state.md)

The web tool retrieved the index but could not open the three Markdown detail pages; direct HTTPS retrieval of those official pages succeeded. Requests used the documented `POST https://api.typesafe.ai/v1/systemone` endpoint and an existing environment credential. No credential is present in this evidence.

Three requests were authored before dispatch. Every explanatory state field, question instruction and option description was freshly rewritten; technical identifiers and option keys retain their intended identity. A field-by-field semantic review checked that facts, constraints, caveats and alternatives were unchanged. A mechanical check confirmed no corresponding prose field was identical across requests. [`wording-audit.json`](wording-audit.json) records that audit. No response was included in any later request.

Each request asked the same six independent Choice questions. Jev received the relevant evidence and limitations; it was not asked to research, generate design prose or inspect the repository.

| Files | HTTP status | Returned model |
| --- | --- | --- |
| [request 1](request-1.json), [response 1](response-1.json), [metadata](response-1.metadata.json) | 200 | jev-1.13.0 |
| [request 2](request-2.json), [response 2](response-2.json), [metadata](response-2.metadata.json) | 200 | jev-1.13.0 |
| [request 3](request-3.json), [response 3](response-3.json), [metadata](response-3.metadata.json) | 200 | jev-1.13.0 |

## Results and reconciliation

Values below are the probability assigned to the selected option, not guarantees that a proposal is correct.

| Question | Selection in all three requests | P1 | P2 | P3 |
| --- | --- | --- | --- | --- |
| Browser direction | Prototype a typed Can browser model while retaining server rendering | 1.00 | 0.78 | 0.76 |
| Web linkage | Investigate linked typed endpoint/form/component contracts | 0.83 | 0.95 | 0.98 |
| Domain values | Demonstrate the current limitation and prototype owner-restricted construction | 0.98 | 0.81 | 0.99 |
| Workflow composition | Measure the best current chain/helper solution before selecting extensions | 1.00 | 1.00 | 1.00 |
| SDK boundary | Reproduce an integration need before considering fully contracted adapters | 1.00 | 1.00 | 1.00 |
| Assertions | Retain attached assertions and strengthen behavioral observations/scenarios | 1.00 | 1.00 | 1.00 |

There were no disagreements in the selected alternatives, so no conflicting selection required an additional investigation. There was material probability sensitivity: the browser alternative received 0.76–1.00 and protected domain construction 0.81–0.99. Request 2 assigned 0.15 to special built-in business types; request 3 assigned 0.15 to retaining HTMX alone. The report therefore leaves the precise browser architecture and construction mechanism as design experiments, not settled implementations.

Agreement does not establish independent empirical validation or remove framing bias. In particular, the supplied rich-client objective favors investigating browser execution; a narrower product goal could justify HTMX alone. Likewise, examples involving two domain types cannot establish the complete opacity/codec/testing design. The report preserves the prior demonstrated-need requirements for error-set parameters, sequencing and native adapters.

Total reported usage across the three calls: 5,623 input tokens and 1,041 output tokens.

## Validation and limits

- `bun run check:runtime`: passed (lint, formatting and TypeScript checking).
- `GOCACHE=/tmp/can-saas-review-go-cache go test ./compiler/internal/syntax ./compiler/internal/check ./compiler/internal/types`: passed. Reported package times were 0.338s, 107.039s and 0.386s. The initial command using the default Go cache could not write under the sandbox; the temporary cache resolved that environmental issue.
- `bun test ./runtime/test/html.test.ts ./runtime/test/form.test.ts ./runtime/test/number.test.ts ./runtime/test/datetime.test.ts ./runtime/test/transaction.test.ts`: 40 passed, 0 failed, 896 expectations. See [`runtime-tests.txt`](runtime-tests.txt).

An earlier non-exact Bun filter also ran matching tests from an existing staged distribution. The saved final run uses exact current-source paths and excludes those copies. No new source tests were added for this documentation-only review.

This is not full release qualification or an assertion that all examples were executed. The review did not rerun live provider/database tests, every example build, Linux deployment or the browser integration suite. The 503 policy interaction and stream cleanup discrepancy are findings from current source inspection; their end-to-end failure scenarios remain useful follow-up tests. No performance results or measured authoring-error improvements are claimed.
