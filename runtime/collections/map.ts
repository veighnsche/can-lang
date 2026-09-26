import { success, failure, type Completion, type AssertionContext } from "../completion.ts";
import { array, record, registerOpaqueContents } from "../data.ts";
import { createDomainRuntime } from "../domain.ts";
import { resourceStateFailure } from "../failure.ts";
import { checkKey, type Key, type KeyKind } from "./set.ts";
declare const brand: unique symbol;
export type ImmutableMap<K extends Key = Key, V = unknown> = Readonly<{
  readonly [brand]: readonly [K, V];
}>;
const storage = new WeakMap<object, { identity: string; values: Map<Key, unknown> }>();
const origin = Object.freeze({
  source: "can:collections:map",
  start: 0,
  end: 0,
  invocation: Object.freeze([] as string[]),
});
export function isMap(identity: string, value: unknown): value is object {
  return value !== null && typeof value === "object" && storage.get(value)?.identity === identity;
}
export function createMap<K extends Key, V>(
  domain: ReturnType<typeof createDomainRuntime>,
  identities: Readonly<{ map: string; entry: string; absent: string; exists: string }>,
  keyKind: KeyKind,
) {
  function own(values: Map<Key, unknown>): ImmutableMap<K, V> {
    const token = Object.freeze(Object.create(null));
    storage.set(token, { identity: identities.map, values });
    registerOpaqueContents(token, Array.from(values.values()));
    return token;
  }
  function backing(value: unknown): Map<Key, unknown> {
    if (!isMap(identities.map, value)) throw resourceStateFailure(undefined, origin);
    return storage.get(value)!.values;
  }
  function error(identity: string): Completion<never> {
    return failure(domain.create(identity, record(identity, []), origin));
  }
  return Object.freeze({
    async empty(_context?: AssertionContext) {
      return success(own(new Map()));
    },
    // Bulk construction from entry records: one native Map, single
    // immutable publication, no point-insert history copying.
    // Collision: the first duplicate key fails the whole build with
    // key_exists; no partial map escapes. Order: first-occurrence
    // insertion order. Per-element failure: keys validate in order and
    // the first invalid key throws before publication. Ownership: the
    // builder Map is function-local and moves into the published token;
    // it never escapes and the input array is only read.
    async build_map(
      entries: ReadonlyArray<Readonly<{ key: K; value: V }>>,
      _context?: AssertionContext,
    ): Promise<Completion<ImmutableMap<K, V>>> {
      const values = new Map<Key, unknown>();
      for (const entry of entries) {
        checkKey(keyKind, entry.key);
        if (values.has(entry.key)) return error(identities.exists);
        values.set(entry.key, entry.value);
      }
      return success(own(values));
    },
    async get(map: unknown, key: K, _context?: AssertionContext): Promise<Completion<V>> {
      const source = backing(map);
      checkKey(keyKind, key);
      return source.has(key) ? success(source.get(key) as V) : error(identities.absent);
    },
    async insert(
      map: unknown,
      key: K,
      value: V,
      _context?: AssertionContext,
    ): Promise<Completion<ImmutableMap<K, V>>> {
      const source = backing(map);
      checkKey(keyKind, key);
      if (source.has(key)) return error(identities.exists);
      return success(own(new Map(source).set(key, value)));
    },
    async replace(
      map: unknown,
      key: K,
      value: V,
      _context?: AssertionContext,
    ): Promise<Completion<ImmutableMap<K, V>>> {
      const source = backing(map);
      checkKey(keyKind, key);
      if (!source.has(key)) return error(identities.absent);
      return success(own(new Map(source).set(key, value)));
    },
    async remove(
      map: unknown,
      key: K,
      _context?: AssertionContext,
    ): Promise<Completion<ImmutableMap<K, V>>> {
      const source = backing(map);
      checkKey(keyKind, key);
      if (!source.has(key)) return error(identities.absent);
      const copy = new Map(source);
      copy.delete(key);
      return success(own(copy));
    },
    async entries(map: unknown, _context?: AssertionContext) {
      return success(
        array(
          Array.from(
            backing(map),
            ([key, value]) =>
              record(identities.entry, [
                ["key", key],
                ["value", value],
              ]) as Readonly<{ key: K; value: V }>,
          ),
        ),
      );
    },
  });
}
