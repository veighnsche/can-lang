# A07 — W4 core iteration and aggregation qualification (2026-09-26)

Source: R05; W4; C-A. Owner: lane A (sole owner of legs W4.1, W4.2,
W4.4, W4.5; W4.3 bounded companion batch is F06-owned and out of
scope here). Machine record: [a07-legs.json](a07-legs.json).

Environment: macOS darwin-arm64 (MacBook Air), bun 1.4.2 pinned
(`CAN_BUN=/Users/vince/.bun/bin/bun`, hash-verified against the pinned
target inside the harness), go1.27.1. One process per measured leg;
single bounded 1M-relay leg, no sustained full-tilt load.

Corpus + harness (committed, A-owned):
`compiler/internal/emit/w4_a07_test.go` — the W4 iteration corpus
(state machine, input-driven relay, relay scan with single bulk
build, three step-60k fault shapes, spread-accumulation contrast,
three non-lowerable controls), a static lowering-shape test, and the
measured execution legs. Bulk-builder source resolution is pinned by
`TestBulkBuildersResolveFromSource` in
`compiler/internal/check/collections_test.go`.

## A05 routing completion (needed by W4.2)

The A05 catalogue merge admitted `collections::build_map/build_set`
with lowering task `A05`, but the checker only routed task `I25` to
collection specialization, so both builders were uncallable from Can
source (`declaration ... is not a generic source function`). A07
admits task `A05` alongside `I25` in `collectionOperation`
(`compiler/internal/check/collections.go`); only the two bulk
builders carry task `A05`, so no other operation changes routing.
Without this fix the relay-plus-bulk aggregation leg cannot execute;
with it, explicit-type-argument calls specialize to collection
contracts with the declared results and failure types.

## W4.1 — 100k-step state machine (PASS)

`machine` (3-state cycle, three relay sites, accumulator) and
`scan_sum` (input-driven relay over `int[]`) lower to native `while`
loops and complete exactly at 100/20k/100k:

- exact accumulators at every size (machine 100k = 199999;
  scan 100k = 5000050000), verified against independent oracles;
- lowered loop retains no per-step state: scalar machine heap delta
  at 100k and at 1M is ~0 (see legs JSON);
- time scales linearly (20k→100k well under the 10x reject bound;
  1M relays complete in tens of ms — one bounded leg).

Memory scoping (honest limit): the scan leg's driver-owned input
graph does not settle deterministically under JSC — large dropped
graphs are intermittently retained by conservative stack scanning
(probed: identical runs variously collect fully, partially, or not
at all), so scan retained bytes are recorded, not gated. Loop
flatness itself is proven by the scalar machine legs, which share
the exact loop shape and settle at ~0.

Bounded (countdown) and input-driven (length-terminated) relays both
qualify. Can has no non-terminating unbounded-relay form — every
relay must reach a base case — so no such claim is made.

## W4.2 — growing bulk aggregation (PASS)

`aggregate` relays a 100k-entry scan threading relay state, then
publishes one immutable map with a single `build_map` call:

- full 100k order identity: every output key/value pair equals the
  input order; the relay-threaded counter is verified by the base
  case returning `rows[0:0]` on drift, so a miscount would fail the
  order check rather than pass silently;
- retained-result growth is linear: the 20k→100k step is ~4x the
  100→20k step (linear predicts ~4.02, quadratic ~25), inside the
  committed [2,10] band; bulk time scales ~5x per 5x input;
- duplicate keys fail the whole build as domain
  `collections::key_exists` with no partial map (bulk_failure leg);
  empty input builds empty output through the relay base case.

Contrast (recorded, not gated): relay accumulation via array spread
(`grow`) measures superlinear time (~17x per 4x input over
200/800/3200), which is why the qualified aggregation shape builds
once with `build_map` instead of accumulating per step. No quadratic
history copying ships in the qualified path.

## W4.4 — non-lowerable recursion unchanged (PASS)

Post-processing recursion (`triangle`), mutual relays
(`ping`/`pong`), and a callable-held frame (`loop_excluded`) keep
nested calls: no `while` in their emission, exact small-scale results
(triangle(100) = 5050, ping(20) = 100, loop_excluded(50) = 0), and
exactly the A04-contracted notes — two mutual `not the enclosing
function` notes plus one `deferred completion (callable value)` note,
all `CAN-CHECK-NOT-LOWERED`. Lease/timer/drain/coordination/fixture
exclusions stay covered by the A04 checker fixtures; A07 confirms
the runtime behavior half (nested calls compute correctly).

## W4.5 — injected step-60k fault (PASS)

Three fault shapes, each followed by a success leg proving failure
containment (no overflow anywhere; sibling runs exact):

- standard fault (`fault_standard`): `arithmetic: integer division
  by zero`, step `step:60000`, occurrence ID present;
- loop-originated declared fault (`fault_originated`): domain
  `collections::key_absent` (concrete type + declaration identity),
  step `step:60000`, occurrence ID present — this leg covers the
  M8 step-index path for domain origination, previously probed for
  standard faults only;
- callee-created declared fault (`fault_declared`): domain
  `collections::key_absent` with identity and occurrence preserved,
  origin source `can:collections:map`. Domain failures keep their
  creation-site origin through every propagation (lowered or not);
  the loop catch only (re)creates standard failures, so no step is
  attached here. This is the language-wide rule, pinned positively,
  not a lowering gap.

## Q6 gate

No new syntax needed: every required W4 shape in
`compiler/internal/check/self_tail_coverage.md` executes as covered.
The callable-capturing-fold exclusion was not needed — aggregation
threads plain relay state plus one bulk build — so the recorded
capture-analysis refinement stays a path, not a trigger. A06 remains
INACTIVE.

## Handoffs

- F06 (W4.3 batch mirror): step/failure/occurrence contract is
  `step:<0-based iteration>` in the failure origin invocation path
  next to the region identity, occurrence ID on every failure,
  domain failures keep creation-site origins; corpus + driver are
  the executable reference.
- H10/H11 (IC1/IC2): W4.1/4.2/4.4/4.5 evidence complete; no
  unsupported iteration claims (unbounded reading and scan-memory
  scoping stated above).
- G04 (completion): no new iteration keywords — A06 inactive, C-A
  keyword set unchanged.
