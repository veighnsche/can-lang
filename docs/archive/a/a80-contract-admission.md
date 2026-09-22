# a80: contract admission (verifier slice 1, no solver)

Status: shipped, unwired. `CheckContractAdmission` (compiler/verify_admission.go)
classifies every contracted function before any solver runs. It is a
pure function with probe-first tests; no pipeline calls it yet.
Activation ("verifier checking is active") waits for the proving
slices. New codes CAN4301/4302/4303, all with `canlc explain` entries.

## The fragment

Mathematical integers, Booleans, and finite acyclic records whose
leaves are those sorts. Integer terms: literals, input/binder
projections, addition, subtraction, multiplication by a literal
coefficient (`2 * (x + y)` admitted after expansion; `x * y`
rejected even on paths that might pin `x`). Comparisons `== != <
<= > >=` over integers, Boolean equality, same-nominal record
equality flattened into field equalities. Brands, `Seq`, `Bytes`,
`dec`, `str`, variants, division, remainder, and mixed sorts are
outside. One unsupported clause rejects the complete contract:
supported clauses are never proved while the rest is silently
omitted.

## Findings

- CAN4301 malformed: non-Boolean requires rows, unknown binders
  or fields, missing/duplicate outcome arms (verdict 3A: exactly
  one arm for `Ok` and every emits kind), ill-sorted combinations,
  non-Boolean ensures-match arms. A predicate position demands a
  Boolean expression, so a never-Boolean shape (constructor, seal)
  is malformed even though its sort is also unsupported.
- CAN4302 unsupported: fragment-external sorts/operators in
  Boolean-shaped predicates, `requires false` (empty admitted
  domain proves vacuously), extern and kernel calls (tested
  behavior is not a proof rule), state effects, direct and mutual
  recursion in the contracted closure (no induction rule;
  language termination rules unchanged).
- CAN4303 unavailable dependency: any function call from a
  contracted body, contracted or not. With no proving run yet,
  no summary is established; a plain callee's unstated summary
  is equally unusable. Self-calls stay silent here so the
  recursion pass reports one cycle finding.

## Deliberate decisions

- Binder paths in bodies (`on Ok r => Ok(value = r.value)`) are
  not name-resolved by admission; the structural proof follows
  them. Only resolvably inadmissible payloads are flagged.
- `ensures false` on a declared error is admitted (verdict 3C):
  the verified body never returns it; the arm, match, and
  injection rows stay.
- Test-row admissibility (`given` rows satisfying `requires`)
  is deferred to the solver-boundary slice, where it becomes
  one more obligation shape, not a separate evaluator.
- Strictness (str-valued construction, brand-typed inputs) is
  intentional: relaxing admission is a later capability with
  its own probes, never silent widening.

## Verification

- 15 probes in compiler/contract_admission_test.go: both public
  pilots admitted, ensures-false admitted, 6 malformed, 5
  unsupported (including the verdict's `x * x` counterexample
  and a provable-but-recursive loop), 2 unavailable with
  caller-side attribution, uncontracted functions silent.
- Gates: suite ok, modcheck 25 OK, gramcheck OK, tsc clean.
