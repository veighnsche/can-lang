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
