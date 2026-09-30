// K01: bounded local handle registry encoding the native-value ownership
// and move rules. Pure in-memory mechanics for the schema self-check: no
// I/O, timers, transports, or live runtimes.
//
// Rules enforced:
//   - every handle binds owner, kind, session, and generation;
//   - mutable handles never transfer between owners or sessions; cross
//     use rejects instead of moving;
//   - close invalidates the session generation; old handles go stale and
//     repeated close joins the first disposal receipt;
//   - every mint is charged against finite limits; over-cap mints reject
//     with resource-limit instead of growing without bound.

import {
  checkHandleUse,
  mintHandle,
  type HandleKind,
  type NativeHandle,
  type NativeValueLimits,
} from "./schema.ts";
import { NativeSchemaError } from "./schema.ts";

export type CloseReceipt = Readonly<{
  sessionId: string;
  released: number;
  remaining: number;
  forced: readonly string[];
  joined: boolean;
}>;

type SessionState = {
  owner: string;
  live: boolean;
  generation: number;
  handles: number;
  pendingActions: number;
  gates: number;
  faults: number;
  releasedOnce: string[];
  receipt: CloseReceipt | null;
};

export class NativeHandleRegistry {
  private readonly limits: NativeValueLimits;
  private readonly sessions = new Map<string, SessionState>();
  private counter = 0;

  constructor(limits: NativeValueLimits) {
    this.limits = limits;
  }

  get sessionCount(): number {
    return this.sessions.size;
  }

  openSession(owner: string, sessionId: string): NativeHandle<"session"> {
    if (this.sessions.has(sessionId)) {
      throw new NativeSchemaError("rejected", "kind-collision", "session id already reserved");
    }
    if (this.sessions.size >= this.limits.maxSessions) {
      throw new NativeSchemaError("rejected", "resource-limit", "session cap reached");
    }
    this.sessions.set(sessionId, {
      owner,
      live: true,
      generation: 0,
      handles: 0,
      pendingActions: 0,
      gates: 0,
      faults: 0,
      releasedOnce: [],
      receipt: null,
    });
    return mintHandle("session", `${sessionId}:root`, sessionId, owner, 0);
  }

  mint<K extends Exclude<HandleKind, "session">>(
    session: NativeHandle<"session">,
    owner: string,
    kind: K,
  ): NativeHandle<K> {
    const state = this.requireLiveSession(session, owner);
    if (state.handles >= this.limits.maxHandlesPerSession) {
      throw new NativeSchemaError("rejected", "resource-limit", "handle cap reached");
    }
    if (kind === "action" && state.pendingActions >= this.limits.maxPendingActions) {
      throw new NativeSchemaError("rejected", "resource-limit", "pending-action cap reached");
    }
    if (kind === "gate" && state.gates >= this.limits.maxGatesPerSession) {
      throw new NativeSchemaError("rejected", "resource-limit", "gate cap reached");
    }
    if (kind === "fault" && state.faults >= this.limits.maxFaultsPerSession) {
      throw new NativeSchemaError("rejected", "resource-limit", "fault cap reached");
    }
    this.counter += 1;
    state.handles += 1;
    if (kind === "action") state.pendingActions += 1;
    if (kind === "gate") state.gates += 1;
    if (kind === "fault") state.faults += 1;
    return mintHandle(
      kind,
      `${session.sessionId}:h${this.counter}`,
      session.sessionId,
      owner,
      state.generation,
    );
  }

  // Present a handle for use. Validates owner, session binding, liveness,
  // and generation; rejects cross-session or cross-owner use without moving
  // the handle.
  use(handle: NativeHandle, owner: string): void {
    const state = this.sessions.get(handle.sessionId);
    if (!state) {
      throw new NativeSchemaError("rejected", "stale-handle", "unknown session for handle");
    }
    if (state.owner !== owner || handle.owner !== owner) {
      throw new NativeSchemaError("rejected", "wrong-owner", "handle owner mismatch");
    }
    checkHandleUse(handle, {
      owner,
      sessionLive: state.live,
      liveGeneration: state.generation,
    });
  }

  // Gates and faults release exactly once; repeats join the first release.
  releaseOnce(handle: NativeHandle<"gate" | "fault">, owner: string): boolean {
    this.use(handle, owner);
    const state = this.sessions.get(handle.sessionId);
    if (!state) {
      throw new NativeSchemaError("rejected", "stale-handle", "unknown session for handle");
    }
    if (state.releasedOnce.includes(handle.id)) return true;
    state.releasedOnce.push(handle.id);
    return false;
  }

  settleAction(handle: NativeHandle<"action">, owner: string): void {
    this.use(handle, owner);
    const state = this.sessions.get(handle.sessionId);
    if (state && state.pendingActions > 0) state.pendingActions -= 1;
  }

  closeSession(session: NativeHandle<"session">, owner: string): CloseReceipt {
    const state = this.sessions.get(session.sessionId);
    if (!state) {
      throw new NativeSchemaError("rejected", "stale-handle", "unknown session");
    }
    if (state.owner !== owner || session.owner !== owner) {
      throw new NativeSchemaError("rejected", "wrong-owner", "session owner mismatch");
    }
    if (state.receipt) {
      return state.receipt;
    }
    state.live = false;
    state.generation += 1;
    const pending = state.pendingActions + state.gates + state.faults;
    const forced: string[] = pending > 0 ? ["cancel-pending"] : [];
    const receipt: CloseReceipt = Object.freeze({
      sessionId: session.sessionId,
      released: state.handles,
      remaining: 0,
      forced: Object.freeze(forced),
      joined: false,
    });
    state.handles = 0;
    state.pendingActions = 0;
    state.gates = 0;
    state.faults = 0;
    state.receipt = receipt;
    return receipt;
  }

  private requireLiveSession(session: NativeHandle<"session">, owner: string): SessionState {
    const state = this.sessions.get(session.sessionId);
    if (!state) {
      throw new NativeSchemaError("rejected", "stale-handle", "unknown session");
    }
    if (state.owner !== owner || session.owner !== owner) {
      throw new NativeSchemaError("rejected", "wrong-owner", "session owner mismatch");
    }
    if (!state.live) {
      throw new NativeSchemaError("rejected", "closed-handle", "session is closed");
    }
    if (session.generation !== state.generation) {
      throw new NativeSchemaError("rejected", "stale-handle", "stale session generation");
    }
    return state;
  }
}
