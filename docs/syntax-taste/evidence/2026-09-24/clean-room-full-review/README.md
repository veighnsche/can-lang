# Evidence: independent full language-design review

Date: 24 September 2026. Source revision: `4c2db1e1162a45e88a58ef154e848790a1a44c60`.

The [synthesized report](/Users/vince/Projects/can-lang/docs/syntax-taste/clean-room-language-review-2026-09-24.md) is the calibrated conclusion. Raw reviewer priorities were not automatically adopted. Files in this directory record observations and recommendations; they do not amend the language contract.

## Independent review protocol

All three agents were spawned with no inherited conversation history. They received the repository location, current task, selected design constraints, and scoped current source/specification areas. They were asked to exclude previous reviews, recommendation/disposition ledgers, historical consultation outcomes, and sibling reports during the independent phase.

| Agent | Model/effort | Coverage | Saved report |
| --- | --- | --- | --- |
| `clean_core_v2` | GPT-6 Astra / high | Syntax, nominal data, variants, patterns, generics, errors, callables, primitives, collections; gallery and maintained standard-library examples | `independent-core.md` |
| `clean_native_v2` | GPT-6 Sol / high | Completion, native AI, coordination, ownership; native-AI, stream, websocket, files, process examples | `independent-native.md` |
| `clean_platform_v2` | GPT-6 Sol / high | Modules, HTML, forms, routing, SQL, ecosystem boundary; account-search, form-validation, dashboard, language-site, cookies, crypto, sqlite, mysql, utilities, markdown | `independent-platform.md` |

The independent phase was followed by explicit reconciliation and report checking. The platform report includes a labeled scope-completeness followup about browser execution. Native/core initial reports remain unchanged; their qualifications after source rechecking are recorded in `reconciliation.md` and incorporated in the synthesis.

The coordinator retained earlier conversation context. “Clean room” here means fresh independent source reviews, not a claim that the entire coordinating process had no prior knowledge. Normative baseline constraints were supplied intentionally; this is not an unconstrained redesign from an empty grammar.

`example-inventory.json` records paths, line counts, and SHA-256 values for the 56 current example source files and four maintained standard-library sources. It is a source-coverage inventory, not an execution manifest. Historical examples were not treated as current language requirements.

## Fresh behavior probes

`cases/` contains all ten source projects and manifests. `probe-results.json` preserves raw checker results and each emitted assertion process's stdout/stderr. `probe-summary.json` is a compact view.

| Case | Checker | Emitted roots | Meaning |
| --- | --- | --- | --- |
| `pattern-correct` | Accepts | 3 pass | Correct three-leaf match |
| `pattern-typo` | Accepts | 3 pass | `decliend` becomes an `any` binding |
| `pattern-typo-new-leaf` | Accepts | 3 pass | Added `refunded` does not expose the typo |
| `pattern-correct-new-leaf` | Rejects | None | Correct old arms become non-exhaustive |
| `pattern-constructor-typo` | Rejects | None | Unknown explicit constructor is diagnosed |
| `variant-direct` | Rejects | None | Direct differing generic specialization rejected |
| `variant-via-bridge` | Accepts | 2 pass | Same value admitted via another variant |
| `variant-via-leaf` | Accepts | 2 pass | Narrowed leaf can enter either specialization |
| `fixture-original` | Accepts | 3 pass | Caller root label selects helper fixture |
| `fixture-renamed-root` | Accepts | 2 pass, 1 expected failure | Only caller label changed; helper now executes the actual operation |

Total: ten programs, seven accepted, three rejected; 19 emitted assertion roots, 18 passes, one intentionally reproduced outcome mismatch. The new-leaf probes establish checker coverage behavior; they do not include an extra assertion invoking the new leaf.

To repeat in this workspace:

```sh
python3 docs/syntax-taste/evidence/2026-09-24/clean-room-full-review/run-probes.py
```

The script has explicit absolute workspace/output paths. It creates its own temporary Go driver under `compiler/.clean-review-probe`, builds against the current internal project loader/checker/emitter, and removes that driver in `finally`. It refuses to create over an existing runner directory. It overwrites its own saved probe artifacts. The source of the driver is also saved as `probe-runner.go.txt`.

