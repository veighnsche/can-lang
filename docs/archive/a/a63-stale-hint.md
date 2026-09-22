# a63: stale-arm nesting hint (diagnostic fix 2 of 2)

When an `on <kind>` arm is indented under the wrong
(nested) match, the prover correctly reports the parsed
tree but the pair reads contradictory: CAN4101
"non-exhaustive match, missing K" on the outer match plus
CAN4102 "stale match arm K" on the inner. The observed
B15 case attached an outer error arm to an inner total
call. Both diags stay; the stale one gains a pointer to
the likely cause.

Scope (compiler only, no .can changes):

1. `verifyExhaustiveAll` (eval.go): thread the ancestor
   match stack through the walk. Each ancestor records
   its line and missing-kind set (unknown-callee
   matches per a62 contribute nothing and are skipped).
   A stale-kind report checks the stack: on a hit,
   append `; enclosing match at line L is missing this
   outcome: check that this on-arm is attached to the
   intended match`. Genuinely independent matches never
   share a stack, so no false hint.
2. `proofDiag` (lsp.go): kind extraction takes the first
   whitespace-delimited field after the `stale match
   arm ` marker, so the hint suffix cannot corrupt the
   squiggle token. Message-shape mirror comment updated
   on both sides.

Out of scope: cascade suppression (a62); any
exhaustiveness rule change (the pair remains two
errors, correctly).

## Rollback

`git checkout -- compiler/eval.go compiler/lsp.go` plus
delete `compiler/diag_hint_test.go`.

## Test plan

- Probe first (red): temp client over KNOWN foreign
  wrappers (`std__hex__decode` failable outer,
  `std__hex__encode` total inner, real text.can,
  scripted givens) with the error arm misattached to
  the inner match: assert the CAN4102 message contains
  the enclosing-match hint. Control: same two calls as
  sibling (non-nested) matches, outer missing the arm
  and inner carrying a genuinely stale one: assert NO
  hint on the stale diag. (New
  `compiler/diag_hint_test.go`.)
- `go test -count=1 ./...`, `go run ./tools/modcheck`,
  `go run ./tools/gramcheck`.
