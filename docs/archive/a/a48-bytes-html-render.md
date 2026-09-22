# a48: Bytes consumer — html__render__utf8 (B4 scope)

Scope: six edits, all inside the HTML row. No compiler change; the
B2 grant mechanism and B3 encoder already shipped.

1. `std/html/html.can`: grant `exports_utf8 Html__Safe via
   html__render__utf8@1` adjacent to the `Html__Safe` brand decl
   (owner-local, mirrors the B2 fixture layout).
2. `std/html/html.can`: `provides += html__render__utf8`.
3. `std/html/html.can`: `fn html__render__utf8(document: Html__Safe)
   -> Bytes__Value rev 1` after `html__render__document`, exact-shape
   body (`match call bytes__utf8__export(document)`, `on Ok` relay),
   `emits []`. Decision-table rows, each on ONE line (the parser
   rejects split rows): empty, ASCII, entity spelling (`&amp;`
   five bytes, proving no decode), markup, two-byte, astral, BOM,
   NUL-first/middle/last. NUL rows carry RAW bytes — the only .can
   spelling — spliced by script, verified by byte inspection.
4. `std/html/html.can`: retire the stale `// utf8 waits on the Bytes
   type (issue #42).` comment above `html__render__document`.
5. `std/html/html.ts`: regen via `go run ./compiler --out std/html
   std/html/html.can`. Expected diff: one union member plus one
   function. `errors.json` regen must be byte-identical (B4 adds no
   errors); any delta fails the slice.
6. `std/html/README.md`: line 62 says "errors empty — the module is
   total"; the registry holds 8 kinds. Reword to the golden
   description. (The `seals_from` remnant from review is already
   current: `[Html__Text, Html__Attributes]`.)

Out of scope: downstream NUL consumer contract. Frozen chatbot
verdict: the defensible claim after B4 is "exact `Html__Safe`
serialization to Bytes", not verified end-to-end delivery.

## Rollback

`git checkout -- std/html/html.can std/html/html.ts
std/html/errors.json std/html/README.md` plus delete
`compiler/bytes_b4_test.go`. No compiler or golden-test change ships
in this slice, so rollback is four checkouts and one deletion.

## Test plan

- Probe first (red): temp-copy composition
  `node("A<NUL>&amp;B")` -> `render__utf8` -> exactly
  `[65, 0, 38, 97, 109, 112, 59, 66]`, exercising REAL bodies
  (Go lexer string halves glued around one real NUL). Fails before
  the .can change (no such function), passes after.
- `go test -count=1 ./...` (decision-table rows run in-suite;
  golden test goes red until regen).
- Regen, inspect `git diff std/html/html.ts`, `cmp` errors.json.
- Node vectors: extract emitted `html__render__utf8` from fresh
  `html.ts`, run ASCII/entity/NUL/BOM inputs, byte-compare with Go.
- `go run ./tools/modcheck`, `go run ./tools/gramcheck`.
