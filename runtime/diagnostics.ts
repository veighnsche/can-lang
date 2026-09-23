// Private diagnostics consume only verified generation maps. Native messages,
// native frame names, source contents and host paths never cross this boundary.
import { readFileSync } from "node:fs";
import { types as nativeTypes } from "node:util";
import { TraceMap, originalPositionFor } from "./vendor/trace-mapping.ts";
import type { FailureOrigin } from "./failure.ts";

type Span = Readonly<{
  start: number;
  end: number;
  line: number;
  column: number;
  endLine: number;
  endColumn: number;
  operation: string;
}>;
type Source = Readonly<{ id: string; path: string; spans: Record<string, Span> }>;
type Frame = Readonly<{
  source: string;
  file: string;
  line: number;
  column: number;
  endLine: number;
  endColumn: number;
  start: number;
  end: number;
  operation: string;
  synthetic: boolean;
}>;
let sources = new Map<string, Source>();
let modules = new Map<string, InstanceType<typeof TraceMap>>();

export function configureDiagnostics(entryURL: string): void {
  const base = new URL("./", entryURL);
  const index = JSON.parse(readFileSync(new URL("diagnostics/source-index.json", base), "utf8"));
  if (index.schemaVersion !== 1 || index.kind !== "can.source-index")
    throw new TypeError("invalid diagnostic index");
  const nextSources = new Map<string, Source>();
  const nextModules = new Map<string, InstanceType<typeof TraceMap>>();
  for (const source of index.sources) nextSources.set(source.id, source);
  for (const module of index.modules) {
    const url = new URL(module.path, base);
    const map = JSON.parse(readFileSync(new URL(module.path + ".map", base), "utf8"));
    nextModules.set(decodeURIComponent(url.pathname), new TraceMap(map, undefined));
  }
  sources = nextSources;
  modules = nextModules;
}
function frame(source: Source, span: Span, synthetic: boolean): Frame {
  // Columns here follow source-map/LSP UTF16 units, displayed one-based.
  return Object.freeze({
    source: source.id,
    file: source.path,
    line: span.line,
    column: span.column + 1,
    endLine: span.endLine,
    endColumn: span.endColumn + 1,
    start: span.start,
    end: span.end,
    operation: span.operation,
    synthetic,
  });
}
function safeStack(cause: unknown): string | undefined {
  try {
    if (
      (typeof cause !== "object" && typeof cause !== "function") ||
      cause === null ||
      nativeTypes.isProxy(cause) ||
      !nativeTypes.isNativeError(cause)
    )
      return;
    const descriptor = Object.getOwnPropertyDescriptor(cause, "stack");
    if (descriptor && "value" in descriptor && typeof descriptor.value === "string")
      return descriptor.value;
  } catch {
    /* Diagnostics never replace the original occurrence. */
  }
}
export function diagnosticFrames(cause: unknown, origin: FailureOrigin): readonly Frame[] {
  const frames: Frame[] = [];
  const stack = safeStack(cause);
  if (stack)
    for (const text of stack.split("\n").slice(1, 65)) {
      const match = /^\s+at (?:.* \()?(.+):(\d+):(\d+)\)?$/.exec(text);
      if (!match) continue;
      const map = modules.get(match[1]);
      if (!map) continue; // Drop native and synthetic runtime paths and frame names.
      const mapped = originalPositionFor(map, {
        line: Number(match[2]),
        column: Number(match[3]) - 1,
      });
      const source = sources.get(mapped.source ?? ""),
        span = source?.spans[mapped.name ?? ""];
      if (source && span) frames.push(frame(source, span, false));
    }
  const source = sources.get(origin.source);
  const span =
    source &&
    Object.values(source.spans).find((s) => s.start === origin.start && s.end === origin.end);
  if (
    source &&
    span &&
    !frames.some((f) => f.source === source.id && f.start === span.start && f.end === span.end)
  )
    frames.unshift(frame(source, span, true));
  return Object.freeze(frames);
}
