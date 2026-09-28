# Parser provenance inventory (A06 wait-time work; input to C03)

Date: 2026-09-28. Read-only inventory; no doc edits (C03 owns updates).

## Must change with the implementation (C03 targets)

- `compiler/internal/sql/sqlite/PROVENANCE.md` — vendored-runtime
  provenance; goes away with B04. Replacement provenance lives with
  the new adapter and must state: upstream module(s) + version +
  commit/hash, license(s), update procedure, authored ownership,
  shared-cache boundary (no copied sources). Must CORRECT the two
  known defects (glue_test.go omitted from authored list; "No Go
  module ships this grammar").
- `compiler/internal/sql/parser.go` (header comment),
  `dialect.go` (DialectSQLite comment), `sqlite.go` (backend
  comments) — tree-sitter wording; B02/B05 rewrite.
- `runtime/platform/sql/descriptor.ts:45-47,116` —
  `parserVersions = { ..., sqlite: 15, ... }` + "tree-sitter SQLite
  grammar language version per the G-SQL exit" + runtime mismatch
  check. B05 updates coherently; editing triggers AGENTS.md runtime
  lint/format/check rules.

## Historical evidence — annotate, do not rewrite (per C03)

- `docs/bun-integration/asap/decisions.md:33` (Jev rounds advise
  defin/tree-sitter-sqlite3; spike notes; 20-row differential).
- `docs/bun-integration/asap/evidence/sql-spike-notes.md`,
  `consultations-b1-02/*`, `evidence/2026-09-23/b1-02/dialect-contract.md:16`
  (contract table names tree-sitter backend).
- `docs/syntax-taste/...` mentions (dated reviews/research).
- `docs/bun-integration/asap/b1-02-sqlite-and-sql-boundary.md` —
  runtime SQLite integration lane doc (Bun.SQL, transactions);
  parser-agnostic except historical context.

## Checked, no action

- `distribution/provisioning-register.md:44` ("SQLite remains
  included") — test-DB provisioning, unrelated to parser sources.
- `distribution/build.go:199` (`GOPROXY=off CGO_ENABLED=1`) — offline
  assembly expectation; C04 verifies, no doc change foreseen.
- No LICENSE/NOTICE file enumerates the vendored tree-sitter
  sources (PROVENANCE.md + in-file headers only); new external
  modules need no NOTICE additions (proxy modules, MIT/CC0).
