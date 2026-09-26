# Carrier protocol v1 — F05 handoff to F06/H

The versioned authenticated protocol between the Can webhook program
(`src/`) and its companion (`companion/`). Can owns receipt, the
ledger, the outbox, attempts, the dead-letter table, and every
decide/ack transaction; the companion owns claims, delivery, retry
scheduling, bounded concurrency, poison handling, crash recovery,
and destination policy enforcement. Anything marked **v1-fixed**
below changes only with a coordinated protocol bump on both sides.

## 1. Authenticated envelope (v1-fixed)

Every carrier request is `POST` with a JSON body of at most 1024
bytes and two headers:

- `x-carrier-protocol: 1` — the only admitted version. Missing or
  foreign versions answer 400 `unsupported_protocol` before any
  signature or store work.
- `x-carrier-signature: <hex>` — lowercase hex HMAC-SHA-256 of the
  exact body bytes under `CARRIER_SECRET` (environment snapshot on
  the Can side, `CARRIER_SECRET` env on the companion side).
  Missing or wrong tags answer 401 `unauthorized`.

The signed body carries, beside the op fields:

- `timestamp_ms` (int): sender wall clock. Outside +-300000ms
  (**v1-fixed** skew) of Can's clock the request answers 401
  `stale`. Operators keep both clocks inside the window.
- `nonce` (str): single-use request id, 16 random bytes hex on the
  companion. The Can side inserts it first in the decision
  transaction; a duplicate answers 401 `replay`. Only committed
  outcomes pin their nonce: rolled-back requests (unknown id, lost
  race, empty page, store failure) free it, so an identical retry
  re-executes safely. Re-execution is safe on every rollback path
  because none of them mutates the outbox.

Comparison is exact, not constant-time (stated limit, unchanged).

## 2. Endpoints and status matrix (v1-fixed)

All responses carry JSON bodies (C-E status honesty: 200/400/401/
404/409/413/503, never 204/205/304).

`POST /outbox/claim` — body `{"worker_id","lease_ms",
"timestamp_ms","nonce"}`. Leases the oldest pending row whose lease
expired to `worker_id` for `lease_ms` (1000..600000, **v1-fixed**):

- 200 `{"status":"claimed","items":[{delivery_id,event,
  subscription,attempts,lease_until_ms,version}]}` — exactly one
  item. The companion pages until `empty`.
- 200 `{"status":"empty","items":[]}` — nothing claimable, or a
  lost lease race. Normal: retry after backoff.
- 400 `invalid_shape` (bad worker/lease), 401 `replay`/`stale`,
  503 `store_unavailable`/`uncertain`.

`POST /outbox/heartbeat` — body `{"worker_id","delivery_id",
"version","lease_ms","timestamp_ms","nonce"}`. Extends the lease
only for the holding worker at the current version:

- 200 `extended` (new `lease_until_ms`, `version` + 1), `done`,
  `dead` (stored lease, no write).
- 404 `unknown`, 409 `lease_lost`, 401 `replay`/`stale`,
  400 `invalid_shape`, 503 `store_unavailable`/`uncertain`.

`POST /outbox/ack` — body `{"delivery_id","settled",
"timestamp_ms","nonce"}`. Idempotent by `delivery_id`,
ownership-free: any authenticated carrier may report the
downstream outcome; receivers deduplicate by `delivery_id`:

- 200 `done` (settled, attempt `delivered`), `pending`
  (unsettled, attempt `failed`, attempts + 1), `dead`
  (already terminal, no new attempt).
- 404 `unknown`, 401 `replay`/`stale`, 503
  `store_unavailable`/`uncertain`.

`POST /outbox/dead_letter` — body `{"delivery_id","worker_id",
"reason","timestamp_ms","nonce"}`. Terminal mark by the holding
worker; reasons (**v1-fixed**): `poison`, `destination_denied`,
`no_destination`:

- 200 `dead` (marked or replayed, attempts pinned).
- 404 `unknown`, 409 `already_done`/`lease_lost`, 401
  `replay`/`stale`, 400 `invalid_shape`, 503
  `store_unavailable`/`uncertain`.

`POST /webhooks/provider` (unchanged T16 shape) and `GET /health`
sit beside the carrier endpoints; see `README.md`.

## 3. Lease, ack, and poison rules (v1-fixed)

- Claim = portable lookup-first lease take: scalar-subquery
  oldest-row read plus a version-guarded conditional write. No
  locking reads, no savepoints, no `RETURNING`. Any number of
  pending rows coexists with single-row claims.
- `N = 5` attempts (**v1-fixed**): a row claimed at 5 attempts is
  poison. The companion dead-letters it without delivering; Can
  never auto-transitions. `dead_letter` needs the holding worker
  (freshness not required); done rows answer 409.
