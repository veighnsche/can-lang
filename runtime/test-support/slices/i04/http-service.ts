// NT-I04 behavioral port of tools/runtime/test-services/http-peer/http.ts (K20, accepted). The runtime ship boundary
// forbids runtime/ from importing tools/ (tools ship as host-executed files,
// never as bundle modules), so the I04 slice carries its own copy of the
// doubles. Service logic below is unchanged from K20; only this header and
// the peer import differ.

// K20: typed in-memory HTTP pending upload/header/body operations.
//
// In-memory doubles only: no fetch, no sockets, no network, no timers. The
// caller builds a request (headers then body chunks), ends the upload, and
// the test double injects the server response explicitly via
// deliverResponse. Responses never arrive on their own and are never
// followed or retried by the service.
//
// Structural guarantees:
// - Finite bounds: declared origins, live requests, headers, header bytes,
//   body bytes, chunk bytes, explicit reissues.
// - No implicit retry: truncated/failed responses never create follow-up
//   requests; only an explicit reissue mints a new request, counted in
//   reissue_count and linked via supersedes.
// - No implicit redirect: a 302 surfaces a redirect fact; the service never
//   follows it (no code path mints a request from a response).
// - Accepted-vs-consumed separation: response bytes admitted by delivery
//   (body_accepted) and bytes handed out by reads (body_consumed) are
//   independent counters.
// - Repeated headers preserved in order, never folded; when a header
//   repeats, the LAST occurrence governs typed interpretation (declared
//   content-length, redirect location) while every occurrence stays in facts.
// - EOF explicit: reads before delivery report eof:false; eof is true only
//   after the delivered body is fully consumed.
// - Partial state explicit: over-cap headers/bodies set sticky truncated
//   facts, declared-length mismatches set length_mismatch facts, and a 302
//   without location sets redirect_incomplete. Short reads report
//   truncated:true with the remainder still buffered.

import {
  checkBytes,
  checkDeclaredSet,
  checkOwner,
  checkPositiveInt,
  HttpPeerError,
} from "./peer-service.ts";

// ---------------------------------------------------------------------------
// Closed vocabularies
// ---------------------------------------------------------------------------

export const HTTP_METHODS = ["GET", "POST"] as const;
export type HttpMethod = (typeof HTTP_METHODS)[number];

export const HTTP_VERSIONS = ["1.1"] as const;
export type HttpVersion = (typeof HTTP_VERSIONS)[number];

export const HTTP_HEADER_NAMES = [
  "host",
  "content-type",
  "content-length",
  "location",
  "set-cookie",
  "cookie",
  "x-trace",
  "x-checksum",
] as const;
export type HttpHeaderName = (typeof HTTP_HEADER_NAMES)[number];

export const HTTP_STATUSES = [200, 201, 302, 400, 404, 500] as const;
export type HttpStatus = (typeof HTTP_STATUSES)[number];

export const HTTP_REQUEST_STATES = [
  "upload_open",
  "upload_complete",
  "responded",
  "closed",
] as const;
export type HttpRequestState = (typeof HTTP_REQUEST_STATES)[number];

export const HTTP_REDIRECT_STATUS: HttpStatus = 302;

function isMethod(value: unknown): value is HttpMethod {
  return typeof value === "string" && (HTTP_METHODS as readonly string[]).includes(value);
}

function isVersion(value: unknown): value is HttpVersion {
  return typeof value === "string" && (HTTP_VERSIONS as readonly string[]).includes(value);
}

function isHeaderName(value: unknown): value is HttpHeaderName {
  return typeof value === "string" && (HTTP_HEADER_NAMES as readonly string[]).includes(value);
}

function isStatus(value: unknown): value is HttpStatus {
  return typeof value === "number" && (HTTP_STATUSES as readonly number[]).includes(value);
}

const PRINTABLE_ASCII = /^[\x20-\x7E]*$/;

