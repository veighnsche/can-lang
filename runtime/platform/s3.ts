// S3-compatible object storage over the native S3Client. Clients bind
// endpoint, region, bucket and credentials explicitly; keys are opaque
// data (slash is not traversal). Reads are bounded (whole, ranged or
// streamed through B1-05 byte readers); writes land singly or through
// a budgeted reader pump; multipart uploads are stateful handles with
// explicit finish and destructive discard; presigned URLs mint locally and stay
// inside opaque handles until described. Credentials and signed URLs
// never enter diagnostics. Native S3 codes map to missing_key,
// access_denied and service_error; required presign headers have no
// native option and stay unrepresentable (contract target gap).
import { success, failure, caught, type Completion, type AssertionContext } from "../completion.ts";
import { denyLiveBoundary } from "../assert/context.ts";
import { record, recordIdentity, dataProperty, array } from "../data.ts";
import { ownBytes, copyBytes, isBytes } from "../bytes.ts";
import { createDomainRuntime } from "../domain.ts";
import { mintTimeInstant } from "./datetime.ts";
import { registerResource } from "../owner.ts";
import { raceBoundary, type BoundaryOutcome } from "../transport/operation-budget.ts";
import { scopeRaceInput } from "../transport/request-scope.ts";
import type { RequestBudget } from "../transport/request-budget.ts";
import {
  registerReader,
  useReader,
  type Fail,
  type ReaderCell,
} from "../transport/stream/lifecycle.ts";
import { openByteCell } from "../transport/stream/readable.ts";
const origin = Object.freeze({ source: "can:s3", start: 0, end: 0, invocation: Object.freeze([]) });
// Structural view over the pinned Bun S3 surface (Bun.S3Client,
// S3File, NetworkSink writer). Method shapes mirror the qualified
// binary; only the operations below are consumed.
type NativeClient = InstanceType<typeof Bun.S3Client>;
type NativeFile = ReturnType<NativeClient["file"]>;
type NativeSink = ReturnType<NativeFile["writer"]>;
type NativeList = Awaited<ReturnType<NativeClient["list"]>>;
export const S3_CLIENT_KIND = "client",
  S3_UPLOAD_KIND = "upload",
  S3_PRESIGNED_KIND = "presigned",
  S3_CONTINUATION_KIND = "continuation";
const UPLOAD_RESOURCE_KIND = "s3-upload";
type UploadBox = {
  client: NativeClient;
  key: string;
  type: string | undefined;
  partSize: number | undefined;
  sink: NativeSink | undefined;
  state: "open" | "finished" | "discarded";
  scrubPending: boolean;
};
// S3Bounds carries the cancel-absent operation bounds (E04-style,
// trailing-optional, checker-mandatory later): the caller bound in
// milliseconds plus an explicit request budget. Emitted S3 calls keep
// their current arity; the runtime supplies bounds.
export type S3Bounds = Readonly<{
  boundMs?: number;
  budget?: RequestBudget;
}>;
// The fixed adapter ceiling (R1): the only new bound knob. Every S3
// await races at most this long, so the common budgetless case —
// dispatch invents no request total — stays bounded. Set above any
// measured live leg (local-MinIO awaits settle in milliseconds to
// low seconds) and below indefinite.
export const S3_AWAIT_CEILING_MS = 30000;
// Layered per-await bound (R1): min(explicit deadline remainder,
// trailing runtime boundMs, adapter ceiling). The ambient
// request-budget remainder joins inside raceBoundary through
// scopeRaceInput, so the race sees the full layered minimum.
export function s3RaceBoundMs(explicitMs?: number, bounds?: S3Bounds): number {
  const explicit = explicitMs ?? Number.POSITIVE_INFINITY;
  const trailing = bounds?.boundMs ?? Number.POSITIVE_INFINITY;
  return Math.max(1, Math.min(explicit, trailing, S3_AWAIT_CEILING_MS));
}
type PresignedBox = Readonly<{
  url: string;
  method: "GET" | "PUT" | "DELETE" | "HEAD";
  expires: object;
}>;
const clients = new WeakMap<object, NativeClient>(),
  uploads = new WeakMap<object, UploadBox>(),
  presigned = new WeakMap<object, PresignedBox>(),
  continuations = new WeakMap<object, string>();
const object = (value: unknown): value is object =>
  value !== null && (typeof value === "object" || typeof value === "function");
