# a35 — Typed fragment endpoints (HTMX-shaped exchange, sketch)

Status: sketch (pre-decision). No rule, no code, no golden. This
records the shape a hypermedia-fragment row could take once its
prerequisites land, and what would have to be proven first.

## Observation

`Html__Safe` is already the fragment type: a serialized child
fragment admissible exactly at ordinary child-fragment boundaries,
with no authority for script, style, attribute, or URL contexts.
An HTMX-style endpoint returns exactly such a fragment plus a swap
instruction. The brand discipline maps onto fragment exchange
without importing the `hx-*` attribute soup.

## Sketch (not specified)

- An endpoint is a function returning `Html__Safe` (via the
  existing `Html__Text` → escape → node chain) plus an explicit
  swap descriptor: target identity and placement (replace, append,
  prepend, before, after).
- The descriptor wants a closed union: one spelling per placement,
  which needs the `variant` row first (methods, form states, and
  validation results need it too).
- Transport wants the HTTP row; target identity wants the
  component row. None of the three exists yet.

## Non-goals

- Out-of-band swaps, morphing, trigger headers, and the rest of
  the implicit client/server convention surface. Each is coupling
  the language exists to make explicit; none is admitted without
  its own justification, as with attribute names (a27).
- Any claim that typed fragments are safe in script, style,
  attribute, or URL contexts (see the `Html__Safe` contract).

## Prerequisites, in order

1. Closed tagged unions (`variant`).
2. HTTP row (requests, responses, routing).
3. Component row (target identity).
4. This sketch graduates to a proposal only after 1–3 land.

## Feasibility trigger

Revisit when 1–3 have landed: check whether fragment endpoints
still want language help, or whether typed functions returning
`Html__Safe` plus a small descriptor record already cover the
use case with no new expressive power. Prefer the latter; a
feature whose proof is future work is a bug with a roadmap.
