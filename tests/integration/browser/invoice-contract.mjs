// UP21 contract-edit observer for the Can-authored invoice grid: drives
// the staged live grid (compiler-built paired server plus real SQLite)
// in one named Playwright browser through a single scenario and records
// the network ledger with request/response bytes plus DOM text. Usage:
//   node invoice-contract.mjs <chromium|firefox|webkit> <base> <outdir> <save|invalid|budget>
// Aborts every non-loopback request. Writes report.json and
// screenshot.png into outdir. C-owned; Gate 5 observers stay B-owned.
import { strict as assert } from "node:assert";
import { mkdirSync, writeFileSync } from "node:fs";
import { join } from "node:path";

const [wanted, base, outdir, scenario] = process.argv.slice(2);
const scenarios = ["save", "invalid", "budget"];
if (
  (wanted !== "chromium" && wanted !== "firefox" && wanted !== "webkit") ||
  !base ||
  !outdir ||
  !scenarios.includes(scenario)
) {
  console.error("usage: node invoice-contract.mjs <chromium|firefox|webkit> <base> <outdir> <save|invalid|budget>");
  process.exit(2);
}
mkdirSync(outdir, { recursive: true });

const playwright = await import("playwright");
const browser = await playwright[wanted].launch({ timeout: 120000 });
const checks = [];
const requests = [];
const ledger = [];
const aborted = [];
const pageerrors = [];
const consoleErrors = [];
const check = (name, fn) =>
  Promise.resolve()
    .then(fn)
    .then((detail = "") => checks.push({ name, passed: true, detail }))
    .catch((error) => checks.push({ name, passed: false, detail: String(error?.message ?? error) }));

