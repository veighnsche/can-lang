// Browser-profile ownership: the explicit-only surface over the canonical
// core. Generated browser calls thread one OwnerContext token across every
// async edge; no ambient store exists on any browser path. The ambient entry
// points below exist solely so shared modules that never invoke them (such
// as stream lifecycle under Bytes-based codecs) still instantiate; every
// ambient call fails closed instead of substituting synchronous context.
import type { Completion } from "../completion.ts";
import type {
  CloseDeadline,
  OwnedGroup,
  OwnerDiagnostic,
  Participant,
  RegisterOptions,
  Resource,
  Scope,
} from "../owner-core.ts";

export type {
  CloseDeadline,
  ExplicitParticipant,
  OwnedGroup,
  OwnerContext,
  OwnerDiagnostic,
  Participant,
  RegisterOptions,
  Resource,
  Scope,
} from "../owner-core.ts";
export {
  closeResourceWithContext,
  guardCallbackWithContext,
  launchNativeWithContext,
  launchOwnedWithContext,
  registerCallableCaptures,
  registerResourceWithContext,
  resourceStatus,
  runExplicitRoot,
  useResourceWithContext,
  withScopeWithContext,
} from "../owner-core.ts";

function unavailable(): never {
  throw new TypeError("ambient ownership is unavailable in the browser profile");
}

export function registerResource(
  _kind: string,
  _native: unknown,
  _close: () => Completion<void> | Promise<Completion<void>>,
  _options?: RegisterOptions,
): Resource {
  return unavailable();
}

export async function useResource<T>(
  _value: unknown,
  _kind: string,
  _operation: (native: unknown) => Completion<T> | Promise<Completion<T>>,
): Promise<Completion<T>> {
  return unavailable();
}

export async function closeResource(
  _value: unknown,
  _kind: string,
  _deadline?: CloseDeadline,
): Promise<Completion<void>> {
  return unavailable();
}

export function launchOwned(_participants: readonly Participant[]): OwnedGroup {
  return unavailable();
}

export function launchNative(_participant: Participant): OwnedGroup {
  return unavailable();
}

export async function withScope<T>(
  _body: (scope: Scope) => Completion<T> | Promise<Completion<T>>,
): Promise<Completion<T>> {
  return unavailable();
}

export function guardCallback<T extends Function>(token: Scope, callback: T): T {
  void token;
  void callback;
  return unavailable();
}

export async function runOwnedRoot<T>(
  _body: () => Completion<T> | Promise<Completion<T>>,
  _report?: (diagnostic: OwnerDiagnostic) => void | Promise<void>,
): Promise<Readonly<{ completion: Completion<T>; cleanupFailed: boolean }>> {
  return unavailable();
}
