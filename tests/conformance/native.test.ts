import { test, expect } from "bun:test";
import manifest from "../../distribution/target.json";
import { qualify, sha256, apiAvailable } from "./native";
import { readFileSync } from "node:fs";
import { types } from "node:util";

const actual = { name: "bun", version: Bun.version, revision: Bun.revision,
  platform: process.platform, architecture: process.arch, sha256: sha256(readFileSync(process.execPath)) };
test("qualified native APIs and behaviors", async () => {
  expect((await qualify(manifest, actual)).failures).toEqual([]);
});
for (const name of ["JSON.rawJSON", "Array.fromAsync", "node:util.types.isProxy", "node:util.types.isNativeError", "Bun.Transpiler.prototype.scanImports", "Bun.Transpiler.prototype.transformSync"]) {
  test(`refuse missing ${name} without fallback`, async () => {
    const owner = name.startsWith("node:util") ? types : name === "JSON.rawJSON" ? JSON : name === "Array.fromAsync" ? Array : Bun.Transpiler.prototype;
    const key = name.split(".").at(-1)!;
    const descriptor = Object.getOwnPropertyDescriptor(owner, key)!;
    try {
      Object.defineProperty(owner, key, { ...descriptor, value: undefined });
      const report = await qualify(manifest, actual);
      expect(report.passed).toBe(false);
      expect(report.failures).toContain(`missing native API: ${name}`);
    } finally { Object.defineProperty(owner, key, descriptor); }
  });
}
const wrongIdentity = { name: "node", version: "0.0.0", revision: "wrong", sha256: "tampered",
  platform: manifest.runtime.platform === "linux" ? "darwin" : "linux",
  architecture: manifest.runtime.architecture === "amd64" ? "arm64" : "amd64" };
for (const [key, value] of Object.entries(wrongIdentity)) {
  test(`refuse different ${key}`, async () => {
    const report = await qualify(manifest, { ...actual, [key]: value });
    expect(report.passed).toBe(false);
    expect(report.failures.some(message => message.startsWith(`runtime.${key}:`))).toBe(true);
  });
}

test("refuse missing AsyncLocalStorage without fallback",async()=>{
 const report=await qualify(manifest,actual,name=>name!=="node:async_hooks.AsyncLocalStorage"&&apiAvailable(name));
 expect(report.passed).toBe(false);expect(report.failures).toContain("missing native API: node:async_hooks.AsyncLocalStorage");
});
