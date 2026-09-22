# ASAP execution queue

Start implementation as soon as a safe integration checkout is available. This is the Bun milestone, separate from LF01–LF21. Do not message, pause or overwrite the current implementer. All entries below are pending implementation; writing a plan does not complete a task.

Read the [readiness audit](readiness.md) before interpreting a prepared task as a settled design.

## How to use this queue

The default sequence follows dependencies. If a test service or a bounded design gate blocks one branch, record the exact missing evidence and immediately take another entry whose prerequisites are satisfied. Numeric order is a recommended execution path, not a requirement to sit idle behind unavailable MySQL. No capability is dropped from ASAP.

Begin SQL grammar qualification, the three-workflow event comparison, and Markdown callback inspection early as preparation spikes. They can run before their parent capability is fully wired. Complete API slices that do not depend on those gates: files, bounded processes, passwords/crypto, URLs/text/time and bounded document parsing. The stream integration portion of document parsing still depends on B1-05.

All implementation files are assigned in the linked capability plans. Shared catalogue/emitter/runtime wiring must be done in each slice. Avoid an enormous initial catalogue-only commit advertising APIs with no implementation; prefer small working vertical slices with native and Can tests.

## Capability order

| Order | Capability | Prerequisites | Completion evidence |
|---|---|---|---|
| 1 | [B1-01 — Files, directories, paths and globbing](b1-01-files.md) | Baseline only | Every acceptance row in plan + shared completion contract |
| 2 | [B1-04 — Bounded child-process execution](b1-04-processes.md) | Baseline only | Every acceptance row in plan + shared completion contract |
| 3 | [B1-05 — Streams and reusable lifecycle contracts](b1-05-streams-and-lifecycles.md) | B1-01, B1-04 | Every acceptance row in plan + shared completion contract |
| 4 | [B1-02 — SQLite integration and the shared multi-dialect SQL boundary](b1-02-sqlite-and-sql-boundary.md) | B1-01 | Every acceptance row in plan + shared completion contract |
| 5 | [B1-03 — MySQL integration](b1-03-mysql.md) | B1-02 | Every acceptance row in plan + shared completion contract |
| 6 | [B1-08 — Password and broader cryptographic operations](b1-08-crypto.md) | Baseline only | Every acceptance row in plan + shared completion contract |
| 7 | [B1-13 — Common URL, text, byte and time utilities](b1-13-common-utilities.md) | Baseline only | Every acceptance row in plan + shared completion contract |
| 8 | [B1-06 — Broader HTTP and incremental request/response bodies](b1-06-http.md) | B1-05 | Every acceptance row in plan + shared completion contract |
| 9 | [B1-07 — WebSocket client and server integration](b1-07-websockets.md) | B1-05, B1-06 | Every acceptance row in plan + shared completion contract |
| 10 | [B1-09 — Cookies and CSRF primitives](b1-09-cookies-csrf.md) | B1-06, B1-08 | Every acceptance row in plan + shared completion contract |
| 11 | [B1-10 — S3-compatible object storage](b1-10-s3.md) | B1-01, B1-05 | Every acceptance row in plan + shared completion contract |
| 12 | [B1-11 — Typed TOML, YAML, JSON5 and JSONL codecs](b1-11-document-formats.md) | B1-05 | Every acceptance row in plan + shared completion contract |
| 13 | [B1-12 — Markdown rendering and structured processing](b1-12-markdown.md) | Baseline only | Every acceptance row in plan + shared completion contract |

The [required filetree](filetree.md) applies to all entries. Before each DONE step, run the size guard and review protected hubs; no per-feature body belongs in program.go or the generic provider/owner.

## Ordered implementation checklist

The machine-readable [queue](execution-queue.json) has explicit prerequisites for every item. A `DONE` item is a required verification step, not a placeholder. Cross-capability dependencies reference verified completion.

- [ ] **ASAP-00** — Obtain an integration checkout containing completed required LF contracts, or isolate work without contacting or disturbing the current implementer. Read AGENTS.md and authoritative specs; record baseline.
- [ ] **ASAP-01** — Verify pinned runtime/archive and rerun supplied native probes; inspect current catalogue and register concrete SQL/event/HTML gate outputs. Record MySQL and S3 service readiness without using production credentials; unavailable services block only their own acceptance branches and do not block completion of this preflight.
- [ ] **ASAP-ARCH** — Record the integration base and implement the emitter responsibility extraction in filetree.md with existing behavior tests. Adopt per-feature binding/runtime modules, protected-hub rules and the changed-file size guard. Perform the prescribed SQL split when starting B1-02; do not refactor unrelated LF code.

