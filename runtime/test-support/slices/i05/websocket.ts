// NT-I05 websocket adapter. Owner-bound connect/send/deliver/poll/close
// operations over one in-memory WebSocketPeerService with fixed declared
// destinations and K21 limits, joining the shared $canHttpPeer
// contribution beside the I04 peer and http adapters. Connections
// travel as plain string ids (w1, w2, ...): K21 has a single flat
// id space with no cross-table interplay, and owner binding
// contains guessing (foreign ids fail wrong-owner into
// stale_handle, unknown ids fail invalid-request), so the I04
// branded-handle tables buy nothing here; the db/store
// families prove the shape. The admitted grant doubles as the
// service owner string (I04 precedent), so every service record
// attributes correctly. Failures map through the shared I04
// failService: closed-handle becomes test::closed_handle,
// wrong-owner becomes test::stale_handle, and all other K21
// kinds become http_peer::peer_fault{kind, reason} with the
// service message preserved verbatim (I04 precedent).
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
} from "../i04/support.ts";
import {
  checkWsLimits,
  WebSocketPeerService,
  type WsCloseFacts,
  type WsCloseReceipt,
  type WsConnectionFacts,
  type WsConnectionHandle,
  type WsDeliverReceipt,
  type WsEvent,
  type WsPollResult,
  type WsSendReceipt,
} from "./websocket-service.ts";

// Declared destinations and service bounds are fixed adapter constants
// (K21 check values): ws-a/ws-b cover two-destination flows.
const DECLARED_DESTINATIONS = ["ws-a", "ws-b"] as const;

export type HttpPeerWsErrors = Readonly<{
  staleHandle: string;
  closedHandle: string;
  peerFault: string;
  some: string;
  none: string;
  connectionHandle: string;
  closeFacts: string;
  event: string;
  connectionFacts: string;
  sendReceipt: string;
  deliverReceipt: string;
  pollResult: string;
  closeReceipt: string;
}>;

function readStr(value: unknown, what: string): string {
  if (typeof value !== "string") throw new TypeError(`${what} needs a str`);
  return value;
}

