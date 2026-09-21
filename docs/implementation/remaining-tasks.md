# Remaining implementation: file ownership and agent handoff

This is the execution companion to [tasks.md](tasks.md), not a new language
specification. It expands the fourteen unchecked tasks without changing their
acceptance criteria or marking drafts complete. The C/Q/A/P specifications linked
from the ledger remain authoritative. Paths here are repository-relative.

**Implementation is paused at the user's request.** This handoff was prepared on
2026-09-21, against committed baseline `a2531f7`, with 36 of 50 tasks checked.
Resume implementation only when the user authorizes it. Planning does not authorize
release upload, signing with credentials, or live provider calls.

## Preserve the existing architecture

Keep these directories and their responsibilities. Add small files inside the
existing packages; do not create a second compiler, runtime, platform framework,
compatibility layer, or broad `utils` package to finish these tasks.

```text
compiler/
  main.go                         command dispatch; delegate current semantics
  lsp.go                          protocol adapter to replace in I41
  internal/
    source/                       byte spans and UTF-16 position conversion
    syntax/                       current AST, lexer, parser, formatting
    project/                      inert manifests, confinement, input snapshots
      assets.go                   planned I34 asset validation
      overlay.go                  proposed I41 unsaved source overlay
    resolve/                      package identity and eligible-kind lookup
    types/                        sealed nominal graph and concrete descriptors
      form_schema.go              I32 closed; shallow form shape derivation
      sql_schema.go               proposed I35 scalar/option SQL shape derivation
    catalogue/                    catalogue.json is the source of API truth
    check/                        source admission, exact bounds, specializations
      http.go                     I32 closed: HTTP/static callback checks
      server.go                   I33 closed: no server-specific checks needed (gate plus generic bare-opaque fixtures)
      assets.go                   proposed I34 static asset resolution
      sql.go                      proposed I37/I35 SQL descriptor/call checks
      transaction.go              proposed I38 callback/scope checks
    sql/                          planned I36/I37 upstream parser integration
      parser.go                   parser/scanner adapter, no second SQL lexer
      descriptors.go              checked statement and binding descriptors
    ir/                           typed, source-spanned operations
      http.go                     only if HTTP needs explicit nonordinary IR
      assets.go                   planned compiler-owned asset descriptions
      sql.go                      planned checked query/template operations
      transaction.go              only for distinctions existing call IR lacks
    emit/                         emit checked IR; never rediscover semantics
      program.go                  shared runtime state and operation bindings
      assets.go                   planned manifest-owned asset lowering
      sql.go                      planned native tagged-template emission
      transaction.go              only if dedicated checked IR requires it
    driver/                       bundled runtime, output ownership, CLI stages
  testdata/current/               maintained Can source fixtures, by feature
runtime/
  completion.ts                   private boxes at every promise boundary
  owner.ts                        resource scopes, leases, close and late observers
  data.ts / bytes.ts              immutable nominal data and owned bytes
  codec/                         the one exact JSON boundary
  transport/                     native fetch and bounded transport adapters
  ai/                            TypeSafe and Responses adapters
  assert/                        assertion identities, queues, fixtures, reports
  platform/
    http.ts / router.ts / form.ts  I32 closed; I33 reused snapshot/dispatch unchanged
    server.ts                    I33 closed: native Bun server ownership
    assets.ts                    planned I34 closed native asset serving
    sql-descriptor.ts            planned I37 private descriptor provenance
    sql.ts                       planned I35 pool/query/row adapters
    transaction.ts               planned I38 native begin/decision bridge
  modules.json                   closed runtime module/import inventory
  test/                          Bun unit and adapter tests beside their contracts
distribution/
  target.json                    exact admitted runtime/platform, no latest pin
  build.go / manifest.go         bundle assembly and integrity metadata
  assets/                        planned I34 pinned HTMX bytes and lock metadata
  release.go / install.go /
  update.go                      planned I39 distribution lifecycle
  notices/                       upstream licenses and required source obligations
tools/
  distbuild/                     thin bundle-building command
  gramcheck/ / modcheck/          maintained source/editor inventory checks
  runtime/                       bundled validation and source-map tools
    vendor/                      pinned tool dependencies and their locks
tests/
  conformance/                   native capability reports, separate evidence
  integration/                   staged release and full application tests
    browser/                     proposed I34/I42 browser-only test harness
    testdata/                    proposed SQL fixtures and test-only fault proxy
examples/                        I42 complete manifest-backed applications
std/ / sketches/                 I43 maintained catalogue/domain examples
research/sql-binding/            I36 bounded comparison, never a runtime fallback
docs/implementation/evidence/    dated task reports and consultation records
```

“Proposed” means the file does not yet exist and should be created only for its
stated responsibility. The tree is an ownership map, not an instruction to create
empty scaffolding. Existing `distribution` source-map tools actually live under
`tools/runtime/`; do not create a duplicate `distribution/tools/` tree from an
older task description. Existing package output names are hashed by the project
loader; preserve that implementation rather than copying illustrative filenames
from the original plan.

### Rules shared by every remaining task

