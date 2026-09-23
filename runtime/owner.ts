// Private lifecycle enforcement over native async execution. Payloads always
// remain protected Completion values; deadline waits never cancel owned work.
import { AsyncLocalStorage } from "node:async_hooks";
import { types as nativeTypes } from "node:util";
import { invoke, checkedCompletion, success, failure, type Completion } from "./completion.ts";
import { dataArray, dataKeys, dataProperty, recordIdentity, opaqueContents } from "./data.ts";
import {
  cleanupFailure,
  resourceStateFailure,
  standardFailureDiagnostics,
  type FailureOrigin,
  type StandardFailure,
} from "./failure.ts";

const origin: FailureOrigin = Object.freeze({
  source: "can:owner",
  start: 0,
  end: 0,
  invocation: Object.freeze([]),
});
type State = "open" | "closing" | "closed";
export type OwnerDiagnostic = Readonly<{
  kind: "can.runtime-diagnostic";
  phase: "late" | "cleanup";
  occurrence: string;
  category: string;
  message: string;
}>;
type Reporter = (diagnostic: OwnerDiagnostic) => void | Promise<void>;
declare const resourceBrand: unique symbol;
declare const scopeBrand: unique symbol;
export type Resource = Readonly<{ [resourceBrand]: true }>;
export type Scope = Readonly<{ [scopeBrand]: true }>;
type ScopeState = {
  exitReleases: (() => void)[];
  root: Root;
  parent?: ScopeState;
  state: State;
  base: object;
};
type ResourceState = {
  creator?: Task;
  initialRelease?: () => void;
  shutdownMs?: number;
  timer?: ReturnType<typeof setTimeout>;
  autoPending?: boolean;
  cleanup?: StandardFailure;
  scopeManaged: boolean;
  token: Resource;
  id: bigint;
  kind: string;
  scope: ScopeState;
  state: State;
  native: unknown;
  leases: Map<object, number>;
  close: () => Completion<void> | Promise<Completion<void>>;
  closing?: Promise<Completion<void>>;
  idempotent: boolean;
};
type Task = {
  scope: ScopeState;
  dynamic: (() => void)[];
  captures?: readonly unknown[];
  promise: Promise<Completion>;
  completion?: Completion;
  release: () => void;
};
type Group = {
  root: Root;
  scope: ScopeState;
  key: object;
  tasks: Task[];
  sealed: boolean;
  selected: Set<number>;
  pending: number;
};
type Root = {
  scope: ScopeState;
  groups: Set<Group>;
  callbacks: Set<Task>;
  resources: ResourceState[];
  waiters: Set<() => void>;
  reports: Set<Promise<void>>;
  reported: WeakSet<StandardFailure>;
  failed: boolean;
  report: Reporter;
};
type Execution = { root: Root; scope: ScopeState; owner?: object; task?: Task };
const context = new AsyncLocalStorage<Execution>();
const resources = new WeakMap<object, ResourceState>();
const scopes = new WeakMap<object, ScopeState>();
const captures = new WeakMap<Function, readonly unknown[]>();
let nextResource = 0n;
const objectLike = (value: unknown): value is object =>
  value !== null && (typeof value === "object" || typeof value === "function");
