// Run against the current staged account-search app and a disposable seeded DB.
// Usage: CAN_PROBE_DB_URL=postgresql://... node probe.mjs BASE OUTFILE
import { strict as assert } from "node:assert";
import { spawnSync } from "node:child_process";
import { writeFileSync } from "node:fs";
import { chromium } from "../../tests/integration/browser/node_modules/playwright/index.mjs";

const [base, outfile] = process.argv.slice(2);
const database = process.env.CAN_PROBE_DB_URL;
if (!base?.startsWith("http://127.0.0.1:") || !outfile || !database?.includes("127.0.0.1:18562")) {
  throw new Error("expected loopback base, output file, and disposable probe database");
}
const sql = (statement) => {
  const result = spawnSync("psql", [database, "-v", "ON_ERROR_STOP=1", "-c", statement], { encoding: "utf8" });
  if (result.status !== 0) throw new Error(`psql failed: ${result.stderr}`);
};

const browser = await chromium.launch();
let tableRenamed = false;
const observations = [];
try {
  const page = await browser.newPage();
  await page.route("**/*", (route) => {
    const url = new URL(route.request().url());
    if (url.hostname !== "127.0.0.1") return route.abort("blockedbyclient");
    return route.continue();
  });
  await page.goto(`${base}/accounts`, { waitUntil: "load" });
  const meta = await page.locator('meta[name="htmx-config"]').getAttribute("content");
  const config = JSON.parse(meta);
  assert.equal(await page.evaluate(() => typeof window.htmx), "object");
  assert.ok(config.noSwap.includes(503));
  assert.ok(!config.noSwap.includes(422));
  const runSearch = async (label, value, expectedStatus) => {
    const before = await page.locator("#account_results").innerHTML();
    await page.locator("#account_query").fill(value);
    const responsePromise = page.waitForResponse((r) => r.url().includes("/accounts/search?") && r.request().method() === "GET");
    await page.locator("#account_query").press("Tab");
    const response = await responsePromise;
    assert.equal(response.status(), expectedStatus);
    const body = await response.text();
    await page.waitForTimeout(250);
    const after = await page.locator("#account_results").innerHTML();
    observations.push({ label, status: response.status(), body, before, after, url: response.url(),
      contentType: response.headers()["content-type"] });
    return { body, before, after };
  };
  const ok = await runSearch("success", "Ann", 200);
  assert.match(ok.body, /<li>Ann<\/li>/);
  assert.match(ok.after, /<li>Ann<\/li>/);
  const invalid = await runSearch("invalid", "", 422);
  assert.match(invalid.body, /Enter a search term\./);
  assert.match(invalid.after, /Enter a search term\./);
  sql("ALTER TABLE can_i42_accounts RENAME TO can_i42_accounts_unavailable_probe");
  tableRenamed = true;
  const unavailable = await runSearch("unavailable", "Ann", 503);
  assert.match(unavailable.body, /Temporarily unavailable\./);
  assert.equal(unavailable.after, unavailable.before);
  assert.ok(!unavailable.after.includes("Temporarily unavailable."));
  await page.close();
  writeFileSync(outfile, JSON.stringify({ passed: true, browser: "chromium", browserVersion: browser.version(),
    playwrightVersion: "1.55.1", config: { mode: config.mode, noSwapIncludes503: config.noSwap.includes(503),
      noSwapIncludes422: config.noSwap.includes(422) }, observations }, null, 2) + "\n");
} finally {
  if (tableRenamed) sql("ALTER TABLE can_i42_accounts_unavailable_probe RENAME TO can_i42_accounts");
  await browser.close();
}
