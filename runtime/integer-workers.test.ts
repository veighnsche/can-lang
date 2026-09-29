import { test, expect } from "bun:test";
import * as arrays from "./collections/array.ts";
import { ownCallable, integerWorker, callableReceipt } from "./callable.ts";
import { success, value } from "./completion.ts";
import { array } from "./data.ts";
import { standardFailureDiagnostics, standardFailureOccurrenceID } from "./failure.ts";
import { runAssertion } from "./assert/runner.ts";
import { withFixture } from "./assert/fixtures.ts";
import type { AssertionContext } from "./assert/context.ts";
import { runExplicitRoot, type OwnerContext } from "./owner-core.ts";
import type { FailureOrigin } from "./failure.ts";

const origin: FailureOrigin = {
  source: "test:integer-workers",
  start: 3,
  end: 9,
  invocation: ["app::run"],
};
const trace = { origin, site: "p::workers#0" };
const callbackOrigin: FailureOrigin = {
  source: "test:integer-worker-callback",
  start: 11,
  end: 17,
  invocation: ["app::double"],
};

function doubleDescriptor(companion: (...operands: bigint[]) => bigint) {
  return { companion, positions: [] as number[], arity: 1, origin: callbackOrigin };
}

test("integer workers map frozen bigint arrays without consulting the async adapter", async () => {
  const source = array([-5n, 0n, 7n, 2n ** 100n]);
  let adapterCalls = 0;
  const action = ownCallable(
    "p::double#0",
    "app::double",
    [],
    async () => {
      adapterCalls++;
      throw new Error("slow adapter reached");
    },
    [],
    undefined,
    doubleDescriptor((item) => item * 2n),
  );
  const result = value(await arrays.map(source, action, trace));
  expect(result).toEqual([-10n, 0n, 14n, 2n ** 101n]);
  for (const item of result) expect(typeof item).toBe("bigint");
  expect(Object.isFrozen(result)).toBe(true);
  expect(result).not.toBe(source);
  expect(source).toEqual([-5n, 0n, 7n, 2n ** 100n]);
  expect(adapterCalls).toBe(0);
  expect(callableReceipt(action)?.target).toBe("app::double");
});

test("integer workers fold with bigint initials and skip empty traversals", async () => {
  const source = array([1n, -2n, 3n]);
  let adapterCalls = 0;
  let companionCalls = 0;
  const action = ownCallable(
    "p::add#0",
    "app::add",
    [],
    async () => {
      adapterCalls++;
      throw new Error("slow adapter reached");
    },
    [],
    undefined,
    {
      companion: (total: bigint, item: bigint) => {
        companionCalls++;
        return total + item;
      },
      positions: [],
      arity: 2,
      origin: callbackOrigin,
    },
  );
  expect(value(await arrays.fold(source, 10n, action, trace))).toBe(12n);
  expect(adapterCalls).toBe(0);
  expect(companionCalls).toBe(3);
  companionCalls = 0;
  expect(value(await arrays.fold(array([]), 10n, action, trace))).toBe(10n);
  expect(value(await arrays.map(array([]), action as never, trace))).toEqual([]);
  expect(companionCalls).toBe(0);
  expect(adapterCalls).toBe(0);
});

test("integer workers bind saved captures once and ignore later capture mutation", async () => {
  const captures: unknown[] = [3n];
  let adapterCalls = 0;
  const action = ownCallable(
    "p::add_offset#0",
    "app::add_offset",
    captures,
    async () => {
      adapterCalls++;
      throw new Error("slow adapter reached");
    },
    [],
    undefined,
    {
      companion: (offset: bigint, item: bigint) => offset + item,
      positions: [0],
      arity: 1,
      origin: callbackOrigin,
    },
  );
  captures[0] = 500n;
  const result = value(await arrays.map(array([1n, 2n]), action, trace));
  expect(result).toEqual([4n, 5n]);
  expect(adapterCalls).toBe(0);
  expect(callableReceipt(action)?.captures).toEqual([3n]);
});

