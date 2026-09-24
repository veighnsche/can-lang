-- invoice slice schema (T14). Can descriptors admit only SELECT and
-- RETURNING-free INSERT/UPDATE/DELETE, so this DDL is applied by the
-- operator (or the test harness) before first serve, never from Can.
CREATE TABLE IF NOT EXISTS invoice_session (
    token TEXT PRIMARY KEY,
    actor TEXT NOT NULL,
    revoked_ms INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE IF NOT EXISTS invoice_membership (
    actor TEXT NOT NULL,
    tenant TEXT NOT NULL,
    PRIMARY KEY (actor, tenant)
);
CREATE TABLE IF NOT EXISTS invoice (
    id TEXT PRIMARY KEY,
    tenant TEXT NOT NULL,
    revision INTEGER NOT NULL,
    customer TEXT NOT NULL,
    updated_ms INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS invoice_line (
    invoice_id TEXT NOT NULL REFERENCES invoice (id),
    line_key TEXT NOT NULL,
    sku TEXT NOT NULL,
    qty INTEGER NOT NULL,
    position INTEGER NOT NULL,
    PRIMARY KEY (invoice_id, line_key)
);
CREATE TABLE IF NOT EXISTS invoice_replay (
    actor TEXT NOT NULL,
    invoice_id TEXT NOT NULL,
    operation_id TEXT NOT NULL,
    digest TEXT NOT NULL,
    revision INTEGER NOT NULL,
    recorded_ms INTEGER NOT NULL,
    PRIMARY KEY (actor, invoice_id, operation_id)
);
