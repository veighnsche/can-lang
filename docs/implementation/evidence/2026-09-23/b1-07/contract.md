# B1-07 WebSocket contract confirmation

Target: Bun 1.4.2 `Bun.serve` websocket hooks + `WebSocket` client.
Every row below is an executed observation (unit, loopback or
integration run) unless marked docs. Catalogue additions only: no new
grammar. G-EVENT pull applies: inbound flow is a
`stream::reader<ws::event>`; sends and close take the opaque
`ws::session`. Surface selection audited in
`docs/bun-integration/asap/evidence/consultations-b1-07/decision-audit.md`.

## Surface and events

| Aspect | Contract |
|---|---|
| Shape | `ws::connect` / `ws::accept` vend `ws::connection{session, events, protocol}`; `events` is a `stream::reader<ws::event>` served by `read_many` with uniform batch-fill semantics (trickling consumers read 1; the empty batch follows the close event) |
| Leaves | `ws::text{text}`, `ws::binary{data}`, `ws::drain{}`, `ws::closed{code, reason}`; no open leaf (connect resolves after the handshake, accept implies a live socket) and no failure leaf (failures fail the read as `read_failed`) |
| Drain | server sessions only, coalesced to one outstanding, bypasses the data cap; clients never emit drain and fail sends fast instead |
| Binary | server hooks deliver `Buffer`, the client sets `binaryType=arraybuffer`; both copy into owned bytes at the hook, so native aliases never reach Can |
| Text | strings cross as-is; caps count UTF-8 bytes |

## Bounds

| Aspect | Contract |
|---|---|
| Caps | `max_message_bytes` 1..67108864 per message both directions, `max_queued_events` 1..1024 data events, `max_send_bytes` 1..67108864 client buffered bytes, `deadline_ms` 1..2147483647 connect wait; violations fail `ws::limit_exceeded` echoing the bad value |
| Oversize inbound | text/binary over `max_message_bytes` fails the read `read_failed{message_too_large}`, terminal for the reader; close still succeeds afterwards |
| Queue overrun | data beyond `max_queued_events` with no read pending fails the pump `read_failed{queue_overrun}` and drops the queue; drain (coalesced) and close bypass the cap so the terminal is never lost to a full queue |
| Client send cap | `bufferedAmount + bytes > max_send_bytes` fails `ws::send_blocked`; accepted counts are bytes queued locally |
| Server send | native returns queued bytes, `-1` at saturation fails `ws::send_blocked`, `0` means empty message on an open socket; sends are message-atomic |

## Close

| Aspect | Contract |
|---|---|
| Validation | adapter-side both ends: 1000-1014 excluding 1004/1005/1006, or 3000-4999; reason well-formed UTF-8 of at most 123 bytes; violations fail `ws::invalid_close{code,reason}` and the native server close (which validates nothing) is never reached |
| Provenance | the initiating side observes its own code/reason; the peer observes the wire frame; abrupt TCP drops surface as `ws::closed{code:1006,reason:""}` |
| Terminality | `ws::close` sends the frame and owner-closes the session; later sends observe resource-state; the close event still arrives on the reader because explicit close keeps the hook registration |

## Upgrade (server)

| Aspect | Contract |
|---|---|
| Flow | the route handler calls `ws::accept(request, protocol, caps)` after authentication; the adapter upgrades through the retained native request and bound server handle, marks the request, and dispatch emits no HTTP reply (fetch returns undefined) |
| Selection | empty string selects nothing (Bun then auto-selects the peer's first offered protocol, observed); a non-empty selection must be a valid token within the peer's offered list or `ws::invalid_protocol` / `ws::unsupported_protocol` fails before any upgrade |
| Refusal | `server.upgrade` false (plain GET, consumed request) fails `ws::upgrade_failed{refused}` and releases the claim; a second accept fails `ws::upgrade_failed{already_upgraded}`; a handler that never accepts answers ordinary HTTP (401/denial stays a normal response) |
| Timing | upgrade-after-`await` and upgrade-after-body-drain both succeed on the pinned target, so buffered ingress routes upgrade directly |

## Client

| Aspect | Contract |
|---|---|
| Validation | unparseable URL fails `ws::invalid_url{unparseable}`, non-ws/wss scheme fails `ws::invalid_url{scheme}`, malformed protocols fail `ws::invalid_protocol` before dialing |
| Handshake failure | close-before-open maps 1002 to `ws::connect_failed{rejected}`, 1006 to `{unreachable}`, 1015 to `{tls}`, anything else to `{closed}`; the caller deadline maps to `{timeout}`; error events carry no provenance and close owns every terminal |
| TLS | wss verifies against OS trust by default; `insecure_tls` true selects the Bun object-form bypass for loopback/self-signed testing and is ignored for ws:// |

## Lifecycle

| Aspect | Contract |
|---|---|
| Ownership | sessions are `ws-session` owner resources in the caller's scope (server sessions: the request scope, which spans the driving handler); readers are scope-managed `stream-reader` resources; forgetting the session close emits the standard abandonment diagnostic |
| Cancel | `cancel_reader` interrupts a pending event read, which reports `cancelled` with the caller reason and delivers nothing |
| Stop | `server_stop` (and the wait signal path) prompts every live session with a 1001 `shutdown` frame and fails its pump `read_failed{server_stopped}` before closing the server; prompting inside the closer would deadlock against parked handlers, so entries suspend first |
| Late events | hooks never throw into Bun and never enter Can; events for unknown or terminal sessions drop silently |

## Error table

| Error | Fields | Meaning |
|---|---|---|
| ws::connect_failed (1333) | reason | timeout, unreachable, rejected, tls, closed |
| ws::upgrade_failed (1334) | reason | refused, already_upgraded |
| ws::unsupported_protocol (1335) | protocol | selection outside the peer's offered list |
| ws::send_failed (1336) | reason | blocked, closed, too_large |
| ws::invalid_close (1337) | reason | code, reason |
| ws::limit_exceeded (1338) | limit | cap or deadline outside its range (echoes the value) |
| ws::invalid_url (1339) | reason | unparseable, scheme |
| ws::invalid_protocol (1340) | protocol | malformed protocol token |
| stream::read_failed (1316) | reason | message_too_large, queue_overrun, server_stopped over event cells |
