// Validate compiler-owned TypeScript without evaluating it or resolving packages.
import "../../runtime/environment.ts";
import { createHash } from "node:crypto";
import { validateStaticGraph } from "./output-graph.ts";
import {validateMaps} from "./source-map-validation.ts";
const raw = await Bun.stdin.text();
const input = JSON.parse(raw);
if (input.schemaVersion !== 1 || !Array.isArray(input.modules)) throw new Error("invalid output validation request");
if (input.sourceIndex !== undefined) validateMaps(input.sourceIndex,input.maps);
const transpiler = new Bun.Transpiler({ loader: "ts" });
for (const module of input.modules) {
  if (typeof module.path !== "string" || typeof module.source !== "string" || !Array.isArray(module.imports)) throw new Error("invalid output module");
  if (input.sourceIndex) {
    const mapped=input.sourceIndex.modules.find((m:any)=>m.path===module.path);
    if (mapped) {
      const lines=module.source.split("\n");
      if (mapped.segments.some((p:any)=>p.line>lines.length || p.column>lines[p.line-1].length)) throw new Error("mapping outside generated module");
    }
  }
  try {
    validateStaticGraph(transpiler.transformSync(module.source), module.imports);
    for (const edge of transpiler.scanImports(module.source)) {
      if (edge.kind !== "import-statement" || !module.imports.includes(edge.path)) throw new Error("undeclared executable module edge");
    }
  } catch {
    throw new Error("invalid generated TypeScript or import inventory: " + module.path);
  }
}
console.log(JSON.stringify({ schemaVersion: 1, kind: "can.output-validation", modules: input.modules.length, requestSHA256: createHash("sha256").update(raw).digest("hex") }));
