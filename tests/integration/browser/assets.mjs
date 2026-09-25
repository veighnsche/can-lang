// Pinned browser evidence for I34: drives the staged Can asset page with
// upstream HTMX only. Aborts every non-localhost request, so a passing run
// proves the page never needs a CDN, authored script, or client runtime.
// Usage: node assets.mjs <base> <outdir> [engine]
// UP23: the engine is an optional trailing parameter (default
// chromium) so the same twelve pinned checks can run under any
// named browser; the two-argument form keeps its legacy meaning.
import { strict as assert } from "node:assert";
import { mkdirSync, writeFileSync } from "node:fs";
import { join } from "node:path";

const [base, outdir, engineArg] = process.argv.slice(2);
if (!base || !outdir) {
  console.error("usage: node assets.mjs <base> <outdir> [engine]");
  process.exit(2);
}
const engine = engineArg ?? "chromium";
if (engine !== "chromium" && engine !== "webkit") {
  console.error(`unsupported engine ${engine}`);
  process.exit(2);
}
mkdirSync(outdir, { recursive: true });

const playwright = await import("playwright");
const browser = await playwright[engine].launch({ timeout: 120000 });
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
  let dashboardPolls = 0;
  page.on("requestfinished", async (request) => {
    const response = await request.response().catch(() => undefined);
    requests.push({ method: request.method(), url: request.url(), status: response?.status() ?? 0 });
    if (request.url().endsWith("/dashboard") && response?.status() === 200) dashboardPolls++;
  });
  const awaitPolls = async (count, timeoutMs) => {
    const start = Date.now();
    while (dashboardPolls < count) {
      if (Date.now() - start > timeoutMs) return dashboardPolls;
      await page.waitForTimeout(200);
    }
    return dashboardPolls;
  };
  const document = await page.goto(base + "/", { waitUntil: "load" });
  const headers = document?.headers() ?? {};

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
    // Guard digest mirrors runtimeHead in runtime/platform/html.ts; re-pin together.
    assert.equal(scripts[1].integrity, "sha384-mr/IRfJgLjok38ftBi21o/T8c9cnFZrvEKtiwVjIOFAlo3Z7h1rGMYsWvebDJ8kG");
  });
  await check("htmx-config", async () => {
    const content = await page.locator('meta[name="htmx-config"]').getAttribute("content");
    const config = JSON.parse(content ?? "");
    assert.equal(config.mode, "same-origin");
    assert.deepEqual(config.noSwap, [204, 304, "4xx", "5xx"]);
  });
  await check("stylesheet-local", async () => {
    const href = await page.locator('link[rel="stylesheet"]').getAttribute("href");
    assert.match(href ?? "", /^\/__can\/project\/[0-9a-f]{64}\/site\.css$/);
    const hit = requests.find((entry) => entry.url.endsWith(href) && entry.status === 200);
    assert.ok(hit, "stylesheet fetched with 200");
  });
  await check("csp-header", async () => {
    const policy = headers["content-security-policy"] ?? "";
    assert.ok(policy.includes("script-src 'self'") && !policy.includes("unsafe-eval"));
    assert.ok(policy.includes("navigate-to 'self'") && policy.includes("form-action 'self'"));
  });
  await check("form-422-quiet", async () => {
    // Plain (non-action) forms carry no hx-status exception, so the 422 stays
    // quiet under the compiler-owned noSwap policy. Action-bound 422 feedback
    // is covered by the pinned guard tests; Can-authored exception producers
    // land with UP19's mounted form actions.
    await page.fill("#account_name", "");
    const answered = page.waitForResponse((r) => r.url().endsWith("/validate") && r.request().method() === "POST");
    await page.locator('#account_form button[type="submit"]').click();
    assert.equal((await answered).status(), 422);
    await page.waitForTimeout(300);
    assert.equal(await page.locator("#account_results").textContent(), "waiting");
  });
  await check("form-200-swap", async () => {
    await page.fill("#account_name", "Ann");
    const answered = page.waitForResponse((r) => r.url().endsWith("/validate") && r.request().method() === "POST");
    await page.locator('#account_form button[type="submit"]').click();
    assert.equal((await answered).status(), 200);
    await page.waitForFunction(() => document.querySelector("#account_results")?.textContent?.includes("Ann"));
  });
  await check("hostile-escaped", async () => {
    await page.fill("#account_name", "<script>window.__evil=1</script>");
    const answered = page.waitForResponse((r) => r.url().endsWith("/validate") && r.request().method() === "POST");
    await page.locator('#account_form button[type="submit"]').click();
    assert.equal((await answered).status(), 200);
    await page.waitForFunction(() => document.querySelector("#account_results")?.textContent?.includes("window.__evil"));
    assert.equal(await page.locator("#account_results script").count(), 0);
    assert.equal(await page.evaluate(() => window.__evil), undefined);
    assert.equal(await page.evaluate(() => window.__pwned), undefined);
    assert.ok((await page.locator("#hostile").textContent())?.includes("window.__pwned"));
  });
  await check("204-no-swap", async () => {
    const answered = page.waitForResponse((r) => r.url().endsWith("/quiet"));
    await page.getByRole("button", { name: "Quiet" }).click();
    assert.equal((await answered).status(), 204);
    await page.waitForTimeout(300);
    assert.equal(await page.locator("#quiet_region").textContent(), "quiet-before");
  });
  await check("500-no-swap", async () => {
    const answered = page.waitForResponse((r) => r.url().endsWith("/boom"));
    await page.getByRole("button", { name: "Boom" }).click();
    assert.equal((await answered).status(), 500);
    await page.waitForTimeout(300);
    assert.equal(await page.locator("#boom_region").textContent(), "boom-before");
  });
  await check("dashboard-polling", async () => {
    await page.waitForFunction(() => document.querySelector("#dashboard")?.textContent !== "dashboard-before", undefined, {
      timeout: 15000,
    });
    assert.match((await page.locator("#dashboard").textContent()) ?? "", /^\d+$/);
    const polls = await awaitPolls(3, 15000);
    assert.ok(polls >= 3, `expected repeated polls, saw ${polls}`);
  });
  await check("no-external-requests", async () => {
    assert.equal(aborted.length, 0);
    for (const entry of requests) {
      assert.ok(new URL(entry.url).hostname === "127.0.0.1", entry.url);
    }
    assert.ok(requests.some((entry) => entry.url.endsWith("/__can/assets/htmx-4.0.0.min.js") && entry.status === 200));
    assert.ok(requests.some((entry) => entry.url.endsWith("/__can/assets/htmx-guard.js") && entry.status === 200));
  });

  await page.screenshot({ path: join(outdir, "screenshot.png"), fullPage: true });
  await context.close();
} finally {
  await browser.close();
}

const passed = checks.every((entry) => entry.passed);
writeFileSync(
  join(outdir, "report.json"),
  JSON.stringify({ browser: engine, version: browser.version(), base, passed, checks, aborted, requests }, null, 2) + "\n"
);
for (const entry of checks) console.log(`${entry.passed ? "PASS" : "FAIL"} ${entry.name}${entry.detail ? ` ${entry.detail}` : ""}`);
if (!passed) process.exit(1);
console.log(`browser asset evidence passed on ${engine}`);
