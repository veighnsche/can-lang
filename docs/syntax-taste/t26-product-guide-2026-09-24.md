# T26 product guide: platforms, action recipes and precise contracts

24 September 2026. Task T26 update to the authoritative product
documentation. This guide states the supported platforms, the recipes
for both action modes, and the precise SQL, race/shutdown, form/JSON
and owner boundaries that the T09–T25 implementation qualifies. It
does not change the [authoritative decisions](decisions.md),
[technical specification](technical-spec.md),
[AI/I/O](ai-io-spec.md), [coordination](coordination-spec.md) or
[platform/testing](platform-testing-spec.md) contracts; where this
guide and a normative contract disagree, the contract wins and the
disagreement is a defect to report, not a silent revision.

Every claim below names the file, test or gate that demonstrates it.
Nothing in this guide is inferred from Jev advice.

**Post-upgrade status:** the platform and delivery descriptions below have been
reconciled with current source and the later final acceptance record. The
[finding ledger](post-upgrade-reconciliation-2026-09-24.md) records unfinished
shared action binding, browser runtime delivery and UI acceptance. Recorded
test passes qualify their tested paths; they do not close those requirements.

**Post-upgrade implementation status (26 September 2026):** the 11
selected fixes are implemented and staged-qualified; the [fix
evidence](post-upgrade-fix-evidence-2026-09-26.md) links each row to
shipped source and passing tests, and the [clean-build
recipe](post-upgrade-clean-build-2026-09-26.md) reproduces the
qualification. The dated notes in §§1–2 supersede the pre-fix
sentences they annotate; those sentences stay as history of the
24 September baseline. Installed-target evidence (UP25) is deferred to
the x86 Linux machine; nothing below claims an installed run.

## 1. Supported platforms

### 1.1 Server runtime

- Bun 1.4.2, revision `744846f844374847c902b5e7fd59b4342a51ef99`.
- Development target `bun-1.4.2-darwin-arm64-v1`
  (`distribution/target.json`): darwin/arm64, archive
  `bun-darwin-aarch64.zip`, size 25377591, sha256
  `90987a3a16d7db556d886ac3d551e7b6d3edf0a1cf43acaed622e8676be1d12f`.
- Linux target `bun-1.4.2-linux-amd64-v1`
  (`distribution/target-linux-amd64.json`,
  `distribution/linux/provenance.json`): Debian 13 (trixie,
  glibc 2.41), archive `bun-linux-x64.zip`, size 36646985, sha256
  `36368faef7527875d5ffa52e53cd48021741f2a83eb6208a8dd64068d422a913`.
  Base image `docker.io/library/debian:13`, qualification database
  PostgreSQL 17.11 (`docker.io/library/postgres:17`).
- Go minimum 1.25 per `go.mod` (observed go1.27.1); TypeScript leg
  pinned by `package.json` (`oxfmt` 0.70.0, `oxlint` 1.85.0,
  `typescript` 7.0.2).
- The distribution is assembled only from an explicitly supplied
  local archive (`make bundle` requires `BUN_ARCHIVE`); assembling
  and running never downloads Bun.

