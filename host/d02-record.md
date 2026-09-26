# D02 delivery record — selected reviewed integration path

Task D02. Source: R01; F-R01-01/02/04; C-D/C-G. Environment: pinned
Bun 1.4.2, go1.27.1 darwin/arm64, isolated worktree at `1cf506e7`
plus Lane-D commits. No pinned browser was launchable (C01
blocked-open), so the T2 live-native legs are pinned and recorded as
D02-owned conformance debt (D02-LIVE-1..4 in
`conformance/live/README.md`) — never claimed passing.

D01 input (all inherited, none re-decided): Op A storage → T2
reviewed adapter, Op B clipboard → T2 reviewed adapter (T3 excluded
by measurement), Widget C chart → T3 companion (workflow-scoped,
trip conditions); T1 stays live per class with selection conditions.
See `tests/host-discrimination/x-r01-1.md`.

## Deliverables (`host/`, Lane-D owned, PENDING-REVIEW)

| Class | Tier | Module | Prototype delta |
|---|---|---|---|
| Op A storage | T2 reviewed adapter | `host/adapters/storage.ts` (238 lines) | header + appended `createNativeStorageHost` only; validators/leaves verbatim |
| Op B clipboard | T2 reviewed adapter | `host/adapters/clipboard.ts` (228 lines) | header + appended `createNativeClipboardHost` only; validators/leaves verbatim |
| Widget C chart | T3 companion | `host/companions/chart.ts` + `chart-recipe.md` (serving/auth/release) | protocol `d02.chart/1`, policy `2026-09-26.d02-chart`, env `D02_CHART_COMPANION_TOKEN`, async transport, fetch transport, 250ms select trip wire, envelope-flag discrimination fix (see below); validators/codecs/leaves verbatim |

F01 destination policy and identity vocabulary consumed unchanged;
F05 pair mechanics untouched (D02 owns only the chart payload
contract). No owned tree references `host/`; no `host` edit touches
`runtime/`, `tools/runtime/`, `compiler/`, `examples/`,
`distribution/`, or `internal/` (verified by leg, not by review).

## Conformance verdicts (runnable)

`bun test host/conformance/`: **40/40** across 4 files.

- Storage (13): 7 D01 legs re-run verbatim + 6 native-binding legs
  (absent/throwing/partial/probe-failed shapes fail closed;
  well-formed shape roundtrips with receiver preserved; quota throw
  maps with no native text).
- Clipboard (13): 8 D01 legs re-run verbatim + 5 native-binding legs
  (absent/insecure/partial shapes fail closed; prompt-and-attempt
  pre-gate pinned; `""→null` empty normalization; native denial
  maps with no native text).
- Chart (12): 8 D01 legs re-run (envelope pins hold byte-for-byte:
  209-byte render request, 164-byte response) + 3 loopback HTTP legs
  (wire roundtrip incl. idempotent release and expiry; wire
  unauthorized and version-mismatch; select roundtrip under budget)
  + 1 sanitizer leg (unknown server codes → `protocol_error`, known
  codes pass through).
- Packaging (2): all three entries bundle twice byte-identical;
  sizes pinned: storage 5618, clipboard 4381, chart 20838 bytes
  (unminified; chart includes the shared F01 C-G policy).

`go test ./host/conformance/ -count=1`: **5 pass, 1 skip**.

- `TestDeliveredCapabilitiesUnadmitted`: `can.std.storage@`,
  `can.std.clipboard@`, `can.std.chart@` absent from
  `catalogue.json` and generated `runtime/catalogue.ts`.
- `TestDeliverablesUnreferenced`: no D02 marker in any owned tree.
- `TestReviewManifestCoversDeliverables`: manifest
  `PENDING-REVIEW`, all three deliverables exactly once.
- `TestDeliverablesAvoidHostNamespaces`: no ambient `Bun.` /
  `process.env` / `Deno.` / `require(` in deliverables.
