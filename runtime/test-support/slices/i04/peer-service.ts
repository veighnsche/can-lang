// NT-I04 behavioral port of tools/runtime/test-services/http-peer/peer.ts (K20, accepted). The runtime ship boundary
// forbids runtime/ from importing tools/ (tools ship as host-executed files,
// never as bundle modules), so the I04 slice carries its own copy of the
// doubles. Service logic below is unchanged from K20; only this header and
// the peer import differ.

// K20: typed in-memory peer listener/read/write/half-close operations.
//
// In-memory doubles only: no sockets, no fetch, no network, no timers, no
// threads. A dial pairs a dialer endpoint with a queued accept on a declared
// listener; accept mints the peer endpoint; writes move bytes into the peer
// inbound buffer synchronously. Every bound is finite: declared destinations,
// live handles, queued accepts, buffered bytes, chunk bytes, explicit
// retries.
//
// Structural guarantees (mirroring P29 generic ownership):
// - Journaled admission first: foreign destinations and closed-vocabulary
//   violations reject before any state changes.
// - No implicit retry: a failed dial stays failed until an explicit
//   retryDial call; every attempt is counted in attempt_count/retry_count.
// - Accepted-vs-consumed separation: bytes admitted into an inbound buffer
//   (accepted_bytes) and bytes handed out by reads (consumed_bytes) are
//   tracked as independent counters.
// - EOF explicit: an empty read on a live connection reports eof:false; eof
//   is true only when the peer write side is gone AND the buffer is drained.
// - Partial state explicit: short reads report truncated:true with the
//   remainder still buffered; half-close discards are counted in facts.

export const HTTP_PEER_SCHEMA_VERSION = "1" as const;

// ---------------------------------------------------------------------------
// Shared error shape (mirrors NativeSchemaError: outcome + kind + message)
// ---------------------------------------------------------------------------

export const HTTP_PEER_OUTCOMES = ["rejected", "failed"] as const;
export type HttpPeerOutcome = (typeof HTTP_PEER_OUTCOMES)[number];

export const HTTP_PEER_KINDS = [
  "invalid-request",
  "unsupported-capability",
  "foreign-destination",
  "resource-limit",
  "wrong-owner",
  "closed-handle",
  "backpressure",
  "no-listener",
  "invalid-state",
  "header-limit",
  "body-limit",
] as const;
export type HttpPeerKind = (typeof HTTP_PEER_KINDS)[number];

export class HttpPeerError extends Error {
  readonly outcome: HttpPeerOutcome;
  readonly kind: HttpPeerKind;
  constructor(outcome: HttpPeerOutcome, kind: HttpPeerKind, message: string) {
    super(message);
    this.name = "HttpPeerError";
    this.outcome = outcome;
    this.kind = kind;
  }
}

// ---------------------------------------------------------------------------
// Closed vocabularies
// ---------------------------------------------------------------------------

export const PEER_HANDLE_KINDS = ["listener", "connection", "dial"] as const;
export type PeerHandleKind = (typeof PEER_HANDLE_KINDS)[number];

export const PEER_SIDES = ["dialer", "acceptor"] as const;
export type PeerSide = (typeof PEER_SIDES)[number];

export const PEER_HALF_CLOSE_DIRECTIONS = ["read", "write"] as const;
export type PeerHalfCloseDirection = (typeof PEER_HALF_CLOSE_DIRECTIONS)[number];

export const PEER_DIAL_STATES = ["completed", "failed"] as const;
export type PeerDialState = (typeof PEER_DIAL_STATES)[number];

export const PEER_DIAL_FAIL_REASONS = [
  "no-listener",
  "listener-closed",
  "accept-queue-full",
  "connection-cap",
] as const;
export type PeerDialFailReason = (typeof PEER_DIAL_FAIL_REASONS)[number];

// ---------------------------------------------------------------------------
// Shared checked primitives (also used by ./http.ts)
// ---------------------------------------------------------------------------

export function checkOwner(value: unknown): string {
  if (typeof value !== "string" || value.length === 0 || value.length > 128) {
    throw new HttpPeerError("rejected", "invalid-request", "owner must be a 1..128 char string");
  }
  return value;
}

