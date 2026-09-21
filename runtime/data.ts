import { types as nativeTypes } from "node:util";
// Compiler-private representation of ordinary nominal Can data. Authored code
// cannot import this module or obtain the nominal key. Opaque resources use
// their own maintained constructors and never enter these ordinary adapters.
const nominal = Symbol("can.nominal");
const nominalRecords = new WeakMap<object, string>();
// Maintained opaque containers can retain resource-bearing data without exposing
// their backing storage. Evidence records containment only; the owner acquires
// leases at participant preparation, never at container construction.
const opaqueChildren = new WeakMap<object, readonly unknown[]>();
export function registerOpaqueContents(token: object, values: readonly unknown[]): void {
  if (opaqueChildren.has(token)) throw new TypeError("opaque contents already registered");
  opaqueChildren.set(token, Object.freeze([...values]));
}
export function opaqueContents(token: object): readonly unknown[] | undefined {
  return opaqueChildren.get(token);
}

export type RecordValue = Readonly<Record<string | symbol, unknown>>;

export function record(identity: string, fields: readonly (readonly [string, unknown])[]): RecordValue {
  const value = Object.create(null);
  Object.defineProperty(value, nominal, { value: identity, enumerable: true });
  for (const [name, field] of fields) {
    Object.defineProperty(value, name, { value: field, enumerable: true });
  }
  nominalRecords.set(value, identity);
  return Object.freeze(value);
}

export function update(original: RecordValue, replacements: readonly (readonly [string, unknown])[]): RecordValue {
  // Generated arguments evaluate receiver once, then replacements in source
  // order. Own data properties and immutable field references need no deep copy.
  const identity = recordIdentity(original);
  if (!identity) throw new TypeError("invalid ordinary record");
  const value = Object.assign(Object.create(null), original);
  for (const [name, replacement] of replacements) {
    Object.defineProperty(value, name, { value: replacement, enumerable: true, configurable: true, writable: true });
  }
  nominalRecords.set(value, identity);
  return Object.freeze(value);
}

export function array<T>(elements: T[]): readonly T[] {
  // Only a fresh generated array literal may enter here; no mutable native alias
  // escapes. Native adapters must transfer ownership or copy before this call.
  return Object.freeze(elements);
}

export function recordIdentity(value: unknown): string | undefined {
  return value !== null && (typeof value === "object" || typeof value === "function") ? nominalRecords.get(value) : undefined;
}
export function dataKeys(value: unknown): (string | symbol)[] {
  if (value === null || typeof value !== "object" || nativeTypes.isProxy(value)) throw new TypeError("expected non-proxy data object");
  return Reflect.ownKeys(value);
}
export function dataProperty(value: unknown, key: string | symbol): unknown {
  if (value === null || typeof value !== "object" || nativeTypes.isProxy(value)) throw new TypeError("expected non-proxy data object");
  const descriptor = Object.getOwnPropertyDescriptor(value, key);
  if (!descriptor || !("value" in descriptor)) throw new TypeError("expected own data property");
  return descriptor.value;
}
export function dataArray(value: unknown): unknown[] {
  dataKeys(value);
  if (!Array.isArray(value)) throw new TypeError("expected data array");
  const length = dataProperty(value, "length") as number;
  if (dataKeys(value).length !== length + 1) throw new TypeError("expected dense data array");
  const out: unknown[] = [];
  for (let index = 0; index < length; index++) out.push(dataProperty(value, String(index)));
  return out;
}
