// Upstream parsing owns JS grammar. This adapter only inspects its syntax tree;
// it neither evaluates source nor resolves imports. Inputs are compiler output,
// not an admission API or sandbox for arbitrary JavaScript.
import { parse } from "./vendor/acorn-8.18.0.mjs";

type Node = { type: string; [key: string]: unknown };
export function validateStaticGraph(javascript: string, imports: readonly string[]): void {
  const tree = parse(javascript, { ecmaVersion: "latest", sourceType: "module" });
  const pending: Node[] = [tree];
  while (pending.length) {
    const node = pending.pop()!;
    if (node.type === "ImportExpression") throw new Error("dynamic imports are not compiler output");
    if (node.type === "ImportDeclaration" || node.type === "ExportNamedDeclaration" || node.type === "ExportAllDeclaration") {
      const source = node.source as { value: unknown } | undefined;
      if (source && (typeof source.value !== "string" || !imports.includes(source.value))) throw new Error("undeclared static module edge");
    }
    for (const value of Object.values(node)) {
      if (Array.isArray(value)) {
        for (const child of value) if (child && typeof child.type === "string") pending.push(child);
      } else if (value && typeof value === "object" && typeof (value as Node).type === "string") {
        pending.push(value as Node);
      }
    }
  }
}
