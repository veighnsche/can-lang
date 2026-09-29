import { expect, test } from "bun:test";
import { createHash } from "node:crypto";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, domainFailureDiagnostics, type FailureShape } from "../domain.ts";
import { value, type Completion } from "../completion.ts";
import { ownBytes } from "../bytes.ts";
import { createImage } from "../image.ts";

const identity = (kind: string, declaration: string) =>
  createHash("sha256")
    .update("can-concrete-type-v1\0" + JSON.stringify([kind, declaration]))
    .digest("hex");
const text: FailureShape = {
  identity: identity("primitive", "str"),
  kind: "primitive",
  declaration: "str",
  arguments: [],
  fields: [],
  leaves: [],
  inputs: [],
  errors: [],
};
const integer: FailureShape = {
  identity: identity("primitive", "int"),
  kind: "primitive",
  declaration: "int",
  arguments: [],
  fields: [],
  leaves: [],
  inputs: [],
  errors: [],
};
const errorDeclaration = catalogue.errors.find((entry) => entry.name === "image::invalid_image")!;
const errorShape: FailureShape = {
  identity: identity("error", errorDeclaration.identity),
  kind: "error",
  declaration: errorDeclaration.identity,
  arguments: [],
  fields: [{ name: "reason", type: text.identity }],
  leaves: [],
  inputs: [],
  errors: [],
};
const metadataType = "can.std.image@1::metadata";
const metadataIdentity = identity("record", metadataType);
const metadataShape: FailureShape = {
  identity: metadataIdentity,
  kind: "record",
  declaration: metadataType,
  arguments: [],
  fields: [
    { name: "format", type: text.identity },
    { name: "width", type: integer.identity },
    { name: "height", type: integer.identity },
  ],
  leaves: [],
  inputs: [],
  errors: [],
};
const domain = createDomainRuntime({
  declarations: [
    { identity: errorDeclaration.identity, name: errorDeclaration.name, parameters: 0 },
  ],
  shapes: [text, integer, errorShape, metadataShape],
});
const api = createImage(domain, errorShape.identity, metadataIdentity);
const pixelPNG = Uint8Array.from(
  Buffer.from(
    "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+/+ioAAAAASUVORK5CYII=",
    "base64",
  ),
);

function failure(completion: Completion<unknown>, reason: string) {
  expect(completion.kind).toBe("domain");
  if (completion.kind !== "domain") throw new Error("expected image error");
  const details = domainFailureDiagnostics(completion.value);
  expect(details.declaration.name).toBe("image::invalid_image");
  expect(details.payload).toMatchObject({ reason });
}

test("sniffs actual JPEG, PNG, and WebP bytes and returns dimensions", async () => {
  const png = value(await api.inspect(ownBytes(pixelPNG), 100n));
  expect(png).toMatchObject({ format: "png", width: 1n, height: 1n });
  for (const format of ["jpeg", "webp"] as const) {
    const encoded = await new Bun.Image(pixelPNG)[format]().bytes();
    const metadata = value(await api.inspect(ownBytes(encoded), 100n));
    expect(metadata).toMatchObject({ format, width: 1n, height: 1n });
  }
});

test("rejects malformed, unsupported, over-bound, and animated images", async () => {
  failure(await api.inspect(ownBytes(new Uint8Array([1, 2, 3])), 100n), "malformed_or_unsupported");
  failure(
    await api.inspect(
      ownBytes(new Uint8Array(Buffer.from("GIF89a\x01\x00\x01\x00\x00\x00\x00"))),
      100n,
    ),
    "unsupported_format",
  );
  const twoPixels = await new Bun.Image(pixelPNG).resize(2, 1).png().bytes();
  failure(await api.inspect(ownBytes(twoPixels), 1n), "too_many_pixels");
  failure(await api.inspect(ownBytes(pixelPNG), 0n), "pixel_bound");

  const apngChunk = new Uint8Array([
    0, 0, 0, 8, 0x61, 0x63, 0x54, 0x4c, 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 0,
  ]);
  const apng = new Uint8Array(pixelPNG.length + apngChunk.length);
  apng.set(pixelPNG.slice(0, 33));
  apng.set(apngChunk, 33);
  apng.set(pixelPNG.slice(33), 33 + apngChunk.length);
  failure(await api.inspect(ownBytes(apng), 100n), "animated");

  const webp = new Uint8Array(30);
  webp.set(Buffer.from("RIFF"), 0);
  new DataView(webp.buffer).setUint32(4, 22, true);
  webp.set(Buffer.from("WEBPVP8X"), 8);
  new DataView(webp.buffer).setUint32(16, 10, true);
  webp[20] = 0x02;
  failure(await api.inspect(ownBytes(webp), 100n), "animated");
});
