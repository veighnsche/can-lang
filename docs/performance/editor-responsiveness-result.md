# Editor responsiveness result — 2026-09-28

Status: complete. The targeted checker change and its correctness checks passed.
All 20 qualified editor-only trials completed; independent review reproduced the
saved summary and original report byte for byte and separately recalculated all
30 comparisons from raw samples. Both provisional 100 ms goals passed on flat-100.

| Interaction, flat-100 | Before (ms) | After (ms) | Saved (ms) | B/A | Goal |
|---|---:|---:|---:|---:|---|
| Warm completion | 497.662 | 15.607 | 482.055 | 0.03136 | ≤100 ms: passed |
| Valid edit → completion | 996.715 | 31.230 | 965.485 | 0.03133 | ≤100 ms: passed |

These are medians of six independent server-trial medians per variant,
about 96.9% less latency (about 32× faster) for each interaction. Every candidate
flat-100 trial median for both goals stayed below 100 ms. The before/after ranges
do not overlap; each of the three ABBA blocks shows the same large improvement.
These are provisional fixture-specific engineering goals. Invoice-compare still
needs 196.638 ms for warm completion and 393.101 ms after a valid edit, despite
large gains. This result does not establish a universal editor latency target.

## Demonstrated cause and change

`Catalogue.Inventory()` defensively deep-copies the embedded 299-operation
catalogue through JSON. The checker repeated that copy inside each intrinsic
metadata iteration, and collection classification repeated it for ordinary
authored symbols. A short diagnostic trace attributed about 97% of the original
completion duration to these copies. Duplicate resolution was about 2.6 ms,
so sharing resolved worlds would address little of the demonstrated delay.

`checkProgramForTarget` now reads one defensive operations copy and reuses it
for contract gathering and intrinsic receiver metadata. `collectionOperation`
rejects identities outside `can.std.collections@1::` before taking an inventory
copy. Positive lookup still performs the original I25/A05 filtering and serves
deeply isolated metadata. An exhaustive test covers all 299 operations, the 14
admitted operations, near-prefix/unknown identities and nested mutation isolation.

Three fresh Jev SystemOne consultations used equivalent facts and alternatives
with every explanatory passage rewritten and checked before sending. Advice was
local reuse, an identity index, then local reuse. The index disagreement was
investigated: it could avoid additional positive copies but would add a new
accessor/index and mutation obligations. The measured first remedy needed only
local reuse and a tested negative guard. All three requested exhaustive invariant
validation. The trace, source argument and tests establish the change; classifier
agreement is advice. Requests, responses and decision review remain in the pilot archive.

## Functional attribution

Final A/B trace exchanges are excluded from accepted latency measurements.
Stage durations are inclusive and process-monotonic; nested stages are never
added to their parents. Client/server reconciliation compares elapsed durations,
not absolute clocks from different processes.

| Trace interaction | Inventory copies A → B | Snapshot checks A/B | Resolver builds A/B | Client minus server A/B (ms) |
|---|---:|---|---|---|
| Warm completion | 451 → 10 | 1 / 1 | 2 / 2 | 0.125 / 0.111 |
| Valid edit → completion | 902 → 20 | 2 / 2 | 4 / 4 | 0.248 / 0.149 |

The ten remaining copies per flat-100 snapshot include resolver, error-registry
and catalogue-contract work. Full checking, diagnosis before the queued completion
and format/rename candidate validation still execute. No project snapshot cache
or asynchronous scheduler was introduced.

## Measurement contract and identity

`editor-responsiveness-v1` is separate from the historical twelve-slice run
`20260927T235706.371974Z` and its mixed syntax/semantic diagnostic median.
Flat-100 uses ABBAABBAABBA, six fresh processes per variant; flat-10 and the
maintained multi-file invoice-compare fixture each use ABBA, two per variant.
Every process retains seven measured batches and two excluded warmups; raw
request latencies are saved. Size 1000 and a new full-system audit were unnecessary
for this bounded first improvement. No p95 or tail-latency claim is made.

The client clock starts at the first JSON-RPC frame write and stops at matching
response receipt. Edit-to-completion includes both the versioned diagnosis and
queued completion. Semantic oracle checks happen after the clock stops;
initialization and oracle capture are excluded. Split syntax, semantic, warm,
edit, after-edit and recovery scenarios retain distinct records.

Each accepted trial passed the original ≥90% CPU-idle gate for 30 continuous
seconds with no detected competing build/test job. Independent review checked
every clean observation sequence, resets, elapsed quiet duration, labeled
interval, exact order, successful process exit and all 406 clean monitor
observations. This is cooperative isolation with sampled host observations,
not CPU reservation or proof that no unrelated activity occurred between samples.

Both A and B bundles were built once using the local pinned Bun 1.4.2 archive
and shared Go cache, with bounded Go workers. Production `runtime-check` verified
asset hashes and pins; the narrow closed-inventory adapter rejected unlisted
files, symlinks and nonregular entries and checked the launcher stamp.
`catalogue-check` currently has a stale 290-versus-299 assertion; it was not used
as successful qualification and its runtime source was not changed.

