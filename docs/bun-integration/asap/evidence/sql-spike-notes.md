# G-SQL backend spikes (B1-02.03)

Selections advised by three unanimous Jev rounds
(`consultations-b1-02/decision-audit.md`): tidb/parser for MySQL,
defin/tree-sitter-sqlite3 for SQLite. Both spikes below ran the
44-case corpus at `sql-corpus.json` (13 postgresql, 18 sqlite, 13
mysql) plus structural extras. PostgreSQL stays on libpg_query via
pg_query_go/v6 v6.2.2 (major 17, version 170007).

## Pins

| Artifact | Pin | SHA256 / hash |
|---|---|---|
| github.com/pingcap/tidb/pkg/parser | v0.0.0-20260923074046-bc2f3a5f6759 | module hash bc2f3a5f6759 |
| tree-sitter runtime (libtree-sitter) | github.com/tree-sitter/tree-sitter v0.27.0 | go module tag |
| defin/tree-sitter-sqlite3 | main 7f69bb66845b (2026-05-03), tracks SQLite 3.47.0, LANGUAGE_VERSION 15 | parser.c 99a9cf64e174…, scanner.c 3b9cb55a3b20…, tree_sitter/parser.h 180b893c8734… |
| Differential engine | Bun 1.4.2 bun:sqlite, SQLite 3.54.0, :memory: | `sqlite-differential-results.json` |

## MySQL spike: tidb/parser (pure Go)

`parser.New().Parse(sql, "", "")` returns statement nodes; parameter
sites come from the concrete `*testdriver.ParamMarkerExpr.Offset`
field (byte offset of `?`; the embedded `OriginTextPosition` stays 0
for LIMIT markers, so the interface method is the wrong read).
`SelectStmt.Limit` exposes `Count`/`Offset` expressions; INSERT,
UPDATE, and DELETE expose `Returning []*SelectField`. SQL mode lives
in the lexer; the spike ran the default mode.

Verdicts over all 13 my-* cases plus span verification (every reported
offset points at a literal `?` byte): my-01/02/03/04/09/11 valid;
my-05 two statements; my-07 no LIMIT; my-08 LIMIT with OFFSET set;
my-10 three sites over a total of two; my-12 UPDATE under cardinality
one; my-13 grammar error. Backslash escapes (my-04) and `#`/`--`/`/**/`
comments (my-11) swallow decoy marks.

Reconciliation: my-06 parses because TiDB's grammar carries
`ReturningClause` on UPDATE/DELETE (TiDB proper rejects it in
planning; real MySQL in prepare). The backend reports `returning`
structurally and Can rejects it; the corpus expectation moved from
`syntax` to `returning` since the never-admitted property holds
either way. MySQL engine differential runs in B1-03 with the live
service; nothing here claims MySQL-server parity beyond the grammar.

## SQLite spike: tree-sitter parse.y mirror (vendored C + cgo)

Runtime amalgamation `lib.c` plus grammar `parser.c`/`scanner.c`
compile warning-free under clang -O1 and link into a cgo probe. The
concrete syntax tree delivers every contract input: `bind_parameter`
nodes with exact byte spans for `?`, `?NNN`, and named spellings;
`limit_clause` subtrees distinguishing bare, literal, `,`, and
`OFFSET` forms; `returning_clause` presence; distinct top-level kinds
(`select_statement`, `insert_statement`, `update_statement`,
`delete_statement`, plus `pragma_statement`, `explain_statement`,
`create_table_statement` for structural `bad_kind` rejection);
`has_error` with `ERROR` node spans for grammar failures; one
trailing `;` tolerated as an anonymous child.

Verdicts over all 18 lite-* cases: lite-01/02/03/04/09/11/12/13/18
valid with exact spans (comments, quoted semicolons, doubled-quote
escapes, double-quoted identifiers, and `$1`-in-literal decoys all
correct); lite-05 three top-level children; lite-06 returning clause;
lite-07 missing limit; lite-08 literal limit; lite-10 `?4` site text
exposed for number mapping; lite-14/15 two limit sites; lite-16/17
`has_error=1` with spans.

Syntax differential: bun:sqlite prepare over a fixture schema accepts
exactly the 15 parseable single statements plus the 3 kind extras and
rejects exactly lite-16/17 (multi-statement lite-05 excluded: it is a
contract dimension, already structural). Backend and engine agree on
all 20 differential rows despite the grammar tracking 3.47.0 against
engine 3.54.0.

## Rejected with evidence

Engine-API prepare (no spans, no LIMIT siting, PRAGMA/EXPLAIN kind
confusion, shared SQLITE_ERROR forcing message matching); live-server
MySQL prepare (non-hermetic, gate step 4); ANTLR (runner-up 0.00-0.01,
JVM regeneration absent); vitess (runner-up 0.00, module weight plus
unverified offsets); regex/splitter, cross-feeding libpg_query,
translate-to-PostgreSQL, hand subset grammars, compile-time Bun
shelling (forbidden by gate/D03). SQLite `?NNN`/named-to-number
mapping arithmetic is fixed in B1-02.04; the corpus pins acceptance.
