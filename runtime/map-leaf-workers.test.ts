import { test, expect } from "bun:test";
import { createHash } from "node:crypto";
import { ownCallable, mapLeafWorker } from "./callable.ts";
import { success, caught, value, type Completion } from "./completion.ts";
import { array } from "./data.ts";
import { runAssertion } from "./assert/runner.ts";
import { withFixture } from "./assert/fixtures.ts";
import type { AssertionContext } from "./assert/context.ts";
import { runExplicitRoot, type OwnerContext } from "./owner-core.ts";
import * as arrays from "./collections/array.ts";
import { createMap, mapMethodWorker } from "./collections/map.ts";
import { catalogue } from "./catalogue.ts";
import { createDomainRuntime, domainFailureDiagnostics, type FailureShape } from "./domain.ts";
import {
  standardFailureDiagnostics,
  standardFailureOccurrenceID,
  type FailureOrigin,
} from "./failure.ts";

const origin: FailureOrigin = {
  source: "test:map-leaf-workers",
  start: 5,
  end: 11,
  invocation: ["app::count_one"],
};
const trace = { origin, site: "p::leaf#0" };

function leafDescriptor(
  companion: (map: unknown, key: unknown) => Completion<unknown>,
  keyKind = "str",
) {
  return { companion, keyKind, origin };
}

test("map leaves register closed two-input companions and forward map and key", () => {
  const seen: unknown[][] = [];
  const completion = success("leaf-ok");
  const action = ownCallable(
    "p::count#0",
    "app::count_one",
    [],
    async () => {
      throw new Error("slow adapter reached");
    },
    [],
    undefined,
    undefined,
    leafDescriptor((map, key) => {
      seen.push([map, key]);
      return completion;
    }),
  );
  const worker = mapLeafWorker(action);
  expect(worker).toBeDefined();
  expect(worker?.keyKind).toBe("str");
  expect(Object.isFrozen(worker)).toBe(true);
  const probe = { token: "map" };
  const out = worker?.run(probe, "k");
  expect(out).toBe(completion);
  expect(value(out!)).toBe("leaf-ok");
  expect(seen).toEqual([[probe, "k"]]);
});

test("map leaf runners return companion failures without assimilation", () => {
  const failed = caught(new Error("boom"), origin);
  const action = ownCallable(
    "p::count#1",
    "app::count_one",
    [],
    async () => {
      throw new Error("slow adapter reached");
    },
    [],
    undefined,
    undefined,
    leafDescriptor((map, key) => {
      void map;
      void key;
      return failed;
    }),
  );
  const worker = mapLeafWorker(action);
  expect(worker?.run({}, "k")).toBe(failed);
});

test("map leaf registration declines captures and resource captures", () => {
  const captured = ownCallable(
    "p::count#2",
    "app::count_one",
    ["stowaway"],
    async () => {
      throw new Error("slow adapter reached");
    },
    [],
    undefined,
    undefined,
    leafDescriptor((map, key) => success([map, key])),
  );
  expect(mapLeafWorker(captured)).toBeUndefined();
  const resourced = ownCallable(
    "p::count#3",
    "app::count_one",
    ["stowaway"],
    async () => {
      throw new Error("slow adapter reached");
    },
    [0],
    undefined,
    undefined,
    leafDescriptor((map, key) => success([map, key])),
  );
  expect(mapLeafWorker(resourced)).toBeUndefined();
});

test("map leaf registration rejects malformed descriptors", () => {
  const companion = (map: unknown, key: unknown) => success([map, key]);
  const cases: Array<[string, unknown]> = [
    ["extra key", { ...leafDescriptor(companion), positions: [] }],
    ["missing key", { companion, keyKind: "str" }],
    ["bad key kind", leafDescriptor(companion, "int[]")],
    ["numeric key kind", { companion, keyKind: 7, origin }],
    ["non-function companion", leafDescriptor("run" as never)],
    ["unary companion", leafDescriptor(((map: unknown) => success(map)) as never)],
    ["null descriptor", null],
    ["proxy descriptor", new Proxy(leafDescriptor(companion), {})],
    ["proxy origin", { companion, keyKind: "str", origin: new Proxy(origin, {}) }],
    [
      "proxy companion runs no trap",
      leafDescriptor(
        new Proxy(companion, {
          getOwnPropertyDescriptor() {
            throw new Error("companion trap reached");
          },
          get() {
            throw new Error("companion trap reached");
          },
        }),
      ),
    ],
  ];
  for (const [index, [, descriptor]] of cases.entries()) {
    expect(() =>
      ownCallable(
        `p::count#${100 + index}`,
        "app::count_one",
        [],
        async () => {
          throw new Error("slow adapter reached");
        },
        [],
        undefined,
        undefined,
        descriptor as never,
      ),
    ).toThrow("invalid map leaf descriptor");
  }
});

