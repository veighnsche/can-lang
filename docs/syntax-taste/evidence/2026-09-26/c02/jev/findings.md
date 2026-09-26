# C02 Jev findings — 2026-09-26 (advice, not qualification evidence)

Model: `jev-1.13.0`. Three consultations, 4 choice questions each, 21
rewritten explanatory fields (wording audit `jev-wording-audit.json` ok).
Total usage: 4812 input + 515 output tokens (probe excluded).
Semantic equivalence preserved at draft time; only phrasing varies.

## Agreement (adopted as advice, verified against the contract)

| Question | R1 | R2 | R3 | Adopted |
| --- | --- | --- | --- | --- |
| caret_projection | symmetric_calls 0.78 | symmetric_calls 0.54 | symmetric_calls 0.54 | snapshot `selection` record + `set_selection` + `read_selection` |
| file_projection | metadata_records 0.93 | metadata_records 0.64 | metadata_records 0.86 | `files: browser::file[]` {name, size, mime}, 128-entry cap + `read_files` |
| modifier_shape | nested_record 0.57 | nested_record 0.92 | nested_record 0.58 | `modifiers: browser::modifiers` {alt, ctrl, meta, shift} |

R1 modifier was the closest agreed call (flat_bools 0.39 vs nested 0.57);
R2's 0.92 for nested and the R3 repeat carry it. Nested keeps the event
record additive with one field per group instead of seven flat fields.

## Disagreement (investigated, decided on contract grounds)

composition_signal split three ways at low confidence: new_kinds 0.56
(conf 0.34), flag_only 0.58 (conf 0.37), flag_plus_end 0.40 (conf 0.10).
No wording-independent pull exists; each consultation's winner tracks its
own most-capable-sounding option text.

Decision: **flag_only** — a `composing: bool` (native `isComposing`) on
input/key snapshots, no new event kinds. Grounds, all outside the model:

- The C02 definition of done requires minimal sufficient additions; the
  IME workflow (defer normalization while composing) needs only the flag.
- The additive snapshot contract lets composition kinds land later if
  C06/W1 evidence shows the flag insufficient; nothing is foreclosed.
- A low-confidence three-way split is not advice for the largest surface.

## Fixed (not consulted; from the R02 contract and workflows)

`checked: bool` and `selected: str[]` snapshot fields; `set_value`,
`set_checked`, `set_selected` live setters; `read_value`, `read_checked`,
`read_selected`, `read_files` live reads. Direction on `set_selection`
takes forward/backward/none with literal admission in the checker.
