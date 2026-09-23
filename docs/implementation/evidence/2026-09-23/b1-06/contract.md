# B1-06 HTTP contract confirmation

Target: Bun 1.4.2 `Bun.serve` + fetch. Every row below is an executed
observation (unit, loopback or integration run) unless marked docs.
Catalogue additions only: no new grammar. Parts landed per slice; this
file grows until the capability closes.

## Method table and dispatch

| Aspect | Contract |
|---|---|
| Methods | exact-match routes for `GET POST PUT PATCH DELETE OPTIONS HEAD` via `http::route_*`; no implicit aliasing (HEAD never falls back to GET) |
| Unknown method | fixed `405` with sorted `Allow` listing the mounted methods (e.g. `DELETE, GET, HEAD, OPTIONS, PATCH, POST, PUT`) |
| Unknown path | fixed `404`; trailing-slash and normalization aliases never match |
| Duplicates | same method+path twice is `duplicate_route`; distinct spellings normalizing together are `ambiguous_route` |
| Redirects | the server never redirects implicitly; the private fetch client uses `redirect: "manual"` (transport/fetch.ts) |

## HEAD and headers

| Aspect | Contract |
|---|---|
| HEAD suppression | dispatch strips the body and keeps the entity length the complete value produced (`content-length` set explicitly); Bun also strips natively, so the adapter rule holds on every path |
| Duplicate fields | non-cookie duplicates arrive comma-joined (`x-multi: 1, 2`, pinned); `Set-Cookie` stays split into separate ordered fields in both directions (observed in `Headers.entries()`) |
| Forbidden response fields | content/hop-by-hop/`hx-*` names, bad tokens and lone surrogates fail `make_server_headers` with `invalid_header` |

## TLS

| Aspect | Contract |
|---|---|
| Config | `http::make_tls_config(cert, key)` validates PEM structure: the cert input must tile into one or more `CERTIFICATE` blocks, the key into exactly one `PRIVATE KEY`, `RSA PRIVATE KEY` or `EC PRIVATE KEY` block; payloads must carry base64; inputs over 1 MiB or non-UTF-8 fail |
| Serve | `http::server_start_tls(config, router, tls)` serves HTTPS; DER the runtime cannot parse fails the start with `bind_failed{address}` |
| Verification | a fresh openssl chain (RSA 2048, SAN `IP:127.0.0.1`) serves loopback; `fetch` with `{tls:{ca}}` verifies it (200), default fetch rejects (self-signed) |
| Flags | the config carries no cipher/version flags, so none can be silently ignored; per-connection handshake failures stay inside Bun's error path and surface as fixed 500s |
| Credentials | chains are generated per test run into tempdirs; nothing credential-shaped is committed or baked into examples |

## Bodies and incremental reads

| Aspect | Contract |
|---|---|
| Selection | routes buffer unless wrapped in `http::route_stream`; the server peeks (`routeKind`) before snapshotting, and anything the peek cannot prove keeps drain-first ingress with the pre-dispatch 413 (Jev `route_marked`) |
| Lazy ingress | stream routes skip the pre-read (`bodyUsed` false until first access); oversize surfaces in-handler as `body_limit{server cap}`, wire failures as `invalid_request{reason:"body_read"}` |
| Consumption | buffered reads repeat from cache; `request_body_stream` opens the one-shot reader (live wire bytes, or replayed bytes when buffered first); buffered-after-live and second open fail `invalid_request{reason:"body_consumed"}` (Jev `buffered_repeatable`) |
| Reader | `stream::reader<bytes::buffer>` with per-chunk `max_chunk` (`body_limit` when < 1); reads flow through `stream::read_many`/`close_reader` with stream failures |
| Disconnect | a dropped client rejects the pending read with `AbortError`, mapped to `stream::read_failed{reason:"aborted"}` on both read and write sides |
| Abandon | dispatch cancels an unread live body (or settles an open reader's native pull) before the per-request scope drains; handlers are not preempted, their stream IO fails fast |
| Scope | each request dispatches in a child scope of the server scope, so readers die at request end with no server-lifetime leak |

## Response streams

| Aspect | Contract |
|---|---|
| Shape | `http::response_stream(status, headers)` builds a pending response; `http::response_writer` vends its `stream::writer` exactly once (second vend: `invalid_request{reason:"writer_taken"}`; non-stream response: `invalid_request{reason:"not_streaming"}`) (Jev `writer_pair`) |
| Model | produce-then-serve: the handler fills a 1 MiB bounded byte queue, closes the writer, and returns the pending response; Bun streams the queue after dispatch. Handlers never outlive dispatch, so no write observes a disconnect |
| Flow control | short writes only (`write_some` returns accepted bytes, 0 when full); writes never block, so production cannot deadlock against an unreturned response |
| Lifecycle | untaken writers serve empty; taken-but-unclosed writers end at request-scope drain (scope-managed by design); stream responses convert once (reuse is a usage violation); HEAD serves stream responses empty with no content-length |

## SSE

| Aspect | Contract |
|---|---|
| Shape | fixed `http::sse_event{data, event, id, retry}` record, empty means absent; `http::response_sse` builds the pending event-stream response; `http::sse_send`/`http::sse_comment` append through the vended writer (Jev `record_send`) |
| Framing | field order event, id, retry, then data lines; data splits on CR/LF/CRLF into `data:` lines; empty data emits no data line so retry/id-only blocks never dispatch; comment lines use `: text` (heartbeat support, no automatic heartbeats) |
| Violations | CR/LF in event/id/comment and non-digit retry fail `invalid_request` (`sse_event`, `sse_id`, `sse_retry`, `sse_comment`); frames are atomic (a frame that cannot fit the remaining queue writes nothing and fails `body_limit{1048576}`) |
| Writers | `sse_send`/`sse_comment` require an HTTP response writer (`not_streaming` otherwise) and return accepted bytes; sink failures map to `stream::write_failed` like `write_some` |

## Multipart

| Aspect | Contract |
|---|---|
| Shape | `http::request_multipart(request, max_bytes, max_file_bytes)` returns a fixed `http::multipart_form{fields, files}`: repeated field names stay as separate ordered entries; files carry `{name, filename, content_type, content}` with binary-safe owned bytes |
| Filenames | `filename=""` (empty file inputs) yields a file with empty filename; parts without filename are text fields; quoted escapes (`\"`, `\\`) unescape |
| Framing | strict CRLF; preamble ignored, epilogue ignored, transport padding on the close line allowed; missing/invalid boundary fails `multipart_boundary`, malformed framing fails `multipart_frame`, part content-types validate strict but store raw, nested `multipart/*` parts fail `multipart_nested` (flat only, documented) |
| Text | field values and header lines decode UTF-8 fatally; failures surface as `codec::invalid_data{path:"multipart",reason:"utf8"}` |
| Bounds | parses the already-budgeted buffered body (buffered ingress or first buffered access on live routes); each file's decoded bytes must also fit `max_file_bytes`, else the whole parse fails `body_limit{max_file_bytes}` (`nonnegative`/`nonpositive` on negative caps); fields stay under the total cap only |
| Files as readers | true incremental multipart is out of scope (Bun buffers via `formData()`); the supported bounded path is the whole-body parse above, and file contents are owned bounded bytes within that budget |
