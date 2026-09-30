# Independent expression recovery repair

Root review found that the canonical expression checker still returned the first child failure for binary/comparison operands, array elements, nominal constructor fields, update values, index receiver/index, and slice bounds. Region-level recovery alone could not expose those siblings.

`Expressions` now receives the canonical recover-only checkpoint callback. Failed children roll back before their next independent sibling is checked. A parent with any failed required child returns no executable expression. Operators run only with proven operand types; comparison operator diagnostics retain their exact operator spans. Standalone expression calls also collect independent argument failures. Strict checking still stops and rejects invalid programs.

The new driver regression checks seven paired-error cases through `CheckSnapshot`, requiring exactly both missing-name token ranges, no invented operator/type cascade, no executable function Region, and strict compiler rejection. Bounded check passed: `go test -p 1 ./compiler/internal/driver -run '^TestRecoveryIndependentExpressionChildren$' -count=1 -timeout=60s` (0.639s).

Unknown update keys and unavailable receiver types block their value type-check because the expected field type is unavailable; replacing it with an unconstrained nil expected type would create false errors for contextual values. Independently known update fields remain checkable. Slice receiver errors use the receiver span.

Independent review of the transaction callback found additional mutable region/aggregate state and mixed deferred/direct joined-error handling; the compiler owner is repairing those before the final gate. Full relevant package and integration verification remain pending.
