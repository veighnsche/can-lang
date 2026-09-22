# I45 release-gate decisions

Three fresh consultations (jev-1.13.0, all prose rewritten per round,
same questions, options, order, and measured facts) advised the gate
topology, the fresh-emit mechanism, and the TS evidence level.
Requests, responses, and the equivalence audit live in this directory
(87/87 explanatory strings pairwise distinct).

Unanimous at full confidence for env_copy (1.0, 1.0, 1.0): the
existing staged stdlib test copies each project's dist under
CAN_FRESH_EMIT_DIR and the workflow runs tsc over the copies.
Followed; no parallel emitter is built.

Unanimous for upload_emit (0.84, 0.79, 0.77): the fresh emit tree
plus the tsc log upload as workflow artifacts. Followed.

Unanimous in direction for extend_verifier (0.70, 0.82, 0.58) with
round 3 hedging toward separate_gates (0.31). The hedge was
investigated: per-family jobs would isolate failures, but the plan
creates separate operations only when warranted, one macos-15 job
shares a single service setup with a single verdict, and
release-qualify already stands as the separately operated gate that
is warranted (offline install/update). Followed: one extended
verifier job plus the recreated tsc workflow.

These judgments are design advice, not verification. Gate, pin, and
proof checks remain required before I45 can be marked complete.
