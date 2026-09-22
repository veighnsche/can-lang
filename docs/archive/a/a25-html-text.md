# a25 — First brand constructor (html__text__escape) plus the seal relaxation

Status: shipped. Row 5 opens with the text encoder and the first
blessed brand. Landing it required two compiler findings, both
proven by probe before anything was written.

## Rule

- `brand Html__Text is str rev 1`, declared in `std/html/`.
- `html__text__escape(raw)` → `Html__TextResult(text)` via a
  same-file worker; the entry seals the computed string, the one
  construction site in the module.
- The worker escapes `&`, `<`, `>` and passes everything else
  through, including the astral plane. No rejection error exists
  in the shipped contract. An earlier note claimed this totality
  was forced — control scalars allegedly inexpressible in
  literals, hence no witnessable reject arm. Probes falsified
  the inexpressibility premise: raw 0x00–0x08, 0x0B–0x0C,
  0x0E–0x1F, and 0x7F all parse, evaluate, and emit faithfully
  (only newline breaks line syntax), so control reject-arms ARE
  witnessable and the coverage law does not force totality here.
  The passthrough stands on HTML text semantics (controls are
  preserved in text context), not on necessity. One open point:
  NUL is a parse error (→ U+FFFD) in HTML text, and the encoder
  has no explicit NUL policy yet — reject-with-error versus
  documented passthrough is undecided. Shipped tables cover the
  printable plus astral domain; controls need explicit rows under
  either policy. Quotes stay unescaped here by the text-context
  contract; attributes get their own constructors.

## Why the seal rule changed

Seals wrapped string literals only, so no function could seal
computed text — the constructor row was unimplementable, and the
tables passed while the seal was rejected. The rule is now:
same-file seals take string literals or string-typed refs and
fields; brands never seal brands (fail closed: only `str`
passes). The file-ownership ban (CAN6004) and the grep-seal
audit are untouched. What changed is the minting discipline for
computed brands: decision tables prove the output instead of
eyeballing literals. The evaluator already accepted arbitrary
string expressions; only the static gate and the emitter's
literal-only branch stood in the way, and both now agree.

## Why wrapper records, not bare-brand returns

The specified `(raw: str) → Html__Text` shape is unexpressible
twice over: the declaration gate rejects brand returns, and
behind it values bound from `Ok` are record dicts at runtime —
a bare-brand flow would typecheck nominally and corrupt
silently. Wrapper records (`Html__TextResult(text)`) keep every
existing mechanism sound: fields resolve through declarations,
params check nominally, tests compare erased strings, emit
erases the seal. Bare-brand returns need a representation
design, not just a gate change; that project is named and
declined for this slice.

## Proof costs

- No new recursion: one unit worker over the remaining count,
  front-consumption shape, the `upper_ascii` precedent — no
  index arithmetic, no out-of-range shape past the bound.
- No linkage interaction, no effects, no kernel changes, no TS
  helpers.
- Compiler: one widened condition in the tycker plus the
  matching emit branch and evaluator message; the old
  literal-pin test becomes the string-boundary test
  (`TestDiagnoseSealString`: int literals and int refs still
  rejected).

## Implementation

- `compiler/types.go`, `emit.go`, `eval.go`: seal takes a
  string (message updates to match).
- `compiler/lsp_test.go`: the superseded literal expectation
  becomes the new boundary pin, with the rule change stated.
- `std/html/`: brand, two types, worker plus entry (12 table
  rows); goldens frozen, `errors.json` empty.
- No new Go unit paths beyond the seal pins (no new runtime
  machinery — the html tables plus the golden cover the slice).

## Still scheduled (not silently dropped)

`html__text__node` promotion, the attribute and URL
constructors, fragments, and elements — each its own slice, in
program order. Control rejection anywhere in the language waits
on writable control scalars in literals (a language feature),
not on policy debate.
