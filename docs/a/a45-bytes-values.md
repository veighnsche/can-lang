# a45 S1: Bytes value admission and literal construction (B1)

First execution slice of the Bytes workstream
([bytes-plan.md](bytes-plan.md) B1).
Template: [a36-seq-typed-construction.md](a36-seq-typed-construction.md).

## Scope (one admission boundary)

- `Bytes` as a known primitive (`knownType`, `typeOf`, `value`/`checkCtor`,
  reference annotations, record/error field admission).
- Literal-only construction `Bytes(Seq<int>[...])`: members must be
  integer-literal AST nodes in `0..255` (`big.Int` comparison, no narrowing).
  New codes `CAN6008 CodeBytesLiteral`, `CAN6009 CodeBytesElementRange`,
  `CAN6012 CodePrimitiveShadow` (reserves the `Bytes` name).
- Evaluator `bytes` kind (owned `[]byte`), constructor validation, structural
  `vEq`, canonical `normalizeValue` spelling `Bytes(Seq<int>[...])`.
- Emission: `tsBase` `Bytes → Uint8Array`, numeric-literal lowering
  (`Uint8Array.from([0, 255])`), defensive direct-comparison rejection,
  conditional typed-array branch in the structural equality runtime.
- `Seq<Bytes>` composition via the uniform element rule; nested sequences
  stay rejected. Direct Bytes `==`/`!=`/ordering rejected; structural `==`
  over admitted records and error payloads preserved (with rows).
- Bare `-> Bytes` returns rejected (wrapper-record rule, Seq precedent).
  Bytes state stays rejected (`CAN6002`, regression row added).

## Non-scope (written deferrals, not accidental gaps)

No codecs, no export authority, no Render consumer, no std wrappers, no
runtime `Seq<int> → Bytes` conversion, no hex literal sugar, no base64url, no
direct byte operators, no state cells, no JSON/random/hash. No `.can` changes.

## Rollback

Revert the slice commit. Nothing outside `compiler/` + this doc + the
`docs/README.md` row moves, so no std regeneration or catalog repair can
linger. `CAN6008/6009/6012` stay registered (codes never change meaning).

## Acceptance (all in `compiler/bytes_b1_test.go`, helpers `seqClean`/`seqCode`)

| # | Row | Gate |
| - | --- | ---- |
| V0 | Empty `Bytes(Seq<int>[])` is a value | clean |
| V1 | Nonempty ordered `[0, 127, 128, 255]` with repeats | clean |
| V2 | Construction in every value position (body, arg, record field, test arg, expectation, given arg/result) | clean |
| V3 | `Seq<Bytes>` empty/nonempty, append, checked access, wrong-member + `Seq<int>`/`Seq<Bytes>` mismatch rejections | clean/code |
| V4 | Record + nested record + error payload + `Seq<Bytes>` structural `==` rows (Go evaluator) | clean |
| V5 | Byte-result linkage mismatch → `CAN3110`; wrong direct expectation → `CAN4200` | code |
| V6 | Emitted TS pins: `Uint8Array.from([0, 255])` numeric literals; helper absent when unused, single when shared | Go test asserts emitted text |
| T0 | `Bytes(Seq<int>[-1])` / `[256]` / huge int → `CAN6009` with index + value | code |
| T1 | `Bytes(Seq<int>[i])` runtime member, `Bytes(xs)` → `CAN6008` | code |
| T2 | `Bytes([0, 1])` → `CAN6007`; bare-`Seq` annotation → `CAN6002`; malformed `Seq<...` → `CAN1000`; unbound `Seq` value → `CAN6003` | code |
| T3 | Direct `==`/`!=`/ordering over Bytes → compile error | code |
| T4 | Bare `-> Bytes` return → compile error; `state x: Bytes` → `CAN6002` | code |
| T5 | `type Bytes` / `brand Bytes` shadow → `CAN6012` | code |

Gates after the slice: `go test -count=1 ./...`, `go run ./tools/modcheck`,
`go run ./tools/gramcheck` — all green at implementation time. Emitted-TS
pins run against the Go outcomes: node vectors executed against the real
emitted `m.ts` (`Uint8Array.from` contents incl. empty, `$canEqBytes`
equal/length/order/typed-vs-plain, `$canEqVal`/`$canEqRec` dispatch) — all
pass. Probe scripts live in `/tmp/bytesvec` (scratch, not committed); the
committed evidence is the static pins in `compiler/bytes_b1_test.go` V6.
