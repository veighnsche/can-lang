import { test, expect } from "bun:test";
import { createHash } from "node:crypto";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, domainFailureDiagnostics, type FailureShape } from "../domain.ts";
import { createBytes, ownBytes, copyBytes, byteLength, type Bytes } from "../bytes.ts";
import { value, invoke, type Completion } from "../completion.ts";
import { standardFailureKind, isStandardFailure } from "../failure.ts";

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
const declaration = catalogue.errors.find((e) => e.name === "codec::invalid_data")!;
const error: FailureShape = {
  identity: identity("error", declaration.identity),
  kind: "error",
  declaration: declaration.identity,
  arguments: [],
  fields: ["path", "reason"].map((name) => ({ name, type: text.identity })),
  leaves: [],
  inputs: [],
  errors: [],
};
const domain = createDomainRuntime({
  declarations: [{ identity: declaration.identity, name: declaration.name, parameters: 0 }],
  shapes: [text, error],
});
const api = createBytes(domain, error.identity);
const origin = { source: "test:bytes", start: 0, end: 0, invocation: [] };
function invalid(completion: Completion, path: string, reason: string) {
  expect(completion.kind).toBe("domain");
  if (completion.kind !== "domain") throw new Error("expected domain failure");
  const details = domainFailureDiagnostics(completion.value);
  expect(details.declaration.name).toBe("codec::invalid_data");
  expect(details.payload).toMatchObject({ path, reason });
}

test("all octets round-trip through new frozen arrays without mutable aliases", async () => {
  const input = Array.from({ length: 256 }, (_, i) => BigInt(i));
  const original = [...input];
  const bytes = value(await api.fromInts(input));
  input.fill(9n);
  const first = value(await api.toInts(bytes));
  const second = value(await api.toInts(bytes));
  expect(first).toEqual(original);
  expect(second).toEqual(original);
  expect(first).not.toBe(second);
  expect(Object.isFrozen(first)).toBe(true);
  expect(byteLength(bytes)).toBe(256n);
  expect(() => {
    (first as bigint[])[0] = 255n;
  }).toThrow();
  expect(byteLength(value(await api.empty()))).toBe(0n);
  expect(value(await api.toUTF8(value(await api.empty())))).toBe("");
  const native = new Uint8Array([1, 2]);
  const owned = ownBytes(native);
  native.fill(3);
  const copy = copyBytes(owned, origin);
  copy.fill(4);
  expect(value(await api.toInts(owned))).toEqual([1n, 2n]);
  // Buffer.slice would retain an alias; the owner must copy all Uint8Array inputs.
  const buffer = Buffer.from([5, 6]);
  const ownedBuffer = ownBytes(buffer);
  buffer.fill(0);
  expect(value(await api.toInts(ownedBuffer))).toEqual([5n, 6n]);
});

test("integer bounds are checked before native numeric narrowing", async () => {
  for (const bad of [-1n, 256n, -(1n << 256n), 1n << 256n])
    invalid(await api.fromInts([0n, bad, 255n]), "/1", "byte_range");
  invalid(await api.fromInts([1] as unknown as bigint[]), "/0", "type");
});

test("fatal UTF-8 preserves BOM and scalar extremes and rejects malformed bytes", async () => {
  for (const text of [
    "",
    "\0",
    "héllo 😀",
    "\ufefftext",
    "\ufffd",
    "\ud7ff\ue000",
    "\udbff\udfff",
  ]) {
    const encoded = value(await api.fromUTF8(text));
    expect(value(await api.toUTF8(encoded))).toBe(text);
    expect(byteLength(encoded)).toBe(BigInt(new TextEncoder().encode(text).length));
  }
  for (const text of ["\ud800", "\udfff", "a\ud800b", "\ud800\ud800"])
    invalid(await api.fromUTF8(text), "", "unicode_scalar");
  for (const bytes of [
    [0x80],
    [0xc0, 0xaf],
    [0xc2],
    [0xe2, 0x82],
    [0xed, 0xa0, 0x80],
    [0xf4, 0x90, 0x80, 0x80],
    [0xff],
  ])
    invalid(await api.toUTF8(ownBytes(Uint8Array.from(bytes))), "", "utf8");
});

test("forged opaque values fail with resource_state without executing traps", async () => {
  let traps = 0;
  const fake = new Proxy(
    {},
    {
      get() {
        traps++;
        throw Error("trap");
      },
      getPrototypeOf() {
        traps++;
        throw Error("trap");
      },
    },
  );
  for (const fakeValue of [fake, {}, new Uint8Array([1]), null]) {
    for (const operation of [
      () => api.toInts(fakeValue as Bytes),
      () => api.toUTF8(fakeValue as Bytes),
    ]) {
      const result = await invoke<unknown>(operation, origin);
      expect(result.kind).toBe("standard");
      if (result.kind === "standard")
        expect(standardFailureKind(result.value)).toBe("resource_state");
    }
    try {
      byteLength(fakeValue);
      throw Error("accepted forged bytes");
    } catch (cause) {
      if (!isStandardFailure(cause)) throw cause;
      expect(standardFailureKind(cause)).toBe("resource_state");
    }
  }
  expect(traps).toBe(0);
});

test("raw transport fixture consumes copies and cannot mutate owned source bytes", async () => {
  const original = value(await api.fromUTF8("héllo 😀"));
  const request = new Request("https://fixture.invalid/echo", {
    method: "POST",
    body: new Uint8Array(copyBytes(original, origin)),
  });
  // An in-process raw transport fixture consumes the real native Request body.
  const delivered = new Uint8Array(await request.arrayBuffer());
  expect(new TextDecoder().decode(delivered)).toBe("héllo 😀");
  const response = new Response(delivered, { status: 200 });
  const received = ownBytes(new Uint8Array(await response.arrayBuffer()));
  delivered.fill(0);
  expect(value(await api.toUTF8(received))).toBe("héllo 😀");
  expect(value(await api.toUTF8(original))).toBe("héllo 😀");
});

test("base64 and hex round-trip strictly without silent truncation", async () => {
  const bytes = value(await api.fromUTF8("héllo 😀"));
  expect(value(await api.encodeBase64(bytes))).toBe(
    Buffer.from("héllo 😀", "utf8").toString("base64"),
  );
  expect(value(await api.encodeHex(bytes))).toBe(Buffer.from("héllo 😀", "utf8").toString("hex"));
  expect(value(await api.encodeBase64(value(await api.empty())))).toBe("");
  expect(value(await api.encodeHex(value(await api.empty())))).toBe("");
  expect(value(await api.toUTF8(value(await api.decodeBase64("aMOpbGxvIPCfmIA="))))).toBe(
    "héllo 😀",
  );
  expect(value(await api.toUTF8(value(await api.decodeHex("68c3a96c6c6f20f09f9880"))))).toBe(
    "héllo 😀",
  );
  expect(value(await api.toUTF8(value(await api.decodeHex("68C3A96C6C6F20F09F9880"))))).toBe(
    "héllo 😀",
  );
  expect(value(await api.toInts(value(await api.decodeBase64(""))))).toEqual([]);
  expect(value(await api.toInts(value(await api.decodeHex(""))))).toEqual([]);
  for (const bad of ["QQ", "QQ===", "Q!Q=", "Q Q=", "AQ-_", "QQQQ\nQQ=="]) {
    invalid(await api.decodeBase64(bad), "", "base64");
  }
  for (const bad of ["0", "0g", "zz", "0x12", "12 34"]) {
    invalid(await api.decodeHex(bad), "", "hex");
  }
});
