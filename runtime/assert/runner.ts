import { runOwnedRoot, type OwnerDiagnostic } from "../owner.ts";
import { isBytes, copyBytes } from "../bytes.ts";
import { types as nativeTypes } from "node:util";
import { callableEqual } from "../callable.ts";
import { invoke, success, type Completion } from "../completion.ts";
import { dataArray, dataKeys, dataProperty, recordIdentity } from "../data.ts";
import { domainFailureDiagnostics } from "../domain.ts";
import { standardFailureDiagnostics, type FailureOrigin } from "../failure.ts";
import {
  assertionContext,
  finishAssertionExecution,
  closeContext,
  contextReport,
  contextProgress,
  type AssertionContext,
  type AssertionRoot,
} from "./context.ts";

import { diagnosticFrames } from "../diagnostics.ts";

export type AssertionCase = Readonly<{
  root: AssertionRoot;
  actual: (context: AssertionContext) => Promise<Completion>;
  expected: (context: AssertionContext) => Promise<Completion>;
}>;
const origin: FailureOrigin = Object.freeze({
  source: "can:assertion",
  start: 0,
  end: 0,
  invocation: Object.freeze([]),
});

// Native strict deepEquals owns ordinary equality. Its one harness adapter
// protects opaque/function token identity: two empty opaque tokens must never
// compare as equal merely because they have no enumerable fields.
function opaqueIdentitiesAgree(
  left: unknown,
  right: unknown,
  seen = new WeakMap<object, WeakSet<object>>(),
  content = false,
): boolean {
  if (Object.is(left, right)) return true;
  // Completion-level comparison may match two genuine byte tokens by content
  // so decoder output can meet constructed expectations. Anything else
  // facing a byte token still fails: content comparison never forges an
  // opaque value from visible data. Argument matching stays identity-strict.
  if (content && (isBytes(left) || isBytes(right))) {
    if (!isBytes(left) || !isBytes(right)) return false;
    const a = copyBytes(left, origin),
      b = copyBytes(right, origin);
    return a.length === b.length && a.every((byte, index) => byte === b[index]);
  }
  if (left === null || right === null) return false;
  const object = (v: unknown) => typeof v === "object" || typeof v === "function";
  if (!object(left) || !object(right)) return object(left) === object(right);
  if (nativeTypes.isProxy(left) || nativeTypes.isProxy(right)) return false;
  const aObject = left as object,
    bObject = right as object;
  let pairs = seen.get(aObject);
  if (pairs?.has(bObject)) return true;
  if (!pairs) {
    pairs = new WeakSet();
    seen.set(aObject, pairs);
  }
  pairs.add(bObject);
  if (Array.isArray(left) || Array.isArray(right)) {
    if (!Array.isArray(left) || !Array.isArray(right)) return false;
    const a = dataArray(left),
      b = dataArray(right);
    return (
      a.length === b.length &&
      a.every((value, i) => opaqueIdentitiesAgree(value, b[i], seen, content))
    );
  }
  const identity = recordIdentity(left);
  if (!identity || identity !== recordIdentity(right)) return false;
  const a = dataKeys(left).filter((k) => typeof k === "string"),
    b = dataKeys(right).filter((k) => typeof k === "string");
  return (
    a.length === b.length &&
    a.every(
      (key) =>
        b.includes(key) &&
        opaqueIdentitiesAgree(dataProperty(left, key), dataProperty(right, key), seen, content),
    )
  );
}
export function assertionEqual(left: unknown, right: unknown): boolean {
  if (Object.is(left, right)) return true;
  // Owned callables compare by receipt (target plus captures): distinct
  // functions are never deep-equal, but the same `callable name` named by
  // a call and its fixture row is one callable.
  if (typeof left === "function" || typeof right === "function")
    return callableEqual(left, right, assertionEqual);
  // Argument lists compare element-wise so callable arguments reach receipt
  // equality; whole-array deep equality would reject distinct functions.
  if (Array.isArray(left) || Array.isArray(right)) {
    if (!Array.isArray(left) || !Array.isArray(right) || left.length !== right.length) return false;
    return left.every((value, index) => assertionEqual(value, right[index]));
  }
  return opaqueIdentitiesAgree(left, right) && Bun.deepEquals(left, right, true);
}
function isOpaqueToken(value: unknown): boolean {
  if (value === null || (typeof value !== "object" && typeof value !== "function")) return false;
  if (Array.isArray(value) || nativeTypes.isProxy(value)) return false;
  return recordIdentity(value) === undefined;
}
export function completionValueEqual(left: unknown, right: unknown): boolean {
  if (Object.is(left, right)) return true;
  if (typeof left === "function" || typeof right === "function")
    return callableEqual(left, right, completionValueEqual);
  if (Array.isArray(left) || Array.isArray(right)) {
    if (!Array.isArray(left) || !Array.isArray(right) || left.length !== right.length) return false;
    return left.every((value, index) => completionValueEqual(value, right[index]));
  }
  return (
    opaqueIdentitiesAgree(left, right, new WeakMap(), true) && Bun.deepEquals(left, right, true)
  );
}
export function completionEqual(actual: Completion, expected: Completion): boolean {
  if (actual.kind !== expected.kind || actual.kind === "standard" || expected.kind === "standard")
    return false;
  // Bare ok expectations (checker-confined to C-excluded opaque results) pass
  // against any opaque token: fixture structure and completion kinds are the
  // verified content, never the unobservable interior.
  if (
    actual.kind === "ok" &&
    expected.kind === "ok" &&
    expected.value === undefined &&
    isOpaqueToken(actual.value)
  )
    return true;
  if (actual.kind === "ok" && expected.kind === "ok")
    return completionValueEqual(actual.value, expected.value);
  if (actual.kind === "domain" && expected.kind === "domain") {
    const a = domainFailureDiagnostics(actual.value),
      b = domainFailureDiagnostics(expected.value);
    return a.typeIdentity === b.typeIdentity && completionValueEqual(a.payload, b.payload);
  }
  return false;
}
// ProgressSink receives best-effort progress records while a root runs. A
// throwing sink never fails the root; a blocked worker simply emits nothing
// further and the supervisor marks its last record stale.
export type ProgressSink = (record: Record<string, unknown>) => void;
export async function runAssertion(test: AssertionCase, sink?: ProgressSink) {
  const context = assertionContext(test.root);
  let frames: ReturnType<typeof diagnosticFrames> = [];
  let reason: "expected evaluation failed" | "outcome mismatch" | "harness violation" | undefined;
  const diagnostics: OwnerDiagnostic[] = [];
  const beat =
    sink === undefined
      ? undefined
      : setInterval(() => {
          try {
            sink({ phase: "running", progress: contextProgress(context) });
          } catch {}
        }, 250);
  try {
    const owned = await runOwnedRoot(
      async () => {
        try {
          const expected = await invoke(() => test.expected(context), origin);
          if (expected.kind === "standard") {
            reason = "expected evaluation failed";
            const details = standardFailureDiagnostics(expected.value);
            frames = diagnosticFrames(details.cause, details.boundaryOrigin ?? details.origin);
          } else {
            const actual = await invoke(() => test.actual(context), origin);
            if (!completionEqual(actual, expected)) {
              reason = "outcome mismatch";
              if (actual.kind === "standard") {
                const details = standardFailureDiagnostics(actual.value);
                frames = diagnosticFrames(details.cause, details.boundaryOrigin ?? details.origin);
              } else if (actual.kind === "domain") {
                const details = domainFailureDiagnostics(actual.value);
                frames = diagnosticFrames(undefined, details.origin);
              }
            }
          }
          return success(undefined);
        } finally {
          finishAssertionExecution(context);
        }
      },
      (diagnostic) => {
        diagnostics.push(diagnostic);
      },
    );
    if (owned.cleanupFailed) reason = "harness violation";
  } catch {
    reason = "outcome mismatch";
  } finally {
    if (beat !== undefined) clearInterval(beat);
    closeContext(context);
  }
  const report = contextReport(context);
  if (report.violations.length) {
    reason = "harness violation";
    if (report.frames.length) frames = report.frames;
  }
  return Object.freeze({
    ...report,
    ...(diagnostics.length ? { diagnostics: Object.freeze(diagnostics) } : {}),
    passed: reason === undefined,
    ...(reason ? { reason, frames } : {}),
  });
}
// runAssertionRoot executes one selected root in a fresh worker. argv carries
// exactly one selector of the form root=<index> into tests. It prints one
// can.assertion-root-report object to stdout and returns 0/1 by delivery;
// exit 2 without a report means the selector itself was unusable. Progress
// records travel on stderr behind the CAN-PROGRESS prefix; every other
// stderr byte is worker diagnostic, never part of the report protocol.
export async function runAssertionRoot(
  tests: readonly AssertionCase[],
  initialize: () => void,
  argv: readonly string[],
): Promise<number> {
  const emit = async (record: Record<string, unknown>): Promise<void> => {
    try {
      await Bun.write(Bun.stderr, "CAN-PROGRESS " + JSON.stringify(record) + "\n");
    } catch {}
  };
  const selection = argv.length === 1 ? /^root=(\d+)$/.exec(argv[0] ?? "") : null;
  const index = selection === null ? NaN : Number(selection[1]);
  if (!Number.isInteger(index) || index < 0 || index >= tests.length) {
    await emit({
      version: 1,
      phase: "protocol-error",
      detail: "root selector must be root=<index>",
    });
    return 2;
  }
  const test = tests[index];
  // Use a real box for setup, just like source functions. The initialized state
  // is private and contains no effectful source initialization.
  try {
    await emit({ version: 1, phase: "started", root: test.root });
    const setup = await invoke(() => {
      initialize();
      return success(undefined);
    }, origin);
    if (setup.kind !== "ok") {
      const details =
        setup.kind === "standard"
          ? standardFailureDiagnostics(setup.value)
          : domainFailureDiagnostics(setup.value);
      const boundary = "boundaryOrigin" in details ? details.boundaryOrigin : undefined;
      const frames = diagnosticFrames(details.cause, boundary ?? details.origin);
      await Bun.write(
        Bun.stdout,
        JSON.stringify({
          schemaVersion: 1,
          kind: "can.assertion-root-report",
          root: test.root,
          passed: false,
          reason: "initialization failed",
          frames,
        }) + "\n",
      );
      return 1;
    }
    const assertion = await runAssertion(
      test,
      (record) => void emit({ version: 1, root: test.root, ...record }),
    );
    const passed = (assertion as { passed: boolean }).passed === true;
    await emit({ version: 1, phase: "finished", root: test.root, passed });
    await Bun.write(
      Bun.stdout,
      JSON.stringify({
        schemaVersion: 1,
        kind: "can.assertion-root-report",
        root: test.root,
        passed,
        assertion,
      }) + "\n",
    );
    return passed ? 0 : 1;
  } catch {
    // Report delivery failure is nonzero, without an uncaught native stack or
    // an alternate output channel that could disclose private runtime paths.
    return 1;
  }
}
