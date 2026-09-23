import { expect, test } from "bun:test";
import { index, slice, intDivide, intRemainder, intPower, primitiveFailureKind } from "./primitive";

function failure(action: () => unknown): string | undefined {
  try {
    action();
  } catch (error) {
    return primitiveFailureKind(error);
  }
  throw new Error("expected a primitive fault");
}

test("native bigint contract and exact fault identities", () => {
  expect(intDivide(-7n, 3n)).toBe(-2n);
  expect(intRemainder(-7n, 3n)).toBe(-1n);
  expect(intPower(0n, 0n)).toBe(1n);
  const huge = 10n ** 100n;
  expect(intDivide(huge * 3n, 3n)).toBe(huge);
  expect(failure(() => intDivide(1n, 0n))).toBe("arithmetic");
  expect(failure(() => intRemainder(1n, 0n))).toBe("arithmetic");
  expect(failure(() => intPower(2n, -1n))).toBe("arithmetic");
  expect(primitiveFailureKind(new Error("Can arithmetic failure"))).toBeUndefined();
  const proxy = new Proxy(
    {},
    {
      get() {
        throw new Error("must not inspect");
      },
    },
  );
  expect(primitiveFailureKind(proxy)).toBeUndefined();
});

test("index validates bigint range before narrowing", () => {
  const values = Object.freeze([10n, 20n, 30n]);
  expect(index(values, 0n)).toBe(10n);
  expect(index(values, 2n)).toBe(30n);
  for (const position of [-1n, 3n, 2n ** 53n, 10n ** 1000n]) {
    expect(failure(() => index(values, position))).toBe("bounds");
    expect(failure(() => index("abc", position))).toBe("bounds");
  }
  expect(failure(() => index([], 0n))).toBe("bounds");
  const astral = "A😀B";
  expect(astral.length).toBe(4);
  expect(index(astral, 1n).charCodeAt(0)).toBe(0xd83d);
  expect(index(astral, 2n).charCodeAt(0)).toBe(0xde00);
});

test("bigint-normalized native slicing and UTF-16 boundaries", () => {
  const values = Object.freeze([0n, 1n, 2n, 3n]);
  expect(slice(values, -3n, -1n)).toEqual([1n, 2n]);
  expect(slice(values, 3n, 1n)).toEqual([]);
  expect(slice(values, -(10n ** 1000n), 10n ** 1000n)).toEqual(values);
  expect(slice(values, 10n ** 1000n)).toEqual([]);
  expect(slice(values, undefined, -(10n ** 1000n))).toEqual([]);
  const copied = slice(values);
  expect(copied).not.toBe(values);
  expect(Object.isFrozen(copied)).toBe(true);
  expect(slice("A😀B", 1n, 3n)).toBe("😀");
  expect(slice("A😀B", 1n, 2n).charCodeAt(0)).toBe(0xd83d);
  expect(slice("A😀B", -1n)).toBe("B");
  expect(slice("", -(10n ** 100n), 10n ** 100n)).toBe("");
});

test("finite slice bounds agree with native array/string slice", () => {
  const values = Object.freeze(["A", "😀", "B", "C"]);
  const text = "A😀BC";
  for (let start = -10; start <= 10; start++) {
    for (let end = -10; end <= 10; end++) {
      expect(slice(values, BigInt(start), BigInt(end))).toEqual(values.slice(start, end));
      expect(slice(text, BigInt(start), BigInt(end))).toBe(text.slice(start, end));
    }
  }
});
