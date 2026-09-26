# Evidence for the 26 September 2026 full language review

Revision: `2cb1bc35435ab5075d51f9f945adfb0e1f9dfe64`. Review: [full report](/Users/vince/Projects/can-lang/docs/syntax-taste/full-language-review-2cb1bc3-2026-09-26.md).

## Provenance and scope

Three fresh-context independent reviewers were given the current source and SaaS objective, with earlier review/consultation/recommendation verdicts excluded. Core and platform used Astra High for open-ended architectural judgment; backend used Sol High for bounded cross-file semantics. Selection followed the model-selection skill. The coordinator retained prior conversation context and reconciled the independent findings. No implementation changes, commits or releases were made.

The initial reports remain unchanged, including superseded first-pass claims. **Use reconciliations and the main report for the final assessment.** In particular, the platform reviewer corrected a stale-README inference about Linux, and the core reviewer narrowed what the result-data retry assertions establish.

- [Independent core report](independent-core.md) and [reconciliation](core-reconciliation.md).
- [Independent backend report](independent-backend.md) and [reconciliation](backend-reconciliation.md).
- [Independent platform report](independent-platform.md) and [reconciliation](platform-reconciliation.md).
- [Source inventory](inventory.json): 68 example sources including two vendored contract copies, four std sources and one canonical shared contract; 19 top-level example projects.

## Fresh verification

| Check | Raw result | Interpretation |
| --- | --- | --- |
| Runtime lint/format/typecheck | [runtime-check.log](runtime-check.log) | Passed. |
| Runtime test directory | [runtime-tests.log](runtime-tests.log) | 705 pass, 39 skip, one Chromium launch failure caused by sandbox permissions. |
| Exact failed browser-owner file rerun outside sandbox | [browser-owner-rerun.log](browser-owner-rerun.log) | 2 pass, 0 fail. Initial runtime failure was before browser test execution. Do not add both runs' counts. |
| Compiler internal suite | [compiler-summary.json](compiler-summary.json), [JSONL log](compiler-tests.jsonl) | 12 test-bearing packages pass, 2,298 pass test/subtest events, five skipped tests; two packages have no tests. |
| Whole-program application/stdlib checks | [example-checks.jsonl](example-checks.jsonl) | All 23 projects pass: 444 function entries and 1,131 assertion roots checked, not all roots freshly executed. Browser closure checked for grid. |
| Focused core tests | [focused-tests.log](core-probes/focused-tests.log) | 67 top-level tests plus subtests pass; overlaps compiler suite, not additive. |

Runtime skips: 16 MySQL, 12 S3 and 11 browser-guard live tests. Compiler skips: three packaged tests requiring a qualified archive, one live MySQL differential test, one deliberately invalid lexer/formatter fixture. The full packaged browser/database matrix, release/Linux qualification, live S3, load testing and provider quality evaluation were not rerun.

Whole-program checks used [check-examples.go](check-examples.go), temporarily placed under `compiler/review_20260926` so Go's internal package rules permit imports. The runner was removed after execution. To reproduce, place the saved file in a new temporary subdirectory of `compiler`, run it from repository root at the reviewed revision, then remove that temporary directory. `CAN_BUN=/Users/vince/.bun/bin/bun` and `GOCACHE=/tmp/can-review-20260926-go-cache` were used for the compiler suite; local Bun was 1.4.2 and Go was 1.27.1.

## Focused probes and limitations

- `core-probes/` preserves source projects, generated program/assertion modules, custom entry drivers, rejected inputs, logs and the additive Go overlay harness. Final recursion evidence is `recurse-small-runtime-v3.log` (100 steps, success) versus `recurse-large-runtime-v3.log` (20,000 steps, native stack overflow). Earlier errors from using the lower-level emitter without publication diagnostic assets are retained but are not findings.
- `core-probes/result-data-generic-assert-v2.log` records five passing assertion roots. They establish representation and returned values; a no-retry implementation could pass those same examples. Do not claim retry count or sequencing is proven.
- `core-probes/owner-test-input/`, `error-generic/`, `named-final-local/` and their logs retain checker rejection cases. A private fixture helper is a valid owner-value workaround; no impossibility theorem is claimed.
- To avoid committing six duplicate private runtime trees, copied runtime modules were omitted from `core-probes` and their exact hashes saved in `runtime-copy-hashes.json`. Likewise, the twelve emitted `htmx-4.0.0.min.js` / `htmx-guard.js` blobs (all byte-identical to `distribution/assets/`) were replaced by `emitted-asset-hashes.json` during cleanup; reproduce bytes from the pinned distribution files. The original full probe trees remain under `/tmp/can-review-core-20260926` for this session. For independent replay use the retained overlay harness at the reviewed revision to regenerate modules/runtime, and adapt the saved absolute paths to the new scratch directory. The probe uses compiler-produced functions with a custom entry, not the qualified release CLI.
- `platform-probes/dom-semantics.mjs` and its JSON output use actual Chromium DOM behavior corresponding to the adapter's calls. This establishes dirty-input attribute/property divergence and indistinguishable checkbox snapshots. It is not an end-to-end Can build test. Chromium required execution outside the filesystem sandbox.
- `backend-probes/race-drain.ts` and its output demonstrate that a selected race does not finish an owning root until its loser settles. This is a deterministic current-runtime experiment, not an HTTP latency benchmark.
- The S3 cancellation/deletion discrepancy and deadline issue are source-based findings in the backend reconciliation and main report. No destructive live storage experiment was performed.

## Required design consultations

[Jev summary](jev-summary.md), `jev-request-1.json` through `jev-request-3.json`, matching response/metadata files, `jev-wording-audit.json`, and the preparation/request scripts preserve the exact consultations. All explanatory fields were rewritten before sending, with facts/alternatives retained and manually checked. Disagreements were investigated through follow-up source review; no majority-vote decision was used. The TypeSafe skill and current official API/Choice documentation guided the calls. No credential values are saved.

`artifact-validation.json` records final link checks, unchanged source hashes/revision and the final working-tree scope.