test("malformed worker descriptors throw while unusable captures decline registration", async () => {
  const valid = {
    companion: (item: bigint) => item * 2n,
    positions: [] as number[],
    arity: 1,
    origin: callbackOrigin,
  };
  const holedPositions: unknown[] = [];
  holedPositions.length = 1;
  const malformed: Array<[string, unknown]> = [
    ["companion not a function", { ...valid, companion: 7 }],
    ["positions not an array", { ...valid, positions: "0" }],
    ["positions with foreign prototype", { ...valid, positions: Object.setPrototypeOf([0], null) }],
    ["arity not an integer", { ...valid, arity: 1.5 }],
    ["arity not a number", { ...valid, arity: "1" }],
    ["null origin", { ...valid, origin: null }],
    ["proxied origin", { ...valid, origin: new Proxy(callbackOrigin, {}) }],
    ["extra key", { ...valid, extra: true }],
    ["missing key", { companion: valid.companion, positions: [], arity: 1 }],
    ["companion arity mismatch", { ...valid, arity: 2 }],
    [
      "negative slot",
      { ...valid, companion: (a: bigint, b: bigint) => a + b, positions: [-1], arity: 1 },
    ],
    [
      "fractional slot",
      { ...valid, companion: (a: bigint, b: bigint) => a + b, positions: [0.5], arity: 1 },
    ],
    [
      "unsorted slots",
      {
        ...valid,
        companion: (a: bigint, b: bigint, c: bigint) => a + b + c,
        positions: [1, 0],
        arity: 1,
      },
    ],
    [
      "duplicate slots",
      {
        ...valid,
        companion: (a: bigint, b: bigint, c: bigint) => a + b + c,
        positions: [0, 0],
        arity: 1,
      },
    ],
    [
      "holed slots",
      { ...valid, companion: (a: bigint, b: bigint) => a + b, positions: holedPositions, arity: 1 },
    ],
    ["proxied descriptor", new Proxy(valid, {})],
  ];
  const accessed: string[] = [];
  const getterDescriptor = {
    get companion() {
      accessed.push("companion");
      return valid.companion;
    },
    positions: [],
    arity: 1,
    origin: callbackOrigin,
  };
  malformed.push(["accessor companion", getterDescriptor]);
  for (const [name, descriptor] of malformed) {
    let message = "";
    try {
      ownCallable(
        "p::worker#0",
        "app::worker",
        [],
        async () => success(0n),
        [],
        undefined,
        descriptor as never,
      );
    } catch (error) {
      message = (error as Error).message;
    }
    expect([name, message]).toEqual([name, "invalid integer worker descriptor"]);
  }
  expect(accessed).toEqual([]);
  // Structurally sound descriptors with non-bigint, out-of-range or
  // resource captures decline fast registration and keep slow behavior.
  const declines: Array<[string, unknown[], number[], { positions: number[]; arity: number }]> = [
    ["non-bigint capture", ["offset"], [], { positions: [0], arity: 1 }],
    ["missing capture slot", [], [], { positions: [0], arity: 1 }],
    ["resource capture", [3n], [0], { positions: [0], arity: 1 }],
  ];
  for (const [name, captures, resources, shape] of declines) {
    let adapterCalls = 0;
    const action = ownCallable(
      "p::worker#0",
      "app::worker",
      captures,
      async (item: bigint) => {
        adapterCalls++;
        return success(item * 2n);
      },
      resources,
      undefined,
      {
        companion: (first: bigint, second: bigint) => first + second,
        positions: shape.positions,
        arity: shape.arity,
        origin: callbackOrigin,
      },
    );
    expect(integerWorker(action)).toBeUndefined();
    const out = value(await arrays.map(array([1n, 2n]), action, trace));
    expect([name, out]).toEqual([name, [2n, 4n]]);
    expect(adapterCalls).toBe(2);
  }
});

test("integer worker queries expose only exact registered metadata", async () => {
  const action = ownCallable(
    "p::double#0",
    "app::double",
    [],
    async (item: bigint) => success(item),
    [],
    undefined,
    doubleDescriptor((item) => item * 2n),
  );
  const worker = integerWorker(action);
  expect(worker?.arity).toBe(1);
  expect(typeof worker?.run).toBe("function");
  expect(worker?.run(21n)).toBe(42n);
  expect(Object.isFrozen(worker)).toBe(true);
  expect(integerWorker(async () => success(0n))).toBeUndefined();
  expect(integerWorker({})).toBeUndefined();
  expect(integerWorker(undefined)).toBeUndefined();
  const plain = ownCallable("p::worker#0", "app::worker", [], async () => success(0n));
  expect(integerWorker(plain)).toBeUndefined();
});

