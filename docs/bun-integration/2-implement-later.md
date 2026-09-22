# Bun → Can: Implement later

This is a separate integration queue. Read [scope, primitive policy and completion rules](README.md) before implementation. Every item is pending. Written order respects listed prerequisites; independent items need not wait for unrelated work. Dependencies may reference earlier tiers. The current LF01–LF21 implementer is unchanged.

- [ ] **B2-01 — Redis/Valkey data and subscriptions**

  **Can surface:** Library; subscription primitive candidate. **Depends:** B1-05.

  **Native basis:** Bun.RedisClient.

  **Why this tier:** Add caching, counters and service communication after the basic storage/event foundation.

  **Work:** Expose a defined typed command set, value codecs, connection ownership and supported subscription/pipelining features. Apply lifecycle design to subscriptions without pretending Redis messages are a one-shot call. No cache framework or durable queue product is implied.

  **Done when:** Controlled-server and raw-fixture tests cover expiry, absent keys, command errors, reconnect/close and subscription event order. State exact delivery guarantees. All common completion rules also pass.

  **Bun references:** [documentation 1](https://bun.sh/docs/runtime/redis).

- [ ] **B2-02 — Compression and decompression**

  **Can surface:** Library; stream adapters. **Depends:** B1-05.

  **Native basis:** Bun gzip/deflate/zstd and supported CompressionStream APIs.

  **Why this tier:** Handle compressed documents and payloads while keeping memory and CPU costs explicit.

  **Work:** Add the native algorithm/options subset and immutable byte results, then supported streaming paths. Put output limits and partial-data failures in the contract; never replace native codecs with Can implementations.

  **Done when:** Round-trips, corrupt/truncated data, output expansion limits and stream cancellation pass with compatible external vectors. All common completion rules also pass.

  **Bun references:** [documentation 1](https://bun.sh/docs/runtime/utils), [documentation 2](https://bun.sh/docs/runtime/streams).

- [ ] **B2-03 — Archive creation, reading and extraction**

  **Can surface:** Library. **Depends:** B1-01, B2-02.

  **Native basis:** Bun.Archive.

  **Why this tier:** Package and consume collections of documents and build artifacts.

  **Work:** Expose only formats/operations native to the qualified Bun build. Define entry metadata, byte/file destinations, extraction roots, links, replacement and expanded-size limits. Do not silently add zip/tar formats unavailable in that API.

  **Done when:** Test native archive interoperability, duplicate names, escaping paths, links, truncation and bounded extraction in temporary trees. All common completion rules also pass.

  **Bun references:** [documentation 1](https://bun.sh/docs/runtime/archive).

- [ ] **B2-04 — Image decoding, transforms and encoding**

  **Can surface:** Library. **Depends:** B1-01.

  **Native basis:** Bun.Image.

  **Why this tier:** Prepare visual inputs and generated thumbnails without adding an image-processing dependency.

  **Work:** Integrate qualified native formats, metadata, resize/crop and encoding options as typed operations. Bound decoded dimensions and preserve explicit color/orientation behavior. No AI image generation or video stack is included.

  **Done when:** Known images and malformed/oversized inputs verify pixel/dimension/format results, allocation limits and native encoding. Record platform-specific format coverage. All common completion rules also pass.

  **Bun references:** [documentation 1](https://bun.sh/docs/runtime/image).

- [ ] **B2-05 — XML and streaming HTML transformation**

  **Can surface:** Library; streaming handler candidate. **Depends:** B1-05, B1-12.

  **Native basis:** Bun.XML and HTMLRewriter.

  **Why this tier:** Support structured documents and incremental HTML extraction/transformation.

  **Work:** Map qualified XML operations to typed data and expose HTMLRewriter handler capabilities with explicit async/error ownership. Keep parsing, extraction and safe rendering separate. Retain source/element locations where available.

  **Done when:** Exercise malformed markup, nested structures, handler errors, streaming cancellation and unsafe attribute/content transformations. No custom parser. All common completion rules also pass.

  **Bun references:** [documentation 1](https://bun.sh/docs/runtime/xml), [documentation 2](https://bun.sh/docs/runtime/html-rewriter).

- [ ] **B2-06 — DNS and TCP/TLS connections**

  **Can surface:** Library plus connection/event primitive candidate. **Depends:** B1-05.

  **Native basis:** Bun.dns, Bun.connect, Bun.listen.

  **Why this tier:** Enable native protocol clients and servers beyond HTTP.

  **Work:** Add DNS records and typed socket data/connect/drain/end/error lifecycles, TLS configuration and explicit framing left to application protocols. Compare a domain connection primitive against the existing event surface.

  **Done when:** Local DNS/socket tests cover partial reads, pressure, peer close, TLS errors and handler failures without unbounded buffers or leaked handles. All common completion rules also pass.

  **Bun references:** [documentation 1](https://bun.sh/docs/runtime/dns), [documentation 2](https://bun.sh/docs/runtime/tcp).

- [ ] **B2-07 — UDP datagrams**

  **Can surface:** Library plus event primitive candidate. **Depends:** B2-06.

  **Native basis:** Bun.udpSocket.

  **Why this tier:** Enable datagram protocols with semantics distinct from a reliable byte stream.

  **Work:** Expose typed address/datagram operations and receive handlers with packet size, ownership and close rules. Do not infer ordering/reliability from the TCP design.

  **Done when:** Loopback tests cover missing/reordered packets, truncation policy, invalid addresses and close during receive; document platform support. All common completion rules also pass.

  **Bun references:** [documentation 1](https://bun.sh/docs/runtime/udp).

- [ ] **B2-08 — Filesystem watching**

  **Can surface:** Event primitive candidate. **Depends:** B1-01, B1-05.

  **Native basis:** Implemented node:fs.watch APIs in Bun.

  **Why this tier:** Enable reactive local tooling and ingestion when files change.

  **Work:** Define watch scope, event normalization, coalescing and rescan behavior with explicit ownership. Evaluate a native declaration with attached event cases; do not force filesystem notifications into a misleading one-event match.

  **Done when:** Temporary-directory tests cover rename/delete/bursts, platform differences and watch closure. Specify limitations rather than promise lossless notifications. All common completion rules also pass.

  **Bun references:** [documentation 1](https://bun.sh/docs/runtime/nodejs-compat).

- [ ] **B2-09 — Timers and in-process scheduled jobs**

  **Can surface:** Schedule primitive candidate plus library. **Depends:** B1-05, B1-13.

  **Native basis:** Native timers and Bun.cron.

  **Why this tier:** Run periodic application work with inspectable lifecycle and failures.

  **Work:** Add scheduled callbacks with timezone/overlap/missed-tick/error/stop contracts. Distinguish process-local jobs from OS-persistent registration in B3-02. Compare a native schedule form against ordinary named callback operations.

  **Done when:** Fake-clock event fixtures plus bounded native timer tests cover DST/timezone cases, slow callbacks, failures and stop. No hidden external scheduler service. All common completion rules also pass.

  **Bun references:** [documentation 1](https://bun.sh/docs/runtime/cron), [documentation 2](https://bun.sh/docs/runtime/web-apis).

- [ ] **B2-10 — Worker-thread tasks and typed messaging**

  **Can surface:** Parallel-task/message primitive candidate. **Depends:** B1-05.

  **Native basis:** Bun Worker and implemented message/clone APIs.

  **Why this tier:** Enable CPU parallelism with explicit isolation instead of changing existing Promise coordination.

  **Work:** Qualify experimental lifecycle support before exposure. Define transferable/cloneable Can data, task entry, errors, messages and termination. Existing concurrent/race keeps its meaning; do not silently move captured resource handles across threads.

  **Done when:** Prove parallel CPU work, serialization errors, worker crash and termination with pinned-target tests. If runtime behavior is insufficient, report the specific blocker and keep the task pending. All common completion rules also pass.

  **Bun references:** [documentation 1](https://bun.sh/docs/runtime/workers).

- [ ] **B2-11 — Browser rendering and automation**

  **Can surface:** Library; session/event primitive candidate. **Depends:** B1-01, B1-05.

  **Native basis:** Bun.WebView.

  **Why this tier:** Enable native browser-assisted extraction and application testing.

  **Work:** Integrate the qualified page/navigation/inspection/rendering subset using owned browser sessions and typed results. Record external browser/OS requirements. This does not imply a Can browser compiler or arbitrary npm automation framework.

  **Done when:** Local-page fixtures cover navigation failure, wait/close, page events and supported output capture; browser process/resources are reaped. No live-site dependency for mandatory tests. All common completion rules also pass.

  **Bun references:** [documentation 1](https://bun.sh/docs/runtime/webview).

- [ ] **B2-12 — Operating-system secret storage**

  **Can surface:** Library. **Depends:** B1-08.

  **Native basis:** Bun.secrets.

  **Why this tier:** Use native credential stores for installed Can applications.

  **Work:** Add named service/account get/set/delete with explicit unavailable/denied/not-found outcomes. Preserve headless and platform distinctions; fixtures must not access real credentials.

  **Done when:** Test adapter behavior against fakes and isolated native-store entries where supported; no secret contents in diagnostics or source snapshots. All common completion rules also pass.

  **Bun references:** [documentation 1](https://bun.sh/docs/runtime/secrets).

- [ ] **B2-13 — Richer native SQL capabilities**

  **Can surface:** Library extensions. **Depends:** B1-02, B1-03, B1-05.

  **Native basis:** Qualified Bun.SQL and bun:sqlite capabilities.

  **Why this tier:** Grow beyond the initial typed CRUD subset without requiring feature parity through the lowest common denominator.

  **Work:** Inventory native-supported SQL value types, batch/prepared operations, savepoints and SQLite serialization/backup capabilities. Admit each with dialect-specific contracts; migration frameworks remain outside Bun-API integration. Do not promise a feature merely because the database server supports it.

  **Done when:** Each admitted operation has real-engine tests, typed null/value conversion, rollback/error cases and a per-dialect support matrix. All common completion rules also pass.

  **Bun references:** [documentation 1](https://bun.sh/docs/runtime/sql), [documentation 2](https://bun.sh/docs/runtime/sqlite).

- [ ] **B2-14 — Native semver, terminal presentation and sortable IDs**

  **Can surface:** Library. **Depends:** B1-13.

  **Native basis:** Bun.semver, Bun.color, Bun.stringWidth, Bun.randomUUIDv7.

  **Why this tier:** Complete developer-tool and presentation utilities after primary application capabilities.

  **Work:** Add native version parsing/comparison, color conversion, display width and UUIDv7 where absent. Specify malformed input, Unicode terminal width and clock/random fixture behavior. Keep existing UUIDv4 and text operations.

  **Done when:** Use edge vectors for prerelease versions, combining characters, color formats and repeated timestamps; no identity/ordering guarantees beyond the qualified native contract. All common completion rules also pass.

  **Bun references:** [documentation 1](https://bun.sh/docs/runtime/semver), [documentation 2](https://bun.sh/docs/runtime/color), [documentation 3](https://bun.sh/docs/runtime/utils).