let report = null;
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
    if (!url.includes("/api/") && !url.includes("/tenants/")) return;
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

  await context.addCookies([{ name: "session", value: "tok-alice", url: base }]);
  const bootResponse = await page.goto(base + "/invoice-grid?tenant=1&invoice=7", { waitUntil: "load" });
  const csp = bootResponse?.headers()?.["content-security-policy"] ?? "";

  // The HTTP legs run first, so the browser may load any committed
  // revision; pin the loaded revision and require the title, status
  // and keyed rows to agree with it.
  let loadedRevision = "";
  await check("boot-loads", async () => {
    await page.waitForFunction(() =>
      document.querySelector("#status")?.textContent?.match(/loaded revision (\d+)/)
    );
    loadedRevision = await page.evaluate(() =>
      document.querySelector("#status")?.textContent?.match(/loaded revision (\d+)/)?.[1]
    );
    assert.ok(loadedRevision, "no loaded revision");
    assert.equal(await page.locator("#grid h1").textContent(), `Tenant 1 invoice 7 revision ${loadedRevision}`);
    assert.equal(await page.locator("#status").textContent(), `loaded revision ${loadedRevision}`);
    assert.deepEqual(
      await page.locator("#grid tbody th").evaluateAll((nodes) => nodes.map((node) => node.textContent)),
      ["k1", "k2"]
    );
  });

  await check("client-identity", async () => {
    const scripts = await page.locator("script").evaluateAll((nodes) => nodes.map((node) => node.getAttribute("src")));
    assert.equal(scripts.length, 1, `scripts: ${JSON.stringify(scripts)}`);
    assert.match(scripts[0], /^\/__can\/assets\/[0-9a-f]{64}\.js$/);
    assert.equal(await page.evaluate(() => typeof window.htmx), "undefined");
    assert.ok(csp.includes("script-src 'self'"), `csp lacks script-src: ${csp}`);
    const bundle = await page.evaluate(async (src) => (await fetch(src)).text(), scripts[0]);
    assert.ok(bundle.includes("$canBrowserMain"), "bundle lacks the Can browser entry");
  });

  const text = (selector) => page.locator(selector).first().textContent({ timeout: 5000 }).catch(() => null);
  const dom = async () => ({
    h1: await text("#grid h1"),
    status: await text("#status"),
    statusRole: await page.locator("#status").first().getAttribute("role", { timeout: 5000 }).catch(() => null),
    totals: await text("#totals"),
    rows: await page.locator("#grid tbody tr").count().catch(() => -1),
  });
  const bodySnippet = () =>
    page.evaluate(() => document.body?.innerHTML?.slice(0, 2000) ?? "").catch(() => "");

  // Input events fold into grid state asynchronously; the status
  // line flips to "unsaved changes" once the edit has landed, and
  // the save press must wait for that or it reads the stale draft.
  // Values are set with a synthetic input event rather than the
  // keyboard: every real keydown on an input also dispatches the
  // grid's Enter-to-save handler, which would press save mid-edit.
  const fill = async (selector, value) => {
    await page.locator(selector).evaluate(
      (node, text) => {
        node.focus();
        node.value = text;
        node.dispatchEvent(new Event("input", { bubbles: true }));
      },
      value
    );
    await page.waitForFunction(() =>
      document.querySelector("#status")?.textContent?.includes("unsaved changes")
    );
  };

  if (scenario === "save") {
    await check("contract-save", async () => {
      await fill("#qty\\:k1", "3");
      await page.locator("#save").click();
      await page.waitForFunction(() =>
        document.querySelector("#status")?.textContent?.match(/saved revision \d+/)
      );
      // The Go suite asserts the exact status (200, or 201 after the
      // status edit); the observer pins exactly one save round trip.
      const saves = ledger.filter((call) => call.method === "POST");
      assert.equal(saves.length, 1, `saves: ${JSON.stringify(saves.map((s) => s.url))}`);
    });
  } else if (scenario === "invalid") {
    // The grid blocks unparsable numbers client-side, so clear the id
    // input instead: the draft still sends and the server rejects it.
    await check("contract-invalid", async () => {
      await fill("#id\\:k1", "");
      await page.locator("#save").click();
      await page.waitForResponse(
        (response) => response.request().method() === "POST" && response.status() === 422,
        { timeout: 15000 }
      );
      await page.waitForFunction(() =>
        document.querySelector("#status")?.textContent?.match(/empty line id/)
      );
      assert.equal(await page.locator("#id\\:k1").inputValue(), "");
    });
  } else {
    await check("contract-budget", async () => {
      await fill("#qty\\:k1", "3");
      await page.locator("#save").click();
      await page.waitForFunction(() =>
        document.querySelector("#status")?.textContent?.includes("save too large; remove lines and retry")
      );
      const saves = ledger.filter((call) => call.method === "POST");
      assert.equal(saves.length, 0, `unexpected saves: ${JSON.stringify(saves.map((s) => s.url))}`);
    });
  }

  const version = await page.evaluate(() => navigator.userAgent).catch(() => "");
  report = {
    passed: checks.every((entry) => entry.passed),
    browser: wanted,
    version,
    scenario,
    loadedRevision,
    checks,
    requests,
    ledger,
    dom: await dom(),
    bodySnippet: await bodySnippet(),
    csp,
    aborted,
    pageerrors,
    consoleErrors,
  };
  await page.screenshot({ path: join(outdir, "screenshot.png") }).catch((error) => {
    report.screenshotError = String(error?.message ?? error);
  });
  writeFileSync(join(outdir, "report.json"), JSON.stringify(report, null, 2) + "\n");
} catch (error) {
  report = report ?? { passed: false, browser: wanted, version: "", scenario, checks, requests, ledger, dom: null, csp: "", aborted, pageerrors, consoleErrors };
  report.fatal = String(error?.message ?? error).slice(0, 2000);
  try {
    writeFileSync(join(outdir, "report.json"), JSON.stringify(report, null, 2) + "\n");
  } catch {}
} finally {
  await browser.close();
}
if (!report.passed) {
  console.error(JSON.stringify(report.checks, null, 2));
  process.exit(1);
}
