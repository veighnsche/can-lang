# B1-03 MySQL contract confirmation

Service: MySQL 8.4.11 (docker, disposable), InnoDB, sql_mode
`ONLY_FULL_GROUP_BY,STRICT_TRANS_TABLES,NO_ZERO_IN_DATE,NO_ZERO_DATE,ERROR_FOR_DIVISION_BY_ZERO,NO_ENGINE_SUBSTITUTION`.
Driver: Bun 1.4.2 `Bun.SQL` mysql adapter. Every row below is an
executed observation unless marked docs. B1-02.01 table cells marked
`qualify: probe, B1-03` resolve here; nothing below redefines the
shared Can scalar cover (`bool`, `int` signed 64, finite `float`,
`str`, `bytes`, one `option` layer).

## Connection

| Aspect | Contract |
|---|---|
| Open | `sql::mysql_open(connection_variable, max_connections)`; URL from env, never logged |
| Wire options | `bigint: true`, `max` validated 1..2^31-1, `tls: true` forced |
| TLS | `TLS_AES_128_GCM_SHA256` / TLSv1.3 observed; a self-signed server connects, so this is encryption without server-chain verification (recorded limitation) |
| Plaintext | fails closed: `ERR_MYSQL_PUBLIC_KEY_RETRIEVAL_NOT_ALLOWED` (caching_sha2_password refuses key retrieval over insecure connections) |
| Refused host | `MySQLError` / `ERR_MYSQL_CONNECTION_REFUSED`, no errno |
| Bad credentials | `MySQLError` errno 1045 at connect |
| Unknown database | `MySQLError` errno 1044 at connect |
| Session | the driver pins every session to `+00:00` itself (observed under all client timezones and server defaults, on pooled and reconnected handles alike, overriding the server default); the open verifies the pin and refuses the pool on drift. A per-connection `SET` cannot broadcast (verified: one pooled handle in five took it), and Can descriptors cannot change it (`SET` is outside the DML grammar), so the driver pin is the only sound zone and the B1-03.04 timezone matrix collapses to it |

## Placeholders and binding

`?` positional only, numbered 1..M in source order (tidb offsets,
each verified to point at a `?` byte). Bound `bigint` values round-trip
exactly including ±2^63 extremes and unsigned 2^64-1. `LIMIT ?` binds.
Server-side prepares persist across same-shape calls (observed
`Prepared_stmt_count` 3 after repeats), so descriptors reuse native
prepares. Static text carries no markers; dynamic identifiers have no
checked operation and are unsupported (identifiers stay literal text).

## Value mappings (wire -> Can field kind)

| Wire | Can kind | Rule |
|---|---|---|
| `TINYINT(1)`/`BOOLEAN` number 0/1, `BIT(1)` boolean | `bool` | native boolean plus exact 0/1 numbers/bigints; anything else mismatches |
| `BIT(n>1)` Buffer | `bytes` | owned copy (`Buffer` is a `Uint8Array`) |
| integer number (safe), bigint (int64), canonical digit string | `int` | exact; unsafe numbers, out-of-range bigints, and unsigned values above int64 reject as `int_range`/`unsafe_integer` |
| `DECIMAL` string (e.g. `12345678901234567890.1234567890`) | `str` | exact decimal text preserved verbatim; never coerced to float. Can has no decimal type, so the validated string is the explicit exact representation |
| `DATETIME`/`TIMESTAMP` Date | `str` | naive UTC `YYYY-MM-DD HH:MM:SS[.mmm]` under the pinned session; `Invalid Date` mismatches. The driver truncates sub-millisecond fractions (observed `.123456` -> `.123`); no date-as-string option exists, so this is a recorded precision limit |
| `FLOAT`/`DOUBLE` number | `float` | finite only, same rule as all dialects |
| `CHAR`/`VARCHAR`/`TEXT`/`ENUM`/`SET`/`TIME` string | `str` | well-formedness rule, same as all dialects |
| `YEAR` number | `int` | safe-integer rule |
| `BINARY`/`BLOB` Buffer | `bytes` | owned copy, padding preserved |
| `JSON` parsed object | — | rejected: `str` (and every other kind) mismatches non-string, non-Date objects |
| `NULL` | option none | shared rule; `null` mismatch on non-option fields |

DECIMAL/TIME binding: Can `str` binds as string; the strict server
parses `DATETIME` text and rejects malformed values. `bool` binds to
`BOOLEAN`/`BIT(1)`; `bytes` binds to binary columns exactly.

## Affected rows and insert IDs

`execute` reads `affectedRows` (observed: insert 1, multi-row update 2,
delete 2, no-op update 0); the `count` field stays 0 on MySQL and is
never used. `lastInsertRowid` carries the auto-increment id (observed 1
after an auto insert, 0 after an explicit-id insert). `sql::execute`
returns the affected count only, identically on all three engines, so
generated ids are read back with `SELECT LAST_INSERT_ID()` as a typed
`query_one<int>` descriptor (covered by the live tests).

## Failure classification (all MySQLError)

| Native | Can |
|---|---|
| `ERR_MYSQL_CONNECTION_REFUSED` / `ERR_MYSQL_CONNECTION_CLOSED` | `sql::connection_failed` phase `connect` (open) or `query` |
| errno 1045 / 1044 | `sql::connection_failed` phase `connect` (open) or `auth` (query path) |
| errno 1062 | `sql::constraint_failed` constraint `ER_DUP_ENTRY` |
| errno 1064 / 1146 | `sql::query_failed` code `ER_PARSE_ERROR` / `ER_NO_SUCH_TABLE` |
| errno 1205 / 1213 | `sql::query_failed` code `ER_LOCK_WAIT_TIMEOUT` / `ER_LOCK_DEADLOCK` (verified by the contention tests) |
| other errno N | `sql::query_failed` code `mysql_errno_N` |
| no errno | `sql::query_failed` with the Bun code, else `unknown` |
| non-MySQL throw | standard failure (driver defect), never a generic mysql error |

Deadlock inside a transaction body surfaces as the verbatim
`query_failed` primary and Bun rolls the attempt back; a rejection
after a commit decision reports `sql::commit_unknown` with the safe
attempt id, never retried. MySQL aborts a failed COMMIT on its own, so
no cleanup rollback runs (unlike SQLite).

## Grammar

Descriptors validate through the pinned tidb parser (default sql_mode,
matching the server default for every lexer-relevant bit) before any
native call. `WITH`/CTE, backtick identifiers, backslash escapes, and
`#`/`--`/block comments follow the G-SQL corpus (48/48 with the
13 mysql rows). `RETURNING` on mutations parses structurally and Can
rejects it. The live differential replays every mysql corpus row
through a real server-side `EXPLAIN` (SQL-level `PREPARE` is unusable:
Bun sends every statement through the binary protocol, where the
server answers `PREPARE` with errno 1295): 12 rows agree directly and
my-06 reconciles (tidb-accept plus the Can `returning` rejection
against the server's 1064); see `mysql-differential-results.json`.
