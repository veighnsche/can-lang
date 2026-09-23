# B1-11 document formats contract confirmation

Parsers: Bun 1.4.2 `Bun.TOML.parse`, `Bun.YAML.parse`, `Bun.JSON5.parse`.
Every row below is an executed observation on the pinned binary; see
[formats-native-results.json](../../../../bun-integration/asap/evidence/formats-native-results.json).
Five operations (`codec::decode_toml`, `codec::decode_yaml`,
`codec::decode_json5`, `codec::decode_jsonl`, `codec::consume_jsonl`)
project through one schema-bound factory shared with `decode_json`.

## Unsafe integers

| Aspect | Contract |
|---|---|
| TOML | `9007199254740993` fails natively: losslessness SyntaxError |
| YAML/JSON5 | `9007199254740993` rounds to `9007199254740992`, which projection rejects (`integer_token`): no representable double sits between MAX_SAFE_INTEGER and 2^53, so every rounded value is itself unsafe and integers stay exact |
| JSONL | strict line framing with the exact per-record JSON path preserves `9007199254740993` verbatim |
| MAX_SAFE_INTEGER | `9007199254740991` projects on every format |

## Duplicate keys

| Aspect | Contract |
|---|---|
| TOML | redefinition fails natively (SyntaxError) |
| YAML/JSON5 | last wins; no native metadata exists to reject, so no rejection mode is offered (documented, pinned) |
| JSON/JSONL | per-record duplicate scan rejects with `duplicate_member` |

## Scalars and structure

| Aspect | Contract |
|---|---|
| Nonfinite | genuine `NaN`/`Infinity` numbers on all three parsers; `float` rejects `nonfinite`, `int` rejects `integer_token` |
| TOML dates | `Temporal` objects (`PlainDate`, `PlainDateTime`, `PlainTime`, `Instant`) reject as non-plain (`type`); no temporal schema leaf exists |
| YAML timestamps | plain strings; `yes`/`on` stay strings (1.2 core), every null spelling is null |
| YAML aliases | shared nodes project per occurrence; self-referential anchors yield genuine cycles, rejected (`cycle`) |
| YAML multi-document | native array of documents; projects against array schemas |
| TOML root | table required natively; mixed arrays accepted natively, schema-typed at projection |
| Null | no schema null exists; null fails against every leaf exactly like JSON |

## JSONL framing (Bun.JSONL unused by design)

| Aspect | Contract |
|---|---|
| Blank lines | skipped; CRLF stripped; final line needs no newline |
| Multi-line records | rejected loudly (`invalid_json` at the record index) |
| Trailing garbage / truncation | fail (`invalid_json`); `Bun.JSONL.parse` prefix-success is never validation |
| Locations | record-index paths (`/<index>/<member>`); byte offsets are unrepresentable in `invalid_data` and stay a recorded gap |
| Incremental consume | byte-level framing retains only the incomplete suffix; per-record bytes cap the suffix; the shared node budget bounds unbounded pumps; each record awaits a total handler; cancellation stops reads; returns the record count |
