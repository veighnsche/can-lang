// D03 conformance: Widget C second vendor (Vendor B) through the assigned
// T3 path. Every leg drives Vendor B with the UNCHANGED D02 client — the
// client import below comes from `chart.ts` only; no Vendor B client
// exists. Per-vendor config (policy, endpoint, credential env name) travels
// as init data. Lifecycle, auth, version, and trip-wire legs mirror the
// D02 chart legs so the two vendors are comparable leg-for-leg; the
// cross-vendor legs pin isolation between them.
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
  decodeChartResponse,
  encodeChartResponse,
  CHART_PROTOCOL,
  CHART_SELECT_ROUNDTRIP_BUDGET_MS,
  type ChartCompanionServer,
  type CompanionTransport,
} from "../companions/chart.ts";
import {
  chartVendorBPolicy,
  createChartVendorBServer,
  CHART_VENDOR_B_CREDENTIAL_ENV,
  CHART_VENDOR_B_ENDPOINT,
  CHART_VENDOR_B_POLICY_VERSION,
} from "../companions/chart-vendor-b.ts";

const SECRET_A = "D02-PROBE-CHART-SECRET-9c1e77";
const SECRET_B = "D03-PROBE-VENDOR-B-SECRET-41f0c2";
const ENDPOINT_A = "https://companion.example/v1/chart";
const POINTS = [
  { label: "Jan", value: 120 },
  { label: "Feb", value: 200 },
];

function vendorBPair(correlation = "corr-3001") {
  const readEnvironment = (key: string): string | undefined =>
    key === CHART_VENDOR_B_CREDENTIAL_ENV ? SECRET_B : undefined;
  const server = createChartVendorBServer({
    policy: chartVendorBPolicy(),
    credentialEnv: envName(CHART_VENDOR_B_CREDENTIAL_ENV),
    readEnvironment,
  });
  const transport: CompanionTransport = {
    async send(url: string, authorization: string, body: Uint8Array): Promise<Uint8Array> {
      expect(url).toBe(CHART_VENDOR_B_ENDPOINT);
      return server.handle(authorization, body);
    },
  };
  // The D02 client, unchanged: only init config names Vendor B.
  const client = createChartCompanionClient({
    policy: chartVendorBPolicy(),
    endpoint: CHART_VENDOR_B_ENDPOINT,
    context: chartCompanionContext(correlation),
    credentialEnv: envName(CHART_VENDOR_B_CREDENTIAL_ENV),
    readEnvironment,
    transport,
  });
  return { client, server };
}

function vendorAPair(correlation = "corr-3101") {
  const readEnvironment = (key: string): string | undefined =>
    key === "D02_CHART_COMPANION_TOKEN" ? SECRET_A : undefined;
  const server = createChartCompanionServer({
    policy: chartCompanionPolicy(),
    credentialEnv: envName("D02_CHART_COMPANION_TOKEN"),
    readEnvironment,
  });
  const transport: CompanionTransport = {
    async send(url: string, authorization: string, body: Uint8Array): Promise<Uint8Array> {
      expect(url).toBe(ENDPOINT_A);
      return server.handle(authorization, body);
    },
  };
  const client = createChartCompanionClient({
    policy: chartCompanionPolicy(),
    endpoint: ENDPOINT_A,
    context: chartCompanionContext(correlation),
    credentialEnv: envName("D02_CHART_COMPANION_TOKEN"),
    readEnvironment,
    transport,
  });
  return { client, server };
}

test("vendor B: render, select, release roundtrip through the unchanged D02 client", async () => {
  const { client, server } = vendorBPair();
  const rendered = await client.render("Invoice totals", POINTS);
  expect(rendered.ok).toBe(true);
  if (!rendered.ok) throw new Error("vendor-B render failed");
  expect(rendered.value.renderId).toMatch(/^vb-/);
  expect(rendered.value.svg).toContain("<svg");
  expect(rendered.value.svg).toContain('data-vendor="b"');
  expect(rendered.value.table).toEqual(POINTS);
  expect(Object.isFrozen(rendered.value)).toBe(true);
  const selected = await client.select(rendered.value.renderId, 1);
  expect(selected).toEqual({ ok: true, value: { label: "Feb", value: 200 } });
  expect((await client.release(rendered.value.renderId)).ok).toBe(true);
  expect(server.liveRenders()).toBe(0);
});

