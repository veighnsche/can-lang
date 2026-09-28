# Performance system

The maintained [suite guide](../../tools/performance/README.md) documents one
runner with twelve sequential slices. All twelve have representative workloads;
coverage is broad, not exhaustive. Compiler speed and the execution speed of
generated TypeScript are measured separately.

## Current investigation

The [editor responsiveness plan](editor-responsiveness-plan.md) defines the first
focused performance improvement: trace completion latency, establish its cause,
protect semantic behavior, and verify a targeted fix. Plan preparation used
source inspection and the saved measured results; no new performance run started.

The later completed measured run contains all 104 cases across twelve sequential
slices. Its local run identifier is `20260927T235706.371974Z`; browser correctness
passed, but its twelve timing cases remain unresolved. PDF export now obtains
advisory grades automatically through TypeSafe Score. The sections below record
the earlier implementation-validation and cleanup state, not the status of that
later measured run. Current commands and grading behavior are in the suite guide.

## Implementation validation on 2026-09-27

The expanded inventory contains **104 cases**, up from 37:

| Slice | Initial | Expanded |
|---|---:|---:|
| Compiler | 7 | 15 |
| Assertions | 1 | 3 |
| Artifacts | 2 | 4 |
| Generated TypeScript | 4 | 24 |
| Runtime helpers | 5 | 12 |
| Codecs | 3 | 10 |
| Startup | 3 | 5 |
| Browser | 3 | 12 |
| Server | 2 | 4 |
| I/O | 2 | 6 |
| Editor | 4 | 6 |
| Application journeys | 1 | 3 |

All 104 cases passed individual driver checks. Those checks executed timed
workloads in quick and standard profiles to verify implementation and reporting.
They were not controlled performance measurements and do not establish a
baseline, a performance rating, or calibrated regression thresholds. Counts
include scenario and implementation variants, not percentages of feature
coverage.

The initial 37-case complete sequential smoke run passed. **The expanded combined
run remains incomplete:** the user stopped it during compiler distribution
preparation because the laptop was busy, before any of its twelve suite trials
started. Its coordinator and preparation processes were verified stopped. The
compiler development checks reused a previously pinned runtime distribution;
fresh current-source preparation and the expanded suite still need validation
together. That interrupted run has not been resumed. Subsequent storage-lifecycle
validation used focused correctness checks, not performance workloads.

Focused checks passed: 13 coordinator/isolation/evidence tests, 7 compiler-driver
tests, 5 runtime-driver tests, and 6 application-driver tests (31 total). Go helper
build checks and runtime lint, formatting and type checks passed. Independent
review found no material implementation blocker. The identified rejection-check
timing issue was corrected, and large application results now use unique files
to avoid stdout truncation and stale-result reuse.

Browser driver checks passed in Chromium, Firefox and WebKit, including standard
sampling. Initial restricted launches failed; later approved launches succeeded.
The separate output-truncation failure was fixed and its complete result was
verified. Browser timings cover callback/set-read completion, not actual paint,
event queue delay or INP.

## Measurement design and remaining work

Three Jev consultations informed the measurement design. Eight of nine decisions
agreed; the server-load disagreement was investigated against measurement
literature, retaining both independently arriving requests and fixed waiting
clients. This advice supports the design; it does not validate future results.
The runner preserves raw observations, workload contracts, source/dependency/tool
identity, correctness checks and independent-trial summaries, and rejects smoke
or incomplete evidence as a baseline.

A fresh expanded combined smoke run remains pending. Performance measurement
should happen later on a suitably quiet machine, explicitly requested by the
user. No measurement has been scheduled automatically.

Remaining coverage includes actual paint/INP, comprehensive memory profiling,
full product/database/authentication journeys, packaging/bundling, the complete
effect/resource assertion corpus, and calibrated regression thresholds.
Production currently does not support prepare-rename; it is documented as a gap,
not counted as a benchmark case. No Rust or WASM rewrite conclusion follows from
these validation checks.

## Local audit evidence

Detailed audit reports, experiments, profiles, logs, consultation requests and
responses, and raw validation results are retained locally under
`docs/performance/2026-09-27/`. That dated archive is deliberately Git-ignored;
its files are not part of the maintained benchmark source. The expanded-driver
inventory records original paths and file hashes,
and the interrupted run preserves its original status. The archive also corrects
earlier claims about overwritten browser failure evidence.

New runner output uses unique directories under the ignored `.performance/`
directory. Each completed or handled-interruption run keeps a compact
`evidence.zip` and ownership metadata. Source copies, dependencies, prepared
bundles, private build caches and child temporary files are removed after their
processes stop. Raw observations remain in the archive for validated comparisons.
Explicit diagnostic retention expires after seven days and is reclaimed on a
later run; abandoned owned scratch is also recovered then. Recovery skips active
or uncertain ownership. See the suite guide for the recovery command and limits.

The storage cleanup preserved historical result records in verified archives and
removed execution copies. Integration, failure-convention and host conformance
suites now own temporary caches, share builds within a suite, remove them on
completion, and recover eligible abandoned caches on later runs. Asset tests and
publication measurements also register temporary-directory cleanup.

Validation of these lifecycle changes used 35 small Python harness checks, the Go
cache-helper unit tests, runtime static checks and ten asset unit tests. It did
not run a benchmark, browser or combined smoke workload. Full integration
validation remains deferred while the laptop is busy.

Commit maintained fixtures, drivers and concise reviewed summaries; keep local
run artifacts outside version control.
