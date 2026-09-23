# Bun → Can: Implement ASAP

This is a separate integration queue. Read [scope, primitive policy and completion rules](README.md) before implementation. Every item is pending. Written order respects listed prerequisites; independent items need not wait for unrelated work. Dependencies may reference earlier tiers. The current LF01–LF21 implementer is unchanged.

- [x] **B1-01 — Files, directories, paths and globbing** (bounded operations complete 2026-09-23; streaming file handles follow via B1-05.08 — see the execution queue and `docs/implementation/evidence/2026-09-23/b1-01/`)

  **Can surface:** Library. **Depends:** none within this roadmap.

  **Native basis:** Bun.file, Bun.write, FileSink; Bun.Glob; implemented node:fs/promises and node:path.

  **Why this tier:** Unlock local tools, document ingestion and generated artifacts immediately.

  **Work:** Add typed text/byte read/write, stat/existence, directory listing/create, copy/move/remove and path/glob operations. Specify explicit path roots, overwrite behavior, limits, symbolic links and file ownership; keep project-build confinement distinct from application file access. Incremental handles connect to B1-05.

  **Done when:** Round-trip binary/text data; missing/denied/invalid paths and interrupted writes yield declared failures; temporary-tree fixtures never alter real user files. Use native filesystem operations, not a Can filesystem. All common completion rules also pass.

  **Bun references:** [documentation 1](https://bun.sh/docs/runtime/file-io), [documentation 2](https://bun.sh/docs/runtime/glob), [documentation 3](https://bun.sh/docs/runtime/nodejs-compat).

- [ ] **B1-02 — SQLite integration and the shared multi-dialect SQL boundary**

  **Can surface:** Library. **Depends:** B1-01.

  **Native basis:** Bun.SQL SQLite adapter and bun:sqlite.

  **Why this tier:** Can should support useful local databases as well as remote services.

  **Work:** Expose in-memory and file SQLite with typed parameters/results and transactions. Separate portable Can SQL contracts from dialect-specific grammar, value mappings and capabilities. Compare the two native SQLite interfaces before choosing the adapter; replace the PostgreSQL-only parser assumption with explicit dialect-aware admission. Keep MySQL in the same ASAP milestone.

  **Done when:** A local app persists/reopens data; rollback, nulls, bigint boundaries, binding and locking are exercised. PostgreSQL regressions remain green. No SQL string concatenation or falsely portable dialect syntax. All common completion rules also pass.

  **Bun references:** [documentation 1](https://bun.sh/docs/runtime/sql), [documentation 2](https://bun.sh/docs/runtime/sqlite).

- [ ] **B1-03 — MySQL integration**

  **Can surface:** Library. **Depends:** B1-02.

  **Native basis:** Bun.SQL MySQL adapter.

  **Why this tier:** Complete the three database engines supplied by Bun rather than treating PostgreSQL as the product limit.

  **Work:** Add MySQL connections, prepared values, query cardinalities and transactions using the shared dialect structure. Define affected rows/insert IDs, placeholder handling, supported value mappings, connection configuration and database error translation. Do not implement another wire driver.

  **Done when:** Run equivalent typed CRUD/rollback workloads against local MySQL, SQLite and PostgreSQL plus dialect-specific negatives; unsupported values/grammar are reported precisely. Record service availability rather than claim mocked queries prove native compatibility. All common completion rules also pass.

  **Bun references:** [documentation 1](https://bun.sh/docs/runtime/sql).

- [x] **B1-04 — Bounded child-process execution** (complete 2026-09-23 — see the execution queue and `docs/implementation/evidence/2026-09-23/b1-04/`)

  **Can surface:** Library; event primitive candidate for incremental output. **Depends:** none within this roadmap.

  **Native basis:** Bun.spawn, Bun.which and process pipes.

  **Why this tier:** Let Can automate existing command-line tools.

  **Work:** Add executable-plus-argument-array invocation, explicit environment/cwd, stdin, bounded stdout/stderr, exit status, timeout and termination. Start with bounded completion; retain incremental process-output integration as a deliverable of B1-05, not a permanent omission. Do not confuse this with shell interpolation.

  **Done when:** Test success, nonzero exit, missing executable, output overflow, signals and hung children with owned fixture processes. Verify no argument reinterpretation as shell code and no orphaned process on owner shutdown. All common completion rules also pass.

  **Bun references:** [documentation 1](https://bun.sh/docs/runtime/child-process), [documentation 2](https://bun.sh/docs/runtime/utils).

- [ ] **B1-05 — Streams and reusable lifecycle contracts**

  **Can surface:** Primitive candidate plus typed stream library. **Depends:** B1-01, B1-04.

  **Native basis:** ReadableStream, WritableStream, TransformStream, AbortController; Bun stream/byte helpers.

  **Why this tier:** Streaming is a shared foundation for files, processes, HTTP, storage and event-driven programs.

  **Work:** Compare complete streaming-document, subprocess-output and WebSocket session programs using ordinary calls/named handlers, domain primitives and a shared primitive. Settle typed chunks, repeated dispatch, terminal completion, backpressure, cancellation and cleanup before grammar. Implement the chosen stream surface and incremental process pipes without a second stream engine.

  **Done when:** Bounded-memory chunk transfer, slow consumers, partial UTF-8, early cancellation, errors after partial data and double-close have deterministic attached event fixtures. Prove native stream use. A one-shot match must not silently become repeated subscription dispatch. All common completion rules also pass.

  **Bun references:** [documentation 1](https://bun.sh/docs/runtime/streams), [documentation 2](https://bun.sh/docs/runtime/binary-data), [documentation 3](https://bun.sh/docs/runtime/web-apis).

- [ ] **B1-06 — Broader HTTP and incremental request/response bodies**

  **Can surface:** Extend existing native forms; event/stream primitive candidate. **Depends:** B1-05.

  **Native basis:** Bun.serve, fetch, Request, Response, FormData.

  **Why this tier:** Expand the existing HTTP foundation into practical upload, download and streaming applications.

  **Work:** Extend methods/routing/headers, multipart uploads, incremental responses including SSE framing, explicit transport cancellation and server configuration/TLS through supported native options. Preserve the existing native fetch and server contracts; identify additions individually rather than replace them with raw JS objects.

  **Done when:** Exercise uploads, streaming progress, client disconnect, malformed multipart, size limits and shutdown; no full buffering when a streaming contract is selected. TLS and routing tests use controlled local endpoints. All common completion rules also pass.

  **Bun references:** [documentation 1](https://bun.sh/docs/runtime/http/server), [documentation 2](https://bun.sh/docs/runtime/streams), [documentation 3](https://bun.sh/docs/runtime/web-apis).

- [ ] **B1-07 — WebSocket client and server integration**

  **Can surface:** Strong primitive candidate plus connection operations. **Depends:** B1-05, B1-06.

  **Native basis:** WebSocket client; Bun.serve upgrade and websocket handlers.

  **Why this tier:** Enable live dashboards, conversations and long-lived typed application sessions.

  **Work:** Design open/message/drain/close/error handling, typed frames, session state, backpressure and termination using B1-05 comparisons. Assess native Can declaration/arm syntax seriously; do not assume existing match or a generic event primitive is automatically correct. Implement both client and server paths with attached ordered event tests.

  **Done when:** Bidirectional typed exchange, malformed frames, slow peers, disconnect/reconnect policy and handler failures have precise outcomes. No automatic retry or erased failure provenance. Native handlers execute in fixtures, not only supplied final results. All common completion rules also pass.

  **Bun references:** [documentation 1](https://bun.sh/docs/runtime/http/websockets), [documentation 2](https://bun.sh/docs/runtime/web-apis).

- [ ] **B1-08 — Password and broader cryptographic operations**

  **Can surface:** Library. **Depends:** none within this roadmap.

  **Native basis:** Bun.password, Bun.CryptoHasher, WebCrypto.

  **Why this tier:** Provide the native primitives needed for authentication, signatures and integrity checks.

  **Work:** Add password hash/verify, keyed hashing and a defined encryption/signature/key-operation subset from Bun-provided APIs. Reuse existing SHA256/random capabilities. Specify opaque keys, algorithm options, byte encodings and failure contracts; do not invent cryptographic algorithms or an authentication framework.

  **Done when:** Use standard vectors and password round-trips; invalid encodings/keys/parameters and failed verification are distinguished. Tests avoid leaking keys or claiming equal randomized ciphertext; inspect native API lowering. All common completion rules also pass.

  **Bun references:** [documentation 1](https://bun.sh/docs/runtime/hashing), [documentation 2](https://bun.sh/docs/runtime/web-apis).

- [ ] **B1-09 — Cookies and CSRF primitives**

  **Can surface:** Library. **Depends:** B1-06, B1-08.

  **Native basis:** Bun.Cookie, Bun.CookieMap, Bun.CSRF.

  **Why this tier:** Complete essential web session building blocks without adding a framework.

  **Work:** Expose typed cookie parsing/serialization and native CSRF generation/verification with explicit attributes, expiry/origin policy and immutable request/response integration. Keep session storage/application authorization out of this low-level integration.

  **Done when:** Check duplicate/malformed cookies, attributes, token mismatch/expiry and native interoperability with deterministic time where needed. Native results cannot bypass safe-header validation. All common completion rules also pass.

  **Bun references:** [documentation 1](https://bun.sh/docs/runtime/cookies), [documentation 2](https://bun.sh/docs/runtime/csrf).

- [ ] **B1-10 — S3-compatible object storage**

  **Can surface:** Library plus stream integration. **Depends:** B1-01, B1-05.

  **Native basis:** Bun.S3Client and native S3 file operations.

  **Why this tier:** Enable document stores, uploads and generated artifacts backed by existing object storage.

  **Work:** Add typed read/write/delete/list and supported signed URL/multipart capabilities, explicit endpoint/credentials and streaming transfers. Use native signing/transport. Preserve metadata, object identifiers and failure details; separate support levels for individual operations.

  **Done when:** Local S3-compatible endpoint tests cover object round-trip, pagination where exposed, absent objects, bad credentials and interrupted transfer. Raw fixtures and stream ownership tests remain separate from live-service claims. All common completion rules also pass.

  **Bun references:** [documentation 1](https://bun.sh/docs/runtime/s3).

- [ ] **B1-11 — Typed TOML, YAML, JSON5 and JSONL codecs**

  **Can surface:** Library. **Depends:** B1-05.

  **Native basis:** Bun.TOML, Bun.YAML, Bun.JSON5, Bun.JSONL where qualified.

  **Why this tier:** Let Can consume common configuration, documents and line-oriented datasets.

  **Work:** Add supported parse/serialize operations with typed Can projections and explicit number/null/duplicate-key policies. JSONL includes incremental record/error locations. Qualify each concrete API on the pinned runtime; do not infer serialization support from parser availability or fall back to a hand-written parser.

  **Done when:** Valid records round-trip where supported; malformed input, integer precision loss, unsupported tags and schema mismatches reject precisely. Stream record ordering and partial final lines are tested. All common completion rules also pass.

  **Bun references:** [documentation 1](https://bun.sh/docs/runtime/toml), [documentation 2](https://bun.sh/docs/runtime/yaml), [documentation 3](https://bun.sh/docs/runtime/json5), [documentation 4](https://bun.sh/docs/runtime/jsonl).

- [ ] **B1-12 — Markdown rendering and structured processing**

  **Can surface:** Library. **Depends:** none within this roadmap.

  **Native basis:** Bun.markdown.

  **Why this tier:** Produce reports, documentation and content views using the runtime already shipped.

  **Work:** Expose supported rendering and structured/token operations through typed results. Decide raw-HTML/link handling and trusted HTML conversion explicitly; rendering alone must not manufacture html::safe. Use existing Can immutable text/HTML abstractions.

  **Done when:** Test malformed/edge syntax, links and embedded HTML against native output; enforce the chosen trust policy and preserve source text. No separate Can Markdown parser or site generator. All common completion rules also pass.

  **Bun references:** [documentation 1](https://bun.sh/docs/runtime/markdown).

- [ ] **B1-13 — Common URL, text, byte and time utilities**

  **Can surface:** Library. **Depends:** none within this roadmap.

  **Native basis:** Bun-provided URL/URLSearchParams, RegExp, TextEncoder/Decoder, Date/Intl and byte utilities.

  **Why this tier:** Fill everyday gaps that otherwise force platform escapes or repeated helper code.

  **Work:** Inventory existing operations first; add missing URL construction/query handling, regex matching, byte encodings and timestamp/formatting operations with explicit native semantics. Distinguish instants from durations and invalid dates; keep Can int precision and immutable buffers. Do not introduce a new numeric type system.

  **Done when:** Test Unicode, invalid encodings, regex state/flags, timezone formatting and exact integer boundaries; native operations remain directly identifiable. Pin locale/timezone-dependent evidence. All common completion rules also pass.

  **Bun references:** [documentation 1](https://bun.sh/docs/runtime/web-apis), [documentation 2](https://bun.sh/docs/runtime/binary-data), [documentation 3](https://bun.sh/docs/runtime/utils).

## Prepared implementation handoff

Read the [ASAP preparation index](asap/README.md), [ordered execution queue](asap/execution-queue.md), [filetree/shared contract](asap/integration-contract.md) and [coding-agent prompt](asap/agent-prompt.md). Every B1 task above has a dedicated plan with native lowering, positive/rejected cases and fixture acceptance. Preparation does not complete the implementation checkboxes.