test("map leaf registration rejects accessor companions and lengths", () => {
  const accessor = {};
  Object.defineProperty(accessor, "companion", {
    enumerable: true,
    get() {
      throw new Error("descriptor getter reached");
    },
  });
  Object.defineProperty(accessor, "keyKind", { enumerable: true, value: "str" });
  Object.defineProperty(accessor, "origin", { enumerable: true, value: origin });
  expect(() =>
    ownCallable(
      "p::count#200",
      "app::count_one",
      [],
      async () => {
        throw new Error("slow adapter reached");
      },
      [],
      undefined,
      undefined,
      accessor as never,
    ),
  ).toThrow("invalid map leaf descriptor");
  const sneaky = (map: unknown, key: unknown) => success([map, key]);
  Object.defineProperty(sneaky, "length", {
    configurable: true,
    get() {
      throw new Error("length getter reached");
    },
  });
  expect(() =>
    ownCallable(
      "p::count#201",
      "app::count_one",
      [],
      async () => {
        throw new Error("slow adapter reached");
      },
      [],
      undefined,
      undefined,
      leafDescriptor(sneaky),
    ),
  ).toThrow("invalid map leaf descriptor");
});

test("map leaf queries decline foreign values and unregistered functions", () => {
  expect(mapLeafWorker(undefined)).toBeUndefined();
  expect(mapLeafWorker({})).toBeUndefined();
  async function plain() {
    return success(1);
  }
  expect(mapLeafWorker(plain)).toBeUndefined();
  const bare = ownCallable("p::bare#0", "app::bare", [], plain, []);
  expect(mapLeafWorker(bare)).toBeUndefined();
});

function countingLeaf(
  site: string,
  keyKind: string,
  companion: (acc: unknown, key: unknown) => Completion<unknown>,
) {
  let adapterCalls = 0;
  let leafCalls = 0;
  const action = ownCallable(
    site,
    "app::count_one",
    [],
    async () => {
      adapterCalls++;
      throw new Error("slow adapter reached");
    },
    [],
    undefined,
    undefined,
    leafDescriptor((acc, key) => {
      leafCalls++;
      return companion(acc, key);
    }, keyKind),
  );
  return { action, adapterCalls: () => adapterCalls, leafCalls: () => leafCalls };
}

test("map leaf folds reduce frozen key arrays without consulting the async adapter", async () => {
  const carriers: Completion<unknown>[] = [];
  const order: unknown[] = [];
  const leaf = countingLeaf("p::leaf#1", "str", (acc, key) => {
    order.push(key);
    const next = new Map(acc as Map<string, number>);
    next.set(key as string, (next.get(key as string) ?? 0) + 1);
    const carrier = success(next);
    carriers.push(carrier);
    return carrier;
  });
  const source = array(["a", "b", "a"]);
  const initial = new Map<string, number>();
  const result = await arrays.fold(source, initial, leaf.action, trace);
  expect(leaf.adapterCalls()).toBe(0);
  expect(leaf.leafCalls()).toBe(3);
  expect(order).toEqual(["a", "b", "a"]);
  expect(value(result)).toEqual(
    new Map([
      ["a", 2],
      ["b", 1],
    ]),
  );
  expect(carriers.includes(result as Completion<unknown>)).toBe(false);
  expect(initial.size).toBe(0);
  const maps = carriers.map((carrier) => value(carrier) as Map<string, number>);
  expect(new Set(maps).size).toBe(3);
  expect(maps.includes(initial)).toBe(false);
  expect(source).toEqual(["a", "b", "a"]);
});

test("map leaf folds return success(initial) on empty arrays without touching the leaf", async () => {
  const leaf = countingLeaf("p::leaf#2", "str", (acc) => success(acc));
  const initial = { token: "empty" };
  const result = await arrays.fold(array([]), initial, leaf.action, trace);
  expect(leaf.adapterCalls()).toBe(0);
  expect(leaf.leafCalls()).toBe(0);
  expect(value(result)).toBe(initial);
});

test("map leaf folds stop at the first failure and retain its carrier", async () => {
  const failed = caught(new Error("leaf blew up"), origin);
  const leaf = countingLeaf("p::leaf#3", "str", (acc, key) =>
    key === "b" ? (failed as Completion<unknown>) : success(String(acc) + String(key)),
  );
  const result = await arrays.fold(array(["a", "b", "c"]), "", leaf.action, trace);
  expect(leaf.adapterCalls()).toBe(0);
  expect(leaf.leafCalls()).toBe(2);
  expect(result).toBe(failed);
});

