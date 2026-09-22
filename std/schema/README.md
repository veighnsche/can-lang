# Current static asset approval

Asset approval happens once at build time, not through runtime
witnesses: the manifest loader snapshots declared project assets,
validates closed extension/MIME inventories with per-format
signatures, and serves them at content-addressed immutable URLs
resolved by `asset::url`. Sealed approval brands, policy handles,
revocation walks, SRI ceremonies, and runtime registries are
excluded. Current demonstrations live in the `assets` fixtures and
the admitted applications, each serving its declared `site.css`
through the manifest loader.

The adjacent legacy source and generated files below are historical
migration inputs scheduled for retirement by I43/I44, not the current
implementation.

## Historical implementation

# schema — construction-time asset approval (S1: pure core)

- `schema.can` — `mod schema`: the asset-approval value model and lookup.
  `Schema__AssetRequest` is untrusted identifying data (plain record, no
  authority). `Schema__ApprovedAsset` is the opaque witness, sealed only in
  this file's bodies and only on the exact-match path of
  `schema__asset__approve`. `Schema__AssetPolicy` is the opaque policy
  handle; the snapshot carries the expected handle and approval checks
  agreement by brand equality, never by inspection. The snapshot, site,
  head sequence, and time arrive as explicit values — their provenance is
  the deployment acceptance workflow's job (S4), not this file's.
- Checks, cheapest first (all failures collapse to the one kind below):
  closed role gate (`stylesheet` | `script`), absolute-`https` prefix,
  tail scan (no NUL/C0/DEL, fragment, backslash, whitespace, userinfo,
  or quote/angle characters — the last for the fixed single-quoted sink
  embedding), single-SHA-384 SRI form (`sha384-` + 64 base64 scalars),
  policy-handle agreement, program agreement, head-sequence currency,
  exactly-one entry match, revocation walk, validity interval (upper
  bound exclusive; lower-edge and inner-upper rows pin both edges).
- `error schema.asset_not_approved(asset: Schema__AssetRequest)` is the
  only lookup failure: unknown asset, wrong digest, wrong role/site,
  wrong policy, revoked, expired, stale, and conflicting snapshots all
  produce this same shape carrying the original request unchanged. No
  registered digest, host list, or alternate entry ever leaves the module.
- NUL is rejected, never stripped; the `tail_nul` row carries a literal
  NUL byte (invisible — verify with `tr -d -c '\000' | wc -c`).
- `schema.ts` + `errors.json` — committed golden TS prod emit. Regenerate:
  `go run ./compiler --out std/schema std/schema/schema.can`; verify:
  `go test ./...`.
- Separator safety (S2): the witness joins its eight fields with `|`
  for the projection kernel, so `schema__tokens__check` refuses `|`
  (and NUL) in ids, revisions, and site fields; the URL tail and the
  digest alphabet exclude it too. A field holding the separator would
  make the kernel split ambiguous, so approval refuses it.
- Fixture root (S5): there is no ambient registry to poison — the
  module holds no state cells and declares no effects or `uses`
  (pinned by `TestAssetNoAmbientAuthority`), and no approval-path
  check runs under a `given` table (pinned by
  `TestAssetNoScriptedEvidence`). Fixture snapshots are explicit
  values; a sealed success-shaped value alone never approves, and
  fixture-signed material under production keys stays a future
  `SchemaAuthorityInvalid` hook (see `docs/a/a84-asset-provenance.md`),
  never a committed row.
- Hostile set (pinned by `TestAssetHostileSet`, each with its row):
  unapproved URL, wrong digest, policy mismatch, revocation,
  role swap, expiry, stale sequence, conflicting snapshot,
  non-transitive dependency URL, quote/control-char URL (shape
  rejects even on exact entry match), tampered witness, mixed policy
  context — all rejected as `schema.asset_not_approved` carrying
  the original request unchanged; cross-role spends rejected by
  the builders with the witness preserved. No rejection kind
  carries registry contents, expected digests, or alternate
  entries.

Rules: `docs/a/a83-astra-schema.md` (trust rulings, §§1, 6, 9–11 in this
slice). Plan: the S1 working plan (agent plans removed; see `docs/a/a83-astra-schema.md`).
Provenance design: `docs/a/a84-asset-provenance.md` (S4 paper).
`schema__asset__recheck` (S4) re-runs authorization and requires
witness equality; currency is enforced on the supplied snapshot, not
historically on the witness. Builders (`html__asset__stylesheet`,
`html__asset__script`) and the two-owner bridge grant arrived in
S2–S3; acceptance certificates are future work — until then this
core is test-gated and makes no production trust claim.
