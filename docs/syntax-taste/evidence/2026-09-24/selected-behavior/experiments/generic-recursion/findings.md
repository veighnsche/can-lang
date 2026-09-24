# Generic recursion and public composition probe

Run on 24 September 2026 at `fbd2a5614dbba660b085b6fec8aef4f5d902c051`.
The experiment reuses the review's public `identity` / `wrapper` contrast and
the selected behavior packet's stationary, expanding and nested examples.
Its [source fixtures](cases/), [runner](run.py), [full results](results.json),
and [type-flow trace](flow-model-results.json) are retained here.

## What the current compiler does

`run.py` builds the current `./compiler` parser and a temporary
`project.Load` → `check.CheckProgram` → `emit.AssertionModules` probe. It
removes the temporary Go runner after completion. All six Can files parsed.

| Fixture | Full `CheckProgram` | Assertion module emission |
| --- | --- | --- |
| Public `identity<T>` control | Accepted | Fails: `unknown emitted type` |
| Stationary public self call `first<T> → first<T>` | Accepted | Fails: `unknown emitted type` |
| Stationary public mutual calls `first<T> ↔ second<T>` | Rejected at `first`'s opaque call to `second` | Not reached |
| Expanding public mutual calls `a<T> → b<T[]> → a<T[]>` | Rejected at `a`'s opaque call to `b` | Not reached |
| Acyclic public `nested<T> → identity<box<T>>` | Rejected at `nested`'s opaque call to `identity` | Not reached |
| Same acyclic chain kept private | Accepted | Seven modules emitted |

The three rejected public cases currently receive the same broad opaque
cross-generic restriction; their diagnostics do not distinguish finite
mutual recursion from expanding recursion or safe acyclic nesting. The
accepted self case confirms the existing same-declaration exception. The
private chain confirms concrete template checking can specialize the nested
call. None of these checker outcomes tests the proposed SCC implementation,
which does not exist yet.

The positive public control exposed a separate implementation prerequisite:
its checked `Program.Model` contains one `types.Parameter` node (as does the
self case), while the private control contains zero. The failure arises in
`emitStateModule` → `stateBuilder.prerequisites` →
`NativeTypeDeclarations(program.Model.Types())`. That emitter handles concrete
type kinds but has no `types.Parameter` case. The production
`driver.stageProgram` calls the same `emit.ProgramModules` or
`emit.AssertionModules` path. A P03 implementation must keep the symbolic
declaration graph out of the emitted model in addition to checking the new
calls; adding a fake TypeScript `Parameter` alias would wrongly expose a
declaration-only proof as runtime output. This conclusion is grounded in the
minimal fixture and call path, not in a successful end-to-end public generic
build. The local `bin/canlc build` cannot run without a development
distribution (`CAN-DIST-UNBUNDLED`).

## Why the proposed rule distinguishes the cases

`flow_model.py` applies only the type argument transformations in these
fixtures. From `first<int>`, stationary mutual recursion repeats after the
two states `first<int>` and `second<int>`; an acyclic nested call stops at
`identity<box<int>>`. The expanding cycle visits `a<int>`, `b<int[]>`,
`a<int[]>`, `b<int[][]>`, and so on. The 12-step trace is illustrative, not a
compiler check or a proof from bounded exploration. Its unboundedness follows
from the written `[]` constructor being added on every `a → b → a` round.

An acyclic-only rule would reject the stationary mutual case despite finite
concrete instance identities. The selected SCC rule admits it only after
both public bodies and signatures validate together. Its internal-edge
restriction to bare caller parameters or closed types rejects the growing
edge while allowing the acyclic `box<T>` call after its callee proof is
committed. This limits compile-time specialization; it does not promise
runtime termination or stack safety.

## Exact reproduction and limits

From the repository root:

```sh
python3 docs/syntax-taste/evidence/2026-09-24/selected-behavior/experiments/generic-recursion/run.py
python3 docs/syntax-taste/evidence/2026-09-24/selected-behavior/experiments/generic-recursion/flow_model.py
```

`results.json` records the exact `go build`, `canlc parse`, checker and emitter
commands, statuses and diagnostics. The checker runner is saved as
[`check-runner.go.txt`](check-runner.go.txt). The experiment does not run
assertions in Bun, build a supported distribution, implement symbolic proof
reuse, or establish that the selected rule is sound in all generic programs.
