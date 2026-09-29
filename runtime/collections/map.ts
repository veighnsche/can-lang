import { success, failure, caught, type Completion, type AssertionContext } from "../completion.ts";
import { array, record, registerOpaqueContents } from "../data.ts";
import { createDomainRuntime } from "../domain.ts";
import { resourceStateFailure, type FailureOrigin } from "../failure.ts";
import { checkKey, type Key, type KeyKind } from "./set.ts";
declare const brand: unique symbol;
export type ImmutableMap<K extends Key = Key, V = unknown> = Readonly<{
  readonly [brand]: readonly [K, V];
}>;
const storage = new WeakMap<object, { identity: string; values: Map<Key, unknown> }>();
// Private synchronous bodies behind the exact factory-owned async
// methods. Only these three canonical operations register; identity is
// by exact function object, never by name or property inspection.
const methodWorkers = new WeakMap<object, (...args: unknown[]) => Completion<unknown>>();
// Compiler-private query for proven leaf companions. Unknown functions
// refuse without inspection, so forged workers fail closed.
export function mapMethodWorker(
  fn: unknown,
): ((...args: unknown[]) => Completion<unknown>) | undefined {
  if ((typeof fn !== "object" && typeof fn !== "function") || fn === null) return undefined;
  return methodWorkers.get(fn);
}
// Private batch runner behind the exact factory-owned async methods. Run
// validates the genuine initial map and all-bigint values before any
// visit, clones once, applies the proven transitions with native
// has/get/set, and publishes through own() once; invalid map state
// declines with undefined before traversal and the first transition
// throw stops with the first-boundary origin. Only the three canonical
// operations register; identity is by exact function object, never by
// name or property inspection.
export type MapBatchRunner = Readonly<{
  run: (
    initial: unknown,
    keys: readonly unknown[],
    absent: (key: unknown) => bigint,
    present: (key: unknown, previous: bigint) => bigint,
    origin: FailureOrigin,
  ) => Completion<unknown> | undefined;
}>;
const batchRunners = new WeakMap<object, MapBatchRunner>();
// Compiler-private query for proven batch folds. Unknown functions
// refuse without inspection, so forged capabilities fail closed.
export function mapBatchCapability(fn: unknown): MapBatchRunner | undefined {
  if ((typeof fn !== "object" && typeof fn !== "function") || fn === null) return undefined;
  return batchRunners.get(fn);
}
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
    registerOpaqueContents(token, values.values());
    return token;
  }
  function backing(value: unknown): Map<Key, unknown> {
    // Single metadata lookup after the same non-null object and concrete
    // identity guard as isMap; no yield or user code ran between the former
    // two lookups. Public isMap, key checks, origins and native copies stay.
    const found = value !== null && typeof value === "object" ? storage.get(value) : undefined;
    if (found === undefined || found.identity !== identities.map)
      throw resourceStateFailure(undefined, origin);
    return found.values;
  }
  function error(identity: string): Completion<never> {
    return failure(domain.create(identity, record(identity, []), origin));
  }
  // Synchronous Completion bodies shared by the native async methods
  // below and proven leaf companions. All key/backing checks, Map
  // copies, insertion order, snapshots, opaque ownership, domain
  // occurrences and origins are identical; only the async wrapper and
  // its context parameter live in the methods.
  function getBody(map: unknown, key: unknown): Completion<V> {
    const source = backing(map);
    checkKey(keyKind, key);
    return source.has(key) ? success(source.get(key) as V) : error(identities.absent);
  }
  function insertBody(map: unknown, key: unknown, value: unknown): Completion<ImmutableMap<K, V>> {
    const source = backing(map);
    checkKey(keyKind, key);
    if (source.has(key)) return error(identities.exists);
    return success(own(new Map(source).set(key, value)));
  }
  function replaceBody(map: unknown, key: unknown, value: unknown): Completion<ImmutableMap<K, V>> {
    const source = backing(map);
    checkKey(keyKind, key);
    if (!source.has(key)) return error(identities.absent);
    return success(own(new Map(source).set(key, value)));
  }
  // Batch reduction for one proven transition. Guards run before the
  // first visit with no effect: genuine initial map of this exact
  // factory (wrong identity, proxy or non-object declines) and every
  // stored value a bigint. Empty input returns the initial token
  // without cloning. Otherwise one private clone is mutated in order
  // and published once through own() with independent opaque
  // ownership; the old backing Map is never mutated and no mutable
  // alias escapes. Absent keys take the insert value, present keys the
  // replace value over the stored bigint — the proof's impossible
  // recovery arms. No per-word token, Completion, Promise, snapshot
  // or origin object on success.
  function runBatch(
    initial: unknown,
    keys: readonly unknown[],
    absent: (key: unknown) => bigint,
    present: (key: unknown, previous: bigint) => bigint,
    origin: FailureOrigin,
  ): Completion<unknown> | undefined {
    const found =
      initial !== null && typeof initial === "object" ? storage.get(initial) : undefined;
    if (found === undefined || found.identity !== identities.map) return undefined;
    for (const current of found.values.values()) {
      if (typeof current !== "bigint") return undefined;
    }
    if (keys.length === 0) return success(initial as ImmutableMap<K, V>);
    const builder = new Map(found.values);
    try {
      for (const key of keys) {
        if (builder.has(key as Key)) {
          builder.set(key as Key, present(key, builder.get(key as Key) as bigint));
        } else {
          builder.set(key as Key, absent(key));
        }
      }
    } catch (cause) {
      return caught(cause, origin);
    }
    return success(own(builder));
  }
  const methods = Object.freeze({
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
      return getBody(map, key);
    },
    async insert(
      map: unknown,
      key: K,
      value: V,
      _context?: AssertionContext,
    ): Promise<Completion<ImmutableMap<K, V>>> {
      return insertBody(map, key, value);
    },
    async replace(
      map: unknown,
      key: K,
      value: V,
      _context?: AssertionContext,
    ): Promise<Completion<ImmutableMap<K, V>>> {
      return replaceBody(map, key, value);
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
  methodWorkers.set(methods.get, getBody);
  methodWorkers.set(methods.insert, insertBody);
  methodWorkers.set(methods.replace, replaceBody);
  const runner = Object.freeze({
    run: runBatch,
  });
  batchRunners.set(methods.get, runner);
  batchRunners.set(methods.insert, runner);
  batchRunners.set(methods.replace, runner);
  return methods;
}
