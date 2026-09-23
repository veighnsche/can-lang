import { test, expect } from "bun:test";
import { Budget, CodecIssue, type Reason } from "../codec/budget.ts";
import { decodeInteger, encodeInteger } from "../codec/numbers.ts";
function decode(token: string, budget = new Budget()) {
  JSON.parse(token);
  return decodeInteger(token, budget, "");
}
function fails(run: () => unknown, reason: Reason) {
  try {
    run();
    throw Error("accepted invalid number");
  } catch (e) {
    expect(e).toBeInstanceOf(CodecIssue);
    expect((e as CodecIssue).reason).toBe(reason);
  }
}
test("native numeric tokens normalize exactly without binary64 loss", () => {
  for (const [token, value] of [
    ["9007199254740993", 9007199254740993n],
    ["9007199254740993.0", 9007199254740993n],
    ["90071992547409930e-1", 9007199254740993n],
    ["1.25e2", 125n],
    ["100e-2", 1n],
    ["-10.00", -10n],
    ["-0", 0n],
    ["-0.000", 0n],
    ["0e999999999999999999999", 0n],
    ["-0e-999999999999999999999", 0n],
  ] as const)
    expect(decode(token)).toBe(value);
  for (const token of ["1.25", "100e-3", "1e-999999999999999999999", "-0.1"])
    fails(() => decode(token), "integer_token");
  fails(() => decode("1e999999999999999999999"), "byte_limit");
});
test("canonical integer bytes are cumulative and signs consume their own byte", () => {
  expect(decode("1e7", new Budget(8))).toBe(10000000n);
  fails(() => decode("1e8", new Budget(8)), "byte_limit");
  expect(decode("-1e6", new Budget(8))).toBe(-1000000n);
  fails(() => decode("-1e7", new Budget(8)), "byte_limit");
  const budget = new Budget(12);
  expect(decode("1e5", budget)).toBe(100000n);
  expect(decode("1e5", budget)).toBe(100000n);
  fails(() => decode("0", budget), "byte_limit");
  const exhausted = new Budget(8);
  exhausted.charge(8, "");
  fails(() => decode("1.1", exhausted), "integer_token");
});
test("encoding bounds are proven before decimal formatting", () => {
  expect(encodeInteger(999n, new Budget(3), "")).toBe("999");
  expect(encodeInteger(-99n, new Budget(3), "")).toBe("-99");
  fails(() => encodeInteger(1000n, new Budget(3), ""), "byte_limit");
  fails(() => encodeInteger(-100n, new Budget(3), ""), "byte_limit");
  const enormous = 10n ** 100000n;
  fails(() => encodeInteger(enormous, new Budget(8), ""), "byte_limit");
  fails(() => encodeInteger(-enormous, new Budget(8), ""), "byte_limit");
  expect(encodeInteger(0n, new Budget(1), "")).toBe("0");
});
