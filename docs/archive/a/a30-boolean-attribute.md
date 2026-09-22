# a30 — Boolean attribute gate plus presence/absence composition

Status: shipped. First boolean-attribute slice: the gate minting
`Html__BooleanAttributeName` (singleton `disabled`) and
`html__attribute__boolean` composing it with a presence flag into
`Html__Attribute`. Constructor milestone, not a rendering
workflow: no element system exists to consume it yet. The
motivating sketch is a conditionally disabled submit button —
presence or omission of one approved attribute, which the
text-attribute constructor cannot express.

## Rule

- `html__attribute__boolean_name(value)` →
  `Html__BooleanNameResult(name)` or
  `html.invalid_attribute_name(value)`. Exact-match gate,
  singleton `disabled`, same shape as the a27 gate. Rejected
  rows pin current non-members (`required`, `readonly`,
  `checked`, case and padding variants); admitting any of them
  is a new slice with its own justification.
- `html__attribute__boolean(name, present)` →
  `Html__AttributeResult`, total (`emits []`): `true` seals the
  reconstructed spelling, `false` seals the empty contribution.
- `Html__Attribute` now holds zero or one complete approved
  serialization with no surrounding separator — amending the
  a29 exactly-one reading. The future attributes-maker must
  omit empty contributions, separate only retained attributes,
  and judge duplicates among retained names. Emptiness is
  absence here, never removal elsewhere, and never a promise
  the control is enabled (an enclosing fieldset can still
  disable). If any existing consumer assumes exactly one
  attribute: none exists; the brand has no other producer or
  consumer yet.

## The guarantee, stated honestly

`disabled` is not inert and this slice never claims it is. It
excludes its control from constraint validation and form data,
affects fieldset descendants (first-legend exception), and has
stylesheet-link behavior. The constructor guarantees an
approved serialization and nothing more. Element applicability
belongs to future element constructors; intended behavior
belongs to form logic. `disabled` + `required` interaction
semantics (a disabled control skips validation) belong to the
consuming form contract, not to either attribute alone.

## The corrected impossibility claim

The slice frame asserted that a second admitted member forces
fused raw-name input, citing a probe where a two-comparison
chain's terminal arm failed the build (`no test takes on
false`). The outside review broke the generalization: N
members need N-1 comparisons plus a final-member literal, and
that shape compiles green (verified by probe after the
review). The failed shape proved only that a defensive
fallback arm is unbuildable — an invariant cannot make an
untaken arm count as taken, but it can justify a reachable
final branch. The catalogue's brand-typed signature therefore
survives at any N, under the same correspondence invariant the
singleton already carries. No family-wide fused-input amendment
is on the table.

## Inputs

- Jev returned 0.87 for singleton-first. It judged the scope
  choice, not the (since corrected) forcing argument.
- The outside review supplied the N-1 correction, the
  non-inertness facts, the absence-as-empty-contribution
  definition with its join law, and the consuming-sketch
  requirement — all adopted above.

## Proof costs

- New shapes proved in /tmp before use: same-brand `==`
  (param-param and param-sealed-literal, both green);
  N-1-chain green, N-comparison-plus-fallback red. No new
  machinery in the shipped code.
- 11 new decision-table rows, 76/76 green with the
  pre-existing 65: 8 gate rows, 1 spelling round-trip row,
  2 composer rows.
- Independent oracle: the generated `html.ts` runs under node
  (present, absent-is-empty, gate admit/reject, end-to-end
  through the gate's mint). Scratch probe at
  `/tmp/bool-probe.mjs`, not committed.
- `errors.json` regenerated; `go test -count=1 ./...`,
  modcheck, and gramcheck all green fresh.

## Still scheduled (not silently dropped)

- Further boolean names (`required`, `readonly`, `checked`,
  …) each need a slice with per-name justification; the
  mechanism (N-1 chain) is ready, the membership is not
  pre-approved.
- The `id` constructor (Text + Brands) is the next attribute
  in catalogue order; `href`/`src` wait on the URL brand and
  its policy; `classes` and the attributes-maker wait on
  Collections.
