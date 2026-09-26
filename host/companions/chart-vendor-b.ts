// D03 second-vendor companion tier, Widget C (PENDING-REVIEW — not admitted).
//
// Vendor B invoice chart renderer behind the SAME `d02.chart/1` protocol the
// D02 client speaks: the protocol codecs, the destination-policy shape, and
// the identity vocabulary are the reusable path, so this module imports them
// and reimplements only the vendor-specific parts (the render engine, the
// server handle, the deployment policy). The D02 client
// (`createChartCompanionClient` in `chart.ts`) drives Vendor B unchanged —
// there is no Vendor B client in this file, and the W2 conformance legs prove
// it by importing the client from `chart.ts` only.
//
// W2.1 path record (P09-M1): the conditional generic-catalogue path was NOT
// taken. X-R01-1 assigned Widget C to T3 companion, not catalogue-only, so
// no catalogue addition was needed or made; this file names no `can.std.*`
// operation and the catalogue legs pin every hypothetical capability absent.
// Serving/auth/release recipe: `chart-vendor-b-recipe.md`; the D02 recipe
// (`chart-recipe.md`) still governs Vendor A.
import {
  credentialValue,
  destinationPolicy,
  type DestinationPolicy,
  type EnvName,
} from "../../runtime/outbound/destination-policy.ts";
import {
  CHART_PROTOCOL,
  decodeChartRequest,
  encodeChartResponse,
  type ChartCompanionServer,
  type ChartPoint,
} from "./chart.ts";

export const CHART_VENDOR_B_POLICY_VERSION = "2026-09-26.d03-chart-vendor-b";
export const CHART_VENDOR_B_CREDENTIAL_ENV = "D03_CHART_VENDOR_B_TOKEN";
export const CHART_VENDOR_B_ENDPOINT = "https://companion-b.example/v1/chart";

// escapeText keeps label/title text inside SVG element content. The bytes
// are engine-internal (the contract calls the SVG opaque); Vendor A does
// not escape, which is a Vendor A engine property, not a protocol term.
function escapeText(value: string): string {
  return value.replaceAll("&", "&amp;").replaceAll("<", "&lt;").replaceAll(">", "&gt;");
}

// renderVendorBSvg is the second render engine: horizontal bars grouped in
// `<g>` elements with an axis line and value labels — structurally distinct
// from Vendor A's vertical `<rect>` bars, deterministically derived from the
// same points. Conformance pins the shape markers and byte determinism,
// never aesthetic equivalence.
function renderVendorBSvg(title: string, points: readonly ChartPoint[]): string {
  const groups = points
    .map(
      (point, index) =>
        `<g class="bar" data-index="${index}"><rect x="0" y="${index * 24}" width="${Math.max(0, Math.min(400, Math.round(point.value * 2)))}" height="20"><title>${escapeText(point.label)}</title></rect><text x="4" y="${index * 24 + 15}">${escapeText(point.label)}: ${point.value}</text></g>`,
    )
    .join("");
  return `<svg data-title="${escapeText(title)}" data-vendor="b"><line x1="0" y1="0" x2="0" y2="${points.length * 24}"/>${groups}</svg>`;
}

function compareSecret(presented: string, expected: string): boolean {
  if (presented.length !== expected.length) return false;
  let diff = 0;
  for (let index = 0; index < expected.length; index += 1) {
    diff |= presented.charCodeAt(index) ^ expected.charCodeAt(index);
  }
  return diff === 0;
}

const textDecoder = new TextDecoder();

