// D02 conformance: reproducible-packaging leg. Every delivered entry
// bundles twice to byte-identical output, and bundle sizes are pinned
// as the packaging measurement for the record. Bun is pinned (1.4.2),
// so the pins are stable; drift means a deliverable changed, which
// must update the record deliberately and re-pass distribution review.
import { test, expect } from "bun:test";

const ENTRIES = [
  "host/adapters/storage.ts",
  "host/adapters/clipboard.ts",
  "host/companions/chart.ts",
];

async function bundle(entry: string): Promise<Uint8Array> {
  const result = await Bun.build({ entrypoints: [entry], minify: false });
  expect(result.success).toBe(true);
  expect(result.outputs.length).toBe(1);
  return new Uint8Array(await result.outputs[0].arrayBuffer());
}

test("delivered bundles are byte-identical across builds", async () => {
  for (const entry of ENTRIES) {
    const first = await bundle(entry);
    const second = await bundle(entry);
    expect(first.length).toBe(second.length);
    expect(Buffer.from(first).toString("hex")).toBe(Buffer.from(second).toString("hex"));
  }
}, 30000);

test("delivered bundle sizes are pinned", async () => {
  const sizes: Record<string, number> = {};
  for (const entry of ENTRIES) {
    sizes[entry] = (await bundle(entry)).length;
  }
  expect(sizes).toEqual({
    "host/adapters/storage.ts": 5618,
    "host/adapters/clipboard.ts": 4381,
    "host/companions/chart.ts": 20838,
  });
}, 30000);
