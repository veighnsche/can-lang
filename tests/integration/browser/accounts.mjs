// Pinned browser evidence for the I42 account-search admission application:
// drives the staged live /accounts page with upstream HTMX only. Aborts
// every non-localhost request, so a passing run proves the page never
// needs a CDN, authored script, or client runtime.
// Usage: node accounts.mjs <base> <outdir>
import { strict as assert } from "node:assert";
import { mkdirSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import { chromium } from "playwright";

const [base, outdir] = process.argv.slice(2);
if (!base || !outdir) {
  console.error("usage: node accounts.mjs <base> <outdir>");
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
  let summaryPolls = 0;
  page.on("requestfinished", async (request) => {
    const response = await request.response().catch(() => undefined);
    requests.push({ method: request.method(), url: request.url(), status: response?.status() ?? 0 });
    if (request.url().endsWith("/dashboard/summary") && response?.status() === 200) summaryPolls++;
  });
  await page.goto(base + "/accounts", { waitUntil: "load" });

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
  const search = async (value) => {
    await page.locator("#account_query").fill(value);
    const answered = page.waitForResponse(
      (r) => r.url().includes("/accounts/search") && r.request().method() === "GET"
    );
    await page.locator("#account_query").press("Tab");
    return answered;
  };
  await check("search-200-swap", async () => {
    const answered = await search("Ann");
    assert.equal(answered.status(), 200);
    await page.waitForFunction(() => document.querySelector("#account_results")?.textContent?.includes("Ann"));
    assert.ok((await page.locator("#account_results li").count()) >= 1);
  });
  await check("search-422-quiet", async () => {
    // The raw GET search route is not action-bound (GET actions answer
    // JSON, so no HTML swap policy exists for it), so its 422 carries no
    // hx-status exception and stays quiet under the compiler-owned noSwap
    // policy; the previous 200 results stand untouched.
    const answered = await search("");
    assert.equal(answered.status(), 422);
    await page.waitForTimeout(300);
    const results = await page.locator("#account_results").textContent();
    assert.ok(!results?.includes("Enter a search term."));
    assert.ok((await page.locator("#account_results li").count()) >= 1);
  });
  await check("search-hostile-escaped", async () => {
    const answered = await search("<em>");
    assert.equal(answered.status(), 200);
    await page.waitForFunction(() =>
      document.querySelector("#account_results")?.textContent?.includes("<em>Hax</em>")
    );
    assert.equal(await page.locator("#account_results em").count(), 0);
    assert.ok((await page.locator("#account_results").innerHTML())?.includes("&lt;em&gt;"));
  });
  await check("validate-422-swap", async () => {
    await page.fill("#account_name", "");
    const answered = page.waitForResponse(
      (r) => r.url().endsWith("/accounts/validate") && r.request().method() === "POST"
    );
    await page.locator('#account_form button[type="submit"]').click();
    assert.equal((await answered).status(), 422);
    await page.waitForFunction(() =>
      document.querySelector("#validation_results")?.textContent?.includes("Name is required.")
    );
  });
  await check("validate-200-swap", async () => {
    await page.fill("#account_name", "Ann");
    const answered = page.waitForResponse(
      (r) => r.url().endsWith("/accounts/validate") && r.request().method() === "POST"
    );
    await page.locator('#account_form button[type="submit"]').click();
    assert.equal((await answered).status(), 200);
    await page.waitForFunction(() =>
      document.querySelector("#validation_results")?.textContent?.includes("Ann")
    );
  });
  await check("dashboard-poll", async () => {
    await page.waitForFunction(() => document.querySelector("#dashboard")?.textContent !== "dashboard-before", undefined, {
      timeout: 15000,
    });
    assert.match((await page.locator("#dashboard").textContent()) ?? "", /^\d+$/);
    assert.ok(summaryPolls >= 1, `expected summary polls, saw ${summaryPolls}`);
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
console.log("browser accounts evidence passed");
