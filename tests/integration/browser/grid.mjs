// UP23 Gate 5 evidence for the Can-authored invoice grid: drives the
// served paired application (compiler-built grid plus real invoice
// server and a disposable database per browser) in one named
// Playwright browser through boot, query edges, edits, rows, every
// save outcome, offline/slow/reconnect, truncated and garbage
// responses after commit, identical-id replay, conflict
// adopt-or-keep, disposal, navigation and denied loads. Usage:
//   node grid.mjs <chromium|webkit> <base> <outdir> <dbpath> <script-url>
// Aborts every non-loopback request, so a passing run proves the grid
// never needs a CDN, authored script, or foreign client runtime: the
// single served script must equal the report-selected paired URL, and
// window.htmx must stay undefined. Writes report.json and
// screenshot.png into outdir.
import { strict as assert } from "node:assert";
import { mkdirSync, writeFileSync, chmodSync } from "node:fs";
import { join, dirname } from "node:path";

const [wanted, base, outdir, dbpath, scriptUrl] = process.argv.slice(2);
if (
  (wanted !== "chromium" && wanted !== "webkit") ||
  !base ||
  !outdir ||
  !dbpath ||
  !scriptUrl?.startsWith("/__can/assets/")
) {
  console.error("usage: node grid.mjs <chromium|webkit> <base> <outdir> <dbpath> <script-url>");
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
const checkLater = [];
const check = (name, fn) => checkLater[0](name, fn);

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
  checkLater[0] = (name, fn) =>
    Promise.resolve()
      .then(fn)
      .then((detail = "") => checks.push({ name, passed: true, detail }))
      .catch(async (error) => {
        let dom = "";
        try {
          dom = await page.evaluate(() => {
            const status = document.querySelector("#status")?.textContent ?? "<no status>";
            const head = document.querySelector("#grid h1")?.textContent ?? "<no h1>";
            const grids = document.querySelectorAll("#grid").length;
            return ` [status=${JSON.stringify(status)} h1=${JSON.stringify(head)} grids=${grids}]`;
          });
        } catch {}
        checks.push({ name, passed: false, detail: String(error?.message ?? error) + dom });
      });
  page.on("pageerror", (error) => pageerrors.push(String(error?.message ?? error)));
  page.on("console", (message) => {
    if (message.type() === "error") consoleErrors.push(message.text());
  });
  const watch = (target) => {
    target.on("request", (request) => {
      if (request.method() === "POST" && request.url().includes("/api/tenants/")) {
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
  };
  watch(page);

  const statusText = () => page.locator("#status").textContent();
  const posts = () => bodies.length;
  const uniqueIds = () =>
    page.evaluate(() => {
      const ids = [...document.querySelectorAll("[id]")].map((node) => node.id);
      return { total: ids.length, duplicates: ids.length - new Set(ids).size };
    });
  const rowOrder = () =>
    page.locator("#grid tbody th").evaluateAll((nodes) => nodes.map((node) => node.textContent));
  const currentRev = async () =>
    parseInt(
      await page.evaluate(() => document.querySelector("#grid h1")?.textContent?.match(/revision (\d+)/)?.[1]),
      10
    );
  // Values are set with a synthetic input event for fast
  // deterministic fills; only Enter dispatches the grid's save
  // handler, so real keys type normally. The real-keystrokes leg
  // proves typing, Enter-to-save and the clean guard with the
  // physical keyboard path.
  const fill = async (selector, value) => {
    await page.locator(selector).evaluate(
      (node, text) => {
        node.focus();
        node.value = text;
        node.dispatchEvent(new Event("input", { bubbles: true }));
      },
      value
    );
    await Promise.race([
      page
        .waitForFunction(() => document.querySelector("#status")?.textContent?.includes("unsaved changes"), null, {
          timeout: 5000,
        })
        .catch(() => undefined),
      page.waitForTimeout(700),
    ]);
    await page.waitForTimeout(200);
    assert.equal(await page.locator(selector).inputValue(), value);
  };
  const saveAndWait = async (match) => {
    await page.locator("#save").click();
    await page.waitForFunction((want) => document.querySelector("#status")?.textContent?.match(new RegExp(want)), match, {
      timeout: 15000,
    });
  };

  await context.addCookies([{ name: "session", value: "tok-alice", url: base }]);
  const bootResponse = await page.goto(base + "/invoice-grid?tenant=1&invoice=7", { waitUntil: "load" });
  const csp = bootResponse?.headers()?.["content-security-policy"] ?? "";

  await check("boot-loads", async () => {
    await page.waitForFunction(() => document.querySelector("#status")?.textContent?.includes("loaded revision 1"));
    assert.equal(await page.locator("#grid h1").textContent(), "Tenant 1 invoice 7 revision 1");
    assert.equal(await statusText(), "loaded revision 1");
    assert.equal(await page.locator("#status").getAttribute("role"), "status");
    assert.equal(await page.locator("#status").getAttribute("aria-live"), "polite");
    assert.deepEqual(await rowOrder(), ["k1", "k2"]);
    assert.equal(await page.locator("#id\\:k1").inputValue(), "sku-1 <b>");
    assert.equal(await page.locator("#qty\\:k1").inputValue(), "2");
    assert.equal(await page.locator("#price\\:k1").inputValue(), "19.99");
    assert.ok((await page.locator("#grid").innerHTML()).includes("sku-1 &lt;b&gt;"));
    assert.equal(await page.locator("#totals").textContent(), "2 lines, 44.98 total");
    assert.equal(await page.locator("#replay").isDisabled(), true);
  });

  await check("client-identity", async () => {
    const scripts = await page.locator("script").evaluateAll((nodes) => nodes.map((node) => node.getAttribute("src")));
    assert.deepEqual(scripts, [scriptUrl]);
    assert.equal(await page.evaluate(() => typeof window.htmx), "undefined");
    assert.ok(csp.includes("script-src 'self'"), `csp lacks script-src: ${csp}`);
    assert.ok(csp.includes("connect-src 'self'"), `csp lacks connect-src: ${csp}`);
    const bundle = await page.evaluate(async (src) => (await fetch(src)).text(), scriptUrl);
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

  const bootNotice = async (query, notice) => {
    const probe = await context.newPage();
    watch(probe);
    probe.on("pageerror", (error) => pageerrors.push(String(error?.message ?? error)));
    try {
      const calls = ledger.length;
      await probe.goto(base + query, { waitUntil: "load" });
      await probe.waitForSelector("#status", { timeout: 8000 });
      assert.equal(await probe.locator("#status").textContent(), notice);
      assert.equal(await probe.locator("#status").getAttribute("role"), "status");
      assert.equal(ledger.length - calls, 0, "boot notice must not call the API");
    } finally {
      await probe.close();
    }
  };

  await check("query-edges", async () => {
    await bootNotice("/invoice-grid", "Choose an invoice");
    await bootNotice("/invoice-grid?tenant=1", "Choose an invoice");
    await bootNotice("/invoice-grid?tenant=x&invoice=7", "Invalid invoice link");
    await bootNotice("/invoice-grid?tenant=1&invoice=x", "Invalid invoice link");
    await bootNotice("/invoice-grid?tenant=1&tenant=2&invoice=7", "Invalid invoice link");
    await bootNotice("/invoice-grid?tenant=1&invoice=%E0%A4%A", "Invalid invoice link");
    return "missing/invalid/duplicate/malformed keys notice without API calls";
  });

  await check("edit-totals", async () => {
    const before = requests.length;
    await fill("#qty\\:k1", "4");
    assert.equal(await statusText(), "unsaved changes");
    assert.equal(await page.locator("#totals").textContent(), "2 lines, 84.96 total");
    assert.equal(requests.length - before, 0, "typing must not send network requests");
  });

  await check("money-exact", async () => {
    // Input folds refresh the totals line in place; per-row amount
    // cells refresh on the next re-render. Both stay exact.
    await fill("#price\\:k2", "0.05");
    assert.equal(await page.locator("#totals").textContent(), "2 lines, 80.01 total");
    const rev = await currentRev();
    await saveAndWait(`saved revision ${rev + 1}`);
    const amounts = await page
      .locator("#grid tbody tr")
      .evaluateAll((rows) => rows.map((row) => row.children[4].textContent));
    assert.deepEqual(amounts, ["79.96", "0.05"]);
    await fill("#price\\:k2", "5.00");
    await fill("#qty\\:k1", "2");
    await saveAndWait(`saved revision ${rev + 2}`);
    assert.equal(await page.locator("#totals").textContent(), "2 lines, 44.98 total");
    return "live totals and rendered amounts exact to the minor unit";
  });

  await check("add-line-focus", async () => {
    await page.locator("#add").click();
    await page.waitForFunction(() => document.querySelector("#status")?.textContent?.includes("unsaved"));
    await page.waitForTimeout(300);
    assert.deepEqual(await rowOrder(), ["k1", "k2", "k3"]);
    assert.equal(await page.evaluate(() => document.activeElement?.id), "id:k3");
    return "focus lands on id:k3 after attach";
  });

  await check("move-row-focus", async () => {
    await page.locator("#up\\:k3").click();
    await page.waitForTimeout(400);
    assert.deepEqual(await rowOrder(), ["k1", "k3", "k2"]);
    assert.equal(await page.evaluate(() => document.activeElement?.id), "up:k3");
    await page.locator("#down\\:k3").click();
    await page.waitForTimeout(400);
    assert.deepEqual(await rowOrder(), ["k1", "k2", "k3"]);
    assert.equal(await page.evaluate(() => document.activeElement?.id), "down:k3");
    assert.equal(await statusText(), "unsaved changes");
  });

  await check("remove-row-focus", async () => {
    await page.locator("#del\\:k3").click();
    await page.waitForTimeout(400);
    assert.deepEqual(await rowOrder(), ["k1", "k2"]);
    assert.equal(await page.evaluate(() => document.activeElement?.id), "add");
  });

  await check("blocked-save", async () => {
    await fill("#qty\\:k1", "many");
    const before = posts();
    await page.locator("#save").click();
    await page.waitForFunction(() =>
      document.querySelector("#status")?.textContent?.includes("line k1: bad quantity")
    );
    await page.waitForTimeout(400);
    assert.equal(posts() - before, 0, "unparsable quantity must never be sent");
    assert.equal(await page.locator("#qty\\:k1").inputValue(), "many");
    assert.equal(await statusText(), "line k1: bad quantity");
    await fill("#qty\\:k1", "2");
    return "notice persists in state across the re-render";
  });

  await check("rejected-422", async () => {
    await fill("#id\\:k1", "");
    const rev = await currentRev();
    await saveAndWait("empty line id");
    assert.equal(await statusText(), "id: empty line id");
    assert.equal(await page.locator("#id\\:k1").inputValue(), "");
    assert.equal(await currentRev(), rev, "rejected save must not move the revision");
    assert.equal(await page.locator("#replay").isDisabled(), true);
  });

  await check("enter-saves", async () => {
    await fill("#id\\:k1", "sku-9");
    const rev = await currentRev();
    const before = posts();
    await page.evaluate(() => {
      window.__prevented = [];
      document.getElementById("id:k1").addEventListener("keydown", (event) => {
        window.__prevented.push(event.defaultPrevented);
      });
    });
    await page.locator("#id\\:k1").press("Enter");
    await page.waitForFunction((want) => document.querySelector("#status")?.textContent?.includes(want), `saved revision ${rev + 1}`, {
      timeout: 15000,
    });
    assert.equal(posts() - before, 1, "Enter must dispatch exactly one save");
    assert.deepEqual(await page.evaluate(() => window.__prevented), [true]);
    assert.equal(await statusText(), `saved revision ${rev + 1}`);
    const sent = bodies[bodies.length - 1];
    assert.match(sent.operation_id, new RegExp(`^t1/i7/r${rev}/e\\d+$`));
    assert.equal(sent.revision, String(rev));
    assert.equal(await page.evaluate(() => document.activeElement?.tagName), "BODY");
    return `op ${sent.operation_id}; Enter prevented synchronously`;
  });

  await check("real-keystrokes", async () => {
    // Only Enter dispatches the save handler; every other keydown
    // returns without touching state or the tree, so real typing
    // lands in the DOM, folds into state, and reaches the wire.
    const rev = await currentRev();
    const before = posts();
    await page.locator("#qty\\:k1").click({ clickCount: 3 });
    await page.locator("#qty\\:k1").press("5");
    await page.waitForFunction(() => document.querySelector("#status")?.textContent?.includes("unsaved changes"), null, {
      timeout: 8000,
    });
    assert.equal(posts() - before, 0, "typing must not dispatch a save");
    assert.equal(await page.locator("#qty\\:k1").inputValue(), "5", "typed char must survive in the DOM");
    assert.equal(await statusText(), "unsaved changes");
    assert.equal(await page.locator("#totals").textContent(), "2 lines, 104.95 total");
    await saveAndWait(`saved revision ${rev + 1}`);
    const proof = bodies[bodies.length - 1].lines.find((line) => line.key === "k1");
    assert.equal(proof.quantity, "5", "typed char must reach the wire");
    assert.equal(await page.locator("#qty\\:k1").inputValue(), "5");
    // Mid-edit Enter dispatches exactly once with the post-keystroke
    // draft; Enter on a clean draft hits the guard and sends nothing.
    await fill("#qty\\:k1", "7");
    const mid = posts();
    await page.locator("#qty\\:k1").click({ clickCount: 3 });
    await page.locator("#qty\\:k1").press("8");
    await page.waitForFunction(() => document.querySelector("#status")?.textContent?.includes("unsaved changes"), null, {
      timeout: 8000,
    });
    assert.equal(await page.locator("#qty\\:k1").inputValue(), "8");
    await page.locator("#qty\\:k1").press("Enter");
    await page.waitForFunction((want) => document.querySelector("#status")?.textContent?.includes(want), `saved revision ${rev + 2}`, {
      timeout: 15000,
    });
    assert.equal(posts() - mid, 1, "Enter must dispatch exactly one save");
    const sent = bodies[bodies.length - 1].lines.find((line) => line.key === "k1");
    assert.equal(sent.quantity, "8", "Enter dispatch carries the post-keystroke draft");
    assert.equal(await statusText(), `saved revision ${rev + 2}`);
    await page.locator("#qty\\:k1").press("Enter");
    await page.waitForFunction(() => document.querySelector("#status")?.textContent?.includes("no changes to save"), null, {
      timeout: 8000,
    });
    assert.equal(posts() - mid, 1, "clean-guard Enter must not send");
    await fill("#qty\\:k1", "2");
    await saveAndWait(`saved revision ${rev + 3}`);
    return `typed chars survive to the wire; Enter sends the typed draft; restored at rev ${rev + 3}`;
  });

  await check("slow-save-pending", async () => {
    // The save flight paints "saving..." when it starts, holds the
    // single-flight guard while outstanding, keeps a mid-flight edit
    // (an interleaved owner after await) through the outcome fold,
    // and clears the notice when the outcome render lands.
    await context.route("**/api/tenants/1/invoices/7", async (route) => {
      if (route.request().method() !== "POST") return route.continue();
      await new Promise((resolve) => setTimeout(resolve, 2500));
      await route.continue();
    });
    try {
      const rev = await currentRev();
      const before = posts();
      await fill("#price\\:k1", "20.00");
      const sent = page.waitForRequest(
        (request) => request.url().includes("/api/tenants/") && request.method() === "POST",
        { timeout: 15000 }
      );
      await page.locator("#save").click();
      await sent;
      await page.waitForFunction(() => document.querySelector("#status")?.textContent === "saving...", null, {
        timeout: 5000,
      });
      assert.equal(await statusText(), "saving...", "pending indication must paint mid-flight");
      assert.equal(posts() - before, 1, "one press must send exactly one flight");
      await fill("#qty\\:k1", "3");
      assert.equal(await statusText(), "saving...", "mid-flight edit must keep the pending notice");
      await page.waitForFunction(
        (r) => {
          const text = document.querySelector("#status")?.textContent ?? "";
          const head = document.querySelector("#grid h1")?.textContent ?? "";
          return text.includes(`saved revision ${r}`) || (text.includes("unsaved changes") && head.includes(`revision ${r}`));
        },
        rev + 1,
        { timeout: 15000 }
      );
      assert.equal(await page.locator("#qty\\:k1").inputValue(), "3", "mid-flight edit must survive");
      assert.equal(await page.locator("#price\\:k1").inputValue(), "20.00");
      assert.equal(await page.locator("#grid").count(), 1, "press plus outcome must leave one grid");
      await saveAndWait(`saved revision ${rev + 2}`);
      return "saving paints mid-flight and clears on the outcome; mid-flight edit preserved";
    } finally {
      await context.unroute("**/api/tenants/1/invoices/7");
    }
  });

  await check("midflight-press-single-grid", async () => {
    // An ignored mid-flight press returns without rendering, so the
    // in-flight outcome render stays the single next render and
    // exactly one grid survives. Proved on a throwaway page whose
    // POST is fulfilled synthetically (the shared server never
    // commits), closed after.
    const probe = await context.newPage();
    watch(probe);
    probe.on("pageerror", (error) => pageerrors.push(String(error?.message ?? error)));
    try {
      await context.route("**/api/tenants/1/invoices/7", async (route) => {
        if (route.request().method() !== "POST") return route.continue();
        await new Promise((resolve) => setTimeout(resolve, 800));
        await route.fulfill({
          status: 200,
          contentType: "application/json",
          body: '{"case":"invoice_contract::grid_saved","value":{"current":{"revision":"99","lines":[{"key":"k1","id":"sku-9","quantity":"9","price":"20.00"}],"total_minor_units":18000}}}',
        });
      });
      try {
        await probe.goto(base + "/invoice-grid?tenant=1&invoice=7", { waitUntil: "load" });
        await probe.waitForFunction(() => document.querySelector("#status")?.textContent?.match(/loaded revision \d+/), null, {
          timeout: 15000,
        });
        await probe.locator("#qty\\:k1").evaluate((node) => {
          node.value = "9";
          node.dispatchEvent(new Event("input", { bubbles: true }));
        });
        await probe.waitForFunction(() => document.querySelector("#status")?.textContent?.includes("unsaved changes"), null, {
          timeout: 8000,
        });
        const before = posts();
        const sent = probe.waitForRequest(
          (request) => request.url().includes("/api/tenants/") && request.method() === "POST",
          { timeout: 15000 }
        );
        await probe.locator("#save").click();
        await sent;
        await probe.waitForFunction(() => document.querySelector("#status")?.textContent === "saving...", null, {
          timeout: 5000,
        });
        await probe.locator("#save").click();
        await probe.waitForTimeout(400);
        assert.equal(posts() - before, 1, "ignored mid-flight press must not send");
        assert.equal(await probe.locator("#grid").count(), 1, "ignored press must not render a second tree");
        await probe.waitForFunction(() => document.querySelector("#status")?.textContent?.includes("saved revision 99"), null, {
          timeout: 15000,
        });
        assert.equal(posts() - before, 1, "press plus outcome must send exactly one flight");
        assert.equal(await probe.locator("#grid").count(), 1, "press plus outcome must leave one grid");
        assert.equal(await probe.locator("#status").textContent(), "saved revision 99");
        return "ignored press renders nothing; outcome leaves one grid at rev 99";
      } finally {
        await context.unroute("**/api/tenants/1/invoices/7");
      }
    } finally {
      await probe.close();
    }
  });

  await check("single-flight", async () => {
    const rev = await currentRev();
    await fill("#price\\:k1", "21.00");
    const before = posts();
    await page.evaluate(() => {
      document.getElementById("save").click();
      document.getElementById("save").click();
    });
    await page.waitForFunction((want) => document.querySelector("#status")?.textContent?.includes(want), `saved revision ${rev + 1}`, {
      timeout: 15000,
    });
    assert.equal(posts() - before, 1, "a double press must send exactly one flight");
  });

  await check("offline-replay", async () => {
    await context.setOffline(true);
    try {
      await fill("#price\\:k1", "22.00");
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
      const rev = await currentRev();
      await page.waitForFunction((want) => document.querySelector("#status")?.textContent?.includes(want), `saved revision ${rev + 1}`, {
        timeout: 15000,
      });
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
    chmodSync(dirname(dbpath), 0o555);
    try {
      await fill("#price\\:k1", "23.00");
      const before = posts();
      await page.locator("#save").click();
      await page.waitForFunction(
        () => document.querySelector("#status")?.textContent?.includes("save may have committed; replay or reread"),
        null,
        { timeout: 15000 }
      );
      assert.equal(await page.locator("#replay").isDisabled(), false);
      assert.ok(bodies[before], "busy attempt must reach the server");
      return `attempt op ${bodies[before].operation_id}`;
    } finally {
      chmodSync(dirname(dbpath), 0o700);
    }
  });

  await check("busy-replay-commit", async () => {
    const rev = await currentRev();
    const before = posts();
    await page.locator("#replay").click();
    await page.waitForFunction((want) => document.querySelector("#status")?.textContent?.includes(want), `saved revision ${rev + 1}`, {
      timeout: 15000,
    });
    assert.equal(bodies[before].operation_id, bodies[before - 1].operation_id, "busy replay must reuse the identical id");
    return `op ${bodies[before].operation_id}`;
  });

  // After-commit corruption: forward the request so the server
  // commits, then corrupt only the bytes the browser reads. Handler
  // faults surface as check failures, never uncaught crashes.
  let faultError = null;
  const corruptNext = async (mode) => {
    faultError = null;
    await context.route("**/api/tenants/1/invoices/7", async (route) => {
      try {
        if (route.request().method() !== "POST") return route.continue();
        const upstream = await route.fetch();
        assert.equal(upstream.status(), 200, "fault leg expected an upstream commit");
        const full = await upstream.text();
        const body = mode === "truncate" ? full.slice(0, 10) : '{"case":';
        await route.fulfill({ status: 200, contentType: "application/json", body });
      } catch (error) {
        faultError = error;
        await route.abort("failed").catch(() => undefined);
      }
    });
  };

  const corruptedReplay = async (mode, price) => {
    await corruptNext(mode);
    try {
      await fill("#price\\:k1", price);
      const before = posts();
      await page.locator("#save").click();
      await page.waitForTimeout(800);
      if (faultError) throw faultError;
      await page.waitForFunction(
        () => document.querySelector("#status")?.textContent?.includes("unreadable response; save may have committed"),
        null,
        { timeout: 15000 }
      );
      assert.equal(await page.locator("#replay").isDisabled(), false);
      await context.unroute("**/api/tenants/1/invoices/7");
      const rev = await currentRev();
      await page.locator("#replay").click();
      await page.waitForFunction((want) => document.querySelector("#status")?.textContent?.includes(want), `saved revision ${rev + 1}`, {
        timeout: 15000,
      });
      assert.equal(bodies[before + 1].operation_id, bodies[before].operation_id, `${mode} replay must reuse the identical id`);
      return `op ${bodies[before + 1].operation_id}`;
    } finally {
      await context.unroute("**/api/tenants/1/invoices/7");
    }
  };

  await check("truncate-replay", async () => corruptedReplay("truncate", "24.00"));
  await check("garbage-replay", async () => corruptedReplay("garbage", "25.00"));

  await check("stale-conflict", async () => {
    const rev = await currentRev();
    const race = await page.evaluate(async (r) => {
      const response = await fetch(`/api/tenants/1/invoices/7`, {
        method: "POST",
        headers: { "content-type": "application/json" },
        body: JSON.stringify({
          operation_id: "op-up23-race",
          revision: String(r),
          lines: [{ key: "k1", id: "sku-1", quantity: "2", price: "19.99" }],
        }),
      });
      return `${response.status} ${(await response.text()).slice(0, 120)}`;
    }, rev);
    assert.match(race, /^200 /);
    await fill("#price\\:k1", "26.00");
    await saveAndWait("stale revision; reread to compare");
    assert.equal(await page.locator("#price\\:k1").inputValue(), "26.00");
    assert.equal(await currentRev(), rev, "conflict must not move the revision");
  });

  await check("conflict-keep-adopt", async () => {
    const rev = await currentRev();
    await page.locator("#reread").click();
    await page.waitForFunction(
      (want) => document.querySelector("#status")?.textContent?.includes(want),
      `server has revision ${rev + 1}; your edits kept`,
      { timeout: 15000 }
    );
    assert.equal(await page.locator("#adopt").count(), 1);
    assert.equal(await page.locator("#keep").count(), 1);
    await page.locator("#keep").click();
    await page.waitForFunction(() => document.querySelector("#status")?.textContent?.includes("unsaved changes"), null, {
      timeout: 15000,
    });
    assert.equal(await page.locator("#price\\:k1").inputValue(), "26.00");
    await page.locator("#reread").click();
    await page.waitForFunction(
      (want) => document.querySelector("#status")?.textContent?.includes(want),
      `server has revision ${rev + 1}; your edits kept`,
      { timeout: 15000 }
    );
    await page.locator("#adopt").click();
    await page.waitForFunction((want) => document.querySelector("#status")?.textContent?.includes(want), `loaded revision ${rev + 1}`, {
      timeout: 15000,
    });
    assert.equal(await page.locator("#price\\:k1").inputValue(), "19.99");
    assert.equal(await page.locator("#totals").textContent(), "1 lines, 39.98 total");
    assert.deepEqual(await rowOrder(), ["k1"]);
    return `adopted server revision ${rev + 1}`;
  });

  await check("detached-click-silent", async () => {
    const oldSave = await page.$("#save");
    const before = posts();
    await page.locator("#add").click();
    await page.waitForFunction(() => document.querySelector("#status")?.textContent?.includes("unsaved"));
    assert.equal(await page.evaluate((node) => !node.isConnected, oldSave), true);
    await page.evaluate((node) => node.click(), oldSave);
    await page.waitForTimeout(600);
    assert.equal(posts() - before, 0, "a disposed view's listener must never fire");
    const added = await page.evaluate(() => [...document.querySelectorAll("input[id^=id]")].map((node) => node.id).pop());
    await page.locator(`#del\\:${added.split(":")[1]}`).click();
    await page.waitForTimeout(400);
    assert.deepEqual(await rowOrder(), ["k1"]);
  });

  await check("disposed-response-silent", async () => {
    await context.route("**/api/tenants/1/invoices/7", async (route) => {
      if (route.request().method() !== "POST") return route.continue();
      await new Promise((resolve) => setTimeout(resolve, 3000));
      await route.continue();
    });
    try {
      const errorsBefore = pageerrors.length;
      await fill("#price\\:k1", "27.00");
      const before = posts();
      const sent = page.waitForRequest(
        (request) => request.url().includes("/api/tenants/") && request.method() === "POST",
        { timeout: 15000 }
      );
      await page.locator("#save").click();
      await sent;
      await page.goto("about:blank");
      await page.waitForTimeout(4500);
      assert.equal(posts() - before, 1, "the flight must leave before navigation");
      assert.equal(pageerrors.length, errorsBefore, "a response landing on a disposed app must stay silent");
      await page.goto(base + "/invoice-grid?tenant=1&invoice=7", { waitUntil: "load" });
      await page.waitForFunction(() => document.querySelector("#status")?.textContent?.match(/loaded revision \d+/), null, {
        timeout: 15000,
      });
      return "late response on disposed app silent; reboot clean";
    } finally {
      await context.unroute("**/api/tenants/1/invoices/7");
    }
  });

  await check("ghost-denied", async () => {
    await context.addCookies([{ name: "session", value: "tok-ghost", url: base }]);
    try {
      const rev = await currentRev();
      await fill("#price\\:k1", "28.00");
      await saveAndWait("access denied; draft kept");
      assert.equal(await page.locator("#price\\:k1").inputValue(), "28.00");
      assert.equal(await currentRev(), rev, "denied save must not move the revision");
    } finally {
      await context.addCookies([{ name: "session", value: "tok-alice", url: base }]);
    }
  });

  await check("unavailable-load", async () => {
    // A locked store still answers reads, so the busy load is
    // injected as the server's exact 503 bytes for one GET; the
    // reread then reaches the live server. Only the transport is
    // faulted; the load handling is the application's.
    const denied = await context.newPage();
    watch(denied);
    denied.on("pageerror", (error) => pageerrors.push(String(error?.message ?? error)));
    try {
      await context.route("**/api/tenants/1/invoices/7", async (route) => {
        if (route.request().method() !== "GET") return route.continue();
        await route.fulfill({
          status: 503,
          contentType: "application/json",
          body: '{"case":"invoice_contract::grid_load_unavailable","value":{"message":"store unavailable"}}',
        });
      });
      try {
        await denied.goto(base + "/invoice-grid?tenant=1&invoice=7", { waitUntil: "load" });
        await denied.waitForFunction(
          () => document.querySelector("#status")?.textContent?.includes("server busy; retry reread"),
          null,
          { timeout: 15000 }
        );
      } finally {
        await context.unroute("**/api/tenants/1/invoices/7");
      }
      // The reread lands on the empty boot draft, which differs from
      // the snapshot, so the fold conservatively offers adopt-or-keep
      // instead of loading outright; adopting converges.
      await denied.locator("#reread").click();
      await denied.waitForFunction(
        () => document.querySelector("#status")?.textContent?.match(/server has revision (\d+); your edits kept/),
        null,
        { timeout: 15000 }
      );
      const target = await denied.evaluate(
        () => document.querySelector("#status")?.textContent?.match(/server has revision (\d+)/)?.[1]
      );
      await denied.locator("#adopt").click();
      await denied.waitForFunction(
        (want) => document.querySelector("#status")?.textContent?.includes(want),
        `loaded revision ${target}`,
        { timeout: 15000 }
      );
      return `busy load retries into adopt at rev ${target}`;
    } finally {
      await denied.close();
    }
  });

  await check("denied-load", async () => {
    const denied = await context.newPage();
    watch(denied);
    denied.on("pageerror", (error) => pageerrors.push(String(error?.message ?? error)));
    try {
      await denied.goto(base + "/invoice-grid?tenant=2&invoice=8", { waitUntil: "load" });
      await denied.waitForFunction(
        () => document.querySelector("#status")?.textContent?.includes("access denied; draft kept"),
        null,
        { timeout: 15000 }
      );
      assert.equal(await denied.locator("#grid h1").textContent(), "Tenant 2 invoice 8 revision ");
      const html = await denied.locator("#grid").innerHTML();
      assert.ok(!html.includes("Globex"), "denied load must not leak stored content");
    } finally {
      await denied.close();
    }
  });

  await check("reload-no-durability", async () => {
    // The ghost leg leaves the "28.00" draft uncommitted; committing
    // it first makes the reload target server truth.
    await saveAndWait(/saved revision \d+/.source);
    const rev = await currentRev();
    const price = await page.locator("#price\\:k1").inputValue();
    const totals = await page.locator("#totals").textContent();
    await fill("#price\\:k1", "29.00");
    await page.reload({ waitUntil: "load" });
    await page.waitForFunction((want) => document.querySelector("#status")?.textContent?.includes(want), `loaded revision ${rev}`, {
      timeout: 15000,
    });
    assert.equal(await page.locator("#price\\:k1").inputValue(), price);
    assert.equal(await page.locator("#totals").textContent(), totals);
  });

  await check("navigation-stable", async () => {
    const errorsBefore = pageerrors.length;
    const rev = await currentRev();
    await page.goto("about:blank");
    await page.waitForTimeout(500);
    assert.equal(page.url(), "about:blank");
    await page.goto(base + "/invoice-grid?tenant=1&invoice=7", { waitUntil: "load" });
    await page.waitForFunction((want) => document.querySelector("#status")?.textContent?.includes(want), `loaded revision ${rev}`, {
      timeout: 15000,
    });
    assert.equal(pageerrors.length, errorsBefore, "navigation must not raise page errors");
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
      const added = await page.evaluate(() => [...document.querySelectorAll("input[id^=id]")].map((node) => node.id).pop());
      await page.locator(`#del\\:${added.split(":")[1]}`).click();
      await page.waitForTimeout(250);
    }
    assert.equal(await count(), baseline, "add/remove cycles must not accumulate nodes");
    return `${baseline} nodes`;
  });

  await check("network-ledger-clean", async () => {
    assert.equal(aborted.length, 0, `non-loopback requests: ${aborted.join(", ")}`);
    for (const entry of requests) {
      const host = new URL(entry.url).hostname;
      assert.ok(host === "127.0.0.1" || host === "localhost", `grid left loopback: ${entry.url}`);
    }
    assert.equal(pageerrors.length, 0, pageerrors.join("; "));
    // Offline legs fail resource loads by design; the served CSP's
    // navigate-to directive warns once per load; WebKit refuses the
    // screenshot mechanism's own stylesheet once. All pinned exactly.
    const pinned = new Set([
      "Unrecognized Content-Security-Policy directive 'navigate-to'.",
      "Refused to apply a stylesheet because its hash, its nonce, or 'unsafe-inline' does not appear in the style-src directive of the Content Security Policy.",
    ]);
    for (const text of consoleErrors) {
      assert.ok(
        /Failed to load resource/.test(text) || pinned.has(text.trim()),
        `non-resource console error: ${text}`
      );
    }
    assert.ok(
      ledger.some((entry) => entry.method === "GET" && entry.url.includes("/api/tenants/1/invoices/7")),
      "ledger lacks the captured-path GET load"
    );
    assert.ok(ledger.every((entry) => entry.url.startsWith(base)), "ledger holds a non-origin invoice call");
    return `${requests.length} requests, ${ledger.length} invoice calls`;
  });

  await page.screenshot({ path: join(outdir, "screenshot.png") });
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