test("worker traversal rejects foreign and overridden arrays on the slow path", async () => {
  const failing = (name: string) => async () => {
    throw new Error(name + " adapter reached");
  };
  const worker = (adapter: () => Promise<never>) =>
    ownCallable(
      "p::worker#0",
      "app::worker",
      [],
      adapter,
      [],
      undefined,
      doubleDescriptor((item) => item * 2n),
    );
  // Proxy, prototype, frozen-state, symbol and override rejections never
  // consult the worker.
  const frozen = Object.freeze([1n, 2n]);
  const foreign: Array<[string, readonly unknown[]]> = [
    ["proxy", new Proxy(frozen, {})],
    [
      "foreign prototype",
      Object.freeze(Object.setPrototypeOf([1n, 2n], Object.create(Array.prototype))),
    ],
    ["mutable", [1n, 2n]],
    ["symbol key", Object.freeze(Object.assign([1n, 2n], { [Symbol("tag")]: 1 }))],
    ["own map", Object.freeze(Object.assign([1n, 2n], { map: 1 }))],
    ["own reduce", Object.freeze(Object.assign([1n, 2n], { reduce: 1 }))],
    ["own constructor", Object.freeze(Object.assign([1n, 2n], { constructor: 1 }))],
  ];
  for (const [name, source] of foreign) {
    const action = worker(failing(name));
    const result = await arrays.map(source as readonly bigint[], action, trace);
    // Only the throwing adapter can fail here; a silent worker would succeed.
    expect([name, result.kind]).toEqual([name, "standard"]);
    if (result.kind === "standard") {
      expect(standardFailureDiagnostics(result.value).origin).toEqual(trace.origin);
    }
  }
  // An own keys member additionally shadows the slow path's own traversal,
  // so the guard rejection surfaces as an honest standard failure.
  const shadowed = Object.freeze(Object.assign([1n, 2n], { keys: 1 }));
  const shadowedResult = await arrays.map(
    shadowed as readonly bigint[],
    worker(failing("keys")),
    trace,
  );
  expect(shadowedResult.kind).toBe("standard");
  // Sparse, non-bigint and accessor slots keep the slow adapter as well.
  const holed: unknown[] = [1n];
  holed.length = 2;
  const sparse = Object.freeze(holed);
  const mixed = Object.freeze([1n, "2"] as unknown as readonly bigint[]);
  let getterCalls = 0;
  const accessor = Object.freeze(
    Object.defineProperties([1n, 2n], {
      "0": {
        get: () => {
          getterCalls++;
          return 1n;
        },
        enumerable: true,
        configurable: false,
      },
    }),
  );
  for (const [name, source] of [
    ["sparse", sparse],
    ["mixed", mixed],
    ["accessor", accessor],
  ] as const) {
    const seen: unknown[] = [];
    const action = ownCallable(
      "p::worker#0",
      "app::worker",
      [],
      async (item: unknown) => {
        seen.push(item);
        return success(0n);
      },
      [],
      undefined,
      doubleDescriptor((item) => item * 2n),
    );
    const out = value(await arrays.map(source as readonly bigint[], action, trace));
    expect([name, out]).toEqual([name, [0n, 0n]]);
    expect(seen.length).toBe(2);
  }
  // The guard reads descriptors without invoking the getter; only the slow
  // path's single read of element zero observes it.
  expect(getterCalls).toBe(1);
});

test("worker output guards refuse hostile results without assimilation", async () => {
  // oxlint-disable no-thenable -- Hostile then members prove refusal happens before assimilation.
  let assimilations = 0;
  const hostile = {
    then() {
      assimilations++;
    },
  };
  // oxlint-enable no-thenable
  for (const [name, raw] of [
    ["number", 5],
    ["string", "5"],
    ["thenable", hostile],
    ["null", null],
  ] as const) {
    const action = ownCallable(
      "p::worker#0",
      "app::worker",
      [],
      async () => {
        throw new Error("slow adapter reached");
      },
      [],
      undefined,
      doubleDescriptor((item: bigint) => {
        void item;
        return raw as unknown as bigint;
      }),
    );
    const result = await arrays.map(array([1n]), action, trace);
    expect([name, result.kind]).toEqual([name, "standard"]);
    if (result.kind === "standard") {
      expect(standardFailureDiagnostics(result.value).origin).toEqual(callbackOrigin);
    }
  }
  expect(assimilations).toBe(0);
});

