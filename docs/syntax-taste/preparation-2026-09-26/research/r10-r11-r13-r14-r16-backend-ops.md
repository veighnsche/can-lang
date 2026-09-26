# Evidence packet: R10, R11, R13, R14, R16 @ `2cb1bc35`

Repo `/Users/vince/Projects/can-lang`, revision `2cb1bc35435ab5075d51f9f945adfb0e1f9dfe64` verified (`git log` head = `2cb1bc35`; status clean except the four untracked doc inputs for this round). Read-only; no repo writes. Environment: Bun 1.4.2, Go 1.27.1 darwin/arm64 — matching the pinned inputs (Bun 1.4.2+`744846f84`, Go 1.27.1 per completion audit).

---

## R10 — Relational breadth (F-R10-01..05)

### 1. Supported path + actual limitation

**Idiom.** SQL lives in `can.project.json` descriptors: exactly one statement, contiguous parameters, declared `parameter_type`/`row_type` records, one of three dialects, one of four cardinalities. Best current shape is the webhook/invoice pattern:

- Dialects `postgresql` | `sqlite` | `mysql`, each through its own pinned backend (libpg_query PG17 / tree-sitter SQLite / TiDB parser) — [dialect.go](/Users/vince/Projects/can-lang/compiler/internal/sql/dialect.go:18).
- Cardinalities `one`/`optional`/`many` require SELECT with a top-level `LIMIT` bound to the trailing parameter exactly once; `execute` requires RETURNING-free INSERT/UPDATE/DELETE — [cardinality.go](/Users/vince/Projects/can-lang/compiler/internal/sql/cardinality.go:29), rejection `"RETURNING is not admitted"` at [cardinality.go](/Users/vince/Projects/can-lang/compiler/internal/sql/cardinality.go:54).
- SELECT subset is wider than "bounded SELECT" suggests: JOINs, subselects, CTEs, UNION, OFFSET all admitted — [descriptors_test.go](/Users/vince/Projects/can-lang/compiler/internal/sql/descriptors_test.go:136). Live examples use `ILIKE`, `ORDER BY`, `COUNT(*)::int` (native-ai manifest), `WHERE`+`LIMIT ?` (webhook manifest).
- Schema admission: only `bool`, `int` (i64), `float` (finite), `str`, `bytes`, or one `option` layer thereof; rejects arrays, JSON, timestamps, decimals, nested options, owner records — [sql_schema.go](/Users/vince/Projects/can-lang/compiler/internal/types/sql_schema.go:5).
- Encodings actually used: money as minor-unit `int` ([records.can](/Users/vince/Projects/can-lang/examples/invoice/src/records/records.can:52) `price_minor`); timestamps as ms-since-epoch `int` via `clock::wall_millis` (`received_ms`/`updated_ms`/`recorded_ms`); JSON payloads as opaque `str`/`TEXT` handled by `codec` at the app layer (committed result JSON in `invoice_replay.result`) — [records.can](/Users/vince/Projects/can-lang/examples/invoice/src/records/records.can:66).
- Post-write reads: `execute` returns only an affected-row count ([postgres.ts](/Users/vince/Projects/can-lang/runtime/platform/sql/postgres.ts:45), [sqlite.ts](/Users/vince/Projects/can-lang/runtime/platform/sql/sqlite.ts:38), [mysql.ts](/Users/vince/Projects/can-lang/runtime/platform/sql/mysql.ts:58)); generated keys are re-fetched with a follow-up SELECT in the same transaction. There is no RETURNING, no last-insert-id API. Both examples use app-supplied keys (invoice integer keys, webhook `delivery_id TEXT PRIMARY KEY`); webhook's `AUTOINCREMENT id` is never read back — [schema.sql](/Users/vince/Projects/can-lang/examples/webhook/schema.sql:20).
- Schema/DDL is operator-owned: Can cannot run DDL ("Can descriptors admit only SELECT and RETURNING-free INSERT/UPDATE/DELETE") — [webhook schema](/Users/vince/Projects/can-lang/examples/webhook/schema.sql:1), [invoice schema](/Users/vince/Projects/can-lang/examples/invoice/schema.sql:1). No migration API (native-ai README: "there is no runtime migration API").