function invalid(): never {
  throw resourceStateFailure(undefined, origin);
}
function execution(): Execution {
  const current = context.getStore();
  if (!current || current.scope.state === "closed") invalid();
  return current;
}
function inside(scope: ScopeState, ancestor: ScopeState): boolean {
  for (let at: ScopeState | undefined = scope; at; at = at.parent) if (at === ancestor) return true;
  return false;
}
function signal(root: Root): void {
  const waiters = [...root.waiters];
  root.waiters.clear();
  for (const resolve of waiters) resolve();
}
function changed(root: Root): Promise<void> {
  return new Promise((resolve) => root.waiters.add(resolve));
}
function emit(root: Root, failure: StandardFailure, phase: "late" | "cleanup"): void {
  if (root.reported.has(failure)) return;
  root.reported.add(failure);
  const details = standardFailureDiagnostics(failure);
  const diagnostic = Object.freeze({
    kind: "can.runtime-diagnostic" as const,
    phase,
    occurrence: String(details.occurrenceID),
    category: details.kind,
    message: details.message,
  });
  // Both rejection branches are installed immediately. Diagnostic delivery alone
  // cannot invalidate an already selected completion or successful process status.
  const pending = Promise.resolve()
    .then(() => root.report(diagnostic))
    .then(
      () => {},
      () => {},
    );
  root.reports.add(pending);
  void pending.then(() => {
    root.reports.delete(pending);
    signal(root);
  });
}
function cleanup(root: Root, cause?: unknown): StandardFailure {
  root.failed = true;
  const failed = cleanupFailure(cause, origin);
  emit(root, failed, "cleanup");
  return failed;
}
function resource(value: unknown, kind?: string): ResourceState {
  if (!objectLike(value)) invalid();
  const state = resources.get(value);
  if (!state || (kind !== undefined && state.kind !== kind)) invalid();
  return state;
}
function validateScope(state: ResourceState, current: Execution): void {
  if (
    state.scope.root !== current.root ||
    !inside(current.scope, state.scope) ||
    state.scope.state === "closed"
  )
    invalid();
}
function acquire(state: ResourceState, current: Execution): () => void {
  validateScope(state, current);
  const owner = current.owner ?? current.scope.base;
  if (
    state.state === "closed" ||
    (state.state === "closing" && !state.leases.has(owner)) ||
    (state.scope.state === "closing" && !state.leases.has(owner))
  )
    invalid();
  state.leases.set(owner, (state.leases.get(owner) ?? 0) + 1);
  let released = false;
  return () => {
    if (released) invalid();
    released = true;
    const count = state.leases.get(owner)!;
    if (count === 1) state.leases.delete(owner);
    else state.leases.set(owner, count - 1);
    signal(current.root);
  };
}
function capturedResources(values: readonly unknown[]): ResourceState[] {
  const found = new Set<ResourceState>(),
    seen = new Set<object>();
  function visit(value: unknown): void {
    if (!objectLike(value) || seen.has(value)) return;
    seen.add(value);
    const retained = resources.get(value);
    if (retained) {
      found.add(retained);
      return;
    }
    const children = opaqueContents(value);
    if (children) {
      for (const item of children) visit(item);
      return;
    }
    if (typeof value === "function") {
      for (const item of captures.get(value) ?? []) visit(item);
      return;
    }
    if (nativeTypes.isProxy(value)) return;
    if (Array.isArray(value)) {
      for (const item of dataArray(value)) visit(item);
      return;
    }
    if (recordIdentity(value))
      for (const key of dataKeys(value))
        if (typeof key === "string") visit(dataProperty(value, key));
  }
  for (const value of values) visit(value);
  return [...found];
}
function retain(values: readonly unknown[], current: Execution): () => void {
  const releases: (() => void)[] = [];
  try {
    for (const state of capturedResources(values)) releases.push(acquire(state, current));
  } catch (cause) {
    for (const release of releases.reverse()) release();
    throw cause;
  }
  return () => {
    for (const release of releases.reverse()) release();
  };
}

export function registerResource(
  kind: string,
  native: unknown,
  close: () => Completion<void> | Promise<Completion<void>>,
  options: Readonly<{
    idempotent?: boolean;
    scopeManaged?: boolean;
    shutdownMilliseconds?: number;
  }> = {},
): Resource {
  const current = execution();
  if (!kind || current.scope.state !== "open") invalid();
  if (
    options.shutdownMilliseconds !== undefined &&
    (!Number.isSafeInteger(options.shutdownMilliseconds) || options.shutdownMilliseconds < 0)
  )
    throw new TypeError("invalid registered shutdown deadline");
  const token = Object.freeze(Object.create(null)) as Resource;
  const state: ResourceState = {
    token,
    id: ++nextResource,
    kind,
    scope: current.scope,
    state: "open",
    native,
    close,
    idempotent: options.idempotent ?? false,
    scopeManaged: options.scopeManaged ?? false,
    shutdownMs: options.shutdownMilliseconds,
    leases: new Map(),
  };
  resources.set(token, state);
  current.root.resources.push(state);
  if (current.task) {
    const release = acquire(state, current);
    let released = false;
    const once = () => {
      if (!released) {
        released = true;
        release();
      }
    };
    state.creator = current.task;
    state.initialRelease = once;
    current.task.dynamic.push(once);
    // An enclosing participant must not keep a child transaction scope alive
    // after its own callback returns. Its initial lease ends at that boundary.
    if (current.scope !== current.task.scope) current.scope.exitReleases.push(once);
  }
  signal(current.root);
  return token;
}
export function resourceStatus(value: unknown) {
  const state = resource(value);
  return Object.freeze({
    id: state.id,
    kind: state.kind,
    state: state.state,
    leases: [...state.leases.values()].reduce((a, b) => a + b, 0),
  });
}
export async function useResource<T>(
  value: unknown,
  kind: string,
  operation: (native: unknown) => Completion<T> | Promise<Completion<T>>,
): Promise<Completion<T>> {
  const current = execution(),
    state = resource(value, kind),
    release = acquire(state, current);
  try {
    return await invoke(() => operation(state.native), origin);
  } finally {
    release();
  }
}
function beginClose(state: ResourceState): Promise<Completion<void>> {
  state.state = "closing";
  state.autoPending = false;
  const result = Promise.resolve().then(async () => {
    while (state.leases.size) await changed(state.scope.root);
    const completion = await invoke(state.close, origin);
    if (state.timer !== undefined) {
      clearTimeout(state.timer);
      state.timer = undefined;
    }
    if (completion.kind === "ok") {
      state.state = "closed";
      state.native = undefined;
      state.creator = undefined;
      state.initialRelease = undefined;
      state.close = () => success(undefined);
    } else state.cleanup = cleanup(state.scope.root, completion.value);
    signal(state.scope.root);
    return completion;
  });
  state.closing = result;
  return result;
}
export async function closeResource(
  value: unknown,
  kind: string,
  deadline?: Readonly<{ milliseconds: number; failure: () => Completion<void> }>,
): Promise<Completion<void>> {
  const current = execution(),
    state = resource(value, kind);
  validateScope(state, current);
  if (state.state !== "open" && !state.idempotent) invalid();
  if (deadline && (!Number.isSafeInteger(deadline.milliseconds) || deadline.milliseconds < 0))
    throw new TypeError("invalid close deadline");
  if (state.creator === current.task) state.initialRelease?.();
  const actual = state.closing ?? beginClose(state);
  if (!deadline) return actual;
  let timer: ReturnType<typeof setTimeout> | undefined;
  const timeout = new Promise<Completion<void>>((resolve) => {
    timer = setTimeout(() => {
      state.cleanup = cleanup(current.root);
      // Normalize even a malformed private deadline adapter into a protected result.
      void invoke<void>(() => {
        const failed = checkedCompletion(deadline.failure());
        if (failed.kind === "ok") throw new TypeError("close deadline must fail");
        return failed;
      }, origin).then(resolve);
    }, deadline.milliseconds);
  });
  try {
    return await Promise.race([actual, timeout]);
  } finally {
    if (timer !== undefined) clearTimeout(timer);
  }
}

