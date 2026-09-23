import { test, expect } from "bun:test";
import { createHash } from "node:crypto";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, domainFailureDiagnostics, type FailureShape } from "../domain.ts";
import { createMap, isMap } from "../collections/map.ts";
import { createSet, isSet, type ImmutableSet } from "../collections/set.ts";
import { record } from "../data.ts";
import { value, type Completion } from "../completion.ts";
const declarations = catalogue.errors.filter((e) => [1007, 1008].includes(e.id));
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
  declarations: declarations.map((e) => ({
    identity: e.identity,
    name: e.name,
    id: e.id,
    parameters: 0,
  })),
  shapes: errors,
});
const maps = createMap<bigint, object>(
  domain,
  {
    map: "map-int-object",
    entry: "entry-int-object",
    absent: errors[0].identity,
    exists: errors[1].identity,
  },
  "int",
);
function invalid(result: Completion, id: number) {
  expect(result.kind).toBe("domain");
  if (result.kind !== "domain") throw Error("expected domain");
  expect(domainFailureDiagnostics(result.value).declaration.id).toBe(id);
}
test("map copies preserve key order, aliases and nominal entries", async () => {
  const empty = value(await maps.empty()),
    a = Object.freeze({ value: 1 }),
    b = Object.freeze({ value: 2 });
  const first = value(await maps.insert(empty, 3n, a)),
    second = value(await maps.insert(first, 2n, b));
  const replaced = value(await maps.replace(second, 3n, b));
  expect(value(await maps.entries(empty))).toEqual([]);
  expect(value(await maps.get(first, 3n))).toBe(a);
  const entries = value(await maps.entries(replaced));
  expect(entries.map((e) => e.key)).toEqual([3n, 2n]);
  expect(entries[0].value).toBe(b);
  expect(Object.isFrozen(replaced)).toBe(true);
  expect(Object.isFrozen(entries)).toBe(true);
  expect(Object.isFrozen(entries[0])).toBe(true);
  const removed = value(await maps.remove(replaced, 3n));
  expect(value(await maps.entries(removed)).map((e) => e.key)).toEqual([2n]);
  expect(
    value(await maps.entries(value(await maps.insert(removed, 3n, a)))).map((e) => e.key),
  ).toEqual([2n, 3n]);
  invalid(await maps.get(empty, 3n), 1007);
  invalid(await maps.remove(empty, 3n), 1007);
  invalid(await maps.replace(empty, 3n, a), 1007);
  invalid(await maps.insert(first, 3n, b), 1008);
});
test("collection storage rejects forged, copied and wrong-specialization tokens and keys", async () => {
  const token = value(await maps.empty());
  expect(isMap("map-int-object", token)).toBe(true);
  expect(isMap("wrong", token)).toBe(false);
  for (const fake of [
    Object.freeze({}),
    Object.freeze({ ...token }),
    new Proxy(token, {}),
    record("map-int-object", []),
  ])
    await expect(maps.entries(fake as any)).rejects.toBeDefined();
  for (const key of [1, NaN, {}, "1", true])
    await expect(maps.insert(token, key as any, {})).rejects.toBeDefined();
  const sets = createSet<bigint>("set-int", "int"),
    set = value(await sets.empty());
  expect(isSet("set-int", set)).toBe(true);
  expect(isSet("wrong", set)).toBe(false);
  await expect(sets.contains(token as any, 1n)).rejects.toBeDefined();
  await expect(maps.entries(set as any)).rejects.toBeDefined();
  for (const key of [1, NaN, {}, "1", true])
    await expect(sets.add(set, key as any)).rejects.toBeDefined();
});
test("native set operations preserve left order through both intersection size branches", async () => {
  const sets = createSet<bigint>("set-int", "int");
  async function build(keys: bigint[]) {
    let set = value(await sets.empty());
    for (const key of keys) set = value(await sets.add(set, key));
    return set;
  }
  // Observe the receiver of a native union to inspect order without exposing storage.
  async function order(set: ImmutableSet<bigint>) {
    const original = Set.prototype.union;
    let observed: bigint[] = [];
    Set.prototype.union = function (this: Set<bigint>, other: any) {
      observed = Array.from(this) as bigint[];
      return original.call(this, other);
    } as any;
    try {
      await sets.union(set, value(await sets.empty()));
      return observed;
    } finally {
      Set.prototype.union = original;
    }
  }
  const left = await build([3n, 2n, 1n]),
    small = await build([1n, 2n]),
    large = await build([1n, 2n, 4n, 5n]);
  expect(await order(value(await sets.intersection(left, small)))).toEqual([2n, 1n]);
  expect(await order(value(await sets.intersection(left, large)))).toEqual([2n, 1n]);
  expect(await order(value(await sets.union(left, large)))).toEqual([3n, 2n, 1n, 4n, 5n]);
  expect(await order(value(await sets.difference(left, small)))).toEqual([3n]);
  expect(await order(value(await sets.add(left, 2n)))).toEqual([3n, 2n, 1n]);
  expect(await order(left)).toEqual([3n, 2n, 1n]);
  expect(await order(small)).toEqual([1n, 2n]);
  expect(value(await sets.contains(left, 3n))).toBe(true);
  expect(value(await sets.contains(left, 4n))).toBe(false);
  const empty = value(await sets.empty());
  expect(await order(value(await sets.intersection(left, empty)))).toEqual([]);
});
test("scalar key families retain native equality and matching factories share provenance", async () => {
  for (const [kind, keys] of [
    ["str", ["", "😀", "__proto__"]],
    ["bool", [false, true]],
  ] as const) {
    const api = createMap<any, object>(
      domain,
      {
        map: "map-" + kind,
        entry: "entry-" + kind,
        absent: errors[0].identity,
        exists: errors[1].identity,
      },
      kind,
    );
    // oxlint-disable no-thenable -- Map values may contain hostile then data without being assimilated.
    let map = value(await api.empty());
    const payload = Object.freeze({
      then: () => {
        throw Error("must not assimilate");
      },
    });
    // oxlint-enable no-thenable
    for (const key of keys) map = value(await api.insert(map, key, payload));
    expect(value(await api.entries(map)).map((e) => e.key)).toEqual([...keys]);
    expect(value(await api.get(map, keys[0]))).toBe(payload);
    const same = createMap<any, object>(
      domain,
      {
        map: "map-" + kind,
        entry: "entry-" + kind,
        absent: errors[0].identity,
        exists: errors[1].identity,
      },
      kind,
    );
    expect(value(await same.get(map, keys[0]))).toBe(payload);
    const wrong = createMap<any, object>(
      domain,
      {
        map: "different",
        entry: "entry-" + kind,
        absent: errors[0].identity,
        exists: errors[1].identity,
      },
      kind,
    );
    await expect(wrong.entries(map)).rejects.toBeDefined();
    const setAPI = createSet<any>("set-" + kind, kind);
    let set = value(await setAPI.empty());
    for (const key of keys) set = value(await setAPI.add(set, key));
    expect(value(await setAPI.contains(set, keys[0]))).toBe(true);
    expect(value(await createSet<any>("set-" + kind, kind).contains(set, keys[0]))).toBe(true);
    await expect(createSet<any>("different", kind).contains(set, keys[0])).rejects.toBeDefined();
  }
});
