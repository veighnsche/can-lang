import { test, expect } from "bun:test";
import { array, record, update, registerOpaqueContents, opaqueContents } from "./data";

test("native strict deep equality observes nominal identity", () => {
  const one = record("root/a::item<int>", [["value", 1n]]);
  const same = record("root/a::item<int>", [["value", 1n]]);
  const otherOwner = record("root/b::item<int>", [["value", 1n]]);
  const otherArgument = record("root/a::item<str>", [["value", 1n]]);
  expect(Bun.deepEquals(one, same, true)).toBe(true);
  expect(Bun.deepEquals(one, otherOwner, true)).toBe(false);
  expect(Bun.deepEquals(one, otherArgument, true)).toBe(false);
  expect(Bun.deepEquals(array([one]), array([otherOwner]), true)).toBe(false);
  expect(
    Bun.deepEquals(record("float", [["value", NaN]]), record("float", [["value", NaN]]), true),
  ).toBe(true);
  expect(
    Bun.deepEquals(record("float", [["value", 0]]), record("float", [["value", -0]]), true),
  ).toBe(false);
});

test("opaque registration copies array clients and freezes the snapshot", () => {
  const a = Object.freeze({ value: 1 });
  const b = Object.freeze({ value: 2 });
  const input: unknown[] = [a, b];
  const token = {};
  registerOpaqueContents(token, input);
  const snapshot = opaqueContents(token)!;
  expect(snapshot).not.toBe(input);
  expect(snapshot.length).toBe(2);
  expect(snapshot[0]).toBe(a);
  expect(snapshot[1]).toBe(b);
  expect(Object.isFrozen(snapshot)).toBe(true);
  input.push(Object.freeze({ value: 3 }));
  input[0] = b;
  expect(snapshot.length).toBe(2);
  expect(snapshot[0]).toBe(a);
  expect(snapshot[1]).toBe(b);
  expect(() => registerOpaqueContents(token, [a])).toThrow("opaque contents already registered");
  expect(opaqueContents({})).toBeUndefined();
});

test("opaque registration consumes single-pass iterables once with exact references", () => {
  const a = Object.freeze({ value: 1 });
  const b = Object.freeze({ value: 2 });
  let iterations = 0;
  const singlePass: Iterable<unknown> = {
    [Symbol.iterator]() {
      iterations++;
      if (iterations > 1) throw new Error("iterable consumed twice");
      let index = 0;
      const items = [a, b];
      return {
        next() {
          return index < items.length
            ? { value: items[index++], done: false }
            : { value: undefined, done: true };
        },
      };
    },
  };
  const token = {};
  registerOpaqueContents(token, singlePass);
  expect(iterations).toBe(1);
  const snapshot = opaqueContents(token)!;
  expect(snapshot.length).toBe(2);
  expect(snapshot[0]).toBe(a);
  expect(snapshot[1]).toBe(b);
  expect(Object.isFrozen(snapshot)).toBe(true);

  const backing = new Map<string, unknown>([
    ["k1", a],
    ["k2", b],
  ]);
  const mapToken = {};
  registerOpaqueContents(mapToken, backing.values());
  const mapSnapshot = opaqueContents(mapToken)!;
  expect(mapSnapshot.length).toBe(2);
  expect(mapSnapshot[0]).toBe(a);
  expect(mapSnapshot[1]).toBe(b);
  expect(Object.isFrozen(mapSnapshot)).toBe(true);
});

test("duplicate registration rejects before iteration and failed iteration leaves no metadata", () => {
  const a = Object.freeze({ value: 1 });
  const b = Object.freeze({ value: 2 });
  const token = {};
  registerOpaqueContents(token, [a]);
  let touched = false;
  const evil: Iterable<unknown> = {
    [Symbol.iterator]() {
      touched = true;
      throw new Error("must not iterate");
    },
  };
  expect(() => registerOpaqueContents(token, evil)).toThrow("opaque contents already registered");
  expect(touched).toBe(false);
  expect(opaqueContents(token)![0]).toBe(a);

  const retryToken = {};
  const throwing: Iterable<unknown> = {
    *[Symbol.iterator]() {
      yield a;
      throw new Error("boom");
    },
  };
  expect(() => registerOpaqueContents(retryToken, throwing)).toThrow("boom");
  expect(opaqueContents(retryToken)).toBeUndefined();
  registerOpaqueContents(retryToken, [b]);
  expect(opaqueContents(retryToken)![0]).toBe(b);
});

test("native copy preserves original aliases, tags and field spelling", () => {
  const shared = array([1n, 2n]);
  const original = record("item", [
    ["value", 1n],
    ["shared", shared],
    ["__proto__", "data"],
    ["then", "ordinary field"],
  ]);
  const changed = update(original, [["value", 3n]]);
  expect(original.value).toBe(1n);
  expect(changed.value).toBe(3n);
  expect(changed.shared).toBe(shared);
  expect(Object.getPrototypeOf(changed)).toBe(null);
  expect(changed.__proto__).toBe("data");
  expect(changed.then).toBe("ordinary field");
  expect(Object.isFrozen(original)).toBe(true);
  expect(Object.isFrozen(changed)).toBe(true);
  expect(Object.isFrozen(shared)).toBe(true);
  expect(
    Bun.deepEquals(
      changed,
      record("item", [
        ["value", 3n],
        ["shared", shared],
        ["__proto__", "data"],
        ["then", "ordinary field"],
      ]),
      true,
    ),
  ).toBe(true);
});