A and B share the final tracer, measurement harness, tests and initial checklist.
The captured patches differ only in the two selected checker changes. Source
fingerprints include the uncommitted tracked and new files against HEAD
`1c5ae7fc917c73f8efc486f468a0c716a8f8415c`; they are not bare commit identities.
Executable digests were stamped by each worker before and after its exchange;
fixture, oracle and harness identities remained compatible across every trial.

| Identity | A | B |
|---|---|---|
| Source SHA-256 | `6d928a79a67b68de095ca5a6f62e3313b97dd7a6e12756c2ba6637e40f143d3c` | `97a284c87287f65568a76dc6ff47857fe9abd8d239cf0576ad62e34e5539d9e4` |
| Executable SHA-256 | `70bfaa636212129a5b7eb9a7e9ac9f2d47886665bb79ec214a8b14764b091ea3` | `85d3868149facbeb147a8481169fc358f6597cce7fff488249e276d0a777c8bd` |

Measured harness SHA-256: `f6afbdb433faabb7d18136787672625c0ac48de2d82496b76a62b7e6e12acae0`.
After retirement, review corrected only the Markdown delimiter width in the
report renderer and added an assertion for all 13 columns. The measured harness
and original report remain in the immutable archive; clocks, validation, sampling
and compiler behavior were unchanged. The 40 editor-experiment tests passed again.
The tables below were regenerated from the independently verified saved summary.

## Complete comparisons

All values are milliseconds. Range and MAD describe independent trial medians,
not individual request tails. MAD is the median absolute deviation around the
variant median. Saved = A − B; ratio = B/A. Three decimal places are display
rounding; the archive retains full precision.

### flat-100

Trials per variant: 6.

| Scenario | A median | A range | A MAD | B median | B range | B MAD | Saved | B/A |
|---|---:|---|---:|---:|---|---:|---:|---:|
| completion-after-edit | 497.419 | 492.017–507.130 | 4.333 | 15.684 | 15.357–16.239 | 0.187 | 481.735 | 0.03153 |
| completion-unchanged | 497.662 | 491.927–503.174 | 4.037 | 15.607 | 15.322–16.285 | 0.213 | 482.055 | 0.03136 |
| diagnostics-semantic | 497.838 | 492.122–504.580 | 4.153 | 15.518 | 15.406–16.136 | 0.095 | 482.320 | 0.03117 |
| diagnostics-syntax | 0.532 | 0.518–0.569 | 0.009 | 0.511 | 0.453–0.717 | 0.054 | 0.021 | 0.96088 |
| edit-to-completion | 996.715 | 986.329–1005.684 | 7.389 | 31.230 | 30.682–32.549 | 0.348 | 965.485 | 0.03133 |
| editor.flat.completion | 501.456 | 495.601–503.575 | 1.786 | 16.020 | 15.774–16.249 | 0.180 | 485.436 | 0.03195 |
| editor.flat.definition | 498.819 | 495.671–503.887 | 1.876 | 15.889 | 15.646–16.119 | 0.086 | 482.930 | 0.03185 |
| editor.flat.diagnostics | 0.686 | 0.579–0.876 | 0.091 | 0.604 | 0.541–0.935 | 0.042 | 0.082 | 0.88011 |
| editor.flat.formatting | 1001.841 | 992.163–1009.294 | 4.398 | 32.475 | 31.913–33.483 | 0.285 | 969.366 | 0.03241 |
| editor.flat.hover | 500.336 | 495.791–503.842 | 2.038 | 15.737 | 15.552–16.132 | 0.180 | 484.600 | 0.03145 |
| editor.flat.rename | 1002.795 | 993.134–1008.959 | 4.640 | 31.742 | 31.269–32.652 | 0.379 | 971.053 | 0.03165 |
| recovery-to-completion | 996.459 | 985.333–1011.715 | 5.424 | 31.296 | 30.855–32.263 | 0.392 | 965.163 | 0.03141 |

### flat-10

Trials per variant: 2.