test("map leaf folds convert synchronous companion throws into retained failures", async () => {
  const leaf = countingLeaf("p::leaf#4", "str", (acc, key) => {
    if (key === "b") throw new Error("sync leaf throw");
    return success(String(acc) + String(key));
  });
  const result = await arrays.fold(array(["a", "b", "c"]), "", leaf.action, trace);
  expect(leaf.adapterCalls()).toBe(0);
  expect(leaf.leafCalls()).toBe(2);
  expect(result.kind).toBe("standard");
});

test("map leaf folds admit int and bool key kinds", async () => {
  const bools = countingLeaf("p::leaf#5", "bool", (acc, key) =>
    success(String(acc) + (key ? "T" : "F")),
  );
  const flipped = await arrays.fold(array([true, false, true]), "", bools.action, trace);
  expect(bools.adapterCalls()).toBe(0);
  expect(bools.leafCalls()).toBe(3);
  expect(value(flipped)).toBe("TFT");
  const ints = countingLeaf("p::leaf#6", "int", (acc, key) =>
    success((acc as bigint) + (key as bigint)),
  );
  const total = await arrays.fold(array([1n, 2n, 3n]), 0n, ints.action, trace);
  expect(ints.adapterCalls()).toBe(0);
  expect(ints.leafCalls()).toBe(3);
  expect(value(total)).toBe(6n);
});

test("hostile arrays keep leaf callables on the slow adapter", async () => {
  const cases: Array<[string, readonly unknown[], string]> = [
    ["mixed kinds", Object.freeze(["a", 7]), "a7"],
    ["boxed elements", Object.freeze(["a", new String("b")]), "ab"],
    ["unfrozen", ["a", "b"], "ab"],
    ["proxied", new Proxy(Object.freeze(["a", "b"]), {}), "ab"],
  ];
  for (const [index, [, source, expected]] of cases.entries()) {
    let adapterCalls = 0;
    const action = ownCallable(
      `p::leaf#${10 + index}`,
      "app::count_one",
      [],
      async (acc: string, item: unknown) => {
        adapterCalls++;
        return success(acc + String(item));
      },
      [],
      undefined,
      undefined,
      leafDescriptor((acc, key) => success(String(acc) + String(key))),
    );
    const result = await arrays.fold(source as readonly string[], "", action, trace);
    expect([adapterCalls, value(result)]).toEqual([
      (source as readonly unknown[]).length,
      expected,
    ]);
  }
});

test("defined assertion contexts keep leaf callables on the fixture-scheduled path", async () => {
  const events: string[] = [];
  const rows = ["a", "b"].map((key) => ({
    selector: "sample",
    owner: "p",
    arguments: async () => success([key]),
    expected: async () => {
      events.push(key);
      return success(key.toUpperCase());
    },
  }));
  let adapterCalls = 0;
  const action = ownCallable(
    "p::leaf#20",
    "app::count_one",
    [],
    async (_acc: string, item: string, callbackContext: AssertionContext | undefined) => {
      adapterCalls++;
      // Fixture rows resolve string payloads; the untyped fixture carrier
      // is asserted to the fold's accumulator type.
      return withFixture(
        callbackContext,
        "shared",
        rows,
        [item],
        async () => success("missing"),
        origin,
      ) as Promise<Completion<string>>;
    },
    [],
    undefined,
    undefined,
    leafDescriptor((acc, key) => success(String(acc) + String(key))),
  );
  const seen: unknown[] = [];
  const report = await runAssertion({
    root: { package: "p", declaration: "p::main", name: "sample" },
    expected: async () => success(undefined),
    actual: async (context) => {
      const out = await arrays.fold(array(["a", "b"]), "", action, { ...trace, context });
      seen.push(value(out));
      return success(undefined);
    },
  });
  expect(report.passed).toBe(true);
  expect(events).toEqual(["a", "b"]);
  expect(seen).toEqual(["B"]);
  expect(adapterCalls).toBe(2);
});