- `TestUnadmittedCapabilitiesRejectWithLocation`: `uses [storage]`,
  `[clipboard]`, `[chart]` each fail the real compiler
  (`canlc inspect-project`, black-box) with file-attributed
  `unknown package "<capability>"` evidence.
- `TestLiveBrowserLegs`: SKIP with the precise unblock command
  (debt, not a pass).

## Trip-wire measurement (Widget C)

Loopback select roundtrip (Bun.serve + fetch transport, 50
samples): p50 0.089ms, pmax 0.175ms against the 250ms
`CHART_SELECT_ROUNDTRIP_BUDGET_MS` trip wire — not tripped, ~1400x
headroom. The budget guards the payload contract against
structural regression; production network latency stays an
operator concern. Other X-R01-1 trip conditions (sub-roundtrip
interaction, offline render) are unchanged and untriggered.

## Supplemental unpinned-browser evidence (non-qualifying)

A throwaway CDP probe (`/tmp/d02-cdp-probe.ts`, not committed) drove
the delivered adapter bundles against the real native calls in the
environment's unpinned `chrome-headless-shell` (HeadlessChrome
153.0.8010.12 — NOT the pinned Chromium 140, and only one engine, so
this qualifies nothing and closes no debt item; the pinned-matrix
legs in `conformance/live/` still own the verdict):

- Storage: real roundtrip/set/get/remove/missing-null green;
  invalid key → `invalid_key` with store length 0→0; a real quota
  breach fired and mapped to `storage::quota_exceeded` with no
  native text, store cleaned to length 0.
- Clipboard ungranted (prompt/prompt): write/read → `denied`/`denied`
  with no throw and no native text — the prompt-and-attempt +
  mapping path works against the real API.
- Clipboard CDP-granted (`clipboardReadWrite`): full write/read
  roundtrip green through the delivered adapter; empty-device read
  resolved `""`, confirming the `""→null` binding normalization
  direction on this engine. Pinned-matrix confirmation still owned
  by D02-LIVE-2 (WebKit/Firefox shapes unknown).

## Honest deltas and open points

- Chart client discrimination fix: the prototype's `"code" in
  decoded` first branch also caught ok:false envelopes (which carry
  a code), stranding the sanitizing switch as dead, untypeable code
  (3 strict-tsc errors, reproduced on the D01 file) and passing
  unknown server codes through a blind cast. D02 discriminates on
  the `ok` flag instead: codec failures fail directly, ok:false
  envelopes run through the switch. Behavior is identical on every
  specified server code; unknown codes now sanitize to
  `protocol_error` (the switch's evident intent), pinned by the new
  sanitizer leg. Strict tsc over all 7 host TS files is clean.

- Clipboard native binding degrades the sync permission pre-gate to
  prompt-and-attempt (the Permissions API is async; no sync query
  exists). Denial still maps to the denied leaf via NotAllowedError.
- Clipboard `""→null` empty normalization is a D02 binding decision,
  verified per browser by live leg 5 (D02-LIVE-2); a rejecting
  browser adds its name to the map instead of assuming.
- The chart reference renderer defines the boundary shape, not
  pixels; W2 runs the second vendor against the assigned tiers with
  no vendor-specific compiler patch either way.
- SDK representative sizing stays the known unmeasured gap from
  D01 (direction known, T3-favoring); unchanged by D02.

## Handoff to D03/H

- Reproducible artifact: the three `host/` deliverables +
  `REVIEW-MANIFEST.json` (`2026-09-26.0-d02`, PENDING-REVIEW) +
  byte-identical bundle legs. Distribution review grants `reviewed`;
  Lane D never does.
- Conformance suite: `bun test host/conformance/` +
  `go test ./host/conformance/ -count=1` (CI shape; live gate
  skips until C01 unblocks, then enforces).
- D02-owned debt: D02-LIVE-1..4 (`conformance/live/README.md`).
- F (no patch): chart payload contract awaits F05's protocol ack on
  D02's schedule; no C-G gap found.
- C/E (no patch): T1 sketches and selection conditions unchanged
  from D01; no catalogue or checker change requested.