**Row-validation boundary** (runtime, fail-closed, never silent casts): exact column-set match (`extra_column`/`missing_column`/`null` mismatches), per-cell scalar checks, int64 range, finite floats, well-formed Unicode — [values.ts](/Users/vince/Projects/can-lang/runtime/platform/sql/values.ts:243). Only documented profile deviations: SQLite/MySQL decode bools from exact 0/1; MySQL additionally renders driver `Date` as naive UTC `"YYYY-MM-DD HH:MM:SS[.mmm]"` text and canonical digit strings as exact i64 — [values.ts](/Users/vince/Projects/can-lang/runtime/platform/sql/values.ts:31). Outbound params validated pre-launch — [values.ts](/Users/vince/Projects/can-lang/runtime/platform/sql/values.ts:143).

**Limitations (F-R10-01/02/04):** no RETURNING; no JSON/timestamp/decimal column types (chosen app-level encodings instead); no schema tooling/drift detection (P19/P20 deferred, audit-confirmed not implemented); **SQLite reread-after-conflict does not transfer to PostgreSQL** — `decide_receive` catches `constraint_failed` then rereads the winner *inside the same transaction* ([model.can](/Users/vince/Projects/can-lang/examples/webhook/src/model/model.can:91)), which SQLite tolerates but PostgreSQL forbids (aborted transaction; only ROLLBACK or savepoint recovery allowed). No savepoint support exists in Can SQL (negative grep). Webhook README already warns the PG port "needs the lookup-first shape or savepoints" — [README](/Users/vince/Projects/can-lang/examples/webhook/README.md:98).

### 2. Platform / primary-source facts

- PostgreSQL 17 (qualified 17.11 on 127.0.0.1:5433; Linux provenance observes `17.11 (Debian 17.11-1.pgdg13+2)`): per PG docs ("Transactions", error handling), any error aborts the current transaction; all further commands fail with `25P02 InFailedSqlTransaction` until `ROLLBACK` (or `ROLLBACK TO SAVEPOINT`). `classifyPostgres` maps `23xxx`/constraint to `constraint_failed` — [postgres.ts](/Users/vince/Projects/can-lang/runtime/platform/sql/postgres.ts:32) — but cannot rescue the transaction.
- SQLite: a failed statement does not poison the transaction (webhook README states this as the relied-upon property); busy waits via validated `PRAGMA busy_timeout` — [sqlite.ts](/Users/vince/Projects/can-lang/runtime/platform/sql/sqlite.ts:99).
- MySQL 8.4 service observed for errno map; sessions pinned to UTC `+00:00` and verified at open; naive DATETIME rendering depends on that pin — [mysql.ts](/Users/vince/Projects/can-lang/runtime/platform/sql/mysql.ts:100).
- Grammar pins: PG17 (`TestVersionPinsPostgreSQL17`), TiDB grammar version 80011 (mysql.test.ts hand-built descriptors), tree-sitter SQLite. Bun 1.4.2 `Bun.SQL` pooled clients, `bigint: true`, `safeIntegers` (SQLite).

### 3. Probes

None executed beyond version checks. Retained evidence covers execution: staged shards ran the live PG legs at `be95d009` (docs-only delta to `2cb1bc35`); the review ran the full compiler suite (2,298 pass events) plus runtime suite. Re-running the CGo-backed `compiler/internal/sql` package here would be a heavy fanless-Air build for no new information.

### 4. Facts vs uncertainties vs counterexamples

- Fact: supported subset, RETURNING rejection, row-validation boundary, per-dialect profiles — all source-anchored above.
- Fact: no real-PostgreSQL-app acceptance exists (F-R10-03 open): no test exercises generated identities, nullable audit times, JSON columns, concurrent replay, or a migration recipe against live PG.
- Uncertainty: whether `SELECT ... FOR UPDATE` / `SKIP LOCKED` (worker-claim shape) passes the PG backend — rich selects are admitted in principle but locking clauses are untested; needs a descriptor probe at implementation time, not a syntax change.
- Counterexample to "narrow SELECT": CTE/UNION/JOIN/subselect/OFFSET/ILIKE/aggregates all work — breadth gap is in *mutations/identity/schema*, not SELECT expressiveness.

### 5. Guarantees to preserve

Bound template parameters only (never string-call/unsafe); pre-launch param validation; fail-closed row decoding; commit-unknown semantics incl. the SQLite post-failed-COMMIT ROLLBACK cleanup — [transaction.ts](/Users/vince/Projects/can-lang/runtime/platform/sql/transaction.ts:234); exactly-one-live-transaction-per-extent; sanitized failure fields (no messages/URLs/credentials).

