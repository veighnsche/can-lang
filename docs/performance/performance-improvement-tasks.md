# Continuing performance improvement checklist

Status: active. Stopping condition: exhaust supported fixes across the performance
queue, not one successful patch and not an arbitrary time/model-step cap.
Follow [the campaign design](performance-improvement-plan.md), AGENTS.md and the
completed [editor checkpoint](editor-responsiveness-result.md). Record real commands,
exits, results and blockers compactly. Muse executes every ready implementation
assignment below in dependency order in one sustained run; Codex resumes the queue
after each independently accepted checkpoint.

## Ownership

Muse implementation lane owns `compiler/internal/catalogue/catalogue.go`, its
targeted tests and checker consumers/tests listed below. During that run, only
Muse edits this checklist's implementation/progress entries and writes its two
read-only investigation notes. Codex owns the plan, evidence directories,
experiment supervision and independent integration. Do not edit tracing, harness,
runtime, emission, supervisor or publication sources in this first packet.

Current code/harness baseline: commit `318ed458`; baseline preparation must finish
and the coordinator must be waiting for candidate before Muse edits compiler files.
While accepted performance trials run, no source edits or competing build/tests.

## Lane 1 — narrow catalogue metadata access

- [x] K1 — prerequisites: frozen A prepared; design consultation complete.
  Files: `compiler/internal/catalogue/catalogue.go` and `catalogue_test.go` (or a
  focused new test file). Add the smallest identity/current-name/type/error/method
  lookup APIs and error-subset accessor used by K2-K4. Reuse validated indexes or
  build indexes during private load validation. Every returned nested value must
  remain independent. Do not expose mutable internals or copy the whole inventory
  just to obtain pins. Preserve explicit `Operation(name,target,revision)` rejection.
  Preserve nominal receiver method first-match ordering. Acceptance: exhaustive
  old-inventory-scan equality for each introduced lookup; absent/near identities;
  wrong explicit pins; nested slice/field mutation isolation; loader validation.
  Evidence: tests, actual exits and API rationale here.
- [x] K2 — prerequisite K1. Files: `collections.go`, `stream.go`, `browser.go`,
  `sql.go`, and applicable focused checker tests. Replace positive whole-inventory
  scans with owned identity lookups. Preserve all existing guards, task filters,
  accepted identities and nil results (SQL's former helper admits any found identity;
  do not invent a new SQL namespace restriction). Keep existing trace stage behavior.
  Acceptance: exhaustive equivalence to each original helper and nested ownership.
- [x] K3 — prerequisite K1. Files: `array.go`, `methods.go`, `infer.go`, applicable
  focused tests. Array pin metadata uses same-instance current operation access;
  all I21/receiver/arity/error checks remain. Scalar-string method admission stays
  before nominal catalogue and project methods, with I24/type-argument checks.
  Nominal method lookup matches old receiver identity/local-name/first-match rules.
  Catalogue constructor metadata preserves constructibility, kind and errors.
  Acceptance: array specialization/inference/reference errors and method/constructor
  precedence, signatures and negative cases; no accidental metadata sharing.
- [x] K4 — prerequisite K1. Files: `errors.go`, applicable error-registry tests.
  Seed catalogue error declarations from a copied error subset rather than the full
  inventory. All duplicate checks, owner/source/registry agreement and project error
  validation remain unchanged. Acceptance: existing error-registry semantics plus
  subset equality and mutation-isolation coverage from K1.

## Lane 2 — bounded correctness and handoff

- [x] K5 — prerequisites K1-K4. Run gofmt on changed Go files and bounded sequential
  catalogue/checker tests using shared Go cache, `GOMAXPROCS=2`, `-p=2`, `-count=1`
  and appropriate per-command timeouts. Run relevant LSP behavior/snapshot/overlay
  tests once and Go vet for changed packages. Avoid tests that build full distributions
  or unrelated broad builds. No authored runtime TS is permitted in this packet;
  do not run unnecessary runtime format fixes. Record exact commands/test counts/exits.
