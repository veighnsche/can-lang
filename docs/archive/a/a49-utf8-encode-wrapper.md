# a49: std__utf8__encode wrapper (B5 scope)

Scope: the public stdlib face of the B3 kernel. No compiler change;
no grant, no brand, no authorization surface.

1. `std/text/text.can`: `provides += std__utf8__encode`. (`uses []`
   stays: kernels resolve without uses, B3-proven.)
2. `std/text/text.can`: `fn std__utf8__encode(value: str) ->
   Bytes__Value rev 1` appended after `std__str__split`, ordinary
   body (`match call bytes__utf8__encode(value)`, `on Ok` relay),
   `emits []`. No
   per-fn comment (file convention is bare fns). Decision-table
   rows, each on ONE line: empty, ASCII, latin, CJK, astral,
   markup-passthrough. All inputs are UTF-8 literals (file
   precedent: héllo/世界); NO committed raw NUL — text.can stays
   a text file, NUL relay is pinned by the Go test instead.
3. `std/text/text.can` line 16 + `std/text/README.md`: retire the
   `hex/base64/utf8 on Bytes` waits-remark to `hex/base64`,
   pointing at the new wrapper.
4. `std/text/text.ts`: regen via `go run ./compiler --out std/text
   std/text/text.can`. Expected diff: one union member plus one
   function. `errors.json` regen must be byte-identical (wrapper
   emits nothing); any delta fails the slice.

Out of scope: decoders (B6/B7), hex/base64 (B8–B15), downstream
NUL consumers (still unresolved per B4).

## Rollback

`git checkout -- std/text/text.can std/text/text.ts
std/text/errors.json std/text/README.md` plus delete
`compiler/bytes_b5_test.go`. No compiler or golden-test change
ships in this slice.

## Test plan

- Foreign-caller test first (red): temp `client.can` with
  `uses [std__utf8__encode@1]` calls the wrapper with a spliced-NUL
  input (`a<NUL>b` -> `[97, 0, 98]`) plus a computed vector. This
  proves the §6 claim "ordinary wrappers are normal functions"
  under foreign-call rules — the wrapper's own rows cannot show
  that. Fails before the .can change, passes after.
- `go test -count=1 ./...` (wrapper rows run in-suite; golden red
  until regen).
- Regen, inspect `git diff std/text/text.ts`, `cmp` errors.json.
- Node vectors: extract emitted `std__utf8__encode` from fresh
  `text.ts`, run empty/ASCII/multibyte/markup/NUL inputs,
  byte-compare with Go.
- `go run ./tools/modcheck`, `go run ./tools/gramcheck`.
