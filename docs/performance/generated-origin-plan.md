# Lazy static origin reuse in generated functions

Status: designed by Codex on 2026-09-28; implementation assigned to Muse by
[G13-G18](generated-origin-tasks.md). Previous runtime checkpoints are committed
as `7e0d5641` and `a7130e42`; the latter passed 53 independent contracts and 24
fresh generated/native validation cases. No current timing benefit is claimed.

## Source evidence and proposed boundary

`compiler/internal/emit/regions.go` builds an object and invocation array at
function entry and at marked expressions/calls. `origin`, `mark`, and `markNode`
repeat compile-time metadata when the ordinary function is called repeatedly.
Actual generated doubled/generic/captured maps, fold and frequency invoke these
functions per element. Removing unused assertion dispatch did not remove these
objects from the emitted callback path.

For non-self-tail functions, source/span/region invocation are static. Real
production and assertion emission sets `SourceID`; `markNode` may instead use a
substituted node's definition source. `outcomeHandler` temporarily switches
`e.region`, so the active region ID belongs in the key. Proven self-tail regions
append a dynamic `"step:"+counter` and stay outside this change entirely.

Runtime `FailureOrigin` is readonly. `failure.freezeOrigin` and domain capture
copy the object and invocation array into each occurrence. Inspection found no
origin mutation or reference-identity contract. This permits sharing only the
immutable compiler metadata; failures, contexts, owners and user values stay
fresh. The existing defensive capture copies remain essential and untouched.

Enable finite private caching only while `RegionEmitter.Function` emits a mapped
function (`SourceID` present) without self-tail lowering. Intern each full
source/start/end/active-region-ID tuple during compilation. At its existing use
site lazily initialize a deeply frozen origin and reuse it thereafter. Keep
marked assignments and paired mapping tokens in their original positions.

Each cache slot is a uniquely named module-private typed `var`, emitted after the
returned async function. The text must still start with the function declaration:
`program_modules.go` adds `export ` directly, and assertion/direct Function callers
also concatenate the result. Hoisted `var` bindings begin undefined even through
cyclic module evaluation; a `let` binding after the function could have a TDZ.
No extra imports, exported API or assembly-wide pooling are needed. Keep slot
types readonly and nullable until initialized, using ordinary native nullish
assignment and `Object.freeze` for the object and invocation array.

These slots retain at most the finite emitted site inventory. Uncalled functions
and unreached branches allocate no origin objects. They retain no inputs or
context data. First-use freezing and additional module bindings have costs;
removing repeated construction is a source conclusion, not a measured latency
claim. An eager module pool would pay allocation for all cold functions/branches;
it is not selected.

## Correctness boundaries

The cache must reset for each Function and be inactive after success or error,
including reuse for `$canActual`, `$canExpected` and later configure-only native
or injection paths. Nested handler origins belong to the enclosing finite cache
but keep their own source/span/region identity. Use deterministic identifiers
derived from the generated function namespace with a stable local slot index;
avoid collisions with existing generated locals or other functions.

Empty `SourceID`, all functions containing a self-tail proof and configure-only
wrapper/fetch/judge/LLM/entry machinery retain current literal construction.
The rest of those emitters are later candidates, not prerequisites for this
bounded fix. No async calling, completion admission, callable capture, assertion
frames, owner/lease/cancellation, native collection algorithm, error occurrence
or runtime defensive-copy changes are authorized here.

Source maps must still identify the original operation and expression, including
definition-file offsets after template substitution. Generated coordinates may
change with the new expression width; mapping extraction and the validator must
accept them. Do not relocate tokens to cache declarations or fix mappings by
removing location coverage. Dynamic tail step origins remain distinct and valid.
Existing tests may depend on old literal spellings: update only such layout
assertions, retaining their source/span/order/occurrence checks. There is no
requirement to preserve old generated layouts or ABI, but actual semantics bind.

## Consultation and acceptance

Three fresh requests and responses are saved under
`.performance/performance-push-20260928/generated-origin-{request,response}-[1-3].json`.
All explanatory context, questions and criteria were reworded; structured facts
and technical samples remain equivalent. `generated-origin-equivalence.json`
records the pre-dispatch check. All three selected lazy sites and Function-only
scope (probability 1.0); no disagreement arose. This is advice, not correctness
proof or a guarantee of bias removal. Independent inspection established the
scope constraints above; tests and actual output review still decide acceptance.
The classifier API follows the live [TypeSafe API](https://docs.typesafe.ai/api)
and [Choice contract](https://docs.typesafe.ai/primitives/choice).

Required checks cover repeated/concurrent calls reusing immutable metadata with
fresh standard/domain occurrences; exact authored and substituted spans and
first boundary origins; handler-region separation; deterministic state reset and
failed-emission cleanup; unchanged dynamic tail steps; production/browser/assertion
exports, strict TypeScript and validated source maps. Fresh actual generated
doubled/generic/captured map, fold and frequency/native endpoint oracles must pass.
Use existing bounded validation preparation with zero-byte path inventory replaced
by the actual checkout runtime link before execution; no placeholder execution,
source copies or retained bundles. Keep shared Go cache and installed Bun.

The campaign quiet waiver still applies only here, and no timing is required to
accept a proven allocation removal with correctness. Any later timing must be
labelled busy-host/non-isolated. The remaining callable, startup and codec queue
continues after this packet; its completion cannot end the campaign.
