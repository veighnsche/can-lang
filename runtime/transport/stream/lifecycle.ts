// Terminal/cancel/cleanup protocol for pull streams. The owner keeps the
// open/closing/closed states and exactly-once terminal publication; this
// module records cancel reasons for interrupted operations, releases native
// locks exactly once, and maps native terminal failures onto declared
// stream failures. State transitions: open -> closing -> closed. End of
// stream is sticky (further reads return the empty batch) until close.
// A native failure marks the cell errored: the failing operation reports
// the domain failure, later operations observe resource-state, and close
// still succeeds so failure arms can clean up. Cancel is terminal and
// records its reason: an operation interrupted by cancel reports
// cancelled instead of a partial result, and later operations observe
// resource-state. Close twice and use-after-close/foreign-owner surface
// as standard resource-state failures, exactly like every other owner
// violation; they are never domain failures.
import { success, type Completion } from "../../completion.ts";
import { resourceStateFailure, type FailureOrigin } from "../../failure.ts";
import { closeResource, registerResource, resourceStatus, useResource } from "../../owner.ts";
export const READER_KIND = "stream-reader";
export const WRITER_KIND = "stream-writer";
// Bounded shutdown for explicit close/cancel when another lease is held.
export const STREAM_CLOSE_MS = 5000;
const origin: FailureOrigin = Object.freeze({
  source: "can:stream-lifecycle",
  start: 0,
  end: 0,
  invocation: Object.freeze([]),
});
export type Fail = (
  identity: string,
  fields: readonly (readonly [string, unknown])[],
  cause?: unknown,
) => Completion<never>;
export type ByteSource = ReadableStream<Uint8Array>;
export type LineFraming = { decoder: TextDecoder; carry: string; pendingBytes: bigint };
// Event cells serve pre-framed producer values (WebSocket session events)
// instead of pulling a byte stream. The pump is producer-owned: take
// resolves queued values in order, end marks the terminal, failed carries
// a read_failed reason, and interrupted answers cancel only. Dispose runs
// after every lease drains, so no take is ever pending inside dispose.
export type EventTake = Readonly<
  | { kind: "event"; value: unknown }
  | { kind: "end" }
  | { kind: "failed"; reason: string }
  | { kind: "interrupted" }
>;
export type EventPump = Readonly<{
  take: () => Promise<EventTake>;
  interrupt: () => void;
  dispose: () => void;
}>;
export type ReaderCell = {
  reader?: ReadableStreamDefaultReader<Uint8Array>;
  item: "bytes" | "text" | "events";
  maxChunk: bigint;
  maxLine: bigint;
  cancelled?: string;
  errored?: boolean;
  ended?: boolean;
  carry?: Uint8Array;
  framing?: LineFraming;
  events?: EventPump;
};
export type SinkLike = { write(chunk: Uint8Array): unknown; flush(): unknown; end(): unknown };
export type WriterCell = { sink: SinkLike; cancelled?: string };
const cells = new WeakMap<object, ReaderCell | WriterCell>();
function cellFor(token: unknown): ReaderCell | WriterCell {
  if (
    (typeof token !== "object" && typeof token !== "function") ||
    token === null ||
    !cells.has(token)
  )
    throw resourceStateFailure(undefined, origin);
  return cells.get(token)!;
}
export function readerCell(token: unknown): ReaderCell {
  const cell = cellFor(token);
  if (!("reader" in cell) && !("events" in cell)) throw resourceStateFailure(undefined, origin);
  return cell as ReaderCell;
}
export function writerCell(token: unknown): WriterCell {
  const cell = cellFor(token);
  if (!("sink" in cell)) throw resourceStateFailure(undefined, origin);
  return cell;
}
export function isStreamHandleValue(kind: string | undefined, value: unknown): boolean {
  if (kind !== READER_KIND && kind !== WRITER_KIND) return false;
  try {
    return resourceStatus(value).kind === kind;
  } catch {
    return false;
  }
}
async function releaseReader(cell: ReaderCell, errored: boolean): Promise<void> {
  if (cell.events !== undefined) {
    cell.events.dispose();
    return;
  }
  if (cell.reader === undefined) throw resourceStateFailure(undefined, origin);
  // Observe cancellation rejection without replacing the original outcome.
  if (!errored)
    try {
      await cell.reader.cancel();
    } catch {}
  cell.reader.releaseLock();
}
export function registerReader(
  cell: ReaderCell,
  fail: Fail,
  closeFailed: string,
  options: Readonly<{ idempotent?: boolean; scopeManaged?: boolean }> = {},
): object {
  const token = registerResource(
    READER_KIND,
    cell,
    async (): Promise<Completion<void>> => {
      try {
        await releaseReader(cell, cell.errored === true);
        return success(undefined);
      } catch (cause) {
        return fail(closeFailed, [["reason", "close"]], cause);
      }
    },
    options,
  );
  cells.set(token, cell);
  return token;
}
export function registerWriter(
  cell: WriterCell,
  fail: Fail,
  closeFailed: string,
  options: Readonly<{ idempotent?: boolean; scopeManaged?: boolean }> = {},
): object {
  const token = registerResource(
    WRITER_KIND,
    cell,
    async (): Promise<Completion<void>> => {
      try {
        await cell.sink.flush();
        await cell.sink.end();
        return success(undefined);
      } catch (cause) {
        return fail(closeFailed, [["reason", "close"]], cause);
      }
    },
    options,
  );
  cells.set(token, cell);
  return token;
}
function closeDeadline(
  fail: Fail,
  closeFailed: string,
): { milliseconds: number; failure: () => Completion<void> } {
  return {
    milliseconds: STREAM_CLOSE_MS,
    failure: () => fail(closeFailed, [["reason", "close_timeout"]]),
  };
}
export async function closeHandle(
  token: unknown,
  kind: string,
  fail: Fail,
  closeFailed: string,
): Promise<Completion<void>> {
  cellFor(token);
  return closeResource(token, kind, closeDeadline(fail, closeFailed));
}
// Cancel records its reason before closing so an interrupted operation
// reports cancelled instead of a partial result. Cancel is terminal:
// the handle closes and later operations observe resource-state. Owner
// close waits for live leases, so a precancel hook first unblocks any
// in-flight native wait (a pending read settles done); the owner close
// callback still performs the authoritative terminal release.
export async function cancelHandle(
  token: unknown,
  kind: string,
  reason: string,
  fail: Fail,
  closeFailed: string,
  precancel?: (cell: ReaderCell | WriterCell) => unknown,
): Promise<Completion<void>> {
  const cell = cellFor(token);
  if (cell.cancelled === undefined) cell.cancelled = reason;
  if (precancel !== undefined)
    try {
      await precancel(cell);
    } catch {}
  return closeResource(token, kind, closeDeadline(fail, closeFailed));
}
export async function useReader<T>(
  token: unknown,
  operation: (cell: ReaderCell) => Completion<T> | Promise<Completion<T>>,
): Promise<Completion<T>> {
  return useResource(token, READER_KIND, async (native: unknown) => {
    const cell = native as ReaderCell;
    if (cell.errored === true) throw resourceStateFailure(undefined, origin);
    return operation(cell);
  });
}
export async function useWriter<T>(
  token: unknown,
  operation: (cell: WriterCell) => Completion<T> | Promise<Completion<T>>,
): Promise<Completion<T>> {
  return useResource(token, WRITER_KIND, async (native: unknown) =>
    operation(native as WriterCell),
  );
}
