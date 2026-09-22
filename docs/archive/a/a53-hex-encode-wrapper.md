# a53: std__hex__encode wrapper (B9 scope)

Scope: the public stdlib face of the B8 kernel. No compiler change.
Total function, same shape as B5.

1. `std/text/text.can`: `provides += std__hex__encode`. (`uses []`
   stays. Mod-level `emits` untouched per the `empty_separator`
   precedent.)
2. `std/text/text.can`: `fn std__hex__encode(value: Bytes) ->
   Encoding__Text rev 1` after `std__utf8__decode`, ordinary
   body (`match call bytes__hex__encode(value)`, `on Ok`
   relay), `emits []`. No per-fn comment (file convention).
   Rows, one line each: empty, zero, `ff`, lowercase proof,
   leading zero, ordered multi-byte, no-text-interpretation,
   full low-nibble sweep (mirroring the H0 kernel rows that
   pin the contract).
3. Header comment + `std/text/README.md`: move `hex` out of
   the waits-remark to the new wrapper (`base64` still waits).
4. Regen: `text.ts` gains the union member, the wrapper fn,
   AND the `$canHexEncode` helper (kernel lowering inlines
   at the call site). `errors.json` must be byte-identical
   (wrapper emits nothing); any delta fails the slice.

Out of scope: hex decode (B10, fallible — chatbot review
candidate), base64 (B12–B15).

## Rollback

`git checkout -- std/text/text.can std/text/text.ts
std/text/errors.json std/text/README.md` plus delete
`compiler/bytes_b9_test.go`. No compiler or golden-test
change ships in this slice.

## Test plan

- Foreign-caller probe first (red): temp `client.can` with
  `uses [std__hex__encode@1]`, proving §6 ordinary-function
  callability plus lowercase/order/notext vectors.
  Fails before the .can change (CAN2102), passes after.
- `go test -count=1 ./...` (wrapper rows run in-suite;
  golden red until regen).
- Regen, inspect `git diff std/text/text.ts`, `cmp`
  errors.json.
- Node vectors: extract emitted `std__hex__encode` (+
  helper) from fresh `text.ts`, run empty/zero/ff/
  lowercase/order/nibble vectors, compare with Go.
- `go run ./tools/modcheck`, `go run ./tools/gramcheck`.
