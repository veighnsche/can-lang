# Independent contract audit (Codex)

Read-only audit before integration, 2026-09-28, base
`79830698227dd2706271ff1427c107ef2325f32a`.

- `compiler/internal/sql/parser.go` is the upstream-independent Analysis boundary.
  Exact occurrence spans must be tested independently: current corpus tiling checks
  compare against the same parser's sites and cannot expose a consistently missing
  site. Existing UTF-8 span tests exercise PostgreSQL, not SQLite.
- `cardinality.go` admits bounded SELECT, non-returning DML execute, and the
  deliberate INSERT RETURNING/one form. Preserve these distinctions.
- Runtime descriptor validation pins SQLite `version:15`; compiler corpus/F03 and
  runtime SQLite/RETURNING fixtures also contain this value. It currently denotes
  grammar ABI, not unique parser content. Any stamp change must be coherent.
- Tree-sitter missing nodes retain expected token names; use `IsMissing` instead
  of searching only for a node type literally named MISSING. Tree-owned nodes
  cannot survive tree Close. Check SetLanguage, close parser and tree on all paths.
- The existing adapter and historical dialect-contract table strip named-parameter
  sigils and call that SQLite engine behavior. This is a claim to investigate,
  not independent support for the intended semantics. Current official
  [SQLite parameter documentation](https://www.sqlite.org/lang_expr.html#parameters)
  names parameters including their sigils. Separate source name validation from
  numeric binding identity and verify exact engine behavior with bounded probes.
- `distribution/build.go` supplies CGo and disables the proxy during assembly;
  preserve this. PostgreSQL remains a CGo dependency even if Meyer wins.
- `distribution/notices` is shipped automatically; SQL binding lock currently
  covers PostgreSQL alone. Add accurate SQLite upstream/module licensing and pins.
- Vendored provenance incorrectly claims no Go module exists and identifies the
  authored glue test as upstream. Remove/correct those claims.

Additional acceptance cases: independently asserted multibyte source positions;
all real occurrences amid quoted/comment decoys; empty/comments/multiple statements;
subquery LIMIT without outer LIMIT; expression/comma/OFFSET LIMIT rejection;
RETURNING text decoys and admitted INSERT versus UPDATE/DELETE distinctions;
syntax error/missing node cursors after multibyte text.

This audit ran no tests, installed no dependencies and allocated no temporary
workspace. Candidate source/resource review and final implementation review follow.

## Candidate source observations

After Muse populated the standard cache, read-only review verified the exact
defin pin `v0.1.1-0.20260503135658-7f69bb66845b` parser/scanner hashes match
Can's current files. Official go-tree-sitter v0.25.0 admits ABI13–15.
The grammar's `malformed_blob_literal` and `malformed_number_id` are ordinary
expression nodes whose comments require consumers to reject them; HasError alone
misses these. The grammar also recognizes `#name` bind nodes, outside Can's stated
parameter family, so do not accidentally admit them. These deserve bounded engine
checks and focused regression cases if official bindings are selected.

## Bounded native SQLite observations

Executed `bun -e` using `Database(":memory:")` from `bun:sqlite` on Bun1.4.2,
with `try/finally db.close()` and no files, on 2026-09-28:

| SQL | Observed result |
| --- | --- |
| `SELECT :a AS a, @a AS b, $a AS c` with `{":a":11,"@a":22,"$a":33}` | `{a:11,b:22,c:33}`: sigils are distinct identities |
| `SELECT $1 AS x` with `{"$1":7}` | `{x:7}`: digit-leading dollar name is accepted |
| `SELECT X'01001'` | rejected: unrecognized token |
| `SELECT 123abc` | rejected: unrecognized token |
| `SELECT #name` | preparation accepted; this is outside Can's documented family |

These checks validate the earlier distinctions; they are correctness probes,
not benchmarks or full-engine conformance. `$1` is a named SQLite parameter,
not PostgreSQL-style numeric slot 1. Named field matching remains a separate
Can restriction. A grammar rejection of `$1` must not be described as engine
equivalence; whether to expose its spelling is a policy decision.

## Meyer source/resource review

Pinned v0.1.2 source has a fixed parser recursion ceiling of 10,000 covering
expression, SELECT and FROM-item recursion. Options exposes UpdateDeleteLimit,
not configurable depth, tokens or cancellation. Parse(ctx, Reader) ignores ctx
and reads without a byte ceiling; use ParseString after a byte limit instead.
ParseString recovers only private parser bail, re-panicking unexpected faults.
`ast.Walk` is recursive without its own bound, and flat left-associative operators
form deep BinaryExpr chains without increasing parser recursion depth. Prefer
bounded iterative Node.Children traversal. Reject embedded NUL before parsing:
the lexer treats NUL as EOF and can otherwise ignore following SQL bytes,
including a second statement. These are source observations, not stress tests.

Meyer lexer positions are half-open byte indices; failures have byte Offset,
which needs conversion using utf8.RuneCountInString for Can's character cursor.
Source review found no omitted bind-bearing Children fields for admitted DML,
CTEs, window/frame expressions, upserts or RETURNING. AST traversal is not always
source order: Limit.Children visits Count before Offset even for `LIMIT x,y`.
Collect then sort bind nodes by byte position and validate bounds/non-overlap.
BindParam.Number is assigned in lexical source order by the upstream parser,
using complete Raw names including sigils. Reuse those numbers instead of
reimplementing them. Inspect only the outer SelectStmt's final query core for
LIMIT; require exact BindParam Count and no Offset or comma. RETURNING is direct
on mutation nodes. These observations do not replace executable regression tests.

Primary pinned source references:
[parser](https://github.com/sqlc-dev/meyer/blob/v0.1.2/parser/parser.go),
[lexer](https://github.com/sqlc-dev/meyer/blob/v0.1.2/lexer/lexer.go),
[AST traversal](https://github.com/sqlc-dev/meyer/blob/v0.1.2/ast/ast.go),
[LIMIT](https://github.com/sqlc-dev/meyer/blob/v0.1.2/ast/select.go),
[bind numbering](https://github.com/sqlc-dev/meyer/blob/v0.1.2/parser/parse_expr.go).

## Baseline verification

Before production edits, Codex ran
`go test -p 1 ./compiler/internal/sql -run '^TestCorpus$' -count=1 -timeout=60s`.
Result: PASS (2026-09-28). This confirms the existing corpus baseline only;
historical expectations remain subject to evidence about intended semantics.

Codex preserved [all 53 additional SQL inputs](qualification-extra-cases.json)
from Muse's completed qualification before temporary cleanup. Together with the
21 SQLite corpus cases, these identify the 74 rows of recorded candidate output.

Final reference scan must include runtime/test/http-hedge.test.ts: it also has
four hand-built SQLite descriptor version15 fixtures. Update these with the
metadata change and run its bounded tests alongside the three SQL runtime files.