1. Follow the current path: project → syntax/resolve/types/check → checked IR →
   emit → driver-owned generation → exact bundled Bun. Root-level predecessor
   compiler files are deletion candidates for I44, not places for new semantics.
2. Keep `compiler/internal/catalogue/catalogue.json` authoritative. When an API
   descriptor changes, regenerate its Go/TS/reference mirrors with `make catalogue`
   and verify `make catalogue-check`. Never patch `runtime/catalogue.ts` alone.
   An implementation need not change a signature already admitted by the inventory.
3. Update `runtime/modules.json` whenever adding a runtime module/import. Runtime
   copies in generated programs must share one namespace so WeakMap provenance,
   domain failures, callback receipts and ownership are not duplicated.
4. Put static contracts in `check`/`types`, explicit runtime distinctions in `ir`,
   and native TS operations in `emit`/`runtime`. Do not parse source spellings in
   the emitter or evaluate Can in Go to implement a new platform feature.
5. Reuse `Completion<T>` and `invoke`. Can `void` is TS `undefined`, including
   first-class callable signatures: use `Completion<undefined>`, not `void`.
   Do not expose raw native promises, streams, Request, Response, SQL, or resource
   objects as Can data. Keep opaque validation membership-based and trap-free.
6. Extend `runtime/owner.ts` only for a missing shared ownership mechanism. Put
   server/SQL-specific behavior in their platform adapters. Deadline expiry never
   means permission to revoke leases, force native close, or claim rollback.
7. Use native operations plus contract adapters. No handwritten SQL parser, JSON
   replacement codec, HTMX clone, custom scheduler, or backwards-compatibility ABI.
8. Preserve existing user/unrelated work. Inspect `git status` before editing.
   Do not sweep untracked files into commits. Commit each completed task with its
   evidence and checked ledger entry; do not check tasks based on partial tests.
9. Difficult design choices require the repository's three fresh Jev consultations:
   supply the evidence, rewrite all explanatory prose, audit semantic equivalence
   before dispatch, save requests/responses, investigate disagreement, then test.
   The consultation is advice rather than a substitute for source or runtime proof.
10. Keep source fixtures under `compiler/testdata/current/<feature>/`, Go tests
    beside the relevant pass, runtime tests under `runtime/test/`, and staged tests
    under `tests/integration/`. Browser JavaScript and DB fault tooling are test
    harnesses only, never application or distribution runtime dependencies.

### Execution order and evidence

A straightforward continuation is I32 → I33 → I34, then I36 → I37 → I35 → I38.
I41 can be completed independently once its existing dependencies remain green.
Then close I42 and I43, retire code in I44, qualify I39, enforce I45, and finish
I50. Dependencies in the ledger govern; numbering does not put I35 before I36.
Do not begin a dependent task by inventing a stub that silently accepts unsupported
behavior. Share any required small prerequisite correction with its owning task.

Each task report should name the baseline/final commit, files changed, exact
runtime/parser/browser/database versions, commands and results, negative cases,
and unresolved limitations. Separate real Can assertions, supplied outcomes,
raw-provider fixtures, native conformance and optional live-quality evidence.
Use dated `docs/implementation/evidence/YYYY-MM-DD/iNN-*.md` reports and link them
from the corresponding ledger row. Do not treat current machine `/tmp` artifacts
as permanent evidence or portable prerequisites.

The existing full compiler/integration command uses `CAN_BUN`, `CAN_BUN_ARCHIVE`,
`CAN_TSC`, and optionally `GOCACHE`. Resolve these from the pinned manifest and
local installation; do not hardcode the prior agent's home or temporary paths in
repository configuration. Run focused checks while editing, then the relevant
full gates. Do not change compiler/runtime/distribution/tools files during staged
integration runs: source snapshots and integrity hashes must remain stable.

## I32 — HTTP values, forms and exact router

**CLOSED 2026-09-21.** Acceptance: [evidence/2026-09-21/i32](evidence/2026-09-21/i32/README.md).
The plan below is kept as dependency context for I33/I35/I38/I42.

### Handoff state: drafts, not completed implementation

Existing uncommitted I32 files are `compiler/internal/types/form_schema.go`,
`form_schema_test.go`, `runtime/platform/form.ts`, `http.ts`, `router.ts`,
`runtime/test/form.test.ts`, and `http-request.test.ts`. `runtime/modules.json`
has their import inventory. Consultations are in
[evidence/2026-09-21/i32-jev](evidence/2026-09-21/i32-jev/decision.md).
Do not recreate these files from scratch or assume they are committed.
The unrelated untracked `docs/implementation/design-revisions.md` was not authored
or reviewed as part of this handoff; leave it outside I32 changes.

The runtime draft has bounded detached request snapshots, strict percent/UTF-8
form decoding, status/header/response constructors, and exact GET/POST dispatch.
Its APIs include `createRequests`, `createResponses`, `snapshotRequest`,
`nativeResponse`, `isHTTPValue`, `createRouter`, `dispatch`, and `isRouterValue`.
Inspect the files for exact signatures. Private response data can be reused;
each dispatch creates a fresh native Response after the callback finishes.
Same-method identical route spellings produce duplicate errors; distinct spellings
normalizing to one path produce ambiguity errors. These are documented design
interpretations, still subject to full compiler/integration verification.

