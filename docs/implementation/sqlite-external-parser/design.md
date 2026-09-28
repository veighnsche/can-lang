# External SQLite parser implementation

The requested endpoint is a Can source tree containing authored compiler code,
with SQLite parser sources supplied by normal pinned external Go modules in the
shared module cache. No copied C or Go grammar/runtime, repository archive,
submodule, generated-source dump, or private dependency cache is acceptable.
This work preserves checked raw SQL and native Bun.SQL execution. It does not
redesign the SQL language, publish a parser repository, benchmark, or build a
distribution bundle. There are no external consumers requiring compatibility.

## Evidence and qualification order

Read [the corrected brainstorm](brainstorm.md) and its
[three saved consultations](evidence/brainstorm-jev.json). The earlier
[extraction proposal](evidence/superseded-extraction-jev.json) is superseded:
the exact current grammar already has a Go module and binding.

1. Establish the current grammar via `github.com/defin/tree-sitter-sqlite3`
   commit `7f69bb66845beaac48d467f4f7d107ea2002865e` and official
   `github.com/tree-sitter/go-tree-sitter` runtime. The grammar ABI is 15;
   upstream's declared v0.24.0 runtime supports only 14 and cannot be used.
   Verify v0.25.0 from the normal Go proxy (commit
   `adc13ffd8b2c0b01b878fda9f7c422ce0df5fad3`) including sums and archive
   source completeness. Its GitHub tag is absent; record that provenance.
2. Qualify `github.com/sqlc-dev/meyer v0.1.2`, an independent MIT Go parser,
   against the same bounded contract cases. Inspect byte spans, visitor
   completeness, statement boundaries, options, and resource behavior.
3. Prefer Meyer if evidence supports the intended contracts with modest authored
   adaptation and acceptable resource behavior. Otherwise use the qualified
   official module pair. A new owned parser repository requires a demonstrated
   limitation of both upstream paths and is outside this implementation scope.

The final candidate decision belongs to Codex after reviewing qualification;
new difficult decisions require three fresh fully reworded Jev requests with
identical facts/alternatives, wording audit, saved responses and reconciliation.
Jev is advice, never proof. Prior consultations establish probe order only.

## Required behavior

The compiler-owned `Analysis` must retain accurate statement count, source byte
spans and kind, parameter occurrences in source order, top-level SELECT LIMIT
shape, RETURNING presence and syntax failures. Descriptor validation continues
to admit one statement, enforce declared parameter coverage and LIMIT placement,
and emit exact text/value segments. Preserve admitted INSERT RETURNING.

Qualify UTF-8 before and inside identifiers/literals, comment/string placeholder
decoys, bare/numbered/named/repeated parameters (including differing sigils and
digit names), statement separators/comments/empty input, CTEs/nested LIMITs,
comma/OFFSET/expression LIMITs, DML RETURNING, and malformed input. SQLite engine
behavior and intended Can contracts resolve old test disagreements; old goldens
are not compatibility obligations. In particular the old code strips sigils
when numbering names and rejects `$1`; investigate rather than enshrine these.

Parser identity must name content/version, not treat Tree-sitter ABI15 as a
unique grammar revision. Keep emitted/runtime metadata coherent. No upstream AST
or parser types escape the compiler analysis boundary. Native SQL execution and
runtime record validation remain in Bun. Dropping SQLite cgo does not remove the
PostgreSQL cgo dependency. Offline distribution assembly keeps `GOPROXY=off`
and `CGO_ENABLED=1`; dependencies are provisioned first.

## Ownership, load and cleanup

Muse owns implementation files, tests, provenance and checklist progress. Codex
owns this design, `decision.md`, independent review and final verification.
Use the existing `/Users/vince/.codex/worktrees/db71/can-lang` checkout on
`codex/sqlite-external-parser`; the main project has concurrent unrelated work.
No concurrent edits to Muse-owned files. Retain compact evidence here; register
cleanup immediately for every temporary workspace and reclaim it on success,
failure or handled interruption. Shared Go caches are allowed; private caches,
retained builds/bundles and copied dependency trees are not. No benchmarks or
broad builds. Use serial/bounded focused correctness checks. If authored runtime
TypeScript changes, follow all runtime lint/format/check rules in AGENTS.md.

Baseline: SQLite subtree has 88 tracked files / 6,729,649 bytes. Its current
provenance incorrectly calls the authored `glue_test.go` upstream and says no
Go module ships this grammar. Correct both in the replacement documentation.
