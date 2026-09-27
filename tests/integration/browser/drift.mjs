// C06 generation-drift evidence: proves a real old-page/new-server
// refusal end to end. The Go driver serves generation V1, this
// harness boots the page and writes READY, the driver restarts the
// same port on generation V2 and writes GO, and the harness drives
// the stale page: the next action must refuse with the exact 409 and
// the blocking prompt, and a refresh must resolve onto V2. Rollout
// and rollback runs swap V1/V2. Usage:
//   node drift.mjs <chromium|firefox|webkit> <base> <outdir> <script-url> <v1> <v2>
// Aborts every non-loopback request. Writes report.json and
// screenshot.png into outdir.
import { strict as assert } from "node:assert";
import { mkdirSync, writeFileSync, existsSync } from "node:fs";
import { join } from "node:path";
import { launchWanted } from "./firefox-remote.mjs";

const [wanted, base, outdir, scriptUrl, v1, v2] = process.argv.slice(2);
if (
  (wanted !== "chromium" && wanted !== "firefox" && wanted !== "webkit") ||
  !base ||
  !outdir ||
  !scriptUrl?.startsWith("/__can/assets/") ||
  !/^[0-9a-f]{64}$/.test(v1 ?? "") ||
  !/^[0-9a-f]{64}$/.test(v2 ?? "") ||
  v1 === v2
) {
  console.error("usage: node drift.mjs <chromium|firefox|webkit> <base> <outdir> <script-url> <v1> <v2>");
  process.exit(2);
}
mkdirSync(outdir, { recursive: true });

const playwright = await import("playwright");
const { browser, remote } = await launchWanted(playwright, wanted);
let userAgent = "";
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

const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms));
const waitForFile = async (path, timeoutMs) => {
  const started = Date.now();
  while (!existsSync(path)) {
    if (Date.now() - started > timeoutMs) throw new Error(`timed out waiting for ${path}`);
    await sleep(250);
  }
};

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
  const posts = [];
  page.on("request", (request) => {
    if (request.method() === "POST" && request.url().includes("/api/tenants/")) posts.push(request.url());
  });

  const statusText = () => page.locator("#status").textContent();
  const slotOf = () =>
    page.evaluate(
      () => document.querySelector("script[data-can-generation]")?.getAttribute("data-can-generation")
    );

  await context.addCookies([{ name: "session", value: "tok-alice", url: base }]);
  await page.goto(base + "/invoice-grid?tenant=1&invoice=7", { waitUntil: "load" });
  userAgent = await page.evaluate(() => navigator.userAgent);

  await check("boot-v1", async () => {
    await page.waitForFunction(() => document.querySelector("#status")?.textContent?.includes("loaded revision 1"));
    assert.equal(await slotOf(), v1);
  });

  // Make the draft unsaved so the post-restart press sends.
  await page.locator("#qty\\:k1").evaluate((node) => {
    node.focus();
    node.value = "3";
    node.dispatchEvent(new Event("input", { bubbles: true }));
  });
  await page.waitForTimeout(300);

  // Rendezvous: the driver restarts the port on V2, then writes GO.
  writeFileSync(join(outdir, "READY"), "ready\n");
  await waitForFile(join(outdir, "GO"), 120000);

  await check("old-page-refused", async () => {
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
    const shape = JSON.parse(refused[0].responseBody);
    assert.deepEqual(Object.keys(shape).sort(), ["kind", "schemaVersion", "serverGeneration"]);
    assert.equal(shape.schemaVersion, 1);
    assert.equal(shape.kind, "can.generation-mismatch");
    assert.equal(shape.serverGeneration, v2);
  });

  await check("refused-save-never-ran", async () => {
    // The refused save committed nothing: the reread below still
    // shows revision 1 and the V2 slot.
    await page.reload({ waitUntil: "load" });
    await page.waitForFunction(() => document.querySelector("#status")?.textContent?.includes("loaded revision 1"));
    assert.equal(await slotOf(), v2);
    assert.equal(await page.locator("#grid h1").textContent(), "Tenant 1 invoice 7 revision 1");
  });

  await check("no-page-faults", async () => {
    assert.deepEqual(pageerrors, []);
    // The refused save fails its resource load by design; the served
    // CSP's navigate-to directive warns once per load. Pinned exactly.
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
    harness: "drift.mjs",
    browser: wanted,
    version: browser.version(),
    userAgent,
    checks,
    requests,
    ledger,
    aborted,
    pageerrors,
    consoleErrors,
    passed: failed.length === 0,
  };
  writeFileSync(join(outdir, "report.json"), JSON.stringify(report, null, 2) + "\n");
  if (!remote) await browser.close();
  if (!report.passed) {
    console.error(`drift ${wanted}: ${failed.length} failing checks`);
    for (const entry of failed) console.error(`- ${entry.name}: ${entry.detail}`);
    process.exit(1);
  }
  process.exit(0);
} catch (error) {
  console.error(`drift ${wanted}: ${String(error?.stack ?? error)}`);
  try {
    if (!remote) await browser.close();
  } catch {}
  process.exit(1);
}
