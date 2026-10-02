// NT-I05 behavioral port of tools/runtime/test-services/http-peer/websocket.ts
// (K21, accepted). The runtime ship boundary forbids runtime/ from importing
// tools/, so the I05 slice carries its own copy of the websocket observer.
// Observer logic below is unchanged from K21; only this header and the peer
// import (../i04/peer-service.ts, the NT-I04 port of the K20 peer core) differ.

// K21: typed in-memory WebSocket client observations.
//
// In-memory doubles only: no sockets, no network, no timers, no threads. The
// caller connects a client, sends frames, and explicitly injects the scripted
// remote side via deliverEvent/deliverRemoteClose. Remote events and remote
// closes never arrive on their own: every inbound frame is an explicit,
// counted delivery. Every bound is finite: declared destinations, live
// connections, queued events, queued bytes, payload bytes, close-reason
// bytes.
//
// Structural guarantees (mirroring K20 peer/http):
// - Journaled admission first: foreign destinations and closed-vocabulary
//   violations reject before any state changes.
// - No spontaneous remote input: pollEvent only returns frames previously
//   admitted by deliverEvent; a remote close exists only after an explicit
//   deliverRemoteClose. A claimed-but-undelivered remote close stays absent
//   in facts.
// - Distinct close facts: local_close and remote_close are independent
//   observations. Local close never sets remote_close and remote delivery
//   never sets local_close. The connection reaches the terminal closed
//   state only when both sides have closed (either order).
// - Accepted-vs-consumed separation: events admitted by delivery
//   (delivered_events) and events handed out by polls (consumed_events) are
//   independent counters.
// - Dropped work explicit: events still queued when the handshake completes
//   are counted in dropped_events/dropped_bytes on the completing receipt
//   and in facts; they never silently vanish or become polls afterwards.
// - Forged remote-close detection: a second deliverRemoteClose, a delivery
//   after the terminal state, or a delivery contradicting observed state
//   rejects; facts keep the single genuine observation.
// - Empty poll on a live connection returns event:null (not an error and
//   not terminal); terminal state is observable only via facts/state.

import {
  checkDeclaredSet,
  checkOwner,
  checkPositiveInt,
  HttpPeerError,
} from "../i04/peer-service.ts";

// ---------------------------------------------------------------------------
// Closed vocabularies
// ---------------------------------------------------------------------------

export const WS_OPCODES = ["text", "binary", "ping", "pong"] as const;
export type WsOpcode = (typeof WS_OPCODES)[number];

export const WS_CLOSE_ORIGINS = ["local", "remote"] as const;
export type WsCloseOrigin = (typeof WS_CLOSE_ORIGINS)[number];

export const WS_CONNECTION_STATES = ["open", "local-closed", "remote-closed", "closed"] as const;
export type WsConnectionState = (typeof WS_CONNECTION_STATES)[number];

function isOpcode(value: unknown): value is WsOpcode {
  return typeof value === "string" && (WS_OPCODES as readonly string[]).includes(value);
}

// WebSocket close codes 1000..4999 (protocol + application range).
const WS_MIN_CLOSE_CODE = 1000;
const WS_MAX_CLOSE_CODE = 4999;

function checkCloseCode(value: unknown): number {
  return checkPositiveInt(value, "close code", WS_MIN_CLOSE_CODE, WS_MAX_CLOSE_CODE);
}

const WS_REASON_ASCII = /^[\x20-\x7E]*$/;

function checkCloseReason(value: unknown, maxBytes: number): string {
  if (typeof value !== "string") {
    throw new HttpPeerError("rejected", "invalid-request", "close reason must be a string");
  }
  if (!WS_REASON_ASCII.test(value)) {
    throw new HttpPeerError("rejected", "invalid-request", "close reason must be printable ASCII");
  }
  if (value.length > maxBytes) {
    throw new HttpPeerError(
      "rejected",
      "invalid-request",
      `close reason exceeds the ${maxBytes}-byte reason cap`,
    );
  }
  return value;
}

