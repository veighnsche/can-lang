# Final candidate consultation assessment

Three fresh fully reworded requests and exact responses are saved here as
`selection-request-{1,2,3}.json` and `selection-response-{1,2,3}.json`.
The reproducible request builder is [consult-selection.py](consult-selection.py).
Each state paragraph, judgment and option was rewritten with the same technical
facts and alternatives; [the wording audit](selection-wording-audit.json) records
the checks. Requests used Jev as a classifier with supplied evidence, not research.

| Round | Choice | Meyer probability | Official probability | More evidence |
| --- | --- | --- | --- | --- |
| 1 | Meyer | 0.95 | 0.05 | 0.00 |
| 2 | Official | 0.35 | 0.61 | 0.04 |
| 3 | Meyer | 0.65 | 0.33 | 0.02 |

The disagreement is material. Do not summarize this as unanimous advice or treat
the majority as proof. Jev supplies no explanation for its competing priorities;
the distribution alone cannot reveal its reasoning. Codex investigated the two
real tradeoffs against source and executable evidence:

- Official bindings are a reproducible, qualified baseline with the existing
  grammar, reducing grammar-change risk. They keep CGo lifecycle obligations,
  the old grammar's proven name-spelling gaps and a proxy-only runtime release
  provenance wrinkle. A scoped adapter must still fix numbering and failure-node
  handling. They fully satisfy the requested source-ownership boundary.
- Meyer v0.1.2 has a short history and sqlc adoption on main, not the inspected
  release. Its supplied AST and lexer match the required contract with a compact
  adapter. The native oracle supports its sigil/name behavior, and a documented
  option matches Bun's mutation LIMIT capability. It avoids another authored
  numbering implementation and SQLite C bridge.
- Both need resource care and neither proves native preparation will succeed.
  Existing CheckSites already rejects invalid zero/overflow bind indices. A
  structural misplaced compound-clause check closes the observed admission gap;
  a full SQLite planner or host-specific expression-depth emulator is unnecessary.
- Read-only independent source review found the necessary Meyer Children coverage
  and byte spans, as well as the concrete NUL and recursive-visitor risks. The
  selected adapter uses a pre-parse input ceiling/NUL check, upstream ParseString,
  bounded iterative traversal, source-order sorting and validated spans. The
  node cap acts after parsing; neither it nor an input limit is a hard heap/time
  isolation boundary. Unexpected panics must not be disguised as syntax errors.

Codex selects Meyer based on these facts, not the classifier vote. Integration
acceptance still requires reviewed code, targeted regressions, native runtime
checks, pin/provenance verification and cleanup. No performance improvement is
claimed; full upstream conformance, broad bundles and platform matrices are not
part of this qualification.

Independent read-only review subsequently read all three complete requests and
responses and found the material facts/alternatives equivalent. It noted framing
variation (round2 emphasizes the established official baseline) and one ambiguous
phrase in round3's objective that could join SQL checking with execution through
Bun.SQL. The detailed contract in that same request correctly locates analysis
in the compiler; only execution uses Bun.SQL. Preserve the sent requests exactly,
record this limitation, and do not attribute the probability split to an inferred
classifier rationale. The reviewer independently supported Meyer from its AST,
native oracle and explicit adapter guards, subject to implementation verification.
