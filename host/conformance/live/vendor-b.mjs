// D03 live-browser leg: the DELIVERED D02 client drives Vendor B from REAL
// page context (fetch from the page to a loopback Vendor B server) in one
// named Playwright browser. The Go gate (`live_vendor_b_test.go`) bundles
// the page entry (`vendor-b-page-entry.ts`) and the Vendor B server
// (`host/companions/chart-vendor-b.ts`) with `bun build`, then this runner
// serves a page importing the client bundle plus the `/v1/chart` companion
// route on the same loopback origin (same-origin fetch, no CORS), and drives
// every check through the in-page client.
// Usage:
//   node vendor-b.mjs <chromium|webkit|firefox> <bundle-dir> <outdir>
// Writes report.json into outdir; exits nonzero on any failed check.
// Pinned but UNRUN while C01 is blocked-open: see README.md.
import { strict as assert } from "node:assert";
import { mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import { createServer } from "node:http";
import { pathToFileURL } from "node:url";

const [wanted, bundleDir, outdir] = process.argv.slice(2);
if (
  (wanted !== "chromium" && wanted !== "webkit" && wanted !== "firefox") ||
  !bundleDir ||
  !outdir
) {
  console.error("usage: node vendor-b.mjs <chromium|webkit|firefox> <bundle-dir> <outdir>");
  process.exit(2);
}
mkdirSync(outdir, { recursive: true });

// The live secret lives in the runner only: the server reads it from the
// closure below and the page receives it as an evaluate argument — never
// baked into the page source, the bundles, or the report.
const SECRET = "D03-LIVE-VENDOR-B-SECRET-77aa10";
const readEnvironment = (key) => (key === "D03_CHART_VENDOR_B_TOKEN" ? SECRET : undefined);
const vendorB = await import(pathToFileURL(join(bundleDir, "vendor-b-server.bundle.mjs")).href);
const server = vendorB.createChartVendorBServer({
  policy: vendorB.chartVendorBPolicy(),
  credentialEnv: "D03_CHART_VENDOR_B_TOKEN",
  readEnvironment,
});

const clientBundle = readFileSync(join(bundleDir, "vendor-b-client.bundle.js"));
const pageHtml = `<!doctype html><html><body><script type="module">
import * as client from "/vendor-b-client.bundle.js";
window.__d03_client = client;
window.__d03_ready = true;
</script></body></html>`;
const http = createServer((request, response) => {
  if (request.url === "/") {
    response.writeHead(200, { "content-type": "text/html" }).end(pageHtml);
    return;
  }
  if (request.url === "/vendor-b-client.bundle.js") {
    response.writeHead(200, { "content-type": "text/javascript" }).end(clientBundle);
    return;
  }
  if (request.url === "/v1/chart" && request.method === "POST") {
    const chunks = [];
    request.on("data", (chunk) => chunks.push(chunk));
    request.on("end", () => {
      const out = server.handle(
        request.headers["authorization"] ?? undefined,
        new Uint8Array(Buffer.concat(chunks)),
      );
      response.writeHead(200, { "content-type": "application/json" }).end(Buffer.from(out));
    });
    return;
  }
  response.writeHead(404).end("not found");
});
await new Promise((resolve) => http.listen(0, "127.0.0.1", resolve));
const base = `http://127.0.0.1:${http.address().port}`;

const playwright = await import("playwright");
const browser = await playwright[wanted].launch({ timeout: 120000 });
const checks = [];
const check = (name, fn) =>
  Promise.resolve()
    .then(fn)
    .then((detail = "") => checks.push({ name, passed: true, detail }))
    .catch((error) => checks.push({ name, passed: false, detail: String(error?.message ?? error) }));

// buildClient runs in page context: the delivered D02 client configured
// for Vendor B over loopback fetch. It is serialized as a string so the
// page construction stays visible here, in the pinned runner.
const buildClient = `
async (endpoint, secret, correlation) => {
  const api = window.__d03_client;
  const port = Number(new URL(endpoint).port);
  const policy = api.destinationPolicy({
    version: "2026-09-26.d03-chart-vendor-b-live",
    rules: [{ scheme: "http", host: "127.0.0.1", port, pathPrefix: "/v1/", credential: "D03_CHART_VENDOR_B_TOKEN" }],
    redirect: "deny",
    maxRedirectHops: 0,
    loopback: "allow",
    privateNetworks: "allow",
  });
  return api.createChartCompanionClient({
    policy,
    endpoint,
    context: api.chartCompanionContext(correlation),
    credentialEnv: api.envName("D03_CHART_VENDOR_B_TOKEN"),
    readEnvironment: (key) => (key === "D03_CHART_VENDOR_B_TOKEN" ? secret : undefined),
    transport: api.createFetchTransport(),
  });
}
`;

try {
  const context = await browser.newContext();
  const page = await context.newPage();
  const pageerrors = [];
  page.on("pageerror", (error) => pageerrors.push(String(error?.message ?? error)));
  await page.goto(base + "/", { waitUntil: "load" });
  await page.waitForFunction(() => window.__d03_ready === true, null, { timeout: 30000 });

  const userAgent = await page.evaluate(() => navigator.userAgent);
  const secureContext = await page.evaluate(() => window.isSecureContext);
  const endpoint = `${base}/v1/chart`;
  const points = [
    { label: "Jan", value: 120 },
    { label: "Feb", value: 200 },
  ];

  await check("page client: render, select, release roundtrip over loopback fetch", () =>
    page.evaluate(
      async ({ endpoint, secret, points, buildClient }) => {
        const localAssert = (cond, message) => {
          if (!cond) throw new Error(message);
        };
        const build = eval(`(${buildClient})`);
        const client = await build(endpoint, secret, "corr-live-1");
        const rendered = await client.render("Invoice totals", points);
        localAssert(rendered.ok === true, `render failed: ${JSON.stringify(rendered)}`);
        localAssert(/^vb-/.test(rendered.value.renderId), `render id not vendor-B: ${rendered.value.renderId}`);
        localAssert(rendered.value.svg.includes('data-vendor="b"'), "svg lacks vendor-B marker");
        localAssert(JSON.stringify(rendered.value.table) === JSON.stringify(points), "table mismatch");
        const selected = await client.select(rendered.value.renderId, 1);
        localAssert(
          selected.ok === true && selected.value.label === "Feb" && selected.value.value === 200,
          `select failed: ${JSON.stringify(selected)}`,
        );
        const released = await client.release(rendered.value.renderId);
        localAssert(released.ok === true, `release failed: ${JSON.stringify(released)}`);
        return "render/select/release from page context against vendor B";
      },
      { endpoint, secret: SECRET, points, buildClient },
    ),
  );

  await check("page client: select-after-release expires from the page", () =>
    page.evaluate(
      async ({ endpoint, secret, points, buildClient }) => {
        const build = eval(`(${buildClient})`);
        const client = await build(endpoint, secret, "corr-live-2");
        const rendered = await client.render("t", points);
        if (rendered.ok !== true) throw new Error(`render failed: ${JSON.stringify(rendered)}`);
        await client.release(rendered.value.renderId);
        const stale = await client.select(rendered.value.renderId, 0);
        if (stale.ok !== false || stale.failure.code !== "chart::expired") {
          throw new Error(`expected chart::expired, got ${JSON.stringify(stale)}`);
        }
        return "expiry honored across the page boundary";
      },
      { endpoint, secret: SECRET, points, buildClient },
    ),
  );

  await check("page client: invalid spec rejects client-side from the page", () =>
    page.evaluate(
      async ({ endpoint, secret, buildClient }) => {
        const build = eval(`(${buildClient})`);
        const client = await build(endpoint, secret, "corr-live-3");
        const got = await client.render("", []);
        if (got.ok !== false || got.failure.code !== "chart::invalid_spec") {
          throw new Error(`expected chart::invalid_spec, got ${JSON.stringify(got)}`);
        }
        return "invalid_spec without a send";
      },
      { endpoint, secret: SECRET, buildClient },
    ),
  );

  await check("page client: select roundtrip timing recorded under the trip wire", () =>
    page.evaluate(
      async ({ endpoint, secret, points, buildClient }) => {
        const build = eval(`(${buildClient})`);
        const client = await build(endpoint, secret, "corr-live-4");
        const budget = window.__d03_client.CHART_SELECT_ROUNDTRIP_BUDGET_MS;
        const rendered = await client.render("t", points);
        if (rendered.ok !== true) throw new Error(`render failed: ${JSON.stringify(rendered)}`);
        const samples = [];
        for (let index = 0; index < 25; index += 1) {
          const start = performance.now();
          const selected = await client.select(rendered.value.renderId, index % points.length);
          samples.push(performance.now() - start);
          if (selected.ok !== true) throw new Error(`select failed: ${JSON.stringify(selected)}`);
        }
        await client.release(rendered.value.renderId);
        const worst = samples.reduce((a, b) => (a > b ? a : b), 0);
        const sorted = [...samples].sort((a, b) => a - b);
        const p50 = sorted[Math.floor(sorted.length / 2)];
        if (!(worst < budget)) throw new Error(`trip wire tripped: worst ${worst}ms >= ${budget}ms`);
        return `p50 ${p50.toFixed(3)}ms, worst ${worst.toFixed(3)}ms of 25 against ${budget}ms`;
      },
      { endpoint, secret: SECRET, points, buildClient },
    ),
  );

  await check("no page errors escape any leg", async () => {
    assert.deepEqual(pageerrors, []);
    return "all legs clean";
  });

  writeFileSync(
    join(outdir, "report.json"),
    JSON.stringify({ browser: wanted, userAgent, secureContext, checks }, null, 2),
  );
} finally {
  await browser.close().catch(() => {});
  http.close();
}

const failed = checks.filter((entry) => !entry.passed);
if (failed.length > 0) {
  console.error(`live leg failures on ${wanted}: ${JSON.stringify(failed, null, 2)}`);
  process.exit(1);
}
console.log(`live legs pass on ${wanted}: ${checks.length}/${checks.length}`);
