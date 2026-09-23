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

## SSE, multipart

Pending: pending-response writer pairs, validated SSE records, and
bounded generic multipart records (Jev `writer_pair` / `record_send`,
see
`docs/bun-integration/asap/evidence/consultations-b1-06/decision-audit.md`).
