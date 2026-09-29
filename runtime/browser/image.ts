import { failure, type AssertionContext, type Completion } from "../completion.ts";
import { record } from "../data.ts";
import { createDomainRuntime } from "../domain.ts";
import { copyBytes, type Bytes } from "../bytes.ts";
import type { FailureOrigin } from "../failure.ts";
type ImageMetadata = Readonly<{ format: string; width: bigint; height: bigint }>;

const origin: FailureOrigin = Object.freeze({
  source: "can:image",
  start: 0,
  end: 0,
  invocation: Object.freeze([]),
});

export function createImage(
  domain: ReturnType<typeof createDomainRuntime>,
  invalidImage: string,
  _metadataIdentity: string,
) {
  return Object.freeze({
    async inspect(
      value: Bytes,
      _maxPixels: bigint,
      _context?: AssertionContext,
    ): Promise<Completion<ImageMetadata>> {
      copyBytes(value, origin);
      return failure(
        domain.create(
          invalidImage,
          record(invalidImage, [["reason", "unsupported_platform"]]),
          origin,
        ),
      );
    },
  });
}
