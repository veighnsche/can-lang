# a66: interpreted string literals `e"..."` (chatbot verdict A)

Additive only. Ordinary `"..."` keeps raw semantics
(pinned by `TestStrSemanticsEmit` and the
`tail_backslash` row in `std/html/html.can`: `"a\b"` is
backslash+b). New `e"..."` form decodes exactly six
escapes, once, left to right; same `str` type and runtime
representation, no new primitive. Numbered a66: a62/a63
are the diagnostic fixes, a64 is taken, and the verdict's
linked-pure runner shifts to a67.

1. Scan: recognize the `e"..."` token boundary in
   `stripComment`, `braceOutsideString`,
   `splitTopInner`, operator/bracket scanners, and
   balanced-delimiter scanning (audit each; the `e`
   prefix must not leak into identifiers).
2. `parseSmall`: decode a complete `e"..."` into
   `Small{Kind: "str", Str: decoded}` with ordinary
   embedding/postfix. `parsePattern`: same decoder
   (no value/pattern split).
3. Escapes: `\"` `\\` `\n` `\r` `\t` `\0` only.
   `e"\\n"` is [92,110]; `e"\01"` is [0,49] (no
   octal); unknown escapes, dangling backslash, and
   unterminated literals are CAN1000/`CodeParse`.
   No numeric/unicode escapes, no interpolation, no
   line continuation; `d"..."` untouched.
4. `patDesc` (lsp.go): preserve the source token/span
   for interpreted patterns (decoded newlines are not
   searchable in source).
5. Emit: `normStr` already covers the six values; no
   new mapping, no runtime unescape. Update comments
   claiming all AST strings are source-raw.
6. Grammar/tests/docs: interpreted-literal samples in
   `gramcheck`, parser/emit regressions, REQUIREMENTS
   compatibility amendment (canonical-spelling
   caveat: one value, two spellings, versioned
   explicit).

Out of scope: `\b \f \v \a` and numeric controls
(deferred in writing); a67 linked-pure runner; any
4107 relaxation (kept per verdict C).

## Rollback

`git checkout -- compiler/` plus delete
`compiler/str_escapes_test.go` and `docs/a65-*`.

## Test plan

- Probe first (red): `compiler/str_escapes_test.go`
  with the verdict's five utf8 rows (raw + escaped +
  decode-once + NUL + not-octal), the base64 rows
  (`e"QUJD"` success for the CAN4107 Ok arm plus the
  two rejects `e"QQ==\n"` and length-eight
  `e"QUJD\r\n\r\n"`), and the `e"\x51Q=="`
  CAN1000 rejection. Pre-fix the `e` forms do not
  parse. Pattern parity (`on e"a\nb"` taken by its
  row) and the untaken-e-arm source-spelling diag
  complete the set.
- Keep green: `TestStrSemanticsEmit`,
  `tail_backslash`, full `go test -count=1 ./...`.
- Post-fix: committed `.ts` unchanged by test-row
  additions (tests stripped); error-payload survival
  verified by structural equality in the rows.
- `go run ./tools/modcheck`,
  `go run ./tools/gramcheck`.
