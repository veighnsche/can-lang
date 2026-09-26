// X-R01-1 (D01) UNREVIEWED PROTOTYPE — companion tier, Widget C.
//
// Invoice chart widget through a typed companion data boundary: Can
// posts a chart spec plus data, the companion validates, renders a
// deterministic SVG plus an accessible data table, and answers with
// immutable bytes; bar selection becomes a typed select request and
// render artifacts release explicitly. No third-party SDK crosses into
// the Can build: the companion owns the renderer behind the boundary.
// The fake renderer below stands in for any real chart engine; its
// output shape (opaque bytes plus typed table) is the contract, not
// its pixels.
import {
  credentialValue,
  destinationPolicy,
  evaluateDestination,
  policyReport,
  type DestinationPolicy,
  type EnvName,
} from "../../../runtime/outbound/destination-policy.ts";
import {
  correlationId,
  invocationContext,
  poolId,
  tenantId,
  type CorrelationId,
  type InvocationContext,
} from "../../../runtime/outbound/identity.ts";

export const CHART_PROTOCOL = "d01.chart/1";
export const CHART_TITLE_MAX_CHARS = 128;
export const CHART_LABEL_MAX_CHARS = 64;
export const CHART_POINTS_MAX = 366;

export type CompanionChartFailureCode =
  | "chart::invalid_spec"
  | "chart::expired"
  | "companion::destination_denied"
  | "companion::unauthorized"
  | "companion::version_mismatch"
  | "companion::transport_failed"
  | "companion::protocol_error";

export type CompanionChartFailure = Readonly<{ code: CompanionChartFailureCode; detail: string }>;

export type CompanionChartResult<T> =
  | Readonly<{ ok: true; value: T }>
  | Readonly<{ ok: false; failure: CompanionChartFailure }>;

function ok<T>(value: T): CompanionChartResult<T> {
  return Object.freeze({ ok: true, value }) as CompanionChartResult<T>;
}

function fail<T>(code: CompanionChartFailureCode, detail: string): CompanionChartResult<T> {
  return Object.freeze({ ok: false, failure: Object.freeze({ code, detail }) }) as CompanionChartResult<T>;
}

export type ChartPoint = Readonly<{ label: string; value: number }>;
export type ChartOp = "render" | "select" | "release";

export type ChartRequest = Readonly<{
  v: string;
  tenant: string;
  pool: string;
  correlation: string;
  op: ChartOp;
  title?: string;
  kind?: string;
  points?: readonly ChartPoint[];
  renderId?: string;
  index?: number;
}>;

export type ChartRender = Readonly<{ renderId: string; svg: string; table: readonly ChartPoint[] }>;
export type ChartSelection = Readonly<{ label: string; value: number }>;

export type ChartResponse = Readonly<
  | { v: string; correlation: string; ok: true; render?: ChartRender; selection?: ChartSelection }
  | { v: string; correlation: string; ok: false; code: string; detail: string }
>;

const textEncoder = new TextEncoder();
const textDecoder = new TextDecoder();

function checkPoint(point: unknown): ChartPoint | null {
  if (point === null || typeof point !== "object" || Array.isArray(point)) return null;
  const shape = point as Record<string, unknown>;
  if (typeof shape.label !== "string" || shape.label.length === 0 || shape.label.length > CHART_LABEL_MAX_CHARS || !shape.label.isWellFormed()) {
    return null;
  }
  if (typeof shape.value !== "number" || !Number.isFinite(shape.value)) return null;
  return { label: shape.label, value: shape.value };
}

function checkPoints(points: unknown): readonly ChartPoint[] | null {
  if (!Array.isArray(points) || points.length === 0 || points.length > CHART_POINTS_MAX) return null;
  const checked: ChartPoint[] = [];
  for (const point of points) {
    const valid = checkPoint(point);
    if (valid === null) return null;
    checked.push(valid);
  }
  return checked;
}

function checkTitle(title: unknown): string | null {
  return typeof title === "string" && title.length > 0 && title.length <= CHART_TITLE_MAX_CHARS && title.isWellFormed() ? title : null;
}

export function encodeChartRequest(request: ChartRequest): Uint8Array {
  return textEncoder.encode(JSON.stringify(request));
}