- [x] K6 — prerequisites K5. Review diff for unrelated changes, metadata aliasing,
  nil/unknown behavior, pin handling and first-match ordering. Do not commit, build
  a fresh distribution or start measurements. Leave all source changes reviewable
  and summarize completion/blockers in this file. Do not send coordinator candidate
  input: Codex alone owns acceptance and the transition to B.

## Lane 3 — next evidence investigations, read-only

- [x] I1 — prerequisite K6. Output only `docs/performance/generated-collection-investigation.md`.
  Inspect the saved generated/native measurements, fixture contracts, emitter paths,
  runtime collection adapters, completion/context/owner layers and representative
  emitted code obtainable without a broad build. Record concrete costs/call boundaries,
  native-equivalent opportunities and semantic obligations. Clearly separate proven
  source repetition from unprofiled timing attribution. Propose small next assignments
  and their file ownership/tests; no runtime/emitter implementation yet. Codex will
  supply any required three design consultations before promoting a proposal.
- [x] I2 — prerequisite I1. Output only `docs/performance/assertion-artifact-investigation.md`.
  Inspect current supervisor/launcher and output store/validation/CAS paths against
  saved timings. Account for intended root count/fresh process and fsync/integrity
  obligations. Name specific duplicate work candidates, tests and a bounded attribution
  plan, or explain why current evidence does not support a fix. No worker pooling,
  durability removal, publication/validation rewrite, benchmark or broad build.

## Codex integration and continuing queue

- [x] C1 — prerequisites K1-K6. Independently review actual diff and evidence,
  repair genuine issues through the same checklist/executor where practical. Confirm
  a compiler-wide semantics check and appropriate bounded runtime check before acceptance.
- [ ] C2 — optional catalogue timing, deferred by explicit user priority. C1 is
  independently accepted for correctness; no timed trial was accepted. Preserve
  the compact transition and qualifications. Any later comparison is sequential,
  bounded, explicitly non-isolated/busy-host under the campaign-only quiet waiver,
  with unchanged sampling/oracles and observed activity. Do not redesign the harness
  merely to remove a waiting prerequisite.
- [ ] C3 — prerequisite an actual completed comparison if C2 is resumed. Independently
  regenerate accepted records and report medians/ranges/MAD/savings/ratios honestly,
  with no p95. Catalogue commits may record correctness and avoided copying now;
  they must not claim an unmeasured speedup. Verify scratch retirement.
- [ ] C4 — prerequisites I1/I2 and independently reviewed source evidence, without
  waiting for C2/C3. Generated TypeScript execution is the current primary lane.
  Follow [the generated execution checklist](generated-execution-tasks.md) after
  the supporting design and three fresh equivalent Jev consultations. Muse owns
  every implementation, test, harness change and correction; Codex designs, reviews
  and independently verifies. Continue beyond the first small packet through
  supported callback/invocation/allocation and startup candidates. Repeat the cycle.
- [ ] C5 — all ready candidates resolved. Final independent source/evidence review
  across all queue lanes. Every deferred/rejected candidate needs a concrete reason.
  Complete only if no supported ready fix remains; otherwise continue. Verify compact
  evidence, clean owned scratch/worktree lifecycle and integration; retire continuation.

## Progress and completion evidence

Codex prepared this packet after source/evidence review and three fresh catalogue
design consultations. All three selected index/equivalence; no design disagreement
was hidden. No new performance conclusion has yet been measured.

## K1-K6 completion evidence (Muse, 2026-09-28)

Baseline: `318ed458` (frozen A); no coordinator input sent; no measurements,
distribution builds, installs, worktrees, commits, or runtime/emitter/supervisor
changes. Owned files only: `catalogue.go`, the eight listed checker consumers,
two new focused test files, and this checklist's progress entries.

