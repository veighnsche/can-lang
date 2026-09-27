// C02 native control facts: drives the static controls page in one
// named Playwright browser through the DOM semantics the Can adapter
// projects — checkbox IDL, dirty live value vs attribute, form reset,
// multiselect/single selection, file metadata, modifier keys,
// composition flags and caret accessors — and records verdicts,
// network bytes and console/page faults. Usage:
//   node controls-native.mjs <chromium|firefox|webkit> <base> <outdir>
// Aborts every non-loopback request. Writes report.json and
// screenshot.png into outdir.
import { strict as assert } from "node:assert";
import { mkdirSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import { launchWanted } from "./firefox-remote.mjs";

const [wanted, base, outdir] = process.argv.slice(2);
if (
  (wanted !== "chromium" && wanted !== "firefox" && wanted !== "webkit") ||
  !base ||
  !outdir
) {
  console.error("usage: node controls-native.mjs <chromium|firefox|webkit> <base> <outdir>");
  process.exit(2);
}
mkdirSync(outdir, { recursive: true });

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
  const page = await context.newPage();
  page.on("requestfinished", async (request) => {
    const response = await request.response().catch(() => undefined);
    requests.push({ method: request.method(), url: request.url(), status: response?.status() ?? 0 });
  });
  page.on("requestfailed", (request) => {
    requests.push({ method: request.method(), url: request.url(), status: -1 });
  });
  page.on("pageerror", (error) => pageerrors.push(String(error?.message ?? error)));
  page.on("console", (message) => {
    if (message.type() === "error") consoleErrors.push(message.text());
  });
  await page.goto(base + "/index.html", { waitUntil: "load" });
  const facts = (expression) => page.evaluate(expression);

  await check("checkbox-idl", async () => {
    assert.equal(await facts(() => document.getElementById("n-check").checked), false);
    assert.equal(await facts(() => document.getElementById("n-check").value), "on");
    await page.click("#n-check");
    assert.equal(await facts(() => document.getElementById("n-check").checked), true);
    assert.equal(await facts(() => document.getElementById("n-check").value), "on");
    await page.click("#n-check");
    assert.equal(await facts(() => document.getElementById("n-check").checked), false);
    return "checked toggles, value stays on";
  });

  await check("dirty-divergence", async () => {
    await page.fill("#n-text", "user edit");
    await facts(() => document.getElementById("n-text").setAttribute("value", "normalized"));
    assert.equal(await facts(() => document.getElementById("n-text").value), "user edit");
    assert.equal(
      await facts(() => document.getElementById("n-text").getAttribute("value")),
      "normalized",
    );
    return "attribute moves the default, live keeps the edit";
  });

  await check("form-reset", async () => {
    await page.reload({ waitUntil: "load" });
    await page.fill("#n-text", "dirty work");
    await page.check("#n-check");
    await facts(() => document.getElementById("n-form").reset());
    assert.equal(await facts(() => document.getElementById("n-text").value), "start");
    assert.equal(await facts(() => document.getElementById("n-check").checked), false);
    return "reset restores start/unchecked";
  });

  await check("multiselect", async () => {
    await page.selectOption("#n-multi", ["a", "c"]);
    assert.deepEqual(
      await facts(() =>
        [...document.getElementById("n-multi").selectedOptions].map((node) => node.value),
      ),
      ["a", "c"],
    );
    await facts(() => {
      for (const node of document.getElementById("n-multi").options) node.selected = false;
      document.getElementById("n-multi").options[1].selected = true;
    });
    assert.deepEqual(
      await facts(() =>
        [...document.getElementById("n-multi").selectedOptions].map((node) => node.value),
      ),
      ["b"],
    );
    return "a,c then b";
  });

  await check("single-last-wins", async () => {
    const picked = await facts(() => {
      const select = document.getElementById("n-single");
      select.options[0].selected = true;
      select.options[2].selected = true;
      return [...select.selectedOptions].map((node) => node.value);
    });
    assert.deepEqual(picked, ["z"]);
    return "last match wins";
  });

  await check("files", async () => {
    await page.setInputFiles("#n-files", [
      { name: "a.csv", mimeType: "text/csv", buffer: Buffer.from("a,b,c,d,e,f,") },
      { name: "b.png", mimeType: "image/png", buffer: Buffer.alloc(300) },
    ]);
    assert.deepEqual(
      await facts(() =>
        [...document.getElementById("n-files").files].map((file) => ({
          name: file.name,
          size: file.size,
          type: file.type,
        })),
      ),
      [
        { name: "a.csv", size: 12, type: "text/csv" },
        { name: "b.png", size: 300, type: "image/png" },
      ],
    );
    return "2 files with name/size/type";
  });

  await check("modifiers", async () => {
    await page.evaluate(() => {
      window.__seen = [];
      document.getElementById("n-form").addEventListener("submit", (event) => event.preventDefault());
      for (const type of ["keydown", "click"]) {
        document.getElementById("n-panel").addEventListener(type, (event) =>
          window.__seen.push({
            type,
            key: event.key ?? "",
            alt: event.altKey,
            ctrl: event.ctrlKey,
            meta: event.metaKey,
            shift: event.shiftKey,
          }),
        );
      }
      document.getElementById("n-text").addEventListener("keydown", (event) =>
        window.__seen.push({
          type: "text-" + event.type,
          key: event.key ?? "",
          alt: event.altKey,
          ctrl: event.ctrlKey,
          meta: event.metaKey,
          shift: event.shiftKey,
        }),
      );
    });
    await page.click("#n-text");
    await page.keyboard.press("Shift+Enter");
    await page.keyboard.press("s", { delay: 10 });
    const seen = await facts(() => window.__seen);
    const shifted = seen.find((entry) => entry.key === "Enter");
    assert.deepEqual(shifted, {
      type: "text-keydown",
      key: "Enter",
      alt: false,
      ctrl: false,
      meta: false,
      shift: true,
    });
    const plain = seen.find((entry) => entry.key === "s");
    assert.deepEqual(plain, {
      type: "text-keydown",
      key: "s",
      alt: false,
      ctrl: false,
      meta: false,
      shift: false,
    });
    const synthetic = await facts(() => {
      const before = window.__seen.length;
      document
        .getElementById("n-panel")
        .dispatchEvent(new KeyboardEvent("keydown", { key: "x", ctrlKey: true, bubbles: true }));
      return window.__seen[before];
    });
    assert.deepEqual(synthetic, {
      type: "keydown",
      key: "x",
      alt: false,
      ctrl: true,
      meta: false,
      shift: false,
    });
    return "real shift+Enter, real s, synthetic ctrl+x";
  });

  await check("autofill-simulated", async () => {
    const seen = await facts(() => {
      const field = document.getElementById("n-text");
      const values = [];
      field.addEventListener("input", () => values.push(field.value));
      field.value = "ann@example.com";
      field.dispatchEvent(new Event("input", { bubbles: true }));
      return values;
    });
    assert.deepEqual(seen, ["ann@example.com"]);
    return "external live change observed";
  });

  await check("composing", async () => {
    const seen = await facts(() => {
      const field = document.getElementById("n-text");
      const flags = [];
      field.addEventListener("input", (event) => flags.push(event.isComposing));
      field.value = "real";
      field.dispatchEvent(new InputEvent("input", { bubbles: true }));
      field.dispatchEvent(new InputEvent("input", { bubbles: true, isComposing: true, data: "k" }));
      return flags;
    });
    assert.deepEqual(seen, [false, true]);
    return "real false, synthetic true";
  });

  await check("selection", async () => {
    await page.fill("#n-text", "hello");
    await facts(() => document.getElementById("n-text").setSelectionRange(1, 3, "backward"));
    assert.deepEqual(
      await facts(() => {
        const field = document.getElementById("n-text");
        return [field.selectionStart, field.selectionEnd, field.selectionDirection];
      }),
      [1, 3, "backward"],
    );
    assert.equal(
      await facts(() => document.getElementById("n-check").selectionStart),
      null,
    );
    assert.equal(
      await facts(() => {
        try {
          document.getElementById("n-check").setSelectionRange(0, 0, "none");
          return "no-throw";
        } catch (error) {
          return error.name;
        }
      }),
      "InvalidStateError",
    );
    return "1,3,backward; checkbox null + InvalidStateError";
  });

  await check("network-ledger-clean", async () => {
    assert.equal(aborted.length, 0, `non-loopback requests: ${aborted.join(", ")}`);
    for (const entry of requests) {
      const host = new URL(entry.url).hostname;
      assert.ok(host === "127.0.0.1" || host === "localhost", `left loopback: ${entry.url}`);
    }
    assert.equal(pageerrors.length, 0, pageerrors.join("; "));
    assert.equal(consoleErrors.length, 0, consoleErrors.join("; "));
    await page.screenshot({ path: join(outdir, "screenshot.png") });
    userAgent = await page.evaluate(() => navigator.userAgent);
    return `${requests.length} loopback requests`;
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
console.log(`browser controls-native evidence passed on ${wanted}`);
if (remote) process.exit(0); // the shared ws outlives a passed leg; release the process