| Scenario | A median | A range | A MAD | B median | B range | B MAD | Saved | B/A |
|---|---:|---|---:|---:|---|---:|---:|---:|
| completion-after-edit | 298.583 | 298.511–298.655 | 0.072 | 14.000 | 13.900–14.100 | 0.100 | 284.583 | 0.04689 |
| completion-unchanged | 296.739 | 296.386–297.092 | 0.353 | 14.129 | 13.989–14.270 | 0.140 | 282.609 | 0.04762 |
| diagnostics-semantic | 295.793 | 295.198–296.388 | 0.595 | 14.013 | 13.999–14.026 | 0.013 | 281.780 | 0.04737 |
| diagnostics-syntax | 0.297 | 0.294–0.299 | 0.002 | 0.227 | 0.222–0.233 | 0.005 | 0.069 | 0.76674 |
| edit-to-completion | 592.156 | 591.622–592.691 | 0.534 | 27.835 | 27.539–28.131 | 0.296 | 564.321 | 0.04701 |
| editor.flat.completion | 301.055 | 298.727–303.383 | 2.328 | 14.495 | 14.476–14.513 | 0.019 | 286.560 | 0.04815 |
| editor.flat.definition | 301.448 | 299.084–303.812 | 2.364 | 14.419 | 14.341–14.497 | 0.078 | 287.029 | 0.04783 |
| editor.flat.diagnostics | 0.499 | 0.473–0.525 | 0.026 | 0.320 | 0.256–0.385 | 0.065 | 0.178 | 0.64222 |
| editor.flat.formatting | 603.325 | 598.282–608.368 | 5.043 | 28.773 | 28.449–29.097 | 0.324 | 574.552 | 0.04769 |
| editor.flat.hover | 301.250 | 298.598–303.901 | 2.651 | 14.277 | 14.210–14.345 | 0.068 | 286.972 | 0.04739 |
| editor.flat.rename | 605.320 | 600.814–609.826 | 4.506 | 28.489 | 28.477–28.501 | 0.012 | 576.831 | 0.04706 |
| recovery-to-completion | 593.266 | 592.878–593.654 | 0.388 | 28.276 | 28.025–28.526 | 0.251 | 564.990 | 0.04766 |

### invoice-compare

Trials per variant: 2.

| Scenario | A median | A range | A MAD | B median | B range | B MAD | Saved | B/A |
|---|---:|---|---:|---:|---|---:|---:|---:|
| completion-after-edit | 1084.156 | 1077.378–1090.934 | 6.778 | 196.990 | 196.467–197.512 | 0.522 | 887.167 | 0.18170 |
| completion-unchanged | 1081.252 | 1075.938–1086.565 | 5.314 | 196.638 | 196.251–197.025 | 0.387 | 884.613 | 0.18186 |
| diagnostics-semantic | 1084.012 | 1078.126–1089.899 | 5.887 | 197.672 | 197.326–198.017 | 0.346 | 886.341 | 0.18235 |
| diagnostics-syntax | 2.008 | 1.984–2.031 | 0.023 | 2.063 | 1.954–2.173 | 0.110 | -0.056 | 1.02780 |
| edit-to-completion | 2169.654 | 2156.305–2183.003 | 13.349 | 393.101 | 392.213–393.990 | 0.888 | 1776.552 | 0.18118 |
| recovery-to-completion | 2170.035 | 2155.291–2184.779 | 14.744 | 393.615 | 393.378–393.853 | 0.238 | 1776.419 | 0.18139 |

Hover, definition, formatting and rename all improve substantially at both flat
sizes; candidate-overlay validation remains covered by behavior tests. Semantic
feedback and invalid-to-valid recovery improve at all three fixtures. The original
`editor.flat.diagnostics` neighbor retains its alternating cheap syntax/expensive
semantic mix and is not used to infer semantic response time. Invoice syntax
diagnosis is 0.056 ms slower in the two-trial median, with overlapping ranges;
this small noisy difference does not establish a repeatable regression.

## Correctness and storage

All trial workers passed exact full-array completion equivalence, flat authored
signature/kind/provenance checks, version-correct erroneous syntax and clean valid
edits, recovery, neighboring query/edit oracles and actual LSP process exits.

Independent bounded correctness checks passed before measurement: 86 Python
harness/driver/storage/runner tests; 87 LSP behavior, nine snapshot, four overlay,
eight trace and one exhaustive collection-invariant Go tests; Go vet; gofmt;
and `bun run check:runtime`. Muse had already run the full 288-test checker suite
once successfully. No authored runtime TypeScript changed. Successful broad
checks were not repeated after the documentation-only integration.

The coordinator and supervisor exited successfully. Both production files match
the saved candidate restoration digests. All three task-owned runs have cleaned
ownership markers and no source/work/private-cache scratch. No managed worktree
was created or attached. Peak observed final scratch was 214,789,615 bytes below
the 2 GiB cap; combined traces are 978,954 bytes below 32 MiB. The final archive
is 204,741 bytes (1,690,173 uncompressed), below 10 MiB, with valid CRCs.
Evidence-only temporary extraction was immediately scoped and removed. Shared
Go caches and the preexisting pinned Bun archive remain available. No user app
was controlled or stopped, and no power setting changed.

Local compact evidence (ignored execution results, not required to build):

- [Final evidence archive](../../.performance/editor-20260928-final/evidence.zip): manifest, all raw samples/oracles, isolation, A/B patches, traces, qualification, independent checks and original generated summary/report.
- [Independent final review](../../.performance/editor-20260928-final/independent-review.json): separate arithmetic, gate, identity, goal, restoration and cleanup checks; archive SHA-256 `a4ae6f0f18592feeae0ad3c6a42a2674e94128d0ac831d79c17adc30ad61047f`.
- [Pilot diagnosis and consultations](../../.performance/editor-20260928b/evidence.zip): 64,408-byte retired diagnostic archive, including all three Jev requests/responses, disagreement review, source reconstruction, Muse completion evidence and recovery transition.
- [Implementation checklist](editor-responsiveness-tasks.md) and [original investigation plan](editor-responsiveness-plan.md).
