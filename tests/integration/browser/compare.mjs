// C06 second-app evidence: drives the served paired invoice-compare
// application (compiler-built compare bundle plus the real invoice
// server) in one named Playwright browser through boot, edits,
// disposal, caret preservation and IME composition — and records
// DOM state, network bytes and console/page faults. Usage:
//   node compare.mjs <chromium|firefox|webkit> <base> <outdir> <script-url>
// Aborts every non-loopback request, so a passing run proves the app
// never needs a CDN, authored script, or foreign client runtime: the
// single served script must equal the report-selected paired URL, and
// window.htmx must stay undefined. Writes report.json and
// screenshot.png into outdir.
import { strict as assert } from "node:assert";
import { mkdirSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import { launchWanted } from "./firefox-remote.mjs";

const [wanted, base, outdir, scriptUrl] = process.argv.slice(2);
if (
  (wanted !== "chromium" && wanted !== "firefox" && wanted !== "webkit") ||
  !base ||
  !outdir ||
  !scriptUrl?.startsWith("/__can/assets/")
) {
  console.error("usage: node compare.mjs <chromium|firefox|webkit> <base> <outdir> <script-url>");
  process.exit(2);
}
mkdirSync(outdir, { recursive: true });

// Probed caret direction after fill/type with no explicit direction:
// Chromium and WebKit report none, Firefox reports forward.
const implicitDirection = wanted === "firefox" ? "forward" : "none";

const playwright = await import("playwright");
const { browser, remote } = await launchWanted(playwright, wanted);
let userAgent = "";
const checks = [];
const requests = [];
const bodies = [];
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

  const statusText = (tag) => page.locator(`#${tag}-status`).textContent();
  const totalsText = (tag) => page.locator(`#${tag}-totals`).textContent();
  // Values are set with a synthetic input event for fast
  // deterministic fills; the real-keystrokes leg proves the physical
  // keyboard path separately.
  const fill = async (selector, value) => {
    await page.locator(selector).evaluate(
      (node, text) => {
        node.focus();
        node.value = text;
        node.dispatchEvent(new Event("input", { bubbles: true }));
      },
      value
    );
    await page.waitForTimeout(300);
    assert.equal(await page.locator(selector).inputValue(), value);
  };
  const selectionOf = (selector) =>
    page.locator(selector).evaluate((node) => ({
      start: node.selectionStart,
      end: node.selectionEnd,
      direction: node.selectionDirection,
    }));

  const bootResponse = await page.goto(base + "/invoice-compare", { waitUntil: "load" });
  const csp = bootResponse?.headers()?.["content-security-policy"] ?? "";
  userAgent = await page.evaluate(() => navigator.userAgent);

  await check("boot-panels", async () => {
    // Empty status lines have no box, so visibility only applies
    // to the seeded inputs; statuses must be attached.
    await page.waitForSelector("#left-id\\:k1", { timeout: 8000 });
    await page.waitForSelector("#right-id\\:k1", { timeout: 8000 });
    await page.waitForSelector("#left-status", { state: "attached", timeout: 8000 });
    await page.waitForSelector("#right-status", { state: "attached", timeout: 8000 });
    assert.equal(await page.locator("#left-id\\:k1").inputValue(), "sku-1");
    assert.equal(await page.locator("#left-qty\\:k1").inputValue(), "2");
    assert.equal(await page.locator("#right-id\\:k2").inputValue(), "sku-2");
    assert.equal(await page.locator("#right-qty\\:k2").inputValue(), "1");
    assert.equal(await totalsText("left"), "2 rows, 3 units");
    assert.equal(await totalsText("right"), "2 rows, 3 units");
    assert.equal(await page.locator("#left-status").getAttribute("role"), "status");
    assert.equal(await page.locator("#left-status").getAttribute("aria-live"), "polite");
  });

  await check("client-identity", async () => {
    const scripts = await page
      .locator("script")
      .evaluateAll((nodes) => nodes.map((node) => node.getAttribute("src")));
    assert.deepEqual(scripts, [scriptUrl]);
    assert.equal(await page.evaluate(() => typeof window.htmx), "undefined");
    assert.ok(csp.includes("script-src 'self'"), `csp lacks script-src: ${csp}`);
    assert.ok(csp.includes("connect-src 'self'"), `csp lacks connect-src: ${csp}`);
    const bundle = await page.evaluate(async (src) => (await fetch(src)).text(), scriptUrl);
    assert.ok(!bundle.includes("cdn"), "bundle references a cdn");
    // The only URL strings are inert URL-parser sentinels (never
    // fetched): any other http(s) reference fails the leg.
    const urls = [...bundle.matchAll(/https?:\/\/[A-Za-z0-9.]+\/?/g)].map((hit) => hit[0]);
    const inert = new Set([
      "http://can.invalid",
      "http://foo.com",
      "http://foo.com/",
      "https://can.invalid",
      "https://can.invalid/",
    ]);
    for (const url of urls) assert.ok(inert.has(url), `bundle references ${url}`);
  });

  await check("generation-slot", async () => {
    const slot = await page.evaluate(
      () => document.querySelector("script[data-can-generation]")?.getAttribute("data-can-generation")
    );
    assert.ok(slot !== null && /^[0-9a-f]{64}$/.test(slot ?? ""), `slot malformed: ${slot}`);
    return `slot=${String(slot).slice(0, 12)}`;
  });

  await check("edit-left-keeps-right", async () => {
    const before = await statusText("right");
    await fill("#left-qty\\:k1", "5");
    assert.equal(await statusText("left"), "");
    assert.equal(await statusText("right"), before);
    assert.equal(await page.locator("#left-qty\\:k1").inputValue(), "5");
    assert.equal(await page.locator("#right-qty\\:k1").inputValue(), "2");
    assert.equal(await totalsText("left"), "2 rows, 6 units");
    assert.equal(await totalsText("right"), "2 rows, 3 units");
  });

  await check("review-blocks-on-invalid", async () => {
    await fill("#right-qty\\:k2", "many");
    assert.equal(await page.locator("#right-qty\\:k2").inputValue(), "many");
    await page.locator("#right-review").click();
    await page.waitForFunction(
      () => document.querySelector("#right-status")?.textContent?.includes("bad count"),
      null,
      { timeout: 5000 }
    );
    assert.equal(await statusText("right"), "line k2 count: bad count");
    assert.equal(await statusText("left"), "");
  });

  await check("real-keystrokes", async () => {
    await page.locator("#left-id\\:k2").click();
    await page.keyboard.press("End");
    await page.keyboard.type("!");
    assert.equal(await page.locator("#left-id\\:k2").inputValue(), "sku-2!");
    const caret = await selectionOf("#left-id\\:k2");
    assert.equal(caret.start, 6);
    assert.equal(caret.end, 6);
  });

  await check("caret-preserved-across-folds", async () => {
    // In-place status/totals updates never rewrite the live value,
    // so the caret stays where typing put it while folds land.
    await page.locator("#left-qty\\:k1").evaluate((node) => {
      node.focus();
      node.setSelectionRange(1, 1);
    });
    await page.keyboard.type("3");
    assert.equal(await page.locator("#left-qty\\:k1").inputValue(), "53");
    const caret = await selectionOf("#left-qty\\:k1");
    assert.deepEqual(caret, { start: 2, end: 2, direction: implicitDirection });
    assert.equal(await totalsText("left"), "2 rows, 54 units");
    assert.equal(await statusText("left"), "");
  });

  await check("composing-input-folds", async () => {
    // A mid-composition input folds like any other input and keeps
    // the caret: the app never rewrites the live value.
    await page.locator("#right-id\\:k1").evaluate((node) => {
      node.focus();
      node.value = "sku-9";
      node.setSelectionRange(2, 4, "backward");
      node.dispatchEvent(new InputEvent("input", { bubbles: true, isComposing: true, data: "9" }));
    });
    await page.waitForTimeout(300);
    assert.equal(await page.locator("#right-id\\:k1").inputValue(), "sku-9");
    const caret = await selectionOf("#right-id\\:k1");
    assert.deepEqual(caret, { start: 2, end: 4, direction: "backward" });
  });

  await check("mirror-folds-left-into-right", async () => {
    await page.locator("#left-review").click();
    await page.waitForFunction(
      () => document.querySelector("#left-status")?.textContent === "ready to review",
      null,
      { timeout: 5000 }
    );
    await page.locator("#mirror").click();
    await page.waitForFunction(
      () => document.querySelector("#right-status")?.textContent === "ready to review",
      null,
      { timeout: 5000 }
    );
    assert.equal(await statusText("left"), "ready to review");
    assert.equal(await statusText("right"), "ready to review");
  });

  await check("drop-left-right-works", async () => {
    await page.locator("#drop-left").click();
    await page.waitForFunction(() => document.querySelector("#left-status") === null, null, {
      timeout: 5000,
    });
    assert.equal(await page.locator("#right-id\\:k1").inputValue(), "sku-9");
    await fill("#right-qty\\:k1", "7");
    assert.equal(await page.locator("#right-qty\\:k1").inputValue(), "7");
    const status = await statusText("right");
    assert.ok(!status.includes("bad quantity"), `right status: ${status}`);
  });

  await check("mirror-after-drop-silent", async () => {
    const before = await statusText("right");
    const mirrorVisible = await page.locator("#mirror").count();
    if (mirrorVisible > 0) {
      await page.locator("#mirror").click();
      await page.waitForTimeout(300);
    }
    assert.equal(await statusText("right"), before);
    assert.equal(await page.locator("#left-status").count(), 0);
  });

  await check("network-ledger-clean", async () => {
    // Seeds only: the compare app makes no API calls.
    const api = requests.filter((entry) => entry.url.includes("/api/"));
    assert.deepEqual(api, []);
    assert.deepEqual(aborted, []);
    assert.deepEqual(bodies, []);
    const failed = requests.filter((entry) => entry.status === -1 || entry.status >= 400);
    assert.deepEqual(failed, []);
  });

  await check("no-page-faults", async () => {
    assert.deepEqual(pageerrors, []);
    // The served CSP's navigate-to directive warns once per load;
    // anything else fails. Pinned exactly like the grid harness.
    const pinned = new Set(["Unrecognized Content-Security-Policy directive 'navigate-to'."]);
    for (const text of consoleErrors) {
      assert.ok(pinned.has(text.trim()), `non-resource console error: ${text}`);
    }
  });

  await page.screenshot({ path: join(outdir, "screenshot.png") });
  const failed = checks.filter((entry) => !entry.passed);
  const report = {
    harness: "compare.mjs",
    browser: wanted,
    version: browser.version(),
    userAgent,
    checks,
    requests,
    aborted,
    pageerrors,
    consoleErrors,
    passed: failed.length === 0,
  };
  writeFileSync(join(outdir, "report.json"), JSON.stringify(report, null, 2) + "\n");
  if (!remote) await browser.close();
  if (!report.passed) {
    console.error(`compare ${wanted}: ${failed.length} failing checks`);
    for (const entry of failed) console.error(`- ${entry.name}: ${entry.detail}`);
    process.exit(1);
  }
  process.exit(0);
} catch (error) {
  console.error(`compare ${wanted}: ${String(error?.stack ?? error)}`);
  try {
    if (!remote) await browser.close();
  } catch {}
  process.exit(1);
}
