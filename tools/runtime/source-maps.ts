// Fixed offline tool. Encoding belongs to upstream gen-mapping, never Go VLQ.
import "../../runtime/environment.ts";
import { createHash } from "node:crypto";
import { GenMapping, addMapping, toEncodedMap } from "./vendor/source-maps.mjs";
import { validateIndex, validateMaps } from "./source-map-validation.ts";
const raw = await Bun.stdin.text(),
  index = JSON.parse(raw);
validateIndex(index);
const sources = new Map<string, any>(index.sources.map((s: any) => [s.id, s]));
const maps: Record<string, any> = Object.create(null);
for (const module of index.modules) {
  const map = new GenMapping({ file: module.path.split("/").at(-1) });
  for (const p of module.segments) {
    const span = sources.get(p.source).spans[p.name];
    addMapping(map, {
      generated: { line: p.line, column: p.column },
      source: p.source,
      original: { line: span.line, column: span.column },
      name: p.name,
    });
  }
  maps[module.path] = toEncodedMap(map);
}
validateMaps(index, maps);
console.log(
  JSON.stringify({
    schemaVersion: 1,
    kind: "can.source-maps",
    requestSHA256: createHash("sha256").update(raw).digest("hex"),
    maps,
  }),
);