export type Participant = Readonly<{
  run: () => Completion | Promise<Completion>;
  captures: readonly unknown[];
}>;
export type OwnedGroup = Readonly<{
  promises: readonly Promise<Completion>[];
  publish: (selected: readonly number[]) => void;
}>;
export function launchOwned(participants: readonly Participant[]): OwnedGroup {
  return launch(participants, false);
}
// Maintained native subwork belongs to an already active owner, including while
// its scope drains. This does not admit a new coordination owner into closing.
export function launchNative(participant: Participant): OwnedGroup {
  return launch([participant], true);
}
function launch(participants: readonly Participant[], native: boolean): OwnedGroup {
  const current = execution();
  if (
    current.scope.state !== "open" &&
    !(native && current.task && current.task.completion === undefined)
  )
    invalid();
  const group: Group = {
    root: current.root,
    scope: current.scope,
    key: native && current.owner ? current.owner : {},
    tasks: [],
    sealed: false,
    selected: new Set(),
    pending: participants.length,
  };
  const child: Execution = { ...current, owner: group.key };
  const retained: (() => void)[] = [];
  // Every lease/capture is prepared before the first participant can run.
  try {
    for (const participant of participants) retained.push(retain(participant.captures, child));
  } catch (cause) {
    for (const release of retained.reverse()) release();
    throw cause;
  }
  current.root.groups.add(group);
  function observe(index: number): void {
    const result = group.tasks[index].completion;
    if (result?.kind === "standard" && !group.selected.has(index))
      emit(group.root, result.value, "late");
  }
  function finish(index?: number): void {
    if (!group.sealed) return;
    // Publication observes completions already available. Each later settlement
    // observes only its own outcome, so draining a batch takes linear work.
    if (index === undefined) {
      for (let i = 0; i < group.tasks.length; i++) observe(i);
    } else observe(index);
    if (group.pending === 0) {
      group.root.groups.delete(group);
      signal(group.root);
    }
  }
  for (let i = 0; i < participants.length; i++) {
    const task: Task = {
      scope: group.scope,
      dynamic: [],
      captures: Object.freeze([...participants[i].captures]),
      promise: undefined!,
      release: retained[i],
    };
    group.tasks.push(task);
    task.promise = context
      .run({ ...child, task }, () => invoke(participants[i].run, origin))
      .then((completion) => {
        task.completion = completion;
        task.release();
        for (const release of task.dynamic) release();
        task.dynamic = [];
        task.captures = undefined;
        group.pending--;
        finish(i);
        return completion;
      });
  }
  return Object.freeze({
    promises: Object.freeze(group.tasks.map((task) => task.promise)),
    publish(selected: readonly number[]) {
      if (
        group.sealed ||
        selected.some((i) => !Number.isSafeInteger(i) || i < 0 || i >= group.tasks.length)
      )
        invalid();
      group.selected = new Set(selected);
      group.sealed = true;
      finish();
    },
  });
}

