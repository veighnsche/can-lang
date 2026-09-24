import { test, expect } from "bun:test";
import { runBrowserWireVectors } from "./browser-wire-vectors.ts";

test("shared strict wire vectors pass under bun", () => {
  const results = runBrowserWireVectors();
  expect(results.length).toBeGreaterThan(0);
  for (const result of results) {
    if (!result.pass) throw new Error(`${result.name}: ${result.detail}`);
  }
});
