import { success, failure, type Completion, type AssertionContext } from "../completion.ts";
import { denyLiveBoundary } from "../assert/context.ts";
import { record, array, dataArray, dataProperty } from "../data.ts";
import { ownBytes, copyBytes, byteLength, isBytes, type Bytes } from "../bytes.ts";
import { createDomainRuntime } from "../domain.ts";
import { resourceStateFailure } from "../failure.ts";
import { CodecIssue } from "../codec/budget.ts";
import { decodeJSON, encodeJSON, type Schema } from "../codec/json.ts";
import { mediaType } from "../transport/media.ts";
import { openByteCell } from "../transport/stream/readable.ts";
import {
  registerReader,
  registerWriter,
  useWriter,
  type Fail,
  type ReaderCell,
  type SinkLike,
} from "../transport/stream/lifecycle.ts";
import { strictParameters, decodeForm, FormIssue, type FormSchema } from "./form.ts";
import { decodeMultipart, MultipartIssue } from "./multipart.ts";
import { renderSafe } from "./html.ts";
const origin = Object.freeze({
  source: "can:http-server",
  start: 0,
  end: 0,
  invocation: Object.freeze([]),
});
type Snapshot = Readonly<{
  method: string;
  path: string;
  query: readonly (readonly [string, string])[];
  queryInvalid: boolean;
  headers: readonly (readonly [string, string])[];
  body: Bytes;
}>;
type BodyCell =
  | { kind: "buffered"; reader: boolean; native: Request }
  | {
      kind: "live";
      native: Request;
      limit: number;
      reader: boolean;
      buffered?: Bytes;
      cell?: ReaderCell;
    };
const requests = new WeakMap<object, Snapshot>(),
  bodies = new WeakMap<object, BodyCell>();
// WebSocket upgrade plumbing. Every snapshot retains its native request;
// the server binds its handle after snapshotting, and accept claims the
// pair exactly once. Claimed requests skip the HTTP reply in dispatch.
export type UpgradeServer = Readonly<{
  upgrade: (
    request: Request,
    options: Readonly<{ headers?: Record<string, string>; data: unknown }>,
  ) => boolean;
}>;
const upgradeServers = new WeakMap<object, UpgradeServer>(),
  upgraded = new WeakSet<object>();
export function bindRequestServer(request: unknown, server: UpgradeServer): void {
  if (object(request) && requests.has(request)) upgradeServers.set(request, server);
}
export function claimUpgrade(
  request: unknown,
): Readonly<{ native: Request; server: UpgradeServer } | "upgraded" | undefined> {
  if (!object(request) || !requests.has(request)) throw resourceStateFailure(undefined, origin);
  if (upgraded.has(request)) return "upgraded";
  const server = upgradeServers.get(request);
  if (server === undefined) return undefined;
  upgraded.add(request);
  return { native: bodies.get(request)!.native, server };
}
export function releaseUpgrade(request: unknown): void {
  if (object(request)) upgraded.delete(request);
}
export function isUpgraded(request: unknown): boolean {
  return object(request) && upgraded.has(request);
}
export function offeredProtocols(request: unknown): readonly string[] {
  const snapshot = requestSnapshot(request),
    offered: string[] = [];
  for (const [name, value] of snapshot.headers)
    if (name === "sec-websocket-protocol")
      for (const part of value.split(",")) {
        const protocol = part.trim();
        if (protocol !== "") offered.push(protocol);
      }
  return Object.freeze(offered);
}
export const UPGRADED_RESPONSE: Response = Object.freeze(
  Object.create(null),
) as unknown as Response;
export function isUpgradedResponse(value: unknown): boolean {
  return value === UPGRADED_RESPONSE;
}
const object = (value: unknown): value is object =>
  value !== null && (typeof value === "object" || typeof value === "function");
export function requestSnapshot(value: unknown): Snapshot {
  if (!object(value) || !requests.has(value)) throw resourceStateFailure(undefined, origin);
  return requests.get(value)!;
}
export function isRequest(value: unknown): boolean {
  return object(value) && requests.has(value);
}
function bodyCell(value: unknown): BodyCell {
  if (!object(value) || !bodies.has(value)) throw resourceStateFailure(undefined, origin);
  return bodies.get(value)!;
}
export type SnapshotResult = Readonly<
  { kind: "request"; value: unknown } | { kind: "rejected"; status: 400 | 413 }
