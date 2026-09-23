// Primitive contract adapters around the qualified native Bun operations.
// I49's standard-failure boundary consumes this private identity classification;
// no user-thrown property or message can impersonate a primitive fault.
type PrimitiveFailureKind = "arithmetic" | "bounds";
const faults = new WeakMap<object, { kind: PrimitiveFailureKind; message: string }>();
function fail(kind: PrimitiveFailureKind, message: string): never {
  const error = new Error(message);
  faults.set(error, { kind, message });
  throw error;
}
export function primitiveFailureKind(value: unknown): PrimitiveFailureKind | undefined {
  return value !== null && (typeof value === "object" || typeof value === "function")
    ? faults.get(value)?.kind
    : undefined;
}
export function intDivide(left: bigint, right: bigint): bigint {
  if (right === 0n) fail("arithmetic", "arithmetic: integer division by zero");
  return left / right;
}
export function intRemainder(left: bigint, right: bigint): bigint {
  if (right === 0n) fail("arithmetic", "arithmetic: integer remainder by zero");
  return left % right;
}
export function intPower(left: bigint, right: bigint): bigint {
  if (right < 0n) fail("arithmetic", "arithmetic: negative integer exponent");
  return left ** right;
}
export function index<T>(value: readonly T[], position: bigint): T;
export function index(value: string, position: bigint): string;
export function index<T>(value: readonly T[] | string, position: bigint): T | string {
  if (position < 0n || position >= BigInt(value.length))
    fail("bounds", "bounds: index out of range");
  // Native finite array/string lengths are exactly representable; conversion
  // occurs only after proving the index belongs to that finite range.
  return value[Number(position)];
}
function sliceBound(value: bigint, length: bigint): number {
  const relative = value < 0n ? length + value : value;
  return Number(relative < 0n ? 0n : relative > length ? length : relative);
}
export function slice<T>(value: readonly T[], start?: bigint, end?: bigint): readonly T[];
export function slice(value: string, start?: bigint, end?: bigint): string;
export function slice<T>(
  value: readonly T[] | string,
  start?: bigint,
  end?: bigint,
): readonly T[] | string {
  const length = BigInt(value.length);
  const from = start === undefined ? 0 : sliceBound(start, length);
  const to = end === undefined ? value.length : sliceBound(end, length);
  const result = value.slice(from, to);
  return typeof result === "string" ? result : Object.freeze(result);
}

export function primitiveFailureMessage(value: unknown): string | undefined {
  return value !== null && (typeof value === "object" || typeof value === "function")
    ? faults.get(value)?.message
    : undefined;
}