### 6. Decisions vs docs choices

- Technical: RETURNING row/cardinality/error semantics per dialect (P20 reopening needs a real post-write-read cost demo); JSON/timestamp/decimal codec choice (new column kinds vs blessed app-level encodings); savepoint-or-lookup-first PG conflict recipe; migration provenance design (P19).
- User/docs: which databases stay supported (PG/SQLite/MySQL?); whether "real PG app" acceptance (F-R10-03) is this round's milestone or a named follow-up.

---

## R11 — Durable workers / outbound policy (F-R11-01..04)

### 1. Supported path + actual limitation

**Idiom.** Can owns durable state; an external carrier owns delivery:

- Webhook slice: ledger + outbox + attempt tables, idempotent `decide_receive`/`decide_ack` transactions, `GET /outbox/pending` (≤16 rows) + `POST /outbox/ack` carrier protocol — [web.can](/Users/vince/Projects/can-lang/examples/webhook/src/web/web.can:184), [README](/Users/vince/Projects/can-lang/examples/webhook/README.md:77). "This sample assigns downstream delivery and retry scheduling to an operator carrier."
- What exists for a Can worker: transactions with commit-unknown + reconcilers, `sql::pool_open`/`query_*`/`with_transaction`, `clock::sleep_millis` (backoff), `process::run` (bounded subprocess carrier, SIGTERM→SIGKILL strategy), WS client with caller deadline. What does **not** exist: a Can-authored carrier/loop, scheduler primitive, claim/lease construct (`FOR UPDATE SKIP LOCKED` unproven — see R10), poison-job convention, multi-worker claim test.
- Fixed-origin HTTP: `connection` declarations take a **literal** `endpoint` (http/https, no userinfo/query/fragment, validated port/IP) + `timeout_ms` + optional bearer-env + static headers — [connections.go](/Users/vince/Projects/can-lang/compiler/internal/check/connections.go:30); runtime confines every request to the endpoint origin (`url.origin !== endpoint.origin` → invalid) — [request.ts](/Users/vince/Projects/can-lang/runtime/platform/request.ts:92) (actual path: `runtime/transport/request.ts:92`). Example: native-ai `generator`/`classifier` connections — [oracles.can](/Users/vince/Projects/can-lang/examples/native-ai/src/oracles/oracles.can:14).
- Dynamic WebSocket URLs already exist and must not be overgeneralized: `ws::connect` takes a runtime `url: str` (+ `deadline_ms`) and validates scheme/host at call time — catalogue `can.std.ws@1::connect`, [websocket.ts](/Users/vince/Projects/can-lang/runtime/platform/websocket.ts:408).

**Limitation:** no tenant-selected HTTP destination policy (O04 retained: "A dynamic client needs destination, credential, redirect and assertion rules, not unrestricted fetch"); no complete Can worker proving bounded concurrency, 2-worker claims, backoff, poison jobs, crash recovery (F-R11-02 open).

### 2. Platform facts

Bun `fetch` + `AbortController` underlie outbound HTTP with caller timeouts (backend reconciliation table); `Bun.SQL` pooling per R10; WS via native `WebSocket` with handshake timer. No external scheduler exists in-repo.

### 3. Probes

None. Carrier boundary is documented + the Go-carrier live test (`webhook_test.go`, SIGKILL crash legs at :235 and :476) is retained staged evidence.

### 4. Facts vs uncertainties vs counterexamples

- Fact: storage half (ledger/outbox/attempts/crash-replay) is implemented and live-tested on SQLite; carrier half is Deliberately Outside Can in the sample.
- Uncertainty: which coordination primitives give bounded multi-worker concurrency in pure Can (race/drain semantics exist but worker-pool shape unproven); whether descriptor SQL admits row-locking claims.
- Counterexample to "no dynamic outbound": WS connect is fully dynamic-URL — the HTTP restriction is a policy choice, not a capability absence.

### 5. Guarantees to preserve

Named fixed-origin clients + confined paths (O04); trusted-carrier envelope scope (O05); commit-unknown on worker writes; credential-by-env-name-only, never logged.

### 6. Decisions vs docs choices

