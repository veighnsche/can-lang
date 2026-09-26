# F03 — blessed encoding guide (2026-09-26)

Source: R10; F-R10-02 (O1: retain app-level encodings + blessed guide).
No new column kinds: the three encodings below already cover money,
time, and JSON in the invoice example, and every rule here is qualified
live through the runtime on PG 17.11
([report](f03-live-report.json) `encodings` leg,
`tests/integration/sql_f03_test.go`). If a future app demonstrates a
shape these encodings cannot carry, that insufficiency returns to
preparation — it does not license new column kinds on its own.

## Money: minor-unit `int` in `BIGINT`

- Store money as integer minor units (`price_minor BIGINT NOT NULL`):
  `$10.99` is `1099`. Never floats, never strings.
- Live: `1099` roundtrips exactly (`encodings.minor_unit`).
- Bounds: the full `int64` range roundtrips (`9223372036854775807` and
  `-9223372036854775808` pinned in `encodings.int64_max/min`). Sizing
  past `int64` minor units is out of scope; demonstrate need first.

## Time: ms-since-epoch `int` in `BIGINT`

- Store instants as milliseconds since the Unix epoch
  (`recorded_ms BIGINT NOT NULL`), produced/consumed at the app layer
  (`clock::wall_millis` in the invoice example). Never timestamp
  columns, never strings.
- Live: `1727222400000` roundtrips exactly (`encodings.ms_epoch`).
- Nullable audit instants are `BIGINT` columns decoded into
  `option<int>`: SQL `NULL` ↔ `none`, value ↔ `some(value)` (both
  directions pinned: `encodings.null_audit_is_none/set_audit`). A `NULL`
  into a non-`option` field is `sql::schema_mismatch` (`null`) — the
  decoder does not guess.

## JSON: opaque `TEXT` + app `codec`

- Store documents as opaque `TEXT` (`result TEXT NOT NULL`); structure
  lives in the app's `codec` layer, never in a JSON column kind.
- Live: a nested document with unicode, quotes, and nesting roundtrips
  byte-exact through store → refetch → parse
  (`encodings.json_codec`).

## Portability notes

- All three encodings are dialect-flat: `BIGINT`/`TEXT` exist
  identically on PG, MySQL, and SQLite, so the F04 parity legs carry no
  encoding risk from these shapes.
- IDs: app-supplied keys stay `TEXT`/`BIGINT` primary keys (the
  lookup-first recipe's `delivery_id TEXT PRIMARY KEY`); generated
  identities are `BIGINT GENERATED ALWAYS AS IDENTITY` on PG,
  `AUTO_INCREMENT` on MySQL, `INTEGER PRIMARY KEY AUTOINCREMENT` on
  SQLite — each returned per the
  [RETURNING contract](f03-returning-contract.md) (§2/§5).
