# Performance driver protocol

The shared Python runner owns isolation, provenance, independent process trials, comparison and reports. Drivers own real workloads and correctness. No production implementation changes belong in this work.

Driver entrypoints are `drivers/compiler.py`, `drivers/runtime.py`, and `drivers/apps.py`.

Each supports:

```
python3 DRIVER prepare --repo ABS --work-dir ABS --output ABS --suites CSV --profile quick|standard
python3 DRIVER run --repo ABS --work-dir ABS --output ABS --suite ID --profile quick|standard --iterations N --warmups N --size N
```

Preparation may build Go helpers or emit fixtures, but is outside measurement. `work-dir` is unique to the family within a run and persists across its suites/trials. All generated artifacts belong there. Drivers must not install dependencies, alter tracked source, print credentials, or acquire the shared lock (the parent holds it). Tool compilation and fixture emission for execution suites stay outside timed intervals; the compiler suite intentionally measures production compiler phases. Missing prerequisites produce a nonzero exit and a `blocked` output; failures produce a nonzero exit and a `failed` output. Never silently skip requested coverage. Use subprocess argument arrays and bounded timeouts, clean up child servers/browsers in finally blocks.

Every output is a JSON object:

```
{
  "schema_version": 1,
  "suite": "runtime",
  "status": "complete",
  "cases": [{
    "name": "map.scalar.insert",
    "unit": "ns/op",
    "samples": [123.0, 125.0],
    "warmup_samples": [140.0],
    "iterations_per_sample": 100,
    "timing_scope": "description of exact timed boundary",
    "parameters": {"size": 100, "contract": "immutable-scalar-map-v1"},
    "correctness": {"passed": true, "checks": ["values", "old-version-unchanged"]},
    "metrics": {}
  }],
  "notes": [],
  "artifacts": []
}
```

Preparation outputs have `suite: "prepare"` and may have empty cases. Blocked/failed outputs have a human-readable `reason`. Metrics may hold additional counts, per-request distributions, memory readings, or stage data with honest names/units. Unsupported metrics are explicitly unavailable, never zero. Startup uses milliseconds per launch; codecs can report bytes/second as additional metrics. No per-request p95/p99 may be inferred from batch averages.

Use current production code, real compiler emission, or maintained representative applications. Handwritten native comparisons must preserve the tested contract and be labeled separately. Browser tests must use actual runtime modules/controls; user-journey tests must exercise real emitted Can, not renamed native examples. Each suite must cover multiple meaningful workload paths and state its remaining limits. Counts of cases or native variants are not percentages of feature coverage.

Family ownership:

- Compiler: `compiler`, `assertions`, `artifacts`, `editor`; own driver, Go helper under `compiler/perfmeasure/`, and compiler fixtures under `tools/performance/fixtures/compiler/`.
- Runtime: `generated`, `runtime`, `codecs`, `startup`; own driver and runtime TS helpers, Go emission helper under `compiler/perfemit/`, fixtures under `tools/performance/fixtures/runtime/`.
- Apps: `browser`, `server`, `io`, `journeys`; own driver and apps TS/browser helpers, fixtures under `tools/performance/fixtures/apps/`; coordinate with runtime owner to reuse generated fixture outputs through a documented independent emitter command if useful.

The coordinator owns `perf.py`, shared runner modules, top-level docs and integration tests. Do not edit another family's files without coordination.

The parent removes execution scratch after the invocation and compresses admitted
raw results and diagnostic evidence into `evidence.zip`. Successful stdout is
discarded once its output JSON validates; drivers must save required semantic
evidence through their result JSON or raw auxiliary files, never only stdout.
Source/dependency copies, emitted bundles, transient publication stores and Go
cache belong under the owned scratch directories. They are not persistent
baseline evidence. `--keep-work` permits exceptional seven-day retention, reaped
by subsequent runs under the same output parent. No driver owns final retention.

Child temporary directories are redirected to the run-owned `work/_tmp/` tree.
Shutdown confirms the recorded process group is gone before clearing ownership;
cleanup additionally refuses live owner PIDs, changed root/marker identities and
open workspace references. Repeated signals are deferred during startup
registration and teardown. Unknown process/inspection state retains scratch with
failure evidence instead of risking deletion beneath a surviving child.

The persistent `raw/` inventory is flat: auxiliary files must be regular files
directly inside it. Nested directories and symlinks are refused. Compaction opens
and pins the raw directory descriptor, verifies its device/inode and inventory
before removal, unlinks only enumerated records through that descriptor, and
removes only the verified empty directory. A replaced directory is preserved.
The coordinator exclusively owns the run directory; concurrent external edits to
its evidence or archive staging files are unsupported.
