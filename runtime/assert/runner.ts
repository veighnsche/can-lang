import { types as nativeTypes } from "node:util";
import { invoke, success, type Completion } from "../completion.ts";
import { dataArray, dataKeys, dataProperty, recordIdentity } from "../data.ts";
import { domainFailureDiagnostics } from "../domain.ts";
import { standardFailureDiagnostics, type FailureOrigin } from "../failure.ts";
import { assertionContext, closeContext, contextReport, type AssertionContext, type AssertionRoot } from "./context.ts";

import { diagnosticFrames } from "../diagnostics.ts";

export type AssertionCase = Readonly<{
  root: AssertionRoot;
  actual: (context: AssertionContext) => Promise<Completion>;
  expected: (context: AssertionContext) => Promise<Completion>;
}>;
const origin: FailureOrigin = Object.freeze({source: "can:assertion", start: 0, end: 0, invocation: Object.freeze([])});

// Native strict deepEquals owns ordinary equality. Its one harness adapter
// protects opaque/function token identity: two empty opaque tokens must never
// compare as equal merely because they have no enumerable fields.
function opaqueIdentitiesAgree(left: unknown, right: unknown, seen = new WeakMap<object, WeakSet<object>>()): boolean {
  if (Object.is(left, right)) return true;
  if (left === null || right === null) return false;
  const object = (v: unknown) => typeof v === "object" || typeof v === "function";
  if (!object(left) || !object(right)) return object(left) === object(right);
  if (nativeTypes.isProxy(left) || nativeTypes.isProxy(right)) return false;
  const aObject = left as object, bObject = right as object;
  let pairs = seen.get(aObject);
  if (pairs?.has(bObject)) return true;
  if (!pairs) { pairs = new WeakSet(); seen.set(aObject, pairs); }
  pairs.add(bObject);
  if (Array.isArray(left) || Array.isArray(right)) {
    if (!Array.isArray(left) || !Array.isArray(right)) return false;
    const a = dataArray(left), b = dataArray(right);
    return a.length === b.length && a.every((value, i) => opaqueIdentitiesAgree(value, b[i], seen));
  }
  const identity = recordIdentity(left);
  if (!identity || identity !== recordIdentity(right)) return false;
  const a = dataKeys(left).filter(k => typeof k === "string"), b = dataKeys(right).filter(k => typeof k === "string");
  return a.length === b.length && a.every(key => b.includes(key) && opaqueIdentitiesAgree(dataProperty(left, key), dataProperty(right, key), seen));
}
export function assertionEqual(left: unknown, right: unknown): boolean {
  return Object.is(left, right) || opaqueIdentitiesAgree(left, right) && Bun.deepEquals(left, right, true);
}
function sameCompletion(actual: Completion, expected: Completion): boolean {
  if (actual.kind !== expected.kind || actual.kind === "standard" || expected.kind === "standard") return false;
  if (actual.kind === "ok" && expected.kind === "ok") return assertionEqual(actual.value, expected.value);
  if (actual.kind === "domain" && expected.kind === "domain") {
    const a = domainFailureDiagnostics(actual.value), b = domainFailureDiagnostics(expected.value);
    return a.typeIdentity === b.typeIdentity && assertionEqual(a.payload, b.payload);
  }
  return false;
}
export async function runAssertion(test: AssertionCase) {
  const context = assertionContext(test.root);
  let frames: ReturnType<typeof diagnosticFrames> = [];
  let reason: "expected evaluation failed" | "outcome mismatch" | "harness violation" | undefined;
  try {
    const expected = await invoke(() => test.expected(context), origin);
    if (expected.kind === "standard") {reason = "expected evaluation failed";const details=standardFailureDiagnostics(expected.value);frames=diagnosticFrames(details.cause,details.origin);}
    else {
      const actual = await invoke(() => test.actual(context), origin);
      if (!sameCompletion(actual, expected)) {
        reason = "outcome mismatch";
        if (actual.kind === "standard") {const details=standardFailureDiagnostics(actual.value);frames=diagnosticFrames(details.cause,details.origin);}
        else if (actual.kind === "domain") {const details=domainFailureDiagnostics(actual.value);frames=diagnosticFrames(undefined,details.origin);}
      }
    }
  } catch { reason = "outcome mismatch"; }
  finally { closeContext(context); }
  const report = contextReport(context);
  if (report.violations.length) reason = "harness violation";
  return Object.freeze({...report, passed: reason === undefined, ...(reason ? {reason,frames} : {})});
}
export async function runAssertions(tests: readonly AssertionCase[], initialize: () => void): Promise<0 | 1> {
  const setup = await invoke(() => {initialize(); return success(undefined);}, origin);
  // Use a real box for setup, just like source functions. The initialized state
  // is private and contains no effectful source initialization.
  try { return await finishSuite(tests, setup); }
  catch {
    // Report delivery failure is nonzero, without an uncaught native stack or
    // an alternate output channel that could disclose private runtime paths.
    return 1;
  }
}
async function finishSuite(tests: readonly AssertionCase[], setup: Completion): Promise<0 | 1> {
  if (setup.kind !== "ok") {
    await Bun.write(Bun.stdout, JSON.stringify({schemaVersion: 1, kind: "can.assertion-report", passed: false, reason: "initialization failed", assertions: []}) + "\n");
    return 1;
  }
  const assertions = [];
  for (const test of tests) assertions.push(await runAssertion(test));
  const passed = assertions.length !== 0 && assertions.every(result => result.passed);
  await Bun.write(Bun.stdout, JSON.stringify({schemaVersion: 1, kind: "can.assertion-report", passed, assertions}) + "\n");
  return passed ? 0 : 1;
}
