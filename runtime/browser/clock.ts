// Browser-profile clock. Wall and monotonic reads are already portable;
// sleeping uses the native timer instead of Bun.sleep, with the same
// duration validation and live-boundary guard as the canonical module.
import { denyLiveBoundary, type AssertionContext } from "../assert/context.ts";
import { success, failure, type Completion } from "../completion.ts";
import { record } from "../data.ts";
import type { DomainRuntime } from "../domain-core.ts";

const origin = Object.freeze({
  source: "can:clock",
  start: 0,
  end: 0,
  invocation: Object.freeze([]),
});

export function createClock(domain: DomainRuntime, invalidDuration: string) {
  return Object.freeze({
    async wallMillis(context?: AssertionContext): Promise<Completion<bigint>> {
      denyLiveBoundary(context, origin);
      return success(BigInt(Date.now()));
    },
    async monotonicMillis(context?: AssertionContext): Promise<Completion<number>> {
      denyLiveBoundary(context, origin);
      return success(performance.now());
    },
    async sleepMillis(
      milliseconds: bigint,
      context?: AssertionContext,
    ): Promise<Completion<undefined>> {
      denyLiveBoundary(context, origin);
      if (milliseconds < 0n || milliseconds > 2147483647n)
        return failure(
          domain.create(
            invalidDuration,
            record(invalidDuration, [["milliseconds", milliseconds]]),
            origin,
          ),
        );
      await new Promise<void>((resolve) => setTimeout(resolve, Number(milliseconds)));
      return success(undefined);
    },
  });
}
