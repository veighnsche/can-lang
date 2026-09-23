import { types as nativeTypes } from "node:util";
import { denyLiveBoundary, type AssertionContext } from "../assert/context.ts";
import { success, failure, type Completion } from "../completion.ts";
import { record } from "../data.ts";
import { ownBytes, copyBytes, type Bytes } from "../bytes.ts";
import { createDomainRuntime } from "../domain.ts";
import type { FailureOrigin } from "../failure.ts";
const origin: FailureOrigin = Object.freeze({
  source: "can:io",
  start: 0,
  end: 0,
  invocation: Object.freeze([]),
});
const codes = new Set([
  "EPIPE",
  "EBADF",
  "EIO",
  "ENOSPC",
  "EACCES",
  "EINVAL",
  "EFBIG",
  "EROFS",
  "EINTR",
  "EAGAIN",
  "EISDIR",
  "ENXIO",
]);
export function expectedIOFailure(cause: unknown): boolean {
  if (
    cause === null ||
    typeof cause !== "object" ||
    nativeTypes.isProxy(cause) ||
    !nativeTypes.isNativeError(cause)
  )
    return false;
  const code = Object.getOwnPropertyDescriptor(cause, "code");
  return (
    code !== undefined && "value" in code && typeof code.value === "string" && codes.has(code.value)
  );
}
type Contracts = Readonly<{ readFailed: string; limit: string; invalidData: string }>;
export function createIO(
  domain: ReturnType<typeof createDomainRuntime>,
  types: Contracts,
  input: () => ReadableStream<Uint8Array> = () => Bun.stdin.stream(),
) {
  const fail = (
    identity: string,
    fields: readonly (readonly [string, unknown])[],
    cause?: unknown,
  ) => failure(domain.create(identity, record(identity, fields), origin, cause));
  async function read(limit: bigint, operation: string): Promise<Completion<Bytes>> {
    if (limit < 0n) return fail(types.limit, [["limit", limit]]);
    let reader: ReadableStreamDefaultReader<Uint8Array> | undefined;
    let complete = false;
    try {
      reader = input().getReader();
      const chunks: Uint8Array[] = [];
      let size = 0n;
      for (;;) {
        const next = await reader.read();
        if (next.done) {
          complete = true;
          break;
        }
        const length = BigInt(next.value.byteLength);
        if (length > limit - size) return fail(types.limit, [["limit", limit]]);
        size += length;
        // The stream owns its views and may reuse them after the next read.
        if (length !== 0n) chunks.push(new Uint8Array(next.value));
      }
      const result = new Uint8Array(Number(size));
      let offset = 0;
      for (const chunk of chunks) {
        result.set(chunk, offset);
        offset += chunk.byteLength;
      }
      return success(ownBytes(result));
    } catch (cause) {
      if (!expectedIOFailure(cause)) throw cause;
      return fail(types.readFailed, [["operation", operation]], cause);
    } finally {
      if (reader) {
        // Observe cancellation rejection without replacing the original outcome.
        if (!complete)
          try {
            await reader.cancel();
          } catch {}
        reader.releaseLock();
      }
    }
  }
  return Object.freeze({
    async stdinBytes(limit: bigint, context?: AssertionContext): Promise<Completion<Bytes>> {
      denyLiveBoundary(context, origin);
      return read(limit, "stdin_bytes");
    },
    async stdinText(limit: bigint, context?: AssertionContext): Promise<Completion<string>> {
      denyLiveBoundary(context, origin);
      const result = await read(limit, "stdin_text");
      if (result.kind !== "ok") return result;
      const data = copyBytes(result.value, origin);
      try {
        return success(new TextDecoder("utf-8", { fatal: true, ignoreBOM: true }).decode(data));
      } catch (cause) {
        if (!(cause instanceof TypeError)) throw cause;
        return fail(types.invalidData, [
          ["path", ""],
          ["reason", "utf8"],
        ]);
      }
    },
  });
}
