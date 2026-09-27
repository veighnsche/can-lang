// C02 emitted-Can control evidence: drives the served paired controls
// fixture (tests/integration/testdata/browser-controls) in one named
// Playwright browser through every snapshot field and live call —
// checked, multiselect, files, modifiers, composition, caret, live
// writes/reads, dirty reset, autofill observation and denials — and
// records DOM verdicts, network bytes and console/page faults. Usage:
//   node controls.mjs <chromium|firefox|webkit> <base> <outdir> <script-url>
// Aborts every non-loopback request, so a passing run proves the
// fixture never needs a CDN, authored script, or foreign client
// runtime: the single served script must equal the report-selected
// paired URL, and window.htmx must stay undefined. Writes report.json
// and screenshot.png into outdir.
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
  console.error("usage: node controls.mjs <chromium|firefox|webkit> <base> <outdir> <script-url>");
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
  const open = async () => {
    const page = await context.newPage();
    watch(page);
    listen(page);
    const response = await page.goto(base + "/invoice-grid?controls=1", { waitUntil: "load" });
    await page.waitForSelector("#done", { timeout: 8000 });
    return { page, response };
  };
  const verdict = (page, id) =>
    page
      .waitForFunction((want) => document.getElementById(want)?.textContent !== "pending", id, {
        timeout: 8000,
      })
      .then(() => page.evaluate((want) => document.getElementById(want).textContent, id));
  const leg = async (name, fn) => {
    await check(name, async () => {
      const { page } = await open();
      try {
        return await fn(page);
      } finally {
        await page.close();
      }
    });
  };

  // Client identity rides the first load: the single script must equal
  // the report-selected paired URL.
  const first = await open();
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
    return "paired script only";
  });
  await check("boot", async () => {
    const lines = await first.page.evaluate(() =>
      [...document.querySelectorAll("#invoice-grid p[id]")].map((node) => `${node.id}=${node.textContent}`),
    );
    assert.deepEqual(lines, [
      "v-text=pending",
      "v-key=pending",
      "v-check=pending",
      "v-multi=pending",
      "v-files=pending",
      "v-setvalue=pending",
      "v-setchecked=pending",
      "v-setselected=pending",
      "v-setcaret=pending",
      "v-readall=pending",
      "v-reset=pending",
      "v-reject=pending",
      "done=done",
    ]);
    return "13 lines render";
  });
  await first.page.close();

  await leg("text-echo", async (page) => {
    await page.fill("#f-text", "Hi");
    assert.equal(await verdict(page, "v-text"), `text=Hi|composing=false|caret=2,2,${implicitDirection}`);
    return "value+composing+caret echo";
  });

  await leg("key-echo", async (page) => {
    await page.click("#f-text");
    await page.keyboard.press("a");
    assert.equal(
      await verdict(page, "v-key"),
      "key=a|alt=false|ctrl=false|meta=false|shift=false",
    );
    await page.keyboard.press("Shift+Enter");
    await page.waitForFunction(
      () => document.getElementById("v-key").textContent !== "key=a|alt=false|ctrl=false|meta=false|shift=false",
      null,
      { timeout: 8000 },
    );
    assert.equal(
      await page.evaluate(() => document.getElementById("v-key").textContent),
      "key=Enter|alt=false|ctrl=false|meta=false|shift=true",
    );
    return "plain a then shift+Enter";
  });

  await leg("check-echo", async (page) => {
    await page.click("#f-check");
    assert.equal(await verdict(page, "v-check"), "check=true|value=on");
    await page.click("#f-check");
    await page.waitForFunction(
      () => document.getElementById("v-check").textContent === "check=false|value=on",
      null,
      { timeout: 8000 },
    );
    return "checked toggles, value on";
  });

  await leg("multi-echo", async (page) => {
    await page.selectOption("#f-multi", ["a", "c"]);
    assert.equal(await verdict(page, "v-multi"), "multi=a,c");
    return "a,c selected";
  });

  await leg("files-echo", async (page) => {
    await page.setInputFiles("#f-files", [
      { name: "a.csv", mimeType: "text/csv", buffer: Buffer.from("a,b,c,d,e,f,") },
      { name: "b.png", mimeType: "image/png", buffer: Buffer.alloc(300) },
    ]);
    assert.equal(await verdict(page, "v-files"), "files=2:a.csv:12:text/csv");
    return "2 files, first detailed";
  });

  await leg("setvalue", async (page) => {
    await page.click("#b-setvalue");
    assert.equal(await verdict(page, "v-setvalue"), "setvalue=can-set");
    assert.equal(await page.evaluate(() => document.getElementById("f-text").value), "can-set");
    return "live write+read agree";
  });

  await leg("setchecked", async (page) => {
    await page.click("#b-setchecked");
    assert.equal(await verdict(page, "v-setchecked"), "setchecked=true");
    assert.equal(await page.evaluate(() => document.getElementById("f-check").checked), true);
    return "live write+read agree";
  });

  await leg("setselected", async (page) => {
    await page.click("#b-setselected");
    assert.equal(await verdict(page, "v-setselected"), "setsel=b|z");
    assert.deepEqual(
      await page.evaluate(() => [...document.getElementById("f-multi").selectedOptions].map((node) => node.value)),
      ["b"],
    );
    assert.deepEqual(
      await page.evaluate(() => [...document.getElementById("f-single").selectedOptions].map((node) => node.value)),
      ["z"],
    );
    return "multi b, single last-wins z";
  });

  await leg("setcaret", async (page) => {
    await page.click("#b-setcaret");
    assert.equal(await verdict(page, "v-setcaret"), "setcaret=1,3,forward");
    assert.deepEqual(
      await page.evaluate(() => {
        const field = document.getElementById("f-text");
        return [field.selectionStart, field.selectionEnd, field.selectionDirection];
      }),
      [1, 3, "forward"],
    );
    return "1,3,forward live";
  });

  await leg("composing-synthetic", async (page) => {
    await page.evaluate(() => {
      const field = document.getElementById("f-text");
      field.setSelectionRange(2, 4, "backward");
      field.dispatchEvent(new InputEvent("input", { bubbles: true, isComposing: true, data: "k" }));
    });
    assert.equal(await verdict(page, "v-text"), "text=start|composing=true|caret=2,4,backward");
    return "synthetic composing input echoes";
  });

  await leg("autofill-simulated", async (page) => {
    await page.evaluate(() => {
      const field = document.getElementById("f-text");
      field.value = "ann@example.com";
      field.setSelectionRange(15, 15, "none");
      field.dispatchEvent(new Event("input", { bubbles: true }));
    });
    // Collapsed-caret direction after an explicit none set is
    // engine-defined: Firefox normalizes to forward, Chromium and
    // WebKit honor none. The snapshot projects each faithfully.
    const collapsed = wanted === "firefox" ? "forward" : "none";
    assert.equal(
      await verdict(page, "v-text"),
      `text=ann@example.com|composing=false|caret=15,15,${collapsed}`,
    );
    return "external live change observed";
  });

  await leg("dirty-reset", async (page) => {
    await page.fill("#f-text", "dirty work");
    await page.check("#f-check");
    await page.selectOption("#f-multi", ["a", "c"]);
    await page.click("#b-reset");
    assert.equal(await verdict(page, "v-reset"), "reset=start|false|");
    assert.equal(await page.evaluate(() => document.getElementById("f-text").value), "start");
    assert.equal(await page.evaluate(() => document.getElementById("f-check").checked), false);
    assert.deepEqual(
      await page.evaluate(() => [...document.getElementById("f-multi").selectedOptions].map((node) => node.value)),
      [],
    );
    return "defaults restored live";
  });

  await leg("readall", async (page) => {
    await page.fill("#f-text", "R1");
    await page.check("#f-check");
    await page.selectOption("#f-multi", ["c"]);
    await page.setInputFiles("#f-files", [
      { name: "r.csv", mimeType: "text/csv", buffer: Buffer.from("x") },
    ]);
    await page.click("#b-readall");
    assert.equal(await verdict(page, "v-readall"), "readall=R1|true|c|2|1");
    return "save-time reads agree";
  });

  await leg("reject", async (page) => {
    await page.click("#b-reject");
    assert.equal(await verdict(page, "v-reject"), "reject=rejected|rejected|-1");
    return "denials named, caret neutral";
  });

  await check("network-ledger-clean", async () => {
    assert.equal(aborted.length, 0, `non-loopback requests: ${aborted.join(", ")}`);
    for (const entry of requests) {
      const host = new URL(entry.url).hostname;
      assert.ok(host === "127.0.0.1" || host === "localhost", `left loopback: ${entry.url}`);
      assert.ok(!entry.url.includes("/api/"), `fixture must not call APIs: ${entry.url}`);
    }
    assert.equal(pageerrors.length, 0, pageerrors.join("; "));
    const pinned = new Set([
      "Unrecognized Content-Security-Policy directive 'navigate-to'.",
      "Refused to apply a stylesheet because its hash, its nonce, or 'unsafe-inline' does not appear in the style-src directive of the Content Security Policy.",
    ]);
    const rest = consoleErrors.filter((text) => !pinned.has(text.trim()));
    assert.equal(rest.length, 0, rest.join("; "));
    const shot = await context.newPage();
    await shot.goto(base + "/invoice-grid?controls=1", { waitUntil: "load" });
    await shot.waitForSelector("#done", { timeout: 8000 });
    await shot.screenshot({ path: join(outdir, "screenshot.png") });
    userAgent = await shot.evaluate(() => navigator.userAgent);
    await shot.close();
    return `${requests.length} loopback requests, ${consoleErrors.length - rest.length} pinned engine warnings`;
  });

  await context.close();
} finally {
  if (!remote) await browser.close();
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
console.log(`browser controls evidence passed on ${wanted}`);
if (remote) process.exit(0); // the shared ws outlives a passed leg; release the process
