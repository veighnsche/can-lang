# D03 qualification record — second vendor integration (W2)

Task D03. Source: R01/R11; F-R01-03; W2. Environment: pinned Bun
1.4.2, go1.27.1 darwin/arm64, node v24.21.0, isolated worktree at
`a6eb8f7b` plus Lane-D commits `3c1475b7`, `0c5f6093`, and the slice 3
commit below. No pinned browser was launchable (C01 blocked-open), so
the Vendor B live legs are pinned and recorded as D03-owned conformance
debt (D03-LIVE-1..4 in `conformance/live/README.md`) — never claimed
passing. D02's live debt (D02-LIVE-1..4) stays open alongside; D03 closes
none of it.

D02 input (all inherited, none re-decided): Op A storage → T2 reviewed
adapter, Op B clipboard → T2 reviewed adapter, Widget C chart → T3
companion (`d02.chart/1`, Vendor A reference renderer); T1 stays live per
class with selection conditions. See `d02-record.md` and
`tests/host-discrimination/x-r01-1.md`.

## W2.1 path record (P09-M1): conditional catalogue path NOT taken

X-R01-1 assigned Widget C to **T3 companion**, not catalogue-only, so
the conditioned generic-catalogue allowance does not apply and was not
used: W2 runs with **no compiler change at all** — no vendor-specific
patch, no generic addition. Evidence:

- The D03 diff touches `host/` only: `git diff a6eb8f7b..HEAD
  --name-only` lists 13 paths, all under `host/`; `compiler/`,
  `runtime/`, `tools/`, `examples/`, `distribution/`, `internal/`, and
  `tests/` are byte-identical to the D02 base (verified by command, not
  by review). In particular `compiler/internal/catalogue/catalogue.json`
  and generated `runtime/catalogue.ts` are untouched — no hand-edited
  generated code, no regenerated admission either.
- The second vendor reuses the delivered D02 client and protocol: Vendor
  B is a server-side engine + deployment identity only. Per-vendor
  config (policy, endpoint, credential env name) travels as client init
  data, never as a client code change.
- The catalogue legs pin the negative: `can.std.storage/clipboard/chart`
  plus the vendor-specific `can.std.chart_vendor_b` / `can.std.vendor_b`
  identities are absent from both `catalogue.json` and generated
  `runtime/catalogue.ts`, and no owned tree references any D02/D03
  deliverable marker.

The vendor dimension applies to the widget class (chart SDKs), where
vendors exist. The adapter classes bind single-vendor WHATWG/W3C
standard shapes, so their W2 contribution is reuse-by-rerun: every D02
adapter leg runs unchanged in the D03 CI gate (new total below),
proving the second integration forked neither adapter path.

## Deliverables (`host/`, Lane-D owned, PENDING-REVIEW)

| Class | Tier | Module | Notes |
|---|---|---|---|
| Widget C chart vendor B | T3 companion | `companions/chart-vendor-b.ts` (152 lines) + `chart-vendor-b-recipe.md` | second engine (`<g>`-grouped horizontal bars, axis, value labels; `data-vendor="b"`), reimplemented server handle, `vb-` render namespace, `vendor-b` diagnostics, own deployment identity (host `companion-b.example`, credential `D03_CHART_VENDOR_B_TOKEN`, policy `2026-09-26.d03-chart-vendor-b`); protocol `d02.chart/1` and all codecs imported from the D02 module, never forked |

No D02 deliverable changed: the D02 bundle pins hold byte-for-byte
(5618/4381/20838), and the D02 envelope pins (209/164) are untouched.

## Conformance verdicts (runnable)

`bun test host/conformance/`: **51/51** across 5 files (40 D02 + 11 D03).

- Vendor B (11): roundtrip through the unchanged D02 client (`vb-` id,
  `data-vendor="b"` bytes, frozen table, clean release); engine
  determinism (same spec → byte-identical SVG) and structural
  distinctness from Vendor A; client-side invalid-spec rejection with
  zero sends; lifecycle (expiry, idempotent release, range, `vendor-b`
  diagnostics, no secret); denied/bad-credential/cross-vendor-secret/
  offline fail-closed; unknown-code sanitize + version mismatch;
  auth parity (same policy both sides, secret never logged);
  cross-vendor isolation (foreign select → `expired`, foreign release
  idempotent-ok without touching the owner); 3 loopback HTTP wire legs
  (roundtrip, wrong-credential/version, 250ms trip wire).
- Packaging (2): all four entries bundle twice byte-identical; sizes
  pinned: storage 5618, clipboard 4381, chart 20838, vendor-b 12160
  bytes (unminified; vendor-b shares the F01 C-G policy).

`go test ./host/conformance/ -count=1`: **9 pass, 2 skip, 0 fail**.

- `TestVendorCapabilitiesRejectWithLocation`: `uses [chart_vendor_b]`,
  `[vendor_b]`, `[storage]`, `[clipboard]`, `[chart]` each fail the real
  compiler (`canlc inspect-project`, black-box) with file-attributed
  `unknown package "<capability>"` evidence — no vendor-specific
  admission exists, and W2 admitted nothing.
- `TestDeliverablesIntroduceNoCatalogueIdentities`: no manifest-listed
  module names a catalogue operation identity or the catalogue merge
  source — W2 adds nothing the browser closure would rule on.
- `TestVendorBReusesDeliveredClient`: Vendor B imports the shared
  contract from `./chart.ts`, uses the shared `CHART_PROTOCOL`, and
  defines no client factory and no protocol constant.