Generated programs run with local Bun 1.4.2 against a symlink to the repository runtime. Each assertion root runs in a fresh process, with an empty native environment supplied on file descriptor 3 and a 30-second external deadline. A minimal empty diagnostic source index enables the harness; source-map behavior is not tested. An initial harness invocation lacked the required file descriptor; the harness was corrected before collecting the saved results. These are internal checker/emitter probes, not packaged release or verified-build qualification.

## Jev consultations

The review used the [TypeSafe skill](/Users/vince/Projects/can-lang/.agents/skills/typesafe-ai/SKILL.md) and current official [API](https://docs.typesafe.ai/api.md) and [Choice](https://docs.typesafe.ai/primitives/choice.md) guidance. Jev was supplied source facts and observed probe outcomes; it did not research the repository. Its outputs are advisory classifications, not explanations, proof, or implementation approval.

All three requests were prepared before dispatch. Every explanatory context field, question instruction, and option description was independently rewritten. Stable technical identifiers and alternatives were retained. The coordinator manually compared factual meaning and mechanically checked corresponding wording for differences before sending. `jev-wording-audit.json` records the compared fields and alternatives. This procedure reduces repeated wording; it cannot establish independence of model errors or remove framing bias.

Files `jev-request-1.json` through `jev-request-3.json` are the exact fresh requests; the response and metadata files preserve each result. No earlier answer was inserted into a subsequent request. The requested alias was `jev-latest`; all responses reported `jev-1.13.0`. The three requests used 6,503 input tokens and 1,288 output tokens in total.

The following are selected-option probabilities, not confidence values or empirical correctness rates. Complete distributions and returned confidence values remain in the raw responses.

| Question | Request 1 | Request 2 | Request 3 |
| --- | --- | --- | --- |
| Pattern intent | Explicit binding 0.88 | Explicit binding 0.73 | Explicit binding 0.44 |
| Variant identity | Extensional 0.57 | Extensional 0.71 | Retain current guard 0.72 |
| Abstract values | Owner control 0.97 | Catalogue only 0.70 | Catalogue only 0.67 |
| Generic contracts | Explicit-operation baseline 0.84 | Explicit-operation baseline 0.90 | Explicit-operation baseline 0.90 |
| Generic errors | Finite-row experiment 0.93 | Finite-row experiment 0.54 | Finite-row experiment 0.77 |
| Fixture ownership | Explicit seams 0.66 | Explicit seams 0.85 | Explicit seams 0.94 |
| Race fault observation | Current timing 0.90 | Current timing 0.61 | All losing standard faults 0.50 |
| Scoped results | Narrow diagnostics 0.95 | Narrow diagnostics 0.90 | Narrow diagnostics 0.80 |

Disagreements were investigated against implementation, probes, tests, and normative contracts; see `reconciliation.md`. No majority vote established a design requirement. Even unanimous directions remain experiments where the appropriate surface is unproven. The consultation did not decide every platform proposal in the full report.

## Verification and reproducibility limits

`runtime-check.txt`: successful `bun run check:runtime` (lint, formatting check, TypeScript).

`compiler-tests.jsonl`: successful invocation:

```sh
GOCACHE=/tmp/can-saas-review-go-cache go test -json ./compiler/internal/syntax ./compiler/internal/types ./compiler/internal/check ./compiler/internal/resolve ./compiler/internal/emit ./compiler/internal/driver
```

All six packages passed. The log contains 1,089 passed test/subtest events and 14 skipped events. One skip is the intentional negative `lexer/core.can` formatting fixture; 11 require `CAN_BUN`, and two driver tests require `CAN_BUN_ARCHIVE`. No such configuration was supplied to this invocation. `verification-summary.json` preserves each skip and its exact reason. Counts of test/subtest events are not counts of independent language properties. Go may reuse eligible cached results.

`runtime-tests.txt`: successful invocation:

```sh
bun test ./runtime/test/coordination.test.ts ./runtime/test/owner.test.ts ./runtime/test/owner-roots.test.ts ./runtime/test/collections.test.ts ./runtime/test/assertions.test.ts ./runtime/test/assert-identity.test.ts ./runtime/test/codec-json.test.ts ./runtime/test/html.test.ts ./runtime/test/sql-descriptor.test.ts
```

Result: 77 passes, zero failures, 870 expectations, nine files. These checks supplement source review; they do not prove every finding or proposed remedy.

No production compiler/runtime/example file was persistently changed. Live database integration, generated Can provider execution, browser tests, all-example builds, and the complete release/installation/source-map qualification suite were not rerun. The only live model traffic was the three direct advisory consultations recorded here.