export function createHttpPeerWs(
  domain: ReturnType<typeof createDomainRuntime>,
  errors: HttpPeerWsErrors,
  owner: TestOwner,
) {
  const service = new WebSocketPeerService(
    [...DECLARED_DESTINATIONS],
    checkWsLimits({
      max_connections: 8,
      max_pending_events: 8,
      max_payload_bytes: 16,
      max_buffer_bytes: 64,
      max_close_reason_bytes: 32,
    }),
  );

  function handleRecord(handle: WsConnectionHandle): unknown {
    return record(errors.connectionHandle, [
      ["kind", handle.kind],
      ["id", handle.id],
      ["owner", handle.owner],
      ["destination", handle.destination],
    ]);
  }

  function closeFactsRecord(facts: WsCloseFacts): unknown {
    return record(errors.closeFacts, [
      ["origin", facts.origin],
      ["close_code", asInt(facts.close_code)],
      ["reason", facts.reason],
    ]);
  }

  function eventRecord(event: WsEvent): unknown {
    return record(errors.event, [
      ["opcode", event.opcode],
      ["payload", intArray(event.payload)],
    ]);
  }

  function connectionFactsRecord(facts: WsConnectionFacts): unknown {
    return record(errors.connectionFacts, [
      ["id", facts.id],
      ["owner", facts.owner],
      ["destination", facts.destination],
      ["state", facts.state],
      [
        "local_close",
        optional(
          errors.some,
          errors.none,
          facts.local_close === null ? null : closeFactsRecord(facts.local_close),
        ),
      ],
      [
        "remote_close",
        optional(
          errors.some,
          errors.none,
          facts.remote_close === null ? null : closeFactsRecord(facts.remote_close),
        ),
      ],
      ["sent_frames", asInt(facts.sent_frames)],
      ["sent_bytes", asInt(facts.sent_bytes)],
      ["delivered_events", asInt(facts.delivered_events)],
      ["consumed_events", asInt(facts.consumed_events)],
      ["pending_events", asInt(facts.pending_events)],
      ["pending_bytes", asInt(facts.pending_bytes)],
      ["dropped_events", asInt(facts.dropped_events)],
      ["dropped_bytes", asInt(facts.dropped_bytes)],
    ]);
  }

  function sendReceiptRecord(receipt: WsSendReceipt): unknown {
    return record(errors.sendReceipt, [
      ["opcode", receipt.opcode],
      ["accepted", asInt(receipt.accepted)],
      ["sent_frames_total", asInt(receipt.sent_frames_total)],
      ["sent_bytes_total", asInt(receipt.sent_bytes_total)],
    ]);
  }

  function deliverReceiptRecord(receipt: WsDeliverReceipt): unknown {
    return record(errors.deliverReceipt, [
      ["opcode", receipt.opcode],
      ["pending_events", asInt(receipt.pending_events)],
      ["pending_bytes", asInt(receipt.pending_bytes)],
      ["delivered_total", asInt(receipt.delivered_total)],
    ]);
  }

  function pollResultRecord(result: WsPollResult): unknown {
    return record(errors.pollResult, [
      [
        "event",
        optional(
          errors.some,
          errors.none,
          result.event === null ? null : eventRecord(result.event),
        ),
      ],
      ["pending_events", asInt(result.pending_events)],
      ["consumed_total", asInt(result.consumed_total)],
      ["delivered_total", asInt(result.delivered_total)],
    ]);
  }

  function closeReceiptRecord(receipt: WsCloseReceipt): unknown {
    return record(errors.closeReceipt, [
      ["origin", receipt.origin],
      ["close_code", asInt(receipt.close_code)],
      ["terminal", receipt.terminal],
      ["dropped_events", asInt(receipt.dropped_events)],
      ["dropped_bytes", asInt(receipt.dropped_bytes)],
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

  return Object.freeze({
    async connect(
      ownerHandle: unknown,
      destination: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = useOwner(ownerHandle, "http_peer::ws_connect");
      if (typeof admitted !== "string") return admitted;
      try {
        const scope = readStr(destination, "http_peer::ws_connect destination");
        return success(handleRecord(service.connect(admitted, scope)));
      } catch (error) {
        return failService(domain, errors, errors.peerFault, admitted, error);
      }
    },

    async send(
      ownerHandle: unknown,
      connection: unknown,
      opcode: unknown,
      payload: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = useOwner(ownerHandle, "http_peer::ws_send");
      if (typeof admitted !== "string") return admitted;
      try {
        const id = readStr(connection, "http_peer::ws_send connection");
        const frame = readStr(opcode, "http_peer::ws_send opcode");
        const bytes = readByteArray(payload, "http_peer::ws_send payload");
        return success(sendReceiptRecord(service.send(admitted, id, frame, bytes)));
      } catch (error) {
        return failService(domain, errors, errors.peerFault, admitted, error);
      }
    },

    async deliverEvent(
      ownerHandle: unknown,
      connection: unknown,
      opcode: unknown,
      payload: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = useOwner(ownerHandle, "http_peer::ws_deliver_event");
      if (typeof admitted !== "string") return admitted;
      try {
        const id = readStr(connection, "http_peer::ws_deliver_event connection");
        const frame = readStr(opcode, "http_peer::ws_deliver_event opcode");
        const bytes = readByteArray(payload, "http_peer::ws_deliver_event payload");
        return success(deliverReceiptRecord(service.deliverEvent(admitted, id, frame, bytes)));
      } catch (error) {
        return failService(domain, errors, errors.peerFault, admitted, error);
      }
    },

    async pollEvent(
      ownerHandle: unknown,
      connection: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = useOwner(ownerHandle, "http_peer::ws_poll_event");
      if (typeof admitted !== "string") return admitted;
      try {
        const id = readStr(connection, "http_peer::ws_poll_event connection");
        return success(pollResultRecord(service.pollEvent(admitted, id)));
      } catch (error) {
        return failService(domain, errors, errors.peerFault, admitted, error);
      }
    },

    async close(
      ownerHandle: unknown,
      connection: unknown,
      closeCode: unknown,
      reason: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = useOwner(ownerHandle, "http_peer::ws_close");
      if (typeof admitted !== "string") return admitted;
      try {
        const id = readStr(connection, "http_peer::ws_close connection");
        const code = readInt(closeCode, "http_peer::ws_close close_code");
        const text = readStr(reason, "http_peer::ws_close reason");
        return success(closeReceiptRecord(service.close(admitted, id, code, text)));
      } catch (error) {
        return failService(domain, errors, errors.peerFault, admitted, error);
      }
    },

    async deliverRemoteClose(
      ownerHandle: unknown,
      connection: unknown,
      closeCode: unknown,
      reason: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = useOwner(ownerHandle, "http_peer::ws_deliver_remote_close");
      if (typeof admitted !== "string") return admitted;
      try {
        const id = readStr(connection, "http_peer::ws_deliver_remote_close connection");
        const code = readInt(closeCode, "http_peer::ws_deliver_remote_close close_code");
        const text = readStr(reason, "http_peer::ws_deliver_remote_close reason");
        return success(closeReceiptRecord(service.deliverRemoteClose(admitted, id, code, text)));
      } catch (error) {
        return failService(domain, errors, errors.peerFault, admitted, error);
      }
    },

    async readConnectionFacts(
      ownerHandle: unknown,
      connection: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = useOwner(ownerHandle, "http_peer::ws_read_connection_facts");
      if (typeof admitted !== "string") return admitted;
      try {
        const id = readStr(connection, "http_peer::ws_read_connection_facts connection");
        return success(connectionFactsRecord(service.connectionFacts(admitted, id)));
      } catch (error) {
        return failService(domain, errors, errors.peerFault, admitted, error);
      }
    },
  });
}