function checkHeaderValue(value: unknown, maxBytes: number): string {
  if (typeof value !== "string") {
    throw new HttpPeerError("rejected", "invalid-request", "header value must be a string");
  }
  if (!PRINTABLE_ASCII.test(value)) {
    throw new HttpPeerError("rejected", "invalid-request", "header value must be printable ASCII");
  }
  if (value.length > maxBytes) {
    throw new HttpPeerError(
      "rejected",
      "invalid-request",
      `header value exceeds the ${maxBytes}-byte value cap`,
    );
  }
  return value;
}

// Deterministic header-block byte model: name + ": " + value + CRLF.
function headerBlockBytes(name: string, value: string): number {
  return name.length + value.length + 4;
}

function lastHeader(headers: readonly (readonly [string, string])[], name: string): string | null {
  let found: string | null = null;
  for (const [key, value] of headers) {
    if (key === name) {
      found = value;
    }
  }
  return found;
}

function parseDeclaredLength(raw: string, what: string): number {
  if (!/^(0|[1-9][0-9]*)$/.test(raw)) {
    throw new HttpPeerError("rejected", "invalid-request", `${what} content-length is malformed`);
  }
  const parsed = Number.parseInt(raw, 10);
  if (!Number.isSafeInteger(parsed)) {
    throw new HttpPeerError("rejected", "invalid-request", `${what} content-length is malformed`);
  }
  return parsed;
}

// ---------------------------------------------------------------------------
// Limits
// ---------------------------------------------------------------------------

export type HttpLimits = Readonly<{
  max_requests: number;
  max_headers: number;
  max_header_bytes: number;
  max_header_value_bytes: number;
  max_body_bytes: number;
  max_chunk_bytes: number;
  max_reissues: number;
  max_target_bytes: number;
}>;

export const HTTP_LIMIT_KEYS = [
  "max_requests",
  "max_headers",
  "max_header_bytes",
  "max_header_value_bytes",
  "max_body_bytes",
  "max_chunk_bytes",
  "max_reissues",
  "max_target_bytes",
] as const;

const HTTP_HARD_CEILINGS: Record<(typeof HTTP_LIMIT_KEYS)[number], number> = {
  max_requests: 1024,
  max_headers: 128,
  max_header_bytes: 1 << 16,
  max_header_value_bytes: 1 << 13,
  max_body_bytes: 1 << 20,
  max_chunk_bytes: 1 << 16,
  max_reissues: 16,
  max_target_bytes: 2048,
};

export function checkHttpLimits(value: unknown): HttpLimits {
  if (typeof value !== "object" || value === null || Array.isArray(value)) {
    throw new HttpPeerError("rejected", "invalid-request", "http limits must be a record");
  }
  const record = value as Record<string, unknown>;
  const out: Record<string, number> = {};
  for (const key of HTTP_LIMIT_KEYS) {
    // max_reissues may be 0 (explicit reissues disabled by bounds); every
    // other bound must admit at least one.
    const min = key === "max_reissues" ? 0 : 1;
    out[key] = checkPositiveInt(record[key], `http limit ${key}`, min, HTTP_HARD_CEILINGS[key]);
  }
  const chunk = out["max_chunk_bytes"] ?? 1;
  const body = out["max_body_bytes"] ?? 1;
  if (chunk > body) {
    throw new HttpPeerError(
      "rejected",
      "invalid-request",
      "http limit max_chunk_bytes must not exceed max_body_bytes",
    );
  }
  const valueCap = out["max_header_value_bytes"] ?? 1;
  const blockCap = out["max_header_bytes"] ?? 1;
  if (valueCap > blockCap) {
    throw new HttpPeerError(
      "rejected",
      "invalid-request",
      "http limit max_header_value_bytes must not exceed max_header_bytes",
    );
  }
  return Object.freeze({
    max_requests: out["max_requests"] ?? 1,
    max_headers: out["max_headers"] ?? 1,
    max_header_bytes: blockCap,
    max_header_value_bytes: valueCap,
    max_body_bytes: body,
    max_chunk_bytes: chunk,
    max_reissues: out["max_reissues"] ?? 0,
    max_target_bytes: out["max_target_bytes"] ?? 1,
  });
}