// createChartVendorBServer implements the Vendor B side of `d02.chart/1`
// from the protocol contract: strict request codec, Bearer [REDACTED] auth,
// render/select/release dispatch, idempotent release, and honest expiry.
// Render ids carry the `vb-` namespace so cross-vendor legs can attribute
// artifacts; diagnostics carry the `vendor-b` prefix for the same reason.
// Lifecycle semantics match Vendor A exactly (same protocol, same leaves);
// only the engine bytes and the deployment identity differ.
export function createChartVendorBServer(init: Readonly<{
  policy: DestinationPolicy;
  credentialEnv: EnvName;
  readEnvironment: (key: string) => string | undefined;
}>): ChartCompanionServer {
  const renders = new Map<string, readonly ChartPoint[]>();
  let sequence = 0;
  const log: string[] = [];
  return Object.freeze({
    policyVersion: init.policy.version,
    handle(authorization: string | undefined, body: Uint8Array): Uint8Array {
      const decoded = decodeChartRequest(body);
      if ("code" in decoded) {
        log.push(`vendor-b reject:${decoded.code}`);
        let correlation = "unknown";
        try {
          const parsed = JSON.parse(textDecoder.decode(body)) as { correlation?: unknown };
          if (typeof parsed.correlation === "string") correlation = parsed.correlation;
        } catch {
          correlation = "unknown";
        }
        return encodeChartResponse({ v: CHART_PROTOCOL, correlation, ok: false, code: decoded.code, detail: decoded.detail });
      }
      const expected = credentialValue(init.credentialEnv, init.readEnvironment);
      const presented = authorization !== undefined && authorization.startsWith("Bearer ") ? authorization.slice(7) : "";
      if (expected === undefined || !compareSecret(presented, expected)) {
        log.push(`vendor-b reject:companion::unauthorized:${decoded.tenant}:${decoded.correlation}`);
        return encodeChartResponse({ v: CHART_PROTOCOL, correlation: decoded.correlation, ok: false, code: "companion::unauthorized", detail: "bad credential" });
      }
      log.push(`vendor-b accept:${decoded.op}:${decoded.tenant}:${decoded.correlation}`);
      if (decoded.op === "render") {
        sequence += 1;
        const renderId = `vb-${sequence}`;
        const table = Object.freeze(decoded.points!.map((point) => Object.freeze({ label: point.label, value: point.value })));
        renders.set(renderId, table);
        return encodeChartResponse({ v: CHART_PROTOCOL, correlation: decoded.correlation, ok: true, render: { renderId, svg: renderVendorBSvg(decoded.title!, table), table } });
      }
      if (decoded.op === "release") {
        // Idempotent like Vendor A: releasing an unknown or released
        // render still succeeds, so at-least-once redelivery never fails.
        renders.delete(decoded.renderId!);
        return encodeChartResponse({ v: CHART_PROTOCOL, correlation: decoded.correlation, ok: true });
      }
      const table = renders.get(decoded.renderId!);
      if (table === undefined) {
        return encodeChartResponse({ v: CHART_PROTOCOL, correlation: decoded.correlation, ok: false, code: "chart::expired", detail: "render unknown or released" });
      }
      const point = table[decoded.index!];
      if (point === undefined) {
        return encodeChartResponse({ v: CHART_PROTOCOL, correlation: decoded.correlation, ok: false, code: "chart::invalid_spec", detail: "point index out of range" });
      }
      return encodeChartResponse({ v: CHART_PROTOCOL, correlation: decoded.correlation, ok: true, selection: { label: point.label, value: point.value } });
    },
    diagnostics(): readonly string[] {
      return Object.freeze([...log]);
    },
    liveRenders(): number {
      return renders.size;
    },
  });
}

// chartVendorBPolicy is the Vendor B deployment identity: its own host,
// its own credential env name, its own policy version — the same rule
// shape as Vendor A, evaluated client-side on every roundtrip. The D02
// client takes this policy as init config; no client code changes.
export function chartVendorBPolicy(): DestinationPolicy {
  return destinationPolicy({
    version: CHART_VENDOR_B_POLICY_VERSION,
    rules: [{ scheme: "https", host: "companion-b.example", pathPrefix: "/v1/", credential: "D03_CHART_VENDOR_B_TOKEN" }],
    redirect: "deny",
    maxRedirectHops: 0,
    loopback: "deny",
    privateNetworks: "deny",
  });
}
