# a69: contract grammar — requires/ensures parse + store (no verifier)

First contracts slice after the a67 design (Part 3 order:
grammar + AST now; §2.3 identity later; verifier last).
Parses and stores outcome-indexed contracts; proves,
checks, and emits nothing. No `.can` changes (temp
probe modules only).

1. `requires` block (single spelling, block only):
   predicate rows via `parseSmall`, stored as
   `FnDecl.Requires []*Small`. Bare block or duplicate
   block rejected.
2. `ensures` block: `on <Outcome> <bind>` arms
   (`Outcome` = `Ok` or dotted kind, `bind` a name).
   Each arm holds `Preds []*Small` (expression rows)
   plus `Matches []*Node` (Boolean match blocks via
   the existing `parseMatchArms`, nested included).
   Empty arm or duplicate block rejected.
3. Outcome names validated against `{Ok} ∪ fn.Emits`
   after the metadata loop (order-independent:
   `ensures` may precede `emits`). Unknown outcome is
   CAN1000/`CodeParse`.
4. Representation: `ContractArm{Outcome, Bind, Preds,
   Matches, Line}` on `FnDecl` (`Requires`,
   `Ensures`). Every other phase ignores the fields:
   no exhaustiveness, coverage, emit, or catalog
   effect. `given` inside a contract match fails at
   check time as today (no special case).

Out of scope: well-formedness beyond outcome names
(bool-ness, bound-name use), §2.3 identity, verifier,
emit, stdlib pilots.

## Rollback

`git checkout -- compiler/parse.go` plus delete
`compiler/contract_grammar_test.go`.

## Test plan

- Probe first (red): `compiler/contract_grammar_test.go`.
  Temp module with the `max` pilot shape (requires,
  two-clause ensures with match-disjunction) and the
  validator shape (two error arms): clean diagnose.
  Negative fixtures: unknown outcome, empty
  requires/ensures arm, duplicate blocks, ensures
  before emits (must still pass).
- `go test -count=1 ./...`, `go run ./tools/modcheck`,
  `go run ./tools/gramcheck`.
