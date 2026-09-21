import { test, expect } from "bun:test";
import { createHash } from "node:crypto";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, domainFailureDiagnostics, type FailureShape } from "../domain.ts";
import { createCLI } from "../platform/cli.ts";
import { copyBytes, ownBytes, isBytes, createBytes } from "../bytes.ts";
import { value, invoke } from "../completion.ts";

const identity = (kind: string, declaration: string) => createHash("sha256").update("can-concrete-type-v1\0" + JSON.stringify([kind, declaration])).digest("hex");
const text: FailureShape = {identity: identity("primitive", "str"), kind: "primitive", declaration: "str", arguments: [], fields: [], leaves: [], inputs: [], errors: []};
const declarations = catalogue.errors.filter(e => e.name === "codec::invalid_data" || e.name === "io::write_failed").map(e => ({identity: e.identity, name: e.name, id: e.id, parameters: 0}));
const errors: FailureShape[] = declarations.map(e => ({identity: identity("error", e.identity), kind: "error", declaration: e.identity, arguments: [], fields: (e.name === "codec::invalid_data" ? ["path", "reason"] : ["operation"]).map(name => ({name, type: text.identity})), leaves: [], inputs: [], errors: []}));
const domain = createDomainRuntime({declarations, shapes: [text, ...errors]});
const bytesAPI = createBytes(domain, identity("error", "can.std.codec@1::invalid_data"));
const cli = createCLI(domain, {writeFailed: identity("error", "can.std.io@1::write_failed")});
const origin = {source: "can:test", start: 0, end: 0, invocation: []};

test("native UTF-8 conversion preserves scalar text, BOM and immutable byte ownership", async () => {
  for (const text of ["", "héllo 😀", "\ufefftext", "\0", "\uFFFD"]) {
    const bytes = value(await bytesAPI.fromUTF8(text));
    const copy = copyBytes(bytes, origin);
    expect(new TextDecoder("utf-8", {fatal: true, ignoreBOM: true}).decode(copy)).toBe(text);
    copy.fill(0xff);
    expect(new TextDecoder("utf-8", {fatal: true, ignoreBOM: true}).decode(copyBytes(bytes, origin))).toBe(text);
  }
  const original = new Uint8Array([1, 2]);
  const bytes = ownBytes(original);
  original.fill(3);
  expect(Array.from(copyBytes(bytes, origin))).toEqual([1, 2]);
  expect(Object.isFrozen(bytes)).toBe(true);
});

test("unpaired surrogates yield the exact codec domain error", async () => {
  for (const text of ["\ud800", "\udfff", "a\ud800b"]) {
    const completion = await bytesAPI.fromUTF8(text);
    expect(completion.kind).toBe("domain");
    if (completion.kind !== "domain") throw new Error("wrong outcome");
    const details = domainFailureDiagnostics(completion.value);
    expect(details.declaration.id).toBe(1110);
    expect(details.payload).toMatchObject({path: "", reason: "unicode_scalar"});
  }
});

test("forged byte handles fail before native output and without running traps", async () => {
  let traps = 0;
  const forged = new Proxy({}, {get() {traps++; throw new Error("secret");}, getPrototypeOf() {traps++; throw new Error("secret");}});
  expect(isBytes(forged)).toBe(false);
  const completion = await invoke(() => cli.stdoutWrite(forged), origin);
  expect(completion.kind).toBe("standard");
  expect(traps).toBe(0);
});
