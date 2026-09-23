import { describe, expect, test } from "bun:test";
import { createHash, createHmac, createPrivateKey, createPublicKey, sign as nodeSign, verify as nodeVerify } from "node:crypto";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, domainFailureDiagnostics, type FailureShape } from "../domain.ts";
import { value, type Completion } from "../completion.ts";
import { record, recordIdentity, dataProperty } from "../data.ts";
import { ownBytes, copyBytes } from "../bytes.ts";
import { createCryptoKeys, isCryptoKeyValue, projectKey } from "../platform/crypto/keys.ts";
import { createPasswords } from "../platform/crypto/password.ts";
import { createCryptoPrimitives, sha256 } from "../platform/crypto/primitives.ts";

const origin = { source: "test:crypto", start: 0, end: 0, invocation: [] };
const identity = (kind: string, declaration: string) =>
  createHash("sha256").update("can-concrete-type-v1\0" + JSON.stringify([kind, declaration])).digest("hex");
const textShape: FailureShape = { identity: identity("primitive", "str"), kind: "primitive", declaration: "str", arguments: [], fields: [], leaves: [], inputs: [], errors: [] };
const intShape: FailureShape = { identity: identity("primitive", "int"), kind: "primitive", declaration: "int", arguments: [], fields: [], leaves: [], inputs: [], errors: [] };
const fieldTypes: Record<string, Record<string, string>> = {
  "password::cost_rejected": { profile: intShape.identity },
  "password::invalid_hash": { reason: textShape.identity },
  "crypto::invalid_key": { reason: textShape.identity },
  "crypto::invalid_nonce": { length: intShape.identity },
  "crypto::key_misuse": { operation: textShape.identity, algorithm: textShape.identity },
  "crypto::decrypt_failed": {},
};
const declarations = catalogue.errors
  .filter(e => fieldTypes[e.name] !== undefined)
  .map(e => ({ identity: e.identity, name: e.name, id: e.id, parameters: 0 }));
const errorShapes: FailureShape[] = declarations.map(e => ({
  identity: identity("error", e.identity), kind: "error", declaration: e.identity, arguments: [],
  fields: Object.entries(fieldTypes[e.name]!).map(([name, type]) => ({ name, type })),
  leaves: [], inputs: [], errors: [],
}));
const domain = createDomainRuntime({ declarations, shapes: [textShape, intShape, ...errorShapes] });
const id = (declaration: string) => identity("error", declaration);
const keyIds = { keyMisuse: id("can.std.crypto@1::key_misuse"), invalidKey: id("can.std.crypto@1::invalid_key"), keypair: "crypto::keypair" };
const keys = createCryptoKeys(domain, keyIds);
const passwords = createPasswords(domain, { costRejected: id("can.std.password@1::cost_rejected"), invalidHash: id("can.std.password@1::invalid_hash") });
const primitives = createCryptoPrimitives(domain, {
  keyMisuse: keyIds.keyMisuse, invalidKey: keyIds.invalidKey,
  invalidNonce: id("can.std.crypto@1::invalid_nonce"), decryptFailed: id("can.std.crypto@1::decrypt_failed"),
  sealed: "crypto::sealed",
});

function domainOutcome(completion: Completion<unknown>, name: string): Record<string, unknown> {
  expect(completion.kind).toBe("domain");
  if (completion.kind !== "domain") throw new Error("wrong outcome");
  const details = domainFailureDiagnostics(completion.value);
  expect(details.declaration.name).toBe(name);
  const payload = details.payload as Record<string, unknown>;
  const plain: Record<string, unknown> = {};
  for (const key of Object.keys(payload)) plain[key] = payload[key];
  return plain;
}
const bytes = (values: readonly number[]) => ownBytes(new Uint8Array(values));
const hexOf = (v: unknown) => Buffer.from(copyBytes(v, origin)).toString("hex");
const utf8 = (s: string) => ownBytes(new TextEncoder().encode(s));

