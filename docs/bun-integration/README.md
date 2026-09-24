# Bun standard-library integration into Can

This is the separate Bun expansion roadmap requested after the PostgreSQL-only scope discussion. It contains three ordered lists:

1. [Implement ASAP — 13 tasks](1-implement-asap.md): files, **SQLite and MySQL**, subprocesses, stream/lifecycle foundations, broader HTTP, WebSockets, password/crypto operations, cookies/CSRF, S3, typed data formats, Markdown and everyday data utilities.
2. [Implement later — 14 tasks](2-implement-later.md): Redis, compression, archives, images, XML/HTML rewriting, DNS/TCP/TLS, UDP, watches, schedules, workers, browser operations, OS secrets, richer SQL and additional utilities.
3. [Implement whenever — 9 tasks](3-implement-whenever.md): native shell composition, persistent OS jobs, controlled native interop, source/artifact tools, module tooling, runtime diagnostics, advanced memory/binary ownership and native test integration.

All 36 tasks are unchecked. These are integration priorities, not release dates or ready-to-code language specifications. “ASAP” means the next Bun integration effort, not interruption of the current implementer. Only this new directory was written; no agent was messaged and no LF task, compiler, runtime, catalogue or authoritative specification was changed.

## Scope

Only capabilities implemented by Bun qualify: its native APIs and its supported standard Web/Node APIs. These may become ordinary Can standard-library operations, extensions to existing native forms, or well-founded new Can primitives. External npm packages, application frameworks, AI providers, migration frameworks, media-generation services and unrelated language redesign are outside these lists.

