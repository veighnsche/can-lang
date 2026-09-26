# Widget C chart companion, Vendor B — serving/auth/release recipe (D03)

Second vendor behind the D02 `d02.chart/1` protocol (`chart-vendor-b.ts`).
The D02 recipe (`chart-recipe.md`) governs the protocol; this file records
only what differs per vendor. The D02 client drives Vendor B unchanged —
client config (policy, endpoint, credential env name) is the only per-vendor
input, and it travels as init data, never as a code change.

## Serving

- Separate runtime/service like Vendor A. Production endpoint:
  `https://companion-b.example/v1/chart` (pinned by
  `chartVendorBPolicy()`; the only admitted Vendor B destination).
- One HTTP POST per chart op with `encodeChartRequest` bytes in and
  `encodeChartResponse` bytes out — the same codecs Vendor A uses, so a
  client cannot tell vendors apart except by engine bytes, render-id
  namespace (`vb-` here, `r-` on Vendor A), and diagnostics prefix.
- The render engine is Vendor B's (`renderVendorBSvg`: horizontal grouped
  bars with axis and value labels). The boundary shape (opaque SVG bytes
  plus the frozen typed table) is unchanged; swapping engines or vendors
  never changes the client.
- Availability: same honesty as Vendor A — offline reads surface as
  `companion::transport_failed`; no offline render mode. Renders are
  server-local: a render id from one vendor means nothing to the other
  (`chart::expired`), and releasing a foreign id is idempotent-ok without
  touching the owning vendor's state.

## Auth

- Bearer [REDACTED] bound by environment name only:
  `D03_CHART_VENDOR_B_TOKEN`, resolved at send/handle time, never logged.
  The value differs from Vendor A's; each vendor's credential authorizes
  only that vendor.
- Comparison is constant-time over equal lengths; missing or mismatched
  credentials fail closed as `companion::unauthorized`.
- Tenant/pool/correlation ride every request from F01's identity
  vocabulary; the server logs `vendor-b accept:...` /
  `vendor-b reject:...` lines that carry no secret. Response correlation
  must match the request or the client fails closed with
  `companion::protocol_error`.
- Destination policy is evaluated client-side on every roundtrip:
  anything outside the pinned Vendor B rule fails closed as
  `companion::destination_denied` with zero sends.

## Release coordination

- Protocol version `d02.chart/1` is shared by both vendors: one protocol,
  two engines. Both codecs reject anything else as
  `companion::version_mismatch` in both directions. Client and server
  releases pair on the protocol version; there is no negotiation and no
  per-vendor protocol fork.
- `release` is idempotent; `select` against an unknown, released, or
  foreign render fails honestly as `chart::expired`; out-of-range indexes
  against a live render fail as `chart::invalid_spec`. Client and server
  validate independently.
- Policy version `2026-09-26.d03-chart-vendor-b` is evaluated on the
  Vendor B side; a release that changes Vendor B admission rules bumps it
  and ships both halves together. Vendor A's policy version is untouched.
- Trip conditions (X-R01-1, unchanged): sub-roundtrip interaction budgets,
  required offline render, or a select roundtrip above
  `CHART_SELECT_ROUNDTRIP_BUDGET_MS` (250ms, loopback conformance leg per
  vendor) reactivates the tier choice. The budget applies to Vendor B
  independently — either vendor tripping reopens the tier question.
