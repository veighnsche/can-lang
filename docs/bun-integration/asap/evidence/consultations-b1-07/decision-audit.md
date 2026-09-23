# B1-07 WebSocket-surface decision audit

Three independently rewritten packets asked four identical questions over
the G-EVENT pull ruling, the HTTP route/dispatch model and the pinned
Bun 1.4.2 probe ledger (ws_client, ws_upgrade, ws_send_close,
ws_backpressure, ws_close_provenance, ws_tls_client). All prose
(state, instructions, criteria) differs across packets; question keys,
option keys, identifiers and measured facts are stable. Verified
mechanically before dispatch (0 identical prose fields of 51
comparisons; 23 fact tokens present in every state). Model:
`jev-1.13.0` via `jev-latest`, three HTTP 200 rounds, 4224 input /
507 output tokens total.

## Results

| Question | Round 1 | Round 2 | Round 3 |
|---|---|---|---|
| surface | reader 0.50 (conf 0.25) | callback 0.45 (conf 0.17) | reader 0.81 (conf 0.72) |
| upgrade | marked_route 0.52 (conf 0.29) | accept_op 0.75 (conf 0.63) | accept_op 0.50 (conf 0.25) |
| backpressure | drain_event 0.63 (conf 0.45) | drain_event 0.82 (conf 0.73) | drain_event 0.60 (conf 0.40) |
| tls_client | explicit_insecure 0.99 (conf 0.98) | explicit_insecure 0.94 (conf 0.90) | explicit_insecure 0.99 (conf 0.99) |

Requests, responses and SHA metadata: `request-N.json`,
`response-N.json`, `response-N.metadata.json` in this directory.

## Agreement (accepted as advice)

- tls_client `explicit_insecure` 3/3 at high confidence: `ws::connect`
  takes an explicit insecure flag defaulting to secure, so loopback
  and self-signed testing stays reachable while OS-trust verification
  remains the default. `no_wss` scores 0.00 in every round.
- backpressure `drain_event` 3/3: server sessions surface a `drain`
  event that resumes queued sends; the client keeps fail-fast because
  its API offers no drain hook. `await_drain` never leads (max 0.30).
- surface `reader` 2/3: connect and accept vend a
  `stream::reader<ws::event>` for inbound flow, reusing the qualified
  B1-05 close/cancel/terminal machinery and fixture transcripts; sends
  stay on the opaque session handle. Round 2 preferred `callback`
  0.45 over `reader` 0.42 at confidence 0.17, a near-tie inside one
  low-confidence round; callbacks would also contradict the G-EVENT
  pull ruling the packets carried, and would need a new
  handler-registration surface instead of reusing `read_many`.
- upgrade `accept_op` 2/3: the route handler upgrades its own request
  through `ws::accept`; the adapter resolves the stored native request
  and server, and dispatch then withholds the HTTP reply. Rounds 1
  and 3 are near-ties (0.52/0.47 and 0.50/0.48) at low confidence, so
  this row was decided on carried evidence rather than the margin:
  B1-07.02 requires authentication before upgrade with a rejected
  handshake remaining ordinary HTTP, which `accept_op` expresses
  directly in handler code (return 401/403 instead of accepting),
  while `marked_route` would need dispatch to upgrade before auth
  with a new handler arity. The deferred-upgrade probe (upgrade after
  `await` succeeds) removes the timing objection. `separate_serve`
  never exceeds 0.02: it abandons router and auth integration.

## Selected record

`ws::connect` / `ws::accept` vend `{session, events}` where `events`
is a `stream::reader<ws::event>`; `ws::send_text`, `ws::send_bytes`
and `ws::close` take the session. Server sessions emit a `drain`
event; client sends fail fast on saturation. Close codes and reasons
validate adapter-side on both ends (the server native close validates
nothing). Catalogue additions only; no new grammar.
