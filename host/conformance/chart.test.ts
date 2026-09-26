// D02 conformance: Widget C chart companion. The first eight legs
// re-run the D01 T3 legs against the delivered `d02.chart/1` contract
// (protocol/version/env renames applied; validators, codecs, and
// failure leaves verbatim); the trailing legs prove the boundary over
// real loopback HTTP and enforce the select-roundtrip trip wire.
// The renderer behind the boundary stays a stand-in: the legs pin the
// boundary shape (opaque bytes plus typed table), never pixels.
import { test, expect } from "bun:test";
import {
  destinationPolicy,
  envName,
  evaluateDestination,
  policyReport,
} from "../../runtime/outbound/destination-policy.ts";
import {
  chartCompanionContext,
  chartCompanionPolicy,
  createChartCompanionClient,
  createChartCompanionServer,
  createFetchTransport,
  decodeChartRequest,
  decodeChartResponse,
  encodeChartRequest,
  encodeChartResponse,
  CHART_PROTOCOL,
  CHART_SELECT_ROUNDTRIP_BUDGET_MS,
  type ChartCompanionServer,
  type CompanionTransport,
} from "../companions/chart.ts";

const SECRET = "D02-PROBE-CHART-SECRET-9c1e77";
const ENDPOINT = "https://companion.example/v1/chart";
const POINTS = [
  { label: "Jan", value: 120 },
  { label: "Feb", value: 200 },
];

