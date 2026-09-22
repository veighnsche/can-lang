# a52: hex encode kernel (B8 scope)

Scope: `bytes__hex__encode`, a total kernel: `Bytes` in,
`Encoding__Text` out (lowercase hex), `emits []`. Same proven
shape as B3; no errors, no catalog churn, no authorization.

1. `compiler/export.go`: `bytesHexEncodeKernel` const +
   descriptor row (`[value: Bytes]`, `Encoding__Text`, `[]`,
   unrestricted) + `isBytesHexEncode` helper.
2. `compiler/eval.go`: `evBytesHexEncodeOp` (Go
   `encoding/hex`, lowercase by construction; empty -> `""`)
   + dispatch branch. Strict `Bytes` admission; static
   mismatch names `want Bytes`.
3. `compiler/emit.go`: `stmtBytesHexEncode` (Ok-only arms,
   own lowering — no sharing with the TextEncoder path) +
   dispatch branch + `$canHexEncode` nibble-table helper,
   emitted only when called (same gating as `$canUtf8Decode`).
4. `compiler/bytes_b8_test.go`: H-rows (below). Fixture
   relays the kernel into `Encoding__Text` directly.

Deliberately untouched: `stmtBytesEncode` (proven path),
module unions (no new kinds), catalogs (emits `[]` adds no
entries — goldens must stay byte-identical with NO regen),
stdlib (B9 ships the wrapper).

Out of scope: hex decode (B10, fallible — chatbot review
candidate), base64 (B12–B15).

## Rollback

`git checkout -- compiler/` plus delete
`compiler/bytes_b8_test.go`. No golden, stdlib, or test-harness
change ships in this slice.

## Test plan

- Tests first (red): H0 vectors incl. `zero [0]->"00"`,
  `ff [255]->"ff"`, lowercase proof (`[171]->"ab"`),
  leading zero (`[1]->"01"`), ordered multi-byte
  (`deadbeef`), no-text-interpretation (`[65,66]->"4142"`),
  full low-nibble sweep; named binding; no-`given`;
  str/int/brand admission; explicit empty contract;
  stale arm; emit pin on `$canHexEncode`.
- `go test -count=1 ./...`; assert `git status` shows no
  golden/std churn (no regen run at all).
- Node vectors: exhaustive 1-byte (256) + multi-byte incl.
  NUL/high bytes through the ACTUAL emitted helper,
  against an independent python oracle (`bytes.hex()`),
  plus a Go-vs-python cross-check on the same inputs.
- `go run ./tools/modcheck`, `go run ./tools/gramcheck`.
