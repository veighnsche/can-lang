# a54: hex decode kernel (B10 plan v2 — verdict folded)

Goal: `bytes__hex__decode`, the second fallible kernel: `str`
in, `Bytes__Value` on success, `encoding.invalid_hex` on
malformed input. Mirrors B6; B11 (stdlib wrapper) is separate.

Verdict on v1: contract holds with tightened lowering and
tests. Grammar, str payload, single kind, empty success, and
arbitrary-byte output all stood; Go/TS parity found no
counterexample (79,927 probe cases, 0 mismatches — host
probes only, emitted-fixture comparison still required).
All corrections folded below. No B10 gates run.

## Contract

```text
bytes__hex__decode
  params: [value: str]
  ret: Bytes__Value
  emits: [encoding.invalid_hex]
  restricted: false
```

```can
error encoding.invalid_hex(value: str)
```

No new record. Payload is the ORIGINAL string: not its UTF-8
encoding, not decoded bytes, not normalized, not trimmed.
Fidelity row: `"aFzz"` must come back exactly `"aFzz"`
(not lowercased, not suffix-only, not partial bytes).
Compiler-owned (B6 pattern): one descriptor + one typed
error decl. Every regenerated catalog gains the entry;
`raised_by` is the kernel entry PLUS any source relay in
that build (union, not replacement). Never add the kind to
unrelated functions' `emits` — catalog inventory and
outcome contracts are different things.

## Grammar: valid iff entire input is `(HH)*`

`H = [0-9a-fA-F]`, even length, every unit in range.
Mixed-case accepted (`"aF"` -> `[175]`). Empty is valid
-> `Ok` of empty Bytes.

Invalid (row each): odd (`"f"`, `"abc"`, long-odd),
`0x`/`0X`, space/tab/newline (each position class),
interior NUL, non-ASCII (`"é"`, CJK, astral, BOM char),
non-hex ASCII (`"zz"`, `"g0"`, `"../"`, `"--"`),
truncated pair alone and after valid text. Single kind
for odd-length and bad-byte alike (Go's internal
length-vs-byte precedence is not exposed; a
length-first TS check stays observationally equivalent).

Parity-masking controls (verdict addition): even-UTF-16
non-ASCII (`"é0"`, `"0中"`, `"😀"`, `"0"+BOM` — length
alone would pass these), whitespace/NUL in even strings
in both nibble positions and after valid prefixes, ASCII
neighbors `/ : @ G backtick g` (off-by-one exposure).
Real control characters in evaluator/Node inputs only —
no backslash escapes (no new literal feature in B10).

Round-trip laws: `decode(encode(b)) = b`;
`encode(decode(h)) = lowercase(h)` on valid `h` only
(never lowercase a rejected payload). `"00"` -> `[0]`
(NUL output is data); `"eda080"` decodes (no UTF-8
pre-judgment of output bytes).

## Lowering

- Go: grammar-owned validation; `DecodeString` may own
  both steps provided EVERY non-nil host error discards
  Go's partial prefix and becomes the original-string
  language error. Malformed -> error `Value`, nil Go
  error (B6 rule — CAN3110 depends on it). Partial bytes
  (`"00ffg0"` -> `[0,255]` + error) must NEVER escape
  as success.
- TS: validate-then-decode, no `parseInt`. Full
  char-code pass FIRST (`48-57, 65-70, 97-102`, even
  length, every unit), then nibble-table decode into a
  fresh `Uint8Array`. Never truncate a unit to a byte;
  no low-nibble arithmetic as admission (safe only after
  both nibbles proved in `0..15`; use an invalid sentinel
  or the full pass first). Corrected claim: "Malformed
  input returns `invalid_hex`; validation prevents
  malformed-input indexing and conversion faults.
  Host-contract and resource failures remain loud, never
  relabeled." No blanket catch (allocation stays loud).

## Shared fallible lowering (Q3: limited generalization)

Generalize `decodeResultUnion` and `stmtBytesDecode` by
kernel name for the two real fallible kernels; keep each
codec's transform separate. Every hardcoded UTF-8 choice
goes: descriptor-resolved success record + emits, helper
selection + usage flag, exact per-kernel tmp unions:

```ts
ok/string | invalid_utf8/Uint8Array   (utf8 — unchanged)
ok/Uint8Array | invalid_hex/string    (hex — new)
```

No `string | Uint8Array` payloads; tag determines type.
Reject missing/unsupported descriptors, never fall back.
Eval gets an explicit hex branch (must not reach the
encoder default). Verdict correction: B6 D6 substring
pins do NOT prove call-site correctness — require a
BOTH-decoders module with per-call union/helper
assertions plus emitted execution of both paths; B6
vectors stay unchanged. Absence checks (`TextEncoder`/
`TextDecoder`) scope to the hex-only fixture, which must
contain neither.

## Tests

- Kernel fixture, no local copies: valid + invalid rows
  above incl. masking controls and fidelity rows.
- Proof rules with diagnostics: missing Ok arm, missing
  error arm, stale arm, forbidden `given`, admission
  (Bytes/int/brand -> CAN6003 `want str`) — THROUGH BOTH
  ARMS: valid-looking `Secret("41")` and malformed
  `Secret("zz")` (the error path would leak the brand's
  representation without admission).
- Wrong-payload error expectation fails CAN4200 (tag
  alone proves nothing).
- Linkage both orders: `"41zz42"` scripted as
  `Ok([65])` (prefix-acceptance attack) AND `"zz"`
  scripted as success; require CAN3110. Provider owns
  valid+invalid direct rows.
- Mixed probe module (test-only, NOT stdlib): the
  verdict's hex->utf8 chain (`"41"`, `"ff"`-then-reject,
  `"41zz42"`, `"00ffa"`), exact per-call unions,
  emitted execution of every arm, then the foreign
  false-script CAN3110 both orders.
- Execution parity: emitted fixture both outcomes, tags
  + bytes + full payloads; node vectors vs Go incl.
  exhaustive short malformed shapes.
- Brand encode/decode rejection repeated with the real
  decoder (no new admission path). B11 stays separate.
- Regen: all catalogs gain exactly the new entry
  (+ genuine fixture attribution); no unused helper or
  outcome in unrelated TS.
- Gates: `go test -count=1 ./...`, `modcheck`, `gramcheck`.