describe("crypto hashes and HMAC", () => {
  test("sha256 matches fixed vectors", async () => {
    expect(hexOf(value(await sha256(utf8("abc"))))).toBe("ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad");
    expect(hexOf(value(await sha256(ownBytes(new Uint8Array(0)))))).toBe("e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855");
  });
  test("hmac matches RFC 4231 case 1 and node interop", async () => {
    const key = ownBytes(new Uint8Array(20).fill(0x0b));
    expect(hexOf(value(await primitives.hmacSha256(key, utf8("Hi There"))))).toBe("b0344c61d8db38535ca8afceaf0bf12b881dc200c9833da726e9376c2e32cff7");
    const raw = new Uint8Array([9, 9, 9]);
    expect(hexOf(value(await primitives.hmacSha256(ownBytes(raw), utf8("can"))))).toBe(createHmac("sha256", raw).update("can").digest("hex"));
  });
  test("empty hmac key rejects without native use", async () => {
    expect(domainOutcome(await primitives.hmacSha256(ownBytes(new Uint8Array(0)), utf8("x")), "crypto::invalid_key")).toEqual({ reason: "empty_hmac_key" });
  });
  test("hmac copies inputs at the edge", async () => {
    const view = new Uint8Array([1, 2, 3]);
    const first = hexOf(value(await primitives.hmacSha256(ownBytes(view), utf8("m"))));
    view.fill(9);
    expect(hexOf(value(await primitives.hmacSha256(ownBytes(view), utf8("m"))))).not.toBe(first);
    expect(first).toBe(createHmac("sha256", new Uint8Array([1, 2, 3])).update("m").digest("hex"));
  });
});

describe("password hashing", () => {
  const envelope = /^\$argon2id\$v=19\$m=(\d+),t=(\d+),p=(\d+)\$[A-Za-z0-9+/]{43}\$[A-Za-z0-9+/]{43}$/;
  test("each preset hashes inside the qualified envelope and verifies", async () => {
    const params = [["m=8192,t=1,p=1"], ["m=65536,t=2,p=1"], ["m=262144,t=3,p=1"]] as const;
    for (let i = 0; i < 3; i++) {
      const hash = value(await passwords.hash("correct horse", BigInt(i)));
      const match = envelope.exec(hash);
      expect(match).not.toBeNull();
      expect(`m=${match![1]},t=${match![2]},p=${match![3]}`).toBe(params[i][0]);
      expect(value(await passwords.verify("correct horse", hash))).toBe(true);
      expect(value(await passwords.verify("wrong", hash))).toBe(false);
    }
  }, 30000);
  test("unknown presets reject distinctly", async () => {
    expect(domainOutcome(await passwords.hash("pw", 3n), "password::cost_rejected")).toEqual({ profile: 3n });
    expect(domainOutcome(await passwords.hash("pw", -1n), "password::cost_rejected")).toEqual({ profile: -1n });
  });
  test("malformed hashes reject instead of reading false or throwing", async () => {
    const bad = [
      "", "not-a-hash", "$2b$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy",
      "$argon2id$v=19$m=65536,t=2,p=1$short$short",
      "$argon2id$v=16$m=65536,t=2,p=1$AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA$AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
      "$argon2id$v=19$m=0,t=2,p=1$AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA$AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
      "$argon2id$v=19$m=99999999,t=2,p=1$AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA$AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
      "$argon2id$v=19$m=65536,t=99,p=1$AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA$AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
      "$argon2id$v=19$m=65536,t=2,p=9$AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA$AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
    ];
    for (const encoded of bad) {
      expect(domainOutcome(await passwords.verify("pw", encoded), "password::invalid_hash")).toEqual({ reason: "format" });
    }
  });
});

