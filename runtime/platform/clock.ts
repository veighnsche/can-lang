import { denyLiveBoundary, type AssertionContext } from "../assert/context.ts";
import { success, failure, type Completion } from "../completion.ts";
import { record } from "../data.ts";
import { createDomainRuntime } from "../domain.ts";
const origin = Object.freeze({
  source: "can:clock",
  start: 0,
  end: 0,
  invocation: Object.freeze([]),
});
export function createClock(
  domain: ReturnType<typeof createDomainRuntime>,
  invalidDuration: string,
) {
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
      await Bun.sleep(Number(milliseconds));
      return success(undefined);
    },
  });
}
