import { test, expect } from "bun:test";
import { runAssertion } from "../assert/runner.ts";
import { callContext, type AssertionContext } from "../assert/context.ts";
import { withFixture } from "../assert/fixtures.ts";
import { settle, type Mode } from "../coordination.ts";
import { success, type Completion } from "../completion.ts";
const root = { package: "p", declaration: "p::main", name: "sample" };
const origin = { source: "can:test", start: 0, end: 0, invocation: [] };
function deferred() {
  let release!: () => void;
  const promise = new Promise<void>((resolve) => {
    release = resolve;
  });
  return { promise, release };
}
const ticks = async () => {
  for (let i = 0; i < 40; i++) await Promise.resolve();
};

test("generated-style calls share lexical FIFO across repeated and transitive invocations", async () => {
  const consumed: number[] = [];
  const rows = [0, 1, 2].map((index) => ({
    selector: "sample",
    owner: "p",
    arguments: async () => success([index]),
    expected: async () => {
      consumed.push(index);
      return success(index);
    },
  }));
  const leaf = (context: AssertionContext | undefined, index: number) =>
    callContext(context, "p::helper#0", (child) =>
      withFixture(
        child,
        "p::helper#0/when",
        rows,
        [index],
        async () => {
          throw new Error("real target executed");
        },
        origin,
      ),
    );
  const report = await runAssertion({
    root,
    expected: async () => success(2),
    actual: (context) =>
      callContext(context, "p::main#0", async (child) => {
        await leaf(child, 0);
        await leaf(child, 1);
        return callContext(child, "p::helper#1", (nested) => leaf(nested, 2));
      }),
  });
  expect(report.passed).toBe(true);
  expect(consumed).toEqual([0, 1, 2]);
  expect(report.fixturePaths).toEqual([]);
});

for (const mode of ["all", "settled", "any", "race"] as Mode[])
  test(`${mode}: reversed real arrival keeps canonical fixture allocation and drains losers`, async () => {
    const delay = deferred(),
      started = deferred(),
      events: string[] = [];
    const rows = [0, 1].map((index) => ({
      selector: "sample",
      owner: "p",
      arguments: async () => success([index]),
      expected: async () => {
        events.push(`row${index}`);
        return success(index);
      },
    }));
    const pending = runAssertion({
      root,
      expected: async () => success("done"),
      actual: async (context) => {
        const selected = await settle(
          mode,
          [0, 1].map((index) => ({
            captures: [],
            run: async (child?: AssertionContext): Promise<Completion> => {
              if (index === 0)
                await callContext(child, "p::pure#0", async () => {
                  started.release();
                  await delay.promise;
                  return success(undefined);
                });
              return callContext(child, "p::helper#0", (nested) =>
                withFixture(
                  nested,
                  "p::helper#0/when",
                  rows,
                  [index],
                  async () => success(-1),
                  origin,
                ),
              );
            },
          })),
          context,
          "p::main#0",
          [[0], [1]],
        );
        events.push("selected");
        if (mode === "any" || mode === "race") {
          expect(selected.kind).toBe("one");
          if (selected.kind === "one") expect(selected.index).toBe(0);
        }
        return success("done");
      },
    });
    await started.promise;
    await ticks();
    expect(events).toEqual([]);
    delay.release();
    const report = await pending;
    expect(report.passed).toBe(true);
    expect(report.violations).toEqual([]);
    expect(events).toEqual(
      mode === "any" || mode === "race"
        ? ["row0", "selected", "row1"]
        : ["row0", "row1", "selected"],
    );
  });

test("first-row mismatch consumes that row and reports full paths without searching", async () => {
  const report = await runAssertion({
    root,
    expected: async () => success(0),
    actual: (context) =>
      callContext(context, "p::main#7", (child) =>
        withFixture(
          child,
          "table",
          [0, 1].map((value) => ({
            selector: "sample",
            owner: "p",
            arguments: async () => success([value]),
            expected: async () => success(value),
          })),
          [1],
          async () => success(0),
          origin,
        ),
      ),
  });
  expect(report.passed).toBe(false);
  expect(report.violations).toEqual(["argument mismatch", "unused fixture"]);
  expect(report.fixturePaths[0]).toMatchObject({
    reason: "argument mismatch",
    expected: {
      table: "table",
      row: 0,
      path: { root, segments: [{ site: "p::main#7", occurrence: 0, participant: [] }] },
    },
    actual: { root, segments: [{ site: "p::main#7", occurrence: 0, participant: [] }] },
  });
});

test("nested coordination and spread positions retain one shared root queue", async () => {
  const consumed: number[] = [];
  const rows = [0, 1, 2].map((index) => ({
    selector: "sample",
    owner: "p",
    arguments: async () => success([index]),
    expected: async () => {
      consumed.push(index);
      return success(index);
    },
  }));
  const participant = (index: number) => ({
    captures: [],
    run: (context?: AssertionContext) =>
      callContext(context, "p::helper#0", (child) =>
        withFixture(child, "table", rows, [index], async () => success(-1), origin),
      ),
  });
  const report = await runAssertion({
    root,
    expected: async () => success(3),
    actual: async (context) => {
      await settle(
        "all",
        [
          {
            captures: [],
            run: async (child?: AssertionContext) => {
              await settle("all", [participant(0), participant(1)], child, "p::nested#0", [
                [0, 0],
                [0, 1],
              ]);
              return success(undefined);
            },
          },
          participant(2),
        ],
        context,
        "p::main#0",
        [[0], [1, 0]],
      );
      return success(3);
    },
  });
  expect(report.passed).toBe(true);
  expect(consumed).toEqual([0, 1, 2]);
});
