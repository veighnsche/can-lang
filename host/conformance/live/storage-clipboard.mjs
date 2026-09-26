// D02 live-browser leg: Op A storage + Op B clipboard reviewed adapters
// against the REAL native calls (`localStorage`, `navigator.clipboard`)
// in one named Playwright browser. The Go gate (`live_test.go`) bundles
// the delivered adapters with `bun build`, then this runner serves a
// page importing both bundles and drives every check through them.
// Usage:
//   node storage-clipboard.mjs <chromium|webkit|firefox> <bundle-dir> <outdir>
// Writes report.json into outdir; exits nonzero on any failed check.
// Pinned but UNRUN while C01 is blocked-open: see README.md.
import { strict as assert } from "node:assert";
import { mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import { createServer } from "node:http";

const [wanted, bundleDir, outdir] = process.argv.slice(2);
if (
  (wanted !== "chromium" && wanted !== "webkit" && wanted !== "firefox") ||
  !bundleDir ||
  !outdir
) {
  console.error("usage: node storage-clipboard.mjs <chromium|webkit|firefox> <bundle-dir> <outdir>");
  process.exit(2);
}
mkdirSync(outdir, { recursive: true });

const assets = {
  "/storage.bundle.js": readFileSync(join(bundleDir, "storage.bundle.js")),
  "/clipboard.bundle.js": readFileSync(join(bundleDir, "clipboard.bundle.js")),
};
const pageHtml = `<!doctype html><html><body><script type="module">
import * as storage from "/storage.bundle.js";
import * as clipboard from "/clipboard.bundle.js";
window.__d02_storage = storage;
window.__d02_clipboard = clipboard;
window.__d02_ready = true;
</script></body></html>`;
const server = createServer((request, response) => {
  if (request.url === "/") {
    response.writeHead(200, { "content-type": "text/html" }).end(pageHtml);
    return;
  }
  const asset = assets[request.url];
  if (asset === undefined) {
    response.writeHead(404).end("not found");
    return;
  }
  response.writeHead(200, { "content-type": "text/javascript" }).end(asset);
});
await new Promise((resolve) => server.listen(0, "127.0.0.1", resolve));
const base = `http://127.0.0.1:${server.address().port}`;

const playwright = await import("playwright");
const browser = await playwright[wanted].launch({ timeout: 120000 });
const checks = [];
const check = (name, fn) =>
  Promise.resolve()
    .then(fn)
    .then((detail = "") => checks.push({ name, passed: true, detail }))
    .catch((error) => checks.push({ name, passed: false, detail: String(error?.message ?? error) }));

try {
  // Chromium honors the grant; the other engines may ignore it, in
  // which case the roundtrip leg below degrades to its documented
  // contract-level branch instead of assuming a grant that never
  // happened. The first green live run pins per-engine behavior.
  const context = await browser.newContext({ permissions: ["clipboard-read", "clipboard-write"] });
  const page = await context.newPage();
  const pageerrors = [];
  page.on("pageerror", (error) => pageerrors.push(String(error?.message ?? error)));
  await page.goto(base + "/", { waitUntil: "load" });
  await page.waitForFunction(() => window.__d02_ready === true, null, { timeout: 30000 });

  const userAgent = await page.evaluate(() => navigator.userAgent);
  const secureContext = await page.evaluate(() => window.isSecureContext);

  await check("storage: real roundtrip", () =>
    page.evaluate(() => {
      const adapter = window.__d02_storage.createStorageAdapter(
        window.__d02_storage.createNativeStorageHost(window),
      );
      assert.deepEqual(adapter.localSet("theme", "dark"), { ok: true, value: null });
      assert.deepEqual(adapter.localGet("theme"), { ok: true, value: "dark" });
      assert.deepEqual(adapter.localRemove("theme"), { ok: true, value: null });
      assert.deepEqual(adapter.localGet("theme"), { ok: true, value: null });
      return "set/get/remove/missing-null against real localStorage";
    }),
  );

  await check("storage: invalid key rejects before host contact", () =>
    page.evaluate(() => {
      const adapter = window.__d02_storage.createStorageAdapter(
        window.__d02_storage.createNativeStorageHost(window),
      );
      const before = window.localStorage.length;
      const got = adapter.localGet("");
      assert.equal(got.ok, false);
      assert.equal(got.failure.code, "storage::invalid_key");
      assert.equal(window.localStorage.length, before);
      return "invalid_key with unchanged store length";
    }),
  );

  await check("storage: real quota breach maps with no native text", () =>
    page.evaluate(() => {
      const adapter = window.__d02_storage.createStorageAdapter(
        window.__d02_storage.createNativeStorageHost(window),
      );
      // Fill toward the ~5MB origin quota in 256KB chunks; the loop
      // must terminate on the mapped leaf, never on an escaping throw.
      const leaves = [];
      for (let fill = 0; fill < 64; fill += 1) {
        const result = adapter.localSet(`quota-probe-${fill}`, "x".repeat(262144));
        if (!result.ok) {
          leaves.push(result.failure.code);
          assert.ok(!JSON.stringify(result).includes("QuotaExceededError"));
          break;
        }
      }
      for (let fill = 0; fill < 64; fill += 1) adapter.localRemove(`quota-probe-${fill}`);
      assert.deepEqual(leaves, ["storage::quota_exceeded"]);
      return "real quota error mapped, store cleaned";
    }),
  );

  await check("clipboard: permission matrix recorded", () =>
    page.evaluate(async () => {
      const states = {};
      for (const name of ["clipboard-read", "clipboard-write"]) {
        try {
          states[name] = (await navigator.permissions.query({ name })).state;
        } catch (error) {
          states[name] = `query-throws:${error?.name ?? "unknown"}`;
        }
      }
      return JSON.stringify(states);
    }),
  );

  await check("clipboard: raw empty-read shape recorded", () =>
    page.evaluate(async () => {
      // Fresh page, no prior write: record what the NATIVE call does
      // — resolution value or rejection name — without assuming it.
      // This leg pins the D01-documented uncertainty per browser.
      try {
        const text = await navigator.clipboard.readText();
        return `resolved:${JSON.stringify(text)}`;
      } catch (error) {
        return `rejected:${error?.name ?? "unknown"}`;
      }
    }),
  );

  // Chromium's grant makes the full roundtrip an assertion; on the
  // other engines the leg asserts the contract floor (no throw, no
  // native text, a known leaf) and records the outcome for the
  // first-run pinning follow-up.
  const strictRoundtrip = wanted === "chromium";
  await check(`clipboard: real write/read roundtrip (${strictRoundtrip ? "strict" : "contract-floor"})`, () =>
    page.evaluate(async (strict) => {
      const adapter = window.__d02_clipboard.createClipboardAdapter(
        window.__d02_clipboard.createNativeClipboardHost(window),
      );
      const written = await adapter.writeText("d02-live-probe");
      const read = written.ok ? await adapter.readText() : written;
      const serialized = JSON.stringify({ written, read });
      assert.ok(!/(NotAllowedError|SecurityError|DataError)/.test(serialized), `native text leaked: ${serialized}`);
      if (strict) {
        assert.deepEqual(written, { ok: true, value: null });
        assert.deepEqual(read, { ok: true, value: "d02-live-probe" });
        return "writeText/readText against the device clipboard";
      }
      assert.ok(["clipboard::denied", "clipboard::unavailable"].includes(read.failure?.code) || read.ok === true);
      return `contract floor holds: ${serialized}`;
    }, strictRoundtrip),
  );

  await check("clipboard: invalid write rejects before host contact", () =>
    page.evaluate(async () => {
      const adapter = window.__d02_clipboard.createClipboardAdapter(
        window.__d02_clipboard.createNativeClipboardHost(window),
      );
      const got = await adapter.writeText("");
      assert.equal(got.ok, false);
      assert.equal(got.failure.code, "clipboard::invalid_text");
      return "invalid_text without clipboard contact";
    }),
  );

  await check("no page errors escape any leg", async () => {
    assert.deepEqual(pageerrors, []);
    return "all legs clean";
  });

  writeFileSync(
    join(outdir, "report.json"),
    JSON.stringify({ browser: wanted, userAgent, secureContext, checks }, null, 2),
  );
} finally {
  await browser.close().catch(() => {});
  server.close();
}

const failed = checks.filter((entry) => !entry.passed);
if (failed.length > 0) {
  console.error(`live leg failures on ${wanted}: ${JSON.stringify(failed, null, 2)}`);
  process.exit(1);
}
console.log(`live legs pass on ${wanted}: ${checks.length}/${checks.length}`);
