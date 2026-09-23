# B1-12 Markdown safe-mode decision audit

Three independently rewritten packets asked three identical questions over
the B1-12 plan, the pinned Bun 1.4.2 Markdown probe ledger (callback
names and payloads, list/table flattening, silent breaks, raw block/span
routing, href rawness, NUL splitting to U+FFFD, throw propagation,
ignored unknowns) and the html.ts factory bounds (async construction,
authorTags/voidTags/src gaps, https-plus-local URL policy). All prose
(state, instructions, criteria) differs across packets; question keys,
option keys, identifiers and measured facts are stable. Verified
mechanically before dispatch (0 identical prose fields of 12 comparisons;
21 fact tokens present in every state). Model: `jev-1.13.0` via
`jev-latest`, three HTTP 200 rounds, 2937 input / 378 output tokens total.

Round 2 first dispatched with a state typo ("captureero"); the run is
preserved as `request-2.json` (overwritten) alongside
`response-2-typo.json`, and the round was re-dispatched clean. The clean
re-run reproduces the same choices, so the round 2 divergence below is
wording sensitivity, not a typo artifact: round 2's state normalizes
unknown-construct degradation ("decays into text"), which pulls toward
flattening.

## Results

| Question | Round 1 | Round 2 (clean) | Round 3 |
|---|---|---|---|
| safe_mode_scope | safe_subset 0.64 (conf 0.47) | full_flattened 0.56 (conf 0.34) | safe_subset 0.78 (conf 0.67) |
| raw_block_policy | escape_paragraph 0.62 (conf 0.43) | reject 0.37 (conf 0.05) | reject 0.51 (conf 0.27) |
| budget_shape | fixed_standalone 0.97 (conf 0.95) | fixed_standalone 1.00 (conf 1.00) | fixed_standalone 1.00 (conf 1.00) |

Requests, responses and SHA metadata: `request-N.json`,
`response-N.json`, `response-N.metadata.json` in this directory.

## Agreement (accepted as advice)

- budget_shape `fixed_standalone` 3/3 at near-certainty: adapter-fixed
  standaloneBytes input/output ceilings, no caller options/limits
  records. `caller_limits` never exceeds 0.03.
- Every round agrees safe mode SHIPS (`blocked` peaks at 0.32): the
  subset the callbacks carry faithfully is worth a typed result.

## Disagreement (investigated, decided below)

- safe_mode_scope splits 2/3 `safe_subset` against round 2's
  `full_flattened` at confidence 0.34. Flattening known lists and tables
  into paragraphs preserves text but drops structure the caller asked
  for with zero signal; the subset instead renders every supported
  construct faithfully (headings, paragraphs, quotes, spans, code,
  links, images) and fails lists, tables and raw blocks loudly. Plan
  selects `safe_subset` with the majority.
- raw_block_policy is weak everywhere (0.62/0.37/0.51, confidences
  0.43/0.05/0.27). Rejecting raw blocks would refuse extremely common
  real-world Markdown (badges, div wrappers, line breaks); escaping the
  block source into a visible paragraph keeps rendering total while
  altering structure visibly rather than silently, and safety still
  holds since every byte passes through the text escaper. Plan selects
  `escape_paragraph` on practicality plus round 1.
