# a50: UTF-8 decode kernel (B6 plan v2 — verdict folded)

Goal: `bytes__utf8__decode`, the first FALLIBLE codec kernel:
`Bytes` in, `Encoding__Text` on success, `encoding.invalid_utf8`
on malformed input. B7 (stdlib wrapper) is a separate slice.

Verdict on v1: REQUEST CHANGES. Validity grammar, original-Bytes
payload, whole-input rejection, and empty-input success stood;
the TS decoder config, builtin-error plumbing, and decode
dispatch needed correction. All folded below. No B6 gates run.

## Contract (Q1, Q2 resolved: compiler-owned)

```can
type Encoding__Text rev 1 (
  value: str
)

error encoding.invalid_utf8(value: Bytes)
```

Both live in the compiler contract registry alongside
`Bytes__Value`/`bytesKernels` — NOT in a new
`std/encoding/encoding.can`, NOT in caller `provides`/`uses`.
Bytes-typed error fields are already supported (B1
`TestBytesV4StructuralEq` proves `error m.boom(value: Bytes)`
end-to-end); B6 newly proves it through the COMPILER-OWNED
path. No string/hex/base64 substitute carrier.

Descriptor pins params as well as types:

```text
bytes__utf8__decode
  params: [value: Bytes]
  ret: Encoding__Text
  emits: [encoding.invalid_utf8]
  restricted: false
```

## New builtin-error plumbing (verdict blocker 1)

Builtin record support does NOT imply builtin error support.
B6 establishes one builtin-aware error lookup used by:
`buildWorld`/`Program.Errors` (fields `["value"]`),
`newTycker` constructors/bindings/expectations (Bytes type
retained), `errorShapes`/`tsErrMember`/`fnResultUnion`/
`externUnion` (payload `Uint8Array`), module result unions
(only builtins the module's outcomes require — no global
union pollution), shadow checks (source cannot redeclare
either contract, even identically).

Catalog: `buildCatalog` enumerates all of `prog.Errors`, so
global registration adds the entry to EVERY regenerated
`errors.json`. Preserve that policy; accept and document the
churn. Attribute the intrinsic as
`kernel.bytes__utf8__decode` with empty `handled_by`/
`hit_by_tests` where no source references exist. Do NOT
silently filter builtins to protect goldens.

A standalone kernel fixture must compile WITHOUT declaring
its own copy of either contract.

## Grammar: valid iff Go `utf8.Valid` over the whole input

Invalid classes (row each): overlongs, surrogate halves
(`ED A0..BF`), above U+10FFFF (`F4 90..`), stray
continuations, `F5–F7` leads (verdict addition), `F8..FF`,
truncation at every length after every valid prefix/suffix.
Valid boundaries: U+007F/0080/07FF/0800, U+D7FF/U+E000
(surrogate gap), U+FFFD (valid scalar — never reject by
"detection"), U+FFFF (noncharacter, valid), U+10000/U+10FFFF.
No normalization. Empty input → `Ok(value = "")`. BOM →
U+FEFF preserved, never stripped. NUL → U+0000 preserved as
valid UTF-8 data (HTML escaping stays the separate rejecting
boundary; no `html.nul_byte` here).

Truncation (Q4): ONE whole-payload error, no index or length
metadata. `truncated_after_ascii([65, 226, 130])` must yield
`invalid_utf8` carrying exactly `[65, 226, 130]` — never
prefix-accept, prefix-drop, or silent repair.

## Decode dispatch (verdict blocker 2)

Current `isBytesKernel` dispatch is encoder-only in both
`eval.go` (`evBytesEncodeOp`) and `emit.go`
(`stmtBytesEncode`). B6 adds an explicit decode branch; a
decoder descriptor alone would misroute today.

- Go: malformed input returns a language error `Value`
  (`Kind "err"`, `ErrKind "encoding.invalid_utf8"`,
  `Dict["value"]` = ORIGINAL Bytes) with NIL Go error.
  Only after `utf8.Valid` succeeds, construct the string.
- TS result: `{ $can_kind: "ok"; value: string } |
  { $can_kind: "encoding.invalid_utf8"; value: Uint8Array }`.
- Strategy: retain operand once, validate the complete
  grammar, invalid → error(original), valid → decode with
  `fatal: true, ignoreBOM: true` (verdict Q3 fix — default
  `TextDecoder` STRIPS a leading BOM: `[239,187,191]`
  decoded to `""` instead of U+FEFF; 65,848-case probe,
  corrected config 0 diffs). No blanket catch; host
  misuse/resource failures stay separate from malformed.
- Decode the `Uint8Array` VIEW, not its backing buffer
  (pin: `subarray(1,2)` of `[255,65,255]` → `"A"`).
- One-shot only: no `stream: true` (defers truncation).

## Linkage fixture (highest-risk hole)

`contradictScriptOk` treats provider Go-errors as "not
contradicted" — so a Go-error implementation would let a
false scripted success pass. Required scenario, both module
orders, diagnostic CAN3110:

```text
provider input:  Bytes(Seq<int>[65, 226, 130])
provider result: encoding.invalid_utf8(same bytes)
false script:    Ok(value = "A")  → must fail CAN3110
```

Provider needs its own valid + invalid direct rows. Invalid
UTF-8 is a computed comparable result, not a modeling gap.

## Test plan

- Kernel fixtures (B3 pattern, NO local contract copies):
  valid boundaries/gap/U+FFFD/noncharacters/BOM
  (only/text/repeated/interior/BOM-then-malformed)/NUL
  positions/empty; invalid per class incl. six nonempty
  proper prefixes of 2-/3-/4-byte sequences alone and
  after valid text; payload fidelity incl. valid
  prefix+suffix with wrong-payload expectation that fails.
- Proof rules, each asserting its diagnostic: missing error
  arm, missing Ok arm, stale arm (declared unrelated
  error), forbidden `given`, strict admission, wrong
  field type, shadow redeclaration rejection.
- Linkage: CAN3110 false-success scenario both orders.
- Execution parity: actual emitted fixture, both outcomes;
  tags + text + full error bytes; independent success
  after a failure; re-encode comparison (never Go bytes
  vs UTF-16 lengths).
- Node adversarial vectors vs Go through the real result
  protocol.
- Unrelated-brand encode→decode rejection fixtures
  repeated with the real decoder (no new admission path;
  export design untouched).
- Regen: `errors.json` gains the builtin entry
  EVERYWHERE (expected delta, existing entries unchanged);
  no unused decoder helpers or error variants in
  unrelated TS.
- Gates: `go test -count=1 ./...`, `modcheck`, `gramcheck`.
