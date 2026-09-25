// Pinned browser evidence for the I42 dashboard admission application:
// drives the staged live /dashboard page with upstream HTMX only. Aborts
// every non-localhost request, so a passing run proves the page never
// needs a CDN, authored script, or client runtime.
// Usage: node dashboard.mjs <base> <outdir>
import { strict as assert } from "node:assert";
import { mkdirSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import { chromium } from "playwright";

const [base, outdir] = process.argv.slice(2);
if (!base || !outdir) {
  console.error("usage: node dashboard.mjs <base> <outdir>");
  process.exit(2);
}
mkdirSync(outdir, { recursive: true });

const browser = await chromium.launch();
const checks = [];
const requests = [];
const aborted = [];
const check = (name, fn) =>
  Promise.resolve()
    .then(fn)
    .then((detail = "") => checks.push({ name, passed: true, detail }))
    .catch((error) => checks.push({ name, passed: false, detail: String(error?.message ?? error) }));

try {
  const context = await browser.newContext();
  await context.route("**/*", (route) => {
    const url = new URL(route.request().url());
    if (url.hostname === "127.0.0.1" || url.hostname === "localhost") return route.continue();
    aborted.push(route.request().url());
    return route.abort("blockedbyclient");
  });
  const page = await context.newPage();
  let dataPolls = 0;
  page.on("requestfinished", async (request) => {
    const response = await request.response().catch(() => undefined);
    requests.push({ method: request.method(), url: request.url(), status: response?.status() ?? 0 });
    if (request.url().endsWith("/dashboard/data") && response?.status() === 200) dataPolls++;
  });
  await page.goto(base + "/dashboard", { waitUntil: "load" });

  await check("htmx-loaded", async () => {
    assert.equal(await page.evaluate(() => typeof window.htmx), "object");
  });
  await check("pinned-scripts", async () => {
    const scripts = await page.locator("script").evaluateAll((nodes) =>
      nodes.map((node) => ({ src: node.getAttribute("src"), integrity: node.getAttribute("integrity") }))
    );
    assert.equal(scripts.length, 2);
    assert.equal(scripts[0].src, "/__can/assets/htmx-4.0.0.min.js");
    assert.equal(scripts[0].integrity, "sha384-BvJpBiO8Kh31EqtJe5DRIeWrHWnCGkwytKs9NKFi86Hhw96dEqdEMzZDeK9iEGTc");
    assert.equal(scripts[1].src, "/__can/assets/htmx-guard.js");
    assert.equal(scripts[1].integrity, "sha384-mr/IRfJgLjok38ftBi21o/T8c9cnFZrvEKtiwVjIOFAlo3Z7h1rGMYsWvebDJ8kG");
  });
  await check("dashboard-poll-swap", async () => {
    await page.waitForFunction(() => document.querySelector("#dashboard")?.textContent !== "dashboard-before", undefined, {
      timeout: 15000,
    });
    assert.ok((await page.locator("#dashboard").textContent())?.includes("3 accounts 3 recent"));
  });
  await check("dashboard-repeated-polls", async () => {
    const start = Date.now();
    while (dataPolls < 2 && Date.now() - start < 20000) {
      await page.waitForTimeout(200);
    }
    assert.ok(dataPolls >= 2, `expected repeated polls, saw ${dataPolls}`);
  });
  await check("no-external-requests", async () => {
    assert.equal(aborted.length, 0);
    for (const entry of requests) {
      assert.ok(new URL(entry.url).hostname === "127.0.0.1", entry.url);
    }
    assert.ok(requests.some((entry) => entry.url.endsWith("/__can/assets/htmx-4.0.0.min.js") && entry.status === 200));
  });

  await page.screenshot({ path: join(outdir, "screenshot.png"), fullPage: true });
  await context.close();
} finally {
  await browser.close();
}

const passed = checks.every((entry) => entry.passed);
writeFileSync(
  join(outdir, "report.json"),
  JSON.stringify({ browser: "chromium", version: browser.version(), base, passed, checks, aborted, requests }, null, 2) + "\n"
);
for (const entry of checks) console.log(`${entry.passed ? "PASS" : "FAIL"} ${entry.name}${entry.detail ? ` ${entry.detail}` : ""}`);
if (!passed) process.exit(1);
console.log("browser dashboard evidence passed");
