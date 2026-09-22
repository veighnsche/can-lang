# a62: unknown-callee cascade suppression (diagnostic fix 1 of 2)

One root cause, one diagnostic. Today a `match call` on a
function that resolves nowhere emits the whole ladder:
CAN3001 (unknown, the cause) + CAN3101 (no given) or
CAN3112 stub-not-in-emits (given present) + CAN4102
stale-arm (the prover treats unknown as Ok-only) + CAN4200
per row (rows cannot run) + CAN4107 coverage fallout.
Every rung vanishes when the 3001 is fixed, so all but the
3001 are noise.

Scope (compiler only, no .can changes):

1. `calleeUnknown(prog, fname)` helper in check.go:
   not in `prog.Fns`, not in `prog.Externs`, not a
   store/decparts/bytes-kernel op. (Module-local externs
   ARE in `prog.Externs`, so they keep full checking;
   extern-from-another-module keeps its own 3001 plus
   given/proof, since the declared emits make those
   meaningful.)
2. `checkGiven`: skip match-calls with unknown callee.
   `checkCalls` owns the CAN3001.
3. `verifyExhaustiveAll` (eval.go): skip the want/got
   comparison for unknown callees, still walk arm RHS
   (nested matches keep their own proofs).
4. Typed `UnknownCallError` from eval's foreign-call path
   when the callee is unknown (before the uses check).
   `checkSem`'s test loop suppresses the CAN4200 for it
   but marks the fn failed, so `checkCoverage` stays
   silent too. CLI shares `checkSem`, so both surfaces
   agree.
5. `checkScriptConsistency` already skips unknown
   callees; `checkEmits`/`checkTypes` emit nothing for
   them. No change there.

Out of scope: the CAN4101+CAN4102 nesting hint (a63);
relaxing CAN4107 or `given` (chatbot items B/C).

## Rollback

`git checkout -- compiler/check.go compiler/eval.go
compiler/lsp.go` plus delete
`compiler/diag_unknown_test.go`.

## Test plan

- Probe first (red): temp module calling a truly-unknown
  fn, with and without given, both arms: assert CAN3001
  present and codes CAN3101/CAN3112/CAN4101/CAN4102/
  CAN4200/CAN4107 all absent (new
  `compiler/diag_unknown_test.go`).
- Existing presence tests (`TestDiagnoseUnknownCallee`,
  `TestDiagnoseMissingGiven`, `TestDiagnoseCallNotInUses`,
  bytes_b2 E5) keep passing unmodified: none pins the
  downstream rungs.
- `go test -count=1 ./...`, `go run ./tools/modcheck`,
  `go run ./tools/gramcheck`.
