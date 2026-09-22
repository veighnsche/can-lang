# a59: base64 decode kernel (B14 plan v2 — verdict folded)

Goal: `bytes__base64__decode`, the third fallible kernel:
`str` in, `Bytes__Value` on success,
`encoding.invalid_base64` on malformed input. Mirrors B10;
B15 (stdlib wrapper) is separate.

Verdict on v1: contract holds with a corrected padding-bit
rule. Required padding, whitespace rejection, standard
alphabet, original-string payload, arbitrary-byte output
all stood; parity probes (346,745 cases, 0 mismatches —
host probes only, emitted-fixture comparison still
required). The defect: `DD==` has FOUR unused bits,
`DDD=` has TWO — one mask for both is wrong either way.
All corrections folded below. No B14 gates run.

## Contract

```text
bytes__base64__decode
  params: [value: str]
  ret: Bytes__Value
  emits: [encoding.invalid_base64]
  restricted: false
```

```can
error encoding.invalid_base64(value: str)
```

No new record. Payload is the ORIGINAL string: no
normalization (case-sensitive — unlike hex), no trim, no
BOM handling, no reconstruction. Compiler-owned (B6/B10
pattern). Catalogs gain the entry everywhere; `raised_by`
is kernel-only where no source function re-raises (union
otherwise). Never add the kind to unrelated `emits`.

Padding REQUIRED because the chosen contract requires it
(workstream examples admit `"Zm9v"`, reject `"Zg"`/`"Zg="`),
not because optional padding would eliminate placement
errors. No standard/raw fallback combination. Empty is
valid -> `Ok` of empty Bytes.

## Grammar: `(DDDD)* (DDD= | DD==)?` over standard alphabet

`D` = `A–Z a–z 0–9 + /`; `=` padding only. Length multiple
of 4 (empty succeeds). Final quartet `DDDD` (no check),
`DDD=` (two-bit check), or `DD==` (four-bit check);
nothing afterward. Exact masks:

| Final quartet | Output | Check (v = sextet value) |
| ------------- | -----: | ------------------------ |
| `a b c d`     |      3 | none                     |
| `a b c =`     |      2 | `(v(c) & 3) == 0`        |
| `a b = =`     |      1 | `(v(b) & 15) == 0`       |

Counterexamples locking each (all direct rows):
`"QUI="` succeeds (universal-4-bit would reject it);
`"AE=="` fails (universal-2-bit would accept it);
`"QUJ="` fails (no-check would accept it as `"QUI="`).
Positives: `"AA=="`, `"QUI="`; unpadded `"AAAB"`,
`"////"` (no mask where no padding); repeats after a
full quartet.

Rejected: whitespace everywhere incl. CR/LF alone, after
a quartet, and between the two `=` (decisive: `"QUJD\r\n\r\n"`,
length 8 — length cannot explain it); URL `-_`
(matched pair: `"+/8="` ok `[251,255]`, `"-_8="` fails —
but shared-alphabet strings are never rejected for
their producer's label); NUL/non-ASCII/BOM (length-valid
controls `"AA😀"`, `"éééé"`, `"AAA"+NUL`); misplaced
padding (`"=QUI"` is LEADING padding, `"QQ==AAAA"`,
`"QUI=AAAA"`, `"===="`, interior `=`); missing padding
(`"QQ"`, `"QUI"`, `"QUJD="`); odd/mod-4-bad lengths.

`"0x00"` is VALID (`[211,29,52]`) — positive regression
against B10-copied prefix logic. Case changes values;
never normalize, including in payloads.

## Lowering

- Go: `StdEncoding.Strict()` (rejects nonzero pad bits;
  ordinary mode accepts them — verdict Q4 table) PLUS
  explicit CR/LF rejection (both Go modes skip newlines)
  PLUS the complete grammar prevalidator (owns the full
  no-whitespace contract either way). Host error checked;
  partial prefix discarded ALWAYS (`"QUJDQUJ="` strict
  yields `[65,66,67]` + error -> must become
  `invalid_base64` of the whole string). Malformed ->
  language error value, nil Go error.
- TS: validate-then-decode, hand-rolled, NO `atob` as
  validator (forgiving algorithm strips whitespace,
  admits unpadded input, drops nonzero bits — do not
  delegate admission; post-validation use would be
  sound but the hand decoder avoids the dependency).
  Steps: length % 4; every data-unit code in range;
  nonfinal quartets all-data; final shape + trailing
  checks; output length `3*(n/4)-p` (no 32-bit coercion
  of lengths). Fresh buffer; explicit invalid sentinel;
  no catch (allocation stays loud).
- Shared fallible path a third time: descriptor row,
  explicit eval branch, `isFallibleDecode` membership,
  helper selection + `hexdec`-style flag + insertion.
  Per-call union (verdict: base64 and hex share input
  AND payload types, so a wrong helper stays well-typed
  — discriminator `"4142"`: hex -> `[65,66]`, base64 ->
  `[227,94,54]`; assert correct helper AND kind per
  call site, not mere co-presence).

## Tests

- Kernel fixture, no local copies: valid rows (empty,
  all padding shapes, mixed alphabet, long, NUL-output
  `"AA=="`, high `"/w=="`, `"0x00"`, `"+/8="`,
  `"AAAB"`, `"////"`); invalid per class incl. masking
  controls, fidelity `"aFzz"`-analog `"QUJD!!!"` AND
  `"QUJD!!!!"` (length-8, alphabet must explain it),
  neighbor nibbles, both-mask matrix with positives.
- Proof rules with diagnostics: missing arms both,
  stale arm, forbidden `given`, admission (Bytes/int/
  valid-brand/malformed-brand -> CAN6003 `want str`,
  error-arm leak covered), wrong field type, shadow,
  wrong-payload CAN4200.
- Linkage, PER-LIE CAN3110 both orders (verdict: no
  aggregate check — B10's X9 aggregate stays as
  shipped, B14 asserts each lie): prefix-attack
  `"41zz42"`-analog `"QUJD!!!"` as `Ok`, plus
  `"QUJDQUJ="` scripted as `Ok([65,66,67,65,66])`
  (unchecked-pad-bits attack). Provider owns
  valid+invalid rows.
- Mixed probe (test-only, verdict's exact module):
  `"QQ=="`->`"A"`, `"/w=="`->`invalid_utf8([255])`,
  `"41zz42"`-analog, `"00ffa"`-analog; exact per-call
  unions; emitted execution of every arm; foreign
  false-script of the strict lie, CAN3110 both orders.
- Execution parity: emitted fixtures both outcomes;
  node vectors exhaustive over short malformed shapes
  vs Go (NEVER bare `StdEncoding` as oracle — it is
  more permissive than the policy by construction).
- Brand encode/decode rejection repeated (no new
  admission path). B15 stays separate.
- Regen: all catalogs gain exactly the new entry;
  no unused helper/outcome in unrelated TS.
- Gates: `go test -count=1 ./...`, `modcheck`,
  `gramcheck`.