test("registered worker faults stop native traversal with fresh occurrences", async () => {
  const occurrences: bigint[] = [];
  for (const endpoint of ["map", "fold"] as const) {
    let calls = 0;
    const failSecond = () => {
      calls++;
      if (calls === 2) throw new Error(endpoint + " boom");
    };
    const companion =
      endpoint === "map"
        ? (item: bigint) => {
            failSecond();
            return item * 2n;
          }
        : (total: bigint, item: bigint) => {
            failSecond();
            return total + item;
          };
    const action = ownCallable(
      "p::worker#0",
      "app::worker",
      [],
      async () => {
        throw new Error("slow adapter reached");
      },
      [],
      undefined,
      {
        companion,
        positions: [],
        arity: endpoint === "map" ? 1 : 2,
        origin: callbackOrigin,
      },
    );
    const result =
      endpoint === "map"
        ? await arrays.map(array([1n, 2n, 3n]), action, trace)
        : await arrays.fold(array([1n, 2n, 3n]), 0n, action, trace);
    expect(calls).toBe(2);
    expect(result.kind).toBe("standard");
    if (result.kind === "standard") {
      const diagnostics = standardFailureDiagnostics(result.value);
      expect(diagnostics.origin).toEqual(trace.origin);
      occurrences.push(diagnostics.occurrenceID);
    }
  }
  expect(occurrences.length).toBe(2);
  expect(occurrences[0]).not.toBe(occurrences[1]);
  // A second identical fault mints another fresh occurrence.
  const again = ownCallable(
    "p::worker#0",
    "app::worker",
    [],
    async () => {
      throw new Error("slow adapter reached");
    },
    [],
    undefined,
    doubleDescriptor((item: bigint) => {
      void item;
      throw new Error("repeat boom");
    }),
  );
  const first = await arrays.map(array([1n]), again, trace);
  const second = await arrays.map(array([1n]), again, trace);
  expect(first.kind).toBe("standard");
  expect(second.kind).toBe("standard");
  if (first.kind === "standard" && second.kind === "standard") {
    expect(standardFailureOccurrenceID(first.value)).not.toBe(
      standardFailureOccurrenceID(second.value),
    );
    expect(standardFailureDiagnostics(first.value).origin).toEqual(trace.origin);
  }
});

test("defined assertion contexts keep worker callables on the fixture-scheduled path", async () => {
  const events: bigint[] = [];
  const rows = [1n, 2n].map((index) => ({
    selector: "sample",
    owner: "p",
    arguments: async () => success([index]),
    expected: async () => {
      events.push(index);
      return success(index * 2n);
    },
  }));
  let adapterCalls = 0;
  const action = ownCallable(
    "p::double#0",
    "app::double",
    [],
    async (item: bigint, callbackContext: AssertionContext | undefined) => {
      adapterCalls++;
      return withFixture(callbackContext, "shared", rows, [item], async () => success(-1n), origin);
    },
    [],
    undefined,
    doubleDescriptor((item) => item * 2n),
  );
  const seen: unknown[] = [];
  const report = await runAssertion({
    root: { package: "p", declaration: "p::main", name: "sample" },
    expected: async () => success(undefined),
    actual: async (context) => {
      const out = await arrays.map(array([1n, 2n]), action, { ...trace, context });
      seen.push(value(out));
      return success(undefined);
    },
  });
  expect(report.passed).toBe(true);
  expect(events).toEqual([1n, 2n]);
  expect(seen).toEqual([[2n, 4n]]);
  expect(adapterCalls).toBe(2);
});

test("explicit owners keep worker callables on the slow path with correct root settling", async () => {
  const seen: unknown[] = [];
  let owned: OwnerContext | undefined;
  let adapterCalls = 0;
  const action = ownCallable(
    "p::double#0",
    "app::double",
    [],
    async (item: bigint, ctx: OwnerContext) => {
      adapterCalls++;
      seen.push(ctx);
      return success(item * 2n);
    },
    [],
    undefined,
    doubleDescriptor((item) => item * 2n),
  );
  const root = await runExplicitRoot(async (owner) => {
    owned = owner;
    return arrays.map(array([1n, 2n]), action, { origin, site: "p::array#owner", owner });
  });
  expect(root.cleanupFailed).toBe(false);
  expect(root.completion.kind).toBe("ok");
  if (root.completion.kind === "ok") expect(value(root.completion)).toEqual(array([2n, 4n]));
  expect(owned).toBeDefined();
  expect(seen).toEqual([owned, owned]);
  expect(adapterCalls).toBe(2);
});

