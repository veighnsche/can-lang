# I06 independent review: unused generic annotations

Two confirmed review findings were corrected:

1. An unused generic could repeat `box<item>` in variant alternatives or
   `failed<item>` in an emits bound because only fully concrete types entered the
   duplicate sets. Resolved symbolic keys now preserve declaration and parameter
   identity, normalize aliases, and normalize callable error-set order. Symbolic
   variant expansion substitutes through nested source/catalogue variants, so
   definite transitive overlap and variant-only cycles also reject.
2. `collections::map<float,item>` escaped the map-key constraint until use because
   its second argument was unknown. Constraints now check each argument's known
   outer kind independently. Definite record/array keys reject too; an actual
   parameter key remains deferred because a later valid scalar substitution may
   satisfy it.

The review's mandatory generic record-cycle fixture exposed another definite
annotation error. Symbolic inhabitation assumes parameters could be inhabited,
then computes the least fixed point on required record/variant paths. A mandatory
self/mutual cycle remains impossible under that allowance and rejects. Ordinary
generic fields, arrays and option-based recursion continue to pass.

Controls prove this is not guessed specialization: `box<left>` and `box<right>`
remain distinct symbolic alternatives, a concrete int/str use passes, and a
concrete int/int use still fails overlap. Bound order and import aliases cannot
hide an identical symbolic type. Parameter-dependent constraints are not converted
into blanket template rejection.

Evidence:

- [Type regression suite](i06-symbolic-regression-tests.txt), including symbolic
  duplicates, nested variants, aliases, callable-bound normalization, per-argument
  catalogue constraints, required cycles and positive/deferred controls.
- CLI regressions in `compiler/current_types_test.go` verify all three originally
  reproduced annotation failures produce nonzero status and no partial report.
- [Full Go suite](i06-symbolic-go-tests.txt), with exact-qualified generated native
  execution enabled: passed.
- [Pinned staged offline integration](i06-symbolic-offline-integration.txt): passed,
  including source/bundle integrity and earlier runtime negative checks.

These checks enforce existing C4 declaration obligations. They do not replace
I46's reachable concrete body checking or generic inference, and do not mark I49
complete; I49 remains the next ledger task.
