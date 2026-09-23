# G-SQL decision audit (B1-02.03)

Three independently rewritten packets asked two identical questions over
the 37-case corpus and the backend comparison evidence. All prose
(state, instructions, criteria) differs across packets with zero shared
sentences after masking stable identifiers (verified mechanically);
question keys, option keys, identifiers, versions, and measured numbers
are stable. Model: `jev-1.13.0` via `jev-latest`, three HTTP 200 rounds.

## Results

| Question | Round 1 | Round 2 | Round 3 |
|---|---|---|---|
| mysql_backend | tidb_parser 1.00 (conf 1.00) | tidb_parser 1.00 (conf 1.00) | tidb_parser 1.00 (conf 1.00) |
| sqlite_backend | treesitter_mirror 0.99 (conf 0.99) | treesitter_mirror 1.00 (conf 1.00) | treesitter_mirror 0.98 (conf 0.98) |

Requests, responses and SHA metadata: `request-N.json`,
`response-N.json`, `response-N.metadata.json` in this directory.

## Agreement (accepted as advice, spike confirmation required)

- MySQL `tidb_parser` (1.00 x3): standalone module
  `github.com/pingcap/tidb/pkg/parser` at v0.0.0-20260923074046,
  Apache-2.0, `Parse` returning statement nodes with
  `OriginTextPosition` and offset-bearing parameter markers, SQL mode
  in the lexer. Runners-up got 0.00 in every round.
- SQLite `treesitter_mirror` (0.99/1.00/0.98):
  defin/tree-sitter-sqlite3, the rule-by-rule parse.y/tokenize.c
  mirror tracking SQLite 3.47.0, blessing-licensed, consumed as
  committed C through cgo like the pg_query_go precedent.
  `antlr_grammar` drew 0.01/0.00/0.01; `engine_api_prepare` 0.00
  except 0.01 in round 3.

There was no selected-option disagreement to investigate. The extreme
probabilities partly reflect the packets' emphasis: the span-precision
soundness analysis structurally disqualifies the engine/live-prepare
options (no spans, no LIMIT siting, message classification), and the
offset-API verification gap plus module weight separate the two MySQL
parsers. Unanimity is therefore advice, not proof: the selections stand
only after each winner passes the corpus in a spike plus the
differential qualification (SQLite engine differential now, offline;
MySQL live-service differential in B1-03).

## Standing constraints carried into implementation

- The winners must expose the qualified dialect backend interface:
  statements, token/byte spans, kind, LIMIT shape, RETURNING, and
  structural failures, matching `compiler/internal/sql/parser.go`.
- Parser and grammar pins are recorded on checked descriptors exactly
  like the PostgreSQL version stamp.
- Cross-feeding (D03), regex/splitter substitution, and message-text
  classification stay forbidden however the spikes turn out.
- If either spike fails its corpus or differential, the gate reopens
  to the runner-up rather than weakening the contract.
