# I44 predecessor-retirement decisions

Three fresh consultations (jev-1.13.0, all prose rewritten per round,
same questions, options, order, and measured facts) advised the
retired-command experience, the residual proof, and the tscheck fate.
Requests, responses, and the equivalence audit live in this directory
(87/87 explanatory strings pairwise distinct).

Unanimous and strong for both_gates (0.96, 0.99, 0.97): prove
retirement with a static machine check that the launcher references
no legacy symbols plus behavior probes (rejection fixtures, dispatch
searches). Followed.

Unanimous in direction with strengthening confidence for
explicit_retired (0.53, 0.65, 0.98): keep one named stub per removed
command exiting 2 with a retired message. The round-1 near-tie
against silent removal (0.45) was investigated: with zero external
users either fails safely, but the I41 precedent already fails the
withdrawn `lsp --baseline` flag with an explicit retired error, and
named stubs distinguish "removed" from "typo" for stale scripts and
notes. Followed.

Majority for delete_now (0.90, 0.70) with a low-confidence
review_needed round 2 (0.50, confidence 0.26). The hesitation was
investigated: it reflects the I45 plan's "adapt/update" wording for
tscheck files, but the gate cannot pass unmodified once its golden
subjects are deleted (tsc fails on zero inputs), while pausing would
keep a dead configuration and a paused job. I45 item 2 rebuilds
TypeScript gating around freshly emitted CI output, which recreates
rather than adapts the config. Followed: tscheck/ and tsc.yml are
deleted now and I45 owns the recreation.

These judgments are design advice, not verification. Deletion,
gate, and proof checks remain required before I44 can be marked
complete.
