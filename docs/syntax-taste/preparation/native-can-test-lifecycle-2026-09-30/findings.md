# Lifecycle decision evidence

Status: source review and design only; no suite, build, service, benchmark or host-enforcement probe ran. The [main lifecycle contract](../../native-can-test-lifecycle-2026-09-30.md) supplies common failure, coverage, cancellation, concurrency and reuse rules for implementation lanes.

## Source review and decisions

Three independent reviews examined [resource failures](resource-failure-review.md), [coverage/reporting](coverage-report-review.md) and [concurrency/reuse](concurrency-reuse-review.md).

- The authoring contract already separates behavior, execution and cleanup. The lifecycle design adds explicit planned scope, admission, evidence and root-verification axes. A selected subset may finish green locally; selected blocked/unrun work cannot be removed after the fact. A valid named mismatch may short-circuit remaining checks without becoming a protocol error, but those unperformed checks gain no coverage.
- Current integration sources skip missing archive, database, storage and browser prerequisites. The new design preserves their environment obligations rather than translating missing selected legs into passes. Deliberate exclusions are established in the plan; missing required service after selection is blocked, and loss after admission is an execution failure unless deliberately tested as subject behavior.
- Current assertion supervision enforces an external deadline and direct-worker reap. It does not prove full descendant or remote-resource release. Current launcher removal errors can be ignored; the new receipt must retain them. Contained local failures can permit later cases, whereas uncertain ownership/cleanup/shared authority stops live admission.
- Current bundle reuse is run-local, keyed and verified, with direct builds when building is the subject. The replacement retains those valuable properties but removes age-only lock stealing and transparent recovery from shared corruption. Failed fills, lost acknowledgment, leases and identity changes are explicit outcomes.
- Current temporary-cache recovery uses markers, locks, identity checks and conservative liveness inspection. The new contract requires externally registered acquisition intent and durable unresolved ownership, not just a marker created after allocation. Recovery cleans old work; it does not resume old Can functions or turn abandoned builds into cache hits.

The main contract resolves several preliminary-note differences. Closing the worker evidence channel does not close native observations prematurely; required intervals are sealed before a successful body terminal and N continues resource/cleanup facts afterward. A failed execution can still reach a finalized nonzero receipt after clean disposal. The final policy permits diagnostic retries but does not let a passing retry erase an earlier failure in the same run. Optional/profile exclusions cannot retroactively change selected requirements. Live observations are not cached as substitutes for new case/engine execution; build reuse is distinct from explicit cross-job evidence aggregation.

The concurrency review preferred leaving most numeric ceilings to the first executable profile. The main document instead specifies initial conservative policy ceilings because the completion contract requires explicit defaults. It labels them unmeasured, preserves finite reviewed overrides, and makes host enforcement/admission qualification mandatory before use. These values are not attributed to source measurements or Jev. Existing 12–30 minute test contexts and a 120-second drain obligation explain why the quick ten-minute envelope cannot represent the complete suite. No performance runs are needed or authorized merely to record these policy choices.

## Jev consultations

All three request contexts, question instructions and option descriptions were independently reworded before sending. Facts, constraints and alternatives were held constant; technical identifiers and option keys stayed stable. Manual semantic comparison and complete-field text differences were checked first. No earlier answer was sent in a later request. Each request used `jev-latest` and each response identifies `jev-1.13.0`.

| Decision | Preferred option | Option probability, requests 1 / 2 / 3 |
| --- | --- | --- |
| Coverage/result representation | Separate scope, behavior, execution and cleanup facts | 1.00 / 0.98 / 1.00 |
| Continuing after failure | Continue only after confirmed local containment/cleanup; stop on shared uncertainty/interruption | 1.00 / 1.00 / 1.00 |
| Build reuse | Same-run verified immutable results, explicit fresh-build cases and visible corruption | 1.00 / 1.00 / 1.00 |

- [Request 1](request-1.json), [response 1](response-1.json), [transport metadata](response-1.metadata.json).
- [Request 2](request-2.json), [response 2](response-2.json), [transport metadata](response-2.metadata.json).
- [Request 3](request-3.json), [response 3](response-3.json), [transport metadata](response-3.metadata.json).
- [Pre-send wording audit](wording-audit.json), [reproduction script](consult.py).

No winning alternative differed. Request 2 assigned 0.02 to a single terminal status with secondary diagnostics. A concise displayed status and small exit-code set remain useful, but they are derived from independent facts; flattening the source record would hide simultaneous mismatch, missing work and cleanup failure. Universal fail-fast remains an explicit Can option, not the default for already-contained local failures. Fresh builds remain mandatory when building is the subject; transparent repair of shared corruption cannot preserve trustworthy prior evidence.

These are classifier probabilities, not measured reliability. Jev did not research the repository, prove containment or validate the concrete time/memory/disk choices, report protocol or recovery implementation. Agreement and wording changes do not guarantee bias removal. The live [API](https://docs.typesafe.ai/api), [Choice](https://docs.typesafe.ai/primitives/choice) and [documentation index](https://docs.typesafe.ai/llms.txt) were read for this step; Markdown endpoints were unavailable, so their normal pages were used. Credentials came from the environment and were not saved.

The [final review](review.md) records contract corrections and implementation gates. [Validation](validation.json) covers documentation, local source links, consultation structure and source fingerprints only. No temporary execution directories, new worktrees or live subject resources were allocated. Existing unrelated workspace changes and `output/` were preserved.
