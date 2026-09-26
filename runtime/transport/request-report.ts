// Automatic redacted request-failure reporting (E06, R12 O2 builtin_auto).
//
// The server request boundary reports one sanitized diagnostic per
// unexpected failure with correlation/source identity. Redaction reuses
// the entry policy: domain failures identify by qualified declaration
// identity plus concrete type identity with a redacted payload; standard
// failures identify by category plus occurrence. Native messages, paths,
// query/header/body bytes, and secrets never serialize into a report.
// Client responses stay fixed 500s; reporting never breaks the boundary.
//
// Correlation is minted per native request from the C-G identity
// vocabulary (lane F); no client-supplied value is honored, so request
// bytes cannot enter a report through the correlation field. Source
// names the dispatch point from authored metadata only: the fixed
// fallback, an action identity from the checked route table, or a
// validated static route mount. Exactly-once-per-occurrence discipline
// is shared with the main, late-owner, and browser reporters through
// claimFailureReport: an occurrence already delivered elsewhere is
// skipped here. The terminal main reporter is the exception: it always
// delivers and only marks, so a repeated main failure still reports.
//
// This module is host-free (no Bun/process/node edges): the JSON-action
// adapter imports it on a browser-reachable path, so the default sink is
// console.error. Expected failures (client rejections, handled domain
// outcomes) stay silent here; authors log those manually per the
// request-failure-reporting guide.
import { caught, isCompletion, type Completion } from "../completion.ts";
import { diagnosticFrames, type DiagnosticFrame } from "../diagnostics-core.ts";
import { domainFailureDiagnostics, isDomainFailure } from "../domain-core.ts";
import {
  claimFailureReport,
  isStandardFailure,
  standardFailureDiagnostics,
  type FailureOrigin,
} from "../failure.ts";
import { newCorrelationId, type CorrelationId } from "../outbound/identity.ts";

const origin: FailureOrigin = Object.freeze({
  source: "can:request-report",
  start: 0,
  end: 0,
  invocation: Object.freeze([]),
});

export type RequestReportSink = (line: string) => void | Promise<void>;
export type RequestReportContext = Readonly<{
  correlation: CorrelationId;
  source: string;
}>;
export type RequestReport = Readonly<
  | Readonly<{
      schemaVersion: 1;
      kind: "can.runtime-failure";
      phase: "request";
      channel: "domain";
      identity: string;
      error: string;
      typeIdentity: string;
      occurrence: string;
      payload: "<redacted>";
      correlation: string;
      source: string;
      frames: readonly DiagnosticFrame[];
    }>
  | Readonly<{
      schemaVersion: 1;
      kind: "can.runtime-failure";
      phase: "request";
      channel: "standard";
      category: string;
      occurrence: string;
      correlation: string;
      source: string;
      frames: readonly DiagnosticFrame[];
    }>
>;

// The delivered line carries no trailing newline; sinks add framing.
export const defaultRequestReportSink: RequestReportSink = (line) => {
  console.error(line);
};
let sink: RequestReportSink = defaultRequestReportSink;
export function setRequestReportSink(next: RequestReportSink): void {
  if (typeof next !== "function") throw new TypeError("invalid request report sink");
  sink = next;
}

const correlations = new WeakMap<object, CorrelationId>();
const sources = new WeakMap<object, string>();

// correlationFor mints the request correlation once per native request.
// Every layer reporting for one request resolves the same identity.
export function correlationFor(native: Request): CorrelationId {
  let id = correlations.get(native);
  if (id === undefined) {
    id = newCorrelationId();
    correlations.set(native, id);
  }
  return id;
}

// Sources stay log-line safe: well-formed text without C0 controls or
// DEL. Anything else is ignored so a bad annotator cannot poison the
// record or inject a second line; reads then fall back to "http".
function checkedSource(source: string): string | undefined {
  if (
    typeof source !== "string" ||
    !source.isWellFormed() ||
    source.length === 0 ||
    source.length > 512
  )
    return undefined;
  for (let i = 0; i < source.length; i++) {
    const code = source.charCodeAt(i);
    if (code < 0x20 || code === 0x7f) return undefined;
  }
  return source;
}

