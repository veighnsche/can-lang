// Shared private byte representation. Native boundaries receive copies, never
// the owned backing array; application code can obtain only the opaque token.
import { resourceStateFailure, type FailureOrigin } from "./failure.ts";

declare const bytesBrand: unique symbol;
export type Bytes = Readonly<{readonly [bytesBrand]: true}>;
const storage = new WeakMap<object, Uint8Array>();
export function ownBytes(fresh: Uint8Array): Bytes {
  const token = Object.freeze(Object.create(null));
  // Copy even at this maintained boundary so ownership cannot depend on a
  // caller accidentally retaining a native alias.
  storage.set(token, fresh.slice());
  return token;
}
export function isBytes(value: unknown): value is Bytes {
  return value !== null && (typeof value === "object" || typeof value === "function") && storage.has(value);
}
export function copyBytes(value: unknown, origin: FailureOrigin): Uint8Array {
  if (!isBytes(value)) throw resourceStateFailure(undefined, origin);
  return storage.get(value)!.slice();
}
