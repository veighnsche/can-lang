// Pinned browser evidence for the T15 server-rendered invoice form:
// drives the staged live /invoices/form page with upstream HTMX only.
// Aborts every non-localhost request, so a passing run proves the page
// never needs a CDN, authored script, or client runtime. Qualifies the
// compiler-owned swap policy end to end: 200/422 swap into the status
// region with the draft and focus preserved and errors announced,
// while 409/403/503 never swap so the prior fragment stays visible.
// Usage: node invoice.mjs <base> <outdir> <dbpath>
import { strict as assert } from "node:assert";
import { randomUUID } from "node:crypto";
import { mkdirSync, writeFileSync, chmodSync } from "node:fs";
import { join } from "node:path";
import { chromium } from "playwright";

const [base, outdir, dbpath] = process.argv.slice(2);
if (!base || !outdir || !dbpath) {
  console.error("usage: node invoice.mjs <base> <outdir> <dbpath>");
  process.exit(2);
}
mkdirSync(outdir, { recursive: true });

const browser = await chromium.launch();
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
  await context.addCookies([{ name: "session", value: "tok-alice", url: base }]);
  await page.goto(base + "/invoices/form?invoice_id=inv-1", { waitUntil: "load" });

  const hidden = (name) =>
    page.evaluate((n) => document.querySelector(`#invoice_form input[name="${n}"]`).value, name);
  const setHidden = (name, value) =>
    page.evaluate(
      ([n, v]) => {
        document.querySelector(`#invoice_form input[name="${n}"]`).value = v;
      },
      [name, value],
    );
  const statusText = () => page.locator("#invoice_status").textContent();
  const saveResponse = () =>
    page.waitForResponse(
      (r) => r.url().endsWith("/invoices/save-form") && r.request().method() === "POST",
    );
  const submit = async () => {
    const answered = saveResponse();
    await page.locator('#invoice_form button[type="submit"]').click();
    return answered;
  };

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
  await check("htmx-config-policy", async () => {
    const content = await page.locator('meta[name="htmx-config"]').getAttribute("content");
    const config = JSON.parse(content);
    assert.equal(config.mode, "same-origin");
    assert.ok(config.noSwap.includes(204));
    assert.ok(config.noSwap.includes(304));
    for (const code of [403, 409, 503]) assert.ok(config.noSwap.includes(code), `noSwap lacks ${code}`);
    assert.ok(!config.noSwap.includes(422), "422 must swap");
    assert.ok(!config.noSwap.includes(200), "200 must swap");
  });
  await check("form-renders", async () => {
    assert.equal(await page.locator("#invoice_form").getAttribute("hx-post"), "/invoices/save-form");
    assert.equal(await page.locator("#invoice_form").getAttribute("hx-target"), "#invoice_status");
    assert.equal(await page.locator("#invoice_form").getAttribute("hx-swap"), "innerHTML");
    assert.equal(await hidden("session_token"), "tok-alice");
    assert.match(await hidden("operation_id"), /^[0-9a-f-]{36}$/);
    assert.equal(await hidden("revision"), "1");
    assert.equal(await page.locator("#invoice_customer").inputValue(), `Acme <em>&" 'coop'"`);
    assert.equal(await page.locator('input[name="lines[k1][sku]"]').inputValue(), "sku-1 <b>");
    const html = await page.locator("#invoice_form").innerHTML();
    assert.ok(html.includes("Acme &lt;em&gt;"));
    assert.ok(html.includes("sku-1 &lt;b&gt;"));
    assert.ok(!html.includes("<em>"));
  });
  await check("save-200-swap", async () => {
    await page.locator("#invoice_customer").fill("Browser Save");
    const answered = await submit();
    assert.equal(answered.status(), 200);
    await page.waitForFunction(() =>
      document.querySelector("#invoice_status")?.textContent?.includes("saved inv-1 revision 2")
    );
    assert.equal(await page.locator('#invoice_status [role="status"]').count(), 1);
    assert.equal(await page.locator("#invoice_customer").inputValue(), "Browser Save");
  });
  await check("validation-422-swap", async () => {
    await page.locator("#invoice_customer").fill("");
    await page.locator('input[name="lines[k1][sku]"]').fill("browser-sku-7");
    const answered = page.waitForResponse(
      (r) => r.url().endsWith("/invoices/save-form") && r.request().method() === "POST",
    );
    await page.locator("#invoice_customer").press("Enter");
    assert.equal((await answered).status(), 422);
    await page.waitForFunction(() =>
      document.querySelector("#invoice_status")?.textContent?.includes("rejected: empty customer")
    );
    const alert = page.locator("#invoice_status [role=alert]");
    assert.equal(await alert.count(), 1);
    assert.ok((await alert.textContent())?.includes("empty customer"));
  });
  await check("draft-and-focus-preserved", async () => {
    assert.equal(await page.locator("#invoice_customer").inputValue(), "");
    assert.equal(await page.locator('input[name="lines[k1][sku]"]').inputValue(), "browser-sku-7");
    assert.equal(await hidden("revision"), "1");
    assert.equal(await page.evaluate(() => document.activeElement?.id), "invoice_customer");
  });
  await check("stale-409-no-swap", async () => {
    await setHidden("operation_id", randomUUID());
    await page.locator("#invoice_customer").fill("Stale Attempt");
    const answered = await submit();
    assert.equal(answered.status(), 409);
    await page.waitForTimeout(300);
    assert.ok((await statusText())?.includes("rejected: empty customer"));
    assert.ok(!(await statusText())?.includes("stale"));
    assert.equal(await page.locator("#invoice_customer").inputValue(), "Stale Attempt");
  });
  await check("busy-503-no-swap", async () => {
    chmodSync(dbpath, 0o000);
    try {
      await setHidden("operation_id", randomUUID());
      await setHidden("revision", "2");
      await page.locator("#invoice_customer").fill("Outage Attempt");
      const answered = await submit();
      assert.equal(answered.status(), 503);
      await page.waitForTimeout(300);
      assert.ok((await statusText())?.includes("rejected: empty customer"));
      assert.ok(!(await statusText())?.includes("busy"));
      assert.equal(await page.locator("#invoice_customer").inputValue(), "Outage Attempt");
    } finally {
      chmodSync(dbpath, 0o600);
    }
  });
  await check("denied-403-no-swap", async () => {
    await setHidden("session_token", "tok-ghost");
    await page.locator("#invoice_customer").fill("Ghost Attempt");
    const answered = await submit();
    assert.equal(answered.status(), 403);
    await page.waitForTimeout(300);
    assert.ok((await statusText())?.includes("rejected: empty customer"));
    assert.ok(!(await statusText())?.includes("denied"));
    assert.equal(await page.locator("#invoice_customer").inputValue(), "Ghost Attempt");
    await setHidden("session_token", "tok-alice");
  });
  await check("target-absence-stable", async () => {
    await page.evaluate(() => {
      const status = document.getElementById("invoice_status");
      window.__t15status = status.cloneNode(true);
      status.remove();
    });
    await setHidden("operation_id", randomUUID());
    await setHidden("revision", "1");
    await page.locator("#invoice_customer").fill("Target Absent Attempt");
    const before = page.url();
    const answered = saveResponse();
    await page.locator('#invoice_form button[type="submit"]').click();
    const outcome = await Promise.race([
      answered.then((r) => `response ${r.status()}`),
      page.waitForTimeout(8000).then(() => "no request sent"),
    ]);
    assert.equal(page.url(), before);
    assert.equal(await page.locator("#invoice_customer").inputValue(), "Target Absent Attempt");
    assert.equal(pageerrors.length, 0, pageerrors.join("; "));
    return outcome;
  });
  await check("target-restored-save", async () => {
    await page.evaluate(() => {
      document.body.appendChild(window.__t15status);
      window.__t15status = undefined;
    });
    await setHidden("operation_id", randomUUID());
    await setHidden("revision", "2");
    await page.locator("#invoice_customer").fill("Target Absent Save");
    const answered = await submit();
    assert.equal(answered.status(), 200);
    await page.waitForFunction(() =>
      document.querySelector("#invoice_status")?.textContent?.includes("saved inv-1 revision 3")
    );
  });
  await check("no-external-requests", async () => {
    assert.equal(aborted.length, 0);
    for (const entry of requests) {
      assert.ok(new URL(entry.url).hostname === "127.0.0.1", entry.url);
    }
    assert.ok(requests.some((entry) => entry.url.endsWith("/__can/assets/htmx-4.0.0.min.js") && entry.status === 200));
    assert.equal(pageerrors.length, 0, pageerrors.join("; "));
  });

  await page.screenshot({ path: join(outdir, "screenshot.png"), fullPage: true });
  await context.close();
} finally {
  await browser.close();
}

const passed = checks.every((entry) => entry.passed);
writeFileSync(
  join(outdir, "report.json"),
  JSON.stringify({ browser: "chromium", version: browser.version(), base, passed, checks, aborted, requests, pageerrors, consoleErrors }, null, 2) + "\n"
);
for (const entry of checks) console.log(`${entry.passed ? "PASS" : "FAIL"} ${entry.name}${entry.detail ? ` ${entry.detail}` : ""}`);
if (!passed) process.exit(1);
console.log("browser invoice evidence passed");
