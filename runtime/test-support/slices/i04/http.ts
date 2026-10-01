// NT-I04 http test adapter. Owner-bound request build/upload/deliver/read/
// reissue operations over one in-memory HttpTestService with fixed declared
// origins and K20 limits. Handles are unforgeable branded objects bound to
// one adapter table: cross-table ids fail as stale, and ids are never
// reused. Facts and receipts are nominal records built by the adapter.
import { success, type AssertionContext, type Completion } from "../../../completion.ts";
import { array, record } from "../../../data.ts";
import { createDomainRuntime } from "../../../domain.ts";
import { ownerTag, type TestOwner } from "../../owner.ts";
import {
  checkHttpLimits,
  HttpTestService,
  type HttpRequestFacts,
  type HttpResponseFacts,
} from "./http-service.ts";
import {
  asInt,
  failService,
  failStale,
  headerRecord,
  intArray,
  optional,
  readByteArray,
  readHeaderArray,
  readInt,
} from "./support.ts";

const requestBrand = Symbol("can.http-peer.request");
export type RequestHandle = Readonly<{ readonly [requestBrand]: number }>;

const MAX_HANDLES = 1024;

// Declared origins and service bounds are fixed adapter constants (K20
// check values): the Can helper vocabulary must use these names.
const DECLARED_ORIGINS = ["origin-a", "origin-b"] as const;

export type HttpPeerHttpErrors = Readonly<{
  staleHandle: string;
  closedHandle: string;
  httpFault: string;
  readResult: string;
  some: string;
  none: string;
  header: string;
  requestFacts: string;
  redirect: string;
  responseFacts: string;
  headerReceipt: string;
  bodyChunkReceipt: string;
  requestCloseReceipt: string;
}>;

function isRequestHandle(value: unknown): value is RequestHandle {
  return (
    typeof value === "object" &&
    value !== null &&
    typeof (value as Record<symbol, unknown>)[requestBrand] === "number"
  );
}