Most recent draft verification: 18 focused Bun tests / 342 expectations; strict
TS of the draft modules and tests; all `runtime/test` tests (228 / 20,555);
`go test ./compiler/internal/...`. These numbers describe the paused workspace,
not a completed I32 acceptance report. No staged Can HTTP program is wired yet.

### File work and sequence

1. `compiler/internal/check/http.go` **new**: own HTTP specialization and static
   route admission. `check/program.go` **extend**: admit the fixed I32 catalogue
   signatures and gather specialization ingredients. Do not feed generic `T` or
   catalogue `$callback` placeholders directly to the ordinary type parser.
2. `check/codec.go`, `check/specialize.go`, `types/form_schema.go` **extend/reuse**:
   specialize `request_json<T>`, `request_form<T>`, `response_json<T>` using the
   current concrete type graph. JSON uses `types.Schema`; forms use the shallow
   form descriptor. Prefer a dedicated HTTP specialization when differing input
   arity/error bounds would make the two existing codec operations less clear.
   Inspect both `finishCodec` and initial specialization finalization in
   `check/program.go`; avoid two inconsistent finishing paths.
3. `check/callables.go`, `check/completions.go`, `check/arguments.go` **integrate**:
   mounted callbacks must be named, accept exactly `http::request`, return
   `http::server_response`, and emit `[]`. Paths must be statically resolved.
   Cover direct calls, inferred/explicit generics, wrappers and callable references;
   a first-class reference must not bypass static-argument requirements. Reuse
   current capture checking and preserve captured pool ownership for I33/I35.
4. `ir/http.go` **only if necessary**: carry checked descriptors/static paths when
   ordinary invocation IR cannot represent them. Do not add a second invocation
   pipeline. `emit/program.go` **extend**: bind fixed methods and typed factories,
   supply descriptor JSON, and import aliases in both program and assertion roots.
   `$canResponses` already names the AI adapter: use a distinct HTTP alias such as
   `$canHTTPResponses`. Instantiate `createRequests<ConcreteHeader>` so headers do
   not degrade to `unknown[]` under strict TS.
5. `emit/program.go` domain initialization **extend**: validate concrete HTTP and
   router opaque kinds through their private membership functions, following the
   existing HTML kind map. Include specialized descriptor/error types in emitted
   type graphs. Avoid mapping concrete generic identities by declaration alone.
6. `runtime/platform/{http,router,form}.ts` **finish/audit**, rather than replacing
   them: verify media-type precedence, body-budget versus decoded-number expansion,
   normalization aliases, duplicate headers, safe HTML sink, cancellation/read
   failure, and route path exclusions against P10. Form byte limits apply to the
   received form bytes, not an invented JSON reserialization. Per-call body reads
   must keep the immutable cached bytes. Validate boundary values before allocation.

### Tests and completion

Add `check/http_test.go` and `emit/http_test.go`; source fixtures in
`compiler/testdata/current/http/` (request/form/response/router contracts, generic
wrappers, callable use, rejection cases). Extend the existing runtime tests rather
than splitting the same HTTP error fixture setup across many unrelated files.
Add `tests/integration/http_test.go` using staged bundle patterns from
`html_test.go` and `fetch_test.go`. Invoke the absolute sidecar from another cwd,
strict-check emitted entry and assertions, and use a real loopback harness to
enter compiled named Can callbacks. Verify query +/percent distinctions, repeated
body reads, malformed form/UTF-8, empty versus absent options, 204/205/304 body
exclusion, 404/405/Allow, HEAD rejection, and completion before response conversion.
Keep the private test bridge separate from the public I33 server API. Close I32
only after source programs exercise all admitted operations with no TS casts
hiding incompatible contracts and no fallback to predecessor HTTP code.

## I33 — Native server lifetime

**Files:** create `runtime/platform/server.ts`, `runtime/test/server.test.ts`,
`tests/integration/server_test.go`, and `compiler/testdata/current/http/server.can`.
Extend `check/program.go`, the HTTP checker if necessary, `emit/program.go`, and
`runtime/modules.json`. Add `check/server_test.go` for configuration/callback
contracts. Reuse `runtime/owner.ts`, `completion.ts`, `entry.ts`, and I32 routing;
add no alternate server framework or parallel lease registry.

1. Represent `server_config` and `server` by private provenance. Validate the exact
   host/port/body-limit/shutdown ranges from P10/catalogue before native binding.
   Freeze reusable configuration separately from the owned live server resource.
2. Wrap `Bun.serve({fetch: async ...})`. Snapshot ingress before entering Can;
   map I32 snapshot refusals to 400/413. Acquire and release request leases exactly
   once; attach captured resources through existing callable/owner metadata.
3. Await dispatch's boxed completion, convert a successful response once, and map
   escaped standard/domain failures to the specified sanitized 500 plus appropriate
   diagnostics. Do not serialize native causes, credentials or domain payloads to
   the client. Test error paths also release their request leases.
4. Make start/open assertion behavior follow the existing external-boundary fixture
   rules in `runtime/assert/`; assertions must not unexpectedly bind a port. Wire
   `server_wait` and `server_stop` as `Completion<undefined>` first-class callables.
