// Run from any directory: bun <this directory>/run.mjs
import { readFileSync, writeFileSync, mkdtempSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { resolve, dirname, join } from "node:path";
import { AsyncLocalStorage } from "node:async_hooks";
import { createHash } from "node:crypto";
import { probeRuntime } from "./runtime-probe.ts";
import { probeContext, probeExplicitTokens } from "./context-probe.mjs";
const here = dirname(new URL(import.meta.url).pathname);
const root = resolve(here, "../../../../../../..");
const grid = readFileSync(join(root, "tests/integration/browser/build-grid.mjs"), "utf8");
const extract = name => {
  const found = grid.match(new RegExp(`const ${name} = \x60([\\s\\S]*?)\x60;`));
  if (!found) throw new Error(`missing ${name}`);
  if (found[1].includes("${")) throw new Error("unexpected shim interpolation");
  return found[1];
};
const asyncShim = extract("asyncHooksShim"), utilShim = extract("utilShim");
const scratch = mkdtempSync(join(tmpdir(), "can-browser-owner-"));
let browser;
try {
  const entry = join(scratch, "entry.ts");
  writeFileSync(entry, `import { probeRuntime } from ${JSON.stringify(join(here, "runtime-probe.ts"))};\nimport { probeContext, probeExplicitTokens } from ${JSON.stringify(join(here, "context-probe.mjs"))};\n${asyncShim.replace("export class", "class")}\nglobalThis.probeOwner = async () => ({ context: await probeContext(AsyncLocalStorage), explicitTokens: await probeExplicitTokens(), runtime: await probeRuntime() });\n`);
  const built = await Bun.build({ entrypoints: [entry], target: "browser", plugins: [{ name: "review-owner-shims", setup(build) {
    build.onResolve({ filter: /^node:/ }, args => ({ path: args.path, namespace: "owner-shim" }));
    build.onLoad({ filter: /.*/, namespace: "owner-shim" }, args => {
      const contents = { "node:async_hooks": asyncShim, "node:util": utilShim, "node:crypto": readFileSync(join(root, "tests/integration/browser/sha256-shim.mjs"), "utf8"), "node:fs": 'export function readFileSync() { throw new Error("probe has no diagnostic source index"); }' }[args.path];
      if (!contents) throw new Error(`uncovered engine import: ${args.path}`);
      return { contents, loader: "js" };
    });
  } }] });
  if (!built.success) throw new Error(built.logs.join("\n"));
  const { chromium } = await import(join(root, "tests/integration/browser/node_modules/playwright/index.mjs"));
  browser = await chromium.launch({ timeout: 30000 });
  const page = await browser.newPage();
  const pageErrors = [];
  page.on("pageerror", error => pageErrors.push(String(error)));
  await page.addScriptTag({ content: await built.outputs[0].text() });
  const browserResult = await page.evaluate(() => globalThis.probeOwner());
  const native = { context: await probeContext(AsyncLocalStorage), explicitTokens: await probeExplicitTokens(), runtime: await probeRuntime() };
  const result = { bun: Bun.version, chromium: browser.version(), asyncShimSha256: createHash("sha256").update(asyncShim).digest("hex"), native, browser: browserResult, pageErrors };
  const checks = {
    nativeContextIsolated: native.context.isolated,
    shimContextLost: !browserResult.context.isolated,
    nativeResourceUsesSucceed: native.runtime.interleavedRoots.outcomes.every(r => r.completion === "ok"),
    shimResourceUsesFail: browserResult.runtime.interleavedRoots.outcomes.every(r => r.completion === "resource_state"),
    nativeCallbackResumes: native.runtime.guardedCallback.completion === "ok",
    shimCallbackFailsAfterAwait: browserResult.runtime.guardedCallback.completion === "resource_state",
    explicitTokenControlsPass: native.explicitTokens.isolated && browserResult.explicitTokens.isolated,
    noPageErrors: pageErrors.length === 0,
  };
  result.checks = checks;
  writeFileSync(join(here, "results.json"), JSON.stringify(result, null, 2) + "\n");
  console.log(JSON.stringify(checks, null, 2));
  if (Object.values(checks).some(x => !x)) process.exitCode = 1;
} finally {
  await browser?.close();
  rmSync(scratch, { recursive: true, force: true });
}
