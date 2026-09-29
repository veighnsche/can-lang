import { failure, success, type AssertionContext, type Completion } from "./completion.ts";
import { record } from "./data.ts";
import { createDomainRuntime } from "./domain.ts";
import { copyBytes, type Bytes } from "./bytes.ts";
import type { FailureOrigin } from "./failure.ts";

export type ImageMetadata = Readonly<{ format: string; width: bigint; height: bigint }>;

const origin: FailureOrigin = Object.freeze({
  source: "can:image",
  start: 0,
  end: 0,
  invocation: Object.freeze([]),
});

const maximumPixels = 0x3fff * 0x3fff;
const pngSignature = [0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a];

function animatedPNG(bytes: Uint8Array): boolean {
  if (bytes.length < 8 || !pngSignature.every((byte, index) => bytes[index] === byte)) return false;
  const view = new DataView(bytes.buffer, bytes.byteOffset, bytes.byteLength);
  let offset = 8;
  while (offset + 12 <= bytes.length) {
    const length = view.getUint32(offset);
    const end = offset + 12 + length;
    if (end > bytes.length) return false;
    const kind = String.fromCharCode(...bytes.subarray(offset + 4, offset + 8));
    if (kind === "acTL") return true;
    if (kind === "IDAT") return false;
    offset = end;
  }
  return false;
}

function animatedWebP(bytes: Uint8Array): boolean {
  if (
    bytes.length < 12 ||
    String.fromCharCode(...bytes.subarray(0, 4)) !== "RIFF" ||
    String.fromCharCode(...bytes.subarray(8, 12)) !== "WEBP"
  )
    return false;
  const view = new DataView(bytes.buffer, bytes.byteOffset, bytes.byteLength);
  let offset = 12;
  while (offset + 8 <= bytes.length) {
    const kind = String.fromCharCode(...bytes.subarray(offset, offset + 4));
    const length = view.getUint32(offset + 4, true);
    const end = offset + 8 + length;
    if (end > bytes.length) return false;
    if (kind === "ANIM" || kind === "ANMF") return true;
    if (kind === "VP8X" && length >= 1 && (bytes[offset + 8]! & 0x02) !== 0) return true;
    offset = end + (length & 1);
  }
  return false;
}

export function createImage(
  domain: ReturnType<typeof createDomainRuntime>,
  invalidImage: string,
  metadataIdentity: string,
) {
  function invalid(reason: string): Completion<ImageMetadata> {
    return failure(domain.create(invalidImage, record(invalidImage, [["reason", reason]]), origin));
  }

  return Object.freeze({
    async inspect(
      value: Bytes,
      maxPixels: bigint,
      _context?: AssertionContext,
    ): Promise<Completion<ImageMetadata>> {
      if (typeof maxPixels !== "bigint" || maxPixels <= 0n) return invalid("pixel_bound");
      const bytes = copyBytes(value, origin);
      if (animatedPNG(bytes) || animatedWebP(bytes)) return invalid("animated");
      const boundedPixels = Number(
        maxPixels > BigInt(maximumPixels) ? BigInt(maximumPixels) : maxPixels,
      );
      try {
        const metadata = await new Bun.Image(bytes, { maxPixels: boundedPixels }).metadata();
        if (metadata.format !== "jpeg" && metadata.format !== "png" && metadata.format !== "webp")
          return invalid("unsupported_format");
        return success(
          record(metadataIdentity, [
            ["format", metadata.format],
            ["width", BigInt(metadata.width)],
            ["height", BigInt(metadata.height)],
          ]) as ImageMetadata,
        );
      } catch (cause) {
        if (!(cause instanceof Error) || !("code" in cause)) throw cause;
        const code = cause.code;
        if (code === "ERR_IMAGE_TOO_MANY_PIXELS") return invalid("too_many_pixels");
        if (
          code === "ERR_IMAGE_UNKNOWN_FORMAT" ||
          code === "ERR_IMAGE_FORMAT_UNSUPPORTED" ||
          code === "ERR_IMAGE_DECODE_FAILED"
        )
          return invalid("malformed_or_unsupported");
        throw cause;
      }
    },
  });
}
