// R14 ledger SQL schema for F04 backend qualification (F01).
//
// Operator-owned DDL per dialect: one version row for optimistic
// whole-state commits, append-only schedule versions keyed by
// (pool, anchor_ms), pinned epoch rows, invocation records, and
// quarantined profiles. Admission reads the newest schedule version
// whose anchor the ledger clock has reached. Counters are
// nonnegative integers under CHECK constraints; states are closed
// string sets. Portability notes:
//
// - SQLite: INTEGER is 64-bit; TEXT keys need no length. CHECK
//   constraints enforced.
// - PostgreSQL: BIGINT counters; TEXT keys need no length. CHECK
//   constraints enforced.
// - MySQL 8.0.16+: CHECK constraints enforced from 8.0.16 (older
//   servers parse and ignore them — do not qualify below 8.0.16).
//   TEXT cannot key an index, so key columns are VARCHAR(191),
//   which fits the 767-byte utf8mb4 prefix limit. Non-key columns
//   with identity values above the key budget (metering
//   provider/model/version at 256 chars each, quarantine display
//   identity) are TEXT: VARCHAR(191) there would reject
//   contract-valid names under strict mode (error 1406) while
//   PG/SQLite accept them. The quarantine key itself is a fixed
//   64-hex-char digest (see profileKey), so it fits VARCHAR(191).
//
// Atomic-op mapping (short native transactions, never held across
// provider I/O): each reserve/fence/settle/release runs as one
// transaction that reads its rows, then commits with
// `UPDATE ai_budget_state SET version = version + 1 WHERE version = ?`.
// Zero affected rows means a lost race: roll back and retry from a
// fresh read. A commit failure after the write is commit-unknown and
// reconciles by invocation ID. F04 qualifies this mapping live per
// dialect; until then the memory and file backends carry the
// executed evidence.
export type LedgerDialect = "sqlite" | "postgres" | "mysql";

const SQLITE_STATEMENTS: readonly string[] = Object.freeze([
  `CREATE TABLE ai_budget_state (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    version INTEGER NOT NULL CHECK (version >= 0)
  )`,
  `INSERT INTO ai_budget_state (id, version) VALUES (1, 0)`,
  `CREATE TABLE ai_budget_schedules (
    pool TEXT NOT NULL,
    token_limit INTEGER NOT NULL CHECK (token_limit >= 0),
    period_ms INTEGER NOT NULL CHECK (period_ms > 0),
    anchor_ms INTEGER NOT NULL CHECK (anchor_ms >= 0),
    version TEXT NOT NULL,
    PRIMARY KEY (pool, anchor_ms)
  )`,
  `CREATE TABLE ai_budget_epochs (
    tenant TEXT NOT NULL,
    pool TEXT NOT NULL,
    epoch_start INTEGER NOT NULL CHECK (epoch_start >= 0),
    epoch_end INTEGER NOT NULL,
    token_limit INTEGER NOT NULL CHECK (token_limit >= 0),
    schedule_version TEXT NOT NULL,
    committed INTEGER NOT NULL CHECK (committed >= 0),
    held INTEGER NOT NULL CHECK (held >= 0),
    PRIMARY KEY (tenant, pool, epoch_start)
  )`,
  `CREATE TABLE ai_budget_invocations (
    invocation_id TEXT PRIMARY KEY,
    tenant TEXT NOT NULL,
    pool TEXT NOT NULL,
    correlation TEXT NOT NULL,
    epoch_start INTEGER NOT NULL,
    epoch_end INTEGER NOT NULL,
    token_limit INTEGER NOT NULL,
    schedule_version TEXT NOT NULL,
    provider TEXT NOT NULL,
    model TEXT NOT NULL,
    profile_version TEXT NOT NULL,
    upper_bound INTEGER NOT NULL CHECK (upper_bound > 0),
    state TEXT NOT NULL CHECK (state IN ('held', 'fenced', 'settled', 'released')),
    fenced INTEGER NOT NULL CHECK (fenced IN (0, 1)),
    input_tokens INTEGER CHECK (input_tokens IS NULL OR input_tokens >= 0),
    output_tokens INTEGER CHECK (output_tokens IS NULL OR output_tokens >= 0),
    breach INTEGER NOT NULL CHECK (breach IN (0, 1))
  )`,
  `CREATE TABLE ai_budget_quarantine (
    profile_key TEXT PRIMARY KEY,
    profile_identity TEXT NOT NULL
  )`,
  `CREATE INDEX ai_budget_invocations_epoch
    ON ai_budget_invocations (tenant, pool, epoch_start, state)`,
]);

