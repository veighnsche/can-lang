# SQLite backend provenance: external Meyer module

The `sqlite` backend parses SQLite descriptor statements with the
external `github.com/sqlc-dev/meyer` Go module plus the authored
adapter in `sqlite.go`. No grammar or runtime sources are copied into
this repository: the module resolves through the normal Go proxy and
checksum database into the shared module cache.

## Upstream pin

- Module: `github.com/sqlc-dev/meyer`
- Version: `v0.1.2` (tag `v0.1.2`)
- Commit: `59b2dbf9b1872316bc6c6d3379b500df84db3e01`
- Published: `2026-08-25T20:44:20Z`
- go.sum: `h1:40Ng9Glnx7CTf3yOYV7jfvDIN7voIFjrStkIULS+uws=`
- go.mod sum: `h1:pS4USCRf/SLjWtaMcnTo4YrEEFKBj8CyyqlxcVUJQH8=`
- Transitive Go dependencies: none (the module declares zero requires).
- License: MIT, `The sqlc Authors`; shipped notice at
  `distribution/notices/meyer-LICENSE.txt`, pinned by
  `distribution/notices/sqlite-binding.lock.json`.
- Parser options: `parser.Options{UpdateDeleteLimit: true}`, matching
  Bun's SQLite build, which accepts ORDER BY and LIMIT on UPDATE and
  DELETE.

## Authored ownership

Can-authored: `sqlite.go` (adapter: input guards, statement spans and
kinds, iterative bind collection with sorting and span validation,
top-level LIMIT shape, RETURNING presence, failure mapping),
`sqlite_params_test.go`, `sqlite_contract_test.go`, the shared
`CheckSites`/`CheckSiteNames`/`CheckCoverage` policy, and this file.
Upstream-owned: the grammar, lexer, AST and numbering inside the
Meyer module. No upstream AST or parser types cross the
compiler-owned `Analysis` boundary.

## Descriptor stamp

Checked SQLite descriptors record `SQLiteVersion = 102`, naming the
Meyer v0.1.2 release. It is not a SQLite engine version, a
Tree-sitter ABI number, or a cryptographic proof: compiler and bundle
content hashes remain the complete compiled-content identity. Parser
or adapter admission changes must update the stamp and the module pin
together, plus the runtime `parserVersions` table and the lock file.

## Verification and updates

Re-verify with:

```sh
go list -m github.com/sqlc-dev/meyer
go test -p 1 ./compiler/internal/sql/ -count=1
go test -p 1 ./compiler/internal/emit/ -run TestSQL -count=1
```

then the runtime SQLite/RETURNING/descriptor suites. Grammar updates
re-pin the module, refresh the sums, stamp and lock file, and must
pass the corpus, contract, emitter and runtime suites before landing.

## History

On 2026-09-28 the backend moved from a vendored tree-sitter SQLite
grammar and runtime (88 tracked files, 6,729,649 bytes, plus a custom
C serialization bridge) to this external module, removing the SQLite
cgo dependency; PostgreSQL cgo remains. The old provenance claim that
no Go module ships a SQLite grammar was incorrect and is superseded.
Qualification evidence, the engine oracle and the candidate decision
live under `docs/implementation/sqlite-external-parser/`.
