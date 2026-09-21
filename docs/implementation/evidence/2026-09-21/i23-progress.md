# I23 historical progress record

This record predates validation. See [I23 validation](i23-validation.md) for the
completed implementation and checks.

At the time of this record I23 was incomplete. Its initial code remained in the worktree while
the independently confirmed I21 admission defects are corrected and committed
separately.

`runtime/number.ts` now has an uncommitted `createExactAmounts` factory for the
three closed C8 operations: native truncating divmod, Euclidean remainder sign
adjustment, and finite half-even ratio rounding with positive denominator and
recomputed exact remainder. The checker and emitter have preliminary wiring for
these contracts and their sealed record/error identities.

Next: add sign/tie/large-integer invariant tests, a current exact-amount project,
source negatives, offline staged assertions and CLI execution, strict TypeScript,
full gates and task evidence. Do not mark I23 complete before those checks pass.
