# Widget C chart companion — serving/auth/release recipe (D02)

Companion tier for the invoice-analytics chart widget, scoped to the
X-R01-1 assignment: occasional-selection analytics widgets over
server-loaded data. Payload contract: `chart.ts`, protocol
`d02.chart/1`. D02 owns this payload contract; F05 owns the pair
mechanics (carrier protocol, supervision); F01 destination policy and
identity vocabulary are consumed unchanged.

## Serving

- The companion is a separate runtime/service, never an in-process
  import. Production endpoint: `https://companion.example/v1/chart`
  (pinned by `chartCompanionPolicy()`; the only admitted destination).
- One HTTP POST per chart op (`render`, `select`, `release`) with a
  JSON body of `encodeChartRequest` bytes and a JSON response body of
  `encodeChartResponse` bytes. The operator's HTTP server parses the
  body to bytes, calls `createChartCompanionServer(...).handle` with
  the `authorization` header value, and writes the returned bytes
  with status 200. Non-POST methods and non-`/v1/chart` paths reject
  at the HTTP layer (operator's server, outside this contract).
- The renderer behind `handle` is the operator's chart engine. The
  reference `renderSvg` in `chart.ts` defines the boundary shape
  (opaque SVG bytes plus the frozen typed table), not the pixels:
  swapping engines never changes the client.
- Availability: the companion is online by construction for the C04
  apps (they load/save through server actions already). Offline reads
  surface honestly as `companion::transport_failed`; there is no
  offline render mode. If H/user ever require offline render of
  cached drafts, the X-R01-1 trip conditions reactivate the tier
  choice (T2 first) — do not bolt a cache onto this contract.

## Auth

- Bearer credential bound by environment name only:
  `D02_CHART_COMPANION_TOKEN`. The value resolves at send time on
  the client (`credentialValue`) and at handle time on the server;
  it never enters a decision, report, diagnostic, or artifact.
- Comparison is constant-time over equal lengths
  (`compareSecret`); missing or mismatched credentials fail closed
  as `companion::unauthorized` on both sides (client-side when its
  own value is unavailable, server-side on mismatch).
- Tenant/pool/correlation ride every request from F01's identity
  vocabulary (`chartCompanionContext`); the server logs
  `accept:<op>:<tenant>:<correlation>` / `reject:<code>...` lines
  that carry no secret. Response correlation must match the request
  or the client fails closed with `companion::protocol_error`.
- Destination policy is evaluated client-side on every roundtrip
  (`evaluateDestination`): anything outside the pinned rule fails
  closed as `companion::destination_denied` with zero sends.

## Release coordination

- Protocol version `d02.chart/1` is the only admitted wire
  version. Both codecs reject anything else as
  `companion::version_mismatch`, failing closed in both
  directions — a client and server on different versions refuse to
  speak rather than misread bodies. Releases pair client and
  server on the same protocol version; there is no negotiation.
- `release` is idempotent: releasing an unknown or already-released
  render succeeds, so at-least-once redelivery never fails.
  `select` against an unknown or released render fails honestly as
  `chart::expired`; out-of-range indexes against a live render fail
  as `chart::invalid_spec` (client and server validate
  independently — the server never trusts the client shape).
- Policy version `2026-09-26.d02-chart` is evaluated on both sides
  (`policyVersion` handshake leg in conformance); a release that
  changes admission rules bumps the version and ships both halves
  together.
- Trip conditions (X-R01-1, unchanged): sub-roundtrip interaction
  budgets (brushing, hover-linked selection), required offline
  render, or a select roundtrip above
  `CHART_SELECT_ROUNDTRIP_BUDGET_MS` (250ms, loopback conformance
  leg) reactivates the tier choice. New widget shapes outside the
  stated scope return individually.
