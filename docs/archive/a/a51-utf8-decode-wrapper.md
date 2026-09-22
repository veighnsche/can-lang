# a51: std__utf8__decode wrapper (B7 scope)

Scope: the public stdlib face of the B6 kernel. No compiler change.
First stdlib function to declare the builtin error in `emits` and
first exhaustive fallible match in real stdlib code.

1. `std/text/text.can`: `provides += std__utf8__decode`. (`uses []`
   stays. Mod-level `emits` untouched per the `empty_separator`
   precedent: it is not an exhaustive inventory.)
2. `std/text/text.can`: `fn std__utf8__decode(value: Bytes) ->
   Encoding__Text rev 1` after `std__utf8__encode`, `emits
   [encoding.invalid_utf8]`, relay body (`on Ok` + `on
   encoding.invalid_utf8`, payload `e.value` unchanged). No
   per-fn comment (file convention). Rows, one line each: valid
   (empty, ASCII, latin, CJK, astral, U+FFFD, markup) and
   invalid (overlong, surrogate, above-max, stray, truncated,
   truncated-after-ASCII, mid-bytes — each carrying the exact
   input). No committed raw NUL/BOM: text.can stays a text
   file; NUL/BOM relay rides the Go probe (in-memory sources).
3. Header comment + `std/text/README.md`: extend the utf8
   pointer to the decode wrapper.
4. Regen: `text.ts` gains the union member, the wrapper fn,
   AND the `$canUtf8Decode` helper (kernel lowering inlines
   at the call site). `errors.json`: the `invalid_utf8`
   entry gains `handled_by` (relay arm) + `hit_by_tests`
   (invalid rows); every other entry byte-identical. Any
   further delta fails the slice.

Out of scope: hex/base64 (B8–B15), downstream NUL consumers
(still unresolved per B4).

## Rollback

`git checkout -- std/text/text.can std/text/text.ts
std/text/errors.json std/text/README.md` plus delete
`compiler/bytes_b7_test.go`. No compiler or golden-test
change ships in this slice.

## Test plan

- Foreign-caller probe first (red): temp `client.can` with
  `uses [std__utf8__decode@1]`, two fns scripting TRUE
  outcomes through `given` (error outcomes are scriptable
  per `checkOutcome`): success (`[65]` -> `"A"`) and
  failure (`[255]` -> identical payload), plus NUL and
  BOM inputs the committed rows omit. Fails before the
  .can change (CAN2102), passes after.
- `go test -count=1 ./...` (wrapper rows run in-suite;
  golden red until regen).
- Regen, inspect `git diff` (helper + member + fn in
  `text.ts`; handled/hit population in `errors.json`
  only), structural JSON check of all other entries.
- Node vectors: extract emitted `std__utf8__decode` from
  fresh `text.ts`, run valid/invalid/NUL/BOM inputs,
  compare tags + text + full error bytes with Go.
- `go run ./tools/modcheck`, `go run ./tools/gramcheck`.
