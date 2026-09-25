// Pinned browser evidence for the UP19 server-rendered invoice form:
// drives the staged live /invoices/form?tenant_id=1&invoice_id=7 page with
// upstream HTMX only. Aborts every non-localhost request, so a passing run
// proves the page never needs a CDN, authored script, or client runtime.
// Qualifies the compiler-derived swap policy end to end: the served form
// carries exact hx-status exceptions generated from the checked HTML action
// case table, so every declared outcome (200/422/409/403/503) swaps into
// the status region with the draft and focus preserved and errors
// announced. Session travels by cookie only: no hidden session, operation,
// or invoice inputs exist anywhere on the page. UP23 runs the same legs on
// the paired server (where the grid module script rides along inertly) in
// both named browsers, and adds guard legs: control headers, OOB content,
// redirects, missing targets before/during requests with
// can:action-occurrence evidence, and remount stability.
// Usage: node invoice.mjs <base> <outdir> <dbpath> [browser] [script-url]
import { strict as assert } from "node:assert";
import { mkdirSync, writeFileSync, chmodSync } from "node:fs";
import { join, dirname } from "node:path";

const [base, outdir, dbpath, wantedArg, scriptArg] = process.argv.slice(2);
if (!base || !outdir || !dbpath) {
  console.error("usage: node invoice.mjs <base> <outdir> <dbpath> [browser] [script-url]");
  process.exit(2);
}
const wanted = wantedArg ?? "chromium";
if (wanted !== "chromium" && wanted !== "webkit") {
  console.error(`unknown browser ${wanted}`);
  process.exit(2);
}
mkdirSync(outdir, { recursive: true });

