# Performance system

The maintained [suite guide](../../tools/performance/README.md) documents one
runner with twelve sequential slices. All twelve have representative workloads;
coverage is broad, not exhaustive. Compiler speed and the execution speed of
generated TypeScript are measured separately.

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
together. No further builds or tests were run after the request to avoid load.

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
its 330 files are not part of the maintained benchmark source. No evidence was
deleted. The expanded-driver inventory records original paths and file hashes,
and the interrupted run preserves its original status. The archive also corrects
earlier claims about overwritten browser failure evidence.

New runner output uses unique directories under the ignored `.performance/`
directory. Commit maintained fixtures, drivers and concise reviewed summaries;
keep local run artifacts outside version control.
