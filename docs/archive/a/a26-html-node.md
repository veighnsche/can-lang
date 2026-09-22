# a26 — Explicitly authorized promotion (html__text__node via seals_from)

Status: shipped. Row 5 continues with the second brand and the first
brand-to-brand operation. The mechanism is D from the second-opinion
review: owner-local, explicitly authorized one-way promotion — not
blanket same-file brand-to-brand sealing.

## Rule

- `brand Html__Safe is str rev 1 seals_from [Html__Text]`, declared in
  `std/html/`. The clause is same-line metadata: the destination names
  its admitted sources. Empty means str-only minting (the a25 rule,
  unchanged).
- `html__text__node(text)` → `Html__SafeResult(safe)` via one promotion
  seal: `Ok(safe = seal Html__Safe(text))`. No worker, no recursion,
  no kernel changes.
- A seal whose argument is a brand requires all four, checked
  mechanically: (1) exact authorization — the source appears in the
  destination's `seals_from`; (2) ownership — source, destination, and
  site in one module (CAN6004 otherwise); (3) representation — v0
  brands are str-backed only, so agreement is structural, not a new
  check; (4) identity — evaluation and emission preserve the string
  exactly (erasure; no eval or emit change was needed).
- No reverse, transitive, or inferred promotion. `Secret → Html__Safe`
  and `Html__Safe → Html__Text` stay rejected unless someone edits the
  authorization — which the declaration diff then shows.

## The missing premise, now stated

`Html__Safe` represents a serialized fragment admissible at the
library's supported ordinary HTML child-fragment boundaries. It grants
no authority for insertion into script, style, attribute, or URL
contexts. Authority, representation preservation, and semantic validity
are three different obligations: file ownership covers the first,
erasure the second, and this definition plus the a25 encoder argument
the third. Without the definition, promotion would be relabeling bytes
and calling it safety.

## Why the identity table is the proof

`erase(node(t).safe) = erase(t)` for every admitted value — notation,
not a language operation. The sharp rows: `a&amp;b → a&amp;b` (no
double escape), `&amp;lt; → &amp;lt;` (entity-looking text untouched),
`&lt;script&gt;` passthrough (encoding, not sanitizing), empty and
astral identity. Six rows run in the evaluator at compile time; the
emitted `return { $can_kind: "ok", safe: text }` is the emit-side
identity, frozen in the golden.

## Corrections to earlier notes

- a25 said a declaration gate rejects brand returns. The probe showed
  the real shape: the tycker silently skips Ok payloads when the
  return is not a record, and only the emitter refuses
  (`returns unknown type`). a26 adds the source repair: a function
  returning a brand is now a static error
  (`bare-brand returns are unsupported, return a record`), with a
  negative test. Zero shipped functions return bare brands, so the
  repair changes nothing in std. The emitter refusal stays as
  defense in depth.
- The a26 frame understated B and overstated against C. B (structural
  nodes to a trusted serializer) is a coherent alternative
  architecture, not merely deferred work — it is declined because the
  catalogue fixes branded serialization, not because the wall is a
  theorem. C's real cost is field correlation (nothing stops
  assembling one result's text with another's safe field), not
  coupling of all future constructors.
- The host-boundary extern declassifier is named and declined: moving
  a pure identity into trusted host code trades a mechanical rule for
  scripted evidence that the host preserved the string.

## Implementation

- `compiler/parse.go`: `BrandDecl.SealsFrom`, extended brand regex.
- `compiler/types.go`: `brandSeals` map, promotion branch in the seal
  rule (CAN6003 unlisted, CAN6004 cross-module), `checkBrandDecl`
  validates sources (CAN6002 unknown, CAN6004 foreign), brand-return
  rejection in `checkTypes`.
- `compiler/lsp.go`: pass `prog` to `checkBrandDecl`.
- `compiler/seal_test.go`: authorized, unauthorized, reverse,
  transitive, unknown-source, cross-file (decl + site), brand-return.
- `std/html/`: brand, result type, node plus six table rows; golden
  regenerated, `errors.json` empty (total module).
- No eval or emit change: erasure already gives identity, and the
  tables plus golden prove it. One TextMate-grammar token:
  `seals_from` joins the `keyword.control.can` alternation with a
  gramcheck sample, so the new declaration keyword highlights.

## Still scheduled (not silently dropped)

Attribute and URL constructors, fragments, elements — each its own
slice, in program order. Each new promotion will name its source in
`seals_from` or be refused; the audit stays `grep seal`.
