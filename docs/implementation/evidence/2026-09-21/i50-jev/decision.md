# I50 documentation decisions

Three fresh consultations (jev-1.13.0, all prose rewritten per round,
same questions, options, order, and measured facts) advised the
REQUIREMENTS.md fate, the release-notes shape, and the docs index
scope. Requests, responses, and the equivalence audit live in this
directory (87/87 explanatory strings pairwise distinct).

Unanimous at full confidence for relabel_index (1.0, 1.0, 0.99):
keep the docs index structure, repair false claims, and route
current readers to the ledger, coverage, and evidence. Followed.

Unanimous for banner_historical (0.67, 0.76, 0.85): keep
REQUIREMENTS.md at the root under a historical banner with its
frozen body untouched. Followed.

Unanimous in direction for distribution_section (0.65, 0.46, 0.88)
with round 2 nearly tied against implementation_notes (0.44) at
confidence 0.19. The tie was investigated: the task areas name
release notes without fixing a path, but shipping truth already
lives in distribution/README.md (install, upgrade, prepared
signing, open gates), and a second notes file would split that
truth and drift. Followed: release notes extend the distribution
guide.

These judgments are design advice, not verification. Reconcile,
link, clean-machine, and recount checks remain required before
I50 can be marked complete.
