# a57: std__hex__decode wrapper (B11 scope)

Scope: the public stdlib face of the B10 kernel. No compiler
change. Same shape as B7 (first-fallible-wrapper precedent).

1. `std/text/text.can`: `provides += std__hex__decode`.
   (`uses []` stays. Mod-level `emits` untouched per the
   `empty_separator` precedent.)
2. `std/text/text.can`: `fn std__hex__decode(value: str) ->
   Bytes__Value rev 1` after `std__hex__encode`, `emits
   [encoding.invalid_hex]`, relay body (`on Ok` + `on
   encoding.invalid_hex`, payload `e.value` unchanged). No
   per-fn comment (file convention). Rows, one line each:
   computed success (empty, `"00"`, lower/upper/mixed,
   `"deadbeef"`, `"eda080"`) and failure with unchanged
   propagation (odd, `0x` prefix, interior space,
   fidelity `"aFzz"`, prefix attack `"41zz42"`,
   truncated `"a"`). No committed raw NUL/BOM: text.can
   stays a text file; NUL/BOM relay rides the Go probe.
3. Header comment + `std/text/README.md`: extend the hex
   pointer to the decode wrapper.
4. Regen: `text.ts` gains the union member, the
   `$canHexDecode` helper (first hex decode call site in
   stdlib), and the wrapper fn. `errors.json`: the
   `invalid_hex`
   entry gains `handled_by` (relay arm) + `hit_by_tests`
   (failure rows); every other entry byte-identical. Any
   further delta fails the slice.

Out of scope: base64 (B12–B15).

## Rollback

`git checkout -- std/text/text.can std/text/text.ts
std/text/errors.json std/text/README.md` plus delete
`compiler/bytes_b11_test.go`. No compiler or golden-test
change ships in this slice.

## Test plan

- Foreign-caller probe first (red): temp `client.can` with
  `uses [std__hex__decode@1]`, two fns covering both arms
  (CAN4107): success + failure + NUL-input... NUL as INPUT
  str needs raw-byte splice in the temp source (fine —
  temp only); BOM/non-ASCII malformed inputs. True
  outcomes scripted through `given`. Fails before the
  .can change (CAN2102), passes after.
- `go test -count=1 ./...` (wrapper rows run in-suite;
  golden red until regen).
- Regen, inspect `git diff` (member + helper + fn in
  `text.ts`; handled/hit population in `errors.json`
  only), structural JSON check of all other entries.
- Node vectors: extract emitted `std__hex__decode` (+
  helper) from fresh `text.ts`, run valid/invalid/NUL/
  non-ASCII inputs, compare tags + bytes + full payloads
  with Go.
- `go run ./tools/modcheck`, `go run ./tools/gramcheck`.
