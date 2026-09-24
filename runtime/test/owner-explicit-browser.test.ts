import { test, expect } from "bun:test";
import { AsyncLocalStorage } from "node:async_hooks";
import { createHash } from "node:crypto";
import { readFileSync, writeFileSync, mkdtempSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { probeExplicitOwner, type ExplicitProbeResult } from "./owner-explicit-probe.ts";

const here = dirname(fileURLToPath(import.meta.url));
const root = join(here, "..", "..");
type BrowserPage = {
  on: (event: string, listener: (error: unknown) => void) => void;
  addScriptTag: (options: { content: string }) => Promise<void>;
  evaluate: <T>(fn: () => T | Promise<T>) => Promise<T>;
};
type TestBrowser = {
  newPage: () => Promise<BrowserPage>;
  close: () => Promise<void>;
};

function assertProbeResult(result: ExplicitProbeResult): void {
  expect(result.interleaved.outcomes).toEqual(["ok", "ok"]);
  expect(result.interleaved.crossRefused).toBe(true);
  expect(result.interleaved.resources).toEqual(["closed", "closed"]);
  expect(result.interleaved.clean).toBe(true);
  expect(result.guarded.completion).toBe("ok");
  expect(result.guarded.rootCompletion).toBe("ok");
  expect(result.guarded.resource).toBe("closed");
  expect(result.guarded.afterClose).toBe("resource_state");
  expect(result.guarded.clean).toBe(true);
  expect(result.leaseAfterSuspension.completion).toBe("ok");
  expect(result.leaseAfterSuspension.subleased).toBe(true);
  expect(result.leaseAfterSuspension.clean).toBe(true);
  expect(result.lateLoser.completion).toBe("ok");
  expect(result.lateLoser.diagnostics).toBe(1);
  expect(result.lateLoser.phase).toBe("late");
  expect(result.lateLoser.clean).toBe(true);
  expect(result.disposal.closes).toBe(1);
  expect(result.disposal.doubleRefused).toBe(true);
  expect(result.disposal.postCloseRefused).toBe(true);
  expect(result.disposal.clean).toBe(true);
  expect(result.nestedScopes.innerWasOpen).toBe(true);
  expect(result.nestedScopes.innerClosedWhileOuterOpen).toBe(true);
  expect(result.nestedScopes.outerClosed).toBe(true);
  expect(result.nestedScopes.clean).toBe(true);
  expect(result.coordination.selection).toBe("all");
  expect(result.coordination.values).toEqual(["true:3", "second"]);
  expect(result.coordination.clean).toBe(true);
}

test("explicit owner probe passes natively", async () => {
  const ambient = new AsyncLocalStorage<string>();
  let observed: string | undefined;
  await ambient.run("A", async () => {
    await Promise.resolve();
    observed = ambient.getStore();
  });
  expect(observed).toBe("A");
  assertProbeResult(await probeExplicitOwner());
});

test("chromium pins the sync ALS failure and passes the explicit adapter", async () => {
  const grid = readFileSync(join(root, "tests/integration/browser/build-grid.mjs"), "utf8");
  const extract = (name: string): string => {
    const found = grid.match(new RegExp(`const ${name} = \`([\\s\\S]*?)\`;`));
    if (!found) throw new Error(`missing ${name}`);
    if (found[1].includes("${")) throw new Error("unexpected shim interpolation");
    return found[1];
  };
  const asyncShim = extract("asyncHooksShim"),
    utilShim = extract("utilShim");
  const asyncShimSha256 = createHash("sha256").update(asyncShim).digest("hex");
  expect(asyncShimSha256).toMatch(/^[0-9a-f]{64}$/);
  // The production adapter must never consult ambient async storage: the
  // bundled node:async_hooks exposes only throwing methods. Construction
  // still succeeds because owner.ts instantiates the store at module load
  // for the ambient Bun path; any runtime use explodes loudly.
  const throwingAls =
    "const forbidden = () => { throw new Error('explicit owner probe forbids AsyncLocalStorage'); };\n" +
    "export class AsyncLocalStorage {\n" +
    "  getStore() { forbidden(); }\n" +
    "  run() { forbidden(); }\n" +
    "  enterWith() { forbidden(); }\n" +
    "  disable() { forbidden(); }\n" +
    "}\n";
  const scratch = mkdtempSync(join(tmpdir(), "can-owner-explicit-"));
  let browser: TestBrowser | undefined;
  try {
    const entry = join(scratch, "entry.ts");
    writeFileSync(
      entry,
      `import { probeExplicitOwner } from ${JSON.stringify(join(here, "owner-explicit-probe.ts"))};\n` +
        `import { probeContext } from ${JSON.stringify(join(root, "docs/syntax-taste/evidence/2026-09-24/selected-behavior/experiments/browser-owner/context-probe.mjs"))};\n` +
        `${asyncShim.replace("export class AsyncLocalStorage", "class SyncShimALS").replace("export class", "class")}\n` +
        `globalThis.probeOwnerExplicit = async () => ({ shim: await probeContext(SyncShimALS), explicit: await probeExplicitOwner() });\n`,
    );
    const built = await Bun.build({
      entrypoints: [entry],
      target: "browser",
      plugins: [
        {
          name: "owner-explicit-shims",
          setup(build) {
            build.onResolve({ filter: /^node:/ }, (args) => ({
              path: args.path,
              namespace: "owner-explicit-shim",
            }));
            build.onLoad({ filter: /.*/, namespace: "owner-explicit-shim" }, (args) => {
              const contents: Record<string, string> = {
                "node:async_hooks": throwingAls,
                "node:util": utilShim,
                "node:crypto": readFileSync(
                  join(root, "tests/integration/browser/sha256-shim.mjs"),
                  "utf8",
                ),
                "node:fs":
                  'export function readFileSync() { throw new Error("probe has no diagnostic source index"); }',
              };
              const found = contents[args.path];
              if (!found) throw new Error(`uncovered engine import: ${args.path}`);
              return { contents: found, loader: "js" };
            });
          },
        },
      ],
    });
    if (!built.success) throw new Error(built.logs.map((log) => log.message).join("\n"));
    const { chromium } = await import(
      join(root, "tests/integration/browser/node_modules/playwright/index.mjs")
    );
    const launched: TestBrowser = await chromium.launch({ timeout: 30000 });
    browser = launched;
    const page = await launched.newPage();
    const pageErrors: string[] = [];
    page.on("pageerror", (error: unknown) => pageErrors.push(String(error)));
    await page.addScriptTag({ content: await built.outputs[0].text() });
    const browserResult = await page.evaluate(() =>
      (
        globalThis as unknown as {
          probeOwnerExplicit: () => Promise<{
            shim: { isolated: boolean };
            explicit: ExplicitProbeResult;
          }>;
        }
      ).probeOwnerExplicit(),
    );
    // Gate 5 regression pin: the synchronous shim still loses owner context
    // after suspension, so it can never supply browser ownership semantics.
    expect(browserResult.shim.isolated).toBe(false);
    assertProbeResult(browserResult.explicit);
    expect(pageErrors).toEqual([]);
  } finally {
    await browser?.close();
    rmSync(scratch, { recursive: true, force: true });
  }
}, 180000);
