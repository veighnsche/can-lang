# B1-12 Markdown contract confirmation

Renderer: Bun 1.4.2 `Bun.markdown.html` and `Bun.markdown.render`.
Every row below is an executed observation on the pinned binary; see
[markdown-native-probe.ts](../../../../bun-integration/asap/evidence/markdown-native-probe.ts)
and the [B1-12b decision audit](../../../../bun-integration/asap/evidence/consultations-b1-12b/decision-audit.md).
Two operations (`markdown::render_text_html`, `markdown::render_safe`)
share fixed adapter budgets; safe output is an opaque `html::safe`
rebuilt through the `html.ts` factory, never branded native output.

## Callback feed

| Aspect | Contract |
|---|---|
| Names | camelCase (`listItem`, `thead`, `tbody`, `tr`, `th`, `td`); lowercase probes miss them |
| `listItem` meta | `{index, depth, ordered, start?, checked?}`; checked present exactly for task items |
| `th`/`td` meta | `{}` unaligned, `{align}` with `left`, `center` or `right` |
| Nesting | depth 0/1/2 verified; ordered start survives (`start="5"`) |
| Missing body | header-only tables fire no `tbody` callback; the adapter synthesizes the empty body the gate requires |
| Code meta | `{language}` fenced (case preserved, first info word), absent indented; children keep a trailing newline |
| Heading meta | `{level, id}` with ids on; slugs lowercase and dedupe (`dup`, `dup-1`) |
| Unregistered elements | children pass through unchanged (visible degradation, text preserved) |
| Links | never nest natively; images may nest in links; image alts arrive as text |

## Safe-mode policy

| Aspect | Contract |
|---|---|
| Links | https plus site-local `/` only; autolinks off; `http:`, `mailto:`, `#fragment`, empty and relative hrefs fail as `html::invalid_url` |
| Raw HTML | `noHtmlBlocks`/`noHtmlSpans` demote blocks and spans (including `<url>` autolinks) to escaped text; `tagFilter` alone only rewrites `<` |
| Breaks | soft and hard both arrive as newlines; safe output normalizes to soft (no `<br />`) |
| Disabled flags | `wikiLinks` off (render silently drops targets); `underline`/`latexMath` ignored by this build |
| Code classes | info strings outside `[A-Za-z0-9_-]` lose their `language-*` class |
| Tables/lists | full fidelity via narrow gate additions (`align` triple, integer `start`); task bullets synthesize disabled checkbox inputs |
| Budgets | input/output bytes at standaloneBytes, node count at maxNodes, all as `markdown::over_limit` |

## String mode

| Aspect | Contract |
|---|---|
| `render_text_html` | default native options; raw HTML, `javascript:` hrefs and `<url>` autolinks pass through into an ordinary `str` that the type system keeps out of `html::safe` and trusted responses |