>;
export function normalizedPath(url: URL): string {
  const path = decodeURIComponent(url.pathname);
  // oxlint-disable-next-line no-control-regex -- Decoded request paths reject C0 controls, DEL, and backslash.
  if (!path.startsWith("/") || /[\x00-\x1f\x7f\\]/.test(path))
    throw new URIError("invalid request path");
  return path;
}
type Head = Readonly<{
  method: string;
  path: string;
  query: readonly (readonly [string, string])[];
  queryInvalid: boolean;
  headers: readonly (readonly [string, string])[];
}>;
function snapshotHead(
  request: Request,
): Readonly<{ kind: "head"; value: Head } | { kind: "rejected"; status: 400 }> {
  let url: URL, path: string;
  try {
    url = new URL(request.url);
    path = normalizedPath(url);
  } catch (cause) {
    if (!(cause instanceof TypeError) && !(cause instanceof URIError)) throw cause;
    return { kind: "rejected", status: 400 };
  }
  let query: readonly (readonly [string, string])[] = [],
    queryInvalid = false;
  try {
    query = Object.freeze(
      Array.from(strictParameters(url.search.slice(1)), (entry) => Object.freeze(entry)),
    );
  } catch (cause) {
    if (!(cause instanceof FormIssue) && !(cause instanceof CodecIssue)) throw cause;
    queryInvalid = true;
  }
  const headers = Object.freeze(
    Array.from(request.headers.entries(), (entry) => Object.freeze(entry)),
  );
  return { kind: "head", value: { method: request.method, path, query, queryInvalid, headers } };
}
async function drainBody(
  body: ReadableStream<Uint8Array> | null,
  limit: number,
): Promise<Readonly<{ kind: "bytes"; value: Bytes } | { kind: "rejected"; status: 400 | 413 }>> {
  const chunks: Uint8Array[] = [];
  let size = 0;
  if (body) {
    const reader = body.getReader();
    let complete = false;
    try {
      for (;;) {
        let next: Awaited<ReturnType<typeof reader.read>>;
        try {
          next = await reader.read();
        } catch {
          return { kind: "rejected", status: 400 };
        }
        if (next.done) {
          complete = true;
          break;
        }
        if (next.value.byteLength > limit - size) return { kind: "rejected", status: 413 };
        size += next.value.byteLength;
        if (next.value.byteLength !== 0) chunks.push(new Uint8Array(next.value));
      }
    } finally {
      if (!complete)
        try {
          await reader.cancel();
        } catch {}
      reader.releaseLock();
    }
  }
  const data = new Uint8Array(size);
  let offset = 0;
  for (const chunk of chunks) {
    data.set(chunk, offset);
    offset += chunk.byteLength;
  }
  return { kind: "bytes", value: ownBytes(data) };
}
// Private server ingress. Can receives only a detached immutable snapshot;
// native Request/Headers/stream objects never enter source-level values.
export async function snapshotRequest(request: Request, limit: number): Promise<SnapshotResult> {
  if (!Number.isSafeInteger(limit) || limit < 1 || limit > 67108864)
    throw new TypeError("invalid server body budget");
  const head = snapshotHead(request);
  if (head.kind === "rejected") return head;
  const drained = await drainBody(request.body, limit);
  if (drained.kind === "rejected") return drained;
  const token = Object.freeze(Object.create(null));
  requests.set(token, Object.freeze({ ...head.value, body: drained.value }));
  bodies.set(token, { kind: "buffered", reader: false, native: request });
  return { kind: "request", value: token };
}
// Lazy ingress for stream-marked routes: head only, no body read. The first
// body access consumes the wire stream exactly once; dispatch abandons an
// unread body so the connection never waits on handler inaction.
export async function snapshotRequestLazy(
  request: Request,
  limit: number,
): Promise<SnapshotResult> {
  if (!Number.isSafeInteger(limit) || limit < 1 || limit > 67108864)
    throw new TypeError("invalid server body budget");
  const head = snapshotHead(request);
  if (head.kind === "rejected") return head;
  const token = Object.freeze(Object.create(null));
  requests.set(token, Object.freeze({ ...head.value, body: ownBytes(new Uint8Array(0)) }));
  bodies.set(token, { kind: "live", native: request, limit, reader: false });
  return { kind: "request", value: token };
}
// Lifetime revocation is separate from body abandonment: abandonRequest only
// releases an unread live body, while revokeRequest ends the Can capability
// for buffered and live tokens alike. The outer request boundary calls this
// after the per-request owner scope drains; retained header, body, snapshot,
// and upgrade operations then fail with the resource-state failure. Unknown,
// forged, or already-revoked input is a silent no-op so the call stays safe
// in finally paths and under overlapping dispatch-plus-server coverage.
export function revokeRequest(request: unknown): void {
  if (!object(request)) return;
  requests.delete(request);
  bodies.delete(request);
  upgradeServers.delete(request);
  upgraded.delete(request);
}
export async function abandonRequest(request: unknown): Promise<void> {
  const cell = object(request) && bodies.has(request) ? bodies.get(request)! : undefined;
  if (cell === undefined || cell.kind !== "live") return;
  if (cell.cell !== undefined) {
    try {
      await cell.cell.reader?.cancel();
    } catch {}
    return;
  }
  if (!cell.reader && cell.buffered === undefined && cell.native.body !== null) {
    const reader = cell.native.body.getReader();
    try {
      await reader.cancel();
    } catch {
    } finally {
      reader.releaseLock();
    }
  }
}
function memoryStream(bytes: Bytes): ReadableStream<Uint8Array> {
  const copy = new Uint8Array(copyBytes(bytes, origin));
  return new ReadableStream<Uint8Array>({
    start(controller) {
      if (copy.byteLength !== 0) controller.enqueue(copy);
      controller.close();
    },
  });
}

