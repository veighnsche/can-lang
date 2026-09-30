# Backend follow-up consultations

Date: 2026-09-30. Research only. User constraints: eliminate host test harnesses,
permit a non-TypeScript test executor, support arbitrary normal Can functions.

The current [TypeSafe API](https://docs.typesafe.ai/api) and
[choice primitive](https://docs.typesafe.ai/primitives/choice) documentation were
read during this investigation. Requests used `jev-latest`; responses identify
`jev-1.13.0`. Credentials came from the environment and were neither saved nor
printed. No implementation or performance runs accompanied these consultations.

| Request | Preferred architecture | Probability | Confidence | Evidence |
| --- | --- | ---: | ---: | --- |
| 1 | `supervised_bun` | 1.00 | 1.00 | [request](request-1.json), [response](response-1.json), [transport metadata](response-1.metadata.json) |
| 2 | `supervised_bun` | 1.00 | 1.00 | [request](request-2.json), [response](response-2.json), [transport metadata](response-2.metadata.json) |
| 3 | `supervised_bun` | .93 | .90 | [request](request-3.json), [response](response-3.json), [transport metadata](response-3.metadata.json) |

All requests supplied the same facts, constraints and alternatives: supervised
reuse of the complete current backend, a complete checked-IR interpreter, a new
native codegen backend, or insufficient evidence to choose. Each of 13
explanatory fields was rewritten in full, including context, instructions and
option descriptions. The [pre-send audit](wording-audit.json) checks exact-string
differences and records manual semantic-equivalence review and request hashes.
Stable JSON identifiers and technical names were intentionally preserved.

There was no selected-option disagreement. The third response assigned .04 to
the interpreter and .03 to deferring selection. Follow-up source inspection
confirmed the reusable checked program, executable actual/expected assertion
regions, separate entry emission, existing worker supervision and missing
live-suite protocol. None of those facts demonstrates a performance advantage.

The recommendation favors lower additional semantic implementation work for this
migration. A separate full-interpreter product goal or future measured execution
constraints could change that judgment. Reworded agreement neither establishes
correctness nor guarantees removal of framing bias. Jev supplied classifications,
not explanations, source verification or implementation evidence.

A separate read-only review of the finished comparison found no material
correction. It checked unrestricted Can support, the native supervisor boundary,
production artifact coverage, interpreter scope and the absence of performance
claims against the cited source. [Documentation validation](validation.json)
confirmed 71 local links, matching request/response hashes and unchanged hashes
for all 224 inventoried `/tests` files. No tests, builds, benchmarks or temporary
execution workspaces were used.

Return to the [backend comparison](../../../native-can-test-backends-2026-09-30.md).
