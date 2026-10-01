// NT-I04 http peer adapter. Owner-bound listener/dial/accept/read/write/
// half-close operations over one in-memory PeerListenerService with fixed
// declared destinations and K20 limits. Handles are unforgeable branded
// objects bound to one adapter table: cross-table ids fail as stale, and
// ids are never reused. Facts and receipts are nominal records built by
// the adapter; subsidiary handles (dial_result.connection) arrive minted.
import { success, type AssertionContext, type Completion } from "../../../completion.ts";
import { record } from "../../../data.ts";
import { createDomainRuntime } from "../../../domain.ts";
import { ownerTag, type TestOwner } from "../../owner.ts";
import {
  asInt,
  failService,
  failStale,
  intArray,
  optional,
  readByteArray,
  readInt,
} from "./support.ts";
import {
  checkPeerLimits,
  PeerListenerService,
  type PeerConnectionFacts,
  type PeerDialFacts,
  type PeerDialResult,
  type PeerListenerFacts,
} from "./peer-service.ts";

const listenerBrand = Symbol("can.http-peer.listener");
export type ListenerHandle = Readonly<{ readonly [listenerBrand]: number }>;

const connectionBrand = Symbol("can.http-peer.connection");
export type ConnectionHandle = Readonly<{ readonly [connectionBrand]: number }>;

const dialBrand = Symbol("can.http-peer.dial");
export type DialHandle = Readonly<{ readonly [dialBrand]: number }>;

const MAX_HANDLES = 1024;

// Declared destinations and service bounds are fixed adapter constants
// (K20 check values): the Can helper vocabulary must use these names.
// peer-a/origin-a are primary; peer-b/origin-b cover two-party flows.
const DECLARED_DESTINATIONS = ["peer-a", "peer-b"] as const;

export type HttpPeerPeerErrors = Readonly<{
  staleHandle: string;
  closedHandle: string;
  peerFault: string;
  some: string;
  none: string;
  listenerFacts: string;
  connectionFacts: string;
  dialFacts: string;
  dialResult: string;
  writeReceipt: string;
  readResult: string;
  connectionCloseReceipt: string;
  listenerCloseReceipt: string;
}>;

function isListenerHandle(value: unknown): value is ListenerHandle {
  return (
    typeof value === "object" &&
    value !== null &&
    typeof (value as Record<symbol, unknown>)[listenerBrand] === "number"
  );
}

function isConnectionHandle(value: unknown): value is ConnectionHandle {
  return (
    typeof value === "object" &&
    value !== null &&
    typeof (value as Record<symbol, unknown>)[connectionBrand] === "number"
  );
}

function isDialHandle(value: unknown): value is DialHandle {
  return (
    typeof value === "object" &&
    value !== null &&
    typeof (value as Record<symbol, unknown>)[dialBrand] === "number"
  );
}

