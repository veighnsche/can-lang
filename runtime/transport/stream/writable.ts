// Owned writer and backpressure. Each write reports the accepted byte
// count and retries stay with the caller: short writes are visible flow
// control, not adapter-held buffering. No coalescing queue sits between
// the caller and the sink; flush and end run at close. Payload bytes are
// copied out of immutable values before crossing, so a later native
// mutation can never reach Can state.
import { denyLiveBoundary, type AssertionContext } from "../../assert/context.ts";
import { failure, success, type Completion } from "../../completion.ts";
import { record } from "../../data.ts";
import { copyBytes } from "../../bytes.ts";
import { createDomainRuntime } from "../../domain.ts";
import type { FailureOrigin } from "../../failure.ts";
import {
  WRITER_KIND,
  registerWriter,
  useWriter,
  closeHandle,
  cancelHandle,
  type Fail,
  type WriterCell,
  type SinkLike,
} from "./lifecycle.ts";
const origin: FailureOrigin = Object.freeze({
  source: "can:stream-write",
  start: 0,
  end: 0,
  invocation: Object.freeze([]),
});
type Contracts = Readonly<{ writeFailed: string; closeFailed: string }>;
function acceptedOf(result: unknown): bigint | undefined {
  if (typeof result === "number" && Number.isSafeInteger(result) && result >= 0)
    return BigInt(result);
  return undefined;
}
export function createStreamWrites(
  domain: ReturnType<typeof createDomainRuntime>,
  types: Contracts,
) {
  const fail: Fail = (identity, fields, cause) =>
    failure(domain.create(identity, record(identity, fields), origin, cause));
  return Object.freeze({
    async writeSome(
      token: unknown,
      chunk: unknown,
      context?: AssertionContext,
    ): Promise<Completion<bigint>> {
      denyLiveBoundary(context, origin);
      const bytes = copyBytes(chunk, origin);
      return useWriter(token, async (cell) => {
        let result: unknown;
        try {
          result = await cell.sink.write(bytes);
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
        const accepted = acceptedOf(result);
        if (accepted === undefined || accepted > BigInt(bytes.byteLength))
          return fail(types.writeFailed, [["reason", "bad_accepted"]]);
        return success(accepted);
      });
    },
    async closeWriter(token: unknown, context?: AssertionContext): Promise<Completion<void>> {
      denyLiveBoundary(context, origin);
      return closeHandle(token, WRITER_KIND, fail, types.closeFailed);
    },
    async cancelWriter(
      token: unknown,
      reason: string,
      context?: AssertionContext,
    ): Promise<Completion<void>> {
      denyLiveBoundary(context, origin);
      return cancelHandle(token, WRITER_KIND, reason, fail, types.closeFailed);
    },
  });
}
export function openSinkCell(sink: SinkLike): WriterCell {
  return { sink };
}
export function registerSink(sink: SinkLike, fail: Fail, closeFailed: string): object {
  return registerWriter(openSinkCell(sink), fail, closeFailed);
}