// Unlike peer byte chunks, a WebSocket frame may carry an empty payload
// (legal for control frames and empty messages), so 0..max is admitted.
function checkWsPayload(value: unknown, what: string, max: number): readonly number[] {
  if (!Array.isArray(value)) {
    throw new HttpPeerError("rejected", "invalid-request", `${what} must be a byte array`);
  }
  if (value.length > max) {
    throw new HttpPeerError(
      "rejected",
      "resource-limit",
      `${what} exceeds the ${max}-byte payload cap`,
    );
  }
  for (const byte of value) {
    if (typeof byte !== "number" || !Number.isInteger(byte) || byte < 0 || byte > 255) {
      throw new HttpPeerError("rejected", "invalid-request", `${what} must hold bytes 0..255`);
    }
  }
  return Object.freeze([...value]);
}

// ---------------------------------------------------------------------------
// Limits
// ---------------------------------------------------------------------------

export type WsLimits = Readonly<{
  max_connections: number;
  max_pending_events: number;
  max_payload_bytes: number;
  max_buffer_bytes: number;
  max_close_reason_bytes: number;
}>;

export const WS_LIMIT_KEYS = [
  "max_connections",
  "max_pending_events",
  "max_payload_bytes",
  "max_buffer_bytes",
  "max_close_reason_bytes",
] as const;

const WS_HARD_CEILINGS: Record<(typeof WS_LIMIT_KEYS)[number], number> = {
  max_connections: 1024,
  max_pending_events: 256,
  max_payload_bytes: 1 << 16,
  max_buffer_bytes: 1 << 20,
  max_close_reason_bytes: 1024,
};

export function checkWsLimits(value: unknown): WsLimits {
  if (typeof value !== "object" || value === null || Array.isArray(value)) {
    throw new HttpPeerError("rejected", "invalid-request", "websocket limits must be a record");
  }
  const record = value as Record<string, unknown>;
  const out: Record<string, number> = {};
  for (const key of WS_LIMIT_KEYS) {
    // max_close_reason_bytes may be 0 (empty reasons only); every other
    // bound must admit at least one.
    const min = key === "max_close_reason_bytes" ? 0 : 1;
    out[key] = checkPositiveInt(record[key], `websocket limit ${key}`, min, WS_HARD_CEILINGS[key]);
  }
  const payload = out["max_payload_bytes"] ?? 1;
  const buffer = out["max_buffer_bytes"] ?? 1;
  if (payload > buffer) {
    throw new HttpPeerError(
      "rejected",
      "invalid-request",
      "websocket limit max_payload_bytes must not exceed max_buffer_bytes",
    );
  }
  return Object.freeze({
    max_connections: out["max_connections"] ?? 1,
    max_pending_events: out["max_pending_events"] ?? 1,
    max_payload_bytes: payload,
    max_buffer_bytes: buffer,
    max_close_reason_bytes: out["max_close_reason_bytes"] ?? 0,
  });
}

// ---------------------------------------------------------------------------
// Handles, facts, receipts (all frozen)
// ---------------------------------------------------------------------------

export type WsConnectionHandle = Readonly<{
  kind: "websocket";
  id: string;
  owner: string;
  destination: string;
}>;

export type WsCloseFacts = Readonly<{
  origin: WsCloseOrigin;
  close_code: number;
  reason: string;
}>;

export type WsEvent = Readonly<{
  opcode: WsOpcode;
  payload: readonly number[];
}>;

export type WsConnectionFacts = Readonly<{
  id: string;
  owner: string;
  destination: string;
  state: WsConnectionState;
  local_close: WsCloseFacts | null;
  remote_close: WsCloseFacts | null;
  sent_frames: number;
  sent_bytes: number;
  delivered_events: number;
  consumed_events: number;
  pending_events: number;
  pending_bytes: number;
  dropped_events: number;
  dropped_bytes: number;
}>;