export function createHttpPeerHttp(
  domain: ReturnType<typeof createDomainRuntime>,
  errors: HttpPeerHttpErrors,
  owner: TestOwner,
) {
  const service = new HttpTestService(
    [...DECLARED_ORIGINS],
    checkHttpLimits({
      max_requests: 16,
      max_headers: 8,
      max_header_bytes: 512,
      max_header_value_bytes: 128,
      max_body_bytes: 64,
      max_chunk_bytes: 16,
      max_reissues: 3,
      max_target_bytes: 128,
    }),
  );
  const requests = new Map<number, string>();
  let nextId = 0;

  function mint(serviceId: string): RequestHandle {
    if (requests.size >= MAX_HANDLES) throw new TypeError("http peer request table cap reached");
    nextId += 1;
    requests.set(nextId, serviceId);
    return Object.freeze({ [requestBrand]: nextId });
  }

  function redirectRecord(redirect: { status: number; location: string } | null): unknown {
    if (redirect === null) return null;
    return record(errors.redirect, [
      ["status", asInt(redirect.status)],
      ["location", redirect.location],
    ]);
  }

  function headersArray(headers: readonly (readonly [string, string])[]): unknown {
    return array(headers.map(([name, value]) => headerRecord(errors.header, name, value)));
  }

  function requestFactsRecord(facts: HttpRequestFacts): unknown {
    return record(errors.requestFacts, [
      ["id", facts.id],
      ["owner", facts.owner],
      ["origin", facts.origin],
      ["method", facts.method],
      ["target", facts.target],
      ["version", facts.version],
      ["state", facts.state],
      ["header_count", asInt(facts.header_count)],
      ["headers", headersArray(facts.headers)],
      ["header_bytes", asInt(facts.header_bytes)],
      ["headers_truncated", facts.headers_truncated],
      ["upload_accepted", asInt(facts.upload_accepted)],
      ["upload_complete", facts.upload_complete],
      ["upload_truncated", facts.upload_truncated],
      ["upload_length_mismatch", facts.upload_length_mismatch],
      ["response_delivered", facts.response_delivered],
      [
        "response_status",
        optional(
          errors.some,
          errors.none,
          facts.response_status === null ? null : asInt(facts.response_status),
        ),
      ],
      ["response_body_accepted", asInt(facts.response_body_accepted)],
      ["response_body_consumed", asInt(facts.response_body_consumed)],
      ["response_headers_truncated", facts.response_headers_truncated],
      ["response_body_truncated", facts.response_body_truncated],
      ["response_length_mismatch", facts.response_length_mismatch],
      ["redirect", optional(errors.some, errors.none, redirectRecord(facts.redirect))],
      ["redirect_incomplete", facts.redirect_incomplete],
      ["reissue_count", asInt(facts.reissue_count)],
      ["supersedes", optional(errors.some, errors.none, facts.supersedes)],
    ]);
  }

  function responseFactsRecord(facts: HttpResponseFacts): unknown {
    return record(errors.responseFacts, [
      ["status", asInt(facts.status)],
      ["header_count", asInt(facts.header_count)],
      ["headers", headersArray(facts.headers)],
      ["header_bytes", asInt(facts.header_bytes)],
      ["body_accepted", asInt(facts.body_accepted)],
      ["body_consumed", asInt(facts.body_consumed)],
      ["body_fully_consumed", facts.body_fully_consumed],
      ["length_mismatch", facts.length_mismatch],
      ["redirect", optional(errors.some, errors.none, redirectRecord(facts.redirect))],
      ["redirect_incomplete", facts.redirect_incomplete],
    ]);
  }

  function useOwner(handle: unknown, what: string): string | Completion<never> {
    const tag = ownerTag(handle);
    if (tag === null) throw new TypeError(`${what} needs an owner handle`);
    if (!owner.ownerLive(handle)) return failStale(domain, errors.staleHandle, tag);
    return tag;
  }

  function useRequest(handle: unknown): string | Completion<never> {
    if (!isRequestHandle(handle))
      throw new TypeError("http_peer request op needs a request handle");
    const id = handle[requestBrand];
    const serviceId = requests.get(id);
    if (serviceId === undefined) {
      return failStale(domain, errors.staleHandle, `request#${id}`);
    }
    return serviceId;
  }

  function failed(cell: string | Completion<never>): cell is Completion<never> {
    return typeof cell !== "string";
  }

  return Object.freeze({
    async openRequest(
      ownerHandle: unknown,
      origin: unknown,
      method: unknown,
      target: unknown,
      version: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<RequestHandle>> {
      const admitted = useOwner(ownerHandle, "http_peer::open_request");
      if (typeof admitted !== "string") return admitted;
      try {
        const handle = service.request(admitted, origin, method, target, version);
        return success(mint(handle.id));
      } catch (error) {
        return failService(domain, errors, errors.httpFault, admitted, error);
      }
    },

    async addHeader(
      ownerHandle: unknown,
      handle: unknown,
      name: unknown,
      value: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = useOwner(ownerHandle, "http_peer::add_header");
      if (typeof admitted !== "string") return admitted;
      const id = useRequest(handle);
      if (failed(id)) return id;
      try {
        const receipt = service.addHeader(admitted, id, name, value);
        return success(
          record(errors.headerReceipt, [
            ["header_count", asInt(receipt.header_count)],
            ["header_bytes", asInt(receipt.header_bytes)],
          ]),
        );
      } catch (error) {
        return failService(domain, errors, errors.httpFault, admitted, error);
      }
    },

    async sendBodyChunk(
      ownerHandle: unknown,
      handle: unknown,
      bytes: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = useOwner(ownerHandle, "http_peer::send_body_chunk");
      if (typeof admitted !== "string") return admitted;
      const id = useRequest(handle);
      if (failed(id)) return id;
      try {
        const receipt = service.sendBodyChunk(
          admitted,
          id,
          readByteArray(bytes, "http_peer::send_body_chunk bytes"),
        );
        return success(
          record(errors.bodyChunkReceipt, [
            ["accepted", asInt(receipt.accepted)],
            ["body_accepted_total", asInt(receipt.body_accepted_total)],
          ]),
        );
      } catch (error) {
        return failService(domain, errors, errors.httpFault, admitted, error);
      }
    },

    async endUpload(
      ownerHandle: unknown,
      handle: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = useOwner(ownerHandle, "http_peer::end_upload");
      if (typeof admitted !== "string") return admitted;
      const id = useRequest(handle);
      if (failed(id)) return id;
      try {
        return success(requestFactsRecord(service.endUpload(admitted, id)));
      } catch (error) {
        return failService(domain, errors, errors.httpFault, admitted, error);
      }
    },

    async deliverResponse(
      ownerHandle: unknown,
      handle: unknown,
      status: unknown,
      headers: unknown,
      body: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = useOwner(ownerHandle, "http_peer::deliver_response");
      if (typeof admitted !== "string") return admitted;
      const id = useRequest(handle);
      if (failed(id)) return id;
      try {
        const facts = service.deliverResponse(
          admitted,
          id,
          readInt(status, "http_peer::deliver_response status"),
          readHeaderArray(headers),
          readByteArray(body, "http_peer::deliver_response body"),
        );
        return success(responseFactsRecord(facts));
      } catch (error) {
        return failService(domain, errors, errors.httpFault, admitted, error);
      }
    },

    async readBodyChunk(
      ownerHandle: unknown,
      handle: unknown,
      maxBytes: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = useOwner(ownerHandle, "http_peer::read_body_chunk");
      if (typeof admitted !== "string") return admitted;
      const id = useRequest(handle);
      if (failed(id)) return id;
      try {
        const result = service.readBodyChunk(
          admitted,
          id,
          readInt(maxBytes, "http_peer::read_body_chunk max_bytes"),
        );
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
        return failService(domain, errors, errors.httpFault, admitted, error);
      }
    },

    async reissue(
      ownerHandle: unknown,
      handle: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<RequestHandle>> {
      const admitted = useOwner(ownerHandle, "http_peer::reissue");
      if (typeof admitted !== "string") return admitted;
      const id = useRequest(handle);
      if (failed(id)) return id;
      try {
        const request = service.reissue(admitted, id);
        return success(mint(request.id));
      } catch (error) {
        return failService(domain, errors, errors.httpFault, admitted, error);
      }
    },

    async closeRequest(
      ownerHandle: unknown,
      handle: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = useOwner(ownerHandle, "http_peer::close_request");
      if (typeof admitted !== "string") return admitted;
      const id = useRequest(handle);
      if (failed(id)) return id;
      try {
        const receipt = service.close(admitted, id);
        return success(
          record(errors.requestCloseReceipt, [
            ["upload_bytes", asInt(receipt.upload_bytes)],
            ["response_body_accepted", asInt(receipt.response_body_accepted)],
            ["response_body_consumed", asInt(receipt.response_body_consumed)],
            ["response_body_unread", asInt(receipt.response_body_unread)],
          ]),
        );
      } catch (error) {
        return failService(domain, errors, errors.httpFault, admitted, error);
      }
    },

    async readRequestFacts(
      ownerHandle: unknown,
      handle: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = useOwner(ownerHandle, "http_peer::read_request_facts");
      if (typeof admitted !== "string") return admitted;
      const id = useRequest(handle);
      if (failed(id)) return id;
      try {
        return success(requestFactsRecord(service.requestFacts(admitted, id)));
      } catch (error) {
        return failService(domain, errors, errors.httpFault, admitted, error);
      }
    },

    async readResponseFacts(
      ownerHandle: unknown,
      handle: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = useOwner(ownerHandle, "http_peer::read_response_facts");
      if (typeof admitted !== "string") return admitted;
      const id = useRequest(handle);
      if (failed(id)) return id;
      try {
        const facts = service.responseFacts(admitted, id);
        return success(
          optional(errors.some, errors.none, facts === null ? null : responseFactsRecord(facts)),
        );
      } catch (error) {
        return failService(domain, errors, errors.httpFault, admitted, error);
      }
    },
  });
}

export type HttpPeerHttp = ReturnType<typeof createHttpPeerHttp>;
