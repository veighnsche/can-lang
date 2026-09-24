// Browser-profile callables. Native closures own execution and captures; the
// private receipt preserves creation-site/target/capture evidence. Assertion
// context stays outside the shipped profile, so identities are deterministic
// per-site occurrence counters instead of assertion digests.
import { registerCallableCaptures } from "../owner-core.ts";

declare const browserCallableBrand: unique symbol;
export type BrowserCallableIdentity = Readonly<{ readonly [browserCallableBrand]: true }>;
type Receipt = Readonly<{
  identity: BrowserCallableIdentity;
  site: string;
  target: string;
  captures: readonly unknown[];
}>;
const receipts = new WeakMap<Function, Receipt>();
const occurrences = new Map<string, number>();

export function ownCallable<T extends Function>(
  site: string,
  target: string,
  captures: readonly unknown[],
  value: T,
  resourceIndices: readonly number[] = captures.map((_, i) => i),
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
  const visit = occurrences.get(site) ?? 0;
  if (!Number.isSafeInteger(visit + 1)) throw new TypeError("callable occurrence overflow");
  occurrences.set(site, visit + 1);
  const identity = Object.freeze(Object.create(null)) as BrowserCallableIdentity;
  receipts.set(
    guarded,
    Object.freeze({ identity, site, target, captures: Object.freeze([...captures]) }),
  );
  return Object.freeze(guarded);
}

export function callableReceipt(value: unknown): Receipt | undefined {
  return typeof value === "function" ? receipts.get(value) : undefined;
}

export function callableInstance(value: unknown): BrowserCallableIdentity | undefined {
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
