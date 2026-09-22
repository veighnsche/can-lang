# Historical HTML brands (retired)

This describes the superseded compiler and is not the current HTML contract.
Its source/tests await the repository-wide I43/I44 removal gates.

# html — constructor-controlled brands, starting with text

- `html.can` — `mod html`: `brand Html__Text`, `html__text__escape`
  plus its `_from` worker. The worker scans scalar by scalar
  (front-consumption shape, `upper_ascii` precedent), emitting
  `&amp;`, `&lt;`, `&gt;` and passing everything else through,
  astral plane included. Raw strings enter `Html__Text` only
  through the entry; the brand erases to string and flows
  opaquely (never inspected, only passed to brand-typed params).
  NUL is rejected with `html.nul_byte` — the one scalar this
  serialization path cannot preserve as that scalar. All other
  controls pass through per HTML text semantics (each would need
  its own kind and justification to reject); quotes stay
  unescaped here by the text-context contract. See
  `docs/encoder-nul-policy.md`.
- `html__text__node` promotes `Html__Text` to the second brand,
  `Html__Safe is str rev 1 seals_from [Html__Text]` — a serialized
  fragment for ordinary child-fragment boundaries, with no authority
  for script, style, attribute, or URL contexts. Promotion is
  relabeling (`erase(node(t)) = erase(t)`): no double escape, no
  normalization. One-way, exact, same-module, non-transitive; see
  `docs/a/a26-html-node.md`.
- `html__attribute__name` gates `Html__TextAttributeName`: a small
  exact lowercase allowlist (currently `title`), each member with
  its own justification; everything else is `invalid_attribute_name`.
  See `docs/a/a27-attribute-name.md`.
- `html__attribute__text` composes a brand-typed name with an
  encoded value into `Html__Attribute`, canonical single-quoted
  form (`title='...'`). The spelling is reconstructed from the
  closed admitted domain, not extracted from the brand. See
  `docs/a/a29-attribute-value.md`.
- `html__attribute__boolean_name` gates
  `Html__BooleanAttributeName` (currently `disabled`,
  `readonly`, `required`, `checked`);
  `html__attribute__boolean` serializes presence as the spelling
  and absence as the empty contribution. Syntactic guarantee
  only — never inertness, never element applicability. See
  `docs/a/a30-boolean-attribute.md`.
- `html__attribute__id` validates an identifier (nonempty, no
  ASCII whitespace) and serializes it as `id='...'`, reusing the
  shared value worker. Uniqueness needs a tree and stays out.
  New error `html.invalid_identifier`. See
  `docs/a/a31-identifier-attribute.md`.
- `html__attribute__href` / `html__attribute__src` validate
  absolute-`https` URLs under a restricted ASCII authority
  profile and serialize them reusing the shared worker. Fused
  raw input: no `Html__Url` brand exists yet because no
  consumer could serialize one. New errors `html.invalid_url`,
  `html.disallowed_scheme`. See `docs/a/a32-url-attributes.md`.
- `html__fragment__empty` seals `""` as the zero `Html__Safe`
  fragment. See `docs/a/a33-empty-fragment.md`.
- `html__fragment__join`/`_from` walk explicit `Html__Children`
  (plain record over `Seq<Html__Safe>`) with same-brand `+`
  assembly, seeded from `fragment__empty`: composition has the
  empty fragment as identity. Order, empties, and spacing
  preserved exactly. See `docs/a/a42-fragment-join.md`.
- `Html__NamedAttribute` pairs a minted attribute with its name
  by construction; `named_text`/`named_boolean`/`named_id`/
  `named_href`/`named_src` relay the five makers (errors
  forwarded unchanged). See `docs/a/a43-named-attributes.md`.
- `html__asset__stylesheet` is the first pinned bridge sink (S2 slice
  plan): `(asset: Schema__ApprovedAsset, policy: Schema__AssetPolicy)`
  → `Html__SafeResult`, authorized by `asset_bridge ... from schema
  via html__asset__stylesheet@1 for stylesheet`. The restricted
  projection kernel discloses the approved url/digest/role; the fixed
  `link` element carries them with `crossorigin='anonymous'`. Any
  other role fails closed as `html.asset_stylesheet_rejected` with the
  witness preserved. The grant defends callers (unapproved assets
  cannot flow in), not the sink owner: shape + rows + golden pin the
  flow. See `std/schema/README.md`, `docs/a/a83-astra-schema.md`.
- `html__asset__script` is the second pinned sink (S3 slice plan),
  classic scripts only: `asset_bridge ... for script`, fixed `script`
  element with `src` + integrity + `crossorigin='anonymous'`. Witnesses
  approved as `stylesheet` fail as `html.asset_script_rejected`, and
  `module`/`worker`/`preload` spends have no builder (a witness row
  per mode pins the rejection). Approving an entry never approves
  its dependencies: each URL needs its own entry (schema row
  `approve_no_transitive`).
- `html.ts` + `errors.json` — committed golden TS prod emit
  (tests stripped; `errors.json` is the error registry).
  Regenerate: `go run ./compiler --out std/html
  std/html/html.can`; verify: `go test ./...`.

Rules: `/REQUIREMENTS.md`. Program: `docs/a/a25-html-text.md`,
`docs/a/a26-html-node.md`, `docs/a/a27-attribute-name.md`,
`docs/a/a29-attribute-value.md`, `docs/a/a30-boolean-attribute.md`,
`docs/a/a31-identifier-attribute.md`, `docs/a/a32-url-attributes.md`,
`docs/a/a33-empty-fragment.md`, `docs/a/a34-boolean-names.md`,
`docs/encoder-nul-policy.md`, brand scope: `docs/a/a15-brands.md`.
