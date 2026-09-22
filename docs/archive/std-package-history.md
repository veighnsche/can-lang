# std package history (predecessor dispositions)

History only. These eight notes were the per-package `std/<pkg>/README.md`
files: each records the current disposition of a predecessor package whose
sources I43/I44 deleted, plus the historical implementation it replaced.
They were merged here verbatim (relative links repaired) so `std/` holds
only live content. Current dispositions are summarized in
[`std/README.md`](../../std/README.md).

## ascii

### Superseded ASCII implementation

The adjacent `ascii.can` and generated `ascii.ts` belong to the obsolete language
and its historical tests. They are not a current catalogue or compiler fallback;
I43/I44 remove them with their old consumers.

Current numeric text conversion is implemented by the native `text::to_int`,
`text::to_float` and `text::to_bool` catalogue operations, with strict whole-input
grammar checks. See [the current scalar project](../../std/scalars/current/src/main.can).
Named text/Unicode APIs are implemented separately by I24; arbitrary ASCII
kernel/prototype access is not admitted.

### Historical ASCII implementation

#### ascii — named ASCII scalar bounds

- `ascii.can` — `mod ascii`: `std__ascii__HASH`, `PLUS`, `MINUS`,
  `DOT`, `SLASH`, `ZERO`, `NINE`, `COLON`, `QUESTION`, `AT`, `A`,
  `Z`, `BACKTICK`, `SMALL_A`, `SMALL_Z` (lowercase needs marking;
  bare `A`/`Z` are uppercase). Consumers pin
  them like any `uses` entry (`std__ascii__COLON@1`).
  `html__url__scheme_token` compares against these names; the
  emitted program is byte-identical to the magic literals.
  Predicates (`std__ascii__is_digit`, `is_alpha`, `is_alnum`)
  sit beside the constants, returning the scalars
  `Bool__Value` wrapper by pin. Digit and alpha are leaves
  over `and`/`or` of comparisons; alnum composes explicitly
  bound call results (no operand calls), alpha outer.

## division

### Current division catalogue

Use `number::divmod` for native truncation toward zero and
`number::euclidean_divmod` for a nonnegative remainder. Both return the ordinary
`number::division` record and emit `number::zero_divisor` for a zero denominator.
The maintained example and sign/tie assertions are in
[the current exact-amount project](../../std/ratio/current/src/main.can).

Adjacent old sources/generated files are historical test inputs scheduled for
I43/I44 retirement, not a current compiler fallback.

### Historical division implementation

#### division — exact Euclidean integer division

- `division.can` — `mod division`: `std__int__divmod`,
  `std__int__mod`, `std__int__is_multiple`, `std__int__is_even`,
  and `std__int__is_odd`, each with all-sign decision tables plus
  zero cases. Every input satisfies dividend = divisor × quotient
  + remainder with 0 ≤ remainder < abs(divisor); a zero divisor
  fails loud. The remainder is always non-negative, so oddness
  tests the false arm, never `value % 2 == 1`.
- `division.ts` + `errors.json` — committed golden TS prod emit
  (tests stripped; the `$canDivMod` helper emits inline).
  Regenerate: `go run ./compiler --out std/division
  std/division/division.can`; verify: `go test ./...`.

Rules: `/REQUIREMENTS.md`. Program: `docs/archive/a/a17-division.md`.
`gcd`, `lcm`, roots, and primality wait on fuel-pattern
recursion (issue 4 stays open).

## host

### Current finite host catalogue

The host shelf is the closed catalogue, not externs: `clock::wall_millis`,
`clock::monotonic_millis`, and `clock::sleep_millis` for time;
`random::secure_bytes` and `random::uuid_v4` for randomness;
`crypto::sha256` for hashing; `env::required` and `env::optional` for
validated names; `log::write_info` and `log::write_error` for JSON
stderr lines. Sealed brands, `Secret__Value` timing comparison,
ambient authority, the filesystem, subprocesses, and broader
crypto/timezone surfaces are excluded. Current demonstrations live in
the `cli` fixtures and the admitted applications (`clock` stamps in
account-search and dashboard, `io`/`env` in native-ai).

The adjacent legacy source, extern decls, ambient types, and generated
files below are historical migration inputs scheduled for retirement
by I43/I44, not the current implementation.

### Historical implementation

