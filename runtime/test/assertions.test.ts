import { test, expect } from "bun:test";
import { runAssertion, assertionEqual } from "../assert/runner.ts";
import { assertionContext, contextReport, denyLiveBoundary, closeContext } from "../assert/context.ts";
import { withFixture } from "../assert/fixtures.ts";
import { success } from "../completion.ts";
import { array, record } from "../data.ts";
import { ownBytes } from "../bytes.ts";

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
