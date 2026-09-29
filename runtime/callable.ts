import { contextIdentity, type AssertionContext } from "./assert/context.ts";
import {
  callableIdentity,
  initializationCallableIdentity,
  type CallableIdentity,
} from "./assert/identity.ts";
import { caught, type Completion } from "./completion.ts";
import type { FailureOrigin } from "./failure.ts";
import { registerCallableCaptures } from "./owner.ts";
import { isHostProxy } from "./reflect.ts";
import { mapBatchCapability, type MapBatchRunner } from "./collections/map.ts";
// Native closures own execution and captures. This private receipt preserves
// creation-site/target/capture evidence for fixture identity; it is never a Can
// data projection or a diagnostic serialization surface.
type Receipt = Readonly<{
  identity: CallableIdentity;
  site: string;
  target: string;
  captures: readonly unknown[];
}>;
const receipts = new WeakMap<Function, Receipt>();
// IntegerWorkerDescriptor is the compiler-private contract for checked
// integer callbacks: a synchronous bigint companion over all inputs,
// capture slot positions, residual arity and the callback origin. The
// emitter produces it only under exact integer-worker proof.
export type IntegerWorkerDescriptor = Readonly<{
  companion: (...operands: bigint[]) => bigint;
  positions: readonly number[];
  arity: number;
  origin: FailureOrigin;
}>;
// IntegerWorker is the registered residual runner: captures are already
// bound from saved values, so only element arguments remain.
export type IntegerWorker = Readonly<{
  run: (...residual: bigint[]) => bigint;
  arity: number;
}>;
const integerWorkers = new WeakMap<Function, IntegerWorker>();
function ownDataValue(holder: object, key: string): unknown {
  if (isHostProxy(holder)) return undefined;
  const descriptor = Object.getOwnPropertyDescriptor(holder, key);
  if (descriptor === undefined || !("value" in descriptor)) return undefined;
  return descriptor.value;
}
function integerWorkerDescriptor(value: unknown):
  | {
      companion: (...operands: bigint[]) => bigint;
      positions: readonly number[];
      arity: number;
      origin: FailureOrigin;
    }
  | undefined {
  if (value === null || typeof value !== "object" || isHostProxy(value)) return undefined;
  const keys = Object.keys(value);
  if (keys.length !== 4) return undefined;
  const companion = ownDataValue(value, "companion");
  const positions = ownDataValue(value, "positions");
  const arity = ownDataValue(value, "arity");
  const origin = ownDataValue(value, "origin");
  if (
    typeof companion !== "function" ||
    positions === null ||
    typeof positions !== "object" ||
    isHostProxy(positions) ||
    !Array.isArray(positions) ||
    Object.getPrototypeOf(positions) !== Array.prototype ||
    typeof arity !== "number" ||
    !Number.isInteger(arity) ||
    origin === null ||
    typeof origin !== "object" ||
    isHostProxy(origin)
  )
    return undefined;
  // Both lengths arrive through own data descriptors before any slot
  // traversal, so Array-shaped objects with accessor lengths never run.
  const length = ownDataValue(companion, "length");
  if (typeof length !== "number" || !Number.isInteger(length) || length < 0) return undefined;
  const count = ownDataValue(positions, "length");
  if (typeof count !== "number" || !Number.isInteger(count) || count < 0 || count > length)
    return undefined;
  const list = positions as readonly unknown[];
  const slots: number[] = [];
  for (let index = 0; index < count; index++) {
    const slot = ownDataValue(list, String(index));
    if (typeof slot !== "number" || !Number.isInteger(slot) || slot < 0) return undefined;
    if (slots.length > 0 && slot <= slots[slots.length - 1]) return undefined;
    slots.push(slot);
  }
  if (slots.length + arity !== length) return undefined;
  return {
    companion: companion as (...operands: bigint[]) => bigint,
    positions: slots,
    arity,
    origin: origin as FailureOrigin,
  };
}
function registerIntegerWorker(
  guarded: Function,
  captures: readonly unknown[],
  resourceIndices: readonly number[],
  descriptor: IntegerWorkerDescriptor,
): void {
  const parsed = integerWorkerDescriptor(descriptor);
  if (parsed === undefined) throw new TypeError("invalid integer worker descriptor");
  // Structurally sound descriptors with unusable captures decline
  // registration and keep slow behavior. Positions name original companion
  // parameters while captures arrive compact in positions order, so values
  // pair by ordinal; every original position must fit the full arity.
  if (resourceIndices.length !== 0) return;
  if (parsed.positions.length !== captures.length) return;
  for (const capture of captures) {
    if (typeof capture !== "bigint") return;
  }
  const saved = captures.slice() as bigint[];
  const total = parsed.positions.length + parsed.arity;
  if (parsed.positions.length > 0 && parsed.positions[parsed.positions.length - 1] >= total) return;
  const origin = parsed.origin;
  const companion = parsed.companion;
  const positions = parsed.positions;
  const run = (...residual: bigint[]): bigint => {
    const operands: bigint[] = [];
    let pending = 0,
      taken = 0;
    for (let slot = 0; slot < total; slot++) {
      operands[slot] = slot === positions[pending] ? saved[pending++] : residual[taken++];
    }
    const out = companion(...operands);
    if (typeof out !== "bigint")
      throw caught(new TypeError("integer worker returned non-bigint"), origin).value;
    return out;
  };
  integerWorkers.set(guarded, Object.freeze({ run, arity: parsed.arity }));
}
export function integerWorker(value: unknown): IntegerWorker | undefined {
  return typeof value === "function" ? integerWorkers.get(value) : undefined;
}
// MapLeafWorkerDescriptor is the compiler-private contract for checked map
// leaves: a synchronous Completion companion over (map, key), the quantized
// key kind and the callback origin. The emitter produces it only under exact
// map-leaf proof for closed two-input callables with no captures.
export type MapLeafWorkerDescriptor = Readonly<{
  companion: (...args: never[]) => Completion<unknown>;
  keyKind: string;
  origin: FailureOrigin;
}>;
// MapLeafWorker is the registered leaf runner over (map, key); the guarded
// fold invokes it inside completion authentication.
export type MapLeafWorker = Readonly<{
  run: (map: unknown, key: unknown) => Completion<unknown>;
  keyKind: string;
}>;
const mapLeafWorkers = new WeakMap<Function, MapLeafWorker>();
function mapLeafWorkerDescriptor(value: unknown):
  | {
      companion: (...args: never[]) => Completion<unknown>;
      keyKind: string;
      origin: FailureOrigin;
    }
  | undefined {
  if (value === null || typeof value !== "object" || isHostProxy(value)) return undefined;
  const keys = Object.keys(value);
  if (keys.length !== 3) return undefined;
  const companion = ownDataValue(value, "companion");
  const keyKind = ownDataValue(value, "keyKind");
  const origin = ownDataValue(value, "origin");
  if (
    typeof companion !== "function" ||
    typeof keyKind !== "string" ||
    (keyKind !== "int" && keyKind !== "bool" && keyKind !== "str") ||
    origin === null ||
    typeof origin !== "object" ||
    isHostProxy(origin)
  )
    return undefined;
  // The companion length arrives through its own data descriptor, so
  // functions or proxies with accessor lengths never attest arity.
  const length = ownDataValue(companion, "length");
  if (length !== 2) return undefined;
  return {
    companion: companion as (...args: never[]) => Completion<unknown>,
    keyKind,
    origin: origin as FailureOrigin,
  };
}
function registerMapLeafWorker(
  guarded: Function,
  captures: readonly unknown[],
  resourceIndices: readonly number[],
  descriptor: MapLeafWorkerDescriptor,
): void {
  const parsed = mapLeafWorkerDescriptor(descriptor);
  if (parsed === undefined) throw new TypeError("invalid map leaf descriptor");
  // Leaves are closed: any capture or resource capture declines
  // registration and keeps slow behavior.
  if (captures.length !== 0 || resourceIndices.length !== 0) return;
  const invoke = parsed.companion as (...args: unknown[]) => Completion<unknown>;
  const keyKind = parsed.keyKind;
  const run = (map: unknown, key: unknown) => invoke(map, key);
  mapLeafWorkers.set(guarded, Object.freeze({ run, keyKind }));
}
export function mapLeafWorker(value: unknown): MapLeafWorker | undefined {
  return typeof value === "function" ? mapLeafWorkers.get(value) : undefined;
}
// MapBatchWorkerDescriptor is the compiler-private contract for checked
// batch transitions: synchronous bigint value companions over (key) and
// (key, previous), the quantized key kind, the first call-site origin,
// and the exact canonical get-method function of the proven factory.
// The emitter produces it only under exact batch proof for closed
// two-input callables with no captures.
export type MapBatchWorkerDescriptor = Readonly<{
  absent: (key: never) => bigint;
  present: (key: never, previous: bigint) => bigint;
  keyKind: string;
  origin: FailureOrigin;
  factory: (...args: never[]) => unknown;
}>;
// MapBatchWorker is the registered batch transition: bigint-checked
// value companions plus the authenticated factory runner and origin.
export type MapBatchWorker = Readonly<{
  absent: (key: unknown) => bigint;
  present: (key: unknown, previous: bigint) => bigint;
  keyKind: string;
  batch: MapBatchRunner;
  origin: FailureOrigin;
}>;
const mapBatchWorkers = new WeakMap<Function, MapBatchWorker>();
function mapBatchWorkerDescriptor(value: unknown):
  | {
      absent: (key: never) => bigint;
      present: (key: never, previous: bigint) => bigint;
      keyKind: string;
      origin: FailureOrigin;
      factory: (...args: never[]) => unknown;
    }
  | undefined {
  if (value === null || typeof value !== "object" || isHostProxy(value)) return undefined;
  const keys = Object.keys(value);
  if (keys.length !== 5) return undefined;
  const absent = ownDataValue(value, "absent");
  const present = ownDataValue(value, "present");
  const keyKind = ownDataValue(value, "keyKind");
  const origin = ownDataValue(value, "origin");
  const factory = ownDataValue(value, "factory");
  if (
    typeof absent !== "function" ||
    typeof present !== "function" ||
    typeof keyKind !== "string" ||
    (keyKind !== "int" && keyKind !== "bool" && keyKind !== "str") ||
    origin === null ||
    typeof origin !== "object" ||
    isHostProxy(origin) ||
    typeof factory !== "function"
  )
    return undefined;
  // Companion lengths arrive through their own data descriptors, so
  // functions or proxies with accessor lengths never attest arity.
  const absentLength = ownDataValue(absent, "length");
  if (absentLength !== 1) return undefined;
  const presentLength = ownDataValue(present, "length");
  if (presentLength !== 2) return undefined;
  return {
    absent: absent as (key: never) => bigint,
    present: present as (key: never, previous: bigint) => bigint,
    keyKind,
    origin: origin as FailureOrigin,
    factory: factory as (...args: never[]) => unknown,
  };
}
function registerMapBatchWorker(
  guarded: Function,
  captures: readonly unknown[],
  resourceIndices: readonly number[],
  descriptor: MapBatchWorkerDescriptor,
): void {
  const parsed = mapBatchWorkerDescriptor(descriptor);
  if (parsed === undefined) throw new TypeError("invalid map batch descriptor");
  // Batches are closed and factory-bound: any capture, resource capture
  // or unknown factory identity declines registration and keeps
  // leaf/generic behavior.
  if (captures.length !== 0 || resourceIndices.length !== 0) return;
  const batch = mapBatchCapability(parsed.factory);
  if (batch === undefined) return;
  const origin = parsed.origin;
  const absent = parsed.absent as (key: unknown) => bigint;
  const present = parsed.present as (key: unknown, previous: bigint) => bigint;
  const checkedAbsent = (key: unknown): bigint => {
    const out = absent(key);
    if (typeof out !== "bigint")
      throw caught(new TypeError("map batch worker returned non-bigint"), origin).value;
    return out;
  };
  const checkedPresent = (key: unknown, previous: bigint): bigint => {
    const out = present(key, previous);
    if (typeof out !== "bigint")
      throw caught(new TypeError("map batch worker returned non-bigint"), origin).value;
    return out;
  };
  mapBatchWorkers.set(
    guarded,
    Object.freeze({
      absent: checkedAbsent,
      present: checkedPresent,
      keyKind: parsed.keyKind,
      batch,
      origin,
    }),
  );
}
export function mapBatchWorker(value: unknown): MapBatchWorker | undefined {
  return typeof value === "function" ? mapBatchWorkers.get(value) : undefined;
}
export function ownCallable<T extends Function>(
  site: string,
  target: string,
  captures: readonly unknown[],
  value: T,
  resourceIndices: readonly number[] = captures.map((_, i) => i),
  context?: AssertionContext,
  worker?: IntegerWorkerDescriptor,
  leaf?: MapLeafWorkerDescriptor,
  batch?: MapBatchWorkerDescriptor,
): T {
  if (!site || !target || typeof value !== "function" || receipts.has(value))
    throw new TypeError("invalid callable construction");
  if (
    new Set(resourceIndices).size !== resourceIndices.length ||
    resourceIndices.some((i) => !Number.isInteger(i) || i < 0 || i >= captures.length)
  )
    throw new TypeError("invalid callable resource capture");
  const guarded = registerCallableCaptures(
    value,
    resourceIndices.map((i) => captures[i]),
  );
  const identity =
    context === undefined
      ? initializationCallableIdentity(site, captures)
      : callableIdentity(contextIdentity(context), site, captures);
  receipts.set(
    guarded,
    Object.freeze({ identity, site, target, captures: Object.freeze([...captures]) }),
  );
  if (worker !== undefined) registerIntegerWorker(guarded, captures, resourceIndices, worker);
  if (leaf !== undefined) registerMapLeafWorker(guarded, captures, resourceIndices, leaf);
  if (batch !== undefined) registerMapBatchWorker(guarded, captures, resourceIndices, batch);
  return Object.freeze(guarded);
}
export function callableReceipt(value: unknown): Receipt | undefined {
  return typeof value === "function" ? receipts.get(value) : undefined;
}

export function callableInstance(value: unknown): CallableIdentity | undefined {
  return callableReceipt(value)?.identity;
}
// callableEqual compares owned callables by target and captures, not by
// construction site: the same `callable name` expression evaluated for a
// call and for its fixture row names one callable. Foreign functions and
// differing targets or captures never compare equal. The element equality
// arrives as a parameter so this module never imports the assert runner.
export function callableEqual(
  left: unknown,
  right: unknown,
  equal: (a: unknown, b: unknown) => boolean,
): boolean {
  if (typeof left !== "function" || typeof right !== "function") return false;
  const a = receipts.get(left),
    b = receipts.get(right);
  if (!a || !b || a.target !== b.target || a.captures.length !== b.captures.length) return false;
  return a.captures.every((capture, i) => equal(capture, b.captures[i]));
}
