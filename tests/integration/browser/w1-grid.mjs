// C06 W1 grid evidence: drives the served paired invoice grid in one
// named Playwright browser through generation slot/header proof,
// truthful denied/not-found reads, IME/caret preservation, and the
// blocking refresh prompt on generation refusal — and records DOM
// state, the API ledger and console/page faults. Usage:
//   node w1-grid.mjs <chromium|firefox|webkit> <base> <outdir> <script-url> <generation>
// Aborts every non-loopback request. Writes report.json and
// screenshot.png into outdir.
import { strict as assert } from "node:assert";
import { mkdirSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import { launchWanted } from "./firefox-remote.mjs";

const [wanted, base, outdir, scriptUrl, generation] = process.argv.slice(2);
if (
  (wanted !== "chromium" && wanted !== "firefox" && wanted !== "webkit") ||
  !base ||
  !outdir ||
  !scriptUrl?.startsWith("/__can/assets/") ||
  !/^[0-9a-f]{64}$/.test(generation ?? "")
) {
  console.error("usage: node w1-grid.mjs <chromium|firefox|webkit> <base> <outdir> <script-url> <generation>");
  process.exit(2);
}
mkdirSync(outdir, { recursive: true });

// Probed caret direction after fill/type with no explicit direction:
// Chromium and WebKit report none, Firefox reports forward.
const implicitDirection = wanted === "firefox" ? "forward" : "none";
const OTHER = "f".repeat(64);
const mismatchBody = JSON.stringify({
  schemaVersion: 1,
  kind: "can.generation-mismatch",
  serverGeneration: OTHER,
});

const playwright = await import("playwright");
const { browser, remote } = await launchWanted(playwright, wanted);
let userAgent = "";
const checks = [];
const requests = [];
const ledger = [];
const aborted = [];
const pageerrors = [];
const consoleErrors = [];
const headers = [];
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
  page.on("response", async (response) => {
    const url = response.url();
    if (!url.includes("/api/")) return;
    const entry = { method: "", url, status: response.status(), requestBody: "", responseBody: "" };
    try {
      const request = response.request();
      entry.method = request.method();
      entry.requestBody = request.postData() ?? "";
      entry.responseBody = await response.text();
    } catch (error) {
      entry.responseBody = `unreadable: ${String(error?.message ?? error).slice(0, 120)}`;
    }
    ledger.push(entry);
  });

  // One API route with a mutable mode: capture records the header
  // and continues; save409/load409 answer the exact refusal.
  let mode = "capture";
  const posts = [];
  await page.route("**/api/**", (route) => {
    const request = route.request();
    headers.push({
      method: request.method(),
      url: request.url(),
      generation: request.headers()["can-generation"] ?? null,
    });
    if (request.method() === "POST" && request.url().includes("/api/tenants/")) {
      posts.push(request.url());
    }
    if (mode === "save409" && request.method() === "POST") {
      return route.fulfill({
        status: 409,
        contentType: "application/json",
        body: mismatchBody,
      });
    }
    if (mode === "load409" && request.method() === "GET") {
      return route.fulfill({
        status: 409,
        contentType: "application/json",
        body: mismatchBody,
      });
    }
    return route.continue();
  });

  const statusText = () => page.locator("#status").textContent();
  const selectionOf = (selector) =>
    page.locator(selector).evaluate((node) => ({
      start: node.selectionStart,
      end: node.selectionEnd,
      direction: node.selectionDirection,
    }));

  await context.addCookies([{ name: "session", value: "tok-alice", url: base }]);
  const bootResponse = await page.goto(base + "/invoice-grid?tenant=1&invoice=7", { waitUntil: "load" });
  const csp = bootResponse?.headers()?.["content-security-policy"] ?? "";
  userAgent = await page.evaluate(() => navigator.userAgent);

  await check("boot-loads", async () => {
    await page.waitForFunction(() => document.querySelector("#status")?.textContent?.includes("loaded revision 1"));
    assert.equal(await page.locator("#grid h1").textContent(), "Tenant 1 invoice 7 revision 1");
    assert.equal(await statusText(), "loaded revision 1");
    assert.equal(await page.locator("#totals").textContent(), "2 lines, 44.98 total");
  });

  await check("client-identity", async () => {
    const scripts = await page
      .locator("script")
      .evaluateAll((nodes) => nodes.map((node) => node.getAttribute("src")));
    assert.deepEqual(scripts, [scriptUrl]);
    assert.equal(await page.evaluate(() => typeof window.htmx), "undefined");
    assert.ok(csp.includes("script-src 'self'"), `csp lacks script-src: ${csp}`);
    assert.ok(csp.includes("connect-src 'self'"), `csp lacks connect-src: ${csp}`);
  });

  await check("generation-slot", async () => {
    const slot = await page.evaluate(
      () => document.querySelector("script[data-can-generation]")?.getAttribute("data-can-generation")
    );
    assert.equal(slot, generation);
  });

  await check("header-on-actions", async () => {
    const api = headers.filter((entry) => entry.url.includes("/api/"));
    assert.ok(api.length >= 1, `no api calls observed: ${JSON.stringify(headers)}`);
    for (const entry of api) assert.equal(entry.generation, generation);
    return `${api.length} calls`;
  });

  await check("missing-id-denied", async () => {
    await page.goto(base + "/invoice-grid?tenant=1&invoice=9", { waitUntil: "load" });
    await page.waitForFunction(() =>
      document.querySelector("#status")?.textContent?.includes("access denied; draft kept")
    );
    assert.equal(await statusText(), "access denied; draft kept");
    assert.equal(await page.locator("#grid h1").textContent(), "Tenant 1 invoice 9 revision ");
    const html = await page.locator("#grid").innerHTML();
    assert.ok(!html.includes("sku-1"), `denied page leaks rows: ${html.slice(0, 200)}`);
    const denied = ledger.filter((entry) => entry.url.includes("/invoices/9") && entry.method === "GET");
    assert.equal(denied.length, 1);
    assert.equal(denied[0].status, 403);
    assert.ok(denied[0].responseBody.includes("grid_load_forbidden"), denied[0].responseBody);
    assert.ok(!denied[0].responseBody.includes("sku-1"), "denial body leaks rows");
  });

  await check("page-denied", async () => {
    const foreign = await page.goto(base + "/invoices/form?tenant_id=2&invoice_id=8");
    assert.equal(foreign?.status(), 403);
    assert.ok((await page.locator("body").textContent())?.includes("denied 2/8"));
    const missing = await page.goto(base + "/invoices/form?tenant_id=1&invoice_id=9");
    assert.equal(missing?.status(), 403);
    assert.ok((await page.locator("body").textContent())?.includes("denied 1/9"));
    await context.clearCookies();
    const ghost = await page.goto(base + "/invoices/form?tenant_id=1&invoice_id=7");
    assert.equal(ghost?.status(), 403);
    await context.addCookies([{ name: "session", value: "tok-alice", url: base }]);
  });

  await check("unknown-path-404", async () => {
    // Fetched, not navigated: the 404 is a plaintext non-document
    // whose render would trip style-src in Chromium. Same origin,
    // same paired server, same truthful status.
    const missing = await page.evaluate(async (origin) => {
      const response = await fetch(origin + "/no-such-path");
      return { status: response.status, body: await response.text() };
    }, base);
    assert.equal(missing.status, 404);
    assert.equal(missing.body, "Not Found");
  });

  await check("grid-caret", async () => {
    await page.goto(base + "/invoice-grid?tenant=1&invoice=7", { waitUntil: "load" });
    await page.waitForFunction(() => document.querySelector("#status")?.textContent?.includes("loaded revision 1"));
    await page.locator("#price\\:k1").evaluate((node) => {
      node.focus();
      node.setSelectionRange(2, 2);
    });
    await page.keyboard.type("5");
    assert.equal(await page.locator("#price\\:k1").inputValue(), "195.99");
    assert.deepEqual(await selectionOf("#price\\:k1"), { start: 3, end: 3, direction: implicitDirection });
    assert.equal(await page.locator("#totals").textContent(), "2 lines, 396.98 total");
    await page.locator("#price\\:k1").evaluate((node) => {
      node.focus();
      node.value = "20.50";
      node.setSelectionRange(1, 3, "backward");
      node.dispatchEvent(new InputEvent("input", { bubbles: true, isComposing: true, data: "0" }));
    });
    await page.waitForTimeout(300);
    assert.equal(await page.locator("#price\\:k1").inputValue(), "20.50");
    assert.deepEqual(await selectionOf("#price\\:k1"), { start: 1, end: 3, direction: "backward" });
    assert.equal(await page.locator("#totals").textContent(), "2 lines, 46.00 total");
  });

  await check("mismatch-save-prompt", async () => {
    mode = "save409";
    posts.length = 0;
    const before = ledger.length;
    await page.locator("#save").click();
    await page.waitForFunction(
      () => document.querySelector("#status")?.textContent?.includes("app updated; refresh to continue"),
      null,
      { timeout: 15000 }
    );
    assert.equal(await statusText(), "app updated; refresh to continue");
    assert.equal(posts.length, 1);
    const refused = ledger.slice(before).filter((entry) => entry.method === "POST");
    assert.equal(refused.length, 1);
    assert.equal(refused[0].status, 409);
    assert.equal(await page.locator("#replay").isDisabled(), true);
    mode = "capture";
  });

  await check("mismatch-load-prompt", async () => {
    mode = "load409";
    await page.goto(base + "/invoice-grid?tenant=1&invoice=7", { waitUntil: "load" });
    await page.waitForFunction(
      () => document.querySelector("#status")?.textContent?.includes("app updated; refresh to continue"),
      null,
      { timeout: 15000 }
    );
    assert.equal(await statusText(), "app updated; refresh to continue");
    mode = "capture";
  });

  await check("refresh-resolves", async () => {
    await page.reload({ waitUntil: "load" });
    await page.waitForFunction(() => document.querySelector("#status")?.textContent?.includes("loaded revision 1"));
    const slot = await page.evaluate(
      () => document.querySelector("script[data-can-generation]")?.getAttribute("data-can-generation")
    );
    assert.equal(slot, generation);
    assert.equal(await page.locator("#price\\:k1").inputValue(), "19.99");
    await page.locator("#qty\\:k1").evaluate((node) => {
      node.focus();
      node.value = "3";
      node.dispatchEvent(new Event("input", { bubbles: true }));
    });
    await page.waitForTimeout(300);
    await page.locator("#save").click();
    await page.waitForFunction(
      () => document.querySelector("#status")?.textContent?.match(/saved revision 2/),
      null,
      { timeout: 15000 }
    );
    assert.equal(await page.locator("#grid h1").textContent(), "Tenant 1 invoice 7 revision 2");
  });

  await check("no-page-faults", async () => {
    assert.deepEqual(pageerrors, []);
    // Denied legs fail resource loads by design; the served CSP's
    // navigate-to directive warns once per load. Pinned exactly.
    const pinned = new Set(["Unrecognized Content-Security-Policy directive 'navigate-to'."]);
    for (const text of consoleErrors) {
      assert.ok(
        /Failed to load resource/.test(text) || pinned.has(text.trim()),
        `non-resource console error: ${text}`
      );
    }
  });

  await page.screenshot({ path: join(outdir, "screenshot.png") });
  const failed = checks.filter((entry) => !entry.passed);
  const report = {
    harness: "w1-grid.mjs",
    browser: wanted,
    version: browser.version(),
    userAgent,
    checks,
    requests,
    ledger,
    headers,
    aborted,
    pageerrors,
    consoleErrors,
    passed: failed.length === 0,
  };
  writeFileSync(join(outdir, "report.json"), JSON.stringify(report, null, 2) + "\n");
  if (!remote) await browser.close();
  if (!report.passed) {
    console.error(`w1-grid ${wanted}: ${failed.length} failing checks`);
    for (const entry of failed) console.error(`- ${entry.name}: ${entry.detail}`);
    process.exit(1);
  }
  process.exit(0);
} catch (error) {
  console.error(`w1-grid ${wanted}: ${String(error?.stack ?? error)}`);
  try {
    if (!remote) await browser.close();
  } catch {}
  process.exit(1);
}
