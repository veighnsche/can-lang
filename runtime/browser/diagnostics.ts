// Browser-profile diagnostics. Source locations come from a sealed
// build-generated table configured once before startup; no filesystem or
// fetches at fault time. Failures report once per occurrence through the
// sealed record below: category, phase and Can file/line/column only. Native
// causes, raw input, stacks and secret bytes never cross this boundary.
import { checkedCompletion, type Completion } from "../completion.ts";
import { configureFromData, locateOrigin } from "../diagnostics-core.ts";
import { domainFailureDiagnostics, isDomainFailure } from "../domain-core.ts";
import { claimFailureReport, isStandardFailure, standardFailureDiagnostics } from "../failure.ts";

export type {
  DiagnosticFrame,
  DiagnosticSource,
  DiagnosticSourceIndex,
  DiagnosticSourceIndexModule,
  DiagnosticSpan,
} from "../diagnostics-core.ts";
export { configureFromData, diagnosticFrames, locateOrigin } from "../diagnostics-core.ts";

export type SealedDiagnosticTable = Readonly<{
  index: unknown;
  maps: Readonly<Record<string, unknown>>;
}>;

export function configureBrowserDiagnostics(table: SealedDiagnosticTable): void {
  if (table === null || typeof table !== "object") throw new TypeError("invalid diagnostic table");
  const { index, maps } = table;
  if (maps === null || typeof maps !== "object" || Array.isArray(maps))
    throw new TypeError("invalid diagnostic table");
  const entries = maps as Record<string, unknown>;
  configureFromData(
    index,
    (modulePath) => entries[modulePath],
    (modulePath) => modulePath,
  );
}

export type BrowserDiagnosticPhase = "startup" | "handler";
export type BrowserDiagnostic = Readonly<{
  kind: "can.runtime-diagnostic";
  phase: BrowserDiagnosticPhase;
  occurrence: string;
  category: string;
  file: string;
  line: number;
  column: number;
}>;

const reported = new WeakSet<object>();

// Successful completions report nothing and return undefined. Failures yield
// one frozen record per occurrence through the default console.error sink.
export function reportBrowserDiagnostic(
  completion: Completion,
  phase: BrowserDiagnosticPhase,
): BrowserDiagnostic | undefined {
  checkedCompletion(completion);
  if (completion.kind === "ok") return undefined;
  const occurrence = completion.value as object;
  if (reported.has(occurrence)) return undefined;
  reported.add(occurrence);
  // The shared occurrence claim runs second: an occurrence already
  // delivered by the main, late-owner, or request reporter stays single.
  if (!claimFailureReport(occurrence)) return undefined;
  let category: string;
  let identity: string;
  let file = "";
  let line = 0;
  let column = 0;
  if (completion.kind === "domain" && isDomainFailure(completion.value)) {
    const details = domainFailureDiagnostics(completion.value);
    category = "domain";
    identity = String(details.occurrenceID);
    const located = locateOrigin(details.origin);
    if (located) ({ file, line, column } = located);
  } else if (completion.kind === "standard" && isStandardFailure(completion.value)) {
    const details = standardFailureDiagnostics(completion.value);
    category = details.kind;
    identity = String(details.occurrenceID);
    const located = locateOrigin(details.boundaryOrigin ?? details.origin);
    if (located) ({ file, line, column } = located);
  } else {
    throw new TypeError("invalid browser failure occurrence");
  }
  const record = Object.freeze({
    kind: "can.runtime-diagnostic" as const,
    phase,
    occurrence: identity,
    category,
    file,
    line,
    column,
  });
  console.error(record);
  return record;
}
