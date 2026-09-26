# D02/D03 host integration package (Lane D)

Delivers the X-R01-1 selected tiers from
`tests/host-discrimination/x-r01-1.md`, plus the D03 second vendor:

| Class | Assigned tier | Delivered here |
|---|---|---|
| Op A persistent client key-value storage | T2 reviewed adapter | `adapters/storage.ts` |
| Op B clipboard text exchange | T2 reviewed adapter | `adapters/clipboard.ts` |
| Widget C invoice chart widget | T3 companion (workflow-scoped, trip conditions) | `companions/chart.ts` |
| Widget C second vendor (D03) | T3 companion, same `d02.chart/1` protocol | `companions/chart-vendor-b.ts` |

Validators and failure leaves are inherited verbatim from the D01
prototypes; each module's header names its prototype source and its
D02 additions. Nothing here is admitted: the package is
`PENDING-REVIEW` (see `REVIEW-MANIFEST.json`), no owned tree
(`runtime/`, `tools/runtime/`, `compiler/`, `examples/`,
`distribution/`, `internal/`) references it, and the hypothetical
`can.std.storage/clipboard/chart` catalogue identities stay absent.
Admission requires distribution review; the conformance suite's Go
legs (`conformance/admission_test.go`) pin all three denials plus
located rejection of Can sources naming the capabilities.

Boundaries: D owns this package. C owns admission/browser, E owns
the catalogue, H owns packaging, F owns network policy (consumed
via `runtime/outbound/`, never patched) and the F05 pair mechanics
(D02 owns only the chart payload contract).

Run the runnable conformance suite:

```sh
bun test host/conformance/
go test ./host/conformance/ -count=1
```

Live-browser legs (real `localStorage`/`navigator.clipboard` on
pinned Chromium/WebKit/Firefox) are pinned under
`conformance/live/` and recorded as D02-owned conformance debt
until C01 unblocks: C01 is blocked-open, so no live leg is claimed
passing. The precise unblock command is in `conformance/live/README.md`.

Vendor B reuses the delivered D02 client unchanged (per-vendor config
as init data) with no compiler or catalogue change — the P09-M1
conditional catalogue path was not taken because X-R01-1 assigned T3,
not catalogue-only. Vendor-specific capability names reject with
location; the C-owned browser closure suite is gated green read-only.

Live-browser legs for Vendor B (delivered client from real page context
on pinned Chromium/WebKit/Firefox) are pinned under `conformance/live/`
and recorded as D03-owned conformance debt until C01 unblocks.

Handoff: reproducible integration artifact + conformance suite to H11.
See `d02-record.md` for the D02 delivery record and `d03-record.md` for
the W2 qualification record.
