// UP23 browser conformance evidence for the query-selected fixture
// (tests/integration/testdata/browser-conformance): drives the served
// paired application (compiler-built server plus real SQLite) in one
// named Playwright browser through every scenario — query edges,
// timer fire/order, view disposal, equality, exact codecs, sync
// cancellation and post-attach focus — and records DOM verdicts,
// network bytes and console/page faults. Usage:
//   node conformance.mjs <chromium|webkit> <base> <outdir> <script-url>
// Aborts every non-loopback request, so a passing run proves the
// fixture never needs a CDN, authored script, or foreign client
// runtime: the single served script must equal the report-selected
// paired URL, and window.htmx must stay undefined. Writes report.json
// and screenshot.png into outdir.
import { strict as assert } from "node:assert";
import { mkdirSync, writeFileSync } from "node:fs";
import { join } from "node:path";

const [wanted, base, outdir, scriptUrl] = process.argv.slice(2);
if (
  (wanted !== "chromium" && wanted !== "webkit") ||
  !base ||
  !outdir ||
  !scriptUrl?.startsWith("/__can/assets/")
) {
  console.error("usage: node conformance.mjs <chromium|webkit> <base> <outdir> <script-url>");
  process.exit(2);
}
mkdirSync(outdir, { recursive: true });

const playwright = await import("playwright");
const browser = await playwright[wanted].launch({ timeout: 120000 });
let userAgent = "";
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
    if (url.protocol !== "http:" && url.protocol !== "https:") return route.continue();
    if (url.hostname === "127.0.0.1" || url.hostname === "localhost") return route.continue();
    aborted.push(route.request().url());
    return route.abort("blockedbyclient");
  });
  const watch = (target) => {
    target.on("requestfinished", async (request) => {
      const response = await request.response().catch(() => undefined);
      requests.push({ method: request.method(), url: request.url(), status: response?.status() ?? 0 });
    });
    target.on("requestfailed", (request) => {
      requests.push({ method: request.method(), url: request.url(), status: -1 });
    });
  };
  const listen = (target) => {
    target.on("pageerror", (error) => pageerrors.push(String(error?.message ?? error)));
    target.on("console", (message) => {
      if (message.type() === "error") consoleErrors.push(message.text());
    });
  };
  const open = async (query) => {
    const page = await context.newPage();
    watch(page);
    listen(page);
    const response = await page.goto(base + query, { waitUntil: "load" });
    return { page, response };
  };
  const verdicts = (page) =>
    page.evaluate(() =>
      [...document.querySelectorAll("#invoice-grid p[id]")].map((node) => `${node.id}=${node.textContent}`)
    );
  const noDuplicates = async (page) =>
    page.evaluate(() => {
      const ids = [...document.querySelectorAll("[id]")].map((node) => node.id);
      return { total: ids.length, duplicates: ids.length - new Set(ids).size };
    });

  // Client identity rides the first scenario load: the single script
  // must equal the report-selected paired URL.
  const first = await open("/invoice-grid?scenario=equality");
  const bootHeaders = first.response?.headers() ?? {};
  const csp = bootHeaders["content-security-policy"] ?? "";
  await check("client-identity", async () => {
    const scripts = await first.page
      .locator("script")
      .evaluateAll((nodes) => nodes.map((node) => node.getAttribute("src")));
    assert.deepEqual(scripts, [scriptUrl]);
    assert.equal(await first.page.evaluate(() => typeof window.htmx), "undefined");
    assert.ok(csp.includes("script-src 'self'"), `csp lacks script-src: ${csp}`);
    assert.ok(csp.includes("connect-src 'self'"), `csp lacks connect-src: ${csp}`);
    const bundle = await first.page.evaluate(async (src) => (await fetch(src)).text(), scriptUrl);
    assert.ok(bundle.includes("$canBrowserMain"), "bundle lacks the Can browser entry");
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
  });
  await check("equality", async () => {
    await first.page.waitForSelector("#done", { timeout: 8000 });
    assert.deepEqual(await verdicts(first.page), [
      "v-eq-int=int-eq=true",
      "v-eq-int-no=int-neq=false",
      "v-eq-max=max-eq=true",
      "v-eq-bounds=bounds-eq=false",
      "v-eq-str=str-eq=true",
      "v-eq-str-no=str-neq=false",
      "done=done",
    ]);
  });
  await first.page.close();

  const scenario = async (query, selector = "#done") => {
    const { page } = await open(query);
    await page.waitForSelector(selector, { timeout: 8000 });
    return page;
  };

  await check("codecs", async () => {
    const page = await scenario("/invoice-grid?scenario=codecs");
    try {
      assert.deepEqual(await verdicts(page), [
        "v-max=max=9223372036854775807",
        "v-min=min=-9223372036854775808",
        "v-zero=zero=0",
        "v-neg=neg=-42",
        "v-parse-whole=to-int:42=42",
        "v-parse-negative=to-int:-7=-7",
        "v-parse-padded=to-int:007=invalid",
        "v-parse-empty=to-int:=invalid",
        "v-parse-max=to-int:9223372036854775807=9223372036854775807",
        "done=done",
      ]);
    } finally {
      await page.close();
    }
  });

  await check("query-plain", async () => {
    const page = await scenario("/invoice-grid?scenario=query&probe=hello");
    try {
      assert.deepEqual(await verdicts(page), [
        "v-probe=probe=some:hello",
        "v-absent=absent=none",
        "v-dup=dup=none",
        "v-huge=huge=none",
        "done=done",
      ]);
    } finally {
      await page.close();
    }
  });

  await check("query-missing-probe", async () => {
    const page = await scenario("/invoice-grid?scenario=query");
    try {
      assert.deepEqual(await verdicts(page), [
        "v-probe=probe=none",
        "v-absent=absent=none",
        "v-dup=dup=none",
        "v-huge=huge=none",
        "done=done",
      ]);
    } finally {
      await page.close();
    }
  });

  await check("query-duplicate", async () => {
    const page = await scenario("/invoice-grid?scenario=query&probe=hello&dup=a&dup=b");
    try {
      assert.deepEqual(await verdicts(page), [
        "v-probe=probe=some:hello",
        "v-absent=absent=none",
        "v-dup=dup=invalid",
        "v-huge=huge=none",
        "done=done",
      ]);
    } finally {
      await page.close();
    }
  });

  await check("query-encoded-alias", async () => {
    const page = await scenario("/invoice-grid?scenario=query&probe=hello&%70robe=alias");
    try {
      assert.deepEqual(await verdicts(page), [
        "v-probe=probe=invalid",
        "v-absent=absent=none",
        "v-dup=dup=none",
        "v-huge=huge=none",
        "done=done",
      ]);
    } finally {
      await page.close();
    }
  });

  // The query precheck is global per read: an overlong value, an
  // overlong search or a malformed escape anywhere fails the
  // scenario read itself, so the boot notice renders with no done.
  const bootNotice = async (query, notice) => {
    const { page } = await open(query);
    try {
      await page.waitForSelector("#status", { timeout: 8000 });
      assert.equal(await page.locator("#status").textContent(), notice);
      assert.equal(await page.locator("#status").getAttribute("role"), "status");
      assert.equal(await page.locator("#done").count(), 0);
      assert.equal(await page.locator("#invoice-grid p[id]").count(), 1);
    } finally {
      await page.close();
    }
  };

  await check("query-huge-value", async () => {
    await bootNotice(`/invoice-grid?scenario=query&probe=hello&huge=${"z".repeat(300)}`, "bad link");
  });

  await check("query-huge-search", async () => {
    await bootNotice(`/invoice-grid?scenario=query&pad=${"z".repeat(8200)}`, "bad link");
  });

  await check("query-malformed", async () => {
    await bootNotice("/invoice-grid?scenario=query&probe=%E0%A4%A", "bad link");
  });

  await check("no-scenario", async () => {
    await bootNotice("/invoice-grid", "no scenario");
  });

  await check("unknown-scenario", async () => {
    await bootNotice("/invoice-grid?scenario=nope", "unknown scenario");
  });

  await check("timers", async () => {
    const page = await scenario("/invoice-grid?scenario=timers");
    try {
      const order = await page.locator("#ticks li").evaluateAll((nodes) => nodes.map((n) => n.textContent));
      assert.deepEqual(order, ["tick-0", "tick-5"]);
      await page.waitForTimeout(500);
      assert.equal(await page.locator("#ticks li").count(), 2, "timers must fire exactly once");
      const ids = await noDuplicates(page);
      assert.equal(ids.duplicates, 0);
      return "ordered tick-0,tick-5; fire-once";
    } finally {
      await page.close();
    }
  });

  await check("dispose", async () => {
    const page = await scenario("/invoice-grid?scenario=dispose", "#armed");
    try {
      await page.evaluate(() => {
        window.__probe = document.getElementById("probe");
      });
      await page.locator("#probe").evaluate((node) => {
        node.value = "x";
        node.dispatchEvent(new Event("input", { bubbles: true }));
      });
      await page.waitForFunction(() => document.querySelectorAll("#marks li").length === 1);
      assert.deepEqual(
        await page.locator("#marks li").evaluateAll((nodes) => nodes.map((n) => n.textContent)),
        ["saw-input"]
      );
      await page.locator("#dispose").click();
      await page.waitForSelector("#done", { timeout: 8000 });
      assert.equal(await page.locator("#v-disposed").textContent(), "disposed");
      assert.equal(await page.locator("#probe").count(), 0, "disposal must remove view nodes");
      assert.equal(await page.locator("#marks").count(), 0, "disposal must remove the marker list");
      await page.evaluate(() => {
        window.__probe.dispatchEvent(new Event("input", { bubbles: true }));
      });
      await page.waitForTimeout(500);
      assert.equal(await page.locator("#v-disposed").textContent(), "disposed");
      const saw = await page.evaluate(() => document.body.innerHTML.includes("saw-"));
      assert.equal(saw, false, "detached listener and cleared timer must stay silent");
      return "listener detached, timer cleared, nodes removed";
    } finally {
      await page.close();
    }
  });

  await check("cancel", async () => {
    const page = await scenario("/invoice-grid?scenario=cancel", "#armed");
    try {
      const before = page.url();
      await page.locator("#name").evaluate((node, value) => {
        node.value = value;
      }, "Ann");
      await page.locator("#go").click();
      await page.waitForFunction(() => document.querySelectorAll("#marks li").length === 1);
      assert.deepEqual(
        await page.locator("#marks li").evaluateAll((nodes) => nodes.map((n) => n.textContent)),
        ["submitted"]
      );
      assert.equal(page.url(), before, "prevented submit must not navigate");
      assert.equal(await page.locator("#name").inputValue(), "Ann");
      // Implicit submission on Enter inside the form is prevented too.
      await page.locator("#name").press("Enter");
      await page.waitForFunction(() => document.querySelectorAll("#marks li").length === 2);
      assert.equal(page.url(), before);
      // The harness listener registers after boot, so it observes the
      // final synchronous preventDefault decision per key.
      await page.evaluate(() => {
        window.__prevented = [];
        document.getElementById("key").addEventListener("keydown", (event) => {
          window.__prevented.push(event.defaultPrevented);
        });
      });
      await page.locator("#key").press("a");
      await page.waitForFunction(() => document.querySelectorAll("#marks li").length === 3);
      await page.locator("#key").press("Enter");
      await page.waitForFunction(() => document.querySelectorAll("#marks li").length === 4);
      assert.deepEqual(await page.evaluate(() => window.__prevented), [false, true]);
      const noncancelable = await page.evaluate(() => {
        const node = document.getElementById("key");
        const event = new KeyboardEvent("keydown", { key: "Enter", cancelable: false, bubbles: true });
        node.dispatchEvent(event);
        return event.defaultPrevented;
      });
      await page.waitForFunction(() => document.querySelectorAll("#marks li").length === 5);
      assert.deepEqual(
        await page.locator("#marks li").evaluateAll((nodes) => nodes.map((n) => n.textContent)),
        ["submitted", "submitted", "key:a", "saved", "saved"]
      );
      assert.equal(noncancelable, false, "noncancelable Enter dispatches without cancellation");
      return "submit+Enter prevented; other/noncancelable dispatch once";
    } finally {
      await page.close();
    }
  });

  await check("focus", async () => {
    const page = await scenario("/invoice-grid?scenario=focus");
    try {
      assert.equal(await page.evaluate(() => document.activeElement?.id), "target");
      const ids = await noDuplicates(page);
      assert.equal(ids.duplicates, 0);
      await page.screenshot({ path: join(outdir, "screenshot.png") });
      userAgent = await page.evaluate(() => navigator.userAgent);
      return "activeElement target after attach";
    } finally {
      await page.close();
    }
  });

  await check("network-ledger-clean", async () => {
    assert.equal(aborted.length, 0, `non-loopback requests: ${aborted.join(", ")}`);
    for (const entry of requests) {
      const host = new URL(entry.url).hostname;
      assert.ok(host === "127.0.0.1" || host === "localhost", `left loopback: ${entry.url}`);
      assert.ok(!entry.url.includes("/api/"), `fixture must not call APIs: ${entry.url}`);
    }
    assert.equal(pageerrors.length, 0, pageerrors.join("; "));
    // Both engines warn once per load about the served policy's
    // navigate-to directive; the bytes are server-owned. WebKit
    // additionally refuses the screenshot mechanism's own injected
    // stylesheet once. Both texts are pinned exactly; anything else
    // on the console fails.
    const pinned = new Set([
      "Unrecognized Content-Security-Policy directive 'navigate-to'.",
      "Refused to apply a stylesheet because its hash, its nonce, or 'unsafe-inline' does not appear in the style-src directive of the Content Security Policy.",
    ]);
    const rest = consoleErrors.filter((text) => !pinned.has(text.trim()));
    assert.equal(rest.length, 0, rest.join("; "));
    return `${requests.length} loopback requests, ${consoleErrors.length - rest.length} pinned engine warnings`;
  });

  await context.close();
} finally {
  await browser.close();
}

const passed = checks.every((entry) => entry.passed);
writeFileSync(
  join(outdir, "report.json"),
  JSON.stringify(
    { browser: wanted, version: browser.version(), userAgent, base, passed, checks, requests, aborted, pageerrors, consoleErrors },
    null,
    2
  ) + "\n"
);
for (const entry of checks) console.log(`${entry.passed ? "PASS" : "FAIL"} ${entry.name}${entry.detail ? ` ${entry.detail}` : ""}`);
if (!passed) process.exit(1);
console.log(`browser conformance evidence passed on ${wanted}`);
