# a16 — Text semantics and construction (part)

Status: landed (part). String concatenation ships; the indexing
semantics are decided; the scalar-access family is specified with
its surface left open.

## What ships

- `+` concatenates strings. Same-type rule preserved (`str +
  str` yields `str`); `-` and `*` on strings stay refused, brands
  and bools compute nothing, and mixed operands never convert.
  Static gate first (`cannot subtract str with str`), loud
  dynamic error past it, native `+` in emit (no helper needed).
- `std/text/text.can` blesses `std__str__concat` with decision
  tables covering empties, multi-scalar text, and
  markup-passthrough (concat preserves everything; escaping
  belongs to context encoders, never to construction).

## Decided: Unicode-scalar indexing

Text operations count Unicode scalar values. Byte operations stay
separate (`std__utf8__encode` / `std__utf8__decode` when they
land). No operation normalizes, trims, or case-folds implicitly;
each such transformation is an explicitly named function over
revision-pinned data. `compare__str` stays byte order until the
collation question gets its own proposal — it predates this
layer and makes no Unicode claim.

## Specified, not shipped: the scalar-access family

`length_scalars`, `scalar_at`, `slice_scalars`, and everything
built on them (`contains`, `find`, `replace_all`, `split`,
`join`, and the case/trim/normalize set) have exact contracts in
the brief but no expression-level surface today: measurement and
indexing are inexpressible with literals, comparisons, and `+`
alone, and strings admit no decomposition in `match`.

The surface is the open decision, not the semantics. Candidates:
a reserved pure-builtin callee kind (precedent: `state__get` /
`state__put` are reserved names with dedicated rules), or new
operator spellings. Either needs its own spec with proof costs
before a single scalar-access function lands — smuggling the
mechanism inside a concat change would violate the one-operation
rule that produced this doc.

## Consequences

- `std/text/` freezes as a golden like the other std modules.
- Issue 3 stays open: concatenation plus decided semantics is
  progress, not resolution. The scalar-access surface decision
  is the next step, and it is now a precise question instead of
  a vague "text is blocked".

## Open decisions (do not block)

- Builtin-callform versus operators for scalar access.
- Collation-aware comparison as a separate proposal.
- Grapheme segmentation and NFC/casefold data pinning.