test("map leaf folds thread real sync map bodies and retain domain failures", async () => {
  const declarations = catalogue.errors.filter((e) =>
    ["collections::key_absent", "collections::key_exists"].includes(e.name),
  );
  const errors: FailureShape[] = declarations.map((e) => ({
    identity: createHash("sha256")
      .update("can-concrete-type-v1\0" + JSON.stringify(["error", e.identity]))
      .digest("hex"),
    kind: "error",
    declaration: e.identity,
    arguments: [],
    fields: [],
    leaves: [],
    inputs: [],
    errors: [],
  }));
  const domain = createDomainRuntime({
    declarations: declarations.map((e) => ({ identity: e.identity, name: e.name, parameters: 0 })),
    shapes: errors,
  });
  const maps = createMap<string, number>(
    domain,
    {
      map: "map-str-int",
      entry: "entry-str-int",
      absent: errors[0].identity,
      exists: errors[1].identity,
    },
    "str",
  );
  const getBody = mapMethodWorker(maps.get);
  const insertBody = mapMethodWorker(maps.insert);
  const replaceBody = mapMethodWorker(maps.replace);
  if (getBody === undefined || insertBody === undefined || replaceBody === undefined)
    throw new Error("sync map bodies unavailable");
  const empty = value(await maps.empty());
  const order: unknown[] = [];
  const leaf = countingLeaf("p::leaf#30", "str", (acc, key) => {
    order.push(key);
    const found = getBody(acc, key);
    if (found.kind !== "ok") return insertBody(acc, key, 1);
    return replaceBody(acc, key, (value(found) as number) + 1);
  });
  const result = await arrays.fold(array(["a", "b", "a"]), empty, leaf.action, trace);
  expect(leaf.adapterCalls()).toBe(0);
  expect(leaf.leafCalls()).toBe(3);
  expect(order).toEqual(["a", "b", "a"]);
  const entries = value(await maps.entries(value(result)));
  expect(entries.map((entry) => [entry.key, entry.value])).toEqual([
    ["a", 2],
    ["b", 1],
  ]);
  expect(value(await maps.entries(empty))).toEqual([]);
  let domainCarrier: Completion<unknown> | undefined;
  const missing = countingLeaf("p::leaf#31", "str", (acc, key) => {
    const carrier = getBody(acc, key);
    domainCarrier = carrier;
    return carrier;
  });
  const failed = await arrays.fold(array(["zz"]), empty, missing.action, trace);
  expect(missing.adapterCalls()).toBe(0);
  expect(missing.leafCalls()).toBe(1);
  expect(failed.kind).toBe("domain");
  if (domainCarrier === undefined) throw new Error("leaf never ran");
  expect(Object.is(failed, domainCarrier)).toBe(true);
  if (failed.kind !== "domain") throw new Error("expected domain");
  expect(domainFailureDiagnostics(failed.value).declaration.name).toBe("collections::key_absent");
});

test("map leaf folds keep standard first boundaries and fresh occurrences", async () => {
  const leafOrigin: FailureOrigin = {
    source: "test:leaf-site",
    start: 1,
    end: 2,
    invocation: ["app::count_one"],
  };
  const leaf = countingLeaf("p::leaf#32", "str", () =>
    caught(new Error("site failure"), leafOrigin),
  );
  const first = await arrays.fold(array(["a"]), "", leaf.action, trace);
  const second = await arrays.fold(array(["a"]), "", leaf.action, trace);
  expect(leaf.adapterCalls()).toBe(0);
  expect(leaf.leafCalls()).toBe(2);
  expect(first.kind).toBe("standard");
  expect(second.kind).toBe("standard");
  if (first.kind !== "standard" || second.kind !== "standard") throw new Error("expected standard");
  expect(first).not.toBe(second);
  expect(standardFailureDiagnostics(first.value).origin).toEqual(leafOrigin);
  expect(standardFailureOccurrenceID(first.value)).not.toBe(
    standardFailureOccurrenceID(second.value),
  );
});

test("explicit owners keep leaf callables on the slow path with correct root settling", async () => {
  const seen: unknown[] = [];
  let owned: OwnerContext | undefined;
  let adapterCalls = 0;
  const action = ownCallable(
    "p::leaf#21",
    "app::count_one",
    [],
    async (acc: string, item: string, ctx: OwnerContext) => {
      adapterCalls++;
      seen.push(ctx);
      return success(acc + item);
    },
    [],
    undefined,
    undefined,
    leafDescriptor((acc, key) => success(String(acc) + String(key))),
  );
  const root = await runExplicitRoot(async (owner) => {
    owned = owner;
    return arrays.fold(array(["a", "b"]), "", action, { origin, site: "p::array#owner", owner });
  });
  expect(root.cleanupFailed).toBe(false);
  expect(root.completion.kind).toBe("ok");
  if (root.completion.kind === "ok") expect(value(root.completion)).toBe("ab");
  expect(owned).toBeDefined();
  expect(seen).toEqual([owned, owned]);
  expect(adapterCalls).toBe(2);
});
