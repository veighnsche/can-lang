# Can performance system

One runner prepares frozen inputs, executes all twelve suites **sequentially**,
and produces a combined report with raw evidence for every case. It never runs
suite drivers concurrently. Server clients and assertion workers inside a
workload have their own explicit concurrency, which is part of that workload.

```sh
# Show suite inventory.
bun run perf list

# Validate all twelve suites. Functional evidence, not a baseline.
bun run perf:smoke

# Measure every suite in order, using six independent process trials each.
bun run perf run

# Measure selected slices or a different fixture scale.
bun run perf run --suite generated --suite runtime --size 1000

# Use a chosen committed revision with the current measurement harness.
bun run perf run --revision HEAD --output /absolute/new/baseline
bun run perf run --working-tree --output /absolute/new/candidate
bun run perf compare /absolute/new/baseline /absolute/new/candidate
```

Python 3.12+, the repository's Go and Bun versions, installed repository
dependencies, and installed Playwright browsers are required for full coverage.
No command installs dependencies or browsers. Compiler-family preparation needs
the pinned Bun archive used by the production distribution builder. Set
`CAN_PERF_BUN_ARCHIVE` to the archive path; the local default is
`/tmp/bun-darwin-aarch64.zip`. Production distribution validation checks its hash.
The current compiler distribution target is macOS ARM64; do not assume full
cross-platform coverage from the runner's macOS/Linux quiet-host support.

For mandatory browser coverage, set
`CAN_PERF_BROWSER_ENGINES=chromium,firefox,webkit`. Without this setting the driver
records probe outcomes and runs available engines; unavailable engines are
reported, and no available engine blocks the browser suite. A requested missing
engine always blocks the suite. Browser timing is headless.

## Coverage and limits

All twelve slices have multiple representative workload paths. Coverage includes
real maintained projects, emitted language features and application contracts;
it does not claim exhaustive feature coverage or a universal performance score.

| Slice | Measured path | Main boundary or limit |
|---|---|---|
| Compiler | Seven synthetic phases plus load/check/emit/pipeline for utilities and the browser invoice-compare project | Flat projects scale to 10/100/1000 functions; maintained anchors stay fixed; phase scopes overlap |
| Assertions | Production supervisor executing flat roots and all utilities roots with one or four workers | Fresh bounded Bun workers; no complete provider/resource corpus claim |
| Artifacts | Production source-map encoding, mapped native validation, first publication and generation reuse | Bundling and release distribution packaging remain uncovered |
| Generated code | Emitted mapping, counting, arithmetic, folds, records, variants, recovery, generic/captured callables, Unicode and bytes, with native references | Native comparisons cover named tested endpoint contracts, not arbitrary ownership/failure behavior |
| Runtime | Collections, nominal records, owned captures, completion chains, thrown errors, resource lifetime and byte ownership | Immutable snapshots checked; no allocation profiler yet |
| Codecs | Exact bigint arrays, nested nominal Unicode records, encoding and measured rejection paths | Native JSON parsing is a component reference, not the full codec contract |
| Startup | Fresh Bun processes for minimal/runtime/emitted controls and an application in bundled and module layouts | Precompiled JavaScript; caches uncontrolled; includes correctness checks, output capture and exit |
| Browser | C02 property snapshots, keyboard input, state/reset and caret flows in each available engine | Callback/set-read timing; queue delay, layout, actual paint and INP are not measured |
| Server | GET and POST body/response adapters under bounded clients and scheduled arrivals | Separate same-host generator; no full server lifecycle, TLS or remote capacity claim |
| I/O | Runtime/native bounded text reads, binary write/read ownership and byte-cap rejection | OS-managed caches; no fsync, durability or cold-storage claim |
| Editor | Real LSP diagnostics, hover, definition, formatting, completion and safe rename | Matching document versions and applied rename checks; startup excluded; prepare-rename unsupported |
| Journeys | Serialized batch and valid/invalid invoice requests through real emitted record/map/fold programs | Representative business workflows; no database, authentication or complete product journey claim |

