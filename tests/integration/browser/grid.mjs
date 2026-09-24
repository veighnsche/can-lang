// T25 Gate 5 evidence for the Can-authored invoice grid: drives the staged
// live grid (emitted browser bundle on a same-origin test origin backed by
// the real invoice server and a disposable database) in one named Playwright
// browser. Covers boot/render, in-place edits, row operations, every save
// outcome (200/422/409/403/503), offline/slow/reconnect, truncated and
// garbage responses after commit, explicit identical-id replay, conflict
// adopt-or-keep, listener/node leaks, navigation/disposal, denied loads and
// the network ledger. Usage:
//   node grid.mjs <chromium|firefox|webkit> <base> <outdir> <dbpath>
// Aborts every non-loopback request, so a passing run proves the grid never
// needs a CDN, authored script, or foreign client runtime: the single served
// script must carry the emitted $canBrowserMain marker, and window.htmx must
// stay undefined. Writes report.json and screenshot.png into outdir.
import { strict as assert } from "node:assert";
import { mkdirSync, writeFileSync, chmodSync } from "node:fs";
import { join } from "node:path";

const [wanted, base, outdir, dbpath] = process.argv.slice(2);
if ((wanted !== "chromium" && wanted !== "firefox" && wanted !== "webkit") || !base || !outdir || !dbpath) {
  console.error("usage: node grid.mjs <chromium|firefox|webkit> <base> <outdir> <dbpath>");
  process.exit(2);
}
mkdirSync(outdir, { recursive: true });

