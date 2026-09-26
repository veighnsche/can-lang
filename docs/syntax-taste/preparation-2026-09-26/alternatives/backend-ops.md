# P04 alternatives — backend/workers/deployment/AI/docs (R10, R11, R13, R14, R16)

## R10 relational breadth

- **RETURNING.** O1 retain (transaction + SELECT re-fetch; app-supplied keys
  as today). O2 admit `RETURNING` descriptors with specified per-dialect row/
  cardinality/error semantics (P20 reopening needs a real post-write-read or
  race-cost demo first).
- **JSON/timestamp/decimal.** O1 retain app-level encodings (minor-unit money,
  ms-since-epoch ints, opaque TEXT + app codecs) with a blessed-encoding guide.
  O2 add column kinds/codecs with validation + row-boundary rules.
- **PostgreSQL conflict recipe.** The webhook `decide_receive` catch-then-reread
  inside one transaction does not transfer to PG (aborted transaction, `25P02`;
  only ROLLBACK or savepoint recovery). O1 lookup-first shape (portable, no
  Can change). O2 savepoint support in Can SQL (new contract: savepoint
  lifetime, failure mapping, commit-uncertainty interaction).
- **Migrations.** O1 retain operator-owned DDL (schema.sql applied by hand).
  O2 versioned migration/deployment provenance with an executed drift case
  (P19 reopening).
- **Worker-claim reads.** Whether `SELECT ... FOR UPDATE / SKIP LOCKED`
  passes the PG backend is untested — a descriptor probe at implementation
  time, not a syntax change. Interacts with R11 claim design.

W6's "real PostgreSQL app" (generated identities, precise values, nullable
audit times, JSON, concurrent replay, migration recipe) is the acceptance
that decides how far past O1 each bullet must go. SELECT expressiveness is
not the gap (CTE/UNION/JOIN/subselect/OFFSET/ILIKE/aggregates admitted).

## R11 durable workers and outbound policy

- **Worker home.** O1 Can worker: bounded concurrency, claims across two
  workers, retries/backoff, poison jobs, crash recovery — all in Can, reusing
  transactions + commit-unknown + reconcilers. Needs a claim mechanism
  (locking-read vs lookup-first vs lease table), backoff/poison conventions,
  and a 2-worker crash-recovery qualification. O2 companion service: assign
  carrier/scheduling outside Can (as the webhook sample does today) with a
  documented protocol + auth envelope. The review explicitly allows either;
  the user scopes it (S1).
- **Destination policy.** Fixed-origin HTTP stays unless this round designs:
  destination allowlist/pattern form, credential binding (env-name-only,
  never logged), redirect rules, assertion/fixture story for tenant-supplied
  URLs. Dynamic WebSocket URLs already exist — the HTTP restriction is policy,
  not missing capability. Unrestricted fetch is not an option under DI-19.
- **Claim mechanism (if O1).** Depends on the R10 locking-read probe; lease
  table and lookup-first are portable fallbacks with different contention
  behavior — compare under a bounded-concurrency + crash-recovery test.

## R13 deployment and support

Implemented path exists (Debian 13+ amd64, pinned Bun 1.4.2, pairing with
7-day retention, network-denied smoke); release qualification is what is
missing (UP25 on the x86 machine — never here).

- **Migrations:** pair with the R10 migration decision; until then the
  operator-applies-schema.sql boundary stays explicit.
- **Lifecycle recipe:** service unit, health checks, rollout/rollback
  procedure, credential provisioning beyond fd-3 snapshot consumption.
- **Old-browser acceptance:** define and qualify already-open-browser vs
  newly deployed server (retention proves old assets stay servable, not old
  app behavior against a new server).

## R14 AI product qualification

Wire/shape/fixture layer is strict and tested; quality, budgets, and eval
above the wire are absent ("loopback stub data only" is the standing truth).

- **Budgets/correlation:** design where enforcement lives, what identity
  (tenant? key? correlation ID?), and what failure surfaces on excess. Open
  scoping: token cap vs spend cap vs rate limit vs per-key quotas.
- **Model-change evaluation:** protocol + harness for re-qualifying a SaaS
  feature across model versions.
- **Scope guard:** streaming/continuation and provider-workflow expansion only
  for an accepted requirement (F-R14-03). The user scopes which SaaS features
  need AI qualification this round (S1/S3).

## R16 examples and documentation (corrections, not options)

- **F-R16-01:** replace invoice `match call env::required("")` startup body
  with bare `env::invalid_name("")`; rewrite the false "only catalogue calls
  originate errors" comment. `startup_window` arms unaffected.
- **F-R16-02:** reword `README.md:99` and `distribution/README.md:196` to
  "Linux target implemented (Debian 13+ amd64); release qualification
  deferred (UP25)". Keep true limits (signatures, upload, source-tree
  installer).
- **F-R16-03:** add the carrier-auth bullet to webhook "stated limits":
  `/outbox/pending` + `/outbox/ack` unauthenticated — loopback/trusted
  carrier only; scoped protocol demo, not an internet-facing template. Keep
  the honest constant-time note.
- **F-R16-04:** write one consistent supported story after R10–R14 boundaries
  land (P07).

All R16 edits keep `canlc assert` green for invoice + webhook (U06 gate).