function pair(correlation = "corr-2001") {
  const readEnvironment = (key: string): string | undefined =>
    key === "D02_CHART_COMPANION_TOKEN" ? SECRET : undefined;
  const server = createChartCompanionServer({
    policy: chartCompanionPolicy(),
    credentialEnv: envName("D02_CHART_COMPANION_TOKEN"),
    readEnvironment,
  });
  const transport: CompanionTransport = {
    async send(url: string, authorization: string, body: Uint8Array): Promise<Uint8Array> {
      expect(url).toBe(ENDPOINT);
      return server.handle(authorization, body);
    },
  };
  const client = createChartCompanionClient({
    policy: chartCompanionPolicy(),
    endpoint: ENDPOINT,
    context: chartCompanionContext(correlation),
    credentialEnv: envName("D02_CHART_COMPANION_TOKEN"),
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
    key === "D02_CHART_COMPANION_TOKEN" ? SECRET : undefined;
  const { server } = pair("corr-2002");
  const client = createChartCompanionClient({
    policy: chartCompanionPolicy(),
    endpoint: ENDPOINT,
    context: chartCompanionContext("corr-2002"),
    credentialEnv: envName("D02_CHART_COMPANION_TOKEN"),
    readEnvironment,
    transport: {
      send: async (url: string, authorization: string, body: Uint8Array) => {
        sent += 1;
        return server.handle(authorization, body);
      },
    },
  });
  for (const [title, points] of [
    ["", POINTS],
    ["t", []],
    ["t", [{ label: "x", value: NaN }]],
    [null, POINTS],
  ] as const) {
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
    key === "D02_CHART_COMPANION_TOKEN" ? SECRET : undefined;
  let sent = 0;
  const denied = createChartCompanionClient({
    policy: chartCompanionPolicy(),
    endpoint: "https://evil.example/v1/chart",
    context: chartCompanionContext("corr-2005"),
    credentialEnv: envName("D02_CHART_COMPANION_TOKEN"),
    readEnvironment,
    transport: {
      send: async () => {
        sent += 1;
        return new Uint8Array();
      },
    },
  });
  const deniedResult = await denied.render("t", POINTS);
  expect(deniedResult.ok).toBe(false);
  if (!deniedResult.ok) expect(deniedResult.failure.code).toBe("companion::destination_denied");
  expect(sent).toBe(0);

  const strict = createChartCompanionServer({
    policy: chartCompanionPolicy(),
    credentialEnv: envName("D02_CHART_COMPANION_TOKEN"),
    readEnvironment: (key: string) => (key === "D02_CHART_COMPANION_TOKEN" ? SECRET : undefined),
  });
  const wrong = createChartCompanionClient({
    policy: chartCompanionPolicy(),
    endpoint: ENDPOINT,
    context: chartCompanionContext("corr-2006"),
    credentialEnv: envName("D02_CHART_COMPANION_TOKEN"),
    readEnvironment: () => "WRONG-VALUE",
    transport: {
      send: async (url: string, authorization: string, body: Uint8Array) =>
        strict.handle(authorization, body),
    },
  });
  const wrongResult = await wrong.render("t", POINTS);
  expect(wrongResult.ok).toBe(false);
  if (!wrongResult.ok) expect(wrongResult.failure.code).toBe("companion::unauthorized");

  const offline = createChartCompanionClient({
    policy: chartCompanionPolicy(),
    endpoint: ENDPOINT,
    context: chartCompanionContext("corr-2007"),
    credentialEnv: envName("D02_CHART_COMPANION_TOKEN"),
    readEnvironment,
    transport: {
      send: async () => {
        throw new Error("offline");
      },
    },
  });
  const offlineResult = await offline.render("t", POINTS);
  expect(offlineResult.ok).toBe(false);
  if (!offlineResult.ok) {
    expect(offlineResult.failure.code).toBe("companion::transport_failed");
    expect(JSON.stringify(offlineResult)).not.toContain("offline");
  }
});

test("unknown server codes sanitize to protocol_error; known codes pass through", async () => {
  const readEnvironment = (key: string): string | undefined =>
    key === "D02_CHART_COMPANION_TOKEN" ? SECRET : undefined;
  const clientFor = (code: string) =>
    createChartCompanionClient({
      policy: chartCompanionPolicy(),
      endpoint: ENDPOINT,
      context: chartCompanionContext("corr-2009"),
      credentialEnv: envName("D02_CHART_COMPANION_TOKEN"),
      readEnvironment,
      transport: {
        send: async () =>
          encodeChartResponse({
            v: CHART_PROTOCOL,
            correlation: "corr-2009",
            ok: false,
            code,
            detail: "server says",
          }),
      },
    });
  const bogus = await clientFor("chart::bogus").render("t", POINTS);
  expect(bogus.ok).toBe(false);
  if (!bogus.ok) expect(bogus.failure.code).toBe("companion::protocol_error");
  const known = await clientFor("chart::expired").render("t", POINTS);
  expect(known.ok).toBe(false);
  if (!known.ok) expect(known.failure.code).toBe("chart::expired");
});

test("version mismatch fails closed both directions", () => {
  const request = decodeChartRequest(
    new TextEncoder().encode(JSON.stringify({ v: "d02.chart/0", op: "render" })),
  );
  expect("code" in request && request.code).toBe("companion::version_mismatch");
  const response = decodeChartResponse(
    new TextEncoder().encode(JSON.stringify({ v: "d02.chart/0", correlation: "c", ok: true })),
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
  expect(exposed).toContain("D02_CHART_COMPANION_TOKEN");
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
  // `d02.chart/1` is the same length as the `d01.chart/1` prototype
  // wire, so the D01 pins carry over byte-for-byte.
  expect(request.length).toBe(209);
  expect(response.length).toBe(164);
});

// serveLoopback hosts one chart server on 127.0.0.1 over real HTTP for
// the wire legs below. The loopback-allow policy is conformance-only:
// production serves the pinned https destination with loopback
// denied (see chart-recipe.md).
async function serveLoopback(
  server: ChartCompanionServer,
): Promise<{ url: string; close(): void }> {
  const listener = Bun.serve({
    hostname: "127.0.0.1",
    port: 0,
    async fetch(request: Request): Promise<Response> {
      if (request.method !== "POST") return new Response("method not allowed", { status: 405 });
      const body = new Uint8Array(await request.arrayBuffer());
      const out = server.handle(request.headers.get("authorization") ?? undefined, body);
      return new Response(out as BodyInit, {
        status: 200,
        headers: { "content-type": "application/json" },
      });
    },
  });
  return {
    url: `http://127.0.0.1:${listener.port}/v1/chart`,
    close(): void {
      listener.stop();
    },
  };
}

function loopbackClient(
  url: string,
  correlation: string,
  readEnvironment: (key: string) => string | undefined,
) {
  const port = Number(new URL(url).port);
  const policy = destinationPolicy({
    version: "2026-09-26.d02-chart-loopback",
    rules: [
      {
        scheme: "http",
        host: "127.0.0.1",
        port,
        pathPrefix: "/v1/",
        credential: "D02_CHART_COMPANION_TOKEN",
      },
    ],
    redirect: "deny",
    maxRedirectHops: 0,
    loopback: "allow",
    privateNetworks: "allow",
  });
  return createChartCompanionClient({
    policy,
    endpoint: url,
    context: chartCompanionContext(correlation),
    credentialEnv: envName("D02_CHART_COMPANION_TOKEN"),
    readEnvironment,
    transport: createFetchTransport(),
  });
}

test("loopback HTTP: render, select, release roundtrip over the wire", async () => {
  const readEnvironment = (key: string): string | undefined =>
    key === "D02_CHART_COMPANION_TOKEN" ? SECRET : undefined;
  const server = createChartCompanionServer({
    policy: chartCompanionPolicy(),
    credentialEnv: envName("D02_CHART_COMPANION_TOKEN"),
    readEnvironment,
  });
  const loop = await serveLoopback(server);
  try {
    const client = loopbackClient(loop.url, "corr-2101", readEnvironment);
    const rendered = await client.render("Invoice totals", POINTS);
    expect(rendered.ok).toBe(true);
    if (!rendered.ok) throw new Error("wire render failed");
    expect(rendered.value.svg).toContain("<svg");
    expect(rendered.value.table).toEqual(POINTS);
    const selected = await client.select(rendered.value.renderId, 1);
    expect(selected).toEqual({ ok: true, value: { label: "Feb", value: 200 } });
    expect((await client.release(rendered.value.renderId)).ok).toBe(true);
    expect((await client.release(rendered.value.renderId)).ok).toBe(true);
    const stale = await client.select(rendered.value.renderId, 0);
    expect(stale.ok).toBe(false);
    if (!stale.ok) expect(stale.failure.code).toBe("chart::expired");
    expect(server.liveRenders()).toBe(0);
  } finally {
    loop.close();
  }
});

test("loopback HTTP: wrong credential and wrong version fail closed over the wire", async () => {
  const server = createChartCompanionServer({
    policy: chartCompanionPolicy(),
    credentialEnv: envName("D02_CHART_COMPANION_TOKEN"),
    readEnvironment: (key: string) => (key === "D02_CHART_COMPANION_TOKEN" ? SECRET : undefined),
  });
  const loop = await serveLoopback(server);
  try {
    const wrong = loopbackClient(loop.url, "corr-2102", () => "WRONG-VALUE");
    const denied = await wrong.render("t", POINTS);
    expect(denied.ok).toBe(false);
    if (!denied.ok) expect(denied.failure.code).toBe("companion::unauthorized");

    // Raw wire probe: a stale protocol version gets a typed
    // version_mismatch envelope, never a render.
    const raw = await fetch(loop.url, {
      method: "POST",
      headers: { "content-type": "application/json", authorization: `Bearer ${SECRET}` },
      body: JSON.stringify({
        v: "d02.chart/0",
        op: "render",
        tenant: "tenant-acme",
        pool: "pool-default",
        correlation: "corr-2103",
      }),
    });
    expect(raw.status).toBe(200);
    const decoded = decodeChartResponse(new Uint8Array(await raw.arrayBuffer()));
    expect("code" in decoded && decoded.code).toBe("companion::version_mismatch");
  } finally {
    loop.close();
  }
});

test("loopback HTTP: select roundtrip stays under the trip-wire budget", async () => {
  const readEnvironment = (key: string): string | undefined =>
    key === "D02_CHART_COMPANION_TOKEN" ? SECRET : undefined;
  const server = createChartCompanionServer({
    policy: chartCompanionPolicy(),
    credentialEnv: envName("D02_CHART_COMPANION_TOKEN"),
    readEnvironment,
  });
  const loop = await serveLoopback(server);
  try {
    const client = loopbackClient(loop.url, "corr-2104", readEnvironment);
    const rendered = await client.render("t", POINTS);
    if (!rendered.ok) throw new Error("wire render failed");
    let worst = 0;
    for (let index = 0; index < 25; index += 1) {
      const start = performance.now();
      const selected = await client.select(rendered.value.renderId, index % POINTS.length);
      const elapsed = performance.now() - start;
      expect(selected.ok).toBe(true);
      if (elapsed > worst) worst = elapsed;
    }
    // The budget guards the payload contract against structural
    // regression; the measured worst roundtrip is recorded in
    // d02-record.md alongside the verdict.
    expect(worst).toBeLessThan(CHART_SELECT_ROUNDTRIP_BUDGET_MS);
  } finally {
    loop.close();
  }
});
