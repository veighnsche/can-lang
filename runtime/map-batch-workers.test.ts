import { test, expect } from "bun:test";
import { createHash } from "node:crypto";
import { ownCallable, mapBatchWorker } from "./callable.ts";
import { success, value, type Completion } from "./completion.ts";
import { array, opaqueContents } from "./data.ts";
import { runAssertion } from "./assert/runner.ts";
import { withFixture } from "./assert/fixtures.ts";
import type { AssertionContext } from "./assert/context.ts";
import { runExplicitRoot, type OwnerContext } from "./owner-core.ts";
import * as arrays from "./collections/array.ts";
import { createMap, mapBatchCapability } from "./collections/map.ts";
import { catalogue } from "./catalogue.ts";
import { createDomainRuntime, type FailureShape } from "./domain.ts";
import {
  standardFailureDiagnostics,
  standardFailureOccurrenceID,
  isStandardFailure,
  type FailureOrigin,
} from "./failure.ts";

const origin: FailureOrigin = {
  source: "test:map-batch-workers",
  start: 317,
  end: 347,
  invocation: ["app::count_one"],
};
const trace = { origin, site: "p::batch#0" };

const batchDeclarations = catalogue.errors.filter((e) =>
  ["collections::key_absent", "collections::key_exists"].includes(e.name),
);
const batchShapes: FailureShape[] = batchDeclarations.map((e) => ({
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
const shapeByName = new Map(
  batchDeclarations.map((entry, index) => [entry.name, batchShapes[index]]),
);
const batchDomain = createDomainRuntime({
  declarations: batchDeclarations.map((e) => ({
    identity: e.identity,
    name: e.name,
    parameters: 0,
  })),
  shapes: batchShapes,
});
function batchIdentities(keyKind: string) {
  return {
    map: `map-batch-${keyKind}`,
    entry: `entry-batch-${keyKind}`,
    absent: shapeByName.get("collections::key_absent")?.identity ?? "",
    exists: shapeByName.get("collections::key_exists")?.identity ?? "",
  };
}
function strMaps() {
  return createMap<string, bigint>(batchDomain, batchIdentities("str"), "str");
}
function intMaps() {
  return createMap<bigint, bigint>(batchDomain, batchIdentities("int"), "int");
}
function boolMaps() {
  return createMap<boolean, bigint>(batchDomain, batchIdentities("bool"), "bool");
}

function batchDescriptor(
  absent: (key: never) => bigint,
  present: (key: never, previous: bigint) => bigint,
  factory: (...args: never[]) => unknown,
  keyKind = "str",
) {
  return { absent, present, keyKind, origin, factory };
}

function countingBatch(
  site: string,
  keyKind: string,
  factory: (...args: never[]) => unknown,
  absent: (key: unknown) => bigint,
  present: (key: unknown, previous: bigint) => bigint,
  leaf?: (acc: unknown, key: unknown) => Completion<unknown>,
) {
  let adapterCalls = 0;
  let absentCalls = 0;
  let presentCalls = 0;
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
    leaf === undefined
      ? undefined
      : {
          companion: (acc: unknown, key: unknown) => {
            leafCalls++;
            return leaf(acc, key);
          },
          keyKind,
          origin,
        },
    {
      absent: (key: never) => {
        absentCalls++;
        return absent(key);
      },
      present: (key: never, previous: bigint) => {
        presentCalls++;
        return present(key, previous);
      },
      keyKind,
      origin,
      factory,
    },
  );
  return {
    action,
    adapterCalls: () => adapterCalls,
    absentCalls: () => absentCalls,
    presentCalls: () => presentCalls,
    leafCalls: () => leafCalls,
  };
}

test("map batches register closed transitions and forward key and previous", () => {
  const maps = strMaps();
  const seen: unknown[][] = [];
  const action = ownCallable(
    "p::batch#1",
    "app::count_one",
    [],
    async () => {
      throw new Error("slow adapter reached");
    },
    [],
    undefined,
    undefined,
    undefined,
    batchDescriptor(
      (key) => {
        seen.push(["absent", key]);
        return 1n;
      },
      (key, previous) => {
        seen.push(["present", key, previous]);
        return previous + 1n;
      },
      maps.get,
    ),
  );
  const worker = mapBatchWorker(action);
  expect(worker).toBeDefined();
  expect(worker?.keyKind).toBe("str");
  expect(Object.isFrozen(worker)).toBe(true);
  expect(worker?.origin).toBe(origin);
  expect(worker?.batch).toBe(mapBatchCapability(maps.get));
  expect(worker?.batch).toBe(mapBatchCapability(maps.insert));
  expect(worker?.batch).toBe(mapBatchCapability(maps.replace));
  expect(worker?.absent("k")).toBe(1n);
  expect(worker?.present("k", 41n)).toBe(42n);
  expect(seen).toEqual([
    ["absent", "k"],
    ["present", "k", 41n],
  ]);
});

test("map batch transitions enforce bigint results without reading payloads", () => {
  const maps = strMaps();
  let traps = 0;
  const hostile = {};
  // oxlint-disable no-thenable -- A hostile then member proves bigint enforcement never reads the payload.
  Object.defineProperty(hostile, "then", {
    enumerable: true,
    get() {
      traps++;
      return undefined;
    },
  });
  // oxlint-enable no-thenable
  const action = ownCallable(
    "p::batch#2",
    "app::count_one",
    [],
    async () => {
      throw new Error("slow adapter reached");
    },
    [],
    undefined,
    undefined,
    undefined,
    batchDescriptor(
      (_key: never) => 7 as unknown as bigint,
      (_key: never, _previous: bigint) => hostile as unknown as bigint,
      maps.get,
    ),
  );
  const worker = mapBatchWorker(action);
  if (worker === undefined) throw new Error("batch worker missing");
  let absentThrown: unknown;
  try {
    worker.absent("k");
  } catch (thrown) {
    absentThrown = thrown;
  }
  expect(isStandardFailure(absentThrown)).toBe(true);
  if (!isStandardFailure(absentThrown)) throw new Error("expected standard failure");
  expect(standardFailureDiagnostics(absentThrown).origin).toEqual(origin);
  let presentThrown: unknown;
  try {
    worker.present("k", 1n);
  } catch (thrown) {
    presentThrown = thrown;
  }
  expect(isStandardFailure(presentThrown)).toBe(true);
  expect(traps).toBe(0);
});

test("map batch registration declines captures and resource captures", () => {
  const maps = strMaps();
  const captured = ownCallable(
    "p::batch#3",
    "app::count_one",
    ["stowaway"],
    async () => {
      throw new Error("slow adapter reached");
    },
    [],
    undefined,
    undefined,
    undefined,
    batchDescriptor(
      (_key: never) => 1n,
      (_key, previous) => previous + 1n,
      maps.get,
    ),
  );
  expect(mapBatchWorker(captured)).toBeUndefined();
  const resourced = ownCallable(
    "p::batch#4",
    "app::count_one",
    ["stowaway"],
    async () => {
      throw new Error("slow adapter reached");
    },
    [0],
    undefined,
    undefined,
    undefined,
    batchDescriptor(
      (_key: never) => 1n,
      (_key, previous) => previous + 1n,
      maps.get,
    ),
  );
  expect(mapBatchWorker(resourced)).toBeUndefined();
});

test("map batch registration declines unknown factories without throwing", () => {
  const maps = strMaps();
  const foreign = [maps.remove, maps.empty, maps.entries, async () => success(0)];
  for (const [index, factory] of foreign.entries()) {
    const action = ownCallable(
      `p::batch#${10 + index}`,
      "app::count_one",
      [],
      async () => {
        throw new Error("slow adapter reached");
      },
      [],
      undefined,
      undefined,
      undefined,
      batchDescriptor(
        (_key: never) => 1n,
        (_key, previous) => previous + 1n,
        factory as never,
      ),
    );
    expect(mapBatchWorker(action)).toBeUndefined();
  }
  const proxied = ownCallable(
    "p::batch#14",
    "app::count_one",
    [],
    async () => {
      throw new Error("slow adapter reached");
    },
    [],
    undefined,
    undefined,
    undefined,
    batchDescriptor(
      (_key: never) => 1n,
      (_key, previous) => previous + 1n,
      new Proxy(maps.get, {}) as never,
    ),
  );
  expect(mapBatchWorker(proxied)).toBeUndefined();
  expect(mapBatchCapability(undefined)).toBeUndefined();
  expect(mapBatchCapability({})).toBeUndefined();
});

test("map batch registration rejects malformed descriptors", () => {
  const maps = strMaps();
  const absent = (_key: never) => 1n;
  const present = (_key: never, previous: bigint) => previous + 1n;
  const valid = batchDescriptor(absent, present, maps.get);
  const cases: Array<[string, unknown]> = [
    ["extra key", { ...valid, positions: [] }],
    ["missing key", { absent, present, keyKind: "str", origin }],
    ["missing factory", { absent, present, keyKind: "str", factory: maps.get }],
    ["bad key kind", batchDescriptor(absent, present, maps.get, "int[]")],
    ["numeric key kind", { ...valid, keyKind: 7 }],
    ["non-function absent", batchDescriptor("run" as never, present, maps.get)],
    ["non-function present", batchDescriptor(absent, "run" as never, maps.get)],
    ["nullary absent", batchDescriptor((() => 1n) as never, present, maps.get)],
    ["binary absent", batchDescriptor(((_a: never, _b: never) => 1n) as never, present, maps.get)],
    ["unary present", batchDescriptor(absent, ((_a: never) => 1n) as never, maps.get)],
    ["non-function factory", batchDescriptor(absent, present, "get" as never)],
    ["null descriptor", null],
    ["proxy descriptor", new Proxy(valid, {})],
    ["proxy origin", { ...valid, origin: new Proxy(origin, {}) }],
    [
      "proxy absent runs no trap",
      batchDescriptor(
        new Proxy(absent, {
          getOwnPropertyDescriptor() {
            throw new Error("absent trap reached");
          },
          get() {
            throw new Error("absent trap reached");
          },
        }),
        present,
        maps.get,
      ),
    ],
  ];
  for (const [index, [, descriptor]] of cases.entries()) {
    expect(() =>
      ownCallable(
        `p::batch#${100 + index}`,
        "app::count_one",
        [],
        async () => {
          throw new Error("slow adapter reached");
        },
        [],
        undefined,
        undefined,
        undefined,
        descriptor as never,
      ),
    ).toThrow("invalid map batch descriptor");
  }
});

test("map batch registration rejects accessor companions and lengths", () => {
  const maps = strMaps();
  const accessor: Record<string, unknown> = {};
  Object.defineProperty(accessor, "absent", {
    enumerable: true,
    get() {
      throw new Error("descriptor getter reached");
    },
  });
  Object.defineProperty(accessor, "present", {
    enumerable: true,
    value: (_key: never, previous: bigint) => previous + 1n,
  });
  Object.defineProperty(accessor, "keyKind", { enumerable: true, value: "str" });
  Object.defineProperty(accessor, "origin", { enumerable: true, value: origin });
  Object.defineProperty(accessor, "factory", { enumerable: true, value: maps.get });
  expect(() =>
    ownCallable(
      "p::batch#200",
      "app::count_one",
      [],
      async () => {
        throw new Error("slow adapter reached");
      },
      [],
      undefined,
      undefined,
      undefined,
      accessor as never,
    ),
  ).toThrow("invalid map batch descriptor");
  const sneaky = (_key: never, previous: bigint) => previous + 1n;
  Object.defineProperty(sneaky, "length", {
    configurable: true,
    get() {
      throw new Error("length getter reached");
    },
  });
  expect(() =>
    ownCallable(
      "p::batch#201",
      "app::count_one",
      [],
      async () => {
        throw new Error("slow adapter reached");
      },
      [],
      undefined,
      undefined,
      undefined,
      batchDescriptor((_key: never) => 1n, sneaky, maps.get),
    ),
  ).toThrow("invalid map batch descriptor");
});

test("map batch queries decline foreign values and unregistered functions", () => {
  expect(mapBatchWorker(undefined)).toBeUndefined();
  expect(mapBatchWorker({})).toBeUndefined();
  async function plain() {
    return success(1);
  }
  expect(mapBatchWorker(plain)).toBeUndefined();
  const bare = ownCallable("p::batch#900", "app::bare", [], plain, []);
  expect(mapBatchWorker(bare)).toBeUndefined();
  async function leafPlain() {
    return success(1);
  }
  const leafOnly = ownCallable(
    "p::batch#901",
    "app::bare",
    [],
    leafPlain,
    [],
    undefined,
    undefined,
    {
      companion: ((_acc: unknown, _key: unknown) => success(1)) as never,
      keyKind: "str",
      origin,
    },
  );
  expect(mapBatchWorker(leafOnly)).toBeUndefined();
});

test("map batch folds reduce str keys with one publication and intact seed", async () => {
  const maps = strMaps();
  const empty = value(await maps.empty());
  const seed = value(await maps.insert(empty, "z", 9n));
  const order: unknown[] = [];
  const batch = countingBatch(
    "p::batch#30",
    "str",
    maps.get,
    (key) => {
      order.push(key);
      return 1n;
    },
    (key, previous) => {
      order.push(key);
      return previous + 1n;
    },
  );
  const result = await arrays.fold(array(["a", "b", "a", "z"]), seed, batch.action, trace);
  expect(batch.adapterCalls()).toBe(0);
  expect(batch.leafCalls()).toBe(0);
  expect(batch.absentCalls()).toBe(2);
  expect(batch.presentCalls()).toBe(2);
  expect(order).toEqual(["a", "b", "a", "z"]);
  const published = value(result);
  expect(published).not.toBe(seed);
  const entries = value(await maps.entries(published));
  expect(entries.map((entry) => [entry.key, entry.value])).toEqual([
    ["z", 10n],
    ["a", 2n],
    ["b", 1n],
  ]);
  expect(value(await maps.entries(seed)).map((entry) => [entry.key, entry.value])).toEqual([
    ["z", 9n],
  ]);
  expect(value(await maps.entries(empty))).toEqual([]);
  const snapshot = opaqueContents(published);
  expect(snapshot).toEqual([10n, 2n, 1n]);
  expect(snapshot).not.toBe(opaqueContents(seed));
  expect(opaqueContents(seed)).toEqual([9n]);
});

test("map batch folds admit int keys with key-dependent values and negatives", async () => {
  const maps = intMaps();
  const empty = value(await maps.empty());
  const seed = value(await maps.insert(empty, -3n, -30n));
  const batch = countingBatch(
    "p::batch#31",
    "int",
    maps.get,
    (key) => (key as bigint) * 2n,
    (_key, previous) => previous - 1n,
  );
  const result = await arrays.fold(array([4n, -3n, 4n]), seed, batch.action, trace);
  expect(batch.adapterCalls()).toBe(0);
  expect(batch.absentCalls()).toBe(1);
  expect(batch.presentCalls()).toBe(2);
  const entries = value(await maps.entries(value(result)));
  expect(entries.map((entry) => [entry.key, entry.value])).toEqual([
    [-3n, -31n],
    [4n, 7n],
  ]);
  expect(value(await maps.entries(seed)).map((row) => [row.key, row.value])).toEqual([[-3n, -30n]]);
});

test("map batch folds admit bool keys and large bigint arithmetic", async () => {
  const maps = boolMaps();
  const big = 2n ** 62n;
  const empty = value(await maps.empty());
  const seed = value(await maps.insert(empty, false, big));
  const batch = countingBatch(
    "p::batch#32",
    "bool",
    maps.get,
    () => 1n,
    (_key, previous) => previous + big,
  );
  const result = await arrays.fold(array([true, false, true]), seed, batch.action, trace);
  expect(batch.adapterCalls()).toBe(0);
  expect(batch.absentCalls()).toBe(1);
  expect(batch.presentCalls()).toBe(2);
  const entries = value(await maps.entries(value(result)));
  expect(entries.map((entry) => [entry.key, entry.value])).toEqual([
    [false, big * 2n],
    [true, 1n + big],
  ]);
});

test("map batch folds return the identical initial token on empty input", async () => {
  const maps = strMaps();
  const seed = value(await maps.insert(value(await maps.empty()), "z", 9n));
  const batch = countingBatch(
    "p::batch#33",
    "str",
    maps.get,
    () => 1n,
    (_key, previous) => previous + 1n,
  );
  const result = await arrays.fold(array([]), seed, batch.action, trace);
  expect(batch.adapterCalls()).toBe(0);
  expect(batch.absentCalls()).toBe(0);
  expect(batch.presentCalls()).toBe(0);
  expect(value(result)).toBe(seed);
});

test("foreign initial maps decline the batch runner to the leaf path", async () => {
  const maps = strMaps();
  const foreign = createMap<string, bigint>(batchDomain, batchIdentities("str-foreign"), "str");
  const foreignSeed = value(await foreign.insert(value(await foreign.empty()), "z", 9n));
  const batch = countingBatch(
    "p::batch#40",
    "str",
    maps.get,
    () => 1n,
    (_key, previous) => previous + 1n,
    (_acc, key) => success(key),
  );
  const result = await arrays.fold(array(["a", "b"]), foreignSeed, batch.action, trace);
  expect(batch.adapterCalls()).toBe(0);
  expect(batch.absentCalls()).toBe(0);
  expect(batch.presentCalls()).toBe(0);
  expect(batch.leafCalls()).toBe(2);
  expect(value(result) as unknown as string).toBe("b");
});

test("proxied initial maps decline the batch runner to the leaf path", async () => {
  const maps = strMaps();
  const seed = value(await maps.insert(value(await maps.empty()), "z", 9n));
  const batch = countingBatch(
    "p::batch#41",
    "str",
    maps.get,
    () => 1n,
    (_key, previous) => previous + 1n,
    (_acc, key) => success(key),
  );
  const result = await arrays.fold(array(["a"]), new Proxy(seed, {}), batch.action, trace);
  expect(batch.absentCalls()).toBe(0);
  expect(batch.presentCalls()).toBe(0);
  expect(batch.leafCalls()).toBe(1);
  expect(batch.adapterCalls()).toBe(0);
  expect(result.kind).toBe("ok");
});

test("non-bigint hosted values decline before traversal without reading payloads", async () => {
  const maps = strMaps();
  let traps = 0;
  const hostile = {};
  // oxlint-disable no-thenable -- A hostile then member proves pre-traversal refusal never reads the payload.
  Object.defineProperty(hostile, "then", {
    enumerable: true,
    get() {
      traps++;
      return undefined;
    },
  });
  // oxlint-enable no-thenable
  const empty = value(await maps.empty());
  // HOST INJECTION (outside valid Can inputs): a host-built map holding a
  // non-bigint thenable where Can guarantees bigint values.
  const poisoned = value(await maps.insert(empty, "evil", hostile as unknown as bigint));
  const batch = countingBatch(
    "p::batch#42",
    "str",
    maps.get,
    () => 1n,
    (_key, previous) => previous + 1n,
    (_acc, key) => success(key),
  );
  const result = await arrays.fold(array(["a"]), poisoned, batch.action, trace);
  expect(batch.absentCalls()).toBe(0);
  expect(batch.presentCalls()).toBe(0);
  expect(batch.leafCalls()).toBe(1);
  expect(batch.adapterCalls()).toBe(0);
  expect(result.kind).toBe("ok");
  expect(traps).toBe(0);
  expect(value(await maps.entries(poisoned)).map((row) => row.key)).toEqual(["evil"]);
});

test("hostile arrays keep batch callables on the slow adapter", async () => {
  const maps = strMaps();
  const seed = value(await maps.insert(value(await maps.empty()), "z", 9n));
  let accessorReads = 0;
  const accessor = ["a", "b"];
  Object.defineProperty(accessor, "1", {
    enumerable: true,
    configurable: true,
    get() {
      accessorReads++;
      return "b";
    },
  });
  const frozenAccessor = Object.freeze(accessor);
  const holed: string[] = ["a"];
  holed[2] = "b";
  const sparse = Object.freeze(holed);
  const symbolled = Object.freeze(Object.assign(["a", "b"], { [Symbol("extra")]: 1 }));
  const shadowed = Object.freeze(Object.assign(["a", "b"], { map: 1 }));
  const cases: Array<[string, readonly unknown[], number, string]> = [
    ["mixed kinds", Object.freeze(["a", 7]), 2, "a7"],
    ["boxed elements", Object.freeze(["a", new String("b")]), 2, "ab"],
    ["unfrozen", ["a", "b"], 2, "ab"],
    ["proxied", new Proxy(Object.freeze(["a", "b"]), {}), 2, "ab"],
    ["accessor element", frozenAccessor, 2, "ab"],
    ["sparse hole", sparse, 2, "ab"],
    ["symbol key", symbolled, 2, "ab"],
    ["own method shadow", shadowed, 2, "ab"],
  ];
  for (const [index, [, source, calls, expected]] of cases.entries()) {
    let adapterCalls = 0;
    let transitions = 0;
    const action = ownCallable(
      `p::batch#${50 + index}`,
      "app::count_one",
      [],
      async (acc: string, item: unknown) => {
        adapterCalls++;
        return success(acc + String(item));
      },
      [],
      undefined,
      undefined,
      {
        companion: ((acc: unknown, key: unknown) => {
          transitions++;
          return success(String(acc) + String(key));
        }) as never,
        keyKind: "str",
        origin,
      },
      batchDescriptor(
        ((_key: never) => {
          transitions++;
          return 1n;
        }) as (key: never) => bigint,
        ((_key: never, _previous: bigint) => {
          transitions++;
          return 1n;
        }) as (key: never, previous: bigint) => bigint,
        maps.get,
      ),
    );
    const result = await arrays.fold(source as readonly string[], `s${index}`, action, trace);
    expect([adapterCalls, value(result), transitions]).toEqual([calls, `s${index}${expected}`, 0]);
  }
  // Exactly the adapter's own element read; both guards declined through
  // descriptor checks without touching the getter.
  expect(accessorReads).toBe(1);
  expect(value(await maps.entries(seed)).map((row) => [row.key, row.value])).toEqual([["z", 9n]]);
});

test("defined assertion contexts keep batch callables on the fixture-scheduled path", async () => {
  const maps = strMaps();
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
  let transitions = 0;
  const action = ownCallable(
    "p::batch#60",
    "app::count_one",
    [],
    async (_acc: string, item: string, callbackContext: AssertionContext | undefined) => {
      adapterCalls++;
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
    {
      companion: ((acc: unknown, key: unknown) => {
        transitions++;
        return success(String(acc) + String(key));
      }) as never,
      keyKind: "str",
      origin,
    },
    batchDescriptor(
      ((_key: never) => {
        transitions++;
        return 1n;
      }) as (key: never) => bigint,
      ((_key: never, _previous: bigint) => {
        transitions++;
        return 1n;
      }) as (key: never, previous: bigint) => bigint,
      maps.get,
    ),
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
  expect(transitions).toBe(0);
});

test("explicit owners keep batch callables on the slow path with correct root settling", async () => {
  const maps = strMaps();
  const seen: unknown[] = [];
  let owned: OwnerContext | undefined;
  let adapterCalls = 0;
  let transitions = 0;
  const action = ownCallable(
    "p::batch#61",
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
    {
      companion: ((acc: unknown, key: unknown) => {
        transitions++;
        return success(String(acc) + String(key));
      }) as never,
      keyKind: "str",
      origin,
    },
    batchDescriptor(
      ((_key: never) => {
        transitions++;
        return 1n;
      }) as (key: never) => bigint,
      ((_key: never, _previous: bigint) => {
        transitions++;
        return 1n;
      }) as (key: never, previous: bigint) => bigint,
      maps.get,
    ),
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
  expect(transitions).toBe(0);
});

test("captured batch callables run on the slow adapter", async () => {
  const maps = strMaps();
  let adapterCalls = 0;
  const action = ownCallable(
    "p::batch#62",
    "app::count_one",
    ["stowaway"],
    async (acc: string, item: string) => {
      adapterCalls++;
      return success(acc + item);
    },
    [],
    undefined,
    undefined,
    {
      companion: ((_acc: unknown, _key: unknown) => success("")) as never,
      keyKind: "str",
      origin,
    },
    batchDescriptor(
      (_key: never) => 1n,
      (_key, previous) => previous + 1n,
      maps.get,
    ),
  );
  expect(mapBatchWorker(action)).toBeUndefined();
  const result = await arrays.fold(array(["a", "b"]), "", action, trace);
  expect(adapterCalls).toBe(2);
  expect(value(result)).toBe("ab");
});

test("map batch folds stop at the first injected fault with the site origin", async () => {
  const maps = strMaps();
  const seed = value(await maps.insert(value(await maps.empty()), "z", 9n));
  // HOST INJECTION (outside valid Can inputs): a throwing transition where
  // emitted pure bigint code cannot fail.
  const batch = countingBatch(
    "p::batch#70",
    "str",
    maps.get,
    (key) => {
      if (key === "b") throw new Error("injected batch fault");
      return 1n;
    },
    (_key, previous) => previous + 1n,
  );
  const result = await arrays.fold(array(["a", "b", "c"]), seed, batch.action, trace);
  expect(batch.adapterCalls()).toBe(0);
  expect(batch.absentCalls()).toBe(2);
  expect(batch.presentCalls()).toBe(0);
  expect(result.kind).toBe("standard");
  if (result.kind !== "standard") throw new Error("expected standard");
  expect(standardFailureDiagnostics(result.value).origin).toEqual(origin);
  expect(value(await maps.entries(seed)).map((row) => [row.key, row.value])).toEqual([["z", 9n]]);
});

test("map batch faults carry fresh occurrences and never replay partial work", async () => {
  const maps = strMaps();
  const seed = value(await maps.insert(value(await maps.empty()), "z", 9n));
  let armed = true;
  const batch = countingBatch(
    "p::batch#71",
    "str",
    maps.get,
    () => 1n,
    (_key, previous) => {
      // HOST INJECTION (outside valid Can inputs): one transient fault.
      if (armed) throw new Error("transient batch fault");
      return previous + 1n;
    },
  );
  const first = await arrays.fold(array(["a", "a"]), seed, batch.action, trace);
  const second = await arrays.fold(array(["a", "a"]), seed, batch.action, trace);
  expect(batch.adapterCalls()).toBe(0);
  expect(first.kind).toBe("standard");
  expect(second.kind).toBe("standard");
  if (first.kind !== "standard" || second.kind !== "standard") throw new Error("expected standard");
  expect(first).not.toBe(second);
  expect(standardFailureDiagnostics(first.value).origin).toEqual(origin);
  expect(standardFailureDiagnostics(second.value).origin).toEqual(origin);
  expect(standardFailureOccurrenceID(first.value)).not.toBe(
    standardFailureOccurrenceID(second.value),
  );
  // Each attempt visits absent then the faulted present exactly once:
  // no replay of the faulted visit and no partial map escapes.
  expect(batch.absentCalls()).toBe(2);
  expect(batch.presentCalls()).toBe(2);
  expect(value(await maps.entries(seed)).map((row) => [row.key, row.value])).toEqual([["z", 9n]]);
  armed = false;
  const retry = await arrays.fold(array(["a", "a"]), seed, batch.action, trace);
  expect(value(await maps.entries(value(retry))).map((row) => [row.key, row.value])).toEqual([
    ["z", 9n],
    ["a", 2n],
  ]);
});

test("map batch folds surface hostile non-bigint results without assimilation", async () => {
  const maps = strMaps();
  const seed = value(await maps.insert(value(await maps.empty()), "z", 9n));
  let traps = 0;
  const hostile = {};
  // oxlint-disable no-thenable -- A hostile then member proves failure surfacing never assimilates the payload.
  Object.defineProperty(hostile, "then", {
    enumerable: true,
    get() {
      traps++;
      return undefined;
    },
  });
  // oxlint-enable no-thenable
  // HOST INJECTION (outside valid Can inputs): a transition returning a
  // thenable where emitted code returns bigint.
  const batch = countingBatch(
    "p::batch#72",
    "str",
    maps.get,
    () => hostile as unknown as bigint,
    (_key, previous) => previous + 1n,
  );
  const result = await arrays.fold(array(["a", "b"]), seed, batch.action, trace);
  expect(batch.absentCalls()).toBe(1);
  expect(batch.presentCalls()).toBe(0);
  expect(result.kind).toBe("standard");
  if (result.kind !== "standard") throw new Error("expected standard");
  expect(standardFailureDiagnostics(result.value).origin).toEqual(origin);
  expect(traps).toBe(0);
});

test("wrong-host initials without a leaf run the full generic fallback", async () => {
  const maps = strMaps();
  const foreign = createMap<string, bigint>(batchDomain, batchIdentities("str-foreign"), "str");
  // HOST INJECTION (outside valid Can inputs): folding a foreign-factory
  // initial under this factory's batch descriptor.
  const foreignSeed = value(await foreign.insert(value(await foreign.empty()), "z", 9n));
  let adapterCalls = 0;
  const action = ownCallable(
    "p::batch#73",
    "app::count_one",
    [],
    async (acc: unknown, item: unknown) => {
      adapterCalls++;
      return success([acc, item]);
    },
    [],
    undefined,
    undefined,
    undefined,
    batchDescriptor(
      (_key: never) => 1n,
      (_key, previous) => previous + 1n,
      maps.get,
    ),
  );
  const result = await arrays.fold(array(["a", "b"]), foreignSeed, action, trace);
  expect(adapterCalls).toBe(2);
  expect(value(result)).toEqual([[foreignSeed, "a"], "b"]);
});

test("leaf-only callables keep the leaf route when no batch registers", async () => {
  const maps = strMaps();
  const seed = value(await maps.insert(value(await maps.empty()), "z", 9n));
  let adapterCalls = 0;
  let leafCalls = 0;
  const action = ownCallable(
    "p::batch#74",
    "app::count_one",
    [],
    async () => {
      adapterCalls++;
      throw new Error("slow adapter reached");
    },
    [],
    undefined,
    undefined,
    {
      companion: ((acc: unknown, key: unknown) => {
        leafCalls++;
        return success([acc, key]);
      }) as never,
      keyKind: "str",
      origin,
    },
  );
  expect(mapBatchWorker(action)).toBeUndefined();
  const result = await arrays.fold(array(["a", "b"]), seed, action, trace);
  expect(adapterCalls).toBe(0);
  expect(leafCalls).toBe(2);
  expect(value(result) as unknown).toEqual([[seed, "a"], "b"]);
});