export function isS3Value(kind: string | undefined, value: unknown): boolean {
  if (!object(value)) return false;
  if (kind === S3_CLIENT_KIND) return clients.has(value);
  if (kind === S3_UPLOAD_KIND) return uploads.has(value);
  if (kind === S3_PRESIGNED_KIND) return presigned.has(value);
  if (kind === S3_CONTINUATION_KIND) return continuations.has(value);
  return false;
}
const READ_CAP = 67108864n,
  SAFE_MAX = 9007199254740991n,
  DEADLINE_CAP = 2147483647n,
  PART_MIN = 5242880n,
  PART_MAX = 5368709120n,
  EXPIRES_MAX = 604800n,
  STREAM_CHUNK = 65536n;
type Ids = Readonly<{
  invalid: string;
  missing: string;
  denied: string;
  service: string;
  closed: string;
  overLimit: string;
  metadata: string;
  entry: string;
  page: string;
  info: string;
  methodGet: string;
  methodPut: string;
  methodDelete: string;
  methodHead: string;
  someText: string;
  someInt: string;
  someContinuation: string;
  none: string;
  readFailed: string;
  cancelled: string;
  closeFailed: string;
}>;
export function createS3(domain: ReturnType<typeof createDomainRuntime>, ids: Ids) {
  const fail: Fail = (identity, fields, cause) =>
    failure(domain.create(identity, record(identity, fields), origin, cause));
  const invalid = (reason: string) => fail(ids.invalid, [["reason", reason]]);
  const missing = (key: string) => fail(ids.missing, [["key", key]]);
  const denied = (operation: string) => fail(ids.denied, [["operation", operation]]);
  const service = (code: string, operation: string) =>
    fail(ids.service, [
      ["code", code],
      ["operation", operation],
    ]);
  const closed = (operation: string, state: string) =>
    fail(ids.closed, [
      ["operation", operation],
      ["state", state],
    ]);
  const overLimit = (limit: bigint, size: bigint) =>
    fail(ids.overLimit, [
      ["limit", limit],
      ["size", size],
    ]);
  const s3Code = (cause: unknown): string | undefined => {
    if (typeof cause !== "object" || cause === null) return undefined;
    const code = (cause as { code?: unknown }).code;
    return typeof code === "string" ? code : undefined;
  };
  // S3Error-shaped faults map by code; anything else (adapter bugs,
  // foreign throws) keeps standard native-exception provenance.
  const wire = (operation: string, key: string | undefined, cause: unknown): Completion<never> => {
    const code = s3Code(cause);
    if (code === "NoSuchKey") return missing(key ?? "");
    if (code === "UnknownError") return denied(operation);
    if (code !== undefined) return service(code, operation);
    return caught(cause, origin);
  };
  // One cancel-absent per-await race (R1). The pending native is
  // captured for deferred convergence: expiry returns the honest
  // timeout outcome while the native stays owned until settlement,
  // with one escalation plus late-settlement observation flowing to
  // the race sink automatically. pending stays undefined when the
  // boundary pre-expired (start never ran), so no phantom native
  // can draw deferred cleanup. Rejections propagate to the caller,
  // exactly like the unraced await they replace.
  const raceS3 = async <T>(
    explicitMs: number | undefined,
    bounds: S3Bounds | undefined,
    start: () => T | Promise<T>,
  ): Promise<{ raced: BoundaryOutcome<T>; pending: Promise<T> | undefined }> => {
    let pending: Promise<T> | undefined;
    const raced = await raceBoundary(
      {
        source: "s3",
        boundMs: s3RaceBoundMs(explicitMs, bounds),
        ...scopeRaceInput(bounds?.budget),
      },
      () => (pending = Promise.resolve(start())),
    );
    return { raced, pending };
  };
  // Deferred convergence (R1 race_and_own): when the hung native
  // later settles either way, run the standard destructive scrub.
  // Best-effort like the pump-finally scrub it extends: by the time
  // it runs, the caller already holds the timeout outcome and no
  // channel remains to report scrub detail.
  const convergeDeferred = <T>(
    pending: Promise<T> | undefined,
    scrubStep: () => Promise<unknown>,
  ): void => {
    pending?.then(
      () => {
        void scrubStep();
      },
      () => {
        void scrubStep();
      },
    );
  };
  // Upload poison on op expiry (R1 poison_discard): the box becomes
  // discarded immediately so no caller code can build on
  // unknown-write state, and the retained sink converges through
  // one deferred scrub on settlement. Single-flight per box:
  // concurrent same-box expiries attach once.
  const poisonUpload = <T>(
    box: UploadBox,
    pending: Promise<T> | undefined,
    operation: string,
  ): void => {
    if (pending === undefined || box.state !== "open") return;
    box.state = "discarded";
    const doomed = box.sink;
    box.sink = undefined;
    if (box.scrubPending) return;
    box.scrubPending = true;
    convergeDeferred(pending, () => scrub(box.client, box.key, doomed, operation));
  };
  const option = (some: string, value: unknown): unknown => {
    const identity = recordIdentity(value);
    if (identity === some) return dataProperty(value, "value");
    if (identity === ids.none) return undefined;
    throw new TypeError("invalid compiler s3 option");
  };
  const methodOf = (value: unknown): PresignedBox["method"] => {
    const identity = recordIdentity(value);
    if (identity === ids.methodGet) return "GET";
    if (identity === ids.methodPut) return "PUT";
    if (identity === ids.methodDelete) return "DELETE";
    if (identity === ids.methodHead) return "HEAD";
    throw new TypeError("invalid compiler s3 method");
  };
  const methodLeaf = (method: PresignedBox["method"]): string => {
    if (method === "GET") return ids.methodGet;
    if (method === "PUT") return ids.methodPut;
    if (method === "DELETE") return ids.methodDelete;
    return ids.methodHead;
  };
  const projectClient = (value: unknown): NativeClient => {
    const client = object(value) ? clients.get(value) : undefined;
    if (client === undefined) throw new TypeError("invalid compiler s3 client");
    return client;
  };
  const projectUpload = (value: unknown): UploadBox => {
    const box = object(value) ? uploads.get(value) : undefined;
    if (box === undefined) throw new TypeError("invalid compiler s3 upload");
    return box;
  };
  const keyOf = (key: unknown): string | Completion<never> => {
    if (typeof key !== "string") throw new TypeError("invalid compiler s3 key");
    if (key === "") return invalid("key");
    return key;
  };
  const contentType = (value: unknown): string | undefined | Completion<never> => {
    const picked = option(ids.someText, value);
    if (picked === undefined) return undefined;
    // Fail fast with a domain error: the native write would throw a
    // bare TypeError for control characters instead.
    // oxlint-disable-next-line no-control-regex -- Reject C0 controls and DEL in S3 content types.
    if (typeof picked !== "string" || picked === "" || /[\x00-\x1F\x7F]/.test(picked))
      return invalid("content_type");
    return picked;
  };
  const metadataOf = (stat: {
    size: number;
    etag: string;
    type: string;
    lastModified: Date;
  }): unknown =>
    record(ids.metadata, [
      ["size", BigInt(stat.size)],
      ["etag", stat.etag],
      ["content_type", stat.type],
      ["last_modified", mintTimeInstant(BigInt(stat.lastModified.getTime()))],
    ]);
  const statOf = async (client: NativeClient, key: string): Promise<unknown> =>
    metadataOf(await client.file(key).stat());
  const mintClient = (client: NativeClient): object => {
    const handle = Object.freeze(Object.create(null));
    clients.set(handle, client);
    return handle;
  };
  const sinkOf = (box: UploadBox): NativeSink =>
    (box.sink ??= box.client.file(box.key).writer({
      ...(box.type === undefined ? {} : { type: box.type }),
      ...(box.partSize === undefined ? {} : { partSize: box.partSize }),
    }));
  const scrub = async (
    client: NativeClient,
    key: string,
    sink: NativeSink | undefined,
    operation: string,
  ): Promise<Completion<never> | undefined> => {
    // Destructive cleanup converges here (E08; the X-R15-1 cancel
    // branch is negative). The native writer has no abort: an
    // un-ended sink pins the Bun event loop forever (unref and GC do
    // not release it) and close() completes asynchronously, which
    // races a delete. So cleanup completes synchronously with end()
    // and then deletes the key. Consequences, all measured live
    // (E07): the completion transiently overwrites any pre-existing
    // key and is visible to concurrent readers; a failed completion
    // strands multipart state the client never aborts; a failed
    // delete leaves the transient completion in place permanently.
    // The delete still runs after a failed end: on the part-failure
    // path it carries the delete-coupled abort that prevents the
    // orphan. The first failed await is returned so discard_upload
    // reports honestly. One blind spot the adapter cannot see: when
    // the delete-coupled abort faults but the delete lands (E07 F7),
    // the MPU orphans with no in-band signal — detection needs the
    // operator list/abort/re-list recipe.
    if (sink === undefined) return undefined;
    let failed: Completion<never> | undefined;
    try {
      await sink.end();
    } catch (cause) {
      failed = wire(operation, key, cause);
    }
    try {
      await client.file(key).delete();
    } catch (cause) {
      failed ??= wire(operation, key, cause);
    }
    return failed;
  };
  const retire = async (box: UploadBox): Promise<Completion<never> | undefined> => {
    if (box.state !== "open") return undefined;
    box.state = "discarded";
    const sink = box.sink;
    box.sink = undefined;
    return scrub(box.client, box.key, sink, "discard_upload");
  };
  return Object.freeze({
    async clientOpen(
      endpoint: unknown,
      region: unknown,
      bucket: unknown,
      accessKey: unknown,
      secretKey: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<object>> {
      if (
        typeof endpoint !== "string" ||
        typeof region !== "string" ||
        typeof bucket !== "string" ||
        typeof accessKey !== "string" ||
        typeof secretKey !== "string"
      )
        throw new TypeError("invalid compiler s3 client");
      try {
        const parsed = new URL(endpoint);
        if (parsed.protocol !== "http:" && parsed.protocol !== "https:") return invalid("endpoint");
      } catch {
        return invalid("endpoint");
      }
      if (region === "") return invalid("region");
      if (bucket === "") return invalid("bucket");
      if (accessKey === "" || secretKey === "") return invalid("credentials");
      try {
        return success(
          mintClient(
            new Bun.S3Client({
              endpoint,
              region,
              bucket,
              accessKeyId: accessKey,
              secretAccessKey: secretKey,
            }),
          ),
        );
      } catch (cause) {
        return caught(cause, origin);
      }
    },
    async readBytes(
      handle: unknown,
      key: unknown,
      maxBytes: unknown,
      context?: AssertionContext,
      bounds?: S3Bounds,
    ): Promise<Completion<unknown>> {
      denyLiveBoundary(context, origin);
      const client = projectClient(handle);
      const checked = keyOf(key);
      if (typeof checked !== "string") return checked;
      if (typeof maxBytes !== "bigint" || maxBytes < 1n || maxBytes > READ_CAP)
        return invalid("limit");
      const file = client.file(checked);
      try {
        const statRaced = await raceS3(undefined, bounds, () => file.stat());
        if (statRaced.raced.kind === "unknown") return service("timeout", "read_bytes");
        const stat = statRaced.raced.value;
        if (BigInt(stat.size) > maxBytes) return overLimit(maxBytes, BigInt(stat.size));
        const bytesRaced = await raceS3(undefined, bounds, () => file.bytes());
        if (bytesRaced.raced.kind === "unknown") return service("timeout", "read_bytes");
        return success(ownBytes(new Uint8Array(bytesRaced.raced.value)));
      } catch (cause) {
        return wire("read_bytes", checked, cause);
      }
    },
    async readRange(
      handle: unknown,
      key: unknown,
      offset: unknown,
      length: unknown,
      context?: AssertionContext,
      bounds?: S3Bounds,
    ): Promise<Completion<unknown>> {
      denyLiveBoundary(context, origin);
      const client = projectClient(handle);
      const checked = keyOf(key);
      if (typeof checked !== "string") return checked;
      if (typeof offset !== "bigint" || offset < 0n || offset > SAFE_MAX) return invalid("offset");
      if (typeof length !== "bigint" || length < 0n || length > READ_CAP) return invalid("length");
      // Zero-length ranges answer without a wire call: the native
      // slice(n,n) yields the single byte at n instead of empty.
      if (length === 0n) return success(ownBytes(new Uint8Array(0)));
      try {
        const window = client.file(checked).slice(Number(offset), Number(offset + length));
        const raced = await raceS3(undefined, bounds, () => window.arrayBuffer());
        if (raced.raced.kind === "unknown") return service("timeout", "read_range");
        return success(ownBytes(new Uint8Array(raced.raced.value)));
      } catch (cause) {
        return wire("read_range", checked, cause);
      }
    },
    async readStream(
      handle: unknown,
      key: unknown,
      maxBytes: unknown,
      context?: AssertionContext,
      bounds?: S3Bounds,
    ): Promise<Completion<object>> {
      denyLiveBoundary(context, origin);
      const client = projectClient(handle);
      const checked = keyOf(key);
      if (typeof checked !== "string") return checked;
      if (typeof maxBytes !== "bigint" || maxBytes < 1n || maxBytes > SAFE_MAX)
        return invalid("max_bytes");
      try {
        const file = client.file(checked);
        const raced = await raceS3(undefined, bounds, () => file.stat());
        if (raced.raced.kind === "unknown") return service("timeout", "read_stream");
        const stat = raced.raced.value;
        if (BigInt(stat.size) > maxBytes) return overLimit(maxBytes, BigInt(stat.size));
        // The eager stat bounds the known size; the counting wrapper
        // guards against growth mid-read and fails the stream with a
        // code the stream contract reports as read_failed.
        let seen = 0n;
        const counted = (file.stream() as ReadableStream<Uint8Array>).pipeThrough(
          new TransformStream<Uint8Array, Uint8Array>({
            transform(chunk, controller) {
              seen += BigInt(chunk.byteLength);
              if (seen > maxBytes) {
                controller.error(
                  Object.assign(new Error("s3 over byte budget"), { code: "over_limit" }),
                );
                return;
              }
              controller.enqueue(chunk);
            },
          }),
        );
        return success(
          registerReader(openByteCell(counted, STREAM_CHUNK), fail, ids.closeFailed, {
            scopeManaged: true,
          }),
        );
      } catch (cause) {
        return wire("read_stream", checked, cause);
      }
    },
    async writeBytes(
      handle: unknown,
      key: unknown,
      body: unknown,
      options: unknown,
      context?: AssertionContext,
      bounds?: S3Bounds,
    ): Promise<Completion<unknown>> {
      denyLiveBoundary(context, origin);
      const client = projectClient(handle);
      const checked = keyOf(key);
      if (typeof checked !== "string") return checked;
      if (!isBytes(body)) throw new TypeError("invalid compiler s3 body");
      const type = contentType(dataProperty(options, "content_type"));
      if (typeof type !== "string" && type !== undefined) return type;
      try {
        const put = await raceS3(undefined, bounds, () =>
          client
            .file(checked)
            .write(copyBytes(body, origin), type === undefined ? undefined : { type }),
        );
        if (put.raced.kind === "unknown") return service("timeout", "write_bytes");
        const stat = await raceS3(undefined, bounds, () => statOf(client, checked));
        if (stat.raced.kind === "unknown") return service("timeout", "write_bytes");
        return success(stat.raced.value);
      } catch (cause) {
        return wire("write_bytes", checked, cause);
      }
    },
    async writeStream(
      handle: unknown,
      key: unknown,
      reader: unknown,
      options: unknown,
      maxBytes: unknown,
      deadlineMs: unknown,
      context?: AssertionContext,
      bounds?: S3Bounds,
    ): Promise<Completion<unknown>> {
      denyLiveBoundary(context, origin);
      const client = projectClient(handle);
      const checked = keyOf(key);
      if (typeof checked !== "string") return checked;
      const type = contentType(dataProperty(options, "content_type"));
      if (typeof type !== "string" && type !== undefined) return type;
      if (typeof maxBytes !== "bigint" || maxBytes < 1n || maxBytes > SAFE_MAX)
        return invalid("max_bytes");
      if (typeof deadlineMs !== "bigint" || deadlineMs < 1n || deadlineMs > DEADLINE_CAP)
        return invalid("deadline");
      // The sink is created lazily on the first chunk: a pump that
      // fails before any byte never touches the service. Every pump
      // await races its layered bound (R1): the deadline remainder
      // plus trailing runtime boundMs, ambient request budget, and
      // the adapter ceiling, cancel-absent with one escalation and
      // late-settlement observation per expiry. Expiry with a live
      // sink converges deferred — the standard destructive scrub
      // runs once the hung native settles — while a pre-sink expiry
      // costs nothing (W5-S5b); an already-expired boundary starts
      // no native and the shared finally below scrubs synchronously.
      let sink: NativeSink | undefined;
      const pump = () =>
        (sink ??= client.file(checked).writer(type === undefined ? undefined : { type }));
      const deadline = Number(deadlineMs);
      const started = Date.now();
      const expired = (): boolean => Date.now() - started >= deadline;
      const remainder = (): number => Math.max(1, deadline - (Date.now() - started));
      const expirePump = <T>(pending: Promise<T> | undefined): Completion<never> => {
        if (pending !== undefined && sink !== undefined) {
          const doomed = sink;
          sink = undefined;
          convergeDeferred(pending, () => scrub(client, checked, doomed, "write_stream"));
        }
        return service("timeout", "write_stream");
      };
      const reasonFor = (cause: unknown): string => {
        if (typeof cause === "object" && cause !== null) {
          if ((cause as { name?: unknown }).name === "AbortError") return "aborted";
          if (typeof (cause as { code?: unknown }).code === "string")
            return (cause as { code: string }).code;
        }
        return "io_error";
      };
      return useReader(reader, async (cell: ReaderCell) => {
        let completed = false;
        try {
          if (cell.item !== "bytes" || cell.reader === undefined) {
            return invalid("reader");
          }
          const source = cell.reader;
          let written = 0n;
          for (;;) {
            if (expired()) {
              return service("timeout", "write_stream");
            }
            let next;
            try {
              const read = await raceS3(remainder(), bounds, () => source.read());
              if (read.raced.kind === "unknown") return expirePump(read.pending);
              next = read.raced.value;
            } catch (cause) {
              cell.errored = true;
              cell.carry = undefined;
              return fail(ids.readFailed, [["reason", reasonFor(cause)]], cause);
            }
            if (next.done) {
              if (cell.cancelled !== undefined) {
                return fail(ids.cancelled, [["reason", cell.cancelled]]);
              }
              break;
            }
            if (next.value.byteLength === 0) continue;
            const length = BigInt(next.value.byteLength);
            if (written + length > maxBytes) {
              return overLimit(maxBytes, written + length);
            }
            try {
              const active = pump();
              const writtenOut = await raceS3(remainder(), bounds, () =>
                active.write(new Uint8Array(next.value)),
              );
              if (writtenOut.raced.kind === "unknown") return expirePump(writtenOut.pending);
              const flushed = await raceS3(remainder(), bounds, () => active.flush());
              if (flushed.raced.kind === "unknown") return expirePump(flushed.pending);
            } catch (cause) {
              return wire("write_stream", checked, cause);
            }
            written += length;
          }
          try {
            const ended = await raceS3(remainder(), bounds, () => pump().end());
            if (ended.raced.kind === "unknown") return expirePump(ended.pending);
            completed = true;
            const stat = await raceS3(remainder(), bounds, () => statOf(client, checked));
            if (stat.raced.kind === "unknown") return service("timeout", "write_stream");
            return success(stat.raced.value);
          } catch (cause) {
            return wire("write_stream", checked, cause);
          }
        } finally {
          const doomed = sink;
          sink = undefined;
          // Best-effort: the pump's own outcome above is the reported
          // one; a failed scrub here cannot change it.
          if (!completed) await scrub(client, checked, doomed, "write_stream");
        }
      });
    },
    async stat(
      handle: unknown,
      key: unknown,
      context?: AssertionContext,
      bounds?: S3Bounds,
    ): Promise<Completion<unknown>> {
      denyLiveBoundary(context, origin);
      const client = projectClient(handle);
      const checked = keyOf(key);
      if (typeof checked !== "string") return checked;
      try {
        const raced = await raceS3(undefined, bounds, () => statOf(client, checked));
        if (raced.raced.kind === "unknown") return service("timeout", "stat");
        return success(raced.raced.value);
      } catch (cause) {
        return wire("stat", checked, cause);
      }
    },
    async exists(
      handle: unknown,
      key: unknown,
      context?: AssertionContext,
      bounds?: S3Bounds,
    ): Promise<Completion<boolean>> {
      denyLiveBoundary(context, origin);
      const client = projectClient(handle);
      const checked = keyOf(key);
      if (typeof checked !== "string") return checked;
      try {
        const raced = await raceS3(undefined, bounds, () => client.file(checked).exists());
        if (raced.raced.kind === "unknown") return service("timeout", "exists");
        return success(raced.raced.value);
      } catch (cause) {
        // Absent keys answer false; every other fault keeps its shape.
        if (s3Code(cause) === "NoSuchKey") return success(false);
        return wire("exists", checked, cause);
      }
    },
    async remove(
      handle: unknown,
      key: unknown,
      context?: AssertionContext,
      bounds?: S3Bounds,
    ): Promise<Completion<void>> {
      denyLiveBoundary(context, origin);
      const client = projectClient(handle);
      const checked = keyOf(key);
      if (typeof checked !== "string") return checked;
      try {
        const raced = await raceS3(undefined, bounds, () => client.file(checked).delete());
        if (raced.raced.kind === "unknown") return service("timeout", "delete");
        return success(undefined);
      } catch (cause) {
        return wire("delete", checked, cause);
      }
    },
    async list(
      handle: unknown,
      options: unknown,
      context?: AssertionContext,
      bounds?: S3Bounds,
    ): Promise<Completion<unknown>> {
      denyLiveBoundary(context, origin);
      const client = projectClient(handle);
      const prefix = dataProperty(options, "prefix"),
        limit = dataProperty(options, "limit");
      if (typeof prefix !== "string") throw new TypeError("invalid compiler s3 list options");
      if (typeof limit !== "bigint" || limit < 1n || limit > SAFE_MAX) return invalid("max_keys");
      const delimiterPick = option(ids.someText, dataProperty(options, "delimiter"));
      if (
        delimiterPick !== undefined &&
        (typeof delimiterPick !== "string" || delimiterPick === "")
      )
        return invalid("delimiter");
      const continuationPick = option(ids.someContinuation, dataProperty(options, "continuation"));
      let continuation: string | undefined;
      if (continuationPick !== undefined) {
        continuation = object(continuationPick) ? continuations.get(continuationPick) : undefined;
        if (continuation === undefined) throw new TypeError("invalid compiler s3 continuation");
      }
      try {
        const raced = await raceS3(undefined, bounds, () =>
          client.list({
            prefix,
            maxKeys: Number(limit),
            ...(delimiterPick === undefined ? {} : { delimiter: delimiterPick }),
            ...(continuation === undefined ? {} : { continuationToken: continuation }),
          }),
        );
        if (raced.raced.kind === "unknown") return service("timeout", "list");
        const response: NativeList = raced.raced.value;
        const entries: unknown[] = [];
        for (const item of response.contents ?? []) {
          // Size, etag and timestamp always accompany real entries;
          // anything else is a malformed service response, not data.
          if (
            typeof item.key !== "string" ||
            typeof item.size !== "number" ||
            typeof item.eTag !== "string" ||
            typeof item.lastModified !== "string"
          )
            return service("malformed_listing", "list");
          const millis = Date.parse(item.lastModified);
          if (Number.isNaN(millis)) return service("malformed_listing", "list");
          entries.push(
            record(ids.entry, [
              ["key", item.key],
              ["size", BigInt(item.size)],
              ["etag", item.eTag],
              ["last_modified", mintTimeInstant(BigInt(millis))],
            ]),
          );
        }
        const prefixes = (response.commonPrefixes ?? []).map((group) => group.prefix);
        const token = response.isTruncated === true ? response.nextContinuationToken : undefined;
        let next: unknown = record(ids.none, []);
        if (typeof token === "string" && token !== "") {
          const handle = Object.freeze(Object.create(null));
          continuations.set(handle, token);
          next = record(ids.someContinuation, [["value", handle]]);
        }
        return success(
          record(ids.page, [
            ["entries", array(entries)],
            ["prefixes", array(prefixes)],
            ["truncated", response.isTruncated === true],
            ["continuation", next],
          ]),
        );
      } catch (cause) {
        return wire("list", prefix, cause);
      }
    },
    async presign(
      handle: unknown,
      method: unknown,
      key: unknown,
      expiresIn: unknown,
      contentTypePick: unknown,
      context?: AssertionContext,
    ): Promise<Completion<object>> {
      denyLiveBoundary(context, origin);
      const client = projectClient(handle);
      const verb = methodOf(method);
      const checked = keyOf(key);
      if (typeof checked !== "string") return checked;
      if (typeof expiresIn !== "bigint" || expiresIn < 1n || expiresIn > EXPIRES_MAX)
        return invalid("expires");
      const type = contentType(contentTypePick);
      if (typeof type !== "string" && type !== undefined) return type;
      try {
        const url = client.presign(checked, {
          method: verb,
          expiresIn: Number(expiresIn),
          ...(type === undefined ? {} : { type }),
        });
        const token = Object.freeze(Object.create(null));
        presigned.set(token, {
          url,
          method: verb,
          expires: mintTimeInstant(BigInt(Date.now()) + expiresIn * 1000n),
        });
        return success(token);
      } catch (cause) {
        return caught(cause, origin);
      }
    },
    async describe(token: unknown, _context?: AssertionContext): Promise<Completion<unknown>> {
      const box = object(token) ? presigned.get(token) : undefined;
      if (box === undefined) throw new TypeError("invalid compiler s3 presigned handle");
      return success(
        record(ids.info, [
          ["url", box.url],
          ["method", record(methodLeaf(box.method), [])],
          ["expires_at", box.expires],
        ]),
      );
    },
    async beginUpload(
      handle: unknown,
      key: unknown,
      options: unknown,
      context?: AssertionContext,
    ): Promise<Completion<object>> {
      denyLiveBoundary(context, origin);
      const client = projectClient(handle);
      const checked = keyOf(key);
      if (typeof checked !== "string") return checked;
      const type = contentType(dataProperty(options, "content_type"));
      if (typeof type !== "string" && type !== undefined) return type;
      const partPick = option(ids.someInt, dataProperty(options, "part_size"));
      if (
        partPick !== undefined &&
        (typeof partPick !== "bigint" || partPick < PART_MIN || partPick > PART_MAX)
      )
        return invalid("part_size");
      const box: UploadBox = {
        client,
        key: checked,
        type,
        partSize: partPick === undefined ? undefined : Number(partPick),
        sink: undefined,
        state: "open",
        scrubPending: false,
      };
      const token = registerResource(
        UPLOAD_RESOURCE_KIND,
        box,
        async (): Promise<Completion<void>> => {
          // Abandoned open uploads discard destructively at scope
          // drain; a failed scrub surfaces through cleanupFailed.
          const failed = await retire(box);
          if (failed !== undefined) return failed;
          return success(undefined);
        },
        { scopeManaged: true },
      );
      uploads.set(token, box);
      return success(token);
    },
    async uploadWrite(
      token: unknown,
      chunk: unknown,
      context?: AssertionContext,
      bounds?: S3Bounds,
    ): Promise<Completion<bigint>> {
      denyLiveBoundary(context, origin);
      const box = projectUpload(token);
      if (box.state !== "open") return closed("upload_write", box.state);
      if (!isBytes(chunk)) throw new TypeError("invalid compiler s3 chunk");
      try {
        const bytes = copyBytes(chunk, origin);
        const sink = sinkOf(box);
        const written = await raceS3(undefined, bounds, () => sink.write(bytes));
        if (written.raced.kind === "unknown") {
          poisonUpload(box, written.pending, "upload_write");
          return service("timeout", "upload_write");
        }
        const flushed = await raceS3(undefined, bounds, () => sink.flush());
        if (flushed.raced.kind === "unknown") {
          poisonUpload(box, flushed.pending, "upload_write");
          return service("timeout", "upload_write");
        }
        // The adapter guards the native silent drop after end: no
        // terminal write ever reaches the sink, so a write that
        // returns buffered the whole chunk or threw.
        return success(BigInt(bytes.byteLength));
      } catch (cause) {
        return wire("upload_write", box.key, cause);
      }
    },
    async uploadFinish(
      token: unknown,
      context?: AssertionContext,
      bounds?: S3Bounds,
    ): Promise<Completion<unknown>> {
      denyLiveBoundary(context, origin);
      const box = projectUpload(token);
      if (box.state !== "open") return closed("upload_finish", box.state);
      try {
        const ended = await raceS3(undefined, bounds, () => sinkOf(box).end());
        if (ended.raced.kind === "unknown") {
          poisonUpload(box, ended.pending, "upload_finish");
          return service("timeout", "upload_finish");
        }
        box.state = "finished";
        const stat = await raceS3(undefined, bounds, () => statOf(box.client, box.key));
        if (stat.raced.kind === "unknown") return service("timeout", "upload_finish");
        return success(stat.raced.value);
      } catch (cause) {
        // A failed completion leaves the handle open: the caller may
        // retry the finish or discard explicitly.
        return wire("upload_finish", box.key, cause);
      }
    },
    async discardUpload(token: unknown, context?: AssertionContext): Promise<Completion<void>> {
      denyLiveBoundary(context, origin);
      const box = projectUpload(token);
      if (box.state !== "open") return closed("discard_upload", box.state);
      const failed = await retire(box);
      if (failed !== undefined) return failed;
      return success(undefined);
    },
  });
}
