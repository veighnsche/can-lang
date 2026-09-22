# a31 — Identifier (`id`) attribute constructor

Status: shipped. The `id` slice: validate an identifier value,
serialize it as `id='...'`. No name brand — the name is fixed,
so there is no gate and no reconstruction. New error
`html.invalid_identifier(value: str)`; `html.nul_byte`
propagates from the reused worker, so the declared set is
`[html.invalid_identifier, html.nul_byte]` — an explicit
catalogue amendment (the catalogue lists only the former).

## Rule

- Policy, exactly: nonempty, no ASCII whitespace. The set is
  TAB/LF/FF/CR/SPACE, codes 9/10/12/13/32, verified against the
  live WHATWG Infra Standard before implementation.
- Uniqueness needs the element's tree and stays out. Nothing
  else is smuggled in: no trimming, no case folding, no
  normalization, no ASCII-only restriction, no letter-first
  rule (CSS escaping of leading digits belongs to a
  selector boundary, not here). Each would be a different
  policy.
- `html__attribute__id_ws(code: int)` classifies one scalar
  code. 9 rows pin all five whitespace codes true and four
  non-whitespace codes false — including vertical tab (11)
  and nonbreaking space (160), which a loose "whitespace"
  predicate would wrongly admit.
- `html__attribute__id_check(orig, s, n)` traverses with the
  shipped front-consumption shape (initial count is the scalar
  length; each step consumes one scalar and decrements).
  LF/CR need no literals: integer comparison recognizes them,
  integer test rows witness the classifier, expressible-string
  rows witness the traversal honoring it. Composition of the
  two is the complete five-code evidence — no exemption taken.
- `html__attribute__id(value)` rejects empty by explicit arm,
  runs the check, then the shared value worker unchanged, and
  seals `id='...'` from the actual local worker result — never
  from an independently supplied "already encoded" string.
  Raw input is validated as raw: `a&#32;b` contains literal
  ampersand/hash/digits (no space), passes, and encodes to
  `id='a&amp;#32;b'`. Decoding before validating would
  corrupt that identifier.
- Precedence is global, not first-offense: any
  empty-or-whitespace input reports `invalid_identifier` even
  when NUL also appears (the identifier scan does not stop at
  NUL); only identifier-clean inputs can report `nul_byte`.
  Both errors name the original input. This is an explicit
  error-priority policy, stable against future
  reimplementation — not an accident of the two-worker
  arrangement.

## Brand promise, stated accurately

`Html__Attribute` carries an approved name with
context-correct escaping. It does not promise parse-error-free
HTML for every admitted input: other controls (vertical tab
is the standing example) pass through per the
text-semantics precedent, and their parse-error status is a
shared serialization-layer question — not tree assembly's to
repair, and not this constructor's to solve with an ID-only
control policy. This clarification covers the shared worker
and brand retroactively; earlier slice docs assumed the
narrower reading without stating it.

## Inputs

- Jev returned 0.84 for check-before-encode precedence.
- The outside review confirmed the design and tightened two
  points, both adopted: the brand-promise boundary above,
  and the classifier/traversal evidence split (its
  LF/CR-witnessability challenge was answered by
  construction, with the runtime probe below closing the
  loop). It also caught the catalogue single-error mismatch,
  resolved by declaring both kinds.

## Proof costs

- New shapes proved in /tmp before use: match on a record
  bool field (`match c.ws`), integer-code classification.
- 19 new decision-table rows, 95/95 green with the
  pre-existing 76: 9 classifier rows, 5 check rows (clean,
  astral passthrough, space, tab, FF), 5 composer rows
  (plain, empty, space, encoded-lookalike, NUL layering).
- Raw control bytes in test rows, counted exactly like the
  a26/a29 rows (tab 0x09, FF 0x0C, NUL 0x00).
- Independent oracle: the generated `html.ts` runs under node
  over inputs can literals cannot express — LF, CR, VT,
  both NUL/whitespace precedence directions, the lookalike.
  12/12 green. Scratch probe at `/tmp/id-probe.mjs`, not
  committed.
- `errors.json` regenerated with complete `hit_by_tests`;
  `go test -count=1 ./...`, modcheck, and gramcheck all
  green fresh.

## Still scheduled (not silently dropped)

- `href`/`src` wait on the URL brand and its policy — the
  next attribute work in catalogue order.
- `classes` and the attributes-maker wait on Collections.
- The shared control/noncharacter serialization policy is
  open at the serialization layer (see the brand promise
  above); no constructor should invent its own dialect
  meanwhile.
