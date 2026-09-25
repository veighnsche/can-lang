// UP23 served-startup evidence for the minimal generated browser
// program: loads the application-served grid shell (paired with the
// empty browser build) in one named Playwright browser and proves
// the degenerate program starts cleanly from served bytes — page,
// exact paired script, source map and diagnostic table — with no
// page faults and the shell untouched. Usage:
//   node empty.mjs <chromium|webkit> <base> <outdir> <script-url> <map-url> <table-url>
// Aborts every non-loopback request. Writes report.json and
// screenshot.png into outdir.
import { strict as assert } from "node:assert";
import { mkdirSync, writeFileSync } from "node:fs";
import { join } from "node:path";

const [wanted, base, outdir, scriptUrl, mapUrl, tableUrl] = process.argv.slice(2);
if (
  (wanted !== "chromium" && wanted !== "webkit") ||
  !base ||
  !outdir ||
  !scriptUrl?.startsWith("/__can/assets/") ||
  !mapUrl?.startsWith("/__can/assets/") ||
  !tableUrl?.startsWith("/__can/assets/")
) {
  console.error("usage: node empty.mjs <chromium|webkit> <base> <outdir> <script-url> <map-url> <table-url>");
  process.exit(2);
}
mkdirSync(outdir, { recursive: true });

const playwright = await import("playwright");
const browser = await playwright[wanted].launch({ timeout: 120000 });
const checks = [];
const requests = [];
const aborted = [];
const pageerrors = [];
const consoleErrors = [];
const check = (name, fn) =>
  Promise.resolve()
    .then(fn)
    .then((detail = "") => checks.push({ name, passed: true, detail }))
    .catch((error) => checks.push({ name, passed: false, detail: String(error?.message ?? error) }));

try {
  const context = await browser.newContext();
  await context.route("**/*", (route) => {
    const url = new URL(route.request().url());
    if (url.protocol !== "http:" && url.protocol !== "https:") return route.continue();
    if (url.hostname === "127.0.0.1" || url.hostname === "localhost") return route.continue();
    aborted.push(route.request().url());
    return route.abort("blockedbyclient");
  });
  const page = await context.newPage();
  page.on("pageerror", (error) => pageerrors.push(String(error?.message ?? error)));
  page.on("console", (message) => {
    if (message.type() === "error") consoleErrors.push(message.text());
  });
  page.on("requestfinished", async (request) => {
    const response = await request.response().catch(() => undefined);
    requests.push({ method: request.method(), url: request.url(), status: response?.status() ?? 0 });
  });
  page.on("requestfailed", (request) => {
    requests.push({ method: request.method(), url: request.url(), status: -1 });
  });

  const response = await page.goto(base + "/invoice-grid?tenant=1&invoice=7", { waitUntil: "load" });
  const csp = response?.headers()?.["content-security-policy"] ?? "";
  await page.waitForTimeout(1500);

  await check("served-shell", async () => {
    assert.equal(response?.status(), 200);
    assert.ok(csp.includes("script-src 'self'"), `csp lacks script-src: ${csp}`);
    assert.equal(await page.locator("#invoice-grid p").textContent(), "Loading invoice grid…");
    const scripts = await page.locator("script").evaluateAll((nodes) => nodes.map((n) => n.getAttribute("src")));
    assert.deepEqual(scripts, [scriptUrl]);
    return "shell intact, single paired script";
  });

  await check("paired-startup-clean", async () => {
    const bundle = await page.evaluate(async (src) => (await fetch(src)).text(), scriptUrl);
    assert.ok(bundle.includes("$canBrowserMain"), "bundle lacks the Can browser entry");
    assert.equal(pageerrors.length, 0, pageerrors.join("; "));
    assert.equal(await page.locator("#invoice-grid p").count(), 1, "empty main must not touch the DOM");
    return "module executed, no faults, DOM untouched";
  });

  await check("served-map-table", async () => {
    const map = await page.evaluate(async (src) => {
      const res = await fetch(src);
      return { status: res.status, json: await res.json() };
    }, mapUrl);
    assert.equal(map.status, 200);
    assert.equal(map.json.version, 3);
    assert.ok(Array.isArray(map.json.sources) && map.json.sources.length > 0, "map lacks sources");
    const table = await page.evaluate(async (src) => {
      const res = await fetch(src);
      return { status: res.status, json: await res.json() };
    }, tableUrl);
    assert.equal(table.status, 200);
    assert.equal(table.json.kind, "can.diagnostic-table");
    assert.equal(table.json.schemaVersion, 1);
    return `map ${map.json.sources.length} sources, sealed table`;
  });

  await check("network-ledger-clean", async () => {
    assert.equal(aborted.length, 0, `non-loopback requests: ${aborted.join(", ")}`);
    for (const entry of requests) {
      const host = new URL(entry.url).hostname;
      assert.ok(host === "127.0.0.1" || host === "localhost", `left loopback: ${entry.url}`);
    }
    assert.equal(pageerrors.length, 0, pageerrors.join("; "));
    const pinned = new Set([
      "Unrecognized Content-Security-Policy directive 'navigate-to'.",
      "Refused to apply a stylesheet because its hash, its nonce, or 'unsafe-inline' does not appear in the style-src directive of the Content Security Policy.",
    ]);
    const rest = consoleErrors.filter((text) => !pinned.has(text.trim()));
    assert.equal(rest.length, 0, rest.join("; "));
    return `${requests.length} loopback requests`;
  });

  await page.screenshot({ path: join(outdir, "screenshot.png") });
  const userAgent = await page.evaluate(() => navigator.userAgent);
  await context.close();

  const passed = checks.every((entry) => entry.passed);
  writeFileSync(
    join(outdir, "report.json"),
    JSON.stringify(
      { browser: wanted, version: browser.version(), userAgent, base, passed, checks, requests, aborted, pageerrors, consoleErrors },
      null,
      2
    ) + "\n"
  );
  for (const entry of checks) console.log(`${entry.passed ? "PASS" : "FAIL"} ${entry.name}${entry.detail ? ` ${entry.detail}` : ""}`);
  if (!passed) process.exit(1);
  console.log(`browser empty-app evidence passed on ${wanted}`);
} finally {
  await browser.close();
}
