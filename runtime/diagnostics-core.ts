// Private diagnostics consume only verified generation maps. Native messages,
// native frame names, source contents and host paths never cross this boundary.
// This core is canonical for both hosts: tables are configured from verified
// in-memory data, and stack text is only an optional hint on top of the
// checked origin frame.
import { isHostNativeError, isHostProxy } from "./reflect.ts";
import { TraceMap, originalPositionFor } from "./vendor/trace-mapping.ts";
import type { FailureOrigin } from "./failure.ts";

export type DiagnosticSpan = Readonly<{
  start: number;
  end: number;
  line: number;
  column: number;
  endLine: number;
  endColumn: number;
  operation: string;
}>;
export type DiagnosticSource = Readonly<{
  id: string;
  path: string;
  spans: Record<string, DiagnosticSpan>;
}>;
export type DiagnosticFrame = Readonly<{
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
export type DiagnosticSourceIndex = Readonly<{
  schemaVersion: number;
  kind: string;
  sources: readonly DiagnosticSource[];
  modules: readonly DiagnosticSourceIndexModule[];
}>;
export type DiagnosticSourceIndexModule = Readonly<{ path: string }>;
let sources = new Map<string, DiagnosticSource>();
let modules = new Map<string, InstanceType<typeof TraceMap>>();

function checkedIndex(index: unknown): asserts index is DiagnosticSourceIndex {
  if (
    index === null ||
    typeof index !== "object" ||
    (index as DiagnosticSourceIndex).schemaVersion !== 1 ||
    (index as DiagnosticSourceIndex).kind !== "can.source-index" ||
    !Array.isArray((index as DiagnosticSourceIndex).sources) ||
    !Array.isArray((index as DiagnosticSourceIndex).modules)
  )
    throw new TypeError("invalid diagnostic index");
}

export function configureFromData(
  index: unknown,
  loadMap: (modulePath: string) => unknown,
  keyOf: (modulePath: string) => string,
): void {
  checkedIndex(index);
  const nextSources = new Map<string, DiagnosticSource>();
  const nextModules = new Map<string, InstanceType<typeof TraceMap>>();
  for (const source of index.sources) nextSources.set(source.id, source);
  for (const module of index.modules) {
    const map = loadMap(module.path);
    if (map === undefined) throw new TypeError("invalid diagnostic index");
    nextModules.set(keyOf(module.path), new TraceMap(map as never, undefined));
  }
  sources = nextSources;
  modules = nextModules;
}

function frame(
  source: DiagnosticSource,
  span: DiagnosticSpan,
  synthetic: boolean,
): DiagnosticFrame {
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
      isHostProxy(cause) ||
      !isHostNativeError(cause)
    )
      return;
    const descriptor = Object.getOwnPropertyDescriptor(cause, "stack");
    if (descriptor && "value" in descriptor && typeof descriptor.value === "string")
      return descriptor.value;
  } catch {
    /* Diagnostics never replace the original occurrence. */
  }
}
export function locateOrigin(
  origin: FailureOrigin,
): { file: string; line: number; column: number } | undefined {
  if (origin === null || typeof origin !== "object") return undefined;
  const source = sources.get(origin.source);
  if (!source || source.spans === null || typeof source.spans !== "object") return undefined;
  const span = Object.values(source.spans).find(
    (s) => s.start === origin.start && s.end === origin.end,
  );
  if (!span) return undefined;
  return { file: source.path, line: span.line, column: span.column + 1 };
}
export function diagnosticFrames(
  cause: unknown,
  origin: FailureOrigin,
): readonly DiagnosticFrame[] {
  const frames: DiagnosticFrame[] = [];
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
