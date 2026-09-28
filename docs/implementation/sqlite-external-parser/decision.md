# Decision: use Meyer v0.1.2

Codex authorizes Lane B onward now. Continue the complete checklist in the
existing Muse session, recording evidence, through implementation and handoff.

Use `github.com/sqlc-dev/meyer v0.1.2`, commit
`59b2dbf9b1872316bc6c6d3379b500df84db3e01`, through normal Go modules.
The executable qualification supports its required AST/spans, and the native
oracle supports its name/number semantics. The official module pair is viable,
but Meyer removes the SQLite C bridge and numbering implementation while matching
more of the qualified engine behavior. Its young maintenance history remains a
limitation, not a reason to preserve observed old bugs.

Read [Codex's independent audit](evidence/codex-audit.md) and
[the consultation reconciliation](evidence/selection-assessment.md) before
implementation. Three fresh fully reworded Jev requests produced Meyer/official/
Meyer; this was not unanimous. Selection rests on the source and probe evidence,
not voting or claimed performance gains.

## Implementation decisions resolving A06

1. Use `parser.Options{UpdateDeleteLimit: true}.ParseString` (the actual upstream
   API spelling as declared). Bun's SQLite3.54.0 accepts those mutation clauses.
   Preserve statement counts via ParseString, including empty/multiple statements.
2. Keep native half-open statement spans, including terminating semicolons.
   Do not strip punctuation to imitate the previous parser. Update expectations
   deliberately where needed. Map admitted kinds at the compiler boundary without
   exporting upstream AST types or retaining the obsolete syntax-tree bridge.
3. Reuse `ast.BindParam.Number` and exact Raw spelling. Distinct sigils are distinct
   SQLite parameter identities. Existing CheckSiteNames still validates the suffix
   against the declared Can field; `$1` is a named spelling, not a PostgreSQL slot.
   Numeric/Tcl names can parse and then fail this field-name policy. Keep free
   row-limit names. Reject `#` sites explicitly as outside Can's declared family.
4. Existing CheckSites already rejects `?0`, invalid/overflow numbers and numbers
   outside declared coverage. Preserve this validation; do not duplicate numeric
   assignment or blindly preserve the old `$1` syntax-error golden.
5. Collect all bind nodes with an iterative Node.Children walk, handle typed nils
   correctly, then sort by byte position and validate bounds/non-overlap/Raw slice
   identity before constructing sites. Comma LIMIT children are not in source order.
   Never call recursive ast.Walk for unbounded authored input. Inspect only direct
   query cores for outer LIMIT, requiring an exact BindParam Count without Offset
   or comma. Keep admitted INSERT RETURNING and rejection of UPDATE/DELETE RETURNING
   descriptors. Capture binds inside RETURNING/window/CTE/upsert expressions.
6. Add a 1 MiB statement-byte ceiling and reject embedded NUL before parsing.
   Keep an iterative traversal budget of one million nodes (the previous useful
   cap). Use small injected-budget tests rather than constructing a million nodes.
   A short flat-expression regression should exercise nonrecursive traversal.
   The upstream parser has its own 10,000 recursion ceiling. Do not write a
   replacement lexer or imitate engine-specific recursion limits. Byte/node
   budgets do not promise a hard memory/time bound. Do not swallow unexpected
   panics as user syntax errors. The qualification claim that parser depth alone
   makes recursive ast.Walk safe is incorrect; correct that prose.
7. Reject misplaced LIMIT/ORDER BY on nonfinal direct compound SELECT cores using
   their structure; this is the observed simple admission gap. Add a case with
   both an early LIMIT and a final bound LIMIT so the existing missing-limit rule
   cannot mask it. Do not broaden this into database preparation/name resolution
   or an entire SQL redesign. The parser does not prove every query will prepare.
8. Set `SQLiteVersion = 102`, explicitly identifying Meyer release v0.1.2 in this
   dialect's descriptor stamp (not a SQLite engine or Tree-sitter ABI number).
   Update emitted/runtime version expectations and reject the obsolete stamp15.
   Record full module/commit/Go sums in provenance and a shipped parser notice/lock;
   compiler and bundle content hashes remain the complete compiled-content identity.
   Version102 is not itself a cryptographic proof. Future parser/adapter admission
   changes must deliberately update the stamp and pin together. Avoid a parallel
   compatibility path or a broad cross-dialect metadata redesign.
9. Remove all local vendored SQLite grammar/runtime and obsolete C/Go bridge.
   Keep only useful authored adapter/tests and accurate provenance/notices. Drop
   probe-only Tree-sitter modules from production go.mod/go.sum. Do not touch
   unrelated pinned vendored tools such as Acorn or generated runtime/catalogue.ts.

## Verification and cleanup refinements

- Strengthen the tests at the actual Analysis/descriptor boundary with independent
  exact span/site expectations, sigil-distinct names, numeric limit names, NUL,
  size/node limits, UTF-8 error cursors, bind-bearing AST variants, top-level LIMIT
  and emitted-template identity. Use focused SQL/checker/emitter tests and the
  relevant SQLite/RETURNING/descriptor runtime tests. No staged bundle build.
- Run required runtime lint:fix, format and check commands after metadata/test
  edits. Run formatting only through supported generators where generated files
  are concerned. Review any formatter changes for unrelated edits.
- Verify normal module pins/sums and offline resolution with existing CGo settings;
  no private caches, source copies or new published repositories.
- Register actual trap/finally cleanup for any new temp workspace. Reclaim current
  owned temporary probes at handoff, including compiler/tmpqualprobe and the
  throwaway module, and preserve only compact useful source/evidence. Correct the
  temporary registry's guessed future timestamp and avoid claiming an unexplained
  tidy downgrade without a reproducible case; missing package files during tidy
  can remove unused requirements. Record verified final pins instead.
- Update A04/A06 comparisons to label native semicolon spans intentional (no strip)
  and recursive Walk unsafe for flat expression chains despite parser depth guard.
  Keep historical raw outputs as evidence rather than silently rewriting them.

Codex will independently review the completed diff and run bounded checks. Leave
the worktree and branch available for review; do not commit, push or archive it.

## Review refinement

R01 in [the independent review](evidence/codex-review.md) applies decision 7 to
every reachable SELECT's own direct cores, including CTEs, scalar subqueries and
mutation sources. Only the descriptor's outer LIMIT extraction stays root-only.
This is the same structural clause check during the existing walk; accepting
the nested malformed cases is not an intended tolerance. Execute the saved
[repair checklist](repair-checklist.md) before final acceptance.
