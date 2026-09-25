# UP10 Jev consultations — findings

25 September 2026 · three fresh SystemOne consultations on the UP10 audit
boundary. Requests, responses and metadata are saved in this directory
(`request-{1,2,3}.json`, `response-{1,2,3}.json`, `response-{N}.metadata.json`,
`send.py`). All prose was rewritten between requests; a wording audit
confirmed zero shared instruction/criteria/state strings across the three
while preserving the same facts, constraints and alternatives.

## Questions and advice

| Question | Wording 1 | Wording 2 | Wording 3 |
| --- | --- | --- | --- |
| Strictness vs baseline | strict_fail_closed 0.96 | fail_every_leak 0.99 | reject_all_reachable 0.98 |
| Type-edge reachability | values_only 0.99 | skip_type_edges 0.61 | verify_but_skip 0.67 |
| Post-bundle edge rule | accounted_chunks 0.86 | listed_chunks_only 0.93 | manifest_bound_chunks 0.96 |

All three wordings agree on every question: reject all reachable host
operations even where current emission trips the rule (repair belongs to
UP11/UP13); verify type-only edges but traverse only value edges; allow
post-bundle static edges only to manifest-listed bundle files.

## Disagreement analysis

The type-edge question drew the weakest agreement (0.61/0.67 on wordings
2–3 for skipping type edges, 0.99 on wording 1). The dissenting pull is
that an uninspected reachable file might conceal executable code. That
concern is answered by mechanism, not vote: `import type` / `export type`
edges are erased by a deterministic transpiler pass before bundling, so a
module reached only through them is never loaded and its body cannot
execute — unlike tree-shaking, which the audit must not trust. The audit
still requires every type-only edge to be declared and to resolve, so no
edge can hide behind a type-only spelling, and the post-bundle audit
re-verifies the final bytes independently. The advice is adopted with that
documented rationale.

## Treatment

Agreement is advice, not proof. The strict posture is additionally
required by the UP10 task contract ("the auditor cannot rely on a
bundler silently removing forbidden reachable code") and by B01, which
UP11/UP13/UP15 depend on as their acceptance boundary. Implementation
proceeds on that basis; residual risk (current browser generations fail
until UP11/UP13 repair emission) is reported in the UP10 handoff.
