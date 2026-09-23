import { contextIdentity, type AssertionContext } from "./assert/context.ts";
import {
  callableIdentity,
  initializationCallableIdentity,
  type CallableIdentity,
} from "./assert/identity.ts";
import { registerCallableCaptures } from "./owner.ts";
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
export function ownCallable<T extends Function>(
  site: string,
  target: string,
  captures: readonly unknown[],
  value: T,
  resourceIndices: readonly number[] = captures.map((_, i) => i),
  context?: AssertionContext,
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
