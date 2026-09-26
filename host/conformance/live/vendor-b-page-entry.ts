// D03 live-leg page entry: a pure re-export bundle root so the pinned
// browsers drive the DELIVERED client (`chart.ts`) plus the F01 C-G
// constructors it needs (`destinationPolicy`, `envName`) from page context.
// Behavior-free by construction: every export below is defined elsewhere.
// The Go gate (`live_vendor_b_test.go`) bundles this file with `bun build`
// for the page; Vendor B's server bundle is built from
// `host/companions/chart-vendor-b.ts` and runs in node.
export {
  chartCompanionContext,
  createChartCompanionClient,
  createFetchTransport,
  CHART_SELECT_ROUNDTRIP_BUDGET_MS,
} from "../../companions/chart.ts";
export { destinationPolicy, envName } from "../../../runtime/outbound/destination-policy.ts";
