import { expect, test } from "bun:test";
import { validateStaticGraph } from "./output-graph.ts";
const transpiler = new Bun.Transpiler({ loader: "ts" });
test("closed graph rejects computed edges anywhere in generated syntax", () => {
  for (const source of [
    `const target="node:fs"; await import(target);`,
    'const target="fs"; await import(`node:${target}`);',
    `export async function f(target:string){return import(/* nested */ target);}`,
    `export const nested=()=>()=>import(String("node:fs"));`,
    `import "./missing.ts";`,
    `export * from "./missing.ts";`,
    `export {x} from "./missing.ts";`,
  ])
    expect(() => validateStaticGraph(transpiler.transformSync(source), [])).toThrow();
});
test("upstream grammar distinguishes data/comments and exact static edges", () => {
  for (const source of [
    `const text="import(target)"; const pattern=/import\\(target\\)/;`,
    `// import(target)\nexport const n=1n;`,
    'export const text=`import(${"target"})`;',
    `export const object={import(value:string){return value;}}; object.import("data");`,
    `import {x} from "./known.ts"; export {x};`,
    `export * from "./known.ts";`,
    `import type {T} from "./known.ts"; export const n:T=1;`,
  ])
    expect(() =>
      validateStaticGraph(transpiler.transformSync(source), ["./known.ts"]),
    ).not.toThrow();
});