export function decodeChartRequest(bytes: Uint8Array): ChartRequest | CompanionChartFailure {
  let parsed: unknown;
  try {
    parsed = JSON.parse(textDecoder.decode(bytes));
  } catch {
    return { code: "companion::protocol_error", detail: "request is not well-formed JSON" };
  }
  if (parsed === null || typeof parsed !== "object" || Array.isArray(parsed)) {
    return { code: "companion::protocol_error", detail: "request must be an object" };
  }
  const shape = parsed as Record<string, unknown>;
  if (shape.v !== CHART_PROTOCOL) return { code: "companion::version_mismatch", detail: "unsupported protocol version" };
  if (shape.op !== "render" && shape.op !== "select" && shape.op !== "release") {
    return { code: "companion::protocol_error", detail: "unknown chart op" };
  }
  try {
    tenantId(shape.tenant);
    poolId(shape.pool);
    correlationId(shape.correlation);
  } catch {
    return { code: "companion::protocol_error", detail: "invalid identity" };
  }
  const base = { v: CHART_PROTOCOL, tenant: String(shape.tenant), pool: String(shape.pool), correlation: String(shape.correlation), op: shape.op } as const;
  if (shape.op === "render") {
    const title = checkTitle(shape.title);
    const points = checkPoints(shape.points);
    if (title === null || shape.kind !== "bar" || points === null) {
      return { code: "chart::invalid_spec", detail: "render needs a title, kind bar, and 1..366 finite points" };
    }
    return { ...base, title, kind: "bar", points };
  }
  if (typeof shape.renderId !== "string" || shape.renderId.length === 0 || shape.renderId.length > 64) {
    return { code: "chart::invalid_spec", detail: "select/release need a render id" };
  }
  if (shape.op === "select") {
    if (typeof shape.index !== "number" || !Number.isInteger(shape.index) || shape.index < 0) {
      return { code: "chart::invalid_spec", detail: "select needs a point index" };
    }
    return { ...base, renderId: shape.renderId, index: shape.index };
  }
  return { ...base, renderId: shape.renderId };
}

export function encodeChartResponse(response: ChartResponse): Uint8Array {
  return textEncoder.encode(JSON.stringify(response));
}

export function decodeChartResponse(bytes: Uint8Array): ChartResponse | CompanionChartFailure {
  let parsed: unknown;
  try {
    parsed = JSON.parse(textDecoder.decode(bytes));
  } catch {
    return { code: "companion::protocol_error", detail: "response is not well-formed JSON" };
  }
  if (parsed === null || typeof parsed !== "object" || Array.isArray(parsed)) {
    return { code: "companion::protocol_error", detail: "response must be an object" };
  }
  const shape = parsed as Record<string, unknown>;
  if (shape.v !== CHART_PROTOCOL) return { code: "companion::version_mismatch", detail: "unsupported protocol version" };
  if (typeof shape.correlation !== "string") return { code: "companion::protocol_error", detail: "response lacks correlation" };
  if (shape.ok === true) return { v: CHART_PROTOCOL, correlation: shape.correlation, ok: true, render: shape.render as ChartRender | undefined, selection: shape.selection as ChartSelection | undefined };
  if (shape.ok === false && typeof shape.code === "string" && typeof shape.detail === "string") {
    return { v: CHART_PROTOCOL, correlation: shape.correlation, ok: false, code: shape.code, detail: shape.detail };
  }
  return { code: "companion::protocol_error", detail: "response shape unknown" };
}

export type CompanionTransport = {
  send(url: string, authorization: string, body: Uint8Array): Uint8Array;
};

export type ChartCompanionClient = Readonly<{
  render: (title: unknown, points: unknown) => Promise<CompanionChartResult<ChartRender>>;
  select: (renderId: unknown, index: unknown) => Promise<CompanionChartResult<ChartSelection>>;
  release: (renderId: unknown) => Promise<CompanionChartResult<null>>;
}>;

export function createChartCompanionClient(init: Readonly<{
  policy: DestinationPolicy;
  endpoint: string;
  context: InvocationContext;
  credentialEnv: EnvName;
  readEnvironment: (key: string) => string | undefined;
  transport: CompanionTransport;
}>): ChartCompanionClient {
  const roundtrip = async (request: Omit<ChartRequest, "v" | "tenant" | "pool" | "correlation">): Promise<CompanionChartResult<ChartResponse & { ok: true }>> => {
    const decision = evaluateDestination(init.policy, init.endpoint);
    if (decision.decision !== "allowed") return fail("companion::destination_denied", `destination denied: ${decision.reason}`);
    const secret = credentialValue(init.credentialEnv, init.readEnvironment);
    if (secret === undefined) return fail("companion::unauthorized", "credential unavailable");
    const body = encodeChartRequest({ v: CHART_PROTOCOL, tenant: init.context.tenant, pool: init.context.pool, correlation: init.context.correlation, ...request } as ChartRequest);
    let raw: Uint8Array;
    try {
      raw = init.transport.send(init.endpoint, `Bearer ${secret}`, body);
    } catch {
      return fail("companion::transport_failed", "companion unreachable");
    }
    const decoded = decodeChartResponse(raw);
    if ("code" in decoded) return fail(decoded.code as CompanionChartFailureCode, decoded.detail);
    if (decoded.correlation !== init.context.correlation) return fail("companion::protocol_error", "response correlation mismatch");
    if (!decoded.ok) {
      const code = ((): CompanionChartFailureCode => {
        switch (decoded.code) {
          case "chart::invalid_spec":
          case "chart::expired":
            return decoded.code;
          case "companion::unauthorized":
            return "companion::unauthorized";
          case "companion::version_mismatch":
            return "companion::version_mismatch";
          default:
            return "companion::protocol_error";
        }
      })();
      return fail(code, decoded.detail);
    }
    return ok(decoded);
  };
  return Object.freeze({
    async render(title: unknown, points: unknown): Promise<CompanionChartResult<ChartRender>> {
      const checkedTitle = checkTitle(title);
      const checkedPoints = checkPoints(points);
      if (checkedTitle === null || checkedPoints === null) {
        return fail("chart::invalid_spec", "render needs a title and 1..366 finite points");
      }
      const result = await roundtrip({ op: "render", title: checkedTitle, kind: "bar", points: checkedPoints });
      if (!result.ok) return fail(result.failure.code, result.failure.detail);
      if (result.value.render === undefined) return fail("companion::protocol_error", "render response lacks artifact");
      return ok(Object.freeze({ renderId: result.value.render.renderId, svg: result.value.render.svg, table: Object.freeze([...result.value.render.table]) }));
    },
    async select(renderId: unknown, index: unknown): Promise<CompanionChartResult<ChartSelection>> {
      if (typeof renderId !== "string" || renderId.length === 0) return fail("chart::invalid_spec", "select needs a render id");
      if (typeof index !== "number" || !Number.isInteger(index) || index < 0) return fail("chart::invalid_spec", "select needs a point index");
      const result = await roundtrip({ op: "select", renderId, index });
      if (!result.ok) return fail(result.failure.code, result.failure.detail);
      if (result.value.selection === undefined) return fail("companion::protocol_error", "select response lacks selection");
      return ok(Object.freeze({ ...result.value.selection }));
    },
    async release(renderId: unknown): Promise<CompanionChartResult<null>> {
      if (typeof renderId !== "string" || renderId.length === 0) return fail("chart::invalid_spec", "release needs a render id");
      const result = await roundtrip({ op: "release", renderId });
      if (!result.ok) return fail(result.failure.code, result.failure.detail);
      return ok(null);
    },
  });
}

