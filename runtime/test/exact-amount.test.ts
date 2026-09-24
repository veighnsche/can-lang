import { test, expect } from "bun:test";
import { createHash } from "node:crypto";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, domainFailureDiagnostics, type FailureShape } from "../domain.ts";
import { createExactAmounts } from "../number.ts";
import { value } from "../completion.ts";
import { recordIdentity } from "../data.ts";
const shape = (kind: string, declaration: string): FailureShape => ({
  identity: createHash("sha256")
    .update("can-concrete-type-v1\0" + JSON.stringify([kind, declaration]))
    .digest("hex"),
  kind,
  declaration,
  arguments: [],
  fields: [],
  leaves: [],
  inputs: [],
  errors: [],
});
const integer = shape("primitive", "int"),
  error = catalogue.errors.find((e) => e.name === "number::zero_divisor")!;
const zero = shape("error", error.identity);
const records = catalogue.types
  .filter((t) => ["number::division", "number::rounded"].includes(t.name))
  .map((t) => ({
    ...shape("record", t.identity),
    fields: t.fields.map((f) => ({ name: f.name, type: integer.identity })),
  }));
const division = records.find((t) => t.declaration?.endsWith("::division"))!,
  rounded = records.find((t) => t.declaration?.endsWith("::rounded"))!;
const domain = createDomainRuntime({
  declarations: [{ identity: error.identity, name: error.name, parameters: 0 }],
  shapes: [integer, zero, ...records],
});
const api = createExactAmounts(domain, {
  zeroDivisor: zero.identity,
  division: division.identity,
  rounded: rounded.identity,
});
const abs = (n: bigint) => (n < 0n ? -n : n);

test("signed divmod preserves the native truncating identity and Euclidean range", async () => {
  for (let n = -33n; n <= 33n; n++)
    for (let d = -9n; d <= 9n; d++)
      if (d !== 0n) {
        const truncated = value(await api.divmod(n, d)),
          euclidean = value(await api.euclideanDivmod(n, d));
        expect(truncated.quotient).toBe(n / d);
        expect(truncated.remainder).toBe(n % d);
        expect(n).toBe(euclidean.quotient * d + euclidean.remainder);
        expect(euclidean.remainder >= 0n && euclidean.remainder < abs(d)).toBe(true);
        expect(recordIdentity(truncated)).toBe(division.identity);
        expect(Object.isFrozen(euclidean)).toBe(true);
      }
});

test("half-even vectors cover both signs, tie parity and normalized denominators", async () => {
  for (const [n, d, q, r] of [
    [5n, 2n, 2n, 1n],
    [7n, 2n, 4n, -1n],
    [-5n, 2n, -2n, -1n],
    [-7n, 2n, -4n, 1n],
    [8n, 3n, 3n, -1n],
    [5n, -2n, -2n, -1n],
    [-5n, -2n, 2n, 1n],
    [1n, 2n, 0n, 1n],
    [-1n, 2n, 0n, -1n],
    [0n, -3n, 0n, 0n],
  ]) {
    const result = value(await api.roundRatioHalfEven(n, d));
    expect(result).toMatchObject({ value: q, remainder_numerator: r, denominator: abs(d) });
    expect(recordIdentity(result)).toBe(rounded.identity);
    expect(Object.isFrozen(result)).toBe(true);
  }
});

test("rounded results conserve the exact rational and minimize distance with even ties", async () => {
  for (let n = -75n; n <= 75n; n++)
    for (let d = -11n; d <= 11n; d++)
      if (d !== 0n) {
        const result = value(await api.roundRatioHalfEven(n, d)),
          r = result.remainder_numerator,
          positive = result.denominator;
        // Cross multiplication and neighboring distances independently characterize
        // the result, without duplicating the adapter's truncation/adjustment steps.
        expect(n * positive).toBe((result.value * positive + r) * d);
        expect(positive > 0n).toBe(true);
        expect(abs(r) <= abs(r - positive) && abs(r) <= abs(r + positive)).toBe(true);
        if (2n * abs(r) === positive) expect(result.value % 2n).toBe(0n);
      }
});

test("large minor units and unreduced remainders stay exact beyond binary64", async () => {
  const balance = 9007199254740993n + 2n;
  expect(balance).toBe(9007199254740995n);
  const amount = value(await api.roundRatioHalfEven(balance, 2n));
  expect(amount).toMatchObject({
    value: 4503599627370498n,
    remainder_numerator: -1n,
    denominator: 2n,
  });
  const huge = 10n ** 500n + 7n;
  for (const d of [3n, -3n, 10n ** 250n + 1n]) {
    const result = value(await api.roundRatioHalfEven(huge, d));
    expect(huge * result.denominator).toBe(
      (result.value * result.denominator + result.remainder_numerator) * d,
    );
  }
  expect(value(await api.roundRatioHalfEven(2n, 4n))).toMatchObject({
    value: 0n,
    remainder_numerator: 2n,
    denominator: 4n,
  });
});

test("each zero denominator produces the allocated domain error", async () => {
  for (const operation of [api.divmod, api.euclideanDivmod, api.roundRatioHalfEven]) {
    const result = await operation(1n, 0n);
    expect(result.kind).toBe("domain");
    if (result.kind !== "domain") throw Error("missing zero divisor domain failure");
    const details = domainFailureDiagnostics(result.value);
    expect(details.declaration.name).toBe("number::zero_divisor");
    expect(recordIdentity(details.payload)).toBe(zero.identity);
  }
});