// ---------------------------------------------------------------------------
// Handles, facts, receipts (all frozen)
// ---------------------------------------------------------------------------

export type HttpRequestHandle = Readonly<{
  kind: "http-request";
  id: string;
  owner: string;
  origin: string;
}>;

export type HttpHeader = readonly [name: string, value: string];

export type HttpRedirect = Readonly<{
  status: HttpStatus;
  location: string;
}>;

export type HttpRequestFacts = Readonly<{
  id: string;
  owner: string;
  origin: string;
  method: HttpMethod;
  target: string;
  version: HttpVersion;
  state: HttpRequestState;
  header_count: number;
  headers: readonly HttpHeader[];
  header_bytes: number;
  headers_truncated: boolean;
  upload_accepted: number;
  upload_complete: boolean;
  upload_truncated: boolean;
  upload_length_mismatch: boolean;
  response_delivered: boolean;
  response_status: HttpStatus | null;
  response_body_accepted: number;
  response_body_consumed: number;
  response_headers_truncated: boolean;
  response_body_truncated: boolean;
  response_length_mismatch: boolean;
  redirect: HttpRedirect | null;
  redirect_incomplete: boolean;
  reissue_count: number;
  supersedes: string | null;
}>;

export type HttpResponseFacts = Readonly<{
  status: HttpStatus;
  header_count: number;
  headers: readonly HttpHeader[];
  header_bytes: number;
  body_accepted: number;
  body_consumed: number;
  body_fully_consumed: boolean;
  length_mismatch: boolean;
  redirect: HttpRedirect | null;
  redirect_incomplete: boolean;
}>;

export type HttpHeaderReceipt = Readonly<{
  header_count: number;
  header_bytes: number;
}>;

export type HttpBodyChunkReceipt = Readonly<{
  accepted: number;
  body_accepted_total: number;
}>;

export type HttpReadResult = Readonly<{
  bytes: readonly number[];
  eof: boolean;
  truncated: boolean;
  consumed_total: number;
  accepted_total: number;
}>;

export type HttpCloseReceipt = Readonly<{
  upload_bytes: number;
  response_body_accepted: number;
  response_body_consumed: number;
  response_body_unread: number;
}>;

// ---------------------------------------------------------------------------
// Internal records (mutable; never escape)
// ---------------------------------------------------------------------------

type HttpResponseRecord = {
  status: HttpStatus;
  headers: [string, string][];
  header_bytes: number;
  body: number[];
  body_accepted: number;
  body_consumed: number;
  length_mismatch: boolean;
  redirect: HttpRedirect | null;
  redirect_incomplete: boolean;
};

type HttpRequestRecord = {
  id: string;
  owner: string;
  origin: string;
  method: HttpMethod;
  target: string;
  version: HttpVersion;
  state: HttpRequestState;
  headers: [string, string][];
  header_bytes: number;
  headers_truncated: boolean;
  upload: number[];
  upload_accepted: number;
  upload_truncated: boolean;
  upload_length_mismatch: boolean;
  response: HttpResponseRecord | null;
  response_headers_truncated: boolean;
  response_body_truncated: boolean;
  reissue_count: number;
  supersedes: string | null;
};

// ---------------------------------------------------------------------------
// Service
// ---------------------------------------------------------------------------

export class HttpTestService {
  private readonly origins: readonly string[];
  private readonly limits: HttpLimits;
  private readonly requests = new Map<string, HttpRequestRecord>();
  private nextRequest = 1;

  constructor(declaredOrigins: unknown, limits: HttpLimits) {
    this.origins = checkDeclaredSet(declaredOrigins, "http origins");
    this.limits = limits;
  }

  get live_request_count(): number {
    let count = 0;
    for (const request of this.requests.values()) {
      if (request.state !== "closed") {
        count += 1;
      }
    }
    return count;
  }

  get request_count(): number {
    return this.requests.size;
  }

