import { test, expect } from "bun:test";
import { createHash } from "node:crypto";
import * as arrays from "../collections/array.ts";
import { success, failure, value, type Completion } from "../completion.ts";
import { array, record, recordIdentity } from "../data.ts";
import { captureStandard, standardFailureKind } from "../failure.ts";
import { createDomainRuntime } from "../domain.ts";
import { runAssertion } from "../assert/runner.ts";
import { withFixture } from "../assert/fixtures.ts";
import { settle } from "../coordination.ts";
const origin = { source: "test:array", start: 0, end: 0, invocation: [] };
const trace = { origin, site: "p::array#0" };
function deferred() {
  let release!: () => void;
  const promise = new Promise<void>((resolve) => {
    release = resolve;
  });
  return { promise, release };
}
const ticks = async () => {
  for (let i = 0; i < 25; i++) await Promise.resolve();
};

test("map awaits each native index observation without assimilating input or output data", async () => {
  let assimilations = 0,
    active = 0,
    peak = 0;
  const started: number[] = [];
  const items = array(
    [0, 1, 2].map((index) =>
      record("item", [
        ["index", index],
        [
          "then",
          () => {
            assimilations++;
          },
        ],
      ]),
    ),
  );
  const gates = items.map(() => deferred());
  const pending = arrays.map(
    items,
    async (item) => {
      const index = item.index as number;
      started.push(index);
      active++;
      peak = Math.max(peak, active);
      await gates[index].promise;
      active--;
      return success(item);
    },
    trace,
  );
  await ticks();
  expect(started).toEqual([0]);
  gates[0].release();
  await ticks();
  expect(started).toEqual([0, 1]);
  gates[1].release();
  await ticks();
  expect(started).toEqual([0, 1, 2]);
  gates[2].release();
  const result = value(await pending);
  expect(result).toEqual(items);
  expect(result[0]).toBe(items[0]);
  expect(result).not.toBe(items);
  expect(Object.isFrozen(result)).toBe(true);
  expect(peak).toBe(1);
  expect(assimilations).toBe(0);
});

test("filter, search and empty cases use awaited boolean decisions and preserve aliases", async () => {
  const source = array([1, 2, 3]),
    seen: number[] = [];
  const predicate = async (item: number) => {
    seen.push(item);
    await Promise.resolve();
    return success(item === 2);
  };
  expect(value(await arrays.filter(source, predicate, trace))).toEqual([2]);
  expect(seen).toEqual([1, 2, 3]);
  seen.length = 0;
  const found = value(
    await arrays.find(source, predicate, { none: "none", some: "some-int" }, trace),
  );
  expect(recordIdentity(found)).toBe("some-int");
  expect((found as { value: number }).value).toBe(2);
  expect(seen).toEqual([1, 2]);
  seen.length = 0;
  expect(value(await arrays.some(source, predicate, trace))).toBe(true);
  expect(seen).toEqual([1, 2]);
  seen.length = 0;
  expect(value(await arrays.every(source, predicate, trace))).toBe(false);
  expect(seen).toEqual([1]);
  const never = async () => {
    throw Error("empty callback reached");
  };
  expect(value(await arrays.map([], never, trace))).toEqual([]);
  expect(value(await arrays.filter([], never, trace))).toEqual([]);
  expect(value(await arrays.some([], never, trace))).toBe(false);
  expect(value(await arrays.every([], never, trace))).toBe(true);
  expect(
    recordIdentity(value(await arrays.find([], never, { none: "none", some: "some-int" }, trace))),
  ).toBe("none");
  expect(value(await arrays.sortBy([], never, trace))).toEqual([]);
  expect(value(await arrays.forEach([], never, trace))).toBeUndefined();
  expect(source).toEqual([1, 2, 3]);
});

