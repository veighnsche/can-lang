# a58: base64 encode kernel + wrapper (B12+B13 scope)

Scope: one encode pair, done together (both total, fully
precedented by B8+B9). No compiler change beyond the kernel
row; no errors, no catalog churn.

B12 — `bytes__base64__encode` kernel:
1. `compiler/export.go`: `bytesB64EncodeKernel` const +
   descriptor (`[value: Bytes]`, `Encoding__Text`, `[]`,
   unrestricted) + `isBytesB64Encode`.
2. `compiler/eval.go`: `evBytesB64EncodeOp` (Go
   `base64.StdEncoding`, padding included; empty -> `""`)
   + dispatch branch. Strict `Bytes` admission.
3. `compiler/emit.go`: `stmtBytesB64Encode` (Ok-only,
   own lowering — no sharing with text/hex paths) +
   dispatch + `$canB64Encode` table helper, call-only
   emission.
4. `compiler/bytes_b12_test.go`: G-rows (empty, `AA==`,
   `/w==`, padding shapes 0/1/2 mod 3, `deadbeef` order,
   no-text-interpretation, full 0-15 sweep; named
   binding; no-`given`; str/int/brand admission;
   explicit empty contract; stale arm; emit pin +
   TextEncoder absence).

B13 — `std__base64__encode` wrapper:
5. `std/text/text.can`: `provides +=`, fn after
   `std__hex__decode` relaying the kernel, `emits []`,
   one-line rows mirroring G0. Header comment + README:
   retire the base64 waits-remark (last one — #42's
   hex/base64/utf8 list is now fully landed).
6. `compiler/bytes_b13_test.go`: foreign-caller probe
   (`uses [std__base64__encode@1]`, padding + order +
   notext vectors).
7. Regen: `text.ts` gains member + helper + fn.
   `errors.json` byte-identical (emits nothing).

Out of scope: base64 decode (B14 — chatbot review),
B15 wrapper.

## Rollback

`git checkout -- compiler/ std/text/` plus delete
`compiler/bytes_b12_test.go compiler/bytes_b13_test.go`
and `docs/a58-*`.

## Test plan

- B12 tests first (red), implement to green.
- B13 probe first (red), implement to green.
- Single regen; `git diff` review; `cmp` errors.json;
  no other golden touched (no regen run for them).
- Node vectors: exhaustive 1-byte + 2/3-byte padding
  shapes + longer inputs through the ACTUAL emitted
  helper, python oracle (`standard_b64encode`) + Go
  cross-check.
- `go test -count=1 ./...`, `modcheck`, `gramcheck`.
