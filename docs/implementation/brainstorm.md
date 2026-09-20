# Stage 2 — Implementation brainstorm

Completed after [research](research.md), before reconciliation. These are implementation alternatives within the approved language, not a syntax vote. The goal is a complete macOS arm64 distribution with native AI, useful CLI/server programs and a small amount of justified glue. No option preserves obsolete syntax or ABI.

## Compiler replacement

| Alternative | Benefit | Cost / failure mode |
|---|---|---|
| New typed pipeline inside this Go repository, retire old passes by milestone | Can package/type/region/fixture identities become explicit once; no old evaluator or proof obligations leak into new checking | Front-load AST and pass interfaces; replacement must be delivered as runnable slices, not an untested rewrite until the end |
| Mutate the current shared AST/parser/checker/emitter in place | Reuses more existing traversal and diagnostics initially | Current string types, global names, proof coupling, flattened completions and regex grammar create repeated intermediate translations; risky hidden old restrictions |
| Parse new syntax into the old IR temporarily | Fastest superficial demo; some old tests remain usable | Imposes old proof, data-only callback and result-layout constraints or requires extensive shims that soon disappear; no compatibility benefit offsets these costs |

Pre-consultation engineering preference: a new typed pipeline under `compiler/internal/` while retaining the Go CLI entry and narrowly audited utilities. Remove superseded production paths when each slice is routed, with a final explicit deletion gate. No second public “legacy mode.” The project may be incomplete between implementation milestones; unsupported new syntax must fail explicitly rather than execute old semantics.

Proposed pass boundaries: project manifest/package graph → source/lexing/parsing → declaration collection/resolution → concrete types/specializations → completion/AI/resource/assertion annotations → checked IR → TypeScript/source mappings → packaged Bun. Keep diagnostics accumulated before execution. LSP stops at static analysis. Normal verification runs real generated code with only specified boundary substitution.

## Bundled Bun alternatives

Both ship exact upstream executable bytes, without a separate Bun installation, and both need pinned digests, notices, architecture checks and signing validation. Neither fetches Bun on first run or uses PATH as a silent fallback.

| Criterion | Sidecar in versioned Can distribution | Embedded bytes in Go; extract to versioned cache |
|---|---|---|
| Installation | One archive/container with launcher, private runtime and notices; launcher resolves its installation root | One larger launcher download plus notices; embedded executable must become an on-disk child |
| First run/offline | Direct exec; no cache write required | Extract, verify, set executable permissions and publish atomically; offline works if disk is writable |
| Integrity | Verify manifest and executable; prevent relative cwd/PATH substitution | Verify embedded digest and extracted file; defend shared-cache links, partial files, stale entries and races |
| Concurrency | Atomic whole-version install; running processes retain old version | Per-version/digest cache plus synchronization and atomic rename; concurrent extraction and cleanup need tests |
| Permissions/storage | Read/execute installation; archive layout must stay intact | Writable executable cache needed; restricted/noexec locations can block; both embedded and extracted copies occupy disk |
| macOS trust | Inspect launcher, unchanged upstream child and signed/notarized container together | Preserving executable bytes preserves embedded signature bytes, but extraction/quarantine/trust behavior still needs actual testing |
| Updates | Install a new version directory, switch launcher pointer, retain active old versions | Replace launcher; old caches need owner-aware cleanup without deleting another process's executable |
| Portability | Per-platform bundles; arm64 now, x64/Linux only after admission | Per-platform launcher payloads; embedding does not avoid architecture or OS policy differences |
| Implementation burden | Layout/installer/lookup tests; fewer runtime filesystem mechanisms | Adds a correctness-critical extractor/cache protocol and related failure diagnostics |

Pre-consultation preference: sidecar, because the user's requirement is no separate installation, not one physical file. Keep embedding as a documented future packaging alternative if an actual distribution constraint warrants its cache complexity. This preference does not claim a signed Can package is already feasible without testing.

Recommended candidate layout: `can-<version>-darwin-arm64/bin/canlc`, `libexec/bun/<version>/bun`, `share/can/{stdlib,runtime,tools,licenses}`, and an integrity manifest. Use `os.Executable` and resolved installation root, then an absolute child path with argument arrays. Distribution resources are not authored backend imports. An installer may create a user-bin symlink, but the launcher resolves the real installation tree. Never write application outputs into the installed distribution.

## SQL parser binding

The official CGo binding minimizes wrapper layers and uses the upstream project's own types/tests. It adds C toolchain work to compiler builds, not to end-user installs; macOS arm64 is a tractable first build target. The WASM/wazero binding avoids CGo toolchain friction and can simplify later cross-builds, but adds an execution engine, cold-start/runtime cost and another compatibility layer. Neither has yet been benchmarked in this repository. Prefer official CGo provisionally; require a bounded parity/size/cold-start spike against the WASM alternative before adoption. Reject a handwritten Go SQL parser in either case.

## Standard library and runtime shape

Keep ordinary domain records, option variants and meaningful examples. Adapt old test scenarios when they assert a current contract. Replace native-equivalent bodies in text/seq/map/set/json/numerics with catalogue lowering. Remove dec, fuel workers, proof brands, extern adapters and generic outcome wrappers that are no longer needed. Newly implement connections, judgments, generation, completion/owner adapters, safe HTML, HTTP/HTMX and typed SQL because the current implementation lacks them.

Two runtime organizations are plausible: inline every adapter in every generated module, or import a versioned private runtime module set. Prefer private modules for completion, codecs, transport, ownership and fixture context; emit simple arithmetic/string operations directly. This keeps identity-bearing registries shared, avoids duplicated resource state, and lets compiler/runtime hashes bind together. Tree-shaking is optional later; do not optimize away visible prepared work or handler effects.

Source-map encoding can run a pinned prebundled upstream JS helper on bundled Bun after Go produces TS and mapping segments. This adds one build subprocess but no npm install or new algorithm. Alternatives are a mature Go encoder with equivalent coverage (not yet identified/qualified) or a custom VLQ writer (unnecessary). Diagnostics own Can spans regardless of mapper availability; map helper failure fails the build, never publishes partially mapped artifacts.

## Vertical slices and risks

First obtain one manifest-backed `void main(str[] args)` CLI program, diagnostics and assertion running under the packaged runtime into an isolated `dist`. Then make a Noul judge with one shared state group, real transport against a local mock and a raw-provider assertion the next slice. Do not postpone native AI behind all utility catalogues.

Expand from that spine to callables/generics/coordination, full Choice/Score/generated records, typed fetch/LLM, then safe HTTP/HTMX and SQL. Resource ownership must precede releasing any early-settlement or server/database feature. Exact JSON needs its own adversarial budgets tests before any external record boundary is admitted.

Highest risks are not typing convenience: duplicate JSON keys lost by native parse; thenable payload assimilation; root lifetime after race; deterministic fixtures confused with production scheduling; transaction commit uncertainty; source-map/diagnostic drift; and safe rebuild deletion. Each has a named task and test gate in the subsequent stages.