test("vendor B: engine is deterministic and structurally distinct from vendor A", async () => {
  const { client } = vendorBPair("corr-3002");
  const first = await client.render("t", POINTS);
  const second = await client.render("t", POINTS);
  if (!first.ok || !second.ok) throw new Error("vendor-B render failed");
  // Same spec renders byte-identical SVG on repeated renders.
  expect(first.value.svg).toBe(second.value.svg);
  // Vendor B shape markers: grouped horizontal bars, axis line, value labels.
  expect(first.value.svg).toContain('<g class="bar"');
  expect(first.value.svg).toContain("<line");
  expect(first.value.svg).toContain("<text");
  // A genuinely different engine, not a relabeled Vendor A.
  const vendorA = vendorAPair("corr-3102");
  const baseline = await vendorA.client.render("t", POINTS);
  if (!baseline.ok) throw new Error("vendor-A render failed");
  expect(baseline.value.svg).not.toBe(first.value.svg);
  expect(baseline.value.svg).not.toContain('data-vendor="b"');
});

test("vendor B: invalid specs reject client-side without sending", async () => {
  let sent = 0;
  const readEnvironment = (key: string): string | undefined =>
    key === CHART_VENDOR_B_CREDENTIAL_ENV ? SECRET_B : undefined;
  const { server } = vendorBPair("corr-3003");
  const client = createChartCompanionClient({
    policy: chartVendorBPolicy(),
    endpoint: CHART_VENDOR_B_ENDPOINT,
    context: chartCompanionContext("corr-3003"),
    credentialEnv: envName(CHART_VENDOR_B_CREDENTIAL_ENV),
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

test("vendor B: lifecycle — expiry, idempotent release, range, attribution", async () => {
  const { client, server } = vendorBPair("corr-3004");
  const rendered = await client.render("t", POINTS);
  if (!rendered.ok) throw new Error("vendor-B render failed");
  expect((await client.release(rendered.value.renderId)).ok).toBe(true);
  expect((await client.release(rendered.value.renderId)).ok).toBe(true);
  const stale = await client.select(rendered.value.renderId, 0);
  expect(stale.ok).toBe(false);
  if (!stale.ok) expect(stale.failure.code).toBe("chart::expired");
  const unknown = await client.select("vb-999", 0);
  expect(unknown.ok).toBe(false);
  if (!unknown.ok) expect(unknown.failure.code).toBe("chart::expired");
  const live = await client.render("t", POINTS);
  if (!live.ok) throw new Error("vendor-B render failed");
  const badIndex = await client.select(live.value.renderId, 7);
  expect(badIndex.ok).toBe(false);
  if (!badIndex.ok) expect(badIndex.failure.code).toBe("chart::invalid_spec");
  expect((await client.release(live.value.renderId)).ok).toBe(true);
  expect(server.liveRenders()).toBe(0);
  const diagnostics = server.diagnostics();
  expect(diagnostics.length).toBeGreaterThan(0);
  for (const line of diagnostics) expect(line.startsWith("vendor-b ")).toBe(true);
  expect(JSON.stringify(diagnostics)).not.toContain(SECRET_B);
});

test("vendor B: denied destination, bad credential, and offline fail closed", async () => {
  const readEnvironment = (key: string): string | undefined =>
    key === CHART_VENDOR_B_CREDENTIAL_ENV ? SECRET_B : undefined;
  let sent = 0;
  const denied = createChartCompanionClient({
    policy: chartVendorBPolicy(),
    endpoint: "https://evil.example/v1/chart",
    context: chartCompanionContext("corr-3005"),
    credentialEnv: envName(CHART_VENDOR_B_CREDENTIAL_ENV),
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

  const strict = createChartVendorBServer({
    policy: chartVendorBPolicy(),
    credentialEnv: envName(CHART_VENDOR_B_CREDENTIAL_ENV),
    readEnvironment: (key: string) => (key === CHART_VENDOR_B_CREDENTIAL_ENV ? SECRET_B : undefined),
  });
  const wrong = createChartCompanionClient({
    policy: chartVendorBPolicy(),
    endpoint: CHART_VENDOR_B_ENDPOINT,
    context: chartCompanionContext("corr-3006"),
    credentialEnv: envName(CHART_VENDOR_B_CREDENTIAL_ENV),
    readEnvironment: () => "WRONG-VALUE",
    transport: {
      send: async (url: string, authorization: string, body: Uint8Array) =>
        strict.handle(authorization, body),
    },
  });
  const wrongResult = await wrong.render("t", POINTS);
  expect(wrongResult.ok).toBe(false);
  if (!wrongResult.ok) expect(wrongResult.failure.code).toBe("companion::unauthorized");

  // Vendor A's credential authorizes nothing on Vendor B: per-vendor secrets.
  const crossWired = createChartCompanionClient({
    policy: chartVendorBPolicy(),
    endpoint: CHART_VENDOR_B_ENDPOINT,
    context: chartCompanionContext("corr-3007"),
    credentialEnv: envName(CHART_VENDOR_B_CREDENTIAL_ENV),
    readEnvironment: () => SECRET_A,
    transport: {
      send: async (url: string, authorization: string, body: Uint8Array) =>
        strict.handle(authorization, body),
    },
  });
  const crossResult = await crossWired.render("t", POINTS);
  expect(crossResult.ok).toBe(false);
  if (!crossResult.ok) expect(crossResult.failure.code).toBe("companion::unauthorized");

  const offline = createChartCompanionClient({
    policy: chartVendorBPolicy(),
    endpoint: CHART_VENDOR_B_ENDPOINT,
    context: chartCompanionContext("corr-3008"),
    credentialEnv: envName(CHART_VENDOR_B_CREDENTIAL_ENV),
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

test("vendor B: unknown server codes sanitize; version mismatch fails closed", async () => {
  const readEnvironment = (key: string): string | undefined =>
    key === CHART_VENDOR_B_CREDENTIAL_ENV ? SECRET_B : undefined;
  const client = createChartCompanionClient({
    policy: chartVendorBPolicy(),
    endpoint: CHART_VENDOR_B_ENDPOINT,
    context: chartCompanionContext("corr-3009"),
    credentialEnv: envName(CHART_VENDOR_B_CREDENTIAL_ENV),
    readEnvironment,
    transport: {
      send: async () =>
        encodeChartResponse({
          v: CHART_PROTOCOL,
          correlation: "corr-3009",
          ok: false,
          code: "chart::bogus",
          detail: "server says",
        }),
    },
  });
  const bogus = await client.render("t", POINTS);
  expect(bogus.ok).toBe(false);
  if (!bogus.ok) expect(bogus.failure.code).toBe("companion::protocol_error");

  // Raw wire probe against the Vendor B server: a stale protocol version
  // gets a typed version_mismatch envelope, never a render.
  const server = createChartVendorBServer({
    policy: chartVendorBPolicy(),
    credentialEnv: envName(CHART_VENDOR_B_CREDENTIAL_ENV),
    readEnvironment,
  });
  const raw = server.handle(
    `Bearer ${SECRET_B}`,
    new TextEncoder().encode(
      JSON.stringify({
        v: "d02.chart/0",
        op: "render",
        tenant: "tenant-acme",
        pool: "pool-default",
        correlation: "corr-3010",
      }),
    ),
  );
  const decoded = decodeChartResponse(raw);
  expect("code" in decoded && decoded.code).toBe("companion::version_mismatch");
});

test("vendor B: auth parity — same policy both sides; secret never logged", async () => {
  const policy = chartVendorBPolicy();
  expect(policy.version).toBe(CHART_VENDOR_B_POLICY_VERSION);
  const { client, server } = vendorBPair("corr-3011");
  expect(evaluateDestination(policy, CHART_VENDOR_B_ENDPOINT).decision).toBe("allowed");
  expect(server.policyVersion).toBe(policy.version);
  expect((await client.render("t", POINTS)).ok).toBe(true);
  const exposed = JSON.stringify({
    report: policyReport(policy),
    decision: evaluateDestination(policy, CHART_VENDOR_B_ENDPOINT),
    diagnostics: server.diagnostics(),
  });
  expect(exposed).not.toContain(SECRET_B);
  expect(exposed).toContain(CHART_VENDOR_B_CREDENTIAL_ENV);
});

test("cross-vendor: renders are server-local; foreign release never touches the owner", async () => {
  const a = vendorAPair("corr-3111");
  const b = vendorBPair("corr-3012");
  const renderedA = await a.client.render("t", POINTS);
  if (!renderedA.ok) throw new Error("vendor-A render failed");
  // Vendor B knows nothing of Vendor A's render id.
  const foreign = await b.client.select(renderedA.value.renderId, 0);
  expect(foreign.ok).toBe(false);
  if (!foreign.ok) expect(foreign.failure.code).toBe("chart::expired");
  // Releasing a foreign id is idempotent-ok on B...
  expect((await b.client.release(renderedA.value.renderId)).ok).toBe(true);
  // ...and leaves the owning vendor's render live.
  const stillLive = await a.client.select(renderedA.value.renderId, 0);
  expect(stillLive).toEqual({ ok: true, value: { label: "Jan", value: 120 } });
  expect(a.server.liveRenders()).toBe(1);
  expect(b.server.liveRenders()).toBe(0);
  expect((await a.client.release(renderedA.value.renderId)).ok).toBe(true);
  expect(a.server.liveRenders()).toBe(0);
});

// serveLoopback hosts one Vendor B server on 127.0.0.1 over real HTTP for
// the wire legs below. The loopback-allow policy is conformance-only:
// production serves the pinned https destination with loopback denied.
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
    version: "2026-09-26.d03-chart-vendor-b-loopback",
    rules: [
      {
        scheme: "http",
        host: "127.0.0.1",
        port,
        pathPrefix: "/v1/",
        credential: "D03_CHART_VENDOR_B_TOKEN",
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
    credentialEnv: envName(CHART_VENDOR_B_CREDENTIAL_ENV),
    readEnvironment,
    transport: createFetchTransport(),
  });
}

test("vendor B loopback HTTP: render, select, release roundtrip over the wire", async () => {
  const readEnvironment = (key: string): string | undefined =>
    key === CHART_VENDOR_B_CREDENTIAL_ENV ? SECRET_B : undefined;
  const server = createChartVendorBServer({
    policy: chartVendorBPolicy(),
    credentialEnv: envName(CHART_VENDOR_B_CREDENTIAL_ENV),
    readEnvironment,
  });
  const loop = await serveLoopback(server);
  try {
    const client = loopbackClient(loop.url, "corr-3201", readEnvironment);
    const rendered = await client.render("Invoice totals", POINTS);
    expect(rendered.ok).toBe(true);
    if (!rendered.ok) throw new Error("wire render failed");
    expect(rendered.value.renderId).toMatch(/^vb-/);
    expect(rendered.value.svg).toContain('data-vendor="b"');
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

test("vendor B loopback HTTP: wrong credential and wrong version fail closed over the wire", async () => {
  const server = createChartVendorBServer({
    policy: chartVendorBPolicy(),
    credentialEnv: envName(CHART_VENDOR_B_CREDENTIAL_ENV),
    readEnvironment: (key: string) => (key === CHART_VENDOR_B_CREDENTIAL_ENV ? SECRET_B : undefined),
  });
  const loop = await serveLoopback(server);
  try {
    const wrong = loopbackClient(loop.url, "corr-3202", () => "WRONG-VALUE");
    const denied = await wrong.render("t", POINTS);
    expect(denied.ok).toBe(false);
    if (!denied.ok) expect(denied.failure.code).toBe("companion::unauthorized");

    const raw = await fetch(loop.url, {
      method: "POST",
      headers: { "content-type": "application/json", authorization: `Bearer ${SECRET_B}` },
      body: JSON.stringify({
        v: "d02.chart/0",
        op: "render",
        tenant: "tenant-acme",
        pool: "pool-default",
        correlation: "corr-3203",
      }),
    });
    expect(raw.status).toBe(200);
    const decoded = decodeChartResponse(new Uint8Array(await raw.arrayBuffer()));
    expect("code" in decoded && decoded.code).toBe("companion::version_mismatch");
  } finally {
    loop.close();
  }
});

test("vendor B loopback HTTP: select roundtrip stays under the trip-wire budget", async () => {
  const readEnvironment = (key: string): string | undefined =>
    key === CHART_VENDOR_B_CREDENTIAL_ENV ? SECRET_B : undefined;
  const server = createChartVendorBServer({
    policy: chartVendorBPolicy(),
    credentialEnv: envName(CHART_VENDOR_B_CREDENTIAL_ENV),
    readEnvironment,
  });
  const loop = await serveLoopback(server);
  try {
    const client = loopbackClient(loop.url, "corr-3204", readEnvironment);
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
    // The budget applies per vendor: either vendor tripping reopens the
    // X-R01-1 tier choice. The measured worst roundtrip is recorded in
    // d03-record.md alongside the verdict.
    expect(worst).toBeLessThan(CHART_SELECT_ROUNDTRIP_BUDGET_MS);
  } finally {
    loop.close();
  }
});