- [ ] **B1-01.01** — Specify missing-vs-denied-vs-invalid-path failures, supported file kinds, overwrite/exclusive-create behavior, symlink following, recursive removal and ordering. Return absence only for actual missing paths, not permission failures.
- [ ] **B1-01.02** — Implement bounded chunk reads; check each chunk before accumulating. Decode text with fatal UTF-8. Copy native views before exposing immutable Can bytes. A stat size check is an optimization, not enforcement against a growing file.
- [ ] **B1-01.03** — Use Bun.file/Bun.write for applicable operations and node:fs/promises for directories, lstat, rename, copy and removal. Exclusive creation must use an atomic native flag, not exists-then-write. Do not call replacement writes atomic unless the implementation and tests support that promise.
- [ ] **B1-01.04** — Use node:path and Bun.Glob; normalize returned paths consistently, impose result limits during enumeration, and sort only if deterministic ordering is in the contract. Keep URL conversion separate from filesystem normalization.
- [ ] **B1-01.05** — Add catalogue signatures, errors, immutable metadata and emitter bindings. Fixture-capable effect calls must use the native-boundary provider before touching disk.
- [ ] **B1-01.06** — Add owned streaming readers/writers once B1-05 settles the lifecycle; bounded operations can ship earlier. Test short writes, failure cleanup and release of locks/handles.
- [ ] **B1-01-DONE** — Complete the capability acceptance/fixture/native-lowering evidence, authoritative docs and maintained example; verify real-native tests ran, then mark the parent complete.
- [ ] **B1-04.01** — Define binary output, explicit UTF-8 conversion, exit-vs-signal representation, environment inheritance opt-in and path resolution. Validate durations and byte caps before spawning.
- [ ] **B1-04.02** — Start native Bun.spawn with arrays, explicit cwd/env and piped I/O. Register ownership immediately after spawn, before waiting for any output.
- [ ] **B1-04.03** — Drain stdout and stderr concurrently while writing stdin with backpressure. Independent per-pipe limits prevent one blocked pipe deadlocking the other. Increment counts before retaining bytes.
- [ ] **B1-04.04** — On timeout, overflow, input error or owner cancellation: stop producing stdin, signal child, wait grace, escalate if supported, await exited and finish/cancel pipe drains under a bounded shutdown deadline.
- [ ] **B1-04.05** — Specify direct-child versus process-tree guarantees. Add an owned-grandchild reproducer; do not promise no descendant survives merely because child.kill() completed. Qualify a platform-native group strategy or make descendants an explicit unsupported contract.
- [ ] **B1-04.06** — Map spawn/permission/I/O errors separately from a successful nonzero status. Preserve original timeout/overflow failure when cleanup also fails.
- [ ] **B1-04.07** — Add catalogue and fixture provider binding. Pass incremental reader ownership to B1-05 instead of exposing mutable native subprocess objects.
- [ ] **B1-04-DONE** — Complete the capability acceptance/fixture/native-lowering evidence, authoritative docs and maintained example; verify real-native tests ran, then mark the parent complete.
- [ ] **B1-05.01** — Write the three programs in library/callback, per-domain primitive and shared-event styles. Include grouped state, normal completion, handler failure, external failure, cancellation, early stop, backpressure and locally attached fixtures in each. Record diagnostic spans and generated-native sketches.
- [ ] **B1-05.02** — Choose the smallest surface justified by complete programs. Freeze its grammar and behavior in the authoritative specs at implementation time; do not wait to start independent library work. New parser/checker/IR files may be introduced only for an actually selected primitive.
- [ ] **B1-05.03** — Specify state transitions open -> closing -> closed, exactly one terminal publication, no new callbacks after closure, and observing late native rejections. Error events and handler-thrown failures remain distinguishable.
- [ ] **B1-05.04** — Prefer native pull/backpressure where available. Serialize handlers for one owned stream/session unless concurrency is explicitly requested. Await each handler before delivering its next item; bound any native event queue that cannot pause.
- [ ] **B1-05.05** — Register opaque resource contents with existing ownership/capture evidence. Reject use-after-close/foreign-owner consistently. Copy bytes at the boundary; no reusable native chunk alias crosses into immutable state.
- [ ] **B1-05.06** — Implement cancellation propagation to readers/writers/AbortController, release locks, bounded shutdown, and cleanup failure reporting without overwriting the primary completion.
- [ ] **B1-05.07** — Make fixtures drive the same dispatch/owner path as native events. Attach deadlines to assertions; a transcript that never ends cannot hang the build.
- [ ] **B1-05.08** — Wire file streams and process stdout/stderr first, then HTTP/S3/WebSocket. Do not invent a universal scheduler or resource system in parallel with owner.ts.
- [ ] **B1-05-DONE** — Complete the capability acceptance/fixture/native-lowering evidence, authoritative docs and maintained example; verify real-native tests ran, then mark the parent complete.
- [ ] **B1-02.01** — Create a three-dialect contract table covering placeholders, null, integers, decimals, booleans, text, blobs, dates, affected rows and insert IDs. Reject unsupported mappings; never silently cast an exact Can integer through Number.
- [ ] **B1-02.02** — Introduce dialect into manifest and checked descriptor identity. Keep PostgreSQL parsing in its backend. Do not feed MySQL backticks or SQLite syntax through libpg_query and call the result dialect validation.
- [ ] **B1-02.03** — Complete the SQL admission gate in decisions.md before declaring frontend support. Compare a pinned maintained dialect parser with native prepare qualification; preserve compile-time syntax guarantees or explicitly reconcile the contract. Runtime-only prepare does not prove offline build validation. A regex/semicolon splitter is forbidden as a substitute parser.
- [ ] **B1-02.04** — Continue static-string/parameter segment binding through native tag templates. No user values in SQL text. Prove repeated and out-of-order parameters, comments/literals and parameter-like text. Preserve row-limit/cardinality contracts per dialect instead of appending PostgreSQL-specific syntax blindly.
- [ ] **B1-02.05** — Open Bun.SQL with explicit sqlite adapter, filename and safeIntegers:true; qualify all numeric conversions. Implement memory lifetime, file open modes and close. File DB tests must reopen with a new connection. Never infer rollback from an in-memory disappearing connection.
- [ ] **B1-02.06** — Implement transaction begin/commit/rollback on the owned connection; a body can await. Test domain failure, standard failure, cancellation and commit failure independently, preserving primary occurrence and cleanup diagnostics.
- [ ] **B1-02.07** — Qualify lock contention, busy timeout and event-loop behavior. Bound connection acquisition and release. If the target cannot interrupt a long native SQLite operation, describe that limitation and require an enforceable execution design before promising hard query deadlines.
- [ ] **B1-02.08** — Complete PostgreSQL plus SQLite evidence, then pass the same descriptor/value tests to B1-03. Do not mark the three-engine milestone done before MySQL.
- [ ] **B1-02-DONE** — Complete the capability acceptance/fixture/native-lowering evidence, authoritative docs and maintained example; verify real-native tests ran, then mark the parent complete.
- [ ] **B1-03.01** — Reuse B1-02 frontend/value tables; implement MySQL-specific options with explicit TLS and credentials. Never log credential-bearing URLs.
- [ ] **B1-03.02** — Use Bun.SQL native prepared bindings. Distinguish identifiers from values; dynamic identifiers require a separate checked operation, not interpolation into a parameter slot.
- [ ] **B1-03.03** — Qualify BIGINT signed/unsigned extremes and DECIMAL. If the driver emits decimal strings, validate and preserve them through an explicit exact representation; do not coerce to floating point.
- [ ] **B1-03.04** — Define DATETIME without timezone separately from instants; set test session timezone explicitly. Exercise bit/boolean, binary and NULL. Reject unsupported native result objects before exposing data.
- [ ] **B1-03.05** — Classify native errors by verified codes for connection/auth/constraint/deadlock/timeout; preserve private cause. Unknown or adapter-programming errors remain standard failures, not generic mysql errors.
- [ ] **B1-03.06** — Run shared typed CRUD/rollback workload across all three engines and dialect-specific cases against a provisioned local MySQL service. Do not add automatic retries to non-idempotent statements.
- [ ] **B1-03.07** — Add clean setup/teardown, version reporting and test-service configuration. Missing service is a visible qualification blocker, never a passing integration result.
- [ ] **B1-03-DONE** — Complete the capability acceptance/fixture/native-lowering evidence, authoritative docs and maintained example; verify real-native tests ran, then mark the parent complete.
- [ ] **B1-08.01** — Inventory existing SHA/random operations; add names without duplicate functionality. Specify Argon2id cost presets and validation; retain native encoded hashes so verification can recognize recorded parameters.
- [ ] **B1-08.02** — Use asynchronous Bun.password APIs to avoid synchronous event-loop blocking. Test wrong password, invalid hash and excessive-cost policy distinctly.
- [ ] **B1-08.03** — Choose the initial WebCrypto suite explicitly: AES-GCM, HMAC-SHA-256, and Ed25519 if target qualification succeeds. Fix nonce/tag lengths and supported key usages; reject other algorithms rather than silently falling back.
- [ ] **B1-08.04** — Keep CryptoKey in a private WeakMap-backed opaque value, nonextractable by default. Provide explicit import/export only for admitted formats/usages. No raw key material in fixture reports, logs or generic object projections.
- [ ] **B1-08.05** — Use native CryptoHasher/SubtleCrypto; add only bytes ownership, exact options, declared failure translation and opaque wrapping. Never implement cipher/HMAC primitives in Can.
- [ ] **B1-08.06** — Separate generated nonce convenience from explicit nonce encryption. Document nonce uniqueness as an encryption contract; generation is native randomness, not a guarantee of caller discipline.
- [ ] **B1-08.07** — Add native interoperability vectors and tampering tests. Ensure one adapter cannot use an encryption-only key for signing.
- [ ] **B1-08-DONE** — Complete the capability acceptance/fixture/native-lowering evidence, authoritative docs and maintained example; verify real-native tests ran, then mark the parent complete.
- [ ] **B1-13.01** — Diff desired operations against the current catalogue and list omissions before adding names. Reuse existing normalize/scalars/graphemes/join and clock APIs.
- [ ] **B1-13.02** — Use URL/URLSearchParams with typed immutable projections. Preserve repeated query keys and distinguish URL decoding from form decoding. Define relative URL resolution and admitted schemes.
- [ ] **B1-13.03** — Use native RegExp; reset or hide lastIndex and mutable state. Specify flags, capture absence, Unicode index units, zero-length match advancement and max result count.
- [ ] **B1-13.04** — Use native TextEncoder/TextDecoder and qualified Buffer/Web byte encoders. Reject malformed hex/base64 under the documented strict policy instead of silently truncating. Copy mutable views at both edges.
- [ ] **B1-13.05** — Represent an instant separately from local date/time and duration. Range-check exact Can int before Number conversion for Date. Use Intl for explicit locale/zone formatting; deterministic tests specify both.
- [ ] **B1-13.06** — Define DST ambiguity/nonexistence if local-time-to-instant conversion is exposed. Do not quietly pick native normalization as the Can contract. Restrict initial conversion to well-defined instant/offset inputs until this is specified.
- [ ] **B1-13.07** — Add known vectors and negative compiler examples. Keep wall time/randomness in their existing assertion boundary; pure parsing/formatting remains real.
- [ ] **B1-13-DONE** — Complete the capability acceptance/fixture/native-lowering evidence, authoritative docs and maintained example; verify real-native tests ran, then mark the parent complete.
- [ ] **B1-06.01** — Inventory existing request/response operations and extend them, preserving native fetch/judge normalization semantics. Define redirect policy, HEAD body suppression, method fallback and duplicate-header behavior.
- [ ] **B1-06.02** — Keep body single-consumption explicit. Retain bounded whole-body convenience functions while using B1-05 for incremental reads/writes. Apply limits while consuming, including multipart boundaries and per-file/total caps.
- [ ] **B1-06.03** — Use Bun.serve, fetch, Request, Response and Headers; adapt immutable Can values at the boundary. Preserve Set-Cookie as separate fields instead of comma-joining it.
- [ ] **B1-06.04** — Implement HTTP method routing and typed body choices. Client disconnect cancels handlers/stream production under the correct owner; avoid continuing expensive work with no recipient.
- [ ] **B1-06.05** — Add SSE framing for id/event/data/retry with validation against line injection, multiline data, heartbeat policy and bounded queues. Native HTTP transport remains the engine.
- [ ] **B1-06.06** — Add TLS configuration from explicit certificate/key sources and test a local certificate chain; reject unsupported flags rather than ignoring them. Do not bake production credentials into examples.
- [ ] **B1-06.07** — Define failures before response publication versus after headers are sent. After publication, terminate/report the stream; never pretend a different status can still be sent.
- [ ] **B1-06.08** — Extend native-request fixtures to chunked input, repeated headers, disconnects and outgoing streaming bodies. Keep existing normalized failure identity and private provenance.
- [ ] **B1-06-DONE** — Complete the capability acceptance/fixture/native-lowering evidence, authoritative docs and maintained example; verify real-native tests ran, then mark the parent complete.
- [ ] **B1-07.01** — Specify client/server event differences, frame types, connection handshake failure versus established-session failure, close code/reason validity and maximum message sizes.
- [ ] **B1-07.02** — Integrate Bun.serve upgrade and native WebSocket client. Authentication/authorization happens before server.upgrade; a rejected handshake remains ordinary HTTP.
- [ ] **B1-07.03** — Use per-session grouped state under serialized checked handlers. Native server hooks are installed once, with typed session data selecting the owner. Do not share mutable application state between sessions by accident.
- [ ] **B1-07.04** — Translate native send/backpressure results deliberately. Server drain can resume queued sends; the client API may need a different bounded strategy. Never claim the client exposes the same drain semantics as ServerWebSocket.
- [ ] **B1-07.05** — Handle cancellation, server stop, failed opening, terminal close and late callbacks through B1-05 ownership. Avoid automatic reconnect/replay; it changes delivery semantics.
- [ ] **B1-07.06** — Expose explicit protocol/subprotocol/TLS options supported by the target. Preserve close provenance; arbitrary event objects must not leak into Can.
- [ ] **B1-07.07** — If the primitive wins, add parser/format/check/IR/emitter support and attached event assertions together. Keep library connection operations available for straightforward operations.
- [ ] **B1-07.08** — Qualify server publication/subscription operations separately if included: distinguish socket publish excluding sender from server publish including all; do not conflate with a general message broker.
- [ ] **B1-07-DONE** — Complete the capability acceptance/fixture/native-lowering evidence, authoritative docs and maintained example; verify real-native tests ran, then mark the parent complete.
- [ ] **B1-09.01** — Specify duplicate request-cookie lookup behavior, malformed input, name/value encoding and cookie attributes. Enforce admitted SameSite combinations and cookie prefix rules if the API claims them.
- [ ] **B1-09.02** — Lower parsing/serialization to Bun.CookieMap/Bun.Cookie and immutable projections. Keep multiple Set-Cookie fields separate. Do not manually concatenate unvalidated header strings.
- [ ] **B1-09.03** — Require caller-supplied CSRF secret and nonempty session identifier. Verify using matching algorithm/encoding and explicit age policy; do not rely on a per-thread secret that changes at restart.
- [ ] **B1-09.04** — Distinguish invalid/expired/mismatched tokens returning false from invalid configuration. No secret or raw token in assertion failure diagnostics.
- [ ] **B1-09.05** — Integrate safe response construction and deletion using matching path/domain. Verify serialization against actual response headers.
- [ ] **B1-09.06** — Document application responsibility for deriving the session from authenticated state and checking relevant unsafe HTTP requests. Token cryptography alone is not session authentication.
- [ ] **B1-09-DONE** — Complete the capability acceptance/fixture/native-lowering evidence, authoritative docs and maintained example; verify real-native tests ran, then mark the parent complete.
- [ ] **B1-10.01** — Qualify target-native methods and options for read/write/delete/stat/list/presign/multipart against official docs and the exact binary. Constructor presence alone proves none of these operations.
- [ ] **B1-10.02** — Separate object key semantics from local paths: slash is data, not local traversal. Require explicit endpoint/region/bucket and credential sourcing; never expose credentials in error payloads.
- [ ] **B1-10.03** — Implement immutable metadata and typed page continuation. Avoid collecting an entire bucket; enforce per-page/result limits and preserve opaque continuation values.
- [ ] **B1-10.04** — Use B1-05 streams for large transfers; enforce byte and deadline budgets during transfer, cancel on handler failure and observe native rejections.
- [ ] **B1-10.05** — Qualify native multipart behavior, abort cleanup and retry semantics. Do not write an S3 signing algorithm or multipart protocol implementation. If the native API lacks an operation, record that exact target gap before changing scope.
- [ ] **B1-10.06** — Represent presigned result as URL plus required method/headers and expiration metadata. Treat signed URLs as sensitive, redacting them in reports.
- [ ] **B1-10.07** — Run native integration against a provisioned local S3-compatible endpoint: create isolated bucket/prefix, verify bytes and metadata, delete only those test objects.
- [ ] **B1-10-DONE** — Complete the capability acceptance/fixture/native-lowering evidence, authoritative docs and maintained example; verify real-native tests ran, then mark the parent complete.
- [ ] **B1-11.01** — Define a format matrix: accepted scalar kinds, null/optional mapping, dates/times, duplicate keys, aliases/merges, multiple documents, nonfinite floats, integers and root shapes. Use native options only when verified; default behavior is not uniform across parsers.
- [ ] **B1-11.02** — Create typed projection sharing Can schema rules and immutable data constructors, with per-format scalar conversion policy. Native numbers used as ints must be finite safe integers; document that this is parsed-value semantics, not proof of every original fractional token.
- [ ] **B1-11.03** — Keep the existing JSON exact-integer decoder intact. For strict JSONL, prefer bounded line framing plus the existing native-backed exact JSON decode path per record when that satisfies the selected JSONL contract. Framing lines is not implementing JSON grammar.
- [ ] **B1-11.04** — If using Bun.JSONL.parseChunk, inspect error, read and done and reject malformed/truncated final input. Never use Bun.JSONL.parse prefix success as complete validation. Specify blank lines, CRLF, final line without newline and multiline JSON acceptance explicitly.
- [ ] **B1-11.05** — Incrementally decode UTF-8 across chunk boundaries; cap bytes per record, record count and total retained output. Await each typed handler. Retain only the incomplete suffix, not every prior chunk.
- [ ] **B1-11.06** — For YAML/TOML native special objects (dates, aliases, maps), project only admitted forms and reject cycles/unsupported values. Node/depth checks after parsing do not protect native parse CPU/memory; a preparse byte cap is still necessary.
- [ ] **B1-11.07** — Test duplicate behavior empirically. If exact duplicate rejection is promised but native metadata cannot support it, keep that mode explicitly blocked rather than writing an ad hoc YAML/JSON5 grammar.
- [ ] **B1-11.08** — Add declared syntax/type/limit errors with safe paths and native source locations when trustworthy. Preserve source offsets through framing.
- [ ] **B1-11-DONE** — Complete the capability acceptance/fixture/native-lowering evidence, authoritative docs and maintained example; verify real-native tests ran, then mark the parent complete.
- [ ] **B1-12.01** — Qualify Bun.markdown.html and render callback inputs/order. Do not invent a token/AST API because callbacks exist; record whether callbacks receive source text, escaped text or already-rendered children.
- [ ] **B1-12.02** — Return native HTML as str with an explicit raw-HTML policy. Prevent implicit conversion of that string into html::safe or a trusted HTTP HTML response.
- [ ] **B1-12.03** — Prototype safe rendering through native callbacks using private trusted construction. Disable raw HTML blocks/spans; enforce URL scheme and attribute rules; preserve escaping exactly once.
- [ ] **B1-12.04** — Audit native callback completeness before claiming safe mode. If callbacks cannot expose enough structured information for the existing builder, ship string rendering and keep safe mode blocked with a reproducer; never regex-sanitize HTML or brand it safe.
- [ ] **B1-12.05** — Extend the safe HTML element allowlist only for required Markdown structures such as code/pre, with precise attribute policy. Reuse existing node/URL validation rather than duplicate a sanitizer.
- [ ] **B1-12.06** — Specify options, input/output budgets and callback failure bounds. Native synchronous parsing cannot be interrupted by a JavaScript timeout; do not promise a hard per-render deadline without an execution strategy.
- [ ] **B1-12.07** — Add ordinary display, safe HTTP response and structured-renderer examples, with attached assertions that execute native parsing.
- [ ] **B1-12-DONE** — Complete the capability acceptance/fixture/native-lowering evidence, authoritative docs and maintained example; verify real-native tests ran, then mark the parent complete.
- [ ] **ASAP-FINAL** — Run required compiler/runtime/bundle and real-service suites; reconcile all thirteen contracts and publish a completion report with generated-native evidence and no hidden skips.

## Delivery milestones

- First usable tools: file/path/glob operations, bounded subprocesses, crypto and common utility slices. Do not require all event syntax to finish first.
- Native lifecycles: stream/handler ownership and fixtures, incremental processes, then HTTP and WebSocket integration.
- Three-engine data: SQLite and MySQL plus continuing PostgreSQL acceptance. A PostgreSQL-only result does not satisfy this milestone.
- Application completeness: cookies/CSRF, S3, all four document formats and Markdown with explicit HTML trust boundaries.
- Final acceptance: all thirteen `DONE` items and `ASAP-FINAL`; missing real service/native proof remains incomplete.

Do not infer dates or effort guarantees from ASAP. The instruction is to implement without unnecessary planning loops, preserve correctness, and move to the next available work item whenever a branch is blocked.