const POSTGRES_STATEMENTS: readonly string[] = Object.freeze([
  `CREATE TABLE ai_budget_state (
    id BIGINT PRIMARY KEY CHECK (id = 1),
    version BIGINT NOT NULL CHECK (version >= 0)
  )`,
  `INSERT INTO ai_budget_state (id, version) VALUES (1, 0)`,
  `CREATE TABLE ai_budget_schedules (
    pool TEXT NOT NULL,
    token_limit BIGINT NOT NULL CHECK (token_limit >= 0),
    period_ms BIGINT NOT NULL CHECK (period_ms > 0),
    anchor_ms BIGINT NOT NULL CHECK (anchor_ms >= 0),
    version TEXT NOT NULL,
    PRIMARY KEY (pool, anchor_ms)
  )`,
  `CREATE TABLE ai_budget_epochs (
    tenant TEXT NOT NULL,
    pool TEXT NOT NULL,
    epoch_start BIGINT NOT NULL CHECK (epoch_start >= 0),
    epoch_end BIGINT NOT NULL,
    token_limit BIGINT NOT NULL CHECK (token_limit >= 0),
    schedule_version TEXT NOT NULL,
    committed BIGINT NOT NULL CHECK (committed >= 0),
    held BIGINT NOT NULL CHECK (held >= 0),
    PRIMARY KEY (tenant, pool, epoch_start)
  )`,
  `CREATE TABLE ai_budget_invocations (
    invocation_id TEXT PRIMARY KEY,
    tenant TEXT NOT NULL,
    pool TEXT NOT NULL,
    correlation TEXT NOT NULL,
    epoch_start BIGINT NOT NULL,
    epoch_end BIGINT NOT NULL,
    token_limit BIGINT NOT NULL,
    schedule_version TEXT NOT NULL,
    provider TEXT NOT NULL,
    model TEXT NOT NULL,
    profile_version TEXT NOT NULL,
    upper_bound BIGINT NOT NULL CHECK (upper_bound > 0),
    state TEXT NOT NULL CHECK (state IN ('held', 'fenced', 'settled', 'released')),
    fenced INTEGER NOT NULL CHECK (fenced IN (0, 1)),
    input_tokens BIGINT CHECK (input_tokens IS NULL OR input_tokens >= 0),
    output_tokens BIGINT CHECK (output_tokens IS NULL OR output_tokens >= 0),
    breach INTEGER NOT NULL CHECK (breach IN (0, 1))
  )`,
  `CREATE TABLE ai_budget_quarantine (
    profile_key TEXT PRIMARY KEY,
    profile_identity TEXT NOT NULL
  )`,
  `CREATE INDEX ai_budget_invocations_epoch
    ON ai_budget_invocations (tenant, pool, epoch_start, state)`,
]);

const MYSQL_STATEMENTS: readonly string[] = Object.freeze([
  `CREATE TABLE ai_budget_state (
    id BIGINT PRIMARY KEY CHECK (id = 1),
    version BIGINT NOT NULL CHECK (version >= 0)
  )`,
  `INSERT INTO ai_budget_state (id, version) VALUES (1, 0)`,
  `CREATE TABLE ai_budget_schedules (
    pool VARCHAR(191) NOT NULL,
    token_limit BIGINT NOT NULL CHECK (token_limit >= 0),
    period_ms BIGINT NOT NULL CHECK (period_ms > 0),
    anchor_ms BIGINT NOT NULL CHECK (anchor_ms >= 0),
    version VARCHAR(191) NOT NULL,
    PRIMARY KEY (pool, anchor_ms)
  )`,
  `CREATE TABLE ai_budget_epochs (
    tenant VARCHAR(191) NOT NULL,
    pool VARCHAR(191) NOT NULL,
    epoch_start BIGINT NOT NULL CHECK (epoch_start >= 0),
    epoch_end BIGINT NOT NULL,
    token_limit BIGINT NOT NULL CHECK (token_limit >= 0),
    schedule_version VARCHAR(191) NOT NULL,
    committed BIGINT NOT NULL CHECK (committed >= 0),
    held BIGINT NOT NULL CHECK (held >= 0),
    PRIMARY KEY (tenant, pool, epoch_start)
  )`,
  `CREATE TABLE ai_budget_invocations (
    invocation_id VARCHAR(191) PRIMARY KEY,
    tenant VARCHAR(191) NOT NULL,
    pool VARCHAR(191) NOT NULL,
    correlation VARCHAR(191) NOT NULL,
    epoch_start BIGINT NOT NULL,
    epoch_end BIGINT NOT NULL,
    token_limit BIGINT NOT NULL,
    schedule_version VARCHAR(191) NOT NULL,
    provider TEXT NOT NULL,
    model TEXT NOT NULL,
    profile_version TEXT NOT NULL,
    upper_bound BIGINT NOT NULL CHECK (upper_bound > 0),
    state VARCHAR(16) NOT NULL CHECK (state IN ('held', 'fenced', 'settled', 'released')),
    fenced INTEGER NOT NULL CHECK (fenced IN (0, 1)),
    input_tokens BIGINT CHECK (input_tokens IS NULL OR input_tokens >= 0),
    output_tokens BIGINT CHECK (output_tokens IS NULL OR output_tokens >= 0),
    breach INTEGER NOT NULL CHECK (breach IN (0, 1))
  )`,
  `CREATE TABLE ai_budget_quarantine (
    profile_key VARCHAR(191) PRIMARY KEY,
    profile_identity TEXT NOT NULL
  )`,
  `CREATE INDEX ai_budget_invocations_epoch
    ON ai_budget_invocations (tenant, pool, epoch_start, state)`,
]);

export const LEDGER_STATEMENTS: Readonly<Record<LedgerDialect, readonly string[]>> = Object.freeze({
  sqlite: SQLITE_STATEMENTS,
  postgres: POSTGRES_STATEMENTS,
  mysql: MYSQL_STATEMENTS,
});

// ledgerDDL renders the operator-owned schema script for one dialect.
export function ledgerDDL(dialect: LedgerDialect): string {
  const statements = LEDGER_STATEMENTS[dialect];
  if (statements === undefined) throw new TypeError("invalid ledger dialect");
  return statements.map((statement) => `${statement};`).join("\n");
}
