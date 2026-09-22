# Elements roster — pick what you want

How to use: put an `x` inside the brackets for every element you want
built (`- [x] div`). Items marked ★ are my recommended 30 — they add
up exactly. Swap freely; just keep the total near 30 for this slice.
Anything unpicked stays available for a later top-up.

Signature shapes (fixed per family, not per element):

- **void** = attributes only, no children, no closing tag (`<br>`)
- **container** = attributes + children (`<div>…</div>`)
- Families 7–8 are containers too; they're split out because misuse
  (e.g. `title` in body) is a caller bug, not a builder bug.

## Family 1 — Void (spec-fixed list, attrs only)

- [x] ★ `br`
- [x] ★ `hr`
- [x] ★ `img`
- [x] ★ `input`
- [x] ★ `link` (head-only: stylesheets, favicons)
- [x] ★ `meta` (head-only: charset, viewport)
- [ ] `area`
- [ ] `base` (head-only)
- [x] `col`
- [ ] `embed`
- [ ] `source` (for video/audio)
- [ ] `track` (for video/audio)
- [ ] `wbr`

## Family 2 — Block containers (attrs + children)

- [x] ★ `div`
- [x] ★ `p`
- [x] ★ `h1`
- [x] ★ `h2`
- [x] ★ `h3`
- [x] ★ `section`
- [x] ★ `header`
- [x] ★ `footer`
- [x] ★ `nav`
- [x] `main`
- [x] `article`
- [x] `aside`
- [x] `blockquote`
- [x] `pre`
- [ ] `h4`
- [ ] `h5`
- [ ] `h6`
- [ ] `address`
- [ ] `figure`
- [ ] `figcaption`
- [ ] `details`
- [ ] `summary`

## Family 3 — Inline containers (attrs + children)

- [x] ★ `span`
- [x] ★ `a`
- [x] ★ `strong`
- [x] ★ `em`
- [x] ★ `code`
- [x] ★ `small`
- [ ] `abbr`
- [ ] `cite`
- [ ] `q`
- [ ] `time`
- [ ] `mark`

## Family 4 — Lists (attrs + children)

- [x] ★ `ul`
- [x] ★ `ol`
- [x] ★ `li`
- [x] `dl`
- [x] `dt`
- [x] `dd`

## Family 5 — Tables (attrs + children)

- [x] ★ `table`
- [x] ★ `tr`
- [x] ★ `td`
- [x] ★ `th`
- [x] `thead`
- [x] `tbody`
- [x] `tfoot`
- [x] `caption`

## Family 6 — Forms (attrs + children, except `input` which is void above)

- [x] ★ `form`
- [x] ★ `button`
- [x] `label`
- [x] `select`
- [x] `option`
- [x] `textarea` (text child only, still escaped — RCDATA-safe)
- [ ] `fieldset`
- [ ] `legend`

## Family 7 — Media containers (attrs + children)

- [ ] `video`
- [ ] `audio`
- [ ] `picture`

## Family 8 — Head-only containers (attrs + children)

- [x] `title`
- [ ] `noscript`

## Deliberately excluded (not options)

- `script`, `style` — outside `Html__Safe` authority. Not a gap, a scope rule.
- `html`, `head`, `body` — belong to Render (`document`), not Elements.

## Fixed build decisions (change only if you disagree)

- [x] Builders return `Html__Safe`, so nesting works with `join` immediately.
- [x] Tag names are literals sealed per builder — never a parameter.
- [x] Empty attributes render with no stray space (`<div>`, not `<div >`).
- [x] New error kinds only if a family needs one (void misuse is
  impossible by shape, so probably none).