K1 APIs (`catalogue.go`): `OperationByIdentity`, `CurrentOperation`,
`TypeByIdentity`, `ErrorByIdentity`, `ReceiverMethods`, `Errors`. Rationale:
positional indexes (`operationsByIdentity`, `typesByIdentity`,
`errorsByIdentity`, `receiverMethods`) built at the end of `validate()` after
uniqueness is established, so lookups observe validated data; receiver methods
keep inventory order for first-match resolution and exclude intrinsic receivers
exactly as the old scan did. Every return is a `clone()` deep copy; index
values are never exposed. `CurrentOperation` delegates to `Operation` with the
instance's own pins, leaving explicit target/revision rejection untouched.
New `catalogue_lookup_test.go`: exhaustive inventory-oracle equality for all
six accessors (incl. per-type receiver ordering), 14 absent/near identities,
wrong-pin rejection vs current-pin success, nested mutation isolation for all
return shapes, and loader validation (`load(source)` parity + retargeted
identity still fails with "identity mismatch").

K2: `collections.go` (prefix guard + I25/A05 filter kept, trace stage kept),
`stream.go` (3-identity switch kept), `browser.go` (prefix + 3-identity + T22
kept), `sql.go` (`sqlOperation` still admits any found identity, no new
namespace restriction). K3: `array.go` both sites use `CurrentOperation` with
all I21/receiver/arity/error checks intact; `methods.go` scalar path uses
`CurrentOperation("str."+local)` + I24 check with identical fall-through
(unique names make the old loop single-candidate), nominal path iterates
`ReceiverMethods` with unchanged local-name/first-match logic; `infer.go`
constructor metadata uses `TypeByIdentity` (record + constructible filter
kept) then `ErrorByIdentity`, equivalent to the old ordered loops under
validated identity uniqueness. K4: `errors.go` seeds from `Errors()`; all
duplicate/owner/registry checks unchanged.

New `catalogue_narrow_equivalence_test.go` (check package): verbatim-reference
exhaustive equivalence for stream/browser-state/sql helpers over all 299
operation identities + 8 negatives, task-drift guard for SQL, helper mutation
isolation, nominal first-match equivalence over all receiver x local-name
pairs, scalar I24 equivalence over all `str.` locals + negatives, constructor
metadata equivalence over all type/error identities + unknowns, I21 pin
equivalence. Pre-existing `TestCollectionOperationNamespaceInvariant` passes
unchanged against the migrated helper.

K5 (sequential, shared Go cache, `GOMAXPROCS=2 -p=2 -count=1`): gofmt on all 11
touched Go files clean (exit 0); catalogue focused
(`TestNarrowLookup|TestCompleteInventoryAndMirrors|TestClosedLookupAndOpacity`)
ok 0.315s; check focused (7 equivalence/invariant tests) 7/7 PASS ok 0.455s;
full `catalogue` package ok 0.288s; full `check` package ok 6.686s; LSP
(`TestG0[1-5]|TestLSP|TestServer|TestWire`) ok 2.124s; snapshot/overlay
(`TestApplyBrowserOverlay|TestCheckSnapshot|TestCleanRebuildAndSnapshotBinding|
TestPublicationRechecksSnapshotBeforeCurrent|TestStandardSnapshot`) driver ok
0.779s + check ok 0.243s; `go vet` on both changed packages exit 0. No
distribution-building or runtime-format commands run (no authored runtime TS).

K6: diff is 9 files (+121/-64) plus the 2 new test files; `grep Inventory()`
over the 8 owned checker files returns nothing. Remaining `Inventory()` uses
(`check/program.go` contract gathering, `resolve/symbols.go`,
`project/graph.go`) are outside this packet's ownership and intentionally
untouched. Reviewed for aliasing (all returns cloned), nil/unknown behavior
(false/empty/nil preserved), pin handling (explicit rejection intact), and
first-match ordering (index built in inventory order). No unrelated changes.
No coordinator candidate input sent; source left uncommitted for Codex C1.

## I1-I2 completion evidence (Muse, 2026-09-28)

