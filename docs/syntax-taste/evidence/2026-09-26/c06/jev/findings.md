# C06 Jev findings — 2026-09-26 (advice, not qualification evidence)

Model: `jev-1.13.0`. Three consultations, 3 choice questions each, 16
rewritten explanatory fields (wording audit `wording-audit.json` ok).
Total usage: 4565 input + 393 output tokens.
Semantic equivalence preserved at draft time; only phrasing varies.

## Agreement (adopted as advice, verified against the contract)

| Question | R1 | R2 | R3 | Adopted |
| --- | --- | --- | --- | --- |
| lowering | phase_value 0.69 | phase_value 0.84 | phase_value 0.77 | distinct `generation_mismatch` fetch outcome lowered to `http::transport_failed` with phase `generation`; runtime-only |
| scope | paired_pin_selective 0.98 | paired_pin_selective 0.98 | paired_pin_selective 0.99 | pin on paired builds only; enforce JSON + form mounts; document mounts exempt |
| htmx_channel | guard_injection 1.0 | guard_injection 1.0 | guard_injection 1.0 | guard `htmx:config:request` hook copies the live slot into `detail.ctx.request.headers` |

R1 lowering was the closest call (new_identity 0.31 vs phase_value
0.69); R2/R3 widen the gap. The phase keeps the refusal inside the
declared bound with no catalogue, checker, or emitter change, following
the E04 timeout/cancelled precedent; the fetch outcome itself stays a
distinct `generation_mismatch` kind so only the byte-exact 409 shape
maps to it (a domain 409 such as `grid_conflict` keeps its path).

Scope and channel agreement is near-unanimous: unpaired servers keep
bit-identical behavior for curl/test/non-browser callers, navigations
(which cannot carry headers and always load the current document) stay
outside enforcement, and the C-E guard mechanism carries the C-H
identity for htmx traffic via guardgen plus the six documented re-pins.

## Disagreement

None. All three consultations agree on all three questions.

## Fixed (not consulted; from the C-H contract)

409 shape `{schemaVersion 1, kind can.generation-mismatch,
serverGeneration 64-hex}`; `data-can-generation` page slot;
`can-generation` header; symmetric rollback refusal; asset routes
unchecked; blocking refresh prompt with no retry and no silent reload.
