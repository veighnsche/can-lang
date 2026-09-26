# webhook

The authenticated Can/companion webhook pair (F05): a Can receiver
that verifies exact-byte HMAC-SHA-256 provider signatures, records
each delivery exactly once in an idempotent ledger-plus-outbox
transaction, and serves an authenticated carrier protocol; plus a
Bun companion that claims leased rows, delivers them downstream
under the C-G destination policy, and settles each row idempotently.
Written in Can against current native HTTP, crypto, SQL and
transaction primitives, with the companion in TypeScript against the
shipped runtime modules; no mutation `RETURNING`, queue or billing
syntax was added.

## Layout

- `src/records/` — wire envelopes, JSON bodies, lease/outbox/attempt/
  dead-letter rows, and every statement parameter record.
- `src/model/` — header scan, signature check, digest, protocol and
  shape predicates, and the four transaction decision functions
  (`decide_receive`, `decide_claim`, `decide_heartbeat`, `decide_ack`,
  `decide_dead`).
- `src/web/` — the six routes, the commit-uncertainty reconcilers,
  `boot` and the serving `main`.
- `companion/` — the operator companion: versioned protocol client,
  delivery worker, bounded supervisor, and entry point, with unit
  tests. See `PROTOCOL.md`.
- `PROTOCOL.md` — the versioned carrier auth/protocol contract, the
  batch step/failure shape, and the honest limits. F06/H handoff.
- `schema.sql` — the five tables, applied by the operator before
  first serve. Descriptors admit only SELECT and RETURNING-free
  INSERT/UPDATE/DELETE, so Can cannot run this DDL itself.
- `can.project.json` — the thirteen statement descriptors.

## Run

```
sqlite3 hook.sqlite < examples/webhook/schema.sql
canlc assert examples/webhook
canlc build examples/webhook   # prints the build directory holding entry.ts
```

`main` takes a port and a database path; both HMAC secrets arrive
through the credential snapshot on fd 3, never argv. Serve the built
entry with the bundled Bun runtime:

```
echo '{"WEBHOOK_SECRET":"...","CARRIER_SECRET":"..."}' > snapshot.json
bun dist/builds/<buildID>/entry.ts 18490 hook.sqlite 3< snapshot.json
```

The database file must already hold the schema; opening it with `rwc`
creates an empty file but no tables.

Run the companion against the served Can side (see `PROTOCOL.md` for
the config and policy file formats):

```
CARRIER_SECRET='...' bun examples/webhook/companion/main.ts \
  --base http://127.0.0.1:18490 --worker worker-1 \
  --config deliveries.json --policy policy.json
```

`bun test examples/webhook/companion/` runs the companion unit suite
(34 tests against a verifying stub Can).

## Contract

`POST /webhooks/provider` reads at most 8192 raw body bytes, requires
one `x-provider-signature` header holding the lowercase hex
HMAC-SHA-256 of the exact bytes under the shared secret, and decodes
a `{"delivery_id","event","subscription"}` envelope. Non-UTF-8,
malformed, over-limit and empty-id bodies answer 400/413 before any
store touch; missing or wrong signatures answer 401. The sha256 hex
of the exact bytes is the ledger digest. UTF-8 decoding is fatal,
so the signature comparison over the decoded text is byte-exact.

The four carrier endpoints share one authenticated envelope
(protocol v1): an `x-carrier-protocol: 1` header, an
`x-carrier-signature` header holding the lowercase hex HMAC-SHA-256
of the exact body bytes under `CARRIER_SECRET`, and a JSON body
carrying `timestamp_ms` plus a single-use `nonce` inside the signed
bytes. Bodies cap at 1024 bytes. Unknown versions answer 400;
missing or wrong signatures, stale timestamps (outside +-300000ms),
and consumed nonces answer 401; malformed shapes answer 400. Every
carrier response carries a JSON body per the C-E status contract.

- `POST /outbox/claim` takes `{"worker_id","lease_ms"}` and leases
  the oldest claimable row to the worker (1000..600000ms).
