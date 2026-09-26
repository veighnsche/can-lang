import { test, expect } from "bun:test";
import { envName, evaluateDestination, policyReport } from "../../../runtime/outbound/destination-policy.ts";
import {
  chartCompanionContext,
  chartCompanionPolicy,
  createChartCompanionClient,
  createChartCompanionServer,
  decodeChartRequest,
  decodeChartResponse,
  encodeChartRequest,
  encodeChartResponse,
  CHART_PROTOCOL,
  type CompanionTransport,
} from "./chart-widget.ts";

const SECRET = "D01-PROBE-CHART-SECRET-4d8f02";
const ENDPOINT = "https://companion.example/v1/chart";
const POINTS = [
  { label: "Jan", value: 120 },
  { label: "Feb", value: 200 },
];

function pair(correlation = "corr-2001") {
  const readEnvironment = (key: string): string | undefined =>
    key === "D01_CHART_COMPANION_TOKEN" ? SECRET : undefined;
  const server = createChartCompanionServer({
    policy: chartCompanionPolicy(),
    credentialEnv: envName("D01_CHART_COMPANION_TOKEN"),
    readEnvironment,
  });
  const transport: CompanionTransport = {
    send(url: string, authorization: string, body: Uint8Array): Uint8Array {
      expect(url).toBe(ENDPOINT);
      return server.handle(authorization, body);
    },
  };
  const client = createChartCompanionClient({
    policy: chartCompanionPolicy(),
    endpoint: ENDPOINT,
    context: chartCompanionContext(correlation),
    credentialEnv: envName("D01_CHART_COMPANION_TOKEN"),
    readEnvironment,
    transport,
  });
  return { client, server };
}

test("render, select, release roundtrip", async () => {
  const { client, server } = pair();
  const rendered = await client.render("Invoice totals", POINTS);
  expect(rendered.ok).toBe(true);
  if (!rendered.ok) throw new Error("render failed");
  expect(rendered.value.renderId).toBe("r-1");
  expect(rendered.value.svg).toContain("<svg");
  expect(rendered.value.table).toEqual(POINTS);
  expect(Object.isFrozen(rendered.value)).toBe(true);
  const selected = await client.select(rendered.value.renderId, 1);
  expect(selected).toEqual({ ok: true, value: { label: "Feb", value: 200 } });
  expect((await client.release(rendered.value.renderId)).ok).toBe(true);
  expect(server.liveRenders()).toBe(0);
});

test("invalid specs reject client-side without sending", async () => {
  let sent = 0;
  const readEnvironment = (key: string): string | undefined =>
    key === "D01_CHART_COMPANION_TOKEN" ? SECRET : undefined;
  const { server } = pair("corr-2002");
  const client = createChartCompanionClient({
    policy: chartCompanionPolicy(),
    endpoint: ENDPOINT,
    context: chartCompanionContext("corr-2002"),
    credentialEnv: envName("D01_CHART_COMPANION_TOKEN"),
    readEnvironment,
    transport: { send: (url: string, authorization: string, body: Uint8Array) => { sent += 1; return server.handle(authorization, body); } },
  });
  for (const [title, points] of [["", POINTS], ["t", []], ["t", [{ label: "x", value: NaN }]], [null, POINTS]] as const) {
    const got = await client.render(title, points);
    expect(got.ok).toBe(false);
    if (!got.ok) expect(got.failure.code).toBe("chart::invalid_spec");
  }
  expect(sent).toBe(0);
});

test("select-after-release expires; double release is idempotent", async () => {
  const { client } = pair("corr-2003");
  const rendered = await client.render("t", POINTS);
  if (!rendered.ok) throw new Error("render failed");
  expect((await client.release(rendered.value.renderId)).ok).toBe(true);
  expect((await client.release(rendered.value.renderId)).ok).toBe(true);
  const stale = await client.select(rendered.value.renderId, 0);
  expect(stale.ok).toBe(false);
  if (!stale.ok) expect(stale.failure.code).toBe("chart::expired");
  const unknown = await client.select("r-999", 0);
  expect(unknown.ok).toBe(false);
  if (!unknown.ok) expect(unknown.failure.code).toBe("chart::expired");
  const badIndex = await client.select("r-999", 7);
  expect(badIndex.ok).toBe(false);
});