  request(
    ownerValue: unknown,
    originValue: unknown,
    methodValue: unknown,
    targetValue: unknown,
    versionValue: unknown,
  ): HttpRequestHandle {
    const owner = checkOwner(ownerValue);
    const origin = this.resolveOrigin(originValue);
    if (!isMethod(methodValue)) {
      throw new HttpPeerError("rejected", "unsupported-capability", "http method is not supported");
    }
    if (!isVersion(versionValue)) {
      throw new HttpPeerError(
        "rejected",
        "unsupported-capability",
        "http version is not supported",
      );
    }
    const target = this.checkTarget(targetValue);
    if (this.requests.size >= this.limits.max_requests) {
      throw new HttpPeerError("rejected", "resource-limit", "request table full");
    }
    const id = `q${this.nextRequest++}`;
    this.requests.set(id, {
      id,
      owner,
      origin,
      method: methodValue,
      target,
      version: versionValue,
      state: "upload_open",
      headers: [],
      header_bytes: 0,
      headers_truncated: false,
      upload: [],
      upload_accepted: 0,
      upload_truncated: false,
      upload_length_mismatch: false,
      response: null,
      response_headers_truncated: false,
      response_body_truncated: false,
      reissue_count: 0,
      supersedes: null,
    });
    return Object.freeze({ kind: "http-request", id, owner, origin });
  }

  addHeader(
    ownerValue: unknown,
    requestIdValue: unknown,
    nameValue: unknown,
    valueValue: unknown,
  ): HttpHeaderReceipt {
    const owner = checkOwner(ownerValue);
    const request = this.useOpenUpload(requestIdValue, owner);
    if (!isHeaderName(nameValue)) {
      throw new HttpPeerError(
        "rejected",
        "unsupported-capability",
        "http header name is not known",
      );
    }
    const value = checkHeaderValue(valueValue, this.limits.max_header_value_bytes);
    const cost = headerBlockBytes(nameValue, value);
    if (
      request.headers.length >= this.limits.max_headers ||
      request.header_bytes + cost > this.limits.max_header_bytes
    ) {
      // Over-cap headers are observable partial state, not a silent drop:
      // the add rejects AND the sticky truncated fact records the loss.
      request.headers_truncated = true;
      throw new HttpPeerError("rejected", "header-limit", "request header block is full");
    }
    request.headers.push([nameValue, value]);
    request.header_bytes += cost;
    return Object.freeze({
      header_count: request.headers.length,
      header_bytes: request.header_bytes,
    });
  }

  sendBodyChunk(
    ownerValue: unknown,
    requestIdValue: unknown,
    bytesValue: unknown,
  ): HttpBodyChunkReceipt {
    const owner = checkOwner(ownerValue);
    const request = this.useOpenUpload(requestIdValue, owner);
    if (request.method !== "POST") {
      throw new HttpPeerError("rejected", "invalid-state", "only POST requests carry a body");
    }
    const bytes = checkBytes(bytesValue, "upload chunk", this.limits.max_chunk_bytes);
    if (request.upload_accepted + bytes.length > this.limits.max_body_bytes) {
      // Over-cap uploads are observable partial state: the chunk rejects
      // AND the sticky truncated fact records the loss.
      request.upload_truncated = true;
      throw new HttpPeerError("rejected", "body-limit", "upload would exceed the body cap");
    }
    for (const byte of bytes) {
      request.upload.push(byte);
    }
    request.upload_accepted += bytes.length;
    return Object.freeze({ accepted: bytes.length, body_accepted_total: request.upload_accepted });
  }

  endUpload(ownerValue: unknown, requestIdValue: unknown): HttpRequestFacts {
    const owner = checkOwner(ownerValue);
    const request = this.useOpenUpload(requestIdValue, owner);
    const declared = lastHeader(request.headers, "content-length");
    if (declared !== null) {
      const expected = parseDeclaredLength(declared, "upload");
      // A short or long upload completes with an explicit mismatch fact;
      // the partial state is observable, never swallowed.
      request.upload_length_mismatch = expected !== request.upload_accepted;
    }
    request.state = "upload_complete";
    return this.snapshot(request);
  }