function compareSecret(presented: string, expected: string): boolean {
  if (presented.length !== expected.length) return false;
  let diff = 0;
  for (let index = 0; index < expected.length; index += 1) {
    diff |= presented.charCodeAt(index) ^ expected.charCodeAt(index);
  }
  return diff === 0;
}

// renderSvg is the fake chart engine: deterministic bars from points.
// Real engines differ in pixels; the boundary contract (opaque bytes
// plus the typed table) does not.
function renderSvg(title: string, points: readonly ChartPoint[]): string {
  const bars = points.map((point, index) => `<rect x="${index * 12}" y="0" width="10" height="${Math.max(0, Math.min(100, Math.round(point.value)))}"><title>${point.label}</title></rect>`).join("");
  return `<svg data-title="${title}">${bars}</svg>`;
}

export type ChartCompanionServer = Readonly<{
  policyVersion: string;
  handle: (authorization: string | undefined, body: Uint8Array) => Uint8Array;
  diagnostics: () => readonly string[];
  liveRenders(): number;
}>;

export function createChartCompanionServer(init: Readonly<{
  policy: DestinationPolicy;
  credentialEnv: EnvName;
  readEnvironment: (key: string) => string | undefined;
}>): ChartCompanionServer {
  const renders = new Map<string, readonly ChartPoint[]>();
  let sequence = 0;
  const log: string[] = [];
  const report = policyReport(init.policy);
  return Object.freeze({
    policyVersion: report.version,
    handle(authorization: string | undefined, body: Uint8Array): Uint8Array {
      const decoded = decodeChartRequest(body);
      if ("code" in decoded) {
        log.push(`reject:${decoded.code}`);
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
        log.push(`reject:companion::unauthorized:${decoded.tenant}:${decoded.correlation}`);
        return encodeChartResponse({ v: CHART_PROTOCOL, correlation: decoded.correlation, ok: false, code: "companion::unauthorized", detail: "bad credential" });
      }
      log.push(`accept:${decoded.op}:${decoded.tenant}:${decoded.correlation}`);
      if (decoded.op === "render") {
        sequence += 1;
        const renderId = `r-${sequence}`;
        const table = Object.freeze(decoded.points!.map((point) => Object.freeze({ label: point.label, value: point.value })));
        renders.set(renderId, table);
        return encodeChartResponse({ v: CHART_PROTOCOL, correlation: decoded.correlation, ok: true, render: { renderId, svg: renderSvg(decoded.title!, table), table } });
      }
      if (decoded.op === "release") {
        // Idempotent: releasing an unknown or released render still
        // succeeds, so at-least-once redelivery never fails.
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

export function chartCompanionPolicy(): DestinationPolicy {
  return destinationPolicy({
    version: "2026-09-26.d01-chart",
    rules: [{ scheme: "https", host: "companion.example", pathPrefix: "/v1/", credential: "D01_CHART_COMPANION_TOKEN" }],
    redirect: "deny",
    maxRedirectHops: 0,
    loopback: "deny",
    privateNetworks: "deny",
  });
}

export function chartCompanionContext(correlation: string): InvocationContext {
  return invocationContext("tenant-acme", "pool-default", correlation as CorrelationId);
}
