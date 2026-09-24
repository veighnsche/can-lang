# UP01 internal interface records

Named handoffs every worker implements against. They constrain
implementation details of the selected behavior only; no row adds public
semantics beyond
`docs/syntax-taste/post-upgrade-selected-behavior-2026-09-24.md`. Suppliers
record concrete identities (file, symbol, hash) in their handoff; consumers
use exactly the names below.

## I-1 canonical action metadata (supplier: UP05; consumers: UP08/09/10/16)

Key: canonical locked package identity + declaration name. Fields:

- `method`: `get` | `post` (selected invoice set; grammar admits the declared verb).
- `path`: literal segments plus `:name` captures; each capture is one strict
  decoded segment bound to a required `str` or canonical int64 `int` field of
  the capture record.
- `captures`: record type name (`invoice_key`); field names/types.
- `input`: `none` | `json <record> limit <bytes>` | `form <record> limit <bytes> rows_limit <n>`.
- `returns`: result variant name (`grid_load_outcome`, `grid_edit_outcome`, `edit_outcome`).
- `body`: `json` | `html`.
- `cases`: ordered (leaf, status, swap?) rows; HTML rows carry `swap inner`.
- `identity`: declaration source span + locked package instance id.

Current-state note: no `mount`/`action::url` support exists in
`check/actions.go` / `emit/actions.go` at baseline; UP05 authors the
declaration side, UP08 the consumer/lowering side.

## I-2 mount callback arities (supplier: UP08; consumers: UP09/19/20)

- JSON GET: `mount(action, handler)` where
  `handler: (http::request, Captures) -> Returns`, `emits []`.
- JSON POST: `mount(action, handler)` where
  `handler: (http::request, Captures, JsonBody) -> Returns`, `emits []`.
- HTML POST: `mount(action, handler, render_ok, render_rejected)` where
  `handler: (http::request, Captures, FormWire) -> Returns`,
  `render_ok: (Returns) -> html`, `render_rejected: (form::rejected<FormWire>) -> html`,
  all `emits []`, non-generic, non-variadic.
- `near` captures (`sql::pool`, `str public_origin`) are explicit lexical
  captures typed at the `routes(pool, public_origin)` call site; `mount`
  returns `http::route` and registers nothing on bare declaration import.

## I-3 explicit execution-context parameter (supplier: UP04; consumers: UP07/11/13)

- Generated async calls thread one explicit owner-context token parameter
  across every emitted `await`/callback/coordination edge; no ambient
  lookup, no synchronous ALS substitute.
- Token contract: acquire after suspension, interleave safely across owners,
  settle at root, fault late callbacks, drain admitted child work, dispose
  exactly once. UP04's runtime adapter plus UP11's generated-call lowering
  share this one parameter shape; UP07 seals it in the profile.

## I-4 browser runtime imports / identity table (supplier: UP07; consumers: UP10/11/13/15)

- Complete module inventory of the portable profile: every reachable
  runtime module and its sealed concrete type/error identities.
- Browser production profile excludes reachable Node `crypto`/`util`/`fs`/
  `async-hooks` (and any server-only module); private Can value branding,
  equality, int64/codecs, immutable completions, and sanitized standard
  failures are preserved.
- UP10 audits pre-bundle graphs against this table; UP15 bundles only it;
  inventory changes go through the coordinator (I), never per-lane edits.

## I-5 diagnostic record (suppliers: UP07/11/13; consumers: UP15/18/23)

- Immutable fields: `category`, `phase`, Can `file`/`line`/`column`
  (build-sealed source locations resolvable via published map/table).
- Excluded: native cause, raw input, stacks, secret bytes.
- Default browser sink: `console.error` of the frozen record, once per
  occurrence; startup faults and admitted event/timer faults share it.
  UP15 publishes the immutable diagnostic table; UP18 serves it as a
  verified asset.

## I-6 browser manifest fields (supplier: UP15; consumers: UP18/19/23/25)

- `browserBuildId`, locked shared-package instances, toolchain/source/
  catalogue inputs, and for **every** reachable runtime/published byte:
  content digest, logical path, and served route.
- Covers final digest JS, source map, and immutable diagnostic table.
  Same inputs → identical bytes and manifest. Tamper, missing map/table,
  secret canary, or unaccounted edge fails before publication.

## I-7 server asset selection (supplier: UP18; consumers: UP19/22/23)

- Server build input: explicit `--browser-manifest <file>` path (proposed
  CLI spelling). The server verifies browser build id, locked instances,
  asset graph, and every hash/reference, then records the exact browser
  build id + digests in its own verified build report.
- Publication is atomic over (report, entire asset table); failed rebuilds
  keep the prior verified set; replaced digest URLs persist ≥ 7 days
  across restart/cleanup/concurrent builds.
- Pages receive exactly one report-selected same-origin digest script URL
  (e.g. `/__can/assets/<sha256>.js`) under CSP
  (`script-src 'self'`, `connect-src 'self'`, no inline script, no
  `unsafe-eval`).

## Handoff protocol

Each supplying task reports the interface id, the concrete provider
(commit + paths + symbol names), and any deviation with evidence. A
deviation that changes observable behavior returns to design under the
three-fresh-consultation rule; workers do not relax the contract silently.