export function checkDeclaredSet(value: unknown, what: string): readonly string[] {
  if (!Array.isArray(value)) {
    throw new HttpPeerError("rejected", "invalid-request", `${what} must be an array`);
  }
  if (value.length === 0 || value.length > 64) {
    throw new HttpPeerError("rejected", "invalid-request", `${what} must declare 1..64 entries`);
  }
  const seen = new Set<string>();
  for (const entry of value) {
    if (typeof entry !== "string" || entry.length === 0 || entry.length > 256) {
      throw new HttpPeerError(
        "rejected",
        "invalid-request",
        `${what} entries must be 1..256 char strings`,
      );
    }
    if (seen.has(entry)) {
      throw new HttpPeerError("rejected", "invalid-request", `${what} entries must be unique`);
    }
    seen.add(entry);
  }
  return Object.freeze([...value]);
}

export function checkPositiveInt(value: unknown, what: string, min: number, max: number): number {
  if (typeof value !== "number" || !Number.isInteger(value) || value < min || value > max) {
    throw new HttpPeerError(
      "rejected",
      "invalid-request",
      `${what} must be an integer ${min}..${max}`,
    );
  }
  return value;
}

export function checkBytes(value: unknown, what: string, max: number): readonly number[] {
  if (!Array.isArray(value)) {
    throw new HttpPeerError("rejected", "invalid-request", `${what} must be a byte array`);
  }
  if (value.length === 0) {
    throw new HttpPeerError("rejected", "invalid-request", `${what} must carry at least one byte`);
  }
  if (value.length > max) {
    throw new HttpPeerError(
      "rejected",
      "resource-limit",
      `${what} exceeds the ${max}-byte chunk cap`,
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

export type PeerLimits = Readonly<{
  max_listeners: number;
  max_connections: number;
  max_pending_accepts: number;
  max_dials: number;
  max_buffer_bytes: number;
  max_chunk_bytes: number;
  max_explicit_retries: number;
}>;

export const PEER_LIMIT_KEYS = [
  "max_listeners",
  "max_connections",
  "max_pending_accepts",
  "max_dials",
  "max_buffer_bytes",
  "max_chunk_bytes",
  "max_explicit_retries",
] as const;

const PEER_HARD_CEILINGS: Record<(typeof PEER_LIMIT_KEYS)[number], number> = {
  max_listeners: 64,
  max_connections: 1024,
  max_pending_accepts: 256,
  max_dials: 1024,
  max_buffer_bytes: 1 << 20,
  max_chunk_bytes: 1 << 16,
  max_explicit_retries: 16,
};

export function checkPeerLimits(value: unknown): PeerLimits {
  if (typeof value !== "object" || value === null || Array.isArray(value)) {
    throw new HttpPeerError("rejected", "invalid-request", "peer limits must be a record");
  }
  const record = value as Record<string, unknown>;
  const out: Record<string, number> = {};
  for (const key of PEER_LIMIT_KEYS) {
    // max_explicit_retries may be 0 (explicit retries disabled by bounds);
    // every other bound must admit at least one.
    const min = key === "max_explicit_retries" ? 0 : 1;
    out[key] = checkPositiveInt(record[key], `peer limit ${key}`, min, PEER_HARD_CEILINGS[key]);
  }
  const chunk = out["max_chunk_bytes"] ?? 1;
  const buffer = out["max_buffer_bytes"] ?? 1;
  if (chunk > buffer) {
    throw new HttpPeerError(
      "rejected",
      "invalid-request",
      "peer limit max_chunk_bytes must not exceed max_buffer_bytes",
    );
  }
  return Object.freeze({
    max_listeners: out["max_listeners"] ?? 1,
    max_connections: out["max_connections"] ?? 1,
    max_pending_accepts: out["max_pending_accepts"] ?? 1,
    max_dials: out["max_dials"] ?? 1,
    max_buffer_bytes: buffer,
    max_chunk_bytes: chunk,
    max_explicit_retries: out["max_explicit_retries"] ?? 0,
  });
}

// ---------------------------------------------------------------------------
// Handles, facts, receipts (all frozen)
// ---------------------------------------------------------------------------

export type PeerListenerHandle = Readonly<{
  kind: "listener";
  id: string;
  owner: string;
  destination: string;
}>;

export type PeerConnectionHandle = Readonly<{
  kind: "connection";
  id: string;
  owner: string;
  destination: string;
  side: PeerSide;
}>;

export type PeerDialHandle = Readonly<{
  kind: "dial";
  id: string;
  owner: string;
  destination: string;
}>;

export type PeerListenerFacts = Readonly<{
  id: string;
  owner: string;
  destination: string;
  closed: boolean;
  pending_accepts: number;
  accepted_total: number;
  stale_skipped: number;
  dropped_on_close: number;
}>;

export type PeerConnectionFacts = Readonly<{
  id: string;
  owner: string;
  destination: string;
  side: PeerSide;
  peer_id: string | null;
  peer_gone: boolean;
  read_closed: boolean;
  write_closed: boolean;
  peer_write_closed: boolean;
  closed: boolean;
  buffered_bytes: number;
  accepted_bytes: number;
  consumed_bytes: number;
  discarded_bytes: number;
}>;

export type PeerDialFacts = Readonly<{
  id: string;
  owner: string;
  destination: string;
  state: PeerDialState;
  fail_reason: PeerDialFailReason | null;
  attempt_count: number;
  retry_count: number;
  connection_id: string | null;
}>;

export type PeerDialResult = Readonly<{
  dial: PeerDialHandle;
  state: PeerDialState;
  fail_reason: PeerDialFailReason | null;
  connection: PeerConnectionHandle | null;
  attempt_count: number;
  retry_count: number;
}>;

export type PeerWriteReceipt = Readonly<{
  accepted: number;
  peer_buffered: number;
  peer_accepted_total: number;
}>;

export type PeerReadResult = Readonly<{
  bytes: readonly number[];
  eof: boolean;
  truncated: boolean;
  consumed_total: number;
  accepted_total: number;
}>;

export type PeerConnectionCloseReceipt = Readonly<{
  unread_bytes: number;
  accepted_bytes: number;
  consumed_bytes: number;
  discarded_bytes: number;
}>;

export type PeerListenerCloseReceipt = Readonly<{
  pending_accepts_dropped: number;
  accepted_total: number;
  stale_skipped: number;
}>;

// ---------------------------------------------------------------------------
// Internal records (mutable; never escape)
// ---------------------------------------------------------------------------

type PeerEndpoint = {
  id: string;
  owner: string;
  destination: string;
  side: PeerSide;
  peer_id: string | null;
  peer_gone: boolean;
  read_closed: boolean;
  write_closed: boolean;
  peer_write_closed: boolean;
  closed: boolean;
  inbound: number[];
  accepted_bytes: number;
  consumed_bytes: number;
  discarded_bytes: number;
};

type PeerListenerRecord = {
  id: string;
  owner: string;
  destination: string;
  closed: boolean;
  queue: string[];
  accepted_total: number;
  stale_skipped: number;
  dropped_on_close: number;
};

type PeerDialRecord = {
  id: string;
  owner: string;
  destination: string;
  state: PeerDialState;
  fail_reason: PeerDialFailReason | null;
  attempt_count: number;
  connection_id: string | null;
};

function isHalfCloseDirection(value: unknown): value is PeerHalfCloseDirection {
  return (
    typeof value === "string" && (PEER_HALF_CLOSE_DIRECTIONS as readonly string[]).includes(value)
  );
}

// ---------------------------------------------------------------------------
// Service
// ---------------------------------------------------------------------------

export class PeerListenerService {
  private readonly declared: readonly string[];
  private readonly limits: PeerLimits;
  private readonly listeners = new Map<string, PeerListenerRecord>();
  private readonly endpoints = new Map<string, PeerEndpoint>();
  private readonly dials = new Map<string, PeerDialRecord>();
  private nextListener = 1;
  private nextEndpoint = 1;
  private nextDial = 1;

  constructor(declaredDestinations: unknown, limits: PeerLimits) {
    this.declared = checkDeclaredSet(declaredDestinations, "peer destinations");
    this.limits = limits;
  }

  get live_listener_count(): number {
    let count = 0;
    for (const listener of this.listeners.values()) {
      if (!listener.closed) {
        count += 1;
      }
    }
    return count;
  }

  get live_endpoint_count(): number {
    let count = 0;
    for (const endpoint of this.endpoints.values()) {
      if (!endpoint.closed) {
        count += 1;
      }
    }
    return count;
  }

  get pending_accept_count(): number {
    let count = 0;
    for (const listener of this.listeners.values()) {
      if (!listener.closed) {
        count += listener.queue.length;
      }
    }
    return count;
  }

  get dial_count(): number {
    return this.dials.size;
  }

  openListener(ownerValue: unknown, destinationValue: unknown): PeerListenerHandle {
    const owner = checkOwner(ownerValue);
    const destination = this.resolveDestination(destinationValue);
    if (this.live_listener_count >= this.limits.max_listeners) {
      throw new HttpPeerError("rejected", "resource-limit", "listener cap reached");
    }
    const id = `l${this.nextListener++}`;
    this.listeners.set(id, {
      id,
      owner,
      destination,
      closed: false,
      queue: [],
      accepted_total: 0,
      stale_skipped: 0,
      dropped_on_close: 0,
    });
    return Object.freeze({ kind: "listener", id, owner, destination });
  }

  closeListener(ownerValue: unknown, listenerIdValue: unknown): PeerListenerCloseReceipt {
    const owner = checkOwner(ownerValue);
    const listener = this.useListener(listenerIdValue, owner);
    if (listener.closed) {
      throw new HttpPeerError("rejected", "closed-handle", "listener is closed");
    }
    listener.closed = true;
    // Dropped dialers observe an explicit peer-gone fact: pending writes
    // reject and pending reads drain to an explicit EOF.
    for (const clientId of listener.queue) {
      const client = this.endpoints.get(clientId);
      if (client !== undefined && !client.closed) {
        client.peer_gone = true;
      }
    }
    listener.dropped_on_close = listener.queue.length;
    listener.queue = [];
    return Object.freeze({
      pending_accepts_dropped: listener.dropped_on_close,
      accepted_total: listener.accepted_total,
      stale_skipped: listener.stale_skipped,
    });
  }

  dial(ownerValue: unknown, destinationValue: unknown): PeerDialResult {
    const owner = checkOwner(ownerValue);
    const destination = this.resolveDestination(destinationValue);
    if (this.dials.size >= this.limits.max_dials) {
      throw new HttpPeerError("rejected", "resource-limit", "dial table full");
    }
    const id = `d${this.nextDial++}`;
    const record: PeerDialRecord = {
      id,
      owner,
      destination,
      state: "failed",
      fail_reason: "no-listener",
      attempt_count: 1,
      connection_id: null,
    };
    this.dials.set(id, record);
    this.attemptDial(record);
    return this.dialResult(record);
  }

  retryDial(ownerValue: unknown, dialIdValue: unknown): PeerDialResult {
    const owner = checkOwner(ownerValue);
    const record = this.useDial(dialIdValue, owner);
    if (record.state === "completed") {
      throw new HttpPeerError("rejected", "invalid-state", "dial already completed");
    }
    if (record.attempt_count - 1 >= this.limits.max_explicit_retries) {
      throw new HttpPeerError("rejected", "resource-limit", "explicit retry budget exhausted");
    }
    record.attempt_count += 1;
    this.attemptDial(record);
    return this.dialResult(record);
  }

  accept(ownerValue: unknown, listenerIdValue: unknown): PeerConnectionHandle {
    const owner = checkOwner(ownerValue);
    const listener = this.useListener(listenerIdValue, owner);
    if (listener.closed) {
      throw new HttpPeerError("rejected", "closed-handle", "listener is closed");
    }
    // Skip dialers that closed before accept. Skips are counted in facts;
    // the queue otherwise preserves FIFO order.
    while (listener.queue.length > 0) {
      const head = listener.queue[0] ?? "";
      const client = this.endpoints.get(head);
      if (client === undefined || client.closed) {
        listener.queue.shift();
        listener.stale_skipped += 1;
      } else {
        break;
      }
    }
    if (listener.queue.length === 0) {
      throw new HttpPeerError("rejected", "invalid-state", "accept queue is empty");
    }
    if (this.live_endpoint_count >= this.limits.max_connections) {
      throw new HttpPeerError("rejected", "resource-limit", "connection cap reached");
    }
    const clientId = listener.queue.shift() ?? "";
    const client = this.endpoints.get(clientId);
    if (client === undefined || client.closed) {
      // Single-threaded: unreachable after the stale sweep, but fail closed.
      throw new HttpPeerError("failed", "invalid-state", "accept entry vanished");
    }
    const server = this.mintEndpoint(client.owner, client.destination, "acceptor");
    server.peer_id = client.id;
    client.peer_id = server.id;
    listener.accepted_total += 1;
    return this.connectionHandle(server);
  }

  write(ownerValue: unknown, connectionIdValue: unknown, bytesValue: unknown): PeerWriteReceipt {
    const owner = checkOwner(ownerValue);
    const endpoint = this.useEndpoint(connectionIdValue, owner);
    if (endpoint.closed) {
      throw new HttpPeerError("rejected", "closed-handle", "connection is closed");
    }
    if (endpoint.write_closed) {
      throw new HttpPeerError("rejected", "invalid-state", "write side is half-closed");
    }
    const bytes = checkBytes(bytesValue, "write chunk", this.limits.max_chunk_bytes);
    if (endpoint.peer_gone || endpoint.peer_id === null) {
      throw new HttpPeerError("rejected", "invalid-state", "peer is gone");
    }
    const peer = this.endpoints.get(endpoint.peer_id);
    if (peer === undefined || peer.closed) {
      throw new HttpPeerError("rejected", "invalid-state", "peer is gone");
    }
    if (peer.read_closed) {
      throw new HttpPeerError("rejected", "invalid-state", "peer read side is half-closed");
    }
    if (peer.inbound.length + bytes.length > this.limits.max_buffer_bytes) {
      // Backpressure rejects the whole chunk: nothing is partially accepted.
      throw new HttpPeerError("rejected", "backpressure", "peer inbound buffer is full");
    }
    for (const byte of bytes) {
      peer.inbound.push(byte);
    }
    peer.accepted_bytes += bytes.length;
    return Object.freeze({
      accepted: bytes.length,
      peer_buffered: peer.inbound.length,
      peer_accepted_total: peer.accepted_bytes,
    });
  }

  read(ownerValue: unknown, connectionIdValue: unknown, maxBytesValue: unknown): PeerReadResult {
    const owner = checkOwner(ownerValue);
    const endpoint = this.useEndpoint(connectionIdValue, owner);
    if (endpoint.closed) {
      throw new HttpPeerError("rejected", "closed-handle", "connection is closed");
    }
    if (endpoint.read_closed) {
      throw new HttpPeerError("rejected", "invalid-state", "read side is half-closed");
    }
    const max = checkPositiveInt(maxBytesValue, "read max_bytes", 1, this.limits.max_chunk_bytes);
    const take = Math.min(max, endpoint.inbound.length);
    const out = endpoint.inbound.splice(0, take);
    endpoint.consumed_bytes += out.length;
    // EOF is explicit: it requires the peer write side to be gone AND a
    // drained buffer. An empty read on a live connection reports eof:false.
    const drained = endpoint.inbound.length === 0;
    const eof = drained && (endpoint.peer_write_closed || endpoint.peer_gone);
    return Object.freeze({
      bytes: Object.freeze(out),
      eof,
      truncated: !drained,
      consumed_total: endpoint.consumed_bytes,
      accepted_total: endpoint.accepted_bytes,
    });
  }

  halfClose(
    ownerValue: unknown,
    connectionIdValue: unknown,
    directionValue: unknown,
  ): PeerConnectionFacts {
    const owner = checkOwner(ownerValue);
    const endpoint = this.useEndpoint(connectionIdValue, owner);
    if (endpoint.closed) {
      throw new HttpPeerError("rejected", "closed-handle", "connection is closed");
    }
    if (!isHalfCloseDirection(directionValue)) {
      throw new HttpPeerError(
        "rejected",
        "unsupported-capability",
        "half-close direction must be read or write",
      );
    }
    if (directionValue === "write") {
      if (endpoint.write_closed) {
        throw new HttpPeerError("rejected", "invalid-state", "write side is already half-closed");
      }
      endpoint.write_closed = true;
      const peer = endpoint.peer_id === null ? undefined : this.endpoints.get(endpoint.peer_id);
      if (peer !== undefined && !peer.closed) {
        peer.peer_write_closed = true;
      }
    } else {
      if (endpoint.read_closed) {
        throw new HttpPeerError("rejected", "invalid-state", "read side is already half-closed");
      }
      endpoint.read_closed = true;
      // Buffered inbound bytes are discarded explicitly and counted; they
      // never silently become reads after a read half-close.
      endpoint.discarded_bytes += endpoint.inbound.length;
      endpoint.inbound = [];
    }
    return this.connectionFacts(owner, endpoint.id);
  }

  close(ownerValue: unknown, connectionIdValue: unknown): PeerConnectionCloseReceipt {
    const owner = checkOwner(ownerValue);
    const endpoint = this.useEndpoint(connectionIdValue, owner);
    if (endpoint.closed) {
      throw new HttpPeerError("rejected", "closed-handle", "connection is closed");
    }
    endpoint.closed = true;
    const peer = endpoint.peer_id === null ? undefined : this.endpoints.get(endpoint.peer_id);
    if (peer !== undefined && !peer.closed) {
      peer.peer_gone = true;
    }
    return Object.freeze({
      unread_bytes: endpoint.inbound.length,
      accepted_bytes: endpoint.accepted_bytes,
      consumed_bytes: endpoint.consumed_bytes,
      discarded_bytes: endpoint.discarded_bytes,
    });
  }

  listenerFacts(ownerValue: unknown, listenerIdValue: unknown): PeerListenerFacts {
    const owner = checkOwner(ownerValue);
    const listener = this.useListener(listenerIdValue, owner);
    return Object.freeze({
      id: listener.id,
      owner: listener.owner,
      destination: listener.destination,
      closed: listener.closed,
      pending_accepts: listener.queue.length,
      accepted_total: listener.accepted_total,
      stale_skipped: listener.stale_skipped,
      dropped_on_close: listener.dropped_on_close,
    });
  }

  connectionFacts(ownerValue: unknown, connectionIdValue: unknown): PeerConnectionFacts {
    const owner = checkOwner(ownerValue);
    const endpoint = this.useEndpoint(connectionIdValue, owner);
    return Object.freeze({
      id: endpoint.id,
      owner: endpoint.owner,
      destination: endpoint.destination,
      side: endpoint.side,
      peer_id: endpoint.peer_id,
      peer_gone: endpoint.peer_gone,
      read_closed: endpoint.read_closed,
      write_closed: endpoint.write_closed,
      peer_write_closed: endpoint.peer_write_closed,
      closed: endpoint.closed,
      buffered_bytes: endpoint.inbound.length,
      accepted_bytes: endpoint.accepted_bytes,
      consumed_bytes: endpoint.consumed_bytes,
      discarded_bytes: endpoint.discarded_bytes,
    });
  }

  dialFacts(ownerValue: unknown, dialIdValue: unknown): PeerDialFacts {
    const owner = checkOwner(ownerValue);
    const record = this.useDial(dialIdValue, owner);
    return Object.freeze({
      id: record.id,
      owner: record.owner,
      destination: record.destination,
      state: record.state,
      fail_reason: record.fail_reason,
      attempt_count: record.attempt_count,
      retry_count: record.attempt_count - 1,
      connection_id: record.connection_id,
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

  private useListener(idValue: unknown, owner: string): PeerListenerRecord {
    if (typeof idValue !== "string") {
      throw new HttpPeerError("rejected", "invalid-request", "listener id must be a string");
    }
    const listener = this.listeners.get(idValue);
    if (listener === undefined) {
      throw new HttpPeerError("rejected", "invalid-request", "unknown listener id");
    }
    if (listener.owner !== owner) {
      throw new HttpPeerError("rejected", "wrong-owner", "listener belongs to another owner");
    }
    return listener;
  }

  private useEndpoint(idValue: unknown, owner: string): PeerEndpoint {
    if (typeof idValue !== "string") {
      throw new HttpPeerError("rejected", "invalid-request", "connection id must be a string");
    }
    const endpoint = this.endpoints.get(idValue);
    if (endpoint === undefined) {
      throw new HttpPeerError("rejected", "invalid-request", "unknown connection id");
    }
    if (endpoint.owner !== owner) {
      throw new HttpPeerError("rejected", "wrong-owner", "connection belongs to another owner");
    }
    return endpoint;
  }

  private useDial(idValue: unknown, owner: string): PeerDialRecord {
    if (typeof idValue !== "string") {
      throw new HttpPeerError("rejected", "invalid-request", "dial id must be a string");
    }
    const record = this.dials.get(idValue);
    if (record === undefined) {
      throw new HttpPeerError("rejected", "invalid-request", "unknown dial id");
    }
    if (record.owner !== owner) {
      throw new HttpPeerError("rejected", "wrong-owner", "dial belongs to another owner");
    }
    return record;
  }

  private mintEndpoint(owner: string, destination: string, side: PeerSide): PeerEndpoint {
    const id = `c${this.nextEndpoint++}`;
    const endpoint: PeerEndpoint = {
      id,
      owner,
      destination,
      side,
      peer_id: null,
      peer_gone: false,
      read_closed: false,
      write_closed: false,
      peer_write_closed: false,
      closed: false,
      inbound: [],
      accepted_bytes: 0,
      consumed_bytes: 0,
      discarded_bytes: 0,
    };
    this.endpoints.set(id, endpoint);
    return endpoint;
  }

  private connectionHandle(endpoint: PeerEndpoint): PeerConnectionHandle {
    return Object.freeze({
      kind: "connection",
      id: endpoint.id,
      owner: endpoint.owner,
      destination: endpoint.destination,
      side: endpoint.side,
    });
  }

  // One synchronous dial attempt. Dials resolve only here, invoked solely by
  // dial/retryDial: nothing in the service completes a failed dial on its
  // own, so there is no implicit retry path.
  private attemptDial(record: PeerDialRecord): void {
    let open: PeerListenerRecord | undefined;
    let sawClosed = false;
    for (const listener of this.listeners.values()) {
      if (listener.destination !== record.destination) {
        continue;
      }
      if (listener.closed) {
        sawClosed = true;
        continue;
      }
      open = listener;
      break;
    }
    if (open === undefined) {
      record.state = "failed";
      record.fail_reason = sawClosed ? "listener-closed" : "no-listener";
      record.connection_id = null;
      return;
    }
    if (open.queue.length >= this.limits.max_pending_accepts) {
      record.state = "failed";
      record.fail_reason = "accept-queue-full";
      record.connection_id = null;
      return;
    }
    if (this.live_endpoint_count >= this.limits.max_connections) {
      record.state = "failed";
      record.fail_reason = "connection-cap";
      record.connection_id = null;
      return;
    }
    const client = this.mintEndpoint(record.owner, record.destination, "dialer");
    open.queue.push(client.id);
    record.state = "completed";
    record.fail_reason = null;
    record.connection_id = client.id;
  }

  private dialResult(record: PeerDialRecord): PeerDialResult {
    const dial: PeerDialHandle = Object.freeze({
      kind: "dial",
      id: record.id,
      owner: record.owner,
      destination: record.destination,
    });
    let connection: PeerConnectionHandle | null = null;
    if (record.connection_id !== null) {
      const endpoint = this.endpoints.get(record.connection_id);
      if (endpoint !== undefined) {
        connection = this.connectionHandle(endpoint);
      }
    }
    return Object.freeze({
      dial,
      state: record.state,
      fail_reason: record.fail_reason,
      connection,
      attempt_count: record.attempt_count,
      retry_count: record.attempt_count - 1,
    });
  }
}
