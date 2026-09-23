# B1-08 crypto contract confirmation

Driver: Bun 1.4.2 (`Bun.password`, `crypto.subtle`, `Bun.CryptoHasher`).
Every row below is an executed observation from the B1-08 probes or the
`crypto.test.ts` suite (18 tests) unless marked as policy. No cipher,
HMAC, or hash primitive is implemented in Can.

## Suite

AES-256-GCM (12-byte nonce, 128-bit tag, both fixed by the adapter),
HMAC-SHA-256 over byte keys, Ed25519 (full lifecycle qualified:
generate, sign, verify, raw public export, pkcs8 private export,
raw public import), and SHA-256 (unchanged I30 behavior, moved into
`platform/crypto/primitives.ts`). No other algorithm is admitted:
there is no algorithm selector to fall back through.

## Passwords

`password::hash` offers exactly three argon2id presets: 0 fast
(8 MiB, 1 pass, ~3 ms), 1 balanced (64 MiB, 2 passes, the native
default), 2 secure (256 MiB, 3 passes, ~440 ms). Any other profile is
`password::cost_rejected`. There are no custom costs.

`password::verify` reads the recorded parameters from the encoded hash
and only inside the qualified envelope: `$argon2id$v=19$m,t,p$` with
memory 8 KiB..1 GiB (spec floor, qualified ceiling), time 1..32,
parallelism 1..4, and 43-character unpadded base64 salt and hash
(32 bytes each, stable across presets). Anything else — including
bcrypt hashes and the empty string — is `password::invalid_hash`.
The gate exists because native verify answers `""` with false but
throws `UnsupportedAlgorithm`/`InvalidEncoding` on other malformed
input; the adapter never lets malformed input reach native, so a
mismatch reads false and only false reads as a mismatch.

## Keys

`crypto::key` is an opaque handle: an empty frozen token whose native
`CryptoKey` lives only in the registry in `keys.ts`. Handles
stringify to `{}` and carry no material into failure payloads or
fixture comparisons. Usage is checked before every native call:
AES handles carry encrypt/decrypt, Ed25519 splits into a sign-only
private handle and a verify-only public handle, and anything else is
`crypto::key_misuse` with the operation and the key's algorithm.

The Ed25519 pair generates native-extractable because the public half
must stay distributable through `export_ed25519_public` (raw 32 bytes,
the only admitted export); the private half never leaves because no
operation accepts a sign-capable handle for export. AES keys generate
native-nonextractable and have no export or import operation: they are
session handles. `import_ed25519_public` admits any 32 bytes (length
is the malformed gate); curve validity is enforced by verification,
which reads false for non-keys such as all-zero input.

## Failure map

`password::cost_rejected {profile}`, `password::invalid_hash
{reason: "format"}`, `crypto::invalid_key {reason:
"empty_hmac_key" | "bad_public_length"}`, `crypto::invalid_nonce
{length}` (native accepts 8-byte nonces, so the adapter enforces 12),
`crypto::key_misuse {operation, algorithm}`, and fieldless
`crypto::decrypt_failed`: tampered ciphertext, key, nonce, and
associated-data mismatch all collapse into it (native
`OperationError`), because distinguishing the cause would build an
oracle. Unexpected native throws surface as standard failures.

## Recorded limits

AES has no known-answer vector in the suite: without a key import
operation no fixed key can be injected, so AES evidence is
determinism under fixed nonces, round-trips, tamper collapse, and
length checks. HMAC and SHA-256 pin RFC 4231 case 1 and fixed
vectors; Ed25519 interoperates with node:crypto in both directions.
Nonce uniqueness for `encrypt_aes_gcm_sealed` is a caller contract:
generation is native randomness, not a guarantee of discipline.

## Fixtures

Hash/verify/vector computation stays real inside attached assertions.
Key and sealed-nonce generation are supplied nondeterministic
boundaries; deterministic fixtures can AES material only through
canned handles, while Ed25519 import plus node-interop vectors stay
real. Opaque does not imply secure hardware storage or guaranteed
zeroization: only what WebCrypto and Bun provide is promised.
