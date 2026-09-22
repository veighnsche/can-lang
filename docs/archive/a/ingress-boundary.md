# Row 1 — Unicode ingress boundary decision

Status: investigated and decided. Every currently supported string
ingress route below ends in **enforced**, **trusted under an
explicit contract**, or **unsupported** — "probably supplied by the
host" appears nowhere. NUL and the control characters are Unicode
scalar values, not malformed input: they belong to the encoder
policy (row 5), not to this boundary.

## Probe facts (all witnessed against the real compiler)

- Literals keep raw source bytes (`parseSmall`: `Str: s[1:j]`, no
  decoding, no validation). Invalid bytes (`FF FE`) parse with no
  diagnostic and table-compare by raw equality.
- The evaluator reads via `[]rune` (substitution per bad byte;
  a20-documented). The emitter substitutes U+FFFD at compile time
  into the golden — so tested bytes and shipped bytes differ for
  malformed input, while the two runtimes agree downstream.
- A side probe collapsed honestly: a text-mode-written fixture
  tested U+00FF (valid) rather than raw `FF`; the binary-written
  fixtures above are the valid ones. The eval code, not the void
  probe, establishes `[]rune` semantics.
- `str_to_int`/`str_to_dec` accept ASCII `48–57` scalar values
  only; anything else is a typed rejection. Decoders enforce.
- Internal producers (`concat`, slices, `replace_all`, case maps,
  trims, `int_to_str`) operate on scalar sequences or generate
  ASCII: preservation is conditional on valid input, which is
  exactly why ingress enforcement matters.

## Boundary table

| Route | Finding | Decision |
|---|---|---|
| Source decoding / literal construction | Malformed entered silently; emit repaired silently and divergently | **REPAIRED (row 4)**: `parseModuleText` refuses malformed UTF-8 (`CAN1000`; all four routes share the choke point), pinned by `TestMalformedSourceRefused` on CLI and editor paths |
| Extern success/error payloads | Real route: host-implemented externs return `str`-bearing records in prod (sketches), scripted in tests. Well-formedness is currently nobody's job | **TRUSTED under explicit contract**: host functions must return valid scalar-value strings, records included. Enforcement (runtime validation of host returns) is disproportionate now; the condition is written down instead of assumed. |
| TypeScript entry points | Emitted functions take unchecked `string`; lone surrogates spread to lone-surrogate code points where Go yields FFFD — live runtime divergence for host-supplied strings | **TRUSTED under explicit contract** on emitted modules: callers supply valid scalar-value strings. Same reasoning as externs; not silently redesignated unsupported. |
| Internal string operations | Conditional preservation (see above) | **CONDITIONAL theorem**: holds iff ingress holds. No per-operation validation. |
| Decoders (`str_to_int`, `str_to_dec`) | Strict ASCII, total-with-error | **ENFORCED** already; noted, no change. |
| Future decoders (utf8/base64/hex) | Do not exist | Reject malformed input or specify repair explicitly at design time. Rule recorded. |

## Consequences

- The source-decode rejection was a **demonstrated defect repair**
  (row 4, shipped): silent entry plus divergent repair, on a
  supported path, against the exact-or-loud doctrine. One check
  at the shared choke point; valid sources byte-identical in
  behavior (full suite green).
- The TS-entry and extern conditions must appear where a host
  author looks (emit header / extern contract docs), not buried
  here alone.
- Internal preservation needs no code; decoders need no change.
- Out of scope, explicitly: NUL/control policy (row 5),
  newline-in-literal syntax (line structure, unchanged).