- At-least-once with idempotent ack plus the C-C unknown-write
  vocabulary: unacked rows redeliver after lease expiry; duplicate
  downstream sends collapse on `delivery_id`, never on single-
  delivery assumptions. Failed acks keep the row leased, so the
  claim lease doubles as backoff until expiry.
- Commit-unknown reconciliation: heartbeat/ack/dead_letter reread
  the outbox row and replay precise states (heartbeat replays
  `extended` only at request version + 1, else 503). Claims cannot
  reread (the claimed id is unknowable outside the lost
  transaction) and answer 503 `uncertain`: the row stays leased
  to the worker and redelivers after expiry.

## 4. Companion behavior

- Scheduling: page claims to empty (max 64/batch default), then
  deliver under bounded concurrency (4 default). One claim per
  request; batching is companion-side.
- Backoff: deterministic exponential between failure batches
  (base 1s, cap 30s, no jitter); idle outboxes sleep 1s. Failed
  rows additionally rest on their lease until expiry. Per-row
  backoff is future work.
- Destination enforcement per send: C-G policy admission, DNS
  recheck of the resolved literal against the loopback/private
  scopes, send-time credential binding (`Authorization: Bearer`,
  resolved from the rule's env name; never logged). Denied and
  unmapped subscriptions dead-letter (`destination_denied`,
  `no_destination`); missing credential values fail the attempt
  for retry, then poison.
- Redirects: manual, each hop re-admitted through the redirect
  policy with hop cap and literal recheck; the webhook body is
  re-posted and credentials never cross origins.
- Slow sends: heartbeat cover at half the lease until the send
  settles; lost leases abort the send and leave the row to its
  new holder. Downstream timeout 10s default.
- Failures: 400/401/413 and malformed bodies throw fatal (fail
  fast, never spin); 500/503 and transport failures back off and
  retry with fresh nonces; mid-flight store pressure leaves rows
  leased for redelivery.
- Supervisor: the entry runs the worker under bounded restart
  (5 restarts, exponential backoff, then loud death). A SIGKILLed
  companion relies on the external supervisor (service unit or
  container restart policy): unacked rows expire and redeliver
  with no Can-side recovery beyond idempotency. SIGTERM/SIGINT
  drains between batches and exits 0.

Config files (operator-owned, secrets never inside):

- Deliveries: `{"deliveries": {"<subscription>": "<url>"}}`.
- Policy: a C-G destination policy JSON (`version`, `rules[]`
  with `scheme`/`host`/`port?`/`pathPrefix?`/`credential?` env
  name, `redirect`, `maxRedirectHops`, `loopback`,
  `privateNetworks`). See `runtime/outbound/fixtures/
  companion-policy.json` for the shipped shape.

## 5. Batch step/failure shape (C-A mirror for F06)

Each batch reports one JSON line on stdout:

```json
{
  "protocol": "1",
  "workerId": "worker-1",
  "batch": 7,
  "steps": [
    {
      "step": 1,
      "deliveryId": "del-1",
      "outcome": "delivered",
      "attempts": 1,
      "version": 1,
      "failure": { "identity": "ok", "occurrence": "" }
    }
  ],
  "batchError": { "identity": "store-unavailable", "occurrence": "claim" }
}
```

- `step` is the 1-based claim-order index inside the batch,
  mirroring C-A step-indexed diagnostics; `outcome` is one of
  `delivered` | `failed` (acked for retry) | `dead` (terminal) |
  `error` (left leased for redelivery).
- `failure.identity` is finite: `ok`, `downstream-status`,
  `downstream-transport`, `downstream-timeout`,
  `credential-missing`, `redirect-denied`, `redirect-hops`,
  `poison`, `destination-denied`, `no-destination`, `lease-lost`,
  `protocol-error`, `store-unavailable`. `occurrence` carries
  the status code or policy reason, never a secret or URL.
- `batchError` appears only when claim paging itself failed;
  already-held rows still deliver in that batch.

F06 qualifies this mirror (two companions, concurrency, poison,
crash/redelivery, measured batch behavior) plus W4-batch.

## 6. Honest limits (unchanged unless separately fixed)

Signature timing, 1000-header scan, SQLite transaction shape,
unforceable local `commit_unknown`, single-claim paging, coarse
worker backoff, redirect re-post semantics, credential-error
retry-then-poison, report redaction, and the DNS-recheck TOCTOU
window are all documented in `README.md` and hold for v1.

## 7. Qualification pointers

- `canlc assert examples/webhook` — 183 roots pin every
  auth/shape/lease/ack/dead-letter/reconcile arm.
- `bun test examples/webhook/companion/` — 34 unit tests of the
  protocol client, worker, and supervisor against a verifying
  stub Can.
- `TestWebhookSliceLive` — the staged live pair: provider legs,
  carrier auth matrix, replay, lease/heartbeat, poison, crash
  and redelivery legs, in-flight convergence, batch pair.
