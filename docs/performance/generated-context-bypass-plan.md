# Bypass unused assertion adoption at proven authored call sites

Production target: compiler/internal/emit/regions.go InvocationStep lowering.
Every Bun invocation constructs an assertion-context callback, computes a private
callable receipt and enters async callContext even when assertion context is
undefined. runtime/assert/context.ts:285 returns run(undefined) in that branch;
only defined context creates a child frame and suspends/resumes the parent.
runtime/callable.ts:51 only reads the private receipt WeakMap. The outer generic
invoke in runtime/completion.ts still supplies carrier admission, caught failures
and first locatedCompletion boundary. Keep it and its thunk unchanged.

Existing assembly authoredProof plus provenAuthoredBinding positively identifies
checked authored native async Function declarations and their exact resolved
bindings. Merely matching a name or a Promise type is insufficient. Current
unknown/native/foreign targets may return or throw synchronously and their async
wrappers can affect raw thenable assimilation. No universal bypass is supported.

Eligibility: Bun only; plain direct call with nil Callee,Native,Array,Asset,
Fixtures,SQL,FormAction,JSONFetch,Action; exact authored identity AND resolved
binding proof. Preserve any receiver/argument semantics already lowered by the
ordinary direct path; the proof must not classify indirect callable values.
Inside the original invoke thunk emit a strict $canContext === undefined branch:
that branch invokes the original authored target with the same lowered arguments
and undefined trailing context; the other branch is the original callContext,
child-context callback and receipt expression. Construct/evaluate assertion arrow
and receipt only in the defined branch. No target/argument reordering or repeats.
Unproven or specialized calls retain identical original lowering. Browser retains
explicit owner/context lowering. configure ExpressionEmitter.Call already has
no callContext wrapper and must stay unchanged. Runtime files stay unchanged.

Preserve branded boxed payloads/thenable non-assimilation, caught/standard/domain
failure occurrence freshness and exact first boundary, origins/substitution spans,
paired mapping markers, callback/fixture ordering, owner/cancellation/lease frames
and context propagation. Defined-context scheduling/receipts remain unchanged;
absent-context proof removes redundant promise adoption rather than promising
identical microtask counts. Test delayed calls and owned work for required behavior.
No compiler-state/proof mutation or export/layout compatibility machinery.

Three fresh fully reworded equivalent Jev requests/responses under .performance/
performance-push-20260928/generated-context-bypass-{request,response}-[1-3].json
choose guarded_authored with1/1/.99 probability. Equivalence/reconciliation saved;
agreement is advice only. Tests and actual lowering establish correctness.

Source-supported removal is one assertion callback closure, private receipt lookup
and callContext async adoption at eligible absent-context calls. Outer invoke,
assertion-enabled behavior and generic native routes remain. No measured speedup
or heap claim; no new timings/harness capability. Auxiliary tools are frozen.
Use existing native Go/Bun contracts and one reused bounded actual generated graph
for strictTS/mappings/24 oracles because emitted call bodies change. Retain exact
source/module/driver identities and explain changed modules. Clean all groups and
scratch, use installed Bun/shared Go cache. Muse implements production/tests from
the ordered checklist; Codex independently reviews/accepts/commits. Startup/native
async binding proofs and final twelve-slice exhaustion review remain separate.
