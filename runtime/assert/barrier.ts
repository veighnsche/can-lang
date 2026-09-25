import { checkedCompletion, type Completion } from "../completion.ts";
import { compareInvocations, invocationPath, type InvocationIdentity } from "./lineage.ts";

declare const barrierBrand: unique symbol;
declare const frameBrand: unique symbol;
export type Barrier = Readonly<{ [barrierBrand]: true }>;
export type Frame = Readonly<{ [frameBrand]: true }>;
type Phase = "reserved" | "running" | "waiting" | "fixture" | "done";
type FrameState = { barrier: State; identity: InvocationIdentity; phase: Phase };
type Request = {
  frame: FrameState;
  sequence: bigint;
  deliver: () => Completion | Promise<Completion>;
  resolve: (value: Completion) => void;
  reject: (cause: unknown) => void;
};
type State = {
  root: InvocationIdentity;
  frames: Set<FrameState>;
  identities: WeakSet<object>;
  active: number;
  pending: Set<Request>;
  incoming: Request[];
  heap: Request[];
  sequence: bigint;
  queued: boolean;
  compare: (a: InvocationIdentity, b: InvocationIdentity) => number;
};
const barriers = new WeakMap<object, State>(),
  frames = new WeakMap<object, FrameState>();
const token = <T>(): T => Object.freeze(Object.create(null)) as T;
function barrier(value: Barrier): State {
  const found = value !== null && typeof value === "object" ? barriers.get(value) : undefined;
  if (!found) throw new TypeError("invalid assertion barrier");
  return found;
}
function frame(value: Frame): FrameState {
  const found = value !== null && typeof value === "object" ? frames.get(value) : undefined;
  if (!found) throw new TypeError("invalid assertion frame");
  return found;
}
function blocking(phase: Phase): number {
  return phase === "running" || phase === "reserved" ? 1 : 0;
}
function phase(entry: FrameState, next: Phase): void {
  entry.barrier.active += blocking(next) - blocking(entry.phase);
  entry.phase = next;
}
function compare(state: State, a: Request, b: Request): number {
  const order = state.compare(a.frame.identity, b.frame.identity);
  if (!Number.isFinite(order)) throw new TypeError("invalid conformance schedule order");
  return order || (a.sequence < b.sequence ? -1 : a.sequence > b.sequence ? 1 : 0);
}
// The heap retains ordering between releases and accepts earlier-path arrivals
// without re-sorting the remaining queue. Each update takes O(log n) comparisons.
function insert(state: State, request: Request): void {
  const heap = state.heap;
  let index = heap.length;
  heap.push(request);
  while (index > 0) {
    const parent = (index - 1) >>> 1;
    if (compare(state, heap[parent], request) <= 0) break;
    heap[index] = heap[parent];
    index = parent;
  }
  heap[index] = request;
}
function take(state: State): Request {
  const heap = state.heap,
    request = heap[0],
    last = heap.pop()!;
  if (heap.length) {
    let index = 0;
    while (index * 2 + 1 < heap.length) {
      let child = index * 2 + 1;
      if (child + 1 < heap.length && compare(state, heap[child + 1], heap[child]) < 0) child++;
      if (compare(state, last, heap[child]) <= 0) break;
      heap[index] = heap[child];
      index = child;
    }
    heap[index] = last;
  }
  return request;
}
function changed(state: State): void {
  if (state.queued) return;
  state.queued = true;
  queueMicrotask(() => {
    state.queued = false;
    // Explicit phase transitions maintain this quiescence count; no frame scan
    // or estimate based on elapsed promise ticks participates in admission.
    if (state.active !== 0 || state.pending.size === 0) return;
    let request: Request;
    try {
      for (const incoming of state.incoming) insert(state, incoming);
      state.incoming = [];
      request = take(state);
    } catch (cause) {
      // A throwing comparator can leave a partially updated heap. The independent
      // set still owns every request, including ones not inserted or removed yet.
      state.heap = [];
      state.incoming = [];
      for (const pending of state.pending) {
        phase(pending.frame, "running");
        pending.reject(cause);
      }
      state.pending.clear();
      return;
    }
    state.pending.delete(request);
    phase(request.frame, "running");
    // Keep the frame active through native promise delivery and its continuation.
    // Generated boundaries must park or finish it before another release occurs.
    Promise.resolve()
      .then(request.deliver)
      .then(checkedCompletion)
      .then(request.resolve, request.reject);
  });
}
// Alternate comparison is compiler-conformance input only. Generated authored
// assertions always use the canonical comparator and expose no scheduling syntax.
export function createBarrier(root: InvocationIdentity, compare = compareInvocations): Barrier {
  invocationPath(root);
  const value = token<Barrier>();
  barriers.set(value, {
    root,
    frames: new Set(),
    identities: new WeakSet(),
    active: 0,
    pending: new Set(),
    incoming: [],
    heap: [],
    sequence: 0n,
    queued: false,
    compare,
  });
  return value;
}
export function reserveFrame(owner: Barrier, identity: InvocationIdentity): Frame {
  const state = barrier(owner);
  compareInvocations(state.root, identity);
  if (state.identities.has(identity)) throw new TypeError("invocation frame already reserved");
  state.identities.add(identity);
  const value = token<Frame>(),
    entry: FrameState = { barrier: state, identity, phase: "reserved" };
  frames.set(value, entry);
  state.frames.add(entry);
  state.active++;
  return value;
}
function transition(value: Frame, from: Phase, to: Phase): void {
  const entry = frame(value);
  if (entry.phase !== from) throw new TypeError("invalid assertion frame transition");
  phase(entry, to);
  if (to === "done") entry.barrier.frames.delete(entry);
  changed(entry.barrier);
}
export function startFrame(value: Frame): void {
  transition(value, "reserved", "running");
}
export function suspendFrame(value: Frame): void {
  transition(value, "running", "waiting");
}
export function resumeFrame(value: Frame): void {
  transition(value, "waiting", "running");
}
export function finishFrame(value: Frame): void {
  transition(value, "running", "done");
}
export function abandonFrame(value: Frame): void {
  transition(value, "reserved", "done");
}
export function fixtureEvent(
  value: Frame,
  deliver: () => Completion | Promise<Completion>,
): Promise<Completion> {
  const entry = frame(value);
  if (entry.phase !== "running")
    throw new TypeError("fixture requested outside a running invocation");
  phase(entry, "fixture");
  const pending = new Promise<Completion>((resolve, reject) => {
    const state = entry.barrier,
      request = { frame: entry, sequence: state.sequence++, deliver, resolve, reject };
    state.pending.add(request);
    state.incoming.push(request);
  });
  changed(entry.barrier);
  return pending;
}
export function barrierState(owner: Barrier) {
  const state = barrier(owner);
  return Object.freeze({
    pending: state.pending.size,
    frames: Object.freeze(
      [...state.frames].map((value) =>
        Object.freeze({ path: invocationPath(value.identity), phase: value.phase }),
      ),
    ),
  });
}