#### host — the host shelf: explicit foreign observations

- `host.can` — `mod host`: `std__clock__wall_now` and
  `std__clock__monotonic_now` over `host__wall_now` /
  `host__mono_now` externs. Time points are millis ints
  (JEV instant_repr millis_int 0.97): Unix-epoch millis for
  wall, unspecified-origin millis for monotonic.
  `std__random__bytes` (CSPRNG, 1MB cap) and `std__hash__digest`
  (sealed `Hash__Profile` brand: sha256, sha512; JEV
  hash_profile brand 0.98) over `host__rand_bytes` /
  `host__hash_digest`. `std__secret__equal` (sealed
  `Secret__Value` brand behind timingSafeEqual; JEV
  secret_repr brand 1.0) and `std__env__read` (sealed
  `Env__Name`; denied is a declared upper bound, v1 hosts no
  policy) over `host__secret_equal` / `host__env_read`.
  `std__log__write` (`Log__Event` level+message; one JSON line on
  stderr, bigint levels as decimal strings) over
  `host__log_write`. `platform.d.ts` carries the tsc ambient
  surface (node:crypto, TextEncoder, process, console).
- `host.externs.ts` — real host implementations (not throwing
  stubs). Same-slice maintenance with the extern decls.
- `host.ts` + `errors.json` — committed golden TS prod emit
  (tests stripped). Regenerate: `go run ./compiler --out std/host
  std/host/host.can`; verify: `go test ./...`.

Rules: `/REQUIREMENTS.md`. Program: `docs/archive/ASTRA_STDLIB.md` §Host.
Consumers pin the wrappers (or externs) via `uses`; see
`docs/archive/sketches/host-clock`.

## json

### Current exact JSON codec

JSON is the exact typed codec, not a value-AST library:
`codec::encode_json<T>` and `codec::decode_json<T>` over compiler
schema descriptors, with `bytes::from_utf8` and `bytes::to_utf8` at
the text boundary. Exact numbers, duplicate rejection, and shared
finite budgets are enforced natively. Fuel parsers, `Json__Value`
trees, monomorphic schema families, and standalone hex/base64
codecs are excluded. Current demonstrations live in the `codec`
fixtures and the admitted applications (native-ai encodes its typed
report; account-search decodes forms at the server boundary).

The adjacent legacy source and generated files below are historical
migration inputs scheduled for retirement by I43/I44, not the current
implementation.

### Historical implementation

#### json — JSON value AST, render, scalar codecs, schemas

