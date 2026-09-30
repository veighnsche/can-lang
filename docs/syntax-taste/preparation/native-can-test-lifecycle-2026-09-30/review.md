# Final lifecycle review

Status: source/document review only. No implementation, builds, suite execution, live services, recovery probes or measurements.

The [lifecycle contract](../../native-can-test-lifecycle-2026-09-30.md) was informed by [resource-failure review](resource-failure-review.md), [coverage/report review](coverage-report-review.md) and [concurrency/reuse review](concurrency-reuse-review.md). Independent final readings are retained in [final-resource-review.md](final-resource-review.md), [final-coverage-review.md](final-coverage-review.md) and [final-reuse-review.md](final-reuse-review.md). They describe the drafts each reviewer saw; this record gives final dispositions.

| Finding | Final disposition |
| --- | --- |
| A single outcome hides mismatch plus missing work/cleanup failure | Separate scope, admission, execution, behavior, checks, evidence, cleanup and verification facts; derive concise outcome/exit from those facts |
| Short-circuit mismatch could be confused with an unrun case | Valid named failure may leave checks not reached and exit 1 when all selected cases finish validly; unstarted selected cases yield 2. Unperformed checks get no coverage credit in either case. |
| Retry attempts cannot all be expanded before execution | Freeze case/variant matrix and finite attempt policy; append actual attempt IDs/reasons under that matrix, preserving history and remaining budget |
| Worker seal followed by late native callback | Required intervals seal before successful body terminal. Later cleanup tail is explicitly separate; a callback belonging to a promised final seal invalidates that seal and dependent evidence. Post-cleanup behavioral tests use an intact outer judge. |
| Finalized resource accounting was conflated with a passing run | Clean disposal can finalize a nonzero generic incomplete envelope after controller failure. Only success requires a valid Can report and matching clean N receipt. |
| Cleanup reserve could be consumed by case bodies/verification | Absolute total, execution-cutoff and aggregate preparation deadlines; admit only when declared body plus case cleanup fits before untouched final reserve; cooperative cancellation is cleanup time |
| New profile budgets were not measurements | Numeric defaults are explicitly conservative policy ceilings. Current source only supports selected existing durations/limits; host enforcement and adequacy remain qualification work. No automatic increase or skip. |
| Default case body can be shorter than a browser/lifecycle critical path | Explicit named overrides and pre-admission critical-path checks; a 120-second drain cannot share a 120-second body with startup/observation |
| Phase-by-phase reservation risks scarce-resource deadlock | Reserve whole-case peak including held resources, then consume phase leases within it; reservation is not possession of the sole active build producer token |
| One run per root is not a host-wide cap | Require shared host admission across managed repositories/ownership roots; different roots cannot bypass it. Do not claim to govern unrelated host applications. |
| Fresh builds accidentally join the same content-key result | Distinct production ID/reuse mode bypasses coalescing, while content key remains comparable; one active producer still applies |
| Build key omits actual dependencies or paired assets | Bind source namespace, locks/assets/browser pairing, toolchains/target/options/environment and validation identity; reconcile actual-read/resolution facts; unexpected dependencies invalidate closed-input reuse |
| Corruption transparently repaired | Stop admission and invalidate affected dependent evidence; do not turn corruption into a benign cache miss. Intentional corruption uses private copies. |
| Prior failed rejection reused as fresh evidence | Compiler rejection and fresh-build obligations require actual identified executions; failed fills never issue reusable artifact handles |
| Cross-job report set could be cherry-picked | Campaign ID and append-only registered job/attempt index; account for failures/blocked/interrupted jobs, with explicit corrective-run supersession. Never union partial attempts into one completed case. |

Expected subject failure remains raw data checked by a healthy reference judge. Expected subordinate-runner failure is permitted only within independently owned outer resources. The top-level judge/N cannot declare its own crash a pass. Missing selected environments are blocked, while explicit preplanned exclusions remain partial scope. A cleanup failure retains ownership/quota and stops further live admission; later recovery cannot rewrite the failed run.

The chosen policy keeps one live case through cleanup, one offline verification worker and one build producer; verification precedes live work and no speculative prebuild is introduced. A case can still declare multiple bounded subjects/contexts when its obligation requires them. Builds are immutable, leased and shared only inside a run; live browser/database state is fresh per case. No host scenario harness or interpreter was introduced.

Implementation gates remain actual supported-host enforcement, durable recovery/transport mechanisms, fully enumerated capability-specific limits, concrete Can declarations and fixtures, wire schemas, and ledger-to-binding coverage. The design neither asserts these mechanisms exist nor marks any test eligible for deletion.

Related architecture, capabilities, authoring, completion and ledger documents now link the lifecycle rules. [validation.json](validation.json) records local-link checks, consultation field checks and source hashes. Existing unrelated changes were preserved; no task-owned temporary execution resources were created.
