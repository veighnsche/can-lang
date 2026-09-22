# a60: std__base64__decode wrapper + final acceptance (B15 scope)

Scope: the public stdlib face of the B14 kernel, plus the
workstream's closing consumer proof. No compiler change.
Last slice of issue #42's Bytes rollout (B1–B15).

1. `std/text/text.can`: `provides += std__base64__decode`.
   (`uses []` stays. Mod-level `emits` untouched per the
   `empty_separator` precedent.)
2. `std/text/text.can`: `fn std__base64__decode(value: str)
   -> Bytes__Value rev 1` after `std__base64__encode`,
   `emits [encoding.invalid_base64]`, relay body (`on Ok`
   + `on encoding.invalid_base64`, payload `e.value`
   unchanged). No per-fn comment. Rows, one line each:
   computed success (empty, `"AA=="`, `"QUI="`,
   `"Zm9v"`, `"+/8="`, `"0x00"`) and failure with
   unchanged propagation (odd, `0x`-prefix... correction:
   `"0x00"` is VALID; failures are `0x`-MISPLACED like
   `"=QUI"`, whitespace, `-_`, nonzero pad bits
   `"AE=="`/`"QUJ="`, strict lie, fidelity pair).
   No committed raw NUL/BOM: text.can stays a text file.
3. Header comment + `std/text/README.md`: base64 now has
   both wrappers; the #42 codec list is complete.
4. Regen: `text.ts` gains member + helper + fn.
   `errors.json`: the `invalid_base64` entry gains
   `handled_by` + `hit_by_tests`; everything else
   byte-identical.

Final consumer acceptance (this slice's extra): wrapper-
level round trips through REAL bodies in a temp-copy
probe (B4 pattern, local calls need no `given`):
utf8 `encode->decode`, hex `encode->decode`,
b64 `encode->decode`, each returning its input
unchanged, plus one cross check (hex of b64-decoded
bytes). Fakerows would prove nothing; these execute.

Out of scope: anything beyond #42 (downstream NUL
consumers remain unresolved per B4; base64url stays
deferred per the B14 row).

## Rollback

`git checkout -- std/text/text.can std/text/text.ts
std/text/errors.json std/text/README.md` plus delete
`compiler/bytes_b15_test.go` and `docs/a60-*`.

## Test plan

- Foreign-caller probe first (red): temp `client.can`
  with `uses [std__base64__decode@1]`, one `client__go`
  covering both arms (the arm-taken rule CAN4107 forbids
  single-arm fns), true outcomes scripted (success,
  identical-payload failure, NUL + non-ASCII malformed).
  CAN2102 before, clean after.
- Round-trip acceptance (red with the same gate):
  temp-copy probe fns asserting `decode(encode(x)) = x`
  per codec plus the cross check, all executed.
- `go test -count=1 ./...` (wrapper rows run in-suite;
  golden red until regen).
- Regen, `git diff` review (member + helper + fn),
  structural JSON check (only `invalid_base64`
  changes).
- Node vectors: emitted wrapper, valid/invalid/NUL/
  non-ASCII, tags + bytes + full payloads vs Go.
- `go run ./tools/modcheck`, `go run ./tools/gramcheck`.
- Close: verify B1–B15 row coverage against the plan
  table (report, not code).
