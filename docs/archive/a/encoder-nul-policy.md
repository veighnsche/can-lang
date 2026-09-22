# Row 5 (E) — Encoder NUL policy (standalone, before a28)

Status: shipped. The text encoder rejects NUL with a typed error.
This reopens the a25 contract deliberately, as its own versioned
change — not a quiet tweak, and not folded into a28.

## Policy

- `html.nul_byte(value: str)`: typed rejection of NUL, not blanket
  control rejection. Rationale: NUL is the one scalar this
  serialization path cannot preserve as that scalar (HTML parsing
  ignores or replaces it; `&#0;` becomes U+FFFD, so no
  entity-workaround exists). Other controls pass through per HTML
  text semantics; each would need its own kind and justification.
- Detection is an integer compare (`s[0] == 0`), not a literal
  match: no invisible bytes in the body, code point explicit,
  following the `str_to_int` digit-compare precedent. Test rows
  carry raw NUL bytes (the only spelling available — `.can` has
  no escape syntax), named `nul_first/middle/last` and per-arm
  (`nul_plain/amp/lt/gt`) so every re-raise path executes.
- `orig` is threaded through the worker (str_to_int precedent):
  every rejection carries the original input, at both levels.
  The entry re-raises with its own `raw`, keeping the
  invalid-attribute convention (errors carry rejected input).

## Three guarantees, stated separately

1. **Exact preservation of the serialized string**: holds for
   admitted (NUL-free) inputs, evaluator and emit.
2. **Prevention of markup interpretation**: `&<>` escaping,
   unchanged.
3. **Preservation of parsed text**: NOT claimed. Hence no HTML
   parsing fixture — checks attach where claims are made, and
   none is made here.

## Brand versus encoder (explicit)

The claim is "the encoder rejects NUL," not "every `Html__Text`
is NUL-free." Test and given-row seals can still mint NUL-bearing
brands (checked data, unchanged rule); strengthening the brand
invariant would need literal-content checking in the tycker —
disproportionate and unasked. Promotion identity is intact: node
relabels whatever it receives, NUL included, with no filtering.

## Proof costs

- 8 new rows (5 worker covering all 4 re-raise arms plus the
  top-level arm, 3 entry covering positions); 38 pre-existing
  rows green, behavior otherwise identical.
- Golden and `errors.json` regenerated (both raisers, all hits).
- No compiler change, no new callers (none exist
  cross-module), no TS helpers touched.

## Still scheduled

a28 (attribute values) consumes this settled policy; it owns its
attribute-context questions, not a repair of a25.
