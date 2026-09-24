// Run from the repository root: node docs/syntax-taste/evidence/2026-09-24/selected-behavior/experiments/htmx-guard/probe.mjs
import { readFileSync, writeFileSync } from "node:fs";
import { resolve } from "node:path";
import { pathToFileURL } from "node:url";

const { chromium } = await import(pathToFileURL(resolve("tests/integration/browser/node_modules/playwright/index.mjs")));
const asset = readFileSync("distribution/assets/htmx-4.0.0.min.js", "utf8");
const browser = await chromium.launch();
const results = {};

async function run(name, response, options = {}) {
  const page = await browser.newPage();
  const requests = [];
  let release;
  let reached;
  const reachedPromise = new Promise((done) => { reached = done; });
  const releasePromise = new Promise((done) => { release = done; });
  await page.route("http://127.0.0.1:19877/**", async (route) => {
    const url = new URL(route.request().url());
    requests.push(url.pathname);
    if (url.pathname === "/asset") return route.fulfill({ status: 200, contentType: "text/javascript", body: asset });
    if (url.pathname === "/") return route.fulfill({ status: 200, contentType: "text/html", body: `<!doctype html><html><head><meta name="htmx-config" content='{"mode":"same-origin","noSwap":[204,304,"4xx","5xx"]}'><script defer src="/asset"></script></head><body><button id="source" hx-get="/reply" hx-target="${options.initialMissing ? "#missing" : "#target"}" hx-swap="innerHTML" hx-status:503='{"swap":"innerHTML"}'>go</button><div id="target">old target</div><div id="other">old other</div></body></html>` });
    if (url.pathname === "/reply") {
      reached();
      if (options.delay) await releasePromise;
      return route.fulfill({ status: response.status, contentType: "text/html", headers: response.headers ?? {}, body: response.body });
    }
    if (url.pathname === "/hijack") return route.fulfill({ status: 200, contentType: "text/html", body: "redirected" });
    return route.abort();
  });
  const consoleMessages = [];
  page.on("console", (message) => consoleMessages.push(`${message.type()}: ${message.text()}`));
  page.on("pageerror", (error) => consoleMessages.push(`pageerror: ${error.message}`));
  await page.goto("http://127.0.0.1:19877/", { waitUntil: "load" });
  await page.evaluate((mode) => {
    window.probeEvents = [];
    const record = (event) => {
      const detail = event.detail ?? {};
      const tasks = detail.tasks;
      window.probeEvents.push({
        name: event.type,
        target: event.target?.id ?? null,
        keys: Object.keys(detail),
        taskCount: Array.isArray(tasks) ? tasks.length : null,
        tasks: Array.isArray(tasks) ? tasks.map((task) => ({
          keys: Object.keys(task),
          kind: task.kind ?? null,
          type: task.type ?? null,
          swap: task.swap ?? null,
          target: task.target?.id ?? task.elt?.id ?? null,
        })) : null,
        responseStatus: detail.ctx?.response?.status ?? detail.response?.status ?? null,
        swap: detail.ctx?.swap ?? null,
        connected: detail.ctx?.target?.isConnected ?? null,
      });
    };
    for (const name of ["htmx:before:request", "htmx:before:response", "htmx:before:swap", "htmx:target:error", "htmx:after:swap"]) {
      document.addEventListener(name, (event) => {
        record(event);
        if (mode === "guard" && name === "htmx:before:request") {
          const target = event.detail?.ctx?.target ?? null;
          if (!target || !target.isConnected) event.preventDefault();
        }
        if (mode === "guard" && name === "htmx:before:response") {
          const headers = event.detail?.ctx?.response?.headers ?? event.detail?.response?.headers;
          if (["HX-Redirect", "HX-Retarget", "HX-Reswap", "HX-Reselect", "HX-Refresh", "HX-Push-Url", "HX-Replace-Url"].some((header) => headers?.has?.(header))) event.preventDefault();
        }
        if (mode === "guard" && name === "htmx:before:swap") {
          const tasks = event.detail?.tasks ?? [];
          const original = document.getElementById("target");
          if (!original?.isConnected || tasks.length !== 1 || tasks[0].target !== original || tasks[0].type !== "main") event.preventDefault();
        }
      });
    }
  }, options.guard ? "guard" : "observe");
  if (options.initialMissing) await page.locator("#target").evaluate((node) => node.remove());
  await page.locator("#source").click();
  if (options.delay) {
    await reachedPromise;
    if (options.removeDuringFlight) await page.locator("#target").evaluate((node) => node.remove());
    release();
  }
  await page.waitForTimeout(150);
  results[name] = {
    requests,
    url: page.url(),
    target: await page.locator("#target").count() ? await page.locator("#target").innerHTML() : null,
    other: await page.locator("#other").count() ? await page.locator("#other").innerHTML() : null,
    events: await page.evaluate(() => window.probeEvents),
    consoleMessages,
  };
  await page.close();
}

try {
  await run("missing_before_unguarded", { status: 200, body: "<p>main</p>" }, { initialMissing: true });
  await run("missing_before_guarded", { status: 200, body: "<p>main</p>" }, { initialMissing: true, guard: true });
  await run("redirect_unguarded", { status: 200, headers: { "HX-Redirect": "/hijack" }, body: "<p>main</p>" });
  await run("redirect_guarded", { status: 200, headers: { "HX-Redirect": "/hijack" }, body: "<p>main</p>" }, { guard: true });
  await run("retarget_unguarded", { status: 200, headers: { "HX-Retarget": "#other" }, body: "<p>main</p>" });
  await run("retarget_guarded", { status: 200, headers: { "HX-Retarget": "#other" }, body: "<p>main</p>" }, { guard: true });
  for (const status of [200, 503]) {
    await run(`oob_${status}_unguarded`, { status, body: '<p>main</p><div id="other" hx-swap-oob="innerHTML">new other</div>' });
    await run(`oob_${status}_guarded`, { status, body: '<p>main</p><div id="other" hx-swap-oob="innerHTML">new other</div>' }, { guard: true });
    await run(`partial_${status}_unguarded`, { status, body: '<p>main</p><template hx type="partial" hx-target="#other"><p>new partial</p></template>' });
    await run(`partial_${status}_guarded`, { status, body: '<p>main</p><template hx type="partial" hx-target="#other"><p>new partial</p></template>' }, { guard: true });
  }
  await run("removed_in_flight_guarded", { status: 200, body: "<p>main</p>" }, { guard: true, delay: true, removeDuringFlight: true });
  writeFileSync("docs/syntax-taste/evidence/2026-09-24/selected-behavior/experiments/htmx-guard/results.json", JSON.stringify(results, null, 2) + "\n");
  process.stdout.write(JSON.stringify(Object.fromEntries(Object.entries(results).map(([name, value]) => [name, { requests: value.requests, url: value.url, target: value.target, other: value.other, events: value.events?.map((event) => ({ name: event.name, taskCount: event.taskCount, tasks: event.tasks, keys: event.keys })) ?? [] }])), null, 2) + "\n");
} finally {
  await browser.close();
}
