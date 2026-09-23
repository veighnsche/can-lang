import { test, expect } from "bun:test";
import { runAssertion, assertionEqual } from "../assert/runner.ts";
import { assertionContext, contextReport, denyLiveBoundary, closeContext } from "../assert/context.ts";
import { withFixture } from "../assert/fixtures.ts";
import { success } from "../completion.ts";
import { array, record } from "../data.ts";
import { ownBytes } from "../bytes.ts";
import { ownCallable } from "../callable.ts";

const root = Object.freeze({package: "can.project.root/app", declaration: "can.project.root/app::subject", name: "sample"});
const origin = {source: "can:test", start: 0, end: 1, invocation: []};

test("an assertion awaits and executes its real body exactly once", async () => {
  let calls = 0;
  let release!: () => void;
  const gate = new Promise<void>(resolve => {release = resolve;});
  const pending = runAssertion({root, expected: async () => success(49n), actual: async () => {
    calls++; await gate; return success(7n * 7n);
  }});
  await Promise.resolve(); await Promise.resolve();
  expect(calls).toBe(1);
  release();
  expect(await pending).toMatchObject({passed: true, root, evidence: ["real-can"]});
  expect(calls).toBe(1);
  expect(await runAssertion({root, expected: async () => success(50n), actual: async () => success(7n * 7n)})).toMatchObject({passed: false, reason: "outcome mismatch"});
});

test("caught live-boundary refusal is sticky and independent roots do not share it", async () => {
  const results = await Promise.all([
    runAssertion({root, expected: async () => success(undefined), actual: async context => {
      await Promise.resolve();
      try {denyLiveBoundary(context, origin);} catch {}
      return success(undefined);
    }}),
    runAssertion({root: {...root, declaration: root.declaration + "_other"}, expected: async () => success(undefined), actual: async () => {await Promise.resolve(); return success(undefined);}}),
  ]);
  expect(results[0]).toMatchObject({passed: false, violations: ["missing fixture"]});
  expect(results[1]).toMatchObject({passed: true, violations: []});
});

test("expected evaluation failure does not execute the subject", async () => {
  let calls = 0;
  const report = await runAssertion({root, expected: async () => {throw new Error("secret");}, actual: async () => {calls++; return success(undefined);}});
  expect(report).toMatchObject({passed: false, reason: "expected evaluation failed"});
  expect(calls).toBe(0);
  expect(JSON.stringify(report)).not.toContain("secret");
});

test("native assertion equality preserves nominal, float and opaque identities", () => {
  expect(assertionEqual(NaN, NaN)).toBe(true);
  expect(assertionEqual(0, -0)).toBe(false);
  expect(assertionEqual(record("a", [["x", 1n]]), record("b", [["x", 1n]]))).toBe(false);
  const handle = ownBytes(new Uint8Array([1]));
  expect(assertionEqual(array([handle]), array([handle]))).toBe(true);
  expect(assertionEqual(array([handle]), array([ownBytes(new Uint8Array([1]))]))).toBe(false);
  let reads = 0;
  const hostile = new Proxy({}, {get() {reads++; throw new Error("private");}, getPrototypeOf() {reads++; throw new Error("private");}});
  expect(assertionEqual(hostile, {})).toBe(false);
  expect(assertionEqual(hostile, 1n)).toBe(false);
  expect(assertionEqual(null, hostile)).toBe(false);
  expect(reads).toBe(0);
});

test("argument lists compare callables by receipt, not construction site", () => {
  const term = "Zed";
  const call = ownCallable("app::run#2", "app::decide", [term], async () => success(1n));
  const row = ownCallable("app::run#3", "app::decide", [term], async () => success(1n));
  expect(call).not.toBe(row);
  expect(assertionEqual([1n, call], [1n, row])).toBe(true);
  const other = ownCallable("app::run#4", "app::other", [term], async () => success(1n));
  expect(assertionEqual([1n, call], [1n, other])).toBe(false);
  const recaptured = ownCallable("app::run#5", "app::decide", ["Hank"], async () => success(1n));
  expect(assertionEqual([1n, call], [1n, recaptured])).toBe(false);
  expect(assertionEqual([1n, call], [1n, async () => success(1n)])).toBe(false);
  expect(assertionEqual([1n, call], [1n])).toBe(false);
  expect(assertionEqual([1n, call], [2n, row])).toBe(false);
});

