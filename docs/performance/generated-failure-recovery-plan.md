# Closed checked-recovery lowering to native TypeScript

Status: bounded production design for the active performance campaign. The
ordered assignments are in [generated-failure-recovery-tasks.md](generated-failure-recovery-tasks.md).
The previous investigation and source evidence are in
[generated-failure-recovery-investigation.md](generated-failure-recovery-investigation.md).

## Problem and evidence

Packet 13 left a large `generated.failure-recovery` gap: the qualified
size-100 busy-host observation was Can 208.354 µs versus native 20.792 µs.
The prior record showed Can 108.375 µs versus native 18.228 µs even though
recovery production code had not changed. These are not a recovery before/after
comparison or an attribution of cost to any one component.

The real checked `gallery::is_positive` function is a closed one-`int`-input,
`bool`-result completion match. It calls canonical `checks::require` with
`number > 0` and a literal reason, returns `false` on exact `checks::failed`,
and `true` on `ok`. Its final target is `$canChecks.require`; its region has no
body statements, declared escaping errors or escapes. The emitted **TypeScript**
currently awaits `$canInvoke`, creates an authenticated Completion and, on a
false condition, a reason record and domain failure before recovering to bool.
The fixed JavaScript driver invokes the exported function once per element,
outside any Can array map/fold. Removing per-element array callback work would
therefore miss this endpoint.

Source-visible work justifies trying a finite lowering, but no speedup is
guaranteed. Three fresh equivalent Jev Choice consultations selected a closed
native route; one had diffuse probabilities. Their saved requests, responses
and reconciliation are advice, not source proof. The runtime false branch
advances a global failure occurrence ID even when the domain carrier is
consumed; later diagnostics expose those numbers.

## Exact proof and generated route

Build a separate private `ClosedRecoveryProof` from the actual concrete
`ProgramFunction`, final merged binding table, checked full region and source.
Never use a benchmark name, source text, exported name, or inferred function
shape. Admit only Bun-authored functions with one `int` input, `bool` result,
no escaping errors/escapes/body statements, and one same-region terminal
completion match. Its only invocation step must be the canonical
`can.std.checks@1::require` bound exactly to `$canChecks.require`, with the
exact checked contract and `checks::failed` error identity. It may have the
checker's positional `Prepare` locals only when each original value is pure,
typed, evaluated once in source order, and each argument is the matching
same-type local at the same position. The first value is precisely an `int`
input `>` an integer literal; the second is a string literal. Permit no
callees, receiver/native/array/asset/SQL/HTTP/action/resource/coordination
step, effectful expression, extra call, payload binding, reordered preparation
or unmatched error arm. Require exactly two arms: the named domain failure
returns literal `false`, and `ok` returns literal `true`; a standard failure
is not consumed. Every unknown or malformed shape declines to ordinary
lowering. Template-substituted defining source/spans must remain exact.

For a proved Bun function, prepend a private fast branch to the unchanged
async authored function in generated TypeScript. Admit only when assertion
context is `undefined`, no ambient owner is active, and the host input is a
primitive `bigint`; evaluate the native `>` comparison exactly once. On
`false`, advance the same private global occurrence counter **once** before
returning a branded `$canSuccess(false)`; on `true`, return
`$canSuccess(true)` without advancing it. No reason record, domain token,
extra Promise chain, `then` access or mutable payload is created in this
closed case. Keep the normal async body byte-for-byte in behavior for every
declined path: defined assertion context or fixtures, active owner,
wrong-host input, browser output and all unproved functions. In particular,
do not bypass the original first-boundary origin or standard-failure
forwarding on a decline. A private Bun owner-state query is an adapter for
admission only; do not add a Can API, public cache or general effect system.

The hot branch uses source-map comments for its exact defining function and
predicate spans. It does not initialize/assign `$canOrigin` or allocate an
origin object/array on success. Any cold synchronous fault is caught and
boxed with the original region source/span; the normal body retains its
existing mapped origins. Keep the function's Promise-of-Completion interface
and direct native JS comparison in its generated TypeScript.

If proving the real checked fixture needs a broader grammar, special casing,
or weakened guards, stop that production edit, release the affected writers,
save the concrete IR obstruction and move to another supported lane. This
design does not authorize a general checked-operation rewrite or a series of
small wrapper changes.

## Verification and comparison

Compiler tests must show real checked/emitted execution for at least two
independently named eligible functions with differing integer thresholds and
reasons, plus the real gallery function. Check true/false results,
authenticated/boxed Promise-visible carriers, primitive/hostile input fallback,
assertion fixtures/context, owner/cancellation route, browser exclusion,
exact source mappings and cold first-boundary origin. Check that each recovered
false consumes exactly one occurrence ID and each true consumes none using a
fresh isolated process; subsequent visible failure IDs must not shift. Include
meaningful proof negatives: payload-reading arm, standard arm, extra/effectful
argument, wrong target/error, extra statement/call, wrong type, and reordered
or foreign preparation. Existing positive checks must still emit their normal
checked route.

Run focused Go checks with installed Bun/shared Go cache and bounded
`GOMAXPROCS=2`, `-p=2`. If authored runtime TypeScript changes, run
`bun run lint:fix:runtime`, `bun run format:runtime`, `bun run check:runtime`
and relevant tests. Use one registered, bounded reused generated graph for
strict TS/mappings, the existing 24 complete emitted/native oracles and
14 fresh saved bindings; ensure actual checkout-runtime linkage, exact
source/module/driver/tool/dependency identities, cleanup and no retained
source/dependency copies. Do not alter the fixed drivers, frozen auxiliary
tools, prior raw records or unrelated production sources.

The immutable packet-13 after record may serve as before **only if** all
applicable producer, fixture, runtime, driver, module, tool, dependency and
control identities are qualified against the current pre-edit checkout.
Otherwise the comparison is unavailable; do not invent equality or run a
replacement baseline. If qualified and all correctness gates pass, authorize
one unchanged-method after record: six fresh sequential size-100 driver
processes, one iteration, two excluded warmups and seven accepted batches
per case, all 24 finite complete controls, 300-second sampling cap, 64 MiB
and 500-file owned scratch bound. Retire process groups then graph/scratch.
Compare median of six trial medians, range, MAD, absolute recovery cost and
Can/native ratio, plus all unaffected controls. Label busy-host/fixed-order
uncertainty; no p95, heap, isolated-causal or guaranteed multiplier claim.
Failure means unavailable comparison and safe cleanup, never a resample,
driver repair or partial merge.
