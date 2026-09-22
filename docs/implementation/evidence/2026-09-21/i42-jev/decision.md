# I42 admission-application decisions

Three fresh consultations (jev-1.13.0, all prose rewritten per round,
same questions, options, order, and measured facts) advised the pool
assertion design, the per-application source layout, and the outcome
row allocation. Requests, responses, and the equivalence audit live in
this directory (87/87 explanatory strings pairwise distinct).

Unanimous direction for elide_pool (0.94, 0.48, 0.44) with a round-2
hedge toward review_needed (0.41): assertion rows omit sql::pool
inputs while the harness splices its frozen typeless token, mirroring
transaction handles. The hedge was investigated: it reflects caution
about widening checker acceptance, but no shipped fixture declares a
pool input, so elision accepts strictly new programs, and unprovisioned
pool use still fails closed at the assertion live boundary. Followed
on the merits; per-request open/close would contradict the P13
near-capture contract. The dominated middle path (elide pools only at
near positions) was considered and dropped: it needs resolve-level
near-ness tracking for no behavioral gain.

Majority for multi_module (0.59, 0.85, 0.80): each application splits
into src/ modules behind provides and uses so admission runs exercise
multi-file resolution, overlays, and navigation like real projects.
Followed.

Unanimous at full confidence for split_by_observability (1.0, 0.97,
1.0): wire-visible rows prove over staged raw HTTP, swap and no-swap
rows in the pinned browser, one native witness per outcome row.
Followed.

These judgments are design advice, not verification. Checker,
application, protocol, and browser checks remain required before I42
can be marked complete.