- Technical: destination-policy design (allowlist/pattern form, credential binding, redirect rules, assertion/fixture story); worker-claim mechanism (locking-read vs lookup-first vs lease table); backoff/poison conventions.
- User/docs: whether the worker lives in Can at all vs a companion service (review's explicit either/or); which tenant-webhook shapes are in scope (S1).

---

## R13 — Deployment / support (F-R13-01..04)

### 1. Supported path + actual limitation

**Implemented Linux target (not a proposal):** Debian 13+ amd64/glibc only — target `bun-1.4.2-linux-amd64-v1`, `minimumOSVersion 13` — [target-linux-amd64.json](/Users/vince/Projects/can-lang/distribution/target-linux-amd64.json:1); host gate `ID=debian` + `VERSION_ID>=13` refusing derivatives — [os_linux.go](/Users/vince/Projects/can-lang/distribution/os_linux.go:25); x86-64 ELF sidecar verification; `LinuxTarget()`/`HostSupported()` selection — [manifest.go](/Users/vince/Projects/can-lang/distribution/manifest.go:196). Pinned Bun 1.4.2 rev `744846f84` (published 2026-09-05), digest-pinned `debian:13` + `postgres:17` (observed 17.11), Go 1.27.1 toolchain — [provenance.json](/Users/vince/Projects/can-lang/distribution/linux/provenance.json). Packaging: `build.sh` → versioned release → fresh-root install with manifest verification + `current`-symlink selection → `smoke.sh` (network-denied native qualification + process/files/crypto/SQLite app legs + rebuild-identity) + separate PG roundtrip leg — [linux README](/Users/vince/Projects/can-lang/distribution/linux/README.md:10).

**Paired server/browser deploy:** `--browser-manifest` pairing verifies every file hash, generation binding, post-bundle audit, and shared-lock agreement before publication — [pairing.go](/Users/vince/Projects/can-lang/compiler/internal/driver/pairing.go:85); replaced digest URLs stay servable 7 days — [pairing.go](/Users/vince/Projects/can-lang/compiler/internal/driver/pairing.go:19); rollback = versioned roots + `current` swap (distribution README). Proven by `TestInvoiceGridPagePaired` + served-matrix legs.

**Qualified vs not:** qualified at `be95d009` (docs-only delta to `2cb1bc35`): 106/106 integration, UP23 11/11 on Chromium 140/WebKit 26, 17 browser reports — completion audit. **Not rerun / deferred:** the entire Linux lane (UP25, deferred to the user's x86 machine — `TestLinuxInstalledArtifactSmoke`, `TestLinuxPostgresRoundtrip` skip off-target, [linux_distribution_test.go](/Users/vince/Projects/can-lang/tests/integration/linux_distribution_test.go:24)); live S3/MySQL; sustained load; model-quality eval. The Sept-26 review explicitly "did not rerun that lane."

**Gaps (F-R13-02/03):** no migration story ("no migrations" — distribution README:220; operator applies `schema.sql`); credential *provisioning* undescribed beyond fd-3 snapshot consumption; installer runs from the Go source tree; publisher signatures/upload open; **no already-open-browser-vs-new-server acceptance scenario** (negative search across tests/drivers — retention proves old *assets* stay servable, not old *app behavior* against a new server).

### 2. Platform facts

Debian 13 "trixie" amd64/glibc; Bun 1.4.2 linux-x64 archive sha256 `36368fae…` (31→36,646,985 bytes); base image + PG image digests pinned 2026-09-24. Builds never cross targets (macOS→darwin bundle, Debian→linux bundle).

### 3. Probes

None — Linux-amd64 execution is explicitly out of scope on this Air (F-R13-04 honored: no emulation, no Docker `linux/amd64`).

### 4. Facts vs uncertainties vs counterexamples

- Fact: packaging/install/smoke lane fully implemented; qualification evidence missing by deferral, not by absence of code.
- Uncertainty: whether the lane passes today (last provenance 2026-09-24; never rerun since).
- Counterexample to "no Linux support": the stale README sentence (see R16) — implementation exists; only *release qualification* is unclaimed.

### 5. Guarantees to preserve

Pinned runtime + provenance verification; never-cross-targets builds; paired-build hash identity + shared-lock binding; 7-day asset retention; network-denied smoke.

### 6. Decisions vs docs choices

- Technical: migration/versioning story; service-lifecycle recipe (systemd unit? health/rollout/rollback procedure); old-browser-open acceptance definition.
- User/docs: UP25 scheduling on the x86 machine; correcting the stale "No Linux support" sentences (R16); whether Linux is release-blocking or pilot-only.

---

## R14 — AI qualification (F-R14-01..03)

### 1. Supported path + actual limitation

**Shape guarantees (implemented):** Responses adapter builds a fixed non-streaming request (`store:false, stream:false, background:false, truncation:"disabled"`, strict JSON schema) — [responses.ts](/Users/vince/Projects/can-lang/runtime/ai/responses.ts:72); validates instructions/Unicode, envelope, terminal status, output shape, refusal vs truncation vs invalid, then `strict` schema-decodes — [responses.ts](/Users/vince/Projects/can-lang/runtime/ai/responses.ts:104). TypeSafe adapter validates question batch (1–256; choice 2–255 options; score 2–10 levels) and answers (exact question IDs, probability bounds, distribution keys + sum≈1, selected-argmax, score≈expectation, legend echo) — [questions.ts](/Users/vince/Projects/can-lang/runtime/ai/questions.ts:80). Raw-provider fixtures pin exact method/URL/headers/body with token-spelling JSON comparison — [provider.ts](/Users/vince/Projects/can-lang/runtime/assert/provider.ts:281). Connections pin literal endpoint + protocol allowlist (`openai_responses_v1`, `typesafe_systemone_v1`) + model + `timeout_ms`/`max_body_bytes`/`max_output_tokens` — [connections.go](/Users/vince/Projects/can-lang/compiler/internal/check/connections.go:113). Admission example: native-ai triage CLI (draft→options→review→gate→reads→report) against placeholder endpoints rewritten to a loopback stub — [native-ai README](/Users/vince/Projects/can-lang/examples/native-ai/README.md:1).

**Not established:** prediction quality, cost, latency for any SaaS feature — "No provider quality is claimed" ([README](/Users/vince/Projects/can-lang/README.md:99), [distribution README](/Users/vince/Projects/can-lang/distribution/README.md:218): "AI evidence is loopback stub data only"). **No tenant budgets, quotas, spend tracking, or usage reporting** (negative grep for budget/quota/spend/cost across `runtime/ai` and native-ai; `decodeResponse` does not parse any `usage` field). **No correlation plumbing**: per-request caps exist (`timeout_ms`, `max_body_bytes`, `max_output_tokens`) but nothing ties calls to a tenant/correlation ID (only `standard_failure`'s `occurrence_id` projection exists, unrelated to AI calls). No model-change evaluation harness. Narrow non-streaming scope is deliberate (F-R14-03).

### 2. Platform / provider-doc facts

Can-side only; no live provider calls inform any verdict. OpenAI Responses API and TypeSafe SystemOne are the two named protocols (connection metadata). Any quality/cost/latency numbers require live runs with credentials at implementation time (P02.4).

### 3. Probes

None needed: the claim is a *lack* of qualification evidence, established by negative source search + explicit "no provider quality is claimed" statements.

### 4. Facts vs uncertainties vs counterexamples

- Fact: wire/shape/fixture layer is strict and tested; everything above the wire (quality, budgets, eval) is absent.
- Uncertainty: what "tenant budget" should mean (token cap? spend cap? rate limit? per-key quotas?) — unscoped.
- No counterexample: nothing in-repo measures or enforces AI cost/quality.

### 5. Guarantees to preserve

Strict response/question validation; exact-match raw fixtures; literal-endpoint + bearer-env connections; per-request caps; O01 batch-atomicity (validate whole batch first, no rollback of authored effects) and O02 choice-cardinality rules.

### 6. Decisions vs docs choices

- Technical: budget/correlation design (where enforced, what identity, what failure when exceeded); model-change eval protocol; whether streaming/continuation ever enters scope (only for an accepted requirement).
- User/docs: which SaaS features actually need AI qualification this round (S1/S3); documenting "loopback stub only" as the standing boundary.

---

## R16 — Examples / docs (F-R16-01..04)

### F-R16-01 — Invoice `env::required` startup (DEF, confirmed)

[web.can](/Users/vince/Projects/can-lang/examples/invoice/src/web/web.can:502): comment claims *"Can failures originate only from catalogue calls, so a poison environment read with an empty name fails closed"* and the body calls `env::required("")` to synthesize `env::invalid_name`. The claim is false: authored error completions exist — a bare error-constructor terminal parses as `FailureBody` ([declarations.go](/Users/vince/Projects/can-lang/compiler/internal/syntax/declarations.go:439): "expected ok or an error constructor completion"), checks against the declared `emits` bound ([completions.go](/Users/vince/Projects/can-lang/compiler/internal/check/completions.go:349)), and emits `domain.create(...)` ([regions.go](/Users/vince/Projects/can-lang/compiler/internal/emit/regions.go:575)). **Minimal correction:** replace the `match call env::required("")` body with the bare completion `env::invalid_name("")` (keep `emits` + asserts; `startup_window`'s match arms on `env::invalid_name` are unaffected) and rewrite the comment to describe the intentional startup rejection.

### F-R16-02 — Stale Linux/browser claims (DOC, partially corrected — re-verified)

- Browser half: **fixed** — no anti-browser claim remains in the top-level README.
- Linux half: **still stale in two places**: [README](/Users/vince/Projects/can-lang/README.md:99) *"No Linux support is claimed"* and [distribution README](/Users/vince/Projects/can-lang/distribution/README.md:196) *"No Linux support is claimed"* contradict the implemented target ([target-linux-amd64.json](/Users/vince/Projects/can-lang/distribution/target-linux-amd64.json:1), [manifest.go](/Users/vince/Projects/can-lang/distribution/manifest.go:196), `distribution/linux/*`, [linux_distribution_test.go](/Users/vince/Projects/can-lang/tests/integration/linux_distribution_test.go:24)). The platform reconciliation already flagged these as stale (platform-reconciliation.md:7). **Minimal correction:** reword both to "Linux target implemented (Debian 13+ amd64); release qualification deferred (UP25)" — accurate because the lane exists but was never rerun. Keep the true limits (signatures, upload, source-tree installer).

### F-R16-03 — Webhook security/deployment limits (DOC, confirmed, partially bounded)

- Constant-time: README honestly states comparison is "exact but not constant-time" — [README](/Users/vince/Projects/can-lang/examples/webhook/README.md:94); implementation is ordinary string equality on hex — [model.can](/Users/vince/Projects/can-lang/examples/webhook/src/model/model.can:46). No code change implied; the bound just needs to survive into the user guide.
- Carrier authorization: **unstated gap**. `receive` requires the HMAC header ([web.can](/Users/vince/Projects/can-lang/examples/webhook/src/web/web.can:143)), but `outbox_pending` ([web.can](/Users/vince/Projects/can-lang/examples/webhook/src/web/web.can:184)) and `receive_ack` ([web.can](/Users/vince/Projects/can-lang/examples/webhook/src/web/web.can:288)) take no secret — any holder of the port can list/ack the outbox. "Stated limits" (README:92-103) covers timing, header-scan bound, SQLite-tx assumption, and `commit_unknown` provability, but not carrier auth. **Minimal correction:** add one bullet: "Carrier endpoints (`/outbox/pending`, `/outbox/ack`) are unauthenticated — bind to loopback or a trusted carrier; this sample is a scoped protocol demonstration, not an internet-facing template."

### F-R16-04 — One supported story (docs choice)

Blocked on scope decisions (S1–S3): the "one consistent supported story" must be written after R10–R14 land their boundaries. No technical probe applies.

### Guarantees to preserve (R16)

Mandatory-assertion evidence on every touched example (any R16 edit must keep `canlc assert` green for invoice + webhook); stated-limits honesty (U06 gate).

---

## Cross-topic notes

- **No credentials** appear anywhere in this packet; env-gated tests reference variable names only.
- **Reuse, not re-audit:** staged-shard evidence (`be95d009`, 106/106 + UP23 11/11 + 17 browser reports), review-run suites (705 runtime / 2,298 compiler events), and retained probes (race-drain, DOM semantics, recursion, result-data) stand as cited; nothing here re-runs them.
- **Decision vs docs-choice split:** R10 needs technical decisions (RETURNING semantics, codec/migration design, PG conflict recipe); R11 needs a technical destination-policy/worker design *or* an explicit companion-service assignment; R13 needs UP25 scheduling + migration/lifecycle recipes; R14 needs budget/eval scoping; R16 is pure correction work once wording is approved.