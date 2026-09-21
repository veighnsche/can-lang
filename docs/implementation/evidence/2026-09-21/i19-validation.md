# I19 coordination validation — report v1

Date: 2026-09-21. Runtime: the exact Bun/archive/revision/OS/architecture in `distribution/target.json`. This report records language/compiler behavior; the native capability qualification report remains separate.

## Requirements and evidence

| Requirement | Executable evidence |
| --- | --- |
| Q12 trace 1: all selects first rejection, no success/late redispatch, domain and handled/unhandled standard variants | `runtime/test/coordination.test.ts`: controlled C/B/A settlement, original completion identity, shared handler count and late diagnostics |
| Trace 2: allSettled waits, dispatches A/B/C, stops at handler failure or unhandled standard B | Controlled C/A/B settlement tests assert handler order and absence of partial results |
| Trace 3: any retains original all-failed values in input order; late success consumes earlier failures | Typed aggregate test verifies generic nominal identity/ID/payload, duplicate references and opaque native cause; last-success and earlier/late standard tests verify selection and diagnostics |
| Trace 4: race selects first domain/standard completion, with and without a standard handler | Runtime mode/error/handler matrix verifies the selected completion, one handler and owner-only late diagnostics |
| Trace 5: complete preparation, then flattened launch order; preparation failure launches nothing | `TestGeneratedCoordinationPreparesWholeChains` controls both argument promises, holds the first participant open, observes direct/spread launch order and checks failed preparation bypasses shared handlers |
| Trace 6: empty all/allSettled/any/race | Staged source fixture executes empty typed spreads in the first three modes; runtime and generated empty-race tests observe pending state at a harness deadline without creating a Can timeout |
| Trace 7: heterogeneous direct records and homogeneous nullary spreads | `TestCoordinationHeterogeneousRecords`, `TestCoordinationSpreadUsesDeclaredCommonBound`, and staged `heterogeneous.can`; unrelated direct records normalize to a variant, races require equal success types, and spreads reject input/result/error-bound violations |
| Trace 8: handler failures never redispatch | Controlled all/any tests use domain and standard handler failures and assert exact escaping completion and no later/shared handler |
| Trace 9: generic aggregate composition | `TestCoordinationAggregateComposition` rejects distinct specializations under one bare arm and array covariance; staged `aggregate-composition.can` normalizes arrays elementwise and preserves original payloads |
| Native selection, protected payloads and ownership | Runtime instrumentation forwards to and identifies each native Promise operation; thenable payload test and captured-resource loser test verify protected completion and lease-preserving drain |
| Static coverage and exact contextual inference | Checker tests reject missing/wrong arms, incompatible handler results, uncovered aggregate leaves, absent/conflicting named variants and undeclared handler errors; operand-order and nested-alias regressions require full final checking |
| Complete chains, immutability and strict output | Generated test preserves checked IR, skips later methods on chain failure and validates direct/spread preparation; staged main includes chained native slices; strict TypeScript checks include opaque standard projections |

## Validation commands and results

- Qualified Bun runtime suite: 14 tests, 103 expectations, zero failures. The integration test reruns this suite through the absolute staged Bun with networking denied, `PATH=/nonexistent`, and an unrelated working directory.
- `go test ./compiler/... ./tests/integration -count=1` passes with `CAN_BUN`, `CAN_BUN_ARCHIVE`, `CAN_TSC` and the isolated Go cache configured. This includes existing compiler behavior, all staged coordination fixtures and development-sidecar integrity checks.
- `TestCurrentBundledCoordination` builds a fresh release and runs `assert` and `run` for three source fixtures: `main` (8 assertion rows), `aggregate-composition` (8), and `heterogeneous` (7). Every generated TypeScript module is checked with strict TypeScript settings when `CAN_TSC` is set.
- The generated preparation/empty-race harness is also checked with strict TypeScript, then executed on qualified Bun. Its optional archive path runs the harness through an absolute staged runtime under network denial, so CI does not require an ambient Bun installation.
- `git diff --check` passes.

Design consultations: [ownership publication](i19-jev/README.md), [contextual aggregate constraints](i19-aggregate-jev/README.md), and [chain preparation](i19-chain-jev/README.md). Requests, responses and wording audits are retained. The lower-confidence ownership result was investigated against Q7/Q10 and controlled pre-/post-winner failures; classifier agreement is not treated as proof.

I18 remains separate: deterministic fixture queues and participant invocation-path allocation are not claimed by this report. I21 likewise owns the broader immutable collection callback catalogue.

## Independent review correction: linear late drain

Review found that every completion after group publication scanned all participants again. This made draining an early winner's remaining participants quadratic even when none failed. Publication now scans outcomes once; each subsequent completion examines only its own participant. Pending-count removal, wakeups, selected outcomes and original-failure occurrence deduplication retain their existing behavior.

The count-based owner regression releases 256 losing participants carrying one shared standard-failure occurrence after publication. It requires exactly 256 selected-index inspections and one diagnostic. Restoring the full-group scan makes the same test fail with 32,896 inspections. The regression uses operation counts rather than timing thresholds and restores its temporary Set instrumentation in a finally block.

Validation: owner and coordination suites pass (32 tests, 191 expectations), and strict TypeScript checking passes for the owner implementation and tests. `TestCurrentBundledAssertions` and `TestCurrentBundledCoordination` also pass using the qualified staged archive, absolute Bun, network denial and strict generated TypeScript checking. These staged checks include the in-progress I18 changes; the owner correction is committed separately.