export type WsSendReceipt = Readonly<{
  opcode: WsOpcode;
  accepted: number;
  sent_frames_total: number;
  sent_bytes_total: number;
}>;

export type WsDeliverReceipt = Readonly<{
  opcode: WsOpcode;
  pending_events: number;
  pending_bytes: number;
  delivered_total: number;
}>;

export type WsPollResult = Readonly<{
  event: WsEvent | null;
  pending_events: number;
  consumed_total: number;
  delivered_total: number;
}>;

export type WsCloseReceipt = Readonly<{
  origin: WsCloseOrigin;
  close_code: number;
  terminal: boolean;
  dropped_events: number;
  dropped_bytes: number;
}>;

// ---------------------------------------------------------------------------
// Internal records (mutable; never escape)
// ---------------------------------------------------------------------------

type WsQueuedEvent = {
  opcode: WsOpcode;
  payload: number[];
};

type WsCloseRecord = {
  close_code: number;
  reason: string;
};

type WsConnectionRecord = {
  id: string;
  owner: string;
  destination: string;
  local_close: WsCloseRecord | null;
  remote_close: WsCloseRecord | null;
  queue: WsQueuedEvent[];
  queued_bytes: number;
  sent_frames: number;
  sent_bytes: number;
  delivered_events: number;
  consumed_events: number;
  dropped_events: number;
  dropped_bytes: number;
};

function wsStateOf(record: WsConnectionRecord): WsConnectionState {
  if (record.local_close !== null && record.remote_close !== null) {
    return "closed";
  }
  if (record.local_close !== null) {
    return "local-closed";
  }
  if (record.remote_close !== null) {
    return "remote-closed";
  }
  return "open";
}

// ---------------------------------------------------------------------------
// Service
// ---------------------------------------------------------------------------

export class WebSocketPeerService {
  private readonly declared: readonly string[];
  private readonly limits: WsLimits;
  private readonly connections = new Map<string, WsConnectionRecord>();
  private nextConnection = 1;

  constructor(declaredDestinations: unknown, limits: WsLimits) {
    this.declared = checkDeclaredSet(declaredDestinations, "websocket destinations");
    this.limits = limits;
  }

  get live_connection_count(): number {
    let count = 0;
    for (const record of this.connections.values()) {
      if (wsStateOf(record) !== "closed") {
        count += 1;
      }
    }
    return count;
  }

  get pending_event_count(): number {
    let count = 0;
    for (const record of this.connections.values()) {
      if (wsStateOf(record) !== "closed") {
        count += record.queue.length;
      }
    }
    return count;
  }

  connect(ownerValue: unknown, destinationValue: unknown): WsConnectionHandle {
    const owner = checkOwner(ownerValue);
    const destination = this.resolveDestination(destinationValue);
    if (this.live_connection_count >= this.limits.max_connections) {
      throw new HttpPeerError("rejected", "resource-limit", "connection cap reached");
    }
    const id = `w${this.nextConnection++}`;
    this.connections.set(id, {
      id,
      owner,
      destination,
      local_close: null,
      remote_close: null,
      queue: [],
      queued_bytes: 0,
      sent_frames: 0,
      sent_bytes: 0,
      delivered_events: 0,
      consumed_events: 0,
      dropped_events: 0,
      dropped_bytes: 0,
    });
    return Object.freeze({ kind: "websocket", id, owner, destination });
  }

