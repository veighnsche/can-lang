# Vendored tree-sitter SQLite grammar: provenance

The `sqlite` package parses SQLite descriptor statements with a vendored
tree-sitter runtime plus the SQLite grammar. Everything under this
directory except `glue.c`, `glue.go`, and this file is verbatim upstream
content, pinned by hash. Only `glue.c`/`glue.go` are Can-authored.

## Runtime: libtree-sitter v0.27.0 (MIT)

Source: `github.com/tree-sitter/tree-sitter` tag `v0.27.0`
(2026-08-30), directory `lib/src` plus `lib/include/tree_sitter/api.h.

Vendored as individual `.c` files (the `lib.c` amalgam is excluded so
cgo compiles each translation unit exactly once) with all headers and
the `unicode/`, `portable/`, and `wasm-stdlib/` subdirectories intact.
License: MIT, see `https://github.com/tree-sitter/tree-sitter/blob/master/LICENSE`.

## Grammar: defin/tree-sitter-sqlite3 (public-domain blessing)

Source: `github.com/defin/tree-sitter-sqlite3`, `main` at
`7f69bb66845beaac48d467f4f7d107ea2002865e` (2026-05-03). The grammar
mirrors SQLite `parse.y`/`tokenize.c` production for production and
tracks SQLite 3.47.0 at `LANGUAGE_VERSION 15`.

| File here | Upstream `src/` file | SHA256 |
|---|---|---|
| `sqlite_parser.c` | `parser.c` (5.8MB generated) | `99a9cf64e174704f6d250a43155e6425af040840aed3eb12441d0474fe28557b` |
| `sqlite_scanner.c` | `scanner.c` | `3b9cb55a3b20ac40ca065b5a16d0d3f2c7e7558f7d4d54f3a1094a7bbce30c6b` |
| `tree_sitter/parser.h` | `tree_sitter/parser.h` | `180b893c8734778fd32f372dfbc27bd6ad1cd2221f26150b31256ff6716320d2` |

The grammar files are renamed with a `sqlite_` prefix only to avoid the
`parser.c` collision with the runtime translation unit. License: the
project follows SQLite's own posture (author copyright disclaimer plus
blessing; `CC0-1.0` in its `tree-sitter.json` metadata).

## Verification and updates

Re-verify with:

```sh
sha256sum compiler/internal/sql/sqlite/sqlite_parser.c \
  compiler/internal/sql/sqlite/sqlite_scanner.c \
  compiler/internal/sql/sqlite/tree_sitter/parser.h
```

then compare against the table above and re-run the G-SQL corpus
(`tests/integration/sql_descriptors_test.go`) plus the SQLite engine
differential procedure in `docs/bun-integration/asap/evidence/`.
Grammar updates re-pin the commit, refresh the three files and hashes,
and must pass the corpus and differential before landing.

## Why vendored

No Go module ships this grammar, and cgo compiles only its own package
directory, so the C must live here. This mirrors the `pg_query_go`
precedent (generated C plus cgo, pinned upstream) without adding a JVM
or Node regeneration step to the build: regeneration needs only the
tree-sitter CLI and is a maintainer action, never part of `go build`.
