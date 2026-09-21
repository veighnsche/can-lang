import { test, expect } from "bun:test";
import { array, record, update } from "./data";

test("native strict deep equality observes nominal identity", () => {
  const one = record("root/a::item<int>", [["value", 1n]]);
  const same = record("root/a::item<int>", [["value", 1n]]);
  const otherOwner = record("root/b::item<int>", [["value", 1n]]);
  const otherArgument = record("root/a::item<str>", [["value", 1n]]);
  expect(Bun.deepEquals(one, same, true)).toBe(true);
  expect(Bun.deepEquals(one, otherOwner, true)).toBe(false);
  expect(Bun.deepEquals(one, otherArgument, true)).toBe(false);
  expect(Bun.deepEquals(array([one]), array([otherOwner]), true)).toBe(false);
  expect(Bun.deepEquals(record("float", [["value", NaN]]), record("float", [["value", NaN]]), true)).toBe(true);
  expect(Bun.deepEquals(record("float", [["value", 0]]), record("float", [["value", -0]]), true)).toBe(false);
});

test("native copy preserves original aliases, tags and field spelling", () => {
  const shared = array([1n, 2n]);
  const original = record("item", [["value", 1n], ["shared", shared], ["__proto__", "data"], ["then", "ordinary field"]]);
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
  expect(Bun.deepEquals(changed, record("item", [["value", 3n], ["shared", shared], ["__proto__", "data"], ["then", "ordinary field"]]), true)).toBe(true);
});
