# a55: hex decode kernel (B10 scope)

Scope: `bytes__hex__decode` per plan a54 v2 (verdict folded).
Second fallible kernel; generalizes the B6 lowering instead
of duplicating it.

1. `compiler/export.go`: `bytesHexDecodeKernel` const +
   descriptor (`[value: str]`, `Bytes__Value`,
   `[encoding.invalid_hex]`) + `isBytesHexDecode`; extend
   `builtinErrorDecls` with `invalid_hex(value: str)`.
   Reused untouched: `isBuiltinError`, shadow rule,
   tycker seeding, `builtinErrorLookup`, module-union
   gating, catalog enumeration/attribution.
2. `compiler/eval.go`: `evBytesHexDecodeOp` (grammar-owned
   validation; host error checked and discarded, never
   propagated; malformed -> language error value, nil Go
   error; partial prefix discarded) + explicit dispatch
   branch (must not reach the encoder default).
3. `compiler/emit.go`: generalize `decodeResultUnion` +
   `stmtBytesDecode` by kernel name (descriptor-resolved
   record/emits; helper+flag selection per kernel; reject
   unknown); add `$canHexDecode` (validate-then-decode,
   explicit -1 sentinel, fresh buffer) + `$canHexVal`
   + `hexdec` flag with call-only emission. B6 D6 pins
   stay unchanged.
4. `compiler/bytes_b10_test.go`: X-rows (below).
5. Regen all goldens: every `errors.json` gains exactly
   `encoding.invalid_hex` (+ genuine fixture attribution
   nowhere — fixtures are test-only); zero `.ts` drift.

Deliberately untouched: `stmtBytesEncode`,
`stmtBytesHexEncode`, B6 vectors, stdlib (B11), catalog
filtering (all-declarations policy stands).

Out of scope: B11 wrapper, base64 (B12–B15).

## Rollback

`git checkout -- compiler/ docs/archive/sketches/ std/` plus delete
`compiler/bytes_b10_test.go`.

## Test plan

- Tests first (red): X0 fixture (valid incl. empty,
  mixed-case, long, NUL-output; invalid per class incl.
  parity-masking controls, fidelity row, neighbor rows);
  X1 missing arms both; X2 stale; X3 no-given; X4
  admission incl. brands through BOTH arms; X5 contracts;
  X6 emit pins (hex helper + union, TextEncoder/
  TextDecoder absent in hex-only fixture); X7 shadow;
  X8 wrong field type + wrong-payload CAN4200; X9 CAN3110
  both orders (`"zz"` and `"41zz42"` lies); X10 catalog
  attribution; X11 mixed probe module (seqClean +
  per-call union/helper assertions + emitted execution +
  foreign false-script CAN3110).
- `go test -count=1 ./...`; regen; structural delta
  check on all 8 catalogs; `cmp` all `.ts` files.
- Node vectors: emitted X0 fixture + emitted mixed
  probe, exhaustive 1-2 ASCII shapes + targeted
  (tab/newline/NUL/non-ASCII/neighbors/long) vs Go,
  tags + bytes + full payloads.
- `go run ./tools/modcheck`, `go run ./tools/gramcheck`.
