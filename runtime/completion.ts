export type { AssertionContext } from "./assert/context.ts";
// Every Promise-visible Can payload stays behind a compiler-private box. The
// brand sets admit only constructors below; property inspection never touches a
// forged value or application-controlled proxy/getter.
import { isDomainFailure, domainFailureDiagnostics, type DomainFailure } from "./domain.ts";
import {
  captureStandard,
  isStandardFailure,
  type StandardFailure,
  type FailureOrigin,
} from "./failure.ts";

declare const completionBrand: unique symbol;
declare const elementBrand: unique symbol;
export type Completion<T = unknown> = Readonly<
  (
    | { kind: "ok"; value: T }
    | { kind: "domain"; value: DomainFailure }
    | { kind: "standard"; value: StandardFailure }
  ) & { readonly [completionBrand]: true }
>;
export type Element<T> = Readonly<{ value: T; readonly [elementBrand]: true }>;
const completions = new WeakSet<object>();
const elements = new WeakSet<object>();
const objectLike = (v: unknown): v is object =>
  v !== null && (typeof v === "object" || typeof v === "function");
function box(kind: "ok" | "domain" | "standard", value: unknown): Completion {
  const result = Object.create(null);
  Object.defineProperties(result, {
    kind: { value: kind, enumerable: true },
    value: { value, enumerable: true },
  });
  Object.freeze(result);
  completions.add(result);
  return result;
}
export function success<T>(value: T): Completion<T> {
  return box("ok", value) as Completion<T>;
}
export function failure(value: DomainFailure | StandardFailure): Completion<never> {
  if (isDomainFailure(value)) return box("domain", value) as Completion<never>;
  if (isStandardFailure(value)) return box("standard", value) as Completion<never>;
  throw new TypeError("invalid completion failure");
}
export function isCompletion(value: unknown): value is Completion {
  return objectLike(value) && completions.has(value);
}
export function checkedCompletion<T>(value: Completion<T>): Completion<T> {
  if (!isCompletion(value)) throw new TypeError("invalid completion carrier");
  return value;
}
// Extraction is synchronous. Never return this value from an async function or
// Promise callback: rebox it first, even if its apparent type is a plain record.
export function value<T>(completion: Completion<T>): T {
  checkedCompletion(completion);
  if (completion.kind !== "ok") throw completion.value;
  return completion.value;
}
export function errorType(completion: Completion): string {
  checkedCompletion(completion);
  if (completion.kind !== "domain") throw new TypeError("completion is not a domain failure");
  return domainFailureDiagnostics(completion.value).typeIdentity;
}
export function errorPayload(completion: Completion): unknown {
  checkedCompletion(completion);
  if (completion.kind !== "domain") throw new TypeError("completion is not a domain failure");
  return domainFailureDiagnostics(completion.value).payload;
}
export function caught(cause: unknown, origin: FailureOrigin): Completion<never> {
  return failure(isDomainFailure(cause) ? cause : captureStandard(cause, origin));
}
export function element<T>(value: T): Element<T> {
  const result = Object.create(null);
  Object.defineProperty(result, "value", { value, enumerable: true });
  Object.freeze(result);
  elements.add(result);
  return result;
}
export function elementValue<T>(box: Element<T>): T {
  if (!objectLike(box) || !elements.has(box)) throw new TypeError("invalid element carrier");
  return box.value;
}
// Generated thunks return a carrier synchronously or a native promise of one.
// Reject an unboxed immediate result before await could assimilate its `then`.
function locatedCompletion<T>(value: Completion<T>, origin: FailureOrigin): Completion<T> {
  const result = checkedCompletion(value);
  if (result.kind === "standard") captureStandard(result.value, origin);
  return result;
}
export async function invoke<T>(
  call: () => Completion<T> | Promise<Completion<T>>,
  origin: FailureOrigin,
): Promise<Completion<T>> {
  try {
    const pending = call();
    if (isCompletion(pending)) return locatedCompletion(pending as Completion<T>, origin);
    if (!(pending instanceof Promise)) throw new TypeError("call returned an unboxed result");
    return locatedCompletion(await pending, origin);
  } catch (cause) {
    return caught(cause, origin);
  }
}
// Native all/allSettled/any/race consume these adapter promises in I19. Only a
// success fulfills; failure rejects with its original private completion box.
export function toPromise<T>(pending: Promise<Completion<T>>): Promise<Completion<T>> {
  return pending.then((completion) => {
    checkedCompletion(completion);
    if (completion.kind !== "ok") throw completion;
    return completion;
  });
}
export async function fromPromise<T>(
  pending: Promise<Completion<T>>,
  origin: FailureOrigin,
): Promise<Completion<T>> {
  try {
    return locatedCompletion(await pending, origin);
  } catch (cause) {
    if (isCompletion(cause) && cause.kind !== "ok") return cause;
    return caught(cause, origin);
  }
}
export type Handlers<T, R> = Readonly<{
  ok: (value: T) => Completion<R> | Promise<Completion<R>>;
  domain: ReadonlyMap<string, (value: unknown) => Completion<R> | Promise<Completion<R>>>;
  standard?: (value: StandardFailure) => Completion<R> | Promise<Completion<R>>;
}>;
// Dispatch is one-shot. invoke contains the selected handler, never the dispatch
// itself; a handler-produced failure is returned directly without selecting again.
export function dispatch<T, R>(
  completion: Completion<T>,
  handlers: Handlers<T, R>,
  origin: FailureOrigin,
): Promise<Completion<R>> {
  checkedCompletion(completion);
  if (completion.kind === "ok") return invoke(() => handlers.ok(completion.value), origin);
  if (completion.kind === "standard") {
    return handlers.standard
      ? invoke(() => handlers.standard!(completion.value), origin)
      : Promise.resolve(completion);
  }
  const handler = handlers.domain.get(errorType(completion));
  if (!handler) throw new TypeError("missing checked domain arm");
  return invoke(() => handler(errorPayload(completion)), origin);
}