I1 wrote only `docs/performance/generated-collection-investigation.md`
(read-only inspection: saved report medians, bench driver + fixtures,
emitter paths, runtime adapters, checked-in emitted sample; no build or
measurement). I2 wrote only
`docs/performance/assertion-artifact-investigation.md` (read-only
inspection of supervisor/launcher/output/CAS/validation sources plus
per-root distributions recomputed from saved `raw/assertions-001.json`
in a /tmp scratch dir, removed after each use; verified absent at end).
No runtime/emitter/supervisor/publication/harness edits, no benchmarks,
no broad builds. K1-K6 + I1-I2 all complete; C1-C5 remain Codex-owned and
the campaign continues after this packet.

## C1 independent acceptance (Codex, 2026-09-28)

Actual eleven-file Go change reviewed against the previous scans and loader
uniqueness. All admission/pin/first-match/constructibility/error-registry rules
remain; returned metadata is defensively cloned. Independently passed six
catalogue tests and nineteen checker tests covering exhaustive reference
equality, negative contracts, mutations, methods, constructors, array
inference and broken error registries. `bun run check:runtime`, `gofmt -l`
on changed files and `git diff --check` passed. Muse already ran full checker,
catalogue, LSP/snapshot/overlay and changed-package vet; those successful
broader checks were not repeated without a new failure. Compact commands and
source hashes: `.performance/performance-push-20260928/catalogue-c1-checks.json`
and `catalogue-c1-review.json`. Muse parent, child and owned group are retired;
temporary prompt absent. This accepts correctness for the comparison, not
a performance gain. I1/I2 notes are proposals awaiting independent source
and attribution review. C2-C5 and the campaign remain active.

## Updated user direction

The user waived quiet-host waiting for this campaign, prioritized more independent
fixes over catalogue timing, and selected generated/compiled TypeScript execution
as the main lane. No global defaults were changed; no quiet evidence is asserted.
Sol/Codex plans and verifies; Muse implements all source/test/harness changes and
corrections with explicit maximum effort from saved ordered checklists.

## Generated checkpoint and next packet

G1-G6 independently accepted: native completion literal and single map containment
snapshot, 25 independent focused tests plus runtime check and 24 real generated/
native validation cases; owner/prompt/execution scratch retired. Source-proven work
removal only, no timed savings claimed. Follow generated-callback-tasks.md G7-G12
next: no-context array callback bypass, success-proven private extraction and map/set
metadata reuse, relevant contracts/actual generated checks and continued source
investigation. General callContext, invoke, emitter ABI and ownership logic remain
outside that packet. C4 and final exhaustion review remain active.

## Continuing generated checkpoints (Codex)

G7-G12 accepted and committed `a7130e42`: 53 applicable contracts, runtime check,
24 actual generated/native validations and identity/cleanup review passed.
G13-G18 plus exact-metadata/evidence corrections accepted and committed `5b1889ee`:
14 applicable independent contracts, 24 fresh actual generated/native cases,
strict TS and matching module/driver hashes, then a final narrow ten-test origin
run with an actual emitted synthetic invocation-boundary oracle. Production stayed
unchanged during test-only corrections; all groups/prompts/scratch retired. Prior
evidence chronology discrepancy is explicitly preserved. Both checkpoints avoid
source-proven work; neither has a measured speedup number.

Next implementation: [checked authored forwarding tasks](generated-forwarding-tasks.md)
G19-G24, after [their design](generated-forwarding-plan.md) and three saved fresh
equivalent consultations. Muse owns implementation/integration; Codex independently
accepts and commits before continuing. Startup attribution, codec retention and all
twelve dispositions remain under C4. C5 stays unchecked until actual final exhaustion.


## Checked authored forwarding checkpoint

G19-G24 independently accepted and committed `4589122c`: only an exact checked
authored identity/resolved binding permits direct promise forwarding. Sixteen
independent tests, zero skips, 24 fresh generated/native oracles, strict TS and
matching source/module/driver identities passed; ownership and execution cleanup
verified. The checkpoint removes source-proven redundant async adoption, with no
measured speedup claim. [Forwarding acceptance](generated-forwarding-tasks.md)
records the review. Next: bounded attribution of actual generated startup loading
and initialization, separate from compilation; then the next supported remedy.
Per-site invoke proof, secondary numeric JSON evidence retention and final
independent twelve-slice exhaustion review remain active.