- `json.can` — `mod json`: `Json__Value` is the parsed tree
  (`Null`/`Bool`/`Num`/`Str`/`Arr`/`Obj`; `Num` carries the wire
  text verbatim, never a lossy float). `std__json__render_value`
  prints any tree through a single self-recursive frame machine:
  only direct self-recursion is admitted, so the natural
  value/fields/array mutual recursion is inexpressible and an
  explicit `Seq<Json__Frame>` stack carries the pending work
  (tag-dispatched records — variant sequences are not admitted;
  stack top is the back since sequence concatenation is not in
  v1). Each step burns one fuel from `std__json__RENDER_FUEL`
  (1M); exhaustion raises payloadless
  `json.render_budget_exhausted`. `std__json__escape` handles
  `"`/`\`/TAB/LF/CR plus `\u00XX` for other C0 controls via an
  armless hex table. Typed codecs bridge scalars to `Json__Doc`
  (`int`/`str`/`bool`/`dec` × encode/decode); decodes reject
  mistyped trees with `json.schema_mismatch` and bad numeric
  text with `json.numeric_out_of_range` (translated from
  `convert.*`). Schemas are a monomorphic family (JEV 0.74 over
  generic-uniform): `Json__IntSchema` & co. carry exact
  encode/decode callbacks invoked through bare-name apply
  wrappers (`invoke` heads must be bare names).
- `json.ts` + `errors.json` — committed golden TS prod emit
  (tests stripped). Regenerate: `go run ./compiler --out
  /tmp/jg std/json/json.can std/scalars/scalars.can`, copy
  `json.ts` + `errors.json`; verify: `go test ./...`.

Rules: `/REQUIREMENTS.md`. Program: `docs/archive/ASTRA_STDLIB.md` §1.8.
Byte-level parse landed in S11b: `std__json__parse_value`
guards empty input, then `std__json__parse_step` runs a
single self-recursive 12-state machine over `Seq<Json__PFrame>`
(empty dispatch + `ArrFirst`/`ArrVal`/`ArrNext` + `ObjFirst`/
`ObjKey`/`ObjColon`/`ObjKeyVal`/`ObjNext` + `StrKey`/`StrVal`/
`NumAcc` + `Tail`). Value states push nested frames; key and
punctuation continuations replace the top frame (a lingering
`ObjFirst` under `StrKey` breaks `parse_attach`, found
in-slice). `StrKey` inherits the parent's fields/keys for
duplicate detection; numbers accumulate raw and validate
through the 9-state numcheck; `Tail` rejects trailing values.
Text drivers landed in S12: per-scalar `encode_text`
(value → schema Doc → render) and `decode_text`
(parse → schema Doc decode) for `int`/`str`/`bool`/`dec`,
reusing the S9 `given` exchanges by test name (`frac`/`exp`
rows added for the int/dec out-of-range paths).

## quota

### Current validation pattern

Validation is ordinary Can, not a validator catalogue: named domain
functions over records, errors, match, and arrays. The maintained
demonstration is the admitted form-validation application, which
validates repeated/optional form values and renders 422 feedback
through safe server-side constructors. Monomorphic
`std__validate__*` helpers, constraint descriptors, envelope
witnesses, and pinned converter versions are excluded; each domain
owns its checks and messages.

The adjacent legacy source and generated files below are historical
migration inputs scheduled for retirement by I43/I44, not the current
implementation.

### Historical implementation

#### quota-counter — validation plus a bounded counter

- `quota.can` — `mod quota`: monomorphic scalar validators
  (`std__validate__require`, `std__validate__int_range`,
  `std__validate__int_nonnegative`, `std__validate__str_nonempty`,
  `std__validate__exclusive_pair`, `std__validate__str_one_of`
  plus its `_from` worker) plus a quota counter
  (`quota__consume`, `quota__usage`) and the S1a request pilot
  (`quota__request__validate`, `quota__request__admit` over
  `Quota__Request` / `Quota__RequestSchema`, plus the S1b
  envelope witness `quota__envelope__validate`): the module pins
  `std__convert__int_to_str@1` for canonical integer rendering
  in violation payloads, with per-site `given` scripts (a18
  verifies the scripted renders against the real converter).
  S2a adds the closed check layer (`std__validate__length_check`,
  `std__validate__range_check`, `std__validate__membership_check`
  over `Validate__Length` / `Validate__Range` /
  `Validate__Membership`): one constraint in, frozen
  `validation.schema_violation` triple out, so consumers bind
  checks to fields without rewriting reconstruction arms
  (see `docs/archive/a/a91-schema-s2-design.md`).
  (`quota__consume`, `quota__usage`) that reuses them through
  same-file local calls, so every call executes its body.
  Validators return the accepted value or a producer-owned typed
  error; bounds are inclusive and reversed bounds fail instead of
  being silently swapped. Every test starts from init.
- `quota.ts` + `errors.json` — committed golden TS prod emit
  (tests/given stripped; the cell is a module-scope `let`).
  Regenerate: `go run ./compiler --out std/quota
  std/quota/quota.can std/scalars/scalars.can`, then delete the
  co-emitted `std/quota/scalars.ts` (quota keeps only its own
  goldens); verify: `go test ./...`.
- Known packaging gap (S1a): `quota.ts` imports `./scalars`,
  which resolves in whole-program compiles but dangles beside
  the committed goldens — and the specifier is extensionless,
  so node ESM needs a harness rewrite to `./scalars.ts` (see
  `rewriteSpecifier` in the parity tests). Cross-dir TS imports
  need their own emit slice; see the a89 implementation
  amendment.

Rules: `/REQUIREMENTS.md`. Program: `docs/archive/a/a13-stdlib.md` (row 1).

## schema

### Current static asset approval

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

### Historical implementation

#### schema — construction-time asset approval (S1: pure core)

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
  `SchemaAuthorityInvalid` hook (see `docs/archive/a/a84-asset-provenance.md`),
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

Rules: `docs/archive/a/a83-astra-schema.md` (trust rulings, §§1, 6, 9–11 in this
slice). Plan: the S1 working plan (agent plans removed; see `docs/archive/a/a83-astra-schema.md`).
Provenance design: `docs/archive/a/a84-asset-provenance.md` (S4 paper).
`schema__asset__recheck` (S4) re-runs authorization and requires
witness equality; currency is enforced on the supplied snapshot, not
historically on the witness. Builders (`html__asset__stylesheet`,
`html__asset__script`) and the two-owner bridge grant arrived in
S2–S3; acceptance certificates are future work — until then this
core is test-gated and makes no production trust claim.

## seq

### Current array catalogue

Ordered sequences are the native array catalogue: literals,
copy-update, field/index/slice/length, `.map`, `.filter`,
`.for_each`, `.fold`, `.find`, `.some`, `.every`, `.sort_by`,
`.slice`, `.concat`, `.to_reversed`, and prelude `append`, all with
native copies and sequential failure-stop traversal. `Seq__Values`
wrappers, `Seq__Item` elements, comparator callbacks beyond
`sort_by`, and group/zip/dedup helpers are excluded. Current
demonstrations live in the `arrays` fixtures and throughout the
admitted applications.

The adjacent legacy source and generated files below are historical
migration inputs scheduled for retirement by I43/I44, not the current
implementation.

### Historical implementation

#### seq — immutable ordered sequences

- `seq.can` — `mod seq`: `std__seq__empty`, `std__seq__singleton`,
  `std__seq__length`, `std__seq__get`, `std__seq__append`,
  `std__seq__concat`, and `std__seq__slice`, each generic over the
  element type with int + str decision tables. Bare-Seq returns are
  rejected by the language, so sequences travel in `Seq__Values<T>`
  and elements in `Seq__Item<T>`. Slice is half-open, same rule as
  text slicing; invalid bounds never clamp. Higher-order traversal
  (`std__seq__map`, `std__seq__filter`, `std__seq__fold`,
  `std__seq__all`, `std__seq__any`, `std__seq__find`) takes total
  callbacks over data-only heads; workers visit left to right, and
  `all`/`any`/`find` stop at the first decisive element. `find`
  reports absence as `sequence.not_found()` (no `Option<T>`:
  generic variants do not exist, so the text.find error shape is
  used instead). `std__seq__sort` takes an explicit `Seq__Order`
  value (`Asc`/`Desc`; lexicographic deferred) over the
  per-instance built-in order — insertion sort, stable by
  construction since equals never reorder. `std__seq__unique`
  keeps first occurrences via per-instance `==`, the same
  contract maps and sets share.
- `seq.ts` + `errors.json` — committed golden TS prod emit
  (tests stripped). Regenerate: `go run ./compiler --out std/seq
  std/seq/seq.can`; verify: `go test ./...`.

Rules: `/REQUIREMENTS.md`. Program: `docs/archive/ASTRA_STDLIB.md` §1.8.

## set

### Current immutable native set catalogue

The maintained `collections` catalogue uses private native Map/Set storage with
immutable opaque values, scalar keys (`int`, `bool`, `str`) and insertion order.
The shared current source project for both maps and sets is
[`std/map/current`](../../std/map/current). Run it with the staged release's
`canlc assert std/map/current` and `canlc run std/map/current`.

All twelve operations are compiler-checked native intrinsics. Map insertion
rejects duplicate keys; get, replace and remove reject absent keys. Replacement
keeps position and copies preserve aliases. Set union appends unseen right keys;
intersection and difference preserve left order. Intersection uses native left
array filtering and right membership to repair the native smaller-set ordering.

The adjacent legacy source and generated files below are historical migration
inputs scheduled for retirement by I43/I44, not the current implementation.

### Historical implementation

#### set — sets with explicit ordering

- `set.can` — `mod set`: `std__set__contains`,
  `std__set__union`, `std__set__intersection`, and
  `std__set__difference` over `Set__Members<T>`. Members keep
  first-occurrence order; union appends only absent members;
  intersection and difference preserve the left order. Equality
  is per-instance `==`, shared with maps.
- `set.ts` + `errors.json` — committed golden TS prod emit
  (tests stripped). Regenerate: `go run ./compiler --out std/set
  std/set/set.can`; verify: `go test ./...`.

Rules: `/REQUIREMENTS.md`. Program: `docs/archive/ASTRA_STDLIB.md` §1.8.
