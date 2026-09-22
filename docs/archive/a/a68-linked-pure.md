# a68: runLinkedPure (verdict B, a67 §1.3)

Explicitly-selected linked-pure execution for committed
Go tests. No `.can` syntax, no `given` change, no
coverage credit. Spec lives in a67 §1.3; this slice is
mechanism only.

1. `Ctx.Linked` mode flag (eval.go). In the foreign-call
   path, when set and the callee resolves to any
   `prog.Fns` body: dispatch via `evLocalCall`
   (file-agnostic already; `FnFile`/`localCallee`
   untouched, identity preserved). `Given` never
   consulted; no script fallback. Unknown callee at
   runtime is an internal refusal (admission below
   excludes it first).
2. Admission walk before execution: BFS from root over
   `walkCalls`. Refuse on externs (`prog.Externs`),
   store ops (`isStoreOp`), non-empty `Effects`,
   unknown names, and anything not CAN source or an
   admitted deterministic kernel (`isBytesKernel`,
   `isDecParts`). All branches inspected, not just the
   exercised one. Cycles terminate the walk via
   visited set; cross-module cycles already fail the
   precondition; runtime `Depth` bound stays.
3. `runLinkedPure(t, files, root, rev, args, expect)`:
   parse explicit set only, `checkProgram` must be
   fully clean first, root rev must match, args parsed
   via `parseSmall` and bound like `runTestValue`,
   fresh store + throwaway coverage per vector,
   compare actual vs expected with `vEq` (errors by
   kind + structural payload, mirroring `runTest`).
   Modeling/domain/resource failure fails the test.
4. Effects-free guarantee is static (walk); kernels
   keep executing as today.

Out of scope: contracts slices (§2.3 identity design
may proceed on paper in parallel); any `given`/4107
relaxation; mixed real/scripted mode.

## Rollback

`git checkout -- compiler/eval.go compiler/check.go`
plus delete `compiler/linked_test.go`.

## Test plan

- Probe first (red): `compiler/linked_test.go`.
  Falsifier (verdict): leaf returns `""`, middle
  relays through `given`, client scripts `"A"`;
  `runLinkedPure(middle__copy@1, "A")` must report
  actual `""` vs expected `"A"`, both module orders;
  correct-leaf control passes. Middle carries unit
  `given` tables (ordinary checks green), proving no
  script fallback.
- Refusals: reachable extern, state op, effectful fn,
  unknown call — each refused before execution.
- `go test -count=1 ./...`, `go run ./tools/modcheck`,
  `go run ./tools/gramcheck`.