describe("aes-gcm", () => {
  test("explicit-nonce round-trip is deterministic with a 16-byte tag", async () => {
    const key = value(await keys.generateAESKey());
    const nonce = bytes([0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11]);
    const aad = utf8("header");
    const first = value(await primitives.encryptAesGcm(key, nonce, utf8("secret"), aad));
    const second = value(await primitives.encryptAesGcm(key, nonce, utf8("secret"), aad));
    expect(hexOf(first)).toBe(hexOf(second));
    expect(copyBytes(first, origin).length).toBe("secret".length + 16);
    expect(hexOf(value(await primitives.decryptAesGcm(key, nonce, first, aad)))).toBe(hexOf(utf8("secret")));
  });
  test("sealed encryption mints a fresh nonce alongside the ciphertext", async () => {
    const key = value(await keys.generateAESKey());
    const first = value(await primitives.encryptAesGcmSealed(key, utf8("sealed"), utf8("aad")));
    const second = value(await primitives.encryptAesGcmSealed(key, utf8("sealed"), utf8("aad")));
    expect(recordIdentity(first)).toBe("crypto::sealed");
    const nonceOf = (sealed: unknown) => dataProperty(sealed, "nonce");
    const cipherOf = (sealed: unknown) => dataProperty(sealed, "ciphertext");
    expect(copyBytes(nonceOf(first), origin).length).toBe(12);
    expect(hexOf(nonceOf(first))).not.toBe(hexOf(nonceOf(second)));
    expect(hexOf(cipherOf(first))).not.toBe(hexOf(cipherOf(second)));
    expect(hexOf(value(await primitives.decryptAesGcm(key, nonceOf(first), cipherOf(first), utf8("aad"))))).toBe(hexOf(utf8("sealed")));
  });
  test("tampered ciphertext, nonce, key, and aad collapse to decrypt_failed", async () => {
    const key = value(await keys.generateAESKey());
    const other = value(await keys.generateAESKey());
    const nonce = new Uint8Array(12).fill(7);
    const cipher = new Uint8Array(copyBytes(value(await primitives.encryptAesGcm(key, ownBytes(nonce), utf8("data"), utf8("aad"))), origin));
    const flip = (at: number) => { const copy = new Uint8Array(cipher); copy[at] ^= 1; return ownBytes(copy); };
    expect(domainOutcome(await primitives.decryptAesGcm(key, ownBytes(nonce), flip(0), utf8("aad")), "crypto::decrypt_failed")).toEqual({});
    expect(domainOutcome(await primitives.decryptAesGcm(key, ownBytes(nonce), flip(cipher.length - 1), utf8("aad")), "crypto::decrypt_failed")).toEqual({});
    const badNonce = new Uint8Array(nonce); badNonce[0] ^= 1;
    expect(domainOutcome(await primitives.decryptAesGcm(key, ownBytes(badNonce), ownBytes(cipher), utf8("aad")), "crypto::decrypt_failed")).toEqual({});
    expect(domainOutcome(await primitives.decryptAesGcm(other, ownBytes(nonce), ownBytes(cipher), utf8("aad")), "crypto::decrypt_failed")).toEqual({});
    expect(domainOutcome(await primitives.decryptAesGcm(key, ownBytes(nonce), ownBytes(cipher), utf8("other")), "crypto::decrypt_failed")).toEqual({});
  });
  test("nonce lengths other than 12 reject before native use", async () => {
    const key = value(await keys.generateAESKey());
    for (const length of [0, 8, 11, 13, 16]) {
      const nonce = ownBytes(new Uint8Array(length));
      expect(domainOutcome(await primitives.encryptAesGcm(key, nonce, utf8("x"), utf8("")), "crypto::invalid_nonce")).toEqual({ length: BigInt(length) });
      expect(domainOutcome(await primitives.decryptAesGcm(key, nonce, utf8("x"), utf8("")), "crypto::invalid_nonce")).toEqual({ length: BigInt(length) });
    }
  });
});

describe("ed25519", () => {
  test("keypairs sign and verify with reimported public keys", async () => {
    const pair = value(await keys.generateEd25519Keypair());
    expect(recordIdentity(pair)).toBe("crypto::keypair");
    const priv = dataProperty(pair, "private_key"), pub = dataProperty(pair, "public_key");
    expect(isCryptoKeyValue("key", priv)).toBe(true);
    const message = utf8("can-b1-08");
    const signature = value(await primitives.signEd25519(priv, message));
    expect(copyBytes(signature, origin).length).toBe(64);
    expect(value(await primitives.verifyEd25519(pub, message, signature))).toBe(true);
    expect(value(await primitives.verifyEd25519(pub, utf8("other"), signature))).toBe(false);
    const raw = value(await keys.exportEd25519Public(pub));
    expect(copyBytes(raw, origin).length).toBe(32);
    const reimported = value(await keys.importEd25519Public(raw));
    expect(value(await primitives.verifyEd25519(reimported, message, signature))).toBe(true);
  });
  test("signatures interoperate with node crypto both directions", async () => {
    const pair = value(await keys.generateEd25519Keypair());
    const priv = dataProperty(pair, "private_key"), pub = dataProperty(pair, "public_key");
    const native = projectKey(priv)!.key;
    const pkcs8 = new Uint8Array(await crypto.subtle.exportKey("pkcs8", native));
    const nodePriv = createPrivateKey({ key: Buffer.from(pkcs8), format: "der", type: "pkcs8" });
    const nodePub = createPublicKey(nodePriv);
    const message = new Uint8Array([1, 2, 3, 4]);
    const nodeSig = nodeSign(null, message, nodePriv);
    expect(value(await primitives.verifyEd25519(pub, ownBytes(message), ownBytes(new Uint8Array(nodeSig))))).toBe(true);
    const subtleSig = new Uint8Array(copyBytes(value(await primitives.signEd25519(priv, ownBytes(message))), origin));
    expect(nodeVerify(null, message, nodePub, subtleSig)).toBe(true);
  });
  test("non-32-byte public imports reject; all-zero keys admit but never verify", async () => {
    for (const length of [0, 31, 33, 64]) {
      expect(domainOutcome(await keys.importEd25519Public(ownBytes(new Uint8Array(length))), "crypto::invalid_key")).toEqual({ reason: "bad_public_length" });
    }
    const zero = value(await keys.importEd25519Public(ownBytes(new Uint8Array(32))));
    expect(value(await primitives.verifyEd25519(zero, utf8("m"), ownBytes(new Uint8Array(64))))).toBe(false);
  });
});