test("native reduce chains serialize for_each and protect fold accumulators", async () => {
  const gate = deferred(),
    seen: number[] = [],
    source = array([1, 2, 3]);
  const pending = arrays.forEach(
    source,
    async (item) => {
      seen.push(item);
      if (item === 1) await gate.promise;
      return success(undefined);
    },
    trace,
  );
  await ticks();
  expect(seen).toEqual([1]);
  gate.release();
  expect(value(await pending)).toBeUndefined();
  expect(seen).toEqual([1, 2, 3]);
  let assimilations = 0;
  const initial = record("acc", [
    ["sum", 0],
    [
      "then",
      () => {
        assimilations++;
      },
    ],
  ]);
  const result = value(
    await arrays.fold(
      source,
      initial,
      async (acc, item) =>
        success(
          record("acc", [
            ["sum", (acc.sum as number) + item],
            [
              "then",
              () => {
                assimilations++;
              },
            ],
          ]),
        ),
      trace,
    ),
  );
  expect(result.sum).toBe(6);
  expect(assimilations).toBe(0);
  expect(
    value(
      await arrays.fold(
        [],
        initial,
        async () => {
          throw Error("empty callback reached");
        },
        trace,
      ),
    ),
  ).toBe(initial);
});

function domainFailure(): Completion<never> {
  const declaration = "can.project.root/array::failed";
  const identity = createHash("sha256")
    .update("can-concrete-type-v1\0" + JSON.stringify(["error", declaration]))
    .digest("hex");
  const domain = createDomainRuntime({
    declarations: [{ identity: declaration, name: "array::failed", parameters: 0 }],
    shapes: [
      {
        identity,
        kind: "error",
        declaration,
        arguments: [],
        fields: [],
        leaves: [],
        inputs: [],
        errors: [],
      },
    ],
  });
  return failure(domain.create(identity, record(identity, []), origin));
}
for (const failed of [failure(captureStandard(new Error("stop"), origin)), domainFailure()])
  test(`${failed.kind} callback failure prevents later visits in every adapter`, async () => {
    const source = array([1, 2, 3]);
    for (const operation of [
      "map",
      "filter",
      "for_each",
      "fold",
      "find",
      "some",
      "every",
      "sort_by",
    ]) {
      const seen: number[] = [];
      const run = async (item: number) => {
        seen.push(item);
        await Promise.resolve();
        return item === 2
          ? failed
          : success(
              operation === "every"
                ? true
                : operation === "filter" || operation === "find" || operation === "some"
                  ? false
                  : item,
            );
      };
      let result: Completion;
      switch (operation) {
        case "map":
          result = await arrays.map(source, run, trace);
          break;
        case "filter":
          result = await arrays.filter(source, run as arrays.Callback<[number], boolean>, trace);
          break;
        case "find":
          result = await arrays.find(
            source,
            run as arrays.Callback<[number], boolean>,
            { none: "none", some: "some-int" },
            trace,
          );
          break;
        case "some":
          result = await arrays.some(source, run as arrays.Callback<[number], boolean>, trace);
          break;
        case "every":
          result = await arrays.every(source, run as arrays.Callback<[number], boolean>, trace);
          break;
        case "sort_by":
          result = await arrays.sortBy(source, run, trace);
          break;
        case "fold":
          result = await arrays.fold(
            source,
            0,
            (_, item) => run(item) as Promise<Completion<number>>,
            trace,
          );
          break;
        default:
          result = await arrays.forEach(
            source,
            run as unknown as arrays.Callback<[number], void>,
            trace,
          );
      }
      expect(result).toBe(failed);
      expect(seen).toEqual([1, 2]);
    }
  });

test("sort keys are computed once in order before native stable sorting", async () => {
  for (const keys of [
    [2n, 1n, 1n],
    [2, 1, 1],
    ["b", "a", "a"],
    [true, false, false],
    [-0, +0, -0],
  ]) {
    const source = array(keys.map((key, index) => Object.freeze({ key, index }))),
      seen: number[] = [];
    const result = value(
      await arrays.sortBy(
        source,
        async (item) => {
          seen.push(item.index);
          await Promise.resolve();
          return success(item.key);
        },
        trace,
      ),
    );
    expect(seen).toEqual([0, 1, 2]);
    expect(result.map((item) => item.index)).toEqual(
      typeof keys[0] === "number" && Object.is(keys[0], -0) ? [0, 1, 2] : [1, 2, 0],
    );
    expect(source.map((item) => item.index)).toEqual([0, 1, 2]);
    expect(Object.isFrozen(result)).toBe(true);
  }
  for (const key of [NaN, Infinity, -Infinity]) {
    const seen: number[] = [];
    const result = await arrays.sortBy(
      [0, 1, 2],
      async (item) => {
        seen.push(item);
        return success(item === 1 ? key : 0);
      },
      trace,
    );
    expect(result.kind).toBe("standard");
    if (result.kind === "standard") expect(standardFailureKind(result.value)).toBe("arithmetic");
    expect(seen).toEqual([0, 1]);
  }
});

