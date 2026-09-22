// Pinned browser evidence for the I42 form-validation admission application:
// drives the staged /signup page with upstream HTMX only. Aborts every
// non-localhost request, so a passing run proves the page never needs a
// CDN, authored script, or client runtime.
// Usage: node forms.mjs <base> <outdir>
import { strict as assert } from "node:assert";
import { mkdirSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import { chromium } from "playwright";

const [base, outdir] = process.argv.slice(2);
if (!base || !outdir) {
  console.error("usage: node forms.mjs <base> <outdir>");
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
  page.on("requestfinished", async (request) => {
    const response = await request.response().catch(() => undefined);
    requests.push({ method: request.method(), url: request.url(), status: response?.status() ?? 0 });
  });
  const document = await page.goto(base + "/signup", { waitUntil: "load" });
  const headers = document?.headers() ?? {};

  await check("htmx-loaded", async () => {
    assert.equal(await page.evaluate(() => typeof window.htmx), "object");
  });
  await check("single-pinned-script", async () => {
    const scripts = await page.locator("script").evaluateAll((nodes) =>
      nodes.map((node) => ({ src: node.getAttribute("src"), integrity: node.getAttribute("integrity") }))
    );
    assert.equal(scripts.length, 1);
    assert.equal(scripts[0].src, "/__can/assets/htmx-4.0.0.min.js");
    assert.equal(scripts[0].integrity, "sha384-BvJpBiO8Kh31EqtJe5DRIeWrHWnCGkwytKs9NKFi86Hhw96dEqdEMzZDeK9iEGTc");
  });
  await check("no-inline-handlers", async () => {
    const handlers = await page.evaluate(
      () => document.querySelectorAll("[onclick],[onchange],[onsubmit],[onload]").length
    );
    assert.equal(handlers, 0);
  });
  await check("csp-header", async () => {
    const policy = headers["content-security-policy"] ?? "";
    assert.ok(policy.includes("script-src 'self'") && !policy.includes("unsafe-eval"));
    assert.ok(policy.includes("navigate-to 'self'") && policy.includes("form-action 'self'"));
  });
  await check("form-422-swap", async () => {
    await page.fill("#signup_name", "");
    const answered = page.waitForResponse(
      (r) => r.url().endsWith("/signup/validate") && r.request().method() === "POST"
    );
    await page.locator('#signup_form button[type="submit"]').click();
    assert.equal((await answered).status(), 422);
    await page.waitForFunction(() =>
      document.querySelector("#signup_results")?.textContent?.includes("Name is required.")
    );
  });
  await check("form-200-swap", async () => {
    await page.fill("#signup_name", "Ann");
    await page.locator('input[name="tags"]').fill("a");
    const answered = page.waitForResponse(
      (r) => r.url().endsWith("/signup/validate") && r.request().method() === "POST"
    );
    await page.locator('#signup_form button[type="submit"]').click();
    assert.equal((await answered).status(), 200);
    await page.waitForFunction(() =>
      document.querySelector("#signup_results")?.textContent?.includes("Ann tags 1")
    );
  });
  await check("hostile-escaped", async () => {
    await page.fill("#signup_name", "<script>window.__evil=1</script>");
    const answered = page.waitForResponse(
      (r) => r.url().endsWith("/signup/validate") && r.request().method() === "POST"
    );
    await page.locator('#signup_form button[type="submit"]').click();
    assert.equal((await answered).status(), 200);
    await page.waitForFunction(() =>
      document.querySelector("#signup_results")?.textContent?.includes("window.__evil")
    );
    assert.equal(await page.locator("#signup_results script").count(), 0);
    assert.equal(await page.evaluate(() => window.__evil), undefined);
    assert.ok((await page.locator("#signup_results").innerHTML())?.includes("&lt;script&gt;"));
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
console.log("browser forms evidence passed");