5. Model open → closing → settled with existing resource ownership. Stop acceptance
   using native `stop(false)`, await its settlement and request/coordination leases,
   and retain observation after a caller deadline. Never call `stop(true)` beneath
   active work as a production timeout shortcut. Verify repeated/stale operations
   against the specified resource-state rules.
6. Install/remove SIGINT/SIGTERM listeners at the lifecycle owner, avoiding one
   global listener leak per request. Explicit stop and signals must converge on
   the same close operation; `server_wait` must not settle merely because another
   caller's stop deadline expired.

Tests must include delayed handlers, thrown failures, early-winning coordination
with a late losing resource user, keep-alive requests, simultaneous stop callers,
signals in child processes, and nonsettling work bounded only by an external test
supervisor. Show successful drain, timed-out caller with still-owned native work,
and absence of unhandled rejection. Do not use force-stop test cleanup as evidence
that production graceful shutdown works.

## I34 — Packaged HTMX and assets

**Files:** create `distribution/assets/htmx-4.0.0.min.js` and its lock/provenance
metadata; add the upstream notice under `distribution/notices/`. Extend
`distribution/build.go`, `manifest.go`, their tests and vendor integrity checks.
Create `compiler/internal/project/assets.go`, `assets_test.go`,
`check/assets.go`, `check/assets_test.go`, `ir/assets.go`, `emit/assets.go`,
`runtime/platform/assets.ts`, and `runtime/test/assets.test.ts` where the stated
responsibilities require them. Extend driver artifact/publication validation and
`runtime/modules.json`; do not create a separate asset output root outside dist.

1. Acquire and verify exactly the P11 HTMX bytes/SRI. Record source URL, version,
   digest and license; no compile/run-time CDN or acquisition. Inspect existing
   `runtime/platform/html.ts` runtime-head output before changing it: I31 already
   emits the selected script/meta configuration and URL scalar checks.
2. Snapshot project assets through the existing `project.Manifest.Assets` and
   confinement helpers. Hash actual bytes, validate extensions/signatures and fixed
   MIME policy, and carry immutable checked descriptions into IR. Reject all
   executable/unsupported project formats, traversal and inconsistent byte signatures.
   Do not trust only an extension or accept user-chosen MIME overrides.
3. Emit manifest-owned `assets/<digest>/<safe-name>` artifacts and deterministic
   `/__can/project/<digest>/<name>` URLs. Include assets in content hashes,
   relocation/rebuild tests, complete publication validation and owned cleanup.
4. Resolve `asset::url` static names in the checker, including dependency ownership
   and missing names. Return the existing HTML URL provenance through a private
   runtime constructor; never create a public way to forge `html::url`.
5. Serve only compiler-known paths via native `Bun.file`. Wire reserved routes
   into `server.ts` before application routing and reject project routes shadowing
   `/__can/`. Set fixed MIME/nosniff/cache/ETag behavior and enforce immutable bytes.
   There is no arbitrary runtime filesystem route or directory listing.
6. Add `tests/integration/assets_test.go` and a pinned browser harness under
   `tests/integration/browser/`. Its dependency lock is test infrastructure, not
   emitted application dependencies. Keep authored example assets CSS/data/images;
   only the packaged HTMX script executes in the browser.

Browser evidence must verify actual 422 swap, 204 no-swap, other error no-swap,
polling/trigger behavior, and the pinned `mode="same-origin"` plus `noSwap`
configuration. Block external network/CDN and attempt hostile response
content/destinations. Store browser/version and screenshots or DOM/network evidence
with the task report. Literal HTML snapshots alone cannot close I34.

## I36 — Select upstream PostgreSQL parser binding

**Files:** create a bounded `research/sql-binding/` comparison with `README.md`,
shared `corpus/`, candidate-specific harnesses and machine-readable results. Only
after measuring, create `compiler/internal/sql/parser.go`, `parser_test.go`, and
adopt the chosen dependency in `go.mod`/`go.sum`; record its notice/source duties
under `distribution/notices/`. Do not add both candidates to production.

1. Read current primary upstream documentation and identify exact revisions for
   the official CGo and WASM/wazero paths, using the same selected PostgreSQL major.
   State parser/scanner availability, supported output schema and build prerequisites.
2. Use identical SQL bytes and expected statement/parameter locations for both
   harnesses: ordinary and repeated parameters, strings, escaped/dollar quoting,
   nested comments, Unicode before tokens, malformed grammar and multiple statements.
   Preserve raw upstream outputs for discrepancies; do not normalize away mismatches.
3. Measure reproducible cold invocation, warmed runs, memory if available, binary
   size and macOS arm64 build complexity. Separate compilation cost from parsing
   cost. Test the release binary without end-user C tools or external WASM files
   unless those files are explicitly bundled and integrity-checked.
4. Choose with evidence and required Jev consultation. The production adapter
   exposes parsed statements, actual scanner tokens and byte locations in a small
   compiler-owned shape. Do not expose backend-specific ASTs across every checker
   file or write a fallback lexer when the scanner does not supply required data.
5. Translate upstream diagnostics into current source/manifest diagnostics without
   executing SQL, reading credentials or creating output generations. Add native
   binding provenance to build/release manifests as appropriate to the chosen model.

