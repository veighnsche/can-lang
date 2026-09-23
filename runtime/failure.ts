// Compiler-private failure occurrences. Can projections never contain native
// causes, stacks, diagnostic origins, or mutable host objects.
import { types as nativeTypes } from "node:util";
import { primitiveFailureKind, primitiveFailureMessage } from "./primitive.ts";

export type StandardKind = "arithmetic" | "bounds" | "resource_state" | "assertion" | "native_exception" | "cleanup";
export type FailureOrigin = Readonly<{ source: string; start: number; end: number; invocation: readonly string[] }>;
// Compiler-private production provenance for domain occurrences. The boundary
// selects the consuming table (native infrastructure versus emitted domain
// obligations); the operation is the statically known producing operation — a
// fetch/judge/LLM declaration for boundary machinery, otherwise unattributed.
// Classification uses this pair, never the error name alone.
export type FailureProvenance = Readonly<{ boundary: "native" | "emitted"; operation: string }>;
declare const standardFailureBrand: unique symbol;
export type StandardFailure = Readonly<{ readonly [standardFailureBrand]: true }>;
type StandardDetails = Readonly<{ occurrenceID: bigint; kind: StandardKind; message: string; cause: unknown; origin: FailureOrigin; boundaryOrigin?: FailureOrigin }>;
const standard = new WeakMap<object, StandardDetails>();
let nextOccurrence = 1n;
export function allocateOccurrenceID(): bigint { return nextOccurrence++; }
const objectLike = (value: unknown): value is object => value !== null && (typeof value === "object" || typeof value === "function");
function freezeOrigin(origin: FailureOrigin): FailureOrigin {
  return Object.freeze({ source: origin.source, start: origin.start, end: origin.end, invocation: Object.freeze([...origin.invocation]) });
}
export function freezeProvenance(provenance?: unknown): FailureProvenance {
  if (provenance === undefined) return Object.freeze({ boundary: "emitted", operation: "" });
  if (typeof provenance !== "object" || provenance === null) throw new TypeError("invalid failure provenance");
  const { boundary, operation } = provenance as Partial<FailureProvenance>;
  if ((boundary !== "native" && boundary !== "emitted") || typeof operation !== "string") throw new TypeError("invalid failure provenance");
  if (boundary === "native" && operation === "") throw new TypeError("native failure requires its producing operation");
  return Object.freeze({ boundary, operation });
}

const nativeErrorNames: readonly (readonly [object, string])[] = [
  [Error.prototype, "Error"], [TypeError.prototype, "TypeError"], [RangeError.prototype, "RangeError"],
  [ReferenceError.prototype, "ReferenceError"], [SyntaxError.prototype, "SyntaxError"],
  [URIError.prototype, "URIError"], [EvalError.prototype, "EvalError"], [AggregateError.prototype, "AggregateError"],
];
function prototypeErrorName(value: object): string {
  let prototype = Object.getPrototypeOf(value);
  // Inspect identities only, never constructor/name properties. Stop before a
  // proxy can receive getPrototypeOf or any descriptor operation.
  for (let depth = 0; prototype !== null && depth < 64; depth++) {
    if (nativeTypes.isProxy(prototype)) return "Error";
    for (const [known, name] of nativeErrorNames) if (prototype === known) return name;
    prototype = Object.getPrototypeOf(prototype);
  }
  return "Error";
}
export function describeNativeFailure(value: unknown): string {
  try {
    if (value === null) return "null";
    switch (typeof value) {
      case "string": return value;
      case "number": case "bigint": case "boolean": case "undefined": return String(value);
      case "symbol": return "native symbol";
    }
    if (nativeTypes.isProxy(value)) return "native proxy";
    if (!nativeTypes.isNativeError(value)) return typeof value === "function" ? "native function" : "native object";
    const nameDescriptor = Object.getOwnPropertyDescriptor(value, "name");
    const messageDescriptor = Object.getOwnPropertyDescriptor(value, "message");
    const name = nameDescriptor && "value" in nameDescriptor && typeof nameDescriptor.value === "string" ? nameDescriptor.value : prototypeErrorName(value);
    const message = messageDescriptor && "value" in messageDescriptor && typeof messageDescriptor.value === "string" ? messageDescriptor.value : "";
    return name && message ? name + ": " + message : name || message;
  } catch {
    return "native failure";
  }
}
function createStandard(kind: StandardKind, message: string, cause: unknown, origin: FailureOrigin): StandardFailure {
  const occurrence = Object.freeze(Object.create(null));
  standard.set(occurrence, Object.freeze({ occurrenceID: allocateOccurrenceID(), kind, message, cause, origin: freezeOrigin(origin) }));
  return occurrence;
}

export function captureStandard(cause: unknown, origin: FailureOrigin): StandardFailure {
  if (objectLike(cause) && standard.has(cause)) {
    const details = standard.get(cause)!;
    // Runtime-created synthetic origins have no authored span. Retain their
    // original metadata and identity while recording the first checked boundary.
    if (details.origin.source.startsWith("can:") && !origin.source.startsWith("can:") && origin.source !== "" && !details.boundaryOrigin) {
      standard.set(cause, Object.freeze({...details, boundaryOrigin: freezeOrigin(origin)}));
    }
    return cause as StandardFailure;
  }
  const kind = primitiveFailureKind(cause);
  if (kind) return createStandard(kind, primitiveFailureMessage(cause)!, cause, origin);
  return createStandard("native_exception", describeNativeFailure(cause), cause, origin);
}
export function isStandardFailure(value: unknown): value is StandardFailure { return objectLike(value) && standard.has(value); }

// Only maintained adapters can choose these finite Can-created categories.
export function resourceStateFailure(cause: unknown, origin: FailureOrigin): StandardFailure {
  return createStandard("resource_state", "resource_state: invalid resource use", cause, origin);
}
export function cleanupFailure(cause: unknown, origin: FailureOrigin): StandardFailure {
  return createStandard("cleanup", "cleanup: automatic resource cleanup failed", cause, origin);
}
export function nonfiniteSortKeyFailure(cause: unknown, origin: FailureOrigin): StandardFailure {
  return createStandard("arithmetic", "arithmetic: nonfinite sort key", cause, origin);
}
export const assertionFailureClasses = Object.freeze(["missing fixture", "argument mismatch", "ambiguous fixture", "malformed fixture", "unexpected live boundary", "unused fixture", "outcome mismatch"] as const);
export function assertionFailure(failureClass: typeof assertionFailureClasses[number], origin: FailureOrigin): StandardFailure {
  if (!(assertionFailureClasses as readonly string[]).includes(failureClass)) throw new TypeError("unknown assertion failure class");
  return createStandard("assertion", "assertion: " + failureClass, undefined, origin);
}
function details(value: StandardFailure): StandardDetails {
  const found = objectLike(value) ? standard.get(value) : undefined;
  if (!found) throw new TypeError("invalid standard failure occurrence");
  return found;
}
export function standardFailureKind(value: StandardFailure): StandardKind { return details(value).kind; }
export function standardFailureMessage(value: StandardFailure): string { return details(value).message; }
export function standardFailureOccurrenceID(value: StandardFailure): bigint { return details(value).occurrenceID; }

// Diagnostic-only access for the private supervisor. Never lower this function
// as an authored projection or serialize its returned metadata into a message.
export function standardFailureDiagnostics(value: StandardFailure): StandardDetails { return details(value); }