  send(
    ownerValue: unknown,
    connectionIdValue: unknown,
    opcodeValue: unknown,
    payloadValue: unknown,
  ): WsSendReceipt {
    const owner = checkOwner(ownerValue);
    const record = this.useConnection(connectionIdValue, owner);
    const state = wsStateOf(record);
    if (state === "closed") {
      throw new HttpPeerError("rejected", "closed-handle", "connection is closed");
    }
    if (state !== "open") {
      throw new HttpPeerError("rejected", "invalid-state", "connection is half-closed");
    }
    if (!isOpcode(opcodeValue)) {
      throw new HttpPeerError(
        "rejected",
        "unsupported-capability",
        "opcode must be a known frame type",
      );
    }
    const payload = checkWsPayload(payloadValue, "send payload", this.limits.max_payload_bytes);
    record.sent_frames += 1;
    record.sent_bytes += payload.length;
    return Object.freeze({
      opcode: opcodeValue,
      accepted: payload.length,
      sent_frames_total: record.sent_frames,
      sent_bytes_total: record.sent_bytes,
    });
  }

  deliverEvent(
    ownerValue: unknown,
    connectionIdValue: unknown,
    opcodeValue: unknown,
    payloadValue: unknown,
  ): WsDeliverReceipt {
    const owner = checkOwner(ownerValue);
    const record = this.useConnection(connectionIdValue, owner);
    const state = wsStateOf(record);
    if (state === "closed") {
      throw new HttpPeerError("rejected", "closed-handle", "connection is closed");
    }
    if (state === "remote-closed") {
      // The remote side already closed: further remote events contradict the
      // observed close and are treated as forged input.
      throw new HttpPeerError("rejected", "invalid-state", "remote side already closed");
    }
    if (!isOpcode(opcodeValue)) {
      throw new HttpPeerError(
        "rejected",
        "unsupported-capability",
        "opcode must be a known frame type",
      );
    }
    const payload = checkWsPayload(payloadValue, "event payload", this.limits.max_payload_bytes);
    if (record.queue.length >= this.limits.max_pending_events) {
      throw new HttpPeerError("rejected", "backpressure", "pending event queue is full");
    }
    if (record.queued_bytes + payload.length > this.limits.max_buffer_bytes) {
      // Backpressure rejects the whole event: nothing is partially queued.
      throw new HttpPeerError("rejected", "backpressure", "pending event buffer is full");
    }
    record.queue.push({ opcode: opcodeValue, payload: [...payload] });
    record.queued_bytes += payload.length;
    record.delivered_events += 1;
    return Object.freeze({
      opcode: opcodeValue,
      pending_events: record.queue.length,
      pending_bytes: record.queued_bytes,
      delivered_total: record.delivered_events,
    });
  }

  pollEvent(ownerValue: unknown, connectionIdValue: unknown): WsPollResult {
    const owner = checkOwner(ownerValue);
    const record = this.useConnection(connectionIdValue, owner);
    if (wsStateOf(record) === "closed") {
      throw new HttpPeerError("rejected", "closed-handle", "connection is closed");
    }
    const next = record.queue.shift();
    if (next === undefined) {
      // An empty poll on a live connection is not an error and proves
      // nothing about absence: only facts over delivered events do.
      return Object.freeze({
        event: null,
        pending_events: 0,
        consumed_total: record.consumed_events,
        delivered_total: record.delivered_events,
      });
    }
    record.queued_bytes -= next.payload.length;
    record.consumed_events += 1;
    const event: WsEvent = Object.freeze({
      opcode: next.opcode,
      payload: Object.freeze([...next.payload]),
    });
    return Object.freeze({
      event,
      pending_events: record.queue.length,
      consumed_total: record.consumed_events,
      delivered_total: record.delivered_events,
    });
  }

  close(
    ownerValue: unknown,
    connectionIdValue: unknown,
    codeValue: unknown,
    reasonValue: unknown,
  ): WsCloseReceipt {
    const owner = checkOwner(ownerValue);
    const record = this.useConnection(connectionIdValue, owner);
    if (wsStateOf(record) === "closed") {
      throw new HttpPeerError("rejected", "closed-handle", "connection is closed");
    }
    if (record.local_close !== null) {
      throw new HttpPeerError("rejected", "invalid-state", "local close already recorded");
    }
    const code = checkCloseCode(codeValue);
    const reason = checkCloseReason(reasonValue, this.limits.max_close_reason_bytes);
    record.local_close = { close_code: code, reason };
    return this.closeReceipt(record, "local");
  }