test("out-of-range index rejects against a live render", async () => {
  const { client } = pair("corr-2004");
  const rendered = await client.render("t", POINTS);
  if (!rendered.ok) throw new Error("render failed");
  const got = await client.select(rendered.value.renderId, 7);
  expect(got.ok).toBe(false);
  if (!got.ok) expect(got.failure.code).toBe("chart::invalid_spec");
});

test("denied destination, bad credential, and offline fail closed", async () => {
  const readEnvironment = (key: string): string | undefined =>
    key === "D01_CHART_COMPANION_TOKEN" ? SECRET : undefined;
  let sent = 0;
  const denied = createChartCompanionClient({
    policy: chartCompanionPolicy(),
    endpoint: "https://evil.example/v1/chart",
    context: chartCompanionContext("corr-2005"),
    credentialEnv: envName("D01_CHART_COMPANION_TOKEN"),
    readEnvironment,
    transport: { send: () => { sent += 1; return new Uint8Array(); } },
  });
  const deniedResult = await denied.render("t", POINTS);
  expect(deniedResult.ok).toBe(false);
  if (!deniedResult.ok) expect(deniedResult.failure.code).toBe("companion::destination_denied");
  expect(sent).toBe(0);

  const strict = createChartCompanionServer({
    policy: chartCompanionPolicy(),
    credentialEnv: envName("D01_CHART_COMPANION_TOKEN"),
    readEnvironment: (key: string) => (key === "D01_CHART_COMPANION_TOKEN" ? SECRET : undefined),
  });
  const wrong = createChartCompanionClient({
    policy: chartCompanionPolicy(),
    endpoint: ENDPOINT,
    context: chartCompanionContext("corr-2006"),
    credentialEnv: envName("D01_CHART_COMPANION_TOKEN"),
    readEnvironment: () => "WRONG-VALUE",
    transport: { send: (url: string, authorization: string, body: Uint8Array) => strict.handle(authorization, body) },
  });
  const wrongResult = await wrong.render("t", POINTS);
  expect(wrongResult.ok).toBe(false);
  if (!wrongResult.ok) expect(wrongResult.failure.code).toBe("companion::unauthorized");

  const offline = createChartCompanionClient({
    policy: chartCompanionPolicy(),
    endpoint: ENDPOINT,
    context: chartCompanionContext("corr-2007"),
    credentialEnv: envName("D01_CHART_COMPANION_TOKEN"),
    readEnvironment,
    transport: { send: () => { throw new Error("offline"); } },
  });
  const offlineResult = await offline.render("t", POINTS);
  expect(offlineResult.ok).toBe(false);
  if (!offlineResult.ok) {
    expect(offlineResult.failure.code).toBe("companion::transport_failed");
    expect(JSON.stringify(offlineResult)).not.toContain("offline");
  }
});

test("version mismatch fails closed both directions", () => {
  const request = decodeChartRequest(new TextEncoder().encode(JSON.stringify({ v: "d01.chart/0", op: "render" })));
  expect("code" in request && request.code).toBe("companion::version_mismatch");
  const response = decodeChartResponse(
    new TextEncoder().encode(JSON.stringify({ v: "d01.chart/0", correlation: "c", ok: true })),
  );
  expect("code" in response && response.code).toBe("companion::version_mismatch");
});

test("auth parity: same policy both sides; secret never logged", async () => {
  const policy = chartCompanionPolicy();
  const { client, server } = pair("corr-2008");
  expect(evaluateDestination(policy, ENDPOINT).decision).toBe("allowed");
  expect(server.policyVersion).toBe(policy.version);
  expect((await client.render("t", POINTS)).ok).toBe(true);
  const exposed = JSON.stringify({
    report: policyReport(policy),
    decision: evaluateDestination(policy, ENDPOINT),
    diagnostics: server.diagnostics(),
  });
  expect(exposed).not.toContain(SECRET);
  expect(exposed).toContain("D01_CHART_COMPANION_TOKEN");
});

test("envelope bytes are deterministic and measured", () => {
  const request = encodeChartRequest({
    v: CHART_PROTOCOL,
    tenant: "tenant-acme",
    pool: "pool-default",
    correlation: "corr-2001",
    op: "render",
    title: "Invoice totals",
    kind: "bar",
    points: POINTS,
  });
  const response = encodeChartResponse({
    v: CHART_PROTOCOL,
    correlation: "corr-2001",
    ok: true,
    render: { renderId: "r-1", svg: "<svg/>", table: POINTS },
  });
  expect(request.length).toBe(209);
  expect(response.length).toBe(164);
});
