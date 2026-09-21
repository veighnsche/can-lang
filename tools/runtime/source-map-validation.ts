import { TraceMap, decodedMappings, GenMapping, addMapping, toEncodedMap } from "./vendor/source-maps.mjs";

export function validateIndex(index: any): void {
  if (index?.schemaVersion !== 1 || index.kind !== "can.source-index" || !Array.isArray(index.sources) || !Array.isArray(index.modules)) throw new Error("invalid source index");
  const ids = new Map<string, any>();
  const safePath = (p: unknown) => typeof p === "string" && p.length > 0 && !p.startsWith("/") && !p.includes("\\") && !p.split("/").some(x => !x || x === "." || x === "..") && !/[\x00-\x1f\x7f]/.test(p);
  const integer = (n: unknown, min = 0) => Number.isSafeInteger(n) && (n as number) >= min;
  for (const s of index.sources) {
    if (typeof s.id !== "string" || ids.has(s.id) || !safePath(s.id) || !safePath(s.path) || !s.path.endsWith(".can") || !s.spans || typeof s.spans !== "object" || Array.isArray(s.spans)) throw new Error("invalid source identity");
    for (const [name, span] of Object.entries(s.spans) as [string, any][]) {
      if (!/^[a-z_]+:[0-9]+:[0-9]+$/.test(name) || name !== `${span.operation}:${span.start}:${span.end}` || !integer(span.start) || !integer(span.end, span.start) || !integer(span.line,1) || !integer(span.column) || !integer(span.endLine,span.line) || !integer(span.endColumn) || (span.endLine===span.line && span.endColumn<span.column)) throw new Error("invalid source span");
    }
    ids.set(s.id, s);
  }
  const paths = new Set<string>();
  for (const m of index.modules) {
    if (!safePath(m.path) || !m.path.endsWith(".ts") || paths.has(m.path) || !Array.isArray(m.segments)) throw new Error("invalid mapped module");
    paths.add(m.path);
    let line=0, column=-1;
    for (const p of m.segments) {
      if (!integer(p.line,1) || !integer(p.column) || p.line<line || (p.line===line && p.column<=column) || !Object.hasOwn(ids.get(p.source)?.spans ?? {},p.name)) throw new Error("invalid mapping segment");
      line=p.line;column=p.column;
    }
  }
}
export function validateMaps(index: any, maps: Record<string, any>): void {
  validateIndex(index);
  if (Object.keys(maps).length!==index.modules.length) throw new Error("source map inventory mismatch");
  const sources = new Map<string, any>(index.sources.map((s:any)=>[s.id,s]));
  for (const module of index.modules) {
    const map=maps[module.path];
    if (!map || map.version!==3 || map.file!==module.path.split("/").at(-1) || map.sourceRoot!==undefined || !Array.isArray(map.sources) || !Array.isArray(map.sourcesContent) || map.sourcesContent.length!==map.sources.length || map.sourcesContent.some((s:any)=>s!==null) || !Array.isArray(map.names) || typeof map.mappings!=="string" || map.sources.some((s:any)=>!sources.has(s)) || new Set(map.sources).size!==map.sources.length) throw new Error("invalid generated source map");
    const expected=new GenMapping({file:map.file});
    for (const p of module.segments) {
      const span=sources.get(p.source).spans[p.name];
      addMapping(expected,{generated:{line:p.line,column:p.column},source:p.source,original:{line:span.line,column:span.column},name:p.name});
    }
    const canonical=toEncodedMap(expected);
    if (Object.keys(map).filter(k=>map[k]!==undefined).sort().join()!==Object.keys(canonical).filter(k=>canonical[k]!==undefined).sort().join() || Object.keys(canonical).some(k=>JSON.stringify(map[k])!==JSON.stringify(canonical[k]))) throw new Error("noncanonical source map");
    const decoded=decodedMappings(new TraceMap(map));
    const actual: any[]=[];
    for (let l=0;l<decoded.length;l++) for (const p of decoded[l]) {
      if (p.length!==5) throw new Error("incomplete source segment");
      const source=map.sources[p[1]], name=map.names[p[4]], span=sources.get(source)?.spans[name];
      if (!span || p[2]!==span.line-1 || p[3]!==span.column) throw new Error("map/source-index disagreement");
      actual.push({line:l+1,column:p[0],source,name});
    }
    if (JSON.stringify(actual)!==JSON.stringify(module.segments)) throw new Error("map segments differ from compiler input");
  }
}
