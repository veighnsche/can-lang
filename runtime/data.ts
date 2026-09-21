// Compiler-private representation of ordinary nominal Can data. Authored code
// cannot import this module or obtain the nominal key. Opaque resources use
// their own maintained constructors and never enter these ordinary adapters.
const nominal = Symbol("can.nominal");
export type RecordValue = Readonly<Record<string | symbol, unknown>>;

export function record(identity: string, fields: readonly (readonly [string, unknown])[]): RecordValue {
  const value = Object.create(null);
  Object.defineProperty(value, nominal, { value: identity, enumerable: true });
  for (const [name, field] of fields) {
    Object.defineProperty(value, name, { value: field, enumerable: true });
  }
  return Object.freeze(value);
}

export function update(original: RecordValue, replacements: readonly (readonly [string, unknown])[]): RecordValue {
  // Generated arguments evaluate receiver once, then replacements in source
  // order. Own data properties and immutable field references need no deep copy.
  const value = Object.assign(Object.create(null), original);
  for (const [name, replacement] of replacements) {
    Object.defineProperty(value, name, { value: replacement, enumerable: true, configurable: true, writable: true });
  }
  return Object.freeze(value);
}

export function array<T>(elements: T[]): readonly T[] {
  // Only a fresh generated array literal may enter here; no mutable native alias
  // escapes. Native adapters must transfer ownership or copy before this call.
  return Object.freeze(elements);
}
