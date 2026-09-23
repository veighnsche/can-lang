import { denyLiveBoundary, type AssertionContext } from "../assert/context.ts";
import { success, failure, type Completion } from "../completion.ts";
import { record } from "../data.ts";
import { ownBytes, type Bytes } from "../bytes.ts";
import { createDomainRuntime } from "../domain.ts";
const origin = Object.freeze({
  source: "can:random",
  start: 0,
  end: 0,
  invocation: Object.freeze([]),
});
export function createRandom(
  domain: ReturnType<typeof createDomainRuntime>,
  invalidLength: string,
) {
  return Object.freeze({
    async secureBytes(length: bigint, context?: AssertionContext): Promise<Completion<Bytes>> {
      denyLiveBoundary(context, origin);
      if (length < 0n || length > 65536n)
        return failure(
          domain.create(invalidLength, record(invalidLength, [["length", length]]), origin),
        );
      return success(ownBytes(crypto.getRandomValues(new Uint8Array(Number(length)))));
    },
    async uuidV4(context?: AssertionContext): Promise<Completion<string>> {
      denyLiveBoundary(context, origin);
      return success(crypto.randomUUID());
    },
  });
}
