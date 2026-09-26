import { test, expect } from "bun:test";

// Reproducible-packaging leg: every prototype entry bundles twice to
// byte-identical output, and bundle sizes are pinned as the packaging
// measurement for the record. Bun is pinned (1.4.2), so the pins are
// stable; a drift means the prototype changed, which must update the
// record deliberately.
const ENTRIES = [
  "tests/host-discrimination/adapters/storage.ts",
  "tests/host-discrimination/adapters/clipboard.ts",
  "tests/host-discrimination/adapters/chart-widget.ts",
  "tests/host-discrimination/companions/storage.ts",
  "tests/host-discrimination/companions/clipboard.ts",
  "tests/host-discrimination/companions/chart-widget.ts",
];

async function bundle(entry: string): Promise<Uint8Array> {
  const result = await Bun.build({ entrypoints: [entry], minify: false });
  expect(result.success).toBe(true);
  expect(result.outputs.length).toBe(1);
  return new Uint8Array(await result.outputs[0].arrayBuffer());
}

test("prototype bundles are byte-identical across builds", async () => {
  for (const entry of ENTRIES) {
    const first = await bundle(entry);
    const second = await bundle(entry);
    expect(first.length).toBe(second.length);
    expect(Buffer.from(first).toString("hex")).toBe(Buffer.from(second).toString("hex"));
  }
}, 30000);

test("bundle sizes are pinned", async () => {
  const sizes: Record<string, number> = {};
  for (const entry of ENTRIES) {
    sizes[entry] = (await bundle(entry)).length;
  }
  expect(sizes).toEqual({
    "tests/host-discrimination/adapters/storage.ts": 4559,
    "tests/host-discrimination/adapters/clipboard.ts": 2987,
    "tests/host-discrimination/adapters/chart-widget.ts": 6051,
    "tests/host-discrimination/companions/storage.ts": 18360,
    "tests/host-discrimination/companions/clipboard.ts": 16768,
    "tests/host-discrimination/companions/chart-widget.ts": 20236,
  });
}, 30000);