  deliverResponse(
    ownerValue: unknown,
    requestIdValue: unknown,
    statusValue: unknown,
    headersValue: unknown,
    bodyValue: unknown,
  ): HttpResponseFacts {
    const owner = checkOwner(ownerValue);
    const request = this.useRequest(requestIdValue, owner);
    if (request.state === "closed") {
      throw new HttpPeerError("rejected", "closed-handle", "request is closed");
    }
    if (request.state === "upload_open") {
      throw new HttpPeerError("rejected", "invalid-state", "end the upload before delivering");
    }
    if (request.state === "responded") {
      throw new HttpPeerError("rejected", "invalid-state", "response already delivered");
    }
    if (!isStatus(statusValue)) {
      throw new HttpPeerError("rejected", "unsupported-capability", "http status is not supported");
    }
    const headers = this.checkResponseHeaders(headersValue, request);
    const body = this.checkResponseBody(bodyValue, request);
    const declared = lastHeader(headers, "content-length");
    let lengthMismatch = false;
    if (declared !== null) {
      // Malformed content-length rejects before the response is stored.
      lengthMismatch = parseDeclaredLength(declared, "response") !== body.length;
    }
    let redirect: HttpRedirect | null = null;
    let redirectIncomplete = false;
    if (statusValue === HTTP_REDIRECT_STATUS) {
      const location = lastHeader(headers, "location");
      if (location === null) {
        redirectIncomplete = true;
      } else {
        redirect = Object.freeze({ status: statusValue, location });
      }
    }
    let headerBytes = 0;
    for (const [name, value] of headers) {
      headerBytes += headerBlockBytes(name, value);
    }
    request.response = {
      status: statusValue,
      headers: headers.map((pair): [string, string] => [pair[0], pair[1]]),
      header_bytes: headerBytes,
      body: [...body],
      body_accepted: body.length,
      body_consumed: 0,
      length_mismatch: lengthMismatch,
      redirect,
      redirect_incomplete: redirectIncomplete,
    };
    request.state = "responded";
    return this.responseSnapshot(request.response);
  }

  readBodyChunk(
    ownerValue: unknown,
    requestIdValue: unknown,
    maxBytesValue: unknown,
  ): HttpReadResult {
    const owner = checkOwner(ownerValue);
    const request = this.useRequest(requestIdValue, owner);
    if (request.state === "closed") {
      throw new HttpPeerError("rejected", "closed-handle", "request is closed");
    }
    const max = checkPositiveInt(maxBytesValue, "read max_bytes", 1, this.limits.max_chunk_bytes);
    const response = request.response;
    if (response === null) {
      // No response yet: empty with eof:false. EOF must never be implied by
      // an empty read.
      return Object.freeze({
        bytes: Object.freeze([]),
        eof: false,
        truncated: false,
        consumed_total: 0,
        accepted_total: 0,
      });
    }
    const take = Math.min(max, response.body.length);
    const out = response.body.splice(0, take);
    response.body_consumed += out.length;
    const drained = response.body.length === 0;
    return Object.freeze({
      bytes: Object.freeze(out),
      eof: drained,
      truncated: !drained,
      consumed_total: response.body_consumed,
      accepted_total: response.body_accepted,
    });
  }