export function registerCallableCaptures<T extends Function>(
  value: T,
  values: readonly unknown[],
): T {
  // Ordinary callable construction/invocation adds no ownership rule. Participant
  // launch consumes this evidence; catalogue operations validate actual use.
  captures.set(value, Object.freeze([...values]));
  return value;
}
export async function withScope<T>(
  body: (scope: Scope) => Completion<T> | Promise<Completion<T>>,
): Promise<Completion<T>> {
  const current = execution();
  if (current.scope.state !== "open") invalid();
  const scope: ScopeState = {
    exitReleases: [],
    root: current.root,
    parent: current.scope,
    state: "open",
    base: {},
  };
  const token = Object.freeze(Object.create(null)) as Scope;
  scopes.set(token, scope);
  return context.run({ ...current, scope }, async () => {
    const result = await invoke(() => body(token), origin);
    for (const release of scope.exitReleases) release();
    scope.exitReleases = [];
    scope.state = "closing";
    const failed = await drain(current.root, scope);
    scope.state = "closed";
    return failed && result.kind === "ok" ? failure(failed) : result;
  });
}
export function guardCallback<T extends Function>(token: Scope, callback: T): T {
  const owner = scopes.get(token);
  if (!owner) invalid();
  const guarded = async (...args: unknown[]) => {
    const ambient = context.getStore();
    const active =
      ambient?.root === owner.root &&
      inside(ambient.scope, owner) &&
      ambient.task !== undefined &&
      ambient.task.completion === undefined;
    if (owner.state === "closed" || (owner.state === "closing" && !active)) invalid();
    // Native event dispatch need not inherit the registration's async context.
    // Bind the operation scope explicitly and retain callback work until settled.
    const scope = active ? ambient.scope : owner;
    const task: Task = {
      scope,
      dynamic: [],
      captures: [callback, ...args],
      promise: undefined!,
      release: () => {},
    };
    owner.root.callbacks.add(task);
    task.promise = context
      .run({ root: owner.root, scope, owner: active ? ambient.owner : {}, task }, () =>
        invoke(() => callback(...args), origin),
      )
      .then((completion) => {
        task.completion = completion;
        for (const release of task.dynamic) release();
        task.dynamic = [];
        task.captures = undefined;
        owner.root.callbacks.delete(task);
        signal(owner.root);
        return completion;
      });
    return task.promise;
  };
  captures.set(guarded, Object.freeze([callback]));
  return guarded as unknown as T;
}
async function drain(root: Root, scope: ScopeState): Promise<StandardFailure | undefined> {
  let first: StandardFailure | undefined;
  for (;;) {
    for (const resource of root.resources)
      if (
        inside(resource.scope, scope) &&
        resource.state !== "closed" &&
        resource.shutdownMs !== undefined &&
        resource.timer === undefined &&
        !resource.cleanup
      ) {
        resource.timer = setTimeout(() => {
          resource.timer = undefined;
          if (resource.state === "closed") return;
          if (resource.state === "open") {
            resource.state = "closing";
            resource.autoPending = true;
          }
          resource.cleanup = cleanup(root);
          signal(root);
        }, resource.shutdownMs);
      }
    if (
      ![...root.groups].some((group) => inside(group.scope, scope)) &&
      ![...root.callbacks].some((task) => inside(task.scope, scope))
    )
      break;
    await changed(root);
  }
  scope.state = "closing";
  // Scoped resources may still have direct operation leases; beginClose retains
  // the native operation and waits. Never force close or discard a live lease.
  for (const resource of [...root.resources].reverse())
    if (inside(resource.scope, scope)) {
      if (resource.state === "open" || resource.autoPending) {
        if (!resource.scopeManaged && !resource.cleanup) resource.cleanup = cleanup(root);
        await beginClose(resource);
      } else if (resource.closing) await resource.closing;
      first ??= resource.cleanup;
    }
  while (root.reports.size) await Promise.all(root.reports);
  return first;
}
export async function runOwnedRoot<T>(
  body: () => Completion<T> | Promise<Completion<T>>,
  report: Reporter = () => {},
): Promise<Readonly<{ completion: Completion<T>; cleanupFailed: boolean }>> {
  const root = {} as Root;
  Object.assign(root, {
    groups: new Set(),
    callbacks: new Set(),
    resources: [],
    waiters: new Set(),
    reports: new Set(),
    reported: new WeakSet(),
    failed: false,
    report,
  });
  root.scope = { exitReleases: [], root, state: "open", base: {} };
  return context.run({ root, scope: root.scope }, async () => {
    const completion = await invoke(body, origin);
    // Existing owners keep their execution context while this waits. No global
    // current-root mutation can redirect a late task into a later assertion.
    await drain(root, root.scope);
    root.scope.state = "closed";
    return Object.freeze({ completion, cleanupFailed: root.failed });
  });
}
