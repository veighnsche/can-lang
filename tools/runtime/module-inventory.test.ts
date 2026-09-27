import { expect, test } from "bun:test";
import { readFileSync } from "node:fs";
import { validateStaticGraph } from "./output-graph.ts";

const runtime = new URL("../../runtime/", import.meta.url);
const inventory: { schemaVersion: number; modules: Record<string, string[]> } = JSON.parse(
  readFileSync(new URL("modules.json", runtime), "utf8"),
);
const transpiler = new Bun.Transpiler({ loader: "ts" });

test("every private runtime module passes native closed-graph validation against its inventory", () => {
  expect(inventory.schemaVersion).toBe(1);
  for (const [path, imports] of Object.entries(inventory.modules)) {
    const source = readFileSync(new URL(path, runtime), "utf8");
    expect(
      () => validateStaticGraph(transpiler.transformSync(source), imports),
      path,
    ).not.toThrow();
    for (const edge of transpiler.scanImports(source)) {
      expect(edge.kind, path).toBe("import-statement");
      expect(imports, `${path}: ${edge.path}`).toContain(edge.path);
    }
  }
});