  // The ONLY path from a response to a new request. Redirects and retries
  // never happen implicitly: the caller reissues explicitly, headers and
  // body are never copied over, and the chain is counted and linked.
  reissue(ownerValue: unknown, requestIdValue: unknown): HttpRequestHandle {
    const owner = checkOwner(ownerValue);
    const prior = this.useRequest(requestIdValue, owner);
    if (prior.state === "closed") {
      throw new HttpPeerError("rejected", "closed-handle", "request is closed");
    }
    if (prior.state === "upload_open") {
      throw new HttpPeerError("rejected", "invalid-state", "end the upload before reissuing");
    }
    if (prior.reissue_count >= this.limits.max_reissues) {
      throw new HttpPeerError("rejected", "resource-limit", "explicit reissue budget exhausted");
    }
    if (this.requests.size >= this.limits.max_requests) {
      throw new HttpPeerError("rejected", "resource-limit", "request table full");
    }
    const id = `q${this.nextRequest++}`;
    this.requests.set(id, {
      id,
      owner: prior.owner,
      origin: prior.origin,
      method: prior.method,
      target: prior.target,
      version: prior.version,
      state: "upload_open",
      headers: [],
      header_bytes: 0,
      headers_truncated: false,
      upload: [],
      upload_accepted: 0,
      upload_truncated: false,
      upload_length_mismatch: false,
      response: null,
      response_headers_truncated: false,
      response_body_truncated: false,
      reissue_count: prior.reissue_count + 1,
      supersedes: prior.id,
    });
    return Object.freeze({ kind: "http-request", id, owner: prior.owner, origin: prior.origin });
  }

  close(ownerValue: unknown, requestIdValue: unknown): HttpCloseReceipt {
    const owner = checkOwner(ownerValue);
    const request = this.useRequest(requestIdValue, owner);
    if (request.state === "closed") {
      throw new HttpPeerError("rejected", "closed-handle", "request is closed");
    }
    request.state = "closed";
    const response = request.response;
    return Object.freeze({
      upload_bytes: request.upload_accepted,
      response_body_accepted: response === null ? 0 : response.body_accepted,
      response_body_consumed: response === null ? 0 : response.body_consumed,
      response_body_unread: response === null ? 0 : response.body.length,
    });
  }

  requestFacts(ownerValue: unknown, requestIdValue: unknown): HttpRequestFacts {
    const owner = checkOwner(ownerValue);
    return this.snapshot(this.useRequest(requestIdValue, owner));
  }

  responseFacts(ownerValue: unknown, requestIdValue: unknown): HttpResponseFacts | null {
    const owner = checkOwner(ownerValue);
    const request = this.useRequest(requestIdValue, owner);
    if (request.response === null) {
      return null;
    }
    return this.responseSnapshot(request.response);
  }

  private resolveOrigin(value: unknown): string {
    if (typeof value !== "string") {
      throw new HttpPeerError("rejected", "invalid-request", "origin must be a string");
    }
    if (!this.origins.includes(value)) {
      throw new HttpPeerError("rejected", "foreign-destination", "origin is not declared");
    }
    return value;
  }

  private checkTarget(value: unknown): string {
    if (typeof value !== "string") {
      throw new HttpPeerError("rejected", "invalid-request", "target must be a string");
    }
    if (!value.startsWith("/") || value.length > this.limits.max_target_bytes) {
      throw new HttpPeerError(
        "rejected",
        "invalid-request",
        "target must start with / and fit the target cap",
      );
    }
    if (!PRINTABLE_ASCII.test(value)) {
      throw new HttpPeerError("rejected", "invalid-request", "target must be printable ASCII");
    }
    return value;
  }

  private useRequest(idValue: unknown, owner: string): HttpRequestRecord {
    if (typeof idValue !== "string") {
      throw new HttpPeerError("rejected", "invalid-request", "request id must be a string");
    }
    const request = this.requests.get(idValue);
    if (request === undefined) {
      throw new HttpPeerError("rejected", "invalid-request", "unknown request id");
    }
    if (request.owner !== owner) {
      throw new HttpPeerError("rejected", "wrong-owner", "request belongs to another owner");
    }
    return request;
  }

  private useOpenUpload(idValue: unknown, owner: string): HttpRequestRecord {
    const request = this.useRequest(idValue, owner);
    if (request.state === "closed") {
      throw new HttpPeerError("rejected", "closed-handle", "request is closed");
    }
    if (request.state !== "upload_open") {
      throw new HttpPeerError("rejected", "invalid-state", "upload is already complete");
    }
    return request;
  }

