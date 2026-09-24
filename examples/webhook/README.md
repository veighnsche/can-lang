# webhook

The signed provider webhook slice (T16): a receiver that verifies
exact-byte HMAC-SHA-256 signatures, records each delivery exactly once
in an idempotent ledger-plus-outbox transaction, and hands downstream
delivery to an operator carrier with durable attempts. Written in Can
against current native HTTP, crypto, SQL and transaction primitives;
no mutation `RETURNING`, queue or billing syntax was added.

## Layout

- `src/records/` — wire envelopes (`delivery`, `ack`), JSON bodies
  (`receipt`, `ack_result`, `outbox_page`) and ledger/outbox/attempt
  rows plus every statement parameter record.
- `src/model/` — header scan, signature check, digest, and the two
  transaction decision functions (`decide_receive`, `decide_ack`).
- `src/web/` — the four routes, the commit-uncertainty reconcilers,
  `boot` and the serving `main`.
- `schema.sql` — the three tables, applied by the operator before
  first serve. Descriptors admit only SELECT and RETURNING-free
  INSERT/UPDATE/DELETE, so Can cannot run this DDL itself.
- `can.project.json` — the seven statement descriptors.

## Run

```
sqlite3 hook.sqlite < examples/webhook/schema.sql
canlc assert examples/webhook
canlc build examples/webhook   # prints the build directory holding entry.ts
```

`main` takes a port and a database path; the HMAC secret arrives
through the `WEBHOOK_SECRET` credential snapshot on fd 3, never argv.
Serve the built entry with the bundled Bun runtime:

```
echo '{"WEBHOOK_SECRET":"..."}' > snapshot.json
bun dist/builds/<buildID>/entry.ts 18490 hook.sqlite 3< snapshot.json
```

The database file must already hold the schema; opening it with `rwc`
creates an empty file but no tables.

## Contract

`POST /webhooks/provider` reads at most 8192 raw body bytes, requires
one `x-provider-signature` header holding the lowercase hex
HMAC-SHA-256 of the exact bytes under the shared secret, and decodes
a `{"delivery_id","event","subscription"}` envelope. Non-UTF-8,
malformed, over-limit and empty-id bodies answer 400/413 before any
store touch; missing or wrong signatures answer 401. The sha256 hex
of the exact bytes is the ledger digest. UTF-8 decoding is fatal,
so the signature comparison over the decoded text is byte-exact.

`GET /outbox/pending` lists up to 16 pending rows for the carrier.
`POST /outbox/ack` takes `{"delivery_id","settled"}` (1024 bytes) and
records the attempt durably. `GET /health` answers `ok`.

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
- Acks follow the same shape: unknown ids answer 404, a done row
  replays 200 `done` without appending a second attempt, and failed
  deliveries stay pending with a durable `failed` attempt row.

## Outbox carrier protocol

Can owns receipt, the ledger, the outbox and the attempt rows. This
sample assigns downstream delivery and retry scheduling to an operator
carrier. Can supports named, fixed-origin outbound HTTP clients; this
sample does not define a tenant-selected destination policy or a
Can-authored carrier. The carrier loops: pull pending, POST each payload to the downstream
receiver, ack `settled:true` on its 200 or `settled:false` otherwise.
Downstream receivers deduplicate by `delivery_id`, so a crash between
their 200 and our ack is safe: the row stays pending, the carrier
redelivers, the stub sees two receipts for one recorded effect. The
attempt table records carrier-reported attempts only; a wire delivery
with no ack leaves no attempt row, which the crash leg demonstrates
honestly instead of hiding.

## Stated limits

- Signature comparison is exact but not constant-time; timing
  resistance is out of scope for this slice.
- Only the first 1000 request headers are scanned; a signature past
  that answers 401, failing closed.
- The SQLite leg assumes a failed statement does not poison its
  transaction (true for SQLite); a PostgreSQL port needs the
  lookup-first shape or savepoints for the reread-after-conflict.
- `commit_unknown` cannot be forced against local SQLite, so the live
  suite proves the surrounding contract (crash, replay, convergence)
  while `canlc assert` pins all six reconcile arms directly.
