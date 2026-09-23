# B1-02.01 three-dialect SQL contract table

Can-side scalar cover is authoritative and dialect-neutral
(`compiler/internal/types/sql_schema.go`): `bool`, `int` (signed 64),
`float` (finite), `str`, `bytes`, or one `option` layer of these.
Arrays, JSON, decimals, timestamps, and nested options are rejected at
check time on every dialect. Each cell below names its provenance:
**code** (shipped implementation), **probe** (executed Bun 1.4.2
observation), **docs** ([Bun SQL docs](https://bun.sh/docs/runtime/sql)),
or **qualify** (acceptance probe still required, with owning step).

## Placeholders and binding

| Aspect | postgres | sqlite | mysql |
|---|---|---|---|
| Source spelling in descriptors | `$N` contiguous, scanner `PARAM` (code) | `?`, `?NNN`, `:name`, `@name`, `$name` (tree-sitter backend, B1-02.03 exit) | `?` positional (qualify: G-SQL backend, B1-03; Bun auto-converts `$1` per docs, but that is a fallback, not the contract) |
| Native binding | positional template values; static text carries no markers (code) | same mechanism (code) | same mechanism (code) |
| Row-limit parameter | top-level `LIMIT $M`, exactly once (code) | top-level `LIMIT ?` equivalent (qualify: B1-02.04, never pg syntax appended blindly) | same as sqlite (qualify: B1-02.04) |
| Repeated / out-of-order sites | expand structurally to one value per number (code) | same (code) | same (code) |

## Value mappings (Can field kind -> native -> Can field kind)

| Can kind | postgres | sqlite | mysql |
|---|---|---|---|
| `bool` | JS boolean both ways (code) | `INTEGER` 0/1; encode bool as 0/1, decode 0/1 only (qualify: probe, B1-02.05) | `TINYINT(1)` decodes to JS number per docs, `BIT(1)` to boolean; bool-field decode of 0/1 numbers and boolean encoding (qualify: probe, B1-03) |
| `int` | `bigint:true`; bigint exact, safe-integer number widened, anything else is `unsafe_integer` mismatch, never a silent round (code) | `safeIntegers:true` REQUIRED: probe rounded 9007199254740993 to ...992 without it and preserved `9007199254740993n` with it (probe) | `BIGINT`: number in i32/u32 range, else string or bigint per `bigint` option (docs); `bigint:true` intended; unsafe shapes fail closed via the same mismatch path (qualify: probe, B1-03) |
| `float` | finite number both ways; nonfinite rejected (code) | `REAL` number; same finiteness rule (qualify: probe, B1-02.05) | `FLOAT`/`DOUBLE` number (docs); same finiteness rule (qualify: probe, B1-03) |
| `str` | well-formed string both ways (code) | `TEXT` string; same well-formedness rule (qualify: probe, B1-02.05) | `CHAR`/`VARCHAR`/`TEXT`, `utf8mb4` (docs); same rule (qualify: probe, B1-03) |
| `bytes` | `bytea`; copied at the boundary (code) | `BLOB`; decode shape (`Buffer` vs `Uint8Array`) by probe (qualify: B1-02.05) | `BLOB` with binary charset decodes to `Buffer` (docs); copied at the boundary (qualify: probe, B1-03) |
| `option none` / `NULL` | none encodes to `NULL`; `NULL` decodes to none, or `null` mismatch on a non-option field (code) | same rule (code path shared) | same rule (code path shared) |
| decimals | REJECTED at check: no cover (code) | same (code) | same (code); native `DECIMAL` decodes to string (docs) and is never admitted |
| dates / times | REJECTED at check: no cover (code) | same (code) | same (code); native `DATE`/`DATETIME`/`TIME` shapes (docs) are never admitted |

## Statement metadata

| Aspect | postgres | sqlite | mysql |
|---|---|---|---|
| affected rows | `result.count`, safe non-negative integer, else `bad_count` (code) | native shape by probe (`changes`/`count`/`lastInsertRowid` candidates); exact int or `bad_count` (qualify: B1-02.05) | `result.affectedRows` (docs); exact int or `bad_count` (qualify: probe, B1-03) |
| insert id | not offered: `RETURNING` is forbidden and pg has no connection-scoped id under current restrictions (code) | `lastInsertRowid` candidate (qualify: probe, B1-02.05) | `result.lastInsertRowid` = `LAST_INSERT_ID()` (docs) (qualify: probe, B1-03) |

`execute` keeps returning `int` affected rows on every dialect (shipped
I35 contract, not reopened here). The plan's "optional insert_id" cannot
share that result shape and has no postgres meaning, so its surface
(separate op vs per-dialect failure) is decided in B1-02.05 once the
sqlite probe fixes the native shape. No silent casts anywhere: every
loss-prone conversion is a `schema_mismatch`/`unsupported_value`
failure, and every rejected mapping above is a compile-time descriptor
error, not a runtime surprise.

## B1-02.04 SQLite number/name mapping rule

Sites map to numbers by engine rules: `?NNN` binds NNN explicitly, a
bare `?` takes one past the current maximum, and each distinct
`:name`/`@name`/`$name` takes the next number with repeats sharing it
(sigils never distinguish names). The contiguity, trailing-limit, and
coverage rules then apply to numbers exactly as on PostgreSQL.
Additionally, a named site binding an application number must spell
that number's declared parameter exactly (typo detection); bare and
`?NNN` sites have no name to check and the row-limit site's name is
free. `$` names with a leading digit are grammar errors, matching the
engine. Corpus cases lite-03/10/13/19/20/21 pin the rule.
