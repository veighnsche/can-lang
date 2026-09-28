# Independent implementation review

Review complete after Muse's initial checklist and same-session repair run.
Codex and the independent reviewer made no concurrent implementation edits.

## R01 — validate compound clauses in reachable SELECT nodes

The initial Meyer adapter calls misplacedCompoundClause once on the root
statement. It therefore admits invalid early LIMIT/ORDER BY in CTEs, scalar
subqueries and INSERT SELECT sources. This is the same cheap structural
validation as the top-level check, not database planning or a new SQL contract.

Concrete native SQLite checks (Bun1.4.2, in-memory Database, CREATE TABLE t(x),
finally Close; no persistent artifacts) rejected all three with
`LIMIT clause should come after UNION not before`:

```sql
WITH c AS (SELECT 1 LIMIT 1 UNION SELECT 2) SELECT * FROM c LIMIT ?
SELECT (SELECT 1 LIMIT 1 UNION SELECT 2) LIMIT ?
INSERT INTO t SELECT 1 LIMIT 1 UNION SELECT 2
```

The source reviewer additionally identified:

```sql
WITH c AS (SELECT 1 ORDER BY 1 UNION SELECT 2) SELECT * FROM c LIMIT ?1
```

Repair: while doing the existing iterative typed-nil-safe walk, validate each
encountered SelectStmt's own direct nonfinal cores. Keep the outer descriptor
LIMIT extraction root-only so a legal nested limit never bounds its parent.
Add descriptor-level rejection tests for CTE, scalar, mutation-source compounds
and valid nested controls, including ORDER BY. This refines the earlier design's
top-level wording rather than preserving the observed malformed-nested gap.

Status: fixed and independently verified. The existing iterative walk invokes
`misplacedCoreClause` for every reachable `*ast.SelectStmt`; the root-only
validation block was removed. Outer LIMIT extraction remains root-only. Three
new tests cover nested rejection, descriptor rejection, UTF-8 failure cursors,
valid nested controls and an inner bound failing to bound its parent.

The separate source reviewer found no remaining concrete defect in the focused
review. Codex independently reran the full SQL suite and focused checker/emitter
tests offline with CGo enabled; all passed. Final reviewed hashes:

- `sqlite.go`: `160cd938187a6149dbc1a13ac5616e3479c05cfefc8b48210eabe31f7e2d2e07`
- `sqlite_contract_test.go`: `4fb0f18b1022de841020687f2163a973bd2c6ee083c1b30c6d308df8ef8f3c56`

See [completion verification](codex-verification.md) for checks and cleanup.

## Independent dependency checks

After B01–B05, Codex ran (both PASS, 2026-09-28):

```sh
GOPROXY=off CGO_ENABLED=1 go list -mod=readonly -m -json github.com/sqlc-dev/meyer
GOPROXY=off CGO_ENABLED=1 go list -mod=readonly -deps -f '{{if .CgoFiles}}{{.ImportPath}}{{end}}' ./compiler/internal/sql
```

The module resolves to the standard shared cache at
`/Users/vince/go/pkg/mod/github.com/sqlc-dev/meyer@v0.1.2`, with the expected
module/GoMod sums and Go1.24 minimum. The only CGo packages in the SQL package's
dependency graph are runtime/cgo and pg_query_go's parser. SQLite adds no CGo.
Production go.mod/go.sum have no Tree-sitter or go-pointer references.
This is offline dependency resolution, not a full distribution build.
