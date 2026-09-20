import { test, expect } from "bun:test";
import manifest from "../../distribution/target.json";
import { qualify, sha256 } from "./native";
import { readFileSync } from "node:fs";

const actual = { name: "bun", version: Bun.version, revision: Bun.revision,
  platform: process.platform, architecture: process.arch, sha256: sha256(readFileSync(process.execPath)) };
test("qualified native APIs and behaviors", async () => {
  expect((await qualify(manifest, actual)).failures).toEqual([]);
});
for (const name of ["JSON.rawJSON", "Array.fromAsync"]) {
  test(`refuse missing ${name} without fallback`, async () => {
    const owner = name === "JSON.rawJSON" ? JSON : Array;
    const key = name === "JSON.rawJSON" ? "rawJSON" : "fromAsync";
    const descriptor = Object.getOwnPropertyDescriptor(owner, key)!;
    try {
      Object.defineProperty(owner, key, { ...descriptor, value: undefined });
      const report = await qualify(manifest, actual);
      expect(report.passed).toBe(false);
      expect(report.failures).toContain(`missing native API: ${name}`);
    } finally { Object.defineProperty(owner, key, descriptor); }
  });
}
for (const [key, value] of Object.entries({ name: "node", version: "0.0.0", revision: "wrong", platform: "linux", architecture: "x64", sha256: "tampered" })) {
  test(`refuse different ${key}`, async () => {
    const report = await qualify(manifest, { ...actual, [key]: value });
    expect(report.passed).toBe(false);
    expect(report.failures.some(message => message.startsWith(`runtime.${key}:`))).toBe(true);
  });
}