function keyed(native: unknown): native is object {
  return native !== null && (typeof native === "object" || typeof native === "function");
}

// noteRequestSource records the dispatch point for the boundary report:
// "action:<identity>" from the checked table or
// "route:<METHOD> <static template>" from the validated mount. Both are
// authored metadata, never request bytes.
export function noteRequestSource(native: Request | undefined, source: string): void {
  if (!keyed(native)) return;
  const checked = checkedSource(source);
  if (checked === undefined) return;
  sources.set(native, checked);
}

export function requestSource(native: Request | undefined): string {
  if (!keyed(native)) return "http";
  return sources.get(native) ?? "http";
}

// requestContext resolves the report identity for one native request.
// An unkeyable request (revoked before adaptation) still reports under
// a fresh correlation rather than dropping the record.
export function requestContext(native: Request | undefined): RequestReportContext {
  if (!keyed(native)) return Object.freeze({ correlation: newCorrelationId(), source: "http" });
  return Object.freeze({ correlation: correlationFor(native), source: requestSource(native) });
}

function normalizedContext(context: RequestReportContext): RequestReportContext {
  const correlation =
    typeof context?.correlation === "string" && context.correlation.length !== 0
      ? context.correlation
      : newCorrelationId();
  const source =
    typeof context?.source === "string" ? (checkedSource(context.source) ?? "http") : "http";
  return Object.freeze({ correlation: correlation as CorrelationId, source });
}

function formatReport(
  completion: Exclude<Completion, { kind: "ok" }>,
  context: RequestReportContext,
): RequestReport {
  const base = {
    schemaVersion: 1 as const,
    kind: "can.runtime-failure" as const,
    phase: "request" as const,
  };
  if (completion.kind === "domain" && isDomainFailure(completion.value)) {
    const details = domainFailureDiagnostics(completion.value);
    return Object.freeze({
      ...base,
      channel: "domain" as const,
      identity: "can.error.v2:" + details.declaration.identity,
      error: details.declaration.name,
      typeIdentity: details.typeIdentity,
      occurrence: String(details.occurrenceID),
      payload: "<redacted>" as const,
      correlation: context.correlation as string,
      source: context.source,
      frames: diagnosticFrames(undefined, details.origin),
    });
  }
  if (completion.kind === "standard" && isStandardFailure(completion.value)) {
    const details = standardFailureDiagnostics(completion.value);
    return Object.freeze({
      ...base,
      channel: "standard" as const,
      category: details.kind,
      occurrence: String(details.occurrenceID),
      correlation: context.correlation as string,
      source: context.source,
      frames: diagnosticFrames(details.cause, details.boundaryOrigin ?? details.origin),
    });
  }
  throw new TypeError("invalid request failure occurrence");
}

// reportRequestFailure delivers the single redacted record for an
// unexpected failure: a failed Completion from handler, adaptation, or
// drain work, or a value that escaped those layers unboxed (raw domain
// and standard occurrences keep their identity; anything else boxes as
// a fresh native_exception). Successful completions report nothing. The
// shared occurrence claim runs before delivery, so a failure observed
// by several layers or reporters still delivers once. Reporting never
// throws and never rejects: a broken sink yields undefined, never a
// second failure at the boundary.
export async function reportRequestFailure(
  input: unknown,
  context: RequestReportContext,
): Promise<RequestReport | undefined> {
  try {
    const completion = isCompletion(input) ? input : caught(input, origin);
    if (completion.kind === "ok") return undefined;
    if (!claimFailureReport(completion.value as object)) return undefined;
    const record = formatReport(completion, normalizedContext(context));
    await sink(JSON.stringify(record));
    return record;
  } catch {
    return undefined;
  }
}