describe("key misuse and redaction", () => {
  test("encryption-only keys cannot sign and signing keys cannot decrypt", async () => {
    const aes = value(await keys.generateAESKey());
    const pair = value(await keys.generateEd25519Keypair());
    const priv = dataProperty(pair, "private_key"), pub = dataProperty(pair, "public_key");
    expect(domainOutcome(await primitives.signEd25519(aes, utf8("m")), "crypto::key_misuse")).toEqual({ operation: "sign", algorithm: "AES-GCM" });
    expect(domainOutcome(await primitives.verifyEd25519(aes, utf8("m"), ownBytes(new Uint8Array(64))), "crypto::key_misuse")).toEqual({ operation: "verify", algorithm: "AES-GCM" });
    expect(domainOutcome(await primitives.encryptAesGcm(priv, ownBytes(new Uint8Array(12)), utf8("m"), utf8("")), "crypto::key_misuse")).toEqual({ operation: "encrypt", algorithm: "Ed25519" });
    expect(domainOutcome(await primitives.decryptAesGcm(pub, ownBytes(new Uint8Array(12)), utf8("m"), utf8("")), "crypto::key_misuse")).toEqual({ operation: "decrypt", algorithm: "Ed25519" });
    expect(domainOutcome(await primitives.signEd25519(pub, utf8("m")), "crypto::key_misuse")).toEqual({ operation: "sign", algorithm: "Ed25519" });
    expect(domainOutcome(await primitives.verifyEd25519(priv, utf8("m"), ownBytes(new Uint8Array(64))), "crypto::key_misuse")).toEqual({ operation: "verify", algorithm: "Ed25519" });
  });
  test("forbidden exports misuse without touching native", async () => {
    const aes = value(await keys.generateAESKey());
    const priv = dataProperty(value(await keys.generateEd25519Keypair()), "private_key");
    expect(domainOutcome(await keys.exportEd25519Public(aes), "crypto::key_misuse")).toEqual({ operation: "export", algorithm: "AES-GCM" });
    expect(domainOutcome(await keys.exportEd25519Public(priv), "crypto::key_misuse")).toEqual({ operation: "export", algorithm: "Ed25519" });
  });
  test("non-key handles throw instead of misusing", async () => {
    await expect(primitives.signEd25519({}, utf8("m"))).rejects.toThrow(TypeError);
    await expect(keys.exportEd25519Public("key")).rejects.toThrow(TypeError);
  });
  test("handles and pairs redact key material", async () => {
    const aes = value(await keys.generateAESKey());
    const pair = value(await keys.generateEd25519Keypair());
    expect(JSON.stringify(aes)).toBe("{}");
    const pubHex = hexOf(value(await keys.exportEd25519Public(dataProperty(pair, "public_key"))));
    expect(JSON.stringify(pair)).not.toContain(pubHex);
    const failure = domainOutcome(await primitives.signEd25519(aes, utf8("m")), "crypto::key_misuse");
    expect(Object.keys(failure).sort()).toEqual(["algorithm", "operation"]);
  });
});