Completion requires measured comparison and an exact dependency pin, both parsing
and scanner tests, successful arm64 release-candidate execution, and documented
licensing. An assumed faster/simpler candidate or parse-only demonstration does not
satisfy this task. I37 must wait for scanner/span correctness.

## I37 — Static SQL descriptors and native bindings

**Files:** `project/manifest.go` already parses `SQLDescriptor`; preserve that
shape. Create `compiler/internal/sql/descriptors.go`, `descriptors_test.go`,
`check/sql.go`, `check/sql_test.go`, `ir/sql.go`, `emit/sql.go`, `emit/sql_test.go`,
`runtime/platform/sql-descriptor.ts`, and `runtime/test/sql-descriptor.test.ts`.
Extend `check/program.go`, `emit/program.go`, driver build-input hashing and
`runtime/modules.json`. Fixtures belong under `compiler/testdata/current/sql/`;
staged tests in `tests/integration/sql_descriptors_test.go`.

1. Resolve manifest parameter/row names through current package identities and
   sealed types. Retain cardinality and parameter field order; do not treat bare
   name strings as proof of a matching concrete record.
2. Feed exact statement bytes to I36. Require one approved statement, contiguous
   application parameters and exact record correspondence. For row-returning SELECT,
   prove the top-level LIMIT is the distinct required parameter after application
   parameters. Reject unbounded SELECT and INSERT/UPDATE/DELETE with RETURNING.
3. Use scanner byte offsets to produce alternating literal segments and checked
   field/limit references. Repeated `$1` must repeat the same prepared value at
   the correct interpolation sites; text resembling `$1` inside a quote/comment
   must remain literal. Preserve UTF-8 boundaries and source locations.
4. Carry the checked descriptor in typed IR, including concrete parameter/row
   identities and cardinality. Static descriptor names must not become runtime SQL
   lookup strings supplied by application input. Cover callable/generic routes
   into the intrinsic, not just direct source calls.
5. Emit native Bun tagged-template operations from those segments. The private
   descriptor constructor is compiler-owned and provenance-checked. Never emit
   `.unsafe()`, `.simple()`, textual value concatenation or a second SQL scanner.
   Keep descriptor construction inert under the existing C8 manifest case.
6. Establish a narrow interface consumed by I35: validated template/binding
   metadata and typed row/parameter descriptions. Avoid opening a pool merely to
   validate or compile a manifest. Parameter value validation still occurs before
   native query launch in I35.

Acceptance includes scanner offset negatives, wrong field/type/order/limit and
forged descriptor tests, strict TS of emitted queries, deterministic relocated
builds, and real PostgreSQL parameterization (including quote/injection payloads).
A test-only native DB harness may exercise descriptors before public I35 pool APIs
exist; it must not become an authored raw-driver escape.

## I35 — Typed Bun.SQL pools and rows

**Files:** create `runtime/platform/sql.ts`, `runtime/test/sql.test.ts`,
`compiler/internal/types/sql_schema.go`, `sql_schema_test.go`; extend I37's
`check/sql.go`, `ir/sql.go`, `emit/sql.go` and shared program wiring. Use
`tests/integration/sql_test.go`, `compiler/testdata/current/sql/`, and explicit
SQL setup data in `tests/integration/testdata/sql/`. Register runtime imports.
Keep descriptor provenance in `sql-descriptor.ts`; keep pool/query behavior here.

1. Qualify pinned Bun.SQL connection establishment and lazy query behavior with
   a disposable PostgreSQL instance. Determine what proves `pool_open` connected
   rather than merely constructing a lazy client. Document the native operation
   chosen; do not claim establishment from construction alone.
2. Read the named credential from the existing captured environment boundary,
   validate max-connections/configuration, and construct native SQL with
   `adapter: "postgres"` and `bigint: true`. No ambient environment enumeration,
   URL logging or credential reread for an already prepared operation.
3. Derive the finite SQL scalar/option record schema: bool, signed-64-bit int,
   finite float, scalar-valid str, owned bytes, or one optional layer of these.
   Reject arrays/JSON/timestamps/decimal/arbitrary variants at the source boundary.
   Validate all parameters, including int64 range and surrogate rejection, before
   lazy native query launch; prepare values once in source order.
4. Decode rows into immutable nominal records using existing data/bytes APIs.
   Check missing/extra columns, null rules, unsafe native numbers, bigint, finite
   floats and copied binary data. Keep path-specific schema mismatch classification.
   Reuse concrete `option::some/none` identities, not a generic declaration-name map.
5. Bind LIMIT 2 for one/optional and max_rows+1 for many. Bound max_rows before
   native numeric conversion. Distinguish missing/overflow and count only observed
   bounded rows. Validate affected-row counts for execute against the driver.
6. Register pools and queries with `owner.ts`; fixture boundaries must not launch
   real SQL. On close, deny new owners, drain leases, then call native close with
   remaining deadline. Timeout leaves the resource closing/observed. Keep native
   failure classification finite and sanitize payloads/diagnostics.