- `TestBrowserClosureSuiteGreen`: the C-owned
  `compiler/internal/browser/` suite runs read-only and is required
  green — D03 patches no closure file (C patches go through C; none
  requested) and gates W2 on the real gate.
- All five D02 Go legs re-run green (unadmitted/vendor markers
  extended with the D03 identities); `TestLiveBrowserLegs` and
  `TestLiveVendorBLegs` SKIP with their precise unblock commands
  (debt, not passes).

Strict tsc over all 10 host TS files (4 deliverables, 5 conformance
tests, 1 live page entry) is clean; `gofmt` clean. oxfmt/oxlint stay
not-applicable to `host/` (repo scripts cover `runtime/` and
`tools/runtime/` only; pre-existing D02 host files do not pass oxfmt).

## Trip-wire measurement (Vendor B)

Loopback select roundtrip (Bun.serve + fetch transport, 50 samples):
p50 0.064ms, pmax 0.196ms against the shared 250ms
`CHART_SELECT_ROUNDTRIP_BUDGET_MS` — not tripped, ~1270x headroom. The
budget applies per vendor: either vendor tripping reactivates the
X-R01-1 tier choice. Production network latency stays an operator
concern.

## Supplemental unpinned-browser evidence (non-qualifying)

A byte-identical copy of the pinned `vendor-b.mjs` runner
(`cmp`-verified) was driven from `/tmp` (throwaway Playwright shim
injecting an executable path; nothing committed) against the
environment's unpinned `chrome-headless-shell` (HeadlessChrome
153.0.8010.12 — NOT the pinned Chromium 140, and only one engine, so
this qualifies nothing and closes no debt item; the pinned-matrix legs
in `conformance/live/` still own the verdict):

- 5/5 live checks green from real page context: render/select/release
  roundtrip over loopback fetch (`vb-` id, `data-vendor="b"` bytes),
  select-after-release expiry, client-side invalid-spec rejection, and
  no page errors; `secureContext: true` on loopback.
- In-page select timing: p50 0.400ms, worst 0.600ms of 25 against the
  250ms trip wire.

## W2 verdicts

| Leg | Verdict |
|---|---|
| W2 positive — second vendor, no vendor-specific compiler patch, no hand-edited generated code; capability rejection + lifecycle green | **PASS (runnable)** — Vendor B runs behind the assigned T3 path; D03 diff is `host/`-only with catalogue/generated files byte-identical; 11 vendor-B legs + all D02 legs green |
| W2 negative — unadmitted capability fails the browser closure with location evidence | **PASS (runnable)** — vendor-specific and D02 capability names reject with file-attributed `unknown package` diagnostics (the check phase of the browser build pipeline); the C-owned closure suite is gated green read-only. No dev-distribution `canlc build --target browser` end-to-end leg exists here (builds need an H-packaged distribution; CAN-DIST-UNBUNDLED in this worktree) — recorded as a gap for H11, not a pass |
| W2 integration — assigned adapter/companion conformance in CI shape | **PASS (runnable) + LIVE DEBT** — `bun test host/conformance/` + `go test ./host/conformance/` are the CI shape and green; the Vendor B live-browser gate skips on D03-LIVE-1..4 until C01 unblocks |

## Honest deltas and open points

- Vendor B escapes `&<>` in interpolated SVG text; Vendor A does not.
  Engine-internal difference (the SVG is opaque boundary bytes), not a
  protocol term — recorded so no one mistakes it for a contract change.
- Vendor B shares the D02 request/response codecs by import. The codecs
  ARE the protocol contract (`chart.ts` per the D02 recipe); the
  reimplemented handle + engine are the vendor-specific parts. A
  from-spec reimplementation would port the codecs; in-repo import is
  the conforming equivalent.
- The end-to-end `canlc build --target browser` negative (unadmitted
  capability through the full browser build) is not runnable in this
  worktree: `canlc build` requires a development distribution
  (`CAN-DIST-UNBUNDLED`), which is H-owned packaging outside Lane D's
  `host/`-only boundary. The check-phase rejection + read-only closure
  witness above are the runnable evidence; H11 owns the fuller leg if
  it wants one after H packaging lands.
- SDK representative sizing stays the known unmeasured gap from D01
  (direction known, T3-favoring); unchanged by D03.
- `canlc build`/`inspect` behavior, catalogue contents, and closure
  rules were read, never patched: Lane D's contract holds.

## Handoff to H11

- Reproducible artifact: the four `host/` deliverables (three D02 +
  Vendor B) + `REVIEW-MANIFEST.json` (`2026-09-26.0-d02`,
  PENDING-REVIEW, Vendor B listed) + byte-identical bundle legs.
  Distribution review grants `reviewed`; Lane D never does.
- Conformance suite: `bun test host/conformance/` (51/51) + `go test
  ./host/conformance/ -count=1` (9 pass, 2 skip) — CI shape; live gates
  skip until C01 unblocks, then enforce.
- W2 evidence: this record + the suite. Release scope for IC2: the
  second vendor demonstrably uses the reusable T3 path (one protocol,
  two engines, one unchanged client); no new checked surface, no
  vendor-specific compiler or catalogue content.
- D03-owned debt: D03-LIVE-1..4 (`conformance/live/README.md`).
  Inherited open debt: D02-LIVE-1..4 (T2 live-native legs).
- F (no patch): Vendor B consumes F01's destination policy and identity
  vocabulary unchanged; chart payload contract still awaits F05's
  protocol ack on D02's schedule (the ack covers both vendors — one
  protocol).
- C/E (no patch): no catalogue or checker change requested; T1 sketches
  and selection conditions unchanged from D01.
