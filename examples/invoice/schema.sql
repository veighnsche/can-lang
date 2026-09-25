-- invoice slice schema (UP16). Can descriptors admit only SELECT and
-- RETURNING-free INSERT/UPDATE/DELETE, so this DDL is applied by the
-- operator (or the test harness) before first serve, never from Can.
-- Tenant and invoice keys are integers; money is integer minor units;
-- the replay ledger keeps the canonical digest plus the committed
-- neutral result until the retention window expires it.
CREATE TABLE IF NOT EXISTS invoice_session (
    token TEXT PRIMARY KEY,
    actor TEXT NOT NULL,
    revoked_ms INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE IF NOT EXISTS invoice_membership (
    actor TEXT NOT NULL,
    tenant INTEGER NOT NULL,
    PRIMARY KEY (actor, tenant)
);
CREATE TABLE IF NOT EXISTS invoice (
    tenant INTEGER NOT NULL,
    id INTEGER NOT NULL,
    revision INTEGER NOT NULL,
    seats INTEGER NOT NULL,
    details TEXT NOT NULL DEFAULT '',
    updated_ms INTEGER NOT NULL,
    PRIMARY KEY (id)
);
CREATE TABLE IF NOT EXISTS invoice_line (
    invoice_id INTEGER NOT NULL,
    line_key TEXT NOT NULL,
    id TEXT NOT NULL,
    quantity INTEGER NOT NULL,
    price_minor INTEGER NOT NULL,
    position INTEGER NOT NULL,
    PRIMARY KEY (invoice_id, line_key)
);
CREATE TABLE IF NOT EXISTS invoice_replay (
    actor TEXT NOT NULL,
    tenant INTEGER NOT NULL,
    invoice_id INTEGER NOT NULL,
    operation_id TEXT NOT NULL,
    digest TEXT NOT NULL,
    result TEXT NOT NULL,
    revision INTEGER NOT NULL,
    recorded_ms INTEGER NOT NULL,
    PRIMARY KEY (actor, tenant, invoice_id, operation_id)
);