Compiler speed and emitted-program speed are separate outcomes. The generated
suite uses the production compiler during preparation, then measures execution
of the emitted TypeScript in Bun alongside handwritten native cases with the
same tested endpoint contracts. This includes the runtime calls made by generated
code. Attributing a change solely to code generation requires keeping runtime
behavior fixed or testing compatible combinations separately. These workloads
still do not establish efficient output across every language feature.

Exact boundaries and raw auxiliary metrics live in driver results. See the
[compiler fixture contract](fixtures/compiler/README.md),
[runtime fixture contract](fixtures/runtime/README.md), and
[application scope](drivers/apps-notes.md).

## Sampling and isolation

Quick produces three measured batches per case; standard produces seven.
`--iterations` controls workload operations per batch, and `--warmups` controls
excluded warmup batches. `--trials` controls fresh driver-process trials, which
are summarized using their batch medians. Defaults for measurement are standard,
six trials, one operation per batch, two warmup batches and size 100. Smoke uses
quick, one trial, one operation per batch, no warmup and size 10. Larger operation
counts help when timer overhead is material. There is no forced GC in headline
timings. Do not infer request tail percentiles from batch averages.

Each run holds the shared audit lock during preparation and execution. Normal
runs require 30 continuous seconds with at least 90% observed CPU idle and no
detected competing build/test jobs before each suite. Trials inspect competing
processes periodically and at completion. The observation is heuristic, not a
reservation of the machine: it cannot rule out every short-lived process,
thermal change, cache effect, VM interruption or background workload. Keep power
settings and host conditions consistent. Weakening idle/window requirements
marks the run exploratory; smoke bypasses the quiet gate and is always labeled
functional evidence. Neither can enter the comparison command.

Only one suite runs at a time, in the listed order. Preparation happens before
the suite loop, and driver timers exclude it. Failures, missing prerequisites,
interference or timeouts fail the entire invocation with a nonzero status.
Completed raw slices remain available, but incomplete runs cannot become
baselines. Child process groups are terminated on completion, failure or timeout.

## Evidence and comparisons

By default output goes into an ignored, unique `.performance/` directory. Existing
output directories are never overwritten. A run saves:

- `manifest.json`: source revision/content hashes, harness/dependency hashes,
  tool binaries/versions, host identity, settings, step outcomes and total wall time.
- `source/`: the exact source and dependency copy used for the run.
- `raw/`: preparation results, every driver-process result, and stdout/stderr.
- `isolation.jsonl`: lock, quiet-window and competing-process observations.
- `summary.json`: completed results only, with per-case independent-trial medians
  and dispersion. Warmups never enter the summary.
- `report.md`: the combined human-readable report with all individual cases,
  completion status, total run duration, preparation time and driver time.

Total run duration includes orchestration and quiet waits. It is not the sum of
application latencies, nor a score combining incompatible units. Overlapping
compiler phases also cannot be summed into pipeline time.

By default production source comes from `HEAD`; tracked local edits require
`--working-tree`. Untracked production files are excluded. The current harness
and its two Go helper packages are always overlaid onto the snapshot so one
instrument can test different revisions. Installed dependencies are copied and
checked for changes after the run. External dependency symlinks are rejected.
Child workloads receive an allowlist of local tool settings, not ambient API keys.

Comparison reloads and validates every expected raw trial, recomputes the
summary, and rejects missing/incorrect results, changed case inventories,
workload contracts, tools, host, harness, dependencies or measurement settings.
Reported median changes are descriptive. Automatic regression thresholds,
confidence claims and pass/fail performance budgets remain uncalibrated.
Confirm suspected improvements with repeated, counterbalanced revision runs.

No Rust/WASM conclusion follows from this harness alone. Any alternative needs
the same observable contracts, realistic boundary costs and representative
inputs before comparison.

## Harness validation

```sh
python3 -m unittest discover -s tools/performance -p 'test_*.py'
python3 tools/performance/drivers/test_runtime_driver.py
python3 tools/performance/drivers/apps_test.py
go test ./compiler/perfemit ./compiler/perfmeasure
bun run check:runtime
bun run perf:smoke
```

The regression tests cover serial sequencing, invalid evidence, raw-result
verification, contract drift, independent-trial summaries, lock contention,
timeout cleanup, and blocked/malformed driver requests. The final smoke command
executes real production paths across the entire system.
