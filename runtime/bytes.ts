// Shared private byte representation. Native boundaries receive copies, never
// the owned backing array; application code can obtain only the opaque token.
import { success, failure, type Completion, type AssertionContext } from "./completion.ts";
import { array, dataArray, record } from "./data.ts";
import { createDomainRuntime } from "./domain.ts";
import { resourceStateFailure, type FailureOrigin } from "./failure.ts";

declare const bytesBrand: unique symbol;
export type Bytes = Readonly<{ readonly [bytesBrand]: true }>;
const storage = new WeakMap<object, Uint8Array>();
export function ownBytes(fresh: Uint8Array): Bytes {
  const token = Object.freeze(Object.create(null));
  // Copy even at this maintained boundary so ownership cannot depend on a
  // caller accidentally retaining a native alias.
  storage.set(token, new Uint8Array(fresh));
  return token;
}
export function isBytes(value: unknown): value is Bytes {
  return (
    value !== null &&
    (typeof value === "object" || typeof value === "function") &&
    storage.has(value)
  );
}
export function copyBytes(value: unknown, origin: FailureOrigin): Uint8Array {
  if (!isBytes(value)) throw resourceStateFailure(undefined, origin);
  return storage.get(value)!.slice();
}

const origin: FailureOrigin = Object.freeze({
  source: "can:bytes",
  start: 0,
  end: 0,
  invocation: Object.freeze([]),
});
function backing(value: unknown): Uint8Array {
  if (!isBytes(value)) throw resourceStateFailure(undefined, origin);
  return storage.get(value)!;
}
export function byteLength(value: unknown): bigint {
  return BigInt(backing(value).length);
}

export function createBytes(domain: ReturnType<typeof createDomainRuntime>, invalidData: string) {
  function invalid(path: string, reason: string) {
    return failure(
      domain.create(
        invalidData,
        record(invalidData, [
          ["path", path],
          ["reason", reason],
        ]),
        origin,
      ),
    );
  }
  return Object.freeze({
    async empty(_context?: AssertionContext): Promise<Completion<Bytes>> {
      return success(ownBytes(new Uint8Array(0)));
    },
    async fromInts(
      values: readonly bigint[],
      _context?: AssertionContext,
    ): Promise<Completion<Bytes>> {
      const checked = dataArray(values);
      for (let i = 0; i < checked.length; i++) {
        const octet = checked[i];
        if (typeof octet !== "bigint") return invalid("/" + i, "type");
        if (octet < 0n || octet > 255n) return invalid("/" + i, "byte_range");
      }
      return success(ownBytes(Uint8Array.from(checked as bigint[], Number)));
    },
    async toInts(
      value: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<readonly bigint[]>> {
      return success(array(Array.from(backing(value), BigInt)));
    },
    async fromUTF8(text: string, _context?: AssertionContext): Promise<Completion<Bytes>> {
      if (typeof text !== "string") return invalid("", "type");
      const encoded = new TextEncoder().encode(text);
      if (new TextDecoder("utf-8", { fatal: true, ignoreBOM: true }).decode(encoded) !== text)
        return invalid("", "unicode_scalar");
      return success(ownBytes(encoded));
    },
    async toUTF8(value: unknown, _context?: AssertionContext): Promise<Completion<string>> {
      const bytes = backing(value);
      try {
        return success(new TextDecoder("utf-8", { fatal: true, ignoreBOM: true }).decode(bytes));
      } catch (cause) {
        // Only the fatal decoder's invalid-input TypeError is a codec outcome.
        // Provenance is checked outside this boundary; other faults propagate.
        if (!(cause instanceof TypeError)) throw cause;
        return invalid("", "utf8");
      }
    },
    async encodeBase64(value: unknown, _context?: AssertionContext): Promise<Completion<string>> {
      return success(Buffer.from(backing(value)).toString("base64"));
    },
    async decodeBase64(text: string, _context?: AssertionContext): Promise<Completion<Bytes>> {
      // Buffer's base64 decoder silently skips whitespace and invalid
      // trailing characters, so the strict shape gates first: standard
      // alphabet, correct padding, no base64url.
      if (typeof text !== "string" || text.length % 4 !== 0 || !/^[A-Za-z0-9+/]*={0,2}$/.test(text))
        return invalid("", "base64");
      return success(ownBytes(Buffer.from(text, "base64")));
    },
    async encodeHex(value: unknown, _context?: AssertionContext): Promise<Completion<string>> {
      return success(Buffer.from(backing(value)).toString("hex"));
    },
    async decodeHex(text: string, _context?: AssertionContext): Promise<Completion<Bytes>> {
      // Buffer's hex decoder silently truncates at the first invalid pair,
      // so even length plus a full hex alphabet gates first. Uppercase
      // decodes; encoding always emits lowercase.
      if (typeof text !== "string" || !/^([0-9a-fA-F]{2})*$/.test(text)) return invalid("", "hex");
      return success(ownBytes(Buffer.from(text, "hex")));
    },
  });
}