test("arity mismatches and foreign callables keep the slow adapter", async () => {
  const source = array([1n, 2n]);
  const unary = ownCallable(
    "p::worker#0",
    "app::worker",
    [],
    async (total: bigint, item: bigint) => success(total + item),
    [],
    undefined,
    doubleDescriptor((item) => item * 2n),
  );
  let unaryCalls = 0;
  const unaryFold = ownCallable(
    "p::worker#0",
    "app::worker",
    [],
    async (total: bigint, item: bigint) => {
      unaryCalls++;
      return success(total + item);
    },
    [],
    undefined,
    doubleDescriptor((item) => item * 2n),
  );
  expect(value(await arrays.fold(source, 0n, unaryFold, trace))).toBe(3n);
  expect(unaryCalls).toBe(2);
  let binaryCalls = 0;
  const binaryMap = ownCallable(
    "p::worker#0",
    "app::worker",
    [],
    async (item: bigint) => {
      binaryCalls++;
      return success(item * 2n);
    },
    [],
    undefined,
    {
      companion: (total: bigint, item: bigint) => total + item,
      positions: [],
      arity: 2,
      origin: callbackOrigin,
    },
  );
  expect(value(await arrays.map(source, binaryMap, trace))).toEqual([2n, 4n]);
  expect(binaryCalls).toBe(2);
  let foreignCalls = 0;
  const foreign = async (item: bigint) => {
    foreignCalls++;
    return success(item * 2n);
  };
  expect(value(await arrays.map(source, foreign, trace))).toEqual([2n, 4n]);
  expect(foreignCalls).toBe(2);
  // A non-bigint fold initial keeps the slow path even with a valid worker.
  let initialCalls = 0;
  const folded = ownCallable(
    "p::worker#0",
    "app::worker",
    [],
    async (total: unknown, item: bigint) => {
      initialCalls++;
      return success((typeof total === "bigint" ? total : 0n) + item);
    },
    [],
    undefined,
    {
      companion: (total: bigint, item: bigint) => total + item,
      positions: [],
      arity: 2,
      origin: callbackOrigin,
    },
  );
  expect(value(await arrays.fold(source, 0 as unknown as bigint, folded, trace))).toBe(3n);
  expect(initialCalls).toBe(2);
  expect(integerWorker(unary)).toBeDefined();
});

test("non-leading captures pair by ordinal with exact reconstructed order", async () => {
  const captures: unknown[] = [5n];
  let adapterCalls = 0;
  const action = ownCallable(
    "p::trailing#0",
    "app::trailing",
    captures,
    async () => {
      adapterCalls++;
      throw new Error("slow adapter reached");
    },
    [],
    undefined,
    {
      companion: (item: bigint, capture: bigint) => item + capture,
      positions: [1],
      arity: 1,
      origin: callbackOrigin,
    },
  );
  expect(integerWorker(action)?.arity).toBe(1);
  expect(integerWorker(action)?.run(2n)).toBe(7n);
  const result = value(await arrays.map(array([1n, 2n]), action, trace));
  expect(result).toEqual([6n, 7n]);
  expect(adapterCalls).toBe(0);
  captures[0] = 500n;
  expect(value(await arrays.map(array([1n]), action, trace))).toEqual([6n]);
  expect(adapterCalls).toBe(0);
});

test("separated captures bind each ordinal value to its original position", async () => {
  let adapterCalls = 0;
  const action = ownCallable(
    "p::sandwich#0",
    "app::sandwich",
    [100n, 1n],
    async () => {
      adapterCalls++;
      throw new Error("slow adapter reached");
    },
    [],
    undefined,
    {
      companion: (prefix: bigint, item: bigint, suffix: bigint) => prefix + item + suffix,
      positions: [0, 2],
      arity: 1,
      origin: callbackOrigin,
    },
  );
  expect(integerWorker(action)?.run(5n)).toBe(106n);
  const result = value(await arrays.map(array([5n, 0n]), action, trace));
  expect(result).toEqual([106n, 101n]);
  expect(adapterCalls).toBe(0);
  expect(Object.isFrozen(result)).toBe(true);
});