const playwright = await import("playwright");
const browser = await playwright[wanted].launch({ timeout: 120000 });
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
  const setSession = (value) =>
    context
      .clearCookies()
      .then(() =>
        value === undefined
          ? undefined
          : context.addCookies([{ name: "session", value, url: base }]),
      );
  await setSession("tok-alice");
  await page.goto(base + "/invoices/form?tenant_id=1&invoice_id=7", { waitUntil: "load" });

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
      (r) => r.url().endsWith("/tenants/1/invoices/7") && r.request().method() === "POST",
    );
  const submit = async () => {
    const answered = saveResponse();
    await page.locator('#invoice_form button[type="submit"]').click();
    return answered;
  };

  await check("htmx-loaded", async () => {
    assert.equal(await page.evaluate(() => typeof window.htmx), "object");
  });
  await check("pinned-scripts", async () => {
    const scripts = await page.locator("script").evaluateAll((nodes) =>
      nodes.map((node) => ({
        src: node.getAttribute("src"),
        integrity: node.getAttribute("integrity"),
        type: node.getAttribute("type"),
      }))
    );
    assert.equal(scripts.length, 2);
    assert.equal(scripts[0].src, "/__can/assets/htmx-4.0.0.min.js");
    assert.equal(scripts[0].integrity, "sha384-BvJpBiO8Kh31EqtJe5DRIeWrHWnCGkwytKs9NKFi86Hhw96dEqdEMzZDeK9iEGTc");
    assert.equal(scripts[1].src, "/__can/assets/htmx-guard.js");
    assert.equal(scripts[1].type, "module");
    assert.equal(scripts[1].integrity, "sha384-mr/IRfJgLjok38ftBi21o/T8c9cnFZrvEKtiwVjIOFAlo3Z7h1rGMYsWvebDJ8kG");
  });
  await check("htmx-config-policy", async () => {
    const content = await page.locator('meta[name="htmx-config"]').getAttribute("content");
    const config = JSON.parse(content);
    assert.equal(config.mode, "same-origin");
    assert.deepEqual(config.noSwap, [204, 304, "4xx", "5xx"]);
  });
  await check("form-renders", async () => {
    assert.equal(await page.locator("#invoice_form").getAttribute("hx-post"), "/tenants/1/invoices/7");
    assert.equal(await page.locator("#invoice_form").getAttribute("hx-target"), "#invoice_status");
    assert.equal(await page.locator("#invoice_form").getAttribute("hx-swap"), "innerHTML");
    for (const code of [403, 409, 422, 503]) {
      assert.equal(
        await page.locator("#invoice_form").getAttribute(`hx-status:${code}`),
        '{"swap":"innerHTML"}',
      );
    }
    assert.equal(await page.locator("#invoice_form").getAttribute("hx-status:200"), null);
    for (const name of ["session_token", "operation_id", "invoice_id"]) {
      assert.equal(await page.locator(`#invoice_form input[name="${name}"]`).count(), 0);
    }
    assert.equal(await hidden("revision"), "1");
    assert.equal(await page.locator("#invoice_seats").inputValue(), "2");
    assert.equal(await page.locator("#invoice_details").inputValue(), `Acme <em>&" 'coop'"`);
    assert.equal(await page.locator('input[name="lines[k1][id]"]').inputValue(), "sku-1 <b>");
    assert.equal(await page.locator('input[name="lines[k1][quantity]"]').inputValue(), "2");
    assert.equal(await page.locator('input[name="lines[k1][price]"]').inputValue(), "19.99");
    const html = await page.locator("#invoice_form").innerHTML();
    assert.ok(html.includes("Acme &lt;em&gt;"));
    assert.ok(html.includes("sku-1 &lt;b&gt;"));
    assert.ok(!html.includes("<em>"));
  });
  await check("save-200-swap", async () => {
    await page.locator("#invoice_details").fill("Browser Save");
    const answered = await submit();
    assert.equal(answered.status(), 200);
    await page.waitForFunction(() =>
      document.querySelector("#invoice_status")?.textContent?.includes("saved revision 2")
    );
    assert.equal(await page.locator('#invoice_status [role="status"]').count(), 1);
    assert.equal(await page.locator("#invoice_details").inputValue(), "Browser Save");
    assert.equal(await page.locator("#invoice_seats").inputValue(), "2");
  });
  await check("validation-422-swap", async () => {
    await page.locator("#invoice_seats").fill("many");
    const answered = page.waitForResponse(
      (r) => r.url().endsWith("/tenants/1/invoices/7") && r.request().method() === "POST",
    );
    await page.locator("#invoice_seats").press("Enter");
    assert.equal((await answered).status(), 422);
    await page.waitForFunction(() =>
      document.querySelector("#invoice_status")?.textContent?.includes("invoice invalid")
    );
    const alert = page.locator("#invoice_status [role=alert]");
    assert.equal(await alert.count(), 1);
    assert.ok((await alert.textContent())?.includes("seats: bad seats"));
  });
  await check("draft-and-focus-preserved", async () => {
    assert.equal(await page.locator("#invoice_seats").inputValue(), "many");
    assert.equal(await page.locator("#invoice_details").inputValue(), "Browser Save");
    assert.equal(await page.locator('input[name="lines[k1][id]"]').inputValue(), "sku-1 <b>");
    assert.equal(await hidden("revision"), "1");
    assert.equal(await page.evaluate(() => document.activeElement?.id), "invoice_seats");
  });
  await check("stale-409-swap", async () => {
    await page.locator("#invoice_seats").fill("2");
    await page.locator("#invoice_details").fill("Stale Attempt");
    const answered = await submit();
    assert.equal(answered.status(), 409);
    await page.waitForFunction(() =>
      document.querySelector("#invoice_status")?.textContent?.includes("conflict: stale revision")
    );
    assert.equal(await page.locator("#invoice_status [role=alert]").count(), 1);
    assert.ok(!(await statusText())?.includes("invoice invalid"));
    assert.equal(await page.locator("#invoice_details").inputValue(), "Stale Attempt");
    assert.equal(await hidden("revision"), "1");
  });
  await check("busy-503-swap", async () => {
    const dir = dirname(dbpath);
    chmodSync(dir, 0o555);
    try {
      await setHidden("revision", "2");
      await page.locator("#invoice_details").fill("Outage Attempt");
      const answered = await submit();
      assert.equal(answered.status(), 503);
      await page.waitForFunction(() =>
        document
          .querySelector("#invoice_status")
          ?.textContent?.includes("unavailable: store unavailable")
      );
      assert.equal(await page.locator("#invoice_status [role=alert]").count(), 1);
      assert.ok(!(await statusText())?.includes("conflict:"));
      assert.equal(await page.locator("#invoice_details").inputValue(), "Outage Attempt");
      assert.equal(await hidden("revision"), "2");
    } finally {
      chmodSync(dir, 0o700);
    }
  });
  await check("denied-403-swap", async () => {
    await setSession("tok-ghost");
    await page.locator("#invoice_details").fill("Ghost Attempt");
    const answered = await submit();
    assert.equal(answered.status(), 403);
    await page.waitForFunction(() =>
      document.querySelector("#invoice_status")?.textContent?.includes("forbidden: request denied")
    );
    assert.equal(await page.locator("#invoice_status [role=alert]").count(), 1);
    assert.ok(!(await statusText())?.includes("unavailable:"));
    assert.equal(await page.locator("#invoice_details").inputValue(), "Ghost Attempt");
    await setSession("tok-alice");
  });
  await check("target-absence-stable", async () => {
    await page.evaluate(() => {
      const status = document.getElementById("invoice_status");
      window.__up19status = status.cloneNode(true);
      status.remove();
    });
    await setHidden("revision", "1");
    await page.locator("#invoice_details").fill("Target Absent Attempt");
    const before = page.url();
    const answered = saveResponse();
    await page.locator('#invoice_form button[type="submit"]').click();
    const outcome = await Promise.race([
      answered.then((r) => `response ${r.status()}`),
      page.waitForTimeout(8000).then(() => "no request sent"),
    ]);
    assert.equal(page.url(), before);
    assert.equal(await page.locator("#invoice_details").inputValue(), "Target Absent Attempt");
    assert.equal(pageerrors.length, 0, pageerrors.join("; "));
    return outcome;
  });
  await check("target-restored-save", async () => {
    await page.evaluate(() => {
      document.body.appendChild(window.__up19status);
      window.__up19status = undefined;
    });
    await setHidden("revision", "2");
    await page.locator("#invoice_details").fill("Target Restored Save");
    const answered = await submit();
    assert.equal(answered.status(), 200);
    await page.waitForFunction(() =>
      document.querySelector("#invoice_status")?.textContent?.includes("saved revision 3")
    );
  });
  await check("no-external-requests", async () => {
    assert.equal(aborted.length, 0);
    for (const entry of requests) {
      assert.ok(new URL(entry.url).hostname === "127.0.0.1", entry.url);
    }
    assert.ok(requests.some((entry) => entry.url.endsWith("/__can/assets/htmx-4.0.0.min.js") && entry.status === 200));
    assert.ok(requests.some((entry) => entry.url.endsWith("/__can/assets/htmx-guard.js") && entry.status === 200));
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