- `POST /outbox/heartbeat` takes `{"worker_id","delivery_id",
  "version","lease_ms"}` and extends a held lease.
- `POST /outbox/ack` takes `{"delivery_id","settled"}` and records
  the attempt durably.
- `POST /outbox/dead_letter` takes `{"delivery_id","worker_id",
  "reason"}` and marks a poison row terminal.
- `GET /health` answers `ok`.

## Receiver duplicate policy

- Same `delivery_id` with the same digest replays the stored outcome
  with 200 `{"status":"duplicate"}` and writes nothing.
- Same `delivery_id` with a different digest answers 409
  `{"status":"conflict"}` and writes nothing; the first body wins and
  the provider must send the correction under a new id.
- A lost insert race rereads the winner inside the same transaction
  and follows the two rules above, so concurrent duplicates converge.
- `commit_unknown` rereads the ledger outside the lost transaction: a
  stored row replays duplicate/conflict by digest, a missing row
  answers 503 `{"status":"uncertain"}` and the provider retry either
  records the delivery or replays it. Retries never duplicate the
  business effect.
- Acks follow the same shape: unknown ids answer 404, a done or dead
  row replays its state without appending a second attempt, and
  failed deliveries stay pending with a durable `failed` attempt row.
- Only committed outcomes pin their nonce: a rolled-back request
  (unknown id, lost race, empty claim) frees its nonce, so an
  identical retry re-executes safely instead of answering replay.
  Replays of committed bytes answer 401.

## Companion lease and poison policy

- Claims are portable lookup-first lease takes: oldest pending row
  with an expired lease, guarded by a version-checked conditional
  write. A lost race answers empty and the loser retries honestly.
- Heartbeats extend only for the holding worker at the current
  version; foreign or stale holders answer 409 `lease_lost`.
- Acks are idempotent by `delivery_id` and ownership-free: any
  authenticated carrier may report the downstream outcome, and
  downstream receivers deduplicate by `delivery_id`.
- After 5 protocol-versioned attempts the companion marks the row
  via `dead_letter` (reasons: `poison`, `destination_denied`,
  `no_destination`); Can owns the dead-letter table and rules.
  Dead-lettering needs the holding worker; done rows answer 409.
- Delivery is at-least-once: unacked rows (crashed worker, lost
  lease, uncertain commit) redeliver after lease expiry, and the
  attempt table records carrier-reported attempts only.
- The companion enforces the destination policy per send (plus a
  resolved-literal recheck), binds credentials from the environment
  at send time, follows redirects only through the redirect policy,
  bounds concurrency, backs off between failure batches, and reports
  step-indexed batch outcomes. The supervisor restarts it on
  failure; a SIGKILLed companion needs no Can-side recovery beyond
  idempotency.

## Stated limits

- Signature comparison is exact but not constant-time; timing
  resistance is out of scope for this slice.
- Only the first 1000 request headers are scanned; a signature past
  that answers 401, failing closed.
- The SQLite leg assumes a failed statement does not poison its
  transaction (true for SQLite); a PostgreSQL port keeps the same
  lookup-first shape (no savepoints, no locking reads).
- `commit_unknown` cannot be forced against local SQLite, so the live
  suite proves the surrounding contract (crash, replay, convergence,
  redelivery) while `canlc assert` pins every reconcile arm directly.
- Claims take one row per request; batching is companion-side paging.
  Failed rows rest on their lease until expiry (the lease doubles as
  backoff); worker-level backoff between failure batches is coarse,
  not per-row.
- The DNS recheck leaves a documented TOCTOU window: a name that
  rebinds between lookup and connect can slip a private address past
  the check. High-risk deployments pin IPs.
- Redirects re-post the webhook body and never forward credentials
  off the configured destination. Downstream credential errors fail
  the attempt for retry, then poison after 5 attempts.
- Batch reports name deliveries and finite reason codes only: no
  secrets and no destination URLs, whose paths may carry tokens.
