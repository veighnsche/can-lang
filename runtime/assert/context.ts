// Compiler-private assertion state travels explicitly through generated calls.
// No process-global active root can leak across independent asynchronous runs.
import { assertionFailure, standardFailureDiagnostics, type FailureOrigin, type StandardFailure } from "../failure.ts";

import { diagnosticFrames } from "../diagnostics.ts";

export type AssertionRoot = Readonly<{package: string; declaration: string; name: string}>;
declare const contextBrand: unique symbol;
export type AssertionContext = Readonly<{readonly [contextBrand]: true}>;
type Violation = "missing fixture" | "argument mismatch" | "ambiguous fixture" | "malformed fixture" | "unexpected live boundary" | "unused fixture";
type State = {root: AssertionRoot; violations: Violation[]; failures: StandardFailure[]; evidence: Set<"real-can" | "supplied-completion">; closed: boolean; tables: Map<string, {used: number; total: number; origin: FailureOrigin}>};
const contexts = new WeakMap<object, State>();
export function assertionContext(root: AssertionRoot): AssertionContext {
  if (!root.package || !root.declaration || !root.name) throw new TypeError("incomplete assertion root");
  const context = Object.freeze(Object.create(null));
  contexts.set(context, {root: Object.freeze({...root}), violations: [], failures: [], evidence: new Set(["real-can"]), closed: false, tables: new Map()});
  return context;
}
function state(context: AssertionContext): State {
  const value = context !== null && (typeof context === "object" || typeof context === "function") ? contexts.get(context) : undefined;
  if (!value) throw new TypeError("invalid assertion context");
  return value;
}
export function violation(context: AssertionContext, reason: Violation, origin: FailureOrigin): StandardFailure {
  const current=state(context), failure=assertionFailure(reason, origin);
  current.violations.push(reason);
  current.failures.push(failure);
  return failure;
}
export function denyLiveBoundary(context: AssertionContext | undefined, origin: FailureOrigin): void {
  if (context === undefined) return;
  const current = state(context);
  throw violation(context, current.closed ? "unexpected live boundary" : "missing fixture", origin);
}
export function suppliedEvidence(context: AssertionContext): void { state(context).evidence.add("supplied-completion"); }
export function contextReport(context: AssertionContext) {
  const current = state(context);
  const frames=Object.freeze(current.failures.flatMap(failure=>{
    const details=standardFailureDiagnostics(failure);
    return diagnosticFrames(details.cause,details.boundaryOrigin ?? details.origin);
  }));
  return Object.freeze({root: current.root, frames, violations: Object.freeze([...current.violations]), evidence: Object.freeze([...current.evidence].sort())});
}
export function fixtureIndex(context: AssertionContext, table: string, total: number, origin: FailureOrigin): number {
  const current = state(context);
  if (current.closed) throw violation(context, "unexpected live boundary", origin);
  let queue = current.tables.get(table);
  if (!queue) { queue = {used: 0, total, origin}; current.tables.set(table, queue); }
  if (queue.total !== total) throw violation(context, "ambiguous fixture", origin);
  if (queue.used >= queue.total) throw violation(context, "missing fixture", origin);
  return queue.used++;
}
export function closeContext(context: AssertionContext): void {
  const current = state(context);
  for (const queue of current.tables.values()) if (queue.used !== queue.total) violation(context,"unused fixture",queue.origin);
  current.closed = true;
}
