-- webhook slice schema (F05 authenticated companion pair). Can descriptors
-- admit only SELECT and RETURNING-free INSERT/UPDATE/DELETE, so this DDL
-- is applied by the operator (or the test harness) before first serve,
-- never from Can.
--
-- webhook_outbox carries the B3 lease: worker_id names the holding
-- companion ('' when unclaimed), lease_until_ms is the ms-epoch expiry
-- (0 when unclaimed), and version guards every lease mutation against
-- lost updates. No NULLs: sentinels keep every Can record total.
-- webhook_dead_letter holds terminal poison rows; the outbox row stays
-- with state 'dead' so replays answer consistently.
-- webhook_carrier_nonce holds single-use carrier nonces; claim prunes
-- rows older than twice the timestamp skew window.
CREATE TABLE IF NOT EXISTS webhook_ledger (
    delivery_id TEXT PRIMARY KEY,
    digest TEXT NOT NULL,
    event TEXT NOT NULL,
    subscription TEXT NOT NULL,
    outcome TEXT NOT NULL,
    received_ms INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS webhook_outbox (
    delivery_id TEXT PRIMARY KEY REFERENCES webhook_ledger (delivery_id),
    event TEXT NOT NULL,
    subscription TEXT NOT NULL,
    state TEXT NOT NULL,
    attempts INTEGER NOT NULL,
    updated_ms INTEGER NOT NULL,
    worker_id TEXT NOT NULL DEFAULT '',
    lease_until_ms INTEGER NOT NULL DEFAULT 0,
    version INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE IF NOT EXISTS webhook_attempt (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    delivery_id TEXT NOT NULL REFERENCES webhook_outbox (delivery_id),
    outcome TEXT NOT NULL,
    at_ms INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS webhook_dead_letter (
    delivery_id TEXT PRIMARY KEY REFERENCES webhook_outbox (delivery_id),
    event TEXT NOT NULL,
    subscription TEXT NOT NULL,
    reason TEXT NOT NULL,
    attempts INTEGER NOT NULL,
    dead_ms INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS webhook_carrier_nonce (
    nonce TEXT PRIMARY KEY,
    seen_ms INTEGER NOT NULL
);
