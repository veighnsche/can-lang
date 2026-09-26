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
const limitations = [];
let finalOccurrences = [];
const limit = (id, detail) => limitations.push({ id, detail });
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
  const bootResponse = await page.goto(base + "/invoices/form?tenant_id=1&invoice_id=7", { waitUntil: "load" });
  const csp = bootResponse?.headers()?.["content-security-policy"] ?? "";
  // Guard occurrences surface as can:action-occurrence CustomEvents
  // with the frozen record in detail; every guard leg below reads
  // the new entries since its own start marker.
  await page.evaluate(() => {
    window.__occurrences = [];
    document.addEventListener("can:action-occurrence", (event) => {
      window.__occurrences.push(event.detail);
    });
  });
  const occurrences = () => page.evaluate(() => window.__occurrences);
  const nodeCount = () => page.evaluate(() => document.querySelectorAll("*").length);

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
    // Unpaired servers carry HTMX plus the guard; paired servers
    // splice the exact report-selected browser script after them.
    const expected = scriptArg ?? "";
    assert.equal(scripts.length, expected ? 3 : 2);
    assert.equal(scripts[0].src, "/__can/assets/htmx-4.0.0.min.js");
    assert.equal(scripts[0].integrity, "sha384-BvJpBiO8Kh31EqtJe5DRIeWrHWnCGkwytKs9NKFi86Hhw96dEqdEMzZDeK9iEGTc");
    assert.equal(scripts[1].src, "/__can/assets/htmx-guard.js");
    assert.equal(scripts[1].type, "module");
    assert.equal(scripts[1].integrity, "sha384-mr/IRfJgLjok38ftBi21o/T8c9cnFZrvEKtiwVjIOFAlo3Z7h1rGMYsWvebDJ8kG");
    assert.ok(csp.includes("script-src 'self'"), `csp lacks script-src: ${csp}`);
    if (expected) {
      assert.equal(scripts[2].src, expected);
      assert.equal(scripts[2].type, "module");
      // The paired grid module must stay inert on the form page: no
      // grid root, no status node, no invoice API call at boot.
      assert.equal(await page.locator("#grid").count(), 0);
      assert.equal(await page.locator("#status").count(), 0);
      assert.ok(
        !requests.some((entry) => entry.url.includes("/api/")),
        "paired grid must not call invoice APIs from the form page"
      );
    }
    return expected ? "htmx+guard+paired (grid inert)" : "htmx+guard";
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
    const nodesBefore = await nodeCount();
    await page.evaluate(() => {
      const status = document.getElementById("invoice_status");
      window.__up19status = status.cloneNode(true);
      status.remove();
    });
    await setHidden("revision", "1");
    await page.locator("#invoice_details").fill("Target Absent Attempt");
    const before = page.url();
    const postsBefore = requests.filter((entry) => entry.method === "POST").length;
    const seenBefore = (await occurrences()).length;
    const answered = saveResponse();
    await page.locator('#invoice_form button[type="submit"]').click();
    const outcome = await Promise.race([
      answered.then((r) => `response ${r.status()}`),
      page.waitForTimeout(8000).then(() => "no request sent"),
    ]);
    // Whether HTMX issues the request with a missing target is
    // upstream behavior; the guard contract is invariant: no swap
    // target, no mutation, draft and page intact, no page error. A
    // sent request must carry a request-phase occurrence instead.
    if (outcome !== "no request sent") {
      const fresh = (await occurrences()).slice(seenBefore);
      assert.equal(fresh.length, 1, JSON.stringify(fresh));
      assert.equal(fresh[0].kind, "action::missing_target");
      assert.equal(fresh[0].phase, "request");
      assert.equal(fresh[0].effect, "none");
    } else {
      assert.equal(requests.filter((entry) => entry.method === "POST").length, postsBefore);
    }
    assert.equal(page.url(), before);
    assert.equal(await page.locator("#invoice_details").inputValue(), "Target Absent Attempt");
    assert.equal(pageerrors.length, 0, pageerrors.join("; "));
    return `${outcome}; ${nodesBefore} nodes before removal`;
  });
  await check("remount-stable", async () => {
    const nodesBefore = await nodeCount();
    const restored = await page.evaluate(() => {
      const subtree = window.__up19status.querySelectorAll("*").length + 1;
      document.body.appendChild(window.__up19status);
      return subtree;
    });
    assert.equal(await nodeCount(), nodesBefore + restored, "remount must restore exactly the removed subtree");
    const ids = await page.evaluate(() => {
      const found = [...document.querySelectorAll("[id]")].map((node) => node.id);
      return { total: found.length, duplicates: found.length - new Set(found).size };
    });
    assert.equal(ids.duplicates, 0, "remount must not duplicate ids");
    return `${restored} nodes restored, ${ids.total} unique ids`;
  });
  await check("target-restored-save", async () => {
    await setHidden("revision", "2");
    await page.locator("#invoice_details").fill("Target Restored Save");
    const answered = await submit();
    assert.equal(answered.status(), 200);
    await page.waitForFunction(() =>
      document.querySelector("#invoice_status")?.textContent?.includes("saved revision 3")
    );
  });
  await check("missing-target-during", async () => {
    await context.route("**/tenants/1/invoices/7", async (route) => {
      if (route.request().method() !== "POST") return route.continue();
      await new Promise((resolve) => setTimeout(resolve, 1500));
      await route.continue();
    });
    try {
      await setHidden("revision", "3");
      await page.locator("#invoice_details").fill("During Flight");
      const regionBefore = await statusText();
      const seenBefore = (await occurrences()).length;
      const sent = page.waitForRequest(
        (request) => request.url().endsWith("/tenants/1/invoices/7") && request.method() === "POST"
      );
      const answered = saveResponse();
      await page.locator('#invoice_form button[type="submit"]').click();
      await sent;
      await page.evaluate(() => {
        document.getElementById("invoice_status").remove();
      });
      const response = await answered;
      assert.equal(response.status(), 200, "the flight must commit upstream");
      await page.waitForTimeout(600);
      // Restore before asserting: later legs need the target even
      // when this leg's occurrence shape changes.
      await page.evaluate(() => {
        if (document.getElementById("invoice_status") === null) {
          document.body.appendChild(window.__up19status);
        }
      });
      const fresh = (await occurrences()).slice(seenBefore);
      assert.equal(fresh.length, 1, JSON.stringify(fresh));
      assert.equal(fresh[0].kind, "action::missing_target");
      assert.equal(fresh[0].phase, "response");
      assert.equal(fresh[0].effect, "uncertain");
      assert.ok(fresh[0].reason === "target_absent" || fresh[0].reason === "target_detached", fresh[0].reason);
      assert.equal(await page.locator("#invoice_details").inputValue(), "During Flight");
      assert.equal(pageerrors.length, 0, pageerrors.join("; "));
      return `uncertain ${fresh[0].reason}; upstream committed`;
    } finally {
      await context.unroute("**/tenants/1/invoices/7");
    }
  });
  await check("final-save", async () => {
    await setHidden("revision", "4");
    await page.locator("#invoice_details").fill("Guard Final");
    const answered = await submit();
    assert.equal(answered.status(), 200);
    await page.waitForFunction(() =>
      document.querySelector("#invoice_status")?.textContent?.includes("saved revision 5")
    );
  });
  const guardFulfill = async (name, fulfill, occurrence) => {
    await check(name, async () => {
      const regionBefore = await statusText();
      const draftBefore = await page.locator("#invoice_details").inputValue();
      const seenBefore = (await occurrences()).length;
      await context.route("**/tenants/1/invoices/7", async (route) => {
        if (route.request().method() !== "POST") return route.continue();
        await fulfill(route);
      });
      try {
        await page.locator('#invoice_form button[type="submit"]').click();
        await page.waitForFunction(
          (count) => window.__occurrences.length > count,
          seenBefore,
          { timeout: 15000 }
        );
      } finally {
        await context.unroute("**/tenants/1/invoices/7");
      }
      const fresh = (await occurrences()).slice(seenBefore);
      assert.equal(fresh.length, 1, JSON.stringify(fresh));
      for (const [key, value] of Object.entries(occurrence)) assert.equal(fresh[0][key], value, key);
      assert.equal(await statusText(), regionBefore, "guarded swap must leave the region untouched");
      assert.equal(await page.locator("#invoice_details").inputValue(), draftBefore);
      assert.equal(pageerrors.length, 0, pageerrors.join("; "));
      return `${fresh[0].phase}/${fresh[0].reason}`;
    });
  };
  await guardFulfill(
    "control-header",
    (route) =>
      route.fulfill({
        status: 200,
        contentType: "text/html; charset=utf-8",
        headers: { "HX-Redirect": "/invoices/form?tenant_id=1&invoice_id=7" },
        body: "<p>Saved revision 99</p>",
      }),
    { kind: "action::protocol", phase: "response", effect: "uncertain", reason: "control_header", header: "redirect" }
  );
  await guardFulfill(
    "oob-rejected",
    (route) =>
      route.fulfill({
        status: 200,
        contentType: "text/html; charset=utf-8",
        body: '<div id="invoice_status" hx-swap-oob="innerHTML">HACK</div><p>Saved revision 99</p>',
      }),
    { kind: "action::protocol", phase: "swap", effect: "uncertain", reason: "task_shape" }
  );
  await guardFulfill(
    "partial-rejected",
    (route) =>
      route.fulfill({
        status: 200,
        contentType: "text/html; charset=utf-8",
        body: '<p>Saved revision 99</p><template hx type="partial" hx-target="#other"><p>partial intruder</p></template>',
      }),
    { kind: "action::protocol", phase: "swap", effect: "uncertain", reason: "task_shape" }
  );
  await check("redirect-rejected", async () => {
    // A followed redirect needs a same-origin 3xx. The server
    // correctly emits none, and only Chromium's interception can
    // fulfill one; on WebKit a harness redirector would have to live
    // cross-port (a second listener), which the served connect-src
    // 'self' policy blocks before any fetch runs. The guard's
    // redirect branch is engine-independent shipped JS (pinned
    // integrity, unit-covered), qualified end to end on Chromium;
    // WebKit records the harness gap as a limit instead of a silent
    // skip.
    if (wanted === "webkit") {
      limit(
        "L-redirect-webkit",
        "no same-origin 302 is producible on WebKit (fulfilled 3xx rejected, " +
          "cross-port redirector blocked by connect-src 'self', server emits no redirects); " +
          "the engine-independent redirect branch is qualified on Chromium."
      );
      return "webkit harness gap, chromium-qualified";
    }
    const regionBefore = await statusText();
    const draftBefore = await page.locator("#invoice_details").inputValue();
    const seenBefore = (await occurrences()).length;
    await context.route("**/tenants/1/invoices/7", async (route) => {
      if (route.request().method() !== "POST") return route.continue();
      await route.fulfill({
        status: 302,
        headers: { location: "/invoices/form?tenant_id=1&invoice_id=7" },
        body: "",
      });
    });
    try {
      await page.locator('#invoice_form button[type="submit"]').click();
      await page.waitForFunction((count) => window.__occurrences.length > count, seenBefore, {
        timeout: 15000,
      });
    } finally {
      await context.unroute("**/tenants/1/invoices/7");
    }
    const fresh = (await occurrences()).slice(seenBefore);
    assert.equal(fresh.length, 1, JSON.stringify(fresh));
    assert.equal(fresh[0].kind, "action::protocol");
    assert.equal(fresh[0].phase, "response");
    assert.equal(fresh[0].effect, "uncertain");
    assert.equal(fresh[0].reason, "redirect");
    assert.equal(await statusText(), regionBefore, "guarded swap must leave the region untouched");
    assert.equal(await page.locator("#invoice_details").inputValue(), draftBefore);
    assert.equal(pageerrors.length, 0, pageerrors.join("; "));
    return "response/redirect";
  });
  await check("no-external-requests", async () => {
    assert.equal(aborted.length, 0);
    for (const entry of requests) {
      assert.ok(new URL(entry.url).hostname === "127.0.0.1", entry.url);
    }
    assert.ok(requests.some((entry) => entry.url.endsWith("/__can/assets/htmx-4.0.0.min.js") && entry.status === 200));
    assert.ok(requests.some((entry) => entry.url.endsWith("/__can/assets/htmx-guard.js") && entry.status === 200));
    assert.equal(pageerrors.length, 0, pageerrors.join("; "));
    const pinned = new Set([
      "Unrecognized Content-Security-Policy directive 'navigate-to'.",
      "Refused to apply a stylesheet because its hash, its nonce, or 'unsafe-inline' does not appear in the style-src directive of the Content Security Policy.",
    ]);
    const rest = consoleErrors.filter((text) => !pinned.has(text.trim()) && !/Failed to load resource/.test(text));
    // HTMX logs a target error when the swap target is missing
    // before the request; anything beyond that and the pinned engine
    // warnings fails.
    const htmxTarget = /htmx:targetError|Target .* not found|invoice_status/;
    const unexplained = rest.filter((text) => !htmxTarget.test(text));
    assert.equal(unexplained.length, 0, unexplained.join("; "));
    return `${(await occurrences()).length} guard occurrences recorded`;
  });

  await page.screenshot({ path: join(outdir, "screenshot.png") });
  finalOccurrences = await occurrences();
  await context.close();
} finally {
  await browser.close();
}

const passed = checks.every((entry) => entry.passed);
writeFileSync(
  join(outdir, "report.json"),
  JSON.stringify({ browser: wanted, version: browser.version(), base, passed, checks, limitations, aborted, requests, pageerrors, consoleErrors, occurrences: finalOccurrences }, null, 2) + "\n"
);
for (const entry of checks) console.log(`${entry.passed ? "PASS" : "FAIL"} ${entry.name}${entry.detail ? ` ${entry.detail}` : ""}`);
for (const entry of limitations) console.log(`LIMIT ${entry.id} ${entry.detail}`);
if (!passed) process.exit(1);
console.log(`browser invoice evidence passed on ${wanted}`);
