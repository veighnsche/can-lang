# a27 — Attribute-name gate (small exact lowercase allowlist)

Status: shipped. Row 5 continues with the first attribute operation.
The catalogue specifies `html__attribute__text` taking an
`Html__TextAttributeName` but lists no constructor for the name
brand; this slice blesses the gate that mints it. The value encoder
and `attribute__text` composition belong to a28.

## Rule

- `brand Html__TextAttributeName is str rev 1`, declared in
  `std/html/`. Definition: exactly the canonical names approved by
  this library revision for the ordinary quoted-text
  attribute-value constructor, within supported element contexts.
  Narrower than "a valid HTML attribute name," more useful than "a
  name absent from three dangerous categories."
- `html__attribute__name(value)` → `Html__NameResult(name)` or
  `html.invalid_attribute_name(value)`. Exact-match arms seal
  canonical literals, so every success visibly mints a reviewed
  member; no slicing, no normalization, no imported helper.
- Initial membership is exactly `{title}`: tooltip text, inert,
  needs no element context. Admission rule for later names: each
  needs its own HTML-contract justification. Explicitly excluded:
  `id`/`class` (specialized constructors own them),
  `srcdoc`/`meta`-content (wrong value context — escaping outer
  syntax never repairs inner interpretation), and everything
  unlisted (API rejection, not a claim of HTML invalidity).
  "Text-valued" is never advertised as "incapable of affecting
  application behavior" (DOM clobbering needs no separate rule
  while additions require justification).

## Why allowlist, and the corrected derivation

An earlier frame claimed denylisting forces case-folding (hence
linkage or duplication). That exclusivity claim is false: a
canonical-syntax grammar plus denial, or even enumerated case
variants, recognizes without transforming. What survives is the
sufficient-condition claim — exact lowercase allowlisting avoids
both normalization and a general name scanner, making it the
simplest self-contained implementation — plus the safe-failure
direction: the next dangerous name is rejected by default, and
completeness needs no argument testing cannot supply. A denylist
over arbitrary strings would need a lexical argument and an
account of every remaining name's interpretation; the positive
case for denying strengthens only as its universe closes, which
erodes the short-list advantage.

Jev returned 0.37 on the allowlist as framed — retrospectively
sound, because the frame asked for a verdict on an unspecified
`…` membership no judge could verify. The outside review supplied
the missing verdict shape (mechanism yes, membership reviewed
per entry) and the `srcdoc` counterexample that makes "not URL,
not event-handler, not style" an incomplete definition.

## Alternatives recorded, not shipped

- Closed zero-arg factories (`html__attribute__name_title()`)
  remove arbitrary strings from the boundary entirely and need no
  rejection branch. Retained validator because the catalogue's
  composition passes names as data; never ship both interfaces
  without a caller that needs each.
- Canonical-grammar-plus-deny stays available if a future slice
  needs a broader universe with a closed, classified membership.

## Proof costs

- No new recursion, no linkage, no kernel changes, no TS helpers.
  One flat match; the wildcard arm carries the single error.
- Evaluator runs all 20 rows at compile time; the golden freezes
  the emit side; `errors.json` is generated with complete
  `hit_by_tests`.
- Consumer forgery stays refused under the existing ownership
  rule (no new test needed — CAN6004 machinery unchanged).

## Still scheduled (not silently dropped)

Note: the a28 number went to parallel multi-scrutinee-match
research, so this work shipped as a29 (`docs/a/a29-attribute-value.md`).

a29 owns the attribute-value encoder and the fate of the specified
`invalid_attribute_value`: with names arriving as brands and the
value encoder total, the error is dropped with a a25-style
justification unless a witnessable value-side reject is found. One
standing distinction for a28: inexpressible-in-literals is a limit
of test syntax, not proof of absence at runtime — an
unwitnessable-but-reachable value rejection would be an
evidence-mechanism problem, not a license to delete the check.
Boolean names, remaining attributes, fragments, elements follow in
program order.