type ResponseSnapshot = Readonly<{
  status: number;
  headers: readonly (readonly [string, string])[];
  body: Bytes | PendingBody | null;
}>;
type PendingBody = {
  stream: ReadableStream<Uint8Array>;
  controller: ReadableStreamDefaultController<Uint8Array> | undefined;
  cancelled: boolean;
  ended: boolean;
  taken: boolean;
  sent: boolean;
};
// Produce-then-serve: the handler fills a bounded byte queue, returns the
// pending response, and Bun streams the queue after dispatch. Handlers never
// outlive dispatch, so no write can observe a disconnect; short writes are
// the only flow control and the queue cap is the only bound.
const RESPONSE_QUEUE_BYTES = 1048576;
const statuses = new WeakMap<object, number>(),
  bodyStatuses = new WeakMap<object, number>();
const serverHeaders = new WeakMap<object, readonly (readonly [string, string])[]>(),
  responses = new WeakMap<object, ResponseSnapshot>();
function opaque<T>(map: WeakMap<object, T>, data: T): unknown {
  const token = Object.freeze(Object.create(null));
  map.set(token, data);
  return token;
}
function read<T>(map: WeakMap<object, T>, value: unknown): T {
  if (!object(value) || !map.has(value)) throw resourceStateFailure(undefined, origin);
  return map.get(value)!;
}
function textBytes(text: string) {
  return ownBytes(new TextEncoder().encode(text));
}
function buildResponse(
  status: unknown,
  headers: unknown,
  body: Bytes | PendingBody | null,
  mime?: string,
): unknown {
  const code = read(body === null ? statuses : bodyStatuses, status),
    entries = read(serverHeaders, headers);
  // Set-Cookie never enters the coalescing Headers: repeats stay separate
  // entries so expires dates (which contain commas) survive intact.
  const rest = entries.filter(([name]) => name.toLowerCase() !== "set-cookie"),
    cookies = entries.filter(([name]) => name.toLowerCase() === "set-cookie");
  const native = new Headers(rest.map(([name, value]) => [name, value]));
  native.set("x-content-type-options", "nosniff");
  if (mime) native.set("content-type", mime);
  return opaque(
    responses,
    Object.freeze({
      status: code,
      headers: Object.freeze([
        ...Array.from(native.entries(), (entry) => Object.freeze(entry)),
        ...cookies.map(([, value]) => Object.freeze(["set-cookie", value] as const)),
      ]),
      body,
    }),
  );
}
// Compiler-owned complete response for adapter ingress failures and action
// outcomes: fixed headers plus caller-selected text or fragment bytes.
export function ownedResponse(status: number, text: string, html: boolean): unknown {
  if (!Number.isInteger(status) || status < 200 || status > 599)
    throw new TypeError("invalid compiler action status");
  return buildResponse(
    opaque(bodyStatuses, status),
    opaque(serverHeaders, Object.freeze([])),
    textBytes(text),
    html ? "text/html; charset=utf-8" : "text/plain; charset=utf-8",
  );
}
export type SnapshotBody = Readonly<
  { kind: "bytes"; value: Bytes } | { kind: "consumed" } | { kind: "failed" } | { kind: "limited" }
