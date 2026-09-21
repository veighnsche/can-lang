# I10 independent review corrections

The monitor reproduced two defects in `8c0461b` before I11 integration.

1. `invocationValue` returned deferred `$canValue(box)` text. Consumers could
   execute later operands before extracting that payload; a wildcard match could
   omit extraction entirely and swallow a standard failure. The emitter now
   materializes extraction in an immediate statement and returns only its
   temporary. Failure therefore leaves the current evaluation before later
   operands, array elements, comparisons or scrutinees can run.
2. Repeated references to one named variant value independently created the same
   inferred arm-local name. The checker now keys inferred narrowings by resolved
   binding identity, shares the local for compatible occurrences, preserves
   parenthesized binding identity and rejects disjoint nominal requirements on
   the same immutable scrutinee. Explicit pattern bindings retain their existing
   duplicate-name rules.

`i10-review-before.txt` reproduces the original failures. Eight emitted traces in
`region_order_test.go` cover binary/array/comparison evaluation, multiple
scrutinees, wildcard and value matches, logical operands and constructors.
Every corrected trace reports a standard completion with exactly one event:
`first`. The existing constructor/logical controls already stopped correctly;
the other six expose the deferred-extraction defect before the correction.

`TestRepeatedScrutineeNarrowing` covers repeated, parenthesized and one-sided
narrowings plus incompatible leaves. The larger generated-region suite executes
both repeated and parenthesized cases and observes the narrowed field value.

Validation passed:

```sh
CAN_REGION_TEST_OUTPUT=/tmp/can-i10-reviewed-regions.ts CAN_BUN=/Users/vince/.bun/bin/bun GOCACHE=/tmp/can-i02-go-cache go test -count=1 ./...
# i10-review-go-tests.txt
CAN_BUN=/Users/vince/.bun/bin/bun CAN_BUN_ARCHIVE=/tmp/can-i01-bun.zip GOCACHE=/tmp/can-i02-go-cache go test -count=1 -v ./compiler/internal/emit ./compiler/internal/driver ./tests/integration -run '^(TestEmittedFailureEvaluationOrder|TestRepeatedScrutineeNarrowing|TestEmittedCompletionRegions|TestPackagedOutputValidationAndExecution|TestDevelopmentSidecar)$'
# i10-review-offline.txt
```

The staged emitted-order and completion suites use the absolute bundled Bun,
network denial, private working directory/environment and disabled ambient
configuration. Existing output publication, inherited leases and sidecar integrity
checks also passed. `i10-review-regressions.txt` records the initial focused pass;
`i10-review-typescript.txt` records the strict TypeScript 7.0.2 check of the updated
generated integration program. No carrier/runtime semantics changed.