  deliverRemoteClose(
    ownerValue: unknown,
    connectionIdValue: unknown,
    codeValue: unknown,
    reasonValue: unknown,
  ): WsCloseReceipt {
    const owner = checkOwner(ownerValue);
    const record = this.useConnection(connectionIdValue, owner);
    if (wsStateOf(record) === "closed") {
      throw new HttpPeerError("rejected", "closed-handle", "connection is closed");
    }
    if (record.remote_close !== null) {
      // A second remote close contradicts the single observed one: forged.
      throw new HttpPeerError("rejected", "invalid-state", "remote close already recorded");
    }
    const code = checkCloseCode(codeValue);
    const reason = checkCloseReason(reasonValue, this.limits.max_close_reason_bytes);
    record.remote_close = { close_code: code, reason };
    return this.closeReceipt(record, "remote");
  }

  connectionFacts(ownerValue: unknown, connectionIdValue: unknown): WsConnectionFacts {
    const owner = checkOwner(ownerValue);
    const record = this.useConnection(connectionIdValue, owner);
    return Object.freeze({
      id: record.id,
      owner: record.owner,
      destination: record.destination,
      state: wsStateOf(record),
      local_close:
        record.local_close === null
          ? null
          : Object.freeze({
              origin: "local",
              close_code: record.local_close.close_code,
              reason: record.local_close.reason,
            }),
      remote_close:
        record.remote_close === null
          ? null
          : Object.freeze({
              origin: "remote",
              close_code: record.remote_close.close_code,
              reason: record.remote_close.reason,
            }),
      sent_frames: record.sent_frames,
      sent_bytes: record.sent_bytes,
      delivered_events: record.delivered_events,
      consumed_events: record.consumed_events,
      pending_events: record.queue.length,
      pending_bytes: record.queued_bytes,
      dropped_events: record.dropped_events,
      dropped_bytes: record.dropped_bytes,
    });
  }

  private resolveDestination(value: unknown): string {
    if (typeof value !== "string") {
      throw new HttpPeerError("rejected", "invalid-request", "destination must be a string");
    }
    // Foreign destinations reject before any state changes (P29 admission).
    if (!this.declared.includes(value)) {
      throw new HttpPeerError("rejected", "foreign-destination", "destination is not declared");
    }
    return value;
  }

  private useConnection(idValue: unknown, owner: string): WsConnectionRecord {
    if (typeof idValue !== "string") {
      throw new HttpPeerError("rejected", "invalid-request", "connection id must be a string");
    }
    const record = this.connections.get(idValue);
    if (record === undefined) {
      throw new HttpPeerError("rejected", "invalid-request", "unknown connection id");
    }
    if (record.owner !== owner) {
      throw new HttpPeerError("rejected", "wrong-owner", "connection belongs to another owner");
    }
    return record;
  }

  // The operation that records the second close completes the handshake:
  // queued events become counted drops and the queue is cleared so they can
  // never be polled afterwards.
  private closeReceipt(record: WsConnectionRecord, origin: WsCloseOrigin): WsCloseReceipt {
    const terminal = wsStateOf(record) === "closed";
    let dropped_events = 0;
    let dropped_bytes = 0;
    if (terminal) {
      dropped_events = record.queue.length;
      dropped_bytes = record.queued_bytes;
      record.dropped_events += dropped_events;
      record.dropped_bytes += dropped_bytes;
      record.queue = [];
      record.queued_bytes = 0;
    }
    const close = origin === "local" ? record.local_close : record.remote_close;
    const close_code = close === null ? WS_MIN_CLOSE_CODE : close.close_code;
    return Object.freeze({ origin, close_code, terminal, dropped_events, dropped_bytes });
  }
}
