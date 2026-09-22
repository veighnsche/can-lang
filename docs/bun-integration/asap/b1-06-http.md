# B1-06 — Broader HTTP and incremental request/response bodies

Status: prepared for ASAP execution; implementation and acceptance still pending. Dependencies: B1-05. Surface: Extend existing native forms; event/stream primitive candidate.

Extend the existing HTTP/fetch/server forms and typed request/response operations. Cover methods, routing, repeated headers, multipart, streaming, SSE, TLS and cancellation. Preserve normalized fetch failures and local native-request tests from the language-fix implementation; inspect its completed contract before adding another boundary.

## Where to implement

Paths are repository-relative; **new** means a proposed file/directory, not existing code. The shared wiring below is part of this task, not optional cleanup.

| Path | State | Responsibility |
|---|---|---|
| `runtime/platform/http.ts` | existing | Typed methods, requests, headers and stream response constructors. |
| `runtime/platform/router.ts` | existing | Methods, path matching and method-not-allowed behavior. |
| `runtime/platform/server.ts` | existing | Bun server lifecycle, TLS and disconnect handling. |
| `runtime/platform/form.ts` | existing | Multipart field/file projection and limits. |
| `runtime/transport/request.ts` | existing | Native request construction and owned request bodies. |
| `runtime/transport/body.ts` | existing | Bounded and incremental body handling. |
| `runtime/transport/named.ts` | existing | Named fetch body/result integration. |
| `compiler/internal/check/http.go` | existing | Checked HTTP configuration and contracts. |
| `tests/integration/http_test.go` | existing | Method/header/body cases. |
| `tests/integration/server_test.go` | existing | Streaming, TLS and shutdown. |
| `compiler/internal/catalogue/catalogue.json` | existing | Canonical public types, signatures, exact error identities, native lowering and assertion policy. |
| `compiler/internal/emit/program.go` | existing | Operation identity -> runtime target binding, imports and adapter construction. |
| `runtime/modules.json` | existing | Register each runtime module and exact import edges. |
| `distribution/target.json` | existing | Add actually required qualified native APIs; do not change the pinned runtime casually. |

Generated mirrors are produced with `make catalogue`: `compiler/internal/catalogue/generated.go`, `runtime/catalogue.ts`, `std/catalogue/README.md`. Never hand-edit them. See [shared integration contract](integration-contract.md) for specs, module boundaries, fixture ownership and completion rules.

## Proposed API contract

This is interface notation, **not accepted Can syntax or a claim that these operations compile today**. Resolve final names against the current catalogue and express them in existing Can declaration syntax; only primitive gates may add grammar.

```text
http routes for GET/HEAD/POST/PUT/PATCH/DELETE/OPTIONS
headers = ordered repeated fields with case-insensitive lookup
request body = one owned byte stream or bounded typed decode
response_stream(status, headers, body)
response_sse(events, options)
multipart fields preserve repeated names; files have owned bounded readers
```

## Implementation sequence

1. Inventory existing request/response operations and extend them, preserving native fetch/judge normalization semantics. Define redirect policy, HEAD body suppression, method fallback and duplicate-header behavior.
2. Keep body single-consumption explicit. Retain bounded whole-body convenience functions while using B1-05 for incremental reads/writes. Apply limits while consuming, including multipart boundaries and per-file/total caps.
3. Use Bun.serve, fetch, Request, Response and Headers; adapt immutable Can values at the boundary. Preserve Set-Cookie as separate fields instead of comma-joining it.
4. Implement HTTP method routing and typed body choices. Client disconnect cancels handlers/stream production under the correct owner; avoid continuing expensive work with no recipient.
5. Add SSE framing for id/event/data/retry with validation against line injection, multiline data, heartbeat policy and bounded queues. Native HTTP transport remains the engine.
6. Add TLS configuration from explicit certificate/key sources and test a local certificate chain; reject unsupported flags rather than ignoring them. Do not bake production credentials into examples.
7. Define failures before response publication versus after headers are sent. After publication, terminate/report the stream; never pretend a different status can still be sent.
8. Extend native-request fixtures to chunked input, repeated headers, disconnects and outgoing streaming bodies. Keep existing normalized failure identity and private provenance.

## Acceptance evidence

- Success: complete method table, duplicate query/header values, multipart text+binary files, incremental upload/download, SSE framing and local TLS.
- Rejected: invalid status/method/header, inconsistent body/status, multipart oversized input, invalid SSE fields, second read of consumed body.
- Runtime: slow client caps buffering; disconnect cancels; pre-header failure emits planned response; post-header failure closes correctly; server stop drains with a deadline.
- Generated output delegates native I/O to Bun.serve/fetch; no custom HTTP stack.

## Fixtures and local assertions

Add per-owner HTTP transcripts with request chunks, delays, close and expected output. Test real loopback servers in addition to deterministic fixtures. Native-request tests cover normalized errors and failure provenance; callback-only mocks do not qualify TLS, multipart or transport behavior.

## Native lowering sketch

This sketch describes the native operation, not a complete Can adapter. Real adapters must return the existing Completion representation, retain source origin and apply declared error boundaries.

```ts
return new Response(nativeOwnedReadable, {status, headers});
const server = Bun.serve({hostname, port, fetch: checkedOwnedHandler, tls});
// Request.signal and owner cancellation cooperate; publication is tracked once.
```

## Gates and limitations

Do not claim native FormData buffering is streaming multipart parsing. If Bun lacks incremental multipart primitives for the required contract, identify the supported bounded path and keep the unresolved streaming subtask visible rather than reimplementing a parser silently.

## Source and proof

- [Pinned native observations](evidence/native-probe-results.json) and [reproducer](evidence/native-probe.ts). Only listed probe behavior was executed.
- [Official Bun documentation](https://bun.sh/docs/runtime/http/server)
- [Official Bun documentation](https://bun.sh/docs/runtime/streams)
- [Official Bun documentation](https://bun.sh/docs/runtime/web-apis)

Do not check off this capability until its acceptance cases, local fixture behavior, generated TypeScript evidence and all required real-native runs are recorded. Use [execution queue](execution-queue.md) to continue immediately with the next unblocked capability.