>;
// Adapter body read: buffered snapshots stay repeatable from cached bytes,
// while a first live touch drains the wire stream exactly once.
export async function snapshotBodyBytes(token: unknown): Promise<SnapshotBody> {
  const snapshot = requestSnapshot(token),
    cell = bodyCell(token);
  if (cell.kind === "live") {
    if (cell.buffered !== undefined) return { kind: "bytes", value: cell.buffered };
    if (cell.reader) return { kind: "consumed" };
    const drained = await drainBody(cell.native.body, cell.limit);
    if (drained.kind === "rejected")
      return drained.status === 413 ? { kind: "limited" } : { kind: "failed" };
    cell.buffered = drained.value;
    return { kind: "bytes", value: drained.value };
  }
  return { kind: "bytes", value: snapshot.body };
}
export function isHTTPValue(kind: string | undefined, value: unknown): boolean {
  if (kind === "request") return isRequest(value);
  const map =
    kind === "status"
      ? statuses
      : kind === "body_status"
        ? bodyStatuses
        : kind === "server_headers"
          ? serverHeaders
          : kind === "server_response"
            ? responses
            : undefined;
  return map !== undefined && object(value) && map.has(value);
}
// Each dispatch converts its complete immutable value once. A later dispatch
// can reuse the Can value without reusing an already-consumed native body.
export function nativeResponse(value: unknown, head = false): Response {
  const response = read(responses, value),
    headers: [string, string][] = response.headers.map(([name, value]) => [name, value]);
  const body = response.body;
  // Stream responses convert once: the native stream is single-use, so reuse
  // is a usage violation rather than a silently empty second body.
  if (body !== null && !isBytes(body)) {
    if (body.sent) throw resourceStateFailure(undefined, origin);
    body.sent = true;
    if (!body.taken && body.controller !== undefined && !body.ended) {
      body.ended = true;
      try {
        body.controller.close();
      } catch {}
    }
    if (head) return new Response(null, { status: response.status, headers });
    return new Response(body.stream, { status: response.status, headers });
  }
  // HEAD suppresses every body while keeping the entity length the complete
  // value produced, so HEAD and GET agree on length by construction.
  const bytes = body as Bytes | null;
  if (head && bytes !== null)
    return new Response(null, {
      status: response.status,
      headers: [...headers, ["content-length", byteLength(bytes).toString()]],
    });
  return new Response(bytes === null ? null : new Uint8Array(copyBytes(bytes, origin)), {
    status: response.status,
    headers,
  });
}
const forbiddenResponseHeaders = new Set([
  "content-type",
  "content-length",
  "x-content-type-options",
  "content-security-policy",
  "content-security-policy-report-only",
  "connection",
  "keep-alive",
  "proxy-authenticate",
  "proxy-authorization",
  "te",
  "trailer",
  "transfer-encoding",
  "upgrade",
]);
export function createResponses(
  domain: ReturnType<typeof createDomainRuntime>,
  types: Pick<Types, "invalid" | "invalidData" | "close" | "writeFailed" | "limit">,
) {
  function sseLine(field: string, value: string): string | undefined {
    if (value === "" || value.includes("\n") || value.includes("\r")) return undefined;
    return field + ": " + value + "\n";
  }
  function sseFrame(event: unknown): Uint8Array | Completion<never> {
    // Record shape is catalogue-typed; anything else is a forged value.
    if (!object(event)) throw resourceStateFailure(undefined, origin);
    const data = dataProperty(event, "data"),
      name = dataProperty(event, "event"),
      id = dataProperty(event, "id"),
      retry = dataProperty(event, "retry");
    if (
      typeof data !== "string" ||
      typeof name !== "string" ||
      typeof id !== "string" ||
      typeof retry !== "string"
    )
      throw resourceStateFailure(undefined, origin);
    let frame = "";
    if (name !== "") {
      const line = sseLine("event", name);
      if (line === undefined) return invalid("sse_event");
      frame += line;
    }
    if (id !== "") {
      const line = sseLine("id", id);
      if (line === undefined) return invalid("sse_id");
      frame += line;
    }
    if (retry !== "") {
      if (!/^[0-9]+$/.test(retry)) return invalid("sse_retry");
      frame += "retry: " + retry + "\n";
    }
    // Data splits across lines; CR, LF and CRLF all break lines. Empty
    // data emits no data line so retry/id-only blocks never dispatch.
    if (data !== "") for (const line of data.split(/\r\n|\n|\r/)) frame += "data: " + line + "\n";
    return new TextEncoder().encode(frame + "\n");
  }
  const invalid = (reason: string) =>
    failure(domain.create(types.invalid, record(types.invalid, [["reason", reason]]), origin));
  const fail: Fail = (identity, fields, cause) =>
    failure(domain.create(identity, record(identity, fields), origin, cause));
  function status(value: bigint, body: boolean): Completion<unknown> {
    if (
      value < 200n ||
      value > 599n ||
      (body && (value === 204n || value === 205n || value === 304n))
    )
      return invalid("invalid_status");
    return success(opaque(body ? bodyStatuses : statuses, Number(value)));
  }
  function streamBody(): PendingBody {
    const pending: PendingBody = {
      stream: undefined as unknown as ReadableStream<Uint8Array>,
      controller: undefined,
      cancelled: false,
      ended: false,
      taken: false,
      sent: false,
    };
    pending.stream = new ReadableStream<Uint8Array>(
      {
        start(controller) {
          pending.controller = controller as ReadableStreamDefaultController<Uint8Array>;
        },
        cancel() {
          pending.cancelled = true;
        },
      },
      {
        highWaterMark: RESPONSE_QUEUE_BYTES,
        size(chunk?: Uint8Array) {
          return chunk === undefined ? 0 : chunk.byteLength;
        },
      },
    );
    return pending;
  }
  const response = buildResponse;
  return Object.freeze({
    async makeStatus(value: bigint, _context?: AssertionContext): Promise<Completion<unknown>> {
      return status(value, false);
    },
    async makeBodyStatus(value: bigint, _context?: AssertionContext): Promise<Completion<unknown>> {
      return status(value, true);
    },
    async ok(_context?: AssertionContext): Promise<Completion<unknown>> {
      return status(200n, true);
    },
    async unprocessable(_context?: AssertionContext): Promise<Completion<unknown>> {
      return status(422n, true);
    },
    async internal(_context?: AssertionContext): Promise<Completion<unknown>> {
      return status(500n, true);
    },
    async unavailable(_context?: AssertionContext): Promise<Completion<unknown>> {
      return status(503n, true);
    },
    async emptyHeaders(_context?: AssertionContext): Promise<Completion<unknown>> {
      return success(opaque(serverHeaders, Object.freeze([])));
    },
    async makeHeaders(input: unknown, _context?: AssertionContext): Promise<Completion<unknown>> {
      const headers = new Headers(),
        scratch = new Headers(),
        setCookies: string[] = [];
      for (const entry of dataArray(input)) {
        const name = dataProperty(entry, "name"),
          value = dataProperty(entry, "value");
        if (typeof name !== "string" || typeof value !== "string")
          throw new TypeError("invalid compiler header");
        if (
          forbiddenResponseHeaders.has(name.toLowerCase()) ||
          name.toLowerCase().startsWith("hx-") ||
          !name.isWellFormed() ||
          !value.isWellFormed()
        )
          return invalid("invalid_header");
        // Set-Cookie validates through a scratch jar but snapshots apart:
        // Headers.entries would comma-join repeats and corrupt expires dates.
        if (name.toLowerCase() === "set-cookie") {
          try {
            scratch.append(name, value);
          } catch (cause) {
            if (cause instanceof TypeError) return invalid("invalid_header");
            throw cause;
          }
          setCookies.push(value);
          continue;
        }
        try {
          headers.append(name, value);
        } catch (cause) {
          if (cause instanceof TypeError) return invalid("invalid_header");
          throw cause;
        }
      }
      return success(
        opaque(
          serverHeaders,
          Object.freeze([
            ...Array.from(headers.entries(), (entry) => Object.freeze(entry)),
            ...setCookies.map((value) => Object.freeze(["set-cookie", value] as const)),
          ]),
        ),
      );
    },
    async empty(
      status: unknown,
      headers: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      return success(response(status, headers, null));
    },
    async bytes(
      status: unknown,
      headers: unknown,
      body: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      return success(
        response(status, headers, ownBytes(copyBytes(body, origin)), "application/octet-stream"),
      );
    },
    async text(
      status: unknown,
      headers: unknown,
      body: string,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      return success(response(status, headers, textBytes(body), "text/plain; charset=utf-8"));
    },
    async html(
      status: unknown,
      headers: unknown,
      body: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      return success(
        response(status, headers, textBytes(renderSafe(body)), "text/html; charset=utf-8"),
      );
    },
    async json<T>(
      schema: Schema,
      status: unknown,
      headers: unknown,
      body: T,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      try {
        return success(
          response(status, headers, encodeJSON(schema, body), "application/json; charset=utf-8"),
        );
      } catch (cause) {
        if (cause instanceof CodecIssue)
          return failure(
            domain.create(
              types.invalidData,
              record(types.invalidData, [
                ["path", cause.path],
                ["reason", cause.reason],
              ]),
              origin,
            ),
          );
        throw cause;
      }
    },
    async stream(
      status: unknown,
      headers: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      return success(response(status, headers, streamBody(), "application/octet-stream"));
    },
    async writer(input: unknown, _context?: AssertionContext): Promise<Completion<object>> {
      const snapshot = read(responses, input);
      if (snapshot.body === null || isBytes(snapshot.body)) return invalid("not_streaming");
      if (snapshot.body.taken) return invalid("writer_taken");
      snapshot.body.taken = true;
      const pending = snapshot.body;
      // Short writes are the only flow control: accept up to the queue's
      // remaining byte budget and let the caller retry the rest. Never block.
      const sink: FrameSink = {
        write(chunk: Uint8Array): unknown {
          if (pending.cancelled || pending.controller === undefined)
            throw new DOMException("response stream cancelled", "AbortError");
          const room = pending.controller.desiredSize;
          if (room === null) throw new DOMException("response stream closed", "AbortError");
          const take = Math.max(0, Math.min(chunk.byteLength, Math.floor(room)));
          if (take === 0) return 0;
          pending.controller.enqueue(chunk.slice(0, take));
          return take;
        },
        // Frames are atomic: a short frame would tear the event stream, so a
        // frame that cannot fit whole writes nothing and reports zero.
        writeFrame(frame: Uint8Array): number {
          if (pending.cancelled || pending.controller === undefined)
            throw new DOMException("response stream cancelled", "AbortError");
          const room = pending.controller.desiredSize;
          if (room === null) throw new DOMException("response stream closed", "AbortError");
          if (frame.byteLength > Math.floor(room)) return 0;
          pending.controller.enqueue(frame.slice());
          return frame.byteLength;
        },
        flush(): unknown {
          return undefined;
        },
        end(): unknown {
          if (!pending.ended && pending.controller !== undefined) {
            pending.ended = true;
            try {
              pending.controller.close();
            } catch {}
          }
          return undefined;
        },
      };
      return success(registerWriter({ sink }, fail, types.close, { scopeManaged: true }));
    },
    async sse(
      status: unknown,
      headers: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      return success(response(status, headers, streamBody(), "text/event-stream; charset=utf-8"));
    },
    async sseSend(
      writer: unknown,
      event: unknown,
      context?: AssertionContext,
    ): Promise<Completion<bigint>> {
      denyLiveBoundary(context, origin);
      const frame = sseFrame(event);
      if (!(frame instanceof Uint8Array)) return frame;
      return sendFrame(writer, frame);
    },
    async sseComment(
      writer: unknown,
      text: unknown,
      context?: AssertionContext,
    ): Promise<Completion<bigint>> {
      denyLiveBoundary(context, origin);
      if (typeof text !== "string" || text.includes("\n") || text.includes("\r"))
        return invalid("sse_comment");
      return sendFrame(writer, new TextEncoder().encode(": " + text + "\n\n"));
    },
  });
  function sendFrame(writer: unknown, frame: Uint8Array): Promise<Completion<bigint>> {
    return useWriter(writer, async (cell) => {
      const sink = cell.sink as Partial<FrameSink>;
      if (typeof sink.writeFrame !== "function") return invalid("not_streaming");
      let accepted: number;
      try {
        accepted = sink.writeFrame(frame);
      } catch (cause) {
        const aborted =
          typeof cause === "object" &&
          cause !== null &&
          (cause as { name?: unknown }).name === "AbortError";
        const reason = aborted
          ? "aborted"
          : typeof cause === "object" &&
              cause !== null &&
              typeof (cause as { code?: unknown }).code === "string"
            ? (cause as { code: string }).code
            : "io_error";
        return fail(types.writeFailed, [["reason", reason]], cause);
      }
      // Zero means the frame cannot fit the remaining queue; nothing was
      // written, and pre-dispatch nothing will drain, so the bound bit.
      if (accepted === 0 && frame.byteLength > 0)
        return failure(
          domain.create(
            types.limit,
            record(types.limit, [["limit", BigInt(RESPONSE_QUEUE_BYTES)]]),
            origin,
          ),
        );
      return success(BigInt(accepted));
    });
  }
}
type Types = Readonly<{
  invalid: string;
  limit: string;
  invalidData: string;
  header: string;
  close: string;
  writeFailed: string;
  multipartForm: string;
  multipartField: string;
  multipartFile: string;
}>;
type FrameSink = SinkLike & { writeFrame(frame: Uint8Array): number };
export function createRequests<Header>(
  domain: ReturnType<typeof createDomainRuntime>,
  types: Types,
) {
  const invalid = (reason: string) =>
    failure(domain.create(types.invalid, record(types.invalid, [["reason", reason]]), origin));
  const overLimit = (value: bigint) =>
    failure(domain.create(types.limit, record(types.limit, [["limit", value]]), origin));
  const fail: Fail = (identity, fields, cause) =>
    failure(domain.create(identity, record(identity, fields), origin, cause));
  // Buffered reads stay repeatable from cached bytes. A live body buffers on
  // first buffered access; once its reader opens, the wire cannot rewind.
  async function bufferedBody(
    token: unknown,
    snapshot: Snapshot,
    limit: bigint,
  ): Promise<Completion<Bytes>> {
    const cell = bodyCell(token);
    let bytes: Bytes;
    if (cell.kind === "live") {
      if (cell.buffered === undefined) {
        if (cell.reader) return invalid("body_consumed");
        const drained = await drainBody(cell.native.body, cell.limit);
        if (drained.kind === "rejected")
          return drained.status === 413 ? overLimit(BigInt(cell.limit)) : invalid("body_read");
        cell.buffered = drained.value;
      }
      bytes = cell.buffered;
    } else bytes = snapshot.body;
    return limit < 0n || byteLength(bytes) > limit ? overLimit(limit) : success(bytes);
  }
  const codecFailure = (cause: unknown): Completion<never> => {
    if (cause instanceof FormIssue) return invalid(cause.reason);
    if (cause instanceof CodecIssue)
      return failure(
        domain.create(
          types.invalidData,
          record(types.invalidData, [
            ["path", cause.path],
            ["reason", cause.reason],
          ]),
          origin,
        ),
      );
    throw cause;
  };
  function media(snapshot: Snapshot, form: boolean): boolean {
    const value = snapshot.headers.find(([name]) => name === "content-type")?.[1];
    if (value === undefined) return false;
    try {
      const parsed = mediaType(value);
      return (
        parsed.type === (form ? "application/x-www-form-urlencoded" : "application/json") &&
        [...parsed.parameters].every(
          ([key, value]) => key === "charset" && value.toLowerCase() === "utf-8",
        )
      );
    } catch (cause) {
      if (cause instanceof CodecIssue) return false;
      throw cause;
    }
  }
  return Object.freeze({
    async method(request: unknown, context?: AssertionContext): Promise<Completion<string>> {
      denyLiveBoundary(context, origin);
      return success(requestSnapshot(request).method);
    },
    async path(request: unknown, context?: AssertionContext): Promise<Completion<string>> {
      denyLiveBoundary(context, origin);
      return success(requestSnapshot(request).path);
    },
    async headers(
      request: unknown,
      context?: AssertionContext,
    ): Promise<Completion<readonly Header[]>> {
      denyLiveBoundary(context, origin);
      return success(
        array(
          requestSnapshot(request).headers.map(
            ([name, value]) =>
              record(types.header, [
                ["name", name],
                ["value", value],
              ]) as Header,
          ),
        ),
      );
    },
    async queryAll(
      request: unknown,
      name: string,
      context?: AssertionContext,
    ): Promise<Completion<readonly string[]>> {
      denyLiveBoundary(context, origin);
      const snapshot = requestSnapshot(request);
      if (!name.isWellFormed()) return invalid("query_name");
      if (snapshot.queryInvalid) return invalid("query_value");
      return success(
        array(snapshot.query.filter(([key]) => key === name).map(([, value]) => value)),
      );
    },
    async queryOne(
      request: unknown,
      name: string,
      context?: AssertionContext,
    ): Promise<Completion<string>> {
      denyLiveBoundary(context, origin);
      const snapshot = requestSnapshot(request);
      if (!name.isWellFormed()) return invalid("query_name");
      if (snapshot.queryInvalid) return invalid("query_value");
      const values = snapshot.query.filter(([key]) => key === name);
      return values.length === 0
        ? invalid("query_missing")
        : values.length !== 1
          ? invalid("query_repeated")
          : success(values[0]![1]);
    },
    async body(
      request: unknown,
      limit: bigint,
      context?: AssertionContext,
    ): Promise<Completion<Bytes>> {
      denyLiveBoundary(context, origin);
      return bufferedBody(request, requestSnapshot(request), limit);
    },
    async json<T>(
      schema: Schema,
      request: unknown,
      limit: bigint,
      context?: AssertionContext,
    ): Promise<Completion<T>> {
      denyLiveBoundary(context, origin);
      const snapshot = requestSnapshot(request),
        body = await bufferedBody(request, snapshot, limit);
      if (body.kind !== "ok") return body;
      if (!media(snapshot, false)) return invalid("unsupported_media_type");
      try {
        return success(
          decodeJSON(
            schema,
            body.value,
            Math.max(1, Number(limit > 67108864n ? 67108864n : limit)),
          ) as T,
        );
      } catch (cause) {
        return codecFailure(cause);
      }
    },
    async form<T>(
      schema: FormSchema,
      request: unknown,
      limit: bigint,
      context?: AssertionContext,
    ): Promise<Completion<T>> {
      denyLiveBoundary(context, origin);
      const snapshot = requestSnapshot(request),
        body = await bufferedBody(request, snapshot, limit);
      if (body.kind !== "ok") return body;
      if (!media(snapshot, true)) return invalid("unsupported_media_type");
      try {
        return success(decodeForm(schema, body.value, Number(byteLength(body.value))) as T);
      } catch (cause) {
        return codecFailure(cause);
      }
    },
    async multipart(
      request: unknown,
      limit: bigint,
      fileLimit: bigint,
      context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      denyLiveBoundary(context, origin);
      if (typeof fileLimit !== "bigint" || fileLimit < 0n)
        return overLimit(typeof fileLimit === "bigint" ? fileLimit : -1n);
      const snapshot = requestSnapshot(request),
        body = await bufferedBody(request, snapshot, limit);
      if (body.kind !== "ok") return body;
      const value = snapshot.headers.find(([name]) => name === "content-type")?.[1];
      let boundary: string | undefined;
      try {
        if (value === undefined) return invalid("unsupported_media_type");
        const parsed = mediaType(value);
        if (parsed.type !== "multipart/form-data") return invalid("unsupported_media_type");
        boundary = parsed.parameters.get("boundary");
      } catch (cause) {
        if (cause instanceof CodecIssue) return invalid("unsupported_media_type");
        throw cause;
      }
      if (boundary === undefined) return invalid("multipart_boundary");
      try {
        const document = decodeMultipart(body.value, boundary);
        // Per-file cap applies on top of the total body cap; fields stay under
        // the total cap only.
        for (const file of document.files)
          if (byteLength(file.content) > fileLimit) return overLimit(fileLimit);
        return success(
          record(types.multipartForm, [
            [
              "fields",
              array(
                document.fields.map((field) =>
                  record(types.multipartField, [
                    ["name", field.name],
                    ["value", field.value],
                  ]),
                ),
              ),
            ],
            [
              "files",
              array(
                document.files.map((file) =>
                  record(types.multipartFile, [
                    ["name", file.name],
                    ["filename", file.filename],
                    ["content_type", file.contentType],
                    ["content", file.content],
                  ]),
                ),
              ),
            ],
          ]),
        );
      } catch (cause) {
        if (cause instanceof MultipartIssue) return invalid(cause.reason);
        return codecFailure(cause);
      }
    },
    async bodyStream(
      request: unknown,
      maxChunk: bigint,
      context?: AssertionContext,
    ): Promise<Completion<object>> {
      denyLiveBoundary(context, origin);
      const snapshot = requestSnapshot(request),
        cell = bodyCell(request);
      if (typeof maxChunk !== "bigint" || maxChunk < 1n)
        return overLimit(typeof maxChunk === "bigint" ? maxChunk : -1n);
      if (cell.reader) return invalid("body_consumed");
      cell.reader = true;
      // A reader after buffered access replays retained bytes; only the first
      // access to a live body observes the wire.
      const source =
        cell.kind === "live"
          ? cell.buffered !== undefined
            ? memoryStream(cell.buffered)
            : (cell.native.body ?? memoryStream(snapshot.body))
          : memoryStream(snapshot.body);
      const opened = openByteCell(source, maxChunk);
      // Request handles live in framework-drained scopes: explicit close
      // controls timing, and drain cleanup is by design, never a leak.
      const token = registerReader(opened, fail, types.close, { scopeManaged: true });
      if (cell.kind === "live") cell.cell = opened;
      return success(token);
    },
  });
}
