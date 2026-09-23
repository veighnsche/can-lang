import { test, expect } from "bun:test";
import { runAssertion } from "../assert/runner.ts";
import { callContext } from "../assert/context.ts";
import { success } from "../completion.ts";
const root = { package: "p", declaration: "p::main", name: "sample" };

test("progress sink observes running barrier snapshots", async () => {
  const records: Record<string, unknown>[] = [];
  const report = await runAssertion(
    {
      root,
      expected: async () => success(1),
      actual: async (context) => {
        await callContext(context, "p::main#0", async () => success(undefined));
        await Bun.sleep(600);
        return success(1);
      },
    },
    (record) => {
      records.push(record);
    },
  );
  expect(report.passed).toBe(true);
  expect(records.length).toBeGreaterThan(0);
  for (const record of records) {
    expect(record.phase).toBe("running");
    const progress = record.progress as { pending: number; frames: unknown[] };
    expect(typeof progress.pending).toBe("number");
    expect(Array.isArray(progress.frames)).toBe(true);
  }
});

test("throwing progress sink never fails the root", async () => {
  const report = await runAssertion(
    {
      root,
      expected: async () => success(1),
      actual: async () => {
        await Bun.sleep(300);
        return success(1);
      },
    },
    () => {
      throw new Error("sink exploded");
    },
  );
  expect(report.passed).toBe(true);
});

test("omitted sink keeps legacy report shape", async () => {
  const report = await runAssertion({
    root,
    expected: async () => success(1),
    actual: async () => success(1),
  });
  expect(report.passed).toBe(true);
  expect(report.root).toEqual(root);
});
