-- webhook slice schema (T16). Can descriptors admit only SELECT and
-- RETURNING-free INSERT/UPDATE/DELETE, so this DDL is applied by the
-- operator (or the test harness) before first serve, never from Can.
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
    updated_ms INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS webhook_attempt (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    delivery_id TEXT NOT NULL REFERENCES webhook_outbox (delivery_id),
    outcome TEXT NOT NULL,
    at_ms INTEGER NOT NULL
);
