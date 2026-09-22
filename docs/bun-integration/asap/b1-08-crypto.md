# B1-08 — Password and broader cryptographic operations

Status: prepared for ASAP execution; implementation and acceptance still pending. Dependencies: none. Surface: Library.

Extend the current SHA-256 operation with password hashing/verification, HMAC and a deliberately explicit WebCrypto key/encryption/signature subset. All cryptography remains native. Distinguish a verification mismatch (false) from malformed/unsupported input and native operational failure.

## Where to implement

**Required layout:** follow [filetree and module boundaries](filetree.md). Its feature modules supersede the coarse existing-file locations below; do not append B1 implementation bodies to existing orchestration hubs.

Paths are repository-relative; **new** means a proposed file/directory, not existing code. The shared wiring below is part of this task, not optional cleanup.

| Path | State | Responsibility |
|---|---|---|
| `runtime/platform/crypto.ts` | existing | Native hashing, password and WebCrypto adapters. |
| `runtime/platform/crypto-key.ts` | new | Private opaque CryptoKey representation and usage checks. |
| `runtime/platform/random.ts` | existing | Reuse secure byte generation where appropriate. |
| `compiler/internal/types/types.go` | existing | Inspect opaque catalogue type admission; extend only if needed. |
| `runtime/test/crypto.test.ts` | new | Vectors, malformed inputs and tampered ciphertext. |
| `tests/integration/crypto_test.go` | new | Typed contracts, key misuse and native lowering. |
| `compiler/internal/catalogue/catalogue.json` | existing | Canonical public types, signatures, exact error identities, native lowering and assertion policy. |
| `compiler/internal/emit/program.go` | existing | Operation identity -> runtime target binding, imports and adapter construction. |
| `runtime/modules.json` | existing | Register each runtime module and exact import edges. |
| `distribution/target.json` | existing | Add actually required qualified native APIs; do not change the pinned runtime casually. |

Generated mirrors are produced with `make catalogue`: `compiler/internal/catalogue/generated.go`, `runtime/catalogue.ts`, `std/catalogue/README.md`. Never hand-edit them. See [shared integration contract](integration-contract.md) for specs, module boundaries, fixture ownership and completion rules.

## Proposed API contract

This is interface notation, **not accepted Can syntax or a claim that these operations compile today**. Resolve final names against the current catalogue and express them in existing Can declaration syntax; only primitive gates may add grammar.

```text
password::hash(password, cost_profile) -> password_hash
password::verify(password, encoded_hash) -> bool
crypto::hmac_sha256(key, bytes) -> bytes
crypto::generate/import_key(algorithm, usages, export_policy) -> opaque_key
crypto::encrypt_aes_gcm(key, nonce, plaintext, associated_data) -> bytes
crypto::decrypt_aes_gcm(key, nonce, ciphertext, associated_data) -> bytes
crypto::sign / verify with explicitly qualified algorithms
```

## Implementation sequence

1. Inventory existing SHA/random operations; add names without duplicate functionality. Specify Argon2id cost presets and validation; retain native encoded hashes so verification can recognize recorded parameters.
2. Use asynchronous Bun.password APIs to avoid synchronous event-loop blocking. Test wrong password, invalid hash and excessive-cost policy distinctly.
3. Choose the initial WebCrypto suite explicitly: AES-GCM, HMAC-SHA-256, and Ed25519 if target qualification succeeds. Fix nonce/tag lengths and supported key usages; reject other algorithms rather than silently falling back.
4. Keep CryptoKey in a private WeakMap-backed opaque value, nonextractable by default. Provide explicit import/export only for admitted formats/usages. No raw key material in fixture reports, logs or generic object projections.
5. Use native CryptoHasher/SubtleCrypto; add only bytes ownership, exact options, declared failure translation and opaque wrapping. Never implement cipher/HMAC primitives in Can.
6. Separate generated nonce convenience from explicit nonce encryption. Document nonce uniqueness as an encryption contract; generation is native randomness, not a guarantee of caller discipline.
7. Add native interoperability vectors and tampering tests. Ensure one adapter cannot use an encryption-only key for signing.

## Acceptance evidence

- Success: password verify true/false, fixed hash vector, HMAC vector, AES-GCM round-trip with AAD, qualified signature interoperability.
- Rejected: wrong key usage/type, forbidden export, invalid nonce length, unsupported algorithm, oversized cost, malformed import.
- Runtime: tampered ciphertext/AAD fails without plaintext; key data is redacted in diagnostics; byte inputs cannot be mutated through native aliases.

## Fixtures and local assertions

Hash/verify/vector computation remains real inside attached assertions. Random key/nonce generation is a supplied nondeterministic boundary when deterministic fixtures are needed; immutable fixture material belongs to the assertion. Tests must still execute native encryption/verification, not return canned success.

## Native lowering sketch

This sketch describes the native operation, not a complete Can adapter. Real adapters must return the existing Completion representation, retain source origin and apply declared error boundaries.

```ts
const hash = await Bun.password.hash(password, {algorithm: "argon2id", ...cost});
const ok = await Bun.password.verify(password, hash);
const cipher = await crypto.subtle.encrypt({name: "AES-GCM", iv, additionalData}, key, input);
```

## Gates and limitations

Opaque does not imply secure hardware storage or guaranteed zeroization. Promise only what WebCrypto/Bun provide. Algorithm availability and import formats need pinned-target tests; preparation only probed password verify and AES-GCM.

## Source and proof

- [Pinned native observations](evidence/native-probe-results.json) and [reproducer](evidence/native-probe.ts). Only listed probe behavior was executed.
- [Official Bun documentation](https://bun.sh/docs/runtime/hashing)
- [Official Bun documentation](https://bun.sh/docs/runtime/web-apis)

Do not check off this capability until its acceptance cases, local fixture behavior, generated TypeScript evidence and all required real-native runs are recorded. Use [execution queue](execution-queue.md) to continue immediately with the next unblocked capability.