test("capture-count and position mismatches decline with slow behavior intact", async () => {
  const cases: Array<[string, unknown[], number[], number, number]> = [
    ["extra capture", [1n, 2n], [0], 1, 2],
    ["missing capture", [1n], [0, 1], 1, 3],
    ["position beyond arity", [1n], [5], 1, 2],
    ["empty positions with captures", [1n], [], 1, 1],
  ];
  for (const [name, captures, positions, arity, params] of cases) {
    let adapterCalls = 0;
    const companion = (...operands: bigint[]) => {
      void operands;
      return 0n;
    };
    Object.defineProperty(companion, "length", { value: params, configurable: true });
    const action = ownCallable(
      "p::worker#0",
      "app::worker",
      captures,
      async (item: bigint) => {
        adapterCalls++;
        return success(item * 2n);
      },
      [],
      undefined,
      { companion, positions, arity, origin: callbackOrigin },
    );
    expect(integerWorker(action)).toBeUndefined();
    const out = value(await arrays.map(array([1n, 2n]), action, trace));
    expect([name, out]).toEqual([name, [2n, 4n]]);
    expect(adapterCalls).toBe(2);
  }
});

test("frozen Array-shaped non-arrays keep the conservative adapter", async () => {
  const shaped = Object.create(Array.prototype, {
    length: { value: 2, writable: false, enumerable: false, configurable: false },
    "0": { value: 1n, writable: false, enumerable: true, configurable: false },
    "1": { value: 2n, writable: false, enumerable: true, configurable: false },
  });
  Object.freeze(shaped);
  expect(Array.isArray(shaped)).toBe(false);
  expect(Object.getPrototypeOf(shaped)).toBe(Array.prototype);
  expect(Object.isFrozen(shaped)).toBe(true);
  let adapterCalls = 0;
  const action = ownCallable(
    "p::worker#0",
    "app::worker",
    [],
    async (item: bigint) => {
      adapterCalls++;
      return success(item * 2n);
    },
    [],
    undefined,
    doubleDescriptor((item) => item * 2n),
  );
  const out = value(await arrays.map(shaped as readonly bigint[], action, trace));
  expect(out).toEqual([2n, 4n]);
  expect(adapterCalls).toBe(2);
});

test("fake positions are refused with zero accessor execution", async () => {
  let getterCalls = 0;
  const fake = Object.create(Array.prototype, {
    length: {
      get: () => {
        getterCalls++;
        return 1;
      },
      enumerable: false,
      configurable: true,
    },
    "0": { value: 0, writable: false, enumerable: true, configurable: true },
  });
  const valid = {
    companion: (item: bigint) => item * 2n,
    positions: [] as number[],
    arity: 1,
    origin: callbackOrigin,
  };
  for (const [name, positions] of [
    ["array-shaped with accessor length", fake],
    ["proxied positions", new Proxy([0], {})],
  ] as const) {
    let message = "";
    try {
      ownCallable("p::worker#0", "app::worker", [], async () => success(0n), [], undefined, {
        ...valid,
        positions: positions as never,
      });
    } catch (error) {
      message = (error as Error).message;
    }
    expect([name, message]).toEqual([name, "invalid integer worker descriptor"]);
  }
  expect(getterCalls).toBe(0);
  // Own data lengths that disagree with slots or arity throw the same way.
  // (Fractional lengths are unrepresentable: real arrays reject them and
  // fakes never reach the length read.)
  const sparse: number[] = [0];
  sparse.length = 2;
  const badLengths: Array<[string, unknown]> = [
    ["sparse length beyond companion", { ...valid, positions: sparse }],
    [
      "slot count beyond arity equation",
      {
        ...valid,
        companion: (first: bigint, second: bigint) => first + second,
        positions: [0, 1],
      },
    ],
  ];
  for (const [name, descriptor] of badLengths) {
    let message = "";
    try {
      ownCallable(
        "p::worker#0",
        "app::worker",
        [],
        async () => success(0n),
        [],
        undefined,
        descriptor as never,
      );
    } catch (error) {
      message = (error as Error).message;
    }
    expect([name, message]).toEqual([name, "invalid integer worker descriptor"]);
  }
});