export function createHttpPeerPeer(
  domain: ReturnType<typeof createDomainRuntime>,
  errors: HttpPeerPeerErrors,
  owner: TestOwner,
) {
  const service = new PeerListenerService(
    [...DECLARED_DESTINATIONS],
    checkPeerLimits({
      max_listeners: 4,
      max_connections: 16,
      max_pending_accepts: 8,
      max_dials: 16,
      max_buffer_bytes: 64,
      max_chunk_bytes: 16,
      max_explicit_retries: 3,
    }),
  );
  const listeners = new Map<number, string>();
  const connections = new Map<number, string>();
  const dials = new Map<number, string>();
  let nextId = 0;

  function mint(
    table: Map<number, string>,
    brand: symbol,
    kind: string,
    serviceId: string,
  ): Record<symbol, number> {
    if (table.size >= MAX_HANDLES) throw new TypeError(`http peer ${kind} table cap reached`);
    nextId += 1;
    table.set(nextId, serviceId);
    return Object.freeze({ [brand]: nextId });
  }

  function listenerFactsRecord(facts: PeerListenerFacts): unknown {
    return record(errors.listenerFacts, [
      ["id", facts.id],
      ["owner", facts.owner],
      ["destination", facts.destination],
      ["closed", facts.closed],
      ["pending_accepts", asInt(facts.pending_accepts)],
      ["accepted_total", asInt(facts.accepted_total)],
      ["stale_skipped", asInt(facts.stale_skipped)],
      ["dropped_on_close", asInt(facts.dropped_on_close)],
    ]);
  }

  function connectionFactsRecord(facts: PeerConnectionFacts): unknown {
    return record(errors.connectionFacts, [
      ["id", facts.id],
      ["owner", facts.owner],
      ["destination", facts.destination],
      ["side", facts.side],
      ["peer_id", optional(errors.some, errors.none, facts.peer_id)],
      ["peer_gone", facts.peer_gone],
      ["read_closed", facts.read_closed],
      ["write_closed", facts.write_closed],
      ["peer_write_closed", facts.peer_write_closed],
      ["closed", facts.closed],
      ["buffered_bytes", asInt(facts.buffered_bytes)],
      ["accepted_bytes", asInt(facts.accepted_bytes)],
      ["consumed_bytes", asInt(facts.consumed_bytes)],
      ["discarded_bytes", asInt(facts.discarded_bytes)],
    ]);
  }

  function dialFactsRecord(facts: PeerDialFacts): unknown {
    return record(errors.dialFacts, [
      ["id", facts.id],
      ["owner", facts.owner],
      ["destination", facts.destination],
      ["state", facts.state],
      ["fail_reason", optional(errors.some, errors.none, facts.fail_reason)],
      ["attempt_count", asInt(facts.attempt_count)],
      ["retry_count", asInt(facts.retry_count)],
      ["connection_id", optional(errors.some, errors.none, facts.connection_id)],
    ]);
  }

  function dialResultRecord(result: PeerDialResult, dialHandle?: DialHandle): unknown {
    const dial = dialHandle ?? (mint(dials, dialBrand, "dial", result.dial.id) as DialHandle);
    const connection =
      result.connection === null
        ? null
        : (mint(
            connections,
            connectionBrand,
            "connection",
            result.connection.id,
          ) as ConnectionHandle);
    return record(errors.dialResult, [
      ["dial", dial],
      ["state", result.state],
      ["fail_reason", optional(errors.some, errors.none, result.fail_reason)],
      ["connection", optional(errors.some, errors.none, connection)],
      ["attempt_count", asInt(result.attempt_count)],
      ["retry_count", asInt(result.retry_count)],
    ]);
  }

  // useOwner admits only live owner handles; the tag doubles as the
  // service owner string, so every service record attributes correctly.
  function useOwner(handle: unknown, what: string): string | Completion<never> {
    const tag = ownerTag(handle);
    if (tag === null) throw new TypeError(`${what} needs an owner handle`);
    if (!owner.ownerLive(handle)) return failStale(domain, errors.staleHandle, tag);
    return tag;
  }

  function useListener(handle: unknown): string | Completion<never> {
    if (!isListenerHandle(handle))
      throw new TypeError("http_peer listener op needs a listener handle");
    const id = handle[listenerBrand];
    const serviceId = listeners.get(id);
    if (serviceId === undefined) {
      return failStale(domain, errors.staleHandle, `listener#${id}`);
    }
    return serviceId;
  }

  function useConnection(handle: unknown): string | Completion<never> {
    if (!isConnectionHandle(handle))
      throw new TypeError("http_peer connection op needs a connection handle");
    const id = handle[connectionBrand];
    const serviceId = connections.get(id);
    if (serviceId === undefined) {
      return failStale(domain, errors.staleHandle, `connection#${id}`);
    }
    return serviceId;
  }

  function useDial(handle: unknown): string | Completion<never> {
    if (!isDialHandle(handle)) throw new TypeError("http_peer dial op needs a dial handle");
    const id = handle[dialBrand];
    const serviceId = dials.get(id);
    if (serviceId === undefined) {
      return failStale(domain, errors.staleHandle, `dial#${id}`);
    }
    return serviceId;
  }

  function failed(cell: string | Completion<never>): cell is Completion<never> {
    return typeof cell !== "string";
  }

  return Object.freeze({
    async openListener(
      ownerHandle: unknown,
      destination: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<ListenerHandle>> {
      const admitted = useOwner(ownerHandle, "http_peer::open_listener");
      if (typeof admitted !== "string") return admitted;
      try {
        const handle = service.openListener(admitted, destination);
        return success(mint(listeners, listenerBrand, "listener", handle.id) as ListenerHandle);
      } catch (error) {
        return failService(domain, errors, errors.peerFault, admitted, error);
      }
    },

    async closeListener(
      ownerHandle: unknown,
      handle: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = useOwner(ownerHandle, "http_peer::close_listener");
      if (typeof admitted !== "string") return admitted;
      const id = useListener(handle);
      if (failed(id)) return id;
      try {
        const receipt = service.closeListener(admitted, id);
        return success(
          record(errors.listenerCloseReceipt, [
            ["pending_accepts_dropped", asInt(receipt.pending_accepts_dropped)],
            ["accepted_total", asInt(receipt.accepted_total)],
            ["stale_skipped", asInt(receipt.stale_skipped)],
          ]),
        );
      } catch (error) {
        return failService(domain, errors, errors.peerFault, admitted, error);
      }
    },

    async dialPeer(
      ownerHandle: unknown,
      destination: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = useOwner(ownerHandle, "http_peer::dial_peer");
      if (typeof admitted !== "string") return admitted;
      try {
        return success(dialResultRecord(service.dial(admitted, destination)));
      } catch (error) {
        return failService(domain, errors, errors.peerFault, admitted, error);
      }
    },

    async retryDial(
      ownerHandle: unknown,
      handle: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = useOwner(ownerHandle, "http_peer::retry_dial");
      if (typeof admitted !== "string") return admitted;
      const id = useDial(handle);
      if (failed(id)) return id;
      try {
        return success(dialResultRecord(service.retryDial(admitted, id), handle as DialHandle));
      } catch (error) {
        return failService(domain, errors, errors.peerFault, admitted, error);
      }
    },

    async accept(
      ownerHandle: unknown,
      handle: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<ConnectionHandle>> {
      const admitted = useOwner(ownerHandle, "http_peer::accept");
      if (typeof admitted !== "string") return admitted;
      const id = useListener(handle);
      if (failed(id)) return id;
      try {
        const endpoint = service.accept(admitted, id);
        return success(
          mint(connections, connectionBrand, "connection", endpoint.id) as ConnectionHandle,
        );
      } catch (error) {
        return failService(domain, errors, errors.peerFault, admitted, error);
      }
    },

    async write(
      ownerHandle: unknown,
      handle: unknown,
      bytes: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = useOwner(ownerHandle, "http_peer::write");
      if (typeof admitted !== "string") return admitted;
      const id = useConnection(handle);
      if (failed(id)) return id;
      try {
        const receipt = service.write(admitted, id, readByteArray(bytes, "http_peer::write bytes"));
        return success(
          record(errors.writeReceipt, [
            ["accepted", asInt(receipt.accepted)],
            ["peer_buffered", asInt(receipt.peer_buffered)],
            ["peer_accepted_total", asInt(receipt.peer_accepted_total)],
          ]),
        );
      } catch (error) {
        return failService(domain, errors, errors.peerFault, admitted, error);
      }
    },

    async read(
      ownerHandle: unknown,
      handle: unknown,
      maxBytes: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = useOwner(ownerHandle, "http_peer::read");
      if (typeof admitted !== "string") return admitted;
      const id = useConnection(handle);
      if (failed(id)) return id;
      try {
        const result = service.read(admitted, id, readInt(maxBytes, "http_peer::read max_bytes"));
        return success(
          record(errors.readResult, [
            ["bytes", intArray(result.bytes)],
            ["eof", result.eof],
            ["truncated", result.truncated],
            ["consumed_total", asInt(result.consumed_total)],
            ["accepted_total", asInt(result.accepted_total)],
          ]),
        );
      } catch (error) {
        return failService(domain, errors, errors.peerFault, admitted, error);
      }
    },

    async halfClose(
      ownerHandle: unknown,
      handle: unknown,
      direction: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = useOwner(ownerHandle, "http_peer::half_close");
      if (typeof admitted !== "string") return admitted;
      const id = useConnection(handle);
      if (failed(id)) return id;
      try {
        return success(connectionFactsRecord(service.halfClose(admitted, id, direction)));
      } catch (error) {
        return failService(domain, errors, errors.peerFault, admitted, error);
      }
    },

    async closeConnection(
      ownerHandle: unknown,
      handle: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = useOwner(ownerHandle, "http_peer::close_connection");
      if (typeof admitted !== "string") return admitted;
      const id = useConnection(handle);
      if (failed(id)) return id;
      try {
        const receipt = service.close(admitted, id);
        return success(
          record(errors.connectionCloseReceipt, [
            ["unread_bytes", asInt(receipt.unread_bytes)],
            ["accepted_bytes", asInt(receipt.accepted_bytes)],
            ["consumed_bytes", asInt(receipt.consumed_bytes)],
            ["discarded_bytes", asInt(receipt.discarded_bytes)],
          ]),
        );
      } catch (error) {
        return failService(domain, errors, errors.peerFault, admitted, error);
      }
    },

    async readListenerFacts(
      ownerHandle: unknown,
      handle: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = useOwner(ownerHandle, "http_peer::read_listener_facts");
      if (typeof admitted !== "string") return admitted;
      const id = useListener(handle);
      if (failed(id)) return id;
      try {
        return success(listenerFactsRecord(service.listenerFacts(admitted, id)));
      } catch (error) {
        return failService(domain, errors, errors.peerFault, admitted, error);
      }
    },

    async readConnectionFacts(
      ownerHandle: unknown,
      handle: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = useOwner(ownerHandle, "http_peer::read_connection_facts");
      if (typeof admitted !== "string") return admitted;
      const id = useConnection(handle);
      if (failed(id)) return id;
      try {
        return success(connectionFactsRecord(service.connectionFacts(admitted, id)));
      } catch (error) {
        return failService(domain, errors, errors.peerFault, admitted, error);
      }
    },

    async readDialFacts(
      ownerHandle: unknown,
      handle: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = useOwner(ownerHandle, "http_peer::read_dial_facts");
      if (typeof admitted !== "string") return admitted;
      const id = useDial(handle);
      if (failed(id)) return id;
      try {
        return success(dialFactsRecord(service.dialFacts(admitted, id)));
      } catch (error) {
        return failService(domain, errors, errors.peerFault, admitted, error);
      }
    },
  });
}

export type HttpPeerPeer = ReturnType<typeof createHttpPeerPeer>;