Tests need real connection refusal, parameter injection resistance, binary/option
round trips, invalid rows, constraint failures, exact error bounds, no launch on
bad parameters, affected counts and close with active captured work. The staged
suite must consume a disposable pinned DB configuration explicitly; never use an
unrelated developer database. Report PostgreSQL/driver versions and cleanup.

## I38 — Transaction decisions and uncertain commit

**Files:** create `runtime/platform/transaction.ts`,
`runtime/test/transaction.test.ts`, `check/transaction.go`,
`check/transaction_test.go`, `tests/integration/transactions_test.go`, and
`compiler/testdata/current/sql/transactions.can`. Extend SQL operation checking,
emission, concrete type graphs and runtime imports. Add dedicated transaction IR
or emitter files only where ordinary checked callbacks do not express the bridge.
A test-only fault proxy belongs under `tests/integration/testdata/`, not runtime.

1. Admit exact named callback `transaction -> decision<T> emits []`, ordinary
   concrete commit/rollback leaves, and distinct transaction query operations.
   Preserve generic results, near captures and callable identity. Reject nested
   transaction entry/escaped scoped handle use through the current scope contract.
2. Use native `pool.begin(async tx => ...)`; wrap the native transaction in a
   private owner-scoped handle. Reuse I35 value/row validation and I37 descriptors.
   Do not duplicate pool query semantics into an unrelated transaction subsystem.
3. After every callback outcome, mark its scope closing and drain registered
   owners, allowing existing subleases to finish. Keep the native callback pending
   throughout. Commit returns normally only after drain; rollback throws a private
   identity-checked sentinel carrying the boxed result, handled outside begin.
4. Preserve primary standard failure across drain/rollback and report secondary
   cleanup failures separately. Do not catch a standard failure as the rollback
   sentinel or turn failed cleanup into successful rollback.
5. Track whether callback entry occurred and whether a commit decision was produced.
   A native rejection after commit decision maps to `commit_unknown`; never retry
   automatically or assert the database rolled back. Keep transaction IDs safe for
   diagnostics without leaking driver payloads or credentials.

Verify typed commit/rollback results, late losing work, stale/escaped handles,
callback failure, rollback failure and nonsettling owners under an external
supervisor. Use a real DB fault proxy to lose commit acknowledgement after callback
completion, and record independent DB observation without turning it into a general
rollback guarantee. Deterministic fixture tests cannot replace this uncertainty test.

## I41 — Current diagnostics, LSP and editor grammar

**Files:** adapt `compiler/lsp.go` as protocol transport; create an inert bridge
`compiler/internal/driver/diagnostics.go` and `diagnostics_test.go` if the current
commands do not expose one. Add `project/overlay.go`/`overlay_test.go` for unsaved
source snapshots instead of writing editor buffers to disk. Update
`editors/vscode/client/extension.js`, `syntaxes/can.tmGrammar.json`,
`language-configuration.json`, the extension README/package metadata as needed,
`tools/gramcheck/`, and `tools/modcheck/`. Current LSP tests should live beside
`compiler/lsp.go` or under driver according to the layer tested.

1. Inventory legacy semantic calls in `lsp.go` before replacement. Keep sound
   Content-Length/JSON-RPC transport, but remove proof/signature-test/baseline
   execution hooks and predecessor AST use. Do not route an editor request through
   build/run/assert commands merely to obtain errors.
2. Build manifest-backed in-memory overlays keyed by canonical project/file identity
   and document version. Parse/resolve/check without network, SQL, environment reads,
   runtime startup, artifact publication or registry mutation. Retain unsaved
   multi-file changes as one coherent snapshot.
3. Convert current source spans through `internal/source` UTF-16 logic. Preserve
   diagnostic codes/severity and avoid basename-only file identity. Navigation
   should use resolved nominal/declaration identities, including generated fields.
4. Publish diagnostics only for the current version; clear stale errors on edits,
   close/delete and import changes. Handle malformed transient source without
   crashing or silently falling back to predecessor semantics.
5. Update grammar obligations to the current declaration/section syntax and remove
   extern/rev/dec/proof obligations. TextMate highlighting may be approximate;
   semantic acceptance remains the current parser/checker. Keep current snippets
   and mandatory assertions consistent with actual source fixtures.

Test protocol exchanges with Unicode, unsaved sibling modules, duplicate basenames,
rapid edits and generated AI fields. Compare static CLI/LSP diagnostics on the same
snapshot. Include deliberate external boundaries and prove zero runtime/provider/DB
activity while editing. Gate grammar/modcheck on maintained fixtures, not old goldens.

## I42 — Complete admission applications

**Files:** create manifest-backed projects in `examples/native-ai/`,
`examples/account-search/`, `examples/form-validation/`, and `examples/dashboard/`.
Each needs `can.project.json`, `can.errors.json`, `src/`, README and only declared
assets/dependencies. Keep sample DB schema/seed files in clearly test/deployment
support locations; no runtime migration API. Add `tests/integration/applications_test.go`
and browser scenarios under the I34 `tests/integration/browser/` harness.

1. Build native-ai from existing current Noul/Choice/Score/generation fixtures:
   generated plan → ordinary transformation → dynamic options → batched judgments
   → explicit confidence decisions → coordinated DB results → typed output.
   Keep handlers and error mapping authored Can, not test-only native callbacks.
