# Compiler family workloads and boundaries

`drivers/compiler.py` prepares independent `flat-10`, `flat-100`, and `flat-1000`
Can scale projects plus unchanged copies of maintained `examples/utilities` and
`examples/invoice-compare` inside its work directory. The flat entry calls every
pure integer function, with a concrete attached assertion for every function and
entry. Expected integers are their function indices. No root uses skip semantics.
The maintained anchors are fixed application inputs; `--size` scales the flat
probe and does not clone or artificially enlarge the anchors.

Anchor case parameters bind an `input_content_sha256` over a sorted relative-file
SHA-256 inventory of source/manifest/vendor inputs (excluding documentation and
generated output). Metrics retain each input-file SHA. Temporary paths never
enter case names or parameters. Comparisons therefore distinguish changed app
inputs from changed compiler behavior. Source-file/project counts and byte sizes
are measured from the actual loaded graph, including vendored controls.

Preparation builds a current-source hash-locked development distribution using
`CAN_PERF_BUN_ARCHIVE` (default `/tmp/bun-darwin-aarch64.zip`), enforcing the pinned
archive digest in `distribution.Build`. No download or installation is performed.
The helper resolves the bundle manifest and production closed runtime inventory.
Checked production modules, mapped modules, assertion modules and root identities
are serialized in each fixture's `prepared.json` for untimed reuse. `CheckProgram`
already checks concrete assertions; preparation reuses that fully checked IR.
Browser anchors use `CheckBrowserProgram` and capability-gated `BrowserModules`.
The shared parent owns the audit lock; drivers/helpers never acquire it.

## Coverage expansion

| Slice | Initial boundary | Expanded workloads/boundaries |
|---|---|---|
| Compiler | Seven flat scale phases | Retains flat phases; adds load, target-specific check, emit and pipeline for utilities and invoice-compare, including the latter's real vendored controls, records, callbacks, generics and browser capability gate |
| Assertions | Full flat roots, one worker | Adds every maintained utilities root with separately measured one-worker and four-worker production supervision: URL success/domain failure, ordered query values, timezone formatting, regex, Unicode/base64 and mocked stdout entry branches |
| Artifacts | Native validation and already-published reuse | Adds checked-span/source-index assembly plus production source-map encoding/validation, and first payload publication into independently empty owned output/object stores; validation now consumes full mapped artifacts |
| Editor | Diagnostics, hover, definition, formatting | Adds exact callable completion-set/signature/provenance oracle and exact safe-rename edits, checked applied overlay, renamed hover identity and clean restoration |

## Compiler timing

Compiler phases use actual production Go internals. `check` includes resolution,
declaration checking, body checking and IR; `pipeline` includes load/check/emit.
These overlapping phase measurements must not be added. Resident runtime bytes
and prerequisite AST/world/IR construction are outside each targeted phase.
The final emitted paths and bytes of each emission/pipeline batch must equal the
prepared checked output; this oracle is outside timing. The browser anchor stops
at module emission, before browser execution, source-map sealing and bundling.

Go allocation/GC counters exclude Bun child allocations and include only the
measured Go operation. `runtime.ReadMemStats` is outside wall-clock timing. Natural
GC remains active; process-tree RSS is explicitly unavailable.

## Assertions

Production `RunSupervised` validates a precompiled generation lease, launches one
fresh bounded Bun worker per concrete root, runs the real harness and consumes
its reports. Every prepared root executes in every operation; every report must
pass. Each operation receives a distinct pre-acquired lease because supervision
consumes the lease. Emission, staging and native TypeScript validation are setup.
Raw production `elapsedMs` observations are retained per root/per sample without
inventing percentiles. The one/four-worker utilities cases isolate fan-out on the
same semantic workload. They do not claim the entire provider/resource corpus.

A benchmark-only source-index adapter translates the real checked IR spans into
the existing encoder protocol. The qualified production `source-maps.ts` tool and
native output validator own map correctness; encoder request receipts are checked.

## Artifacts

Source-map timing includes the checked-span/index adapter, actual qualified
production encoding tool launch, encoding, production map validation and receipt.
It excludes checking/emission and filesystem publication. A prepared byte/path
oracle validates each batch after its clock stops.

Native validation includes content identities, closed import inventories,
TypeScript transpilation and source-index/map relationships on resident mapped
production artifacts. First publication writes the same validated mapped payload
into a distinct fresh owned store for every operation, including object storage,
generation manifests and current selection. Copying source inputs, acquiring the
project lock, loading the snapshot and validating TypeScript occur beforehand.
Each published payload is byte-checked after timing. Generation reuse is separate
and includes existing-generation filesystem integrity checks and selection.
Payload/map file counts and byte sizes are reported. Browser bundling and release
distribution packaging remain uncovered.

## Editor

Actual Content-Length JSON-RPC stdio requests exercise production LSP. Versioned
invalid/valid changes, real hover/definition/formatting responses, the exact
source-authored completion set with checked kinds/arities/provenance, and the
exact two-token `f0` to `f_zero` WorkspaceEdit are checked. Applied rename overlays
must produce clean diagnostics and the renamed symbol's hover; originals are
restored cleanly before the next measured request. Initialization, restoration,
response oracles and applied-overlay checks are outside query clocks. Every
measured request latency is preserved separately; batch averages are never tails.

Per-request deadlines remain bounded at 30 seconds for sizes 10/100 and 250
seconds for size 1000; the shared runner additionally bounds the whole trial.

`prepareRename` is explicitly unavailable in current production: initialize
advertises `renameProvider: true` without `prepareProvider`, and an actual
`textDocument/prepareRename` probe returns `-32601 unknown method`. The probe is
untimed, checked exactly and reported as a gap; no fake latency case is counted.

`--iterations` counts operations per batch. Quick returns three measured batches;
standard returns seven. `--warmups` counts excluded batches. Explicit scale sizes
10/100/1000 remain supported; other sizes are blocked rather than approximated.
