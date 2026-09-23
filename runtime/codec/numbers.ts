import { Budget, reject } from "./budget.ts";

// Compare a source exponent with a bounded native count without constructing a
// bigint or Number from the exponent. The token was validated by JSON.parse.
function compareExponent(source: string, bound: number): number {
  let negative = source[0] === "-";
  let start = source[0] === "-" || source[0] === "+" ? 1 : 0;
  while (source[start] === "0") start++;
  const digits = source.slice(start);
  if (!digits.length) return bound === 0 ? 0 : bound > 0 ? -1 : 1;
  const boundNegative = bound < 0;
  if (negative !== boundNegative) return negative ? -1 : 1;
  const limit = String(Math.abs(bound));
  const magnitude =
    digits.length !== limit.length
      ? digits.length < limit.length
        ? -1
        : 1
      : digits === limit
        ? 0
        : digits < limit
          ? -1
          : 1;
  return negative ? -magnitude : magnitude;
}

export function decodeInteger(token: string, budget: Budget, path: string): bigint {
  // This recognizes components of an already native-validated numeric token;
  // it is not used to admit JSON syntax or parse complete documents.
  if (token.length > budget.bytes) reject(path, "byte_limit");
  const parts = /^(-?)(\d+)(?:\.(\d+))?(?:[eE]([+-]?\d+))?$/.exec(token);
  if (!parts) throw new TypeError("missing native numeric token evidence");
  const fraction = parts[3] ?? "";
  const coefficient = (parts[2] + fraction).replace(/^0+/, "");
  if (coefficient.length === 0) {
    budget.charge(1, path);
    return 0n;
  }
  let trailing = 0;
  while (coefficient[coefficient.length - 1 - trailing] === "0") trailing++;
  const exponent = parts[4] ?? "0";
  if (compareExponent(exponent, fraction.length - trailing) < 0) reject(path, "integer_token");
  const sign = parts[1] ? 1 : 0;
  if (compareExponent(exponent, budget.remaining - sign - coefficient.length + fraction.length) > 0)
    reject(path, "byte_limit");
  // Both comparisons now prove an exactly representable bounded native count.
  const shift = Number(exponent) - fraction.length;
  const size = coefficient.length + shift + sign;
  budget.charge(size, path);
  const canonical =
    parts[1] +
    (shift >= 0
      ? coefficient + "0".repeat(shift)
      : coefficient.slice(0, coefficient.length + shift));
  return BigInt(canonical);
}

export function encodeInteger(value: bigint, budget: Budget, path: string): string {
  const sign = value < 0n ? 1 : 0;
  const digits = budget.remaining - sign;
  if (digits < 1) reject(path, "byte_limit");
  // Small values avoid a potentially budget-sized exponentiation. Every path
  // still proves the output bound before native decimal formatting.
  if (
    !(
      budget.remaining >= 20 &&
      value > -10_000_000_000_000_000_000n &&
      value < 10_000_000_000_000_000_000n
    )
  ) {
    const threshold = 10n ** BigInt(digits);
    if (value >= threshold || value <= -threshold) reject(path, "byte_limit");
  }
  const text = String(value);
  budget.charge(text.length, path);
  return text;
}