2. Implement account-search from the P13 descriptor and named mounted callback
   contracts. Capture a pool through `near`, use shared typed SQL operations,
   immutable row rendering, safe HTML and packaged assets. Add full declaration
   documentation/assertions; spec fragments are not complete source programs.
3. Make form-validation exercise repeated/optional form values and 422 feedback;
   dashboard exercises polling and coordinated reads. Reuse the platform's reserved
   HTMX route and fixed response policy. Do not add authored client JS, templates
   with raw HTML, component frameworks or a second server/router.
4. Keep provider fixtures/local compatible endpoints explicit and separately label
   raw-provider versus supplied boundary results. Demonstrate once-only requests,
   whole-batch validation and early-failure/late-owner behavior in the applications.
5. Run complete programs from staged release layouts against local HTTP, disposable
   PostgreSQL and the pinned browser. Exercise malformed forms, invalid generation,
   invalid answers, all_failed, database failure and late resource diagnostics.

Completion requires every application to build freshly, assert its real ordinary
Can computation and run its documented interaction end-to-end. Tests may share
small staging helpers in `tests/integration`; do not extract application behavior
into a test harness to make an incomplete source example appear to work.

## I43 — Replace maintained stdlib and samples

**Files:** update `std/README.md`, `std/<existing-package>/`, `sketches/`, current
example manifests and catalogue references. Use
`compiler/internal/catalogue/integration_test.go` and `tools/modcheck/` for the
machine-checked inventory, adding `tests/integration/stdlib_test.go` for fresh
staged builds. Preserve useful existing `std/*/current/` projects until their
replacement is verified; do not rename every package solely for visual uniformity.

1. Inventory tracked Can examples and label each maintained replacement, intentional
   historical document, or retired artifact. Map each included coverage row to its
   catalogue operation/domain example, source assertion and integration evidence.
2. Replace old kernels/externs/proof brands/fuel algorithms with current declarations
   or ordinary Can examples calling the closed native catalogue. Do not promise
   compatibility for predecessor helper names or reintroduce excluded operations.
3. Keep `std/catalogue/` generated from the inventory; explanations/domain examples
   live in the owning package. Avoid a second handwritten stdlib declaration source
   that can drift from `catalogue.json`.
4. Rewrite surviving examples using current manifests, error registries, documentation,
   assertions and explicit imports. Their output belongs only in owned dist, never
   committed TS beside `.can` files. Trace assets and SQL descriptors through the
   same manifest loader used by applications.
5. Add inventory checks for an included operation without implementation/evidence,
   unsupported names reaching a legacy path, stale generated mirrors and examples
   omitted from fresh compilation. Record intentional P14 exclusions distinctly.

Do not declare the inventory complete based only on a directory sweep or a green
legacy test suite. I44 performs final physical retirement after current replacement
and editor gates are present; coordinate the deletion list with that task.

## I44 — Retire predecessor production paths

**Files:** inventory root-level `compiler/{parse,check,types,eval,emit,expand,result,bridge,verify_*}`
and associated tests/commands before deletion; exact filenames must come from the
checkout. Update `compiler/main.go`, `Makefile`, `go.mod`/`go.sum`, legacy test/data
owners, `.github/workflows/`, `tscheck/`, `std/`, and `sketches/` only as required.
Keep `compiler/internal/` current semantics, authored `runtime/*.ts`, bundled tool
sources and labelled historical design documents.

1. Produce an ownership/deletion table: old path, current replacement, passing gate,
   remaining importer/command/test. Use actual Go imports and command dispatch,
   not filename matching alone. I41 must have removed legacy LSP dependencies first.
2. Remove obsolete command modes, evaluator/proof/Z3 invocation and old ABI/extern
   grants. Delete old goldens/tests whose contract was superseded; preserve useful
   scenarios by rewriting them under current fixture/test locations first.
3. Remove tracked generated TS only after identifying its source owner. Do not
   classify `runtime`, `tools/runtime/vendor`, editor scripts or TS tests as generated
   Can output. Never restore the retired audit-probes archive.
4. Simplify main/build/module dependencies after all consumers migrate. Native source
   maps remain in `tools/runtime`; catalogue generation remains authoritative.
   No hidden compatibility flag, fallback parser or test-only legacy execution path.
5. Run clean-checkout Go, current Can, strict emitted TS and staged release tests.
   Search for remaining Z3/extern/old-syntax dispatch and validate explicit rejection
   fixtures. Ensure fresh builds do not create artifacts beside source files.

Retirement is gated deletion, not a broad cleanup opportunity. Do not reorganize
current internal packages or change public semantics while removing dead paths.
Commit a reviewable deletion/change list and link replacement evidence.

## I39 — Signed offline distribution, install and updates

**Files:** add `distribution/release.go`, `install.go`, `update.go` and adjacent
Go tests; extend `build.go`, `manifest.go`, `target.json` only when justified,
`notices/`, `distribution/README.md`, `tools/distbuild/main.go` and a dedicated
release workflow under `.github/workflows/`. Use
`tests/integration/release_test.go` for installed-layout behavior. Thin command
entry points belong in existing CLI/tools; distribution policy remains here.