Qualification: Gate 3 server matrix (`tests/integration/gate3_matrix_test.go`)
and Gate 4 fault suites (`tests/integration/gate4_fault_test.go`) run
against staged builds on these targets. The first emulated linux/amd64 run
failed a test-local 25ms budget in `runtime/test/transport-late.test.ts`.
Follow-up `5d96ac9` raised that budget to 2000ms while preserving the assertions;
the [final acceptance record](can-implementation-task-list-2026-09-24.md#final-acceptance-2026-09-24-head-5d96ac9)
reports the installed-artifact Linux rerun passing, including the shutdown
suite. This is recorded qualification under emulation, not evidence of a new
native (non-emulated) linux/amd64 run or arbitrary Linux-host coverage.

Post-upgrade staged acceptance (26 September 2026) reran the full
tree on darwin/arm64 at the fix candidate: Gate 3/4 server suites,
Gate 5 browser matrix (Chromium 140.0.7339.186, WebKit 26.0) and the
invoice suites pass; see the [fix evidence](post-upgrade-fix-evidence-2026-09-26.md)
and the [clean-build recipe](post-upgrade-clean-build-2026-09-26.md#full-staged-qualification).
The native linux/amd64 installed rerun remains the deferred UP25 step.

### 1.2 Browser matrix

Qualified by the Gate 5 grid matrix
(`tests/integration/gate5_frontend_test.go` plus the committed
harness `tests/integration/browser/grid.mjs`):

- chromium 140.0.7339.186 — qualified (27 checks).
- webkit 26.0 — qualified (same 27 checks).
- firefox — unavailable in the qualification run (Playwright
  launcher timeout); best-effort per design, not qualified.

`canlc build --target browser` emits TypeScript, a distinct browser root and
a content-addressed asset manifest. The served JavaScript in Gate 5 is built
separately by `tests/integration/browser/build-grid.mjs`, with filesystem,
utility, crypto and async-context shims. Those shims have grid-specific limits;
the test does not establish a general browser runtime or supported public
bundling path. Server projects are rejected for the browser target (9 SQL
descriptors in the invoice project exercise that rejection). The artifact
audit's text scan and skipped runtime bodies are remaining gaps.

The preceding paragraph is the 24 September baseline. Since the
upgrade, `canlc build --target browser` produces a served-ready
content-addressed module, source map, diagnostic table and manifest
through the compiler-owned bundle stage, and `canlc build
--browser-manifest <file>` verifies the server/browser pairing (U04);
no test bundler enters the supported path. The audit is structural
over every generated, dependency and runtime module plus the final
bundled bytes (B01). Current evidence: [U04 and B01 rows](post-upgrade-fix-evidence-2026-09-26.md#b01--structural-browser-audit).

### 1.3 What is not supported

No worker profile, no authored JS/TS in browser code, no arbitrary
raw markup or scripts, no server secrets/SQL pools/process access in
browser code (compile-time transitive capability closure, T21/T22).
Firefox, database migrations and offline reload durability are outside the
qualified scope below. SQLite is implemented and used by the invoice/webhook
examples; PostgreSQL 17.11 has separate native/application test evidence.
Other database versions or dialects require their own qualification.

## 2. Action recipes

The form action is mounted through its typed adapter in
`examples/invoice/src/web/web.can`; JSON save/load use manual authenticated
routes. The declared captured GET is not the mounted `/invoices/load` query
route. The browser test rewrites between them, and the grid duplicates action
declarations with stub handlers. The three server declarations are quoted below;
their presence does not establish shared contextual mounting (24 September
baseline; superseded — see the note after the listing):

```text
action save_invoice
    post "/invoices/save"
    body json records::invoice_json_wire
    handles save_json_validated
    result records::save_outcome
    cases
        records::saved => 200
        records::rejected => 422
        records::stale => 409
        records::denied => 403
        records::busy => 503

action save_invoice_form
    post "/invoices/save-form"
    body form records::invoice_form_wire
    handles save_form_validated
    result records::save_outcome
    cases
        records::saved => 200
        records::rejected => 422
        records::stale => 409
        records::denied => 403
        records::busy => 503

action load_invoice
    get "/invoices/{invoice_id}"
    captures
        str invoice_id
    handles model::load_validated
    result records::load_outcome
    cases
        records::found => 200
        records::load_denied => 403
        records::load_busy => 503
```

The listing above is the pre-upgrade spelling. Since the upgrade,
actions are handler-free declarations in one shared locked package
([`invoice_contract.can`](../../shared/invoice-contract/src/invoice_contract/invoice_contract.can)):
`action` rows declare method, path, captures, body mode, result and
cases; `handles` is a removal diagnostic; handlers bind separately at
the server via `action::mount`, which consumes the action symbol plus
an exactly-shaped callback and returns an `http::route` (U01/U02).
The declared captured path is the mounted route — no test rewrite —
and contract edits rebuild both targets or diagnose (U03). Current
evidence: [U01–U03 rows](post-upgrade-fix-evidence-2026-09-26.md#u01--shared-handler-free-action-declaration).

### 2.1 JSON POST-save and bodyless GET-load

- POST carries `body json <wire>`; GET carries no body. A GET with a
  body and a POST without its required body are checker diagnostics.
- POST wire is capped at 8192 bytes (`ACTION_JSON_BODY_LIMIT` in
  `runtime/platform/action-json.ts`); oversize requests receive a
  fixed 413 and the fetch consumer refuses to send them. A media-type
  gate rejects non-JSON posts before decoding.
- The compiler freezes typed `action::request` metadata (captures
  plus the JSON body wire contract) and typed `action::response`
  metadata (result plus the case table and the shared JSON wire
  schema of the result variant) into each `$canActions` row. Domain results
  map to the declared finite statuses; adapter/protocol failures can produce
  separate responses such as 400, 413 or 415.
- Browser Fetch lowering (`canlc build --target browser` plus
  `runtime/platform/action-json.ts`) distinguishes transport, abort,
  codec and unknown-status failures from finite domain cases.
  Browser cancellation is never interpreted as server rollback.
- JSON wire types reject form-only rows and owner records
  (`compiler/internal/check` JSON body checks;
  `bun test runtime/test/action-json.test.ts`, 14 pass).
- Consumers of the same action symbol use checked metadata. Before the
  upgrade the invoice/grid projects duplicated declarations, so
  cross-project edits could drift without a diagnostic. Since the
  upgrade both targets import the same locked package instance and
  route/capture/field/leaf/body edits propagate or diagnose (U01/U03);
  the old Gate 5 rewrite/wire/unlink limitations are gone with the
  origin proxy.

### 2.2 Keyed-row form POST

- POST carries `body form <wire>` over `form::rows<line_wire>` with
  `form::rejected<T>` retention. One native FormData parse, exact
  `lines_order`, a 64-row cap (`maxFormRows`) and a 2048-byte
  per-row cap (`maxFormRowBytes`) in `runtime/platform/form.ts`.
  Duplicate, unknown and partial rows are rejected while known-raw
  values are retained for redisplay.
- `http::serve_form_action` binds the three server callables for one
  form action: the valid-wire handler supplied at `action::mount`,
  the outcome renderer, and the structural-422
  renderer (`compiler/internal/catalogue/catalogue.json`,
  `can.std.http@1::serve_form_action`). The pre-upgrade `handles`
  row no longer exists (U01).
- Field and row names come from checked builders; rendering uses
  safe `html::safe` builders with values escaped at serialization.
  The emitted policy keeps `noSwap: [204, 304, "4xx", "5xx"]` and
  admits exactly the checked HTML action's declared cases through
  generated `hx-status:<code>` attributes, so 200/422/409/403/503
  fragments swap into their connected targets
  (`runtime/platform/html.ts`, compiler-owned
  `runtime/platform/htmx-guard.ts`). The accepted visible-503
  requirement is now met (S01); the pre-upgrade "served but never
  swapped" sentence is history.

### 2.3 Routes and captures

- Captures are typed `str`/`int64` whole-`{name}` path segments with
  a canonical URL builder (`runtime/platform/action-routes.ts`).
- Dispatch is method-first with static-beats-capture within a
  method; static/capture overlap at different positions is allowed,
  same-shape collisions are diagnostics.
- Classification is 400 (malformed escapes, dot segments, encoded
  separators — never reach the protected handler), 404 (no route)
  and 405 with an `allow` header naming the served methods.

## 3. Precise SQL guarantees

What the compiler checks (P12, `compiler/internal/sql/`):

- Each `sql` entry in `can.project.json` carries exactly `dialect`,
  `statement`, `parameters`, `parameter_type`, `row_type`,
  `cardinality` and, for row-returning descriptors,
  `row_limit_parameter`. Calls pass the static manifest name, never
  SQL text.
- The compiler parses with the pinned `libpg_query` PostgreSQL
  parser: exactly one statement, the cardinality's statement shape,
  contiguous application parameters `$1` through `$N` matching the
  parameter-record fields in declaration order.
- Every row-returning descriptor is a `SELECT` whose top-level
  `LIMIT` is the distinct trailing parameter named by
  `row_limit_parameter`. The adapter binds 2 for `one`/`optional`
  and `max_rows + 1` for `many`; the extra row produces
  `row_count`/`row_limit`, so the database bounds transmitted rows
  before Bun builds its result array. Mutation descriptors admit
  `RETURNING`-free INSERT/UPDATE/DELETE only.
- Generated TypeScript emits a Bun tagged-template query at the
  scanner-confirmed sites. It never emits `sql.unsafe`, `.simple()`
  or runtime-constructed SQL.

What the compiler does not check:

- The declared projection is not proven against a live versioned
  schema. Row shape is validated at runtime (`sql::schema_mismatch`
  on drift). Schema DDL is applied by the operator or harness
  before first serve (see `examples/invoice/schema.sql`), never
  from Can. There is no migration, unsafe-query, raw-driver or
  adapter escape.

Transactions (`sql::with_transaction<T>`, `sql::commit_unknown`):

- The adapter runs `pool.begin(async tx => ...)` and drains every
  coordination owner holding a transaction lease before the native
  callback settles. `commit` finishes normally; `rollback` throws a
  private sentinel so Bun rolls back while the payload returns as
  the successful Can value.
- If `begin` rejects after the callback produced `commit`, the
  adapter returns `sql::commit_unknown(transaction_id)`: the effect
  may or may not have committed, and the invoice replay ledger
  (§5) is the reconciliation path. A rollback/cleanup failure
  beside a primary standard failure is a secondary diagnostic and
  never replaces the primary failure.
- `with_transaction<T>` can return a now-closed handle inside
  otherwise successful data; later use is `resource_state`.

## 4. Race and shutdown semantics

Demonstrated, not inferred (`docs/implementation/shutdown.md`,
`runtime/test/shutdown.test.ts` 6 pass, owner/server/transport
suites). There is no general `finally`, no cancellation of
arbitrary Can computation, and no consumed-fault channel.

- Races follow native selection: the winner returns, losers keep
  their leases until they settle, and the root waits for them.
  Losing domain failures stay silent; each losing standard failure
  produces one sanitized late diagnostic. An empty first-completion
  race stays pending; the harness owns its observation deadline.
- The original outcome wins over close failure: a failing scope
  body keeps its exact completion while a failing automatic close
  sets `cleanupFailed` with a cleanup diagnostic. Cleanup never
  replaces an already selected domain outcome.
- Explicit close is exactly once per handle. A second close, like
  any use after close, is a standard `resource_state` rejection,
  never a domain failure.
- Scope-owned handles die with their scope, including a handle
  returned as the scope result; later use is `resource_state`.
- Abort exists only where an operation offers it: transport
  deadlines with native abort and owner-signal cancellation, stream
  cancel, WebSocket cancel/stop prompts. Cancelling one operation
  never disturbs a sibling.
- Client disconnect does not abort an owned server handler: the
  handler runs to completion under its lease and `stop` drains it.
- A caller close deadline bounds only the caller's wait; expiry
  reports the deadline failure while the native close keeps
  waiting for live leases and runs exactly once. Server `stop`
  closes with the configured `shutdown_ms`: the caller observes
  `shutdown_failed("deadline")` on expiry while handlers stay
  owned, and `wait` observes eventual settlement.
- Stream `close_reader`/`close_writer`/`cancel_reader` wait under
  a fixed 5s caller bound (`STREAM_CLOSE_MS`); cancel is terminal
  and records its reason for the interrupted operation.

## 5. Form/JSON/owner boundaries and the replay ledger

Owner records (DI-02):

- Construction, `with` update and field reads are owner-only; the
  package exports an explicit projection. Public leaf match and
  equality stay available to foreign readers.
- Generic codec and schema derivation reaching an owner record is
  rejected even inside the declaring package. Form schemas reject
  owner records; JSON wire types reject form-only rows and owner
  records. Real wire-decode-to-owner-factory positives go through
  the admitted factory path.
- A typed ID never grants authorization. The invoice server derives
  the actor from authenticated state and checks membership,
  invoice/tenant and revision at the protected write. Foreign and
  nonexistent resources share one nondisclosing 403.

Replay ledger (same-transaction, T14):

- `invoice_replay` is keyed by `(actor, invoice_id, operation_id)`
  with a payload digest, written in the same transaction as the
  invoice effect. The same payload replays to its recorded outcome
  without a second business effect; a changed digest under a reused
  ID conflicts (`stale`, 409). The 5 HTML + 5 JSON save cases and
  3 GET load cases in §2 exhaust the adapter mapping.

Form/JSON boundary summary: untrusted bytes enter only through the
§2 adapters (FormData parse or 8192-byte JSON decode), validate
into typed wire records with field paths and accumulated errors,
and cross into owner values solely via owner factories. No path
mints an owner value from raw bytes.

## 6. Initial offline limit

The Can-authored grid (`examples/invoice-grid/`) keeps an in-view
draft while offline, shows unsaved/failed/uncertain state, and
permits explicit identical-ID replay or authorized reconciliation.
Version-checked late outcomes never clobber newer edits, and
view-owned disposal releases listeners and timers (leak-tested).

Explicitly not promised: no reload durability, no automatic queue.
A reload drops the draft; there is no background retry queue. The server-rendered HTMX invoice form (§2.2) is the narrower
supported path for connected editing; it does not cover offline or
highly interactive use.

## 7. Stream example correction

`examples/stream/src/main.can` teaches the intended cleanup funnel
(T18): `pump` closes its reader exactly once on every terminal
path. Success reports a close failure; each failure arm
(`stream::read_failed`, `stream::cancelled`,
`files::limit_exceeded`) calls `stream::close_reader` and then
re-raises the primary error — a close failure during failure
cleanup never replaces the primary error. The doc comment on `pump`
states this contract, and `TestGate4FaultMatrix` plus
`runtime/test/shutdown.test.ts` pin precedence and exactly-once
close on real execution.

## 8. Deferred mechanisms

Per the task-list disposition, the following stay deferred and this
guide claims none of them: DI-03 finite error-set parameters, DI-08
capture-binding syntax, DI-14a mutation `RETURNING`, DI-17 new
native AI wrapper/batch/callable forms, DI-18 static
escape/consumed-fault additions, DI-19 unrestricted SDK imports,
DI-20 bulk immutable collections, DI-22 named-field constructors,
DI-23 multiline grammar. DI-21 billing/calendar is a later
policy-explicit library/example. DI-10 declarative markup and typed
DOM-target scopes remain unselected; the supported UI paths are the
safe builders plus the HTMX form (§2.2) and the bounded browser
catalogue (opaque `browser::app`/`browser::view`/`browser::node`/
`browser::state` handles, T22). The native-judgment agent-comparison
candidate is deferred with the current idiom retained; see
`tests/baseline/candidates/T26-native-ai/README.md`.

Post-upgrade boundary note (26 September 2026): the current
retained/deferred boundary is the [disposition
ledger](post-upgrade-dispositions-2026-09-24.md) — 19 deferred
proposals plus 9 retentions — not the DI list above alone. The
[completion audit](post-upgrade-completion-audit-2026-09-26.md)
re-verifies every exclusion; none of the deferred behavior is
described as shipped anywhere in this guide.

## 9. Consultations and agent comparisons

Jev SystemOne consultations (`docs/syntax-taste/jev-consultations/`)
are advisory classifications, not proof: three fresh requests,
unanimous 1.00 selections for single locked integration (T09),
same-transaction ledger (T14) and strict transitive browser closure
(T21/T22), investigated as a coherence check against the task
list's own acceptance rules. The retained engineering positions
follow from the failure-mode arguments, and every one of them is
demonstrated by the gate suites above rather than by the vote
count.

Held-out AI-agent creation/refactor/repair comparisons for the new
language mechanisms were registered under the T01 protocol and
replayed mechanically in
`tests/baseline/reports/t26-agent-comparison-2026-09-24.md`. No
live model trials ran in this environment, whole-task tokens are
reported only where measured, and no significance is claimed from
small samples. Candidates that fail the T01 mechanism acceptance
rule are deferred, including the native-AI candidate (§8).