test("supplied completion replaces only the exact call and labels its evidence", async () => {
  let nativeCalls = 0, continuations = 0;
  const report = await runAssertion({root, expected: async () => success(8n), actual: async context => {
    const result = await withFixture(context, "subject/call/0", [{selector: "sample", arguments: async () => success([3n]), expected: async () => success(7n)}], [3n], async () => {nativeCalls++; return success(100n);}, origin);
    if (result.kind !== "ok") return result;
    continuations++;
    return success((result.value as bigint) + 1n);
  }});
  expect(report).toMatchObject({passed: true, evidence: ["real-can", "supplied-completion"]});
  expect(nativeCalls).toBe(0);
  expect(continuations).toBe(1);
});

test("basic fixture FIFO never searches later rows and reports leftovers", async () => {
  const rows = [1n, 2n].map(value => ({selector: "sample", arguments: async () => success([value]), expected: async () => success(value)}));
  const context = assertionContext(root);
  await expect(withFixture(context, "table", rows, [2n], async () => success(0n), origin)).rejects.toBeDefined();
  closeContext(context);
  expect(contextReport(context).violations).toEqual(["argument mismatch", "unused fixture"]);
  expect(() => denyLiveBoundary(context, origin)).toThrow();
  expect(contextReport(context).violations).toContain("unexpected live boundary");
});

test("mismatch reports identify root, site and invocations without dumping values", async () => {
  const secret = "capture-secret-Zed", other = "leftover-Hank", actual = "actual-Quinn";
  const sited = {source: "can:test", start: 0, end: 1, invocation: ["can.project.root/app::subject"]};
  const context = assertionContext(root);
  const rows = [
    {selector: "sample", arguments: async () => success([secret]), expected: async () => success(1n)},
    {selector: "sample", arguments: async () => success([other]), expected: async () => success(2n)},
  ];
  await expect(withFixture(context, "table", rows, [actual], async () => success(0n), sited)).rejects.toBeDefined();
  closeContext(context);
  const report = contextReport(context);
  expect(report.root).toEqual(root);
  expect(report.violations).toEqual(["argument mismatch", "unused fixture"]);
  const [mismatch, leftover] = report.fixturePaths;
  expect(mismatch.reason).toBe("argument mismatch");
  expect(mismatch.expected).toMatchObject({table: "table", row: 0});
  expect(mismatch.actual.root).toEqual({package: root.package, declaration: root.declaration, name: root.name});
  expect(leftover.reason).toBe("unused fixture");
  expect(leftover.expected).toMatchObject({table: "table", row: 1});
  expect(mismatch.origin).toEqual(sited);
  const text = JSON.stringify(report);
  expect(text).toContain("can:test");
  expect(text).not.toContain(secret);
  expect(text).not.toContain(other);
  expect(text).not.toContain(actual);
});

test("placeholder platform origins stay out of mismatch reports", async () => {
  const { violation } = await import("../assert/context.ts");
  const context = assertionContext(root);
  const placeholder = {source: "can:cli", start: 0, end: 0, invocation: []};
  violation(context, "missing fixture", placeholder);
  const report = contextReport(context);
  expect(report.fixturePaths).toEqual([{reason: "missing fixture", expected: null, actual: report.fixturePaths[0].actual, origin: null}]);
  expect(JSON.stringify(report)).not.toContain("can:cli");
});

test("pending invocation progress reports structure without values", async () => {
  const { coordinationContexts, scheduledFixture, contextProgress } = await import("../assert/context.ts");
  const context = assertionContext(root);
  const coord = coordinationContexts(context, "p::main#0", [[0], [1]], "all");
  coord.start(0);
  let delivered = false;
  const pending = scheduledFixture(coord.contexts[0], "table", 1, origin, async () => { delivered = true; return success(1n); });
  for (let i = 0; i < 30; i++) await Promise.resolve();
  expect(delivered).toBe(false);
  const progress = contextProgress(context);
  expect(progress.pending).toBe(1);
  expect(progress.frames.some(frame => frame.phase === "fixture")).toBe(true);
  const text = JSON.stringify(progress);
  expect(text).toContain("p::main#0");
  expect(text).not.toContain("captures");
  expect(text).not.toContain("arguments");
  coord.start(1);
  const pending2 = scheduledFixture(coord.contexts[1], "other", 1, origin, async () => success(2n));
  expect((await pending).kind).toBe("ok");
  coord.observed(0, success(1n));
  expect((await pending2).kind).toBe("ok");
  coord.observed(1, success(2n));
  coord.selected();
  closeContext(context);
  expect(contextReport(context).violations).toEqual([]);
});
