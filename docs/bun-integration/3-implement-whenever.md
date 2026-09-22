# Bun → Can: Implement whenever

This is a separate integration queue. Read [scope, primitive policy and completion rules](README.md) before implementation. Every item is pending. Written order respects listed prerequisites; independent items need not wait for unrelated work. Dependencies may reference earlier tiers. The current LF01–LF21 implementer is unchanged.

“Whenever” means lower urgency, not silent deletion. Boundary-gated items require an explicit contract decision before implementation; inclusion does not approve arbitrary native execution or weaken Can’s type/resource rules.

- [ ] **B3-01 — Native shell composition**

  **Can surface:** Library or pipeline primitive candidate. **Depends:** B1-04, B1-05.

  **Native basis:** Bun Shell ($).

  **Why this tier:** A useful convenience after safe explicit subprocess and pipe contracts already exist.

  **Work:** Compare Bun-shell-native composition with typed process pipelines on real command workflows. Define interpolation, redirects, environment, failure/pipe status and source spans before any new syntax. No custom Can shell implementation.

  **Done when:** Test metacharacters, paths with spaces, redirect failures and multi-process cleanup; keep behavior distinct from argument-array execution. All common completion rules also pass.

  **Bun references:** [documentation 1](https://bun.sh/docs/runtime/shell).

- [ ] **B3-02 — Persistent OS job registration**

  **Can surface:** Library plus deployment-oriented primitive candidate. **Depends:** B2-09.

  **Native basis:** Bun.cron OS-level registration.

  **Why this tier:** Useful for installed scheduled tools, but introduces persistence outside the current process.

  **Work:** Define install/list/remove/update ownership, executable identity and platform-specific support. Separate scheduling data from an application silently installing itself. No system jobs are created by this roadmap or its normal assertion fixtures.

  **Done when:** Use disposable host/job contexts for native qualification; uninstall and changed executable paths have explicit outcomes. Tests never leave scheduled work on the developer machine. All common completion rules also pass.

  **Bun references:** [documentation 1](https://bun.sh/docs/runtime/cron).

- [ ] **B3-03 — Controlled native foreign-function integration**

  **Can surface:** Boundary-gated library/primitive candidate. **Depends:** none within this roadmap.

  **Native basis:** bun:ffi and native C compilation APIs.

  **Why this tier:** Potential access to specialized native facilities, with a larger ABI and memory contract.

  **Work:** First resolve how distribution-owned typed adapters can use the built-ins while preserving Can data/ownership. Assess any author-visible interface separately against the current foreign-code boundary. Do not count arbitrary external libraries as Bun stdlib or adopt unrestricted pointers by default.

  **Done when:** Only a selected bounded interface can proceed to ABI/ownership/callback/error tests. Explicitly record unsupported cases; no generic foreign escape is approved by placement here. All common completion rules also pass.

  **Bun references:** [documentation 1](https://bun.sh/docs/runtime/ffi), [documentation 2](https://bun.sh/docs/runtime/c-compiler).

- [ ] **B3-04 — Node-API integration boundary**

  **Can surface:** Boundary-gated native library integration. **Depends:** none within this roadmap.

  **Native basis:** Bun Node-API loader/support.

  **Why this tier:** Potential reuse of native modules for specialized applications after the base library is useful.

  **Work:** Inventory supported Node-API operations and compare a distribution-owned bridge with B3-03. This item concerns Bun’s loading/interop facility, not adding npm packages to this roadmap. Authoring, installation and opaque ownership policy require a recorded decision.

  **Done when:** Qualify a minimal owned test module for load/unload/error/memory behavior only after the boundary is settled; do not claim all Node addons are compatible. All common completion rules also pass.

  **Bun references:** [documentation 1](https://bun.sh/docs/runtime/node-api).

- [ ] **B3-05 — Typed source transformation and artifact generation**

  **Can surface:** Boundary-gated library. **Depends:** B1-01.

  **Native basis:** Bun.Transpiler and Bun.build.

  **Why this tier:** Useful for Can-based developer tools rather than most applications.

  **Work:** Evaluate a typed input-to-output-artifact interface over native transforms/builds. Separate processing source as data from executing generated host code or changing Can’s compiler backend. Resolve conflicts with existing authored-backend restrictions before admission.

  **Done when:** Inspect deterministic outputs, maps, malformed input and import/file confinement. No npm auto-install or arbitrary plugin execution is implied. All common completion rules also pass.

  **Bun references:** [documentation 1](https://bun.sh/docs/runtime/transpiler), [documentation 2](https://bun.sh/docs/bundler).

- [ ] **B3-06 — Native module graph and file-system routing helpers**

  **Can surface:** Boundary-gated library/tooling surface. **Depends:** B3-05.

  **Native basis:** Bun.ModuleGraph, Bun.FileSystemRouter, Bun.resolve, Bun.plugin.

  **Why this tier:** Useful for specialized tooling once normal file and artifact APIs cover common workflows.

  **Work:** Evaluate typed graph/resolution/routing results. Keep dynamic loader hooks behind an explicit boundary decision; ordinary application library use must not bypass Can imports, contracts or trusted compilation. No second Can module resolver.

  **Done when:** Validate aliases/path cycles/source maps and unsupported dynamic hooks on the exact target. Public admission remains pending until typed contracts are selected. All common completion rules also pass.

  **Bun references:** [documentation 1](https://bun.sh/docs/runtime/module-graph), [documentation 2](https://bun.sh/docs/runtime/file-system-router), [documentation 3](https://bun.sh/docs/runtime/plugins), [documentation 4](https://bun.sh/docs/runtime/utils).

- [ ] **B3-07 — Runtime diagnostics, inspection and profiling**

  **Can surface:** Library for safe observations; compiler-owned internals. **Depends:** none within this roadmap.

  **Native basis:** Bun.inspect, version/revision, nanoseconds, heap snapshots, gc and bun:jsc.

  **Why this tier:** Useful for performance and operational tooling, with low-level functions often better kept internal.

  **Work:** Expose useful typed runtime/capability observations and bounded profiling outputs. Record which low-level operations remain compiler-owned. Do not replace Can equality with Bun.deepEquals, leak private failure causes or expose Promise.peek as a new completion model.

  **Done when:** Test stable report schemas, unavailable features and sensitive-value omission. Distinguish observations from commands that alter GC/runtime behavior. All common completion rules also pass.

  **Bun references:** [documentation 1](https://bun.sh/docs/runtime/utils), [documentation 2](https://bun.sh/docs/runtime/bun-apis).

- [ ] **B3-08 — Memory mapping and advanced binary ownership**

  **Can surface:** Boundary-gated library/primitive candidate. **Depends:** B1-01, B1-05.

  **Native basis:** Bun.mmap, ArrayBufferSink and relevant native buffer APIs.

  **Why this tier:** Potential performance gains for large datasets once ordinary files and streams have evidence.

  **Work:** Benchmark complete workloads before adding an owned mapping/buffer contract. Preserve immutable views, bounds, close/detach semantics and lifetime checks. Do not expose allocUnsafe or shared mutable memory as ordinary Can immutable bytes.

  **Done when:** Compare native throughput/memory with stream baseline and test aliases, truncation, use-after-close and cleanup. Admit only features with both measured need and a sound boundary. All common completion rules also pass.

  **Bun references:** [documentation 1](https://bun.sh/docs/runtime/binary-data), [documentation 2](https://bun.sh/docs/runtime/bun-apis).

- [ ] **B3-09 — Bun test and host-tooling integration**

  **Can surface:** Tooling-facing typed library; preserve Can assertions. **Depends:** none within this roadmap.

  **Native basis:** bun:test and native test/mocking facilities.

  **Why this tier:** Useful for native adapter conformance and developer tools without duplicating the language’s assertion system.

  **Work:** Assess typed conformance/result integration and distribution-owned native probes. Preserve attached Can assertions and their fixtures as the user-facing verification contract. Native test runners do not establish an alternative way to skip Can checks.

  **Done when:** Cross-language conformance results retain their evidence labels and failure status; a mocked host test cannot be presented as a passing native Can integration. All common completion rules also pass.

  **Bun references:** [documentation 1](https://bun.sh/docs/test).