  // Validates the full response header block before storing anything. An
  // over-cap block rejects AND sets the sticky truncated fact on the still
  // pending request, so the caller may re-deliver a smaller response while
  // the loss stays observable.
  private checkResponseHeaders(value: unknown, request: HttpRequestRecord): [string, string][] {
    if (!Array.isArray(value)) {
      throw new HttpPeerError("rejected", "invalid-request", "response headers must be an array");
    }
    const headers: [string, string][] = [];
    let bytes = 0;
    for (const entry of value) {
      if (!Array.isArray(entry) || entry.length !== 2) {
        throw new HttpPeerError("rejected", "invalid-request", "response headers must be pairs");
      }
      const [nameValue, valueValue] = entry;
      if (!isHeaderName(nameValue)) {
        throw new HttpPeerError(
          "rejected",
          "unsupported-capability",
          "response header name is not known",
        );
      }
      const headerValue = checkHeaderValue(valueValue, this.limits.max_header_value_bytes);
      bytes += headerBlockBytes(nameValue, headerValue);
      headers.push([nameValue, headerValue]);
    }
    if (headers.length > this.limits.max_headers || bytes > this.limits.max_header_bytes) {
      request.response_headers_truncated = true;
      throw new HttpPeerError("rejected", "header-limit", "response header block is full");
    }
    return headers;
  }

  // Validates the full response body before storing anything. An over-cap
  // body rejects AND sets the sticky truncated fact on the still pending
  // request, so the caller may re-deliver a smaller response.
  private checkResponseBody(value: unknown, request: HttpRequestRecord): readonly number[] {
    if (!Array.isArray(value)) {
      throw new HttpPeerError("rejected", "invalid-request", "response body must be a byte array");
    }
    for (const byte of value) {
      if (typeof byte !== "number" || !Number.isInteger(byte) || byte < 0 || byte > 255) {
        throw new HttpPeerError("rejected", "invalid-request", "response body must hold bytes");
      }
    }
    if (value.length > this.limits.max_body_bytes) {
      request.response_body_truncated = true;
      throw new HttpPeerError("rejected", "body-limit", "response body exceeds the body cap");
    }
    return Object.freeze([...value]);
  }

  private snapshot(request: HttpRequestRecord): HttpRequestFacts {
    const response = request.response;
    return Object.freeze({
      id: request.id,
      owner: request.owner,
      origin: request.origin,
      method: request.method,
      target: request.target,
      version: request.version,
      state: request.state,
      header_count: request.headers.length,
      headers: Object.freeze(
        request.headers.map(([name, value]) => Object.freeze([name, value]) as HttpHeader),
      ),
      header_bytes: request.header_bytes,
      headers_truncated: request.headers_truncated,
      upload_accepted: request.upload_accepted,
      upload_complete: request.state !== "upload_open",
      upload_truncated: request.upload_truncated,
      upload_length_mismatch: request.upload_length_mismatch,
      response_delivered: response !== null,
      response_status: response === null ? null : response.status,
      response_body_accepted: response === null ? 0 : response.body_accepted,
      response_body_consumed: response === null ? 0 : response.body_consumed,
      response_headers_truncated: request.response_headers_truncated,
      response_body_truncated: request.response_body_truncated,
      response_length_mismatch: response !== null && response.length_mismatch,
      redirect: response === null ? null : response.redirect,
      redirect_incomplete: response !== null && response.redirect_incomplete,
      reissue_count: request.reissue_count,
      supersedes: request.supersedes,
    });
  }

  private responseSnapshot(response: HttpResponseRecord): HttpResponseFacts {
    return Object.freeze({
      status: response.status,
      header_count: response.headers.length,
      headers: Object.freeze(
        response.headers.map(([name, value]) => Object.freeze([name, value]) as HttpHeader),
      ),
      header_bytes: response.header_bytes,
      body_accepted: response.body_accepted,
      body_consumed: response.body_consumed,
      body_fully_consumed: response.body.length === 0,
      length_mismatch: response.length_mismatch,
      redirect: response.redirect,
      redirect_incomplete: response.redirect_incomplete,
    });
  }
}