The [Bun native API inventory](https://bun.sh/docs/runtime/bun-apis), [Web API inventory](https://bun.sh/docs/runtime/web-apis) and [Node compatibility documentation](https://bun.sh/docs/runtime/nodejs-compat) are discovery sources. The detailed task links support individual integrations. Current documentation is not evidence that every API works on Can's pinned runtime/platform: each task must qualify the exact shipped Bun or explicitly propose a runtime update before depending on unavailable functionality. Support is per operation, not an unqualified claim of parity with every Bun or Node API.

The [catalogue snapshot](evidence/catalogue-baseline.json) distinguishes existing Can exposure from prospective work. Existing AI, HTTP, PostgreSQL, JSON/bytes, collections, basic clock/random/SHA256, HTML and assertions are foundations to extend, not duplicate. Catalogue presence is not itself a runtime conformance result, especially while another implementer is working.

Prior deferrals such as filesystem access and non-PostgreSQL SQL are explicitly reopened for this independent roadmap. They are not permanent product exclusions. The currently executing LF contract remains unchanged; when a task starts, record its exact accepted contract and affected restrictions in the authoritative documents before coding it. Do not convert a smaller implementation milestone into a permanent support boundary without saying so.

## Library or Can primitive

Neither “everything is a library call” nor “every Bun API needs a keyword” is the rule. The proposed surface on each task is a planning hypothesis. New grammar must be settled using complete examples, ordinary Can constraints and the repository's fresh-consultation requirement for difficult design choices.

| Family | Why a primitive might help | What must be settled beyond match-like appearance |
| --- | --- | --- |
| Streams | Typed chunks with a visible processing region | Pull/push behavior, backpressure, repeated values, terminal failure and cancellation |
| WebSockets | Named message/open/close/drain handlers and typed session state | Repeated dispatch, direction, close ownership, handler errors and pressure |
| Subprocess output | stdout/stderr/exit are distinct events from one owned operation | Concurrent drains, output limits, process termination and exit versus transport failure |
| TCP/UDP | Typed receive/send/end/error regions | Framing, partial data, datagram reliability and connection lifetime |
| Filesystem watches | Declared change handlers | Coalescing/loss, rename semantics, rescan and watch cancellation |
| Scheduled jobs | A named schedule and execution body | Timezones, overlap, missed ticks, error ownership and process lifetime |
| Worker tasks | Typed inputs, results and message handlers | Isolation, transferable data, captures, crash and termination |
| HTTP/browser sessions | Existing native handlers may benefit from richer event boundaries | Streaming/disconnect/navigation ownership without erasing current contracts |

B1-05 compares at least a document stream, subprocess-output consumer and WebSocket session; later tasks test whether the result fits their domains. Alternatives include normal library operations with named handlers, domain-specific Can forms, or a shared primitive if the programs justify one. WebSockets are one case in that comparison. A one-shot completion match and a long-lived subscription have different semantics; syntax must make that distinction clear.

Three [fresh Jev consultations](evidence/README.md) advised on broad priority, mixed library/primitive exposure and multi-domain examples. They agree on those planning principles. This is advice, not approval of a universal event primitive, any exact spelling or every individual tier assignment. The detailed ordering and task scope are engineering judgments recorded here.

## Order and shared completion rules

The numeric order in each file respects its explicit dependencies. For example, shared SQL dialect admission precedes MySQL, lifecycle streams precede WebSockets, and ordinary process execution precedes optional shell composition. Priority is not a requirement that every unrelated ASAP task finish before an independent later task can start. Do not add a global “all language fixes must finish” technical dependency: use the actual required compiler/harness capability, while leaving the running implementer untouched.

Each integration task must:

1. Verify its exact native API, availability, runtime/platform requirements and current Can overlap. Freeze accepted operations and explicit exclusions; do not silently reduce the requested family to one favored backend.
2. Select its Can signatures or primitive grammar, input/output types, public errors, resource/state ownership and testing boundary before implementation. Record any genuinely new difficult choice with three fully reworded Jev consultations and disagreement analysis.
3. Lower to native Bun/standard operations. Add only adapters for Can's typed contracts, immutability, precise failure provenance and lifecycle. No replacement SQL driver, parser, stream engine, crypto algorithm, scheduler or shell.
4. Supply before/after complete Can programs, positive cases and deliberately rejected programs. Negative programs fail for the intended obligation. Primitive prototypes include complete lifecycle traces, not just attractive arm syntax.
5. Execute the real generated native path, raw boundary fixtures and deterministic local assertion cases. Cover malformed input, external failures, partial completion, limits and cleanup as applicable. Keep supplied-completion, policy, raw-native and runtime-conformance evidence distinct.
6. Preserve local test ownership, sticky harness violations and verified-build behavior when those foundations are available. Provider/model quality is not ordinary build evidence. Persistent host changes use isolated qualification contexts; planning creates none.
7. Publish supported-operation/platform documentation and reproducible evidence before checking off the task. Missing services, unsupported Bun APIs, untested platform behavior and design-gated boundaries remain visibly pending.

The “whenever” native interop/tooling items concern Bun facilities themselves. Raw pointers, arbitrary npm loading, backend-code execution and mutable shared memory are not implicitly admitted. Some APIs may remain compiler-owned after review; record that explicit disposition and rationale instead of manufacturing an unsafe public Can equivalent. A task cannot be marked “implemented” merely because the disposition was to keep an API internal.

## Coverage map

This maps capability families, including existing exposure and specialized runtime APIs, so no broad area disappears silently. It is a family-level inventory, not a promise to mirror every overload or the complete Node library.

| Bun-provided family | Task / treatment |
| --- | --- |
| File I/O, directories, paths, glob | B1-01; incremental ownership B1-05 |
| SQL PostgreSQL | Existing foundation; shared admission B1-02, richer operations B2-13 |
| SQL SQLite / bun:sqlite | B1-02, B2-13 |
| SQL MySQL | B1-03, B2-13 |
| Spawn, executable lookup, pipes | B1-04, B1-05 |
| Web streams, binary stream conversion, cancellation | B1-05 |
| HTTP server, fetch, request/response, multipart, TLS | Existing native forms extended by B1-06 |
| WebSocket client and server | B1-07 |
| Password, hashing, WebCrypto | Existing basic hash/random extended by B1-08 |
| Cookies and CSRF | B1-09 |
| S3-compatible storage | B1-10 |
| TOML, YAML, JSON5, JSONL | B1-11; existing JSON codecs retained |
| Markdown | B1-12 |
| URL, regex, encodings, Date/Intl | B1-13; existing Can primitives retained |
| Redis/Valkey | B2-01 |
| Compression | B2-02 |
| Archives | B2-03 |
| Images | B2-04 |
| XML and HTMLRewriter | B2-05; existing safe HTML remains distinct |
| DNS, TCP/TLS and UDP | B2-06, B2-07 |
| Node-compatible filesystem watchers | B2-08 |
| Native timers and in-process cron | Existing sleep/clock extended by B2-09 |
| Workers and native message transfer | B2-10; experimental support requires qualification |
| WebView | B2-11 |
| Native OS secret stores | B2-12 |
| Semver, color, display width, UUIDv7 | B2-14 |
| Native shell | B3-01 |
| Persistent OS cron | B3-02 |
| FFI, native C compilation | B3-03; explicit boundary gate |
| Node-API | B3-04; native facility only, no npm package wishlist |
| Bun.build and Transpiler | B3-05; typed artifact/data use must be distinguished from execution |
| ModuleGraph, FileSystemRouter, resolution and plugin hooks | B3-06; dynamic hooks require a separate boundary decision |
| Version/revision, inspection, profiling/heap/GC/JSC | B3-07; distinguish safe public observations from compiler internals |
| Promise peek and deep comparison | B3-07 review; no replacement of Can completion/equality semantics |
| mmap, buffer sinks, unsafe/shared memory | B3-08 review; no raw mutable/unsafe public buffer assumption |
| bun:test and native mocks | B3-09; conformance support, not a second Can assertion contract |
| Package manager/install, runtime auto-install | Outside Can runtime stdlib integration; no npm expansion or import bypass |
| JSX, transpilation, browser build output | B3-05 covers additional artifact-processing candidates. A Can browser target and invoice grid now exist (T21–T25); general runtime/bundling support remains unfinished. No Can frontend framework is selected. See the [current reconciliation](../syntax-taste/post-upgrade-reconciliation-2026-09-24.md). |
| Other Bun-implemented Web/Node APIs | Operation-level inventory under the owning family before admission; no assumed complete compatibility |

[Machine-readable tasks](tasks.json), [planning validator](evidence/validate.py) and its [report](evidence/validation.json) check priority/dependency order, pending status, source references and consultation integrity. They do not claim any integration is implemented.

## Detailed ASAP preparation

The [ASAP implementation handoff](asap/README.md) expands all thirteen near-term capabilities into file-level plans, 94 implementation steps, native evidence, bounded design gates and [agent execution instructions](asap/agent-prompt.md). Follow its [ordered queue](asap/execution-queue.md) for implementation. This adds detail to the existing roadmap; no task is marked implemented.
