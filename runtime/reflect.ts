// Canonical host-object predicates. Bun resolves this module literally; the
// browser profile overlays runtime/browser/reflect.ts at bundle time. The
// predicates below are trap-free engine internals; the portable alternate
// documents its bounded contained divergence.
import { types as nativeTypes } from "node:util";

export function isHostProxy(value: unknown): boolean {
  return nativeTypes.isProxy(value);
}

export function isHostNativeError(value: unknown): boolean {
  return nativeTypes.isNativeError(value);
}