test("native mapping gaps keep later participant fixtures behind the current traversal", async () => {
  const events: number[] = [];
  const rows = [0, 1, 2].map((index) => ({
    selector: "sample",
    arguments: async () => success([index]),
    expected: async () => {
      events.push(index);
      return success(index);
    },
  }));
  const report = await runAssertion({
    root: { package: "p", declaration: "p::main", name: "sample" },
    expected: async () => success(undefined),
    actual: async (context) => {
      await settle(
        "all",
        [
          {
            captures: [],
            run: (child) =>
              arrays.map(
                [0, 1],
                async (item, callbackContext) =>
                  withFixture(
                    callbackContext,
                    "shared",
                    rows,
                    [item],
                    async () => success(-1),
                    origin,
                  ),
                { ...trace, context: child },
              ),
          },
          {
            captures: [],
            run: (child) =>
              withFixture(child, "shared", rows, [2], async () => success(-1), origin),
          },
        ],
        context,
        "p::main#0",
        [[0], [1]],
      );
      return success(undefined);
    },
  });
  expect(report.passed).toBe(true);
  expect(events).toEqual([0, 1, 2]);
});

test("native copying operations preserve order, element aliases and immutable sources", () => {
  const first = record("item", [["index", 1]]),
    second = record("item", [["index", 2]]);
  const source = array([first]),
    right = array([second]);
  const joined = arrays.concat(source, right),
    appended = arrays.append(source, second),
    reversed = arrays.toReversed(joined);
  expect(joined).toEqual([first, second]);
  expect(appended).toEqual(joined);
  expect(reversed).toEqual([second, first]);
  expect(reversed[1]).toBe(first);
  expect(source).toEqual([first]);
  expect(right).toEqual([second]);
  for (const result of [joined, appended, reversed]) {
    expect(Object.isFrozen(result)).toBe(true);
    expect(result).not.toBe(source);
  }
});

test("collection adapters delegate mapping, filtering, reduction and sorting to native APIs", async () => {
  const originals = {
    fromAsync: Array.fromAsync,
    filter: Array.prototype.filter,
    reduce: Array.prototype.reduce,
    toSorted: Array.prototype.toSorted,
  };
  const calls = { mapping: 0, filtering: 0, reducing: 0, sorting: 0 };
  Array.fromAsync = (async (...args: Parameters<typeof Array.fromAsync>) => {
    calls.mapping++;
    return originals.fromAsync.apply(Array, args);
  }) as typeof Array.fromAsync;
  Array.prototype.filter = function (...args: Parameters<typeof Array.prototype.filter>) {
    calls.filtering++;
    return originals.filter.apply(this, args);
  };
  Array.prototype.reduce = function (
    this: unknown[],
    ...args: Parameters<typeof Array.prototype.reduce>
  ) {
    calls.reducing++;
    return originals.reduce.apply(this, args);
  } as typeof Array.prototype.reduce;
  Array.prototype.toSorted = function (...args: Parameters<typeof Array.prototype.toSorted>) {
    calls.sorting++;
    return originals.toSorted.apply(this, args);
  };
  try {
    const source = array([2, 1]);
    await arrays.map(source, async (item) => success(item), trace);
    await arrays.filter(source, async () => success(true), trace);
    await arrays.fold(source, 0, async (acc, item) => success(acc + item), trace);
    await arrays.forEach(source, async () => success(undefined), trace);
    await arrays.sortBy(source, async (item) => success(item), trace);
  } finally {
    Array.fromAsync = originals.fromAsync;
    Array.prototype.filter = originals.filter;
    Array.prototype.reduce = originals.reduce;
    Array.prototype.toSorted = originals.toSorted;
  }
  expect(calls).toEqual({ mapping: 3, filtering: 1, reducing: 2, sorting: 1 });
});