1. Assemble launcher, exact Bun, current runtime, parser dependencies, HTMX, bundled
   source-map tooling, catalogue and notices into one integrity-described version.
   Preserve real-executable-root resolution and offline execution from I02.
2. Record upstream signature/entitlement inspection and licensing/source obligations.
   Prepare reviewable signing/notarization commands and chosen container. Actual
   signing with credentials, notarization submission and release upload require
   the corresponding user authorization; preparation is not release admission.
3. Implement staging/verification before atomic version selection. Keep running old
   installations intact; update never overwrites their Bun or calls `bun upgrade`.
   Failed verification/publication must preserve the previous selected version.
4. Validate ownership, containment, symlinks and active-use markers before removal.
   Test concurrent installation/update/run, interrupted staging, unknown files,
   malicious archives, wrong platform/minimum OS and permissions. Do not auto-acquire
   dependencies during compile/run or bypass Gatekeeper for a successful test.
5. Exercise clean quarantined installation on the admitted macOS arm64 target with
   PATH lacking development tools and networking disabled for runtime smoke tests.
   Record signature/notarization/container evidence separately from the development
   sidecar qualification report.

If signing credentials or a clean-machine environment are unavailable, report the
precise remaining gate and leave I39 unchecked. A locally runnable unsigned bundle
is useful evidence but cannot be labelled a signed release candidate.

## I45 — Mandatory macOS release-candidate gates

**Files:** adapt `.github/workflows/verifier.yml` and `tsc.yml`; add focused workflow
files only if separately operated release/browser/SQL gates warrant them. Update
`tscheck/package.json`, lock and tsconfig; `tests/conformance/`; maintained
integration harnesses; `distribution/qualify.py` and report schemas where needed.
Keep native qualification reports separate from language/assertion evidence.

1. Inventory all mandatory C/A/Q/P gates and their exact commands, inputs and report
   paths. Pin TS/Bun/Node type definitions, browser and DB test dependencies through
   their owning lockfiles. Do not rely on temporary tooling installed by this agent.
2. Build current examples and generated TS fresh in CI. Replace the existing
   committed-golden TS assumptions; check source and assertion outputs plus the
   packaged runtime. Treat missing required tools/tests as failures in the release
   job, not successful skips. Local focused suites may document optional setup.
3. Run Go/static, runtime Bun, staged CLI/assert, raw local HTTP, PostgreSQL ownership/
   transactions, HTMX browser, source maps and distribution integrity/installation
   gates on the actual admitted architecture. Check the runner architecture;
   a runner label or Ubuntu green status does not establish arm64/macOS admission.
4. Add bounded negative controls: break a fixture row, native capability, codec limit,
   owner drain, source map and asset digest, and show the corresponding gate fails.
   Keep these controls isolated from production source and restore clean inputs.
5. Stage the release layout and perform offline smoke from unrelated cwd with no
   global Bun/Node/npm/Go/C/Z3 requirement. Upload versioned reports even on failure,
   redact secrets, and preserve five evidence categories rather than one green badge.

I45 closes only when I39's actual signed artifact and the preceding application,
editor and retirement tasks pass their required gates. Do not lower mandatory
checks merely because the CI host lacks a browser, database or signing evidence.

## I50 — Documentation and final traceability

**Files:** root `README.md`, `compiler/README.md`, `std/README.md`,
`distribution/README.md`, editor/example READMEs, the docs index,
`docs/implementation/{tasks,coverage,plan,cli,assertions}.md`, and release notes.
Update other implementation guides only when their described behavior changed.
Keep historical design material labelled and linked, not silently rewritten as
current authority.

1. Reconcile documented commands/options/layout with actual shipped command output.
   Remove proof, backwards-compatibility, obsolete syntax and unsupported platform
   claims. Distinguish developer source-build prerequisites from end-user install.
2. Turn coverage rows into verified links: contract/capability → owner implementation
   → source/runtime/integration evidence → admitted release report. Include all
   F01–F15 closures and intentional exclusions. Broken or merely planned links do
   not count as completion.
3. Document exact runtime/upstream pins, asset/browser boundary, SQL value/transaction
   limits, fixture labels and shutdown/deadline behavior. Do not claim provider
   quality, cleanup guarantees or rollback certainty beyond the tests/contracts.
4. Follow install → build → assert → run instructions on a clean machine using the
   signed shipped layout and complete examples. Record command results and check
   that no undocumented tool/environment/cache/source-tree dependency is required.
5. Recount all fifty checkboxes and inspect every unfinished admission gate. Only
   then check I50 and complete the overarching implementation goal when resumed.
   Refresh or archive this handoff so future agents do not mistake paused draft
   status for the final release state.

## Before the next agent edits code

Read the user’s latest authorization, `AGENTS.md`, `tasks.md`, this handoff and the
relevant current specification sections. Inspect `git status` and the actual files
named by the next task. Reconfirm that the I32 drafts and unrelated user files have
not changed since this snapshot. Choose a bounded task, preserve the directory
ownership above, and finish its positive, negative and staged integration evidence
before marking it checked. No part of this planning update resumes implementation.
