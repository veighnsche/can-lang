# D02 live-browser legs — pinned spec and conformance debt

Status: **debt, unrun** — C01 is blocked-open (no pinned browser is
launchable), so no live leg below is claimed passing. Everything else
in `host/conformance/` runs and passes without a browser.

## What the live leg checks

`storage-clipboard.mjs` drives the **delivered** adapters
(`host/adapters/storage.ts`, `host/adapters/clipboard.ts`, bundled
with `bun build --format=esm`) against the real native calls in one
named browser (`chromium`, `webkit`, or `firefox`):

Storage (real `localStorage`):

1. `real roundtrip`: set/get/remove/missing-null through the adapter.
2. `invalid key rejects before host contact`: `invalid_key` with the
   store length unchanged.
3. `real quota breach maps with no native text`: fill toward the
   origin quota until the mapped `quota_exceeded` leaf, then clean
   the store; the native error name never appears in the result.

Clipboard (real `navigator.clipboard`):

4. `permission matrix recorded`: `clipboard-read`/`clipboard-write`
   states (or query-throw names) into `report.json`.
5. `raw empty-read shape recorded`: the native `readText()` outcome
   on a fresh page — resolution value or rejection name — into
   `report.json`. This pins the D01-documented uncertainty per
   browser; it records, it does not assume.
6. `real write/read roundtrip`: strict on Chromium (grant-backed
   `writeText` then `readText` equality); contract-floor on
   WebKit/Firefox (no throw, no native text, a known leaf). The
   first green live run pins per-engine behavior and may tighten
   the floor to strict-or-documented-denial.
7. `invalid write rejects before host contact`: `invalid_text`.
8. `no page errors escape any leg`.

Every leg runs on all three pinned browsers (Chromium 140, WebKit 26,
Firefox via Playwright 1.55.1, per C01). A passing run writes
`report.json` (`browser`, `userAgent`, `secureContext`, per-check
verdicts) per browser.

## Precise unblock command

Prerequisite (C01): provision the pinned harness once:

```sh
cd tests/integration/browser && bun ci && ./node_modules/.bin/playwright install chromium webkit firefox
```

Then, from the repo root:

```sh
CAN_D02_LIVE=1 go test ./host/conformance/ -run TestLiveBrowserLegs -count=1 -v
```

The gate (`live_test.go`) pre-bundles both adapters with
`bun build --format=esm` into a temp dir, resolves Playwright from
`tests/integration/browser/node_modules`, runs the runner once per
browser, and fails on any failed check or missing `report.json`.

## Debt ledger

- [ ] D02-LIVE-1: run the unblock command on all three pinned
  browsers; attach the three `report.json` files to `d02-record.md`.
- [ ] D02-LIVE-2: pin the per-browser empty-read shape (leg 5) and
  either confirm the `""→null` binding normalization or add the
  observed rejection name to the binding map with a conformance leg.
- [ ] D02-LIVE-3: pin the per-browser permission/engagement matrix
  (leg 4 + leg 6 outcomes); tighten the WebKit/Firefox
  contract-floor to strict-or-documented-denial.
- [ ] D02-LIVE-4: if any live leg fails, the failure returns to the
  D02 deliverable (binding or adapter fix + re-review), never to a
  weakened leg.

Until all four close, the T2 conformance verdict stays
"runnable legs green, live legs debt" — never "passing".

---

# D03 Vendor B live-browser legs — pinned spec and conformance debt

Status: **debt, unrun** — C01 is blocked-open (no pinned browser is
launchable), so no live leg below is claimed passing. Everything else
in `host/conformance/` runs and passes without a browser.

## What the live leg checks

`vendor-b.mjs` drives the **delivered** D02 client
(`host/companions/chart.ts`, bundled for the page via the behavior-free
`vendor-b-page-entry.ts` re-export root) against a loopback Vendor B
server (`host/companions/chart-vendor-b.ts`, bundled for node) from REAL
page context in one named browser (`chromium`, `webkit`, or `firefox`).
Page and `/v1/chart` share one loopback origin (same-origin fetch); the
runner secret travels as an evaluate argument, never in page source:

1. `page client: render, select, release roundtrip over loopback fetch`:
   render/select/release through the in-page client; `vb-` render id,
   `data-vendor="b"` engine bytes, exact table, clean release.
2. `page client: select-after-release expires from the page`:
   `chart::expired` across the page boundary.
3. `page client: invalid spec rejects client-side from the page`:
   `chart::invalid_spec` with no send.
4. `page client: select roundtrip timing recorded under the trip wire`:
   25 in-page selects; worst must stay under the shared 250ms
   `CHART_SELECT_ROUNDTRIP_BUDGET_MS`; p50/worst recorded into
   `report.json`. The first green live run pins per-engine numbers.
5. `no page errors escape any leg`.

Every leg runs on all three pinned browsers (Chromium 140, WebKit 26,
Firefox via Playwright 1.55.1, per C01). A passing run writes
`report.json` (`browser`, `userAgent`, `secureContext`, per-check
verdicts) per browser.

## Precise unblock command

Prerequisite (C01): provision the pinned harness once:

```sh
cd tests/integration/browser && bun ci && ./node_modules/.bin/playwright install chromium webkit firefox
```

Then, from the repo root:

```sh
CAN_D03_LIVE=1 go test ./host/conformance/ -run TestLiveVendorBLegs -count=1 -v
```

The gate (`live_vendor_b_test.go`) pre-bundles the page entry and the
Vendor B server with `bun build --format=esm` into a temp dir, resolves
Playwright from `tests/integration/browser/node_modules`, runs the
runner once per browser, and fails on any failed check or missing
`report.json`.

## Debt ledger

- [ ] D03-LIVE-1: run the unblock command on all three pinned
  browsers; attach the three `report.json` files to `d03-record.md`.
- [ ] D03-LIVE-2: pin the per-browser Vendor B select timing (leg 4)
  and in-page fetch behavior; confirm the trip wire holds per engine
  with headroom stated, or document the observed floor.
- [ ] D03-LIVE-3: pin the per-browser secure-context/page profile for
  the companion fetch path (leg reports `secureContext`); any engine
  that blocks loopback fetch from page context gets a documented
  operator posture, not a weakened leg.
- [ ] D03-LIVE-4: if any live leg fails, the failure returns to the
  D03 deliverable (server fix + re-review) or the shared client via
  its owner, never to a weakened leg.

Until all four close, the W2 live verdict stays
"runnable legs green, live legs debt" — never "passing".
