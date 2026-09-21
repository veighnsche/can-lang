# I46 — concrete generics and finite inference

## Implementation

`compiler/internal/types/infer.go` resolves finite inference patterns and solves exact equalities against sealed concrete types. Constraints are transactional; failure cannot retain partial bindings. `specialize.go` admits isolated sealed graph extensions and interns concrete type identities. Source function instances use resolved declaration identity and ordered argument identities, with a shared whole-body checking worklist.

Calls, generic constructors, receiver methods and callable references infer concrete arguments. Expected callable contracts exclude captured receiver/near inputs; near captures supply exact constraints. Assertion rows infer independent concrete instances and schedule complete bodies even when application code does not call them. Deferred arguments such as empty arrays are retried only while constraints make progress. No union, numeric widening, error-set inference or arbitrary instance search is introduced.

Same-instance recursion uses the cache. Proven structural growth along a repeated generic declaration is rejected, and separately labelled finite implementation bounds stop excessive discovery. Existing recursive nominal graph inhabitation/expansion admission remains in force. Generic body diagnostics name the source and requesting application sites.

## Acceptance evidence

| Requirement | Evidence |
| --- | --- |
| Cache by declaration/concrete arguments | `TestSpecializationKeysAndAggregateConstraints`, `TestExplicitGenericFunctionInstances`, `TestGenericCrossModuleIdentity` |
| Preserve recursive nominal identity and sealed evidence | `TestSpecializationExtendsSealedEvidence`, `TestFailedExtensionSealDoesNotPoisonEarlierTypes`; isolated graph copies preserve back edges and failed admissions leave prior snapshots valid |
| Whole reachable body, including unexecuted branches | `TestGenericWholeBodyAndRecursion`; `TestGenericAssertionsOwnInstances`; staged integration replaces the generic body with an invalid unexecuted branch and requires build failure |
| Invariance and finite ambiguity | `TestInferenceRejectsConflictAmbiguityAndWidening`, `TestInferenceExpectedAndFixedConstraintsAgree`, `TestGenericCallInference`, `TestGenericConstructorInference` |
| Specialized callables and methods | `TestExplicitGenericCallableReference`, `TestGenericReferenceInference`, `TestGenericMethods`; expected signatures and exact near captures, with conflicting contracts rejected |
| Same-instance recursion versus expanding recursion | `TestGenericWholeBodyAndRecursion`; existing builder tests also reject infinitely expanding nominal type graphs |
| Aggregate expected-type constraints | `TestSpecializationKeysAndAggregateConstraints`, `TestFiniteInferenceEqualities`: `all_failed<item>` resolves exactly from `all_failed<failures>`; the shared solver rejects unresolved/conflicting constraints finitely. Coordination invocation lowering remains I19 |
| Shared instance across source modules | `TestGenericCrossModuleIdentity` requires one `identity<box<int>>` instance requested from helper and main modules; nominal result identity equals its concrete argument |
| Real generated code, offline staged release | `TestCurrentBundledGenerics`: builds a fresh release, invokes absolute `bin/canlc` outside project cwd with `PATH=/nonexistent` and networking denied, executes six assertion rows and main across three source modules, checks recursive record equality and a generic captured method reference |

## Validation

Bun 1.4.2 (`744846f84`), archive `/private/tmp/can-i01-bun.zip`:

```
CAN_BUN_ARCHIVE=/private/tmp/can-i01-bun.zip \
CAN_BUN=/Users/vince/.bun/bin/bun GOCACHE=/tmp/can-i02-go-cache \
go test ./compiler/... ./tests/integration -count=1
```

All packages passed, including the driver development-sidecar checks and every staged integration test. `git diff --check` passed. The targeted staged generic integration also passed independently before the full run.

The three saved design consultations and equivalence audit are in [i46-jev](i46-jev/README.md). They advised isolated sealed extensions; tests, not consultation agreement, supply the implementation evidence. [Usage and limits](../../generics.md) documents the shipped behavior. Superseded legacy stamping files remain outside the active CLI path pending their dependency-gated I44 deletion.