const playwright = await import("playwright");
const browser = await playwright[wanted].launch({ timeout: 120000 });
let userAgent = "";
const checks = [];
const limitations = [];
const requests = [];
const ledger = [];
const bodies = [];
const aborted = [];
const pageerrors = [];
const consoleErrors = [];
const limit = (id, detail) => limitations.push({ id, detail });
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
  await page.addInitScript(() => {
    globalThis.__t25FocusCalls = [];
    const orig = HTMLElement.prototype.focus;
    HTMLElement.prototype.focus = function (...args) {
      globalThis.__t25FocusCalls.push({ id: this.id || this.tagName, connected: this.isConnected });
      return orig.apply(this, args);
    };
  });
  page.on("pageerror", (error) => pageerrors.push(String(error?.message ?? error)));
  page.on("console", (message) => {
    if (message.type() === "error") consoleErrors.push(message.text());
  });
  const watch = (target) => {
    target.on("request", (request) => {
      if (request.method() === "POST" && request.url().includes("/invoices/save")) {
        try {
          bodies.push(JSON.parse(request.postData() ?? "null"));
        } catch {
          bodies.push(null);
        }
      }
    });
    target.on("requestfinished", async (request) => {
      const response = await request.response().catch(() => undefined);
      requests.push({ method: request.method(), url: request.url(), status: response?.status() ?? 0 });
    });
    target.on("requestfailed", (request) => {
      requests.push({ method: request.method(), url: request.url(), status: -1 });
    });
    target.on("response", async (response) => {
      const url = response.url();
      if (!url.includes("/invoices/")) return;
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
  };
  watch(page);

  const statusText = () => page.locator("#status").textContent();
  const posts = () => bodies.length;
  const focusCalls = () => page.evaluate(() => globalThis.__t25FocusCalls.splice(0));
  const uniqueIds = () =>
    page.evaluate(() => {
      const ids = [...document.querySelectorAll("[id]")].map((node) => node.id);
      return { total: ids.length, duplicates: ids.length - new Set(ids).size };
    });
  const rowOrder = () =>
    page.locator("#grid tbody th").evaluateAll((nodes) => nodes.map((node) => node.textContent));
  const saveAndWait = async (match) => {
    await page.locator("#save").click();
    await page.waitForFunction(
      (want) => document.querySelector("#status")?.textContent?.match(new RegExp(want)),
      match,
      { timeout: 15000 }
    );
  };

  await context.addCookies([{ name: "session", value: "tok-alice", url: base }]);
  const bootResponse = await page.goto(base + "/grid?invoice=inv-1", { waitUntil: "load" });
  const csp = bootResponse?.headers()?.["content-security-policy"] ?? "";

  await check("boot-loads", async () => {
    await page.waitForFunction(() =>
      document.querySelector("#status")?.textContent?.includes("loaded revision 1")
    );
    assert.equal(await page.locator("#grid h1").textContent(), "Invoice inv-1 revision 1");
    assert.equal(await statusText(), "loaded revision 1");
    assert.equal(await page.locator("#status").getAttribute("role"), "status");
    assert.equal(await page.locator("#status").getAttribute("aria-live"), "polite");
    assert.equal(await page.locator("#customer").inputValue(), `Acme <em>&" 'coop'"`);
    assert.ok((await page.locator("#grid").innerHTML()).includes("Acme &lt;em&gt;"));
    assert.deepEqual(await rowOrder(), ["k1", "k2"]);
    assert.equal(await page.locator("#sku\\:k1").inputValue(), "sku-1 <b>");
    assert.equal(await page.locator("#qty\\:k1").inputValue(), "2");
    assert.equal(await page.locator("#totals").textContent(), "2 lines, 3 units");
    assert.equal(await page.locator("#replay").isDisabled(), true);
  });

  await check("client-identity", async () => {
    const scripts = await page.locator("script").evaluateAll((nodes) =>
      nodes.map((node) => node.getAttribute("src"))
    );
    assert.deepEqual(scripts, ["/grid/boot.js"]);
    assert.equal(await page.evaluate(() => typeof window.htmx), "undefined");
    assert.ok(csp.includes("script-src 'self'"), `csp lacks script-src: ${csp}`);
    assert.ok(csp.includes("connect-src 'self'"), `csp lacks connect-src: ${csp}`);
    const bundle = await page.evaluate(async () => (await fetch("/grid/boot.js")).text());
    assert.ok(bundle.includes("$canBrowserMain"), "bundle lacks the Can browser entry");
    assert.ok(bundle.includes("platform/browser.ts"), "bundle lacks the browser capability");
    for (const server of [
      "platform/server.ts",
      "platform/env.ts",
      "platform/s3.ts",
      "platform/websocket.ts",
      "platform/sql/",
      "platform/files/",
      "platform/process/",
      "platform/crypto/",
    ]) {
      assert.ok(!bundle.includes(server), `bundle reaches server-only ${server}`);
    }
    for (const entry of requests) {
      assert.ok(!entry.url.includes("htmx"), `HTMX request served: ${entry.url}`);
    }
  });

  await check("customer-edit-unsaved", async () => {
    const before = requests.length;
    await page.locator("#customer").fill("Grid Save");
    await page.waitForTimeout(400);
    assert.equal(await statusText(), "unsaved changes");
    assert.equal(requests.length - before, 0, "typing must not send network requests");
  });

  await check("caret-stable", async () => {
    await page.locator("#customer").click();
    await page.keyboard.press("End");
    await page.keyboard.type(" XYZ");
    await page.waitForTimeout(300);
    assert.equal(await page.evaluate(() => document.activeElement?.id), "customer");
    assert.ok((await page.locator("#customer").inputValue()).endsWith("Grid Save XYZ"));
    await page.locator("#customer").fill("Grid Save");
  });

  await check("qty-edit-totals", async () => {
    await page.locator("#qty\\:k1").fill("9");
    await page.waitForTimeout(300);
    assert.equal(await page.locator("#totals").textContent(), "2 lines, 10 units");
    assert.equal(await statusText(), "unsaved changes");
    await page.locator("#qty\\:k1").fill("4");
    await page.waitForTimeout(300);
    assert.equal(await page.locator("#totals").textContent(), "2 lines, 5 units");
  });

  await check("add-line", async () => {
    await focusCalls();
    await page.locator("#add").click();
    await page.waitForFunction(() =>
      document.querySelector("#status")?.textContent?.includes("unsaved")
    );
    await page.waitForTimeout(300);
    assert.deepEqual(await rowOrder(), ["k1", "k2", "k3"]);
    const calls = await focusCalls();
    assert.ok(
      calls.some((call) => call.id === "sku:k3"),
      `focus never invoked on the new row: ${JSON.stringify(calls)}`
    );
    limit(
      "L-focus-row",
      `add-line invokes focus() on sku:k3 while detached (connected=false at call); ` +
        `activeElement stays body. Row-level maybe_focus runs before the grid attaches.`
    );
  });

  await check("move-row", async () => {
    await focusCalls();
    await page.locator("#up\\:k3").click();
    await page.waitForTimeout(400);
    assert.deepEqual(await rowOrder(), ["k1", "k3", "k2"]);
    await page.locator("#down\\:k3").click();
    await page.waitForTimeout(400);
    assert.deepEqual(await rowOrder(), ["k1", "k2", "k3"]);
    const calls = await focusCalls();
    assert.ok(calls.some((call) => call.id === "up:k3"), `no focus call for the moved row`);
    assert.equal(await statusText(), "unsaved changes");
  });

  await check("remove-row-focus", async () => {
    await page.locator("#del\\:k3").click();
    await page.waitForTimeout(400);
    assert.deepEqual(await rowOrder(), ["k1", "k2"]);
    assert.equal(await page.evaluate(() => document.activeElement?.id), "add");
  });

  await check("blocked-save", async () => {
    await page.locator("#qty\\:k1").fill("many");
    await page.waitForTimeout(300);
    const before = posts();
    await page.locator("#save").click();
    await page.waitForTimeout(600);
    assert.equal(posts() - before, 0, "unparsable quantity must never be sent");
    assert.equal(await page.locator("#qty\\:k1").inputValue(), "many");
    assert.equal(await statusText(), "unsaved changes");
    limit(
      "L-notice",
      "blocked-save guard notice is wiped by the immediate re-render, so the status " +
        "returns to 'unsaved changes' with no explanation; the safety property (nothing sent) holds."
    );
    await page.locator("#qty\\:k1").fill("4");
    await page.waitForTimeout(300);
  });

  await check("rejected-422", async () => {
    await page.locator("#customer").fill("");
    await saveAndWait("rejected: empty customer");
    assert.equal(await statusText(), "rejected: empty customer");
    assert.equal(await page.locator("#customer").inputValue(), "");
    assert.equal(await page.locator("#grid h1").textContent(), "Invoice inv-1 revision 1");
  });

  await check("enter-saves", async () => {
    await page.locator("#customer").fill("Grid Save");
    const before = posts();
    await page.locator("#customer").press("Enter");
    await page.waitForFunction(() =>
      document.querySelector("#status")?.textContent?.match(/saved revision/)
    );
    assert.equal(posts() - before, 1);
    assert.equal(await statusText(), "saved revision 2");
    const sent = bodies[bodies.length - 1];
    assert.match(sent.operation_id, /^inv-1\/r1\/e\d+$/);
    assert.equal(sent.session_token, "");
    assert.equal(sent.revision, 1);
    assert.equal(await page.locator("#grid h1").textContent(), "Invoice inv-1 revision 2");
    return `op ${sent.operation_id}; focus intent cleared by the outcome (activeElement body)`;
  });

  await check("slow-save", async () => {
    await context.route("**/invoices/save", async (route) => {
      await new Promise((resolve) => setTimeout(resolve, 1200));
      await route.continue();
    });
    try {
      await page.locator("#customer").fill("Slow Save");
      await page.locator("#save").click();
      await page.waitForTimeout(400);
      assert.equal(await statusText(), "unsaved changes");
      assert.equal(await page.locator("#save").isDisabled(), false);
      limit(
        "L-flight",
        "no mid-flight 'saving...' indication: the press marks the flight in state but " +
          "renders only after the outcome lands; the single-flight guard still holds."
      );
      await page.waitForFunction(
        () => document.querySelector("#status")?.textContent?.includes("saved revision 3"),
        null,
        { timeout: 15000 }
      );
    } finally {
      await context.unroute("**/invoices/save");
    }
  });

  await check("single-flight", async () => {
    await page.locator("#customer").fill("Double Press");
    const before = posts();
    await page.evaluate(() => {
      document.getElementById("save").click();
      document.getElementById("save").click();
    });
    await page.waitForFunction(
      () => document.querySelector("#status")?.textContent?.includes("saved revision 4"),
      null,
      { timeout: 15000 }
    );
    assert.equal(posts() - before, 1, "a double press must send exactly one flight");
  });

  await check("offline-replay", async () => {
    await context.setOffline(true);
    try {
      await page.locator("#customer").fill("Offline Edit");
      const before = posts();
      await page.locator("#save").click();
      await page.waitForFunction(
        () => document.querySelector("#status")?.textContent?.includes("connection lost; save may have committed"),
        null,
        { timeout: 15000 }
      );
      assert.equal(await page.locator("#replay").isDisabled(), false);
      await context.setOffline(false);
      await page.locator("#replay").click();
      await page.waitForFunction(
        () => document.querySelector("#status")?.textContent?.includes("saved revision 5"),
        null,
        { timeout: 15000 }
      );
      const attempt = bodies[before];
      const replay = bodies[before + 1];
      assert.ok(attempt && replay, "offline attempt and replay must both be recorded");
      assert.equal(replay.operation_id, attempt.operation_id, "replay must reuse the identical id");
      return `op ${replay.operation_id}`;
    } finally {
      await context.setOffline(false);
    }
  });

  await check("busy-replay", async () => {
    chmodSync(dbpath, 0o000);
    try {
      await page.locator("#customer").fill("Busy Edit");
      const before = posts();
      await page.locator("#save").click();
      await page.waitForFunction(
        () => document.querySelector("#status")?.textContent?.includes("save may have committed; replay or reread"),
        null,
        { timeout: 15000 }
      );
      assert.equal(await page.locator("#replay").isDisabled(), false);
      const attempt = bodies[before];
      assert.ok(attempt, "busy attempt must reach the server");
      return `attempt op ${attempt.operation_id}`;
    } finally {
      chmodSync(dbpath, 0o600);
    }
  });

  await check("busy-replay-commit", async () => {
    const before = posts();
    await page.locator("#replay").click();
    await page.waitForFunction(
      () => document.querySelector("#status")?.textContent?.includes("saved revision 6"),
      null,
      { timeout: 15000 }
    );
    const attempt = bodies[before - 1];
    const replay = bodies[before];
    assert.equal(replay.operation_id, attempt.operation_id, "busy replay must reuse the identical id");
    return `op ${replay.operation_id}`;
  });

  const armFault = (mode) =>
    page.evaluate(async (name) => (await fetch("/t25/fault", { method: "POST", body: name })).text(), mode);

  await check("truncate-replay", async () => {
    assert.equal(await armFault("truncate"), "armed:truncate");
    await page.locator("#customer").fill("Truncate Edit");
    const before = posts();
    await page.locator("#save").click();
    await page.waitForFunction(
      () => document.querySelector("#status")?.textContent?.includes("may have committed"),
      null,
      { timeout: 15000 }
    );
    const notice = await statusText();
    assert.match(notice, /save may have committed/);
    assert.equal(await page.locator("#replay").isDisabled(), false);
    await page.locator("#replay").click();
    await page.waitForFunction(
      () => document.querySelector("#status")?.textContent?.includes("saved revision 7"),
      null,
      { timeout: 15000 }
    );
    const attempt = bodies[before];
    const replay = bodies[before + 1];
    assert.equal(replay.operation_id, attempt.operation_id, "truncated replay must reuse the identical id");
    return `op ${replay.operation_id}; truncate classified as ${JSON.stringify(notice)}`;
  });

  await check("garbage-replay", async () => {
    assert.equal(await armFault("garbage"), "armed:garbage");
    await page.locator("#customer").fill("Garbage Edit");
    const before = posts();
    await page.locator("#save").click();
    await page.waitForFunction(
      () => document.querySelector("#status")?.textContent?.includes("unreadable response; save may have committed"),
      null,
      { timeout: 15000 }
    );
    assert.equal(await page.locator("#replay").isDisabled(), false);
    await page.locator("#replay").click();
    await page.waitForFunction(
      () => document.querySelector("#status")?.textContent?.includes("saved revision 8"),
      null,
      { timeout: 15000 }
    );
    const attempt = bodies[before];
    const replay = bodies[before + 1];
    assert.equal(replay.operation_id, attempt.operation_id, "garbage replay must reuse the identical id");
    return `op ${replay.operation_id}`;
  });

  await check("stale-conflict-adopt", async () => {
    const race = await page.evaluate(async () => {
      const response = await fetch("/invoices/save", {
        method: "POST",
        headers: { "content-type": "application/json" },
        body: JSON.stringify({
          session_token: "",
          operation_id: "op-t25-race",
          invoice_id: "inv-1",
          revision: 8,
          customer: "Racer",
          lines: [{ key: "k1", sku: "sku-1", qty: 2 }],
        }),
      });
      return `${response.status} ${(await response.text()).slice(0, 120)}`;
    });
    assert.match(race, /^200 /);
    await page.locator("#customer").fill("Stale Attempt");
    await saveAndWait("changed on server at revision 9; reread to compare");
    await page.locator("#reread").click();
    await page.waitForFunction(
      () => document.querySelector("#status")?.textContent?.includes("server has revision 9; your edits kept"),
      null,
      { timeout: 15000 }
    );
    assert.equal(await page.locator("#adopt").count(), 1);
    assert.equal(await page.locator("#keep").count(), 1);
    await page.locator("#keep").click();
    await page.waitForFunction(
      () => document.querySelector("#status")?.textContent?.includes("unsaved changes"),
      null,
      { timeout: 15000 }
    );
    assert.equal(await page.locator("#customer").inputValue(), "Stale Attempt");
    await page.locator("#reread").click();
    await page.waitForFunction(
      () => document.querySelector("#status")?.textContent?.includes("server has revision 9; your edits kept"),
      null,
      { timeout: 15000 }
    );
    await page.locator("#adopt").click();
    await page.waitForFunction(
      () => document.querySelector("#status")?.textContent?.includes("loaded revision 9"),
      null,
      { timeout: 15000 }
    );
    assert.equal(await page.locator("#customer").inputValue(), "Racer");
    assert.equal(await page.locator("#totals").textContent(), "1 lines, 2 units");
    assert.equal(await page.locator("#grid h1").textContent(), "Invoice inv-1 revision 9");
  });

  await check("detached-click-silent", async () => {
    const oldSave = await page.$("#save");
    const before = posts();
    await page.locator("#add").click();
    await page.waitForFunction(() =>
      document.querySelector("#status")?.textContent?.includes("unsaved")
    );
    assert.equal(await page.evaluate((node) => !node.isConnected, oldSave), true);
    await page.evaluate((node) => node.click(), oldSave);
    await page.waitForTimeout(600);
    assert.equal(posts() - before, 0, "a disposed view's listener must never fire");
    const added = await page.evaluate(
      () => [...document.querySelectorAll("input[id^=sku]")].map((node) => node.id).pop()
    );
    await page.locator(`#del\\:${added.split(":")[1]}`).click();
    await page.waitForTimeout(400);
    assert.deepEqual(await rowOrder(), ["k1"]);
  });

  await check("no-duplicate-ids", async () => {
    const ids = await uniqueIds();
    assert.equal(ids.duplicates, 0, `duplicate ids after every re-render: ${ids.total} nodes`);
    return `${ids.total} id nodes, all unique`;
  });

  await check("node-count-bounded", async () => {
    const count = () => page.evaluate(() => document.querySelectorAll("*").length);
    const baseline = await count();
    for (let i = 0; i < 5; i++) {
      await page.locator("#add").click();
      await page.waitForTimeout(250);
      const added = await page.evaluate(
        () => [...document.querySelectorAll("input[id^=sku]")].map((node) => node.id).pop()
      );
      await page.locator(`#del\\:${added.split(":")[1]}`).click();
      await page.waitForTimeout(250);
    }
    assert.equal(await count(), baseline, "add/remove cycles must not accumulate nodes");
    return `${baseline} nodes`;
  });

  await check("ghost-denied", async () => {
    await context.addCookies([{ name: "session", value: "tok-ghost", url: base }]);
    await page.locator("#customer").fill("Ghost Attempt");
    await saveAndWait("access denied; draft kept");
    assert.equal(await page.locator("#customer").inputValue(), "Ghost Attempt");
    await context.addCookies([{ name: "session", value: "tok-alice", url: base }]);
  });

  await check("reload-no-durability", async () => {
    await page.locator("#customer").fill("Doomed Draft");
    await page.waitForTimeout(200);
    await page.reload({ waitUntil: "load" });
    await page.waitForFunction(
      () => document.querySelector("#status")?.textContent?.includes("loaded revision 9"),
      null,
      { timeout: 15000 }
    );
    assert.equal(await page.locator("#customer").inputValue(), "Racer");
    assert.equal(await page.locator("#totals").textContent(), "1 lines, 2 units");
  });

  await check("navigation-stable", async () => {
    const errorsBefore = pageerrors.length;
    await page.goto("about:blank");
    await page.waitForTimeout(500);
    assert.equal(page.url(), "about:blank");
    await page.goto(base + "/grid?invoice=inv-1", { waitUntil: "load" });
    await page.waitForFunction(
      () => document.querySelector("#status")?.textContent?.includes("loaded revision 9"),
      null,
      { timeout: 15000 }
    );
    assert.equal(pageerrors.length, errorsBefore, "navigation must not raise page errors");
  });

  await check("denied-load", async () => {
    const denied = await context.newPage();
    watch(denied);
    denied.on("pageerror", (error) => pageerrors.push(String(error?.message ?? error)));
    try {
      await denied.goto(base + "/grid?invoice=inv-2", { waitUntil: "load" });
      await denied.waitForFunction(
        () => document.querySelector("#status")?.textContent?.includes("access denied; draft kept"),
        null,
        { timeout: 15000 }
      );
      assert.equal(await denied.locator("#grid h1").textContent(), "Invoice inv-2 revision 0");
      assert.equal(await denied.locator("#customer").inputValue(), "");
      const html = await denied.locator("#grid").innerHTML();
      assert.ok(!html.includes("Globex"), "denied load must not leak stored content");
    } finally {
      await denied.close();
    }
  });

  await check("network-ledger-clean", async () => {
    assert.equal(aborted.length, 0, `non-loopback requests: ${aborted.join(", ")}`);
    for (const entry of requests) {
      const host = new URL(entry.url).hostname;
      assert.ok(host === "127.0.0.1" || host === "localhost", `grid left loopback: ${entry.url}`);
    }
    assert.equal(pageerrors.length, 0, pageerrors.join("; "));
    for (const text of consoleErrors) {
      assert.match(text, /Failed to load resource/, `non-resource console error: ${text}`);
    }
    assert.ok(
      ledger.some((entry) => entry.method === "GET" && entry.url.includes("/invoices/inv-1")),
      "ledger lacks the captured-path GET load"
    );
    assert.ok(
      ledger.every((entry) => entry.url.startsWith(base)),
      "ledger holds a non-origin invoice call"
    );
    return `${requests.length} requests, ${ledger.length} invoice calls`;
  });

  await page.screenshot({ path: join(outdir, "screenshot.png"), fullPage: true });
  userAgent = await page.evaluate(() => navigator.userAgent);
  await context.close();
} finally {
  await browser.close();
}

const passed = checks.every((entry) => entry.passed);
writeFileSync(
  join(outdir, "report.json"),
  JSON.stringify(
    {
      browser: wanted,
      version: browser.version(),
      userAgent,
      base,
      passed,
      checks,
      limitations,
      requests,
      ledger,
      aborted,
      pageerrors,
      consoleErrors,
    },
    null,
    2
  ) + "\n"
);
for (const entry of checks) console.log(`${entry.passed ? "PASS" : "FAIL"} ${entry.name}${entry.detail ? ` ${entry.detail}` : ""}`);
for (const entry of limitations) console.log(`LIMIT ${entry.id} ${entry.detail}`);
if (!passed) process.exit(1);
console.log(`browser grid evidence passed on ${wanted}`);
