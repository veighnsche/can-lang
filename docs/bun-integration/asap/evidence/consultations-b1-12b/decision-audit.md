# B1-12 Markdown safe-renderer decision audit (round B, reframed gate)

Supersedes `consultations-b1-12/`, whose premise (callbacks flatten
lists/tables) was overturned by corrected probes: the pinned Bun 1.4.2
`RenderCallbacks` use camelCase names (`listItem`, `thead`, `tbody`,
`tr`, `th`, `td`), and with correct names the feed is structurally
complete — `listItem` carries `{index,depth,ordered,start?,checked?}`,
`th`/`td` carry `align`, nesting depth and ordered start survive. Three
independently rewritten packets asked three identical questions over the
corrected ledger. All prose (state, instructions, criteria) differs
across packets; question keys, option keys, identifiers and measured
facts are stable. Verified mechanically before dispatch (0 identical
prose fields of 39 comparisons; 22 fact tokens present in every state).
Model: `jev-1.13.0` via `jev-latest`, three HTTP 200 rounds, 3475 input
/ 407 output tokens total.

## Results

| Question | Round 1 | Round 2 | Round 3 |
|---|---|---|---|
| break_policy | dual_mode 0.64 (conf 0.46) | normalize_soft 0.69 (conf 0.54) | dual_mode 0.78 (conf 0.67) |
| link_scope | strict_https_local 0.96 (conf 0.94) | strict_https_local 0.99 (conf 0.99) | strict_https_local 1.00 (conf 1.00) |
| table_list_fidelity | drop_details 0.56 (conf 0.34) | drop_details 0.91 (conf 0.86) | full_fidelity 0.51 (conf 0.27) |

Requests, responses and SHA metadata: `request-N.json`,
`response-N.json`, `response-N.metadata.json` in this directory.

## Agreement (accepted as advice)

- link_scope `strict_https_local` 3/3 at near-certainty: https plus
  site-local `/` hrefs only; autolinks off (Bun mints `http://` and
  `mailto:` the gate rejects); heading ids on, self-links off.
  Identifier mapping: the packets proposed `markdown::unsupported_url`,
  but the catalogue contract emits `html::invalid_url` and the adapter
  propagates the `html.parseURL` failure unchanged, so no new markdown
  error is introduced.
  `allow_all_native` polls 0.00 in every round: unanimous against
  loosening the URL policy to match native output.
- Loud refusal is dead everywhere: `reject_hard` peaks at 0.05,
  `subset_reject` peaks at 0.03. Rendering stays total; degradation
  must be visible, never a refusal of common documents.

## Disagreement (investigated, decided below)

- break_policy splits 2/3 `dual_mode` against round 2's
  `normalize_soft`. Dual mode (run `html()` alongside `render()` and
  splice `<br />` positions back) hides an alignment risk the packets
  did not state: matching `<br />` sites in an HTML string to
  callback-tree positions needs a parallel offset model over parser
  output, and any quirk (breaks in headings, code spans, tables)
  misaligns silently — the exact failure class the safe renderer
  exists to prevent — while doubling parse cost and trusted surface.
  The `<br />` loss from `normalize_soft` is minor, visible, and
  predictable. Plan selects `normalize_soft` on silent-misalignment
  risk plus round 2.
- table_list_fidelity splits 2/3 `drop_details` against round 3's
  `full_fidelity` (conf 0.27, weakest answer of the run). The packets
  framed the gap as "the gate cannot express today", biasing toward
  not touching `html.ts`, but omitted a decisive codebase fact:
  `html.ts` already validates analogous presentational attributes
  (`colspan`/`rowspan` integers, `scope` enum, `type` on `ol`) through
  the same `applicability` + `validValue` pattern the extension needs,
  so `align` (fixed `left|center|right` triple) on `th`/`td` and
  integer `start` on `ol` add ~nil attack surface while removing a
  permanent silent precision loss (right-aligned numeric columns carry
  meaning). Task checkboxes need no gate change (`input`/`checkbox`/
  `checked`/`disabled` already allowed). Round 3, which framed the
  gate most concretely, independently leaned this way. Plan selects
  `full_fidelity` on fit-with-existing-pattern plus round 3,
  overruling the 2/3 wording-driven majority as advice, not proof.

## Carried constraints (not re-asked)

- Fixed adapter `standaloneBytes` ceilings, no caller limit options
  (prior round, 3/3 near-certainty).
- `wikiLinks` off (render silently drops targets); `underline` and
  `latexMath` off (dead switches in this build); raw markup degrades
  to visible escaped text via `noHtmlBlocks`/`noHtmlSpans` (parser
  demotes tags to text callbacks — zero custom code, strengthening the
  prior weak `escape_paragraph` selection).
