# a81: proof obligations + z3 boundary (verifier slice 2)

Status: shipped, unwired. `VerifyContracts` (compiler/verify_prove.go)
proves admitted functions callee-first through z3 QF_LIA queries.
Pure classifier with probe-first tests; pipeline activation is a
later slice. New codes CAN4304/4305/4306, all with `canlc explain`
entries.

## Obligations

Per exit: Requires ∧ Path ⟹ Ensures, discharged only by an
unsatisfiable negation. Per call site: the caller path ⟹ callee
requires, then each outcome arm extends the path with the
verified summary (fresh payload variables constrained by the
callee ensures). Summaries become usable only after their own
obligations discharge in the same run; every run re-proves, no
cache. Records flatten to field variables; contract matches
become `ite`; literal-coefficient multiplication is native
QF_LIA. Scripts (`given`) are never proof facts.

## Findings

- CAN4304 unproven: the negation is satisfiable, with input
  countermodels read back via `get-value` (e.g. max+1 reports
  its witnesses). Against a weak summary the finding is marked
  modulo callee assumptions, never a demonstrated execution.
- CAN4305 inconclusive: timeout (30s per query), `unknown`,
  unrecognized responses, converter failure, or no solver
  binary. Never acceptance.
- CAN4306 inadmissible test: a row unsatisfiable under
  `requires`. Non-literal rows cannot be assessed and stay
  silent.

## Deliberate decisions

- The solver is external z3 over SMT-LIB text: integer
  completeness (`2*x==1` unsat) is load-bearing and not
  reimplemented. `CANLC_Z3` overrides discovery for tests.
- A failed callee cascades: its callers degrade to CAN4303
  (summary unavailable), never to false proofs.
- `ensures false` arms make paths vacuous exactly when the
  summary was proved: the outage pilot's unreachable handler
  verifies while its injected test still passes ordinary
  checks (proof and scripted evidence stay separate).
- Structural substitution (AST, never strings) instantiates
  summaries; records included.

## Verification

- 13 probes in compiler/contract_verify_test.go: max,
  validator, affine composition, integer-theory control,
  record equality, and outage separation verify; max+1,
  helper-body, weak-summary, and bad-precondition mutants
  refute with countermodels; bad rows reject; missing
  solver is inconclusive; response/model parsers pinned
  with canned I/O.
- Gates: suite ok, modcheck 25 OK, gramcheck OK, tsc clean.
